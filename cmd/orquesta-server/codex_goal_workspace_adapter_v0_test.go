package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntimecodexappserver "orquesta/modulos/orquesta-runtime-codex-appserver"
	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
)

func TestCodexGoalWorkspaceAdapterV0ProvisionsAndResolvesAfterRestart(t *testing.T) {
	repo := newCodexGoalWorkspaceAdapterGitRepoV0(t)
	adapter := codexGoalWorkspaceAdapterV0{
		SourceWorkDir:       repo,
		WorkspaceRoot:       filepath.Join(t.TempDir(), "runtime"),
		ProjectRefFallback:  "project-ref-fallback",
		WorktreeRefFallback: "worktree-ref-fallback",
	}
	first, err := adapter.PrepareCodexGoalWorkspaceV0(context.Background(), orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef: "goal-ref-workspace-adapter-001", RequestRef: "run-ref-workspace-adapter-001", ProjectRef: "project-ref-packet-001",
	})
	if err != nil {
		t.Fatalf("first prepare: %v", err)
	}
	second, err := adapter.PrepareCodexGoalWorkspaceV0(context.Background(), orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef: "goal-ref-workspace-adapter-002",
	})
	if err != nil {
		t.Fatalf("second prepare: %v", err)
	}
	if first.ProjectWorkDir == second.ProjectWorkDir {
		t.Fatalf("workspaces must differ: first=%q second=%q", first.ProjectWorkDir, second.ProjectWorkDir)
	}
	if err := os.WriteFile(filepath.Join(first.ProjectWorkDir, "only-first.txt"), []byte("first\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(second.ProjectWorkDir, "only-first.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("second workspace observes first mutation: %v", err)
	}

	restarted := codexGoalWorkspaceAdapterV0{
		SourceWorkDir:       repo,
		WorkspaceRoot:       adapter.WorkspaceRoot,
		ProjectRefFallback:  adapter.ProjectRefFallback,
		WorktreeRefFallback: adapter.WorktreeRefFallback,
	}
	resolved, err := restarted.ResolveCodexGoalWorkspaceV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		GoalRef: "goal-ref-workspace-adapter-001", ExternalGoalRef: "thread-ref-workspace-adapter-001",
		WorkspaceAuthoritySchemaVersion: orquestagoal.GoalWorkspaceAuthoritySchemaV0,
		WorkspaceRef:                    orquestagoal.GoalWorkspaceRefForGoalV0("goal-ref-workspace-adapter-001"),
		ProviderRef:                     orquestaruntimecodexgoal.CodexGoalProviderRefV0, RuntimeGenerationRef: "runtime-generation-ref-workspace-adapter-001",
	})
	if err != nil || resolved.ProjectWorkDir != first.ProjectWorkDir {
		t.Fatalf("resolved=%+v err=%v first=%+v", resolved, err, first)
	}
}

