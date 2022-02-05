package configuration

import (
	"bytes"
	"context"
	"time"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/event"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/reason"

	stackerr "github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Configuration holds required for Jenkins configuration.
type Configuration struct {
	Client                       client.Client
	ClientSet                    kubernetes.Clientset
	Notifications                *chan event.Event
	Jenkins                      *v1alpha2.Jenkins
	Scheme                       *runtime.Scheme
	Config                       *rest.Config
	JenkinsAPIConnectionSettings jenkinsclient.JenkinsAPIConnectionSettings
	KubernetesClusterDomain      string
}

// RestartJenkinsMasterPod terminate Jenkins master pod and notifies about it.
func (c *Configuration) RestartJenkinsMasterPod(reason reason.Reason) error {
	currentJenkinsMasterPod, err := c.GetJenkinsMasterPod()
	if err != nil {
		return err
	}

	if c.IsJenkinsTerminating(*currentJenkinsMasterPod) {
		return nil
	}

	*c.Notifications <- event.Event{
		Jenkins: *c.Jenkins,
		Phase:   event.PhaseBase,
		Level:   v1alpha2.NotificationLevelInfo,
		Reason:  reason,
	}

	return stackerr.WithStack(c.Client.Delete(context.TODO(), currentJenkinsMasterPod))
}

// GetJenkinsMasterPod gets the jenkins master pod.
func (c *Configuration) GetJenkinsMasterPod() (*corev1.Pod, error) {
	jenkinsMasterPodName := resources.GetJenkinsMasterPodName(c.Jenkins)
	currentJenkinsMasterPod := &corev1.Pod{}
	err := c.Client.Get(context.TODO(), types.NamespacedName{Name: jenkinsMasterPodName, Namespace: c.Jenkins.Namespace}, currentJenkinsMasterPod)
	if err != nil {
		return nil, err // don't wrap error
	}
	return currentJenkinsMasterPod, nil
}

// GetJenkinsMasterPod gets the jenkins master pod.
func (c *Configuration) GetJenkinsDeployment() (*appsv1.Deployment, error) {
	jenkinsDeploymentName := resources.GetJenkinsDeploymentName(c.Jenkins)
	currentJenkinsDeployment := &appsv1.Deployment{}
	err := c.Client.Get(context.TODO(), types.NamespacedName{Name: jenkinsDeploymentName, Namespace: c.Jenkins.Namespace}, currentJenkinsDeployment)
	if err != nil {
		return nil, stackerr.WithStack(err)
	}
	return currentJenkinsDeployment, nil
}

// IsJenkinsTerminating returns true if the Jenkins pod is terminating.
func (c *Configuration) IsJenkinsTerminating(pod corev1.Pod) bool {
	return pod.ObjectMeta.DeletionTimestamp != nil
}

// CreateResource is creating kubernetes resource and references it to Jenkins CR
func (c *Configuration) CreateResource(obj metav1.Object) error {
	clientObj, ok := obj.(client.Object)
	if !ok {
		return stackerr.Errorf("is not a %T a runtime.Object", obj)
	}

	// Set Jenkins instance as the owner and controller.
	if err := controllerutil.SetControllerReference(c.Jenkins, obj, c.Scheme); err != nil {
		return stackerr.WithStack(err)
	}

	return c.Client.Create(context.TODO(), clientObj) // don't wrap error
}

// UpdateResource is updating kubernetes resource and references it to Jenkins CR.
func (c *Configuration) UpdateResource(obj metav1.Object) error {
	clientObj, ok := obj.(client.Object)
	if !ok {
		return stackerr.Errorf("is not a %T a runtime.Object", obj)
	}

	// set Jenkins instance as the owner and controller, don't check errors(can be already set)
	_ = controllerutil.SetControllerReference(c.Jenkins, obj, c.Scheme)

	return c.Client.Update(context.TODO(), clientObj) // don't wrap error
}

// CreateOrUpdateResource is creating or updating kubernetes resource and references it to Jenkins CR.
func (c *Configuration) CreateOrUpdateResource(obj metav1.Object) error {
	clientObj, ok := obj.(client.Object)
	if !ok {
		return stackerr.Errorf("is not a %T a runtime.Object", obj)
	}

	// set Jenkins instance as the owner and controller, don't check error(can be already set)
	_ = controllerutil.SetControllerReference(c.Jenkins, obj, c.Scheme)

	err := c.Client.Create(context.TODO(), clientObj)
	if err != nil && errors.IsAlreadyExists(err) {
		return c.UpdateResource(obj)
	} else if err != nil && !errors.IsAlreadyExists(err) {
		return stackerr.WithStack(err)
	}

	return nil
}

// Exec executes command in the given pod and it's container.
func (c *Configuration) Exec(podName, containerName string, command []string) (stdout, stderr bytes.Buffer, err error) {
	req := c.ClientSet.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(c.Jenkins.Namespace).
		SubResource("exec")
	req.VersionedParams(&corev1.PodExecOptions{
		Command:   command,
		Container: containerName,
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(c.Config, "POST", req.URL())
	if err != nil {
		return stdout, stderr, stackerr.Wrap(err, "pod exec error while creating Executor")
	}

	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})
	if err != nil {
		return stdout, stderr, stackerr.Wrapf(err, "pod exec error operation on stream: stdout '%s' stderr '%s'", stdout.String(), stderr.String())
	}

	return
}

// GetJenkinsMasterContainer returns the Jenkins master container from the CR.
func (c *Configuration) GetJenkinsMasterContainer() *v1alpha2.Container {
	if len(c.Jenkins.Spec.Master.Containers) > 0 {
		// the first container is the Jenkins master, it is forced jenkins_controller.go
		return &c.Jenkins.Spec.Master.Containers[0]
	}
	return nil
}

// GetJenkinsClient gets jenkins client from a configuration.
func (c *Configuration) GetJenkinsClient() (jenkinsclient.Jenkins, error) {
	switch c.Jenkins.Spec.JenkinsAPISettings.AuthorizationStrategy {
	case v1alpha2.ServiceAccountAuthorizationStrategy:
		return c.GetJenkinsClientFromServiceAccount()
	case v1alpha2.CreateUserAuthorizationStrategy:
		return c.GetJenkinsClientFromSecret()
	default:
		return nil, stackerr.Errorf("unrecognized '%s' spec.jenkinsAPISettings.authorizationStrategy", c.Jenkins.Spec.JenkinsAPISettings.AuthorizationStrategy)
	}
}

func (c *Configuration) getJenkinsAPIUrl() (string, error) {
	var service corev1.Service

	err := c.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: c.Jenkins.ObjectMeta.Namespace,
		Name:      resources.GetJenkinsHTTPServiceName(c.Jenkins),
	}, &service)

	if err != nil {
		return "", err
	}
	jenkinsURL := c.JenkinsAPIConnectionSettings.BuildJenkinsAPIUrl(service.Name, service.Namespace, service.Spec.Ports[0].Port, service.Spec.Ports[0].NodePort)
	if prefix, ok := resources.GetJenkinsOpts(*c.Jenkins)["prefix"]; ok {
		jenkinsURL += prefix
	}
	return jenkinsURL, nil
}

