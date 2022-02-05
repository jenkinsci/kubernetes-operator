package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/event"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/provider"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	infoColor    = "#439FE0"
	warningColor = "danger"
	defaultColor = "#c8c8c8"
)

// Slack is a Slack notification service.
type Slack struct {
	httpClient http.Client
	k8sClient  k8sclient.Client
	config     v1alpha2.Notification
}

// New returns instance of Slack.
func New(k8sClient k8sclient.Client, config v1alpha2.Notification, httpClient http.Client) *Slack {
	return &Slack{k8sClient: k8sClient, config: config, httpClient: httpClient}
}

// Message is representation of json message.
type Message struct {
	Text        string       `json:"text"`
	Attachments []Attachment `json:"attachments"`
}

// Attachment is representation of json attachment.
type Attachment struct {
	Fallback string            `json:"fallback"`
	Color    event.StatusColor `json:"color"`
	Pretext  string            `json:"pretext"`
	Title    string            `json:"title"`
	Text     string            `json:"text"`
	Fields   []Field           `json:"fields"`
	Footer   string            `json:"footer"`
}

// Field is representation of json field.
type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

func (s Slack) getStatusColor(logLevel v1alpha2.NotificationLevel) event.StatusColor {
	switch logLevel {
	case v1alpha2.NotificationLevelInfo:
		return infoColor
	case v1alpha2.NotificationLevelWarning:
		return warningColor
	default:
		return defaultColor
	}
}

func (s Slack) generateMessage(e event.Event) Message {
	var messageStringBuilder strings.Builder
	if s.config.Verbose {
		for _, msg := range e.Reason.Verbose() {
			messageStringBuilder.WriteString("\n - " + msg + "\n")
		}
	} else {
		for _, msg := range e.Reason.Short() {
			messageStringBuilder.WriteString("\n - " + msg + "\n")
		}
	}

	sm := Message{
		Attachments: []Attachment{
			{
				Title:    provider.NotificationTitle(e),
				Fallback: "",
				Color:    s.getStatusColor(e.Level),
				Fields: []Field{
					{
						Title: "",
						Value: messageStringBuilder.String(),
						Short: false,
					},
					{
						Title: provider.NamespaceFieldName,
						Value: e.Jenkins.Namespace,
						Short: true,
					},
					{
						Title: provider.CrNameFieldName,
						Value: e.Jenkins.Name,
						Short: true,
					},
					{
						Title: provider.PhaseFieldName,
						Value: string(e.Phase),
						Short: true,
					},
				},
			},
		},
	}

	return sm
}

// Send is function for sending directly to API.
func (s Slack) Send(e event.Event) error {
	secret := &corev1.Secret{}
	selector := s.config.Slack.WebHookURLSecretKeySelector

	err := s.k8sClient.Get(context.TODO(), types.NamespacedName{Name: selector.Name, Namespace: e.Jenkins.Namespace}, secret)
	if err != nil {
		return err
	}

	slackMessage, err := json.Marshal(s.generateMessage(e))
	if err != nil {
		return err
	}

	secretValue := string(secret.Data[selector.Key])
	if secretValue == "" {
		return errors.Errorf("Slack WebHook URL is empty in secret '%s/%s[%s]", e.Jenkins.Namespace, selector.Name, selector.Key)
	}

	request, err := http.NewRequest("POST", secretValue, bytes.NewBuffer(slackMessage))
	if err != nil {
		return err
	}

	resp, err := s.httpClient.Do(request)
	if err != nil {
		return err
	}

	defer func() { _ = resp.Body.Close() }()
	return nil
}
