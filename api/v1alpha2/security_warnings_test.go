package v1alpha2

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const securityWarningsJSON = `{"warnings":[
	{"type":"plugin","name":"workflow-cps","id":"SECURITY-1266-workflow-cps",
	 "versions":[{"pattern":"(?!2[.]61[.].*)(1|2[.]([0-9]|[1-5][0-9]|6[01]))(|[.-].*)"}]},
	{"type":"plugin","name":"sonar-quality-gates","id":"SECURITY-1523",
	 "versions":[{"pattern":"(1|309|312)(|[.-].+)"}]},
	{"type":"plugin","name":"github","id":"SECURITY-3704",
	 "versions":[{"pattern":"(0|1[.][0-9]|1[.]1[0-9]|1[.]2[0-9]|1[.]3[0-9]|1[.]4[0-5])(|[.-].+)|(1[.]46[.]0)"}]},
	{"type":"core","name":"github","id":"CORE-EXAMPLE","versions":[{"pattern":".*"}]},
	{"type":"future","name":"github","id":"FUTURE-EXAMPLE","versions":[{"pattern":"["}]}
]}`

type securityWarningsTransport struct {
	target    *url.URL
	transport http.RoundTripper
}

func (in securityWarningsTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	request = request.Clone(request.Context())
	request.URL.Scheme = in.target.Scheme
	request.URL.Host = in.target.Host
	return in.transport.RoundTrip(request)
}

func serveSecurityWarnings(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	target, err := url.Parse(server.URL)
	require.NoError(t, err)
	previous := http.DefaultTransport
	http.DefaultTransport = securityWarningsTransport{target: target, transport: server.Client().Transport}
	t.Cleanup(func() { http.DefaultTransport = previous })
}

func isolateSecurityValidator(t *testing.T) {
	t.Helper()
	previous := SecValidator
	previousDownloaded := initialSecurityWarningsDownloadSucceded
	SecValidator = *NewSecurityValidator()
	initialSecurityWarningsDownloadSucceded = false
	t.Cleanup(func() {
		SecValidator = previous
		initialSecurityWarningsDownloadSucceded = previousDownloaded
	})
}

func TestSecurityValidatorUpdateCenter(t *testing.T) {
	isolateSecurityValidator(t)
	data := []byte(securityWarningsJSON)
	serveSecurityWarnings(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/update-center.actual.json", r.URL.Path)
		assert.Empty(t, r.URL.RawQuery)
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write(data)
		assert.NoError(t, err)
	})
	fetched := make(chan bool, 1)
	SecValidator.checkForSecurityVulnerabilities(fetched)
	require.True(t, SecValidator.isCached)
	require.True(t, <-fetched)
	tests := []struct {
		name       string
		plugin     string
		version    string
		vulnerable bool
	}{
		{"vulnerable numeric release", "github", "1.38.0", true},
		{"last affected release", "github", "1.46.0", true},
		{"fixed release", "github", "1.47.0", false},
		{"lookahead affected release", "workflow-cps", "2.61", true},
		{"lookahead patched release", "workflow-cps", "2.61.1", false},
		{"lookahead newer release", "workflow-cps", "2.62", false},
		{"vulnerable hash release", "sonar-quality-gates", "312.vcde1075036ef", true},
		{"fixed hash release", "sonar-quality-gates", "313.v123456789abc", false},
		{"pattern must match the whole version", "sonar-quality-gates", "1312.vcde1075036ef", false},
		{"unlisted plugin", "unlisted", "1.0", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := createJenkinsCR([]Plugin{{Name: tt.plugin, Version: tt.version}}, true)
			_, err := resource.ValidateCreate()
			if tt.vulnerable {
				require.ErrorContains(t, err, tt.plugin+":"+tt.version)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSecurityValidatorFailedRefreshKeepsCache(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		body      string
		wantError string
	}{
		{"forbidden", http.StatusForbidden, "<html>Forbidden</html>", "403"},
		{"server error", http.StatusServiceUnavailable, "Unavailable", "503"},
		{"malformed JSON", http.StatusOK, "{", "JSON"},
		{"JSONP", http.StatusOK, "updateCenter.post({});", "JSON"},
		{"absent warnings", http.StatusOK, `{"plugins":{}}`, "warnings"},
		{"null warnings", http.StatusOK, `{"warnings":null}`, "warnings"},
		{"wrong warnings type", http.StatusOK, `{"warnings":{}}`, "JSON"},
		{"truncated body", http.StatusOK, `{"warnings":[]}`, "read plugin data"},
		{"trailing JSON", http.StatusOK, `{"warnings":[]} {}`, "JSON"},
		{"invalid pattern", http.StatusOK, `{"warnings":[{"type":"plugin","name":"github","id":"broken","versions":[{"pattern":"["}]}]}`, "broken"},
		{"missing pattern", http.StatusOK, `{"warnings":[{"type":"plugin","name":"github","id":"broken","versions":[{}]}]}`, "broken"},
		{"missing plugin name", http.StatusOK, `{"warnings":[{"type":"plugin","id":"broken","versions":[]}]}`, "name"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateSecurityValidator(t)
			data := []byte(securityWarningsJSON)
			status := http.StatusOK
			truncate := false
			serveSecurityWarnings(t, func(w http.ResponseWriter, _ *http.Request) {
				if truncate {
					w.Header().Set("Content-Length", "1000")
				}
				w.WriteHeader(status)
				_, err := w.Write(data)
				assert.NoError(t, err)
			})
			SecValidator.checkForSecurityVulnerabilities(make(chan bool, 1))
			require.True(t, SecValidator.isCached)
			status, data = tt.status, []byte(tt.body)
			truncate = tt.name == "truncated body"
			err := SecValidator.fetchPluginData()
			require.ErrorContains(t, err, tt.wantError)
			require.True(t, SecValidator.isCached)
			vulnerable := createJenkinsCR([]Plugin{{Name: "github", Version: "1.38.0"}}, true)
			_, err = vulnerable.ValidateCreate()
			require.ErrorContains(t, err, "github:1.38.0")
		})
	}
}

func TestSecurityValidatorSuccessfulRefreshReplacesCache(t *testing.T) {
	isolateSecurityValidator(t)
	data := []byte(securityWarningsJSON)
	serveSecurityWarnings(t, func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write(data)
		assert.NoError(t, err)
	})
	SecValidator.checkForSecurityVulnerabilities(make(chan bool, 1))
	require.True(t, SecValidator.isCached)
	data = []byte(`{"warnings":[]}`)
	require.NoError(t, SecValidator.fetchPluginData())
	resource := createJenkinsCR([]Plugin{{Name: "github", Version: "1.38.0"}}, true)
	_, err := resource.ValidateCreate()
	require.NoError(t, err)
}

