package e2e

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os/exec"
	"time"

	"github.com/jenkinsci/kubernetes-operator/test/e2e/testdata/reconciliation-ignore/identitywebhook"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

var identityFixtureLoaded bool

func installIdentityWebhook(namespace string) func() {
	if !identityFixtureLoaded {
		cmd := exec.Command("make", "identity-webhook-kind-load")
		cmd.Dir = "../.."
		cmd.Stdout = GinkgoWriter
		cmd.Stderr = GinkgoWriter
		Expect(cmd.Run()).To(Succeed())
		identityFixtureLoaded = true
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	Expect(err).NotTo(HaveOccurred())
	template := &x509.Certificate{SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:               pkix.Name{CommonName: "identity-webhook"},
		DNSNames:              []string{"identity-webhook." + namespace + ".svc"},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	Expect(err).NotTo(HaveOccurred())
	cert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "identity-tls",
		Namespace: namespace},
		Data: map[string][]byte{"tls.crt": cert,
			"tls.key": pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY",
				Bytes: x509.MarshalPKCS1PrivateKey(key)})}}
	Expect(K8sClient.Create(context.Background(), secret)).To(Succeed())
	Expect(K8sClient.Create(context.Background(),
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "identity-config",
			Namespace: namespace},
			Data: map[string]string{"credentials.json": `{"type":"external_account","audience":"identity-test","subject_token_type":"urn:ietf:params:oauth:token-type:jwt","token_url":"https://sts.googleapis.com/v1/token","credential_source":{"file":"/var/run/secrets/google/token"}}`}})).To(Succeed())
	labels := map[string]string{"app": "identity-webhook"}
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "identity-webhook",
		Namespace: namespace},
		Spec: appsv1.DeploymentSpec{Replicas: ptr.To(int32(1)),
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "webhook",
					Image:           "identity-webhook:test",
					ImagePullPolicy: corev1.PullNever,
					Ports:           []corev1.ContainerPort{{ContainerPort: 8443}},
					VolumeMounts: []corev1.VolumeMount{{Name: "tls",
						MountPath: "/tls",
						ReadOnly:  true}},
					ReadinessProbe: &corev1.Probe{ProbeHandler: corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{Path: "/healthz",
						Port:   intstr.FromInt32(8443),
						Scheme: corev1.URISchemeHTTPS}}}}},
					Volumes: []corev1.Volume{{Name: "tls",
						VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: secret.Name}}}}}}}}
	Expect(K8sClient.Create(context.Background(), deployment)).To(Succeed())
	Expect(K8sClient.Create(context.Background(),
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "identity-webhook",
			Namespace: namespace},
			Spec: corev1.ServiceSpec{Selector: labels,
				Ports: []corev1.ServicePort{{Port: 443,
					TargetPort: intstr.FromInt32(8443)}}}})).To(Succeed())
	Eventually(func() int32 {
		current := &appsv1.Deployment{}
		if K8sClient.Get(context.Background(), types.NamespacedName{Name: deployment.Name, Namespace: namespace}, current) != nil {
			return 0
		}
		return current.Status.AvailableReplicas
	}, 2*time.Minute, 2*time.Second).Should(Equal(int32(1)))
	webhook := &admissionv1.MutatingWebhookConfiguration{ObjectMeta: metav1.ObjectMeta{Name: "identity-" + namespace},
		Webhooks: []admissionv1.MutatingWebhook{{Name: "identity.test.jenkins.io",
			AdmissionReviewVersions: []string{"v1"},
			SideEffects:             ptr.To(admissionv1.SideEffectClassNone),
			FailurePolicy:           ptr.To(admissionv1.Fail),
			NamespaceSelector:       &metav1.LabelSelector{MatchLabels: map[string]string{"kubernetes.io/metadata.name": namespace}},
			ObjectSelector: &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{{Key: identitywebhook.ProfileLabel,
				Operator: metav1.LabelSelectorOpExists}}},
			ClientConfig: admissionv1.WebhookClientConfig{CABundle: cert,
				Service: &admissionv1.ServiceReference{Name: "identity-webhook",
					Namespace: namespace,
					Path:      ptr.To("/mutate"),
					Port:      ptr.To(int32(443))}},
			Rules: []admissionv1.RuleWithOperations{{Operations: []admissionv1.OperationType{admissionv1.Create},
				Rule: admissionv1.Rule{APIGroups: []string{""},
					APIVersions: []string{"v1"},
					Resources:   []string{"pods"}}}}}}}
	Expect(K8sClient.Create(context.Background(), webhook)).To(Succeed())
	return func() { Expect(K8sClient.Delete(context.Background(), webhook)).To(Succeed()) }
}
