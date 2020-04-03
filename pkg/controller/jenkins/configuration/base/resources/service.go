package resources

import (
	"fmt"
	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/constants"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	stackerr "github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"

	"net"
	"strings"
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

// GetJenkinsSlavesServiceName returns Kubernetes service name used for expose Jenkins slave endpoint
func GetJenkinsSlavesServiceName(jenkins *v1alpha2.Jenkins) string {
	return fmt.Sprintf("%s-slave-%s", constants.OperatorName, jenkins.ObjectMeta.Name)
}

// GetJenkinsHTTPServiceFQDN returns Kubernetes service FQDN used for expose Jenkins HTTP endpoint
func GetJenkinsHTTPServiceFQDN(jenkins *v1alpha2.Jenkins) (string, error) {
	clusterDomain, err := getClusterDomain()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s-http-%s.%s.svc.%s", constants.OperatorName, jenkins.ObjectMeta.Name, jenkins.ObjectMeta.Namespace, clusterDomain), nil
}

// GetJenkinsSlavesServiceFQDN returns Kubernetes service FQDN used for expose Jenkins slave endpoint
func GetJenkinsSlavesServiceFQDN(jenkins *v1alpha2.Jenkins) (string, error) {
	clusterDomain, err := getClusterDomain()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s-slave-%s.%s.svc.%s", constants.OperatorName, jenkins.ObjectMeta.Name, jenkins.ObjectMeta.Namespace, clusterDomain), nil
}

// GetClusterDomain returns Kubernetes cluster domain, default to "cluster.local"
func getClusterDomain() (string, error) {
	clusterDomain := "cluster.local"

	if ok, err := isRunningInCluster(); !ok {
		return clusterDomain, nil
	} else if err != nil {
		return "", nil
	}

	apiSvc := "kubernetes.default.svc"

	cname, err := net.LookupCNAME(apiSvc)
	if err != nil {
		return "", stackerr.WithStack(err)
	}

	clusterDomain = strings.TrimPrefix(cname, "kubernetes.default.svc")
	clusterDomain = strings.TrimPrefix(clusterDomain, ".")
	clusterDomain = strings.TrimSuffix(clusterDomain, ".")

	return clusterDomain, nil
}

func isRunningInCluster() (bool, error) {
	_, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		if err == k8sutil.ErrNoNamespace || err == k8sutil.ErrRunLocal {
			return false, nil
		}
		return true, nil
	}
	return false, stackerr.WithStack(err)
}