/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"math/rand"
	"reflect"
	"time"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/user"
	"github.com/jenkinsci/kubernetes-operator/pkg/constants"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/event"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/reason"
	"github.com/jenkinsci/kubernetes-operator/pkg/plugins"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type reconcileError struct {
	err     error
	counter uint64
}

const (
	APIVersion             = "core/v1"
	SecretKind             = "Secret"
	ConfigMapKind          = "ConfigMap"
	containerProbeURI      = "login"
	containerProbePortName = "http"
)

var reconcileErrors = map[string]reconcileError{}
var logx = log.Log

// JenkinsReconciler reconciles a Jenkins object
type JenkinsReconciler struct {
	Client                       client.Client
	Scheme                       *runtime.Scheme
	JenkinsAPIConnectionSettings jenkinsclient.JenkinsAPIConnectionSettings
	ClientSet                    kubernetes.Clientset
	Config                       rest.Config
	NotificationEvents           *chan event.Event
	KubernetesClusterDomain      string
}

// SetupWithManager sets up the controller with the Manager.
func (r *JenkinsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	jenkinsHandler := &enqueueRequestForJenkins{}
	configMapResource := &source.Kind{Type: &corev1.ConfigMap{TypeMeta: metav1.TypeMeta{APIVersion: APIVersion, Kind: ConfigMapKind}}}
	secretResource := &source.Kind{Type: &corev1.Secret{TypeMeta: metav1.TypeMeta{APIVersion: APIVersion, Kind: SecretKind}}}
	decorator := jenkinsDecorator{handler: &handler.EnqueueRequestForObject{}}
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha2.Jenkins{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ConfigMap{}).
		Watches(secretResource, jenkinsHandler).
		Watches(configMapResource, jenkinsHandler).
		Watches(&source.Kind{Type: &v1alpha2.Jenkins{}}, &decorator).
		Complete(r)
}

func (r *JenkinsReconciler) newJenkinsReconcilier(jenkins *v1alpha2.Jenkins) configuration.Configuration {
	config := configuration.Configuration{
		Client:                       r.Client,
		ClientSet:                    r.ClientSet,
		Notifications:                r.NotificationEvents,
		Jenkins:                      jenkins,
		Scheme:                       r.Scheme,
		Config:                       &r.Config,
		JenkinsAPIConnectionSettings: r.JenkinsAPIConnectionSettings,
		KubernetesClusterDomain:      r.KubernetesClusterDomain,
	}
	return config
}

