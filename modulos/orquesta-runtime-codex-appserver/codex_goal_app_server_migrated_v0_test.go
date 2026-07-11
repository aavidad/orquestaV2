package orquestaruntimecodexappserver

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
	orquestaservershutdown "orquesta/modulos/orquesta-server-shutdown"
)

func TestServerCodexAppServerGoalBackendV0LanzaThreadGoalYTurnMigradoV0(t *testing.T) {
	protocol := &fakeCodexAppServerProtocolV0{
		thread: serverCodexAppServerThreadV0{ID: "thread-ref-goal-001"},
		goal:   serverCodexAppServerThreadGoalV0{ThreadID: "thread-ref-goal-001", Status: "active"},
		turn:   serverCodexAppServerTurnV0{ID: "turn-ref-goal-001", Status: "inProgress"},
	}
	backend := serverCodexAppServerGoalBackendV0{
		Protocol:        protocol,
		Model:           "gpt-test",
		ReasoningEffort: "medium",
		Sandbox:         "workspace-write",
		ApprovalPolicy:  "never",
	}
	packet := orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef:   "goal-ref-001",
		Objective: "hacer una migracion acotada",
	}

	receipt, err := backend.StartCodexGoalV0(context.Background(), packet)
	if err != nil {
		t.Fatalf("StartCodexGoalV0: %v", err)
	}
	if receipt.Status != orquestagoal.GoalStatusRunningV0 ||
		receipt.ExternalGoalRef != "thread-ref-goal-001" ||
		len(protocol.calls) < 4 ||
		!reflect.DeepEqual(protocol.calls[:4], []string{"thread/start", "thread/settings/update", "thread/goal/set", "turn/start"}) {
		t.Fatalf("receipt=%+v calls=%+v", receipt, protocol.calls)
	}
	if protocol.settingsParams != (serverCodexAppServerThreadSettingsUpdateParamsV0{ThreadID: "thread-ref-goal-001", Effort: "medium"}) {
		t.Fatalf("settings params=%+v", protocol.settingsParams)
	}
	if protocol.turnParams.Model != "gpt-test" ||
		protocol.turnParams.Effort != "medium" ||
		protocol.turnParams.ApprovalPolicy != "never" {
		t.Fatalf("turn params=%+v", protocol.turnParams)
	}
}

func TestServerCodexAppServerGoalBackendV0FallaCerradoSiNoActualizaSettingsAntesDeGoalV0(t *testing.T) {
	updateErr := errors.New("settings update unavailable")
	protocol := &fakeCodexAppServerProtocolV0{
		thread:            serverCodexAppServerThreadV0{ID: "thread-ref-settings-fail-001"},
		updateSettingsErr: updateErr,
	}
	backend := serverCodexAppServerGoalBackendV0{
		Protocol:        protocol,
		ReasoningEffort: "medium",
		Sandbox:         "workspace-write",
	}

	receipt, err := backend.StartCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef: "goal-ref-settings-fail-001",
	})
	if !errors.Is(err, updateErr) || receipt.IssueCode != "codex_app_server_thread_settings_update_failed" ||
		!reflect.DeepEqual(protocol.calls, []string{"thread/start", "thread/settings/update"}) {
		t.Fatalf("receipt=%+v err=%v calls=%+v", receipt, err, protocol.calls)
	}
}

func TestCodexAppServerThreadSettingsUpdateParamsV0SerializaExactamenteV0(t *testing.T) {
	params := serverCodexAppServerThreadSettingsUpdateParamsV0{
		ThreadID: " thread-settings-json-001 ",
		Effort:   " medium ",
	}
	if got := params.toJSONV0(); !reflect.DeepEqual(got, map[string]interface{}{
		"threadId": "thread-settings-json-001",
		"effort":   "medium",
	}) {
		t.Fatalf("json=%#v", got)
	}
}

func TestServerCodexAppServerGoalBackendV0TurnStartInyectaContratoSalidaCompactaV0(t *testing.T) {
	protocol := &fakeCodexAppServerProtocolV0{
		thread: serverCodexAppServerThreadV0{ID: "thread-ref-goal-compacto-001"},
		goal:   serverCodexAppServerThreadGoalV0{ThreadID: "thread-ref-goal-compacto-001", Status: "active"},
		turn:   serverCodexAppServerTurnV0{ID: "turn-ref-goal-compacto-001", Status: "inProgress"},
	}
	backend := serverCodexAppServerGoalBackendV0{
		Protocol:       protocol,
		Sandbox:        "workspace-write",
		ApprovalPolicy: "never",
	}
	packet := orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef:   "goal-ref-compacto-001",
		Objective: "hacer una migracion acotada",
		Prompt:    "prompt operativo minimo",
		DirectionContract: orquestaruntimecodexgoal.CodexGoalDirectionContractV0{
			RequireEarlyCheckpoint: true,
			EarlyCheckpointFile:    "checkpoint_started.txt",
			ToolOutputPolicy: orquestaruntimecodexgoal.CodexGoalToolOutputPolicyV0{
				MaxTextBytes:           orquestaruntimecodexgoal.CodexGoalToolOutputMaxBytesV0,
				RequireBoundedCommands: true,
				BoundedCommandHints:    []string{"rg --max-count", "sed -n"},
			},
		},
	}

	receipt, err := backend.StartCodexGoalV0(context.Background(), packet)
	if err != nil {
		t.Fatalf("StartCodexGoalV0: %v", err)
	}
	input := protocol.turnParams.InputText
	if receipt.Status != orquestagoal.GoalStatusRunningV0 ||
		!containsStringMigratedTestV0(protocol.calls, "turn/start") ||
		!containsStringMigratedTestV0(receipt.EvidenceRefs, codexAppServerTurnStartToolOutputPolicySentV0) ||
		!containsStringMigratedTestV0(receipt.EvidenceRefs, codexAppServerTurnStartToolOutputPolicyAcceptedV0) ||
		containsStringMigratedTestV0(receipt.EvidenceRefs, codexAppServerTurnStartToolOutputPolicyFallbackV0) ||
		!strings.Contains(input, "prompt operativo minimo") ||
		!strings.Contains(input, codexAppServerTurnStartRuntimeContractHeaderV0) ||
		!strings.Contains(input, "checkpoint_started.txt") ||
		!strings.Contains(input, "max_text_bytes=16384") ||
		!strings.Contains(input, "thread_read_max_bytes=256 KiB") ||
		!strings.Contains(input, "rg --max-count") ||
		!strings.Contains(input, orquestaruntimecodexgoal.CodexGoalResultMarkerV0) {
		t.Fatalf("receipt=%+v input=%q calls=%+v", receipt, input, protocol.calls)
	}
	if strings.Contains(protocol.startParams.CWD, "prompt operativo minimo") ||
		strings.Contains(protocol.startParams.Model, "prompt operativo minimo") ||
		strings.Contains(protocol.startParams.Sandbox, "prompt operativo minimo") {
		t.Fatalf("thread/start no debe transportar prompt en campos de arranque: %+v", protocol.startParams)
	}
	policy := protocol.turnParams.ToolOutputPolicy
	if policy.MaxTextBytes != orquestaruntimecodexgoal.CodexGoalToolOutputMaxBytesV0 ||
		policy.ThreadReadMaxBytes != codexAppServerThreadReadMaxResponseFrameBytesV0 ||
		!policy.RequireBoundedCommands ||
		!policy.DurableEvidenceRequired ||
		!containsStringMigratedTestV0(policy.BoundedCommandHints, "rg --max-count") ||
		!containsStringMigratedTestV0(policy.BoundedCommandHints, "sed -n") {
		t.Fatalf("tool output policy turn/start=%+v", policy)
	}
}

func TestServerCodexAppServerGoalBackendV0TurnStartNoRelajaContratoSalidaCompactaV0(t *testing.T) {
	protocol := &fakeCodexAppServerProtocolV0{
		thread: serverCodexAppServerThreadV0{ID: "thread-ref-goal-compacto-flojo-001"},
		goal:   serverCodexAppServerThreadGoalV0{ThreadID: "thread-ref-goal-compacto-flojo-001", Status: "active"},
		turn:   serverCodexAppServerTurnV0{ID: "turn-ref-goal-compacto-flojo-001", Status: "inProgress"},
	}
	backend := serverCodexAppServerGoalBackendV0{
		Protocol:       protocol,
		Sandbox:        "workspace-write",
		ApprovalPolicy: "never",
	}
	packet := orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef:   "goal-ref-compacto-flojo-001",
		Objective: "probar que appserver no relaja contrato de salidas",
		Prompt:    "prompt operativo con contrato debil",
		DirectionContract: orquestaruntimecodexgoal.CodexGoalDirectionContractV0{
			ToolOutputPolicy: orquestaruntimecodexgoal.CodexGoalToolOutputPolicyV0{
				MaxTextBytes:        orquestaruntimecodexgoal.CodexGoalToolOutputMaxBytesV0 * 8,
				BoundedCommandHints: []string{"custom bounded helper"},
			},
		},
	}

	receipt, err := backend.StartCodexGoalV0(context.Background(), packet)
	if err != nil {
		t.Fatalf("StartCodexGoalV0: %v", err)
	}
	input := protocol.turnParams.InputText
	if receipt.Status != orquestagoal.GoalStatusRunningV0 ||
		!strings.Contains(input, codexAppServerTurnStartRuntimeContractHeaderV0) ||
		!strings.Contains(input, "checkpoint_started.txt") ||
		!strings.Contains(input, "max_text_bytes=16384") ||
		strings.Contains(input, "max_text_bytes=131072") ||
		!strings.Contains(input, "rg --max-count") ||
		!strings.Contains(input, "rg --files | head") ||
		!strings.Contains(input, "custom bounded helper") {
		t.Fatalf("contrato runtime relajado: receipt=%+v input=%q", receipt, input)
	}
	policy := protocol.turnParams.ToolOutputPolicy
	if policy.MaxTextBytes != orquestaruntimecodexgoal.CodexGoalToolOutputMaxBytesV0 ||
		!containsStringMigratedTestV0(policy.BoundedCommandHints, "rg --files | head") ||
		!containsStringMigratedTestV0(policy.BoundedCommandHints, "custom bounded helper") {
		t.Fatalf("tool output policy relajado: %+v", policy)
	}
}

