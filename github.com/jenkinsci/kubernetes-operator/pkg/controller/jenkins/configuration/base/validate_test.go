package base

import (
	"context"
	"testing"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/plugins"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func TestValidatePlugins(t *testing.T) {
	log.SetupLogger(true)
	baseReconcileLoop := New(nil, nil, log.Log,
		nil, false, false, nil, nil)
	t.Run("empty", func(t *testing.T) {
		var requiredBasePlugins []plugins.Plugin
		var basePlugins []v1alpha2.Plugin
		var userPlugins []v1alpha2.Plugin

		got := baseReconcileLoop.validatePlugins(requiredBasePlugins, basePlugins, userPlugins)

		assert.True(t, got)
	})
	t.Run("valid user plugin", func(t *testing.T) {
		var requiredBasePlugins []plugins.Plugin
		var basePlugins []v1alpha2.Plugin
		userPlugins := []v1alpha2.Plugin{{Name: "simple-plugin", Version: "0.0.1"}}

		got := baseReconcileLoop.validatePlugins(requiredBasePlugins, basePlugins, userPlugins)

		assert.True(t, got)
	})
	t.Run("invalid user plugin name", func(t *testing.T) {
		var requiredBasePlugins []plugins.Plugin
		var basePlugins []v1alpha2.Plugin
		userPlugins := []v1alpha2.Plugin{{Name: "INVALID", Version: "0.0.1"}}

		got := baseReconcileLoop.validatePlugins(requiredBasePlugins, basePlugins, userPlugins)

		assert.False(t, got)
	})
	t.Run("invalid user plugin version", func(t *testing.T) {
		var requiredBasePlugins []plugins.Plugin
		var basePlugins []v1alpha2.Plugin
		userPlugins := []v1alpha2.Plugin{{Name: "simple-plugin", Version: "invalid"}}

		got := baseReconcileLoop.validatePlugins(requiredBasePlugins, basePlugins, userPlugins)

		assert.False(t, got)
	})
	t.Run("valid base plugin", func(t *testing.T) {
		var requiredBasePlugins []plugins.Plugin
		basePlugins := []v1alpha2.Plugin{{Name: "simple-plugin", Version: "0.0.1"}}
		var userPlugins []v1alpha2.Plugin

		got := baseReconcileLoop.validatePlugins(requiredBasePlugins, basePlugins, userPlugins)

		assert.True(t, got)
	})
	t.Run("invalid base plugin name", func(t *testing.T) {
		var requiredBasePlugins []plugins.Plugin
		basePlugins := []v1alpha2.Plugin{{Name: "INVALID", Version: "0.0.1"}}
		var userPlugins []v1alpha2.Plugin

		got := baseReconcileLoop.validatePlugins(requiredBasePlugins, basePlugins, userPlugins)

		assert.False(t, got)
	})
	t.Run("invalid base plugin version", func(t *testing.T) {
		var requiredBasePlugins []plugins.Plugin
		basePlugins := []v1alpha2.Plugin{{Name: "simple-plugin", Version: "invalid"}}
		var userPlugins []v1alpha2.Plugin

		got := baseReconcileLoop.validatePlugins(requiredBasePlugins, basePlugins, userPlugins)

		assert.False(t, got)
	})
	t.Run("valid user and base plugin version", func(t *testing.T) {
		var requiredBasePlugins []plugins.Plugin
		basePlugins := []v1alpha2.Plugin{{Name: "simple-plugin", Version: "0.0.1"}}
		userPlugins := []v1alpha2.Plugin{{Name: "simple-plugin", Version: "0.0.1"}}

		got := baseReconcileLoop.validatePlugins(requiredBasePlugins, basePlugins, userPlugins)

		assert.True(t, got)
	})
	t.Run("invalid user and base plugin version", func(t *testing.T) {
		var requiredBasePlugins []plugins.Plugin
		basePlugins := []v1alpha2.Plugin{{Name: "simple-plugin", Version: "0.0.1"}}
		userPlugins := []v1alpha2.Plugin{{Name: "simple-plugin", Version: "0.0.2"}}

		got := baseReconcileLoop.validatePlugins(requiredBasePlugins, basePlugins, userPlugins)

		assert.False(t, got)
	})
	t.Run("required base plugin set with the same version", func(t *testing.T) {
		requiredBasePlugins := []plugins.Plugin{{Name: "simple-plugin", Version: "0.0.1"}}
		basePlugins := []v1alpha2.Plugin{{Name: "simple-plugin", Version: "0.0.1"}}
		var userPlugins []v1alpha2.Plugin

		got := baseReconcileLoop.validatePlugins(requiredBasePlugins, basePlugins, userPlugins)

		assert.True(t, got)
	})
	t.Run("required base plugin set with different version", func(t *testing.T) {
		requiredBasePlugins := []plugins.Plugin{{Name: "simple-plugin", Version: "0.0.1"}}
		basePlugins := []v1alpha2.Plugin{{Name: "simple-plugin", Version: "0.0.2"}}
		var userPlugins []v1alpha2.Plugin

		got := baseReconcileLoop.validatePlugins(requiredBasePlugins, basePlugins, userPlugins)

		assert.True(t, got)
	})
	t.Run("missign required base plugin", func(t *testing.T) {
		requiredBasePlugins := []plugins.Plugin{{Name: "simple-plugin", Version: "0.0.1"}}
		var basePlugins []v1alpha2.Plugin
		var userPlugins []v1alpha2.Plugin

		got := baseReconcileLoop.validatePlugins(requiredBasePlugins, basePlugins, userPlugins)

		assert.False(t, got)
	})
}

