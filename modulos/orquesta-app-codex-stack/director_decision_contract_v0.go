package orquestaappcodexstack

import (
	"strings"

	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func directorDecisionInstructionsV0(
	payload orquestaruntime.LaunchRuntimeAgentRequestV0,
) []string {
	runRef := strings.TrimSpace(payload.RunID)
	brainstormRef := directorBrainstormRefFromTaskRefV0(payload.TaskRef)
	policy := orquestafactory.ResolveRequestPolicyV0(
		directorRequestKindV0(payload),
		directorExecutionModeV0(payload),
	)
	minimums := strings.Join(policy.MinimumDeliverables, ", ")
	return []string{
		"Contrato ejecutable del director principal:",
		"RunID para decisiones: " + runRef + ".",
		"BrainstormRef inicial: " + brainstormRef + ".",
		"request_kind actual para cierre: " + policy.RequestKind + ".",
		"execution_mode actual para cierre: " + policy.ExecutionMode + ".",
		"minimum_deliverables exactos para este request_kind: " + minimums + ".",
		"Escribe director_decisions.json en decision_path con schema director_agent_decisions_file.v0 y decisions[] de schema director_agent_decision.v0.",
		"Workflow completo: brainstorming_arquitectura -> votacion_y_decision -> planificacion_microtareas -> programacion -> revision -> validacion_final -> cierre.",
		"Primera entrega accionable del director: open_phase(votacion_y_decision), request_vote con BrainstormRef inicial, accept_decision, open_phase(planificacion_microtareas), publish_function_contract, create_microtask(s) con decision.phase_id=planificacion_microtareas y task.phase_id=programacion, y despues open_phase(programacion).",
		"En la primera entrega no emitas revision, validacion_final, cierre, request_review, record_review_result, accept_review, register_final_validation ni close_run; esas fases solo en continuacion con estadisticas/evidencias reales.",
		"Cada decision incluye schema_version=director_agent_decision.v0, run_id exacto, phase_id de nivel superior, decision_ref unico, command_type, command_ref unico, summary y evidence_refs compactas.",
		"Fases: en open_phase, decision.phase_id es la fase actual y open_phase.phase_id la fase a abrir; en create_microtask, decision.phase_id es planificacion_microtareas y create_microtask.task.phase_id es la fase objetivo programacion; en los demas comandos con phase_id, decision.phase_id debe ser igual al phase_id del payload.",
		"evidence_refs son identificadores [A-Za-z0-9._:-], sin espacios, slash, rutas ni texto humano; ejemplo evidence-ref-docs-arquitectura-v0.",
		"minimum_recommended_capacity solo admite low, medium, high o xhigh; usa high para la votacion inicial.",
		"Payloads iniciales: open_phase.phase_id/reason; request_vote.vote_request_id/phase_id/decision_topic_ref/brainstorm_ref/summary/minimum_recommended_capacity/evidence_refs; accept_decision.decision_ref/phase_id/vote_ref/accepted_option_ref/summary/evidence_refs; publish_function_contract.contract_ref/phase_id/decision_ref/summary/function_names/evidence_refs; create_microtask.task workflow_task.v0.",
		"Referencias causales exactas: accept_decision.vote_ref debe ser identico al request_vote.vote_request_id anterior; publish_function_contract.decision_ref debe ser identico al accept_decision.decision_ref anterior; cada create_microtask.task.function_contract_refs[].contract_ref debe ser identico al publish_function_contract.contract_ref anterior.",
		"workflow_task.v0 incluye schema_version=workflow_task.v0 y usa task_id, run_id, phase_id, title, summary, write_set, acceptance_criteria, function_contract_refs[{contract_ref,function_name}] y required_tests si phase_id=programacion; no uses task_ref.",
		"Cada create_microtask de programacion debe anclarse a la peticion actual: acceptance_criteria incluye una linea `objetivo_actual: ...` con la capacidad concreta solicitada y sus tokens principales; las microtareas genericas o historicas sin ese anclaje son invalidas.",
		"Para crear_app_completa Go normal, crea microtarea bootstrap con go.mod y entrypoint; las posteriores que importen modulo o ejecuten go test ./... incluyen depends_on con el task_id del bootstrap, usan write_set pequeno, imports desde go.mod y sin imports relativos ../.",
		"Revision/cierre payloads: request_review.review_request_id/phase_id/delivery_ref; record_review_result.review_result_ref/review_request_id/delivery_ref/status; accept_review.accepted_review_ref/review_request_id/delivery_ref; register_final_validation.validation_ref/closed_task_ref/request_kind/execution_mode/minimum_deliverables; close_run.closure_ref/validation_ref/request_kind/execution_mode/minimum_deliverables.",
		"Cierre normal: conserva minimum_deliverables exactos (" + minimums + ") y no emitas register_final_validation/close_run hasta tener contratos, microtareas, entregas de programacion, tarea cerrada, revision aceptada y validacion; si falta algo, crea trabajo/revision o CONSULTA AL DIRECTOR.",
		"En debug puedes cerrar alcance reducido solo si summary de register_final_validation y close_run lista omisiones_debug con cada minimum_deliverable ausente; evidence_refs debe incluir una ref compacta de omisiones. Debug no convierte cierre reducido en cierre productivo.",
		"No uses action, decision_id, payload ni refs; create_microtask contiene create_microtask.task.",
		"No escribas terminos prohibidos en campos de decision: secret, secreto, token, credential, api_key, provider, model, db, sql, base de datos, runtime, adapter, adaptador, filesystem, git, docker, tmux, home u oauth; para seguridad usa datos sensibles.",
	}
}

func directorBrainstormRefFromTaskRefV0(taskRef string) string {
	taskRef = strings.TrimSpace(taskRef)
	if strings.HasPrefix(taskRef, "task-") {
		return "brainstorm-" + strings.TrimPrefix(taskRef, "task-")
	}
	if taskRef == "" {
		return "brainstorm-director"
	}
	return "brainstorm-" + taskRef
}
