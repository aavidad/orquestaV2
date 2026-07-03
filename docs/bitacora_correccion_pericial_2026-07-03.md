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
| T-PER-101 | **hecho vía Orquesta** (goal T266, Codex) | orquesta+codex, supervisa claude-fable-5 | 2026-07-03 | módulo `orquesta-estado-vivo` integrado: 9 tests verdes (los 6 obligatorios + 2 extra), frontera neutral cubierta y verde, `go build ./...` limpio. Matiz: `puertos_v0.go` quedó solo con la cláusula de paquete — T-PER-102 debe definir `FuenteEvidenciaEstadoPortV0` (consumer-side) al crear los adaptadores. Segundo shutdown también dejó residuos (3ª evidencia T-PER-401) |
| T-PER-102 | **hecho vía Orquesta** (goal T267, integrado como `26cab434`) | orquesta+codex, revisa Codex local | 2026-07-03 | adaptadores de evidencia integrados; verificados `go test -count=1 ./modulos/orquesta-estado-vivo`, `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'EvidenciaEstado|AgregadorEvidencia'`, `go test -count=1 ./`, `go build ./...`. Orquesta dejó el goal `blocked` por contrato de reconciliación, pero el código/tests eran válidos |
| T-PER-103 | **hecho por Codex local** | Codex local | 2026-07-03 | `autoprogramming/status` consume `FuenteEvidenciaEstadoPortV0`, mezcla `ConstruirProyeccionCicloVidaV0` en `queue_health`, `stale_running` y `efficiency_summary`, y el stack Codex cablea el agregador real de run/goal/marker/procesos/receipts. Verificado `go test -count=1 ./modulos/orquesta-estado-vivo ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway`, `go test -count=1 ./modulos/orquesta-app-codex-stack`, `go build ./...`, `git diff --check`. |
| T-PER-104 | **hecho por Codex local** | Codex local + subagente read-only | 2026-07-03 | `director/stats` consume `FuenteEvidenciaEstadoPortV0` y el stack Codex le cablea el agregador real; `queue/global-status` conserva una sola verdad vía `autoprogramming/status`. Se corrigió doble conteo de `estado vivo` al reconciliar clases previas de queue/stats antes de aplicar la proyección. Verificado `go test -count=1 ./modulos/orquesta-mcp`, `go test -count=1 ./modulos/orquesta-app-codex-stack`, `go test -count=1 ./...`, `go build ./...`, `git diff --check` |
| T-PER-105 | **hecho por Codex local** | Codex local | 2026-07-03 | `domain-work/status` hereda `estado vivo` desde `autoprogramming/status` sin fuente nueva: `entregado_parcial` ya sale como `partial_artifacts_written`/`review_partial_artifacts`. `observe goal` recibe overlay opt-in de `FuenteEvidenciaEstadoPortV0` en MCP y stack Codex; parcial/conflicto/rework bloquean verde con evidencia, `desconocido` no pisa snapshots existentes. Verificado `go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPDomainWorkStatusHTTPHandlerV0DerivaVidaDesdeProyeccionV0\|TestMCPObserveAppDirectorGoalToolExecutorV0SnapshotDesdeProyeccionV0'`, `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-app-codex-stack`, `go test -count=1 ./...`, `go build ./...` |
| T-PER-106 | **hecho por Codex local** | Codex local | 2026-07-03 | añadido `estado_vivo_boundaries_test.go`: `paquetesStatus` protege `orquesta-mcp`, `orquesta-app-gateway` y `orquesta-app-codex-stack` contra imports directos de stores/registries/runtime delivery de estado fuera de `orquesta-estado-vivo`; allowlist literal solo conserva `orquesta-runtime-codex-delivery` en el stack actual con comentario de ratchet. Verificado `go test -count=1 ./ -run 'TestEstadoVivoStatusSurfacesDoNotImportStateStoresDirectly\|TestNeutralOrchestrationPackagesDoNotImportProductAdapters'`, `go test -count=1 ./...`, `go build ./...`, `git diff --check`; pruebas negativas temporales: quitar la allowlist usada falla y anadir un import prohibido en `orquesta-mcp` falla |
| T-PER-201 | **hecho por Codex local** | Codex local | 2026-07-03 | sunset formal de `legacy_director_loop` documentado en `AGENTS.md` e inventario (`BUG-ORQ-20260703-148`); `StartAppDirectorV0` y `orquesta.apps.arrancar_director.v0` publican `legacy_sunset_notice` cuando se fuerza legacy. `scripts/smoke_opes_reviews_providers_real.sh` exporta `ORQUESTA_AUTOPROGRAMMING_LEGACY_DIRECTOR_LOOP=true` solo para ese harness legacy con comentario de sunset. Verificado `bash -n scripts/smoke_opes_reviews_providers_real.sh`, `go test -count=1 ./modulos/orquesta-app-director-service ./modulos/orquesta-mcp ./cmd/orquesta-server -run 'TestStartAppDirectorV0LegacyLoopPublicaSunsetNoticeV0\|TestMCPArrancarDirectorAppToolExecutorV0UsaServicioCanonico\|TestSmokeOPESReviewsProvidersRealExigeOptInLegacyYWorkdirOPESV0'`, paquetes afectados completos, `go test -count=1 ./...`, `go build ./...`, `git diff --check` |
| T-PER-202 | **hecho vía Orquesta** (goal T268, integrado como `db09e240`) | orquesta+codex, revisa Codex local | 2026-07-03 | espina Director V2 congelada con notas en 5 `AGENTS.md` y `director_v2_freeze_test.go`. Verificado `go test -count=1 ./ -run 'TestDirectorV2Freeze|TestStatusSurfaceBudgetTPer502V0'`, `go test -count=1 ./`, `go build ./...` |
| T-PER-601 | **hecho vía Orquesta** (goal T269, integrado como `b99dffb4`) | orquesta+codex, revisa Codex local | 2026-07-03 | `ARQUITECTURA.md` enlaza informe/manual pericial y corrige persistencia actual a file-based. Verificado grep de enlaces y `go build ./...` tras integrar |
| RELEVO 98% cuota | **cerrado por Codex local** | siguiente agente | 2026-07-03 | Los 3 pilotajes en vuelo de Claude (`T267`, `T268`, `T269`) fueron revisados, verificados e integrados en principal. Servidores/panes de esos pilotajes apagados manualmente tras reproducir `backend_still_running`; el bug de shutdown sigue como evidencia T-PER-401 |
| T-PER-203 | **hecho por Codex local** | Codex local | 2026-07-03 | creado `docs/mapa_generaciones_director_2026-07-03.md` con los 17 módulos `*director*`, generación, estado, consumidores por imports actuales y decisión; `AGENTS.md` lo enlaza en el orden de autoridad documental nivel 2. Declara goal-first como único camino de producción y `legacy_director_loop` en sunset. Verificado recuento de 17 filas, grep de enlace/decisión, `go test -count=1 ./ -run 'TestDirectorV2Freeze\|TestEstadoVivoStatusSurfacesDoNotImportStateStoresDirectly'`, `go build ./...`, `git diff --check` |
| T-PER-301 | AVANCE-PARCIAL | subagente Socrates + Codex local | 2026-07-03 | creado `modulos/orquesta-runtime-codex-appserver` y el wiring principal de `cmd/orquesta-server/codex_goal_backend_env_v0.go` instancia `CommandProtocolV0`, `TmuxBackendV0`, `LazyTmuxProtocolV0` y `GoalBackendV0` desde el módulo nuevo; añadido boundary `TestCodexAppServerAdapterDoesNotImportCmdOrProductStorage`. Tests: `go test -count=1 ./modulos/orquesta-runtime-codex-appserver ./cmd/orquesta-server` y boundary raíz. No se marca cerrada porque siguen duplicados los `cmd/orquesta-server/codex_goal_app_server*.go` legacy y los tests principales viven fuera del módulo; ver `BUG-ORQ-20260703-150`. |
| T-PER-302 | AVANCE-PARCIAL | Codex local | 2026-07-03 | añadido `estado_backend_v0.go` con `EstadoBackendAppServerV0`, `ObservacionBackendV0` y `TransicionBackendV0` pura, más tests de transiciones legales/ilegales. No se marca cerrada porque Ensure/shutdown/cleanup/active-work aún no consumen la máquina de estados ni existe recolector único de observaciones; queda dependiente de cerrar T-PER-301/`BUG-ORQ-20260703-150`. |
| RELEVO CLAUDE ORQUESTADOR | guardado parcial | Codex local | 2026-07-03 | operador ordena cortar todo para que Claude siga como orquestador. Subagentes cerrados, `orquesta-server run` local de prueba parado, remoto sin agentes vivos observados. Relevo detallado en `docs/relevo_claude_orquestador_2026-07-03.md`. Desviacion principal: T-PER-301/T-PER-302 siguen abiertas por doble implementacion `cmd`/modulo y estado backend no consumido. |
| T-PER-401 | **hecho por Codex local** | Codex local | 2026-07-03 | contrato de shutdown en dos fases implementado sobre rama limpia, sin integrar el WIP remoto: `/api/v0/server/shutdown` aumenta respuestas ready con `exit_pending=true` y `pid`, el runtime programa salida forzada por puerto inyectado si no termina tras `ShutdownGracePeriod`, y el smoke espera desaparicion del PID antes de fallar. Tests: `go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-mcp ./modulos/orquesta-server-shutdown ./cmd/orquesta-server`; focal `TestRuntimeV0ShutdownReadyProgramaSalidaForzadaSiNoTerminaV0`; guard script `TestSmokeGoalFirstScript*`. |
| T-PER-402 | **hecho por subagente Bernoulli** | Codex subagente | 2026-07-03 | añadido `scripts/orquesta_smoke_nightly.sh`, guard `scripts/test_orquesta_smoke_nightly.sh` y runbook `docs/runbooks/smoke_nightly_2026-07.md`. Preflight por defecto sin cuota, modo real solo con `ORQUESTA_NIGHTLY_REAL_CONFIRM=1`, JSON diario y bloqueo OPES/productivo. Tests: `bash -n scripts/orquesta_smoke_nightly.sh scripts/test_orquesta_smoke_nightly.sh`, `scripts/test_orquesta_smoke_nightly.sh`, preflight real sin cuota con `ORQUESTA_NIGHTLY_RESULTS_DIR=/tmp/...`. |
| T-PER-501 | **hecho vía Orquesta** (goal T265, Codex) | orquesta+codex, supervisado por claude-fable-5 | 2026-07-03 | scripts integrados y verificados; test `orquesta_metricas_deuda_ok=true`; métricas: env=500, status=16, interfaces=64, director=17 |
| T-PER-502 | **hecho por Codex local** (integrado como `77fa3c0a`) | Codex local | 2026-07-03 | añadido `status_surface_budget_test.go` con allowlist literal de 16 endpoints status/observe/control/readiness. Verificado `go test -count=1 ./ -run TestStatusSurfaceBudgetTPer502V0`, `go test -count=1 ./`, `bash scripts/test_orquesta_metricas_deuda.sh` |
| T-PER-701 | **hecho por Codex local** | Codex local | 2026-07-03 | Nueva App transporta `technical_constraint:language=<x>` y `technical_constraint:framework=<y>` como reglas hard en `GoalWorkSpecV0`; el cierre goal-first usa tabla de manifiestos `go->go.mod`, `python->pyproject.toml/setup.py`, `node->package.json`, `rust->Cargo.toml` y bloquea con `wrong_language_generated`; observe recomienda `replan_with_language_constraint`. Verificado `go test -count=1 ./modulos/orquesta-app-director-service ./modulos/orquesta-mcp -run 'Test(BuildStartAppDirectorGoalWorkSpecV0TransportaConstraintLenguaje\|StartAppDirectorV0GoalFirstLanzaGoalYNoEjecutaLoopLegacy\|ObserveAppDirectorGoalV0BloqueaRunSiStackTecnicoContradiceAppSpec\|CodexStackV0GoalFirst(NoCierraNuevaAppConLenguajeEquivocado\|CierraNuevaAppSinConstraintDeLenguaje)\|ObserveAppDirectorGoalRecommendedActionV0LenguajeEquivocadoReplanConstraint)'`, `go test -count=1 ./modulos/orquesta-app-director-service ./modulos/orquesta-mcp`, `go test -count=1 ./...`, `go build ./...`, `git diff --check` |
| T-PER-801 (router contexto híbrido) | **hecho por Codex local** | Codex local | 2026-07-03 | creado `docs/revision_router_contexto_hibrido_2026-07-03.md`; el diseño citado por el manual no existe ni local ni remoto, se documenta la ausencia y se decide recortar: no crear router paralelo, extender `CodeContextQueryPortV0`/broker central con estrategias internas. Compara LLMLingua/LongLLMLingua, repo-map Aider, tree-sitter y prompt caching; deja backlog `CTX-TASK-801A..D` con `Alcance`, `Criterios`, `Tests`, sin nueva fuente de verdad. Verificado `rg` local/remoto del documento ausente, revision de broker actual y `git diff --check` |
| BUG write-set prepare (pilotaje) | hecho | claude-fable-5 | 2026-07-03 | commit 13d526d1, tests focales + frontera verdes |
| OPS-KANBAN | hecho por Codex local | Codex local + Orquesta observador | 2026-07-03 | panel `/ops/kanban` read-only sobre `autoprogramming/status` y `queue/global-status`, sin store ni fuente de verdad nueva. Verificado con tests focales web/gateway/app-gateway, `go build ./...`, `cmd/orquesta-server` y smoke real contra Orquesta aislado |

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

