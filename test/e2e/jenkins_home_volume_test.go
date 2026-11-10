package e2e

import (
	"context"
	"fmt"
	"time"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

var _ = Describe("Jenkins home storage", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() { namespace = CreateNamespace() })
	AfterEach(func() {
		ShowLogsIfTestHasFailed(CurrentGinkgoTestDescription().Failed, namespace.Name)
		DestroyNamespace(namespace)
	})

	It("provisions ephemeral home storage and replaces it with the Pod", func() {
		ctx := context.Background()
		customization := v1alpha2.Customization{Configurations: []v1alpha2.ConfigMapRef{}}
		j := RenderJenkinsCR("home-volume", namespace.Name, nil, v1alpha2.GroovyScripts{Customization: customization}, v1alpha2.ConfigurationAsCode{Customization: customization}, "")
		j.Spec.Master.Plugins = nil
		j.Spec.Master.Containers = j.Spec.Master.Containers[:1]
		j.Spec.Master.HomeVolume = &v1alpha2.JenkinsHomeVolume{
			EmptyDir:  &corev1.EmptyDirVolumeSource{},
			Ephemeral: &corev1.PersistentVolumeClaimSpec{AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}, Resources: corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("2Gi")}}},
		}
		By("rejecting conflicting home storage sources through the API")
		Expect(apierrors.IsInvalid(K8sClient.Create(ctx, j))).To(BeTrue())
		j.Spec.Master.HomeVolume.EmptyDir = nil
		Expect(K8sClient.Create(ctx, j)).To(Succeed())
		WaitForJenkinsBaseConfigurationToComplete(j)
		WaitForJenkinsUserConfigurationToComplete(j)

		pod := getJenkinsMasterPod(j)
		claimKey := types.NamespacedName{Namespace: namespace.Name, Name: pod.Name + "-" + resources.JenkinsHomeVolumeName}
		claim := &corev1.PersistentVolumeClaim{}
		Eventually(func() (corev1.PersistentVolumeClaimPhase, error) {
			err := K8sClient.Get(ctx, claimKey, claim)
			return claim.Status.Phase, err
		}, 3*time.Minute, retryInterval).Should(Equal(corev1.ClaimBound))
		Expect(metav1.IsControlledBy(claim, pod)).To(BeTrue())
		Expect(claim.Spec.StorageClassName).NotTo(BeNil())
		Expect(*claim.Spec.StorageClassName).NotTo(BeEmpty())
		size := claim.Spec.Resources.Requests[corev1.ResourceStorage]
		Expect(size.Cmp(resource.MustParse("2Gi"))).To(BeZero())
		var home *corev1.Volume
		for i := range pod.Spec.Volumes {
			if pod.Spec.Volumes[i].Name == resources.JenkinsHomeVolumeName {
				home = &pod.Spec.Volumes[i]
			}
		}
		Expect(home).NotTo(BeNil())
		Expect(home.Ephemeral).NotTo(BeNil())
		Expect(home.EmptyDir).To(BeNil())
		Expect(home.Ephemeral.VolumeClaimTemplate.Spec.StorageClassName).To(BeNil())
		clientSet, err := kubernetes.NewForConfig(Cfg)
		Expect(err).NotTo(HaveOccurred())
		config := configuration.Configuration{Jenkins: j, Client: K8sClient, ClientSet: *clientSet, Config: Cfg}
		exec := func(command string) {
			_, stderr, err := config.Exec(pod.Name, resources.JenkinsMasterContainerName, []string{"sh", "-c", command})
			Expect(err).NotTo(HaveOccurred(), stderr.String())
		}
		exec(`printf present > "$JENKINS_HOME/home-volume-test"`)
		oldPodUID, oldClaimUID := pod.UID, claim.UID

		By("keeping the Pod, PVC and home data stable across reconciliations")
		for cycle := 0; cycle < 3; cycle++ {
			Expect(retry.RetryOnConflict(retry.DefaultRetry, func() error {
				if err := K8sClient.Get(ctx, types.NamespacedName{Name: j.Name, Namespace: j.Namespace}, j); err != nil {
					return err
				}
				j.Annotations = map[string]string{"home-volume.test/reconcile": fmt.Sprint(time.Now().UnixNano())}
				return K8sClient.Update(ctx, j)
			})).To(Succeed())
			Consistently(func() (bool, error) {
				if err := K8sClient.Get(ctx, types.NamespacedName{Name: pod.Name, Namespace: namespace.Name}, pod); err != nil {
					return false, err
				}
				if err := K8sClient.Get(ctx, claimKey, claim); err != nil {
					return false, err
				}
				return pod.UID == oldPodUID && pod.DeletionTimestamp == nil && claim.UID == oldClaimUID, nil
			}, 10*time.Second, time.Second).Should(BeTrue())
		}
		exec(`test "$(cat "$JENKINS_HOME/home-volume-test")" = present`)

		By("replacing the Pod and its owned PVC with fresh home storage")
		Expect(K8sClient.Delete(ctx, pod)).To(Succeed())
		Eventually(func() (bool, error) {
			err := K8sClient.Get(ctx, types.NamespacedName{Name: pod.Name, Namespace: namespace.Name}, pod)
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			if err != nil {
				return false, err
			}
			err = K8sClient.Get(ctx, claimKey, claim)
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			if err != nil {
				return false, err
			}
			if pod.UID == oldPodUID || pod.DeletionTimestamp != nil || claim.UID == oldClaimUID || claim.Status.Phase != corev1.ClaimBound || !metav1.IsControlledBy(claim, pod) {
				return false, nil
			}
			for _, condition := range pod.Status.Conditions {
				if condition.Type == corev1.PodReady {
					return condition.Status == corev1.ConditionTrue, nil
				}
			}
			return false, nil
		}, 15*time.Minute, 5*time.Second).Should(BeTrue())
		WaitForJenkinsBaseConfigurationToComplete(j)
		WaitForJenkinsUserConfigurationToComplete(j)
		exec(`test ! -e "$JENKINS_HOME/home-volume-test"`)
	})
})
