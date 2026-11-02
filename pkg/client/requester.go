package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bndr/gojenkins"
)

type jenkinsRequester struct {
	*gojenkins.Requester
}

func (r *jenkinsRequester) Post(ctx context.Context, endpoint string, payload io.Reader, response interface{}, query map[string]string) (*http.Response, error) {
	return r.post(ctx, endpoint, payload, "application/x-www-form-urlencoded", response, query)
}

func (r *jenkinsRequester) PostXML(ctx context.Context, endpoint, xml string, response interface{}, query map[string]string) (*http.Response, error) {
	return r.post(ctx, endpoint, strings.NewReader(xml), "application/xml;charset=utf-8", response, query)
}

func (r *jenkinsRequester) post(ctx context.Context, endpoint string, payload io.Reader, contentType string, response interface{}, query map[string]string) (*http.Response, error) {
	request := gojenkins.NewAPIRequest(http.MethodPost, endpoint, payload)
	if err := r.SetCrumb(ctx, request); err != nil {
		return nil, err
	}
	request.SetHeader("Content-Type", contentType)
	switch response.(type) {
	case nil, struct{}:
		var discarded string
		response = &discarded
	}
	return r.Do(ctx, request, response, query)
}

func (r *jenkinsRequester) GetJSON(ctx context.Context, endpoint string, response interface{}, query map[string]string) (*http.Response, error) {
	result, err := r.Requester.GetJSON(ctx, endpoint, response, query)
	if result != nil && result.StatusCode >= http.StatusBadRequest {
		if result.StatusCode == http.StatusNotFound {
			return result, ErrNotFound
		}
		return result, fmt.Errorf("Jenkins API returned HTTP %d", result.StatusCode)
	}
	return result, err
}
