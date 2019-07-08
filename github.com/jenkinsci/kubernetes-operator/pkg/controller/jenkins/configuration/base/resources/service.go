package resources

import (
	"fmt"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/constants"

	corev1 "k8s.io/api/core/v1"
)

// UpdateService returns new service with override fields from config
func UpdateService(actual corev1.Service, config v1alpha2.Service) corev1.Service {
	actual.ObjectMeta.Annotations = config.Annotations
	for key, value := range config.Labels {
		actual.ObjectMeta.Labels[key] = value
	}
	actual.Spec.Type = config.Type
	actual.Spec.LoadBalancerIP = config.LoadBalancerIP
	actual.Spec.LoadBalancerSourceRanges = config.LoadBalancerSourceRanges
	if len(actual.Spec.Ports) == 0 {
		actual.Spec.Ports = []corev1.ServicePort{{}}
	}
	actual.Spec.Ports[0].Port = config.Port
	if config.NodePort != 0 {
		actual.Spec.Ports[0].NodePort = config.NodePort
	}

	return actual
}

// GetJenkinsHTTPServiceName returns Kubernetes service name used for expose Jenkins HTTP endpoint
func GetJenkinsHTTPServiceName(jenkins *v1alpha2.Jenkins) string {
	return fmt.Sprintf("%s-http-%s", constants.OperatorName, jenkins.ObjectMeta.Name)
}

func GetJenkinsServiceNodePort(jenkins *v1alpha2.Jenkins) int32 {
	return jenkins.Spec.Service.NodePort
}

// GetJenkinsSlavesServiceName returns Kubernetes service name used for expose Jenkins slave endpoint
func GetJenkinsSlavesServiceName(jenkins *v1alpha2.Jenkins) string {
	return fmt.Sprintf("%s-slave-%s", constants.OperatorName, jenkins.ObjectMeta.Name)
}
