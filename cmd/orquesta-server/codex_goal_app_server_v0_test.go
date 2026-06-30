package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestServerSupervisorWithCodexGoalBackendV0ExponePuertosSoloConBackendV0(t *testing.T) {
	base := serverStackSupervisorV0{}
	plain := serverSupervisorWithCodexGoalBackendV0(base, serverCodexGoalBackendV0{})
	if _, ok := plain.(orquestaserver.IdleSelfImprovementGoalLauncherPortV0); ok {
		t.Fatalf("supervisor sin backend no debe exponer launcher goal")
	}
	if _, ok := plain.(orquestaserver.IdleSelfImprovementGoalObserverPortV0); ok {
		t.Fatalf("supervisor sin backend no debe exponer observer goal")
	}

	protocol := &fakeCodexAppServerProtocolV0{}
	backend := serverCodexAppServerGoalBackendV0{Protocol: protocol}
	wrapped := serverSupervisorWithCodexGoalBackendV0(base, serverCodexGoalBackendV0{
		Starter:  backend,
		Observer: backend,
	})
	if _, ok := wrapped.(orquestaserver.IdleSelfImprovementGoalLauncherPortV0); !ok {
		t.Fatalf("supervisor con backend debe exponer launcher goal")
	}
	if _, ok := wrapped.(orquestaserver.IdleSelfImprovementGoalObserverPortV0); !ok {
		t.Fatalf("supervisor con backend debe exponer observer goal")
	}
}

func TestServerGoalObservationFingerprintFromBackendV0EsOptInV0(t *testing.T) {
	protocol := &fakeCodexAppServerProtocolV0{}
	backend := serverCodexAppServerGoalBackendV0{Protocol: protocol}
	if serverGoalObservationFingerprintFromBackendV0(serverCodexGoalBackendV0{
		Observer: backend,
	}, false) != nil {
		t.Fatalf("fingerprint no debe exponerse sin opt-in")
	}
	fingerprint := serverGoalObservationFingerprintFromBackendV0(serverCodexGoalBackendV0{
		Observer: backend,
	}, true)
	if fingerprint == nil {
		t.Fatalf("backend Codex app-server debe exponer fingerprint con opt-in")
	}
}

func TestCodexAppServerGoalBackendFingerprintDetectaCambioDeEstadoV0(t *testing.T) {
	protocol := &fakeCodexAppServerProtocolV0{
		observedGoal: &serverCodexAppServerThreadGoalV0{
			ThreadID:        "thread-ref-fingerprint-001",
			Status:          "active",
			TokensUsed:      11,
			TimeUsedSeconds: 7,
		},
	}
	backend := serverCodexAppServerGoalBackendV0{Protocol: protocol}
	state := orquestagoal.GoalWorkStateV0{
		SchemaVersion:   orquestagoal.GoalWorkStateSchemaV0,
		RunRef:          "run-ref-fingerprint-001",
		GoalRef:         "goal-ref-fingerprint-001",
		ExternalGoalRef: "thread-ref-fingerprint-001",
		Status:          orquestagoal.GoalStatusRunningV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
			GoalRef:       "goal-ref-fingerprint-001",
			Objective:     "probar fingerprint",
			DirectorKind:  orquestagoal.GoalDirectorKindCodexGoalV0,
			WorkKind:      "app_change",
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			SchemaVersion:   orquestagoal.GoalWorkLaunchReceiptSchemaV0,
			Status:          orquestagoal.GoalStatusRunningV0,
			GoalRef:         "goal-ref-fingerprint-001",
			ExternalGoalRef: "thread-ref-fingerprint-001",
		},
	}
	first, ok, err := backend.FingerprintGoalObservationV0(context.Background(), state)
	if err != nil || !ok {
		t.Fatalf("FingerprintGoalObservationV0 first ok=%v err=%v", ok, err)
	}
	if first.LastStatus != orquestagoal.GoalStatusRunningV0 || first.ProcessAlive {
		t.Fatalf("first fingerprint=%+v", first)
	}
	second, ok, err := backend.FingerprintGoalObservationV0(context.Background(), state)
	if err != nil || !ok {
		t.Fatalf("FingerprintGoalObservationV0 second ok=%v err=%v", ok, err)
	}
	if !orquestagoal.GoalObservationUnchangedV0(first, second) {
		t.Fatalf("fingerprint estable debe comparar sin cambios: first=%+v second=%+v", first, second)
	}
	protocol.observedGoal.Status = "complete"
	changed, ok, err := backend.FingerprintGoalObservationV0(context.Background(), state)
	if err != nil || !ok {
		t.Fatalf("FingerprintGoalObservationV0 changed ok=%v err=%v", ok, err)
	}
	if orquestagoal.GoalObservationUnchangedV0(first, changed) {
		t.Fatalf("fingerprint debe cambiar al cerrar: first=%+v changed=%+v", first, changed)
	}
	if changed.LastStatus != orquestagoal.GoalStatusCompleteV0 || changed.ProcessAlive {
		t.Fatalf("changed fingerprint=%+v", changed)
	}
}

func TestCodexAppServerGoalBackendFingerprintLeeThreadSiGoalGetNoExisteV0(t *testing.T) {
	protocol := &fakeCodexAppServerProtocolV0{
		getGoalErr: codexAppServerCallErrorV0{Code: "codex_app_server_rpc_method_not_found"},
		readThread: serverCodexAppServerThreadReadV0{
			ID:     "thread-ref-fingerprint-read-001",
			Status: serverCodexAppServerThreadStatusV0("idle"),
		},
	}
	backend := serverCodexAppServerGoalBackendV0{Protocol: protocol}
	state := orquestagoal.GoalWorkStateV0{
		SchemaVersion:   orquestagoal.GoalWorkStateSchemaV0,
		RunRef:          "run-ref-fingerprint-read-001",
		GoalRef:         "goal-ref-fingerprint-read-001",
		ExternalGoalRef: "thread-ref-fingerprint-read-001",
		Status:          orquestagoal.GoalStatusRunningV0,
	}

	fingerprint, ok, err := backend.FingerprintGoalObservationV0(context.Background(), state)

	if err != nil || !ok {
		t.Fatalf("FingerprintGoalObservationV0 ok=%v err=%v", ok, err)
	}
	if fingerprint.LastStatus != orquestagoal.GoalStatusRunningV0 ||
		fingerprint.EvidenceHash == "" ||
		!reflect.DeepEqual(protocol.calls, []string{"thread/goal/get", "thread/read"}) ||
		protocol.readIncludeTurns {
		t.Fatalf("fingerprint=%+v calls=%v include=%v", fingerprint, protocol.calls, protocol.readIncludeTurns)
	}
}

func TestCodexAppServerThreadReadStatusV0AceptaObjetoV0(t *testing.T) {
	var response serverCodexAppServerThreadReadResponseV0
	raw := []byte(`{"thread":{"id":"thread-ref-status-object","status":{"type":"systemError"}}}`)

	if err := json.Unmarshal(raw, &response); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if response.Thread.Status != serverCodexAppServerThreadStatusV0("systemError") {
		t.Fatalf("status=%q", response.Thread.Status)
	}
	if got := codexAppServerThreadStatusToGoalWorkStatusV0(response.Thread.Status); got != orquestagoal.GoalStatusBlockedV0 {
		t.Fatalf("mapped status=%q", got)
	}
}

func TestServerCodexAppServerGoalBackendV0LanzaThreadGoalYTurnV0(t *testing.T) {
	protocol := &fakeCodexAppServerProtocolV0{
		thread: serverCodexAppServerThreadV0{ID: "thread-ref-goal-001"},
		goal:   serverCodexAppServerThreadGoalV0{ThreadID: "thread-ref-goal-001", Status: "active"},
		turn:   serverCodexAppServerTurnV0{ID: "turn-ref-goal-001", Status: "inProgress"},
	}
	backend := serverCodexAppServerGoalBackendV0{
		Protocol:        protocol,
		CWD:             "/tmp/orquesta-goal-fixture",
		Model:           "gpt-5.4",
		ReasoningEffort: "medium",
		Sandbox:         "workspace-write",
		ApprovalPolicy:  "never",
	}

	receipt, err := backend.StartCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		SchemaVersion: orquestaruntimecodexgoal.CodexGoalStartPacketSchemaV0,
		GoalRef:       "goal-ref-codex-app-server-001",
		Objective:     "programar una mejora acotada",
		Prompt:        "prompt compacto",
		Budget:        orquestagoal.GoalBudgetV0{TokenBudget: 123},
	})

	if err != nil {
		t.Fatalf("StartCodexGoalV0: %v", err)
	}
	if receipt.Status != orquestagoal.GoalStatusRunningV0 ||
		receipt.GoalRef != "goal-ref-codex-app-server-001" ||
		receipt.ExternalGoalRef != "thread-ref-goal-001" {
		t.Fatalf("receipt=%+v", receipt)
	}
	wantCalls := []string{"thread/start", "thread/goal/set", "turn/start"}
	if !reflect.DeepEqual(protocol.calls, wantCalls) {
		t.Fatalf("calls=%v want=%v", protocol.calls, wantCalls)
	}
	if protocol.startParams.Ephemeral ||
		protocol.startParams.CWD != "/tmp/orquesta-goal-fixture" ||
		protocol.startParams.ApprovalPolicy != "never" {
		t.Fatalf("start params=%+v", protocol.startParams)
	}
	if protocol.setParams.ThreadID != "thread-ref-goal-001" ||
		protocol.setParams.Objective != "programar una mejora acotada" ||
		protocol.setParams.Status != "active" ||
		protocol.setParams.TokenBudget != 123 {
		t.Fatalf("set params=%+v", protocol.setParams)
	}
	if protocol.turnParams.ThreadID != "thread-ref-goal-001" ||
		protocol.turnParams.InputText != "prompt compacto" ||
		protocol.turnParams.ClientMessageID != "goal-ref-codex-app-server-001" ||
		protocol.turnParams.Effort != "medium" {
		t.Fatalf("turn params=%+v", protocol.turnParams)
	}
}