// +kubebuilder:rbac:groups=jenkins.io,resources=jenkins,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=jenkins.io,resources=jenkins/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=jenkins.io,resources=jenkins/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services;configmaps;secrets,verbs=get;list;watch;create;update
// +kubebuilder:rbac:groups=apps,resources=deployments;daemonsets;replicasets;statefulsets,verbs=*
// +kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=get;list;watch;create;update
// +kubebuilder:rbac:groups=core,resources=pods/portforward,verbs=create
// +kubebuilder:rbac:groups=core,resources=pods/log,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=pods;pods/exec,verbs=*
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;watch;list;create;patch
// +kubebuilder:rbac:groups=apps;jenkins-operator,resources=deployments/finalizers,verbs=update
// +kubebuilder:rbac:groups=jenkins.io,resources=*,verbs=*
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch
// +kubebuilder:rbac:groups=route.openshift.io,resources=routes,verbs=get;list;watch;create;update
// +kubebuilder:rbac:groups=image.openshift.io,resources=imagestreams,verbs=get;list;watch
// +kubebuilder:rbac:groups=build.openshift.io,resources=builds;buildconfigs,verbs=get;list;watch

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *JenkinsReconciler) Reconcile(_ context.Context, request ctrl.Request) (ctrl.Result, error) {
	reconcileFailLimit := uint64(10)
	logger := logx.WithValues("cr", request.Name)
	logger.V(log.VDebug).Info("Reconciling Jenkins")

	result, jenkins, err := r.reconcile(request)
	if err != nil && apierrors.IsConflict(err) {
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		lastErrors, found := reconcileErrors[request.Name]
		if found {
			if err.Error() == lastErrors.err.Error() {
				lastErrors.counter++
			} else {
				lastErrors.counter = 1
				lastErrors.err = err
			}
		} else {
			lastErrors = reconcileError{
				err:     err,
				counter: 1,
			}
		}
		reconcileErrors[request.Name] = lastErrors
		if lastErrors.counter >= reconcileFailLimit {
			if log.Debug {
				logger.V(log.VWarn).Info(fmt.Sprintf("Reconcile loop failed %d times with the same errors, giving up: %+v", reconcileFailLimit, err))
			} else {
				logger.V(log.VWarn).Info(fmt.Sprintf("Reconcile loop failed %d times with the same errors, giving up: %s", reconcileFailLimit, err))
			}

			*r.NotificationEvents <- event.Event{
				Jenkins: *jenkins,
				Phase:   event.PhaseBase,
				Level:   v1alpha2.NotificationLevelWarning,
				Reason: reason.NewReconcileLoopFailed(
					reason.OperatorSource,
					[]string{fmt.Sprintf("Reconcile loop failed %d times with the same errors, giving up: %s", reconcileFailLimit, err)},
				),
			}
			return reconcile.Result{Requeue: false}, nil
		}

		if log.Debug {
			logger.V(log.VWarn).Info(fmt.Sprintf("Reconcile loop failed: %+v", err))
		} else if err.Error() != fmt.Sprintf("Operation cannot be fulfilled on jenkins.jenkins.io \"%s\": the object has been modified; please apply your changes to the latest version and try again", request.Name) {
			logger.V(log.VWarn).Info(fmt.Sprintf("Reconcile loop failed: %s", err))
		}

		if groovyErr, ok := err.(*jenkinsclient.GroovyScriptExecutionFailed); ok {
			*r.NotificationEvents <- event.Event{
				Jenkins: *jenkins,
				Phase:   event.PhaseBase,
				Level:   v1alpha2.NotificationLevelWarning,
				Reason: reason.NewGroovyScriptExecutionFailed(
					reason.OperatorSource,
					[]string{fmt.Sprintf("%s Source '%s' Name '%s' groovy script execution failed", groovyErr.ConfigurationType, groovyErr.Source, groovyErr.Name)},
					[]string{fmt.Sprintf("%s Source '%s' Name '%s' groovy script execution failed, logs: %+v", groovyErr.ConfigurationType, groovyErr.Source, groovyErr.Name, groovyErr.Logs)}...,
				),
			}
			return reconcile.Result{Requeue: false}, nil
		}
		return reconcile.Result{Requeue: true}, nil
	}
	if result.Requeue && result.RequeueAfter == 0 {
		result.RequeueAfter = time.Duration(rand.Intn(10)) * time.Millisecond
	}
	return result, nil
}