### 2026-07-03 (cierre del pilotaje) — RESULTADO: T265 completada por Codex vía Orquesta

Segundo intento tras el fix `13d526d1`: **el ciclo completo funcionó**. El
planner idle recogió T265, el goal se lanzó por `app_server_tmux`, Codex
escribió `scripts/orquesta_metricas_deuda.sh` y
`scripts/test_orquesta_metricas_deuda.sh` respetando el `Alcance:` (write-set)
exacto, con calidad alta (contrato exit-2, `--json`, líneas base, `set -euo
pipefail`). Verificado por el supervisor: `bash -n` ok, test propio
`orquesta_metricas_deuda_ok=true`, salida real env=500/status=16/
interfaces=64/director=17 (status 16 vs 15 de la línea base: diferencia de
regex aceptable). Scripts integrados a la rama de trabajo.

**BUG NUEVO D (abierto): goal `complete` sin `GoalWorkResultV0` ni validación
de cierre.** El estado durable quedó `status=complete` con `result` y
`closure_validation` vacíos; `autoprogramming/status` lo proyecta como
`goal_first_blocked`/`attention_required` (correcto, no falso verde) pero
`POST /autoprogramming/goal/observe` devolvió
`autoprogramming_observe_goal_error` genérico en vez del estado parcial.
Consecuencia: un goal funcionalmente terminado queda "bloqueado" para siempre
sin acción clara. Refuerza T-PER-101/105 (el observe debe leer la proyección)
y sugiere que el prompt/packet goal-first de automejora idle debe exigir el
marcador `ORQUESTA_GOAL_RESULT_V0`/fichero result como hace el smoke Nueva App.

