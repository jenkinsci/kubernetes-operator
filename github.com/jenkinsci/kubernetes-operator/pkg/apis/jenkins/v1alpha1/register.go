// NOTE: Boilerplate only.  Ignore this file.

// Package v1alpha1 contains API Schema definitions for the jenkins.io v1alpha1 API group
// +k8s:deepcopy-gen=package,register
// +groupName=jenkins.io
package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/runtime/scheme"
)

const (
	// Kind defines Jenkins CRD kind name
	Kind = "Jenkins"
)

var (
	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: "jenkins.io", Version: "v1alpha1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}
)

// GetObjectKind returns Jenkins object kind
func (in *Jenkins) GetObjectKind() schema.ObjectKind { return in }
