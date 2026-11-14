//go:build integration

package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestNamespaceCache(t *testing.T) {
	environment := &envtest.Environment{}
	cfg, err := environment.Start()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, environment.Stop()) })

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, rbacv1.AddToScheme(scheme))
	direct, err := client.New(cfg, client.Options{Scheme: scheme})
	require.NoError(t, err)
	ctx := context.Background()
	for _, namespace := range []string{"first", "second"} {
		require.NoError(t, direct.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}))
		require.NoError(t, direct.Create(ctx, &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "jenkins", Namespace: namespace},
			RoleRef:    rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: "Role", Name: "jenkins"},
		}))
	}

	for _, tc := range []struct {
		name    string
		options cache.Options
		scoped  bool
	}{
		{name: "all namespaces", options: buildCacheOptions("")},
		{name: "nil namespace map", options: cache.Options{}},
		{name: "single namespace", options: buildCacheOptions("first"), scoped: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.options.Scheme = scheme
			reader, err := cache.New(cfg, tc.options)
			require.NoError(t, err)
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			_, err = reader.GetInformer(ctx, &rbacv1.RoleBinding{}, cache.BlockUntilSynced(false))
			require.NoError(t, err)
			stopped := make(chan error, 1)
			go func() { stopped <- reader.Start(ctx) }()
			t.Cleanup(func() {
				cancel()
				require.NoError(t, <-stopped)
			})
			require.True(t, reader.WaitForCacheSync(ctx))

			for _, namespace := range []string{"first", "second"} {
				binding := &rbacv1.RoleBinding{}
				getErr := reader.Get(ctx, client.ObjectKey{Name: "jenkins", Namespace: namespace}, binding)
				bindings := &rbacv1.RoleBindingList{}
				listErr := reader.List(ctx, bindings, client.InNamespace(namespace))
				if tc.scoped && namespace == "second" {
					require.ErrorContains(t, getErr, "unknown namespace")
					require.ErrorContains(t, listErr, "unknown namespace")
					continue
				}
				require.NoError(t, getErr)
				require.Equal(t, namespace, binding.Namespace)
				require.NoError(t, listErr)
				require.Len(t, bindings.Items, 1)
				require.Equal(t, namespace, bindings.Items[0].Namespace)
			}
		})
	}
}
