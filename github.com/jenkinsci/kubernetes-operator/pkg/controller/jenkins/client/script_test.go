package client

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bndr/gojenkins"
	"github.com/stretchr/testify/assert"
)

func Test_ExecuteScript(t *testing.T) {
	verifier := "verifier-text"
	t.Run("logs have verifier text", func(t *testing.T) {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
			if strings.Contains(request.URL.Path, "/scriptText") {
				_, _ = fmt.Fprint(responseWriter, "some output\n"+verifier)
				return
			}
			responseWriter.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()

		client := ts.Client()

		jenkinsClient := &jenkins{}
		jenkinsClient.Server = ts.URL
		jenkinsClient.Requester = &gojenkins.Requester{
			Base:      ts.URL,
			SslVerify: true,
			Client:    client,
			BasicAuth: &gojenkins.BasicAuth{Username: "unused", Password: "unused"},
		}

		script := "some groovy code"
		logs, err := jenkinsClient.executeScript(script, verifier)
		assert.NoError(t, err, logs)
	})
	t.Run("logs don't have verifier text", func(t *testing.T) {
		response := "some exception stack trace without verifier"
		ts := httptest.NewTLSServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
			if strings.Contains(request.URL.Path, "/scriptText") {
				_, _ = fmt.Fprint(responseWriter, response)
				return
			}
			responseWriter.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()

		client := ts.Client()

		jenkinsClient := &jenkins{}
		jenkinsClient.Server = ts.URL
		jenkinsClient.Requester = &gojenkins.Requester{
			Base:      ts.URL,
			SslVerify: true,
			Client:    client,
			BasicAuth: &gojenkins.BasicAuth{Username: "unused", Password: "unused"},
		}

		script := "some groovy code"
		logs, err := jenkinsClient.executeScript(script, verifier)
		assert.EqualError(t, err, "script execution failed", logs)
		assert.Equal(t, response, logs)
	})
	t.Run("throw 500", func(t *testing.T) {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
			responseWriter.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()

		client := ts.Client()

		jenkinsClient := &jenkins{}
		jenkinsClient.Server = ts.URL
		jenkinsClient.Requester = &gojenkins.Requester{
			Base:      ts.URL,
			SslVerify: true,
			Client:    client,
			BasicAuth: &gojenkins.BasicAuth{Username: "unused", Password: "unused"},
		}

		script := "some groovy code"
		logs, err := jenkinsClient.executeScript(script, verifier)
		assert.EqualError(t, err, "invalid status code '500', logs ''", logs)
	})
}
