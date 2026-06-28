# Auditoría de código y contratos Orquesta

Fecha: 2026-06-27.
Revisor: agente de revisión (no programador).
Para: agente que programa Orquesta (aquí solo se documentan los fallos; no se ha
modificado código).

## Alcance ejecutado

Auditoría completa del repositorio, no solo de un contrato concreto:

- `go build ./...`
- `go vet ./...`
- `staticcheck ./...` (todas las categorías SA/ST/S/U)
- `govulncheck ./...`
- `go test ./...` (toda la batería)
- `go test -race ./...` en los paquetes con goroutines (mcp, run-supervisor,
  core-concurrency, core-leases, outbox-dispatch, server)
- Revisión manual de coherencia `modulos/*/docs/contratos.md` ↔ código en las
  zonas ligadas a las incidencias OPES (supervise, cola, outbox, dispatch,
  drain/recovery, runs stale).

## Baseline

- `go build`: **OK**.
- `go test ./...`: **85 paquetes OK / 0 FAIL** (sin `-race`).
- `go vet`: 1 categoría de hallazgo (copylocks → Hallazgo 1).
- `staticcheck`: 138× U1000 (código muerto) + varios SA/ST/S (detallados abajo).
- `govulncheck`: 16 vulnerabilidades de stdlib (toolchain) → Hallazgo 3.
- `go test -race`: 3 tests de `orquesta-server` fallan por DATA RACE → Hallazgo 2.

Orden de prioridad sugerido: **H7 (cableado goal-first / observador, liga con las
incidencias OPES) → H2 (race) → H1 (copylocks) → H3 (vulns) → H4 (intake loop) →
H5 grupo B (validación muerta) → limpieza H5 grupo A / H6**.

---

## Hallazgo 1 — BUG LATENTE: copia por valor de `sync.Mutex` y `strings.Builder`

Módulo: `orquesta-runtime-required-test`. Confirmado por `go vet` (8 avisos
copylocks).

`outputBufferV0` (`output_buffer_v0.go:8`) contiene `sync.Mutex` **y**
`strings.Builder`, ambos prohibidos de copiar por valor (copiar un
`strings.Builder` ya usado provoca `panic: strings: illegal use of non-zero
Builder copied by value`). El código lo pasa/devuelve por valor en:

- `local_command_executor_v0.go:200` — `runLocalCommandV0` hace `return *output`.
- `local_command_executor_v0.go:245` — `localCommandOutputWithDiagnosticV0(output outputBufferV0) outputBufferV0` (valor de entrada y de salida).
- `local_command_executor_v0.go:77,82,84,109` — llamadas asociadas.
- `output_artifact_v0.go:21` — `writeOutputArtifactV0(..., output outputBufferV0, ...)`.

Hoy no rompe porque las copias solo se **leen** después (`String()`,
`Truncated()`), nunca se vuelve a escribir en ellas. Es latente: cualquier
`Write` sobre una copia, o concurrencia, lo activa.

Corrección: operar siempre con `*outputBufferV0` (devolver `output` sin
desreferenciar; firmas con puntero). Validar con
`go vet ./modulos/orquesta-runtime-required-test/...` limpio.

---

## Hallazgo 2 — BUG REAL: data races en 3 tests de `orquesta-server`

Confirmado por `go test -race ./modulos/orquesta-server/`. Fallan:

- `TestRuntimeV0SupervisorAsyncCoalesceaUnTickPendienteV0`
- `TestRuntimeV0ResidentDirectorAsyncCoalesceaUnTickPendienteV0`
- `TestRuntimeV0IdleSelfImprovementGoalFirstLanzaGoalSpecV0`

Causa: el test lee `store.last` (p.ej. `supervisor_loop_async_idle_v0_test.go:47`)
mientras la goroutine asíncrona de producción lo escribe vía
`runSupervisorTickAsyncV0 → runAsyncWorkV0 → persistStateTransitionV0 →
saveStateV0 → SaveServerStateV0`. El test sincroniza con
`supervisor.waitDone(t)` + `time.Sleep(10 * time.Millisecond)`
(`supervisor_loop_async_idle_v0_test.go:45-46`), pero la persistencia ocurre en
el `defer` **después** de `waitDone`, así que el sleep no garantiza
happens-before → carrera y flakiness.

