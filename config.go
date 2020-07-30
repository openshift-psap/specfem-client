package main

import (
	"fmt"
	"math"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	specfemv1 "gitlab.com/kpouget_psap/specfem-api/pkg/apis/specfem/v1alpha1"
)

func checkSpecfemConfig(app *specfemv1.SpecfemApp) error {
	
	actual_nproc_val := int32(math.Sqrt(float64(app.Spec.Exec.Nproc)))
	if actual_nproc_val*actual_nproc_val != app.Spec.Exec.Nproc {
		return fmt.Errorf("Invalid nproc value (%d), it must be a perfect square ...",
			app.Spec.Exec.Nproc)
	}

	nex := app.Spec.Specfem.Nex
	if nex % (8*actual_nproc_val) != 0 {
		return fmt.Errorf("NEX(=%d) must be a multiple of 8*NPROC(=%d)", nex, actual_nproc_val)
	}
	
	return nil
}

func getSpecfemApp() *specfemv1.SpecfemApp {
	return &specfemv1.SpecfemApp{
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
				Ncore: 8,
			},
			Specfem: specfemv1.SpecfemSpec{
				Nex: 32,
			},
			Resources: specfemv1.ResourcesSpec{
				StorageClassName: "aws-efs",
				WorkerNodeSelector: map[string]string{
					"node-role.kubernetes.io/worker": "",
				},
				SlotsPerWorker: 1,
			},
		},
	}
}
