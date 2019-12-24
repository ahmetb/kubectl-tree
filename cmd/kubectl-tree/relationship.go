package main

import (
	"sort"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
)

// objectDirectory stores objects and owner relationships between them.
type objectDirectory struct {
	items     map[types.UID]unstructured.Unstructured
	ownership map[types.UID]map[types.UID]bool
}

// newObjectDirectory builds object lookup and hierarchy.
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

// getObject finds object by ID, since objectDirectory is built with specified objects, id should exist in there.
func (od objectDirectory) getObject(id types.UID) unstructured.Unstructured { return od.items[id] }

// ownedBy returns objects directly owned by specified id, sorted by Kind, then by Name, then by Namespace.
func (od objectDirectory) ownedBy(id types.UID) []unstructured.Unstructured {
	var out sortedObjects
	for k := range od.ownership[id] {
		out = append(out, od.getObject(k))
	}
	sort.Sort(out)
	return out
}

// sortedObjects sorts objects by Kind, then by Name, then by Namespace.
type sortedObjects []unstructured.Unstructured

func (s sortedObjects) Len() int { return len(s) }

func (s sortedObjects) Less(i, j int) bool {
	a, b := s[i], s[j]
	if a.GetKind() != b.GetKind() {
		return a.GetKind() < b.GetKind()
	}
	if a.GetName() != b.GetName() {
		return a.GetName() < b.GetName()
	}
	return a.GetNamespace() < b.GetNamespace()
}

func (s sortedObjects) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
