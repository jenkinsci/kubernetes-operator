package e2e

import (
	"context"
	"fmt"
	"text/template"
	"time"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/internal/render"
	"github.com/jenkinsci/kubernetes-operator/internal/try"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/user/seedjobs"
	"github.com/jenkinsci/kubernetes-operator/pkg/constants"

	"github.com/bndr/gojenkins"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type seedJobConfig struct {
	v1alpha2.SeedJob
	JobNames   []string `json:"jobNames,omitempty"`
	Username   string   `json:"username,omitempty"`
	Password   string   `json:"password,omitempty"`
	PrivateKey string   `json:"privateKey,omitempty"`
}

func createKubernetesCredentialsProviderSecret(namespace string, config seedJobConfig) {
	if config.JenkinsCredentialType == v1alpha2.NoJenkinsCredentialCredentialType {
		return
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.CredentialID,
			Namespace: namespace,
			Annotations: map[string]string{
				"jenkins.io/credentials-description": "credentials from Kubernetes " + config.ID,
			},
			Labels: map[string]string{
				seedjobs.JenkinsCredentialTypeLabelName: config.CredentialID,
			},
		},
		StringData: map[string]string{
			seedjobs.UsernameSecretKey:   config.Username,
			seedjobs.PasswordSecretKey:   config.Password,
			seedjobs.PrivateKeySecretKey: config.PrivateKey,
		},
	}

	Expect(K8sClient.Create(context.TODO(), secret)).Should(Succeed())
}

func verifyJenkinsSeedJobs(jenkinsClient jenkinsclient.Jenkins, seedJobs []seedJobConfig) {
	By("creating Jenkins jobs by seed jobs")

	var err error
	for _, seedJob := range seedJobs {
		if seedJob.JenkinsCredentialType == v1alpha2.BasicSSHCredentialType || seedJob.JenkinsCredentialType == v1alpha2.UsernamePasswordCredentialType {
			err = verifyIfJenkinsCredentialExists(jenkinsClient, seedJob.CredentialID)
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Jenkins credential '%s' not created for seed job ID '%s'\n", seedJob.CredentialID, seedJob.ID))
		}

		verifySeedJobProperties(jenkinsClient, seedJob)

		for _, requireJobName := range seedJob.JobNames {
			err = try.Until(func() (end bool, err error) {
				_, err = jenkinsClient.GetJob(requireJobName)
				return err == nil, err
			}, time.Second*2, time.Minute*2)
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Jenkins job '%s' not created by seed job ID '%s'\n", requireJobName, seedJob.ID))
		}
	}
}

func verifySeedJobProperties(jenkinsClient jenkinsclient.Jenkins, seedJob seedJobConfig) {
	data := struct {
		ID                    string
		CredentialID          string
		Targets               string
		ExecuteShell          string
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
		ExecuteShell:          seedJob.ExecuteShell,
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
		AgentName:             seedjobs.AgentName,
	}

	groovyScript, err := render.Render(verifySeedJobPropertiesGroovyScriptTemplate, data)
	Expect(err).NotTo(HaveOccurred(), groovyScript)

	logs, err := jenkinsClient.ExecuteScript(groovyScript)
	Expect(err).NotTo(HaveOccurred(), logs, groovyScript)
}

func verifyIfJenkinsCredentialExists(jenkinsClient jenkinsclient.Jenkins, credentialName string) error {
	groovyScriptFmt := `import com.cloudbees.plugins.credentials.Credentials

Set<Credentials> allCredentials = new HashSet<Credentials>();

def creds = com.cloudbees.plugins.credentials.CredentialsProvider.lookupCredentials(
	com.cloudbees.plugins.credentials.Credentials.class
);

allCredentials.addAll(creds)

Jenkins.instance.getAllItems(com.cloudbees.hudson.plugins.folder.Folder.class).each{ f ->
	creds = com.cloudbees.plugins.credentials.CredentialsProvider.lookupCredentials(
      	com.cloudbees.plugins.credentials.Credentials.class, f)
	allCredentials.addAll(creds)
}

def found = false
for (c in allCredentials) {
	if("%s".equals(c.id)) found = true
}
if(!found) {
	throw new Exception("Expected credential not found")
}`
	groovyScript := fmt.Sprintf(groovyScriptFmt, credentialName)
	_, err := jenkinsClient.ExecuteScript(groovyScript)
	return err
}

