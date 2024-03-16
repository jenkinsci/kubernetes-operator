package e2e

import (
	"context"
	"fmt"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	// +kubebuilder:scaffold:imports
)

var _ = Describe("Jenkins controller configuration", func() {

	const (
		jenkinsCRName            = e2e
		numberOfExecutors        = 6
		numberOfExecutorsEnvName = "NUMBER_OF_EXECUTORS"
		systemMessage            = "Configuration as Code integration works!!!"
		systemMessageEnvName     = "SYSTEM_MESSAGE"
		priorityClassName        = ""
	)

	var (
		namespace *corev1.Namespace
		jenkins   *v1alpha2.Jenkins
		mySeedJob = seedJobConfig{
			SeedJob: v1alpha2.SeedJob{
				ID:                    "jenkins-operator",
				CredentialID:          "jenkins-operator",
				JenkinsCredentialType: v1alpha2.NoJenkinsCredentialCredentialType,
				Targets:               "cicd/jobs/*.jenkins",
				Description:           "Jenkins Operator repository",
				RepositoryBranch:      "master",
				RepositoryURL:         "https://github.com/jenkinsci/kubernetes-operator.git",
				PollSCM:               "1 1 1 1 1",
				UnstableOnDeprecation: true,
				BuildPeriodically:     "1 1 1 1 1",
				FailOnMissingPlugin:   true,
				IgnoreMissingFiles:    true,
				//AdditionalClasspath: can fail with the seed job agent
				GitHubPushTrigger: true,
			},
		}
		groovyScripts = v1alpha2.GroovyScripts{
			Customization: v1alpha2.Customization{
				Configurations: []v1alpha2.ConfigMapRef{
					{
						Name: userConfigurationConfigMapName,
					},
				},
				Secret: v1alpha2.SecretRef{
					Name: userConfigurationSecretName,
				},
			},
		}
		casc = v1alpha2.ConfigurationAsCode{
			Customization: v1alpha2.Customization{
				Configurations: []v1alpha2.ConfigMapRef{
					{
						Name: userConfigurationConfigMapName,
					},
				},
				Secret: v1alpha2.SecretRef{
					Name: userConfigurationSecretName,
				},
			},
		}
		userConfigurationSecretData = map[string]string{
			systemMessageEnvName:     systemMessage,
			numberOfExecutorsEnvName: fmt.Sprintf("%d", numberOfExecutors),
		}
	)

	BeforeEach(func() {
		namespace = CreateNamespace()
		createUserConfigurationSecret(namespace.Name, userConfigurationSecretData)
		createUserConfigurationConfigMap(namespace.Name, numberOfExecutorsEnvName, fmt.Sprintf("${%s}", systemMessageEnvName))
		jenkins = RenderJenkinsCR(jenkinsCRName, namespace.Name, &[]v1alpha2.SeedJob{mySeedJob.SeedJob}, groovyScripts, casc, priorityClassName)
		Expect(K8sClient.Create(context.TODO(), jenkins)).Should(Succeed())
		createDefaultLimitsForContainersInNamespace(namespace.Name)
		createKubernetesCredentialsProviderSecret(namespace.Name, mySeedJob)
	})

	AfterEach(func() {
		ShowLogsIfTestHasFailed(CurrentGinkgoTestDescription().Failed, namespace.Name)
		DestroyNamespace(namespace)
	})

	Context("when deploying CR to cluster", func() {
		It("creates vanilla Jenkins instance and configures it", func() {
			WaitForJenkinsBaseConfigurationToComplete(jenkins)
			verifyJenkinsMasterPodAttributes(jenkins)
			verifyServices(jenkins)
			jenkinsClient, cleanUpFunc := verifyJenkinsAPIConnection(jenkins, namespace.Name)
			defer cleanUpFunc()
			verifyPlugins(jenkinsClient, jenkins)
			WaitForJenkinsUserConfigurationToComplete(jenkins)
			verifyUserConfiguration(jenkinsClient, numberOfExecutors, systemMessage)
			verifyJenkinsSeedJobs(jenkinsClient, []seedJobConfig{mySeedJob})
		})
	})
})

