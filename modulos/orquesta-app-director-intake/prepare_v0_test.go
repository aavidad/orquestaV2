package orquestaappdirectorintake

import (
	"context"
	"strings"
	"testing"
	"time"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestPrepareAppDirectorIntakeV0CreatesBrainstormRun(t *testing.T) {
	spec := validFactoryAppSpecForDirectorIntakeTestV0(t)

	prepared, err := PrepareAppDirectorIntakeV0(PrepareAppDirectorIntakeRequestV0{
		AppSpec: spec,
	})
	if err != nil {
		t.Fatalf("PrepareAppDirectorIntakeV0: %v", err)
	}
	if prepared.SchemaVersion != AppDirectorIntakePreparedSchemaVersionV0 {
		t.Fatalf("schema=%q", prepared.SchemaVersion)
	}
	if prepared.Run.Status != orquestacoreworkflow.OrchestrationRunStatusActiveV0 ||
		prepared.Run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0 {
		t.Fatalf("run inicial=%+v", prepared.Run)
	}
	if prepared.Run.AppSpecRef != spec.SpecID {
		t.Fatalf("app_spec_ref=%q want %q", prepared.Run.AppSpecRef, spec.SpecID)
	}
	if len(prepared.InitialEvents) != 3 {
		t.Fatalf("initial_events=%d events=%+v", len(prepared.InitialEvents), prepared.InitialEvents)
	}
	if !directorIntakeStringInSetV0(prepared.Run.Brainstorms, prepared.DirectorTask.BrainstormRef) {
		t.Fatalf("brainstorms=%v task=%+v", prepared.Run.Brainstorms, prepared.DirectorTask)
	}
	if prepared.DirectorTask.TaskRef == "" || len(prepared.Run.Tasks) != 0 {
		t.Fatalf("director_task no debe entrar en Run.Tasks: task=%+v run_tasks=%v", prepared.DirectorTask, prepared.Run.Tasks)
	}
	if prepared.DirectorTask.Capacity != orquestacoreworkflow.OrchestrationCapacityXHighV0 {
		t.Fatalf("capacity=%q", prepared.DirectorTask.Capacity)
	}
}

func TestPrepareAppDirectorIntakeV0RefsAisladasPorSpecID(t *testing.T) {
	first := mustPrepareDirectorIntakeWithSpecForTestV0(
		t,
		"run-app-director-ref-001",
		validFactoryAppSpecWithRequestIDForDirectorIntakeTestV0(t, "request-ref-app-director-ref-001"),
	)
	second := mustPrepareDirectorIntakeWithSpecForTestV0(
		t,
		"run-app-director-ref-002",
		validFactoryAppSpecWithRequestIDForDirectorIntakeTestV0(t, "request-ref-app-director-ref-002"),
	)

	if first.DirectorTask.AgentRequestID == second.DirectorTask.AgentRequestID ||
		first.DirectorTask.TaskRef == second.DirectorTask.TaskRef ||
		first.DirectorTask.BrainstormRef == second.DirectorTask.BrainstormRef {
		t.Fatalf("refs colisionan: first=%+v second=%+v", first.DirectorTask, second.DirectorTask)
	}
}

func TestPrepareAppDirectorIntakeV0PropagaPoliticaDePeticionEnSummary(t *testing.T) {
	req := validFactoryAppSpecRequestForDirectorIntakeTestV0()
	req.RequestKind = orquestafactory.RequestKindDocumentarAppV0
	req.ExecutionMode = orquestafactory.ExecutionModeDebugV0
	spec, issues := orquestafactory.SolicitarNuevaAppV0(req, time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC))
	if len(issues) > 0 {
		t.Fatalf("build spec: %+v", issues)
	}

	prepared, err := PrepareAppDirectorIntakeV0(PrepareAppDirectorIntakeRequestV0{AppSpec: spec})
	if err != nil {
		t.Fatalf("PrepareAppDirectorIntakeV0: %v", err)
	}

	if !strings.Contains(prepared.DirectorTask.Summary, "request_kind=documentar_app") ||
		!strings.Contains(prepared.DirectorTask.Summary, "execution_mode=debug") {
		t.Fatalf("summary no transporta politica: %q", prepared.DirectorTask.Summary)
	}
}

func TestPrepareAppDirectorIntakeV0PropagaContextoFuncionalEnSummary(t *testing.T) {
	req := validFactoryAppSpecRequestForDirectorIntakeTestV0()
	req.Nombre = "Inventario Review"
	req.Objetivo = "Crear API REST en Go para gestionar articulos con HTML minimo."
	req.Descripcion = "Arquitectura hexagonal y persistencia en memoria detras de puerto."
	req.TipoApp = "mixed"
	req.Plataformas = []string{"web", "api"}
	req.Datos.NecesidadFuncional = "Gestionar articulos sin servicios externos."
	spec, issues := orquestafactory.SolicitarNuevaAppV0(req, time.Date(2026, 5, 22, 9, 20, 0, 0, time.UTC))
	if len(issues) > 0 {
		t.Fatalf("build spec: %+v", issues)
	}

	prepared, err := PrepareAppDirectorIntakeV0(PrepareAppDirectorIntakeRequestV0{AppSpec: spec})
	if err != nil {
		t.Fatalf("PrepareAppDirectorIntakeV0: %v", err)
	}

	for _, want := range []string{
		"app=Inventario Review",
		"objetivo=Crear API REST en Go para gestionar articulos con HTML minimo.",
		"descripcion=Arquitectura hexagonal y persistencia en memoria detras de puerto.",
		"tipo=mixed",
		"plataformas=web,api",
		"datos=Gestionar articulos sin servicios externos.",
		"request_kind=crear_app_completa",
		"execution_mode=normal",
	} {
		if !strings.Contains(prepared.DirectorTask.Summary, want) {
			t.Fatalf("summary no contiene %q:\n%s", want, prepared.DirectorTask.Summary)
		}
	}
}

