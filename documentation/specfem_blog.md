---
Running Specfem HPC benchmark on OpenShift
---

Introduction
============

`Specfem3D_Globe` is a scientific HPC code that simulates seismic wave
propagation, at global or regional scale. It relies on a 3D crustal
model and takes into account parameters such as the Earth density,
topography/bathymetry, rotation, oceans, or self-gravitation.  Specfem
is a reference application for supercomputer benchmarking, thanks to
its good scaling capabilities. It supports OpenMP multithreading,
asynchronous MPI communications, and GPU acceleration (Cuda or
OpenCL).

This blog post presents the design and implementation of a GO client
for building and running Specfem on OpenShift. Specfem has the
particularity to require two stages of parallel execution: the mesher
runs first, and generates a source-code header file (required to build
the solver) plus a mesh database. Then the actual solver runs and
performs the simulation.

In the following, we go in depth into Specfem build steps and how they
are carried out in OpenShift. See the figure below for an overview of
the build flow.

![Specfem client control flow](control-flow.png)

Configuring Specfem
===================

Our goal with Specfem GO client is to benchmark the performance of our
in-house OpenShift cluster. We chose to design a GO client (instead of
an operator), so that we could control more interactively the
configuration and execution of the application.

For the configuration, we wanted to have control over four properties:
1. the number of OpenMP cores
2. the number of MPI processes
3. the number of MPI processes per worker node
4. Specfem problem size (`NEX`)

So we designed [a custom Kubernetes datatype API] for storing these
properties, along with a few other settings (source repository/branch,
storage type, ...). This let us configure the application with a YAML
resource description:

```
apiVersion: specfem.kpouget.psap/v1alpha1
kind: SpecfemApp
metadata:
  name: specfem-sample
  namespace: specfem
spec:
  git:
    uri: https://gitlab.com/kpouget_psap/specfem3d_globe.git
    ref: default
  exec:
    nproc: 1
    ncore: 8
    slotsPerWorker: 1
  specfem:
    nex: 32
  resources:
    useUbiImage: true
    storageClassName: "ocs-external-storagecluster-cephfs"
    workerNodeSelector:
      node-role.kubernetes.io/worker:
```

In the GO client, the configuration is read from the
`config/<name>.yaml` when the application is launched (the default
value is `specfem-sample` for loading
[this configuration file](specfem-sample)):

```
go run . [<name>]
```

[a custom Kubernetes datatype API]: https://gitlab.com/kpouget_psap/specfem-api/-/blob/c3dd290b6b1108ed7da87e6631b5c932cadb169c/pkg/apis/specfem/v1alpha1/specfemapp_types.go
[specfem-sample]: https://gitlab.com/kpouget_psap/specfem-client/-/blob/6af449d47f8bff9aeeda8e103c19c1880b0c3056/config/specfem-sample.yaml

Building the Base Image
-----------------------

In the first stage of the build process, we build the base image,
where we install all the necessary packages. This is done with an
OpenShift [`BuildConfig`], where we inject:
1. a Dockerfile based on Red Hat [UBI] (requires a
[container entitlement]) (otherwise based on Ubuntu)
2. Specfem source repository URI and branch name

The [injection of the configuration bits] is done with the help of GO
templates, similarly to what we can find in Red Hat's
[Special Resource Operator]. This design allows a clear separation
between the resource specifications and the GO code driving the
execution.

When the [base-image `BuildConfig`] has been [created][base_bc_created],
we [wait][base_bc_wait] for the successful completion of the build. If
it fails, the execution is aborted; and if the build was already done
previously, the execution continues without any modification or
delay. This follows the idempotency principle of Kubernetes commands.

