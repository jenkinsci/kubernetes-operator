package v1alpha2

import (
	"encoding/json"
	"reflect"
	"testing"

	"k8s.io/utils/ptr"
)

func TestReconciliationSerializationAndDeepCopy(t *testing.T) {
	original := &Jenkins{Spec: JenkinsSpec{Reconciliation: JenkinsReconciliation{IgnoreWellKnownInjections: ptr.To(true), Ignore: JenkinsReconciliationIgnore{
		Annotations: []string{"annotation"}, Labels: []string{"label"}, Env: []string{"ENV"}, Volumes: []string{"token"}, VolumeNamePrefixes: []string{"azure-workload-identity-reserved-"},
	}}}}
	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Jenkins
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(original.Spec.Reconciliation, decoded.Spec.Reconciliation) {
		t.Fatalf("policy lost in JSON round trip: %s", encoded)
	}
	copy := original.DeepCopy()
	*copy.Spec.Reconciliation.IgnoreWellKnownInjections = false
	copy.Spec.Reconciliation.Ignore.Annotations[0] = "other"
	copy.Spec.Reconciliation.Ignore.Labels[0] = "other"
	copy.Spec.Reconciliation.Ignore.Env[0] = "OTHER"
	copy.Spec.Reconciliation.Ignore.Volumes[0] = "other"
	copy.Spec.Reconciliation.Ignore.VolumeNamePrefixes[0] = "other-"
	if !reflect.DeepEqual(original.Spec.Reconciliation, decoded.Spec.Reconciliation) {
		t.Fatal("deepcopy aliased policy slices")
	}
	for _, input := range []string{`{}`, `{"reconciliation":{}}`, `{"reconciliation":{"ignore":{}}}`, `{"reconciliation":{"ignore":{"env":[],"volumeNamePrefixes":[]}}}`} {
		var spec JenkinsSpec
		if err := json.Unmarshal([]byte(input), &spec); err != nil {
			t.Fatal(err)
		}
		if spec.Reconciliation.IgnoreWellKnownInjections != nil {
			t.Fatalf("omitted switch must remain unset: %s", input)
		}
		if len(spec.Reconciliation.Ignore.Env) != 0 || len(spec.Reconciliation.Ignore.VolumeNamePrefixes) != 0 {
			t.Fatalf("unexpected policy: %s", input)
		}
	}
	for _, enabled := range []bool{true, false} {
		spec := JenkinsSpec{Reconciliation: JenkinsReconciliation{IgnoreWellKnownInjections: ptr.To(enabled)}}
		encoded, err := json.Marshal(spec)
		if err != nil {
			t.Fatal(err)
		}
		var decoded JenkinsSpec
		if err := json.Unmarshal(encoded, &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded.Reconciliation.IgnoreWellKnownInjections == nil || *decoded.Reconciliation.IgnoreWellKnownInjections != enabled {
			t.Fatalf("explicit switch lost in round trip: %s", encoded)
		}
	}
}
