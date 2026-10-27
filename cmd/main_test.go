package main

import (
	"testing"
)

func TestBuildCacheOptions(t *testing.T) {
	t.Run("empty namespace watches all namespaces with a cluster-wide cache", func(t *testing.T) {
		opts := buildCacheOptions("")

		// DefaultNamespaces must stay nil so controller-runtime builds a single
		// cluster-wide cache that can serve namespace-scoped List calls.
		if opts.DefaultNamespaces != nil {
			t.Errorf("expected DefaultNamespaces to be nil for all-namespaces mode, got %v", opts.DefaultNamespaces)
		}
	})

	t.Run("single namespace scopes the cache to that namespace", func(t *testing.T) {
		opts := buildCacheOptions("jenkins")

		if len(opts.DefaultNamespaces) != 1 {
			t.Fatalf("expected exactly one namespace in DefaultNamespaces, got %d", len(opts.DefaultNamespaces))
		}
		if _, ok := opts.DefaultNamespaces["jenkins"]; !ok {
			t.Errorf("expected DefaultNamespaces to be keyed by %q, got %v", "jenkins", opts.DefaultNamespaces)
		}
	})

	t.Run("does not use the empty-string key that breaks namespace-scoped lists", func(t *testing.T) {
		// Regression guard: DefaultNamespaces{"": {}} builds a multiNamespaceCache
		// keyed only by "" and fails with "unknown namespace for the cache".
		opts := buildCacheOptions("")
		if _, ok := opts.DefaultNamespaces[""]; ok {
			t.Error("DefaultNamespaces must not contain the empty-string key in all-namespaces mode")
		}
	})

	// A scoped namespace should carry no extra restrictions (default cache.Config).
	t.Run("scoped namespace uses a default cache.Config", func(t *testing.T) {
		opts := buildCacheOptions("team-a")
		got, ok := opts.DefaultNamespaces["team-a"]
		if !ok {
			t.Fatalf("expected namespace %q to be present", "team-a")
		}
		// cache.Config is not comparable (it holds a func), so assert on its fields.
		if got.LabelSelector != nil || got.FieldSelector != nil {
			t.Errorf("expected no selectors on scoped namespace config, got %+v", got)
		}
	})
}
