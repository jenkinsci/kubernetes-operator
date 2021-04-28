package e2e

import (
	"context"
	"fmt"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var (
	Cfg       *rest.Config
	K8sClient client.Client
	testEnv   *envtest.Environment

	hostname    *string
	port        *int
	useNodePort *bool
)

func CreateNamespace() *corev1.Namespace {
	ginkgo.By("creating temporary namespace")

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("ns%d", time.Now().Unix()),
		},
	}
	gomega.Expect(K8sClient.Create(context.TODO(), namespace)).Should(gomega.Succeed())
	return namespace
}

func DestroyNamespace(namespace *corev1.Namespace) {
	ginkgo.By("deleting temporary namespace")

	gomega.Expect(K8sClient.Delete(context.TODO(), namespace)).Should(gomega.Succeed())

	gomega.Eventually(func() (bool, error) {
		namespaces := &corev1.NamespaceList{}
		err := K8sClient.List(context.TODO(), namespaces)
		if err != nil {
			return false, err
		}

		exists := false
		for _, namespaceItem := range namespaces.Items {
			if namespaceItem.Name == namespace.Name {
				exists = true
				break
			}
		}

		return !exists, nil
	}, time.Second*120, time.Second).Should(gomega.BeTrue())
}
