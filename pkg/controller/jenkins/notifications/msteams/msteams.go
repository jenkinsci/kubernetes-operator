package msteams

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

const (
	infoColor    = "439FE0"
	warningColor = "E81123"
	defaultColor = "C8C8C8"
)

// Teams is a Microsoft MicrosoftTeams notification service
type Teams struct {
	httpClient http.Client
	k8sClient  k8sclient.Client
	config     v1alpha2.Notification
}

// New returns instance of Teams
func New(k8sClient k8sclient.Client, config v1alpha2.Notification, httpClient http.Client) *Teams {
	return &Teams{k8sClient: k8sClient, config: config, httpClient: httpClient}
}

// Message is representation of json message structure
type Message struct {
	Type       string            `json:"@type"`
	Context    string            `json:"@context"`
	ThemeColor event.StatusColor `json:"themeColor"`
	Title      string            `json:"title"`
	Sections   []Section         `json:"sections"`
	Summary    string            `json:"summary"`
}

// Section is MS Teams message section
type Section struct {
	Facts []Fact `json:"facts"`
	Text  string `json:"text"`
}

// Fact is field where we can put content
type Fact struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (t Teams) getStatusColor(logLevel v1alpha2.NotificationLevel) event.StatusColor {
	switch logLevel {
	case v1alpha2.NotificationLevelInfo:
		return infoColor
	case v1alpha2.NotificationLevelWarning:
		return warningColor
	default:
		return defaultColor
	}
}

// Send is function for sending directly to API
func (t Teams) Send(e event.Event) error {
	secret := &corev1.Secret{}

	selector := t.config.Teams.WebHookURLSecretKeySelector

	err := t.k8sClient.Get(context.TODO(), types.NamespacedName{Name: selector.Name, Namespace: e.Jenkins.Namespace}, secret)
	if err != nil {
		return errors.WithStack(err)
	}

	secretValue := string(secret.Data[selector.Key])
	if secretValue == "" {
		return errors.Errorf("Microsoft Teams WebHook URL is empty in secret '%s/%s[%s]", e.Jenkins.Namespace, selector.Name, selector.Key)
	}

	var reason string
	if t.config.Verbose {
		reason = strings.Join(e.Reason.Verbose(), "\n\n - ")
	} else {
		reason = strings.Join(e.Reason.Short(), "\n\n - ")
	}

	tm := &Message{
		Title:      provider.NotificationTitle(e),
		Type:       "MessageCard",
		Context:    "https://schema.org/extensions",
		ThemeColor: t.getStatusColor(e.Level),
		Sections: []Section{
			{
				Facts: []Fact{
					{
						Name:  provider.CrNameFieldName,
						Value: e.Jenkins.Name,
					},
					{
						Name:  provider.NamespaceFieldName,
						Value: e.Jenkins.Namespace,
					},
					{
						Name:  provider.PhaseFieldName,
						Value: string(e.Phase),
					},
				},
				Text: reason,
			},
		},
		Summary: reason,
	}

	msg, err := json.Marshal(tm)
	if err != nil {
		return errors.WithStack(err)
	}

	request, err := http.NewRequest("POST", secretValue, bytes.NewBuffer(msg))
	if err != nil {
		return errors.WithStack(err)
	}

	resp, err := t.httpClient.Do(request)
	if err != nil {
		return errors.WithStack(err)
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("Invalid response from server: %s", resp.Status))
	}
	defer func() { _ = resp.Body.Close() }()

	return nil
}
