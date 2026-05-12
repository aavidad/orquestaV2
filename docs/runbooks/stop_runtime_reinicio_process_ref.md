# Runbook: stop tras reinicio del runtime en memoria

Fecha: 2026-05-13.

## Causa

`StopRuntimeAgent` resuelve `run_id + agent_request_id` en el registro durable
de procesos, pero `ProcessRuntimeConnectorV0` gobierna procesos desde un mapa en
memoria. Si el servidor se reinicia, el registro puede seguir conteniendo el
`process_ref` antiguo mientras el conector nuevo no conserva el handle del
proceso ni un mecanismo durable para reatarlo.

En ese estado, `ProcessAgentStopperV0` no puede demostrar que el proceso
correspondiente fue parado. Confirmar `AgentStopConfirmed` usando solo el
`process_ref` registrado seria un falso positivo.

## Decision

No se confirma el stop cuando el runtime actual devuelve
`process_runtime_no_encontrado` para un `process_ref` registrado. El mensaje de
outbox debe permanecer sin ACK hasta que exista un conector productivo capaz de
hacer una de estas operaciones de forma verificable:

- reatar el `process_ref` a un proceso vivo mediante estado durable propio;
- verificar que el proceso ya no existe con una identidad durable suficiente;
- enviar una parada real por un supervisor externo idempotente.

## Frontera necesaria

La recuperacion completa requiere un puerto/conector durable de runtime o de
supervisor de procesos. Ese puerto debe vivir fuera del nucleo de orquestacion y
debe devolver evidencia de parada verificable antes de que el executor registre
`AgentStopConfirmed`.

Mientras no exista ese puerto, la respuesta operativa correcta es tratar el
`StopRuntimeAgent` como no confirmado, no como completado.

Si el ledger deja el mensaje en `claimed` tras un fallo de dispatch, la
recuperacion pertenece al contrato de leases/reintentos del outbox: expirar o
liberar el claim fallido para reintentar con un stopper capaz de verificar la
parada. No debe resolverse registrando un ACK de stop sin evidencia.
