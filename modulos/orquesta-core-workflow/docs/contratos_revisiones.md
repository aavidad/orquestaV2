# Contratos de revisiones v0

## `ReviewResultV0`

```text
Nombre: ReviewResultV0
Tipo: DTO/validador puro
Version: v0
Propietario: orquesta-core-workflow
Consumidores: adaptadores o comandos futuros antes de `AcceptReview`
Campos:
  - review_result_ref
  - review_request_id
  - delivery_ref
  - status: accepted | changes_requested | rejected
  - summary
  - evidence_refs opcional
  - quality_gate_ref opcional, ref opaca a `QualityGateRecorded`
Invariantes:
  - DTO compacto; no es comando, evento, handler, reducer ni outbox.
  - Requiere refs obligatorias no vacias y status del catalogo cerrado.
  - `quality_gate_ref` puede citar un gate durable de `quality_gates`, pero `ReviewResultV0` no consulta ni exige esa proyeccion.
  - `accepted` puede alimentar una aceptacion durable posterior, pero no cierra tarea por si mismo.
  - `changes_requested` y `rejected` no cierran tarea ni generan `AcceptReview`.
  - No contiene valores reales de provider/proveedor, HOME, OAuth, DB, runtime, conectores ni secretos; bloquea prompts/transcripts crudos.
Errores:
  - review_result_invalido
  - status_no_soportado
  - payload_invalido
  - detalle_prohibido
Estado: implementado local como DTO puro en NCW-035.
```

## `RecordReviewResult`

```text
Nombre: RecordReviewResult
Tipo: comando
Version: v0
Propietario: orquesta-core-workflow
Consumidores: entradas del workflow durable tras una revision externa
Payload: ReviewResultV0
Invariantes:
  - Requiere run activo.
  - Requiere fase actual `revision` activa.
  - `review_request_id` debe existir ya en `OrchestrationRunV0.Reviews`.
  - `delivery_ref` debe existir ya en `OrchestrationRunV0.Deliveries`.
  - Idempotencia por `review_result_ref`; repetir el mismo resultado exacto es no-op.
  - Un mismo `review_result_ref` con status, review o entrega distinta es conflicto de transicion.
  - Si `review_result_ref` ya esta reflejado, el retry tambien debe coincidir en command/idempotency/event esperado y payload completo normalizado.
  - No emite outbox.
  - No genera `AcceptReview`, no cierra tarea, no cierra fase y no dispara rework automatico.
  - No contiene valores reales de provider/proveedor, HOME, OAuth, DB, runtime, conectores ni secretos; bloquea prompts/transcripts crudos.
Errores:
  - payload_invalido
  - transicion_invalida
  - detalle_prohibido
Estado: implementado local en NCW-042; identidad fuerte extendida en NCW-060.
```

## `ReviewResultRecorded`

```text
Nombre: ReviewResultRecorded
Tipo: evento
Version: v0
Propietario: orquesta-core-workflow
Consumidores: reducer, replay durable, replan/rework futuro
Payload: ReviewResultV0
Proyeccion:
  - `OrchestrationRunV0.ReviewResults` guarda refs compactas `review_result_ref + status + review_request_id + delivery_ref`.
Invariantes:
  - Evento compacto y append-only.
  - Solo pertenece a la fase `revision`.
  - `review_request_id` y `delivery_ref` deben existir ya en la proyeccion del run antes de aplicar.
  - `accepted`, `changes_requested` y `rejected` solo se registran en `ReviewResults`.
  - `accepted` no crea `ReviewAccepted` automaticamente.
  - `changes_requested` y `rejected` no cierran tarea ni crean rework automatico.
  - `ApplyEventV0` registra/verifica `CommandEffects` para impedir que la misma ref oculte otro evento, idempotency key, command_id o payload.
  - No contiene valores reales de DB, provider/proveedor, HOME, OAuth, runtime, conectores ni secretos; bloquea prompts/transcripts crudos.
Errores:
  - evento_invalido
  - payload_invalido
  - secuencia_invalida
  - detalle_prohibido
Estado: implementado local en NCW-042; identidad fuerte extendida en NCW-060.
```

