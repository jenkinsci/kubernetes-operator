package reason

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCopyToVerboseIfNil(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		var verbose []string
		short := []string{"test", "string"}

		copyToVerboseIfNil(short, &verbose)

		assert.NotNil(t, verbose)
		assert.Equal(t, short, verbose)
	})

	t.Run("copy valid first, then invalid", func(t *testing.T) {
		var verbose []string
		valid := []string{"valid", "string"}
		invalid := []string{"invalid", "string"}

		copyToVerboseIfNil(valid, &verbose)
		copyToVerboseIfNil(invalid, &verbose)

		assert.NotNil(t, verbose)
		assert.Equal(t, valid, verbose)
	})

}
