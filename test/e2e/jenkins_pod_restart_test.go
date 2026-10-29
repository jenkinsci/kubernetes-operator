package e2e

import (
	"context"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	// +kubebuilder:scaffold:imports
)

var _ = Describe("Jenkins controller", func() {

	const (
		jenkinsCRName            = e2e
		priorityClassName        = ""
		numberOfExecutors        = 3
		numberOfExecutorsEnvName = "NUMBER_OF_EXECUTORS"
		systemMessage            = "User configuration survives pod replacement"
	)

	var (
		namespace     *corev1.Namespace
		jenkins       *v1alpha2.Jenkins
		groovyScripts = v1alpha2.GroovyScripts{
			Customization: v1alpha2.Customization{
				Configurations: []v1alpha2.ConfigMapRef{{Name: userConfigurationConfigMapName}},
				Secret:         v1alpha2.SecretRef{Name: userConfigurationSecretName},
			},
		}
		casc = v1alpha2.ConfigurationAsCode{
			Customization: v1alpha2.Customization{
				Configurations: []v1alpha2.ConfigMapRef{{Name: userConfigurationConfigMapName}},
				Secret:         v1alpha2.SecretRef{Name: userConfigurationSecretName},
			},
		}
	)

	BeforeEach(func() {
		namespace = CreateNamespace()

		createUserConfigurationSecret(namespace.Name, map[string]string{numberOfExecutorsEnvName: "3"})
		createUserConfigurationConfigMap(namespace.Name, numberOfExecutorsEnvName, systemMessage)
		jenkins = RenderJenkinsCR(jenkinsCRName, namespace.Name, nil, groovyScripts, casc, priorityClassName)
		Expect(K8sClient.Create(context.TODO(), jenkins)).Should(Succeed())
	})

	AfterEach(func() {
		ShowLogsIfTestHasFailed(CurrentGinkgoTestDescription().Failed, namespace.Name)
		DestroyNamespace(namespace)
	})

	Context("when restarting Jenkins master pod", func() {
		It("new Jenkins pod should be created after a restart", func() {
			WaitForJenkinsBaseConfigurationToComplete(jenkins)
			WaitForJenkinsUserConfigurationToComplete(jenkins)
			originalPodUID := getJenkinsMasterPod(jenkins).UID
			restartJenkinsMasterPod(jenkins)
			waitForRecreateJenkinsMasterPod(jenkins)
			Expect(getJenkinsMasterPod(jenkins).UID).NotTo(Equal(originalPodUID))
			checkBaseConfigurationCompleteTimeIsNotSet(jenkins)
			Expect(getJenkins(jenkins.Namespace, jenkins.Name).Status.UserConfigurationCompletedTime).To(BeNil())
			WaitForJenkinsBaseConfigurationToComplete(jenkins)
			WaitForJenkinsUserConfigurationToComplete(jenkins)
			jenkinsClient, cleanUpFunc := verifyJenkinsAPIConnection(jenkins, namespace.Name)
			defer cleanUpFunc()
			verifyUserConfiguration(jenkinsClient, numberOfExecutors, systemMessage)
		})
	})
})

var _ = Describe("Jenkins controller", func() {

	const (
		jenkinsCRName     = e2e
		priorityClassName = ""
	)

	var (
		namespace     *corev1.Namespace
		jenkins       *v1alpha2.Jenkins
		groovyScripts = v1alpha2.GroovyScripts{
			Customization: v1alpha2.Customization{
				Configurations: []v1alpha2.ConfigMapRef{
					{
						Name: userConfigurationConfigMapName,
					},
				},
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

		configureAuthorizationToUnSecure(namespace.Name, userConfigurationConfigMapName)
		jenkins = createJenkinsCRSafeRestart(jenkinsCRName, namespace.Name, nil, groovyScripts, casc, priorityClassName)
	})

	AfterEach(func() {
		ShowLogsIfTestHasFailed(CurrentGinkgoTestDescription().Failed, namespace.Name)
		DestroyNamespace(namespace)
	})

	Context("when running Jenkins safe restart", func() {
		It("authorization strategy is not overwritten after a restart", func() {
			// TODO: @brokenpip3 temporary disable this flaky test
			Skip("Temporary skipping this test")
			WaitForJenkinsUserConfigurationToComplete(jenkins)
			jenkinsClient, cleanUpFunc := verifyJenkinsAPIConnection(jenkins, namespace.Name)
			defer cleanUpFunc()
			checkIfAuthorizationStrategyUnsecuredIsSet(jenkinsClient)
			err := jenkinsClient.SafeRestart(context.TODO())
			Expect(err).NotTo(HaveOccurred())
			waitForJenkinsSafeRestart(jenkinsClient)
			checkIfAuthorizationStrategyUnsecuredIsSet(jenkinsClient)
		})
	})
})
