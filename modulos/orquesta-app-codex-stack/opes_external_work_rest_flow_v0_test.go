package orquestaappcodexstack

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func TestCodexStackV0OPESExternalWorkRESTCreaMicrotareaSinWriteSetLocal(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	director := postDirectorAPIV0(t, stack)

	changeResult := postOPESExternalWorkChangeV0(t, stack, opesExternalWorkChangeV0(director.RunRef))
	if changeResult.DirectorQuestionRef == "" ||
		changeResult.ChangeRef != "opes-job-job-ref-opes-001" ||
		changeResult.AppRef != "opes" {
		t.Fatalf("changeResult=%+v", changeResult)
	}

	run, err := stack.Stores.RunStore.LoadRunV0(context.Background(), director.RunRef)
	if err != nil {
		t.Fatalf("load run: %v", err)
	}
	if !codexStackStringInSetForTestV0(run.DirectorQuestions, changeResult.DirectorQuestionRef) {
		t.Fatalf("director_questions=%v missing=%s", run.DirectorQuestions, changeResult.DirectorQuestionRef)
	}

	if _, err := stack.DrainRunV0(context.Background(), DrainRunRequestV0{
		RunRef:               director.RunRef,
		CorrelationID:        "corr-opes-rest-drain-001",
		MaxBursts:            16,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 8,
		MaxExternalWaits:     2,
	}); err != nil {
		t.Fatalf("DrainRunV0: %v", err)
	}

	run, err = stack.Stores.RunStore.LoadRunV0(context.Background(), director.RunRef)
	if err != nil {
		t.Fatalf("load run after drain: %v", err)
	}
	tasks, err := stack.Stores.TaskStore.LoadWorkflowTasksV0(context.Background(), director.RunRef, run.Tasks)
	if err != nil {
		t.Fatalf("load tasks: %v", err)
	}
	task := findOPESExternalWorkflowTaskV0(tasks)
	if task.TaskID == "" ||
		!codexStackStringInSetForTestV0(task.WriteSet, "external/opes/draft_content_block") ||
		!codexStackStringInSetForTestV0(task.WriteSet, "external/opes/opes-job-job-ref-opes-001") ||
		!codexStackStringInSetForTestV0(task.RequiredTests, "validar contrato externo de dominio") ||
		!taskHasFunctionV0(task, "ApplyExternalDomainWorkV0") {
		t.Fatalf("task=%+v all=%+v", task, tasks)
	}

	stats := postOPESDirectorStatsV0(t, stack, director.RunRef)
	if stats.Estado != orquestamcp.MCPDirectorStatsEstadoOKV0 ||
		stats.RunRef != director.RunRef ||
		stats.Stats == nil ||
		stats.Stats.Counts.TasksTotal == 0 {
		t.Fatalf("stats=%+v", stats)
	}
}

func postOPESExternalWorkChangeV0(
	t *testing.T,
	stack StackV0,
	change orquestaappchange.AppChangeRequestV0,
) orquestamcp.MCPRequestAppChangeToolResultV0 {
	t.Helper()
	body := new(bytes.Buffer)
	if err := json.NewEncoder(body).Encode(orquestamcp.MCPRequestAppChangeToolInputV0{
		RequestID:        change.RequestID,
		CorrelationID:    change.CorrelationID,
		AppChangeRequest: change,
	}); err != nil {
		t.Fatalf("encode change: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v0/apps/opes/changes", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	stack.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("change status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPRequestAppChangeToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode change: %v", err)
	}
	if result.Estado != orquestamcp.MCPRequestAppChangeEstadoOKV0 {
		t.Fatalf("change result=%+v", result)
	}
	return result
}

func opesExternalWorkChangeV0(runRef string) orquestaappchange.AppChangeRequestV0 {
	return orquestaappchange.AppChangeRequestV0{
		SchemaVersion: "app_change_request.v0",
		RequestID:     "req-opes-rest-change-001",
		CorrelationID: "corr-opes-rest-change-001",
		RunRef:        runRef,
		AppRef:        "opes",
		ChangeRef:     "opes-job-job-ref-opes-001",
		ActorRef:      "opes",
		Locale:        "es",
		UserIntent: "Resolver job OPES draft_content_block para crear un bloque de temario " +
			"con fuentes verificables y entrega por artefacto OPES.",
		TargetArea: "domain_work",
		CurrentStateRefs: []string{
			"opes-job-job-ref-opes-001",
			"opes-topic-topic-ref-opes-001",
			"opes-chapter-chapter-ref-opes-001",
		},
		AcceptanceCriteria: []string{
			"markdown valido",
			"sin placeholders",
			"fuentes verificables cuando aplique",
			"listo para revision legal y pedagogica",
		},
		Constraints: []string{
			"no inventar normativa",
			"no leer internals de OPES",
			"devolver resultado por artefacto OPES",
		},
		ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
			ProjectRef:    "opes",
			InterfaceRefs: []string{"opes-rest-v0", "opes-mcp-v0"},
			WorkKind:      "draft_content_block",
			WorkRefs: []string{
				"opes-job-job-ref-opes-001",
				"opes-topic-topic-ref-opes-001",
				"opes-chapter-chapter-ref-opes-001",
			},
		},
	}
}

func findOPESExternalWorkflowTaskV0(
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) orquestacoreworkflow.WorkflowTaskV0 {
	for _, task := range tasks {
		if task.Title == "Resolver bloque documental externo" {
			return task
		}
	}
	return orquestacoreworkflow.WorkflowTaskV0{}
}

func taskHasFunctionV0(
	task orquestacoreworkflow.WorkflowTaskV0,
	functionName string,
) bool {
	for _, ref := range task.FunctionContractRefs {
		if ref.FunctionName == functionName {
			return true
		}
	}
	return false
}

func postOPESDirectorStatsV0(
	t *testing.T,
	stack StackV0,
	runRef string,
) orquestamcp.MCPDirectorStatsToolResultV0 {
	t.Helper()
	body := new(bytes.Buffer)
	if err := json.NewEncoder(body).Encode(orquestamcp.MCPDirectorStatsToolInputV0{
		RequestID:            "req-opes-rest-stats-001",
		CorrelationID:        "corr-opes-rest-stats-001",
		RunRef:               runRef,
		IncludeAgentProgress: true,
	}); err != nil {
		t.Fatalf("encode stats: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v0/director/stats", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	stack.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("stats status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPDirectorStatsToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode stats: %v", err)
	}
	return result
}
