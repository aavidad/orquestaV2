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
		"Cadena completa del workflow, no del primer fichero",
		"brainstorming_arquitectura -> votacion_y_decision -> planificacion_microtareas -> programacion -> revision -> validacion_final -> cierre",
		"Primera entrega accionable del director",
		"open_phase programacion",
		"En la primera entrega no emitas open_phase revision",
		"request_review",
		"record_review_result",
		"accept_review",
		"open_phase validacion_final",
		"register_final_validation",
		"open_phase cierre",
		"close_run",
		"solo en una continuacion del director cuando Orquesta entregue estadisticas/evidencias reales",
		"no inventes refs futuras",
		"phase_id de nivel superior",
		"evidence_refs son identificadores",
		"no uses espacios, slash, rutas ni etiquetas humanas",
		"evidence-ref-docs-arquitectura-v0",
		"No omitas phase_id de nivel superior",
		"Mapeo phase_id superior",
		"close_run usa cierre",
		"minimum_recommended_capacity solo admite low, medium, high o xhigh",
		"usa high para la votacion inicial",
		"Minimos para emitir register_final_validation/close_run",
		"no emitas register_final_validation hasta que el run tenga evidencia compacta de todos los minimos del request_kind",
		"no emitas close_run hasta que register_final_validation haya sido aceptado",
		"Para crear_app_completa normal, antes de register_final_validation deben existir contratos, tareas de programacion, entregas de programacion, tarea cerrada y revision aceptada",
		"antes de close_run tambien validacion_final registrada",
		"decision_ref",
		"command_type",
		"No uses action, decision_id, payload ni refs",
		"create_microtask.task",
		"open_phase.phase_id y open_phase.reason",
		"request_vote requiere vote_request_id",
		"accept_decision.vote_ref debe ser exactamente igual",
		"publish_function_contract.decision_ref debe ser exactamente igual",
		"publish_function_contract requiere contract_ref",
		"function_contract_refs usa objetos",
		"function_contract_refs.contract_ref debe ser exactamente igual",
		"microtareas de programacion deben cubrir go.mod, cmd/server o entrypoint equivalente",
		"required_tests con go test ./...",
		"acceptance_criteria debe exigir imports de modulo desde go.mod",
		"prohibir imports relativos ../",
		"request_review requiere review_request_id",
		"record_review_result requiere review_result_ref",
		"accept_review.review_request_id debe ser exactamente igual",
		"register_final_validation requiere validation_ref, phase_id validacion_final",
		"close_run requiere closure_ref, phase_id cierre",
		"No escribas terminos prohibidos",
		"adapter, adaptador",
		"para seguridad usa datos sensibles",
		"usa task_id; no uses task_ref",
	} {
		if !strings.Contains(task.Objective, want) {
			t.Fatalf("objective no contiene %q:\n%s", want, task.Objective)
		}
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
	if !strings.Contains(task.Objective, "En la primera entrega no emitas open_phase revision") {
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
