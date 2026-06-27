# Autoprogramacion por MCP - 2026-05-23

## Alcance

Este runbook valida la frontera MCP para autoprogramacion y trabajo externo.
MCP queda como adaptador inbound opt-in sobre puertos publicos: no abre runtime,
no lee persistencia productiva y no conoce Codex, OPES, DB, HOME, OAuth,
credenciales, proveedor ni modelo.

Write-set del corte:

- `modulos/orquesta-mcp`
- `modulos/orquesta-operator-mcp`
- `modulos/orquesta-operator-mcp-client`
- `cmd/orquesta-server`
- `docs/runbooks/autoprogramacion_mcp_2026-05-23.md`
- `docs/runbooks/smoke_mcp_real_transport_opt_in_2026-05-24.md`

## Contrato publico

Tools MCP relevantes ya publicados por `RegisterMCPTransportV0`:

- `orquesta.autoprogramming.prepare_run.v0`: por executor inyectado prepara una
  run continuable legacy con `run_ref`, `workflow_task_refs`, `wait_agent_refs`
  y request `continue`, o un handoff Goal-first con `goal_specs[]` sin
  `run_ref` legacy.
- `orquesta.autoprogramming.observe_goal.v0`: observa un `GoalWorkStateV0` por
  `run_ref`, valida cierre por puerto y sincroniza cola terminal. Para `run_ref`
  goal-first, `orquesta.runs.supervisor.v0` no drena legacy y debe recomendar
  `observe_goal`.
- `orquesta.autoprogramming.self_improvement.propose.v0`: transforma un fallo
  observado por director/agente en `AutoprogrammingRequestV0` de segundo plano,
  con `priority_score` bajo y `prepare_run` listo para el paso siguiente. Si
  `auto_prepare_run=true` y hay executor inyectado, tambien devuelve
  `prepared_run`; si falta el puerto conserva la propuesta y publica accion de
  reparacion.
- `orquesta.director.human_work.review_plan.v0`: convierte una orden humana
  amplia en plan revisable y, si procede, en request de prepare-run sin saltarse
  el review. Si `raise_operator_question=true`, eleva una consulta dirigida por
  puerto MCP de operador; Hermes/OpenClaw entran solo como conectores MCP
  externos, no como coupling de core.
- `orquesta.runs.supervisor.v0`: supervisa una run concreta o una cola
  inyectada con limites acotados; en goal-first actua como compatibilidad y
  redirige a `observe_goal`.
- `orquesta.director.stats.v0`: consulta stats compactas y contexto de decision
  por refs opacas.
- `orquesta.observability.workspace_timeline.query.v0`: consulta timeline de
  workspace por puerto inyectado. `time_window` acepta la forma canonica
  `{ "preset": "last_30m" }` y, por compatibilidad de operador, el alias string
  `"last_30m"`; ambos se normalizan antes de validar. Entradas de forma invalida
  devuelven error publico saneado y no exponen rutas locales ni material crudo.
- `orquesta.server.shutdown.v0`: coordina apagado por caso de uso inyectado,
  RunControl, supervisor y stats.
- `orquesta.domain_work.v0`: crea jobs de dominio o entrega artefactos por
  puertos `DomainWork` inyectados, sin nombrar OPES ni otro producto.
- `orquesta.external_work.run.v0`: crea una run neutral para un trabajo externo
  ya especificado.

Resources MCP relevantes:

- `orquesta.contracts.shared.v0`
- `orquesta.operational.status.v0`
- `orquesta.core_workflow.contracts.v0`
- `orquesta.operator.operations.v0`
- `orquesta-operator-mcp-client`: conector externo opt-in para operadores que
  exponen tools MCP compatibles; implementa `OperatorMCPConnectorV0` sin
  importar `orquesta-mcp` ni transporte local.

## Fronteras

- Los tools MCP delegan en ejecutores o puertos publicos; si falta binding,
  devuelven error publico `mcp_transport_tool_unbound`.
- `orquesta-operator-mcp` define capacidades operativas por refs opacas y
  conectores; no expone tablas, event-store, procesos, PID, prompts ni
  transcripts.
- El conector de operador real queda fuera de `orquesta-mcp`: HTTP/MCP reciben
  solo DTOs publicos, y `orquesta-operator-mcp-client` delega en un cliente MCP
  generico configurado por nombres de tool y refs opacas.
- `domain_work` y `external_work/run` transportan refs opacas y contratos de
  dominio. El dominio externo conserva datos, validadores, persistencia y
  ensamblado.