func TestServerCodexAppServerGoalBackendV0LanzaTurnSiGoalSetNoExisteV0(t *testing.T) {
	protocol := &fakeCodexAppServerProtocolV0{
		thread:     serverCodexAppServerThreadV0{ID: "thread-ref-goal-no-set-001"},
		turn:       serverCodexAppServerTurnV0{ID: "turn-ref-goal-no-set-001", Status: "inProgress"},
		setGoalErr: codexAppServerCallErrorV0{Code: "codex_app_server_rpc_invalid_request"},
	}
	backend := serverCodexAppServerGoalBackendV0{Protocol: protocol}

	receipt, err := backend.StartCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		SchemaVersion: orquestaruntimecodexgoal.CodexGoalStartPacketSchemaV0,
		GoalRef:       "goal-ref-codex-app-server-no-set-001",
		Objective:     "programar una mejora acotada",
		Prompt:        "prompt compacto",
	})

	if err != nil {
		t.Fatalf("StartCodexGoalV0 fallback: %v", err)
	}
	if receipt.Status != orquestagoal.GoalStatusRunningV0 ||
		receipt.ExternalGoalRef != "thread-ref-goal-no-set-001" ||
		!containsStringForTestV0(receipt.EvidenceRefs, "evidence-ref-codex-app-server-goal-set-unsupported") ||
		!containsStringForTestV0(receipt.EvidenceRefs, "evidence-ref-codex-app-server-turn-started") {
		t.Fatalf("receipt=%+v", receipt)
	}
	if want := []string{"thread/start", "thread/goal/set", "turn/start"}; !reflect.DeepEqual(protocol.calls, want) {
		t.Fatalf("calls=%v want=%v", protocol.calls, want)
	}
}

func TestServerCodexAppServerGoalBackendV0ObservaPorThreadReadSiGoalGetNoExisteV0(t *testing.T) {
	protocol := &fakeCodexAppServerProtocolV0{
		getGoalErr: codexAppServerCallErrorV0{Code: "codex_app_server_rpc_method_not_found"},
		readThread: serverCodexAppServerThreadReadV0{
			ID:     "thread-ref-goal-read-001",
			Status: serverCodexAppServerThreadStatusV0("idle"),
			Turns: []serverCodexAppServerReadTurnV0{{
				ID:     "turn-ref-goal-read-001",
				Status: "completed",
				Items: []serverCodexAppServerReadItemV0{{
					ID:    "item-ref-goal-read-001",
					Type:  "agentMessage",
					Phase: "final_answer",
					Text: orquestaruntimecodexgoal.CodexGoalResultMarkerV0 +
						` {"summary":"ok","artifact_refs":["artifact-ref-read"],"evidence_refs":["evidence-ref-read"]}`,
				}},
			}},
		},
	}
	backend := serverCodexAppServerGoalBackendV0{Protocol: protocol}

	receipt, err := backend.ObserveCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		GoalRef:         "goal-ref-codex-app-server-read-001",
		ExternalGoalRef: "thread-ref-goal-read-001",
	})

	if err != nil {
		t.Fatalf("ObserveCodexGoalV0 fallback: %v", err)
	}
	if receipt.Status != orquestagoal.GoalStatusCompleteV0 ||
		receipt.Summary != "ok" ||
		!containsStringForTestV0(receipt.ArtifactRefs, "artifact-ref-read") ||
		!containsStringForTestV0(receipt.EvidenceRefs, "evidence-ref-codex-app-server-goal-rpc-unsupported") ||
		!containsStringForTestV0(receipt.EvidenceRefs, "evidence-ref-codex-app-server-goal-result-marker") {
		t.Fatalf("receipt=%+v", receipt)
	}
	if want := []string{"thread/goal/get", "thread/read"}; !reflect.DeepEqual(protocol.calls, want) {
		t.Fatalf("calls=%v want=%v", protocol.calls, want)
	}
	if protocol.readThreadID != "thread-ref-goal-read-001" || !protocol.readIncludeTurns {
		t.Fatalf("read thread id=%q include=%v", protocol.readThreadID, protocol.readIncludeTurns)
	}
}

func TestServerCodexAppServerGoalBackendV0BloqueaThreadReadSinResultadoTrasTimeoutV0(t *testing.T) {
	now := time.Date(2026, 6, 30, 3, 20, 0, 0, time.UTC)
	runtime := &serverCodexAppServerGoalRuntimeV0{}
	runtime.recordStartedAtV0("thread-ref-goal-read-timeout-001", now.Add(-3*time.Second))
	protocol := &fakeCodexAppServerProtocolV0{
		getGoalErr: codexAppServerCallErrorV0{Code: "codex_app_server_rpc_method_not_found"},
		readThread: serverCodexAppServerThreadReadV0{
			ID:     "thread-ref-goal-read-timeout-001",
			Status: serverCodexAppServerThreadStatusV0("idle"),
		},
	}
	backend := serverCodexAppServerGoalBackendV0{
		Protocol: protocol,
		CWD:      t.TempDir(),
		Runtime:  runtime,
		Timeout:  2 * time.Second,
		Now:      func() time.Time { return now },
	}

	receipt, err := backend.ObserveCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		GoalRef:         "goal-ref-codex-app-server-read-timeout-001",
		ExternalGoalRef: "thread-ref-goal-read-timeout-001",
	})

	if err != nil {
		t.Fatalf("ObserveCodexGoalV0 fallback: %v", err)
	}
	if receipt.Status != orquestagoal.GoalStatusBlockedV0 ||
		receipt.IssueCode != "codex_app_server_goal_result_missing_after_timeout" ||
		receipt.Summary != "codex_app_server_goal_result_missing_after_timeout" ||
		!containsStringForTestV0(receipt.EvidenceRefs, "evidence-ref-codex-app-server-goal-rpc-unsupported") ||
		!containsStringForTestV0(receipt.EvidenceRefs, "evidence-ref-codex-app-server-goal-result-missing-after-timeout") {
		t.Fatalf("receipt=%+v", receipt)
	}
	if !reflect.DeepEqual(protocol.calls, []string{"thread/goal/get", "thread/read"}) {
		t.Fatalf("calls=%v", protocol.calls)
	}
}

func TestServerCodexAppServerGoalBackendV0NoBloqueaThreadReadConTurnoActivoTrasTimeoutV0(t *testing.T) {
	now := time.Date(2026, 6, 30, 3, 21, 0, 0, time.UTC)
	runtime := &serverCodexAppServerGoalRuntimeV0{}
	runtime.recordStartedAtV0("thread-ref-goal-read-active-001", now.Add(-3*time.Second))
	protocol := &fakeCodexAppServerProtocolV0{
		getGoalErr: codexAppServerCallErrorV0{Code: "codex_app_server_rpc_method_not_found"},
		readThread: serverCodexAppServerThreadReadV0{
			ID:     "thread-ref-goal-read-active-001",
			Status: serverCodexAppServerThreadStatusV0("idle"),
			Turns: []serverCodexAppServerReadTurnV0{{
				ID:     "turn-ref-goal-read-active-001",
				Status: "inProgress",
			}},
		},
	}
	backend := serverCodexAppServerGoalBackendV0{
		Protocol: protocol,
		CWD:      t.TempDir(),
		Runtime:  runtime,
		Timeout:  2 * time.Second,
		Now:      func() time.Time { return now },
	}

	receipt, err := backend.ObserveCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		GoalRef:         "goal-ref-codex-app-server-read-active-001",
		ExternalGoalRef: "thread-ref-goal-read-active-001",
	})

	if err != nil {
		t.Fatalf("ObserveCodexGoalV0 fallback: %v", err)
	}
	if receipt.Status != orquestagoal.GoalStatusRunningV0 ||
		receipt.IssueCode == "codex_app_server_goal_result_missing_after_timeout" ||
		containsStringForTestV0(receipt.EvidenceRefs, "evidence-ref-codex-app-server-goal-result-missing-after-timeout") {
		t.Fatalf("receipt=%+v", receipt)
	}
	if !reflect.DeepEqual(protocol.calls, []string{"thread/goal/get", "thread/read"}) {
		t.Fatalf("calls=%v", protocol.calls)
	}
}

func TestServerCodexAppServerGoalBackendV0NoPromueveThreadReadResultadoPendienteV0(t *testing.T) {
	projectDir := t.TempDir()
	resultDir := filepath.Join(projectDir, "generated-apps", "agenda", "docs")
	if err := os.MkdirAll(resultDir, 0o755); err != nil {
		t.Fatalf("mkdir result dir: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(resultDir, orquestaruntimecodexgoal.CodexGoalResultFileNameV0),
		[]byte(`{"goal_ref":"goal-ref-codex-app-server-read-pending-001","summary":"resultado inicial","required_test_results":[{"test_ref":"test-ref-pending","status":"pending"}]}`),
		0o644,
	); err != nil {
		t.Fatalf("write result file: %v", err)
	}
	protocol := &fakeCodexAppServerProtocolV0{
		getGoalErr: codexAppServerCallErrorV0{Code: "codex_app_server_rpc_method_not_found"},
		readThread: serverCodexAppServerThreadReadV0{
			ID:     "thread-ref-goal-read-pending-001",
			Status: serverCodexAppServerThreadStatusV0("idle"),
		},
	}
	backend := serverCodexAppServerGoalBackendV0{
		Protocol: protocol,
		CWD:      projectDir,
		Runtime:  &serverCodexAppServerGoalRuntimeV0{},
		Timeout:  2 * time.Second,
		Now:      func() time.Time { return time.Date(2026, 6, 30, 3, 22, 0, 0, time.UTC) },
	}

	receipt, err := backend.ObserveCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		GoalRef:         "goal-ref-codex-app-server-read-pending-001",
		ExternalGoalRef: "thread-ref-goal-read-pending-001",
	})

	if err != nil {
		t.Fatalf("ObserveCodexGoalV0 fallback: %v", err)
	}
	if receipt.Status != orquestagoal.GoalStatusRunningV0 ||
		receipt.Summary == "resultado inicial" ||
		containsStringForTestV0(receipt.EvidenceRefs, "evidence-ref-codex-app-server-goal-result-file") {
		t.Fatalf("receipt=%+v", receipt)
	}
	if !reflect.DeepEqual(protocol.calls, []string{"thread/goal/get", "thread/read"}) {
		t.Fatalf("calls=%v", protocol.calls)
	}
}

