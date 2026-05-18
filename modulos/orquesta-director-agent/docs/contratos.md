# Contratos locales

## DirectorAgentDecisionV0

DTO producido por un agente director externo y consumido por un adaptador de Orquesta.

Campos obligatorios:

- `schema_version`: `director_agent_decision.v0`;
- `decision_ref`, `run_id`, `phase_id`, `command_type`, `command_ref`;
- `summary` compacto;
- `evidence_refs` compactas;
- payload especifico del comando.

Comandos soportados en v0:

- `request_brainstorm`: propone iniciar brainstorming de arquitectura.
- `open_phase`: propone abrir una fase publica del workflow.
- `request_vote`: propone solicitar votacion para una decision arquitectonica.
- `accept_decision`: propone aceptar una opcion votada.
- `publish_function_contract`: propone publicar un contrato funcional compacto ya respaldado por una decision aceptada.
- `create_microtask`: propone crear una microtarea pequena enlazada a contratos funcionales publicados.
- `ask_director`, `ask_user`: propone una consulta compacta cuando falta informacion para avanzar; no resuelve la duda dentro del DTO.
- `request_capacity`, `request_agent`: propone capacidad y agente especializado ya acotados por refs.
- `request_review`, `record_review_result`, `accept_review`: proponen el ciclo compacto de revision de entrega.
- `request_rework`, `record_replan_decision`: propone retrabajo y registra una decision de replan por refs compactas.
- `close_task`: propone cerrar una microtarea revisada en fase `revision`.
- `register_final_validation`: propone registrar validacion final con politica de solicitud declarada.
- `close_run`: propone cerrar el run con politica de solicitud declarada.
- `propose_autonomous_plan_team`: propone un plan/equipo autonomo compacto como DTO, sin materializar runtime, DB ni proveedor.

Invariantes:

- sin proveedor, modelo, HOME, OAuth, tokens, secrets ni transcripts;
- sin payloads largos;
- refs opacas y compactas;
- el agente externo no aplica el comando: solo propone decision.
- el director no hace el trabajo pesado: solo decide el siguiente movimiento y delega en agentes especializados con contexto pequeno.
- el director es neutral de dominio: puede planificar programacion,
  documentacion, OPES, seguridad, deploy, refactor o migracion si el contrato
  de entrada aporta reglas y refs suficientes.
- las apps externas no deben reemplazar al director con logica de juicio; deben
  pedirle planificacion a Orquesta y conservar solo reglas/validacion/ensamblado
  de dominio.
- `create_microtask` no puede incluir proveedor, modelo, HOME, OAuth, DB, runtime ni secretos en `write_set`, criterios o refs.
- en `create_microtask`, `phase_id` de la decision es la fase actual que autoriza crear trabajo: `planificacion_microtareas` para el plan inicial o `programacion` para cambios en caliente; `task.phase_id` es la fase objetivo de ejecucion, normalmente `programacion`.
- `propose_autonomous_plan_team` vive en `planificacion_microtareas`, cita contratos funcionales por ref y solo describe miembros, capacidades y unidades de trabajo compactas.
- `close_task` debe citar `task_id`, `phase_id=revision`, `delivery_ref`, `accepted_review_ref`, `summary` y `evidence_refs`; queda entre `accept_review` y `open_phase` hacia `validacion_final`.
- `register_final_validation` y `close_run` deben incluir `request_kind`, `execution_mode` y `minimum_deliverables` para que la capa de aplicacion pueda impedir cierres parciales fuera de `debug`.
- `record_review_result.status` solo acepta `accepted`, `changes_requested` o `rejected`.
- `ask_user` se expresa con `target_group=user`; `ask_director` con `target_group=director` o vacio normalizable.

## DirectorAgentCompactStatsV0

Entrada compacta opcional para que un director externo observe estado sin cargar
run completo.

Campos:

- `schema_version`: `director_agent_compact_stats.v0`;
- `run_id`, `status`, `current_phase`;
- contadores agregados de fases, tareas, capacidad, agentes, entregas, rework,
  replan y cierre;
- `pending_refs` y `evidence_refs` compactas.

Invariantes:

- no contiene payloads, transcripts, rutas, proveedor, runtime, DB, HOME ni secretos;
- los contadores no pueden ser negativos;
- las refs pendientes son marcadores compactos, no datos operativos.
