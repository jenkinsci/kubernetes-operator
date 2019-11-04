package base

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/backuprestore"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/groovy"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/notifications/event"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/notifications/reason"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/plugins"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"
	"github.com/jenkinsci/kubernetes-operator/version"

	"github.com/bndr/gojenkins"
	"github.com/go-logr/logr"
	stackerr "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	fetchAllPlugins = 1
)

// ReconcileJenkinsBaseConfiguration defines values required for Jenkins base configuration
type ReconcileJenkinsBaseConfiguration struct {
	configuration.Configuration
	scheme          *runtime.Scheme
	logger          logr.Logger
	local, minikube bool
	config          *rest.Config
}

// New create structure which takes care of base configuration
func New(config configuration.Configuration, scheme *runtime.Scheme, logger logr.Logger, local, minikube bool, restConfig *rest.Config) *ReconcileJenkinsBaseConfiguration {
	return &ReconcileJenkinsBaseConfiguration{
		Configuration: config,
		scheme:        scheme,
		logger:        logger,
		local:         local,
		minikube:      minikube,
		config:        restConfig,
	}
}

// Reconcile takes care of base configuration
func (r *ReconcileJenkinsBaseConfiguration) Reconcile() (reconcile.Result, jenkinsclient.Jenkins, error) {
	metaObject := resources.NewResourceObjectMeta(r.Configuration.Jenkins)

	err := r.ensureResourcesRequiredForJenkinsPod(metaObject)
	if err != nil {
		return reconcile.Result{}, nil, err
	}
	r.logger.V(log.VDebug).Info("Kubernetes resources are present")

	result, err := r.ensureJenkinsMasterPod(metaObject)
	if err != nil {
		return reconcile.Result{}, nil, err
	}
	if result.Requeue {
		return result, nil, nil
	}
	r.logger.V(log.VDebug).Info("Jenkins master pod is present")

	stopReconcileLoop, err := r.detectJenkinsMasterPodStartingIssues(metaObject)
	if err != nil {
		return reconcile.Result{}, nil, err
	}
	if stopReconcileLoop {
		return reconcile.Result{Requeue: false}, nil, nil
	}

	result, err = r.waitForJenkins(metaObject)
	if err != nil {
		return reconcile.Result{}, nil, err
	}
	if result.Requeue {
		return result, nil, nil
	}
	r.logger.V(log.VDebug).Info("Jenkins master pod is ready")

	jenkinsClient, err := r.ensureJenkinsClient(metaObject)
	if err != nil {
		return reconcile.Result{}, nil, err
	}
	r.logger.V(log.VDebug).Info("Jenkins API client set")

	ok, err := r.verifyPlugins(jenkinsClient)
	if err != nil {
		return reconcile.Result{}, nil, err
	}
	if !ok {
		message := "Some plugins have changed, restarting Jenkins"
		r.logger.Info(message)

		restartReason := reason.NewPodRestart(
			reason.OperatorSource,
			[]string{message},
		)
		return reconcile.Result{Requeue: true}, nil, r.Configuration.RestartJenkinsMasterPod(restartReason)
	}

	result, err = r.ensureBaseConfiguration(jenkinsClient)

	return result, jenkinsClient, err
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

func (r *ReconcileJenkinsBaseConfiguration) ensureResourcesRequiredForJenkinsPod(metaObject metav1.ObjectMeta) error {
	if err := r.createOperatorCredentialsSecret(metaObject); err != nil {
		return err
	}
	r.logger.V(log.VDebug).Info("Operator credentials secret is present")

	if err := r.createScriptsConfigMap(metaObject); err != nil {
		return err
	}
	r.logger.V(log.VDebug).Info("Scripts config map is present")

	if err := r.createInitConfigurationConfigMap(metaObject); err != nil {
		return err
	}
	r.logger.V(log.VDebug).Info("Init configuration config map is present")

	if err := r.createBaseConfigurationConfigMap(metaObject); err != nil {
		return err
	}
	r.logger.V(log.VDebug).Info("Base configuration config map is present")

	if err := r.addLabelForWatchesResources(r.Configuration.Jenkins.Spec.GroovyScripts.Customization); err != nil {
		return err
	}
	r.logger.V(log.VDebug).Info("GroovyScripts Secret and ConfigMap added watched labels")

	if err := r.addLabelForWatchesResources(r.Configuration.Jenkins.Spec.ConfigurationAsCode.Customization); err != nil {
		return err
	}
	r.logger.V(log.VDebug).Info("ConfigurationAsCode Secret and ConfigMap added watched labels")

	if err := r.createRBAC(metaObject); err != nil {
		return err
	}
	r.logger.V(log.VDebug).Info("Service account, role and role binding are present")

	if err := r.createService(metaObject, resources.GetJenkinsHTTPServiceName(r.Configuration.Jenkins), r.Configuration.Jenkins.Spec.Service); err != nil {
		return err
	}
	r.logger.V(log.VDebug).Info("Jenkins HTTP Service is present")
	if err := r.createService(metaObject, resources.GetJenkinsSlavesServiceName(r.Configuration.Jenkins), r.Configuration.Jenkins.Spec.SlaveService); err != nil {
		return err
	}
	r.logger.V(log.VDebug).Info("Jenkins slave Service is present")

	return nil
}

func (r *ReconcileJenkinsBaseConfiguration) verifyPlugins(jenkinsClient jenkinsclient.Jenkins) (bool, error) {
	allPluginsInJenkins, err := jenkinsClient.GetPlugins(fetchAllPlugins)
	if err != nil {
		return false, stackerr.WithStack(err)
	}

	var installedPlugins []string
	for _, jenkinsPlugin := range allPluginsInJenkins.Raw.Plugins {
		if isValidPlugin(jenkinsPlugin) {
			installedPlugins = append(installedPlugins, plugins.Plugin{Name: jenkinsPlugin.ShortName, Version: jenkinsPlugin.Version}.String())
		}
	}
	r.logger.V(log.VDebug).Info(fmt.Sprintf("Installed plugins '%+v'", installedPlugins))

	status := true
	allRequiredPlugins := [][]v1alpha2.Plugin{r.Configuration.Jenkins.Spec.Master.BasePlugins, r.Configuration.Jenkins.Spec.Master.Plugins}
	for _, requiredPlugins := range allRequiredPlugins {
		for _, plugin := range requiredPlugins {
			if found, ok := isPluginInstalled(allPluginsInJenkins, plugin); !ok {
				r.logger.V(log.VWarn).Info(fmt.Sprintf("Missing plugin '%s', actual '%+v'", plugin, found))
				status = false
			}
			if found, ok := isPluginVersionCompatible(allPluginsInJenkins, plugin); !ok {
				r.logger.V(log.VWarn).Info(fmt.Sprintf("Incompatible plugin '%s' version, actual '%+v'", plugin, found.Version))
				status = false
			}
		}
	}

	return status, nil
}

func isPluginVersionCompatible(plugins *gojenkins.Plugins, plugin v1alpha2.Plugin) (gojenkins.Plugin, bool) {
	p := plugins.Contains(plugin.Name)
	if p == nil {
		return gojenkins.Plugin{}, false
	}

	return *p, p.Version == plugin.Version
}

func isValidPlugin(plugin gojenkins.Plugin) bool {
	return plugin.Active && plugin.Enabled && !plugin.Deleted
}

func isPluginInstalled(plugins *gojenkins.Plugins, requiredPlugin v1alpha2.Plugin) (gojenkins.Plugin, bool) {
	p := plugins.Contains(requiredPlugin.Name)
	if p == nil {
		return gojenkins.Plugin{}, false
	}

	return *p, isValidPlugin(*p)
}

func (r *ReconcileJenkinsBaseConfiguration) createOperatorCredentialsSecret(meta metav1.ObjectMeta) error {
	found := &corev1.Secret{}
	err := r.Configuration.Client.Get(context.TODO(), types.NamespacedName{Name: resources.GetOperatorCredentialsSecretName(r.Configuration.Jenkins), Namespace: r.Configuration.Jenkins.ObjectMeta.Namespace}, found)

	if err != nil && apierrors.IsNotFound(err) {
		return stackerr.WithStack(r.createResource(resources.NewOperatorCredentialsSecret(meta, r.Configuration.Jenkins)))
	} else if err != nil && !apierrors.IsNotFound(err) {
		return stackerr.WithStack(err)
	}

	if found.Data[resources.OperatorCredentialsSecretUserNameKey] != nil &&
		found.Data[resources.OperatorCredentialsSecretPasswordKey] != nil {
		return nil
	}

	return stackerr.WithStack(r.updateResource(resources.NewOperatorCredentialsSecret(meta, r.Configuration.Jenkins)))
}

func (r *ReconcileJenkinsBaseConfiguration) createScriptsConfigMap(meta metav1.ObjectMeta) error {
	configMap, err := resources.NewScriptsConfigMap(meta, r.Configuration.Jenkins)
	if err != nil {
		return err
	}
	return stackerr.WithStack(r.createOrUpdateResource(configMap))
}

func (r *ReconcileJenkinsBaseConfiguration) createInitConfigurationConfigMap(meta metav1.ObjectMeta) error {
	configMap, err := resources.NewInitConfigurationConfigMap(meta, r.Configuration.Jenkins)
	if err != nil {
		return err
	}
	return stackerr.WithStack(r.createOrUpdateResource(configMap))
}

func (r *ReconcileJenkinsBaseConfiguration) createBaseConfigurationConfigMap(meta metav1.ObjectMeta) error {
	configMap := resources.NewBaseConfigurationConfigMap(meta, r.Configuration.Jenkins)
	return stackerr.WithStack(r.createOrUpdateResource(configMap))
}

func (r *ReconcileJenkinsBaseConfiguration) addLabelForWatchesResources(customization v1alpha2.Customization) error {
	labelsForWatchedResources := resources.BuildLabelsForWatchedResources(*r.Configuration.Jenkins)

	if len(customization.Secret.Name) > 0 {
		secret := &corev1.Secret{}
		err := r.Client.Get(context.TODO(), types.NamespacedName{Name: customization.Secret.Name, Namespace: r.Configuration.Jenkins.Namespace}, secret)
		if err != nil {
			return stackerr.WithStack(err)
		}

		if !resources.VerifyIfLabelsAreSet(secret, labelsForWatchedResources) {
			if len(secret.ObjectMeta.Labels) == 0 {
				secret.ObjectMeta.Labels = map[string]string{}
			}
			for key, value := range labelsForWatchedResources {
				secret.ObjectMeta.Labels[key] = value
			}

			if err = r.Client.Update(context.TODO(), secret); err != nil {
				return stackerr.WithStack(r.Client.Update(context.TODO(), secret))
			}
		}
	}

	for _, configMapRef := range customization.Configurations {
		configMap := &corev1.ConfigMap{}
		err := r.Client.Get(context.TODO(), types.NamespacedName{Name: configMapRef.Name, Namespace: r.Configuration.Jenkins.Namespace}, configMap)
		if err != nil {
			return stackerr.WithStack(err)
		}

		if !resources.VerifyIfLabelsAreSet(configMap, labelsForWatchedResources) {
			if len(configMap.ObjectMeta.Labels) == 0 {
				configMap.ObjectMeta.Labels = map[string]string{}
			}
			for key, value := range labelsForWatchedResources {
				configMap.ObjectMeta.Labels[key] = value
			}

			if err = r.Client.Update(context.TODO(), configMap); err != nil {
				return stackerr.WithStack(r.Client.Update(context.TODO(), configMap))
			}
		}
	}

	return nil
}

func (r *ReconcileJenkinsBaseConfiguration) createRBAC(meta metav1.ObjectMeta) error {
	serviceAccount := resources.NewServiceAccount(meta)
	err := r.createResource(serviceAccount)
	if err != nil && !errors.IsAlreadyExists(err) {
		return stackerr.WithStack(err)
	}

	role := resources.NewRole(meta)
	err = r.createOrUpdateResource(role)
	if err != nil {
		return stackerr.WithStack(err)
	}

	roleBinding := resources.NewRoleBinding(meta)
	err = r.createOrUpdateResource(roleBinding)
	if err != nil {
		return stackerr.WithStack(err)
	}

	return nil
}

func (r *ReconcileJenkinsBaseConfiguration) createService(meta metav1.ObjectMeta, name string, config v1alpha2.Service) error {
	service := corev1.Service{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: meta.Namespace}, &service)
	if err != nil && errors.IsNotFound(err) {
		service = resources.UpdateService(corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: meta.Namespace,
				Labels:    meta.Labels,
			},
			Spec: corev1.ServiceSpec{
				Selector: meta.Labels,
			},
		}, config)
		if err = r.createResource(&service); err != nil {
			return stackerr.WithStack(err)
		}
	} else if err != nil {
		return stackerr.WithStack(err)
	}

	service = resources.UpdateService(service, config)
	return stackerr.WithStack(r.updateResource(&service))
}

