package e2e

import (
	"flag"
	"testing"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis"
	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/constants"

	f "github.com/operator-framework/operator-sdk/pkg/test"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	jenkinsOperatorDeploymentName     = constants.OperatorName
	seedJobConfigurationParameterName = "seed-job-config"
	hostnameParameterName             = "jenkins-api-hostname"
	portParameterName                 = "jenkins-api-port"
	nodePortParameterName             = "jenkins-api-use-nodeport"
)

var (
	seedJobConfigurationFile *string
	hostname                 *string
	port                     *int
	useNodePort              *bool
)

func TestMain(m *testing.M) {
	seedJobConfigurationFile = flag.String(seedJobConfigurationParameterName, "", "path to seed job config")
	hostname = flag.String(hostnameParameterName, "", "Hostname or IP of Jenkins API. It can be service name, node IP or localhost.")
	port = flag.Int(portParameterName, -1, "The port on which Jenkins API is working. Note: If you want to use nodePort don't set this setting and --jenkins-api-use-nodeport must be false.")
	useNodePort = flag.Bool(nodePortParameterName, false, "Connect to Jenkins API using the nodePort instead of service port. If you want to set this as true - don't set --jenkins-api-port.")

	f.MainEntry(m)
}

func setupTest(t *testing.T) (string, *framework.TestCtx) {
	ctx := framework.NewTestCtx(t)
	err := ctx.InitializeClusterResources(nil)
	if err != nil {
		t.Fatalf("could not initialize cluster resources: %v", err)
	}

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

	namespace, err := ctx.GetNamespace()
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
