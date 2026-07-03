# Clasificacion T279 wip-remoto-71

Entrada triada:
- `docs/triaje_wip_remoto_2026-07-04/wip-remoto-71.patch`: 68 ficheros.
- `docs/triaje_wip_remoto_2026-07-04/wip-untracked.tgz`: 2 ficheros Go.

Metodo aplicado: se comparo cada ruta del patch y del tgz con el worktree actual sin aplicar el patch. Cuando el comportamiento existe en el arbol actual, el veredicto cita el fichero o test vigente. Cuando la ruta ya no existe, se marca `obsoleto`. Cuando el hunk viejo aporta una correccion todavia no cubierta y reimplementable sobre el codigo actual, se marca `valioso_reimplementar` y se referencia una tarea al final.

## Tabla

| fichero | area | veredicto | justificacion |
| --- | --- | --- | --- |
| `cmd/orquesta-server/codex_goal_app_server_tmux_v0.go` | appserver-goal | obsoleto | La ruta ya no existe; el backend tmux vive en `modulos/orquesta-runtime-codex-appserver/codex_goal_app_server_tmux_v0.go` y el `cmd` solo expone shims en `cmd/orquesta-server/codex_goal_app_server_v0.go`. |
| `cmd/orquesta-server/codex_goal_app_server_tmux_v0_test.go` | appserver-goal | obsoleto | La ruta ya no existe; la cobertura migrada esta en `modulos/orquesta-runtime-codex-appserver/codex_goal_app_server_migrated_v0_test.go` y wiring en `cmd/orquesta-server/codex_goal_app_server_wiring_v0_test.go`. |
| `cmd/orquesta-server/codex_goal_app_server_types_v0.go` | appserver-goal | obsoleto | La ruta ya no existe; los tipos actuales estan en `modulos/orquesta-runtime-codex-appserver/codex_goal_app_server_types_v0.go` y se reexportan desde `cmd/orquesta-server/codex_goal_app_server_v0.go`. |
| `cmd/orquesta-server/codex_goal_app_server_v0.go` | appserver-goal | ya_implementado | El `cmd` actual es shim y el fallback Goal RPC/read-thread esta implementado en `modulos/orquesta-runtime-codex-appserver/codex_goal_app_server_v0.go` con prueba base en `codex_goal_app_server_migrated_v0_test.go`. |
| `cmd/orquesta-server/codex_goal_app_server_v0_test.go` | appserver-goal | obsoleto | La ruta ya no existe; las pruebas equivalentes se migraron a `modulos/orquesta-runtime-codex-appserver/codex_goal_app_server_migrated_v0_test.go` y `cmd/orquesta-server/codex_goal_app_server_wiring_v0_test.go`. |
| `cmd/orquesta-server/effective_config_v0.go` | opes-bridge-config | ya_implementado | Remote QA ya sale en configuracion efectiva en `cmd/orquesta-server/effective_config_v0.go` y se cubre con `cmd/orquesta-server/opes_bridge_drain_test.go:TestOPESBridgeExternalCapabilitiesFromEnvV0DeclaraRemoteQA`. |
| `cmd/orquesta-server/goal_first_app_http_flow_v0_test.go` | goal-first-http | ya_implementado | El cierre terminal sin reabrir legacy y la reanudacion goal-first se cubren en `cmd/orquesta-server/goal_first_app_http_flow_v0_test.go:TestServerAppHTTPGoalFirstReanudaTrasRestartSinLegacyV0`. |
| `cmd/orquesta-server/opes_bridge_config.go` | opes-bridge-config | ya_implementado | Las capacidades `speech_synthesis` y `remote_qa_provider` con subchecks estan en `cmd/orquesta-server/opes_bridge_config.go` y tests `TestOPESBridgeExternalCapabilitiesFromEnvV0DeclaraRemoteQA`. |
| `cmd/orquesta-server/opes_bridge_drain_test.go` | opes-bridge-config | ya_implementado | La cobertura actual incluye remote QA, evidencia y subchecks en `cmd/orquesta-server/opes_bridge_drain_test.go`, mas estricta que el hunk viejo. |
| `cmd/orquesta-server/opes_operational_docs_guard_v0_test.go` | opes-doc-guard | valioso_reimplementar | El guard actual solo cubre parte de T12; faltan `decisiones.md` y `pruebas.md` del conector, que aun contienen narrativa stale; ver `WIP-TASK-002`. |
| `cmd/orquesta-server/opes_registry_finalpkg_producer_v0.go` | opes-finalpkg | ya_implementado | El requisito actual usa `manifest_cierre.json` y `rag/manifest.json` en `cmd/orquesta-server/opes_registry_finalpkg_producer_v0.go`, reemplazando el `manifest.json` viejo. |
| `cmd/orquesta-server/opes_registry_finalpkg_test.go` | opes-finalpkg | ya_implementado | Las pruebas actuales exigen `manifest_cierre.json` y claves estructuradas en `TestOPESRegistryFinalPkgPackageCompleteRequiresEvidenceManifestV0` y `RequiresStructuredClosureRefsV0`. |
| `cmd/orquesta-server/server_env_registry_v0.go` | opes-bridge-config | ya_implementado | El registro actual declara `ORQUESTA_OPES_BRIDGE_REMOTE_QA_*` y subchecks de red/auth/cuota en `cmd/orquesta-server/server_env_registry_v0.go`. |
| `docs/matriz_pruebas_reales_y_smoke_2026-05-17.md` | matriz-smokes | valioso_reimplementar | La fila `OPES-DER-DRAFT-AGENT` conserva formula anterior sin `app_server_tmux`; la fila debe alinearse con el cierre goal-first vigente; ver `WIP-TASK-001`. |
| `modulos/orquesta-app-codex-stack/README.md` | app-codex-stack | ya_implementado | El README actual ya documenta goal-first como camino normal y legacy solo explicito en `modulos/orquesta-app-codex-stack/README.md`. |
| `modulos/orquesta-app-codex-stack/autoprogramming_staging_promotion_e2e_v0_test.go` | app-codex-stack | ya_implementado | La propagacion de `evidence_refs` al resultado de drain esta en `TestCodexStackAutoprogrammingPromotionV0PropagaEvidenceRefsAResultadoDeDrainV0`. |
| `modulos/orquesta-app-codex-stack/codex_supervisor_stack_lifecycle_v0.go` | app-codex-stack | ya_implementado | El lifecycle actual consume `stackDrainQueueStatusAndEvidenceForCoordinatorV0` y conserva evidencias en `modulos/orquesta-app-codex-stack/codex_supervisor_stack_lifecycle_v0.go`. |
| `modulos/orquesta-app-codex-stack/operational_closure_domain_work_source_v0.go` | opes-closure | ya_implementado | El cierre OPES final exige evidencias completas mediante `codexStackOPESFinalPackageSubmissionEvidenceCompleteV0` y `manifest_cierre` en el codigo actual. |
| `modulos/orquesta-app-codex-stack/operational_closure_opes_domain_work_v0_test.go` | opes-closure | ya_implementado | Las pruebas actuales bloquean cierre sin `manifest_cierre` y aceptan cierre con manifest/evidencias en `TestOperationalClosureSourceV0NoCierraOPESFinalSinManifestDeCierre`. |
| `modulos/orquesta-app-codex-stack/run_coordinator_queue_status_v0.go` | run-coordinator | ya_implementado | El estado de cola con evidencias esta implementado en `stackDrainQueueStatusAndEvidenceForCoordinatorV0` y probado desde `domain_work_pending_completion_v0_test.go`. |
| `modulos/orquesta-app-codex-stack/run_coordinator_v0.go` | run-coordinator | ya_implementado | `DrainRunV0` anexa `evidenceRefs` del estado de cola en `modulos/orquesta-app-codex-stack/run_coordinator_v0.go`, cubierto por tests de coordinator/drain. |
| `modulos/orquesta-app-gateway/gateway_flow_v0_test.go` | app-gateway | ya_implementado | El timeout configurable de external-work run se cubre en `modulos/orquesta-app-gateway/gateway_flow_v0_test.go:TestExternalWorkRunAPIUsaTimeoutDeConfigV0`. |
| `modulos/orquesta-app-gateway/handler_v0.go` | app-gateway | ya_implementado | El handler usa `NewMCPExternalWorkRunHTTPHandlerWithResponseTimeoutV0` con `config.Timeout` en `modulos/orquesta-app-gateway/handler_v0.go`. |
| `modulos/orquesta-domain-work/artifact_contract_v0.go` | domain-work | ya_implementado | `DomainWorkArtifactContractV0` y canonicalidad/materializacion estan implementados en `modulos/orquesta-domain-work/artifact_contract_v0.go`. |
| `modulos/orquesta-domain-work/artifact_contract_v0_test.go` | domain-work | ya_implementado | La clasificacion canonico/derivado/evidencia esta cubierta por `TestExpectedDomainWorkArtifactContractForWorkKindV0ClasificaCanonicoDerivadoYEvidencia`. |
| `modulos/orquesta-domain-work/docs/contratos.md` | domain-work-docs | ya_implementado | El contrato de artefactos y remote QA ya esta documentado en `modulos/orquesta-domain-work/docs/contratos.md`. |
| `modulos/orquesta-domain-work/docs/pruebas.md` | domain-work-docs | ya_implementado | La cobertura de contratos y remote QA esta documentada en `modulos/orquesta-domain-work/docs/pruebas.md`. |
| `modulos/orquesta-domain-work/external_capability_v0.go` | domain-work | ya_implementado | `remote_qa_provider`, perfiles de red/auth/cuota y timeouts estan en `modulos/orquesta-domain-work/external_capability_v0.go`. |
| `modulos/orquesta-domain-work/external_capability_v0_test.go` | domain-work | ya_implementado | Remote QA y TTS estan cubiertos en `TestDomainWorkExternalCapabilityRequirementsV0RevisionRemotaRequierePerfilProveedor` y tests de TTS. |
| `modulos/orquesta-mcp/external_work_run_http_v0.go` | mcp-http | ya_implementado | El timeout por respuesta y default de 30s existen en `modulos/orquesta-mcp/external_work_run_http_v0.go`, cubiertos por timeout publico. |
| `modulos/orquesta-mcp/external_work_run_http_v0_test.go` | mcp-http | ya_implementado | La cancelacion y JSON publico de timeout se cubren en `TestMCPExternalWorkRunHTTPHandlerV0TimeoutDevuelveJSONPublico`. |
| `modulos/orquesta-opes-bridge/artifact_contract_map_v0_test.go` | opes-bridge | ya_implementado | La proyeccion de `DomainWorkArtifactContractV0` a `input_fields` OPES esta en `artifact_contract_map_v0_test.go`. |
| `modulos/orquesta-opes-bridge/docs/contratos.md` | opes-bridge-docs | ya_implementado | Contrato de artefactos, RAG regenerable, manifest final y seis subroles ya figuran en `modulos/orquesta-opes-bridge/docs/contratos.md`. |
| `modulos/orquesta-opes-bridge/docs/pruebas.md` | opes-bridge-docs | ya_implementado | La cobertura de contratos, RAG y seis subroles esta documentada en `modulos/orquesta-opes-bridge/docs/pruebas.md`. |
| `modulos/orquesta-opes-bridge/docs/tareas.md` | opes-bridge-docs | ya_implementado | El cierre final con `manifest_cierre.json` y evidencias obligatorias ya aparece en `modulos/orquesta-opes-bridge/docs/tareas.md`. |
| `modulos/orquesta-opes-bridge/mapper_refs_v0.go` | opes-bridge | ya_implementado | Los criterios de RAG regenerable y exclusion de corpus derivados estan en `mapper_refs_v0.go`, con pruebas en `mapper_v0_test.go`. |
| `modulos/orquesta-opes-bridge/mapper_v0.go` | opes-bridge | ya_implementado | El mapeo actual anade contrato de artefacto y refs de seis subroles, pero bloquea hasta `WorkflowTaskStore` real en vez de declarar `ready`. |
| `modulos/orquesta-opes-bridge/mapper_v0_test.go` | opes-bridge | ya_implementado | Los tests actuales cubren contrato de artefactos, seis subroles y bloqueo hasta workflow tasks en `TestBuildExternalWorkRunRequestV0BloqueaContratoSeisSubrolesHastaWorkflowTasks`. |
| `modulos/orquesta-opes-bridge/required_tests_v0.go` | opes-bridge | ya_implementado | `opes-final-package-manifest-*` y `manifest_cierre` estan en `modulos/orquesta-opes-bridge/required_tests_v0.go`. |
| `modulos/orquesta-opes-bridge/required_tests_v0_test.go` | opes-bridge | ya_implementado | La politica final exige `manifest_cierre`, checksums, validacion y matriz en `required_tests_v0_test.go`. |
| `modulos/orquesta-opes-connector/docs/decisiones.md` | opes-connector-docs | valioso_reimplementar | El documento aun dice que T12 esta bloqueada por falta de entorno temporal; debe reconciliarse con el cierre funcional 2026-06-28; ver `WIP-TASK-002`. |
| `modulos/orquesta-opes-connector/docs/pruebas.md` | opes-connector-docs | valioso_reimplementar | El documento aun afirma que el smoke real OPES sigue bloqueado; debe enlazar el runbook goal-first cerrado y dejar solo residuales; ver `WIP-TASK-002`. |
| `modulos/orquesta-opes-connector/docs/tareas.md` | opes-connector-docs | valioso_reimplementar | Aunque ya declara cierre funcional, debe incorporar las precondiciones vigentes `app_server_tmux`/scope duro y quedar bajo guard ampliado; ver `WIP-TASK-002`. |
| `modulos/orquesta-run-coordinator/coordinator_v0.go` | run-coordinator | ya_implementado | La compactacion y mezcla de `evidence_refs` del drainer estan en `modulos/orquesta-run-coordinator/coordinator_v0.go`. |
| `modulos/orquesta-run-coordinator/coordinator_v0_test.go` | run-coordinator | ya_implementado | La rotacion de cola con evidencias del drainer esta cubierta por `TestCoordinateRunsTickRotaColaConEvidenceRefsDelDrainerV0`. |
| `modulos/orquesta-run-coordinator/docs/contratos.md` | run-coordinator-docs | ya_implementado | La reconciliacion ExternalWork/DomainWork y distincion causal estan documentadas en `modulos/orquesta-run-coordinator/docs/contratos.md`. |
| `modulos/orquesta-runtime-codex/codex_profile_v0.go` | runtime-codex | ya_implementado | El issue code `codex_agent_contexto_requerido_no_materializado` existe en `codex_profile_v0.go`, aunque el flujo actual no lo usa como rail fuerte. |
| `modulos/orquesta-runtime-codex/codex_prompt_context_guard_v0_test.go` | runtime-codex | obsoleto | El hunk viejo bloqueaba pre-launch, pero el test actual `TestCodexExecResolverV0PromptAdvierteContextoRequiredRefOnly` exige no bloquear y advertir en prompt. |
| `modulos/orquesta-runtime-codex/codex_resolver_v0.go` | runtime-codex | obsoleto | El bloqueo `codexRequiredContextMaterializedIssuesV0` no existe porque el contrato actual conserva contexto requerido como recuperable por prompt/ACK. |
| `modulos/orquesta-runtime-codex/codex_resolver_v0_test.go` | runtime-codex | obsoleto | Los tests de bloqueo pre-launch contradicen la politica vigente documentada y probada de advertencia recuperable. |
| `modulos/orquesta-runtime-codex/docs/contratos.md` | runtime-codex-docs | obsoleto | El documento actual dice explicitamente que el resolver no bloquea por contexto truncado/ref_only y pide resolucion en ACK. |
| `modulos/orquesta-runtime-codex/docs/pruebas.md` | runtime-codex-docs | obsoleto | La cobertura actual documenta que el prompt advierte y no bloquea antes del runtime; reintroducir el hunk seria un rail fuerte stale. |
| `modulos/orquesta-server/runtime_v0.go` | server-shutdown | ya_implementado | El canal `shutdownReadyRequested` y parada por `http_shutdown_ready` estan en `modulos/orquesta-server/runtime_v0.go`. |
| `modulos/orquesta-server/shutdown_freeze_v0.go` | server-shutdown | ya_implementado | `requestShutdownReadyV0`, `stop_pending` y fuerza cooperativa estan implementados en `modulos/orquesta-server/shutdown_freeze_v0.go`. |
| `modulos/orquesta-server/shutdown_freeze_v0_test.go` | server-shutdown | ya_implementado | La parada dos fases esta cubierta por `TestRuntimeV0ServerShutdownReadyDetieneRuntimeHTTPV0`, `StopPendingNoPublicaReadyV0` y tests vecinos. |
| `modulos/orquesta-server/supervisor_loop_v0_test.go` | idle-self-improvement | ya_implementado | El encadenado goal-first de automejora residente esta cubierto por `TestRuntimeV0SelfAuditBacklogGoalFirstResidenteCierraYEncadenaSiguienteV0`. |
| `modulos/orquesta-web/nueva_app_endpoint_handler_v0.go` | web-nueva-app | ya_implementado | El limite de seis integraciones existe como `nuevaAppMaxIntegrationRowsV0 = 6` en `nueva_app_endpoint_handler_v0.go`. |
| `modulos/orquesta-web/nueva_app_endpoint_post_v0_test.go` | web-nueva-app | ya_implementado | El POST conserva seis integraciones en `modulos/orquesta-web/nueva_app_endpoint_post_v0_test.go`. |
| `modulos/orquesta-web/nueva_app_html_handler_v0_test.go` | web-nueva-app | ya_implementado | El HTML expone `integraciones.5.*` e `Integracion 6` en `nueva_app_html_handler_v0_test.go`. |
| `modulos/orquesta-web/nueva_app_html_render_v0.go` | web-nueva-app | ya_implementado | El render actual incluye seis filas de integraciones y campos ampliados de calidad/documentacion/i18n. |
| `modulos/orquesta-web/nueva_app_i18n_en_v0.go` | web-nueva-app | ya_implementado | Las claves inglesas de `Integration 5/6` y campos nuevos existen en `nueva_app_i18n_en_v0.go`. |
| `modulos/orquesta-web/nueva_app_i18n_es_v0.go` | web-nueva-app | ya_implementado | Las claves espanolas de `Integracion 5/6` y campos nuevos existen en `nueva_app_i18n_es_v0.go`. |
| `modulos/orquesta-web/nueva_app_i18n_keys_v0.go` | web-nueva-app | ya_implementado | Las claves obligatorias de integraciones 5/6 estan en `nueva_app_i18n_keys_v0.go`. |
| `modulos/orquesta-web/nueva_app_intake_guided_turn_v0.go` | web-nueva-app | ya_implementado | El guided turn usa `nuevaAppMaxIntegrationRowsV0` y genera decisiones para seis integraciones. |
| `modulos/orquesta-web/nueva_app_intake_guided_turn_v0_test.go` | web-nueva-app | ya_implementado | La sexta integracion esta cubierta por `TestGuidedCapabilityIntegrationDecisionsV0PermiteSeisFilasV0`. |
| `modulos/orquesta-web/nueva_app_intake_session_decision_v0.go` | web-nueva-app | ya_implementado | Las decisiones para calidad, documentacion, agentes e i18n ya se aplican en `nueva_app_intake_session_decision_v0.go`. |
| `modulos/orquesta-web/nueva_app_intake_session_v0.go` | web-nueva-app | ya_implementado | La captura de documentacion y campos ampliados esta en `nueva_app_intake_session_v0.go` y tests de session. |
| `scripts/smoke_goal_first_app_server_real.sh` | smoke-goal-first | ya_implementado | El script actual ya valida shutdown `app_server_tmux`, fallback de `ORQUESTA_CODEX_CODE_HOME` y resultado durable; guard en `smoke_goal_first_script_guard_v0_test.go`. |
| `modulos/orquesta-run-coordinator/external_work_reconciliation_v0.go` | run-coordinator-tgz | ya_implementado | El fichero del tgz ya existe trackeado y mas estricto: distingue ACK sin cierre, artefacto local, outbox, wait externo y proceso verificado. |
| `modulos/orquesta-run-coordinator/external_work_reconciliation_v0_test.go` | run-coordinator-tgz | ya_implementado | Las pruebas actuales anaden casos no incluidos en el tgz, como ACK sin cierre y running stale sin proceso vivo. |

