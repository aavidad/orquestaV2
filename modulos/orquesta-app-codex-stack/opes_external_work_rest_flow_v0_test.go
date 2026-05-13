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
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
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
		!codexStackStringInSetForTestV0(task.WriteSet, "external/opes/job-ref-opes-001") ||
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

func TestCodexStackV0OPESExternalWorkDeliveryEnviaArtefactoDomainWork(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	domainWork := &fakeCodexStackDomainWorkExecutorV0{}
	stack := mustBuildCodexStackWithDomainWorkForTestV0(t, runtime, domainWork)
	director := postDirectorAPIV0(t, stack)
	postOPESExternalWorkChangeV0(t, stack, opesExternalWorkChangeV0(director.RunRef))

	for attempt := 1; attempt <= 6; attempt++ {
		if _, err := stack.DrainRunV0(context.Background(), DrainRunRequestV0{
			RunRef:               director.RunRef,
			CorrelationID:        "corr-opes-rest-artifact-drain",
			MaxBursts:            16,
			MaxStepsPerBurst:     8,
			MaxDispatchesPerWait: 8,
			MaxExternalWaits:     2,
		}); err != nil {
			t.Fatalf("DrainRunV0 intento %d: %v", attempt, err)
		}
		if _, ok := findDomainWorkSubmitForTestV0(domainWork.inputs); ok {
			break
		}
	}

	submit, ok := findDomainWorkSubmitForTestV0(domainWork.inputs)
	if !ok {
		run, _ := stack.Stores.RunStore.LoadRunV0(context.Background(), director.RunRef)
		t.Fatalf(
			"submit_artifact no invocado: inputs=%+v phase=%s tasks=%v agents=%v started=%v deliveries=%v artifacts=%v bridge=%+v",
			domainWork.inputs,
			run.CurrentPhase,
			run.Tasks,
			run.Agents,
			run.StartedAgents,
			run.Deliveries,
			run.PhaseArtifacts,
			stack.DomainDelivery,
		)
	}
	artifact := submit.ArtifactSubmission
	if artifact.JobRef != "job-ref-opes-001" ||
		artifact.ArtifactType != "content_block" ||
		artifact.CompleteJob != true ||
		!domainWorkFieldValueForTestV0(artifact.PayloadFields, "topic_id", "topic-ref-opes-001") ||
		!domainWorkFieldValueForTestV0(artifact.PayloadFields, "body", "entrega fake para revision") ||
		!domainWorkExternalRefForTestV0(artifact.ExternalRefs, "run_ref", director.RunRef) {
		t.Fatalf("artifact=%+v", artifact)
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
			JobRef:        "job-ref-opes-001",
			InterfaceRefs: []string{"opes-rest-v0", "opes-mcp-v0"},
			WorkKind:      "draft_content_block",
			WorkRefs: []string{
				"opes-topic-topic-ref-opes-001",
				"opes-chapter-chapter-ref-opes-001",
			},
			InputFields: opesExternalWorkInputFieldsV0(),
		},
	}
}

func opesExternalWorkInputFieldsV0() []orquestadomainwork.DomainWorkFieldV0 {
	return []orquestadomainwork.DomainWorkFieldV0{
		{Name: "topic_id", Value: "topic-ref-opes-001"},
		{Name: "chapter_id", Value: "chapter-ref-opes-001"},
		{Name: "language_code", Value: "es"},
		{Name: "block_position", ValueJSON: []byte(`{"chapter_order":1,"block_order":1,"total_blocks":3}`)},
		{Name: "syllabus_full", Value: "Temario compacto validado por OPES."},
		{Name: "outline", Value: "Esquema del tema validado por OPES."},
		{Name: "source_refs", Values: []string{"BOE-A-001"}},
	}
}

func findDomainWorkSubmitForTestV0(
	inputs []orquestamcp.MCPDomainWorkToolInputV0,
) (orquestamcp.MCPDomainWorkToolInputV0, bool) {
	for _, input := range inputs {
		if input.Action == orquestamcp.MCPDomainWorkActionSubmitArtifactV0 {
			return input, true
		}
	}
	return orquestamcp.MCPDomainWorkToolInputV0{}, false
}

func domainWorkFieldValueForTestV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	name string,
	value string,
) bool {
	for _, field := range fields {
		if field.Name == name && field.Value == value {
			return true
		}
	}
	return false
}

func domainWorkExternalRefForTestV0(
	refs []orquestadomainwork.DomainWorkExternalRefV0,
	kind string,
	ref string,
) bool {
	for _, item := range refs {
		if item.Kind == kind && item.Ref == ref {
			return true
		}
	}
	return false
}

func findOPESExternalWorkflowTaskV0(
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) orquestacoreworkflow.WorkflowTaskV0 {
	for _, task := range tasks {
		if task.Title == "Redactar bloque documental OPES" {
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
