# Indice y mapa local: orquesta-app-codex-stack

Fecha: 2026-05-26.

Este documento es una guia de entrada rapida al stack Codex. No sustituye a la
foto vigente del repo: `AGENTS.md`, `docs/estado_actual_2026-05-17.md`,
`docs/guia_nucleo_orquestacion_2026-05-17.md` y
`docs/matriz_pruebas_reales_y_smoke_2026-05-17.md` mandan si hay contradiccion.

## Frontera

`orquesta-app-codex-stack` es una composicion exterior opt-in. Puede conocer
runtime Codex, stores concretos, ficheros de ACK, bridge de entregas y rutas
HTTP/MCP del servidor porque vive fuera del core. No define el nucleo y no debe
meter proveedor, modelo, HOME, DB, OPES ni paths locales en paquetes neutrales.

El flujo operativo local por defecto se lee asi:

```text
web/API/MCP/servidor opt-in
  -> StackV0 / ejecutores MCP del stack
  -> StartAppDirectorV0 con modo vacio normalizado a goal_first
  -> GoalWorkSpecV0 con objetivo, reglas, contexto, write-set y tests
  -> GoalLauncher / GoalObserver opt-in
  -> GoalWorkStateV0 durable
  -> observe_goal, validacion final y cierre por evidencias
  -> bridge domain_work solo si esta inyectado
```

El loop historico queda como ruta de compatibilidad explicita:

```text
web/API/MCP/servidor opt-in con director_execution_mode=legacy_director_loop
  -> StackV0 / ejecutores MCP del stack
  -> app-director-service por puertos inyectados
  -> workflow/outbox neutral
  -> dispatcher Codex o fake runtime inyectado
  -> ACK/delivery/progress observados por el stack
  -> review/rework/tests/cierre por refs causales
  -> bridge domain_work solo si esta inyectado
```

## Mapa de piezas

| Area | Ficheros guia | Responsabilidad |
| --- | --- | --- |
| Composicion y config | `config_v0.go`, `stack_v0.go`, `sources_v0.go`, `dispatchers_v0.go` | Construir puertos reales/fakes, handlers y dispatchers sin defaults ocultos. |
| Paquete de agente | `spec_packet_v0.go`, `spec_task_v0.go`, `spec_task_council_v0.go`, `spec_resolver_v0.go`, `director_decision_contract_v0.go` | Convertir tareas Orquesta en contexto Codex compacto con linaje, write-set, criterios y refs; `CODEX-COUNCIL-PACKET-V0` deja propuesta/critica/voto como `decision_council`, no como director generico. |
| Wait/drain/observacion | `waiter_v0.go`, `drain_v0.go`, `drain_wait_scope_v0.go`, `drain_observations_v0.go` | Ruta legacy/no-Goal: esperar solo `WaitAgentRefs` cuando existan, ingerir ACK/deliveries y reentrar al director. Las runs goal-first deben observarse por estado Goal y no entrar en `DrainRunV0`. |
| Supervision residente | `run_supervisor_v0.go`, `run_coordinator_v0.go`, `codex_supervisor_stack_lifecycle_v0.go` | Ruta legacy/no-Goal: avanzar runs por ticks acotados, cola global y ciclo de drain/outbox. En goal-first, la supervision debe redirigir a `observe_goal` o bloquear con estado reparable si falta `GoalWorkStateV0`. |
| Autoprogramacion | `autoprogramming_bridge_v0.go`, `autoprogramming_prepare_run_mcp_executor_v0.go`, `autoprogramming_resident_*` | Con backend Goal compila/lanza `GoalWorkSpecV0` y guarda `GoalWorkStateV0`; solo conserva plan-state legacy y self-repair del loop historico cuando no hay Goal o se fuerza compatibilidad. |
| Review/rework/replan | `review_rework_replan_source_v0.go`, `review_gate_*`, `assessment_replan_source_v0.go` | Traducir review negativa, falta de progreso o rework en followups causales. |
| Tests requeridos | `codex_ack_required_test_runner_v0.go`, `domain_work_required_tests_*` | Generar/recoger `RequiredTestEvidenceV0` por puerto y no cerrar con summaries textuales. |
| Cierre operativo | `operational_closure_source_*.go`, `operational_closure_domain_work_source_v0.go` | Construir requests de cierre desde task store, eventos, reviews, tests y receipts de dominio. |
| Domain work | `domain_work_delivery_*`, `external_job_stats_source_v0.go` | Enviar artefactos a conectores externos opt-in por refs opacas e idempotencia. |
| Operacion y panel | `agent_usage_*`, `ops_agent_runtime_detail_*`, `run_control_*` tests | Exponer stats, uso, progreso y control sin filtrar internals sensibles al core. |

## Estado operativo local

Cerrado para este stack salvo regresion demostrada:

- `WaitAgentRefs` no vacio acota pending, wait e ingesta.
- `CODEX-REQTEST-REAL-E2E` cubre un agente Codex real con tests requeridos y
  cierre causal.
- `CODEX-WAVE-REAL` cubre ola/cohorte amplia con proveedor real.
- `CODEX-RECURSION-REAL` cubre arbol 1 -> 2 -> 4 con provider real, refs
  parent/child, waits acotados, review causal y cierre.
- `EXT-NO-OPES` cubre app externa temporal no-OPES con `codex-fake`, submitter
  real opt-in, review, required tests y plan cerrado.

Pendiente verificable, no hecho:

- OPES temporal real de derivados/cierre hasta
  `generate_audio_asset -> audio_asset`.
- Politica productiva de tests de dominio no-OPES mas alla del smoke temporal.
- Conectores productivos de uso/cuota por proveedor.
- Cualquier nuevo blocker real de autoprogramacion que no este cubierto por las
  decisiones y pruebas ya documentadas.

## Plan de secciones

Usar los documentos locales con esta funcion:

- `README.md`: entrada corta, frontera opt-in y comandos basicos.
- `docs/contratos.md`: contratos vivos de composicion, puertos, supervisor,
  autoprogramacion, domain work y cierre.
- `docs/decisiones.md`: decisiones historicas aceptadas. No usar como backlog
  si la matriz o `estado_actual` ya cerraron el frente.
- `docs/pruebas.md`: evidencia local, smokes opt-in y guardas de pruebas.
- `docs/tareas.md`: backlog local historico del modulo; las tareas nuevas deben
  citar evidencia vigente y no reabrir smokes cerrados sin regresion.
- `docs/review_rework_replan.md`: detalle focal de la fuente de rework/replan.
- Este indice: mapa de lectura y clasificacion de estado.

## Regla de actualizacion

Cuando un cambio anada una pieza transversal al stack, actualizar este indice si
el lector ya no puede encontrar su owner en la tabla. Si solo cambia un caso
focal, actualizar `docs/pruebas.md` o `docs/tareas.md` sin inflar este mapa.
