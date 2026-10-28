package v1alpha2

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func TestJenkinsCustomValidator(t *testing.T) {
	previous := SecValidator
	t.Cleanup(func() { SecValidator = previous })
	SecValidator = SecurityValidator{
		isCached: true,
		PluginDataCache: PluginsInfo{Plugins: []PluginInfo{
			{Name: "sample-plugin", SecurityWarnings: createSecurityWarnings("1.0", "2.0")},
		}},
	}
	validator := &JenkinsCustomValidator{}
	ctx := context.Background()
	safe := createJenkinsCR([]Plugin{{Name: "sample-plugin", Version: "3.0"}}, true)
	vulnerable := createJenkinsCR([]Plugin{{Name: "sample-plugin", Version: "1.5"}}, true)
	disabled := createJenkinsCR([]Plugin{{Name: "sample-plugin", Version: "1.5"}}, false)

	tests := []struct {
		name      string
		validate  func() (admission.Warnings, error)
		wantError bool
	}{
		{"create rejects vulnerable plugins", func() (admission.Warnings, error) { return validator.ValidateCreate(ctx, vulnerable) }, true},
		{"create allows safe plugins", func() (admission.Warnings, error) { return validator.ValidateCreate(ctx, safe) }, false},
		{"create allows disabled validation", func() (admission.Warnings, error) { return validator.ValidateCreate(ctx, disabled) }, false},
		{"update rejects new vulnerable plugins", func() (admission.Warnings, error) { return validator.ValidateUpdate(ctx, safe, vulnerable) }, true},
		{"update allows removal of vulnerable plugins", func() (admission.Warnings, error) { return validator.ValidateUpdate(ctx, vulnerable, safe) }, false},
		{"update allows disabled validation", func() (admission.Warnings, error) { return validator.ValidateUpdate(ctx, safe, disabled) }, false},
		{"delete allows vulnerable plugins", func() (admission.Warnings, error) { return validator.ValidateDelete(ctx, vulnerable) }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings, err := tt.validate()
			assert.Empty(t, warnings)
			if tt.wantError {
				assert.ErrorContains(t, err, "sample-plugin:1.5")
			} else {
				assert.NoError(t, err)
			}
		})
	}
	_, err := validator.ValidateCreate(ctx, nil)
	assert.Error(t, err)
	_, err = validator.ValidateUpdate(ctx, safe, nil)
	assert.Error(t, err)
	_, err = validator.ValidateDelete(ctx, nil)
	assert.Error(t, err)
}

func TestJenkinsAdmissionSecurityValidation(t *testing.T) {
	previous := SecValidator
	t.Cleanup(func() { SecValidator = previous })
	SecValidator = SecurityValidator{
		isCached: true,
		PluginDataCache: PluginsInfo{Plugins: []PluginInfo{
			{Name: "sample-plugin", SecurityWarnings: createSecurityWarnings("1.0", "2.0")},
		}},
	}
	scheme := runtime.NewScheme()
	require.NoError(t, AddToScheme(scheme))
	handler := admission.WithValidator(scheme, &JenkinsCustomValidator{})
	safe := createJenkinsCR([]Plugin{{Name: "sample-plugin", Version: "3.0"}}, true)
	vulnerable := createJenkinsCR([]Plugin{{Name: "sample-plugin", Version: "1.5"}}, true)
	disabled := createJenkinsCR([]Plugin{{Name: "sample-plugin", Version: "1.5"}}, false)
	tests := []struct {
		name      string
		operation admissionv1.Operation
		oldObj    *Jenkins
		newObj    *Jenkins
		allowed   bool
	}{
		{"create rejects vulnerable plugins", admissionv1.Create, nil, vulnerable, false},
		{"create accepts safe plugins", admissionv1.Create, nil, safe, true},
		{"create accepts disabled validation", admissionv1.Create, nil, disabled, true},
		{"update rejects vulnerable plugins", admissionv1.Update, safe, vulnerable, false},
		{"update accepts removal of vulnerable plugins", admissionv1.Update, vulnerable, safe, true},
		{"delete accepts vulnerable plugins", admissionv1.Delete, vulnerable, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := admission.Request{AdmissionRequest: admissionv1.AdmissionRequest{
				Operation: tt.operation,
				Kind:      metav1.GroupVersionKind{Group: "jenkins.io", Version: "v1alpha2", Kind: "Jenkins"},
			}}
			if tt.oldObj != nil {
				raw, err := json.Marshal(tt.oldObj)
				require.NoError(t, err)
				req.OldObject = runtime.RawExtension{Raw: raw}
			}
			if tt.newObj != nil {
				raw, err := json.Marshal(tt.newObj)
				require.NoError(t, err)
				req.Object = runtime.RawExtension{Raw: raw}
			}
			response := handler.Handle(context.Background(), req)
			assert.Equal(t, tt.allowed, response.Allowed)
			if !tt.allowed {
				require.NotNil(t, response.Result)
				assert.Contains(t, response.Result.Message, "sample-plugin:1.5")
			}
		})
	}
}

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
		_, got := jenkinscr.ValidateCreate()
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
		_, got := jenkinscr.ValidateCreate()
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
		_, got := jenkinscr.ValidateCreate()
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
		_, got := newjenkinscr.ValidateUpdate(&oldjenkinscr)
		assert.Equal(t, got, errors.New("security vulnerabilities detected in the following user-defined plugins: \nhandy-uri-templates-2-api:2.1.8-1.0\nresource-disposer:0.8\nblueocean-github-pipeline:1.2.0-beta-3\nghprb:1.39"))
	})

	t.Run("Validation is turned off", func(t *testing.T) {
		userplugins := []Plugin{{Name: "google-login", Version: "1.2"}, {Name: "mailer", Version: "1.1"}, {Name: "git", Version: "4.8.1"}, {Name: "command-launcher", Version: "1.6"}, {Name: "workflow-cps", Version: "2.59"}}
		jenkinscr := *createJenkinsCR(userplugins, false)
		_, got := jenkinscr.ValidateCreate()
		assert.Nil(t, got)

		userplugins = []Plugin{{Name: "google-login", Version: "1.2"}, {Name: "mailer", Version: "1.1"}, {Name: "git", Version: "4.8.1"}, {Name: "command-launcher", Version: "1.6"}, {Name: "workflow-cps", Version: "2.59"}}
		newjenkinscr := *createJenkinsCR(userplugins, false)
		_, got = newjenkinscr.ValidateUpdate(&jenkinscr)
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
