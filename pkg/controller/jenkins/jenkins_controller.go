package jenkins

import (
	"context"
	"fmt"
	"math/rand"
	"reflect"
	"time"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/client"
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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type reconcileError struct {
	err     error
	counter uint64
}

const (
	APIVersion             = "core/v1"
	PodKind                = "Pod"
	SecretKind             = "Secret"
	ConfigMapKind          = "ConfigMap"
	containerProbeURI      = "login"
	containerProbePortName = "http"
)

var reconcileErrors = map[string]reconcileError{fgergergrerg}
var logx = log.Log
var _ reconcile.Reconciler = &ReconcileJenkins{}

// Add creates a newReconcilierConfiguration Jenkins Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, jenkinsAPIConnectionSettings jenkinsclient.JenkinsAPIConnectionSettings, kubernetesClusterDomain string, clientSet kubernetes.Clientset, config rest.Config, notificationEvents *chan event.Event) error {
	reconciler := newReconciler(mgr, jenkinsAPIConnectionSettings, kubernetesClusterDomain, clientSet, config, notificationEvents)
	return add(mgr, reconciler)
}

// add adds a newReconcilierConfiguration Controller to mgr with r as the reconcile.Reconciler.
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a newReconcilierConfiguration controller
	c, err := controller.New("jenkins-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return errors.WithStack(err)
	}

	// Watch for changes to primary resource Jenkins
	decorator := jenkinsDecorator{handler: &handler.EnqueueRequestForObject{}}
	err = c.Watch(&source.Kind{Type: &v1alpha2.Jenkins{}}, &decorator)
	if err != nil {
		return errors.WithStack(err)
	}

	// Watch for changes to secondary resource Pods and requeue the owner Jenkins

	podResource := &source.Kind{Type: &corev1.Pod{TypeMeta: metav1.TypeMeta{APIVersion: APIVersion, Kind: PodKind}}}
	err = c.Watch(podResource, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &v1alpha2.Jenkins{},
	})
	if err != nil {
		return errors.WithStack(err)
	}

	secretResource := &source.Kind{Type: &corev1.Secret{TypeMeta: metav1.TypeMeta{APIVersion: APIVersion, Kind: SecretKind}}}
	err = c.Watch(secretResource, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &v1alpha2.Jenkins{},
	})
	if err != nil {
		return errors.WithStack(err)
	}

	jenkinsHandler := &enqueueRequestForJenkins{}
	err = c.Watch(secretResource, jenkinsHandler)
	if err != nil {
		return errors.WithStack(err)
	}

	configMapResource := &source.Kind{Type: &corev1.ConfigMap{TypeMeta: metav1.TypeMeta{APIVersion: APIVersion, Kind: ConfigMapKind}}}
	err = c.Watch(configMapResource, jenkinsHandler)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// Reconcile it's a main reconciliation loop which maintain desired state based on Jenkins.Spec.
