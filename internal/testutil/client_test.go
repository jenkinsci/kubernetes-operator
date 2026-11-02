package testutil_test

import (
	"context"
	"testing"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/internal/testutil"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestClientPreservesBackupCounters(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, v1alpha2.AddToScheme(scheme))
	jenkins := &v1alpha2.Jenkins{ObjectMeta: metav1.ObjectMeta{Name: "example", Namespace: "default"}}
	k8sClient := testutil.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(jenkins).Build()
	ctx := context.Background()
	require.NoError(t, k8sClient.Create(ctx, jenkins))
	jenkins.Spec.Backup.Interval = 30
	require.NoError(t, k8sClient.Update(ctx, jenkins))
	jenkins.Status.LastBackup = 9007199254740993
	jenkins.Status.PendingBackup = 9007199254740994
	require.NoError(t, k8sClient.Status().Update(ctx, jenkins))
	got := &v1alpha2.Jenkins{}
	require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(jenkins), got))
	require.Equal(t, uint64(30), got.Spec.Backup.Interval)
	require.Equal(t, uint64(9007199254740993), got.Status.LastBackup)
	require.Equal(t, uint64(9007199254740994), got.Status.PendingBackup)
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "credentials", Namespace: "default"}, Data: map[string][]byte{"token": []byte("secret")}}
	require.NoError(t, k8sClient.Create(ctx, secret))
	fetched := &corev1.Secret{}
	require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(secret), fetched))
	require.Equal(t, []byte("secret"), fetched.Data["token"])
}
