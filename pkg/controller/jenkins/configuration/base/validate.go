package base

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/constants"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/plugins"

	docker "github.com/docker/distribution/reference"
	stackerr "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

var (
	dockerImageRegexp = regexp.MustCompile(`^` + docker.TagRegexp.String() + `$`)
)

// Validate validates Jenkins CR Spec.master section
func (r *ReconcileJenkinsBaseConfiguration) Validate(jenkins *v1alpha2.Jenkins) ([]string, error) {
	var messages []string

	if msg := r.validateReservedVolumes(); len(msg) > 0 {
		messages = append(messages, msg...)
	}

	if msg, err := r.validateVolumes(); err != nil {
		return nil, err
	} else if len(msg) > 0 {
		messages = append(messages, msg...)
	}

	for _, container := range jenkins.Spec.Master.Containers {
		if msg := r.validateContainer(container); len(msg) > 0 {
			for _, m := range msg {
				messages = append(messages, fmt.Sprintf("Container `%s` - %s", container.Name, m))
			}
		}
	}

	if msg := r.validatePlugins(plugins.BasePlugins(), jenkins.Spec.Master.BasePlugins, jenkins.Spec.Master.Plugins); len(msg) > 0 {
		messages = append(messages, msg...)
	}

	if msg := r.validateJenkinsMasterPodEnvs(); len(msg) > 0 {
		messages = append(messages, msg...)
	}

	if msg, err := r.validateCustomization(r.Configuration.Jenkins.Spec.GroovyScripts.Customization, "spec.groovyScripts"); err != nil {
		return nil, err
	} else if len(msg) > 0 {
		messages = append(messages, msg...)
	}
	if msg, err := r.validateCustomization(r.Configuration.Jenkins.Spec.ConfigurationAsCode.Customization, "spec.configurationAsCode"); err != nil {
		return nil, err
	} else if len(msg) > 0 {
		messages = append(messages, msg...)
	}

	if jenkins.Spec.JenkinsAPISettings.AuthorizationStrategy != v1alpha2.CreateUserAuthorizationStrategy && jenkins.Spec.JenkinsAPISettings.AuthorizationStrategy != v1alpha2.ServiceAccountAuthorizationStrategy {
		messages = append(messages, fmt.Sprintf("unrecognized '%s' spec.jenkinsAPISettings.authorizationStrategy", jenkins.Spec.JenkinsAPISettings.AuthorizationStrategy))
	}

	return messages, nil
}

func (r *ReconcileJenkinsBaseConfiguration) validateJenkinsMasterContainerCommand() []string {
	masterContainer := r.Configuration.GetJenkinsMasterContainer()
	if masterContainer == nil {
		return []string{}
	}

	jenkinsOperatorInitScript := fmt.Sprintf("%s/%s && ", resources.JenkinsScriptsVolumePath, resources.InitScriptName)
	correctCommand := []string{
		"bash",
		"-c",
		fmt.Sprintf("%s<optional-custom-command> && exec <command-which-start-jenkins>", jenkinsOperatorInitScript),
	}
	invalidCommandMessage := []string{fmt.Sprintf("spec.master.containers[%s].command is invalid, make sure it looks like '%v', otherwise the operator won't configure default user and install plugins. 'exec' is required to propagate signals to the Jenkins.", masterContainer.Name, correctCommand)}
	if len(masterContainer.Command) != 3 {
		return invalidCommandMessage
	}
	if masterContainer.Command[0] != correctCommand[0] {
		return invalidCommandMessage
	}
	if masterContainer.Command[1] != correctCommand[1] {
		return invalidCommandMessage
	}
	if !strings.HasPrefix(masterContainer.Command[2], jenkinsOperatorInitScript) {
		return invalidCommandMessage
	}
	if !strings.Contains(masterContainer.Command[2], "exec") {
		return invalidCommandMessage
	}

	return []string{}
}