- OPES, Codex, MCPO, red productiva, filesystem productivo y runtime real son
  adaptadores de composicion opt-in fuera de estos modulos. El smoke MCP real
  disponible en `cmd/orquesta-server` solo usa JSON-RPC HTTP temporal en
  loopback y no queda habilitado por defecto.
- `operational_director.task_source: director_decision` pertenece a la cadena
  causal del Director Operativo y no se materializa desde MCP como contrato de
  producto.

## Validacion del corte

Ejecutar desde la raiz del repo:

```bash
go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp ./modulos/orquesta-operator-mcp-client
```

Smoke opt-in del transporte MCP real de composicion:

```bash
ORQUESTA_MCP_REAL_SMOKE_CONFIRM=1 \
GOCACHE=/tmp/orquesta-go-build-cache \
go run ./cmd/orquesta-server mcp-real-smoke
```

Criterios de aceptacion manual:

- `RegisterMCPTransportV0` registra resources compactos y los tools listados.
- `prepare_run`, supervisor, stats, shutdown, `domain_work` y
  `external_work/run` quedan opt-in por puerto o executor inyectado.
- `self_improvement` solo prepara run en cola cuando `auto_prepare_run` y el
  executor de prepare-run estan inyectados; sin puerto devuelve reparacion
  publica sin perder la propuesta.
- `human_work.review_plan` conserva el plan si falta el conector de consulta
  dirigida y devuelve `operator_mcp_port_unavailable` como reparacion publica.
- El cliente MCP externo preserva errores publicos de operador, reduce fallos
  opacos a `operator_mcp_port_error` y permite sustituir Hermes/OpenClaw u otro
  operador sin acoplar el nucleo.
- Las respuestas usan refs opacas, errores publicos y payloads compactos.
- No hay imports de OPES, Codex runtime concreto, DB, HOME, OAuth, credenciales,
  proveedor ni modelo dentro de `orquesta-mcp` ni `orquesta-operator-mcp`.
- La ruta de dominio externo conserva arquitectura hexagonal: Orquesta coordina;
  la app propietaria valida y ensambla.
- El smoke MCP real temporal prueba una lectura de resource, una propuesta de
  automejora no destructiva y el error publico
  `operator_mcp_port_unavailable` cuando no hay operador inyectado.

## Resultado 2026-05-24 T16

Se anade transporte JSON-RPC HTTP opt-in en `cmd/orquesta-server` para cerrar el
smoke temporal de servidor MCP real sin mover logica a MCP. El handler registra
resources/tools por `RegisterMCPTransportV0`; el comando `mcp-real-smoke` exige
`ORQUESTA_MCP_REAL_SMOKE_CONFIRM=1`, arranca loopback temporal y valida:

- lectura de `orquesta.operator.operations.v0`;
- `orquesta.autoprogramming.self_improvement.propose.v0` con propuesta no
  destructiva y refs opacas preservadas;
- `orquesta.operator.status.query.v0` sin operador real, normalizado como
  `operator_mcp_port_unavailable`.

Runbook focal: `docs/runbooks/smoke_mcp_real_transport_opt_in_2026-05-24.md`.

## Resultado 2026-05-23

Validado con la bateria focal obligatoria del paquete:

- `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp`
- `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp ./modulos/orquesta-operator-mcp-client`
- `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./modulos/orquesta-app-codex-stack`
- `validar criterios de aceptacion del cambio`
- `validar contrato externo de dominio`

## Revision/rework request-ref-automejora-mcp-operador-004

El rework acotado revisa solo la frontera MCP/HTTP de operador humano y
automejora dentro del write-set:

- `POST /api/v0/autoprogramming/status` delega en cola y stats por puertos
  inyectados; si faltan puertos conserva diagnostico publico reparable.
- `POST /api/v0/autoprogramming/goal/observe` es la ruta normal para observar y
  cerrar trabajos `goal_first`.
- `POST /api/v0/autoprogramming/supervise` queda como compatibilidad
  legacy/resident cuando no exista `GoalWorkStateV0`; sin `run_ref` deja al
  executor decidir cola residente, sin abrir runtime desde MCP/HTTP y sin
  empujar runs `goal_first` al loop historico.
- `orquesta.autoprogramming.self_improvement.propose.v0` conserva evidencia de
  fallo, genera automejora secundaria de baja prioridad y solo llama
  `prepare-run` cuando `auto_prepare_run=true` y el executor existe.
