package main

import (
	"fmt"
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

func findAPIs(client discovery.DiscoveryInterface) (*resourceMap, error) {
	start := time.Now()
	resList, err := client.ServerPreferredResources()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch api groups from kubernetes: %w", err)
	}
	klog.V(2).Infof("queried api discovery in %v", time.Since(start))
	klog.V(3).Infof("found %d items (groups) in server-preferred APIResourceList", len(resList))

	rm := &resourceMap{
		m: make(resourceNameLookup),
	}
	for _, group := range resList {
		klog.V(5).Infof("iterating over group %s/%s (%d apis)", group.GroupVersion, group.APIVersion, len(group.APIResources))
		gv, err := schema.ParseGroupVersion(group.GroupVersion)
		if err != nil {
			return nil, fmt.Errorf("%q cannot be parsed into groupversion: %w", group.GroupVersion, err)
		}

		for _, apiRes := range group.APIResources {
			klog.V(5).Infof("  api=%s namespaced=%v", apiRes.Name, apiRes.Namespaced)
			if !contains(apiRes.Verbs, "list") {
				klog.V(4).Infof("    api (%s) doesn't have required verb, skipping: %v", apiRes.Name, apiRes.Verbs)
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
	for _, vv := range v {
		if vv == s {
			return true
		}
	}
	return false
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

func figureOutKindName(args []string) (string, string, error) {
	if l := len(args); l == 0 || l > 2 {
		return "", "", fmt.Errorf("accepts between 1 and 2 arg(s), received %d", l)
	}
	if len(args) == 2 {
		return args[0], args[1], nil
	}
	seg := strings.Split(args[0], "/")
	if len(seg) < 2 {
		return "", "", fmt.Errorf("specify the kubernetes object in KIND NAME or KIND/NAME form")
	}
	if len(seg) > 2 {
		return "", "", fmt.Errorf("arguments in KIND/NAME form may not have more than one slash")
	}
	return seg[0], seg[1], nil
}
