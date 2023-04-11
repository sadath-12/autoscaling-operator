package utils

import (
	// custom "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"

	"github.com/go-logr/logr"
	"github.com/prometheus-operator/prometheus-operator/pkg/client/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var Log logr.Logger

func generateK8sClient() *kubernetes.Clientset {
	config, err := generateK8sConfig()
	if err != nil {

		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	return clientset

}

// generateK8sConfig will load the kube config file
func generateK8sConfig() (*rest.Config, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	// if you want to change the loading rules (which files in which order), you can do so here
	configOverrides := &clientcmd.ConfigOverrides{}
	// if you want to change override values or bind them to flags, there are methods to help you
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	return kubeConfig.ClientConfig()
}

func generatePromClient() (*versioned.Clientset, error) {
	config, err := generateK8sConfig()
	if err != nil {

		panic(err.Error())
	}
	promClient, err := versioned.NewForConfig(config)
	if err != nil {

		panic(err.Error())
	}

	return promClient, nil
}