## `RequestReview`

```text
Nombre: RequestReview
Tipo: comando
Version: v0
Propietario: orquesta-core-workflow
Consumidores: entradas futuras del workflow durable
Campos:
  - review_request_id
  - phase_id
  - delivery_ref
  - summary
  - evidence_refs opcional
Invariantes:
  - Requiere run activo.
  - `phase_id` debe ser `revision` y estar como fase actual activa.
  - `delivery_ref` debe existir ya en `OrchestrationRunV0.Deliveries`.
  - Si `review_request_id` ya esta reflejado, el retry solo es idempotente si coincide la huella durable de comando y payload normalizado.
  - No decide revisor, modelo, runtime, DB, proveedor, HOME, OAuth ni adaptadores concretos.
  - No emite outbox ni acepta la revision.
Errores:
  - payload_invalido
  - fase_no_soportada
  - transicion_invalida
  - detalle_prohibido
Estado: implementado local en NCW-020; identidad fuerte extendida en NCW-059.
```

## `RequestRework`

```text
Nombre: RequestRework
Tipo: comando
Version: v0
Propietario: orquesta-core-workflow
Consumidores: entradas futuras del workflow durable tras una revision no aceptada
Campos:
  - rework_request_ref
  - phase_id
  - review_result_ref
  - review_request_id
  - delivery_ref
  - summary
  - evidence_refs opcional
Invariantes:
  - Requiere run activo.
  - `phase_id` debe ser `revision` y estar como fase actual activa.
  - `review_request_id` debe existir ya en `OrchestrationRunV0.Reviews`.
  - `delivery_ref` debe existir ya en `OrchestrationRunV0.Deliveries`.
  - Debe existir antes un `ReviewResultRecorded` con status `changes_requested` o `rejected`, mismo `review_result_ref`, `review_request_id` y `delivery_ref`.
  - Idempotencia por `rework_request_ref`; repetir la misma proyeccion compacta es no-op.
  - Un mismo `rework_request_ref` con resultado, revision o entrega distinta es conflicto de transicion.
  - Si `rework_request_ref` ya esta reflejado, el retry tambien debe coincidir en command/idempotency/event esperado y payload completo normalizado.
  - No emite outbox, no relanza agentes, no registra replan, no acepta revision, no cierra tarea, no cierra fase y no cierra run.
  - No contiene valores reales de provider/proveedor, HOME, OAuth, DB, runtime, conectores ni secretos; bloquea prompts/transcripts crudos.
Errores:
  - payload_invalido
  - fase_no_soportada
  - transicion_invalida
  - detalle_prohibido
Estado: implementado local en NCW-044; identidad fuerte extendida en NCW-060.
```

## `ReworkRequested`

```text
Nombre: ReworkRequested
Tipo: evento
Version: v0
Propietario: orquesta-core-workflow
Consumidores: reducer, replay durable, replan futuro
Campos:
  - rework_request_ref
  - phase_id
  - review_result_ref
  - review_request_id
  - delivery_ref
  - summary
  - evidence_refs opcional
Proyeccion:
  - `OrchestrationRunV0.ReworkRequests` guarda refs compactas `rework_request_ref + review_result_ref + review_request_id + delivery_ref`.
Invariantes:
  - Evento compacto y append-only.
  - Solo pertenece a la fase `revision`.
  - `review_request_id`, `delivery_ref` y `review_result_ref` deben existir ya en la proyeccion del run antes de aplicar.
  - El resultado asociado debe tener status `changes_requested` o `rejected` para la misma revision y entrega.
  - `ApplyEventV0` proyecta `rework_request_ref` en `rework_requests` sin duplicar, rechaza conflictos y registra/verifica `CommandEffects`.
  - No contiene resultado completo de revision, prompts, transcripts, DB, provider/proveedor, HOME, OAuth, runtime, conectores ni secretos.
  - No emite outbox, no relanza agentes, no registra replan, no acepta revision, no cierra tarea, no cierra fase y no cierra run.
Errores:
  - evento_invalido
  - payload_invalido
  - secuencia_invalida
  - detalle_prohibido
Estado: implementado local en NCW-044; identidad fuerte extendida en NCW-060.
```