func (r *ReconcileJenkinsBaseConfiguration) validateImagePullSecrets() ([]string, error) {
	var messages []string
	for _, sr := range r.Configuration.Jenkins.Spec.Master.ImagePullSecrets {
		msg, err := r.validateImagePullSecret(sr.Name)
		if err != nil {
			return nil, err
		}
		if len(msg) > 0 {
			messages = append(messages, msg...)
		}
	}
	return messages, nil
}

func (r *ReconcileJenkinsBaseConfiguration) validateImagePullSecret(secretName string) ([]string, error) {
	var messages []string
	secret := &corev1.Secret{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: secretName, Namespace: r.Configuration.Jenkins.ObjectMeta.Namespace}, secret)
	if err != nil && apierrors.IsNotFound(err) {
		messages = append(messages, fmt.Sprintf("Secret %s not found defined in spec.master.imagePullSecrets", secretName))
	} else if err != nil && !apierrors.IsNotFound(err) {
		return nil, stackerr.WithStack(err)
	}

	if secret.Data["docker-server"] == nil {
		messages = append(messages, fmt.Sprintf("Secret '%s' defined in spec.master.imagePullSecrets doesn't have 'docker-server' key.", secretName))
	}
	if secret.Data["docker-username"] == nil {
		messages = append(messages, fmt.Sprintf("Secret '%s' defined in spec.master.imagePullSecrets doesn't have 'docker-username' key.", secretName))
	}
	if secret.Data["docker-password"] == nil {
		messages = append(messages, fmt.Sprintf("Secret '%s' defined in spec.master.imagePullSecrets doesn't have 'docker-password' key.", secretName))
	}
	if secret.Data["docker-email"] == nil {
		messages = append(messages, fmt.Sprintf("Secret '%s' defined in spec.master.imagePullSecrets doesn't have 'docker-email' key.", secretName))
	}

	return messages, nil
}

func (r *ReconcileJenkinsBaseConfiguration) validateVolumes() ([]string, error) {
	var messages []string
	for _, volume := range r.Configuration.Jenkins.Spec.Master.Volumes {
		switch {
		case volume.ConfigMap != nil:
			if msg, err := r.validateConfigMapVolume(volume); err != nil {
				return nil, err
			} else if len(msg) > 0 {
				messages = append(messages, msg...)
			}
		case volume.Secret != nil:
			if msg, err := r.validateSecretVolume(volume); err != nil {
				return nil, err
			} else if len(msg) > 0 {
				messages = append(messages, msg...)
			}
		case volume.PersistentVolumeClaim != nil:
			if msg, err := r.validatePersistentVolumeClaim(volume); err != nil {
				return nil, err
			} else if len(msg) > 0 {
				messages = append(messages, msg...)
			}
		}
	}

	return messages, nil
}

func (r *ReconcileJenkinsBaseConfiguration) validatePersistentVolumeClaim(volume corev1.Volume) ([]string, error) {
	var messages []string

	pvc := &corev1.PersistentVolumeClaim{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: volume.PersistentVolumeClaim.ClaimName, Namespace: r.Configuration.Jenkins.ObjectMeta.Namespace}, pvc)
	if err != nil && apierrors.IsNotFound(err) {
		messages = append(messages, fmt.Sprintf("PersistentVolumeClaim '%s' not found for volume '%v'", volume.PersistentVolumeClaim.ClaimName, volume))
	} else if err != nil && !apierrors.IsNotFound(err) {
		return nil, stackerr.WithStack(err)
	}

	return messages, nil
}

func (r *ReconcileJenkinsBaseConfiguration) validateConfigMapVolume(volume corev1.Volume) ([]string, error) {
	var messages []string
	if volume.ConfigMap.Optional != nil && *volume.ConfigMap.Optional {
		return nil, nil
	}

	configMap := &corev1.ConfigMap{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: volume.ConfigMap.Name, Namespace: r.Configuration.Jenkins.ObjectMeta.Namespace}, configMap)
	if err != nil && apierrors.IsNotFound(err) {
		messages = append(messages, fmt.Sprintf("ConfigMap '%s' not found for volume '%v'", volume.ConfigMap.Name, volume))
	} else if err != nil && !apierrors.IsNotFound(err) {
		return nil, stackerr.WithStack(err)
	}

	return messages, nil
}

