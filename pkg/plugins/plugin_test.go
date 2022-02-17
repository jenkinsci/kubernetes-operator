package plugins

import (
	"testing"

	"github.com/jenkinsci/kubernetes-operator/pkg/log"

	"github.com/stretchr/testify/assert"
)

func TestValidatePlugin(t *testing.T) {
	validPluginName := "valid"
	validPluginVersion := "0.1.2"
	t.Run("version 1.6+build.162", func(t *testing.T) {
		got := validatePlugin(validPluginName, "1.6+build.162", "")
		assert.NoError(t, got)
	})
	t.Run("version 1.8+build.201601050116", func(t *testing.T) {
		got := validatePlugin(validPluginName, "1.8+build.201601050116", "")
		assert.NoError(t, got)
	})
	t.Run("version 1.8-RELEASE", func(t *testing.T) {
		got := validatePlugin(validPluginName, "1.8-RELEASE", "")
		assert.NoError(t, got)
	})
	t.Run("version 20.810504d7462", func(t *testing.T) {
		got := validatePlugin(validPluginName, "20.810504d7462", "")
		assert.NoError(t, got)
	})

	t.Run("version 3.0-rc1", func(t *testing.T) {
		got := validatePlugin(validPluginName, "3.0-rc1", "")
		assert.NoError(t, got)
	})
	t.Run("version 3.1.20180605-140134.c2e96c4", func(t *testing.T) {
		got := validatePlugin(validPluginName, "3.1.20180605-140134.c2e96c4", "")
		assert.NoError(t, got)
	})
	t.Run("version 1074.v60e6c29b_b_44b_", func(t *testing.T) {
		got := validatePlugin(validPluginName, "1074.v60e6c29b_b_44b_", "")
		assert.NoError(t, got)
	})
	t.Run("invalid version !", func(t *testing.T) {
		got := validatePlugin(validPluginName, "0.5.1!", "")
		assert.Error(t, got)
	})
	t.Run("name 01234567890-abcdefghijklmnoprstuwxz_ABCDEFGHIJKLMNOPQRSTUVWXYZ", func(t *testing.T) {
		got := validatePlugin("01234567890-abcdefghijklmnoprstuwxz_ABCDEFGHIJKLMNOPQRSTUVWXYZ", validPluginVersion, "")
		assert.NoError(t, got)
	})
	t.Run("invalid name !", func(t *testing.T) {
		got := validatePlugin("!", validPluginVersion, "")
		assert.Error(t, got)
	})
	t.Run("invalid download URL", func(t *testing.T) {
		got := validatePlugin(validPluginName, validPluginVersion, "http://www.jenkins/plugin.hpi")
		assert.Error(t, got)
	})
	t.Run("valid http download URL", func(t *testing.T) {
		got := validatePlugin(validPluginName, validPluginVersion, "http://www.jenkins.com/plugin.hpi")
		assert.NoError(t, got)
	})
	t.Run("valid https download URL", func(t *testing.T) {
		got := validatePlugin(validPluginName, validPluginVersion, "https://www.jenkins.com/plugin.hpi")
		assert.NoError(t, got)
	})
}

func TestVerifyDependencies(t *testing.T) {
	log.SetupLogger(false)

	t.Run("happy, single root plugin with one dependent plugin", func(t *testing.T) {
		basePlugins := map[Plugin][]Plugin{
			Must(New("first-root-plugin:1.0.0")): {
				Must(New("first-plugin:0.0.1")),
			},
		}
		got := VerifyDependencies(basePlugins)
		assert.Nil(t, got)
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
		assert.Nil(t, got)
	})
	t.Run("happy, two plugin names with names with underscores", func(t *testing.T) {
		basePlugins := map[Plugin][]Plugin{
			Must(New("first_root_plugin:1.0.0")): {
				Must(New("first_plugin:0.0.1")),
			},
			Must(New("second_root_plugin:1.0.0")): {
				Must(New("first_plugin:0.0.1")),
			},
		}
		got := VerifyDependencies(basePlugins)
		assert.Nil(t, got)
	})
	t.Run("happy, two plugin names with uppercase names", func(t *testing.T) {
		basePlugins := map[Plugin][]Plugin{
			Must(New("First-Root-Plugin:1.0.0")): {
				Must(New("First_Plugin:0.0.1")),
			},
			Must(New("Second_Root_Plugin:1.0.0")): {
				Must(New("First_Plugin:0.0.1")),
			},
		}
		got := VerifyDependencies(basePlugins)
		assert.Nil(t, got)
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
		assert.Contains(t, got, "Plugin 'first-root-plugin:1.0.0' requires version '1.0.0' but plugin 'first-root-plugin:2.0.0' requires '2.0.0' for plugin 'first-root-plugin'")
		assert.Contains(t, got, "Plugin 'first-root-plugin:2.0.0' requires version '2.0.0' but plugin 'first-root-plugin:1.0.0' requires '1.0.0' for plugin 'first-root-plugin'")
	})
	t.Run("happy, no version collision with two separate plugins lists", func(t *testing.T) {
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
		assert.Nil(t, got)
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
		assert.Contains(t, got, "Plugin 'first-root-plugin:1.0.0' requires version '1.0.0' but plugin 'first-root-plugin:2.0.0' requires '2.0.0' for plugin 'first-root-plugin'")
		assert.Contains(t, got, "Plugin 'first-root-plugin:2.0.0' requires version '2.0.0' but plugin 'first-root-plugin:1.0.0' requires '1.0.0' for plugin 'first-root-plugin'")
		assert.Contains(t, got, "Plugin 'first-root-plugin:1.0.0' requires version '0.0.1' but plugin 'first-root-plugin:2.0.0' requires '0.0.2' for plugin 'first-plugin'")
		assert.Contains(t, got, "Plugin 'first-root-plugin:2.0.0' requires version '0.0.2' but plugin 'first-root-plugin:1.0.0' requires '0.0.1' for plugin 'first-plugin'")
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
		assert.Contains(t, got, "Plugin 'first-root-plugin:1.0.0' requires version '1.0.0' but plugin 'first-root-plugin:2.0.0' requires '2.0.0' for plugin 'first-root-plugin'")
		assert.Contains(t, got, "Plugin 'first-root-plugin:2.0.0' requires version '2.0.0' but plugin 'first-root-plugin:1.0.0' requires '1.0.0' for plugin 'first-root-plugin'")
		assert.Contains(t, got, "Plugin 'first-root-plugin:1.0.0' requires version '0.0.1' but plugin 'first-root-plugin:2.0.0' requires '0.0.2' for plugin 'first-plugin'")
		assert.Contains(t, got, "Plugin 'first-root-plugin:2.0.0' requires version '0.0.2' but plugin 'first-root-plugin:1.0.0' requires '0.0.1' for plugin 'first-plugin'")
	})
	t.Run("happy with dash in version", func(t *testing.T) {
		basePlugins := map[Plugin][]Plugin{
			Must(New("first-root-plugin:1.0.0-1")): {
				Must(New("first-plugin:0.0.1-1")),
			},
		}
		got := VerifyDependencies(basePlugins)
		assert.Nil(t, got)
	})
}
