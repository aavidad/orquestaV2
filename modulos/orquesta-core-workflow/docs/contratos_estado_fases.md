# Contratos de estado y fases v0

## `OrchestrationRunV0`

```text
Nombre: OrchestrationRunV0
Tipo: dto
Version: v0
Propietario: orquesta-core-workflow
Consumidores: core-workflow, persistence futura, observability futura, MCP/web/CLI como proyeccion
Campos minimos:
  - run_id
  - project_ref
  - app_spec_ref
  - status
  - current_phase
  - phases
  - tasks
  - function_contracts
  - decisions
  - capacity_requests
  - capacity_decisions
  - agents
  - started_agents
  - failed_agents
  - stopped_agents
  - confirmed_stopped_agents
  - agent_assessments
  - agent_lease_expirations
  - concurrency_gates
  - quality_gates
  - deliveries
  - reviews
  - review_results
  - rework_requests
  - replan_decisions
  - accepted_reviews
  - closed_tasks
  - validations
  - closures
  - director_questions
  - director_answered_questions
  - blockers
  - command_effects
  - last_event_id
  - last_sequence
Invariantes:
  - Se reconstruye por replay de eventos.
  - No contiene tablas, DSN, HOME, OAuth, comandos tmux, proveedor LLM ni secretos.
  - `current_phase` debe existir en `phases` salvo estado inicial vacio.
  - Los comandos/eventos operativos con `phase_id` solo progresan si esa fase es la `current_phase`, salvo no-op idempotente ya reflejado.
  - `capacity_requests` registra solicitudes; `capacity_decisions` registra una decision compacta por solicitud antes de permitir agentes.
  - `agents` significa solicitudes de agente registradas; `started_agents`, `failed_agents`, `stopped_agents`, `confirmed_stopped_agents` y `agent_assessments` son proyecciones separadas del ciclo logico.
  - `stopped_agents` significa parada solicitada; `confirmed_stopped_agents` significa confirmacion durable posterior de parada efectiva por ref opaca.
  - `agent_lease_expirations` guarda expiraciones observadas por `lease_ref`; no implica parada, fallo ni replan automatico.
  - `concurrency_gates` guarda gates evaluados por `gate_ref`; no implica scheduler ni bloqueo global del run.
  - `quality_gates` guarda gates de calidad por `gate_ref` con decision y sujeto compactos; no implica cierre, rework, bloqueo ni outbox automatico.
  - Las tareas y contratos guardan referencias opacas, no datos privados de otros modulos.
  - `director_questions` guarda refs compactas a preguntas emitidas, no el payload completo.
  - `director_answered_questions` guarda `question_id` respondidos para impedir re-bloqueos por reintentos antiguos.
  - `review_results`, `rework_requests` y `replan_decisions` guardan proyecciones compactas de revision/rework/replan, no payloads completos ni efectos ejecutados.
  - `closed_tasks` guarda `task_id` compactos cerrados por `TaskClosed`, no entregas ni revisiones completas.
  - `validations` guarda `validation_ref` compactas registradas por `FinalValidationRegistered`, no resultados completos de validacion.
  - `closures` guarda `closure_ref` compactas registradas por `RunClosed`, no resultados completos de cierre.
  - El estado `cerrado` del run solo cambia por `RunClosed`; cerrar una fase con `ClosePhase` no cierra el run.
  - `command_effects` guarda huellas compactas para `RunStarted`, `PhaseOpened`, `RunBlocked`, `PhaseClosed` y el resto de refs criticas; no guarda payload completo ni secretos.
  - `last_sequence` refleja la ultima secuencia aplicada y no se infiere desde contadores de dominio.
Errores:
  - run_invalido
  - fase_invalida
  - estado_inconsistente
  - detalle_prohibido
Pruebas de contrato:
  - Replay durable de eventos minimos reconstruye el mismo estado y secuencia.
  - Serializacion no contiene detalles prohibidos.
Estado: implementado inicial en NCW-001 y endurecido hasta NCW-067.
```

## `OrchestrationPhaseV0`

Fases permitidas iniciales:

- `descubrimiento`;
- `brainstorming_arquitectura`;
- `votacion_y_decision`;
- `planificacion_microtareas`;
- `programacion`;
- `documentacion`;
- `integracion`;
- `revision`;
- `validacion_final`;
- `cierre`.

```text
Nombre: OrchestrationPhaseV0
Tipo: dto
Version: v0
Propietario: orquesta-core-workflow
Consumidores: core-workflow, web/MCP/observability como proyeccion
Campos:
  - id
  - status
  - opened_at
  - closed_at
  - entry_criteria
  - exit_criteria
  - evidence_required
  - recommended_capacity
Invariantes:
  - `id` pertenece al catalogo de fases v0.
  - Solo una fase puede estar `activa`.
  - Una fase cerrada conserva evidencia de cierre.
  - `PhaseOpened` registra identidad fuerte por `event_id` para no bloquear reaperturas por `phase_id`.
  - `PhaseClosed` registra identidad fuerte por `closure_ref`.
  - La capacidad recomendada no contiene proveedor concreto.
Errores:
  - fase_no_soportada
  - fase_ya_abierta
  - fase_sin_evidencia
Estado: candidato para NCW-001.
```