## `RecordReplanDecision`

```text
Nombre: RecordReplanDecision
Tipo: comando
Version: v0
Propietario: orquesta-core-workflow
Consumidores: entradas futuras del workflow durable tras una solicitud de retrabajo
Campos:
  - replan_ref
  - run_ref
  - task_ref
  - source_ref
  - accepted_action: split_task | retry_task | replace_agent | escalate_capacity | ask_director | abort_task
  - followup_refs
  - summary
  - evidence_refs opcional
Invariantes:
  - Requiere run activo.
  - `run_ref` debe coincidir con el run actual.
  - Si `source_ref` apunta a `ReworkRequested`, la fase actual debe ser `revision`.
  - Si `source_ref` apunta a `AgentFailed`, `AgentWorkAssessed` o `QualityGateRecorded(blocked)`, la fase actual debe ser `programacion`.
  - `task_ref` debe existir ya en `OrchestrationRunV0.Tasks`.
  - `source_ref` debe apuntar a un `ReworkRequested`, `AgentFailed`, `AgentWorkAssessed` o `QualityGateRecorded(blocked)` ya proyectado.
  - Idempotencia por `replan_ref`; repetir la misma proyeccion compacta es no-op.
  - Un mismo `replan_ref` con fuente, tarea, accion o followups distintos es conflicto de transicion.
  - Si `replan_ref` ya esta reflejado, el retry tambien debe coincidir en command/idempotency/event esperado y payload completo normalizado.
  - No emite outbox, no crea tareas, no pide capacidad, no relanza agentes, no cierra tarea/fase/run.
  - `followup_refs` solo traza efectos que deben materializarse por comandos separados.
  - No contiene valores reales de provider/proveedor, HOME, OAuth, DB, runtime, conectores ni secretos; bloquea prompts/transcripts crudos.
Errores:
  - payload_invalido
  - transicion_invalida
  - detalle_prohibido
Estado: implementado local en NCW-045, ampliado en NCW-054/NCW-069 e identidad fuerte extendida en NCW-060.
```

## `ReplanDecisionRecorded`

```text
Nombre: ReplanDecisionRecorded
Tipo: evento
Version: v0
Propietario: orquesta-core-workflow
Consumidores: reducer, replay durable, director y futuros cortes de capacidad/agentes
Campos:
  - replan_ref
  - run_ref
  - task_ref
  - source_ref
  - accepted_action
  - followup_refs
  - summary
  - evidence_refs opcional
Proyeccion:
  - `OrchestrationRunV0.ReplanDecisions` guarda refs compactas `replan_ref + source_ref + task_ref + accepted_action + followup_refs`.
Invariantes:
  - Evento compacto y append-only.
  - `run_ref`, `task_ref` y `source_ref` deben coincidir con proyeccion durable previa.
  - `source_ref` apunta a `ReworkRequested` en `revision`, o a `AgentFailed`, `AgentWorkAssessed` o `QualityGateRecorded(blocked)` en `programacion`.
  - `ApplyEventV0` registra/verifica `CommandEffects` para impedir que la misma ref oculte otro evento, idempotency key, command_id o payload.
  - No contiene prompts, transcripts, DB, provider/proveedor, HOME, OAuth, runtime, conectores ni secretos.
  - No emite outbox, no crea tareas, no pide capacidad, no relanza agentes, no acepta revision y no cierra tarea/fase/run.
Errores:
  - evento_invalido
  - payload_invalido
  - secuencia_invalida
  - detalle_prohibido
Estado: implementado local en NCW-045, ampliado en NCW-054/NCW-069 e identidad fuerte extendida en NCW-060.
```

