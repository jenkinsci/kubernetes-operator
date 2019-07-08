package plugins

import (
	"testing"

	"github.com/jenkinsci/kubernetes-operator/pkg/log"

	"github.com/stretchr/testify/assert"
)

func TestVerifyDependencies(t *testing.T) {
	log.SetupLogger(false)

	t.Run("happy, single root plugin with one dependent plugin", func(t *testing.T) {
		basePlugins := map[Plugin][]Plugin{
			Must(New("first-root-plugin:1.0.0")): {
				Must(New("first-plugin:0.0.1")),
			},
		}
		got := VerifyDependencies(basePlugins)
		assert.Equal(t, true, got)
	})
	t.Run("happy, two root plugins with one depended plugin with the same version", func(t *testing.T) {
		basePlugins := map[Plugin][]Plugin{
			Must(New("first-root-plugin:1.0.0")): {
				Must(New("first-plugin:0.0.1")),
			},
			Must(New("second-root-plugin:1.0.0")): {
				Must(New("first-plugin:0.0.1")),
			},
		}
		got := VerifyDependencies(basePlugins)
		assert.Equal(t, true, got)
	})
	t.Run("fail, two root plugins have different versions", func(t *testing.T) {
		basePlugins := map[Plugin][]Plugin{
			Must(New("first-root-plugin:1.0.0")): {
				Must(New("first-plugin:0.0.1")),
			},
			Must(New("first-root-plugin:2.0.0")): {
				Must(New("first-plugin:0.0.1")),
			},
		}
		got := VerifyDependencies(basePlugins)
		assert.Equal(t, false, got)
	})
	t.Run("happy, no version collision with two sperate plugins lists", func(t *testing.T) {
		basePlugins := map[Plugin][]Plugin{
			Must(New("first-root-plugin:1.0.0")): {
				Must(New("first-plugin:0.0.1")),
			},
		}
		extraPlugins := map[Plugin][]Plugin{
			Must(New("second-root-plugin:2.0.0")): {
				Must(New("first-plugin:0.0.1")),
			},
		}
		got := VerifyDependencies(basePlugins, extraPlugins)
		assert.Equal(t, true, got)
	})
	t.Run("fail, dependent plugins have different versions", func(t *testing.T) {
		basePlugins := map[Plugin][]Plugin{
			Must(New("first-root-plugin:1.0.0")): {
				Must(New("first-plugin:0.0.1")),
			},
			Must(New("first-root-plugin:2.0.0")): {
				Must(New("first-plugin:0.0.2")),
			},
		}
		got := VerifyDependencies(basePlugins)
		assert.Equal(t, false, got)
	})
	t.Run("fail, root and dependent plugins have different versions", func(t *testing.T) {
		basePlugins := map[Plugin][]Plugin{
			Must(New("first-root-plugin:1.0.0")): {
				Must(New("first-plugin:0.0.1")),
			},
		}
		extraPlugins := map[Plugin][]Plugin{
			Must(New("first-root-plugin:2.0.0")): {
				Must(New("first-plugin:0.0.2")),
			},
		}
		got := VerifyDependencies(basePlugins, extraPlugins)
		assert.Equal(t, false, got)
	})
}
