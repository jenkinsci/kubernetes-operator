package resources

import (
	"testing"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

var jenkins = v1alpha2.Jenkins{
	Spec: v1alpha2.JenkinsSpec{
		Master: v1alpha2.JenkinsMaster{
			Containers: []v1alpha2.Container{
				{
					Env: []corev1.EnvVar{},
					ReadinessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{},
						},
					},
					LivenessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{},
						},
					},
				},
			},
		},
	},
}

func TestGetJenkinsOpts(t *testing.T) {
	t.Run("JENKINS_OPTS is uninitialized", func(t *testing.T) {
		jenkins.Spec.Master.Containers[0].Env =
			[]corev1.EnvVar{
				{Name: "", Value: ""},
			}

		opts := GetJenkinsOpts(jenkins)
		assert.Equal(t, 0, len(opts))
	})

	t.Run("JENKINS_OPTS is empty", func(t *testing.T) {
		jenkins.Spec.Master.Containers[0].Env =
			[]corev1.EnvVar{
				{Name: "JENKINS_OPTS", Value: ""},
			}

		opts := GetJenkinsOpts(jenkins)
		assert.Equal(t, 0, len(opts))
	})

	t.Run("JENKINS_OPTS have --prefix argument ", func(t *testing.T) {
		jenkins.Spec.Master.Containers[0].Env =
			[]corev1.EnvVar{
				{Name: "JENKINS_OPTS", Value: "--prefix=/jenkins"},
			}

		opts := GetJenkinsOpts(jenkins)

		assert.Equal(t, 1, len(opts))
		assert.NotContains(t, opts, "httpPort")
		assert.Contains(t, opts, "prefix")
		assert.Equal(t, opts["prefix"], "/jenkins")
	})

	t.Run("JENKINS_OPTS have --prefix and --httpPort argument", func(t *testing.T) {
		jenkins.Spec.Master.Containers[0].Env =
			[]corev1.EnvVar{
				{Name: "JENKINS_OPTS", Value: "--prefix=/jenkins --httpPort=8080"},
			}

		opts := GetJenkinsOpts(jenkins)

		assert.Equal(t, 2, len(opts))

		assert.Contains(t, opts, "prefix")
		assert.Equal(t, opts["prefix"], "/jenkins")

		assert.Contains(t, opts, "httpPort")
		assert.Equal(t, opts["httpPort"], "8080")
	})

	t.Run("JENKINS_OPTS have --httpPort argument", func(t *testing.T) {
		jenkins.Spec.Master.Containers[0].Env =
			[]corev1.EnvVar{
				{Name: "JENKINS_OPTS", Value: "--httpPort=8080"},
			}

		opts := GetJenkinsOpts(jenkins)

		assert.Equal(t, 1, len(opts))
		assert.NotContains(t, opts, "prefix")
		assert.Contains(t, opts, "httpPort")
		assert.Equal(t, opts["httpPort"], "8080")
	})

	t.Run("JENKINS_OPTS have --httpPort=--8080 argument", func(t *testing.T) {
		jenkins.Spec.Master.Containers[0].Env =
			[]corev1.EnvVar{
				{Name: "JENKINS_OPTS", Value: "--httpPort=--8080"},
			}

		opts := GetJenkinsOpts(jenkins)

		assert.Equal(t, 1, len(opts))
		assert.NotContains(t, opts, "prefix")
		assert.Contains(t, opts, "httpPort")
		assert.Equal(t, opts["httpPort"], "--8080")
	})
}

func TestSetLivenessAndReadinessPath(t *testing.T) {
	t.Run("JENKINS_OPTS uninitialized. Probes' paths default.", func(t *testing.T) {
		jenkins.Spec.Master.Containers[0].Env = []corev1.EnvVar{}

		jenkins.Spec.Master.Containers[0].ReadinessProbe =
			&corev1.Probe{
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/login",
					},
				},
			}

		jenkins.Spec.Master.Containers[0].LivenessProbe =
			&corev1.Probe{
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/login",
					},
				},
			}

		setLivenessAndReadinessPath(&jenkins)

		assert.Equal(t, httpGetPath, jenkins.Spec.Master.Containers[0].ReadinessProbe.HTTPGet.Path)
		assert.Equal(t, httpGetPath, jenkins.Spec.Master.Containers[0].LivenessProbe.HTTPGet.Path)
	})

	t.Run("JENKINS_OPTS initialized. Probes' paths default", func(t *testing.T) {
		jenkins.Spec.Master.Containers[0].Env =
			[]corev1.EnvVar{
				{Name: "JENKINS_OPTS", Value: "--prefix=/jenkins"},
			}

		jenkins.Spec.Master.Containers[0].ReadinessProbe =
			&corev1.Probe{
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/login",
					},
				},
			}

		jenkins.Spec.Master.Containers[0].LivenessProbe =
			&corev1.Probe{
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/login",
					},
				},
			}

		setLivenessAndReadinessPath(&jenkins)

		assert.Equal(t, "/jenkins/login", jenkins.Spec.Master.Containers[0].ReadinessProbe.HTTPGet.Path)
		assert.Equal(t, "/jenkins/login", jenkins.Spec.Master.Containers[0].LivenessProbe.HTTPGet.Path)
	})

	t.Run("JENKINS_OPTS initialized. Probes' paths customized", func(t *testing.T) {
		jenkins.Spec.Master.Containers[0].Env =
			[]corev1.EnvVar{
				{Name: "JENKINS_OPTS", Value: "--prefix=/jenkins"},
			}

		jenkins.Spec.Master.Containers[0].ReadinessProbe =
			&corev1.Probe{
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/jenkins/login",
					},
				},
			}

		jenkins.Spec.Master.Containers[0].LivenessProbe =
			&corev1.Probe{
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/jenkins/login",
					},
				},
			}

		setLivenessAndReadinessPath(&jenkins)

		assert.Equal(t, "/jenkins/login", jenkins.Spec.Master.Containers[0].ReadinessProbe.HTTPGet.Path)
		assert.Equal(t, "/jenkins/login", jenkins.Spec.Master.Containers[0].LivenessProbe.HTTPGet.Path)
	})

	t.Run("JENKINS_OPTS uninitialized. Probes' paths customized", func(t *testing.T) {
		jenkins.Spec.Master.Containers[0].Env = []corev1.EnvVar{}

		jenkins.Spec.Master.Containers[0].ReadinessProbe =
			&corev1.Probe{
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/jenkins/login",
					},
				},
			}

		jenkins.Spec.Master.Containers[0].LivenessProbe =
			&corev1.Probe{
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/jenkins/login",
					},
				},
			}

		setLivenessAndReadinessPath(&jenkins)

		assert.Equal(t, "/login", jenkins.Spec.Master.Containers[0].ReadinessProbe.HTTPGet.Path)
		assert.Equal(t, "/login", jenkins.Spec.Master.Containers[0].LivenessProbe.HTTPGet.Path)
	})
}
