package controllers

import (
	"testing"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/internal/testutil"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestDefaultJenkinsProbesUseHealth(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, v1alpha2.AddToScheme(scheme))
	manifest := &v1alpha2.Jenkins{ObjectMeta: metav1.ObjectMeta{Name: "health", Namespace: "default"}}
	reconciler := &JenkinsReconciler{Client: testutil.NewClientBuilder().WithScheme(scheme).WithObjects(manifest).Build()}
	_, err := reconciler.setDefaults(manifest)
	require.NoError(t, err)
	master := manifest.Spec.Master.Containers[0]
	require.Equal(t, "/health", master.ReadinessProbe.HTTPGet.Path)
	require.Equal(t, "/health", master.LivenessProbe.HTTPGet.Path)
}

func TestDefaultingPreservesCustomJenkinsProbes(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, v1alpha2.AddToScheme(scheme))
	manifest := &v1alpha2.Jenkins{
		ObjectMeta: metav1.ObjectMeta{Name: "custom-probes", Namespace: "default"},
		Spec: v1alpha2.JenkinsSpec{Master: v1alpha2.JenkinsMaster{Containers: []v1alpha2.Container{{
			Name:           "jenkins-master",
			ReadinessProbe: &corev1.Probe{ProbeHandler: corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{Path: "/ready"}}},
			LivenessProbe:  &corev1.Probe{ProbeHandler: corev1.ProbeHandler{Exec: &corev1.ExecAction{Command: []string{"true"}}}},
		}}}},
	}
	reconciler := &JenkinsReconciler{Client: testutil.NewClientBuilder().WithScheme(scheme).WithObjects(manifest).Build()}
	_, err := reconciler.setDefaults(manifest)
	require.NoError(t, err)
	master := manifest.Spec.Master.Containers[0]
	require.Equal(t, "/ready", master.ReadinessProbe.HTTPGet.Path)
	require.Equal(t, []string{"true"}, master.LivenessProbe.Exec.Command)
}
