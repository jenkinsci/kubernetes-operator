package base

import (
	"strings"
	"testing"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/version"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func comparisonJenkins() *v1alpha2.Jenkins {
	now := metav1.Now()
	return &v1alpha2.Jenkins{ObjectMeta: metav1.ObjectMeta{Name: "comparison"}, Spec: v1alpha2.JenkinsSpec{Master: v1alpha2.JenkinsMaster{
		Labels: map[string]string{"managed": "desired"}, Annotations: map[string]string{"managed": "desired"},
		Containers: []v1alpha2.Container{{Name: resources.JenkinsMasterContainerName, Image: "jenkins:test", Env: []corev1.EnvVar{{Name: "MANAGED", Value: "desired"}}}, {Name: "backup", Image: "backup:test"}},
	}}, Status: v1alpha2.JenkinsStatus{OperatorVersion: version.Version, UserAndPasswordHash: "same", BaseConfigurationCompletedTime: &now, UserConfigurationCompletedTime: &now}}
}

func TestPodReconciliationPolicy(t *testing.T) {
	cases := []struct {
		name   string
		ignore v1alpha2.JenkinsReconciliationIgnore
		mutate func(*corev1.Pod)
		equal  bool
	}{
		{"baseline", v1alpha2.JenkinsReconciliationIgnore{}, func(_ *corev1.Pod) {}, true},
		{"extra metadata", v1alpha2.JenkinsReconciliationIgnore{}, func(p *corev1.Pod) { p.Labels["extra"] = "accepted"; p.Annotations["extra"] = "accepted" }, true},
		{"changed label", v1alpha2.JenkinsReconciliationIgnore{}, func(p *corev1.Pod) { p.Labels["managed"] = "mutated" }, false},
		{"ignored label", v1alpha2.JenkinsReconciliationIgnore{Labels: []string{"managed"}}, func(p *corev1.Pod) { delete(p.Labels, "managed") }, true},
		{"ignored annotation", v1alpha2.JenkinsReconciliationIgnore{Annotations: []string{"managed"}}, func(p *corev1.Pod) { p.Annotations["managed"] = "mutated" }, true},
		{"case sensitive", v1alpha2.JenkinsReconciliationIgnore{Annotations: []string{"MANAGED"}}, func(p *corev1.Pod) { p.Annotations["managed"] = "mutated" }, false},
		{"ignored env valueFrom", v1alpha2.JenkinsReconciliationIgnore{Env: []string{"MANAGED"}}, func(p *corev1.Pod) {
			for i := range p.Spec.Containers[0].Env {
				if p.Spec.Containers[0].Env[i].Name == "MANAGED" {
					p.Spec.Containers[0].Env[i] = corev1.EnvVar{Name: "MANAGED", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}}
				}
			}
		}, true},
		{"envFrom remains managed", v1alpha2.JenkinsReconciliationIgnore{Env: []string{"MANAGED"}}, func(p *corev1.Pod) { p.Spec.Containers[0].EnvFrom = []corev1.EnvFromSource{{Prefix: "changed"}} }, false},
		{"empty prefix does not match all", v1alpha2.JenkinsReconciliationIgnore{VolumeNamePrefixes: []string{""}}, func(p *corev1.Pod) { p.Spec.Volumes = append(p.Spec.Volumes, corev1.Volume{Name: "unmanaged"}) }, false},
		{"unrelated image", v1alpha2.JenkinsReconciliationIgnore{Env: []string{"MANAGED"}}, func(p *corev1.Pod) { p.Spec.Containers[0].Image = "other:image" }, false},
		{"retained probe", v1alpha2.JenkinsReconciliationIgnore{Env: []string{"MANAGED"}}, func(p *corev1.Pod) { p.Spec.Containers[0].LivenessProbe = &corev1.Probe{TimeoutSeconds: 99} }, false},
		{"retained env order", v1alpha2.JenkinsReconciliationIgnore{Env: []string{"OTHER"}}, func(p *corev1.Pod) {
			p.Spec.Containers[0].Env = append([]corev1.EnvVar{{Name: "OTHER", Value: "ignored"}, {Name: "ADDED", Value: "retained"}}, p.Spec.Containers[0].Env...)
		}, false},
		{"extra container", v1alpha2.JenkinsReconciliationIgnore{}, func(p *corev1.Pod) { p.Spec.Containers = append(p.Spec.Containers, corev1.Container{Name: "injected"}) }, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			j := comparisonJenkins()
			j.Spec.Reconciliation.Ignore = tc.ignore
			original := j.DeepCopy()
			desired := desiredPodForComparison(j)
			require.Equal(t, original, j)
			desiredSnapshot := desired.DeepCopy()
			actual := desired.DeepCopy()
			tc.mutate(actual)
			snapshot := actual.DeepCopy()
			policy := newPodComparisonPolicy(tc.ignore, false)
			differences := comparePodConfiguration(*desired, *actual, policy)
			require.Equal(t, tc.equal, len(differences) == 0, "%+v", differences)
			require.Equal(t, differences, comparePodConfiguration(*desired, *actual, policy))
			require.Equal(t, snapshot, actual)
			require.Equal(t, desiredSnapshot, desired)
			require.Equal(t, original, j)
		})
	}
}

