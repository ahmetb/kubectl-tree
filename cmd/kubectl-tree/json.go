package main

import (
	"encoding/json"
	"io"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
)

// treeNode is the machine-readable form of one object in the tree.
type treeNode struct {
	APIVersion        string     `json:"apiVersion"`
	Kind              string     `json:"kind"`
	Namespace         string     `json:"namespace,omitempty"`
	Name              string     `json:"name"`
	UID               types.UID  `json:"uid"`
	Ready             string     `json:"ready,omitempty"`
	Reason            string     `json:"reason,omitempty"`
	Status            string     `json:"status,omitempty"`
	CreationTimestamp *time.Time `json:"creationTimestamp,omitempty"`
	Children          []treeNode `json:"children"`
}

func buildTreeNode(objs objectDirectory, obj unstructured.Unstructured, conditionTypes []string) treeNode {
	ready, reason, kstatus := extractStatus(obj, conditionTypes)
	n := treeNode{
		APIVersion: obj.GetAPIVersion(),
		Kind:       obj.GetKind(),
		Namespace:  obj.GetNamespace(),
		Name:       obj.GetName(),
		UID:        obj.GetUID(),
		Ready:      string(ready),
		Reason:     string(reason),
		Status:     string(kstatus),
		Children:   []treeNode{},
	}
	if c := obj.GetCreationTimestamp(); !c.IsZero() {
		t := c.UTC()
		n.CreationTimestamp = &t
	}
	for _, child := range objs.ownedBy(obj.GetUID()) {
		n.Children = append(n.Children, buildTreeNode(objs, child, conditionTypes))
	}
	return n
}

// jsonView prints object hierarchy to out stream as JSON.
func jsonView(out io.Writer, objs objectDirectory, obj unstructured.Unstructured, conditionTypes []string) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(buildTreeNode(objs, obj, conditionTypes))
}
