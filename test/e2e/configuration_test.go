package e2e

import (
	"context"
	"fmt"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/constants"
	"github.com/jenkinsci/kubernetes-operator/pkg/groovy"
	"github.com/jenkinsci/kubernetes-operator/pkg/plugins"

	"github.com/bndr/gojenkins"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const e2e = "e2e"

var expectedBasePluginsList = []plugins.Plugin{
	plugins.Must(plugins.New("configuration-as-code:1569.vb_72405b_80249")),
	plugins.Must(plugins.New("git:4.14.2")),
	plugins.Must(plugins.New("kubernetes:3734.v562b_b_a_627ea_c")),
	plugins.Must(plugins.New("kubernetes-credentials-provider:1.206.v7ce2cf7b_0c8b")),
	plugins.Must(plugins.New("job-dsl:1.81")),
	plugins.Must(plugins.New("workflow-aggregator:590.v6a_d052e5a_a_b_5")),
	plugins.Must(plugins.New("workflow-job:1254.v3f64639b_11dd")),
}

func createUserConfigurationSecret(namespace string, stringData map[string]string) {
	By("creating user configuration secret")

	userConfiguration := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      userConfigurationSecretName,
			Namespace: namespace,
		},
		StringData: stringData,
	}

	_, _ = fmt.Fprintf(GinkgoWriter, "User configuration secret %+v\n", *userConfiguration)
	Expect(K8sClient.Create(context.TODO(), userConfiguration)).Should(Succeed())
}

func createUserConfigurationConfigMap(namespace string, numberOfExecutorsSecretKeyName string, systemMessage string) {
	By("creating user configuration config map")

	userConfiguration := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      userConfigurationConfigMapName,
			Namespace: namespace,
		},
		Data: map[string]string{
			"1-set-executors.groovy": fmt.Sprintf(`
import jenkins.model.Jenkins

Jenkins.instance.setNumExecutors(new Integer(secrets['%s']))
Jenkins.instance.save()`, numberOfExecutorsSecretKeyName),
			"1-casc.yaml": fmt.Sprintf(`
jenkins:
  systemMessage: "%s"`, systemMessage),
			"2-casc.yaml": `
unclassified:
  location:
    url: http://external-jenkins-url:8080`,
		},
	}

	_, _ = fmt.Fprintf(GinkgoWriter, "User configuration %+v\n", *userConfiguration)
	Expect(K8sClient.Create(context.TODO(), userConfiguration)).Should(Succeed())
}

func createDefaultLimitsForContainersInNamespace(namespace string) {
	By("creating default limits for containers in namespace")

	limitRange := &corev1.LimitRange{
		ObjectMeta: metav1.ObjectMeta{
			Name:      e2e,
			Namespace: namespace,
		},
		Spec: corev1.LimitRangeSpec{
			Limits: []corev1.LimitRangeItem{
				{
					Type: corev1.LimitTypeContainer,
					DefaultRequest: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:    resource.MustParse("128m"),
						corev1.ResourceMemory: resource.MustParse("256Mi"),
					},
					Default: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:    resource.MustParse("256m"),
						corev1.ResourceMemory: resource.MustParse("512Mi"),
					},
				},
			},
		},
	}

	_, _ = fmt.Fprintf(GinkgoWriter, "LimitRange %+v\n", *limitRange)
	Expect(K8sClient.Create(context.TODO(), limitRange)).Should(Succeed())
}

