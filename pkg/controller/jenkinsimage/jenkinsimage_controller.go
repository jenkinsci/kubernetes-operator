package jenkinsimage

import (
	"context"

	jenkinsv1alpha2 "github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"

	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_jenkinsimage")

// Add creates a new JenkinsImage Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	r := &ReconcileJenkinsImage{client: mgr.GetClient(), scheme: mgr.GetScheme()}
	return add(mgr, r)
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("jenkinsimage-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource JenkinsImage
	eventHandlerForObject := &handler.EnqueueRequestForObject{}
	src := &source.Kind{Type: &jenkinsv1alpha2.JenkinsImage{}}
	err = c.Watch(src, eventHandlerForObject)
	if err != nil {
		return errors.WithStack(err)
	}

	// Watch for changes to secondary resource Pods and requeue the owner JenkinsImage
	eventHandlerForOwner := &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &jenkinsv1alpha2.JenkinsImage{},
	}
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, eventHandlerForOwner)
	if err != nil {
		return errors.WithStack(err)
	}
	// Watch for changes to secondary ConfigMap and requeue the owner JenkinsImage
	err = c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, eventHandlerForOwner)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// blank assignment to verify that ReconcileJenkinsImage implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileJenkinsImage{}

// ReconcileJenkinsImage reconciles a JenkinsImage object
type ReconcileJenkinsImage struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a JenkinsImage object and makes changes based on the state read
// and what is in the JenkinsImage.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileJenkinsImage) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling JenkinsImage")

	// Fetch the JenkinsImage instance
	instance := &jenkinsv1alpha2.JenkinsImage{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Define a new ConfigMap containing the Dockerfile used to build the image
	dockerfile := resources.NewDockerfileConfigMap(instance)
	// Set JenkinsImage instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, dockerfile, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this ConfigMap already exists
	foundConfigMap := &corev1.ConfigMap{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: dockerfile.Name, Namespace: dockerfile.Namespace}, foundConfigMap)
	if err != nil && apierrors.IsNotFound(err) {
		reqLogger.Info("Creating a new ConfigMap", "ConfigMap.Namespace", dockerfile.Namespace, "ConfigMap.Name", dockerfile.Name)
		err = r.client.Create(context.TODO(), dockerfile)
		if err != nil {
			return reconcile.Result{}, err
		}
		// ConfigMap created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}
	// ConfigMap already exists - don't requeue
	reqLogger.Info("Skip reconcile: ConfigMap already exists", "ConfigMap.Namespace", foundConfigMap.Namespace, "ConfigMap.Name", foundConfigMap.Name)

	// Define a new Pod object
	pod := resources.NewBuilderPod(instance)
	// Set JenkinsImage instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, pod, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Pod already exists
	foundPod := &corev1.Pod{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, foundPod)
	if err != nil && apierrors.IsNotFound(err) {
		reqLogger.Info("Creating a new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
		err = r.client.Create(context.TODO(), pod)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Pod created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}
	// Pod already exists - don't requeue
	reqLogger.Info("Skip reconcile: Pod already exists", "Pod.Namespace", foundPod.Namespace, "Pod.Name", foundPod.Name)

	return reconcile.Result{}, nil
}
