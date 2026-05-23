package orquestaappchangedirectorsource

import (
	"context"
	"strings"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
)

func TestAppChangeDirectorDecisionSourceV0GeneraCadenaCompleta(t *testing.T) {
	record := appChangeRecordForSourceTestV0()
	store := orquestaappchange.NewInMemoryAppChangeStoreV0(record)
	run := appChangeRunForSourceTestV0(record)

	decisions, err := (AppChangeDirectorDecisionSourceV0{Store: store}).
		ListDirectorAgentDecisionsV0(
			context.Background(),
			orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{Run: run},
		)

	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	if len(decisions) != 8 {
		t.Fatalf("decisions=%+v", decisions)
	}
	if decisions[0].CommandType != orquestadirectoragent.DirectorAgentCommandAnswerQuestionV0 ||
		decisions[6].CommandType != orquestadirectoragent.DirectorAgentCommandCreateMicrotaskV0 {
		t.Fatalf("command types=%s %s", decisions[0].CommandType, decisions[6].CommandType)
	}
	task := decisions[6].CreateMicrotask.Task
	if len(task.WriteSet) != 1 || task.WriteSet[0] != "web/agenda" {
		t.Fatalf("write_set=%+v", task.WriteSet)
	}
	if task.WorkProfileKind != "implementation" {
		t.Fatalf("work_profile_kind=%s", task.WorkProfileKind)
	}
	if len(task.RequiredTests) == 0 {
		t.Fatalf("required_tests vacio: %+v", task)
	}
	if issues := orquestadirectoragent.ValidateDirectorAgentDecisionV0(decisions[6]); len(issues) != 0 {
		t.Fatalf("decision invalida: %+v", issues)
	}
}

func TestAppChangeDirectorDecisionSourceV0ExigeGoTestSiTocaCodigoGo(t *testing.T) {
	record := appChangeRecordForSourceTestV0()
	record.Request.AllowedWriteSet = []string{"web/agenda/view.go"}
	store := orquestaappchange.NewInMemoryAppChangeStoreV0(record)
	run := appChangeRunForSourceTestV0(record)

	decisions, err := (AppChangeDirectorDecisionSourceV0{Store: store}).
		ListDirectorAgentDecisionsV0(
			context.Background(),
			orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{Run: run},
		)

	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	task := decisions[6].CreateMicrotask.Task
	if len(task.RequiredTests) == 0 || task.RequiredTests[0] != "go test ./..." {
		t.Fatalf("required_tests=%+v", task.RequiredTests)
	}
}

func TestAppChangeDirectorDecisionSourceV0ConservaRequiredTestsExplicitos(t *testing.T) {
	record := appChangeRecordForSourceTestV0()
	record.Request.AllowedWriteSet = []string{"web/agenda/view.go"}
	record.Request.RequiredTests = []string{"go test -count=1 ./...", "npm test"}
	store := orquestaappchange.NewInMemoryAppChangeStoreV0(record)
	run := appChangeRunForSourceTestV0(record)

	decisions, err := (AppChangeDirectorDecisionSourceV0{Store: store}).
		ListDirectorAgentDecisionsV0(
			context.Background(),
			orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{Run: run},
		)

	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	task := decisions[6].CreateMicrotask.Task
	if len(task.RequiredTests) < 3 ||
		task.RequiredTests[0] != "go test -count=1 ./..." ||
		task.RequiredTests[1] != "npm test" ||
		stringInSetV0(task.RequiredTests, "go test ./...") {
		t.Fatalf("required_tests=%+v", task.RequiredTests)
	}
}

func TestAppChangeDirectorDecisionSourceV0ProyectaMetadataRefsComoContextRefs(t *testing.T) {
	record := appChangeRecordForSourceTestV0()
	record.Request.MetadataRefs = []string{
		" context-ref-app-change-scope-001 ",
		"context-ref-app-change-policy-001",
		"context-ref-app-change-scope-001",
	}
	store := orquestaappchange.NewInMemoryAppChangeStoreV0(record)
	run := appChangeRunForSourceTestV0(record)

	decisions, err := (AppChangeDirectorDecisionSourceV0{Store: store}).
		ListDirectorAgentDecisionsV0(
			context.Background(),
			orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{Run: run},
		)

	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	task := decisions[6].CreateMicrotask.Task
	if len(task.ContextRefs) != 2 ||
		task.ContextRefs[0] != "context-ref-app-change-scope-001" ||
		task.ContextRefs[1] != "context-ref-app-change-policy-001" {
		t.Fatalf("context_refs=%+v", task.ContextRefs)
	}
	if _, err := orquestacoreworkflow.NewWorkflowTaskV0(workflowTaskFromDirectorTaskForTestV0(task)); err != nil {
		t.Fatalf("workflow task invalida con context_refs: %v task=%+v", err, task)
	}
}