func TestCodexGoalWorkspaceAdapterV0IndiceSoloProyectaAutoridadGoalV0(t *testing.T) {
	repo := newCodexGoalWorkspaceAdapterGitRepoV0(t)
	adapter := codexGoalWorkspaceAdapterV0{
		SourceWorkDir: repo, WorkspaceRoot: filepath.Join(t.TempDir(), "runtime"),
		ProjectRefFallback: "project-ref-authority", WorktreeRefFallback: "worktree-ref-authority",
	}
	spec := orquestagoal.GoalWorkSpecV0{
		GoalRef: "goal-ref-workspace-authority-adapter-001", Objective: "Probar autoridad neutral.",
		WriteSet: []orquestagoal.GoalWriteScopeV0{{Path: "base.txt"}},
	}
	prepared, err := adapter.PrepareGoalWorkspaceV0(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	index, found, err := adapter.loadGoalIndexV0(spec.GoalRef)
	if err != nil || !found || index.Identity.WorkspaceRef != prepared.WorkspaceRef ||
		index.Identity.ProviderRef != "" || index.Identity.RuntimeGenerationRef != "" {
		t.Fatalf("index=%+v prepared=%+v found=%v err=%v", index, prepared, found, err)
	}
	authority := orquestagoal.GoalObservationRequestV0{
		GoalRef: spec.GoalRef, ExternalGoalRef: "thread-ref-workspace-authority-adapter-001",
		WorkspaceAuthoritySchemaVersion: orquestagoal.GoalWorkspaceAuthoritySchemaV0,
		WorkspaceRef:                    prepared.WorkspaceRef,
		ProviderRef:                     orquestaruntimecodexgoal.CodexGoalProviderRefV0,
		RuntimeGenerationRef:            "runtime-generation-ref-authority-001",
	}
	resolved, err := adapter.ResolveGoalWorkspaceV0(context.Background(), authority)
	if err != nil || resolved.WorkspaceRef != authority.WorkspaceRef ||
		resolved.ProviderRef != authority.ProviderRef ||
		resolved.RuntimeGenerationRef != authority.RuntimeGenerationRef ||
		resolved.ProjectWorkDir != prepared.ProjectWorkDir {
		t.Fatalf("resolved=%+v prepared=%+v err=%v", resolved, prepared, err)
	}
	bound, found, err := adapter.loadGoalIndexV0(spec.GoalRef)
	if err != nil || !found || bound.Identity.ProviderRef != authority.ProviderRef ||
		bound.Identity.ExternalGoalRef != authority.ExternalGoalRef ||
		bound.Identity.RuntimeGenerationRef != authority.RuntimeGenerationRef {
		t.Fatalf("bound=%+v found=%v err=%v", bound, found, err)
	}
	if _, err := adapter.ResolveGoalWorkspaceV0(context.Background(), orquestagoal.GoalObservationRequestV0{GoalRef: spec.GoalRef}); !errors.Is(err, errCodexGoalWorkspaceAdapterConflictV0) {
		t.Fatalf("bound v2 accepted request without execution authority: %v", err)
	}

	providerMutation := authority
	providerMutation.ProviderRef = "provider-ref-mutated"
	if _, err := adapter.ResolveGoalWorkspaceV0(context.Background(), providerMutation); !errors.Is(err, errCodexGoalWorkspaceProviderMismatchV0) {
		t.Fatalf("provider mutation err=%v", err)
	}
	generationMutation := authority
	generationMutation.RuntimeGenerationRef = "runtime-generation-ref-mutated"
	if _, err := adapter.ResolveGoalWorkspaceV0(context.Background(), generationMutation); !errors.Is(err, errCodexGoalWorkspaceGenerationMismatchV0) {
		t.Fatalf("generation mutation err=%v", err)
	}
	externalMutation := authority
	externalMutation.ExternalGoalRef = "thread-ref-workspace-authority-adapter-mutated"
	if _, err := adapter.ResolveGoalWorkspaceV0(context.Background(), externalMutation); !errors.Is(err, errCodexGoalWorkspaceExternalGoalMismatchV0) {
		t.Fatalf("external goal mutation err=%v", err)
	}
	workspaceMutation := authority
	workspaceMutation.WorkspaceRef = "workspace-ref-mutated"
	if _, err := adapter.ResolveGoalWorkspaceV0(context.Background(), workspaceMutation); !errors.Is(err, errCodexGoalWorkspaceRefMismatchV0) {
		t.Fatalf("workspace mutation err=%v", err)
	}
	manifestMutation := authority
	manifestMutation.IntentManifestRef = "intent-manifest-ref-mutated"
	manifestMutation.IntentManifestSHA256 = strings.Repeat("b", 64)
	if _, err := adapter.ResolveGoalWorkspaceV0(context.Background(), manifestMutation); !errors.Is(err, errCodexGoalWorkspaceAdapterConflictV0) {
		t.Fatalf("manifest mutation err=%v", err)
	}
}

func TestCodexGoalWorkspaceAdapterV0RechazaIdentidadDePacketAntesDeCrearRuntimeV0(t *testing.T) {
	repo := newCodexGoalWorkspaceAdapterGitRepoV0(t)
	for _, test := range []struct {
		name   string
		packet orquestaruntimecodexgoal.CodexGoalStartPacketV0
		want   error
	}{
		{
			name: "workspace no canonico",
			packet: orquestaruntimecodexgoal.CodexGoalStartPacketV0{
				GoalRef: "goal-ref-prepare-identity-workspace", WorkspaceRef: "workspace-ref-manipulated",
			},
			want: errCodexGoalWorkspaceRefMismatchV0,
		},
		{
			name: "provider ajeno",
			packet: orquestaruntimecodexgoal.CodexGoalStartPacketV0{
				GoalRef: "goal-ref-prepare-identity-provider", ProviderRef: "provider-ref-other",
			},
			want: errCodexGoalWorkspaceProviderMismatchV0,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			workspaceRoot := filepath.Join(t.TempDir(), "runtime-must-not-exist")
			adapter := codexGoalWorkspaceAdapterV0{
				SourceWorkDir: repo, WorkspaceRoot: workspaceRoot,
				ProjectRefFallback: "project-ref-fail-before-prepare", WorktreeRefFallback: "worktree-ref-fail-before-prepare",
			}
			if _, err := adapter.PrepareCodexGoalWorkspaceV0(context.Background(), test.packet); !errors.Is(err, test.want) {
				t.Fatalf("err=%v want=%v", err, test.want)
			}
			if _, err := os.Lstat(workspaceRoot); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("runtime side effect exists: %v", err)
			}
			if _, err := os.Lstat(adapter.goalIndexPathV0(test.packet.GoalRef)); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("index side effect exists: %v", err)
			}
		})
	}
}

func TestCodexGoalWorkspaceAdapterV0MaterializesVerifiedIntentManifestV0(t *testing.T) {
	repo := newCodexGoalWorkspaceAdapterGitRepoV0(t)
	request := serverPrepareRunBatchWorkspaceInputV0().AutoprogrammingRequest
	manifest, issues := orquestaautoprogramming.BuildAutoprogrammingIntentManifestV0(request)
	if len(issues) != 0 {
		t.Fatalf("manifest=%+v", issues)
	}
	store := serverAutoprogrammingIntentManifestStoreV0{RootDir: newIntentManifestStoreRootForTestV0(t)}
	if _, err := store.CreateAutoprogrammingIntentManifestIfAbsentV0(context.Background(), manifest); err != nil {
		t.Fatal(err)
	}
	adapter := codexGoalWorkspaceAdapterV0{SourceWorkDir: repo, WorkspaceRoot: filepath.Join(t.TempDir(), "runtime"), ProjectRefFallback: request.ProjectRef, WorktreeRefFallback: request.WorktreeRef, IntentManifestStore: store}
	binding, err := adapter.PrepareCodexGoalWorkspaceV0(context.Background(), orquestaruntimecodexgoal.CodexGoalStartPacketV0{GoalRef: "goal-ref-intent-manifest-workspace-001", RequestRef: request.RequestRef, ProjectRef: request.ProjectRef, IntentManifestRef: manifest.ManifestRef, IntentManifestSHA256: manifest.RequestSHA256})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(binding.ProjectWorkDir, ".orquesta-runtime", "intent-manifests", manifest.ManifestRef+".json")
	raw, err := os.ReadFile(path)
	if err != nil || string(raw) != string(manifest.RequestJSON) {
		t.Fatalf("raw=%q err=%v", raw, err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o400 {
		t.Fatalf("mode=%v err=%v", info.Mode(), err)
	}
	successorGoalRef := "goal-ref-intent-manifest-workspace-001-rework-1"
	if _, err := adapter.PrepareCodexGoalWorkspaceV0(context.Background(), orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef: successorGoalRef, RequestRef: request.RequestRef, ProjectRef: request.ProjectRef,
		IntentManifestRef: manifest.ManifestRef, IntentManifestSHA256: manifest.RequestSHA256,
		ContextRefs:  []orquestagoal.GoalContextRefV0{{Kind: "source_goal", Ref: "goal-ref-intent-manifest-workspace-001", Required: true}},
		WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "base.txt", Purpose: "preservar checkpoint gobernado"}},
		ReworkPolicy: orquestagoal.GoalReworkPolicyV0{PreserveArtifacts: true},
	}); !errors.Is(err, errCodexGoalWorkspaceReworkSourceNotStoppedV0) {
		t.Fatalf("live causal rework must wait for 056 stop receipt: %v", err)
	} else if err.Error() != "requires_056_selective_stop_receipt" {
		t.Fatalf("causal error=%q", err)
	}
	if _, found, err := adapter.loadGoalIndexV0(successorGoalRef); err != nil || found {
		t.Fatalf("blocked rework persisted an index: found=%v err=%v", found, err)
	}
	rework, err := adapter.PrepareCodexGoalWorkspaceV0(context.Background(), orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef: "goal-ref-intent-manifest-workspace-initial-preserve", RequestRef: request.RequestRef, ProjectRef: request.ProjectRef,
		IntentManifestRef: manifest.ManifestRef, IntentManifestSHA256: manifest.RequestSHA256,
		ReworkPolicy: orquestagoal.GoalReworkPolicyV0{PreserveArtifacts: true},
	})
	if err != nil || rework.ProjectWorkDir == binding.ProjectWorkDir {
		t.Fatalf("initial preserve binding=%+v err=%v", rework, err)
	}
	reworkRaw, err := os.ReadFile(filepath.Join(rework.ProjectWorkDir, ".orquesta-runtime", "intent-manifests", manifest.ManifestRef+".json"))
	if err != nil || !bytes.Equal(reworkRaw, manifest.RequestJSON) {
		t.Fatalf("initial preserve did not preserve exact bytes: err=%v raw=%q", err, reworkRaw)
	}
}

