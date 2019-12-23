package main

import (
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"sync"
)

func QueryResources(client dynamic.Interface, apis []apiResource) ([]unstructured.Unstructured, error) {
	var mu sync.Mutex
	var wg sync.WaitGroup
	var out []unstructured.Unstructured

	var errResult error
	for _, api := range apis {
		wg.Add(1)
		go func(a apiResource) {
			defer wg.Done()
			v, err := queryAPI(client, a)
			if err != nil {
				errResult = err
				return
			}
			mu.Lock()
			out = append(out, v...)
			mu.Unlock()
		}(api)
	}
	wg.Wait()
	return out, errResult
}

func queryAPI(client dynamic.Interface, api apiResource) ([]unstructured.Unstructured, error) {
	var out []unstructured.Unstructured

	var next string
	for {
		resp, err := client.Resource(api.GroupVersionResource()).List(metav1.ListOptions{
			Limit:    250,
			Continue: next,
		})
		if err != nil {
			return nil, fmt.Errorf("listing resources failed (%s): %w", api.GroupVersionResource(), err)
		}
		out = append(out, resp.Items...)

		fmt.Printf("found %d objects in %s (next=%s)\n", len(resp.Items), api.GroupVersionResource(), resp.GetContinue())
		next = resp.GetContinue()
		if next == "" {
			break
		}
	}
	return out, nil
}