// GetJenkinsClientFromServiceAccount gets jenkins client from a serviceAccount.
func (c *Configuration) GetJenkinsClientFromServiceAccount() (jenkinsclient.Jenkins, error) {
	jenkinsAPIUrl, err := c.getJenkinsAPIUrl()
	if err != nil {
		return nil, err
	}

	podName := resources.GetJenkinsMasterPodName(c.Jenkins)
	token, _, err := c.Exec(podName, resources.JenkinsMasterContainerName, []string{"cat", "/var/run/secrets/kubernetes.io/serviceaccount/token"})
	if err != nil {
		return nil, err
	}

	return jenkinsclient.NewBearerTokenAuthorization(jenkinsAPIUrl, token.String())
}

// GetJenkinsClientFromSecret gets jenkins client from a secret.
func (c *Configuration) GetJenkinsClientFromSecret() (jenkinsclient.Jenkins, error) {
	jenkinsURL, err := c.getJenkinsAPIUrl()
	if err != nil {
		return nil, err
	}
	credentialsSecret := &corev1.Secret{}
	err = c.Client.Get(context.TODO(), types.NamespacedName{Name: resources.GetOperatorCredentialsSecretName(c.Jenkins), Namespace: c.Jenkins.ObjectMeta.Namespace}, credentialsSecret)
	if err != nil {
		return nil, stackerr.WithStack(err)
	}
	currentJenkinsMasterPod, err := c.GetJenkinsMasterPod()
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
		err = c.UpdateResource(credentialsSecret)
		if err != nil {
			return nil, stackerr.WithStack(err)
		}
	}
	return jenkinsclient.NewUserAndPasswordAuthorization(
		jenkinsURL,
		string(credentialsSecret.Data[resources.OperatorCredentialsSecretUserNameKey]),
		string(credentialsSecret.Data[resources.OperatorCredentialsSecretTokenKey]))
}
