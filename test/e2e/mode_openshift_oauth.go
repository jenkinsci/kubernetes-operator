//go:build OpenShiftOAuth
// +build OpenShiftOAuth

package e2e

// TODO
/*
import (
	"context"
	"testing"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/constants"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	skipTestSafeRestart   = true
	skipTestPriorityClass = true
)

func updateJenkinsCR(t *testing.T, jenkins *v1alpha2.Jenkins) {
	t.Log("Update Jenkins CR: OpenShiftOAuth")

	adminRoleBinding := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "admin",
			Namespace: jenkins.Namespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "admin",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      constants.OperatorName,
				Namespace: jenkins.Namespace,
			},
		},
	}
	if err := framework.Global.Client.Create(context.TODO(), adminRoleBinding, nil); err != nil {
		t.Fatal(err)
	}

	jenkins.Spec.JenkinsAPISettings = v1alpha2.JenkinsAPISettings{AuthorizationStrategy: v1alpha2.ServiceAccountAuthorizationStrategy}
	jenkins.Spec.Master.Containers[0].Image = "quay.io/openshift/origin-jenkins"
	jenkins.Spec.Master.Containers[0].Command = []string{
		"bash",
		"-c",
		"/var/jenkins/scripts/init.sh && exec /usr/bin/go-init -main /usr/libexec/s2i/run",
	}
	jenkins.Spec.Roles = []rbacv1.RoleRef{
		{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "admin",
		},
	}
	jenkins.Spec.Master.DisableCSRFProtection = true
	jenkins.Spec.Master.Containers[0].Env = append(jenkins.Spec.Master.Containers[0].Env,
		corev1.EnvVar{
			Name:  "OPENSHIFT_ENABLE_OAUTH",
			Value: "true",
		},
		corev1.EnvVar{
			Name:  "OPENSHIFT_ENABLE_REDIRECT_PROMPT",
			Value: "true",
		},
		corev1.EnvVar{
			Name:  "DISABLE_ADMINISTRATIVE_MONITORS",
			Value: "false",
		},
		corev1.EnvVar{
			Name:  "KUBERNETES_TRUST_CERTIFICATES",
			Value: "true",
		},
		corev1.EnvVar{
			Name:  "JENKINS_UC_INSECURE",
			Value: "false",
		},
		corev1.EnvVar{
			Name:  "JENKINS_SERVICE_NAME",
			Value: resources.GetJenkinsHTTPServiceName(jenkins),
		},
		corev1.EnvVar{
			Name:  "JNLP_SERVICE_NAME",
			Value: resources.GetJenkinsSlavesServiceName(jenkins),
		},
	)

	if len(jenkins.Spec.Master.Plugins) == 4 && jenkins.Spec.Master.Plugins[3].Name == "devoptics" {
		jenkins.Spec.Master.Plugins = jenkins.Spec.Master.Plugins[0:3] // remove devoptics plugin
	}
}
*/
