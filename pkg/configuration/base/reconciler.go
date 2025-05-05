package base

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/constants"
	"github.com/jenkinsci/kubernetes-operator/pkg/groovy"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/reason"

	"github.com/go-logr/logr"
	stackerr "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	fetchAllPlugins = 1
)

// ReconcileJenkinsBaseConfiguration defines values required for Jenkins base configuration.
type JenkinsBaseConfigurationReconciler struct {
	configuration.Configuration
	logger                       logr.Logger
	jenkinsAPIConnectionSettings jenkinsclient.JenkinsAPIConnectionSettings
}

// New create structure which takes care of base configuration
func New(config configuration.Configuration, jenkinsAPIConnectionSettings jenkinsclient.JenkinsAPIConnectionSettings) *JenkinsBaseConfigurationReconciler {
	return &JenkinsBaseConfigurationReconciler{
		Configuration:                config,
		logger:                       log.Log.WithValues("cr", config.Jenkins.Name),
		jenkinsAPIConnectionSettings: jenkinsAPIConnectionSettings,
	}
}

// Reconcile takes care of base configuration.
func (r *JenkinsBaseConfigurationReconciler) Reconcile() (reconcile.Result, jenkinsclient.Jenkins, error) {
	metaObject := resources.NewResourceObjectMeta(r.Configuration.Jenkins)

	// Create Necessary Resources
	err := r.ensureResourcesRequiredForJenkinsPod(metaObject)
	if err != nil {
		return reconcile.Result{}, nil, err
	}
	r.logger.V(log.VDebug).Info("Kubernetes resources are present")

	if useDeploymentForJenkinsMaster(r.Configuration.Jenkins) {
		result, err := r.ensureJenkinsDeployment(metaObject)
		if err != nil {
			return reconcile.Result{}, nil, err
		}
		if result.Requeue {
			return result, nil, nil
		}
		r.logger.V(log.VDebug).Info("Jenkins Deployment is present")

		return result, nil, err
	}

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

	jenkinsClient, err := r.Configuration.GetJenkinsClient()
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
	if err != nil {
		return reconcile.Result{}, nil, err
	}

	return result, jenkinsClient, err
}

func useDeploymentForJenkinsMaster(jenkins *v1alpha2.Jenkins) bool {
	if val, ok := jenkins.Annotations["jenkins.io/use-deployment"]; ok {
		if val == "true" {
			return true
		}
	}
	return false
}

func (r *JenkinsBaseConfigurationReconciler) ensureResourcesRequiredForJenkinsPod(metaObject metav1.ObjectMeta) error {
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

	httpServiceName := resources.GetJenkinsHTTPServiceName(r.Configuration.Jenkins)
	if err := r.createService(metaObject, httpServiceName, r.Configuration.Jenkins.Spec.Service, constants.DefaultHTTPPortInt32); err != nil {
		return err
	}
	r.logger.V(log.VDebug).Info("Jenkins HTTP Service is present")

	if err := r.createService(metaObject, resources.GetJenkinsSlavesServiceName(r.Configuration.Jenkins), r.Configuration.Jenkins.Spec.SlaveService, constants.DefaultSlavePortInt32); err != nil {
		return err
	}
	r.logger.V(log.VDebug).Info("Jenkins slave Service is present")

	if resources.IsRouteAPIAvailable(&r.ClientSet) {
		r.logger.V(log.VDebug).Info("Route API is available. Now creating route.")
		if err := r.createRoute(metaObject, httpServiceName, r.Configuration.Jenkins); err != nil {
			return err
		}
		r.logger.V(log.VDebug).Info("Jenkins Route is present")
	}

	return nil
}

func (r *JenkinsBaseConfigurationReconciler) createOperatorCredentialsSecret(meta metav1.ObjectMeta) error {
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

func (r *JenkinsBaseConfigurationReconciler) calculateUserAndPasswordHash() (string, error) {
	credentialsSecret := &corev1.Secret{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: resources.GetOperatorCredentialsSecretName(r.Configuration.Jenkins), Namespace: r.Configuration.Jenkins.ObjectMeta.Namespace}, credentialsSecret)
	if err != nil {
		return "", stackerr.WithStack(err)
	}

	hash := sha256.New()
	_, err = hash.Write(credentialsSecret.Data[resources.OperatorCredentialsSecretUserNameKey])
	if err != nil {
		return "", stackerr.WithStack(err)
	}
	_, err = hash.Write(credentialsSecret.Data[resources.OperatorCredentialsSecretPasswordKey])
	if err != nil {
		return "", stackerr.WithStack(err)
	}
	return base64.StdEncoding.EncodeToString(hash.Sum(nil)), nil
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

// CompareContainerVolumeMounts returns true if two containers volume mounts are the same.
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
func (r *JenkinsBaseConfigurationReconciler) compareVolumes(actualPod corev1.Pod) bool {
	var toCompare []corev1.Volume
	for _, volume := range actualPod.Spec.Volumes {

		if r.isVolumeIgnored(volume.Name) {
			continue
		}

		// filter out service account
		if strings.HasPrefix(volume.Name, actualPod.Spec.ServiceAccountName) {
			continue
		}

		// hotfix for k8s 1.21 - filter out kube-api-access-<random-suffix>
		const kubeAPIAccessPrefix = "kube-api-access-"
		if strings.HasPrefix(volume.Name, kubeAPIAccessPrefix) {
			continue
		}

		toCompare = append(toCompare, volume)
	}

	return reflect.DeepEqual(
		append(resources.GetJenkinsMasterPodBaseVolumes(r.Configuration.Jenkins), r.Configuration.Jenkins.Spec.Master.Volumes...),
		toCompare,
	)
}

func (r *JenkinsBaseConfigurationReconciler) detectJenkinsMasterPodStartingIssues() (stopReconcileLoop bool, err error) {
	jenkinsMasterPod, err := r.Configuration.GetJenkinsMasterPod()
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

func (r *JenkinsBaseConfigurationReconciler) filterEvents(source corev1.EventList, jenkinsMasterPod corev1.Pod) []string {
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

func (r *JenkinsBaseConfigurationReconciler) waitForJenkins() (reconcile.Result, error) {
	jenkinsMasterPod, err := r.Configuration.GetJenkinsMasterPod()
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

func (r *JenkinsBaseConfigurationReconciler) ensureBaseConfiguration(jenkinsClient jenkinsclient.Jenkins) (reconcile.Result, error) {
	customization := v1alpha2.GroovyScripts{
		Customization: v1alpha2.Customization{
			Secret:         v1alpha2.SecretRef{Name: ""},
			Configurations: []v1alpha2.ConfigMapRef{{Name: resources.GetBaseConfigurationConfigMapName(r.Configuration.Jenkins)}},
		},
	}
	groovyClient := groovy.New(jenkinsClient, r.Client, r.Configuration.Jenkins, "base-groovy", customization.Customization)
	requeue, err := groovyClient.Ensure(func(name string) bool {
		return strings.HasSuffix(name, ".groovy")
	}, func(groovyScript string) string {
		return groovyScript
	})
	return reconcile.Result{Requeue: requeue}, err
}

// isVolumeIgnored checks if the given volume name is in the list of ignored volumes
func (r *JenkinsBaseConfigurationReconciler) isVolumeIgnored(volumeName string) bool {
	for _, ignoredVolume := range r.Jenkins.Spec.Master.IgnoredVolumes {
		if ignoredVolume == volumeName {
			return true
		}
	}
	return false
}
