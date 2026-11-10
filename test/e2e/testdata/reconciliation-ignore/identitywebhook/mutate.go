// Package identitywebhook supplies admission fixtures for reconciliation tests, without cloud calls.
package identitywebhook

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ProfileLabel = "identity.test/profile"
	ManagedKey   = "identity.test/managed"
)

type patch struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value any    `json:"value"`
}

// Mutate injects provider-shaped credentials into selected test pods on creation only.
func Mutate(request admissionv1.AdmissionRequest) *admissionv1.AdmissionResponse {
	response := &admissionv1.AdmissionResponse{UID: request.UID, Allowed: true}
	if request.Operation != admissionv1.Create || request.Resource.Resource != "pods" {
		return response
	}
	var pod corev1.Pod
	if err := json.Unmarshal(request.Object.Raw, &pod); err != nil {
		response.Allowed = false
		response.Result = &metav1.Status{Message: err.Error()}
		return response
	}
	profile := pod.Labels[ProfileLabel]
	var name, path, audience string
	tokenFile := "token"
	var env []corev1.EnvVar
	switch profile {
	case "azure", "azure-prefix":
		name = "azure-identity-token"
		path = "/var/run/secrets/azure/tokens"
		audience = "api://AzureADTokenExchange"
		if profile == "azure-prefix" {
			hash := sha256.Sum256([]byte(request.UID))
			name = fmt.Sprintf("azure-workload-identity-reserved-%x", hash[:6])
			path = "/var/run/secrets/azure/wi/" + name
		}
		env = []corev1.EnvVar{{Name: "AZURE_CLIENT_ID", Value: "test-client"}, {Name: "AZURE_TENANT_ID", Value: "test-tenant"}, {Name: "AZURE_AUTHORITY_HOST", Value: "https://login.microsoftonline.com/"}, {Name: "AZURE_FEDERATED_TOKEN_FILE", Value: path + "/token"}}
	case "irsa":
		name = "aws-iam-token"
		path = "/var/run/secrets/eks.amazonaws.com/serviceaccount"
		audience = "sts.amazonaws.com"
		env = []corev1.EnvVar{{Name: "AWS_ROLE_ARN", Value: "arn:aws:iam::123456789012:role/test"}, {Name: "AWS_WEB_IDENTITY_TOKEN_FILE", Value: path + "/token"}, {Name: "AWS_REGION", Value: "eu-west-1"}, {Name: "AWS_DEFAULT_REGION", Value: "eu-west-1"}, {Name: "AWS_STS_REGIONAL_ENDPOINTS", Value: "regional"}}
	case "eks-pod-identity":
		name = "eks-pod-identity-token"
		path = "/var/run/secrets/pods.eks.amazonaws.com/serviceaccount"
		audience = "pods.eks.amazonaws.com"
		tokenFile = "eks-pod-identity-token"
		env = []corev1.EnvVar{{Name: "AWS_CONTAINER_AUTHORIZATION_TOKEN_FILE", Value: path + "/" + tokenFile}, {Name: "AWS_CONTAINER_CREDENTIALS_FULL_URI", Value: "http://169.254.170.23/v1/credentials"}}
	case "gcp":
		name = "gcp-identity-token"
		path = "/var/run/secrets/google"
		audience = "identity-test"
		env = []corev1.EnvVar{{Name: "GOOGLE_APPLICATION_CREDENTIALS", Value: path + "/credentials.json"}}
	default:
		return response
	}
	sources := []corev1.VolumeProjection{{ServiceAccountToken: &corev1.ServiceAccountTokenProjection{Audience: audience, Path: tokenFile}}}
	if profile == "gcp" {
		sources = append(sources, corev1.VolumeProjection{ConfigMap: &corev1.ConfigMapProjection{LocalObjectReference: corev1.LocalObjectReference{Name: "identity-config"}}})
	}
	pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{Name: name, VolumeSource: corev1.VolumeSource{Projected: &corev1.ProjectedVolumeSource{Sources: sources}}})
	for i := range pod.Spec.Containers {
		pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, env...)
		pod.Spec.Containers[i].VolumeMounts = append(pod.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{Name: name, MountPath: path, ReadOnly: true})
	}
	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}
	pod.Annotations[ManagedKey] = "mutated"
	pod.Labels[ManagedKey] = "mutated"
	const add = "add"
	encoded, err := json.Marshal([]patch{{Op: add, Path: "/spec/containers", Value: pod.Spec.Containers}, {Op: add, Path: "/spec/volumes", Value: pod.Spec.Volumes}, {Op: add, Path: "/metadata/labels", Value: pod.Labels}, {Op: add, Path: "/metadata/annotations", Value: pod.Annotations}})
	if err != nil {
		response.Allowed = false
		response.Result = &metav1.Status{Message: err.Error()}
		return response
	}
	patchType := admissionv1.PatchTypeJSONPatch
	response.PatchType = &patchType
	response.Patch = encoded
	return response
}
