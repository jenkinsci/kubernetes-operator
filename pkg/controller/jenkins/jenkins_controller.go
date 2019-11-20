package jenkins

import (
	"context"
	"fmt"
	"reflect"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/base"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/user"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/constants"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/notifications/event"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/notifications/reason"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/plugins"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"
	"github.com/jenkinsci/kubernetes-operator/version"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

var reconcileErrors = map[string]reconcileError{}

// Add creates a new Jenkins Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, hostname string, port int, useNodePort bool, clientSet kubernetes.Clientset, config rest.Config, notificationEvents *chan event.Event) error {
	return add(mgr, newReconciler(mgr, hostname, port, useNodePort, clientSet, config, notificationEvents))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, hostname string, port int, useNodePort bool, clientSet kubernetes.Clientset, config rest.Config, notificationEvents *chan event.Event) reconcile.Reconciler {
	return &ReconcileJenkins{
		client:             mgr.GetClient(),
		scheme:             mgr.GetScheme(),
		hostname:           hostname,
		port:               port,
		useNodePort:        useNodePort,
		clientSet:          clientSet,
		config:             config,
		notificationEvents: notificationEvents,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
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
	err = c.Watch(&source.Kind{Type: &corev1.Pod{TypeMeta: metav1.TypeMeta{APIVersion: "core/v1", Kind: "Pod"}}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &v1alpha2.Jenkins{},
	})
	if err != nil {
		return errors.WithStack(err)
	}

	err = c.Watch(&source.Kind{Type: &corev1.Secret{TypeMeta: metav1.TypeMeta{APIVersion: "core/v1", Kind: "Secret"}}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &v1alpha2.Jenkins{},
	})
	if err != nil {
		return errors.WithStack(err)
	}

	jenkinsHandler := &enqueueRequestForJenkins{}
	err = c.Watch(&source.Kind{Type: &corev1.Secret{TypeMeta: metav1.TypeMeta{APIVersion: "core/v1", Kind: "Secret"}}}, jenkinsHandler)
	if err != nil {
		return errors.WithStack(err)
	}

	err = c.Watch(&source.Kind{Type: &corev1.ConfigMap{TypeMeta: metav1.TypeMeta{APIVersion: "core/v1", Kind: "ConfigMap"}}}, jenkinsHandler)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileJenkins{}

// ReconcileJenkins reconciles a Jenkins object
type ReconcileJenkins struct {
	client             client.Client
	scheme             *runtime.Scheme
	hostname           string
	port               int
	useNodePort        bool
	clientSet          kubernetes.Clientset
	config             rest.Config
	notificationEvents *chan event.Event
}

// Reconcile it's a main reconciliation loop which maintain desired state based on Jenkins.Spec
func (r *ReconcileJenkins) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reconcileFailLimit := uint64(10)
	logger := r.buildLogger(request.Name)
	logger.V(log.VDebug).Info("Reconciling Jenkins")

	result, jenkins, err := r.reconcile(request, logger)
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
		} else {
			if err.Error() != fmt.Sprintf("Operation cannot be fulfilled on jenkins.jenkins.io \"%s\": the object has been modified; please apply your changes to the latest version and try again", request.Name) {
				logger.V(log.VWarn).Info(fmt.Sprintf("Reconcile loop failed: %s", err))
			}
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
	return result, nil
}

func (r *ReconcileJenkins) reconcile(request reconcile.Request, logger logr.Logger) (reconcile.Result, *v1alpha2.Jenkins, error) {
	// Fetch the Jenkins instance
	jenkins := &v1alpha2.Jenkins{}
	err := r.client.Get(context.TODO(), request.NamespacedName, jenkins)
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
	err = r.setDefaults(jenkins, logger)
	if err != nil {
		return reconcile.Result{}, jenkins, err
	}

	config := configuration.Configuration{
		Client:        r.client,
		ClientSet:     r.clientSet,
		Notifications: r.notificationEvents,
		Jenkins:       jenkins,
		Scheme:        r.scheme,
	}

	// Reconcile base configuration
	baseConfiguration := base.New(config, logger, r.hostname, r.port, r.useNodePort, &r.config)

	baseMessages, err := baseConfiguration.Validate(jenkins)
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

	result, jenkinsClient, err := baseConfiguration.Reconcile()
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
	// Reconcile user configuration
	userConfiguration := user.New(config, jenkinsClient, logger, r.config)

	messages, err := userConfiguration.Validate(jenkins)
	if err != nil {
		return reconcile.Result{}, jenkins, err
	}
	if len(messages) > 0 {
		message := fmt.Sprintf("Validation of user configuration failed, please correct Jenkins CR")
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

	result, err = userConfiguration.Reconcile()
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

func (r *ReconcileJenkins) buildLogger(jenkinsName string) logr.Logger {
	return log.Log.WithValues("cr", jenkinsName)
}

func (r *ReconcileJenkins) setDefaults(jenkins *v1alpha2.Jenkins, logger logr.Logger) error {
	changed := false

	var jenkinsContainer v1alpha2.Container
	if len(jenkins.Spec.Master.Containers) == 0 {
		changed = true
		jenkinsContainer = v1alpha2.Container{Name: resources.JenkinsMasterContainerName}
	} else {
		if jenkins.Spec.Master.Containers[0].Name != resources.JenkinsMasterContainerName {
			return errors.Errorf("first container in spec.master.containers must be Jenkins container with name '%s', please correct CR", resources.JenkinsMasterContainerName)
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
		jenkinsContainer.ReadinessProbe = &corev1.Probe{
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/login",
					Port:   intstr.FromString("http"),
					Scheme: corev1.URISchemeHTTP,
				},
			},
			InitialDelaySeconds: int32(30),
		}
	}
	if jenkinsContainer.LivenessProbe == nil {
		logger.Info("Setting default Jenkins livenessProbe")
		changed = true
		jenkinsContainer.LivenessProbe = &corev1.Probe{
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/login",
					Port:   intstr.FromString("http"),
					Scheme: corev1.URISchemeHTTP,
				},
			},
			InitialDelaySeconds: int32(80),
			TimeoutSeconds:      int32(5),
			FailureThreshold:    int32(12),
		}
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
	if len(jenkins.Status.OperatorVersion) > 0 && version.Version != jenkins.Status.OperatorVersion {
		logger.Info("Setting default operator plugins after Operator version change")
		changed = true
		jenkins.Spec.Master.BasePlugins = basePlugins()
	}
	if len(jenkins.Status.OperatorVersion) == 0 {
		logger.Info("Setting operator version")
		changed = true
		jenkins.Status.OperatorVersion = version.Version
	}
	if isResourceRequirementsNotSet(jenkinsContainer.Resources) {
		logger.Info("Setting default Jenkins master container resource requirements")
		changed = true
		jenkinsContainer.Resources = corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1"),
				corev1.ResourceMemory: resource.MustParse("500Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1500m"),
				corev1.ResourceMemory: resource.MustParse("3Gi"),
			},
		}
	}
	if reflect.DeepEqual(jenkins.Spec.Service, v1alpha2.Service{}) {
		logger.Info("Setting default Jenkins master service")
		changed = true
		jenkins.Spec.Service = v1alpha2.Service{
			Type: corev1.ServiceTypeClusterIP,
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
			if setDefaultsForContainer(jenkins, i+1, logger.WithValues("container", container.Name)) {
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

	if jenkins.Spec.Master.SecurityContext == nil {
		logger.Info("Setting default Jenkins master security context")
		changed = true
		var id int64 = 1000
		securityContext := corev1.PodSecurityContext{
			RunAsUser: &id,
			FSGroup:   &id,
		}
		jenkins.Spec.Master.SecurityContext = &securityContext
	}

	if changed {
		return errors.WithStack(r.client.Update(context.TODO(), jenkins))
	}
	return nil
}

func isJavaOpsVariableNotSet(container v1alpha2.Container) bool {
	for _, env := range container.Env {
		if env.Name == constants.JavaOpsVariableName {
			return false
		}
	}
	return true
}

func setDefaultsForContainer(jenkins *v1alpha2.Jenkins, containerIndex int, logger logr.Logger) bool {
	changed := false

	if len(jenkins.Spec.Master.Containers[containerIndex].ImagePullPolicy) == 0 {
		logger.Info(fmt.Sprintf("Setting default container image pull policy: %s", corev1.PullAlways))
		changed = true
		jenkins.Spec.Master.Containers[containerIndex].ImagePullPolicy = corev1.PullAlways
	}
	if isResourceRequirementsNotSet(jenkins.Spec.Master.Containers[containerIndex].Resources) {
		logger.Info("Setting default container resource requirements")
		changed = true
		jenkins.Spec.Master.Containers[containerIndex].Resources = corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("50m"),
				corev1.ResourceMemory: resource.MustParse("50Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("100Mi"),
			},
		}
	}

	return changed
}

func isResourceRequirementsNotSet(requirements corev1.ResourceRequirements) bool {
	_, requestCPUSet := requirements.Requests[corev1.ResourceCPU]
	_, requestMemorySet := requirements.Requests[corev1.ResourceMemory]
	_, limitCPUSet := requirements.Limits[corev1.ResourceCPU]
	_, limitMemorySet := requirements.Limits[corev1.ResourceMemory]

	return !limitCPUSet || !limitMemorySet || !requestCPUSet || !requestMemorySet
}

func basePlugins() (result []v1alpha2.Plugin) {
	for _, value := range plugins.BasePlugins() {
		result = append(result, v1alpha2.Plugin{Name: value.Name, Version: value.Version})
	}
	return
}
