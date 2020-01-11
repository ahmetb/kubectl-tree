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
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/dynamic"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // combined authprovider import
	"k8s.io/klog"
)

var cf *genericclioptions.ConfigFlags

// This variable is populated by goreleaser
var version string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:          "kubectl tree KIND NAME",
	SilenceUsage: true, // for when RunE returns an error
	Short:        "Show sub-resources of the Kubernetes object",
	Example: "  kubectl tree deployment my-app\n" +
		"  kubectl tree kservice.v1.serving.knative.dev my-app", // TODO add more examples about disambiguation etc
	Args:    cobra.MinimumNArgs(2),
	RunE:    run,
	Version: versionString(),
}

// versionString returns the version prefixed by 'v'
// or an empty string if no version has been populated by goreleaser.
// In this case, the --version flag will not be added by cobra.
func versionString() string {
	if len(version) == 0 {
		return ""
	}
	return "v" + version
}

func run(_ *cobra.Command, args []string) error {
	restConfig, err := cf.ToRESTConfig()
	if err != nil {
		return err
	}
	restConfig.QPS = 1000
	restConfig.Burst = 1000
	dyn, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("failed to construct dynamic client: %w", err)
	}
	dc, err := cf.ToDiscoveryClient()
	if err != nil {
		return err
	}

	apis, err := findAPIs(dc)
	if err != nil {
		return err
	}
	klog.V(3).Info("completed querying APIs list")

	kind, name := args[0], args[1]
	klog.V(3).Infof("parsed kind=%v name=%v", kind, name)

	var api apiResource
	if k, ok := overrideType(kind, apis); ok {
		klog.V(2).Infof("kind=%s override found: %s", kind, k.GroupVersionResource())
		api = k
	} else {
		apiResults := apis.lookup(kind)
		klog.V(5).Infof("kind matches=%v", apiResults)
		if len(apiResults) == 0 {
			return fmt.Errorf("could not find api kind %q", kind)
		} else if len(apiResults) > 1 {
			names := make([]string, 0, len(apiResults))
			for _, a := range apiResults {
				names = append(names, fullAPIName(a))
			}
			return fmt.Errorf("ambiguous kind %q. use one of these as the KIND disambiguate: [%s]", kind,
				strings.Join(names, ", "))
		}
		api = apiResults[0]
	}

	ns := *cf.Namespace
	if ns == "" {
		clientConfig := cf.ToRawKubeConfigLoader()
		defaultNamespace, _, err := clientConfig.Namespace()
		if err != nil {
			defaultNamespace = "default"
		}
		ns = defaultNamespace
	}

	obj, err := dyn.Resource(api.GroupVersionResource()).Namespace(ns).Get(name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get %s/%s: %w", kind, name, err)
	}

	klog.V(5).Infof("target parent object: %#v", obj)

	klog.V(2).Infof("querying all api objects")
	apiObjects, err := getAllResources(dyn, apis.resources())
	if err != nil {
		return fmt.Errorf("error while querying api objects: %w", err)
	}
	klog.V(2).Infof("found total %d api objects", len(apiObjects))

	objs := newObjectDirectory(apiObjects)
	if len(objs.ownership[obj.GetUID()]) == 0 {
		fmt.Println("No resources are owned by this object through ownerReferences.")
		return nil
	}
	treeView(os.Stderr, objs, *obj)
	klog.V(2).Infof("done printing tree view")
	return nil
}

func init() {
	klog.InitFlags(nil)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	// hide all glog flags except for -v
	flag.CommandLine.VisitAll(func(f *flag.Flag) {
		if f.Name != "v" {
			pflag.Lookup(f.Name).Hidden = true
		}
	})

	cf = genericclioptions.NewConfigFlags(true)
	cf.AddFlags(rootCmd.Flags())
	if err := flag.Set("logtostderr", "true"); err != nil {
		fmt.Fprintf(os.Stderr, "failed to set logtostderr flag: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	defer klog.Flush()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