func TestServerCodexAppServerTurnStartParamsV0SerializaToolOutputPolicyV0(t *testing.T) {
	params := serverCodexAppServerTurnStartParamsV0{
		ThreadID:  "thread-policy-json-001",
		InputText: "turn acotado",
		ToolOutputPolicy: serverCodexAppServerTurnStartToolOutputPolicyV0{
			MaxTextBytes:            4096,
			ThreadReadMaxBytes:      codexAppServerThreadReadMaxResponseFrameBytesV0,
			RequireBoundedCommands:  true,
			BoundedCommandHints:     []string{"rg --max-count", "rg --max-count", "sed -n"},
			DurableEvidenceRequired: true,
		},
	}

	raw := params.toJSONV0()
	policy, ok := raw["toolOutputPolicy"].(map[string]interface{})
	if !ok {
		t.Fatalf("toolOutputPolicy no serializado: %#v", raw)
	}
	if policy["maxTextBytes"] != 4096 ||
		policy["threadReadMaxBytes"] != codexAppServerThreadReadMaxResponseFrameBytesV0 ||
		policy["requireBoundedCommands"] != true ||
		policy["durableEvidenceRequired"] != true {
		t.Fatalf("policy json=%#v", policy)
	}
	hints, ok := policy["boundedCommandHints"].([]string)
	if !ok ||
		len(hints) != 2 ||
		!containsStringMigratedTestV0(hints, "rg --max-count") ||
		!containsStringMigratedTestV0(hints, "sed -n") {
		t.Fatalf("hints=%#v policy=%#v", policy["boundedCommandHints"], policy)
	}

	params.DisablePolicyJSON = true
	if _, exists := params.toJSONV0()["toolOutputPolicy"]; exists {
		t.Fatalf("DisablePolicyJSON debe omitir toolOutputPolicy")
	}
}

func TestServerCodexAppServerGoalBackendV0TurnStartToolOutputPolicyFallbackCompatibleV0(t *testing.T) {
	protocol := &fakeCodexAppServerProtocolV0{
		thread: serverCodexAppServerThreadV0{ID: "thread-ref-policy-fallback-001"},
		goal:   serverCodexAppServerThreadGoalV0{ThreadID: "thread-ref-policy-fallback-001", Status: "active"},
		turn:   serverCodexAppServerTurnV0{ID: "turn-ref-policy-fallback-001", Status: "inProgress"},
		startTurnErrs: []error{
			codexAppServerCallErrorV0{
				Code: "codex_app_server_rpc_invalid_params",
				Err:  errors.New("unknown field toolOutputPolicy"),
			},
			nil,
		},
	}
	backend := serverCodexAppServerGoalBackendV0{
		Protocol:       protocol,
		Sandbox:        "workspace-write",
		ApprovalPolicy: "never",
	}
	packet := orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef:   "goal-ref-policy-fallback-001",
		Objective: "lanzar goal con app-server legacy",
		DirectionContract: orquestaruntimecodexgoal.CodexGoalDirectionContractV0{
			ToolOutputPolicy: orquestaruntimecodexgoal.CodexGoalToolOutputPolicyV0{
				MaxTextBytes:        2048,
				BoundedCommandHints: []string{"custom head helper"},
			},
		},
	}

	receipt, err := backend.StartCodexGoalV0(context.Background(), packet)
	if err != nil {
		t.Fatalf("StartCodexGoalV0 receipt=%+v err=%v", receipt, err)
	}
	if receipt.Status != orquestagoal.GoalStatusRunningV0 ||
		!containsStringMigratedTestV0(receipt.EvidenceRefs, codexAppServerTurnStartToolOutputPolicySentV0) ||
		!containsStringMigratedTestV0(receipt.EvidenceRefs, codexAppServerTurnStartToolOutputPolicyFallbackV0) ||
		containsStringMigratedTestV0(receipt.EvidenceRefs, codexAppServerTurnStartToolOutputPolicyAcceptedV0) {
		t.Fatalf("receipt=%+v", receipt)
	}
	if len(protocol.turnParamsHistory) != 2 {
		t.Fatalf("turn starts=%d calls=%+v", len(protocol.turnParamsHistory), protocol.calls)
	}
	first := protocol.turnParamsHistory[0]
	second := protocol.turnParamsHistory[1]
	if first.DisablePolicyJSON || first.ToolOutputPolicy.emptyV0() {
		t.Fatalf("primer turn/start debe enviar policy estructurada: %+v", first)
	}
	if !second.DisablePolicyJSON || second.ToolOutputPolicy.emptyV0() {
		t.Fatalf("retry debe conservar contrato textual y omitir policy JSON: %+v", second)
	}
	if !strings.Contains(second.InputText, "max_text_bytes=2048") ||
		!strings.Contains(second.InputText, "custom head helper") {
		t.Fatalf("retry perdio contrato textual: %q", second.InputText)
	}
}

func TestServerCodexAppServerGoalBackendV0TurnStartToolOutputPolicyFallbackSoloSchemaV0(t *testing.T) {
	protocol := &fakeCodexAppServerProtocolV0{
		thread: serverCodexAppServerThreadV0{ID: "thread-ref-policy-no-fallback-001"},
		goal:   serverCodexAppServerThreadGoalV0{ThreadID: "thread-ref-policy-no-fallback-001", Status: "active"},
		turn:   serverCodexAppServerTurnV0{ID: "turn-ref-policy-no-fallback-001", Status: "inProgress"},
		startTurnErrs: []error{
			codexAppServerCallErrorV0{
				Code: "codex_app_server_rpc_internal",
				Err:  errors.New("toolOutputPolicy runtime failure"),
			},
		},
	}
	backend := serverCodexAppServerGoalBackendV0{
		Protocol:       protocol,
		Sandbox:        "workspace-write",
		ApprovalPolicy: "never",
	}
	packet := orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef:   "goal-ref-policy-no-fallback-001",
		Objective: "no relajar policy ante error no schema",
		DirectionContract: orquestaruntimecodexgoal.CodexGoalDirectionContractV0{
			ToolOutputPolicy: orquestaruntimecodexgoal.CodexGoalToolOutputPolicyV0{
				MaxTextBytes:           2048,
				RequireBoundedCommands: true,
			},
		},
	}

	receipt, err := backend.StartCodexGoalV0(context.Background(), packet)
	if err == nil {
		t.Fatalf("StartCodexGoalV0 debe fallar sin fallback: receipt=%+v", receipt)
	}
	if len(protocol.turnParamsHistory) != 1 {
		t.Fatalf("fallback inesperado: turn starts=%d calls=%+v", len(protocol.turnParamsHistory), protocol.calls)
	}
	first := protocol.turnParamsHistory[0]
	if first.DisablePolicyJSON || first.ToolOutputPolicy.emptyV0() {
		t.Fatalf("primer turn/start debe conservar policy estructurada: %+v", first)
	}
}

func TestServerCodexAppServerGoalBackendV0MaterializaCheckpointAntesDeTurnStartV0(t *testing.T) {
	root := t.TempDir()
	checkpointPath := filepath.Join(root, ".orquesta-runtime", "goal-receipts", "goal-ref-checkpoint-preturn-001", "checkpoint_started.txt")
	protocol := &fakeCodexAppServerProtocolV0{
		thread: serverCodexAppServerThreadV0{ID: "thread-ref-goal-checkpoint-preturn-001"},
		goal:   serverCodexAppServerThreadGoalV0{ThreadID: "thread-ref-goal-checkpoint-preturn-001", Status: "active"},
		turn:   serverCodexAppServerTurnV0{ID: "turn-ref-goal-checkpoint-preturn-001", Status: "inProgress"},
		onStartTurn: func() {
			if _, err := os.Stat(checkpointPath); err != nil {
				t.Fatalf("checkpoint no existe antes de turn/start: %v", err)
			}
		},
	}
	backend := serverCodexAppServerGoalBackendV0{
		Protocol:       protocol,
		CWD:            root,
		Sandbox:        "workspace-write",
		ApprovalPolicy: "never",
	}
	packet := orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef:   "goal-ref-checkpoint-preturn-001",
		Objective: "materializar checkpoint antes de usar herramientas",
		WriteSet: []orquestagoal.GoalWriteScopeV0{{
			Path: "generated-apps",
		}},
		DirectionContract: orquestaruntimecodexgoal.CodexGoalDirectionContractV0{
			RequireEarlyCheckpoint: true,
			EarlyCheckpointFile:    "checkpoint_started.txt",
		},
	}

	receipt, err := backend.StartCodexGoalV0(context.Background(), packet)
	if err != nil {
		t.Fatalf("StartCodexGoalV0: %v", err)
	}
	data, readErr := os.ReadFile(checkpointPath)
	if readErr != nil {
		t.Fatalf("read checkpoint: %v", readErr)
	}
	body := string(data)
	if receipt.Status != orquestagoal.GoalStatusRunningV0 ||
		!containsStringMigratedTestV0(protocol.calls, "turn/start") ||
		!containsStringPrefixMigratedTestV0(receipt.EvidenceRefs, "evidence-ref-codex-app-server-early-checkpoint-materialized:.orquesta-runtime/goal-receipts/goal-ref-checkpoint-preturn-001/checkpoint_started.txt") ||
		!strings.Contains(body, "schema_version=orquesta.codex_app_server.early_checkpoint.v0") ||
		!strings.Contains(body, "goal_ref=goal-ref-checkpoint-preturn-001") ||
		!strings.Contains(body, "external_goal_ref=thread-ref-goal-checkpoint-preturn-001") {
		t.Fatalf("receipt=%+v body=%q calls=%+v", receipt, body, protocol.calls)
	}
}

