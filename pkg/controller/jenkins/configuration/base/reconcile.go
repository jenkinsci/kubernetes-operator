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
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	fetchAllPlugins = 1
)

// ReconcileJenkinsBaseConfiguration defines values required for Jenkins base configuration
type ReconcileJenkinsBaseConfiguration struct {
	configuration.Configuration
	logger                       logr.Logger
	jenkinsAPIConnectionSettings jenkinsclient.JenkinsAPIConnectionSettings
}

// New create structure which takes care of base configuration
func New(config configuration.Configuration, logger logr.Logger, jenkinsAPIConnectionSettings jenkinsclient.JenkinsAPIConnectionSettings) *ReconcileJenkinsBaseConfiguration {
	return &ReconcileJenkinsBaseConfiguration{
		Configuration:                config,
		logger:                       logger,
		jenkinsAPIConnectionSettings: jenkinsAPIConnectionSettings,
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

	stopReconcileLoop, err := r.detectJenkinsMasterPodStartingIssues()
	if err != nil {
		return reconcile.Result{}, nil, err
	}
	if stopReconcileLoop {
		return reconcile.Result{Requeue: false}, nil, nil
	}

	result, err = r.waitForJenkins()
	if err != nil {
		return reconcile.Result{}, nil, err
	}
	if result.Requeue {
		return result, nil, nil
	}
	r.logger.V(log.VDebug).Info("Jenkins master pod is ready")

	jenkinsClient, err := r.ensureJenkinsClient()
	if err != nil {
		return reconcile.Result{}, nil, err
	}
	r.logger.V(log.VDebug).Info("Jenkins API client set")

	ok, err := r.verifyPlugins(jenkinsClient)
	if err != nil {
		return reconcile.Result{}, nil, err
	}
	if !ok {
		//TODO add what plugins have been changed
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

	if err := r.ensureExtraRBAC(metaObject); err != nil {
		return err
	}
	r.logger.V(log.VDebug).Info("Extra role bindings are present")

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
			if _, ok := isPluginInstalled(allPluginsInJenkins, plugin); !ok {
				r.logger.V(log.VWarn).Info(fmt.Sprintf("Missing plugin '%s'", plugin))
				status = false
				continue
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
		return stackerr.WithStack(r.CreateResource(resources.NewOperatorCredentialsSecret(meta, r.Configuration.Jenkins)))
	} else if err != nil && !apierrors.IsNotFound(err) {
		return stackerr.WithStack(err)
	}

	if found.Data[resources.OperatorCredentialsSecretUserNameKey] != nil &&
		found.Data[resources.OperatorCredentialsSecretPasswordKey] != nil {
		return nil
	}

	return stackerr.WithStack(r.UpdateResource(resources.NewOperatorCredentialsSecret(meta, r.Configuration.Jenkins)))
}

func (r *ReconcileJenkinsBaseConfiguration) createScriptsConfigMap(meta metav1.ObjectMeta) error {
	configMap, err := resources.NewScriptsConfigMap(meta, r.Configuration.Jenkins)
	if err != nil {
		return err
	}
	return stackerr.WithStack(r.CreateOrUpdateResource(configMap))
}

func (r *ReconcileJenkinsBaseConfiguration) createInitConfigurationConfigMap(meta metav1.ObjectMeta) error {
	configMap, err := resources.NewInitConfigurationConfigMap(meta, r.Configuration.Jenkins)
	if err != nil {
		return err
	}
	return stackerr.WithStack(r.CreateOrUpdateResource(configMap))
}

func (r *ReconcileJenkinsBaseConfiguration) createBaseConfigurationConfigMap(meta metav1.ObjectMeta) error {
	configMap, err := resources.NewBaseConfigurationConfigMap(meta, r.Configuration.Jenkins)
	if err != nil {
		return stackerr.WithStack(err)
	}
	return stackerr.WithStack(r.CreateOrUpdateResource(configMap))
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

func (r *ReconcileJenkinsBaseConfiguration) createServiceAccount(meta metav1.ObjectMeta) error {
	serviceAccount := &corev1.ServiceAccount{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: meta.Name, Namespace: meta.Namespace}, serviceAccount)
	if err != nil && apierrors.IsNotFound(err) {
		serviceAccount = resources.NewServiceAccount(meta, r.Configuration.Jenkins.Spec.ServiceAccount.Annotations)
		if err = r.CreateResource(serviceAccount); err != nil {
			return stackerr.WithStack(err)
		}
	} else if err != nil {
		return stackerr.WithStack(err)
	}

	if !compareMap(r.Configuration.Jenkins.Spec.ServiceAccount.Annotations, serviceAccount.Annotations) {
		if serviceAccount.Annotations == nil {
			serviceAccount.Annotations = map[string]string{}
		}
		for key, value := range r.Configuration.Jenkins.Spec.ServiceAccount.Annotations {
			serviceAccount.Annotations[key] = value
		}
		if err = r.UpdateResource(serviceAccount); err != nil {
			return stackerr.WithStack(err)
		}
	}

	return nil
}

func (r *ReconcileJenkinsBaseConfiguration) createRBAC(meta metav1.ObjectMeta) error {
	err := r.createServiceAccount(meta)
	if err != nil {
		return err
	}

	role := resources.NewRole(meta)
	err = r.CreateOrUpdateResource(role)
	if err != nil {
		return stackerr.WithStack(err)
	}

	roleBinding := resources.NewRoleBinding(meta.Name, meta.Namespace, meta.Name, rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "Role",
		Name:     meta.Name,
	})
	err = r.CreateOrUpdateResource(roleBinding)
	if err != nil {
		return stackerr.WithStack(err)
	}

	return nil
}

func (r *ReconcileJenkinsBaseConfiguration) ensureExtraRBAC(meta metav1.ObjectMeta) error {
	var err error
	var name string
	for _, roleRef := range r.Configuration.Jenkins.Spec.Roles {
		name = getExtraRoleBindingName(meta.Name, roleRef)
		roleBinding := resources.NewRoleBinding(name, meta.Namespace, meta.Name, roleRef)
		err = r.CreateOrUpdateResource(roleBinding)
		if err != nil {
			return stackerr.WithStack(err)
		}
	}

	roleBindings := &rbacv1.RoleBindingList{}
	err = r.Client.List(context.TODO(), roleBindings, client.InNamespace(r.Configuration.Jenkins.Namespace))
	if err != nil {
		return stackerr.WithStack(err)
	}
	for _, roleBinding := range roleBindings.Items {
		if !strings.HasPrefix(roleBinding.Name, getExtraRoleBindingName(meta.Name, rbacv1.RoleRef{Kind: "Role"})) &&
			!strings.HasPrefix(roleBinding.Name, getExtraRoleBindingName(meta.Name, rbacv1.RoleRef{Kind: "ClusterRole"})) {
			continue
		}

		found := false
		for _, roleRef := range r.Configuration.Jenkins.Spec.Roles {
			name = getExtraRoleBindingName(meta.Name, roleRef)
			if roleBinding.Name == name {
				found = true
				continue
			}
		}
		if !found {
			r.logger.Info(fmt.Sprintf("Deleting RoleBinding '%s'", roleBinding.Name))
			if err = r.Client.Delete(context.TODO(), &roleBinding); err != nil {
				return stackerr.WithStack(err)
			}
		}
	}

	return nil
}

func getExtraRoleBindingName(serviceAccountName string, roleRef rbacv1.RoleRef) string {
	var typeName string
	if roleRef.Kind == "ClusterRole" {
		typeName = "cr"
	} else {
		typeName = "r"
	}
	return fmt.Sprintf("%s-%s-%s", serviceAccountName, typeName, roleRef.Name)
}

func (r *ReconcileJenkinsBaseConfiguration) createService(meta metav1.ObjectMeta, name string, config v1alpha2.Service) error {
	service := corev1.Service{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: meta.Namespace}, &service)
	if err != nil && apierrors.IsNotFound(err) {
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
		if err = r.CreateResource(&service); err != nil {
			return stackerr.WithStack(err)
		}
	} else if err != nil {
		return stackerr.WithStack(err)
	}

	service.Spec.Selector = meta.Labels // make sure that user won't break service by hand
	service = resources.UpdateService(service, config)
	return stackerr.WithStack(r.UpdateResource(&service))
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
	if err != nil && apierrors.IsNotFound(err) {
		jenkinsMasterPod := resources.NewJenkinsMasterPod(meta, r.Configuration.Jenkins)
		*r.Notifications <- event.Event{
			Jenkins: *r.Configuration.Jenkins,
			Phase:   event.PhaseBase,
			Level:   v1alpha2.NotificationLevelInfo,
			Reason:  reason.NewPodCreation(reason.OperatorSource, []string{"Creating a new Jenkins Master Pod"}),
		}
		r.logger.Info(fmt.Sprintf("Creating a new Jenkins Master Pod %s/%s", jenkinsMasterPod.Namespace, jenkinsMasterPod.Name))
		err = r.CreateResource(jenkinsMasterPod)
		if err != nil {
			return reconcile.Result{}, stackerr.WithStack(err)
		}

		currentJenkinsMasterPod, err := r.waitUntilCreateJenkinsMasterPod()
		if err == nil {
			r.handleAdmissionControllerChanges(currentJenkinsMasterPod)
		} else {
			r.logger.V(log.VWarn).Info(fmt.Sprintf("waitUntilCreateJenkinsMasterPod has failed: %s", err))
		}

		now := metav1.Now()
		r.Configuration.Jenkins.Status = v1alpha2.JenkinsStatus{
			OperatorVersion:     version.Version,
			ProvisionStartTime:  &now,
			LastBackup:          r.Configuration.Jenkins.Status.LastBackup,
			PendingBackup:       r.Configuration.Jenkins.Status.LastBackup,
			UserAndPasswordHash: userAndPasswordHash,
		}
		return reconcile.Result{Requeue: true}, r.Client.Update(context.TODO(), r.Configuration.Jenkins)
	} else if err != nil && !apierrors.IsNotFound(err) {
		return reconcile.Result{}, stackerr.WithStack(err)
	}

	if currentJenkinsMasterPod == nil {
		return reconcile.Result{Requeue: true}, nil
	}

	if r.IsJenkinsTerminating(*currentJenkinsMasterPod) && r.Configuration.Jenkins.Status.UserConfigurationCompletedTime != nil {
		backupAndRestore := backuprestore.New(r.Configuration, r.logger)
		if backupAndRestore.IsBackupTriggerEnabled() {
			backupAndRestore.StopBackupTrigger()
		}
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

	if !r.IsJenkinsTerminating(*currentJenkinsMasterPod) {
		restartReason := r.checkForPodRecreation(*currentJenkinsMasterPod, userAndPasswordHash)
		if restartReason.HasMessages() {
			for _, msg := range restartReason.Verbose() {
				r.logger.Info(msg)
			}

			return reconcile.Result{Requeue: true}, r.Configuration.RestartJenkinsMasterPod(restartReason)
		}
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

func (r *ReconcileJenkinsBaseConfiguration) checkForPodRecreation(currentJenkinsMasterPod corev1.Pod, userAndPasswordHash string) reason.Reason {
	var messages []string
	var verbose []string

	if currentJenkinsMasterPod.Status.Phase == corev1.PodFailed ||
		currentJenkinsMasterPod.Status.Phase == corev1.PodSucceeded ||
		currentJenkinsMasterPod.Status.Phase == corev1.PodUnknown {
		//TODO add Jenkins last 10 line logs
		messages = append(messages, fmt.Sprintf("Invalid Jenkins pod phase '%s'", currentJenkinsMasterPod.Status.Phase))
		verbose = append(verbose, fmt.Sprintf("Invalid Jenkins pod phase '%+v'", currentJenkinsMasterPod.Status))
		return reason.NewPodRestart(reason.KubernetesSource, messages, verbose...)
	}

	userAndPasswordHashIsDifferent := userAndPasswordHash != r.Configuration.Jenkins.Status.UserAndPasswordHash
	userAndPasswordHashStatusNotEmpty := r.Configuration.Jenkins.Status.UserAndPasswordHash != ""

	if userAndPasswordHashIsDifferent && userAndPasswordHashStatusNotEmpty {
		messages = append(messages, "User or password have changed")
		verbose = append(verbose, "User or password have changed, recreating pod")
	}

	if r.Configuration.Jenkins.Spec.Restore.RecoveryOnce != 0 && r.Configuration.Jenkins.Status.RestoredBackup != 0 {
		messages = append(messages, "spec.restore.recoveryOnce is set")
		verbose = append(verbose, "spec.restore.recoveryOnce is set, recreating pod")
	}

	if version.Version != r.Configuration.Jenkins.Status.OperatorVersion {
		messages = append(messages, "Jenkins Operator version has changed")
		verbose = append(verbose, fmt.Sprintf("Jenkins Operator version has changed, actual '%+v' new '%+v'",
			r.Configuration.Jenkins.Status.OperatorVersion, version.Version))
	}

	if !reflect.DeepEqual(r.Configuration.Jenkins.Spec.Master.SecurityContext, currentJenkinsMasterPod.Spec.SecurityContext) {
		messages = append(messages, "Jenkins pod security context has changed")
		verbose = append(verbose, fmt.Sprintf("Jenkins pod security context has changed, actual '%+v' required '%+v'",
			currentJenkinsMasterPod.Spec.SecurityContext, r.Configuration.Jenkins.Spec.Master.SecurityContext))
	}

	if !compareImagePullSecrets(r.Configuration.Jenkins.Spec.Master.ImagePullSecrets, currentJenkinsMasterPod.Spec.ImagePullSecrets) {
		messages = append(messages, "Jenkins Pod ImagePullSecrets has changed")
		verbose = append(verbose, fmt.Sprintf("Jenkins Pod ImagePullSecrets has changed, actual '%+v' required '%+v'",
			currentJenkinsMasterPod.Spec.ImagePullSecrets, r.Configuration.Jenkins.Spec.Master.ImagePullSecrets))
	}

	if !compareMap(r.Configuration.Jenkins.Spec.Master.NodeSelector, currentJenkinsMasterPod.Spec.NodeSelector) {
		messages = append(messages, "Jenkins pod node selector has changed")
		verbose = append(verbose, fmt.Sprintf("Jenkins pod node selector has changed, actual '%+v' required '%+v'",
			currentJenkinsMasterPod.Spec.NodeSelector, r.Configuration.Jenkins.Spec.Master.NodeSelector))
	}

	if !compareMap(r.Configuration.Jenkins.Spec.Master.Labels, currentJenkinsMasterPod.Labels) {
		messages = append(messages, "Jenkins pod labels have changed")
		verbose = append(verbose, fmt.Sprintf("Jenkins pod labels have changed, actual '%+v' required '%+v'",
			currentJenkinsMasterPod.Labels, r.Configuration.Jenkins.Spec.Master.Labels))
	}

	if !compareMap(r.Configuration.Jenkins.Spec.Master.Annotations, currentJenkinsMasterPod.ObjectMeta.Annotations) {
		messages = append(messages, "Jenkins pod annotations have changed")
		verbose = append(verbose, fmt.Sprintf("Jenkins pod annotations have changed, actual '%+v' required '%+v'",
			currentJenkinsMasterPod.ObjectMeta.Annotations, r.Configuration.Jenkins.Spec.Master.Annotations))
	}

	if !r.compareVolumes(currentJenkinsMasterPod) {
		messages = append(messages, "Jenkins pod volumes have changed")
		verbose = append(verbose, fmt.Sprintf("Jenkins pod volumes have changed, actual '%v' required '%v'",
			currentJenkinsMasterPod.Spec.Volumes, r.Configuration.Jenkins.Spec.Master.Volumes))
	}

	if len(r.Configuration.Jenkins.Spec.Master.Containers) != len(currentJenkinsMasterPod.Spec.Containers) {
		messages = append(messages, "Jenkins amount of containers has changed")
		verbose = append(verbose, fmt.Sprintf("Jenkins amount of containers has changed, actual '%+v' required '%+v'",
			len(currentJenkinsMasterPod.Spec.Containers), len(r.Configuration.Jenkins.Spec.Master.Containers)))
	}

	customResourceReplaced := (r.Configuration.Jenkins.Status.BaseConfigurationCompletedTime == nil ||
		r.Configuration.Jenkins.Status.UserConfigurationCompletedTime == nil) &&
		r.Configuration.Jenkins.Status.UserAndPasswordHash == ""

	if customResourceReplaced {
		messages = append(messages, "Jenkins CR has been replaced")
		verbose = append(verbose, "Jenkins CR has been replaced")
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
			messages = append(messages, fmt.Sprintf("Container '%s' not found in pod", actualContainer.Name))
			verbose = append(verbose, fmt.Sprintf("Container '%+v' not found in pod", actualContainer))
			continue
		}

		containerMessages, verboseMessages := r.compareContainers(*expectedContainer, actualContainer)

		messages = append(messages, containerMessages...)
		verbose = append(verbose, verboseMessages...)
	}

	return reason.NewPodRestart(reason.OperatorSource, messages, verbose...)
}

func (r *ReconcileJenkinsBaseConfiguration) compareContainers(expected corev1.Container, actual corev1.Container) (messages []string, verbose []string) {
	if !reflect.DeepEqual(expected.Args, actual.Args) {
		messages = append(messages, "Arguments have changed")
		verbose = append(messages, fmt.Sprintf("Arguments have changed to '%+v' in container '%s'", expected.Args, expected.Name))
	}
	if !reflect.DeepEqual(expected.Command, actual.Command) {
		messages = append(messages, "Command has changed")
		verbose = append(verbose, fmt.Sprintf("Command has changed to '%+v' in container '%s'", expected.Command, expected.Name))
	}
	if !compareEnv(expected.Env, actual.Env) {
		messages = append(messages, "Env has changed")
		verbose = append(verbose, fmt.Sprintf("Env has changed to '%+v' in container '%s'", expected.Env, expected.Name))
	}
	if !reflect.DeepEqual(expected.EnvFrom, actual.EnvFrom) {
		messages = append(messages, "EnvFrom has changed")
		verbose = append(verbose, fmt.Sprintf("EnvFrom has changed to '%+v' in container '%s'", expected.EnvFrom, expected.Name))
	}
	if !reflect.DeepEqual(expected.Image, actual.Image) {
		messages = append(messages, "Image has changed")
		verbose = append(verbose, fmt.Sprintf("Image has changed to '%+v' in container '%s'", expected.Image, expected.Name))
	}
	if !reflect.DeepEqual(expected.ImagePullPolicy, actual.ImagePullPolicy) {
		messages = append(messages, "Image pull policy has changed")
		verbose = append(verbose, fmt.Sprintf("Image pull policy has changed to '%+v' in container '%s'", expected.ImagePullPolicy, expected.Name))
	}
	if !reflect.DeepEqual(expected.Lifecycle, actual.Lifecycle) {
		messages = append(messages, "Lifecycle has changed")
		verbose = append(verbose, fmt.Sprintf("Lifecycle has changed to '%+v' in container '%s'", expected.Lifecycle, expected.Name))
	}
	if !reflect.DeepEqual(expected.LivenessProbe, actual.LivenessProbe) {
		messages = append(messages, "Liveness probe has changed")
		verbose = append(verbose, fmt.Sprintf("Liveness probe has changed to '%+v' in container '%s'", expected.LivenessProbe, expected.Name))
	}
	if !reflect.DeepEqual(expected.Ports, actual.Ports) {
		messages = append(messages, "Ports have changed")
		verbose = append(verbose, fmt.Sprintf("Ports have changed to '%+v' in container '%s'", expected.Ports, expected.Name))
	}
	if !reflect.DeepEqual(expected.ReadinessProbe, actual.ReadinessProbe) {
		messages = append(messages, "Readiness probe has changed")
		verbose = append(verbose, fmt.Sprintf("Readiness probe has changed to '%+v' in container '%s'", expected.ReadinessProbe, expected.Name))
	}
	if !compareContainerResources(expected.Resources, actual.Resources) {
		messages = append(messages, "Resources have changed")
		verbose = append(verbose, fmt.Sprintf("Resources have changed to '%+v' in container '%s'", expected.Resources, expected.Name))
	}
	if !reflect.DeepEqual(expected.SecurityContext, actual.SecurityContext) {
		messages = append(messages, "Security context has changed")
		verbose = append(verbose, fmt.Sprintf("Security context has changed to '%+v' in container '%s'", expected.SecurityContext, expected.Name))
	}
	if !reflect.DeepEqual(expected.WorkingDir, actual.WorkingDir) {
		messages = append(messages, "Working directory has changed")
		verbose = append(verbose, fmt.Sprintf("Working directory has changed to '%+v' in container '%s'", expected.WorkingDir, expected.Name))
	}
	if !CompareContainerVolumeMounts(expected, actual) {
		messages = append(messages, "Volume mounts have changed")
		verbose = append(verbose, fmt.Sprintf("Volume mounts have changed to '%+v' in container '%s'", expected.VolumeMounts, expected.Name))
	}

	return messages, verbose
}

func compareContainerResources(expected corev1.ResourceRequirements, actual corev1.ResourceRequirements) bool {
	expectedRequestCPU, expectedRequestCPUSet := expected.Requests[corev1.ResourceCPU]
	expectedRequestMemory, expectedRequestMemorySet := expected.Requests[corev1.ResourceMemory]
	expectedLimitCPU, expectedLimitCPUSet := expected.Limits[corev1.ResourceCPU]
	expectedLimitMemory, expectedLimitMemorySet := expected.Limits[corev1.ResourceMemory]

	actualRequestCPU, actualRequestCPUSet := actual.Requests[corev1.ResourceCPU]
	actualRequestMemory, actualRequestMemorySet := actual.Requests[corev1.ResourceMemory]
	actualLimitCPU, actualLimitCPUSet := actual.Limits[corev1.ResourceCPU]
	actualLimitMemory, actualLimitMemorySet := actual.Limits[corev1.ResourceMemory]

	if expectedRequestCPUSet && (!actualRequestCPUSet || expectedRequestCPU.String() != actualRequestCPU.String()) {
		return false
	}
	if expectedRequestMemorySet && (!actualRequestMemorySet || expectedRequestMemory.String() != actualRequestMemory.String()) {
		return false
	}
	if expectedLimitCPUSet && (!actualLimitCPUSet || expectedLimitCPU.String() != actualLimitCPU.String()) {
		return false
	}
	if expectedLimitMemorySet && (!actualLimitMemorySet || expectedLimitMemory.String() != actualLimitMemory.String()) {
		return false
	}

	return true
}

func compareImagePullSecrets(expected, actual []corev1.LocalObjectReference) bool {
	for _, expected := range expected {
		found := false
		for _, actual := range actual {
			if expected.Name == actual.Name {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func compareMap(expected, actual map[string]string) bool {
	for expectedKey, expectedValue := range expected {
		actualValue, found := actual[expectedKey]
		if !found {
			return false
		}
		if expectedValue != actualValue {
			return false
		}
	}

	return true
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

func (r *ReconcileJenkinsBaseConfiguration) detectJenkinsMasterPodStartingIssues() (stopReconcileLoop bool, err error) {
	jenkinsMasterPod, err := r.getJenkinsMasterPod()
	if err != nil {
		return false, err
	}

	if r.Configuration.Jenkins.Status.ProvisionStartTime == nil {
		return true, nil
	}

	if jenkinsMasterPod.Status.Phase == corev1.PodPending {
		timeout := r.Configuration.Jenkins.Status.ProvisionStartTime.Add(time.Minute * 2).UTC()
		now := time.Now().UTC()
		if now.After(timeout) {
			events := &corev1.EventList{}
			err = r.Client.List(context.TODO(), events, client.InNamespace(r.Configuration.Jenkins.Namespace))
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
	for _, eventItem := range source.Items {
		if r.Configuration.Jenkins.Status.ProvisionStartTime.UTC().After(eventItem.LastTimestamp.UTC()) {
			continue
		}
		if eventItem.Type == corev1.EventTypeNormal {
			continue
		}
		if !strings.HasPrefix(eventItem.ObjectMeta.Name, jenkinsMasterPod.Name) {
			continue
		}
		events = append(events, fmt.Sprintf("Message: %s Subobject: %s", eventItem.Message, eventItem.InvolvedObject.FieldPath))
	}

	return events
}

func (r *ReconcileJenkinsBaseConfiguration) waitForJenkins() (reconcile.Result, error) {
	jenkinsMasterPod, err := r.getJenkinsMasterPod()
	if err != nil {
		return reconcile.Result{}, err
	}

	if r.IsJenkinsTerminating(*jenkinsMasterPod) {
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
			message := fmt.Sprintf("Container '%s' is terminated, status '%+v'", containerStatus.Name, containerStatus)
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

func (r *ReconcileJenkinsBaseConfiguration) ensureJenkinsClient() (jenkinsclient.Jenkins, error) {
	switch r.Configuration.Jenkins.Spec.JenkinsAPISettings.AuthorizationStrategy {
	case v1alpha2.ServiceAccountAuthorizationStrategy:
		return r.ensureJenkinsClientFromServiceAccount()
	case v1alpha2.CreateUserAuthorizationStrategy:
		return r.ensureJenkinsClientFromSecret()
	default:
		return nil, stackerr.Errorf("unrecognized '%s' spec.jenkinsAPISettings.authorizationStrategy", r.Configuration.Jenkins.Spec.JenkinsAPISettings.AuthorizationStrategy)
	}
}

func (r *ReconcileJenkinsBaseConfiguration) getJenkinsAPIUrl() (string, error) {
	var service corev1.Service

	err := r.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: r.Configuration.Jenkins.ObjectMeta.Namespace,
		Name:      resources.GetJenkinsHTTPServiceName(r.Configuration.Jenkins),
	}, &service)

	if err != nil {
		return "", err
	}

	jenkinsURL := r.jenkinsAPIConnectionSettings.BuildJenkinsAPIUrl(service.Name, service.Namespace, service.Spec.Ports[0].Port, service.Spec.Ports[0].NodePort)

	if prefix, ok := GetJenkinsOpts(*r.Configuration.Jenkins)["prefix"]; ok {
		jenkinsURL = jenkinsURL + prefix
	}

	return jenkinsURL, nil
}

func (r *ReconcileJenkinsBaseConfiguration) ensureJenkinsClientFromServiceAccount() (jenkinsclient.Jenkins, error) {
	jenkinsAPIUrl, err := r.getJenkinsAPIUrl()
	if err != nil {
		return nil, err
	}

	podName := resources.GetJenkinsMasterPodName(*r.Configuration.Jenkins)
	token, _, err := r.Configuration.Exec(podName, resources.JenkinsMasterContainerName, []string{"cat", "/var/run/secrets/kubernetes.io/serviceaccount/token"})
	if err != nil {
		return nil, err
	}

	return jenkinsclient.NewBearerTokenAuthorization(jenkinsAPIUrl, token.String())
}

func (r *ReconcileJenkinsBaseConfiguration) ensureJenkinsClientFromSecret() (jenkinsclient.Jenkins, error) {
	jenkinsURL, err := r.getJenkinsAPIUrl()
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
		jenkinsClient, err := jenkinsclient.NewUserAndPasswordAuthorization(
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
		err = r.UpdateResource(credentialsSecret)
		if err != nil {
			return nil, stackerr.WithStack(err)
		}
	}

	return jenkinsclient.NewUserAndPasswordAuthorization(
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

func (r *ReconcileJenkinsBaseConfiguration) waitUntilCreateJenkinsMasterPod() (currentJenkinsMasterPod *corev1.Pod, err error) {
	currentJenkinsMasterPod, err = r.getJenkinsMasterPod()
	for {
		if err != nil && !apierrors.IsNotFound(err) {
			return nil, stackerr.WithStack(err)
		} else if err == nil {
			break
		}
		currentJenkinsMasterPod, err = r.getJenkinsMasterPod()
		time.Sleep(time.Millisecond * 10)
	}

	return
}

func (r *ReconcileJenkinsBaseConfiguration) handleAdmissionControllerChanges(currentJenkinsMasterPod *corev1.Pod) {
	if !reflect.DeepEqual(r.Configuration.Jenkins.Spec.Master.SecurityContext, currentJenkinsMasterPod.Spec.SecurityContext) {
		r.Configuration.Jenkins.Spec.Master.SecurityContext = currentJenkinsMasterPod.Spec.SecurityContext
		r.logger.Info(fmt.Sprintf("The Admission controller has changed the Jenkins master pod spec.securityContext, changing the Jenkinc CR spec.master.securityContext to '%+v'", currentJenkinsMasterPod.Spec.SecurityContext))
	}
	for i, container := range r.Configuration.Jenkins.Spec.Master.Containers {
		if !reflect.DeepEqual(container.SecurityContext, currentJenkinsMasterPod.Spec.Containers[i].SecurityContext) {
			r.Configuration.Jenkins.Spec.Master.Containers[i].SecurityContext = currentJenkinsMasterPod.Spec.Containers[i].SecurityContext
			r.logger.Info(fmt.Sprintf("The Admission controller has changed the securityContext, changing the Jenkins CR spec.master.containers[%s].securityContext to '+%v'", container.Name, currentJenkinsMasterPod.Spec.Containers[i].SecurityContext))
		}
	}
}
