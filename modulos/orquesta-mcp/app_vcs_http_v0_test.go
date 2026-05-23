package orquestamcp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMCPAppVCSHTTPHandlerV0PostDelegaYPropagaCorrelacion(t *testing.T) {
	executor := &fakeMCPAppVCSExecutorV0{}
	body, err := json.Marshal(MCPAppVCSToolInputV0{
		RequestID:     "request-ref-app-vcs-http-001",
		CorrelationID: "corr-app-vcs-http-001",
		Action:        MCPAppVCSActionCommitV0,
		AppRef:        "app-ref-vcs-http-001",
		RepoRef:       "repo-ref-vcs-http-001",
		CommitMessage: "test: commit local",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, MCPAppVCSHTTPPathV0, bytes.NewReader(body))
	response := httptest.NewRecorder()

	NewMCPAppVCSHTTPHandlerV0(executor).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if response.Header().Get("X-Correlation-ID") != "corr-app-vcs-http-001" {
		t.Fatalf("correlation header=%q", response.Header().Get("X-Correlation-ID"))
	}
	if executor.input.AppRef != "app-ref-vcs-http-001" {
		t.Fatalf("input no delegado=%+v", executor.input)
	}
}