func TestPodReconciliationIdentityProfiles(t *testing.T) {
	profiles := []struct {
		name, volume, path, audience string
		env                          []string
		prefix                       bool
	}{
		{"azure", "azure-identity-token", "/var/run/secrets/azure/tokens", "api://AzureADTokenExchange", []string{"AZURE_CLIENT_ID", "AZURE_TENANT_ID", "AZURE_AUTHORITY_HOST", "AZURE_FEDERATED_TOKEN_FILE"}, false},
		{"azure hashed", "azure-workload-identity-reserved-first", "/var/run/secrets/azure/wi/first", "api://AzureADTokenExchange", []string{"AZURE_CLIENT_ID", "AZURE_TENANT_ID", "AZURE_AUTHORITY_HOST", "AZURE_FEDERATED_TOKEN_FILE"}, true},
		{"AWS IRSA", "aws-iam-token", "/var/run/secrets/eks.amazonaws.com/serviceaccount", "sts.amazonaws.com", []string{"AWS_ROLE_ARN", "AWS_WEB_IDENTITY_TOKEN_FILE", "AWS_REGION", "AWS_DEFAULT_REGION", "AWS_STS_REGIONAL_ENDPOINTS"}, false},
		{"GCP external account", "gcp-identity-token", "/var/run/secrets/google", "identity-test", []string{"GOOGLE_APPLICATION_CREDENTIALS"}, false},
	}
	for _, profile := range profiles {
		t.Run(profile.name, func(t *testing.T) {
			j := comparisonJenkins()
			desired := desiredPodForComparison(j)
			actual := desired.DeepCopy()
			inject := func(p *corev1.Pod, name string) {
				p.Spec.Volumes = append(p.Spec.Volumes, corev1.Volume{Name: name, VolumeSource: corev1.VolumeSource{Projected: &corev1.ProjectedVolumeSource{Sources: []corev1.VolumeProjection{{ServiceAccountToken: &corev1.ServiceAccountTokenProjection{Audience: profile.audience, Path: "token"}}}}}})
				for i := range p.Spec.Containers {
					p.Spec.Containers[i].VolumeMounts = append(p.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{Name: name, MountPath: profile.path, ReadOnly: true})
					for _, env := range profile.env {
						p.Spec.Containers[i].Env = append(p.Spec.Containers[i].Env, corev1.EnvVar{Name: env, Value: "injected"})
					}
				}
			}
			inject(actual, profile.volume)
			require.NotEmpty(t, comparePodConfiguration(*desired, *actual, newPodComparisonPolicy(v1alpha2.JenkinsReconciliationIgnore{}, false)))
			ignore := v1alpha2.JenkinsReconciliationIgnore{Env: profile.env, Volumes: []string{profile.volume}}
			if profile.prefix {
				ignore.Volumes = nil
				ignore.VolumeNamePrefixes = []string{"azure-workload-identity-reserved-"}
			}
			policy := newPodComparisonPolicy(ignore, false)
			require.Empty(t, comparePodConfiguration(*desired, *actual, policy))
			if profile.prefix {
				second := desired.DeepCopy()
				inject(second, "azure-workload-identity-reserved-second")
				require.Empty(t, comparePodConfiguration(*desired, *second, policy))
			}
			actual.Spec.Volumes = append(actual.Spec.Volumes, corev1.Volume{Name: "azure-workload-identity-reserve-other"})
			require.NotEmpty(t, comparePodConfiguration(*desired, *actual, policy))
			// The same rules apply if the ignored values were also explicitly desired.
			ignore.Volumes = append(ignore.Volumes, "azure-workload-identity-reserve-other")
			require.Empty(t, comparePodConfiguration(*actual, *desired, newPodComparisonPolicy(ignore, false)))
		})
	}
}

