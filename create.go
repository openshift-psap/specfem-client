package main

import (
	"fmt"
	"log"

	specfemv1 "gitlab.com/kpouget_psap/specfem-operator/pkg/apis/specfem/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type ResourceCreator func(app *specfemv1.SpecfemApp)(schema.GroupVersionResource, string, runtime.Object)

func CreateResource(app *specfemv1.SpecfemApp,
	creatorFunction ResourceCreator, stage string) (string, error) {
	resType, objName, obj:= creatorFunction(app)
	
	if _, ok := to_delete[stage]; !ok {
		msg := fmt.Sprintf("Invalid stage '%v' for object %s/%s", stage, resType, objName)
		if delete_mode {
			return "", fmt.Errorf(msg)
		} else {
			log.Printf(msg)
		}
	}

	objDesc := fmt.Sprintf("%s/%s", resType.Resource, objName)
	var err error
	if delete_mode {		
		if to_delete[stage] {
			log.Printf("Delete %s | %s", objDesc, stage)
			err = client.Delete(resType, objName)
			if err != nil && !errors.IsNotFound(err) {
				log.Printf("Could not delete %s: %+v", objDesc, err)
			}
		} else {
			log.Printf("Keep %s | %s", objDesc, stage)
		}
		objName = ""
	} else {
		log.Printf("Create %s", objDesc)
		err = client.Create(resType, obj)
		if err != nil && !errors.IsAlreadyExists(err) {
			log.Printf("Failed to create %s: %+v", objDesc, err)
			return "", err
		}
	}

	

	return objName, nil
}

func CreateBaseAndMesherImages(app *specfemv1.SpecfemApp) error {
	_, err := CreateResource(app, newImageStream, "all")
	if err != nil {
		return err
	}

	if err := CreateAndWaitBuildConfig(app, newBaseImageBuildConfig, "all"); err != nil {
		return err
	}

	if err := CheckImageTag(app, "specfem:base", "all"); err != nil {
		return err
	}
	
	if err := CreateAndWaitBuildConfig(app, newMesherImageBuildConfig, "config"); err != nil {
		return err
	}

	if err := CheckImageTag(app, "specfem:mesher", "config"); err != nil {
		return err
	}
	
	return nil
}

func CreateAndWaitBuildConfig(app *specfemv1.SpecfemApp, creatorFunction ResourceCreator, stage string) error{
	bcName, err := CreateResource(app, creatorFunction, stage)
	if err != nil || bcName == "" {
		return err
	}
	log.Printf("BuildConfig '%s' created", bcName)

	err = WaitForBuildComplete(bcName)
	if err != nil {
		log.Fatalf("Failed to wait for %s: %v", bcName, err)
		return err
	}

	return nil
}

func CreateResources(app *specfemv1.SpecfemApp) error {
	if err := CreateBaseAndMesherImages(app); err != nil {
		return err
	}

	if _, err := CreateResource(app, newMesherScriptCM, "all"); err != nil {
		return err
	}

	if err := CreateSharedPVC(app); err != nil {
		return err
	}
	
	if app.Spec.Exec.Nproc == 1 {		
		if err := RunSeqMesher(app); err != nil {
			return err
		}
	} else {
		if err := RunMpiMesher(app); err != nil {
			return err
		}
	}
	fmt.Println("----------")
	CreateSolverImage := CreateSolverImage_buildah
	if err := CreateSolverImage(app); err != nil {
		return err
	}

	if err := CheckImageTag(app, "specfem:solver", "mesher"); err != nil {
		return err
	}

	if _, err := CreateResource(app, newBashRunSolverCM, "all"); err != nil {
		return err
	}
	
	if app.Spec.Exec.Nproc == 1 {
		if err := RunSeqSolver(app); err != nil {
			return err
		}
	} else {
		if err := RunMpiSolver(app); err != nil {
			return err
		}
	}

	if err := RunSaveSolverOutput(app); err != nil {
		return err
	}

	log.Println("All done!")

	return nil
}

func CheckImageTag(app *specfemv1.SpecfemApp, imagetagName string, stage string) error {
	var err error = nil
	var gvr = imagestreamtagResource
	
	if delete_mode {		
		if to_delete[stage] {
			log.Printf("Delete imagestreamtag/%s | %s", imagetagName, stage)
			err = client.Delete(gvr, imagetagName)
		}
	} else {
		log.Printf("Checking imagestreamtag/%s | %s", imagetagName, stage)
		_, err = client.Get(gvr, imagetagName)
	}
	
	return err
}


func CreateSharedPVC(app *specfemv1.SpecfemApp) error {
	USE_DEFAULT_PCV_SC := false
	
	if USE_DEFAULT_PCV_SC {
		if _, err := CreateResource(app, newDefaultPVC, "mesher"); err != nil {
			return err
		}
	} else {
		if err := CreateEfsPVC(app); err != nil {
			return err
		}
	}

	return nil
}
