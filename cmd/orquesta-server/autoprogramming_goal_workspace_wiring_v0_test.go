package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaruntimecodexappserver "orquesta/modulos/orquesta-runtime-codex-appserver"
	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestServerPrepareRunBatchLaunchesGoalsInPhysicalWorkspacesV0(t *testing.T) {
	repo := newCodexGoalWorkspaceAdapterGitRepoV0(t)
	workspaceRoot := codexGoalWorkspaceRootForSourceV0(repo)
	t.Cleanup(func() { _ = os.RemoveAll(workspaceRoot) })
	t.Setenv(
		envGoalRequiredTestAttestationConfigFileV0,
		writeCompleteGoalRequiredTestAttestationConfigForTestV0(t, repo, filepath.Join(t.TempDir(), "attestation-runtime")),
	)
	config := orquestaserver.ConfigV0{
		ProjectWorkDir:                    repo,
		IdleSelfImprovementProjectWorkDir: repo,
		RuntimeWorkDir:                    filepath.Join(t.TempDir(), "runtime"),
		StateDir:                          filepath.Join(t.TempDir(), "state"),
	}
	normalProtocol := &serverPrepareRunWorkspaceProtocolV0{}
	autoprogrammingProtocol := &serverPrepareRunWorkspaceProtocolV0{}
	normalBackend := serverCodexGoalBackendForPrepareRunWorkspaceTestV0(repo, nil, normalProtocol)
	autoprogrammingBackend := serverCodexGoalBackendForPrepareRunWorkspaceTestV0(
		repo,
		codexGoalWorkspaceAdapterV0{
			SourceWorkDir:       repo,
			WorkspaceRoot:       workspaceRoot,
			ProjectRefFallback:  "project-ref-prepare-run-workspace",
			WorktreeRefFallback: "worktree-ref-prepare-run-workspace",
		},
		autoprogrammingProtocol,
	)
	stack, err := buildStackFromProjectConfigWithGoalBackendsV0(
		config,
		normalBackend,
		autoprogrammingBackend,
		serverProjectConfigFileV0{},
	)
	if err != nil {
		t.Fatalf("build stack: %v", err)
	}
	handler, err := buildServerAppHandlerV0(stack)
	if err != nil {
		t.Fatalf("build handler: %v", err)
	}
	payload, err := json.Marshal(serverPrepareRunBatchWorkspaceInputV0())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, orquestamcp.MCPAutoprogrammingPrepareRunHTTPPathV0, bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	var result orquestamcp.MCPAutoprogrammingPrepareRunToolResultV0
	if err := json.NewDecoder(recorder.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v body=%s", err, recorder.Body.String())
	}
	if recorder.Code != http.StatusOK || !result.Accepted {
		t.Fatalf("prepare-run code=%d result=%+v", recorder.Code, result)
	}
	if normalProtocol.starts != 0 {
		t.Fatalf("prepare-run batch uso AppGoal normal: starts=%d", normalProtocol.starts)
	}
	if len(autoprogrammingProtocol.cwds) != 2 {
		t.Fatalf("goals fisicos lanzados=%d cwds=%v", len(autoprogrammingProtocol.cwds), autoprogrammingProtocol.cwds)
	}
	for _, cwd := range autoprogrammingProtocol.cwds {
		if filepath.Clean(cwd) == filepath.Clean(repo) || !serverWorktreeGitOKV0(context.Background(), cwd, "rev-parse", "--is-inside-work-tree") {
			t.Fatalf("goal lanzado fuera de su worktree fisico: cwd=%q repo=%q", cwd, repo)
		}
	}
	selector, ok := stack.Ports.GoalRequiredTestSnapshotObserver.(*serverGoalRequiredTestAttestationWorkspaceSelectorV0)
	if !ok {
		t.Fatalf("snapshot observer=%T", stack.Ports.GoalRequiredTestSnapshotObserver)
	}
	workspaceSnapshots := make([]orquestagoal.GoalRequiredTestFinalSnapshotV0, 0, len(autoprogrammingProtocol.cwds))
	for index := range autoprogrammingProtocol.cwds {
		if index >= len(result.Goals) {
			t.Fatalf("respuesta sin goal %d: %+v", index, result.Goals)
		}
		path := []string{"alpha.go", "beta.go"}[index]
		if err := os.WriteFile(
			filepath.Join(autoprogrammingProtocol.cwds[index], path),
			[]byte(fmt.Sprintf("workspace-%d\n", index+1)),
			0o600,
		); err != nil {
			t.Fatalf("write workspace fixture %d: %v", index, err)
		}
		snapshot, err := selector.CaptureGoalRequiredTestFinalSnapshotV0(context.Background(), orquestagoal.GoalRequiredTestFinalSnapshotRequestV0{
			RunRef:         result.Goals[index].RunRef,
			GoalRef:        result.Goals[index].GoalRef,
			WriteSet:       []orquestagoal.GoalWriteScopeV0{{Path: path}},
			WriteSetSHA256: orquestagoal.GoalWriteSetSHA256V0([]orquestagoal.GoalWriteScopeV0{{Path: path}}),
		})
		if err != nil {
			t.Fatalf("capture workspace snapshot %d: %v", index, err)
		}
		workspaceSnapshots = append(workspaceSnapshots, snapshot)
	}
	for index, snapshot := range workspaceSnapshots {
		path := []string{"alpha.go", "beta.go"}[index]
		canonicalSnapshot, err := selector.Canonical.CaptureGoalRequiredTestFinalSnapshotV0(context.Background(), orquestagoal.GoalRequiredTestFinalSnapshotRequestV0{
			RunRef: "run-ref-canonical-attestation", GoalRef: "goal-ref-canonical-attestation-" + path,
			WriteSet:       []orquestagoal.GoalWriteScopeV0{{Path: path}},
			WriteSetSHA256: orquestagoal.GoalWriteSetSHA256V0([]orquestagoal.GoalWriteScopeV0{{Path: path}}),
		})
		if err != nil || len(snapshot.Hashes) != 1 || len(canonicalSnapshot.Hashes) != 1 || snapshot.Hashes[0].SHA256 == canonicalSnapshot.Hashes[0].SHA256 {
			t.Fatalf("atestacion no leyo la worktree fisica: workspace=%+v canonical=%+v err=%v", snapshot, canonicalSnapshot, err)
		}
	}
}