func TestAppChangeDirectorDecisionSourceV0ProyectaTrabajoExterno(t *testing.T) {
	record := appChangeRecordForSourceTestV0()
	record.Request.ExternalWork = &orquestaappchange.AppChangeExternalWorkV0{
		ProjectRef:    "project-ref-opes",
		InterfaceRefs: []string{"mcp-contract-ref-opes-v0"},
		WorkKind:      "documentation",
		WorkRefs:      []string{"domain-work-ref-opes-topic-001"},
	}
	store := orquestaappchange.NewInMemoryAppChangeStoreV0(record)
	run := appChangeRunForSourceTestV0(record)

	decisions, err := (AppChangeDirectorDecisionSourceV0{Store: store}).
		ListDirectorAgentDecisionsV0(
			context.Background(),
			orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{Run: run},
		)

	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	contract := decisions[5].PublishContract
	task := decisions[6].CreateMicrotask.Task
	if contract == nil ||
		len(contract.FunctionNames) != 1 ||
		contract.FunctionNames[0] != "ApplyExternalDomainWorkV0" ||
		task.Title != "Resolver trabajo documental OPES" ||
		task.WorkProfileKind != "domain_work" ||
		task.Summary != "Resolver trabajo documental con paquete de dominio suficiente: temario, esquema, objetivo, fuentes, criterios y longitud si llegan." ||
		!stringInSetV0(task.AcceptanceCriteria, "Tratar el paquete de dominio OPES como entrada suficiente, no como contexto minimo.") ||
		!stringInSetV0(task.AcceptanceCriteria, "Usar temario, esquema, objetivo, fuentes, criterios y longitud si llegan en input_fields.") ||
		!stringInSetV0(task.RequiredTests, "validar contrato externo de dominio") ||
		!stringInSetV0(task.RequiredTests, "validar paquete documental de dominio") {
		t.Fatalf("contract=%+v task=%+v", contract, task)
	}
}

func TestAppChangeDirectorDecisionSourceV0ProyectaPlanDocumental(t *testing.T) {
	record := appChangeRecordForSourceTestV0()
	record.Request.ExternalWork = &orquestaappchange.AppChangeExternalWorkV0{
		ProjectRef:    "opes",
		InterfaceRefs: []string{"opes-rest-v0", "opes-mcp-v0"},
		WorkKind:      "plan_tema",
		WorkRefs:      []string{"opes-job-plan-001", "opes-topic-080"},
	}
	store := orquestaappchange.NewInMemoryAppChangeStoreV0(record)
	run := appChangeRunForSourceTestV0(record)

	decisions, err := (AppChangeDirectorDecisionSourceV0{Store: store}).
		ListDirectorAgentDecisionsV0(
			context.Background(),
			orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{Run: run},
		)

	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	task := decisions[6].CreateMicrotask.Task
	if task.Title != "Planificar tema documental externo" ||
		task.Summary != "Crear plan documental validable; no redactar ni ensamblar el documento final en esta tarea." ||
		!stringInSetV0(task.AcceptanceCriteria, "Devolver artifact_type=document_plan compatible con DomainDocumentPlanV0.") ||
		!stringInSetV0(task.AcceptanceCriteria, "No redactar el documento final en esta tarea; solo plan verificable.") ||
		!stringInSetV0(task.RequiredTests, "validar document_plan") ||
		!stringInSetV0(task.RequiredTests, "validar secciones y entregables del plan") {
		t.Fatalf("task=%+v", task)
	}
}

