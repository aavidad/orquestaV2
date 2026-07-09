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

### 2026-07-09 — TAREA-E1 deploy atomico local

Implementado pipeline local `scripts/orquesta_server_deploy.sh`: sync
fast-forward-only de worktree por ref, build desde arbol exportado, sha256,
backup y swap de binario, validacion opcional de config canonica, arranque via
`scripts/orquesta_server_ctl.sh start`, verificacion de identidad runtime si el
status/readiness/supervisor expone hash, y receipt JSON durable en state dir.
Cobertura: `scripts/test_orquesta_server_deploy.sh` con fakes para success,
`deploy_config_missing`, `deploy_not_fast_forward` y
`deploy_runtime_identity_mismatch`; guard Go para impedir copias/arranques
gestionados fuera de `ctl`/`deploy`; runbook
`docs/runbooks/orquesta_server_deploy_atomico_2026-07-09.md`.

### 2026-07-09 — BUG-ORQ-20260709-202 smoke recursivo alineado con guard unmanaged

Durante la verificacion local inside-out, `go test -count=1 ./...` paso y el
smoke de ola operativa fake paso, pero
`scripts/smoke_codex_director_recursive_wave.sh` fallo con
`unmanaged_launch_blocked`. La causa no era el nucleo recursivo: el comando
`codex-launch-director-wave` ya exige breakglass auditado para launches fuera
del servidor/cola de Orquesta y confirmacion explicita para `--purge-runtime`.
Se actualizo el smoke para declarar `--allow-unmanaged-launch`, razon auditada,
confirmacion unmanaged por `wave_ref` y confirmacion de purga por `wave_ref`.
Tambien se alineo el validador con el resumen publico `refs_only`: las rutas
internas se comprueban leyendo los registries locales por `wave_ref`, y el
wrapper acepta `medium`/`high` sin permitir `xhigh`.

### 2026-07-09 — BUG-ORQ-20260709-203 smokes de conectores actualizados

Al pasar de nucleo a conectores locales, fallaron dos smokes por contrato
antiguo. `smoke_external_domain_fake_real.sh` esperaba 400 en
`submit_artifact`, pero el backend `orquesta-domain-work-file` ya tiene
submitter local y devuelve receipt aceptado. `smoke_external_domain_non_opes_real.sh`
arrancaba adaptador HTTP sin declarar `ORQUESTA_DOMAIN_WORK_HTTP_EGRESS_MODE`,
lo que ahora bloquea correctamente con `domain_work_http_egress_policy_required`.
Se actualizaron los smokes para validar receipt/snapshot file-based y para
usar `smoke_local` en el adaptador HTTP temporal. En ambos casos se declara
tambien `ORQUESTA_AUTOPROGRAMMING_LEGACY_DIRECTOR_LOOP=true`, requisito vigente
del supervisor legacy que estos smokes aun ejercitan. El fake file-based escribe
ahora `external_summary.md` en sus rutas de entrega para que el intake de
DomainWork seleccione el artefacto por tipo cuando hay varios ficheros.

### 2026-07-09 — BUG-ORQ-20260709-204 E5 wizard/i18n sin falsos verdes

Siguiendo la orden de arreglar de dentro hacia afuera, se reviso primero el
wizard de Nueva App antes de seguir con conectores/remoto. E5 tenia dos
residuales reales: la lista de textos i18n prohibidos estaba copiada dentro de
un helper del wizard, y el aplicador de decisiones reemplazaba listas
acumulativas. Con respuestas T7 resiliencia + T8 cumplimiento en el mismo
batch, `agentes.preferencias` podia perder la decision anterior.

Cierre local: las listas acumulativas de restricciones/preferencias/compliance
y locales se mezclan con dedupe; T8 declara destino primario
`calidad.compliance`; el selector de preguntas de un turno evita campos destino
repetidos; los tests cubren unicidad de campo en turno visible y en el catalogo
tecnico T1-T8; y la prohibicion de placeholders se centraliza para aplicarse a
todo `NuevaAppI18nRequiredKeysV0`, no solo al wizard. Evidencia:
`go test -count=1 ./modulos/orquesta-web` y `git diff --check`.

### 2026-07-09 — BUG-ORQ-20260709-205 E4 pausa local por proveedor caido

Revision inside-out de E4: la clasificacion
`codex_app_server_provider_unauthorized` ya existia, pero el bloqueador de
automejora no actuaba porque `idleSelfImprovementRunHasProviderAuthBlockerV0`
devolvia siempre `false`. Se arreglo para detectar codigos canonicos de
provider/auth/cuota en proyecciones del run, sin inferir por assessments
criticos genericos. La pausa publica queda como
`provider_unavailable_paused`.

Tambien se extendio `operator_notifications.v0` con evento `provider` y un
metodo local de servidor que envia un aviso deduplicado por
`reason_code+run_ref` con accion de reauth/cuota (`hermes auth`, `hermes model`
o restaurar `CODEX_HOME` aislado segun caso). Evidencia focal:
`go test -count=1 ./cmd/orquesta-server -run 'TestIdleSelfImprovementBlockersV0|TestOperatorNotificationServerV0Notifica'`,
`go test -count=1 ./modulos/orquesta-server -run 'TestRuntimeV0SupervisorNoPreparaAutomejoraConProveedorAuthBloqueadoV0'`
y `go test -count=1 ./modulos/orquesta-operator-notifications`. Residual:
wiring/smoke Telegram remoto real se difiere a fase remota.

Ampliacion local del mismo corte: `serverStackSupervisorV0` tiene ahora un
notifier opcional construido desde la configuracion canonica Telegram
(`token` + `notification_target_ref`). Cuando
`IdleSelfImprovementBlockersV0` devuelve `provider_unavailable_paused`, invoca
`NotifyProviderIssueV0` y anade evidencias de notificacion al blocker. Sin
sender configurado es no-op; si el envio falla, no anula la pausa. Evidencia:
`TestIdleSelfImprovementBlockersV0NotificaProviderPausadoUnaVezV0` y
`go test -count=1 ./cmd/orquesta-server -run 'TestIdleSelfImprovementBlockersV0|TestOperatorNotificationServerV0Notifica|TestOperatorNotificationHermesTelegramNotifierV0UsaSendSinAuthCodex|TestTelegramBotAPI'`.

### 2026-07-09 — BUG-079 observacion activa de stdout gigante

Inside-out adicional antes de volver a remoto: el protocolo ya tenia tests
directos para respuesta `thread/read` sobredimensionada, pero faltaba fijar que
la capa superior de observacion no silenciase ese error si el goal seguia
activo. Se cambio `observeCodexAppServerActiveGoalResultV0` para que solo el
caso estructural `codex_app_server_thread_read_response_too_large` bloquee la
observacion activa con ese issue exacto y evidencia
`evidence-ref-codex-app-server-active-goal-thread-read-failed`; si existe
resultado durable en fichero, ese resultado sigue teniendo prioridad.

Evidencia focal:
`go test -count=1 ./modulos/orquesta-runtime-codex-appserver`. Esto no cierra
`BUG-079`: sigue faltando demostrar con proveedor/app-server real que el limite
se aplica antes de que una herramienta emita stdout gigante. Si no se cumple,
queda como frontera externa del runtime, no como fallo del nucleo puro.

### 2026-07-09 — TAREA-E3 inventario focal MCP interno

Avance inside-out de E3 sin tocar remoto ni conectores externos. El test global
de MCP ya impedia registrar tools sin DTO canonico y detectaba campos stale
anunciados por `input_schema`, pero no detectaba el caso contrario: campo nuevo
en DTO y descriptor sin actualizar. Se anadio un inventario focal para
superficies internas: Nueva App (`orquesta.apps.solicitar_nueva.v0`,
`orquesta.nueva_app.wizard.v0`, `orquesta.nueva_app.wizard.bot.v0`),
`orquesta.autoprogramming.prepare_run.v0`,
`orquesta.autoprogramming.status.v0` y
`orquesta.operator.director.message.v0`. El guard exige que esas tools esten
inventariadas y que todos sus campos DTO aparezcan en el descriptor. En el
cierre local se alinearon los `input_schema` de autoprogramming status y del
canal operador-Director.

Evidencia:
`go test -count=1 ./modulos/orquesta-mcp -run 'TestMCP(TransportToolInputSchema|InternalContractSurfaceInventory)'`
y `go test -count=1 ./modulos/orquesta-mcp`.
Continuacion local: Telegram se cubre en el adaptador puro
`modulos/orquesta-operator-telegram`, sin tocar bot remoto ni credenciales.
`CommandCatalogV0` pasa a ser el inventario canonico de comandos/aliases y
`ParseCommandV0` lo usa; el test
`TestCommandCatalogV0CubreComandosPublicosYParserV0` valida comandos publicos,
aliases duplicados y metadatos de seguridad (`stop` con confirmacion,
`director_message` con destino y cuerpo).

Evidencia adicional:
`go test -count=1 ./modulos/orquesta-operator-telegram`.
Residual: falta extender el inventario a HTTP/web cuando esa superficie se
trabaje. La prueba real de Telegram remoto queda fuera de este cierre interno.

### 2026-07-09 — checkpoint dentro-fuera local

Revision de prioridad tras el cierre E3/E4/E5 local y la orden del operador de
arreglar dentro hacia afuera:

- Capa 1 nucleo/contratos puros: E3 queda reducido a HTTP/web residual; E5
  queda cerrado local; subagentes/causalidad no muestra un bug puro nuevo sin
  entrar en backend/OPES. Tests y build globales verdes.
- Capa 2 composicion/adaptadores locales: E4 queda reducido a frontera de probe
  real de proveedor; BUG-079/200 ya tiene transporte `toolOutputPolicy`,
  budgets de respuestas y harness adversarial, pero el enforcement pre-tool
  real sigue dependiendo del app-server/proveedor. BUG-065/165 conserva
  residuales de smoke amplio con proveedor lento/stale, no de unit local.
- Capa 3 conectores externos/OPES: siguen pendientes BUG-058/066/075 en
  field tests OPES temporal/preprod, no en core.
- Capa 4 remoto/Hermes/Telegram/deploy: E1/E2, auth/cuota, Git remoto y bot real
  quedan al final por instruccion vigente.

Verificado tras este checkpoint:
`git diff --check`, `go test -count=1 ./...` y `go build ./...`.

### 2026-07-09 — E6 guard scripts revisado como residual no bloqueante

Subagente Godel reviso E6 en modo read-only. Resultado: los guards funcionales
estan verdes, pero la consolidacion no esta hecha como tabla unica. Tests
actuales cubren shutdown comun, cleanup delegado, `runtime_dir`, shutdown HTTP
directo, endpoint gestionado, puerto historico y arranque/copia de servidor
fuera de `ctl/deploy`, repartidos en
`cmd/orquesta-server/smoke_goal_first_script_guard_v0_test.go`. No bloquea el
cierre local de nucleo/conectores; queda como limpieza futura de test:
`scriptGuardContractV0` + tabla `script -> contratos exigidos`.

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

## Continuacion Codex 2026-07-04 noche 18

Avance MEJ-103/T18: persistencia y adopcion tras reinicio para procesos
Claude/Gemini goal-first.

- `claude_process` y `gemini_process` persisten un manifiesto interno por goal
  bajo su runtime aislado con `goal_ref`, refs opacas de
  `ProcessRuntimeConnectorV0` y PID solo interno.
- Si el servidor se reconstruye y el mapa en memoria esta vacio, el backend
  carga el manifiesto, usa `ProcessRuntimeConnectorV0.AdoptProcessV0` y recupera
  el control del proceso vivo antes de observar o parar.
- El PID no se publica en specs, prompts, issues ni evidencias publicas; solo
  queda en el fichero de control interno del runtime.
- Se anade evidencia `*-goal-process-adopted` en los resultados de observe/stop
  cuando la adopcion ocurre.

Archivos tocados en este avance:

- `modulos/orquesta-runtime-claude/claude_goal_process_backend_v0.go`
- `modulos/orquesta-runtime-claude/claude_goal_process_backend_v0_test.go`
- `modulos/orquesta-runtime-gemini/gemini_goal_process_backend_v0.go`
- `modulos/orquesta-runtime-gemini/gemini_goal_process_backend_v0_test.go`
- `docs/plan_mejora_continua_orquesta_2026-07-04.md`
- `docs/bitacora_correccion_pericial_2026-07-03.md`

Verificacion focal inicial:

- `go test -count=1 ./modulos/orquesta-runtime-claude ./modulos/orquesta-runtime-gemini`

Pendiente real tras este tramo: smoke opt-in contra Claude real con
credenciales, smoke opt-in contra Gemini real con credenciales/tier valido y
prueba real amplia de shutdown/control con proveedor externo.

## Continuacion Codex 2026-07-04 noche 19

Avance MEJ-103/T18: smoke real opt-in de `claude_process`.

- Preflight local: `claude --version` devuelve Claude Code `2.1.201` y
  `gemini --version` devuelve `0.45.1`.
- Claude real responde `OK` con `claude -p --model sonnet
  --permission-mode bypassPermissions --output-format text --max-budget-usd
  0.20`.
- Gemini real queda bloqueado por `IneligibleTierError / UNSUPPORTED_CLIENT`;
  esto coincide con `BUG-ORQ-20260703-140` y con el diagnostico estructurado ya
  implementado `provider_auth_or_tier_blocked`.
- Primer intento real Claude encontro un fallo de contrato blando: el fichero
  durable podia no ser JSON puro o podia omitir `artifact_ref` en
  `materialized_artifacts`. Se endurecio el protocolo de prompt para exigir JSON
  puro sin markdown/fences y `artifact_ref` no vacio.
- Segundo intento Claude real pasa con `status=complete`, artefacto material,
  result durable parseable, `artifact_refs`, `materialized_artifacts` completos
  y `required_test_results=passed`.

Archivos tocados en este avance:

- `modulos/orquesta-runtime-claude/claude_prompt_v0.go`
- `modulos/orquesta-runtime-claude/claude_goal_process_backend_v0_test.go`
- `modulos/orquesta-runtime-gemini/gemini_prompt_v0.go`
- `modulos/orquesta-runtime-gemini/gemini_goal_process_backend_v0_test.go`
- `docs/runbooks/smoke_goal_first_provider_process_real_2026-07-04.md`
- `docs/plan_mejora_continua_orquesta_2026-07-04.md`
- `docs/bitacora_correccion_pericial_2026-07-03.md`

Verificacion:

- `go test -count=1 ./modulos/orquesta-runtime-claude ./modulos/orquesta-runtime-gemini`
- `SMOKE_CLAUDE_GOAL_PROCESS_REAL=1 SMOKE_CLAUDE_MAX_BUDGET_USD=0.50 SMOKE_CLAUDE_KEEP_DIR=1 go test -count=1 ./modulos/orquesta-runtime-claude -run TestClaudeGoalProcessBackendV0RealOptInEscribeResultadoDurableV0 -v`

Evidencia retenida:

- `/tmp/orquesta-claude-goal-real-smoke-3215275659/project/docs/provider_goal_smoke.txt`
- `/tmp/orquesta-claude-goal-real-smoke-3215275659/project/docs/orquesta_goal_result_v0.json`

Pendiente real tras este tramo: smoke Gemini con tier valido y prueba real
amplia a traves de `cmd/orquesta-server`/run-control con proveedor externo.

## Continuacion Codex 2026-07-04 noche 20

Avance MEJ-103/T18 y `BUG-ORQ-20260704-165`: smoke real de `claude_process`
atravesando `cmd/orquesta-server`.

- Nuevo smoke opt-in
  `scripts/smoke_goal_first_claude_process_server_real.sh`: arranca
  `orquesta-server` temporal con `ORQUESTA_CODEX_GOAL_BACKEND=claude_process`,
  automejora/resident observer desactivados, proyecto/runtime aislados,
  lanzamiento por `POST /api/v0/apps/director`, observacion por
  `POST /api/v0/apps/director/goal/observe` y cleanup con
  `cleanup_goal_backends=true` mas fallback por manifiesto interno Claude.
- Resultado real aceptado: `smoke_goal_first_claude_process_server_real=ok`,
  `poll=31`, `goal_status=complete`, `run_status=cerrada`,
  `closure_status=accepted`, `closure_accepted=true`,
  `recommended_action=no_action_closed`.
- Evidencia retenida:
  `/tmp/orquesta-claude-process-server.x5N8pq`,
  `run_ref=run-spec-smoke-claude-process-server-req-smoke-claude-process-server-389d151256b512f15e130b3042aa99fe`,
  `external_goal_ref=claude-goal-c51e3c2c56c62b53`.
- Metricas del cierre: `observe_response.json` contiene 9 `artifact_refs` y
  16 `evidence_refs`; `orquesta_goal_result_v0.json` contiene 3 refs
  requeridas, 13 `artifact_paths`, 3 `materialized_artifacts` y 1
  `required_test_results` `passed`.
- Incidencia recuperable cerrada: Claude real puede devolver `evidence_refs`
  como objetos `{ref, description}`; Claude/Gemini normalizan ahora solo esa
  forma recuperable y los prompts exigen arrays de strings.
- Incidencia operativa cerrada: un intento sin `--safe-mode` heredo
  configuracion local de Claude y lanzo `codebase-memory-mcp`; el smoke usa
  `SMOKE_CLAUDE_GOAL_PROCESS_SERVER_SAFE_MODE=1` por defecto y el wrapper
  retenido muestra `claude -p ... --safe-mode`. La comprobacion posterior no
  encontro procesos residuales de `orquesta-server run`, wrapper Claude,
  `claude -p`, `codebase-memory-mcp`, `codex app-server` ni
  `orquesta-goal-*`.

Archivos tocados en este avance:

- `scripts/smoke_goal_first_claude_process_server_real.sh`
- `cmd/orquesta-server/smoke_goal_first_script_guard_v0_test.go`
- `modulos/orquesta-runtime-claude/claude_goal_backend_v0.go`
- `modulos/orquesta-runtime-claude/claude_goal_backend_v0_test.go`
- `modulos/orquesta-runtime-claude/claude_prompt_v0.go`
- `modulos/orquesta-runtime-gemini/gemini_goal_backend_v0.go`
- `modulos/orquesta-runtime-gemini/gemini_goal_backend_v0_test.go`
- `modulos/orquesta-runtime-gemini/gemini_prompt_v0.go`
- `docs/runbooks/smoke_goal_first_provider_process_real_2026-07-04.md`
- `docs/plan_mejora_continua_orquesta_2026-07-04.md`
- `docs/inventario_bugs_orquesta_2026-06-30.md`
- `docs/bitacora_correccion_pericial_2026-07-03.md`

Verificacion ejecutada en este tramo:

- `bash -n scripts/smoke_goal_first_claude_process_server_real.sh`
- `go test -count=1 ./cmd/orquesta-server -run TestSmokeGoalFirstClaudeProcessServerRealEsOptInYLimpiaBackendV0`
- `go test -count=1 ./modulos/orquesta-runtime-claude ./modulos/orquesta-runtime-gemini`
- `SMOKE_CLAUDE_GOAL_PROCESS_SERVER_REAL=1 SMOKE_CLAUDE_GOAL_PROCESS_SERVER_SKIP_PREFLIGHT=1 SMOKE_CLAUDE_GOAL_PROCESS_SERVER_KEEP_DIR=1 SMOKE_CLAUDE_GOAL_PROCESS_SERVER_MAX_BUDGET_USD=0.80 ./scripts/smoke_goal_first_claude_process_server_real.sh`

Pendiente real tras este tramo: `runs/control`/shutdown con proveedor externo
vivo o lento, y smoke Gemini cuando exista tier/credencial valido. Antes de
commit quedan `git diff --check` y `go test -count=1 ./...`.

## Continuacion Codex 2026-07-04 noche 21

Avance MEJ-103/T18 y reduccion de `BUG-ORQ-20260704-165`: forced stop real de
`claude_process` atravesando `cmd/orquesta-server`.

- `scripts/smoke_goal_first_claude_process_server_real.sh` gana modo
  `SMOKE_CLAUDE_GOAL_PROCESS_SERVER_CONTROL_MODE=forced_stop`.
- El modo nuevo arranca el mismo servidor temporal con
  `ORQUESTA_CODEX_GOAL_BACKEND=claude_process`, espera un manifiesto interno
  `claude_goal_process_state_*.json` con wrapper vivo, llama a
  `/api/v0/runs/control` con `action=stop` y `forced=true`, observa el goal
  despues del control y valida que no queda proceso Claude vivo.
- Resultado real aceptado:
  `smoke_goal_first_claude_process_forced_stop_server_real=ok`,
  `run_control_estado=ok`, `run_control_status=stopped`,
  `run_control_final_status=stopped`,
  `run_control_goal_status_after=blocked`,
  `run_control_goal_control_signal_confirmed=true`,
  `observe_after_control_poll=1 goal_status=blocked closure_status=blocked
  recommended_action=replan`, `claude_processes_alive_after_control=0`.
- Evidencia retenida:
  `/tmp/orquesta-claude-process-server.GnFFpX`,
  `run_ref=run-spec-smoke-claude-process-server-req-smoke-claude-process-server-dff02568d5f2f00ac86d8f91237536c7`,
  `external_goal_ref=claude-goal-06148fd846fd9f64`.
- Evidencias de control:
  `evidence-ref-claude-goal-process-stop-completed`,
  `evidence-ref-run-control-goal-forced-stop-terminal`,
  `evidence-ref-run-control-terminal-after-goal-forced-stop`,
  `evidence-ref-mcp-run-control-checkpoint-recorded`.
- Shutdown/limpieza: `state/orquesta_server_state_v0.json` queda
  `status=stopped`, `shutdown_status=stopped`, `shutdown_ready=true`, y la
  comprobacion posterior no encuentra `orquesta-server run`,
  `claude_goal_wrapper`, `claude -p`, `codebase-memory-mcp`,
  `codex app-server` ni `orquesta-goal-*`.

Archivos tocados en este avance:

- `scripts/smoke_goal_first_claude_process_server_real.sh`
- `cmd/orquesta-server/smoke_goal_first_script_guard_v0_test.go`
- `docs/runbooks/smoke_goal_first_provider_process_real_2026-07-04.md`
- `docs/plan_mejora_continua_orquesta_2026-07-04.md`
- `docs/inventario_bugs_orquesta_2026-06-30.md`
- `docs/bitacora_correccion_pericial_2026-07-03.md`

Verificacion ejecutada:

- `bash -n scripts/smoke_goal_first_claude_process_server_real.sh`
- `go test -count=1 ./cmd/orquesta-server -run TestSmokeGoalFirstClaudeProcessServerRealEsOptInYLimpiaBackendV0`
- `SMOKE_CLAUDE_GOAL_PROCESS_SERVER_REAL=1 SMOKE_CLAUDE_GOAL_PROCESS_SERVER_CONTROL_MODE=forced_stop SMOKE_CLAUDE_GOAL_PROCESS_SERVER_SKIP_PREFLIGHT=1 SMOKE_CLAUDE_GOAL_PROCESS_SERVER_KEEP_DIR=1 SMOKE_CLAUDE_GOAL_PROCESS_SERVER_MAX_BUDGET_USD=0.80 ./scripts/smoke_goal_first_claude_process_server_real.sh`

Pendiente real tras este tramo: Gemini real con tier/credencial valido y, si
reaparece, smoke largo de observabilidad global `status/observe`; Claude ya
cubre launch/observe/closure accepted y forced stop por HTTP con backend vivo.

## Continuacion Codex 2026-07-04 noche 22

Limpieza de residuales de contrato publico MCP enlazados a
`BUG-ORQ-20260701-065/088` y `BUG-ORQ-20260701-066/088`.

- Se revisan las filas del inventario que seguian contando como mixtas/abiertas
  por descriptores compactos obsoletos.
- `external_work.run`, `run_queue.priority` y `arrancar_director` ya tenian
  descriptores y tests que declaran `external_goal_ref?`/`evidence_refs?`.
- `director.stats` transportaba `external_goal_ref` en `external_job` y `goal`,
  pero su descriptor compacto no lo anunciaba. Se actualiza el output compacto
  y se endurece `TestMCPDirectorStatsToolDescriptorV0ExponeContratoCompacto`.
- El inventario marca esas cuatro filas como `cerrado` para el residual de
  discovery/descriptor. Los bugs padre `065` y `066` siguen abiertos solo por
  sus filas propias de coordinacion automatica/OPES.

Archivos tocados:

- `modulos/orquesta-mcp/director_stats_tool_v0.go`
- `modulos/orquesta-mcp/director_stats_tool_v0_test.go`
- `docs/inventario_bugs_orquesta_2026-06-30.md`
- `docs/bitacora_correccion_pericial_2026-07-03.md`

Verificacion focal:

- `go test -count=1 ./modulos/orquesta-mcp -run TestMCPDirectorStatsToolDescriptorV0ExponeContratoCompacto`

## Continuacion Codex 2026-07-04 noche 23

Limpieza de filas stale del inventario tras auditoria paralela con subagentes
read-only (`Franklin` y `Hubble`), sin `codebase-memory-mcp`, sin indexadores y
sin ediciones delegadas.

- `BUG-ORQ-20260701-065`: se mantienen abierto el padre de coordinacion
  automatica `shutdown/backend/checkpoint/stop/cancel/wait`, pero se cierran las
  cinco subfilas de cleanup externo no forzado (`stop/cancel`, HTTP y transporte
  MCP) porque ya tienen pruebas y defaults `external cleanup` sin degradar a
  `forced`.
- `BUG-ORQ-20260701-076`: se cierra como stale/supersedido por el cierre real de
  forced stop app-server y la cobertura actual de active work/cleanup propio.
  El residual de cleanup externo amplio queda en `BUG-065`/`BUG-165`.
- `BUG-ORQ-20260701-079`: se mantienen abierto el padre por enforcement duro de
  checkpoint temprano/salidas gigantes antes de herramientas, pero se cierran
  las subfilas de `turn/start`, `thread_read` saneado y proyeccion publica en
  status/queue/observe/efficiency/domain-work.
- `BUG-ORQ-20260701-075` y `BUG-ORQ-20260701-066/075`: se mantienen abiertos los
  padres OPES que requieren smoke temporal/e2e y criterios done/settled, pero se
  cierran las subfilas de QA/materialized artifacts, fase 0, required evidence,
  `observe_goal`, `domain-work/status`, `efficiency_summary`, `queue/global`
  y `ops_snapshot`.
- `BUG-ORQ-20260704-165`: se deja una sola fila abierta para el residual global
  de observabilidad/control largo; las filas duplicadas de Sueldos forced-stop
  y timeout del observador residente quedan cerradas por los cierres ya
  documentados de app-server, Claude process y `GoalObserverTimeout`.

Conteo despues del corte documental:

- 208 filas de tabla.
- 173 IDs/keys.
- 7 filas `abierto`.
- 187 filas `cerrado`, 1 `cerrado funcionalmente`,
  5 `cerrado/supersedido`, 8 `historico/supersedido`.

Abiertos reales que quedan:

- `BUG-ORQ-20260704-165`: observabilidad/control global largo si reaparece,
  aunque Codex app-server y Claude process ya cubren forced stop real.
- `BUG-ORQ-20260701-058`: OPES contrato calidad/lifecycle amplio.
- `BUG-ORQ-20260701-065`: coordinacion automatica shutdown/backend/checkpoint/
  stop/cancel/wait.
- `BUG-ORQ-20260701-066`: OPES cierre/observacion done-settled y evitar
  reescritura tardia.
- `BUG-ORQ-20260701-073`: timeout tras checkpoint con reconciliacion real amplia.
- `BUG-ORQ-20260701-075`: QA OPES temporal end-to-end y validadores semanticos
  por work kind.
- `BUG-ORQ-20260701-079`: enforcement duro runtime/proveedor de checkpoint
  temprano y salida gigante.

Archivos usados/tocados:

- `docs/inventario_bugs_orquesta_2026-06-30.md`
- `docs/bitacora_correccion_pericial_2026-07-03.md`

Evidencia revisada por subagentes y Codex local:

- `modulos/orquesta-mcp/run_control_tool_executor_v0.go`
- `modulos/orquesta-mcp/run_control_http_v0_test.go`
- `modulos/orquesta-mcp/run_control_transport_v0_test.go`
- `modulos/orquesta-server-shutdown/*`
- `modulos/orquesta-runtime-codex-appserver/*`
- `modulos/orquesta-app-codex-stack/*`
- `modulos/orquesta-mcp/autoprogramming_status_tool_v0_test.go`
- `modulos/orquesta-mcp/domain_work_http_v0_test.go`
- `modulos/orquesta-mcp/observe_app_director_goal_*`
- `modulos/orquesta-opes-director/*`

Verificacion ejecutada en este corte:

- `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-app-codex-stack ./modulos/orquesta-runtime-codex-appserver ./modulos/orquesta-server-shutdown ./modulos/orquesta-server ./cmd/orquesta-server`
- `git diff --check`
- `go test -count=1 ./...`

## Continuacion Codex 2026-07-04 noche 24

Reduccion adicional de `BUG-ORQ-20260704-165` en la superficie
`orquesta.autoprogramming.observe_active_goals.v0`.

- La pasada de `observe_active_goals` ya no queda bloqueada por un unico
  `observe_goal` lento: cada run se observa con `PerGoalTimeout` propio
  (`2s` por defecto).
- Si un run excede el timeout, se cancela su contexto, se conserva snapshot
  parcial desde `GoalWorkState`, se publica issue/diagnostico
  `autoprogramming_observe_active_goal_timeout`, evidencia
  `evidence-ref-autoprogramming-observe-active-goal-timeout` y acciones
  `observe_goal_single_run`/`poll_autoprogramming_status`.
- El batch continua y puede devolver observaciones de los demas runs sin caer
  a timeout global ni relanzar proveedor.

Archivos tocados:

- `modulos/orquesta-mcp/autoprogramming_observe_active_goals_tool_v0.go`
- `modulos/orquesta-mcp/autoprogramming_observe_active_goals_tool_v0_test.go`
- `docs/inventario_bugs_orquesta_2026-06-30.md`
- `docs/bitacora_correccion_pericial_2026-07-03.md`

Verificacion ejecutada en este corte:

- `go test -count=1 ./modulos/orquesta-mcp -run TestMCPAutoprogrammingObserveActiveGoalsToolExecutorV0TimeoutPorRunConservaBatch`
- `go test -count=1 ./modulos/orquesta-mcp`
- `git diff --check`
- `go test -count=1 ./...`

## Continuacion Codex 2026-07-04 noche 25

Reduccion adicional de `BUG-ORQ-20260704-165` y `BUG-ORQ-20260701-079`:

- `autoprogramming/status` convierte el caso
  `goal_backend_missing_after_external_cleanup` en una `safe_action`
  ejecutable contra `/api/v0/runs/control` con accion `stop`, `run_ref`,
  `requested_by=orquesta-autoprogramming-status` y evidencia
  `evidence-ref-goal-backend-missing-after-external-cleanup`.
- Si `Goal.Status=running` viene de una observacion stale pero la liveness del
  run clasifica `running_stale_no_process` con `SafeToReconcile`,
  `autoprogramming/status` y `runs/control` ya no lo tratan como backend vivo:
  publican/reconcilian `run_control_reconcile_external_cleanup` y evitan
  `control_not_propagated_to_goal_backend` falso.
- En el borde app-server, `turn/start` ya no puede relajar el contrato de
  salida compacta: `max_text_bytes` queda acotado al maximo canonico y los
  hints acotados canonicos se conservan aunque el packet intente sustituirlos.

Archivos tocados:

- `modulos/orquesta-mcp/autoprogramming_operator_v0.go`
- `modulos/orquesta-mcp/autoprogramming_status_tool_v0.go`
- `modulos/orquesta-mcp/autoprogramming_status_stale_running_v0.go`
- `modulos/orquesta-mcp/run_control_tool_executor_v0.go`
- `modulos/orquesta-mcp/autoprogramming_status_tool_v0_test.go`
- `modulos/orquesta-mcp/run_control_tool_v0_test.go`
- `modulos/orquesta-runtime-codex-appserver/codex_goal_app_server_v0.go`
- `modulos/orquesta-runtime-codex-appserver/codex_goal_app_server_migrated_v0_test.go`
- `docs/inventario_bugs_orquesta_2026-06-30.md`
- `docs/bitacora_correccion_pericial_2026-07-03.md`

Verificacion ejecutada en este corte:

- `go test -count=1 ./modulos/orquesta-mcp -run 'TestMCP(AutoprogrammingStatusExecutorV0GoalRunning(StaleSinProcesoPideReconciliarCleanupExterno|SinBackendActivoPideReconciliarCleanupExterno)|RunControlExecutorV0Stop(ReconcilesExternalCleanupConGoalStatsRunningStaleSinProceso|ReconcilesExternalCleanupConEvidenciaSinForce|ForcedNoPublicaStoppedSiGoalBackendSigueActive))V0'`
- `go test -count=1 ./modulos/orquesta-mcp`
- `go test -count=1 ./modulos/orquesta-runtime-codex-appserver -run 'TestServerCodexAppServerGoalBackendV0TurnStart(InyectaContratoSalidaCompacta|NoRelajaContratoSalidaCompacta)V0'`
- `go test -count=1 ./modulos/orquesta-runtime-codex-appserver`

