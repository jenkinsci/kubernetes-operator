package base

import (
	"context"
	"fmt"
	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/backuprestore"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/event"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/reason"
	"github.com/jenkinsci/kubernetes-operator/version"
	"reflect"
	"slices"

	stackerr "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *JenkinsBaseConfigurationReconciler) checkForPodRecreation(currentJenkinsMasterPod corev1.Pod, userAndPasswordHash string) reason.Reason {
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

	//FIXME too hacky
	var jenkinsSecurityContext *corev1.PodSecurityContext
	if r.Configuration.Jenkins.Spec.Master.SecurityContext == nil {
		jenkinsSecurityContext = &corev1.PodSecurityContext{}
	} else {
		jenkinsSecurityContext = r.Configuration.Jenkins.Spec.Master.SecurityContext
	}
	if !reflect.DeepEqual(jenkinsSecurityContext, currentJenkinsMasterPod.Spec.SecurityContext) {
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

	if !r.compareAnnotations(currentJenkinsMasterPod) {
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

	if r.Configuration.Jenkins.Spec.Master.PriorityClassName != currentJenkinsMasterPod.Spec.PriorityClassName {
		messages = append(messages, "Jenkins priorityClassName has changed")
		verbose = append(verbose, fmt.Sprintf("Jenkins priorityClassName has changed, actual '%+v' required '%+v'",
			currentJenkinsMasterPod.Spec.PriorityClassName, r.Configuration.Jenkins.Spec.Master.PriorityClassName))
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

func (r *JenkinsBaseConfigurationReconciler) compareAnnotations(currentJenkinsMasterPod corev1.Pod) bool {
	ignoredAnnotations := r.Jenkins.Spec.Lifecycle.Ignore.IgnoredAnnotations
	annotations := r.Configuration.Jenkins.Spec.Master.Annotations

	res := make(map[string]string)
	for key, val := range annotations {
		if slices.Contains(ignoredAnnotations, key) {
			continue
		}
		res[key] = val
	}
	return compareMap(res, currentJenkinsMasterPod.ObjectMeta.Annotations)
}

func (r *JenkinsBaseConfigurationReconciler) ensureJenkinsMasterPod(meta metav1.ObjectMeta) (reconcile.Result, error) {
	userAndPasswordHash, err := r.calculateUserAndPasswordHash()
	if err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Pod already exists
	currentJenkinsMasterPod, err := r.Configuration.GetJenkinsMasterPod()
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

		now := metav1.Now()
		r.Configuration.Jenkins.Status = v1alpha2.JenkinsStatus{
			OperatorVersion:     version.Version,
			ProvisionStartTime:  &now,
			LastBackup:          r.Configuration.Jenkins.Status.LastBackup,
			PendingBackup:       r.Configuration.Jenkins.Status.LastBackup,
			UserAndPasswordHash: userAndPasswordHash,
		}
		return reconcile.Result{Requeue: true}, r.Client.Status().Update(context.TODO(), r.Configuration.Jenkins)
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
			return reconcile.Result{Requeue: true}, nil
		}
		if r.Configuration.Jenkins.Spec.Backup.MakeBackupBeforePodDeletion && !r.Configuration.Jenkins.Status.BackupDoneBeforePodDeletion {
			if r.Configuration.Jenkins.Status.LastBackup == r.Configuration.Jenkins.Status.PendingBackup {
				r.Configuration.Jenkins.Status.PendingBackup++
			}
			if err = backupAndRestore.Backup(true); err != nil {
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
