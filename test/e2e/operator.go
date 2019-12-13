package e2e

import (
	"bytes"
	"io"
	"testing"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var (
	podLogTailLimit int64 = 15
	// MUST match the labels in the deployment manifest: deploy/operator.yaml
	operatorPodLabels = map[string]string{
		"name": "jenkins-operator",
	}
)

func getOperatorPod(t *testing.T, namespace string) *v1.Pod {
	listOptions := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(operatorPodLabels).String(),
	}

	podList, err := framework.Global.KubeClient.CoreV1().Pods(namespace).List(listOptions)
	if err != nil {
		t.Fatal(err)
	}
	if len(podList.Items) != 1 {
		t.Fatalf("Expected exactly one pod, got: '%+v'", podList)
	}

	return &podList.Items[0]
}

func getOperatorLogs(pod v1.Pod) (string, error) {
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

func failTestAndPrintLogs(t *testing.T, namespace string, err error) {
	operatorPod := getOperatorPod(t, namespace)
	logs, logsErr := getOperatorLogs(*operatorPod)
	if logsErr != nil {
		t.Errorf("Couldn't get pod logs: %s", logsErr)
	} else {
		t.Logf("Last %d lines of log from operator:\n %s", podLogTailLimit, logs)
	}

	t.Fatal(err)
}
