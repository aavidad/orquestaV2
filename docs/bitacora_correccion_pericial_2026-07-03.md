<!--
Bitácora de ejecución del plan de corrección pericial.
Regla: cada agente añade filas; nunca borra ni reescribe filas ajenas.
Manual: docs/manual_correccion_pericial_orquesta_2026-07-03.md
Informe: docs/informe_pericial_claude_orquesta_2026-07-03.md
-->

# Bitácora — corrección pericial Orquesta

<!--
Checkpoint task-ref-doc-cleanup-bitacora-20260704:
alcance autorizado: docs/bitacora_correccion_pericial_2026-07-03.md.
Objetivo: anotar estado vigente y supersedencias sin reescribir la historia.
Evidencia inicial: refs stale localizadas para T-PER-301/302, BUG-149,
BUG-088, BUG-161/162 y BUG-085/164.
-->

## Nota de lectura vigente

Esta bitácora es cronológica. Las filas y entradas antiguas son snapshots del
momento en que se escribieron; no son estado actual salvo que una nota vigente
lo confirme. No relanzar tareas ni abrir trabajo nuevo solo porque una entrada
antigua diga `AVANCE-PARCIAL`, `abierto`, `en vuelo` o `pendiente`: leer esta
nota, la última entrada cronológica aplicable y el inventario de bugs vigente.

| Ref | Estado vigente | Cómo leer entradas antiguas |
| --- | --- | --- |
| `T-PER-301` / `T-PER-302` | Cerradas por T270 / `BUG-ORQ-20260703-150`. | Las filas `AVANCE-PARCIAL` y las menciones a T270 en vuelo son snapshots supersedidos por la entrada "T270 integrada: BUG-ORQ-20260703-150 cerrado". |
| `BUG-ORQ-20260703-150` | Cerrado. | No reabrir la doble implementacion `cmd`/modulo salvo regresion nueva con write-set y pruebas propias. |
| `BUG-ORQ-20260703-149` | Cerrado. | Las notas que lo listan como abierto quedaron supersedidas por "Codex avanza BUG-088 y cierra BUG-149/160" y por el inventario. |
| `BUG-ORQ-20260701-088` | Cerrado funcionalmente para la ruta real acotada de alto consumo/checkpoint. | La frase "`BUG-088` no se cierra" es snapshot anterior al smoke real acotado y al cierre posterior de `BUG-161/162`. |
| `BUG-ORQ-20260703-161` / `BUG-ORQ-20260703-162` | Cerrados. | La lista de residuales abiertos tras el smoke de `BUG-088` queda supersedida por la entrada inmediata de cierre de residuales. |
| `BUG-ORQ-20260701-085` / `BUG-ORQ-20260704-164` | El residual de runtime guard/write-set queda cerrado por `BUG-ORQ-20260704-164`. | Las menciones a `BUG-085` como pendiente de guard runtime son snapshots previos; no relanzar esa guardia sin una regresion nueva. |

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
| T-PER-301 | AVANCE-PARCIAL (snapshot supersedido) | subagente Socrates + Codex local | 2026-07-03 | creado `modulos/orquesta-runtime-codex-appserver` y el wiring principal de `cmd/orquesta-server/codex_goal_backend_env_v0.go` instancia `CommandProtocolV0`, `TmuxBackendV0`, `LazyTmuxProtocolV0` y `GoalBackendV0` desde el módulo nuevo; añadido boundary `TestCodexAppServerAdapterDoesNotImportCmdOrProductStorage`. Tests: `go test -count=1 ./modulos/orquesta-runtime-codex-appserver ./cmd/orquesta-server` y boundary raíz. Nota vigente 2026-07-04: este snapshot queda supersedido por T270 / `BUG-ORQ-20260703-150`, cerrados en la entrada "T270 integrada". |
| T-PER-302 | AVANCE-PARCIAL (snapshot supersedido) | Codex local | 2026-07-03 | añadido `estado_backend_v0.go` con `EstadoBackendAppServerV0`, `ObservacionBackendV0` y `TransicionBackendV0` pura, más tests de transiciones legales/ilegales. Nota vigente 2026-07-04: este snapshot queda supersedido por T270 / `BUG-ORQ-20260703-150`, cerrados en la entrada "T270 integrada". |
| RELEVO CLAUDE ORQUESTADOR | guardado parcial (snapshot supersedido para T-PER-301/302) | Codex local | 2026-07-03 | operador ordena cortar todo para que Claude siga como orquestador. Subagentes cerrados, `orquesta-server run` local de prueba parado, remoto sin agentes vivos observados. Relevo detallado en `docs/relevo_claude_orquestador_2026-07-03.md`. Nota vigente 2026-07-04: la desviacion de T-PER-301/T-PER-302 abiertas queda supersedida por T270 / `BUG-ORQ-20260703-150`; no relanzar por esta fila. |
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
  Nota vigente 2026-07-04: este bloqueo era un snapshot inicial;
  T-PER-301/302 quedaron cerradas por T270 / `BUG-ORQ-20260703-150` y
  T-PER-401/402 tambien tienen cierres posteriores en esta bitácora.
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
- Historico de decision humana de ese momento: bloque remoto de 71 ficheros
  en `/srv/orquesta-self/worktrees/orquesta` (no integrar completo, triar) y
  aclarar quien borro `docs/diseno_router_contexto_hibrido_2026-07-03.md`.

Nota vigente 2026-07-04: esta entrada de T270 en vuelo es histórica; T270 se
integró más abajo y cerró `BUG-ORQ-20260703-150` / T-PER-301-302. El WIP remoto
de 71 ficheros tambien dejo de ser una decision viva: quedo preservado en stash
remoto `triaje-claude-2026-07-04` y tratado como evidencia historica, no como
patch pendiente de integrar.

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
- Nota historica en ese momento del corte: seguian abiertos como residuales de
  esa tanda:
  `BUG-ORQ-20260703-149`, `BUG-ORQ-20260703-154`,
  `BUG-ORQ-20260703-155`, `BUG-ORQ-20260703-156` y
  `BUG-ORQ-20260703-157`. El estado vigente queda actualizado en las entradas
  posteriores de esta bitacora y en el inventario: `BUG-154/155/156/157/159`
  estan cerrados; en ese momento solo quedaba abierto `BUG-149` dentro de esa
  tanda.

Nota vigente 2026-07-04: la lectura intermedia de `BUG-149` como unico residual
vivo queda supersedida por la entrada posterior "Codex avanza BUG-088 y cierra
BUG-149/160" y por el inventario vigente.

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

### 2026-07-03 noche — Codex cierra BUG-156/157 de shutdown y harness

Trabajo directo acotado, sin relanzar pilotajes congelados. Se investigaron los
procesos residuales de pilotos y el falso arranque background de MEJ-206 con dos
subagentes de solo lectura:
- `019f29b0-562d-74d1-8d36-eac6119d0961` (Epicurus): confirmo que el cleaner de
  `codex app-server` tmux existe, pero los hooks no corrian si shutdown salia
  por `async_work_timeout`;
- `019f29b0-683a-7412-a969-8d2970a74db3` (Gibbs): confirmo que
  `orquesta-server run` es foreground, que `start/status/stop` es el contrato
  daemon y que el helper comun de readiness no validaba PID vivo.

Cambios aplicados:
- `RuntimeV0` ejecuta hooks de shutdown una sola vez tambien en rutas de salida
  por timeout, usando contexto fresco best-effort antes de devolver error;
- `smoke_wait_orquesta_readiness_from_state_file` rechaza statefiles con `pid`
  no numerico, cero o muerto antes de aceptar readiness HTTP por `addr`;
- el inventario cierra `BUG-ORQ-20260703-156` y `BUG-ORQ-20260703-157` con causa
  estructural y evidencia.

Evidencia focal ejecutada:
- `bash -n scripts/lib/smoke_common.sh`
- `go test -count=1 ./cmd/orquesta-server -run 'TestSmokeCommonReadinessStateFileRechazaPIDMuertoV0|TestSmokeCommonShutdownCleanupMataAppServerPropioSinBaseURLV0'`
- `go test -count=1 ./modulos/orquesta-server -run 'TestRuntimeV0ShutdownTimeoutPublicaStopTimeoutV0|TestRuntimeV0ShutdownEsperaPreparacionIdleAntesDeStoppedV0|TestRuntimeV0CompactaShutdownHooksV0'`

Pendiente antes de entregar: suites globales, revision de procesos vivos, commit
y push. Residuales nuevos no abiertos en este corte; siguen como deuda
estructural visible `BUG-ORQ-20260703-149` y `BUG-ORQ-20260703-154`.

### 2026-07-03 noche — Codex reconcilia BUG-154/T290 para Claude

Trabajo directo acotado, sin relanzar automejora ni pilotajes congelados. Se
audito `BUG-ORQ-20260703-154` contra codigo, commits y docs: `eab3be97` ya esta
en `trabajo/plataforma-agentes` y contiene T290, por lo que la nota antigua que
decia que faltaba "cortar/replanificar un goal ya activo" estaba stale.

Subagentes usados, ambos solo lectura y sin `codebase-memory-mcp`:
- `019f29b9-371f-7290-94ff-be8763382b7f` (Franklin): auditoria T290/MEJ-104;
  confirmo que el corte durante ejecucion es cerrable y que solo queda smoke
  real si se exige validacion de piloto.
- `019f29b9-4921-79a3-97fc-6587f3a1a5d8` (Confucius): auditoria del inventario;
  confirmo `BUG-ORQ-20260703-149` como WIP remoto no integrable y separo deuda
  antigua de esta tanda.

Evidencia revisada:
- `modulos/orquesta-server/goal_progress_governor_v0.go`: razon
  `goal_high_consumption_without_progress`, persistencia `blocked`, rework y
  stop cooperativo;
- `modulos/orquesta-server/goal_observation_loop_v0.go`: el observer aplica el
  gobernador;
- `cmd/orquesta-server/goal_cooperative_stop_run_control_v0.go` y
  `cmd/orquesta-server/stack.go`: adaptador y wiring de `GoalStopper`;
- `modulos/orquesta-server/docs/orquesta_goal_result_goal-ref-autoprogramming-backlog-t290-corte-durante-ejecucion-goal-sin-progreso.json`:
  resultado durable completo de T290 sin `missing_refs`;