func TestAppChangeDirectorDecisionSourceV0ProyectaTrabajoExternoSinWriteSetLocal(t *testing.T) {
	record := appChangeRecordForSourceTestV0()
	record.Request.AllowedWriteSet = nil
	record.Request.ExternalWork = &orquestaappchange.AppChangeExternalWorkV0{
		ProjectRef:    "opes",
		InterfaceRefs: []string{"opes-rest-v0", "opes-mcp-v0"},
		WorkKind:      "draft_content_block",
		WorkRefs:      []string{"opes-job-001", "opes-topic-001"},
	}
	store := orquestaappchange.NewInMemoryAppChangeStoreV0(record)
	run := appChangeRunForSourceTestV0(record)

	decisions, err := (AppChangeDirectorDecisionSourceV0{Store: store}).
		ListDirectorAgentDecisionsV0(
			context.Background(),
			orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{Run: run},
		)

	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	if len(decisions) != 8 {
		t.Fatalf("decisions=%+v", decisions)
	}
	task := decisions[6].CreateMicrotask.Task
	if task.Title != "Redactar bloque documental OPES" ||
		task.Summary != "Redactar unidad editorial amplia con paquete de dominio suficiente; no trocear un temario largo en parrafos sin continuidad." ||
		!stringInSetV0(task.WriteSet, "external/opes/draft_content_block") ||
		!stringInSetV0(task.WriteSet, "external/opes/opes-job-001") ||
		!stringInSetV0(task.AcceptanceCriteria, "Tratar el paquete de dominio OPES como entrada suficiente, no como contexto minimo.") ||
		!stringInSetV0(task.AcceptanceCriteria, "Para draft_content_block, usar paquete editorial suficiente y no sobreatomizar: bloque, subcapitulo o capitulo coherente si OPES lo envio asi.") ||
		!stringInSetV0(task.AcceptanceCriteria, "No redactar un tema de 50 folios sin paquete suficiente; pedir division editorial a OPES si excede contexto o trazabilidad.") ||
		!stringInSetV0(task.RequiredTests, "validar paquete de bloque documental") ||
		!stringInSetV0(task.RequiredTests, "validar granularidad editorial coherente") ||
		!stringInSetV0(task.RequiredTests, "validar fuentes, criterios y longitud si llegan") {
		t.Fatalf("task=%+v", task)
	}
}

func TestAppChangeDirectorDecisionSourceV0DraftContentBlockExternoNoHeredaOPES(t *testing.T) {
	record := appChangeRecordForSourceTestV0()
	record.Request.AllowedWriteSet = nil
	record.Request.ExternalWork = &orquestaappchange.AppChangeExternalWorkV0{
		ProjectRef:    "external-editorial",
		InterfaceRefs: []string{"domain-contract-v0"},
		WorkKind:      "draft_content_block",
		WorkRefs:      []string{"job-001", "topic-001"},
	}
	store := orquestaappchange.NewInMemoryAppChangeStoreV0(record)
	run := appChangeRunForSourceTestV0(record)

	decisions, err := (AppChangeDirectorDecisionSourceV0{Store: store}).
		ListDirectorAgentDecisionsV0(
			context.Background(),
			orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{Run: run},
		)

	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	task := decisions[6].CreateMicrotask.Task
	if task.Title != "Redactar bloque documental externo" ||
		!stringInSetV0(task.AcceptanceCriteria, "Para draft_content_block, usar paquete editorial suficiente y no sobreatomizar: bloque, subcapitulo o capitulo coherente si la app externa lo envio asi.") ||
		!stringInSetV0(task.AcceptanceCriteria, "No redactar un tema de 50 folios sin paquete suficiente; pedir division editorial a la app externa si excede contexto o trazabilidad.") ||
		containsFragmentInSetForTestV0(task.AcceptanceCriteria, "OPES") {
		t.Fatalf("task=%+v", task)
	}
}