## Resumen de veredictos

- `ya_implementado`: 56
- `valioso_reimplementar`: 5
- `obsoleto`: 9
- `dudoso`: 0

## Backlog ejecutable

## WIP-TASK-001
Objetivo: Actualizar la fila `OPES-DER-DRAFT-AGENT` de `docs/matriz_pruebas_reales_y_smoke_2026-05-17.md` para que refleje el flujo vigente goal-first con `ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux`, `external_goal_ref`, observacion/cierre Goal y legacy solo historico opt-in.
Estado: pendiente.
Alcance: write-set `docs/matriz_pruebas_reales_y_smoke_2026-05-17.md`; no tocar smokes ni codigo.
Criterios: la fila deja de describir "si hay backend" como cierre suficiente; menciona `app_server_tmux`; conserva advertencias de OPES temporal/no productivo; queda subordinada a `OPES-DER-RESTO` y `CODEX-GOAL-FIRST-APP-SERVER-REAL`.
Tests: `git diff --check -- docs/matriz_pruebas_reales_y_smoke_2026-05-17.md`; revision manual de la fila `OPES-DER-DRAFT-AGENT`.

## WIP-TASK-002
Objetivo: Reconciliar docs y guard de T12 OPES connector con el cierre funcional de 2026-06-28, eliminando narrativa stale de "smoke real sigue bloqueado" y extendiendo el guard a decisiones/pruebas/tareas del conector.
Estado: pendiente.
Alcance: write-set `modulos/orquesta-opes-connector/docs/decisiones.md`, `modulos/orquesta-opes-connector/docs/pruebas.md`, `modulos/orquesta-opes-connector/docs/tareas.md`, `cmd/orquesta-server/opes_operational_docs_guard_v0_test.go`.
Criterios: `decisiones.md` y `pruebas.md` apuntan al runbook `docs/runbooks/resultado_smoke_opes_derivados_goal_first_real_2026-06-28.md`; las precondiciones residuales exigen OPES temporal, scope duro y `app_server_tmux` cuando haya efectos; el guard falla si reaparecen frases de T12 bloqueada 2026-05-27 como estado vigente.
Tests: `go test -count=1 ./cmd/orquesta-server -run TestOPES.*DocsGuardV0`; `git diff --check -- cmd/orquesta-server/opes_operational_docs_guard_v0_test.go modulos/orquesta-opes-connector/docs`.
