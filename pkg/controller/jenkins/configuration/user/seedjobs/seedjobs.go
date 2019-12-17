package seedjobs

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"reflect"
	"text/template"

	"github.com/jenkinsci/kubernetes-operator/internal/render"
	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/constants"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/groovy"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/notifications/reason"

	"github.com/go-logr/logr"
	stackerr "github.com/pkg/errors"
	apps "k8s.io/api/apps/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// UsernameSecretKey is username data key in Kubernetes secret used to create Jenkins username/password credential
	UsernameSecretKey = "username"
	// PasswordSecretKey is password data key in Kubernetes secret used to create Jenkins username/password credential
	PasswordSecretKey = "password"
	// PrivateKeySecretKey is private key data key in Kubernetes secret used to create Jenkins SSH credential
	PrivateKeySecretKey = "privateKey"

	// JenkinsCredentialTypeLabelName is label for kubernetes-credentials-provider-plugin which determine Jenkins
	// credential type
	JenkinsCredentialTypeLabelName = "jenkins.io/credentials-type"

	// AgentName is the name of seed job agent
	AgentName = "seed-job-agent"

	creatingGroovyScriptName = "seed-job-groovy-script.groovy"
)

var seedJobGroovyScriptTemplate = template.Must(template.New(creatingGroovyScriptName).Parse(`
import hudson.model.FreeStyleProject;
import hudson.plugins.git.GitSCM;
import hudson.plugins.git.BranchSpec;
import hudson.triggers.SCMTrigger;
import hudson.triggers.TimerTrigger;
import hudson.util.Secret;
import javaposse.jobdsl.plugin.*;
import jenkins.model.Jenkins;
import jenkins.model.JenkinsLocationConfiguration;
import com.cloudbees.plugins.credentials.CredentialsScope;
import com.cloudbees.plugins.credentials.domains.Domain;
import com.cloudbees.plugins.credentials.SystemCredentialsProvider;
import jenkins.model.JenkinsLocationConfiguration;
import org.jenkinsci.plugins.workflow.job.WorkflowJob;
import org.jenkinsci.plugins.workflow.cps.CpsScmFlowDefinition;
{{ if .GitHubPushTrigger }}
import com.cloudbees.jenkins.GitHubPushTrigger;
{{ end }}
import hudson.model.FreeStyleProject;
import hudson.model.labels.LabelAtom;
import hudson.plugins.git.BranchSpec;
import hudson.plugins.git.GitSCM;
import hudson.plugins.git.SubmoduleConfig;
import hudson.plugins.git.extensions.impl.CloneOption;
import javaposse.jobdsl.plugin.ExecuteDslScripts;
import javaposse.jobdsl.plugin.LookupStrategy;
import javaposse.jobdsl.plugin.RemovedJobAction;
import javaposse.jobdsl.plugin.RemovedViewAction;

import static com.google.common.collect.Lists.newArrayList;

Jenkins jenkins = Jenkins.instance

def jobDslSeedName = "{{ .ID }}-{{ .SeedJobSuffix }}";
def jobRef = jenkins.getItem(jobDslSeedName)

def repoList = GitSCM.createRepoList("{{ .RepositoryURL }}", "{{ .CredentialID }}")
def gitExtensions = [new CloneOption(true, true, ";", 10)]
def scm = new GitSCM(
        repoList,
        newArrayList(new BranchSpec("{{ .RepositoryBranch }}")),
        false,
        Collections.<SubmoduleConfig>emptyList(),
        null,
        null,
        gitExtensions
)

def executeDslScripts = new ExecuteDslScripts()
executeDslScripts.setTargets("{{ .Targets }}")
executeDslScripts.setSandbox(false)
executeDslScripts.setRemovedJobAction(RemovedJobAction.DELETE)
executeDslScripts.setRemovedViewAction(RemovedViewAction.DELETE)
executeDslScripts.setLookupStrategy(LookupStrategy.SEED_JOB)
executeDslScripts.setAdditionalClasspath("{{ .AdditionalClasspath }}")
executeDslScripts.setFailOnMissingPlugin({{ .FailOnMissingPlugin }})
executeDslScripts.setUnstableOnDeprecation({{ .UnstableOnDeprecation }})
executeDslScripts.setIgnoreMissingFiles({{ .IgnoreMissingFiles }})

if (jobRef == null) {
        jobRef = jenkins.createProject(FreeStyleProject, jobDslSeedName)
}

jobRef.getBuildersList().clear()
jobRef.getBuildersList().add(executeDslScripts)
jobRef.setDisplayName("Seed Job from {{ .ID }}")
jobRef.setScm(scm)

{{ if .PollSCM }}
jobRef.addTrigger(new SCMTrigger("{{ .PollSCM }}"))
{{ end }}

{{ if .GitHubPushTrigger }}
jobRef.addTrigger(new GitHubPushTrigger())
{{ end }}

{{ if .BuildPeriodically }}
jobRef.addTrigger(new TimerTrigger("{{ .BuildPeriodically }}"))
{{ end}}
jobRef.setAssignedLabel(new LabelAtom("{{ .AgentName }}"))
jenkins.getQueue().schedule(jobRef)
`))

