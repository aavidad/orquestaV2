# Contratos de capacidad: orquesta-core-workflow

Este archivo contiene contratos locales de solicitud de capacidad. El indice
global del modulo sigue en `docs/contratos.md`.

## `RequestCapacity`

```text
Nombre: RequestCapacity
Tipo: comando
Version: v0 candidato local
Propietario: orquesta-core-workflow
Consumidores: adaptadores inbound futuros, planificador durable futuro
Payload:
  - capacity_request_id
  - phase_id
  - task_ref opcional
  - reason_code
  - summary
  - minimum_recommended_capacity opcional: low | medium | high | xhigh
  - evidence_refs opcional
Salida:
  - CapacityRequested
  - outbox RequestCapacityDecision con target capacity
Invariantes:
  - Handler puro: no ejecuta nada ni decide cuenta, proveedor, modelo o runtime concretos.
  - `phase_id` pertenece al catalogo v0.
  - `phase_id` debe coincidir con `run.current_phase` salvo no-op idempotente ya reflejado.
  - Payload compacto, sin valores reales de DB, HOME, OAuth, proveedor, modelo, runtime ni adaptadores; refs opacas de politica o arquitectura siguen permitidas por `orquesta-rails`.
  - Si `capacity_request_id` ya esta proyectado, el comando debe coincidir con la huella `CommandEffects` original.
  - Si coincide y falta `CapacityDecided`, reemite solo outbox `RequestCapacityDecision`.
  - Si coincide y existe `CapacityDecided`, devuelve no-op idempotente sin outbox.
Errores:
  - payload_invalido
  - detalle_prohibido
  - fase_no_soportada
  - transicion_invalida
Estado: implementado local en NCW-010; idempotencia de efectos endurecida en NCW-055.
```

## `CapacityRequested`

```text
Nombre: CapacityRequested
Tipo: evento
Version: v0 candidato local
Propietario: orquesta-core-workflow
Consumidores: reducer, replay durable, observability futura
Payload:
  - capacity_request_id
  - phase_id
  - task_ref opcional
  - reason_code
  - summary
  - minimum_recommended_capacity opcional
  - evidence_refs opcional
Invariantes:
  - Evento compacto y append-only.
  - `phase_id` debe coincidir con `run.current_phase` al aplicar/replayar el evento.
  - `ApplyEventV0` proyecta `capacity_request_id` en `capacity_requests` sin duplicar.
  - Proyecta una huella `CommandEffects` por `(CapacityRequested, capacity_request_id)`.
  - Rechaza otra solicitud durable con el mismo `capacity_request_id` y distinta key, event_id, causation_id o payload.
  - `ReplayDurableEventsV0` acepta el evento con sequence estricta.
  - No contiene decision ni valores reales de proveedor/modelo, runtime, HOME o adaptador.
Errores:
  - evento_invalido
  - payload_invalido
  - detalle_prohibido
  - secuencia_invalida
Estado: implementado local en NCW-010; idempotencia de efectos endurecida en NCW-055.
```

## `RegisterCapacityDecision`

```text
Nombre: RegisterCapacityDecision
Tipo: comando
Version: v0 candidato local
Propietario: orquesta-core-workflow
Consumidores: adaptadores inbound futuros del puerto capacity
Payload:
  - capacity_request_id
  - decision_ref
  - tier: low | medium | high | xhigh
  - reasoning_effort: low | medium | high | xhigh
  - summary opcional
  - evidence_refs opcional
Salida:
  - CapacityDecided
Invariantes:
  - Handler puro: no ejecuta runtime ni decide proveedor, modelo concreto, HOME, OAuth, cuenta o cuota real.
  - `capacity_request_id` debe existir ya en `run.capacity_requests`.
  - Solo puede existir una decision durable por `capacity_request_id`.
  - Repetir la misma decision devuelve no-op solo si coincide la huella durable de comando y payload normalizado.
  - Una decision distinta para la misma solicitud se rechaza como transicion invalida.
  - Payload compacto, sin valores reales de DB, HOME, OAuth, proveedor, modelo, runtime ni adaptadores; refs opacas de politica o arquitectura siguen permitidas por `orquesta-rails`.
Errores:
  - payload_invalido
  - detalle_prohibido
  - transicion_invalida
Estado: implementado local en NCW-039; identidad fuerte extendida en NCW-062.
```

## `CapacityDecided`

```text
Nombre: CapacityDecided
Tipo: evento
Version: v0 candidato local
Propietario: orquesta-core-workflow
Consumidores: reducer, replay durable, observability futura, RequestAgent
Payload:
  - capacity_request_id
  - decision_ref
  - tier
  - reasoning_effort
  - summary opcional
  - evidence_refs opcional
Invariantes:
  - Evento compacto y append-only.
  - `ApplyEventV0` exige `capacity_request_id` ya proyectado en `capacity_requests`.
  - `ApplyEventV0` proyecta una ref compacta unica en `capacity_decisions` y registra/verifica `CommandEffects`.
  - `RequestAgent` y `AgentRequested` exigen que `capacity_request_ref` apunte a una decision ya proyectada.
  - No contiene proveedor/modelo concreto, runtime real, HOME, OAuth, cuenta, cuota real ni adaptador.
Errores:
  - evento_invalido
  - payload_invalido
  - detalle_prohibido
  - secuencia_invalida
Estado: implementado local en NCW-039; identidad fuerte extendida en NCW-062.
```

## `CapacityDecisionRequestV0`

```text
Nombre: CapacityDecisionRequestV0
Tipo: outbox
Version: v0 candidato local
Propietario: orquesta-core-workflow
Consumidores: puerto logico capacity
Campos:
  - capacity_request_id
  - run_id
  - phase_id
  - task_ref opcional
  - reason_code
  - summary
  - minimum_recommended_capacity opcional
  - evidence_refs opcional
Invariantes:
  - Se entrega como `RequestCapacityDecision` con target `capacity`.
  - Es una solicitud de decision, no una decision.
  - `ValidateOutboxMessageV0` valida sus campos obligatorios y `run_id`.
  - No incluye valores reales de proveedor, modelo, cuenta, HOME, runtime ni adaptador.
Estado: implementado local en NCW-010.
```
