package resources

import (
	"k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	createVerb = "create"
	deleteVerb = "delete"
	getVerb    = "get"
	listVerb   = "list"
	watchVerb  = "watch"
	patchVerb  = "patch"
	updateVerb = "update"
)

// NewRole returns rbac role for jenkins master
func NewRole(meta metav1.ObjectMeta) *v1.Role {
	return &v1.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: meta,
		Rules: []v1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods/portforward"},
				Verbs:     []string{createVerb},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{createVerb, deleteVerb, getVerb, listVerb, patchVerb, updateVerb, watchVerb},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"pods/exec"},
				Verbs:     []string{createVerb, deleteVerb, getVerb, listVerb, patchVerb, updateVerb, watchVerb},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"pods/log"},
				Verbs:     []string{getVerb, listVerb, watchVerb},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{getVerb, listVerb, watchVerb},
			},
		},
	}
}

// NewRoleBinding returns rbac role binding for jenkins master
func NewRoleBinding(meta metav1.ObjectMeta) *v1.RoleBinding {
	return &v1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: meta,
		RoleRef: v1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     meta.Name,
		},
		Subjects: []v1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      meta.Name,
				Namespace: meta.Namespace,
			},
		},
	}
}