func TestServerCodexAppServerGoalBackendV0DiagnosticaThreadSystemErrorSinResultadoDurableV0(t *testing.T) {
	protocol := &fakeCodexAppServerProtocolV0{
		getGoalErr: codexAppServerCallErrorV0{Code: "codex_app_server_rpc_method_not_found"},
		readThread: serverCodexAppServerThreadReadV0{
			ID:     "thread-ref-goal-read-system-error-001",
			Status: serverCodexAppServerThreadStatusV0("systemError"),
		},
	}
	backend := serverCodexAppServerGoalBackendV0{
		Protocol: protocol,
		CWD:      t.TempDir(),
	}

	receipt, err := backend.ObserveCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		GoalRef:         "goal-ref-codex-app-server-read-system-error-001",
		ExternalGoalRef: "thread-ref-goal-read-system-error-001",
	})

	if err != nil {
		t.Fatalf("ObserveCodexGoalV0 fallback: %v", err)
	}
	if receipt.Status != orquestagoal.GoalStatusBlockedV0 ||
		receipt.IssueCode != "codex_app_server_thread_system_error" ||
		receipt.Summary != "codex_app_server_thread_status_systemError" ||
		!containsStringForTestV0(receipt.EvidenceRefs, "evidence-ref-codex-app-server-goal-rpc-unsupported") ||
		!containsStringForTestV0(receipt.EvidenceRefs, "evidence-ref-codex-app-server-thread-system-error") {
		t.Fatalf("receipt=%+v", receipt)
	}
	if !reflect.DeepEqual(protocol.calls, []string{"thread/goal/get", "thread/read"}) {
		t.Fatalf("calls=%v", protocol.calls)
	}
}

func TestServerCodexAppServerGoalBackendV0CompactaObjetivoLargoParaAppServerV0(t *testing.T) {
	protocol := &fakeCodexAppServerProtocolV0{
		thread: serverCodexAppServerThreadV0{ID: "thread-ref-goal-long-objective"},
		goal:   serverCodexAppServerThreadGoalV0{ThreadID: "thread-ref-goal-long-objective", Status: "active"},
		turn:   serverCodexAppServerTurnV0{ID: "turn-ref-goal-long-objective", Status: "inProgress"},
	}
	backend := serverCodexAppServerGoalBackendV0{Protocol: protocol}
	longObjective := "cerrar automejora goal-first: " + strings.Repeat("contexto operativo ", 260)
	fullPrompt := "prompt conserva el objetivo completo:\n" + longObjective

	receipt, err := backend.StartCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		SchemaVersion: orquestaruntimecodexgoal.CodexGoalStartPacketSchemaV0,
		GoalRef:       "goal-ref-codex-app-server-long-objective",
		Objective:     longObjective,
		Prompt:        fullPrompt,
	})

	if err != nil {
		t.Fatalf("StartCodexGoalV0: %v", err)
	}
	if receipt.Status != orquestagoal.GoalStatusRunningV0 {
		t.Fatalf("receipt=%+v", receipt)
	}
	if got := len([]rune(protocol.setParams.Objective)); got > codexAppServerGoalObjectiveMaxRunesV0 {
		t.Fatalf("objective len=%d max=%d", got, codexAppServerGoalObjectiveMaxRunesV0)
	}
	for _, want := range []string{
		"cerrar automejora goal-first",
		"objective_compacted",
		"original_sha256=",
		"prompt_contains_full_objective",
	} {
		if !strings.Contains(protocol.setParams.Objective, want) {
			t.Fatalf("objective compactado no contiene %q:\n%s", want, protocol.setParams.Objective)
		}
	}
	if protocol.turnParams.InputText != fullPrompt ||
		!strings.Contains(protocol.turnParams.InputText, longObjective) {
		t.Fatalf("prompt no conserva objetivo completo")
	}
}

func TestServerCodexAppServerGoalBackendV0ObservaGoalPorThreadIDV0(t *testing.T) {
	protocol := &fakeCodexAppServerProtocolV0{
		observedGoal: &serverCodexAppServerThreadGoalV0{
			ThreadID: "thread-ref-goal-002",
			Status:   "complete",
		},
		readThread: serverCodexAppServerThreadReadV0{
			ID:     "thread-ref-goal-002",
			Status: "closed",
		},
	}
	backend := serverCodexAppServerGoalBackendV0{Protocol: protocol}

	receipt, err := backend.ObserveCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		SchemaVersion:   orquestaruntimecodexgoal.CodexGoalObservationRequestSchemaV0,
		GoalRef:         "goal-ref-codex-app-server-002",
		ExternalGoalRef: "thread-ref-goal-002",
	})

	if err != nil {
		t.Fatalf("ObserveCodexGoalV0: %v", err)
	}
	if receipt.Status != orquestagoal.GoalStatusCompleteV0 ||
		receipt.GoalRef != "goal-ref-codex-app-server-002" ||
		receipt.ExternalGoalRef != "thread-ref-goal-002" ||
		!containsStringForTestV0(receipt.EvidenceRefs, "evidence-ref-codex-app-server-goal-observed") {
		t.Fatalf("receipt=%+v", receipt)
	}
	if !reflect.DeepEqual(protocol.calls, []string{"thread/goal/get", "thread/read"}) {
		t.Fatalf("calls=%v", protocol.calls)
	}
	if protocol.getThreadID != "thread-ref-goal-002" {
		t.Fatalf("get thread id=%q", protocol.getThreadID)
	}
	if protocol.readThreadID != "thread-ref-goal-002" || !protocol.readIncludeTurns {
		t.Fatalf("read thread id=%q include=%v", protocol.readThreadID, protocol.readIncludeTurns)
	}
}

func TestServerCodexAppServerGoalBackendV0ObservaResultadoMarcadoV0(t *testing.T) {
	marked := `ORQUESTA_GOAL_RESULT_V0 {"summary":"cierre con {llaves}","artifact_refs":["artifact-ref-goal-summary"],"required_test_results":[{"test_ref":"test-ref-goal","status":"passed","evidence_refs":["evidence-ref-test-pass"]}],"domain_receipt_refs":["domain-receipt-ref-001"],"evidence_refs":["evidence-ref-required"]}`
	protocol := &fakeCodexAppServerProtocolV0{
		observedGoal: &serverCodexAppServerThreadGoalV0{
			ThreadID: "thread-ref-goal-003",
			Status:   "complete",
		},
		readThread: serverCodexAppServerThreadReadV0{
			ID: "thread-ref-goal-003",
			Turns: []serverCodexAppServerReadTurnV0{{
				ID:        "turn-ref-goal-003",
				Status:    "completed",
				ItemsView: "full",
				Items: []serverCodexAppServerReadItemV0{{
					ID:    "item-ref-goal-003",
					Type:  "agentMessage",
					Phase: "final_answer",
					Text:  "Hecho.\n" + marked,
				}},
			}},
		},
	}
	backend := serverCodexAppServerGoalBackendV0{Protocol: protocol}

	receipt, err := backend.ObserveCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		SchemaVersion:   orquestaruntimecodexgoal.CodexGoalObservationRequestSchemaV0,
		GoalRef:         "goal-ref-codex-app-server-003",
		ExternalGoalRef: "thread-ref-goal-003",
	})

	if err != nil {
		t.Fatalf("ObserveCodexGoalV0: %v", err)
	}
	if receipt.Summary != "cierre con {llaves}" ||
		!containsStringForTestV0(receipt.ArtifactRefs, "artifact-ref-goal-summary") ||
		!containsStringForTestV0(receipt.DomainReceiptRefs, "domain-receipt-ref-001") ||
		!containsStringForTestV0(receipt.EvidenceRefs, "evidence-ref-required") ||
		!containsStringForTestV0(receipt.EvidenceRefs, "evidence-ref-codex-app-server-goal-result-marker") ||
		len(receipt.RequiredTestResults) != 1 ||
		receipt.RequiredTestResults[0].TestRef != "test-ref-goal" ||
		receipt.RequiredTestResults[0].Status != "passed" {
		t.Fatalf("receipt=%+v", receipt)
	}
}

