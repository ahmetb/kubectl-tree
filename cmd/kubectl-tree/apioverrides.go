package main

import "strings"

// overrideType hardcodes lookup overrides for certain service types
func overrideType(kind string, v *resourceMap) (apiResource, bool) {
	kind = strings.ToLower(kind)

	switch kind {
	case "svc", "service", "services": // Knative also registers "Service", prefer v1.Service
		out := v.lookup("service.v1.")
		if len(out) != 0 {
			return out[0], true
		}

	case "deploy", "deployment", "deployments": // most clusters will have Deployment in apps/v1 and extensions/v1beta1, extensions/v1/beta2
		out := v.lookup("deployment.v1.apps")
		if len(out) != 0 {
			return out[0], true
		}
		out = v.lookup("deployment.v1beta1.extensions")
		if len(out) != 0 {
			return out[0], true
		}
	}
	return apiResource{}, false
}
