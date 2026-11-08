package e2e

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/test/e2e/testdata/reconciliation-ignore/identitywebhook"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func identityJenkins(namespace string) *v1alpha2.Jenkins {
	j := RenderJenkinsCR("identity", namespace, nil, v1alpha2.GroovyScripts{}, v1alpha2.ConfigurationAsCode{}, "")
	j.Spec.GroovyScripts.Configurations = []v1alpha2.ConfigMapRef{}
	j.Spec.ConfigurationAsCode.Configurations = []v1alpha2.ConfigMapRef{}
	j.Spec.Master.Plugins = nil
	j.Spec.Master.Containers = j.Spec.Master.Containers[:1]
	j.Spec.Master.Containers[0].ReadinessProbe.InitialDelaySeconds = 10
	return j
}

func identityPod(j *v1alpha2.Jenkins) (*corev1.Pod, error) {
	pod := &corev1.Pod{}
	err := K8sClient.Get(context.Background(), types.NamespacedName{Name: resources.GetJenkinsMasterPodName(j), Namespace: j.Namespace}, pod)
	return pod, err
}

func stableIdentityPod(j *v1alpha2.Jenkins, uid types.UID) {
	for cycle := 0; cycle < 3; cycle++ {
		// A CR update explicitly queues reconciliation even after the controller becomes idle.
		Expect(retry.RetryOnConflict(retry.DefaultRetry, func() error {
			current := &v1alpha2.Jenkins{}
			if err := K8sClient.Get(context.Background(), types.NamespacedName{Name: j.Name, Namespace: j.Namespace}, current); err != nil {
				return err
			}
			if current.Annotations == nil {
				current.Annotations = map[string]string{}
			}
			current.Annotations["identity.test/reconcile"] = fmt.Sprint(time.Now().UnixNano())
			return K8sClient.Update(context.Background(), current)
		})).To(Succeed())
		Consistently(func() (types.UID, error) {
			pod, err := identityPod(j)
			if err == nil && pod.DeletionTimestamp != nil {
				return "", nil
			}
			return pod.UID, err
		}, 10*time.Second, time.Second).Should(Equal(uid))
	}
	Expect(K8sClient.Get(context.Background(), types.NamespacedName{Name: j.Name, Namespace: j.Namespace}, j)).To(Succeed())
}

func replaceIdentityPod(j *v1alpha2.Jenkins, old types.UID) *corev1.Pod {
	var current *corev1.Pod
	Eventually(func() (bool, error) {
		pod, err := identityPod(j)
		current = pod
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return err == nil && pod.UID != old && pod.DeletionTimestamp == nil, err
	}, 3*time.Minute, 2*time.Second).Should(BeTrue())
	Eventually(func() bool {
		pod, err := identityPod(j)
		if err != nil || pod.UID == old {
			return false
		}
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady {
				return condition.Status == corev1.ConditionTrue
			}
		}
		return false
	}, 10*time.Minute, 5*time.Second).Should(BeTrue())

	WaitForJenkinsBaseConfigurationToComplete(j)
	return current
}

