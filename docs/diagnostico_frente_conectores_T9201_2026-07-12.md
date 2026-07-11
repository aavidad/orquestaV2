# Diagnóstico del frente CONECTORES — T9201

Fecha de inspección: 2026-07-12. Alcance autorizado: `docs/`.

Este documento separa evidencia directa de código/tests (marcada como
**verificado**) de conclusiones que requieren ejecución o integración adicional
(marcadas como **hipótesis/pending**). No modifica código.

## Resumen ejecutivo

El núcleo tiene puertos neutrales explícitos y los adaptadores principales
existen. La evidencia local es fuerte para contratos, fakes y wiring del stack,
pero no demuestra por sí sola que todos los conectores estén operativos en una
instancia real con proveedor. El frente pendiente es de cierre de evidencia e
integración, no de ausencia general de interfaces.

## Matriz por módulo

| Módulo/superficie | Puerto del núcleo | Adaptador observado | Estado probado | Hueco y acción |
|---|---|---|---|---|
| `orquesta-runtime-codex-goal` | `GoalWorkLauncherPortV0`, `GoalWorkObservationPortV0` (`modulos/orquesta-goal/types_v0.go`) | `CodexGoalLauncherV0` y `CodexGoalObserverV0`, sobre `CodexGoalStarterPortV0`/`CodexGoalObserverPortV0` (`modulos/orquesta-runtime-codex-goal/packet_v0.go`) | **Verificado** por `packet_v0_test.go` y `packet_usage_observation_v0_test.go`; el README documenta que el proveedor real queda fuera del módulo. | La prueba de este módulo no prueba proveedor real. **Acción:** conservar un smoke de composición que pruebe launch→observe→closure con backend configurado y receipt durable; no meter proveedor en este paquete. |
| `orquesta-runtime-codex-delivery` | `AgentDeliveryObservationProviderPortV0`, `AgentProgressObservationProviderPortV0`, `ReviewGateObservationProviderPortV0`, `ExternalAgentLaunchSpecResolverPortV0` | `CodexDeliveryObservationSourceV0`, `CodexProgressObservationSourceV0`, `CodexReviewGateObservationSourceV0`, `CodexReceiptRecordingSpecResolverV0` (aserciones `var _` en los fuentes) | **Verificado** por aserciones de compilación y tests de ACK, worktree, review, progreso y scope `WaitAgentRefs`. | No se observó en esta inspección un único test que pruebe todas esas fuentes contra un proveedor externo real. **Acción:** añadir/identificar un smoke de composición con ACK+delivery+review causal y registrar su ref; mantener la promoción sin ACK como recuperación advisory, no como veto. |
| `orquesta-runtime-codex-appserver` | Adaptación composicional de launcher/observer Codex; puertos internos `ProbePortV0` y `ProtocolPortV0` | Backend tmux/WebSocket UDS, lazy protocol, workspace router, result marker/file y diagnósticos (`api_v0.go` y fuentes `codex_goal_app_server_*`) | **Verificado** por tests de migración, protocolo, timeout, startup/cleanup, router y placeholders; el contrato vigente exige `app_server_tmux` para operación normal. | El código soporta estados degradados cuando falta binario/socket/handshake. **Hipótesis/pending:** la disponibilidad real del proveedor no se demuestra leyendo el código. **Acción:** ejecutar smoke aislado `app_server_tmux` y conservar solo refs compactas de probe, launch, observe y stop. |
| `orquesta-state-file` | `RunStorePortV0`, `EventSinkPortV0`, readers de eventos, workflow task/wait, required-test evidence, plan state, `GoalWorkStateStorePortV0`, `GoalWorkStateListPortV0`, `AgentProcessRegistryPortV0` | `StoreV0` y subpaquete `outbox`; `app_director_goal_state_store_v0.go`, `workflow_wait_state_store_v0.go`, `required_test_evidence_store_v0.go`, etc. | **Verificado** por README, aserciones de uso y tests de persistencia/restart/CAS, incluidos tests multiproceso de goal state. | El inventario README no enumera explícitamente todos los puertos auxiliares recientes (por ejemplo marker store/attestation store), aunque hay implementación/tests. **Acción:** actualizar el inventario documental del módulo tras confirmar las aserciones públicas; no declarar hueco de código sin esa comprobación. |
| `orquesta-runtime-required-test` | `RequiredTestCommandExecutorPortV0` (implementado por el adaptador) y consumido por `RequiredTestRunnerPortV0` del núcleo | `LocalCommandExecutorV0`, política allowlist, output artifact y attestation local (`local_command_executor_v0.go`, `goal_attestation_local_v0.go`) | **Verificado** por tests de allowlist, entorno, redacción, retención y attestation; wiring opt-in comprobado en `cmd/orquesta-server/required_test_runner_env_v0_test.go` y `stack_wiring_test.go`. | La ejecución depende de configuración/allowlist. **Acción:** documentar en la composición el perfil mínimo habilitado para el test requerido y probar el comando contractual en un workspace limpio; no ampliar allowlist desde el adaptador. |
| Superficies `orquesta-mcp` MCP/HTTP | Puertos de transporte y ejecutores inyectados (`MCPTransportPortV0` y handlers/transport ports por superficie) | `RegisterMCPTransportV0`, registro de resources/tools, handlers HTTP JSON y transportes de observar goal, nueva app, workflow, status, VCS y external work | **Verificado** por `mcp_real_transport_*_test.go`, tests HTTP de handlers y validaciones de transporte; los envelopes declaran modo opt-in y budgets. | La superficie depende de que el bootstrap inyecte cada binding. **Hipótesis/pending:** el inventario de tests no prueba que cada tool se registre simultáneamente en el servidor arrancado. **Acción:** ejecutar un test/smoke de bootstrap que enumere resources/tools registrados y haga una llamada representativa MCP y HTTP por grupo. |
| `orquesta-app-codex-stack` + `cmd/orquesta-server` | Wiring de launcher/observer/state/required tests/delivery y shutdown; puertos neutralizados en `orquesta-goal` y `orquesta-orchestration-core` | `stack.go`, `codex_goal_app_server_v0.go`, `required_test_runner_stack_v0.go`, `goal_required_test_attestation_config_v0.go`; stores `StoreV0`; delivery file store; backend app-server | **Verificado** por `stack_wiring_test.go`, `codex_goal_app_server_wiring_v0_test.go`, tests de goal workspace/required tests y status/readiness. | El wiring tiene rutas opt-in y puede producir backend unavailable/degraded. **Hipótesis/pending:** no se puede afirmar cierre end-to-end del frente sin smoke real de la composición actual y sus receipts. **Acción:** ejecutar el smoke real acotado del backend configurado y el smoke MCP/bootstrap, enlazando refs en la matriz de pruebas vigente. |

