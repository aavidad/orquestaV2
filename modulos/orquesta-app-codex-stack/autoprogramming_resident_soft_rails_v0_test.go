package orquestaappcodexstack

import (
	"testing"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func TestAutoprogrammingResidentModeV0RailBlandoNoPreparaSelfRepairV0(t *testing.T) {
	shouldRepair := autoprogrammingResidentShouldRepairV0(
		orquestamcp.MCPRunSupervisorToolInputV0{ResidentMode: true},
		CodexSupervisorResultV0{
			Last: CodexSupervisorRuntimeSnapshotV0{Status: CodexSupervisorRuntimeStoppedV0},
		},
		orquestamcp.MCPRunSupervisorToolResultV0{
			Estado: orquestamcp.MCPRunSupervisorEstadoOKV0,
			RunRef: "run-ref-soft-rail-001",
			EvidenceRefs: []string{
				"gate-followup-required",
				"gate-action:request_followup_review",
				"gate-issue:file_too_large",
				"gate-issue:file_outside_write_set",
				"gate-issue:write_set_target_missing:web",
				"ack-pending-rail:provider",
			},
		},
	)

	if shouldRepair {
		t.Fatalf("rail blando aceptado no debe preparar self-repair bloqueante")
	}
}

func TestAutoprogrammingResidentModeV0NoConfundeNoBloqueanteConBloqueoV0(t *testing.T) {
	shouldRepair := autoprogrammingResidentShouldRepairV0(
		orquestamcp.MCPRunSupervisorToolInputV0{ResidentMode: true},
		CodexSupervisorResultV0{
			Last: CodexSupervisorRuntimeSnapshotV0{Status: CodexSupervisorRuntimeStoppedV0},
		},
		orquestamcp.MCPRunSupervisorToolResultV0{
			Estado: orquestamcp.MCPRunSupervisorEstadoOKV0,
			RunRef: "run-ref-soft-rail-non-blocking-001",
			EvidenceRefs: []string{
				"gate-status:non-blocked",
				"gate-summary:no_blocked_by_soft_rail",
				"gate-issue:file_too_large:non-blocking",
			},
		},
	)

	if shouldRepair {
		t.Fatalf("evidencia no bloqueante no debe disparar self-repair")
	}
}

func TestAutoprogrammingResidentModeV0NoBloqueaPorBlockedGenericoV0(t *testing.T) {
	for _, evidenceRef := range []string{
		"capacity_blocked",
		"provider_auth_blocked",
		"model_switch_blocked",
		"runtime_blocked",
		"quality-gate-ref-001#decision:blocked",
		"nota: blocked por cuota externa recuperable",
	} {
		shouldRepair := autoprogrammingResidentShouldRepairV0(
			orquestamcp.MCPRunSupervisorToolInputV0{ResidentMode: true},
			CodexSupervisorResultV0{
				Last: CodexSupervisorRuntimeSnapshotV0{Status: CodexSupervisorRuntimeStoppedV0},
			},
			orquestamcp.MCPRunSupervisorToolResultV0{
				Estado:       orquestamcp.MCPRunSupervisorEstadoOKV0,
				RunRef:       "run-ref-generic-blocked-001",
				EvidenceRefs: []string{evidenceRef},
			},
		)
		if shouldRepair {
			t.Fatalf("blocked generico no debe disparar self-repair: %s", evidenceRef)
		}
	}
}

func TestAutoprogrammingResidentModeV0AliasesDeRailBlandoNoBloqueanV0(t *testing.T) {
	for _, evidenceRef := range []string{
		"gate-issue:artifact_path_outside_write_set:docs/extra.md#policy_blocked",
		"review-gate-issue:outside-of-write-set:docs/extra.md#blocked",
		"quality_gate_issue:line_limit_exceeded:README.md#policy-blocked",
		"write-set-target-missing:web#blocked",
		"gate-issue:ack-pending-rail:provider#blocked",
		"gate-action:request_followup_review#blocked",
		"ack_pending_rail:token#blocked",
		"rail_blando:provider#policy_blocked",
		"file_too_large#policy_blocked",
	} {
		shouldRepair := autoprogrammingResidentShouldRepairV0(
			orquestamcp.MCPRunSupervisorToolInputV0{ResidentMode: true},
			CodexSupervisorResultV0{
				Last: CodexSupervisorRuntimeSnapshotV0{Status: CodexSupervisorRuntimeStoppedV0},
			},
			orquestamcp.MCPRunSupervisorToolResultV0{
				Estado:       orquestamcp.MCPRunSupervisorEstadoOKV0,
				RunRef:       "run-ref-soft-rail-alias-001",
				EvidenceRefs: []string{evidenceRef},
			},
		)
		if shouldRepair {
			t.Fatalf("rail blando alias no debe disparar self-repair: %s", evidenceRef)
		}
	}
}

func TestAutoprogrammingResidentModeV0BloqueoRealPreparaSelfRepairV0(t *testing.T) {
	shouldRepair := autoprogrammingResidentShouldRepairV0(
		orquestamcp.MCPRunSupervisorToolInputV0{ResidentMode: true},
		CodexSupervisorResultV0{
			Last: CodexSupervisorRuntimeSnapshotV0{Status: CodexSupervisorRuntimeStoppedV0},
		},
		orquestamcp.MCPRunSupervisorToolResultV0{
			Estado: orquestamcp.MCPRunSupervisorEstadoOKV0,
			RunRef: "run-ref-required-tests-failed-001",
			EvidenceRefs: []string{
				"quality-gate-ref-001#decision:blocked",
				"required-tests-failed",
			},
		},
	)

	if !shouldRepair {
		t.Fatalf("bloqueo real debe conservar self-repair")
	}
}

func TestAutoprogrammingResidentModeV0RailBlandoNoOcultaBloqueoRealV0(t *testing.T) {
	shouldRepair := autoprogrammingResidentShouldRepairV0(
		orquestamcp.MCPRunSupervisorToolInputV0{ResidentMode: true},
		CodexSupervisorResultV0{
			Last: CodexSupervisorRuntimeSnapshotV0{Status: CodexSupervisorRuntimeStoppedV0},
		},
		orquestamcp.MCPRunSupervisorToolResultV0{
			Estado: orquestamcp.MCPRunSupervisorEstadoOKV0,
			RunRef: "run-ref-soft-rail-hard-failure-001",
			EvidenceRefs: []string{
				"gate-issue:file_too_large",
				"gate-issue:required_test_failed",
			},
		},
	)

	if !shouldRepair {
		t.Fatalf("bloqueo real no debe quedar oculto por rail blando")
	}
}

func TestAutoprogrammingResidentModeV0RailBlandoMixtoNoOcultaAckBloqueanteV0(t *testing.T) {
	shouldRepair := autoprogrammingResidentShouldRepairV0(
		orquestamcp.MCPRunSupervisorToolInputV0{ResidentMode: true},
		CodexSupervisorResultV0{
			Last: CodexSupervisorRuntimeSnapshotV0{Status: CodexSupervisorRuntimeStoppedV0},
		},
		orquestamcp.MCPRunSupervisorToolResultV0{
			Estado: orquestamcp.MCPRunSupervisorEstadoOKV0,
			RunRef: "run-ref-soft-rail-mixed-blocking-001",
			EvidenceRefs: []string{
				"gate-issue:file_too_large#gate-issue:ack_missing#blocked",
			},
		},
	)

	if !shouldRepair {
		t.Fatalf("ack_missing mixto con rail blando debe conservar self-repair")
	}
}
