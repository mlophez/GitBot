package app

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func getClientSet(k string) *kubernetes.Clientset {
	config, err := rest.InClusterConfig()
	//config, err := clientcmd.BuildConfigFromFlags("", k)
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	return clientset
}
