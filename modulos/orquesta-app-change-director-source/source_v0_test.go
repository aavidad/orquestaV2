package orquestaappchangedirectorsource

import (
	"context"
	"strings"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
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
	if len(decisions) != 3 {
		t.Fatalf("decisions=%+v", decisions)
	}
	if decisions[0].CommandType != orquestadirectoragent.DirectorAgentCommandAnswerQuestionV0 ||
		decisions[1].CommandType != orquestadirectoragent.DirectorAgentCommandPublishContractV0 ||
		microtaskDecisionForTestV0(t, decisions).CommandType != orquestadirectoragent.DirectorAgentCommandCreateMicrotaskV0 {
		t.Fatalf("decisions=%+v", decisions)
	}
	refs := appChangeRefsV0(record.Request.ChangeRef)
	if decisions[1].PublishContract == nil || decisions[1].PublishContract.DecisionRef != refs.AnswerRef {
		t.Fatalf("publish_contract.decision_ref=%+v want %s", decisions[1].PublishContract, refs.AnswerRef)
	}
	task := microtaskDecisionForTestV0(t, decisions).CreateMicrotask.Task
	if len(task.WriteSet) != 1 || task.WriteSet[0] != "web/agenda" {
		t.Fatalf("write_set=%+v", task.WriteSet)
	}
	if task.WorkProfileKind != "implementation" {
		t.Fatalf("work_profile_kind=%s", task.WorkProfileKind)
	}
	if len(task.RequiredTests) == 0 {
		t.Fatalf("required_tests vacio: %+v", task)
	}
	if issues := orquestadirectoragent.ValidateDirectorAgentDecisionV0(microtaskDecisionForTestV0(t, decisions)); len(issues) != 0 {
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
	task := microtaskDecisionForTestV0(t, decisions).CreateMicrotask.Task
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
	task := microtaskDecisionForTestV0(t, decisions).CreateMicrotask.Task
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
	task := microtaskDecisionForTestV0(t, decisions).CreateMicrotask.Task
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
	contract := decisions[1].PublishContract
	task := microtaskDecisionForTestV0(t, decisions).CreateMicrotask.Task
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

func TestAppChangeDirectorDecisionSourceV0MaterializaSeisSubrolesOPES(t *testing.T) {
	record := appChangeRecordForSourceTestV0()
	record.Request.AllowedWriteSet = nil
	record.Request.ExternalWork = &orquestaappchange.AppChangeExternalWorkV0{
		ProjectRef:    "opes",
		JobRef:        "opes-job-tema-001",
		InterfaceRefs: []string{"opes-rest-v0", "opes.padre-tema-6-subroles.v1"},
		WorkKind:      "draft_content_block",
		WorkRefs:      []string{"curso-servicios-multiples", "tema-001"},
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
	microtasks := microtaskDecisionsForTestV0(decisions)
	if len(microtasks) != 7 {
		t.Fatalf("microtasks=%d decisions=%+v", len(microtasks), decisions)
	}
	parent, children := appChangeOPESSubroleMicrotasksForTestV0(microtasks)
	if parent.TaskID != appChangeRefsV0(record.Request.ChangeRef).TaskRef ||
		parent.MaxChildAgents != 6 ||
		len(parent.ChildTaskRefs) != 6 ||
		len(parent.DependsOn) != 6 ||
		parent.CohortRef == "" ||
		parent.WaveRef == "" ||
		containsFragmentInSetForTestV0(parent.WriteSet, "/subroles/") ||
		!stringInSetV0(parent.ContextRefs, "opes-padre-tema-6-subroles-v1") {
		t.Fatalf("parent=%+v", parent)
	}
	requireStringSetForTestV0(t, parent.WriteSet, []string{
		"external/opes/draft_content_block",
		"external/opes/opes-job-tema-001",
		"external/opes/curso-servicios-multiples",
		"external/opes/tema-001",
		"external/opes/draft_content_block/coordinacion",
		"external/opes/opes-job-tema-001/coordinacion",
		"external/opes/curso-servicios-multiples/coordinacion",
		"external/opes/tema-001/coordinacion",
	})
	for _, decision := range microtasks {
		task := decision.CreateMicrotask.Task
		if _, err := orquestacoreworkflow.NewWorkflowTaskV0(workflowTaskFromDirectorTaskForTestV0(task)); err != nil {
			t.Fatalf("workflow task invalida %s: %v task=%+v", task.TaskID, err, task)
		}
	}
	seenChildren := map[string]bool{}
	for _, child := range children {
		seenChildren[child.TaskID] = true
		subroleRef := strings.TrimPrefix(child.TaskID, parent.TaskID+"-subrole-")
		if child.ParentTaskRef != parent.TaskID ||
			subroleRef == child.TaskID ||
			subroleRef == "" ||
			child.DelegationDepth != 1 ||
			child.CohortRef != parent.CohortRef ||
			child.WaveRef != parent.WaveRef ||
			len(child.DependsOn) != 0 ||
			len(child.ChildTaskRefs) != 0 ||
			containsFragmentInSetForTestV0(child.WriteSet, "/coordinacion") ||
			containsFragmentInSetForTestV0(child.WriteSet, "/coordinacion/subroles/") ||
			!containsFragmentInSetForTestV0(child.ContextRefs, "opes-subrole-") {
			t.Fatalf("child=%+v parent=%+v", child, parent)
		}
		requireStringSetForTestV0(t, child.WriteSet, []string{
			"external/opes/draft_content_block/subroles/" + subroleRef,
			"external/opes/opes-job-tema-001/subroles/" + subroleRef,
			"external/opes/curso-servicios-multiples/subroles/" + subroleRef,
			"external/opes/tema-001/subroles/" + subroleRef,
		})
		if (subroleRef == "fuentes" || subroleRef == "reutilizacion") &&
			(!strings.Contains(child.AcceptanceCriteria[0], "excluir backups") ||
				!strings.Contains(child.AcceptanceCriteria[0], "runtime")) {
			t.Fatalf("subrole %s sin acotacion de busqueda: criteria=%+v", subroleRef, child.AcceptanceCriteria)
		}
	}
	for _, childRef := range parent.ChildTaskRefs {
		if !seenChildren[childRef] {
			t.Fatalf("child_ref %s no materializado: seen=%v", childRef, seenChildren)
		}
		if !stringInSetV0(parent.DependsOn, childRef) {
			t.Fatalf("parent.depends_on no contiene child_ref %s: parent=%+v", childRef, parent)
		}
	}
}

func TestAppChangeDirectorDecisionSourceV0OPESSubrolesPadreConservaWriteSetProductoAutorizado(t *testing.T) {
	record := appChangeRecordForSourceTestV0()
	record.Request.AllowedWriteSet = []string{"temas/tema_032"}
	record.Request.ExternalWork = &orquestaappchange.AppChangeExternalWorkV0{
		ProjectRef:    "opes",
		JobRef:        "opes-job-tema-032",
		InterfaceRefs: []string{"opes-rest-v0", "opes.padre-tema-6-subroles.v1"},
		WorkKind:      "draft_content_block",
		WorkRefs:      []string{"curso-servicios-multiples", "tema-032"},
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
	microtasks := microtaskDecisionsForTestV0(decisions)
	if len(microtasks) != 7 {
		t.Fatalf("microtasks=%d decisions=%+v", len(microtasks), decisions)
	}
	parent, children := appChangeOPESSubroleMicrotasksForTestV0(microtasks)
	if parent.TaskID == "" ||
		len(parent.ChildTaskRefs) != 6 ||
		len(parent.DependsOn) != 6 ||
		containsFragmentInSetForTestV0(parent.WriteSet, "/subroles/") {
		t.Fatalf("parent=%+v", parent)
	}
	requireStringSetForTestV0(t, parent.WriteSet, []string{
		"temas/tema_032",
		"temas/tema_032/coordinacion",
	})
	redaccionRef := parent.TaskID + "-subrole-redaccion"
	for _, childRef := range parent.ChildTaskRefs {
		if !stringInSetV0(parent.DependsOn, childRef) {
			t.Fatalf("parent.depends_on no contiene %s: parent=%+v", childRef, parent)
		}
	}
	var redaccion orquestadirectoragent.DirectorAgentMicrotaskV0
	for _, child := range children {
		if child.TaskID == redaccionRef {
			redaccion = child
			break
		}
	}
	if redaccion.TaskID == "" ||
		!stringInSetV0(redaccion.WriteSet, "temas/tema_032/subroles/redaccion") ||
		stringInSetV0(redaccion.WriteSet, "temas/tema_032") {
		t.Fatalf("redaccion=%+v children=%+v", redaccion, children)
	}
}

func TestAppChangeDirectorDecisionSourceV0NoExpandeSubrolesFueraDeOPES(t *testing.T) {
	record := appChangeRecordForSourceTestV0()
	record.Request.AllowedWriteSet = nil
	record.Request.ExternalWork = &orquestaappchange.AppChangeExternalWorkV0{
		ProjectRef:    "external-editorial",
		InterfaceRefs: []string{"domain-contract-v0"},
		WorkKind:      "validate_topic",
		WorkRefs:      []string{"topic-001"},
		InputFields: []orquestadomainwork.DomainWorkFieldV0{{
			Name:  "subroles_required",
			Value: "6",
		}},
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
	microtasks := microtaskDecisionsForTestV0(decisions)
	if len(microtasks) != 1 {
		t.Fatalf("microtasks=%d decisions=%+v", len(microtasks), decisions)
	}
	task := microtasks[0].CreateMicrotask.Task
	if task.MaxChildAgents != 0 ||
		len(task.ChildTaskRefs) != 0 ||
		task.CohortRef != "" ||
		task.WaveRef != "" ||
		!stringInSetV0(task.WriteSet, "external/external-editorial/validate_topic") ||
		containsFragmentInSetForTestV0(task.WriteSet, "/subroles/") ||
		containsFragmentInSetForTestV0(task.WriteSet, "/coordinacion") {
		t.Fatalf("task=%+v", task)
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
	task := microtaskDecisionForTestV0(t, decisions).CreateMicrotask.Task
	if task.Title != "Planificar tema documental externo" ||
		task.Summary != "Crear plan documental validable; no redactar ni ensamblar el documento final en esta tarea." ||
		!stringInSetV0(task.AcceptanceCriteria, "Devolver artifact_type=document_plan compatible con DomainDocumentPlanV0.") ||
		!stringInSetV0(task.AcceptanceCriteria, "No redactar el documento final en esta tarea; solo plan verificable.") ||
		!stringInSetV0(task.RequiredTests, "validar document_plan") ||
		!stringInSetV0(task.RequiredTests, "validar secciones y entregables del plan") {
		t.Fatalf("task=%+v", task)
	}
}

func TestAppChangeDirectorDecisionSourceV0ProyectaCierreTemarioExterno(t *testing.T) {
	record := appChangeRecordForSourceTestV0()
	record.Request.AllowedWriteSet = nil
	record.Request.ExternalWork = &orquestaappchange.AppChangeExternalWorkV0{
		ProjectRef:    "opes",
		InterfaceRefs: []string{"opes-rest-v0", "opes-mcp-v0"},
		WorkKind:      "finalize_temario_package",
		WorkRefs:      []string{"opes-job-finalize-001", "opes-course-a2-informatica"},
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
	task := microtaskDecisionForTestV0(t, decisions).CreateMicrotask.Task
	if task.Title != "Cerrar paquete de temario externo" ||
		task.Summary != "Cerrar paquete de dominio externo con matriz de evidencias, validaciones y bloqueos causales antes de cualquier publicacion." ||
		!stringInSetV0(task.WriteSet, "external/opes/finalize_temario_package") ||
		!stringInSetV0(task.AcceptanceCriteria, "Devolver artifact_type=final_domain_package con manifest, matriz de evidencias y estado de validacion.") ||
		!stringInSetV0(task.AcceptanceCriteria, "No ejecutar subida a produccion desde esta tarea; dejarla como trabajo posterior con confirmacion explicita.") ||
		!stringInSetV0(task.RequiredTests, "validar final_domain_package") ||
		!stringInSetV0(task.RequiredTests, "validar matriz de evidencias de cierre") ||
		!stringInSetV0(task.RequiredTests, "validar que no hay subida a produccion sin confirmacion") {
		t.Fatalf("task=%+v", task)
	}
}

func TestAppChangeDirectorDecisionSourceV0ProyectaCierreTemaExterno(t *testing.T) {
	record := appChangeRecordForSourceTestV0()
	record.Request.AllowedWriteSet = nil
	record.Request.ExternalWork = &orquestaappchange.AppChangeExternalWorkV0{
		ProjectRef:    "opes",
		InterfaceRefs: []string{"opes-rest-v0", "opes-mcp-v0"},
		WorkKind:      "finalize_topic_package",
		WorkRefs:      []string{"opes-job-finalpkg-001", "opes-topic-001"},
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
	task := microtaskDecisionForTestV0(t, decisions).CreateMicrotask.Task
	if task.Title != "Cerrar paquete de tema externo" ||
		task.Summary != "Cerrar paquete de dominio externo con matriz de evidencias, validaciones y bloqueos causales antes de cualquier publicacion." ||
		!stringInSetV0(task.WriteSet, "external/opes/finalize_topic_package") ||
		!stringInSetV0(task.RequiredTests, "validar final_domain_package") {
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
	if len(decisions) != 3 {
		t.Fatalf("decisions=%+v", decisions)
	}
	task := microtaskDecisionForTestV0(t, decisions).CreateMicrotask.Task
	if task.Title != "Redactar bloque documental OPES" ||
		task.Summary != "Redactar unidad editorial amplia con paquete de dominio suficiente; no trocear un temario largo en parrafos sin continuidad." ||
		!stringInSetV0(task.WriteSet, "external/opes/draft_content_block") ||
		!stringInSetV0(task.WriteSet, "external/opes/opes-job-001") ||
		!stringInSetV0(task.AcceptanceCriteria, "Tratar el paquete de dominio OPES como entrada suficiente, no como contexto minimo.") ||
		!stringInSetV0(task.AcceptanceCriteria, "Para draft_content_block, usar paquete editorial suficiente y no sobreatomizar: bloque, subcapitulo o capitulo coherente si OPES lo envio asi.") ||
		!stringInSetV0(task.AcceptanceCriteria, "Si el paquete no da para una version larga trazable, redactar version parcial usable y dejar tarea de ampliacion editorial para OPES.") ||
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
	task := microtaskDecisionForTestV0(t, decisions).CreateMicrotask.Task
	if task.Title != "Redactar bloque documental externo" ||
		!stringInSetV0(task.AcceptanceCriteria, "Para draft_content_block, usar paquete editorial suficiente y no sobreatomizar: bloque, subcapitulo o capitulo coherente si la app externa lo envio asi.") ||
		!stringInSetV0(task.AcceptanceCriteria, "Si el paquete no da para una version larga trazable, redactar version parcial usable y dejar tarea de ampliacion editorial para la app externa.") ||
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
	task := microtaskDecisionForTestV0(t, decisions).CreateMicrotask.Task
	if task.Title != "Ampliar tema documental externo" ||
		task.Summary != "Ampliar tema documental completo desde resumen trazable y paquete de dominio suficiente." ||
		!stringInSetV0(task.RequiredTests, "validar topic_expansion_package") ||
		!stringInSetV0(task.RequiredTests, "validar longitud declarada del tema grande") {
		t.Fatalf("task=%+v", task)
	}
	if !stringInSetV0(task.AcceptanceCriteria, "no leer DB ni ficheros internos de OPES") {
		t.Fatalf("criterio de dominio perdido: %+v", task.AcceptanceCriteria)
	}
	if _, err := orquestacoreworkflow.NewWorkflowTaskV0(workflowTaskFromDirectorTaskForTestV0(task)); err != nil {
		t.Fatalf("workflow task invalida tras saneo: %v task=%+v", err, task)
	}
}

func TestAppChangeDirectorDecisionSourceV0ExpansionOPESConRequiredTestLargoNoBloquea(t *testing.T) {
	record := appChangeRecordForSourceTestV0()
	record.Request.RunRef = "run-ref-opes-tractorista-t001-rework-extension-20260622"
	record.Request.ChangeRef = "change-opes-tractorista-t001-extension-minimum-AP"
	record.Request.UserIntent = "Ampliar tema OPES de Tractorista hasta minimo AP sin generar audios."
	record.Request.AllowedWriteSet = []string{
		"opes-salidas/diputacion_granada/cursos_opes_2026/administracion_especial/operario-tractorista-grupo-5/temas/tema_001",
	}
	record.Request.AcceptanceCriteria = []string{
		"el ampliado del tema propio queda entre 3150 y 5400 palabras utiles, sin relleno ni repeticiones mecanicas",
		"castellano correcto: tildes, ñ y signos ¿? ¡! completos",
	}
	record.Request.RequiredTests = []string{
		"python3 -c \"import pathlib,re; p=pathlib.Path('opes-salidas/diputacion_granada/cursos_opes_2026/administracion_especial/operario-tractorista-grupo-5/temas/tema_001/02_markdown/tema_001_ampliado.md'); text=p.read_text(encoding='utf-8'); words=len(re.findall(r'\\\\b\\\\w+\\\\b',text)); print(words); assert words >= 3150\"",
	}
	record.Request.ExternalWork = &orquestaappchange.AppChangeExternalWorkV0{
		ProjectRef:    "opes",
		JobRef:        "job-opes-tractorista-t001-expand-minimum-AP-20260622",
		InterfaceRefs: []string{"opes-local-files-v0", "orquesta-external-work-v0"},
		WorkKind:      "expand_topic_from_summary",
		WorkRefs:      []string{"tractorista-grupo-5", "topic-001", "minimum-extension-AP"},
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
	taskDecision := microtaskDecisionForTestV0(t, decisions)
	task := taskDecision.CreateMicrotask.Task
	if task.Title != "Ampliar tema documental externo" ||
		!stringInSetV0(task.RequiredTests, appChangeExternalRequiredTestMarkerV0) ||
		containsFragmentInSetForTestV0(task.RequiredTests, "pathlib.Path") {
		t.Fatalf("task=%+v", task)
	}
	for _, requiredTest := range task.RequiredTests {
		if len(requiredTest) > 300 {
			t.Fatalf("required_test demasiado largo: %d %q", len(requiredTest), requiredTest)
		}
	}
	if issues := orquestadirectoragent.ValidateDirectorAgentDecisionV0(taskDecision); len(issues) != 0 {
		t.Fatalf("decision invalida: %+v", issues)
	}
	if _, err := orquestacoreworkflow.NewWorkflowTaskV0(workflowTaskFromDirectorTaskForTestV0(task)); err != nil {
		t.Fatalf("workflow task invalida: %v task=%+v", err, task)
	}
}

func TestAppChangeDirectorDecisionSourceV0ExternalWorkAccionableSinCriteriosCreaMicrotarea(t *testing.T) {
	record := appChangeRecordForSourceTestV0()
	record.Request.UserIntent = "Resolver contrato externo OPES y devolver artefacto verificable."
	record.Request.AllowedWriteSet = nil
	record.Request.AcceptanceCriteria = nil
	record.Request.ExternalWork = &orquestaappchange.AppChangeExternalWorkV0{
		ProjectRef:    "opes",
		JobRef:        "job-opes-tractorista-t001-expand-minimum-AP-20260622",
		InterfaceRefs: []string{"opes-local-files-v0", "orquesta-external-work-v0"},
		WorkKind:      "expand_topic_from_summary",
		WorkRefs:      []string{"tractorista-grupo-5", "topic-001", "minimum-extension-AP"},
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
	task := microtaskDecisionForTestV0(t, decisions).CreateMicrotask.Task
	if task.Title != "Ampliar tema documental externo" ||
		!stringInSetV0(task.AcceptanceCriteria, "Resolver solo el contrato externo de dominio con refs opacas.") ||
		!stringInSetV0(task.AcceptanceCriteria, "Usar temario, esquema, objetivo, fuentes, criterios y longitud si llegan en input_fields.") ||
		!stringInSetV0(task.RequiredTests, "validar topic_expansion_package") {
		t.Fatalf("task=%+v", task)
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
	task := microtaskDecisionForTestV0(t, decisions).CreateMicrotask.Task
	if !stringInSetV0(task.AcceptanceCriteria, "Reglas de hexagonal, i18n, archivos manejables y token economy llegan a agentes desde codigo.") ||
		!stringInSetV0(task.AcceptanceCriteria, "No fijar Codex, modelo ni provider en nucleo.") {
		t.Fatalf("criteria=%+v", task.AcceptanceCriteria)
	}
}

func TestAppChangeDirectorDecisionSourceV0BloqueaDetalleSensibleEnAutoPlan(t *testing.T) {
	enableAppChangeSourceRailsModeEnforcedForTestV0(t)
	record := appChangeRecordForSourceTestV0()
	record.Request.AcceptanceCriteria = []string{
		"No persistir api_key=valor en criterios publicos.",
	}
	store := orquestaappchange.NewInMemoryAppChangeStoreV0(record)
	run := appChangeRunForSourceTestV0(record)

	decisions, err := (AppChangeDirectorDecisionSourceV0{Store: store}).
		ListDirectorAgentDecisionsV0(
			context.Background(),
			orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{Run: run},
		)

	if err != nil {
		return
	}
	for _, decision := range decisions {
		if decision.CreateMicrotask != nil {
			t.Fatalf("detalle sensible no debe crear microtarea automatica: %+v", decision)
		}
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
	task := microtaskDecisionForTestV0(t, decisions).CreateMicrotask.Task
	if len(task.AcceptanceCriteria) != maxAppChangeTaskCriteriaV0 ||
		!stringInSetV0(task.AcceptanceCriteria, "markdown valido") ||
		!stringInSetV0(task.AcceptanceCriteria, "contenido listo para revision editorial") {
		t.Fatalf("criteria=%+v", task.AcceptanceCriteria)
	}
	if issues := orquestadirectoragent.ValidateDirectorAgentDecisionV0(microtaskDecisionForTestV0(t, decisions)); len(issues) != 0 {
		t.Fatalf("decision invalida: %+v", issues)
	}
}

func TestAppChangeDirectorDecisionSourceV0CompactaCriteriosLargosSinBloquearDirector(t *testing.T) {
	record := appChangeRecordForSourceTestV0()
	record.Request.AllowedWriteSet = nil
	record.Request.AcceptanceCriteria = []string{
		strings.Repeat("criterio editorial OPES detallado ", 20),
		"devolver DomainDocumentPlanV0 valido",
	}
	record.Request.ExternalWork = &orquestaappchange.AppChangeExternalWorkV0{
		ProjectRef:    "opes",
		JobRef:        "34f317ccf529318fd2943466c246756f",
		InterfaceRefs: []string{"opes-rest-v0", "opes-mcp-v0"},
		WorkKind:      "plan_temario",
		WorkRefs:      []string{"opes-program-001"},
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
	task := microtaskDecisionForTestV0(t, decisions).CreateMicrotask.Task
	if !containsFragmentInSetForTestV0(task.AcceptanceCriteria, "detalle completo en paquete externo") {
		t.Fatalf("criteria sin marcador de compactacion: %+v", task.AcceptanceCriteria)
	}
	if issues := orquestadirectoragent.ValidateDirectorAgentDecisionV0(microtaskDecisionForTestV0(t, decisions)); len(issues) != 0 {
		t.Fatalf("decision invalida: %+v", issues)
	}
}

func TestAppChangeDirectorDecisionSourceV0PlanTemarioOPESCompactaCriteriosYComando(t *testing.T) {
	record := appChangeRecordForSourceTestV0()
	record.Request.AllowedWriteSet = []string{"external/opes/plan_temario/job-ref-operario"}
	record.Request.AcceptanceCriteria = []string{
		"devolver artifact_type=document_plan",
		"payload_json valido y trazable",
		"sin placeholders",
		"sin leer internals de OPES",
		"entrega en fichero unico bajo allowed_write_set",
		"devolver DomainDocumentPlanV0 valido",
		"incluir sections y deliverables obligatorios",
		"incluir quality_criteria y review_steps aplicables",
		"para OPES, incluir research_exam_precedents para buscar examenes, convocatorias y temarios de administraciones relacionadas por internet usando fuentes publicas verificables",
		"para OPES, incluir generate_visual_asset para crear infografias utiles y no excesivas en puntos importantes, dificiles, comparativos o procedimentales donde aporten aprendizaje, incluidos apartados criticos cuando proceda",
		"para OPES, incluir generate_question_bank y deliverable question_bank para tests por tema",
		"para OPES, incluir generate_audio_asset y deliverable audio_asset para audio accesible por tema y por apartado/seccion",
		"para OPES, incluir generate_tutor_assets para tutor y bots del temario",
		"para OPES, incluir generate_html_site y deliverable local_html_site para HTML local operativo con logos USO y aspecto USO/TCAE promocion interna antes de subir a produccion",
		"no redactar el documento final dentro del plan",
		"para OPES, respetar flujo editorial: inventario, investigacion externa, agrupacion, mapa de dependencias, temas maestros, derivacion por nivel, redaccion, infografias, tests, revision, ensamblado, audio, tutor/bots y HTML local publicable",
		"si existe maestro A1/A2 o A1 equivalente, planificar primero ese maestro y despues derivar B/C1/C2/AP por resumen, reduccion editorial y adaptacion de nivel",
		"si no existe equivalente superior, marcar creacion_directa_nivel en criterios, constraints o secciones",
		"aplicar metodo OPES de asimilacion: recuperacion activa, repaso espaciado, ejemplos trabajados, carga cognitiva controlada, visuales utiles, elaboracion e intercalado",
	}
	record.Request.ExternalWork = &orquestaappchange.AppChangeExternalWorkV0{
		ProjectRef:    "opes",
		JobRef:        "job-ref-operario",
		InterfaceRefs: []string{"opes-rest-v0", "opes-mcp-v0"},
		WorkKind:      "plan_temario",
		WorkRefs:      []string{"opes-job-operario", "opes-program-operario"},
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
	task := microtaskDecisionForTestV0(t, decisions).CreateMicrotask.Task
	if len(task.AcceptanceCriteria) != maxAppChangeTaskCriteriaV0 {
		t.Fatalf("criteria=%+v", task.AcceptanceCriteria)
	}
	if issues := orquestadirectoragent.ValidateDirectorAgentDecisionV0(microtaskDecisionForTestV0(t, decisions)); len(issues) != 0 {
		t.Fatalf("decision invalida: %+v", issues)
	}
	command, issues := orquestadirectoragentworkflow.BuildDirectorAgentWorkflowCommandV0(
		orquestadirectoragentworkflow.DirectorAgentWorkflowCommandRequestV0{
			Decision:   microtaskDecisionForTestV0(t, decisions),
			OccurredAt: "2026-06-02T12:00:00Z",
		},
	)
	if len(issues) != 0 {
		t.Fatalf("workflow issues=%+v", issues)
	}
	if len(command.Payload) == 0 {
		t.Fatalf("payload vacio")
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
	task := microtaskDecisionForTestV0(t, decisions).CreateMicrotask.Task
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
	task := microtaskDecisionForTestV0(t, decisions).CreateMicrotask.Task
	if task.Title != "Resolver trabajo externo de app" ||
		task.Summary != "Resolver trabajo documental con paquete de dominio suficiente: temario, esquema, objetivo, fuentes, criterios y longitud si llegan." ||
		!stringInSetV0(task.WriteSet, "external/external-editorial/validate_topic") ||
		!stringInSetV0(task.AcceptanceCriteria, "Usar temario, esquema, objetivo, fuentes, criterios y longitud si llegan en input_fields.") ||
		!stringInSetV0(task.RequiredTests, "validar paquete documental de dominio") ||
		!stringInSetV0(task.RequiredTests, "validar fuentes y criterios documentales") {
		t.Fatalf("task=%+v", task)
	}
}

func TestAppChangeDirectorDecisionSourceV0CaracterizaResearchExamPrecedents(t *testing.T) {
	record := appChangeRecordForSourceTestV0()
	record.Request.AllowedWriteSet = nil
	record.Request.ExternalWork = &orquestaappchange.AppChangeExternalWorkV0{
		ProjectRef:    "opes",
		InterfaceRefs: []string{"opes-rest-v0", "opes-mcp-v0"},
		WorkKind:      "research_exam_precedents",
		WorkRefs:      []string{"opes-job-research-001", "opes-program-operario"},
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
	task := microtaskDecisionForTestV0(t, decisions).CreateMicrotask.Task
	if task.Title != "Investigar precedentes de examen externos" ||
		task.Summary != "Resolver trabajo documental con paquete de dominio suficiente: temario, esquema, objetivo, fuentes, criterios y longitud si llegan." ||
		!stringInSetV0(task.AcceptanceCriteria, "Usar temario, esquema, objetivo, fuentes, criterios y longitud si llegan en input_fields.") ||
		!stringInSetV0(task.RequiredTests, "validar paquete documental de dominio") {
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

func TestAppChangeDirectorDecisionSourceV0RecuperaCadenaRespondidaSinMicrotarea(t *testing.T) {
	record := appChangeRecordForSourceTestV0()
	store := orquestaappchange.NewInMemoryAppChangeStoreV0(record)
	run := appChangeRunForSourceTestV0(record)
	refs := appChangeRefsV0(record.Request.ChangeRef)
	run.CurrentPhase = orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0
	run.DirectorAnsweredQuestions = []string{refs.QuestionRef}
	run.Decisions = []string{refs.DecisionRef}

	decisions, err := (AppChangeDirectorDecisionSourceV0{Store: store}).
		ListDirectorAgentDecisionsV0(
			context.Background(),
			orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{Run: run},
		)

	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	if !containsCommandTypeForTestV0(decisions, orquestadirectoragent.DirectorAgentCommandCreateMicrotaskV0) {
		t.Fatalf("no recupera create_microtask: %+v", decisions)
	}
	if containsCommandTypeForTestV0(decisions, orquestadirectoragent.DirectorAgentCommandRequestVoteV0) ||
		containsCommandTypeForTestV0(decisions, orquestadirectoragent.DirectorAgentCommandAcceptDecisionV0) ||
		containsOpenPhaseForTestV0(decisions, orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0) {
		t.Fatalf("recuperacion no debe reabrir fases previas: %+v", decisions)
	}
}

func TestAppChangeDirectorDecisionSourceV0RecuperaCadenaRespondidaSinAbrirProgramacion(t *testing.T) {
	record := appChangeRecordForSourceTestV0()
	store := orquestaappchange.NewInMemoryAppChangeStoreV0(record)
	run := appChangeRunForSourceTestV0(record)
	refs := appChangeRefsV0(record.Request.ChangeRef)
	run.CurrentPhase = orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0
	run.DirectorAnsweredQuestions = []string{refs.QuestionRef}
	run.Decisions = []string{refs.DecisionRef}
	run.Tasks = []string{refs.TaskRef}

	decisions, err := (AppChangeDirectorDecisionSourceV0{Store: store}).
		ListDirectorAgentDecisionsV0(
			context.Background(),
			orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{Run: run},
		)

	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	if !containsOpenPhaseForTestV0(decisions, orquestacoreworkflow.OrchestrationPhaseProgramacionV0) {
		t.Fatalf("no recupera open_program: %+v", decisions)
	}
	if containsCommandTypeForTestV0(decisions, orquestadirectoragent.DirectorAgentCommandCreateMicrotaskV0) {
		t.Fatalf("no debe recrear microtarea existente: %+v", decisions)
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

func containsCommandTypeForTestV0(
	decisions []orquestadirectoragent.DirectorAgentDecisionV0,
	commandType string,
) bool {
	for _, decision := range decisions {
		if decision.CommandType == commandType {
			return true
		}
	}
	return false
}

func microtaskDecisionForTestV0(
	t *testing.T,
	decisions []orquestadirectoragent.DirectorAgentDecisionV0,
) orquestadirectoragent.DirectorAgentDecisionV0 {
	t.Helper()
	for _, decision := range decisions {
		if decision.CommandType == orquestadirectoragent.DirectorAgentCommandCreateMicrotaskV0 {
			if decision.CreateMicrotask == nil {
				t.Fatalf("create_microtask sin payload: %+v", decision)
			}
			return decision
		}
	}
	t.Fatalf("create_microtask no encontrada en decisiones: %+v", decisions)
	return orquestadirectoragent.DirectorAgentDecisionV0{}
}

func containsOpenPhaseForTestV0(
	decisions []orquestadirectoragent.DirectorAgentDecisionV0,
	phase orquestacoreworkflow.OrchestrationPhaseIDV0,
) bool {
	for _, decision := range decisions {
		if decision.OpenPhase != nil && decision.OpenPhase.PhaseID == string(phase) {
			return true
		}
	}
	return false
}

func appChangeOPESSubroleMicrotasksForTestV0(
	decisions []orquestadirectoragent.DirectorAgentDecisionV0,
) (orquestadirectoragent.DirectorAgentMicrotaskV0, []orquestadirectoragent.DirectorAgentMicrotaskV0) {
	children := make([]orquestadirectoragent.DirectorAgentMicrotaskV0, 0, 6)
	var parent orquestadirectoragent.DirectorAgentMicrotaskV0
	for _, decision := range decisions {
		if decision.CreateMicrotask == nil {
			continue
		}
		task := decision.CreateMicrotask.Task
		if strings.TrimSpace(task.ParentTaskRef) != "" {
			children = append(children, task)
			continue
		}
		if len(task.ChildTaskRefs) > 0 {
			parent = task
		}
	}
	return parent, children
}

func containsFragmentInSetForTestV0(values []string, fragment string) bool {
	for _, value := range values {
		if strings.Contains(value, fragment) {
			return true
		}
	}
	return false
}

func requireStringSetForTestV0(t *testing.T, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("set len got=%d want=%d got=%v want=%v", len(got), len(want), got, want)
	}
	for _, value := range want {
		if !stringInSetV0(got, value) {
			t.Fatalf("set missing %q got=%v want=%v", value, got, want)
		}
	}
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
		ParentTaskRef:        task.ParentTaskRef,
		CohortRef:            task.CohortRef,
		WaveRef:              task.WaveRef,
		DelegationDepth:      task.DelegationDepth,
		MaxChildAgents:       task.MaxChildAgents,
		ChildTaskRefs:        task.ChildTaskRefs,
		FunctionContractRefs: refs,
	}
}

func microtaskDecisionsForTestV0(
	decisions []orquestadirectoragent.DirectorAgentDecisionV0,
) []orquestadirectoragent.DirectorAgentDecisionV0 {
	out := make([]orquestadirectoragent.DirectorAgentDecisionV0, 0, len(decisions))
	for _, decision := range decisions {
		if decision.CreateMicrotask != nil {
			out = append(out, decision)
		}
	}
	return out
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
