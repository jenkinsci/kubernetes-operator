package notifications

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"

	"github.com/pkg/errors"
	"gopkg.in/gomail.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	mailSubject = "Jenkins Operator Notification"
)

// SMTP is Simple Mail Transport Protocol used for sending emails
type SMTP struct {
	k8sClient k8sclient.Client
}

// Send is function for sending notification by SMTP server
func (s SMTP) Send(event Event, config v1alpha2.Notification) error {
	usernameSecret := &corev1.Secret{}
	passwordSecret := &corev1.Secret{}

	usernameSelector := config.SMTP.UsernameSecretKeySelector
	passwordSelector := config.SMTP.PasswordSecretKeySelector

	err := s.k8sClient.Get(context.TODO(), types.NamespacedName{Name: usernameSelector.Name, Namespace: event.Jenkins.Namespace}, usernameSecret)
	if err != nil {
		return err
	}

	err = s.k8sClient.Get(context.TODO(), types.NamespacedName{Name: passwordSelector.Name, Namespace: event.Jenkins.Namespace}, passwordSecret)
	if err != nil {
		return err
	}

	usernameSecretValue := string(usernameSecret.Data[usernameSelector.Key])
	if usernameSecretValue == "" {
		return errors.Errorf("SMTP username is empty in secret '%s/%s[%s]", event.Jenkins.Namespace, usernameSelector.Name, usernameSelector.Key)
	}

	passwordSecretValue := string(passwordSecret.Data[passwordSelector.Key])
	if passwordSecretValue == "" {
		return errors.Errorf("SMTP password is empty in secret '%s/%s[%s]", event.Jenkins.Namespace, passwordSelector.Name, passwordSelector.Key)
	}

	mailer := gomail.NewDialer(config.SMTP.Server, config.SMTP.Port, usernameSecretValue, passwordSecretValue)
	mailer.TLSConfig = &tls.Config{InsecureSkipVerify: config.SMTP.TLSInsecureSkipVerify}

	var statusMessage string

	if config.Verbose {
		message := event.Message + "<ul>"
		for _, msg := range event.MessagesVerbose {
			message = message + "<li>" + msg + "</li>"
		}
		message = message + "</ul>"
		statusMessage = message
	} else {
		statusMessage = event.Message
	}

	htmlMessage := fmt.Sprintf(content, s.getStatusColor(event.LogLevel), notificationTitle(event), statusMessage, event.Jenkins.Name, event.Phase)
	message := gomail.NewMessage()

	message.SetHeader("From", config.SMTP.From)
	message.SetHeader("To", config.SMTP.To)
	message.SetHeader("Subject", mailSubject)
	message.SetBody("text/html", htmlMessage)

	if err := mailer.DialAndSend(message); err != nil {
		return err
	}

	return nil
}

func (s SMTP) getStatusColor(logLevel v1alpha2.NotificationLogLevel) StatusColor {
	switch logLevel {
	case v1alpha2.NotificationLogLevelInfo:
		return "blue"
	case v1alpha2.NotificationLogLevelWarning:
		return "red"
	default:
		return "gray"
	}
}
