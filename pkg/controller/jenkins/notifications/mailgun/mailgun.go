package mailgun

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/notifications/event"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/notifications/provider"

	"github.com/mailgun/mailgun-go/v3"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	infoColor    = "blue"
	warningColor = "red"
	defaultColor = "gray"

	content = `
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

// MailGun is a sending emails notification service
type MailGun struct {
	k8sClient k8sclient.Client
	config    v1alpha2.Notification
}

// New returns instance of MailGun
func New(k8sClient k8sclient.Client, config v1alpha2.Notification) *MailGun {
	return &MailGun{k8sClient: k8sClient, config: config}
}

func (m MailGun) getStatusColor(logLevel v1alpha2.NotificationLevel) event.StatusColor {
	switch logLevel {
	case v1alpha2.NotificationLevelInfo:
		return infoColor
	case v1alpha2.NotificationLevelWarning:
		return warningColor
	default:
		return defaultColor
	}
}

func (m MailGun) generateMessage(event event.Event) string {
	var statusMessage strings.Builder
	var reasons string

	if m.config.Verbose {
		reasons = strings.TrimRight(strings.Join(event.Reason.Verbose(), "</li><li>"), "<li>")
	} else {
		reasons = strings.TrimRight(strings.Join(event.Reason.Short(), "</li><li>"), "<li>")
	}

	statusMessage.WriteString("<ul><li>")
	statusMessage.WriteString(reasons)
	statusMessage.WriteString("</ul>")

	statusColor := m.getStatusColor(event.Level)
	messageTitle := provider.NotificationTitle(event)
	message := statusMessage.String()
	crName := event.Jenkins.Name
	phase := event.Phase

	return fmt.Sprintf(content, statusColor, messageTitle, message, crName, phase)
}

// Send is function for sending directly to API
func (m MailGun) Send(event event.Event) error {
	secret := &corev1.Secret{}
	selector := m.config.Mailgun.APIKeySecretKeySelector

	err := m.k8sClient.Get(context.TODO(),
		types.NamespacedName{Name: selector.Name, Namespace: event.Jenkins.Namespace}, secret)
	if err != nil {
		return errors.WithStack(err)
	}

	secretValue := string(secret.Data[selector.Key])
	if secretValue == "" {
		return errors.Errorf("Mailgun API secret is empty in secret '%s/%s[%s]",
			event.Jenkins.Namespace, selector.Name, selector.Key)
	}

	mg := mailgun.NewMailgun(m.config.Mailgun.Domain, secretValue)
	from := fmt.Sprintf("Jenkins Operator Notifier <%s>", m.config.Mailgun.From)
	subject := provider.NotificationTitle(event)
	recipient := m.config.Mailgun.Recipient

	msg := mg.NewMessage(from, subject, "", recipient)
	msg.SetHtml(m.generateMessage(event))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	_, _, err = mg.Send(ctx, msg)

	return err
}
