package base

import (
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"

	stackerr "github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *JenkinsBaseConfigurationReconciler) createScriptsConfigMap(meta metav1.ObjectMeta) error {
	configMap, err := resources.NewScriptsConfigMap(meta, r.Jenkins)
	if err != nil {
		return err
	}
	return stackerr.WithStack(r.CreateOrUpdateResource(configMap))
}

func (r *JenkinsBaseConfigurationReconciler) createInitConfigurationConfigMap(meta metav1.ObjectMeta) error {
	configMap, err := resources.NewInitConfigurationConfigMap(meta, r.Jenkins)
	if err != nil {
		return err
	}
	return stackerr.WithStack(r.CreateOrUpdateResource(configMap))
}

func (r *JenkinsBaseConfigurationReconciler) createBaseConfigurationConfigMap(meta metav1.ObjectMeta) error {
	configMap, err := resources.NewBaseConfigurationConfigMap(meta, r.Jenkins, r.KubernetesClusterDomain)
	if err != nil {
		return err
	}
	return stackerr.WithStack(r.CreateOrUpdateResource(configMap))
}
