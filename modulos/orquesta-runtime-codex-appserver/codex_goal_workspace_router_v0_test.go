package orquestaruntimecodexappserver

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
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
		EvidenceRefs:   []string{"evidence-ref-test-workspace"},
	}}
	backend := serverCodexAppServerGoalBackendV0{Protocol: protocol, CWD: t.TempDir(), WorkspaceRouter: router}
	receipt, err := backend.StartCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef: "goal-ref-workspace-001", Objective: "cambio aislado",
	})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if protocol.startParams.CWD != filepath.Clean(workspace) || protocol.turnParams.CWD != filepath.Clean(workspace) {
		t.Fatalf("start cwd=%q turn cwd=%q", protocol.startParams.CWD, protocol.turnParams.CWD)
	}
	if !codexGoalWorkspaceContainsForTestV0(receipt.EvidenceRefs, "evidence-ref-test-workspace") || router.prepared != 1 {
		t.Fatalf("receipt=%+v router=%+v", receipt, router)
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

func TestCodexGoalWorkspaceRouterV0BindsExternalGoalBeforeReturningAuthoritativeLaunchV0(t *testing.T) {
	workspace := t.TempDir()
	goalRef := "goal-ref-workspace-authority-bind-001"
	router := &fakeCodexGoalWorkspaceRouterV0{binding: GoalWorkspaceBindingV0{ProjectWorkDir: workspace}}
	protocol := &fakeCodexAppServerProtocolV0{
		thread: serverCodexAppServerThreadV0{ID: "thread-ref-workspace-authority-bind-001"},
		goal:   serverCodexAppServerThreadGoalV0{ThreadID: "thread-ref-workspace-authority-bind-001", Status: "active"},
		turn:   serverCodexAppServerTurnV0{ID: "turn-ref-workspace-authority-bind-001", Status: "inProgress"},
	}
	packet := orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef: goalRef, Objective: "bind authoritative launch",
		WorkspaceRef:         orquestagoal.GoalWorkspaceRefForGoalV0(goalRef),
		ProviderRef:          orquestaruntimecodexgoal.CodexGoalProviderRefV0,
		IntentManifestRef:    "intent-manifest-ref-workspace-authority-bind-001",
		IntentManifestSHA256: strings.Repeat("a", 64),
	}
	receipt, err := (serverCodexAppServerGoalBackendV0{Protocol: protocol, CWD: t.TempDir(), Runtime: &serverCodexAppServerGoalRuntimeV0{}, WorkspaceRouter: router}).StartCodexGoalV0(context.Background(), packet)
	if err != nil || router.bound != 1 || router.lastBound.ExternalGoalRef != receipt.ExternalGoalRef ||
		router.lastBound.RuntimeGenerationRef != receipt.RuntimeGenerationRef ||
		router.lastBound.WorkspaceRef != packet.WorkspaceRef || router.lastBound.IntentManifestSHA256 != packet.IntentManifestSHA256 {
		t.Fatalf("receipt=%+v router=%+v err=%v", receipt, router, err)
	}
}

