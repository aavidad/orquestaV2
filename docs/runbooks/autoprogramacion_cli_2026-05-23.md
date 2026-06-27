# Autoprogramacion CLI - 2026-05-23

## Alcance

Este runbook valida el borde CLI de autoprogramacion. El binario es un adaptador
fino: parsea argumentos, lee JSON de entrada, invoca clientes REST existentes y
emite `CliOutputEnvelopeV0` automatizable. No decide negocio, no persiste estado,
no lee DB, no arranca runtime y no usa fallback local.

Write-set del corte:

- `modulos/orquesta-cli`
- `cmd/orquesta-cli`
- `docs/runbooks/autoprogramacion_cli_2026-05-23.md`

## Comandos cubiertos

- `app spec solicitar`
- `app spec bootstrap`
- `servidor estado`
- `autoprogramacion preparar`
- `autoprogramacion estado ver`
- `autoprogramacion supervisar`
- `autoprogramacion cola listar`
- `autoprogramacion run ver`
- `autoprogramacion run controlar`
- `doctor contratos`
- `contratos funcion listar`
- `contratos funcion ver`
- `contratos funcion registrar` bloqueado por contrato
- `gobernanza catalogo listar`
- `gobernanza catalogo ver`

## Flujo operativo web/CLI

La CLI cubre el recorrido de operador para autoprogramacion sin entrar en el
nucleo ni en runtime local:

1. `servidor estado` comprueba que el residente responde y muestra solo estado
   publico del proceso.
2. `autoprogramacion cola listar` lista runs candidatos desde
   `orquesta.run_queue.priority.v0`.
3. `autoprogramacion estado ver` muestra cola, run seleccionado, agentes,
   progreso compacto, diagnosticos y acciones seguras desde
   `orquesta.autoprogramming.status.v0`.
4. `autoprogramacion preparar` crea una run continuable solo en la rama legacy;
   para trabajo `goal_ready` debe mostrar `goal_specs[]` sin `run_ref` legacy.
   En ambos casos conserva `worktree_ref` y `branch_ref` opacas, write-set y
   tests requeridos.
5. Si preparar devuelve goal/run_ref goal-first, el siguiente paso no es
   supervisar: es observar goal por `POST /api/v0/autoprogramming/goal/observe`.
   Si la CLI aun no tiene comando dedicado, documentarlo como hueco de cliente y
   no recomendar `supervise` para esos runs.
6. `autoprogramacion supervisar` ejecuta un pulso acotado del supervisor por
   API solo para runs legacy/resident; `--max-ticks 1` es el valor seguro para
   operacion manual.
7. `autoprogramacion run ver` consulta detalle de una run por
   `orquesta.director.stats.v0`.
8. `autoprogramacion run controlar` delega `pause|resume|stop|cancel` en
   `orquesta.runs.control.v0`.

El detalle de apps/runs se obtiene por refs opacas (`run_ref`, `app_ref`,
`queue_ref`, `external_job_ref`). La CLI no convierte esas refs en rutas,
ramas Git, nombres de proveedor ni paths locales.

## Contrato externo

La CLI consume solo rutas publicas versionadas:

- `POST /api/v0/apps/spec`
- `POST /api/v0/director/bootstrap/appspec`
- `GET /api/v0/server/status`
- `POST /api/v0/autoprogramming/prepare-run`
- `POST /api/v0/autoprogramming/status`
- `POST /api/v0/autoprogramming/goal/observe`
- `POST /api/v0/autoprogramming/supervise`
- `POST /api/v0/runs/queue/priority`
- `POST /api/v0/director/stats`
- `POST /api/v0/runs/control`
- `POST /api/v0/operational-status/query`
- `POST /api/v0/core/function-contracts/list`
- `POST /api/v0/core/function-contracts/view`
- `POST /api/v0/governance/catalog/query`

Todas las llamadas REST propagan `X-Correlation-ID`, usan JSON y devuelven
envelopes publicos sin cuerpos privados de errores remotos. Las refs son opacas:
la CLI solo transporta `project_ref`, `app_spec_ref`, `function_contract_ref` y
refs equivalentes declaradas por contratos propietarios.

## Configuracion residente visible

El operador configura el servidor residente antes de arrancarlo; la CLI no lee
variables locales ni statefiles como fuente de verdad. Para automejora residente
de segundo plano, las variables relevantes del borde `cmd/orquesta-server` son:

