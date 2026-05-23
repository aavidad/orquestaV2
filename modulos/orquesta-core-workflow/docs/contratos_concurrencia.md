# Contratos de concurrencia: orquesta-core-workflow

Este archivo documenta solo la promocion durable dentro del workflow. La
politica pura vive en `../orquesta-core-concurrency`.

## `RecordConcurrencyGate`

```text
Nombre: RecordConcurrencyGate
Tipo: comando
Version: v0 candidato local
Propietario: orquesta-core-workflow
Consumidores: director futuro, adaptadores inbound futuros
Payload:
  - run_ref
  - gate_ref
  - plan_ref
  - subject_claim_refs opcional
  - ready_claim_refs opcional
  - blocked_claim_refs opcional
  - conflict_refs opcional
  - decision: allow_request_agent | block_request_agent | ask_director
  - summary
  - evidence_refs opcional
Salida:
  - ConcurrencyGateRecorded
  - outbox vacio
Invariantes:
  - Handler puro: no evalua scopes, no lee filesystem, no consulta Git y no crea scheduler.
  - `run_ref` debe coincidir con el run activo.
  - `allow_request_agent` exige que todos los sujetos esten en `ready_claim_refs`.
  - `block_request_agent` exige que al menos un sujeto este en `blocked_claim_refs`.
  - `ask_director` cubre sujeto vacio o desconocido; no se infiere estado.
  - La identidad fuerte se registra por `gate_ref` en `CommandEffects`.
  - Un retry exacto ya reflejado es no-op solo si coinciden command_id, idempotency_key y payload normalizado.
  - La misma ref con decision/plan distinta, command_id distinto o idempotency_key distinta se rechaza.
  - Payload compacto, con refs opacas de adaptador/ejecucion permitidas y corte fuerte solo para secretos o credenciales. Esta apertura queda pendiente de revision futura en `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`.
Errores:
  - payload_invalido
  - detalle_prohibido
  - transicion_invalida
Estado: implementado local en NCW-047; identidad fuerte extendida en NCW-063.
```

## `ConcurrencyGateRecorded`

```text
Nombre: ConcurrencyGateRecorded
Tipo: evento
Version: v0 candidato local
Propietario: orquesta-core-workflow
Consumidores: reducer, replay durable, director futuro, observability futura
Payload:
  - run_ref
  - gate_ref
  - plan_ref
  - subject_claim_refs opcional
  - ready_claim_refs opcional
  - blocked_claim_refs opcional
  - conflict_refs opcional
  - decision
  - summary
  - evidence_refs opcional
Invariantes:
  - Evento compacto y append-only.
  - Proyecta una ref compacta en `concurrency_gates`.
  - Registra/verifica `CommandEffects` por `gate_ref`; replay rechaza la misma ref con otro event_id, causation_id, idempotency_key o payload normalizado.
  - No lanza agentes, no bloquea el run completo y no genera outbox.
  - El director decide comandos posteriores: RequestAgent, AskDirector o replan.
  - No contiene secretos ni credenciales. Puede transportar refs opacas de adaptador, ejecucion o pruebas para que el director pueda normalizar/reparar sin cortar el ciclo. Esta apertura queda pendiente de revision futura.
Estado: implementado local en NCW-047; identidad fuerte extendida en NCW-063.
```
