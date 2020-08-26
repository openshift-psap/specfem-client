package main

import (
	"context"
	"log"
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	kubeflow "github.com/kubeflow/mpi-operator/pkg/apis/kubeflow/v1"
)

var client MyClient

type MyClient struct {
	ClientSet *kubernetes.Clientset
	ClientSetDyn dynamic.Interface
	Config *rest.Config
}

func InitClient() error {
	// use the current context in kubeconfig
	var err error
	kubeflow.AddToScheme(scheme.Scheme)

	client.Config, err = clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		log.Fatalln("failed to build k8s config")
		return err
	}

	client.ClientSet, err = kubernetes.NewForConfig(client.Config)
    if err != nil {
		log.Fatalln("failed to build k8s client")
		return err
    }

	client.ClientSetDyn, err = dynamic.NewForConfig(client.Config)
	if err != nil {
		return err
	}

	
	return nil
}

func (c MyClient) Create(gvr schema.GroupVersionResource,  unstructuredObj *unstructured.Unstructured) error {
	var err error

	if gvr.Resource != "storageclasses" && gvr.Resource != "persistentvolumes" {
		_, err = client.ClientSetDyn.Resource(gvr).Namespace(NAMESPACE).Create(context.TODO(), unstructuredObj, metav1.CreateOptions{})
	} else {
		_, err = client.ClientSetDyn.Resource(gvr).Create(context.TODO(), unstructuredObj, metav1.CreateOptions{})
	}
	return err
}

func (c MyClient) Delete(gvr schema.GroupVersionResource, objName string) error {

	return client.ClientSetDyn.Resource(gvr).Namespace(NAMESPACE).Delete(context.TODO(), objName, metav1.DeleteOptions{})

}

func (c MyClient) Get(gvr schema.GroupVersionResource, objName string) (*unstructured.Unstructured, error) {
		return client.ClientSetDyn.Resource(gvr).Namespace(NAMESPACE).Get(context.TODO(), objName, metav1.GetOptions{})
}