Detalle importante: el runtime **ya tiene** la primitiva correcta
`waitAsyncWorkV0(ctx)` (`runtime_async_work_v0.go:38`), que espera el
`sync.WaitGroup` y sí establece happens-before. Los tests no la usan.

Corrección: en esos tests, sustituir `time.Sleep(...)` por
`runtime.waitAsyncWorkV0(context.Background())` antes de leer `store.last`.
Validar con `go test -race ./modulos/orquesta-server/...` en verde.

Acción recomendada de proceso: añadir `go test -race` a CI para que estas
carreras no vuelvan a pasar desapercibidas (hoy el suite sin `-race` está verde
y las oculta).

---

## Hallazgo 3 — Vulnerabilidades de stdlib (toolchain Go desactualizado)

`govulncheck` reporta **16 vulnerabilidades de la stdlib** alcanzables desde el
código, por compilar con `go1.25.5`. Ejemplos:

- `GO-2026-4341` (memory exhaustion en `net/url.ParseQuery`) — alcanzado desde
  `orquesta-web/http_resource_limits_v0.go:35` (`ParseForm`) y
  `public_params_policy_v0.go:30` (`URL.Query`). Corregido en `go1.25.6`.
- `GO-2026-4340` y `GO-2026-4337` (`crypto/tls`) — alcanzados desde servidores
  HTTP y el sidecar de privacidad. Corregidos en `go1.25.6`/`go1.25.7`.

Corrección: subir el toolchain (≥ `go1.25.7`) y revisar las 3 vulnerabilidades
adicionales en paquetes importados (p.ej. `golang.org/x/text v0.29.0` →
`go get golang.org/x/text@latest`). Recompilar y reejecutar `govulncheck ./...`.

---

## Hallazgo 4 — REVISAR DISEÑO: el intake de artefactos solo procesa el primer fichero

`modulos/orquesta-app-codex-stack/domain_work_delivery_artifact_intake_v0.go:28-39`
(staticcheck `SA4004`: "the surrounding loop is unconditionally terminated").

```go
for _, file := range ack.Files {
    path, ok := safeDomainWorkDeliveryFilePathV0(...)
    if !ok { return ..., error }
    return readDomainWorkDeliveryArtifactFileV0(path, string(file), artifactType)
}
```

El `for` **siempre retorna en la primera iteración**, así que solo se lee
`ack.Files[0]` y no se filtra por `artifactType`. Si por contrato un ACK puede
declarar varios ficheros y hay que elegir el que corresponde al `artifactType`,
esto es un bug (ignora el resto). Si por contrato es siempre un único fichero,
conviene reescribir como `if len(ack.Files) > 0 { ... }` para dejar la intención
explícita. **Confirmar contra el contrato de `DomainWork`/ACK** antes de cambiar.

---

## Hallazgo 5 — Código muerto (PARA BORRAR / PARA CABLEAR)

`staticcheck` detecta **138 símbolos U1000 sin uso** (117 en producción, 21 en
tests). El usuario pide marcar para borrar todo lo que carezca de sentido, pero
hay que distinguir dos grupos: lo realmente obsoleto (borrar) y la
**validación/seguridad muerta** que probablemente debía estar cableada (revisar
antes de borrar; borrarla elimina protecciones previstas).

Reproducir la lista completa: `staticcheck ./... | grep U1000`.

### Grupo A — BORRAR (obsoleto, sin sentido conservar)

Restos del viejo loop forzado / idle-self-improvement ya superado por goal-first
(ver Hallazgo 7) y helpers sueltos:

- `orquesta-server/supervisor_idle_prepare_v0.go:9` `idleSelfImprovementRequestFallbackV0`,
  `supervisor_idle_request_v0.go:71` `normalizeIdleSelfImprovementRequestsV0`,
  `:107` `idleSelfImprovementCooldownBlocksV0`,
  `supervisor_idle_schedule_v0.go:11` `maybeScheduleIdleSelfImprovementV0`,
  `runtime_v0.go:184` `(*RuntimeV0).persistStateV0`.
- `cmd/orquesta-server`: `idle_self_improvement_ack_correlation_v0.go:146`,
  `idle_self_improvement_backlog_request_helpers_v0.go:42`,
  `idle_self_improvement_backlog_state_v0.go:143`,
  `external_bridge_lifecycle_helpers_v0.go:114` `externalBridgeLoopContextV0`,
  `server_env_autonomy_v0.go:34`, `codex_wave_*`, `config.go:258`,
  `public_error_catalog_v0.go:18`, `mcp_real_*`, `opes_bridge_*`.
