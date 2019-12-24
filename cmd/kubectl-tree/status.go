package main

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type ReadyStatus string // True False Unknown or ""
type Reason string

func extractStatus(obj unstructured.Unstructured) (ReadyStatus, Reason) {
	statusF, ok := obj.Object["status"]
	if !ok {
		return "", ""
	}
	statusV, ok := statusF.(map[string]interface{})
	if !ok {
		return "", ""
	}
	conditionsF, ok := statusV["conditions"]
	if !ok {
		return "", ""
	}
	conditionsV, ok := conditionsF.([]interface{})
	if !ok {
		return "", ""
	}

	for _, cond := range conditionsV {
		condM, ok := cond.(map[string]interface{})
		if !ok {
			return "", ""
		}
		condType, ok := condM["type"].(string)
		if !ok {
			return "", ""
		}
		if condType == "Ready" {
			condStatus, _ := condM["status"].(string)
			condReason, _ := condM["reason"].(string)
			return ReadyStatus(condStatus), Reason(condReason)
		}
	}
	return "", ""
}
