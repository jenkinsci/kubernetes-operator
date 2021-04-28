package helm

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"

	"github.com/jenkinsci/kubernetes-operator/test/e2e"
	"github.com/onsi/ginkgo"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/events/v1beta1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	podLogTailLimit       int64 = 15
	kubernetesEventsLimit int64 = 15
	// MUST match the labels in the deployment manifest: deploy/operator.yaml
	operatorPodLabels = map[string]string{
		"name": "jenkins-operator",
	}
)

func getOperatorPod(namespace string) (*corev1.Pod, error) {
	lo := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(operatorPodLabels),
		Namespace:     namespace,
	}
	pods := &corev1.PodList{}
	err := e2e.K8sClient.List(context.TODO(), pods, lo)
	if err != nil {
		return nil, err
	}
	return &pods.Items[0], nil
}

func getOperatorLogs(namespace string) (string, error) {
	pod, err := getOperatorPod(namespace)
	if err != nil {
		return "Operator pod doesn't exist", err
	}
	logOptions := corev1.PodLogOptions{TailLines: &podLogTailLimit}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(e2e.Cfg)
	if err != nil {
		return "", err
	}
	req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &logOptions)
	podLogs, err := req.Stream(context.TODO())
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

func printOperatorLogs(namespace string) {
	_, _ = fmt.Fprintf(ginkgo.GinkgoWriter, "Operator logs in '%s' namespace:\n", namespace)
	logs, err := getOperatorLogs(namespace)
	if err != nil {
		_, _ = fmt.Fprintf(ginkgo.GinkgoWriter, "Couldn't get the operator pod logs: %s", err)
	} else {
		_, _ = fmt.Fprintf(ginkgo.GinkgoWriter, "Last %d lines of log from operator:\n %s", podLogTailLimit, logs)
	}
}

func getKubernetesEvents(namespace string) ([]v1beta1.Event, error) {
	listOptions := &client.ListOptions{
		Limit:     kubernetesEventsLimit,
		Namespace: namespace,
	}

	events := &v1beta1.EventList{}
	err := e2e.K8sClient.List(context.TODO(), events, listOptions)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(events.Items, func(i, j int) bool {
		return events.Items[i].CreationTimestamp.Unix() < events.Items[j].CreationTimestamp.Unix()
	})

	return events.Items, nil
}

func printKubernetesEvents(namespace string) {
	_, _ = fmt.Fprintf(ginkgo.GinkgoWriter, "Kubernetes events in '%s' namespace:\n", namespace)
	events, err := getKubernetesEvents(namespace)
	if err != nil {
		_, _ = fmt.Fprintf(ginkgo.GinkgoWriter, "Couldn't get kubernetes events: %s", err)
	} else {
		_, _ = fmt.Fprintf(ginkgo.GinkgoWriter, "Last %d events from kubernetes:\n", kubernetesEventsLimit)

		for _, event := range events {
			_, _ = fmt.Fprintf(ginkgo.GinkgoWriter, "%+v\n\n", event)
		}
	}
}

func printKubernetesPods(namespace string) {
	_, _ = fmt.Fprintf(ginkgo.GinkgoWriter, "All pods in '%s' namespace:\n", namespace)

	pod, err := getOperatorPod(namespace)
	if err == nil {
		_, _ = fmt.Fprintf(ginkgo.GinkgoWriter, "%+v\n\n", pod)
	}
}

func showLogsIfTestHasFailed(failed bool, namespace string) {
	if failed {
		_, _ = fmt.Fprintf(ginkgo.GinkgoWriter, "Test failed. Bellow here you can check logs:")

		printKubernetesEvents(namespace)
		printKubernetesPods(namespace)
		printOperatorLogs(namespace)
	}
}
