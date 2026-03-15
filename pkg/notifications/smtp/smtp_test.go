package smtp

import (
	"testing"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/event"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/reason"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const nilConst = "nil"

func TestGenerateMessage(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		crName := "test-jenkins"
		phase := event.PhaseBase
		level := v1alpha2.NotificationLevelInfo
		res := reason.NewUndefined(reason.KubernetesSource, []string{"test"}, []string{"test-verbose"}...)

		from := "from@jenkins.local"
		to := "to@jenkins.local"

		e := event.Event{
			Jenkins: v1alpha2.Jenkins{
				ObjectMeta: metav1.ObjectMeta{
					Name: crName,
				},
			},
			Phase:  phase,
			Level:  level,
			Reason: res,
		}

		s := SMTP{
			k8sClient: fake.NewClientBuilder().Build(),
			config: v1alpha2.Notification{
				LoggingLevel: level,
				SMTP: &v1alpha2.SMTP{
					From: from,
					To:   to,
				},
			},
		}

		message := s.generateMessage(e)
		assert.NotNil(t, message)
	})

	t.Run("with nils", func(t *testing.T) {
		crName := nilConst
		phase := event.PhaseBase
		level := v1alpha2.NotificationLevelInfo
		res := reason.NewUndefined(reason.KubernetesSource, []string{nilConst}, []string{nilConst}...)

		from := nilConst
		to := nilConst

		e := event.Event{
			Jenkins: v1alpha2.Jenkins{
				ObjectMeta: metav1.ObjectMeta{
					Name: crName,
				},
			},
			Phase:  phase,
			Level:  level,
			Reason: res,
		}

		s := SMTP{
			k8sClient: fake.NewClientBuilder().Build(),
			config: v1alpha2.Notification{
				LoggingLevel: level,
				SMTP: &v1alpha2.SMTP{
					From: from,
					To:   to,
				},
			},
		}

		message := s.generateMessage(e)
		assert.NotNil(t, message)
	})

	t.Run("with empty strings", func(t *testing.T) {
		crName := ""
		phase := event.PhaseBase
		level := v1alpha2.NotificationLevelInfo
		res := reason.NewUndefined(reason.KubernetesSource, []string{""}, []string{""}...)

		from := ""
		to := ""

		e := event.Event{
			Jenkins: v1alpha2.Jenkins{
				ObjectMeta: metav1.ObjectMeta{
					Name: crName,
				},
			},
			Phase:  phase,
			Level:  level,
			Reason: res,
		}

		s := SMTP{
			k8sClient: fake.NewClientBuilder().Build(),
			config: v1alpha2.Notification{
				LoggingLevel: level,
				SMTP: &v1alpha2.SMTP{
					From: from,
					To:   to,
				},
			},
		}

		message := s.generateMessage(e)
		assert.NotNil(t, message)
	})
}
