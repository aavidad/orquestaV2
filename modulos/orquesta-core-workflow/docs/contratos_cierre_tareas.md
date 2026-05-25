# Contratos de cierre de tareas v0

## `CloseTask`

```text
Nombre: CloseTask
Tipo: comando
Version: v0
Propietario: orquesta-core-workflow
Consumidores: entradas futuras del workflow durable
Campos:
  - task_id
  - phase_id
  - delivery_ref
  - accepted_review_ref
  - summary
  - evidence_refs opcional
Invariantes:
  - Requiere run activo.
  - `phase_id` debe ser `revision` y estar como fase actual activa.
  - `task_id` debe existir ya en `OrchestrationRunV0.Tasks`.
  - `delivery_ref` debe existir ya en `OrchestrationRunV0.Deliveries`.
  - `accepted_review_ref` debe existir ya en `OrchestrationRunV0.AcceptedReviews`.
  - Si `task_id` ya existe en `OrchestrationRunV0.ClosedTasks`, devuelve no-op solo si coincide la huella durable de comando y payload normalizado.
  - No emite outbox.
  - No cierra fase, no cierra run, no abre otra fase y no ejecuta conectores.
  - No contiene codigo, diffs, commits, transcripts crudos, rutas locales ni valores reales de runtime, DB, proveedor, modelo, OAuth, HOME o secretos.
Errores:
  - payload_invalido
  - fase_no_soportada
  - transicion_invalida
  - detalle_prohibido
Estado: implementado local en NCW-023; identidad fuerte extendida en NCW-061.
```

## `TaskClosed`

```text
Nombre: TaskClosed
Tipo: evento
Version: v0
Propietario: orquesta-core-workflow
Consumidores: reducer, replay durable, observability futura
Campos:
  - task_id
  - phase_id
  - delivery_ref
  - accepted_review_ref
  - summary
  - evidence_refs opcional
Invariantes:
  - Evento compacto y append-only.
  - Solo pertenece a la fase `revision`.
  - `task_id`, `delivery_ref` y `accepted_review_ref` deben existir ya en la proyeccion del run antes de aplicar.
  - `ApplyEventV0` proyecta `task_id` en `closed_tasks` sin duplicar y registra/verifica `CommandEffects`.
  - No marca la fase como cerrada, no cambia `current_phase`, no cambia el estado del run y no emite outbox.
  - No contiene resultado completo de revision, implementacion, prompts, transcripts ni detalles de conectores.
Errores:
  - evento_invalido
  - payload_invalido
  - secuencia_invalida
  - detalle_prohibido
Estado: implementado local en NCW-023; identidad fuerte extendida en NCW-061.
```