func TestAppChangeDirectorDecisionSourceV0SaneaGuardasInternasEnExpansion(t *testing.T) {
	record := appChangeRecordForSourceTestV0()
	record.Request.AllowedWriteSet = []string{"external/opes/expand_topic_from_summary/job-ref-001"}
	record.Request.AcceptanceCriteria = []string{
		"devolver artifact_type=topic_expansion_package",
		"no leer DB ni ficheros internos de OPES",
	}
	record.Request.ExternalWork = &orquestaappchange.AppChangeExternalWorkV0{
		ProjectRef:    "opes",
		InterfaceRefs: []string{"opes-rest-v0", "opes-mcp-v0"},
		WorkKind:      "expand_topic_from_summary",
		WorkRefs:      []string{"opes-job-001", "opes-topic-001"},
	}
	store := orquestaappchange.NewInMemoryAppChangeStoreV0(record)
	run := appChangeRunForSourceTestV0(record)

	decisions, err := (AppChangeDirectorDecisionSourceV0{Store: store}).
		ListDirectorAgentDecisionsV0(
			context.Background(),
			orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{Run: run},
		)

	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	task := decisions[6].CreateMicrotask.Task
	if task.Title != "Ampliar tema documental externo" ||
		task.Summary != "Ampliar tema documental completo desde resumen trazable y paquete de dominio suficiente." ||
		!stringInSetV0(task.RequiredTests, "validar topic_expansion_package") ||
		!stringInSetV0(task.RequiredTests, "validar longitud declarada del tema grande") {
		t.Fatalf("task=%+v", task)
	}
	if stringInSetV0(task.AcceptanceCriteria, "no leer DB ni ficheros internos de OPES") {
		t.Fatalf("criterio no saneado: %+v", task.AcceptanceCriteria)
	}
	if _, err := orquestacoreworkflow.NewWorkflowTaskV0(workflowTaskFromDirectorTaskForTestV0(task)); err != nil {
		t.Fatalf("workflow task invalida tras saneo: %v task=%+v", err, task)
	}
}

func TestAppChangeDirectorDecisionSourceV0SaneaCriteriosOperativosSinBloquearAutoPlan(t *testing.T) {
	record := appChangeRecordForSourceTestV0()
	record.Request.AllowedWriteSet = []string{"modulos/orquesta-director-agent"}
	record.Request.AcceptanceCriteria = []string{
		"Reglas de hexagonal, i18n, archivos manejables y token economy llegan a agentes desde codigo.",
		"No fijar Codex, modelo ni provider en nucleo.",
	}
	store := orquestaappchange.NewInMemoryAppChangeStoreV0(record)
	run := appChangeRunForSourceTestV0(record)

	decisions, err := (AppChangeDirectorDecisionSourceV0{Store: store}).
		ListDirectorAgentDecisionsV0(
			context.Background(),
			orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{Run: run},
		)

	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	if len(decisions) == 0 {
		t.Fatalf("decisions vacias")
	}
	task := decisions[6].CreateMicrotask.Task
	if !stringInSetV0(task.AcceptanceCriteria, "Reglas de hexagonal, i18n, archivos manejables y economia de contexto llegan a agentes desde codigo.") ||
		containsFragmentInSetForTestV0(task.AcceptanceCriteria, "token") ||
		containsFragmentInSetForTestV0(task.AcceptanceCriteria, "Codex") ||
		containsFragmentInSetForTestV0(task.AcceptanceCriteria, "modelo") ||
		containsFragmentInSetForTestV0(task.AcceptanceCriteria, "provider") {
		t.Fatalf("criteria=%+v", task.AcceptanceCriteria)
	}
}

func TestAppChangeDirectorDecisionSourceV0CompactaCriteriosExternosAlLimiteDelDirector(t *testing.T) {
	record := appChangeRecordForSourceTestV0()
	record.Request.AllowedWriteSet = nil
	record.Request.AcceptanceCriteria = []string{
		"markdown valido",
		"sin placeholders",
		"fuentes verificables cuando aplique",
		"contenido listo para revision editorial",
		"entrega en fichero bajo write set externo",
	}
	record.Request.ExternalWork = &orquestaappchange.AppChangeExternalWorkV0{
		ProjectRef:    "opes",
		JobRef:        "34f317ccf529318fd2943466c246756f",
		InterfaceRefs: []string{"opes-rest-v0", "opes-mcp-v0"},
		WorkKind:      "draft_content_block",
		WorkRefs: []string{
			"opes-topic-e712a4a2e848d18d1f5efd90434a1ea7",
			"opes-chapter-9088fb1124055f2a0e69d6964c73590e",
		},
	}
	store := orquestaappchange.NewInMemoryAppChangeStoreV0(record)
	run := appChangeRunForSourceTestV0(record)

	decisions, err := (AppChangeDirectorDecisionSourceV0{Store: store}).
		ListDirectorAgentDecisionsV0(
			context.Background(),
			orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{Run: run},
		)

	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	task := decisions[6].CreateMicrotask.Task
	if len(task.AcceptanceCriteria) != maxAppChangeTaskCriteriaV0 ||
		!stringInSetV0(task.AcceptanceCriteria, "markdown valido") ||
		!stringInSetV0(task.AcceptanceCriteria, "contenido listo para revision editorial") {
		t.Fatalf("criteria=%+v", task.AcceptanceCriteria)
	}
	if issues := orquestadirectoragent.ValidateDirectorAgentDecisionV0(decisions[6]); len(issues) != 0 {
		t.Fatalf("decision invalida: %+v", issues)
	}
}

