package try

import (
	"fmt"
	"time"

	"github.com/jenkinsci/kubernetes-operator/internal/errors"
	time2 "github.com/jenkinsci/kubernetes-operator/internal/time"
)

// ErrTimeout is used when the set timeout has been reached
type ErrTimeout struct {
	text  string
	cause error
}

func (e *ErrTimeout) Error() string {
	return fmt.Sprintf("%s: %s", e.text, e.cause.Error())
}

// Cause returns the error that caused ErrTimeout
func (e *ErrTimeout) Cause() error {
	return e.cause
}

// Format implements fmt.Formatter used by Sprint(f) or Fprint(f) etc.
func (e *ErrTimeout) Format(s fmt.State, verb rune) {
	errors.Format(e.cause, s, verb)
}

// Until keeps trying until timeout or there is a result or an error
func Until(something func() (end bool, err error), tick, timeout time.Duration) error {
	counter := 0
	tickChan := time2.Every(tick)
	timeoutChan := time.After(timeout)
	var lastErr error
	for {
		select {
		case <-tickChan:
			end, err := something()
			lastErr = err
			if end {
				return err
			}
			counter = counter + 1
		case <-timeoutChan:
			return &ErrTimeout{
				text:  fmt.Sprintf("timed out after: %s, tries: %d", timeout, counter),
				cause: lastErr,
			}
		}
	}
}
