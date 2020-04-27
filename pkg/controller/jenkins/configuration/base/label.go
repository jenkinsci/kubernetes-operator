package base

import (
	"context"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/base/resources"

	stackerr "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (r *ReconcileJenkinsBaseConfiguration) addLabelForWatchesResources(customization v1alpha2.Customization) error {
	labelsForWatchedResources := resources.BuildLabelsForWatchedResources(*r.Configuration.Jenkins)

	if len(customization.Secret.Name) > 0 {
		secret := &corev1.Secret{}
		err := r.Client.Get(context.TODO(), types.NamespacedName{Name: customization.Secret.Name, Namespace: r.Configuration.Jenkins.Namespace}, secret)
		if err != nil {
			return stackerr.WithStack(err)
		}

		if !resources.VerifyIfLabelsAreSet(secret, labelsForWatchedResources) {
			if len(secret.ObjectMeta.Labels) == 0 {
				secret.ObjectMeta.Labels = map[string]string{}
			}
			for key, value := range labelsForWatchedResources {
				secret.ObjectMeta.Labels[key] = value
			}

			if err = r.Client.Update(context.TODO(), secret); err != nil {
				return stackerr.WithStack(r.Client.Update(context.TODO(), secret))
			}
		}
	}

	for _, configMapRef := range customization.Configurations {
		configMap := &corev1.ConfigMap{}
		err := r.Client.Get(context.TODO(), types.NamespacedName{Name: configMapRef.Name, Namespace: r.Configuration.Jenkins.Namespace}, configMap)
		if err != nil {
			return stackerr.WithStack(err)
		}

		if !resources.VerifyIfLabelsAreSet(configMap, labelsForWatchedResources) {
			if len(configMap.ObjectMeta.Labels) == 0 {
				configMap.ObjectMeta.Labels = map[string]string{}
			}
			for key, value := range labelsForWatchedResources {
				configMap.ObjectMeta.Labels[key] = value
			}

			if err = r.Client.Update(context.TODO(), configMap); err != nil {
				return stackerr.WithStack(r.Client.Update(context.TODO(), configMap))
			}
		}
	}
	return nil
}
