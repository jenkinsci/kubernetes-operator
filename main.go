/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"os"
	r "runtime"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/controllers"
	"github.com/jenkinsci/kubernetes-operator/pkg/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/constants"
	"github.com/jenkinsci/kubernetes-operator/pkg/event"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications"
	e "github.com/jenkinsci/kubernetes-operator/pkg/notifications/event"
	"github.com/jenkinsci/kubernetes-operator/version"

	routev1 "github.com/openshift/api/route/v1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

var (
	metricsHost       = "0.0.0.0"
	metricsPort int32 = 8383
	scheme            = runtime.NewScheme()
	logger            = logf.Log.WithName("cmd")
)

func printInfo() {
	logger.Info(fmt.Sprintf("Version: %s", version.Version))
	logger.Info(fmt.Sprintf("Git commit: %s", version.GitCommit))
	logger.Info(fmt.Sprintf("Go Version: %s", r.Version()))
	logger.Info(fmt.Sprintf("Go OS/Arch: %s/%s", r.GOOS, r.GOARCH))
}

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha2.AddToScheme(scheme))
	utilruntime.Must(routev1.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string

	isRunningInCluster, err := resources.IsRunningInCluster()
	if err != nil {
		fatal(errors.Wrap(err, "failed to determine if operator is running in cluster"), true)
	}

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", isRunningInCluster, "Enable leader election for controller manager. "+
		"Enabling this will ensure there is only one active controller manager.")
	hostname := flag.String("jenkins-api-hostname", "", "Hostname or IP of Jenkins API. It can be service name, node IP or localhost.")
	port := flag.Int("jenkins-api-port", 0, "The port on which Jenkins API is running. Note: If you want to use nodePort don't set this setting and --jenkins-api-use-nodeport must be true.")
	useNodePort := flag.Bool("jenkins-api-use-nodeport", false, "Connect to Jenkins API using the service nodePort instead of service port. If you want to set this as true - don't set --jenkins-api-port.")
	kubernetesClusterDomain := flag.String("cluster-domain", "cluster.local", "Use custom domain name instead of 'cluster.local'.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	debug := &opts.Development
	log.Debug = *debug
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	printInfo()

	namespace, found := os.LookupEnv("WATCH_NAMESPACE")
	if !found {
		fatal(errors.New("failed to get watch namespace, please set up WATCH_NAMESPACE environment variable"), *debug)
	}
	logger.Info(fmt.Sprintf("Watch namespace: %v", namespace))

	// get a config to talk to the API server
	cfg, err := config.GetConfig()
	if err != nil {
		fatal(errors.Wrap(err, "failed to get config"), *debug)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     fmt.Sprintf("%s:%d", metricsHost, metricsPort),
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "c674355f.jenkins.io",
		Namespace:              namespace,
	})
	if err != nil {
		fatal(errors.Wrap(err, "unable to start manager"), *debug)
	}

	// setup events
	events, err := event.New(cfg, constants.OperatorName)
	if err != nil {
		fatal(errors.Wrap(err, "failed to setup events"), *debug)
	}

	// setup controller
	clientSet, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		fatal(errors.Wrap(err, "failed to create Kubernetes client set"), *debug)
	}

	if resources.IsRouteAPIAvailable(clientSet) {
		logger.Info("Route API found: Route creation will be performed")
	}
	notificationEvents := make(chan e.Event)
	go notifications.Listen(notificationEvents, events, mgr.GetClient())

	// validate jenkins API connection
	jenkinsAPIConnectionSettings := client.JenkinsAPIConnectionSettings{Hostname: *hostname, Port: *port, UseNodePort: *useNodePort}
	if err := jenkinsAPIConnectionSettings.Validate(); err != nil {
		fatal(errors.Wrap(err, "invalid command line parameters"), *debug)
	}

	// validate kubernetes cluster domain
	if *kubernetesClusterDomain == "" {
		fatal(errors.Wrap(err, "Kubernetes cluster domain can't be empty"), *debug)
	}

	if err = (&controllers.JenkinsReconciler{
		Client:                       mgr.GetClient(),
		Scheme:                       mgr.GetScheme(),
		JenkinsAPIConnectionSettings: jenkinsAPIConnectionSettings,
		ClientSet:                    *clientSet,
		Config:                       *cfg,
		NotificationEvents:           &notificationEvents,
		KubernetesClusterDomain:      *kubernetesClusterDomain,
	}).SetupWithManager(mgr); err != nil {
		fatal(errors.Wrap(err, "unable to create Jenkins controller"), *debug)
	}

	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("health", healthz.Ping); err != nil {
		fatal(errors.Wrap(err, "unable to set up health check"), *debug)
	}
	if err := mgr.AddReadyzCheck("check", healthz.Ping); err != nil {
		fatal(errors.Wrap(err, "unable to set up ready check"), *debug)
	}

	logger.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		fatal(errors.Wrap(err, "problem running manager"), *debug)
	}
}

func fatal(err error, debug bool) {
	if debug {
		logger.Error(nil, fmt.Sprintf("%+v", err))
	} else {
		logger.Error(nil, fmt.Sprintf("%s", err))
	}
	os.Exit(1)
}