func TestSecurityValidatorUnboundedWarnings(t *testing.T) {
	isolateSecurityValidator(t)
	serveSecurityWarnings(t, func(w http.ResponseWriter, _ *http.Request) {
		_, err := io.WriteString(w, `{"warnings":[{"type":"plugin","name":"unbounded","id":"all","message":"All versions affected","url":"https://www.jenkins.io/security/"}]}`)
		assert.NoError(t, err)
	})
	SecValidator.checkForSecurityVulnerabilities(make(chan bool, 1))
	require.True(t, SecValidator.isCached)
	resource := createJenkinsCR([]Plugin{{Name: "unbounded", Version: "123.vabcdef"}}, true)
	_, err := resource.ValidateCreate()
	require.ErrorContains(t, err, "unbounded:123.vabcdef")
}

func TestSecurityValidatorConcurrentRefresh(t *testing.T) {
	isolateSecurityValidator(t)
	data := []byte(securityWarningsJSON)
	serveSecurityWarnings(t, func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write(data)
		assert.NoError(t, err)
	})
	SecValidator.checkForSecurityVulnerabilities(make(chan bool, 1))
	require.True(t, SecValidator.isCached)
	var wg sync.WaitGroup
	wg.Go(func() {
		for range 10 {
			assert.NoError(t, SecValidator.fetchPluginData())
		}
	})
	for range 4 {
		wg.Go(func() {
			for range 20 {
				resource := createJenkinsCR([]Plugin{{Name: "github", Version: "1.38.0"}}, true)
				_, err := resource.ValidateCreate()
				assert.ErrorContains(t, err, "github:1.38.0")
			}
		})
	}
	wg.Wait()
}

func TestSecurityValidatorTransportFailure(t *testing.T) {
	isolateSecurityValidator(t)
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	target, err := url.Parse(server.URL)
	require.NoError(t, err)
	transport := server.Client().Transport
	server.Close()
	previous := http.DefaultTransport
	http.DefaultTransport = securityWarningsTransport{target: target, transport: transport}
	t.Cleanup(func() { http.DefaultTransport = previous })
	err = SecValidator.fetchPluginData()
	require.Error(t, err)
	require.False(t, SecValidator.isCached)
	resource := createJenkinsCR([]Plugin{{Name: "github", Version: "1.38.0"}}, true)
	_, err = resource.ValidateCreate()
	require.ErrorContains(t, err, "plugins data has not been fetched")
}