func TestServerCodexAppServerGoalBackendV0ObservaCheckpointStartedComoReasonCodeV0(t *testing.T) {
	root := t.TempDir()
	checkpointPath := filepath.Join(root, ".orquesta-runtime", "goal-receipts", "goal-ref-checkpoint-observe-001", "checkpoint_started.txt")
	protocol := &fakeCodexAppServerProtocolV0{
		observedGoal: &serverCodexAppServerThreadGoalV0{
			ThreadID: "thread-ref-goal-checkpoint-observe-001",
			Status:   "active",
		},
		readThread: serverCodexAppServerThreadReadV0{
			ID:     "thread-ref-goal-checkpoint-observe-001",
			Status: "running",
		},
	}
	backend := serverCodexAppServerGoalBackendV0{
		Protocol: protocol,
		CWD:      root,
	}
	packet := orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef: "goal-ref-checkpoint-observe-001",
		WriteSet: []orquestagoal.GoalWriteScopeV0{{
			Path: "generated-apps",
		}},
		DirectionContract: orquestaruntimecodexgoal.CodexGoalDirectionContractV0{
			RequireEarlyCheckpoint: true,
			EarlyCheckpointFile:    "checkpoint_started.txt",
		},
	}
	body := codexAppServerEarlyCheckpointBodyV0(packet, "thread-ref-goal-checkpoint-observe-001")
	if err := os.MkdirAll(filepath.Dir(checkpointPath), 0o700); err != nil {
		t.Fatalf("mkdir checkpoint: %v", err)
	}
	if err := os.WriteFile(checkpointPath, []byte(body), 0o600); err != nil {
		t.Fatalf("write checkpoint: %v", err)
	}

	receipt, err := backend.ObserveCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		GoalRef:         "goal-ref-checkpoint-observe-001",
		ExternalGoalRef: "thread-ref-goal-checkpoint-observe-001",
	})
	if err != nil {
		t.Fatalf("ObserveCodexGoalV0: %v", err)
	}
	if receipt.Status != orquestagoal.GoalStatusRunningV0 ||
		receipt.IssueCode != codexAppServerGoalResultCheckpointReasonCodeV0 ||
		!containsStringMigratedTestV0(receipt.ArtifactPaths, ".orquesta-runtime/goal-receipts/goal-ref-checkpoint-observe-001/checkpoint_started.txt") ||
		!containsStringMigratedTestV0(receipt.EvidenceRefs, codexAppServerGoalResultCheckpointEvidenceRefV0) {
		t.Fatalf("receipt=%+v", receipt)
	}
}

func TestCodexAppServerGoalResultFromWorkspaceV0PrefiereReciboRuntimeV0(t *testing.T) {
	root := t.TempDir()
	goalRef := "goal-ref-runtime-receipt-priority-001"
	externalGoalRef := "thread-ref-runtime-receipt-priority-001"
	runtimeDir := orquestaruntimecodexgoal.CodexGoalRuntimeReceiptRelativeDirV0(goalRef)
	writeCodexAppServerGoalResultForTestV0(t, root, runtimeDir, goalRef, externalGoalRef)
	writeCodexAppServerGoalResultForTestV0(t, root, "legacy-write-set", goalRef, externalGoalRef)

	result, ok, err := codexAppServerGoalResultFromWorkspaceV0(root, goalRef, externalGoalRef)
	if err != nil || !ok ||
		!containsStringMigratedTestV0(result.ArtifactPaths, runtimeDir+"/ok.md") ||
		containsStringMigratedTestV0(result.ArtifactPaths, "legacy-write-set/ok.md") ||
		containsStringMigratedTestV0(result.ArtifactPaths, runtimeDir+"/"+orquestaruntimecodexgoal.CodexGoalResultFileNameForGoalRefV0(goalRef)) {
		t.Fatalf("result=%+v ok=%v err=%v", result, ok, err)
	}
}

func TestServerCodexAppServerGoalBackendV0StopForcedBloqueaGoalYApagaBackendV0(t *testing.T) {
	protocol := &fakeCodexAppServerProtocolV0{
		goal: serverCodexAppServerThreadGoalV0{ThreadID: "thread-ref-stop-forced-001", Status: "blocked"},
	}
	shutdown := &fakeCodexAppServerBackendShutdownV0{}
	backend := serverCodexAppServerGoalBackendV0{
		Protocol:        protocol,
		BackendShutdown: shutdown,
	}

	result, err := backend.StopCodexGoalV0(context.Background(), CodexGoalStopRequestV0{
		GoalRef:         "goal-ref-stop-forced-001",
		ExternalGoalRef: "thread-ref-stop-forced-001",
		Action:          "stop",
		Forced:          true,
		EvidenceRefs:    []string{"evidence-ref-operator-forced-stop"},
	})
	if err != nil {
		t.Fatalf("StopCodexGoalV0: %v result=%+v", err, result)
	}
	if !result.GoalStatusSet ||
		!result.BackendStopped ||
		result.Status != orquestagoal.GoalStatusBlockedV0 ||
		protocol.setParams.ThreadID != "thread-ref-stop-forced-001" ||
		protocol.setParams.Status != "blocked" ||
		shutdown.calls != 0 ||
		shutdown.forcedCalls != 1 ||
		!containsStringMigratedTestV0(result.EvidenceRefs, codexAppServerGoalForcedStopSetEvidenceV0) ||
		!containsStringMigratedTestV0(result.EvidenceRefs, codexAppServerGoalForcedStopTmuxStoppedV0) {
		t.Fatalf("result=%+v set=%+v shutdown_calls=%d forced_calls=%d", result, protocol.setParams, shutdown.calls, shutdown.forcedCalls)
	}
}

func TestCodexAppServerTmuxBackendV0ForcedStopPreservaProcesoSinIdentidadPersistidaV0(t *testing.T) {
	root := shortUnixSocketTestRootV0(t)
	runtimeDir := filepath.Join(root, "runtime")
	socketPath := filepath.Join(runtimeDir, codexAppServerTmuxDirV0, "s.sock")
	if err := os.MkdirAll(filepath.Dir(socketPath), 0o700); err != nil {
		t.Fatalf("mkdir socket dir: %v", err)
	}
	listener := listenUnixForGenerationTestV0(t, socketPath)
	defer listener.Close()
	backend := serverCodexAppServerTmuxBackendV0{
		SocketPath:     socketPath,
		SessionName:    "orquesta-goal-cooperative-1234567890",
		RuntimeWorkDir: runtimeDir,
		Timeout:        time.Second,
	}
	assertGenerationConflictTestV0(t, backend.ShutdownForcedStopV0(context.Background()))
	if !codexAppServerTmuxSocketListenerAliveV0(socketPath) {
		t.Fatal("forced stop ambiguo termino el listener sin identidad persistida")
	}
}

func TestServerCodexAppServerGoalBackendV0ObservaResultadoMarcadoMigradoV0(t *testing.T) {
	protocol := &fakeCodexAppServerProtocolV0{
		observedGoal: &serverCodexAppServerThreadGoalV0{
			ThreadID: "thread-ref-goal-002",
			Status:   "complete",
		},
		readThread: serverCodexAppServerThreadReadV0{
			ID:     "thread-ref-goal-002",
			Status: "closed",
			Turns: []serverCodexAppServerReadTurnV0{{
				ID: "turn-ref-goal-002",
				Items: []serverCodexAppServerReadItemV0{{
					Type:  "agentMessage",
					Phase: "final_answer",
					Text:  orquestaruntimecodexgoal.CodexGoalResultMarkerV0 + ` {"schema_version":"orquesta_goal_work_result.v0","status":"complete","summary":"ok","goal_ref":"goal-ref-002","external_goal_ref":"thread-ref-goal-002"}`,
				}},
			}},
		},
	}
	backend := serverCodexAppServerGoalBackendV0{Protocol: protocol}

	receipt, err := backend.ObserveCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		GoalRef:         "goal-ref-002",
		ExternalGoalRef: "thread-ref-goal-002",
	})
	if err != nil {
		t.Fatalf("ObserveCodexGoalV0: %v", err)
	}
	if receipt.Status != orquestagoal.GoalStatusCompleteV0 || receipt.Summary != "ok" {
		t.Fatalf("receipt=%+v", receipt)
	}
}

func TestServerCodexAppServerGoalBackendV0NormalizaUnTypoDeGoalRefLigadoAlThreadV0(t *testing.T) {
	expectedGoalRef := "goal-ref-task-autoprogramming-c0ae86a2ae2e-g01"
	protocol := &fakeCodexAppServerProtocolV0{
		observedGoal: &serverCodexAppServerThreadGoalV0{ThreadID: "thread-ref-goal-typo-001", Status: "complete"},
		readThread: serverCodexAppServerThreadReadV0{
			ID: "thread-ref-goal-typo-001", Status: "closed",
			Turns: []serverCodexAppServerReadTurnV0{{
				ID: "turn-ref-goal-typo-001",
				Items: []serverCodexAppServerReadItemV0{{
					Type: "agentMessage", Phase: "final_answer",
					Text: orquestaruntimecodexgoal.CodexGoalResultMarkerV0 + ` {"schema_version":"orquesta_goal_result.v0","status":"complete","summary":"ok","goal_ref":"goal-ref-task-autoprogramming-a0ae86a2ae2e-g01","external_goal_ref":"thread-ref-goal-typo-001","evidence_refs":["evidence-ref-work"]}`,
				}},
			}},
		},
	}
	backend := serverCodexAppServerGoalBackendV0{Protocol: protocol}
	receipt, err := backend.ObserveCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		GoalRef: expectedGoalRef, ExternalGoalRef: "thread-ref-goal-typo-001",
	})
	if err != nil || receipt.Status != orquestagoal.GoalStatusCompleteV0 || receipt.GoalRef != expectedGoalRef ||
		!containsStringMigratedTestV0(receipt.EvidenceRefs, codexAppServerGoalResultGoalRefNormalizedEvidenceV0) {
		t.Fatalf("receipt=%+v err=%v", receipt, err)
	}
	if !codexAppServerGoalResultMarkerGoalRefMismatchV0(codexAppServerGoalResultMarkerV0{GoalRef: "goal-ref-unrelated-999"}, expectedGoalRef) {
		t.Fatal("unrelated goal_ref must remain rejected")
	}
}