func TestGoalRequiredTestAttestationWorkspaceSelectorResolvesAfterRestartAndFailsClosedV0(t *testing.T) {
	repo := newCodexGoalWorkspaceAdapterGitRepoV0(t)
	workspaceRoot := codexGoalWorkspaceRootForSourceV0(repo)
	t.Cleanup(func() { _ = os.RemoveAll(workspaceRoot) })
	t.Setenv(
		envGoalRequiredTestAttestationConfigFileV0,
		writeCompleteGoalRequiredTestAttestationConfigForTestV0(t, repo, filepath.Join(t.TempDir(), "attestation-runtime")),
	)
	adapter := codexGoalWorkspaceAdapterV0{
		SourceWorkDir: repo, WorkspaceRoot: workspaceRoot,
		ProjectRefFallback: "project-ref-attestation-restart", WorktreeRefFallback: "worktree-ref-attestation-restart",
	}
	packet := orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef: "goal-ref-attestation-restart", RequestRef: "run-ref-attestation-restart", ProjectRef: "project-ref-attestation-restart",
	}
	binding, err := adapter.PrepareCodexGoalWorkspaceV0(context.Background(), packet)
	if err != nil {
		t.Fatalf("prepare workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(binding.ProjectWorkDir, "workspace-only.txt"), []byte("workspace\n"), 0o600); err != nil {
		t.Fatalf("write workspace fixture: %v", err)
	}
	restarted := codexGoalWorkspaceAdapterV0{
		SourceWorkDir: adapter.SourceWorkDir, WorkspaceRoot: adapter.WorkspaceRoot,
		ProjectRefFallback: adapter.ProjectRefFallback, WorktreeRefFallback: adapter.WorktreeRefFallback,
	}
	selector, err := goalRequiredTestAttestationWorkspaceSelectorFromConfigV0(
		orquestaserver.ConfigV0{ProjectWorkDir: repo},
		serverProjectConfigFileV0{},
		restarted,
	)
	if err != nil {
		t.Fatalf("selector: %v", err)
	}
	request := orquestagoal.GoalRequiredTestFinalSnapshotRequestV0{
		RunRef: "run-ref-attestation-restart", GoalRef: packet.GoalRef,
		WriteSet:       []orquestagoal.GoalWriteScopeV0{{Path: "workspace-only.txt"}},
		WriteSetSHA256: orquestagoal.GoalWriteSetSHA256V0([]orquestagoal.GoalWriteScopeV0{{Path: "workspace-only.txt"}}),
	}
	workspaceSnapshot, err := selector.CaptureGoalRequiredTestFinalSnapshotV0(context.Background(), request)
	if err != nil {
		t.Fatalf("capture restarted workspace: %v", err)
	}
	canonicalSnapshot, err := selector.CaptureGoalRequiredTestFinalSnapshotV0(context.Background(), orquestagoal.GoalRequiredTestFinalSnapshotRequestV0{
		RunRef: "run-ref-attestation-canonical", GoalRef: "goal-ref-attestation-canonical",
		WriteSet:       []orquestagoal.GoalWriteScopeV0{{Path: "workspace-only.txt"}},
		WriteSetSHA256: orquestagoal.GoalWriteSetSHA256V0([]orquestagoal.GoalWriteScopeV0{{Path: "workspace-only.txt"}}),
	})
	if err != nil || len(workspaceSnapshot.Hashes) != 1 || len(canonicalSnapshot.Hashes) != 1 || workspaceSnapshot.Hashes[0].SHA256 == canonicalSnapshot.Hashes[0].SHA256 {
		t.Fatalf("workspace=%+v canonical=%+v err=%v", workspaceSnapshot, canonicalSnapshot, err)
	}
	if err := os.RemoveAll(binding.ProjectWorkDir); err != nil {
		t.Fatalf("remove workspace: %v", err)
	}
	if _, err := selector.CaptureGoalRequiredTestFinalSnapshotV0(context.Background(), request); err == nil {
		t.Fatal("capture debe fallar si un indice persistido no resuelve su workspace")
	}
	if _, err := selector.AttestGoalRequiredTestsV0(context.Background(), orquestagoal.GoalRequiredTestAttestationRequestV0{
		RunRef: request.RunRef, GoalRef: request.GoalRef, FinalSnapshot: workspaceSnapshot,
	}); err == nil {
		t.Fatal("attest debe fallar si un indice persistido no resuelve su workspace")
	}
}

