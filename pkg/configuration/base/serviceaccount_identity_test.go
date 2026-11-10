package base

import (
	"context"
	"testing"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestNativeGKEIdentityServiceAccount(t *testing.T) {
	j := comparisonJenkins()
	j.Namespace = "identity"
	j.Spec.ServiceAccount.Annotations = map[string]string{"iam.gke.io/gcp-service-account": "test@example.iam.gserviceaccount.com"}
	j.Spec.Master.NodeSelector = map[string]string{"iam.gke.io/gke-metadata-server-enabled": "true"}
	testScheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(testScheme))
	require.NoError(t, v1alpha2.AddToScheme(testScheme))
	k8sClient := fake.NewClientBuilder().WithScheme(testScheme).Build()
	r := New(configuration.Configuration{Jenkins: j, Client: k8sClient, Scheme: testScheme}, jenkinsclient.JenkinsAPIConnectionSettings{})
	meta := resources.NewResourceObjectMeta(j)
	require.NoError(t, r.createServiceAccount(meta))
	account := &corev1.ServiceAccount{}
	key := types.NamespacedName{Name: meta.Name, Namespace: meta.Namespace}
	require.NoError(t, k8sClient.Get(context.Background(), key, account))
	require.Equal(t, j.Spec.ServiceAccount.Annotations, account.Annotations)
	account.Annotations["external"] = "preserved"
	require.NoError(t, k8sClient.Update(context.Background(), account))
	j.Spec.ServiceAccount.Annotations["iam.gke.io/gcp-service-account"] = "updated@example.iam.gserviceaccount.com"
	require.NoError(t, r.createServiceAccount(meta))
	require.NoError(t, k8sClient.Get(context.Background(), key, account))
	require.Equal(t, "preserved", account.Annotations["external"])
	require.Equal(t, j.Spec.ServiceAccount.Annotations["iam.gke.io/gcp-service-account"], account.Annotations["iam.gke.io/gcp-service-account"])
	pod := desiredPodForComparison(j)
	require.Equal(t, meta.Name, pod.Spec.ServiceAccountName)
	require.Equal(t, j.Spec.Master.NodeSelector, pod.Spec.NodeSelector)
	require.Empty(t, comparePodConfiguration(*pod, *pod.DeepCopy(), newPodComparisonPolicy(j.Spec.Reconciliation.Ignore, false)))
}
