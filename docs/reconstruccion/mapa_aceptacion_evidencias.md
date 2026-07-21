# Mapa de aceptación y evidencias

Fecha de corte: 2026-07-22

Estado: mapa explicativo del corte acreditado V16. No gobierna el roadmap total
ni sustituye los receipts estructurados.

[`product/capabilities.json`](../../product/capabilities.json) es el manifest
ejecutable del corte. `status: accepted` significa que la capacidad tiene un
contrato verificable en su `acceptance_ref`; este documento explica qué cubre y
separa ese contrato de los resultados de una ejecución concreta.

## Contratos por capacidad

| ID | Contrato acreditado | `acceptance_ref` |
|---|---|---|
| `CORE-INTENT` | manifiesto inmutable, hash de integridad, refs opacas y errores estables | `internal/goal/intent_test.go` |
| `CORE-GOAL` | transiciones válidas y cierre solo tras terminalidad de sus WorkItems | `internal/goal/goal_test.go` |
| `CORE-WORK-ITEM` | revisiones inmutables, terminalidad y rechazo de transiciones imposibles | `internal/goal/work_item_test.go` |
| `CORE-SINGLE-WRITER` | aplicación como único escritor durable e idempotencia de submit | `internal/application/orchestrator_test.go` |
| `CORE-EVIDENCE` | éxito con artefacto/atestación y fallo explícito sin falsa evidencia | `internal/application/closure_test.go` |
| `IDENTITY-LOCAL-OWNER` | principal local estable, refs opacas y contexto de proyecto explícito | `internal/identity/contracts_test.go` |
| `IDENTITY-LOCAL-TOKEN` | secreto local privado y estable, Bearer obligatorio y rechazo sin fuga | `internal/adapters/auth/localtoken/localtoken_test.go` |
| `STATE-SQLITE` | migración, scope, idempotencia, leases, CAS atómico y restauración | `internal/adapters/state/sqlite/repository_test.go` |
| `ARTIFACT-FILESYSTEM` | blobs CAS idempotentes, privados, íntegros y sin escape por symlink | `internal/adapters/artifact/filesystem/store_test.go` |
| `CONFIG-CANONICAL` | registro tipado, precedencia, TOML estricto, redacción y sincronía | `internal/config/config_test.go` |
| `AGENT-CONTRACT` | causalidad, identidad, terminalidad recuperable y output acotado | `internal/ports/agent_contract_test.go` |
| `AGENT-CODEX` | Codex real cierra un Goal por el servidor MCP y produce evidencia legible | `internal/bootstrap/codex_real_e2e_test.go` |
| `API-MCP` | cliente MCP oficial autenticado recorre API, SQLite, artefactos, replay y status; acceso anónimo recibe 401 | `internal/bootstrap/mcp_e2e_test.go` |
| `OPS-RESTART-REPLAY` | reinicio no duplica ejecución, artefacto ni evidencia | `internal/bootstrap/restart_e2e_test.go` |
| `OPS-SHUTDOWN` | parada cancela y recolecta el proceso de agente propio | `internal/bootstrap/shutdown_e2e_test.go` |
| `ARCH-HEXAGONAL` | fronteras de imports, env canónico, cmd fino y ausencia de YAML/legacy | `architecture_rebuild_test.go` |
| `SURFACE-I18N` | paridad de catálogo y fallback español | `internal/i18n/catalog_test.go` |

Evidencia focal adicional, sin sustituir los refs canónicos:

- Codex: `internal/adapters/agent/codex/adapter_integration_test.go` cubre
  idempotencia tras reinicio, límites de salida/diagnóstico, cancelación,
  shutdown, persistencia terminal y fallo si no puede limpiar el grupo. En
  Linux, `process_group_unix_test.go` acredita que no queda el descendiente.
- MCP: `internal/interfaces/mcp/interface_integration_test.go` cubre el conjunto
  exacto de tools, códigos públicos localizados y protecciones HTTP.
- Restore: `internal/goal/snapshot_test.go` prueba round-trip, separación de
  snapshots y rechazo de manipulación o revisiones incoherentes.

## V13: mailbox acreditado

`AC-V13-MAILBOX` está cerrado con receipt V3 `PASS`. La ejecución contractual
partió del commit sellado `5daf174bde3ec5d9a98f387de05491f258634264`
en checkout `detached_clean` y acredita exactamente `ORC-04`, `ORC-05` y
`ORC-14`.

