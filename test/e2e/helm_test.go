// +build Helm

package e2e

import (
	"fmt"
	"os/exec"
	"testing"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis"
	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestLintHelmChart(t *testing.T) {
	t.Parallel()

	cmd := exec.Command("helm", "lint", "./chart/jenkins-operator")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
}

func TestDeployHelmChart(t *testing.T) {
	// Given
	t.Parallel()
	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup()

	namespace, err := ctx.GetNamespace()
	require.NoError(t, err)

	jenkinsServiceList := &v1alpha2.JenkinsList{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1alpha2.Kind,
			APIVersion: v1alpha2.SchemeGroupVersion.String(),
		},
	}
	err = framework.AddToFrameworkScheme(apis.AddToScheme, jenkinsServiceList)
	require.NoError(t, err)

	jenkins := &v1alpha2.Jenkins{
		TypeMeta: v1alpha2.JenkinsTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jenkins",
			Namespace: namespace,
		},
	}

	cmd := exec.Command("helm", "install", "helm", "./chart/jenkins-operator", "--namespace", namespace, "--debug",
		"--set-string", fmt.Sprintf("jenkins.namespace=%s", namespace))
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))

	// Then
	waitForJenkinsBaseConfigurationToComplete(t, jenkins)
	waitForJenkinsUserConfigurationToComplete(t, jenkins)
}
