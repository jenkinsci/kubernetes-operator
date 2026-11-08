package base

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/event"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func meshTestPod(mesh string, native bool) (*corev1.Pod, *corev1.Pod) {
	j := comparisonJenkins()
	if mesh == "istio" {
		j.Spec.Master.Annotations["sidecar.istio.io/rewriteAppHTTPProbers"] = "false"
	}
	expected := desiredPodForComparison(j)
	actual := expected.DeepCopy()
	proxy, init, volume := "istio-proxy", "istio-init", "istio-envoy"
	if mesh == "linkerd" {
		proxy, init, volume = "linkerd-proxy", "linkerd-init", "linkerd-identity-end-entity"
		actual.Annotations["linkerd.io/proxy-version"] = "edge-26.11.1"
		actual.Annotations["linkerd.io/created-by"] = "linkerd/proxy-injector edge-26.11.1"
		actual.Labels["linkerd.io/control-plane-ns"] = "linkerd"
	} else {
		status := map[string]any{"containers": []string{proxy}, "initContainers": []string{init}, "volumes": []string{volume}}
		if native {
			status["containers"] = []string{}
			status["initContainers"] = []string{init, proxy}
		}
		encoded, _ := json.Marshal(status)
		actual.Annotations["sidecar.istio.io/status"] = string(encoded)
	}
	sidecar := corev1.Container{Name: proxy, Image: "proxy:test", VolumeMounts: []corev1.VolumeMount{{Name: volume, MountPath: "/mesh"}}}
	actual.Spec.InitContainers = append(actual.Spec.InitContainers, corev1.Container{Name: init, Image: "init:test"})
	if native {
		sidecar.RestartPolicy = ptr.To(corev1.ContainerRestartPolicyAlways)
		actual.Spec.InitContainers = append(actual.Spec.InitContainers, sidecar)
	} else {
		actual.Spec.Containers = append(actual.Spec.Containers, sidecar)
	}
	actual.Spec.Volumes = append(actual.Spec.Volumes, corev1.Volume{Name: volume, VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}})
	return expected, actual
}

func TestMeshInjectionComparison(t *testing.T) {
	for _, mesh := range []string{"istio", "linkerd"} {
		for _, native := range []bool{false, true} {
			t.Run(mesh+map[bool]string{true: " native", false: " regular"}[native], func(t *testing.T) {
				expected, actual := meshTestPod(mesh, native)
				beforeExpected, beforeActual := expected.DeepCopy(), actual.DeepCopy()
				require.Empty(t, comparePodConfiguration(*expected, *actual, newPodComparisonPolicy(v1alpha2.JenkinsReconciliationIgnore{}, true)))
				require.NotEmpty(t, comparePodConfiguration(*expected, *actual, newPodComparisonPolicy(v1alpha2.JenkinsReconciliationIgnore{}, false)))
				require.Equal(t, beforeExpected, expected)
				require.Equal(t, beforeActual, actual)
				mutations := map[string]func(*corev1.Pod){
					"probe": func(p *corev1.Pod) { p.Spec.Containers[0].ReadinessProbe = &corev1.Probe{TimeoutSeconds: 99} },
					"image": func(p *corev1.Pod) { p.Spec.Containers[0].Image = "changed:image" },
					"env": func(p *corev1.Pod) {
						p.Spec.Containers[0].Env = append(p.Spec.Containers[0].Env, corev1.EnvVar{Name: "CUSTOM", Value: "changed"})
					},
					"sidecar": func(p *corev1.Pod) {
						p.Spec.Containers = append(p.Spec.Containers, corev1.Container{Name: "unknown-proxy", Image: "unknown:test"})
					},
					"volume": func(p *corev1.Pod) {
						p.Spec.Volumes = append(p.Spec.Volumes, corev1.Volume{Name: "unrelated", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}})
					},
					"declared metadata": func(p *corev1.Pod) { p.Labels["managed"] = "changed" },
					"missing markers": func(p *corev1.Pod) {
						delete(p.Annotations, "sidecar.istio.io/status")
						delete(p.Annotations, "linkerd.io/created-by")
					},
				}
				for name, mutate := range mutations {
					t.Run(name, func(t *testing.T) {
						changed := actual.DeepCopy()
						mutate(changed)
						require.NotEmpty(t, comparePodConfiguration(*expected, *changed, newPodComparisonPolicy(v1alpha2.JenkinsReconciliationIgnore{}, true)))
					})
				}
				actual.Labels["managed"] = "changed"
				require.Empty(t, comparePodConfiguration(*expected, *actual, newPodComparisonPolicy(v1alpha2.JenkinsReconciliationIgnore{Labels: []string{"managed"}}, true)))
			})
		}
	}
}

func TestMeshDeclaredComponentsStayManaged(t *testing.T) {
	for _, mesh := range []string{"istio", "linkerd"} {
		t.Run(mesh, func(t *testing.T) {
			expected, actual := meshTestPod(mesh, false)
			expected.Spec.Containers = append(expected.Spec.Containers, *actual.Spec.Containers[len(actual.Spec.Containers)-1].DeepCopy())
			expected.Spec.Volumes = append(expected.Spec.Volumes, *actual.Spec.Volumes[len(actual.Spec.Volumes)-1].DeepCopy())
			actual.Spec.Containers[len(actual.Spec.Containers)-1].Image = "changed:proxy"
			require.NotEmpty(t, comparePodConfiguration(*expected, *actual, newPodComparisonPolicy(v1alpha2.JenkinsReconciliationIgnore{}, true)))
			actual.Spec.Containers[len(actual.Spec.Containers)-1].Image = "proxy:test"
			actual.Spec.Volumes[len(actual.Spec.Volumes)-1].EmptyDir.Medium = corev1.StorageMediumMemory
			require.NotEmpty(t, comparePodConfiguration(*expected, *actual, newPodComparisonPolicy(v1alpha2.JenkinsReconciliationIgnore{}, true)))
		})
	}
}

