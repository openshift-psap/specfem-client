package main

import (
	"bytes"
	"context"
	"fmt"
	"text/template"
	"log"
	"math"
	"strings"

	errs "github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	specfemv1 "github.com/openshift-psap/specfem-client-api/pkg/apis/specfem/v1alpha1"
	"github.com/openshift-psap/specfem-client/yamlutil"


	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"

)

type ResourceCreator func(app *specfemv1.SpecfemApp)(schema.GroupVersionResource, string, runtime.Object)
type YamlResourceTmpl func(app *specfemv1.SpecfemApp) *TemplateCfg
type YamlResourceSpec func() (string, YamlResourceTmpl)

type TemplateBase struct {
	App *specfemv1.SpecfemApp
	Cfg *TemplateCfg
	Manifests *map[string]string
}

func NoTemplateCfg(app *specfemv1.SpecfemApp) (cfg *TemplateCfg) {
	return
}

func applyTemplate(yamlSpec *[]byte, templateFct YamlResourceTmpl, app *specfemv1.SpecfemApp) error {
	tmpl_data := TemplateBase{
		App: app,
		Cfg: templateFct(app),
		Manifests: &manifests,
	}
	fmap := template.FuncMap{
        "indent": func(len int, txt string) string {
			return strings.ReplaceAll(txt, "\n", "\n"+strings.Repeat(" ", len))
		},
		"isqrt": func(n int32) int32 {
			return int32(math.Sqrt(float64(n)))
		},
		"escape": func(src, dst string, txt string) string {
			return strings.ReplaceAll(txt, src, dst)
		},
    }

	tmpl := template.Must(template.New("runtime").Funcs(fmap).Parse(string(*yamlSpec)))

	var buff bytes.Buffer
	if err := tmpl.Execute(&buff, tmpl_data); err != nil {
		return errs.Wrap(err, "Cannot templatize spec for resource info injection, check manifest")
	}
	*yamlSpec = buff.Bytes()

	return nil
}

func createFromYamlManifest(yamlManifest string, templateFct YamlResourceTmpl, app *specfemv1.SpecfemApp) (schema.GroupVersionResource, *unstructured.Unstructured, error) {
	namespace := app.ObjectMeta.Namespace
	scanner := yamlutil.NewYAMLScanner([]byte(manifests[yamlManifest]))

	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return schema.GroupVersionResource{}, nil, errs.Wrap(err, "Failed to scan manifest ")
		}
		return schema.GroupVersionResource{}, nil, fmt.Errorf("YAML empty document")
	}

	yamlSpec := scanner.Bytes()

	if scanner.Scan() {
		return schema.GroupVersionResource{}, nil, fmt.Errorf("Cannot have multiple YAML in one file")
	}

	if err := applyTemplate(&yamlSpec, templateFct, app); err != nil {
		return schema.GroupVersionResource{}, nil, errs.Wrap(err, "Cannot inject runtime information")
	}
	// apply twice as file may be inject in the file run
	if err := applyTemplate(&yamlSpec, templateFct, app); err != nil {
		return schema.GroupVersionResource{}, nil, errs.Wrap(err, "Cannot inject runtime information")
	}

	obj := &unstructured.Unstructured{}
	jsonSpec, err := yaml.YAMLToJSON(yamlSpec)
	if err != nil {
		return schema.GroupVersionResource{}, nil, errs.Wrap(err, "Could not convert yaml file to json "+string(yamlSpec))
	}

	if err = obj.UnmarshalJSON(jsonSpec); err != nil {
		return schema.GroupVersionResource{}, nil, errs.Wrap(err, "Cannot unmarshall json spec, check your manifests")
	}

	if obj.GetNamespace() == "" {
		obj.SetNamespace(namespace)
	}

	resType := schema.GroupVersionResource{
		Version: obj.GroupVersionKind().Version,
		Group: obj.GroupVersionKind().Group,
		Resource: strings.ToLower(obj.GetKind()) + "s",
	}

	return resType, obj, nil
}

