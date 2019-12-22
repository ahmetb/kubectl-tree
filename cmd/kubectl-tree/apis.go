package main

import (
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"strings"
)

type resourceMap map[string][]schema.GroupVersionResource // names to apis binding

func (rm resourceMap) lookup(s string) []schema.GroupVersionResource {
	return rm[strings.ToLower(s)]
}

func fullAPIName(a schema.GroupVersionResource) string {
	return strings.Join([]string{a.Resource, a.Version, a.Group}, ".")
}

func buildAPILookup(client discovery.DiscoveryInterface) (resourceMap, error) {
	resList, err := client.ServerPreferredResources()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch api groups from kubernetes: %w", err)
	}

	nameMap := make(resourceMap)

	for _, group := range resList {
		gv, err := schema.ParseGroupVersion(group.GroupVersion)
		if err != nil {
			return nil, fmt.Errorf("%q cannot be parsed into groupversion: %w", group.GroupVersion, err)
		}

		for _, apiRes := range group.APIResources {
			if len(apiRes.Verbs) == 0 {
				continue
			}

			for _, name := range apiNames(apiRes, gv) {
				nameMap[name] = append(nameMap[name], schema.GroupVersionResource{
					Group:    gv.Group,
					Version:  gv.Version,
					Resource: apiRes.Name,
				})
			}
		}
	}
	return nameMap, nil
}

// return all names that could refer to this APIResource
func apiNames(a metav1.APIResource, gv schema.GroupVersion) []string {
	var out []string
	singularName := a.SingularName
	pluralName := a.Name
	shortNames := a.ShortNames
	names := append([]string{singularName, pluralName}, shortNames...)
	for _, n := range names {
		fmtBare := n                                                            // e.g. deployment
		fmtWithGroup := fmt.Sprintf("%s.%s", n, gv.Group)                       // e.g. deployment.apps
		fmtWithGroupVersion := fmt.Sprintf("%s.%s.%s", n, gv.Version, gv.Group) // e.g. deployment.v1.apps

		out = append(out,
			fmtBare, fmtWithGroup, fmtWithGroupVersion)
	}
	return out
}
