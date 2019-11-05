package msteams

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/notifications/event"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/notifications/provider"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/notifications/reason"

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

func TestTeams_Send(t *testing.T) {
	fakeClient := fake.NewFakeClient()
	testURLSelectorKeyName := "test-url-selector"
	testSecretName := "test-secret"

	event := event.Event{
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
	teams := Teams{k8sClient: fakeClient, config: v1alpha2.Notification{
		Teams: &v1alpha2.MicrosoftTeams{
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

		assert.Equal(t, message.Title, provider.NotificationTitle(event))
		assert.Equal(t, message.ThemeColor, teams.getStatusColor(event.Level))

		mainSection := message.Sections[0]

		reason := strings.Join(event.Reason.Short(), "\n\n - ")

		assert.Equal(t, mainSection.Text, reason)

		for _, fact := range mainSection.Facts {
			switch fact.Name {
			case provider.PhaseFieldName:
				assert.Equal(t, fact.Value, string(event.Phase))
			case provider.CrNameFieldName:
				assert.Equal(t, fact.Value, event.Jenkins.Name)
			case provider.MessageFieldName:
				assert.Equal(t, fact.Value, reason)
			case provider.LevelFieldName:
				assert.Equal(t, fact.Value, string(event.Level))
			case provider.NamespaceFieldName:
				assert.Equal(t, fact.Value, event.Jenkins.Namespace)
			default:
				t.Errorf("Found unexpected '%+v' fact", fact)
			}
		}
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

	err = teams.Send(event)
	assert.NoError(t, err)
}

func TestGenerateMessages(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		crName := "test-jenkins"
		crNamespace := "test-namespace"
		phase := event.PhaseBase
		level := v1alpha2.NotificationLevelInfo
		res := reason.NewUndefined(reason.KubernetesSource, []string{"test-string"}, "test-verbose")

		s := Teams{
			httpClient: http.Client{},
			k8sClient:  fake.NewFakeClient(),
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

		msg := strings.Join(e.Reason.Verbose(), "\n\n - ")

		mainSection := message.Sections[0]

		crNameFact := mainSection.Facts[0]
		namespaceFact := mainSection.Facts[1]
		phaseFact := mainSection.Facts[2]

		assert.Equal(t, mainSection.Text, msg)
		assert.Equal(t, crNameFact.Value, e.Jenkins.Name)
		assert.Equal(t, namespaceFact.Value, e.Jenkins.Namespace)
		assert.Equal(t, event.Phase(phaseFact.Value), e.Phase)
	})

	t.Run("with nils", func(t *testing.T) {
		crName := "nil"
		crNamespace := "nil"
		phase := event.PhaseBase
		level := v1alpha2.NotificationLevelInfo
		res := reason.NewUndefined(reason.KubernetesSource, []string{"nil"}, "nil")

		s := Teams{
			httpClient: http.Client{},
			k8sClient:  fake.NewFakeClient(),
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

		msg := strings.Join(e.Reason.Verbose(), "\n\n - ")

		mainSection := message.Sections[0]

		crNameFact := mainSection.Facts[0]
		namespaceFact := mainSection.Facts[1]
		phaseFact := mainSection.Facts[2]

		assert.Equal(t, mainSection.Text, msg)
		assert.Equal(t, crNameFact.Value, e.Jenkins.Name)
		assert.Equal(t, namespaceFact.Value, e.Jenkins.Namespace)
		assert.Equal(t, event.Phase(phaseFact.Value), e.Phase)
	})

	t.Run("with empty strings", func(t *testing.T) {
		crName := ""
		crNamespace := ""
		phase := event.PhaseBase
		level := v1alpha2.NotificationLevelInfo
		res := reason.NewUndefined(reason.KubernetesSource, []string{""}, "")

		s := Teams{
			httpClient: http.Client{},
			k8sClient:  fake.NewFakeClient(),
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

		msg := strings.Join(e.Reason.Verbose(), "\n\n - ")

		mainSection := message.Sections[0]

		crNameFact := mainSection.Facts[0]
		namespaceFact := mainSection.Facts[1]
		phaseFact := mainSection.Facts[2]

		assert.Equal(t, mainSection.Text, msg)
		assert.Equal(t, crNameFact.Value, e.Jenkins.Name)
		assert.Equal(t, namespaceFact.Value, e.Jenkins.Namespace)
		assert.Equal(t, event.Phase(phaseFact.Value), e.Phase)
	})

	t.Run("with utf-8 characters", func(t *testing.T) {
		crName := "ąśćńółżź"
		crNamespace := "ąśćńółżź"
		phase := event.PhaseBase
		level := v1alpha2.NotificationLevelInfo
		res := reason.NewUndefined(reason.KubernetesSource, []string{"ąśćńółżź"}, "ąśćńółżź")
		s := Teams{
			httpClient: http.Client{},
			k8sClient:  fake.NewFakeClient(),
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

		msg := strings.Join(e.Reason.Verbose(), "\n\n - ")

		mainSection := message.Sections[0]

		crNameFact := mainSection.Facts[0]
		namespaceFact := mainSection.Facts[1]
		phaseFact := mainSection.Facts[2]

		assert.Equal(t, mainSection.Text, msg)
		assert.Equal(t, crNameFact.Value, e.Jenkins.Name)
		assert.Equal(t, namespaceFact.Value, e.Jenkins.Namespace)
		assert.Equal(t, event.Phase(phaseFact.Value), e.Phase)
	})
}