- `docs/inventario_bugs_orquesta_2026-06-30.md` y
  `docs/runbooks/handoff_claude_mejora_continua_orquesta_2026-07-03.md`:
  reconciliados para no pedir a Claude trabajo ya cerrado.

Validacion focal reejecutada:
- `go test -count=1 ./modulos/orquesta-server -run 'TestRuntimeV0GoalObservation(AltoConsumoSinProgreso|NoCortaConProgresoUtilReciente|CheckpointInvalidoRepetidoCuentaSinProgreso)V0|TestRuntimeV0IdleSelfImprovement(Goal|Observe)'`
- `go test -count=1 ./modulos/orquesta-server -run 'TestRuntimeV0AutomejoraIdle(AplazaPorPresupuestoAgotado|DegradaLotePorPresupuestoContexto)V0|TestServerPublicStatusV0ExponePresupuestoAutomejoraIdleV0'`
- `go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPAutoprogrammingStatusExecutorV0PublicaPresupuestoIdleV0'`
- `go test -count=1 ./cmd/orquesta-server -run 'TestServerConfigFromEnvV0LeePresupuestoAutomejoraIdleV0|TestServerAutoprogrammingIdleBudgetSourceV0(LeeEstadoDurable|OmiteDecisionVacia)'`

Resultado: `BUG-ORQ-20260703-154` queda cerrado en inventario por MEJ-104/T290.
No se ejecuta smoke real por la congelacion operativa; queda recomendado como
validacion acotada antes de reactivar automejora/pilotajes caros. En esta tanda
solo sigue abierto `BUG-ORQ-20260703-149`; la deuda antigua OPES/goal-first/
shutdown/write-set/status permanece inventariada como frente estructural aparte.

Nota vigente 2026-07-04: esta lectura de `BUG-149` abierto queda supersedida
por la entrada inmediatamente posterior, que lo cierra como WIP remoto
triado/obsoleto.

### 2026-07-03 noche — Codex avanza BUG-088 y cierra BUG-149/160

Trabajo directo acotado, sin relanzar automejora ni pilotajes congelados. Se
usaron dos subagentes solo lectura, sin `codebase-memory-mcp`:
- `019f29cc-f586-7b30-b6cd-2bde332358d8` (Singer): audito
  `BUG-ORQ-20260703-149`; confirmo que el WIP remoto ya no tiene piezas
  pequenas recomendables para extraer sobre HEAD. Los focales citados por la
  incidencia estan verdes y el unico resto visible reintroduciria rail duro por
  contexto truncado/ref-only.
- `019f29cd-0c0d-7310-8c9c-8e1e27fb4223` (Ampere): audito `BUG-065/088` y
  recomendo mover al `GoalObserver` residente el corte automatico de alto
  consumo/checkpoint-only que ya estaba resuelto por superficies de status y
  run-control.

Cambios aplicados:
- `BUG-ORQ-20260703-149` queda cerrado como WIP remoto triado/obsoleto: no se
  integra ni se extraen mas piezas; el patch queda solo como evidencia
  historica.
- Nuevo helper `goal_observation_high_consumption_v0.go`: el observador
  residente bloquea cualquier goal-first `running` con alto consumo y sin
  artefacto util publicable, distingue `checkpoint_only_high_consumption` de
  `goal_active_no_checkpoint_high_consumption`, persiste `blocked`/`NeedsRework`
  y pide stop cooperativo con `replan_narrow_context`.
- El director de escalada conserva evidencias compactas en el stop cooperativo
  (`BUG-ORQ-20260703-160` cerrado): `evidence-ref-escalation-director-stop`,
  codigo y campo saneados.

Evidencia ejecutada:
- `go test -count=1 ./modulos/orquesta-server -run 'TestRuntimeV0GoalObserver(AltoConsumoCheckpointOnlyPideStopCooperativo|AltoConsumoSinCheckpointPideStopCooperativo|NoParaSiHayArtefactoUtil)V0|TestRuntimeV0EscalationDirectorAplicaStopYEsIdempotentePorFirmaV0'`
- `go test -count=1 ./modulos/orquesta-server`

`BUG-088` no se cierra: falta smoke real que confirme la ruta completa alto
consumo/checkpoint -> segundo artefacto o replan sin app-server residual.

Nota vigente 2026-07-04: este no-cierre es snapshot anterior; `BUG-088` queda
cerrado funcionalmente por el smoke real acotado documentado más abajo y sus
residuales `BUG-161/162` quedan cerrados en la entrada siguiente.

### 2026-07-03 noche — Codex cierra MEJ-106 de deuda residual gobernada

Trabajo directo acotado, sin cambios de produccion. Subagente solo lectura
`019f29d8-ee54-7580-8cb2-984b65828d31` (Poincare) audito MEJ-106 y confirmo
que faltaban ratchet de env vars, primer subpaquete concreto y checklist de
retirada `legacy_director_loop`.

Cambios aplicados:
- `env_vars_budget_test.go` fija `TestEnvVarsBudgetMEJ106V0` con baseline real
  `env_vars_orquesta <= 513`, medido por
  `scripts/orquesta_metricas_deuda.sh --json`. Si baja, debe actualizarse a la
  baja; si sube, falla.
- `docs/runbooks/plan_troceo_hubs_orquesta_2026-07-01.md` nombra el primer
  subpaquete a extraer: workflow tasks/microtasks de
  `orquesta-core-workflow`, con ficheros candidatos exactos, test focal y
  ratchet de paquete plano (154 ficheros no-test).
- La checklist de retirada `legacy_director_loop` queda condicionada a ventana
  §9 verde, redundancia/decision de backend goal, smokes goal-first equivalentes
  y decision explicita del operador. No se borra legacy en esta tarea.

Evidencia ejecutada para cierre:
- `go test -count=1 ./ -run TestEnvVarsBudget`
- `bash scripts/orquesta_metricas_deuda.sh --json`
- `bash scripts/test_orquesta_metricas_deuda.sh`

El plan `docs/plan_mejora_continua_orquesta_2026-07-04.md` queda actualizado
para no relanzar MEJ-106 como aparcada; retirada legacy sigue siendo decision
posterior, no parte de este cierre.

### 2026-07-03 noche — Codex cierra smoke real acotado BUG-088

Trabajo directo acotado con Orquesta/Codex real opt-in, OPES vacio y runtime
temporal. Subagente solo lectura `019f29df-1fc9-7212-9d83-750ccf8d0ed8`
(Einstein) confirmo que no existia script exacto: el smoke normal Nueva App
apagaba el observador residente y no cerraba la ruta alto consumo/checkpoint.

Cambios aplicados:
- `modulos/orquesta-runtime-codex-appserver` permite configurar el umbral real
  de uso alto del backend app-server con
  `ORQUESTA_AUTOPROGRAMMING_CHECKPOINT_ONLY_HIGH_CONSUMPTION_TOKENS`; default
  compatible: 100000 tokens.
- `scripts/smoke_goal_first_checkpoint_only_high_consumption_real.sh` envuelve
  el smoke normal en modo `BUG-088`: observador residente activo, fingerprint
  apagado para smoke, umbral bajo, doble confirmacion y cleanup gobernado.
- `scripts/smoke_goal_first_app_server_real.sh` conserva el comportamiento
  normal, pero en modo alto consumo acepta dos cierres de smoke: replan
  gobernado o segundo artefacto/artefactos parciales con stop cooperativo y
  limpieza `app_server_tmux`.
- Runbook nuevo:
  `docs/runbooks/smoke_goal_first_checkpoint_only_high_consumption_real_2026-07-03.md`.

Evidencia ejecutada:
- Preflight: `ORQUESTA_GOAL_FIRST_SMOKE_PREFLIGHT_ONLY=1 ./scripts/smoke_goal_first_checkpoint_only_high_consumption_real.sh`.
- Smoke real final:
  `ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_REAL_CONFIRM=1 ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_CODEX_EXECUTION_CONFIRMED=1 ORQUESTA_GOAL_FIRST_SMOKE_POLLS=50 ./scripts/smoke_goal_first_checkpoint_only_high_consumption_real.sh`.
- Resultado final: `smoke_goal_first_high_consumption_real=ok`,
  `bug088_path=second_artifact_or_partial_artifacts`,
  `recommended_action=review_partial_artifacts`,
  `app_server_tmux_processes_alive=0`,
  `smoke_root=/tmp/orquesta-goal-first-app-server.lwNV5d`.
- Focales: `go test -count=1 ./modulos/orquesta-runtime-codex-appserver -run 'TestServerCodexAppServerGoalBackendV0(UmbralUsoAltoConfigurable|UmbralUsoAltoDefault|ObservaResultadoMarcado)'`,
  `go test -count=1 ./cmd/orquesta-server -run 'TestServerCodexGoalBackendFromEnvV0TmuxNoArrancaAppServerEnConstruccionV0|TestSmokeGoalFirstHighConsumptionWrapperActivaObserverYUmbralBajoV0'`,
  `bash -n scripts/smoke_goal_first_app_server_real.sh scripts/smoke_goal_first_checkpoint_only_high_consumption_real.sh`.

`BUG-088` queda cerrado funcionalmente para esta ruta real acotada. Quedan
abiertos y documentados para revision estructural:
- `BUG-ORQ-20260703-161`: desfase/proyeccion de ficheros directos del smoke
  como `artifact_refs`.
- `BUG-ORQ-20260703-162`: `shutdown_ready=true` sin `exit_pending/pid` en una
  rama de fallo con cleanup efectivo.

Nota vigente 2026-07-04: estos residuales quedan cerrados por la entrada
inmediata "Codex cierra residuales BUG-161/162 del smoke BUG-088".

### 2026-07-03 noche — Codex cierra residuales BUG-161/162 del smoke BUG-088

Trabajo directo acotado tras el smoke real. Subagente solo lectura
`019f29f6-d108-7cf0-b45a-304ae5da833a` (Kierkegaard) audito `BUG-161` sin MCP
ni edits: confirmo que el scanner no reconocia `checkpoint_started_bug088.txt`
ni `bug088_second_artifact.txt`, y que `autoprogramming observe` no aplicaba el
mismo enriquecimiento de refs materializadas que `apps/director/goal/observe`.

Cambios aplicados:
- `goal_materialized_refs_v0.go` reconoce de forma estrecha
  `checkpoint_started*.txt` como checkpoint y `*_artifact.txt` como artefacto,
  sin aceptar `.txt` generico.
