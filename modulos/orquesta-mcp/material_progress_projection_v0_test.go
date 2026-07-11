package orquestamcp

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
)

func TestMCPAutoprogrammingStatusMaterialProgressPersistidoPrecedeStatsContradictoriosV0(t *testing.T) {
	tests := []struct {
		name              string
		action            orquestaautoprogramming.MaterialProgressActionV0
		wantCode          string
		wantSeverity      string
		observedTokens    int64
		observedArtifacts []string
	}{
		{"continue sobre high checkpoint", orquestaautoprogramming.MaterialProgressActionContinueV0, mcpAutoprogrammingActionMaterialProgressContinueV0, "info", 900000, []string{"artifact-ref-checkpoint:legacy-high"}},
		{"warning sobre high checkpoint", orquestaautoprogramming.MaterialProgressActionWarningV0, mcpAutoprogrammingActionMaterialProgressWarningV0, "warning", 900000, []string{"artifact-ref-checkpoint:legacy-high"}},
		{"replan sobre stats bajos", orquestaautoprogramming.MaterialProgressActionReplanRequiredV0, mcpAutoprogrammingActionMaterialProgressReplanV0, "blocked", 1, []string{"artifact-ref-result:legacy-low"}},
		{"hard stop sobre stats bajos", orquestaautoprogramming.MaterialProgressActionHardStopRequiredV0, mcpAutoprogrammingActionMaterialProgressHardStopV0, "blocked", 1, []string{"artifact-ref-result:legacy-low"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runRef := "run-ref-material-progress-status-" + strings.ReplaceAll(string(test.action), "_", "-")
			goalRef := "goal-ref-material-progress-status-" + strings.ReplaceAll(string(test.action), "_", "-")
			state := materialProgressStateForMCPTestV0(t, runRef, goalRef, test.action)
			reader := &materialProgressReaderForMCPTestV0{states: map[string]orquestaautoprogramming.MaterialProgressStateV0{
				materialProgressReaderKeyForMCPTestV0(runRef, goalRef): state,
			}}
			goalStates := materialProgressGoalStateStoreForMCPTestV0(t, runRef, goalRef)
			stats := &fakeMCPAutoprogrammingRunStatusV0{
				stats: &orquestacionnucleoapp.DirectorRunStatsV0{
					RunRef:       runRef,
					Status:       "running",
					UsageSummary: &orquestacionnucleoapp.DirectorRunUsageStatsV0{TotalTokens: test.observedTokens},
				},
				goal: &MCPDirectorGoalStatsV0{
					RunRef: runRef, GoalRef: goalRef, ExternalGoalRef: "thread-ref-" + goalRef,
					Status: "active", ArtifactRefs: test.observedArtifacts,
				},
			}

			result, err := (MCPAutoprogrammingStatusToolExecutorV0{
				Queue:                       &fakeMCPAutoprogrammingQueueStatusV0{empty: true},
				Stats:                       stats,
				GoalStateStore:              goalStates,
				MaterialProgressStateReader: reader,
			}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{RunRef: runRef})
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			action, ok := materialProgressActionForMCPTestV0(result.StaleRunning, test.wantCode)
			if !ok {
				t.Fatalf("persisted action %s missing: %+v", test.wantCode, result.StaleRunning)
			}
			if action.TokensUsed != state.LastDecision.TokensWithoutMaterial ||
				action.Severity != test.wantSeverity ||
				action.RunStatus != "running" ||
				action.GoalStatus != "active" ||
				len(action.ArtifactRefs) != 1 || action.ArtifactRefs[0] != state.LastCheckpointRef {
				t.Fatalf("action does not prefer persisted decision: %+v state=%+v", action, state)
			}
			for _, candidate := range result.StaleRunning {
				if candidate.RunRef == runRef && mcpAutoprogrammingLegacyMaterialProgressActionV0(candidate.Code) {
					t.Fatalf("legacy material inference survived persisted state: %+v", candidate)
				}
			}
		})
	}
}

