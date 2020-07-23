package main

import (
	"flag"
	"log"
	
	specfemv1 "gitlab.com/kpouget_psap/specfem-operator/pkg/apis/specfem/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

)

var DELETE_KEYS = []string{
	"all",
	"buildah",
	"port",
	"config",
	"mesher",
	"solver",
}
var delete_mode = false

var to_delete = map[string]bool{}

func initDelete() {
	var flag_delete = flag.String("delete", "", "solver,mesher,config,all|none")
	flag.Parse()

	for _, key := range DELETE_KEYS {
		delete_it := (key == *flag_delete) || delete_mode
		to_delete[key] = delete_it 
		if delete_it {
			if !delete_mode {
				log.Println("Stages to delete:")
			}
			delete_mode = true
			log.Println("- ", key)
		}
	}

	if *flag_delete == "" {
		return
	}
	
	if !delete_mode {
		log.Fatalf("FATAL: wrong delete flag option: %v\n", *flag_delete)
	}
}

func main() {
	initDelete()
	
	if err := InitClient(); err != nil {
		log.Fatalf("FATAL: %+v\n", err)
	}

	app := &specfemv1.SpecfemApp{
		ObjectMeta: metav1.ObjectMeta{
			Name: "specfemapp",
		},
		Spec: specfemv1.SpecfemAppSpec{
			Git: specfemv1.GitSpec{
				Uri: "https://gitlab.com/kpouget_psap/specfem3d_globe.git",
				Ref: "mockup",
			},
			Exec: specfemv1.ExecSpec{
				Nproc: 4,
				Ncore: 16,
			},
			Specfem: specfemv1.SpecfemSpec{
				Nex: 16,
			},
		},
	}

	if err := CreateResources(app); err != nil {
		log.Fatalf("FATAL: %+v\n", err)
	}
	log.Println("Done :)")
}
