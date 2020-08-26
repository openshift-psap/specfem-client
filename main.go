package main

import (
	"flag"
	"log"
	"os"
	
	specfemv1 "gitlab.com/kpouget_psap/specfem-api/pkg/apis/specfem/v1alpha1"
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
	var err error
	
	initDelete()

	if err = InitClient(); err != nil {
		log.Fatalf("FATAL: %+v\n", err)
	}

	if err = FetchManifests(); err != nil {
		log.Fatalf("FATAL: %+v\n", err)
	}
	
	var configName string
	if len(os.Args) > 1 && os.Args[1][0] != '-' {
		configName = os.Args[1]
	} else {
		configName = "specfem-sample"
	}
	
	var app *specfemv1.SpecfemApp

	if app, err = getSpecfemConfig(configName); err != nil {
		log.Fatalf("FATAL: failed to get the application configuration: %+v\n", err)
	}

	if err = checkSpecfemConfig(app); err != nil {
		log.Fatalf("FATAL: config error: %+v\n", err)
	}

	if err = RunSpecfem(app); err != nil {
		log.Fatalf("FATAL: %v\n", err)
	}
	
	log.Println("Done :)")
}