func TestIstioProbeOptOutRequired(t *testing.T) {
	expected, actual := meshTestPod("istio", true)
	delete(expected.Annotations, "sidecar.istio.io/rewriteAppHTTPProbers")
	require.Contains(t, comparePodConfiguration(*expected, *actual, newPodComparisonPolicy(v1alpha2.JenkinsReconciliationIgnore{}, true))[0].String(), "rewriteAppHTTPProbers")
	expected, actual = meshTestPod("istio", true)
	actual.Annotations["sidecar.istio.io/status"] = "{"
	require.NotEmpty(t, comparePodConfiguration(*expected, *actual, newPodComparisonPolicy(v1alpha2.JenkinsReconciliationIgnore{}, true)))
	actual.Annotations["sidecar.istio.io/status"] = `{"initContainers":["istio-proxy"],"volumes":["istio-envoy","unrelated"]}`
	actual.Spec.Volumes = append(actual.Spec.Volumes, corev1.Volume{Name: "unrelated"})
	require.NotEmpty(t, comparePodConfiguration(*expected, *actual, newPodComparisonPolicy(v1alpha2.JenkinsReconciliationIgnore{}, true)))
}

func TestMeshReadiness(t *testing.T) {
	for _, mesh := range []string{"istio", "linkerd"} {
		for _, native := range []bool{false, true} {
			t.Run(mesh+map[bool]string{true: " native", false: " regular"}[native], func(t *testing.T) {
				_, pod := meshTestPod(mesh, native)
				j := comparisonJenkins()
				j.Namespace = "default"
				pod.Name = "jenkins-" + j.Name
				pod.Namespace = j.Namespace
				pod.Status.Phase = corev1.PodRunning
				for _, c := range pod.Spec.Containers {
					pod.Status.ContainerStatuses = append(pod.Status.ContainerStatuses, corev1.ContainerStatus{Name: c.Name, Ready: true})
				}
				proxyName := "istio-proxy"
				if mesh == "linkerd" {
					proxyName = "linkerd-proxy"
				}
				for _, ready := range []bool{false, true} {
					current := pod.DeepCopy()
					if native {
						current.Status.InitContainerStatuses = []corev1.ContainerStatus{{Name: proxyName, Ready: ready}}
					} else {
						current.Status.ContainerStatuses[len(current.Status.ContainerStatuses)-1].Ready = ready
					}
					c := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(current).Build()
					r := New(configuration.Configuration{Client: c, Jenkins: j}, jenkinsclient.JenkinsAPIConnectionSettings{})
					result, err := r.waitForJenkins()
					require.NoError(t, err)
					if ready {
						require.Zero(t, result.RequeueAfter)
					} else {
						require.Equal(t, 5*time.Second, result.RequeueAfter)
					}
				}
			})
		}
	}
}

func TestMeshReadinessRequiresProxyStatus(t *testing.T) {
	for _, native := range []bool{false, true} {
		_, pod := meshTestPod("linkerd", native)
		j := comparisonJenkins()
		j.Namespace = "default"
		pod.Name = "jenkins-" + j.Name
		pod.Namespace = j.Namespace
		pod.Status.Phase = corev1.PodRunning
		for _, container := range pod.Spec.Containers {
			if container.Name != "linkerd-proxy" {
				pod.Status.ContainerStatuses = append(pod.Status.ContainerStatuses, corev1.ContainerStatus{Name: container.Name, Ready: true})
			}
		}
		c := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(pod).Build()
		r := New(configuration.Configuration{Client: c, Jenkins: j}, jenkinsclient.JenkinsAPIConnectionSettings{})
		result, err := r.waitForJenkins()
		require.NoError(t, err)
		require.Equal(t, 5*time.Second, result.RequeueAfter)
	}
}

func TestMeshProxyTermination(t *testing.T) {
	for _, native := range []bool{false, true} {
		_, pod := meshTestPod("linkerd", native)
		j := comparisonJenkins()
		j.Namespace = "default"
		pod.Name = "jenkins-" + j.Name
		pod.Namespace = j.Namespace
		pod.Status.Phase = corev1.PodRunning
		for _, container := range pod.Spec.Containers {
			status := corev1.ContainerStatus{Name: container.Name, Ready: true}
			if container.Name == "linkerd-proxy" {
				status.Ready = false
				status.State.Terminated = &corev1.ContainerStateTerminated{ExitCode: 1}
			}
			pod.Status.ContainerStatuses = append(pod.Status.ContainerStatuses, status)
		}
		if native {
			pod.Status.InitContainerStatuses = []corev1.ContainerStatus{{Name: "linkerd-proxy", State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 1}}}}
		}
		c := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(pod).Build()
		notifications := make(chan event.Event, 1)
		r := New(configuration.Configuration{Client: c, Jenkins: j, Notifications: &notifications}, jenkinsclient.JenkinsAPIConnectionSettings{})
		result, err := r.waitForJenkins()
		require.NoError(t, err)
		require.Positive(t, result.RequeueAfter)
		err = c.Get(t.Context(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, &corev1.Pod{})
		if native {
			require.NoError(t, err)
		} else {
			require.True(t, apierrors.IsNotFound(err))
		}
	}
}
