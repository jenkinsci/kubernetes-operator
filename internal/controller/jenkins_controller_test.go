package controllers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/event"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type failingReadClient struct {
	client.Client
	err error
}

func (c failingReadClient) Get(context.Context, client.ObjectKey, client.Object, ...client.GetOption) error {
	return c.err
}

func TestReconcileReturnsRetryableErrors(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
	}{
		{"API unavailable", errors.New("API unavailable")},
		{"conflict", apierrors.NewConflict(schema.GroupResource{Group: "jenkins.io", Resource: "jenkins"}, "example", errors.New("changed"))},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := ctrl.Request{NamespacedName: types.NamespacedName{Name: t.Name(), Namespace: "default"}}
			t.Cleanup(func() { delete(reconcileErrors, request.NamespacedName) })
			r := &JenkinsReconciler{Client: failingReadClient{err: test.err}}
			result, err := r.Reconcile(context.Background(), request)
			require.ErrorIs(t, err, test.err)
			require.Equal(t, ctrl.Result{}, result)
			if apierrors.IsConflict(test.err) {
				require.NotContains(t, reconcileErrors, request.NamespacedName)
			}
		})
	}
}

func TestReconcileDefaultingSchedulesPositiveJitter(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, v1alpha2.AddToScheme(s))
	for range 100 {
		jenkins := &v1alpha2.Jenkins{ObjectMeta: metav1.ObjectMeta{Name: "example", Namespace: "default"}}
		r := &JenkinsReconciler{Client: fake.NewClientBuilder().WithScheme(s).WithObjects(jenkins).Build()}
		result, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(jenkins)})
		require.NoError(t, err)
		require.GreaterOrEqual(t, result.RequeueAfter, time.Second)
		require.LessOrEqual(t, result.RequeueAfter, 9*time.Second)
		require.Zero(t, result.RequeueAfter%time.Second)
	}
}

func TestReconcileDeletedJenkinsDoesNotRetry(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, v1alpha2.AddToScheme(s))
	r := &JenkinsReconciler{Client: fake.NewClientBuilder().WithScheme(s).Build()}
	result, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: "deleted", Namespace: "default"}})
	require.NoError(t, err)
	require.Equal(t, ctrl.Result{}, result)
}

func TestReconcileStopsAfterTenIdenticalErrors(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, v1alpha2.AddToScheme(s))
	jenkins := &v1alpha2.Jenkins{ObjectMeta: metav1.ObjectMeta{Name: t.Name(), Namespace: "default"}, Spec: v1alpha2.JenkinsSpec{Master: v1alpha2.JenkinsMaster{Containers: []v1alpha2.Container{{Name: "invalid"}}}}}
	events := make(chan event.Event, 1)
	r := &JenkinsReconciler{Client: fake.NewClientBuilder().WithScheme(s).WithObjects(jenkins).Build(), NotificationEvents: &events}
	request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(jenkins)}
	t.Cleanup(func() { delete(reconcileErrors, request.NamespacedName) })
	for range 9 {
		result, err := r.Reconcile(context.Background(), request)
		require.Error(t, err)
		require.Equal(t, ctrl.Result{}, result)
		require.Empty(t, events)
	}
	result, err := r.Reconcile(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, ctrl.Result{}, result)
	require.Len(t, events, 1)
	require.Equal(t, jenkins.Name, (<-events).Jenkins.Name)
}

func TestReconcileTerminalGroovyFailureNotifiesWithoutRetry(t *testing.T) {
	jenkins := &v1alpha2.Jenkins{ObjectMeta: metav1.ObjectMeta{Name: t.Name(), Namespace: "default"}}
	events := make(chan event.Event, 1)
	r := &JenkinsReconciler{NotificationEvents: &events}
	request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(jenkins)}
	t.Cleanup(func() { delete(reconcileErrors, request.NamespacedName) })
	failure := &jenkinsclient.GroovyScriptExecutionFailed{ConfigurationType: "base-groovy", Source: "configmap", Name: "invalid.groovy", Logs: "syntax error"}
	result, err := r.handleReconcileResult(request, ctrl.Result{}, jenkins, failure)
	require.NoError(t, err)
	require.Equal(t, ctrl.Result{}, result)
	require.Len(t, events, 1)
	notification := <-events
	require.Equal(t, jenkins.Name, notification.Jenkins.Name)
	require.Equal(t, event.PhaseBase, notification.Phase)
	require.Equal(t, v1alpha2.NotificationLevelWarning, notification.Level)
}