- Fingerprints legacy: `orquesta-domain-work-{file,memory,sql}/fingerprint_v0.go`
  (`*RequestFingerprintInputV0`, `*RequestLegacyFingerprintV0`).
- `orquesta-app-codex-stack/drain_attempt_v0.go:15` `drainRunAttemptV0` y `:298`
  `continueDrainRunAfterExternalV0` (métodos de drain ya no llamados),
  `autoprogramming_bridge_continue_v0.go:60`,
  `autoprogramming_prepare_run_retry_v0.go:75`, `spec_packet_v0.go:41`.
- CLI bootstrap-appspec sin uso: `orquesta-cli/bootstrap_appspec_client_v0.go` y
  `bootstrap_appspec_client_helpers_v0.go` (tipo + ~10 funciones), más
  `command_flags_v0.go:64`, `transport_rest_response_v0.go:13`,
  `transport_rest_v0.go:144`.
- `cmd/orquesta-guardian/guardian_files_v0.go` (`copyFileAtomicV0`,
  `envBoolOrDefaultV0`, `envDurationOrDefaultV0`, `envIntOrDefaultV0`,
  `envInt64OrDefaultV0`) y `guardian_command_io_v0.go:72,91`.
- `orquesta-web/director_stats_projection_v0.go:3,9`,
  `orquesta-observability/*` (`containsAnyV0`, `workspaceTimelineTimeWindowPtrV0`),
  `orquesta-director-scheduler/*` (`schedulerNeedsDirectorPlanV0`, etc.),
  `orquesta-runtime-worktree/control_paths_v0.go:65`,
  `orquesta-runtime/runtime_launch_request_helpers_v0.go:204` `containsParentTraversal`,
  `orquesta-autoprogramming/strings_v0.go:25`,
  `orquesta-persistence/outbox_ledger_*` (`dispatchIssuesFromOutboxLedgerV0`,
  `forbiddenOutboxLedgerTermsV0`).
- Helpers de tests sin uso (21 símbolos en `*_test.go`): borrar con el test que
  los dejó huérfanos.

### Grupo B — REVISAR ANTES DE BORRAR (validación/seguridad muerta = posible bug de cableado)

Estas funciones implementan **políticas que parecen necesarias y no se ejecutan**.
Antes de borrar, decidir si deben cablearse (especialmente las de ACK, dado que
las incidencias OPES incluyen ACK no escritos / rails pendientes / detalle local
prohibido):

- **Política de ACK Codex (toda dead):**
  `orquesta-runtime-codex/codex_ack_local_path_policy_v0.go` (7 funcs +
  `codexAckLocalProductForbiddenFragmentsV0`),
  `codex_ack_path_policy_v0.go:196,216,237` (`codexAckMissingWriteSetV0`,
  `codexAckPathMatchesWriteSetEntryV0`, `codexAckPathGlobstarMatchV0`),
  `codex_ack_pending_rail_refs_v0.go` (8 símbolos),
  `codex_ack_strict_validation_v0.go:159` `codexAckPathAllowedByWriteSetV0`,
  `codex_ack_text_policy_v0.go` (4 funcs),
  `codex_delivery_observation_v0.go:295,314,320`.
  → Si el write-set / rails pendientes / detalle local del ACK **debían**
  validarse y hoy no se invocan, es un agujero funcional, no código a borrar.
- **Señales de "proceso/agente vivo" en status:**
  `orquesta-mcp/autoprogramming_status_health_v0.go:230,258`
  (`hasMCPAutoprogrammingLiveProcessSignalV0`, `hasMCPAutoprogrammingLiveAgentSignalV0`).
  → Relacionado directamente con las incidencias `alive_percentage=0` y
  `running_stale`. Verificar si la salud de cola debía usar estas señales.
- **Detalle sensible / forbidden-detail rails:**
  `orquesta-core-workflow/sensitive_detail_rails_v0.go:23` `detailProhibitedRailsEnabledV0`,
  `orquesta-director-agent/director_decision_ref_helpers_v0.go` (5 funcs de
  detección de fragmentos prohibidos),
  `orquesta-context/context_materialization_helpers_v0.go:101`,
  `orquesta-app-director-service/wait_refs_v0.go:30`.