func (r *JenkinsReconciler) reconcile(request reconcile.Request) (reconcile.Result, *v1alpha2.Jenkins, error) {
	logger := logx.WithValues("cr", request.Name)
	// Fetch the Jenkins instance
	jenkins := &v1alpha2.Jenkins{}
	var err error
	err = r.Client.Get(context.TODO(), request.NamespacedName, jenkins)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, nil, errors.WithStack(err)
	}
	var requeue bool
	requeue, err = r.setDefaults(jenkins)
	if err != nil {
		return reconcile.Result{}, jenkins, err
	}
	if requeue {
		return reconcile.Result{Requeue: true}, jenkins, nil
	}

	config := r.newJenkinsReconcilier(jenkins)
	// Reconcile base configuration
	baseConfiguration := base.New(config, r.JenkinsAPIConnectionSettings)

	var baseMessages []string
	baseMessages, err = baseConfiguration.Validate(jenkins)
	if err != nil {
		return reconcile.Result{}, jenkins, err
	}
	if len(baseMessages) > 0 {
		message := "Validation of base configuration failed, please correct Jenkins CR."
		*r.NotificationEvents <- event.Event{
			Jenkins: *jenkins,
			Phase:   event.PhaseBase,
			Level:   v1alpha2.NotificationLevelWarning,
			Reason:  reason.NewBaseConfigurationFailed(reason.HumanSource, []string{message}, append([]string{message}, baseMessages...)...),
		}
		logger.V(log.VWarn).Info(message)
		for _, msg := range baseMessages {
			logger.V(log.VWarn).Info(msg)
		}
		return reconcile.Result{}, jenkins, nil // don't requeue
	}

	var result reconcile.Result
	var jenkinsClient jenkinsclient.Jenkins
	result, jenkinsClient, err = baseConfiguration.Reconcile()
	if err != nil {
		return reconcile.Result{}, jenkins, err
	}
	if result.Requeue {
		return result, jenkins, nil
	}
	if jenkinsClient == nil {
		return reconcile.Result{Requeue: false}, jenkins, nil
	}

	if jenkins.Status.BaseConfigurationCompletedTime == nil {
		now := metav1.Now()
		jenkins.Status.BaseConfigurationCompletedTime = &now
		err = r.Client.Status().Update(context.TODO(), jenkins)
		if err != nil {
			return reconcile.Result{}, jenkins, errors.WithStack(err)
		}

		message := fmt.Sprintf("Base configuration phase is complete, took %s",
			jenkins.Status.BaseConfigurationCompletedTime.Sub(jenkins.Status.ProvisionStartTime.Time))
		*r.NotificationEvents <- event.Event{
			Jenkins: *jenkins,
			Phase:   event.PhaseBase,
			Level:   v1alpha2.NotificationLevelInfo,
			Reason:  reason.NewBaseConfigurationComplete(reason.OperatorSource, []string{message}),
		}
		logger.Info(message)
	}

	// Reconcile casc, seedjobs and backups
	userConfiguration := user.New(config, jenkinsClient)

	var messages []string
	messages, err = userConfiguration.Validate(jenkins)
	if err != nil {
		return reconcile.Result{}, jenkins, err
	}
	if len(messages) > 0 {
		message := "Validation of user configuration failed, please correct Jenkins CR"
		*r.NotificationEvents <- event.Event{
			Jenkins: *jenkins,
			Phase:   event.PhaseUser,
			Level:   v1alpha2.NotificationLevelWarning,
			Reason:  reason.NewUserConfigurationFailed(reason.HumanSource, []string{message}, append([]string{message}, messages...)...),
		}

		logger.V(log.VWarn).Info(message)
		for _, msg := range messages {
			logger.V(log.VWarn).Info(msg)
		}
		return reconcile.Result{}, jenkins, nil // don't requeue
	}

	// Reconcile casc
	result, err = userConfiguration.ReconcileCasc()
	if err != nil {
		return reconcile.Result{}, jenkins, err
	}
	if result.Requeue {
		return result, jenkins, nil
	}

	// Reconcile seedjobs, backups
	result, err = userConfiguration.ReconcileOthers()
	if err != nil {
		return reconcile.Result{}, jenkins, err
	}
	if result.Requeue {
		return result, jenkins, nil
	}

	if jenkins.Status.UserConfigurationCompletedTime == nil {
		now := metav1.Now()
		jenkins.Status.UserConfigurationCompletedTime = &now
		err = r.Client.Status().Update(context.TODO(), jenkins)
		if err != nil {
			return reconcile.Result{}, jenkins, errors.WithStack(err)
		}
		message := fmt.Sprintf("User configuration phase is complete, took %s",
			jenkins.Status.UserConfigurationCompletedTime.Sub(jenkins.Status.ProvisionStartTime.Time))
		*r.NotificationEvents <- event.Event{
			Jenkins: *jenkins,
			Phase:   event.PhaseUser,
			Level:   v1alpha2.NotificationLevelInfo,
			Reason:  reason.NewUserConfigurationComplete(reason.OperatorSource, []string{message}),
		}
		logger.Info(message)
	}
	return reconcile.Result{}, jenkins, nil
}

