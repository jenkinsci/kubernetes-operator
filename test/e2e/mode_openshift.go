// +build OpenShift

package e2e

import (
	"testing"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/base/resources"

	corev1 "k8s.io/api/core/v1"
)

const skipTestSafeRestart = false

func updateJenkinsCR(t *testing.T, jenkins *v1alpha2.Jenkins) {
	t.Log("Update Jenkins CR: OpenShift")

	jenkins.Spec.Master.Containers[0].Image = "quay.io/openshift/origin-jenkins"
	jenkins.Spec.Master.Containers[0].Command = []string{
		"bash",
		"-c",
		"/var/jenkins/scripts/init.sh && exec /usr/bin/go-init -main /usr/libexec/s2i/run",
	}
	jenkins.Spec.Master.Containers[0].Env = append(jenkins.Spec.Master.Containers[0].Env,
		corev1.EnvVar{
			Name:  "JENKINS_SERVICE_NAME",
			Value: resources.GetJenkinsHTTPServiceName(jenkins),
		},
		corev1.EnvVar{
			Name:  "JNLP_SERVICE_NAME",
			Value: resources.GetJenkinsSlavesServiceName(jenkins),
		},
	)
}
