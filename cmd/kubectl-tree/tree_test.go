package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/fatih/color"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestTreeViewCompletedPodReason(t *testing.T) {
	oldGray, oldRed, oldYellow, oldGreen := gray, red, yellow, green
	oldNoColor := color.NoColor
	t.Cleanup(func() {
		gray, red, yellow, green = oldGray, oldRed, oldYellow, oldGreen
		color.NoColor = oldNoColor
	})

	for _, tc := range []struct {
		name, apiVersion, kind, phase, reason string
		noColor                               bool
		wantReason                            string
	}{
		{"completed pod", "v1", "Pod", "Succeeded", "PodCompleted", false, "\x1b[32mPodCompleted\x1b[0m"},
		{"failed pod", "v1", "Pod", "Failed", "PodFailed", false, "\x1b[31mPodFailed\x1b[0m"},
		{"incomplete pod", "v1", "Pod", "Running", "PodCompleted", false, "\x1b[31mPodCompleted\x1b[0m"},
		{"other kind", "v1", "Service", "Succeeded", "PodCompleted", false, "\x1b[31mPodCompleted\x1b[0m"},
		{"custom pod kind", "example.com/v1", "Pod", "Succeeded", "PodCompleted", false, "\x1b[31mPodCompleted\x1b[0m"},
		{"other reason", "v1", "Pod", "Succeeded", "OtherReason", false, "\x1b[31mOtherReason\x1b[0m"},
		{"no color", "v1", "Pod", "Succeeded", "PodCompleted", true, "PodCompleted"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			color.NoColor = tc.noColor
			gray, red, yellow, green = color.New(color.FgHiBlack), color.New(color.FgRed), color.New(color.FgYellow), color.New(color.FgGreen)
			for _, c := range []*color.Color{gray, red, yellow, green} {
				if tc.noColor {
					c.DisableColor()
				} else {
					c.EnableColor()
				}
			}
			obj := unstructured.Unstructured{Object: map[string]interface{}{
				"apiVersion": tc.apiVersion,
				"kind":       tc.kind,
				"metadata":   map[string]interface{}{"name": "test", "namespace": "default"},
				"status": map[string]interface{}{
					"phase": tc.phase,
					"conditions": []interface{}{map[string]interface{}{
						"type": "Ready", "status": "False", "reason": tc.reason,
					}},
				},
			}}
			var out bytes.Buffer
			treeView(&out, newObjectDirectory(nil), obj, []string{"Ready"})
			if !strings.Contains(out.String(), tc.wantReason) {
				t.Errorf("output %q does not contain reason %q", out.String(), tc.wantReason)
			}
			if tc.noColor {
				if strings.Contains(out.String(), "\x1b[") || !strings.Contains(out.String(), "False") {
					t.Errorf("expected plain output retaining Ready=False, got %q", out.String())
				}
			} else if !strings.Contains(out.String(), "\x1b[31mFalse\x1b[0m") {
				t.Errorf("expected Ready=False to remain red, got %q", out.String())
			}
		})
	}
}