func TestCodexGoalWorkspaceAdapterV0DetectsStandardGoalReworkAndFailsClosedUntil056V0(t *testing.T) {
	repo := newCodexGoalWorkspaceAdapterGitRepoV0(t)
	adapter := codexGoalWorkspaceAdapterV0{
		SourceWorkDir: repo, WorkspaceRoot: filepath.Join(t.TempDir(), "runtime"),
		ProjectRefFallback: "project-ref-rework", WorktreeRefFallback: "worktree-ref-rework",
	}
	sourceGoalRef := "goal-ref-standard-causal-source"
	if _, err := adapter.PrepareCodexGoalWorkspaceV0(context.Background(), orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef: sourceGoalRef, RequestRef: "request-ref-standard-causal-source",
	}); err != nil {
		t.Fatal(err)
	}
	packet := orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef:    sourceGoalRef + "-rework-abc",
		RequestRef: "request-ref-standard-causal-source-rework-abc",
		ContextRefs: []orquestagoal.GoalContextRefV0{{
			Kind: "goal", Ref: sourceGoalRef, Required: true,
		}},
		WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "base.txt"}},
		ReworkPolicy: orquestagoal.GoalReworkPolicyV0{PreserveArtifacts: true},
	}
	if _, err := adapter.PrepareCodexGoalWorkspaceV0(context.Background(), packet); !errors.Is(err, errCodexGoalWorkspaceReworkSourceNotStoppedV0) {
		t.Fatalf("standard goal causal rework err=%v", err)
	} else {
		var typed *codexGoalWorkspaceReworkSourceNotStoppedErrorV0
		if !errors.As(err, &typed) {
			t.Fatalf("requires_056 error lost its type: %T %v", err, err)
		}
	}
	if _, found, err := adapter.loadGoalIndexV0(packet.GoalRef); err != nil || found {
		t.Fatalf("blocked standard rework persisted index: found=%v err=%v", found, err)
	}
	nonPreserving := orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef: sourceGoalRef + "-rework-without-artifacts", RequestRef: "request-ref-standard-causal-source-rework-without-artifacts",
		ContextRefs: []orquestagoal.GoalContextRefV0{{Kind: "source_goal", Ref: sourceGoalRef, Required: true}},
	}
	binding, err := adapter.PrepareCodexGoalWorkspaceV0(context.Background(), nonPreserving)
	if err != nil || binding.ProjectWorkDir == "" {
		t.Fatalf("non-preserving causal rework should use a fresh worktree: binding=%+v err=%v", binding, err)
	}
	indexed, found, err := adapter.loadGoalIndexV0(nonPreserving.GoalRef)
	if err != nil || !found || indexed.Identity.SourceGoalRef != sourceGoalRef {
		t.Fatalf("non-preserving causal identity=%+v found=%v err=%v", indexed.Identity, found, err)
	}
	numericParentRef := "goal-ref-standard-numeric-rework-1"
	if _, err := adapter.PrepareCodexGoalWorkspaceV0(context.Background(), orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef: numericParentRef, RequestRef: "request-ref-standard-numeric",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.PrepareCodexGoalWorkspaceV0(context.Background(), orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef: "goal-ref-standard-numeric-rework-2", RequestRef: "request-ref-standard-numeric",
		ContextRefs:  []orquestagoal.GoalContextRefV0{{Kind: "goal", Ref: numericParentRef, Required: true}},
		WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "base.txt"}},
		ReworkPolicy: orquestagoal.GoalReworkPolicyV0{PreserveArtifacts: true},
	}); !errors.Is(err, errCodexGoalWorkspaceReworkSourceNotStoppedV0) {
		t.Fatalf("numeric parent rework-1 -> rework-2 was not recognized: %v", err)
	}
}

