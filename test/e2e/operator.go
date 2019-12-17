package e2e

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"testing"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/events/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var (
	podLogTailLimit       int64 = 15
	kubernetesEventsLimit int64 = 15
	// MUST match the labels in the deployment manifest: deploy/operator.yaml
	operatorPodLabels = map[string]string{
		"name": "jenkins-operator",
	}
)

func getOperatorPod(namespace string) (*v1.Pod, error) {
	listOptions := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(operatorPodLabels).String(),
	}

	podList, err := framework.Global.KubeClient.CoreV1().Pods(namespace).List(listOptions)
	if err != nil {
		return nil, err
	}
	if len(podList.Items) != 1 {
		return nil, fmt.Errorf("expected exactly one pod, got: '%+v'", podList)
	}

	return &podList.Items[0], nil
}

func getOperatorLogs(namespace string) (string, error) {
	pod, err := getOperatorPod(namespace)
	if err != nil {
		return "", err
	}

	logOptions := v1.PodLogOptions{TailLines: &podLogTailLimit}
	req := framework.Global.KubeClient.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &logOptions)
	podLogs, err := req.Stream()
	if err != nil {
		return "", err
	}

	defer func() {
		if podLogs != nil {
			_ = podLogs.Close()
		}
	}()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", err
	}

	logs := buf.String()
	return logs, nil
}

func printOperatorLogs(t *testing.T, namespace string) {
	logs, err := getOperatorLogs(namespace)
	if err != nil {
		t.Errorf("Couldn't get pod logs: %s", err)
	} else {
		t.Logf("Last %d lines of log from operator:\n %s", podLogTailLimit, logs)
	}
}

func getKubernetesEvents(namespace string) ([]v1beta1.Event, error) {
	listOptions := metav1.ListOptions{
		Limit: kubernetesEventsLimit,
	}

	events, err := framework.Global.KubeClient.EventsV1beta1().Events(namespace).List(listOptions)
	if err != nil {
		return nil, err
	}

	sort.SliceStable(events.Items, func(i, j int) bool {
		return events.Items[i].CreationTimestamp.Unix() < events.Items[j].CreationTimestamp.Unix()
	})

	return events.Items, nil
}

func printKubernetesEvents(t *testing.T, namespace string) {
	events, err := getKubernetesEvents(namespace)
	if err != nil {
		t.Errorf("Couldn't get kubernetes events: %s", err)
	} else {
		t.Logf("Last %d events from kubernetes:\n", kubernetesEventsLimit)

		for _, event := range events {
			t.Logf("%+v\n\n", event)
		}
	}
}

func getKubernetesPods(namespace string) ([]v1.Pod, error) {
	pods, err := framework.Global.KubeClient.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return pods.Items, nil
}

func printKubernetesPods(t *testing.T, namespace string) {
	podList, err := getKubernetesPods(namespace)
	if err != nil {
		t.Errorf("Couldn't get kubernetes pods: %s", err)
	}

	for _, pod := range podList {
		t.Logf("%+v\n\n", pod)
	}
}

func cleanupAndShowLogs(t *testing.T, ctx *framework.TestCtx) {
	if t.Failed() {
		t.Error("Test failed. Bellow here you can check logs:")
		t.Fail()
		namespace, err := ctx.GetNamespace()
		if err != nil {
			t.Fatalf("Failed to get '%s' namespace", err)
		}

		t.Logf("Operator logs in '%s' namespace:\n", namespace)
		printOperatorLogs(t, namespace)

		t.Logf("Kubernetes events in '%s' namespace:\n", namespace)
		printKubernetesEvents(t, namespace)

		t.Logf("Pods in '%s' namespace:\n", namespace)
		printKubernetesPods(t, namespace)
	}

	ctx.Cleanup()
}