func TestServerCodexAppServerGoalBackendV0ObservaResultadoDurableSinMarcadorV0(t *testing.T) {
	projectDir := t.TempDir()
	resultDir := filepath.Join(projectDir, "generated-apps", "agenda", "docs")
	if err := os.MkdirAll(resultDir, 0o755); err != nil {
		t.Fatalf("mkdir result dir: %v", err)
	}
	resultPath := filepath.Join(resultDir, orquestaruntimecodexgoal.CodexGoalResultFileNameV0)
	payload := `{
		"goal_ref":"goal-ref-codex-app-server-file-001",
		"summary":"resultado durable",
		"artifact_refs":["artifact-ref-goal-source","artifact-ref-goal-handoff"],
		"required_test_results":[{"test_ref":"test-ref-goal","status":"passed","evidence_refs":["evidence-ref-test-pass"]}],
		"evidence_refs":["evidence-ref-required"]
	}`
	if err := os.WriteFile(resultPath, []byte(payload), 0o644); err != nil {
		t.Fatalf("write result file: %v", err)
	}
	protocol := &fakeCodexAppServerProtocolV0{
		observedGoal: &serverCodexAppServerThreadGoalV0{
			ThreadID: "thread-ref-goal-file-001",
			Status:   "complete",
		},
		readThread: serverCodexAppServerThreadReadV0{
			ID:     "thread-ref-goal-file-001",
			Status: "closed",
			Turns: []serverCodexAppServerReadTurnV0{{
				ID: "turn-ref-goal-file-001",
				Items: []serverCodexAppServerReadItemV0{{
					ID:    "item-ref-goal-file-001",
					Type:  "agentMessage",
					Phase: "commentary",
					Text:  "marcado completo por update_goal",
				}},
			}},
		},
	}
	backend := serverCodexAppServerGoalBackendV0{Protocol: protocol, CWD: projectDir}

	receipt, err := backend.ObserveCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		SchemaVersion:   orquestaruntimecodexgoal.CodexGoalObservationRequestSchemaV0,
		GoalRef:         "goal-ref-codex-app-server-file-001",
		ExternalGoalRef: "thread-ref-goal-file-001",
	})

	if err != nil {
		t.Fatalf("ObserveCodexGoalV0: %v", err)
	}
	if receipt.Summary != "resultado durable" ||
		!containsStringForTestV0(receipt.ArtifactRefs, "artifact-ref-goal-source") ||
		!containsStringForTestV0(receipt.EvidenceRefs, "evidence-ref-required") ||
		!containsStringForTestV0(receipt.EvidenceRefs, "evidence-ref-codex-app-server-goal-result-file") ||
		len(receipt.RequiredTestResults) != 1 ||
		receipt.RequiredTestResults[0].TestRef != "test-ref-goal" ||
		receipt.RequiredTestResults[0].Status != "passed" {
		t.Fatalf("receipt=%+v", receipt)
	}
}

func TestServerCodexAppServerGoalBackendV0PromueveResultadoDurableAunqueGoalSigaActivoV0(t *testing.T) {
	projectDir := t.TempDir()
	resultDir := filepath.Join(projectDir, "generated-apps", "agenda", "docs")
	if err := os.MkdirAll(resultDir, 0o755); err != nil {
		t.Fatalf("mkdir result dir: %v", err)
	}
	resultPath := filepath.Join(resultDir, orquestaruntimecodexgoal.CodexGoalResultFileNameV0)
	payload := `{
		"goal_ref":"goal-ref-codex-app-server-file-active-001",
		"summary":"resultado durable antes de cierre backend",
		"artifact_refs":["artifact-ref-goal-active"],
		"required_test_results":[{"test_ref":"test-ref-goal-active","status":"passed","evidence_refs":["evidence-ref-test-pass"]}],
		"domain_receipt_refs":["domain-receipt-ref-active"],
		"evidence_refs":["evidence-ref-required-active"]
	}`
	if err := os.WriteFile(resultPath, []byte(payload), 0o644); err != nil {
		t.Fatalf("write result file: %v", err)
	}
	protocol := &fakeCodexAppServerProtocolV0{
		observedGoal: &serverCodexAppServerThreadGoalV0{
			ThreadID: "thread-ref-goal-file-active-001",
			Status:   "active",
		},
	}
	backend := serverCodexAppServerGoalBackendV0{Protocol: protocol, CWD: projectDir}

	receipt, err := backend.ObserveCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		SchemaVersion:   orquestaruntimecodexgoal.CodexGoalObservationRequestSchemaV0,
		GoalRef:         "goal-ref-codex-app-server-file-active-001",
		ExternalGoalRef: "thread-ref-goal-file-active-001",
	})

	if err != nil {
		t.Fatalf("ObserveCodexGoalV0: %v", err)
	}
	if receipt.Status != orquestagoal.GoalStatusCompleteV0 ||
		receipt.Summary != "resultado durable antes de cierre backend" ||
		!containsStringForTestV0(receipt.ArtifactRefs, "artifact-ref-goal-active") ||
		!containsStringForTestV0(receipt.DomainReceiptRefs, "domain-receipt-ref-active") ||
		!containsStringForTestV0(receipt.EvidenceRefs, "evidence-ref-codex-app-server-goal-result-file") ||
		len(receipt.RequiredTestResults) != 1 ||
		receipt.RequiredTestResults[0].TestRef != "test-ref-goal-active" ||
		receipt.RequiredTestResults[0].Status != "passed" {
		t.Fatalf("receipt=%+v", receipt)
	}
	if !reflect.DeepEqual(protocol.calls, []string{"thread/goal/get", "thread/read"}) {
		t.Fatalf("calls=%v", protocol.calls)
	}
}

func TestServerCodexAppServerGoalBackendV0NoPromueveResultadoDurableActivoConTestsPendientesV0(t *testing.T) {
	projectDir := t.TempDir()
	resultDir := filepath.Join(projectDir, "generated-apps", "agenda", "docs")
	if err := os.MkdirAll(resultDir, 0o755); err != nil {
		t.Fatalf("mkdir result dir: %v", err)
	}
	resultPath := filepath.Join(resultDir, orquestaruntimecodexgoal.CodexGoalResultFileNameV0)
	payload := `{
		"goal_ref":"goal-ref-codex-app-server-file-active-pending-001",
		"summary":"resultado inicial materializado; pendiente de completar artefactos y verificacion",
		"artifact_refs":[],
		"required_test_results":[{"test_ref":"test-ref-goal-active-pending","status":"pending","evidence_refs":[]}],
		"evidence_refs":[]
	}`
	if err := os.WriteFile(resultPath, []byte(payload), 0o644); err != nil {
		t.Fatalf("write result file: %v", err)
	}
	protocol := &fakeCodexAppServerProtocolV0{
		observedGoal: &serverCodexAppServerThreadGoalV0{
			ThreadID: "thread-ref-goal-file-active-pending-001",
			Status:   "active",
		},
	}
	backend := serverCodexAppServerGoalBackendV0{Protocol: protocol, CWD: projectDir}

	receipt, err := backend.ObserveCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		SchemaVersion:   orquestaruntimecodexgoal.CodexGoalObservationRequestSchemaV0,
		GoalRef:         "goal-ref-codex-app-server-file-active-pending-001",
		ExternalGoalRef: "thread-ref-goal-file-active-pending-001",
	})

	if err != nil {
		t.Fatalf("ObserveCodexGoalV0: %v", err)
	}
	if receipt.Status != orquestagoal.GoalStatusRunningV0 ||
		receipt.Summary == "resultado inicial materializado; pendiente de completar artefactos y verificacion" ||
		containsStringForTestV0(receipt.EvidenceRefs, "evidence-ref-codex-app-server-goal-result-file") {
		t.Fatalf("receipt=%+v", receipt)
	}
	if !reflect.DeepEqual(protocol.calls, []string{"thread/goal/get", "thread/read"}) {
		t.Fatalf("calls=%v", protocol.calls)
	}
}

func TestServerCodexAppServerGoalBackendV0ConservaBloqueadoAlFusionarResultadoDurableV0(t *testing.T) {
	projectDir := t.TempDir()
	resultDir := filepath.Join(projectDir, "external", "opes", "update_topic_registry", "docs")
	if err := os.MkdirAll(resultDir, 0o755); err != nil {
		t.Fatalf("mkdir result dir: %v", err)
	}
	resultPath := filepath.Join(resultDir, orquestaruntimecodexgoal.CodexGoalResultFileNameV0)
	payload := `{
		"goal_ref":"goal-ref-codex-app-server-file-blocked-001",
		"summary":"resultado durable con artefacto aunque el goal remoto quedo bloqueado",
		"artifact_refs":["artifact-ref-goal-blocked"],
		"required_test_results":[{"test_ref":"opes-domain-test-blocked","status":"blocked","evidence_refs":["evidence-ref-spec"]}],
		"domain_receipt_refs":["domain-receipt-blocked-public-v0"],
		"evidence_refs":["evidence-ref-required-blocked"]
	}`
	if err := os.WriteFile(resultPath, []byte(payload), 0o644); err != nil {
		t.Fatalf("write result file: %v", err)
	}
	protocol := &fakeCodexAppServerProtocolV0{
		observedGoal: &serverCodexAppServerThreadGoalV0{
			ThreadID: "thread-ref-goal-file-blocked-001",
			Status:   "blocked",
		},
	}
	backend := serverCodexAppServerGoalBackendV0{Protocol: protocol, CWD: projectDir}

	receipt, err := backend.ObserveCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		SchemaVersion:   orquestaruntimecodexgoal.CodexGoalObservationRequestSchemaV0,
		GoalRef:         "goal-ref-codex-app-server-file-blocked-001",
		ExternalGoalRef: "thread-ref-goal-file-blocked-001",
	})

	if err != nil {
		t.Fatalf("ObserveCodexGoalV0: %v", err)
	}
	if receipt.Status != orquestagoal.GoalStatusBlockedV0 ||
		receipt.Summary != "resultado durable con artefacto aunque el goal remoto quedo bloqueado" ||
		!containsStringForTestV0(receipt.ArtifactRefs, "artifact-ref-goal-blocked") ||
		!containsStringForTestV0(receipt.DomainReceiptRefs, "domain-receipt-blocked-public-v0") ||
		!containsStringForTestV0(receipt.EvidenceRefs, "evidence-ref-codex-app-server-goal-result-file") ||
		len(receipt.RequiredTestResults) != 1 ||
		receipt.RequiredTestResults[0].TestRef != "opes-domain-test-blocked" {
		t.Fatalf("receipt=%+v", receipt)
	}
	if !reflect.DeepEqual(protocol.calls, []string{"thread/goal/get", "thread/read"}) {
		t.Fatalf("calls=%v", protocol.calls)
	}
}

