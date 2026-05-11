# Contratos de fases v0

## `ClosePhase`

```text
Nombre: ClosePhase
Tipo: comando
Version: v0
Propietario: orquesta-core-workflow
Consumidores: entradas futuras del workflow durable
Campos:
  - phase_id
  - closure_ref
  - summary
  - evidence_refs opcional
Invariantes:
  - Requiere run activo.
  - `phase_id` debe existir y ser la fase actual activa.
  - `closure_ref` y `summary` son obligatorios.
  - `evidence_refs` son refs compactas, normalizadas y sin duplicados.
  - No emite outbox ni ejecuta efectos externos.
Errores:
  - payload_invalido
  - fase_no_soportada
  - transicion_invalida
  - detalle_prohibido
Estado: implementado local en NCW-012.
```

## `PhaseClosed`

```text
Nombre: PhaseClosed
Tipo: evento
Version: v0
Propietario: orquesta-core-workflow
Consumidores: reducer, replay durable, proyecciones futuras
Campos:
  - phase_id
  - closure_ref
  - summary
  - evidence_refs opcional
Invariantes:
  - Se aplica solo sobre la fase actual.
  - Marca la fase como `cerrada`.
  - Conserva `opened_at`.
  - Asigna `closed_at` con `occurred_at` del evento.
  - Actualiza `last_event_id` y `last_sequence`.
  - Replay durable acepta duplicado exacto por huella estable y rechaza conflicto.
Errores:
  - evento_invalido
  - payload_invalido
  - secuencia_invalida
  - detalle_prohibido
Estado: implementado local en NCW-012.
```
