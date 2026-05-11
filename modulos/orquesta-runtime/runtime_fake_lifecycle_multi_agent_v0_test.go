package orquestaruntime

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestRuntimeFakeLifecycleV0DosAgentesIndependientesMismoRun(t *testing.T) {
	runtime := NewRuntimeFakeLifecycleV0()
	const runID = "run-ref-runtime-multi-001"
	const agent1ID = "agent-request-ref-runtime-multi-001"
	const agent2ID = "agent-request-ref-runtime-multi-002"

	launch1 := runtimeFakeLifecycleMultiAgentLaunchV0(agent1ID, runID, "corr-agent-launcher-multi-001", "idem-agent-launcher-multi-001")
	launch2 := runtimeFakeLifecycleMultiAgentLaunchV0(agent2ID, runID, "corr-agent-launcher-multi-002", "idem-agent-launcher-multi-002")
	requireRuntimeFakeLifecycleDistinctLaunchesV0(t, launch1, launch2)

	agent1Launched, err := runtime.LaunchAgentV0(launch1)
	requireNoRuntimeFakeLifecycleErrorV0(t, err)
	agent2Launched, err := runtime.LaunchAgentV0(launch2)
	requireNoRuntimeFakeLifecycleErrorV0(t, err)
	if agent1Launched.Status != RuntimeFakeLifecycleLaunchedV0 {
		t.Fatalf("agent1 status inicial = %q", agent1Launched.Status)
	}
	if agent2Launched.Status != RuntimeFakeLifecycleLaunchedV0 {
		t.Fatalf("agent2 status inicial = %q", agent2Launched.Status)
	}

	agent2Progress := runtimeFakeLifecycleMultiAgentProgressV0(agent2ID, runID, "agent-progress-report-ref-multi-002", AgentProgressingV0)
	agent2Progressing, err := runtime.ReportProgressV0(agent2Progress)
	requireNoRuntimeFakeLifecycleErrorV0(t, err)
	if agent2Progressing.Status != RuntimeFakeLifecycleProgressingV0 {
		t.Fatalf("agent2 status tras progress = %q", agent2Progressing.Status)
	}

	agent1Loop := runtimeFakeLifecycleMultiAgentProgressV0(agent1ID, runID, "agent-progress-report-ref-loop-multi-001", AgentLoopDetectedV0)
	agent1Looped, err := runtime.ReportProgressV0(agent1Loop)
	requireNoRuntimeFakeLifecycleErrorV0(t, err)
	if agent1Looped.Status != RuntimeFakeLifecycleLoopDetectedV0 {
		t.Fatalf("agent1 status tras loop = %q", agent1Looped.Status)
	}

	stop1 := runtimeFakeLifecycleMultiAgentStopV0(agent1ID, runID, "corr-agent-stopper-multi-001", "idem-agent-stopper-multi-001")
	agent1Stopped, err := runtime.StopAgentV0(stop1)
	requireNoRuntimeFakeLifecycleErrorV0(t, err)
	if agent1Stopped.Status != RuntimeFakeLifecycleStoppedV0 {
		t.Fatalf("agent1 status tras stop = %q", agent1Stopped.Status)
	}
	if agent1Stopped.StopRef != stop1.CorrelationID {
		t.Fatalf("agent1 stop_ref = %q", agent1Stopped.StopRef)
	}
	if agent1Stopped.LastReportRef != agent1Loop.ReportID {
		t.Fatalf("agent1 last_report_ref = %q", agent1Stopped.LastReportRef)
	}

	agent2Repeated, err := runtime.LaunchAgentV0(launch2)
	requireNoRuntimeFakeLifecycleErrorV0(t, err)
	if !reflect.DeepEqual(agent2Repeated, agent2Progressing) {
		t.Fatalf("launch repetido de agent2 cambio snapshot: before=%+v after=%+v", agent2Progressing, agent2Repeated)
	}
	if agent2Repeated.Status != RuntimeFakeLifecycleProgressingV0 {
		t.Fatalf("agent2 status estable = %q", agent2Repeated.Status)
	}
	if agent2Repeated.StopRef != "" {
		t.Fatalf("agent2 heredo stop_ref: %q", agent2Repeated.StopRef)
	}
	if agent2Repeated.LastReportRef != agent2Progress.ReportID {
		t.Fatalf("agent2 last_report_ref = %q", agent2Repeated.LastReportRef)
	}
	if agent2Repeated.LastReportRef == agent1Stopped.LastReportRef {
		t.Fatalf("agent2 heredo last_report_ref de agent1: %q", agent2Repeated.LastReportRef)
	}
	if agent2Repeated.LaunchRef != launch2.CorrelationID {
		t.Fatalf("agent2 launch_ref = %q", agent2Repeated.LaunchRef)
	}

	requireRuntimeFakeLifecycleNoExternalDetailV0(t, struct {
		Launches  []AgentLauncherInboundV0
		Reports   []AgentProgressReportV0
		Stops     []AgentStopperInboundV0
		Snapshots []RuntimeFakeLifecycleSnapshotV0
	}{
		Launches:  []AgentLauncherInboundV0{launch1, launch2},
		Reports:   []AgentProgressReportV0{agent2Progress, agent1Loop},
		Stops:     []AgentStopperInboundV0{stop1},
		Snapshots: []RuntimeFakeLifecycleSnapshotV0{agent1Launched, agent2Launched, agent2Progressing, agent1Looped, agent1Stopped, agent2Repeated},
	})
}

