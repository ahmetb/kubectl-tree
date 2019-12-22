package main

import (
	"fmt"
	"k8s.io/client-go/dynamic"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
)

func dynamicClient(config *rest.Config) (dynamic.Interface, error) {
	config.QPS = 1000 // TODO: use .Burst as well?
	c, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to construct dynamic client: %w", err)
	}
	return c, nil
}
