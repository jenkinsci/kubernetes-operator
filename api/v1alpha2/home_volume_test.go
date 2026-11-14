package v1alpha2

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

func TestHomeVolumeValidation(t *testing.T) {
	claim := func() *corev1.PersistentVolumeClaimSpec {
		return &corev1.PersistentVolumeClaimSpec{AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}, Resources: corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("2Gi")}}}
	}
	for _, tt := range []struct {
		name    string
		home    *JenkinsHomeVolume
		invalid bool
	}{
		{"omitted", nil, false},
		{"emptyDir", &JenkinsHomeVolume{EmptyDir: &corev1.EmptyDirVolumeSource{}}, false},
		{"ephemeral", &JenkinsHomeVolume{Ephemeral: claim()}, false},
		{"no source", &JenkinsHomeVolume{}, true},
		{"both sources", &JenkinsHomeVolume{EmptyDir: &corev1.EmptyDirVolumeSource{}, Ephemeral: claim()}, true},
	} {
		t.Run(tt.name, func(t *testing.T) { require.Equal(t, tt.invalid, len(tt.home.Validate()) > 0) })
	}
	for _, tt := range []struct {
		name    string
		mutate  func(*corev1.PersistentVolumeClaimSpec)
		invalid bool
	}{
		{"missing storage", func(c *corev1.PersistentVolumeClaimSpec) { c.Resources.Requests = nil }, true},
		{"zero", func(c *corev1.PersistentVolumeClaimSpec) {
			c.Resources.Requests[corev1.ResourceStorage] = resource.MustParse("0")
		}, true},
		{"negative", func(c *corev1.PersistentVolumeClaimSpec) {
			c.Resources.Requests[corev1.ResourceStorage] = resource.MustParse("-1Gi")
		}, true},
		{"no access mode", func(c *corev1.PersistentVolumeClaimSpec) { c.AccessModes = nil }, true},
		{"invalid access mode", func(c *corev1.PersistentVolumeClaimSpec) {
			c.AccessModes = []corev1.PersistentVolumeAccessMode{"invalid"}
		}, true},
		{"block", func(c *corev1.PersistentVolumeClaimSpec) { c.VolumeMode = ptr.To(corev1.PersistentVolumeBlock) }, true},
		{"filesystem", func(c *corev1.PersistentVolumeClaimSpec) { c.VolumeMode = ptr.To(corev1.PersistentVolumeFilesystem) }, false},
		{"empty class", func(c *corev1.PersistentVolumeClaimSpec) { c.StorageClassName = ptr.To("") }, false},
		{"named class", func(c *corev1.PersistentVolumeClaimSpec) { c.StorageClassName = ptr.To("fast") }, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := claim()
			tt.mutate(c)
			home := &JenkinsHomeVolume{Ephemeral: c}
			require.Equal(t, tt.invalid, len(home.Validate()) > 0)
			j := &Jenkins{Spec: JenkinsSpec{Master: JenkinsMaster{HomeVolume: home}}}
			_, err := j.ValidateCreate()
			require.Equal(t, tt.invalid, err != nil)
			_, err = j.ValidateUpdate(&Jenkins{})
			require.Equal(t, tt.invalid, err != nil)
		})
	}
}

func TestHomeVolumeDeepCopy(t *testing.T) {
	j := &Jenkins{Spec: JenkinsSpec{Master: JenkinsMaster{HomeVolume: &JenkinsHomeVolume{Ephemeral: &corev1.PersistentVolumeClaimSpec{StorageClassName: ptr.To("fast"), AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}, Resources: corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("2Gi")}}}}}}}
	copy := j.DeepCopy()
	*copy.Spec.Master.HomeVolume.Ephemeral.StorageClassName = "other"
	copy.Spec.Master.HomeVolume.Ephemeral.AccessModes[0] = corev1.ReadWriteMany
	copy.Spec.Master.HomeVolume.Ephemeral.Resources.Requests[corev1.ResourceStorage] = resource.MustParse("4Gi")
	require.Equal(t, "fast", *j.Spec.Master.HomeVolume.Ephemeral.StorageClassName)
	require.Equal(t, corev1.ReadWriteOnce, j.Spec.Master.HomeVolume.Ephemeral.AccessModes[0])
	require.Equal(t, resource.MustParse("2Gi"), j.Spec.Master.HomeVolume.Ephemeral.Resources.Requests[corev1.ResourceStorage])
}
