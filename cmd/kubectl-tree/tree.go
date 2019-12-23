package main

import (
	"fmt"
	"io"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func treeView(out io.Writer, objs objectDirectory, obj unstructured.Unstructured) {
	treeViewInner("", out, objs, obj)
}

func treeViewInner(prefix string, out io.Writer, objs objectDirectory, obj unstructured.Unstructured) {
	fmt.Fprintf(out, prefix+"%s/%s (#%s#)\n", obj.GetKind(), obj.GetName(),obj.GetUID())
	for _, child := range objs.ownedBy(obj.GetUID()) {
		treeViewInner(prefix+"  ", out, objs, objs.getObject(child))
	}
}
