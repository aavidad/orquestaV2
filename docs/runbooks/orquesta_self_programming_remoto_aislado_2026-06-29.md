# Orquesta self-programming remoto aislado - 2026-06-29

Objetivo: ejecutar Orquesta en `berserk@uso.dipgra.cloud` para que se
autoprograme a si misma sin influir en `uso-app`, OPES productivo ni temarios.

## Reglas

- No se sube nada a produccion desde este entorno.
- No se modifican contenedores productivos (`uso-app`, `opes-api`, postgres,
  nginx-proxy ni servicios asociados).
- No se monta `/home/berserk/deploy/opes`, rutas de temarios, backups
  productivos ni `/var/run/docker.sock`.
- No se abren puertos externos. La web queda publicada solo en
  `127.0.0.1:19039` del host y se accede por tunel SSH.
- En el host se usa `sudo` solo para preparar `/srv/orquesta-self` y operar
  Docker. Dentro del contenedor Orquesta debe ejecutarse como `10001:10001`, no
  como root.
- Las pruebas reales de conectores deben usar fakes, temporales o rutas bajo
  `/srv/orquesta-self`; nunca OPES productivo.
- Si mas adelante se hace una prueba de temario, el material generado no se
  borra: queda conservado como artefacto de prueba aislado para revision
  editorial/tecnica y una promocion manual posterior. La prueba no sube nada a
  produccion por si misma.

## Perfil de ejecucion

El contenedor usa:

- `ORQUESTA_SERVER_SELF_PROGRAMMING_ONLY=true`
- `ORQUESTA_SERVER_SELF_PROGRAMMING_ROOT=/workspace`
- `ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux`
- `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_ENABLED=false`
- `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_ARCHIVE_DIR=/workspace/state/autoprogramming-promotion-archive`
- `ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED=false`
- `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_MAX_REQUESTS=1`
- `ORQUESTA_SERVER_MAX_RUNS_PER_TICK=1`
- `ORQUESTA_SERVER_MAX_EXECUTIONS_PER_TICK=1`

Las variables OPES y DomainWork HTTP quedan vacias o desactivadas. Si alguien
las activa en modo self-programming, el servidor debe rechazar el arranque.
La promocion de autoprogramacion puede activarse solo dentro del root aislado:
si `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_ENABLED=true`, el archive dir
debe estar bajo `ORQUESTA_SERVER_SELF_PROGRAMMING_ROOT`. No se promociona nada a
produccion desde este perfil.

El contexto del Codex/Goal residente debe incluir
`deploy/self-programming/goal_context.md`. Ese documento fija que no se usa
`stdio` nunca. `app_server_tmux` es obligatorio y cualquier prueba de OPES, web
o conectores se hace solo con fakes/temporales aislados.

El entorno de agente tambien debe ejecutar
`scripts/bootstrap_agent_tooling.sh --repo /workspace --status` antes de una
sesion larga y el bootstrap equivalente al preparar el contenedor/worktree. Debe
quedar preparado el broker central de contexto; no deben quedar MCPs
`codebase-memory-mcp` directos ni activos en subagentes salvo opt-in explicito
con `--install-direct-mcp`. La comunicacion compacta tipo `caveman`, si existe,
y la norma persistente en `~/.codex/AGENTS.md` siguen siendo obligatorias. Ver
`docs/runbooks/herramientas_agentes_orquesta_2026-06-30.md`.

## Direccion operativa

En remoto manda Orquesta. `orquesta-server` recibe el trabajo por web/API o lo
crea mediante `idle self-improvement`, construye el `GoalWorkSpec` y lo lanza
con Codex Goal por `app_server_tmux` dentro del contenedor. Codex no se arranca
como sesion manual permanente por SSH: es un runtime controlado por Orquesta.

El operador puede abrir un tunel SSH y usar la web/API local. Si hay que
diagnosticar, se puede inspeccionar el contenedor y sus tmux, pero cualquier
trabajo productivo de autoprogramacion vuelve a entrar por Orquesta.

Si falta autenticacion Codex, cuota, tmux o socket del app-server, el entorno
debe quedar bloqueado con evidencia. No se permite degradar a `stdio`,
`app_server_proxy` ni al loop historico salvo prueba legacy aislada y explicita.

## Cola inicial del Goal residente

El Goal residente no debe descubrir el proyecto desde cero. Debe recibir
`deploy/self-programming/goal_context.md` y empezar por estos cierres:

1. Perfil remoto aislado arrancado y verificado: puerto solo loopback, binds
   solo `/srv/orquesta-self`, estado por API y cero OPES/DomainWork productivo.
2. Goal por `app_server_tmux` probado con una tarea real acotada; no se usa backend `stdio` ni `app_server_proxy` en el camino goal-first.
3. Autoprogramacion autosuficiente: incidencias abiertas a `GoalWorkSpec`,
   pruebas focales, evidencia y commit local.
4. Reparacion automatica: fallos de pruebas/smokes convertidos en tareas
   causales de arreglo y relanzados por Orquesta.
5. Web Nueva App cerrada: tooltips, validacion castellana, wizard
   conversacional, modo experto de datos/integraciones, arquitecturas amplias,
   calidad/accesibilidad configurable y manual profundo.
6. Conectores OPES/DomainWork cerrados solo en fake/temporal aislado, con
   artefactos conservados y sin promocion productiva.
7. Documentacion operativa actualizada con evidencia: runbooks, matriz, estado
   actual y arquitectura.

## Evidencia requerida antes de dejarlo programando

1. `go test -count=1 ./cmd/orquesta-server -run 'TestSelfProgrammingOnly'`.
2. `docker compose config` no contiene `/home/berserk/deploy/opes`,
   `/var/run/docker.sock`, `0.0.0.0:19039:` ni rutas de temarios.
3. `docker inspect` confirma usuario `10001:10001`, `CapDrop=["ALL"]`,
   `no-new-privileges:true`, binds solo bajo `/srv/orquesta-self` y puerto
   ligado a `127.0.0.1`.
4. `/api/v0/server/status` muestra `ORQUESTA_SERVER_SELF_PROGRAMMING_ONLY=true`,
   `ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux`, redacta
   `ORQUESTA_SERVER_SELF_PROGRAMMING_ROOT` y no muestra OPES activo.

## Evidencia 2026-06-29: Objective compacto en idle Goal

Incidencia cerrada: la automejora residente llego a preparar un
`GoalWorkSpecV0` con `Objective` mayor de 4000 caracteres y Codex app-server lo
rechazo como `codex_app_server_rpc_error: goal objective must be at most 4000
characters`. El commit `1f9eec3c` compacta el objetivo de
`idle_self_improvement` a 4000 runas maximo y conserva el contexto completo en
refs, criterios, evidencias y marcador `objective_compacted`.

Verificacion focal ejecutada en el contenedor:

```bash
PATH=/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin \
GOTMPDIR=/workspace/runtime/go-test-tmp \
GOCACHE=/workspace/runtime/go-cache \
go test -count=1 ./modulos/orquesta-server \
  -run 'TestRuntimeV0IdleSelfImprovementGoalFirst(CompactaObjectiveLargo|LanzaGoalWorkSpec|SinLauncherNoCaeALegacy)'
```

Resultado: `ok`.

Smoke real aislado por `app_server_tmux`: servidor temporal bajo
`/workspace/runtime/ig2`, OPES vacio, sin Docker, `approval-policy=never`,
`ORQUESTA_CODEX_SANDBOX=danger-full-access` por la misma restriccion de
`workspace-write` observada en el smoke de `/nueva-app`. Se indujo un criterio
base largo para forzar el caso que antes superaba 4000 caracteres.

Estado durable preservado en
`/workspace/runtime/ig2/s/orquesta_server_state_v0.json`:

```text
idle_self_improvement_reason=goal_running;goal_ref=goal-ref-autoprogramming-backlog-t900-idle-goal-objective-compact-smoke-f5a428f1;external_goal_ref=019f1326-d189-70e2-83f2-84dcad3a0f88;status=running
idle_self_improvement_runs=1
idle_self_improvement_ok=1
idle_self_improvement_goal_receipt.status=running
idle_self_improvement_goal_result.status=running
objective_len=4000
objective_compacted=true
objective_too_long=false
goal_rpc_4000_errors=0
prepare_failed_recent=0
recent_errors=0
```

El primer intento manual uso una ruta mas larga
`/workspace/runtime/smokes/orquesta-idle-goal-objective-20260629T113040Z` y
fallo antes de lanzar el goal con `path must be shorter than SUN_LEN`; esa
evidencia no reabre la incidencia de `Objective`, solo fija que los smokes
`app_server_tmux` deben usar una raiz corta para no exceder el limite de socket
Unix.

## Acceso

```bash
ssh -L 19039:127.0.0.1:19039 berserk@uso.dipgra.cloud
```

Navegar a `http://127.0.0.1:19039`.