func verifyJenkinsMasterPodAttributes(jenkins *v1alpha2.Jenkins) {
	By("creating Jenkins master pod properly")

	jenkinsPod := getJenkinsMasterPod(jenkins)
	jenkins = getJenkins(jenkins.Namespace, jenkins.Name)

	assertMapContainsElementsFromAnotherMap(jenkins.Spec.Master.Annotations, jenkinsPod.ObjectMeta.Annotations)
	Expect(jenkinsPod.Spec.NodeSelector).Should(Equal(jenkins.Spec.Master.NodeSelector))

	Expect(jenkinsPod.Spec.Containers[0].Name).Should(Equal(resources.JenkinsMasterContainerName))
	Expect(jenkinsPod.Spec.Containers).Should(HaveLen(len(jenkins.Spec.Master.Containers)))

	if jenkins.Spec.Master.SecurityContext == nil {
		jenkins.Spec.Master.SecurityContext = &corev1.PodSecurityContext{}
	}
	Expect(jenkinsPod.Spec.SecurityContext).Should(Equal(jenkins.Spec.Master.SecurityContext))
	Expect(jenkinsPod.Spec.Containers[0].Command).Should(Equal(jenkins.Spec.Master.Containers[0].Command))

	Expect(jenkinsPod.Labels).Should(Equal(resources.GetJenkinsMasterPodLabels(*jenkins)))
	Expect(jenkinsPod.Spec.PriorityClassName).Should(Equal(jenkins.Spec.Master.PriorityClassName))

	for _, actualContainer := range jenkinsPod.Spec.Containers {
		if actualContainer.Name == resources.JenkinsMasterContainerName {
			verifyContainer(resources.NewJenkinsMasterContainer(jenkins), actualContainer)
			continue
		}

		var expectedContainer *corev1.Container
		for _, jenkinsContainer := range jenkins.Spec.Master.Containers {
			if jenkinsContainer.Name == actualContainer.Name {
				tmp := resources.ConvertJenkinsContainerToKubernetesContainer(jenkinsContainer)
				expectedContainer = &tmp
			}
		}

		if expectedContainer == nil {
			Fail(fmt.Sprintf("Container '%+v' not found in pod", actualContainer))
			continue
		}

		verifyContainer(*expectedContainer, actualContainer)
	}

	for _, expectedVolume := range jenkins.Spec.Master.Volumes {
		volumeFound := false
		for _, actualVolume := range jenkinsPod.Spec.Volumes {
			if expectedVolume.Name == actualVolume.Name {
				volumeFound = true
				Expect(actualVolume).Should(Equal(expectedVolume))
			}
		}

		if !volumeFound {
			Fail(fmt.Sprintf("Missing volume '+%v', actaul volumes '%+v'", expectedVolume, jenkinsPod.Spec.Volumes))
		}
	}
}

func verifyContainer(expected corev1.Container, actual corev1.Container) {
	Expect(actual.Args).Should(Equal(expected.Args), expected.Name)
	Expect(actual.Command).Should(Equal(expected.Command), expected.Name)
	Expect(actual.Env).Should(ConsistOf(expected.Env), expected.Name)
	Expect(actual.EnvFrom).Should(Equal(expected.EnvFrom), expected.Name)
	Expect(actual.Image).Should(Equal(expected.Image), expected.Name)
	Expect(actual.ImagePullPolicy).Should(Equal(expected.ImagePullPolicy), expected.Name)
	Expect(actual.Lifecycle).Should(Equal(expected.Lifecycle), expected.Name)
	Expect(actual.LivenessProbe).Should(Equal(expected.LivenessProbe), expected.Name)
	Expect(actual.Ports).Should(Equal(expected.Ports), expected.Name)
	Expect(actual.ReadinessProbe).Should(Equal(expected.ReadinessProbe), expected.Name)
	Expect(actual.Resources).Should(Equal(expected.Resources), expected.Name)
	Expect(actual.SecurityContext).Should(Equal(expected.SecurityContext), expected.Name)
	Expect(actual.WorkingDir).Should(Equal(expected.WorkingDir), expected.Name)
	if !base.CompareContainerVolumeMounts(expected, actual) {
		Fail(fmt.Sprintf("Volume mounts are different in container '%s': expected '%+v', actual '%+v'",
			expected.Name, expected.VolumeMounts, expected.VolumeMounts))
	}
}