## Continuacion Codex 2026-07-04 noche 27

Reduccion adicional de `BUG-ORQ-20260701-079`:

- `orquesta-runtime-codex-appserver` materializa un checkpoint runtime dentro
  del `write_set` autorizado tras `thread/start` y antes de `turn/start`. Asi,
  cuando el agente empieza el turno y puede ejecutar herramientas internas del
  app-server, ya existe una evidencia durable recuperable
  `checkpoint_started.txt`.
- La evidencia de launch incluye
  `evidence-ref-codex-app-server-early-checkpoint-materialized:<relpath>` para
  distinguir checkpoint materializado por runtime de avance semantico del
  agente.
- Auditoria del schema local generado por `codex app-server generate-json-schema`
  confirma que `turn/start` no expone hoy un campo compatible de limite duro de
  stdout/tool-output; Orquesta conserva los limites de `thread/read` y
  saneamiento posterior, pero el cap pre-tool real queda como frontera del
  runtime/proveedor.

Archivos tocados:

- `modulos/orquesta-runtime-codex-appserver/codex_goal_app_server_v0.go`
- `modulos/orquesta-runtime-codex-appserver/codex_goal_app_server_migrated_v0_test.go`
- `docs/inventario_bugs_orquesta_2026-06-30.md`
- `docs/bitacora_correccion_pericial_2026-07-03.md`

Verificacion ejecutada en este corte:

- `go test -count=1 ./modulos/orquesta-runtime-codex-appserver -run 'TestServerCodexAppServerGoalBackendV0(TurnStart(InyectaContratoSalidaCompacta|NoRelajaContratoSalidaCompacta)|MaterializaCheckpointAntesDeTurnStart)V0'`
- `go test -count=1 ./modulos/orquesta-runtime-codex-appserver`
- `git diff --check`
- `go test -count=1 ./...`

## Corte Codex para auditoria 2026-07-04 tarde

El operador paro la implementacion para auditoria. El detalle operativo queda en
`docs/runbooks/handoff_claude_mejora_continua_orquesta_2026-07-03.md`, seccion
`Corte Codex para auditoria 2026-07-04 tarde`.

Resumen:

- No se aplico parche nuevo despues de `f4c9984e`.
- Git estaba limpio y sincronizado con origin.
- El inventario tenia 209 filas, 167 IDs unicos y 6 bugs abiertos reales.
- Se estaba investigando `BUG-ORQ-20260704-165`/`BUG-ORQ-20260701-065`.
- Hallazgo: gran parte de la limpieza stale/stopped ya existe; la siguiente
  accion recomendada es test focal sobre narrativa contradictoria
  `shutdown_status`/`shutdown_ready` en snapshot `stopped`, no refactor amplio.

## Continuacion Codex 2026-07-04 tarde 2

Reduccion adicional de `BUG-ORQ-20260704-165` y
`BUG-ORQ-20260701-065`:

- El test focal recomendado en el corte de auditoria fallo: un snapshot
  `Status="stopped"` podia conservar `shutdown_status=stop_timeout` y
  `shutdown_ready=false` si no tenia active work vivo.
- `NormalizeStoppedServerSnapshotV0` ya considera ese caso sucio y lo
  reconcilia a `shutdown_status=stopped` y `shutdown_ready=true`.
- `orquesta-server status` cubre y persiste esa normalizacion desde statefile.

Archivos tocados:

- `modulos/orquesta-server/status_process_stale_v0.go`
- `modulos/orquesta-server/status_process_stale_v0_test.go`
- `cmd/orquesta-server/command_public_output_v0_test.go`
- `docs/inventario_bugs_orquesta_2026-06-30.md`
- `docs/runbooks/handoff_claude_mejora_continua_orquesta_2026-07-03.md`
- `docs/bitacora_correccion_pericial_2026-07-03.md`

Verificacion focal ejecutada:

- `go test -count=1 ./modulos/orquesta-server -run 'TestNormalizeStoppedServerSnapshotV0|TestStatusTracker(Stopped|RuntimeStopped)'`
- `go test -count=1 ./cmd/orquesta-server -run 'TestStatusServerCommandV0ReconciliaStopped'`

## Claude: test focal narrativa shutdown residual (2026-07-04 tarde)

Siguiendo el "Mensaje para Claude" del corte de auditoria: el test focal
demostro incoherencia real. `NormalizeStoppedServerSnapshotV0` reparaba
status/ready/active_work pero conservaba `shutdown_agents_in_flight`,
`shutdown_checkpoints_pending` y `shutdown_checkpoint_agents_pending`
residuales: un snapshot `stopped` heredado de timeout quedaba `shutdown_ready`
con agentes en vuelo declarados. Fix minimo: esos contadores (y
`shutdown_runs_*`) entran en el clean-check y en el clear. Test
`TestNormalizeStoppedServerSnapshotV0LimpiaContadoresShutdownResidualesV0`.
Suites: `./modulos/orquesta-server` y `./cmd/orquesta-server` verdes.
Reduce el residual de narrativa stale de BUG-ORQ-20260704-165/BUG-065.

Revision subagente `Hume`: coherente, riesgo bajo; no encontro contrato interno
que dependa de conservar `stopped + shutdown_status=stop_timeout`. Sugirio
blindar explicitamente `shutdown_runs_*`; el test de contadores queda ampliado
con `ShutdownRunsRequested > ShutdownRunsStopped`.

## Cola descongelada por el operador (2026-07-04 tarde)

Orden textual: "dale todas las tareas a codex para que lo programe". Las 6
tareas autorizadas quedan en docs/instrucciones_director_codex_2026-07-04.md
seccion 5 (MEJ-104 smoke+activacion, broker MCP, enrutado por coste,
shutdown amplio, OPES 058/066/075, ratchet de envs), con orden sugerido y
protocolo. Commit 465bf3e0. El director Codex vivo debe tomarlas de ahi.

## Continuacion Codex 2026-07-04 tarde 3

Trabajo ejecutado con subagentes read-only/worker y sin `codebase-memory-mcp`.
`scripts/bootstrap_agent_tooling.sh --status` devolvio `broker_only`,
`live_codebase_memory_mcp_processes=0`.

Reducciones cerradas en este bloque:

- `BUG-ORQ-20260701-079`: el app-server command/legacy RPC queda mas acotado.
  El lector JSON-RPC legacy baja a 256 KiB, `stderr` de comandos queda retenido
  como cola acotada y el clasificador de logs lee solo tail acotado.
- `BUG-ORQ-20260701-079` / residual `BUG-ORQ-20260704-165`:
  `autoprogramming/status` ya no deja un goal `running` con backend activo,
  alto consumo y cero checkpoint/artefactos/receipts como simple
  `observe_goal`; publica `goal_active_no_checkpoint_high_consumption` y
  `replan_narrow_context`. Se ajusto la guarda para no pisar el caso distinto
  de backend ausente/stale sin proceso, que sigue por reconciliacion de cleanup
  externo.
- `BUG-ORQ-20260701-065` / `BUG-ORQ-20260704-165`: test combinado de shutdown
  con dos goals activos cubre la secuencia `waiting_checkpoint`,
  `waiting_drain`, `backend_still_running`, `ready`, todos los POST con
  `cleanup_goal_backends=true` y `cleanup_completed` no bloqueante.
- `BUG-ORQ-20260701-058/066`: el conector REST OPES acepta `completed`,
  `done` y `settled` como terminales nativos en receipts con `CompleteJob=true`,
  manteniendo `pending` como invalido.

TAREA-6/MEJ-106 queda documentada, no endurecida: la medicion real actual es
`env_vars_orquesta=513` y ya existe `env_vars_budget_test.go` con ratchet 513.
La orden de bajarlo a 511 requiere retirar o consolidar dos nombres
`ORQUESTA_*` reales antes de cambiar el test; hacerlo ahora meteria un rojo
falso.

Archivos tocados:

- `cmd/orquesta-server/shutdown_client_v0_test.go`
- `modulos/orquesta-mcp/autoprogramming_status_stale_running_v0.go`
- `modulos/orquesta-mcp/autoprogramming_status_tool_v0.go`
- `modulos/orquesta-mcp/autoprogramming_status_tool_v0_test.go`
- `modulos/orquesta-opes-connector/result_v0.go`
- `modulos/orquesta-opes-connector/rest_client_receipt_validation_v0_test.go`
- `modulos/orquesta-runtime-codex-appserver/codex_goal_app_server_command_protocol_v0.go`
- `modulos/orquesta-runtime-codex-appserver/codex_goal_app_server_migrated_v0_test.go`
- `modulos/orquesta-runtime-codex-appserver/codex_goal_app_server_rpc_v0.go`
- `docs/inventario_bugs_orquesta_2026-06-30.md`
- `docs/runbooks/handoff_claude_mejora_continua_orquesta_2026-07-03.md`
- `docs/bitacora_correccion_pericial_2026-07-03.md`

Verificacion ejecutada:

- `go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPAutoprogrammingStatusExecutorV0(GoalRunningHighConsumptionNoDegradaAObserveGoal|GoalRunningStaleSinProcesoPideReconciliarCleanupExterno|SinCheckpointHighConsumptionEsBloqueante|CheckpointOnlyHighConsumptionEsBloqueante)'`
- `go test -count=1 ./modulos/orquesta-opes-connector -run 'TestRESTClientV0SubmitDomainWorkArtifact(AceptaEstadosTerminalesNativosOPES|RechazaReceiptNoCausal)'`
- `go test -count=1 ./modulos/orquesta-runtime-codex-appserver -run 'TestCodexAppServer(CommandProtocol(ThreadReadResponseBudget|TurnStartResponseBudget|StderrDiagnosticoAcotado)|LegacyRPCReaderResponseBudget|DiagnosticTailFile|WebSocket(ThreadReadResponseBudget|DefaultFrameBudgetConserva16MiB))V0'`
- `go test -count=1 ./cmd/orquesta-server -run TestRequestServerShutdownV0CoordinaDosGoalsActivosHastaGoalActionsResueltasV0`
- `git diff --check`
- `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-opes-connector ./modulos/orquesta-runtime-codex-appserver ./cmd/orquesta-server`
- `go test -count=1 ./...`

No sobrecerrar:

- `BUG-079` sigue abierto para enforcement duro pre-tool dentro del
  runtime/proveedor y smoke largo real.
- `BUG-165/065` siguen abiertos para smoke real amplio de proveedor/status
  lento y coordinacion automatica completa backend/checkpoint/stop/cancel/wait.
- `BUG-058/066` siguen abiertos para lifecycle OPES end-to-end con instancia
  temporal y external-work real.

## Continuacion Codex 2026-07-04 tarde 4

TAREA-7/7b wizard de programacion conversacional: integrado el nucleo web y
corregidos los huecos detectados por revision paralela de subagente.

Hecho:

- `orquesta-web` tiene tipos puros de wizard, motor de huecos R1-R8,
  defaults de ingenieria y aceptacion automatica de recomendaciones.
- El caso canonico `"quiero una app para una agenda"` pregunta por movil/PC,
  personal/compartido e integracion de agenda Google/Microsoft/CalDAV, y llega
  a spec valido en <=6 turnos con recomendaciones.
- El endpoint guided devuelve `wizard` y acepta `wizard_answers`, manteniendo
  `Contrasts` si el usuario elige una opcion distinta de la recomendada.
- Se corrigio un riesgo estructural: la pregunta de gobierno de integracion ya
  no genera claves i18n dinamicas por indice; el `question_ref` es estable y el
  indice queda solo en `field`.
- Catalogos es/en y `NuevaAppI18nRequiredKeysV0` cubren las claves nuevas;
  test exacto por locale recorre agenda, pagos, mapas, uso compartido, mobile,
  storage, integracion y deploy incompatible.
- `docs/diseno_wizard_programacion_2026-07-04.md` deja de declarar
  falsamente "NO implementado" y marca estado parcial verificado.

Archivos tocados:

- `modulos/orquesta-web/nueva_app_wizard_types_v0.go`
- `modulos/orquesta-web/nueva_app_wizard_defaults_v0.go`
- `modulos/orquesta-web/nueva_app_wizard_gaps_v0.go`
- `modulos/orquesta-web/nueva_app_wizard_turn_v0.go`
- `modulos/orquesta-web/nueva_app_wizard_turn_v0_test.go`
- `modulos/orquesta-web/nueva_app_intake_guided_endpoint_v0.go`
- `modulos/orquesta-web/nueva_app_i18n_es_v0.go`
- `modulos/orquesta-web/nueva_app_i18n_en_v0.go`
- `modulos/orquesta-web/nueva_app_i18n_keys_v0.go`
- `docs/diseno_wizard_programacion_2026-07-04.md`
- `docs/runbooks/handoff_claude_mejora_continua_orquesta_2026-07-03.md`
- `docs/bitacora_correccion_pericial_2026-07-03.md`

Verificacion ejecutada:

- `go test -count=1 ./modulos/orquesta-web -run 'TestWizard|TestNuevaAppIntakeGuidedResponseV0IncluyeWizardRicoV0|TestNuevaAppIntakeGuidedResponseV0AplicaWizardAnswersV0|TestNuevaAppI18nCatalogV0CatalogosCubrenClavesRequeridas'`
- `go test -count=1 ./modulos/orquesta-web`
- `go test -count=1 ./modulos/orquesta-app-gateway ./modulos/orquesta-http-gateway`
- `git diff --check`
- `go test -count=1 ./...`

No sobrecerrar:

- Falta tool MCP equivalente `orquesta.nueva_app.wizard.v0`.
- Falta render web completo de preguntas/opciones/recomendacion/contraste.
- El pilot t297 queda como contexto historico; la version canonica es la del
  repo principal.

## Continuacion Codex 2026-07-04 tarde 5

TAREA-7/7b G2 cerrada en el repo principal, tras revisar la ampliacion de
Claude en `docs/diseno_wizard_programacion_2026-07-04.md`.

Hecho:

- MCP expone `orquesta.nueva_app.wizard.v0` con contrato de turno compatible
  con `need`, `action_id`, `answer`, `wizard_answers`, `session`, `locale`,
  `nombre` e `idea`; devuelve `turn`, `session` y `wizard` como JSON.
- `orquesta-app-codex-stack` cablea el executor real del wizard MCP usando el
  handler guiado existente, sin duplicar reglas de negocio.
- `/nueva-app` renderiza el turno rico: preguntas, opciones, recomendacion
  visible, racionales y defaults de ingenieria.
- El cliente web envia clicks como `wizard_answers` y repinta preguntas,
  contrastes y defaults desde la respuesta del endpoint.
- Se serializa un mapa i18n es/en del wizard al cliente para no mostrar claves
  internas en turnos dinamicos.

Archivos tocados:

- `modulos/orquesta-mcp/nueva_app_wizard_tool_v0.go`
- `modulos/orquesta-mcp/nueva_app_wizard_transport_v0.go`
- `modulos/orquesta-mcp/nueva_app_wizard_tool_v0_test.go`
- `modulos/orquesta-mcp/mcp_transport_bindings_v0.go`
- `modulos/orquesta-mcp/mcp_transport_registry_v0_test.go`
- `modulos/orquesta-mcp/mcp_transport_tool_descriptors_v0.go`
- `modulos/orquesta-mcp/mcp_transport_tool_input_schema_v0.go`
- `modulos/orquesta-mcp/mcp_transport_tools_v0.go`
- `modulos/orquesta-app-codex-stack/nueva_app_wizard_mcp_executor_v0.go`
- `modulos/orquesta-app-codex-stack/stack_v0.go`
- `modulos/orquesta-app-codex-stack/stack_flow_build_v0_test.go`
- `modulos/orquesta-app-codex-stack/stack_flow_v0_test.go`
- `modulos/orquesta-web/nueva_app_html_render_v0.go`
- `modulos/orquesta-web/nueva_app_html_handler_v0_test.go`
- `modulos/orquesta-web/nueva_app_i18n_es_v0.go`
- `modulos/orquesta-web/nueva_app_i18n_en_v0.go`
- `modulos/orquesta-web/nueva_app_i18n_keys_v0.go`
- `docs/diseno_wizard_programacion_2026-07-04.md`

Verificacion ejecutada:

- `go test -count=1 ./modulos/orquesta-web -run 'TestNuevaAppHTMLHandlerV0GETMuestraFormularioUsableSinDelegar|TestWizard|TestNuevaAppIntakeGuidedResponseV0IncluyeWizardRicoV0|TestNuevaAppIntakeGuidedResponseV0AplicaWizardAnswersV0|TestNuevaAppI18nCatalogV0CatalogosCubrenClavesRequeridas'`
- `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-app-codex-stack ./modulos/orquesta-web`

No sobrecerrar:

- La taxonomia nueva de Claude U1-U12 y packs de dominio queda pendiente; el
  motor actual cubre R1-R8 y packs iniciales, no el universo completo.
- Siguen abiertos `BUG-079`, `BUG-165/065`, `BUG-058/066`, `BUG-075` y
  `MEJ-106` segun el handoff vigente.

## Continuacion Codex 2026-07-04 tarde 6

TAREA-6/MEJ-106 avanza el ratchet de entorno de 513 a 511 sin tocar nucleo
neutral. Se consolidan dos `ORQUESTA_*` reales de la composition root:
`ORQUESTA_CAPACITY_MODEL_REF` y `ORQUESTA_CAPACITY_QUOTA_REF` salen de la
superficie configurable porque no tenian uso operativo documentado fuera del
registro/lectura de la composicion. El stack conserva `ModelRef` y `QuotaRef`
internos por defecto para la decision de capacidad, y mantiene como knobs
externos `ORQUESTA_CAPACITY_POLICY_REF` y `ORQUESTA_CAPACITY_POOL_REF`.

Cambios aplicados:

- `cmd/orquesta-server/server_env_registry_v0.go` retira los dos nombres y su
  metadata publica.
- `cmd/orquesta-server/stack.go` deriva `ModelRef`/`QuotaRef` desde defaults
  internos estables.
- `cmd/orquesta-server/effective_config_v0.go` deja de publicar settings de
  entorno que ya no existen.
- `cmd/orquesta-server/env_vars_ratchet_v0_test.go` fija ratchet focal
  `env_vars_orquesta <= 511` y falla ante subidas salvo justificacion explicita
  con `env_vars_orquesta_allow_increase_to=<valor>` en inventario o bitacora.
- `scripts/orquesta_metricas_deuda.sh` actualiza la lectura comentada a 511.

Evidencia local antes de cierre:

- `scripts/orquesta_metricas_deuda.sh --json` devuelve
  `{"env_vars_orquesta":511,"endpoints_status":16,"interfaces_estado":65,"modulos_director":17}`.

Pendiente antes de cerrar la tarea: ejecutar el test requerido
`go test -count=1 ./cmd/orquesta-server`.

## Continuacion Codex 2026-07-04 tarde 7

Corte de auditoria tras intentar paralelizar T1-T7 por Orquesta. No hay cierre
funcional nuevo de esas tareas: el resultado util de esta tanda es diagnostico
de autonomia y limpieza, mas un relanzamiento previsto con contexto estrecho.

Observado:

- El lote conjunto en `/tmp/orquesta-autonomia-start-20260704T142421Z` fallo
  durante `prepare-run` por carreras/ownership del backend `app_server_tmux`
  (`codex_app_server_tmux_session_exited`, `code_home_unavailable`,
  `start_failed`, `session_unowned`, `kill_failed`).
- Los relanzamientos aislados T1-T6 fueron aceptados, pero materializaron solo
  checkpoints o quedaron bloqueados por alto consumo sin receipt terminal util.
- El T7 viejo del wizard quedo obsoleto: su payload solo cubria la seccion 10
  antigua y no las secciones 10 tecnica, 11/11.5 y 12 que Claude anadio
  despues.
- `POST /api/v0/runs/control` con `forced=true` devolvio `status=stopped` y
  `goal_control_signal_confirmed=true`, pero `/api/v0/server/shutdown` siguio
  devolviendo `backend_still_running` para backends `app_server_tmux`.
  Se hizo limpieza local limitada a procesos temporales bajo
  `/tmp/orquesta-autonomia-*` y `/tmp/oq-gsrv-*`; comprobacion posterior sin
  `orquesta-server run`, sin `codex app-server` y sin `codebase-memory-mcp`.

Subagentes read-only usados, sin `codebase-memory-mcp`:

- Euler audito el wizard frente a las secciones nuevas: faltan T1-T8,
  `WizardFactV0`, `RequiresFacts`, `ExcludedByFacts`,
  `ResolvedByExclusion`, `HelpKey`, `ExampleKey`, justificacion de contraste,
  `GlossaryExpanded`, `ComprehensionQuery`, glosario generado y bot RAG/MCP
  `orquesta.nueva_app.wizard.bot.v0`.
- Darwin audito los pilotajes: confirma que no son verdes y recomienda tratar
  el patron como residual abierto de lifecycle goal-first/Codex tmux y shutdown
  no cooperativo.

Decision de direccion:

- No integrar checkpoints/result placeholders como cierre funcional.
- Documentar la reproduccion en `BUG-ORQ-20260704-165`.
- Relanzar por Orquesta solo goals pequenos con contexto estrecho. Primero T7A:
  contratos/exclusion/ayuda del wizard en `orquesta-web`, sin tocar MCP bot ni
  stack. Despues, si T7A cierra, T7B para corpus/bot determinista y T7C para
  MCP/web chat/LLM opt-in.

## Continuacion Codex 2026-07-04 tarde 8

Revision de `docs/auditoria_envs_pisadas_2026-07-04.md` y TAREA-8 de
`docs/instrucciones_director_codex_2026-07-04.md`.

Hecho en codigo:

- `ORQUESTA_CODEX_CODE_HOME` queda como canónica para auth/config Codex;
  `CODEX_HOME` queda como alias legacy diagnosticado. Si solo existe legacy, el
  setting canónico sale con `source=legacy_alias` y `deprecated_env_used`; si
  ambas existen y difieren, sale `env_alias_conflict` y gana la canónica.
- `ORQUESTA_OPES_BASE_URL` queda como canónica para OPES; `OPES_BASE_URL` queda
  como alias legacy con el mismo patrón.
- `ORQUESTA_CODEX_HOME` se documenta en el registry como HOME del proceso Codex,
  no como fuente de auth/config.
- Se conserva `env_vars_orquesta=511`; no se sube el ratchet.

Archivos principales:

- `cmd/orquesta-server/server_env_registry_v0.go`
- `cmd/orquesta-server/codex_env_v0.go`
- `cmd/orquesta-server/codex_wave_config_v0.go`
- `cmd/orquesta-server/effective_config_v0.go`
- `cmd/orquesta-server/config_test.go`
- `docs/auditoria_envs_pisadas_2026-07-04.md`
- `docs/inventario_bugs_orquesta_2026-06-30.md`

Verificacion:

- `go test -count=1 ./cmd/orquesta-server -run 'TestServerConfigFromEnvV0Diagnostica(OPESBaseURLLegacyAlias|OPESBaseURLPisada|CodexCodeHomeLegacyAlias|CodexCodeHomePisado)V0|TestServerDaemonStartEnvironmentV0ProyectaPoliticaAutoprogramacion|TestServerCodexGoalBackendFromEnvV0TmuxNoArrancaAppServerEnConstruccionV0'`
- `go test -count=1 ./cmd/orquesta-server`
- `scripts/orquesta_metricas_deuda.sh --json`

Pendiente de la auditoria de variables:

- `ORQUESTA_SERVER_URL`/`ORQUESTA_BASE_URL`: riesgo bajo, falta diagnostico de
  conflicto.
- Timeouts de smoke con unidades distintas: mover a prefijo claro o marcar como
  solo-proceso-hijo.
- `ORQUESTA_GUARDIAN_*`: declarar como `child_process_env` o excluir con
  metadata.
- TAREA-8.1/8.2 grande: fichero canónico `orquesta.config.*` y guard
  `config_projection_mismatch`.

## Continuacion Codex 2026-07-04 tarde 9

Se completa otro corte de la ola 1 de TAREA-8 con paralelizacion por
subagentes y prueba real acotada de Orquesta.

Hecho:

- `ORQUESTA_SERVER_URL` queda canónica frente a `ORQUESTA_BASE_URL`, con
  setting sensible en `effective_config`, alias legacy, conflicto diagnosticado
  y precedencia corregida en scripts OPES.
- Timeouts de smoke normalizados a sufijo `_MS`: Codex usa
  `ORQUESTA_CODEX_SMOKE_TIMEOUT_MS`; el alias legacy
  `ORQUESTA_CODEX_SMOKE_TIMEOUT_SECONDS` queda retirado el 2026-07-09.
  Claude/Gemini directos usan
  `SMOKE_*_TIMEOUT_MS`; el script Claude server acepta
  `SMOKE_CLAUDE_GOAL_PROCESS_SERVER_REQUEST_TIMEOUT_MS`.
- `ORQUESTA_GUARDIAN_*` emitidas por el servidor quedan en registry de
  `child_process` y los tests impiden claves no registradas o heredadas desde
  el entorno padre.
- Se uso Orquesta real temporal: `orquesta-server run` con state/runtime bajo
  `/tmp` publico `env_alias_conflict` para `ORQUESTA_BASE_URL`,
  `deprecated_env_used` para `CODEX_HOME` y `deprecated_env_used` para
  `OPES_BASE_URL`, sin valores crudos. El primer intento quedo bloqueado por la
  guarda correcta `opes_destination_confirmation_required`; se reintento contra
  loopback con `ORQUESTA_OPES_TEMPORAL_CONFIRM=1`.

Ratchet:

- `scripts/orquesta_metricas_deuda.sh --json` devolvia
  `env_vars_orquesta=512` en esta ola historica. Tras cambios posteriores el
  valor real subio a 514; la ola 2026-07-09 retira el alias legacy Codex
  `_SECONDS` y deja el techo vigente en
  `env_vars_orquesta_allow_increase_to=513` por las dos envs Telegram
  operativas pendientes.

Verificado:

- `go test -count=1 ./cmd/orquesta-server`
- `go test -count=1 ./modulos/orquesta-runtime-claude ./modulos/orquesta-runtime-gemini ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-app-codex-stack`
- `bash -n` de los scripts de smoke tocados.
- `git diff --check`

Pendiente real:

- TAREA-8.1: fichero canónico `orquesta.config.*`.
- TAREA-8.3 completa: ratchet bidireccional de envs registradas/leídas.
- Retirada final de aliases legacy tras compatibilidad.

## Continuacion Codex 2026-07-04 tarde 10

Se cierra TAREA-8.2 en primer corte operativo para las dos rutas que lanzan
trabajo:

- `POST /api/v0/autoprogramming/prepare-run` y `POST /api/v0/apps/director`
  aceptan `required_settings:[{key,value}]`.
- `cmd/orquesta-server` convierte `effective_config.settings` a una proyección
  MCP pequeña y la inyecta en `orquesta-app-codex-stack`.
- El stack valida antes de lanzar goal, persistir run legacy o encolar. En
  mismatch devuelve `config_projection_mismatch` con HTTP 400.
- El guard queda en adaptador/composición; no se mete `effective_config` en
  `orquesta-goal` ni en core.

Verificado:

- `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-i18n-docs`
- `go test -count=1 ./modulos/orquesta-app-codex-stack`
- `go test -count=1 ./cmd/orquesta-server`
- `go test -count=1 ./...`
- Smoke Orquesta temporal: ambos endpoints devuelven
  `config_projection_mismatch` cuando `required_settings` exige un umbral que
  no coincide con la proyección efectiva.

Pendiente real ahora:

- TAREA-8.1: fichero canónico `orquesta.config.*`.
- TAREA-8.3 completa: ratchet bidireccional y reducción del techo
  `env_vars_orquesta=512` cuando se retiren aliases legacy.
- Retirada final de aliases legacy tras ventana de compatibilidad.

## Continuacion Codex 2026-07-04 tarde 11

Se cierra un primer corte de TAREA-8.1 y se deja TAREA-8.3 como diagnóstico
AST no bloqueante.

Hecho:

- Nuevo `orquesta.config.json` por proyecto con
  `schema_version:"orquesta_config.v0"`.
- Primer bloque soportado: `autoprogramming.checkpoint_only_high_consumption_tokens`,
  `checkpoint_only_max_wait_seconds` y
  `no_checkpoint_warning_max_wait_seconds`.
- Precedencia: env explícita gana al fichero; fichero gana a default.
- `effective_config` publica `source=config_file` para esos valores.
- El stack y `/api/v0/autoprogramming/status` consumen la política efectiva
  desde `ConfigV0`, no recalculada solo desde env.

Bug encontrado y cerrado durante smoke:

- El status REST del gateway no recibía `AutoprogrammingGoalProgressPolicy`;
  con fichero `450000/1500/900` publicaba defaults `100000/900/600`.
- Cierre: `orquesta-app-gateway.ConfigV0` y el handler REST reciben
  `AutoprogrammingGoalProgressPolicy` desde `orquesta-app-codex-stack`.

Ratchet:

- `cmd/orquesta-server/server_env_registry_ast_v0_test.go` detecta por AST
  lecturas `ORQUESTA_*` en producción. Queda no-failing: 226 lecturas actuales
  faltan en registry/allowlist. Es deuda estructural para TAREA-8.3, no verde
  falso.

Verificado:

- `go test -count=1 ./modulos/orquesta-app-gateway ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server -run 'AutoprogrammingStatusAPIRouteV0PublicaGoalProgressPolicy|FicheroCanonico|EnvExplicito|SchemaInvalido|BuildStackFromEnvV0'`
- `go test -count=1 ./modulos/orquesta-server`
- `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-app-codex-stack ./modulos/orquesta-i18n-docs`
- `go test -count=1 ./cmd/orquesta-server`
- `git diff --check`
- Smoke Orquesta temporal con `orquesta.config.json`:
  `status_http=200`, `match_http=400` por validación de spec incompleta pero
  sin `config_projection_mismatch`, y `mismatch_http=400` con
  `config_projection_mismatch`.

Pendiente:

- Ampliar fichero canónico a todas las familias grandes de configuración.
- Diseñar `--config`/snapshot de daemon si se adopta como contrato operativo.
- Hacer estricto el ratchet AST por fases y bajar las 226 lecturas pendientes.

## Continuacion Codex 2026-07-04 tarde 12

Se amplía TAREA-8.1 y se endurece TAREA-8.3 sin romper compatibilidad.

Hecho:

- `orquesta.config.json` ya no cubre solo `autoprogramming`: añade `server`,
  `daemon_logs` y `codex_runtime`.
- Campos activos: `server.addr`, `server.state_dir`, `server.audit_file`,
  `server.audit_disabled`, `daemon_logs.max_bytes`,
  `daemon_logs.max_rotated_files`, `daemon_logs.retention_days`,
  `daemon_logs.local_raw_enabled`, `daemon_logs.local_raw_reason` y
  `codex_runtime.runtime_work_dir`.
- `effective_config` publica `source=config_file` campo a campo y redacta paths
  sensibles como refs.
- `cmd/orquesta-server/server_env_registry_ast_v0_test.go` pasa a ratchet de
  no-incremento. La baseline vigente baja a 220 lecturas pendientes.

Verificado:

- `go test -count=1 ./cmd/orquesta-server -run 'FicheroCanonico|EnvExplicitoGana|AuditFileInvalido|EnvRegistryAST|EnvVars|Ratchet'`
- `go test -count=1 ./cmd/orquesta-server`
- `go test -count=1 ./modulos/orquesta-app-gateway ./modulos/orquesta-app-codex-stack ./modulos/orquesta-mcp ./modulos/orquesta-i18n-docs ./modulos/orquesta-server`
- Smoke Orquesta temporal ampliado:
  `orquesta_temp_config_file_expanded_smoke=passed`, status publicó
  `450000/1500`, `required_settings=450000` no produjo mismatch y
  `required_settings=999999` produjo `400/config_projection_mismatch`.

Pendiente:

- Familias restantes de fichero canónico: `goal_backend`, límites Codex/agentes,
  OPES/bridges, codebase broker, rails/seguridad y domain work.
- `--config`/snapshot daemon.
- Ratchet AST estricto por categorías hasta bajar de 220 a cero.

## Continuacion Codex 2026-07-04 tarde 13

Se amplía TAREA-8.1 con otra familia pequeña y segura del servidor.

Hecho:

- `orquesta.config.json` soporta ahora `server_http`:
  `read_header_timeout_ms`, `read_timeout_ms`, `write_timeout_ms`,
  `idle_timeout_ms`, `max_header_bytes` y `control_body_max_bytes`.
- Añadido `server_lifecycle.shutdown_grace_ms`.
- La precedencia sigue siendo por campo: env explícita > fichero > default.
- `effective_config` marca `source=config_file` para estos nuevos settings.
- Se revisó el aviso sobre `ORQUESTA_CAPACITY_MODEL_REF` y
  `ORQUESTA_CAPACITY_QUOTA_REF`: no se repone. Su retirada era intencional para
  MEJ-106; reponerlas subía el ratchet a 514 y sería regresión.

Verificación:

