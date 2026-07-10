package orquestaappcodexstack

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestaappchangedirectorsource "orquesta/modulos/orquesta-app-change-director-source"
	orquestacapacity "orquesta/modulos/orquesta-capacity"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestaexternalworkrun "orquesta/modulos/orquesta-external-work-run"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaopesbridge "orquesta/modulos/orquesta-opes-bridge"
	orquestaopesconnector "orquesta/modulos/orquesta-opes-connector"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestCodexStackV0OPESPlanTemaDeliveryEnviaDocumentPlan(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0().
		withDeliveryBodyForTargetV0("external/opes/plan_tema", validDocumentPlanPayloadForTestV0())
	domainWork := &fakeCodexStackDomainWorkExecutorV0{}
	stack := mustBuildCodexStackWithDomainWorkForTestV0(t, runtime, domainWork)
	director := postDirectorAPIV0(t, stack)
	postOPESExternalWorkChangeV0(t, stack, opesPlanExternalWorkChangeV0(director.RunRef))

	for attempt := 1; attempt <= 6; attempt++ {
		if _, err := stack.DrainRunV0(context.Background(), DrainRunRequestV0{
			RunRef:               director.RunRef,
			CorrelationID:        "corr-opes-plan-drain",
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
		t.Fatalf("submit_artifact no invocado: inputs=%+v", domainWork.inputs)
	}
	artifact := submit.ArtifactSubmission
	if artifact.JobRef != "job-ref-plan-001" ||
		artifact.ArtifactType != orquestadomainwork.DomainDocumentPlanArtifactTypeV0 ||
		artifact.CompleteJob != true ||
		!domainWorkFieldValueForTestV0(artifact.PayloadFields, "plan_ref", "plan_ref_080") ||
		!domainWorkFieldHasJSONForTestV0(artifact.PayloadFields, "sections") ||
		!domainWorkFieldHasJSONForTestV0(artifact.PayloadFields, "deliverables") {
		t.Fatalf("artifact=%+v", artifact)
	}
}

func TestCodexStackV0OPESPlanTemarioOperadoresSupervisorXHighEnviaDocumentPlan(t *testing.T) {
	const target = "external/opes/plan_temario/job-ref-plan-operadores-001"
	runtime := newFakeCodexStackRuntimeV0().
		withDeliveryBodyForTargetV0(target, validPlanTemarioOperadoresPayloadForTestV0())
	domainWork := &fakeCodexStackDomainWorkExecutorV0{}
	request, ok := orquestaopesbridge.BuildExternalWorkRunRequestV0(
		orquestaopesconnector.ExternalJobV0{
			ID:            "job-ref-plan-operadores-001",
			Type:          "plan_temario",
			CorrelationID: "corr-job-plan-operadores-001",
			RequestedBy:   "opes",
			PayloadJSON: `{
				"program_id":"program-ref-operadores-001",
				"document_kind":"temario_oposicion",
				"language_code":"es",
				"title":"Temario operadores",
				"official_outline":"Operadores, tipos, precedencia, asociatividad y usos.",
				"quality_criteria":["completo","sin placeholders"],
				"target_pages_min":20,
				"target_pages_max":40
			}`,
		},
		orquestaopesbridge.JobRunConfigV0{
			PriorityScore: 90,
			RequestedBy:   "opes-test",
		},
	)
	if !ok {
		t.Fatalf("request no construida")
	}
	config := codexStackBaseConfigForTestV0(t, runtime, domainWork, nil)
	config.Codex.ModelRouting.TaskRoutes[orquestaappchangedirectorsource.AppChangeTaskRefV0(request.AppChangeRequest.ChangeRef)] = orquestacapacity.ModelRoutingRequestV0{
		Level:                 orquestacapacity.ModelRoutingLevelComplexV0,
		RequestedEffort:       "xhigh",
		ReasonRef:             "reason-ref-opes-plan-temario-xhigh",
		EvidenceRefs:          []string{"evidence-ref-opes-plan-temario-xhigh"},
		XHighAuthorizationRef: "authorization-ref-opes-plan-temario-xhigh",
	}
	stack, err := BuildStackV0(config)
	if err != nil {
		t.Fatalf("BuildStackV0: %v", err)
	}
	run := postExternalWorkRunRequestStackV0(t, stack, request)
	supervisor := postRunSupervisorStackV0(t, stack, legacyRunSupervisorInputForStackTestV0(orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:            "request-ref-plan-temario-operadores-supervisor-001",
		CorrelationID:        "corr-plan-temario-operadores-supervisor-001",
		RunRef:               run.RunRef,
		MaxTicks:             8,
		MaxBursts:            16,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 8,
		MaxCommands:          20,
		MaxOutboxPerCycle:    8,
		MaxExternalWaits:     2,
		ContinueMessage:      "sigue hasta entregar document_plan",
		AllowRepeatedRuns:    true,
	}))
	if supervisor.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 ||
		supervisor.RunRef != run.RunRef ||
		supervisor.Last.Status == string(CodexSupervisorRuntimeFailedV0) {
		t.Fatalf("supervisor=%+v run=%+v", supervisor, run)
	}

	submit, ok := findDomainWorkSubmitForTestV0(domainWork.inputs)
	if !ok {
		t.Fatalf("submit_artifact no invocado: inputs=%+v supervisor=%+v", domainWork.inputs, supervisor)
	}
	artifact := submit.ArtifactSubmission
	if artifact.JobRef != "job-ref-plan-operadores-001" ||
		artifact.ArtifactType != orquestadomainwork.DomainDocumentPlanArtifactTypeV0 ||
		artifact.CompleteJob != true ||
		!domainWorkFieldValueForTestV0(artifact.PayloadFields, "plan_ref", "plan-job-ref-plan-operadores-001") ||
		!domainWorkFieldValueForTestV0(artifact.PayloadFields, "work_kind", "plan_temario") ||
		!domainWorkFieldValueForTestV0(artifact.PayloadFields, "scope_ref", "program-ref-operadores-001") ||
		!domainWorkFieldHasJSONForTestV0(artifact.PayloadFields, "sections") ||
		!domainWorkFieldHasJSONForTestV0(artifact.PayloadFields, "deliverables") ||
		!domainWorkExternalRefForTestV0(artifact.ExternalRefs, "run_ref", run.RunRef) {
		t.Fatalf("artifact=%+v", artifact)
	}
	descriptor := codexStackDescriptorByWriteSetForTestV0(t, stack, target)
	if descriptor.Spec.AgentPacket.CapacityLevel != "xhigh" {
		t.Fatalf("capacity_level=%q descriptor=%+v", descriptor.Spec.AgentPacket.CapacityLevel, descriptor)
	}
	for _, want := range []string{
		"opes_level_derivation_policy",
		"A1/A2 o A1 maestro",
		"B/C1",
		"C2/AP",
		"opes_assimilation_method",
		"recuperacion activa",
		"opes_global_editorial_policy_2026_05_18",
		"modo tutor completo",
		"no infantilizar",
		"notas de test separadas",
		"20.250",
		"opes_html_publication_policy_2026_05_19",
		"patron web tipo Tema 11",
		"primera lectura activa por defecto",
	} {
		if !codexStackContextContainsForTestV0(descriptor.Spec.AgentPacket.Context.Entries, want) {
			t.Fatalf("contexto de agente no contiene %q: %+v", want, descriptor.Spec.AgentPacket.Context.Entries)
		}
	}
	wrapperPath := filepath.Join(filepath.Dir(descriptor.AckPath), orquestaruntimecodex.CodexWrapperFileNameV0)
	wrapper, err := os.ReadFile(wrapperPath)
	if err != nil {
		t.Fatalf("read wrapper: %v", err)
	}
	if !strings.Contains(string(wrapper), `model_reasoning_effort="xhigh"`) {
		t.Fatalf("wrapper sin xhigh: %s", string(wrapper))
	}
}

func opesPlanExternalWorkChangeV0(runRef string) orquestaappchange.AppChangeRequestV0 {
	return orquestaappchange.AppChangeRequestV0{
		SchemaVersion: "app_change_request.v0",
		RequestID:     "req-opes-plan-change-001",
		CorrelationID: "corr-opes-plan-change-001",
		RunRef:        runRef,
		AppRef:        "opes",
		ChangeRef:     "opes-job-job-ref-plan-001",
		ActorRef:      "opes",
		Locale:        "es",
		UserIntent: "Planificar tema OPES antes de redactarlo, con secciones, " +
			"entregables, visuales y revisiones.",
		TargetArea: "domain_work",
		CurrentStateRefs: []string{
			"opes-job-job-ref-plan-001",
			"opes-topic-topic-ref-080",
		},
		AcceptanceCriteria: []string{
			"devolver artifact_type=document_plan",
			"payload_json valido y trazable",
			"sin placeholders",
			"no redactar el documento final dentro del plan",
		},
		Constraints: []string{
			"no leer internals de OPES",
			"usar solo el paquete de dominio recibido",
		},
		ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
			ProjectRef:    "opes",
			JobRef:        "job-ref-plan-001",
			InterfaceRefs: []string{"opes-rest-v0", "opes-mcp-v0"},
			WorkKind:      "plan_tema",
			WorkRefs: []string{
				"opes-topic-topic-ref-080",
			},
			InputFields: []orquestadomainwork.DomainWorkFieldV0{
				{Name: "topic_id", Value: "topic-ref-080"},
				{Name: "language_code", Value: "es"},
				{Name: "expected_artifact_type", Value: orquestadomainwork.DomainDocumentPlanArtifactTypeV0},
			},
		},
	}
}

