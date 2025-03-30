package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/event"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/provider"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/reason"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	testPhase     = event.PhaseUser
	testCrName    = "test-cr"
	testNamespace = "default"
	testReason    = reason.NewPodRestart(
		reason.KubernetesSource,
		[]string{"test-reason-1"},
		[]string{"test-verbose-1"}...,
	)
	testLevel = v1alpha2.NotificationLevelWarning
)

func TestSlack_Send(t *testing.T) {
	fakeClient := fake.NewClientBuilder().Build()
	testURLSelectorKeyName := "test-url-selector"
	testSecretName := "test-secret"

	e := event.Event{
		Jenkins: v1alpha2.Jenkins{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testCrName,
				Namespace: testNamespace,
			},
		},
		Phase:  testPhase,
		Level:  testLevel,
		Reason: testReason,
	}

	slack := Slack{k8sClient: fakeClient, config: v1alpha2.Notification{
		Slack: &v1alpha2.Slack{
			WebHookURLSecretKeySelector: v1alpha2.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: testSecretName,
				},
				Key: testURLSelectorKeyName,
			},
		},
	}}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var message Message
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&message)

		if err != nil {
			t.Fatal(err)
		}

		mainAttachment := message.Attachments[0]

		assert.Equal(t, mainAttachment.Title, provider.NotificationTitle(e))
		for _, field := range mainAttachment.Fields {
			switch field.Title {
			case provider.PhaseFieldName:
				assert.Equal(t, field.Value, string(e.Phase))
			case provider.CrNameFieldName:
				assert.Equal(t, field.Value, e.Jenkins.Name)
			case "":
				message := ""
				for _, msg := range e.Reason.Short() {
					message = message + fmt.Sprintf("\n - %s \n", msg)
				}
				assert.Equal(t, field.Value, message)
			case provider.LevelFieldName:
				assert.Equal(t, field.Value, string(e.Level))
			case provider.NamespaceFieldName:
				assert.Equal(t, field.Value, e.Jenkins.Namespace)
			default:
				t.Errorf("Unexpected field %+v", field)
			}
		}

		assert.Equal(t, mainAttachment.Footer, "")
		assert.Equal(t, mainAttachment.Color, slack.getStatusColor(e.Level))
	}))

	defer server.Close()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testSecretName,
			Namespace: testNamespace,
		},

		Data: map[string][]byte{
			testURLSelectorKeyName: []byte(server.URL),
		},
	}

	err := fakeClient.Create(context.TODO(), secret)
	assert.NoError(t, err)

	err = slack.Send(e)
	assert.NoError(t, err)
}