func verifyPlugins(jenkinsClient jenkinsclient.Jenkins, jenkins *v1alpha2.Jenkins) {
	By("installing plugins in Jenkins instance")

	installedPlugins, err := jenkinsClient.GetPlugins(1)
	Expect(err).NotTo(HaveOccurred())

	for _, basePlugin := range expectedBasePluginsList {
		if found, ok := isPluginValid(installedPlugins, basePlugin); !ok {
			Fail(fmt.Sprintf("Invalid plugin '%s', actual '%+v'", basePlugin, found))
		}
	}

	for _, userPlugin := range jenkins.Spec.Master.Plugins {
		plugin := plugins.Plugin{Name: userPlugin.Name, Version: userPlugin.Version}
		if found, ok := isPluginValid(installedPlugins, plugin); !ok {
			Fail(fmt.Sprintf("Invalid plugin '%s', actual '%+v'", plugin, found))
		}
	}
}

func isPluginValid(plugins *gojenkins.Plugins, requiredPlugin plugins.Plugin) (*gojenkins.Plugin, bool) {
	p := plugins.Contains(requiredPlugin.Name)
	if p == nil {
		return p, false
	}

	if !p.Active || !p.Enabled || p.Deleted {
		return p, false
	}

	return p, requiredPlugin.Version == p.Version
}

func verifyUserConfiguration(jenkinsClient jenkinsclient.Jenkins, amountOfExecutors int, systemMessage string) {
	By("configuring Jenkins by groovy scripts")

	checkConfigurationViaGroovyScript := fmt.Sprintf(`
if (!new Integer(%d).equals(Jenkins.instance.numExecutors)) {
	throw new Exception("Configuration via groovy scripts failed")
}`, amountOfExecutors)
	logs, err := jenkinsClient.ExecuteScript(checkConfigurationViaGroovyScript)
	Expect(err).NotTo(HaveOccurred(), logs)

	checkSecretLoaderViaGroovyScript := fmt.Sprintf(`
if (!new Integer(%d).equals(new Integer(secrets['NUMBER_OF_EXECUTORS']))) {
	throw new Exception("Secret not found by given key: NUMBER_OF_EXECUTORS")
}`, amountOfExecutors)

	loader := groovy.AddSecretsLoaderToGroovyScript("/var/jenkins/groovy-scripts-secrets")
	logs, err = jenkinsClient.ExecuteScript(loader(checkSecretLoaderViaGroovyScript))
	Expect(err).NotTo(HaveOccurred(), logs)

	By("configuring Jenkins by CasC")
	checkConfigurationAsCode := fmt.Sprintf(`
if (!"%s".equals(Jenkins.instance.systemMessage)) {
	throw new Exception("Configuration as code failed")
}`, systemMessage)
	logs, err = jenkinsClient.ExecuteScript(checkConfigurationAsCode)
	Expect(err).NotTo(HaveOccurred(), logs)
}

func verifyServices(jenkins *v1alpha2.Jenkins) {
	By("creating Jenkins services properly")

	jenkinsHTTPService := getJenkinsService(jenkins, "http")
	jenkinsSlaveService := getJenkinsService(jenkins, "slave")
	Expect(jenkinsHTTPService.Spec.Ports[0].TargetPort).Should(Equal(intstr.IntOrString{IntVal: constants.DefaultHTTPPortInt32, Type: intstr.Int}))
	Expect(jenkinsSlaveService.Spec.Ports[0].TargetPort).Should(Equal(intstr.IntOrString{IntVal: constants.DefaultSlavePortInt32, Type: intstr.Int}))
}

func assertMapContainsElementsFromAnotherMap(expected map[string]string, actual map[string]string) {
	for key, expectedValue := range expected {
		actualValue, keyExists := actual[key]
		if !keyExists {
			Fail(fmt.Sprintf("key '%s' doesn't exist in map '%+v'", key, actual))
			continue
		}
		Expect(actualValue).Should(Equal(expectedValue), key, expected, actual)
	}
}
