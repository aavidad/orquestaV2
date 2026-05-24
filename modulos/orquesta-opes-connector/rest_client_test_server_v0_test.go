package orquestaopesconnector

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type restClientTestTransportV0 struct {
	t       *testing.T
	handler http.Handler
}

func newRESTClientForTestV0(t *testing.T, handler http.Handler) RESTClientV0 {
	t.Helper()
	return NewRESTClientV0(RESTClientConfigV0{
		BaseURL: "http://opes.test",
		HTTPClient: &http.Client{
			Transport: restClientTestTransportV0{t: t, handler: handler},
		},
	})
}

func (transport restClientTestTransportV0) RoundTrip(req *http.Request) (*http.Response, error) {
	transport.t.Helper()
	body, err := readRequestBodyForTestV0(req)
	if err != nil {
		return nil, err
	}
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.RequestURI = ""
	recorder := httptest.NewRecorder()
	transport.handler.ServeHTTP(recorder, req)
	return recorder.Result(), nil
}

func readRequestBodyForTestV0(req *http.Request) ([]byte, error) {
	if req.Body == nil {
		return nil, nil
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	_ = req.Body.Close()
	return body, nil
}