- `autoprogramming_observe_goal_mcp_executor_v0.go` reutiliza
  `withMaterializedRefsV0` para que `/api/v0/autoprogramming/goal/observe`
  publique `artifact_refs`/evidencias igual que la ruta de apps.
- `shutdown_freeze_v0.go` conserva la proteccion contra falsos `ready`, pero no
  reinyecta snapshots activos stale cuando la respuesta `ready` trae evidencia
  de cleanup de backend. Asi `shutdown_ready=true` vuelve a salir con
  `exit_pending=true` y `pid`.

Evidencia ejecutada:
- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestStackGoalMaterializedRefsSourceV0DetectaArtefactosTxtBUG088V0|TestCodexStackAutoprogrammingPrepareRunAPIV0GoalReadyLanzaGoalFirstSinColaLegacy'`
- `go test -count=1 ./modulos/orquesta-server -run 'TestRuntimeV0ServerShutdown(ReadyNoBorraSnapshotPrevioActivo|ReadyTrasCleanupNoHeredaSnapshotPrevioActivo|ConflictSinCuerpoConservaSnapshotPrevioActivo|ReadyDetieneRuntimeHTTP)V0|TestShutdownProjectionFromHTTPV0ReadyConActiveWork'`

Estado: `BUG-ORQ-20260703-161` y `BUG-ORQ-20260703-162` quedan cerrados por
codigo y pruebas focales. `BUG-088` sigue cerrado funcionalmente por el smoke
real ya documentado; no se relanza otro smoke Codex real en este bloque.

### 2026-07-04 — Codex cierra BUG-163 de skills curadas

Revision rapida del handoff de Claude sin relanzar automejora ni pilotos. Se
cerro `BUG-ORQ-20260704-163`: el filtro de MEJ-206 para skills curadas ya no
acepta rutas absolutas genericas tipo `/workspaces/...`, `/project/.../file.md`,
`C:\Users\...` o UNC `\\server\share\...` en metadata compacta o propuestas de
destilacion.

Evidencia ejecutada:
- `go test -count=1 ./modulos/orquesta-autoprogramming -run 'Test(ValidateAutoprogrammingCuratedSkillCatalogV0RechazaRutasAbsolutasGenericas|BuildAutoprogrammingSkillDistillationReviewProposalV0RechazaRutaAbsolutaGenerica)'`
- `go test -count=1 ./cmd/orquesta-server -run 'TestServerCuratedSkillsFromProjectV0IgnoraRutaAbsolutaGenericaV0'`
- `go test -count=1 ./modulos/orquesta-autoprogramming ./cmd/orquesta-server`
- `git diff --check`

Pendientes localizados por subagentes solo lectura y no implementados en este
corte:
- `BUG-065`: anadir `recommended_action` al contrato publico de shutdown para
  active goals/backend still running antes de abordar coordinacion automatica.
- `BUG-085`: runtime path guard en `orquesta-runtime-codex-appserver` usando
  snapshot/verificacion de write-set antes de promover cierre.
- OPES done/settled: conservar `settlement_*` y lifecycle en
  `orquesta-opes-topic-registry` antes de validar `release`.

Nota vigente 2026-07-04: el residual runtime guard de `BUG-085` queda cerrado
por `BUG-ORQ-20260704-164` en la entrada siguiente; `BUG-065` y OPES
done/settled conservan su estado propio.

### 2026-07-04 — Codex cierra guard runtime de write-set BUG-164 / BUG-085 residual

Trabajo directo acotado, sin OPES productivo ni smoke real. Se cerro el
residual de `BUG-ORQ-20260701-085` registrado como
`BUG-ORQ-20260704-164`: el backend app-server ya no acepta un resultado
`complete` si el worktree cambio fuera de `DirectionContract.allowed_write_set`.

Cambios aplicados:
- `orquesta-runtime-codex-appserver` captura un baseline del worktree al lanzar
  goals con `write_set_enforcement=workspace_write_guard`.
- Antes de fusionar un resultado terminal `complete`, ejecuta
  `VerifyWorktreeWriteSetV0`; si hay cambios fuera de scope, bloquea el goal
  como `codex_app_server_runtime_write_set_violation`, elimina recibos de
  dominio del cierre y conserva evidencias compactas de la ruta fuera de scope.
- `orquesta-mcp` proyecta esa senal en `director.stats`,
  `observe_director_goal`, `autoprogramming.status`, `efficiency_summary` y
  `/api/v0/domain-work/status` como bloqueo recuperable con accion
  `rework_write_set_violation`.

Evidencia ejecutada:
- `go test -count=1 ./modulos/orquesta-runtime-codex-appserver -run 'TestServerCodexAppServerGoalBackendV0RuntimeWriteSetGuard'`
- `go test -count=1 ./modulos/orquesta-runtime-worktree -run 'TestVerifyWorktreeWriteSetV0(RechazaCambioFueraDelWriteSet|AceptaCambiosDentroDelWriteSet)'`
- `go test -count=1 ./modulos/orquesta-mcp -run 'Test(MCPAutoprogrammingStatusExecutorV0RuntimeWriteSetViolationPideReworkV0|MCPDirectorStatsToolExecutorV0GoalFirstProyectaRuntimeWriteSetViolationV0|EnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0RuntimeWriteSetViolationPideRework|MCPDomainWorkStatusHTTPHandlerV0NormalizaSenalesGoalFirstRecuperablesComoBloqueadas)'`
- `go test -count=1 ./modulos/orquesta-runtime-codex-appserver`
- `go test -count=1 ./modulos/orquesta-runtime-worktree ./modulos/orquesta-runtime-codex-goal`
- `go test -count=1 ./modulos/orquesta-mcp`
- `go test -count=1 ./cmd/orquesta-server`
- `go test -count=1 ./...`
- `git diff --check`

Limites del cierre:
- No se relanza smoke real Codex/OPES en este bloque.
- La proteccion es de verificacion runtime previa a aceptar cierre; no sustituye
  a un sandbox del proveedor ni a enforcement kernel/FS preventivo.
- Queda pendiente fuera de este corte: `BUG-065` recomendado publico de
  shutdown y OPES done/settled con `settlement_*` en topic registry.

### 2026-07-04 — Orquesta limpia datos historicos falsos para relevo Claude

Trabajo dirigido por Orquesta con tres goals paralelos y write-sets separados:

- `task-ref-doc-cleanup-inventario-20260704`: inventario de bugs.
- `task-ref-doc-cleanup-bitacora-20260704`: bitacora pericial.
- `task-ref-doc-cleanup-handoff-plan-20260704`: handoff, relevo y plan de
  mejora continua.

Refs de Orquesta aceptadas para la ola:
`request-ref-doc-cleanup-historicos-falsos-20260704`,
`goal-ref-task-autoprogramming-39ecd0187760-g01`,
`goal-ref-task-autoprogramming-39ecd0187760-g02` y
`goal-ref-task-autoprogramming-39ecd0187760-g03`.

Cambios documentales integrados:

- `docs/inventario_bugs_orquesta_2026-06-30.md` anade lectura vigente
  2026-07-04: `BUG-085` queda historico/supersedido por `BUG-164`,
  `BUG-088` cerrado funcionalmente, `BUG-120` cerrado por cierre posterior y
  `BUG-065` sigue abierto. Las filas mixtas antiguas ya no cuentan `BUG-085` ni
  `BUG-088` como deuda viva.
- `docs/bitacora_correccion_pericial_2026-07-03.md` conserva los snapshots
  historicos, pero marca como supersedidas las frases de T-PER-301/302,
  `BUG-149`, `BUG-088`, `BUG-161/162` y `BUG-085` que ya no reflejan estado
  vigente.
- `docs/runbooks/handoff_claude_mejora_continua_orquesta_2026-07-03.md`,
  `docs/relevo_claude_orquestador_2026-07-03.md` y
  `docs/plan_mejora_continua_orquesta_2026-07-04.md` quedan alineados:
  `BUG-149` esta cerrado, `MEJ-106` esta cerrado como deuda gobernada y el
  WIP remoto de 71 ficheros queda preservado/triado, no vivo.

Fallo observado usando Orquesta y registrado como bug nuevo:
`BUG-ORQ-20260704-165`. La ejecucion escribio los docs, pero
`/api/v0/autoprogramming/status` y `observe_goal` devolvieron timeouts; antes
del reinicio con backend, `runs/control stop forced=true` sobre T260 no pudo
propagar control pese a no observarse `codex app-server` local vivo. Esto queda
pendiente como problema de observabilidad/control goal-first, no como bloqueo
de los cambios documentales.

Cierre operativo de la sesion: `orquesta-server stop --force --reason ...`
quedo sin respuesta mas de 60s con `shutdown_in_progress` y active works stale
de esta limpieza. Se aborto el cliente de parada y se envio SIGINT al
`orquesta-server run` local de prueba; despues no quedaron procesos
`orquesta-server run`, `codex app-server`, sesiones tmux `orquesta-goal-*` ni
`codebase-memory-mcp`. El state residual queda `stopped/degraded` y forma parte
de `BUG-ORQ-20260704-165`.

## Test de campo Sueldos cerrado accepted (director Claude, 2026-07-04 madrugada)

- Run `run-spec-mapa-de-gasto-publico-req-mapa-de-gasto-publico-1edc177159293ad5e0cd489f78a5fb96`
  (feature cargos/partidos) lanzado goal-first por `/api/v0/apps/director` en
  servidor aislado `.orquesta-feature-cargos-2` con umbral de consumo 450k.
- Resultado durable `complete` con `missing_refs=[]` y test requerido `passed`;
  `observe` reconcilio y cerro: `run=cerrada, goal=complete, closure=accepted,
  no_action_closed`. Autonomia extremo a extremo verificada sin intervencion
  manual (validacion de campo de los fixes BUG-166/167/168).
- Servidor parado limpio con SIGINT; sin tmux ni app-server residuales del run.
- En paralelo sigue `pilot-t294` (fix BUG-ORQ-20260704-165) con watcher activo.
- Instrucciones de relevo para director Codex: `docs/instrucciones_director_codex_2026-07-04.md`.

## Relevo nocturno a director Codex (2026-07-04 ~06:30)

- Sueldos cerrado accepted (ver seccion anterior). Tres pilotajes prepare-run
  corriendo: t294 (BUG-165), t295 (regresion escaner idle: 3 ciclos no-op sin
  ejecutar tarea; via idle inutilizable hoy), t296 (BUG-065/076 shutdown).
