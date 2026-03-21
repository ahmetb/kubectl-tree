package main

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/klog"
)

type apiResource struct {
	r  metav1.APIResource
	gv schema.GroupVersion
}

func (a apiResource) GroupVersionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    a.gv.Group,
		Version:  a.gv.Version,
		Resource: a.r.Name,
	}
}

type resourceNameLookup map[string][]apiResource

type resourceMap struct {
	list []apiResource
	m    resourceNameLookup
}

func (rm *resourceMap) lookup(s string) []apiResource {
	return rm.m[strings.ToLower(s)]
}

func (rm *resourceMap) resources() []apiResource { return rm.list }

func fullAPIName(a apiResource) string {
	sgv := a.GroupVersionResource()
	return strings.Join([]string{sgv.Resource, sgv.Version, sgv.Group}, ".")
}

func matchAny(patterns []string, match func(string) (bool, error)) bool {
	if len(patterns) == 0 {
		return true
	}

	hasPositive := false
	matchedPositive := false

	for _, p := range patterns {
		negated := strings.HasPrefix(p, "!")
		if negated {
			p = p[1:]
		} else {
			hasPositive = true
		}

		ok, err := match(p)
		if err != nil {
			klog.V(1).Infof("%s is an invalid pattern: %v", p, err)
			continue
		}
		if negated && ok {
			return false
		}
		if !negated && ok {
			matchedPositive = true
		}
	}

	return !hasPositive || matchedPositive
}

func matchGroups(patterns []string, g string) bool {
	return matchAny(patterns, func(pattern string) (bool, error) {
		if pattern == "*" {
			return true, nil
		}
		if pattern == "core" || pattern == "" {
			return g == "", nil
		}

		return filepath.Match(pattern, g)
	})
}

func matchResources(patterns []string, apiRes metav1.APIResource) bool {
	if apiRes.SingularName == "" {
		apiRes.SingularName = strings.ToLower(apiRes.Kind)
	}
	names := []string{apiRes.Name, apiRes.SingularName, apiRes.Kind}
	names = append(names, apiRes.ShortNames...)

	return matchAny(patterns, func(pattern string) (bool, error) {
		for _, name := range names {
			ok, err := filepath.Match(pattern, name)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}

		return false, nil
	})
}

func findAPIs(client discovery.DiscoveryInterface, apiGroups, resources []string) (*resourceMap, error) {
	start := time.Now()
	resList, err := client.ServerPreferredResources()
	if err != nil {
		klog.V(1).Infof("failed to fetch api groups from kubernetes: %v\n", err)
	}
	klog.V(2).Infof("queried api discovery in %v", time.Since(start))
	klog.V(3).Infof("found %d items (groups) in server-preferred APIResourceList", len(resList))

	rm := &resourceMap{
		m: make(resourceNameLookup),
	}
	for _, group := range resList {
		gv, err := schema.ParseGroupVersion(group.GroupVersion)
		if err != nil {
			return nil, fmt.Errorf("%q cannot be parsed into groupversion: %w", group.GroupVersion, err)
		}

		if !matchGroups(apiGroups, gv.Group) {
			klog.V(5).Infof("ignoring group %s/%s (%d apis)", group.GroupVersion, group.APIVersion, len(group.APIResources))
			continue
		}
		klog.V(5).Infof("iterating over group %s/%s (%d apis)", group.GroupVersion, group.APIVersion, len(group.APIResources))

		for _, apiRes := range group.APIResources {
			klog.V(5).Infof("  api=%s namespaced=%v", apiRes.Name, apiRes.Namespaced)
			if !contains(apiRes.Verbs, "list") {
				klog.V(4).Infof("    api (%s) doesn't have required verb, skipping: %v", apiRes.Name, apiRes.Verbs)
				continue
			}
			// NOTE: if a intermediate owner is excluded that will break the chain, even if the leaf is included
			// for example --resources=deployments,pods will return nothing because replicasets are not included
			if !matchResources(resources, apiRes) {
				klog.V(5).Infof("    api (%s) doesn't match any resource pattern, skipping: %v", apiRes.Name, apiRes.Verbs)
				continue
			}
			v := apiResource{
				gv: gv,
				r:  apiRes,
			}
			names := apiNames(apiRes, gv)
			klog.V(6).Infof("names: %s", strings.Join(names, ", "))
			for _, name := range names {
				rm.m[name] = append(rm.m[name], v)
			}
			rm.list = append(rm.list, v)
		}
	}
	klog.V(5).Infof("  found %d apis", len(rm.m))
	return rm, nil
}

func contains(v []string, s string) bool {
	return slices.Contains(v, s)
}

// return all names that could refer to this APIResource
func apiNames(a metav1.APIResource, gv schema.GroupVersion) []string {
	var out []string
	singularName := a.SingularName
	if singularName == "" {
		// TODO(ahmetb): sometimes SingularName is empty (e.g. Deployment), use lowercase Kind as fallback - investigate why
		singularName = strings.ToLower(a.Kind)
	}
	names := []string{singularName}

	pluralName := a.Name
	if singularName != pluralName {
		names = append(names, pluralName)
	}

	shortNames := a.ShortNames
	names = append(names, shortNames...)

	for _, n := range names {
		fmtBare := n                                                                // e.g. deployment
		fmtWithGroup := strings.Join([]string{n, gv.Group}, ".")                    // e.g. deployment.apps
		fmtWithGroupVersion := strings.Join([]string{n, gv.Version, gv.Group}, ".") // e.g. deployment.v1.apps

		out = append(out,
			fmtBare, fmtWithGroup, fmtWithGroupVersion)
	}
	return out
}
