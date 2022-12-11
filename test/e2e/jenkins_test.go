package e2e

import (
	"context"
	"fmt"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/constants"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	userConfigurationConfigMapName = "user-config"
	userConfigurationSecretName    = "user-secret"
)

func getJenkins(namespace, name string) *v1alpha2.Jenkins {
	jenkins := &v1alpha2.Jenkins{}
	namespaceName := types.NamespacedName{Namespace: namespace, Name: name}
	Expect(K8sClient.Get(context.TODO(), namespaceName, jenkins)).Should(Succeed())
	return jenkins
}

func getJenkinsMasterPod(jenkins *v1alpha2.Jenkins) *corev1.Pod {
	lo := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(resources.GetJenkinsMasterPodLabels(*jenkins)),
		Namespace:     jenkins.Namespace,
	}
	pods := &corev1.PodList{}
	Expect(K8sClient.List(context.TODO(), pods, lo)).Should(Succeed())
	Expect(pods.Items).Should(HaveLen(1), fmt.Sprintf("Jenkins pod not found, pod list: %+v", pods.Items))
	return &pods.Items[0]
}

func createJenkinsCRSafeRestart(name, namespace string, seedJob *[]v1alpha2.SeedJob, groovyScripts v1alpha2.GroovyScripts, casc v1alpha2.ConfigurationAsCode, priorityClassName string) *v1alpha2.Jenkins {
	var seedJobs []v1alpha2.SeedJob
	if seedJob != nil {
		seedJobs = append(seedJobs, *seedJob...)
	}

	jenkins := &v1alpha2.Jenkins{
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
							PeriodSeconds:       int32(10),
						},
						LivenessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Path:   "/login",
									Port:   intstr.FromString("http"),
									Scheme: corev1.URISchemeHTTP,
								},
							},
							InitialDelaySeconds: int32(100),
							TimeoutSeconds:      int32(5),
							FailureThreshold:    int32(12),
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
		},
	}
	jenkins.Spec.Roles = []rbacv1.RoleRef{
		{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     resources.GetResourceName(jenkins),
		},
	}
	updateJenkinsCR(jenkins)

	_, _ = fmt.Fprintf(GinkgoWriter, "Jenkins CR %+v\n", *jenkins)

	Expect(K8sClient.Create(context.TODO(), jenkins)).Should(Succeed())

	return jenkins
}

func createJenkinsAPIClientFromServiceAccount(jenkins *v1alpha2.Jenkins, jenkinsAPIURL string) (jenkinsclient.Jenkins, error) {
	podName := resources.GetJenkinsMasterPodName(jenkins)

	clientSet, err := kubernetes.NewForConfig(Cfg)
	if err != nil {
		return nil, err
	}
	config := configuration.Configuration{Jenkins: jenkins, ClientSet: *clientSet, Config: Cfg}
	r := base.New(config, jenkinsclient.JenkinsAPIConnectionSettings{})

	token, _, err := r.Configuration.Exec(podName, resources.JenkinsMasterContainerName, []string{"cat", "/var/run/secrets/kubernetes.io/serviceaccount/token"})
	if err != nil {
		return nil, err
	}

	return jenkinsclient.NewBearerTokenAuthorization(jenkinsAPIURL, token.String())
}

func createJenkinsAPIClientFromSecret(jenkins *v1alpha2.Jenkins, jenkinsAPIURL string) (jenkinsclient.Jenkins, error) {
	_, _ = fmt.Fprintf(GinkgoWriter, "Creating Jenkins API client from secret\n")

	adminSecret := &corev1.Secret{}
	namespaceName := types.NamespacedName{Namespace: jenkins.Namespace, Name: resources.GetOperatorCredentialsSecretName(jenkins)}
	if err := K8sClient.Get(context.TODO(), namespaceName, adminSecret); err != nil {
		return nil, err
	}

	return jenkinsclient.NewUserAndPasswordAuthorization(
		jenkinsAPIURL,
		string(adminSecret.Data[resources.OperatorCredentialsSecretUserNameKey]),
		string(adminSecret.Data[resources.OperatorCredentialsSecretTokenKey]),
	)
}

func verifyJenkinsAPIConnection(jenkins *v1alpha2.Jenkins, namespace string) (jenkinsclient.Jenkins, func()) {
	By("establishing Jenkins API connection")

	var service corev1.Service
	err := K8sClient.Get(context.TODO(), types.NamespacedName{
		Namespace: jenkins.Namespace,
		Name:      resources.GetJenkinsHTTPServiceName(jenkins),
	}, &service)
	Expect(err).NotTo(HaveOccurred())

	podName := resources.GetJenkinsMasterPodName(jenkins)
	port, cleanUpFunc, waitFunc, portForwardFunc, err := setupPortForwardToPod(namespace, podName, int(constants.DefaultHTTPPortInt32))
	Expect(err).NotTo(HaveOccurred())
	go portForwardFunc()
	waitFunc()

	jenkinsAPIURL := jenkinsclient.JenkinsAPIConnectionSettings{
		Hostname:    "localhost",
		Port:        port,
		UseNodePort: false,
	}.BuildJenkinsAPIUrl(service.Name, service.Namespace, service.Spec.Ports[0].Port, service.Spec.Ports[0].NodePort)

	var jenkinsClient jenkinsclient.Jenkins
	if jenkins.Spec.JenkinsAPISettings.AuthorizationStrategy == v1alpha2.ServiceAccountAuthorizationStrategy {
		jenkinsClient, err = createJenkinsAPIClientFromServiceAccount(jenkins, jenkinsAPIURL)
	} else {
		jenkinsClient, err = createJenkinsAPIClientFromSecret(jenkins, jenkinsAPIURL)
	}

	if err != nil {
		defer cleanUpFunc()
		Fail(err.Error())
	}

	_, _ = fmt.Fprintf(GinkgoWriter, "I can establish connection to Jenkins API\n")
	return jenkinsClient, cleanUpFunc
}

func restartJenkinsMasterPod(jenkins *v1alpha2.Jenkins) {
	_, _ = fmt.Fprintf(GinkgoWriter, "Restarting Jenkins master pod\n")
	jenkinsPod := getJenkinsMasterPod(jenkins)
	Expect(K8sClient.Delete(context.TODO(), jenkinsPod)).Should(Succeed())

	Eventually(func() (bool, error) {
		jenkinsPod = getJenkinsMasterPod(jenkins)
		return jenkinsPod.DeletionTimestamp != nil, nil
	}, 30*retryInterval, retryInterval).Should(BeTrue())

	_, _ = fmt.Fprintf(GinkgoWriter, "Jenkins master pod has been restarted\n")
}

func getJenkinsService(jenkins *v1alpha2.Jenkins, serviceKind string) *corev1.Service {
	service := &corev1.Service{}
	serviceName := constants.OperatorName + "-" + serviceKind + "-" + jenkins.ObjectMeta.Name
	Expect(K8sClient.Get(context.TODO(), client.ObjectKey{Name: serviceName, Namespace: jenkins.Namespace}, service)).Should(Succeed())

	return service
}
