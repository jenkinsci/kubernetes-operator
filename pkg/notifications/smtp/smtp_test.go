package smtp

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	smtp "github.com/emersion/go-smtp"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/event"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/reason"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testSMTPUsername = "username"
	testSMTPPassword = "password"

	testSMTPPort = 1025

	testFrom    = "test@localhost"
	testTo      = "test.to@localhost"
	testSubject = "Jenkins Operator Notification"

	fromHeader    = "From"
	toHeader      = "To"
	subjectHeader = "Subject"

	nilConst = "nil"
)

var (
	testPhase     = event.PhaseUser
	testCrName    = "test-cr"
	testNamespace = "default"
	testReason    = reason.NewPodRestart(
		reason.KubernetesSource,
		[]string{"test-reason-1"},
		[]string{"test-verbose-1"}...,
	)
	testLevel = v1alpha2.NotificationLevelWarning
)

type testServer struct {
	event event.Event
}

func (t *testServer) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &testSession{event: t.event}, nil
}

func (bkd *testServer) Login(_ *smtp.Conn, username, password string) (smtp.Session, error) {
	if username != testSMTPUsername || password != testSMTPPassword {
		return nil, errors.New("invalid username or password")
	}
	return &testSession{event: bkd.event}, nil
}

type testSession struct {
	event event.Event
}

func (s *testSession) Mail(from string, opts *smtp.MailOptions) error {
	if from != testFrom {
		return fmt.Errorf("from header mismatch: got %s expected %s", from, testFrom)
	}
	return nil
}

func (s *testSession) Rcpt(to string, opts *smtp.RcptOptions) error {
	if to != testTo {
		return fmt.Errorf("to header mismatch: got %s expected %s", to, testTo)
	}
	return nil
}

func (s *testSession) Data(r io.Reader) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	content := string(b)

	if !strings.Contains(content, s.event.Jenkins.Name) {
		return fmt.Errorf("jenkins name missing in email")
	}

	if !strings.Contains(content, string(s.event.Phase)) {
		return fmt.Errorf("phase missing in email")
	}

	return nil
}

func (s *testSession) Reset() {}

func (s *testSession) Logout() error {
	return nil
}

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
}
