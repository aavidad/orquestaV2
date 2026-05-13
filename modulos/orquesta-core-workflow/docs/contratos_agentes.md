# Contratos de agentes: orquesta-core-workflow

Este archivo contiene contratos locales de solicitud de agente. El indice
global del modulo sigue en `docs/contratos.md`.

## `RequestAgent`

```text
Nombre: RequestAgent
Tipo: comando
Version: v0 candidato local
Propietario: orquesta-core-workflow
Consumidores: adaptadores inbound futuros, planificador durable futuro
Payload:
  - agent_request_id
  - phase_id
  - task_ref opcional
  - capacity_request_ref
  - role
  - summary
  - evidence_refs opcional
Salida:
  - AgentRequested
  - outbox LaunchRuntimeAgent con target agent_launcher
Invariantes:
  - Handler puro: no ejecuta nada ni decide runtime, proveedor, modelo, HOME, OAuth, cuenta o adaptador.
  - `phase_id` pertenece al catalogo v0.
  - `phase_id` debe coincidir con `run.current_phase` salvo no-op idempotente ya reflejado.
  - `capacity_request_ref` debe apuntar a una `CapacityDecided` ya proyectada; enlaza la decision sin transportar su detalle.
  - Payload compacto, sin DB, HOME, OAuth, Codex, Claude, Ollama, vLLM ni secretos.
  - Si `agent_request_id` ya esta proyectado, el comando debe coincidir con la huella `CommandEffects` original.
  - Si coincide y no hay lifecycle terminal, reemite solo outbox `LaunchRuntimeAgent`.
  - Si coincide y existe `AgentStarted`, `AgentFailed` o `AgentStopRequested`, devuelve no-op idempotente sin outbox.
Errores:
  - payload_invalido
  - detalle_prohibido
  - fase_no_soportada
  - transicion_invalida
Estado: implementado local en NCW-011 y endurecido en NCW-039/NCW-055.
```

## `StopAgent`

```text
Nombre: StopAgent
Tipo: comando
Version: v0 candidato local
Propietario: orquesta-core-workflow
Consumidores: adaptadores inbound futuros, planificador durable futuro
Payload:
  - agent_request_id
  - reason_code
  - summary
  - evidence_refs opcional
Salida:
  - AgentStopRequested
  - outbox StopRuntimeAgent con target agent_launcher
Invariantes:
  - Handler puro: no ejecuta nada ni decide runtime, proveedor, modelo, HOME, OAuth, cuenta o adaptador.
  - `agent_request_id` debe existir ya en `run.agents`.
  - Si `agent_request_id` ya esta en `run.stopped_agents`, el comando debe coincidir con la huella `CommandEffects` original.
  - Si coincide y la parada no esta confirmada, reemite solo outbox `StopRuntimeAgent`; si ya esta confirmada, devuelve no-op idempotente.
  - No puede existir `AgentFailed` previo para el mismo agente.
  - Payload compacto, sin DB, HOME, OAuth, Codex, Claude, Ollama, vLLM ni secretos.
  - `agent_request_id` y `reason_code` no pueden contener `#`, porque forman una proyeccion compacta parseable en `agent_stop_requests`.
Errores:
  - payload_invalido
  - detalle_prohibido
  - transicion_invalida
Estado: implementado local en NCW-028 y endurecido en NCW-053/NCW-055.
```

## `RegisterAgentStopConfirmed`

```text
Nombre: RegisterAgentStopConfirmed
Tipo: comando
Version: v0 candidato local
Propietario: orquesta-core-workflow
Consumidores: adaptadores inbound futuros, dispatcher/runtime futuro
Payload:
  - confirmation_ref
  - agent_request_id
  - observed_at
  - summary
  - evidence_refs opcional
Salida:
  - AgentStopConfirmed
Invariantes:
  - Handler puro: no ejecuta nada ni consulta procesos, DB, proveedor, HOME, OAuth, cuenta o adaptador.
  - `agent_request_id` debe existir ya en `run.stopped_agents`; una confirmacion no puede adelantarse a `AgentStopRequested`.
  - Si `agent_request_id` ya esta en `run.confirmed_stopped_agents`, devuelve no-op solo si coincide la huella durable de comando y payload normalizado.
  - No emite outbox; es una senal compacta entrante de un adaptador externo.
  - Payload compacto, sin DB, HOME, OAuth, Codex, Claude, Ollama, vLLM ni secretos.
