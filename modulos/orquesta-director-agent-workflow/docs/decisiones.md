# Decisiones locales

```text
Fecha: 2026-05-10
Decision: Traducir la superficie minima de app completa del director externo a comandos publicos existentes.
Motivo: el director reemplazable debe poder pedir informacion, capacidad, agentes, rework y replan sin que la sesion humana ejecute pasos manuales.
Impacto: el puente traduce `ask_director`, `ask_user`, `request_capacity`, `request_agent`, `request_rework` y `record_replan_decision`; `ask_user` usa `AskDirector` con `target_group=user` porque el core no publica un comando `AskUser` separado.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: Exponer stats compactas al puerto de fuente de decisiones.
Motivo: el director debe tomar decisiones de enrutado con contadores y pendientes, no con snapshots largos ni payloads operativos.
Impacto: `DirectorAgentDecisionSourceRequestV0` incluye `Stats` y `BuildDirectorAgentCompactStatsV0` resume `OrchestrationRunV0` publico sin stores, runtime, DB ni proveedor.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: La idempotencia del puente usa `command_ref`, no `decision_ref`.
Motivo: una decision del director puede contener un payload que referencia una
decision aceptada previa, por ejemplo `publish_function_contract.decision_ref`.
Si el puente usa esa ref de negocio como idempotency key, mezcla comandos
distintos y el workflow rechaza decisiones validas por conflicto reflejado.
Impacto: cada comando traducido queda idempotente por su `command_ref`
ejecutable, mientras los payloads pueden seguir apuntando a decisiones previas
cuando el contrato del core lo exige.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: Traducir `close_task` del director a `CloseTask` antes de validacion final.
Motivo: el flujo durable separa `AcceptReview`, `CloseTask` y `RegisterFinalValidation`; el puente debe respetar esa granularidad sin mutar el run directamente.
Impacto: `BuildDirectorAgentWorkflowCommandV0` construye `NewCloseTaskCommandV0` y `ApplyDirectorAgentDecisionV0` lo aplica por `HandleCommandV0`, stores y sinks inyectados.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: Traducir cierre del director como comandos publicos separados.
Motivo: el director debe poder finalizar una solicitud sin que la sesion humana
ejecute comandos manuales. El cierre sigue pasando por el core y por la capa de
politica de aplicacion, no por mutaciones directas.
Impacto: `register_final_validation` se traduce a `RegisterFinalValidation` y
`close_run` a `CloseRun`. Los campos `request_kind` y `execution_mode` quedan en
el contrato del director para que la aplicacion pueda bloquear cierres parciales
antes de aplicar el comando.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: Separar contrato de director y adaptador a workflow.
Motivo: el DTO del director debe seguir puro y portable; la traduccion a
comandos publicos pertenece a un puente hexagonal.
Impacto: se puede cambiar el director externo sin tocar el core y se puede
probar la traduccion sin runtime real.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: El puente cubre fases y decision arquitectonica antes de planificar tareas.
Motivo: `PublishFunctionContract` depende de una decision aceptada y `CreateMicrotask` depende de contratos publicados; la cadena debe poder nacer desde brainstorming sin intervencion manual.
Impacto: `open_phase`, `request_vote` y `accept_decision` se traducen a comandos publicos, y la prueba de flujo aplica la cadena completa del director.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: `create_microtask` exige materializar el DTO completo en `TaskStore`.
Motivo: el evento durable del core solo proyecta refs compactas; sin store externo el scheduler veria `run.tasks` pero no podria construir candidatos de programacion.
Impacto: `ApplyDirectorAgentDecisionV0` requiere `TaskStore` solo para microtareas, guarda por puerto inyectado y evita crear trabajo no programable.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: Traducir decisiones de planificacion del director a comandos publicos, no a mutaciones directas.
Motivo: el director debe poder autoprogramar Orquesta creando contratos y microtareas, pero sin conocer estructura interna del core ni persistencia.
Impacto: `publish_function_contract` y `create_microtask` pasan por `HandleCommandV0`, reducers y puertos inyectados; no se crea store, runtime, DB ni proveedor por defecto.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: No traducir `propose_autonomous_plan_team` en el puente workflow v0.
Motivo: hoy es un DTO de propuesta del director, no un comando publico de `orquesta-core-workflow`.
Impacto: el adaptador lo rechaza como `director_agent_command_no_soportado`; una capa de aplicacion puede leerlo como artefacto validado sin store, runtime ni proveedor por defecto.
Estado: aceptada.
```