**Evidencia T-PER-401 (segunda vez, reproducible)**: shutdown forzado →
`status=backend_still_running`, `shutdown_ready=false`, proceso servidor no
sale (kill manual), sesión tmux y `codex app-server` residuales (kill manual).
Reproducido 2 de 2 veces en el pilotaje.

**Balance del pilotaje**: 1 tarea completada de punta a punta por Orquesta,
4 bugs nuevos encontrados (A arreglado+commiteado, B/C/D abiertos y
documentados), 2 evidencias reproducibles de shutdown para T-PER-401.
El dogfooding funciona: seguir usándolo tarea a tarea con esta receta.

### 2026-07-03 (noche) — claude-fable-5 asume dirección (relevo del operador)

Todos los Codex locales cortados por el operador; ver
`docs/relevo_claude_orquestador_2026-07-03.md`. Dirección actual:

- **T270 en vuelo vía Orquesta** (rama `pericial/pilot-t270`, puerto 35231):
  cerrar BUG-ORQ-20260703-150 / T-PER-301-302 — recolector único de
  observaciones + consumo de `TransicionBackendV0` + migración de tests +
  reducción de los `cmd/orquesta-server/codex_goal_app_server*.go` a fachadas.
- **T272 en vuelo vía Orquesta** (rama `pericial/pilot-t272`, puerto 45849):
  limpieza de raíz — mover TAREA_OPES_*/HANDOFF_* a `docs/historico/2026-06/`
  con índice, quitar binarios *.test, sin borrar contenido OPES.