func TestCodexGoalWorkspaceAdapterV0ReworkLineageLegacyAndManifestImmutabilityV0(t *testing.T) {
	legacy := codexGoalWorkspaceRequestIdentityV0{
		RunRef: "request-ref-legacy", GoalRef: "goal-ref-legacy", ProjectRef: "project-ref", WorktreeRef: "worktree-ref",
	}
	legacySuccessor := legacy
	legacySuccessor.RunRef = legacy.RunRef + "-rework-abc"
	legacySuccessor.GoalRef = legacy.GoalRef + "-rework-abc"
	legacySuccessor.SourceGoalRef = legacy.GoalRef
	if !codexGoalWorkspaceValidReworkLineageV0(legacy, legacySuccessor) {
		t.Fatal("resident legacy derived request_ref was rejected")
	}
	manifestSource := legacy
	manifestSource.IntentManifestRef = "intent-manifest-ref-immutable"
	manifestSource.IntentManifestSHA256 = strings.Repeat("a", 64)
	manifestSuccessor := manifestSource
	manifestSuccessor.GoalRef = manifestSource.GoalRef + "-rework-abc"
	manifestSuccessor.SourceGoalRef = manifestSource.GoalRef
	if !codexGoalWorkspaceValidReworkLineageV0(manifestSource, manifestSuccessor) {
		t.Fatal("immutable manifest successor was rejected")
	}
	manifestSuccessor.RunRef += "-rework-abc"
	if codexGoalWorkspaceValidReworkLineageV0(manifestSource, manifestSuccessor) {
		t.Fatal("manifest-backed rework substituted request_ref")
	}
	manifestSuccessor = manifestSource
	manifestSuccessor.GoalRef = manifestSource.GoalRef + "-rework-abc"
	manifestSuccessor.SourceGoalRef = manifestSource.GoalRef
	manifestSuccessor.IntentManifestSHA256 = strings.Repeat("b", 64)
	if codexGoalWorkspaceValidReworkLineageV0(manifestSource, manifestSuccessor) {
		t.Fatal("manifest-backed rework substituted immutable digest")
	}
}

func TestCodexGoalWorkspaceAdapterV0SeleccionaPredecesorInmediatoEnHistoriaResidentV0(t *testing.T) {
	first := "goal-ref-resident-chain"
	second := first + "-rework-a1"
	third := second + "-rework-b2"
	packet := orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef: third,
		ContextRefs: []orquestagoal.GoalContextRefV0{
			{Kind: "source_goal", Ref: first, Required: true},
			{Kind: "source_goal", Ref: second, Required: true},
		},
		ReworkPolicy: orquestagoal.GoalReworkPolicyV0{PreserveArtifacts: true},
	}
	got, err := codexGoalWorkspaceReworkSourceV0(packet)
	if err != nil || got != second {
		t.Fatalf("source=%q want=%q err=%v", got, second, err)
	}
	packet.ContextRefs[1].Required = false
	if _, err := codexGoalWorkspaceReworkSourceV0(packet); !errors.Is(err, errCodexGoalWorkspaceAdapterConflictV0) {
		t.Fatalf("optional causal source accepted: %v", err)
	}
	if codexGoalWorkspaceCausalSuccessorRefV0(first, first+"-rework-2") {
		t.Fatal("numeric rework-2 accepted base as immediate predecessor")
	}
	if !codexGoalWorkspaceCausalSuccessorRefV0(first+"-rework-1", first+"-rework-2") {
		t.Fatal("numeric rework-1 was not accepted as immediate predecessor")
	}
}

func TestCodexGoalWorkspaceAdapterV0ConservaIndexV0Pre057SinAutoridadCompletaV0(t *testing.T) {
	repo := newCodexGoalWorkspaceAdapterGitRepoV0(t)
	adapter := codexGoalWorkspaceAdapterV0{
		SourceWorkDir: repo, WorkspaceRoot: filepath.Join(t.TempDir(), "runtime"),
		ProjectRefFallback: "project-ref-pre057", WorktreeRefFallback: "worktree-ref-pre057",
	}
	packet := orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef: "goal-ref-pre057-replay", RequestRef: "request-ref-pre057-replay",
		WriteSet: []orquestagoal.GoalWriteScopeV0{{Path: "base.txt", Purpose: "legacy durable spec"}},
	}
	first, err := adapter.PrepareCodexGoalWorkspaceV0(context.Background(), packet)
	if err != nil {
		t.Fatal(err)
	}
	current, found, err := adapter.loadGoalIndexV0(packet.GoalRef)
	if err != nil || !found {
		t.Fatalf("current=%+v found=%v err=%v", current, found, err)
	}
	legacy := current
	legacy.SchemaVersion = codexGoalWorkspaceAdapterIndexSchemaV0
	legacy.Identity.IntentManifestRef = ""
	legacy.Identity.IntentManifestSHA256 = ""
	legacy.Identity.SourceGoalRef = ""
	legacy.Identity.WriteSetSHA256 = ""
	legacy.Identity.ReworkPolicySHA256 = ""
	legacy.Identity.WorkspaceRef = ""
	legacy.Identity.ProviderRef = ""
	legacy.Identity.RuntimeGenerationRef = ""
	legacy.Identity.ExternalGoalRef = ""
	raw, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	path := adapter.goalIndexPathV0(packet.GoalRef)
	dir, err := openCodexGoalWorkspaceIndexDirV1(filepath.Dir(path), false)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeCodexGoalWorkspaceIndexV1(dir, filepath.Base(path), append(raw, '\n')); err != nil {
		_ = dir.Close()
		t.Fatal(err)
	}
	_ = dir.Close()

	restarted := adapter
	replayed, err := restarted.PrepareCodexGoalWorkspaceV0(context.Background(), packet)
	if err != nil || replayed.ProjectWorkDir != first.ProjectWorkDir {
		t.Fatalf("replayed=%+v first=%+v err=%v", replayed, first, err)
	}
	resolved, err := restarted.ResolveCodexGoalWorkspaceV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{GoalRef: packet.GoalRef})
	if err != nil || resolved.ProjectWorkDir != first.ProjectWorkDir {
		t.Fatalf("resolved=%+v first=%+v err=%v", resolved, first, err)
	}
	preserved, found, err := adapter.loadGoalIndexV0(packet.GoalRef)
	if err != nil || !found || preserved.SchemaVersion != codexGoalWorkspaceAdapterIndexSchemaV0 ||
		preserved.Identity.WorkspaceRef != "" || preserved.Identity.ProviderRef != "" || preserved.Identity.RuntimeGenerationRef != "" {
		t.Fatalf("preserved=%+v found=%v err=%v", preserved, found, err)
	}
	if _, err := restarted.ResolveGoalWorkspaceV0(context.Background(), orquestagoal.GoalObservationRequestV0{
		GoalRef: packet.GoalRef, WorkspaceAuthoritySchemaVersion: orquestagoal.GoalWorkspaceAuthoritySchemaV0,
		WorkspaceRef: orquestagoal.GoalWorkspaceRefForGoalV0(packet.GoalRef), ProviderRef: orquestaruntimecodexgoal.CodexGoalProviderRefV0,
		RuntimeGenerationRef: "generation-ref-v0-cannot-be-verified", ExternalGoalRef: "thread-ref-v0-cannot-be-verified",
	}); !errors.Is(err, errCodexGoalWorkspaceAdapterConflictV0) {
		t.Fatalf("v0 manufactured authority from request: %v", err)
	}
}

