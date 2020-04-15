package base

import (
	"fmt"
	"reflect"

	corev1 "k8s.io/api/core/v1"
)

func (r *ReconcileJenkinsBaseConfiguration) compareContainers(expected corev1.Container, actual corev1.Container) (messages []string, verbose []string) {
	if !reflect.DeepEqual(expected.Args, actual.Args) {
		messages = append(messages, "Arguments have changed")
		verbose = append(messages, fmt.Sprintf("Arguments have changed to '%+v' in container '%s'", expected.Args, expected.Name))
	}
	if !reflect.DeepEqual(expected.Command, actual.Command) {
		messages = append(messages, "Command has changed")
		verbose = append(verbose, fmt.Sprintf("Command has changed to '%+v' in container '%s'", expected.Command, expected.Name))
	}
	if !compareEnv(expected.Env, actual.Env) {
		messages = append(messages, "Env has changed")
		verbose = append(verbose, fmt.Sprintf("Env has changed to '%+v' in container '%s'", expected.Env, expected.Name))
	}
	if !reflect.DeepEqual(expected.EnvFrom, actual.EnvFrom) {
		messages = append(messages, "EnvFrom has changed")
		verbose = append(verbose, fmt.Sprintf("EnvFrom has changed to '%+v' in container '%s'", expected.EnvFrom, expected.Name))
	}
	if !reflect.DeepEqual(expected.Image, actual.Image) {
		messages = append(messages, "Image has changed")
		verbose = append(verbose, fmt.Sprintf("Image has changed to '%+v' in container '%s'", expected.Image, expected.Name))
	}
	if !reflect.DeepEqual(expected.ImagePullPolicy, actual.ImagePullPolicy) {
		messages = append(messages, "Image pull policy has changed")
		verbose = append(verbose, fmt.Sprintf("Image pull policy has changed to '%+v' in container '%s'", expected.ImagePullPolicy, expected.Name))
	}
	if !reflect.DeepEqual(expected.Lifecycle, actual.Lifecycle) {
		messages = append(messages, "Lifecycle has changed")
		verbose = append(verbose, fmt.Sprintf("Lifecycle has changed to '%+v' in container '%s'", expected.Lifecycle, expected.Name))
	}
	if !reflect.DeepEqual(expected.LivenessProbe, actual.LivenessProbe) {
		messages = append(messages, "Liveness probe has changed")
		verbose = append(verbose, fmt.Sprintf("Liveness probe has changed to '%+v' in container '%s'", expected.LivenessProbe, expected.Name))
	}
	if !reflect.DeepEqual(expected.Ports, actual.Ports) {
		messages = append(messages, "Ports have changed")
		verbose = append(verbose, fmt.Sprintf("Ports have changed to '%+v' in container '%s'", expected.Ports, expected.Name))
	}
	if !reflect.DeepEqual(expected.ReadinessProbe, actual.ReadinessProbe) {
		messages = append(messages, "Readiness probe has changed")
		verbose = append(verbose, fmt.Sprintf("Readiness probe has changed to '%+v' in container '%s'", expected.ReadinessProbe, expected.Name))
	}
	if !compareContainerResources(expected.Resources, actual.Resources) {
		messages = append(messages, "Resources have changed")
		verbose = append(verbose, fmt.Sprintf("Resources have changed to '%+v' in container '%s'", expected.Resources, expected.Name))
	}
	if !reflect.DeepEqual(expected.SecurityContext, actual.SecurityContext) {
		messages = append(messages, "Security context has changed")
		verbose = append(verbose, fmt.Sprintf("Security context has changed to '%+v' in container '%s'", expected.SecurityContext, expected.Name))
	}
	if !reflect.DeepEqual(expected.WorkingDir, actual.WorkingDir) {
		messages = append(messages, "Working directory has changed")
		verbose = append(verbose, fmt.Sprintf("Working directory has changed to '%+v' in container '%s'", expected.WorkingDir, expected.Name))
	}
	if !CompareContainerVolumeMounts(expected, actual) {
		messages = append(messages, "Volume mounts have changed")
		verbose = append(verbose, fmt.Sprintf("Volume mounts have changed to '%+v' in container '%s'", expected.VolumeMounts, expected.Name))
	}
	return messages, verbose
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
