package resources

import (
	"fmt"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

// NewJenkinsMasterPod builds Jenkins Master Kubernetes Pod resource.
func NewJenkinsDeployment(objectMeta metav1.ObjectMeta, jenkins *v1alpha2.Jenkins) *appsv1.Deployment {
	serviceAccountName := objectMeta.Name
	objectMeta.Annotations = jenkins.Spec.Master.Annotations
	objectMeta.Name = GetJenkinsDeploymentName(jenkins)
	selector := &metav1.LabelSelector{MatchLabels: objectMeta.Labels}
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      objectMeta.Name,
			Namespace: objectMeta.Namespace,
			Labels:    objectMeta.Labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32Ptr(1),
			Strategy: appsv1.DeploymentStrategy{Type: appsv1.RollingUpdateDeploymentStrategyType},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: objectMeta,
				Spec: corev1.PodSpec{
					ServiceAccountName: serviceAccountName,
					NodeSelector:       jenkins.Spec.Master.NodeSelector,
					Containers:         newContainers(jenkins),
					Volumes:            append(GetJenkinsMasterPodBaseVolumes(jenkins), jenkins.Spec.Master.Volumes...),
					SecurityContext:    jenkins.Spec.Master.SecurityContext,
					ImagePullSecrets:   jenkins.Spec.Master.ImagePullSecrets,
					Tolerations:        jenkins.Spec.Master.Tolerations,
					PriorityClassName:  jenkins.Spec.Master.PriorityClassName,
					HostAliases: 	    jenkins.Spec.Master.HostAliases,
				},
			},
			Selector: selector,
		},
	}
}

// GetJenkinsMasterPodName returns Jenkins pod name for given CR
func GetJenkinsDeploymentName(jenkins *v1alpha2.Jenkins) string {
	return fmt.Sprintf("jenkins-%s", jenkins.Name)
}
