package orquestaappcodexstack

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	orquestaagentprocessregistrymemory "orquesta/modulos/orquesta-agent-process-registry-memory"
	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestastatefile "orquesta/modulos/orquesta-state-file"
)

func TestCodexStackObserveAppDirectorGoalExecutorV0TimeoutSnapshotIncluyeProcessRefsV0(t *testing.T) {
	ctx := context.Background()
	runRef := "run-ref-stack-observe-goal-snapshot-001"
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	state, err := orquestagoal.NewGoalWorkStateFromLaunchV0(orquestagoal.GoalWorkStateFromLaunchRequestV0{
		RunRef: runRef,
		Spec: orquestagoal.GoalWorkSpecV0{
			SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
			GoalRef:       "goal-ref-stack-observe-snapshot-001",
			RunRef:        runRef,
			Objective:     "Publicar snapshot parcial con proceso vivo.",
			DirectorKind:  orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet: []orquestagoal.GoalWriteScopeV0{{
				Path: "docs/goal_snapshot.md",
			}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			Status:          orquestagoal.GoalStatusRunningV0,
			GoalRef:         "goal-ref-stack-observe-snapshot-001",
			ExternalGoalRef: "thread-ref-stack-observe-snapshot-001",
		},
		EvidenceRefs: []string{"evidence-ref-stack-observe-snapshot-state-001"},
	})
	if err != nil {
		t.Fatalf("NewGoalWorkStateFromLaunchV0: %v", err)
	}
	state.StoreVersion = 1
	if err := goalStates.SaveGoalWorkStateV0(ctx, state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	processRegistry := orquestaagentprocessregistrymemory.NewInMemoryAgentProcessRegistryV0()
	if err := processRegistry.RecordAgentProcessV0(ctx, orquestaagentprocessregistrymemory.AgentProcessRecordV0{
		RunID:          runRef,
		AgentRequestID: "agent-ref-stack-observe-snapshot-001",
		ProcessRef:     "process-ref-stack-observe-snapshot-001",
		SessionRef:     "session-ref-stack-observe-snapshot-001",
		LaunchRef:      "launch-ref-stack-observe-snapshot-001",
		ReadinessRef:   "readiness-ref-stack-observe-snapshot-001",
		EvidenceRefs:   []string{"evidence-ref-stack-observe-snapshot-process-001"},
	}); err != nil {
		t.Fatalf("RecordAgentProcessV0: %v", err)
	}
	stack := &StackV0{
		Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			GoalStateStore: goalStates,
		},
		Stores: StoresV0{
			ProcessRegistry: processRegistry,
		},
	}

	result, err := NewCodexStackObserveAppDirectorGoalExecutorV0(stack).ObserveAppDirectorGoalTimeoutSnapshotV0(
		ctx,
		orquestamcp.MCPObserveAppDirectorGoalToolInputV0{RunRef: runRef},
	)

	if err != nil {
		t.Fatalf("ObserveAppDirectorGoalTimeoutSnapshotV0: %v", err)
	}
	if !result.Partial ||
		result.GoalStatus == orquestagoal.GoalStatusRunningV0 ||
		result.RecommendedAction != "observe_estado_vivo" ||
		!result.ClosureNeedsRework ||
		!codexStackStringInSetForTestV0(result.ProcessRefs, "process-ref-stack-observe-snapshot-001") ||
		!codexStackStringInSetForTestV0(result.EvidenceRefs, "evidence-ref-stack-observe-snapshot-process-001") {
		t.Fatalf("result=%+v", result)
	}
}

func TestCodexStackObserveAppDirectorGoalExecutorV0TimeoutSnapshotRespetaRunControlTerminalV0(t *testing.T) {
	ctx := context.Background()
	runRef := "run-ref-stack-observe-run-control-terminal-001"
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	state, err := orquestagoal.NewGoalWorkStateFromLaunchV0(orquestagoal.GoalWorkStateFromLaunchRequestV0{
		RunRef: runRef,
		Spec: orquestagoal.GoalWorkSpecV0{
			SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
			GoalRef:       "goal-ref-stack-observe-run-control-terminal-001",
			RunRef:        runRef,
			Objective:     "Snapshot no debe publicar running tras RunControl terminal.",
			DirectorKind:  orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet: []orquestagoal.GoalWriteScopeV0{{
				Path: "docs/goal_snapshot_terminal.md",
			}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			Status:          orquestagoal.GoalStatusRunningV0,
			GoalRef:         "goal-ref-stack-observe-run-control-terminal-001",
			ExternalGoalRef: "thread-ref-stack-observe-run-control-terminal-001",
		},
		EvidenceRefs: []string{"evidence-ref-stack-observe-run-control-terminal-state"},
	})
	if err != nil {
		t.Fatalf("NewGoalWorkStateFromLaunchV0: %v", err)
	}
	if err := goalStates.SaveGoalWorkStateV0(ctx, state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	stack := &StackV0{
		Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			GoalStateStore: goalStates,
			RunControl: codexStackRunControlReaderForObserveTestV0{state: orquestaruncontrol.RunControlStateV0{
				RunRef:       runRef,
				Status:       orquestaruncontrol.RunControlStatusStoppedV0,
				Forced:       true,
				EvidenceRefs: []string{"evidence-ref-stack-run-control-terminal-stopped"},
			}},
		},
	}

	result, err := NewCodexStackObserveAppDirectorGoalExecutorV0(stack).ObserveAppDirectorGoalTimeoutSnapshotV0(
		ctx,
		orquestamcp.MCPObserveAppDirectorGoalToolInputV0{RunRef: runRef},
	)

	if err != nil {
		t.Fatalf("ObserveAppDirectorGoalTimeoutSnapshotV0: %v", err)
	}
	if result.GoalStatus != orquestagoal.GoalStatusBlockedV0 ||
		result.RecommendedAction != "replan" ||
		result.ClosureStatus != orquestagoal.GoalStatusBlockedV0 ||
		!result.ClosureNeedsRework ||
		!codexStackStringInSetForTestV0(result.EvidenceRefs, "evidence-ref-stack-run-control-terminal-stopped") ||
		!codexStackStringInSetForTestV0(result.EvidenceRefs, "evidence-ref-observe-goal-run-control-terminal") {
		t.Fatalf("result=%+v", result)
	}
}

func TestCodexStackObserveAppDirectorGoalExecutorV0TimeoutSnapshotReparaReceiptMaterializadoV0(t *testing.T) {
	ctx := context.Background()
	runRef := "run-ref-stack-observe-goal-repair-receipt-001"
	projectDir := t.TempDir()
	topicDir := filepath.Join(projectDir, "temas", "tema_031")
	validationDir := filepath.Join(topicDir, "09_validacion")
	if err := os.MkdirAll(validationDir, 0o700); err != nil {
		t.Fatalf("mkdir validation: %v", err)
	}
	if err := os.WriteFile(filepath.Join(topicDir, "tema_ampliado.md"), []byte("# Tema\n"), 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(validationDir, "informe_qa.json"),
		[]byte(`{"qa_passes":{"extension_pass":true,"official_text_qa_pass":true,"strict_editorial_qa_pass":true,
				"question_bank_publicable":true,
				"tutor_assets_publicable":true}}`),
		0o600,
	); err != nil {
		t.Fatalf("write qa: %v", err)
	}
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	state, err := orquestagoal.NewGoalWorkStateFromLaunchV0(orquestagoal.GoalWorkStateFromLaunchRequestV0{
		RunRef: runRef,
		Spec: orquestagoal.GoalWorkSpecV0{
			SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
			GoalRef:       "goal-ref-stack-observe-repair-receipt-001",
			RunRef:        runRef,
			Objective:     "Publicar snapshot con repair receipt.",
			DirectorKind:  orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:      []orquestagoal.GoalWriteScopeV0{{Path: "temas/tema_031"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			Status:  orquestagoal.GoalStatusRunningV0,
			GoalRef: "goal-ref-stack-observe-repair-receipt-001",
		},
	})
	if err != nil {
		t.Fatalf("NewGoalWorkStateFromLaunchV0: %v", err)
	}
	if err := goalStates.SaveGoalWorkStateV0(ctx, state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	stack := &StackV0{
		Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			GoalStateStore:       goalStates,
			GoalClosureValidator: orquestagoal.DefaultGoalWorkClosureValidatorV0{},
		},
		Codex: CodexRuntimeConfigV0{ProjectWorkDir: projectDir},
	}

	result, err := NewCodexStackObserveAppDirectorGoalExecutorV0(stack).ObserveAppDirectorGoalTimeoutSnapshotV0(
		ctx,
		orquestamcp.MCPObserveAppDirectorGoalToolInputV0{RunRef: runRef},
	)

	if err != nil {
		t.Fatalf("ObserveAppDirectorGoalTimeoutSnapshotV0: %v", err)
	}
	if result.RecommendedAction != "no_action_closed" ||
		!result.ClosureAccepted ||
		result.ClosureNeedsRework ||
		result.ClosureStatus != orquestagoal.GoalStatusAcceptedV0 ||
		result.GoalStatus != orquestagoal.GoalStatusCompleteV0 ||
		!codexStackStringInSetForTestV0(result.EvidenceRefs, goalFirstRepairReceiptAcceptedEvidenceRefV0) ||
		result.ResultRef == "" {
		t.Fatalf("result=%+v", result)
	}
	for _, issue := range result.ClosureIssues {
		if issue.Code == orquestamcp.MCPGoalFirstMissingTerminalReceiptAfterArtifactsPassV0 {
			t.Fatalf("missing receipt no debe sobrevivir tras repair: result=%+v", result)
		}
	}
	persisted, err := goalStates.LoadGoalWorkStateV0(ctx, runRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if persisted.Status != orquestagoal.GoalStatusCompleteV0 ||
		persisted.LastResult == nil ||
		persisted.LastClosure == nil ||
		!persisted.LastClosure.Accepted ||
		!codexStackStringInSetForTestV0(persisted.LastResult.EvidenceRefs, goalFirstRepairReceiptAttemptedEvidenceRefV0) {
		t.Fatalf("persisted=%+v", persisted)
	}
}

func TestCodexStackObserveAppDirectorGoalExecutorV0IngiereReceiptTerminalMaterializadoV0(t *testing.T) {
	ctx := context.Background()
	runRef := "run-ref-stack-observe-goal-terminal-receipt-001"
	goalRef := "goal-ref-stack-observe-terminal-receipt-001"
	testRef := "test-ref-terminal-receipt-generated-app"
	projectDir := t.TempDir()
	appDir := filepath.Join(projectDir, "generated-apps", "smoke-goal-first")
	docsDir := filepath.Join(appDir, "docs")
	if err := os.MkdirAll(docsDir, 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "README.md"), []byte("# App\n"), 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	receipt := `{
		"schema_version":"orquesta_goal_result.v0",
		"status":"complete",
		"summary":"receipt terminal materializado",
		"artifact_refs":["artifact-ref-terminal-source","artifact-ref-terminal-handoff"],
		"artifact_paths":["generated-apps/smoke-goal-first/README.md"],
		"required_test_results":[{"test_ref":"` + testRef + `","status":"passed","evidence_refs":["evidence-ref-terminal-test"]}],
		"evidence_refs":["evidence-ref-terminal-receipt"]
	}`
	if err := os.WriteFile(filepath.Join(docsDir, "orquesta_goal_result_v0.json"), []byte(receipt), 0o600); err != nil {
		t.Fatalf("write receipt: %v", err)
	}
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	state, err := orquestagoal.NewGoalWorkStateFromLaunchV0(orquestagoal.GoalWorkStateFromLaunchRequestV0{
		RunRef: runRef,
		Spec: orquestagoal.GoalWorkSpecV0{
			SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
			GoalRef:       goalRef,
			RunRef:        runRef,
			Objective:     "Ingerir receipt terminal ya materializado.",
			DirectorKind:  orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:      []orquestagoal.GoalWriteScopeV0{{Path: "generated-apps/smoke-goal-first"}},
			RequiredTests: []orquestagoal.GoalRequiredTestV0{{TestRef: testRef}},
			ClosurePolicy: orquestagoal.GoalClosurePolicyV0{RequireRequiredTests: true, RequireArtifacts: true, RequiredEvidenceRefs: []string{"evidence-ref-terminal-receipt"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			Status:  orquestagoal.GoalStatusRunningV0,
			GoalRef: goalRef,
		},
	})
	if err != nil {
		t.Fatalf("NewGoalWorkStateFromLaunchV0: %v", err)
	}
	if err := goalStates.SaveGoalWorkStateV0(ctx, state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	stack := &StackV0{
		Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			GoalStateStore:       goalStates,
			GoalClosureValidator: orquestagoal.DefaultGoalWorkClosureValidatorV0{},
		},
		Codex: CodexRuntimeConfigV0{ProjectWorkDir: projectDir},
	}

	result, err := NewCodexStackObserveAppDirectorGoalExecutorV0(stack).ObserveAppDirectorGoalTimeoutSnapshotV0(
		ctx,
		orquestamcp.MCPObserveAppDirectorGoalToolInputV0{RunRef: runRef},
	)

	if err != nil {
		t.Fatalf("ObserveAppDirectorGoalTimeoutSnapshotV0: %v", err)
	}
	if result.GoalStatus != orquestagoal.GoalStatusCompleteV0 ||
		result.ClosureStatus != orquestagoal.GoalStatusAcceptedV0 ||
		!result.ClosureAccepted ||
		result.ClosureNeedsRework ||
		!codexStackStringInSetForTestV0(result.EvidenceRefs, goalFirstRepairReceiptAcceptedEvidenceRefV0) {
		t.Fatalf("result=%+v", result)
	}
	persisted, err := goalStates.LoadGoalWorkStateV0(ctx, runRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if persisted.LastResult == nil ||
		persisted.LastClosure == nil ||
		!persisted.LastClosure.Accepted ||
		!codexStackStringInSetForTestV0(persisted.LastResult.EvidenceRefs, goalFirstRepairReceiptAttemptedEvidenceRefV0) {
		t.Fatalf("persisted=%+v", persisted)
	}
}

func TestCodexStackObserveAppDirectorGoalExecutorV0RepairReceiptRequiereReworkSinReceiptDominioV0(t *testing.T) {
	ctx := context.Background()
	runRef := "run-ref-stack-observe-goal-repair-receipt-rework-001"
	projectDir := t.TempDir()
	topicDir := filepath.Join(projectDir, "temas", "tema_034")
	validationDir := filepath.Join(topicDir, "09_validacion")
	if err := os.MkdirAll(validationDir, 0o700); err != nil {
		t.Fatalf("mkdir validation: %v", err)
	}
	if err := os.WriteFile(filepath.Join(topicDir, "tema_ampliado.md"), []byte("# Tema\n"), 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(validationDir, "informe_qa.json"),
		[]byte(`{"qa_passes":{"extension_pass":true,"official_text_qa_pass":true,"strict_editorial_qa_pass":true,
				"question_bank_publicable":true,
				"tutor_assets_publicable":true}}`),
		0o600,
	); err != nil {
		t.Fatalf("write qa: %v", err)
	}
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	state, err := orquestagoal.NewGoalWorkStateFromLaunchV0(orquestagoal.GoalWorkStateFromLaunchRequestV0{
		RunRef: runRef,
		Spec: orquestagoal.GoalWorkSpecV0{
			SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
			GoalRef:       "goal-ref-stack-observe-repair-receipt-rework-001",
			RunRef:        runRef,
			Objective:     "No aceptar repair sin receipt de dominio requerido.",
			DirectorKind:  orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:      []orquestagoal.GoalWriteScopeV0{{Path: "temas/tema_034"}},
			ClosurePolicy: orquestagoal.GoalClosurePolicyV0{
				RequireDomainReceipt: true,
			},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			Status:  orquestagoal.GoalStatusRunningV0,
			GoalRef: "goal-ref-stack-observe-repair-receipt-rework-001",
		},
	})
	if err != nil {
		t.Fatalf("NewGoalWorkStateFromLaunchV0: %v", err)
	}
	if err := goalStates.SaveGoalWorkStateV0(ctx, state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	stack := &StackV0{
		Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			GoalStateStore:       goalStates,
			GoalClosureValidator: orquestagoal.DefaultGoalWorkClosureValidatorV0{},
		},
		Codex: CodexRuntimeConfigV0{ProjectWorkDir: projectDir},
	}

	result, err := NewCodexStackObserveAppDirectorGoalExecutorV0(stack).ObserveAppDirectorGoalTimeoutSnapshotV0(
		ctx,
		orquestamcp.MCPObserveAppDirectorGoalToolInputV0{RunRef: runRef},
	)

	if err != nil {
		t.Fatalf("ObserveAppDirectorGoalTimeoutSnapshotV0: %v", err)
	}
	if result.RecommendedAction != "replan" ||
		result.ClosureAccepted ||
		!result.ClosureNeedsRework ||
		result.ClosureStatus != orquestagoal.GoalStatusBlockedV0 ||
		!codexStackStringInSetForTestV0(result.EvidenceRefs, goalFirstRepairReceiptRequiresReworkEvidenceRefV0) {
		t.Fatalf("result=%+v", result)
	}
	if len(result.ClosureIssues) == 0 ||
		!codexStackValidationIssueCodeInSetForTestV0(result.ClosureIssues, goalFirstRepairReceiptRequiresReworkIssueV0) {
		t.Fatalf("closure issues=%+v", result.ClosureIssues)
	}
	persisted, err := goalStates.LoadGoalWorkStateV0(ctx, runRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if persisted.Status != orquestagoal.GoalStatusCompleteV0 ||
		persisted.LastClosure == nil ||
		!persisted.LastClosure.NeedsRework ||
		!goalFirstReceiptRepairHasIssueV0(persisted.LastClosure.Issues, goalFirstRepairReceiptRequiresReworkIssueV0) {
		t.Fatalf("persisted=%+v", persisted)
	}
}

func TestCodexStackAutoprogrammingObserveGoalExecutorV0ErrorPublicoIncluyeSnapshotParcialV0(t *testing.T) {
	ctx := context.Background()
	runRef := "run-ref-stack-autoprogramming-observe-rejected-001"
	goalRef := "goal-ref-stack-autoprogramming-observe-rejected-001"
	externalGoalRef := "thread-ref-stack-autoprogramming-observe-rejected-001"
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	state, err := orquestagoal.NewGoalWorkStateFromLaunchV0(orquestagoal.GoalWorkStateFromLaunchRequestV0{
		RunRef: runRef,
		Spec: orquestagoal.GoalWorkSpecV0{
			SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
			GoalRef:       goalRef,
			RunRef:        runRef,
			Objective:     "Autoprogramacion debe publicar snapshot local si Codex rechaza observar.",
			DirectorKind:  orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:      []orquestagoal.GoalWriteScopeV0{{Path: "docs/goal_result.md"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			Status:          orquestagoal.GoalStatusRunningV0,
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
			EvidenceRefs:    []string{"evidence-ref-stack-autoprogramming-observe-rejected-launch-001"},
		},
	})
	if err != nil {
		t.Fatalf("NewGoalWorkStateFromLaunchV0: %v", err)
	}
	if err := goalStates.SaveGoalWorkStateV0(ctx, state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	processRegistry := orquestaagentprocessregistrymemory.NewInMemoryAgentProcessRegistryV0()
	if err := processRegistry.RecordAgentProcessV0(ctx, orquestaagentprocessregistrymemory.AgentProcessRecordV0{
		RunID:          runRef,
		AgentRequestID: "agent-ref-stack-autoprogramming-observe-rejected-001",
		ProcessRef:     "process-ref-stack-autoprogramming-observe-rejected-001",
		SessionRef:     "session-ref-stack-autoprogramming-observe-rejected-001",
		LaunchRef:      "launch-ref-stack-autoprogramming-observe-rejected-001",
		ReadinessRef:   "readiness-ref-stack-autoprogramming-observe-rejected-001",
		EvidenceRefs:   []string{"evidence-ref-stack-autoprogramming-observe-rejected-process-001"},
	}); err != nil {
		t.Fatalf("RecordAgentProcessV0: %v", err)
	}
	stack := &StackV0{
		Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			GoalStateStore: goalStates,
			GoalObserver: codexStackObserveRejectedForTestV0{result: orquestagoal.GoalWorkResultV0{
				SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
				Status:          orquestagoal.GoalStatusInvalidV0,
				GoalRef:         goalRef,
				ExternalGoalRef: externalGoalRef,
				Issues: []orquestagoal.GoalWorkIssueV0{{
					Code:  "codex_goal_observation_rejected",
					Field: "codex_goal_backend",
				}},
			}},
		},
		Stores: StoresV0{
			ProcessRegistry: processRegistry,
		},
	}

	result, err := NewCodexStackAutoprogrammingObserveGoalExecutorV0(stack).Execute(
		ctx,
		orquestamcp.MCPAutoprogrammingObserveGoalToolInputV0{
			RequestID: "request-ref-stack-autoprogramming-observe-rejected-001",
			RunRef:    runRef,
		},
	)

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != orquestamcp.MCPAutoprogrammingObserveGoalEstadoErrorV0 ||
		!result.Partial ||
		result.GoalRef != goalRef ||
		result.ExternalGoalRef != externalGoalRef ||
		result.GoalStatus != orquestagoal.GoalStatusInvalidV0 ||
		result.RecommendedAction != "observe_estado_vivo" ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "codex_goal_observation_rejected" ||
		!codexStackStringInSetForTestV0(result.ProcessRefs, "process-ref-stack-autoprogramming-observe-rejected-001") ||
		!codexStackStringInSetForTestV0(result.EvidenceRefs, "evidence-ref-stack-autoprogramming-observe-rejected-process-001") {
		t.Fatalf("result=%+v", result)
	}
}

func TestCodexStackAutoprogrammingObserveGoalExecutorV0RecuperaSucesorRunningTrasErrorRawV0(t *testing.T) {
	ctx := context.Background()
	runRef := "run-ref-stack-autoprogramming-observe-successor-001"
	goalRef := "goal-ref-stack-autoprogramming-observe-successor-001-rework-1"
	externalGoalRef := "thread-ref-stack-autoprogramming-observe-successor-001"
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	durableStates := &codexStackCASGoalStateStoreForObserveTestV0{goalFirstQueueStateStoreForTestV0: goalStates}
	state, err := orquestagoal.NewGoalWorkStateFromLaunchV0(orquestagoal.GoalWorkStateFromLaunchRequestV0{
		RunRef: runRef,
		Spec: orquestagoal.GoalWorkSpecV0{
			SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0, GoalRef: goalRef, RunRef: runRef,
			Objective: "Recuperar el sucesor causal ya materializado.", DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet: []orquestagoal.GoalWriteScopeV0{{Path: "docs/goal_result.md"}},
			ContextRefs: []orquestagoal.GoalContextRefV0{
				{Kind: "goal", Ref: "goal-ref-stack-autoprogramming-observe-successor-001", Required: true},
				{Kind: "closure", Ref: "closure-ref-stack-autoprogramming-observe-successor-001", Required: true},
			},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			Status: orquestagoal.GoalStatusRunningV0, GoalRef: goalRef, ExternalGoalRef: externalGoalRef,
			EvidenceRefs: []string{"evidence-ref-stack-autoprogramming-observe-successor-launch-001"},
		},
	})
	if err != nil {
		t.Fatalf("NewGoalWorkStateFromLaunchV0: %v", err)
	}
	state.StoreVersion = 1
	if err := goalStates.SaveGoalWorkStateV0(ctx, state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	if err := goalStates.SaveGoalWorkRunMarkerV0(ctx, orquestagoal.GoalWorkRunMarkerV0{
		RunRef: runRef, GoalRef: goalRef, ExternalGoalRef: externalGoalRef,
		DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0, Status: orquestagoal.GoalStatusRunningV0,
		Spec: &state.Spec, LaunchReceipt: &state.LaunchReceipt,
		EvidenceRefs: []string{"evidence-ref-stack-autoprogramming-observe-successor-marker-001"},
	}); err != nil {
		t.Fatalf("SaveGoalWorkRunMarkerV0: %v", err)
	}
	observer := &codexStackObserveRawFailureForTestV0{}
	stack := &StackV0{Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
		GoalStateStore: durableStates, GoalFirstRunMarkerStore: durableStates, GoalObserver: observer,
	}, Stores: StoresV0{AppGoalStateStore: durableStates}}

	result, err := NewCodexStackAutoprogrammingObserveGoalExecutorV0(stack).Execute(ctx,
		orquestamcp.MCPAutoprogrammingObserveGoalToolInputV0{RequestID: "request-ref-stack-autoprogramming-observe-successor-001", RunRef: runRef})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if observer.calls != 1 || result.Estado != orquestamcp.MCPAutoprogrammingObserveGoalEstadoOKV0 ||
		!result.Partial || result.GoalRef != goalRef || result.GoalStatus != orquestagoal.GoalStatusRunningV0 ||
		result.RecommendedAction != "observe_later" || result.Summary != autoprogrammingObserveSuccessorRecoveryReasonCodeV0 ||
		!codexStackStringInSetForTestV0(result.EvidenceRefs, autoprogrammingObserveSuccessorRecoveryEvidenceRefV0) ||
		!codexStackStringInSetForTestV0(result.EvidenceRefs, "evidence-ref-autoprogramming-observe-error-sha256-4293458cbfb96c5a316518ded1c9b6a62b4530cd1a508f8e4f14421d87451d4f") ||
		!codexStackStringInSetForTestV0(result.EvidenceRefs, "evidence-ref-stack-autoprogramming-observe-successor-marker-001") {
		t.Fatalf("result=%+v calls=%d", result, observer.calls)
	}
	persisted, err := durableStates.LoadGoalWorkStateV0(ctx, runRef)
	if err != nil || durableStates.casCalls != 1 ||
		!codexStackStringInSetForTestV0(persisted.EvidenceRefs, autoprogrammingObserveErrorDiagnosticEvidenceRefV0(errors.New("raw_observer_failure"))) ||
		strings.Contains(strings.Join(persisted.EvidenceRefs, " "), "raw_observer_failure") {
		t.Fatalf("persisted=%+v calls=%d err=%v", persisted, durableStates.casCalls, err)
	}
	if strings.Contains(strings.Join(result.EvidenceRefs, " "), "raw_observer_failure") {
		t.Fatalf("result leaked raw observer error: %+v", result)
	}
}

func TestCodexStackAutoprogrammingObserveGoalExecutorV0RecoveryDiagnosticFailClosedV0(t *testing.T) {
	ctx := context.Background()
	runRef := "run-ref-stack-autoprogramming-observe-diagnostic-no-cas-001"
	goalRef := "goal-ref-stack-autoprogramming-observe-diagnostic-no-cas-001-rework-2"
	externalGoalRef := "thread-ref-stack-autoprogramming-observe-diagnostic-no-cas-001"
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	state, err := orquestagoal.NewGoalWorkStateFromLaunchV0(orquestagoal.GoalWorkStateFromLaunchRequestV0{
		RunRef: runRef,
		Spec: orquestagoal.GoalWorkSpecV0{SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0, GoalRef: goalRef, RunRef: runRef,
			Objective: "Sin CAS no se recupera el sucesor.", DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:    []orquestagoal.GoalWriteScopeV0{{Path: "docs/goal_result.md"}},
			ContextRefs: []orquestagoal.GoalContextRefV0{{Kind: "goal", Ref: "goal-ref-stack-autoprogramming-observe-diagnostic-no-cas-001-rework-1", Required: true}, {Kind: "closure", Ref: "closure-ref-stack-autoprogramming-observe-diagnostic-no-cas-001", Required: true}}},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{Status: orquestagoal.GoalStatusRunningV0, GoalRef: goalRef, ExternalGoalRef: externalGoalRef},
	})
	state.StoreVersion = 1
	if err != nil || goalStates.SaveGoalWorkStateV0(ctx, state) != nil || goalStates.SaveGoalWorkRunMarkerV0(ctx, orquestagoal.GoalWorkRunMarkerV0{RunRef: runRef, GoalRef: goalRef, ExternalGoalRef: externalGoalRef, DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0, Status: orquestagoal.GoalStatusRunningV0, Spec: &state.Spec, LaunchReceipt: &state.LaunchReceipt}) != nil {
		t.Fatalf("preparar sucesor sin CAS: %v", err)
	}
	_, err = NewCodexStackAutoprogrammingObserveGoalExecutorV0(&StackV0{Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
		GoalStateStore: goalStates, GoalFirstRunMarkerStore: goalStates, GoalObserver: &codexStackObserveRawFailureForTestV0{},
	}}).Execute(ctx, orquestamcp.MCPAutoprogrammingObserveGoalToolInputV0{RunRef: runRef})
	if err == nil || err.Error() != "raw_observer_failure" {
		t.Fatalf("err=%v", err)
	}
	persisted, loadErr := goalStates.LoadGoalWorkStateV0(ctx, runRef)
	if loadErr != nil || len(persisted.EvidenceRefs) != 0 {
		t.Fatalf("persisted=%+v loadErr=%v", persisted, loadErr)
	}

	t.Run("cas_sin_confirmacion_no_oculta_error_raw", func(t *testing.T) {
		durableStates := &codexStackCASGoalStateStoreForObserveTestV0{
			goalFirstQueueStateStoreForTestV0: newGoalFirstQueueStateStoreForTestV0(),
			discardCASWrite:                   true,
		}
		if err := durableStates.SaveGoalWorkStateV0(ctx, state); err != nil {
			t.Fatalf("SaveGoalWorkStateV0: %v", err)
		}
		if err := durableStates.SaveGoalWorkRunMarkerV0(ctx, orquestagoal.GoalWorkRunMarkerV0{
			RunRef: runRef, GoalRef: goalRef, ExternalGoalRef: externalGoalRef,
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0, Status: orquestagoal.GoalStatusRunningV0,
			Spec: &state.Spec, LaunchReceipt: &state.LaunchReceipt,
		}); err != nil {
			t.Fatalf("SaveGoalWorkRunMarkerV0: %v", err)
		}
		_, err := NewCodexStackAutoprogrammingObserveGoalExecutorV0(&StackV0{Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			GoalStateStore: durableStates, GoalFirstRunMarkerStore: durableStates, GoalObserver: &codexStackObserveRawFailureForTestV0{},
		}}).Execute(ctx, orquestamcp.MCPAutoprogrammingObserveGoalToolInputV0{RunRef: runRef})
		if err == nil || err.Error() != "raw_observer_failure" || durableStates.casCalls != 1 {
			t.Fatalf("err=%v casCalls=%d", err, durableStates.casCalls)
		}
		persisted, loadErr := durableStates.LoadGoalWorkStateV0(ctx, runRef)
		if loadErr != nil || len(persisted.EvidenceRefs) != 0 {
			t.Fatalf("persisted=%+v loadErr=%v", persisted, loadErr)
		}
	})
}

func TestAutoprogrammingObserveRunningSuccessorAccreditedV0Matrix(t *testing.T) {
	state, err := orquestagoal.NewGoalWorkStateFromLaunchV0(orquestagoal.GoalWorkStateFromLaunchRequestV0{
		RunRef: "run-ref-stack-observe-accreditation-matrix-001",
		Spec: orquestagoal.GoalWorkSpecV0{
			SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
			GoalRef:       "goal-ref-stack-observe-accreditation-matrix-001-rework-2",
			RunRef:        "run-ref-stack-observe-accreditation-matrix-001",
			Objective:     "Acreditar solo el sucesor inmediato y causal.",
			DirectorKind:  orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:      []orquestagoal.GoalWriteScopeV0{{Path: "docs/goal_result.md"}},
			ContextRefs: []orquestagoal.GoalContextRefV0{
				{Kind: "goal", Ref: "goal-ref-stack-observe-accreditation-matrix-001-rework-1", Required: true},
				{Kind: "closure", Ref: "closure-ref-stack-observe-accreditation-matrix-001", Required: true},
			},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			Status: orquestagoal.GoalStatusRunningV0, GoalRef: "goal-ref-stack-observe-accreditation-matrix-001-rework-2",
			ExternalGoalRef: "thread-ref-stack-observe-accreditation-matrix-001",
		},
	})
	if err != nil {
		t.Fatalf("NewGoalWorkStateFromLaunchV0: %v", err)
	}
	state.StoreVersion = 1
	marker := orquestagoal.GoalWorkRunMarkerV0{
		RunRef: state.RunRef, GoalRef: state.GoalRef, ExternalGoalRef: state.ExternalGoalRef,
		DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0, Status: orquestagoal.GoalStatusRunningV0,
		Spec: &state.Spec, LaunchReceipt: &state.LaunchReceipt,
	}
	cases := []struct {
		name   string
		mutate func(*orquestagoal.GoalWorkStateV0, *orquestagoal.GoalWorkRunMarkerV0)
		want   bool
	}{
		{name: "causal_immediate_rework_two", want: true},
		{name: "store_version_zero", mutate: func(state *orquestagoal.GoalWorkStateV0, _ *orquestagoal.GoalWorkRunMarkerV0) {
			state.StoreVersion = 0
		}},
		{name: "director_kind_crossed", mutate: func(_ *orquestagoal.GoalWorkStateV0, marker *orquestagoal.GoalWorkRunMarkerV0) {
			marker.DirectorKind = orquestagoal.GoalDirectorKindRuntimeGoalV0
		}},
		{name: "marker_spec_missing", mutate: func(_ *orquestagoal.GoalWorkStateV0, marker *orquestagoal.GoalWorkRunMarkerV0) {
			marker.Spec = nil
		}},
		{name: "marker_receipt_missing", mutate: func(_ *orquestagoal.GoalWorkStateV0, marker *orquestagoal.GoalWorkRunMarkerV0) {
			marker.LaunchReceipt = nil
		}},
		{name: "marker_spec_stale", mutate: func(_ *orquestagoal.GoalWorkStateV0, marker *orquestagoal.GoalWorkRunMarkerV0) {
			marker.Spec.Objective = "marcador obsoleto"
		}},
		{name: "marker_receipt_stale", mutate: func(_ *orquestagoal.GoalWorkStateV0, marker *orquestagoal.GoalWorkRunMarkerV0) {
			marker.LaunchReceipt.ExternalGoalRef = "thread-ref-stack-observe-accreditation-matrix-stale"
		}},
		{name: "goal_ref_duplicated", mutate: func(state *orquestagoal.GoalWorkStateV0, marker *orquestagoal.GoalWorkRunMarkerV0) {
			duplicate := state.Spec.ContextRefs[0]
			state.Spec.ContextRefs = append(state.Spec.ContextRefs, duplicate)
			marker.Spec.ContextRefs = append(marker.Spec.ContextRefs, duplicate)
		}},
		{name: "closure_ref_duplicated", mutate: func(state *orquestagoal.GoalWorkStateV0, marker *orquestagoal.GoalWorkRunMarkerV0) {
			duplicate := state.Spec.ContextRefs[1]
			state.Spec.ContextRefs = append(state.Spec.ContextRefs, duplicate)
			marker.Spec.ContextRefs = append(marker.Spec.ContextRefs, duplicate)
		}},
		{name: "closure_missing", mutate: func(state *orquestagoal.GoalWorkStateV0, _ *orquestagoal.GoalWorkRunMarkerV0) {
			state.Spec.ContextRefs[1].Required = false
		}},
		{name: "parent_not_immediate", mutate: func(state *orquestagoal.GoalWorkStateV0, _ *orquestagoal.GoalWorkRunMarkerV0) {
			state.Spec.ContextRefs[0].Ref = "goal-ref-stack-observe-accreditation-matrix-001"
		}},
		{name: "receipt_goal_mismatch", mutate: func(state *orquestagoal.GoalWorkStateV0, _ *orquestagoal.GoalWorkRunMarkerV0) {
			state.LaunchReceipt.GoalRef = "goal-ref-otro-rework-2"
		}},
		{name: "external_mismatch", mutate: func(_ *orquestagoal.GoalWorkStateV0, marker *orquestagoal.GoalWorkRunMarkerV0) {
			marker.ExternalGoalRef = "thread-ref-otro"
		}},
		{name: "rework_leading_zero", mutate: func(state *orquestagoal.GoalWorkStateV0, marker *orquestagoal.GoalWorkRunMarkerV0) {
			codexStackReplaceAccreditedGoalRefForTestV0(state, marker, "goal-ref-stack-observe-accreditation-matrix-001-rework-01", "goal-ref-stack-observe-accreditation-matrix-001")
		}},
		{name: "rework_plus_sign", mutate: func(state *orquestagoal.GoalWorkStateV0, marker *orquestagoal.GoalWorkRunMarkerV0) {
			codexStackReplaceAccreditedGoalRefForTestV0(state, marker, "goal-ref-stack-observe-accreditation-matrix-001-rework-+1", "goal-ref-stack-observe-accreditation-matrix-001")
		}},
		{name: "rework_space", mutate: func(state *orquestagoal.GoalWorkStateV0, marker *orquestagoal.GoalWorkRunMarkerV0) {
			codexStackReplaceAccreditedGoalRefForTestV0(state, marker, "goal-ref-stack-observe-accreditation-matrix-001-rework-1 ", "goal-ref-stack-observe-accreditation-matrix-001")
		}},
		{name: "rework_marker_space", mutate: func(_ *orquestagoal.GoalWorkStateV0, marker *orquestagoal.GoalWorkRunMarkerV0) {
			marker.GoalRef = "goal-ref-stack-observe-accreditation-matrix-001-rework-2 "
			marker.Spec.GoalRef = marker.GoalRef
			marker.LaunchReceipt.GoalRef = marker.GoalRef
		}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			candidate := state
			candidate.Spec.ContextRefs = append([]orquestagoal.GoalContextRefV0(nil), state.Spec.ContextRefs...)
			candidateMarker := marker
			markerSpec := *marker.Spec
			markerReceipt := *marker.LaunchReceipt
			candidateMarker.Spec = &markerSpec
			candidateMarker.LaunchReceipt = &markerReceipt
			if test.mutate != nil {
				test.mutate(&candidate, &candidateMarker)
			}
			if got := autoprogrammingObserveRunningSuccessorAccreditedV0(candidate, candidateMarker); got != test.want {
				t.Fatalf("accredited=%t want=%t state=%+v marker=%+v", got, test.want, candidate, candidateMarker)
			}
		})
	}
}

func TestCodexStackAutoprogrammingObserveGoalExecutorV0RecoveryDiagnosticRetriesOnlyTypedCASConflictV0(t *testing.T) {
	ctx := context.Background()
	state := codexStackAccreditedSuccessorStateForObserveTestV0(t, "run-ref-stack-observe-typed-cas-001", "goal-ref-stack-observe-typed-cas-001-rework-1")
	t.Run("typed_conflict_reloads_and_retries_once", func(t *testing.T) {
		store := &codexStackCASGoalStateStoreForObserveTestV0{goalFirstQueueStateStoreForTestV0: newGoalFirstQueueStateStoreForTestV0(), conflictOnce: true}
		if err := store.SaveGoalWorkStateV0(ctx, state); err != nil || store.SaveGoalWorkRunMarkerV0(ctx, codexStackAccreditedMarkerForObserveTestV0(state)) != nil {
			t.Fatalf("prepare store: %v", err)
		}
		result, err := NewCodexStackAutoprogrammingObserveGoalExecutorV0(&StackV0{Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			GoalStateStore: store, GoalFirstRunMarkerStore: store, GoalObserver: &codexStackObserveRawFailureForTestV0{},
		}}).Execute(ctx, orquestamcp.MCPAutoprogrammingObserveGoalToolInputV0{RunRef: state.RunRef})
		if err != nil || store.casCalls != 2 || result.GoalRef != state.GoalRef {
			t.Fatalf("result=%+v err=%v casCalls=%d", result, err, store.casCalls)
		}
	})
	t.Run("non_conflict_error_does_not_retry", func(t *testing.T) {
		store := &codexStackCASGoalStateStoreForObserveTestV0{goalFirstQueueStateStoreForTestV0: newGoalFirstQueueStateStoreForTestV0(), forcedCASFailure: errors.New("cas_backend_failure")}
		if err := store.SaveGoalWorkStateV0(ctx, state); err != nil || store.SaveGoalWorkRunMarkerV0(ctx, codexStackAccreditedMarkerForObserveTestV0(state)) != nil {
			t.Fatalf("prepare store: %v", err)
		}
		_, err := NewCodexStackAutoprogrammingObserveGoalExecutorV0(&StackV0{Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			GoalStateStore: store, GoalFirstRunMarkerStore: store, GoalObserver: &codexStackObserveRawFailureForTestV0{},
		}}).Execute(ctx, orquestamcp.MCPAutoprogrammingObserveGoalToolInputV0{RunRef: state.RunRef})
		if err == nil || err.Error() != "raw_observer_failure" || store.casCalls != 1 {
			t.Fatalf("err=%v casCalls=%d", err, store.casCalls)
		}
	})
}

func TestCodexStackAutoprogrammingObserveGoalExecutorV0RecoveryDiagnosticStoreV0RaceV0(t *testing.T) {
	ctx := context.Background()
	state := codexStackAccreditedSuccessorStateForObserveTestV0(t, "run-ref-stack-observe-storev0-race-001", "goal-ref-stack-observe-storev0-race-001-rework-1")
	store, err := orquestastatefile.NewStoreV0(orquestastatefile.ConfigV0{RootDir: t.TempDir()})
	state.StoreVersion = 0
	if err != nil {
		t.Fatalf("prepare StoreV0: %v", err)
	}
	if err := store.SaveGoalWorkStateV0(ctx, state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	if err := store.SaveGoalWorkRunMarkerV0(ctx, codexStackAccreditedMarkerForObserveTestV0(state)); err != nil {
		t.Fatalf("SaveGoalWorkRunMarkerV0: %v", err)
	}
	executor := NewCodexStackAutoprogrammingObserveGoalExecutorV0(&StackV0{Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
		GoalStateStore: store, GoalFirstRunMarkerStore: store, GoalObserver: &codexStackObserveRawFailureForTestV0{},
	}})
	errs := make(chan error, 2)
	for range 2 {
		go func() {
			_, err := executor.Execute(ctx, orquestamcp.MCPAutoprogrammingObserveGoalToolInputV0{RunRef: state.RunRef})
			errs <- err
		}()
	}
	for range 2 {
		if err := <-errs; err != nil {
			t.Fatalf("Execute: %v", err)
		}
	}
	persisted, err := store.LoadGoalWorkStateV0(ctx, state.RunRef)
	if err != nil || !autoprogrammingObserveEvidenceRefPresentV0(persisted.EvidenceRefs, autoprogrammingObserveErrorDiagnosticEvidenceRefV0(errors.New("raw_observer_failure"))) {
		t.Fatalf("persisted=%+v err=%v", persisted, err)
	}
}

func TestCodexStackAutoprogrammingObserveGoalExecutorV0RecoveryDiagnosticStoreV0ReloadV0(t *testing.T) {
	ctx := context.Background()
	state := codexStackAccreditedSuccessorStateForObserveTestV0(t, "run-ref-stack-observe-storev0-reload-001", "goal-ref-stack-observe-storev0-reload-001-rework-1")
	state.StoreVersion = 0
	store, err := orquestastatefile.NewStoreV0(orquestastatefile.ConfigV0{RootDir: t.TempDir()})
	if err != nil {
		t.Fatalf("NewStoreV0: %v", err)
	}
	if err := store.SaveGoalWorkStateV0(ctx, state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	if err := store.SaveGoalWorkRunMarkerV0(ctx, codexStackAccreditedMarkerForObserveTestV0(state)); err != nil {
		t.Fatalf("SaveGoalWorkRunMarkerV0: %v", err)
	}
	observer := &codexStackObserveRawFailureForTestV0{}
	input := orquestamcp.MCPAutoprogrammingObserveGoalToolInputV0{RunRef: state.RunRef}
	if _, err := NewCodexStackAutoprogrammingObserveGoalExecutorV0(&StackV0{Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{GoalStateStore: store, GoalFirstRunMarkerStore: store, GoalObserver: observer}}).Execute(ctx, input); err != nil {
		t.Fatalf("first Execute: %v", err)
	}
	first, err := store.LoadGoalWorkStateV0(ctx, state.RunRef)
	if err != nil {
		t.Fatalf("first LoadGoalWorkStateV0: %v", err)
	}
	reopened, err := orquestastatefile.NewStoreV0(orquestastatefile.ConfigV0{RootDir: store.RootDirV0()})
	if err != nil {
		t.Fatalf("reopen StoreV0: %v", err)
	}
	if _, err := NewCodexStackAutoprogrammingObserveGoalExecutorV0(&StackV0{Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{GoalStateStore: reopened, GoalFirstRunMarkerStore: reopened, GoalObserver: observer}}).Execute(ctx, input); err != nil {
		t.Fatalf("second Execute: %v", err)
	}
	second, err := reopened.LoadGoalWorkStateV0(ctx, state.RunRef)
	if err != nil || first.StoreVersion != second.StoreVersion || observer.calls != 2 {
		t.Fatalf("first=%+v second=%+v calls=%d err=%v", first, second, observer.calls, err)
	}
}

func codexStackAccreditedSuccessorStateForObserveTestV0(t *testing.T, runRef, goalRef string) orquestagoal.GoalWorkStateV0 {
	t.Helper()
	parentRef := strings.TrimSuffix(goalRef, "-rework-1")
	state, err := orquestagoal.NewGoalWorkStateFromLaunchV0(orquestagoal.GoalWorkStateFromLaunchRequestV0{RunRef: runRef,
		Spec:          orquestagoal.GoalWorkSpecV0{SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0, GoalRef: goalRef, RunRef: runRef, Objective: "Acreditar sucesor.", DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0, WriteSet: []orquestagoal.GoalWriteScopeV0{{Path: "docs/goal_result.md"}}, ContextRefs: []orquestagoal.GoalContextRefV0{{Kind: "goal", Ref: parentRef, Required: true}, {Kind: "closure", Ref: "closure-ref-" + runRef, Required: true}}},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{Status: orquestagoal.GoalStatusRunningV0, GoalRef: goalRef, ExternalGoalRef: "thread-ref-" + runRef}})
	if err != nil {
		t.Fatalf("NewGoalWorkStateFromLaunchV0: %v", err)
	}
	state.StoreVersion = 1
	return state
}

func codexStackAccreditedMarkerForObserveTestV0(state orquestagoal.GoalWorkStateV0) orquestagoal.GoalWorkRunMarkerV0 {
	return orquestagoal.GoalWorkRunMarkerV0{RunRef: state.RunRef, GoalRef: state.GoalRef, ExternalGoalRef: state.ExternalGoalRef, DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0, Status: orquestagoal.GoalStatusRunningV0, Spec: &state.Spec, LaunchReceipt: &state.LaunchReceipt}
}

func codexStackReplaceAccreditedGoalRefForTestV0(state *orquestagoal.GoalWorkStateV0, marker *orquestagoal.GoalWorkRunMarkerV0, goalRef, parentRef string) {
	state.GoalRef, state.Spec.GoalRef, state.LaunchReceipt.GoalRef = goalRef, goalRef, goalRef
	state.Spec.ContextRefs[0].Ref = parentRef
	marker.GoalRef, marker.Spec.GoalRef, marker.LaunchReceipt.GoalRef = goalRef, goalRef, goalRef
	marker.Spec.ContextRefs[0].Ref = parentRef
}

func TestCodexStackAutoprogrammingObserveGoalExecutorV0ErrorRawSinSucesorPermaneceFailClosedV0(t *testing.T) {
	ctx := context.Background()
	runRef := "run-ref-stack-autoprogramming-observe-no-successor-001"
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	state, err := orquestagoal.NewGoalWorkStateFromLaunchV0(orquestagoal.GoalWorkStateFromLaunchRequestV0{
		RunRef: runRef,
		Spec: orquestagoal.GoalWorkSpecV0{
			SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0, GoalRef: "goal-ref-stack-autoprogramming-observe-no-successor-001", RunRef: runRef,
			Objective: "Un error raw sin sucesor no se recupera.", DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet: []orquestagoal.GoalWriteScopeV0{{Path: "docs/goal_result.md"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{Status: orquestagoal.GoalStatusRunningV0, GoalRef: "goal-ref-stack-autoprogramming-observe-no-successor-001"},
	})
	if err != nil {
		t.Fatalf("NewGoalWorkStateFromLaunchV0: %v", err)
	}
	if err := goalStates.SaveGoalWorkStateV0(ctx, state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	stack := &StackV0{Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
		GoalStateStore: goalStates, GoalFirstRunMarkerStore: goalStates, GoalObserver: &codexStackObserveRawFailureForTestV0{},
	}}
	_, err = NewCodexStackAutoprogrammingObserveGoalExecutorV0(stack).Execute(ctx,
		orquestamcp.MCPAutoprogrammingObserveGoalToolInputV0{RunRef: runRef})
	if err == nil || err.Error() != "raw_observer_failure" {
		t.Fatalf("err=%v", err)
	}
}

type codexStackObserveRejectedForTestV0 struct {
	result orquestagoal.GoalWorkResultV0
}

type codexStackObserveRawFailureForTestV0 struct {
	mu    sync.Mutex
	calls int
}

type codexStackCASGoalStateStoreForObserveTestV0 struct {
	*goalFirstQueueStateStoreForTestV0
	casCalls         int
	discardCASWrite  bool
	conflictOnce     bool
	forcedCASFailure error
}

func (store *codexStackCASGoalStateStoreForObserveTestV0) CompareAndSwapGoalWorkStateV0(
	_ context.Context,
	expectedVersion uint64,
	state orquestagoal.GoalWorkStateV0,
) (orquestagoal.GoalWorkStateV0, error) {
	current, err := store.LoadGoalWorkStateV0(context.Background(), state.RunRef)
	if err != nil {
		return orquestagoal.GoalWorkStateV0{}, err
	}
	if current.StoreVersion != expectedVersion {
		return orquestagoal.GoalWorkStateV0{}, orquestagoal.GoalWorkStateCASConflictErrorV0{RunRef: state.RunRef, ExpectedVersion: expectedVersion, CurrentVersion: current.StoreVersion}
	}
	if store.forcedCASFailure != nil {
		store.casCalls++
		return orquestagoal.GoalWorkStateV0{}, store.forcedCASFailure
	}
	if store.conflictOnce {
		store.conflictOnce = false
		current.StoreVersion++
		store.states[current.RunRef] = current
		store.casCalls++
		return orquestagoal.GoalWorkStateV0{}, orquestagoal.GoalWorkStateCASConflictErrorV0{RunRef: state.RunRef, ExpectedVersion: expectedVersion, CurrentVersion: current.StoreVersion}
	}
	state.StoreVersion = expectedVersion + 1
	normalized, err := orquestagoal.NewGoalWorkStateV0(state)
	if err != nil {
		return orquestagoal.GoalWorkStateV0{}, err
	}
	if !store.discardCASWrite {
		store.states[normalized.RunRef] = normalized
	}
	store.casCalls++
	return normalized, nil
}

func (observer *codexStackObserveRawFailureForTestV0) ObserveGoalWorkV0(
	_ context.Context,
	_ orquestagoal.GoalObservationRequestV0,
) (orquestagoal.GoalWorkResultV0, error) {
	observer.mu.Lock()
	defer observer.mu.Unlock()
	observer.calls++
	return orquestagoal.GoalWorkResultV0{}, errors.New("raw_observer_failure")
}

type codexStackRunControlReaderForObserveTestV0 struct {
	state orquestaruncontrol.RunControlStateV0
}

func (reader codexStackRunControlReaderForObserveTestV0) ReadRunControlStateV0(
	_ context.Context,
	_ orquestaruncontrol.RunControlReadRequestV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	return reader.state, nil
}

func (observer codexStackObserveRejectedForTestV0) ObserveGoalWorkV0(
	_ context.Context,
	_ orquestagoal.GoalObservationRequestV0,
) (orquestagoal.GoalWorkResultV0, error) {
	return observer.result, errors.New("codex_goal_observation_rejected")
}

func codexStackValidationIssueCodeInSetForTestV0(
	issues []orquestamcp.MCPValidationIssueV0,
	code string,
) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

type codexStackObserveRunningForTestV0 struct {
	result orquestagoal.GoalWorkResultV0
}

func (observer codexStackObserveRunningForTestV0) ObserveGoalWorkV0(
	_ context.Context,
	_ orquestagoal.GoalObservationRequestV0,
) (orquestagoal.GoalWorkResultV0, error) {
	return observer.result, nil
}

func TestCodexStackObserveAppDirectorGoalExecutorV0ExecuteCableaVeredictoCausalV0(t *testing.T) {
	ctx := context.Background()
	runRef := "run-ref-stack-observe-causal-wiring-001"
	goalRef := "goal-ref-stack-observe-causal-wiring-001"
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	state, err := orquestagoal.NewGoalWorkStateFromLaunchV0(orquestagoal.GoalWorkStateFromLaunchRequestV0{
		RunRef: runRef,
		Spec: orquestagoal.GoalWorkSpecV0{
			SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
			GoalRef:       goalRef,
			RunRef:        runRef,
			Objective:     "Execute debe publicar veredicto causal desde la fuente real.",
			DirectorKind:  orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:      []orquestagoal.GoalWriteScopeV0{{Path: "docs/causal_wiring.md"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			Status:          orquestagoal.GoalStatusRunningV0,
			GoalRef:         goalRef,
			ExternalGoalRef: "thread-ref-stack-observe-causal-wiring-001",
		},
	})
	if err != nil {
		t.Fatalf("NewGoalWorkStateFromLaunchV0: %v", err)
	}
	if err := goalStates.SaveGoalWorkStateV0(ctx, state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	processRegistry := orquestaagentprocessregistrymemory.NewInMemoryAgentProcessRegistryV0()
	if err := processRegistry.RecordAgentProcessV0(ctx, orquestaagentprocessregistrymemory.AgentProcessRecordV0{
		RunID:          runRef,
		AgentRequestID: "agent-ref-stack-observe-causal-wiring-001",
		ProcessRef:     "process-ref-stack-observe-causal-wiring-001",
		SessionRef:     "session-ref-stack-observe-causal-wiring-001",
		LaunchRef:      "launch-ref-stack-observe-causal-wiring-001",
		ReadinessRef:   "readiness-ref-stack-observe-causal-wiring-001",
		EvidenceRefs:   []string{"evidence-ref-stack-observe-causal-wiring-process"},
	}); err != nil {
		t.Fatalf("RecordAgentProcessV0: %v", err)
	}
	stack := &StackV0{
		Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			GoalStateStore: goalStates,
			GoalObserver: codexStackObserveRunningForTestV0{result: orquestagoal.GoalWorkResultV0{
				SchemaVersion: orquestagoal.GoalWorkResultSchemaV0,
				Status:        orquestagoal.GoalStatusRunningV0,
				GoalRef:       goalRef,
			}},
		},
		Stores: StoresV0{
			AppGoalStateStore: goalStates,
			ProcessRegistry:   processRegistry,
		},
		Codex: CodexRuntimeConfigV0{SnapshotSource: evidenciaEstadoSnapshotSourceForTestV0{
			snapshots: map[string]orquestaruntime.ProcessRuntimeSnapshotV0{
				"process-ref-stack-observe-causal-wiring-001": {
					SchemaVersion: orquestaruntime.ProcessRuntimeConnectorVersionV0,
					ProcessRef:    "process-ref-stack-observe-causal-wiring-001",
					SessionRef:    "session-ref-stack-observe-causal-wiring-001",
					LaunchRef:     "launch-ref-stack-observe-causal-wiring-001",
					Status:        orquestaruntime.ProcessRuntimeStoppedV0,
				},
			},
		}},
	}

	result, err := NewCodexStackObserveAppDirectorGoalExecutorV0(stack).Execute(
		ctx,
		orquestamcp.MCPObserveAppDirectorGoalToolInputV0{RunRef: runRef},
	)

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.CausalVerdict != "process_dead_state_stale" ||
		result.CausalReasonCode == "" ||
		result.GoalStatus == orquestagoal.GoalStatusRunningV0 ||
		result.RecommendedAction != "reconcile_goal_state" {
		t.Fatalf("veredicto causal no cableado en Execute: %+v", result)
	}
}
