package base

import (
	"context"
	"fmt"

	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"

	stackerr "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (r *JenkinsBaseConfigurationReconciler) createServiceAccount(meta metav1.ObjectMeta) error {
	serviceAccount := &corev1.ServiceAccount{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: meta.Name, Namespace: meta.Namespace}, serviceAccount)
	annotations := r.Jenkins.Spec.ServiceAccount.Annotations
	msg := fmt.Sprintf("createServiceAccount with annotations %v", annotations)
	r.logger.V(log.VDebug).Info(msg)
	if err != nil && apierrors.IsNotFound(err) {
		serviceAccount = resources.NewServiceAccount(meta, annotations)
		if err = r.CreateResource(serviceAccount); err != nil {
			return stackerr.WithStack(err)
		}
	} else if err != nil {
		return stackerr.WithStack(err)
	}

	if !compareMap(annotations, serviceAccount.Annotations) {
		if serviceAccount.Annotations == nil {
			serviceAccount.Annotations = map[string]string{}
		}
		for key, value := range annotations {
			serviceAccount.Annotations[key] = value
		}
		if err = r.UpdateResource(serviceAccount); err != nil {
			return stackerr.WithStack(err)
		}
	}

	return nil
}
