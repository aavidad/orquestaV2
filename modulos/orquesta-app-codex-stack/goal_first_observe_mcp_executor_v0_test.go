package orquestaappcodexstack

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	orquestaagentprocessregistrymemory "orquesta/modulos/orquesta-agent-process-registry-memory"
	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
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

type codexStackObserveRejectedForTestV0 struct {
	result orquestagoal.GoalWorkResultV0
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
