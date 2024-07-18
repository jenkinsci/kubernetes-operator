package resources

import (
	"fmt"
	"strings"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/constants"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// JenkinsMasterContainerName is the Jenkins master container name in pod
	JenkinsMasterContainerName = "jenkins-master"
	// JenkinsHomeVolumeName is the Jenkins home volume name
	JenkinsHomeVolumeName    = "jenkins-home"
	jenkinsPath              = "/var/jenkins"
	httpGetPath              = "/login"
	jenkinsScriptsVolumeName = "scripts"
	// JenkinsScriptsVolumePath is a path where are scripts used to configure Jenkins
	JenkinsScriptsVolumePath = jenkinsPath + "/scripts"
	// InitScriptName is the init script name which configures init.groovy.d, scripts and install plugins
	InitScriptName = "init.sh"

	jenkinsOperatorCredentialsVolumeName = "operator-credentials"
	jenkinsOperatorCredentialsVolumePath = jenkinsPath + "/operator-credentials"

	jenkinsInitConfigurationVolumeName = "init-configuration"
	jenkinsInitConfigurationVolumePath = jenkinsPath + "/init-configuration"

	// GroovyScriptsSecretVolumePath is a path where are groovy scripts used to configure Jenkins
	// This script is provided by user
	GroovyScriptsSecretVolumePath = jenkinsPath + "/groovy-scripts-secrets"
	// ConfigurationAsCodeSecretVolumePath is a path where are CasC configs used to configure Jenkins
	// This script is provided by user
	ConfigurationAsCodeSecretVolumePath = jenkinsPath + "/configuration-as-code-secrets"

	httpPortName  = "http"
	slavePortName = "slavelistener"
)

func buildPodTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		Kind:       "Pod",
		APIVersion: "v1",
	}
}

// GetJenkinsMasterContainerBaseCommand returns default Jenkins master container command
func GetJenkinsMasterContainerBaseCommand() []string {
	return []string{
		"bash",
		"-c",
		fmt.Sprintf("%s/%s && exec /usr/bin/tini -s -- /usr/local/bin/jenkins.sh",
			JenkinsScriptsVolumePath, InitScriptName),
	}
}

// GetJenkinsMasterContainerBaseEnvs returns Jenkins master pod envs required by operator
func GetJenkinsMasterContainerBaseEnvs(jenkins *v1alpha2.Jenkins) []corev1.EnvVar {
	envVars := []corev1.EnvVar{
		{
			Name:  "COPY_REFERENCE_FILE_LOG",
			Value: fmt.Sprintf("%s/%s", getJenkinsHomePath(jenkins), "copy_reference_file.log"),
		},
	}

	if len(jenkins.Spec.ConfigurationAsCode.Secret.Name) > 0 {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "SECRETS", // https://github.com/jenkinsci/configuration-as-code-plugin/blob/master/demos/kubernetes-secrets/README.md
			Value: ConfigurationAsCodeSecretVolumePath,
		})
	}

	return envVars
}

// getJenkinsHomePath fetches the Home Path for Jenkins
func getJenkinsHomePath(jenkins *v1alpha2.Jenkins) string {
	defaultJenkinsHomePath := "/var/lib/jenkins"
	for _, envVar := range jenkins.Spec.Master.Containers[0].Env {
		if envVar.Name == "JENKINS_HOME" {
			return envVar.Value
		}
	}
	return defaultJenkinsHomePath
}

// GetJenkinsMasterPodBaseVolumes returns Jenkins master pod volumes required by operator
func GetJenkinsMasterPodBaseVolumes(jenkins *v1alpha2.Jenkins) []corev1.Volume {
	configMapVolumeSourceDefaultMode := corev1.ConfigMapVolumeSourceDefaultMode
	secretVolumeSourceDefaultMode := corev1.SecretVolumeSourceDefaultMode
	var scriptsVolumeDefaultMode int32 = 0777
	volumes := []corev1.Volume{
		{
			Name: JenkinsHomeVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: jenkinsScriptsVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					DefaultMode: &scriptsVolumeDefaultMode,
					LocalObjectReference: corev1.LocalObjectReference{
						Name: getScriptsConfigMapName(jenkins),
					},
				},
			},
		},
		{
			Name: jenkinsInitConfigurationVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					DefaultMode: &configMapVolumeSourceDefaultMode,
					LocalObjectReference: corev1.LocalObjectReference{
						Name: GetInitConfigurationConfigMapName(jenkins),
					},
				},
			},
		},
		{
			Name: jenkinsOperatorCredentialsVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					DefaultMode: &secretVolumeSourceDefaultMode,
					SecretName:  GetOperatorCredentialsSecretName(jenkins),
				},
			},
		},
	}

	if len(jenkins.Spec.GroovyScripts.Secret.Name) > 0 {
		volumes = append(volumes, corev1.Volume{
			Name: getGroovyScriptsSecretVolumeName(jenkins),
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					DefaultMode: &secretVolumeSourceDefaultMode,
					SecretName:  jenkins.Spec.GroovyScripts.Secret.Name,
				},
			},
		})
	}
	if len(jenkins.Spec.ConfigurationAsCode.Secret.Name) > 0 {
		volumes = append(volumes, corev1.Volume{
			Name: getConfigurationAsCodeSecretVolumeName(jenkins),
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					DefaultMode: &secretVolumeSourceDefaultMode,
					SecretName:  jenkins.Spec.ConfigurationAsCode.Secret.Name,
				},
			},
		})
	}

	return volumes
}

