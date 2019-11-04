package reason

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCopyToVerboseIfNil(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		var verbose []string
		short := []string {"test", "string"}

		copyToVerboseIfNil(short, &verbose)

		assert.NotNil(t, verbose)
		assert.Equal(t, short, verbose)
	})
}