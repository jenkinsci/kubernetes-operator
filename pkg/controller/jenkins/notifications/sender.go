package notifications

import (
	"fmt"
	"net/http"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/event"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"

	"github.com/pkg/errors"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	infoTitleText         = "Jenkins Operator reconciliation info"
	warnTitleText         = "Jenkins Operator reconciliation warning"
	messageFieldName      = "Message"
	loggingLevelFieldName = "Logging Level"
	crNameFieldName       = "CR Name"
	phaseFieldName        = "Phase"
	namespaceFieldName    = "Namespace"
)

const (
	// PhaseBase is core configuration of Jenkins provided by the Operator
	PhaseBase Phase = "base"

	// PhaseUser is user-defined configuration of Jenkins
	PhaseUser Phase = "user"

	// PhaseUnknown is untraceable type of configuration
	PhaseUnknown Phase = "unknown"
)

var (
	testPhase          = PhaseUser
	testCrName         = "test-cr"
	testNamespace      = "default"
	testMessage        = "test-message"
	testMessageVerbose = []string{"detail-test-message"}
	testLoggingLevel   = v1alpha2.NotificationLogLevelWarning

	client = http.Client{}
)

// Phase defines the type of configuration
type Phase string

// StatusColor is useful for better UX
type StatusColor string

// LoggingLevel is type for selecting different logging levels
type LoggingLevel string

// Event contains event details which will be sent as a notification
type Event struct {
	Jenkins         v1alpha2.Jenkins
	Phase           Phase
	LogLevel        v1alpha2.NotificationLogLevel
	Message         string
	MessagesVerbose []string
}

type service interface {
	Send(event Event, notificationConfig v1alpha2.Notification) error
}

// Listen listens for incoming events and send it as notifications
func Listen(events chan Event, k8sEvent event.Recorder, k8sClient k8sclient.Client) {
	for evt := range events {
		logger := log.Log.WithValues("cr", evt.Jenkins.Name)
		for _, notificationConfig := range evt.Jenkins.Spec.Notifications {
			var err error
			var svc service

			if notificationConfig.Slack != nil {
				svc = Slack{k8sClient: k8sClient}
			} else if notificationConfig.Teams != nil {
				svc = Teams{k8sClient: k8sClient}
			} else if notificationConfig.Mailgun != nil {
				svc = MailGun{k8sClient: k8sClient}
			} else if notificationConfig.SMTP != nil {
				svc = SMTP{k8sClient: k8sClient}
			} else {
				logger.V(log.VWarn).Info(fmt.Sprintf("Unknown notification service `%+v`", notificationConfig))
				continue
			}

			go func(notificationConfig v1alpha2.Notification) {
				err = notify(svc, evt, notificationConfig)

				if err != nil {
					if log.Debug {
						logger.Error(nil, fmt.Sprintf("%+v", errors.WithMessage(err, fmt.Sprintf("failed to send notification '%s'", notificationConfig.Name))))
					} else {
						logger.Error(nil, fmt.Sprintf("%s", errors.WithMessage(err, fmt.Sprintf("failed to send notification '%s'", notificationConfig.Name))))
					}
				}
			}(notificationConfig)
		}
		k8sEvent.Emit(&evt.Jenkins, logLevelEventType(evt.LogLevel), "NotificationSent", evt.Message)
	}
}

func logLevelEventType(level v1alpha2.NotificationLogLevel) event.Type {
	switch level {
	case v1alpha2.NotificationLogLevelWarning:
		return event.TypeWarning
	case v1alpha2.NotificationLogLevelInfo:
		return event.TypeNormal
	default:
		return event.TypeNormal
	}
}

func notify(svc service, event Event, manifest v1alpha2.Notification) error {
	if event.LogLevel == v1alpha2.NotificationLogLevelInfo && manifest.LoggingLevel == v1alpha2.NotificationLogLevelWarning {
		return nil
	}
	return svc.Send(event, manifest)
}

func notificationTitle(event Event) string {
	if event.LogLevel == v1alpha2.NotificationLogLevelInfo {
		return infoTitleText
	} else if event.LogLevel == v1alpha2.NotificationLogLevelWarning {
		return warnTitleText
	} else {
		return ""
	}
}
