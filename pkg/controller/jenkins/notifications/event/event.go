package event

import (
	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/notifications/reason"
)

// Phase defines the context where notification has been generated: base or user.
type Phase string

// StatusColor is useful for better UX
type StatusColor string

// LoggingLevel is type for selecting different logging levels
type LoggingLevel string

// Event contains event details which will be sent as a notification
type Event struct {
	Jenkins v1alpha2.Jenkins
	Phase   Phase
	Level   v1alpha2.NotificationLevel
	Reason  reason.Reason
}

const (
	// PhaseBase is core configuration of Jenkins provided by the Operator
	PhaseBase Phase = "base"

	// PhaseUser is user-defined configuration of Jenkins
	PhaseUser Phase = "user"

	// PhaseUnknown is untraceable type of configuration
	PhaseUnknown Phase = "unknown"
)