func TestServerCodexAppServerGoalBackendV0NormalizaMissingRefsRaizEnChecklistV0(t *testing.T) {
	protocol := &fakeCodexAppServerProtocolV0{
		observedGoal: &serverCodexAppServerThreadGoalV0{
			ThreadID: "thread-ref-goal-missing-refs-001",
			Status:   "complete",
		},
		readThread: serverCodexAppServerThreadReadV0{
			ID:     "thread-ref-goal-missing-refs-001",
			Status: "closed",
			Turns: []serverCodexAppServerReadTurnV0{{
				ID: "turn-ref-goal-missing-refs-001",
				Items: []serverCodexAppServerReadItemV0{{
					Type:  "agentMessage",
					Phase: "final_answer",
					Text: orquestaruntimecodexgoal.CodexGoalResultMarkerV0 + ` {
						"schema_version":"orquesta_goal_result.v0",
						"status":"invalid",
						"goal_ref":"goal-ref-missing-refs-001",
						"external_goal_ref":"thread-ref-goal-missing-refs-001",
						"missing_refs":["source_tree","handoff_report","technical_stack_manifest","go_app","tests"],
						"evidence_refs":["evidence-ref-app-director-goal-first-v0"]
					}`,
				}},
			}},
		},
	}
	backend := serverCodexAppServerGoalBackendV0{Protocol: protocol}

	receipt, err := backend.ObserveCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		GoalRef:         "goal-ref-missing-refs-001",
		ExternalGoalRef: "thread-ref-goal-missing-refs-001",
	})
	if err != nil {
		t.Fatalf("ObserveCodexGoalV0: %v", err)
	}
	if receipt.Status != orquestagoal.GoalStatusInvalidV0 ||
		!containsStringMigratedTestV0(receipt.Checklist.MissingRefs, "source_tree") ||
		!containsStringMigratedTestV0(receipt.Checklist.MissingRefs, "tests") {
		t.Fatalf("receipt=%+v", receipt)
	}
}

func TestServerCodexAppServerGoalBackendV0UmbralUsoAltoConfigurableV0(t *testing.T) {
	protocol := &fakeCodexAppServerProtocolV0{
		observedGoal: &serverCodexAppServerThreadGoalV0{
			ThreadID:        "thread-ref-goal-high-usage-001",
			Status:          "active",
			TokensUsed:      42,
			TimeUsedSeconds: 7,
		},
		readThread: serverCodexAppServerThreadReadV0{
			ID:     "thread-ref-goal-high-usage-001",
			Status: "running",
		},
	}
	backend := serverCodexAppServerGoalBackendV0{
		Protocol:                protocol,
		HighTokenUsageThreshold: 10,
	}

	receipt, err := backend.ObserveCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		GoalRef:         "goal-ref-high-usage-001",
		ExternalGoalRef: "thread-ref-goal-high-usage-001",
	})
	if err != nil {
		t.Fatalf("ObserveCodexGoalV0: %v", err)
	}
	if receipt.Status != orquestagoal.GoalStatusRunningV0 ||
		!strings.Contains(receipt.Summary, "codex_app_server_goal_status_active_high_token_usage") ||
		!strings.Contains(receipt.Summary, "tokens_used=42") ||
		!containsStringMigratedTestV0(receipt.EvidenceRefs, "evidence-ref-codex-app-server-goal-high-token-usage") {
		t.Fatalf("receipt=%+v", receipt)
	}
}

func TestServerCodexAppServerGoalBackendV0ObservaThreadReadGiganteComoBloqueoV0(t *testing.T) {
	protocol := &fakeCodexAppServerProtocolV0{
		observedGoal: &serverCodexAppServerThreadGoalV0{
			ThreadID: "thread-ref-goal-thread-read-big-001",
			Status:   "active",
		},
		readThreadErr: codexAppServerCallErrorV0{
			Code: codexAppServerThreadReadFrameTooLargeIssueCodeV0,
			Err:  errors.New("bytes=262145 limit=262144"),
		},
	}
	backend := serverCodexAppServerGoalBackendV0{Protocol: protocol}

	receipt, err := backend.ObserveCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		GoalRef:         "goal-ref-thread-read-big-001",
		ExternalGoalRef: "thread-ref-goal-thread-read-big-001",
	})
	if err != nil {
		t.Fatalf("ObserveCodexGoalV0: %v", err)
	}
	if receipt.Status != orquestagoal.GoalStatusBlockedV0 ||
		receipt.IssueCode != codexAppServerThreadReadFrameTooLargeIssueCodeV0 ||
		receipt.Summary != codexAppServerThreadReadFrameTooLargeIssueCodeV0 ||
		!containsStringMigratedTestV0(receipt.EvidenceRefs, "evidence-ref-codex-app-server-active-goal-thread-read-failed") {
		t.Fatalf("receipt=%+v", receipt)
	}
}

func TestServerCodexAppServerGoalBackendV0PolicyAceptadaYThreadReadGiganteBloqueaV0(t *testing.T) {
	protocol := &fakeCodexAppServerProtocolV0{
		thread: serverCodexAppServerThreadV0{ID: "thread-ref-policy-big-read-001"},
		goal:   serverCodexAppServerThreadGoalV0{ThreadID: "thread-ref-policy-big-read-001", Status: "active"},
		turn:   serverCodexAppServerTurnV0{ID: "turn-ref-policy-big-read-001", Status: "inProgress"},
	}
	backend := serverCodexAppServerGoalBackendV0{
		Protocol:       protocol,
		Sandbox:        "workspace-write",
		ApprovalPolicy: "never",
	}
	packet := orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef:   "goal-ref-policy-big-read-001",
		Objective: "probar salida gigante sin relajar policy",
		DirectionContract: orquestaruntimecodexgoal.CodexGoalDirectionContractV0{
			ToolOutputPolicy: orquestaruntimecodexgoal.CodexGoalToolOutputPolicyV0{
				MaxTextBytes:           2048,
				RequireBoundedCommands: true,
				BoundedCommandHints:    []string{"custom bounded helper"},
			},
		},
	}

	startReceipt, err := backend.StartCodexGoalV0(context.Background(), packet)
	if err != nil {
		t.Fatalf("StartCodexGoalV0: %v", err)
	}
	if startReceipt.Status != orquestagoal.GoalStatusRunningV0 ||
		!containsStringMigratedTestV0(startReceipt.EvidenceRefs, codexAppServerTurnStartToolOutputPolicySentV0) ||
		!containsStringMigratedTestV0(startReceipt.EvidenceRefs, codexAppServerTurnStartToolOutputPolicyAcceptedV0) ||
		containsStringMigratedTestV0(startReceipt.EvidenceRefs, codexAppServerTurnStartToolOutputPolicyFallbackV0) {
		t.Fatalf("start receipt=%+v", startReceipt)
	}
	if protocol.turnParams.ToolOutputPolicy.ThreadReadMaxBytes != codexAppServerThreadReadMaxResponseFrameBytesV0 ||
		protocol.turnParams.ToolOutputPolicy.MaxTextBytes != 2048 ||
		!protocol.turnParams.ToolOutputPolicy.RequireBoundedCommands {
		t.Fatalf("policy=%+v", protocol.turnParams.ToolOutputPolicy)
	}

	protocol.observedGoal = &serverCodexAppServerThreadGoalV0{
		ThreadID: startReceipt.ExternalGoalRef,
		Status:   "active",
	}
	protocol.readThreadErr = codexAppServerCallErrorV0{
		Code: codexAppServerThreadReadFrameTooLargeIssueCodeV0,
		Err:  errors.New("bytes=262145 limit=262144"),
	}

	observeReceipt, err := backend.ObserveCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		GoalRef:         packet.GoalRef,
		ExternalGoalRef: startReceipt.ExternalGoalRef,
	})
	if err != nil {
		t.Fatalf("ObserveCodexGoalV0: %v", err)
	}
	if observeReceipt.Status != orquestagoal.GoalStatusBlockedV0 ||
		observeReceipt.IssueCode != codexAppServerThreadReadFrameTooLargeIssueCodeV0 ||
		observeReceipt.Summary != codexAppServerThreadReadFrameTooLargeIssueCodeV0 ||
		!containsStringMigratedTestV0(observeReceipt.EvidenceRefs, "evidence-ref-codex-app-server-active-goal-thread-read-failed") ||
		!containsStringMigratedTestV0(observeReceipt.EvidenceRefs, "evidence-ref-codex-app-server-thread-read-response-too-large") {
		t.Fatalf("observe receipt=%+v", observeReceipt)
	}
}