func TestGenerateMessage(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		crName := "test-jenkins"
		crNamespace := "test-namespace"
		phase := event.PhaseBase
		level := v1alpha2.NotificationLevelInfo
		res := reason.NewUndefined(reason.KubernetesSource, []string{"test-string"}, "test-verbose")

		s := Slack{
			httpClient: http.Client{},
			k8sClient:  fake.NewClientBuilder().Build(),
			config: v1alpha2.Notification{
				Verbose: true,
			},
		}

		e := event.Event{
			Jenkins: v1alpha2.Jenkins{
				ObjectMeta: metav1.ObjectMeta{
					Name:      crName,
					Namespace: crNamespace,
				},
			},
			Phase:  phase,
			Level:  level,
			Reason: res,
		}

		message := s.generateMessage(e)

		var messageStringBuilder strings.Builder
		for _, msg := range e.Reason.Verbose() {
			messageStringBuilder.WriteString(fmt.Sprintf("\n - %s \n", msg))
		}

		mainAttachment := message.Attachments[0]
		messageField := mainAttachment.Fields[0]
		namespaceField := mainAttachment.Fields[1]
		crNameField := mainAttachment.Fields[2]
		phaseField := mainAttachment.Fields[3]

		assert.Equal(t, messageField.Value, messageStringBuilder.String())
		assert.Equal(t, namespaceField.Value, e.Jenkins.Namespace)
		assert.Equal(t, crNameField.Value, e.Jenkins.Name)
		assert.Equal(t, event.Phase(phaseField.Value), e.Phase)
	})

	t.Run("with nils", func(t *testing.T) {
		crName := "nil"
		crNamespace := "nil"
		phase := event.PhaseBase
		level := v1alpha2.NotificationLevelInfo
		res := reason.NewUndefined(reason.KubernetesSource, []string{"nil"}, "nil")

		s := Slack{
			httpClient: http.Client{},
			k8sClient:  fake.NewClientBuilder().Build(),
			config: v1alpha2.Notification{
				Verbose: true,
			},
		}

		e := event.Event{
			Jenkins: v1alpha2.Jenkins{
				ObjectMeta: metav1.ObjectMeta{
					Name:      crName,
					Namespace: crNamespace,
				},
			},
			Phase:  phase,
			Level:  level,
			Reason: res,
		}

		message := s.generateMessage(e)

		var messageStringBuilder strings.Builder
		for _, msg := range e.Reason.Verbose() {
			messageStringBuilder.WriteString(fmt.Sprintf("\n - %s \n", msg))
		}

		mainAttachment := message.Attachments[0]
		messageField := mainAttachment.Fields[0]
		namespaceField := mainAttachment.Fields[1]
		crNameField := mainAttachment.Fields[2]
		phaseField := mainAttachment.Fields[3]

		assert.Equal(t, messageField.Value, messageStringBuilder.String())
		assert.Equal(t, namespaceField.Value, e.Jenkins.Namespace)
		assert.Equal(t, crNameField.Value, e.Jenkins.Name)
		assert.Equal(t, event.Phase(phaseField.Value), e.Phase)
	})

	t.Run("with empty strings", func(t *testing.T) {
		crName := ""
		crNamespace := ""
		phase := event.PhaseBase
		level := v1alpha2.NotificationLevelInfo
		res := reason.NewUndefined(reason.KubernetesSource, []string{""}, "")

		s := Slack{
			httpClient: http.Client{},
			k8sClient:  fake.NewClientBuilder().Build(),
			config: v1alpha2.Notification{
				Verbose: true,
			},
		}

		e := event.Event{
			Jenkins: v1alpha2.Jenkins{
				ObjectMeta: metav1.ObjectMeta{
					Name:      crName,
					Namespace: crNamespace,
				},
			},
			Phase:  phase,
			Level:  level,
			Reason: res,
		}

		message := s.generateMessage(e)

		var messageStringBuilder strings.Builder
		for _, msg := range e.Reason.Verbose() {
			messageStringBuilder.WriteString(fmt.Sprintf("\n - %s \n", msg))
		}

		mainAttachment := message.Attachments[0]
		messageField := mainAttachment.Fields[0]
		namespaceField := mainAttachment.Fields[1]
		crNameField := mainAttachment.Fields[2]
		phaseField := mainAttachment.Fields[3]

		assert.Equal(t, messageField.Value, messageStringBuilder.String())
		assert.Equal(t, namespaceField.Value, e.Jenkins.Namespace)
		assert.Equal(t, crNameField.Value, e.Jenkins.Name)
		assert.Equal(t, event.Phase(phaseField.Value), e.Phase)
	})

	t.Run("with utf-8 characters", func(t *testing.T) {
		crName := "ąśćńółżź"
		crNamespace := "ąśćńółżź"
		phase := event.PhaseBase
		level := v1alpha2.NotificationLevelInfo
		res := reason.NewUndefined(reason.KubernetesSource, []string{"ąśćńółżź"}, "ąśćńółżź")

		s := Slack{
			httpClient: http.Client{},
			k8sClient:  fake.NewClientBuilder().Build(),
			config: v1alpha2.Notification{
				Verbose: true,
			},
		}

		e := event.Event{
			Jenkins: v1alpha2.Jenkins{
				ObjectMeta: metav1.ObjectMeta{
					Name:      crName,
					Namespace: crNamespace,
				},
			},
			Phase:  phase,
			Level:  level,
			Reason: res,
		}

		message := s.generateMessage(e)

		var messageStringBuilder strings.Builder
		for _, msg := range e.Reason.Verbose() {
			messageStringBuilder.WriteString(fmt.Sprintf("\n - %s \n", msg))
		}

		mainAttachment := message.Attachments[0]
		messageField := mainAttachment.Fields[0]
		namespaceField := mainAttachment.Fields[1]
		crNameField := mainAttachment.Fields[2]
		phaseField := mainAttachment.Fields[3]

		assert.Equal(t, messageField.Value, messageStringBuilder.String())
		assert.Equal(t, namespaceField.Value, e.Jenkins.Namespace)
		assert.Equal(t, crNameField.Value, e.Jenkins.Name)
		assert.Equal(t, event.Phase(phaseField.Value), e.Phase)
	})
}
