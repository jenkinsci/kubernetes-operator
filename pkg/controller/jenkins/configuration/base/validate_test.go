package base

import (
	"context"
	"fmt"
	"testing"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/constants"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/plugins"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func TestValidatePlugins(t *testing.T) {
	log.SetupLogger(true)
	baseReconcileLoop := New(configuration.Configuration{}, log.Log, client.JenkinsAPIConnectionSettings{}, nil)
	t.Run("empty", func(t *testing.T) {
		var requiredBasePlugins []plugins.Plugin
		var basePlugins []v1alpha2.Plugin
		var userPlugins []v1alpha2.Plugin

		got := baseReconcileLoop.validatePlugins(requiredBasePlugins, basePlugins, userPlugins)

		assert.Nil(t, got)
	})
	t.Run("valid user plugin", func(t *testing.T) {
		var requiredBasePlugins []plugins.Plugin
		var basePlugins []v1alpha2.Plugin
		userPlugins := []v1alpha2.Plugin{{Name: "simple-plugin", Version: "0.0.1"}}

		got := baseReconcileLoop.validatePlugins(requiredBasePlugins, basePlugins, userPlugins)

		assert.Nil(t, got)
	})
	t.Run("invalid user plugin name", func(t *testing.T) {
		var requiredBasePlugins []plugins.Plugin
		var basePlugins []v1alpha2.Plugin
		userPlugins := []v1alpha2.Plugin{{Name: "INVALID?", Version: "0.0.1"}}

		got := baseReconcileLoop.validatePlugins(requiredBasePlugins, basePlugins, userPlugins)

		assert.Equal(t, got, []string{"invalid plugin name 'INVALID?:0.0.1', must follow pattern '" + plugins.NamePattern.String() + "'"})
	})
	t.Run("invalid user plugin version", func(t *testing.T) {
		var requiredBasePlugins []plugins.Plugin
		var basePlugins []v1alpha2.Plugin
		userPlugins := []v1alpha2.Plugin{{Name: "simple-plugin", Version: "invalid!"}}

		got := baseReconcileLoop.validatePlugins(requiredBasePlugins, basePlugins, userPlugins)

		assert.Equal(t, got, []string{"invalid plugin version 'simple-plugin:invalid!', must follow pattern '" + plugins.VersionPattern.String() + "'"})
	})
	t.Run("valid base plugin", func(t *testing.T) {
		var requiredBasePlugins []plugins.Plugin
		basePlugins := []v1alpha2.Plugin{{Name: "simple-plugin", Version: "0.0.1"}}
		var userPlugins []v1alpha2.Plugin

		got := baseReconcileLoop.validatePlugins(requiredBasePlugins, basePlugins, userPlugins)

		assert.Nil(t, got)
	})
	t.Run("invalid base plugin name", func(t *testing.T) {
		var requiredBasePlugins []plugins.Plugin
		basePlugins := []v1alpha2.Plugin{{Name: "INVALID?", Version: "0.0.1"}}
		var userPlugins []v1alpha2.Plugin

		got := baseReconcileLoop.validatePlugins(requiredBasePlugins, basePlugins, userPlugins)

		assert.Equal(t, got, []string{"invalid plugin name 'INVALID?:0.0.1', must follow pattern '" + plugins.NamePattern.String() + "'"})
	})
	t.Run("invalid base plugin version", func(t *testing.T) {
		var requiredBasePlugins []plugins.Plugin
		basePlugins := []v1alpha2.Plugin{{Name: "simple-plugin", Version: "invalid!"}}
		var userPlugins []v1alpha2.Plugin

		got := baseReconcileLoop.validatePlugins(requiredBasePlugins, basePlugins, userPlugins)

		assert.Equal(t, got, []string{"invalid plugin version 'simple-plugin:invalid!', must follow pattern '" + plugins.VersionPattern.String() + "'"})
	})
	t.Run("valid user and base plugin version", func(t *testing.T) {
		var requiredBasePlugins []plugins.Plugin
		basePlugins := []v1alpha2.Plugin{{Name: "simple-plugin", Version: "0.0.1"}}
		userPlugins := []v1alpha2.Plugin{{Name: "simple-plugin", Version: "0.0.1"}}

		got := baseReconcileLoop.validatePlugins(requiredBasePlugins, basePlugins, userPlugins)

		assert.Nil(t, got)
	})
	t.Run("invalid user and base plugin version", func(t *testing.T) {
		var requiredBasePlugins []plugins.Plugin
		basePlugins := []v1alpha2.Plugin{{Name: "simple-plugin", Version: "0.0.1"}}
		userPlugins := []v1alpha2.Plugin{{Name: "simple-plugin", Version: "0.0.2"}}

		got := baseReconcileLoop.validatePlugins(requiredBasePlugins, basePlugins, userPlugins)

		assert.Contains(t, got, "Plugin 'simple-plugin:0.0.1' requires version '0.0.1' but plugin 'simple-plugin:0.0.2' requires '0.0.2' for plugin 'simple-plugin'")
		assert.Contains(t, got, "Plugin 'simple-plugin:0.0.2' requires version '0.0.2' but plugin 'simple-plugin:0.0.1' requires '0.0.1' for plugin 'simple-plugin'")
	})
	t.Run("required base plugin set with the same version", func(t *testing.T) {
		requiredBasePlugins := []plugins.Plugin{{Name: "simple-plugin", Version: "0.0.1"}}
		basePlugins := []v1alpha2.Plugin{{Name: "simple-plugin", Version: "0.0.1"}}
		var userPlugins []v1alpha2.Plugin

		got := baseReconcileLoop.validatePlugins(requiredBasePlugins, basePlugins, userPlugins)

		assert.Nil(t, got)
	})
	t.Run("required base plugin set with different version", func(t *testing.T) {
		requiredBasePlugins := []plugins.Plugin{{Name: "simple-plugin", Version: "0.0.1"}}
		basePlugins := []v1alpha2.Plugin{{Name: "simple-plugin", Version: "0.0.2"}}
		var userPlugins []v1alpha2.Plugin

		got := baseReconcileLoop.validatePlugins(requiredBasePlugins, basePlugins, userPlugins)

		assert.Nil(t, got)
	})
	t.Run("missing required base plugin", func(t *testing.T) {
		requiredBasePlugins := []plugins.Plugin{{Name: "simple-plugin", Version: "0.0.1"}}
		var basePlugins []v1alpha2.Plugin
		var userPlugins []v1alpha2.Plugin

		got := baseReconcileLoop.validatePlugins(requiredBasePlugins, basePlugins, userPlugins)

		assert.Equal(t, got, []string{"Missing plugin 'simple-plugin' in spec.master.basePlugins"})
	})
}

func TestReconcileJenkinsBaseConfiguration_validateImagePullSecrets(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-ref",
			},
			Data: map[string][]byte{
				"docker-server":   []byte("test_server"),
				"docker-username": []byte("test_user"),
				"docker-password": []byte("test_password"),
				"docker-email":    []byte("test_email"),
			},
		}

		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				Master: v1alpha2.JenkinsMaster{
					ImagePullSecrets: []corev1.LocalObjectReference{
						{Name: secret.ObjectMeta.Name},
					},
				},
			},
		}

		fakeClient := fake.NewFakeClient()
		err := fakeClient.Create(context.TODO(), secret)
		assert.NoError(t, err)

		baseReconcileLoop := New(configuration.Configuration{
			Client:  fakeClient,
			Jenkins: &jenkins,
		}, logf.ZapLogger(false), client.JenkinsAPIConnectionSettings{}, nil)

		got, err := baseReconcileLoop.validateImagePullSecrets()
		fmt.Println(got)
		assert.Nil(t, got)
		assert.NoError(t, err)
	})

	t.Run("no secret", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				Master: v1alpha2.JenkinsMaster{
					ImagePullSecrets: []corev1.LocalObjectReference{
						{Name: "test-ref"},
					},
				},
			},
		}

		fakeClient := fake.NewFakeClient()

		baseReconcileLoop := New(configuration.Configuration{
			Client:  fakeClient,
			Jenkins: &jenkins,
		}, nil, client.JenkinsAPIConnectionSettings{}, nil)

		got, _ := baseReconcileLoop.validateImagePullSecrets()

		assert.Equal(t, got, []string{"Secret test-ref not found defined in spec.master.imagePullSecrets", "Secret 'test-ref' defined in spec.master.imagePullSecrets doesn't have 'docker-server' key.", "Secret 'test-ref' defined in spec.master.imagePullSecrets doesn't have 'docker-username' key.", "Secret 'test-ref' defined in spec.master.imagePullSecrets doesn't have 'docker-password' key.", "Secret 'test-ref' defined in spec.master.imagePullSecrets doesn't have 'docker-email' key."})
	})

	t.Run("no docker email", func(t *testing.T) {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-ref",
			},
			Data: map[string][]byte{
				"docker-server":   []byte("test_server"),
				"docker-username": []byte("test_user"),
				"docker-password": []byte("test_password"),
			},
		}

		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				Master: v1alpha2.JenkinsMaster{
					ImagePullSecrets: []corev1.LocalObjectReference{
						{Name: secret.ObjectMeta.Name},
					},
				},
			},
		}

		fakeClient := fake.NewFakeClient()
		err := fakeClient.Create(context.TODO(), secret)
		assert.NoError(t, err)

		baseReconcileLoop := New(configuration.Configuration{
			Client:  fakeClient,
			Jenkins: &jenkins,
		}, nil, client.JenkinsAPIConnectionSettings{}, nil)

		got, _ := baseReconcileLoop.validateImagePullSecrets()

		assert.Equal(t, got, []string{"Secret 'test-ref' defined in spec.master.imagePullSecrets doesn't have 'docker-email' key."})
	})

	t.Run("no docker password", func(t *testing.T) {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-ref",
			},
			Data: map[string][]byte{
				"docker-server":   []byte("test_server"),
				"docker-username": []byte("test_user"),
				"docker-email":    []byte("test_email"),
			},
		}

		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				Master: v1alpha2.JenkinsMaster{
					ImagePullSecrets: []corev1.LocalObjectReference{
						{Name: secret.ObjectMeta.Name},
					},
				},
			},
		}

		fakeClient := fake.NewFakeClient()
		err := fakeClient.Create(context.TODO(), secret)
		assert.NoError(t, err)

		baseReconcileLoop := New(configuration.Configuration{
			Client:  fakeClient,
			Jenkins: &jenkins,
		}, logf.ZapLogger(false), client.JenkinsAPIConnectionSettings{}, nil)

		got, _ := baseReconcileLoop.validateImagePullSecrets()

		assert.Equal(t, got, []string{"Secret 'test-ref' defined in spec.master.imagePullSecrets doesn't have 'docker-password' key."})
	})

	t.Run("no docker username", func(t *testing.T) {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-ref",
			},
			Data: map[string][]byte{
				"docker-server":   []byte("test_server"),
				"docker-password": []byte("test_password"),
				"docker-email":    []byte("test_email"),
			},
		}

		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				Master: v1alpha2.JenkinsMaster{
					ImagePullSecrets: []corev1.LocalObjectReference{
						{Name: secret.ObjectMeta.Name},
					},
				},
			},
		}

		fakeClient := fake.NewFakeClient()
		err := fakeClient.Create(context.TODO(), secret)
		assert.NoError(t, err)

		baseReconcileLoop := New(configuration.Configuration{
			Client:  fakeClient,
			Jenkins: &jenkins,
		}, logf.ZapLogger(false), client.JenkinsAPIConnectionSettings{}, nil)

		got, _ := baseReconcileLoop.validateImagePullSecrets()

		assert.Equal(t, got, []string{"Secret 'test-ref' defined in spec.master.imagePullSecrets doesn't have 'docker-username' key."})
	})

	t.Run("no docker server", func(t *testing.T) {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-ref",
			},
			Data: map[string][]byte{
				"docker-username": []byte("test_user"),
				"docker-password": []byte("test_password"),
				"docker-email":    []byte("test_email"),
			},
		}

		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				Master: v1alpha2.JenkinsMaster{
					ImagePullSecrets: []corev1.LocalObjectReference{
						{Name: secret.ObjectMeta.Name},
					},
				},
			},
		}

		fakeClient := fake.NewFakeClient()
		err := fakeClient.Create(context.TODO(), secret)
		assert.NoError(t, err)

		baseReconcileLoop := New(configuration.Configuration{
			Client:  fakeClient,
			Jenkins: &jenkins,
		}, logf.ZapLogger(false), client.JenkinsAPIConnectionSettings{}, nil)

		got, _ := baseReconcileLoop.validateImagePullSecrets()

		assert.Equal(t, got, []string{"Secret 'test-ref' defined in spec.master.imagePullSecrets doesn't have 'docker-server' key."})
	})
}

