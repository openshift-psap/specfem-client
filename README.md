Specfem OpenShift GO client
===========================

This repository is *experimental*. It is provided without guarantee of
stability and/or valid results.

This repository contains a GO client for running
[Specfem](https://geodynamics.org/cig/software/specfem3d_globe/)
[Globe](https://github.com/geodynamics/specfem3d_globe) on OpenShift
(4.4).

>  SPECFEM3D_GLOBE simulates global and regional (continental-scale) seismic wave propagation.

> Effects due to lateral variations in compressional-wave speed, shear-wave speed, density, a 3D crustal model, ellipticity, topography and bathymetry, the oceans, rotation, and self-gravitation are all included.

> The version 7.0 release offers GPU graphics card support for both OpenCL and CUDA hardware accelerators, based on an automatic source-to-source transformation library (Videau et al. 2013). It offers additional support for ADIOS file I/O formats and contains important bug fixes related to 3D topography and geographic/geocentric transformations. Seismogram file names adapt a new naming convention, with better compatibility to the seismogram specifications by the Incorporated Research Institutions for Seismology (IRIS).

> The version embeds non-blocking MPI communications and includes several performance improvements in mesher and solver. It provides a perfectly load-balanced mesh for 3D mantle models honoring shallow oceanic Moho (depths less than 15 km) and deep continental Moho (depths greater than 35 km). It also accommodates European crustal models EPcrust (Molinari & Morelli, 2011) and EuCrust07 (Tesauro et al., 2008), which may be combined with global crustal model Crust2.0. Sedimentary wavespeeds are superimposed on the mesh if sediment thickness exceeds 2 km.

Requirements
============

* The Node Tuning Operator must be [installed](https://github.com/openshift/cluster-node-tuning-operator)
* Kubeflow MPI operator must be [installed](https://github.com/kubeflow/mpi-operator#installation)

* To build the `UBI8` base image, the cluster must be correctly
  [entitled](https://www.openshift.com/blog/how-to-use-entitled-image-builds-to-build-drivercontainers-with-ubi-on-openshift)
  (alternatively, the
  [flag](https://github.com/openshift-psap/specfem-client/blob/v1.0/config/specfem-sample.yaml#L17)
  `spec.resources.useUbiImage` can be set to `false` to use the
  `ubuntu:eon` as base image).
* A `ReadWriteMany` file-system must be available (see
  [`spec.resources.useUbiImage`](https://github.com/openshift-psap/specfem-client/blob/v1.0/config/specfem-sample.yaml#L18)
  to configure the storage-class name) for the storage-class
  configuration). [Amazon EFS](https://aws.amazon.com/efs/) provides
  such a file-system.
* Environment variable `KUBECONFIG` must point to a valid `kubeconfig`
  file
* Tested with OpenShift `4.4.8`

Control flow
============

The GO client controls the flow of Specfem build process and validates
that each step is successful (by bailing out or waiting forever). See
this diagram for an overview of the control flow:

![Control flow of Specfem GO client](specfem_flow.png)

Configuration
=============

The configuration file must be stored in the `config` directory, and
passed as the first command-line argument (without `.yaml`
extension). If no argument is provided,
[`config/specfem-sample.yaml`](https://github.com/openshift-psap/specfem-client/blob/master/config/specfem-sample.yaml)
is used:

```
apiVersion: specfem.kpouget.psap/v1alpha1
kind: SpecfemApp
metadata:
  name: specfem-sample
  namespace: specfem
spec:
  git:
    uri: https://github.com/geodynamics/specfem3d_globe.git
    ref: fecb1af5
  exec:
    nproc: 1
    ncore: 8
    slotsPerWorker: 1
  specfem:
    nex: 32
  resources:
    useUbiImage: true
    storageClassName: ""
    workerNodeSelector:
      node-role.kubernetes.io/worker:
```

Usage
=====

```
go run . [config name]
```

Sample output logs:

```
Create ImageStream/specfem
Create BuildConfig/specfem-base-image
BuildConfig 'specfem-base-image' created
Build status of build/specfem-base-image-1 status: "Complete"
Checking imagestreamtag/specfem:base | all
Create BuildConfig/specfem-mesher-image
BuildConfig 'specfem-mesher-image' created
Build status of build/specfem-mesher-image-1 status: "Complete"
Checking imagestreamtag/specfem:mesher | config
Create ConfigMap/run-mesher-sh
Create PersistentVolumeClaim/specfem
Create MPIJob/mpi-mesher
Status of pod/mpi-mesher-launcher-gjgks: Succeeded
Status of pod/mpi-mesher-launcher-gjgks: Succeeded
<mesher output redacted>
MPI mesher done!
Create BuildConfig/specfem-after-mesh-helper
BuildConfig 'specfem-after-mesh-helper' created
Build status of build/specfem-after-mesh-helper-1 status: "Complete"
Create ConfigMap/run-mesher2solver-sh
Create Tuned/specfem-fuse-for-buildah
Checking imagestreamtag/specfem:solver | mesher
Found solver image, don't recreate it.
Checking imagestreamtag/specfem:solver | mesher
Create ConfigMap/run-solver-sh
Create MPIJob/mpi-solver
<solver output redacted>
MPI solver done!
Create Job/save-solver-output
Status of pod/save-solver-output-rsbcq: Succeeded
Saved solver logs into '/tmp/specfem.solver-1proc-8cores-32nex_20200831_163809.log'
All done!
```

As mentioned in the log messages, the solver output logfile
`output_solver.txt` is saved locally into
`/tmp/specfem.solver-4proc-16cores-32nex_20200831_163809.log`.

Cleanup
=======

Currently, only one instance/configuration of the application can
exist in the cluster (the object names / namespace are static), so the
workload objects must be deleted/recreated to rerun the application.

A helper flag helps deleting the relevant resources:

```
go run . [config name] -delete <flag>
```

The delete `flag` can take the following values:

- `solver` deletes the workload resources related to the solver execution
- `mesher` deletes the workload resources related to the mesher and
  the solver execution
- `config` deletes the resources related to Specfem (compile-time) configuration
- `port` (internal) deletes the resources related to the `port`
mesher->solver information sharing
- `buildah` (internal) deletes the resources related to the `buildah`
mesher->solver information sharing
- `all` deletes all the resources created by this client

Note that the flags are ordered: setting one flag deletes all the
resources listed *above*.