func TestServerCodexAppServerGoalBackendV0ToolOutputPolicyYThreadReadGigantePorWebSocketDeterministaV0(t *testing.T) {
	socketPath, records := startCodexAppServerWebSocketScriptForTestV0(t)
	protocol := serverCodexAppServerWebSocketProtocolV0{
		SocketPath: socketPath,
		Timeout:    2 * time.Second,
	}
	backend := serverCodexAppServerGoalBackendV0{
		Protocol:        protocol,
		ReasoningEffort: "medium",
		Sandbox:         "workspace-write",
		ApprovalPolicy:  "never",
	}
	packet := orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef:   "goal-ref-websocket-policy-big-read-001",
		Objective: "validar protocolo sin agente real",
		DirectionContract: orquestaruntimecodexgoal.CodexGoalDirectionContractV0{
			ToolOutputPolicy: orquestaruntimecodexgoal.CodexGoalToolOutputPolicyV0{
				MaxTextBytes:           2048,
				RequireBoundedCommands: true,
				BoundedCommandHints:    []string{"usar comandos acotados"},
			},
		},
	}

	startReceipt, err := backend.StartCodexGoalV0(context.Background(), packet)
	if err != nil {
		t.Fatalf("StartCodexGoalV0: %v", err)
	}
	if startReceipt.Status != orquestagoal.GoalStatusRunningV0 ||
		!containsStringMigratedTestV0(startReceipt.EvidenceRefs, codexAppServerTurnStartToolOutputPolicySentV0) ||
		!containsStringMigratedTestV0(startReceipt.EvidenceRefs, codexAppServerTurnStartToolOutputPolicyAcceptedV0) {
		t.Fatalf("start receipt=%+v", startReceipt)
	}
	startCalls := readCodexAppServerWebSocketRecordsForTestV0(t, records, 4)
	if got := []string{startCalls[0].Method, startCalls[1].Method, startCalls[2].Method, startCalls[3].Method}; !reflect.DeepEqual(got, []string{"thread/start", "thread/settings/update", "thread/goal/set", "turn/start"}) {
		t.Fatalf("start methods=%+v", got)
	}
	settings := findCodexAppServerWebSocketRecordForTestV0(t, startCalls, "thread/settings/update")
	if !reflect.DeepEqual(settings.Params, map[string]interface{}{
		"threadId": "thread-ref-websocket-direct-001",
		"effort":   "medium",
	}) {
		t.Fatalf("settings websocket=%#v", settings.Params)
	}
	turnStart := findCodexAppServerWebSocketRecordForTestV0(t, startCalls, "turn/start")
	policy, ok := turnStart.Params["toolOutputPolicy"].(map[string]interface{})
	if !ok {
		t.Fatalf("turn/start sin toolOutputPolicy: %#v", turnStart.Params)
	}
	if policy["maxTextBytes"] != float64(2048) ||
		policy["threadReadMaxBytes"] != float64(codexAppServerThreadReadMaxResponseFrameBytesV0) ||
		policy["requireBoundedCommands"] != true {
		t.Fatalf("policy websocket=%#v", policy)
	}

	observeReceipt, err := backend.ObserveCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		GoalRef:         packet.GoalRef,
		ExternalGoalRef: startReceipt.ExternalGoalRef,
	})
	if err != nil {
		t.Fatalf("ObserveCodexGoalV0: %v", err)
	}
	if observeReceipt.Status != orquestagoal.GoalStatusBlockedV0 ||
		observeReceipt.IssueCode != codexAppServerThreadReadFrameTooLargeIssueCodeV0 ||
		!containsStringMigratedTestV0(observeReceipt.EvidenceRefs, "evidence-ref-codex-app-server-thread-read-response-too-large") {
		t.Fatalf("observe receipt=%+v", observeReceipt)
	}
	observeCalls := readCodexAppServerWebSocketRecordsForTestV0(t, records, 3)
	findCodexAppServerWebSocketRecordForTestV0(t, observeCalls, "thread/goal/get")
	findCodexAppServerWebSocketRecordForTestV0(t, observeCalls, "thread/read")
}

func TestServerCodexAppServerGoalBackendV0UmbralUsoAltoDefaultConserva100kV0(t *testing.T) {
	backend := serverCodexAppServerGoalBackendV0{}

	if got := backend.codexAppServerGoalHighTokenUsageThresholdV0(); got != 100000 {
		t.Fatalf("threshold=%d", got)
	}
}

func TestCodexAppServerIssueCodeFromLogFileV0ClasificaResetStdioMigradoV0(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(root, "orquesta-goal.log")
	raw := strings.Repeat("x", codexAppServerDiagnosticLogMaxBytesV0+64) +
		"\nvoid node::ResetStdio() at ../src/node.cc:751\n"
	if err := os.WriteFile(logPath, []byte(raw), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}

	if got := codexAppServerIssueCodeFromLogFileV0(logPath); got != "codex_app_server_wrapper_stdio_failed" {
		t.Fatalf("code=%q", got)
	}
}

func TestPrepareCodexGoalWriteSetV0NoFallaConFicheroExistenteMigradoV0(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "modulos", "orquesta-server")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	existing := filepath.Join(dir, "config_v0.go")
	if err := os.WriteFile(existing, []byte("package orquestaserver\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	backend := serverCodexAppServerGoalBackendV0{CWD: root}
	packet := orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		WriteSet: []orquestagoal.GoalWriteScopeV0{
			{Path: "modulos/orquesta-server"},
			{Path: "modulos/orquesta-server/config_v0.go"},
		},
	}
	if err := backend.prepareCodexGoalWriteSetV0(packet); err != nil {
		t.Fatalf("prepare fallo con fichero existente en write-set: %v", err)
	}
	info, err := os.Stat(existing)
	if err != nil {
		t.Fatal(err)
	}
	if info.IsDir() {
		t.Fatal("el fichero existente del write-set se convirtio en directorio")
	}
}

func TestServerCodexAppServerGoalBackendV0LaunchPrepareFallidoPublicaDetailRelativoV0(t *testing.T) {
	root := t.TempDir()
	blocker := filepath.Join(root, "modulos")
	if err := os.WriteFile(blocker, []byte("not a directory\n"), 0o600); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	protocol := &fakeCodexAppServerProtocolV0{
		thread: serverCodexAppServerThreadV0{ID: "thread-ref-prepare-detail-001"},
	}
	backend := serverCodexAppServerGoalBackendV0{
		Protocol: protocol,
		CWD:      root,
	}

	receipt, err := backend.StartCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef:   "goal-ref-prepare-detail-001",
		Objective: "preparar write set",
		WriteSet: []orquestagoal.GoalWriteScopeV0{{
			Path: "modulos/orquesta-goal",
		}},
	})

	if err == nil {
		t.Fatal("StartCodexGoalV0 debe fallar al preparar write_set bloqueado por fichero intermedio")
	}
	if len(protocol.calls) != 0 {
		t.Fatalf("no debe llamar al backend tras prepare fallido: calls=%v", protocol.calls)
	}
	if receipt.Status != orquestagoal.GoalStatusInvalidV0 ||
		!strings.HasPrefix(receipt.IssueCode, "codex_app_server_write_set_prepare_failed: ") ||
		!strings.Contains(receipt.IssueCode, "write_set_prepare_failed") ||
		!strings.Contains(receipt.IssueCode, "modulos/orquesta-goal") ||
		strings.Contains(receipt.IssueCode, root) {
		t.Fatalf("receipt=%+v root=%q", receipt, root)
	}
	normalized := orquestagoal.NormalizeGoalLaunchReceiptV0(orquestagoal.GoalLaunchReceiptV0{
		Status:  receipt.Status,
		GoalRef: receipt.GoalRef,
		Issues: []orquestagoal.GoalWorkIssueV0{{
			Code: receipt.IssueCode,
		}},
	})
	if len(normalized.Issues) != 1 ||
		normalized.Issues[0].Code != "codex_app_server_write_set_prepare_failed" ||
		!strings.Contains(normalized.Issues[0].Detail, "modulos/orquesta-goal") ||
		strings.Contains(normalized.Issues[0].Detail, root) {
		t.Fatalf("normalized=%+v root=%q", normalized.Issues, root)
	}
}

func TestServerCodexAppServerGoalBackendV0LaunchBloqueaWorkdirInexistenteSinRecrearloV0(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workdir-borrado")
	protocol := &fakeCodexAppServerProtocolV0{
		thread: serverCodexAppServerThreadV0{ID: "thread-ref-workdir-missing-001"},
	}
	backend := serverCodexAppServerGoalBackendV0{
		Protocol: protocol,
		CWD:      root,
	}

	receipt, err := backend.StartCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef:   "goal-ref-workdir-missing-001",
		Objective: "detectar workdir inexistente",
		WriteSet: []orquestagoal.GoalWriteScopeV0{{
			Path: "modulos/orquesta-server",
		}},
	})

	if err == nil {
		t.Fatal("StartCodexGoalV0 debe fallar si el CWD configurado ya no existe")
	}
	if len(protocol.calls) != 0 {
		t.Fatalf("no debe llamar al backend con workdir inexistente: calls=%v", protocol.calls)
	}
	if _, statErr := os.Stat(root); !os.IsNotExist(statErr) {
		t.Fatalf("el runtime recreo o oculto el workdir inexistente: stat=%v", statErr)
	}
	if receipt.Status != orquestagoal.GoalStatusInvalidV0 ||
		!strings.HasPrefix(receipt.IssueCode, "codex_app_server_write_set_prepare_failed: ") ||
		!strings.Contains(receipt.IssueCode, "workdir_unavailable") ||
		strings.Contains(receipt.IssueCode, root) ||
		!containsStringMigratedTestV0(
			receipt.EvidenceRefs,
			"evidence-ref-codex-app-server-write-set-prepare-failed",
		) {
		t.Fatalf("receipt sin diagnostico publico de workdir: %+v root=%q", receipt, root)
	}
}

func TestServerCodexAppServerGoalBackendV0RuntimeWriteSetGuardBloqueaCambioFueraDeScope(t *testing.T) {
	root := t.TempDir()
	protocol := &fakeCodexAppServerProtocolV0{
		thread: serverCodexAppServerThreadV0{ID: "thread-ref-runtime-write-set-001"},
		goal:   serverCodexAppServerThreadGoalV0{ThreadID: "thread-ref-runtime-write-set-001", Status: "active"},
		turn:   serverCodexAppServerTurnV0{ID: "turn-ref-runtime-write-set-001", Status: "complete"},
	}
	backend := serverCodexAppServerGoalBackendV0{
		Protocol: protocol,
		CWD:      root,
		Sandbox:  "workspace-write",
		Runtime:  &serverCodexAppServerGoalRuntimeV0{},
	}
	packet := codexAppServerRuntimeWriteSetGuardPacketForTestV0(
		"goal-ref-runtime-write-set-001",
		"docs",
	)

	receipt, err := backend.StartCodexGoalV0(context.Background(), packet)
	if err != nil || receipt.Status != orquestagoal.GoalStatusRunningV0 {
		t.Fatalf("StartCodexGoalV0 receipt=%+v err=%v", receipt, err)
	}
	writeCodexAppServerGoalResultForTestV0(t, root, "docs", packet.GoalRef, receipt.ExternalGoalRef)
	writeCodexAppServerTestFileV0(t, root, "docs/ok.md", "ok\n")
	writeCodexAppServerTestFileV0(t, root, "fuera.md", "fuera\n")
	protocol.observedGoal = &serverCodexAppServerThreadGoalV0{
		ThreadID: receipt.ExternalGoalRef,
		Status:   "complete",
	}

	observed, err := backend.ObserveCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		GoalRef:         packet.GoalRef,
		ExternalGoalRef: receipt.ExternalGoalRef,
	})
	if err != nil {
		t.Fatalf("ObserveCodexGoalV0: %v", err)
	}
	if observed.Status != orquestagoal.GoalStatusBlockedV0 ||
		observed.IssueCode != codexAppServerRuntimeWriteSetViolationV0 ||
		len(observed.DomainReceiptRefs) != 0 ||
		!containsStringMigratedTestV0(observed.EvidenceRefs, codexAppServerRuntimeWriteSetViolationEvidenceV0) {
		t.Fatalf("observed=%+v", observed)
	}
	if !containsStringPrefixMigratedTestV0(observed.EvidenceRefs, "evidence-ref-codex-app-server-runtime-write-set-outside-fuera-md") {
		t.Fatalf("evidence_refs=%+v", observed.EvidenceRefs)
	}
}

