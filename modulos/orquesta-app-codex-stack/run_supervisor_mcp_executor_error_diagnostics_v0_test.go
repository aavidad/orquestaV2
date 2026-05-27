package orquestaappcodexstack

import (
	"errors"
	"strings"
	"testing"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
)

func TestCodexStackRunSupervisorDrainRequestV0AplicaPresupuestoConservadorPorDefecto(t *testing.T) {
	request := codexStackRunSupervisorDrainRequestV0(orquestamcp.MCPRunSupervisorToolInputV0{
		RunRef:   "run-ref-budget-default-001",
		MaxTicks: 1,
	})

	if request.MaxBursts != 1 ||
		request.MaxStepsPerBurst != 1 ||
		request.MaxDispatchesPerWait != 1 ||
		request.MaxCommands != 1 ||
		request.MaxOutboxPerCycle != 1 ||
		request.MaxDecisionCycles != 1 ||
		request.MaxExternalWaits != 0 {
		t.Fatalf("presupuesto por defecto no acotado: %+v", request)
	}
}

func TestCodexStackRunSupervisorDrainRequestV0ConservaPresupuestoExplicito(t *testing.T) {
	request := codexStackRunSupervisorDrainRequestV0(orquestamcp.MCPRunSupervisorToolInputV0{
		RunRef:               "run-ref-budget-explicit-001",
		MaxBursts:            3,
		MaxStepsPerBurst:     4,
		MaxDispatchesPerWait: 5,
		MaxCommands:          6,
		MaxOutboxPerCycle:    7,
		MaxDecisionCycles:    8,
		MaxExternalWaits:     9,
	})

	if request.MaxBursts != 3 ||
		request.MaxStepsPerBurst != 4 ||
		request.MaxDispatchesPerWait != 5 ||
		request.MaxCommands != 6 ||
		request.MaxOutboxPerCycle != 7 ||
		request.MaxDecisionCycles != 8 ||
		request.MaxExternalWaits != 9 {
		t.Fatalf("presupuesto explicito no conservado: %+v", request)
	}
}

func TestCodexStackRunSupervisorCommandV0AplicaPresupuestoConservadorPorDefecto(t *testing.T) {
	command := codexStackRunSupervisorCommandV0(orquestamcp.MCPRunSupervisorToolInputV0{
		QueueRef: "queue-ref-budget-default-001",
	})

	if command.DrainLimits.MaxBursts != 1 ||
		command.DrainLimits.MaxStepsPerBurst != 1 ||
		command.DrainLimits.MaxDispatchesPerWait != 1 ||
		command.DrainLimits.MaxCommands != 1 ||
		command.DrainLimits.MaxOutboxPerCycle != 1 ||
		command.DrainLimits.MaxDecisionCycles != 1 ||
		command.DrainLimits.MaxExternalWaits != 0 {
		t.Fatalf("presupuesto por defecto del coordinador no acotado: %+v", command.DrainLimits)
	}
}

func TestCodexStackRunSupervisorErrorResultMCPV0ExponeDiagnosticoPublicoDelDrain(t *testing.T) {
	partial := CodexSupervisorResultV0{
		StopReason: CodexSupervisorStopRuntimeErrorV0,
		Ticks:      1,
		Last: CodexSupervisorRuntimeSnapshotV0{
			Status:       CodexSupervisorRuntimeFailedV0,
			SessionRef:   "run-ref-diagnostics-001",
			EvidenceRefs: []string{"evidence-ref-drain"},
			Diagnostics: []orquestaruncoordinator.RunDrainDiagnosticV0{{
				Kind:        "outbox_batch_dispatch_error",
				Status:      "error",
				RunRef:      "run-ref-diagnostics-001",
				Error:       "codex launch failed: /home/alberto/secreto/token.txt",
				MessageType: "LaunchRuntimeAgent",
				MessageID:   "message-ref-001",
				Issues:      1,
			}},
		},
	}

	result := codexStackRunSupervisorErrorResultMCPV0(
		orquestamcp.MCPRunSupervisorToolInputV0{},
		partial,
		errors.New("runtime error"),
	)

	if len(result.Diagnostics) == 0 {
		t.Fatalf("diagnostics vacios: %+v", result)
	}
	if result.Diagnostics[0].Code != "outbox_batch_dispatch_error" {
		t.Fatalf("diagnostic code=%q", result.Diagnostics[0].Code)
	}
	if !strings.Contains(result.Diagnostics[0].Message, "LaunchRuntimeAgent") ||
		!strings.Contains(result.Diagnostics[0].Message, "capacity-missing") {
		t.Fatalf("diagnostic message sin contexto publico: %q", result.Diagnostics[0].Message)
	}
	if strings.Contains(result.Diagnostics[0].Message, "/home/alberto") ||
		strings.Contains(result.Diagnostics[0].Message, "token.txt") {
		t.Fatalf("diagnostic filtra path local sensible: %q", result.Diagnostics[0].Message)
	}
}