func TestServerCodexAppServerGoalBackendV0BloqueaGoalActivoPorTimeoutV0(t *testing.T) {
	protocol := &fakeCodexAppServerProtocolV0{
		observedGoal: &serverCodexAppServerThreadGoalV0{
			ThreadID:        "thread-ref-goal-timeout-001",
			Status:          "active",
			TimeUsedSeconds: 3,
		},
	}
	backend := serverCodexAppServerGoalBackendV0{
		Protocol: protocol,
		Timeout:  2 * time.Second,
	}

	receipt, err := backend.ObserveCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		SchemaVersion:   orquestaruntimecodexgoal.CodexGoalObservationRequestSchemaV0,
		GoalRef:         "goal-ref-timeout-001",
		ExternalGoalRef: "thread-ref-goal-timeout-001",
	})

	if err != nil {
		t.Fatalf("ObserveCodexGoalV0: %v", err)
	}
	if receipt.Status != orquestagoal.GoalStatusBlockedV0 ||
		receipt.IssueCode != "codex_app_server_goal_active_timeout" ||
		receipt.Summary != "codex_app_server_goal_active_timeout" ||
		!containsStringForTestV0(receipt.EvidenceRefs, "evidence-ref-codex-app-server-goal-active-timeout") {
		t.Fatalf("receipt=%+v", receipt)
	}
	if !reflect.DeepEqual(protocol.calls, []string{"thread/goal/get", "thread/read"}) {
		t.Fatalf("calls=%v", protocol.calls)
	}
}

func TestServerCodexAppServerGoalBackendV0BloqueaGoalActivoPorCreatedAtSiTimeUsedFaltaV0(t *testing.T) {
	now := time.Date(2026, 6, 28, 17, 20, 0, 0, time.UTC)
	protocol := &fakeCodexAppServerProtocolV0{
		observedGoal: &serverCodexAppServerThreadGoalV0{
			ThreadID:  "thread-ref-goal-timeout-created-001",
			Status:    "active",
			CreatedAt: serverCodexAppServerTimestampFromTimeV0(now.Add(-3 * time.Second)),
		},
	}
	backend := serverCodexAppServerGoalBackendV0{
		Protocol: protocol,
		Timeout:  2 * time.Second,
		Now:      func() time.Time { return now },
	}

	receipt, err := backend.ObserveCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		SchemaVersion:   orquestaruntimecodexgoal.CodexGoalObservationRequestSchemaV0,
		GoalRef:         "goal-ref-timeout-created-001",
		ExternalGoalRef: "thread-ref-goal-timeout-created-001",
	})

	if err != nil {
		t.Fatalf("ObserveCodexGoalV0: %v", err)
	}
	if receipt.Status != orquestagoal.GoalStatusBlockedV0 ||
		receipt.IssueCode != "codex_app_server_goal_active_timeout" {
		t.Fatalf("receipt=%+v", receipt)
	}
}

func TestServerCodexAppServerGoalBackendV0BloqueaGoalActivoPorRuntimeSiTimeUsedFaltaV0(t *testing.T) {
	now := time.Date(2026, 6, 28, 17, 25, 0, 0, time.UTC)
	protocol := &fakeCodexAppServerProtocolV0{
		thread: serverCodexAppServerThreadV0{ID: "thread-ref-goal-timeout-runtime-001"},
		turn:   serverCodexAppServerTurnV0{ID: "turn-ref-goal-timeout-runtime-001"},
		observedGoal: &serverCodexAppServerThreadGoalV0{
			ThreadID: "thread-ref-goal-timeout-runtime-001",
			Status:   "active",
		},
	}
	backend := serverCodexAppServerGoalBackendV0{
		Protocol: protocol,
		Timeout:  2 * time.Second,
		Runtime:  &serverCodexAppServerGoalRuntimeV0{},
		Now:      func() time.Time { return now },
	}
	if _, err := backend.StartCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		SchemaVersion: orquestaruntimecodexgoal.CodexGoalStartPacketSchemaV0,
		GoalRef:       "goal-ref-timeout-runtime-001",
		Objective:     "probar timeout runtime",
		Prompt:        "haz una entrega compacta",
	}); err != nil {
		t.Fatalf("StartCodexGoalV0: %v", err)
	}
	now = now.Add(3 * time.Second)

	receipt, err := backend.ObserveCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		SchemaVersion:   orquestaruntimecodexgoal.CodexGoalObservationRequestSchemaV0,
		GoalRef:         "goal-ref-timeout-runtime-001",
		ExternalGoalRef: "thread-ref-goal-timeout-runtime-001",
	})

	if err != nil {
		t.Fatalf("ObserveCodexGoalV0: %v", err)
	}
	if receipt.Status != orquestagoal.GoalStatusBlockedV0 ||
		receipt.IssueCode != "codex_app_server_goal_active_timeout" {
		t.Fatalf("receipt=%+v", receipt)
	}
}

func TestServerCodexAppServerGoalBackendV0ObservaResultadoDurableSiThreadReadFallaV0(t *testing.T) {
	projectDir := t.TempDir()
	resultDir := filepath.Join(projectDir, "generated-apps", "agenda", "docs")
	if err := os.MkdirAll(resultDir, 0o755); err != nil {
		t.Fatalf("mkdir result dir: %v", err)
	}
	resultPath := filepath.Join(resultDir, orquestaruntimecodexgoal.CodexGoalResultFileNameV0)
	payload := `{
		"goal_ref":"goal-ref-codex-app-server-file-002",
		"summary":"resultado durable tras fallo de thread/read",
		"artifact_refs":["artifact-ref-goal-source"],
		"required_test_results":[{"test_ref":"test-ref-goal","status":"passed","evidence_refs":["evidence-ref-test-pass"]}],
		"evidence_refs":["evidence-ref-required"]
	}`
	if err := os.WriteFile(resultPath, []byte(payload), 0o644); err != nil {
		t.Fatalf("write result file: %v", err)
	}
	protocol := &fakeCodexAppServerProtocolV0{
		observedGoal: &serverCodexAppServerThreadGoalV0{
			ThreadID: "thread-ref-goal-file-002",
			Status:   "complete",
		},
		readThreadErr: errors.New("thread read unavailable"),
	}
	backend := serverCodexAppServerGoalBackendV0{Protocol: protocol, CWD: projectDir}

	receipt, err := backend.ObserveCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		SchemaVersion:   orquestaruntimecodexgoal.CodexGoalObservationRequestSchemaV0,
		GoalRef:         "goal-ref-codex-app-server-file-002",
		ExternalGoalRef: "thread-ref-goal-file-002",
	})

	if err != nil {
		t.Fatalf("ObserveCodexGoalV0: %v", err)
	}
	if receipt.IssueCode != "" ||
		receipt.Summary != "resultado durable tras fallo de thread/read" ||
		!containsStringForTestV0(receipt.ArtifactRefs, "artifact-ref-goal-source") ||
		!containsStringForTestV0(receipt.EvidenceRefs, "evidence-ref-required") ||
		!containsStringForTestV0(receipt.EvidenceRefs, "evidence-ref-codex-app-server-goal-result-file") ||
		len(receipt.RequiredTestResults) != 1 ||
		receipt.RequiredTestResults[0].TestRef != "test-ref-goal" {
		t.Fatalf("receipt=%+v", receipt)
	}
	if !reflect.DeepEqual(protocol.calls, []string{"thread/goal/get", "thread/read"}) {
		t.Fatalf("calls=%v", protocol.calls)
	}
}

func TestServerCodexAppServerGoalBackendV0PrefiereResultadoDurableSobreMarcadorV0(t *testing.T) {
	projectDir := t.TempDir()
	resultDir := filepath.Join(projectDir, "generated-apps", "agenda", "docs")
	if err := os.MkdirAll(resultDir, 0o755); err != nil {
		t.Fatalf("mkdir result dir: %v", err)
	}
	resultPath := filepath.Join(resultDir, orquestaruntimecodexgoal.CodexGoalResultFileNameV0)
	payload := `{
		"goal_ref":"goal-ref-codex-app-server-file-003",
		"summary":"resultado durable preferente",
		"artifact_refs":["artifact-ref-goal-source"],
		"required_test_results":[{"test_ref":"test-ref-goal","status":"passed","evidence_refs":["evidence-ref-test-pass"]}],
		"evidence_refs":["evidence-ref-required"]
	}`
	if err := os.WriteFile(resultPath, []byte(payload), 0o644); err != nil {
		t.Fatalf("write result file: %v", err)
	}
	marker := orquestaruntimecodexgoal.CodexGoalResultMarkerV0 + ` {"goal_ref":"goal-ref-codex-app-server-file-003","summary":"marcador incompleto","evidence_refs":["evidence-ref-marker"]}`
	protocol := &fakeCodexAppServerProtocolV0{
		observedGoal: &serverCodexAppServerThreadGoalV0{
			ThreadID: "thread-ref-goal-file-003",
			Status:   "complete",
		},
		readThread: serverCodexAppServerThreadReadV0{
			ID: "thread-ref-goal-file-003",
			Turns: []serverCodexAppServerReadTurnV0{{
				ID: "turn-ref-goal-file-003",
				Items: []serverCodexAppServerReadItemV0{{
					ID:    "item-ref-goal-file-003",
					Type:  "agentMessage",
					Phase: "final_answer",
					Text:  marker,
				}},
			}},
		},
	}
	backend := serverCodexAppServerGoalBackendV0{Protocol: protocol, CWD: projectDir}

	receipt, err := backend.ObserveCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		SchemaVersion:   orquestaruntimecodexgoal.CodexGoalObservationRequestSchemaV0,
		GoalRef:         "goal-ref-codex-app-server-file-003",
		ExternalGoalRef: "thread-ref-goal-file-003",
	})

	if err != nil {
		t.Fatalf("ObserveCodexGoalV0: %v", err)
	}
	if receipt.IssueCode != "" ||
		receipt.Summary != "resultado durable preferente" ||
		!containsStringForTestV0(receipt.ArtifactRefs, "artifact-ref-goal-source") ||
		!containsStringForTestV0(receipt.EvidenceRefs, "evidence-ref-codex-app-server-goal-result-file") ||
		containsStringForTestV0(receipt.EvidenceRefs, "evidence-ref-codex-app-server-goal-result-marker") {
		t.Fatalf("receipt=%+v", receipt)
	}
}

