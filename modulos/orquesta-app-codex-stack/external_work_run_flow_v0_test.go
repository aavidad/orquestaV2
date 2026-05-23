package orquestaappcodexstack

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestCodexStackV0ExternalWorkRunCreaRunSinDirectorInicial(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())

	result := postExternalWorkRunStackV0(t, stack)
	run, err := stack.Stores.RunStore.LoadRunV0(context.Background(), result.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if string(run.CurrentPhase) != "programacion" ||
		len(run.DirectorQuestions) != 1 ||
		len(run.StartedAgents) != 0 {
		t.Fatalf("run=%+v", run)
	}

	ranking := postRunQueuePriorityStackV0(t, stack, orquestamcp.MCPRunQueuePriorityToolInputV0{
		Action:   "rank",
		QueueRef: DefaultRunQueueRefV0,
	})
	if len(ranking.Ranked) != 1 ||
		ranking.Ranked[0].RunRef != result.RunRef ||
		ranking.Ranked[0].AppRef != "opes" {
		t.Fatalf("ranking=%+v result=%+v", ranking, result)
	}
}

func TestCodexStackV0ExternalWorkRunAceptaContratoAmplioV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	body := bytes.NewBuffer(nil)
	err := json.NewEncoder(body).Encode(orquestamcp.MCPExternalWorkRunToolInputV0{
		RequestID:     "req-external-work-wide-001",
		CorrelationID: "corr-external-work-wide-001",
		AppChangeRequest: orquestaappchange.AppChangeRequestV0{
			ChangeRef:  "opes-job-job-ref-wide-001",
			AppRef:     "opes",
			UserIntent: "Resolver trabajo externo amplio de OPES.",
			AcceptanceCriteria: []string{
				"devolver artifact_type=topic_expansion_package",
				"payload_json valido y trazable",
				"sin placeholders",
				"sin leer internals de OPES",
				"entrega en fichero unico bajo allowed_write_set",
				"paquete apto para tema_grande con capitulos y bloques trazables",
				"conservar base para tema_mediano sin perder autores normativa ni procedimientos",
				"incluir resumen/memoria de repaso derivado del tema desarrollado",
				"incluir esquema de examen y plan de visuales cuando aporten valor",
			},
			ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
				ProjectRef: "opes",
				JobRef:     "job-ref-wide-001",
				WorkKind:   "expand_topic_from_summary",
				InputFields: []orquestadomainwork.DomainWorkFieldV0{{
					Name:  "expected_artifact",
					Value: "topic_expansion_package",
				}},
			},
		},
	})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/external-work/run", body)
	req.Header.Set("Content-Type", "application/json")
	stack.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("external work run status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPExternalWorkRunToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode external work run: %v", err)
	}
	if result.Estado != orquestamcp.MCPExternalWorkRunEstadoOKV0 {
		t.Fatalf("result=%+v", result)
	}
	run, err := stack.Stores.RunStore.LoadRunV0(context.Background(), result.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if len(run.DirectorQuestions) != 1 {
		t.Fatalf("director_questions=%v", run.DirectorQuestions)
	}
}