func (r *ReconcileJenkinsBaseConfiguration) validateSecretVolume(volume corev1.Volume) ([]string, error) {
	var messages []string
	if volume.Secret.Optional != nil && *volume.Secret.Optional {
		return nil, nil
	}

	secret := &corev1.Secret{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: volume.Secret.SecretName, Namespace: r.Configuration.Jenkins.ObjectMeta.Namespace}, secret)
	if err != nil && apierrors.IsNotFound(err) {
		messages = append(messages, fmt.Sprintf("Secret '%s' not found for volume '%v'", volume.Secret.SecretName, volume))
	} else if err != nil && !apierrors.IsNotFound(err) {
		return nil, stackerr.WithStack(err)
	}

	return messages, nil
}

func (r *ReconcileJenkinsBaseConfiguration) validateReservedVolumes() []string {
	var messages []string

	for _, baseVolume := range resources.GetJenkinsMasterPodBaseVolumes(r.Configuration.Jenkins) {
		for _, volume := range r.Configuration.Jenkins.Spec.Master.Volumes {
			if baseVolume.Name == volume.Name {
				messages = append(messages, fmt.Sprintf("Jenkins Master pod volume '%s' is reserved please choose different one", volume.Name))
			}
		}
	}

	return messages
}

func (r *ReconcileJenkinsBaseConfiguration) validateContainer(container v1alpha2.Container) []string {
	var messages []string
	if container.Image == "" {
		messages = append(messages, "Image not set")
	}

	if !dockerImageRegexp.MatchString(container.Image) && !docker.ReferenceRegexp.MatchString(container.Image) {
		messages = append(messages, "Invalid image")
	}

	if container.ImagePullPolicy == "" {
		messages = append(messages, "Image pull policy not set")
	}

	if msg := r.validateContainerVolumeMounts(container); len(msg) > 0 {
		messages = append(messages, msg...)
	}

	return messages
}

func (r *ReconcileJenkinsBaseConfiguration) validateContainerVolumeMounts(container v1alpha2.Container) []string {
	var messages []string
	allVolumes := append(resources.GetJenkinsMasterPodBaseVolumes(r.Configuration.Jenkins), r.Configuration.Jenkins.Spec.Master.Volumes...)

	for _, volumeMount := range container.VolumeMounts {
		if len(volumeMount.MountPath) == 0 {
			messages = append(messages, fmt.Sprintf("mountPath not set for '%s' volume mount in container '%s'", volumeMount.Name, container.Name))
		}

		foundVolume := false
		for _, volume := range allVolumes {
			if volumeMount.Name == volume.Name {
				foundVolume = true
			}
		}

		if !foundVolume {
			messages = append(messages, fmt.Sprintf("Not found volume for '%s' volume mount in container '%s'", volumeMount.Name, container.Name))
		}
	}

	return messages
}

func (r *ReconcileJenkinsBaseConfiguration) validateJenkinsMasterPodEnvs() []string {
	var messages []string
	baseEnvs := resources.GetJenkinsMasterContainerBaseEnvs(r.Configuration.Jenkins)
	baseEnvNames := map[string]string{}
	for _, env := range baseEnvs {
		baseEnvNames[env.Name] = env.Value
	}

	javaOpts := corev1.EnvVar{}
	for _, userEnv := range r.Configuration.Jenkins.Spec.Master.Containers[0].Env {
		if userEnv.Name == constants.JavaOpsVariableName {
			javaOpts = userEnv
		}
		if _, overriding := baseEnvNames[userEnv.Name]; overriding {
			messages = append(messages, fmt.Sprintf("Jenkins Master container env '%s' cannot be overridden", userEnv.Name))
		}
	}

	requiredFlags := map[string]bool{
		"-Djenkins.install.runSetupWizard=false": false,
		"-Djava.awt.headless=true":               false,
	}
	for _, setFlag := range strings.Split(javaOpts.Value, " ") {
		for requiredFlag := range requiredFlags {
			if setFlag == requiredFlag {
				requiredFlags[requiredFlag] = true
				break
			}
		}
	}
	for requiredFlag, set := range requiredFlags {
		if !set {
			messages = append(messages, fmt.Sprintf("Jenkins Master container env '%s' doesn't have required flag '%s'", constants.JavaOpsVariableName, requiredFlag))
		}
	}

	return messages
}

