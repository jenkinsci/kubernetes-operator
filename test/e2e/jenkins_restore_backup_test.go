package e2e

import (
	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
		namespace = createNamespace()

		createPVC(namespace.Name)
		jenkins = createJenkinsWithBackupAndRestoreConfigured(jenkinsCRName, namespace.Name)
	})

	AfterEach(func() {
		destroyNamespace(namespace)
	})

	Context("when deploying CR with backup enabled to cluster", func() {
		It("performs backups before pod deletion and restores them", func() {
			waitForJenkinsUserConfigurationToComplete(jenkins)
			jenkinsClient, cleanUpFunc := verifyJenkinsAPIConnection(jenkins, namespace.Name)
			defer cleanUpFunc()
			waitForJobCreation(jenkinsClient, jobID)
			verifyJobCanBeRun(jenkinsClient, jobID)

			jenkins = getJenkins(jenkins.Namespace, jenkins.Name)
			lastDoneBackup := jenkins.Status.LastBackup
			restartJenkinsMasterPod(jenkins)
			waitForRecreateJenkinsMasterPod(jenkins)
			waitForJenkinsUserConfigurationToComplete(jenkins)
			jenkins = getJenkins(jenkins.Namespace, jenkins.Name)
			Expect(jenkins.Status.RestoredBackup).To(BeNumerically("<=", lastDoneBackup))
			jenkinsClient2, cleanUpFunc2 := verifyJenkinsAPIConnection(jenkins, namespace.Name)
			defer cleanUpFunc2()
			waitForJobCreation(jenkinsClient2, jobID)
			verifyJobBuildsAfterRestoreBackup(jenkinsClient2, jobID)

			jenkins = getJenkins(jenkins.Namespace, jenkins.Name)
			lastDoneBackup = jenkins.Status.LastBackup
			resetJenkinsStatus(jenkins)
			waitForJenkinsUserConfigurationToComplete(jenkins)
			jenkins = getJenkins(jenkins.Namespace, jenkins.Name)

			Expect(jenkins.Status.RestoredBackup).To(BeNumerically("<=", lastDoneBackup))
		})
	})
})