- Hallazgo: receta de pilotaje por cola idle OBSOLETA con el escaner nuevo;
  usar prepare-run directo (contrato en docs/instrucciones_director_codex_2026-07-04.md §4).
- Umbral aplicado en pilotajes: ORQUESTA_AUTOPROGRAMMING_CHECKPOINT_ONLY_HIGH_CONSUMPTION_TOKENS=450000,
  ORQUESTA_CODEX_GOAL_TIMEOUT_MS=1800000.
- Pendientes tras integrar: 058, 066, 073, 075, 079 y smoke real MEJ-104.

## Relevo director Codex 2026-07-04

Integrado en rama `trabajo/plataforma-agentes`:

- `0e0dcedc`: BUG-166/167/168 Sueldos. Reconciliacion de receipts por
  `goal_ref`, exclusion de `.orquesta-*`/`.gocache-local` y evidencia de campo
  accepted ya documentada.
- `7a6dea0d`: T294 / BUG-165 parcial. `runs/control stop|cancel forced=true`
  propaga cierre al backend Goal tmux por puerto de control, conserva cierre
  terminal `blocked/canceled` con evidencias y evita que `observe_goal` siga
  publicando `running` cuando ya hay evidencia terminal forzada.
- `6fe19d06`: T295 / BUG-169 cerrado. `Dependencias: ninguna` ya no se trata
  como dependencia real pendiente; el planner emite `backlog_autoprogramming`
  antes que scanner/fallback cuando hay backlog ejecutable.
- `67dd7fa9`: T296 / BUG-065/076 reducido. Shutdown publica `goal_actions`
  tipadas (`wait_checkpoint`, `forced_stop_requested`, `cleanup_required`,
  `cleanup_requested`, `cleanup_completed`) y revalida active work antes de
  publicar ready.

Evidencia ejecutada por Codex:

- `go test -count=1 ./cmd/orquesta-server -run 'TestIdleSelfImprovementBacklogPlannerV0(BacklogMinimoPendienteNoCedeCicloAlScanner|RespetaDependencias|PlanificaDependiente|SaltaTareasYaEnCola|AnadeScannerSiTodoEstaEnCola|NoInventaFallbackSiTodoEstaEnCola)'`
- `go test -count=1 ./modulos/orquesta-runtime-codex-appserver ./modulos/orquesta-app-director-service ./modulos/orquesta-app-codex-stack ./modulos/orquesta-orchestration-core ./modulos/orquesta-server-shutdown`
- `go test -count=1 ./modulos/orquesta-server-shutdown`
- `go test -count=1 ./cmd/orquesta-server`
- `git diff --check`

Limpieza operativa:

- Cerrados servidores piloto T294/T295/T296 (`server.pid` 2226334, 2435566,
  2435721).
- Cerradas sesiones tmux `orquesta-goal-7905bea451024ae4`,
  `orquesta-goal-c2a9a1ad94fcc2a2` y
  `orquesta-goal-7b561bb4f88959a8`.
- Cerrados app-servers residuales de esos sockets.
- Retirados worktrees/ramas `pilot-t294`, `pilot-t295`, `pilot-t296`.
- `scripts/bootstrap_agent_tooling.sh --status` queda `estado=ok`,
  `live_codebase_memory_mcp_processes=0`. Se dejo viva solo la sesion
  `claude --resume`.

Estado y pendientes para Claude:

- BUG-165 no se declara cerrado total: falta revalidacion real posterior al
  fix con alto consumo/status lento o un smoke equivalente. El codigo de stop
  forzado esta integrado y probado en suites focales.
- BUG-065/076 quedan reducidos por acciones tipadas y relectura de active work,
  pero falta smoke/escenario real amplio de cleanup externo y coordinacion
  completa backend/checkpoint/stop/cancel/wait.
- T295/BUG-169 queda cerrado en codigo y tests.
- Siguen abiertos los frentes vivos ya listados por Claude: 058, 066, 073, 075,
  079 y smoke real MEJ-104 si sigue vigente.

## Relevo director Codex 2026-07-04 tarde

Avances integrados antes de entregar a Claude:

- `BUG-ORQ-20260701-073` cerrado por reejeucion real acotada del smoke
  `scripts/smoke_goal_first_checkpoint_only_high_consumption_real.sh` con
  backend `app_server_tmux`: `smoke_goal_first_high_consumption_real=ok`,
  `bug088_path=second_artifact_or_partial_artifacts`,
  `recommended_action=review_partial_artifacts`,
  `app_server_tmux_processes_alive=0`, run
  `run-spec-smoke-goal-first-bug088-req-smoke-goal-first-bug088-e83f3577077c1956448f461f65d4230b`
  y external goal `019f2bbd-86a5-7fd1-b7c1-bb7c39f220e7`.
- MEJ-104 / `BUG-ORQ-20260703-154` cerrado tambien en smoke real acotado:
  servidor temporal con budget diario agotado y backend `app_server_tmux`
  publica `budget_deferred` en `/api/v0/server/status` y
  `/api/v0/autoprogramming/status`, sin lanzar goal Codex. Evidencia en
  `docs/runbooks/smoke_autoprogramming_idle_budget_2026-07-04.md`.
- `BUG-ORQ-20260701-075` reducido por parche OPES: una entrega terminal sin
  `required-evidence-*` queda en `pendiente_rework_evidencia_minima`,
  `operational_status=needs_rework`,
  `settlement_status=needs_rework`,
  `settlement_scope=required_evidence` y
  `settlement_reason=required_evidence_missing`, con siguiente trabajo
  `review_director_consolidation`.
- `BUG-ORQ-20260701-079` revisado por subagente: no hay parche pequeno seguro
  en `modulos/orquesta-runtime-codex-goal`; el packet ya pide checkpoint
  temprano y limite de salida, pero el enforcement preventivo real pertenece al
  backend/proveedor o a un diseno runtime mayor.

Evidencia ejecutada en este tramo:

- `ORQUESTA_GOAL_FIRST_SMOKE_PREFLIGHT_ONLY=1 ./scripts/smoke_goal_first_checkpoint_only_high_consumption_real.sh`
- `ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_REAL_CONFIRM=1 ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_CODEX_EXECUTION_CONFIRMED=1 ORQUESTA_GOAL_FIRST_SMOKE_POLLS=50 ./scripts/smoke_goal_first_checkpoint_only_high_consumption_real.sh`
- smoke manual residente MEJ-104 con state temporal bajo
  `/tmp/orquesta-idle-budget-smoke-20260704` y resultado `idle_budget_smoke=ok`.
- `go test -count=1 ./modulos/orquesta-server -run 'TestRuntimeV0GoalObservation|TestRuntimeV0AutomejoraIdle'`
- `go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPAutoprogrammingStatusExecutorV0(OutputGiganteSaneado|CheckpointOnlyHighConsumption|SinCheckpointHighConsumption)|TestMCPRunControlExecutorV0StopForcedReconcilesGoalHighConsumption'`
- `go test -count=1 ./cmd/orquesta-server -run 'TestSmokeGoalFirstHighConsumptionWrapperActivaObserverYUmbralBajoV0'`
- `go test -count=1 ./modulos/orquesta-runtime-codex-goal`
- `go test -count=1 ./modulos/orquesta-opes-director ./modulos/orquesta-opes-bridge`
- `git diff --check -- modulos/orquesta-opes-director modulos/orquesta-opes-bridge`
- `go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-autoprogramming ./modulos/orquesta-mcp ./cmd/orquesta-server ./modulos/orquesta-runtime-codex-goal`
- `go test -count=1 ./...`
- `git diff --check`

Pendientes reales para Claude:

- `BUG-165` sigue abierto para timeouts amplios de `status/observe` y
  reconciliacion/control completo en escenarios reales fuera del smoke acotado.
- `BUG-065/076` siguen abiertos para smoke amplio de cleanup externo y
  coordinacion backend/checkpoint/stop/cancel/wait.
- `BUG-058/066` siguen abiertos para lifecycle OPES end-to-end con instancia
  temporal y criterios nativos `done/settled`.
- `BUG-075` sigue abierto para matriz completa de validadores OPES por
  `work_kind`, aunque el caso terminal sin evidencia minima ya queda corregido.
- `BUG-079` sigue abierto para enforcement runtime/proveedor de checkpoint
  temprano y limites preventivos de salida de herramientas.

## Relevo director Codex 2026-07-04 noche

Cambios integrados en este tramo:

- `BUG-ORQ-20260701-075`: el director OPES ya no solo marca el registro de tema
  cuando falta evidencia minima. Ahora el follow-up causal
  `review_director_consolidation` recibe campos accionables:
  `rework_reason=required_evidence_missing`,
  `required_evidence_missing_refs`, `publication_status` no publicable y
  `recommended_action=review_required_evidence`. Se anadio matriz ejecutiva
  para toda la secuencia OPES: cada `work_kind` terminal sin evidencia minima
  queda en `needs_rework` y no publicable.
- `BUG-ORQ-20260701-079`: `thread/read` del backend
  `orquesta-runtime-codex-appserver` tiene limite especifico de respuesta de
  256 KiB. Si el frame WebSocket anunciado supera ese limite, Orquesta falla
  antes de reservar/leer el payload completo con
  `codex_app_server_thread_read_response_too_large`. Los RPCs no `thread/read`
  conservan el limite general de 16 MiB.
- `BUG-ORQ-20260704-165`: el observador residente goal-first tiene timeout
  interno `GoalObserverTimeout`, default 2000 ms, configurable por composicion
  `ConfigV0` sin env nueva por el ratchet MEJ-106. El tick de
  `ObserveActiveGoalWorksV0` usa contexto acotado y publica
  `goal_observer_timeout` como error accionable.

Archivos tocados por este tramo:

- `modulos/orquesta-opes-director/job_requests_v0.go`
- `modulos/orquesta-opes-director/producer_v0_test.go`
- `modulos/orquesta-runtime-codex-appserver/codex_goal_app_server_websocket_protocol_v0.go`
- `modulos/orquesta-runtime-codex-appserver/codex_goal_app_server_migrated_v0_test.go`
- `modulos/orquesta-server/config_v0.go`
- `modulos/orquesta-server/config_v0_test.go`
- `modulos/orquesta-server/goal_observation_loop_v0.go`
- `modulos/orquesta-server/goal_observation_loop_v0_test.go`
- `cmd/orquesta-server/config_test.go`
- `docs/inventario_bugs_orquesta_2026-06-30.md`
- `docs/bitacora_correccion_pericial_2026-07-03.md`
- `docs/instrucciones_director_codex_2026-07-04.md`