func CleanupJobPods(app *specfemv1.SpecfemApp, yamlSpecFct YamlResourceSpec) error {
	yamlManifest, templateFct := yamlSpecFct()
	_, obj, err := createFromYamlManifest(yamlManifest, templateFct, app)
	if err != nil {
		return errs.Wrap(err, fmt.Sprintf("Cannot create the YAML resource from Yaml file '%+v'", yamlManifest))
	}
	jobName := obj.GetName()
	pods, err := client.ClientSet.CoreV1().Pods(NAMESPACE).List(context.TODO(),
		metav1.ListOptions{LabelSelector: "job-name="+jobName})
	if err != nil {
		return errs.Wrap(err, fmt.Sprintf("Cannot list the pods associated with job/%s", jobName))
	}
	podResType := schema.GroupVersionResource{
		Version: "v1",
		Resource: "pods",
	}

	for _, pod := range pods.Items {
		podName := pod.ObjectMeta.Name
		fmt.Printf("delete job/%s --> pod/%s\n", jobName, podName)
		err = client.Delete(podResType, podName)
		if err != nil {
			return errs.Wrap(err, fmt.Sprintf("Cannot delete pod/%s associated with job/%s", podName, jobName))
		}
	}
	return nil
}

func CreateYamlResource(app *specfemv1.SpecfemApp, yamlSpecFct YamlResourceSpec, stage string) (string, error) {
	yamlManifest, templateFct := yamlSpecFct()

	resType, obj, err := createFromYamlManifest(yamlManifest, templateFct, app)
	if err != nil {
		return "", errs.Wrap(err, fmt.Sprintf("Cannot create the YAML resource from Yaml file '%+v'", yamlManifest))
	}

	objName := obj.GetName()

	return doCreateResource(resType, obj, objName, stage)
}

func CreateResource(app *specfemv1.SpecfemApp,
	creatorFunction ResourceCreator, stage string) (string, error) {
	resType, objName, obj:= creatorFunction(app)

	if _, ok := to_delete[stage]; !ok {
		msg := fmt.Sprintf("Invalid stage '%v' for object %s/%s | %q", stage, resType, objName, to_delete)
		if delete_mode {
			return "", fmt.Errorf(msg)
		} else {
			log.Printf(msg)
		}
	}

	mapObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil{
		return "", err
	}

	unstructuredObj := &unstructured.Unstructured{}
	unstructuredObj.SetUnstructuredContent(mapObj)

	return doCreateResource(resType, unstructuredObj, objName, stage)
}

func doCreateResource(resType schema.GroupVersionResource, obj *unstructured.Unstructured, objName string, stage string) (string, error) {

	if _, ok := to_delete[stage]; !ok {
		msg := fmt.Sprintf("Invalid stage '%v' for object %s/%s | %q", stage, resType, objName, to_delete)
		if delete_mode {
			return "", fmt.Errorf(msg)
		} else {
			log.Printf(msg)
		}
	}
	var objDesc string
	if  obj.GetKind() != "" {
		objDesc = fmt.Sprintf("%s/%s", obj.GetKind(), objName)
	} else {
		res := resType.GroupResource().Resource
		objDesc = fmt.Sprintf("@X@ %s/%s", res[:len(res)-1], objName)
	}
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
		return "", nil
	}

	log.Printf("Create %s", objDesc)
	err = client.Create(resType, obj)
	if err != nil && !errors.IsAlreadyExists(err) {
		log.Printf("Failed to create %s: %+v", objDesc, err)
		return "", err
	}

	return objName, nil
}

func CreateAndWaitYamlBuildConfig(app *specfemv1.SpecfemApp, yamlSpecFct YamlResourceSpec, stage string) error{
	bcName, err := CreateYamlResource(app, yamlSpecFct, stage)
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

func CheckImageTag(imagetagName string, stage string) error {
	var err error = nil
	var gvr = imagestreamtagResource
	var objDesc = "imagestreamtag/"+imagetagName

	if delete_mode {
		if to_delete[stage] {
			log.Printf("Delete %s | %s", objDesc, stage)
			err = client.Delete(gvr, imagetagName)
			if err != nil && !errors.IsNotFound(err) {
				log.Printf("Could not delete %s: %+v", objDesc, err)
			}
			err = nil
		}
	} else {
		log.Printf("Checking %s | %s", objDesc, stage)
		_, err = client.Get(gvr, imagetagName)
	}

	return err
}
