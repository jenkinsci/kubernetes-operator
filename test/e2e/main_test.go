package e2e

import (
	"flag"
	"testing"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis"
	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/constants"

	"github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	jenkinsOperatorDeploymentName     = constants.OperatorName
	seedJobConfigurationParameterName = "seed-job-config"
)

var (
	seedJobConfigurationFile *string
)

func TestMain(m *testing.M) {
	seedJobConfigurationFile = flag.String(seedJobConfigurationParameterName, "", "path to seed job config")
	test.MainEntry(m)
}

func setupTest(t *testing.T) (string, *test.Context) {
	apiVersion := v1alpha2.SchemeGroupVersion.String()
	kind := v1alpha2.Kind
	jenkinsList := &v1alpha2.JenkinsList{
		TypeMeta: metav1.TypeMeta{
			Kind:       kind,
			APIVersion: apiVersion,
		},
	}
	if err := test.AddToFrameworkScheme(apis.AddToScheme, jenkinsList); err != nil {
		t.Fatalf("failed to add '%s %s': %v", apiVersion, kind, err)
	}

	ctx := test.NewContext(t)
	err := ctx.InitializeClusterResources(nil)
	if err != nil {
		t.Fatalf("could not initialize cluster resources: %v", err)
	}
	defer func() {
		showLogsIfTestHasFailed(t, ctx)
		if t.Failed() && ctx != nil {
			ctx.Cleanup()
		}
	}()
	namespace, err := ctx.GetOperatorNamespace()
	if err != nil {
		t.Fatalf("could not get namespace: %s : %v", namespace, err)
	}
	t.Logf("Test namespace '%s'", namespace)

	// wait for jenkins-operator to be ready
	err = e2eutil.WaitForOperatorDeployment(t, test.Global.KubeClient, namespace, jenkinsOperatorDeploymentName, 1, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}
	return namespace, ctx
}
