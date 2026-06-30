# Incidencia OPES Wait Unhandled Outbox QA Continuacion Tractorista

Fecha: 2026-06-22.

## Contexto

Run OPES original:
`run-opes-tractorista-qa-texto-tests-continuacion-20260622`.

Run de verificacion posterior:
`run-opes-tractorista-qa-restantes-20260622-v2`.

Objetivo: QA de los temas 004, 006 y 008 tras quedar pendiente en una run QA
anterior.

## Sintomas

1. `prepare-run` acepto la run y la cola la marco como `running`.
2. No se creo ningun directorio runtime de agente.
3. `autoprogramming/status` mostro 3 tareas `pending`, 0 agentes y
   `recommended_action=supervise:queue`.
4. `autoprogramming/supervise` y `/runs/supervise` devolvieron
   `stop_reason=max_ticks`, `last.status=waiting_outbox` y evidencia
   `wait_unhandled_outbox`.
5. Diagnostico:
   `run_supervisor_queue_pressure queue_ref=global total=7 executable=1 ready=0
   running=1 delivered=0 stopped=3 closed=3`.

## Impacto

- La run queda en cola como activa sin despachar agentes.
- Repetir supervisiones no cambia el estado.
- El operador humano debe parar/recrear la run o arreglar el dispatcher.

## Causa Raiz Confirmada

La run `run-opes-tractorista-qa-restantes-20260622-v2` reprodujo el mismo patron
con una tercera solicitud de capacidad:

- existia el outbox `RequestCapacityDecision` de `g03`;
- el ledger persistente lo tenia con claim activo y sin ACK;
- la proyeccion de la run no tenia la `capacity_request` correspondiente;
- `ListPendingOutboxV0` no devolvia el mensaje porque oculta claims recientes;
- el supervisor no podia reconstruir la fase y repetia `waiting_outbox`.

Ademas, en el caso parcial la idempotency original podia estar ya en el indice de
eventos aunque la proyeccion no hubiera absorbido el evento. Reemitir con la
misma idempotency no reparaba la run.

## Arreglo Aplicado

Commit pendiente en la rama `trabajo/plataforma-agentes`:

- `drainRunAttemptControlV0` llama ahora a
  `reconcileOrphanCapacityOutboxForRunV0` antes de la recuperacion normal de
  outbox.
- El reconciliador consulta `OutboxLedger.ListPending` por `run_ref` y puerto
  `capacity`, de forma que ve tambien claims persistentes recientes que el
  dispatcher oculta.
- Si la `capacity_request` falta, reconstruye el comando con idempotency nueva
  `idem-reconcile-capacity-outbox-*` y libera el claim antiguo.
- Si la solicitud y la decision ya estan en la run, marca el outbox antiguo como
  `dispatched` con evidencia de supersesion para que no bloquee mas drenajes.

## Pruebas

- `go test -count=1 ./modulos/orquesta-app-codex-stack`.
- `go test -count=1 ./modulos/orquesta-mcp`.
- `go test -count=1 ./modulos/orquesta-persistence ./modulos/orquesta-orchestration-core`.
  Nota 2026-07-01: el paquete historico `./modulos/orquesta-state-file/outbox`
  fue retirado por `ARCH-ORQ-20260630-004`; el ledger file-based vivo queda en
  `orquesta-persistence`.

Casos nuevos:

- reconstruccion de `capacity_request` desde outbox huerfano y liberacion de
  claim;
- no reutilizacion de idempotency original cuando habia persistencia parcial;
- lectura de claims persistentes recientes mediante `ListPending`;
- ACK de outbox supersedido cuando la decision ya existe.

## Humo Real OPES

La run `run-opes-tractorista-qa-restantes-20260622-v2` termino con:

- `status=cerrada`;
- 3 tareas;
- 3 agentes lanzados;
- 3 ACK entregados;
- 3 revisiones aceptadas;
- 3 tareas cerradas.

Evidencias de curso:

- `temas/tema_004/09_validacion/informe_qa_texto_tests_tema.md`;
- `temas/tema_006/09_validacion/informe_qa_texto_tests_tema.md`;
- `temas/tema_008/09_validacion/informe_qa_texto_tests_tema.md`.

El outbox de la run quedo sin claims vivos para capacidad ni lanzamiento.

## Tareas Tecnicas Restantes

- `OUTBOX-TASK-001`: mejorar el diagnostico publico de `wait_unhandled_outbox`
  con el tipo de outbox, id de mensaje y estado claim/ack.
- `OUTBOX-TASK-003`: asegurar que una run nueva no hereda presion de cola de
  runs paradas/rotas del mismo proyecto.

## Criterio De Cierre

El bloqueo concreto de outbox de capacidad reclamado queda cerrado si el commit
incluye los tests anteriores y una run real equivalente acaba con agentes
lanzados, ACKs absorbidos, tareas cerradas y outbox sin claims vivos. La mejora
del diagnostico publico sigue abierta como tarea separada.
