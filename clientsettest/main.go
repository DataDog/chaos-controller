package main

import (
	"context"
	"fmt"
	"log"

	clientset "github.com/DataDog/chaos-controller/clientset/v1beta1"
	"github.com/pkg/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// Build Kubernetes configuration using BuildKubeConfig()
	config, err := BuildKubeConfig()
	if err != nil {
		log.Fatalf("Failed to build kubeconfig: %v", err)
	}

	// Create a new Clientset for the given config
	cs, err := clientset.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create clientset: %v", err)
	}

	// Specify the namespace to list disruptions from
	namespace := "chaos-demo"

	// List disruptions in the specified namespace
	disruptions, err := cs.Chaos().Disruptions(namespace).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		log.Fatalf("Failed to list disruptions: %v", err)
	}

	// Immediately iterate over the disruptions.
	for _, d := range disruptions.Items {
		fmt.Printf("- %s\n", d.Name)
	}
}

func BuildKubeConfig() (*rest.Config, error) {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		if errors.Is(err, rest.ErrNotInCluster) {
			// Specifying "colima" as the current context for local development
			currentContext := "colima"

			restConfig, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
				clientcmd.NewDefaultClientConfigLoadingRules(),
				&clientcmd.ConfigOverrides{
					CurrentContext: currentContext,
				}).ClientConfig()
			if err != nil {
				return nil, fmt.Errorf("unable to build out-of-cluster configuration: %w", err)
			}
		} else {
			return nil, fmt.Errorf("unable to build in-cluster configuration: %w", err)
		}
	}

	return restConfig, nil
}