- `ORQUESTA_SERVER_SUPERVISOR_MAX_TICKS`: ticks por pulso residente; default
  seguro `1`.
- `ORQUESTA_SERVER_ALLOW_REPEATED_RUNS=true`: permite repetir runs dentro del
  mismo pulso cuando la composicion lo autoriza; por defecto esta apagado. El
  nombre exacto es `ORQUESTA_SERVER_ALLOW_REPEATED_RUNS`; no hay alias abreviado
  `ORQUESTA_SERVER_ALLOW_REPEAT`.
- `ORQUESTA_SERVER_ALLOW_REPEAT`: alias no valido. Si aparece en scripts,
  documentacion o entorno de operador, corregirlo a
  `ORQUESTA_SERVER_ALLOW_REPEATED_RUNS`; la CLI no debe inferir equivalencia.
- `ORQUESTA_SERVER_MAX_RUNS_PER_TICK`: limite de runs candidatos por pulso.
- `ORQUESTA_SERVER_MAX_EXECUTIONS_PER_TICK`: limite de ejecuciones lanzadas por
  pulso; default operativo `2`.
- `ORQUESTA_CODEX_MAX_BATCH_READY`: agentes Codex `ready` a despachar por
  tanda; es el nombre canonico, no `ORQUESTA_CODEX_MAX_BATCH`.
- `ORQUESTA_CODEX_MAX_CONCURRENCY`: procesos Codex concurrentes; es el nombre
  canonico, no `ORQUESTA_CODEX_MAX_CONCURRENT`.
- `ORQUESTA_CODEX_REASONING_EFFORT`: esfuerzo de razonamiento de agentes Codex;
  es el nombre canonico, no `ORQUESTA_CODEX_REASONING`.
- `ORQUESTA_CODEX_DIRECTOR_WAVE_AGENTS`: agentes padre por ola del Director.
- `ORQUESTA_CODEX_DIRECTOR_MAX_SUBAGENTS_PER_AGENT`: fanout maximo de
  subagentes por agente padre en delegacion recursiva; default operativo `6`.
- `ORQUESTA_SERVER_DRAIN_MAX_BURSTS`: rondas maximas de drain por ejecucion;
  default operativo `4`.
- `ORQUESTA_SERVER_DRAIN_MAX_STEPS`: pasos maximos por ronda de drain; default
  operativo `6`.
- `ORQUESTA_SERVER_DRAIN_MAX_DISPATCHES`: despachos maximos por espera; default
  operativo `4`.
- `ORQUESTA_SERVER_DRAIN_MAX_COMMANDS`: comandos maximos por drain; default
  operativo `20`.
- `ORQUESTA_SERVER_DRAIN_MAX_OUTBOX`: elementos maximos de outbox por ciclo;
  default operativo `4`.
- `ORQUESTA_SERVER_DRAIN_MAX_DECISIONS`: ciclos maximos de decision por drain;
  default operativo `1`.
- `ORQUESTA_SERVER_DRAIN_MAX_EXTERNAL_WAITS`: esperas externas maximas por
  drain; default operativo `1` y maximo normalizado `70`.
- `ORQUESTA_SERVER_TICK_INTERVAL_MS`: intervalo entre pulsos automaticos;
  default operativo `5000`.
- `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS`: segundos sin ejecuciones
  antes de proponer automejora idle; default operativo `60`; valor `0` la
  desactiva.
- `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_PROJECT_REF`: `project_ref` opaco de la
  automejora idle; default `project-ref-orquesta-server`.
- `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_WORKTREE_REF`: `worktree_ref` opaco de
  la automejora idle; default
  `worktree-ref-orquesta-server-idle-self-improvement`.
- `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_BRANCH_REF`: `branch_ref` opaco de la
  automejora idle; default `branch-ref-orquesta-server-idle-self-improvement`.
- `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AREA`: area sugerida; default
  `automejora`.
- `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_WRITE_SET`: lista CSV de write-set para
  tareas idle; default acotado a piezas del servidor residente.
- `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_REQUIRED_TESTS`: lista CSV de pruebas
  requeridas; default `go test -count=1 ./...`.

El panel `/ops` consume `GET /api/v0/server/status` y muestra
`effective_config.settings`. Las ediciones de variables desde Admin son
pendientes de reinicio: no cambian el proceso vivo. Para aplicarlas hay que
pedir cierre ordenado o forzoso desde Admin y relanzar el servidor con esos
valores. El forzoso debe mostrar aviso de posible perdida de datos no
checkpointados.
- `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_CONTEXT_REFS`: lista CSV de refs opacas
  de contexto adicional; default vacio.
