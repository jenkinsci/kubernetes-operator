// NOTE: Boilerplate only.  Ignore this file.

// Package v1alpha3 contains API Schema definitions for the jenkins v1alpha3 API group
// +k8s:deepcopy-gen=package,register
// +groupName=jenkins.io
package v1alpha3

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: "jenkins.io", Version: "v1alpha3"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}
)