## `AcceptReview`

```text
Nombre: AcceptReview
Tipo: comando
Version: v0
Propietario: orquesta-core-workflow
Consumidores: entradas futuras del workflow durable
Campos:
  - accepted_review_ref
  - phase_id
  - review_request_id
  - delivery_ref
  - summary
  - evidence_refs opcional
Invariantes:
  - Requiere run activo.
  - `phase_id` debe ser `revision` y estar como fase actual activa.
  - `review_request_id` debe existir ya en `OrchestrationRunV0.Reviews`.
  - `delivery_ref` debe existir ya en `OrchestrationRunV0.Deliveries`.
  - Debe existir antes un `ReviewResultRecorded` con status `accepted`, mismo `review_request_id` y mismo `delivery_ref`.
  - Si `accepted_review_ref` ya esta reflejado, el retry solo es idempotente si coincide la huella durable de comando y payload normalizado.
  - No decide merge, cierre de tarea, cierre de fase ni despliegue.
  - El cierre de tarea posterior pertenece a `CloseTask`; ver `docs/contratos_cierre_tareas.md`.
  - No ejecuta conectores ni contiene valores reales de runtime, DB, proveedor, OAuth o HOME.
Errores:
  - payload_invalido
  - fase_no_soportada
  - transicion_invalida
  - detalle_prohibido
Estado: implementado local en NCW-022.
Endurecimiento: NCW-043 exige resultado de revision aceptado antes de aceptar formalmente; NCW-059 anade identidad fuerte por `CommandEffects`.
```

## `ReviewAccepted`

```text
Nombre: ReviewAccepted
Tipo: evento
Version: v0
Propietario: orquesta-core-workflow
Consumidores: reducer, replay durable, cierre de tarea futuro
Campos:
  - accepted_review_ref
  - phase_id
  - review_request_id
  - delivery_ref
  - summary
  - evidence_refs opcional
Invariantes:
  - Evento compacto y append-only.
  - Solo pertenece a la fase `revision`.
  - `review_request_id` y `delivery_ref` deben existir ya en la proyeccion del run antes de aplicar.
  - Debe existir antes un `ReviewResultRecorded` con status `accepted`, mismo `review_request_id` y mismo `delivery_ref`.
  - `ApplyEventV0` proyecta `accepted_review_ref` en `accepted_reviews` sin duplicar y registra/verifica `CommandEffects`.
  - No contiene resultado completo de revision, prompts, transcripts ni detalles de conectores.
Errores:
  - evento_invalido
  - payload_invalido
  - secuencia_invalida
  - detalle_prohibido
Estado: implementado local en NCW-022.
Endurecimiento: NCW-043 exige resultado de revision aceptado antes de aplicar el evento; NCW-059 anade identidad fuerte por `CommandEffects`.
```

## `ReviewRequested`

```text
Nombre: ReviewRequested
Tipo: evento
Version: v0
Propietario: orquesta-core-workflow
Consumidores: reducer, replay durable, aceptacion de revision futura
Campos:
  - review_request_id
  - phase_id
  - delivery_ref
  - summary
  - evidence_refs opcional
Invariantes:
  - Evento compacto y append-only.
  - Solo pertenece a la fase `revision`.
  - `delivery_ref` debe existir ya en la proyeccion del run antes de aplicar.
  - `ApplyEventV0` proyecta `review_request_id` en `reviews` sin duplicar y registra/verifica `CommandEffects`.
  - No contiene resultado completo de revision, prompts, transcripts ni detalles de runtime.
Errores:
  - evento_invalido
  - payload_invalido
  - secuencia_invalida
  - detalle_prohibido
Estado: implementado local en NCW-020; identidad fuerte extendida en NCW-059.
```