func TestMCPRunControlMaterialProgressPersistidoPrecedeStatsContradictoriosV0(t *testing.T) {
	tests := []struct {
		name           string
		action         orquestaautoprogramming.MaterialProgressActionV0
		observedTokens int64
		observedRefs   []string
		wantReconciled bool
	}{
		{"continue blocks legacy reconcile", orquestaautoprogramming.MaterialProgressActionContinueV0, 900000, []string{"artifact-ref-checkpoint:legacy-high"}, false},
		{"warning blocks legacy reconcile", orquestaautoprogramming.MaterialProgressActionWarningV0, 900000, []string{"artifact-ref-checkpoint:legacy-high"}, false},
		{"replan forces reconcile", orquestaautoprogramming.MaterialProgressActionReplanRequiredV0, 1, []string{"artifact-ref-result:legacy-low"}, true},
		{"hard stop forces reconcile", orquestaautoprogramming.MaterialProgressActionHardStopRequiredV0, 1, []string{"artifact-ref-result:legacy-low"}, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runRef := "run-ref-material-progress-control-" + strings.ReplaceAll(string(test.action), "_", "-")
			goalRef := "goal-ref-material-progress-control-" + strings.ReplaceAll(string(test.action), "_", "-")
			progressState := materialProgressStateForMCPTestV0(t, runRef, goalRef, test.action)
			goalStates := materialProgressGoalStateStoreForMCPTestV0(t, runRef, goalRef)
			port := &fakeMCPRunControlPortV0{
				readState:  runControlStateForMCPTestV0(runRef, orquestaruncontrol.RunControlStatusRunningV0, false),
				stopStatus: orquestaruncontrol.RunControlStatusStoppedV0,
			}
			executor := MCPRunControlToolExecutorV0{
				Port:           port,
				GoalStateStore: goalStates,
				MaterialProgressStateReader: &materialProgressReaderForMCPTestV0{states: map[string]orquestaautoprogramming.MaterialProgressStateV0{
					materialProgressReaderKeyForMCPTestV0(runRef, goalRef): progressState,
				}},
				GoalBackendState: &fakeMCPRunControlGoalBackendStateV0{results: []MCPDirectorStatsToolResultV0{
					mcpRunControlGoalBackendStatsWithUsageForTestV0(runRef, goalRef, "active", test.observedTokens, test.observedRefs, nil),
					{Estado: MCPDirectorStatsEstadoOKV0, RunRef: runRef},
				}},
			}

			result, err := executor.Execute(context.Background(), MCPRunControlToolInputV0{
				RequestID: "request-ref-" + runRef, Action: "stop", RunRef: runRef,
				RequestedBy: "orquesta-director", Reason: "persisted material progress decision", Forced: true,
			})
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			goalState, err := goalStates.LoadGoalWorkStateV0(context.Background(), runRef)
			if err != nil {
				t.Fatalf("LoadGoalWorkStateV0: %v", err)
			}
			if !test.wantReconciled {
				if goalState.Status != orquestagoal.GoalStatusRunningV0 ||
					containsMCPRunControlDiagnosticForTestV0(result.Diagnostics, "goal_state_terminal_reconciled_after_forced_stop") {
					t.Fatalf("legacy stats reconciled persisted %s: result=%+v state=%+v", test.action, result, goalState)
				}
				return
			}
			if goalState.Status != orquestagoal.GoalStatusBlockedV0 || goalState.LastResult == nil ||
				!materialProgressGoalIssueForMCPTestV0(goalState.LastResult.Issues, string(test.action)) ||
				!containsStringMCPTestV0(goalState.EvidenceRefs, progressState.LastCheckpointRef) ||
				!containsMCPRunControlDiagnosticForTestV0(result.Diagnostics, "goal_state_terminal_reconciled_after_forced_stop") {
				t.Fatalf("persisted %s did not reconcile: result=%+v state=%+v", test.action, result, goalState)
			}
		})
	}
}

