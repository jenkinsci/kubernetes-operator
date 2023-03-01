package resources

import (
	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	routev1 "github.com/openshift/api/route/v1"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
)

var isRouteAPIAvailable = false
var routeAPIChecked = false

// UpdateRoute returns new route matching the service
func UpdateRoute(actual routev1.Route, jenkins *v1alpha2.Jenkins) routev1.Route {
	actualTargetService := actual.Spec.To
	serviceName := GetJenkinsHTTPServiceName(jenkins)
	if actualTargetService.Name != serviceName {
		actual.Spec.To.Name = serviceName
	}
	port := jenkins.Spec.Service.Port
	if actual.Spec.Port.TargetPort.IntVal != port {
		actual.Spec.Port.TargetPort = intstr.FromInt(int(port))
	}
	return actual
}

// IsRouteAPIAvailable tells if the Route API is installed and discoverable
func IsRouteAPIAvailable(clientSet *kubernetes.Clientset) bool {
	if routeAPIChecked {
		return isRouteAPIAvailable
	}
	gv := schema.GroupVersion{
		Group:   routev1.GroupName,
		Version: routev1.SchemeGroupVersion.Version,
	}
	if err := discovery.ServerSupportsVersion(clientSet, gv); err != nil {
		// error, API not available
		routeAPIChecked = true
		isRouteAPIAvailable = false
	} else {
		// API Exists
		routeAPIChecked = true
		isRouteAPIAvailable = true
	}
	return isRouteAPIAvailable
}
