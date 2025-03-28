package e2e

import (
	"flag"
	"path/filepath"
	"testing"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	controllers "github.com/jenkinsci/kubernetes-operator/internal/controller"
	jenkinsClient "github.com/jenkinsci/kubernetes-operator/pkg/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/constants"
	"github.com/jenkinsci/kubernetes-operator/pkg/event"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications"
	e "github.com/jenkinsci/kubernetes-operator/pkg/notifications/event"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

func init() {
	hostname = flag.String("jenkins-api-hostname", "", "Hostname or IP of Jenkins API. It can be service name, node IP or localhost.")
	port = flag.Int("jenkins-api-port", 0, "The port on which Jenkins API is running. Note: If you want to use nodePort don't set this setting and --jenkins-api-use-nodeport must be true.")
	useNodePort = flag.Bool("jenkins-api-use-nodeport", false, "Connect to Jenkins API using the service nodePort instead of service port. If you want to set this as true - don't set --jenkins-api-port.")
}

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(false)))

	By("bootstrapping test environment")
	useExistingCluster := true
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "config", "crd", "bases")},
		//BinaryAssetsDirectory: path.Join("..", "..", "testbin", "bin"),
		UseExistingCluster: &useExistingCluster,
	}

	var err error
	Cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(Cfg).NotTo(BeNil())

	err = v1alpha2.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	// setup manager
	k8sManager, err := ctrl.NewManager(Cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).NotTo(HaveOccurred())

	// setup controller
	clientSet, err := kubernetes.NewForConfig(Cfg)
	Expect(err).NotTo(HaveOccurred())

	// setup events
	events, err := event.New(Cfg, constants.OperatorName)
	Expect(err).NotTo(HaveOccurred())
	notificationEvents := make(chan e.Event)
	go notifications.Listen(notificationEvents, events, K8sClient)

	jenkinsAPIConnectionSettings := jenkinsClient.JenkinsAPIConnectionSettings{
		Hostname:    *hostname,
		UseNodePort: *useNodePort,
		Port:        *port,
	}

	err = (&controllers.JenkinsReconciler{
		Client:                       k8sManager.GetClient(),
		Scheme:                       k8sManager.GetScheme(),
		JenkinsAPIConnectionSettings: jenkinsAPIConnectionSettings,
		ClientSet:                    *clientSet,
		Config:                       *Cfg,
		NotificationEvents:           &notificationEvents,
		KubernetesClusterDomain:      "cluster.local",
	}).SetupWithManager(k8sManager)
	Expect(err).NotTo(HaveOccurred())

	go func() {
		err = k8sManager.Start(ctrl.SetupSignalHandler())
		Expect(err).NotTo(HaveOccurred())
	}()

	K8sClient = k8sManager.GetClient()
	Expect(K8sClient).NotTo(BeNil())
	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
