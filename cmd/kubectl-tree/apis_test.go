package main

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMatchGroups(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		value    string
		want     bool
	}{
		{name: "empty patterns", value: "apps", want: true},
		{name: "positive match", patterns: []string{"apps"}, value: "apps", want: true},
		{name: "positive glob match", patterns: []string{"app*"}, value: "apps", want: true},
		{name: "positive glob miss", patterns: []string{"app*"}, value: "batch", want: false},
		{name: "positive miss", patterns: []string{"apps"}, value: "batch", want: false},
		{name: "negative match excluded", patterns: []string{"!batch"}, value: "batch", want: false},
		{name: "negative miss included", patterns: []string{"!batch"}, value: "apps", want: true},
		{name: "positive and negative", patterns: []string{"*", "!batch"}, value: "batch", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchGroups(tt.patterns, tt.value); got != tt.want {
				t.Fatalf("matchAny(%v, %q) = %v, want %v", tt.patterns, tt.value, got, tt.want)
			}
		})
	}
}

func TestMatchResources(t *testing.T) {
	apiRes := metav1.APIResource{
		Name:         "pods",
		SingularName: "pod",
		Kind:         "Pod",
		ShortNames:   []string{"po"},
	}

	tests := []struct {
		name     string
		patterns []string
		want     bool
	}{
		{name: "exclude plural excludes resource", patterns: []string{"!pods"}, want: false},
		{name: "exclude singular excludes resource", patterns: []string{"!pod"}, want: false},
		{name: "exclude short name excludes resource", patterns: []string{"!po"}, want: false},
		{name: "positive include works", patterns: []string{"pods"}, want: true},
		{name: "positive and negative prefers exclusion", patterns: []string{"*", "!po"}, want: false},
		{name: "negative unrelated keeps resource", patterns: []string{"!deployments"}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchResources(tt.patterns, apiRes); got != tt.want {
				t.Fatalf("matchResources(%v, %v) = %v, want %v", tt.patterns, apiRes.Name, got, tt.want)
			}
		})
	}
}
