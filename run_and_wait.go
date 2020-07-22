
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	buildv1client "github.com/openshift/client-go/build/clientset/versioned/typed/build/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	buildv1 "github.com/openshift/api/build/v1"
	buildhelpers "github.com/openshift/oc/pkg/helpers/build"

	specfemv1 "gitlab.com/kpouget_psap/specfem-operator/pkg/apis/specfem/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/apimachinery/pkg/fields"
)

func WaitPodRunning(podName string) error {
	c := client.ClientSet.CoreV1().Pods(NAMESPACE)
	pods, err := c.List(context.TODO(),
		metav1.ListOptions{FieldSelector: fields.Set{"metadata.name": podName}.AsSelector().String()})
	if err != nil {
		return err
	}

	isRunning := func(pod *corev1.Pod) bool {
		return pod.Status.Phase == corev1.PodRunning
	}
	isOK := func(pod *corev1.Pod) bool {
		return pod.Status.Phase == corev1.PodSucceeded
	}
	isFailed := func(pod *corev1.Pod) bool {
		if pod.Status.Phase == corev1.PodFailed {
			return true
		}
		if pod.Status.ContainerStatuses != nil {
			for _, containerStatus := range pod.Status.ContainerStatuses {
				if containerStatus.State.Waiting != nil {
					if containerStatus.State.Waiting.Reason == "CrashLoopBackOff" {
						return true
					}
				}
			}
		}
		return false
	}

	printStatus := func(pod *corev1.Pod) {
		log.Printf("Status of pod/%s: %s", podName, pod.Status.Phase)
		if pod.Status.ContainerStatuses != nil {
			for _, containerStatus := range pod.Status.ContainerStatuses {
				if containerStatus.State.Waiting != nil {
					log.Printf("  - status of container %s: %s",
						containerStatus.Name, containerStatus.State.Waiting.Reason)
					if containerStatus.State.Waiting.Message != "" {
						log.Printf("    --> %s", containerStatus.State.Waiting.Message)
					}
				}
			}
		}	
	}
	
	
	for _, pod := range pods.Items {
		podName = pod.ObjectMeta.Name
		fmt.Printf("found pod/%s --> %s\n", podName, pod.Status.Phase)
		printStatus(&pod)

		if isOK(&pod) || isRunning(&pod) {
			return nil
		}
		if isFailed(&pod) {
			return fmt.Errorf("pod/%s failed to run ...", podName)
		}

		
		break
	}

	log.Printf("Wait for the completion of pod/%s ...", podName)

	rv := pods.ResourceVersion
	w, err := c.Watch(context.TODO(),
		metav1.ListOptions{
			FieldSelector: fields.Set{"metadata.name": podName}.AsSelector().String(),
			ResourceVersion: rv})

	if err != nil {
		return err
	}
	defer w.Stop()
	for {
		val, ok := <-w.ResultChan()
		if !ok {
			break // reget and re-watch
		}

		if pod, ok := val.Object.(*corev1.Pod); ok {
			printStatus(pod)

			if isOK(pod) || isRunning(pod) {
				return nil
			}
			if isFailed(pod) {
				return fmt.Errorf("pod/%s status is %q", podName, pod.Status.Phase)
			}

		}
	}

	return nil
}

func WaitWithPodLogs(parentName string, podName string, search string, p_logs **string) error {
	var podLogs io.ReadCloser
	var err error
	var req *rest.Request

	var parent string = ""
	if parentName != "" {
		parent = fmt.Sprintf("(from %s)", parentName)
	}

	err = WaitPodRunning(podName)
	if err != nil {
		return err
	}
	
	for {
		req = client.ClientSet.CoreV1().Pods(NAMESPACE).GetLogs(podName, &corev1.PodLogOptions{Follow: true})

		if req == nil {
			time.Sleep(2 * time.Second)
			fmt.Printf("failed to get the log stream for pod/%s %s...\n", podName, parent)
			continue
		}
		
		podLogs, err = req.Stream(context.TODO())

		if err != nil {
			time.Sleep(2 * time.Second)
			fmt.Printf("failed to open the log stream for pod/%s %s...\n", podName, parent)
			continue
		}
		
		break
	}
	defer podLogs.Close()

	logs := ""
	b := make([]byte, 8)
	for {
		n, err := podLogs.Read(b)
		str := string(b[:n])
		logs += str
		if p_logs == nil {
			fmt.Print(str)
		}
		if search != "" && strings.Contains(logs, search) {
			if p_logs == nil {
				fmt.Println()
			}
			fmt.Printf("found '%s' in the logs...\n", search)

			break
		}

		if err == io.EOF {
			if p_logs == nil {
				fmt.Println()
			}
			fmt.Printf("found EOF in the logs...\n")
			break
		}
	}

	if p_logs != nil {
		*p_logs = &logs
	}

	// check final status
	return WaitPodRunning(podName)

}