| IDs acreditados | Contrato acreditado | Alcance exacto |
|---|---|---|
| `ORC-04`, `ORC-05`, `ORC-14` | `acceptance/v13_mailbox_test.go`; `internal/application/mailbox*_test.go`; `internal/adapters/state/sqlite/mailbox*_test.go` | Solo `child_delivery` contractual: destinatario exacto, lifecycle `admitted → claimed → delivered → consumed → acknowledged|blocked`, retiro sistémico, replay/fencing, barrera causal padre/hijo y recovery SQLite sobre la misma autoridad |

El cierre conserva un único writer de Goal, `StateRepository`, outbox y
`Fence`. `BuildMailboxResolutionGoal` es helper puro; SQLite valida el mismo
resultado y no muta Goal por una ruta lateral. `mailbox.max_envelope_bytes`
entra por el registro canónico. Los ratchets V02, V05, V06, V09 y V10 forman
parte de `TestAcceptanceV13Mailbox`. `ORC-15` y los bindings públicos de mailbox
no pertenecen a esta acreditación.

Cadena V13:

```text
producto P:    a8bf28a2acb3a246d8c1a3c1cedea38d240f370a
sellado C:     5daf174bde3ec5d9a98f387de05491f258634264
evidencia E:   93b3c44f37869d72ea77db24fa5185dc29d62d32
candidate SHA: sha256:3fdd094fee1504b5c246d02687552a0c563b905e992ef532f3d4a820dfd7e435
output SHA:    sha256:b750a94e24755f4b287e85c6304d5bad86ce9e9397153ca15a178f8fd187e34e
```

Reachability queda separada de la semántica de linaje: `Parent` no implica
handoff, `HandoffRequired` es explícito y `false` por defecto, y `true` sin
padre se rechaza. `TestMCPParentMetadataClosesWithoutMailboxOrRequeue` acredita
que V13 no expone ese opt-in en `WorkItemInput` público y que el DAG V05 cierra
sin mailbox, requeue ni acciones pendientes mientras llegan los bindings de
V20–V22.

## V14: controles acreditados

`AC-V14-CONTROLS` está cerrado. Su argv exacto pasó desde checkout
`detached_clean` y `product/evidence/v14_controls.json` es receipt V3 `PASS`.

```text
producto P:   6dbc0d808de63973305914b002c3bc2b8a806bb0
sellado S:    e3e7c28e669ccd7e67a8661c40333d649ab82dd5
evidencia E:  5b97545ad14a40fd0063fc3671f6e79d9978ec09
```

| IDs acreditados | Contrato acreditado | Alcance exacto |
|---|---|---|
| `GOV-07`, `STG-15`, `ORC-03`, `ORC-16` | `acceptance/v14_controls_test.go`; `internal/application/control*_test.go`; `internal/adapters/state/sqlite/*control*_test.go`; `internal/bootstrap/controls_e2e_test.go` | Pause/resume, cancel, stop cooperativo/forzado, retry de Execution y replan causal sobre el mismo Goal, CAS, outbox y `StateRepository` |

Garantías acreditadas:

- controles autenticados, idempotentes y cercados por proyecto, Goal, AppSpec,
  PlanGeneration, revisión de WorkItem, intento de Execution y fingerprint;
- pausa Goal/WorkItem bloquea solo nuevos launches; observación, mailbox y
  trabajo in-flight continúan; resume reutiliza la acción existente;
- stop apunta a una Execution exacta, requiere capability y receipt exactos,
  separa request/confirmation y nunca sustituye el shutdown del runtime;
- stop forzado solo puede superseder el cooperativo activo del mismo intento;
  A/B/C/D, crash/restart, PID/PGID, owner lock y launch gate prueban aislamiento
  y convergencia sin proceso huérfano ni efecto terminal duplicado;
- cancel, completion, retry y replan compiten por el mismo CAS; terminales no
  se reabren, retry crea un intento nuevo y replan conserva historia append-only;
- SQLite, backup/recovery y validación física preservan fences, receipts,
  outbox, mailbox V13 y una sola autoridad;
- `TestRealCodexControlsThroughProductionComposition` recorre bootstrap
  productivo, proceso controlable, stop forzado selectivo y receipt durable,
  dejando Goal abierto, WorkItem `interrupted` y Execution `stopped`.
