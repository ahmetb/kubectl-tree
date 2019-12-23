package main

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
)

type objectDirectory struct {
	items     map[types.UID]unstructured.Unstructured
	ownership map[types.UID]map[types.UID]bool
}

func newObjectDirectory(objs []unstructured.Unstructured) objectDirectory {
	v := objectDirectory{
		items:     make(map[types.UID]unstructured.Unstructured),
		ownership: make(map[types.UID]map[types.UID]bool),
	}
	for _, obj := range objs {
		v.items[obj.GetUID()] = obj
		for _, ownerRef := range obj.GetOwnerReferences() {
			if v.ownership[ownerRef.UID] == nil {
				v.ownership[ownerRef.UID] = make(map[types.UID]bool)
			}
			v.ownership[ownerRef.UID][obj.GetUID()] = true
		}
	}
	return v
}

func (od objectDirectory) getObject(id types.UID) unstructured.Unstructured { return od.items[id] }

func (od objectDirectory) ownedBy(ownerID types.UID) []types.UID {
	var out []types.UID
	for k := range od.ownership[ownerID] {
		out = append(out, k)
	}
	return out
}
