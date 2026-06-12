package orquestaappcodexstack

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
)

func TestStackDrainDiagnosticsV0ConservaOutboxYPendientesParaAuditoria(t *testing.T) {
	result := orquestacionnucleoapp.ManagedProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusWaitUnhandledOutboxV0,
		Attempts: []orquestacionnucleoapp.ManagedProgressiveLoopAttemptV0{{
			AttemptNumber: 1,
			Result: orquestacionnucleoapp.ProgressiveLoopResultV0{
				Status: orquestacionnucleoapp.ProgressiveLoopStatusWaitUnhandledOutboxV0,
				Run:    orquestacoreworkflow.OrchestrationRunV0{RunID: "run-ref-001"},
				BatchDispatches: []orquestacionnucleoapp.OutboxDispatchBatchRunResultV0{{
					Status:            orquestacionnucleoapp.OutboxDispatchBatchRunNoPendingV0,
					RunRef:            "run-ref-001",
					TargetPort:        "agent_launcher",
					MessageType:       "LaunchRuntimeAgent",
					PlannedCount:      0,
					Issues:            1,
					ClaimedMessageIDs: []string{"outbox-launch-001"},
				}},
				Dispatches: []orquestacionnucleoapp.OutboxDispatchOnceResultV0{{
					Status:      "no_pending",
					RunRef:      "run-ref-001",
					TargetPort:  "capacity",
					MessageType: "RequestCapacityDecision",
					Issues:      0,
				}},
				FirstPendingRefs:   []string{"outbox-ref-001"},
				FirstPendingCount:  1,
				PendingOutboxRefs:  []string{"outbox-ref-001"},
				PendingOutboxCount: 1,
			},
		}},
		Final: orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status:             orquestacionnucleoapp.ProgressiveLoopStatusWaitUnhandledOutboxV0,
			Run:                orquestacoreworkflow.OrchestrationRunV0{RunID: "run-ref-001"},
			PendingOutboxRefs:  []string{"outbox-ref-001"},
			PendingOutboxCount: 1,
		},
	}

	diagnostics := stackDrainDiagnosticsV0(result)
	if !stackDiagnosticsHasV0(diagnostics, "outbox_batch_dispatch", "LaunchRuntimeAgent") ||
		!stackDiagnosticsHasV0(diagnostics, "outbox_dispatch", "RequestCapacityDecision") ||
		!stackDiagnosticsHasV0(diagnostics, "drain_final", "") {
		t.Fatalf("diagnostics=%+v", diagnostics)
	}
	if diagnostics[len(diagnostics)-1].PendingOutboxCount != 1 ||
		diagnostics[len(diagnostics)-1].PendingOutboxRefs[0] != "outbox-ref-001" {
		t.Fatalf("final=%+v", diagnostics[len(diagnostics)-1])
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Kind == "outbox_batch_dispatch" &&
			len(diagnostic.ClaimedMessages) != 1 {
			t.Fatalf("batch diagnostic sin claimed messages: %+v", diagnostic)
		}
	}
}

func TestStackDrainDiagnosticsWithErrorV0ConservaMensajeYError(t *testing.T) {
	result := orquestacionnucleoapp.ManagedProgressiveLoopResultV0{
		Attempts: []orquestacionnucleoapp.ManagedProgressiveLoopAttemptV0{{
			AttemptNumber: 2,
			Result: orquestacionnucleoapp.ProgressiveLoopResultV0{
				Run: orquestacoreworkflow.OrchestrationRunV0{RunID: "run-ref-error-001"},
				BatchDispatches: []orquestacionnucleoapp.OutboxDispatchBatchRunResultV0{{
					Status:            orquestacionnucleoapp.OutboxDispatchBatchRunExecutionFailedV0,
					RunRef:            "run-ref-error-001",
					TargetPort:        "agent_launcher",
					MessageType:       "LaunchRuntimeAgent",
					ClaimedMessageIDs: []string{"outbox-launch-error-001"},
					Issues:            1,
				}},
			},
		}},
		Final: orquestacionnucleoapp.ProgressiveLoopResultV0{
			Run: orquestacoreworkflow.OrchestrationRunV0{RunID: "run-ref-error-001"},
		},
	}

	diagnostics := stackDrainDiagnosticsWithErrorV0(
		result,
		stackDrainDiagnosticsV0(result),
		"run-ref-error-001",
		"transicion_invalida: payload.agent_request_id",
	)
	if !stackDiagnosticsHasErrorV0(diagnostics, "drain_error", "") ||
		!stackDiagnosticsHasErrorV0(diagnostics, "outbox_batch_dispatch_error", "outbox-launch-error-001") {
		t.Fatalf("diagnostics=%+v", diagnostics)
	}
}

func TestStackDrainDiagnosticsWithErrorV0UsaRunRefDeRequestSiFinalVacio(t *testing.T) {
	diagnostics := stackDrainDiagnosticsWithErrorV0(
		orquestacionnucleoapp.ManagedProgressiveLoopResultV0{},
		nil,
		"run-ref-from-request-001",
		"drain fallo antes de cargar run",
	)
	if len(diagnostics) != 1 ||
		diagnostics[0].Kind != "drain_error" ||
		diagnostics[0].RunRef != "run-ref-from-request-001" ||
		diagnostics[0].Error == "" {
		t.Fatalf("diagnostic sin run_ref de request: %+v", diagnostics)
	}
}

func TestCodexStackQueuedRecoveryAdvisoryErrorV0SoloPayloadRecuperable(t *testing.T) {
	if !codexStackQueuedRecoveryAdvisoryErrorV0(orquestacoreworkflow.OrchestrationCommandErrorV0{
		Code:  orquestacoreworkflow.ErrTransicionInvalidaV0,
		Field: "payload.phase_id",
	}) {
		t.Fatalf("payload transicion_invalida debe ser advisory en reconciliacion")
	}
	if !codexStackQueuedRecoveryAdvisoryErrorV0(orquestacionnucleoapp.ErrorV0{
		Code:  orquestacionnucleoapp.ErrNucleoOrquestacionInvalidoV0,
		Field: "payload",
	}) {
		t.Fatalf("payload nucleo_orquestacion_invalido debe ser advisory en reconciliacion")
	}
	if codexStackQueuedRecoveryAdvisoryErrorV0(orquestacoreworkflow.OrchestrationCommandErrorV0{
		Code:  orquestacoreworkflow.ErrTransicionInvalidaV0,
		Field: "run_ref",
	}) {
		t.Fatalf("refs imposibles no deben degradarse a advisory")
	}
}

func stackDiagnosticsHasV0(
	diagnostics []orquestaruncoordinator.RunDrainDiagnosticV0,
	kind string,
	messageType string,
) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Kind != kind {
			continue
		}
		if messageType == "" || diagnostic.MessageType == messageType {
			return true
		}
	}
	return false
}

func stackDiagnosticsHasErrorV0(
	diagnostics []orquestaruncoordinator.RunDrainDiagnosticV0,
	kind string,
	messageID string,
) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Kind != kind || diagnostic.Error == "" {
			continue
		}
		if messageID == "" || diagnostic.MessageID == messageID {
			return true
		}
	}
	return false
}
