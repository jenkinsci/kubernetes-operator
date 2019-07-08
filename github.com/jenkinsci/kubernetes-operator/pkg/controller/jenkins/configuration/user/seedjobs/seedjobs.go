package seedjobs

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"reflect"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/constants"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/jobs"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"

	"github.com/go-logr/logr"
	stackerr "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ConfigureSeedJobsName this is the fixed seed job name
	ConfigureSeedJobsName = constants.OperatorName + "-configure-seed-job"

	idParameterName               = "ID"
	credentialIDParameterName     = "CREDENTIAL_ID"
	repositoryURLParameterName    = "REPOSITORY_URL"
	repositoryBranchParameterName = "REPOSITORY_BRANCH"
	targetsParameterName          = "TARGETS"
	displayNameParameterName      = "SEED_JOB_DISPLAY_NAME"

	// UsernameSecretKey is username data key in Kubernetes secret used to create Jenkins username/password credential
	UsernameSecretKey = "username"
	// PasswordSecretKey is password data key in Kubernetes secret used to create Jenkins username/password credential
	PasswordSecretKey = "password"
	// PrivateKeySecretKey is private key data key in Kubernetes secret used to create Jenkins SSH credential
	PrivateKeySecretKey = "privateKey"

	// JenkinsCredentialTypeLabelName is label for kubernetes-credentials-provider-plugin which determine Jenkins
	// credential type
	JenkinsCredentialTypeLabelName = "jenkins.io/credentials-type"
)

// SeedJobs defines API for configuring and ensuring Jenkins Seed Jobs and Deploy Keys
type SeedJobs struct {
	jenkinsClient jenkinsclient.Jenkins
	k8sClient     k8s.Client
	logger        logr.Logger
}

// New creates SeedJobs object
func New(jenkinsClient jenkinsclient.Jenkins, k8sClient k8s.Client, logger logr.Logger) *SeedJobs {
	return &SeedJobs{
		jenkinsClient: jenkinsClient,
		k8sClient:     k8sClient,
		logger:        logger,
	}
}

// EnsureSeedJobs configures seed job and runs it for every entry from Jenkins.Spec.SeedJobs
func (s *SeedJobs) EnsureSeedJobs(jenkins *v1alpha2.Jenkins) (done bool, err error) {
	if s.isRecreatePodNeeded(*jenkins) {
		s.logger.Info("Some seed job has been deleted, recreating pod")
		return false, s.restartJenkinsMasterPod(*jenkins)
	}

	if err = s.createJob(); err != nil {
		s.logger.V(log.VWarn).Info("Couldn't create jenkins seed job")
		return false, err
	}

	if err = s.ensureLabelsForSecrets(*jenkins); err != nil {
		return false, err
	}

	done, err = s.buildJobs(jenkins)
	if err != nil {
		s.logger.V(log.VWarn).Info("Couldn't build jenkins seed job")
		return false, err
	}

	seedJobIDs := s.getAllSeedJobIDs(*jenkins)
	if done && !reflect.DeepEqual(seedJobIDs, jenkins.Status.CreatedSeedJobs) {
		jenkins.Status.CreatedSeedJobs = seedJobIDs
		return false, stackerr.WithStack(s.k8sClient.Update(context.TODO(), jenkins))
	}

	return done, nil
}

// createJob is responsible for creating jenkins job which configures jenkins seed jobs and deploy keys
func (s *SeedJobs) createJob() error {
	_, created, err := s.jenkinsClient.CreateOrUpdateJob(seedJobConfigXML, ConfigureSeedJobsName)
	if err != nil {
		return err
	}
	if created {
		s.logger.Info(fmt.Sprintf("'%s' job has been created", ConfigureSeedJobsName))
	}
	return nil
}

