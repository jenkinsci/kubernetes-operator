package e2e

import (
	"context"
	"time"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
)

func checkBaseConfigurationCompleteTimeIsNotSet(jenkins *v1alpha2.Jenkins) {
	By("checking that Base Configuration's complete time is not set")

	Eventually(func() (bool, error) {
		actualJenkins := &v1alpha2.Jenkins{}
		err := K8sClient.Get(context.TODO(), types.NamespacedName{Name: jenkins.Name, Namespace: jenkins.Namespace}, actualJenkins)
		if err != nil {
			return false, err
		}
		return actualJenkins.Status.BaseConfigurationCompletedTime == nil, nil
	}, 1100*time.Second, time.Second).Should(BeTrue())
}
