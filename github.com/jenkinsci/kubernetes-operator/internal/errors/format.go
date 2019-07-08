package errors

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
)

// Format helps to implement fmt.Formatter used by Sprint(f) or Fprint(f) etc.
func Format(err error, s fmt.State, verb rune) {
	formatter, ok := errors.WithStack(err).(fmt.Formatter)
	if !ok {
		// should never occur if the error was wrapped properly
		panic(errors.New("this was unexpected, merged error is not fmt.Formatter"))
	}
	_, _ = io.WriteString(s, err.Error())
	formatter.Format(s, verb)
}