func TestCodexStackV0ExternalWorkRunSaneaPreguntaDirectorSinPerderContextoV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	body := bytes.NewBuffer(nil)
	err := json.NewEncoder(body).Encode(orquestamcp.MCPExternalWorkRunToolInputV0{
		RequestID:     "req-external-work-safe-director-001",
		CorrelationID: "corr-external-work-safe-director-001",
		AppChangeRequest: orquestaappchange.AppChangeRequestV0{
			ChangeRef:  "self-review-rework-001",
			AppRef:     "orquesta",
			UserIntent: "Corregir flujo Codex sin exponer HOME token runtime ni provider en outbox.",
			AcceptanceCriteria: []string{
				"El conector Codex queda probado sin filtrar token ni HOME.",
				"El adapter conserva hexagonalidad.",
			},
			AllowedWriteSet: []string{"modulos/orquesta-app-codex-stack"},
			RequiredTests:   []string{"go test -count=1 ./modulos/orquesta-app-codex-stack"},
			MetadataRefs:    []string{"rules-token-economy-high"},
			ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
				ProjectRef: "orquesta",
				JobRef:     "job-safe-director-001",
				WorkKind:   "self_programming",
				WorkRefs:   []string{"runtime-codex-review"},
				InputFields: []orquestadomainwork.DomainWorkFieldV0{{
					Name:  "context_profile",
					Value: "large",
				}},
			},
		},
	})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/external-work/run", body)
	req.Header.Set("Content-Type", "application/json")
	stack.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("external work run status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPExternalWorkRunToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode external work run: %v", err)
	}
	if result.Estado != orquestamcp.MCPExternalWorkRunEstadoOKV0 {
		t.Fatalf("result=%+v", result)
	}
	run, err := stack.Stores.RunStore.LoadRunV0(context.Background(), result.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if len(run.DirectorQuestions) != 1 {
		t.Fatalf("director_questions=%v", run.DirectorQuestions)
	}
}

func TestCodexStackV0ExternalWorkRunSelfProgrammingSupervisaSinBloqueoGoBootstrapV0(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	body := bytes.NewBuffer(nil)
	err := json.NewEncoder(body).Encode(orquestamcp.MCPExternalWorkRunToolInputV0{
		RequestID:     "req-self-programming-policy-001",
		CorrelationID: "corr-self-programming-policy-001",
		AppChangeRequest: orquestaappchange.AppChangeRequestV0{
			ChangeRef:  "self-programming-web-001",
			AppRef:     "orquesta",
			UserIntent: "Completar web de Orquesta reutilizando modulos existentes.",
			AcceptanceCriteria: []string{
				"UI operativa sin logica de orquestacion en web.",
				"Archivos manejables y tests focales.",
			},
			AllowedWriteSet: []string{
				"modulos/orquesta-web",
				"docs/runbooks/autoprogramacion_web_2026-05-23.md",
			},
			RequiredTests: []string{"go test -count=1 ./modulos/orquesta-web"},
			MetadataRefs: []string{
				"rules-hexagonal-pure",
				"rules-i18n",
				"rules-manageable-files",
			},
			ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
				ProjectRef: "orquesta",
				JobRef:     "job-self-programming-web-001",
				WorkKind:   "self_programming",
				WorkRefs:   []string{"web-cockpit"},
				InputFields: []orquestadomainwork.DomainWorkFieldV0{{
					Name:  "context_profile",
					Value: "large",
				}},
			},
		},
	})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/external-work/run", body)
	req.Header.Set("Content-Type", "application/json")
	stack.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("external work run status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPExternalWorkRunToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode external work run: %v", err)
	}
	if result.Estado != orquestamcp.MCPExternalWorkRunEstadoOKV0 {
		t.Fatalf("result=%+v", result)
	}

	drain, err := stack.DrainRunV0(context.Background(), DrainRunRequestV0{
		RunRef:               result.RunRef,
		CorrelationID:        "corr-self-programming-policy-drain-001",
		OccurredAt:           "2026-05-23T08:00:00Z",
		MaxBursts:            16,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 8,
		MaxCommands:          32,
		MaxOutboxPerCycle:    8,
		MaxDecisionCycles:    4,
		MaxExternalWaits:     0,
	})
	if err != nil {
		t.Fatalf("DrainRunV0: %v drain=%+v", err, drain)
	}
	run, err := stack.Stores.RunStore.LoadRunV0(context.Background(), result.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if stringInSetV0(run.Blockers, "app-director-decision-director-decision-source-error-director-decision-source") {
		t.Fatalf("run bloqueado por source_error: %+v", run)
	}
	if len(run.Tasks) != 1 || len(run.StartedAgents) != 1 || runtime.launchCountV0() != 1 {
		t.Fatalf("run no lanzo agente: tasks=%v started=%v launches=%d blockers=%v", run.Tasks, run.StartedAgents, runtime.launchCountV0(), run.Blockers)
	}
}

