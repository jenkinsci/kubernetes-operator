package client

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetNodeSecretAuthorization(t *testing.T) {
	for _, bearer := range []bool{false, true} {
		name := "basic"
		if bearer {
			name = "bearer"
		}
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if bearer {
					assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
				} else {
					user, password, ok := r.BasicAuth()
					assert.True(t, ok)
					assert.Equal(t, "operator", user)
					assert.Equal(t, "test-password", password)
				}
				switch r.URL.Path {
				case "/jenkins/api/json":
					_, _ = fmt.Fprint(w, `{}`)
				case "/jenkins/crumbIssuer/api/json/api/json":
					w.Header().Set("Content-Type", "application/json")
					_, _ = fmt.Fprint(w, `{"crumbRequestField":"Jenkins-Crumb","crumb":"test-crumb"}`)
				case "/jenkins/scriptText":
					assert.Equal(t, http.MethodPost, r.Method)
					assert.Equal(t, "test-crumb", r.Header.Get("Jenkins-Crumb"))
					assert.NoError(t, r.ParseForm())
					assert.Equal(t, `println(jenkins.model.Jenkins.getInstance().getComputer("agent&one").getJnlpMac())`, r.Form.Get("script"))
					_, _ = fmt.Fprint(w, "agent-secret\n")
				default:
					t.Errorf("unexpected path: %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()
			var api Jenkins
			var err error
			if bearer {
				api, err = NewBearerTokenAuthorization(server.URL+"/jenkins", "test-token")
			} else {
				api, err = NewUserAndPasswordAuthorization(server.URL+"/jenkins", "operator", "test-password")
			}
			require.NoError(t, err)
			secret, err := api.GetNodeSecret("agent&one")
			require.NoError(t, err)
			assert.Equal(t, "agent-secret\n", secret)
		})
	}
}
