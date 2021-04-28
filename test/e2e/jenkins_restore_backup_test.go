package e2e

import (
	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"

	. "github.com/onsi/ginkgo"
	corev1 "k8s.io/api/core/v1"
	// +kubebuilder:scaffold:imports
)

var _ = Describe("Jenkins controller backup and restore", func() {

	const (
		jenkinsCRName = e2e
		jobID         = "e2e-jenkins-operator"
	)

	var (
		namespace *corev1.Namespace
		jenkins   *v1alpha2.Jenkins
	)

	BeforeEach(func() {
		namespace = CreateNamespace()

		createPVC(namespace.Name)
		jenkins = createJenkinsWithBackupAndRestoreConfigured(jenkinsCRName, namespace.Name)
	})

	AfterEach(func() {
		DestroyNamespace(namespace)
	})

	Context("when deploying CR with backup enabled to cluster", func() {
		It("performs backups before pod deletion and restores them even Jenkins status is restarted", func() {
			WaitForJenkinsUserConfigurationToComplete(jenkins)
			jenkinsClient, cleanUpFunc := verifyJenkinsAPIConnection(jenkins, namespace.Name)
			defer cleanUpFunc()
			waitForJobCreation(jenkinsClient, jobID)
			verifyJobCanBeRun(jenkinsClient, jobID)

			jenkins = getJenkins(jenkins.Namespace, jenkins.Name)
			restartJenkinsMasterPod(jenkins)
			waitForRecreateJenkinsMasterPod(jenkins)
			WaitForJenkinsUserConfigurationToComplete(jenkins)
			jenkinsClient2, cleanUpFunc2 := verifyJenkinsAPIConnection(jenkins, namespace.Name)
			defer cleanUpFunc2()
			waitForJobCreation(jenkinsClient2, jobID)
			verifyJobBuildsAfterRestoreBackup(jenkinsClient2, jobID)

			resetJenkinsStatus(jenkins)
			jenkins = getJenkins(jenkins.Namespace, jenkins.Name)
			checkBaseConfigurationCompleteTimeIsNotSet(jenkins)
			WaitForJenkinsUserConfigurationToComplete(jenkins)
			jenkinsClient3, cleanUpFunc3 := verifyJenkinsAPIConnection(jenkins, namespace.Name)
			defer cleanUpFunc3()
			waitForJobCreation(jenkinsClient3, jobID)
			verifyJobBuildsAfterRestoreBackup(jenkinsClient3, jobID)
		})
	})
})
