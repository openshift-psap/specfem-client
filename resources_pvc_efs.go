package main

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/api/resource"

	specfemv1 "gitlab.com/kpouget_psap/specfem-operator/pkg/apis/specfem/v1"
)

var pvResource = schema.GroupVersionResource{Version: "v1", Resource: "persistentvolumes"}
var scResource = schema.GroupVersionResource{Version: "v1", Resource: "storageclasses", Group: "storage.k8s.io"}

func newEfsSc(app *specfemv1.SpecfemApp) (schema.GroupVersionResource, string, runtime.Object) {
	objName := "specfem-efs"
	bindingMode := storagev1.VolumeBindingWaitForFirstConsumer
	return scResource, objName, &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
    	    Name: objName,
			Labels:    map[string]string{
				"app": "specfem",
			},
        },
		Provisioner: "efs.csi.aws.com",
		VolumeBindingMode: &bindingMode,
	}
}
func newEfsPv(app *specfemv1.SpecfemApp) (schema.GroupVersionResource, string, runtime.Object) {
	objName := "specfem-efs"

	return pvResource, objName, &corev1.PersistentVolume{
        ObjectMeta: metav1.ObjectMeta{
    	    Name: objName,
			Labels:    map[string]string{
				"app": "specfem",
			},
        },
        Spec: corev1.PersistentVolumeSpec{
			StorageClassName: "specfem-efs",
            PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRecycle,
            AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany,},
            Capacity: corev1.ResourceList{
            	corev1.ResourceName(corev1.ResourceStorage): resource.MustParse("1Gi"),
            
            },
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				CSI: &corev1.CSIPersistentVolumeSource{
					Driver: "efs.csi.aws.com",
					VolumeHandle: "fs-642da695",
				},
			},
    	},
	}
}

func newEfsPvc(app *specfemv1.SpecfemApp) (schema.GroupVersionResource, string, runtime.Object) {
	objName := "specfem"
	scName := "specfem-efs"
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
			StorageClassName: &scName,
		},
	}
}

func CreateEfsPVC(app *specfemv1.SpecfemApp) error {
	if _, err := CreateResource(app, newEfsSc, "all"); err != nil {
		return err
	}

	if _, err := CreateResource(app, newEfsPv, "all"); err != nil {
		return err
	}

	if _, err := CreateResource(app, newEfsPvc, "mesher"); err != nil {
		return err
	}

	return nil
}