- `go test -count=1 ./cmd/orquesta-server -run 'LeeHTTPYLifecycle|LeeServerDaemonRuntime|EnvExplicitoGanaCampo|ShutdownGrace|EnvRegistryAST|EnvVarsOrquestaRatchet'`
- Smoke Orquesta temporal aislado fuera del repo:
  `orquesta_config_http_smoke=passed`; `/api/v0/server/status` publicó
  `ORQUESTA_SERVER_READ_HEADER_TIMEOUT_MS=1111` y
  `ORQUESTA_SERVER_SHUTDOWN_GRACE_MS=7777` con `source=config_file`;
  `POST /api/v0/autoprogramming/prepare-run` devolvió
  `config_projection_mismatch` al exigir un valor divergente.

Pendiente:

- Familias restantes: `goal_backend`, límites Codex/agentes, OPES/bridges,
  codebase broker, rails/seguridad, domain work, usage accounting y snapshot de
  daemon/`--config`.
- Ratchet AST vigente: 220 lecturas pendientes. Métrica global vigente:
  `env_vars_orquesta=512`.

## Continuacion Codex 2026-07-04 tarde 14

Se completa otro bloque pequeño de TAREA-8.1 y baja el ratchet AST.

Hecho:

- `orquesta.config.json` soporta ahora `worktree_snapshot.max_files`,
  `worktree_snapshot.max_file_bytes` y
  `worktree_snapshot.max_total_bytes`.
- El presupuesto efectivo alimenta tanto `effective_config` como el
  `SnapshotReadBudget` del stack Codex.
- Las tres envs `ORQUESTA_WORKTREE_SNAPSHOT_*` pasan al registro central.
- Baseline AST baja de 220 a 217 lecturas pendientes.

Verificación:

- `go test -count=1 ./cmd/orquesta-server -run 'WorktreeSnapshot|EnvRegistryAST|EnvVarsOrquestaRatchet|LeeHTTPYLifecycle' -v`
- `scripts/orquesta_metricas_deuda.sh --json` -> `env_vars_orquesta=512`
- Smoke Orquesta temporal aislado:
  `orquesta_config_snapshot_smoke=passed`; status publicó
  `ORQUESTA_WORKTREE_SNAPSHOT_MAX_FILES=17` y
  `ORQUESTA_WORKTREE_SNAPSHOT_MAX_FILE_BYTES=18000` con
  `source=config_file`; `required_settings` divergente devolvió
  `config_projection_mismatch`.

Pendiente:

- Familias restantes: `goal_backend`, límites Codex/agentes, OPES/bridges,
  codebase broker, rails/seguridad, domain work, usage accounting y snapshot de
  daemon/`--config`.
- Ratchet AST vigente: 217.

## Continuacion Codex 2026-07-04 tarde 15

Se completa `codex_usage_accounting` dentro de TAREA-8.1.

Hecho:

- `orquesta.config.json` soporta `codex_usage_accounting.mode` y
  `codex_usage_accounting.log_max_bytes`.
- El opt-in sigue siendo explícito: solo activa métricas con
  `redacted_report` o `runtime_usage_report`.
- `effective_config` publica `ORQUESTA_CODEX_USAGE_ACCOUNTING` y
  `ORQUESTA_CODEX_USAGE_LOG_MAX_BYTES` con `source=config_file` cuando aplica.
- Ambas envs quedan registradas en `serverEffectiveEnvRegistryV0`.
- Baseline AST baja de 217 a 215.
- Bug cerrado durante verificación: `BUG-ORQ-20260704-176`. T90 falló al
  crecer `server_env_registry_v0.go` a 903 líneas; se movió la metadata nueva a
  registries por familia y el fichero queda en 878 líneas.

Verificación:

- `go test -count=1 ./cmd/orquesta-server -run 'CodexUsage|EnvRegistryAST|EnvVarsOrquestaRatchet' -v`
- `go test -count=1 ./cmd/orquesta-server -run 'ResidualGoFileBudget|CodexUsage|WorktreeSnapshot|EnvRegistryAST|EnvVarsOrquestaRatchet' -v`
- Smoke Orquesta temporal aislado: `orquesta_config_usage_smoke=passed`, con
  `required_settings` divergente detectado como `config_projection_mismatch`.
- Métrica global sigue en `env_vars_orquesta=512`.

Pendiente:

- `goal_backend`, límites Codex/agentes, OPES/bridges, codebase broker,
  rails/seguridad, domain work y `--config`/snapshot daemon.
- Ratchet AST vigente: 215.

## Continuacion Codex 2026-07-04 tarde 16

Se completa `codebase_broker` dentro de TAREA-8.1 y se usa Orquesta para
verificarlo con servidor temporal aislado.

Hecho:

- `orquesta.config.json` soporta ahora `codebase_broker`:
  `provider_kind`, `external_indexer_enabled`, `max_concurrent`, `timeout_ms`,
  `state_dir`, `watchdog_enabled`, `watchdog_stop_orphans`,
  `watchdog_orphan_min_age_seconds`, `command` y `project_name`.
- El wiring del broker central lee esa seccion con precedencia
  `env explicita > fichero > default`, sin arrancar `codebase-memory-mcp` salvo
  opt-in central `provider_kind=codebase_memory_mcp` con `state_dir`.
- El watchdog de herramientas de contexto lee la misma seccion desde el
  proyecto.
- `effective_config` publica la familia con `source=config_file`; rutas y
  comandos quedan como settings sensibles/redactados en estado publico.
- Se cerro `BUG-ORQ-20260704-177`: faltaba montar
  `/api/v0/codebase/query` en el handler HTTP directo del servidor.
- Se cerro `BUG-ORQ-20260704-178`: `scope:["."]` se descartaba y no buscaba en
  la raiz de apps externas.

Verificacion:

- `go test -count=1 ./cmd/orquesta-server -run 'CodeContextBroker|Codebase|EnvRegistryAST|EnvVarsOrquestaRatchet|ResidualGoFileBudget|EffectiveConfigV0PublicaCodebaseBroker|WatchdogLoopConfig' -v`
- `go test -count=1 ./cmd/orquesta-server -run 'ScopePunto|CodebaseQueryPublico|CodeContextBroker|Codebase|EnvRegistryAST|EnvVarsOrquestaRatchet|ResidualGoFileBudget' -v`
- `go test -count=1 ./cmd/orquesta-server` -> verde en 65.685s antes de los dos
  fixes de ruta/scope; queda pendiente repetir paquete completo tras este corte.
- Smoke Orquesta temporal aislado:
  `effective_config_codebase_broker=passed`, `codebase_status=passed`,
  `codebase_query=passed`.
- Ratchet AST sigue en 215. `env_vars_orquesta` no sube.

Pendiente:

- Repetir `go test -count=1 ./cmd/orquesta-server` y el set transversal tras
  documentacion.
- Siguiente bloque TAREA-8 sugerido por subagente: `domain_work`, con cuidado
  en `domain_work_stack_v0.go`, `config_file_v0.go`, `config.go`,
  `effective_config_v0.go` y tests.
- Siguen pendientes `goal_backend`, limites Codex/agentes, OPES/bridges,
  rails/seguridad y `--config`/snapshot daemon.

## Continuacion Codex 2026-07-04 tarde 17

Se completa `domain_work` dentro de TAREA-8.1.

Hecho:

- `orquesta.config.json` soporta ahora `domain_work`:
  `http_base_url`, `http_domain_ref`, `file_dir`, `file_enabled`,
  `http_create_path`, `http_submit_path`, `http_timeout_seconds`,
  `http_egress_mode`, `http_allowed_hosts` y `delivery_ledger_path`.
- El executor `domain_work` consume esos campos con precedencia
  `env explicita > fichero > default`.
- El backend file y el adaptador HTTP neutral quedan cubiertos por tests desde
  config file.
- El ledger de entregas domain_work lee `delivery_ledger_path` desde config.
- `effective_config` publica la familia con `source=config_file`; URL, rutas
  locales y ledger quedan como sensibles/redactados.
- La metadata se añadio en `domain_work_env_registry_v0.go`, no en el registro
  grande.
- Baseline AST baja de 215 a 202.

Verificacion:

- `go test -count=1 ./cmd/orquesta-server -run 'DomainWork|EnvRegistryAST|EnvVarsOrquestaRatchet|ResidualGoFileBudget' -v`
- Smoke Orquesta temporal aislado:
  `effective_config_domain_work=passed`, `domain_work_create=passed`,
  `domain_work_snapshot=passed`.

Pendiente:

- Repetir `go test -count=1 ./cmd/orquesta-server` y set transversal tras
  documentacion.
- Siguen pendientes `goal_backend`, limites Codex/agentes, OPES/bridges,
  rails/seguridad y `--config`/snapshot daemon.
- Ratchet AST vigente: 202.

## Continuacion Codex 2026-07-04 tarde 18

Se completa el bloque de limites Codex/agentes y supervisor residente dentro de
TAREA-8.1.

Hecho:

- `orquesta.config.json` soporta ahora `server_supervisor`:
  `max_runs_per_tick`, `max_executions_per_tick`, `queue_limit`,
  `drain_max_dispatches`, `drain_max_commands`, `drain_max_outbox` y
  `drain_max_external_waits`.
- Añade `server_idle_self_improvement.max_requests` y
  `server_resident_director.max_actions`.
- Amplia `codex_runtime` con `execution_mode`, `reasoning_effort`,
  `max_expected_seconds`, `max_batch_ready` y `max_concurrency`.
- Añade `codex_director.wave_agents`, `max_subagents_per_agent` y
  `recursive_agent_budget`.
- La precedencia queda `env explicita > fichero > default`; `serial` sigue
  capando batch/concurrency/wave agents a `1`.
- `effective_config` publica esos limites con `source=config_file`.
- `ORQUESTA_CODEX_MAX_EXPECTED_SECONDS` y
  `ORQUESTA_SERVER_DRAIN_MAX_COMMANDS` quedan visibles en effective config.
- Se bajo el ratchet AST de 202 a 201.
- Se cerro `BUG-ORQ-20260704-179`: el status publico redactaba
  `ORQUESTA_SERVER_DRAIN_MAX_COMMANDS` por contener `COMMAND`; ahora los
  limites numericos `*_MAX_COMMANDS` no se redactan salvo marca sensible
  explicita.

Verificacion:

- `go test -count=1 ./cmd/orquesta-server -run 'FicheroCanonico|LimitesCodex|Capacidad|CodexRuntime|CodexStackCapacity|CodexDirectorWaveConfig|ModoSerial|SupervisorResidente|EnvRegistryAST|EnvVarsOrquestaRatchet|ResidualGoFileBudget' -v`
- `go test -count=1 ./modulos/orquesta-server -run 'PublicServerEffectiveConfig|StatusPublic' -v`
- Smoke Orquesta temporal aislado:
  `orquesta_config_limits_smoke=passed`.

Pendiente:

- Repetir `go test -count=1 ./cmd/orquesta-server` y set transversal tras
  documentacion.
- Siguen pendientes `goal_backend`, OPES/bridges, rails/seguridad y
  `--config`/snapshot daemon.

## Continuacion Codex 2026-07-04 tarde 19

Se completa el bloque `--config`/snapshot daemon de TAREA-8.1.

Hecho:

- `orquesta-server run --config <path>`, `start --config <path>`,
  `status --config <path>` y `stop --config <path>` cargan el fichero canonico
  explicito.
- Si no hay `ORQUESTA_CODEX_PROJECT_WORKDIR`, `--config` usa el directorio del
  fichero como fallback de proyecto; si la env existe, sigue ganando la env.
- `start --config` escribe un snapshot validado en
  `state/config-snapshots/orquesta.config.json` y arranca el hijo como
  `run --config <snapshot>`, sin introducir env nueva.
- `ConfigV0` conserva `ProjectConfigFilePath` como dato interno para que
  `effective_config` marque `source=config_file` aunque el daemon lea desde
  snapshot.
- Se cerro `BUG-ORQ-20260704-180`: `start` devolvia `readiness_timeout` cuando
  readiness estricta estaba degradada por conectores externos aunque el
  servidor ya estaba `startup_ready`.

Uso de Orquesta:

- `prepare-run` real contra servidor temporal con
  `ORQUESTA_CODEX_GOAL_BACKEND=claude_file_control`; primer intento rechazo
  `worktree_isolated=false`, segundo accepted con
  `goal_ref=goal-ref-task-autoprogramming-00f701576d89-g01`.
- Evidencia:
  `/tmp/orquesta-config-cli.kXgni4/prepare.out.json` y
  `/tmp/orquesta-config-cli.kXgni4/claude-goal/`.

Verificacion:

- `go test -count=1 ./cmd/orquesta-server -run 'ReadinessOK|ConfigPath|DaemonRunArgs|DaemonStart|Status|CommandPublic|EnvRegistryAST|EnvVarsOrquestaRatchet' -v`
- Smoke real temporal:
  `orquesta_start_config_snapshot_smoke=passed root=/tmp/orquesta-start-config-smoke.fImpbt`.
  El smoke arranca con config, muta el fichero original y verifica que el daemon
  sigue usando el snapshot.

Pendiente:

- Repetir `go test -count=1 ./cmd/orquesta-server`, set transversal y
  `go test -count=1 ./...`.
- Resto TAREA-8: `goal_backend`, OPES/bridges y rails/seguridad.

## Continuacion Codex 2026-07-04 tarde 20

Se completa el bloque `goal_backend` de TAREA-8.1.

Hecho:

- `orquesta.config.json` soporta ahora `goal_backend`:
  `kind`, `timeout_ms`, `preflight_timeout_ms` y
  `allow_app_server_proxy_diagnostic`.
- Se consolidan `ORQUESTA_CODEX_GOAL_BACKEND`,
  `ORQUESTA_CODEX_GOAL_TIMEOUT_MS`,
  `ORQUESTA_CODEX_GOAL_PREFLIGHT_TIMEOUT_MS` y
  `ORQUESTA_ALLOW_APP_SERVER_PROXY_DIAGNOSTIC` con precedencia
  `env explicita > fichero > default`.
- El wiring de Codex app-server, Claude file/process y Gemini file/process
  resuelve el backend desde el fichero cargado/snapshot cuando existe.
- La derivacion de `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_GOAL_FIRST_ENABLED`
  y el diagnostico `external_work_goal_backend_required` ya usan el backend
  efectivo, no solo env cruda.
- `cleanup` de app-server tmux tras fallo de startup y la guarda
  `self_programming_only` respetan el backend efectivo.
- `effective_config` publica `source=config_file` para backend, timeouts y
  opt-in proxy.

Uso de Orquesta:

- Smoke real con `orquesta-server start --config` y
  `goal_backend.kind=claude_file_control`.
- `POST /api/v0/autoprogramming/prepare-run` aceptado con
  `goal_ref=goal-ref-task-autoprogramming-88e68c22e8c4-g01`.
- El primer assert del arnes esperaba `.status=="accepted"` aunque el contrato
  real devuelve `accepted:true`; no fue bug de Orquesta. Se materializo resultado
  durable temporal en el write-set del smoke y se cerro el daemon
  cooperativamente.

Verificacion:

- `go test -count=1 ./cmd/orquesta-server -run 'GoalBackend|GoalFirst|ExternalWorkLegacy|EnvRegistryAST|EnvVarsOrquestaRatchet|ResidualGoFileBudget' -v`
- Smoke real temporal:
  `orquesta_goal_backend_config_smoke=passed root=/tmp/orquesta-goal-backend-config-smoke.7TWwyo`.

Pendiente:

- Repetir `go test -count=1 ./cmd/orquesta-server`, set transversal y
  `go test -count=1 ./...`.
- Resto TAREA-8: OPES/bridges, OPES registry y rails/egress.

## Continuacion Codex 2026-07-04 tarde 21

Se completa el bloque `rails_security` + `egress_sanitizer` de TAREA-8.1.

Hecho:

- `orquesta.config.json` soporta ahora `rails_security.security_mode`,
  `rails_security.rails_mode`, `rails_security.detail_prohibited_rails` y
  `rails_security.detail_prohibited_rails_scope`.
- Aunque el fichero pida `rails_mode=enforced` o `detail_prohibited_rails=on`,
  la normalizacion mantiene `ORQUESTA_RAILS_MODE=offline` y
  `ORQUESTA_DETAIL_PROHIBITED_RAILS=off`; no se reactivan rails duros.
- `orquesta.config.json` soporta `egress_sanitizer.enabled`,
  `sanitizer_ref`, `local_model.*` y `sidecar.*`.
- `effective_config` publica `source=config_file` para estas familias y redacta
  rutas, comandos, endpoints y refs sensibles.
- `buildStackFromEnvV0` cablea el sanitizer desde el fichero/snapshot, sin
  meter opciones concretas en core.
- Se cerro `BUG-ORQ-20260704-181`: en `start --config`, las envs de rails
  proyectadas al daemon ocultaban el origen `config_file`. Ahora solo el
  snapshot daemon reclasifica esa proyeccion como `config_file`; un
  `run --config` manual con env explicita sigue siendo `explicit`.

Uso de Orquesta:

- Smoke real con `orquesta-server start --config` y configuracion canonica de
  rails/egress.
- Verificado que `/status` expone rails y egress con `source=config_file` y que
  no filtra comando, ruta de modelo ni endpoint local.

Verificacion:

- `go test -count=1 ./cmd/orquesta-server -run 'Rails|EgressSanitizer|EnvRegistryAST|EnvVarsOrquestaRatchet|ResidualGoFileBudget|DaemonStartEnvironment' -v`
- Smoke real temporal:
  `orquesta_rails_egress_config_smoke=passed root=/tmp/orquesta-rails-egress-config-smoke.hgpepi`.

Pendiente:

- Repetir `go test -count=1 ./cmd/orquesta-server`, set transversal y
  `go test -count=1 ./...`.
- Resto TAREA-8: OPES/bridge, OPES registry y limpieza final de docs/handoff.

## Continuacion Codex 2026-07-04 tarde 22

Se completa el bloque OPES seguro de TAREA-8.1.

Hecho:

- Se extrae `cmd/orquesta-server/opes_config_file_v0.go` para que las structs
  OPES no hinchen `config_file_v0.go`; el fichero central baja a 846 lineas y
  el ratchet T90 vuelve a verde.
- `orquesta.config.json` soporta `opes.base_url`.
- `orquesta.config.json` soporta campos seguros de `opes_bridge`: scope
  (`job_type`, `job_type_sequence`, `job_ref`, `program_id`, `topic_id`,
  `correlation_id`), limites (`limit`, `timeout_seconds`, `priority`,
  `interval_seconds`, `initial_delay_seconds`, `max_ticks`), espera residente
  (`wait_resident_seconds`, `wait_resident_interval_ms`),
  `supervise_submitted`, `dry_run` y compatibilidad runtime declarativa.
- `orquesta.config.json` soporta `opes_registry_finalpkg.*` y
  `opes_topic_registry.*`; rutas locales se publican redactadas.
- `effective_config` baja el ratchet AST de 201 a 191 al dejar de leer env
  cruda de OPES bridge en el bloque de estado efectivo.
- El parser de `opes_bridge.job_type_sequence` conserva la normalizacion
  historica de comas, punto y coma, espacios y saltos de linea.

Frontera deliberada:

- Quedan env/script-only por seguridad operativa e idempotencia:
  `ORQUESTA_OPES_BRIDGE_ENABLED`, `ORQUESTA_OPES_BRIDGE_CONFIRM`,
  `ORQUESTA_OPES_TEMPORAL_CONFIRM`,
  `ORQUESTA_OPES_BRIDGE_PRODUCTIVE_CONFIRM`,
  `ORQUESTA_OPES_BRIDGE_ALLOW_UNFILTERED`,
  `ORQUESTA_OPES_BRIDGE_DESTINATION_EVIDENCE_REF`,
  `ORQUESTA_OPES_BRIDGE_INPUT_LEDGER_DISABLED`,
  `ORQUESTA_OPES_BRIDGE_INPUT_LEDGER_PATH` y comandos/preflight/readiness live
  de speech/remote QA. No se migran a fichero en este corte para no abrir
  produccion, saltar filtros ni romper idempotencia de ledger desde un config
  persistente.

Uso de Orquesta:

- Smoke real con `orquesta-server start --config` combinando OPES bridge seguro,
  finalpkg y topic registry. Solo inspecciona `/status`; no drena OPES ni crea
  runs.

Verificacion:

- Focal `OPESBridgeConfigLeeFicheroCanonico|OPESDrainConfig|OPESBridgeLoopConfig|OPESSpeechSynthesis|ServerConfigFromEnvV0PublicaConfiguracionEfectivaCanonica|ServerConfigFromEnvV0PublicaOPESSpeechSynthesisPreflightRedactado|EnvRegistryAST|EnvVarsOrquestaRatchet|ResidualGoFileBudget`.
- Smoke real temporal:
  `orquesta_opes_config_smoke=passed root=/tmp/orquesta-opes-config-smoke.R6mcB5`.
- `go test -count=1 ./cmd/orquesta-server` -> verde en 66.096s.
- `git diff --check` -> verde.
- `bash scripts/orquesta_metricas_deuda.sh --json` ->
  `{"env_vars_orquesta":512,"endpoints_status":16,"interfaces_estado":65,"modulos_director":17}`.
- `go test -count=1 ./...` -> verde.
- Comprobado que no quedan daemons temporales de smoke ni
  `codebase-memory-mcp` vivos.

Pendiente:

- Repetir set transversal y `go test -count=1 ./...` tras documentacion.
- Si se quiere migrar confirmaciones/ledger/comandos a fichero, debe hacerse
  como tarea gobernada con nuevas guardas de operador, no como simple traslado
  de envs.

## Continuacion Codex 2026-07-04 tarde 23

Se abre TAREA-9 con herramienta reproducible y primera ola segura de limpieza.

Hecho:

- Nuevo `scripts/orquesta_auditoria_codigo.sh`.
  - Genera JSON y SQLite local como reporte/caché derivada.
  - Mide `deadcode`, módulos huérfanos, helpers copiados y ficheros grandes.
  - No escribe en el repo analizado salvo los artefactos de salida indicados.
- Nuevo `scripts/test_orquesta_auditoria_codigo.sh`.
- `scripts/orquesta_smoke_nightly.sh` ejecuta auditoría por defecto y añade
  ratchet contra el último nightly verde: falla si suben
  `deadcode_candidates` o `helper_duplicate_definitions`.
- `scripts/test_orquesta_smoke_nightly.sh` cubre auditoría fake, baseline verde
  y fallo de ratchet simulado.
- `BUG-ORQ-20260704-182` cerrado: la primera versión recorría `**/*.go` desde
  raíz y podía colgar auditorías en workspaces grandes; ahora solo recorre
  `cmd/` y `modulos/`.
- Primera ola `orquesta-deploy`: eliminados siete helpers exportados muertos
  `Has*IssueV0` sin consumidores internos.

Números actuales:

- Medición fresca con `deadcode` instalado en GOPATH:
  `deadcode_candidates=1230`, `helper_duplicate_definitions=288`,
  `orphan_modules=1`, `large_files_over_800=17`.
- `modulos/orquesta-deploy=87` candidatos frente a 94 en el snapshot histórico
  de Claude.
- El snapshot histórico marcaba 1188 globales; no usarlo como ratchet tras los
  cambios de hoy. La línea base reproducible es la salida del script.

Verificación focal:

- `scripts/test_orquesta_auditoria_codigo.sh` -> verde.
- `scripts/test_orquesta_smoke_nightly.sh` -> verde.
- `GOFLAGS=-buildvcs=false go test -count=1 ./modulos/orquesta-deploy ./modulos/orquesta-app-planner` -> verde.

Pendiente:

- Ejecutar set transversal final tras TAREA-2.
- TAREA-2: añadir operaciones estructuradas del analizador
  (`callers`, `imports`, `module_exports`, `relevant_snippets`) y obligación en
  prompt Goal para write-sets de código.

## Continuacion Codex 2026-07-04 tarde 24

TAREA-2 primer parche implementado.

Hecho:

- `CodeContextQueryPortV0` acepta nuevos `query_kind`: `callers`, `imports`,
  `module_exports` y `relevant_snippets`.
- MCP `orquesta.codebase.query.v0` publica esos enums.
- `cmd/orquesta-server` añade `code_context_structured_fallback_v0.go`:
  adaptador con `go/parser` para llamadas entrantes simples, imports, exports
  de modulo y snippets relevantes. No usa `codebase-memory-mcp` ni arranca
  indexadores.
- `codebase-memory-mcp` opt-in ya no rompe por esos `query_kind`; degrada a
  búsqueda central si se usa ese proveedor.
- El prompt Goal añade regla estable solo con write-set de codigo: consultar
  `orquesta.codebase.query.v0` antes de leer ficheros completos y no arrancar
  indexadores propios.
- Docs locales de `modulos/orquesta-context` actualizados: CTX-P008/CTX-009.

Verificacion focal:

- `GOFLAGS=-buildvcs=false go test -count=1 ./modulos/orquesta-context ./modulos/orquesta-mcp ./modulos/orquesta-runtime-codex-goal` -> verde.
- `GOFLAGS=-buildvcs=false go test -count=1 ./cmd/orquesta-server -run 'Test(ServerRGCodeContextProviderV0|MCPCodebaseQuery|BuildServerAppHandlerV0CodebaseQuery)'` -> verde.
- Smoke REST real con `orquesta-server` temporal y
  `POST /api/v0/codebase/query`:
  `callers`, `imports`, `module_exports` y `relevant_snippets` -> verde,
  `codebase_query_structured_smoke=passed`.
- `GOFLAGS=-buildvcs=false go test -count=1 ./...` -> verde.
- `scripts/test_orquesta_auditoria_codigo.sh`,
  `scripts/test_orquesta_smoke_nightly.sh`,
  `scripts/orquesta_metricas_deuda.sh --json` -> verde;
  metricas `{"env_vars_orquesta":512,"endpoints_status":16,"interfaces_estado":65,"modulos_director":17}`.
- `git diff --check` -> verde.
- Comprobado despues del smoke: sin `orquesta-server run` ni
  `codebase-memory-mcp` vivos.

Pendiente TAREA-2:

- Smoke sandbox real demostrando uso de `orquesta.codebase.query.v0`.
- Métrica real de tokens antes/después contra baseline 165k.
- Persistencia/caché derivada alineada con el motor único del despliegue cuando
  exista Postgres; hoy sigue siendo caché local de servidor.

## Continuacion Codex 2026-07-04 tarde 25

TAREA-2: preparación automática de analizador para goals de código.

Hecho:

- `modulos/orquesta-app-codex-stack` envuelve `GoalLauncher` y
  `GoalReworkLauncher` cuando la composición tiene `CodeContext`.
- Antes de lanzar un goal con write-set de código, Orquesta ejecuta una consulta
  `repo_map` por `CodeContextQueryPortV0` con scope del write-set para calentar
  cache/broker.
- El spec lanzado recibe `ContextRef` `code_context_prepared:repo_map:<hash>`
  y evidencias del broker; si el broker falla, recibe
  `code_context_prepare_failed` y el goal no se bloquea.
- El receipt de lanzamiento propaga `ContextBudget.CodeContextCacheStatus`
  cuando existe.
- Los goals documentales no disparan esta precarga.
- `BUG-ORQ-20260704-183` cerrado durante el parche: la primera version añadia
  un helper `compact*` y subia `helper_duplicate_definitions` de 288 a 289; se
  renombro a `uniqueCodeContextGoalStringsV0` y la auditoria volvio a 288.

Verificacion focal:

- `GOFLAGS=-buildvcs=false go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestBuildDirectorPortsV0.*CodeContext|TestBuildDirectorPortsV0CableaAppGoalLauncher'` -> verde.
- Auditoria fresca posterior:
  `deadcode_candidates=1230`, `helper_duplicate_definitions=288`,
  `orphan_modules=1`, `large_files_over_800=17`.

Pendiente TAREA-2:

- Smoke sandbox real demostrando uso de `orquesta.codebase.query.v0` por un
  goal.
- Métrica real de tokens antes/después contra baseline 165k.
- Persistencia/caché derivada alineada con el motor único del despliegue cuando
  exista Postgres; hoy sigue siendo caché local de servidor.

## Continuacion Codex 2026-07-04 tarde 26

TAREA-9 segunda ola segura en `orquesta-capacity`.

Hecho:

- Inlinados helpers privados de un solo uso en
  `modulos/orquesta-capacity/capacity_policy_v0.go`.
- Inlinados `joinModelEscalationPathV0` e
  `isForbiddenModelEscalationKeyV0` en
  `modulos/orquesta-capacity/model_escalation_policy_helpers_v0.go`.
- No se tocaron APIs exportadas ni helpers con consumidores internos.

Verificacion:

- `GOFLAGS=-buildvcs=false go test -count=1 ./modulos/orquesta-capacity ./modulos/orquesta-app-codex-stack ./modulos/orquesta-director` -> verde.
- Auditoria fresca:
  `deadcode_candidates=1228`, `helper_duplicate_definitions=288`,
  `orphan_modules=1`, `large_files_over_800=17`,
  `modulos/orquesta-capacity=89`.

## Continuacion Codex 2026-07-04 tarde 27

TAREA-9 ratchet/auditoria y conector Gemini.

Hecho:

- `BUG-ORQ-20260704-184` cerrado: `scripts/orquesta_auditoria_codigo.sh`
  localiza `deadcode` tambien en `GOBIN`/`GOPATH/bin`, no solo en `PATH`.
- `BUG-ORQ-20260704-185` cerrado: `scripts/orquesta_smoke_nightly.sh` ya no
  acepta auditorias con schema incorrecto, metricas requeridas ausentes o
  fuente `snapshot_file`/`unavailable`/`deadcode_tool_failed`.
- `BUG-ORQ-20260704-186` cerrado: `geminiRuntimeConfigV0` usa
  `ORQUESTA_GEMINI_OUTPUT_FORMAT` con default `text`, igual que Claude y el
  backend goal Gemini.
- Residuales documentados para Claude: `app_server_proxy` sigue como valor
  historico de diagnostico no operativo, el default local OPES debe moverse a
  config canonica obligatoria cuando se cierre TAREA-8 completa, y los prompts
  Claude/Gemini quedan pendientes si i18n cubre textos enviados a proveedores.

Verificacion:

- `bash -n scripts/orquesta_auditoria_codigo.sh scripts/test_orquesta_auditoria_codigo.sh scripts/orquesta_smoke_nightly.sh scripts/test_orquesta_smoke_nightly.sh` -> verde.
- `scripts/test_orquesta_auditoria_codigo.sh` -> verde.
- `scripts/test_orquesta_smoke_nightly.sh` -> verde.
- `GOFLAGS=-buildvcs=false go test -count=1 ./cmd/orquesta-server -run 'TestGeminiRuntimeConfigV0|GoalBackend|Claude|Gemini'` -> verde.
- Auditoria viva:
  `deadcode_source=deadcode_tool`, `deadcode_candidates=1228`,
  `helper_duplicate_definitions=288`, `orphan_modules=1`,
  `large_files_over_800=17`.
- `GOFLAGS=-buildvcs=false go test -count=1 ./...` -> verde.
- Smoke Orquesta real por API publica con servidor temporal:
  `POST /api/v0/codebase/query` (`schema_version=code_context_query.v0`,
  `query_kind=relevant_snippets`, `query=geminiRuntimeConfigV0`) -> verde,
  `provider_kind=fallback_rg`, `results=1`, shutdown sin proceso vivo. Un
  primer intento fallo con HTTP 400 por payload manual con schema incorrecto
  `orquesta_code_context_query.v0`; no fue bug de producto.

## Continuacion Codex 2026-07-04 tarde 28

Conector OPES/project workdir.

Hecho:

- `BUG-ORQ-20260704-187` cerrado: `external_work` guard y topic registry dejan
  de inyectar `/home/alberto/Trabajo/OPES` como default local.
- La fuente queda canonica: `opes.project_workdir` en `orquesta.config.json` o
  `ORQUESTA_OPES_PROJECT_WORKDIR`.
- Si OPES no esta configurado, no se inventa ruta local; si esta configurado,
  se mantiene la guarda de `project_work_dir` y el descubrimiento de
  `registro_trabajo_temas.py` bajo el workspace OPES.

Verificacion:

- `GOFLAGS=-buildvcs=false go test -count=1 ./cmd/orquesta-server -run 'TestExternalWorkRunProjectWorkDirGuardConfigV0|TestOPESTopicRegistryConfig|TestOPESTopicRegistryEffectiveConfig|TestServerConfigFromEnvV0ContextoOPES|TestServerConfigFromEnvV0PermiteOPES'` -> verde.
- `GOFLAGS=-buildvcs=false go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestExternalWorkRunProjectWorkDirGuard'` -> verde.

## Continuacion Codex 2026-07-04 tarde 29

i18n goal-first Claude/Gemini.

Hecho:

- `BUG-ORQ-20260704-188` cerrado parcialmente para goal-first: los backends
  Claude/Gemini aceptan `PromptLocale`.
- Builders nuevos `BuildClaudeGoalPromptWithLocaleV0` y
  `BuildGeminiGoalPromptWithLocaleV0` soportan `es-ES` por defecto y `en-*`.
