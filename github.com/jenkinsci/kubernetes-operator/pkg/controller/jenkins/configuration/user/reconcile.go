package user

import (
	"strings"
	"time"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/backuprestore"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/user/casc"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/user/seedjobs"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/groovy"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/jobs"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ReconcileUserConfiguration defines values required for Jenkins user configuration
type ReconcileUserConfiguration struct {
	k8sClient     k8s.Client
	jenkinsClient jenkinsclient.Jenkins
	logger        logr.Logger
	jenkins       *v1alpha2.Jenkins
	clientSet     kubernetes.Clientset
	config        rest.Config
}

// New create structure which takes care of user configuration
func New(k8sClient k8s.Client, jenkinsClient jenkinsclient.Jenkins, logger logr.Logger,
	jenkins *v1alpha2.Jenkins, clientSet kubernetes.Clientset, config rest.Config) *ReconcileUserConfiguration {
	return &ReconcileUserConfiguration{
		k8sClient:     k8sClient,
		jenkinsClient: jenkinsClient,
		logger:        logger,
		jenkins:       jenkins,
		clientSet:     clientSet,
		config:        config,
	}
}

// Reconcile it's a main reconciliation loop for user supplied configuration
func (r *ReconcileUserConfiguration) Reconcile() (reconcile.Result, error) {
	backupAndRestore := backuprestore.New(r.k8sClient, r.clientSet, r.logger, r.jenkins, r.config)

	result, err := r.ensureSeedJobs()
	if err != nil {
		return reconcile.Result{}, err
	}
	if result.Requeue {
		return result, nil
	}

	if err := backupAndRestore.Restore(r.jenkinsClient); err != nil {
		return reconcile.Result{}, err
	}

	result, err = r.ensureUserConfiguration(r.jenkinsClient)
	if err != nil {
		return reconcile.Result{}, err
	}
	if result.Requeue {
		return result, nil
	}

	if err := backupAndRestore.Backup(); err != nil {
		return reconcile.Result{}, err
	}
	if err := backupAndRestore.EnsureBackupTrigger(); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileUserConfiguration) ensureSeedJobs() (reconcile.Result, error) {
	seedJobs := seedjobs.New(r.jenkinsClient, r.k8sClient, r.logger)
	done, err := seedJobs.EnsureSeedJobs(r.jenkins)
	if err != nil {
		// build failed and can be recovered - retry build and requeue reconciliation loop with timeout
		if err == jobs.ErrorBuildFailed {
			return reconcile.Result{Requeue: true, RequeueAfter: time.Second * 10}, nil
		}
		// build failed and cannot be recovered
		if err == jobs.ErrorUnrecoverableBuildFailed {
			return reconcile.Result{}, nil
		}
		// unexpected error - requeue reconciliation loop
		return reconcile.Result{}, errors.WithStack(err)
	}
	// build not finished yet - requeue reconciliation loop with timeout
	if !done {
		return reconcile.Result{Requeue: true, RequeueAfter: time.Second * 5}, nil
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileUserConfiguration) ensureUserConfiguration(jenkinsClient jenkinsclient.Jenkins) (reconcile.Result, error) {
	groovyClient := groovy.New(jenkinsClient, r.k8sClient, r.logger, r.jenkins, "user-groovy", r.jenkins.Spec.GroovyScripts.Customization)

	requeue, err := groovyClient.WaitForSecretSynchronization(resources.GroovyScriptsSecretVolumePath)
	if err != nil {
		return reconcile.Result{}, err
	}
	if requeue {
		return reconcile.Result{Requeue: true}, nil
	}
	requeue, err = groovyClient.Ensure(func(name string) bool {
		return strings.HasSuffix(name, ".groovy")
	}, func(groovyScript string) string {
		// TODO load secrets to variables
		return groovyScript
	})
	if err != nil {
		return reconcile.Result{}, err
	}
	if requeue {
		return reconcile.Result{Requeue: true}, nil
	}

	configurationAsCodeClient := casc.New(jenkinsClient, r.k8sClient, r.logger, r.jenkins)
	requeue, err = configurationAsCodeClient.Ensure(r.jenkins)
	if err != nil {
		return reconcile.Result{}, err
	}
	if requeue {
		return reconcile.Result{Requeue: true}, nil
	}

	return reconcile.Result{}, nil
}
