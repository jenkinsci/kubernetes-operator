package helm

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/constants"
	"github.com/jenkinsci/kubernetes-operator/test/e2e"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	// +kubebuilder:scaffold:imports
)

const JenkinsTestImage = "jenkins/jenkins:2.303.2-lts"

var _ = Describe("Jenkins Controller with webhook", func() {

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
					Name:      "jenkins",
					Namespace: namespace.Name,
				},
			}

			cmd := exec.Command("../../bin/helm", "upgrade", "jenkins", "../../chart/jenkins-operator", "--namespace", namespace.Name, "--debug",
				"--set-string", fmt.Sprintf("jenkins.namespace=%s", namespace.Name),
				"--set-string", fmt.Sprintf("jenkins.image=%s", "jenkins/jenkins:2.303.2-lts"),
				"--set-string", fmt.Sprintf("operator.image=%s", *imageName), "--install")
			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(output))

			e2e.WaitForJenkinsBaseConfigurationToComplete(jenkins)
			e2e.WaitForJenkinsUserConfigurationToComplete(jenkins)

		})
	})

	Context("Deploys jenkins operator with helm charts with validating webhook and jenkins instance disabled", func() {
		It("Deploys operator,denies creating a jenkins cr and creates jenkins cr with validation turned off", func() {

			By("Deploying the operator along with webhook and cert-manager")
			cmd := exec.Command("../../bin/helm", "upgrade", "jenkins", "../../chart/jenkins-operator", "--namespace", namespace.Name, "--debug",
				"--set-string", fmt.Sprintf("jenkins.namespace=%s", namespace.Name), "--set-string", fmt.Sprintf("operator.image=%s", *imageName),
				"--set-string", fmt.Sprintf("jenkins.image=%s", "jenkins/jenkins:2.303.3-lts"),
				"--set", fmt.Sprintf("webhook.enabled=%t", true), "--set", fmt.Sprintf("jenkins.enabled=%t", false), "--install", "--wait")
			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(output))

			By("Waiting for the operator to fetch the plugin data ")
			time.Sleep(time.Duration(200) * time.Second)

			By("Denying a create request for a Jenkins custom resource with some plugins having security warnings and validation is turned on")
			userplugins := []v1alpha2.Plugin{
				{Name: "simple-theme-plugin", Version: "0.6"},
				{Name: "audit-trail", Version: "3.5"},
				{Name: "github", Version: "1.29.0"},
			}
			jenkins := CreateJenkinsCR("jenkins", namespace.Name, userplugins, true)
			Expect(e2e.K8sClient.Create(context.TODO(), jenkins)).Should(MatchError("admission webhook \"vjenkins.kb.io\" denied the request: security vulnerabilities detected in the following user-defined plugins: \naudit-trail:3.5\ngithub:1.29.0"))

			By("Creating the Jenkins resource with plugins not having any security warnings and validation is turned on")
			userplugins = []v1alpha2.Plugin{
				{Name: "simple-theme-plugin", Version: "0.6"},
				{Name: "audit-trail", Version: "3.8"},
				{Name: "github", Version: "1.31.0"},
			}
			jenkins = CreateJenkinsCR("jenkins", namespace.Name, userplugins, true)
			Expect(e2e.K8sClient.Create(context.TODO(), jenkins)).Should(Succeed())
			e2e.WaitForJenkinsBaseConfigurationToComplete(jenkins)
			e2e.WaitForJenkinsUserConfigurationToComplete(jenkins)

		})

		It("Deploys operator, creates a jenkins cr and denies update request for another one", func() {
			By("Deploying the operator along with webhook and cert-manager")
			cmd := exec.Command("../../bin/helm", "upgrade", "jenkins", "../../chart/jenkins-operator", "--namespace", namespace.Name, "--debug",
				"--set-string", fmt.Sprintf("jenkins.namespace=%s", namespace.Name), "--set-string", fmt.Sprintf("operator.image=%s", *imageName),
				"--set-string", fmt.Sprintf("jenkins.image=%s", "jenkins/jenkins:2.303.3-lts"),
				"--set", fmt.Sprintf("webhook.enabled=%t", true), "--set", fmt.Sprintf("jenkins.enabled=%t", false), "--install", "--wait")
			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(output))

			By("Waiting for the operator to fetch the plugin data ")
			time.Sleep(time.Duration(200) * time.Second)

			By("Creating a Jenkins custom resource with some plugins having security warnings but validation is turned off")
			userplugins := []v1alpha2.Plugin{
				{Name: "simple-theme-plugin", Version: "0.6"},
				{Name: "audit-trail", Version: "3.5"},
				{Name: "github", Version: "1.29.0"},
			}
			jenkins := CreateJenkinsCR("jenkins", namespace.Name, userplugins, false)
			Expect(e2e.K8sClient.Create(context.TODO(), jenkins)).Should(Succeed())
			e2e.WaitForJenkinsBaseConfigurationToComplete(jenkins)
			e2e.WaitForJenkinsUserConfigurationToComplete(jenkins)

			By("Failing to update the Jenkins custom resource because some plugins have security warnings and validation is turned on")
			userplugins = []v1alpha2.Plugin{
				{Name: "vncviewer", Version: "1.7"},
				{Name: "build-timestamp", Version: "1.0.3"},
				{Name: "deployit-plugin", Version: "7.5.5"},
				{Name: "github-branch-source", Version: "2.0.7"},
				{Name: "aws-lambda-cloud", Version: "0.4"},
				{Name: "groovy", Version: "1.31"},
				{Name: "google-login", Version: "1.2"},
			}
			jenkins.Spec.Master.Plugins = userplugins
			jenkins.Spec.ValidateSecurityWarnings = true
			Expect(e2e.K8sClient.Update(context.TODO(), jenkins)).Should(MatchError("admission webhook \"vjenkins.kb.io\" denied the request: security vulnerabilities detected in the following user-defined plugins: \nvncviewer:1.7\ndeployit-plugin:7.5.5\ngithub-branch-source:2.0.7\ngroovy:1.31\ngoogle-login:1.2"))

		})
	})
})

