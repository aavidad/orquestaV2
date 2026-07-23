package commands

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

type perRequestCredentialTransport struct {
	base  http.RoundTripper
	token func() string
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (roundTrip roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTrip(request)
}

func (transport perRequestCredentialTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	clone := request.Clone(request.Context())
	clone.Header.Set("Authorization", "Bearer "+transport.token())
	return transport.base.RoundTrip(clone)
}

func TestSDKAcceptsOnlySafeCanonicalBaseURLs(t *testing.T) {
	valid := []string{
		"http://localhost",
		"http://127.0.0.1:8080/",
		"http://[::1]:8080",
		"https://example.com",
		"https://example.com:443/",
		"https://127.0.0.1",
	}
	for _, baseURL := range valid {
		t.Run("valid "+baseURL, func(t *testing.T) {
			client, err := New(Config{BaseURL: baseURL, HTTPClient: http.DefaultClient, MaxResponseBytes: 4096})
			if err != nil || client.baseURL != strings.TrimSuffix(baseURL, "/") {
				t.Fatalf("client=%+v err=%v", client, err)
			}
		})
	}

	invalid := []string{
		"",
		" https://example.com",
		"https://example.com ",
		"ftp://example.com",
		"http://example.com",
		"http://192.0.2.1",
		"http://localhost.",
		"https://",
		"https:example.com",
		"//example.com",
		"https://user:secret@example.com",
		"https://example.com/api",
		"https://example.com?",
		"https://example.com?x=1",
		"https://example.com#",
		"https://example.com#fragment",
		"https://example.com:",
		"https://example.com:0",
		"https://example.com:65536",
		"https://example.com:invalid",
		"https://bad_host",
		"https://-bad.example",
		"https://bad-.example",
	}
	for _, baseURL := range invalid {
		t.Run("invalid "+baseURL, func(t *testing.T) {
			if _, err := New(Config{BaseURL: baseURL, HTTPClient: http.DefaultClient, MaxResponseBytes: 4096}); err == nil ||
				err.Error() != "commandsdk.config_invalid" {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestSDKRejectsInvalidInvocationBeforeNetwork(t *testing.T) {
	valid := Request{
		CommandID: "orquesta.mailbox.mark_delivered", Version: "1", RequestRef: "request:sdk",
		ProjectRef: "project:sdk", ClaimedExecutionRef: "execution:sdk", Payload: json.RawMessage(`{}`),
	}
	tests := []struct {
		name   string
		mutate func(*Request)
	}{
		{name: "command empty", mutate: func(request *Request) { request.CommandID = "" }},
		{name: "command prefix", mutate: func(request *Request) { request.CommandID = "other.status" }},
		{name: "command uppercase", mutate: func(request *Request) { request.CommandID = "orquesta.System.status" }},
		{name: "command empty segment", mutate: func(request *Request) { request.CommandID = "orquesta.system..status" }},
		{name: "command path", mutate: func(request *Request) { request.CommandID = "orquesta.system/status" }},
		{name: "command hyphen", mutate: func(request *Request) { request.CommandID = "orquesta.system-status" }},
		{name: "version empty", mutate: func(request *Request) { request.Version = "" }},
		{name: "version zero", mutate: func(request *Request) { request.Version = "0" }},
		{name: "version leading zero", mutate: func(request *Request) { request.Version = "01" }},
		{name: "version text", mutate: func(request *Request) { request.Version = "v1" }},
		{name: "request empty", mutate: func(request *Request) { request.RequestRef = "" }},
		{name: "request whitespace", mutate: func(request *Request) { request.RequestRef = " request:sdk" }},
		{name: "request control", mutate: func(request *Request) { request.RequestRef = "request:\tsdk" }},
		{name: "request utf8", mutate: func(request *Request) { request.RequestRef = string([]byte{'r', 0xff}) }},
		{name: "project empty", mutate: func(request *Request) { request.ProjectRef = "" }},
		{name: "project newline", mutate: func(request *Request) { request.ProjectRef = "project:\nsdk" }},
		{name: "execution whitespace", mutate: func(request *Request) { request.ClaimedExecutionRef = "execution:sdk " }},
		{name: "payload nil", mutate: func(request *Request) { request.Payload = nil }},
		{name: "payload malformed", mutate: func(request *Request) { request.Payload = json.RawMessage(`{`) }},
		{name: "payload concatenated", mutate: func(request *Request) { request.Payload = json.RawMessage(`{} {}`) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			httpClient := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				calls++
				return nil, errors.New("network must not run")
			})}
			client, err := New(Config{BaseURL: "https://example.com", HTTPClient: httpClient, MaxResponseBytes: 4096})
			if err != nil {
				t.Fatal(err)
			}
			request := valid
			test.mutate(&request)
			if _, err := client.Invoke(context.Background(), request); err == nil ||
				err.Error() != "commandsdk.command_invalid" || calls != 0 {
				t.Fatalf("err=%v calls=%d", err, calls)
			}
		})
	}
}

func TestSDKRejectsRedirectWithoutMutatingCallerHTTPClient(t *testing.T) {
	redirected := 0
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		redirected++
	}))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Redirect(writer, request, target.URL, http.StatusTemporaryRedirect)
	}))
	defer source.Close()

	httpClient := source.Client()
	if httpClient.CheckRedirect != nil {
		t.Fatal("test client unexpectedly has redirect policy")
	}
	client, err := New(Config{BaseURL: source.URL, HTTPClient: httpClient, MaxResponseBytes: 4096})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Invoke(context.Background(), Request{
		CommandID: "orquesta.system.status", Version: "1", RequestRef: "request:redirect",
		ProjectRef: "project:sdk", Payload: json.RawMessage(`{}`),
	})
	if !errors.Is(err, ErrRedirectRejected) || redirected != 0 || httpClient.CheckRedirect != nil {
		t.Fatalf("err=%v redirected=%d caller_policy_changed=%t", err, redirected, httpClient.CheckRedirect != nil)
	}
}

