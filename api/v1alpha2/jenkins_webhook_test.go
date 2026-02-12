package v1alpha2

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMakeSemanticVersion(t *testing.T) {
	t.Run("only major version specified", func(t *testing.T) {
		got := makeSemanticVersion("1")
		assert.Equal(t, got, "v1.0.0")
	})

	t.Run("major and minor version specified", func(t *testing.T) {
		got := makeSemanticVersion("1.2")
		assert.Equal(t, got, "v1.2.0")
	})

	t.Run("major,minor and patch version specified", func(t *testing.T) {
		got := makeSemanticVersion("1.2.3")
		assert.Equal(t, got, "v1.2.3")
	})

	t.Run("semantic versions begin with a leading v and no patch version", func(t *testing.T) {
		got := makeSemanticVersion("v2.5")
		assert.Equal(t, got, "v2.5.0")
	})

	t.Run("semantic versions with prerelease versions", func(t *testing.T) {
		got := makeSemanticVersion("2.1.2-alpha.1")
		assert.Equal(t, got, "v2.1.2-alpha.1")
	})

	t.Run("semantic versions with prerelease versions", func(t *testing.T) {
		got := makeSemanticVersion("0.11.2-9.c8b45b8bb173")
		assert.Equal(t, got, "v0.11.2-9.c8b45b8bb173")
	})

	t.Run("semantic versions with build suffix", func(t *testing.T) {
		got := makeSemanticVersion("1.7.9+meta")
		assert.Equal(t, got, "v1.7.9")
	})

	t.Run("invalid semantic version", func(t *testing.T) {
		got := makeSemanticVersion("google-login-1.2")
		assert.Equal(t, got, "")
	})
}

func TestCompareVersions(t *testing.T) {
	t.Run("Plugin Version lies between first and last version", func(t *testing.T) {
		got := compareVersions("1.2", "1.6", "1.4")
		assert.Equal(t, got, true)
	})
	t.Run("Plugin Version is greater than the last version", func(t *testing.T) {
		got := compareVersions("1", "2", "3")
		assert.Equal(t, got, false)
	})
	t.Run("Plugin Version is less than the first version", func(t *testing.T) {
		got := compareVersions("1.4", "2.5", "1.1")
		assert.Equal(t, got, false)
	})

	t.Run("Plugins Versions have prerelease version and it lies between first and last version", func(t *testing.T) {
		got := compareVersions("1.2.1-alpha", "1.2.1", "1.2.1-beta")
		assert.Equal(t, got, true)
	})

	t.Run("Plugins Versions have prerelease version and it is greater than the last version", func(t *testing.T) {
		got := compareVersions("v2.2.1-alpha", "v2.5.1-beta.1", "v2.5.1-beta.2")
		assert.Equal(t, got, false)
	})
}

