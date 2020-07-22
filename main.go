package main

import (
	"flag"
	"log"
	"strings"
	
	specfemv1 "gitlab.com/kpouget_psap/specfem-operator/pkg/apis/specfem/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

)

var delete_mode = false
var to_delete = map[string]bool{
	"all": false,
	"buildah": false,
	"port": false,
	"config": false,
	"mesher": false,
	"solver": false,
	"none": false,
}


func main() {
	var flag_delete = flag.String("delete", "", "solver,mesher,config,all|none")
	flag.Parse()
	if *flag_delete != "" {
		for _, opt_key := range strings.Split(*flag_delete, ",") {
			if _, ok := to_delete[opt_key]; ok {
				to_delete[opt_key] = true
				delete_mode = true
			} else {
				log.Fatalf("FATAL: wrong delete flag option: %v\n", opt_key)
			}
		}
	}
	if to_delete["all"] {
		for key, _ := range to_delete {
			to_delete[key] = true
		}
		delete_mode = true
	}

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