func TestMCPMaterialProgressFallbackSoloReaderNilONotFoundV0(t *testing.T) {
	nilReader := mcpMaterialProgressStateForRunGoalV0(context.Background(), nil, "run-ref-fallback", "goal-ref-fallback")
	if !nilReader.LegacyFallback || nilReader.DiagnosticCode != "material_progress_state_reader_unbound" {
		t.Fatalf("nil reader=%+v", nilReader)
	}
	notFound := mcpMaterialProgressStateForRunGoalV0(context.Background(), &materialProgressReaderForMCPTestV0{
		err: orquestaautoprogramming.MaterialProgressStateNotFoundErrorV0{
			RunRef: "run-ref-fallback", GoalRef: "goal-ref-fallback",
		},
	}, "run-ref-fallback", "goal-ref-fallback")
	if !notFound.LegacyFallback || notFound.DiagnosticCode != "material_progress_state_not_found" {
		t.Fatalf("not found=%+v", notFound)
	}
	unavailable := mcpMaterialProgressStateForRunGoalV0(context.Background(), &materialProgressReaderForMCPTestV0{
		err: errors.New("material progress backend unavailable"),
	}, "run-ref-fallback", "goal-ref-fallback")
	if unavailable.LegacyFallback || unavailable.DiagnosticCode != "material_progress_state_unavailable" {
		t.Fatalf("unavailable=%+v", unavailable)
	}
	actions := []MCPAutoprogrammingActionableRunV0{
		{RunRef: "run-ref-not-found", Code: mcpAutoprogrammingActionCheckpointOnlyHighConsumptionV0},
		{RunRef: "run-ref-unavailable", Code: mcpAutoprogrammingActionCheckpointOnlyHighConsumptionV0},
	}
	filtered := filterMCPAutoprogrammingLegacyMaterialProgressActionsV0(actions, mcpMaterialProgressStatusProjectionV0{
		"run-ref-not-found":   notFound,
		"run-ref-unavailable": unavailable,
	}.legacySuppressedRuns())
	if len(filtered) != 1 || filtered[0].RunRef != "run-ref-not-found" {
		t.Fatalf("legacy fallback must survive only not found: %+v", filtered)
	}
}

func TestMCPTransportBindingsPropaganMaterialProgressReaderV0(t *testing.T) {
	reader := &materialProgressReaderForMCPTestV0{}
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	status, ok := autoprogrammingStatusExecutorFromBindingsV0(MCPTransportBindingsV0{
		AutoprogrammingGoalStates:                  goalStates,
		AutoprogrammingMaterialProgressStateReader: reader,
	}).(MCPAutoprogrammingStatusToolExecutorV0)
	if !ok || status.MaterialProgressStateReader != reader {
		t.Fatalf("status binding=%T %+v", status, status)
	}
	control, ok := runControlExecutorFromBindingsV0(MCPTransportBindingsV0{
		RunControl: MCPRunControlToolExecutorV0{},
		AutoprogrammingMaterialProgressStateReader: reader,
	}).(MCPRunControlToolExecutorV0)
	if !ok || control.MaterialProgressStateReader != reader {
		t.Fatalf("run control binding=%T %+v", control, control)
	}
}

type materialProgressReaderForMCPTestV0 struct {
	states map[string]orquestaautoprogramming.MaterialProgressStateV0
	err    error
}

func (reader *materialProgressReaderForMCPTestV0) LoadMaterialProgressStateV0(
	_ context.Context,
	runRef string,
	goalRef string,
) (orquestaautoprogramming.MaterialProgressStateV0, error) {
	if reader.err != nil {
		return orquestaautoprogramming.MaterialProgressStateV0{}, reader.err
	}
	state, ok := reader.states[materialProgressReaderKeyForMCPTestV0(runRef, goalRef)]
	if !ok {
		return orquestaautoprogramming.MaterialProgressStateV0{}, orquestaautoprogramming.MaterialProgressStateNotFoundErrorV0{
			RunRef: runRef, GoalRef: goalRef,
		}
	}
	return state, nil
}

