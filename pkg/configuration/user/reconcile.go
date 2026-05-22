package user

import (
	"strings"
	"time"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/backuprestore"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/user/casc"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/user/seedjobs"
	"github.com/jenkinsci/kubernetes-operator/pkg/groovy"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ReconcileUserConfiguration defines API client for reconcile User Configuration
type ReconcileUserConfiguration interface {
	ReconcileCasc() (reconcile.Result, error)
	ReconcileOthers() (reconcile.Result, error)
	Validate(jenkins *v1alpha2.Jenkins) ([]string, error)
}

type reconcileUserConfiguration struct {
	configuration.Configuration
	jenkinsClient jenkinsclient.Jenkins
	logger        logr.Logger
}

// New create structure which takes care of user configuration.
func New(configuration configuration.Configuration, jenkinsClient jenkinsclient.Jenkins) ReconcileUserConfiguration {
	return &reconcileUserConfiguration{
		Configuration: configuration,
		jenkinsClient: jenkinsClient,
		logger:        log.Log.WithValues("cr", configuration.Jenkins.Name),
	}
}

// ReconcileCasc is a reconcile loop for casc.
func (r *reconcileUserConfiguration) ReconcileCasc() (reconcile.Result, error) {
	result, err := r.ensureCasc(r.jenkinsClient)
	if err != nil {
		return reconcile.Result{}, err
	}
	if result.RequeueAfter > 0 {
		return result, nil
	}

	return reconcile.Result{}, nil
}

// Reconcile it's a main reconciliation loop for user supplied configuration
func (r *reconcileUserConfiguration) ReconcileOthers() (reconcile.Result, error) {
	backupAndRestore := backuprestore.New(r.Configuration, r.logger)

	if err := backupAndRestore.Restore(r.jenkinsClient); err != nil {
		return reconcile.Result{}, err
	}

	if err := backupAndRestore.Backup(false); err != nil {
		return reconcile.Result{}, err
	}

	if err := backupAndRestore.EnsureBackupTrigger(); err != nil {
		return reconcile.Result{}, err
	}

	result, err := r.ensureSeedJobs()
	if err != nil {
		return reconcile.Result{}, err
	}
	if result.RequeueAfter > 0 {
		return result, nil
	}

	return reconcile.Result{}, nil
}

func (r *reconcileUserConfiguration) ensureSeedJobs() (reconcile.Result, error) {
	seedJobs := seedjobs.New(r.jenkinsClient, r.Configuration)
	done, err := seedJobs.EnsureSeedJobs(r.Jenkins)
	if err != nil {
		return reconcile.Result{}, err
	}
	if !done {
		return reconcile.Result{RequeueAfter: time.Second * 30}, nil
	}
	return reconcile.Result{}, nil
}

func (r *reconcileUserConfiguration) ensureCasc(jenkinsClient jenkinsclient.Jenkins) (reconcile.Result, error) {
	configurationAsCodeClient := casc.New(jenkinsClient, r.Client, r.Jenkins)
	requeue, err := configurationAsCodeClient.Ensure(r.Jenkins)
	if err != nil {
		return reconcile.Result{}, err
	}
	if requeue {
		return reconcile.Result{Requeue: true}, nil
	}

	groovyClient := groovy.New(jenkinsClient, r.Client, r.Jenkins, "user-groovy", r.Jenkins.Spec.GroovyScripts.Customization)
	requeue, err = groovyClient.WaitForSecretSynchronization(resources.GroovyScriptsSecretVolumePath)
	if err != nil {
		return reconcile.Result{}, err
	}
	if requeue {
		return reconcile.Result{Requeue: true}, nil
	}
	requeue, err = groovyClient.Ensure(func(name string) bool {
		return strings.HasSuffix(name, ".groovy")
	}, groovy.AddSecretsLoaderToGroovyScript(resources.GroovyScriptsSecretVolumePath))
	if err != nil {
		return reconcile.Result{}, err
	}
	if requeue {
		return reconcile.Result{Requeue: true}, nil
	}

	return reconcile.Result{}, nil
}
