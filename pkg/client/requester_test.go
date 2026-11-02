package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateNodeAcceptsHTMLResponse(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusForbidden} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/json", "/crumbIssuer/api/json/api/json":
					_, _ = fmt.Fprint(w, `{}`)
				case "/computer/doCreateItem":
					assert.Equal(t, http.MethodPost, r.Method)
					assert.Equal(t, "agent", r.URL.Query().Get("name"))
					w.Header().Set("Content-Type", "text/html")
					w.WriteHeader(status)
					_, _ = fmt.Fprint(w, "<html>Jenkins agent page</html>")
				case "/computer/agent/api/json":
					_, _ = fmt.Fprint(w, `{"displayName":"agent","numExecutors":5}`)
				default:
					t.Errorf("unexpected request: %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()
			api, err := NewUserAndPasswordAuthorization(server.URL, "operator", "token")
			require.NoError(t, err)
			node, err := api.CreateNode(context.Background(), "agent", 5, "agent", "/home/jenkins", "agent")
			if status == http.StatusForbidden {
				require.EqualError(t, err, "403")
				require.Nil(t, node)
			} else {
				require.NoError(t, err)
				require.Equal(t, "agent", node.GetName())
			}
		})
	}
}

func TestSafeRestartAcceptsHTMLResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/json", "/crumbIssuer/api/json/api/json":
			_, _ = fmt.Fprint(w, `{}`)
		case "/safeRestart":
			assert.Equal(t, http.MethodPost, r.Method)
			w.Header().Set("Content-Type", "text/html")
			_, _ = fmt.Fprint(w, "<html>Jenkins is restarting</html>")
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	api, err := NewUserAndPasswordAuthorization(server.URL, "operator", "token")
	require.NoError(t, err)
	require.NoError(t, api.SafeRestart(context.Background()))
}

func TestUpdateJobConfigurationAcceptsHTMLResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/json", "/crumbIssuer/api/json/api/json":
			_, _ = fmt.Fprint(w, `{}`)
		case "/job/example/api/json":
			_, _ = fmt.Fprint(w, `{"name":"example"}`)
		case "/job/example/config.xml":
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "application/xml;charset=utf-8", r.Header.Get("Content-Type"))
			w.Header().Set("Content-Type", "text/html")
			_, _ = fmt.Fprint(w, "<html>Configuration updated</html>")
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	api, err := NewUserAndPasswordAuthorization(server.URL, "operator", "token")
	require.NoError(t, err)
	job, err := api.GetJob(context.Background(), "example")
	require.NoError(t, err)
	require.NoError(t, job.UpdateConfig(context.Background(), "<project/>"))
}

func TestGenerateTokenRequiresJSONResponse(t *testing.T) {
	for _, valid := range []bool{true, false} {
		name := "valid"
		if !valid {
			name = "malformed"
		}
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/json", "/crumbIssuer/api/json/api/json":
					_, _ = fmt.Fprint(w, `{}`)
				case "/user/operator/descriptorByName/jenkins.security.ApiTokenProperty/generateNewToken":
					if valid {
						_, _ = fmt.Fprint(w, `{"status":"ok","data":{"tokenName":"test","tokenValue":"generated-token"}}`)
					} else {
						_, _ = fmt.Fprint(w, "<html>Unexpected page</html>")
					}
				default:
					t.Errorf("unexpected request: %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()
			api, err := NewUserAndPasswordAuthorization(server.URL, "operator", "token")
			require.NoError(t, err)
			token, err := api.GenerateToken("operator", "test")
			if valid {
				require.NoError(t, err)
				require.Equal(t, "generated-token", token.GetToken())
			} else {
				require.ErrorContains(t, err, "invalid character")
				require.Nil(t, token)
			}
		})
	}
}

func TestGetNodeReturnsHTTPErrorForHTMLResponse(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusForbidden, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/json" {
					_, _ = fmt.Fprint(w, `{}`)
					return
				}
				assert.Equal(t, "/computer/agent/api/json", r.URL.Path)
				w.Header().Set("Content-Type", "text/html")
				w.WriteHeader(status)
				_, _ = fmt.Fprint(w, "<html>Jenkins error page</html>")
			}))
			defer server.Close()
			api, err := NewUserAndPasswordAuthorization(server.URL, "operator", "token")
			require.NoError(t, err)
			node, err := api.GetNode(context.Background(), "agent")
			require.Nil(t, node)
			if status == http.StatusNotFound {
				require.ErrorIs(t, err, ErrNotFound)
			} else {
				require.ErrorContains(t, err, fmt.Sprint(status))
			}
		})
	}
}
