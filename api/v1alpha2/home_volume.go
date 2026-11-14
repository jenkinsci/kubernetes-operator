package v1alpha2

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// JenkinsHomeVolume selects Jenkins home storage for the lifetime of its Pod.
// +kubebuilder:validation:XValidation:rule="has(self.emptyDir) != has(self.ephemeral)",message="exactly one of emptyDir or ephemeral must be set"
type JenkinsHomeVolume struct {
	// +optional
	EmptyDir *corev1.EmptyDirVolumeSource `json:"emptyDir,omitempty"`
	// Ephemeral contains the spec of the Pod-owned PVC.
	// +optional
	Ephemeral *corev1.PersistentVolumeClaimSpec `json:"ephemeral,omitempty"`
}

// Validate checks home storage independently of webhook availability.
func (home *JenkinsHomeVolume) Validate() field.ErrorList {
	if home == nil {
		return nil
	}
	path := field.NewPath("spec", "master", "homeVolume")
	if (home.EmptyDir == nil) == (home.Ephemeral == nil) {
		return field.ErrorList{field.Invalid(path, home, "exactly one of emptyDir or ephemeral must be set")}
	}
	if home.Ephemeral == nil {
		return nil
	}
	claim := home.Ephemeral
	path = path.Child("ephemeral")
	var errs field.ErrorList
	size, exists := claim.Resources.Requests[corev1.ResourceStorage]
	if !exists || size.Sign() <= 0 {
		errs = append(errs, field.Invalid(path.Child("resources", "requests", "storage"), size.String(), "must be greater than zero"))
	}
	if len(claim.AccessModes) == 0 {
		errs = append(errs, field.Required(path.Child("accessModes"), "at least one access mode is required"))
	}
	modes := sets.New[corev1.PersistentVolumeAccessMode]()
	for i, mode := range claim.AccessModes {
		switch mode {
		case corev1.ReadWriteOnce, corev1.ReadOnlyMany, corev1.ReadWriteMany, corev1.ReadWriteOncePod:
		default:
			errs = append(errs, field.NotSupported(path.Child("accessModes").Index(i), mode, []string{"ReadWriteOnce", "ReadOnlyMany", "ReadWriteMany", "ReadWriteOncePod"}))
		}
		if modes.Has(mode) {
			errs = append(errs, field.Duplicate(path.Child("accessModes").Index(i), mode))
		}
		modes.Insert(mode)
	}
	if claim.VolumeMode != nil && *claim.VolumeMode != corev1.PersistentVolumeFilesystem {
		errs = append(errs, field.NotSupported(path.Child("volumeMode"), *claim.VolumeMode, []string{"Filesystem"}))
	}
	return errs
}
