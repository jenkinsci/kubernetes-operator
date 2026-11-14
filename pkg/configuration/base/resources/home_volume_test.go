package resources

import (
	"testing"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

func TestJenkinsHomeVolume(t *testing.T) {
	for _, class := range []*string{nil, ptr.To(""), ptr.To("fast")} {
		j := jenkins.DeepCopy()
		j.Spec.Master.HomeVolume = &v1alpha2.JenkinsHomeVolume{Ephemeral: &corev1.PersistentVolumeClaimSpec{StorageClassName: class, AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}, Resources: corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("2Gi")}}}}
		before := j.DeepCopy()
		pod := NewJenkinsMasterPod(NewResourceObjectMeta(j), j)
		deployment := NewJenkinsDeployment(NewResourceObjectMeta(j), j)
		for _, volumes := range [][]corev1.Volume{pod.Spec.Volumes, deployment.Spec.Template.Spec.Volumes} {
			require.Equal(t, JenkinsHomeVolumeName, volumes[0].Name)
			require.Nil(t, volumes[0].EmptyDir)
			require.Equal(t, before.Spec.Master.HomeVolume.Ephemeral, &volumes[0].Ephemeral.VolumeClaimTemplate.Spec)
			volumes[0].Ephemeral.VolumeClaimTemplate.Spec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse("3Gi")
		}
		require.Equal(t, before.Spec.Master.HomeVolume, j.Spec.Master.HomeVolume)
	}
	for _, home := range []*v1alpha2.JenkinsHomeVolume{nil, {EmptyDir: &corev1.EmptyDirVolumeSource{}}, {EmptyDir: &corev1.EmptyDirVolumeSource{Medium: corev1.StorageMediumMemory, SizeLimit: ptr.To(resource.MustParse("1Gi"))}}} {
		j := &v1alpha2.Jenkins{Spec: v1alpha2.JenkinsSpec{Master: v1alpha2.JenkinsMaster{HomeVolume: home}}}
		volume := GetJenkinsMasterPodBaseVolumes(j)[0]
		want := &corev1.EmptyDirVolumeSource{}
		if home != nil {
			want = home.EmptyDir
		}
		require.Equal(t, want, volume.EmptyDir)
		require.Nil(t, volume.Ephemeral)
	}
}
