package e2e

import (
	"context"
	"fmt"
	"time"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/constants"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

const JenkinsTestImage = "jenkins/jenkins:2.375.2-lts"

var (
	Cfg       *rest.Config
	K8sClient client.Client
	testEnv   *envtest.Environment

	hostname    *string
	port        *int
	useNodePort *bool
)

func CreateNamespace() *corev1.Namespace {
	ginkgo.By("creating temporary namespace")

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("ns%d", time.Now().Unix()),
		},
	}
	gomega.Expect(K8sClient.Create(context.TODO(), namespace)).Should(gomega.Succeed())
	return namespace
}

func DestroyNamespace(namespace *corev1.Namespace) {
	ginkgo.By("deleting temporary namespace")

	gomega.Expect(K8sClient.Delete(context.TODO(), namespace)).Should(gomega.Succeed())

	gomega.Eventually(func() (bool, error) {
		namespaces := &corev1.NamespaceList{}
		err := K8sClient.List(context.TODO(), namespaces)
		if err != nil {
			return false, err
		}

		exists := false
		for _, namespaceItem := range namespaces.Items {
			if namespaceItem.Name == namespace.Name {
				exists = true
				break
			}
		}

		return !exists, nil
	}, time.Second*120, time.Second).Should(gomega.BeTrue())
}

func RenderJenkinsCR(name, namespace string, seedJob *[]v1alpha2.SeedJob, groovyScripts v1alpha2.GroovyScripts, casc v1alpha2.ConfigurationAsCode, priorityClassName string) *v1alpha2.Jenkins {
	var seedJobs []v1alpha2.SeedJob
	if seedJob != nil {
		seedJobs = append(seedJobs, *seedJob...)
	}

	return &v1alpha2.Jenkins{
		TypeMeta: v1alpha2.JenkinsTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha2.JenkinsSpec{
			GroovyScripts:       groovyScripts,
			ConfigurationAsCode: casc,
			Master: v1alpha2.JenkinsMaster{
				Annotations: map[string]string{"test": "label"},
				Containers: []v1alpha2.Container{
					{
						Name:  resources.JenkinsMasterContainerName,
						Image: JenkinsTestImage,
						Env: []corev1.EnvVar{
							{
								Name:  "TEST_ENV",
								Value: "test_env_value",
							},
						},
						ReadinessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Path:   "/login",
									Port:   intstr.FromString("http"),
									Scheme: corev1.URISchemeHTTP,
								},
							},
							InitialDelaySeconds: int32(100),
							TimeoutSeconds:      int32(4),
							FailureThreshold:    int32(12),
							SuccessThreshold:    int32(1),
							PeriodSeconds:       int32(1),
						},
						LivenessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Path:   "/login",
									Port:   intstr.FromString("http"),
									Scheme: corev1.URISchemeHTTP,
								},
							},
							InitialDelaySeconds: int32(80),
							TimeoutSeconds:      int32(4),
							FailureThreshold:    int32(30),
							SuccessThreshold:    int32(1),
							PeriodSeconds:       int32(5),
						},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("250m"),
								corev1.ResourceMemory: resource.MustParse("500Mi"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("1000m"),
								corev1.ResourceMemory: resource.MustParse("3Gi"),
							},
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "plugins-cache",
								MountPath: "/usr/share/jenkins/ref/plugins",
							},
						},
					},
					{
						Name:  "envoyproxy",
						Image: "envoyproxy/envoy-alpine:v1.14.1",
					},
				},
				Plugins: []v1alpha2.Plugin{
					{Name: "audit-trail", Version: "3.11"},
					{Name: "simple-theme-plugin", Version: "136.v23a_15f86c53d"},
					{Name: "github", Version: "1.36.0"},
					{Name: "devoptics", Version: "1.1973", DownloadURL: "https://jenkins-updates.cloudbees.com/download/plugins/devoptics/1.1973/devoptics.hpi"},
				},
				PriorityClassName: priorityClassName,
				NodeSelector:      map[string]string{"kubernetes.io/os": "linux"},
				Volumes: []corev1.Volume{
					{
						Name: "plugins-cache",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
				},
			},
			SeedJobs: seedJobs,
			Service: v1alpha2.Service{
				Type: corev1.ServiceTypeNodePort,
				Port: constants.DefaultHTTPPortInt32,
			},
			Roles: []rbacv1.RoleRef{
				{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "Role",
					Name:     "jenkins-operator-jenkins",
				},
			},
		},
	}
}