func TestServerCodexAppServerGoalBackendV0UsaDurableSiMarcadorEsInvalidoV0(t *testing.T) {
	projectDir := t.TempDir()
	resultDir := filepath.Join(projectDir, "generated-apps", "agenda", "docs")
	if err := os.MkdirAll(resultDir, 0o755); err != nil {
		t.Fatalf("mkdir result dir: %v", err)
	}
	resultPath := filepath.Join(resultDir, orquestaruntimecodexgoal.CodexGoalResultFileNameV0)
	payload := `{
		"goal_ref":"goal-ref-codex-app-server-file-004",
		"summary":"resultado durable con marcador invalido",
		"artifact_refs":["artifact-ref-goal-source"],
		"required_test_results":[{"test_ref":"test-ref-goal","status":"passed","evidence_refs":["evidence-ref-test-pass"]}],
		"evidence_refs":["evidence-ref-required"]
	}`
	if err := os.WriteFile(resultPath, []byte(payload), 0o644); err != nil {
		t.Fatalf("write result file: %v", err)
	}
	protocol := &fakeCodexAppServerProtocolV0{
		observedGoal: &serverCodexAppServerThreadGoalV0{
			ThreadID: "thread-ref-goal-file-004",
			Status:   "complete",
		},
		readThread: serverCodexAppServerThreadReadV0{
			ID: "thread-ref-goal-file-004",
			Turns: []serverCodexAppServerReadTurnV0{{
				ID: "turn-ref-goal-file-004",
				Items: []serverCodexAppServerReadItemV0{{
					ID:    "item-ref-goal-file-004",
					Type:  "agentMessage",
					Phase: "final_answer",
					Text:  orquestaruntimecodexgoal.CodexGoalResultMarkerV0 + ` sin-json`,
				}},
			}},
		},
	}
	backend := serverCodexAppServerGoalBackendV0{Protocol: protocol, CWD: projectDir}

	receipt, err := backend.ObserveCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		SchemaVersion:   orquestaruntimecodexgoal.CodexGoalObservationRequestSchemaV0,
		GoalRef:         "goal-ref-codex-app-server-file-004",
		ExternalGoalRef: "thread-ref-goal-file-004",
	})

	if err != nil {
		t.Fatalf("ObserveCodexGoalV0: %v", err)
	}
	if receipt.IssueCode != "" ||
		receipt.Summary != "resultado durable con marcador invalido" ||
		!containsStringForTestV0(receipt.EvidenceRefs, "evidence-ref-codex-app-server-goal-result-file") {
		t.Fatalf("receipt=%+v", receipt)
	}
}

func TestServerCodexAppServerGoalBackendV0DiagnosticaThreadReadSinResultadoDurableV0(t *testing.T) {
	protocol := &fakeCodexAppServerProtocolV0{
		observedGoal: &serverCodexAppServerThreadGoalV0{
			ThreadID: "thread-ref-goal-file-005",
			Status:   "complete",
		},
		readThreadErr: errors.New("thread read unavailable"),
	}
	backend := serverCodexAppServerGoalBackendV0{Protocol: protocol, CWD: t.TempDir()}

	receipt, err := backend.ObserveCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		SchemaVersion:   orquestaruntimecodexgoal.CodexGoalObservationRequestSchemaV0,
		GoalRef:         "goal-ref-codex-app-server-file-005",
		ExternalGoalRef: "thread-ref-goal-file-005",
	})

	if err != nil {
		t.Fatalf("ObserveCodexGoalV0: %v", err)
	}
	if receipt.IssueCode != "codex_app_server_thread_read_failed" {
		t.Fatalf("receipt=%+v", receipt)
	}
}

func TestCodexAppServerGoalResultMarkerV0AceptaFallbackSinPhaseV0(t *testing.T) {
	thread := serverCodexAppServerThreadReadV0{
		ID: "thread-ref-goal-004",
		Turns: []serverCodexAppServerReadTurnV0{{
			ID: "turn-ref-goal-004",
			Items: []serverCodexAppServerReadItemV0{{
				ID:   "item-ref-goal-004",
				Type: "agentMessage",
				Text: `texto ` + orquestaruntimecodexgoal.CodexGoalResultMarkerV0 + ` {"summary":"ok","evidence_refs":["evidence-ref-ok"]}`,
			}},
		}},
	}

	marked, found, err := codexAppServerGoalResultFromThreadV0(thread)

	if err != nil || !found || marked.Summary != "ok" ||
		!containsStringForTestV0(marked.EvidenceRefs, "evidence-ref-ok") {
		t.Fatalf("marked=%+v found=%v err=%v", marked, found, err)
	}
}

func TestCodexAppServerGoalResultFromWorkspaceV0IgnoraArchivoInvalidoAntesDelValidoV0(t *testing.T) {
	projectDir := t.TempDir()
	invalidDir := filepath.Join(projectDir, "generated-apps", "a-stale", "docs")
	validDir := filepath.Join(projectDir, "generated-apps", "z-valid", "docs")
	if err := os.MkdirAll(invalidDir, 0o755); err != nil {
		t.Fatalf("mkdir invalid dir: %v", err)
	}
	if err := os.MkdirAll(validDir, 0o755); err != nil {
		t.Fatalf("mkdir valid dir: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(invalidDir, orquestaruntimecodexgoal.CodexGoalResultFileNameV0),
		[]byte(`{invalid-json`),
		0o644,
	); err != nil {
		t.Fatalf("write invalid result: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(validDir, orquestaruntimecodexgoal.CodexGoalResultFileNameV0),
		[]byte(`{"goal_ref":"goal-ref-codex-app-server-file-006","summary":"resultado valido","evidence_refs":["evidence-ref-required"]}`),
		0o644,
	); err != nil {
		t.Fatalf("write valid result: %v", err)
	}

	marked, found, err := codexAppServerGoalResultFromWorkspaceV0(projectDir, "goal-ref-codex-app-server-file-006", "")

	if err != nil || !found || marked.Summary != "resultado valido" ||
		!containsStringForTestV0(marked.EvidenceRefs, "evidence-ref-required") {
		t.Fatalf("marked=%+v found=%v err=%v", marked, found, err)
	}
}

func TestCodexAppServerGoalResultFromWorkspaceV0IgnoraArchivoDeOtroThreadV0(t *testing.T) {
	projectDir := t.TempDir()
	resultDir := filepath.Join(projectDir, "generated-apps", "agenda", "docs")
	if err := os.MkdirAll(resultDir, 0o755); err != nil {
		t.Fatalf("mkdir result dir: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(resultDir, orquestaruntimecodexgoal.CodexGoalResultFileNameV0),
		[]byte(`{"goal_ref":"goal-ref-codex-app-server-file-thread-001","external_goal_ref":"thread-ref-other","summary":"resultado stale"}`),
		0o644,
	); err != nil {
		t.Fatalf("write result: %v", err)
	}

	marked, found, err := codexAppServerGoalResultFromWorkspaceV0(
		projectDir,
		"goal-ref-codex-app-server-file-thread-001",
		"thread-ref-current",
	)

	if err != nil || found || marked.Summary != "" {
		t.Fatalf("archivo de otro thread debe ignorarse: marked=%+v found=%v err=%v", marked, found, err)
	}
}

func TestServerCodexAppServerGoalBackendV0RechazaMarcadorDeOtroThreadV0(t *testing.T) {
	marker := orquestaruntimecodexgoal.CodexGoalResultMarkerV0 + ` {"goal_ref":"goal-ref-marker-thread-001","external_goal_ref":"thread-ref-other","summary":"resultado ajeno"}`
	protocol := &fakeCodexAppServerProtocolV0{
		observedGoal: &serverCodexAppServerThreadGoalV0{
			ThreadID: "thread-ref-current",
			Status:   "complete",
		},
		readThread: serverCodexAppServerThreadReadV0{
			ID: "thread-ref-current",
			Turns: []serverCodexAppServerReadTurnV0{{
				ID: "turn-ref-current",
				Items: []serverCodexAppServerReadItemV0{{
					ID:    "item-ref-current",
					Type:  "agentMessage",
					Phase: "final_answer",
					Text:  marker,
				}},
			}},
		},
	}
	backend := serverCodexAppServerGoalBackendV0{Protocol: protocol}

	receipt, err := backend.ObserveCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		SchemaVersion:   orquestaruntimecodexgoal.CodexGoalObservationRequestSchemaV0,
		GoalRef:         "goal-ref-marker-thread-001",
		ExternalGoalRef: "thread-ref-current",
	})

	if err != nil {
		t.Fatalf("ObserveCodexGoalV0: %v", err)
	}
	if receipt.IssueCode != "codex_app_server_goal_result_marker_external_goal_ref_mismatch" ||
		receipt.Summary == "resultado ajeno" ||
		len(receipt.ArtifactRefs) != 0 {
		t.Fatalf("receipt=%+v", receipt)
	}
}

func TestCodexAppServerIssueCodeForErrorV0ClasificaDiagnosticosV0(t *testing.T) {
	cases := []struct {
		name    string
		message string
		want    string
	}{
		{
			name:    "standalone missing",
			message: "Error: managed standalone Codex install not found at /home/user/.codex/packages/standalone/current/codex",
			want:    "codex_app_server_standalone_missing",
		},
		{
			name:    "socket missing",
			message: "Error: failed to connect to socket at /home/user/.codex/app-server-control/app-server-control.sock",
			want:    "codex_app_server_control_socket_missing",
		},
		{
			name:    "permission denied",
			message: "Permission denied (os error 13)",
			want:    "codex_app_server_permission_denied",
		},
		{
			name:    "operation not permitted",
			message: "Error: Operation not permitted (os error 1)",
			want:    "codex_app_server_operation_not_permitted",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := codexAppServerIssueCodeForErrorV0(errors.New(tc.message), "fallback"); got != tc.want {
				t.Fatalf("code=%q want=%q", got, tc.want)
			}
		})
	}
}

