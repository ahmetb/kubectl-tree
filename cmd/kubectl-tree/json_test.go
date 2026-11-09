package main

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
)

func testObject(apiVersion, kind, name string, uid types.UID, owner *unstructured.Unstructured) unstructured.Unstructured {
	var obj unstructured.Unstructured
	obj.SetAPIVersion(apiVersion)
	obj.SetKind(kind)
	obj.SetNamespace("default")
	obj.SetName(name)
	obj.SetUID(uid)
	if owner != nil {
		obj.SetOwnerReferences([]metav1.OwnerReference{{
			APIVersion: owner.GetAPIVersion(),
			Kind:       owner.GetKind(),
			Name:       owner.GetName(),
			UID:        owner.GetUID(),
		}})
	}
	return obj
}

func TestJSONView(t *testing.T) {
	deploy := testObject("apps/v1", "Deployment", "web", "d1", nil)
	deploy.SetCreationTimestamp(metav1.NewTime(time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)))
	rs := testObject("apps/v1", "ReplicaSet", "web-abc", "r1", &deploy)
	podB := testObject("v1", "Pod", "web-abc-b", "p2", &rs)
	podA := testObject("v1", "Pod", "web-abc-a", "p1", &rs)
	podA.Object["status"] = map[string]interface{}{
		"phase": "Running",
		"conditions": []interface{}{
			map[string]interface{}{"type": "Ready", "status": "False", "reason": "ContainersNotReady"},
		},
	}
	unrelated := testObject("v1", "Pod", "other", "p3", nil)

	objs := newObjectDirectory([]unstructured.Unstructured{deploy, rs, podB, podA, unrelated})

	var buf bytes.Buffer
	if err := jsonView(&buf, objs, deploy, []string{"Ready"}); err != nil {
		t.Fatal(err)
	}

	var got treeNode
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}

	if got.Kind != "Deployment" || got.Name != "web" || got.APIVersion != "apps/v1" || got.Namespace != "default" {
		t.Errorf("unexpected root: %+v", got)
	}
	if got.CreationTimestamp == nil || !got.CreationTimestamp.Equal(time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)) {
		t.Errorf("unexpected creationTimestamp: %v", got.CreationTimestamp)
	}
	if len(got.Children) != 1 || got.Children[0].Name != "web-abc" {
		t.Fatalf("expected one ReplicaSet child, got %+v", got.Children)
	}
	pods := got.Children[0].Children
	if len(pods) != 2 || pods[0].Name != "web-abc-a" || pods[1].Name != "web-abc-b" {
		t.Fatalf("expected pods sorted by name, got %+v", pods)
	}
	if pods[0].Ready != "False" || pods[0].Reason != "ContainersNotReady" || pods[0].Status == "" {
		t.Errorf("unexpected pod status fields: %+v", pods[0])
	}
	if pods[1].Children == nil || len(pods[1].Children) != 0 {
		t.Errorf("leaf should have an empty children list, got %#v", pods[1].Children)
	}
	if bytes.Contains(buf.Bytes(), []byte(`"children": null`)) {
		t.Errorf("children should never be null:\n%s", buf.String())
	}
}