func TestValidateJenkinsMasterPodEnvs(t *testing.T) {
	validJenkinsOps := "-Djenkins.install.runSetupWizard=false -Djava.awt.headless=true"
	t.Run("happy", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				Master: v1alpha2.JenkinsMaster{
					Containers: []v1alpha2.Container{
						{
							Env: []v1.EnvVar{
								{
									Name:  "SOME_VALUE",
									Value: "",
								},
								{
									Name:  constants.JavaOpsVariableName,
									Value: validJenkinsOps,
								},
							},
						},
					},
				},
			},
		}
		baseReconcileLoop := New(configuration.Configuration{
			Jenkins: &jenkins,
		}, logf.ZapLogger(false), client.JenkinsAPIConnectionSettings{}, nil)
		got := baseReconcileLoop.validateJenkinsMasterPodEnvs()
		assert.Nil(t, got)
	})
	t.Run("missing -Djava.awt.headless=true in JAVA_OPTS env", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				Master: v1alpha2.JenkinsMaster{
					Containers: []v1alpha2.Container{
						{
							Env: []v1.EnvVar{
								{
									Name:  constants.JavaOpsVariableName,
									Value: "-Djenkins.install.runSetupWizard=false",
								},
							},
						},
					},
				},
			},
		}
		baseReconcileLoop := New(configuration.Configuration{
			Jenkins: &jenkins,
		}, logf.ZapLogger(false), client.JenkinsAPIConnectionSettings{}, nil)
		got := baseReconcileLoop.validateJenkinsMasterPodEnvs()

		assert.Equal(t, got, []string{"Jenkins Master container env 'JAVA_OPTS' doesn't have required flag '-Djava.awt.headless=true'"})
	})
	t.Run("missing -Djenkins.install.runSetupWizard=false in JAVA_OPTS env", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				Master: v1alpha2.JenkinsMaster{
					Containers: []v1alpha2.Container{
						{
							Env: []v1.EnvVar{
								{
									Name:  constants.JavaOpsVariableName,
									Value: "-Djava.awt.headless=true",
								},
							},
						},
					},
				},
			},
		}
		baseReconcileLoop := New(configuration.Configuration{
			Jenkins: &jenkins,
		}, logf.ZapLogger(false), client.JenkinsAPIConnectionSettings{}, nil)
		got := baseReconcileLoop.validateJenkinsMasterPodEnvs()

		assert.Equal(t, got, []string{"Jenkins Master container env 'JAVA_OPTS' doesn't have required flag '-Djenkins.install.runSetupWizard=false'"})
	})
}