Errores:
  - payload_invalido
  - detalle_prohibido
  - transicion_invalida
Estado: implementado local en NCW-049; identidad fuerte extendida en NCW-062.
```

## `AssessAgentWork`

```text
Nombre: AssessAgentWork
Tipo: comando
Version: v0 candidato local
Propietario: orquesta-core-workflow
Consumidores: adaptadores inbound futuros, supervision durable futura
Payload:
  - assessment_ref
  - phase_id
  - agent_request_id
  - task_ref opcional
  - delivery_ref opcional
  - verdict: acceptable | needs_revision | garbage | loop_detected
  - action: continue | request_revision | stop_agent | ask_director
  - severity: low | medium | high | critical
  - summary
  - evidence_refs opcional
Salida:
  - AgentWorkAssessed
  - si action=stop_agent: AgentStopRequested y outbox StopRuntimeAgent con target agent_launcher
Invariantes:
  - Handler puro: no ejecuta nada ni decide runtime, proveedor, modelo, HOME, OAuth, cuenta o adaptador.
  - `phase_id` pertenece al catalogo v0.
  - `phase_id` debe coincidir con `run.current_phase` para registrar una evaluacion nueva.
  - `agent_request_id` debe existir ya en `run.agents`.
  - Si `delivery_ref` viene informado, debe existir ya en `run.deliveries`.
  - Si `assessment_ref` ya esta en `run.agent_assessments`, el comando debe coincidir con la huella `CommandEffects` original.
  - Si `AgentWorkAssessed` ya quedo proyectado pero falta `AgentStopRequested`, repetir el mismo comando emite solo `AgentStopRequested` y outbox `StopRuntimeAgent`.
  - Si tambien quedo proyectado `AgentStopRequested` pero se perdio el outbox, repetir el mismo comando reconstruye solo `StopRuntimeAgent` con la idempotency key del evento de parada.
  - `action=stop_agent` solo es valida para `verdict=garbage` o `verdict=loop_detected`.
  - Si el agente ya esta en `stopped_agents`, una nueva evaluacion se registra sin reenviar outbox de parada.
  - Payload compacto, sin DB, HOME, OAuth, Codex, Claude, Ollama, vLLM ni secretos.
Errores:
  - payload_invalido
  - detalle_prohibido
  - fase_no_soportada
  - transicion_invalida
Estado: implementado local en NCW-029; idempotencia de efectos endurecida en NCW-055.
```

## `AgentWorkAssessed`

```text
Nombre: AgentWorkAssessed
Tipo: evento
Version: v0 candidato local
Propietario: orquesta-core-workflow
Consumidores: reducer, replay durable, observability futura
Payload:
  - assessment_ref
  - phase_id
  - agent_request_id
  - task_ref opcional
  - delivery_ref opcional
  - verdict
  - action
  - severity
  - summary
  - evidence_refs opcional
Invariantes:
  - Evento compacto y append-only.
  - `phase_id` debe coincidir con `run.current_phase` al aplicar/replayar el evento.
  - `ApplyEventV0` exige que el agente exista en `agents`.
  - Si `delivery_ref` viene informado, `ApplyEventV0` exige que exista en `deliveries`.
  - Proyecta `assessment_ref` en `agent_assessments` sin duplicar.
  - Proyecta una huella `CommandEffects` por `(AgentWorkAssessed, assessment_ref)`.
  - Rechaza otra evaluacion durable con el mismo `assessment_ref` y distinta key, event_id, causation_id o payload.
  - `ReplayDurableEventsV0` acepta el evento y un duplicado exacto con la misma huella durable.
  - No contiene runtime, proveedor, modelo, HOME, OAuth, cuenta, adaptador ni secreto.
Errores:
  - evento_invalido
  - payload_invalido
  - detalle_prohibido
  - secuencia_invalida
Estado: implementado local en NCW-029; idempotencia de efectos endurecida en NCW-055.
```

## `AgentStopRequested`

```text
Nombre: AgentStopRequested
Tipo: evento
Version: v0 candidato local
Propietario: orquesta-core-workflow
Consumidores: reducer, replay durable, observability futura
Payload:
  - agent_request_id
  - reason_code
  - summary
  - evidence_refs opcional
