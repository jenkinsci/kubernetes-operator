package e2e

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

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
	_, jenkinsConfig := createOpenShiftConfiguration(namespace, openshfitE2e)
	jenkins := createJenkinsCRFromConfiguration(t, jenkinsConfig)
	waitForJenkinsBaseConfigurationToComplete(t, jenkins)
	verifyServiceAccountAnnotations(t, jenkins)
	jenkinsClient, cleanUpFunc := verifyJenkinsAPIConnection(t, jenkins, namespace)
	defer cleanUpFunc()
	verifyPlugins(t, jenkinsClient, jenkins)
}

func TestOpenShiftPlugins(t *testing.T) {
	t.Parallel()
	namespace, ctx := setupTest(t)
	// Deletes test namespace
	defer showLogsAndCleanup(t, ctx)
	plugins, sample := createOpenShiftConfiguration(namespace, "openshift-plugins-e2e")
	jenkins := createJenkinsCRFromConfiguration(t, sample)
	if !arrayContainsArray(plugins, jenkins.Spec.Master.Plugins) {
		assert.Fail(t, "Specified plugins not found in the created Jenkins")
	}
}

func verifyServiceAccountAnnotations(t *testing.T, jenkins *v1alpha2.Jenkins) {
	serviceaccount := getServiceAccount(t, jenkins)
	assert.NotNil(t, serviceaccount)
	routeAnnotation := "'{\"kind\":\"OAuthRedirectReference\",\"apiVersion\":\"v1\",\"reference\":{\"kind\":\"Route\",\"name\":\"jenkins-route\"}}"
	annotations := make(map[string]string)
	annotations["serviceaccounts.openshift.io/oauth-redirectreference.jenkins"] = routeAnnotation
	assertMapContainsElementsFromAnotherMap(t, serviceaccount.Annotations, annotations)
}

func createOpenShiftConfiguration(namespace string, name string) ([]v1alpha2.Plugin, JenkinsE2EConfiguration) {
	plugins := []v1alpha2.Plugin{
		{Name: "openshift-sync", Version: "1.0.44"},
		{Name: "openshift-oauth-login-plugin", Version: "1.0.33"},
		{Name: "openshift-client", Version: "1.29.4"},
	}
	sample := JenkinsE2EConfiguration{
		name:              name,
		namespace:         namespace,
		priorityClassName: "",
		plugins:           plugins,
		seedJob:           &[]v1alpha2.SeedJob{},
		groovyScripts:     v1alpha2.GroovyScripts{},
		casc:              v1alpha2.ConfigurationAsCode{},
	}
	return plugins, sample
}

func arrayContainsArray(expected []v1alpha2.Plugin, actual []v1alpha2.Plugin) bool {
	for _, expectedValue := range expected {
		if !contains(actual, expectedValue) {
			return false
		}
	}
	return true
}

func contains(array []v1alpha2.Plugin, element v1alpha2.Plugin) bool {
	for _, a := range array {
		if a == element {
			return true
		}
	}
	return false
}
