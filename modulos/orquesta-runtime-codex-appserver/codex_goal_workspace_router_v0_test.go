package orquestaruntimecodexappserver

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
)

func TestCodexGoalWorkspaceRouterV0UsesGoalSpecificCWDAtStartV0(t *testing.T) {
	workspace := t.TempDir()
	protocol := &fakeCodexAppServerProtocolV0{
		thread: serverCodexAppServerThreadV0{ID: "thread-ref-workspace-001"},
		goal:   serverCodexAppServerThreadGoalV0{ThreadID: "thread-ref-workspace-001", Status: "active"},
		turn:   serverCodexAppServerTurnV0{ID: "turn-ref-workspace-001", Status: "inProgress"},
	}
	router := &fakeCodexGoalWorkspaceRouterV0{binding: GoalWorkspaceBindingV0{
		ProjectWorkDir: workspace,
		WritableRoots:  []string{"  " + workspace + "/generated/../generated  ", workspace + "/generated"},
		EvidenceRefs:   []string{"evidence-ref-test-workspace"},
	}}
	backend := serverCodexAppServerGoalBackendV0{Protocol: protocol, CWD: t.TempDir(), Sandbox: "workspace-write", WorkspaceRouter: router}
	receipt, err := backend.StartCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef: "goal-ref-workspace-001", Objective: "cambio aislado",
	})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if protocol.startParams.CWD != filepath.Clean(workspace) || protocol.turnParams.CWD != filepath.Clean(workspace) {
		t.Fatalf("start cwd=%q turn cwd=%q", protocol.startParams.CWD, protocol.turnParams.CWD)
	}
	policy := protocol.turnParams.SandboxPolicy
	if policy.Type != "workspaceWrite" || len(policy.WritableRoots) != 1 || policy.WritableRoots[0] != filepath.Join(workspace, "generated") {
		t.Fatalf("sandbox policy=%+v", policy)
	}
	if !codexGoalWorkspaceContainsForTestV0(receipt.EvidenceRefs, "evidence-ref-test-workspace") || router.prepared != 1 {
		t.Fatalf("receipt=%+v router=%+v", receipt, router)
	}
}

func TestCodexGoalWorkspaceRouterV0RejectsRelativeWritableRootV0(t *testing.T) {
	router := &fakeCodexGoalWorkspaceRouterV0{binding: GoalWorkspaceBindingV0{
		ProjectWorkDir: t.TempDir(),
		WritableRoots:  []string{"relative/output"},
	}}
	backend := serverCodexAppServerGoalBackendV0{Protocol: &fakeCodexAppServerProtocolV0{}, WorkspaceRouter: router}
	receipt, err := backend.StartCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalStartPacketV0{GoalRef: "goal-ref-relative-root"})
	if err == nil || receipt.IssueCode != codexGoalWorkspaceUnavailableIssueV0 {
		t.Fatalf("receipt=%+v err=%v", receipt, err)
	}
}

func TestCodexGoalWorkspaceRouterV0ResolvesSameWorkspaceAtObserveV0(t *testing.T) {
	workspace := t.TempDir()
	router := &fakeCodexGoalWorkspaceRouterV0{binding: GoalWorkspaceBindingV0{ProjectWorkDir: workspace}}
	goal := &serverCodexAppServerThreadGoalV0{ThreadID: "thread-ref-workspace-002", Status: "active"}
	protocol := &fakeCodexAppServerProtocolV0{observedGoal: goal}
	backend := serverCodexAppServerGoalBackendV0{Protocol: protocol, CWD: t.TempDir(), WorkspaceRouter: router}
	_, err := backend.ObserveCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		GoalRef: "goal-ref-workspace-002", ExternalGoalRef: "thread-ref-workspace-002",
	})
	if err != nil || router.resolved != 1 {
		t.Fatalf("err=%v router=%+v", err, router)
	}
}

func TestCodexGoalWorkspaceRouterV0FailsClosedWithoutBindingV0(t *testing.T) {
	router := &fakeCodexGoalWorkspaceRouterV0{err: errors.New("unavailable")}
	backend := serverCodexAppServerGoalBackendV0{Protocol: &fakeCodexAppServerProtocolV0{}, WorkspaceRouter: router}
	receipt, err := backend.StartCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalStartPacketV0{GoalRef: "goal-ref-workspace-003"})
	if err == nil || receipt.IssueCode != codexGoalWorkspaceUnavailableIssueV0 {
		t.Fatalf("receipt=%+v err=%v", receipt, err)
	}
}

type fakeCodexGoalWorkspaceRouterV0 struct {
	binding  GoalWorkspaceBindingV0
	err      error
	prepared int
	resolved int
}

func (router *fakeCodexGoalWorkspaceRouterV0) PrepareCodexGoalWorkspaceV0(
	context.Context,
	orquestaruntimecodexgoal.CodexGoalStartPacketV0,
) (GoalWorkspaceBindingV0, error) {
	router.prepared++
	return router.binding, router.err
}

func (router *fakeCodexGoalWorkspaceRouterV0) ResolveCodexGoalWorkspaceV0(
	context.Context,
	orquestaruntimecodexgoal.CodexGoalObservationRequestV0,
) (GoalWorkspaceBindingV0, error) {
	router.resolved++
	return router.binding, router.err
}

func codexGoalWorkspaceContainsForTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
