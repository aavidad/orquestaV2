# Mapa de aceptación y evidencias

Fecha de corte: 2026-07-14

Estado: evidencia histórica del corte mínimo. No gobierna el roadmap total.

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

## Ejecuciones finales registradas

| Fecha | Comando | Resultado | Alcance |
|---|---|---|---|
| 2026-07-14 | `go test -mod=vendor -count=1 ./...` | `PASS` histórico; comando revocado | después se comprobó que `./...` enumera 131 paquetes y puede lanzar smokes legacy; no es gate vigente |
| 2026-07-14 | `go test -mod=vendor -v -count=1 ./internal/bootstrap -run '^TestRealCodexAdapterClosesGoalThroughProductionMCPServer$' -args -orquesta-real-codex-config=<TOML temporal>` | `PASS` en 3,54 s | servidor de producción, Bearer local, cliente MCP oficial, Codex real, SQLite, CAS, artefacto y atestación |
| 2026-07-14 | `go test -mod=vendor -race -count=1 ./internal/application ./internal/adapters/agent/codex ./internal/adapters/artifact/filesystem ./internal/adapters/auth/localtoken ./internal/adapters/state/sqlite ./internal/bootstrap ./internal/interfaces/mcp` | `PASS` | concurrencia en el vertical nuevo y sus adaptadores con estado |
| 2026-07-14 | `GOFLAGS=-mod=vendor go vet ./internal/... ./cmd/orquesta` | `PASS` | análisis estático del producto nuevo |
| 2026-07-14 | `git diff --check` y `scripts/check_rebuild_write_set.sh` | `PASS` | higiene del diff y aislamiento frente al árbol legacy; 2332 rutas autorizadas, incluido vendor |

El placeholder `<TOML temporal>` es intencional: la ruta efímera usada por la
prueba no es contrato operativo ni se conserva como configuración. La prueba
creó un request nuevo, verificó un único artefacto, una única atestación y el
marcador único
`ORQUESTA_CODEX_E2E_OK_a30b46e51709ad4243f6927aba07f9e7` leído de vuelta por
MCP. El recibo durable
[`product/evidence/real_codex_mcp_e2e.json`](../../product/evidence/real_codex_mcp_e2e.json)
liga marcador, comando y fecha al SHA-256 exacto de las fuentes, dependencias
vendorizadas, configuración y manifest del producto.

## Gates de integración

Los resultados de la tabla pertenecen al candidato mínimo. Para trabajo nuevo,
la raíz valida manifest/receipt y los paquetes se enumeran de forma explícita;
`./...` queda prohibido porque incluye superficies congeladas.

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
