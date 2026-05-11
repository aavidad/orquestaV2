package orquestaruntime

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildAgentProgressReportFromHeartbeatV0DistingueProgresoEstancamientoYBucle(t *testing.T) {
	policy := AgentProgressHeartbeatPolicyV0{
		StalledAfterNoProgressTicks: 1,
		LoopAfterRepeatedActions:    3,
	}
	assertAgentProgressPolicyIsNeutralV0(t, policy)

	snapshot := processRuntimeSnapshotForProgressHeartbeatTestV0(ProcessRuntimeRunningV0)
	previous := agentProgressHeartbeatForTestV0(1, 1, 0)

	cases := []struct {
		name                  string
		current               AgentProgressHeartbeatV0
		wantStatus            AgentProgressStatusV0
		wantNoProgressTicks   int
		wantRepeatedActionCnt int
	}{
		{
			name:                "trabajo vivo con avance",
			current:             agentProgressHeartbeatForTestV0(2, 2, 0),
			wantStatus:          AgentProgressingV0,
			wantNoProgressTicks: 0,
		},
		{
			name:                "heartbeat vivo sin progreso",
			current:             agentProgressHeartbeatForTestV0(2, 1, 0),
			wantStatus:          AgentStalledV0,
			wantNoProgressTicks: 1,
		},
		{
			name:                  "heartbeat vivo en bucle",
			current:               agentProgressHeartbeatForTestV0(3, 1, 3),
			wantStatus:            AgentLoopDetectedV0,
			wantNoProgressTicks:   2,
			wantRepeatedActionCnt: 3,
		},
	}

	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			report, issues := BuildAgentProgressReportFromHeartbeatV0(
				"agent-progress-report-ref-heartbeat-00"+string(rune('1'+i)),
				snapshot,
				&previous,
				tc.current,
				policy,
			)
			if len(issues) != 0 {
				t.Fatalf("issues inesperados: %#v", issues)
			}
			if report.Status != tc.wantStatus {
				t.Fatalf("status = %q, se esperaba %q", report.Status, tc.wantStatus)
			}
			if report.NoProgressTicks != tc.wantNoProgressTicks {
				t.Fatalf("no_progress_ticks = %d", report.NoProgressTicks)
			}
			if report.RepeatedActionCount != tc.wantRepeatedActionCnt {
				t.Fatalf("repeated_action_count = %d", report.RepeatedActionCount)
			}
			assertAgentProgressReportConsumiblePorDirectorV0(t, report)
		})
	}
}

func assertAgentProgressPolicyIsNeutralV0(t *testing.T, policy AgentProgressHeartbeatPolicyV0) {
	t.Helper()
	raw, err := json.Marshal(policy)
	if err != nil {
		t.Fatalf("marshal policy: %v", err)
	}
	lower := strings.ToLower(string(raw))
	for _, forbidden := range []string{"provider", "proveedor", "database", "db", "model", "modelo"} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("policy filtra detalle prohibido %q: %s", forbidden, raw)
		}
	}
}

func assertAgentProgressReportConsumiblePorDirectorV0(t *testing.T, report AgentProgressReportV0) {
	t.Helper()
	if issues := ValidateAgentProgressReportV0(report); len(issues) != 0 {
		t.Fatalf("reporte invalido: %#v", issues)
	}
	if report.ReportID == "" || report.AgentRequestID == "" || report.RunID == "" {
		t.Fatalf("reporte sin refs publicas: %+v", report)
	}
	if len(report.EvidenceRefs) < 2 {
		t.Fatalf("reporte sin refs de heartbeat/proceso: %+v", report)
	}
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	lower := strings.ToLower(string(raw))
	for _, forbidden := range []string{
		"provider",
		"proveedor",
		"database",
		"db",
		"command_path",
		"working_dir",
		"pid",
		"home",
		"oauth",
		"token",
	} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("reporte filtra detalle prohibido %q: %s", forbidden, raw)
		}
	}
}

func agentProgressHeartbeatForTestV0(
	tickCounter int,
	progressCounter int,
	repeatedActionCount int,
) AgentProgressHeartbeatV0 {
	return AgentProgressHeartbeatV0{
		HeartbeatRef:        "heartbeat-ref-001",
		RunID:               "run-ref-001",
		AgentRequestID:      "agent-request-ref-001",
		ProcessRef:          "process-ref-v0-000001",
		TickCounter:         tickCounter,
		ProgressCounter:     progressCounter,
		RepeatedActionCount: repeatedActionCount,
		EvidenceRefs:        []string{"supervision-evidence-ref-001"},
	}
}

func processRuntimeSnapshotForProgressHeartbeatTestV0(
	status ProcessRuntimeStatusV0,
) ProcessRuntimeSnapshotV0 {
	return ProcessRuntimeSnapshotV0{
		SchemaVersion: ProcessRuntimeConnectorVersionV0,
		ProcessRef:    "process-ref-v0-000001",
		SessionRef:    "session-ref-v0-000001",
		LaunchRef:     "launch-ref-v0-000001",
		Status:        status,
	}
}