func TestSDKSerializesCanonicalInvocationAndStableErrorEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v1/commands/orquesta.system.status" || request.Header.Get("Authorization") != "Bearer token:test" {
			t.Errorf("request=%s auth=%q", request.URL.Path, request.Header.Get("Authorization"))
		}
		var input map[string]any
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			t.Error(err)
		}
		if input["request_ref"] != "request:sdk" || input["project_ref"] != "project:sdk" {
			t.Errorf("input=%v", input)
		}
		writer.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(writer).Encode(Result{CommandID: "orquesta.system.status", CommandVersion: "1", RequestRef: "request:sdk", Failure: &Failure{Code: CodeForbidden, MessageKey: "error.forbidden"}, AuditRef: "audit:sdk"})
	}))
	defer server.Close()
	httpClient := server.Client()
	httpClient.Transport = perRequestCredentialTransport{base: httpClient.Transport, token: func() string { return "token:test" }}
	client, err := New(Config{BaseURL: server.URL, HTTPClient: httpClient, MaxResponseBytes: 4096})
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Invoke(context.Background(), Request{CommandID: "orquesta.system.status", Version: "1", RequestRef: "request:sdk", ProjectRef: "project:sdk", Payload: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	if result.Failure == nil || result.Failure.Code != CodeForbidden || result.AuditRef != "audit:sdk" {
		t.Fatalf("result=%+v", result)
	}
}

func TestSDKInvokesGenericCommandWithoutPrivateRegistry(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v1/commands/orquesta.future.command" {
			t.Errorf("path=%q", request.URL.Path)
		}
		_ = json.NewEncoder(writer).Encode(Result{
			CommandID: "orquesta.future.command", CommandVersion: "7", RequestRef: "request:future",
		})
	}))
	defer server.Close()
	client, err := New(Config{BaseURL: server.URL, HTTPClient: server.Client(), MaxResponseBytes: 4096})
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Invoke(context.Background(), Request{
		CommandID: "orquesta.future.command", Version: "7", RequestRef: "request:future",
		ProjectRef: "project:future", Payload: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.CommandID != "orquesta.future.command" || result.CommandVersion != "7" {
		t.Fatalf("result=%+v", result)
	}
}

func TestSDKRejectsConcatenatedResponseJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(`{"command_id":"orquesta.system.status","command_version":"1","request_ref":"request:sdk"} {}`))
	}))
	defer server.Close()
	client, err := New(Config{BaseURL: server.URL, HTTPClient: server.Client(), MaxResponseBytes: 4096})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Invoke(context.Background(), Request{CommandID: "orquesta.system.status", Version: "1", RequestRef: "request:sdk", ProjectRef: "project:sdk", Payload: json.RawMessage(`{}`)}); err == nil {
		t.Fatal("concatenated response accepted")
	}
}