[`BuildConfig`]: https://gitlab.com/kpouget_psap/specfem-client/-/blob/master/manifests/01_buildconfig_base.yaml
[UBI]: https://www.redhat.com/en/blog/introducing-red-hat-universal-base-image
[container entitlement]: https://www.openshift.com/blog/how-to-use-entitled-image-builds-to-build-drivercontainers-with-ubi-on-openshift
[injection of the configuration bits]: https://gitlab.com/kpouget_psap/specfem-client/-/blob/master/create.go#L85
[Special Resource Operator]: https://github.com/openshift-psap/special-resource-operator/blob/659da39/pkg/controller/specialresource/resources.go#L205
[base-image `BuildConfig`]: https://gitlab.com/kpouget_psap/specfem-client/-/blob/master/manifests/01_buildconfig_base.yaml
[base_bc_created]: https://gitlab.com/kpouget_psap/specfem-client/-/blob/master/resources_manager.go#L37
[base_bc_wait]: https://gitlab.com/kpouget_psap/specfem-client/-/blob/master/run_and_wait.go#L227

Building the Mesher image
-------------------------

Then the [mesher image is built similarly][mesher_build], with Specfem
problem size (`Nex`) and number of processes injected in the template
and passed to the [Dockerfile][mesher_dockerfile] via environment
variables. We construct the mesher image by configuring Specfem and
building its mesher binary on top our base image.

[mesher_build]: https://gitlab.com/kpouget_psap/specfem-client/-/blob/master/manifests/02_buildconfig_mesher.yaml
[mesher_dockerfile]: https://gitlab.com/kpouget_psap/specfem-client/-/blob/master/manifests/Dockerfile.mesher

Running the Mesher with MPI
---------------------------
<a name="running-the-mesher"></a>

Once the mesher image has been constructed, we can launch the parallel
execution of the mesher. We do this with the help of Google's
[Kubeflow MPI Operator]. In the [`MPIJob` resource], we inject:
1. the name of the current stage (`mesher` or `solver`),
2. the number of MPI processes to spawn
3. the number of MPI processes to spawn on each worker node
4. the number of OpenMP threads
5. the script to launch on each MPI process (`/mnt/helper/run.sh`,
mounted from `ConfigMap/run-mesher-sh` and created from [`run_mesher.sh`])

The `MPIJob` execution creates a launcher pod running our base image
(where OpenMPI is installed), and spawns the right number of worker
pods on the worker nodes. Then the launcher pod kicks the OpenMPI
execution that spawns the MPI processes inside the worker pods. And at
last, Specfem mesher is executed on the OpenShift cluster.

The last missing bit required to properly run Specfem mesher is a
shared filesystem (`ReadWriteMany` access mode). Each of the mesher
processes store their mesh database in this volume, and the lead
mesher process writes a header file (`values_from_mesher.h`) required
to build the solver (more in this in the next section). The setup of
this shared filesystem is out of the scope of this article. In our GO
client, the name of a compliant stage class should be set
[in the configuration resource] (see [Red Hat OCS] or [Amazon EFS] for
instance).

Finally, the `MPIJob` is [created][mpi_created] and
[awaited][mpi_awaited]. As a side node, OpenMPI executions seem never
to return a non-null error code, so we parse the launch pod logs to
detect issues and abort the client execution if necessary.

[Kubeflow MPI Operator]: https://github.com/kubeflow/mpi-operator
[`MPIJob` resource]: https://gitlab.com/kpouget_psap/specfem-client/-/blob/master/manifests/99_mpijob_meshersolver.yaml
[`run_mesher.sh`]: https://gitlab.com/kpouget_psap/specfem-client/-/blob/master/manifests/run_mesher.sh
[in the configuration resource]: https://gitlab.com/kpouget_psap/specfem-client/-/blob/master/config/specfem-sample.yaml#L18
[Red Hat OCS]: https://www.openshift.com/blog/introducing-openshift-container-storage-4-2
[Amazon EFS]: https://docs.openshift.com/container-platform/4.5/storage/persistent_storage/persistent-storage-efs.html
[mpi_created]: https://gitlab.com/kpouget_psap/specfem-client/-/blob/master/resources_manager.go#L22
[mpi_awaited]: https://gitlab.com/kpouget_psap/specfem-client/-/blob/master/run_and_wait.go#L439

