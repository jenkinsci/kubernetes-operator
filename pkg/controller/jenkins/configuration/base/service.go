package base

import (
	"context"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/base/resources"

	stackerr "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (r *ReconcileJenkinsBaseConfiguration) createService(meta metav1.ObjectMeta, name string, config v1alpha2.Service) error {
	service := corev1.Service{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: meta.Namespace}, &service)
	if err != nil && apierrors.IsNotFound(err) {
		service = resources.UpdateService(corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: meta.Namespace,
				Labels:    meta.Labels,
			},
			Spec: corev1.ServiceSpec{
				Selector: meta.Labels,
			},
		}, config)
		if err = r.CreateResource(&service); err != nil {
			return stackerr.WithStack(err)
		}
	} else if err != nil {
		return stackerr.WithStack(err)
	}

	service.Spec.Selector = meta.Labels // make sure that user won't break service by hand
	service = resources.UpdateService(service, config)
	return stackerr.WithStack(r.UpdateResource(&service))
}