- `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_EVIDENCE_REFS`: lista CSV de refs
  opacas de evidencia; default vacio.
- `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_ACCEPTANCE`: lista CSV de criterios de
  aceptacion; default documenta el disparo tras 60s y el apagado con `0`.
- `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_COMPACT_RULES`: lista CSV de reglas
  compactas para agentes; default pide comunicacion compacta y trabajo
  secundario.

Perfil por entorno:

| Entorno | Ajuste recomendado | Uso |
| --- | --- | --- |
| Observacion | `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS=0` y `ORQUESTA_SERVER_ALLOW_REPEATED_RUNS=false` | Auditar servidor, cola y runs sin sembrar automejora idle. |
| Conservador | `ORQUESTA_SERVER_SUPERVISOR_MAX_TICKS=1`, `ORQUESTA_SERVER_MAX_EXECUTIONS_PER_TICK=1`, `ORQUESTA_SERVER_ALLOW_REPEATED_RUNS=false` | Ejecutar pulsos manuales acotados desde CLI. |
| Residente | `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS=60`, `ORQUESTA_SERVER_ALLOW_REPEATED_RUNS=true` y limites `ORQUESTA_SERVER_DRAIN_MAX_*` explicitos | Dejar automejora secundaria gobernada por el servidor residente. |

Estos perfiles son guias de entorno del proceso servidor. La CLI no los aplica
como flags, no lee el entorno local para inferirlos y no convierte `worktree_ref`
ni `branch_ref` en rutas o nombres Git.

Comprobacion por CLI:

```bash
orquesta-cli servidor estado --server-url http://127.0.0.1:8787 --json
orquesta-cli autoprogramacion preparar --server-url http://127.0.0.1:8787 --input prepare-run.json --json
orquesta-cli autoprogramacion estado ver --server-url http://127.0.0.1:8787 --run-ref RUN_REF --json
orquesta-cli autoprogramacion cola listar --server-url http://127.0.0.1:8787 --json
orquesta-cli autoprogramacion supervisar --server-url http://127.0.0.1:8787 --max-ticks 1 --json
orquesta-cli autoprogramacion run ver --server-url http://127.0.0.1:8787 --run-ref RUN_REF --json
orquesta-cli autoprogramacion run controlar --server-url http://127.0.0.1:8787 --run-ref RUN_REF --action pause --json
```

`servidor estado` muestra estado publico, ultimos ticks, ejecuciones, skips y
errores del supervisor cuando el servidor los expone. Si se necesita auditar el
valor efectivo de una variable no expuesta por `/api/v0/server/status`, eso queda
como mejora del servidor, no como lectura local de la CLI.
La CLI debe presentar ese hueco como configuracion residente no visible, no como
valor calculado desde variables locales, statefiles, worktrees ni runtime.

La visibilidad CLI de esta configuracion es observacional: `cola listar` muestra
prioridad y candidatos, `run ver` muestra stats publicas de una run y
`estado ver`/`supervisar` consultan o ejecutan pulsos acotados por API.
`run controlar` solo envia una accion publica al servidor; no cambia variables
residentes ni manipula runtime local. Ningun comando infiere valores efectivos
leyendo variables del entorno local, statefiles o runtime.
Las refs de automejora idle (`project_ref`, `worktree_ref`, `branch_ref`,
`context_refs` y `evidence_refs`) son opacas: el operador las configura como
identificadores contractuales y la CLI no las transforma en rutas, ramas Git ni
nombres de proveedor.

Nota de rework `task-ref-self-improvement-ed850d0c5d8d`: la causa corregida es
documental y generalizable. La variable soportada es
`ORQUESTA_SERVER_ALLOW_REPEATED_RUNS`; `ORQUESTA_SERVER_ALLOW_REPEAT` queda solo
como forma erronea a reparar. La CLI sigue siendo adaptador server-first y la
automejora idle queda como trabajo secundario, no bloqueo de runs principales.

## Fronteras

- `cmd/orquesta-cli` solo delega en `RunOrquestaCLIV0`.
- `modulos/orquesta-cli` contiene parseo tecnico, lectura de JSON y clientes por
  contrato.
