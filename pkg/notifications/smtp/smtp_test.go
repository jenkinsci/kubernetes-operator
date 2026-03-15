package smtp

import (
	"context"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	smtp "github.com/emersion/go-smtp"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/event"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/reason"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testSMTPPort = 1025

	testFrom = "test@localhost"
	testTo   = "test.to@localhost"

	testSMTPUsername = "username"
	testSMTPPassword = "password"

	testNamespace = "default"
)

type testServer struct {
	event event.Event
}

func (t *testServer) NewSession(_ *smtp.Conn) (smtp.Session, error) {
	return &testSession{event: t.event}, nil
}

type testSession struct {
	event event.Event
}

func (s *testSession) Mail(from string, _ *smtp.MailOptions) error {
	if from != testFrom {
		return assert.AnError
	}
	return nil
}

func (s *testSession) Rcpt(to string, _ *smtp.RcptOptions) error {
	if to != testTo {
		return assert.AnError
	}
	return nil
}

func (s *testSession) Data(r io.Reader) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	content := string(b)

	if !strings.Contains(content, s.event.Jenkins.Name) {
		return assert.AnError
	}

	if !strings.Contains(content, string(s.event.Phase)) {
		return assert.AnError
	}

	return nil
}

func (s *testSession) Reset() {}

func (s *testSession) Logout() error {
	return nil
}

func TestSMTP_Send(t *testing.T) {

	e := event.Event{
		Jenkins: v1alpha2.Jenkins{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-cr",
				Namespace: testNamespace,
			},
		},
		Phase: event.PhaseUser,
		Level: v1alpha2.NotificationLevelWarning,
		Reason: reason.NewPodRestart(
			reason.KubernetesSource,
			[]string{"test-reason"},
			[]string{"test-verbose"}...,
		),
	}

	fakeClient := fake.NewClientBuilder().Build()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			"username": []byte(testSMTPUsername),
			"password": []byte(testSMTPPassword),
		},
	}

	err := fakeClient.Create(context.TODO(), secret)
	assert.NoError(t, err)

	smtpClient := SMTP{
		k8sClient: fakeClient,
		config: v1alpha2.Notification{
			SMTP: &v1alpha2.SMTP{
				Server: "localhost",
				From:   testFrom,
				To:     testTo,
				Port:   testSMTPPort,
				UsernameSecretKeySelector: v1alpha2.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "test-secret",
					},
					Key: "username",
				},
				PasswordSecretKeySelector: v1alpha2.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "test-secret",
					},
					Key: "password",
				},
			},
		},
	}

	ts := &testServer{event: e}

	server := smtp.NewServer(ts)
	server.Addr = ":1025"
	server.Domain = "localhost"
	server.ReadTimeout = 10 * time.Second
	server.WriteTimeout = 10 * time.Second
	server.AllowInsecureAuth = true

	l, err := net.Listen("tcp", "127.0.0.1:1025")
	assert.NoError(t, err)

	go func() {
		_ = server.Serve(l)
	}()

	err = smtpClient.Send(e)
	assert.NoError(t, err)
}

func TestGenerateMessage(t *testing.T) {

	t.Run("happy", func(t *testing.T) {

		crName := "test-jenkins"
		phase := event.PhaseBase
		level := v1alpha2.NotificationLevelInfo

		res := reason.NewUndefined(
			reason.KubernetesSource,
			[]string{"test"},
			[]string{"test-verbose"}...,
		)

		e := event.Event{
			Jenkins: v1alpha2.Jenkins{
				ObjectMeta: metav1.ObjectMeta{
					Name: crName,
				},
			},
			Phase:  phase,
			Level:  level,
			Reason: res,
		}

		s := SMTP{
			k8sClient: fake.NewClientBuilder().Build(),
			config: v1alpha2.Notification{
				LoggingLevel: level,
				SMTP: &v1alpha2.SMTP{
					From: "from@jenkins.local",
					To:   "to@jenkins.local",
				},
			},
		}

		message := s.generateMessage(e)
		assert.NotNil(t, message)
	})
}
