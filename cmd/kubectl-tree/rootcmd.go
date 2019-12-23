/*
Copyright © 2019 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"fmt"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"os"
	"strings"
)

var cf *genericclioptions.ConfigFlags

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:          "kubectl tree KIND NAME",
	SilenceUsage: true, // for when RunE returns an error
	Short:        "Show sub-resources of the Kubernetes object",
	Example: "  kubectl tree deployment my-app\n" +
		"  kubectl tree kservice.v1.serving.knative.dev my-app", // TODO add more examples about disambiguation etc
	Args: cobra.MinimumNArgs(2),
	RunE: run,
}

func run(cmd *cobra.Command, args []string) error {
	restConfig, err := cf.ToRESTConfig()
	if err != nil {
		return err
	}
	restConfig.QPS = 1000
	restConfig.Burst = 1000
	dyn, err := dynamicClient(restConfig)
	if err != nil {
		return err
	}
	dc, err := cf.ToDiscoveryClient()
	if err != nil {
		return err
	}
	apis, err := buildAPILookup(dc)
	if err != nil {
		return err
	}

	kind, name := args[0], args[1]
	apiRes := apis.lookup(kind)
	if len(apiRes) == 0 {
		return fmt.Errorf("could not find api kind %q", kind)
	} else if len(apiRes) > 1 {
		names := make([]string, 0, len(apiRes))
		for _, a := range apiRes {
			names = append(names, fullAPIName(a))
		}
		return fmt.Errorf("ambiguous kind %q. use one of these as the KIND disambiguate: [%s]", kind,
			strings.Join(names, ", "))
	}

	ns  := *cf.Namespace
	if ns ==  ""{
		ns = "default" // TODO(ahmetb): how to get current-namespace from kubeconfig?
	}
	obj, err := dyn.Resource(apiRes[0].GroupVersionResource()).Namespace(ns).Get(name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get %s/%s: %w", kind, name, err)
	}

	apiObjects, err := getAllResources(dyn, apis.resources())
	if err != nil {
		return fmt.Errorf("error while querying api objects: %w", err)
	}

	objs := newObjectDirectory(apiObjects)
	if len(objs.ownership[obj.GetUID()]) == 0 {
		fmt.Println("No resources are owned by the specified object through ownerReferences.")
		return nil
	}
	treeView(os.Stderr, objs, *obj)
	return nil
}