func TestPrepareAppDirectorIntakeV0DirectorNormalPuedeCrearManuales(t *testing.T) {
	req := validFactoryAppSpecRequestForDirectorIntakeTestV0()
	req.RequestKind = orquestafactory.RequestKindDocumentarAppV0
	req.ExecutionMode = orquestafactory.ExecutionModeNormalV0
	spec, issues := orquestafactory.SolicitarNuevaAppV0(req, time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC))
	if len(issues) > 0 {
		t.Fatalf("build spec: %+v", issues)
	}

	prepared, err := PrepareAppDirectorIntakeV0(PrepareAppDirectorIntakeRequestV0{AppSpec: spec})
	if err != nil {
		t.Fatalf("PrepareAppDirectorIntakeV0: %v", err)
	}

	if prepared.DirectorTask.Capacity != orquestacoreworkflow.OrchestrationCapacityXHighV0 {
		t.Fatalf("capacity=%q", prepared.DirectorTask.Capacity)
	}
	for _, path := range []string{
		"docs/manual_usuario.md",
		"docs/manual_desarrollador.md",
		"docs/manual_sistemas_deploy.md",
		"docs/decisiones.md",
		"docs/pruebas.md",
		"docs/pendientes.md",
	} {
		if !directorIntakeStringInSetV0(prepared.DirectorTask.WriteSet, path) {
			t.Fatalf("write_set no contiene %s: %+v", path, prepared.DirectorTask.WriteSet)
		}
	}
}

func TestPrepareAppDirectorIntakeV0RejectsSpecNoValidada(t *testing.T) {
	spec := validFactoryAppSpecForDirectorIntakeTestV0(t)
	spec.Validation.Estado = "provisional"

	if _, err := PrepareAppDirectorIntakeV0(PrepareAppDirectorIntakeRequestV0{
		RunRef:     "run-app-director-invalid-001",
		OccurredAt: "2026-05-09T22:00:00Z",
		AppSpec:    spec,
	}); err == nil {
		t.Fatalf("esperaba error")
	}
}

func TestAppDirectorCandidateProviderV0EmitsDirectorAgent(t *testing.T) {
	prepared := mustPrepareDirectorIntakeForTestV0(t)

	candidates := mustBuildDirectorIntakeCandidatesForTestV0(t, prepared)
	if len(candidates.WorkCandidates) != 1 {
		t.Fatalf("work candidates=%+v", candidates.WorkCandidates)
	}
	candidate := candidates.WorkCandidates[0]
	if candidate.AgentCandidate == nil ||
		candidate.AgentCandidate.Payload.AgentRequestID != prepared.DirectorTask.AgentRequestID ||
		candidate.AgentCandidate.Payload.Role != "director" {
		t.Fatalf("agent candidate=%+v", candidate.AgentCandidate)
	}
	if candidate.CapacityCandidate == nil ||
		candidate.CapacityCandidate.Payload.MinimumRecommendedCapacity != prepared.DirectorTask.Capacity {
		t.Fatalf("capacity candidate=%+v", candidate.CapacityCandidate)
	}
	if len(candidate.Claims) != 1 ||
		candidate.Claims[0].AgentRequestID != prepared.DirectorTask.AgentRequestID {
		t.Fatalf("claims=%+v", candidate.Claims)
	}
}

func TestAppDirectorCandidateProviderV0DoesNotReemitStartedDirector(t *testing.T) {
	prepared := mustPrepareDirectorIntakeForTestV0(t)
	prepared.Run.StartedAgents = append(prepared.Run.StartedAgents, prepared.DirectorTask.AgentRequestID)

	candidates := mustBuildDirectorIntakeCandidatesForTestV0(t, prepared)
	if len(candidates.WorkCandidates) != 0 {
		t.Fatalf("work candidates=%+v", candidates.WorkCandidates)
	}
}

func TestAppDirectorIntakeV0ProgressiveLoopStartsDirector(t *testing.T) {
	prepared := mustPrepareDirectorIntakeForTestV0(t)
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(prepared.Run)
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	service := orquestacionnucleoapp.ServiceV0{
		RunStore:          store,
		EventSink:         sink,
		CandidateProvider: prepared.CandidateProvider,
		OutboxLedger:      ledger,
		MaxCommands:       8,
		MaxOutboxPerCycle: 2,
	}

	result, err := service.RunProgressiveLoopV0(context.Background(), orquestacionnucleoapp.ProgressiveLoopRequestV0{
		RunRef:               prepared.Run.RunID,
		OccurredAt:           "2026-05-09T22:01:00Z",
		MaxBursts:            6,
		MaxStepsPerBurst:     4,
		MaxDispatchesPerWait: 4,
		CorrelationID:        "corr-app-director-loop-001",
		EvidenceRefs:         []string{"evidence-ref-app-director-loop-001"},
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			directorIntakeCapacityDispatcherForTestV0(store, sink, ledger),
			directorIntakeAgentLauncherDispatcherForTestV0(store, sink, ledger),
		},
	})
	if err != nil {
		t.Fatalf("RunProgressiveLoopV0: %v", err)
	}
	if result.Status != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 {
		t.Fatalf("status=%s result=%+v", result.Status, result)
	}
	if !directorIntakeStringInSetV0(result.Run.StartedAgents, prepared.DirectorTask.AgentRequestID) {
		t.Fatalf("started_agents=%v", result.Run.StartedAgents)
	}
	if result.PendingOutboxCount != 0 {
		t.Fatalf("pending=%d refs=%v", result.PendingOutboxCount, result.PendingOutboxRefs)
	}
}
