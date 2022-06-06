package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog"
)

// getAllResources finds all API objects in specified API resources in all namespaces (or non-namespaced).
func getAllResources(client dynamic.Interface, apis []apiResource, allNs bool) ([]unstructured.Unstructured, error) {
	var mu sync.Mutex
	var wg sync.WaitGroup
	var out []unstructured.Unstructured

	start := time.Now()
	klog.V(2).Infof("starting to query %d APIs in concurrently", len(apis))

	var errResult error
	for _, api := range apis {
		if !allNs && !api.r.Namespaced {
			klog.V(4).Infof("[query api] api (%s) is non-namespaced, skipping", api.r.Name)
			continue
		}
		wg.Add(1)
		go func(a apiResource) {
			defer wg.Done()
			klog.V(4).Infof("[query api] start: %s", a.GroupVersionResource())
			v, err := queryAPI(client, a, allNs)
			if err != nil {
				klog.V(4).Infof("[query api] error querying: %s, error=%v", a.GroupVersionResource(), err)
				errResult = err
				return
			}
			mu.Lock()
			out = append(out, v...)
			mu.Unlock()
			klog.V(4).Infof("[query api]  done: %s, found %d apis", a.GroupVersionResource(), len(v))
		}(api)
	}

	klog.V(2).Infof("fired up all goroutines to query APIs")
	wg.Wait()
	klog.V(2).Infof("all goroutines have returned in %v", time.Since(start))
	klog.V(2).Infof("query result: error=%v, objects=%d", errResult, len(out))
	return out, errResult
}

func queryAPI(client dynamic.Interface, api apiResource, allNs bool) ([]unstructured.Unstructured, error) {
	var out []unstructured.Unstructured

	var next string
	var ns string

	if !allNs {
		ns = getNamespace()
	}
	for {
		var intf dynamic.ResourceInterface
		nintf := client.Resource(api.GroupVersionResource())
		if !allNs {
			intf = nintf.Namespace(ns)
		} else {
			intf = nintf
		}
		resp, err := intf.List(context.TODO(), metav1.ListOptions{
			Limit:    250,
			Continue: next,
		})
		if err != nil {
			return nil, fmt.Errorf("listing resources failed (%s): %w", api.GroupVersionResource(), err)
		}
		out = append(out, resp.Items...)

		next = resp.GetContinue()
		if next == "" {
			break
		}
	}
	return out, nil
}