- `TestRealCodexCooperativeStopLeavesResidentSchedulerLive` conserva progreso
  del scheduler único ante `SIGTERM` ignorado y hace alcanzable la escalada
  forzada; el replay de intent forced sin proof reintenta `SIGKILL` idempotente
  contra la identidad exacta y converge.

V14 solo añade casos de uso internos de aplicación. No añade bindings HTTP,
MCP o CLI ni otra tool; el registro único y esos bindings son V20, y la paridad
i18n completa es V21. Presupuestos, approvals, fairness y retry de efectos se
acreditan por separado en V15.

## V15: presupuestos y efectos condicionados al receipt

`AC-V15-BUDGETS-EFFECTS` solo está cerrado cuando su argv exacto pasa desde
checkout `detached_clean` y `product/evidence/v15_budgets_effects.json` valida
como receipt V3 `PASS`. Un fichero ausente, `{}` o ligado al candidato anterior
mantiene V15 pendiente. El receipt válido conserva los OID y digests sellados;
este mapa no mantiene una copia manual susceptible de divergir.

| IDs acreditados | Contrato acreditado | Alcance exacto |
|---|---|---|
| `GOV-15`, `STG-09`, `ORC-08`, `ORC-09`, `ORC-10`, `ORC-11`, `EVD-03`, `EVD-14` | `acceptance/v15_budgets_effects_test.go`; suites V15 de aplicación, SQLite, bootstrap y Codex | Envelopes global/proyecto/Goal, reservas y settlement durables, fairness jerárquica, criticidad/esfuerzo tipados y ledger `intent -> approval -> attempt -> receipt` para launch/stop |

Garantías ejercitadas por el candidato y acreditadas solo tras ese gate:

- cien claims concurrentes respetan simultáneamente límites global, proyecto y
  Goal; cuota temporal aparca sin fabricar intento o estado terminal;
- round-robin durable `project -> Goal` progresa bajo saturación sin impedir
  observaciones, stop o mailbox;
- policy, target, TTL, autoridad, membership, fence y tiempos se validan antes
  del adaptador y de nuevo en persistencia/recovery;
- uso exacto, parcial o desconocido produce settlement conservador; overrun
  queda visible y bloquea exposición nueva incluso con capacidad aparente;
- launch y stop separan intent, approval, attempt, receipt y consumo de outbox;
  crash/retry reutiliza idempotency key y no duplica efecto;
- acciones históricas V14 se aparcan o terminan sin aprobación retroactiva ni
  settlement inventado; su adopción operativa masiva queda en V32;
- composición Codex real demuestra ledger completo de launch y stop, receipt
  durable y enlace exacto desde el consumo;
- backup/restore, recovery adversarial, SQLite concurrente, `-race`, ratchets de
  arquitectura y presupuesto de simplicidad forman parte del argv sellado.

V15 no añade bindings públicos, workspace/Git ni otro scheduler/store.

## V16: workspace y Git local acreditados

`AC-V16-WORKSPACE-GIT` está cerrado: su argv exacto pasó desde S en checkout
`detached_clean` y `product/evidence/v16_workspace_git.json` valida como receipt
V3 `PASS`.

| IDs acreditados | Contrato acreditado | Alcance exacto |
|---|---|---|
| `STG-02`, `STG-10`, `EXT-10` | `acceptance/v16_workspace_git_test.go`; contratos y suites de `internal/application`, `internal/ports`, SQLite, `internal/adapters/workspace/gitlocal`, Codex y bootstrap | Workspace opaco por Execution, inventario/write-set, commit causal, integración local explícita por CAS, pendientes RBAC, crash/replay y E2E Git+SQLite real |

Garantías acreditadas por ese gate:

- cada Execution con `WriteSet` obtiene un binding privado, opaco e idempotente;
  una Execution sustituta no reutiliza workspace y un `WriteSet` vacío no crea
  ninguno;
- prepare, commit e integrate recorren el mismo state/outbox/effect ledger de
  V06/V15; no aparecen store, scheduler, cola ni lifecycle Git paralelos;
- el adapter usa Git CLI local con base, target, refs, OID y CAS exactos; un
  conflicto o target stale conserva el ChangeSet y deja intacto el destino;
- prepare fija su base antes del claim; commit solo reconcilia su objeto
  determinista; el perdedor CAS revalida el marker exacto y release conserva
  replay después de eliminar físicamente el worktree;
- integración exitosa exige aprobación/permiso explícitos y receipt causal;
  Goal `succeeded`, commit local o resultado `clean` no equivalen a integración;
