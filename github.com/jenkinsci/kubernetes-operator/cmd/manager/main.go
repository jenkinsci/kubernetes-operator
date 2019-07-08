package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/constants"
	"github.com/jenkinsci/kubernetes-operator/pkg/event"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"
	"github.com/jenkinsci/kubernetes-operator/version"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/operator-framework/operator-sdk/pkg/leader"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"github.com/operator-framework/operator-sdk/pkg/metrics"
	"github.com/operator-framework/operator-sdk/pkg/restmapper"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

// Change below variables to serve metrics on different host or port.
var (
	metricsHost       = "0.0.0.0"
	metricsPort int32 = 8383
)

//var log = logf.Log.WithName("cmd")

func printInfo() {
	log.Log.Info(fmt.Sprintf("Version: %s", version.Version))
	log.Log.Info(fmt.Sprintf("Git commit: %s", version.GitCommit))
	log.Log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Log.Info(fmt.Sprintf("operator-sdk Version: %v", sdkVersion.Version))
}

func main() {
	// Add the zap logger flag set to the CLI. The flag set must
	// be added before calling pflag.Parse().
	pflag.CommandLine.AddFlagSet(zap.FlagSet())

	// Add flags registered by imported packages (e.g. glog and
	// controller-runtime)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	minikube := pflag.Bool("minikube", false, "Use minikube as a Kubernetes platform")
	local := pflag.Bool("local", false, "Run operator locally")
	debug := pflag.Bool("debug", false, "Set log level to debug")
	pflag.Parse()

	log.SetupLogger(*debug)
	printInfo()

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		fatal(errors.Wrap(err, "failed to get watch namespace"), *debug)
	}
	log.Log.Info(fmt.Sprintf("Watch namespace: %v", namespace))

	// get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		fatal(errors.Wrap(err, "failed to get config"), *debug)
	}

	ctx := context.TODO()

	// Become the leader before proceeding
	err = leader.Become(ctx, "jenkins-operator-lock")
	if err != nil {
		fatal(errors.Wrap(err, "failed to become leader"), *debug)
	}

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{
		Namespace:          namespace,
		MapperProvider:     restmapper.NewDynamicRESTMapper,
		MetricsBindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
	})
	if err != nil {
		fatal(errors.Wrap(err, "failed to create manager"), *debug)
	}

	log.Log.Info("Registering Components.")

	// setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		fatal(errors.Wrap(err, "failed to setup scheme"), *debug)
	}

	// setup events
	events, err := event.New(cfg, constants.OperatorName)
	if err != nil {
		fatal(errors.Wrap(err, "failed to create manager"), *debug)
	}

	clientSet, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		fatal(errors.Wrap(err, "failed to create Kubernetes client set"), *debug)
	}

	// setup Jenkins controller
	if err := jenkins.Add(mgr, *local, *minikube, events, *clientSet, *cfg); err != nil {
		fatal(errors.Wrap(err, "failed to setup controllers"), *debug)
	}

	// Create Service object to expose the metrics port.
	_, err = metrics.ExposeMetricsPort(ctx, metricsPort)
	if err != nil {
		log.Log.Info(err.Error())
	}

	log.Log.Info("Starting the Cmd.")

	// start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		fatal(errors.Wrap(err, "failed to start cmd"), *debug)
	}
}

func fatal(err error, debug bool) {
	if debug {
		log.Log.Error(nil, fmt.Sprintf("%+v", err))
	} else {
		log.Log.Error(nil, fmt.Sprintf("%s", err))
	}
	os.Exit(-1)
}