func TestPodRecreationKeepsOperationalReasons(t *testing.T) {
	j := comparisonJenkins()
	j.Spec.Reconciliation.Ignore = v1alpha2.JenkinsReconciliationIgnore{Labels: []string{"managed"}, Annotations: []string{"managed"}, Env: []string{"MANAGED"}}
	actual := desiredPodForComparison(j)
	actual.Labels["managed"] = "mutated"
	actual.Annotations["managed"] = "mutated"
	r := New(configuration.Configuration{Jenkins: j}, jenkinsclient.JenkinsAPIConnectionSettings{})
	require.False(t, r.checkForPodRecreation(*actual, "same").HasMessages())
	require.True(t, r.checkForPodRecreation(*actual, "changed").HasMessages())
	j.Status.OperatorVersion = "old"
	require.True(t, r.checkForPodRecreation(*actual, "same").HasMessages())
	j.Status.OperatorVersion = version.Version
	j.Spec.Restore.RecoveryOnce = 1
	j.Status.RestoredBackup = 1
	require.True(t, r.checkForPodRecreation(*actual, "same").HasMessages())
	j.Spec.Restore.RecoveryOnce = 0
	j.Status.RestoredBackup = 0
	actual.Status.Phase = corev1.PodFailed
	require.True(t, r.checkForPodRecreation(*actual, "same").HasMessages())
	actual.Status.Phase = corev1.PodRunning
	actual.Spec.Containers[0].Image = "changed"
	require.True(t, r.checkForPodRecreation(*actual, "same").HasMessages())
	for _, msg := range r.checkForPodRecreation(*actual, "same").Verbose() {
		require.False(t, strings.Contains(msg, "desired"), "do not print env values")
	}
}

func TestVolumePrefixBoundaries(t *testing.T) {
	p := newPodComparisonPolicy(v1alpha2.JenkinsReconciliationIgnore{Volumes: []string{"fixed"}, VolumeNamePrefixes: []string{"azure-workload-identity-reserved-", "literal*", ""}}, false)
	for _, name := range []string{"fixed", "azure-workload-identity-reserved-first", "azure-workload-identity-reserved-second", "literal*token"} {
		require.True(t, p.ignoresVolume(name), name)
	}
	for _, name := range []string{"", "FIXED", "Azure-workload-identity-reserved-first", "azure-workload-identity-reserve-other", "literal-token", "unrelated"} {
		require.False(t, p.ignoresVolume(name), name)
	}
	expected := corev1.Container{VolumeMounts: []corev1.VolumeMount{{Name: "retained", MountPath: "/retained"}, {Name: "fixed", MountPath: "/desired"}}}
	actual := corev1.Container{VolumeMounts: []corev1.VolumeMount{{Name: "retained", MountPath: "/retained"}, {Name: "fixed", MountPath: "/injected"}}}
	require.True(t, p.mountsEqual(expected, actual))
	actual.VolumeMounts[0].MountPath = "/injected"
	require.False(t, p.mountsEqual(expected, actual), "mount paths are not ignore keys")
}

func TestPodDefaultModeRestartReason(t *testing.T) {
	j := comparisonJenkins()
	j.Spec.Master.Volumes = []corev1.Volume{{Name: "custom", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "original"}}}}}
	actual := desiredPodForComparison(j)
	var volume *corev1.Volume
	for i := range actual.Spec.Volumes {
		if actual.Spec.Volumes[i].Name == "custom" {
			volume = &actual.Spec.Volumes[i]
		}
	}
	require.NotNil(t, volume)
	mode := int32(0644)
	volume.ConfigMap.DefaultMode = &mode
	r := New(configuration.Configuration{Jenkins: j}, jenkinsclient.JenkinsAPIConnectionSettings{})
	require.False(t, r.checkForPodRecreation(*actual, "same").HasMessages())
	volume.ConfigMap.Name = "different"
	require.True(t, r.checkForPodRecreation(*actual, "same").HasMessages())
	volume.ConfigMap.Name = "original"
	mode = 0600
	require.True(t, r.checkForPodRecreation(*actual, "same").HasMessages())
}

func TestIgnoredEnvPreservesRetainedOrder(t *testing.T) {
	policy := newPodComparisonPolicy(v1alpha2.JenkinsReconciliationIgnore{Env: []string{"IGNORE"}}, false)
	first := corev1.EnvVar{Name: "FIRST", Value: "one"}
	second := corev1.EnvVar{Name: "SECOND", Value: "two"}
	expected := []corev1.EnvVar{first, {Name: "IGNORE", Value: "desired"}, second}
	require.True(t, policy.envEqual(expected, []corev1.EnvVar{{Name: "IGNORE", Value: "injected"}, first, second}))
	require.False(t, policy.envEqual(expected, []corev1.EnvVar{second, first}), "retained env order remains managed")
	require.False(t, policy.envEqual(expected, []corev1.EnvVar{first, second, {Name: "ignore", Value: "different-case"}}), "env name matching is case-sensitive")
}
