package base

import (
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/base/resources"

	stackerr "github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *ReconcileJenkinsBaseConfiguration) createScriptsConfigMap(meta metav1.ObjectMeta) error {
	configMap, err := resources.NewScriptsConfigMap(meta, r.Configuration.Jenkins)
	if err != nil {
		return err
	}
	return stackerr.WithStack(r.CreateOrUpdateResource(configMap))
}

func (r *ReconcileJenkinsBaseConfiguration) createInitConfigurationConfigMap(meta metav1.ObjectMeta) error {
	configMap, err := resources.NewInitConfigurationConfigMap(meta, r.Configuration.Jenkins)
	if err != nil {
		return err
	}
	return stackerr.WithStack(r.CreateOrUpdateResource(configMap))
}

func (r *ReconcileJenkinsBaseConfiguration) createBaseConfigurationConfigMap(meta metav1.ObjectMeta) error {
	configMap, err := resources.NewBaseConfigurationConfigMap(meta, r.Configuration.Jenkins)
	if err != nil {
		return err
	}
	return stackerr.WithStack(r.CreateOrUpdateResource(configMap))
}