func CreateJenkinsCR(name string, namespace string, userPlugins []v1alpha2.Plugin, validateSecurityWarnings bool) *v1alpha2.Jenkins {
	jenkins := &v1alpha2.Jenkins{
		TypeMeta: v1alpha2.JenkinsTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha2.JenkinsSpec{
			GroovyScripts: v1alpha2.GroovyScripts{
				Customization: v1alpha2.Customization{
					Configurations: []v1alpha2.ConfigMapRef{},
					Secret: v1alpha2.SecretRef{
						Name: "",
					},
				},
			},
			ConfigurationAsCode: v1alpha2.ConfigurationAsCode{
				Customization: v1alpha2.Customization{
					Configurations: []v1alpha2.ConfigMapRef{},
					Secret: v1alpha2.SecretRef{
						Name: "",
					},
				},
			},
			Master: v1alpha2.JenkinsMaster{
				Containers: []v1alpha2.Container{
					{
						Name:  resources.JenkinsMasterContainerName,
						Image: JenkinsTestImage,
						Env: []corev1.EnvVar{
							{
								Name:  "TEST_ENV",
								Value: "test_env_value",
							},
						},
						ReadinessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Path:   "/login",
									Port:   intstr.FromString("http"),
									Scheme: corev1.URISchemeHTTP,
								},
							},
							InitialDelaySeconds: int32(100),
							TimeoutSeconds:      int32(10),
							FailureThreshold:    int32(40),
							SuccessThreshold:    int32(1),
							PeriodSeconds:       int32(10),
						},
						LivenessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Path:   "/login",
									Port:   intstr.FromString("http"),
									Scheme: corev1.URISchemeHTTP,
								},
							},
							InitialDelaySeconds: int32(80),
							TimeoutSeconds:      int32(4),
							FailureThreshold:    int32(30),
							SuccessThreshold:    int32(1),
							PeriodSeconds:       int32(5),
						},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("1"),
								corev1.ResourceMemory: resource.MustParse("500Mi"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("1000m"),
								corev1.ResourceMemory: resource.MustParse("3Gi"),
							},
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "plugins-cache",
								MountPath: "/usr/share/jenkins/ref/plugins",
							},
						},
					},
					{
						Name:  "envoyproxy",
						Image: "envoyproxy/envoy:v1.20.0",
					},
				},
				Plugins:               userPlugins,
				DisableCSRFProtection: false,
				NodeSelector:          map[string]string{"kubernetes.io/os": "linux"},
				Volumes: []corev1.Volume{
					{
						Name: "plugins-cache",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
				},
			},
			ValidateSecurityWarnings: validateSecurityWarnings,
			Service: v1alpha2.Service{
				Type: corev1.ServiceTypeNodePort,
				Port: constants.DefaultHTTPPortInt32,
			},
			JenkinsAPISettings: v1alpha2.JenkinsAPISettings{AuthorizationStrategy: v1alpha2.CreateUserAuthorizationStrategy},
		},
	}

	return jenkins
}