Evidencia ejecutada en este tramo:

- `go test -count=1 ./modulos/orquesta-opes-director -run 'TestProduceOPESCausalJobsV0(BloqueaSecuenciaOPESCompletaSinEvidenciaMinima|BloqueaDerivadoOPESSinEvidenciaMinima|DerivadoOPESConEvidenciaMinimaNoCreaRework)|TestTopicRegistryRequiredEvidencePolicyV0CubreSecuenciaOPESCompleta'`
- `go test -count=1 ./modulos/orquesta-opes-director ./modulos/orquesta-opes-bridge`
- `go test -count=1 ./modulos/orquesta-runtime-codex-appserver ./modulos/orquesta-server ./cmd/orquesta-server`

Pendientes reales para Claude tras este tramo:

- `BUG-075` ya no queda pendiente por la matriz terminal sin evidencia minima.
  Sigue abierto para validadores OPES semanticos/editoriales por artefacto
  canonico y smoke OPES temporal end-to-end.
- `BUG-079` queda reducido por corte de ingesta en Orquesta, pero sigue abierto
  para enforcement preventivo real antes de que el proveedor/runtime genere
  salidas gigantes o avance sin checkpoint temprano.
- `BUG-165` queda reducido para el observador residente. Tras el avance
  posterior de esta bitacora, el tick tampoco queda bloqueado si un backend
  ignora `context.Context`; sigue abierto para revalidacion real amplia de
  `status/observe` lento con backend/proveedor y coordinacion completa.
- `BUG-065/076` siguen abiertos para smoke amplio de cleanup externo y
  coordinacion completa backend/checkpoint/stop/cancel/wait.
- `BUG-058/066` siguen abiertos para lifecycle OPES end-to-end con instancia
  temporal y criterios nativos `done/settled`.

## Continuacion Codex 2026-07-04 noche 2

Avance adicional sobre `BUG-ORQ-20260704-165`:

- `runGoalObservationTickV0` ya no llama directamente al backend residente.
  La llamada `ObserveActiveGoalWorksV0` queda aislada con deadline duro; si el
  backend no respeta `context.Context`, el tick vuelve igualmente, persiste
  `goal_observer_timeout` y no bloquea el loop residente.
- Mientras esa llamada backend anterior siga viva, ticks posteriores no abren
  llamadas infinitas: publican `goal_observer_backend_call_in_flight`.
- `self_watchdog` considera `goalObservationBackendActive` como causa
  operacional viva, de forma que una llamada backend colgada no desaparece
  del diagnostico interno al haber finalizado el tick externo.
- La goroutine de backend conserva recuperacion de panics y los transforma en
  error `panic:<causa>` en vez de sacar el proceso por un panic fuera del recover
  del tick.

Archivos tocados en este avance:

- `modulos/orquesta-server/runtime_v0.go`
- `modulos/orquesta-server/goal_observation_loop_v0.go`
- `modulos/orquesta-server/goal_observation_loop_v0_test.go`
- `modulos/orquesta-server/self_watchdog_loop_v0.go`
- `modulos/orquesta-server/self_watchdog_v0_test.go`
- `docs/inventario_bugs_orquesta_2026-06-30.md`
- `docs/bitacora_correccion_pericial_2026-07-03.md`

Evidencia ejecutada:

- `go test -count=1 ./modulos/orquesta-server -run 'TestRuntimeV0GoalObservation(TickTimeout|AsyncCoalescea|TickCorre|TickPanic)'`
- `go test -count=1 ./modulos/orquesta-server`

Pendiente real: `BUG-165` no se declara cerrado total sin un smoke real amplio
de `status/observe` lento con backend/proveedor y coordinacion completa con
`runs/control`/cleanup.

## Continuacion Codex 2026-07-04 noche 3

Avance adicional sobre `BUG-ORQ-20260701-065` / `BUG-076`:

- El contrato de shutdown ya no deja `goal_actions` como informacion interna de
  `orquesta-server-shutdown`: MCP/HTTP las serializan, `active_works` conserva
  `action_taken`/`action_evidence_refs` y el status publico del servidor
  persiste `shutdown_goal_actions`.
- `orquesta-server stop` normaliza `goal_actions` en refs compactas y las trata
  como bloqueo operativo si `action_taken` no es `cleanup_completed`. Una
  respuesta `ready/shutdown_ready=true` con acciones pendientes ya no permite
  enviar la senal final.
- `shutdown_freeze` parsea `goal_actions` desde el body HTTP upstream, las
  conserva en `StateV0`/status publico y fuerza `stop_pending` si quedan
  acciones bloqueantes aunque no haya `active_work_count` ni `active_work_refs`.

Archivos tocados en este avance:

- `modulos/orquesta-server/status_tracker_idle_shutdown_v0.go`
- `modulos/orquesta-server/state_v0.go`
- `modulos/orquesta-server/status_public_v0.go`
- `modulos/orquesta-server/status_tracker_v0.go`
- `modulos/orquesta-server/status_tracker_restore_v0.go`
- `modulos/orquesta-server/status_process_stale_v0.go`
- `modulos/orquesta-server/shutdown_freeze_v0.go`
- `modulos/orquesta-server/shutdown_freeze_v0_test.go`
- `cmd/orquesta-server/shutdown_client.go`
- `cmd/orquesta-server/shutdown_client_readiness_v0.go`
- `cmd/orquesta-server/shutdown_client_v0_test.go`
- `modulos/orquesta-mcp/server_shutdown_tool_v0.go`
- `modulos/orquesta-mcp/server_shutdown_tool_v0_test.go`
- `modulos/orquesta-mcp/server_shutdown_http_v0_test.go`
- `modulos/orquesta-mcp/docs/contratos.md`
- `modulos/orquesta-mcp/docs/pruebas.md`
- `modulos/orquesta-server-shutdown/docs/contratos.md`
- `docs/inventario_bugs_orquesta_2026-06-30.md`
- `docs/bitacora_correccion_pericial_2026-07-03.md`

Evidencia ejecutada:

- `go test -count=1 ./cmd/orquesta-server -run 'TestRequestServerShutdownV0ReadyNoSaltaGoalActionsSinActiveWorkV0|TestNormalizeServerShutdownClientResultV0ConvierteActiveWorksEnRefs|TestShutdownClientReadyV0'`
- `go test -count=1 ./modulos/orquesta-server -run 'TestShutdownProjectionFromHTTPV0ReadyConGoalActionsQuedaStopPendingV0|TestServerPublicStatusV0ExponeShutdownGoalActionsV0'`
- `go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPServerShutdown(DescriptorV0DeclaraEvidenciaV0|ToolExecutorV0ExponeGoalsActivos|HTTPHandlerV0BackendStillRunningDevuelveConflict)'`

Pendiente real: `BUG-065/076` sigue abierto para smoke real/corte externo
amplio y coordinacion automatica completa backend/checkpoint/stop/cancel/wait.

## Continuacion Codex 2026-07-04 noche 4

Avance adicional sobre `BUG-ORQ-20260701-058/066`:

- `update_topic_registry` ya no publica un tema con `settlement_status=settled_text`
  como `proposed_status=en_progreso_orquesta` y
  `operational_status=working`.
- Si el texto OPES tiene contrato de calidad completo, no tiene rework/pending
  refs, cumple el checkpoint goal-first cuando aplica y aun quedan derivados,
  publica `proposed_status=texto_asentado_pendiente_derivados` y
  `operational_status=waiting`.
- El objetivo es distinguir texto asentado de trabajo todavia escribiendose,
  sin promoverlo a `settled_final` ni a paquete completo.

Archivos tocados en este avance:

- `modulos/orquesta-opes-director/topic_registry_v0.go`
- `modulos/orquesta-opes-director/producer_v0_test.go`
- `modulos/orquesta-opes-director/README.md`
- `docs/inventario_bugs_orquesta_2026-06-30.md`
- `docs/bitacora_correccion_pericial_2026-07-03.md`

Evidencia ejecutada:

- `go test -count=1 ./modulos/orquesta-opes-director`
- `go test -count=1 ./modulos/orquesta-opes-director ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-topic-registry`
- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestCodexStackV0ExternalWorkGoalFirstCierraSecuenciaOPESDerivadosConReceiptsLedgerV0|TestOperationalClosureSourceV0NoCierraOPESFinalSinResultadosTopicQualityPorTema|TestGoalDomainReceiptClosureValidatorV0BloqueaOPESFinalSinTopicQualityContractRefsV0'`
- `go test -count=1 ./...`
- `git diff --check`

Pendiente real: `BUG-058/066` sigue abierto para smoke OPES temporal
end-to-end, reconciliacion tras cortes externos/manuales y criterios completos
de cierre del arbol OPES hasta paquete final.

Brechas locales detectadas por subagentes al inicio de este corte:

- `BUG-165`: `observe_goal` en timeout/snapshot puede publicar `goal_status=running`
  si `GoalStateStore` sigue running pero `RunControl` ya esta terminal.
- `BUG-079`: `CommandProtocolV0` todavia no aplicaba el budget especifico de
  256 KiB de `thread/read`; el WebSocket si lo aplicaba.

## Continuacion Codex 2026-07-04 noche 5

Avance adicional sobre `BUG-ORQ-20260701-079`:

- `serverCodexAppServerCommandProtocolV0.ReadThreadV0` usa ahora el mismo
  presupuesto especifico de 256 KiB que WebSocket para respuestas
  `thread/read`.
- Si la linea stdout JSON-RPC de `thread/read` supera ese limite, Orquesta
  devuelve `codex_app_server_thread_read_response_too_large`, cierra stdin y
  mata el proceso hijo para que el diagnostico no se degrade a timeout.
- Las llamadas command no `thread/read` mantienen el limite historico de 1 MiB;
  los RPCs WebSocket no `thread/read` mantienen su limite global.

Archivos tocados en este avance:

- `modulos/orquesta-runtime-codex-appserver/codex_goal_app_server_command_protocol_v0.go`
- `modulos/orquesta-runtime-codex-appserver/codex_goal_app_server_migrated_v0_test.go`
- `docs/inventario_bugs_orquesta_2026-06-30.md`
- `docs/bitacora_correccion_pericial_2026-07-03.md`

