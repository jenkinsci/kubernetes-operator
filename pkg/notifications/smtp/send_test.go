package smtp

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/event"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/reason"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type smtpDelivery struct {
	authenticated bool
	from, to      string
	message       []byte
	err           error
}

func startSMTPPeer(t *testing.T) (int, <-chan smtpDelivery) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	require.NoError(t, listener.(*net.TCPListener).SetDeadline(time.Now().Add(5*time.Second)))
	delivery := make(chan smtpDelivery, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := listener.Accept()
		if err != nil {
			delivery <- smtpDelivery{err: err}
			return
		}
		defer func() { _ = conn.Close() }()
		if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
			delivery <- smtpDelivery{err: err}
			return
		}
		delivery <- receiveSMTP(conn)
	}()
	t.Cleanup(func() {
		_ = listener.Close()
		select {
		case <-done:
		case <-time.After(6 * time.Second):
			t.Error("SMTP peer did not stop")
		}
	})
	return listener.Addr().(*net.TCPAddr).Port, delivery
}

func receiveSMTP(conn net.Conn) smtpDelivery {
	var result smtpDelivery
	peer := textproto.NewConn(conn)
	if err := peer.PrintfLine("220 localhost ESMTP"); err != nil {
		result.err = err
		return result
	}
	for {
		line, err := peer.ReadLine()
		if err != nil {
			result.err = err
			return result
		}
		command, argument, _ := strings.Cut(line, " ")
		response := "250 OK"
		switch command {
		case "EHLO":
			response = "250-localhost\r\n250 AUTH PLAIN"
		case "AUTH":
			mechanism, encoded, _ := strings.Cut(argument, " ")
			credentials, err := base64.StdEncoding.DecodeString(encoded)
			if err != nil || mechanism != "PLAIN" || string(credentials) != "\x00smtp-user\x00smtp-password" {
				response = "535 Authentication failed"
			} else {
				result.authenticated = true
				response = "235 Authentication successful"
			}
		case "MAIL":
			if !result.authenticated {
				response = "530 Authentication required"
			} else {
				result.from = strings.TrimPrefix(argument, "FROM:")
			}
		case "RCPT":
			if !result.authenticated {
				response = "530 Authentication required"
			} else {
				result.to = strings.TrimPrefix(argument, "TO:")
			}
		case "DATA":
			if !result.authenticated {
				response = "530 Authentication required"
				break
			}
			if err := peer.PrintfLine("354 End with a dot"); err != nil {
				result.err = err
				return result
			}
			result.message, err = peer.ReadDotBytes()
			if err != nil {
				result.err = err
				return result
			}
		case "QUIT":
			result.err = peer.PrintfLine("221 Bye")
			return result
		default:
			response = "500 Unexpected command"
		}
		if err := peer.PrintfLine("%s", response); err != nil {
			result.err = err
			return result
		}
	}
}

func smtpFixture() (SMTP, event.Event, *corev1.Secret) {
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "credentials", Namespace: "notifications"}, Data: map[string][]byte{"username": []byte("smtp-user"), "password": []byte("smtp-password")}}
	notification := v1alpha2.Notification{SMTP: &v1alpha2.SMTP{
		Server: "127.0.0.1", From: "operator@example.com", To: "recipient@example.com",
		UsernameSecretKeySelector: v1alpha2.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "credentials"}, Key: "username"},
		PasswordSecretKeySelector: v1alpha2.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "credentials"}, Key: "password"},
	}}
	e := event.Event{Jenkins: v1alpha2.Jenkins{ObjectMeta: metav1.ObjectMeta{Name: "test-jenkins", Namespace: "notifications"}}, Phase: event.PhaseUser, Level: v1alpha2.NotificationLevelWarning, Reason: reason.NewUndefined(reason.KubernetesSource, []string{"short reason"}, "verbose reason")}
	return SMTP{config: notification}, e, secret
}

func TestSMTP_Send(t *testing.T) {
	port, delivery := startSMTPPeer(t)
	sender, e, secret := smtpFixture()
	sender.config.SMTP.Port = port
	sender.k8sClient = fake.NewClientBuilder().WithObjects(secret).Build()
	require.NoError(t, sender.Send(e))
	received := <-delivery
	require.NoError(t, received.err)
	require.True(t, received.authenticated)
	require.Equal(t, "<operator@example.com>", received.from)
	require.Equal(t, "<recipient@example.com>", received.to)
	message, err := mail.ReadMessage(bytes.NewReader(received.message))
	require.NoError(t, err)
	require.Equal(t, "operator@example.com", message.Header.Get("From"))
	require.Equal(t, "recipient@example.com", message.Header.Get("To"))
	require.Equal(t, "Jenkins Operator Notification", message.Header.Get("Subject"))
	contentType, _, err := mime.ParseMediaType(message.Header.Get("Content-Type"))
	require.NoError(t, err)
	require.Equal(t, "text/html", contentType)
	var body io.Reader = message.Body
	if message.Header.Get("Content-Transfer-Encoding") == "quoted-printable" {
		body = quotedprintable.NewReader(body)
	}
	content, err := io.ReadAll(body)
	require.NoError(t, err)
	require.Contains(t, string(content), "test-jenkins")
	require.Contains(t, string(content), string(event.PhaseUser))
	require.Contains(t, string(content), "short reason")
	require.NotContains(t, string(content), "verbose reason")
}

func TestSMTP_SendRejectsInvalidCredentials(t *testing.T) {
	port, delivery := startSMTPPeer(t)
	sender, e, secret := smtpFixture()
	secret.Data["password"] = []byte("wrong-password")
	sender.config.SMTP.Port = port
	sender.k8sClient = fake.NewClientBuilder().WithObjects(secret).Build()
	require.ErrorContains(t, sender.Send(e), "535")
	received := <-delivery
	require.NoError(t, received.err)
	require.False(t, received.authenticated)
	require.Empty(t, received.message)
}

func TestSMTP_SendRequiresCredentials(t *testing.T) {
	for _, tt := range []struct {
		name, field   string
		missingSecret bool
	}{
		{"missing username secret", "username", true}, {"missing password secret", "password", true},
		{"empty username", "username", false}, {"empty password", "password", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sender, e, secret := smtpFixture()
			if tt.missingSecret {
				if tt.field == "username" {
					sender.config.SMTP.UsernameSecretKeySelector.Name = "missing"
				} else {
					sender.config.SMTP.PasswordSecretKeySelector.Name = "missing"
				}
			} else {
				delete(secret.Data, tt.field)
			}
			sender.k8sClient = fake.NewClientBuilder().WithObjects(secret).Build()
			err := sender.Send(e)
			if tt.missingSecret {
				require.ErrorContains(t, err, `secrets "missing" not found`)
			} else {
				require.ErrorContains(t, err, fmt.Sprintf("SMTP %s is empty", tt.field))
			}
		})
	}
}