- El protocolo durable del resultado goal-first se localiza en es/en sin tocar
  el contrato JSON ni `orquesta-goal`.
- Composicion: `goal_backend.prompt_locale` se lee desde
  `orquesta.config.json` y se cablea a `file_control` y `process`, sin nuevas
  variables `ORQUESTA_*`.
- Residual documentado: prompts legacy de agente `Build*AgentPrompt*` siguen en
  español si se exige i18n fuera de goal-first.

Verificacion:

- `GOFLAGS=-buildvcs=false go test -count=1 ./modulos/orquesta-runtime-claude ./modulos/orquesta-runtime-gemini ./cmd/orquesta-server -run 'Test(Build(Claude|Gemini)GoalPromptWithLocale|ClaudeGoalBackendV0Launch|GeminiGoalBackendV0Launch|ServerGoalBackendFromEnvV0(Claude|Gemini)FileControl|ClaudeRuntimeConfigV0|GeminiRuntimeConfigV0)'` -> verde.
- `GOFLAGS=-buildvcs=false go test -count=1 ./cmd/orquesta-server -run 'EnvVarsOrquestaRatchet|EnvRegistryAST|ClaudeRuntimeConfig|GeminiRuntimeConfig|ServerGoalBackendFromEnvV0(Claude|Gemini)FileControl'` -> verde.
- `scripts/orquesta_auditoria_codigo.sh` -> `deadcode_candidates=1228`,
  `helper_duplicate_definitions=288`.

## Continuacion Codex 2026-07-04 tarde 30

Smoke REST/readiness.

Hecho:

- `BUG-ORQ-20260704-189` cerrado: el smoke REST local fallaba porque trataba
  HTTP 503 de readiness como servidor no listo, aunque el JSON publicara
  `startup_ready=true`, `status=running` y `availability_status=running`.
- `scripts/lib/smoke_common.sh` incorpora `smoke_orquesta_readiness_ok`, que
  acepta 2xx o el 503 degradado compatible con el contrato vigente.
- `scripts/smoke_orquesta_server_rest_director.sh` usa el helper comun en vez
  de `curl -f` directo contra readiness.

Verificacion:

- `bash -n scripts/lib/smoke_common.sh scripts/smoke_orquesta_server_rest_director.sh` -> verde.
- `GOFLAGS=-buildvcs=false go test -count=1 ./cmd/orquesta-server -run 'TestSmokeCommon(Readiness|Shutdown)'` -> verde.
- `ORQUESTA_LEGACY_DIRECTOR_LOOP_SMOKE_CONFIRM=1 GOFLAGS=-buildvcs=false ./scripts/smoke_orquesta_server_rest_director.sh` -> verde; `POST /api/v0/apps/director -> HTTP 200`, `agents_started=1`, progress/usage/process refs verificados y parada limpia.
- `GOFLAGS=-buildvcs=false go test -count=1 ./...` -> verde.

## Continuacion Codex 2026-07-04 tarde 31

Wizard dominio y OPES finalpkg live-config.

Hecho:

- Avance parcial del wizard: la capa web soporta mas packs de dominio
  (`inventario`, `notas/documentos`, `tareas/proyectos`, `finanzas`, `crm`,
  `reservas`, `salud`, `educacion`, `comunidad`, `iot`, `media`,
  `facturacion` y `ecommerce`) y puede emitir varias preguntas R3 de dominio
  sin duplicar campos. La pregunta abierta de dominio solo aparece cuando el
  objetivo no encaja con ningun pack conocido.
- `BUG-ORQ-20260704-190` cerrado: OPES `finalpkg` ya no usa
  `course_id`, `template_run_ref` ni `template_topic_id` hardcodeados cuando
  `dry_run=false`; esos defaults quedan restringidos a dry-run/fixtures.
- El cierre se hizo como reparacion local acotada de composicion y despues se
  valido con Orquesta por API publica temporal, sin tocar OPES productivo.

Verificacion:

- `GOFLAGS=-buildvcs=false go test -count=1 ./modulos/orquesta-web -run 'TestWizard|TestNuevaAppIntakeGuidedResponseV0IncluyeWizardRicoV0|TestNuevaAppIntakeGuidedResponseV0AplicaWizardAnswersV0|TestNuevaAppI18nCatalogV0CatalogosCubrenClavesRequeridas'` -> verde.
- `GOFLAGS=-buildvcs=false go test -count=1 ./cmd/orquesta-server -run 'TestOPESRegistryFinalPkgConfig|TestOPESRegistryFinalPkgDryRunSelectsCandidatesWithoutSubmit|TestOPESRegistryFinalPkgPostsExternalWorkRunEnvelope'` -> verde.
- Smoke temporal con `orquesta-server run --config <temp>`:
  `POST /api/v0/apps/intake/guided-turn` -> schema
  `web_nueva_app_intake_guided_response.v0`, `wizard_questions=10`;
  `POST /api/v0/autoprogramming/status` -> `estado=ok`; readiness `running`;
  el loop residente `opes-registry-finalpkg` queda bloqueado con
  `opes_registry_finalpkg_config_incomplete` y las tres claves live faltantes.
- `GOFLAGS=-buildvcs=false go test -count=1 ./...` -> verde.
- Auditoria viva: `deadcode_candidates=1228`,
  `helper_duplicate_definitions=288`, `orphan_modules=1`,
  `large_files_over_800=17`; deuda config `env_vars_orquesta=512`.

Pendiente:

- Wizard no esta completo: faltan U1-U12 completos, T1-T8 efectivos, motor de
  exclusion runtime, ayudas i18n/glosario y bot RAG determinista/LLM opt-in.
- `BUG-058/066/075` siguen abiertos: falta smoke OPES temporal real con
  external-work/observe, proveedor, derivados, paquete final y ausencia de
  reescritura tardia.

## Claude: análisis de retomables en el deadcode (2026-07-04 noche)

Analizadas las candidatas de la auditoría: tres piezas son inversión parada,
no basura — (1) wizard viejo de intake: absorber cierre real
SolicitarNuevaAppV0 + rutas punteadas en el wizard nuevo y borrar el resto;
(2) HeuristicAutonomousDirectorPolicyV0 + quality policy: cerebro de
dimensionado de equipo/paralelismo/calidad sin invocar — retomar para
TAREA-3/paralelismo por evidencia; (3) codeContextBrokerFromEnvV0: el wiring
completo del analizador (rg fallback + codebase-MCP + cache + leases) ya
existe y no se llama — la TAREA-2 parte de ahí. Detalle e instrucciones en
docs/instrucciones_director_codex_2026-07-04.md TAREA-10. Regla nueva para
TAREA-9: clasificar antes de borrar.

## AVISO OPERATIVO: migración a servidor remoto (2026-07-04 noche)

Orden del operador: este equipo local SE APAGA. Todo el trabajo continúa en
el servidor remoto. Para el director Codex:

1. NO iniciar tareas nuevas en este equipo local. Termina el corte actual,
   commitea TODO (incluida obra a medias, marcada WIP si hace falta) y
   registra aquí el estado exacto en que lo dejas.
2. La rama `trabajo/plataforma-agentes` se empuja a origin (GitHub
   aavidad/orquestador) como canal de sincronización; el servidor remoto
   hace pull de ahí.
3. El relevo continúa en el servidor: mismos documentos de coordinación
   (esta bitácora, instrucciones del director, inventario). La cola vigente
   es: TAREA-2 AMPLIADA (analizador, partir del wiring TAREA-10.3),
   TAREA-8 olas de envs, TAREA-9 poda con ratchet, TAREA-10 retomas,
   wizard pendiente (U/T/exclusión/glosario/bot RAG), smoke OPES real
   (058/066/075).
4. Los placeholders de goal results y checkpoints sueltos de los pilotajes
   T1-T7A quedan commiteados como evidencia citada por el inventario.

## Corte Codex local: TAREA-10.3 y relevo remoto (2026-07-04 noche)

Revision read-only con subagentes sobre la nota nueva de Claude:

- `TAREA-10.3` no esta realmente sin cablear en el HEAD actual. El constructor
  `codeContextBrokerWiringFromEnvV0` se llama desde el arranque del stack,
  inyecta `CodeContext`/leases en `BuildStackV0`, expone
  `/api/v0/codebase/query` y `orquesta.codebase.query.v0`, y el launcher
  goal-first precarga `repo_map` para write-sets de codigo.
- La conclusion de Claude sigue siendo util como guardia contra reescrituras:
  no rehacer el broker. La siguiente accion correcta es test focal
  extremo-a-extremo del wiring real y, si se confirma necesario, proyectar la
  herramienta MCP local al `CODEX_HOME` aislado del agente sin saltarse el
  broker central.
- Subagentes usados: auditoria de wiring servidor/stack y auditoria de
  superficie API/MCP/goal. Ambos coinciden en que el broker central existe y
  esta cableado; hueco real: falta test que construya
  `buildStackFromEnvWithGoalBackendV0`, consulte el binding MCP/HTTP con
  `ProjectWorkDir` real y demuestre `code_context_prepared:*` en el goal.
- `scripts/bootstrap_agent_tooling.sh --status` devuelve
  `attention_required` porque hay un proceso `codebase-memory-mcp` vivo. No se
  paro en este corte porque pertenece a una sesion Claude activa; el servidor
  remoto debe revisar procesos vivos antes de lanzar trabajo largo.

Estado para el servidor remoto:

- Rama local: `trabajo/plataforma-agentes`, con commit de Claude
  `5bc76f1d` y este corte de handoff.
- Artefactos `checkpoint_started.txt` y
  `orquesta_goal_result_goal-ref-task-autoprogramming-*.json` de los pilotajes
  T2/T3/T4/T5/T6/T7A/T10 quedan commiteados solo como evidencia de arranque o
  interrupcion (`status=invalid` en los JSON). No cerrar esas tareas sin nueva
  evidencia.
- Siguiente parche recomendado: test focal TAREA-10.3; despues TAREA-10.2
  (politica autonoma de paralelismo/coste) y TAREA-10.1 (absorber cierre real
  del wizard viejo y eliminar duplicado).

## Actualizacion Codex 2026-07-04 noche 32

TAREA-10.3 verificada con codigo, test y smoke vivo:

- Anado `TestBuildStackFromEnvV0CableaBrokerContextoEnMCPHTTPYGoalV0` en
  `cmd/orquesta-server/stack_wiring_test.go`.
- El test construye `buildStackFromEnvWithGoalBackendV0` desde env real con
  `ProjectWorkDir` temporal, crea fixture `cmd/demo/service.go`, comprueba
  `stack.CodeContext`, ejecuta el binding MCP `CodebaseQuery`, monta
  `buildServerAppHandlerV0`, hace POST a `/api/v0/codebase/query` y lanza un
  goal fake de codigo verificando `code_context_prepared:repo_map:*` y
  `ContextBudget.CodeContextCacheStatus`.
- Smoke Orquesta real temporal: servidor aislado con
  `ORQUESTA_SERVER_ADDR=127.0.0.1:0`,
  `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_DISABLED=true`,
  `ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED=false` y
  `ORQUESTA_OPES_REGISTRY_FINALPKG_ENABLED=false`; `POST
  /api/v0/codebase/query` con `query_kind=repo_map`,
  `query=CodebaseWiringDemoV0`, `scope=["cmd/demo"]` -> HTTP 200,
  `estado=ok`, `provider_kind=fallback_rg`, `results=1`,
  `cache_status=stored`.

Verificacion:

- `GOFLAGS=-buildvcs=false go test -count=1 ./cmd/orquesta-server -run 'TestBuildStackFromEnvV0CableaBrokerContextoEnMCPHTTPYGoalV0'` -> verde.
- `GOFLAGS=-buildvcs=false go test -count=1 ./cmd/orquesta-server -run 'TestBuildStackFromEnvV0CableaBrokerContextoEnMCPHTTPYGoalV0|TestBuildServerAppHandlerV0CodebaseQueryPublicoUsaBindingDirecto|TestServerCodeContext|TestCodeContextBroker'` -> verde.
- `git diff --check` -> verde.
- `GOFLAGS=-buildvcs=false go test -count=1 ./...` -> verde.

Estado: TAREA-10.3 queda cerrada para el cableado servidor/MCP/HTTP/goal. No
rehacer el broker. Residual separado si producto lo exige: proyeccion de MCP
local en `CODEX_HOME` aislado del agente, siempre pasando por el broker
central.

TAREA-10.1 avanzadilla de cierre del wizard nuevo:

- Subagente read-only verifico que el wizard nuevo ya aplica rutas punteadas
  mas ricas que el viejo (`integraciones.N.*`,
  `datos.tipos_detallados.N.*`, `datos.fuentes.N.*`, `datos.storage.N.*`) via
  `ApplyWebNuevaAppWizardAnswersV0` -> `ApplyDecisionV0`.
- El cierre `SolicitarNuevaAppV0(draft, now)` no debe meterse directo en
  `orquesta-web`; ya existe por puerto/adaptador: `RESTSolicitarNuevaAppClientV0`
  contra `orquesta-factory-http`.
- Anado `TestWizardNuevoListoCierraFactoryRealPorPuertoV0` en
  `modulos/orquesta-web/nueva_app_wizard_rest_flow_v0_test.go`: crea sesion
  wizard, acepta recomendaciones hasta `LaunchReady`, valida `SpecPreview`,
  comprueba conector `calendar` preservado desde rutas punteadas, llama al
  factory HTTP real por cliente REST y exige spec/backlog reales validos.

Verificacion:

- `GOFLAGS=-buildvcs=false go test -count=1 ./modulos/orquesta-web -run 'TestWizardNuevoListoCierraFactoryRealPorPuertoV0|TestWizardAgendaDesdeSoloObjetivoV0|TestNuevaAppRESTFlowV0ValidaWebRESTFactory|TestNuevaAppIntakeGuidedResponseV0AplicaWizardAnswersV0'` -> verde.
- `GOFLAGS=-buildvcs=false go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-app-director-intake ./modulos/orquesta-factory-http -run 'Test(AdvanceAppDirectorIntakeWizardV0|ApplyWebNuevaAppIntake|WebNuevaAppIntakeSessionV0|NuevaAppIntakeGuided|Wizard|NuevaAppRESTFlow|AppSpecHTTPV0PostValido)'` -> verde.

Estado: el hueco funcional "wizard nuevo listo -> cierre factory real" queda
cubierto por test. No borrar aun `modulos/orquesta-app-director-intake` ni sus
`wizard_*.go`: el modulo tiene consumidores historicos y el borrado exige
deprecacion/refactor con evidencia propia.

## Actualizacion Codex 2026-07-04 noche 33

TAREA-10.2 queda cerrada para el cableado de politica autonoma en goal-first:

- `StartAppDirectorPortsV0` ahora expone `AutonomousDirectorPolicy`.
- `startAppDirectorGoalFirstV0` invoca la politica antes de lanzar el goal,
  con `DirectorRunStatsV0` derivado del run preparado y limites de la request.
- La decision ajusta `GoalWorkSpecV0.Budget.MaxSubgoals`, anade
  `context_ref kind=autonomous_director_policy ref=autonomous_director_policy:v0`
  y conserva evidencias/recomendaciones de calidad como criterios compactos.
- `BuildStackV0` transporta la politica desde `ConfigV0` y
  `cmd/orquesta-server` inyecta
  `HeuristicAutonomousDirectorPolicyV0`.
- No se toca `orquesta-goal` ni se mete proveedor/modelo/coste en el nucleo.
  Residual TAREA-3: `task_cost_class` ya se deriva en
  `orquesta-runtime-codex-goal`, pero falta seleccionar backend/effort barato
  para doc vs code desde la composicion.

Tests/smoke verdes:

- `GOFLAGS=-buildvcs=false go test -count=1 ./modulos/orquesta-app-director-service -run 'TestStartAppDirectorV0GoalFirstAplicaPoliticaAutonomaV0|TestStartAppDirectorGoalSpecWithAutonomousPolicyV0StatsDistintosDecisionDistinta|TestStartAppDirectorV0GoalFirstLanzaGoalYNoEjecutaLoopLegacy'`
- `GOFLAGS=-buildvcs=false go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestBuildDirectorPortsV0CableaPoliticaAutonomaV0|TestBuildDirectorPortsV0CableaAppGoalLauncher|TestBuildDirectorPortsV0PrecargaCodeContextParaGoalCodigo'`
- `GOFLAGS=-buildvcs=false go test -count=1 ./cmd/orquesta-server -run 'TestBuildStackFromEnvV0CableaBrokerContextoEnMCPHTTPYGoalV0'`
- `GOFLAGS=-buildvcs=false go test -count=1 ./modulos/orquesta-app-director-service ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`
- `GOFLAGS=-buildvcs=false go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPCodebaseQuery|TestMCPArrancarDirectorApp.*GoalFirst|TestToStartAppDirectorRequestV0TransportaDirectorExecutionMode'`
- Smoke Orquesta temporal sin backend Goal: `/api/v0/apps/director` con
  `director_execution_mode=goal_first` devuelve HTTP 500
  `goal_backend_unavailable` y no cae a legacy.

## Actualizacion Codex 2026-07-04 noche 34

TAREA-3 queda cerrada para el routing por coste del backend Codex app-server:

- `orquesta-runtime-codex-goal` deriva `task_cost_class` desde el write-set
  (`doc|code|mixed`) y lo incluye en el prompt/packet. Se amplia para tratar
  carpetas `docs/...` como documentales, no solo ficheros `.md/.markdown`.
- `serverCodexGoalCostRoutingStarterV0` en `cmd/orquesta-server` ya envolvia
  el backend app-server; se deja evidencia con tests de que `doc` fuerza
  `reasoning_effort=low`, mientras `code` y `mixed` conservan el esfuerzo
  configurado.
- El wrapper vuelve a derivar la clase desde el write-set y no confia en una
  clase declarada incoherente.
- No se anaden envs ni campos nuevos: el default barato `low` queda como
  politica de composicion Codex app-server; proveedor alternativo barato
  (Gemini/Claude) queda como decision de producto/config futura, no como core.

Verificacion:

- `GOFLAGS=-buildvcs=false go test -count=1 ./cmd/orquesta-server -run 'TestServerCodexGoalCostRoutingStarterV0BajaSoloDocumentacionALowV0|TestServerCodexGoalTaskCostClassForPacketV0IgnoraDeclaradoIncoherenteV0|TestServerCodexGoalBackendFromEnvV0TmuxNoArrancaAppServerEnConstruccionV0'`
- `GOFLAGS=-buildvcs=false go test -count=1 ./modulos/orquesta-runtime-codex-goal -run 'TestBuildCodexGoalStartPacketV0DerivaTaskCostClassDesdeWriteSetV0'`

## Migración a servidor remoto COMPLETADA (Claude, 2026-07-04 noche)

- Servidor: srv1651826 (berserk@uso.dipgra.cloud), repo en
  /srv/orquesta-self/worktrees/orquesta, rama trabajo/plataforma-agentes
  en f5a3d7a2 (idéntica a local y GitHub).
- Respaldos previos al force-push autorizado: rama
  respaldo-pre-migracion-2026-07-04 + tag homónimo-tag (estado anterior
  1667a411) + stash triaje-claude-2026-07-04 intactos.
- Verificado allí: go build ./... EXIT=0 (Go 1.25.11 en
  /srv/orquesta-self/tools/go/bin), suites goal/estado-vivo/web verdes,
  codex CLI en /usr/local/bin/codex, perfil self-programming en
  /srv/orquesta-self/orquesta-self.env. Disco: 20G libres (80%),
  primera tarea allí: limpiar caches/runtimes viejos.
- GitHub queda como canal de sincronización (el fetch directo del servidor
  a GitHub no tiene credenciales; empujar desde donde se trabaje).
- El equipo local queda APAGADO. Continuación: cola vigente en
  instrucciones del director (TAREA-2 ampliada, 8, 9, 10, wizard U/T/bot,
  smoke OPES real).

## Ciclo remoto autonomo: wizard completado en 4 goals (Claude revisor, 2026-07-05 madrugada)

Orquesta autonoma en srv1651826 (worktree pilot-remoto-1, sandbox
workspace-write, umbral 450k) completo en cadena, con Claude validando e
integrando cada uno fuera del sandbox:

1. afa499ef ayudas en lenguaje llano + glosario generado (seccion 11) —
   incluyo transporte de test in-process sin sockets para el sandbox.
2. 7ef4385c dimensiones universales U1-U12 + motor de exclusion por hechos
   (9.1/10.2), con TestWizardKernelLinuxCExcluyeWebV0.
3. 83241927 capa tecnica T1-T8 activada por hechos + defaults silenciosos
   (10.1/10.3), con test de Active Directory por contexto de empresa.
4. 638148ff bot guia determinista con RAG del catalogo (G4, 12.1-12.5),
   grounding estricto, sesion completable sin LLM.

El wizard del diseno docs/diseno_wizard_programacion_2026-07-04.md queda
implementado salvo el nivel LLM del bot (G5, opt-in con presupuesto, 12.7).
Todo sincronizado servidor=local=GitHub tras cada ciclo. Hallazgo menor
anotado: un placeholder de progreso uso status=blocked+"implementacion
pendiente" y confundio a observe/watchers; el contrato de placeholder
deberia reservar blocked para estados terminales reales.

Siguiente en cola remota: TAREA-9 ola 1 (poda deadcode orquesta-deploy),
TAREA-8 olas de envs, smoke OPES real (058/066/075), G5 opt-in.
## Actualizacion Codex remoto 2026-07-04 noche 35

TAREA-9 ola `orquesta-deploy` + `orquesta-capacity`, goal
`goal-ref-task-autoprogramming-49e10ec01497-g01`.

Hecho:

- Materializado checkpoint temprano en
  `modulos/orquesta-deploy/checkpoint_started.txt`.
- Clasificadas las 185 candidatas del snapshot de
  `docs/auditoria_codigo_deadcode_2026-07-04.txt` para ambos modulos en
  `modulos/orquesta-deploy/docs/deadcode_classification_goal-ref-task-autoprogramming-49e10ec01497-g01.json`.
- Resultado de clasificacion: 9 candidatas categoria (a) ya no existen en el
  arbol actual por podas previas verificadas; 176 quedan como categoria (b)
  porque son contratos/DTOs/adaptadores dry-run cubiertos por docs o tests
  locales, o helpers de esos contratos no cableados desde `cmd/...`.
- No se borran mas simbolos en esta ola: la poda segura ya estaba aplicada en
  el arbol actual y borrar APIs retenidas romperia contratos locales.

Verificacion:

- `git diff --check` -> verde.
- `GOCACHE=/tmp/orquesta-goal-g01-gocache GOTMPDIR=/tmp/orquesta-goal-g01-gotmp GOFLAGS=-buildvcs=false go test -count=1 ./modulos/orquesta-deploy ./modulos/orquesta-capacity` -> verde.
- `GOCACHE=/tmp/orquesta-goal-g01-gocache GOTMPDIR=/tmp/orquesta-goal-g01-gotmp GOFLAGS=-buildvcs=false go build ./...` -> verde.
- `scripts/orquesta_auditoria_codigo.sh --no-sqlite --deadcode-file docs/auditoria_codigo_deadcode_2026-07-04.txt --json-out modulos/orquesta-deploy/docs/auditoria_codigo_goal-ref-task-autoprogramming-49e10ec01497-g01.json` -> `deadcode_candidates=1188`, `helper_duplicate_definitions=288`.

Nota operativa: el primer intento de pruebas/build fallo porque `GOCACHE`
apuntaba a una ruta runtime de solo lectura; se reejecuto con cache temporal
aislada. El primer intento de `go build ./...` con `GOMODCACHE` vacio fallo por
red restringida; el build verde uso el cache de modulos existente y solo
movio `GOCACHE`/`GOTMPDIR`.

## Actualizacion Codex remoto 2026-07-04 noche 36

TAREA-9 ola 2 `orquesta-orchestration-core` + `orquesta-runtime`, goal
`goal-ref-task-autoprogramming-7050ce832266-g01`.

Hecho:

- Checkpoint temprano conservado en
  `modulos/orquesta-orchestration-core/checkpoint_started.txt` y checkpoint del
  agente en `modulos/orquesta-orchestration-core/docs/checkpoint_started_goal-ref-task-autoprogramming-7050ce832266-g01.txt`.
- Clasificadas las 222 candidatas del scope desde
  `docs/auditoria_codigo_deadcode_2026-07-04.txt` en
  `modulos/orquesta-orchestration-core/docs/deadcode_classification_goal-ref-task-autoprogramming-7050ce832266-g01.json`.
- Poda real aplicada a la unica entrada clase `b` verificable:
  `ValidateProcessRuntimeSnapshotV0`. Se elimina solo el wrapper exportado sin
  consumidores; el validador interno `validateProcessRuntimeSnapshotV0` se
  conserva porque lo usan adaptadores del runtime.
- El resto queda clase `c` por contrato, tests, docs o integracion externa
  vigente; no se borra API publica usada por otros modulos aunque el snapshot
  de deadcode la marque unreachable.

Verificacion:

- `rg -n "ValidateProcessRuntimeSnapshotV0" modulos/orquesta-orchestration-core modulos/orquesta-runtime cmd --glob '*.go'`
  -> sin coincidencias en codigo Go vivo.
- `git diff --check` -> verde.
- `GOCACHE=/tmp/orquesta-go-cache-poda-ola2 GOTMPDIR=/tmp go test ./modulos/orquesta-orchestration-core ./modulos/orquesta-runtime` -> verde.

## Noche remota autonoma 2026-07-05: 9 goals, 9 integrados (Claude revisor)

Ciclos 5-9 tras el wizard (1-4, ya registrados): 5) poda ola 1 deploy/capacity
clasificada (7dcfed38); 6) TAREA-8 ola 1 envs pisadas consolidadas con alias
deprecated (ed782213); 7) poda ola 2 orchestration-core/runtime: 221 clase C
legacy con contrato — el deadcode restante ES el subsistema legacy, su
retirada queda gateada a la ventana §9 de 7 nightlies verdes (9886f157);
8) smoke OPES lifecycle real 24/24 fases en verde ejecutado por el revisor
fuera del sandbox — residual comun de BUG-058/066/075 cubierto (d2ef83e7);
9) TAREA-8 ola 2: familias OPES_BRIDGE(47)+CODEX_WAVE(26) al fichero
canonico con envs de alias avisado (3b131124).

Patron operativo consolidado: Orquesta ejecuta en pilot-remoto-1 (sandbox
workspace-write, umbral 450k), Claude valida fuera del sandbox (suites +
smokes con sockets), integra por cherry-pick, sincroniza servidor=local=
GitHub tras cada ciclo. Bug de contrato anotado pendiente: placeholders de
progreso usan status=blocked con redacciones variables; normalizar a un
status/summary canonico de checkpoint para no confundir observadores.

Cola restante gateada: retirada legacy (tras §9), G5 bot LLM (opt-in coste,
decision operador), TAREA-8 olas 3+ (SERVER_IDLE, OPES_REGISTRY...),
enforcement proveedor BUG-079 (frontera runtime externo).

## Ciclos 10-11 y cierre de causas organicas (2026-07-05 madrugada)

- Ciclo 10 (fb170012): TAREA-8 ola 3, server_idle al fichero canonico;
  familias restantes verificadas con focales existentes.
- Ciclo 11 (dcb41f04): TAREA-8.2 guard config_projection_mismatch con
  required_settings en prepare-run/apps-director y evidencia de settings
  verificados. La clase del incidente T7A (config perdida en silencio entre
  shell y daemon) queda cerrada por diseno: fichero canonico + alias
  deprecated avisados + ratchet doble + guard de proyeccion.
- Marcador acumulado del ciclo remoto autonomo: 11 goals lanzados, 11
  completados e integrados (servidor=local=GitHub) con revision externa al
  sandbox en cada uno. Cola restante: gateada (legacy tras ventana §9;
  G5 bot LLM opt-in; enforcement proveedor BUG-079).

## Nightly instalado en el servidor (2026-07-05 madrugada)

- Cron 03:30 en srv1651826 ejecutando scripts/orquesta_smoke_nightly.sh con
  PATH del toolchain y ORQUESTA_CODEX_COMMAND; log en
  /srv/orquesta-self/nightly/cron.log; resultados en
  /home/berserk/.orquesta-nightly/resultado_YYYYMMDD.json.
- Herramienta deadcode instalada en /srv/orquesta-self/tools/go/bin (la fase
  de auditoria del nightly la exige; el fallback a snapshot se rechaza por
  diseno). Ejecucion manual de validacion: status=ok, phase=preflight_ok.
- La ventana §9 de 7 nightlies verdes cuenta desde ahora EN EL SERVIDOR;
  el timer del equipo local queda irrelevante (equipo se apaga).

## Siguiente mision tras G5: test de campo OPES REAL (orden del operador 2026-07-05)

Orden textual: "necesitaria probar orquesta con el conector de OPES y crear
un temario que no tengamos entero, asi vemos si todo funciona bien".

Plan (ejecutar cuando G5 este integrado y no quede nada en la tanda):
1. Inventariar en el OPES real del servidor (opes-api + postgres ya
   corriendo en srv1651826) los programas/temarios existentes y elegir uno
   INCOMPLETO o inexistente como objetivo.
2. Conectar Orquesta al OPES real (bridge/drain con ORQUESTA_OPES_BASE_URL
   canonica apuntando al opes-api local; NADA de fixtures), perfil goal-first
   con umbral 450k y guard required_settings del nuevo TAREA-8.2 para
   garantizar la config proyectada.
3. Dejar que la cadena real recorra las 24 fases (registry -> research ->
   draft -> visual -> question bank -> 7 revisiones -> validate -> assemble
   -> audio/tutor/juegos/html/manual -> finalize_temario_package) con
   observacion por eventos y Claude de revisor.
4. Criterio de exito: temario completo materializado en OPES con settlement
   durable, calidad por tema aceptada, sin reescritura tardia y sin procesos
   residuales. Cualquier fallo se registra como bug de campo con refs y se
   programa el fix por Orquesta.
Esto ejecuta de facto MEJ-101 (OPES real) con decision del operador.

## Resultado del field test OPES real tras G5 (2026-07-05)

Goal `goal-ref-task-autoprogramming-72bb10d53162-g04`, tarea
`task-remote-opes-real-field-test-after-g5-20260705`.

Resultado: blocked por `missing_required_settings`. La sesion del goal solo
tenia `ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux`; no estaban declarados
`ORQUESTA_OPES_BASE_URL`, `ORQUESTA_BASE_URL`,
`ORQUESTA_OPES_TEMPORAL_CONFIRM=1`, `ORQUESTA_OPES_BRIDGE_CONFIRM=1` ni scope
duro por `job_ref` o programa/tema/correlacion. No se inventario OPES real, no
se dreno cola y no se creo ni completo temario para evitar efectos sobre OPES
productivo o colas ajenas.

Artefactos: `docs/runbooks/opes_real_field_test_2026-07-05.md` documenta el
bloqueo gobernado y `scripts/smoke_opes_lifecycle_real.sh --help` queda como
ruta sin efectos con settings requeridas. El resultado durable del goal se
conserva en `cmd/orquesta-server/docs/orquesta_goal_result_goal-ref-task-autoprogramming-72bb10d53162-g04.json`
porque la ruta pedida bajo `scripts/smoke_opes_lifecycle_real.sh/docs/` no es
materializable sin convertir el script existente en directorio.

## Integracion G5 wizard bot LLM (2026-07-05)

Goal `goal-ref-task-autoprogramming-72bb10d53162-g01`, tarea
`task-remote-g5-integrate-20260705`.

Resultado: blocked por pruebas requeridas externas al cambio y worktree vivo con
cambios concurrentes fuera del write-set. Queda integrado el diff G5 dentro del
write-set autorizado: puerto LLM opt-in para el bot del wizard, adaptador
app-server Codex con `wizard_bot.*` canonico, degradacion determinista por
presupuesto/proveedor, tests focales y documentacion de no introducir envs
`ORQUESTA_WIZARD_BOT_*`.

Evidencia: `git diff --check` verde; prueba focal
`GOFLAGS=-buildvcs=false go test -count=1 ./modulos/orquesta-web
./cmd/orquesta-server -run 'Test.*Wizard.*Bot|Test.*WizardBot|Test.*ConfigFile|TestServerConfig'`
verde usando cache Go local offline. La bateria amplia requerida falla en
frentes ajenos a G5, incluyendo `orquesta-app-codex-stack` y tests de proceso
proveedor/ratchet; no se crea commit local para no mezclar esos cambios.

## Ratchet de envs tras canal operador Telegram (2026-07-05)

El checkpoint remoto introduce el adaptador operador Telegram opt-in para que el
operador reciba avisos terminales y pueda enviar comandos al Director. Primer
recorte tras el merge: `bot_link_ref`, `authorized_chat_refs`,
`notification_target_ref` y `require_confirmation` dejan de tener env propia y
quedan solo en `orquesta.config.json`/`effective_config` como
`telegram_operator.*`; se conservan como env solo `enabled` y `token`, por
arranque/secretos. La metrica baja de 518 a 514 y el aumento queda justificado
de forma temporal para que el ratchet no oculte la deuda restante:
`env_vars_orquesta_allow_increase_to=514`.

