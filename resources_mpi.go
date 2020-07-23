package main

import (
	"fmt"
	"log"
	
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
									Name:  objName+"-launcher",
									Image: "image-registry.openshift-image-registry.svc:5000/"+NAMESPACE+"/specfem:base",
									Command: []string{
										"/usr/bin/mpirun.openmpi", "--allow-run-as-root",
										"-np", fmt.Sprintf("%d", app.Spec.Exec.Nproc),
										"-bind-to", "none",
										"-map-by", "slot",
										"-mca", "pml", "ob1",
										"-mca", "btl", "^openib",
										"bash",
										"-c",
										"/mnt/helper/run.sh",
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
									Name:  objName+"-worker",
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
									Name: "bash-run-mesher",
									VolumeSource: corev1.VolumeSource{
										ConfigMap: &corev1.ConfigMapVolumeSource{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "bash-run-mesher",
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

	var logs *string = nil
	err = WaitWithJobLogs(jobName+"-launcher", "", &logs)
	if err != nil {
		return err
	}
	if logs == nil {
		return fmt.Errorf("Failed to get logs for job/%s", jobName)
	}
	
	fmt.Printf(*logs)
	
	log.Printf("MPI mesher done!")

	return nil
}

func RunMpiSolver(app *specfemv1.SpecfemApp) error {
	jobName, err := CreateResource(app, newSolverMpiJob, "solver")
	if err != nil || jobName == "" {
		return err
	}

	//need to wait for the job here
	
	if err := WaitWithJobLogs(jobName, "", nil); err != nil {
		return err
	}
	
	return nil
}