Evidencia ejecutada:

- `go test -count=1 ./modulos/orquesta-runtime-codex-appserver`

Pendiente real: `BUG-079` sigue abierto para enforcement runtime/proveedor de
checkpoint temprano y limites de salidas de herramientas antes de que se genere
la salida gigante. `BUG-165` sigue abierto para reconciliar `observe_goal`
timeout/snapshot con `RunControl` terminal.

## Continuacion Codex 2026-07-04 noche 6

Avance adicional sobre `BUG-ORQ-20260704-165`:

- `ObserveAppDirectorGoalTimeoutSnapshotV0` consulta ahora `RunControl` cuando
  construye un snapshot parcial desde `GoalStateStore`.
- Si el estado goal local sigue `running/accepted`, pero `RunControl` ya esta
  terminal (`stopped` o `canceled`), el snapshot no publica `goal_status=running`.
- La proyeccion publica pasa a `goal_status=blocked`,
  `closure_status=blocked`, `closure_needs_rework=true`,
  `recommended_action=replan` y conserva evidencia de `RunControl` junto a
  `evidence-ref-observe-goal-run-control-terminal`.
- El cambio es no destructivo: no persiste estado goal desde la ruta de
  timeout; solo evita que la superficie publica contradiga un control terminal.

Archivos tocados en este avance:

- `modulos/orquesta-mcp/observe_app_director_goal_tool_executor_v0.go`
- `modulos/orquesta-mcp/observe_app_director_goal_http_v0_test.go`
- `modulos/orquesta-app-codex-stack/goal_first_observe_mcp_executor_v0_test.go`
- `docs/inventario_bugs_orquesta_2026-06-30.md`
- `docs/bitacora_correccion_pericial_2026-07-03.md`

Evidencia ejecutada:

- `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-app-codex-stack -run 'TestMCPObserveAppDirectorGoalToolExecutorV0TimeoutSnapshot(NoPublicaRunningSiRunControlTerminal|LeeGoalState)|TestCodexStackObserveAppDirectorGoalExecutorV0TimeoutSnapshot(RespetaRunControlTerminal|IncluyeProcessRefs)'`

Pendiente real: `BUG-165` sigue abierto para smoke real amplio con
backend/proveedor lento o vivo tras stop forzado, y para confirmar propagacion
completa de stop/cancel cuando el backend siga activo.

## Continuacion Codex 2026-07-04 noche 7

Avance adicional sobre `BUG-ORQ-20260701-065`:

- El supervisor residente goal-first ya cerraba `GoalWorkState` a
  `blocked/rework` cuando detectaba que un backend Goal habia desaparecido por
  cleanup externo y preparaba un rework causal.
- Ahora esa misma ruta completa tambien `RunControl` como `stopped`, con
  reason/idempotencia de `external cleanup`, sin texto `forced`.
- Conserva `evidence-ref-run-control-terminal-after-goal-reconcile` junto a la
  evidencia causal `evidence-ref-autoprogramming-goal-backend-missing-after-external-cleanup`.
- Esto evita que el estado goal quede terminal mientras `RunControl` sigue
  pendiente y obliga al operador a reconciliarlo manualmente.

Archivos tocados en este avance:

- `modulos/orquesta-app-codex-stack/goal_first_resident_rework_v0.go`
- `modulos/orquesta-app-codex-stack/goal_first_resident_rework_v0_test.go`
- `docs/inventario_bugs_orquesta_2026-06-30.md`
- `docs/bitacora_correccion_pericial_2026-07-03.md`

Evidencia ejecutada:

- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestRunSupervisorGoalFirstResidentReconciliaBackendMissingTrasCleanupExternoV0|TestRunSupervisorGoalFirstNoResidentNoReconciliaBackendMissingTrasCleanupExternoV0'`

Pendiente real: `BUG-065` sigue abierto para smoke real amplio y coordinacion
automatica completa backend/checkpoint/stop/cancel/wait tras cortes externos o
manuales.

## Continuacion Codex 2026-07-04 noche 8

Avance adicional sobre `BUG-ORQ-20260701-079`:

- `serverCodexAppServerGoalBackendV0.turnStartParamsV0` ya no copia
  `packet.Prompt` sin refuerzo al `turn/start`.
- Antes de llamar al provider, el input efectivo inyecta un contrato runtime
  con checkpoint temprano dentro del write-set, no pegar salidas largas,
  `max_text_bytes`, `thread_read_max_bytes=256 KiB`, comandos acotados,
  evidencia durable y final/ACK compacto.
- El contrato se deduplica si ya esta presente y usa defaults seguros si un
  packet legacy no trae `DirectionContract`.
- Un subagente reviso el runtime app-server: `turnStartParamsV0` es el unico
  punto donde `packet.Prompt` alimenta `turn/start`; WebSocket y command
  protocol consumen el mismo `params.toJSONV0()`.

Archivos tocados en este avance:

- `modulos/orquesta-runtime-codex-appserver/codex_goal_app_server_v0.go`
- `modulos/orquesta-runtime-codex-appserver/codex_goal_app_server_migrated_v0_test.go`
- `docs/inventario_bugs_orquesta_2026-06-30.md`
- `docs/bitacora_correccion_pericial_2026-07-03.md`

Evidencia ejecutada:

- `go test -count=1 ./modulos/orquesta-runtime-codex-appserver -run 'TestServerCodexAppServerGoalBackendV0(TurnStartInyectaContratoSalidaCompacta|LanzaThreadGoalYTurnMigrado)V0'`

Pendiente real: `BUG-079` sigue abierto para enforcement duro del
proveedor/runtime antes de ejecutar herramientas y smoke real largo que confirme
checkpoint temprano sin consumo gigante previo.

Revision adicional sobre scanner idle/T295:

- La lectura vigente del inventario marca `BUG-ORQ-20260704-169` cerrado por
  `6fe19d06`: `Dependencias: ninguna` ya no bloquea una tarea ejecutable ni
  cede el ciclo al scanner.
- Riesgo residual documentado: si un scanner futuro deja solo `## Escaneo
  backlog ...` y no materializa/actualiza secciones `## Txx` pendientes, el
  planner no tendra tarea ejecutable. Eso debe reabrirse como regresion nueva
  del contrato scanner -> `Txx pendiente`, no como cierre de T295.

## Continuacion Codex 2026-07-04 noche 9

Avance adicional sobre `BUG-ORQ-20260704-165` / control goal-first:

- `runs/control` consideraba terminales `complete`, `accepted`, `canceled`,
  `stopped` y `failed`, pero no `blocked` ni `invalid`.
- El backend `orquesta-runtime-codex-appserver` marca el thread goal como
  `blocked` cuando ejecuta forced stop, por lo que esa respuesta podia salir
  como `goal_control_signal_confirmed=false` aunque el backend ya no estuviera
  activo.
- Ahora `blocked` e `invalid` cuentan como terminales para confirmar la senal
  de control del backend goal-first.
- Tambien cuentan como terminales los estados de proveedor/presupuesto/politica
  `usageLimited`, `quotaLimited`, `providerLimited`, `budgetLimited`,
  `policyLimited` y sus variantes snake_case, coherentes con la normalizacion
  del app-server a `GoalStatusBlockedV0`.
- Si el backend sigue `active`, se conserva el bloqueo duro existente
  `control_not_propagated_to_goal_backend`.

Archivos tocados en este avance:

- `modulos/orquesta-mcp/run_control_tool_executor_v0.go`
- `modulos/orquesta-mcp/run_control_tool_v0_test.go`
- `docs/inventario_bugs_orquesta_2026-06-30.md`
- `docs/bitacora_correccion_pericial_2026-07-03.md`

Evidencia ejecutada:

- `go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPRunControlExecutorV0StopForced(ConfirmaBackendBlocked|ConfirmaBackendProviderLimited|PermiteTerminalSiGoalBackendYaComplete|NoPublicaStoppedSiGoalBackendSigueActive)'`
- `go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPRunControlExecutorV0StopForced(ConfirmaBackendBlocked|PermiteTerminalSiGoalBackendYaComplete|NoPublicaStoppedSiGoalBackendSigueActive|ReconcilesGoalHighConsumptionSinCheckpoint|ReconcilesGoalHighConsumptionCheckpointOnly)'`

Pendiente real: `BUG-165` sigue abierto para smoke real amplio con proveedor o
backend lento/vivo tras stop forzado, y `BUG-065/076` siguen abiertos para
coordinacion automatica completa de shutdown/backend/checkpoint/stop/cancel/wait.

## Continuacion Codex 2026-07-04 noche 10

Reejecucion real del smoke alto consumo goal-first:

- Preflight `app_server_tmux`: `smoke_goal_first_app_server_preflight=ok`.
- Smoke real:
  `ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_REAL_CONFIRM=1`,
  `ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_CODEX_EXECUTION_CONFIRMED=1`,
  `ORQUESTA_GOAL_FIRST_SMOKE_POLLS=50`,
  `ORQUESTA_KEEP_SMOKE_DIR=1`,
  `./scripts/smoke_goal_first_checkpoint_only_high_consumption_real.sh`.
- Resultado: `smoke_goal_first_high_consumption_real=ok`,
  `bug088_path=second_artifact_or_partial_artifacts`,
  `recommended_action=review_partial_artifacts`,
  `app_server_tmux_processes_alive=0`.
- Refs: `run_ref=run-spec-smoke-goal-first-bug088-req-smoke-goal-first-bug088-6c8dc4317888c8e25bb0e91f7f910aab`,
  `goal_ref=goal-ref-app-director-run-spec-smoke-goal-first-bug088-req-smoke-goal-first-bug088-6c8dc4317888c8e25bb0e91f7f910aab`,
  `external_goal_ref=019f2c17-c3a7-78c3-857b-93f04e3286b1`.
- `observe_response.json` publico alto consumo con `tokens_used=13658` y
  `time_used_seconds=17`; el write-set contiene
  `generated-apps/checkpoint_started_bug088.txt` y
  `generated-apps/bug088_second_artifact.txt`.
- `state/orquesta_server_state_v0.json` quedo `status=stopped`,
  `shutdown_status=stopped`, `shutdown_ready=true`; comprobacion de procesos
  posterior sin `orquesta-server run`, `codex app-server` ni sesion
  `orquesta-goal-e6a74570c0641004` vivos.
