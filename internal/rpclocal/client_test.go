package rpclocal

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestClientPing(t *testing.T) {
	httpClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodGet || req.URL.Path != HealthPath {
				t.Fatalf("request inesperada: %s %s", req.Method, req.URL.Path)
			}
			body := `{"ok":true,"addr":"127.0.0.1:17899","pid":12}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	resp, err := NewClient("127.0.0.1:17899", httpClient).Ping(context.Background())
	if err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if !resp.OK || resp.PID != 12 {
		t.Fatalf("respuesta inesperada: %+v", resp)
	}
}

func TestClientExec(t *testing.T) {
	httpClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPost || req.URL.Path != ExecPath {
				t.Fatalf("request inesperada: %s %s", req.Method, req.URL.Path)
			}
			if got := req.Header.Get(HeaderAuthToken); got != "secret-token" {
				t.Fatalf("header token inesperado: %q", got)
			}
			data, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("ReadAll: %v", err)
			}
			if !bytes.Contains(data, []byte(`"args":["status"]`)) {
				t.Fatalf("payload inesperado: %s", string(data))
			}
			body := `{"stdout":"ok\n","stderr":"","exit_code":0}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	resp, err := NewClient("127.0.0.1:17899", httpClient).WithToken("secret-token").Exec(context.Background(), &ExecRequest{
		Args: []string{"status"},
	})
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if resp.ExitCode != 0 || resp.Stdout != "ok\n" {
		t.Fatalf("respuesta inesperada: %+v", resp)
	}
}