func TestCodexGoalWorkspaceRouterV0BindingFailureStopsBeforeGoalAndTurnV0(t *testing.T) {
	workspace := t.TempDir()
	goalRef := "goal-ref-workspace-authority-bind-failure-001"
	generationRef := "runtime-generation-ref-workspace-authority-bind-failure-001"
	router := &fakeCodexGoalWorkspaceRouterV0{
		binding: GoalWorkspaceBindingV0{ProjectWorkDir: workspace},
		bindErr: errors.New("durable binding unavailable"),
	}
	protocol := &fakeCodexAppServerProtocolV0{
		thread: serverCodexAppServerThreadV0{ID: "thread-ref-workspace-authority-bind-failure-001"},
		goal:   serverCodexAppServerThreadGoalV0{ThreadID: "thread-ref-workspace-authority-bind-failure-001", Status: "active"},
		turn:   serverCodexAppServerTurnV0{ID: "turn-ref-workspace-authority-bind-failure-001", Status: "inProgress"},
	}
	packet := orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef: goalRef, Objective: "do not work before durable binding",
		WorkspaceRef:         orquestagoal.GoalWorkspaceRefForGoalV0(goalRef),
		ProviderRef:          orquestaruntimecodexgoal.CodexGoalProviderRefV0,
		IntentManifestRef:    "intent-manifest-ref-workspace-authority-bind-failure-001",
		IntentManifestSHA256: strings.Repeat("b", 64),
	}
	receipt, err := (serverCodexAppServerGoalBackendV0{
		Protocol: protocol, CWD: t.TempDir(), Runtime: &serverCodexAppServerGoalRuntimeV0{},
		WorkspaceRouter: router, startRuntimeGenerationRef: generationRef,
	}).StartCodexGoalV0(context.Background(), packet)
	if err == nil || receipt.IssueCode != codexGoalWorkspaceUnavailableIssueV0 ||
		receipt.ExternalGoalRef != protocol.thread.ID || receipt.RuntimeGenerationRef != "" {
		t.Fatalf("receipt=%+v err=%v", receipt, err)
	}
	if router.bound != 1 || router.lastBound.ExternalGoalRef != protocol.thread.ID ||
		router.lastBound.RuntimeGenerationRef != generationRef {
		t.Fatalf("router=%+v", router)
	}
	if len(protocol.calls) != 1 || protocol.calls[0] != "thread/start" ||
		protocol.setParams.ThreadID != "" || protocol.turnParams.ThreadID != "" {
		t.Fatalf("work started before durable binding: calls=%v set=%+v turn=%+v", protocol.calls, protocol.setParams, protocol.turnParams)
	}
}

func TestCodexGoalWorkspaceRouterV0PreservesDurableAuthorityOnLaterTurnFailureV0(t *testing.T) {
	workspace := t.TempDir()
	goalRef := "goal-ref-workspace-authority-turn-failure-001"
	generationRef := "runtime-generation-ref-workspace-authority-turn-failure-001"
	router := &fakeCodexGoalWorkspaceRouterV0{binding: GoalWorkspaceBindingV0{ProjectWorkDir: workspace}}
	protocol := &fakeCodexAppServerProtocolV0{
		thread:        serverCodexAppServerThreadV0{ID: "thread-ref-workspace-authority-turn-failure-001"},
		goal:          serverCodexAppServerThreadGoalV0{ThreadID: "thread-ref-workspace-authority-turn-failure-001", Status: "active"},
		startTurnErrs: []error{errors.New("turn unavailable")},
	}
	packet := orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef: goalRef, Objective: "preserve durable authority after bind",
		WorkspaceRef:         orquestagoal.GoalWorkspaceRefForGoalV0(goalRef),
		ProviderRef:          orquestaruntimecodexgoal.CodexGoalProviderRefV0,
		IntentManifestRef:    "intent-manifest-ref-workspace-authority-turn-failure-001",
		IntentManifestSHA256: strings.Repeat("c", 64),
	}
	receipt, err := (serverCodexAppServerGoalBackendV0{
		Protocol: protocol, CWD: t.TempDir(), Runtime: &serverCodexAppServerGoalRuntimeV0{},
		WorkspaceRouter: router, startRuntimeGenerationRef: generationRef,
	}).StartCodexGoalV0(context.Background(), packet)
	if err == nil || router.bound != 1 || receipt.ExternalGoalRef != protocol.thread.ID ||
		receipt.RuntimeGenerationRef != generationRef || router.lastBound.RuntimeGenerationRef != generationRef {
		t.Fatalf("receipt=%+v router=%+v err=%v", receipt, router, err)
	}
}

type fakeCodexGoalWorkspaceRouterV0 struct {
	binding   GoalWorkspaceBindingV0
	err       error
	prepared  int
	resolved  int
	bound     int
	lastBound orquestaruntimecodexgoal.CodexGoalObservationRequestV0
	bindErr   error
}

func (router *fakeCodexGoalWorkspaceRouterV0) BindCodexGoalExecutionV0(
	_ context.Context,
	request orquestaruntimecodexgoal.CodexGoalObservationRequestV0,
) error {
	router.bound++
	router.lastBound = request
	return router.bindErr
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