// SeedJobs defines API for configuring and ensuring Jenkins Seed Jobs and Deploy Keys
type SeedJobs struct {
	configuration.Configuration
	jenkinsClient jenkinsclient.Jenkins
	logger        logr.Logger
}

// New creates SeedJobs object
func New(jenkinsClient jenkinsclient.Jenkins, config configuration.Configuration, logger logr.Logger) *SeedJobs {
	return &SeedJobs{
		Configuration: config,
		jenkinsClient: jenkinsClient,
		logger:        logger,
	}
}

// EnsureSeedJobs configures seed job and runs it for every entry from Jenkins.Spec.SeedJobs
func (s *SeedJobs) EnsureSeedJobs(jenkins *v1alpha2.Jenkins) (done bool, err error) {
	if s.isRecreatePodNeeded(*jenkins) {
		message := "Some seed job has been deleted, recreating pod"
		s.logger.Info(message)

		restartReason := reason.NewPodRestart(
			reason.OperatorSource,
			[]string{message},
		)
		return false, s.RestartJenkinsMasterPod(restartReason)
	}

	if len(jenkins.Spec.SeedJobs) > 0 {
		err := s.createAgent(s.jenkinsClient, s.Client, jenkins, jenkins.Namespace, AgentName)
		if err != nil {
			return false, err
		}

		requeue, err := s.waitForSeedJobAgent(AgentName)
		if err != nil {
			return false, err
		}
		if requeue {
			return false, nil
		}
	} else if len(jenkins.Spec.SeedJobs) == 0 {
		err := s.Client.Delete(context.TODO(), &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: jenkins.Namespace,
				Name:      agentDeploymentName(*jenkins, AgentName),
			},
		})

		if err != nil && !apierrors.IsNotFound(err) {
			return false, stackerr.WithStack(err)
		}
	}

	if err = s.ensureLabelsForSecrets(*jenkins); err != nil {
		return false, err
	}

	requeue, err := s.createJobs(jenkins)
	if err != nil {
		return false, err
	}
	if requeue {
		return false, nil
	}

	seedJobIDs := s.getAllSeedJobIDs(*jenkins)
	if !reflect.DeepEqual(seedJobIDs, jenkins.Status.CreatedSeedJobs) {
		jenkins.Status.CreatedSeedJobs = seedJobIDs
		return false, stackerr.WithStack(s.Client.Update(context.TODO(), jenkins))
	}

	return true, nil
}

func (s SeedJobs) waitForSeedJobAgent(agentName string) (requeue bool, err error) {
	agent := appsv1.Deployment{}
	err = s.Client.Get(context.TODO(), types.NamespacedName{Name: agentDeploymentName(*s.Jenkins, agentName), Namespace: s.Jenkins.Namespace}, &agent)
	if apierrors.IsNotFound(err) {
		return true, nil
	} else if err != nil {
		return true, err
	}

	noReadyReplicas := agent.Status.ReadyReplicas == 0
	if noReadyReplicas {
		s.logger.Info(fmt.Sprintf("Waiting for Seed Job Agent `%s`...", agentName))
		return true, nil
	}

	return false, nil
}

// createJob is responsible for creating jenkins job which configures jenkins seed jobs and deploy keys
func (s *SeedJobs) createJobs(jenkins *v1alpha2.Jenkins) (requeue bool, err error) {
	groovyClient := groovy.New(s.jenkinsClient, s.Client, s.logger, jenkins, "seed-jobs", jenkins.Spec.GroovyScripts.Customization)
	for _, seedJob := range jenkins.Spec.SeedJobs {
		credentialValue, err := s.credentialValue(jenkins.Namespace, seedJob)
		if err != nil {
			return true, err
		}

		groovyScript, err := seedJobCreatingGroovyScript(seedJob)
		if err != nil {
			return true, err
		}

		hash := sha256.New()
		hash.Write([]byte(groovyScript))
		hash.Write([]byte(credentialValue))
		requeue, err := groovyClient.EnsureSingle(seedJob.ID, fmt.Sprintf("%s.groovy", seedJob.ID), base64.URLEncoding.EncodeToString(hash.Sum(nil)), groovyScript)
		if err != nil {
			return true, err
		}

		if requeue {
			return true, nil
		}
	}

	return false, nil
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
			err := s.Client.Get(context.TODO(), namespaceName, secret)
			if err != nil {
				return stackerr.WithStack(err)
			}

			if !resources.VerifyIfLabelsAreSet(secret, requiredLabels) {
				secret.ObjectMeta.Labels = requiredLabels
				if err = s.Client.Update(context.TODO(), secret); err != nil {
					return stackerr.WithStack(err)
				}
			}
		}
	}

	return nil
}