func (r *ReconcileJenkinsBaseConfiguration) validatePlugins(requiredBasePlugins []plugins.Plugin, basePlugins, userPlugins []v1alpha2.Plugin) []string {
	var messages []string
	allPlugins := map[plugins.Plugin][]plugins.Plugin{}

	for _, jenkinsPlugin := range basePlugins {
		plugin, err := plugins.NewPlugin(jenkinsPlugin.Name, jenkinsPlugin.Version, jenkinsPlugin.DownloadURL)
		if err != nil {
			messages = append(messages, err.Error())
		}

		if plugin != nil {
			allPlugins[*plugin] = []plugins.Plugin{}
		}
	}

	for _, jenkinsPlugin := range userPlugins {
		plugin, err := plugins.NewPlugin(jenkinsPlugin.Name, jenkinsPlugin.Version, jenkinsPlugin.DownloadURL)
		if err != nil {
			messages = append(messages, err.Error())
		}

		if plugin != nil {
			allPlugins[*plugin] = []plugins.Plugin{}
		}
	}

	if msg := plugins.VerifyDependencies(allPlugins); len(msg) > 0 {
		messages = append(messages, msg...)
	}

	if msg := r.verifyBasePlugins(requiredBasePlugins, basePlugins); len(msg) > 0 {
		messages = append(messages, msg...)
	}

	return messages
}

func (r *ReconcileJenkinsBaseConfiguration) verifyBasePlugins(requiredBasePlugins []plugins.Plugin, basePlugins []v1alpha2.Plugin) []string {
	var messages []string

	for _, requiredBasePlugin := range requiredBasePlugins {
		found := false
		for _, basePlugin := range basePlugins {
			if requiredBasePlugin.Name == basePlugin.Name {
				found = true
				break
			}
		}
		if !found {
			messages = append(messages, fmt.Sprintf("Missing plugin '%s' in spec.master.basePlugins", requiredBasePlugin.Name))
		}
	}

	return messages
}

func (r *ReconcileJenkinsBaseConfiguration) validateCustomization(customization v1alpha2.Customization, name string) ([]string, error) {
	var messages []string
	if len(customization.Secret.Name) == 0 && len(customization.Configurations) == 0 {
		return nil, nil
	}
	if len(customization.Secret.Name) > 0 && len(customization.Configurations) == 0 {
		messages = append(messages, fmt.Sprintf("%s.secret.name is set but %s.configurations is empty", name, name))
	}

	if len(customization.Secret.Name) > 0 {
		secret := &corev1.Secret{}
		err := r.Client.Get(context.TODO(), types.NamespacedName{Name: customization.Secret.Name, Namespace: r.Configuration.Jenkins.ObjectMeta.Namespace}, secret)
		if err != nil && apierrors.IsNotFound(err) {
			messages = append(messages, fmt.Sprintf("Secret '%s' configured in %s.secret.name not found", customization.Secret.Name, name))
		} else if err != nil && !apierrors.IsNotFound(err) {
			return nil, stackerr.WithStack(err)
		}
	}

	for index, configMapRef := range customization.Configurations {
		if len(configMapRef.Name) == 0 {
			messages = append(messages, fmt.Sprintf("%s.configurations[%d] name is empty", name, index))
			continue
		}

		configMap := &corev1.ConfigMap{}
		err := r.Client.Get(context.TODO(), types.NamespacedName{Name: configMapRef.Name, Namespace: r.Configuration.Jenkins.ObjectMeta.Namespace}, configMap)
		if err != nil && apierrors.IsNotFound(err) {
			messages = append(messages, fmt.Sprintf("ConfigMap '%s' configured in %s.configurations[%d] not found", configMapRef.Name, name, index))
		} else if err != nil && !apierrors.IsNotFound(err) {
			return nil, stackerr.WithStack(err)
		}
	}

	return messages, nil
}
