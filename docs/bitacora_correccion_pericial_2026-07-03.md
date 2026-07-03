<!--
Bitácora de ejecución del plan de corrección pericial.
Regla: cada agente añade filas; nunca borra ni reescribe filas ajenas.
Manual: docs/manual_correccion_pericial_orquesta_2026-07-03.md
Informe: docs/informe_pericial_claude_orquesta_2026-07-03.md
-->

# Bitácora — corrección pericial Orquesta

## Estado de tareas

| Tarea | Estado | Agente | Claim | Evidencia de cierre |
| --- | --- | --- | --- | --- |
| T-PER-101 | libre | — | — | — |
| T-PER-102 | libre (dep: 101) | — | — | — |
| T-PER-103 | libre (dep: 102) | — | — | — |
| T-PER-104 | libre (dep: 102) | — | — | — |
| T-PER-105 | libre (dep: 102) | — | — | — |
| T-PER-106 | libre (dep: 103-105) | — | — | — |
| T-PER-201 | libre | — | — | — |
| T-PER-202 | libre | — | — | — |
| T-PER-203 | libre (dep: 201,202) | — | — | — |
| T-PER-301 | BLOQUEADA-POR-WRITE-SET | — | — | — |
| T-PER-302 | bloqueada (dep: 301) | — | — | — |
| T-PER-401 | BLOQUEADA-POR-WRITE-SET | — | — | — |
| T-PER-402 | bloqueada (dep: 401) | — | — | — |
| T-PER-501 | en curso (pilotaje Orquesta) | claude-fable-5 | 2026-07-03 | — |
| T-PER-502 | libre (dep: 501) | — | — | — |
| T-PER-601 | libre | — | — | — |
| T-PER-701 | libre | — | — | — |

## Registro cronológico

### 2026-07-03 — claude-fable-5 (sesión de revisión pericial)

- Creados informe pericial y manual de corrección (ver cabecera).
- Verificado: `go build ./...` limpio; tests de frontera raíz verdes;
  130 bugs únicos en inventario (30-06 a 02-07), 11 abiertos.
- Métricas de línea base medidas: env_vars_orquesta=500,
  endpoints_status=15, interfaces_estado=64, modulos_director=17.
- Write-set ajeno vigente (agente local del informe preliminar) sobre
  `cmd/orquesta-server/codex_goal_app_server*.go` y smoke goal-first:
  T-PER-301/302/401/402 bloqueadas hasta integración.
- Decisión de pilotaje: T-PER-501 como primera tarea vía Orquesta
  (write-set mínimo, solo `scripts/`), siguiendo manual sección 8.
- Pilotaje paso 1: preflight barato del entorno goal-first (sin cuota).
  Resultado: pendiente de anotar debajo.

<!-- Añadir entradas nuevas debajo de esta línea, más reciente al final. -->

### 2026-07-03 (tarde) — claude-fable-5: pilotaje real ejecutado, 3 bugs nuevos encontrados, 1 arreglado

**El pilotaje funcionó como prueba de fuego: Orquesta NO pudo lanzar el goal
al primer intento y la causa fue un bug real del núcleo.** Cadena completa:

1. Preflight goal-first: `ok` (backend `app_server_tmux`, codex wrapper Node).
2. Servidor pilotado en perfil self-programming aislado (worktree
   `pericial/pilot-t501`, backlog mínimo con solo T265 = T-PER-501).
3. El planner idle SÍ eligió T265, pero el goal quedó `invalid`.

**BUG NUEVO A (arreglado en local): `codex_app_server_write_set_prepare_failed`.**
`codexAppServerWriteSetLooksLikeFileV0` solo trataba `.md/.markdown` como
fichero (cierre incompleto de BUG-ORQ-20260630-036). Cualquier write-set con
ficheros de código existentes (`modulos/orquesta-server/config_v0.go`…)
hacía `MkdirAll` sobre un fichero → el launch fallaba SIEMPRE. Fix aplicado en
`cmd/orquesta-server/codex_goal_app_server_v0.go`: (a) stat previo — si el
target existe como fichero se salta; (b) cualquier extensión marca fichero,
con `/` final forzando directorio. Tests nuevos en
`cmd/orquesta-server/codex_goal_write_set_prepare_v0_test.go`:
`TestPrepareCodexGoalWriteSetV0NoFallaConFicheroExistenteV0`,
`TestPrepareCodexGoalWriteSetV0NoCreaDirectorioParaRutaConExtensionV0`,
`TestCodexAppServerWriteSetLooksLikeFileV0TrataExtensionesComoFicheroV0`.
Verificado: focales verdes + `go test -count=1 ./` verde.
PENDIENTE: fila en `docs/inventario_bugs_orquesta_2026-06-30.md` (no añadida
por write-set ajeno vigente sobre ese fichero).

**BUG NUEVO B (abierto): el receipt `invalid` de launch no conserva causa.**
El `launch_receipt` persistido solo trae `issues[].code` sin mensaje ni
detalle; el operador tiene que reconstruir la causa con auditoría+código
fuente. Además `autoprogramming/status` lo disfraza como
`goal_backend_state_unreconciled` y `observe` como
`codex_goal_observation_rejected`, cuando la verdad era "launch falló en
prepare". Encaja en P1/T-PER-101: la fase debería ser `lanzado-fallido` con
la causa de primera clase.