Invariantes:
  - Evento compacto y append-only.
  - `ApplyEventV0` exige que el agente exista en `agents` y proyecta `agent_request_id` en `stopped_agents` sin duplicar.
  - Proyecta `agent_request_id#reason:<reason_code>` en `agent_stop_requests` para observabilidad historica sin payload largo.
  - Proyecta una huella `CommandEffects` por `(AgentStopRequested, agent_request_id)`.
  - Rechaza otro stop durable para el mismo agente con distinta key, event_id, causation_id o payload.
  - Rechaza `agent_request_id` si ya existe en `failed_agents`.
  - `ReplayDurableEventsV0` acepta el evento y un duplicado exacto con la misma huella durable.
  - No contiene runtime, proveedor, modelo, HOME, OAuth, cuenta, adaptador ni secreto.
Errores:
  - evento_invalido
  - payload_invalido
  - detalle_prohibido
  - secuencia_invalida
Estado: implementado local en NCW-028; endurecido en NCW-053, NCW-055 y NCW-073.
```

## `AgentStopConfirmed`

```text
Nombre: AgentStopConfirmed
Tipo: evento
Version: v0 candidato local
Propietario: orquesta-core-workflow
Consumidores: reducer, replay durable, observability futura
Payload:
  - confirmation_ref
  - agent_request_id
  - observed_at
  - summary
  - evidence_refs opcional
Invariantes:
  - Evento compacto y append-only.
  - `ApplyEventV0` exige que el agente ya este en `stopped_agents` y proyecta `agent_request_id` en `confirmed_stopped_agents` sin duplicar.
  - Proyecta una huella `CommandEffects` por `(AgentStopConfirmed, agent_request_id)`.
  - Rechaza otra confirmacion durable para el mismo agente con distinta key, event_id, causation_id o payload.
  - `ReplayDurableEventsV0` acepta el evento y un duplicado exacto con la misma huella durable.
  - No contiene runtime, proveedor, modelo, HOME, OAuth, cuenta, adaptador ni secreto.
Errores:
  - evento_invalido
  - payload_invalido
  - detalle_prohibido
  - secuencia_invalida
Estado: implementado local en NCW-049; identidad fuerte extendida en NCW-062.
```

## `StopRuntimeAgentRequestV0`

```text
Nombre: StopRuntimeAgentRequestV0
Tipo: outbox
Version: v0 candidato local
Propietario: orquesta-core-workflow
Consumidores: puerto logico agent_launcher
Campos:
  - agent_request_id
  - run_id
  - reason_code
  - summary
  - evidence_refs opcional
Invariantes:
  - Se entrega como `StopRuntimeAgent` con target `agent_launcher`.
  - Es una solicitud logica de parada, no una llamada a runtime real.
  - `ValidateOutboxMessageV0` valida sus campos obligatorios y `run_id`.
  - No incluye proveedor, modelo, cuenta, HOME, OAuth, Codex, Claude, Ollama, vLLM ni adaptador.
Estado: implementado local en NCW-028.
```

## `AgentRequested`

```text
Nombre: AgentRequested
Tipo: evento
Version: v0 candidato local
Propietario: orquesta-core-workflow
Consumidores: reducer, replay durable, observability futura
Payload:
  - agent_request_id
  - phase_id
  - task_ref opcional
  - capacity_request_ref
  - role
  - summary
  - evidence_refs opcional
Invariantes:
  - Evento compacto y append-only.
  - `capacity_request_ref` es obligatorio y debe estar decidido en `capacity_decisions` al aplicar/replayar.
  - `ApplyEventV0` proyecta `agent_request_id` en `agents` sin duplicar.
  - `ReplayDurableEventsV0` acepta el evento con sequence estricta.
  - No contiene runtime, proveedor, modelo, HOME, OAuth, cuenta, adaptador ni secreto.
Errores:
  - evento_invalido
  - payload_invalido
  - detalle_prohibido
  - secuencia_invalida
Estado: implementado local en NCW-011 y endurecido en NCW-039.
```

## `AgentStarted`

```text
Nombre: AgentStarted
Tipo: comando_evento
Version: v0 candidato local
Propietario: orquesta-core-workflow
Consumidores: reducer, replay durable, observability futura
Entrada: RegisterAgentStarted
Payload:
  - agent_request_id
  - launch_ref
  - ack_ref
  - readiness_ref
  - evidence_refs opcional
