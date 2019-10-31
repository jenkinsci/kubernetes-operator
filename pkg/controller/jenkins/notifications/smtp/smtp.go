package smtp

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/notifications/event"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/notifications/provider"

	"github.com/pkg/errors"
	"gopkg.in/gomail.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	mailSubject = "Jenkins Operator Notification"
	content     = `
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN"
        "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html>
<head></head>
<body>
		<h1 style="background-color: %s; color: white; padding: 3px 10px;">%s</h1>
		<h3>%s</h3>
		<table>
			<tr>
				<td><b>CR name:</b></td>
				<td>%s</td>
			</tr>
			<tr>
				<td><b>Phase:</b></td>
				<td>%s</td>
			</tr>
		</table>
		<h6 style="font-size: 11px; color: grey; margin-top: 15px;">Powered by Jenkins Operator <3</h6>
</body>
</html>`
)

// SMTP is Simple Mail Transport Protocol used for sending emails
type SMTP struct {
	k8sClient k8sclient.Client
	config    v1alpha2.Notification
}

// New returns instance of SMTP
func New(k8sClient k8sclient.Client, config v1alpha2.Notification) *SMTP {
	return &SMTP{k8sClient: k8sClient, config: config}
}

// Send is function for sending notification by SMTP server
func (s SMTP) Send(e event.Event) error {
	usernameSecret := &corev1.Secret{}
	passwordSecret := &corev1.Secret{}

	usernameSelector := s.config.SMTP.UsernameSecretKeySelector
	passwordSelector := s.config.SMTP.PasswordSecretKeySelector

	err := s.k8sClient.Get(context.TODO(), types.NamespacedName{Name: usernameSelector.Name, Namespace: e.Jenkins.Namespace}, usernameSecret)
	if err != nil {
		return err
	}

	err = s.k8sClient.Get(context.TODO(), types.NamespacedName{Name: passwordSelector.Name, Namespace: e.Jenkins.Namespace}, passwordSecret)
	if err != nil {
		return err
	}

	usernameSecretValue := string(usernameSecret.Data[usernameSelector.Key])
	if usernameSecretValue == "" {
		return errors.Errorf("SMTP username is empty in secret '%s/%s[%s]", e.Jenkins.Namespace, usernameSelector.Name, usernameSelector.Key)
	}

	passwordSecretValue := string(passwordSecret.Data[passwordSelector.Key])
	if passwordSecretValue == "" {
		return errors.Errorf("SMTP password is empty in secret '%s/%s[%s]", e.Jenkins.Namespace, passwordSelector.Name, passwordSelector.Key)
	}

	mailer := gomail.NewDialer(s.config.SMTP.Server, s.config.SMTP.Port, usernameSecretValue, passwordSecretValue)
	mailer.TLSConfig = &tls.Config{InsecureSkipVerify: s.config.SMTP.TLSInsecureSkipVerify}

	var statusMessage string

	reasons := strings.TrimRight(strings.Join(e.Reason.Short(), "</li><li>"), "<li>")
	statusMessage = "<ul><li>"
	statusMessage = statusMessage + reasons
	statusMessage = statusMessage + "</ul>"

	// TODO: add verbose

	htmlMessage := fmt.Sprintf(content, s.getStatusColor(e.Level), provider.NotificationTitle(e), statusMessage, e.Jenkins.Name, e.Phase)
	message := gomail.NewMessage()

	message.SetHeader("From", s.config.SMTP.From)
	message.SetHeader("To", s.config.SMTP.To)
	message.SetHeader("Subject", mailSubject)
	message.SetBody("text/html", htmlMessage)

	if err := mailer.DialAndSend(message); err != nil {
		return err
	}

	return nil
}

func (s SMTP) getStatusColor(logLevel v1alpha2.NotificationLevel) event.StatusColor {
	switch logLevel {
	case v1alpha2.NotificationLevelInfo:
		return "blue"
	case v1alpha2.NotificationLevelWarning:
		return "red"
	default:
		return "gray"
	}
}