func TestAppChangeDirectorDecisionSourceV0ResumenPermiteGranularidadPequena(t *testing.T) {
	record := appChangeRecordForSourceTestV0()
	record.Request.AllowedWriteSet = nil
	record.Request.ExternalWork = &orquestaappchange.AppChangeExternalWorkV0{
		ProjectRef:    "opes",
		InterfaceRefs: []string{"opes-rest-v0", "opes-mcp-v0"},
		WorkKind:      "summarize_chapter",
		WorkRefs:      []string{"opes-topic-001", "opes-chapter-001"},
	}
	store := orquestaappchange.NewInMemoryAppChangeStoreV0(record)
	run := appChangeRunForSourceTestV0(record)

	decisions, err := (AppChangeDirectorDecisionSourceV0{Store: store}).
		ListDirectorAgentDecisionsV0(
			context.Background(),
			orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{Run: run},
		)

	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	task := decisions[6].CreateMicrotask.Task
	if task.Title != "Resumir capitulo documental externo" ||
		task.Summary != "Crear resumen derivado compacto con trazabilidad a bloques, capitulos, tema y fuentes de origen." ||
		!stringInSetV0(task.AcceptanceCriteria, "Para summarize_* y create_exam_outline, aceptar granularidad pequena porque son artefactos derivados y trazables.") ||
		!stringInSetV0(task.AcceptanceCriteria, "Conservar matices, excepciones, plazos, organos, fuentes criticas y advertencias de examen del material de origen.") ||
		!stringInSetV0(task.RequiredTests, "validar trazabilidad del resumen") ||
		!stringInSetV0(task.RequiredTests, "validar conservacion de matices criticos") {
		t.Fatalf("task=%+v", task)
	}
}

func TestAppChangeDirectorDecisionSourceV0CaracterizaAliasDocumentalGenerico(t *testing.T) {
	record := appChangeRecordForSourceTestV0()
	record.Request.AllowedWriteSet = nil
	record.Request.ExternalWork = &orquestaappchange.AppChangeExternalWorkV0{
		ProjectRef:    "external-editorial",
		InterfaceRefs: []string{"domain-contract-v0"},
		WorkKind:      "validate_topic",
		WorkRefs:      []string{"topic-001"},
	}
	store := orquestaappchange.NewInMemoryAppChangeStoreV0(record)
	run := appChangeRunForSourceTestV0(record)

	decisions, err := (AppChangeDirectorDecisionSourceV0{Store: store}).
		ListDirectorAgentDecisionsV0(
			context.Background(),
			orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{Run: run},
		)

	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	task := decisions[6].CreateMicrotask.Task
	if task.Title != "Resolver trabajo externo de app" ||
		task.Summary != "Resolver trabajo documental con paquete de dominio suficiente: temario, esquema, objetivo, fuentes, criterios y longitud si llegan." ||
		!stringInSetV0(task.WriteSet, "external/external-editorial/validate_topic") ||
		!stringInSetV0(task.AcceptanceCriteria, "Usar temario, esquema, objetivo, fuentes, criterios y longitud si llegan en input_fields.") ||
		!stringInSetV0(task.RequiredTests, "validar paquete documental de dominio") ||
		!stringInSetV0(task.RequiredTests, "validar fuentes y criterios documentales") {
		t.Fatalf("task=%+v", task)
	}
}

func TestAppChangeDirectorDecisionSourceV0NoInventaMicrotareaSinWriteSetNiExternalWork(t *testing.T) {
	record := appChangeRecordForSourceTestV0()
	record.Request.AllowedWriteSet = nil
	store := orquestaappchange.NewInMemoryAppChangeStoreV0(record)
	run := appChangeRunForSourceTestV0(record)

	decisions, err := (AppChangeDirectorDecisionSourceV0{Store: store}).
		ListDirectorAgentDecisionsV0(
			context.Background(),
			orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{Run: run},
		)

	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	if len(decisions) != 0 {
		t.Fatalf("decisions=%+v", decisions)
	}
}