- pending work se reconstruye desde hechos durables y queda aislado por actor,
  proyecto y RBAC tras restart;
- recovery liga binding, ChangeSet, observación, receipt, intent, attempt,
  fence y estado, y rechaza tanto tampering como borrado de hechos requeridos;
- roots disjuntos, entorno Git mínimo, `.git` abierto sin seguir links ni
  bloquear special files, `common-dir` ligado al repositorio autorizado,
  ausencia de paths/argv/secretos y tests concurrentes cubren la frontera local;
- el E2E real recorre Git temporal, SQLite, dos worktrees, commit, integración,
  conflict/stale, replay y consulta pendiente.

V16 no añade ninguna de las seis tools MCP públicas ni bindings HTTP/CLI.
`ListPendingChanges` e `IntegrateChange` son casos de uso internos hasta V20.
No acredita sandbox/atestación V17 ni Forge remoto `EXT-11`, que sigue en V28.

Estado contable:

```text
corte histórico V13: 44/257 = 17,12 %; 13/34 = 38,24 %; 13/13 receipts
corte histórico V14: 48/257 = 18,68 %; 14/34 = 41,18 %; 14/14 receipts
corte sin receipt V15 válido: 48/257 = 18,68 %; 14/34 = 41,18 %; 14/14 receipts
corte con receipt V15 válido: 56/257 = 21,79 %; 15/34 = 44,12 %; 15/15 receipts
corte histórico antes de V16: 56/257 = 21,79 %; 15/34 = 44,12 %; 15/15 receipts
corte vigente V16: 59/257 = 22,96 %; 16/34 = 47,06 %; 16/16 receipts
```

## Ejecuciones finales registradas

| Fecha | Comando | Resultado | Alcance |
|---|---|---|---|
| 2026-07-22 | argv exacto de `AC-V16-WORKSPACE-GIT`, registrado en `product/evidence/v16_workspace_git.json` | receipt V3 `PASS`; P=`b48162b0433dd32b6369ee324358e5f87af325ad`, S=`a4f602ab01c4e77f0d876c79c2c8e86b44b68fa4`, E=`38e1ffb82d6f71610ac0a745d693d52fd43dac22`; candidate `sha256:83962cf0feca66a38030b18f655d86201117df1d5b1cbe2efa795e8010c939a1`; output `sha256:1cf6dc65ac2138927bff503198eee1c63451437a04949ccb0d62e369e3187f2b` | workspace/Git local, SQLite/recovery, Codex binding, crash/replay, RBAC, CAS, 22 carreras, ratchets y vet desde `detached_clean` |
| 2026-07-21 | `go test -mod=vendor -v -count=1 ./internal/bootstrap -run '^TestRealCodexAdapterClosesGoalThroughProductionMCPServer$' -args -orquesta-real-codex-config=/home/alberto/Trabajo/.orquesta-rebuild-real-e2e-v14-20260716T171502/orquesta.toml` | `PASS` en `6.49s`; marcador `ORQUESTA_CODEX_E2E_OK_798736afa7e84a5f97e5be139c1d3e1e` | no regresión de composición productiva, MCP, Codex real, SQLite y CAS sobre source digest V16; no acredita workspace/Git ni sustituye receipt P/S/E |
| 2026-07-18 | argv exacto de `AC-V15-BUDGETS-EFFECTS`, registrado en `product/evidence/v15_budgets_effects.json` | válido únicamente si el receipt V3 `PASS` supera su test estricto desde `detached_clean`; OID/digests en el receipt | presupuestos jerárquicos, fairness, riesgo/esfuerzo, cadena causal launch/stop, policy histórica, recovery/backup, 100 claims concurrentes, carreras, ratchets y composición Codex V15 |
| 2026-07-16 | argv exacto de `AC-V14-CONTROLS`, registrado en `product/evidence/v14_controls.json` | receipt V3 `PASS`, `detached_clean`, P=`6dbc0d808de63973305914b002c3bc2b8a806bb0`, S=`e3e7c28e669ccd7e67a8661c40333d649ab82dd5`, E=`5b97545ad14a40fd0063fc3671f6e79d9978ec09` | controles internos, SQLite/recovery, fake/Codex, stop selectivo, scheduler vivo, carreras, ratchets y composición productiva V14 |
| 2026-07-16 | argv exacto de `AC-V13-MAILBOX`, registrado en `product/evidence/v13_mailbox.json` | `PASS`, `detached_clean` | contrato mailbox, guards de arquitectura/trazabilidad y paquetes Goal, identidad, config, aplicación, puertos, SQLite, bootstrap y cmd sobre el candidato sellado |
| 2026-07-16 | `go test -mod=vendor -v -count=1 ./internal/bootstrap -run '^TestRealCodexAdapterClosesGoalThroughProductionMCPServer$' -args -orquesta-real-codex-config=/home/alberto/Trabajo/.orquesta-rebuild-real-e2e-v13-20260716T115042/orquesta.toml` | `PASS` en `6.15s` | composición productiva, Bearer local, MCP, Codex real, SQLite y CAS sobre fuentes V13; smoke de no regresión, no contrato mailbox |
| 2026-07-14 | `go test -mod=vendor -count=1 ./...` | `PASS` histórico; comando revocado | después se comprobó que `./...` enumera 131 paquetes y puede lanzar smokes legacy; no es gate vigente |
| 2026-07-14 | `go test -mod=vendor -v -count=1 ./internal/bootstrap -run '^TestRealCodexAdapterClosesGoalThroughProductionMCPServer$' -args -orquesta-real-codex-config=<TOML temporal>` | `PASS` en 3,54 s | servidor de producción, Bearer local, cliente MCP oficial, Codex real, SQLite, CAS, artefacto y atestación |
| 2026-07-14 | `go test -mod=vendor -race -count=1 ./internal/application ./internal/adapters/agent/codex ./internal/adapters/artifact/filesystem ./internal/adapters/auth/localtoken ./internal/adapters/state/sqlite ./internal/bootstrap ./internal/interfaces/mcp` | `PASS` | concurrencia en el vertical nuevo y sus adaptadores con estado |
| 2026-07-14 | `GOFLAGS=-mod=vendor go vet ./internal/... ./cmd/orquesta` | `PASS` | análisis estático del producto nuevo |
| 2026-07-14 | `git diff --check` y `scripts/check_rebuild_write_set.sh` | `PASS` | higiene del diff y aislamiento frente al árbol legacy; 2332 rutas autorizadas, incluido vendor |