## Endpoint Telegram no-LLM propio de Orquesta (2026-07-06)

Se completa un avance pequeno del pendiente
`BUG-ORQ-20260705-TELEGRAM-NOLLM-ACCEPTED-INVISIBLE`: `cmd/orquesta-server`
monta `POST /api/v0/operator/telegram/update` cuando
`telegram_operator.enabled=true`. La ruta procesa updates Telegram sin LLM,
normaliza `chat.id` a `telegram:<id>`, valida chat autorizado con
`modulos/orquesta-operator-telegram`, despacha comandos contra los puertos ya
existentes y expone el bloqueo de configuracion como JSON `blocked` con
`missing_fields`.

Pruebas:
- `GOFLAGS=-buildvcs=false go test -count=1 ./cmd/orquesta-server -run 'TestTelegramOperator|TestOperatorDirector|TestOperatorNotificationHermes|TestEnvVarsOrquestaRatchetMEJ106V0'`.
- `scripts/orquesta_metricas_deuda.sh --json` -> `env_vars_orquesta=514`.
- `git diff --check`.

No se declara cerrado el bug completo: falta desplegar en remoto, conectar
poller/webhook/Bot API real y revalidar desde Telegram movil.

## Sender Bot API directo para endpoint Telegram no-LLM (2026-07-06)

Avance pequeno adicional sobre `BUG-ORQ-20260705-TELEGRAM-NOLLM-ACCEPTED-INVISIBLE`:

- `cmd/orquesta-server/telegram_bot_api_sender_v0.go` implementa
  `telegramBotAPISenderV0`, adaptador Bot API directo sobre el puerto
  `SendTelegramMessageV0`.
- `buildServerAppHandlerV0` inyecta ese sender en
  `POST /api/v0/operator/telegram/update` cuando existe
  `telegram_operator.token`.
- No se anaden variables `ORQUESTA_*`; tras el corte Codex 2026-07-09 el token
  vive solo en `telegram_operator.token` dentro de `orquesta.config.json`.
- Los errores publicos del sender no incluyen el token.

Pruebas:

- `GOFLAGS=-buildvcs=false go test -count=1 ./cmd/orquesta-server -run 'TestTelegram(BotAPI|Operator)|TestOperatorNotificationHermes'`.

Estado para Claude: codigo local listo y probado. No esta verificado en
produccion/remoto hasta que se haga pull/build/restart solo de Orquesta en
`srv1651826` y se pruebe un update real desde Telegram.

## Direccion Claude 2026-07-06: sync, redeploy remoto, supervisor y repliegue a local

Acciones ejecutadas por el director sobre el servidor remoto:

1. Limpieza de disco ordenada por el operador: solo cachés regenerables
   (cachés Go de agentes 8G, cachés del runtime antiguo `server-latest` 4.7G
   conservando su `state/` y logs, smokes y workspace de auditoria viejos,
   `/tmp/orquesta-*` de goals integrados). Disco: 17G -> 33G libres.
2. Sincronizacion a tres bandas: el merge local de Codex `173b69e41c`
   (checkpoint remoto G5 + canal operador-Director + Telegram + mitigacion
   tick-input) quedo en servidor = local = GitHub. Suite de validacion en el
   servidor: verde salvo el ratchet MEJ-106, que pasa en el arbol sincronizado
   (la aguja `env_vars_orquesta_allow_increase_to=518` esta en esta bitacora).
3. Redeploy: binario recompilado desde `173b69e41c` como
   `orquesta-server-claude` (backup `.885e76b0.bak`). Incidencia de arranque:
   el estado quedo con dueno root de la tanda del 2026-07-05
   (`codex_receipt_descriptor_file_store: read_failed`); corregido con chown
   a berserk. Servidor arranco `startup_ready` con 10 runs adoptados.
4. Verificacion del supervisor en vivo: el error
   `director_tick_input_build_invalido: field=scheduler_input.payload`
   DESAPARECIO (mitigacion verificada). Emerge la capa siguiente:
   `nucleo_orquestacion_store: events.budget: events_full_history_budget_exceeded`
   (~1 error tick cada 5s sobre el run T137, historial > 10000 eventos).
   Incidencia nueva con plan de fix para Codex:
   `docs/incidencias/incidencia_orquesta_supervisor_events_budget_2026-07-06.md`.
5. Decision del operador: el servidor no tiene cuota de proveedor (ademas del
   401 de auth); "hay que trabajar en local hasta entonces". El servidor
   remoto se detiene limpiamente para no quemar ticks en error; la cola queda
   congelada sin perdida de estado. Al recuperar cuota/auth: reautenticar
   `CODEX_HOME=/srv/orquesta-self/codex-home`, redeployar el fix del
   presupuesto de eventos y drenar la cola.

En paralelo Codex trabaja en local: consolidacion de envs telegram_operator al
fichero canonico (`74a98263c`, metricas 518 -> 512) y endpoint Telegram no-LLM
(en curso, sin commitear al escribir esta entrada). El nightly del servidor
(cron 03:30) sigue activo y no depende del server Orquesta.

## Codex local 2026-07-06: falso verde i18n del wizard

Frente disjunto del supervisor para no pisar a Claude. Se detecto que el wizard
de nueva app ya tenia U1-U12/T1-T8, ayuda, glosario y bot con pruebas focales,
pero el catalogo universal aceptaba textos genericos como si fueran ayuda real.
Esto era un falso verde de i18n: el test comprobaba clave presente, no calidad
minima del texto.

Cierre aplicado:

- `modulos/orquesta-web/nueva_app_wizard_universal_i18n_v0.go`: ayudas y
  rationales es/en reales para las opciones universales y tecnicas.
- `modulos/orquesta-web/nueva_app_wizard_turn_v0_test.go`: ratchet contra
  placeholders conocidos en preguntas, opciones, rationales y defaults.
- `docs/wizard_glosario_generado.md`: regenerado desde el catalogo i18n.
- `docs/inventario_bugs_orquesta_2026-06-30.md`: registrado
  `BUG-ORQ-20260706-WIZARD-I18N-PLACEHOLDER` como cerrado.

Prueba verde:

- `go test -count=1 ./modulos/orquesta-web -run 'TestWizard|TestNuevaApp.*Wizard'`

Subagente Codex reviso el wizard y dejo siguiente brecha separada: MCP wizard
no transporta `justification`, `comprehension_query` ni `glossary_expanded`, y
no existe todavia `orquesta.nueva_app.wizard.bot.v0`. Esa microtarea puede
hacerse en `modulos/orquesta-mcp` sin tocar supervisor/state-file/app-codex-stack.

## Codex local 2026-07-06: paridad MCP del wizard existente

Se cierra la brecha pequena detectada por subagente para el tool
`orquesta.nueva_app.wizard.v0`:

- `modulos/orquesta-mcp/nueva_app_wizard_tool_v0.go` expone y normaliza
  `glossary_expanded`, `wizard_answers[].comprehension_query` y
  `wizard_answers[].justification`.
- `modulos/orquesta-mcp/nueva_app_wizard_tool_v0_test.go` exige esos campos en
  el descriptor y verifica que llegan al puerto MCP normalizados.
- `modulos/orquesta-app-codex-stack/stack_flow_v0_test.go` valida el stack real:
  la justificacion aparece en `wizard.contrasts`, `glossary_expanded` vuelve en
  el turno y una consulta de comprension via MCP produce `glossary_response`.
- `docs/inventario_bugs_orquesta_2026-06-30.md` registra
  `BUG-ORQ-20260706-WIZARD-MCP-CONTRACT-PARITY` como cerrado parcial.

Pruebas verdes:

- `go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPNuevaAppWizard|TestNormalizeMCPNuevaAppWizard|TestNewMCPNuevaAppWizard|TestMCPTransportV0NuevaAppWizard'`
- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestBuildStackV0CableaNuevaAppWizardMCPRico'`

Residual vivo tras este corte: falta panel chat web visible para alternar
formulario/chat. El tool MCP especifico del bot queda abordado en el corte
siguiente.

## Codex local 2026-07-06: tool MCP del bot del wizard

Se cierra el contrato MCP pendiente `orquesta.nueva_app.wizard.bot.v0` sin
meter logica web en MCP:

- `modulos/orquesta-mcp/nueva_app_wizard_bot_tool_v0.go`: contrato, descriptor,
  normalizacion, resultado y transporte opt-in del bot.
- `modulos/orquesta-mcp/nueva_app_wizard_bot_tool_v0_test.go` y
  `mcp_transport_registry_v0_test.go`: descriptor, transporte, publicacion y
  unbound opt-in.
- `modulos/orquesta-app-codex-stack/nueva_app_wizard_bot_mcp_executor_v0.go`:
  adaptador del stack que llama a `NewWebNuevaAppWizardBotReplyWithLLMV0` y
  devuelve `reply` + `session` compactos.
- `modulos/orquesta-app-codex-stack/config_v0.go` y `stack_v0.go`: puerto
  opcional `WizardBotAssistant`; sin proveedor funciona determinista.
- `cmd/orquesta-server/stack.go`: inyecta el asistente LLM opt-in existente
  desde `wizard_bot.*` cuando esta habilitado.

Pruebas verdes:

- `go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPNuevaAppWizardBot|TestMCPTransportV0(NuevaAppWizardBot|ExponeOperaciones)'`
- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestBuildStackV0(CableaNuevaAppWizardBotMCPDeterminista|ExponeBindingsMCPNativos)'`
- `go test -count=1 ./cmd/orquesta-server -run 'TestWizardBot|TestBuildStack|TestServerCodexGoal'`
- Verificacion conjunta: `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-app-codex-stack -run 'Test.*Wizard.*MCP|TestMCPNuevaAppWizardBot|TestMCPTransportV0NuevaAppWizardBot|TestBuildStackV0ExponeBindingsMCPNativos'`

Residual anterior: panel chat web visible. Queda cerrado por Codex en el corte
siguiente `BUG-ORQ-20260706-WIZARD-WEB-CHAT-PANEL`.

## Codex local 2026-07-06: panel chat web del bot del wizard

Se cierra la brecha UI dejada tras el tool MCP del bot. El objetivo fue exponer
el mismo bot conversacional en `/nueva-app` sin meter logica de dominio en el
gateway ni duplicar el motor:

- `modulos/orquesta-web/nueva_app_wizard_bot_endpoint_v0.go`: endpoint HTTP
  fino para `POST /api/v0/apps/intake/wizard-bot`, devuelve `reply` y
  `session`, funciona sin proveedor LLM y rechaza metodo/content-type invalido.
- `modulos/orquesta-web/nueva_app_html_render_v0.go`: panel `data-wizard-bot`,
  log accesible, input, envio por fetch, conservacion de `guidedSession`,
  aplicacion del formulario y render de `turn_result`.
- `modulos/orquesta-web/nueva_app_i18n_*`: textos es/en y ayuda del boton sin
  placeholders ni tooltips vacios.
- `modulos/orquesta-http-gateway`: ruta exacta
  `/api/v0/apps/intake/wizard-bot`, manifiesto y guard de mutabilidad como
  lectura aunque cuelgue bajo `/api/v0/apps/`.
- `modulos/orquesta-app-gateway` y `modulos/orquesta-app-codex-stack`: cableado
  del asistente LLM opcional existente hasta el endpoint HTTP real.

Pruebas verdes:

- `go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway ./modulos/orquesta-app-codex-stack -run 'TestNuevaAppWizardBotHTTPHandler|TestNuevaAppHTMLHandlerV0GET|TestWizardBot|TestNuevaAppHTMLHelpKeysV0CubrenClavesUsadasEnPlantilla|TestNewAppGatewayMux|TestPublicRouteMutability|TestPublicRouteManifest|TestGatewayRouteRegistrations|TestNewHTTPHandlerV0ExponeNuevaApp(IntakeGuidedTurn|WizardBot)|TestNewHTTPHandlerV0InyectaNuevaAppIntakeAssistant|TestNewHTTPHandlerV0PropagaFallback|TestBuildStackV0(CableaNuevaAppWizardBotMCPDeterminista|ExponeNuevaAppWizardBotHTTP|ExponeBindingsMCPNativos)'`
- `git diff --check`

## Codex local 2026-07-06: rutas durables de Codex Goal sobre write-set fichero

Frente disjunto de Telegram/supervisor. Se revisaron `BUG-ORQ-20260705-195` y
`BUG-ORQ-20260705-198`: el prompt de Codex Goal podia pedir
`<fichero>.go/docs/orquesta_goal_result_*.json` o
`<script>.sh/docs/orquesta_goal_result_*.json` porque solo detectaba Markdown
como write-set fichero.

Cierre aplicado:

- `modulos/orquesta-runtime-codex-goal/packet_v0.go`: nuevo helper
  `codexGoalWriteScopeLooksLikeFileV0`; `codexGoalResultFilePathV0` salta
  scopes `.go` y `.sh` igual que ya saltaba `.md/.markdown`.
- Si hay un siguiente scope directorio autorizado, el resultado durable se pide
  ahi (`directorio/docs/orquesta_goal_result_<goal>.json`).
- Si todos los scopes son ficheros, no se pide una ruta durable imposible; se
  conserva el cierre obligatorio por marcador `ORQUESTA_GOAL_RESULT_V0`.
- `docs/inventario_bugs_orquesta_2026-06-30.md`: `BUG-ORQ-20260705-195` y
  `BUG-ORQ-20260705-198` pasan a cerrado en codigo.

Prueba verde:

- `go test -count=1 ./modulos/orquesta-runtime-codex-goal`

## Codex local 2026-07-06: parking de runs sobredimensionados por presupuesto de eventos

Frente disjunto de Telegram. Se cierra en codigo
`BUG-ORQ-20260706-SUPERVISOR-EVENTS-BUDGET-PARKING`, el fix 3 pendiente de
`docs/incidencias/incidencia_orquesta_supervisor_events_budget_2026-07-06.md`.

Lectura tecnica:

- El coordinador ya podia continuar tras un error de drain si
  `ContinueOnDrainError=true`, pero la preparacion previa de
  `StackV0.RunGlobalTickV0` podia devolver
  `events_full_history_budget_exceeded` antes de entrar al coordinador.
- Si se limitaba el cambio a poner la cola en `stopped`, el reconciliador de
  runs parados podia reactivar el candidato en el tick siguiente. Por eso el
  parking ahora tambien completa `RunControl` como `stopped` sin evidencia de
  auto-resume.

Cierre aplicado:

- `modulos/orquesta-app-codex-stack/operational_plan_state_recoverable_error_v0.go`:
  clasificador tipado del error de presupuesto de eventos, diagnostico
  `run_events_budget_exceeded`, helper de parking de cola y run-control.
- `modulos/orquesta-app-codex-stack/run_coordinator_v0.go`: el drain convierte
  ese error en `Outcome=run_oversized`, `QueueStatus=stopped`, evidencia durable
  y sin error de tick si el run-control pudo sellarse.
- `modulos/orquesta-app-codex-stack/run_coordinator_reconcile_v0.go`,
  `run_coordinator_running_stale_reconcile_v0.go` y
  `run_coordinator_domain_recovery_v0.go`: las rutas de preparacion con
  candidato concreto aparcan y continuan en lugar de tumbar el tick global.
- `modulos/orquesta-app-codex-stack/run_coordinator_flow_v0_test.go`: tests de
  continuidad de cola y de no reactivacion posterior.
- `docs/incidencias/incidencia_orquesta_supervisor_events_budget_2026-07-06.md`
  y `docs/inventario_bugs_orquesta_2026-06-30.md`: cierre documentado y
  residual de redeploy/verificacion viva.

Pruebas verdes:

- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestCodexStackV0RunGlobalTickAparcaRunSobredimensionadoYContinuaColaV0|TestCodexStackV0RunSobredimensionadoAparcadoNoSeReanudaEnPreparacionV0'`
- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestCodexStackV0RunGlobalTick|TestCodexStackV0RunSobredimensionado|TestCodexStackV0RunGlobalSupervisor|TestCoordinate|TestCodexSupervisorRuntimeStateFromGlobalSupervisor'`
- `go test -count=1 ./modulos/orquesta-run-coordinator`

Durante la verificacion transversal, `go test -count=1 ./...` destapo dos
fallos ajenos al parking pero pequenos y disjuntos:

- `BUG-ORQ-20260706-MCP-WIZARD-BOT-SCHEMA-STALE`: el bot nuevo del wizard no
  estaba en el registro canonico de DTOs MCP y el test de schema no entendia
  arrays `[...]`.
- `BUG-ORQ-20260706-ENV-RATCHET-ROOT-DIVERGENCE`: el test raiz de ratchet de
  entorno no aceptaba la excepcion documentada que ya acepta el test del
  servidor.

Ambos quedan cerrados en el mismo corte para recuperar la suite completa sin
tocar Telegram:

- `modulos/orquesta-mcp/mcp_transport_tool_input_schema_v0.go`
- `modulos/orquesta-mcp/mcp_transport_tool_input_schema_v0_test.go`
- `env_vars_budget_test.go`

Pruebas verdes adicionales:

- `go test -count=1 ./modulos/orquesta-mcp ./cmd/orquesta-server -run 'TestMCPTransportToolInputSchemaV0CubreToolsRegistrados|TestMCPRealTransportV0InputSchemaSaleDeDTOCanonico'`
- `go test -count=1 . -run TestEnvVarsBudgetMEJ106V0`

Residual para Claude/remoto: desplegar cuando haya cuota/auth y verificar que
`supervisor_error_ticks` no crece, T137 queda aparcado si supera presupuesto
real y la cola avanza al siguiente candidato.

## D1 accepted legacy nunca invisible (2026-07-06)

Se cierra el tramo de codigo de
`BUG-ORQ-20260705-TELEGRAM-NOLLM-ACCEPTED-INVISIBLE` que podia dejar un
`prepare-run` aceptado fuera de `autoprogramming/status`.

Lectura tecnica para Claude:

- La ruta legacy aceptaba el request
  `request-ref-remoto-telegram-nollm-runtime-20260705-001`, persistia
  `workflow_task_refs` y encolaba el run con `reason=autoprogramming_prepare_run`.
- La proyeccion publica de `autoprogramming/status` no pasaba `run_ref` hasta
  `run_queue.priority`.
- `orquesta-run-file` podia aplicar `limit` antes de filtrar el run exacto, asi
  que una consulta con `run_ref` y ventana pequena podia no ver el run aceptado.
- El resultado era un falso positivo: `accepted=true` sin visibilidad durable en
  cola/status.

Cierre aplicado:

- `modulos/orquesta-run-queue/contracts_v0.go`: `RunQueueReadRequestV0.RunRef`
  y `RunSchedulingCandidateV0.Reason`.
- `modulos/orquesta-run-memory/queue_v0.go` y
  `modulos/orquesta-run-file/queue_v0.go`: filtro exacto por `RunRef` antes de
  ranking/limite, y conservacion de `Reason`.
- `modulos/orquesta-mcp/run_queue_priority_tool_v0.go`: transporte de
  `run_ref` y salida compacta con `reason`.
- `modulos/orquesta-mcp/autoprogramming_status_tool_v0.go`: `status(run_ref)`
  consulta la cola por ese run exacto.
- `modulos/orquesta-mcp/autoprogramming_status_queued_not_dispatched_v0.go`:
  diagnostico especifico `autoprogramming_prepare_run_pending_dispatch` cuando
  el run aceptado por `prepare-run` sigue en cola.
- `modulos/orquesta-app-codex-stack/autoprogramming_prepare_run_queue_v0.go`:
  readback inmediato tras encolar; si no se ve el run, devuelve error
  `autoprogramming_prepare_run_visibility_error` en vez de aceptar invisible.

Pruebas verdes:

- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestCodexStackAutoprogrammingPrepareRunAPIV0AcceptedLegacyVisibleEnStatusV0|TestCodexStackAutoprogrammingPrepareRunAPIV0NoAceptaSiColaNoProyectaRunV0|TestCodexStackAutoprogrammingPrepareRunAPIV0PreparaRunYSupervisorArranca'`
- `go test -count=1 ./modulos/orquesta-run-memory ./modulos/orquesta-run-file ./modulos/orquesta-mcp -run 'TestRunMemoryStoreListSchedulingCandidatesFiltraRunRefAntesDeLimitV0|TestRunFileStoreListSchedulingCandidatesFiltraRunRefAntesDeLimitV0|TestMCPRunQueuePriorityExecutorV0RankTransportaRunRefExactoV0|TestMCPRunQueuePriorityDescriptorV0DeclaraEvidenciaDeCandidatos|TestMCPAutoprogrammingStatusExecutorV0DiagnosticaQueuedNotDispatchedV0'`
- `go test -count=1 ./modulos/orquesta-run-queue ./modulos/orquesta-run-memory ./modulos/orquesta-run-file ./modulos/orquesta-mcp ./modulos/orquesta-app-codex-stack`
- `git diff --check`

Residual: queda desplegar en remoto y probar desde Telegram real. Si el run ya
visible intenta ejecutar agente, el servidor aun puede topar con
`BUG-ORQ-20260705-CODEX-HOME-TOKEN-INVALIDADO`, que es un bloqueo de proveedor
separado.

## D2 contrato unico de presupuestos (2026-07-06)

Se cierra en codigo local `BUG-ORQ-20260706-BUDGET-CONTRACT-DESALINEADO`,
derivado de la auditoria P1 de Claude: las capas tenian presupuestos privados
para eventos y payload (`250`, `1000`, `10000`, `20000`, `256 KiB`) sin contrato
comun ni test de coherencia.

Commit: este mismo corte con mensaje `budget: centralizar presupuestos de
orquestacion`.

Cierre aplicado:

- `modulos/orquesta-orchestration-budget`: paquete neutral nuevo, sin imports,
  con constantes canonicas de pagina de lectura, pagina store, lectura total,
  maximo store, payload de evento, snapshot de tick y payload scheduler.
- `modulos/orquesta-state-file`: consume el contrato para pagina/maximo de
  eventos y payload de evento, conservando overrides de `ConfigV0`.
- `modulos/orquesta-orchestration-core` y
  `modulos/orquesta-app-director-service`: lectores paginados usan pagina
  canonica `250` y corte total `10000`.
- `modulos/orquesta-director-scheduler`: el limite de payload del tick sale de
  `SchedulerTickPayloadMaxBytesV0` y sigue en 256 KiB.
- `modulos/orquesta-director-tick-input`: el gate de compactacion del snapshot
  de progreso usa el presupuesto canonico de snapshot, menor que el payload del
  scheduler.
- `modulos/orquesta-director-scheduler/docs/decisiones.md`: se corrige el dato
  stale que seguia diciendo 16 KiB.
- `docs/inventario_bugs_orquesta_2026-06-30.md`: se registra la clase de bug y
  el cierre local.

Pruebas verdes:

- `git diff --check`
- `go test -count=1 ./modulos/orquesta-orchestration-budget ./modulos/orquesta-state-file ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-director-service ./modulos/orquesta-director-scheduler ./modulos/orquesta-director-tick-input`
- `go test -count=1 ./modulos/orquesta-director-cycle ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`

Residual: pendiente despliegue/verificacion remota. D2 no cambia valores ni
arregla D5; solo evita que vuelvan a divergir presupuestos entre capas.

## TAREA-D3 completada por Claude sobre WIP de Codex (2026-07-06)

Codex dejo la D3 (bateria requerida derivada de dependencias del write-set) a
mitad al agotar cuota: 4 ficheros nuevos sin compilar. Claude la termino:

- Corregido el corte de edicion (campo `Detail` de `GoalWorkIssueV0`).
- Dos correcciones de diseno sobre el WIP, coherentes con la auditoria P2:
  1. El resolver ya NO cachea errores de `go list` (un fallo transitorio
     quedaba pegado hasta reiniciar el servidor).
  2. Fail-open gobernado: si la resolucion falla, el goal se lanza con sus
     tests declarados y evidencia
     `evidence-ref-goal-required-test-dependency-resolution-unavailable`,
     en vez de bloquear todos los lanzamientos (el nightly completo sigue
     siendo la red dura). Test del contrato actualizado.
- Suites completas verdes: `./modulos/orquesta-app-codex-stack` y
  `./cmd/orquesta-server` (los dos flujos que el fail-closed rompia pasan).

Con esto D1, D2 y D3 de la cola del 2026-07-06 quedan cerradas. Siguientes
para Codex: D4 (emisores sin dedupe), D5 (gate comun de compactacion), D6
(reason codes de placeholders), D7 (runbook de arranque).

## TAREAS D5 y D7 completadas por Claude (2026-07-06)

- D5 (`f83014450`): gate comun de compactacion en las lanes delivery,
  phase-artifact y review-gate — solo recortan snapshot si el input supera
  `SchedulerTickSnapshotBudgetBytesV0`. Tests de compactacion actualizados a
  fixtures sobre presupuesto + tests espejo de conservacion por lane. Suites
  verdes: tick-input, orchestration-core, director-cycle, director-scheduler,
  app-codex-stack, cmd/orquesta-server (regla P5 aplicada).
- D7 (`cf6632f71`): `scripts/orquesta_server_ctl.sh` — arranque/parada/status
  con usuario de servicio verificado (rechaza root: `wrong_service_user`),
  deteccion previa de estado con dueno equivocado
  (`state_permission_denied`), perfil de envs canonico y SIGINT cooperativo
  verificado.

Quedan para Codex: D4 (auditoria de emisores sin dedupe semantico) y D6
(reason codes de placeholders — ver nota anadida a la tarea con el analisis
de los 3 puntos de fix). Estado cola: D1-D3, D5, D7 cerradas.

## TAREA-D6 (nucleo) cerrada por Claude (2026-07-06)

`32be9b9dd`: el normalizador de resultados del appserver clasifica los
placeholders por FORMA (blocked sin tests, sin checklist completada y sin
missing_refs) y proyecta `IssueCode=goal_result_placeholder_in_progress` +
evidencia estable. Watchers y status pueden discriminar sin leer texto libre.
Tests con las redacciones reales que causaron las falsas alarmas. Residual
menor de D6 para Codex: instruir el reason_code en el paquete goal-first y
usar el IssueCode en las proyecciones de observe (el dato ya viaja).
`9a031eb96`: el runbook D7 usa el shutdown comun del repo (API -> SIGINT ->
SIGTERM + limpieza tmux); los dos guards de scripts pasan. Suites verdes:
runtime-codex-appserver, cmd/orquesta-server completa, app-codex-stack.

Cola D1-D7: solo queda D4 (auditoria de emisores) integra para Codex.

## TAREA-D4 completada por Claude (2026-07-06) — cola D1-D7 CERRADA

`0670415ba`: dedupe causal en origen en la supervision de progreso
(`progress_candidate_provider.go`):

- Refs semanticas estables tambien para stopped/loop_detected (antes
  excluidos del hash estable: era el camino que inflo T137 a 2669 eventos,
  una pareja assessment+question nueva por ReportID en cada tick).
- Salto del candidato cuando la misma pareja sigue pendiente en la proyeccion
  del run: pregunta sin responder o stop ya pedido/confirmado. Al responderse
  la pregunta la supervision vuelve a ser elegible.
- Mecanismo check-before-emit (mismo patron que el drain ec07bf300): las
  claves de comando/evento siguen siendo por-reporte (los payloads llevan
  refs volatiles y una clave estable provocaria conflicto de payload en el
  store idempotente); lo que se evita es emitir el duplicado.

Auditoria del resto del inventario D4 (justificaciones):
- `progress_lease_bridge_v0.go` (refs por ReportID): no es fuente de
  inflacion observada (T137: AgentLeaseExpirations=[]); el ciclo de vida de
  una lease (expira -> stop) limita la repeticion. Vigilar si aparece
  inflacion de AgentLeaseExpired en runs atascados.
- Emisores de replan (app-director-service, core-replanner,
  assessment_replan_source): claves derivadas de refs causales de
  review/delivery (one-shot por ciclo de review); T137 tenia cero
  ReplanDecisionRecorded. Sin accion.

Bateria: orchestration-core, app-director-service, app-codex-stack y
cmd/orquesta-server completas en verde. Con esto la cola
docs/instrucciones_director_codex_2026-07-06.md queda CERRADA (D1-D7).

## Codex local 2026-07-07: BUG-193 broker de codigo no inyectado en goal

Se cierra en codigo `BUG-ORQ-20260705-193`. El prompt de Codex Goal trataba
`orquesta.codebase.query.v0` como obligacion dura para write-sets de codigo,
pero algunos goals no reciben el toolbelt MCP local ni recursos/templates del
broker. Eso podia bloquear trabajos validos o forzar una excepcion manual aun
cuando el servidor Orquesta publicaba HTTP o bastaba una lectura acotada.

Cierre aplicado:

- `modulos/orquesta-runtime-codex-goal/packet_v0.go`: el analizador de codigo
  queda formulado por orden de preferencia: toolbelt MCP
  `orquesta.codebase.query.v0`, HTTP `POST /api/v0/codebase/query`, y fallback
  `rg`/`sed` acotado con evidencia `codebase_broker_unavailable` si el broker
  no esta inyectado en ese goal.
- `modulos/orquesta-runtime-codex-goal/packet_v0_test.go`: nuevo test de
  degradacion sin broker inyectado y ajuste de expectativas del contrato.
- `docs/inventario_bugs_orquesta_2026-06-30.md`: `BUG-ORQ-20260705-193` pasa
  a cerrado en codigo; `BUG-ORQ-20260705-SUPERVISOR-SCHEDULER-PAYLOAD` queda
  reconciliado como cerrado porque su incidencia ya tenia cierre remoto y el
  fallo posterior pertenece al presupuesto de eventos.

Pruebas verdes:

- `go test -count=1 ./modulos/orquesta-runtime-codex-goal`
- `git diff --check`

## Codex local 2026-07-07: residual D6 reason_code en placeholders

Se cierra el residual menor que habia quedado tras `32be9b9dd`: el runtime ya
clasificaba placeholders por forma, pero el contrato de prompt no exigia
`reason_code`, el app-server no proyectaba `checkpoint_started` cuando solo
existia el checkpoint temprano sin resultado final, y `director_stats` podia
meter `LastResult.Summary` dentro de `issue_codes`.

Cierre aplicado:

- `modulos/orquesta-runtime-codex-goal/packet_v0.go`: el JSON obligatorio del
  goal incluye `reason_code`; los placeholders/intermedios deben usar codigos
  de catalogo y dejar `summary` solo como texto informativo.
- `modulos/orquesta-runtime-codex-appserver`: el resultado durable acepta
  `reason_code`, lo proyecta como `IssueCode`, y si encuentra
  `checkpoint_started.txt` validado por `goal_ref`/`external_goal_ref` en un
  goal running, observa `IssueCode=checkpoint_started` sin depender de
  substrings del summary.
- `modulos/orquesta-mcp/director_stats_tool_v0.go`: `issue_codes` sale solo de
  `GoalWorkIssue.Code`, no de `LastResult.Summary`; los tests historicos de
  timeout quedan migrados a `Issues.Code`.

Pruebas verdes:

- `go test -count=1 ./modulos/orquesta-runtime-codex-goal ./modulos/orquesta-runtime-codex-appserver ./modulos/orquesta-mcp ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`
- `git diff --check`

Reverificacion Codex 2026-07-08:

- D5 no se reprograma: el HEAD actual ya aplica `tickInputWithinSnapshotBudgetV0`
  a progress, delivery, phase-artifact y review-gate, y mantiene tests espejo
  de snapshot pequeno completo por lane.
- D6 no se reprograma: el HEAD actual exige `reason_code` en el contrato
  goal-first, normaliza placeholders por forma, proyecta `checkpoint_started`
  desde checkpoint temprano validado y evita derivar `issue_codes` desde
  `summary`.
- Refuerzo posterior: `TestCodexGoalObserverV0CheckpointStartedNoDerivaIssueDesdeSummaryV0`
  fija que `IssueCode=checkpoint_started` viaja como codigo estructurado aunque
  el `summary` conserve una redaccion libre historica.
- Tests ejecutados:
  `go test -count=1 ./modulos/orquesta-director-tick-input -run 'TestBuildDirectorSchedulerTickInputV0(CarrilProgressPequeno|CarrilReviewGatePequeno|CarrilDeliveryPequeno|CarrilPhaseArtifactPequeno|CompactaCarrilProgress|CompactaCarrilReviewGate|FiltraConfirmedStoppedAgents)'`
  y
  `go test -count=1 ./modulos/orquesta-runtime-codex-goal ./modulos/orquesta-runtime-codex-appserver ./modulos/orquesta-mcp -run 'Test(GoalWorkSpecV0|CodexGoal|ServerCodexAppServerGoalBackendV0ObservaCheckpointStartedComoReasonCodeV0|MergeCodexAppServerGoalResultV0|DirectorStats|AutoprogrammingStatus|ObserveAppDirectorGoal)'`.
  Verificacion adicional:
  `go test -count=1 ./modulos/orquesta-runtime-codex-goal`.

## Codex local 2026-07-07: BUG-066 OPES done/settled sin reescritura tardia

Se reduce `BUG-ORQ-20260701-066` en el adaptador OPES local. No se cierra el
bug completo porque sigue pendiente el smoke temporal/residente que demuestre
reconciliacion automatica tras cortes externos/manuales del backend goal-first.

