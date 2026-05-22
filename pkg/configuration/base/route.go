package base

import (
	"context"
	"fmt"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"

	routev1 "github.com/openshift/api/route/v1"
	stackerr "github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// createRoute takes the ServiceName and Creates the Route based on it
func (r *JenkinsBaseConfigurationReconciler) createRoute(meta metav1.ObjectMeta, serviceName string, config *v1alpha2.Jenkins) error {
	route := routev1.Route{}
	name := fmt.Sprintf("jenkins-%s", config.Name)
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: meta.Namespace}, &route)
	if err != nil && apierrors.IsNotFound(err) {
		port := &routev1.RoutePort{
			TargetPort: intstr.FromString(""),
		}

		routeSpec := routev1.RouteSpec{
			TLS: &routev1.TLSConfig{
				InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyRedirect,
				Termination:                   routev1.TLSTerminationEdge,
			},
			To: routev1.RouteTargetReference{
				Kind: resources.ServiceKind,
				Name: serviceName,
			},
			Port: port,
		}
		actual := routev1.Route{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: meta.Namespace,
				Labels:    meta.Labels,
			},
			Spec: routeSpec,
		}
		route = resources.UpdateRoute(actual, config)
		if err = r.CreateResource(&route); err != nil {
			return stackerr.WithStack(err)
		}
	} else if err != nil {
		return stackerr.WithStack(err)
	}

	route.Labels = meta.Labels // make sure that user won't break service by hand
	route = resources.UpdateRoute(route, config)
	return stackerr.WithStack(r.UpdateResource(&route))
}