func TestCodexGoalWorkspaceAdapterV0ResolveV1SoloMigraConAutoridadCompletaV0(t *testing.T) {
	repo := newCodexGoalWorkspaceAdapterGitRepoV0(t)
	adapter := codexGoalWorkspaceAdapterV0{
		SourceWorkDir: repo, WorkspaceRoot: filepath.Join(t.TempDir(), "runtime"),
		ProjectRefFallback: "project-ref-pre057-v1", WorktreeRefFallback: "worktree-ref-pre057-v1",
	}
	packet := orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef: "goal-ref-pre057-v1-replay", RequestRef: "request-ref-pre057-v1-replay",
		WriteSet: []orquestagoal.GoalWriteScopeV0{{Path: "base.txt"}},
	}
	prepared, err := adapter.PrepareCodexGoalWorkspaceV0(context.Background(), packet)
	if err != nil {
		t.Fatal(err)
	}
	legacy, found, err := adapter.loadGoalIndexV0(packet.GoalRef)
	if err != nil || !found {
		t.Fatalf("legacy source=%+v found=%v err=%v", legacy, found, err)
	}
	legacy.SchemaVersion = codexGoalWorkspaceAdapterIndexSchemaV1
	legacy.Identity.WorkspaceRef = ""
	legacy.Identity.ProviderRef = ""
	legacy.Identity.RuntimeGenerationRef = ""
	legacy.Identity.ExternalGoalRef = ""
	overwriteCodexGoalWorkspaceIndexForTestV0(t, adapter, legacy)

	resolved, err := adapter.ResolveGoalWorkspaceV0(context.Background(), orquestagoal.GoalObservationRequestV0{GoalRef: packet.GoalRef})
	if err != nil || resolved.ProjectWorkDir != prepared.ProjectWorkDir || resolved.WorkspaceRef != "" || resolved.ProviderRef != "" || resolved.RuntimeGenerationRef != "" {
		t.Fatalf("legacy projection=%+v err=%v", resolved, err)
	}
	preserved, _, err := adapter.loadGoalIndexV0(packet.GoalRef)
	if err != nil || preserved.SchemaVersion != codexGoalWorkspaceAdapterIndexSchemaV1 {
		t.Fatalf("v1 migrated without authority: %+v err=%v", preserved, err)
	}
	authority := orquestagoal.GoalObservationRequestV0{
		GoalRef: packet.GoalRef, ExternalGoalRef: "thread-ref-v1-verified", WorkspaceAuthoritySchemaVersion: orquestagoal.GoalWorkspaceAuthoritySchemaV0,
		WorkspaceRef: orquestagoal.GoalWorkspaceRefForGoalV0(packet.GoalRef), ProviderRef: orquestaruntimecodexgoal.CodexGoalProviderRefV0,
		RuntimeGenerationRef: "generation-ref-v1-verified",
	}
	bound, err := adapter.ResolveGoalWorkspaceV0(context.Background(), authority)
	if err != nil || bound.WorkspaceRef != authority.WorkspaceRef || bound.ProviderRef != authority.ProviderRef || bound.RuntimeGenerationRef != authority.RuntimeGenerationRef {
		t.Fatalf("bound=%+v err=%v", bound, err)
	}
	migrated, _, err := adapter.loadGoalIndexV0(packet.GoalRef)
	if err != nil || migrated.SchemaVersion != codexGoalWorkspaceAdapterIndexSchemaV2 ||
		migrated.Identity.WorkspaceRef != authority.WorkspaceRef || migrated.Identity.ProviderRef != authority.ProviderRef || migrated.Identity.RuntimeGenerationRef != authority.RuntimeGenerationRef || migrated.Identity.ExternalGoalRef != authority.ExternalGoalRef {
		t.Fatalf("migrated=%+v err=%v", migrated, err)
	}
}

func overwriteCodexGoalWorkspaceIndexForTestV0(t *testing.T, adapter codexGoalWorkspaceAdapterV0, index codexGoalWorkspaceAdapterIndexV0) {
	t.Helper()
	raw, err := json.Marshal(index)
	if err != nil {
		t.Fatal(err)
	}
	path := adapter.goalIndexPathV0(index.GoalRef)
	dir, err := openCodexGoalWorkspaceIndexDirV1(filepath.Dir(path), false)
	if err != nil {
		t.Fatal(err)
	}
	defer dir.Close()
	if err := writeCodexGoalWorkspaceIndexV1(dir, filepath.Base(path), append(raw, '\n')); err != nil {
		t.Fatal(err)
	}
}