Hallazgo corregido:

- La clave idempotente de trabajos causales OPES no incluia `receipt_ref`,
  aunque el contrato local exige source job, artifact, receipt y followup/rework.
  Una actualizacion antigua de `update_topic_registry` podia bloquear una
  posterior del mismo artefacto con evidencia suficiente de settlement.
- `pending_refs/rework_refs` stale podian reabrir `assemble_topic` o
  `review_director_consolidation` aunque el record trajera
  `settlement_status=settled_text|settled_final` valido.

Cierre aplicado:

- `modulos/orquesta-opes-director/job_requests_v0.go`: la idempotencia causal
  incluye `receipt_ref`; `followupRefsForRecordV0` ignora pendientes declarados
  solo si existe terminal settlement valido, pero conserva siempre blockers
  calculados de QA/evidencia/lifecycle.
- `modulos/orquesta-opes-director/topic_registry_settlement_v0.go`: nuevo
  contrato de terminal settlement explicito. `settled_text` exige texto/QA
  publicable; `settled_final` exige `CompleteJob=true` y manifest de cierre
  completo. Ningun terminal tapa blockers estructurados.
- `modulos/orquesta-opes-director/topic_registry_v0.go`: `settled_text` se
  proyecta como `texto_asentado_pendiente_derivados` y
  `operational_status=waiting`; `settled_final` como
  `paquete_final_local_verificable` y `operational_status=complete`.
- `modulos/orquesta-opes-director/producer_v0_test.go`: cobertura de cambio de
  receipt en el mismo artefacto, pendientes stale en texto asentado, pendientes
  stale en paquete final y alias `settled*`.

Pruebas verdes:

- `go test -count=1 ./modulos/orquesta-opes-director`

Pendiente explicito para Claude:

- Mantener `BUG-066` abierto para el tramo real/residente:
  `/api/v0/external-work/observe`, corte externo/manual del backend,
  reconciliacion automatica de terminalidad y ausencia de goal residual.

## Codex local 2026-07-07: BUG-194 OPES required_settings transportado al goal

Se reduce `BUG-ORQ-20260705-194` sin tocar OPES productivo.

Hallazgo:

- El bridge OPES ya tenia guardas de ejecucion real, pero `effective_config` no
  proyectaba todas las confirmaciones que el runbook exigia para un field test
  real (`TEMPORAL_CONFIRM`, `BRIDGE_CONFIRM`, `ENABLED`, `DRY_RUN`).
- El compilador `external-work -> GoalWorkSpec` podia omitir por presupuesto
  campos de contrato como `required_settings`, confirmaciones, `limit=1` y
  scope duro; entonces el agente remoto solo podia bloquear por
  `missing_required_settings` sin ver un contrato durable completo.

Cierre aplicado:

- `cmd/orquesta-server/opes_bridge_config.go`: `effective_config` publica las
  guardas OPES bridge reales junto a URL, scope y limites, sin exponer URL cruda.
- `modulos/orquesta-external-work-run/goal_spec_v0.go`: los input fields de
  required settings/scope OPES pasan a prioridad maxima para quedar inlineados
  antes que ruido operativo.
- `modulos/orquesta-external-work-run/goal_spec_opes_v0.go`: detector OPES
  separado para mantener `goal_spec_v0.go` bajo el ratchet T90.
- `goal_spec_v0.go`: criterio de aceptacion OPES real exige settings, scope
  duro y `limit=1`; si falta algo, bloquear con
  `reason_code=missing_required_settings` y no tocar colas ni OPES productivo.

Pruebas verdes:

- `go test -count=1 ./modulos/orquesta-external-work-run ./cmd/orquesta-server`

Comprobacion remota:

- `srv1651826:/srv/orquesta-self/worktrees/pilot-remoto-1` esta en
  `0188739c`, 24 commits por detras de `origin/trabajo/plataforma-agentes`, con
  remote local `/tmp/orquesta-self.bundle`; no hay cierre mas avanzado de este
  bug en el servidor. Pendiente: redeploy/sync remoto antes del nuevo field
  test real.

## Codex local 2026-07-08: Telegram no-LLM requiere config canonica en ctl

Hallazgo:

- D1 `accepted invisible` ya estaba desplegado en remoto y las pruebas focales
  de visibilidad/cola/MCP/Telegram estaban verdes, pero la prueba real
  `POST /api/v0/operator/telegram/update` devolvio `404`.
- Causa inmediata: el servidor estaba arrancado sin `orquesta.config.json`; el
  endpoint Telegram es opt-in y solo se monta cuando
  `telegram_operator.enabled=true` llega a `cmd/orquesta-server`.
- El script operativo `scripts/orquesta_server_ctl.sh` no pasaba `--config` al
  binario, asi que una config canonica fuera del default podia quedar ignorada
  en arranques gestionados por `ctl`.

Cierre local aplicado:

- `scripts/orquesta_server_ctl.sh` acepta `ORQUESTA_CTL_CONFIG` y, si no se
  define, usa `$ORQUESTA_CTL_WORKDIR/orquesta.config.json` cuando existe.
- `preflight` falla con `config_missing` si se declara una config no legible.
- `start` ejecuta `orquesta-server run --config <config>` cuando corresponde y
  conserva el comportamiento anterior si no hay config.
- `scripts/test_orquesta_server_ctl.sh` prueba con binario fake que no se pasa
  `--config` sin fichero y que si se pasa con auto-config.

Pruebas verdes:

- `bash scripts/test_orquesta_server_ctl.sh`
- `go test -count=1 ./cmd/orquesta-server -run 'TestServerConfigFromEnvWithProjectConfigPathV0CargaFicheroExplicito|TestServerDaemonRunArgsV0UsaSnapshotDeConfigExplicita|TestTelegram(BotAPI|Operator)|TestOperatorNotificationHermes'`
- `git diff --check`

Residual para Claude:

- No se ha creado config real con token Telegram ni chats autorizados. Ese dato
  no debe inventarse en codigo ni docs.
- Falta deploy/sync del cambio de `ctl`, instalar config canonica, configurar
  webhook o poller y validar desde el movil.
- Las medidas de ahorro de tokens/programacion minima quedan opt-in y sujetas a
  A/B empirico en golden tasks: si reducen tokens/diff pero aumentan fallos,
  rework, tiempo o tests rotos, no se activan como default amplio.

## Codex local 2026-07-08: goal terminal con artefacto declarado y ausente

Se cierra en codigo local la incidencia
`docs/incidencias/incidencia_orquesta_remoto_automejora_goal_first_receipt_desreconciliado_2026-07-01.md`
para la clase observada en APG-004: `GoalWorkState` terminal `complete` conserva
`ArtifactPaths` o receipt en `LastResult`, pero el fichero ya no existe en el
write-set.

Cierre aplicado:

- `modulos/orquesta-app-codex-stack` detecta rutas declaradas por resultados
  terminales que faltan en disco y emite
  `terminal_artifact_missing_after_goal_complete` con evidencia por path.
- `modulos/orquesta-mcp` proyecta esa senal en `observe_goal`,
  `director/stats`, `autoprogramming/status` y `efficiency_summary`.
- La accion recomendada es `replan`, no `repair_receipt`: el recibo puede estar
  formalmente completo, pero falta recuperar o rehacer el artefacto.

Pruebas verdes:

- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestStackGoalMaterializedRefsSourceV0(DetectaArtifactPathDeclaradoPeroBorrado|DetectaArtifactPathsOmitidos|DetectaRequiredTestEvidence)'`
- `go test -count=1 ./modulos/orquesta-mcp -run 'Test(EnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0QAFailedPublicTextRunningNoEspera|MCPDirectorStatsToolExecutorV0GoalFirstProyectaArtifactPathDeclaradoPeroBorrado|MCPAutoprogrammingStatusExecutorV0ArtifactPathDeclaradoPeroBorradoPideReplan)'`
- `git diff --check`

Residual: falta deploy/sync y repeticion remota de automejora residente cuando
haya proveedor/cuota.

## Codex local 2026-07-08: shutdown parcial no deja goal_action stale

Hallazgo:

- El subagente de revision de `BUG-ORQ-20260701-065` /
  `BUG-ORQ-20260704-165` detecto un borde no cubierto: con dos backends goal,
  `cleanup_goal_backends=true` podia limpiar uno y dejar otro vivo.
- El resultado re-leia `active_work`, pero `goal_actions` conservaba
  `cleanup_requested` para todos los works iniciales; el work ya limpio podia
  quedar como accion bloqueante stale para clientes que bloquean cualquier
  accion distinta de `cleanup_completed`.

Cierre local aplicado:

- `orquesta-server-shutdown` calcula los works completados tras cleanup por
  identidad estable `kind/run_ref/work_ref/external_work_ref`.
- Si un work desaparece tras el cleanup y el cleaner reporta al menos un
  limpiado, se publica `cleanup_completed` para esa identidad aunque otros
  backends sigan vivos.
- `compactServerShutdownGoalActionsV0` suprime acciones previas no terminales
  de una identidad que ya tiene `cleanup_completed`, para no publicar falsos
  bloqueos.
- El helper nuevo vive en fichero propio para respetar el ratchet de tamano del
  modulo.

Pruebas verdes:

- `go test -count=1 ./modulos/orquesta-server-shutdown`
- `go test -count=1 ./cmd/orquesta-server -run 'Test(RequestServerShutdownV0CoordinaDosGoalsActivosHastaGoalActionsResueltas|RequestServerShutdownV0ReadyNoSaltaGoalActionsSinActiveWork|RequestServerShutdownV0PostColgadoConsultaStatusAccionable|WaitServerShutdownReadyV0RepostColgadoRespetaDeadlineYDevuelveStatusAccionable)'`
- `go test -count=1 ./modulos/orquesta-server -run 'Test(ShutdownProjectionFromHTTPV0ReadyConGoalActionsQuedaStopPending|ServerPublicStatusV0ExponeShutdownGoalActions|StatusTracker|ServerPublicStatus)'`

Residual: esto cierra el borde local de cleanup parcial; no cierra por si solo
la observabilidad/control largo de `BUG-165` con proveedor real. Falta
deploy/sync y smoke real residente amplio.

## Codex local 2026-07-08: auditoria de programacion minima y tokens

Se documenta en
`docs/auditoria_programacion_minima_tokens_2026-07-08.md` la busqueda externa y
la auditoria local sobre prompts/skills/reglas para reducir tokens y evitar
codigo innecesario.

Conclusion operativa:

- Orquesta ya tiene `skills/orquesta-programacion-minima/SKILL.md`, reglas de
  comunicacion compacta, write-set estrecho, contexto acotado, golden evals y
  benchmark de compresion documental.
- No conviene inflar `AGENTS.md`: hay evidencia externa de que context files
  grandes pueden aumentar pasos, lecturas, escrituras y coste.
- La mejora pendiente no es mas doctrina, sino medicion: extender golden evals
  con tokens reales, diff stats, rework y score por `run_ref/goal_ref`, y solo
  activar defaults si el A/B no degrada calidad.

Avance de codigo: `scripts/orquesta_golden_evals.sh` acepta ahora
`result.json.metrics`, agrega `summary.metrics` y conserva `tasks[].metrics`.
`docs/runbooks/orquesta_golden_evals_2026-07-04.md` documenta el contrato.
Avance posterior Codex 2026-07-08: `scripts/orquesta_golden_metrics_launcher.sh`
queda como wrapper A/B opt-in para launchers reales: mide tiempo, exit code,
diff Git si recibe `ORQUESTA_GOLDEN_TASK_WORKTREE` y normaliza tokens si el
proveedor los publica. Test:
`bash scripts/test_orquesta_golden_metrics_launcher.sh`.

Avance Codex 2026-07-09: `scripts/orquesta_golden_ab_launcher.sh` ejecuta brazo
baseline y variante por tarea, conserva `arms/baseline` y `arms/variant`,
publica deltas de tokens/tiempo/diff/rework y falla sin perder diagnostico si
un brazo falla. Test: `bash scripts/test_orquesta_golden_ab_launcher.sh`.
Residual: falta un launcher de proveedor/agente que rellene tokens reales desde
Codex/Claude/Gemini en cada brazo. No se activa ninguna regla nueva global en
este corte.

## Codex local 2026-07-09: BUG-188 ampliado a prompts legacy Claude/Gemini

Hallazgo:

- El cierre anterior de `BUG-ORQ-20260704-188` cubria goal-first
  Claude/Gemini, pero los prompts legacy de agente (`Build*AgentPrompt*`)
  seguian monolingues en español.
- El stack ya tenia `PromptLocale` en config y `goal_backend.prompt_locale`,
  pero los perfiles legacy no transportaban ese campo y los resolvers siempre
  llamaban builders sin locale.

Cierre local aplicado:

- `ClaudeConnectorProfileV0` y `GeminiConnectorProfileV0` ganan
  `prompt_locale` validado como valor seguro.
- Builders nuevos `BuildClaudeAgentPromptWithLocaleV0`,
  `BuildClaudeAgentPromptWithLocaleAndControlFilesV0`,
  `BuildGeminiAgentPromptWithLocaleV0` y
  `BuildGeminiAgentPromptWithLocaleAndControlFilesV0`.
- Default compatible: locale vacio sigue generando el prompt español.
- Locale `en-*`: protocolo operativo legacy, control files, ACK, shutdown,
  durable result, secciones de titulo/objetivo/write-set/tests y notas del
  conector salen en ingles; los codigos de contrato se mantienen iguales.
- `ClaudeGoalProcessBackendV0` y `GeminiGoalProcessBackendV0` heredan
  `PromptLocale` desde `Control.PromptLocale` si el perfil no lo trae.
- `cmd/orquesta-server` propaga `goal_backend.prompt_locale` tambien al perfil
  process.
- `orquesta-app-codex-stack` propaga `PromptLocale` a perfiles reales
  Claude/Gemini para lanzamientos legacy por proveedor.

Pruebas verdes:

- `go test -count=1 ./modulos/orquesta-runtime-claude -run 'Test(ValidateClaudeConnectorProfileV0|ClaudeExecResolverV0|BuildClaudeAgentPrompt)'`
- `go test -count=1 ./modulos/orquesta-runtime-gemini -run 'Test(ValidateGeminiConnectorProfileV0|GeminiExecResolverV0|BuildGeminiAgentPrompt)'`
- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestProviderLaunchSpecResolverV0'`
- `go test -count=1 ./modulos/orquesta-runtime-claude ./modulos/orquesta-runtime-gemini ./modulos/orquesta-app-codex-stack`
- `go test -count=1 ./cmd/orquesta-server -run 'Test(ServerGoalBackendFromEnvV0|ClaudeRuntimeConfigV0|GeminiRuntimeConfigV0)'`

Residual para Claude:

- Falta smoke real opt-in con proveedor verificando idioma efectivo en sesion
  viva. No se lanza localmente porque depende de credenciales/cuota/modelo.

## Codex local 2026-07-09: BUG-079 policy estructurada en turn/start

Hallazgo:

- `BUG-ORQ-20260701-079` seguia abierto por el residual de salidas gigantes
  antes de herramientas internas.
- El contrato `DirectionContract.ToolOutputPolicy` ya existia y el app-server
  lo inyectaba como texto en `input`, pero `turn/start` no transportaba un
  parametro estructurado que un app-server compatible pudiera aplicar antes de
  ejecutar herramientas.

Cierre local aplicado:

- `serverCodexAppServerTurnStartParamsV0` gana `ToolOutputPolicy` y lo serializa
  como `toolOutputPolicy` con `maxTextBytes`, `threadReadMaxBytes`,
  `requireBoundedCommands`, `boundedCommandHints` y
  `durableEvidenceRequired`.
- La politica se deriva de `DirectionContract.ToolOutputPolicy` y se clampa al
  maximo canonico de Orquesta; no puede relajar `max_text_bytes` ni sustituir
  los hints acotados por defecto.
- Compatibilidad: si el app-server real responde `invalid_params` o
  `invalid_request` por `toolOutputPolicy`, Orquesta reintenta el mismo
  `turn/start` sin ese campo JSON, mantiene el contrato textual y conserva una
  evidencia de fallback.
- `codexAppServerStartImmediateLimitedReceiptV0` conserva ahora las evidencias
  reales de `turn/start`, incluido el fallback si ocurrio.

Pruebas verdes:

- `go test -count=1 ./modulos/orquesta-runtime-codex-appserver -run 'TestServerCodexAppServerGoalBackendV0TurnStart|TestServerCodexAppServerTurnStartParamsV0|TestCodexAppServerCommandProtocolTurnStartResponseBudgetV0|TestCodexAppServerLegacyRPCReaderResponseBudgetV0'`
- `go test -count=1 ./modulos/orquesta-runtime-codex-appserver`

Residual:

- No se declara cierre total del bug hasta probar con app-server/proveedor real
  que `toolOutputPolicy` se aplica antes de herramientas. Si el proveedor ignora
  el campo, Orquesta conserva prompt/read caps/sanitizacion/replan, pero no un
  corte duro pre-tool.

## Codex local 2026-07-09: golden metrics lee usage real de proveedor

Hallazgo:

- El A/B de programacion minima ya podia comparar deltas, pero el wrapper de
  metricas solo normalizaba tokens si el launcher interno los escribia
  directamente en `result.json.metrics`.
- Para runs reales, Codex/launchers pueden dejar usage como artefacto separado
  (`codex_usage_accounting.json`, `usage.json`, snapshots de observe/status o
  ruta declarada). Sin lector comun, cada launcher tendria que duplicar parsing.

Cierre local aplicado:

- `scripts/orquesta_golden_metrics_launcher.sh` busca usage en
  `metrics`, `usage_path`, `usage_report_path`, `codex_usage_accounting.json`,
  `usage.json`, `observe_response.json` y `status.json`.
- Normaliza `input_tokens`, `output_tokens`, `reasoning_tokens`,
  `cached_input_tokens` y `total_tokens` desde formas snake/camel y wrappers
  `usage`/`prompt_cache`, sin inventar valores si no hay reporte.
- Anota `metrics.provider_usage_detected=true` y
  `metrics.provider_usage_source` cuando la fuente existe.

Pruebas verdes:

- `bash -n scripts/orquesta_golden_metrics_launcher.sh scripts/test_orquesta_golden_metrics_launcher.sh`
- `bash scripts/test_orquesta_golden_metrics_launcher.sh`
- `bash -n scripts/orquesta_golden_ab_launcher.sh scripts/test_orquesta_golden_ab_launcher.sh`
- `bash scripts/test_orquesta_golden_ab_launcher.sh`
- `scripts/orquesta_golden_evals.sh --self-test --output /tmp/orquesta-golden-self-test-provider-usage.json`

Residual:

- Falta un launcher real Codex/Claude/Gemini que ejecute cada brazo y deje esos
  reportes de usage en el resultado. La infraestructura de medicion ya puede
  ingerirlos sin activar reglas globales.

## Codex local 2026-07-09: BUG-065/165 cleanup_completed compactado en cliente

Hallazgo:

- El servidor ya compactaba `goal_actions` de shutdown cuando una identidad
  tenia `cleanup_completed`.
- El cliente CLI podia recibir una respuesta/status heredada con
  `cleanup_completed` y una accion antigua (`cleanup_requested`,
  `cleanup_attempted`, etc.) para el mismo backend, ignorar solo el
  `cleanup_completed` y conservar la accion vieja como bloqueo.

Cierre local aplicado:

- `shutdownClientBlockingGoalActionsV0` construye un set de identidades
  resueltas por `cleanup_completed`.
- Cualquier accion no terminal con la misma identidad
  `kind/run_ref/work_ref/external_work_ref` se descarta antes de calcular
  `active_work_refs` y `active_work_count`.
- `normalizeServerShutdownClientResultV0` ya no publica un falso
  `shutdown_not_ready active_work=1` cuando el unico bloqueo venia de una
  accion stale ya resuelta.

Pruebas verdes:

- `go test -count=1 ./cmd/orquesta-server -run 'Test(NormalizeServerShutdownClientResultV0GoalActionCompletedOcultaAccionStale|RequestServerShutdownV0ReadyNoSaltaGoalActionsSinActiveWork|RequestServerShutdownV0CoordinaDosGoalsActivosHastaGoalActionsResueltas|RequestServerShutdownV0PostColgadoConsultaStatusAccionable|WaitServerShutdownReadyV0RepostColgadoRespetaDeadline)'`
- `go test -count=1 ./modulos/orquesta-server-shutdown ./modulos/orquesta-server ./cmd/orquesta-server`

Residual:

- Esto cierra un borde local del cliente. No sustituye el smoke real amplio de
  `BUG-165/065` con proveedor lento, backend vivo tras stop o corte externo.

## Codex local 2026-07-09: BUG-196 smoke OPES lifecycle finalpkg sin cuelgue

Contexto:

- Estaba revalidando `BUG-ORQ-20260701-066` con
  `scripts/smoke_opes_lifecycle_real.sh`, sin tocar OPES productivo.
- El primer intento local quedo colgado en
  `/tmp/orquesta-opes-lifecycle-real-20260709T065427Z`: derivados 24/24,
  `goal_receipts_manifest_status=ok`, `empty_after_final=true`, pero
  `finalpkg_requests.jsonl` seguia vacio y el fake server `finalpkg` vivo.

Hallazgo:

- El cuelgue estaba en el harness Bash, no en Orquesta/OPES.
- `run_finalpkg_live_config` hacia
  `fake_orquesta_url="$(start_fake_orquesta_for_finalpkg)"`.
- `start_fake_orquesta_for_finalpkg` arrancaba el servidor Python en background
  heredando stdout; el command substitution no cerraba porque el pipe quedaba
  abierto por el proceso servidor.

Cierre aplicado:

- `scripts/smoke_opes_lifecycle_real.sh`: el fake server redirige stdout/stderr
  a `finalpkg/fake_orquesta_server.log`, de forma que la captura de URL termina
  y el launcher `finalpkg` puede continuar.
- Se corto solo el smoke local colgado y su fake server; no habia proceso OPES
  productivo ni servidor Orquesta real afectado.
- Inventario actualizado con `BUG-ORQ-20260709-196`.

Pruebas verdes:

- `bash -n scripts/smoke_opes_lifecycle_real.sh`
- `timeout 180 env ORQUESTA_KEEP_SMOKE_DIR=1 scripts/smoke_opes_lifecycle_real.sh`

Resultado de revalidacion:

- Evidencia:
  `/tmp/orquesta-opes-lifecycle-real-20260709T065803Z/out/opes_lifecycle_result.json`
- `status=passed`
- 24/24 work kinds OPES cubiertos hasta `finalize_temario_package`.
- `finalpkg_dry_run=false`.
- Un unico POST fake:
  `run-ref-opes-a1-t002-finalpkg-20260612`,
  `job-ref-opes-a1-t002-finalpkg-20260612`,
  `work_kind=finalize_temario_package`.
- `settlement_status=settled_final`.
- Verificacion externa posterior: sin procesos residuales
  `smoke_opes_lifecycle_real`, `fake_orquesta_finalpkg` ni
  `smoke_opes_derivatives_rest`.

Lectura para Claude:

- El residual comun del smoke temporal/fake OPES queda revalidado localmente.
- `BUG-066` solo debe seguir abierto si se exige un hueco mas concreto:
  proveedor real/residente, corte externo/manual real, deploy remoto o OPES
  temporal/preproduccion con credenciales y scope duro.

## Codex local 2026-07-09: golden agent launcher para A/B real

Contexto:

- El harness de golden evals ya tenia manifest, evaluador, wrapper de metricas
  y launcher A/B.
- El residual operativo era que cada proveedor real debia inventar como leer
  `ORQUESTA_GOLDEN_TASK_REQUEST`, como convertirlo en prompt, donde dejar
  `result.json` y como conservar diagnostico si fallaba.

Cierre aplicado:

- Nuevo `scripts/orquesta_golden_agent_launcher.sh`.
- Consume solo las variables existentes `ORQUESTA_GOLDEN_TASK_ID`,
  `ORQUESTA_GOLDEN_TASK_REQUEST` y `ORQUESTA_GOLDEN_TASK_RESULT_DIR`.
- Genera `agent_prompt.md` con objetivo, write-set, ficheros esperados, tests,
  paths prohibidos, skill refs y contrato de salida.
- Ejecuta el proveedor/agent command con variables `GOLDEN_AGENT_*`
  (`REQUEST_PATH`, `PROMPT_PATH`, `RESULT_DIR`, `RESULT_PATH`, `SKILL_REFS`).
- Si falta `result.json`, o el agente sale con error, escribe/preserva
  `status=failed`, `agent_exit_code`, `launcher_issues` y
  `evidence.golden_agent_launcher`; no pierde diagnostico por un exit code.
- No acopla el evaluador a Codex/Claude/Gemini y no anade variables
  `ORQUESTA_*` nuevas.

Pruebas verdes:

- `bash -n scripts/orquesta_golden_agent_launcher.sh scripts/test_orquesta_golden_agent_launcher.sh`
- `bash scripts/test_orquesta_golden_agent_launcher.sh`

Bug adicional cerrado durante la verificacion:

- `BUG-ORQ-20260709-197`: `scripts/orquesta_golden_evals.sh --run --task
  golden-new-app-smoke-v0` lanzaba una sola tarea, pero evaluaba las cinco del
  manifest. Resultado observado antes del fix: la tarea ejecutada pasaba con
  `score=1.0`, pero el reporte global quedaba `score=0.2` y `status=failed`
  por cuatro `result_missing` artificiales.
- Cierre: `manifest_with_tasks()` filtra el manifest que llega a `evaluate()`
  tanto en `--run` como en `--evaluate`.
- Prueba nueva: `scripts/test_orquesta_golden_evals.sh`.

Lectura para Claude:

- El A/B empirico de `orquesta-programacion-minima` ya tiene tres piezas
  locales: launcher puente de agente, wrapper de metricas/usage y comparador
  A/B. Tambien queda corregida la ejecucion acotada por `--task`, clave para no
  gastar cuota en las cinco tareas cuando solo se quiere validar un brazo. Sigue
  pendiente la ejecucion con proveedor/cuota real para obtener tokens reales de
  Codex/Claude/Gemini.

## Codex local 2026-07-09: retirada efectiva de app_server_proxy

Contexto:

- El inventario conservaba un residual de decision sobre `app_server_proxy`.
- El contrato vigente ya decia que `app_server_tmux` es el unico backend Codex
  app-server operativo normal.
- El codigo rechazaba `app_server_proxy`, pero quedaban helpers muertos capaces
  de construir `codex app-server proxy`.

Cierre aplicado:

- `serverCodexGoalBackendFromEnvForWorkDirV0` mantiene el reconocimiento del
  valor historico para devolver errores especificos.
- Sin opt-in devuelve `codex_goal_backend_proxy_diagnostic_opt_in_required`.
- Con opt-in devuelve `codex_goal_backend_proxy_diagnostic_not_operational`.
- Se retiraron los helpers muertos que generaban argumentos `app-server proxy`
  y el lector directo de opt-in ya no usado.

Prueba nueva:

- `TestServerGoalBackendFromEnvV0RechazaProxyHistoricoAunqueTengaOptInV0`.

Lectura para Claude:

- No queda una ruta local que convierta `app_server_proxy` en comando ejecutable
  desde el servidor. Si en remoto se quiere recuperar proxy real, debe entrar
  como nuevo diseno/adaptador con contrato y pruebas, no como fallback oculto.

## Codex local 2026-07-09: protocolo remoto unico de integracion Git

Contexto:

- `BUG-ORQ-20260705-191` mostro que un goal remoto podia terminar `complete`
  con codigo util en un worktree piloto, pero sin commit ni integracion en la
  rama principal.
- El inventario mantenia pendiente documentar un protocolo unico para bundle,
  checkout, identidad Git, patch, summary y no tocar produccion.

Cierre documental:

- `docs/runbooks/protocolo_git_remoto_orquesta_2026-07-02.md` queda como
  protocolo canonico.
- Declara que `goal_result status=complete`, `queue.count=0`,
  `pending_push` o cambios en piloto no equivalen a integracion.
- Exige `integration_receipt_ref` o estado `pending_integration`/`blocked_push`
  con patch o bundle exportado, summary, write-set, pruebas y accion siguiente.
- Añade comprobacion de identidad Git, checkout/worktrees y modo bundle/patch
  oficial si el remoto no puede hacer `fetch`/`push` directo a GitHub.
- La incidencia G5 queda enlazada al protocolo.

Lectura para Claude:

- El residual documental queda cerrado localmente. Sigue pendiente la validacion
  operativa en servidor con remote Git canonico o flujo bundle/push real cuando
  haya acceso remoto/auth/cuota.

## Codex local 2026-07-09: operational-status consume estado vivo

Contexto:

- Russell reviso el residual de `BUG-ORQ-20260704-165`: habia mitigaciones para
  `observe/status` largo, pero el diagnostico residente aun salia solo del
  `StatusTrackerV0` y no agregaba la fuente neutral de runs, procesos,
  ACK/receipts y deliveries que ya usa MCP.

Cierre aplicado:

- `ResidentOperationalStatusSourceV0` acepta `EstadoVivoSource` de
  `orquesta-estado-vivo`.
- La consulta agrega contadores compactos de evidencias, nodos, conflictos,
  procesos vivos, terminales y entregas parciales.
- Agrega referencias opacas a run/goal/evidencias respetando el presupuesto de
  `DiagnosticoCompactoV0`.
- La fuente se consulta con timeout de 500 ms y limite 32; si falla, queda
  warning `estado_vivo_unavailable` y el status no se cae.
- `RuntimeV0` recibe el puerto por deps y `cmd/orquesta-server` le pasa
  `stack.MCPTransportBindings.AutoprogrammingEstadoVivoSource`, reutilizando la
  composicion existente.

Pruebas verdes:

- `go test -count=1 ./modulos/orquesta-server -run 'TestResidentOperationalStatusSourceV0|TestHandlerV0OperationalStatusQueryV0'`
- `go test -count=1 ./modulos/orquesta-server`
- `go test -count=1 ./cmd/orquesta-server -run 'Test.*(OperationalStatus|Stack|Runtime|GoalFirstAppHTTP|BuildRuntime)'`

Lectura para Claude:

- Esto reduce el falso verde de operational-status: ahora puede ver estado vivo
  multi-fuente sin importar el stack desde `orquesta-server`. No declara cerrado
  `BUG-165` total; sigue pendiente smoke real amplio con proveedor lento y
  auditoria completa de submit/ack/observe largos.

## Codex local 2026-07-09: wizard U12 deja de pisar autonomia

Contexto:

- Hubble reviso el wizard y encontro un bug concreto: U6 colaboracion y U12
  usaban el mismo campo `agentes.autonomia`.
- Eso podia cerrar U12 al responder U6 y podia sobrescribir la autonomia al
  responder la pregunta que debia tratar historico/versiones.

Cierre aplicado:

- `wizard-u12-autonomia` se sustituye por
  `wizard-u12-historico-versiones`.
- U12 escribe en `datos.operacion.restricciones`, campo existente del contrato
  publico, sin anadir schema nuevo.
- Opciones U12: `sin_historico`, `versionado_basico`,
  `historial_completo`.
- `wizardDecisionsForAnswerV0` materializa esas opciones como restricciones
  operativas y no toca `agentes.autonomia`.
- Catalogo i18n es/en y `docs/wizard_glosario_generado.md` regenerado.

Pruebas verdes:

- `ORQUESTA_UPDATE_WIZARD_GLOSSARY=1 go test -count=1 ./modulos/orquesta-web -run TestWizardGlosarioGeneradoV0`
- `go test -count=1 ./modulos/orquesta-web`

Lectura para Claude:

- Cierra `BUG-ORQ-20260709-WIZARD-U12-AUTONOMIA-PISADA`.
- No declara completo todo el redisenyo U1-U12; solo corrige una pisada real
  de campos dentro del wizard existente.

## Codex local 2026-07-09: revision programacion minima/tokens

Contexto:

- El operador pidio buscar skills/prompts/configuracion anti-overengineering y
  ahorro de tokens, pero con prueba empirica antes de activar medidas nuevas.
- Ya existia `docs/auditoria_programacion_minima_tokens_2026-07-08.md`.

Cierre documental:

- Se revalido la auditoria con fuentes externas actuales: OpenAI Codex
  `AGENTS.md`, OpenAI Skills, Claude Code Costs/Memory, `agents-md`, Caveman y
  `claude-token-efficient`.
- Se corrigio la referencia empirica de paper a
  `https://arxiv.org/html/2601.20404v2`.
- Conclusion conservadora: no tocar `AGENTS.md`; mantener
  `skills/orquesta-programacion-minima/SKILL.md` como opt-in y activar default
  solo tras A/B real.

Pruebas verdes:

- `bash -n scripts/orquesta_golden_evals.sh scripts/orquesta_golden_metrics_launcher.sh scripts/orquesta_golden_ab_launcher.sh scripts/orquesta_golden_agent_launcher.sh`
- `scripts/orquesta_golden_evals.sh --self-test --output /tmp/orquesta-golden-selftest-codex-20260709.json` -> `status=passed`, `score=1.0`