- `orquesta.director.human_work.review_plan.v0` conserva plan revisable aunque
  falte puente humano; la consulta a operador es opt-in por
  `OperatorMCPDirectedQueryPortV0`.
- `orquesta.operator.operations.v0` publica capabilities compactas y el
  transporte registra status, burst supervisado, outbox y consulta dirigida por
  puertos o conector agregado, nunca por internals.
- `operator_advice` en HTTP es observacion no bloqueante: se normalizan aliases
  reparables de refs, accion y texto; no decide proveedor, runtime, cola ni
  cierre.

Refs opacas preservadas en la entrega: `task-ref-self-improvement-18eca7b1eae7`,
`worktree-ref-orquesta-automejora-mcp-operador-004` y
`trabajo-plataforma-agentes`. No se convierten en rutas ni nombres Git.

Prueba focal obligatoria reejecutada:

```bash
go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp ./modulos/orquesta-operator-mcp-client
```

Revalidacion `task-autoprogramming-18eca7b1eae7-g01`: sin cambios de codigo en
la frontera tras revisar contrato y pruebas locales. La entrega conserva como
refs opacas `task-ref-self-improvement-18eca7b1eae7`,
`worktree-ref-orquesta-automejora-mcp-operador-004` y
`trabajo-plataforma-agentes`; no se usan como rutas ni nombres Git.

Rework focal `agent-ref-task-autoprogramming-18eca7b1eae7-g01-562352fb086c`:
se refuerza la cobertura de contrato sin ampliar la frontera. Las pruebas
locales verifican que `operator_advice` se conserva como observacion no
bloqueante en `status` y `supervise`, incluso con puerto ausente, y que
`self_improvement` lo inyecta como contexto/regla compacta de automejora sin
decidir runtime, cola, proveedor ni cierre. Revalidado con:

```bash
go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp ./modulos/orquesta-operator-mcp-client
```

Revalidacion `agent-ref-task-autoprogramming-18eca7b1eae7-g01-dd735cd48a5e`:
se reviso de nuevo el conector MCP/HTTP de operador humano y automejora contra
los contratos locales. No se amplio la frontera: `status`, `supervise`,
`self_improvement`, `human_work.review_plan`, `operator.operations` y el cliente
MCP externo siguen como adaptadores finos sobre puertos inyectados, refs opacas
y errores publicos. El transporte real MCP, operadores concretos, red
productiva y runtime real permanecen como composicion opt-in fuera del nucleo.
Revalidado con:

```bash
go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp ./modulos/orquesta-operator-mcp-client
```

Revalidacion `agent-ref-task-autoprogramming-18eca7b1eae7-g01-ca965c319ed5`:
se revisa de nuevo la frontera MCP/HTTP de operador humano/automejora tras
rework no aceptado. El contrato se mantiene acotado: `status` y `supervise`
delegan en puertos inyectados, `operator_advice` sigue como observacion no
bloqueante, `self_improvement` conserva evidencia y solo auto-prepara con
opt-in, `human_work.review_plan` conserva el plan aunque falte operador y
`operator.operations`/cliente MCP externo siguen usando refs opacas y errores
publicos. Las refs `task-ref-self-improvement-18eca7b1eae7`,
`worktree-ref-orquesta-automejora-mcp-operador-004` y
`trabajo-plataforma-agentes` se preservan como refs opacas.

Revalidado con:

```bash
go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp ./modulos/orquesta-operator-mcp-client
```

Revalidacion `agent-ref-task-autoprogramming-18eca7b1eae7-g01-a84f653220c2`:
se reviso la entrega rechazada contra el paquete de rework y los contratos
locales. El cierre queda como contrato ya implementado y probado dentro del
write-set: HTTP `status` y `supervise` conservan `operator_advice` como
observacion no bloqueante, `self_improvement` preserva evidencia y solo
auto-prepara por puerto inyectado, `human_work.review_plan` conserva plan aunque
falte operador, `operator.operations` delega por puertos/ref opacas y el cliente
MCP externo reduce fallos no publicos a `operator_mcp_port_error`. No se abre
servidor MCP real, red productiva ni runtime concreto desde estos modulos.
Refs opacas preservadas: `task-ref-self-improvement-18eca7b1eae7`,
`worktree-ref-orquesta-automejora-mcp-operador-004` y
`trabajo-plataforma-agentes`.

Revalidado con:

```bash
go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp ./modulos/orquesta-operator-mcp-client
```