El placeholder `<TOML temporal>` de 2026-07-14 era intencional y esa ejecución
queda como evidencia histórica. La renovación vigente del 2026-07-21 reutilizó
una configuración aislada, creó un request nuevo, cerró el Goal por el servidor
MCP productivo y leyó de vuelta el marcador
`ORQUESTA_CODEX_E2E_OK_798736afa7e84a5f97e5be139c1d3e1e`. El receipt durable
[`product/evidence/real_codex_mcp_e2e.json`](../../product/evidence/real_codex_mcp_e2e.json)
liga comando, `2026-07-21T23:46:53+02:00` y source digest
`sha256:15018d857a263e8f3b99aa6eb68609ecbd288c55fe164ed53d5ddeb3a93a080d`.

Este `PASS` demuestra ausencia de regresión en composición productiva, MCP,
Codex, SQLite y CAS. El Goal sin `WriteSet` no activa workspace Git y el smoke
no acredita por sí solo `AC-V16-WORKSPACE-GIT`; el receipt V3 V16 separado ya
validado es el gate de esa vertical.

## Gates de integración

Los resultados de la tabla pertenecen al candidato mínimo. Para trabajo nuevo,
la raíz valida manifest/receipt y los paquetes se enumeran de forma explícita;
`./...` queda prohibido porque incluye superficies congeladas.

El cierre V14 ejecutó literalmente `execution_argv` de
`acceptance/fixtures/v14_controls.json` desde su candidato sellado. El receipt
V3 conserva la identidad de fuentes y el resultado reproducible; los verdes
posteriores del worktree no lo sustituyen.

Checklist reproducible:

```bash
git diff --check
scripts/check_rebuild_write_set.sh
go test -mod=vendor -count=1 .
go test -mod=vendor -count=1 ./internal/... ./cmd/orquesta
go test -mod=vendor -race -count=1 ./internal/application ./internal/adapters/agent/codex ./internal/adapters/artifact/filesystem ./internal/adapters/auth/localtoken ./internal/adapters/state/sqlite ./internal/bootstrap ./internal/interfaces/mcp
GOFLAGS=-mod=vendor go vet ./internal/... ./cmd/orquesta
```

El E2E Codex real se repite con el comando opt-in de la tabla y una instancia
TOML aislada. No se apunta a estado, artifacts, work roots ni runtime legacy.
