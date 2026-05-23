package orquestamcp

import (
	"context"
	"strings"
	"testing"
)

func TestMCPAppVCSDescriptorV0EsAdaptadorFino(t *testing.T) {
	descriptor := MCPAppVCSDescriptorV0()
	if descriptor.Name != MCPAppVCSToolNameV0 ||
		descriptor.ResourceURI != MCPAppVCSResourceURIV0 ||
		!containsMCPTestStringV0(descriptor.InputSchema, MCPAppVCSActionReviewRepoV0) ||
		!containsMCPTestStringV0(descriptor.InputSchema, MCPAppVCSActionCommitV0) ||
		!containsMCPTestStringV0(descriptor.InputSchema, MCPAppVCSActionPushV0) {
		t.Fatalf("descriptor inesperado: %+v", descriptor)
	}
}

func TestMCPAppVCSExecutorV0ValidaRefsYOptInPush(t *testing.T) {
	result, err := NewMCPAppVCSToolExecutorV0(&fakeMCPAppVCSExecutorV0{}).Execute(
		context.Background(),
		MCPAppVCSToolInputV0{Action: MCPAppVCSActionPushV0, AppRef: "app-ref-1", RepoRef: "repo-ref-1"},
	)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Estado != MCPAppVCSEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != MCPAppVCSPushOptInRequiredV0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPAppVCSExecutorV0DelegaEnConectorInyectado(t *testing.T) {
	fake := &fakeMCPAppVCSExecutorV0{}
	result, err := NewMCPAppVCSToolExecutorV0(fake).Execute(context.Background(), MCPAppVCSToolInputV0{
		RequestID:     "request-ref-vcs-001",
		CorrelationID: "corr-vcs-001",
		Action:        MCPAppVCSActionCommitV0,
		AppRef:        "app-ref-vcs-001",
		RepoRef:       "repo-ref-vcs-001",
		WorktreeRef:   "worktree-ref-vcs-001",
		BranchRef:     "branch-ref-vcs-001",
		CommitMessage: "test: commit local",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if fake.input.AppRef != "app-ref-vcs-001" ||
		result.Estado != MCPAppVCSEstadoOKV0 ||
		result.CommitRef != "commit-ref-vcs-001" {
		t.Fatalf("fake=%+v result=%+v", fake.input, result)
	}
}

func TestMCPAppVCSExecutorV0AceptaReviewRepo(t *testing.T) {
	fake := &fakeMCPAppVCSExecutorV0{}
	result, err := NewMCPAppVCSToolExecutorV0(fake).Execute(context.Background(), MCPAppVCSToolInputV0{
		RequestID:     "request-ref-vcs-review-001",
		CorrelationID: "corr-vcs-review-001",
		Action:        MCPAppVCSActionReviewRepoV0,
		AppRef:        "app-ref-vcs-review-001",
		RepoRef:       "repo-ref-vcs-review-001",
		WorktreeRef:   "worktree-ref-orquesta-autoprog-git-review-20260523",
		BranchRef:     "branch-ref-orquesta-autoprog-git-review-20260523",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Estado != MCPAppVCSEstadoOKV0 ||
		fake.input.Action != MCPAppVCSActionReviewRepoV0 ||
		result.WorktreeRef != "worktree-ref-orquesta-autoprog-git-review-20260523" ||
		result.BranchRef != "branch-ref-orquesta-autoprog-git-review-20260523" {
		t.Fatalf("fake=%+v result=%+v", fake.input, result)
	}
}

type fakeMCPAppVCSExecutorV0 struct {
	input MCPAppVCSToolInputV0
}

func (fake *fakeMCPAppVCSExecutorV0) Execute(
	_ context.Context,
	input MCPAppVCSToolInputV0,
) (MCPAppVCSToolResultV0, error) {
	fake.input = input
	return MCPAppVCSToolResultV0{
		Estado:         MCPAppVCSEstadoOKV0,
		RequestID:      input.RequestID,
		CorrelationID:  input.CorrelationID,
		Action:         input.Action,
		Status:         "completed",
		AppRef:         input.AppRef,
		RepoRef:        input.RepoRef,
		WorktreeRef:    input.WorktreeRef,
		BranchRef:      input.BranchRef,
		CommitRef:      "commit-ref-vcs-001",
		CommitShortRef: "commit-ref-v",
		ChangedPaths:   []string{"README.md"},
		EvidenceRefs:   []string{"evidence-ref-app-vcs-v0"},
	}, nil
}

func containsMCPTestStringV0(value string, want string) bool {
	return strings.Contains(value, want)
}