Building the Solver Image
-------------------------

The second phase of Specfem build consists in building the solver,
which will perform the actual simulation. However, the solver image
cannot be constructed as simply as other images, as it requires an
input file from the mesher phase: `values_from_mesher.h`. In the
previous section, we explained how this file, along with the MPI mesh
database, was saved in a shared volume. But as of OpenShift 4.5, it is
unfortunately not possible to include persistent volumes in the
`BuildConfig`.

To by-pass this limitation, we have to find a solution to retrieve
this file while building the solver binary. We found three possible
ways, and finally only kept the last one:

1. sharing via an HTTP server. First we launch a helper pod with the
shared volume. This pod launches a Python micro-HTTP server, and a
`Service`+`Route` expose the HTTP server at a fixed address. In the GO
client, we monitor the pod logs, and when the HTTP server is ready, we
launch the `BuildConfig`, where we retrieve the header file via with a
`curl` download. When the file has been shared, the pod cleanly
terminates its execution.

2. sharing via a GIT repository. First we launch another helper pod
with the shared volume. This pods receives a `Secret` with the
credentials to access a GIT repository, where it will push the header
file in a dedicated commit. Then the GO client launches a
`BuildConfig` that will fetch the commit from the GIT repository, and
perhaps clean it up afterwards.

3. building the image from a custom `buildah` pod. If `BuildConfig`
`buildah` scripts cannot have volumes, we can still design a custom
`Pod` that will receive the shared volume, run `buildah` and push the
image to the `ImageStream`. 

This last sharing possibility is the most flexible (no coordination as
in with the HTTP-sharing, no external storage as with the GIT
repository), so we kept only this one in the final version of the
code.

However, for `buildah` to work properly in a pod, it must be:
1. configured to use `fuse` overlays (see the
[`buildah`-in-a-container blog post][buildah_blogpost] or our
[Dockerfile][Dockerfile.mesher2solver] for more details);
2. the `DockerPushCfg` secret must be passed to the pod and
transformed before `buildah` can use it to push to our `ImageStream`
(see [OpenShift `buildah` documentation]);
3. the `fuse` module must be loaded in the host kernel. This is done
with a [Node Tuning Operator] `tuned`
[resource configuration][tuned_resource]. In addition, we have to tag
one of the worker nodes with the `buildah.specfem.build` label, to
ensure that the node running the `buildah` pod is actually the one
with `fuse` module.

With all these steps in place, we can trigger the build of Specfem
solver image, and simply wait for the pod execution completion.

[OpenShift `buildah` documentation]: https://docs.openshift.com/container-platform/4.5/builds/custom-builds-buildah.html
[buildah_blogpost]: https://developers.redhat.com/blog/2019/08/14/best-practices-for-running-buildah-in-a-container/
[Dockerfile.mesher2solver]: https://gitlab.com/kpouget_psap/specfem-client/-/blob/master/manifests/Dockerfile.mesher2solver_helper
[Node Tuning Operator]: https://docs.openshift.com/container-platform/4.5/scalability_and_performance/using-node-tuning-operator.html
[tuned_resource]: https://gitlab.com/kpouget_psap/specfem-client/-/blob/master/manifests/05b_tuned_fuse-module.yaml

Running the Solver and Saving Output Logs
-----------------------------------------

Once Specfem solver image has been built, we can create a new `MPIJob`
(see [Running the Mesher with MPI](running-the-mesher) for further
details about the MPI execution) for running Specfem simulation. 

The last action of the GO client after the solver execution is to run
a helper pod that retrieves Specfem solver output logs and saves it in
the local workstation. This simple pod receives the shared volume and
`cat`s the content of the log file. In the GO client, we wait for the
completion of the pod, and save to disk the content of the pod's log,
and we give a unique name to the file (eg,
`specfem.solver-1proc-8cores-32nex_20200827_140133.log`) to simplify
the benchmark of the Specfem execution on OpenShift.

