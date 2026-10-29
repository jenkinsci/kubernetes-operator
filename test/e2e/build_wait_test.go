package e2e

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bndr/gojenkins"
	"github.com/stretchr/testify/require"
)

func TestWaitForJobToFinishTracksQueuedBuild(t *testing.T) {
	var queuePolls, buildPolls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/queue/item/17/api/json":
			if queuePolls.Add(1) == 1 {
				_, _ = fmt.Fprint(w, `{"id":17,"executable":{"number":0}}`)
				return
			}
			_, _ = fmt.Fprint(w, `{"id":17,"executable":{"number":42}}`)
		case "/job/example/42/api/json":
			if buildPolls.Add(1) == 1 {
				_, _ = fmt.Fprint(w, `{"number":42,"building":true,"result":null}`)
				return
			}
			_, _ = fmt.Fprint(w, `{"number":42,"building":false,"result":"SUCCESS"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	jenkins := gojenkins.CreateJenkins(server.Client(), server.URL)
	job := &gojenkins.Job{Jenkins: jenkins, Raw: &gojenkins.JobResponse{URL: server.URL + "/job/example"}}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	build, err := waitForJobToFinish(ctx, job, 17)
	require.NoError(t, err)
	require.NotNil(t, build)
	require.Equal(t, int64(42), build.GetBuildNumber())
	require.Equal(t, "SUCCESS", build.GetResult())
}

func TestWaitForJobToFinishReportsFailedBuild(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/queue/item/17/api/json":
			_, _ = fmt.Fprint(w, `{"id":17,"executable":{"number":42}}`)
		case "/job/example/42/api/json":
			_, _ = fmt.Fprint(w, `{"number":42,"building":false,"result":"FAILURE"}`)
		case "/job/example/42/consoleText", "/job/example/42/consoleText/":
			_, _ = fmt.Fprint(w, "fixture build failed")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	jenkins := gojenkins.CreateJenkins(server.Client(), server.URL)
	job := &gojenkins.Job{Jenkins: jenkins, Raw: &gojenkins.JobResponse{URL: server.URL + "/job/example"}}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := waitForJobToFinish(ctx, job, 17)
	require.ErrorContains(t, err, "build 42 finished with FAILURE")
	require.ErrorContains(t, err, "fixture build failed")
}

func TestWaitForJobToFinishBoundsQueueWait(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `{"id":17,"executable":{"number":0}}`)
	}))
	defer server.Close()
	jenkins := gojenkins.CreateJenkins(server.Client(), server.URL)
	job := &gojenkins.Job{Jenkins: jenkins, Raw: &gojenkins.JobResponse{URL: server.URL + "/job/example"}}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := waitForJobToFinish(ctx, job, 17)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}
