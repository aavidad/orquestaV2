# Encargo programador: correcciones de auditoría Orquesta

Fecha: 2026-06-27.
Origen: auditoría completa en
`TAREA_OPES_ORQUESTA_REVISION_CODIGO_Y_CONTRATOS_2026-06-27.md` (evidencia y
detalle). Este fichero es la **orden de trabajo ejecutable**: hazla en orden,
cada tarea trae fix propuesto y verificación. No toques OPES.

Baseline de partida: `go build` OK, `go test ./...` 85 ok / 0 fail (sin `-race`).

---

## T1 (BLOQUEANTE) — Goal-first: garantizar observación/cierre sin director residente

Problema: en goal-first, `external-work/run` **lanza** el goal
(`modulos/orquesta-app-codex-stack/external_work_goal_first_executor_v0.go`,
`GoalLauncher.LaunchGoalWorkV0`) pero el **cierre depende de un observador**
(`ObserveGoalWorkV0` / `autoprogramming_observe_active_goals_transport_v0.go`).
En la instancia OPES de las incidencias `resident_director_status=disabled` y
`external_bridge_status=disabled`, así que los goals lanzados pueden quedar sin
avanzar (`waiting_outbox`/`running_stale`).

Hacer:
1. Determinar qué componente ejecuta la observación cuando el director residente
   está apagado. Si ninguno, exponerlo: `external-work/run` goal-first debe
   devolver en su resultado un estado claro
   (`accepted_goal_launched_no_observer` o similar) y/o `autoprogramming/status`
   debe indicar `observer_required`.
2. Permitir disparar la observación de forma acotada y sin polling manual por
   `run_ref` (drenar observación de goals activos por cola).
3. Verificación: con director residente desactivado, lanzar un goal y comprobar
   que el estado público dice si avanzará solo o necesita acción, y que existe
   una llamada única que progresa los goals activos.

## T2 (BLOQUEANTE) — Distinguir `running_live` de `running_stale` en status

Problema: `autoprogramming/status` muestra runs como `running` sin proceso vivo
(incidencia `alive_percentage=0`). Las señales existen pero están **muertas**:
`modulos/orquesta-mcp/autoprogramming_status_health_v0.go:230`
`hasMCPAutoprogrammingLiveProcessSignalV0` y `:258`
`hasMCPAutoprogrammingLiveAgentSignalV0`.

Hacer: cablear esas señales (o equivalentes) en la proyección de salud para que
`status` separe `running_live` de `running_stale`, y que el reconciliador degrade
runs `running` sin proceso vivo ni ACK pendiente.
Verificación: test que, dada una run `running` sin proceso vivo, `status` la
reporte `running_stale` y proponga acción segura.

Revalidación 2026-06-28: T2 queda cubierto en el stack Codex actual. La
clasificación pura vive en `orquesta-run-coordinator/liveness_v0.go`;
`autoprogramming/status` la usa para separar `running_live`,
`running_stale_no_process` y `running_without_recent_stats`; y
`orquesta-app-codex-stack` ejecuta `reconcileQueuedRunningStaleRunsV0` antes de
coordinar el tick. Pruebas focales verdes:
`go test -count=1 ./modulos/orquesta-mcp -run NoMarcaStaleConDirectorStatsSnapshotRunning`
y `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'RunningStale|OrphanQueue'`.
Queda fuera de esta marca cualquier composición no Codex que no inyecte store,
registry y snapshot equivalentes.

## T3 — Arreglar data races en tests de `orquesta-server`

Confirmado con `go test -race ./modulos/orquesta-server/`. Fallan:
`TestRuntimeV0SupervisorAsyncCoalesceaUnTickPendienteV0`,
`TestRuntimeV0ResidentDirectorAsyncCoalesceaUnTickPendienteV0`,
`TestRuntimeV0IdleSelfImprovementGoalFirstLanzaGoalSpecV0`.

Revalidación 2026-06-28: no reproduce en el árbol actual con
`go test -race -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`.
Ambos paquetes pasan. Si reaparece, conservar la traza concreta porque la
incidencia original ya no es reproducible con esta matriz.

