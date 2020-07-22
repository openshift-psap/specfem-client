package main

import (
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"

	buildv1 "github.com/openshift/api/build/v1"
	imagev1 "github.com/openshift/api/image/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	specfemv1 "gitlab.com/kpouget_psap/specfem-operator/pkg/apis/specfem/v1"
)

var imagestreamtagResource         = schema.GroupVersionResource{Version: "v1", Resource: "imagestreamtags", Group: "image.openshift.io"}
var podResource         = schema.GroupVersionResource{Version: "v1", Resource: "pods"}
var pvcResource         = schema.GroupVersionResource{Version: "v1", Resource: "persistentvolumeclaims"}
var cmResource          = schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}
var svcResource         = schema.GroupVersionResource{Version: "v1", Resource: "services"}
var jobResource         = schema.GroupVersionResource{Version: "v1", Resource: "jobs",         Group: "batch"}
var buildconfigResource = schema.GroupVersionResource{Version: "v1", Resource: "buildconfigs", Group: "build.openshift.io"}
var imagestreamResource = schema.GroupVersionResource{Version: "v1", Resource: "imagestreams", Group: "image.openshift.io"}
var routeResource       = schema.GroupVersionResource{Version: "v1", Resource: "routes",       Group: "route.openshift.io"}


var NAMESPACE = "specfem"

func newImageStream(app *specfemv1.SpecfemApp) (schema.GroupVersionResource, string, runtime.Object) {
	objName := "specfem"

	return imagestreamResource, objName, &imagev1.ImageStream{
		ObjectMeta: metav1.ObjectMeta{
			Name:      objName,
			Namespace: NAMESPACE,
			Labels:    map[string]string{
				"app": "specfem",
			},
		},
	}
}

func newBaseImageBuildConfig(app *specfemv1.SpecfemApp) (schema.GroupVersionResource, string, runtime.Object) {
	objName := "specfem-base-image"

	return buildconfigResource, objName, &buildv1.BuildConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      objName,
			Namespace: NAMESPACE,
			Labels:    map[string]string{
				"app": "specfem",
			},
		},
		Spec: buildv1.BuildConfigSpec{
			CommonSpec: buildv1.CommonSpec{
				Strategy: buildv1.BuildStrategy{
					Type: buildv1.DockerBuildStrategyType,
					DockerStrategy: &buildv1.DockerBuildStrategy{
						From: &corev1.ObjectReference{
							Kind: "DockerImage",
							Name: "docker.io/ubuntu:eoan",
						},
					},
				},
				Source: buildv1.BuildSource{
					Type: buildv1.BuildSourceGit,
					Git: &buildv1.GitBuildSource{
						URI: "https://gitlab.com/kpouget_psap/specfem-on-openshift.git",
						Ref: "00_specfem-base-container",
					},
				},
				Output: buildv1.BuildOutput{
					To: &corev1.ObjectReference{
						Kind: "ImageStreamTag",
						Name: "specfem:base",
					},
				},
			},
			Triggers: []buildv1.BuildTriggerPolicy{
				buildv1.BuildTriggerPolicy{
					Type: buildv1.ConfigChangeBuildTriggerType,
				},
			},
		},
	}
}

func newMesherImageBuildConfig(app *specfemv1.SpecfemApp) (schema.GroupVersionResource, string, runtime.Object) {
	objName := "specfem-mesher-image"

	return buildconfigResource, objName, &buildv1.BuildConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      objName,
			Namespace: NAMESPACE,
			Labels:    map[string]string{
				"app": "specfem",
			},
		},
		Spec: buildv1.BuildConfigSpec{
			CommonSpec: buildv1.CommonSpec{
				Strategy: buildv1.BuildStrategy{
					Type: buildv1.DockerBuildStrategyType,
					DockerStrategy: &buildv1.DockerBuildStrategy{
						From: &corev1.ObjectReference{
							Kind: "ImageStreamTag",
							Name: "specfem:base",
						},
						Env: []corev1.EnvVar{
							{Name: "SPECFEM_GIT_REPO", Value: app.Spec.Git.Uri},
							{Name: "SPECFEM_GIT_BRANCH", Value: app.Spec.Git.Ref},
							{Name: "OMP_NUM_THREADS", Value: fmt.Sprint(app.Spec.Exec.Ncore)},
							{Name: "SPECFEM_NPROC", Value: fmt.Sprint(app.Spec.Exec.Nproc)},
							{Name: "SPECFEM_NEX", Value: fmt.Sprint(app.Spec.Specfem.Nex)},
						},
					},
				},
				Source: buildv1.BuildSource{
					Type: buildv1.BuildSourceGit,
					Git: &buildv1.GitBuildSource{
						URI: "https://gitlab.com/kpouget_psap/specfem-on-openshift.git",
						Ref: "01_specfem-mesher-container",
					},
				},
				Output: buildv1.BuildOutput{
					To: &corev1.ObjectReference{
						Kind: "ImageStreamTag",
						Name: "specfem:mesher",
					},
				},
			},
			Triggers: []buildv1.BuildTriggerPolicy{
				buildv1.BuildTriggerPolicy{
					Type: buildv1.ImageChangeBuildTriggerType,
				},
			},
		},
	}
}

func newSaveSolverOutputJob(app *specfemv1.SpecfemApp) (schema.GroupVersionResource, string, runtime.Object) {
	f32 := func(s int32) *int32 {
        return &s
    }
	f64 := func(s int64) *int64 {
        return &s
    }
	objName := "save-solver-output"

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
			ActiveDeadlineSeconds: f64(10),
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
							Image: "docker.io/centos:7",
							Command: []string{
								"cat", "/mnt/shared/OUTPUT_FILES/output_solver.txt",
							},
							VolumeMounts: []corev1.VolumeMount{
								corev1.VolumeMount{
									Name: "shared-volume",
									MountPath: "/mnt/shared/",
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
					},
				},
			},
		},
	}
}
