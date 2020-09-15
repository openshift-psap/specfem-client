package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
	
	errs "github.com/pkg/errors"
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

func getSpecfemConfig(configName string) (*specfemv1.SpecfemApp, error) {
	configs, err := fetchManifestsFromPath("config")
	if err != nil {
		return nil, errs.Wrap(err, "Failed to read config files ...")
	}

	yamlSpecfem, ok := configs[configName+".yaml"]

	if !ok {
		return nil, fmt.Errorf("Could not find config file 'config/%s.yaml'", configName)
	}

	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme,
                scheme.Scheme)

	var app specfemv1.SpecfemApp
	if _, _, err := s.Decode([]byte(yamlSpecfem), nil, &app); err != nil {
		return nil, errs.Wrap(err, fmt.Sprintf("Could not (re)parse config file 'config/%s.yaml'", configName))
	}
	
	fmt.Printf("Application configured from config/%s.yaml: \n%s", configName, yamlSpecfem)
	
	return &app, nil
}

var manifests map[string]string

func FetchManifests() error {
	var FETCH_FROM_CM = false
	var err error
	
	if FETCH_FROM_CM {
		return fetchManifestsFromCM()
	} else {
		manifests, err = fetchManifestsFromPath("manifests")
	}
	return err
}

func fetchManifestsFromCM() error {
	var config *unstructured.Unstructured
	var err error
	var found bool

	cm := &unstructured.Unstructured{}
	cm.SetAPIVersion("v1")
	cm.SetKind("ConfigMap")

	cmName :="specfem-app-manifests"
	config, err = client.Get(cmResource, cmName)
	if err != nil {
		return errs.Wrap(err, "Failed to fetch the manifests ConfigMap ...")
	}
	
	manifests_, found, err := unstructured.NestedMap(config.Object, "data")
	
	if !found {
		return fmt.Errorf("configmap/%s not found ...", cmName)
	} else if err != nil {
		return errs.Wrap(err, "Unexpected error while fetching configmap/"+cmName+" ...")
	}
	log.Println(manifests_)
	for k, v := range manifests_ {
		manifests[k] = v.(string)
	}

	return nil
}

func filePathWalkDir(root string) ([]string, error) {
	var filenames []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Println("DEBUG: error in filepath.Walk on %s: %v", root, err)
			return nil
		}
		if !info.IsDir() {
			filenames = append(filenames, path)
		}
		return nil
	})
	return filenames, err
}

func fetchManifestsFromPath(path string) (map[string]string, error) {
	path_manifests := make(map[string]string)
	filenames, err := filePathWalkDir(path)
	if err != nil {
		return nil, errs.Wrap(err, fmt.Sprintf("Failed to fetch the manifests from '%s'", path))
	}
	
	for _, filename := range filenames {
		buffer, err := ioutil.ReadFile(filename)
		
		if err != nil {
			// ignore IsNotExist errors...this is expected
			if os.IsNotExist(err) {
				continue
			}
			return nil, errs.Wrap(err, fmt.Sprintf("Failed to read manifest '%s'", filename))
		}
		_, fname := filepath.Split(filename)

		path_manifests[fname] = string(buffer)
	}

	return path_manifests, nil
}