func TestValidateReservedVolumes(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				Master: v1alpha2.JenkinsMaster{
					Volumes: []v1.Volume{
						{
							Name: "not-used-name",
						},
					},
				},
			},
		}
		baseReconcileLoop := New(configuration.Configuration{
			Jenkins: &jenkins,
		}, logf.ZapLogger(false), client.JenkinsAPIConnectionSettings{}, nil)
		got := baseReconcileLoop.validateReservedVolumes()
		assert.Nil(t, got)
	})
	t.Run("used reserved name", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				Master: v1alpha2.JenkinsMaster{
					Volumes: []v1.Volume{
						{
							Name: resources.JenkinsHomeVolumeName,
						},
					},
				},
			},
		}
		baseReconcileLoop := New(configuration.Configuration{
			Jenkins: &jenkins,
		}, logf.ZapLogger(false), client.JenkinsAPIConnectionSettings{}, nil)
		got := baseReconcileLoop.validateReservedVolumes()

		assert.Equal(t, got, []string{"Jenkins Master pod volume 'jenkins-home' is reserved please choose different one"})
	})
}

func TestValidateContainerVolumeMounts(t *testing.T) {
	t.Run("default Jenkins master container", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				Master: v1alpha2.JenkinsMaster{},
			},
		}
		baseReconcileLoop := New(configuration.Configuration{
			Jenkins: &jenkins,
		}, logf.ZapLogger(false), client.JenkinsAPIConnectionSettings{}, nil)
		got := baseReconcileLoop.validateContainerVolumeMounts(v1alpha2.Container{})
		assert.Nil(t, got)
	})
	t.Run("one extra volume", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				Master: v1alpha2.JenkinsMaster{
					Volumes: []v1.Volume{
						{
							Name: "example",
						},
					},
					Containers: []v1alpha2.Container{
						{
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "example",
									MountPath: "/test",
								},
							},
						},
					},
				},
			},
		}
		baseReconcileLoop := New(configuration.Configuration{
			Jenkins: &jenkins,
		}, logf.ZapLogger(false), client.JenkinsAPIConnectionSettings{}, nil)
		got := baseReconcileLoop.validateContainerVolumeMounts(jenkins.Spec.Master.Containers[0])
		assert.Nil(t, got)
	})
	t.Run("empty mountPath", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				Master: v1alpha2.JenkinsMaster{
					Volumes: []v1.Volume{
						{
							Name: "example",
						},
					},
					Containers: []v1alpha2.Container{
						{
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "example",
									MountPath: "", // empty
								},
							},
						},
					},
				},
			},
		}
		baseReconcileLoop := New(configuration.Configuration{
			Jenkins: &jenkins,
		}, logf.ZapLogger(false), client.JenkinsAPIConnectionSettings{}, nil)
		got := baseReconcileLoop.validateContainerVolumeMounts(jenkins.Spec.Master.Containers[0])
		assert.Equal(t, got, []string{"mountPath not set for 'example' volume mount in container ''"})
	})
	t.Run("missing volume", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				Master: v1alpha2.JenkinsMaster{
					Containers: []v1alpha2.Container{
						{
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "missing-volume",
									MountPath: "/test",
								},
							},
						},
					},
				},
			},
		}
		baseReconcileLoop := New(configuration.Configuration{
			Jenkins: &jenkins,
		}, logf.ZapLogger(false), client.JenkinsAPIConnectionSettings{}, nil)
		got := baseReconcileLoop.validateContainerVolumeMounts(jenkins.Spec.Master.Containers[0])

		assert.Equal(t, got, []string{"Not found volume for 'missing-volume' volume mount in container ''"})
	})
}

