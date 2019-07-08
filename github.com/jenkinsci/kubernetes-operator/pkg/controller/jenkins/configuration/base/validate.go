package base

import (
	"context"
	"fmt"
	"regexp"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/plugins"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"

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
func (r *ReconcileJenkinsBaseConfiguration) Validate(jenkins *v1alpha2.Jenkins) (bool, error) {
	if !r.validateReservedVolumes() {
		return false, nil
	}
	if valid, err := r.validateVolumes(); err != nil {
		return false, err
	} else if !valid {
		return false, nil
	}

	for _, container := range jenkins.Spec.Master.Containers {
		if !r.validateContainer(container) {
			return false, nil
		}
	}

	if !r.validatePlugins(plugins.BasePlugins(), jenkins.Spec.Master.BasePlugins, jenkins.Spec.Master.Plugins) {
		return false, nil
	}

	if !r.validateJenkinsMasterPodEnvs() {
		return false, nil
	}

	if valid, err := r.validateCustomization(r.jenkins.Spec.GroovyScripts.Customization, "spec.groovyScripts"); err != nil {
		return false, err
	} else if !valid {
		return false, nil
	}
	if valid, err := r.validateCustomization(r.jenkins.Spec.ConfigurationAsCode.Customization, "spec.configurationAsCode"); err != nil {
		return false, err
	} else if !valid {
		return false, nil
	}

	return true, nil
}

func (r *ReconcileJenkinsBaseConfiguration) validateVolumes() (bool, error) {
	valid := true
	for _, volume := range r.jenkins.Spec.Master.Volumes {
		switch {
		case volume.ConfigMap != nil:
			if ok, err := r.validateConfigMapVolume(volume); err != nil {
				return false, err
			} else if !ok {
				valid = false
			}
		case volume.Secret != nil:
			if ok, err := r.validateSecretVolume(volume); err != nil {
				return false, err
			} else if !ok {
				valid = false
			}
		case volume.PersistentVolumeClaim != nil:
			if ok, err := r.validatePersistentVolumeClaim(volume); err != nil {
				return false, err
			} else if !ok {
				valid = false
			}
		default: //TODO add support for rest of volumes
			valid = false
			r.logger.V(log.VWarn).Info(fmt.Sprintf("Unsupported volume '%v'", volume))
		}
	}

	return valid, nil
}

func (r *ReconcileJenkinsBaseConfiguration) validatePersistentVolumeClaim(volume corev1.Volume) (bool, error) {
	pvc := &corev1.PersistentVolumeClaim{}
	err := r.k8sClient.Get(context.TODO(), types.NamespacedName{Name: volume.PersistentVolumeClaim.ClaimName, Namespace: r.jenkins.ObjectMeta.Namespace}, pvc)
	if err != nil && apierrors.IsNotFound(err) {
		r.logger.V(log.VWarn).Info(fmt.Sprintf("PersistentVolumeClaim '%s' not found for volume '%v'", volume.PersistentVolumeClaim.ClaimName, volume))
		return false, nil
	} else if err != nil && !apierrors.IsNotFound(err) {
		return false, stackerr.WithStack(err)
	}

	return true, nil
}

func (r *ReconcileJenkinsBaseConfiguration) validateConfigMapVolume(volume corev1.Volume) (bool, error) {
	if volume.ConfigMap.Optional != nil && *volume.ConfigMap.Optional {
		return true, nil
	}

	configMap := &corev1.ConfigMap{}
	err := r.k8sClient.Get(context.TODO(), types.NamespacedName{Name: volume.ConfigMap.Name, Namespace: r.jenkins.ObjectMeta.Namespace}, configMap)
	if err != nil && apierrors.IsNotFound(err) {
		r.logger.V(log.VWarn).Info(fmt.Sprintf("ConfigMap '%s' not found for volume '%v'", volume.ConfigMap.Name, volume))
		return false, nil
	} else if err != nil && !apierrors.IsNotFound(err) {
		return false, stackerr.WithStack(err)
	}

	return true, nil
}

func (r *ReconcileJenkinsBaseConfiguration) validateSecretVolume(volume corev1.Volume) (bool, error) {
	if volume.Secret.Optional != nil && *volume.Secret.Optional {
		return true, nil
	}

	secret := &corev1.Secret{}
	err := r.k8sClient.Get(context.TODO(), types.NamespacedName{Name: volume.Secret.SecretName, Namespace: r.jenkins.ObjectMeta.Namespace}, secret)
	if err != nil && apierrors.IsNotFound(err) {
		r.logger.V(log.VWarn).Info(fmt.Sprintf("Secret '%s' not found for volume '%v'", volume.Secret.SecretName, volume))
		return false, nil
	} else if err != nil && !apierrors.IsNotFound(err) {
		return false, stackerr.WithStack(err)
	}

	return true, nil
}

func (r *ReconcileJenkinsBaseConfiguration) validateReservedVolumes() bool {
	valid := true

	for _, baseVolume := range resources.GetJenkinsMasterPodBaseVolumes(r.jenkins) {
		for _, volume := range r.jenkins.Spec.Master.Volumes {
			if baseVolume.Name == volume.Name {
				r.logger.V(log.VWarn).Info(fmt.Sprintf("Jenkins Master pod volume '%s' is reserved please choose different one", volume.Name))
				valid = false
			}
		}
	}

	return valid
}

func (r *ReconcileJenkinsBaseConfiguration) validateContainer(container v1alpha2.Container) bool {
	logger := r.logger.WithValues("container", container.Name)
	if container.Image == "" {
		logger.V(log.VWarn).Info("Image not set")
		return false
	}

	if !dockerImageRegexp.MatchString(container.Image) && !docker.ReferenceRegexp.MatchString(container.Image) {
		logger.V(log.VWarn).Info("Invalid image")
		return false
	}

	if container.ImagePullPolicy == "" {
		logger.V(log.VWarn).Info("Image pull policy not set")
		return false
	}

	if !r.validateContainerVolumeMounts(container) {
		return false
	}

	return true
}

func (r *ReconcileJenkinsBaseConfiguration) validateContainerVolumeMounts(container v1alpha2.Container) bool {
	logger := r.logger.WithValues("container", container.Name)
	allVolumes := append(resources.GetJenkinsMasterPodBaseVolumes(r.jenkins), r.jenkins.Spec.Master.Volumes...)
	valid := true

	for _, volumeMount := range container.VolumeMounts {
		if len(volumeMount.MountPath) == 0 {
			logger.V(log.VWarn).Info(fmt.Sprintf("mountPath not set for '%s' volume mount in container '%s'", volumeMount.Name, container.Name))
			valid = false
		}

		foundVolume := false
		for _, volume := range allVolumes {
			if volumeMount.Name == volume.Name {
				foundVolume = true
			}
		}

		if !foundVolume {
			logger.V(log.VWarn).Info(fmt.Sprintf("Not found volume for '%s' volume mount in container '%s'", volumeMount.Name, container.Name))
			valid = false
		}
	}

	return valid
}

func (r *ReconcileJenkinsBaseConfiguration) validateJenkinsMasterPodEnvs() bool {
	baseEnvs := resources.GetJenkinsMasterContainerBaseEnvs(r.jenkins)
	baseEnvNames := map[string]string{}
	for _, env := range baseEnvs {
		baseEnvNames[env.Name] = env.Value
	}

	valid := true
	for _, userEnv := range r.jenkins.Spec.Master.Containers[0].Env {
		if _, overriding := baseEnvNames[userEnv.Name]; overriding {
			r.logger.V(log.VWarn).Info(fmt.Sprintf("Jenkins Master pod env '%s' cannot be overridden", userEnv.Name))
			valid = false
		}
	}

	return valid
}

func (r *ReconcileJenkinsBaseConfiguration) validatePlugins(requiredBasePlugins []plugins.Plugin, basePlugins, userPlugins []v1alpha2.Plugin) bool {
	valid := true
	allPlugins := map[plugins.Plugin][]plugins.Plugin{}

	for _, jenkinsPlugin := range basePlugins {
		plugin, err := plugins.NewPlugin(jenkinsPlugin.Name, jenkinsPlugin.Version)
		if err != nil {
			r.logger.V(log.VWarn).Info(err.Error())
			valid = false
		}

		if plugin != nil {
			allPlugins[*plugin] = []plugins.Plugin{}
		}
	}

	for _, jenkinsPlugin := range userPlugins {
		plugin, err := plugins.NewPlugin(jenkinsPlugin.Name, jenkinsPlugin.Version)
		if err != nil {
			r.logger.V(log.VWarn).Info(err.Error())
			valid = false
		}

		if plugin != nil {
			allPlugins[*plugin] = []plugins.Plugin{}
		}
	}

	if !plugins.VerifyDependencies(allPlugins) {
		valid = false
	}

	if !r.verifyBasePlugins(requiredBasePlugins, basePlugins) {
		valid = false
	}

	return valid
}

func (r *ReconcileJenkinsBaseConfiguration) verifyBasePlugins(requiredBasePlugins []plugins.Plugin, basePlugins []v1alpha2.Plugin) bool {
	valid := true

	for _, requiredBasePlugin := range requiredBasePlugins {
		found := false
		for _, basePlugin := range basePlugins {
			if requiredBasePlugin.Name == basePlugin.Name {
				found = true
				break
			}
		}
		if !found {
			valid = false
			r.logger.V(log.VWarn).Info(fmt.Sprintf("Missing plugin '%s' in spec.master.basePlugins", requiredBasePlugin.Name))
		}
	}

	return valid
}

func (r *ReconcileJenkinsBaseConfiguration) validateCustomization(customization v1alpha2.Customization, name string) (bool, error) {
	valid := true
	if len(customization.Secret.Name) == 0 && len(customization.Configurations) == 0 {
		return true, nil
	}
	if len(customization.Secret.Name) > 0 && len(customization.Configurations) == 0 {
		valid = false
		r.logger.V(log.VWarn).Info(fmt.Sprintf("%s.secret.name is set but %s.configurations is empty", name, name))
	}

	if len(customization.Secret.Name) > 0 {
		secret := &corev1.Secret{}
		err := r.k8sClient.Get(context.TODO(), types.NamespacedName{Name: customization.Secret.Name, Namespace: r.jenkins.ObjectMeta.Namespace}, secret)
		if err != nil && apierrors.IsNotFound(err) {
			valid = false
			r.logger.V(log.VWarn).Info(fmt.Sprintf("Secret '%s' configured in %s.secret.name not found", customization.Secret.Name, name))
		} else if err != nil && !apierrors.IsNotFound(err) {
			return false, stackerr.WithStack(err)
		}
	}

	for index, configMapRef := range customization.Configurations {
		if len(configMapRef.Name) == 0 {
			r.logger.V(log.VWarn).Info(fmt.Sprintf("%s.configurations[%d] name is empty", name, index))
			valid = false
			continue
		}

		configMap := &corev1.ConfigMap{}
		err := r.k8sClient.Get(context.TODO(), types.NamespacedName{Name: configMapRef.Name, Namespace: r.jenkins.ObjectMeta.Namespace}, configMap)
		if err != nil && apierrors.IsNotFound(err) {
			valid = false
			r.logger.V(log.VWarn).Info(fmt.Sprintf("ConfigMap '%s' configured in %s.configurations[%d] not found", configMapRef.Name, name, index))
			return false, nil
		} else if err != nil && !apierrors.IsNotFound(err) {
			return false, stackerr.WithStack(err)
		}
	}

	return valid, nil
}