func TestServerCodexAppServerGoalBackendV0RuntimeWriteSetGuardPermiteCambioDentroDeScope(t *testing.T) {
	root := t.TempDir()
	protocol := &fakeCodexAppServerProtocolV0{
		thread: serverCodexAppServerThreadV0{ID: "thread-ref-runtime-write-set-002"},
		goal:   serverCodexAppServerThreadGoalV0{ThreadID: "thread-ref-runtime-write-set-002", Status: "active"},
		turn:   serverCodexAppServerTurnV0{ID: "turn-ref-runtime-write-set-002", Status: "complete"},
	}
	backend := serverCodexAppServerGoalBackendV0{
		Protocol: protocol,
		CWD:      root,
		Sandbox:  "workspace-write",
		Runtime:  &serverCodexAppServerGoalRuntimeV0{},
	}
	packet := codexAppServerRuntimeWriteSetGuardPacketForTestV0(
		"goal-ref-runtime-write-set-002",
		"docs",
	)

	receipt, err := backend.StartCodexGoalV0(context.Background(), packet)
	if err != nil || receipt.Status != orquestagoal.GoalStatusRunningV0 {
		t.Fatalf("StartCodexGoalV0 receipt=%+v err=%v", receipt, err)
	}
	writeCodexAppServerGoalResultForTestV0(t, root, "docs", packet.GoalRef, receipt.ExternalGoalRef)
	writeCodexAppServerTestFileV0(t, root, "docs/ok.md", "ok\n")
	protocol.observedGoal = &serverCodexAppServerThreadGoalV0{
		ThreadID: receipt.ExternalGoalRef,
		Status:   "complete",
	}

	observed, err := backend.ObserveCodexGoalV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		GoalRef:         packet.GoalRef,
		ExternalGoalRef: receipt.ExternalGoalRef,
	})
	if err != nil {
		t.Fatalf("ObserveCodexGoalV0: %v", err)
	}
	if observed.Status != orquestagoal.GoalStatusCompleteV0 ||
		!containsStringMigratedTestV0(observed.DomainReceiptRefs, "domain-receipt-ref-runtime-write-set") ||
		!containsStringMigratedTestV0(observed.EvidenceRefs, codexAppServerRuntimeWriteSetGuardEvidenceV0) {
		t.Fatalf("observed=%+v", observed)
	}
}

func TestCodexAppServerWriteSetLooksLikeFileV0TrataExtensionesComoFicheroMigradoV0(t *testing.T) {
	cases := map[string]bool{
		"docs/informe.md":              true,
		"scripts/foo.sh":               true,
		"modulos/x/config_v0.go":       true,
		"docs/resultado.json":          true,
		"modulos/orquesta-server":      false,
		"generated-apps/demo":          false,
		"docs/paquete/":                false,
		"modulos/orquesta-estado-vivo": false,
	}
	for path, want := range cases {
		if got := codexAppServerWriteSetLooksLikeFileV0(path); got != want {
			t.Fatalf("codexAppServerWriteSetLooksLikeFileV0(%q) = %v, esperado %v", path, got, want)
		}
	}
}

func TestCodexAppServerTmuxBackendV0EnsureShutdownCleanupMigradoV0(t *testing.T) {
	root := shortUnixSocketTestRootV0(t)
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	tmuxLog := filepath.Join(root, "tmux.log")
	fakeTmux := filepath.Join(binDir, "tmux")
	if err := os.WriteFile(fakeTmux, []byte(fakeCodexAppServerTmuxCommandMigratedTestV0()), 0o700); err != nil {
		t.Fatalf("write fake tmux: %v", err)
	}
	runtimeDir := filepath.Join(root, "runtime")
	socketPath := filepath.Join(runtimeDir, codexAppServerTmuxDirV0, "goal.sock")
	backend := serverCodexAppServerTmuxBackendV0{
		CommandPath:    filepath.Join(root, "codex"),
		PathEnv:        binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		SocketPath:     socketPath,
		SessionName:    "orquesta-goal-migrated-1234567890",
		RuntimeWorkDir: runtimeDir,
		ProjectWorkDir: filepath.Join(root, "project"),
		Timeout:        time.Second,
	}
	t.Setenv("ORQUESTA_TEST_TMUX_LOG", tmuxLog)
	t.Setenv("ORQUESTA_TEST_BINARY", os.Args[0])
	probeListener := listenUnixForGenerationTestV0(t, socketPath)
	if err := probeListener.Close(); err != nil {
		t.Fatal(err)
	}

	if err := backend.EnsureV0(context.Background(), fakeCodexAppServerProbeV0{}); err != nil {
		t.Fatalf("EnsureV0: %v", err)
	}
	rawTmuxLog, err := os.ReadFile(tmuxLog)
	if err != nil {
		t.Fatalf("read tmux log: %v", err)
	}
	if !strings.Contains(string(rawTmuxLog), "mkfifo") ||
		!strings.Contains(string(rawTmuxLog), "stdin.pipe") ||
		strings.Contains(string(rawTmuxLog), "< /dev/null") {
		t.Fatalf("app-server tmux debe arrancar con stdin FIFO propia; log=%s", string(rawTmuxLog))
	}
	if _, err := os.Stat(backend.tmuxOwnerMarkerPathV0()); err != nil {
		t.Fatalf("owner marker ausente: %v", err)
	}
	active, err := backend.ReadActiveShutdownWorkV0(context.Background(), orquestaservershutdown.ActiveShutdownWorkRequestV0{})
	if err != nil {
		t.Fatalf("ReadActiveShutdownWorkV0: %v", err)
	}
	if len(active.ActiveWorks) != 1 ||
		!containsStringMigratedTestV0(active.EvidenceRefs, "evidence-ref-codex-app-server-tmux-session") {
		t.Fatalf("active=%+v", active)
	}
	if err := backend.ShutdownV0(context.Background()); err != nil {
		t.Fatalf("ShutdownV0: %v", err)
	}
	active, err = backend.ReadActiveShutdownWorkV0(context.Background(), orquestaservershutdown.ActiveShutdownWorkRequestV0{})
	if err != nil {
		t.Fatalf("ReadActiveShutdownWorkV0 after shutdown: %v", err)
	}
	if len(active.ActiveWorks) != 0 {
		t.Fatalf("active after shutdown=%+v", active)
	}
	_, err = backend.CleanupActiveShutdownWorkV0(context.Background(), orquestaservershutdown.ActiveShutdownWorkCleanupCommandV0{
		CleanupGoalBackends: true,
	})
	if err != nil {
		t.Fatalf("CleanupActiveShutdownWorkV0: %v", err)
	}
}

func TestCodexAppServerTmuxBackendV0ReadActiveShutdownWorkIgnoraEstadoDegradadoSinResiduoVivoV0(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	fakeTmux := filepath.Join(binDir, "tmux")
	if err := os.WriteFile(fakeTmux, []byte("#!/usr/bin/env bash\nexit 1\n"), 0o700); err != nil {
		t.Fatalf("write fake tmux: %v", err)
	}
	runtimeDir := filepath.Join(root, "runtime")
	goalDir := filepath.Join(runtimeDir, codexAppServerTmuxDirV0)
	if err := os.MkdirAll(goalDir, 0o700); err != nil {
		t.Fatalf("mkdir goal dir: %v", err)
	}
	backend := serverCodexAppServerTmuxBackendV0{
		PathEnv:        binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		SocketPath:     filepath.Join(goalDir, "goal.sock"),
		SessionName:    "orquesta-goal-stale-1234567890",
		RuntimeWorkDir: runtimeDir,
		ProjectWorkDir: filepath.Join(root, "project"),
		Timeout:        time.Second,
	}

	active, err := backend.ReadActiveShutdownWorkV0(context.Background(), orquestaservershutdown.ActiveShutdownWorkRequestV0{})
	if err != nil {
		t.Fatalf("ReadActiveShutdownWorkV0: %v", err)
	}
	if len(active.ActiveWorks) != 0 {
		t.Fatalf("active stale sin owner/socket/proceso=%+v", active)
	}
}

func TestCodexAppServerWebSocketThreadReadResponseBudgetV0(t *testing.T) {
	payload := bytes.Repeat([]byte("x"), codexAppServerThreadReadMaxResponseFrameBytesV0+1)
	reader := bufio.NewReader(bytes.NewReader(codexAppServerTestWebSocketFrameV0(payload)))

	err := codexAppServerWebSocketReadResponseWithMaxFrameBytesV0(
		reader,
		2,
		&serverCodexAppServerThreadReadResponseV0{},
		codexAppServerThreadReadMaxResponseFrameBytesV0,
		codexAppServerThreadReadFrameTooLargeIssueCodeV0,
	)
	var callErr codexAppServerCallErrorV0
	if !errors.As(err, &callErr) {
		t.Fatalf("err=%T %v", err, err)
	}
	if callErr.Code != codexAppServerThreadReadFrameTooLargeIssueCodeV0 {
		t.Fatalf("code=%q", callErr.Code)
	}
}

