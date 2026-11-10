package base

import (
	"testing"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

func TestWellKnownInjectionCatalog(t *testing.T) {
	empty := v1alpha2.JenkinsReconciliationIgnore{}
	enabled := newPodComparisonPolicy(empty, true)
	disabled := newPodComparisonPolicy(empty, false)
	for _, name := range []string{"AWS_ROLE_ARN", "AWS_WEB_IDENTITY_TOKEN_FILE", "AWS_REGION", "AWS_DEFAULT_REGION", "AWS_STS_REGIONAL_ENDPOINTS", "AWS_CONTAINER_CREDENTIALS_FULL_URI", "AWS_CONTAINER_AUTHORIZATION_TOKEN_FILE", "AZURE_CLIENT_ID", "AZURE_TENANT_ID", "AZURE_AUTHORITY_HOST", "AZURE_FEDERATED_TOKEN_FILE", "GOOGLE_APPLICATION_CREDENTIALS"} {
		env := []corev1.EnvVar{{Name: name, ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}}}
		require.True(t, enabled.envEqual(nil, env), name)
		require.True(t, enabled.envEqual(env, nil), name)
		require.False(t, disabled.envEqual(nil, env), name)
	}
	for _, name := range []string{"aws-iam-token", "eks-pod-identity-token", "azure-identity-token", "azure-workload-identity-reserved-first", "azure-workload-identity-reserved-second"} {
		require.True(t, enabled.ignoresVolume(name), name)
		require.False(t, disabled.ignoresVolume(name), name)
		require.True(t, enabled.mountsEqual(corev1.Container{}, corev1.Container{VolumeMounts: []corev1.VolumeMount{{Name: name, MountPath: "/identity"}}}), name)
	}
	for _, name := range []string{"AWS_SECRET_ACCESS_KEY", "AWS_CUSTOM", "AZURE_CUSTOM", "GOOGLE_CLOUD_PROJECT", "aws_role_arn", "ISTIO_META_POD_NAME"} {
		require.False(t, enabled.envEqual(nil, []corev1.EnvVar{{Name: name, Value: "managed"}}), name)
	}
	for _, name := range []string{"aws-custom-token", "gcp-identity-token", "azure-workload-identity-reserve-other", "istio-envoy", "linkerd-identity-end-entity"} {
		require.False(t, enabled.ignoresVolume(name), name)
	}
	require.False(t, compareMapIgnoringKeys(map[string]string{"azure.workload.identity/use": "true"}, nil, enabled.labels))
	require.False(t, compareMapIgnoringKeys(map[string]string{"azure.workload.identity/client-id": "desired"}, nil, enabled.annotations))
}

func TestWellKnownInjectionsCombineWithManualRules(t *testing.T) {
	j := comparisonJenkins()
	ignore := v1alpha2.JenkinsReconciliationIgnore{Env: []string{"CUSTOM_ENV"}, Volumes: []string{"custom-token"}, VolumeNamePrefixes: []string{"custom-"}, Annotations: []string{"managed"}, Labels: []string{"managed"}}
	j.Spec.Reconciliation = v1alpha2.JenkinsReconciliation{IgnoreWellKnownInjections: ptr.To(true), Ignore: ignore}
	original := j.DeepCopy()
	desired := desiredPodForComparison(j)
	actual := desired.DeepCopy()
	actual.Labels["managed"] = "injected"
	actual.Annotations["managed"] = "injected"
	for i := range actual.Spec.Containers {
		actual.Spec.Containers[i].Env = append(actual.Spec.Containers[i].Env, corev1.EnvVar{Name: "AWS_ROLE_ARN", Value: "injected"}, corev1.EnvVar{Name: "CUSTOM_ENV", Value: "custom"})
		for _, name := range []string{"aws-iam-token", "custom-token", "custom-second"} {
			actual.Spec.Containers[i].VolumeMounts = append(actual.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{Name: name, MountPath: "/" + name})
		}
	}
	for _, name := range []string{"aws-iam-token", "custom-token", "custom-second"} {
		actual.Spec.Volumes = append(actual.Spec.Volumes, corev1.Volume{Name: name, VolumeSource: corev1.VolumeSource{Projected: &corev1.ProjectedVolumeSource{Sources: []corev1.VolumeProjection{{ServiceAccountToken: &corev1.ServiceAccountTokenProjection{Audience: "test", Path: "token"}}}}}})
	}
	snapshot := actual.DeepCopy()
	policy := newPodComparisonPolicy(ignore, true)
	require.Empty(t, comparePodConfiguration(*desired, *actual, policy))
	require.Empty(t, comparePodConfiguration(*desired, *actual, policy))
	require.Equal(t, original, j)
	require.Equal(t, snapshot, actual)
	require.NotEmpty(t, comparePodConfiguration(*desired, *actual, newPodComparisonPolicy(ignore, false)))
	// An unrelated projection and source changes remain managed with the catalog enabled.
	unrelated := corev1.Volume{Name: "azure-workload-identity-reserve-other", VolumeSource: corev1.VolumeSource{Projected: &corev1.ProjectedVolumeSource{Sources: []corev1.VolumeProjection{{ServiceAccountToken: &corev1.ServiceAccountTokenProjection{Audience: "original", Path: "token"}}}}}}
	desired.Spec.Volumes = append(desired.Spec.Volumes, unrelated)
	actual.Spec.Volumes = append(actual.Spec.Volumes, *unrelated.DeepCopy())
	require.Empty(t, comparePodConfiguration(*desired, *actual, policy))
	actual.Spec.Volumes[len(actual.Spec.Volumes)-1].Projected.Sources[0].ServiceAccountToken.Audience = "changed"
	require.NotEmpty(t, comparePodConfiguration(*desired, *actual, policy))
}

func TestWellKnownSwitchWiredToRecreation(t *testing.T) {
	j := comparisonJenkins()
	actual := desiredPodForComparison(j)
	actual.Spec.Containers[0].Env = append(actual.Spec.Containers[0].Env, corev1.EnvVar{Name: "GOOGLE_APPLICATION_CREDENTIALS", Value: "/injected.json"})
	r := New(configuration.Configuration{Jenkins: j}, jenkinsclient.JenkinsAPIConnectionSettings{})
	require.False(t, r.checkForPodRecreation(*actual, "same").HasMessages())
	j.Spec.Reconciliation.IgnoreWellKnownInjections = ptr.To(false)
	require.True(t, r.checkForPodRecreation(*actual, "same").HasMessages())
	j.Spec.Reconciliation.IgnoreWellKnownInjections = ptr.To(true)
	require.False(t, r.checkForPodRecreation(*actual, "same").HasMessages())
	require.True(t, r.checkForPodRecreation(*actual, "changed").HasMessages())
	actual.Spec.Containers[0].Image = "retained:image"
	require.True(t, r.checkForPodRecreation(*actual, "same").HasMessages())
	actual.Spec.Containers[0].Image = j.Spec.Master.Containers[0].Image
	actual.Spec.Containers = append(actual.Spec.Containers, corev1.Container{Name: "istio-proxy", Image: "mesh:test"})
	require.NotEmpty(t, comparePodConfiguration(*desiredPodForComparison(j), *actual, newPodComparisonPolicy(j.Spec.Reconciliation.Ignore, true)))
}
