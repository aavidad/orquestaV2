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
		"Escribe director_decisions.json en el decision_path indicado por el prompt.",
		"Usa schema director_agent_decisions_file.v0 con campo decisions.",
		"Cadena completa del workflow, no del primer fichero: brainstorming_arquitectura -> votacion_y_decision -> planificacion_microtareas -> programacion -> revision -> validacion_final -> cierre.",
		"Primero preserva el brainstorming inicial: usa BrainstormRef inicial como evidencia de brainstorming_arquitectura antes de abrir votacion_y_decision.",
		"Primera entrega accionable del director: emite solo open_phase votacion_y_decision; request_vote con BrainstormRef inicial; accept_decision; open_phase planificacion_microtareas; publish_function_contract; create_microtask(s) de phase_id programacion; open_phase programacion.",
		"En la primera entrega no emitas open_phase revision, request_review, record_review_result, accept_review, open_phase validacion_final, register_final_validation, open_phase cierre ni close_run.",
		"Las fases revision, validacion_final y cierre se emiten solo en una continuacion del director cuando Orquesta entregue estadisticas/evidencias reales de entregas, revisiones y validacion; no inventes refs futuras.",
		"Cada decision usa schema director_agent_decision.v0, run_id exacto, phase_id de nivel superior, decision_ref, command_type, command_ref, summary y evidence_refs compactas.",
		"decision_ref superior y command_ref son identificadores del comando actual y deben ser unicos por decision dentro del fichero.",
		"evidence_refs son identificadores, no texto: usa solo letras, numeros, punto, guion, guion bajo o dos puntos; no uses espacios, slash, rutas ni etiquetas humanas.",
		"Ejemplos validos de evidence_refs: evidence-ref-docs-arquitectura-v0, evidence.ref.plan.v0, evidence:vote:001.",
		"No uses refs como docs/arquitectura.md, README.Agenda API Web ni docs/archivo.md:Decision humana.",
		"No omitas phase_id de nivel superior; es la fase actual antes de aplicar el comando.",
		"Mapeo phase_id superior: open_phase a votacion_y_decision usa brainstorming_arquitectura; request_vote, accept_decision y open_phase a planificacion_microtareas usan votacion_y_decision; publish_function_contract, create_microtask y open_phase a programacion usan planificacion_microtareas; open_phase a revision usa programacion; request_review, record_review_result, accept_review y open_phase a validacion_final usan revision; register_final_validation y open_phase a cierre usan validacion_final; close_run usa cierre.",
		"minimum_recommended_capacity solo admite low, medium, high o xhigh; usa high para la votacion inicial, nunca baja, media ni alta.",
		"Payload tipado por command_type: open_phase, request_vote, accept_decision, publish_function_contract o create_microtask.",
		"Payload tipado de revision y cierre: request_review, record_review_result, accept_review, register_final_validation y close_run.",
		"Minimos para emitir register_final_validation/close_run: conserva minimum_deliverables exactos: " + minimums + ".",
		"En execution_mode normal no emitas register_final_validation hasta que el run tenga evidencia compacta de todos los minimos del request_kind; si falta uno, emite trabajo/revision o pregunta, no cierre.",
		"En execution_mode normal no emitas close_run hasta que register_final_validation haya sido aceptado y exista validation_ref registrado.",
		"Para crear_app_completa normal, antes de register_final_validation deben existir contratos, tareas de programacion, entregas de programacion, tarea cerrada y revision aceptada; antes de close_run tambien validacion_final registrada.",
		"Para request_kind de programacion, cambio o refactor normal exige entregas de programacion y tarea cerrada; para documentacion, plan, brainstorming o deploy normal exige artefactos o entregas; para otros request_kind exige resultado/evidencia compacta.",
		"En execution_mode debug puedes cerrar alcance reducido solo si summary de register_final_validation y close_run lista omisiones_debug con cada minimum_deliverable ausente; evidence_refs debe incluir una ref compacta de omisiones.",
		"Debug no convierte cierre reducido en cierre productivo; no declares que todos los minimos se cumplieron si faltan.",
		"register_final_validation y close_run deben declarar request_kind, execution_mode y minimum_deliverables usados para cerrar.",
		"En execution_mode normal, crear_app_completa no puede cerrarse solo con documentacion: requiere contratos, microtareas, entregas de programacion, revision aceptada y validacion.",
		"No uses action, decision_id, payload ni refs; create_microtask contiene create_microtask.task.",
		"open_phase requiere open_phase.phase_id y open_phase.reason; no uses title ni summary dentro de open_phase.",
		"request_vote requiere vote_request_id, phase_id, decision_topic_ref, brainstorm_ref, summary, minimum_recommended_capacity y evidence_refs.",
		"accept_decision requiere decision_ref, phase_id, vote_ref, accepted_option_ref, summary y evidence_refs.",
		"accept_decision.vote_ref debe ser exactamente igual al request_vote.vote_request_id previo.",
		"publish_function_contract.decision_ref debe ser exactamente igual al accept_decision.decision_ref previo, pero el decision_ref superior de publish_function_contract debe seguir siendo unico.",
		"publish_function_contract requiere contract_ref, phase_id, decision_ref, summary, function_names y evidence_refs; no uses inputs, outputs, ports ni constraints.",
		"create_microtask.task requiere schema_version, task_id, run_id, phase_id, title, summary, write_set, acceptance_criteria y function_contract_refs; si phase_id es programacion tambien requiere required_tests.",
		"create_microtask.task puede usar depends_on con task_id previos para ordenar dependencias reales; no inventes dependencias externas.",
		"function_contract_refs usa objetos con contract_ref y function_name; no uses strings sueltos.",
		"Cada function_contract_refs.contract_ref debe ser exactamente igual al publish_function_contract.contract_ref previo.",
		"Para crear_app_completa Go normal, las microtareas de programacion deben cubrir go.mod, cmd/server o entrypoint equivalente, internal/* hexagonal, web si aplica, README/docs y required_tests con go test ./....",
		"Para crear_app_completa Go normal, crea primero una microtarea bootstrap que escriba go.mod y entrypoint; toda microtarea posterior que ejecute go test ./... o importe modulo debe declarar depends_on con el task_id del bootstrap.",
		"Para crear_app_completa Go normal, acceptance_criteria debe exigir imports de modulo desde go.mod, prohibir imports relativos ../ y dejar la app compilable con go test ./....",
		"request_review requiere review_request_id, phase_id revision, delivery_ref, summary y evidence_refs.",
		"record_review_result requiere review_result_ref, review_request_id, delivery_ref, status, summary y evidence_refs; usa status accepted solo si no quedan bloqueos.",
		"accept_review requiere accepted_review_ref, phase_id revision, review_request_id, delivery_ref, summary y evidence_refs.",
		"accept_review.review_request_id debe ser exactamente igual al request_review.review_request_id previo y delivery_ref debe coincidir.",
		"register_final_validation requiere validation_ref, phase_id validacion_final, closed_task_ref, summary, request_kind, execution_mode, minimum_deliverables y evidence_refs.",
		"close_run requiere closure_ref, phase_id cierre, validation_ref, summary, request_kind, execution_mode, minimum_deliverables y evidence_refs.",
		"No escribas terminos prohibidos en campos de decision: secret, secreto, token, credential, api_key, provider, model, db, sql, base de datos, runtime, adapter, adaptador, filesystem, git, docker, tmux, home u oauth; para seguridad usa datos sensibles.",
		"En workflow_task.v0 usa task_id; no uses task_ref.",
		"Cada microtarea usa schema workflow_task.v0, run_id exacto, write_set pequeno y acceptance_criteria verificables.",
		"No incluyas proveedores, modelos, rutas, secretos ni motores de datos concretos en campos de decision.",
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