func TestValidateResourceQuota(t *testing.T) {
	t.Run("happy - not set", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				Master: v1alpha2.JenkinsMaster{
					Containers: []v1alpha2.Container{
						{
							Resources: v1.ResourceRequirements{},
						},
					},
				},
			},
		}

		baseReconcileLoop := New(configuration.Configuration{
			Jenkins: &jenkins,
		}, logf.ZapLogger(false), client.JenkinsAPIConnectionSettings{}, nil)
		got := baseReconcileLoop.validateResourceQuota()

		assert.Nil(t, got)
	})
	t.Run("happy - set minimal limits", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				Master: v1alpha2.JenkinsMaster{
					Containers: []v1alpha2.Container{
						{
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									v1.ResourceCPU:    resource.MustParse("1000m"),
									v1.ResourceMemory: resource.MustParse("512Mi"),
								},
							},
						},
					},
				},
			},
		}

		baseReconcileLoop := New(configuration.Configuration{
			Jenkins: &jenkins,
		}, logf.ZapLogger(false), client.JenkinsAPIConnectionSettings{}, nil)
		got := baseReconcileLoop.validateResourceQuota()

		assert.Nil(t, got)
	})
	t.Run("too low cpu milliicores", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				Master: v1alpha2.JenkinsMaster{
					Containers: []v1alpha2.Container{
						{
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									v1.ResourceCPU:    resource.MustParse("999m"),
									v1.ResourceMemory: resource.MustParse("512Mi"),
								},
							},
						},
					},
				},
			},
		}

		baseReconcileLoop := New(configuration.Configuration{
			Jenkins: &jenkins,
		}, logf.ZapLogger(false), client.JenkinsAPIConnectionSettings{}, nil)
		got := baseReconcileLoop.validateResourceQuota()

		assert.NotNil(t, got)
	})

	t.Run("too low memory mebibytes", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				Master: v1alpha2.JenkinsMaster{
					Containers: []v1alpha2.Container{
						{
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									v1.ResourceCPU:    resource.MustParse("1000m"),
									v1.ResourceMemory: resource.MustParse("511Mi"),
								},
							},
						},
					},
				},
			},
		}

		baseReconcileLoop := New(configuration.Configuration{
			Jenkins: &jenkins,
		}, logf.ZapLogger(false), client.JenkinsAPIConnectionSettings{}, nil)
		got := baseReconcileLoop.validateResourceQuota()

		assert.NotNil(t, got)
	})

	t.Run("too low memory mebibytes and cpu millicores", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				Master: v1alpha2.JenkinsMaster{
					Containers: []v1alpha2.Container{
						{
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									v1.ResourceCPU:    resource.MustParse("999m"),
									v1.ResourceMemory: resource.MustParse("511Mi"),
								},
							},
						},
					},
				},
			},
		}

		baseReconcileLoop := New(configuration.Configuration{
			Jenkins: &jenkins,
		}, logf.ZapLogger(false), client.JenkinsAPIConnectionSettings{}, nil)
		got := baseReconcileLoop.validateResourceQuota()

		assert.NotNil(t, got)
	})
}

