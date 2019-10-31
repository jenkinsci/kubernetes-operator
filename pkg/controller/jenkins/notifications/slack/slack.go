package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/notifications/event"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/notifications/provider"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Slack is a Slack notification service
type Slack struct {
	httpClient http.Client
	k8sClient  k8sclient.Client
	config     v1alpha2.Notification
}

// New returns instance of Slack
func New(k8sClient k8sclient.Client, config v1alpha2.Notification, httpClient http.Client) *Slack {
	return &Slack{k8sClient: k8sClient, config: config, httpClient: httpClient}
}

// Message is representation of json message
type Message struct {
	Text        string       `json:"text"`
	Attachments []Attachment `json:"attachments"`
}

// Attachment is representation of json attachment
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
		return "#439FE0"
	case v1alpha2.NotificationLevelWarning:
		return "danger"
	default:
		return "#c8c8c8"
	}
}

// Send is function for sending directly to API
func (s Slack) Send(e event.Event) error {
	secret := &corev1.Secret{}
	selector := s.config.Slack.WebHookURLSecretKeySelector

	err := s.k8sClient.Get(context.TODO(), types.NamespacedName{Name: selector.Name, Namespace: e.Jenkins.Namespace}, secret)
	if err != nil {
		return err
	}

	sm := &Message{
		Attachments: []Attachment{
			{
				Fallback: "",
				Color:    s.getStatusColor(e.Level),
				Fields: []Field{
					{
						Title: "",
						Value: "",
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
				},
			},
		},
	}

	mainAttachment := &sm.Attachments[0]

	var messageStringBuilder strings.Builder
	for _, msg := range e.Reason.Short() {
		messageStringBuilder.WriteString("\n - " + msg + "\n")
	}
	mainAttachment.Fields[0].Value = messageStringBuilder.String()
	// TODO: add verbose

	mainAttachment.Title = provider.NotificationTitle(e)

	if e.Phase != event.PhaseUnknown {
		mainAttachment.Fields = append(mainAttachment.Fields, Field{
			Title: provider.PhaseFieldName,
			Value: string(e.Phase),
			Short: true,
		})
	}

	slackMessage, err := json.Marshal(sm)
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