- Las reglas de dominio permanecen en `orquesta-factory`, `orquesta-director`,
  `orquesta-observability`, `orquesta-core` y `orquesta-governance`.
- `autoprogramacion run controlar` delega `pause|resume|stop|cancel` en
  `orquesta.runs.control.v0`; no lee RunControl, runtime ni stores locales.
- `registrar FunctionContractV0` conserva el bloqueo publico
  `registrar_function_contract_bloqueado`.

## Revision web/CLI 004

El corte `request-ref-automejora-web-cli-004` queda revisado como cliente fino:

- CLI: `modulos/orquesta-cli` ya expone clientes y handlers para preparar,
  listar cola, ver estado, supervisar, ver detalle y controlar runs por API.
- Web: `modulos/orquesta-web` ya tiene clientes/proyecciones para
  `prepare-run`, `status`, cola, detalle por `/director-stats` y control seguro
  por `/run-control`; cualquier pantalla posterior debe montar esos view-models
  sin leer stores ni runtime.
- Contrato comun: web y CLI comparten endpoints publicos y preservan
  `worktree_ref=worktree-ref-orquesta-automejora-web-cli-004` y
  `branch_ref=trabajo-plataforma-agentes` como refs opacas cuando viajan en la
  peticion de prepare-run.
- Limite vigente: no hay promocion Git/worktree desde CLI/web; promocion,
  archivo y staging temporal pertenecen al adaptador opt-in del servidor.
- Revalidacion `task-autoprogramming-7447aef8a77d-g01`: la entrega existente se
  corrige como revision documental del contrato real; no amplia alcance ni
  convierte `worktree_ref`/`branch_ref` en rutas o nombres Git.
- Revalidacion `agent-ref-task-autoprogramming-7447aef8a77d-g01-ff7908424b11`:
  la CLI queda revisada como cliente fino para `servidor estado`,
  `autoprogramacion preparar`, listado de cola/runs, detalle de run,
  `status`, `supervisar` y `run controlar`. La evidencia ejecutada es
  `go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-cli ./cmd/orquesta-server`.

## Validacion del corte

Ejecutar desde la raiz del repo:

```bash
go test -count=1 ./modulos/orquesta-cli
```

Criterios de aceptacion manual:

- El binario no reintroduce control-plane legacy.
- No hay imports de DB, runtime, HOME, tokens ni proveedores reales en la CLI.
- Los comandos mutantes usan contratos publicos y claves idempotentes cuando el
  contrato las declara.
- Los comandos read-only no mutan estado ni activan reglas.
- La ayuda y errores visibles usan espanol y codigos/i18n estables.
- Los argumentos posicionales sobrantes fallan como `opcion_invalida`.

## Resultado 2026-05-23

Validado con la bateria focal obligatoria de la revalidacion web/CLI:

- `go test -count=1 ./modulos/orquesta-cli`
- `go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-cli ./cmd/orquesta-server`
- `validar criterios de aceptacion del cambio`
- `validar contrato externo de dominio`

Revalidacion `agent-ref-task-autoprogramming-7447aef8a77d-g01-95364243f6e0`:
la CLI queda cerrada como cliente fino de API para listar cola/runs, consultar
detalle, preparar runs, supervisar y controlar acciones seguras. La prueba
obligatoria compartida paso el 2026-05-23 y no se declara acceso local a stores,
runtime, Git/worktrees, HOME, proveedor ni rutas de filesystem.

Revalidacion `agent-ref-task-autoprogramming-7447aef8a77d-g01-6ea621489958`:
se repite la revision tras rechazo previo y no se detecta hueco ejecutable en
CLI. La entrega queda acotada a los contratos publicos ya implementados
(`prepare-run`, `status`, cola, detalle, supervise y control seguro), con
`worktree_ref=worktree-ref-orquesta-automejora-web-cli-004` y
`branch_ref=trabajo-plataforma-agentes` preservados como refs opacas.
Evidencia ejecutada: `go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-cli ./cmd/orquesta-server`.

Revalidacion `agent-ref-task-autoprogramming-7447aef8a77d-g01-35bb4432e77b`:
se confirma la misma frontera CLI tras rework. Los comandos de autoprogramacion
siguen consumiendo API publica, sin fallback local ni conversion de refs opacas
a rutas, ramas Git, HOME, proveedor o stores.
Evidencia ejecutada: `go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-cli ./cmd/orquesta-server`.