func (s *SeedJobs) credentialValue(namespace string, seedJob v1alpha2.SeedJob) (string, error) {
	if seedJob.JenkinsCredentialType == v1alpha2.BasicSSHCredentialType || seedJob.JenkinsCredentialType == v1alpha2.UsernamePasswordCredentialType {
		secret := &corev1.Secret{}
		namespaceName := types.NamespacedName{Namespace: namespace, Name: seedJob.CredentialID}
		err := s.Client.Get(context.TODO(), namespaceName, secret)
		if err != nil {
			return "", stackerr.WithStack(err)
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

// createAgent deploys Jenkins agent to Kubernetes cluster
func (s SeedJobs) createAgent(jenkinsClient jenkinsclient.Jenkins, k8sClient client.Client, jenkinsManifest *v1alpha2.Jenkins, namespace string, agentName string) error {
	_, err := jenkinsClient.GetNode(agentName)

	// Create node if not exists
	if err != nil && err.Error() == "No node found" {
		_, err = jenkinsClient.CreateNode(agentName, 1, "The jenkins-operator generated agent", "/home/jenkins", agentName)
		if err != nil {
			return stackerr.WithStack(err)
		}
	} else if err != nil {
		return stackerr.WithStack(err)
	}

	secret, err := jenkinsClient.GetNodeSecret(agentName)
	if err != nil {
		return err
	}

	deployment := agentDeployment(jenkinsManifest, namespace, agentName, secret)

	err = k8sClient.Create(context.TODO(), deployment)
	if apierrors.IsAlreadyExists(err) {
		err := k8sClient.Update(context.TODO(), deployment)
		if err != nil {
			return stackerr.WithStack(err)
		}
	} else if err != nil {
		return stackerr.WithStack(err)
	}

	return nil
}

func agentDeploymentName(jenkins v1alpha2.Jenkins, agentName string) string {
	return fmt.Sprintf("%s-%s", agentName, jenkins.Name)
}

func agentDeployment(jenkins *v1alpha2.Jenkins, namespace string, agentName string, secret string) *apps.Deployment {
	return &apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agentDeploymentName(*jenkins, agentName),
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					BlockOwnerDeletion: &[]bool{true}[0],
					Controller:         &[]bool{true}[0],
					Kind:               jenkins.Kind,
					Name:               jenkins.Name,
					APIVersion:         jenkins.APIVersion,
					UID:                jenkins.UID,
				},
			},
		},
		Spec: apps.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "jnlp",
							Image: "jenkins/jnlp-slave:alpine",
							Env: []corev1.EnvVar{
								{
									Name: "JENKINS_TUNNEL",
									Value: fmt.Sprintf("%s.%s:%d",
										resources.GetJenkinsSlavesServiceName(jenkins),
										jenkins.ObjectMeta.Namespace,
										jenkins.Spec.SlaveService.Port),
								},
								{
									Name:  "JENKINS_SECRET",
									Value: secret,
								},
								{
									Name:  "JENKINS_AGENT_NAME",
									Value: agentName,
								},
								{
									Name: "JENKINS_URL",
									Value: fmt.Sprintf("http://%s.%s:%d",
										resources.GetJenkinsHTTPServiceName(jenkins),
										jenkins.ObjectMeta.Namespace,
										jenkins.Spec.Service.Port,
									),
								},
								{
									Name:  "JENKINS_AGENT_WORKDIR",
									Value: "/home/jenkins/agent",
								},
							},
						},
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": fmt.Sprintf("%s-selector", agentName),
					},
				},
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": fmt.Sprintf("%s-selector", agentName),
				},
			},
		},
	}
}

func seedJobCreatingGroovyScript(seedJob v1alpha2.SeedJob) (string, error) {
	data := struct {
		ID                    string
		CredentialID          string
		Targets               string
		RepositoryBranch      string
		RepositoryURL         string
		GitHubPushTrigger     bool
		BuildPeriodically     string
		PollSCM               string
		IgnoreMissingFiles    bool
		AdditionalClasspath   string
		FailOnMissingPlugin   bool
		UnstableOnDeprecation bool
		SeedJobSuffix         string
		AgentName             string
	}{
		ID:                    seedJob.ID,
		CredentialID:          seedJob.CredentialID,
		Targets:               seedJob.Targets,
		RepositoryBranch:      seedJob.RepositoryBranch,
		RepositoryURL:         seedJob.RepositoryURL,
		GitHubPushTrigger:     seedJob.GitHubPushTrigger,
		BuildPeriodically:     seedJob.BuildPeriodically,
		PollSCM:               seedJob.PollSCM,
		IgnoreMissingFiles:    seedJob.IgnoreMissingFiles,
		AdditionalClasspath:   seedJob.AdditionalClasspath,
		FailOnMissingPlugin:   seedJob.FailOnMissingPlugin,
		UnstableOnDeprecation: seedJob.UnstableOnDeprecation,
		SeedJobSuffix:         constants.SeedJobSuffix,
		AgentName:             AgentName,
	}

	output, err := render.Render(seedJobGroovyScriptTemplate, data)
	if err != nil {
		return "", err
	}

	return output, nil
}
