package base

import (
	"testing"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

func TestCompareVolumesDefaultMode(t *testing.T) {
	sources := map[string]corev1.VolumeSource{
		"configMap":   {ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "config"}}},
		"secret":      {Secret: &corev1.SecretVolumeSource{SecretName: "secret"}},
		"projected":   {Projected: &corev1.ProjectedVolumeSource{}},
		"downwardAPI": {DownwardAPI: &corev1.DownwardAPIVolumeSource{}},
	}
	for name, source := range sources {
		t.Run(name, func(t *testing.T) {
			jenkins := &v1alpha2.Jenkins{Spec: v1alpha2.JenkinsSpec{Master: v1alpha2.JenkinsMaster{Volumes: []corev1.Volume{{Name: "custom", VolumeSource: source}}}}}
			actual := corev1.Pod{Spec: corev1.PodSpec{ServiceAccountName: "identity-service-account", Volumes: append(resources.GetJenkinsMasterPodBaseVolumes(jenkins), *jenkins.Spec.Master.Volumes[0].DeepCopy())}}
			volume := &actual.Spec.Volumes[len(actual.Spec.Volumes)-1]
			mode := int32(0644)
			setTestVolumeMode(volume, &mode)
			original := jenkins.DeepCopy()
			originalPod := actual.DeepCopy()
			r := &JenkinsBaseConfigurationReconciler{Configuration: configuration.Configuration{Jenkins: jenkins}}
			require.True(t, r.compareVolumes(actual), "API-defaulted mode must not recreate the pod")
			require.True(t, r.compareVolumes(actual))
			require.Equal(t, original, jenkins)
			require.Equal(t, originalPod, &actual)
			mode = int32(0600)
			require.False(t, r.compareVolumes(actual), "an explicit different mode must still recreate the pod")
		})
	}
}

func setTestVolumeMode(v *corev1.Volume, mode *int32) {
	switch {
	case v.ConfigMap != nil:
		v.ConfigMap.DefaultMode = mode
	case v.Secret != nil:
		v.Secret.DefaultMode = mode
	case v.Projected != nil:
		v.Projected.DefaultMode = mode
	case v.DownwardAPI != nil:
		v.DownwardAPI.DefaultMode = mode
	}
}

func TestHomeVolumeComparison(t *testing.T) {
	j := &v1alpha2.Jenkins{Spec: v1alpha2.JenkinsSpec{Master: v1alpha2.JenkinsMaster{HomeVolume: &v1alpha2.JenkinsHomeVolume{Ephemeral: &corev1.PersistentVolumeClaimSpec{AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}, Resources: corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("2Gi")}}}}}}}
	actual := corev1.Pod{Spec: corev1.PodSpec{Volumes: resources.GetJenkinsMasterPodBaseVolumes(j)}}
	actual.Spec.Volumes[0].Ephemeral.VolumeClaimTemplate.Spec.VolumeMode = ptr.To(corev1.PersistentVolumeFilesystem)
	before, original := j.DeepCopy(), actual.DeepCopy()
	r := &JenkinsBaseConfigurationReconciler{Configuration: configuration.Configuration{Jenkins: j}}
	require.True(t, r.compareVolumes(actual), "API-defaulted filesystem mode must not recreate the Pod")
	require.Equal(t, before, j)
	require.Equal(t, original, &actual)
	equivalent := actual.DeepCopy()
	equivalent.Spec.Volumes[0].Ephemeral.VolumeClaimTemplate.Spec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse("2048Mi")
	require.True(t, r.compareVolumes(*equivalent), "equivalent storage quantities must not cause drift")
	for _, mutate := range []func(*corev1.Volume){
		func(v *corev1.Volume) { v.Ephemeral.VolumeClaimTemplate.Spec.StorageClassName = ptr.To("") },
		func(v *corev1.Volume) { v.Ephemeral.VolumeClaimTemplate.Spec.StorageClassName = ptr.To("fast") },
		func(v *corev1.Volume) {
			v.Ephemeral.VolumeClaimTemplate.Spec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse("3Gi")
		},
		func(v *corev1.Volume) {
			v.Ephemeral.VolumeClaimTemplate.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}
		},
		func(v *corev1.Volume) { v.Ephemeral = nil; v.EmptyDir = &corev1.EmptyDirVolumeSource{} },
	} {
		changed := actual.DeepCopy()
		mutate(&changed.Spec.Volumes[0])
		require.False(t, r.compareVolumes(*changed))
	}
	j.Spec.Reconciliation.Ignore.Volumes = []string{resources.JenkinsHomeVolumeName}
	changed := actual.DeepCopy()
	changed.Spec.Volumes[0].Ephemeral = nil
	changed.Spec.Volumes[0].EmptyDir = &corev1.EmptyDirVolumeSource{}
	require.True(t, r.compareVolumes(*changed), "explicit volume ignore rules still apply")
}
