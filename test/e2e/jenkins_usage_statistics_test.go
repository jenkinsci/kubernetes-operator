package e2e

import (
	"context"
	"fmt"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Jenkins usage statistics", func() {
	var namespace *corev1.Namespace

	AfterEach(func() {
		if namespace != nil {
			ShowLogsIfTestHasFailed(CurrentGinkgoTestDescription().Failed, namespace.Name)
			DestroyNamespace(namespace)
		}
	})

	It("preserves Jenkins defaults and supports opting out of usage statistics", func() {
		for _, disabled := range []bool{false, true} {
			By(fmt.Sprintf("creating a fresh Jenkins instance with disableUsageStatistics=%t", disabled))
			namespace = CreateNamespace()
			customization := v1alpha2.Customization{Configurations: []v1alpha2.ConfigMapRef{}}
			jenkins := RenderJenkinsCR("usage-statistics", namespace.Name, nil, v1alpha2.GroovyScripts{Customization: customization}, v1alpha2.ConfigurationAsCode{Customization: customization}, "")
			jenkins.Spec.Master.DisableUsageStatistics = disabled
			jenkins.Spec.Master.Plugins = nil
			jenkins.Spec.Master.Containers = jenkins.Spec.Master.Containers[:1]
			Expect(K8sClient.Create(context.Background(), jenkins)).To(Succeed())
			WaitForJenkinsBaseConfigurationToComplete(jenkins)
			WaitForJenkinsUserConfigurationToComplete(jenkins)
			func() {
				jenkinsClient, cleanup := verifyJenkinsAPIConnection(jenkins, namespace.Name)
				defer cleanup()
				output, err := jenkinsClient.ExecuteScript(fmt.Sprintf(`
if (jenkins.model.Jenkins.get().isUsageStatisticsCollected() != %t) {
 throw new Exception("Unexpected Jenkins usage statistics setting")
}`, !disabled))
				Expect(err).NotTo(HaveOccurred(), output)
			}()
			DestroyNamespace(namespace)
			namespace = nil
		}
	})
})