func TestCodexStackV0ExternalWorkRunSupervisorConsumeDeliverySinExpirarWaitV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	planStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0()
	stack.Ports.OperationalPlanStateStore = planStore
	stack.Ports.OperationalPlanStateWriter = planStore
	result := postExternalWorkRunStackV0(t, stack)

	drain, err := stack.DrainRunV0(context.Background(), DrainRunRequestV0{
		RunRef:               result.RunRef,
		CorrelationID:        "corr-external-work-supervisor-delivery-001",
		OccurredAt:           "2026-05-22T18:10:00Z",
		MaxBursts:            16,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 8,
		MaxCommands:          32,
		MaxOutboxPerCycle:    8,
		MaxDecisionCycles:    4,
		MaxExternalWaits:     0,
	})
	if err != nil {
		t.Fatalf("DrainRunV0: %v drain=%+v", err, drain)
	}
	run, err := stack.Stores.RunStore.LoadRunV0(context.Background(), result.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if len(run.DeliveredAgents) != 1 || len(run.Deliveries) != 1 {
		t.Fatalf("run sin entrega: delivered=%v deliveries=%v", run.DeliveredAgents, run.Deliveries)
	}
	state, err := planStore.LoadOperationalDirectorPlanStateV0(
		context.Background(),
		result.RunRef,
		"operational-director-plan-director-decisions-"+appChangeSafeRefPartForTestV0(result.RunRef),
	)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	waitStep := codexStackExternalWorkPlanStepForTestV0(t, state, "step-wait-subagents")
	reviewStep := codexStackExternalWorkPlanStepForTestV0(t, state, "step-review-deliveries")
	if state.Status == orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		waitStep.Status == orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		waitStep.Reason == "external-wait-exhausted" {
		t.Fatalf("wait expiro tras delivery: state=%+v wait=%+v", state, waitStep)
	}
	if state.ActiveStepID == "step-wait-subagents" ||
		(reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 &&
			reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepAcceptedV0) {
		t.Fatalf("state no avanzo a review: state=%+v review=%+v", state, reviewStep)
	}
}

func postExternalWorkRunStackV0(
	t *testing.T,
	stack StackV0,
) orquestamcp.MCPExternalWorkRunToolResultV0 {
	t.Helper()
	body := bytes.NewBuffer(nil)
	err := json.NewEncoder(body).Encode(orquestamcp.MCPExternalWorkRunToolInputV0{
		RequestID:     "req-external-work-stack-001",
		CorrelationID: "corr-external-work-stack-001",
		AppChangeRequest: orquestaappchange.AppChangeRequestV0{
			ChangeRef:  "opes-job-job-ref-001",
			AppRef:     "opes",
			UserIntent: "Resolver trabajo externo de OPES.",
			AcceptanceCriteria: []string{
				"markdown valido",
			},
			ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
				ProjectRef: "opes",
				JobRef:     "job-ref-001",
				WorkKind:   "draft_content_block",
				InputFields: []orquestadomainwork.DomainWorkFieldV0{{
					Name:  "topic_id",
					Value: "topic-ref-001",
				}},
			},
		},
	})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/external-work/run", body)
	req.Header.Set("Content-Type", "application/json")
	stack.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("external work run status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPExternalWorkRunToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode external work run: %v", err)
	}
	if result.Estado != orquestamcp.MCPExternalWorkRunEstadoOKV0 {
		t.Fatalf("result=%+v", result)
	}
	return result
}

func codexStackExternalWorkPlanStepForTestV0(
	t *testing.T,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	stepID string,
) orquestacionnucleoapp.OperationalDirectorPlanStepStateV0 {
	t.Helper()
	for _, step := range state.Steps {
		if step.StepID == stepID {
			return step
		}
	}
	t.Fatalf("step %s no existe en %+v", stepID, state)
	return orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{}
}

func appChangeSafeRefPartForTestV0(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\\", "-")
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, " ", "-")
	return value
}