Causa: leen `store.last` mientras la goroutine async escribe; sincronizan con
`time.Sleep` (`supervisor_loop_async_idle_v0_test.go:45-47`).
Fix: sustituir `time.Sleep(...)` por
`runtime.waitAsyncWorkV0(context.Background())` (ya existe en
`runtime_async_work_v0.go:38`) antes de leer `store.last`. Revisar otros tests
con `time.Sleep` como sincronización.
Verificación: `go test -race ./modulos/orquesta-server/...` en verde.

Avance 2026-06-28 noche: el gate ampliado `go test -race ./...` destapo una
carrera adicional en `cmd/orquesta-server`
(`TestRuntimeV0SelfAuditBacklogGoalFirstLanzaSpecOperacionalV0`): el launcher
fake retenia un `GoalWorkSpecV0` con slices compartidas y el lifecycle goal-first
las normalizaba de nuevo. Se corrige en `orquesta-goal` haciendo que
`NormalizeGoalWorkSpecV0` clone en profundidad las slices antes de normalizar, y
queda fijado con `TestNormalizeGoalWorkSpecV0NoMutaSlicesDeEntrada`. Evidencia:
`go test -race -count=1 ./...`, `go vet ./...`, `go test -count=1 ./...` y
`git diff --check` verdes.

Avance 2026-06-28 noche posterior: se instalaron las herramientas de analisis
en el entorno local y tambien quedan verdes `staticcheck ./...` y
`govulncheck ./...` (`No vulnerabilities found.`). El workflow
`.github/workflows/go-quality.yml` ya ejecuta `git diff --check`, `go build`,
`go vet`, `go test`, `govulncheck`, `staticcheck` y `go test -race`.

## T4 — Corregir copia por valor de mutex/Builder (`go vet`)

`go vet` marca copylocks en `modulos/orquesta-runtime-required-test`.
`outputBufferV0` (`output_buffer_v0.go:8`) tiene `sync.Mutex` + `strings.Builder`.
Fix: usar siempre `*outputBufferV0`:
- `local_command_executor_v0.go:188-200` `runLocalCommandV0` → devolver
  `*outputBufferV0` (`return output, err`).
- `:245` `localCommandOutputWithDiagnosticV0(output *outputBufferV0) *outputBufferV0`.
- `output_artifact_v0.go:21` y llamadas `:84,:105-109` →
  `writeOutputArtifactV0(..., output *outputBufferV0, ...)`.
Verificación: `go vet ./modulos/orquesta-runtime-required-test/...` limpio y
`go test ./modulos/orquesta-runtime-required-test/...` verde.

## T5 — Subir toolchain Go y dependencias (vulnerabilidades)

`govulncheck` reporta 16 vulns de stdlib por `go1.25.5` (p.ej. `GO-2026-4341`
net/url, `GO-2026-4340`/`GO-2026-4337` crypto/tls).
Fix: subir toolchain a ≥`go1.25.7`; `go get golang.org/x/text@latest`; recompilar.
Verificación: `govulncheck ./...` sin vulnerabilidades alcanzables.

## T6 — Revisar intake de artefactos (¿solo primer fichero?)

`modulos/orquesta-app-codex-stack/domain_work_delivery_artifact_intake_v0.go:28-39`:
el `for range ack.Files` siempre retorna en la 1ª iteración (staticcheck SA4004),
así que solo lee `ack.Files[0]` y no filtra por `artifactType`.
Hacer: confirmar contra el contrato de `DomainWork`/ACK. Si puede haber varios
ficheros, filtrar por `artifactType`; si siempre es uno, reescribir como
`if len(ack.Files) > 0 { ... }` para dejar la intención explícita.

## T7 — DECIDIR: validación muerta (cablear o borrar) — NO borrar sin decidir

Funciones de política sin uso que parecen necesarias (staticcheck U1000). Por
cada bloque, decidir **cablear** (si era protección prevista) o **borrar**:
- Política de ACK Codex: `modulos/orquesta-runtime-codex/codex_ack_local_path_policy_v0.go`,
  `codex_ack_path_policy_v0.go`, `codex_ack_pending_rail_refs_v0.go`,
  `codex_ack_strict_validation_v0.go:159`, `codex_ack_text_policy_v0.go`,
  `codex_delivery_observation_v0.go:295-320`.
  (Liga con incidencias de ACK no escrito / rails pendientes / detalle local.)