func getGroovyScriptsSecretVolumeName(jenkins *v1alpha2.Jenkins) string {
	return "gs-" + jenkins.Spec.GroovyScripts.Secret.Name
}

func getConfigurationAsCodeSecretVolumeName(jenkins *v1alpha2.Jenkins) string {
	return "casc-" + jenkins.Spec.ConfigurationAsCode.Secret.Name
}

// GetJenkinsMasterContainerBaseVolumeMounts returns Jenkins master pod volume mounts required by operator
func GetJenkinsMasterContainerBaseVolumeMounts(jenkins *v1alpha2.Jenkins) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      JenkinsHomeVolumeName,
			MountPath: getJenkinsHomePath(jenkins),
			ReadOnly:  false,
		},
		{
			Name:      jenkinsScriptsVolumeName,
			MountPath: JenkinsScriptsVolumePath,
			ReadOnly:  true,
		},
		{
			Name:      jenkinsInitConfigurationVolumeName,
			MountPath: jenkinsInitConfigurationVolumePath,
			ReadOnly:  true,
		},
		{
			Name:      jenkinsOperatorCredentialsVolumeName,
			MountPath: jenkinsOperatorCredentialsVolumePath,
			ReadOnly:  true,
		},
	}

	if len(jenkins.Spec.GroovyScripts.Secret.Name) > 0 {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      getGroovyScriptsSecretVolumeName(jenkins),
			MountPath: GroovyScriptsSecretVolumePath,
			ReadOnly:  true,
		})
	}
	if len(jenkins.Spec.ConfigurationAsCode.Secret.Name) > 0 {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      getConfigurationAsCodeSecretVolumeName(jenkins),
			MountPath: ConfigurationAsCodeSecretVolumePath,
			ReadOnly:  true,
		})
	}

	return volumeMounts
}