func TestCodexAppServerCommandProtocolThreadReadResponseBudgetV0(t *testing.T) {
	root := t.TempDir()
	fakeCodex := filepath.Join(root, "fake-codex-app-server")
	oversizedThreadReadBytes := codexAppServerThreadReadMaxResponseFrameBytesV0 + 1
	body := `#!/bin/sh
set -eu
IFS= read -r _init
printf '%s\n' '{"jsonrpc":"2.0","id":1,"result":{}}'
IFS= read -r _initialized
IFS= read -r _call
printf '%s' '{"jsonrpc":"2.0","id":2,"result":{"thread":{"id":"thread-big","items":["'
dd if=/dev/zero bs=1 count=` + strconv.Itoa(oversizedThreadReadBytes) + ` 2>/dev/null | tr '\000' x
printf '%s\n' '"]}}}'
`
	if err := os.WriteFile(fakeCodex, []byte(body), 0o700); err != nil {
		t.Fatalf("write fake codex app server: %v", err)
	}
	protocol := serverCodexAppServerCommandProtocolV0{
		CommandPath: fakeCodex,
		Args:        []string{"--fake"},
		Timeout:     20 * time.Second,
	}

	_, err := protocol.ReadThreadV0(context.Background(), "thread-big", true)
	var callErr codexAppServerCallErrorV0
	if !errors.As(err, &callErr) {
		t.Fatalf("err=%T %v", err, err)
	}
	if callErr.Code != codexAppServerThreadReadFrameTooLargeIssueCodeV0 {
		t.Fatalf("code=%q", callErr.Code)
	}
}

func TestCodexAppServerCommandProtocolTurnStartResponseBudgetV0(t *testing.T) {
	root := t.TempDir()
	fakeCodex := filepath.Join(root, "fake-codex-app-server")
	oversizedResponseBytes := codexAppServerCommandProtocolDefaultMaxResponseLineBytesV0 + 1
	body := `#!/bin/sh
set -eu
IFS= read -r _init
printf '%s\n' '{"jsonrpc":"2.0","id":1,"result":{}}'
IFS= read -r _initialized
IFS= read -r _call
printf '%s' '{"jsonrpc":"2.0","id":2,"result":{"turn":{"id":"turn-big","status":"'
dd if=/dev/zero bs=1 count=` + strconv.Itoa(oversizedResponseBytes) + ` 2>/dev/null | tr '\000' x
printf '%s\n' '"}}}'
`
	if err := os.WriteFile(fakeCodex, []byte(body), 0o700); err != nil {
		t.Fatalf("write fake codex app server: %v", err)
	}
	protocol := serverCodexAppServerCommandProtocolV0{
		CommandPath: fakeCodex,
		Args:        []string{"--fake"},
		Timeout:     20 * time.Second,
	}

	_, err := protocol.StartTurnV0(context.Background(), serverCodexAppServerTurnStartParamsV0{
		ThreadID:  "thread-big",
		InputText: "turn con respuesta gigante",
	})
	var callErr codexAppServerCallErrorV0
	if !errors.As(err, &callErr) {
		t.Fatalf("err=%T %v", err, err)
	}
	if callErr.Code != codexAppServerCommandResponseTooLargeIssueCodeV0 {
		t.Fatalf("code=%q", callErr.Code)
	}
}

func TestCodexAppServerCommandProtocolThreadSettingsUpdateJSONV0(t *testing.T) {
	root := t.TempDir()
	fakeCodex := filepath.Join(root, "fake-codex-app-server")
	capturePath := filepath.Join(root, "request.json")
	body := `#!/bin/sh
set -eu
IFS= read -r _init
printf '%s\n' '{"jsonrpc":"2.0","id":1,"result":{}}'
IFS= read -r _initialized
IFS= read -r _call
printf '%s' "$_call" > "$1"
printf '%s\n' '{"jsonrpc":"2.0","id":2,"result":{}}'
`
	if err := os.WriteFile(fakeCodex, []byte(body), 0o700); err != nil {
		t.Fatalf("write fake codex app server: %v", err)
	}
	protocol := serverCodexAppServerCommandProtocolV0{
		CommandPath: fakeCodex,
		Args:        []string{capturePath},
		Timeout:     2 * time.Second,
	}
	if err := protocol.UpdateThreadSettingsV0(context.Background(), serverCodexAppServerThreadSettingsUpdateParamsV0{
		ThreadID: "thread-command-settings-001",
		Effort:   "medium",
	}); err != nil {
		t.Fatalf("UpdateThreadSettingsV0: %v", err)
	}
	raw, err := os.ReadFile(capturePath)
	if err != nil {
		t.Fatalf("read request: %v", err)
	}
	var request struct {
		Method string                 `json:"method"`
		Params map[string]interface{} `json:"params"`
	}
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatalf("decode request: %v", err)
	}
	if request.Method != "thread/settings/update" || !reflect.DeepEqual(request.Params, map[string]interface{}{
		"threadId": "thread-command-settings-001",
		"effort":   "medium",
	}) {
		t.Fatalf("request=%#v", request)
	}
}

func TestCodexAppServerLegacyRPCReaderResponseBudgetV0(t *testing.T) {
	oversizedResponseBytes := codexAppServerCommandProtocolDefaultMaxResponseLineBytesV0 + 1
	var payload bytes.Buffer
	payload.WriteString(`{"jsonrpc":"2.0","id":2,"result":{"turn":{"id":"turn-legacy-big","status":"`)
	payload.Write(bytes.Repeat([]byte("x"), oversizedResponseBytes))
	payload.WriteString(`"}}}`)
	payload.WriteByte('\n')

	err := decodeCodexAppServerRPCResponseReaderV0(
		bytes.NewReader(payload.Bytes()),
		2,
		&serverCodexAppServerTurnStartResponseV0{},
	)
	var callErr codexAppServerCallErrorV0
	if !errors.As(err, &callErr) {
		t.Fatalf("err=%T %v", err, err)
	}
	if callErr.Code != codexAppServerCommandResponseTooLargeIssueCodeV0 {
		t.Fatalf("code=%q", callErr.Code)
	}
}

func TestCodexAppServerCommandProtocolStderrDiagnosticoAcotadoV0(t *testing.T) {
	root := t.TempDir()
	fakeCodex := filepath.Join(root, "fake-codex-app-server")
	body := `#!/bin/sh
set -eu
dd if=/dev/zero bs=1024 count=64 2>/dev/null | tr '\000' x >&2
printf '%s\n' '401 unauthorized api.openai.com/v1/responses' >&2
exit 1
`
	if err := os.WriteFile(fakeCodex, []byte(body), 0o700); err != nil {
		t.Fatalf("write fake codex app server: %v", err)
	}
	protocol := serverCodexAppServerCommandProtocolV0{
		CommandPath: fakeCodex,
		Args:        []string{"--fake"},
		Timeout:     2 * time.Second,
	}

	_, err := protocol.StartTurnV0(context.Background(), serverCodexAppServerTurnStartParamsV0{
		ThreadID:  "thread-stderr-big",
		InputText: "turn con stderr gigante",
	})
	var callErr codexAppServerCallErrorV0
	if !errors.As(err, &callErr) {
		t.Fatalf("err=%T %v", err, err)
	}
	if callErr.Code != "codex_app_server_provider_unauthorized" {
		t.Fatalf("code=%q", callErr.Code)
	}
}

func TestCodexAppServerIssueCodeFromMessageV0ClasificaTokenInvalidadoV0(t *testing.T) {
	for _, message := range []string{
		`{"error":{"code":"token_invalidated","message":"refresh_token_invalidated"}}`,
		"Your access token could not be refreshed because you have since logged out or signed in to another account. Please sign in again.",
		"refresh_token_reused / token_expired",
	} {
		if got := codexAppServerIssueCodeFromMessageV0(message); got != "codex_app_server_provider_unauthorized" {
			t.Fatalf("message=%q issue=%q", message, got)
		}
	}
}

func TestCodexAppServerDiagnosticTailFileV0LeeSoloColaV0(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app-server.log")
	raw := strings.Repeat("prefix-noise\n", codexAppServerDiagnosticLogMaxBytesV0) +
		"401 unauthorized api.openai.com/v1/responses\n"
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}

	tail, err := readCodexAppServerDiagnosticTailFileV0(path, codexAppServerDiagnosticLogMaxBytesV0)

	if err != nil {
		t.Fatalf("tail: %v", err)
	}
	if len(tail) > codexAppServerDiagnosticLogMaxBytesV0 {
		t.Fatalf("tail len=%d", len(tail))
	}
	if !strings.Contains(string(tail), "401 unauthorized api.openai.com/v1/responses") {
		t.Fatalf("tail no conserva diagnostico final")
	}
	if got := codexAppServerIssueCodeFromLogFileV0(path); got != "codex_app_server_provider_unauthorized" {
		t.Fatalf("issue=%q", got)
	}
}

func TestCodexAppServerWebSocketDefaultFrameBudgetConserva16MiBV0(t *testing.T) {
	payload := []byte(`{"id":2,"result":{"thread":{"id":"thread-ref-small","status":"closed"}}}`)
	reader := bufio.NewReader(bytes.NewReader(codexAppServerTestWebSocketFrameV0(payload)))
	var out serverCodexAppServerThreadReadResponseV0

	if err := codexAppServerWebSocketReadResponseV0(reader, 2, &out); err != nil {
		t.Fatalf("ReadResponseV0: %v", err)
	}
	if out.Thread.ID != "thread-ref-small" || out.Thread.Status != "closed" {
		t.Fatalf("out=%+v", out)
	}
}

func codexAppServerTestWebSocketFrameV0(payload []byte) []byte {
	header := []byte{0x81}
	switch size := len(payload); {
	case size < 126:
		header = append(header, byte(size))
	case size <= 65535:
		header = append(header, 126, byte(size>>8), byte(size))
	default:
		header = append(header, 127)
		var length [8]byte
		binary.BigEndian.PutUint64(length[:], uint64(size))
		header = append(header, length[:]...)
	}
	return append(header, payload...)
}

type codexAppServerWebSocketRecordForTestV0 struct {
	Method string
	Params map[string]interface{}
}

