package resources

import (
	"fmt"

	jenkinsv1alpha2 "github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	NameWithSuffixFormat         = "%s-%s"
	PluginDefinitionFormat       = "%s:%s"
	BuilderDockerfileArg         = "--dockerfile=/workspace/dockerfile/Dockerfile"
	BuilderContextDirArg         = "--context=dir://workspace/"
	BuilderPushArg               = "--no-push"
	BuilderDigestFileArg         = "--digest-file=/dev/termination-log"
	BuilderSuffix                = "builder"
	DockerfileStorageSuffix      = "dockerfile-storage"
	DockerfileNameSuffix         = "dockerfile"
	JenkinsImageBuilderImage     = "gcr.io/kaniko-project/executor:latest"
	JenkinsImageBuilderName      = "jenkins-image-builder"
	JenkinsImageDefaultBaseImage = "jenkins/jenkins:lts"
	DockerfileName               = "Dockerfile"
	DockerfileTemplate           = `FROM %s
RUN curl -o /tmp/install-plugins.sh https://raw.githubusercontent.com/jenkinsci/docker/master/install-plugins.sh
RUN chmod +x /tmp/install-plugins.sh
RUN install-plugins.sh %s `
)

var log = logf.Log.WithName("controller_jenkinsimage")

// NewBuilderPod returns a busybox pod with the same name/namespace as the cr.
func NewBuilderPod(cr *jenkinsv1alpha2.JenkinsImage) *corev1.Pod {
	name := fmt.Sprintf(NameWithSuffixFormat, cr.Name, BuilderSuffix)
	args := []string{BuilderDockerfileArg, BuilderContextDirArg, BuilderPushArg, BuilderDigestFileArg}
	volumes := getVolumes(cr)
	volumeMounts := getVolumesMounts(cr)
	p := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:         JenkinsImageBuilderName,
					Image:        JenkinsImageBuilderImage,
					Args:         args,
					VolumeMounts: volumeMounts,
				},
			},
			Volumes: volumes,
		},
	}
	return p
}

// NewDockerfileConfigMap returns a busybox pod with the same name/namespace as the cr.
func NewDockerfileConfigMap(cr *jenkinsv1alpha2.JenkinsImage) *corev1.ConfigMap {
	dockerfileContent := fmt.Sprintf(DockerfileTemplate, getDefaultedBaseImage(cr), getPluginsList(cr))
	name := fmt.Sprintf(NameWithSuffixFormat, cr.Name, DockerfileNameSuffix)
	data := map[string]string{DockerfileName: dockerfileContent}
	dockerfile := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
		},
		Data: data,
	}
	return dockerfile
}

func getPluginsList(cr *jenkinsv1alpha2.JenkinsImage) string {
	logger := log.WithName("jenkinsimage_getPluginsList")
	plugins := ""
	for _, v := range cr.Spec.Plugins {
		plugins += fmt.Sprintf(PluginDefinitionFormat, v.Name, v.Version) + " "
		logger.Info(fmt.Sprintf("Adding plugin %s:%s ", v.Name, v.Version))
	}
	return plugins
}

func getDefaultedBaseImage(cr *jenkinsv1alpha2.JenkinsImage) string {
	if len(cr.Spec.BaseImage.Name) != 0 {
		return cr.Spec.BaseImage.Name
	}
	return JenkinsImageDefaultBaseImage
}

func getVolumes(cr *jenkinsv1alpha2.JenkinsImage) []corev1.Volume {
	name := fmt.Sprintf(NameWithSuffixFormat, cr.Name, DockerfileStorageSuffix)
	storage := corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}

	name = fmt.Sprintf(NameWithSuffixFormat, cr.Name, DockerfileNameSuffix)
	config := corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: name},
			},
		},
	}
	volumes := []corev1.Volume{storage, config}
	return volumes
}

func getVolumesMounts(cr *jenkinsv1alpha2.JenkinsImage) []corev1.VolumeMount {
	name := fmt.Sprintf(NameWithSuffixFormat, cr.Name, DockerfileStorageSuffix)
	storage := corev1.VolumeMount{
		Name:      name,
		MountPath: "/workspace",
	}
	name = fmt.Sprintf(NameWithSuffixFormat, cr.Name, DockerfileNameSuffix)
	config := corev1.VolumeMount{
		Name:      name,
		MountPath: "/workspace/dockerfile",
	}
	volumeMounts := []corev1.VolumeMount{storage, config}
	return volumeMounts
}
