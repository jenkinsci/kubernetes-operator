package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/jenkinsci/kubernetes-operator/internal/try"
	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/user/seedjobs"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

type seedJobsConfig struct {
	SeedJobs []seedJobConfig `json:"seedJobs,omitempty"`
}

func TestSeedJobs(t *testing.T) {
	t.Parallel()
	if seedJobConfigurationFile == nil || len(*seedJobConfigurationFile) == 0 {
		t.Skipf("Skipping test because flag '%+v' is not set", seedJobConfigurationFile)
	}
	seedJobsConfig := loadSeedJobsConfig(t)
	namespace, ctx := setupTest(t)
	// Deletes test namespace
	defer ctx.Cleanup()

	jenkinsCRName := "e2e"
	var seedJobs []v1alpha2.SeedJob

	// base
	for _, seedJobConfig := range seedJobsConfig.SeedJobs {
		createKubernetesCredentialsProviderSecret(t, namespace, seedJobConfig)
		seedJobs = append(seedJobs, seedJobConfig.SeedJob)
	}
	jenkins := createJenkinsCR(t, jenkinsCRName, namespace, &seedJobs, v1alpha2.GroovyScripts{}, v1alpha2.ConfigurationAsCode{})
	waitForJenkinsBaseConfigurationToComplete(t, jenkins)

	verifyJenkinsMasterPodAttributes(t, jenkins)
	client := verifyJenkinsAPIConnection(t, jenkins)
	verifyPlugins(t, client, jenkins)

	// user
	waitForJenkinsUserConfigurationToComplete(t, jenkins)
	verifyJenkinsSeedJobs(t, client, seedJobsConfig.SeedJobs)
}

func loadSeedJobsConfig(t *testing.T) seedJobsConfig {
	jsonFile, err := os.Open(*seedJobConfigurationFile)
	assert.NoError(t, err)
	defer func() { _ = jsonFile.Close() }()

	byteValue, err := ioutil.ReadAll(jsonFile)
	assert.NoError(t, err)

	var result seedJobsConfig
	err = json.Unmarshal([]byte(byteValue), &result)
	assert.NoError(t, err)
	assert.NotEmpty(t, result.SeedJobs)
	return result
}

func createKubernetesCredentialsProviderSecret(t *testing.T, namespace string, config seedJobConfig) {
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
				seedjobs.JenkinsCredentialTypeLabelName: string(config.CredentialID),
			},
		},
		StringData: map[string]string{
			seedjobs.UsernameSecretKey:   config.Username,
			seedjobs.PasswordSecretKey:   config.Password,
			seedjobs.PrivateKeySecretKey: config.PrivateKey,
		},
	}

	err := framework.Global.Client.Create(context.TODO(), secret, nil)
	require.NoError(t, err)
}

func verifyJenkinsSeedJobs(t *testing.T, jenkinsClient jenkinsclient.Jenkins, seedJobs []seedJobConfig) {
	var err error
	for _, seedJob := range seedJobs {
		if seedJob.JenkinsCredentialType == v1alpha2.BasicSSHCredentialType || seedJob.JenkinsCredentialType == v1alpha2.UsernamePasswordCredentialType {
			err = verifyIfJenkinsCredentialExists(jenkinsClient, seedJob.CredentialID)
			assert.NoErrorf(t, err, "Jenkins credential '%s' not created for seed job ID '%s'", seedJob.CredentialID, seedJob.ID)
		}

		for _, requireJobName := range seedJob.JobNames {
			err = try.Until(func() (end bool, err error) {
				_, err = jenkinsClient.GetJob(requireJobName)
				return err == nil, err
			}, time.Second*2, time.Minute*2)
			assert.NoErrorf(t, err, "Jenkins job '%s' not created by seed job ID '%s'", requireJobName, seedJob.ID)
		}
	}
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