func startCodexAppServerWebSocketScriptForTestV0(
	t *testing.T,
) (string, <-chan codexAppServerWebSocketRecordForTestV0) {
	t.Helper()
	root := shortUnixSocketTestRootV0(t)
	socketPath := filepath.Join(root, "codex-app-server.sock")
	if err := validateUnixSocketPathLengthV0(socketPath); err != nil {
		t.Fatalf("ruta socket Unix: %v", err)
	}
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "operation not permitted") {
			t.Skipf("unix socket no disponible en este sandbox: %v", err)
		}
		t.Fatalf("listen unix: %v", err)
	}
	records := make(chan codexAppServerWebSocketRecordForTestV0, 8)
	errs := make(chan error, 1)
	go func() {
		defer close(records)
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			if err := serveCodexAppServerWebSocketCallForTestV0(conn, records); err != nil {
				select {
				case errs <- err:
				default:
				}
				return
			}
		}
	}()
	t.Cleanup(func() {
		_ = listener.Close()
		select {
		case err := <-errs:
			if codexAppServerWebSocketBenignCloseForTestV0(err) {
				return
			}
			t.Fatalf("fake websocket app-server: %v", err)
		default:
		}
	})
	return socketPath, records
}

func codexAppServerWebSocketBenignCloseForTestV0(err error) bool {
	return errors.Is(err, net.ErrClosed) ||
		errors.Is(err, os.ErrClosed) ||
		errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, syscall.ECONNRESET)
}

func serveCodexAppServerWebSocketCallForTestV0(
	conn net.Conn,
	records chan<- codexAppServerWebSocketRecordForTestV0,
) error {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	request, err := http.ReadRequest(reader)
	if err != nil {
		return err
	}
	accept := codexAppServerWebSocketAcceptV0(request.Header.Get("Sec-WebSocket-Key"))
	if _, err := fmt.Fprintf(conn, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\n\r\n", accept); err != nil {
		return err
	}
	initPayload, _, err := codexAppServerWebSocketReadFrameV0(reader)
	if err != nil {
		return err
	}
	var initMessage map[string]interface{}
	if err := json.Unmarshal(initPayload, &initMessage); err != nil {
		return err
	}
	if method := strings.TrimSpace(fmt.Sprint(initMessage["method"])); method != "initialize" {
		return fmt.Errorf("initialize method=%q", method)
	}
	if _, err := conn.Write(codexAppServerTestWebSocketFrameV0([]byte(`{"id":1,"result":{}}`))); err != nil {
		return err
	}
	callPayload, _, err := codexAppServerWebSocketReadFrameV0(reader)
	if err != nil {
		return err
	}
	var call struct {
		Method string                 `json:"method"`
		Params map[string]interface{} `json:"params"`
	}
	if err := json.Unmarshal(callPayload, &call); err != nil {
		return err
	}
	call.Method = strings.TrimSpace(call.Method)
	records <- codexAppServerWebSocketRecordForTestV0{Method: call.Method, Params: call.Params}
	return writeCodexAppServerWebSocketScriptResponseForTestV0(conn, call.Method)
}

func writeCodexAppServerWebSocketScriptResponseForTestV0(conn net.Conn, method string) error {
	switch strings.TrimSpace(method) {
	case "thread/start":
		_, err := conn.Write(codexAppServerTestWebSocketFrameV0([]byte(`{"id":2,"result":{"thread":{"id":"thread-ref-websocket-direct-001"}}}`)))
		return err
	case "thread/settings/update":
		_, err := conn.Write(codexAppServerTestWebSocketFrameV0([]byte(`{"id":2,"result":{}}`)))
		return err
	case "thread/goal/set":
		_, err := conn.Write(codexAppServerTestWebSocketFrameV0([]byte(`{"id":2,"result":{"goal":{"threadId":"thread-ref-websocket-direct-001","status":"active"}}}`)))
		return err
	case "turn/start":
		_, err := conn.Write(codexAppServerTestWebSocketFrameV0([]byte(`{"id":2,"result":{"turn":{"id":"turn-ref-websocket-direct-001","status":"inProgress"}}}`)))
		return err
	case "thread/goal/get":
		_, err := conn.Write(codexAppServerTestWebSocketFrameV0([]byte(`{"id":2,"result":{"goal":{"threadId":"thread-ref-websocket-direct-001","status":"active"}}}`)))
		return err
	case "thread/read":
		payload := bytes.Repeat([]byte("x"), codexAppServerThreadReadMaxResponseFrameBytesV0+1)
		_, err := conn.Write(codexAppServerTestWebSocketFrameV0(payload))
		return err
	default:
		return fmt.Errorf("method inesperado: %s", method)
	}
}

func readCodexAppServerWebSocketRecordsForTestV0(
	t *testing.T,
	records <-chan codexAppServerWebSocketRecordForTestV0,
	count int,
) []codexAppServerWebSocketRecordForTestV0 {
	t.Helper()
	out := make([]codexAppServerWebSocketRecordForTestV0, 0, count)
	timeout := time.After(2 * time.Second)
	for len(out) < count {
		select {
		case record, ok := <-records:
			if !ok {
				t.Fatalf("records channel closed after %d/%d records", len(out), count)
			}
			out = append(out, record)
		case <-timeout:
			t.Fatalf("timeout esperando records websocket: got=%+v want=%d", out, count)
		}
	}
	return out
}

func findCodexAppServerWebSocketRecordForTestV0(
	t *testing.T,
	records []codexAppServerWebSocketRecordForTestV0,
	method string,
) codexAppServerWebSocketRecordForTestV0 {
	t.Helper()
	for _, record := range records {
		if record.Method == method {
			return record
		}
	}
	t.Fatalf("method %q no encontrado en %+v", method, records)
	return codexAppServerWebSocketRecordForTestV0{}
}

type fakeCodexAppServerProbeV0 struct{}

func (fakeCodexAppServerProbeV0) ProbeV0(context.Context) error {
	return nil
}

type fakeCodexAppServerBackendShutdownV0 struct {
	calls       int
	forcedCalls int
	err         error
	forcedErr   error
}

func (fake *fakeCodexAppServerBackendShutdownV0) ShutdownV0(context.Context) error {
	fake.calls++
	return fake.err
}

func (fake *fakeCodexAppServerBackendShutdownV0) ShutdownForcedStopV0(context.Context) error {
	fake.forcedCalls++
	return fake.forcedErr
}

type fakeCodexAppServerProtocolV0 struct {
	calls             []string
	startParams       serverCodexAppServerThreadStartParamsV0
	settingsParams    serverCodexAppServerThreadSettingsUpdateParamsV0
	setParams         serverCodexAppServerThreadGoalSetParamsV0
	turnParams        serverCodexAppServerTurnStartParamsV0
	turnParamsHistory []serverCodexAppServerTurnStartParamsV0
	getThreadID       string
	onStartTurn       func()

	thread            serverCodexAppServerThreadV0
	goal              serverCodexAppServerThreadGoalV0
	turn              serverCodexAppServerTurnV0
	setGoalErr        error
	updateSettingsErr error
	startTurnErrs     []error
	getGoalErr        error
	observedGoal      *serverCodexAppServerThreadGoalV0
	readThread        serverCodexAppServerThreadReadV0
	readThreadErr     error
	readThreadID      string
	readIncludeTurns  bool
}

func (fake *fakeCodexAppServerProtocolV0) StartThreadV0(
	_ context.Context,
	params serverCodexAppServerThreadStartParamsV0,
) (serverCodexAppServerThreadV0, error) {
	fake.calls = append(fake.calls, "thread/start")
	fake.startParams = params
	return fake.thread, nil
}

func (fake *fakeCodexAppServerProtocolV0) UpdateThreadSettingsV0(
	_ context.Context,
	params serverCodexAppServerThreadSettingsUpdateParamsV0,
) error {
	fake.calls = append(fake.calls, "thread/settings/update")
	fake.settingsParams = params
	return fake.updateSettingsErr
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
	fake.turnParamsHistory = append(fake.turnParamsHistory, params)
	if fake.onStartTurn != nil {
		fake.onStartTurn()
	}
	if len(fake.startTurnErrs) > 0 {
		err := fake.startTurnErrs[0]
		fake.startTurnErrs = fake.startTurnErrs[1:]
		if err != nil {
			return serverCodexAppServerTurnV0{}, err
		}
	}
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

func fakeCodexAppServerTmuxCommandMigratedTestV0() string {
	return fakeGenerationLeaseTmuxScriptV0()
}

func containsStringMigratedTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func waitForMigratedTestConditionV0(t *testing.T, timeout time.Duration, ready func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if ready() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	if ready() {
		return
	}
	t.Fatalf("condition not ready before %s", timeout)
}

func containsStringPrefixMigratedTestV0(values []string, prefix string) bool {
	for _, value := range values {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func codexAppServerRuntimeWriteSetGuardPacketForTestV0(
	goalRef string,
	writeSet string,
) orquestaruntimecodexgoal.CodexGoalStartPacketV0 {
	scopes := []orquestagoal.GoalWriteScopeV0{{Path: writeSet}}
	return orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef:   goalRef,
		Objective: "probar guard runtime de write-set",
		WriteSet:  scopes,
		DirectionContract: orquestaruntimecodexgoal.CodexGoalDirectionContractV0{
			WriteSetEnforcement: orquestaruntimecodexgoal.CodexGoalWriteSetEnforcementV0,
			MinimumSandbox:      orquestaruntimecodexgoal.CodexGoalMinimumSandboxV0,
			AllowedWriteSet:     scopes,
		},
	}
}

func writeCodexAppServerGoalResultForTestV0(
	t *testing.T,
	root string,
	dir string,
	goalRef string,
	externalGoalRef string,
) {
	t.Helper()
	payload, err := json.Marshal(map[string]any{
		"schema_version":      orquestaruntimecodexgoal.CodexGoalResultSchemaV0,
		"status":              orquestagoal.GoalStatusCompleteV0,
		"goal_ref":            goalRef,
		"external_goal_ref":   externalGoalRef,
		"summary":             "resultado terminal",
		"artifact_paths":      []string{filepath.ToSlash(filepath.Join(dir, "ok.md"))},
		"domain_receipt_refs": []string{"domain-receipt-ref-runtime-write-set"},
		"evidence_refs":       []string{"evidence-ref-runtime-write-set-result"},
	})
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	writeCodexAppServerTestFileV0(
		t,
		root,
		filepath.ToSlash(filepath.Join(dir, orquestaruntimecodexgoal.CodexGoalResultFileNameForGoalRefV0(goalRef))),
		string(payload),
	)
}

func writeCodexAppServerTestFileV0(t *testing.T, root string, rel string, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

var _ = errors.New