func TestValidateJenkinsMasterPodEnvs(t *testing.T) {
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
							},
						},
					},
				},
			},
		}
		baseReconcileLoop := New(nil, nil, logf.ZapLogger(false),
			&jenkins, false, false, nil, nil)
		got := baseReconcileLoop.validateJenkinsMasterPodEnvs()
		assert.Equal(t, true, got)
	})
	t.Run("override JENKINS_HOME env", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				Master: v1alpha2.JenkinsMaster{
					Containers: []v1alpha2.Container{
						{
							Env: []v1.EnvVar{
								{
									Name:  "JENKINS_HOME",
									Value: "",
								},
							},
						},
					},
				},
			},
		}
		baseReconcileLoop := New(nil, nil, logf.ZapLogger(false),
			&jenkins, false, false, nil, nil)
		got := baseReconcileLoop.validateJenkinsMasterPodEnvs()
		assert.Equal(t, false, got)
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
		baseReconcileLoop := New(nil, nil, logf.ZapLogger(false),
			&jenkins, false, false, nil, nil)
		got := baseReconcileLoop.validateReservedVolumes()
		assert.Equal(t, true, got)
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
		baseReconcileLoop := New(nil, nil, logf.ZapLogger(false),
			&jenkins, false, false, nil, nil)
		got := baseReconcileLoop.validateReservedVolumes()
		assert.Equal(t, false, got)
	})
}

