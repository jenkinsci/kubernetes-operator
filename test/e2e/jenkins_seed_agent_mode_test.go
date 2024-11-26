package e2e

import (
	"context"
	"time"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/ptr"
)

var _ = Describe("Seed agent scheduling", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() { namespace = CreateNamespace() })
	AfterEach(func() {
		ShowLogsIfTestHasFailed(CurrentGinkgoTestDescription().Failed, namespace.Name)
		DestroyNamespace(namespace)
	})

	It("reserves the seed agent by default and allows unlabeled jobs after opting out", func() {
		customization := v1alpha2.Customization{Configurations: []v1alpha2.ConfigMapRef{}}
		seed := v1alpha2.SeedJob{ID: "seed-agent-mode", JenkinsCredentialType: v1alpha2.NoJenkinsCredentialCredentialType, Targets: "cicd/jobs/*.jenkins", RepositoryURL: "https://github.com/jenkinsci/kubernetes-operator.git", RepositoryBranch: "master"}
		j := RenderJenkinsCR("seed-agent-mode", namespace.Name, &[]v1alpha2.SeedJob{seed}, v1alpha2.GroovyScripts{Customization: customization}, v1alpha2.ConfigurationAsCode{Customization: customization}, "")
		j.Spec.Master.Plugins = nil
		j.Spec.Master.Containers = j.Spec.Master.Containers[:1]
		Expect(K8sClient.Create(context.Background(), j)).To(Succeed())
		WaitForJenkinsBaseConfigurationToComplete(j)
		WaitForJenkinsUserConfigurationToComplete(j)
		Expect(ptr.Deref(j.Spec.SeedJobRestrictJobsToLabel, false)).To(BeTrue())
		api, cleanup := verifyJenkinsAPIConnection(j, namespace.Name)
		defer cleanup()
		check := func(script string) error { _, err := api.ExecuteScript(script); return err }
		Eventually(func() error {
			return check(`
def j = jenkins.model.Jenkins.get()
def agent = j.getNode('seed-job-agent')
assert agent != null && agent.toComputer().isOnline()
assert agent.getMode() == hudson.model.Node.Mode.EXCLUSIVE
assert j.getMode() == hudson.model.Node.Mode.EXCLUSIVE
`)
		}, 2*time.Minute, retryInterval).Should(Succeed())
		output, err := api.ExecuteScript(`
def j = jenkins.model.Jenkins.get()
def labeled = j.createProject(hudson.model.FreeStyleProject, 'labeled')
labeled.setAssignedLabel(j.getLabel('seed-job-agent'))
labeled.save()
labeled.scheduleBuild2(0)
def unlabeled = j.createProject(hudson.model.FreeStyleProject, 'unlabeled')
unlabeled.save()
unlabeled.scheduleBuild2(0)
`)
		Expect(err).NotTo(HaveOccurred(), output)
		Eventually(func() error {
			return check(`
def build = jenkins.model.Jenkins.get().getItem('labeled').getLastBuild()
assert build != null && !build.isBuilding() && build.getResult().toString() == 'SUCCESS'
assert build.getBuiltOnStr() == 'seed-job-agent'
`)
		}, 2*time.Minute, retryInterval).Should(Succeed())
		Consistently(func() error {
			return check(`
def j = jenkins.model.Jenkins.get()
def job = j.getItem('unlabeled')
assert job.getLastBuild() == null
assert j.getQueue().getItem(job) != null
`)
		}, 10*time.Second, 2*time.Second).Should(Succeed())
		By("opting out on the existing agent")
		Expect(retry.RetryOnConflict(retry.DefaultRetry, func() error {
			if err := K8sClient.Get(context.Background(), types.NamespacedName{Name: j.Name, Namespace: j.Namespace}, j); err != nil {
				return err
			}
			j.Spec.SeedJobRestrictJobsToLabel = ptr.To(false)
			return K8sClient.Update(context.Background(), j)
		})).To(Succeed())
		Eventually(func() error {
			return check(`
def j = jenkins.model.Jenkins.get()
assert j.getNode('seed-job-agent').getMode() == hudson.model.Node.Mode.NORMAL
def build = j.getItem('unlabeled').getLastBuild()
assert build != null && !build.isBuilding() && build.getResult().toString() == 'SUCCESS'
assert build.getBuiltOnStr() == 'seed-job-agent'
`)
		}, 2*time.Minute, retryInterval).Should(Succeed())
	})
})