func TestSecurityValidatorPatternBoundaries(t *testing.T) {
	isolateSecurityValidator(t)
	data := fmt.Sprintf(`{"warnings":[{"type":"plugin","name":"sample-plugin","id":"boundary","versions":[{"pattern":%q}]}]}`, `(1|2)[.]0`)
	serveSecurityWarnings(t, func(w http.ResponseWriter, _ *http.Request) {
		_, err := io.Copy(w, strings.NewReader(data))
		assert.NoError(t, err)
	})
	SecValidator.checkForSecurityVulnerabilities(make(chan bool, 1))
	require.True(t, SecValidator.isCached)
	for _, version := range []string{"1.0", "2.0"} {
		resource := createJenkinsCR([]Plugin{{Name: "sample-plugin", Version: version}}, true)
		_, err := resource.ValidateCreate()
		require.ErrorContains(t, err, "sample-plugin:"+version)
	}
	for _, version := range []string{"11.0", "2.0.1", "1.0\n"} {
		resource := createJenkinsCR([]Plugin{{Name: "sample-plugin", Version: version}}, true)
		_, err := resource.ValidateCreate()
		require.NoError(t, err)
	}
}

func TestSecurityValidatorUserHashVersionOfBasePlugin(t *testing.T) {
	isolateSecurityValidator(t)
	serveSecurityWarnings(t, func(w http.ResponseWriter, _ *http.Request) {
		_, err := io.WriteString(w, `{"warnings":[{"type":"plugin","name":"configuration-as-code","id":"hash-example","versions":[{"pattern":"999[.]vbad"}]}]}`)
		assert.NoError(t, err)
	})
	SecValidator.checkForSecurityVulnerabilities(make(chan bool, 1))
	require.True(t, SecValidator.isCached)
	resource := createJenkinsCR([]Plugin{{Name: "configuration-as-code", Version: "999.vbad"}}, true)
	_, err := resource.ValidateCreate()
	require.ErrorContains(t, err, "configuration-as-code:999.vbad")
}

func TestSecurityValidatorBasePluginWarning(t *testing.T) {
	isolateSecurityValidator(t)
	serveSecurityWarnings(t, func(w http.ResponseWriter, _ *http.Request) {
		_, err := io.WriteString(w, `{"warnings":[{"type":"plugin","name":"git","id":"base-example","versions":[{"pattern":".*"}]}]}`)
		assert.NoError(t, err)
	})
	SecValidator.checkForSecurityVulnerabilities(make(chan bool, 1))
	require.True(t, SecValidator.isCached)
	resource := createJenkinsCR(nil, true)
	_, err := resource.ValidateCreate()
	require.ErrorContains(t, err, "base plugins:")
	require.ErrorContains(t, err, "git:")
}

func TestSecurityValidatorMatchingErrorDeniesAdmission(t *testing.T) {
	isolateSecurityValidator(t)
	serveSecurityWarnings(t, func(w http.ResponseWriter, _ *http.Request) {
		_, err := io.WriteString(w, `{"warnings":[{"type":"plugin","name":"expensive","id":"timeout-example","versions":[{"pattern":"(a|aa)+"}]}]}`)
		assert.NoError(t, err)
	})
	SecValidator.checkForSecurityVulnerabilities(make(chan bool, 1))
	require.True(t, SecValidator.isCached)
	resource := createJenkinsCR([]Plugin{{Name: "expensive", Version: strings.Repeat("a", 1000) + "!"}}, true)
	_, err := resource.ValidateCreate()
	require.ErrorContains(t, err, "cannot evaluate security warning timeout-example")
}

func TestSecurityValidatorMultipleWarningsAndRanges(t *testing.T) {
	isolateSecurityValidator(t)
	serveSecurityWarnings(t, func(w http.ResponseWriter, _ *http.Request) {
		_, err := io.WriteString(w, `{"warnings":[{"type":"plugin","name":"sample-plugin","id":"first","versions":[{"pattern":"1[.]0"}]},{"type":"plugin","name":"sample-plugin","id":"second","versions":[{"pattern":"2[.]0"},{"pattern":"3[.]0"}]}]}`)
		assert.NoError(t, err)
	})
	SecValidator.checkForSecurityVulnerabilities(make(chan bool, 1))
	require.True(t, SecValidator.isCached)
	for _, version := range []string{"1.0", "2.0", "3.0"} {
		resource := createJenkinsCR([]Plugin{{Name: "sample-plugin", Version: version}}, true)
		_, err := resource.ValidateCreate()
		require.ErrorContains(t, err, "sample-plugin:"+version)
	}
	resource := createJenkinsCR([]Plugin{{Name: "sample-plugin", Version: "4.0"}}, true)
	_, err := resource.ValidateCreate()
	require.NoError(t, err)
}