func serverCodexGoalBackendForPrepareRunWorkspaceTestV0(
	workDir string,
	router orquestaruntimecodexappserver.GoalWorkspaceRouterPortV0,
	protocol *serverPrepareRunWorkspaceProtocolV0,
) serverCodexGoalBackendV0 {
	client := serverCodexAppServerGoalBackendV0{
		Protocol:        protocol,
		CWD:             workDir,
		Sandbox:         "workspace-write",
		Runtime:         &serverCodexAppServerGoalRuntimeV0{},
		WorkspaceRouter: router,
	}
	return serverCodexGoalBackendV0{Starter: client, Observer: client, Controller: client}
}

func serverPrepareRunBatchWorkspaceInputV0() orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0 {
	return orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID: "request-prepare-run-workspace-batch",
		AutoprogrammingRequest: orquestaautoprogramming.AutoprogrammingRequestV0{
			RequestRef:       "request-prepare-run-workspace-batch",
			ProjectRef:       "project-ref-prepare-run-workspace",
			WorktreeRef:      "worktree-ref-prepare-run-workspace",
			WorktreeIsolated: true,
			BranchRef:        "branch-ref-prepare-run-workspace",
			Tasks: []orquestaautoprogramming.AutoprogrammingTaskGroupCandidateV0{
				serverPrepareRunWorkspaceTaskV0("alpha", "alpha.go"),
				serverPrepareRunWorkspaceTaskV0("beta", "beta.go"),
			},
			WriteSet:      []string{"alpha.go", "beta.go"},
			RequiredTests: []string{"test -d ."},
		},
	}
}

func serverPrepareRunWorkspaceTaskV0(suffix, path string) orquestaautoprogramming.AutoprogrammingTaskGroupCandidateV0 {
	return orquestaautoprogramming.AutoprogrammingTaskGroupCandidateV0{
		TaskRef:   "task-ref-prepare-run-workspace-" + suffix,
		Area:      "workspace-" + suffix,
		Title:     "Workspace " + suffix,
		Objective: "Comprobar que este goal usa su worktree fisico.",
		ContextRefs: []string{
			"goal_migration:goal-first",
			"goal_capability:starter",
			"goal_capability:observer",
			"goal_capability:closure-validator",
		},
		AcceptanceCriteria: []string{"el goal queda aislado"},
		WriteSet:           []string{path},
		RequiredTests:      []string{"test -d ."},
	}
}

type serverPrepareRunWorkspaceProtocolV0 struct {
	starts int
	cwds   []string
}

func (protocol *serverPrepareRunWorkspaceProtocolV0) StartThreadV0(
	_ context.Context,
	params serverCodexAppServerThreadStartParamsV0,
) (serverCodexAppServerThreadV0, error) {
	protocol.starts++
	protocol.cwds = append(protocol.cwds, params.CWD)
	return serverCodexAppServerThreadV0{ID: fmt.Sprintf("thread-ref-workspace-%d", protocol.starts)}, nil
}

func (protocol *serverPrepareRunWorkspaceProtocolV0) UpdateThreadSettingsV0(
	context.Context,
	serverCodexAppServerThreadSettingsUpdateParamsV0,
) error {
	return nil
}

func (protocol *serverPrepareRunWorkspaceProtocolV0) SetGoalV0(
	_ context.Context,
	params serverCodexAppServerThreadGoalSetParamsV0,
) (serverCodexAppServerThreadGoalV0, error) {
	return serverCodexAppServerThreadGoalV0{ThreadID: params.ThreadID, Status: "active"}, nil
}

func (protocol *serverPrepareRunWorkspaceProtocolV0) StartTurnV0(
	_ context.Context,
	params serverCodexAppServerTurnStartParamsV0,
) (serverCodexAppServerTurnV0, error) {
	return serverCodexAppServerTurnV0{ID: "turn-" + params.ThreadID, Status: "inProgress"}, nil
}

func (protocol *serverPrepareRunWorkspaceProtocolV0) GetGoalV0(
	context.Context,
	string,
) (*serverCodexAppServerThreadGoalV0, error) {
	return nil, nil
}

func (protocol *serverPrepareRunWorkspaceProtocolV0) ReadThreadV0(
	context.Context,
	string,
	bool,
) (serverCodexAppServerThreadReadV0, error) {
	return serverCodexAppServerThreadReadV0{}, nil
}

var _ orquestaruntimecodexgoal.CodexGoalStarterPortV0 = serverCodexAppServerGoalBackendV0{}
var _ orquestagoal.GoalWorkLauncherPortV0 = serverAutoprogrammingGoalWorkspaceLauncherV0{}