func materialProgressStateForMCPTestV0(
	t *testing.T,
	runRef string,
	goalRef string,
	action orquestaautoprogramming.MaterialProgressActionV0,
) orquestaautoprogramming.MaterialProgressStateV0 {
	t.Helper()
	tokens := map[orquestaautoprogramming.MaterialProgressActionV0]int64{
		orquestaautoprogramming.MaterialProgressActionContinueV0:         5,
		orquestaautoprogramming.MaterialProgressActionWarningV0:          15,
		orquestaautoprogramming.MaterialProgressActionReplanRequiredV0:   25,
		orquestaautoprogramming.MaterialProgressActionHardStopRequiredV0: 35,
	}[action]
	policy := orquestaautoprogramming.MaterialProgressPolicyV0{
		WarningAfterTokens: 10, ReplanRequiredAfterTokens: 20, HardStopRequiredAfterTokens: 30, MaxReplans: 2,
	}
	segment := orquestaautoprogramming.MaterialProgressSegmentV0{
		StartSequence: 1, StartTokensAccumulated: 100, ContextRevisionRef: "context-ref-material-progress",
	}
	checkpoint := orquestaautoprogramming.MaterialProgressCheckpointV0{
		Sequence: 2, TokensAccumulated: 100 + tokens, ContextRevisionRef: segment.ContextRevisionRef,
		MaterialClass: orquestaautoprogramming.MaterialProgressClassNoneV0,
	}
	decision := orquestaautoprogramming.DecideMaterialProgressV0(orquestaautoprogramming.MaterialProgressInputV0{
		Policy: policy, Segment: segment, Checkpoint: checkpoint,
	})
	if !decision.Accepted || decision.Action != action {
		t.Fatalf("decision=%+v want=%s", decision, action)
	}
	state := orquestaautoprogramming.MaterialProgressStateV0{
		SchemaVersion: orquestaautoprogramming.MaterialProgressStateSchemaVersionV0,
		StoreVersion:  1, RunRef: runRef, GoalRef: goalRef, Policy: policy, Segment: segment,
		LastCheckpoint: checkpoint, LastDecision: decision, BaselineRef: "baseline-ref-material-progress",
		WriteSetSHA256: strings.Repeat("a", 64), ContextRevisionRef: segment.ContextRevisionRef,
		ObservedAt: time.Date(2026, 7, 11, 10, 0, 0, 0, time.UTC),
	}
	state.LastCheckpointRef = orquestaautoprogramming.MaterialProgressCheckpointRefV0(
		state.RunRef, state.GoalRef, state.BaselineRef, state.WriteSetSHA256, state.LastCheckpoint,
	)
	state.LastActionIdempotencyKey = orquestaautoprogramming.MaterialProgressActionIdempotencyKeyV0(
		state.RunRef, state.GoalRef, state.LastCheckpointRef, state.LastDecision.Action,
	)
	validation := orquestaautoprogramming.ValidateMaterialProgressStateV0(state)
	if !validation.Accepted {
		t.Fatalf("invalid material progress fixture: %+v", validation.Issues)
	}
	return validation.State
}

func materialProgressGoalStateStoreForMCPTestV0(
	t *testing.T,
	runRef string,
	goalRef string,
) *mcpGoalStateStoreForTestV0 {
	t.Helper()
	store := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	err := store.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef: runRef, GoalRef: goalRef, ExternalGoalRef: "thread-ref-" + goalRef,
		Status: orquestagoal.GoalStatusRunningV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef: runRef, GoalRef: goalRef, Objective: "project persisted material progress",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "modulos/orquesta-mcp"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef: goalRef, ExternalGoalRef: "thread-ref-" + goalRef, Status: orquestagoal.GoalStatusRunningV0,
		},
		EvidenceRefs: []string{"evidence-ref-material-progress-goal"},
	})
	if err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	return store
}

func materialProgressReaderKeyForMCPTestV0(runRef string, goalRef string) string {
	return strings.TrimSpace(runRef) + "\x00" + strings.TrimSpace(goalRef)
}

func materialProgressActionForMCPTestV0(
	actions []MCPAutoprogrammingActionableRunV0,
	code string,
) (MCPAutoprogrammingActionableRunV0, bool) {
	for _, action := range actions {
		if action.Code == code {
			return action, true
		}
	}
	return MCPAutoprogrammingActionableRunV0{}, false
}

func materialProgressGoalIssueForMCPTestV0(issues []orquestagoal.GoalWorkIssueV0, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
