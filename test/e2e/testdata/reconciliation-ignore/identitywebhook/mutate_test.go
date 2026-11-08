package identitywebhook

import (
	"encoding/json"
	"testing"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

func TestIdentityMutation(t *testing.T) {
	for _, profile := range []string{"azure", "azure-prefix", "irsa", "eks-pod-identity", "gcp"} {
		t.Run(profile, func(t *testing.T) {
			pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{ProfileLabel: profile, ManagedKey: "desired"}, Annotations: map[string]string{ManagedKey: "desired"}}, Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "jenkins"}, {Name: "auxiliary"}}, InitContainers: []corev1.Container{{Name: "init"}}}}
			raw, err := json.Marshal(pod)
			if err != nil {
				t.Fatal(err)
			}
			req := admissionv1.AdmissionRequest{UID: types.UID("first"), Operation: admissionv1.Create, Resource: metav1.GroupVersionResource{Version: "v1", Resource: "pods"}, Object: runtime.RawExtension{Raw: raw}}
			response := Mutate(req)
			if !response.Allowed || response.UID != req.UID || response.PatchType == nil {
				t.Fatalf("invalid response: %+v", response)
			}
			var patches []struct {
				Path  string          `json:"path"`
				Value json.RawMessage `json:"value"`
			}
			if err := json.Unmarshal(response.Patch, &patches); err != nil {
				t.Fatal(err)
			}
			var containers []corev1.Container
			var volumes []corev1.Volume
			for _, patch := range patches {
				switch patch.Path {
				case "/spec/containers":
					if err := json.Unmarshal(patch.Value, &containers); err != nil {
						t.Fatal(err)
					}
				case "/spec/volumes":
					if err := json.Unmarshal(patch.Value, &volumes); err != nil {
						t.Fatal(err)
					}
				case "/spec/initContainers":
					t.Fatal("must not mutate init containers")
				}
			}
			if len(volumes) != 1 || volumes[0].Projected == nil {
				t.Fatalf("missing projected identity token: %+v", volumes)
			}
			if len(containers) != 2 {
				t.Fatal("must preserve container count")
			}
			for _, c := range containers {
				if len(c.Env) == 0 || len(c.VolumeMounts) != 1 || c.VolumeMounts[0].Name != volumes[0].Name {
					t.Fatalf("missing injection: %+v", c)
				}
			}
			if profile == "azure-prefix" {
				req.UID = "second"
				if string(Mutate(req).Patch) == string(response.Patch) {
					t.Fatal("Azure prefix profile must vary between admissions")
				}
			}
			req.Operation = admissionv1.Update
			if len(Mutate(req).Patch) != 0 {
				t.Fatal("must not mutate updates")
			}
		})
	}
	pod := corev1.Pod{}
	raw, _ := json.Marshal(pod)
	response := Mutate(admissionv1.AdmissionRequest{Operation: admissionv1.Create, Resource: metav1.GroupVersionResource{Version: "v1", Resource: "pods"}, Object: runtime.RawExtension{Raw: raw}})
	if !response.Allowed || len(response.Patch) != 0 {
		t.Fatal("unselected pod must pass through")
	}
	if response := Mutate(admissionv1.AdmissionRequest{Operation: admissionv1.Create, Resource: metav1.GroupVersionResource{Resource: "pods"}, Object: runtime.RawExtension{Raw: []byte("invalid")}}); response.Allowed {
		t.Fatal("invalid pod must be rejected")
	}
}