func TestValidateContainerVolumeMounts(t *testing.T) {
	t.Run("default Jenkins master container", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				Master: v1alpha2.JenkinsMaster{},
			},
		}
		baseReconcileLoop := New(nil, nil, logf.ZapLogger(false),
			&jenkins, false, false, nil, nil)
		got := baseReconcileLoop.validateContainerVolumeMounts(v1alpha2.Container{})
		assert.Equal(t, true, got)
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
		baseReconcileLoop := New(nil, nil, logf.ZapLogger(false),
			&jenkins, false, false, nil, nil)
		got := baseReconcileLoop.validateContainerVolumeMounts(jenkins.Spec.Master.Containers[0])
		assert.Equal(t, true, got)
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
		baseReconcileLoop := New(nil, nil, logf.ZapLogger(false),
			&jenkins, false, false, nil, nil)
		got := baseReconcileLoop.validateContainerVolumeMounts(jenkins.Spec.Master.Containers[0])
		assert.Equal(t, false, got)
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
		baseReconcileLoop := New(nil, nil, logf.ZapLogger(false),
			&jenkins, false, false, nil, nil)
		got := baseReconcileLoop.validateContainerVolumeMounts(jenkins.Spec.Master.Containers[0])
		assert.Equal(t, false, got)
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
		baseReconcileLoop := New(fakeClient, nil, logf.ZapLogger(false),
			nil, false, false, nil, nil)

		got, err := baseReconcileLoop.validateConfigMapVolume(volume)

		assert.NoError(t, err)
		assert.True(t, got)
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
		baseReconcileLoop := New(fakeClient, nil, logf.ZapLogger(false),
			jenkins, false, false, nil, nil)

		got, err := baseReconcileLoop.validateConfigMapVolume(volume)

		assert.NoError(t, err)
		assert.True(t, got)
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
		baseReconcileLoop := New(fakeClient, nil, logf.ZapLogger(false),
			jenkins, false, false, nil, nil)

		got, err := baseReconcileLoop.validateConfigMapVolume(volume)

		assert.NoError(t, err)
		assert.False(t, got)
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
		baseReconcileLoop := New(fakeClient, nil, logf.ZapLogger(false),
			nil, false, false, nil, nil)

		got, err := baseReconcileLoop.validateSecretVolume(volume)

		assert.NoError(t, err)
		assert.True(t, got)
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
		baseReconcileLoop := New(fakeClient, nil, logf.ZapLogger(false),
			jenkins, false, false, nil, nil)

		got, err := baseReconcileLoop.validateSecretVolume(volume)

		assert.NoError(t, err)
		assert.True(t, got)
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
		baseReconcileLoop := New(fakeClient, nil, logf.ZapLogger(false),
			jenkins, false, false, nil, nil)

		got, err := baseReconcileLoop.validateSecretVolume(volume)

		assert.NoError(t, err)
		assert.False(t, got)
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
		baseReconcileLoop := New(fakeClient, nil, logf.ZapLogger(false),
			jenkins, false, false, nil, nil)

		got, err := baseReconcileLoop.validateCustomization(customization, "spec.groovyScripts")

		assert.NoError(t, err)
		assert.True(t, got)
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
		baseReconcileLoop := New(fakeClient, nil, logf.ZapLogger(false),
			jenkins, false, false, nil, nil)
		err := fakeClient.Create(context.TODO(), secret)
		require.NoError(t, err)

		got, err := baseReconcileLoop.validateCustomization(customization, "spec.groovyScripts")

		assert.NoError(t, err)
		assert.False(t, got)
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
		baseReconcileLoop := New(fakeClient, nil, logf.ZapLogger(false),
			jenkins, false, false, nil, nil)
		err := fakeClient.Create(context.TODO(), secret)
		require.NoError(t, err)
		err = fakeClient.Create(context.TODO(), configMap)
		require.NoError(t, err)

		got, err := baseReconcileLoop.validateCustomization(customization, "spec.groovyScripts")

		assert.NoError(t, err)
		assert.True(t, got)
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
		baseReconcileLoop := New(fakeClient, nil, logf.ZapLogger(false),
			jenkins, false, false, nil, nil)
		err := fakeClient.Create(context.TODO(), configMap)
		require.NoError(t, err)

		got, err := baseReconcileLoop.validateCustomization(customization, "spec.groovyScripts")

		assert.NoError(t, err)
		assert.False(t, got)
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
		baseReconcileLoop := New(fakeClient, nil, logf.ZapLogger(false),
			jenkins, false, false, nil, nil)
		err := fakeClient.Create(context.TODO(), secret)
		require.NoError(t, err)

		got, err := baseReconcileLoop.validateCustomization(customization, "spec.groovyScripts")

		assert.NoError(t, err)
		assert.False(t, got)
	})
}
