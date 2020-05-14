package e2e

import (
	"fmt"
	"testing"

	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/plugins"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
)

const openshfitE2e = "openshift-base-configuration-e2e"

func TestBaseOpenShiftConfiguration(t *testing.T) {
	t.Parallel()
	namespace, ctx := setupTest(t)
	defer showLogsAndCleanup(t, ctx)

	numberOfExecutors := 6
	numberOfExecutorsEnvName := "NUMBER_OF_EXECUTORS"
	stringData := make(map[string]string)
	stringData[numberOfExecutorsEnvName] = fmt.Sprintf("%d", numberOfExecutors)

	// base
	createUserConfigurationSecret(t, namespace, stringData)
	createUserConfigurationConfigMap(t, namespace, numberOfExecutorsEnvName, "")
	sample := JenkinsSample{
		name:              openshfitE2e,
		namespace:         namespace,
		priorityClassName: "",
		seedJob:           &[]v1alpha2.SeedJob{},
		groovyScripts:     v1alpha2.GroovyScripts{},
		casc:              v1alpha2.ConfigurationAsCode{},
	}
	jenkins := createJenkinsCRFromSample(t, sample)
	createDefaultLimitsForContainersInNamespace(t, namespace)
	waitForJenkinsBaseConfigurationToComplete(t, jenkins)
	verifyJenkinsMasterPodAttributes(t, jenkins)
	verifyServiceAccountAnnotations(t, jenkins)
	jenkinsClient, cleanUpFunc := verifyJenkinsAPIConnection(t, jenkins, namespace)
	defer cleanUpFunc()
	verifyPlugins(t, jenkinsClient, jenkins)
}

//func verifyServiceAccountAnnotations(t *testing.T, jenkins *v1alpha2.Jenkins) {
//	serviceaccount := getServiceAccount(t, jenkins)
//	assert.NotNil(t, serviceaccount)
//	routeAnnotation := "'{\"kind\":\"OAuthRedirectReference\",\"apiVersion\":\"v1\",\"reference\":{\"kind\":\"Route\",\"name\":\"jenkins-route\"}}"
//	annotations := make(map[string]string)
//	annotations["serviceaccounts.openshift.io/oauth-redirectreference.jenkins"] = routeAnnotation
//	assertMapContainsElementsFromAnotherMap(t, serviceaccount.Annotations, annotations)
//}

func TestOpenShiftPlugins(t *testing.T) {
	t.Parallel()
	namespace, ctx := setupTest(t)
	// Deletes test namespace
	defer showLogsAndCleanup(t, ctx)
	sample := JenkinsSample{
		name:              "openshift-plugins-e2e",
		namespace:         namespace,
		priorityClassName: "",
		seedJob:           &[]v1alpha2.SeedJob{},
		groovyScripts:     v1alpha2.GroovyScripts{},
		casc:              v1alpha2.ConfigurationAsCode{},
	}
	jenkins := createJenkinsCRFromSample(t, sample)
	jenkinsClient, cleanUpFunc := verifyJenkinsAPIConnection(t, jenkins, namespace)
	defer cleanUpFunc()
	installedPlugins, err := jenkinsClient.GetPlugins(1)
	if err != nil {
		t.Fatal(err)
	}

	for _, openshiftPlugin := range plugins.OpenshiftPlugins() {
		if found, ok := isPluginValid(installedPlugins, openshiftPlugin); !ok {
			t.Fatalf("Invalid plugin '%s', actual '%v'", openshiftPlugin, found)
		}
	}
}