func WaitWithJobLogs(jobName string, search string, logs **string) error {
	var podName string
	
	for {
		pods, _ := client.ClientSet.CoreV1().Pods(NAMESPACE).List(context.TODO(),
			metav1.ListOptions{LabelSelector: "job-name="+jobName})
		podName = ""
		for _, pod := range pods.Items {
			podName = pod.ObjectMeta.Name

			fmt.Printf("found pod/%s for job/%s\n", podName, jobName)
			break
		}

		if podName != "" {
			break
		}
		
		fmt.Printf("failed to find pod for job/%s [len: %d] ...\n", jobName, len(pods.Items))
		time.Sleep(2 * time.Second)
	}
	
	return WaitWithPodLogs("job/"+jobName, podName, search, logs)
}


/* from github.com/openshift/oc/pkg/cli/startbuild/ */
// WaitForBuildComplete waits for a build identified by the name to complete
func WaitForBuildComplete(bcName string) error {
	buildClient, err := buildv1client.NewForConfig(client.Config)
	if err != nil {
        return err
    }

	buildConfigObj, err := buildClient.BuildConfigs(NAMESPACE).Get(context.TODO(), bcName, metav1.GetOptions{})
	if err != nil {
		log.Fatalf("Failed to get build for %s: %v", bcName, err)
		return err
	}
	lastVersion := int(buildConfigObj.Status.LastVersion)
	if lastVersion == 0 {
		log.Println("Build config still 0, forcing it to 1.")
		lastVersion = 1

	}
	buildName := buildhelpers.BuildNameForConfigVersion(bcName, lastVersion)

	c := buildClient.Builds(NAMESPACE)

	isOK := func(b *buildv1.Build) bool {
		return b.Status.Phase == buildv1.BuildPhaseComplete
	}
	isFailed := func(b *buildv1.Build) bool {
		return b.Status.Phase == buildv1.BuildPhaseFailed ||
			b.Status.Phase == buildv1.BuildPhaseCancelled ||
			b.Status.Phase == buildv1.BuildPhaseError
	}

	for {
		list, err := c.List(context.TODO(), metav1.ListOptions{FieldSelector: fields.Set{"metadata.name": buildName}.AsSelector().String()})
		if err != nil {
			return err
		}

		for i := range list.Items {
			log.Printf("Build status of build/%s status: %q", list.Items[i].Name, list.Items[i].Status.Phase)

			if buildName == list.Items[i].Name && isOK(&list.Items[i]) {
				return nil
			}
			if buildName != list.Items[i].Name || isFailed(&list.Items[i]) {
				return fmt.Errorf("the build %s/%s status is %q", list.Items[i].Namespace, list.Items[i].Name, list.Items[i].Status.Phase)
			}
		}

		rv := list.ResourceVersion

		log.Printf("Wait for the completion of build/%s ...", buildName)
		w, err := c.Watch(context.TODO(), metav1.ListOptions{FieldSelector: fields.Set{"metadata.name": buildName}.AsSelector().String(), ResourceVersion: rv})
		if err != nil {
			return err
		}
		defer w.Stop()

		for {
			val, ok := <-w.ResultChan()
			if !ok {
				break // reget and re-watch
			}

			if e, ok := val.Object.(*buildv1.Build); ok {
				log.Printf("Build status of build/%s status: %q", buildName, e.Status.Phase)

				if buildName == e.Name && isOK(e) {
					return nil
				}
				if buildName != e.Name || isFailed(e) {
					return fmt.Errorf("the build build/%s status is %q", buildName, e.Status.Phase)
				}
			}
		}
	}
}

func RunSaveSolverOutput(app *specfemv1.SpecfemApp) error {
	jobName, err := CreateResource(app, newSaveSolverOutputJob, "solver")
	if err != nil {
		return err
	}

	var logs *string = nil

	err = WaitWithJobLogs(jobName, "", &logs)
	if err != nil {
		return err
	}
	if logs == nil {
		return fmt.Errorf("Failed to get logs for job/%s", jobName)
	}

	BUILDLOG_FILENAME := "/tmp/specfem.solver.log"
	f, err := os.Create(BUILDLOG_FILENAME)
	if err != nil {
		return err
	}
	defer f.Close()

	f.WriteString(*logs)

	log.Printf("Saved solver logs into '%s'", BUILDLOG_FILENAME)
	
	return nil

}
