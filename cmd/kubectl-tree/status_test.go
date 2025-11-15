package main

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestExtractStatusWithMultipleConditionTypes(t *testing.T) {
	tests := []struct {
		name           string
		obj            unstructured.Unstructured
		conditionTypes []string
		wantReady      ReadyStatus
		wantReason     Reason
	}{
		{
			name: "finds Ready condition",
			obj: unstructured.Unstructured{
				Object: map[string]interface{}{
					"status": map[string]interface{}{
						"conditions": []interface{}{
							map[string]interface{}{
								"type":   "Ready",
								"status": "True",
								"reason": "AllGood",
							},
						},
					},
				},
			},
			conditionTypes: []string{"Ready"},
			wantReady:      "True",
			wantReason:     "AllGood",
		},
		{
			name: "finds Processed condition",
			obj: unstructured.Unstructured{
				Object: map[string]interface{}{
					"status": map[string]interface{}{
						"conditions": []interface{}{
							map[string]interface{}{
								"type":   "Processed",
								"status": "True",
								"reason": "ProcessingComplete",
							},
						},
					},
				},
			},
			conditionTypes: []string{"Processed"},
			wantReady:      "True",
			wantReason:     "ProcessingComplete",
		},
		{
			name: "finds first matching condition from multiple types",
			obj: unstructured.Unstructured{
				Object: map[string]interface{}{
					"status": map[string]interface{}{
						"conditions": []interface{}{
							map[string]interface{}{
								"type":   "Scheduled",
								"status": "True",
								"reason": "ScheduledOK",
							},
							map[string]interface{}{
								"type":   "Processed",
								"status": "False",
								"reason": "ProcessingFailed",
							},
						},
					},
				},
			},
			conditionTypes: []string{"Ready", "Processed", "Scheduled"},
			wantReady:      "False",
			wantReason:     "ProcessingFailed",
		},
		{
			name: "returns empty when condition type not found",
			obj: unstructured.Unstructured{
				Object: map[string]interface{}{
					"status": map[string]interface{}{
						"conditions": []interface{}{
							map[string]interface{}{
								"type":   "Ready",
								"status": "True",
								"reason": "AllGood",
							},
						},
					},
				},
			},
			conditionTypes: []string{"NonExistent"},
			wantReady:      "",
			wantReason:     "",
		},
		{
			name: "handles object without status",
			obj: unstructured.Unstructured{
				Object: map[string]interface{}{},
			},
			conditionTypes: []string{"Ready"},
			wantReady:      "",
			wantReason:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotReady, gotReason, _ := extractStatus(tt.obj, tt.conditionTypes)
			if gotReady != tt.wantReady {
				t.Errorf("extractStatus() gotReady = %v, want %v", gotReady, tt.wantReady)
			}
			if gotReason != tt.wantReason {
				t.Errorf("extractStatus() gotReason = %v, want %v", gotReason, tt.wantReason)
			}
		})
	}
}
