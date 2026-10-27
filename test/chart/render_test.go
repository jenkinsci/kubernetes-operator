// Package chart holds lightweight `helm template` rendering tests for the
// jenkins-operator chart. They do not need a Kubernetes cluster: they only
// assert that the chart renders the expected manifests for a given set of
// values. They are excluded from the heavyweight e2e suites in test/helm.
package chart

import (
	"os/exec"
	"strings"
	"testing"
)

const chartPath = "../../chart/jenkins-operator"

// helmTemplate renders a single chart template with the given --set overrides
// and returns its stdout. The test is skipped when helm is not installed.
func helmTemplate(t *testing.T, showOnly string, sets ...string) string {
	t.Helper()
	if _, err := exec.LookPath("helm"); err != nil {
		t.Skip("helm binary not found in PATH; skipping chart render test")
	}

	args := []string{"template", "test-release", chartPath, "--show-only", showOnly}
	for _, s := range sets {
		args = append(args, "--set", s)
	}

	out, err := exec.Command("helm", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("helm template %v failed: %v\n%s", args, err, out)
	}
	return string(out)
}

// watchNamespaceValue extracts the value the chart assigned to the
// WATCH_NAMESPACE env var. It returns the raw literal (e.g. `""` or
// `"team-a"`), or "valueFrom" when the operator falls back to the downward API.
func watchNamespaceValue(t *testing.T, rendered string) string {
	t.Helper()
	lines := strings.Split(rendered, "\n")
	for i, l := range lines {
		if !strings.Contains(l, "name: WATCH_NAMESPACE") {
			continue
		}
		for j := i + 1; j < len(lines) && j <= i+4; j++ {
			s := strings.TrimSpace(lines[j])
			switch {
			case strings.HasPrefix(s, "value:"):
				return strings.TrimSpace(strings.TrimPrefix(s, "value:"))
			case strings.HasPrefix(s, "valueFrom:"):
				return "valueFrom"
			}
		}
	}
	t.Fatalf("WATCH_NAMESPACE env var not found in rendered output:\n%s", rendered)
	return ""
}

// hasKindLine reports whether the rendered manifest declares the given kind.
// It matches the whole line so "Role" does not match "RoleBinding".
func hasKindLine(rendered, kind string) bool {
	for _, l := range strings.Split(rendered, "\n") {
		if strings.TrimSpace(l) == "kind: "+kind {
			return true
		}
	}
	return false
}

// TestWatchNamespaceEnv covers how the chart derives the operator's
// WATCH_NAMESPACE env var. An empty value means "watch all namespaces".
func TestWatchNamespaceEnv(t *testing.T) {
	cases := []struct {
		name string
		sets []string
		want string
	}{
		{
			name: "bundled jenkins with default namespace is scoped",
			sets: nil,
			want: `"default"`,
		},
		{
			name: "bundled jenkins with empty namespace watches all namespaces",
			sets: []string{"jenkins.namespace="},
			want: `""`,
		},
		{
			name: "standalone operator with explicit watchNamespace is scoped",
			sets: []string{"jenkins.enabled=false", "operator.watchNamespace=team-a"},
			want: `"team-a"`,
		},
		{
			name: "standalone operator with empty watchNamespace watches all namespaces",
			sets: []string{"jenkins.enabled=false", "operator.watchNamespace="},
			want: `""`,
		},
		{
			name: "standalone operator without watchNamespace defaults to its own namespace",
			sets: []string{"jenkins.enabled=false"},
			want: "valueFrom",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rendered := helmTemplate(t, "templates/operator.yaml", tc.sets...)
			if got := watchNamespaceValue(t, rendered); got != tc.want {
				t.Errorf("WATCH_NAMESPACE = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestOperatorRBACKind verifies the fix's RBAC switch: watching all namespaces
// must produce a ClusterRole/ClusterRoleBinding, while a scoped watch produces a
// namespace-bound Role/RoleBinding.
func TestOperatorRBACKind(t *testing.T) {
	cases := []struct {
		name        string
		sets        []string
		wantCluster bool
	}{
		{
			name:        "scoped namespace uses namespaced Role",
			sets:        nil,
			wantCluster: false,
		},
		{
			name:        "bundled jenkins watching all namespaces uses ClusterRole",
			sets:        []string{"jenkins.namespace="},
			wantCluster: true,
		},
		{
			name:        "standalone operator watching all namespaces uses ClusterRole",
			sets:        []string{"jenkins.enabled=false", "operator.watchNamespace="},
			wantCluster: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			role := helmTemplate(t, "templates/role.yaml", tc.sets...)
			binding := helmTemplate(t, "templates/role_binding.yaml", tc.sets...)

			if tc.wantCluster {
				if !hasKindLine(role, "ClusterRole") {
					t.Errorf("expected a ClusterRole, got:\n%s", role)
				}
				if !hasKindLine(binding, "ClusterRoleBinding") {
					t.Errorf("expected a ClusterRoleBinding, got:\n%s", binding)
				}
			} else {
				if !hasKindLine(role, "Role") {
					t.Errorf("expected a namespaced Role, got:\n%s", role)
				}
				if !hasKindLine(binding, "RoleBinding") {
					t.Errorf("expected a namespaced RoleBinding, got:\n%s", binding)
				}
			}
		})
	}
}
