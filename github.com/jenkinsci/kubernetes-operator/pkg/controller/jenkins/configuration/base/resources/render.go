package resources

import (
	"bytes"
	"text/template"

	"github.com/pkg/errors"
)

// render executes a parsed template (go-template) with configuration from data
func render(template *template.Template, data interface{}) (string, error) {
	var buffer bytes.Buffer
	if err := template.Execute(&buffer, data); err != nil {
		return "", errors.WithStack(err)
	}

	return buffer.String(), nil
}