func postExternalWorkRunRequestStackV0(
	t *testing.T,
	stack StackV0,
	request orquestaexternalworkrun.StartExternalWorkRunRequestV0,
) orquestamcp.MCPExternalWorkRunToolResultV0 {
	t.Helper()
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(orquestamcp.MCPExternalWorkRunToolInputV0{
		RequestID:              request.RequestID,
		CorrelationID:          request.CorrelationID,
		DirectorExecutionMode:  orquestamcp.MCPExternalWorkRunDirectorExecutionModeLegacyLoopV0,
		ExternalWorkRunRequest: request,
	}); err != nil {
		t.Fatalf("encode external work run: %v", err)
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
		t.Fatalf("external work run result=%+v", result)
	}
	return result
}

func postRunSupervisorStackV0(
	t *testing.T,
	stack StackV0,
	input orquestamcp.MCPRunSupervisorToolInputV0,
) orquestamcp.MCPRunSupervisorToolResultV0 {
	t.Helper()
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(input); err != nil {
		t.Fatalf("encode supervisor: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/runs/supervise", body)
	req.Header.Set("Content-Type", "application/json")
	stack.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("supervisor status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPRunSupervisorToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode supervisor: %v", err)
	}
	return result
}

func codexStackDescriptorByWriteSetForTestV0(
	t *testing.T,
	stack StackV0,
	writeSet string,
) orquestaruntimecodexdelivery.CodexReceiptDescriptorV0 {
	t.Helper()
	store := stack.Stores.ReceiptStore.(*orquestaruntimecodexdelivery.InMemoryCodexReceiptDescriptorStoreV0)
	for _, descriptor := range codexStackRealSmokeDescriptorsV0(t, store) {
		if codexStackStringInSetForTestV0(descriptor.Spec.AgentPacket.Task.WriteSet, writeSet) {
			return descriptor
		}
	}
	t.Fatalf("descriptor con write_set=%s no encontrado", writeSet)
	return orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{}
}

