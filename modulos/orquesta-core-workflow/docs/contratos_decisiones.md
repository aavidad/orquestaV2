# Contratos de decisiones v0

## `AcceptDecision`

```text
Nombre: AcceptDecision
Tipo: comando
Version: v0
Propietario: orquesta-core-workflow
Consumidores: entradas futuras del workflow durable
Campos:
  - decision_ref
  - phase_id
  - vote_ref
  - accepted_option_ref
  - summary
  - evidence_refs opcional
Invariantes:
  - Requiere run activo.
  - `phase_id` debe ser `votacion_y_decision` y estar como fase actual activa.
  - `vote_ref` debe existir ya en `OrchestrationRunV0.Votes`.
  - Si `decision_ref` ya esta proyectada, el comando debe coincidir con la huella `CommandEffects` original.
  - No decide proveedor, modelo, HOME, OAuth, cuenta, adaptador ni secreto.
  - No emite outbox ni abre la siguiente fase.
Errores:
  - payload_invalido
  - fase_no_soportada
  - transicion_invalida
  - detalle_prohibido
Estado: implementado local en NCW-015; identidad durable endurecida en NCW-057.
```

## `ArchitectureDecisionAccepted`

```text
Nombre: ArchitectureDecisionAccepted
Tipo: evento
Version: v0
Propietario: orquesta-core-workflow
Consumidores: reducer, replay durable, proyecciones futuras
Campos:
  - decision_ref
  - phase_id
  - vote_ref
  - accepted_option_ref
  - summary
  - evidence_refs opcional
Invariantes:
  - Evento compacto y append-only.
  - Solo pertenece a la fase `votacion_y_decision`.
  - `vote_ref` debe existir ya en la proyeccion del run antes de aplicar el evento.
  - `ApplyEventV0` proyecta `decision_ref` en `decisions` sin duplicar.
  - Proyecta una huella `CommandEffects` por `(ArchitectureDecisionAccepted, decision_ref)`.
  - Rechaza otra decision durable con la misma `decision_ref` y distinta key, event_id, causation_id o payload.
  - No contiene actas completas, prompts, transcripts ni detalles de runtime.
Errores:
  - evento_invalido
  - payload_invalido
  - secuencia_invalida
  - detalle_prohibido
Estado: implementado local en NCW-015; identidad durable endurecida en NCW-057.
```
