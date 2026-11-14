package base

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/util/sets"
)

func normalizeVolumesForComparison(volumes []corev1.Volume) []corev1.Volume {
	var normalized []corev1.Volume
	for _, original := range volumes {
		volume := original.DeepCopy()
		switch {
		case volume.ConfigMap != nil && volume.ConfigMap.DefaultMode == nil:
			mode := corev1.ConfigMapVolumeSourceDefaultMode
			volume.ConfigMap.DefaultMode = &mode
		case volume.Secret != nil && volume.Secret.DefaultMode == nil:
			mode := corev1.SecretVolumeSourceDefaultMode
			volume.Secret.DefaultMode = &mode
		case volume.Projected != nil && volume.Projected.DefaultMode == nil:
			mode := corev1.ProjectedVolumeSourceDefaultMode
			volume.Projected.DefaultMode = &mode
		case volume.DownwardAPI != nil && volume.DownwardAPI.DefaultMode == nil:
			mode := corev1.DownwardAPIVolumeSourceDefaultMode
			volume.DownwardAPI.DefaultMode = &mode
		case volume.Ephemeral != nil && volume.Ephemeral.VolumeClaimTemplate != nil && volume.Ephemeral.VolumeClaimTemplate.Spec.VolumeMode == nil:
			mode := corev1.PersistentVolumeFilesystem
			volume.Ephemeral.VolumeClaimTemplate.Spec.VolumeMode = &mode
		}
		normalized = append(normalized, *volume)
	}
	return normalized
}

type podComparisonExclusions struct {
	containers, initContainers, volumes sets.Set[string]
}

func newPodComparisonExclusions() podComparisonExclusions {
	return podComparisonExclusions{containers: sets.New[string](), initContainers: sets.New[string](), volumes: sets.New[string]()}
}

func (ignored podComparisonExclusions) normalize(expected, actual *corev1.Pod) {
	for _, c := range expected.Spec.Containers {
		ignored.containers.Delete(c.Name)
		ignored.initContainers.Delete(c.Name)
	}
	for _, c := range expected.Spec.InitContainers {
		ignored.containers.Delete(c.Name)
		ignored.initContainers.Delete(c.Name)
	}
	for _, v := range expected.Spec.Volumes {
		ignored.volumes.Delete(v.Name)
	}
	actual.Spec.Containers = slices.DeleteFunc(actual.Spec.Containers, func(c corev1.Container) bool { return ignored.containers.Has(c.Name) })
	actual.Spec.InitContainers = slices.DeleteFunc(actual.Spec.InitContainers, func(c corev1.Container) bool { return ignored.initContainers.Has(c.Name) })
	actual.Spec.Volumes = slices.DeleteFunc(actual.Spec.Volumes, func(v corev1.Volume) bool { return ignored.volumes.Has(v.Name) })
	for i := range actual.Spec.Containers {
		actual.Spec.Containers[i].VolumeMounts = slices.DeleteFunc(actual.Spec.Containers[i].VolumeMounts, func(m corev1.VolumeMount) bool { return ignored.volumes.Has(m.Name) })
	}
}

// podComparisonPolicy is a snapshot of user rules; it never changes desired or actual objects.
type podComparisonPolicy struct {
	annotations, labels, env, volumes sets.Set[string]
	volumePrefixes                    []string
	wellKnownInjections               bool
	injectedComponents                podComparisonExclusions
}

func newPodComparisonPolicy(ignore v1alpha2.JenkinsReconciliationIgnore, wellKnown bool) podComparisonPolicy {
	p := podComparisonPolicy{wellKnownInjections: wellKnown, annotations: sets.New(ignore.Annotations...), labels: sets.New(ignore.Labels...), env: sets.New(ignore.Env...), volumes: sets.New(ignore.Volumes...)}
	for _, prefix := range ignore.VolumeNamePrefixes {
		if prefix != "" {
			p.volumePrefixes = append(p.volumePrefixes, prefix)
		}
	}
	if wellKnown {
		p.addWellKnownInjections()
	}
	return p
}