func (r *JenkinsReconciler) setDefaults(jenkins *v1alpha2.Jenkins) (requeue bool, err error) {
	changed := false
	logger := logx.WithValues("cr", jenkins.Name)

	var jenkinsContainer v1alpha2.Container
	if len(jenkins.Spec.Master.Containers) == 0 {
		changed = true
		jenkinsContainer = v1alpha2.Container{Name: resources.JenkinsMasterContainerName}
	} else {
		if jenkins.Spec.Master.Containers[0].Name != resources.JenkinsMasterContainerName {
			return false, errors.Errorf("first container in spec.master.containers must be Jenkins container with name '%s', please correct CR", resources.JenkinsMasterContainerName)
		}
		jenkinsContainer = jenkins.Spec.Master.Containers[0]
	}

	if len(jenkinsContainer.Image) == 0 {
		logger.Info("Setting default Jenkins master image: " + constants.DefaultJenkinsMasterImage)
		changed = true
		jenkinsContainer.Image = constants.DefaultJenkinsMasterImage
		jenkinsContainer.ImagePullPolicy = corev1.PullAlways
	}
	if len(jenkinsContainer.ImagePullPolicy) == 0 {
		logger.Info(fmt.Sprintf("Setting default Jenkins master image pull policy: %s", corev1.PullAlways))
		changed = true
		jenkinsContainer.ImagePullPolicy = corev1.PullAlways
	}

	if jenkinsContainer.ReadinessProbe == nil {
		logger.Info("Setting default Jenkins readinessProbe")
		changed = true
		jenkinsContainer.ReadinessProbe = resources.NewProbe(containerProbeURI, containerProbePortName, corev1.URISchemeHTTP, 60, 1, 10)
	}
	if jenkinsContainer.LivenessProbe == nil {
		logger.Info("Setting default Jenkins livenessProbe")
		changed = true
		jenkinsContainer.LivenessProbe = resources.NewProbe(containerProbeURI, containerProbePortName, corev1.URISchemeHTTP, 80, 5, 12)
	}
	if jenkinsContainer.Lifecycle == nil {
		logger.Info("Setting default Jenkins lifecycle")
		changed = true
		jenkinsContainer.Lifecycle = &corev1.Lifecycle{}
	}
	if len(jenkinsContainer.Command) == 0 {
		logger.Info("Setting default Jenkins container command")
		changed = true
		jenkinsContainer.Command = resources.GetJenkinsMasterContainerBaseCommand()
	}
	if isJavaOpsVariableNotSet(jenkinsContainer) {
		logger.Info("Setting default Jenkins container JAVA_OPTS environment variable")
		changed = true
		jenkinsContainer.Env = append(jenkinsContainer.Env, corev1.EnvVar{
			Name:  constants.JavaOpsVariableName,
			Value: "-XX:MinRAMPercentage=50.0 -XX:MaxRAMPercentage=80.0 -Djenkins.install.runSetupWizard=false -Djava.awt.headless=true",
		})
	}
	if len(jenkins.Spec.Master.BasePlugins) == 0 {
		logger.Info("Setting default operator plugins")
		changed = true
		jenkins.Spec.Master.BasePlugins = basePlugins()
	}
	if isResourceRequirementsNotSet(jenkinsContainer.Resources) {
		logger.Info("Setting default Jenkins master container resource requirements")
		changed = true
		jenkinsContainer.Resources = resources.NewResourceRequirements("1", "500Mi", "1500m", "3Gi")
	}
	if reflect.DeepEqual(jenkins.Spec.Service, v1alpha2.Service{}) {
		logger.Info("Setting default Jenkins master service")
		changed = true
		var serviceType = corev1.ServiceTypeClusterIP
		if r.JenkinsAPIConnectionSettings.UseNodePort {
			serviceType = corev1.ServiceTypeNodePort
		}
		jenkins.Spec.Service = v1alpha2.Service{
			Type: serviceType,
			Port: constants.DefaultHTTPPortInt32,
		}
	}
	if reflect.DeepEqual(jenkins.Spec.SlaveService, v1alpha2.Service{}) {
		logger.Info("Setting default Jenkins slave service")
		changed = true
		jenkins.Spec.SlaveService = v1alpha2.Service{
			Type: corev1.ServiceTypeClusterIP,
			Port: constants.DefaultSlavePortInt32,
		}
	}
	if len(jenkins.Spec.Master.Containers) > 1 {
		for i, container := range jenkins.Spec.Master.Containers[1:] {
			if r.setDefaultsForContainer(jenkins, container.Name, i+1) {
				changed = true
			}
		}
	}
	if len(jenkins.Spec.Backup.ContainerName) > 0 && jenkins.Spec.Backup.Interval == 0 {
		logger.Info("Setting default backup interval")
		changed = true
		jenkins.Spec.Backup.Interval = 30
	}

	if len(jenkins.Spec.Master.Containers) == 0 || len(jenkins.Spec.Master.Containers) == 1 {
		jenkins.Spec.Master.Containers = []v1alpha2.Container{jenkinsContainer}
	} else {
		noJenkinsContainers := jenkins.Spec.Master.Containers[1:]
		containers := []v1alpha2.Container{jenkinsContainer}
		containers = append(containers, noJenkinsContainers...)
		jenkins.Spec.Master.Containers = containers
	}

	if reflect.DeepEqual(jenkins.Spec.JenkinsAPISettings, v1alpha2.JenkinsAPISettings{}) {
		logger.Info("Setting default Jenkins API settings")
		changed = true
		jenkins.Spec.JenkinsAPISettings = v1alpha2.JenkinsAPISettings{AuthorizationStrategy: v1alpha2.CreateUserAuthorizationStrategy}
	}

	if jenkins.Spec.JenkinsAPISettings.AuthorizationStrategy == "" {
		logger.Info("Setting default Jenkins API settings authorization strategy")
		changed = true
		jenkins.Spec.JenkinsAPISettings.AuthorizationStrategy = v1alpha2.CreateUserAuthorizationStrategy
	}

	if changed {
		return changed, errors.WithStack(r.Client.Update(context.TODO(), jenkins))
	}
	return changed, nil
}