func TestReconcileReadFailureLimitDoesNotPanic(t *testing.T) {
	request := ctrl.Request{NamespacedName: types.NamespacedName{Name: t.Name(), Namespace: "default"}}
	t.Cleanup(func() { delete(reconcileErrors, request.NamespacedName) })
	r := &JenkinsReconciler{Client: failingReadClient{err: errors.New("API unavailable")}}
	for range 9 {
		_, err := r.Reconcile(context.Background(), request)
		require.Error(t, err)
	}
	result, err := r.Reconcile(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, ctrl.Result{}, result)
}

func TestReconcilePreservesDelayAndDiscardsItOnError(t *testing.T) {
	request := ctrl.Request{NamespacedName: types.NamespacedName{Name: t.Name(), Namespace: "default"}}
	t.Cleanup(func() { delete(reconcileErrors, request.NamespacedName) })
	r := &JenkinsReconciler{}
	pending := ctrl.Result{RequeueAfter: 5 * time.Second}
	result, err := r.handleReconcileResult(request, pending, nil, nil)
	require.NoError(t, err)
	require.Equal(t, pending, result)
	failure := errors.New("retryable")
	result, err = r.handleReconcileResult(request, pending, nil, failure)
	require.ErrorIs(t, err, failure)
	require.Equal(t, ctrl.Result{}, result)
}

func TestReconcileFailureHistoryIsIsolatedByNamespace(t *testing.T) {
	r := &JenkinsReconciler{}
	failure := errors.New("API unavailable")
	first := ctrl.Request{NamespacedName: types.NamespacedName{Name: t.Name(), Namespace: "first"}}
	second := ctrl.Request{NamespacedName: types.NamespacedName{Name: t.Name(), Namespace: "second"}}
	t.Cleanup(func() {
		delete(reconcileErrors, first.NamespacedName)
		delete(reconcileErrors, second.NamespacedName)
	})
	for range 9 {
		_, err := r.handleReconcileResult(first, ctrl.Result{}, nil, failure)
		require.ErrorIs(t, err, failure)
	}
	_, err := r.handleReconcileResult(second, ctrl.Result{}, nil, failure)
	require.ErrorIs(t, err, failure)
}

func TestReconcileSuccessResetsFailureHistory(t *testing.T) {
	for _, test := range []struct {
		name   string
		result ctrl.Result
	}{
		{name: "completed"},
		{name: "polling", result: ctrl.Result{RequeueAfter: 5 * time.Second}},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := ctrl.Request{NamespacedName: types.NamespacedName{Name: t.Name(), Namespace: "default"}}
			t.Cleanup(func() { delete(reconcileErrors, request.NamespacedName) })
			r := &JenkinsReconciler{}
			failure := errors.New("API unavailable")
			for range 9 {
				_, err := r.handleReconcileResult(request, ctrl.Result{}, nil, failure)
				require.ErrorIs(t, err, failure)
			}
			result, err := r.handleReconcileResult(request, test.result, nil, nil)
			require.NoError(t, err)
			require.Equal(t, test.result, result)
			for range 9 {
				_, err = r.handleReconcileResult(request, ctrl.Result{}, nil, failure)
				require.ErrorIs(t, err, failure)
			}
		})
	}
}

func TestReconcileDeletionResetsFailureHistory(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, v1alpha2.AddToScheme(s))
	r := &JenkinsReconciler{Client: fake.NewClientBuilder().WithScheme(s).Build()}
	request := ctrl.Request{NamespacedName: types.NamespacedName{Name: t.Name(), Namespace: "default"}}
	t.Cleanup(func() { delete(reconcileErrors, request.NamespacedName) })
	failure := errors.New("API unavailable")
	for range 9 {
		_, err := r.handleReconcileResult(request, ctrl.Result{}, nil, failure)
		require.ErrorIs(t, err, failure)
	}
	result, err := r.Reconcile(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, ctrl.Result{}, result)
	_, err = r.handleReconcileResult(request, ctrl.Result{}, nil, failure)
	require.ErrorIs(t, err, failure)
}