func TestAppChangeDirectorDecisionSourceV0NoInventaMicrotareaSinCriterios(t *testing.T) {
	record := appChangeRecordForSourceTestV0()
	record.Request.AcceptanceCriteria = nil
	store := orquestaappchange.NewInMemoryAppChangeStoreV0(record)
	run := appChangeRunForSourceTestV0(record)

	decisions, err := (AppChangeDirectorDecisionSourceV0{Store: store}).
		ListDirectorAgentDecisionsV0(
			context.Background(),
			orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{Run: run},
		)

	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	if len(decisions) != 0 {
		t.Fatalf("decisions=%+v", decisions)
	}
}

func TestAppChangeDirectorDecisionSourceV0ConsumeCambioRecibidoComoEvento(t *testing.T) {
	store := orquestaappchange.NewInMemoryAppChangeStoreV0()
	event := orquestaappchange.AppChangeIntentEventV0{
		SchemaVersion:      orquestaappchange.AppChangeIntentEventSchemaV0,
		EventID:            "event-ref-source-mid-001",
		Source:             "orquesta-mcp",
		OccurredAt:         "2026-05-10T10:25:00Z",
		RunRef:             "run-ref-source-001",
		ChangeRef:          "change-ref-source-mid-001",
		UserIntent:         "Quiero modificar la web para mostrar filtros.",
		AcceptanceCriteria: []string{"filtros visibles"},
		AllowedWriteSet:    []string{"web/agenda"},
	}

	result, err := orquestaappchange.ReceiveAppChangeIntentEventV0(
		context.Background(),
		event,
		orquestaappchange.AppChangePortsV0{
			Store:            store,
			DirectorNotifier: appChangeSourceNoopNotifierV0{},
		},
	)
	if err != nil {
		t.Fatalf("ReceiveAppChangeIntentEventV0: %v", err)
	}
	run := orquestacoreworkflow.OrchestrationRunV0{
		RunID:             event.RunRef,
		CurrentPhase:      orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		DirectorQuestions: []string{result.DirectorQuestionRef},
		Decisions:         []string{"decision-ref-base-001"},
	}
	decisions, err := (AppChangeDirectorDecisionSourceV0{Store: store}).
		ListDirectorAgentDecisionsV0(
			context.Background(),
			orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{Run: run},
		)
	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	if len(decisions) != 3 ||
		decisions[2].CommandType != orquestadirectoragent.DirectorAgentCommandCreateMicrotaskV0 {
		t.Fatalf("decisions=%+v", decisions)
	}
}

func TestAppChangeDirectorDecisionSourceV0AbreRevisionTrasEntrega(t *testing.T) {
	record := appChangeRecordForSourceTestV0()
	store := orquestaappchange.NewInMemoryAppChangeStoreV0(record)
	run := appChangeRunForSourceTestV0(record)
	refs := appChangeRefsV0(record.Request.ChangeRef)
	run.CurrentPhase = orquestacoreworkflow.OrchestrationPhaseProgramacionV0
	run.DirectorAnsweredQuestions = []string{refs.QuestionRef}
	run.Tasks = []string{"task-ref-base-001", refs.TaskRef}
	run.Deliveries = []string{"delivery-ref-base-001", "delivery-ref-change-001"}

	decisions, err := (AppChangeDirectorDecisionSourceV0{Store: store}).
		ListDirectorAgentDecisionsV0(
			context.Background(),
			orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{Run: run},
		)

	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	if len(decisions) != 1 ||
		decisions[0].CommandType != orquestadirectoragent.DirectorAgentCommandOpenPhaseV0 ||
		decisions[0].OpenPhase == nil ||
		decisions[0].OpenPhase.PhaseID != string(orquestacoreworkflow.OrchestrationPhaseRevisionV0) {
		t.Fatalf("decisions=%+v", decisions)
	}
}

