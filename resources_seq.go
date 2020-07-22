package main

import (
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	specfemv1 "gitlab.com/kpouget_psap/specfem-operator/pkg/apis/specfem/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)


func newSpecfemPVC(app *specfemv1.SpecfemApp) (schema.GroupVersionResource, string, runtime.Object) {
	objName := "specfem"

	return pvcResource, objName, &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      objName,
			Namespace: NAMESPACE,
			Labels:    map[string]string{
				"app": "specfem",
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
		},
	}
}

func newMesherScriptCM(app *specfemv1.SpecfemApp) (schema.GroupVersionResource, string, runtime.Object) {
	objName := "bash-run-mesher"

	return cmResource, objName, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      objName,
			Namespace: NAMESPACE,
			Labels:    map[string]string{
				"app": "specfem",
			},
		},
		Data: map[string]string{
			"run.sh": `
set -ex

cd app && ./bin/xmeshfem3D
rm -f /mnt/shared/mesher.tgz
tar cfz /mnt/shared/mesher.tgz OUTPUT_FILES/ DATABASES_MPI/
cat OUTPUT_FILES/output_mesher.txt | grep "buffer creation in seconds"
`,
		},
	}
}

func newMesherJob(app *specfemv1.SpecfemApp) (schema.GroupVersionResource, string, runtime.Object) {
	f32 := func(s int32) *int32 {
        return &s
    }
	f64 := func(s int64) *int64 {
        return &s
    }
	objName := "run-mesher"

	return jobResource, objName, &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "run-mesher",
			Namespace: NAMESPACE,
			Labels:    map[string]string{
				"app": "specfem",
			},
		},
		Spec: batchv1.JobSpec{
			Parallelism: f32(1),
			Completions: f32(1),
			ActiveDeadlineSeconds: f64(150),
			BackoffLimit: f32(1),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      objName,
					Namespace: NAMESPACE,
					Labels:    map[string]string{
						"app": objName,
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						corev1.Container{
							Name: objName,
							ImagePullPolicy: corev1.PullAlways,
							Image: fmt.Sprintf("image-registry.openshift-image-registry.svc:5000/%v/specfem:mesher", NAMESPACE),
							Env: []corev1.EnvVar{
								{Name: "OMPI_MCA_btl_base_warn_component_unused", Value: "0"},
							},
							Command: []string{
								"bash", "-c",
								"/mnt/helper/run.sh",
							},
							VolumeMounts: []corev1.VolumeMount{
								corev1.VolumeMount{
									Name: "shared-volume",
									MountPath: "/mnt/shared/",
								},
								corev1.VolumeMount{
									Name: "bash-run-mesher",
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
	}
}

func newBashRunSolverCM(app *specfemv1.SpecfemApp) (schema.GroupVersionResource, string, runtime.Object) {
	objName := "bash-run-solver"

	return cmResource, objName, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bash-run-solver",
			Namespace: NAMESPACE,
			Labels:    map[string]string{
				"app": "specfem",
			},
		},
		Data: map[string]string{
			"run.sh": `
set -x

cd app || exit

if ! tar xvf /mnt/shared/mesher.tgz DATABASES_MPI; then
  echo "Failed to extract MPI database ..."
  exit 1
fi

SPECFEM_TIMEOUT=15
timeout $SPECFEM_TIMEOUT ./bin/xspecfem3D
ret="$?"
if [ "$ret" == 124 ]; then
  echo "Timed out after ${SPECFEM_TIMEOUT}s"
  echo "Timed out after ${SPECFEM_TIMEOUT}s" >> oc.build.log
elif [ "$ret" != 0 ]; then
  echo "Execution failed ... (ret=$ret)"
  exit 1
fi

set -x
rm -rf /mnt/shared/OUTPUT_FILES/
mkdir /mnt/shared/OUTPUT_FILES/
cp oc.build.log OUTPUT_FILES/* /mnt/shared/OUTPUT_FILES/
env > /mnt/shared/OUTPUT_FILES/env
cat OUTPUT_FILES/output_solver.txt | grep "Total elapsed time in seconds"
`,
		},
	}
}

func newRunSolverJob(app *specfemv1.SpecfemApp) (schema.GroupVersionResource, string, runtime.Object) {
	f32 := func(s int32) *int32 {
        return &s
    }
	f64 := func(s int64) *int64 {
        return &s
    }
	objName := "run-solver"

	return jobResource, objName, &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      objName,
			Namespace: NAMESPACE,
			Labels:    map[string]string{
				"app": "specfem",
			},
		},
		Spec: batchv1.JobSpec{
			Parallelism: f32(1),
			Completions: f32(1),
			ActiveDeadlineSeconds: f64(1500),
			BackoffLimit: f32(1),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      objName,
					Namespace: NAMESPACE,
					Labels:    map[string]string{
						"app": "specfem",
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						corev1.Container{
							Name: objName,
							ImagePullPolicy: corev1.PullAlways,
							Image: fmt.Sprintf("image-registry.openshift-image-registry.svc:5000/%v/specfem:solver", NAMESPACE),
							Command: []string{
								"bash", "-c", "/mnt/helper/run.sh",
							},
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
	}
}