func (r *ReconcileJenkinsBaseConfiguration) getJenkinsMasterPod() (*corev1.Pod, error) {
	jenkinsMasterPodName := resources.GetJenkinsMasterPodName(*r.Configuration.Jenkins)
	currentJenkinsMasterPod := &corev1.Pod{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: jenkinsMasterPodName, Namespace: r.Configuration.Jenkins.Namespace}, currentJenkinsMasterPod)
	if err != nil {
		return nil, err // don't wrap error
	}
	return currentJenkinsMasterPod, nil
}

func (r *ReconcileJenkinsBaseConfiguration) ensureJenkinsMasterPod(meta metav1.ObjectMeta) (reconcile.Result, error) {
	userAndPasswordHash, err := r.calculateUserAndPasswordHash()
	if err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Pod already exists
	currentJenkinsMasterPod, err := r.getJenkinsMasterPod()
	if err != nil && errors.IsNotFound(err) {
		jenkinsMasterPod := resources.NewJenkinsMasterPod(meta, r.Configuration.Jenkins)
		if !reflect.DeepEqual(jenkinsMasterPod.Spec.Containers[0].Command, resources.GetJenkinsMasterContainerBaseCommand()) {
			r.logger.Info(fmt.Sprintf("spec.master.containers[%s].command has been overridden make sure the command looks like: '%v', otherwise the operator won't configure default user and install plugins",
				resources.JenkinsMasterContainerName, []string{"bash", "-c", fmt.Sprintf("%s/%s && <custom-command-here> && /sbin/tini -s -- /usr/local/bin/jenkins.sh",
					resources.JenkinsScriptsVolumePath, resources.InitScriptName)}))
		}
		*r.Notifications <- event.Event{
			Jenkins: *r.Configuration.Jenkins,
			Phase:   event.PhaseBase,
			Level:   v1alpha2.NotificationLevelInfo,
			Reason:  reason.NewPodCreation(reason.OperatorSource, []string{"Creating a new Jenkins Master Pod"}),
		}
		r.logger.Info(fmt.Sprintf("Creating a new Jenkins Master Pod %s/%s", jenkinsMasterPod.Namespace, jenkinsMasterPod.Name))
		err = r.createResource(jenkinsMasterPod)
		if err != nil {
			return reconcile.Result{}, stackerr.WithStack(err)
		}
		now := metav1.Now()
		r.Configuration.Jenkins.Status = v1alpha2.JenkinsStatus{
			ProvisionStartTime:  &now,
			LastBackup:          r.Configuration.Jenkins.Status.LastBackup,
			PendingBackup:       r.Configuration.Jenkins.Status.LastBackup,
			UserAndPasswordHash: userAndPasswordHash,
		}
		err = r.Client.Update(context.TODO(), r.Configuration.Jenkins)
		if err != nil {
			return reconcile.Result{Requeue: true}, err
		}
		return reconcile.Result{}, nil
	} else if err != nil && !errors.IsNotFound(err) {
		return reconcile.Result{}, stackerr.WithStack(err)
	}

	if currentJenkinsMasterPod != nil && isPodTerminating(*currentJenkinsMasterPod) && r.Configuration.Jenkins.Status.UserConfigurationCompletedTime != nil {
		backupAndRestore := backuprestore.New(r.Client, r.ClientSet, r.logger, r.Configuration.Jenkins, *r.config)
		backupAndRestore.StopBackupTrigger()
		if r.Configuration.Jenkins.Spec.Backup.MakeBackupBeforePodDeletion {
			if r.Configuration.Jenkins.Status.LastBackup == r.Configuration.Jenkins.Status.PendingBackup && !r.Configuration.Jenkins.Status.BackupDoneBeforePodDeletion {
				r.Configuration.Jenkins.Status.PendingBackup = r.Configuration.Jenkins.Status.PendingBackup + 1
				r.Configuration.Jenkins.Status.BackupDoneBeforePodDeletion = true
				err = r.Client.Update(context.TODO(), r.Configuration.Jenkins)
				if err != nil {
					return reconcile.Result{}, err
				}
			}
			if err = backupAndRestore.Backup(); err != nil {
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{Requeue: true}, nil
	}

	if currentJenkinsMasterPod == nil {
		return reconcile.Result{Requeue: true}, nil
	}

	restartReason := r.isRecreatePodNeeded(*currentJenkinsMasterPod, userAndPasswordHash)
	if hasMessages := len(restartReason.Short()) > 0; hasMessages {
		for _, msg := range restartReason.Verbose() {
			r.logger.Info(msg)
		}

		return reconcile.Result{Requeue: true}, r.Configuration.RestartJenkinsMasterPod(restartReason)
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileJenkinsBaseConfiguration) calculateUserAndPasswordHash() (string, error) {
	credentialsSecret := &corev1.Secret{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: resources.GetOperatorCredentialsSecretName(r.Configuration.Jenkins), Namespace: r.Configuration.Jenkins.ObjectMeta.Namespace}, credentialsSecret)
	if err != nil {
		return "", stackerr.WithStack(err)
	}

	hash := sha256.New()
	hash.Write(credentialsSecret.Data[resources.OperatorCredentialsSecretUserNameKey])
	hash.Write(credentialsSecret.Data[resources.OperatorCredentialsSecretPasswordKey])
	return base64.StdEncoding.EncodeToString(hash.Sum(nil)), nil
}

func isPodTerminating(pod corev1.Pod) bool {
	return pod.ObjectMeta.DeletionTimestamp != nil
}

func (r *ReconcileJenkinsBaseConfiguration) isRecreatePodNeeded(currentJenkinsMasterPod corev1.Pod, userAndPasswordHash string) *reason.PodRestart {
	var messages []string
	var verbose []string
	if userAndPasswordHash != r.Configuration.Jenkins.Status.UserAndPasswordHash {
		messages = append(messages, "User or password have changed, recreating pod")
	}

	if r.Configuration.Jenkins.Spec.Restore.RecoveryOnce != 0 && r.Configuration.Jenkins.Status.RestoredBackup != 0 {
		messages = append(messages, "spec.restore.recoveryOnce is set, recreating pod")
	}

	if version.Version != r.Configuration.Jenkins.Status.OperatorVersion {
		messages = append(messages, "Jenkins Operator version has changed")
		verbose = append(verbose, fmt.Sprintf("Jenkins Operator version has changed, actual '%+v' new '%+v', recreating pod",
			r.Configuration.Jenkins.Status.OperatorVersion, version.Version))
	}

	if currentJenkinsMasterPod.Status.Phase == corev1.PodFailed ||
		currentJenkinsMasterPod.Status.Phase == corev1.PodSucceeded ||
		currentJenkinsMasterPod.Status.Phase == corev1.PodUnknown {
		messages = append(messages, fmt.Sprintf("Invalid Jenkins pod phase '%+v', recreating pod", currentJenkinsMasterPod.Status))
	}

	if !reflect.DeepEqual(r.Configuration.Jenkins.Spec.Master.SecurityContext, currentJenkinsMasterPod.Spec.SecurityContext) {
		messages = append(messages, fmt.Sprintf("Jenkins pod security context has changed"))
		verbose = append(verbose, fmt.Sprintf("Jenkins pod security context has changed, actual '%+v' required '%+v', recreating pod",
			currentJenkinsMasterPod.Spec.SecurityContext, r.Configuration.Jenkins.Spec.Master.SecurityContext))
	}

	if !reflect.DeepEqual(r.Configuration.Jenkins.Spec.Master.ImagePullSecrets, currentJenkinsMasterPod.Spec.ImagePullSecrets) {
		messages = append(messages, "Jenkins Pod ImagePullSecrets has changed")
		verbose = append(verbose, fmt.Sprintf("Jenkins Pod ImagePullSecrets has changed, actual '%+v' required '%+v', recreating pod",
			currentJenkinsMasterPod.Spec.ImagePullSecrets, r.Configuration.Jenkins.Spec.Master.ImagePullSecrets))
	}

	if !reflect.DeepEqual(r.Configuration.Jenkins.Spec.Master.NodeSelector, currentJenkinsMasterPod.Spec.NodeSelector) {
		messages = append(messages, "Jenkins pod node selector has changed")
		verbose = append(verbose, fmt.Sprintf("Jenkins pod node selector has changed, actual '%+v' required '%+v', recreating pod",
			currentJenkinsMasterPod.Spec.NodeSelector, r.Configuration.Jenkins.Spec.Master.NodeSelector))
	}

	if len(r.Configuration.Jenkins.Spec.Master.Annotations) > 0 &&
		!reflect.DeepEqual(r.Configuration.Jenkins.Spec.Master.Annotations, currentJenkinsMasterPod.ObjectMeta.Annotations) {
		messages = append(messages, "Jenkins pod annotations have changed")
		verbose = append(verbose, fmt.Sprintf("Jenkins pod annotations have changed to '%+v', recreating pod", r.Configuration.Jenkins.Spec.Master.Annotations))
	}

	if !r.compareVolumes(currentJenkinsMasterPod) {
		messages = append(messages, "Jenkins pod volumes have changed")
		verbose = append(verbose, fmt.Sprintf("Jenkins pod volumes have changed, actual '%v' required '%v', recreating pod",
			currentJenkinsMasterPod.Spec.Volumes, r.Configuration.Jenkins.Spec.Master.Volumes))
	}

	if len(r.Configuration.Jenkins.Spec.Master.Containers) != len(currentJenkinsMasterPod.Spec.Containers) {
		messages = append(messages, "Jenkins amount of containers has changed")
		verbose = append(verbose, fmt.Sprintf("Jenkins amount of containers has changed to '%+v', recreating pod", len(r.Configuration.Jenkins.Spec.Master.Containers)))
	}

	for _, actualContainer := range currentJenkinsMasterPod.Spec.Containers {
		if actualContainer.Name == resources.JenkinsMasterContainerName {
			containerMessages, verboseMessages := r.compareContainers(resources.NewJenkinsMasterContainer(r.Configuration.Jenkins), actualContainer)
			messages = append(messages, containerMessages...)
			verbose = append(verbose, verboseMessages...)
			continue
		}

		var expectedContainer *corev1.Container
		for _, jenkinsContainer := range r.Configuration.Jenkins.Spec.Master.Containers {
			if jenkinsContainer.Name == actualContainer.Name {
				tmp := resources.ConvertJenkinsContainerToKubernetesContainer(jenkinsContainer)
				expectedContainer = &tmp
			}
		}

		if expectedContainer == nil {
			messages = append(messages, fmt.Sprintf("Container '%+v' not found in pod, recreating pod", actualContainer))
		}

		containerMessages, verboseMessages := r.compareContainers(*expectedContainer, actualContainer)

		messages = append(messages, containerMessages...)
		verbose = append(verbose, verboseMessages...)
	}

	return reason.NewPodRestart(reason.KubernetesSource, messages, verbose...)
}

func (r *ReconcileJenkinsBaseConfiguration) compareContainers(expected corev1.Container, actual corev1.Container) (messages []string, verbose []string) {
	if !reflect.DeepEqual(expected.Args, actual.Args) {
		messages = append(messages, "Arguments have changed")
		verbose = append(messages, fmt.Sprintf("Arguments have changed to '%+v' in container '%s', recreating pod", expected.Args, expected.Name))
	}
	if !reflect.DeepEqual(expected.Command, actual.Command) {
		messages = append(messages, "Command has changed")
		verbose = append(verbose, fmt.Sprintf("Command has changed to '%+v' in container '%s', recreating pod", expected.Command, expected.Name))
	}
	if !compareEnv(expected.Env, actual.Env) {
		messages = append(messages, "Env has changed")
		verbose = append(verbose, fmt.Sprintf("Env has changed to '%+v' in container '%s', recreating pod", expected.Env, expected.Name))
	}
	if !reflect.DeepEqual(expected.EnvFrom, actual.EnvFrom) {
		messages = append(messages, "EnvFrom has changed")
		verbose = append(verbose, fmt.Sprintf("EnvFrom has changed to '%+v' in container '%s', recreating pod", expected.EnvFrom, expected.Name))
	}
	if !reflect.DeepEqual(expected.Image, actual.Image) {
		messages = append(messages, "Image has changed")
		verbose = append(verbose, fmt.Sprintf("Image has changed to '%+v' in container '%s', recreating pod", expected.Image, expected.Name))
	}
	if !reflect.DeepEqual(expected.ImagePullPolicy, actual.ImagePullPolicy) {
		messages = append(messages, "Image pull policy has changed")
		verbose = append(verbose, fmt.Sprintf("Image pull policy has changed to '%+v' in container '%s', recreating pod", expected.ImagePullPolicy, expected.Name))
	}
	if !reflect.DeepEqual(expected.Lifecycle, actual.Lifecycle) {
		messages = append(messages, "Lifecycle has changed")
		verbose = append(verbose, fmt.Sprintf("Lifecycle has changed to '%+v' in container '%s', recreating pod", expected.Lifecycle, expected.Name))
	}
	if !reflect.DeepEqual(expected.LivenessProbe, actual.LivenessProbe) {
		messages = append(messages, "Liveness probe has changed")
		verbose = append(verbose, fmt.Sprintf("Liveness probe has changed to '%+v' in container '%s', recreating pod", expected.LivenessProbe, expected.Name))
	}
	if !reflect.DeepEqual(expected.Ports, actual.Ports) {
		messages = append(messages, "Ports have changed")
		verbose = append(verbose, fmt.Sprintf("Ports have changed to '%+v' in container '%s', recreating pod", expected.Ports, expected.Name))
	}
	if !reflect.DeepEqual(expected.ReadinessProbe, actual.ReadinessProbe) {
		messages = append(messages, "Readiness probe has changed")
		verbose = append(verbose, fmt.Sprintf("Readiness probe has changed to '%+v' in container '%s', recreating pod", expected.ReadinessProbe, expected.Name))
	}
	if !reflect.DeepEqual(expected.Resources, actual.Resources) {
		messages = append(messages, "Resources have changed")
		verbose = append(verbose, fmt.Sprintf("Resources have changed to '%+v' in container '%s', recreating pod", expected.Resources, expected.Name))
	}
	if !reflect.DeepEqual(expected.SecurityContext, actual.SecurityContext) {
		messages = append(messages, "Security context has changed")
		verbose = append(verbose, fmt.Sprintf("Security context has changed to '%+v' in container '%s', recreating pod", expected.SecurityContext, expected.Name))
	}
	if !reflect.DeepEqual(expected.WorkingDir, actual.WorkingDir) {
		messages = append(messages, "Working directory has changed")
		verbose = append(verbose, fmt.Sprintf("Working directory has changed to '%+v' in container '%s', recreating pod", expected.WorkingDir, expected.Name))
	}
	if !CompareContainerVolumeMounts(expected, actual) {
		messages = append(messages, "Volume mounts have changed")
		verbose = append(verbose, fmt.Sprintf("Volume mounts have changed to '%+v' in container '%s', recreating pod", expected.VolumeMounts, expected.Name))
	}

	return messages, verbose
}

func compareEnv(expected, actual []corev1.EnvVar) bool {
	var actualEnv []corev1.EnvVar
	for _, env := range actual {
		if env.Name == "KUBERNETES_PORT_443_TCP_ADDR" || env.Name == "KUBERNETES_PORT" ||
			env.Name == "KUBERNETES_PORT_443_TCP" || env.Name == "KUBERNETES_SERVICE_HOST" {
			continue
		}
		actualEnv = append(actualEnv, env)
	}
	return reflect.DeepEqual(expected, actualEnv)
}

// CompareContainerVolumeMounts returns true if two containers volume mounts are the same
func CompareContainerVolumeMounts(expected corev1.Container, actual corev1.Container) bool {
	var withoutServiceAccount []corev1.VolumeMount
	for _, volumeMount := range actual.VolumeMounts {
		if volumeMount.MountPath != "/var/run/secrets/kubernetes.io/serviceaccount" {
			withoutServiceAccount = append(withoutServiceAccount, volumeMount)
		}
	}

	return reflect.DeepEqual(expected.VolumeMounts, withoutServiceAccount)
}

// compareVolumes returns true if Jenkins pod and Jenkins CR volumes are the same
func (r *ReconcileJenkinsBaseConfiguration) compareVolumes(actualPod corev1.Pod) bool {
	var withoutServiceAccount []corev1.Volume
	for _, volume := range actualPod.Spec.Volumes {
		if !strings.HasPrefix(volume.Name, actualPod.Spec.ServiceAccountName) {
			withoutServiceAccount = append(withoutServiceAccount, volume)
		}
	}

	return reflect.DeepEqual(
		append(resources.GetJenkinsMasterPodBaseVolumes(r.Configuration.Jenkins), r.Configuration.Jenkins.Spec.Master.Volumes...),
		withoutServiceAccount,
	)
}

func (r *ReconcileJenkinsBaseConfiguration) detectJenkinsMasterPodStartingIssues(meta metav1.ObjectMeta) (stopReconcileLoop bool, err error) {
	jenkinsMasterPod, err := r.getJenkinsMasterPod()
	if err != nil {
		return false, err
	}

	if jenkinsMasterPod.Status.Phase == corev1.PodPending {
		timeout := r.Configuration.Jenkins.Status.ProvisionStartTime.Add(time.Minute * 2).UTC()
		now := time.Now().UTC()
		if now.After(timeout) {
			events := &corev1.EventList{}
			err = r.Client.List(context.TODO(), &client.ListOptions{Namespace: r.Configuration.Jenkins.Namespace}, events)
			if err != nil {
				return false, stackerr.WithStack(err)
			}

			filteredEvents := r.filterEvents(*events, *jenkinsMasterPod)

			if len(filteredEvents) == 0 {
				return false, nil
			}

			r.logger.Info(fmt.Sprintf("Jenkins master pod starting timeout, events '%+v'", filteredEvents))
			return true, nil
		}
	}

	return false, nil
}

func (r *ReconcileJenkinsBaseConfiguration) filterEvents(source corev1.EventList, jenkinsMasterPod corev1.Pod) []string {
	events := []string{}
	for _, event := range source.Items {
		if r.Configuration.Jenkins.Status.ProvisionStartTime.UTC().After(event.LastTimestamp.UTC()) {
			continue
		}
		if event.Type == corev1.EventTypeNormal {
			continue
		}
		if !strings.HasPrefix(event.ObjectMeta.Name, jenkinsMasterPod.Name) {
			continue
		}
		events = append(events, fmt.Sprintf("Message: %s Subobject: %s", event.Message, event.InvolvedObject.FieldPath))
	}

	return events
}

func (r *ReconcileJenkinsBaseConfiguration) waitForJenkins(meta metav1.ObjectMeta) (reconcile.Result, error) {
	jenkinsMasterPod, err := r.getJenkinsMasterPod()
	if err != nil {
		return reconcile.Result{}, err
	}

	if jenkinsMasterPod.ObjectMeta.DeletionTimestamp != nil {
		r.logger.V(log.VDebug).Info("Jenkins master pod is terminating")
		return reconcile.Result{Requeue: true, RequeueAfter: time.Second * 5}, nil
	}

	if jenkinsMasterPod.Status.Phase != corev1.PodRunning {
		r.logger.V(log.VDebug).Info("Jenkins master pod not ready")
		return reconcile.Result{Requeue: true, RequeueAfter: time.Second * 5}, nil
	}

	containersReadyCount := 0
	for _, containerStatus := range jenkinsMasterPod.Status.ContainerStatuses {
		if containerStatus.State.Terminated != nil {
			message := fmt.Sprintf("Container '%s' is terminated, status '%+v', recreating pod", containerStatus.Name, containerStatus)
			r.logger.Info(message)

			restartReason := reason.NewPodRestart(
				reason.KubernetesSource,
				[]string{message},
			)
			return reconcile.Result{Requeue: true}, r.Configuration.RestartJenkinsMasterPod(restartReason)
		}
		if !containerStatus.Ready {
			r.logger.V(log.VDebug).Info(fmt.Sprintf("Container '%s' not ready, readiness probe failed", containerStatus.Name))
		} else {
			containersReadyCount++
		}
	}
	if containersReadyCount != len(jenkinsMasterPod.Status.ContainerStatuses) {
		return reconcile.Result{Requeue: true, RequeueAfter: time.Second * 5}, nil
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileJenkinsBaseConfiguration) ensureJenkinsClient(meta metav1.ObjectMeta) (jenkinsclient.Jenkins, error) {
	jenkinsURL, err := jenkinsclient.BuildJenkinsAPIUrl(
		r.Configuration.Jenkins.ObjectMeta.Namespace, resources.GetJenkinsHTTPServiceName(r.Configuration.Jenkins), r.Configuration.Jenkins.Spec.Service.Port, r.local, r.minikube)

	if prefix, ok := GetJenkinsOpts(*r.Configuration.Jenkins)["prefix"]; ok {
		jenkinsURL = jenkinsURL + prefix
	}

	if err != nil {
		return nil, err
	}
	r.logger.V(log.VDebug).Info(fmt.Sprintf("Jenkins API URL '%s'", jenkinsURL))

	credentialsSecret := &corev1.Secret{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: resources.GetOperatorCredentialsSecretName(r.Configuration.Jenkins), Namespace: r.Configuration.Jenkins.ObjectMeta.Namespace}, credentialsSecret)
	if err != nil {
		return nil, stackerr.WithStack(err)
	}
	currentJenkinsMasterPod, err := r.getJenkinsMasterPod()
	if err != nil {
		return nil, err
	}

	var tokenCreationTime *time.Time
	tokenCreationTimeBytes := credentialsSecret.Data[resources.OperatorCredentialsSecretTokenCreationKey]
	if tokenCreationTimeBytes != nil {
		tokenCreationTime = &time.Time{}
		err = tokenCreationTime.UnmarshalText(tokenCreationTimeBytes)
		if err != nil {
			tokenCreationTime = nil
		}

	}
	if credentialsSecret.Data[resources.OperatorCredentialsSecretTokenKey] == nil ||
		tokenCreationTimeBytes == nil || tokenCreationTime == nil ||
		currentJenkinsMasterPod.ObjectMeta.CreationTimestamp.Time.UTC().After(tokenCreationTime.UTC()) {
		r.logger.Info("Generating Jenkins API token for operator")
		userName := string(credentialsSecret.Data[resources.OperatorCredentialsSecretUserNameKey])
		jenkinsClient, err := jenkinsclient.New(
			jenkinsURL,
			userName,
			string(credentialsSecret.Data[resources.OperatorCredentialsSecretPasswordKey]))
		if err != nil {
			return nil, err
		}

		token, err := jenkinsClient.GenerateToken(userName, "token")
		if err != nil {
			return nil, err
		}

		credentialsSecret.Data[resources.OperatorCredentialsSecretTokenKey] = []byte(token.GetToken())
		now, _ := time.Now().UTC().MarshalText()
		credentialsSecret.Data[resources.OperatorCredentialsSecretTokenCreationKey] = now
		err = r.updateResource(credentialsSecret)
		if err != nil {
			return nil, stackerr.WithStack(err)
		}
	}

	return jenkinsclient.New(
		jenkinsURL,
		string(credentialsSecret.Data[resources.OperatorCredentialsSecretUserNameKey]),
		string(credentialsSecret.Data[resources.OperatorCredentialsSecretTokenKey]))
}

func (r *ReconcileJenkinsBaseConfiguration) ensureBaseConfiguration(jenkinsClient jenkinsclient.Jenkins) (reconcile.Result, error) {
	customization := v1alpha2.GroovyScripts{
		Customization: v1alpha2.Customization{
			Secret:         v1alpha2.SecretRef{Name: ""},
			Configurations: []v1alpha2.ConfigMapRef{{Name: resources.GetBaseConfigurationConfigMapName(r.Configuration.Jenkins)}},
		},
	}

	groovyClient := groovy.New(jenkinsClient, r.Client, r.logger, r.Configuration.Jenkins, "base-groovy", customization.Customization)

	requeue, err := groovyClient.Ensure(func(name string) bool {
		return strings.HasSuffix(name, ".groovy")
	}, func(groovyScript string) string {
		return groovyScript
	})

	return reconcile.Result{Requeue: requeue}, err
}
