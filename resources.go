package main

import (
	"fmt"
	"math"

	buildv1 "github.com/openshift/api/build/v1"
	imagev1 "github.com/openshift/api/image/v1"
	
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
	batchv1 "k8s.io/api/batch/v1"
	
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/api/resource"

	specfemv1 "gitlab.com/kpouget_psap/specfem-api/pkg/apis/specfem/v1alpha1"
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
var secretResource       = schema.GroupVersionResource{Version: "v1", Resource: "secrets"}

var NAMESPACE = "specfem"

var BASE_GIT_REPO = "https://gitlab.com/kpouget_psap/specfem-client.git"
var USE_UBI_BASE_IMAGE = true

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
	build_branch := "00_specfem-base-container"
	
	var from_image string
	if USE_UBI_BASE_IMAGE {
		from_image = "registry.access.redhat.com/ubi8/ubi"
		objName += "-ubi"
		build_branch += "_ubi"
	} else {
		from_image = "docker.io/ubuntu:eoan"
		objName += "-ubuntu"
		build_branch += "_ubuntu"
	}
	
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
							Name: from_image,
						},
						Env: []corev1.EnvVar{
							{Name: "SPECFEM_GIT_REPO", Value: app.Spec.Git.Uri},
							{Name: "SPECFEM_GIT_BRANCH", Value: app.Spec.Git.Ref},
						},						
					},
				},
				Source: buildv1.BuildSource{
					Type: buildv1.BuildSourceGit,
					Git: &buildv1.GitBuildSource{
						URI: BASE_GIT_REPO,
						Ref: build_branch,
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

	nproc_value := fmt.Sprint(int32(math.Sqrt(float64(app.Spec.Exec.Nproc))))
	
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
							{Name: "SPECFEM_NPROC", Value: nproc_value},
							{Name: "SPECFEM_NEX", Value: fmt.Sprint(app.Spec.Specfem.Nex)},
						},
					},
				},
				Source: buildv1.BuildSource{
					Type: buildv1.BuildSourceGit,
					Git: &buildv1.GitBuildSource{
						URI: BASE_GIT_REPO,
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

func newPVC(app *specfemv1.SpecfemApp) (schema.GroupVersionResource, string, runtime.Object) {
	objName := "specfem"
	storageClass := app.Spec.Resources.StorageClassName
	volumeMode := corev1.PersistentVolumeFilesystem
	return pvcResource, objName, &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      objName,
			Namespace: NAMESPACE,
			Labels:    map[string]string{
				"app": "specfem",
			},
			Finalizers: []string{
				"kubernetes.io/pvc-protection",
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
			StorageClassName: &storageClass,
			VolumeMode: &volumeMode,
		},
	}
}
