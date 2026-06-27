package orquestacli

import (
	"testing"
	"time"

	orquestacore "orquesta/modulos/orquesta-core"
	orquestadirector "orquesta/modulos/orquesta-director"
	orquestafactory "orquesta/modulos/orquesta-factory"
)

func mustNewBootstrapAppSpecCliClientV0(t *testing.T, serverURL string, timeout time.Duration) *BootstrapAppSpecCliClientV0 {
	t.Helper()
	client, err := NewBootstrapAppSpecCliClientV0(serverURL, timeout)
	if err != nil {
		t.Fatalf("NewBootstrapAppSpecCliClientV0: %v", err)
	}
	return client
}

func invocationForBootstrapAppSpecCliV0(serverURL string) CliInvocationContextV0 {
	return CliInvocationContextV0{
		RequestID:      "req-bootstrap-test",
		CorrelationID:  "corr-bootstrap-test",
		IdempotencyKey: "idem-bootstrap-test",
		ServerURL:      serverURL,
		Timeout:        time.Second,
		OutputFormat:   CliOutputFormatJSONV0,
	}
}

func minimalBootstrapAppSpecCommandForCliV0(t *testing.T) orquestadirector.BootstrapProyectoDesdeAppSpecCommandV0 {
	t.Helper()
	req := minimalAppSpecRequestForCliV0()
	req.RequestID = "req-bootstrap-test"
	req.Source = SolicitarNuevaAppCliSourceV0
	spec, issues := orquestafactory.SolicitarNuevaAppV0(req, fixedSolicitarNuevaAppCliClockV0())
	if len(issues) > 0 {
		t.Fatalf("fixture spec invalida: %+v", issues)
	}
	backlog, issues := orquestafactory.GenerarBacklogInicialPropuestoV0(spec)
	if len(issues) > 0 {
		t.Fatalf("fixture backlog invalido: %+v", issues)
	}
	return orquestadirector.BootstrapProyectoDesdeAppSpecCommandV0{
		AppSpec:        registrarAppSpecFromFactoryForBootstrapCliTestV0(spec),
		Backlog:        registrarBacklogFromFactoryForBootstrapCliTestV0(backlog),
		RequestedBy:    "cli-test",
		OccurredAt:     "2026-05-04T10:30:00Z",
		IdempotencyKey: "idem-bootstrap-test",
	}
}

func registrarAppSpecFromFactoryForBootstrapCliTestV0(spec orquestafactory.AppSpecV0) orquestacore.RegistrarAppSpecV0 {
	return orquestacore.RegistrarAppSpecV0{
		SchemaVersion: spec.SchemaVersion,
		SpecID:        spec.SpecID,
		RequestID:     spec.RequestID,
		CreatedAt:     spec.CreatedAt,
		App: orquestacore.RegistrarAppInfoV0{
			Nombre: spec.App.Nombre,
		},
		Validation: orquestacore.RegistrarValidationSummaryV0{
			Estado: spec.Validation.Estado,
		},
	}
}

func registrarBacklogFromFactoryForBootstrapCliTestV0(backlog orquestafactory.BacklogInicialPropuestoV0) orquestacore.RegistrarBacklogInicialV0 {
	return orquestacore.RegistrarBacklogInicialV0{
		SchemaVersion:       backlog.SchemaVersion,
		SpecID:              backlog.SpecID,
		Fases:               registrarFasesFromFactoryForBootstrapCliTestV0(backlog.Fases),
		Microtareas:         registrarMicrotareasFromFactoryForBootstrapCliTestV0(backlog.Microtareas),
		ContratosRequeridos: append([]string(nil), backlog.ContratosRequeridos...),
		Riesgos:             append([]string(nil), backlog.Riesgos...),
		PreguntasAbiertas:   append([]string(nil), backlog.PreguntasAbiertas...),
	}
}

func registrarFasesFromFactoryForBootstrapCliTestV0(fases []orquestafactory.FaseInicialV0) []orquestacore.RegistrarFaseInicialV0 {
	result := make([]orquestacore.RegistrarFaseInicialV0, 0, len(fases))
	for _, fase := range fases {
		result = append(result, orquestacore.RegistrarFaseInicialV0{
			ID:       fase.ID,
			Nombre:   fase.Nombre,
			Objetivo: fase.Objetivo,
			Orden:    fase.Orden,
		})
	}
	return result
}

func registrarMicrotareasFromFactoryForBootstrapCliTestV0(tasks []orquestafactory.MicrotareaPropuestaV0) []orquestacore.RegistrarMicrotareaV0 {
	result := make([]orquestacore.RegistrarMicrotareaV0, 0, len(tasks))
	for _, task := range tasks {
		result = append(result, orquestacore.RegistrarMicrotareaV0{
			ID:               task.ID,
			Fase:             task.Fase,
			ModuloSugerido:   task.ModuloSugerido,
			Objetivo:         task.Objetivo,
			WriteSetPrevisto: append([]string(nil), task.WriteSetPrevisto...),
			Contrato:         task.Contrato,
			Validacion:       task.Validacion,
			Bloqueos:         append([]string(nil), task.Bloqueos...),
		})
	}
	return result
}
