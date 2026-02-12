package base

import (
	"context"
	"fmt"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/event"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/reason"
	"github.com/jenkinsci/kubernetes-operator/version"

	stackerr "github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *JenkinsBaseConfigurationReconciler) ensureJenkinsDeployment(meta metav1.ObjectMeta) (reconcile.Result, error) {
	userAndPasswordHash, err := r.calculateUserAndPasswordHash()
	if err != nil {
		return reconcile.Result{}, err
	}

	_, err = r.GetJenkinsDeployment()
	if apierrors.IsNotFound(err) {
		jenkinsDeployment := resources.NewJenkinsDeployment(meta, r.Jenkins)
		*r.Notifications <- event.Event{
			Jenkins: *r.Jenkins,
			Phase:   event.PhaseBase,
			Level:   v1alpha2.NotificationLevelInfo,
			Reason:  reason.NewPodCreation(reason.OperatorSource, []string{"Creating a Jenkins Deployment"}),
		}

		r.logger.Info(fmt.Sprintf("Creating a new Jenkins Deployment %s/%s", jenkinsDeployment.Namespace, jenkinsDeployment.Name))
		err := r.CreateResource(jenkinsDeployment)
		if err != nil {
			return reconcile.Result{}, stackerr.WithStack(err)
		}

		now := metav1.Now()
		r.Jenkins.Status = v1alpha2.JenkinsStatus{
			OperatorVersion:     version.Version,
			ProvisionStartTime:  &now,
			LastBackup:          r.Jenkins.Status.LastBackup,
			PendingBackup:       r.Jenkins.Status.LastBackup,
			UserAndPasswordHash: userAndPasswordHash,
		}
		return reconcile.Result{Requeue: true}, r.Client.Update(context.TODO(), r.Jenkins)
	} else if err != nil && !apierrors.IsNotFound(err) {
		return reconcile.Result{}, stackerr.WithStack(err)
	}

	return reconcile.Result{}, nil
}