func TestCodexGoalWorkspaceAdapterV0LockRespetaCancelacionV0(t *testing.T) {
	adapter := codexGoalWorkspaceAdapterV0{WorkspaceRoot: filepath.Join(t.TempDir(), "runtime")}
	goalRef := "goal-ref-lock-cancel"
	lockPath := adapter.goalIndexLockPathV0(goalRef)
	dir, err := openCodexGoalWorkspaceIndexDirV1(filepath.Dir(lockPath), true)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := openCodexGoalWorkspaceIndexLockV1(dir, filepath.Base(lockPath))
	if err != nil {
		_ = dir.Close()
		t.Fatal(err)
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
		_ = lock.Close()
		_ = dir.Close()
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, err = adapter.withGoalIndexLockV0(ctx, goalRef, func() (orquestaruntimecodexappserver.GoalWorkspaceBindingV0, error) {
		t.Fatal("callback executed while lock held")
		return orquestaruntimecodexappserver.GoalWorkspaceBindingV0{}, nil
	})
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(started) > time.Second {
		t.Fatalf("lock cancellation err=%v elapsed=%s", err, time.Since(started))
	}
}

func TestCodexGoalWorkspaceAdapterV0IndexRechazaJSONAmbiguoSymlinkYHardlinkV0(t *testing.T) {
	repo := newCodexGoalWorkspaceAdapterGitRepoV0(t)
	adapter := codexGoalWorkspaceAdapterV0{
		SourceWorkDir: repo, WorkspaceRoot: filepath.Join(t.TempDir(), "runtime"),
		ProjectRefFallback: "project-ref-index-strict", WorktreeRefFallback: "worktree-ref-index-strict",
	}
	packet := orquestaruntimecodexgoal.CodexGoalStartPacketV0{GoalRef: "goal-ref-index-strict", RequestRef: "request-ref-index-strict"}
	if _, err := adapter.PrepareCodexGoalWorkspaceV0(context.Background(), packet); err != nil {
		t.Fatal(err)
	}
	index, found, err := adapter.loadGoalIndexV0(packet.GoalRef)
	if err != nil || !found {
		t.Fatalf("index=%+v found=%v err=%v", index, found, err)
	}
	raw, err := json.Marshal(index)
	if err != nil {
		t.Fatal(err)
	}
	duplicate := bytes.Replace(raw, []byte(`"schema_version":`), []byte(`"schema_version":"`+codexGoalWorkspaceAdapterIndexSchemaV1+`","schema_version":`), 1)
	unknown := append(append([]byte(nil), raw[:len(raw)-1]...), []byte(`,"unknown":true}`)...)
	for name, candidate := range map[string][]byte{
		"duplicate": duplicate,
		"unknown":   unknown,
		"trailing":  append(append([]byte(nil), raw...), []byte("\n{}\n")...),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeCodexGoalWorkspaceIndexV1(candidate); err == nil {
				t.Fatal("ambiguous index JSON accepted")
			}
		})
	}

	path := adapter.goalIndexPathV0(packet.GoalRef)
	hardlink := path + ".hard"
	if err := os.Link(path, hardlink); err != nil {
		t.Fatal(err)
	}
	if _, _, err := adapter.loadGoalIndexV0(packet.GoalRef); err == nil {
		t.Fatal("hardlinked index accepted")
	}
	if err := os.Remove(hardlink); err != nil {
		t.Fatal(err)
	}
	if _, found, err := adapter.loadGoalIndexV0(packet.GoalRef); err != nil || !found {
		t.Fatalf("index did not recover after removing hardlink: found=%v err=%v", found, err)
	}
	target := filepath.Join(t.TempDir(), "index-target.json")
	if err := os.WriteFile(target, append(raw, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	if _, _, err := adapter.loadGoalIndexV0(packet.GoalRef); err == nil {
		t.Fatal("symlink index accepted")
	}
}

func TestCodexGoalWorkspaceAdapterV0IndexFixesWriteSetAndReworkPolicyV0(t *testing.T) {
	repo := newCodexGoalWorkspaceAdapterGitRepoV0(t)
	adapter := codexGoalWorkspaceAdapterV0{
		SourceWorkDir: repo, WorkspaceRoot: filepath.Join(t.TempDir(), "runtime"),
		ProjectRefFallback: "project-ref-identity", WorktreeRefFallback: "worktree-ref-identity",
	}
	packet := orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef: "goal-ref-fixed-workspace-identity", RequestRef: "request-ref-fixed-workspace-identity",
		WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "base.txt", Purpose: "first"}},
		ReworkPolicy: orquestagoal.GoalReworkPolicyV0{PreserveArtifacts: true},
	}
	if _, err := adapter.PrepareCodexGoalWorkspaceV0(context.Background(), packet); err != nil {
		t.Fatal(err)
	}
	changed := packet
	changed.WriteSet = []orquestagoal.GoalWriteScopeV0{{Path: "base.txt", Purpose: "substituted"}}
	if _, err := adapter.PrepareCodexGoalWorkspaceV0(context.Background(), changed); !errors.Is(err, errCodexGoalWorkspaceAdapterConflictV0) {
		t.Fatalf("write_set substitution err=%v", err)
	}
	changed = packet
	changed.ReworkPolicy.PreferNewGoal = true
	if _, err := adapter.PrepareCodexGoalWorkspaceV0(context.Background(), changed); !errors.Is(err, errCodexGoalWorkspaceAdapterConflictV0) {
		t.Fatalf("rework policy substitution err=%v", err)
	}
}

func TestCodexGoalWorkspaceAdapterV0RejectsNilContextV0(t *testing.T) {
	adapter := codexGoalWorkspaceAdapterV0{SourceWorkDir: t.TempDir(), WorkspaceRoot: t.TempDir()}
	if _, err := adapter.PrepareCodexGoalWorkspaceV0(nil, orquestaruntimecodexgoal.CodexGoalStartPacketV0{GoalRef: "goal-ref-nil-context"}); !errors.Is(err, errCodexGoalWorkspaceAdapterUnavailableV0) {
		t.Fatalf("prepare nil context err=%v", err)
	}
	if _, err := adapter.ResolveCodexGoalWorkspaceV0(nil, orquestaruntimecodexgoal.CodexGoalObservationRequestV0{GoalRef: "goal-ref-nil-context"}); !errors.Is(err, errCodexGoalWorkspaceAdapterUnavailableV0) {
		t.Fatalf("resolve nil context err=%v", err)
	}
	if _, err := adapter.HasCodexGoalWorkspaceBindingV0(nil, "goal-ref-nil-context"); !errors.Is(err, errCodexGoalWorkspaceAdapterUnavailableV0) {
		t.Fatalf("has nil context err=%v", err)
	}
}

