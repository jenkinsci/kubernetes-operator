package resources

import (
	"fmt"
	"text/template"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/internal/render"
	"github.com/jenkinsci/kubernetes-operator/pkg/constants"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const createOperatorUserFileName = "createOperatorUser.groovy"

var createOperatorUserGroovyFmtTemplate = template.Must(template.New(createOperatorUserFileName).Parse(`
import hudson.security.*

{{- if .Enable }}
def jenkins = jenkins.model.Jenkins.getInstance()
def operatorUserCreatedFile = new File('{{ .OperatorUserCreatedFilePath }}')

if (!operatorUserCreatedFile.exists()) {
	def hudsonRealm = new HudsonPrivateSecurityRealm(false)
	hudsonRealm.createAccount(
		new File('{{ .OperatorCredentialsPath }}/{{ .OperatorUserNameFile }}').text,
		new File('{{ .OperatorCredentialsPath }}/{{ .OperatorPasswordFile }}').text)
	jenkins.setSecurityRealm(hudsonRealm)

	def strategy = new FullControlOnceLoggedInAuthorizationStrategy()
	strategy.setAllowAnonymousRead(false)
	jenkins.setAuthorizationStrategy(strategy)
	jenkins.save()

	operatorUserCreatedFile.createNewFile()
}
{{- end }}
`))

func buildCreateJenkinsOperatorUserGroovyScript(jenkins *v1alpha2.Jenkins) (*string, error) {
	data := struct {
		Enable                      bool
		OperatorCredentialsPath     string
		OperatorUserNameFile        string
		OperatorPasswordFile        string
		OperatorUserCreatedFilePath string
	}{
		Enable:                      jenkins.Spec.JenkinsAPISettings.AuthorizationStrategy == v1alpha2.CreateUserAuthorizationStrategy,
		OperatorCredentialsPath:     jenkinsOperatorCredentialsVolumePath,
		OperatorUserNameFile:        OperatorCredentialsSecretUserNameKey,
		OperatorPasswordFile:        OperatorCredentialsSecretPasswordKey,
		OperatorUserCreatedFilePath: getJenkinsHomePath(jenkins) + "/operatorUserCreated",
	}

	output, err := render.Render(createOperatorUserGroovyFmtTemplate, data)
	if err != nil {
		return nil, err
	}

	return &output, nil
}

// GetInitConfigurationConfigMapName returns name of Kubernetes config map used to init configuration
func GetInitConfigurationConfigMapName(jenkins *v1alpha2.Jenkins) string {
	return fmt.Sprintf("%s-init-configuration-%s", constants.OperatorName, jenkins.Name)
}

// NewInitConfigurationConfigMap builds Kubernetes config map used to init configuration
func NewInitConfigurationConfigMap(meta metav1.ObjectMeta, jenkins *v1alpha2.Jenkins) (*corev1.ConfigMap, error) {
	meta.Name = GetInitConfigurationConfigMapName(jenkins)

	createJenkinsOperatorUserGroovy, err := buildCreateJenkinsOperatorUserGroovyScript(jenkins)
	if err != nil {
		return nil, err
	}

	return &corev1.ConfigMap{
		TypeMeta:   buildConfigMapTypeMeta(),
		ObjectMeta: meta,
		Data: map[string]string{
			createOperatorUserFileName: *createJenkinsOperatorUserGroovy,
		},
	}, nil
}