func TestValidateConfigMapVolume(t *testing.T) {
	namespace := "default"
	t.Run("optional", func(t *testing.T) {
		optional := true
		volume := corev1.Volume{
			Name: "name",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					Optional: &optional,
				},
			},
		}
		fakeClient := fake.NewFakeClient()
		baseReconcileLoop := New(configuration.Configuration{
			Client: fakeClient,
		}, logf.ZapLogger(false), client.JenkinsAPIConnectionSettings{}, nil)

		got, err := baseReconcileLoop.validateConfigMapVolume(volume)

		assert.NoError(t, err)
		assert.Nil(t, got)
	})
	t.Run("happy, required", func(t *testing.T) {
		optional := false
		configMap := corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: "configmap-name"}}
		jenkins := &v1alpha2.Jenkins{ObjectMeta: metav1.ObjectMeta{Namespace: namespace}}
		volume := corev1.Volume{
			Name: "volume-name",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					Optional: &optional,
					LocalObjectReference: corev1.LocalObjectReference{
						Name: configMap.Name,
					},
				},
			},
		}
		fakeClient := fake.NewFakeClient()
		err := fakeClient.Create(context.TODO(), &configMap)
		assert.NoError(t, err)
		baseReconcileLoop := New(configuration.Configuration{
			Client:  fakeClient,
			Jenkins: jenkins,
		}, logf.ZapLogger(false), client.JenkinsAPIConnectionSettings{}, nil)

		got, err := baseReconcileLoop.validateConfigMapVolume(volume)

		assert.NoError(t, err)
		assert.Nil(t, got)
	})
	t.Run("missing configmap", func(t *testing.T) {
		optional := false
		configMap := corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: "configmap-name"}}
		jenkins := &v1alpha2.Jenkins{ObjectMeta: metav1.ObjectMeta{Namespace: namespace}}
		volume := corev1.Volume{
			Name: "volume-name",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					Optional: &optional,
					LocalObjectReference: corev1.LocalObjectReference{
						Name: configMap.Name,
					},
				},
			},
		}
		fakeClient := fake.NewFakeClient()
		baseReconcileLoop := New(configuration.Configuration{
			Client:  fakeClient,
			Jenkins: jenkins,
		}, logf.ZapLogger(false), client.JenkinsAPIConnectionSettings{}, nil)

		got, err := baseReconcileLoop.validateConfigMapVolume(volume)

		assert.NoError(t, err)

		assert.Equal(t, got, []string{"ConfigMap 'configmap-name' not found for volume '{volume-name {nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil &ConfigMapVolumeSource{LocalObjectReference:LocalObjectReference{Name:configmap-name,},Items:[],DefaultMode:nil,Optional:*false,} nil nil nil nil nil nil nil nil}}'"})
	})
}

