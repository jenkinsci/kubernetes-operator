package resources

import (
	"fmt"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/constants"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// OperatorUserName defines username for Jenkins API calls
	OperatorUserName = "jenkins-operator"
	// OperatorCredentialsSecretUserNameKey defines key of username in operator credentials secret
	OperatorCredentialsSecretUserNameKey = "user"
	// OperatorCredentialsSecretPasswordKey defines key of password in operator credentials secret
	OperatorCredentialsSecretPasswordKey = "password"
	// OperatorCredentialsSecretTokenKey defines key of token in operator credentials secret
	OperatorCredentialsSecretTokenKey = "token"
	// OperatorCredentialsSecretTokenCreationKey defines key of token creation time in operator credentials secret
	OperatorCredentialsSecretTokenCreationKey = "tokenCreationTime"
)

func buildSecretTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		Kind:       "Secret",
		APIVersion: "v1",
	}
}

// GetOperatorCredentialsSecretName returns name of Kubernetes secret used to store jenkins operator credentials
// to allow calls to Jenkins API
func GetOperatorCredentialsSecretName(jenkins *v1alpha2.Jenkins) string {
	return fmt.Sprintf("%s-credentials-%s", constants.OperatorName, jenkins.Name)
}

// NewOperatorCredentialsSecret builds the Kubernetes secret used to store jenkins operator credentials
// to allow calls to Jenkins API
func NewOperatorCredentialsSecret(meta metav1.ObjectMeta, jenkins *v1alpha2.Jenkins) *corev1.Secret {
	meta.Name = GetOperatorCredentialsSecretName(jenkins)
	return &corev1.Secret{
		TypeMeta:   buildSecretTypeMeta(),
		ObjectMeta: meta,
		Data: map[string][]byte{
			OperatorCredentialsSecretUserNameKey: []byte(OperatorUserName),
			OperatorCredentialsSecretPasswordKey: []byte(randomString(20)),
		},
	}
}
