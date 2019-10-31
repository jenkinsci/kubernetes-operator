package configuration

import (
	"context"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/notifications/event"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/notifications/reason"

	stackerr "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Configuration holds required for Jenkins configuration
type Configuration struct {
	Client        client.Client
	ClientSet     kubernetes.Clientset
	Notifications *chan event.Event
	Jenkins       *v1alpha2.Jenkins
}

// RestartJenkinsMasterPod terminate Jenkins master pod and notifies about it
func (c *Configuration) RestartJenkinsMasterPod(reason reason.Reason) error {
	currentJenkinsMasterPod, err := c.getJenkinsMasterPod()
	if err != nil {
		return err
	}

	*c.Notifications <- event.Event{
		Jenkins: *c.Jenkins,
		Phase:   event.PhaseBase,
		Level:   v1alpha2.NotificationLevelInfo,
		Reason:  reason,
	}

	return stackerr.WithStack(c.Client.Delete(context.TODO(), currentJenkinsMasterPod))
}

func (c *Configuration) getJenkinsMasterPod() (*corev1.Pod, error) {
	jenkinsMasterPodName := resources.GetJenkinsMasterPodName(*c.Jenkins)
	currentJenkinsMasterPod := &corev1.Pod{}
	err := c.Client.Get(context.TODO(), types.NamespacedName{Name: jenkinsMasterPodName, Namespace: c.Jenkins.Namespace}, currentJenkinsMasterPod)
	if err != nil {
		return nil, err // don't wrap error
	}
	return currentJenkinsMasterPod, nil
}
