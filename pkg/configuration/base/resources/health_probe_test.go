package resources

import (
	"testing"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestHealthProbePaths(t *testing.T) {
	tests := []struct {
		name, prefix, readiness, liveness, wantReadiness, wantLiveness string
	}{
		{"default", "", "/health", "/health", "/health", "/health"},
		{"prefix", "/jenkins", "/health", "/health", "/jenkins/health", "/jenkins/health"},
		{"trailing slash", "/jenkins/", "/health", "/health", "/jenkins/health", "/jenkins/health"},
		{"root prefix", "/", "/health", "/health", "/health", "/health"},
		{"already prefixed", "/jenkins", "/jenkins/health", "/jenkins/health", "/jenkins/health", "/jenkins/health"},
		{"custom paths", "", "/ready", "/live", "/ready", "/live"},
		{"custom paths with prefix", "/jenkins", "/custom/ready", "/custom/live", "/custom/ready", "/custom/live"},
		{"explicit login probes", "", "/login", "/login", "/login", "/login"},
		{"custom readiness", "/jenkins", "/ready", "/health", "/ready", "/jenkins/health"},
		{"custom liveness", "/jenkins", "/health", "/live", "/jenkins/health", "/live"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest := &v1alpha2.Jenkins{Spec: v1alpha2.JenkinsSpec{Master: v1alpha2.JenkinsMaster{Containers: []v1alpha2.Container{{
				ReadinessProbe: &corev1.Probe{ProbeHandler: corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{Path: tt.readiness}}},
				LivenessProbe:  &corev1.Probe{ProbeHandler: corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{Path: tt.liveness}}},
			}}}}}
			if tt.prefix != "" {
				manifest.Spec.Master.Containers[0].Env = []corev1.EnvVar{{Name: "JENKINS_OPTS", Value: "--prefix=" + tt.prefix}}
			}
			for range 2 {
				container := NewJenkinsMasterContainer(manifest)
				assert.Equal(t, tt.wantReadiness, container.ReadinessProbe.HTTPGet.Path)
				assert.Equal(t, tt.wantLiveness, container.LivenessProbe.HTTPGet.Path)
			}
		})
	}
}

func TestHealthProbesWithMixedHandlers(t *testing.T) {
	health := &corev1.Probe{ProbeHandler: corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{Path: "/health"}}}
	exec := &corev1.Probe{ProbeHandler: corev1.ProbeHandler{Exec: &corev1.ExecAction{Command: []string{"true"}}}}
	tcp := &corev1.Probe{ProbeHandler: corev1.ProbeHandler{TCPSocket: &corev1.TCPSocketAction{}}}
	tests := []struct {
		name                string
		readiness, liveness *corev1.Probe
	}{
		{"HTTP and exec", health, exec},
		{"exec and HTTP", exec, health},
		{"HTTP and TCP", health, tcp},
		{"TCP and HTTP", tcp, health},
		{"nil and HTTP", nil, health},
		{"HTTP and nil", health, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			master := v1alpha2.Container{Env: []corev1.EnvVar{{Name: "JENKINS_OPTS", Value: "--prefix=/jenkins"}}}
			if tt.readiness != nil {
				master.ReadinessProbe = tt.readiness.DeepCopy()
			}
			if tt.liveness != nil {
				master.LivenessProbe = tt.liveness.DeepCopy()
			}
			manifest := &v1alpha2.Jenkins{Spec: v1alpha2.JenkinsSpec{Master: v1alpha2.JenkinsMaster{Containers: []v1alpha2.Container{master}}}}
			var container corev1.Container
			if !assert.NotPanics(t, func() { container = NewJenkinsMasterContainer(manifest) }) {
				return
			}
			for _, pair := range []struct{ original, got *corev1.Probe }{{tt.readiness, container.ReadinessProbe}, {tt.liveness, container.LivenessProbe}} {
				if pair.original != nil && pair.original.HTTPGet != nil {
					assert.Equal(t, "/jenkins/health", pair.got.HTTPGet.Path)
				} else {
					assert.Equal(t, pair.original, pair.got)
				}
			}
		})
	}
}
