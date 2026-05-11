# Contratos de quality gates v0

## `RecordQualityGate`

```text
Nombre: RecordQualityGate
Tipo: comando
Version: v0 candidato local
Propietario: orquesta-core-workflow
Consumidores: director futuro, revisores externos, adaptadores inbound futuros
Campos:
  - run_ref
  - gate_ref
  - phase_id
  - subject_ref
  - decision: accepted | rework_required | blocked | ask_director
  - issue_refs opcional
  - summary
  - evidence_refs opcional
Invariantes:
  - Requiere run activo.
  - `run_ref` debe coincidir con el run actual.
  - `phase_id` debe pertenecer al catalogo v0 y ser la fase actual activa.
  - `gate_ref`, `subject_ref`, `decision` y `summary` son obligatorios.
  - `accepted` no exige `issue_refs`.
  - `rework_required`, `blocked` y `ask_director` exigen al menos una `issue_ref`.
  - La identidad fuerte es `gate_ref` en `CommandEffects`.
  - Repetir el mismo gate exacto ya reflejado es no-op solo si coinciden command_id, idempotency_key y payload normalizado.
  - La misma `gate_ref` con decision o sujeto distinto se rechaza como transicion invalida.
  - No emite outbox, no cierra revision, no pide rework, no bloquea el run y no pregunta al director por si mismo.
  - Payload compacto; sin DB, runtime, proveedor, modelo, HOME, OAuth, Git productivo, prompts, transcripts ni secretos.
Errores:
  - payload_invalido
  - transicion_invalida
  - detalle_prohibido
Estado: implementado local en NCW-067.
```

## `QualityGateRecorded`

```text
Nombre: QualityGateRecorded
Tipo: evento
Version: v0 candidato local
Propietario: orquesta-core-workflow
Consumidores: reducer, replay durable, observability futura, revisiones futuras
Payload: igual que `RecordQualityGate`
Proyeccion:
  - `OrchestrationRunV0.QualityGates` guarda refs compactas `gate_ref#decision:<decision>#subject:<subject_ref>`.
Invariantes:
  - Evento compacto y append-only.
  - Solo se aplica con run activo y fase actual activa indicada por `phase_id`.
  - `ApplyEventV0` proyecta `quality_gates` sin duplicar por `gate_ref`.
  - `ValidateOrchestrationRunV0` rechaza `quality_gates` mal formados o duplicados por `gate_ref`.
  - Registra/verifica `CommandEffects` por `gate_ref`; replay rechaza misma ref con otra key, event_id, command_id o payload.
  - No materializa `ReviewResultRecorded`, `RequestRework`, `BlockRun`, `AskDirector` ni outbox.
  - No contiene payload completo de revision, runtime, proveedor, HOME, OAuth, DB, rutas reales, comandos ni secretos.
Errores:
  - evento_invalido
  - payload_invalido
  - secuencia_invalida
  - detalle_prohibido
Estado: implementado local en NCW-067.
```
