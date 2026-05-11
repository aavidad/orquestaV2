# Contratos de funcion v0

## `PublishFunctionContract`

```text
Nombre: PublishFunctionContract
Tipo: comando
Version: v0
Propietario: orquesta-core-workflow
Consumidores: entradas futuras del workflow durable
Campos:
  - contract_ref
  - phase_id
  - decision_ref
  - summary
  - function_names opcional
  - evidence_refs opcional
Invariantes:
  - Requiere run activo.
  - `phase_id` debe ser `planificacion_microtareas` y estar como fase actual activa.
  - `decision_ref` debe existir ya en `OrchestrationRunV0.Decisions`.
  - Si `contract_ref` ya esta proyectada, el comando debe coincidir con la huella `CommandEffects` original.
  - No decide runtime, DB, proveedor, modelo, HOME, OAuth, Docker, tmux, filesystem ni secretos.
  - No emite outbox, no crea microtareas y no ejecuta codigo.
Errores:
  - payload_invalido
  - fase_no_soportada
  - transicion_invalida
  - detalle_prohibido
Estado: implementado local en NCW-016; identidad durable endurecida en NCW-057.
```

## `FunctionContractPublished`

```text
Nombre: FunctionContractPublished
Tipo: evento
Version: v0
Propietario: orquesta-core-workflow
Consumidores: reducer, replay durable, proyecciones futuras
Campos:
  - contract_ref
  - phase_id
  - decision_ref
  - summary
  - function_names opcional
  - evidence_refs opcional
Invariantes:
  - Evento compacto y append-only.
  - Solo pertenece a la fase `planificacion_microtareas`.
  - `decision_ref` debe existir ya en la proyeccion del run antes de aplicar el evento.
  - `ApplyEventV0` proyecta `contract_ref` en `function_contracts` sin duplicar.
  - Proyecta una huella `CommandEffects` por `(FunctionContractPublished, contract_ref)`.
  - Rechaza otro contrato durable con la misma `contract_ref` y distinta key, event_id, causation_id o payload.
  - No contiene implementacion, prompts, transcripts ni detalles de runtime/adaptadores.
Errores:
  - evento_invalido
  - payload_invalido
  - secuencia_invalida
  - detalle_prohibido
Estado: implementado local en NCW-016; identidad durable endurecida en NCW-057.
```
