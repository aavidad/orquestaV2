# Contratos de votaciones v0

## `RequestVote`

```text
Nombre: RequestVote
Tipo: comando
Version: v0
Propietario: orquesta-core-workflow
Consumidores: entradas futuras del workflow durable
Campos:
  - vote_request_id
  - phase_id
  - decision_topic_ref
  - brainstorm_ref
  - summary
  - minimum_recommended_capacity opcional
  - evidence_refs opcional
Invariantes:
  - Requiere run activo.
  - `phase_id` debe ser `votacion_y_decision` y estar como fase actual activa.
  - Requiere `brainstorm_ref` para no votar sin opciones previas.
  - No acepta ni decide proveedor, modelo, HOME, OAuth, cuenta, adaptador ni secreto.
  - No emite outbox; pedir agentes o capacidad se hace con comandos separados.
  - Si `vote_request_id` ya esta reflejado, el retry solo es idempotente si coincide la huella durable de comando y payload normalizado.
Errores:
  - payload_invalido
  - fase_no_soportada
  - transicion_invalida
  - detalle_prohibido
Estado: implementado local en NCW-014.
```

## `VoteRequested`

```text
Nombre: VoteRequested
Tipo: evento
Version: v0
Propietario: orquesta-core-workflow
Consumidores: reducer, replay durable, proyecciones futuras
Campos:
  - vote_request_id
  - phase_id
  - decision_topic_ref
  - brainstorm_ref
  - summary
  - minimum_recommended_capacity opcional
  - evidence_refs opcional
Invariantes:
  - Evento compacto y append-only.
  - Solo pertenece a la fase `votacion_y_decision`.
  - `ApplyEventV0` proyecta `vote_request_id` en `votes` sin duplicar.
  - `ApplyEventV0` registra/verifica `CommandEffects` para impedir que la misma ref oculte otro evento, idempotency key, command_id o payload.
  - No representa la decision aceptada; eso queda para `AcceptDecision`.
Errores:
  - evento_invalido
  - payload_invalido
  - secuencia_invalida
  - detalle_prohibido
Estado: implementado local en NCW-014; identidad fuerte extendida en NCW-058.
```
