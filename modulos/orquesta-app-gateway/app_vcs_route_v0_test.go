package orquestaappgateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func TestAppVCSAPIDelegaEnExecutorRESTSinRutasLocalesV0(t *testing.T) {
	executor := &recordingAppVCSExecutorV0{}
	handler := NewHTTPHandlerV0(ConfigV0{
		AppVCS:  executor,
		Timeout: time.Second,
	})
	input := orquestamcp.MCPAppVCSToolInputV0{
		RequestID:     "request-ref-app-vcs-gateway-001",
		CorrelationID: "corr-app-vcs-gateway-001",
		Action:        orquestamcp.MCPAppVCSActionCommitV0,
		AppRef:        "app-ref-vcs-gateway-001",
		RepoRef:       "repo-ref-vcs-gateway-001",
		WorktreeRef:   "worktree-ref-app-vcs-workspaces-20260523-06",
		BranchRef:     "branch-ref-app-vcs-workspaces-20260523-06",
		CommitMessage: "test: commit local",
	}
	body, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, orquestamcp.MCPAppVCSHTTPPathV0, strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-Orquesta-Control-Plane-Header-Policy") != "control-plane-json-v0" {
		t.Fatalf("security headers=%v", rec.Header())
	}
	if executor.Input.AppRef != "app-ref-vcs-gateway-001" ||
		executor.Input.WorktreeRef != "worktree-ref-app-vcs-workspaces-20260523-06" {
		t.Fatalf("input no delegado=%+v", executor.Input)
	}
	var result orquestamcp.MCPAppVCSToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Estado != orquestamcp.MCPAppVCSEstadoOKV0 ||
		result.CommitRef != "commit-ref-vcs-gateway-001" {
		t.Fatalf("result=%+v", result)
	}
}

func TestAppVCSAPIPrecedePrefijoAppChangeV0(t *testing.T) {
	appVCS := &recordingAppVCSExecutorV0{}
	appChange := &recordingRequestAppChangeExecutorV0{}
	handler := NewHTTPHandlerV0(ConfigV0{
		AppVCS:           appVCS,
		RequestAppChange: appChange,
		Timeout:          time.Second,
	})
	body, err := json.Marshal(orquestamcp.MCPAppVCSToolInputV0{
		RequestID:     "request-ref-app-vcs-precedence-001",
		CorrelationID: "corr-app-vcs-precedence-001",
		Action:        orquestamcp.MCPAppVCSActionReviewRepoV0,
		AppRef:        "app-ref-vcs-precedence-001",
		RepoRef:       "repo-ref-vcs-precedence-001",
	})
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, orquestamcp.MCPAppVCSHTTPPathV0, strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if appVCS.Input.Action != orquestamcp.MCPAppVCSActionReviewRepoV0 {
		t.Fatalf("app_vcs input=%+v", appVCS.Input)
	}
	if appChange.Input.AppChangeRequest.AppRef != "" {
		t.Fatalf("app_change capturo app_vcs path=%+v", appChange.Input.AppChangeRequest)
	}
}

type recordingAppVCSExecutorV0 struct {
	Input orquestamcp.MCPAppVCSToolInputV0
}

func (executor *recordingAppVCSExecutorV0) Execute(
	_ context.Context,
	input orquestamcp.MCPAppVCSToolInputV0,
) (orquestamcp.MCPAppVCSToolResultV0, error) {
	executor.Input = input
	return orquestamcp.MCPAppVCSToolResultV0{
		Estado:         orquestamcp.MCPAppVCSEstadoOKV0,
		RequestID:      input.RequestID,
		CorrelationID:  input.CorrelationID,
		Action:         input.Action,
		Status:         "completed",
		AppRef:         input.AppRef,
		RepoRef:        input.RepoRef,
		WorktreeRef:    input.WorktreeRef,
		BranchRef:      input.BranchRef,
		CommitRef:      "commit-ref-vcs-gateway-001",
		CommitShortRef: "commit-ref-v",
		ChangedPaths:   []string{"README.md"},
		EvidenceRefs:   []string{"evidence-ref-app-vcs-v0"},
	}, nil
}
