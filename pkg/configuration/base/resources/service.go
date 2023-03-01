package resources

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/constants"

	stackerr "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// ServiceKind the kind name for Service
const ServiceKind = "Service"

// UpdateService returns new service with override fields from config
func UpdateService(actual corev1.Service, config v1alpha2.Service, targetPort int32) corev1.Service {
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
	actual.Spec.Ports[0].TargetPort = intstr.IntOrString{IntVal: targetPort, Type: intstr.Int}
	if config.NodePort != 0 {
		actual.Spec.Ports[0].NodePort = config.NodePort
	}

	return actual
}

// GetJenkinsHTTPServiceName returns Kubernetes service name used for expose Jenkins HTTP endpoint
func GetJenkinsHTTPServiceName(jenkins *v1alpha2.Jenkins) string {
	return fmt.Sprintf("%s-http-%s", constants.OperatorName, jenkins.ObjectMeta.Name)
}

// GetJenkinsSlavesServiceName returns Kubernetes service name used for expose Jenkins slave endpoint
func GetJenkinsSlavesServiceName(jenkins *v1alpha2.Jenkins) string {
	return fmt.Sprintf("%s-slave-%s", constants.OperatorName, jenkins.ObjectMeta.Name)
}

// GetJenkinsHTTPServiceFQDN returns Kubernetes service FQDN used for expose Jenkins HTTP endpoint
func GetJenkinsHTTPServiceFQDN(jenkins *v1alpha2.Jenkins, kubernetesClusterDomain string) (string, error) {
	clusterDomain, err := getClusterDomain(kubernetesClusterDomain)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s-http-%s.%s.svc.%s", constants.OperatorName, jenkins.ObjectMeta.Name, jenkins.ObjectMeta.Namespace, clusterDomain), nil
}

// GetJenkinsSlavesServiceFQDN returns Kubernetes service FQDN used for expose Jenkins slave endpoint
func GetJenkinsSlavesServiceFQDN(jenkins *v1alpha2.Jenkins, kubernetesClusterDomain string) (string, error) {
	clusterDomain, err := getClusterDomain(kubernetesClusterDomain)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s-slave-%s.%s.svc.%s", constants.OperatorName, jenkins.ObjectMeta.Name, jenkins.ObjectMeta.Namespace, clusterDomain), nil
}

// GetClusterDomain returns Kubernetes cluster domain, default to "cluster.local"
func getClusterDomain(kubernetesClusterDomain string) (string, error) {
	isRunningInCluster, err := IsRunningInCluster()
	if !isRunningInCluster {
		return kubernetesClusterDomain, nil
	}
	if err != nil {
		return "", nil
	}

	apiSvc := "kubernetes.default.svc"

	cname, err := net.LookupCNAME(apiSvc)
	if err != nil {
		return "", stackerr.WithStack(err)
	}

	kubernetesClusterDomain = strings.TrimPrefix(cname, "kubernetes.default.svc")
	kubernetesClusterDomain = strings.TrimPrefix(kubernetesClusterDomain, ".")
	kubernetesClusterDomain = strings.TrimSuffix(kubernetesClusterDomain, ".")

	return kubernetesClusterDomain, nil
}

func IsRunningInCluster() (bool, error) {
	const inClusterNamespacePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	_, err := os.Stat(inClusterNamespacePath)
	if os.IsNotExist(err) {
		return false, nil
	} else if err == nil {
		return true, nil
	}
	return false, err
}
