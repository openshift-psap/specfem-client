package main

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/api/resource"

	specfemv1 "gitlab.com/kpouget_psap/specfem-api/pkg/apis/specfem/v1alpha1"
)

func newEfsPvc(app *specfemv1.SpecfemApp) (schema.GroupVersionResource, string, runtime.Object) {
	objName := "specfem"
	storageClass := "aws-efs"
	volumeMode := corev1.PersistentVolumeFilesystem
	return pvcResource, objName, &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      objName,
			Namespace: NAMESPACE,
			Labels:    map[string]string{
				"app": "specfem",
			},
			Annotations: map[string]string{
				"volume.beta.kubernetes.io/storage-provisioner": "openshift.org/aws-efs",
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
