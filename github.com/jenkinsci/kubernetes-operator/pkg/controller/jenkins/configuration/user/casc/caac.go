package casc

import (
	"fmt"
	"strings"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/groovy"

	"github.com/go-logr/logr"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
)

// ConfigurationAsCode defines API which configures Jenkins with help Configuration as a code plugin
type ConfigurationAsCode struct {
	groovyClient *groovy.Groovy
}

// New creates new instance of ConfigurationAsCode
func New(jenkinsClient jenkinsclient.Jenkins, k8sClient k8s.Client, logger logr.Logger, jenkins *v1alpha2.Jenkins) *ConfigurationAsCode {
	return &ConfigurationAsCode{
		groovyClient: groovy.New(jenkinsClient, k8sClient, logger, jenkins, "user-casc", jenkins.Spec.ConfigurationAsCode.Customization),
	}
}

// Ensure configures Jenkins with help Configuration as a code plugin
func (c *ConfigurationAsCode) Ensure(jenkins *v1alpha2.Jenkins) (requeue bool, err error) {
	requeue, err = c.groovyClient.WaitForSecretSynchronization(resources.ConfigurationAsCodeSecretVolumePath)
	if err != nil || requeue {
		return requeue, err
	}

	return c.groovyClient.Ensure(func(name string) bool {
		return strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml")
	}, func(groovyScript string) string {
		return fmt.Sprintf(applyConfigurationAsCodeGroovyScriptFmt, groovyScript)
	})
}

const applyConfigurationAsCodeGroovyScriptFmt = `
def config = '''
%s
'''
def stream = new ByteArrayInputStream(config.getBytes('UTF-8'))

def source = new io.jenkins.plugins.casc.yaml.YamlSource(stream, io.jenkins.plugins.casc.yaml.YamlSource.READ_FROM_INPUTSTREAM)
io.jenkins.plugins.casc.ConfigurationAsCode.get().configureWith(source)
`