- Siguiente en cola (NO lanzar en paralelo con T270, cruza `cmd/`):
  T-PER-901 control plane dormido por eventos; después CTX-TASK-801A..D.
- Verificación e integración: según receta de esta bitácora; los Tests de
  cada sección son el criterio. Integrar T270 con revisión especialmente
  cuidadosa (toca el componente más crítico).
- Pendiente de decisión humana: WIP remoto sucio de 71 ficheros en
  `/srv/orquesta-self/worktrees/orquesta` (no integrar completo, triar);
  quién borró `docs/diseno_router_contexto_hibrido_2026-07-03.md`.

### 2026-07-03 (noche) — T272 integrada; T270 sigue en vuelo

T272 (limpieza raíz) terminó `blocked` en Orquesta pero con trabajo completo
y correcto (patrón BUG D otra vez: falta result formal). Verificado e
integrado como `757fef2b`: 29 históricos movidos a `docs/historico/2026-06/`
con INDICE y resúmenes, raíz sin TAREA_OPES_*/HANDOFF_*/*.test, `.gitignore`
cubre `*.test`, build limpio. Instancia T272 apagada (la sesión tmux del
backend de T270 sigue viva, no tocar hasta su terminal).

### 2026-07-03 (noche) — T270 integrada: BUG-ORQ-20260703-150 cerrado

T270 vía Orquesta completó T-PER-301/302: 19 ficheros legacy borrados de
`cmd/`, fachada fina + wiring test únicos restantes,
`recolector_observacion_backend_v0.go` como único punto con
has-session/pane_pid, `TransicionBackendV0` gobernando el ciclo. Verificado
en worktree y en main: módulo + suite completa `cmd` + fronteras verdes.
La integración rompió el guard documental
`TestOperationalDocsRuntimeManualMentionsAreHistoricalOrHarnessV0` por
interacción con T272; arreglado eximiendo `docs/historico/` por ruta.
Matiz de monitoreo: el run de T270 desapareció de `active_runs` al terminar
en vez de quedar terminal visible (variante del BUG D, añadir al inventario).
Nueva regla de contrato en `AGENTS.md`: paralelización por defecto.
Cola pendiente: T-PER-901 (ya sin conflicto de write-set), CTX-TASK-801A..D,
BUG D (result formal de goals idle), filas de inventario para bugs B/C/D.
Instancias de pilotaje apagadas; ramas `pericial/pilot-*` conservan evidencia.

### 2026-07-03 (madrugada) — T273 en vuelo (primer tramo T-PER-901)

Lanzado a Orquesta en rama `pericial/pilot-t901`, puerto 44971: anti-churn
idempotente + despertar por evento + tick como watchdog en supervisor/idle
de `modulos/orquesta-server`. Verificar con los Tests de la sección T273 e
integrar tras revisión (receta de esta bitácora). Tras integrar T273, los
siguientes paralelizables son: BUG D (result formal de goals idle),
CTX-TASK-801A..D y filas de inventario de bugs B/C/D del pilotaje.

### 2026-07-03/04 — dirección con cuota renovada: T273+T274 en paralelo

- T273 (supervisor dormido por eventos) sigue en vuelo con avance real
  (idempotencia, wakeup decorators, checkpoint declarado).
- T274 lanzada en paralelo (rama `pericial/pilot-t274`, puerto 33811): filas
  de inventario para bugs B/C/D del pilotaje. Write-sets disjuntos.
- Timer nightly: unidades systemd de usuario pendientes de activación por el
  operador (bloqueo de permisos correcto del harness); comandos en
  `docs/runbooks/smoke_nightly_2026-07.md` sección systemd.

### 2026-07-03/04 — T273 y T274 integradas: plan T-PER 18/18 completado

T273 integrada como `a9eafe5e` (supervisor idle dormido por eventos,
anti-churn idempotente, tick watchdog; suites server+cmd+fronteras verdes).
T274 integrada como `794b541d` (bugs 151/152/153 inventariados).
Quinta reproducción del bug 153: el goal T273 estaba `complete` con result
formal escrito y Orquesta lo mantenía `running` sin reconciliar.
Instancias y residuos de pilotaje barridos; ramas `pericial/pilot-*`
conservadas como evidencia.

**Cola siguiente (paralelizable, write-sets disjuntos):**
1. Fix BUG-ORQ-20260703-153 (result formal/reconciliación de goals idle) —
   el más valioso: elimina la clase de falso `blocked`/`running` vista en
   5 de 8 pilotajes.
2. Fix BUG-ORQ-20260703-151 (causa de primera clase en launch receipt).
3. Fix BUG-ORQ-20260703-152 (diagnóstico de etiquetas de backlog).
4. CTX-TASK-801A..D (ahorro de tokens del broker, incremental).

**Pendiente de operador:** activar timer nightly (runbook
`docs/runbooks/smoke_nightly_2026-07.md`); triaje del WIP remoto de 71
ficheros; ventana de observación §9 de dos semanas una vez el nightly corra.

### 2026-07-04 — ola paralela de cierre: T275+T276+T278

Tres pilotajes simultáneos (doctrina de paralelización, write-sets
module-disjuntos):
- T275 (puerto 36361, rama `pericial/pilot-t275`): fix BUG-153 —
  reconciliación de goals idle por result materializado, decaimiento con
  causa y terminal visible. El fix de autonomía definitivo.
- T276 (puerto 37149, rama `pericial/pilot-t276`): fix BUG-151 — `Detail`
  saneado en issues de launch receipt hasta las proyecciones.
- T278 (puerto 43787, rama `pericial/pilot-t278`): CTX-TASK-801B —
  `query_kind=repo_map` en el broker central de contexto.
En cola para la ola siguiente (cruzan write-sets con esta): BUG-152
(diagnóstico etiquetas backlog), CTX-TASK-801A (presupuesto de contexto),
801C (prompt estable para cache), 801D (evaluación LLMLingua opt-in).
Verificación/integración: receta de esta bitácora.

### 2026-07-04 — nightly activado, remoto saneado, 4 pilotajes simultáneos

- Timer nightly instalado y activo (autorizado por operador): systemd user
  `orquesta-smoke-nightly.timer`, preflight 03:30, primer disparo esta noche.
- Remoto `uso.dipgra.cloud`: WIP de 71 ficheros preservado en stash remoto
  `triaje-claude-2026-07-04` y como evidencia local commiteada en
  `docs/triaje_wip_remoto_2026-07-04/` (parche + tgz de no-trackeados);
  worktree remoto limpio. 4 ficheros ya obsoletos de partida (backend cmd
  migrado). Sin tocar contenedores ni servicios productivos.
- T279 lanzada (puerto 46461, rama `pericial/pilot-t279`): triaje del parche
  a `clasificacion.md` + secciones WIP-TASK ejecutables para lo valioso.
- En vuelo simultáneo: T275 (bug 153), T276 (bug 151), T278 (repo-map),
  T279 (triaje) — 4 goals, write-sets disjuntos.

### 2026-07-04 — triaje remoto cerrado; ola de 4 activa

T279 integrada (`1097671f`): clasificación de los 72 elementos del WIP
remoto — 57 ya implementados (spot-check del supervisor confirmó los 4
veredictos de mayor riesgo: socket corto, symlink guard, reconciliador
external-work más estricto trackeado, QA remota), 11 obsoletos, 5 doc-tasks
→ WIP-TASK-001/002. T280 lanzada (puerto 37393, rama `pericial/pilot-t280`)
ejecutando ambas WIP-TASK. En vuelo simultáneo: T275 (bug 153), T276
(bug 151), T278 (repo-map), T280 (docs/guard T12+matriz). El stash remoto
`triaje-claude-2026-07-04` puede borrarse tras integrar T280 (decisión
operador); mientras, permanece como respaldo.

### 2026-07-04 — ola de 4 completada e integrada: bugs 151 y 153 CERRADOS

Los 4 goals entregaron result formal `complete` (la disciplina de result ya
prende en los agentes). Integrados en orden con verificación completa:
- `72fecb30` T280: matriz smokes + docs/guard T12.
- `e47c2561` T275: **BUG-153 cerrado** — reconciliación de goals idle por
  result materializado, decaimiento `goal_backend_gone_without_result`,
  terminal siempre visible. El bucle autónomo ya se cierra solo.
- `c8e9dbd1` T276: **BUG-151 cerrado** — `Detail` saneado de causa raíz
  hasta status/observe con acción recomendada.
- `60b0ed47` T278: repo_map compacto en el broker (CTX-801B). Un conflicto
  cosmético en 2 tests de mcp resuelto conservando la versión integrada.
Suites verdes en main tras cada integración. Ironía de cierre: los goals de
esta ola aún aparecían `running` en sus instancias por el propio bug 153
que T275 arregla — las olas futuras ya reconciliarán solas.
Pendiente de código: última ola (BUG-152 + CTX-801A/C/D). Después: solo
ventana §9 y decisión de subir el nightly a modo real.

### 2026-07-04 — EXPERIMENTO A/B de contexto (decisión del propietario: probar en práctica, no en papel)

Diseño T286-EXP, a ejecutar tras integrar T284/T285:

**Hipótesis**: el broker central de contexto (rg + codebase-memory + repo_map
por refs compactas) reduce tokens/tiempo frente a exploración libre del
agente, sin degradar calidad.

**Método**: misma tarea acotada de exploración (redactar
`modulos/orquesta-goal/docs/mapa_publico.md` con funciones públicas del
módulo y sus consumidores por imports — tarea que obliga a buscar), dos
pilotajes gemelos desde el mismo HEAD:
- Brazo A (control): perfil actual, broker apagado.
- Brazo B: broker encendido en el perfil self-programming
  (`ORQUESTA_CODEBASE_BROKER_STATE_DIR` + opt-in central + toolbelt).

**Métricas por brazo** (todas ya disponibles):
1. `tokens_used` de `thread_goals` (goals_1.sqlite del codex-home del pilotaje).
2. `context_budget_total_bytes` / `static_prompt_bytes` /
   `dynamic_context_bytes` (801A, en el estado del goal).
3. `prompt_cache.cached_input_tokens` (801C).
4. Duración del goal (created→updated en thread_goals).
5. Calidad: el doc resultante cubre las mismas funciones (diff manual).

**Criterio**: si B no mejora ≥15% en tokens o tiempo con calidad igual, el
broker no se impone por defecto en self-programming y se documenta; si
mejora, se enciende por defecto y se abre seguimiento con las métricas de
deuda. Una muestra por brazo = indicativo, no estadístico; si el resultado
es dudoso, repetir con 3 tareas distintas antes de decidir.

### 2026-07-04 — Plan de mejora continua creado (fase 2)

`docs/plan_mejora_continua_orquesta_2026-07-04.md`: 13 tareas ejecutables por
Orquesta — 6 huecos estructurales (MEJ-101..106: ciclo OPES real,
meta-director de olas, backend Claude, gobernador de presupuesto, memoria
entre goals, deuda residual gobernada) y 7 técnicas del campo (MEJ-201..207:
simulación determinista, property-based, mutation testing, actor-crítico,
golden evals, biblioteca de habilidades, cascada medida). Olas sugeridas por
Alcance disjunto en el propio plan. Pendiente además: experimento A/B del
broker (T286-EXP, diseñado más arriba).

### 2026-07-04 — RELEVO A DIRECTOR CODEX (cuota Claude agotada)

**En vuelo ahora (verificar e integrar con la receta de esta bitácora):**
| Pilotaje | Puerto | Rama | Tarea | Verificación |
| --- | --- | --- | --- | --- |
| T285 despertar-por-result | 35175 | pericial/pilot-t285 | wakeup <2s al materializar result | suites stack+server+cmd+raíz |
| T287 (MEJ-202) property-based | 41411 | pericial/pilot-m202 | propiedades núcleo puro | suites estado-vivo+appserver+run-coordinator+raíz |
| T288 (MEJ-203) mutation piloto | 44545 | pericial/pilot-m203 | script opt-in + runbook | bash -n + test shell focal |
| T289 (MEJ-205) golden evals | 44909 | pericial/pilot-m205 | banco 5 tareas doradas | bash -n + test shell focal |

Protocolo por pilotaje: esperar result durable con matching EXACTO del
task-ref en el nombre del json (no glob amplio: los worktrees contienen
results antiguos commiteados); ignorar placeholders `invalid` iniciales;
verificar con los Tests de su sección; `git checkout --` del backlog mínimo;
commit en la rama del worktree; cherry-pick a `trabajo/plataforma-agentes`;
resolver conflictos aditivos por unión; suites en main; apagar servidor
(kill PID en `<pilot>/server.pid`) y tmux/app-server residuales.

**Cola tras integrar lo anterior (paralelizar por Alcance disjunto):**
1. MEJ-206 (espera a T285: cruza cmd/autoprogramming).
2. T286-EXP experimento A/B broker (diseño completo más arriba; usar
   thread_goals.tokens_used + métricas 801A; criterio ≥15% predefinido).
3. Ola 2: MEJ-201 + MEJ-204 + MEJ-104. Ola 3: MEJ-102 + MEJ-105 + MEJ-207.
4. Con decisión del operador: MEJ-101 (OPES real), MEJ-103 (backend Claude),
   MEJ-106 (deuda residual; retirada legacy solo tras ventana §9 verde).

**Estado global**: plan pericial 18/18 + bugs 151/152/153 cerrados + CTX-801
A/B/C/D integrados + nightly activo (03:30) + remoto saneado. La ventana §9
corre desde hoy: 7 nightlies verdes + métricas planas/bajando = firmar
autonomía. Regla permanente: paralelización por contrato (AGENTS.md),
verificación antes de integrar, y todo cierre deja evidencia aquí.

### 2026-07-03 noche — MEJ-104 empezado por excepcion local

Tras el push de `a9f3b455`, no se relanzo Orquesta real para la siguiente ola
porque BUG-ORQ-20260703-154 ya habia probado consumo alto sin progreso. Codex
integro de forma acotada el gobernador pre-launch de automejora idle: presupuesto
diario por goals/contexto, decision `budget_deferred`/`budget_degraded`,
persistencia en state/status publico y exposicion en
`orquesta.autoprogramming.status.v0` por puerto inyectado. Focal verde:
`go test -count=1 ./modulos/orquesta-autoprogramming ./modulos/orquesta-server ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
Validacion global verde: `git diff --check`, `go test -count=1 ./...` y
`go build ./...`. Pendiente antes de cierre productivo: smoke real acotado y
corte durante ejecucion para goals ya activos con alto consumo/checkpoint
invalido repetido.

### 2026-07-04 — Nota de coordinación del supervisor Claude (cuota renovada)

Leído el informe del director Codex. Coordinación:
- **T285 está verificada verde y asegurada** en la rama
  `pericial/pilot-t285`, commit `2f8c54f9` (suites stack+server+cmd+raíz
  pasadas por el supervisor en su worktree). NO relanzar: solo cherry-pick a
  la rama de trabajo cuando el write-set actual de MEJ-104 quede commiteado.
  Ojo: toca `modulos/orquesta-app-codex-stack` y `modulos/orquesta-server`,
  puede cruzar con MEJ-104 — integrar DESPUÉS y resolver por unión.
- La ola T287/T288/T289 quedó integrada por Codex en `a9f3b455` (rapid en
  go.mod, golden evals, mutation pilot). Los worktrees/ramas
  `pericial/pilot-m20*` pueden limpiarse tras confirmar que nada quedó fuera.
- MEJ-104 empezado por excepción local: correcto dado BUG-154, pero al
  cerrar debe cumplir la regla de evidencia completa (diff-check, suites,
  build) y dejar el corte durante ejecución como tarea propia si no entra.
- Recordatorio de cola: MEJ-206 sigue esperando a T285; T286-EXP (A/B
  broker) listo para lanzar en cuanto haya hueco de máquina.

### 2026-07-04 — Corrección de coordinación: T285 YA integrada por Codex

El director Codex integró el watcher de T285 desde el worktree del pilotaje
(mismo `goal_materialized_result_watcher_v0.go`). La rama
`pericial/pilot-t285` (commit `2f8c54f9`) queda SUPERSEDIDA: no hacer
cherry-pick (duplicaría). Se conserva solo como evidencia. Anula la nota de
coordinación anterior del supervisor en ese punto.

### 2026-07-04 — T290 y T291 integradas; T292 en vuelo

T291 (`df2bed2f`): simulador determinista de fallos en la suite (modo corto
<7s, semilla reproducible, invariantes del ciclo goal-first). T290
(`eab3be97`): corte de goals activos sin progreso útil
(`goal_high_consumption_without_progress` + stop cooperativo) — cierra la
causa raíz de BUG-154. Nota: un flake único en la suite del stack bajo carga
de apagado de pilotos, no reproducido en 2 reruns; vigilar si reaparece.
T292 lanzada (MEJ-204 actor-crítico, rama `pericial/pilot-t292`). Cola tras
T292: T286-EXP (A/B broker), MEJ-102/105/207, y decisiones de operador
(MEJ-101/103/106).

### 2026-07-03 noche — T292 integrada; AUTOMEJORA CORTADA por orden del operador

T292 integrada en `0ce95763` (actor-crítico de tests congelados, MEJ-204),
suites focales verdes (autoprogramming + stack + server + cmd) verificadas en
el worktree del pilotaje antes de integrar. Dos sesiones supervisoras
hicieron cherry-pick concurrente y el commit entró UNA sola vez; sin
duplicados.

**Anomalías del run T292 (deuda real del director, no del código entregado):**
1. El resumen vivo del goal al cerrar decía "MEJ-TASK-201 implementada"
   (tarea equivocada, la de T291) mientras el result durable dice MEJ-204.
   El autoinforme vivo no es fiable como señal de qué se hizo.
2. El result JSON materializó ~3 minutos DESPUÉS de `goal complete`
   (21:22 → 21:25): ventana ciega en la que un watcher por fichero no ve el
   cierre y un observador por estado no tiene evidencia durable.

**ORDEN DEL OPERADOR (2026-07-03): automejora CORTADA.** No lanzar T286-EXP,
MEJ-102/105/207 ni ninguna tarea nueva de mejora. La cola queda CONGELADA.
Prioridad única: que Orquesta funcione autónoma con lo ya integrado y que el
director no se atranque. A cualquier otra sesión supervisora: NO relanzar
pilotajes; los servidores/tmux residuales los apaga la sesión dd9320fa.

**Siguiente trabajo (directo, sin pilotajes Codex):** el director debe
detectar por sí mismo (a) `goal complete` sin result materializado
(publicar anomalía y acción recomendada, no quedarse ciego) y (b)
discrepancia entre el task-ref de la sección de backlog y el autoinforme
del goal.

### 2026-07-03 noche — TESTIGO A CODEX; rumbo fijado por el operador

Integrado por el supervisor Claude (suites `orquesta-server` verdes):
- `6167cff6`: detección de integridad de cierres (`goal_completed_without_
  materialized_result` solo si el cierre no fue aceptado, y
  `goal_self_report_task_mismatch`).
- `717dc4ae`: director de escalada POR EVENTOS: ante anomalías sin acción
  automática invoca un comando externo (default `claude -p`) que decide
  `stop`/`review_ok`/`defer` en JSON; presupuesto diario (default 8),
  idempotencia por firma, lista blanca de acciones, todo auditado.

**Para Codex (siguiente trabajo, por este orden):**
1. Fontanería env del director de escalada en `cmd/orquesta-server`
   siguiendo el patrón de T292 (`ESCALATION_DIRECTOR_ENABLED`, `_COMMAND`
   CSV argv, `_TIMEOUT_SECONDS`, `_MAX_PER_DAY`) + test de config.
2. Ampliar `escalationDirectorEscalatableCodesV0` con los códigos "review_*"
   ya existentes que hoy esperan humano (p.ej. `partial_artifacts_written`
   expuesto vía observación) — solo cablear, sin inventar códigos.

**Decisión del operador sobre la cola congelada (2026-07-03):** entra solo
lo que acerque a TERMINAR Orquesta; nada nuevo después. Entran: MEJ-106
(deuda residual/ratchets; reduce código) tras ventana §9, y MEJ-101 (ciclo
OPES real de validación en campo) como cierre final. Muertas/pospuestas sine
die: MEJ-102 (meta-director de olas), MEJ-105 (memoria entre goals), MEJ-206
(biblioteca habilidades), MEJ-207 (cascada modelos), T286-EXP (A/B broker) y
MEJ-103 en su forma residente (supersedida por el director de escalada).
Regla operativa: programa Codex vía Orquesta; Claude solo supervisa.

### 2026-07-03 noche — Codex cierra fontanería T292/escalation director

Trabajo directo por orden de Claude/operador, sin relanzar pilotajes de
automejora: se uso Codex local para integrar el tapon acotado y subagentes
compactos para auditoria de codigos, config y documentacion. El director de
escalada por eventos queda cableado desde `cmd/orquesta-server`:
`ORQUESTA_SERVER_ESCALATION_DIRECTOR_ENABLED`,
`ORQUESTA_SERVER_ESCALATION_DIRECTOR_COMMAND` (CSV argv redactado en
effective config), `ORQUESTA_SERVER_ESCALATION_DIRECTOR_TIMEOUT_SECONDS` y
`ORQUESTA_SERVER_ESCALATION_DIRECTOR_MAX_PER_DAY`.

Tambien queda ampliada la ingesta de anomalias escalables sin inventar codigos:
`partial_artifacts_written` entra como codigo existente y cualquier `review_*`
se acepta desde issues top-level, `observation.Result.Issues` y
`observation.Closure.Issues`, con dedupe por run/goal/code/field. Esto evita
que revisiones humanas o artefactos parciales observados queden fuera del
director de escalada.

Evidencia ejecutada:
- `go test -count=1 ./cmd/orquesta-server -run 'TestServerConfigFromEnvV0LeeDirectorEscaladaV0|TestServerEnvRegistryV0'`
- `go test -count=1 ./modulos/orquesta-server -run 'Test(EscalationDirector|RuntimeV0EscalationDirector|ParseEscalationDirector)'`
- `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`
- `go test -count=1 ./...`
- `go build ./...`
- `git diff --check`

Documentacion actualizada para Claude:
- `docs/inventario_bugs_orquesta_2026-06-30.md` reconcilia `BUG-150/151/152/153`
  como cerrados segun esta bitacora.
- No se abre `BUG-158`: el hueco de escalation director queda cubierto por
  codigo y tests.
- Siguen abiertos como residuales de esta tanda:
  `BUG-ORQ-20260703-149`, `BUG-ORQ-20260703-154`,
  `BUG-ORQ-20260703-155`, `BUG-ORQ-20260703-156` y
  `BUG-ORQ-20260703-157`.

Pendiente para cierre final, sin relanzar lo congelado: revisar procesos vivos
antes de entregar, commitear y hacer push para que Claude siga.

### 2026-07-03 noche — Codex cierra BUG-155 sin relanzar automejora

Trabajo directo acotado, coherente con la cola congelada: se corrige la mezcla
de criterios de backlog vista en MEJ-206/T290. La causa no estaba en el parser
de secciones, sino en `idleSelfImprovementRequestForBacklogSectionV0`: cada
seccion ejecutable heredaba `base.AcceptanceCriteria`, que incluye politicas
globales de idle/scanner como
`ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS`, planner scanner y
proyeccion publica de outbox/wait_external.

Cambio aplicado:
- las secciones ejecutables usan solo `section.Criteria` mas guardas causales
  propias de backlog/dependencias/manual verification/task instance;
- las revisiones documentales ambiguas tampoco heredan criterios base;
- scanner y fallback conservan criterios globales, porque son inventario
  documental y no implementacion de una tarea concreta.

Evidencia focal:
- antes del fix, `TestIdleSelfImprovementBacklogPlannerV0SeccionEjecutableNoHeredaCriteriosBaseV0`
  fallaba reproduciendo `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS`
  en `AcceptanceCriteria`;
- tras el fix, pasa junto a los tests focales de parser/planner ya existentes.
- `go test -count=1 ./cmd/orquesta-server`;
- `go build ./...`;
- primer `go test -count=1 ./...` tuvo un fallo aislado en
  `TestRuntimeV0SupervisorPreparaAutomejoraTrasIdleV0`; se inventaria como
  `BUG-ORQ-20260703-159` porque el repo exige registrar flakes observados;
- reruns verdes: `go test -count=1 ./modulos/orquesta-server -run TestRuntimeV0SupervisorPreparaAutomejoraTrasIdleV0`,
  `go test -count=1 ./modulos/orquesta-server` y `go test -count=1 ./...`.

Cierre operativo antes de commit: `git diff --check` limpio y sin procesos
vivos de `orquesta-server run`, `codebase-memory-mcp` ni
`codex app-server --listen`.

### 2026-07-03 noche — Codex cierra BUG-159 de automejora idle async

Trabajo directo acotado, sin relanzar pilotajes: se investigó el fallo
`retry no lanzado tras cooldown` observado al validar BUG-155. La incidencia se
reprodujo con `go test -count=100 ./modulos/orquesta-server -run
TestRuntimeV0SupervisorPreparaAutomejoraTrasIdleV0`.

Causa: la idempotencia de publicaciones idle conservaba una publicacion
`scheduled` tras `MarkIdleSelfImprovementErrorV0` o
`MarkIdleSelfImprovementPrepareFailedV0`. Si el retry posterior usaba el mismo
`request_ref`, `RegisterIdleSelfImprovementPublicationV0` lo trataba como
duplicado identico y no llegaba a invocar `PrepareIdleSelfImprovementV0`.
Tambien se estabilizo el test para que espere el drenaje del tick async previo
antes de mutar manualmente el tracker.

Cambio aplicado:
- `resetIdleSelfImprovementPublicationLockedV0` limpia la publicacion activa de
  automejora idle;
- los cierres por error y `prepare_failed` liberan esa idempotencia para permitir
  retry tras cooldown;
- quedan pruebas focales de tracker para error y prepare_failed, mas el stress
  del test async que antes reproducia el fallo.

Evidencia ejecutada:
- `go test -count=100 ./modulos/orquesta-server -run TestRuntimeV0SupervisorPreparaAutomejoraTrasIdleV0`
- `go test -count=1 ./modulos/orquesta-server -run 'TestStatusTrackerV0(IdleSelfImprovementError|PrepareFailed)LiberaPublicacionParaRetryV0|TestRuntimeV0IdleSelfImprovementSaltaBloqueoIdenticoYPublicaWatchdogV0'`

Subagente usado: `019f29a7-e2e5-7fd0-b5b5-6770deb56e88` (explorer Wegener),
solo lectura, sin `codebase-memory-mcp`; coincidió en la causa y recomendó el
reset de publicacion en error/fallo. Pendiente antes de entregar: suites
globales, revision de procesos vivos, commit y push.
