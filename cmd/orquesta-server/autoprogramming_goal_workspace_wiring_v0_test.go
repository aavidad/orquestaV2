package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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
			IntentManifestStore: serverAutoprogrammingIntentManifestStoreV0{RootDir: filepath.Join(config.StateDir, "autoprogramming-intent-manifests")},
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
	input := serverPrepareRunBatchWorkspaceInputV0()
	for index := range input.AutoprogrammingRequest.Tasks {
		input.AutoprogrammingRequest.Tasks[index].ContextRefs = []string{"source_surface:operator-request"}
	}
	input.AutoprogrammingRequest.Tasks[0].Context = []string{strings.Repeat("contexto-á漢🙂-", 1400)}
	input.AutoprogrammingRequest.Tasks[0].AcceptanceCriteria = nil
	for index := range 20 {
		input.AutoprogrammingRequest.Tasks[0].AcceptanceCriteria = append(
			input.AutoprogrammingRequest.Tasks[0].AcceptanceCriteria,
			fmt.Sprintf("criterio original %02d á漢🙂", index+1),
		)
	}
	wantManifest, manifestIssues := orquestaautoprogramming.BuildAutoprogrammingIntentManifestV0(input.AutoprogrammingRequest)
	if len(manifestIssues) != 0 || len(wantManifest.RequestJSON) <= 14*1024 {
		t.Fatalf("large manifest fixture invalid: bytes=%d issues=%+v", len(wantManifest.RequestJSON), manifestIssues)
	}
	if bytes.Contains(wantManifest.RequestJSON, []byte("goal_capability:")) {
		t.Fatalf("fixture already contains runtime markers: %s", wantManifest.RequestJSON)
	}
	payload, err := json.Marshal(input)
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
	durableManifest, err := (serverAutoprogrammingIntentManifestStoreV0{
		RootDir: filepath.Join(config.StateDir, "autoprogramming-intent-manifests"),
	}).LoadAutoprogrammingIntentManifestV0(context.Background(), input.AutoprogrammingRequest.RequestRef)
	if err != nil {
		t.Fatalf("load durable envelope manifest: %v", err)
	}
	for _, cwd := range autoprogrammingProtocol.cwds {
		if filepath.Clean(cwd) == filepath.Clean(repo) || !serverWorktreeGitOKV0(context.Background(), cwd, "rev-parse", "--is-inside-work-tree") {
			t.Fatalf("goal lanzado fuera de su worktree fisico: cwd=%q repo=%q", cwd, repo)
		}
		manifestPath := filepath.Join(cwd, ".orquesta-runtime", "intent-manifests", durableManifest.ManifestRef+".json")
		raw, err := os.ReadFile(manifestPath)
		if err != nil || !bytes.Equal(raw, durableManifest.RequestJSON) || bytes.Contains(raw, []byte("goal_capability:")) {
			t.Fatalf("original intent lost in %q: bytes=%d durable_bytes=%d equal=%v runtime_markers=%v err=%v", manifestPath, len(raw), len(durableManifest.RequestJSON), bytes.Equal(raw, durableManifest.RequestJSON), bytes.Contains(raw, []byte("goal_capability:")), err)
		}
		info, err := os.Stat(manifestPath)
		if err != nil || info.Mode().Perm() != 0o400 {
			t.Fatalf("manifest mode in %q: info=%v err=%v", manifestPath, info, err)
		}
	}
	if len(autoprogrammingProtocol.prompts) != 2 {
		t.Fatalf("turn prompts=%d", len(autoprogrammingProtocol.prompts))
	}
	for _, prompt := range autoprogrammingProtocol.prompts {
		for _, want := range []string{durableManifest.ManifestRef, durableManifest.RequestSHA256, ".orquesta-runtime/intent-manifests/" + durableManifest.ManifestRef + ".json"} {
			if !strings.Contains(prompt, want) {
				t.Fatalf("prompt missing %q: %s", want, prompt)
			}
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
	state := serverAttestationGoalStateWithWorkspaceAuthorityForTestV0(packet.RequestRef, packet.GoalRef)
	selector, err := goalRequiredTestAttestationWorkspaceSelectorFromConfigV0(
		orquestaserver.ConfigV0{ProjectWorkDir: repo},
		serverProjectConfigFileV0{},
		restarted,
		serverAttestationGoalStateStoreForAuthorityTestV0{state: state},
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
	resultRootResolver := serverGoalMaterializedResultWorkspaceRootV0{
		CanonicalRoot:   repo,
		WorkspaceLookup: restarted,
	}
	resolvedRoot, err := resultRootResolver.ResolveGoalMaterializedResultProjectRootV0(
		context.Background(),
		state,
	)
	if err != nil || filepath.Clean(resolvedRoot) != filepath.Clean(binding.ProjectWorkDir) {
		t.Fatalf("result watcher root=%q want=%q err=%v", resolvedRoot, binding.ProjectWorkDir, err)
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
	if _, err := resultRootResolver.ResolveGoalMaterializedResultProjectRootV0(
		context.Background(),
		state,
	); err == nil {
		t.Fatal("result watcher debe fallar si un indice persistido no resuelve su workspace")
	}
	if _, err := selector.AttestGoalRequiredTestsV0(context.Background(), orquestagoal.GoalRequiredTestAttestationRequestV0{
		RunRef: request.RunRef, GoalRef: request.GoalRef, FinalSnapshot: workspaceSnapshot,
	}); err == nil {
		t.Fatal("attest debe fallar si un indice persistido no resuelve su workspace")
	}
}

func TestGoalRequiredTestAttestationWorkspaceSelectorUsesPersistedExecutionAuthorityV0(t *testing.T) {
	repo := newCodexGoalWorkspaceAdapterGitRepoV0(t)
	t.Setenv(
		envGoalRequiredTestAttestationConfigFileV0,
		writeCompleteGoalRequiredTestAttestationConfigForTestV0(t, repo, filepath.Join(t.TempDir(), "attestation-runtime")),
	)
	state := orquestagoal.GoalWorkStateV0{
		RunRef: "run-ref-attestation-authority-001", GoalRef: "goal-ref-attestation-authority-001",
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			ExternalGoalRef:                 "external-goal-ref-attestation-authority-001",
			WorkspaceAuthoritySchemaVersion: orquestagoal.GoalWorkspaceAuthoritySchemaV0,
			WorkspaceRef:                    orquestagoal.GoalWorkspaceRefForGoalV0("goal-ref-attestation-authority-001"),
			ProviderRef:                     "provider-ref-codex",
			RuntimeGenerationRef:            "runtime-generation-ref-attestation-authority-001",
		},
	}
	lookup := &serverAttestationWorkspaceLookupForAuthorityTestV0{
		binding: orquestagoal.GoalWorkspaceBindingV0{ProjectWorkDir: repo},
	}
	selector, err := goalRequiredTestAttestationWorkspaceSelectorFromConfigV0(
		orquestaserver.ConfigV0{ProjectWorkDir: repo},
		serverProjectConfigFileV0{},
		lookup,
		serverAttestationGoalStateStoreForAuthorityTestV0{state: state},
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = selector.Close() })
	adapter, err := selector.adapterForGoalV0(context.Background(), state.RunRef, state.GoalRef)
	if err != nil {
		t.Fatal(err)
	}
	if adapter != selector.Canonical {
		_ = adapter.Close()
	}
	want := orquestagoal.GoalObservationRequestFromStateV0(state)
	got := lookup.request
	if got.GoalRef != want.GoalRef || got.ExternalGoalRef != want.ExternalGoalRef ||
		got.WorkspaceAuthoritySchemaVersion != want.WorkspaceAuthoritySchemaVersion ||
		got.WorkspaceRef != want.WorkspaceRef || got.ProviderRef != want.ProviderRef ||
		got.RuntimeGenerationRef != want.RuntimeGenerationRef {
		t.Fatalf("authority lost: got=%+v want=%+v", got, want)
	}

	selector.GoalStateStore = serverAttestationGoalStateStoreForAuthorityTestV0{err: errors.New("state unavailable")}
	lookup.request = orquestagoal.GoalObservationRequestV0{}
	if _, err := selector.adapterForGoalV0(context.Background(), state.RunRef, state.GoalRef); err == nil {
		t.Fatal("attestation accepted a workspace without persisted execution authority")
	}
	if lookup.request.GoalRef != "" {
		t.Fatalf("workspace resolved before authority: %+v", lookup.request)
	}
}

type serverAttestationGoalStateStoreForAuthorityTestV0 struct {
	state orquestagoal.GoalWorkStateV0
	err   error
}

func (store serverAttestationGoalStateStoreForAuthorityTestV0) SaveGoalWorkStateV0(context.Context, orquestagoal.GoalWorkStateV0) error {
	return nil
}

func (store serverAttestationGoalStateStoreForAuthorityTestV0) LoadGoalWorkStateV0(context.Context, string) (orquestagoal.GoalWorkStateV0, error) {
	return store.state, store.err
}

func serverAttestationGoalStateWithWorkspaceAuthorityForTestV0(runRef string, goalRef string) orquestagoal.GoalWorkStateV0 {
	spec := orquestagoal.GoalWorkSpecV0{RunRef: runRef, GoalRef: goalRef}
	authority := orquestagoal.GoalExecutionAuthorityForProviderV0(spec, "provider-ref-codex")
	externalGoalRef := "external-" + goalRef
	return orquestagoal.GoalWorkStateV0{
		RunRef: runRef, GoalRef: goalRef, ExternalGoalRef: externalGoalRef, Spec: spec,
		LaunchReceipt: orquestagoal.ApplyGoalExecutionAuthorityToReceiptV0(
			orquestagoal.GoalLaunchReceiptV0{ExternalGoalRef: externalGoalRef},
			authority,
		),
	}
}

type serverAttestationWorkspaceLookupForAuthorityTestV0 struct {
	binding orquestagoal.GoalWorkspaceBindingV0
	request orquestagoal.GoalObservationRequestV0
}

func (*serverAttestationWorkspaceLookupForAuthorityTestV0) HasGoalWorkspaceBindingV0(context.Context, string) (bool, error) {
	return true, nil
}

func (lookup *serverAttestationWorkspaceLookupForAuthorityTestV0) ResolveGoalWorkspaceV0(_ context.Context, request orquestagoal.GoalObservationRequestV0) (orquestagoal.GoalWorkspaceBindingV0, error) {
	lookup.request = request
	return lookup.binding, nil
}

func TestGoalRequiredTestAttestationWorkspaceSelectorRepeatedCaptureAttestDoesNotLeakFDsV0(t *testing.T) {
	repo := newCodexGoalWorkspaceAdapterGitRepoV0(t)
	workspaceRoot := codexGoalWorkspaceRootForSourceV0(repo)
	t.Cleanup(func() { _ = os.RemoveAll(workspaceRoot) })
	t.Setenv(
		envGoalRequiredTestAttestationConfigFileV0,
		writeCompleteGoalRequiredTestAttestationConfigForTestV0(t, repo, filepath.Join(t.TempDir(), "attestation-runtime")),
	)
	lookup := codexGoalWorkspaceAdapterV0{
		SourceWorkDir: repo, WorkspaceRoot: workspaceRoot,
		ProjectRefFallback: "project-ref-attestation-fd-cycle", WorktreeRefFallback: "worktree-ref-attestation-fd-cycle",
	}
	packet := orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef: "goal-ref-attestation-fd-cycle", RequestRef: "run-ref-attestation-fd-cycle", ProjectRef: "project-ref-attestation-fd-cycle",
	}
	binding, err := lookup.PrepareCodexGoalWorkspaceV0(context.Background(), packet)
	if err != nil {
		t.Fatalf("prepare workspace: %v", err)
	}
	if filepath.Clean(binding.ProjectWorkDir) == filepath.Clean(repo) {
		t.Fatalf("workspace lookup resolved canonical root: %q", binding.ProjectWorkDir)
	}
	if err := os.WriteFile(filepath.Join(binding.ProjectWorkDir, "workspace-only.txt"), []byte("workspace-only\n"), 0o600); err != nil {
		t.Fatalf("write workspace fixture: %v", err)
	}
	selector, err := goalRequiredTestAttestationWorkspaceSelectorFromConfigV0(
		orquestaserver.ConfigV0{ProjectWorkDir: repo},
		serverProjectConfigFileV0{},
		lookup,
		serverAttestationGoalStateStoreForAuthorityTestV0{
			state: serverAttestationGoalStateWithWorkspaceAuthorityForTestV0(packet.RequestRef, packet.GoalRef),
		},
	)
	if err != nil {
		t.Fatalf("selector: %v", err)
	}
	t.Cleanup(func() { _ = selector.Close() })
	writeSet := []orquestagoal.GoalWriteScopeV0{{Path: "workspace-only.txt"}}
	bound, err := selector.BindGoalRequiredTestSpecV0(context.Background(), orquestagoal.NormalizeGoalWorkSpecV0(orquestagoal.GoalWorkSpecV0{
		RunRef: packet.RequestRef, GoalRef: packet.GoalRef,
		Objective: "Attest the isolated workspace without leaking command descriptors.", DirectorKind: orquestagoal.GoalDirectorKindRuntimeGoalV0,
		WriteSet: writeSet, WriteSetSHA256: orquestagoal.GoalWriteSetSHA256V0(writeSet),
		RequiredTests: []orquestagoal.GoalRequiredTestV0{orquestagoal.FreezeGoalRequiredTestV0(orquestagoal.GoalRequiredTestV0{
			TestRef: "test-ref-attestation-fd-cycle", CommandRef: "command-ref-attestation-fd-cycle", Command: "test -f workspace-only.txt",
		})},
		ClosurePolicy: orquestagoal.GoalClosurePolicyV0{RequireRequiredTests: true, RequireIndependentRequiredTestAttestation: true},
	}))
	if err != nil {
		t.Fatalf("bind: %v", err)
	}
	runCycle := func(cycle int) {
		t.Helper()
		snapshot, err := selector.CaptureGoalRequiredTestFinalSnapshotV0(context.Background(), orquestagoal.GoalRequiredTestFinalSnapshotRequestV0{
			RunRef: bound.RunRef, GoalRef: bound.GoalRef, WriteSet: bound.WriteSet, WriteSetSHA256: bound.WriteSetSHA256,
		})
		if err != nil || len(snapshot.Hashes) != 1 {
			t.Fatalf("capture cycle %d: snapshot=%+v err=%v", cycle, snapshot, err)
		}
		receipts, err := selector.AttestGoalRequiredTestsV0(context.Background(), orquestagoal.GoalRequiredTestAttestationRequestV0{
			RunRef: bound.RunRef, GoalRef: bound.GoalRef,
			ImplementerAgentRef: bound.ImplementerAgentRef, ImplementerCredentialRef: bound.ImplementerCredentialRef,
			AttestorTrustPolicyRef: bound.ClosurePolicy.RequiredAttestorTrustPolicyRef,
			FinalSnapshot:          snapshot, RequiredTests: bound.RequiredTests,
		})
		if err != nil || len(receipts) != 1 || receipts[0].Status != orquestagoal.GoalRequiredTestAttestationStatusPassedV0 || receipts[0].ExitCode != 0 {
			t.Fatalf("attest cycle %d: receipts=%+v err=%v", cycle, receipts, err)
		}
	}

	// Warm the real Git/preflight/test path before accounting descriptors.
	runCycle(-1)
	before, err := serverGoalRequiredTestOpenFDCountV0()
	if err != nil {
		t.Skipf("/proc fd accounting unavailable: %v", err)
	}
	for cycle := 0; cycle < 16; cycle++ {
		runCycle(cycle)
	}
	after, err := serverGoalRequiredTestOpenFDCountV0()
	if err != nil {
		t.Fatal(err)
	}
	// Subprocess fixtures from other package tests may finish closing inherited
	// descriptors during this loop. Only growth is evidence of a leak here.
	if after > before {
		t.Fatalf("workspace Capture+Attest fd leak: before=%d after=%d", before, after)
	}
}

func serverGoalRequiredTestOpenFDCountV0() (int, error) {
	entries, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		return 0, err
	}
	return len(entries), nil
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
	starts  int
	cwds    []string
	prompts []string
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
	protocol.prompts = append(protocol.prompts, params.InputText)
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
