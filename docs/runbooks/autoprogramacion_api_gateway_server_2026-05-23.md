# Autoprogramacion API gateway y servidor - 2026-05-23

## Alcance

Este runbook valida el contrato externo de autoprogramacion servido por el
gateway HTTP y el servidor residente. No define producto nuevo ni mete logica de
nucleo en HTTP: las rutas solo transportan requests hacia puertos inyectados.

Write-set del corte:

- `modulos/orquesta-http-gateway`
- `modulos/orquesta-app-gateway`
- `modulos/orquesta-server`
- `cmd/orquesta-server`

## Contrato publico

Rutas REST estables expuestas por el gateway:

- `POST /api/v0/autoprogramming/validate-request`
- `POST /api/v0/autoprogramming/self-improvement`
- `POST /api/v0/autoprogramming/prepare-run` (Goal-first normal; salida legacy
  solo con opt-in de composicion y `director_execution_mode=legacy_director_loop`)
- `POST /api/v0/autoprogramming/goal/observe`
- `POST /api/v0/autoprogramming/status`
- `POST /api/v0/autoprogramming/supervise` (compatibilidad legacy/diagnostico;
  no avanzar runs `goal_first`; legacy solo con opt-in de composicion y
  `director_execution_mode=legacy_director_loop`)
- `POST /api/v0/director/human-work/review-plan`
- `POST /api/v0/apps/director`
- `POST /api/v0/apps/{app_ref}/changes`
- `POST /api/v0/director/stats`
- `POST /api/v0/runs/control`
- `POST /api/v0/runs/queue/priority`
- `POST /api/v0/server/shutdown`
- `POST /api/v0/domain-work`
- `POST /api/v0/external-work/run` (Goal-first normal; fallback legacy solo con
  opt-in de composicion y `director_execution_mode=legacy_director_loop`)

Rutas web compatibles:

- `GET|POST /nueva-app`
- `GET|POST /app-change`
- `GET /director-stats`
- `GET|POST /run-control`
- `GET /run-queue`

Rutas consumidas por web/CLI para autoprogramacion:

- preparar run: `POST /api/v0/autoprogramming/prepare-run`;
- observar goal-first: `POST /api/v0/autoprogramming/goal/observe`;
- listar cola/runs: `POST /api/v0/runs/queue/priority`;
- estado operativo compacto: `POST /api/v0/autoprogramming/status`;
- detalle de run: `POST /api/v0/director/stats`;
- pulso supervisado legacy/resident: `POST /api/v0/autoprogramming/supervise`;
- control seguro: `POST /api/v0/runs/control`.

Todas transportan refs opacas. `worktree_ref` y `branch_ref` son identificadores
contractuales de la composicion, no rutas locales ni nombres Git.

Rutas propias del servidor residente:

- `GET /healthz` solo como liveness.
- `GET /api/v0/server/readiness` antes de preparar runs, drenar OPES, lanzar
  Codex o ejecutar automejora.
- `GET /api/status`
- `GET /api/v0/server/status`

## Automejora idle residente

El servidor puede preparar automejoras de baja prioridad cuando el supervisor
global queda sin ejecuciones durante una ventana configurable. La logica vive
tras un puerto del servidor y el stack Codex la traduce a
`/api/v0/autoprogramming/self-improvement` con `auto_prepare_run=true`.

Desde 2026-06-26, con `ORQUESTA_CODEX_GOAL_BACKEND` configurado, la automejora
residente deriva goal-first por defecto salvo false explicito. `prepare-run`
puede devolver goal/run_ref lanzado; la continuacion publica es
`POST /api/v0/autoprogramming/goal/observe`. `supervise` es compatibilidad
legacy/resident y debe devolver `observe_required` para runs goal-first.

Configuracion:

- `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS`: segundos de idle antes
  de proponer automejora. Por defecto son `60`; `0` desactiva el rail.
- `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_PROJECT_REF`,
  `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_WORKTREE_REF` y
  `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_BRANCH_REF`: refs opacas; no son rutas
  ni nombres Git.
- `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_WRITE_SET`,
  `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_REQUIRED_TESTS`,
  `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_ACCEPTANCE`,
  `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_COMPACT_RULES`,
  `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_CONTEXT_REFS` y
  `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_EVIDENCE_REFS`: listas separadas por
  coma para acotar la tarea generada.

Guardas:

- si el tick del supervisor tuvo ejecuciones, no se prepara automejora idle;
- si ya se intento automejora para la ventana idle vigente, no se duplica;
- preparar automejora corre en segundo plano y no bloquea ticks posteriores;
- si una run de autoprogramacion ya tiene agente pendiente, el supervisor no la
  relanza en ticks globales: conserva espera externa hasta observacion causal.

## Fronteras

- `orquesta-http-gateway` registra solo handlers inyectados; no importa web,
  MCP, runtime, DB, Codex, OPES ni `cmd`.
- `orquesta-app-gateway` compone web + REST/MCP por puertos; no abre sockets,
  no lee env y no crea proveedores reales.
- `orquesta-server` sirve estado residente y delega el resto en el handler de
  aplicacion; supervisor, startup y state store entran por puertos.
- `cmd/orquesta-server` es el borde opt-in para Codex, state-file, domain-work
  file/HTTP, OPES y runner de tests requeridos.
- Shutdown y run-control son opt-in por rutas/puertos; no tocan OPES productivo
  salvo configuracion explicita del bridge.
- El rail idle de automejora usa refs opacas y puertos; staging temporal,
  promocion y archivo quedan en el adaptador opt-in documentado en
  `docs/runbooks/autoprogramacion_staging_promocion_2026-05-24.md`.

## Validacion del corte