- **Validación de registro de procesos de agente:**
  `orquesta-orchestration-core/agent_process_registry_validation.go:9,21`
  (`normalizeAgentProcessRecordV0`, `validateAgentProcessRecordV0`) —
  normalización/validación de records de proceso sin invocar.
- `orquesta-orchestration-core/operational_director_closure_validation_v0.go:106`
  `operationalDirectorClosureTaskRequiresTestsV0` (¿el cierre debía exigir tests?),
  `orquesta-core-workflow/review_result_record_projection_v0.go:59`
  `reviewResultAlreadyReflectedV0`.

---

## Hallazgo 6 — Limpiezas menores señaladas por staticcheck (sin impacto de comportamiento)

Verificadas como inofensivas pero conviene sanearlas:

- `SA4006` (valor asignado nunca usado): inicializadores/asignaciones muertas en
  `orquesta-app-codex-stack/drain_attempt_v0.go:248,253`,
  `run_coordinator_launch_outbox_recovery_v0.go:225,230` (reconciliación que se
  reasigna y se descarta; el resultado final es correcto pero el código engaña),
  y `cmd/orquesta-server/idle_self_improvement_backlog_planner_v0.go:41-48`
  (inicializadores vacíos sobrescritos en la línea siguiente).
- `SA4009` en `orquesta-cli/autoprogramming_client_v0.go:189-190`: los parámetros
  `estado`/`issues` se sobrescriben antes de usarse (línea 200 los recomputa);
  son parámetros muertos. La salida es correcta, pero deberían eliminarse de la
  firma (los callers pasan `result.Estado`/`result.Errores` cuando `result` aún
  es el valor cero).
- `SA4017`/`SA4006` en `orquesta-mcp/autoprogramming_validate_request_tool_v0.go:92-94`:
  `ctx = context.Background()` muerto (el ejecutor no usa `ctx`).
- `SA4031` (nil check imposible) en
  `orquesta-domain-work*/clone_v0.go` y `orquesta-domain-work/job_identity_v0.go`:
  `if out == nil` tras `make(...)`; nunca se cumple. Inofensivo (ya devuelven
  slice no-nil), pero es ruido.
- `SA1012` (×22): tests que pasan `nil` como `context.Context`; usar
  `context.TODO()`.
- Estilo: `ST1005`/`ST1008`/`S1016`/`S1009`/`S1017`/`S1011`/`S1001`/`S1002`.

---

## Hallazgo 7 — Goal-first: el loop pasa a "lanzar goal + observar", y quedan restos del loop forzado

Contexto (confirmado en commits recientes —"Centraliza lifecycle goal-first
neutral", "Lista goals activos para status goal-first", "Exige doble llave en
supervisor legacy", "Exige modo legacy en prepare-run"— y en
`docs/handoff_orquesta_goal_first_parada_2026-06-26.md`): Orquesta deja de
**forzar el loop** desde el orquestador y pasa a **goal-first**: declara un
`GoalWorkSpecV0`, lo lanza y luego lo **observa** hasta cierre, en vez de conducir
ticks de director.

Cómo queda el flujo (revisado en código, correcto):

- `external-work/run` en modo goal-first
  (`orquesta-app-codex-stack/external_work_goal_first_executor_v0.go`) **lanza el
  goal inmediatamente** (`GoalLauncher.LaunchGoalWorkV0`) y persiste
  `GoalWorkStateV0`; no solo encola.
- El loop legacy queda detrás de **doble llave**: requiere
  `AllowLegacyDirectorLoop` **y** `DirectorExecutionMode=legacy_loop` explícito;
  si no, devuelve `legacy_director_loop_opt_in_required` /
  `goal_first_backend_required`.
- El supervisor de un run goal-first
  (`goal_first_supervisor_guard_v0.go`) ya no ejecuta loop: si hay estado en
  `GoalStateStore` devuelve `observe_required` (`QueueStatus=delivered`); si es un
  contenedor goal-first **sin estado** devuelve `state_missing`
  (`QueueStatus=stopped`).

Implicaciones a vigilar (encajan con las incidencias OPES abiertas):

