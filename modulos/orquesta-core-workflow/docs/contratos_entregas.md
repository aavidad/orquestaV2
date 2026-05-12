# Contratos de entregas v0

## `RegisterDelivery`

```text
Nombre: RegisterDelivery
Tipo: comando
Version: v0
Propietario: orquesta-core-workflow
Consumidores: entradas futuras del workflow durable
Campos:
  - delivery_ref
  - phase_id
  - task_id
  - agent_ref
  - summary
  - evidence_refs opcional
Invariantes:
  - Requiere run activo.
  - `phase_id` debe ser `programacion` y estar como fase actual activa.
  - `task_id` debe existir ya en `OrchestrationRunV0.Tasks`.
  - `agent_ref` debe existir ya en `OrchestrationRunV0.Agents`.
  - `agent_ref` debe existir en `started_agents`; `AgentRequested` no basta para entregar.
  - `agent_ref` no puede estar en `failed_agents`.
  - `agent_ref` no puede estar en `stopped_agents`; una parada solicitada corta entregas posteriores.
  - Si `delivery_ref` ya esta reflejado, el retry solo es no-op si coincide la huella durable de comando y payload normalizado.
  - No contiene codigo, commits, rutas locales, runtime, DB, proveedor, modelo, HOME, OAuth, Docker, tmux ni secretos.
  - No emite outbox ni solicita revision.
Errores:
  - payload_invalido
  - fase_no_soportada
  - transicion_invalida
  - detalle_prohibido
Estado: implementado local en NCW-019, endurecido en NCW-050/NCW-051 e identidad fuerte extendida en NCW-062.
```

## `DeliveryRegistered`

```text
Nombre: DeliveryRegistered
Tipo: evento
Version: v0
Propietario: orquesta-core-workflow
Consumidores: reducer, replay durable, revision futura
Campos:
  - delivery_ref
  - phase_id
  - task_id
  - agent_ref
  - summary
  - evidence_refs opcional
Invariantes:
  - Evento compacto y append-only.
  - Solo pertenece a la fase `programacion`.
  - `task_id` y `agent_ref` deben existir ya en la proyeccion del run antes de aplicar.
  - `agent_ref` debe tener `AgentStarted` previo.
  - `agent_ref` no puede tener `AgentFailed` previo.
  - `agent_ref` no puede estar en `stopped_agents`.
  - `ApplyEventV0` proyecta `delivery_ref` en `deliveries`, `task_id` en `delivered_tasks` y `agent_ref` en `delivered_agents` sin duplicar.
  - La proyeccion `delivered_agents` es la fuente canonica para stats de agentes completados por entrega; no se infiere desde nombres de ACK.
  - Registra/verifica `CommandEffects`.
  - No contiene implementacion, diff, transcripts ni detalles de runtime/adaptadores.
Errores:
  - evento_invalido
  - payload_invalido
  - secuencia_invalida
  - detalle_prohibido
Estado: implementado local en NCW-019, endurecido en NCW-050/NCW-051 e identidad fuerte extendida en NCW-062.
```
