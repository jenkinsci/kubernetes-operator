package provider

import (
	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/notifications/event"
)

const (
	// InfoTitleText is info header of notification
	InfoTitleText = "Jenkins Operator reconciliation info"

	// WarnTitleText is warning header of notification
	WarnTitleText = "Jenkins Operator reconciliation warning"

	// MessageFieldName is field title for message content
	MessageFieldName = "Message"

	// LevelFieldName is field title for level enum
	LevelFieldName = "Level"

	// CrNameFieldName is field title for CR Name string
	CrNameFieldName = "CR Name"

	// PhaseFieldName is field title for Phase enum
	PhaseFieldName = "Phase"

	// NamespaceFieldName is field title for Namespace string
	NamespaceFieldName = "Namespace"
)

// NotificationTitle converts NotificationLevel enum to string
func NotificationTitle(event event.Event) string {
	if event.Level == v1alpha2.NotificationLevelInfo {
		return InfoTitleText
	} else if event.Level == v1alpha2.NotificationLevelWarning {
		return WarnTitleText
	} else {
		return ""
	}
}
