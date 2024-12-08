package notifications

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	k8sevent "github.com/jenkinsci/kubernetes-operator/pkg/event"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/event"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/mailgun"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/msteams"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/slack"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/smtp"

	"github.com/pkg/errors"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Provider is the communication service handler.
type Provider interface {
	Send(event event.Event) error
}

// Listen listens for incoming events and send it as notifications.
func Listen(events chan event.Event, k8sEvent k8sevent.Recorder, k8sClient k8sclient.Client) {
	httpClient := http.Client{}
	for e := range events {
		logger := log.Log.WithValues("cr", e.Jenkins.Name)

		if !e.Reason.HasMessages() {
			logger.V(log.VWarn).Info("Reason has no messages, this should not happen")
			continue // skip empty messages
		}

		k8sEvent.Emit(&e.Jenkins,
			eventLevelToKubernetesEventType(e.Level),
			k8sevent.Reason(reflect.TypeOf(e.Reason).Name()),
			strings.Join(e.Reason.Short(), "; "),
		)

		for _, notificationConfig := range e.Jenkins.Spec.Notifications {
			var err error
			var provider Provider
			switch {
			case notificationConfig.Slack != nil:
				provider = slack.New(k8sClient, notificationConfig, httpClient)
			case notificationConfig.Teams != nil:
				provider = msteams.New(k8sClient, notificationConfig, httpClient)
			case notificationConfig.Mailgun != nil:
				provider = mailgun.New(k8sClient, notificationConfig)
			case notificationConfig.SMTP != nil:
				provider = smtp.New(k8sClient, notificationConfig)
			default:
				logger.V(log.VWarn).Info(fmt.Sprintf("Unknown notification service `%+v`", notificationConfig))
				continue
			}

			isInfoEvent := e.Level == v1alpha2.NotificationLevelInfo
			wantsWarning := notificationConfig.LoggingLevel == v1alpha2.NotificationLevelWarning
			if isInfoEvent && wantsWarning {
				continue // skip the event
			}

			go func(notificationConfig v1alpha2.Notification, e event.Event) {
				err = provider.Send(e)
				if err != nil {
					wrapped := errors.WithMessage(err,
						fmt.Sprintf("failed to send notification '%s'", notificationConfig.Name))
					if log.Debug {
						logger.Error(nil, fmt.Sprintf("%+v", wrapped))
					} else {
						logger.Error(nil, fmt.Sprintf("%s", wrapped))
					}
				}
			}(notificationConfig, e)
		}
	}
}

func eventLevelToKubernetesEventType(level v1alpha2.NotificationLevel) k8sevent.Type {
	switch level {
	case v1alpha2.NotificationLevelWarning:
		return k8sevent.TypeWarning
	case v1alpha2.NotificationLevelInfo:
		return k8sevent.TypeNormal
	default:
		return k8sevent.TypeNormal
	}
}
