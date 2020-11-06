package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	errs "github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	specfemv1 "github.com/openshift-psap/specfem-client-api/pkg/apis/specfem/v1alpha1"
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

	if ! delete_mode {
		if err := CheckImageTag("specfem:base", "all"); err == nil {
			log.Printf("Found base image, don't build it.")
			goto CheckMesher
		}
	}

	if err := CreateAndWaitYamlBuildConfig(app, yamlBaseImageBuildConfig, "all"); err != nil {
		return err
	}

	if err := CheckImageTag("specfem:base", "all"); err != nil {
		return err
	}

CheckMesher:
	mesher_image := fmt.Sprintf("specfem:mesher-%dproc-%dnex",
		app.Spec.Exec.Nproc, app.Spec.Specfem.Nex)

	if ! delete_mode {
		if err := CheckImageTag(mesher_image, "cache"); err == nil {
			log.Println("Found mesher image, don't recreate it.")
			return nil
		}
	}

	if err := CreateAndWaitYamlBuildConfig(app, yamlMesherImageBuildConfig, "mesher"); err != nil {
		return err
	}

	if err := CheckImageTag(mesher_image, "cache"); err != nil {
		return err
	}

	return nil
}

func CreateSolverImage(app *specfemv1.SpecfemApp, solver_image string) error {
	if err:= CreateAndWaitYamlBuildConfig(app, newMesher2SolverHelperBuildConfig, "all"); err != nil {
		return err
	}

	builderNodeName, err := GetOrSetWorkerNodeTag("buildah.specfem.build")

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
		if err := CheckImageTag(solver_image, "cache"); err == nil {
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
		var logs *string // ignore logs if everything's ok
		if err := WaitWithPodLogs("", podName, "", &logs); err != nil {
			fmt.Println(logs)
			return err
		}
	}

	if err := CheckImageTag(solver_image, "cache"); err != nil {
		return err
	}

	return nil
}

func HasMpiWorkerPods(app *specfemv1.SpecfemApp, stage string) (int, error) {
	pods, err := client.ClientSet.CoreV1().Pods(app.ObjectMeta.Namespace).List(context.TODO(),
		metav1.ListOptions{LabelSelector: "mpi_role_type=worker,mpi_job_name=mpi-"+stage})

	if err != nil {
		return 0, err
	}

	return len(pods.Items), nil
}

func RunMpiJob(app *specfemv1.SpecfemApp, stage string) error {
	if delete_mode {
		goto skip_pod_check
	}
	for {
		pod_cnt, err := HasMpiWorkerPods(app, stage)
		if err != nil {
			return err
		}

		fmt.Printf("found %d worker pods from previous mpijob/mpi-%s ...\n", pod_cnt, stage)
		if pod_cnt == 0 {
			break
		}

		time.Sleep(2 * time.Second)
		// loop
	}

skip_pod_check:
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

	if delete_mode {
		CleanupJobPods(app, yamlSaveSolverOutputJob)
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
	solver_image := fmt.Sprintf("specfem:solver-%dproc-%dnex",
		app.Spec.Exec.Nproc, app.Spec.Specfem.Nex)


	if ! delete_mode {
		if err := CheckImageTag(solver_image, "cache"); err == nil {
			log.Printf("Found solver image, skip mesher.")
			goto RunSolver
		}
	}

	if err := CreateBaseAndMesherImages(app); err != nil {
		return err
	}

	if _, err := CreateYamlResource(app, yamlMesherScriptCM, "all"); err != nil {
		return err
	}

	if _, err := CreateYamlResource(app, yamlPVC, "cache"); err != nil {
		return err
	}

	if err := RunMesherSolver(app, "mesher"); err != nil {
		return err
	}

	if err := CreateSolverImage(app, solver_image); err != nil {
		return err
	}

	if err := CheckImageTag(solver_image, "cache"); err != nil {
		return err
	}

RunSolver:
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
