package e2e

import (
	"context"
	"time"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func configureAuthorizationToUnSecure(namespace, configMapName string) {
	limitRange := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: namespace,
		},
		Data: map[string]string{
			"8-set-unsecured-authorization.groovy": `
import hudson.security.*

def jenkins = jenkins.model.Jenkins.getInstance()

def strategy = new AuthorizationStrategy.Unsecured()
jenkins.setAuthorizationStrategy(strategy)
jenkins.save()
`,
		},
	}

	Expect(K8sClient.Create(context.TODO(), limitRange)).Should(Succeed())
}

func checkIfAuthorizationStrategyUnsecuredIsSet(jenkinsClient jenkinsclient.Jenkins) {
	By("checking if Authorization Strategy Unsecured is set")

	logs, err := jenkinsClient.ExecuteScript(`
	import hudson.security.*

	def jenkins = jenkins.model.Jenkins.getInstance()

	if (!(jenkins.getAuthorizationStrategy() instanceof AuthorizationStrategy.Unsecured)) {
	  throw new Exception('AuthorizationStrategy.Unsecured is not set')
	}
	`)
	Expect(err).NotTo(HaveOccurred(), logs)
}

func checkBaseConfigurationCompleteTimeIsNotSet(jenkins *v1alpha2.Jenkins) {
	By("checking that Base Configuration's complete time is not set")

	Eventually(func() (bool, error) {
		actualJenkins := &v1alpha2.Jenkins{}
		err := K8sClient.Get(context.TODO(), types.NamespacedName{Name: jenkins.Name, Namespace: jenkins.Namespace}, actualJenkins)
		if err != nil {
			return false, err
		}
		return actualJenkins.Status.BaseConfigurationCompletedTime == nil, nil
	}, time.Duration(110)*retryInterval, time.Second).Should(BeTrue())
}
