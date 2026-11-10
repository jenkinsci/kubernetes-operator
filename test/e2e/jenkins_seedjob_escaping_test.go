package e2e

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/groovy"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Jenkins seed-job escaping", func() {
	var namespace *corev1.Namespace

	BeforeEach(func() { namespace = CreateNamespace() })
	AfterEach(func() {
		ShowLogsIfTestHasFailed(CurrentGinkgoTestDescription().Failed, namespace.Name)
		DestroyNamespace(namespace)
	})

	It("keeps seed-job values literal and user Groovy scripts executable", func() {
		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "user-groovy", Namespace: namespace.Name},
			Data: map[string]string{"compatibility.groovy": `
def answer = 6 * 7
System.setProperty('operator.seedjobs.userScript', "answer-${answer}")
`},
		}
		Expect(K8sClient.Create(context.Background(), configMap)).To(Succeed())
		seed := seedJobConfig{SeedJob: v1alpha2.SeedJob{
			ID:                    "escaping",
			JenkinsCredentialType: v1alpha2.NoJenkinsCredentialCredentialType,
			RepositoryURL:         "https://github.com/jenkinsci/kubernetes-operator.git",
			RepositoryBranch:      "master",
			Targets:               `jobs/*.groovy"); System.setProperty("operator.seedjobs.injected", "yes"); println("${System.setProperty('operator.seedjobs.injected', 'yes')}") //`,
			PollSCM:               "H/15 * * * *",
			BuildPeriodically:     "H 1 * * *\nH 2 * * *",
			IgnoreMissingFiles:    true,
		}}
		groovyScripts := v1alpha2.GroovyScripts{Customization: v1alpha2.Customization{Configurations: []v1alpha2.ConfigMapRef{{Name: configMap.Name}}}}
		casc := v1alpha2.ConfigurationAsCode{Customization: v1alpha2.Customization{Configurations: []v1alpha2.ConfigMapRef{}}}
		jenkins := RenderJenkinsCR("seedjob-escaping", namespace.Name, &[]v1alpha2.SeedJob{seed.SeedJob}, groovyScripts, casc, "")
		jenkins.Spec.Master.Plugins = nil
		jenkins.Spec.Master.Containers = jenkins.Spec.Master.Containers[:1]
		Expect(K8sClient.Create(context.Background(), jenkins)).To(Succeed())
		WaitForJenkinsBaseConfigurationToComplete(jenkins)
		WaitForJenkinsUserConfigurationToComplete(jenkins)
		jenkinsClient, cleanup := verifyJenkinsAPIConnection(jenkins, namespace.Name)
		defer cleanup()

		By("checking seed-job settings and unchanged user Groovy execution")
		verifySeedJobProperties(jenkinsClient, seed)
		check := fmt.Sprintf(`
assert System.getProperty('operator.seedjobs.injected') == null
assert System.getProperty('operator.seedjobs.userScript') == 'answer-42'
def job = jenkins.model.Jenkins.get().getItem('escaping-job-dsl-seed')
assert job != null
assert job.getBuildersList()[0].getTargets().getBytes('UTF-8').encodeBase64().toString() == '%s'
`, base64.StdEncoding.EncodeToString([]byte(seed.Targets)))
		output, err := jenkinsClient.ExecuteScript(check)
		Expect(err).NotTo(HaveOccurred(), output)

		By("round-tripping Groovy literals without interpolation or side effects")
		corpus := []string{
			"", "jobs/*.groovy", `it's "quoted"`, `a\\b\`, "a\r\nb\tc",
			"café 日本語 😀", "\u0085\u2028\u2029", `\u0027\u000a\u005c`, "$HOME ${name}",
			`${System.setProperty('operator.seedjobs.injected', 'yes')}`,
			`'); System.setProperty("operator.seedjobs.injected", "yes"); //`,
			seed.Targets,
		}
		var controls strings.Builder
		for r := rune(0); r < 0x20; r++ {
			controls.WriteRune(r)
		}
		controls.WriteRune(0x7f)
		corpus = append(corpus, controls.String())
		var roundTrip strings.Builder
		for _, value := range corpus {
			fmt.Fprintf(&roundTrip, "assert (%s).getBytes('UTF-8').encodeBase64().toString() == '%s'\n", groovy.StringLiteral(value), base64.StdEncoding.EncodeToString([]byte(value)))
		}
		roundTrip.WriteString("assert System.getProperty('operator.seedjobs.injected') == null\n")
		output, err = jenkinsClient.ExecuteScript(roundTrip.String())
		Expect(err).NotTo(HaveOccurred(), output)
	})
})