var _ = Describe("Jenkins reconciliation ignore", func() {
	var namespace *corev1.Namespace
	var cleanup func()
	BeforeEach(func() {
		namespace = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{GenerateName: "identity-"}}
		Expect(K8sClient.Create(context.Background(), namespace)).To(Succeed())
	})
	AfterEach(func() {
		defer DestroyNamespace(namespace)
		if cleanup != nil {
			cleanup()
			cleanup = nil
		}
		ShowLogsIfTestHasFailed(CurrentGinkgoTestDescription().Failed, namespace.Name)
	})
	identityScenario := func(profile string) {
		cleanup = installIdentityWebhook(namespace.Name)
		j := identityJenkins(namespace.Name)
		j.Spec.Master.Labels = map[string]string{identitywebhook.ProfileLabel: profile, identitywebhook.ManagedKey: "desired"}
		j.Spec.Master.Annotations[identitywebhook.ManagedKey] = "desired"
		var injectedEnv []string
		ignore := v1alpha2.JenkinsReconciliationIgnore{Labels: []string{identitywebhook.ManagedKey}, Annotations: []string{identitywebhook.ManagedKey}}
		switch profile {
		case "azure", "azure-prefix":
			injectedEnv = []string{"AZURE_CLIENT_ID", "AZURE_TENANT_ID", "AZURE_AUTHORITY_HOST", "AZURE_FEDERATED_TOKEN_FILE"}
			j.Spec.ServiceAccount.Annotations = map[string]string{"azure.workload.identity/client-id": "test-client"}
			j.Spec.Master.Labels["azure.workload.identity/use"] = "true"
		case "irsa":
			injectedEnv = []string{"AWS_ROLE_ARN", "AWS_WEB_IDENTITY_TOKEN_FILE", "AWS_REGION", "AWS_DEFAULT_REGION", "AWS_STS_REGIONAL_ENDPOINTS"}
			j.Spec.ServiceAccount.Annotations = map[string]string{"eks.amazonaws.com/role-arn": "arn:aws:iam::123456789012:role/test"}
		case "eks-pod-identity":
			injectedEnv = []string{"AWS_CONTAINER_AUTHORIZATION_TOKEN_FILE", "AWS_CONTAINER_CREDENTIALS_FULL_URI"}
		case "gcp":
			injectedEnv = []string{"GOOGLE_APPLICATION_CREDENTIALS"}
			ignore.Volumes = []string{"gcp-identity-token"}
		}
		// Exercise injected mounts and envs in another compared regular container as well.
		j.Spec.Master.Containers = append(j.Spec.Master.Containers, v1alpha2.Container{Name: "identity-auxiliary", Image: JenkinsTestImage, Command: []string{"/usr/bin/tini", "--", "sleep", "infinity"}})
		// Bound the no-policy control to one observed recreation, then enable the policy.
		j.Spec.Reconciliation.IgnoreWellKnownInjections = ptr.To(false)
		j.Spec.Reconciliation.Ignore = v1alpha2.JenkinsReconciliationIgnore{}
		Expect(K8sClient.Create(context.Background(), j)).To(Succeed())
		var firstUID types.UID
		Eventually(func() (types.UID, error) {
			p, err := identityPod(j)
			if apierrors.IsNotFound(err) {
				return "", nil
			}
			firstUID = p.UID
			return firstUID, err
		}, time.Minute, time.Second).ShouldNot(BeEmpty())
		Eventually(func() (bool, error) {
			p, err := identityPod(j)
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			return p.UID != firstUID, err
		}, time.Minute, time.Second).Should(BeTrue())
		Expect(retry.RetryOnConflict(retry.DefaultRetry, func() error {
			if err := K8sClient.Get(context.Background(), types.NamespacedName{Name: j.Name, Namespace: j.Namespace}, j); err != nil {
				return err
			}
			j.Spec.Reconciliation.Ignore = ignore
			j.Spec.Reconciliation.IgnoreWellKnownInjections = ptr.To(true)
			return K8sClient.Update(context.Background(), j)
		})).To(Succeed())
		Eventually(func() (string, error) {
			events := &corev1.EventList{}
			if err := K8sClient.List(context.Background(), events, client.InNamespace(j.Namespace)); err != nil {
				return "", err
			}
			var messages []string
			for _, event := range events.Items {
				messages = append(messages, event.Message)
			}
			return strings.Join(messages, "\n"), nil
		}, 30*time.Second, time.Second).Should(ContainSubstring("Env has changed"))
		WaitForJenkinsBaseConfigurationToComplete(j)
		Expect(j.Spec.Reconciliation.Ignore).To(Equal(ignore))
		Expect(ptr.Deref(j.Spec.Reconciliation.IgnoreWellKnownInjections, true)).To(BeTrue())
		pod, err := identityPod(j)
		Expect(err).NotTo(HaveOccurred())
		Expect(pod.Annotations[identitywebhook.ManagedKey]).To(Equal("mutated"))
		Expect(pod.Labels[identitywebhook.ManagedKey]).To(Equal("mutated"))
		for _, c := range pod.Spec.Containers {
			for _, name := range injectedEnv {
				found := false
				for _, env := range c.Env {
					if env.Name == name {
						found = true
					}
				}
				Expect(found).To(BeTrue(), "missing %s in %s", name, c.Name)
			}
		}
		sa := &corev1.ServiceAccount{}
		Expect(K8sClient.Get(context.Background(), types.NamespacedName{Name: pod.Spec.ServiceAccountName, Namespace: j.Namespace}, sa)).To(Succeed())
		for key, value := range j.Spec.ServiceAccount.Annotations {
			Expect(sa.Annotations[key]).To(Equal(value))
		}
		stableIdentityPod(j, pod.UID)
		// A retained env change must still replace the pod, with identity injection applied again.
		j.Spec.Master.Containers[0].Env = append(j.Spec.Master.Containers[0].Env, corev1.EnvVar{Name: "IDENTITY_TEST_CONTROL", Value: "changed"})
		Expect(K8sClient.Update(context.Background(), j)).To(Succeed())
		replacement := replaceIdentityPod(j, pod.UID)
		if profile == "azure-prefix" {
			volumeName := func(p *corev1.Pod) string {
				for _, v := range p.Spec.Volumes {
					if strings.HasPrefix(v.Name, "azure-workload-identity-reserved-") {
						return v.Name
					}
				}
				return ""
			}
			Expect(volumeName(pod)).NotTo(BeEmpty())
			Expect(volumeName(replacement)).NotTo(Equal(volumeName(pod)))
		}
		stableIdentityPod(j, replacement.UID)
	}
	It("tolerates workload identity admission for azure fixed-name", func() { identityScenario("azure") })
	It("tolerates workload identity admission for azure hashed-name", func() { identityScenario("azure-prefix") })
	It("tolerates workload identity admission for irsa", func() { identityScenario("irsa") })
	It("tolerates workload identity admission for EKS Pod Identity", func() { identityScenario("eks-pod-identity") })
	It("tolerates workload identity admission for generic gcp", func() { identityScenario("gcp") })

	It("keeps native GKE ServiceAccount identity stable without ignore rules", func() {
		j := identityJenkins(namespace.Name)
		j.Spec.ServiceAccount.Annotations = map[string]string{"iam.gke.io/gcp-service-account": "test@example.iam.gserviceaccount.com"}
		// Kind supplies this real node selector; native GKE identity uses metadata-server-enabled nodes.
		j.Spec.Master.NodeSelector = map[string]string{"kubernetes.io/os": "linux"}
		Expect(K8sClient.Create(context.Background(), j)).To(Succeed())
		WaitForJenkinsBaseConfigurationToComplete(j)
		pod, err := identityPod(j)
		Expect(err).NotTo(HaveOccurred())
		Expect(pod.Spec.NodeSelector).To(HaveKeyWithValue("kubernetes.io/os", "linux"))
		sa := &corev1.ServiceAccount{}
		key := types.NamespacedName{Name: pod.Spec.ServiceAccountName, Namespace: j.Namespace}
		Expect(K8sClient.Get(context.Background(), key, sa)).To(Succeed())
		Expect(sa.Annotations).To(HaveKeyWithValue("iam.gke.io/gcp-service-account", "test@example.iam.gserviceaccount.com"))
		sa.Annotations["identity.test/external"] = "preserved"
		Expect(K8sClient.Update(context.Background(), sa)).To(Succeed())
		stableIdentityPod(j, pod.UID)
		Expect(K8sClient.Get(context.Background(), key, sa)).To(Succeed())
		Expect(sa.Annotations).To(HaveKeyWithValue("identity.test/external", "preserved"))
		Expect(j.Spec.Reconciliation.Ignore.Env).To(BeEmpty())
	})
	It("normalizes ConfigMap defaultMode without ignore rules", func() {
		Expect(K8sClient.Create(context.Background(), &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "mode-test", Namespace: namespace.Name}, Data: map[string]string{"config": "test"}})).To(Succeed())
		j := identityJenkins(namespace.Name)
		j.Spec.Master.Volumes = append(j.Spec.Master.Volumes, corev1.Volume{Name: "mode-test", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "mode-test"}}}})
		j.Spec.Master.Containers[0].VolumeMounts = append(j.Spec.Master.Containers[0].VolumeMounts, corev1.VolumeMount{Name: "mode-test", MountPath: "/tmp/mode-test", ReadOnly: true})
		Expect(K8sClient.Create(context.Background(), j)).To(Succeed())
		WaitForJenkinsBaseConfigurationToComplete(j)
		pod, err := identityPod(j)
		Expect(err).NotTo(HaveOccurred())
		for _, volume := range pod.Spec.Volumes {
			if volume.Name == "mode-test" {
				Expect(volume.ConfigMap.DefaultMode).To(Equal(ptr.To(int32(0644))))
			}
		}
		stableIdentityPod(j, pod.UID)
		for i := range j.Spec.Master.Volumes {
			if j.Spec.Master.Volumes[i].Name == "mode-test" {
				j.Spec.Master.Volumes[i].ConfigMap.DefaultMode = ptr.To(int32(0600))
			}
		}
		Expect(K8sClient.Update(context.Background(), j)).To(Succeed())
		replacement := replaceIdentityPod(j, pod.UID)
		stableIdentityPod(j, replacement.UID)
	})
	It("rejects an empty volume name prefix through the API", func() {
		j := identityJenkins(namespace.Name)
		j.Spec.Reconciliation.Ignore.VolumeNamePrefixes = []string{""}
		err := K8sClient.Create(context.Background(), j)
		Expect(apierrors.IsInvalid(err)).To(BeTrue())
		Expect(err.Error()).To(ContainSubstring("volumeNamePrefixes[0]"))
	})
})