Ejecutar desde la raiz del repo:

```bash
go test -count=1 ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway ./modulos/orquesta-server ./cmd/orquesta-server
```

Criterios de aceptacion manual:

- Gateway sin logica de nucleo ni imports prohibidos.
- `cmd/orquesta-server` queda como wiring fino del servidor real.
- Las rutas web, CLI/MCP y REST usan los mismos contratos publicos.
- `domain-work` y `external-work/run` aceptan refs opacas; el dominio externo
  conserva validadores, persistencia y ensamblado. `external-work/run` no cae al
  loop historico sin modo legacy explicito.
- No reaparecen el control-plane heredado ni helpers locales de DB.

## Revision web/CLI 004

El borde servidor queda alineado con los clientes revisados:

- `prepare-run` crea runs de autoprogramacion continuables y encolables solo en
  la rama legacy con opt-in explicito
  `ORQUESTA_AUTOPROGRAMMING_LEGACY_DIRECTOR_LOOP=1`; cuando el trabajo es
  `goal_ready`, devuelve `goal_spec_summaries[]` sin `run_ref` legacy ni cola.
- `status` agrega cola, run, agentes, diagnosticos y acciones seguras para web
  y CLI sin exponer stores, runtime, proveedor, prompts ni filesystem.
- `apps/director/goal/observe` y `autoprogramming/goal/observe` son las rutas
  normales para avanzar/cerrar trabajo `goal_first`; `director/stats`,
  `runs/control` y `runs/queue/priority` inspeccionan y controlan runs desde
  clientes finos.
- `supervise` queda como compatibilidad legacy/diagnostico para runs no
  migrados; no debe empujar el loop historico sobre runs `goal_first` y debe
  transportar `director_execution_mode=legacy_director_loop` junto al opt-in de
  composicion para empujar un `run_ref` legacy.
- Si el servidor no inyecta un puerto real para alguna ruta, debe devolver error
  publico del contrato; web/CLI no deben reconstruirlo con acceso local.
- Revalidacion `task-autoprogramming-7447aef8a77d-g01`: web y CLI quedan como
  adaptadores del contrato publico del servidor. La revision no mueve logica a
  HTTP, no introduce persistencia nueva y conserva `worktree_ref` y `branch_ref`
  como refs opacas.

## Resultado 2026-05-23

Validado con la bateria focal historica del servidor:

- `go test -count=1 ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway ./modulos/orquesta-server ./cmd/orquesta-server`
- `validar criterios de aceptacion del cambio`
- `validar contrato externo de dominio`

Revalidacion web/CLI 004 ejecutada desde la raiz del repo:

- `go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-cli ./cmd/orquesta-server`

Revalidacion `agent-ref-task-autoprogramming-7447aef8a77d-g01-95364243f6e0`:
el servidor queda como borde de composicion para las rutas consumidas por web y
CLI (`prepare-run`, `status`, `supervise`, `runs/queue/priority`,
`director/stats` y `runs/control`). La prueba obligatoria paso el 2026-05-23;
las refs `worktree_ref` y `branch_ref` siguen siendo opacas y no se convierten
en rutas ni nombres Git en los clientes.

Revalidacion `agent-ref-task-autoprogramming-7447aef8a77d-g01-6ea621489958`:
la revision de rework mantiene web/CLI como clientes del gateway publico del
servidor, sin mover logica a nucleo ni a adaptadores locales. Las rutas
operativas revisadas son `prepare-run`, `status`, `supervise`,
`runs/queue/priority`, `director/stats` y `runs/control`. Evidencia ejecutada:
`go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-cli ./cmd/orquesta-server`.

Revalidacion `agent-ref-task-autoprogramming-7447aef8a77d-g01-35bb4432e77b`:
el gateway/servidor conserva esas rutas como borde de composicion y web/CLI las
usan como adaptadores finos. No se anade persistencia, runtime local ni
interpretacion de `worktree_ref` o `branch_ref` fuera del contrato.
Evidencia ejecutada: `go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-cli ./cmd/orquesta-server`.

Revalidacion `agent-ref-task-autoprogramming-7447aef8a77d-g01-ff7908424b11`:
el contrato externo queda revisado para listado de cola/runs, detalle por
`director/stats`, `prepare-run`, `status`, `supervise` y control seguro por
`runs/control`. Web/CLI consumen esas rutas sin acceso a stores, runtime,
filesystem, proveedor ni conversion de refs opacas. Evidencia ejecutada:
`go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-cli ./cmd/orquesta-server`.

Revalidacion `agent-ref-task-autoprogramming-1089cc70db2b-g01-5eb20ecf12e9`:
el servidor residente mantiene la automejora idle como rail configurable por
entorno y puerto de composicion. El disparo por defecto es 60 segundos sin
ejecuciones; `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS=0` lo
desactiva. La preparacion usa `self_improvement` con `auto_prepare_run=true` en
segundo plano, sin bloquear ticks posteriores, y el stack no relanza runs de
autoprogramacion que ya tienen agente externo pendiente. Las refs
`worktree-ref-orquesta-automejora-residente-004` y
`trabajo-plataforma-agentes` siguen siendo opacas. Evidencia ejecutada:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-app-codex-stack`.

Revalidacion `agent-ref-task-autoprogramming-1089cc70db2b-g01-cf0b5d995b28`:
se repite el rework con la bateria obligatoria del servidor residente. Se
mantienen las mismas fronteras: idle configurable por entorno y puerto de
composicion, `self_improvement` con `auto_prepare_run=true` en segundo plano,
sin relanzar agentes para runs de autoprogramacion con espera externa pendiente,
y sin convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git. Evidencia
ejecutada:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-app-codex-stack`.