func TestValidateSecretVolume(t *testing.T) {
	namespace := "default"
	t.Run("optional", func(t *testing.T) {
		optional := true
		volume := corev1.Volume{
			Name: "name",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					Optional: &optional,
				},
			},
		}
		fakeClient := fake.NewFakeClient()
		baseReconcileLoop := New(configuration.Configuration{
			Client: fakeClient,
		}, logf.ZapLogger(false), client.JenkinsAPIConnectionSettings{}, nil)

		got, err := baseReconcileLoop.validateSecretVolume(volume)

		assert.NoError(t, err)
		assert.Nil(t, got)
	})
	t.Run("happy, required", func(t *testing.T) {
		optional := false
		secret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: "secret-name"}}
		jenkins := &v1alpha2.Jenkins{ObjectMeta: metav1.ObjectMeta{Namespace: namespace}}
		volume := corev1.Volume{
			Name: "volume-name",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					Optional:   &optional,
					SecretName: secret.Name,
				},
			},
		}
		fakeClient := fake.NewFakeClient()
		err := fakeClient.Create(context.TODO(), &secret)
		assert.NoError(t, err)
		baseReconcileLoop := New(configuration.Configuration{
			Client:  fakeClient,
			Jenkins: jenkins,
		}, logf.ZapLogger(false), client.JenkinsAPIConnectionSettings{}, nil)

		got, err := baseReconcileLoop.validateSecretVolume(volume)

		assert.NoError(t, err)
		assert.Nil(t, got)
	})
	t.Run("missing secret", func(t *testing.T) {
		optional := false
		secret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: "secret-name"}}
		jenkins := &v1alpha2.Jenkins{ObjectMeta: metav1.ObjectMeta{Namespace: namespace}}
		volume := corev1.Volume{
			Name: "volume-name",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					Optional:   &optional,
					SecretName: secret.Name,
				},
			},
		}
		fakeClient := fake.NewFakeClient()
		baseReconcileLoop := New(configuration.Configuration{
			Client:  fakeClient,
			Jenkins: jenkins,
		}, logf.ZapLogger(false), client.JenkinsAPIConnectionSettings{}, nil)
		got, err := baseReconcileLoop.validateSecretVolume(volume)

		assert.NoError(t, err)

		assert.Equal(t, got, []string{"Secret 'secret-name' not found for volume '{volume-name {nil nil nil nil nil &SecretVolumeSource{SecretName:secret-name,Items:[],DefaultMode:nil,Optional:*false,} nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil}}'"})
	})
}