- Higiene: el temporal retenido
  `/tmp/orquesta-goal-first-app-server.gibwtZ` se saneo borrando
  `bin/orquesta-server` y `runtime/goal-srv/codex-home` para no conservar
  credenciales/cache; quedan JSON/logs/estado/artefactos (~256 KiB).

Archivos tocados en este avance:

- `docs/runbooks/smoke_goal_first_checkpoint_only_high_consumption_real_2026-07-03.md`
- `docs/inventario_bugs_orquesta_2026-06-30.md`
- `docs/bitacora_correccion_pericial_2026-07-03.md`

Pendiente real: la reejecucion reduce `BUG-079`, `BUG-165` y `BUG-065/076` en
la ruta alto consumo -> artefacto recuperable -> cleanup, pero no cierra el
caso Sueldos de forced stop con backend vivo ni el enforcement duro previo a
herramientas.

## Continuacion Codex 2026-07-04 noche 11

Cierre de la ruta Sueldos de `BUG-ORQ-20260704-165`:

- Se anade `scripts/smoke_goal_first_forced_stop_backend_real.sh`, wrapper real
  opt-in para alto consumo + backend `app_server_tmux` vivo + forced stop.
- El modo del smoke usa `SMOKE_GOAL_FIRST_FORCED_STOP_MODE=1`, no una variable
  `ORQUESTA_*`, para no romper el ratchet MEJ-106.
- `runs/control` completa `stopped/canceled` si el observer ya dejo el goal en
  estado terminal/rework antes del control forzado. Esto cubre el caso
  `goal_status_before=blocked` que antes quedaba en `stop_requested`.
- El backend `app_server_tmux` expone `ShutdownForcedStopV0` y usa timeout corto
  con cleanup de proceso/socket en contexto fresco, evitando que un wait
  agotado deje el backend vivo.
- El smoke valida que el observe posterior no publique `running`: exige
  `goal_status=blocked`, `closure_status=blocked` y
  `recommended_action=replan`.

Smoke real final:

```text
smoke_goal_first_forced_stop_backend_real=ok
run_ref=run-spec-smoke-goal-first-bug088-req-smoke-goal-first-bug088-116f51fcff09efa9aa525ee7dfd94ce7
goal_ref=goal-ref-app-director-run-spec-smoke-goal-first-bug088-req-smoke-goal-first-bug088-116f51fcff09efa9aa525ee7dfd94ce7
external_goal_ref=019f2c28-d0c3-7551-9972-dcba0f3daeb2
run_control_estado=ok
run_control_status=stopped
run_control_final_status=stopped
run_control_goal_status_after=blocked
observe_after_forced_stop_goal_status=blocked
observe_after_forced_stop_closure_status=blocked
observe_after_forced_stop_recommended_action=replan
app_server_tmux_processes_alive=0
smoke_root=/tmp/orquesta-goal-first-app-server.Sc7e7K
```

Evidencia saneada: se borraron `runtime/goal-srv/codex-home` y
`bin/orquesta-server` del temporal retenido. Quedan JSON/logs/estado/artefactos
compactos (~296 KiB) y no quedan `orquesta-server run`, `codex app-server` ni
tmux `orquesta-goal-*` vivos.

Archivos tocados en este avance:

- `scripts/smoke_goal_first_app_server_real.sh`
- `scripts/smoke_goal_first_forced_stop_backend_real.sh`
- `cmd/orquesta-server/smoke_goal_first_script_guard_v0_test.go`
- `modulos/orquesta-app-codex-stack/goal_first_run_control_v0.go`
- `modulos/orquesta-app-codex-stack/goal_first_run_control_v0_test.go`
- `modulos/orquesta-runtime-codex-appserver/codex_goal_app_server_stop_v0.go`
- `modulos/orquesta-runtime-codex-appserver/codex_goal_app_server_tmux_v0.go`
- `modulos/orquesta-runtime-codex-appserver/codex_goal_app_server_migrated_v0_test.go`
- `docs/runbooks/smoke_goal_first_forced_stop_backend_real_2026-07-04.md`
- `docs/incidencia_sueldos_goal_first_app_invalid_checkpoint_2026-07-04.md`
- `docs/inventario_bugs_orquesta_2026-06-30.md`
- `docs/bitacora_correccion_pericial_2026-07-03.md`

Verificacion:

- `find scripts -name '*.sh' -print0 | xargs -0 -n1 bash -n`
- `go test -count=1 ./ -run TestEnvVarsBudget`
- `go test -count=1 ./cmd/orquesta-server -run 'TestSmokeGoalFirst(AppServerReal|HighConsumption|ForcedStop)'`
- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestGoalFirstRunControl|TestCodexStackRunControl'`
- `go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPRunControl'`
- `go test -count=1 ./modulos/orquesta-runtime-codex-appserver -run 'TestServerCodexAppServerGoalBackendV0StopForced|TestCodexAppServerTmuxBackendV0EnsureShutdownCleanup'`
- `go test -count=1 ./modulos/orquesta-runtime-codex-appserver ./modulos/orquesta-app-codex-stack ./modulos/orquesta-mcp ./cmd/orquesta-server`
- `go test -count=1 ./...`
- `git diff --check`

Pendiente real: `BUG-165` queda abierto solo para residuales amplios de
`status/observe` lento y coordinacion automatica completa
shutdown/backend/checkpoint/stop/cancel/wait. La ruta Sueldos forced stop con
backend vivo queda cerrada.

## Continuacion Codex 2026-07-04 noche 12

Avance de conectores Claude/Gemini hacia paridad de contrato durable:

- `modulos/orquesta-runtime-claude` y `modulos/orquesta-runtime-gemini` siguen
  siendo conectores CLI opt-in de proceso externo, no backend goal-first
  residente equivalente a Codex.
- Se refuerza el prompt de ambos conectores para que, si objetivo, criterios o
  tests piden `goal-first`, `result durable`, `orquesta_goal_result_v0.json` u
  `ORQUESTA_GOAL_RESULT_V0`, escriban un JSON dentro del write-set con
  `schema_version=orquesta_goal_result.v0`, estado terminal,
  `artifact_paths`, `materialized_artifacts`, `checklist`,
  `required_test_results` y evidencias compactas.
- Se anade en `orquesta-runtime-claude` un primer backend goal-first offline
  de fichero/control: implementa `GoalWorkLauncherPortV0` y
  `GoalWorkObservationPortV0`, escribe spec/prompt en runtime aislado y observa
  `orquesta_goal_result*.json` bajo el write-set del proyecto.
- El protocolo explicita que no debe escribirse el resultado en
  `runtime_work_dir` salvo que el write-set lo permita, que no se oculten
  artefactos fuera de scope y que faltas de QA/evidencia cierren como
  `blocked/invalid`.
- No cambia el default Codex ni requiere credenciales Claude/Gemini.

Archivos tocados en este avance:

- `modulos/orquesta-runtime-claude/claude_prompt_v0.go`
- `modulos/orquesta-runtime-claude/claude_goal_backend_v0.go`
- `modulos/orquesta-runtime-claude/claude_goal_paths_v0.go`
- `modulos/orquesta-runtime-claude/claude_goal_backend_v0_test.go`
- `modulos/orquesta-runtime-claude/claude_resolver_v0_test.go`
- `modulos/orquesta-runtime-gemini/gemini_prompt_v0.go`
- `modulos/orquesta-runtime-gemini/gemini_resolver_v0_test.go`
- `docs/plan_mejora_continua_orquesta_2026-07-04.md`
- `docs/bitacora_correccion_pericial_2026-07-03.md`

Verificacion focal:

- `go test -count=1 ./modulos/orquesta-runtime-claude ./modulos/orquesta-runtime-gemini`

Pendiente real: MEJ-103 no queda cerrado. Falta cablear seleccion opt-in por
composicion/env, proceso real Claude, state/shutdown/control equivalentes al
backend Codex y smoke fake/real opt-in; Gemini queda pendiente de backend
goal-first propio.

## Continuacion Codex 2026-07-04 noche 13

Avance OPES `question_bank` para reducir falsos verdes de `BUG-058/075`:

- Se anade contrato puro `ValidateOPESQuestionBankQualityContractV0` en
  `modulos/orquesta-opes-director`: valida banco por tema con minimo 50
  preguntas, 4 opciones, una unica respuesta correcta resoluble, explicacion
  tutor, informe estructural, informe de dificultad/proximidad y revision triple
  Codex/Gemini/Claude.
- Se cablea en `update_topic_registry` y settlement: si una entrega
  `question_bank` trae evidencia nominal `question_bank_publicable` pero falla
  el contrato, no queda publicable; publica
  `proposed_status=pendiente_rework_tests`,
  `operational_status=needs_rework`,
  `settlement_scope=question_bank_quality` y crea rework causal
  `review_director_consolidation` con
  `recommended_action=review_question_bank_quality`.
- Se conserva la precedencia previa: si falta toda la evidencia minima, el
  camino primario sigue siendo `required_evidence_missing`, no un falso fallo
  estructural de preguntas.

Archivos tocados en este avance:

- `modulos/orquesta-opes-director/question_bank_quality_contract_v0.go`
- `modulos/orquesta-opes-director/question_bank_quality_helpers_v0.go`
- `modulos/orquesta-opes-director/question_bank_payload_v0.go`
- `modulos/orquesta-opes-director/topic_registry_question_bank_quality_v0.go`
- `modulos/orquesta-opes-director/topic_registry_settlement_v0.go`
- `modulos/orquesta-opes-director/topic_registry_v0.go`
- `modulos/orquesta-opes-director/job_requests_v0.go`
- `modulos/orquesta-opes-director/question_bank_quality_contract_v0_test.go`
- `modulos/orquesta-opes-director/producer_v0_test.go`
- `docs/inventario_bugs_orquesta_2026-06-30.md`
- `docs/bitacora_correccion_pericial_2026-07-03.md`

Verificacion focal:

- `go test -count=1 ./modulos/orquesta-opes-director`
- `go test -count=1 ./modulos/orquesta-opes-director ./modulos/orquesta-opes-bridge`
- `go test -count=1 ./...`
- `git diff --check`

Pendiente real: `BUG-058/075` no queda cerrado por completo. Faltan smoke OPES
temporal end-to-end y validadores semanticos/editoriales completos por
artefacto canonico; este corte cierra el falso verde mecanico de banco de
preguntas con evidencia nominal.

