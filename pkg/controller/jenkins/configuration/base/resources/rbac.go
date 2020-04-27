package resources

import (
	"k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	createVerb        = "create"
	deleteVerb        = "delete"
	getVerb           = "get"
	listVerb          = "list"
	watchVerb         = "watch"
	patchVerb         = "patch"
	updateVerb        = "update"
	EmptyApiGroups    = ""
	OpenshiftApiGroup = "image.openshift.io"
	BuildApiGroup     = "build.openshift.io"

)

// NewRole returns rbac role for jenkins master
func NewRole(meta metav1.ObjectMeta) *v1.Role {
	rules := NewDefaultPolicyRules()
	return &v1.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: meta,
		Rules:      rules,
	}
}

// NewRoleBinding returns rbac role binding for jenkins master
func NewRoleBinding(name, namespace, serviceAccountName string, roleRef v1.RoleRef) *v1.RoleBinding {
	return &v1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		RoleRef: roleRef,
		Subjects: []v1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      serviceAccountName,
				Namespace: namespace,
			},
		},
	}
}

func NewDefaultPolicyRules() []v1.PolicyRule {
	var rules []v1.PolicyRule
	Default := []string{createVerb, deleteVerb, getVerb, listVerb, patchVerb, updateVerb, watchVerb}
	ReadOnly := []string{getVerb, listVerb, watchVerb}

	rules = append(rules,  NewPolicyRule(EmptyApiGroups, "configmaps", Default))
	rules = append(rules,  NewPolicyRule(EmptyApiGroups, "pods", Default))
	rules = append(rules,  NewPolicyRule(EmptyApiGroups, "pods/exec", Default))
	rules = append(rules,  NewPolicyRule(EmptyApiGroups, "pods/portforward", Default))
	rules = append(rules,  NewPolicyRule(EmptyApiGroups, "pods/log", ReadOnly))
	rules = append(rules,  NewPolicyRule(EmptyApiGroups, "secrets", ReadOnly))
	rules = append(rules,  NewPolicyRule(OpenshiftApiGroup, "imagestreams", ReadOnly))
	rules = append(rules,  NewPolicyRule(BuildApiGroup, "buildconfigs", ReadOnly))
	rules = append(rules,  NewPolicyRule(BuildApiGroup, "builds", ReadOnly))
	return rules
}

// NewPolicyRule returns a policyRule allowing verbs on resources
func NewPolicyRule(apiGroup string, resource string, verbs []string) v1.PolicyRule {
	rule := v1.PolicyRule{
		APIGroups: []string{apiGroup},
		Resources: []string{resource},
		Verbs:     verbs,
	}
	return rule
}
