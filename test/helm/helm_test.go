package helm

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/test/e2e"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// +kubebuilder:scaffold:imports
)

const jenkinsCRName = "jenkins"

var _ = Describe("Jenkins Controller", func() {
	var (
		namespace *corev1.Namespace
	)

	BeforeEach(func() {
		namespace = e2e.CreateNamespace()
	})
	AfterEach(func() {
		cmd := exec.Command("../../bin/helm", "delete", "jenkins", "--namespace", namespace.Name)
		output, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), string(output))

		e2e.ShowLogsIfTestHasFailed(CurrentGinkgoTestDescription().Failed, namespace.Name)
		e2e.DestroyNamespace(namespace)
	})

	Context("Deploys jenkins operator with helm charts with default values", func() {
		It("Deploys Jenkins operator and configures default Jenkins instance", func() {
			jenkins := &v1alpha2.Jenkins{
				TypeMeta: v1alpha2.JenkinsTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      jenkinsCRName,
					Namespace: namespace.Name,
				},
			}

			cmd := exec.Command("../../bin/helm", "upgrade", "jenkins", "../../chart/jenkins-operator", "--namespace", namespace.Name, "--debug",
				"--set-string", fmt.Sprintf("jenkins.namespace=%s", namespace.Name),
				"--set-string", fmt.Sprintf("jenkins.image=%s", "jenkins/jenkins:2.375.3-lts"),
				"--set-string", fmt.Sprintf("operator.image=%s", *imageName), "--install")
			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(output))

			e2e.WaitForJenkinsBaseConfigurationToComplete(jenkins)
			e2e.WaitForJenkinsUserConfigurationToComplete(jenkins)

		})
	})
})

var _ = Describe("Jenkins Controller with security validator", func() {

	var (
		namespace     *corev1.Namespace
		seedJobs      = &[]v1alpha2.SeedJob{}
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
		invalidPlugins = []v1alpha2.Plugin{
			{Name: "simple-theme-plugin", Version: "0.6"},
			{Name: "audit-trail", Version: "3.8"},
			{Name: "github", Version: "1.31.0"},
		}
		validPlugins = []v1alpha2.Plugin{
			{Name: "simple-theme-plugin", Version: "136.v23a_15f86c53d"},
			{Name: "audit-trail", Version: "3.11"},
			{Name: "github", Version: "1.36.0"},
		}
	)

	BeforeEach(func() {
		namespace = e2e.CreateNamespace()
	})
	AfterEach(func() {
		cmd := exec.Command("../../bin/helm", "delete", "jenkins", "--namespace", namespace.Name)
		output, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), string(output))

		e2e.ShowLogsIfTestHasFailed(CurrentGinkgoTestDescription().Failed, namespace.Name)
		e2e.DestroyNamespace(namespace)
	})

	Context("When Jenkins CR contains plugins with security warnings", func() {
		It("Denies creating a jenkins CR with a warning", func() {
			By("Deploying the operator along with webhook and cert-manager")
			cmd := exec.Command("../../bin/helm", "upgrade", "jenkins", "../../chart/jenkins-operator", "--namespace", namespace.Name, "--debug",
				"--set-string", fmt.Sprintf("jenkins.namespace=%s", namespace.Name),
				"--set-string", fmt.Sprintf("operator.image=%s", *imageName),
				"--set", fmt.Sprintf("jenkins.securityValidator=%t", true),
				"--set", fmt.Sprintf("jenkins.enabled=%t", false),
				"--set", fmt.Sprintf("webhook.enabled=%t", true), "--install")
			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(output))

			By("Waiting for the operator to fetch the plugin data")
			time.Sleep(time.Duration(200) * time.Second)

			By("Denying a create request for a Jenkins custom resource")
			jenkins := e2e.RenderJenkinsCR(jenkinsCRName, namespace.Name, seedJobs, groovyScripts, casc, "")
			jenkins.Spec.Master.Plugins = invalidPlugins
			jenkins.Spec.ValidateSecurityWarnings = true
			Expect(e2e.K8sClient.Create(context.TODO(), jenkins)).Should(MatchError("admission webhook \"vjenkins.kb.io\" denied the request: security vulnerabilities detected in the following user-defined plugins: \ngithub:1.31.0"))
		})
	})
	Context("When Jenkins CR doesn't contain plugins with security warnings", func() {
		It("Jenkins instance is successfully created", func() {
			By("Deploying the operator along with webhook and cert-manager")
			cmd := exec.Command("../../bin/helm", "upgrade", "jenkins", "../../chart/jenkins-operator", "--namespace", namespace.Name, "--debug",
				"--set-string", fmt.Sprintf("jenkins.namespace=%s", namespace.Name),
				"--set-string", fmt.Sprintf("operator.image=%s", *imageName),
				"--set", fmt.Sprintf("webhook.enabled=%t", true),
				"--set", fmt.Sprintf("jenkins.enabled=%t", false), "--install")
			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(output))

			By("Waiting for the operator to fetch the plugin data ")
			time.Sleep(time.Duration(200) * time.Second)

			By("Creating a Jenkins custom resource with some plugins having security warnings but validation is turned off")
			jenkins := e2e.RenderJenkinsCR(jenkinsCRName, namespace.Name, seedJobs, groovyScripts, casc, "")
			jenkins.Spec.Master.Plugins = validPlugins
			jenkins.Spec.ValidateSecurityWarnings = true
			Expect(e2e.K8sClient.Create(context.TODO(), jenkins)).Should(Succeed())
			e2e.WaitForJenkinsBaseConfigurationToComplete(jenkins)
			e2e.WaitForJenkinsUserConfigurationToComplete(jenkins)
		})
	})
})