**BUG NUEVO C (abierto): contrato de backlog poco visible.** El parser de
secciones de backlog solo honra las etiquetas `Objetivo:`, `Alcance:` (=
write-set), `Criterios:`, `Tests:`, `Estado:`, `Dependencias:`
(`cmd/orquesta-server/idle_self_improvement_backlog_parser_v0.go`). Una
sección con "Write-set previsto:" (formato usado en
`modulos/*/docs/tareas.md`) se acepta pero cae al write-set enlatado del área
por defecto sin aviso. Recomendación: diagnosticar "section_scope_missing" o
unificar etiquetas.

**Evidencia extra para T-PER-401**: `POST /server/shutdown` con `forced=true`
respondió `backend_still_running`, `shutdown_ready=false`, y el proceso
servidor no salió (hubo que matarlo con SIGTERM); la sesión tmux del backend
y el app-server quedaron vivos y se limpiaron a mano. Exactamente la clase de
bug del contrato de shutdown en dos fases.

**Detalle menor**: `/api/v0/autoprogramming/status` y `/goal/observe` solo
aceptan POST; un GET devuelve `metodo_no_permitido`. Para operadores/scripts
conviene alias GET (mismo criterio que BUG-047 con `/health`).

**Estado del pilotaje al escribir esto**: servidor relanzado con el fix
(SHA `f03b525c…`), backlog T265 reescrito con etiquetas correctas, monitor
armado. El resultado del goal queda en el worktree
`pericial/pilot-t501` (rama ya existente en el repo) y en
`/tmp/claude-1000/-home-alberto-Trabajo-orquesta/8be66426-b9e9-428a-a040-c4ee62477495/scratchpad/pilot-t501/`
(state/, logs/, runtime/ — volátil, no sobrevive reinicio).

### Receta completa para repetir el pilotaje (para agentes Codex)

1. Worktree: `git worktree add <dir>/worktree -b pericial/pilot-<tarea>` (o
   reutilizar la rama `pericial/pilot-t501`).
2. Backlog mínimo: sustituir en el worktree
   `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md` por un doc con
   SOLO la tarea a pilotar, en formato de sección `## Txxx slug` con
   etiquetas `Objetivo:` (una línea larga), `Estado: pendiente.`,
   `Alcance:` (lista = write-set), `Criterios:` (lista), `Tests:` (lista).
   Ejemplo vivo: rama `pericial/pilot-t501`, sección T265.
3. Arranque (equivalente a `scripts/smoke_goal_first_app_server_real.sh` pero
   con automejora idle activa): compilar `./cmd/orquesta-server`, exportar:
   `ORQUESTA_SERVER_ADDR=127.0.0.1:0`, `ORQUESTA_SERVER_STATE_DIR=<aislado>`,
   `ORQUESTA_SERVER_SELF_PROGRAMMING_ONLY=true`,
   `ORQUESTA_SERVER_SELF_PROGRAMMING_ROOT=<raíz aislada>`,
   `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_ENABLED=false`,
   `ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED=false`,
   `ORQUESTA_CODEX_PROJECT_WORKDIR=<worktree>`,
   `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_PROJECT_WORKDIR=<worktree>`,
   `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_DISABLED=false`,
   `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS=5`,
   `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_MAX_REQUESTS=1`,
   `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_TARGET_QUEUE=1`,
   `ORQUESTA_SERVER_MAX_RUNS_PER_TICK=1`,
   `ORQUESTA_SERVER_MAX_EXECUTIONS_PER_TICK=1`,
   `ORQUESTA_CODEX_RUNTIME_WORKDIR=<aislado>`,
   `ORQUESTA_CODEX_COMMAND=$(command -v codex)`,
   `ORQUESTA_CODEX_CODE_HOME=$HOME/.codex` (si tiene auth.json+config.toml),
   `ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux`,
   `ORQUESTA_CODEX_GOAL_TIMEOUT_MS=900000`,
   `ORQUESTA_CODEX_APPROVAL_POLICY=never`,
   `ORQUESTA_CODEX_SANDBOX=workspace-write`,
   `ORQUESTA_OPES_BASE_URL=""`; lanzar `orquesta-server run` en background.
4. Descubrir addr en `<state>/orquesta_server_state_v0.json` campo `addr`;
   esperar readiness `GET /api/v0/server/readiness`.
5. Observar con `POST /api/v0/autoprogramming/status` (body `{}`) y
   `POST /api/v0/autoprogramming/goal/observe` (body `{"run_ref":"..."}`).
6. Diagnóstico si falla: estado durable del goal en
   `<state>/orchestration-state/app_director_goal_states/*.json`
   (campo `state.launch_receipt.issues`), auditoría en
   `<state>/audit/orquesta_server_audit_v0.jsonl`, goals del backend en
   `<runtime>/goal-srv/codex-home/goals_1.sqlite` (tabla `thread_goals`),
   pane tmux `orquesta-goal-*`.
7. Al terminar: revisar diff del worktree, ejecutar los Tests de la sección,
   y solo integrar a la rama de trabajo tras revisión. Shutdown con
   `POST /api/v0/server/shutdown` body
   `{"forced":true,"reason":"...","idempotency_key":"..."}` y verificar que
   el PID sale (si no sale: evidencia extra para T-PER-401; matar y anotar).
