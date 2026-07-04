package orquestamcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
)

func TestMCPObserveAppDirectorGoalHTTPHandlerV0DelegaEnExecutor(t *testing.T) {
	executor := &fakeMCPObserveAppDirectorGoalHTTPExecutorV0{
		result: MCPObserveAppDirectorGoalToolResultV0{
			Estado:                MCPObserveAppDirectorGoalEstadoOKV0,
			RunRef:                "run-ref-goal-http-001",
			RunStatus:             "closed",
			DirectorExecutionMode: orquestaappdirectorservice.AppDirectorExecutionModeGoalFirstV0,
			GoalRef:               "goal-ref-http-001",
			GoalStatus:            "complete",
			EvidenceRefs:          []string{"evidence-ref-goal-http-001"},
		},
	}
	body := &bytes.Buffer{}
	if err := json.NewEncoder(body).Encode(MCPObserveAppDirectorGoalToolInputV0{
		RequestID: "req-goal-http-001",
		RunRef:    "run-ref-goal-http-001",
	}); err != nil {
		t.Fatalf("encode input: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, MCPObserveAppDirectorGoalHTTPPathV0, body)
	req.Header.Set("X-Correlation-ID", "corr-goal-http-001")
	rec := httptest.NewRecorder()

	NewMCPObserveAppDirectorGoalHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-Correlation-ID") != "corr-goal-http-001" {
		t.Fatalf("correlation=%q", rec.Header().Get("X-Correlation-ID"))
	}
	if executor.input.RunRef != "run-ref-goal-http-001" {
		t.Fatalf("input=%+v", executor.input)
	}
	var result MCPObserveAppDirectorGoalToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.GoalRef != "goal-ref-http-001" ||
		result.RunStatus != "closed" ||
		result.DirectorExecutionMode != orquestaappdirectorservice.AppDirectorExecutionModeGoalFirstV0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPObserveAppDirectorGoalHTTPHandlerV0TimeoutDevuelveJSONPublico(t *testing.T) {
	executor := &blockingMCPObserveAppDirectorGoalHTTPExecutorV0{done: make(chan struct{})}
	body := &bytes.Buffer{}
	if err := json.NewEncoder(body).Encode(MCPObserveAppDirectorGoalToolInputV0{
		RequestID:     "req-goal-http-timeout-001",
		CorrelationID: "corr-goal-http-timeout-input-001",
		RunRef:        "run-ref-goal-http-timeout-001",
	}); err != nil {
		t.Fatalf("encode input: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, MCPObserveAppDirectorGoalHTTPPathV0, body)
	req.Header.Set("X-Correlation-ID", "corr-goal-http-timeout-header-001")
	rec := httptest.NewRecorder()

	newMCPObserveAppDirectorGoalHTTPHandlerWithTimeoutV0(executor, time.Millisecond).ServeHTTP(rec, req)

	if rec.Code != http.StatusGatewayTimeout ||
		rec.Header().Get("X-Correlation-ID") != "corr-goal-http-timeout-header-001" {
		t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
	}
	var result MCPObserveAppDirectorGoalToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Estado != MCPObserveAppDirectorGoalEstadoErrorV0 ||
		!result.Partial ||
		result.RunRef != "run-ref-goal-http-timeout-001" ||
		result.RecommendedAction != "observe_later" ||
		result.Summary == "" ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != MCPObserveAppDirectorGoalHTTPTimeoutCodeV0 ||
		result.Errores[0].Field != "executor" ||
		!stringInSliceForMCPObserveGoalHTTPTestV0(result.EvidenceRefs, "evidence-ref-observe-app-director-goal-timeout") {
		t.Fatalf("result=%+v", result)
	}
	select {
	case <-executor.done:
	case <-time.After(time.Second):
		t.Fatalf("executor no recibio cancelacion tras timeout HTTP")
	}
}

func TestMCPObserveAppDirectorGoalHTTPHandlerV0TimeoutIncluyeSnapshotParcialV0(t *testing.T) {
	executor := &blockingSnapshotMCPObserveAppDirectorGoalHTTPExecutorV0{
		done: make(chan struct{}),
		snapshot: MCPObserveAppDirectorGoalToolResultV0{
			Estado:                MCPObserveAppDirectorGoalEstadoOKV0,
			Partial:               true,
			RunRef:                "run-ref-goal-http-timeout-snapshot-001",
			DirectorExecutionMode: orquestaappdirectorservice.AppDirectorExecutionModeGoalFirstV0,
			GoalRef:               "goal-ref-http-timeout-snapshot-001",
			ExternalGoalRef:       "thread-ref-http-timeout-snapshot-001",
			GoalStatus:            orquestagoal.GoalStatusRunningV0,
			RecommendedAction:     "observe_later",
			EvidenceRefs:          []string{"evidence-ref-http-timeout-snapshot-001"},
		},
	}
	body := &bytes.Buffer{}
	if err := json.NewEncoder(body).Encode(MCPObserveAppDirectorGoalToolInputV0{
		RequestID: "req-goal-http-timeout-snapshot-001",
		RunRef:    "run-ref-goal-http-timeout-snapshot-001",
	}); err != nil {
		t.Fatalf("encode input: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, MCPObserveAppDirectorGoalHTTPPathV0, body)
	req.Header.Set("X-Correlation-ID", "corr-goal-http-timeout-snapshot-001")
	rec := httptest.NewRecorder()

	newMCPObserveAppDirectorGoalHTTPHandlerWithTimeoutV0(executor, time.Millisecond).ServeHTTP(rec, req)

	if rec.Code != http.StatusGatewayTimeout {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPObserveAppDirectorGoalToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Estado != MCPObserveAppDirectorGoalEstadoErrorV0 ||
		!result.Partial ||
		result.GoalRef != "goal-ref-http-timeout-snapshot-001" ||
		result.ExternalGoalRef != "thread-ref-http-timeout-snapshot-001" ||
		result.GoalStatus != orquestagoal.GoalStatusRunningV0 ||
		result.RecommendedAction != "observe_later" ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != MCPObserveAppDirectorGoalHTTPTimeoutCodeV0 ||
		!stringInSliceForMCPObserveGoalHTTPTestV0(result.EvidenceRefs, "evidence-ref-http-timeout-snapshot-001") {
		t.Fatalf("result=%+v", result)
	}
	select {
	case <-executor.done:
	case <-time.After(time.Second):
		t.Fatalf("executor no recibio cancelacion tras timeout HTTP")
	}
}

func TestMCPObserveAppDirectorGoalHTTPHandlerV0RunRefRequerido(t *testing.T) {
	executor := &fakeMCPObserveAppDirectorGoalHTTPExecutorV0{}
	req := httptest.NewRequest(http.MethodPost, MCPObserveAppDirectorGoalHTTPPathV0, strings.NewReader(`{}`))
	rec := httptest.NewRecorder()

	NewMCPObserveAppDirectorGoalHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if executor.called != 0 {
		t.Fatalf("executor llamado sin run_ref")
	}
	var result MCPObserveAppDirectorGoalToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPObserveAppDirectorGoalEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Field != "run_ref" ||
		result.Errores[0].Message != "run_ref_requerido" {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPObserveAppDirectorGoalHTTPHandlerV0ErrorOperativoNoDevuelve500(t *testing.T) {
	executor := NewMCPObserveAppDirectorGoalToolExecutorV0(orquestaappdirectorservice.StartAppDirectorPortsV0{})
	body := &bytes.Buffer{}
	if err := json.NewEncoder(body).Encode(MCPObserveAppDirectorGoalToolInputV0{
		RequestID: "req-goal-http-operational-error-001",
		RunRef:    "run-ref-goal-http-operational-error-001",
	}); err != nil {
		t.Fatalf("encode input: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, MCPObserveAppDirectorGoalHTTPPathV0, body)
	rec := httptest.NewRecorder()

	NewMCPObserveAppDirectorGoalHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPObserveAppDirectorGoalToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPObserveAppDirectorGoalEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "observe_app_director_goal_state_store_unbound" ||
		result.Errores[0].Field != "goal_state_store" ||
		result.Errores[0].Message != "goal_state_store_not_configured" {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPObserveAppDirectorGoalErrorResultFromErrorV0PreservaIssueCodeV0(t *testing.T) {
	result, ok := NewMCPObserveAppDirectorGoalErrorResultFromErrorV0(
		MCPObserveAppDirectorGoalToolInputV0{RunRef: "run-ref-goal-issue-001"},
		orquestagoal.GoalWorkLifecycleIssueErrorV0{
			Field: "goal_result",
			Issues: []orquestagoal.GoalWorkIssueV0{{
				Code:  "codex_app_server_control_socket_missing",
				Field: "codex_goal_backend",
			}},
		},
	)
	if !ok ||
		result.Estado != MCPObserveAppDirectorGoalEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "codex_app_server_control_socket_missing" ||
		result.Errores[0].Field != "codex_goal_backend" ||
		result.Errores[0].Message != "codex_app_server_control_socket_missing" {
		t.Fatalf("ok=%v result=%+v", ok, result)
	}
}

func TestMCPObserveAppDirectorGoalErrorResultFromErrorV0PreservaQuotaFilesystemV0(t *testing.T) {
	result, ok := NewMCPObserveAppDirectorGoalErrorResultFromErrorV0(
		MCPObserveAppDirectorGoalToolInputV0{RunRef: "run-ref-goal-issue-001"},
		orquestagoal.GoalWorkLifecycleIssueErrorV0{
			Field: "goal_result",
			Issues: []orquestagoal.GoalWorkIssueV0{{
				Code:  "codex_app_server_storage_quota_exceeded",
				Field: "codex_goal_backend",
			}},
		},
	)
	if !ok ||
		result.Estado != MCPObserveAppDirectorGoalEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "codex_app_server_storage_quota_exceeded" ||
		result.Errores[0].Field != "codex_goal_backend" ||
		result.Errores[0].Message != "codex_app_server_storage_quota_exceeded" {
		t.Fatalf("ok=%v result=%+v", ok, result)
	}
}

func TestMCPObserveAppDirectorGoalToolExecutorV0RunRefRequerido(t *testing.T) {
	result, err := NewMCPObserveAppDirectorGoalToolExecutorV0(orquestaappdirectorservice.StartAppDirectorPortsV0{}).Execute(
		context.Background(),
		MCPObserveAppDirectorGoalToolInputV0{},
	)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if result.Estado != MCPObserveAppDirectorGoalEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Field != "run_ref" {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPObserveAppDirectorGoalToolExecutorV0ErrorPublicoIncluyeSnapshotParcialV0(t *testing.T) {
	state, err := orquestagoal.NewGoalWorkStateFromLaunchV0(orquestagoal.GoalWorkStateFromLaunchRequestV0{
		RunRef: "run-ref-goal-observe-rejected-snapshot-001",
		Spec: orquestagoal.GoalWorkSpecV0{
			SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
			GoalRef:       "goal-ref-observe-rejected-snapshot-001",
			RunRef:        "run-ref-goal-observe-rejected-snapshot-001",
			Objective:     "Conservar snapshot local si el backend rechaza observar el goal.",
			DirectorKind:  orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:      []orquestagoal.GoalWriteScopeV0{{Path: "docs/goal_result.md"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			Status:          orquestagoal.GoalStatusRunningV0,
			GoalRef:         "goal-ref-observe-rejected-snapshot-001",
			ExternalGoalRef: "thread-ref-observe-rejected-snapshot-001",
			EvidenceRefs:    []string{"evidence-ref-observe-rejected-launch-001"},
		},
	})
	if err != nil {
		t.Fatalf("NewGoalWorkStateFromLaunchV0: %v", err)
	}
	store := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := store.SaveGoalWorkStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}

	result, err := NewMCPObserveAppDirectorGoalToolExecutorV0(orquestaappdirectorservice.StartAppDirectorPortsV0{
		GoalStateStore: store,
		GoalObserver: mcpObserveGoalRejectedObserverForTestV0{result: orquestagoal.GoalWorkResultV0{
			SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
			Status:          orquestagoal.GoalStatusInvalidV0,
			GoalRef:         state.GoalRef,
			ExternalGoalRef: state.ExternalGoalRef,
			Issues: []orquestagoal.GoalWorkIssueV0{{
				Code:  "codex_goal_observation_rejected",
				Field: "codex_goal_backend",
			}},
		}},
	}).Execute(context.Background(), MCPObserveAppDirectorGoalToolInputV0{
		RequestID: "request-ref-observe-rejected-snapshot-001",
		RunRef:    state.RunRef,
	})

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPObserveAppDirectorGoalEstadoErrorV0 ||
		!result.Partial ||
		result.GoalRef != state.GoalRef ||
		result.ExternalGoalRef != state.ExternalGoalRef ||
		result.GoalStatus != orquestagoal.GoalStatusRunningV0 ||
		result.RecommendedAction != "observe_later" ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "codex_goal_observation_rejected" ||
		result.Errores[0].Field != "codex_goal_backend" ||
		!stringInSliceForMCPObserveGoalHTTPTestV0(result.EvidenceRefs, "evidence-ref-observe-rejected-launch-001") {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPObserveAppDirectorGoalToolExecutorV0TimeoutSnapshotLeeGoalStateV0(t *testing.T) {
	state, err := orquestagoal.NewGoalWorkStateFromLaunchV0(orquestagoal.GoalWorkStateFromLaunchRequestV0{
		RunRef: "run-ref-goal-snapshot-store-001",
		Spec: orquestagoal.GoalWorkSpecV0{
			SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
			GoalRef:       "goal-ref-snapshot-store-001",
			RunRef:        "run-ref-goal-snapshot-store-001",
			Objective:     "Observar estado parcial del goal sin relanzar el ejecutor.",
			DirectorKind:  orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet: []orquestagoal.GoalWriteScopeV0{{
				Path: "docs/resultado_snapshot.md",
			}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			Status:          orquestagoal.GoalStatusRunningV0,
			GoalRef:         "goal-ref-snapshot-store-001",
			ExternalGoalRef: "thread-ref-snapshot-store-001",
			EvidenceRefs:    []string{"evidence-ref-snapshot-launch-001"},
		},
	})
	if err != nil {
		t.Fatalf("NewGoalWorkStateFromLaunchV0: %v", err)
	}
	state.LastResult = &orquestagoal.GoalWorkResultV0{
		SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
		Status:          orquestagoal.GoalStatusRunningV0,
		GoalRef:         "goal-ref-snapshot-store-001",
		ExternalGoalRef: "thread-ref-snapshot-store-001",
		Summary:         "Goal en ejecucion con resultado parcial persistido.",
		ArtifactRefs:    []string{"artifact-ref-snapshot-001"},
		EvidenceRefs:    []string{"evidence-ref-goal-result-snapshot-001"},
	}
	store := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := store.SaveGoalWorkStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	result, err := NewMCPObserveAppDirectorGoalToolExecutorV0(orquestaappdirectorservice.StartAppDirectorPortsV0{
		GoalStateStore: store,
		EventReader: mcpObserveGoalEventReaderForTestV0{events: []orquestacoreworkflow.OrchestrationEventV0{{
			EventID:    "event-ref-goal-snapshot-store-001",
			RunID:      "run-ref-goal-snapshot-store-001",
			Sequence:   1,
			OccurredAt: "2026-06-30T18:00:00Z",
		}, {
			EventID:    "event-ref-goal-snapshot-store-002",
			RunID:      "run-ref-goal-snapshot-store-001",
			Sequence:   2,
			OccurredAt: "2026-06-30T18:05:00Z",
		}}},
		RunStore: mcpObserveGoalRunStoreForTestV0{run: orquestacoreworkflow.OrchestrationRunV0{
			RunID:        "run-ref-goal-snapshot-store-001",
			Status:       orquestacoreworkflow.OrchestrationRunStatusActiveV0,
			CurrentPhase: orquestacoreworkflow.OrchestrationPhaseRevisionV0,
		}},
	}).ObserveAppDirectorGoalTimeoutSnapshotV0(
		context.Background(),
		MCPObserveAppDirectorGoalToolInputV0{
			RequestID: "req-goal-snapshot-store-001",
			RunRef:    "run-ref-goal-snapshot-store-001",
		},
	)
	if err != nil {
		t.Fatalf("ObserveAppDirectorGoalTimeoutSnapshotV0: %v", err)
	}
	if !result.Partial ||
		result.Estado != MCPObserveAppDirectorGoalEstadoOKV0 ||
		result.DirectorExecutionMode != orquestaappdirectorservice.AppDirectorExecutionModeGoalFirstV0 ||
		result.GoalRef != "goal-ref-snapshot-store-001" ||
		result.ExternalGoalRef != "thread-ref-snapshot-store-001" ||
		result.GoalStatus != orquestagoal.GoalStatusRunningV0 ||
		result.RunStatus != string(orquestacoreworkflow.OrchestrationRunStatusActiveV0) ||
		result.CurrentPhase != string(orquestacoreworkflow.OrchestrationPhaseRevisionV0) ||
		result.LastEventAt != "2026-06-30T18:05:00Z" ||
		result.ResultRef != "evidence-ref-goal-result-snapshot-001" ||
		result.RecommendedAction != "observe_later" ||
		!stringInSliceForMCPObserveGoalHTTPTestV0(result.ArtifactRefs, "artifact-ref-snapshot-001") ||
		!stringInSliceForMCPObserveGoalHTTPTestV0(result.EvidenceRefs, "evidence-ref-snapshot-launch-001") {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPObserveAppDirectorGoalToolExecutorV0TimeoutSnapshotNoPublicaRunningSiRunControlTerminalV0(t *testing.T) {
	runRef := "run-ref-goal-snapshot-run-control-terminal-001"
	state, err := orquestagoal.NewGoalWorkStateFromLaunchV0(orquestagoal.GoalWorkStateFromLaunchRequestV0{
		RunRef: runRef,
		Spec: orquestagoal.GoalWorkSpecV0{
			SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
			GoalRef:       "goal-ref-snapshot-run-control-terminal-001",
			RunRef:        runRef,
			Objective:     "No publicar running si RunControl ya esta terminal.",
			DirectorKind:  orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet: []orquestagoal.GoalWriteScopeV0{{
				Path: "docs/resultado_snapshot_terminal.md",
			}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			Status:          orquestagoal.GoalStatusRunningV0,
			GoalRef:         "goal-ref-snapshot-run-control-terminal-001",
			ExternalGoalRef: "thread-ref-snapshot-run-control-terminal-001",
			EvidenceRefs:    []string{"evidence-ref-snapshot-run-control-terminal-launch"},
		},
	})
	if err != nil {
		t.Fatalf("NewGoalWorkStateFromLaunchV0: %v", err)
	}
	store := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := store.SaveGoalWorkStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}

	result, err := NewMCPObserveAppDirectorGoalToolExecutorV0(orquestaappdirectorservice.StartAppDirectorPortsV0{
		GoalStateStore: store,
		RunControl: mcpObserveGoalRunControlForTestV0{state: orquestaruncontrol.RunControlStateV0{
			RunRef:       runRef,
			Status:       orquestaruncontrol.RunControlStatusStoppedV0,
			Forced:       true,
			EvidenceRefs: []string{"evidence-ref-run-control-stopped-forced"},
		}},
	}).ObserveAppDirectorGoalTimeoutSnapshotV0(
		context.Background(),
		MCPObserveAppDirectorGoalToolInputV0{
			RequestID: "req-goal-snapshot-run-control-terminal-001",
			RunRef:    runRef,
		},
	)
	if err != nil {
		t.Fatalf("ObserveAppDirectorGoalTimeoutSnapshotV0: %v", err)
	}
	if result.GoalStatus == orquestagoal.GoalStatusRunningV0 ||
		result.GoalStatus != orquestagoal.GoalStatusBlockedV0 ||
		result.RecommendedAction != "replan" ||
		!stringInSliceForMCPObserveGoalHTTPTestV0(result.EvidenceRefs, "evidence-ref-run-control-stopped-forced") ||
		!stringInSliceForMCPObserveGoalHTTPTestV0(result.EvidenceRefs, "evidence-ref-observe-goal-run-control-terminal") {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPObserveAppDirectorGoalToolExecutorV0TimeoutSnapshotProyectaMetadataOPESAudioV0(t *testing.T) {
	state, err := orquestagoal.NewGoalWorkStateFromLaunchV0(orquestagoal.GoalWorkStateFromLaunchRequestV0{
		RunRef: "run-ref-goal-snapshot-opes-audio-001",
		Spec: orquestagoal.GoalWorkSpecV0{
			SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
			GoalRef:       "goal-ref-snapshot-opes-audio-001",
			RunRef:        "run-ref-goal-snapshot-opes-audio-001",
			DomainRef:     "opes",
			WorkKind:      "generate_audio_asset",
			Objective:     "Observar metadata OPES audio durable.",
			DirectorKind:  orquestagoal.GoalDirectorKindCodexGoalV0,
			ContextRefs: []orquestagoal.GoalContextRefV0{{
				Kind:    "input_field_value",
				Ref:     "input-field-audio-current-phase-value-test",
				Purpose: `Campo input_fields.audio_current_phase inlineado de forma acotada: {"name":"audio_current_phase","value":"tts"}`,
			}, {
				Kind:    "input_field_value",
				Ref:     "input-field-provider-timeout-value-test",
				Purpose: `Campo input_fields.provider_timeout inlineado de forma acotada: {"name":"provider_timeout","value":"true"}`,
			}, {
				Kind:    "input_field_value",
				Ref:     "input-field-audio-counters-value-test",
				Purpose: `Campo input_fields.audio_counters inlineado de forma acotada: {"name":"audio_counters","value_json":{"segments_pending":7,"segments_done":3}}`,
			}},
			WriteSet: []orquestagoal.GoalWriteScopeV0{{Path: "external/opes/audio"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			Status:  orquestagoal.GoalStatusBlockedV0,
			GoalRef: "goal-ref-snapshot-opes-audio-001",
		},
	})
	if err != nil {
		t.Fatalf("NewGoalWorkStateFromLaunchV0: %v", err)
	}
	store := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := store.SaveGoalWorkStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}

	result, err := NewMCPObserveAppDirectorGoalToolExecutorV0(orquestaappdirectorservice.StartAppDirectorPortsV0{
		GoalStateStore: store,
	}).ObserveAppDirectorGoalTimeoutSnapshotV0(
		context.Background(),
		MCPObserveAppDirectorGoalToolInputV0{
			RequestID: "req-goal-snapshot-opes-audio-001",
			RunRef:    "run-ref-goal-snapshot-opes-audio-001",
		},
	)
	if err != nil {
		t.Fatalf("ObserveAppDirectorGoalTimeoutSnapshotV0: %v", err)
	}
	if !result.Partial ||
		result.CurrentPhase != "tts" ||
		result.RetryFromPhase != "tts" ||
		result.DomainCounters["segments_pending"] != 7 ||
		result.DomainCounters["segments_done"] != 3 ||
		result.RecommendedAction != "blocked" {
		t.Fatalf("result=%+v", result)
	}
}

type mcpObserveGoalEventReaderForTestV0 struct {
	events []orquestacoreworkflow.OrchestrationEventV0
}

func (reader mcpObserveGoalEventReaderForTestV0) LoadRunEventsV0(
	_ context.Context,
	_ string,
) ([]orquestacoreworkflow.OrchestrationEventV0, error) {
	return reader.events, nil
}

type mcpObserveGoalRunStoreForTestV0 struct {
	run orquestacoreworkflow.OrchestrationRunV0
}

func (store mcpObserveGoalRunStoreForTestV0) LoadRunV0(
	_ context.Context,
	_ string,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	return store.run, nil
}

func (store mcpObserveGoalRunStoreForTestV0) SaveRunV0(
	_ context.Context,
	_ orquestacoreworkflow.OrchestrationRunV0,
) error {
	return nil
}

type mcpObserveGoalRunControlForTestV0 struct {
	state orquestaruncontrol.RunControlStateV0
}

func (control mcpObserveGoalRunControlForTestV0) ReadRunControlStateV0(
	_ context.Context,
	_ orquestaruncontrol.RunControlReadRequestV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	return control.state, nil
}

type fakeMCPObserveAppDirectorGoalHTTPExecutorV0 struct {
	input  MCPObserveAppDirectorGoalToolInputV0
	called int
	result MCPObserveAppDirectorGoalToolResultV0
}

func (executor *fakeMCPObserveAppDirectorGoalHTTPExecutorV0) Execute(
	_ context.Context,
	input MCPObserveAppDirectorGoalToolInputV0,
) (MCPObserveAppDirectorGoalToolResultV0, error) {
	executor.called++
	executor.input = input
	if executor.result.Estado == "" {
		executor.result = MCPObserveAppDirectorGoalToolResultV0{
			Estado: MCPObserveAppDirectorGoalEstadoOKV0,
			RunRef: input.RunRef,
		}
	}
	return executor.result, nil
}

type mcpObserveGoalRejectedObserverForTestV0 struct {
	result orquestagoal.GoalWorkResultV0
}

func (observer mcpObserveGoalRejectedObserverForTestV0) ObserveGoalWorkV0(
	_ context.Context,
	_ orquestagoal.GoalObservationRequestV0,
) (orquestagoal.GoalWorkResultV0, error) {
	return observer.result, errors.New("codex_goal_observation_rejected")
}

type blockingMCPObserveAppDirectorGoalHTTPExecutorV0 struct {
	done chan struct{}
}

func (executor *blockingMCPObserveAppDirectorGoalHTTPExecutorV0) Execute(
	ctx context.Context,
	input MCPObserveAppDirectorGoalToolInputV0,
) (MCPObserveAppDirectorGoalToolResultV0, error) {
	<-ctx.Done()
	close(executor.done)
	return MCPObserveAppDirectorGoalToolResultV0{
		Estado: MCPObserveAppDirectorGoalEstadoOKV0,
		RunRef: input.RunRef,
	}, nil
}

type blockingSnapshotMCPObserveAppDirectorGoalHTTPExecutorV0 struct {
	done     chan struct{}
	snapshot MCPObserveAppDirectorGoalToolResultV0
}

func (executor *blockingSnapshotMCPObserveAppDirectorGoalHTTPExecutorV0) Execute(
	ctx context.Context,
	input MCPObserveAppDirectorGoalToolInputV0,
) (MCPObserveAppDirectorGoalToolResultV0, error) {
	<-ctx.Done()
	close(executor.done)
	return MCPObserveAppDirectorGoalToolResultV0{
		Estado: MCPObserveAppDirectorGoalEstadoOKV0,
		RunRef: input.RunRef,
	}, nil
}

func (executor *blockingSnapshotMCPObserveAppDirectorGoalHTTPExecutorV0) ObserveAppDirectorGoalTimeoutSnapshotV0(
	_ context.Context,
	_ MCPObserveAppDirectorGoalToolInputV0,
) (MCPObserveAppDirectorGoalToolResultV0, error) {
	return executor.snapshot, nil
}

func stringInSliceForMCPObserveGoalHTTPTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