func TestValidateCustomization(t *testing.T) {
	namespace := "default"
	secretName := "secretName"
	configMapName := "configmap-name"
	jenkins := &v1alpha2.Jenkins{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
	}
	t.Run("empty", func(t *testing.T) {
		customization := v1alpha2.Customization{}
		fakeClient := fake.NewFakeClient()
		baseReconcileLoop := New(configuration.Configuration{
			Jenkins: jenkins,
			Client:  fakeClient,
		}, logf.ZapLogger(false), client.JenkinsAPIConnectionSettings{}, nil)

		got, err := baseReconcileLoop.validateCustomization(customization, "spec.groovyScripts")

		assert.NoError(t, err)
		assert.Nil(t, got)
	})
	t.Run("secret set but configurations is empty", func(t *testing.T) {
		customization := v1alpha2.Customization{
			Secret:         v1alpha2.SecretRef{Name: secretName},
			Configurations: []v1alpha2.ConfigMapRef{},
		}
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: namespace,
			},
		}
		fakeClient := fake.NewFakeClient()
		baseReconcileLoop := New(configuration.Configuration{
			Jenkins: jenkins,
			Client:  fakeClient,
		}, logf.ZapLogger(false), client.JenkinsAPIConnectionSettings{}, nil)
		err := fakeClient.Create(context.TODO(), secret)
		require.NoError(t, err)

		got, err := baseReconcileLoop.validateCustomization(customization, "spec.groovyScripts")

		assert.NoError(t, err)

		assert.Equal(t, got, []string{"spec.groovyScripts.secret.name is set but spec.groovyScripts.configurations is empty"})
	})
	t.Run("secret and configmap exists", func(t *testing.T) {
		customization := v1alpha2.Customization{
			Secret:         v1alpha2.SecretRef{Name: secretName},
			Configurations: []v1alpha2.ConfigMapRef{{Name: configMapName}},
		}
		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      configMapName,
				Namespace: namespace,
			},
		}
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: namespace,
			},
		}
		fakeClient := fake.NewFakeClient()
		baseReconcileLoop := New(configuration.Configuration{
			Jenkins: jenkins,
			Client:  fakeClient,
		}, logf.ZapLogger(false), client.JenkinsAPIConnectionSettings{}, nil)
		err := fakeClient.Create(context.TODO(), secret)
		require.NoError(t, err)
		err = fakeClient.Create(context.TODO(), configMap)
		require.NoError(t, err)

		got, err := baseReconcileLoop.validateCustomization(customization, "spec.groovyScripts")

		assert.NoError(t, err)
		assert.Nil(t, got)
	})
	t.Run("secret not exists and configmap exists", func(t *testing.T) {
		configMapName := "configmap-name"
		customization := v1alpha2.Customization{
			Secret:         v1alpha2.SecretRef{Name: secretName},
			Configurations: []v1alpha2.ConfigMapRef{{Name: configMapName}},
		}
		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      configMapName,
				Namespace: namespace,
			},
		}
		fakeClient := fake.NewFakeClient()
		baseReconcileLoop := New(configuration.Configuration{
			Jenkins: jenkins,
			Client:  fakeClient,
		}, logf.ZapLogger(false), client.JenkinsAPIConnectionSettings{}, nil)
		err := fakeClient.Create(context.TODO(), configMap)
		require.NoError(t, err)

		got, err := baseReconcileLoop.validateCustomization(customization, "spec.groovyScripts")

		assert.NoError(t, err)

		assert.Equal(t, got, []string{"Secret 'secretName' configured in spec.groovyScripts.secret.name not found"})
	})
	t.Run("secret exists and configmap not exists", func(t *testing.T) {
		customization := v1alpha2.Customization{
			Secret:         v1alpha2.SecretRef{Name: secretName},
			Configurations: []v1alpha2.ConfigMapRef{{Name: configMapName}},
		}
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: namespace,
			},
		}
		fakeClient := fake.NewFakeClient()
		baseReconcileLoop := New(configuration.Configuration{
			Jenkins: jenkins,
			Client:  fakeClient,
		}, logf.ZapLogger(false), client.JenkinsAPIConnectionSettings{}, nil)
		err := fakeClient.Create(context.TODO(), secret)
		require.NoError(t, err)

		got, err := baseReconcileLoop.validateCustomization(customization, "spec.groovyScripts")

		assert.NoError(t, err)

		assert.Equal(t, got, []string{"ConfigMap 'configmap-name' configured in spec.groovyScripts.configurations[0] not found"})
	})
}
