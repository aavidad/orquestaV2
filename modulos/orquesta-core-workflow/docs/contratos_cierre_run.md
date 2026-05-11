# Contratos de cierre de run v0

## `CloseRun`

```text
Nombre: CloseRun
Tipo: comando
Version: v0
Propietario: orquesta-core-workflow
Consumidores: entradas futuras del workflow durable
Campos:
  - closure_ref
  - phase_id
  - validation_ref
  - summary
  - evidence_refs opcional
Invariantes:
  - Requiere run activo.
  - `phase_id` debe ser `cierre` y estar como fase actual activa.
  - `validation_ref` debe existir ya en `OrchestrationRunV0.Validations`.
  - `closure_ref` y `summary` son obligatorios.
  - Si `closure_ref` ya existe en `OrchestrationRunV0.Closures`, devuelve no-op solo si coincide la huella durable de comando y payload normalizado.
  - No emite outbox.
  - No cierra fase automaticamente, no abre otra fase y no ejecuta conectores.
  - La decision local mantiene `ClosePhase` separado: si la regla de producto exige cerrar la fase `cierre`, debe invocarse `ClosePhase` antes o despues como transicion independiente.
  - No contiene codigo, diffs, commits, transcripts, rutas locales, runtime, DB, provider/proveedor, modelo, OAuth, HOME ni secretos salvo como prohibiciones.
Errores:
  - payload_invalido
  - fase_no_soportada
  - transicion_invalida
  - detalle_prohibido
Estado: implementado local en NCW-025; identidad fuerte extendida en NCW-061.
```

## `RunClosed`

```text
Nombre: RunClosed
Tipo: evento
Version: v0
Propietario: orquesta-core-workflow
Consumidores: reducer, replay durable, observability futura
Campos:
  - closure_ref
  - phase_id
  - validation_ref
  - summary
  - evidence_refs opcional
Invariantes:
  - Evento compacto y append-only.
  - Solo pertenece a la fase `cierre`.
  - `validation_ref` debe existir ya en la proyeccion `validations` antes de aplicar.
  - `ApplyEventV0` proyecta `closure_ref` en `closures` sin duplicar y registra/verifica `CommandEffects`.
  - Marca `OrchestrationRunV0.status` como `cerrado`.
  - No marca la fase como cerrada, no cambia `current_phase` y no emite outbox.
  - Replay durable acepta duplicado exacto por huella estable y rechaza conflicto.
  - No contiene resultado completo de cierre, implementacion, prompts, transcripts ni detalles de conectores.
Errores:
  - evento_invalido
  - payload_invalido
  - secuencia_invalida
  - detalle_prohibido
Estado: implementado local en NCW-025; identidad fuerte extendida en NCW-061.
```
