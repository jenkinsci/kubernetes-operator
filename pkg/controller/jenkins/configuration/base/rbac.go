package base

import (
	"context"
	"fmt"
	"strings"

	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/base/resources"

	stackerr "github.com/pkg/errors"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ReconcileJenkinsBaseConfiguration) createRBAC(meta metav1.ObjectMeta) error {
	err := r.createServiceAccount(meta)
	if err != nil {
		return err
	}

	role := resources.NewRole(meta)
	err = r.CreateOrUpdateResource(role)
	if err != nil {
		return stackerr.WithStack(err)
	}

	roleBinding := resources.NewRoleBinding(meta.Name, meta.Namespace, meta.Name, rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "Role",
		Name:     meta.Name,
	})
	err = r.CreateOrUpdateResource(roleBinding)
	if err != nil {
		return stackerr.WithStack(err)
	}

	return nil
}

func (r *ReconcileJenkinsBaseConfiguration) ensureExtraRBAC(meta metav1.ObjectMeta) error {
	var err error
	var name string
	for _, roleRef := range r.Configuration.Jenkins.Spec.Roles {
		name = getExtraRoleBindingName(meta.Name, roleRef)
		roleBinding := resources.NewRoleBinding(name, meta.Namespace, meta.Name, roleRef)
		err = r.CreateOrUpdateResource(roleBinding)
		if err != nil {
			return stackerr.WithStack(err)
		}
	}

	roleBindings := &rbacv1.RoleBindingList{}
	err = r.Client.List(context.TODO(), roleBindings, client.InNamespace(r.Configuration.Jenkins.Namespace))
	if err != nil {
		return stackerr.WithStack(err)
	}
	for _, roleBinding := range roleBindings.Items {
		if !strings.HasPrefix(roleBinding.Name, getExtraRoleBindingName(meta.Name, rbacv1.RoleRef{Kind: "Role"})) &&
			!strings.HasPrefix(roleBinding.Name, getExtraRoleBindingName(meta.Name, rbacv1.RoleRef{Kind: "ClusterRole"})) {
			continue
		}

		found := false
		for _, roleRef := range r.Configuration.Jenkins.Spec.Roles {
			name = getExtraRoleBindingName(meta.Name, roleRef)
			if roleBinding.Name == name {
				found = true
				continue
			}
		}
		if !found {
			r.logger.Info(fmt.Sprintf("Deleting RoleBinding '%s'", roleBinding.Name))
			if err = r.Client.Delete(context.TODO(), &roleBinding); err != nil {
				return stackerr.WithStack(err)
			}
		}
	}

	return nil
}

func getExtraRoleBindingName(serviceAccountName string, roleRef rbacv1.RoleRef) string {
	var typeName string
	if roleRef.Kind == "ClusterRole" {
		typeName = "cr"
	} else {
		typeName = "r"
	}
	return fmt.Sprintf("%s-%s-%s", serviceAccountName, typeName, roleRef.Name)
}


