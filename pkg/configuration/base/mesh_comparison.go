package base

import (
	"encoding/json"
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

type meshInjection struct {
	podComparisonExclusions
	istio bool
}

func recognizedMeshInjection(pod *corev1.Pod) meshInjection {
	const regularContainers = "containers"
	const nativeContainers = "initContainers"
	injection := meshInjection{podComparisonExclusions: newPodComparisonExclusions()}
	proxyLocation := func(name string) string {
		location := ""
		for _, c := range pod.Spec.Containers {
			if c.Name == name {
				if location != "" || c.Image == "" {
					return ""
				}
				location = regularContainers
			}
		}
		for _, c := range pod.Spec.InitContainers {
			if c.Name == name {
				if location != "" || c.Image == "" || ptr.Deref(c.RestartPolicy, corev1.ContainerRestartPolicy("")) != corev1.ContainerRestartPolicyAlways {
					return ""
				}
				location = nativeContainers
			}
		}
		return location
	}
	if location := proxyLocation("istio-proxy"); location != "" {
		var status struct {
			Containers     []string `json:"containers"`
			InitContainers []string `json:"initContainers"`
			Volumes        []string `json:"volumes"`
		}
		if json.Unmarshal([]byte(pod.Annotations["sidecar.istio.io/status"]), &status) == nil && ((location == regularContainers && slices.Contains(status.Containers, "istio-proxy")) || (location == nativeContainers && slices.Contains(status.InitContainers, "istio-proxy"))) {
			injection.istio = true
			if location == regularContainers {
				injection.containers.Insert("istio-proxy")
			} else {
				injection.initContainers.Insert("istio-proxy")
			}
			for _, name := range status.InitContainers {
				if name == "istio-init" || name == "istio-validation" {
					injection.initContainers.Insert(name)
				}
			}
			for _, name := range status.Volumes {
				switch name {
				case "workload-socket", "credential-socket", "workload-certs", "istio-envoy", "istio-data", "istio-podinfo", "istio-token", "istiod-ca-cert", "istio-ca-crl":
					injection.volumes.Insert(name)
				}
			}
		}
	}
	if location := proxyLocation("linkerd-proxy"); location != "" && pod.Annotations["linkerd.io/proxy-version"] != "" && strings.HasPrefix(pod.Annotations["linkerd.io/created-by"], "linkerd/proxy-injector ") && pod.Labels["linkerd.io/control-plane-ns"] != "" {
		if location == regularContainers {
			injection.containers.Insert("linkerd-proxy")
		} else {
			injection.initContainers.Insert("linkerd-proxy")
		}
		injection.initContainers.Insert("linkerd-init", "linkerd-network-validator")
		injection.volumes.Insert("linkerd-identity-end-entity", "linkerd-identity-token", "linkerd-proxy-init-xtables-lock")
	}
	return injection
}

func (injection meshInjection) validate(expected *corev1.Pod) []podConfigurationDifference {
	var differences []podConfigurationDifference
	if injection.istio && expected.Annotations["sidecar.istio.io/rewriteAppHTTPProbers"] != "false" {
		differences = append(differences, podConfigurationDifference{Message: `Istio requires spec.master.annotations[sidecar.istio.io/rewriteAppHTTPProbers] to be "false"; rewritten probes are unsupported`})
	}
	return differences
}
