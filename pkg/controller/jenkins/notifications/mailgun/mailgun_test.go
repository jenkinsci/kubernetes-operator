package mailgun

import (
	"fmt"
	"strings"
	"testing"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/notifications/event"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/notifications/provider"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/notifications/reason"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGenerateMessages(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		crName := "test-jenkins"
		crNamespace := "test-namespace"
		phase := event.PhaseBase
		level := v1alpha2.NotificationLevelInfo
		res := reason.NewUndefined(reason.KubernetesSource, []string{"test-string"}, "test-verbose")

		s := MailGun{
			k8sClient: fake.NewFakeClient(),
			config: v1alpha2.Notification{
				Verbose: true,
			},
		}

		e := event.Event{
			Jenkins: v1alpha2.Jenkins{
				ObjectMeta: metav1.ObjectMeta{
					Name:      crName,
					Namespace: crNamespace,
				},
			},
			Phase:  phase,
			Level:  level,
			Reason: res,
		}

		var statusMessage strings.Builder
		r := strings.TrimRight(strings.Join(e.Reason.Verbose(), "</li><li>"), "<li>")

		statusMessage.WriteString("<ul><li>")
		statusMessage.WriteString(r)
		statusMessage.WriteString("</ul>")

		message := s.generateMessage(e)

		assert.Equal(t, message, fmt.Sprintf(content, s.getStatusColor(e.Level),
			provider.NotificationTitle(e), statusMessage.String(), e.Jenkins.Name, e.Phase))
	})

	t.Run("with nils", func(t *testing.T) {
		crName := "nil"
		crNamespace := "nil"
		phase := event.PhaseBase
		level := v1alpha2.NotificationLevelInfo
		res := reason.NewUndefined(reason.KubernetesSource, []string{"nil"}, "nil")

		s := MailGun{
			k8sClient: fake.NewFakeClient(),
			config: v1alpha2.Notification{
				Verbose: true,
			},
		}

		e := event.Event{
			Jenkins: v1alpha2.Jenkins{
				ObjectMeta: metav1.ObjectMeta{
					Name:      crName,
					Namespace: crNamespace,
				},
			},
			Phase:  phase,
			Level:  level,
			Reason: res,
		}

		var statusMessage strings.Builder
		r := strings.TrimRight(strings.Join(e.Reason.Verbose(), "</li><li>"), "<li>")

		statusMessage.WriteString("<ul><li>")
		statusMessage.WriteString(r)
		statusMessage.WriteString("</ul>")

		message := s.generateMessage(e)

		assert.Equal(t, message, fmt.Sprintf(content, s.getStatusColor(e.Level),
			provider.NotificationTitle(e), statusMessage.String(), e.Jenkins.Name, e.Phase))
	})

	t.Run("with empty strings", func(t *testing.T) {
		crName := ""
		crNamespace := ""
		phase := event.PhaseBase
		level := v1alpha2.NotificationLevelInfo
		res := reason.NewUndefined(reason.KubernetesSource, []string{""}, "")

		s := MailGun{
			k8sClient: fake.NewFakeClient(),
			config: v1alpha2.Notification{
				Verbose: true,
			},
		}

		e := event.Event{
			Jenkins: v1alpha2.Jenkins{
				ObjectMeta: metav1.ObjectMeta{
					Name:      crName,
					Namespace: crNamespace,
				},
			},
			Phase:  phase,
			Level:  level,
			Reason: res,
		}

		var statusMessage strings.Builder
		r := strings.TrimRight(strings.Join(e.Reason.Verbose(), "</li><li>"), "<li>")

		statusMessage.WriteString("<ul><li>")
		statusMessage.WriteString(r)
		statusMessage.WriteString("</ul>")

		message := s.generateMessage(e)

		assert.Equal(t, message, fmt.Sprintf(content, s.getStatusColor(e.Level),
			provider.NotificationTitle(e), statusMessage.String(), e.Jenkins.Name, e.Phase))
	})

	t.Run("with utf-8 characters", func(t *testing.T) {
		crName := "ąśćńółżź"
		crNamespace := "ąśćńółżź"
		phase := event.PhaseBase
		level := v1alpha2.NotificationLevelInfo
		res := reason.NewUndefined(reason.KubernetesSource, []string{"ąśćńółżź"}, "ąśćńółżź")

		s := MailGun{
			k8sClient: fake.NewFakeClient(),
			config: v1alpha2.Notification{
				Verbose: true,
			},
		}

		e := event.Event{
			Jenkins: v1alpha2.Jenkins{
				ObjectMeta: metav1.ObjectMeta{
					Name:      crName,
					Namespace: crNamespace,
				},
			},
			Phase:  phase,
			Level:  level,
			Reason: res,
		}

		var statusMessage strings.Builder
		r := strings.TrimRight(strings.Join(e.Reason.Verbose(), "</li><li>"), "<li>")

		statusMessage.WriteString("<ul><li>")
		statusMessage.WriteString(r)
		statusMessage.WriteString("</ul>")

		message := s.generateMessage(e)

		assert.Equal(t, message, fmt.Sprintf(content, s.getStatusColor(e.Level),
			provider.NotificationTitle(e), statusMessage.String(), e.Jenkins.Name, e.Phase))
	})
}