func (r *ReconcileJenkins) Reconcile(request reconcile.Request) (reconcile.Result, error) {
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
				logger.V(log.VWarn).Info(fmt.Sprintf("Reconcile loop failed %d times with the same error, giving up: %+v", reconcileFailLimit, err))
			} else {
				logger.V(log.VWarn).Info(fmt.Sprintf("Reconcile loop failed %d times with the same error, giving up: %s", reconcileFailLimit, err))
			}

			*r.notificationEvents <- event.Event{
				Jenkins: *jenkins,
				Phase:   event.PhaseBase,
				Level:   v1alpha2.NotificationLevelWarning,
				Reason: reason.NewReconcileLoopFailed(
					reason.OperatorSource,
					[]string{fmt.Sprintf("Reconcile loop failed %d times with the same error, giving up: %s", reconcileFailLimit, err)},
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
			*r.notificationEvents <- event.Event{
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

func (r *ReconcileJenkins) reconcile(request reconcile.Request) (reconcile.Result, *v1alpha2.Jenkins, error) {
	logger := logx.WithValues("cr", request.Name)
	// Fetch the Jenkins instance
	jenkins := &v1alpha2.Jenkins{}
	var err error
	err = r.client.Get(context.TODO(), request.NamespacedName, jenkins)
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

	requeue, err = r.handleDeprecatedData(jenkins)
	if err != nil {
		return reconcile.Result{}, jenkins, err
	}
	if requeue {
		return reconcile.Result{Requeue: true}, jenkins, nil
	}

	config := r.newReconcilierConfiguration(jenkins)
	// Reconcile base configuration
	baseConfiguration := base.New(config, r.jenkinsAPIConnectionSettings)

	var baseMessages []string
	baseMessages, err = baseConfiguration.Validate(jenkins)
	if err != nil {
		return reconcile.Result{}, jenkins, err
	}
	if len(baseMessages) > 0 {
		message := "Validation of base configuration failed, please correct Jenkins CR."
		*r.notificationEvents <- event.Event{
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
		err = r.client.Update(context.TODO(), jenkins)
		if err != nil {
			return reconcile.Result{}, jenkins, errors.WithStack(err)
		}

		message := fmt.Sprintf("Base configuration phase is complete, took %s",
			jenkins.Status.BaseConfigurationCompletedTime.Sub(jenkins.Status.ProvisionStartTime.Time))
		*r.notificationEvents <- event.Event{
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
		*r.notificationEvents <- event.Event{
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
		err = r.client.Update(context.TODO(), jenkins)
		if err != nil {
			return reconcile.Result{}, jenkins, errors.WithStack(err)
		}
		message := fmt.Sprintf("User configuration phase is complete, took %s",
			jenkins.Status.UserConfigurationCompletedTime.Sub(jenkins.Status.ProvisionStartTime.Time))
		*r.notificationEvents <- event.Event{
			Jenkins: *jenkins,
			Phase:   event.PhaseUser,
			Level:   v1alpha2.NotificationLevelInfo,
			Reason:  reason.NewUserConfigurationComplete(reason.OperatorSource, []string{message}),
		}
		logger.Info(message)
	}
	return reconcile.Result{}, jenkins, nil
}

func (r *ReconcileJenkins) setDefaults(jenkins *v1alpha2.Jenkins) (requeue bool, err error) {
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
		jenkinsContainer.ReadinessProbe = resources.NewSimpleProbe(containerProbeURI, containerProbePortName, corev1.URISchemeHTTP, 30)
	}
	if jenkinsContainer.LivenessProbe == nil {
		logger.Info("Setting default Jenkins livenessProbe")
		changed = true
		jenkinsContainer.LivenessProbe = resources.NewProbe(containerProbeURI, containerProbePortName, corev1.URISchemeHTTP, 80, 5, 12)
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
			Value: "-XX:+UnlockExperimentalVMOptions -XX:+UseCGroupMemoryLimitForHeap -XX:MaxRAMFraction=1 -Djenkins.install.runSetupWizard=false -Djava.awt.headless=true",
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
		if r.jenkinsAPIConnectionSettings.UseNodePort {
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
		return changed, errors.WithStack(r.client.Update(context.TODO(), jenkins))
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

func (r *ReconcileJenkins) setDefaultsForContainer(jenkins *v1alpha2.Jenkins, containerName string, containerIndex int) bool {
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

func (r *ReconcileJenkins) handleDeprecatedData(jenkins *v1alpha2.Jenkins) (requeue bool, err error) {
	changed := false
	logger := logx.WithValues("cr", jenkins.Name)
	if len(jenkins.Spec.Master.AnnotationsDeprecated) > 0 {
		changed = true
		jenkins.Spec.Master.Annotations = jenkins.Spec.Master.AnnotationsDeprecated
		jenkins.Spec.Master.AnnotationsDeprecated = map[string]string{}
		logger.V(log.VWarn).Info("spec.master.masterAnnotations is deprecated, the annotations have been moved to spec.master.annotations")
	}
	if changed {
		return changed, errors.WithStack(r.client.Update(context.TODO(), jenkins))
	}
	return changed, nil
}