Lectura para Claude:

- El harness local esta listo; falta A/B con proveedor/cuota real para saber si
  la skill reduce tokens/diff sin aumentar rework ni bajar calidad.

## Codex local 2026-07-09: BUG-075 aliases OPES extendidos en director

Contexto:

- El explorador read-only confirmo que `BUG-075` no se puede cerrar completo con
  unit tests: faltan smoke OPES temporal/external-work, proveedor real y prueba
  de ausencia de reescritura tardia.
- Hueco local detectado: el bridge OPES emite tipos propietarios
  `learning_games_package`, `help_manual_package`,
  `opes_quality_audit_report` y `completed_syllabus_package`; el director OPES
  podia no aplicarles el mismo contrato estructurado que a los tipos neutrales.

Cierre aplicado:

- Nuevo helper local `artifact_type_aliases_v0.go` en `orquesta-opes-director`
  normaliza esos aliases solo dentro del adaptador OPES.
- `OPESArtifactQualityContractV0` valida `learning_games_package` como
  practica interactiva, `help_manual_package` como paquete de ayuda y
  `opes_quality_audit_report` con `decision_global` + evidencias/rework.
- El registro OPES trata `completed_syllabus_package` como paquete final para
  release/followup de `manifest_cierre`, conservando el tipo original como
  `source_artifact_type`.

Pruebas verdes:

- `go test -count=1 ./modulos/orquesta-opes-director -run 'Test(ValidateOPESArtifactQualityContractV0(CubreWorkKindsMinimos|NormalizaArtefactosOPESExtendidos)|ProduceOPESCausalJobsV0(ArtefactoOPESExtendidoAplicaArtifactQuality|CompletedSyllabusPackageConManifestLiberaRegistro|WorkDeliveryGenericoConWorkKindVisualAplicaArtifactQuality|HTMLConContratoArtifactQualityPassNoCreaRework|BloqueaSecuenciaOPESCompletaSinEvidenciaMinima)|TopicRegistryRequiredEvidencePolicyV0CubreSecuenciaOPESCompleta)V0'`
- `go test -count=1 ./modulos/orquesta-opes-director`

Lectura para Claude:

- Esto reduce `BUG-075`; no lo cierres como resuelto total. Pendientes reales:
  smoke OPES temporal con external-work, artefactos reales/proveedor y prueba de
  que el cierre no reescribe tarde ni deja parcialidades sin rework causal.

## Codex local 2026-07-09: smoke servidor local tras BUG-075

Evidencia:

- Se compilo `cmd/orquesta-server` en `/tmp/orquesta-server-codex-local` desde
  `b56d1cbb9ca6`.
- Se arranco un servidor temporal aislado con
  `ORQUESTA_SERVER_STATE_DIR=/tmp/orquesta-codex-local-smoke/state`,
  `ORQUESTA_CODEX_RUNTIME_WORKDIR=/tmp/orquesta-codex-local-smoke/runtime`,
  automejora idle desactivada y director residente desactivado.
- State aislado: `startup_status=startup_ready`,
  `availability_status=running`, `availability_reason=server_ready`,
  `addr=127.0.0.1:44851`.
- Shutdown HTTP gobernado con `cleanup_goal_backends=true` devolvio
  `shutdown_ready=true`, `agents_in_flight=0`, `runs_requested=0`,
  `runs_stopped=0`; el proceso foreground termino con codigo 0.

Lectura para Claude:

- Esto solo valida build/start/status/shutdown local aislado del servidor en el
  commit actual. No sustituye los smokes reales pendientes de proveedor Goal,
  OPES temporal, remoto ni Telegram.

## Codex local 2026-07-09: retirada alias seconds en smokes Codex

Contexto:

- El subagente McClintock detecto un resto de TAREA-8.4/MEJ-106: el alias
  legacy `ORQUESTA_CODEX_SMOKE_TIMEOUT_SECONDS` seguia aceptado en smokes Codex
  reales aunque la superficie canonica ya era `ORQUESTA_CODEX_SMOKE_TIMEOUT_MS`.
- Mantener ambas unidades era una fuente de error x1000 y mantenia una env extra
  en el ratchet.

Cierre aplicado:

- `orquesta-runtime-codex-delivery` y `orquesta-app-codex-stack` leen solo
  `ORQUESTA_CODEX_SMOKE_TIMEOUT_MS`; si falta, usan su default local de smoke.
- Scripts y ejemplos se pasan a milisegundos.
- La metrica baja de 514 a `env_vars_orquesta=513`.
- No se tocan `ORQUESTA_TELEGRAM_OPERATOR_ENABLED` ni
  `ORQUESTA_TELEGRAM_OPERATOR_TOKEN`: pertenecen al canal operador y deben pasar
  a config/secreto gestionado en un corte propio.

Verificado:

- `bash -n scripts/smoke_codex_real_required_test_runner.sh scripts/smoke_codex_real_operational_wave.sh scripts/smoke_codex_real_recursive_tree.sh`
- `go test -count=1 ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-app-codex-stack -run Smoke`
- `go test -count=1 ./cmd/orquesta-server -run 'EnvVarsOrquestaRatchet|ServerEnvRegistry'`
- `bash scripts/orquesta_metricas_deuda.sh --json`
- `git diff --check`

Lectura para Claude:

- Este corte no cierra la consolidacion completa de variables. Cierra solo el
  alias de timeout Codex en segundos y deja el techo temporal documentado en
  `env_vars_orquesta_allow_increase_to=513`.

## Codex local 2026-07-09: Telegram operator solo por config canonica

Contexto:

- El corte anterior dejo pendiente bajar `env_vars_orquesta=513` a la base 511:
  las dos envs restantes eran `ORQUESTA_TELEGRAM_OPERATOR_ENABLED` y
  `ORQUESTA_TELEGRAM_OPERATOR_TOKEN`.
- Telegram operator ya tenia seccion canonica `telegram_operator.*` en
  `orquesta.config.json`, con redaccion de token/chats/target en
  `effective_config`; la duplicidad de envs ya no aportaba contrato nuevo.
- El subagente Hume detecto ademas un falso source: si
  `notification_target_ref` se derivaba de `authorized_chat_refs[0]`,
  `effective_config` lo publicaba como `defaulted`.

Cierre aplicado:

- `cmd/orquesta-server` deja de leer `ORQUESTA_TELEGRAM_OPERATOR_ENABLED` y
  `ORQUESTA_TELEGRAM_OPERATOR_TOKEN`.
- `telegram_operator.enabled` y `telegram_operator.token` se publican como
  settings canonicos de fichero, el token siempre redactado.
- `telegram_operator.notification_target_ref` derivado desde chat autorizado
  publica `source=config_file`, no `defaulted`, y mantiene valor redactado.
- Se elimina el registry de envs Telegram operator y la metrica vuelve a
  `env_vars_orquesta=511`.

Verificado:

- `go test -count=1 ./cmd/orquesta-server -run 'TestTelegramOperator|TestTelegramBotAPI|TestServerEnvRegistry|TestEnvVarsOrquestaRatchetMEJ106V0'`
- `bash scripts/orquesta_metricas_deuda.sh --json`

Lectura para Claude:

- Esto cierra el residual local de MEJ-106/TAREA-8 sobre Telegram envs. No
  cierra el despliegue remoto Telegram real: sigue haciendo falta config local
  con token, reinicio de Orquesta remoto y prueba desde Telegram/webhook/poller.

## Codex local 2026-07-09: smoke forced-stop cubre autoprogramming/status

Contexto:

- El subagente Ptolemy recomendo como siguiente frente `BUG-165/065/079`:
  smoke real amplio de `status/observe/runs-control/shutdown` con proveedor
  lento/vivo.
- El harness `scripts/smoke_goal_first_forced_stop_backend_real.sh` ya validaba
  `runs/control`, `observe` posterior y shutdown/cleanup de app-server, pero no
  materializaba snapshots de `/api/v0/autoprogramming/status`.

Cierre aplicado:

- `scripts/smoke_goal_first_app_server_real.sh` anade
  `post_autoprogramming_status_snapshot`.
- En modo forced-stop, el smoke consulta `/api/v0/autoprogramming/status` antes
  de `runs/control` y despues del `observe` terminal.
- El snapshot debe contener `run_ref`/`goal_ref`/`external_goal_ref`; despues del
  forced-stop, el status no puede publicar ese goal como `running`.
- El guard `TestSmokeGoalFirstForcedStopWrapperEjercitaRunControlBackendVivoV0`
  fija que el contrato no desaparezca del harness.

Verificado:

- `bash -n scripts/smoke_goal_first_app_server_real.sh scripts/smoke_goal_first_forced_stop_backend_real.sh`
- `go test -count=1 ./cmd/orquesta-server -run 'TestSmokeGoalFirst(AppServerReal|ForcedStop)'`
- Smoke real Codex app-server tmux:
  `ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_REAL_CONFIRM=1 ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_CODEX_EXECUTION_CONFIRMED=1 ORQUESTA_GOAL_FIRST_SMOKE_POLLS=50 ORQUESTA_KEEP_SMOKE_DIR=1 ORQUESTA_SMOKE_PARENT=/tmp/orquesta-smokes-codex ./scripts/smoke_goal_first_forced_stop_backend_real.sh`
  -> `smoke_goal_first_forced_stop_backend_real=ok`,
  `autoprogramming_status_before_forced_stop_visible=true`,
  `run_control_status=stopped`, `run_control_goal_status_after=blocked`,
  `observe_after_forced_stop_goal_status=blocked`,
  `autoprogramming_status_after_forced_stop_not_running=true`,
  `app_server_tmux_processes_alive=0`.

Lectura para Claude:

- Esto cierra con proveedor real el subcaso forced-stop/status/observe/shutdown
  del harness. No cierra el bug global: quedan escenarios amplios de
  observabilidad lenta/stale fuera de este smoke y despliegue remoto.
- Evidencia saneada: `/tmp/orquesta-smokes-codex/orquesta-goal-first-app-server.5Y0PiP`
  (~512 KiB; se borro `codex-home` y el binario temporal).

## Codex local 2026-07-09: revalidacion OPES lifecycle y golden A/B

Contexto:

- Tras los commits de Telegram config y forced-stop/status, se revalidaron dos
  cierres recientes que alimentan pendientes vivos: `BUG-ORQ-20260709-196`
  (smoke OPES lifecycle/finalpkg) y `BUG-ORQ-20260709-197` (golden evals por
  tarea y A/B).
- No se cambio codigo en este corte; solo se conserva evidencia para Claude.

Verificado:

- `timeout 240 env ORQUESTA_KEEP_SMOKE_DIR=1 scripts/smoke_opes_lifecycle_real.sh`
  -> `status=passed`, `derivatives_sequence_complete=true`, 24/24 work kinds
  cubiertos hasta `finalize_temario_package`, `finalpkg_dry_run=false`,
  `finalpkg_run_ref=run-ref-opes-a1-t002-finalpkg-20260612`,
  `settlement_status=settled_final` y `no_residual_processes=true`.
- Evidencia retenida:
  `/tmp/orquesta-opes-lifecycle-real-20260709T111212Z/out/opes_lifecycle_result.json`.
- `bash scripts/test_orquesta_golden_evals.sh &&
  bash scripts/test_orquesta_golden_ab_launcher.sh &&
  bash scripts/test_orquesta_golden_agent_launcher.sh &&
  bash scripts/test_orquesta_golden_metrics_launcher.sh`
  -> los cuatro harness locales en verde.

Lectura para Claude:

- OPES fake/residente queda revalidado en el repo actual, incluido `finalpkg`
  live contra fake loopback y settlement final. No usar esto como prueba de
  proveedor real, despliegue remoto ni OPES productivo.
- El A/B local de `orquesta-programacion-minima` ya tiene infraestructura y
  tests verdes. Falta ejecutar comparativa con proveedor/cuota real antes de
  convertir la skill en default amplio.

## Codex local 2026-07-09: control_plane en config canonica

Contexto:

- El subagente Cicero detecto un residual estrecho de TAREA-8: `control_plane`
  seguia siendo env-only aunque el resto de familias operativas grandes ya
  estaban movidas a `orquesta.config.json`.
- No se retiran envs en este corte para conservar compatibilidad de despliegue
  remoto; se mantiene `env > config > default` y se marca la env como override
  deprecated cuando pisa fichero.

Cierre aplicado:

- `orquesta.config.json` acepta seccion `control_plane` con
  `remote_access_opt_in`, `token`, `principal`, `permission_ref` y
  `public_reason`.
- `serverConfigFromEnvV0` usa esos valores para
  `orquestaserver.ControlPlaneConfigV0`.
- `effective_config` publica fuente `config_file` para las cinco claves y
  redacta el token como presencia `present/absent`.
- Si una env `ORQUESTA_SERVER_CONTROL_*` coexiste con fichero, se emite
  diagnostico `deprecated_env_used` con scope `control_plane`.

Verificado:

- `go test -count=1 ./cmd/orquesta-server -run 'TestServerConfigFromEnvV0(.*Control|LeeControlPlane|ControlPlaneEnv)|TestServerEnvRegistry|TestEnvVarsOrquestaRatchetMEJ106V0|TestServerEnvRegistryASTV0'`
- `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`
- `go test -count=1 ./...`
- `bash scripts/orquesta_metricas_deuda.sh --json` ->
  `env_vars_orquesta=511`
- `git diff --check`

Lectura para Claude:

- Esto reduce TAREA-8 para control-plane local. No cierres el frente remoto:
  `scripts/orquesta_server_ctl.sh` y despliegue servidor siguen siendo corte
  separado, y el token real debe vivir en config/secreto del servidor.

## Codex local 2026-07-09: shutdown/status real sin run-control directo

Contexto:

- Se uso subagentes para revisar servidor/status/shutdown, guard tests y docs
  de `BUG-165/065/079`.
- El objetivo era cubrir otro subcaso real: `app_server_tmux` vivo,
  `/api/v0/autoprogramming/status` visible y shutdown por
  `/api/v0/server/shutdown` con `cleanup_goal_backends=true`, sin llamar al
  endpoint `/api/v0/runs/control` como camino principal del smoke.

Hallazgo y fix:

- Primer smoke retenido en
  `/tmp/orquesta-smokes-codex/orquesta-goal-first-app-server.3QX7lP` reprodujo
  `backend_still_running` falso: tras cleanup no quedaban tmux/socket/owner ni
  procesos, pero `ReadActiveShutdownWorkV0` seguia reportando active work por
  estado degradado sin evidencia viva. El harness tambien tenia
  `session_name: unbound variable` al diagnosticar el fallo.
- Se corrigio `app_server_tmux`: un residuo configurado solo bloquea shutdown si
  observa owner, sesion, socket, pane o proceso real.
- Se corrigio el harness inicializando `session_name` y se agrego wrapper
  `scripts/smoke_goal_first_shutdown_coordination_real.sh` con guard para no
  convertirse en no-op ni usar `/api/v0/runs/control`.

Verificado:

- `bash -n scripts/smoke_goal_first_app_server_real.sh scripts/smoke_goal_first_shutdown_coordination_real.sh`
- `go test -count=1 ./modulos/orquesta-runtime-codex-appserver -run 'TestCodexAppServerTmuxBackendV0(EnsureShutdownCleanupMigrado|ReadActiveShutdownWorkIgnoraEstadoDegradadoSinResiduoVivo)'`
- `go test -count=1 ./cmd/orquesta-server -run 'TestSmokeGoalFirstShutdownCoordinationReal'`
- Smoke real:
  `ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_REAL_CONFIRM=1 ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_CODEX_EXECUTION_CONFIRMED=1 ORQUESTA_GOAL_FIRST_SMOKE_POLLS=50 ORQUESTA_GOAL_FIRST_SMOKE_SLEEP_SECONDS=3 ORQUESTA_KEEP_SMOKE_DIR=1 ORQUESTA_SMOKE_PARENT=/tmp/orquesta-smokes-codex ./scripts/smoke_goal_first_shutdown_coordination_real.sh`
  -> `smoke_goal_first_shutdown_coordination_real=ok`,
  `autoprogramming_status_before_shutdown_visible=true`,
  `status=ready`, `shutdown_ready=true`, `active_work_count=0`,
  `cleanup_completed` y `app_server_tmux_processes_alive=0`.
- Verificacion final del corte: `go test -count=1 ./...` y
  `git diff --check`.

Lectura para Claude:

- Esto cierra `BUG-ORQ-20260709-198` y reduce `BUG-165/065` en el subcaso
  cleanup/status/shutdown real de backend `app_server_tmux`.
- No cerrar `BUG-165/065` global: el smoke final acaba con
  `runs_requested=0`, asi que queda pendiente una prueba/correccion de
  reconciliacion completa de runs goal-first fuera de cola durante shutdown
  amplio, mas los escenarios de proveedor lento/stale y despliegue remoto.
- Evidencias saneadas:
  `/tmp/orquesta-smokes-codex/orquesta-goal-first-app-server.Ya4ayF` (verde)
  y `/tmp/orquesta-smokes-codex/orquesta-goal-first-app-server.3QX7lP`
  (reproduccion fallida), ambas sin `codex-home` ni binario temporal.

## Codex local 2026-07-09: shutdown coordina goal-first terminal fuera de cola

Subcaso nuevo cerrado sobre el residual anterior de `BUG-165/065`: si un run
goal-first ya no aparece como candidato ejecutable de `RunQueue`, pero conserva
`RunControl=stop_requested`/`cancel_requested` y el `GoalWorkState` ya esta
terminal, `/api/v0/server/shutdown` no debe publicar un falso
`runs_requested=0` invisible.

Cambio aplicado en `modulos/orquesta-app-codex-stack/server_shutdown_v0.go`,
sin tocar core puro:

- `serverShutdownExecutorV0` usa un `QueueReader` de stack que suplementa
  candidatos goal-first terminales con control pendiente.
- `stackShutdownActiveWorkReaderV0` publica esos estados como `goal_first`
  activo mientras no haya backend Goal vivo, para impedir falso `ready`.
- `stackShutdownRunControlWriterV0` completa el control forzado a
  `stopped/canceled` cuando el goal ya esta terminal y conserva evidencia
  `evidence-ref-server-shutdown-goal-terminal-run-control-reconciled`.

Prueba nueva:

- `TestStackShutdownV0ForzadoCoordinaGoalFirstTerminalFueraDeColaV0`: cola
  vacia, `GoalWorkState=complete` con closure aceptada,
  `RunControl=stop_requested`; el shutdown forzado acaba con
  `runs_requested=1`, `runs_stopped=1`, `shutdown_ready=true`,
  `ControlStatus=stopped` y `RunControl=stopped`.

Verificado:

- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestStackShutdown(V0ForzadoCoordinaGoalFirstTerminalFueraDeCola|ActiveWorkReaderV0|RunControlWriterV0)'`
- `go test -count=1 ./modulos/orquesta-app-codex-stack`
- `go test -count=1 ./modulos/orquesta-server-shutdown`

Pendiente: smoke real amplio con proveedor lento/stale/remoto antes de cerrar
`BUG-165/065` global.

## Codex local 2026-07-09: shutdown HTTP semi-real y smoke real endurecido

Avance adicional sobre `BUG-165/065`:

- `cmd/orquesta-server/goal_first_app_http_flow_v0_test.go` anade
  `TestServerAppHTTPGoalFirstShutdownCoordinaControlPendienteFueraDeColaV0`.
  La prueba levanta el handler HTTP real con backend goal-first fake, lanza por
  `/api/v0/apps/director`, cierra el goal con `/api/v0/apps/director/goal/observe`,
  deja `RunControl=stop_requested` y `RunControl=cancel_requested` con
  `/api/v0/runs/control forced=false`, y verifica que
  `/api/v0/server/shutdown forced=true cleanup_goal_backends=true` devuelve
  `runs_requested=1`, `runs_stopped=1`, `shutdown_ready=true`,
  `active_work_count=0` y terminaliza en `stopped/canceled`.
- `stackShutdownRunControlWriterV0` conserva un `cancel_requested` previo como
  `canceled` aunque el shutdown amplio invoque internamente `StopRunV0`.
- `scripts/smoke_goal_first_app_server_real.sh` ya no acepta un
  `shutdown_ready=true` del modo `shutdown_coordination` si no hay
  `runs_requested>=1`, `runs_stopped>=runs_requested` y todos los
  `runs[].control_status=stopped`.
- El guard `TestSmokeGoalFirstShutdownCoordinationRealNoEsNoopV0` comprueba ese
  contrato en el script real.

Verificado:

- `bash -n scripts/smoke_goal_first_app_server_real.sh scripts/smoke_goal_first_shutdown_coordination_real.sh`
- `go test -count=1 ./cmd/orquesta-server -run 'TestServerAppHTTPGoalFirstShutdownCoordinaControlPendienteFueraDeCola|TestSmokeGoalFirstShutdownCoordinationReal'`
- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestStackShutdownV0ForzadoCoordinaGoalFirstTerminalFueraDeCola|TestStackShutdownRunControlWriterV0ForcedStopMarcaGoalTerminalReplanificable'`

Pendiente: ejecutar el smoke real amplio con proveedor lento/stale/remoto. Este
corte prepara el smoke para fallar si reaparece el falso `runs_requested=0`,
pero no sustituye la ejecucion real.

## Codex local 2026-07-09: smoke real shutdown goal-first fuera de cola

Ejecucion real posterior al endurecimiento del smoke:

```bash
ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_REAL_CONFIRM=1 \
ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_CODEX_EXECUTION_CONFIRMED=1 \
ORQUESTA_GOAL_FIRST_SMOKE_POLLS=50 \
ORQUESTA_GOAL_FIRST_SMOKE_SLEEP_SECONDS=3 \
ORQUESTA_KEEP_SMOKE_DIR=1 \
ORQUESTA_SMOKE_PARENT=/tmp/orquesta-smokes-codex \
./scripts/smoke_goal_first_shutdown_coordination_real.sh
```

Resultado:

- `smoke_goal_first_shutdown_coordination_real=ok`.
- `autoprogramming_status_before_shutdown_visible=true`.
- `/api/v0/server/shutdown` devolvio `status=ready`,
  `shutdown_ready=true`, `runs_requested=1`, `runs_stopped=1`.
- `run_control_statuses=stopped` y
  `shutdown_coordination_all_runs_stopped=true`.
- `goal_actions[0].action_taken=cleanup_completed`.
- `app_server_tmux_processes_alive=0`.

Refs:

- `run_ref=run-spec-smoke-goal-first-bug088-req-smoke-goal-first-bug088-bd43a0d5c0ed93df6c8f195055979a74`.
- `external_goal_ref=019f46cc-ae82-7352-bd4f-563f6ff34200`.
- Evidencia retenida y saneada:
  `/tmp/orquesta-smokes-codex/orquesta-goal-first-app-server.kALS5q`
  (~376 KiB, sin `codex-home`, `auth.json`, `config.toml` ni binario temporal).

Lectura:

- El hueco local `runs_requested=0` de `BUG-165/065` queda cerrado para
  backend real `app_server_tmux`: el smoke no usa `/api/v0/runs/control` como
  camino principal, el status previo crea el control pendiente por alto consumo
  y el shutdown amplio lo reconcilia a `stopped`.
- No se detectaron procesos residuales `orquesta-server run`,
  `codex app-server`, `orquesta-goal-*` ni sesiones tmux al terminar.
- Queda como residual externo, si se exige antes de cierre de producto, repetir
  la misma validacion en servidor remoto/stale. En local no queda un bug de
  codigo conocido en este eje.

## Codex local 2026-07-09: BUG-079 evidencia toolOutputPolicy

Avance pequeno y verificable sobre `BUG-ORQ-20260701-079`:

- `orquesta-runtime-codex-appserver` ya distingue en el receipt de arranque si
  la politica estructurada `toolOutputPolicy` fue enviada, aceptada por
  `turn/start` o si se uso fallback legacy.
- Evidencias nuevas:
  `evidence-ref-codex-app-server-turn-start-tool-output-policy-sent`,
  `evidence-ref-codex-app-server-turn-start-tool-output-policy-accepted` y
  `evidence-ref-codex-app-server-turn-start-tool-output-policy-fallback`.
- No cambia el contrato ni anade rails nuevos: solo hace observable el camino
  real para que el siguiente smoke distinga "app-server acepto el campo" de
  "Orquesta reintento sin JSON estructurado".

Verificado:

- `go test -count=1 ./modulos/orquesta-runtime-codex-appserver -run 'TestServerCodexAppServerGoalBackendV0TurnStart|TestServerCodexAppServerTurnStartParamsV0'`
- `go test -count=1 ./modulos/orquesta-runtime-codex-appserver`
- `git diff --check`

Pendiente: no cerrar `BUG-079` hasta una prueba real que demuestre que el
proveedor/app-server aplica el limite antes de ejecutar herramientas con stdout
gigante. Este corte solo mejora la evidencia causal del transporte.

## Codex local 2026-07-09: BUG-066 cierre local/fake-residente

Lectura actualizada para Claude:

- `BUG-ORQ-20260701-066` queda cerrado en alcance local/fake-residente.
- Evidencia OPES lifecycle: `scripts/smoke_opes_lifecycle_real.sh` paso con
  24/24 work kinds, `finalpkg_dry_run=false`,
  `settlement_status=settled_final` y `no_residual_processes=true`.
- Evidencia retenida:
  `/tmp/orquesta-opes-lifecycle-real-20260709T111212Z/out/opes_lifecycle_result.json`.
- Evidencia shutdown local complementaria: smoke real goal-first con backend
  `app_server_tmux`, `runs_requested=1`, `runs_stopped=1`,
  `run_control_statuses=stopped`, cleanup completo y sin procesos residuales.
- Evidencia retenida:
  `/tmp/orquesta-smokes-codex/orquesta-goal-first-app-server.kALS5q`.

No sobrecerrar producto: queda como residual externo repetir la misma clase de
prueba en servidor remoto/stale, OPES temporal/preproduccion o proveedor real
autorizado si se exige evidencia fuera de local. Eso ya no debe contarse como
bug local de codigo abierto.

## Orquesta local 2026-07-09: ola real BUG-079 y BUG-199

Se uso Orquesta para programar, no solo Codex directo:

- Ola: `codex-launch-director-wave`.
- `wave_ref=codex-bug079-smoke-policy-real-20260709`.
- Agentes: 1.
- Write-set declarado:
  `scripts/smoke_goal_first_app_server_real.sh`,
  `scripts/smoke_goal_first_forced_stop_backend_real.sh`,
  `cmd/orquesta-server/goal_first_app_http_flow_v0_test.go`,
  `cmd/orquesta-server/smoke_goal_first_scripts_v0_test.go`.

Entrega de Orquesta:

- `scripts/smoke_goal_first_app_server_real.sh` ahora calcula
  `tool_output_policy_transport=accepted|fallback|sent_without_accept_or_fallback|missing`.
- El smoke falla si va a declarar verde sin evidencia
  `evidence-ref-codex-app-server-turn-start-tool-output-policy-sent` y
  `accepted` o `fallback`.
- `cmd/orquesta-server/goal_first_app_http_flow_v0_test.go` fija evidencia
  `sent` + `accepted` en el fake HTTP.
- `cmd/orquesta-server/smoke_goal_first_scripts_v0_test.go` guarda que el
  script no pierda esa comprobacion.

Lectura:

- `BUG-ORQ-20260701-079` sigue abierto para enforcement pre-tool real del
  proveedor/app-server. Este corte solo evita falsos verdes del smoke cuando no
  hay evidencia de transporte.
- La ola de Orquesta destapo `BUG-ORQ-20260709-199`: el agente termino y dejo
  `codex_last_message.txt`, pero no aparecio `codex_process_done_v0` porque el
  marcador dependia de un goroutine `cmd.Wait()` del CLI lanzador. Si el CLI
  sale, esa evidencia durable puede perderse.
- Cierre de `BUG-199`: `orquesta_codex_exec_v0.sh` escribe por si mismo
  `codex_process_done_v0=completed|failed`, con escritura temporal + `mv`, y
  tambien marca `failed` en trap de senal.
- Reproduccion post-fix con Orquesta y runtime fake:
  `wave=codex-process-done-repro-20260709T135205Z`,
  runtime `/home/alberto/Trabajo/runtime/codex-process-done-repro-20260709T135205Z`,
  resultado `agent-01/codex_process_done_v0=completed`.

Verificado:

- `bash -n scripts/smoke_goal_first_app_server_real.sh scripts/smoke_goal_first_forced_stop_backend_real.sh`
- `go test -count=1 ./cmd/orquesta-server -run 'TestSmokeGoalFirst(AppServerReal|ForcedStop|ShutdownCoordinationReal)|TestServerAppHTTPGoalFirstLanzaObservaYCierraV0|TestCodexLaunchDirectorWaveCommandV0RecursiveFakeRuntimeEjecutableConLinaje'`
- `go test -count=1 ./modulos/orquesta-runtime-codex -run 'TestCodexWrapperV0EscribeProcessDone|TestCodexWrapperV0'`
- `go test -count=1 ./modulos/orquesta-runtime-codex`
- `go test -count=1 ./modulos/orquesta-runtime-codex-appserver -run 'TestServerCodexAppServerGoalBackendV0TurnStart|TestServerCodexAppServerTurnStartParamsV0'`

## Codex local 2026-07-09: smoke real BUG-079 toolOutputPolicy accepted

Objetivo: no cerrar `BUG-079` por unit tests; comprobar con app-server real si
la politica estructurada `toolOutputPolicy` llega como `accepted`, `fallback` o
`missing`.

Preflight:

- `ORQUESTA_GOAL_FIRST_SMOKE_PREFLIGHT_ONLY=1 ./scripts/smoke_goal_first_app_server_real.sh`
- Resultado: `smoke_goal_first_app_server_preflight=ok`,
  `goal_backend=app_server_tmux`.

Smoke normal corto:

- Comando: smoke real normal con 50 polls, `KEEP_SMOKE_DIR=1`.
- `run_ref=run-spec-smoke-goal-first-req-smoke-goal-first-4bde3b2d0c8fc9f42c555b714a3521f0`.
- Resultado: no cerro accepted en 50 polls; no se cuenta como verde.
- Valor util: el estado de shutdown incluyo
  `evidence-ref-codex-app-server-turn-start-tool-output-policy-sent` y
  `evidence-ref-codex-app-server-turn-start-tool-output-policy-accepted`.
- Cleanup: sin `orquesta-server run`, sin `codex app-server` ni tmux
  `orquesta-goal-*` del smoke.
- Evidencia retenida:
  `/tmp/orquesta-smokes-codex/orquesta-goal-first-app-server.ojE5Us`.

Smoke alto consumo/checkpoint:

- Comando:
  `ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_REAL_CONFIRM=1 ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_CODEX_EXECUTION_CONFIRMED=1 ORQUESTA_GOAL_FIRST_SMOKE_POLLS=30 ORQUESTA_GOAL_FIRST_SMOKE_SLEEP_SECONDS=3 ORQUESTA_KEEP_SMOKE_DIR=1 ORQUESTA_SMOKE_PARENT=/tmp/orquesta-smokes-codex ./scripts/smoke_goal_first_checkpoint_only_high_consumption_real.sh`.
- `run_ref=run-spec-smoke-goal-first-bug088-req-smoke-goal-first-bug088-a93fa78e27925c5262c0b25f2445dc0f`.
- Resultado: `tool_output_policy_transport=accepted`,
  `smoke_goal_first_high_consumption_real=ok`,
  `bug088_path=second_artifact_or_partial_artifacts`,
  `recommended_action=review_partial_artifacts`,
  `tokens_used=8392`, checkpoint durable y
  `app_server_tmux_processes_alive=0`.
- Evidencia retenida:
  `/tmp/orquesta-smokes-codex/orquesta-goal-first-app-server.CZC8Ek`.

Lectura:

- El proveedor/app-server real actual acepta `toolOutputPolicy`; el residual ya
  no es transporte local ni fallback legacy.
- `BUG-079` sigue abierto para la prueba adversarial/larga de enforcement
  pre-tool: stdout gigante debe cortarse antes de quemar contexto, no solo
  sanearse en lectura posterior.

## Orquesta local 2026-07-09: harness adversarial BUG-079 y BUG-200

Se uso Orquesta para programar el siguiente paso de `BUG-079`:

- Ola: `codex-launch-director-wave`.
- `wave_ref=codex-bug079-adversarial-smoke-20260709`.
- Agentes: 1.
- Resultado: `codex_process_done_v0=completed`.

Cambios de harness:

- Nuevo wrapper opt-in:
  `scripts/smoke_goal_first_tool_output_policy_adversarial_real.sh`.
- Nuevo modo en `scripts/smoke_goal_first_app_server_real.sh`:
  `ORQUESTA_GOAL_FIRST_SMOKE_TOOL_OUTPUT_POLICY_ADVERSARIAL_MODE=1`.
- El modo exige `tool_output_policy_transport=accepted`.
- Falla si observa
  `evidence-ref-codex-app-server-thread-output-sanitized` o
  `codex_app_server_thread_read_response_too_large`.
