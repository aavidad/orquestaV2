package orquestaappcodexstack

import (
	"strings"
	"testing"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestDirectorTaskV0IncluyeContratoDeDecisionesEjecutables(t *testing.T) {
	payload := orquestaruntime.LaunchRuntimeAgentRequestV0{
		RunID:   "run-ref-agenda-001",
		TaskRef: "task-agenda-api-web-director",
		Summary: "Agenda con API REST y web. request_kind=crear_app_completa execution_mode=normal.",
	}

	task := directorTaskV0(payload, "director")

	for _, want := range []string{
		"Tipo de peticion: crear_app_completa.",
		"Modo de ejecucion: normal.",
		"Minimos crear_app_completa",
		"request_kind actual para cierre: crear_app_completa.",
		"execution_mode actual para cierre: normal.",
		"minimum_deliverables exactos para este request_kind: arquitectura, contratos, programacion, pruebas, seguridad, documentacion, deploy_si_aplica, revision_final.",
		"director_decisions.json",
		"director_agent_decisions_file.v0",
		"director_agent_decision.v0",
		"workflow_task.v0",
		"RunID para decisiones: run-ref-agenda-001.",
		"BrainstormRef inicial: brainstorm-agenda-api-web-director.",
		"Workflow completo",
		"brainstorming_arquitectura -> votacion_y_decision -> planificacion_microtareas -> programacion -> revision -> validacion_final -> cierre",
		"Primera entrega accionable del director",
		"open_phase(programacion)",
		"create_microtask.task con una o mas tareas completas",
		"decision.phase_id=planificacion_microtareas y task.phase_id=programacion",
		"En la primera entrega no emitas revision",
		"request_review",
		"record_review_result",
		"accept_review",
		"validacion_final",
		"register_final_validation",
		"cierre",
		"close_run",
		"solo en continuacion con estadisticas/evidencias reales",
		"phase_id de nivel superior",
		"evidence_refs son identificadores [A-Za-z0-9._:-]",
		"sin espacios, slash, rutas ni texto humano",
		"evidence-ref-docs-arquitectura-v0",
		"Fases: en open_phase",
		"en create_microtask, decision.phase_id es planificacion_microtareas",
		"create_microtask.task.phase_id es la fase objetivo programacion",
		"en los demas comandos con phase_id, decision.phase_id debe ser igual al phase_id del payload",
		"minimum_recommended_capacity solo admite low, medium, high o xhigh",
		"usa high para la votacion inicial",
		"Payloads iniciales",
		"decision_ref",
		"command_type",
		"command_ref",
		"No uses action, decision_id, payload ni refs",
		"create_microtask.task",
		"open_phase.phase_id/reason",
		"request_vote.vote_request_id",
		"publish_function_contract.contract_ref",
		"function_contract_refs[{contract_ref,function_name}]",
		"El comando legacy create_microtask conserva nombre v0",
		"debe transportar tareas completas por entregable o area",
		"no lo uses para trozos minimos",
		"Para documentar_app normal, por defecto crea una sola tarea completa de documentacion",
		"Para crear_app_completa Go normal, prefiere tareas completas verticales",
		"una tarea puede cubrir go.mod, entrypoint, paquete interno, README.md y pruebas",
		"depends_on con el task_id inicial",
		"go test ./...",
		"imports desde go.mod",
		"sin imports relativos ../",
		"Revision/cierre payloads",
		"request_review.review_request_id",
		"record_review_result.review_result_ref",
		"accept_review.accepted_review_ref",
		"register_final_validation.validation_ref",
		"close_run.closure_ref",
		"Cierre normal",
		"no emitas register_final_validation/close_run hasta tener contratos, tareas completas, entregas de programacion, tarea cerrada, revision aceptada y validacion",
		"No escribas terminos prohibidos",
		"adapter, adaptador",
		"para seguridad usa datos sensibles",
		"no uses task_ref",
	} {
		if !strings.Contains(task.Objective, want) {
			t.Fatalf("objective no contiene %q:\n%s", want, task.Objective)
		}
	}
	if len(task.Objective) > 7000 {
		t.Fatalf("objective demasiado grande: %d bytes", len(task.Objective))
	}
	if !codexStackStringInSetForTestV0(
		task.DoneCriteria,
		"director_decisions.json escrito en decision_path con decisiones ejecutables.",
	) {
		t.Fatalf("done_criteria=%v", task.DoneCriteria)
	}
	if !codexStackDoneCriteriaContainsForTestV0(task.DoneCriteria, "arquitectura, contratos, programacion") {
		t.Fatalf("done_criteria no recoge minimos por request_kind: %v", task.DoneCriteria)
	}
	for _, path := range []string{
		"docs/plan_tareas.md",
		"docs/manual_usuario.md",
		"docs/manual_desarrollador.md",
		"docs/manual_sistemas_deploy.md",
		"docs/decisiones.md",
		"docs/pruebas.md",
		"docs/pendientes.md",
	} {
		if !codexStackStringInSetForTestV0(task.WriteSet, path) {
			t.Fatalf("write_set no contiene %s: %+v", path, task.WriteSet)
		}
	}
}

func TestDirectorTaskV0NoOrdenaCierreEnPrimeraEntrega(t *testing.T) {
	payload := orquestaruntime.LaunchRuntimeAgentRequestV0{
		RunID:   "run-ref-agenda-001",
		TaskRef: "task-agenda-api-web-director",
		Summary: "Agenda con API REST y web. request_kind=crear_app_completa execution_mode=normal.",
	}

	task := directorTaskV0(payload, "director")

	for _, forbidden := range []string{
		"Despues emite open_phase votacion_y_decision; request_vote con BrainstormRef inicial; accept_decision; open_phase planificacion_microtareas; publish_function_contract; create_microtask(s) de phase_id programacion; open_phase programacion; espera entregas de programacion registradas; open_phase revision; request_review; record_review_result; accept_review; open_phase validacion_final; register_final_validation; open_phase cierre; close_run.",
		"Cadena completa esperada",
	} {
		if strings.Contains(task.Objective, forbidden) {
			t.Fatalf("objective conserva instruccion peligrosa %q:\n%s", forbidden, task.Objective)
		}
	}
	if !strings.Contains(task.Objective, "Primera entrega accionable del director") {
		t.Fatalf("objective no separa primera entrega:\n%s", task.Objective)
	}
	if !strings.Contains(task.Objective, "En la primera entrega no emitas revision") {
		t.Fatalf("objective no bloquea revision prematura:\n%s", task.Objective)
	}
}

func TestDirectorTaskV0NoExigeDecisionFileEnAreasEspecializadas(t *testing.T) {
	payload := orquestaruntime.LaunchRuntimeAgentRequestV0{
		RunID:   "run-ref-agenda-001",
		TaskRef: "task-agenda-api-web-web",
		Summary: "Agenda con API REST y web. request_kind=documentar_app execution_mode=debug.",
	}

	task := directorTaskV0(payload, "web")

	if strings.Contains(task.Objective, "director_decisions.json") {
		t.Fatalf("objective especializado no debe emitir contrato ejecutable: %s", task.Objective)
	}
	if codexStackStringInSetForTestV0(
		task.DoneCriteria,
		"director_decisions.json escrito en decision_path con decisiones ejecutables.",
	) {
		t.Fatalf("done_criteria especializado=%v", task.DoneCriteria)
	}
	if !codexStackDoneCriteriaContainsForTestV0(task.DoneCriteria, "entregables omitidos por debug") {
		t.Fatalf("done_criteria debug=%v", task.DoneCriteria)
	}
}

func TestDirectorTaskV0DocumentarNormalNoQuedaSoloEnPlan(t *testing.T) {
	payload := orquestaruntime.LaunchRuntimeAgentRequestV0{
		RunID:   "run-ref-docs-normal-001",
		TaskRef: "task-docs-normal-director",
		Summary: "Documentacion completa. request_kind=documentar_app execution_mode=normal.",
	}

	task := directorTaskV0(payload, "director")

	for _, want := range []string{
		"No eres un worker de area",
		"No te limites a planificar",
		"por defecto crea una sola tarea completa de documentacion",
		"Minimos documentar_app: manual_usuario, manual_desarrollador, manual_sistemas_deploy, decisiones, pruebas_documentales, pendientes.",
	} {
		if !strings.Contains(task.Objective, want) {
			t.Fatalf("objective no contiene %q:\n%s", want, task.Objective)
		}
	}
	for _, path := range []string{
		"docs/manual_usuario.md",
		"docs/manual_desarrollador.md",
		"docs/manual_sistemas_deploy.md",
		"docs/decisiones.md",
		"docs/pruebas.md",
		"docs/pendientes.md",
	} {
		if !codexStackStringInSetForTestV0(task.WriteSet, path) {
			t.Fatalf("write_set no contiene %s: %+v", path, task.WriteSet)
		}
	}
}

func TestDirectorTaskV0DebugDeclaraOmisionesDeCierre(t *testing.T) {
	payload := orquestaruntime.LaunchRuntimeAgentRequestV0{
		RunID:   "run-ref-docs-debug-001",
		TaskRef: "task-docs-debug-director",
		Summary: "Documentacion acotada. request_kind=documentar_app execution_mode=debug.",
	}

	task := directorTaskV0(payload, "director")

	for _, want := range []string{
		"Tipo de peticion: documentar_app.",
		"Modo de ejecucion: debug.",
		"request_kind actual para cierre: documentar_app.",
		"execution_mode actual para cierre: debug.",
		"minimum_deliverables exactos para este request_kind: manual_usuario, manual_desarrollador, manual_sistemas_deploy, decisiones, pruebas_documentales, pendientes.",
		"summary de register_final_validation y close_run lista omisiones_debug con cada minimum_deliverable ausente",
		"evidence_refs debe incluir una ref compacta de omisiones",
		"Debug no convierte cierre reducido en cierre productivo",
	} {
		if !strings.Contains(task.Objective, want) {
			t.Fatalf("objective no contiene %q:\n%s", want, task.Objective)
		}
	}
	if !codexStackDoneCriteriaContainsForTestV0(task.DoneCriteria, "entregables omitidos por debug") {
		t.Fatalf("done_criteria debug=%v", task.DoneCriteria)
	}
}

func codexStackStringInSetForTestV0(values []string, want string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}

func codexStackDoneCriteriaContainsForTestV0(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}
