package main

import (
	"fmt"
	
	kubeflow "github.com/kubeflow/mpi-operator/pkg/apis/kubeflow/v1alpha2"
	kf_common "github.com/kubeflow/common/pkg/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
	
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime"

	specfemv1 "gitlab.com/kpouget_psap/specfem-operator/pkg/apis/specfem/v1"

)



var mpijobResource = schema.GroupVersionResource{Version: "v1alpha2", Resource: "mpijobs", Group: "kubeflow.org"}

func newMesherMpiJob(app *specfemv1.SpecfemApp) (schema.GroupVersionResource, string, runtime.Object) {
	objName := "mpi-mesher"
	f32 := func(s int32) *int32 {
        return &s
    }
	
	policy := kf_common.CleanPodPolicyRunning
	return mpijobResource, objName, &kubeflow.MPIJob{
		TypeMeta: metav1.TypeMeta{Kind: "MPIJob", APIVersion: "kubeflow.org/v1alpha2"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      objName,
			Namespace: NAMESPACE,
		},
		Spec: kubeflow.MPIJobSpec{
			SlotsPerWorker: f32(2),
			CleanPodPolicy: &policy,
			MPIReplicaSpecs: map[kubeflow.MPIReplicaType]*kf_common.ReplicaSpec{
				kubeflow.MPIReplicaTypeLauncher: &kf_common.ReplicaSpec{
					Replicas: f32(1),
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								corev1.Container{					
									Name:  objName+"-mpi-auncher",
									Image: "image-registry.openshift-image-registry.svc:5000/"+NAMESPACE+"/specfem:base",
									Command: []string{
										"/usr/lib64/openmpi/bin/mpirun", "--allow-run-as-root",
										"-np", fmt.Sprintf("%d", app.Spec.Exec.Nproc),
										"-bind-to", "none",
										"-map-by", "slot",
										"-mca", "pml", "ob1",
										"-mca", "btl", "^openib",
										"env",
									},
								},
							},
						},
					},
				},
				kubeflow.MPIReplicaTypeWorker: &kf_common.ReplicaSpec{
					Replicas: f32(2),
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								corev1.Container{					
									Name:  objName+"-mpi-worker",
									Image: "image-registry.openshift-image-registry.svc:5000/"+NAMESPACE+"/specfem:mesher",
									VolumeMounts: []corev1.VolumeMount{
										corev1.VolumeMount{
											Name: "shared-volume",
											MountPath: "/mnt/shared/",
										},
										corev1.VolumeMount{
											Name: "bash-run-solver",
											MountPath: "/mnt/helper/run.sh",
											ReadOnly: true,
											SubPath: "run.sh",
										},
									},
								},
							},
							Volumes: []corev1.Volume{
								corev1.Volume{
									Name: "shared-volume",
									VolumeSource: corev1.VolumeSource{
										PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
											ClaimName: "specfem",
										},
									},
								},
								corev1.Volume{
									Name: "bash-run-solver",
									VolumeSource: corev1.VolumeSource{
										ConfigMap: &corev1.ConfigMapVolumeSource{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "bash-run-solver",
											},
											DefaultMode: f32(0777),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func newSolverMpiJob(app *specfemv1.SpecfemApp) (schema.GroupVersionResource, string, runtime.Object) {
	objName := "mpi-solver"

	return mpijobResource, objName, &kubeflow.MPIJob{
		
	}
}

func RunMpiMesher(app *specfemv1.SpecfemApp) error {
	jobName, err := CreateResource(app, newMesherMpiJob, "mesher")
	if err != nil || jobName == "" {
		return err
	}

	return nil
}

func RunMpiSolver(app *specfemv1.SpecfemApp) error {
	jobName, err := CreateResource(app, newSolverMpiJob, "solver")
	if err != nil || jobName == "" {
		return err
	}

	if err := WaitWithJobLogs(jobName, "", nil); err != nil {
		return err
	}
	
	return nil
}