func TestSDKRejectsMismatchedEnvelopeAndHTTPStatus(t *testing.T) {
	tests := []struct {
		name   string
		status int
		result Result
	}{
		{name: "command", status: http.StatusOK, result: Result{CommandID: "orquesta.other", CommandVersion: "1", RequestRef: "request:sdk"}},
		{name: "version", status: http.StatusOK, result: Result{CommandID: "orquesta.system.status", CommandVersion: "2", RequestRef: "request:sdk"}},
		{name: "request", status: http.StatusOK, result: Result{CommandID: "orquesta.system.status", CommandVersion: "1", RequestRef: "request:other"}},
		{name: "success status", status: http.StatusInternalServerError, result: Result{CommandID: "orquesta.system.status", CommandVersion: "1", RequestRef: "request:sdk"}},
		{name: "failure status", status: http.StatusOK, result: Result{
			CommandID: "orquesta.system.status", CommandVersion: "1", RequestRef: "request:sdk",
			Failure: &Failure{Code: CodeForbidden, MessageKey: "error.forbidden"},
		}},
		{name: "unknown failure", status: http.StatusInternalServerError, result: Result{
			CommandID: "orquesta.system.status", CommandVersion: "1", RequestRef: "request:sdk",
			Failure: &Failure{Code: "unknown", MessageKey: "error.unknown"},
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(test.status)
				_ = json.NewEncoder(writer).Encode(test.result)
			}))
			defer server.Close()
			client, err := New(Config{BaseURL: server.URL, HTTPClient: server.Client(), MaxResponseBytes: 4096})
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.Invoke(context.Background(), Request{
				CommandID: "orquesta.system.status", Version: "1", RequestRef: "request:sdk",
				ProjectRef: "project:sdk", Payload: json.RawMessage(`{}`),
			})
			if err == nil || err.Error() != "commandsdk.response_invalid" {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestSDKRejectsOversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(`{"command_id":"orquesta.system.status","command_version":"1","request_ref":"request:sdk","data":"oversized"}`))
	}))
	defer server.Close()
	client, err := New(Config{BaseURL: server.URL, HTTPClient: server.Client(), MaxResponseBytes: 32})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Invoke(context.Background(), Request{
		CommandID: "orquesta.system.status", Version: "1", RequestRef: "request:sdk",
		ProjectRef: "project:sdk", Payload: json.RawMessage(`{}`),
	})
	if err == nil || err.Error() != "commandsdk.response_invalid" {
		t.Fatalf("err=%v", err)
	}
}

func TestSDKObtainsCredentialFromRoundTripperForEveryRequest(t *testing.T) {
	wantToken := "token:first"
	seen := make([]string, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		seen = append(seen, request.Header.Get("Authorization"))
		var input struct {
			RequestRef string `json:"request_ref"`
		}
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			t.Error(err)
		}
		_ = json.NewEncoder(writer).Encode(Result{
			CommandID: "orquesta.system.status", CommandVersion: "1", RequestRef: input.RequestRef,
		})
	}))
	defer server.Close()
	httpClient := server.Client()
	httpClient.Transport = perRequestCredentialTransport{base: httpClient.Transport, token: func() string { return wantToken }}
	client, err := New(Config{BaseURL: server.URL, HTTPClient: httpClient, MaxResponseBytes: 4096})
	if err != nil {
		t.Fatal(err)
	}
	for index, token := range []string{"token:first", "token:second"} {
		wantToken = token
		_, err := client.Invoke(context.Background(), Request{
			CommandID: "orquesta.system.status", Version: "1", RequestRef: "request:" + string(rune('a'+index)),
			ProjectRef: "project:sdk", Payload: json.RawMessage(`{}`),
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(seen, []string{"Bearer token:first", "Bearer token:second"}) {
		t.Fatalf("credentials=%v", seen)
	}
}
