package e2e

import (
	"flag"
	"io/ioutil"
	"log"
	"testing"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis"
	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/constants"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
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
	log.SetOutput(ioutil.Discard)
	framework.MainEntry(m)
}

func setupTest(t *testing.T) (string, *framework.Context) {
	ctx := framework.NewContext(t)
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

	jenkinsServiceList := &v1alpha2.JenkinsList{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1alpha2.Kind,
			APIVersion: v1alpha2.SchemeGroupVersion.String(),
		},
	}
	err = framework.AddToFrameworkScheme(apis.AddToScheme, jenkinsServiceList)
	if err != nil {
		t.Fatalf("could not add scheme to framework scheme: %v", err)
	}

	namespace, err := ctx.GetOperatorNamespace()
	if err != nil {
		t.Fatalf("could not get namespace: %v", err)
	}
	t.Logf("Test namespace '%s'", namespace)

	// wait for jenkins-operator to be ready
	err = e2eutil.WaitForDeployment(t, framework.Global.KubeClient, namespace, jenkinsOperatorDeploymentName, 1, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	return namespace, ctx
}