- Detalle sensible / forbidden-detail:
  `modulos/orquesta-core-workflow/sensitive_detail_rails_v0.go:23`,
  `modulos/orquesta-director-agent/director_decision_ref_helpers_v0.go` (5 funcs),
  `modulos/orquesta-context/context_materialization_helpers_v0.go:101`.
- Validación de records de proceso:
  `modulos/orquesta-orchestration-core/agent_process_registry_validation.go:9,21`.
- Cierre exige tests:
  `modulos/orquesta-orchestration-core/operational_director_closure_validation_v0.go:106`.

## T8 — Borrar código muerto obsoleto (Grupo A)

Restos del viejo loop forzado / idle-self-improvement (superado por goal-first) y
helpers sueltos. Borrar (lista completa: `staticcheck ./... | grep U1000`):
- `modulos/orquesta-server/supervisor_idle_*_v0.go` (funcs idle sin uso),
  `runtime_v0.go:184` `persistStateV0`,
  `cmd/orquesta-server/idle_self_improvement_*`,
  `external_bridge_lifecycle_helpers_v0.go:114`, `server_env_autonomy_v0.go:34`.
- Fingerprints legacy: `modulos/orquesta-domain-work-{file,memory,sql}/fingerprint_v0.go`.
- `modulos/orquesta-app-codex-stack/drain_attempt_v0.go:15,298`,
  `autoprogramming_bridge_continue_v0.go:60`,
  `autoprogramming_prepare_run_retry_v0.go:75`, `spec_packet_v0.go:41`.
- CLI bootstrap-appspec: `modulos/orquesta-cli/bootstrap_appspec_client_v0.go`,
  `bootstrap_appspec_client_helpers_v0.go`, `command_flags_v0.go:64`,
  `transport_rest_response_v0.go:13`, `transport_rest_v0.go:144`.
- `cmd/orquesta-guardian/guardian_files_v0.go`, `guardian_command_io_v0.go:72,91`.
- `modulos/orquesta-web/director_stats_projection_v0.go:3,9`,
  `orquesta-observability/*`, `orquesta-director-scheduler/*`,
  `orquesta-persistence/outbox_ledger_*`, y los 21 helpers en `*_test.go`.
Verificación: `staticcheck ./... | grep -c U1000` baja a 0 (tras T7) y suite verde.

## T9 — Limpiezas menores (staticcheck)

- `SA4006` asignaciones muertas: `drain_attempt_v0.go:248,253`,
  `run_coordinator_launch_outbox_recovery_v0.go:225,230`,
  `cmd/orquesta-server/idle_self_improvement_backlog_planner_v0.go:41-48`.
- `SA4009` params muertos: `modulos/orquesta-cli/autoprogramming_client_v0.go:189-190`
  (quitar `estado`/`issues` de la firma).
- `SA4017`: `modulos/orquesta-mcp/autoprogramming_validate_request_tool_v0.go:92-94`.
- `SA4031` nil checks imposibles: `orquesta-domain-work*/clone_v0.go`,
  `orquesta-domain-work/job_identity_v0.go:112,130`.
- `SA1012` (×22): tests con `nil` context → `context.TODO()`.
- Estilo: `ST1005/ST1008/S1016/S1009/S1017/S1011/S1001/S1002`.

## T10 — CI: gates que habrían pillado esto

Añadir al pipeline: `go vet ./...`, `go test -race ./...`, `staticcheck ./...`
(al menos categorías SA + copylocks) y `govulncheck ./...`.

---

## Comprobación final (toda la batería)

```bash
go build ./...
go vet ./...
PATH="$PATH:$(go env GOPATH)/bin"
staticcheck ./...
govulncheck ./...
go test ./...
go test -race ./...
```
Objetivo: build OK, vet limpio, staticcheck sin SA ni U1000, govulncheck sin
vulns alcanzables, tests (incl. `-race`) en verde.
