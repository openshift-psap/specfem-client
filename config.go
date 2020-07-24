package main

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	specfemv1 "gitlab.com/kpouget_psap/specfem-operator/pkg/apis/specfem/v1"
)

func checkSpecfemConfig(app *specfemv1.SpecfemApp) error {
	nex := app.Spec.Specfem.Nex
	nproc := app.Spec.Exec.Nproc
	if nex % (8*nproc) != 0 {
		return fmt.Errorf("NEX(=%d) must be a multiple of 8*NPROC(=%d)", nex, nproc)
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
				Nproc: 2, // number of MPI proc : nproc*nproc
				Ncore: 16,
			},
			Specfem: specfemv1.SpecfemSpec{
				Nex: 32,
			},
		},
	}
}