func (p podComparisonPolicy) ignoresVolume(name string) bool {
	if p.volumes.Has(name) {
		return true
	}
	for _, prefix := range p.volumePrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func compareMapIgnoringKeys(expected, actual map[string]string, ignored sets.Set[string]) bool {
	for key, value := range expected {
		if ignored.Has(key) {
			continue
		}
		if actualValue, ok := actual[key]; !ok || actualValue != value {
			return false
		}
	}
	return true
}

func (p podComparisonPolicy) envEqual(expected, actual []corev1.EnvVar) bool {
	if p.env.Len() == 0 {
		return compareEnv(expected, actual)
	}
	filter := func(envs []corev1.EnvVar) []corev1.EnvVar {
		var retained []corev1.EnvVar
		for _, env := range envs {
			if !p.env.Has(env.Name) {
				retained = append(retained, env)
			}
		}
		return retained
	}
	return compareEnv(filter(expected), filter(actual))
}

func (p podComparisonPolicy) mountsEqual(expected, actual corev1.Container) bool {
	if p.volumes.Len() == 0 && len(p.volumePrefixes) == 0 {
		return CompareContainerVolumeMounts(expected, actual)
	}
	filter := func(mounts []corev1.VolumeMount) []corev1.VolumeMount {
		var retained []corev1.VolumeMount
		for _, mount := range mounts {
			if !p.ignoresVolume(mount.Name) {
				retained = append(retained, mount)
			}
		}
		return retained
	}
	expected.VolumeMounts = filter(expected.VolumeMounts)
	actual.VolumeMounts = filter(actual.VolumeMounts)
	return CompareContainerVolumeMounts(expected, actual)
}

func (p podComparisonPolicy) volumesEqual(expected, actual []corev1.Volume, serviceAccount string) bool {
	var retainedExpected, retainedActual []corev1.Volume
	for _, v := range expected {
		if !p.ignoresVolume(v.Name) {
			retainedExpected = append(retainedExpected, v)
		}
	}
	for _, v := range actual {
		if p.ignoresVolume(v.Name) {
			continue
		}
		// filter out service account
		if serviceAccount != "" && strings.HasPrefix(v.Name, serviceAccount) {
			continue
		}
		// hotfix for k8s 1.21 - filter out kube-api-access-<random-suffix>
		const kubeAPIAccessPrefix = "kube-api-access-"
		if strings.HasPrefix(v.Name, kubeAPIAccessPrefix) {
			continue
		}
		retainedActual = append(retainedActual, v)
	}
	return apiequality.Semantic.DeepEqual(normalizeVolumesForComparison(retainedExpected), normalizeVolumesForComparison(retainedActual))
}

type podConfigurationDifference struct{ Field, Container, Name, Message string }

func (d podConfigurationDifference) String() string {
	if d.Message != "" {
		return d.Message
	}
	if d.Container != "" {
		return fmt.Sprintf("%s has changed in container '%s'", d.Field, d.Container)
	}
	if d.Name != "" {
		return fmt.Sprintf("Container '%s' not found in pod configuration", d.Name)
	}
	return "Jenkins pod " + d.Field + " has changed"
}

func desiredPodForComparison(jenkins *v1alpha2.Jenkins) *corev1.Pod {
	desired := resources.NewJenkinsMasterPod(resources.NewResourceObjectMeta(jenkins), jenkins.DeepCopy())
	// Compare declared labels only; generated labels have always been accepted as extra actual metadata.
	desired.Labels = maps.Clone(jenkins.Spec.Master.Labels)
	//FIXME too hacky
	if desired.Spec.SecurityContext == nil {
		desired.Spec.SecurityContext = &corev1.PodSecurityContext{}
	}
	return desired
}

func comparePodConfiguration(expected, actual corev1.Pod, policy podComparisonPolicy) []podConfigurationDifference {
	var differences []podConfigurationDifference
	if policy.wellKnownInjections {
		expected = *expected.DeepCopy()
		actual = *actual.DeepCopy()
		injection := recognizedMeshInjection(&actual)
		differences = injection.validate(&expected)
		policy.injectedComponents = injection.podComparisonExclusions
		policy.injectedComponents.normalize(&expected, &actual)
	}
	add := func(field string, equal bool) {
		if !equal {
			differences = append(differences, podConfigurationDifference{Field: field})
		}
	}
	add("security context", reflect.DeepEqual(expected.Spec.SecurityContext, actual.Spec.SecurityContext))
	add("image pull secrets", compareImagePullSecrets(expected.Spec.ImagePullSecrets, actual.Spec.ImagePullSecrets))
	add("node selector", compareMap(expected.Spec.NodeSelector, actual.Spec.NodeSelector))
	add("labels", compareMapIgnoringKeys(expected.Labels, actual.Labels, policy.labels))
	add("annotations", compareMapIgnoringKeys(expected.Annotations, actual.Annotations, policy.annotations))
	add("volumes", policy.volumesEqual(expected.Spec.Volumes, actual.Spec.Volumes, actual.Spec.ServiceAccountName))
	add("container count", len(expected.Spec.Containers) == len(actual.Spec.Containers))
	add("priorityClassName", expected.Spec.PriorityClassName == actual.Spec.PriorityClassName)
	for _, container := range actual.Spec.Containers {
		var desired *corev1.Container
		for i := range expected.Spec.Containers {
			if expected.Spec.Containers[i].Name == container.Name {
				desired = &expected.Spec.Containers[i]
				break
			}
		}
		if desired == nil {
			differences = append(differences, podConfigurationDifference{Name: container.Name})
			continue
		}
		differences = append(differences, compareContainerConfiguration(*desired, container, policy)...)
	}
	return differences
}
