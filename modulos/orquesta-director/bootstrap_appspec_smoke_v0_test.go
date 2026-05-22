package orquestadirector

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestSmokeOrquestaAppSimpleV0ArrancaRunYAbreDescubrimiento(t *testing.T) {
	cmd := smokeAppSimpleCommandV0(t)

	result, err := BootstrapProyectoDesdeAppSpecV0(cmd)
	if err != nil {
		t.Fatalf("BootstrapProyectoDesdeAppSpecV0: %v", err)
	}

	assertSmokeBootstrapResultV0(t, result)

	events := append([]orquestacoreworkflow.OrchestrationEventV0{}, result.WorkflowResult.Events...)
	run := replaySmokeRunV0(t, events)
	if run.Status != orquestacoreworkflow.OrchestrationRunStatusActiveV0 ||
		run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseDescubrimientoV0 {
		t.Fatalf("run inicial inesperado: status=%q phase=%q", run.Status, run.CurrentPhase)
	}

	open := mustSmokeOpenPhaseCommandV0(t, run, orquestacoreworkflow.OrchestrationPhaseDescubrimientoV0)
	openResult, err := orquestacoreworkflow.HandleCommandV0(run, open)
	if err != nil {
		t.Fatalf("HandleCommandV0 OpenPhase: %v", err)
	}
	if len(openResult.Events) != 1 || openResult.Events[0].EventType != orquestacoreworkflow.OrchestrationEventPhaseOpenedV0 {
		t.Fatalf("open events inesperados: %+v", openResult.Events)
	}
	if len(openResult.Outbox) != 0 {
		t.Fatalf("OpenPhase no debe emitir outbox: %+v", openResult.Outbox)
	}

	events = append(events, openResult.Events...)
	run = replaySmokeRunV0(t, events)
	if !smokePhaseIsActiveV0(run, orquestacoreworkflow.OrchestrationPhaseDescubrimientoV0) {
		t.Fatalf("descubrimiento no quedo activa: %+v", run.Phases)
	}

	t.Logf(
		"orquesta smoke simple: app=%q estado=%s fase=%s fases=%d microtareas=%d contratos=%d eventos=%d outbox=%d",
		cmd.AppSpec.App.Nombre,
		result.RegistroAceptado.Estado,
		run.CurrentPhase,
		result.RegistroAceptado.FasesIniciales,
		result.RegistroAceptado.Microtareas,
		result.RegistroAceptado.Contratos,
		len(events),
		len(openResult.Outbox),
	)
}

func smokeAppSimpleCommandV0(t *testing.T) BootstrapProyectoDesdeAppSpecCommandV0 {
	t.Helper()
	specID := "appspec-lista-simple-001"
	return BootstrapProyectoDesdeAppSpecCommandV0{
		IdempotencyKey: "idem-smoke-app-simple-001",
		AppSpec:        neutralRegistrarAppSpecV0(specID, "req-smoke-app-simple-001", "Lista simple de tareas", "2026-05-04T13:00:00Z"),
		Backlog:        neutralRegistrarBacklogV0(specID, "lista-simple"),
		RequestedBy:    "director",
		OccurredAt:     "2026-05-04T13:05:00Z",
		RequestID:      "req-smoke-app-simple-001",
		CorrelationID:  "corr-smoke-app-simple-001",
	}
}

func assertSmokeBootstrapResultV0(t *testing.T, result BootstrapProyectoDesdeAppSpecResultV0) {
	t.Helper()
	if result.RegistroAceptado.Estado != "borrador" {
		t.Fatalf("estado=%q, want borrador", result.RegistroAceptado.Estado)
	}
	if result.RegistroAceptado.FasesIniciales != 5 ||
		result.RegistroAceptado.Microtareas != 5 ||
		result.RegistroAceptado.Contratos != 5 {
		t.Fatalf("conteos inesperados: %+v", result.RegistroAceptado)
	}
	if result.StartRunCommand.CommandType != orquestacoreworkflow.OrchestrationCommandStartRunV0 {
		t.Fatalf("start command=%q", result.StartRunCommand.CommandType)
	}
	if len(result.WorkflowResult.Events) != 1 ||
		result.WorkflowResult.Events[0].EventType != orquestacoreworkflow.OrchestrationEventRunStartedV0 {
		t.Fatalf("workflow result inesperado: %+v", result.WorkflowResult)
	}
	if len(result.WorkflowResult.Outbox) != 0 {
		t.Fatalf("bootstrap no debe emitir outbox: %+v", result.WorkflowResult.Outbox)
	}
}

func mustSmokeOpenPhaseCommandV0(
	t *testing.T,
	run orquestacoreworkflow.OrchestrationRunV0,
	phase orquestacoreworkflow.OrchestrationPhaseIDV0,
) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	cmd, err := orquestacoreworkflow.NewOpenPhaseCommandV0(
		orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-smoke-open-descubrimiento-001",
			RunID:          run.RunID,
			IdempotencyKey: "idem-smoke-open-descubrimiento-001",
			CorrelationID:  "corr-smoke-app-simple-001",
			RequestedBy:    "director",
			OccurredAt:     "2026-05-04T13:06:00Z",
		},
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: string(phase),
			Reason:  "Comenzar alcance inicial de la app simple.",
		},
	)
	if err != nil {
		t.Fatalf("NewOpenPhaseCommandV0: %v", err)
	}
	return cmd
}

func replaySmokeRunV0(t *testing.T, events []orquestacoreworkflow.OrchestrationEventV0) orquestacoreworkflow.OrchestrationRunV0 {
	t.Helper()
	run, err := orquestacoreworkflow.ReplayDurableEventsV0(events)
	if err != nil {
		t.Fatalf("ReplayDurableEventsV0: %v", err)
	}
	return run
}

func smokePhaseIsActiveV0(run orquestacoreworkflow.OrchestrationRunV0, phaseID orquestacoreworkflow.OrchestrationPhaseIDV0) bool {
	for _, phase := range run.Phases {
		if phase.ID == phaseID {
			return phase.Status == orquestacoreworkflow.OrchestrationPhaseStatusActiveV0
		}
	}
	return false
}