Invariantes:
  - `agent_request_id` debe existir ya en `run.agents`.
  - No puede existir `AgentFailed` ni `AgentStopRequested` previo para el mismo agente.
  - Proyecta `agent_request_id` en `started_agents` sin duplicar y registra/verifica `CommandEffects`.
  - Repetir `RegisterAgentStarted` solo es no-op si coincide la huella durable de comando y payload normalizado.
  - `launch_ref`, `ack_ref` y `readiness_ref` son refs opacas; no se resuelven en el core.
  - No contiene runtime, proveedor, modelo, HOME, OAuth, cuenta, adaptador, PID, ruta, comando ni secreto.
Estado: implementado local en NCW-038, endurecido en NCW-052 e identidad fuerte extendida en NCW-062.
```

## `AgentFailed`

```text
Nombre: AgentFailed
Tipo: comando_evento
Version: v0 candidato local
Propietario: orquesta-core-workflow
Consumidores: reducer, replay durable, observability futura
Entrada: RegisterAgentFailed
Payload:
  - agent_request_id
  - launch_ref opcional
  - reason_code
  - retryable
  - evidence_refs opcional
Invariantes:
  - `agent_request_id` debe existir ya en `run.agents`.
  - No puede existir `AgentStarted` ni `AgentStopRequested` previo para el mismo agente.
  - Proyecta `agent_request_id` en `failed_agents` sin duplicar y registra/verifica `CommandEffects`.
  - Repetir `RegisterAgentFailed` solo es no-op si coincide la huella durable de comando y payload normalizado.
  - `reason_code` es compacto y no transporta detalles operativos.
  - No contiene runtime, proveedor, modelo, HOME, OAuth, cuenta, adaptador, PID, ruta, comando ni secreto.
Estado: implementado local en NCW-038, endurecido en NCW-052 e identidad fuerte extendida en NCW-062.
```

## `AgentLeaseExpired`

```text
Nombre: AgentLeaseExpired
Tipo: comando_evento
Version: v0 candidato local
Propietario: orquesta-core-workflow
Consumidores: reducer, replay durable, observability futura, director futuro
Entrada: RegisterAgentLeaseExpired
Payload:
  - run_ref
  - agent_request_id
  - lease_ref
  - reason_code
  - observed_at
  - recommended_action: retry | ask_director | stop_agent | mark_failed | mark_stopped | replan_task | alert_only
  - evidence_refs opcional
Invariantes:
  - `agent_request_id` debe existir ya en `run.agents`.
  - `run_ref` debe coincidir con el run activo.
  - `observed_at` llega de fuera en formato UTC compacto; el core no consulta reloj interno.
  - Proyecta una ref compacta en `agent_lease_expirations` sin duplicar y registra/verifica `CommandEffects`.
  - La identidad fuerte es por `lease_ref`; misma ref con accion/razon distinta, command_id distinto o idempotency_key distinta se rechaza como conflicto.
  - Repetir `RegisterAgentLeaseExpired` solo es no-op si coincide la huella durable de comando y payload normalizado.
  - No emite outbox, no para procesos, no marca agentes como failed/stopped y no replanifica por si solo.
  - `recommended_action` es una recomendacion durable; la accion real entra despues por `StopAgent`, `AskDirector`, `RecordReplanDecision` u otro comando separado.
  - No contiene runtime, proveedor, modelo, HOME, OAuth, cuenta, adaptador, PID, ruta, comando ni secreto.
Estado: implementado local en NCW-046; identidad fuerte extendida en NCW-063.
```

## `LaunchRuntimeAgentRequestV0`

```text
Nombre: LaunchRuntimeAgentRequestV0
Tipo: outbox
Version: v0 candidato local
Propietario: orquesta-core-workflow
Consumidores: puerto logico agent_launcher
Campos:
  - agent_request_id
  - run_id
  - phase_id
  - task_ref opcional
  - capacity_request_ref
  - role
  - summary
  - evidence_refs opcional
Invariantes:
  - Se entrega como `LaunchRuntimeAgent` con target `agent_launcher`.
  - Es una solicitud logica de lanzamiento, no una decision de runtime.
  - Solo se emite despues de una `CapacityDecided` durable para `capacity_request_ref`.
  - `ValidateOutboxMessageV0` valida sus campos obligatorios y `run_id`.
  - No incluye proveedor, modelo, cuenta, HOME, OAuth, Codex, Claude, Ollama, vLLM ni adaptador.
Estado: implementado local en NCW-011 y endurecido en NCW-039.
```
