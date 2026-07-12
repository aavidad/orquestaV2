# Verificación causal independiente de candidatos BORRAR H4

Fecha: 2026-07-12. Base de la primera pasada: `0e3c863299`; desde esa base hasta
`af9d0efccc` solo cambió documentación. Cada candidato se retiró por AST en un
worktree disjunto. Después se ejecutaron, en este orden:

```text
GOPROXY=off go build -mod=vendor ./...
GOPROXY=off go test -mod=vendor -count=1 <paquete-focal>
```

La primera ejecución, contaminada por `no space left on device`, fue descartada
íntegramente. Esta tabla procede de la repetición con caché en `/tmp`, 90 GB
libres y paralelismo 2.

| Entrada | Símbolo | Build | Test focal | Salida causal | Veredicto |
|---:|---|---:|---:|---|---|
| 1 | `codexAppServerIssueCodeForErrorV0` | 0 | 0 | `ok orquesta/cmd/orquesta-server` | BORRAR |
| 2 | `codexAppServerIssueCodeFromCommandFailureV0` | 0 | 0 | `ok orquesta/cmd/orquesta-server` | BORRAR |
| 3 | `codexAppServerIssueCodeFromLogFileV0` | 0 | 0 | `ok orquesta/cmd/orquesta-server` | BORRAR |
| 4 | `codexAppServerTmuxStartupTimeoutV0` | 0 | 0 | retirada conjunta del import huérfano `time`; `ok orquesta/cmd/orquesta-server` | BORRAR |
| 6 | `CliPublicErrorCodeKnownV0` | 0 | 0 | `ok orquesta/modulos/orquesta-cli` | BORRAR |
| 18 | `DefaultDirectorAgentDecisionBatchBudgetV0` | 0 | 0 | `ok orquesta/modulos/orquesta-director-agent-workflow` | BORRAR |
| 28 | `documentToolPublicErrorV0` | 0 | 0 | retirada conjunta del import huérfano `strings`; `ok orquesta/modulos/orquesta-document-extraction` | BORRAR |
| 29 | `DefaultDocumentExtractionPolicyV0` | 0 | 0 | `ok orquesta/modulos/orquesta-document-extraction` | BORRAR |
| 34 | `NewMCPArrancarDirectorAppErrorResultV0` | 0 | 0 | `ok orquesta/modulos/orquesta-mcp` | BORRAR |
| 55 | `CodexAppServerIssueCodeFromCommandFailureV0` | 1 | — | `codex_goal_app_server_v0.go:523:39: undefined` | CONSERVAR |
| 56 | `CodexAppServerIssueCodeFromLogFileV0` | 1 | — | `codex_goal_app_server_v0.go:527:39: undefined` | CONSERVAR |
| 67 | `autoprogrammingStatusToolInputV0` | 1 | — | `autoprogramming_prepare_run_client_v0.go:171:3: undefined` | CONSERVAR |
| 73 | `NuevaAppI18nTextV0` | 0 | 0 | `ok orquesta/modulos/orquesta-web` | BORRAR |
| 78 | `goalFirstHTTPMCPClosureIssuesContainCodeForTestV0` | 0 | 0 | `ok orquesta/cmd/orquesta-server` | BORRAR |
| 79 | `postAutoprogrammingGoalFirstObserveForTestV0` | 0 | 0 | `ok orquesta/cmd/orquesta-server` | BORRAR |
| 80 | `postAutoprogrammingGoalFirstStatusForTestV0` | 0 | 0 | `ok orquesta/cmd/orquesta-server` | BORRAR |
| 81 | `postAutoprogrammingGoalFirstSuperviseForTestV0` | 0 | 0 | `ok orquesta/cmd/orquesta-server` | BORRAR |
| 82 | `minIntForTestV0` | 0 | 0 | `ok orquesta/cmd/orquesta-server` | BORRAR |
| 98 | `markNoExecutionWindowStartForTestV0` | 0 | 0 | `ok orquesta/modulos/orquesta-server` | BORRAR |
| 99 | `resetNoExecutionWindowForTestV0` | 0 | 0 | `ok orquesta/modulos/orquesta-server` | BORRAR |

Resultado: 17 BORRAR acreditados y 3 falsos positivos del inventario de
deadcode que deben CONSERVARSE. La clasificación global propuesta cambia de
`38 CONECTAR / 20 BORRAR / 41 CONSERVAR` a
`38 CONECTAR / 17 BORRAR / 44 CONSERVAR`.

Un import que queda huérfano al retirar la única función que lo usa no demuestra
vida del símbolo: es parte inseparable de la misma mutación mecánica. Una
referencia `undefined` desde otro punto sí demuestra vida y bloquea BORRAR.
