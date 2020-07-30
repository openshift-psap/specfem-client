Specfem OpenShift GO client
===========================

This repository is *experimental*, and provided without guarantee of
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

* To build the `UBI8` base image, the cluster must be correctly
  [entitled](https://www.openshift.com/blog/how-to-use-entitled-image-builds-to-build-drivercontainers-with-ubi-on-openshift)
  (alternatively, the
  [flag](https://gitlab.com/kpouget_psap/specfem-client/-/blob/7a0c6476ab4a1e5ca8f7052c7f54a6a0f536eed4/resources.go#L32)
  `USE_UBI_BASE_IMAGE` can be set to `false` to use the `ubuntu:eon`
  as base image).
* The Kubeflow MPI operator must be [installed](https://github.com/kubeflow/mpi-operator#installation)
* Amazon [EFS](https://aws.amazon.com/efs/) must be setup, to provide
  a `ReadWriteMany` filesystem (see
  [`resources_pvc_efs.go`](https://gitlab.com/kpouget_psap/specfem-client/-/blob/master/resources_pvc_efs.go#L15)
  for the storage-class configuration) (TODO: pass storage class name
  in the main SpecfemApp configuration object).
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

The execution configuration is currently hard-coded in `config.go`:

```
    specfemv1.SpecfemApp{
		ObjectMeta: metav1.ObjectMeta{
			Name: "specfemapp",
		},
		Spec: specfemv1.SpecfemAppSpec{
			Git: specfemv1.GitSpec{
				Uri: "https://gitlab.com/kpouget_psap/specfem3d_globe.git",
				Ref: "master",
			},
			Exec: specfemv1.ExecSpec{
				Nproc: 4,
				Ncore: 16,
			},
			Specfem: specfemv1.SpecfemSpec{
				Nex: 32,
			},
		},
	}
```

Usage
=====

```
go run .
```

Sample output logs:

```
2020/07/30 11:42:59 Create imagestreams/specfem
2020/07/30 11:42:59 Create buildconfigs/specfem-base-image-ubi
2020/07/30 11:42:59 BuildConfig 'specfem-base-image-ubi' created
2020/07/30 11:42:59 Build status of build/specfem-base-image-ubi-1 status: "Complete"
2020/07/30 11:42:59 Checking imagestreamtag/specfem:base | all
2020/07/30 11:43:00 Create buildconfigs/specfem-mesher-image
2020/07/30 11:43:00 BuildConfig 'specfem-mesher-image' created
2020/07/30 11:43:00 Build status of build/specfem-mesher-image-1 status: "Complete"
2020/07/30 11:43:00 Checking imagestreamtag/specfem:mesher | config
2020/07/30 11:43:00 Create configmaps/bash-run-mesher
2020/07/30 11:43:00 Create persistentvolumeclaims/specfem
2020/07/30 11:43:01 Create mpijobs/mpi-mesher
2020/07/30 11:43:01 Status of pod/mpi-mesher-launcher-2nhdl: Succeeded
[...]
Elapsed time for mesh generation and buffer creation in seconds =    29.471269823000000
[...]
2020/07/30 11:43:02 MPI mesher done!
2020/07/30 11:43:02 Create buildconfigs/specfem-after-mesh-helper
2020/07/30 11:43:02 BuildConfig 'specfem-after-mesh-helper' created
2020/07/30 11:43:02 Build status of build/specfem-after-mesh-helper-1 status: "Complete"
2020/07/30 11:43:02 Create configmaps/bash-run-buildah-helper
2020/07/30 11:43:03 Create tuneds/specfem-fuse-for-buildah
2020/07/30 11:43:03 Checking imagestreamtag/specfem:solver | mesher
2020/07/30 11:43:03 Found solver image, don't recreate it.
2020/07/30 11:43:03 Checking imagestreamtag/specfem:solver | mesher
2020/07/30 11:43:03 Create configmaps/bash-run-solver
2020/07/30 11:43:03 Create mpijobs/mpi-solver
[...]
 Total elapsed time in seconds =    465.18405300000001
[...]
2020/07/30 11:43:05 MPI solver done!
2020/07/30 11:43:05 Create jobs/save-solver-output
2020/07/30 11:43:05 Status of pod/save-solver-output-dt9s8: Succeeded
2020/07/30 11:43:06 Status of pod/save-solver-output-dt9s8: Succeeded
2020/07/30 11:43:06 Saved solver logs into '/tmp/specfem.solver-4proc-16cores-32nex_20200730_114306.log'
2020/07/30 11:43:06 All done!
2020/07/30 11:43:06 Done :)
```

As mentioned in the log messages, the solver output logfile
`output_solver.txt` is saved locally into
`/tmp/specfem.solver-4proc-16cores-32nex_20200730_114306.log`.

Cleanup
=======

Currently, only one instance/configuration of the application can
exist in the cluster (the object names / namespace are static), so the
workload objects must be deleted/recreated to rerun the application.

A helper flag helps deleting the relevant resources:

``
go run . -delete <flag>
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







