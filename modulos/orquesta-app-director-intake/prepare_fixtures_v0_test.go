package orquestaappdirectorintake

import (
	"context"
	"testing"
	"time"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestacionnucleoapp "orquesta/orquestacionnucleoapp"
)

func mustPrepareDirectorIntakeForTestV0(t *testing.T) AppDirectorIntakePreparedV0 {
	t.Helper()
	return mustPrepareDirectorIntakeWithSpecForTestV0(
		t,
		"run-app-director-001",
		validFactoryAppSpecForDirectorIntakeTestV0(t),
	)
}

func mustPrepareDirectorIntakeWithSpecForTestV0(
	t *testing.T,
	runRef string,
	spec orquestafactory.AppSpecV0,
) AppDirectorIntakePreparedV0 {
	t.Helper()
	prepared, err := PrepareAppDirectorIntakeV0(PrepareAppDirectorIntakeRequestV0{
		RunRef:        runRef,
		ProjectRef:    "project-app-director-001",
		OccurredAt:    "2026-05-09T22:00:00Z",
		CorrelationID: "corr-app-director-001",
		RequestedBy:   "orquesta-app-director-intake-test",
		AppSpec:       spec,
	})
	if err != nil {
		t.Fatalf("PrepareAppDirectorIntakeV0: %v", err)
	}
	return prepared
}

func validFactoryAppSpecForDirectorIntakeTestV0(t *testing.T) orquestafactory.AppSpecV0 {
	t.Helper()
	return validFactoryAppSpecWithRequestIDForDirectorIntakeTestV0(t, "request-ref-app-director-001")
}

func validFactoryAppSpecWithRequestIDForDirectorIntakeTestV0(
	t *testing.T,
	requestID string,
) orquestafactory.AppSpecV0 {
	t.Helper()
	observability := true
	req := validFactoryAppSpecRequestForDirectorIntakeTestV0()
	req.RequestID = requestID
	req.Calidad.Observabilidad = &observability
	spec, issues := orquestafactory.SolicitarNuevaAppV0(req, time.Date(2026, 5, 9, 22, 0, 0, 0, time.UTC))
	if len(issues) != 0 {
		t.Fatalf("SolicitarNuevaAppV0 issues: %+v", issues)
	}
	return spec
}

func validFactoryAppSpecRequestForDirectorIntakeTestV0() orquestafactory.AppSpecRequestV0 {
	return orquestafactory.AppSpecRequestV0{
		SchemaVersion: orquestafactory.AppSpecRequestSchemaV0,
		RequestID:     "request-ref-app-director-001",
		Source:        "orquesta-web",
		Locale:        "es-ES",
		Nombre:        "Agenda",
		Objetivo:      "Gestionar contactos y citas desde una API y una web.",
		TipoApp:       "mixed",
		PreferenciasTecnicas: orquestafactory.PreferenciasTecnicasV0{
			Lenguaje:     "go",
			Arquitectura: "hexagonal",
		},
		Calidad: orquestafactory.CalidadRequestV0{
			Pruebas:       "media",
			Accesibilidad: "basica",
		},
	}
}

func mustBuildDirectorIntakeCandidatesForTestV0(
	t *testing.T,
	prepared AppDirectorIntakePreparedV0,
) orquestacionnucleoapp.SchedulerCandidateSetV0 {
	t.Helper()
	candidates, err := prepared.CandidateProvider.BuildSchedulerCandidatesV0(
		context.Background(),
		orquestacionnucleoapp.SchedulerCandidateRequestV0{
			Run:           prepared.Run,
			OccurredAt:    "2026-05-09T22:00:00Z",
			CorrelationID: "corr-app-director-candidates-001",
			EvidenceRefs:  []string{"evidence-ref-app-director-candidates-001"},
		},
	)
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	return candidates
}

func directorIntakeCapacityDispatcherForTestV0(
	store *orquestacionnucleoapp.InMemoryRunStoreV0,
	sink *orquestacionnucleoapp.InMemoryEventSinkV0,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
) orquestacionnucleoapp.OutboxDispatcherBindingV0 {
	return orquestacionnucleoapp.OutboxDispatcherBindingV0{
		TargetPort: orquestacoreworkflow.OutboxTargetCapacityV0,
		Reader:     ledger,
		Claimer:    ledger,
		Executor: orquestacionnucleoapp.CapacityDecisionExecutorV0{
			RunStore:        store,
			EventSink:       sink,
			Tier:            orquestacoreworkflow.OrchestrationCapacityHighV0,
			ReasoningEffort: orquestacoreworkflow.OrchestrationCapacityHighV0,
			OccurredAt:      "2026-05-09T22:02:00Z",
			CorrelationID:   "corr-app-director-capacity-001",
			RequestedBy:     "orquesta-app-director-intake-test",
		},
		Acker: ledger,
	}
}

func directorIntakeAgentLauncherDispatcherForTestV0(
	store *orquestacionnucleoapp.InMemoryRunStoreV0,
	sink *orquestacionnucleoapp.InMemoryEventSinkV0,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
) orquestacionnucleoapp.OutboxDispatcherBindingV0 {
	return orquestacionnucleoapp.OutboxDispatcherBindingV0{
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		Reader:     ledger,
		Claimer:    ledger,
		Executor: orquestacionnucleoapp.AgentLauncherExecutorV0{
			RunStore:      store,
			EventSink:     sink,
			Launcher:      orquestacionnucleoapp.NewFakeLifecycleAgentLauncherV0(),
			OccurredAt:    "2026-05-09T22:03:00Z",
			CorrelationID: "corr-app-director-agent-001",
			RequestedBy:   "orquesta-app-director-intake-test",
		},
		Acker: ledger,
	}
}