var verifySeedJobPropertiesGroovyScriptTemplate = template.Must(template.New("test-e2e-verify-job-properties").Parse(`
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
import hudson.tasks.BuildStep;

Jenkins jenkins = Jenkins.instance

def jobDslSeedName = "{{ .ID }}-{{ .SeedJobSuffix }}"
def jobRef = jenkins.getItem(jobDslSeedName)

if (jobRef == null) {
	throw new Exception("Job with given name not found")
}

if (!jobRef.getDisplayName().equals("Seed Job from {{ .ID }}")) {
	throw new Exception("Display name is not equal")	
}

if (jobRef.getScm() == null) {
	throw new Exception("No SCM found")
}

if (jobRef.getScm().getBranches().find { val -> val.getName() == "{{ .RepositoryBranch }}" } == null) {
	throw new Exception("Specified SCM branch not found")	
}

if(jobRef.getScm().getRepositories().find { it.getURIs().find { uri -> uri.toString().equals("https://github.com/jenkinsci/kubernetes-operator.git") } } == null) {
	throw new Exception("Specified SCM repositories are invalid")
}

{{ if .PollSCM }}
if (jobRef.getTriggers().find { key, val -> val.getClass().getSimpleName() == "SCMTrigger" && val.getSpec() == "{{ .PollSCM }}"  } == null) {
	throw new Exception("SCMTrigger not found but set")
}
{{ end }}

{{ if .GitHubPushTrigger }}
if (jobRef.getTriggers().find { key, val -> val.getClass().getSimpleName() == "GitHubPushTrigger" } == null) {
	throw new Exception("GitHubPushTrigger not found but set")
}
{{ end }}

{{ if .BuildPeriodically }}
if (jobRef.getTriggers().find { key, val -> val.getClass().getSimpleName() == "TimerTrigger" && val.getSpec() == "{{ .BuildPeriodically }}" } == null) {
	throw new Exception("BuildPeriodically not found but set")
}
{{ end }}

for (BuildStep step : jobRef.getBuildersList()) {
	if (!step.getTargets().equals("{{ .Targets }}")) {
		throw new Exception("Targets are not equals'")	
	}

	if (!step.getAdditionalClasspath().equals(null)) {
		throw new Exception("AdditionalClasspath is not equal")
	}

	if (!step.isFailOnMissingPlugin().equals({{ .FailOnMissingPlugin }})) {
		throw new Exception("FailOnMissingPlugin is not equal")
	}

	if (!step.isUnstableOnDeprecation().equals({{ .UnstableOnDeprecation }})) {
		throw new Exception("UnstableOnDeprecation is not equal")
	}

	if (!step.isIgnoreMissingFiles().equals({{ .IgnoreMissingFiles }})) {
		throw new Exception("IgnoreMissingFiles is not equal")
	}
}
`))

func verifyJobCanBeRun(jenkinsClient jenkinsclient.Jenkins, jobID string) {
	By("retrieving created Jenkins job")
	job, err := jenkinsClient.GetJob(jobID)
	Expect(err).To(BeNil())

	By("running Jenkins job")
	_, err = job.InvokeSimple(map[string]string{})
	Expect(err).To(BeNil())

	// FIXME: waitForJobToFinish use
	By("waiting for the job to finish")
	time.Sleep(100 * time.Second) // wait for the build to complete
}
func verifyJobHasBeenRunCorrectly(jenkinsClient jenkinsclient.Jenkins, jobID string) {
	By("retrieving finished job")

	var (
		err   error
		job   *gojenkins.Job
		build *gojenkins.Build
	)

	Eventually(func() (bool, error) {
		job, err = jenkinsClient.GetJob(jobID)
		Expect(err).To(BeNil())
		build, err = job.GetLastBuild()
		Expect(err).To(BeNil())

		By("evaluating correctness of the outcome")
		return build.IsGood(), err
	}, time.Duration(110)*retryInterval, retryInterval).Should(BeTrue())
}