func TestCodexGoalWorkspaceAdapterV0NeutralPortPreservesManifestAndResolveV0(t *testing.T) {
	repo := newCodexGoalWorkspaceAdapterGitRepoV0(t)
	request := serverPrepareRunBatchWorkspaceInputV0().AutoprogrammingRequest
	request.RequestRef = "request-ref-neutral-workspace-manifest"
	manifest, issues := orquestaautoprogramming.BuildAutoprogrammingIntentManifestV0(request)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	store := serverAutoprogrammingIntentManifestStoreV0{RootDir: newIntentManifestStoreRootForTestV0(t)}
	if _, err := store.CreateAutoprogrammingIntentManifestIfAbsentV0(context.Background(), manifest); err != nil {
		t.Fatal(err)
	}
	adapter := codexGoalWorkspaceAdapterV0{
		SourceWorkDir: repo, WorkspaceRoot: filepath.Join(t.TempDir(), "runtime"),
		ProjectRefFallback: request.ProjectRef, WorktreeRefFallback: request.WorktreeRef,
		IntentManifestStore: store,
	}
	spec := orquestagoal.GoalWorkSpecV0{
		SchemaVersion:        orquestagoal.GoalWorkSpecSchemaV0,
		GoalRef:              "goal-ref-neutral-workspace-manifest",
		RequestRef:           request.RequestRef,
		RunRef:               "run-ref-neutral-workspace-manifest",
		ProjectRef:           request.ProjectRef,
		Objective:            "Verificar el puerto neutral de workspace.",
		DirectorKind:         orquestagoal.GoalDirectorKindRuntimeGoalV0,
		IntentManifestRef:    manifest.ManifestRef,
		IntentManifestSHA256: manifest.RequestSHA256,
		ContextRefs:          []orquestagoal.GoalContextRefV0{{Kind: "worktree", Ref: request.WorktreeRef, Required: true}},
		WriteSet:             []orquestagoal.GoalWriteScopeV0{{Path: "docs", Purpose: "prueba neutral"}},
		RequiredTests:        []orquestagoal.GoalRequiredTestV0{{TestRef: "required-test-ref-neutral-workspace", Command: "test -d ."}},
	}
	binding, err := adapter.PrepareGoalWorkspaceV0(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(binding.ProjectWorkDir, ".orquesta-runtime", "intent-manifests", manifest.ManifestRef+".json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil || !bytes.Equal(raw, manifest.RequestJSON) {
		t.Fatalf("neutral manifest err=%v raw=%q", err, raw)
	}
	info, err := os.Stat(manifestPath)
	if err != nil {
		t.Fatalf("neutral manifest stat: %v", err)
	}
	if info.Mode().Perm() != 0o400 {
		t.Fatalf("neutral manifest mode=%v", info.Mode())
	}
	indexed, found, err := adapter.loadGoalIndexV0(spec.GoalRef)
	if err != nil || !found || indexed.Identity.RunRef != spec.RequestRef || indexed.Identity.ProjectRef != spec.ProjectRef || indexed.Identity.WorktreeRef != request.WorktreeRef {
		t.Fatalf("neutral identity=%+v found=%v err=%v", indexed.Identity, found, err)
	}
	restarted := codexGoalWorkspaceAdapterV0{
		SourceWorkDir: adapter.SourceWorkDir, WorkspaceRoot: adapter.WorkspaceRoot,
		ProjectRefFallback: adapter.ProjectRefFallback, WorktreeRefFallback: adapter.WorktreeRefFallback,
		IntentManifestStore: store,
	}
	authority := orquestagoal.GoalExecutionAuthorityForProviderV0(spec, orquestaruntimecodexgoal.CodexGoalProviderRefV0)
	resolved, err := restarted.ResolveGoalWorkspaceV0(
		context.Background(),
		orquestagoal.GoalObservationRequestForAuthorityV0(authority, "thread-ref-neutral-workspace-manifest"),
	)
	if err != nil || resolved.ProjectWorkDir != binding.ProjectWorkDir {
		t.Fatalf("resolved=%+v binding=%+v err=%v", resolved, binding, err)
	}
	if err := os.Chmod(manifestPath, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := restarted.ResolveGoalWorkspaceV0(context.Background(), orquestagoal.GoalObservationRequestV0{GoalRef: spec.GoalRef}); err == nil {
		t.Fatal("tampered manifest accepted during observation")
	}
}

func TestCodexGoalWorkspaceAdapterV0IntentManifestFailsClosedOnFilesystemObstaclesV0(t *testing.T) {
	request := serverPrepareRunBatchWorkspaceInputV0().AutoprogrammingRequest
	request.RequestRef = "request-ref-workspace-obstacles-001"
	manifest, issues := orquestaautoprogramming.BuildAutoprogrammingIntentManifestV0(request)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	store := serverAutoprogrammingIntentManifestStoreV0{RootDir: newIntentManifestStoreRootForTestV0(t)}
	if _, err := store.CreateAutoprogrammingIntentManifestIfAbsentV0(context.Background(), manifest); err != nil {
		t.Fatal(err)
	}
	adapter := codexGoalWorkspaceAdapterV0{IntentManifestStore: store}
	packet := orquestaruntimecodexgoal.CodexGoalStartPacketV0{GoalRef: "goal-ref-workspace-obstacles-001", RequestRef: request.RequestRef, IntentManifestRef: manifest.ManifestRef, IntentManifestSHA256: manifest.RequestSHA256}

	t.Run("concurrent identical", func(t *testing.T) {
		workspace := t.TempDir()
		start := make(chan struct{})
		errs := make(chan error, 8)
		var wg sync.WaitGroup
		for range 8 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				errs <- adapter.materializeIntentManifestV0(context.Background(), packet, workspace)
			}()
		}
		close(start)
		wg.Wait()
		close(errs)
		for err := range errs {
			if err != nil {
				t.Fatal(err)
			}
		}
		path := filepath.Join(workspace, ".orquesta-runtime", "intent-manifests", manifest.ManifestRef+".json")
		raw, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(raw, manifest.RequestJSON) {
			t.Fatalf("partial materialization: err=%v raw=%q", err, raw)
		}
		entries, _ := os.ReadDir(filepath.Dir(path))
		if len(entries) != 1 {
			t.Fatalf("temporary files leaked: %v", entries)
		}
	})

	t.Run("runtime symlink", func(t *testing.T) {
		workspace, outside := t.TempDir(), t.TempDir()
		if err := os.Symlink(outside, filepath.Join(workspace, ".orquesta-runtime")); err != nil {
			t.Fatal(err)
		}
		if err := adapter.materializeIntentManifestV0(context.Background(), packet, workspace); !errors.Is(err, errCodexGoalWorkspaceAdapterUnavailableV0) {
			t.Fatalf("runtime symlink err=%v", err)
		}
		if entries, err := os.ReadDir(outside); err != nil || len(entries) != 0 {
			t.Fatalf("wrote outside workspace: entries=%v err=%v", entries, err)
		}
	})

	for _, tc := range []struct {
		name    string
		prepare func(t *testing.T, path string)
		want    string
	}{
		{name: "expanded permissions", prepare: func(t *testing.T, path string) {
			if err := os.WriteFile(path, manifest.RequestJSON, 0o600); err != nil {
				t.Fatal(err)
			}
		}, want: string(manifest.RequestJSON)},
		{name: "partial final", prepare: func(t *testing.T, path string) {
			if err := os.WriteFile(path, []byte("{"), 0o400); err != nil {
				t.Fatal(err)
			}
		}, want: "{"},
		{name: "final symlink", prepare: func(t *testing.T, path string) {
			target := filepath.Join(t.TempDir(), "target")
			if err := os.WriteFile(target, []byte("sentinel"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, path); err != nil {
				t.Fatal(err)
			}
		}, want: "sentinel"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			workspace := t.TempDir()
			root := filepath.Join(workspace, ".orquesta-runtime", "intent-manifests")
			if err := os.MkdirAll(root, 0o700); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, manifest.ManifestRef+".json")
			tc.prepare(t, path)
			if err := adapter.materializeIntentManifestV0(context.Background(), packet, workspace); !errors.Is(err, errCodexGoalWorkspaceAdapterConflictV0) {
				t.Fatalf("obstacle accepted: %v", err)
			}
			readPath := path
			if tc.name == "final symlink" {
				target, err := os.Readlink(path)
				if err != nil {
					t.Fatal(err)
				}
				readPath = target
			}
			raw, _ := os.ReadFile(readPath)
			if string(raw) != tc.want {
				t.Fatalf("obstacle mutated: %q", raw)
			}
		})
	}

	t.Run("final directory", func(t *testing.T) {
		workspace := t.TempDir()
		root := filepath.Join(workspace, ".orquesta-runtime", "intent-manifests")
		if err := os.MkdirAll(root, 0o700); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(root, manifest.ManifestRef+".json")
		if err := os.Mkdir(path, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := adapter.materializeIntentManifestV0(context.Background(), packet, workspace); !errors.Is(err, errCodexGoalWorkspaceAdapterConflictV0) {
			t.Fatalf("directory obstacle accepted: %v", err)
		}
		if info, err := os.Stat(path); err != nil || !info.IsDir() {
			t.Fatalf("directory obstacle mutated: info=%v err=%v", info, err)
		}
	})
}