## Continuacion Codex 2026-07-04 noche 14

Avance MEJ-103: backend goal-first Claude cableado por composicion/env en el
servidor sin cambiar el default Codex.

- Se anade `ORQUESTA_CODEX_GOAL_BACKEND=claude_file_control` como opt-in
  explicito para Claude goal-first sin crear una variable `ORQUESTA_*` nueva;
  esto respeta el ratchet MEJ-106 de presupuesto de configuracion.
- `serverCodexGoalBackendV0` acepta ahora puertos neutrales
  `GoalWorkLauncherPortV0`/`GoalWorkObservationPortV0`; Codex conserva sus
  puertos especificos y Claude entra por el contrato neutral.
- `cmd/orquesta-server` materializa el backend
  `orquesta-runtime-claude.ClaudeGoalBackendV0` para app goal e idle goal,
  usando runtime de control fuera del proyecto por defecto cuando no se define
  `ORQUESTA_CLAUDE_RUNTIME_WORKDIR`.
- La automejora goal-first se deriva tambien de Claude cuando
  `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_GOAL_FIRST_ENABLED` no esta fijada; el
  effective config publica el valor `claude_file_control` en
  `ORQUESTA_CODEX_GOAL_BACKEND` y el diagnostico de derivacion.
- Valores de backend no soportados siguen cortando en arranque con diagnostico
  explicito; Codex no cambia cuando el valor es `app_server_tmux`.

Archivos tocados en este avance:

- `cmd/orquesta-server/codex_goal_app_server_v0.go`
- `cmd/orquesta-server/codex_goal_backend_env_v0.go`
- `cmd/orquesta-server/claude_runtime_config_v0.go`
- `cmd/orquesta-server/config.go`
- `cmd/orquesta-server/effective_config_v0.go`
- `cmd/orquesta-server/codex_goal_app_server_wiring_v0_test.go`
- `cmd/orquesta-server/config_test.go`
- `docs/plan_mejora_continua_orquesta_2026-07-04.md`
- `docs/bitacora_correccion_pericial_2026-07-03.md`

Verificacion focal inicial:

- `go test -count=1 ./cmd/orquesta-server -run 'Test(ServerGoalBackendFromEnvV0ClaudeFileControlExponePuertosNeutrales|ServerGoalBackendFromEnvV0RechazaBackendNoSoportado|ServerConfigFromEnvV0DerivaAutomejoraGoalFirstDeBackend(Claude|Codex)Goal|ServerEffectiveConfigV0ExponeGoalFirstYBackend)V0'`
- `go test -count=1 ./modulos/orquesta-runtime-claude`

Pendiente real: MEJ-103 aun no se cierra al 100%. Queda lanzar proceso Claude
real supervisado, control/shutdown equivalente al backend Codex, smoke fake/real
opt-in de lanzamiento completo y backend goal-first propio para Gemini.

## Continuacion Codex 2026-07-04 noche 15

Avance MEJ-103: backend Claude goal-first con proceso supervisado.

- Se anade `ClaudeGoalProcessBackendV0` en `orquesta-runtime-claude`.
- El backend reutiliza `ClaudeGoalBackendV0` para spec/prompt/result durable y
  lanza un wrapper CLI por goal con `ProcessRuntimeConnectorV0`.
- Nuevo selector sin env adicional:
  `ORQUESTA_CODEX_GOAL_BACKEND=claude_process`.
- El wrapper ejecuta `ORQUESTA_CLAUDE_COMMAND -p` con el prompt del goal por
  stdin y stdout/stderr redirigidos a runtime de control.
- Si el proceso termina sin materializar `orquesta_goal_result_v0.json`, la
  observacion devuelve `blocked` con
  `claude_goal_process_stopped_without_result`; no queda como `running` falso.
- `cmd/orquesta-server` cablea `claude_process` para app goal e idle goal y
  conserva `claude_file_control` como modo sin proceso.

Archivos tocados en este avance:

- `modulos/orquesta-runtime-claude/claude_goal_process_backend_v0.go`
- `modulos/orquesta-runtime-claude/claude_goal_process_backend_v0_test.go`
- `modulos/orquesta-runtime-claude/claude_wrapper_v0.go`
- `cmd/orquesta-server/claude_runtime_config_v0.go`
- `cmd/orquesta-server/codex_goal_backend_env_v0.go`
- `cmd/orquesta-server/effective_config_v0.go`
- `cmd/orquesta-server/codex_goal_app_server_wiring_v0_test.go`
- `docs/plan_mejora_continua_orquesta_2026-07-04.md`
- `docs/bitacora_correccion_pericial_2026-07-03.md`

Verificacion focal inicial:

- `go test -count=1 ./modulos/orquesta-runtime-claude`
- `go test -count=1 ./cmd/orquesta-server -run 'TestServerGoalBackendFromEnvV0(ClaudeFileControlExponePuertosNeutrales|ClaudeProcessLanzaYObservaResultado|RechazaBackendNoSoportado)V0|TestServerConfigFromEnvV0DerivaAutomejoraGoalFirstDeBackendClaudeGoalV0'`

Pendiente real: MEJ-103 queda reducido, no cerrado total. Faltan control y
shutdown persistentes equivalentes a Codex app-server, smoke opt-in contra
Claude real con credenciales y backend goal-first propio para Gemini.

## Continuacion Codex 2026-07-04 noche 16

Avance MEJ-103: control/stop para `claude_process` en la instancia viva del
servidor.

- `ClaudeGoalProcessBackendV0` anade `StopClaudeGoalV0` con request/result
  propios, sin depender de `orquesta-app-codex-stack`.
- El stop localiza el `process_ref` del `goal_ref`, llama a
  `ProcessRuntimeConnectorV0.StopV0` y devuelve `status=blocked`,
  `goal_status_set=true`, `backend_stopped=true`, `issue_code` si falla y
  evidencias compactas `claude-goal-process-stop-*`.
- `cmd/orquesta-server` adapta ese resultado al `GoalBackendControlPortV0`, el
  mismo puerto usado por run-control goal-first para stops forzados.
- Se anaden pruebas de módulo y servidor con proceso fake largo, verificando
  que el control para el proceso y publica evidencia de stop completado.

Archivos tocados en este avance:

- `modulos/orquesta-runtime-claude/claude_goal_process_backend_v0.go`
- `modulos/orquesta-runtime-claude/claude_goal_process_backend_v0_test.go`
- `cmd/orquesta-server/codex_goal_app_server_v0.go`
- `cmd/orquesta-server/codex_goal_control_adapter_v0.go`
- `cmd/orquesta-server/codex_goal_backend_env_v0.go`
- `cmd/orquesta-server/codex_goal_app_server_wiring_v0_test.go`
- `docs/plan_mejora_continua_orquesta_2026-07-04.md`
- `docs/bitacora_correccion_pericial_2026-07-03.md`

Verificacion focal inicial:

- `go test -count=1 ./modulos/orquesta-runtime-claude`
- `go test -count=1 ./cmd/orquesta-server -run 'TestServerGoalBackendFromEnvV0ClaudeProcess(ControlParaProceso|LanzaYObservaResultado)V0|TestServerGoalBackendFromEnvV0ClaudeFileControlExponePuertosNeutralesV0'`

Pendiente real: el stop ya existe para la instancia viva; falta persistir o
adoptar procesos Claude tras reinicio, smoke opt-in contra Claude real con
credenciales y backend goal-first propio para Gemini.

## Continuacion Codex 2026-07-04 noche 17

Avance MEJ-103/T18: backend goal-first Gemini con paridad fake/offline frente
al corte Claude.

- `orquesta-runtime-gemini` anade `GeminiGoalBackendV0` file-control: escribe
  spec/prompt en runtime aislado y observa resultados durables
  `orquesta_goal_result_v0.json` o `orquesta_goal_result_<goal>.json` dentro
  del write-set.
- `GeminiGoalProcessBackendV0` lanza un wrapper Gemini por goal usando
  `ProcessRuntimeConnectorV0`, conserva refs de proceso en la instancia viva,
  devuelve `blocked` si el proceso termina sin result durable y expone
  `StopGeminiGoalV0` para control/stop.
- `cmd/orquesta-server` reconoce sin variables nuevas:
  `ORQUESTA_CODEX_GOAL_BACKEND=gemini_file_control` y
  `ORQUESTA_CODEX_GOAL_BACKEND=gemini_process`.
- La automejora goal-first se deriva tambien desde backend Gemini cuando
  `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_GOAL_FIRST_ENABLED` no esta fijada.
- El control Gemini se adapta al mismo `GoalBackendControlPortV0` usado por
  Codex/Claude, de forma que run-control puede parar un `gemini_process` activo
  en la misma instancia.

Archivos tocados en este avance:

- `modulos/orquesta-runtime-gemini/gemini_goal_backend_v0.go`
- `modulos/orquesta-runtime-gemini/gemini_goal_backend_v0_test.go`
- `modulos/orquesta-runtime-gemini/gemini_goal_process_backend_v0.go`
- `modulos/orquesta-runtime-gemini/gemini_goal_process_backend_v0_test.go`
- `modulos/orquesta-runtime-gemini/gemini_wrapper_v0.go`
- `cmd/orquesta-server/gemini_runtime_config_v0.go`
- `cmd/orquesta-server/codex_goal_backend_env_v0.go`
- `cmd/orquesta-server/codex_goal_app_server_v0.go`
- `cmd/orquesta-server/codex_goal_control_adapter_v0.go`
- `cmd/orquesta-server/effective_config_v0.go`
- `cmd/orquesta-server/codex_goal_app_server_wiring_v0_test.go`
- `cmd/orquesta-server/config_test.go`
- `docs/plan_mejora_continua_orquesta_2026-07-04.md`
- `docs/bitacora_correccion_pericial_2026-07-03.md`

Verificacion focal inicial:

- `go test -count=1 ./modulos/orquesta-runtime-gemini`
- `go test -count=1 ./cmd/orquesta-server -run 'TestServerGoalBackendFromEnvV0Gemini|TestServerConfigFromEnvV0DerivaAutomejoraGoalFirstDeBackendGeminiGoalV0'`

Pendiente real tras este tramo: smoke opt-in contra Gemini real con
credenciales/tier valido, smoke opt-in contra Claude real con credenciales,
persistencia/adopcion de procesos Claude/Gemini tras reinicio y prueba real
amplia de shutdown/control con proveedor externo.