// NewJenkinsMasterContainer returns Jenkins master Kubernetes container
func NewJenkinsMasterContainer(jenkins *v1alpha2.Jenkins) corev1.Container {
	jenkinsContainer := jenkins.Spec.Master.Containers[0]

	envs := GetJenkinsMasterContainerBaseEnvs(jenkins)
	envs = append(envs, jenkinsContainer.Env...)

	jenkinsHomeEnvVar := corev1.EnvVar{
		Name:  "JENKINS_HOME",
		Value: getJenkinsHomePath(jenkins),
	}

	jenkinsHomeEnvVarExists := false
	for _, env := range jenkinsContainer.Env {
		if env.Name == jenkinsHomeEnvVar.Name {
			jenkinsHomeEnvVarExists = true
			break
		}
	}

	if !jenkinsHomeEnvVarExists {
		envs = append(envs, jenkinsHomeEnvVar)
	}

	if jenkins.Spec.Master.Containers[0].ReadinessProbe.HTTPGet != nil {
		setLivenessAndReadinessPath(jenkins)
	}

	return corev1.Container{
		Name:            JenkinsMasterContainerName,
		Image:           jenkinsContainer.Image,
		ImagePullPolicy: jenkinsContainer.ImagePullPolicy,
		Command:         jenkinsContainer.Command,
		LivenessProbe:   jenkinsContainer.LivenessProbe,
		ReadinessProbe:  jenkinsContainer.ReadinessProbe,
		Ports: []corev1.ContainerPort{
			{
				Name:          httpPortName,
				ContainerPort: constants.DefaultHTTPPortInt32,
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          slavePortName,
				ContainerPort: constants.DefaultSlavePortInt32,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		SecurityContext: jenkinsContainer.SecurityContext,
		Env:             envs,
		EnvFrom:         jenkinsContainer.EnvFrom,
		Resources:       jenkinsContainer.Resources,
		VolumeMounts:    append(GetJenkinsMasterContainerBaseVolumeMounts(jenkins), jenkinsContainer.VolumeMounts...),
	}
}

func setLivenessAndReadinessPath(jenkins *v1alpha2.Jenkins) {
	ReadinessProbePath := jenkins.Spec.Master.Containers[0].ReadinessProbe.HTTPGet.Path
	LivenessProbePath := jenkins.Spec.Master.Containers[0].ReadinessProbe.HTTPGet.Path

	if prefix, ok := GetJenkinsOpts(*jenkins)["prefix"]; ok {
		if !strings.HasPrefix(ReadinessProbePath, prefix) {
			jenkins.Spec.Master.Containers[0].ReadinessProbe.HTTPGet.Path = prefix + httpGetPath
		}
		if !strings.HasPrefix(LivenessProbePath, prefix) {
			jenkins.Spec.Master.Containers[0].LivenessProbe.HTTPGet.Path = prefix + httpGetPath
		}
	} else {
		if ReadinessProbePath != httpGetPath {
			jenkins.Spec.Master.Containers[0].ReadinessProbe.HTTPGet.Path = httpGetPath
		}
		if LivenessProbePath != httpGetPath {
			jenkins.Spec.Master.Containers[0].LivenessProbe.HTTPGet.Path = httpGetPath
		}
	}
}

// GetJenkinsOpts gets JENKINS_OPTS env parameter, parses it's values and returns it as a map`
func GetJenkinsOpts(jenkins v1alpha2.Jenkins) map[string]string {
	envs := jenkins.Spec.Master.Containers[0].Env
	jenkinsOpts := make(map[string]string)

	for key, value := range envs {
		if value.Name == "JENKINS_OPTS" {
			jenkinsOptsEnv := envs[key]
			jenkinsOptsWithDashes := jenkinsOptsEnv.Value
			if len(jenkinsOptsWithDashes) == 0 {
				return nil
			}

			jenkinsOptsWithEqOperators := strings.Split(jenkinsOptsWithDashes, " ")

			for _, vx := range jenkinsOptsWithEqOperators {
				opt := strings.Split(vx, "=")
				jenkinsOpts[strings.ReplaceAll(opt[0], "--", "")] = opt[1]
			}

			return jenkinsOpts
		}
	}
	return nil
}

// ConvertJenkinsContainerToKubernetesContainer converts Jenkins container to Kubernetes container
func ConvertJenkinsContainerToKubernetesContainer(container v1alpha2.Container) corev1.Container {
	return corev1.Container{
		Name:            container.Name,
		Image:           container.Image,
		Command:         container.Command,
		Args:            container.Args,
		WorkingDir:      container.WorkingDir,
		Ports:           container.Ports,
		EnvFrom:         container.EnvFrom,
		Env:             container.Env,
		Resources:       container.Resources,
		VolumeMounts:    container.VolumeMounts,
		LivenessProbe:   container.LivenessProbe,
		ReadinessProbe:  container.ReadinessProbe,
		Lifecycle:       container.Lifecycle,
		ImagePullPolicy: container.ImagePullPolicy,
		SecurityContext: container.SecurityContext,
	}
}

func newContainers(jenkins *v1alpha2.Jenkins) (containers []corev1.Container) {
	containers = append(containers, NewJenkinsMasterContainer(jenkins))

	for _, container := range jenkins.Spec.Master.Containers[1:] {
		containers = append(containers, ConvertJenkinsContainerToKubernetesContainer(container))
	}

	return
}

// GetJenkinsMasterPodName returns Jenkins pod name for given CR
func GetJenkinsMasterPodName(jenkins *v1alpha2.Jenkins) string {
	return fmt.Sprintf("jenkins-%s", jenkins.Name)
}

// GetJenkinsMasterPodLabels returns Jenkins pod labels for given CR
func GetJenkinsMasterPodLabels(jenkins v1alpha2.Jenkins) map[string]string {
	var labels map[string]string
	if jenkins.Spec.Master.Labels == nil {
		labels = map[string]string{}
	} else {
		labels = jenkins.Spec.Master.Labels
	}
	for key, value := range BuildResourceLabels(&jenkins) {
		labels[key] = value
	}
	return labels
}

// NewJenkinsMasterPod builds Jenkins Master Kubernetes Pod resource
func NewJenkinsMasterPod(objectMeta metav1.ObjectMeta, jenkins *v1alpha2.Jenkins) *corev1.Pod {
	serviceAccountName := objectMeta.Name
	objectMeta.Annotations = jenkins.Spec.Master.Annotations
	objectMeta.Name = GetJenkinsMasterPodName(jenkins)
	objectMeta.Labels = GetJenkinsMasterPodLabels(*jenkins)

	if jenkins.Spec.Master.TerminationGracePeriodSeconds == nil {
		defaultGracePeriod := constants.DefaultTerminationGracePeriodSeconds
		jenkins.Spec.Master.TerminationGracePeriodSeconds = &defaultGracePeriod
	}

	return &corev1.Pod{
		TypeMeta:   buildPodTypeMeta(),
		ObjectMeta: objectMeta,
		Spec: corev1.PodSpec{
			ServiceAccountName:            serviceAccountName,
			RestartPolicy:                 corev1.RestartPolicyNever,
			NodeSelector:                  jenkins.Spec.Master.NodeSelector,
			Containers:                    newContainers(jenkins),
			Volumes:                       append(GetJenkinsMasterPodBaseVolumes(jenkins), jenkins.Spec.Master.Volumes...),
			SecurityContext:               jenkins.Spec.Master.SecurityContext,
			ImagePullSecrets:              jenkins.Spec.Master.ImagePullSecrets,
			Tolerations:                   jenkins.Spec.Master.Tolerations,
			PriorityClassName:             jenkins.Spec.Master.PriorityClassName,
			HostAliases:                   jenkins.Spec.Master.HostAliases,
			TerminationGracePeriodSeconds: jenkins.Spec.Master.TerminationGracePeriodSeconds,
		},
	}
}
