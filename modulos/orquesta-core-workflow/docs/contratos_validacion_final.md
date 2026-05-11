# Contratos de validacion final v0

## `RegisterFinalValidation`

```text
Nombre: RegisterFinalValidation
Tipo: comando
Version: v0
Propietario: orquesta-core-workflow
Consumidores: entradas futuras del workflow durable
Campos:
  - validation_ref
  - phase_id
  - closed_task_ref
  - summary
  - evidence_refs opcional
Invariantes:
  - Requiere run activo.
  - `phase_id` debe ser `validacion_final` y estar como fase actual activa.
  - `closed_task_ref` debe existir ya en `OrchestrationRunV0.ClosedTasks`.
  - Si `validation_ref` ya existe en `OrchestrationRunV0.Validations`, devuelve no-op solo si coincide la huella durable de comando y payload normalizado.
  - No emite outbox.
  - No cierra fase, no cierra run, no abre otra fase y no ejecuta conectores.
  - No contiene codigo, diffs, commits, transcripts, rutas locales, runtime, DB, proveedor, modelo, OAuth, HOME ni secretos salvo como prohibiciones.
Errores:
  - payload_invalido
  - fase_no_soportada
  - transicion_invalida
  - detalle_prohibido
Estado: implementado local en NCW-024; identidad fuerte extendida en NCW-061.
```

## `FinalValidationRegistered`

```text
Nombre: FinalValidationRegistered
Tipo: evento
Version: v0
Propietario: orquesta-core-workflow
Consumidores: reducer, replay durable, observability futura
Campos:
  - validation_ref
  - phase_id
  - closed_task_ref
  - summary
  - evidence_refs opcional
Invariantes:
  - Evento compacto y append-only.
  - Solo pertenece a la fase `validacion_final`.
  - `closed_task_ref` debe existir ya en la proyeccion `closed_tasks` antes de aplicar.
  - `ApplyEventV0` proyecta `validation_ref` en `validations` sin duplicar y registra/verifica `CommandEffects`.
  - No marca la fase como cerrada, no cambia `current_phase`, no cambia el estado del run y no emite outbox.
  - No contiene resultado completo de validacion, implementacion, prompts, transcripts ni detalles de conectores.
Errores:
  - evento_invalido
  - payload_invalido
  - secuencia_invalida
  - detalle_prohibido
Estado: implementado local en NCW-024; identidad fuerte extendida en NCW-061.
```
