package reason

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCopyToVerboseIfNil(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		var verbose []string
		short := []string{"test", "string"}

		assert.Equal(t, checkIfVerboseEmpty(short, verbose), short)
	})

	t.Run("check with invalid slice", func(t *testing.T) {
		valid := []string{"valid", "string"}
		invalid := []string{"invalid", "string"}

		assert.Equal(t, checkIfVerboseEmpty(invalid, valid), valid)
	})

	t.Run("check with two empty slices", func(t *testing.T) {
		var short []string
		var verbose []string

		assert.Equal(t, checkIfVerboseEmpty(short, verbose), verbose)
	})

	t.Run("with nils", func(t *testing.T) {
		assert.Equal(t, checkIfVerboseEmpty(nil, nil), []string(nil))
	})
}

func TestUndefined_HasMessages(t *testing.T) {
	t.Run("short full", func(t *testing.T) {
		podRestart := NewUndefined(KubernetesSource, []string{"test", "another-test"})
		assert.True(t, podRestart.HasMessages())
	})
	
	t.Run("verbose full", func(t *testing.T) {
		podRestart := NewUndefined(KubernetesSource, []string{}, []string{"test", "another-test"}...)
		assert.True(t, podRestart.HasMessages())
	})
	
	t.Run("short empty", func(t *testing.T) {
		podRestart := NewUndefined(KubernetesSource, []string{})
		assert.False(t, podRestart.HasMessages())
	})

	t.Run("verbose and short full", func(t *testing.T) {
		podRestart := NewUndefined(KubernetesSource, []string{"test", "another-test"}, []string{"test", "another-test"}...)
		assert.True(t, podRestart.HasMessages())
	})

	t.Run("verbose and short empty", func(t *testing.T) {
		podRestart := NewUndefined(KubernetesSource, []string{}, []string{}...)
		assert.False(t, podRestart.HasMessages())
	})

}

func TestPodRestartPrepend(t *testing.T) {
	t.Run("happy with one message", func(t *testing.T) {
		res := "test-reason"
		podRestart := NewPodRestart(KubernetesSource, []string{res})

		assert.Equal(t, podRestart.short[0], fmt.Sprintf("Jenkins master pod restarted by: %s", res))
	})

	t.Run("happy with multiple message", func(t *testing.T) {
		podRestart := NewPodRestart(KubernetesSource, []string{"first-reason", "second-reason", "third-reason"})

		assert.Equal(t, podRestart.short[0], "Jenkins master pod restarted by:")
	})
}