## Puertos sin adaptador probado o con evidencia insuficiente

1. **No se ha demostrado un hueco de implementación neutral**: los puertos
   goal, state, required-test, delivery y transport tienen implementaciones o
   wiring local identificable.
2. **Sí hay huecos de evidencia de integración**: proveedor app-server real,
   registro simultáneo completo MCP/HTTP y recorrido delivery→review→closure.
   Son hipótesis/pending, no fallos funcionales confirmados.
3. La lectura no autoriza a convertir la ausencia de un smoke en “adaptador no
   existe”. La acción correcta es añadir/ejecutar la prueba de composición y
   conservar el artefacto durable, respetando el write-set de cada tarea.

## Evidencia focal ejecutable

- `go test -count=1 ./modulos/orquesta-estado-vivo` es el test requerido por
  T9201; su resultado debe acompañar al cierre.
- Para ampliar evidencia sin cambiar código: `go test -count=1` en los módulos
  del frente y los tests focales de `cmd/orquesta-server` citados arriba.

## Conclusión operativa

T9201 queda diagnosticado como frente con conectores implementados y
parcialmente probados, pero con cierre real pendiente de tres evidencias de
composición: backend app-server real, registro MCP/HTTP completo y ciclo
delivery/review/closure. Cada pendiente tiene acción y fichero de entrada
identificable. Este diagnóstico no reabre el loop histórico ni modifica código.
