package base

import (
	"reflect"

	corev1 "k8s.io/api/core/v1"
)

func compareContainerConfiguration(expected corev1.Container, actual corev1.Container, policy podComparisonPolicy) (differences []podConfigurationDifference) {
	if !reflect.DeepEqual(expected.Args, actual.Args) {
		differences = append(differences, podConfigurationDifference{Field: "Arguments", Container: expected.Name})
	}
	if !reflect.DeepEqual(expected.Command, actual.Command) {
		differences = append(differences, podConfigurationDifference{Field: "Command", Container: expected.Name})
	}
	if !policy.envEqual(expected.Env, actual.Env) {
		differences = append(differences, podConfigurationDifference{Field: "Env", Container: expected.Name})
	}
	if !reflect.DeepEqual(expected.EnvFrom, actual.EnvFrom) {
		differences = append(differences, podConfigurationDifference{Field: "EnvFrom", Container: expected.Name})
	}
	if !reflect.DeepEqual(expected.Image, actual.Image) {
		differences = append(differences, podConfigurationDifference{Field: "Image", Container: expected.Name})
	}
	if !reflect.DeepEqual(expected.ImagePullPolicy, actual.ImagePullPolicy) {
		differences = append(differences, podConfigurationDifference{Field: "Image pull policy", Container: expected.Name})
	}
	if !reflect.DeepEqual(expected.Lifecycle, actual.Lifecycle) {
		differences = append(differences, podConfigurationDifference{Field: "Lifecycle", Container: expected.Name})
	}
	if !reflect.DeepEqual(expected.LivenessProbe, actual.LivenessProbe) {
		differences = append(differences, podConfigurationDifference{Field: "Liveness probe", Container: expected.Name})
	}
	if !reflect.DeepEqual(expected.Ports, actual.Ports) {
		differences = append(differences, podConfigurationDifference{Field: "Ports", Container: expected.Name})
	}
	if !reflect.DeepEqual(expected.ReadinessProbe, actual.ReadinessProbe) {
		differences = append(differences, podConfigurationDifference{Field: "Readiness probe", Container: expected.Name})
	}
	if !compareContainerResources(expected.Resources, actual.Resources) {
		differences = append(differences, podConfigurationDifference{Field: "Resources", Container: expected.Name})
	}
	if !reflect.DeepEqual(expected.SecurityContext, actual.SecurityContext) {
		differences = append(differences, podConfigurationDifference{Field: "Security context", Container: expected.Name})
	}
	if !reflect.DeepEqual(expected.WorkingDir, actual.WorkingDir) {
		differences = append(differences, podConfigurationDifference{Field: "Working directory", Container: expected.Name})
	}
	if !policy.mountsEqual(expected, actual) {
		differences = append(differences, podConfigurationDifference{Field: "Volume mounts", Container: expected.Name})
	}
	return differences
}

func compareContainerResources(expected corev1.ResourceRequirements, actual corev1.ResourceRequirements) bool {
	expectedRequestCPU, expectedRequestCPUSet := expected.Requests[corev1.ResourceCPU]
	expectedRequestMemory, expectedRequestMemorySet := expected.Requests[corev1.ResourceMemory]
	expectedLimitCPU, expectedLimitCPUSet := expected.Limits[corev1.ResourceCPU]
	expectedLimitMemory, expectedLimitMemorySet := expected.Limits[corev1.ResourceMemory]
	actualRequestCPU, actualRequestCPUSet := actual.Requests[corev1.ResourceCPU]
	actualRequestMemory, actualRequestMemorySet := actual.Requests[corev1.ResourceMemory]
	actualLimitCPU, actualLimitCPUSet := actual.Limits[corev1.ResourceCPU]
	actualLimitMemory, actualLimitMemorySet := actual.Limits[corev1.ResourceMemory]

	if expectedRequestCPUSet && (!actualRequestCPUSet || expectedRequestCPU.String() != actualRequestCPU.String()) {
		return false
	}
	if expectedRequestMemorySet && (!actualRequestMemorySet || expectedRequestMemory.String() != actualRequestMemory.String()) {
		return false
	}
	if expectedLimitCPUSet && (!actualLimitCPUSet || expectedLimitCPU.String() != actualLimitCPU.String()) {
		return false
	}
	if expectedLimitMemorySet && (!actualLimitMemorySet || expectedLimitMemory.String() != actualLimitMemory.String()) {
		return false
	}
	return true
}