- El prompt del smoke pide checkpoint temprano, crear y ejecutar una vez un
  probe que imprimiria `ORQUESTA_BUG079_STDOUT_PROBE:` + 200000 `X`, y escribir
  solo `probe_result.txt` antes del `ORQUESTA_GOAL_RESULT_V0`.

Verificacion local:

- `bash -n scripts/smoke_goal_first_app_server_real.sh scripts/smoke_goal_first_tool_output_policy_adversarial_real.sh`.
- `go test -count=1 ./cmd/orquesta-server -run 'TestSmokeGoalFirst'`.
- `git diff --check`.

Ejecucion real:

- Intento 1: 30 polls, evidencia
  `/tmp/orquesta-smokes-codex/orquesta-goal-first-app-server.YAf0mK`,
  `run_ref=run-spec-smoke-goal-first-bug079-tool-output-policy-req-smoke-goal-first-bug079-tool-output-policy-42cbaac6e57fce5425a51a63a`.
- Intento 2: 60 polls, evidencia
  `/tmp/orquesta-smokes-codex/orquesta-goal-first-app-server.6wB72G`,
  `run_ref=run-spec-smoke-goal-first-bug079-tool-output-policy-req-smoke-goal-first-bug079-tool-output-policy-5e97d87c7c9d9492c24b03907`.
- Ambos: `toolOutputPolicy accepted`, checkpoint temprano materializado,
  `goal_status=running`, `recommended_action=observe_later`, sin
  `thread-output-sanitized`, sin `thread_read_response_too_large`, cleanup sin
  procesos y state final `stopped/shutdown_ready=true`.

Lectura:

- `BUG-079` no se cierra: aun no hay prueba de enforcement pre-tool.
- Nuevo `BUG-ORQ-20260709-200` abierto: el harness adversarial existe pero el
  agente real no avanza de checkpoint a ejecucion del probe. Siguiente paso:
  hacer el probe mas determinista o bajar la prueba al nivel app-server/protocolo
  directo para no depender de que el agente decida ejecutar la salida gigante.

## Orquesta local 2026-07-09: BUG-200 reducido con programacion paralela

Se usaron dos olas Orquesta en paralelo, con write-sets separados:

- `codex-core-bug200-harness-20260709T141956Z`
  (`codex_process_done_v0=completed`):
  `scripts/smoke_goal_first_app_server_real.sh` y
  `cmd/orquesta-server/smoke_goal_first_scripts_v0_test.go`.
- `codex-core-bug079-protocol-20260709T141956Z`
  (`codex_process_done_v0=completed`):
  `modulos/orquesta-runtime-codex-appserver/codex_goal_app_server_migrated_v0_test.go`.

Cambios:

- El adversarial de BUG-079 ya no declara OK ni queda genericamente inconcluso
  si solo hay checkpoint: exige `probe_result.txt`, falla con
  `reason=bug200_probe_not_executed` cuando hay checkpoint sin probe, y con
  `reason=no_probe_result` cuando el resultado no documenta exit code,
  `probe_stdout.py` y `200000`.
- El app-server queda cubierto por test para no hacer fallback de
  `toolOutputPolicy` ante errores internos no-schema.

Verificado localmente:

- `bash -n scripts/smoke_goal_first_app_server_real.sh scripts/smoke_goal_first_tool_output_policy_adversarial_real.sh`
- `go test -count=1 ./cmd/orquesta-server -run 'TestSmokeGoalFirst'`
- `go test -count=1 ./modulos/orquesta-runtime-codex-appserver -run 'TestServerCodexAppServerGoalBackendV0TurnStart|TestServerCodexAppServerTurnStartParamsV0'`
- `git diff --check`

Nota operativa:

- La ola del harness no pudo ejecutar su `go test` dentro del home aislado de
  Codex porque Go intento descargar `golang.org/x/text v0.38.0` y la sandbox
  nego DNS/socket. El mismo test paso con el cache local normal.
- Nota MEJ-106: una copia limpia de `HEAD` y el worktree actual miden
  `env_vars_orquesta=512`; este corte no introduce nuevas `ORQUESTA_*` en Go.
  Para no bloquear la sincronizacion remota por un ratchet ya desfasado y
  mantener la deuda visible, se documenta la excepcion exacta requerida por el
  test: `env_vars_orquesta_allow_increase_to=512`.
- `BUG-079` sigue abierto: esto no prueba enforcement pre-tool del proveedor;
  solo evita falso OK/inconclusion del harness y refuerza el contrato local.

## Codex local 2026-07-09: revalidacion OPES fake lifecycle

Se aplico la regla de usar Orquesta como superficie para OPES. No habia servidor
Orquesta local vivo (`pgrep` no encontro `orquesta-server`); para no tocar OPES
productivo ni depender de remoto, se uso el smoke local con fake OPES y fake
Orquesta/goal-first integrado por el propio arnes.

Comandos ejecutados:

- `ORQUESTA_OPES_DERIVATIVES_FAKE_SERVER=1 ORQUESTA_OPES_DERIVATIVES_SMOKE_MODE=dry-run-once ./scripts/smoke_opes_derivatives_rest.sh`
- `ORQUESTA_OPES_DERIVATIVES_FAKE_SERVER=1 ORQUESTA_OPES_DERIVATIVES_EXECUTE=1 ORQUESTA_OPES_DERIVATIVES_SMOKE_MODE=run-until-finalize ORQUESTA_OPES_DERIVATIVES_TICK_SLEEP_SECONDS=0 ORQUESTA_OPES_BRIDGE_MAX_TICKS=40 ./scripts/smoke_opes_derivatives_rest.sh`

Resultado:

- `dry-run-once` selecciona `assemble_topic`, `seen=1`, `submitted=0`,
  `status=dry_run`.
- `run-until-finalize` completa 24/24 receipts hasta
  `finalize_temario_package`, con `goal_receipts_manifest_status=ok`,
  `run_until_status=completed` y `empty_after_final=true`.
- Manifest:
  `/tmp/opes-salidas/derivatives-rest-codex-local-finalize-20260709T154800Z/goal_receipts_manifest.json`.

Lectura:

- Esto valida el conector OPES local/fake y corrige la nota antigua de sandbox
  sin sockets loopback.
- No cierra `BUG-058/066/075`: siguen pendientes OPES temporal/preprod con
  proveedor real y prueba de ausencia de reescritura tardia con agente real.

## Orquesta local 2026-07-09: BUG-206 cierre operativo proyectado en DirectorStats

Se siguio el criterio de arreglo de dentro hacia afuera. El smoke no-OPES
cerraba bien el `OperationalPlanState` con
`operational-closure-succeeded`, pero la capa externa
`/api/v0/director/stats` podia seguir publicando
`closure.status=blocked` por una evidencia anterior de
`estado_vivo_entregado_parcial`.

Diagnostico:

- El nucleo operativo no estaba reabierto: `operational_plan_status=closed` y
  `operational_closure_reason=operational-closure-succeeded`.
- El bug estaba en observabilidad/composicion: `DirectorStats` derivaba el
  cierre desde el run y `estado-vivo`, pero no reconciliaba con el
  `OperationalPlanStateStore` ya cerrado.
- El estado parcial stale era blando. No debe ganar a un plan operativo cerrado;
  los conflictos duros de proceso vivo, bloqueo, conflicto o rework terminal si
  se conservan como veto.

Cambios:

- `orquesta.director.stats.v0` acepta ahora
  `OperationalPlanStateStore` inyectado.
- La proyeccion de PlanState cerrado se aplica despues de `estado-vivo` y marca
  `status=closed`, `closure.status=closed`, `closed=true` y progreso 100.
- No pisa bloqueos duros de `estado-vivo`: `proceso_vivo`, `conflicto`,
  `bloqueado` ni `terminal_rework`.
- El stack Codex cablea el store real hacia el executor MCP.

Evidencia:

- `go test -count=1 ./modulos/orquesta-mcp`
- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestBuildStackV0|Test.*DirectorStats|TestCodexStackV0DirectorStats|TestBuildStackFromEnvV0DirectorStats'`
- `go test -count=1 ./cmd/orquesta-server -run 'TestBuildStackFromEnvV0DirectorStatsExponeProgressSourceConfiguradoV0|TestBuildStackFromEnvV0|TestStackWiring'`
- `ORQUESTA_EXTERNAL_WORK_LEGACY_DIRECTOR_LOOP=1 ./scripts/smoke_external_domain_fake_real.sh`
  con `external_cycle_closure_status=closed`,
  `external_cycle_closed=true`, `operational_plan_status=closed` y
  `closure_status=closed`.
- `ORQUESTA_EXTERNAL_WORK_LEGACY_DIRECTOR_LOOP=1 ./scripts/smoke_external_domain_non_opes_real.sh`
  con app externa temporal, `external_app_artifact_count=2`,
  `operational_plan_status=closed`, review aceptada y `opes_touched=false`.

Residual:

- Esto cierra el falso bloqueo local de `DirectorStats`, no los residuales de
  proveedor real, OPES temporal/preprod ni despliegue remoto.
- Si `estado-vivo` detecta conflicto duro real, el cierre sigue bloqueado de
  forma intencionada.

## Orquesta local 2026-07-09: BUG-200 cerrado como harness local de BUG-079

Se siguio arreglando de dentro hacia afuera. El residual de `BUG-079` no estaba
ya en el transporte local de `toolOutputPolicy`, sino en demostrar enforcement
real del proveedor antes de que una herramienta emita stdout gigante. El bug
local pendiente era `BUG-ORQ-20260709-200`: el smoke adversarial podia quedar
inconcluso si el agente solo escribia checkpoint y no ejecutaba el probe.

Cambios:

- `scripts/smoke_goal_first_app_server_real.sh` incorpora
  `SMOKE_GOAL_FIRST_BUG079_GUARD_SELFTEST=1`, un self-test local que
  no arranca servidor ni Codex.
- El self-test crea un fixture checkpoint-only y exige fallo con
  `reason=bug200_probe_not_executed`.
- El fixture valido solo pasa con `probe_result.txt` que declare exit code,
  `probe_stdout.py`, `200000`, sentinel `BUG079_STDOUT_PROBE` y
  `executions=1`.
- El test Go de scripts ejecuta ese self-test, ya no se limita a buscar strings.
- El runtime app-server anade cobertura para `toolOutputPolicy` aceptada seguida
  de `thread/read` gigante: Orquesta bloquea con
  `codex_app_server_thread_read_response_too_large` y evidencia especifica.

Evidencia:

- `SMOKE_GOAL_FIRST_BUG079_GUARD_SELFTEST=1
  scripts/smoke_goal_first_app_server_real.sh`
- `bash -n scripts/smoke_goal_first_app_server_real.sh
  scripts/smoke_goal_first_tool_output_policy_adversarial_real.sh`
- `go test -count=1 ./modulos/orquesta-runtime-codex-appserver -run
  'TestServerCodexAppServerGoalBackendV0(PolicyAceptadaYThreadReadGiganteBloquea|TurnStartToolOutputPolicy|ObservaThreadReadGigante|LanzaThreadGoalYTurnMigrado)|TestCodexAppServer(WebSocketThreadReadResponseBudget|CommandProtocolThreadReadResponseBudget)'`
- `go test -count=1 ./cmd/orquesta-server -run
  'TestSmokeGoalFirstToolOutputPolicyAdversarial|TestSmokeGoalFirstAppServerRealExponeToolOutputPolicyTransport'`

Lectura:

- `BUG-ORQ-20260709-200` queda cerrado como falso inconcluso/harness local.
- `BUG-ORQ-20260701-079` sigue abierto solo para proveedor/app-server real:
  falta ejecutar o construir la prueba E2E que fuerce stdout gigante y confirme
  si el proveedor aplica el cap pre-tool o si hay que corregir esa frontera.

## Orquesta local 2026-07-09: BUG-079 smoke real adversarial pasa con forced-stop

Se ejecuto el smoke real adversarial tras corregir el contrato de write-set del
probe. El fallo anterior no era que el proveedor no ejecutara el probe: lo
ejecutaba, pero el arnes lo habia pedido fuera del write-set autorizado
(`generated-apps/bug079-tool-output-policy`). Orquesta solo observaba el
checkpoint dentro de
`generated-apps/smoke-goal-first-bug079-tool-output-policy`, por eso el goal
seguia `running`.

Cambios:

- El checkpoint BUG-079 y `probe_stdout.py`/`probe_result.txt` viven ahora bajo
  `generated-apps/smoke-goal-first-bug079-tool-output-policy`.
- El guard falla con `reason=bug079_probe_out_of_write_set` si detecta el probe
  en la ruta legacy fuera del write-set.
- El modo adversarial ya no exige cierre completo de app de negocio: si
  `toolOutputPolicy` fue `accepted`, el probe esta ejecutado dentro del
  write-set y no hay senal de salida gigante, marca
  `smoke_goal_first_tool_output_policy_adversarial_real=ok` y corta el goal con
  `runs/control forced=true`, verificando que queda `blocked` y sin proceso
  app-server vivo.

Ejecucion real:

- `ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_REAL_CONFIRM=1
  ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_CODEX_EXECUTION_CONFIRMED=1
  ORQUESTA_KEEP_SMOKE_DIR=1 ORQUESTA_GOAL_FIRST_SMOKE_POLLS=60
  ORQUESTA_GOAL_FIRST_SMOKE_SLEEP_SECONDS=2
  ./scripts/smoke_goal_first_tool_output_policy_adversarial_real.sh`
- `run_ref=run-spec-smoke-goal-first-bug079-tool-output-policy-req-smoke-goal-first-bug079-tool-output-policy-e4e76c6dacea00864e51d01ef`
- `external_goal_ref=019f47b3-3b98-7e30-8fc1-6704b4f5bdfb`
- Evidencia retenida:
  `/tmp/orquesta-goal-first-app-server.ZHXRWf`

Resultado:

- `tool_output_policy_transport=accepted`
- `bug079_probe_path=generated-apps/smoke-goal-first-bug079-tool-output-policy/bug079-tool-output-policy/probe_result.txt`
- `bug079_probe_result=executed`
- `smoke_goal_first_tool_output_policy_adversarial_real=ok`
- `run_control_status=stopped`
- `run_control_goal_status_after=blocked`
- `observe_after_forced_stop_goal_status=blocked`
- `autoprogramming_status_after_forced_stop_not_running=true`
- `app_server_tmux_processes_alive=0`

Lectura:

- Esto cierra el smoke operativo real de BUG-079 y demuestra que el proveedor
  real acepto `toolOutputPolicy`, ejecuto el probe dentro del write-set y no
  expuso una salida gigante a Orquesta. `rg -a 'X{100,}'` sobre runtime,
  proyecto, JSON y logs del smoke no encontro salida cruda.
- No cierra el residual teorico de cap duro si una herramienta vuelca stdout
  crudo sin redireccion: en esta ejecucion el agente ejecuto
  `probe_stdout.py > /dev/null`, que es comportamiento seguro pero no una prueba
  de truncado forzado por el proveedor.
- Nuevo residual `BUG-ORQ-20260709-207`: el log del app-server en el forced-stop
  contiene `Node.js[...] ResetStdio` / `Assertion failed`, aunque el control
  termino `stopped`, el goal quedo `blocked` replanificable y no quedaron
  procesos vivos.

## Orquesta local 2026-07-09: cierre BUG-207 forced-stop app-server sin ResetStdio

Se corrigio el residual `BUG-ORQ-20260709-207` desde dentro hacia afuera, en el
adaptador runtime `app_server_tmux`, sin tocar core puro ni conectores.

Causa acotada:

- `codex app-server --listen unix://...` se arrancaba dentro de tmux heredando
  stdin del PTY de la sesion.
- El control funcional ya era correcto: `runs/control forced=true` dejaba el
  goal `blocked`, `autoprogramming/status` sin running y cero procesos.
- El problema era el teardown ruidoso del proveedor: al cerrar/matar tmux, Node
  podia registrar `ResetStdio` / `Assertion failed`.

Intento descartado:

- Se probo desacoplar stdin con `< /dev/null`.
- El proveedor no lo tolero: el smoke dedicado fallo al arrancar con
  `codex_app_server_tmux_session_exited` en
  `/tmp/orquesta-goal-first-app-server.66gWvN`.
- Lectura: el app-server necesita stdin abierto, aunque no deba depender del
  PTY de tmux.

Cambio aplicado:

- `tmuxStartSessionV0` crea una FIFO propia `stdin.pipe` junto al socket y
  alimenta stdin del app-server desde esa FIFO, no desde el PTY de tmux.
- `ShutdownForcedStopV0` conserva una fase cooperativa: detecta procesos propios
  por socket/runtime, envia `SIGTERM`, espera corto, reobserva, y solo cae a
  `kill-session` si la sesion sigue viva.
- El shutdown limpia socket, `stdin.pipe` y `owner.json`.

Evidencia real:

- `scripts/smoke_goal_first_forced_stop_backend_real.sh` en modo real opt-in
  con directorio retenido.
- `smoke_goal_first_forced_stop_backend_real=ok`
- `run_ref=run-spec-smoke-goal-first-bug088-req-smoke-goal-first-bug088-b69a60d252ccb0cdd3440cad4adda2bb`
- `external_goal_ref=019f47c5-99e2-7670-a175-2c2fad844a42`
- Evidencia retenida: `/tmp/orquesta-goal-first-app-server.oj0Aat`
- Resultado: `run_control_status=stopped`,
  `run_control_goal_status_after=blocked`,
  `autoprogramming_status_after_forced_stop_not_running=true`,
  `app_server_tmux_processes_alive=0`.
- `rg -a 'ResetStdio|Assertion failed'` sobre `runtime/goal-srv` y logs del
  smoke no encontro coincidencias.
- En `runtime/goal-srv` solo quedo el log del app-server; no quedo FIFO
  residual.

Tests:

- `go test -count=1 ./modulos/orquesta-runtime-codex-appserver`
- `git diff --check`

Lectura:

- `BUG-ORQ-20260709-207` queda cerrado.
- `BUG-ORQ-20260701-079` no queda cerrado por este cambio: el residual vivo
  sigue siendo el cap duro pre-tool si una herramienta vuelca stdout crudo
  dentro del proveedor/runtime. Orquesta ya transporta `toolOutputPolicy`,
  sanea/corta `thread/read` y bloquea con evidencia si la salida gigante llega a
  observacion.

## Orquesta local 2026-07-09: BUG-079 baja a protocolo determinista sin LLM

Se reforzo la verificacion local de `BUG-079`/`BUG-200` siguiendo la TAREA-E4:
la prueba adversarial ya no depende solo de que un agente real decida ejecutar
el probe stdout gigante.

Cambio:

- Se anadio
  `TestServerCodexAppServerGoalBackendV0ToolOutputPolicyYThreadReadGigantePorWebSocketDeterministaV0`.
- La prueba levanta un servidor WebSocket Unix falso compatible con el protocolo
  de `codex app-server`.
- `StartCodexGoalV0` usa el protocolo real: `thread/start`,
  `thread/goal/set` y `turn/start`.
- El test inspecciona el JSON recibido en `turn/start` y exige
  `toolOutputPolicy` con `maxTextBytes`, `threadReadMaxBytes` y
  `requireBoundedCommands`.
- Despues, en `ObserveCodexGoalV0`, el servidor falso inyecta un frame gigante
  de `thread/read`; Orquesta corta antes de decodificar el payload completo y
  devuelve `GoalStatus=blocked` con
  `codex_app_server_thread_read_response_too_large` y evidencia especifica.

Evidencia:

- `go test -count=1 ./modulos/orquesta-runtime-codex-appserver -run
  'TestServerCodexAppServerGoalBackendV0ToolOutputPolicyYThreadReadGigantePorWebSocketDeterministaV0'`

Lectura:

- Queda cerrada la cobertura local de protocolo/ingesta sin LLM para la familia
  TAREA-E4 item 3.
- No se declara cerrado `BUG-ORQ-20260701-079` entero: Orquesta no puede probar
  desde este test un cap duro dentro del proveedor antes de que el stdout crudo
  entre en el historial del app-server. Ese residual sigue siendo frontera
  proveedor/runtime y queda documentado como tal.

## Orquesta local 2026-07-09: shutdown goal-first validado dentro hacia afuera

Se revalido el eje interior de `BUG-ORQ-20260701-065` /
`BUG-ORQ-20260704-165` antes de tocar OPES, remoto, Hermes o Telegram.

Ejecucion real:

```bash
ORQUESTA_KEEP_SMOKE_DIR=1 \
ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_REAL_CONFIRM=1 \
ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_CODEX_EXECUTION_CONFIRMED=1 \
./scripts/smoke_goal_first_shutdown_coordination_real.sh
```

Resultado:

- `smoke_goal_first_shutdown_coordination_real=ok`.
- `run_ref=run-spec-smoke-goal-first-bug088-req-smoke-goal-first-bug088-12dc1a2f9e726db1738e5b860b2227e1`.
- `external_goal_ref=019f47d2-332b-7840-ae2f-94d096a8b55f`.
- `autoprogramming_status_before_shutdown_visible=true`.
- `/api/v0/server/shutdown` devolvio `status=ready`,
  `shutdown_ready=true`, `runs_requested=1`, `runs_stopped=1`.
- `run_control_statuses=stopped` y
  `shutdown_coordination_all_runs_stopped=true`.
- `goal_actions[0].action_taken=cleanup_completed`.
- `app_server_tmux_processes_alive=0`.

Evidencia:

- Smoke crudo inicial: `/tmp/orquesta-goal-first-app-server.3sGyca`.
- Copia saneada para revision:
  `/tmp/orquesta-smokes-codex/orquesta-goal-first-shutdown-coordination-20260709-local`.
- Tras copiar la evidencia, se limpio del smoke crudo `runtime/goal-srv/codex-home`
  y `bin` para no dejar credenciales, caches ni binario temporal retenidos.
- No quedaron procesos `orquesta-server run`, `codex app-server`, socket
  `g-78cec46f183f809b.sock` ni tmux `orquesta-goal-78cec46f183f809b`.

Pruebas:

- `bash -n scripts/smoke_goal_first_app_server_real.sh scripts/smoke_goal_first_shutdown_coordination_real.sh`
- `go test -count=1 ./cmd/orquesta-server -run 'Test(ServerAppHTTPGoalFirstShutdownCoordinaControlPendienteFueraDeCola|SmokeGoalFirstShutdownCoordinationReal|NormalizeServerShutdownClientResultV0GoalActionCompletedOcultaAccionStale)'`
- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestStackShutdown(V0ForzadoCoordinaGoalFirstTerminalFueraDeCola|RunControlWriterV0ForcedStopMarcaGoalTerminalReplanificable)'`
- `go test -count=1 ./modulos/orquesta-runtime-codex-appserver -run 'TestCodexAppServerTmuxBackendV0(EnsureShutdownCleanupMigrado|ReadActiveShutdownWorkIgnoraEstadoDegradadoSinResiduoVivo)'`
- `scripts/orquesta_metricas_deuda.sh --json` ->
  `{"env_vars_orquesta":513,"endpoints_status":16,"interfaces_estado":67,"modulos_director":18}`.

Lectura:

- El nucleo/runtime local goal-first coordina shutdown de un goal fuera de cola
  con backend `app_server_tmux` real: el status previo crea visibilidad, shutdown
  amplio solicita control, `RunControl` termina `stopped`, el backend se limpia
  y el proceso servidor sale.
- No se cambia codigo en este corte porque la brecha local no reaparece.
- No se declara cerrado `BUG-065` global: el residual que queda es externo o de
  entorno, concretamente repetir evidencia en remoto/stale/proveedor lento y
  comprobar cortes manuales fuera de esta maquina.

## Orquesta local 2026-07-09: E3 HTTP/web entra en inventario de contratos

Se cerro el residual local de E3 que habia quedado tras el inventario MCP:
las rutas HTTP/web focales podian existir en `PublicRouteManifestV0` sin decir
que contrato canonico exponian.

Cambio:

- `RouteManifestEntryV0` gana `ContractRefs`.
- `route_manifest_entries_v0.go` declara contratos para:
  - `/nueva-app`: `nueva_app.solicitar.v0`, `nueva_app.wizard.v0`,
    `nueva_app.wizard_bot.v0`.
  - `/api/v0/apps/spec`: `nueva_app.solicitar.v0`.
  - `/api/v0/apps/intake/guided-turn`: `nueva_app.wizard.v0`.
  - `/api/v0/apps/intake/wizard-bot`: `nueva_app.wizard_bot.v0`.
  - `/autoprogramming`: `autoprogramming.prepare_run.v0` y
    `autoprogramming.status.v0`.
  - `/api/v0/autoprogramming/prepare-run`:
    `autoprogramming.prepare_run.v0`.
  - `/api/v0/autoprogramming/status`: `autoprogramming.status.v0`.
  - `/api/v0/director/human-work/review-plan`:
    `operator_director.review_plan.v0`.
- `ServerRouteResourceV0` propaga `contract_refs` en el discovery publico del
  servidor.

Guards:

- `TestPublicRouteManifestV0DeclaraContratosE3InternosV0`.
- `TestServerResourcesRouteManifestIncluyeDiscoveryOPESV0` comprueba que el
  discovery conserva refs de wizard-bot, prepare-run y status.

Lectura:

- Esto no compara campos web y MCP 1:1: web usa formularios/viewmodels y MCP
  usa envelopes canonicos. Forzar igualdad literal ahora produciria falsos
  rojos y acoplaria mal las capas.
- El siguiente paso, si se reabre E3 por campos web, debe ser un inventario de
  aliases/nesting explicito por contrato, no reflexion ingenua.

## Orquesta local 2026-07-09: E6 primera tabla de guards de scripts

Se redujo el residual de mantenimiento E6 sin reescribir el archivo completo de
guards.

Cambio:

- En `cmd/orquesta-server/smoke_goal_first_script_guard_v0_test.go` se anade
  `scriptContractGuardV0`.
- `TestSmokeScriptContractsConsolidadosV0` recorre una tabla
  `script -> wants/forbids`.
- La tabla cubre:
  - `scripts/smoke_goal_first_app_server_real.sh`
  - `scripts/smoke_goal_first_checkpoint_only_high_consumption_real.sh`
  - `scripts/smoke_goal_first_forced_stop_backend_real.sh`
  - `scripts/smoke_goal_first_shutdown_coordination_real.sh`
  - `scripts/smoke_goal_first_claude_process_server_real.sh`
  - `scripts/lib/smoke_common.sh`
  - `scripts/orquesta_server_deploy.sh`

Lectura:

- Se dejan fuera de la tabla los guards `WalkDir` y los checks de orden porque
  son reglas globales de flota o de precedencia, no contratos simples de un
  script exacto. Moverlos ahora haria el fallo menos claro.
- E6 queda reducido como mantenimiento local. No desbloquea ni bloquea OPES,
  remoto, Hermes ni Telegram.

## Orquesta local 2026-07-09: cierre local dentro-fuera E3/notificaciones/shutdown

Subagentes read-only encontraron tres falsos verdes locales despues del
checkpoint dentro-fuera. Se corrigieron antes de pasar a OPES/remoto.

Cambios:

- `BUG-ORQ-20260709-208`: `operator_director.review_plan.v0` se incorpora al
  inventario MCP interno. El contrato ya estaba en HTTP, pero la tool MCP
  `orquesta.director.human_work.review_plan.v0` no exigia paridad DTO ->
  `input_schema`.
- `BUG-ORQ-20260709-209`: `operator_notifications.v0` libera la clave de
  dedupe si el sender falla, para que un error temporal de Telegram/Hermes no
  suprima el reintento.
- `BUG-ORQ-20260709-210`: `orquesta-server-shutdown` gana identidad opcional de
  active work; los wrappers goal-first de `cmd` propagan la identidad del
  backend y `orquesta-app-codex-stack` deduplica readers/cleaners antes de
  invocarlos. Evita limpiar varias veces el mismo backend cuando launcher,
  rework y observer envuelven el mismo `app_server_tmux`.

Lectura:

- No se mete HTTP gateway dentro de MCP ni `cmd` dentro del stack. La identidad
  vive en el puerto neutral de shutdown y es opcional.
- Estos cierres son locales. No sustituyen los smokes remotos de Telegram,
  proveedor lento ni deploy real.

Evidencia focal:

- `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-http-gateway -run 'TestMCPInternalContractSurfaceInventory|TestPublicRouteManifestV0DeclaraContratosE3InternosV0'`
- `go test -count=1 ./modulos/orquesta-operator-notifications -run 'TestOperatorNotificationV0'`
- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestStackShutdownActiveWorkCleanerV0DeduplicaPuertosConMismaIdentidadV0'`
- `go test -count=1 ./cmd/orquesta-server -run 'TestServerGoalWorkPortsFromBackendV0PropaganActiveShutdownWorkV0'`
- `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-http-gateway ./modulos/orquesta-operator-notifications ./modulos/orquesta-server-shutdown ./modulos/orquesta-app-codex-stack ./modulos/orquesta-runtime-codex-appserver ./cmd/orquesta-server`
- `git diff --check`
- `go test -count=1 ./...`
- `go build ./...`

## Orquesta local 2026-07-09: revalidacion OPES fake-residente

Tras cerrar los falsos verdes locales de nucleo/runtime, se paso al siguiente
anillo: conectores OPES.

Ejecucion:

- `timeout 240 env ORQUESTA_KEEP_SMOKE_DIR=1 scripts/smoke_opes_lifecycle_real.sh`

Resultado:

- `status=passed`
- 24/24 `work_kind` cubiertos hasta `finalize_temario_package`.
- `finalpkg_dry_run=false`.
- `finalpkg_run_ref=run-ref-opes-a1-t002-finalpkg-20260612`.
- `settlement_status=settled_final`.
- `settlement_reason=final_package_closure_evidence_complete`.
- `no_residual_processes=true`.
- Evidencia retenida:
  `/tmp/orquesta-opes-lifecycle-real-20260709T172932Z/out/opes_lifecycle_result.json`.

Revision paralela:

- Subagente read-only sobre `BUG-058/066/075` no encontro bug local accionable.
- Confirmo que los residuales vigentes son de campo: OPES temporal/preprod,
  proveedor real/remoto y prueba con agente real de no reescritura tardia.

Lectura:

- En local, el conector OPES queda validado en fake-residente. No se toca OPES
  productivo ni se declara cerrado el residual externo.

## Orquesta local 2026-07-09: cierre E1/E2 de borde operativo

Siguiendo la regla dentro -> fuera, despues de nucleo local, runtime local y
OPES fake-residente se revisaron los bordes operativos E1/E2 con subagentes
read-only.

Hallazgos:

- E1 deploy atomico tenia falsos verdes: `ctl status` podia fallar sin bloquear,
  URLs de readiness configuradas podian no responder sin bloquear, y si ninguna
  superficie exponia `sha256` del binario vivo el deploy seguia como `ok`.
  Tambien habia fallos por `set -e` sin recibo durable.
- E2 nightly guardaba JSON/log, pero no registraba ref git ni notificaba
  Telegram. Si Telegram estaba activado pero incompleto, no habia fallo terminal
  visible.

Cambios:

- `scripts/orquesta_server_deploy.sh` registra fase, usa recibo
  `deploy_unhandled_failure` para roturas no controladas, exige `ctl status`,
  bloquea readiness configurada inalcanzable y requiere identidad runtime
  `binary_sha256`, `runtime_binary_sha256` u `orquesta_server_sha256`.
- `scripts/test_orquesta_server_deploy.sh` comprueba que el binario se instala
  realmente y cubre `deploy_status_failed` y
  `deploy_runtime_identity_missing`.
- `scripts/orquesta_smoke_nightly.sh` anade `git.ref`, `git.branch`,
  `git.dirty` al resultado JSON y envia notificacion terminal por Bot API si
  `telegram_operator.enabled=true` en `orquesta.config.json`. El token/chat
  siguen saliendo solo de config canonica; no se reintroducen envs Telegram de
  secreto. Si Telegram activado esta roto, el cierre pasa a
  `notification_failed`.
- `scripts/test_orquesta_smoke_nightly.sh` cubre Bot API falso, envio correcto,
  config Telegram incompleta y presencia de git/ref en JSON.

Lectura:

- No se toca core/workflow/domain-work. Deploy y nightly son composicion
  operativa exterior.
- `ok` en preflight nightly sigue siendo solo preflight; para cierre real de
  campo hace falta `mode=real`, ref git, despliegue del arbol correcto y mensaje
  Telegram recibido.
- Queda residual externo: ejecutar E1/E2 en el host real con config canonica y
  confirmar Telegram real. La parte local ya no debe dar falso verde por
  identidad runtime o notificacion terminal.

Evidencia:

- `bash -n scripts/orquesta_server_deploy.sh scripts/test_orquesta_server_deploy.sh scripts/orquesta_smoke_nightly.sh scripts/test_orquesta_smoke_nightly.sh`
- `bash scripts/test_orquesta_server_deploy.sh`
- `bash scripts/test_orquesta_smoke_nightly.sh`
- `GOFLAGS=-buildvcs=false go test -count=1 ./modulos/orquesta-operator-notifications ./modulos/orquesta-operator-telegram ./cmd/orquesta-server -run 'TestOperatorNotification|TestTelegram(BotAPI|Operator)|TestAdapterV0DespachaMensajeAlCanalDirector'`
