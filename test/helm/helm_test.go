package helm

import (
	"fmt"
	"os/exec"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/test/e2e"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// +kubebuilder:scaffold:imports
)

var _ = Describe("Jenkins controller", func() {
	var (
		namespace *corev1.Namespace
	)

	BeforeEach(func() {
		namespace = e2e.CreateNamespace()
	})
	AfterEach(func() {
		showLogsIfTestHasFailed(CurrentGinkgoTestDescription().Failed, namespace.Name)
		e2e.DestroyNamespace(namespace)
	})
	Context("when deploying Helm Chart to cluster", func() {
		It("creates Jenkins instance and configures it", func() {

			jenkins := &v1alpha2.Jenkins{
				TypeMeta: v1alpha2.JenkinsTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "jenkins",
					Namespace: namespace.Name,
				},
			}

			cmd := exec.Command("../../bin/helm", "upgrade", "jenkins", "../../chart/jenkins-operator", "--namespace", namespace.Name, "--debug",
				"--set-string", fmt.Sprintf("jenkins.namespace=%s", namespace.Name),
				"--set-string", fmt.Sprintf("operator.image=%s", *imageName), "--install")
			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(output))

			e2e.WaitForJenkinsBaseConfigurationToComplete(jenkins)
			e2e.WaitForJenkinsUserConfigurationToComplete(jenkins)

			cmd = exec.Command("../../bin/helm", "upgrade", "jenkins", "../../chart/jenkins-operator", "--namespace", namespace.Name, "--debug",
				"--set-string", fmt.Sprintf("jenkins.namespace=%s", namespace.Name),
				"--set-string", fmt.Sprintf("operator.image=%s", *imageName), "--install")
			output, err = cmd.CombinedOutput()

			Expect(err).NotTo(HaveOccurred(), string(output))

			e2e.WaitForJenkinsBaseConfigurationToComplete(jenkins)
			e2e.WaitForJenkinsUserConfigurationToComplete(jenkins)
		})
	})
})
