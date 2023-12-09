package main

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog"
	"sigs.k8s.io/cli-utils/pkg/kstatus/status"
)

type ReadyStatus string // True False Unknown or ""
type Reason string

func extractStatus(obj unstructured.Unstructured) (ReadyStatus, Reason, status.Status) {
	jsonVal, _ := json.Marshal(obj.Object["status"])
	klog.V(6).Infof("status for object=%s/%s: %s", obj.GetKind(), obj.GetName(), string(jsonVal))
	result, err := status.Compute(&obj)
	if err != nil {
		return "", "", ""
	}
	statusF, ok := obj.Object["status"]
	if !ok {
		return "", "", ""
	}
	statusV, ok := statusF.(map[string]interface{})
	if !ok {
		return "", "", ""
	}
	conditionsF, ok := statusV["conditions"]
	if !ok {
		return "", "", ""
	}
	conditionsV, ok := conditionsF.([]interface{})
	if !ok {
		return "", "", ""
	}

	for _, cond := range conditionsV {
		condM, ok := cond.(map[string]interface{})
		if !ok {
			return "", "", ""
		}
		condType, ok := condM["type"].(string)
		if !ok {
			return "", "", ""
		}
		if condType == "Ready" {
			condStatus, _ := condM["status"].(string)
			condReason, _ := condM["reason"].(string)
			return ReadyStatus(condStatus), Reason(condReason), status.Status(result.Status.String())
		}
	}
	return "", "", ""
}
