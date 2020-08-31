package main

import (
	"fmt"
	"log"
	"os"
	"time"
	
	errs "github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	specfemv1 "gitlab.com/kpouget_psap/specfem-api/pkg/apis/specfem/v1alpha1"
)

func RunMesherSolver(app *specfemv1.SpecfemApp, stage string) error {
	if app.Spec.Exec.Nproc == 0 {		
		return RunSeqMesherSolver(app, stage)
	} else {
		return RunMpiJob(app, stage)
	}
}

func RunSeqMesherSolver(app *specfemv1.SpecfemApp, stage string) error {
	jobName, err := CreateYamlResource(app, yamlRunSeqMesherSolverJob(stage), stage)
	if err != nil || jobName == "" {
		return err
	}

	if err = WaitWithJobLogs(jobName, "", nil); err != nil {
		return err
	}

	return nil
}

func CreateBaseAndMesherImages(app *specfemv1.SpecfemApp) error {
	_, err := CreateYamlResource(app, yamlImageStream, "all")
	if err != nil {
		return  errs.Wrap(err, "Cannot create resource for yamlImageStream")
	}

	if err := CreateAndWaitYamlBuildConfig(app, yamlBaseImageBuildConfig, "all"); err != nil {
		return err
	}

	if err := CheckImageTag("specfem:base", "all"); err != nil {
		return err
	}
	
	if err := CreateAndWaitYamlBuildConfig(app, yamlMesherImageBuildConfig, "config"); err != nil {
		return err
	}

	if err := CheckImageTag("specfem:mesher", "config"); err != nil {
		return err
	}

	return nil
}

func CreateSolverImage(app *specfemv1.SpecfemApp) error {
	if err:= CreateAndWaitYamlBuildConfig(app, newMesher2SolverHelperBuildConfig, "mesher"); err != nil {
		return err
	}

	builderNodeName, err := GetOrSetNodeTag("buildah.specfem.build")

	if err != nil {
		return errs.Wrap(err, "Could not find or define builder node ...")
	}
	
	for _, yamlResource := range[]YamlResourceSpec{
		yamlBuildahMesher2SolverScriptCM, yamlTunedLoadFuseModule} {	
		if _, err := CreateYamlResource(app, yamlResource, "mesher"); err != nil {
			return err
		}
	}

	if !delete_mode {
		if err := CheckImageTag("specfem:solver", "mesher"); err == nil {
			log.Println("Found solver image, don't recreate it.")
			return nil
		}
		
		if err := WaitForTunedProfile("openshift-fuse", builderNodeName, 
			metav1.ListOptions{LabelSelector: "openshift-app=tuned"}); err != nil {
			return err
		}
	}
	
	podName, err := CreateYamlResource(app, yamlBuildahBuildSolverImagePod, "mesher")
	if err != nil {
		return err
	}

	if podName != "" {
		if err := WaitWithPodLogs("", podName, "", nil); err != nil {
			return err
		}
	}

	if err := CheckImageTag("specfem:solver", "mesher"); err != nil {
		return err
	}
	
	return nil
}

func RunMpiJob(app *specfemv1.SpecfemApp, stage string) error {
	mpijobName, err := CreateYamlResource(app, yamlRunMpiMesherSolverJob(stage), stage)
	if err != nil || mpijobName == "" {
		return err
	}

	if err = WaitMpiJob(mpijobName); err != nil {
		return err
	}

	log.Printf("MPI %s done!", stage)

	return nil
}



func RunSaveSolverOutput(app *specfemv1.SpecfemApp) error {
	jobName, err := CreateYamlResource(app, yamlSaveSolverOutputJob, "solver")
	if err != nil {
		return err
	}

	if jobName == "" {
		return nil
	}
	
	var logs *string = nil
	err = WaitWithJobLogs(jobName, "", &logs)
	if err != nil {
		return err
	}
	if logs == nil {
		return fmt.Errorf("Failed to get logs for job/%s", jobName)
	}

	date_uid := time.Now().Format("20060102_150405")
	
    SAVELOG_FILENAME := fmt.Sprintf("/tmp/specfem.solver-%dproc-%dcores-%dnex_%s.log",
		app.Spec.Exec.Nproc, app.Spec.Exec.Ncore, app.Spec.Specfem.Nex, date_uid)
		
	output_f, err := os.Create(SAVELOG_FILENAME)

	if err != nil {
		return err
	}
	
	defer output_f.Close()

	output_f.WriteString(*logs)

	log.Printf("Saved solver logs into '%s'", SAVELOG_FILENAME)
	
	return nil
}

func RunSpecfem(app *specfemv1.SpecfemApp) error {
	if err := CreateBaseAndMesherImages(app); err != nil {
		return err
	}

	if _, err := CreateYamlResource(app, yamlMesherScriptCM, "all"); err != nil {
		return err
	}

	if _, err := CreateYamlResource(app, yamlPVC, "config"); err != nil {
		return err
	}
	
	if err := RunMesherSolver(app, "mesher"); err != nil {
		return err
	}

	if err := CreateSolverImage(app); err != nil {
		return err
	}

	if err := CheckImageTag("specfem:solver", "mesher"); err != nil {
		return err
	}

	if _, err := CreateYamlResource(app, yamlSolverScriptCM, "all"); err != nil {
		return err
	}

	if err := RunMesherSolver(app, "solver"); err != nil {
		return err
	}

	if err := RunSaveSolverOutput(app); err != nil {
		return err
	}

	log.Println("All done!")

	return nil
}
