package user

import (
	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/backuprestore"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/user/seedjobs"
)

// Validate validates Jenkins CR Spec section
func (r *ReconcileUserConfiguration) Validate(jenkins *v1alpha2.Jenkins) (bool, error) {
	backupAndRestore := backuprestore.New(r.k8sClient, r.clientSet, r.logger, r.jenkins, r.config)
	if ok := backupAndRestore.Validate(); !ok {
		return false, nil
	}

	seedJobs := seedjobs.New(r.jenkinsClient, r.k8sClient, r.logger)
	return seedJobs.ValidateSeedJobs(*jenkins)
}
