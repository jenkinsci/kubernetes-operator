package resources

import (
	"fmt"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/constants"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewResourceObjectMeta builds ObjectMeta for all Kubernetes resources created by operator
func NewResourceObjectMeta(jenkins *v1alpha2.Jenkins) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      GetResourceName(jenkins),
		Namespace: jenkins.Namespace,
		Labels:    BuildResourceLabels(jenkins),
	}
}

// BuildResourceLabels returns labels for all Kubernetes resources created by operator
func BuildResourceLabels(jenkins *v1alpha2.Jenkins) map[string]string {
	return map[string]string{
		constants.LabelAppKey:       constants.LabelAppValue,
		constants.LabelJenkinsCRKey: jenkins.Name,
	}
}

// BuildLabelsForWatchedResources returns labels for Kubernetes resources which operator want to watch
// resources with that labels should not be deleted after Jenkins CR deletion, to prevent this situation don't set
// any owner
func BuildLabelsForWatchedResources(jenkins v1alpha2.Jenkins) map[string]string {
	return map[string]string{
		constants.LabelAppKey:       constants.LabelAppValue,
		constants.LabelJenkinsCRKey: jenkins.Name,
		constants.LabelWatchKey:     constants.LabelWatchValue,
	}
}

// GetResourceName returns name of Kubernetes resource base on Jenkins CR
func GetResourceName(jenkins *v1alpha2.Jenkins) string {
	return fmt.Sprintf("%s-%s", constants.LabelAppValue, jenkins.Name)
}

// VerifyIfLabelsAreSet check is selected labels are set for specific resource
func VerifyIfLabelsAreSet(object metav1.Object, requiredLabels map[string]string) bool {
	for key, value := range requiredLabels {
		if object.GetLabels()[key] != value {
			return false
		}
	}

	return true
}
