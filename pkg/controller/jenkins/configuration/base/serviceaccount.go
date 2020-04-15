package base

import (
	"context"

	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/base/resources"
	
	stackerr "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

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