// ensureLabelsForSecrets adds labels to Kubernetes secrets where are Jenkins credentials used for seed jobs,
// thanks to them kubernetes-credentials-provider-plugin will create Jenkins credentials in Jenkins and
// Operator will able to watch any changes made to them
func (s *SeedJobs) ensureLabelsForSecrets(jenkins v1alpha2.Jenkins) error {
	for _, seedJob := range jenkins.Spec.SeedJobs {
		if seedJob.JenkinsCredentialType == v1alpha2.BasicSSHCredentialType || seedJob.JenkinsCredentialType == v1alpha2.UsernamePasswordCredentialType {
			requiredLabels := resources.BuildLabelsForWatchedResources(jenkins)
			requiredLabels[JenkinsCredentialTypeLabelName] = string(seedJob.JenkinsCredentialType)

			secret := &corev1.Secret{}
			namespaceName := types.NamespacedName{Namespace: jenkins.ObjectMeta.Namespace, Name: seedJob.CredentialID}
			err := s.k8sClient.Get(context.TODO(), namespaceName, secret)
			if err != nil {
				return stackerr.WithStack(err)
			}

			if !resources.VerifyIfLabelsAreSet(secret, requiredLabels) {
				secret.ObjectMeta.Labels = requiredLabels
				err = stackerr.WithStack(s.k8sClient.Update(context.TODO(), secret))
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// buildJobs is responsible for running jenkins builds which configures jenkins seed jobs and deploy keys
func (s *SeedJobs) buildJobs(jenkins *v1alpha2.Jenkins) (done bool, err error) {
	allDone := true
	for _, seedJob := range jenkins.Spec.SeedJobs {
		credentialValue, err := s.credentialValue(jenkins.Namespace, seedJob)
		if err != nil {
			return false, err
		}
		parameters := map[string]string{
			idParameterName:               seedJob.ID,
			credentialIDParameterName:     seedJob.CredentialID,
			repositoryURLParameterName:    seedJob.RepositoryURL,
			repositoryBranchParameterName: seedJob.RepositoryBranch,
			targetsParameterName:          seedJob.Targets,
			displayNameParameterName:      fmt.Sprintf("Seed Job from %s", seedJob.ID),
		}

		hash := sha256.New()
		hash.Write([]byte(parameters[idParameterName]))
		hash.Write([]byte(parameters[credentialIDParameterName]))
		hash.Write([]byte(credentialValue))
		hash.Write([]byte(parameters[repositoryURLParameterName]))
		hash.Write([]byte(parameters[repositoryBranchParameterName]))
		hash.Write([]byte(parameters[targetsParameterName]))
		hash.Write([]byte(parameters[displayNameParameterName]))
		encodedHash := base64.URLEncoding.EncodeToString(hash.Sum(nil))

		jobsClient := jobs.New(s.jenkinsClient, s.k8sClient, s.logger)
		done, err := jobsClient.EnsureBuildJob(ConfigureSeedJobsName, encodedHash, parameters, jenkins, true)
		if err != nil {
			return false, err
		}
		if !done {
			allDone = false
		}
	}
	return allDone, nil
}

func (s *SeedJobs) credentialValue(namespace string, seedJob v1alpha2.SeedJob) (string, error) {
	if seedJob.JenkinsCredentialType == v1alpha2.BasicSSHCredentialType || seedJob.JenkinsCredentialType == v1alpha2.UsernamePasswordCredentialType {
		secret := &corev1.Secret{}
		namespaceName := types.NamespacedName{Namespace: namespace, Name: seedJob.CredentialID}
		err := s.k8sClient.Get(context.TODO(), namespaceName, secret)
		if err != nil {
			return "", err
		}

		if seedJob.JenkinsCredentialType == v1alpha2.BasicSSHCredentialType {
			return string(secret.Data[PrivateKeySecretKey]), nil
		}
		return string(secret.Data[UsernameSecretKey]) + string(secret.Data[PasswordSecretKey]), nil
	}
	return "", nil
}

func (s *SeedJobs) getAllSeedJobIDs(jenkins v1alpha2.Jenkins) []string {
	var ids []string
	for _, seedJob := range jenkins.Spec.SeedJobs {
		ids = append(ids, seedJob.ID)
	}
	return ids
}

//TODO move to k8sClient
func (s *SeedJobs) getJenkinsMasterPod(jenkins v1alpha2.Jenkins) (*corev1.Pod, error) {
	jenkinsMasterPodName := resources.GetJenkinsMasterPodName(jenkins)
	currentJenkinsMasterPod := &corev1.Pod{}
	err := s.k8sClient.Get(context.TODO(), types.NamespacedName{Name: jenkinsMasterPodName, Namespace: jenkins.Namespace}, currentJenkinsMasterPod)
	if err != nil {
		return nil, err // don't wrap error
	}
	return currentJenkinsMasterPod, nil
}

//TODO move to k8sClient
func (s *SeedJobs) restartJenkinsMasterPod(jenkins v1alpha2.Jenkins) error {
	currentJenkinsMasterPod, err := s.getJenkinsMasterPod(jenkins)
	if err != nil {
		return err
	}
	s.logger.Info(fmt.Sprintf("Terminating Jenkins Master Pod %s/%s", currentJenkinsMasterPod.Namespace, currentJenkinsMasterPod.Name))
	return stackerr.WithStack(s.k8sClient.Delete(context.TODO(), currentJenkinsMasterPod))
}

func (s *SeedJobs) isRecreatePodNeeded(jenkins v1alpha2.Jenkins) bool {
	for _, createdSeedJob := range jenkins.Status.CreatedSeedJobs {
		found := false
		for _, seedJob := range jenkins.Spec.SeedJobs {
			if createdSeedJob == seedJob.ID {
				found = true
				break
			}
		}
		if !found {
			return true
		}
	}
	return false
}

// seedJobConfigXML this is the XML representation of seed job
var seedJobConfigXML = `
<flow-definition plugin="workflow-job@2.30">
  <actions/>
  <description>Configure Seed Jobs</description>
  <keepDependencies>false</keepDependencies>
  <properties>
    <hudson.model.ParametersDefinitionProperty>
      <parameterDefinitions>
        <hudson.model.StringParameterDefinition>
          <name>` + idParameterName + `</name>
          <description></description>
          <defaultValue></defaultValue>
          <trim>false</trim>
        </hudson.model.StringParameterDefinition>
        <hudson.model.StringParameterDefinition>
          <name>` + credentialIDParameterName + `</name>
          <description></description>
          <defaultValue></defaultValue>
          <trim>false</trim>
        </hudson.model.StringParameterDefinition>
        <hudson.model.StringParameterDefinition>
          <name>` + repositoryURLParameterName + `</name>
          <description></description>
          <defaultValue></defaultValue>
          <trim>false</trim>
        </hudson.model.StringParameterDefinition>
        <hudson.model.StringParameterDefinition>
          <name>` + repositoryBranchParameterName + `</name>
          <description></description>
          <defaultValue>master</defaultValue>
          <trim>false</trim>
        </hudson.model.StringParameterDefinition>
        <hudson.model.StringParameterDefinition>
          <name>` + displayNameParameterName + `</name>
          <description></description>
          <defaultValue></defaultValue>
          <trim>false</trim>
        </hudson.model.StringParameterDefinition>
        <hudson.model.StringParameterDefinition>
          <name>` + targetsParameterName + `</name>
          <description></description>
          <defaultValue>cicd/jobs/*.jenkins</defaultValue>
          <trim>false</trim>
        </hudson.model.StringParameterDefinition>
      </parameterDefinitions>
    </hudson.model.ParametersDefinitionProperty>
  </properties>
  <definition class="org.jenkinsci.plugins.workflow.cps.CpsFlowDefinition" plugin="workflow-cps@2.61">
    <script>
import hudson.model.FreeStyleProject
import hudson.model.labels.LabelAtom
import hudson.plugins.git.BranchSpec
import hudson.plugins.git.GitSCM
import hudson.plugins.git.SubmoduleConfig
import hudson.plugins.git.extensions.impl.CloneOption
import javaposse.jobdsl.plugin.ExecuteDslScripts
import javaposse.jobdsl.plugin.LookupStrategy
import javaposse.jobdsl.plugin.RemovedJobAction
import javaposse.jobdsl.plugin.RemovedViewAction

import static com.google.common.collect.Lists.newArrayList

Jenkins jenkins = Jenkins.instance

def jobDslSeedName = &quot;${params.` + idParameterName + `}-` + constants.SeedJobSuffix + `&quot;
def jobRef = jenkins.getItem(jobDslSeedName)

def repoList = GitSCM.createRepoList(&quot;${params.` + repositoryURLParameterName + `}&quot;, &quot;${params.` + credentialIDParameterName + `}&quot;)
def gitExtensions = [new CloneOption(true, true, &quot;&quot;, 10)]
def scm = new GitSCM(
        repoList,
        newArrayList(new BranchSpec(&quot;${params.` + repositoryBranchParameterName + `}&quot;)),
        false,
        Collections.&lt;SubmoduleConfig&gt; emptyList(),
        null,
        null,
        gitExtensions
)

def executeDslScripts = new ExecuteDslScripts()
executeDslScripts.setTargets(&quot;${params.` + targetsParameterName + `}&quot;)
executeDslScripts.setSandbox(false)
executeDslScripts.setRemovedJobAction(RemovedJobAction.DELETE)
executeDslScripts.setRemovedViewAction(RemovedViewAction.DELETE)
executeDslScripts.setLookupStrategy(LookupStrategy.SEED_JOB)
executeDslScripts.setAdditionalClasspath(&quot;src&quot;)

if (jobRef == null) {
        jobRef = jenkins.createProject(FreeStyleProject, jobDslSeedName)
}
jobRef.getBuildersList().clear()
jobRef.getBuildersList().add(executeDslScripts)
jobRef.setDisplayName(&quot;${params.` + displayNameParameterName + `}&quot;)
jobRef.setScm(scm)
// TODO don't use master executors
jobRef.setAssignedLabel(new LabelAtom(&quot;master&quot;))

jenkins.getQueue().schedule(jobRef)
</script>
    <sandbox>false</sandbox>
  </definition>
  <triggers/>
  <disabled>false</disabled>
</flow-definition>
`