func runtimeFakeLifecycleMultiAgentLaunchV0(agentRequestID, runID, correlationID, idempotencyKey string) AgentLauncherInboundV0 {
	req := launchRuntimeAgentRequestValidaV0()
	req.AgentRequestID = agentRequestID
	req.RunID = runID
	req.PhaseID = "phase-ref-runtime-multi-001"
	req.TaskRef = "task-ref-runtime-multi-001"
	req.CapacityRequestRef = "capacity-request-ref-runtime-multi-001"
	req.Role = "implementador_runtime_multi"
	req.Summary = "Orden compacta para agente independiente."
	req.EvidenceRefs = []string{"workflow-evidence-ref-multi-001"}
	return AgentLauncherInboundV0{
		TargetPort:     AgentLauncherTargetPortV0,
		MessageType:    AgentLauncherMessageTypeV0,
		CorrelationID:  correlationID,
		IdempotencyKey: idempotencyKey,
		Payload:        &req,
	}
}

func runtimeFakeLifecycleMultiAgentProgressV0(agentRequestID, runID, reportID string, status AgentProgressStatusV0) AgentProgressReportV0 {
	report := agentProgressReportValidoV0()
	report.ReportID = reportID
	report.RunID = runID
	report.AgentRequestID = agentRequestID
	report.Status = status
	report.NoProgressTicks = 0
	report.RepeatedActionCount = 0
	report.Summary = "Senal compacta de supervision del agente."
	report.EvidenceRefs = []string{"supervision-evidence-ref-multi-001"}
	if status == AgentLoopDetectedV0 {
		report.NoProgressTicks = 2
		report.RepeatedActionCount = 1
	}
	return report
}

func runtimeFakeLifecycleMultiAgentStopV0(agentRequestID, runID, correlationID, idempotencyKey string) AgentStopperInboundV0 {
	req := stopRuntimeAgentRequestValidaV0()
	req.AgentRequestID = agentRequestID
	req.RunID = runID
	req.ReasonCode = "director.loop_detected_stop"
	req.Summary = "Solicitud logica de parada del agente."
	req.EvidenceRefs = []string{"director-evidence-ref-multi-001"}
	return AgentStopperInboundV0{
		TargetPort:     AgentStopperTargetPortV0,
		MessageType:    AgentStopperMessageTypeV0,
		CorrelationID:  correlationID,
		IdempotencyKey: idempotencyKey,
		Payload:        &req,
	}
}

func requireRuntimeFakeLifecycleDistinctLaunchesV0(t *testing.T, first, second AgentLauncherInboundV0) {
	t.Helper()
	if first.Payload == nil || second.Payload == nil {
		t.Fatalf("payload ausente en launch multiagente")
	}
	if first.Payload.RunID != second.Payload.RunID {
		t.Fatalf("run_id distinto: first=%q second=%q", first.Payload.RunID, second.Payload.RunID)
	}
	if first.Payload.AgentRequestID == second.Payload.AgentRequestID {
		t.Fatalf("agent_request_id compartido: %q", first.Payload.AgentRequestID)
	}
	if first.CorrelationID == second.CorrelationID {
		t.Fatalf("correlation_id compartido: %q", first.CorrelationID)
	}
	if first.IdempotencyKey == second.IdempotencyKey {
		t.Fatalf("idempotency_key compartida: %q", first.IdempotencyKey)
	}
}

func requireRuntimeFakeLifecycleNoExternalDetailV0(t *testing.T, value any) {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal para validar detalles externos: %v", err)
	}
	normalized := strings.ToLower(string(raw))
	for _, token := range runtimeFakeLifecycleForbiddenExternalTokensV0 {
		if strings.Contains(normalized, token) {
			t.Fatalf("detalle externo %q detectado en %s", token, raw)
		}
	}
}

var runtimeFakeLifecycleForbiddenExternalTokensV0 = []string{
	"runtime real", "pid", "tmux", "docker", "http://", "https://", "://",
	"/home/", "\\users\\", "$home", "home=", "oauth",
	"openai", "anthropic", "claude", "gpt-", "gemini", "provider", "proveedor", "model", "modelo",
	"sqlite", "postgres", "mysql", "mongodb", "database", "db://",
}