func TestValidate(t *testing.T) {
	t.Run("Validating when plugins data file is not fetched", func(t *testing.T) {
		userplugins := []Plugin{{Name: "script-security", Version: "1.77"}, {Name: "git-client", Version: "3.9"}, {Name: "git", Version: "4.8.1"}, {Name: "plain-credentials", Version: "1.7"}}
		jenkinscr := *createJenkinsCR(userplugins, true)
		_, got := jenkinscr.ValidateCreate(context.TODO(), &jenkinscr)
		assert.Equal(t, got, errors.New("plugins data has not been fetched"))
	})

	SecValidator.isCached = true
	t.Run("Validating a Jenkins CR with plugins not having security warnings and validation is turned on", func(t *testing.T) {
		SecValidator.PluginDataCache = PluginsInfo{Plugins: []PluginInfo{
			{Name: "security-script"},
			{Name: "git-client"},
			{Name: "git"},
			{Name: "google-login", SecurityWarnings: createSecurityWarnings("", "1.2")},
			{Name: "sample-plugin", SecurityWarnings: createSecurityWarnings("", "0.8")},
			{Name: "mailer"},
			{Name: "plain-credentials"}}}
		userplugins := []Plugin{{Name: "script-security", Version: "1.77"}, {Name: "git-client", Version: "3.9"}, {Name: "git", Version: "4.8.1"}, {Name: "plain-credentials", Version: "1.7"}}
		jenkinscr := *createJenkinsCR(userplugins, true)
		_, got := jenkinscr.ValidateCreate(context.TODO(), &jenkinscr)
		assert.Nil(t, got)
	})

	t.Run("Validating a Jenkins CR with some of the plugins having security warnings and validation is turned on", func(t *testing.T) {
		SecValidator.PluginDataCache = PluginsInfo{Plugins: []PluginInfo{
			{Name: "security-script", SecurityWarnings: createSecurityWarnings("1.2", "2.2")},
			{Name: "workflow-cps", SecurityWarnings: createSecurityWarnings("2.59", "")},
			{Name: "git-client"},
			{Name: "git"},
			{Name: "sample-plugin", SecurityWarnings: createSecurityWarnings("0.8", "")},
			{Name: "command-launcher", SecurityWarnings: createSecurityWarnings("1.2", "1.4")},
			{Name: "plain-credentials"},
			{Name: "google-login", SecurityWarnings: createSecurityWarnings("1.1", "1.3")},
			{Name: "mailer", SecurityWarnings: createSecurityWarnings("1.0.3", "1.1.4")},
		}}
		userplugins := []Plugin{{Name: "google-login", Version: "1.2"}, {Name: "mailer", Version: "1.1"}, {Name: "git", Version: "4.8.1"}, {Name: "command-launcher", Version: "1.6"}, {Name: "workflow-cps", Version: "2.59"}}
		jenkinscr := *createJenkinsCR(userplugins, true)
		_, got := jenkinscr.ValidateCreate(context.TODO(), &jenkinscr)
		assert.Equal(t, got, errors.New("security vulnerabilities detected in the following user-defined plugins: \nworkflow-cps:2.59\ngoogle-login:1.2\nmailer:1.1"))
	})

	t.Run("Updating a Jenkins CR with some of the plugins having security warnings and validation is turned on", func(t *testing.T) {
		SecValidator.PluginDataCache = PluginsInfo{Plugins: []PluginInfo{
			{Name: "handy-uri-templates-2-api", SecurityWarnings: createSecurityWarnings("2.1.8-1.0", "2.2.8-1.0")},
			{Name: "workflow-cps", SecurityWarnings: createSecurityWarnings("2.59", "")},
			{Name: "resource-disposer", SecurityWarnings: createSecurityWarnings("0.7", "1.2")},
			{Name: "git"},
			{Name: "jjwt-api"},
			{Name: "blueocean-github-pipeline", SecurityWarnings: createSecurityWarnings("1.2.0-alpha-2", "1.2.0-beta-5")},
			{Name: "command-launcher", SecurityWarnings: createSecurityWarnings("1.2", "1.4")},
			{Name: "plain-credentials"},
			{Name: "ghprb", SecurityWarnings: createSecurityWarnings("1.1", "1.43")},
			{Name: "mailer", SecurityWarnings: createSecurityWarnings("1.0.3", "1.1.4")},
		}}

		userplugins := []Plugin{{Name: "google-login", Version: "1.2"}, {Name: "mailer", Version: "1.1"}, {Name: "git", Version: "4.8.1"}, {Name: "command-launcher", Version: "1.6"}, {Name: "workflow-cps", Version: "2.59"}}
		oldjenkinscr := *createJenkinsCR(userplugins, true)

		userplugins = []Plugin{{Name: "handy-uri-templates-2-api", Version: "2.1.8-1.0"}, {Name: "resource-disposer", Version: "0.8"}, {Name: "jjwt-api", Version: "0.11.2-9.c8b45b8bb173"}, {Name: "blueocean-github-pipeline", Version: "1.2.0-beta-3"}, {Name: "ghprb", Version: "1.39"}}
		newjenkinscr := *createJenkinsCR(userplugins, true)
		_, got := newjenkinscr.ValidateUpdate(context.TODO(), &oldjenkinscr, &newjenkinscr)
		assert.Equal(t, got, errors.New("security vulnerabilities detected in the following user-defined plugins: \nhandy-uri-templates-2-api:2.1.8-1.0\nresource-disposer:0.8\nblueocean-github-pipeline:1.2.0-beta-3\nghprb:1.39"))
	})

	t.Run("Validation is turned off", func(t *testing.T) {
		userplugins := []Plugin{{Name: "google-login", Version: "1.2"}, {Name: "mailer", Version: "1.1"}, {Name: "git", Version: "4.8.1"}, {Name: "command-launcher", Version: "1.6"}, {Name: "workflow-cps", Version: "2.59"}}
		jenkinscr := *createJenkinsCR(userplugins, false)
		_, got := jenkinscr.ValidateCreate(context.TODO(), &jenkinscr)
		assert.Nil(t, got)

		userplugins = []Plugin{{Name: "google-login", Version: "1.2"}, {Name: "mailer", Version: "1.1"}, {Name: "git", Version: "4.8.1"}, {Name: "command-launcher", Version: "1.6"}, {Name: "workflow-cps", Version: "2.59"}}
		newjenkinscr := *createJenkinsCR(userplugins, false)
		_, got = newjenkinscr.ValidateUpdate(context.TODO(), &jenkinscr, &newjenkinscr)
		assert.Nil(t, got)
	})
}

func createJenkinsCR(userPlugins []Plugin, validateSecurityWarnings bool) *Jenkins {
	jenkins := &Jenkins{
		TypeMeta: JenkinsTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jenkins",
			Namespace: "test",
		},
		Spec: JenkinsSpec{
			Master: JenkinsMaster{
				Plugins:               userPlugins,
				DisableCSRFProtection: false,
			},
			ValidateSecurityWarnings: validateSecurityWarnings,
		},
	}

	return jenkins
}

func createSecurityWarnings(firstVersion string, lastVersion string) []Warning {
	return []Warning{{Versions: []Version{{FirstVersion: firstVersion, LastVersion: lastVersion}}, ID: "null", Message: "unit testing", URL: "null", Active: false}}
}
