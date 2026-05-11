# Contratos de brainstorming v0

## `RequestBrainstorm`

```text
Nombre: RequestBrainstorm
Tipo: comando
Version: v0
Propietario: orquesta-core-workflow
Consumidores: entradas futuras del workflow durable
Campos:
  - brainstorm_request_id
  - phase_id
  - topic_ref
  - summary
  - minimum_recommended_capacity opcional
  - evidence_refs opcional
Invariantes:
  - Requiere run activo.
  - `phase_id` debe ser `brainstorming_arquitectura` y estar como fase actual activa.
  - No emite outbox ni ejecuta agentes.
  - No decide proveedor, modelo, HOME, OAuth, cuenta, adaptador ni secreto.
  - `minimum_recommended_capacity` es recomendacion minima, no seleccion de modelo.
  - Si `brainstorm_request_id` ya esta reflejado, el retry solo es idempotente si coincide la huella durable de comando y payload normalizado.
Errores:
  - payload_invalido
  - fase_no_soportada
  - transicion_invalida
  - detalle_prohibido
Estado: implementado local en NCW-013.
```

## `BrainstormRequested`

```text
Nombre: BrainstormRequested
Tipo: evento
Version: v0
Propietario: orquesta-core-workflow
Consumidores: reducer, replay durable, proyecciones futuras
Campos:
  - brainstorm_request_id
  - phase_id
  - topic_ref
  - summary
  - minimum_recommended_capacity opcional
  - evidence_refs opcional
Invariantes:
  - Evento compacto y append-only.
  - Solo pertenece a la fase `brainstorming_arquitectura`.
  - `ApplyEventV0` proyecta `brainstorm_request_id` en `brainstorms` sin duplicar.
  - `ApplyEventV0` registra/verifica `CommandEffects` para impedir que la misma ref oculte otro evento, idempotency key, command_id o payload.
  - No contiene prompts, transcripts, runtime real, proveedor, modelo, HOME ni secretos.
Errores:
  - evento_invalido
  - payload_invalido
  - secuencia_invalida
  - detalle_prohibido
Estado: implementado local en NCW-013; identidad fuerte extendida en NCW-058.
```