1. **Goal-first necesita un observador activo para cerrar.** El avance/cierre del
   goal depende del bucle de observación (`ObserveGoalWorkV0` /
   `autoprogramming_observe_active_goals_transport_v0.go`). En la instancia OPES
   de las incidencias, `resident_director_status=disabled` y
   `external_bridge_status=disabled`. Si en esa configuración no hay quien observe
   los goals lanzados, los runs quedan "aceptados/lanzados pero sin progresar"
   —exactamente el síntoma `waiting_outbox`/`running_stale` reportado—. **Verificar
   qué componente ejecuta la observación cuando el director residente está
   apagado**, y si `external-work/run` goal-first debe exigir/avisar que sin
   observador no habrá cierre autónomo.
2. **Coherencia de estado de cola goal-first.** `goalFirstSupervisorDispositionV0`
   marca `delivered`/`stopped`, pero `autoprogramming/status` sigue mostrando
   runs como `running` sin proceso vivo (incidencia `alive_percentage=0`). Cruza
   con el Grupo B del Hallazgo 5: las señales de "proceso/agente vivo"
   (`hasMCPAutoprogrammingLiveProcessSignalV0`) están **muertas**; si el status
   debiera distinguir `running_live` de `running_stale` con esas señales y no las
   usa, ahí está el bug de observabilidad que piden las incidencias.
3. **Limpieza del loop forzado.** Los símbolos `idleSelfImprovement*`,
   `maybeScheduleIdleSelfImprovementV0`, `externalBridgeLoopContextV0`,
   `drainRunAttemptV0`/`continueDrainRunAfterExternalV0` (Grupo A del Hallazgo 5)
   son residuos del modelo anterior; borrarlos reduce el riesgo de reactivar el
   loop forzado por error y aclara que el camino vivo es goal-first.

## Verificado como CORRECTO (no reinvestigar)

Para acotar el trabajo, estas hipótesis derivadas de las incidencias se
comprobaron y **no** son fallos de código:

1. **Timeouts de los handlers HTTP.** `runs/supervise`, `autoprogramming/supervise`,
   `autoprogramming/status` y `runs/queue/priority`
   (`modulos/orquesta-mcp/*_http_v0.go`) **sí** acotan con `responseTimeout` y
   devuelven `202 Accepted` con cuerpo al exceder el plazo
   (`run_supervisor_http_v0.go:13` = 2s). Los cuelgues de 90–180s de las
   incidencias **no** vienen de falta de timeout en estos handlers; buscar en
   capa inferior (gateway/proceso/lock del ejecutor) o en el despliegue concreto.
2. **Dispatch que continúa tras cortar el cliente.** Es intencionado:
   `runSupervisorExecutionContextV0` usa `context.WithoutCancel(r.Context())`
   (`run_supervisor_http_context_v0.go:12`). Explica las materializaciones
   tardías; no es bug.
3. **Coherencia contrato↔código en `orquesta-run-queue`.** `RankRunCandidatesV0`
   /`IsExecutableRunStatusV0` (`rank_v0.go:54`) filtran exactamente el conjunto
   terminal documentado en `docs/contratos.md`.
4. **Existe detección de runs stale** (`orquesta-mcp/autoprogramming_status_stale_running_v0.go`).
   Si el síntoma "running sin proceso vivo" persiste, es lógica de reconciliación
   a revisar, no una pieza ausente.

---

## Mejoras recomendadas (proceso / calidad)

1. **CI con `-race`**: el suite sin `-race` oculta el Hallazgo 2. Añadir
   `go test -race ./...` al pipeline.
2. **CI con `staticcheck` + `govulncheck` + `go vet`**: hoy hay 138 U1000 y 16
   vulns que pasan desapercibidos. Añadir un gate (al menos para SA*/copylocks).
3. **Subir el toolchain Go** a ≥1.25.7 (cierra Hallazgo 3) y actualizar
   dependencias (`golang.org/x/text`).
4. **Tests deterministas**: prohibir `time.Sleep` como sincronización; usar
   `waitAsyncWorkV0`/canales. Hay más tests con `time.Sleep` además de los 3 que
   ya fallan.
5. **Auditar la política de ACK no cableada** en `orquesta-runtime-codex`
   (Hallazgo 5) frente a las incidencias de ACK no escrito / rails pendientes.

## Cómo reproducir

```bash
go build ./...
go vet ./...                                   # copylocks en runtime-required-test
PATH="$PATH:$(go env GOPATH)/bin"
staticcheck ./...                              # SA*/U1000
govulncheck ./...                              # 16 vulns stdlib
go test ./...                                  # 85 ok / 0 fail
go test -race ./modulos/orquesta-server/...    # 3 FAIL por DATA RACE
```
