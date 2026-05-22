# Tareas locales

## DAG-008 - Refs de contexto opacas en microtareas

Estado: hecho.

Write-set:

- `director_decision_types_v0.go`;
- `director_decision_helpers_v0.go`;
- `director_decision_validation_*.go`;
- tests y docs locales.

Cierre:

- `create_microtask.task.context_refs` transporta refs opacas compactas;
- no introduce campos de producto ni composicion;
- rechaza refs no compactas o con detalles prohibidos.

## DAG-001 - Contrato de decision del director externo

Estado: hecho.

Write-set:

- `director_decision_types_v0.go`;
- `director_decision_validation_v0.go`;
- `director_decision_helpers_v0.go`;
- `director_decision_v0_test.go`.

Cierre:

- valida decision compacta;
- rechaza detalles operativos;
- cubre comando `request_brainstorm`;
- no importa core, runtime, persistence ni proveedores.

## DAG-002 - Prueba real opt-in desde Orquesta

Estado: en `orquesta-e2e`.

Cierre:

- Orquesta abre fase y emite outbox `LaunchRuntimeAgent`;
- runtime `AgentCLI` lanza un agente externo real;
- el agente escribe `director_decision.json`;
- el adaptador valida el DTO y aplica el comando publico al workflow.

## DAG-003 - Decisiones de autoprogramacion del director

Estado: hecho.

Write-set:

- `director_decision_types_v0.go`;
- `director_decision_validation_v0.go`;
- `director_decision_helpers_v0.go`;
- `director_decision_v0_test.go`.

Cierre:

- valida `publish_function_contract`;
- valida `create_microtask`;
- valida `open_phase`, `request_vote` y `accept_decision`;
- exige run/fase coherente y refs de contrato explicitas;
- permite `write_set` compacto sin convertirlo en ref opaca;
- mantiene prohibidos proveedor, modelo, HOME, OAuth, DB, runtime y secretos.

## DAG-004 - DTO de plan/equipo autonomo

Estado: hecho.

Write-set:

- `director_decision_types_v0.go`;
- `director_autonomous_plan_helpers_v0.go`;
- `director_autonomous_plan_validation_v0.go`;
- `director_autonomous_plan_v0_test.go`.

Cierre:

- valida `propose_autonomous_plan_team`;
- exige `planificacion_microtareas`, equipo, unidades y contratos funcionales;
- rechaza detalles de proveedor, modelo, HOME, DB, runtime y secretos;
- no materializa comandos, stores ni agentes.

## DAG-005 - DTOs de validacion final y cierre

Estado: hecho.

Write-set:

- `director_decision_types_v0.go`;
- `director_decision_helpers_close_v0.go`;
- `director_decision_validation_close_v0.go`;
- `director_decision_close_v0_test.go`.

Cierre:

- valida `register_final_validation`;
- valida `close_run`;
- exige `request_kind`, `execution_mode` y `minimum_deliverables`;
- rechaza refs humanas o rutas en evidencias;
- mantiene el DTO como propuesta del director, sin decidir si el cierre esta
  permitido por politica de producto.

## DAG-006 - DTO de cierre de microtarea revisada

Estado: hecho.

Write-set:

- `director_decision_types_v0.go`;
- `director_decision_helpers_close_v0.go`;
- `director_decision_validation_close_v0.go`;
- `director_decision_review_v0_test.go`;
- docs locales.

Cierre:

- valida `close_task`;
- exige `task_id`, `phase_id=revision`, `delivery_ref`,
  `accepted_review_ref`, `summary` y `evidence_refs`;
- mantiene `close_task` despues de `accept_review` y antes de `open_phase`
  hacia `validacion_final`;
- no importa core, runtime, persistence ni proveedores.

## DAG-007 - Superficie minima para app completa y stats compactas

Estado: hecho.

Write-set:

- `director_decision_types_v0.go`;
- `director_routing_types_v0.go`;
- `director_decision_routing_helpers_v0.go`;
- `director_decision_routing_validation_v0.go`;
- `director_stats_types_v0.go`;
- `director_stats_v0.go`;
- tests y docs locales.

Cierre:

- valida `ask_director` y `ask_user`;
- valida `request_capacity` y `request_agent`;
- valida `request_rework` y `record_replan_decision`;
- limita `record_review_result.status` a accepted/changes_requested/rejected;
- valida `DirectorAgentCompactStatsV0`;
- mantiene el director como pieza intercambiable y sin peso de programacion.