func validPlanTemarioOperadoresPayloadForTestV0() string {
	return `{
		"schema_version":"domain_document_plan.v0",
		"artifact_type":"document_plan",
		"plan_ref":"plan-job-ref-plan-operadores-001",
		"domain_ref":"opes",
		"work_kind":"plan_temario",
		"document_kind":"temario_oposicion",
		"scope_ref":"program-ref-operadores-001",
		"language_code":"es",
		"title":"Temario operadores",
		"objective":"Planificar un temario completo sobre operadores sin redactar el documento final.",
		"estimated_pages_min":20,
		"estimated_pages_max":40,
		"sections":[
			{
				"section_ref":"section-operadores-basicos",
				"order":1,
				"title":"Operadores basicos",
				"objective":"Cubrir definicion, clasificacion y usos habituales.",
				"work_kind":"draft_content_block",
				"target_words_min":900,
				"acceptance_criteria":["explicar precedencia","incluir ejemplos"]
			},
			{
				"section_ref":"section-precedencia-asociatividad",
				"order":2,
				"title":"Precedencia y asociatividad",
				"objective":"Planificar el desarrollo de reglas de evaluacion.",
				"work_kind":"draft_content_block",
				"target_words_min":900,
				"acceptance_criteria":["orden de evaluacion claro","sin ambiguedades"]
			}
		],
		"visuals":[
			{
				"visual_ref":"visual-tabla-precedencia",
				"visual_type":"tabla",
				"placement_ref":"section-precedencia-asociatividad",
				"objective":"Comparar precedencia de operadores.",
				"work_kind":"generate_visual_asset"
			}
		],
		"review_steps":[
			{
				"review_ref":"review-quality-operadores",
				"order":1,
				"work_kind":"review_quality",
				"objective":"Validar coherencia tecnica y pedagogica."
			}
		],
		"deliverables":[
			{
				"deliverable_ref":"deliverable-temario-operadores",
				"artifact_type":"assembled_topic",
				"title":"Temario operadores ensamblado",
				"required":true
			}
		],
		"quality_criteria":[
			"completo",
			"sin placeholders",
			"trazable al programa",
			"partir de maestro A1/A2 o A1 si existe equivalente",
			"derivar niveles inferiores por resumen, reduccion editorial y adaptacion"
		],
		"constraints":[
			"no construir maestro superior elevando resumen B/C1/C2/AP",
			"aplicar recuperacion activa, repaso espaciado y visuales utiles"
		]
	}`
}
