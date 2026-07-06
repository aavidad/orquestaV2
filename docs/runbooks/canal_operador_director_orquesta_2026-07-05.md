# Canal operador-Director Orquesta 2026-07-05

## Checkpoint goal-first 2026-07-05

- goal_ref: `goal-ref-task-autoprogramming-90e96a780f43-g01`
- objetivo: cablear el canal operador-Director en la composicion real MCP/API
  para status, queue, observe, launch, handoff y bloqueo seguro de stop/control.
- alcance autorizado: stack Codex, MCP, servidor, inventario de bugs y este
  runbook.
- siguiente artefacto: tests focales de wiring real y puertos funcionales.
- evidencia compacta inicial: `evidence-ref-checkpoint-started-operator-director-stack-wiring-20260705`

## Superficie

El canal v0 recibe mensajes de operador desde cualquier adaptador como
`OperatorMessageV0` y los entrega al Director por puertos hexagonales. El
nucleo neutral vive en `modulos/orquesta-operator-director-channel` y no conoce
Hermes, Telegram, filesystem, red real ni runtime.

La herramienta MCP publica es `orquesta.operator.director.message.v0`. El input
minimo usa refs opacas:

- `request_ref`
- `target_ref`
- `body`

Campos opcionales como `message_ref`, `conversation_ref`, `adapter_ref`,
`sender_ref`, `intent` y `evidence_refs` permiten a Telegram, Hermes u otro
adaptador conservar correlacion sin publicar internos.

## Flujo

1. El adaptador externo normaliza su evento a `OperatorMessageV0`.
2. MCP/API llama `orquesta.operator.director.message.v0`.
3. `OperatorDirectorChannelServiceV0` valida refs opacas y cuerpo.
4. El puerto `OperatorDirectorDispatchPortV0` entrega el mensaje al Director o
   al puente temporal configurado.
5. El puerto `OperatorDirectorExchangeStorePortV0` persiste el exchange
   mensaje/respuesta antes de devolver el ACK compacto.

Hermes queda como puente temporal en `cmd/orquesta-server`: el bridge convierte
el mensaje a `OperatorDirectedQueryV0` usando el conector operador existente.
Esa decision es composicion, no dependencia del modulo neutral.

El stack real cablea `OperatorDirectorMessage` siempre con un store de exchange
de composicion. En `orquesta-app-codex-stack`, el dispatcher de composicion usa
los executors MCP ya inyectados para responder:

- `status`: consulta `orquesta.director.stats.v0`.
- `queue`: consulta `orquesta.run_queue.priority.v0` en modo `rank`.
- `observe`: observa un `run_ref` por `orquesta.apps.observe_director_goal.v0`.
- `launch` y `handoff`: bloquean de forma segura si el mensaje libre no trae el
  payload estructurado requerido por `prepare_run` o `arrancar_director`.
- `stop` y `control`: solo llaman `orquesta.runs.control.v0` si hay
  confirmacion explicita y `run_ref`; si falta confirmacion o puerto real,
  devuelven `blocked`.

El puente temporal de `cmd/orquesta-server` conserva la compatibilidad Hermes:
si se inyecta `OperatorMCPDirectedQueryPortV0`, entrega por ese puerto publico.

## Garantias v0

- Las refs se tratan como opacas y no se aceptan paths, URLs ni secretos.
- El ACK devuelve `ack_ref`, `message_ref`, estado, refs de evidencia y una
  respuesta compacta.
- Sin puerto de dispatch o sin store durable, la herramienta devuelve error
  compacto y no finge cierre.
- Los adaptadores deben aportar un store durable real en produccion cuando se
  sustituya el puente temporal por un adaptador productivo. El store en memoria
  de `cmd/orquesta-server` es auxiliar de composicion/test y conserva el
  contrato de puerto sin meter filesystem en el modulo neutral.

## Pruebas focales

```bash
GOFLAGS=-buildvcs=false go test -count=1 ./modulos/orquesta-operator-mcp ./modulos/orquesta-mcp ./cmd/orquesta-server -run TestOperatorDirector
GOFLAGS=-buildvcs=false go test -count=1 ./modulos/orquesta-operator-mcp ./modulos/orquesta-mcp ./cmd/orquesta-server -run TestOperatorMessage
GOFLAGS=-buildvcs=false go test -count=1 ./modulos/orquesta-operator-mcp ./modulos/orquesta-mcp ./cmd/orquesta-server -run TestMCPOperator
git diff --check
```

Evidencia focal goal-first 2026-07-05:

```bash
TMPDIR=.orquesta-runtime/tmp GOCACHE=.orquesta-runtime/go-cache GOFLAGS=-buildvcs=false go test -count=1 ./modulos/orquesta-app-codex-stack ./modulos/orquesta-mcp ./cmd/orquesta-server -run 'TestBuildStack.*OperatorDirector|TestOperatorDirector|TestMCPOperator'
git diff --check
```