func isJavaOpsVariableNotSet(container v1alpha2.Container) bool {
	for _, env := range container.Env {
		if env.Name == constants.JavaOpsVariableName {
			return false
		}
	}
	return true
}

func (r *JenkinsReconciler) setDefaultsForContainer(jenkins *v1alpha2.Jenkins, containerName string, containerIndex int) bool {
	changed := false
	logger := logx.WithValues("cr", jenkins.Name, "container", containerName)

	if len(jenkins.Spec.Master.Containers[containerIndex].ImagePullPolicy) == 0 {
		logger.Info(fmt.Sprintf("Setting default container image pull policy: %s", corev1.PullAlways))
		changed = true
		jenkins.Spec.Master.Containers[containerIndex].ImagePullPolicy = corev1.PullAlways
	}
	if isResourceRequirementsNotSet(jenkins.Spec.Master.Containers[containerIndex].Resources) {
		logger.Info("Setting default container resource requirements")
		changed = true
		jenkins.Spec.Master.Containers[containerIndex].Resources = resources.NewResourceRequirements("50m", "50Mi", "100m", "100Mi")
	}
	return changed
}

func isResourceRequirementsNotSet(requirements corev1.ResourceRequirements) bool {
	return reflect.DeepEqual(requirements, corev1.ResourceRequirements{})
}

func basePlugins() (result []v1alpha2.Plugin) {
	for _, value := range plugins.BasePlugins() {
		result = append(result, v1alpha2.Plugin{Name: value.Name, Version: value.Version})
	}
	return
}