func TestAppChangeDirectorDecisionSourceV0PriorizaCambioPendienteAntesDeRevision(t *testing.T) {
	delivered := appChangeRecordForSourceTestV0()
	pending := appChangeRecordForSourceTestV0()
	pending.Request.ChangeRef = "change-ref-source-pending-002"
	store := orquestaappchange.NewInMemoryAppChangeStoreV0(delivered, pending)
	run := appChangeRunForSourceTestV0(delivered)
	deliveredRefs := appChangeRefsV0(delivered.Request.ChangeRef)
	pendingRefs := appChangeRefsV0(pending.Request.ChangeRef)
	run.CurrentPhase = orquestacoreworkflow.OrchestrationPhaseProgramacionV0
	run.DirectorQuestions = append(run.DirectorQuestions, pendingRefs.QuestionRef)
	run.DirectorAnsweredQuestions = []string{deliveredRefs.QuestionRef}
	run.Tasks = []string{deliveredRefs.TaskRef}
	run.Deliveries = []string{"delivery-ref-change-001"}
	run.Decisions = []string{"decision-ref-source-base-001"}

	decisions, err := (AppChangeDirectorDecisionSourceV0{Store: store}).
		ListDirectorAgentDecisionsV0(
			context.Background(),
			orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{Run: run},
		)

	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	if len(decisions) == 0 ||
		decisions[0].CommandType != orquestadirectoragent.DirectorAgentCommandAnswerQuestionV0 ||
		containsOpenReviewDecisionForTestV0(decisions) {
		t.Fatalf("decisions=%+v", decisions)
	}
}

func appChangeRecordForSourceTestV0() orquestaappchange.AppChangeRecordV0 {
	return orquestaappchange.AppChangeRecordV0{
		Request: orquestaappchange.AppChangeRequestV0{
			RunRef:             "run-ref-source-001",
			ChangeRef:          "change-ref-source-001",
			UserIntent:         "Cambiar la web para mostrar vista semanal.",
			AcceptanceCriteria: []string{"vista semanal visible"},
			AllowedWriteSet:    []string{"web/agenda"},
		},
	}
}

func containsOpenReviewDecisionForTestV0(
	decisions []orquestadirectoragent.DirectorAgentDecisionV0,
) bool {
	for _, decision := range decisions {
		if decision.OpenPhase != nil &&
			decision.OpenPhase.PhaseID == string(orquestacoreworkflow.OrchestrationPhaseRevisionV0) {
			return true
		}
	}
	return false
}

func containsFragmentInSetForTestV0(values []string, fragment string) bool {
	for _, value := range values {
		if strings.Contains(value, fragment) {
			return true
		}
	}
	return false
}

func workflowTaskFromDirectorTaskForTestV0(
	task orquestadirectoragent.DirectorAgentMicrotaskV0,
) orquestacoreworkflow.WorkflowTaskV0 {
	refs := make([]orquestacoreworkflow.WorkflowFunctionContractRefV0, 0, len(task.FunctionContractRefs))
	for _, ref := range task.FunctionContractRefs {
		refs = append(refs, orquestacoreworkflow.WorkflowFunctionContractRefV0{
			ContractRef:  ref.ContractRef,
			FunctionName: ref.FunctionName,
		})
	}
	return orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:        task.SchemaVersion,
		TaskID:               task.TaskID,
		RunID:                task.RunID,
		PhaseID:              orquestacoreworkflow.OrchestrationPhaseIDV0(task.PhaseID),
		WorkProfileKind:      orquestacoreworkflow.WorkProfileKindV0(task.WorkProfileKind),
		Title:                task.Title,
		Summary:              task.Summary,
		WriteSet:             task.WriteSet,
		AcceptanceCriteria:   task.AcceptanceCriteria,
		RequiredTests:        task.RequiredTests,
		DependsOn:            task.DependsOn,
		ContextRefs:          task.ContextRefs,
		FunctionContractRefs: refs,
	}
}

type appChangeSourceNoopNotifierV0 struct{}

func (appChangeSourceNoopNotifierV0) NotifyAppChangeRequestedV0(
	context.Context,
	orquestaappchange.AppChangeRecordV0,
) (orquestaappchange.AppChangeDirectorNotificationV0, error) {
	return orquestaappchange.AppChangeDirectorNotificationV0{}, nil
}

func appChangeRunForSourceTestV0(
	record orquestaappchange.AppChangeRecordV0,
) orquestacoreworkflow.OrchestrationRunV0 {
	return orquestacoreworkflow.OrchestrationRunV0{
		RunID:             record.Request.RunRef,
		CurrentPhase:      orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		DirectorQuestions: []string{appChangeQuestionRefV0(record.Request.ChangeRef)},
	}
}