func TestCodexAppServerIssueCodeFromCommandFailureV0UsaErrorSinStderrV0(t *testing.T) {
	code := codexAppServerIssueCodeFromCommandFailureV0(
		"",
		errors.New(`exec: "codex-missing": executable file not found in $PATH`),
	)

	if code != "codex_app_server_command_missing" {
		t.Fatalf("code=%q", code)
	}
}

func TestServerCodexAppServerUnavailableBackendV0BloqueaConIssueCodeV0(t *testing.T) {
	backend := serverCodexUnavailableGoalBackendV0{IssueCode: "codex_app_server_control_socket_missing"}

	receipt, err := backend.StartCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef: "goal-ref-unavailable-001",
	})

	if err == nil ||
		receipt.Status != orquestagoal.GoalStatusInvalidV0 ||
		receipt.IssueCode != "codex_app_server_control_socket_missing" ||
		receipt.GoalRef != "goal-ref-unavailable-001" {
		t.Fatalf("receipt=%+v err=%v", receipt, err)
	}
}

func TestServerCodexGoalBackendFromEnvV0ProxyDiagnosticoNoCreaBackendOperativoV0(t *testing.T) {
	script := filepath.Join(t.TempDir(), "codex-fake")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho 'Error: failed to connect to socket at /tmp/app-server-control.sock' >&2\nexit 1\n"), 0o700); err != nil {
		t.Fatalf("write fake codex: %v", err)
	}
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	t.Setenv(envCodexCommandV0, script)
	t.Setenv(envCodexGoalBackendV0, codexGoalBackendAppServerProxyV0)
	t.Setenv(envAllowAppServerProxyDiagnosticV0, "1")
	t.Setenv(envCodexGoalPreflightTimeoutMSV0, "1000")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	_, err = serverCodexGoalBackendFromEnvV0(config)
	if err == nil || !strings.Contains(err.Error(), "codex_goal_backend_proxy_diagnostic_not_operational") {
		t.Fatalf("err=%v", err)
	}
}

func TestServerCodexGoalBackendFromEnvV0ProxyDiagnosticoPreflightOKNoCreaBackendRealV0(t *testing.T) {
	script := filepath.Join(t.TempDir(), "codex-fake-ok")
	if err := os.WriteFile(script, []byte(fakeCodexAppServerPreflightScriptV0("")), 0o700); err != nil {
		t.Fatalf("write fake codex: %v", err)
	}
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	t.Setenv(envCodexCommandV0, script)
	t.Setenv(envCodexGoalBackendV0, codexGoalBackendAppServerProxyV0)
	t.Setenv(envAllowAppServerProxyDiagnosticV0, "1")
	t.Setenv(envCodexGoalPreflightTimeoutMSV0, "1000")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	_, err = serverCodexGoalBackendFromEnvV0(config)
	if err == nil || !strings.Contains(err.Error(), "codex_goal_backend_proxy_diagnostic_not_operational") {
		t.Fatalf("err=%v", err)
	}
}

func TestServerCodexGoalBackendFromEnvV0StdioNoEsBackendOperativoV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	t.Setenv(envCodexGoalBackendV0, "app_server_stdio")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	_, err = serverCodexGoalBackendFromEnvV0(config)
	if err == nil || !strings.Contains(err.Error(), "codex_goal_backend_no_soportado:app_server_stdio") {
		t.Fatalf("err=%v", err)
	}
}

func TestServerCodexGoalBackendFromEnvV0ProxyRequiereOptInDiagnosticoV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	t.Setenv(envCodexGoalBackendV0, codexGoalBackendAppServerProxyV0)

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	_, err = serverCodexGoalBackendFromEnvV0(config)
	if err == nil || !strings.Contains(err.Error(), "codex_goal_backend_proxy_diagnostic_opt_in_required") {
		t.Fatalf("err=%v", err)
	}
	if config.IdleSelfImprovementGoalFirst {
		t.Fatalf("proxy sin opt-in no debe derivar goal-first operativo: %+v", config)
	}
}

func TestCodexGoalBackendOperationalFromEnvV0SoloTmuxV0(t *testing.T) {
	t.Setenv(envCodexGoalBackendV0, codexGoalBackendAppServerProxyV0)
	t.Setenv(envAllowAppServerProxyDiagnosticV0, "1")
	if codexGoalBackendOperationalFromEnvV0() {
		t.Fatalf("app_server_proxy diagnostico no debe contar como backend operativo")
	}

	t.Setenv(envCodexGoalBackendV0, codexGoalBackendAppServerTmuxV0)
	if !codexGoalBackendOperationalFromEnvV0() {
		t.Fatalf("app_server_tmux debe contar como backend operativo")
	}
}

func TestCodexGoalBackendArgsV0NoUsaProxyParaTmuxV0(t *testing.T) {
	if args := codexGoalBackendArgsV0(codexGoalBackendAppServerTmuxV0); len(args) != 0 {
		t.Fatalf("app_server_tmux no debe heredar args de proxy: %v", args)
	}
	if args := codexGoalBackendArgsV0(codexGoalBackendAppServerProxyV0); !reflect.DeepEqual(args, []string{"app-server", "proxy"}) {
		t.Fatalf("app_server_proxy conserva args diagnosticos: %v", args)
	}
}

func TestServerGoalShutdownHooksFromBackendsV0IncluyeAppEIdleV0(t *testing.T) {
	appHook := &fakeServerGoalShutdownHookV0{id: "app"}
	idleHook := &fakeServerGoalShutdownHookV0{id: "idle"}

	hooks := serverGoalShutdownHooksFromBackendsV0(
		serverCodexGoalBackendV0{ShutdownHook: appHook},
		serverCodexGoalBackendV0{ShutdownHook: idleHook},
	)

	if len(hooks) != 2 || hooks[0] != appHook || hooks[1] != idleHook {
		t.Fatalf("hooks=%+v", hooks)
	}
}

func TestServerGoalShutdownHooksFromBackendsV0ConservaIdleSiAppNoTieneHookV0(t *testing.T) {
	idleHook := &fakeServerGoalShutdownHookV0{id: "idle"}

	hooks := serverGoalShutdownHooksFromBackendsV0(
		serverCodexGoalBackendV0{},
		serverCodexGoalBackendV0{ShutdownHook: idleHook},
	)

	if len(hooks) != 1 || hooks[0] != idleHook {
		t.Fatalf("hooks=%+v", hooks)
	}
}

func TestServerGoalShutdownHooksFromBackendsV0DeduplicaMismoHookV0(t *testing.T) {
	hook := &fakeServerGoalShutdownHookV0{id: "shared"}

	hooks := serverGoalShutdownHooksFromBackendsV0(
		serverCodexGoalBackendV0{ShutdownHook: hook},
		serverCodexGoalBackendV0{ShutdownHook: hook},
	)

	if len(hooks) != 1 || hooks[0] != hook {
		t.Fatalf("hooks=%+v", hooks)
	}
}

func TestServerGoalShutdownHooksFromBackendsV0DeduplicaTmuxEquivalenteV0(t *testing.T) {
	hook := serverCodexAppServerTmuxBackendV0{
		CommandPath: "codex",
		SocketPath:  "/tmp/orquesta-goal.sock",
		SessionName: "orquesta-goal-session",
		Timeout:     time.Second,
	}

	hooks := serverGoalShutdownHooksFromBackendsV0(
		serverCodexGoalBackendV0{ShutdownHook: hook},
		serverCodexGoalBackendV0{ShutdownHook: hook},
	)

	if len(hooks) != 1 || hooks[0] != hook {
		t.Fatalf("hooks=%+v", hooks)
	}
}

type fakeServerGoalShutdownHookV0 struct {
	id string
}

func (hook *fakeServerGoalShutdownHookV0) ShutdownV0(context.Context) error {
	return nil
}

func TestServerConfigFromEnvV0ProxyDiagnosticoNoDerivaGoalFirstIdleV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	t.Setenv(envCodexGoalBackendV0, codexGoalBackendAppServerProxyV0)
	t.Setenv(envAllowAppServerProxyDiagnosticV0, "1")
	t.Setenv(envServerIdleSelfImprovementGoalFirstV0, "")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if config.IdleSelfImprovementGoalFirst {
		t.Fatalf("proxy diagnostico no debe derivar goal-first idle: %+v", config)
	}
}

