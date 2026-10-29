package e2e

import (
	"context"
	"strings"
	"time"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/constants"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/util/retry"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Jenkins controller recovery", func() {
	var namespace *corev1.Namespace
	var jenkins *v1alpha2.Jenkins

	BeforeEach(func() {
		namespace = CreateNamespace()
		customization := v1alpha2.Customization{Configurations: []v1alpha2.ConfigMapRef{}}
		jenkins = RenderJenkinsCR(e2e, namespace.Name, nil, v1alpha2.GroovyScripts{Customization: customization}, v1alpha2.ConfigurationAsCode{Customization: customization}, "")
		jenkins.Spec.Master.Containers[0].Name = "invalid-master"
		Expect(K8sClient.Create(context.TODO(), jenkins)).To(Succeed())
	})

	AfterEach(func() {
		defer DestroyNamespace(namespace)
		ShowLogsIfTestHasFailed(CurrentGinkgoTestDescription().Failed, namespace.Name)
	})

	It("recovers after correcting an invalid Jenkins configuration", func() {
		By("waiting for a warning about the invalid master container")
		Eventually(func() (bool, error) {
			events := &corev1.EventList{}
			if err := K8sClient.List(context.TODO(), events, k8sclient.InNamespace(namespace.Name)); err != nil {
				return false, err
			}
			for _, event := range events.Items {
				if event.InvolvedObject.UID == jenkins.UID && event.Type == corev1.EventTypeWarning && event.Source.Component == constants.OperatorName && strings.Contains(event.Message, "first container in spec.master.containers") {
					return true, nil
				}
			}
			return false, nil
		}, 2*time.Minute, retryInterval).Should(BeTrue())

		By("checking that invalid configuration did not start Jenkins")
		current := getJenkins(jenkins.Namespace, jenkins.Name)
		Expect(current.Status.BaseConfigurationCompletedTime).To(BeNil())
		Expect(current.Status.UserConfigurationCompletedTime).To(BeNil())
		pods := &corev1.PodList{}
		Expect(K8sClient.List(context.TODO(), pods, k8sclient.InNamespace(namespace.Name))).To(Succeed())
		Expect(pods.Items).To(BeEmpty())

		By("correcting the existing Jenkins CR")
		originalUID := jenkins.UID
		Expect(retry.RetryOnConflict(retry.DefaultRetry, func() error {
			current := &v1alpha2.Jenkins{}
			if err := K8sClient.Get(context.TODO(), k8sclient.ObjectKeyFromObject(jenkins), current); err != nil {
				return err
			}
			current.Spec.Master.Containers[0].Name = resources.JenkinsMasterContainerName
			return K8sClient.Update(context.TODO(), current)
		})).To(Succeed())

		WaitForJenkinsBaseConfigurationToComplete(jenkins)
		WaitForJenkinsUserConfigurationToComplete(jenkins)
		Expect(getJenkins(jenkins.Namespace, jenkins.Name).UID).To(Equal(originalUID))
		master := getJenkinsMasterPod(jenkins).Spec.Containers[0]
		Expect(master.ReadinessProbe.HTTPGet.Path).To(Equal("/health"))
		Expect(master.LivenessProbe.HTTPGet.Path).To(Equal("/health"))
		jenkinsClient, cleanUpFunc := verifyJenkinsAPIConnection(jenkins, namespace.Name)
		defer cleanUpFunc()
		_, err := jenkinsClient.ExecuteScript("assert jenkins.model.Jenkins.get().isQuietingDown() == false")
		Expect(err).NotTo(HaveOccurred())
	})
})