var _ = Describe("Jenkins controller priority class", func() {

	const (
		jenkinsCRName     = "k8s-ete-priority-class-existing"
		priorityClassName = "system-cluster-critical"
	)

	var (
		namespace     *corev1.Namespace
		jenkins       *v1alpha2.Jenkins
		groovyScripts = v1alpha2.GroovyScripts{
			Customization: v1alpha2.Customization{
				Configurations: []v1alpha2.ConfigMapRef{},
			},
		}
		casc = v1alpha2.ConfigurationAsCode{
			Customization: v1alpha2.Customization{
				Configurations: []v1alpha2.ConfigMapRef{},
			},
		}
	)

	BeforeEach(func() {
		namespace = CreateNamespace()
		jenkins = RenderJenkinsCR(jenkinsCRName, namespace.Name, nil, groovyScripts, casc, priorityClassName)
		Expect(K8sClient.Create(context.TODO(), jenkins)).Should(Succeed())
	})

	AfterEach(func() {
		ShowLogsIfTestHasFailed(CurrentGinkgoTestDescription().Failed, namespace.Name)
		DestroyNamespace(namespace)
	})

	Context("when deploying CR with priority class to cluster", func() {
		It("creates Jenkins instance with priority class and configures it", func() {
			WaitForJenkinsBaseConfigurationToComplete(jenkins)
			verifyJenkinsMasterPodAttributes(jenkins)
		})
	})
})

var _ = Describe("Jenkins controller plugins test", func() {

	const (
		jenkinsCRName     = e2e
		priorityClassName = ""
		jobID             = "k8s-e2e"
	)

	var (
		namespace *corev1.Namespace
		jenkins   *v1alpha2.Jenkins
		mySeedJob = seedJobConfig{
			SeedJob: v1alpha2.SeedJob{
				ID:                    "jenkins-operator",
				CredentialID:          "jenkins-operator",
				JenkinsCredentialType: v1alpha2.NoJenkinsCredentialCredentialType,
				Targets:               "cicd/jobs/k8s.jenkins",
				Description:           "Jenkins Operator repository",
				RepositoryBranch:      "master",
				RepositoryURL:         "https://github.com/jenkinsci/kubernetes-operator.git",
			},
		}
		groovyScripts = v1alpha2.GroovyScripts{
			Customization: v1alpha2.Customization{
				Configurations: []v1alpha2.ConfigMapRef{},
			},
		}
		casc = v1alpha2.ConfigurationAsCode{
			Customization: v1alpha2.Customization{
				Configurations: []v1alpha2.ConfigMapRef{},
			},
		}
	)

	BeforeEach(func() {
		namespace = CreateNamespace()
		jenkins = RenderJenkinsCR(jenkinsCRName, namespace.Name, &[]v1alpha2.SeedJob{mySeedJob.SeedJob}, groovyScripts, casc, priorityClassName)
		Expect(K8sClient.Create(context.TODO(), jenkins)).Should(Succeed())
	})

	AfterEach(func() {
		ShowLogsIfTestHasFailed(CurrentGinkgoTestDescription().Failed, namespace.Name)
		DestroyNamespace(namespace)
	})

	Context("when deploying CR with a SeedJob to cluster", func() {
		It("runs kubernetes plugin job successfully", func() {
			WaitForJenkinsUserConfigurationToComplete(jenkins)
			jenkinsClient, cleanUpFunc := verifyJenkinsAPIConnection(jenkins, namespace.Name)
			defer cleanUpFunc()
			waitForJobCreation(jenkinsClient, jobID)
			verifyJobCanBeRun(jenkinsClient, jobID)
			verifyJobHasBeenRunCorrectly(jenkinsClient, jobID)
		})
	})
})