func TestServerCodexGoalBackendsFromEnvV0ProxyDiagnosticoNoCableaAppNiIdleV0(t *testing.T) {
	script := filepath.Join(t.TempDir(), "codex-fake-ok")
	if err := os.WriteFile(script, []byte(fakeCodexAppServerPreflightScriptV0("")), 0o700); err != nil {
		t.Fatalf("write fake codex: %v", err)
	}
	root := t.TempDir()
	appDir := filepath.Join(root, "app-workdir")
	idleDir := filepath.Join(root, "orquesta-idle-workdir")
	t.Setenv(envCodexProjectWorkDirV0, appDir)
	t.Setenv(envServerIdleSelfImprovementProjectWorkDirV0, idleDir)
	t.Setenv(envCodexCommandV0, script)
	t.Setenv(envCodexGoalBackendV0, codexGoalBackendAppServerProxyV0)
	t.Setenv(envAllowAppServerProxyDiagnosticV0, "1")
	t.Setenv(envCodexGoalPreflightTimeoutMSV0, "1000")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	_, err = serverCodexGoalBackendsFromEnvV0(config)
	if err == nil || !strings.Contains(err.Error(), "codex_goal_backend_proxy_diagnostic_not_operational") {
		t.Fatalf("err=%v", err)
	}
}

func TestCodexAppServerRPCPayloadYDecodeV0(t *testing.T) {
	payload, err := codexAppServerRPCPayloadV0("thread/goal/get", map[string]interface{}{"threadId": "thread-ref-003"})
	if err != nil {
		t.Fatalf("payload: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(payload), "\n")
	if len(lines) != 3 {
		t.Fatalf("payload lines=%q", payload)
	}
	var initialized map[string]interface{}
	if err := json.Unmarshal([]byte(lines[1]), &initialized); err != nil {
		t.Fatalf("json initialized: %v", err)
	}
	if initialized["method"] != "initialized" {
		t.Fatalf("initialized=%+v", initialized)
	}
	var request map[string]interface{}
	if err := json.Unmarshal([]byte(lines[2]), &request); err != nil {
		t.Fatalf("json request: %v", err)
	}
	if request["method"] != "thread/goal/get" || int(request["id"].(float64)) != 2 {
		t.Fatalf("request=%+v", request)
	}

	stdout := []byte(`{"method":"thread/goal/updated","params":{"goal":{"threadId":"thread-ref-003","objective":"x","status":"active","createdAt":1,"updatedAt":2,"tokensUsed":3,"timeUsedSeconds":4}}}` + "\n" +
		`{"id":2,"result":{"goal":{"threadId":"thread-ref-003","objective":"x","status":"active","createdAt":1,"updatedAt":2,"tokensUsed":3,"timeUsedSeconds":4}}}` + "\n")
	var response serverCodexAppServerThreadGoalGetResponseV0
	if err := decodeCodexAppServerRPCResponseV0(stdout, 2, &response); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if response.Goal == nil ||
		response.Goal.ThreadID != "thread-ref-003" ||
		response.Goal.Status != "active" {
		t.Fatalf("response=%+v", response)
	}

	stdout = []byte(`{"id":2,"result":{"thread":{"id":"thread-ref-003","status":"closed","turns":[{"id":"turn-ref-003","status":"completed","itemsView":"full","items":[{"id":"item-ref-003","type":"agentMessage","phase":"final_answer","text":"` + orquestaruntimecodexgoal.CodexGoalResultMarkerV0 + ` {\"summary\":\"ok\"}"}]}]}}}` + "\n")
	var threadResponse serverCodexAppServerThreadReadResponseV0
	if err := decodeCodexAppServerRPCResponseV0(stdout, 2, &threadResponse); err != nil {
		t.Fatalf("decode thread/read: %v", err)
	}
	if threadResponse.Thread.ID != "thread-ref-003" ||
		len(threadResponse.Thread.Turns) != 1 ||
		len(threadResponse.Thread.Turns[0].Items) != 1 ||
		threadResponse.Thread.Turns[0].Items[0].Phase != "final_answer" {
		t.Fatalf("thread response=%+v", threadResponse)
	}

	stdout = []byte(`{"id":2,"error":{"code":-32602,"message":"params invalidos"}}` + "\n")
	err = decodeCodexAppServerRPCResponseV0(stdout, 2, &threadResponse)
	var callErr codexAppServerCallErrorV0
	if !errors.As(err, &callErr) || callErr.Code != "codex_app_server_rpc_invalid_params" {
		t.Fatalf("rpc error no estructurado: err=%v callErr=%+v", err, callErr)
	}
}

func TestCodexAppServerCommandProtocolV0RechazaArgsVaciosV0(t *testing.T) {
	protocol := serverCodexAppServerCommandProtocolV0{
		CommandPath: os.Args[0],
		Timeout:     time.Second,
	}
	err := protocol.ProbeV0(context.Background())
	var callErr codexAppServerCallErrorV0
	if !errors.As(err, &callErr) || callErr.Code != "codex_app_server_command_args_required" {
		t.Fatalf("err=%v callErr=%+v", err, callErr)
	}
}

func TestCodexAppServerWebSocketReadResponseV0EstructuraErroresRPCV0(t *testing.T) {
	frame := codexAppServerWebSocketFrameForTestV0(
		[]byte(`{"id":2,"error":{"code":-32601,"message":"method missing"}}`),
	)
	err := codexAppServerWebSocketReadResponseV0(bufio.NewReader(bytes.NewReader(frame)), 2, nil)
	var callErr codexAppServerCallErrorV0
	if !errors.As(err, &callErr) || callErr.Code != "codex_app_server_rpc_method_not_found" {
		t.Fatalf("websocket rpc error no estructurado: err=%v callErr=%+v", err, callErr)
	}
}

func TestServerEffectiveConfigV0ExponeGoalFirstYBackendV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	t.Setenv(envServerIdleSelfImprovementGoalFirstV0, "true")
	t.Setenv(envCodexGoalBackendV0, codexGoalBackendAppServerTmuxV0)
	t.Setenv(envCodexGoalTimeoutMSV0, "12000")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	settings := config.EffectiveConfig.Settings
	if got := effectiveSettingValueForTestV0(settings, envServerIdleSelfImprovementGoalFirstV0); got != "true" {
		t.Fatalf("goal-first setting=%q", got)
	}
	if got := effectiveSettingValueForTestV0(settings, envCodexGoalBackendV0); got != codexGoalBackendAppServerTmuxV0 {
		t.Fatalf("goal backend=%q", got)
	}
	if got := effectiveSettingValueForTestV0(settings, envCodexGoalTimeoutMSV0); got != "12000" {
		t.Fatalf("goal timeout=%q", got)
	}
	if got := effectiveSettingValueForTestV0(settings, envCodexGoalPreflightTimeoutMSV0); got != "3000" {
		t.Fatalf("goal preflight timeout=%q", got)
	}
}

func fakeCodexAppServerPreflightScriptV0(mode string) string {
	argsCheck := ""
	return "#!/bin/sh\n" + argsCheck + "while IFS= read -r line; do\n  case \"$line\" in\n    *'\"id\":1'*) printf '{\"id\":1,\"result\":{}}\\n' ;;\n    *'\"id\":2'*) printf '{\"id\":2,\"result\":{\"data\":[]}}\\n'; exit 0 ;;\n  esac\ndone\nexit 1\n"
}

func codexAppServerWebSocketFrameForTestV0(payload []byte) []byte {
	frame := []byte{0x81}
	switch size := len(payload); {
	case size < 126:
		frame = append(frame, byte(size))
	case size <= 65535:
		frame = append(frame, 126, byte(size>>8), byte(size))
	default:
		panic("test websocket frame too large")
	}
	return append(frame, payload...)
}

type fakeCodexAppServerProtocolV0 struct {
	calls       []string
	startParams serverCodexAppServerThreadStartParamsV0
	setParams   serverCodexAppServerThreadGoalSetParamsV0
	turnParams  serverCodexAppServerTurnStartParamsV0
	getThreadID string

	thread           serverCodexAppServerThreadV0
	goal             serverCodexAppServerThreadGoalV0
	turn             serverCodexAppServerTurnV0
	setGoalErr       error
	getGoalErr       error
	observedGoal     *serverCodexAppServerThreadGoalV0
	readThread       serverCodexAppServerThreadReadV0
	readThreadErr    error
	readThreadID     string
	readIncludeTurns bool
}

func (fake *fakeCodexAppServerProtocolV0) StartThreadV0(
	_ context.Context,
	params serverCodexAppServerThreadStartParamsV0,
) (serverCodexAppServerThreadV0, error) {
	fake.calls = append(fake.calls, "thread/start")
	fake.startParams = params
	return fake.thread, nil
}

func (fake *fakeCodexAppServerProtocolV0) SetGoalV0(
	_ context.Context,
	params serverCodexAppServerThreadGoalSetParamsV0,
) (serverCodexAppServerThreadGoalV0, error) {
	fake.calls = append(fake.calls, "thread/goal/set")
	fake.setParams = params
	if fake.setGoalErr != nil {
		return serverCodexAppServerThreadGoalV0{}, fake.setGoalErr
	}
	return fake.goal, nil
}

func (fake *fakeCodexAppServerProtocolV0) StartTurnV0(
	_ context.Context,
	params serverCodexAppServerTurnStartParamsV0,
) (serverCodexAppServerTurnV0, error) {
	fake.calls = append(fake.calls, "turn/start")
	fake.turnParams = params
	return fake.turn, nil
}

func (fake *fakeCodexAppServerProtocolV0) GetGoalV0(
	_ context.Context,
	threadID string,
) (*serverCodexAppServerThreadGoalV0, error) {
	fake.calls = append(fake.calls, "thread/goal/get")
	fake.getThreadID = threadID
	if fake.getGoalErr != nil {
		return nil, fake.getGoalErr
	}
	return fake.observedGoal, nil
}

func (fake *fakeCodexAppServerProtocolV0) ReadThreadV0(
	_ context.Context,
	threadID string,
	includeTurns bool,
) (serverCodexAppServerThreadReadV0, error) {
	fake.calls = append(fake.calls, "thread/read")
	fake.readThreadID = threadID
	fake.readIncludeTurns = includeTurns
	if fake.readThreadErr != nil {
		return serverCodexAppServerThreadReadV0{}, fake.readThreadErr
	}
	return fake.readThread, nil
}
