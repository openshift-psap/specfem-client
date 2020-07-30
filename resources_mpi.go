package main

import (
	"fmt"
	"log"
	"strings"
	
	kubeflow "github.com/kubeflow/mpi-operator/pkg/apis/kubeflow/v1alpha2"
	kf_common "github.com/kubeflow/common/pkg/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
	
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime"

	specfemv1 "gitlab.com/kpouget_psap/specfem-api/pkg/apis/specfem/v1alpha1"
)



var mpijobResource = schema.GroupVersionResource{Version: "v1alpha2", Resource: "mpijobs", Group: "kubeflow.org"}

func newSpecfemMpiJob(app *specfemv1.SpecfemApp, stage string) (schema.GroupVersionResource, string, runtime.Object) {
	objName := "mpi-"+stage
	f32 := func(s int32) *int32 {
        return &s
    }
	
	policy := kf_common.CleanPodPolicyRunning
	np := app.Spec.Exec.Nproc
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
									ImagePullPolicy: corev1.PullAlways,
									Command: []string{
										"mpirun", "--allow-run-as-root",
										"-np", fmt.Sprintf("%d", np),
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
					Replicas: f32(np),
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							NodeSelector: app.Spec.Resources.WorkerNodeSelector,
							Containers: []corev1.Container{
								corev1.Container{					
									Name:  objName+"-worker",
									Image: "image-registry.openshift-image-registry.svc:5000/"+NAMESPACE+"/specfem:"+stage,
									ImagePullPolicy: corev1.PullAlways,
									VolumeMounts: []corev1.VolumeMount{
										corev1.VolumeMount{
											Name: "shared-volume",
											MountPath: "/mnt/shared/",
										},
										corev1.VolumeMount{
											Name: "bash-run-"+stage,
											MountPath: "/mnt/helper/run.sh",
											ReadOnly: true,
											SubPath: "run.sh",
										},
									},
									Env: []corev1.EnvVar{
										{Name: "OMP_NUM_THREADS", Value: fmt.Sprint(app.Spec.Exec.Ncore)},
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
									Name: "bash-run-"+stage,
									VolumeSource: corev1.VolumeSource{
										ConfigMap: &corev1.ConfigMapVolumeSource{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "bash-run-"+stage,
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

func newMesherMpiJob(app *specfemv1.SpecfemApp) (schema.GroupVersionResource, string, runtime.Object) {
	return newSpecfemMpiJob(app, "mesher")
}

func newSolverMpiJob(app *specfemv1.SpecfemApp) (schema.GroupVersionResource, string, runtime.Object) {
	return newSpecfemMpiJob(app, "solver")
}

func RunMpiJob(app *specfemv1.SpecfemApp, stage string) error {
	var newMpiJob ResourceCreator
	if stage == "mesher" {
		newMpiJob = newMesherMpiJob
	} else {
		newMpiJob = newSolverMpiJob
	}
	mpijobName, err := CreateResource(app, newMpiJob, stage)
	if err != nil || mpijobName == "" {
		return err
	}

	jobName := mpijobName+"-launcher"

	var logs *string = nil
	err = WaitWithJobLogs(jobName, "", &logs)
	if err != nil {
		return err
	}
	if logs == nil {
		return fmt.Errorf("Failed to get logs for job/%s (from mpijob/%s)", jobName, mpijobName)
	}
	
	
	fmt.Printf(*logs)

	if strings.Contains(*logs, "processes exited with non-zero status") {
		return fmt.Errorf("mpijob/%s failed to run properly (job/%s)", mpijobName, jobName)
	}

	if strings.Contains(*logs, "MPI_ABORT was invoked on rank") {
		return fmt.Errorf("mpijob/%s was aborted (job/%s)", mpijobName, jobName)
	}

	if strings.Contains(*logs, "ORTE was unable to reliably start") {
		return fmt.Errorf("mpijob/%s could not properly start (job/%s)", mpijobName, jobName)
	}

	if strings.Contains(*logs, "ORTE has lost communication with a remote daemon") {
		return fmt.Errorf("mpijob/%s could not properly communicate (job/%s)", mpijobName, jobName)
	}

	// check for failure
	err = WaitWithJobLogs(jobName, "", &logs)
	if err != nil {
		return err
	}
	
	log.Printf("MPI %s done!", stage)
	
	return nil
}

func RunMpiMesher(app *specfemv1.SpecfemApp) error {
	return RunMpiJob(app, "mesher")
}

func RunMpiSolver(app *specfemv1.SpecfemApp) error {
	return RunMpiJob(app, "solver")
}