func TestCodexGoalWorkspaceAdapterV0RejectsConflictingRequestIdentity(t *testing.T) {
	repo := newCodexGoalWorkspaceAdapterGitRepoV0(t)
	adapter := codexGoalWorkspaceAdapterV0{
		SourceWorkDir: repo, WorkspaceRoot: filepath.Join(t.TempDir(), "runtime"),
		ProjectRefFallback: "project-ref-fallback", WorktreeRefFallback: "worktree-ref-fallback",
	}
	packet := orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef: "goal-ref-workspace-adapter-conflict", RequestRef: "run-ref-workspace-adapter-conflict",
	}
	if _, err := adapter.PrepareCodexGoalWorkspaceV0(context.Background(), packet); err != nil {
		t.Fatalf("prepare: %v", err)
	}
	packet.ProjectRef = "project-ref-conflict"
	if _, err := adapter.PrepareCodexGoalWorkspaceV0(context.Background(), packet); !errors.Is(err, errCodexGoalWorkspaceAdapterConflictV0) {
		t.Fatalf("conflicting identity err=%v", err)
	}
}

func TestCodexGoalWorkspaceAdapterV0UsaWorktreeTipadoDelGoalV0(t *testing.T) {
	adapter := codexGoalWorkspaceAdapterV0{
		SourceWorkDir:       t.TempDir(),
		WorkspaceRoot:       t.TempDir(),
		ProjectRefFallback:  "project-ref-fallback",
		WorktreeRefFallback: "worktree-ref-fallback",
	}
	request := adapter.requestForStartV0(orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef:    "goal-ref-workspace-typed",
		RequestRef: "run-ref-workspace-typed",
		ProjectRef: "project-ref-workspace-typed",
		ContextRefs: []orquestagoal.GoalContextRefV0{{
			Kind: "worktree",
			Ref:  "worktree-ref-workspace-typed",
		}},
	})
	if request.WorktreeRef != "worktree-ref-workspace-typed" {
		t.Fatalf("worktree_ref=%q", request.WorktreeRef)
	}
}

func TestCodexGoalWorkspaceRootForSourceV0IsStableAndOutsideRepo(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	root := codexGoalWorkspaceRootForSourceV0(repo)
	if root == "" || filepath.Dir(root) != filepath.Dir(repo) || root == repo {
		t.Fatalf("root=%q repo=%q", root, repo)
	}
	if replay := codexGoalWorkspaceRootForSourceV0(repo); replay != root {
		t.Fatalf("root not stable: %q / %q", root, replay)
	}
}

func newCodexGoalWorkspaceAdapterGitRepoV0(t *testing.T) string {
	t.Helper()
	repo := filepath.Join(t.TempDir(), "source")
	if err := os.MkdirAll(repo, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "test@example.invalid"},
		{"config", "user.name", "Orquesta Test"},
	} {
		codexGoalWorkspaceAdapterRunGitV0(t, repo, args...)
	}
	if err := os.WriteFile(filepath.Join(repo, "base.txt"), []byte("base\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	codexGoalWorkspaceAdapterRunGitV0(t, repo, "add", "base.txt")
	codexGoalWorkspaceAdapterRunGitV0(t, repo, "commit", "-q", "-m", "base")
	return repo
}

func codexGoalWorkspaceAdapterRunGitV0(t *testing.T, repo string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", repo}, args...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
}
