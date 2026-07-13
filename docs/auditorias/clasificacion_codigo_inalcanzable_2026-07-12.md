# Clasificación de código inalcanzable — H4 (2026-07-12)

Fuente vinculante: `docs/auditorias/codigo_inalcanzable_2026-07-12.txt`, generado con `deadcode -test ./...`. Cada entrada aparece una sola vez y tiene exactamente una salida. La clasificación es documental: no se modifica código en H4.

| # | Entrada | Salida | Motivo técnico |
|---:|---|---|---|
| 1 | `cmd/orquesta-server/codex_goal_app_server_v0.go` — `codexAppServerIssueCodeForErrorV0` | BORRAR | Símbolo privado; `rg` no encuentra caller ni implementación de puerto/error ni reflexión real. |
| 2 | `cmd/orquesta-server/codex_goal_app_server_v0.go` — `codexAppServerIssueCodeFromCommandFailureV0` | BORRAR | Símbolo privado; `rg` no encuentra caller ni contrato verificable. |
| 3 | `cmd/orquesta-server/codex_goal_app_server_v0.go` — `codexAppServerIssueCodeFromLogFileV0` | BORRAR | Símbolo privado; `rg` no encuentra caller ni contrato verificable. |
| 4 | `cmd/orquesta-server/codex_goal_app_server_v0.go` — `codexAppServerTmuxStartupTimeoutV0` | BORRAR | Símbolo privado; `rg` no encuentra caller ni contrato verificable. |
| 5 | `modulos/orquesta-app-planner/unit_lookup_v0.go` — `FindAppPlanUnitByTaskRefV0` | RETIRAR | Wrapper exportado sin caller; la búsqueda causal ya vive en `findAppPlanUnitByTaskRefV0`, usada por `WorkProfileForUnitV0` y resolvers. `WorkProfileForUnitV0` conserva la validación de `task_ref` y `delivery_ref`. |
| 6 | `modulos/orquesta-cli/public_error_catalog_v0.go` — `CliPublicErrorCodeKnownV0` | BORRAR | Función sin caller verificable; ser exportada no demuestra consumidor externo ni contrato de puerto. |
| 7 | `modulos/orquesta-core-leases/lease_evaluator_v0.go` — `AgentLeaseEvaluationInputV0.Validate` | CONECTAR | Validación de entrada de leases; dejarla sin caller permite evaluar expiraciones con datos inválidos y pierde la garantía H2. |
| 8 | `modulos/orquesta-core-leases/lease_evaluator_v0.go` — `DecodeAgentTimeoutAssessmentV0` | RETIRAR | No existe frontera JSON de assessment: la ruta real produce el tipo validado en `EvaluateAgentLeaseV0`, lo transforma a `AgentLeaseExpiredV0` y valida ese contrato durable antes del replay. Serializar y decodificar internamente sería wiring cosmético. |
| 9 | `modulos/orquesta-core/puertos_salida_events_v0.go` — `OrquestaEventPublishErrorV0.Error` | CONSERVAR | Método requerido por `error`; deadcode no ve necesariamente la invocación polimórfica. |
| 10 | `modulos/orquesta-core/puertos_salida_governance_v0.go` — `GovernanceCatalogErrorV0.Error` | CONSERVAR | Implementación del contrato estándar `error`, consumible por callers externos. |
| 11 | `modulos/orquesta-core/puertos_salida_persistence_v0.go` — `PersistenceRepositoryErrorV0.Error` | CONSERVAR | Método de interfaz `error`; eliminarlo rompería el contrato de errores tipados. |
| 12 | `modulos/orquesta-core/puertos_salida_runtime_v0.go` — `RuntimeLaunchErrorV0.Error` | CONSERVAR | Método polimórfico del error de lanzamiento, no una función muerta semánticamente. |
| 13 | `modulos/orquesta-core-workflow/payload_budget_v0.go` — `ValidateOrchestrationEventPayloadBudgetV0` | CONECTAR | Garantía de tamaño de payload de eventos; debe ejecutarse antes de persistir/publicar eventos para evitar entradas fuera de presupuesto. |
| 14 | `modulos/orquesta-core-workflow/replay_v0.go` — `ValidateStrictEventSequenceV0` | CONECTAR | Garantía de causalidad del replay; debe validar secuencias antes de reconstruir estado, como exige H4. |
| 15 | `modulos/orquesta-data-ingestion-file/adapter_v0.go` — `AdapterV0.AdapterIdentityV0` | BORRAR | Método sin caller ni interfaz/registro verificable; la posibilidad de reflexión no es evidencia. |
| 16 | `modulos/orquesta-director-agent/director_decision_validation_v0.go` — `DirectorAgentDecisionValidV0` | CONECTAR | Validación de decisiones del director; sin cablearla se aceptan decisiones estructuralmente inválidas. |
| 17 | `modulos/orquesta-director-agent/director_stats_v0.go` — `DirectorAgentCompactStatsValidV0` | CONECTAR | Validación de estadísticas compactas recibidas del agente; debe proteger review/telemetría de datos corruptos. |
| 18 | `modulos/orquesta-director-agent-workflow/decision_source_budget_v0.go` — `DefaultDirectorAgentDecisionBatchBudgetV0` | BORRAR | Función default sin caller ni contrato verificable; el posible consumo de configuración no está demostrado. |
| 19 | `modulos/orquesta-director-supervised-burst/burst_adapters_v0.go` — `DirectorCycleStepExecutorFuncV0.ExecuteDirectorCycleStepV0` | CONSERVAR | Método de adaptador funcional que satisface un puerto; su uso es indirecto mediante interfaz. |
| 20 | `modulos/orquesta-director-supervised-burst/burst_adapters_v0.go` — `DirectorCycleStepInputBuilderFuncV0.BuildDirectorCycleStepInputV0` | CONSERVAR | Igual que la entrada anterior: método de función adaptadora invocado polimórficamente. |
| 21 | `modulos/orquesta-document-extraction-csv/exporter_v0.go` — `CSVDocumentExporterV0.AdapterIdentityV0` | BORRAR | Método sin caller ni interfaz/registro verificable; la posibilidad de reflexión no es evidencia. |
| 22 | `modulos/orquesta-document-extraction-fake/fake_v0.go` — `AcceptingDocumentHumanReviewV0.AdapterIdentityV0` | CONSERVAR | Identidad de fake para composición y pruebas de contrato; no es producción inalcanzable. |
| 23 | `modulos/orquesta-document-extraction-fake/fake_v0.go` — `AcceptingDocumentHumanReviewV0.ReviewDocumentFieldV0` | CONSERVAR | Método de puerto de revisión humana usado indirectamente por el caso de uso. |
| 24 | `modulos/orquesta-document-extraction-fake/fake_v0.go` — `StaticDocumentToolCapabilityRegistryV0.AdapterIdentityV0` | CONSERVAR | Identidad del registro fake, necesaria para sustituir el adaptador en tests. |
| 25 | `modulos/orquesta-document-extraction-fake/fake_v0.go` — `StaticDocumentToolCapabilityRegistryV0.DiscoverDocumentToolCapabilitiesV0` | CONSERVAR | Método de puerto de discovery; el análisis no sigue la llamada a través de la interfaz. |
| 26 | `modulos/orquesta-document-extraction-json/exporter_v0.go` — `JSONDocumentExporterV0.AdapterIdentityV0` | CONSERVAR | Identidad contractual del exporter JSON, no helper descartable. |
| 27 | `modulos/orquesta-document-extraction-tool-capability/adapter_v0.go` — `AttachDocumentExtractionToolV0` | CONECTAR | Registro de capability de extracción; debe componerse para que la tool declarada tenga binding efectivo. |
| 28 | `modulos/orquesta-document-extraction/tool_usecase_v0.go` — `documentToolPublicErrorV0` | BORRAR | Constructor privado; `rg` no encuentra caller ni mecanismo real de reflexión o contrato de puerto/error. |
| 29 | `modulos/orquesta-document-extraction/validation_v0.go` — `DefaultDocumentExtractionPolicyV0` | BORRAR | Función default sin caller ni contrato verificable; la configuración hipotética no basta. |
| 30 | `modulos/orquesta-document-extraction/validation_v0.go` — `ValidateDocumentConnectorRegistryV0` | CONECTAR | Validación del registry de conectores; debe ejecutarse en bootstrap para detectar bindings incompletos. |
| 31 | `modulos/orquesta-factory/architecture_policy_v0.go` — `SupportedArchitecturePatternsV0` | CONSERVAR | Catálogo público de patrones admitidos por factory; API declarativa, no código prescindible. |
| 32 | `modulos/orquesta-factory/request_policy_v0.go` — `SupportedExecutionModesV0` | CONSERVAR | Catálogo de modos de ejecución expuesto por la policy de requests. |
| 33 | `modulos/orquesta-factory/request_policy_v0.go` — `SupportedRequestKindsV0` | CONSERVAR | Catálogo de tipos de request para consumidores y validación futura del factory. |
| 34 | `modulos/orquesta-mcp/arrancar_director_app_tool_v0.go` — `NewMCPArrancarDirectorAppErrorResultV0` | BORRAR | Constructor sin caller verificable; la compatibilidad de transporte no está respaldada por una interfaz o uso real. |
| 35 | `modulos/orquesta-mcp/external_work_run_http_v0.go` — `NewMCPExternalWorkRunHTTPHandlerV0` | CONECTAR | Handler MCP/HTTP declarado; debe cablearse en la composición cuando el adaptador externo esté habilitado. |
| 36 | `modulos/orquesta-mcp/public_error_catalog_v0.go` — `MCPPublicErrorDescriptorV0` | CONSERVAR | Descriptor de catálogo público consumido por clientes MCP y generado por registro. |
| 37 | `modulos/orquesta-observability/orquesta_event_v0.go` — `DecodePublishOrquestaEventRequestV0` | CONECTAR | Decodificador de requests de publicación; debe preceder al sink para validar payload y contrato. |
| 38 | `modulos/orquesta-observability/orquesta_event_v0.go` — `HasOrquestaEventIssueV0` | CONECTAR | Detector de issues del evento; debe usarse para no publicar observaciones inválidas silenciosamente. |
| 39 | `modulos/orquesta-opes-bridge/document_plan_contract_v0.go` — `OPESGlobalEditorialPolicyV0` | CONSERVAR | Política editorial de dominio OPES, consultable por composición y no parte del núcleo genérico. |
| 40 | `modulos/orquesta-opes-bridge/document_plan_contract_v0.go` — `OPESHTMLPublicationPolicyV0` | CONSERVAR | Política de publicación HTML opt-in; contrato de consumidor externo. |
| 41 | `modulos/orquesta-opes-bridge/document_plan_contract_v0.go` — `OPESHTMLTopicTemplateV1` | CONSERVAR | Plantilla contractual para derivados HTML; puede materializarse por el adaptador. |
| 42 | `modulos/orquesta-opes-bridge/document_plan_contract_v0.go` — `OPESTemarioAgentRulesV0` | CONSERVAR | Reglas editoriales del consumidor OPES; conservar aunque el flujo concreto sea opt-in. |
| 43 | `modulos/orquesta-orchestration-core/candidate_provider.go` — `CandidateProviderFuncV0.BuildSchedulerCandidatesV0` | CONSERVAR | Método de adaptador funcional del puerto de candidatos; invocación indirecta por scheduler. |
| 44 | `modulos/orquesta-outbox-dispatch/outbox_delivery_lease_normalize_v0.go` — `normalizeLeaseReleaseV0` | CONECTAR | Normalización de release de lease; debe pasar por el camino de ACK/release para mantener idempotencia. |
| 45 | `modulos/orquesta-outbox-dispatch/outbox_delivery_lease_plan_v0.go` — `PlanOutboxDeliveryLeaseReleaseV0` | CONECTAR | Planifica liberación de leases del outbox; dejarlo aislado puede retener entregas y agentes. |
| 46 | `modulos/orquesta-outbox-dispatch/outbox_delivery_lease_plan_v0.go` — `validateObservedAtV0` | CONECTAR | Valida timestamp de observación antes de calcular lease; es una garantía temporal que debe estar en la ruta real. |
| 47 | `modulos/orquesta-presentation-extraction-tool-capability/adapter_v0.go` — `AttachPresentationExtractionToolV0` | CONECTAR | Binding de capability de presentación; debe cablearse cuando se anuncia esa tool. |
| 48 | `modulos/orquesta-rails/security_mode_v0.go` — `RailsModeV0` | CONSERVAR | Tipo/contrato de modo de seguridad usado por configuración y adaptadores. |
| 49 | `modulos/orquesta-rails/security_mode_v0.go` — `RailsOfflineV0` | CONSERVAR | Constante exportada del tipo `RailsModeV0`, usada como valor nominal del contrato de modos de seguridad. |
| 50 | `modulos/orquesta-rails/security_mode_v0.go` — `SecurityModeProductionEnabledV0` | CONSERVAR | Predicado público de modo productivo; debe mantenerse como API de decisión de seguridad. |
| 51 | `modulos/orquesta-rails/text_policy_v0.go` — `ValuesContainOperationalSensitiveDetailV0` | CONECTAR | Detector advisory de detalle sensible; debe alimentar auditoría/política sin convertir heurística en veto automático. |
| 52 | `modulos/orquesta-runtime-claude/claude_prompt_v0.go` — `BuildClaudeAgentPromptV0` | CONECTAR | Constructor base de prompt; debe ser el camino común del adaptador Claude para evitar prompts sin reglas. |
| 53 | `modulos/orquesta-runtime-claude/claude_prompt_v0.go` — `BuildClaudeAgentPromptWithControlFilesV0` | CONECTAR | Variante que incorpora control files; debe usarse cuando el spec declara esos refs. |
| 54 | `modulos/orquesta-runtime-claude/claude_prompt_v0.go` — `BuildClaudeAgentPromptWithLocaleV0` | CONECTAR | Variante localizada del prompt; debe conectarse al launcher Claude cuando hay locale en el contrato. |
| 55 | `modulos/orquesta-runtime-codex-appserver/api_v0.go` — `CodexAppServerIssueCodeFromCommandFailureV0` | BORRAR | Función exportada sin caller, interfaz ni contrato verificable; utilidad indirecta no demostrada. |
| 56 | `modulos/orquesta-runtime-codex-appserver/api_v0.go` — `CodexAppServerIssueCodeFromLogFileV0` | BORRAR | Función exportada sin caller, interfaz ni contrato verificable; utilidad indirecta no demostrada. |
| 57 | `modulos/orquesta-runtime-codex-appserver/api_v0.go` — `CodexAppServerTmuxOwnerMarkerPathV0` | CONECTAR | Deriva marker de ownership; debe usarse en lifecycle para evitar sesiones huérfanas o cruzadas. |
| 58 | `modulos/orquesta-runtime-codex-goal/packet_v0.go` — `BuildCodexGoalContextBudgetV0` | CONECTAR | Compila el presupuesto de contexto del goal; debe alimentar el packet para respetar límites declarados. |
| 59 | `modulos/orquesta-runtime-gemini/gemini_prompt_v0.go` — `BuildGeminiAgentPromptV0` | CONECTAR | Constructor base del prompt Gemini; debe asegurar reglas y cierre del goal en el adaptador. |
| 60 | `modulos/orquesta-runtime-gemini/gemini_prompt_v0.go` — `BuildGeminiAgentPromptWithControlFilesV0` | CONECTAR | Variante con archivos de control; debe cablearse al launcher cuando están declarados. |
| 61 | `modulos/orquesta-runtime-gemini/gemini_prompt_v0.go` — `BuildGeminiAgentPromptWithLocaleV0` | CONECTAR | Variante localizada; debe usarse desde la composición Gemini con locale efectivo. |
| 62 | `modulos/orquesta-runtime-worktree/control_paths_v0.go` — `IsWorktreeControlPathV0` | CONECTAR | Protección de rutas de control; debe aplicarse al validar write-set para no tocar metadatos del runtime. |
| 63 | `modulos/orquesta-server/server_resources_v0.go` — `NewServerResourcesV0` | CONECTAR | Constructor de recursos compartidos; debe ser la única composición de ports para evitar wiring parcial. |
| 64 | `modulos/orquesta-tool-capability-file/store_v0.go` — `ToolOperationFileStoreV0.GetToolOperationRecordByReceiptRefV0` | CONECTAR | Lectura por receipt necesaria para replay/idempotencia de operaciones de tools. |
| 65 | `modulos/orquesta-tool-capability-file/store_v0.go` — `ToolOperationFileStoreV0.ListToolOperationReceiptsV0` | CONECTAR | Listado de receipts necesario para auditoría y recuperación durable del store. |
| 66 | `modulos/orquesta-tool-capability/usecase_v0.go` — `UninstallGeneratedAppToolV0` | CONECTAR | Caso de uso de desinstalación declarado; debe exponerse por adaptador gobernado si existe la operación. |
| 67 | `modulos/orquesta-web/autoprogramming_prepare_run_client_v0.go` — `autoprogrammingStatusToolInputV0` | BORRAR | DTO privado sin referencias de uso en el repositorio; la búsqueda exacta solo encuentra su declaración. No hay contrato de reflexión ni consumidor externo documentado. |
| 68 | `modulos/orquesta-web/autoprogramming_prepare_run_client_v0.go` — `decodeAutoprogrammingStatusResponseV0` | CONECTAR | Decodificador de status; debe usarse para validar respuestas REST antes de proyectarlas. |
| 69 | `modulos/orquesta-web/autoprogramming_prepare_run_client_v0.go` — `normalizeAutoprogrammingStatusQueryV0` | CONECTAR | Normaliza query de status; debe proteger refs y estados equivalentes antes de enviar la petición. |
| 70 | `modulos/orquesta-web/autoprogramming_prepare_run_client_v0.go` — `RESTAutoprogrammingPrepareRunClientV0.ConsultarAutoprogrammingStatus` | CONECTAR | Operación REST declarada de consulta; debe cablearse en el cliente para completar el ciclo de observación. |
| 71 | `modulos/orquesta-web/bootstrap_client_v0.go` — `NewLocalBootstrapProyectoDesdeAppSpecClientV0` | CONECTAR | Cliente de bootstrap local declarado; debe integrarse en el wizard/app flow que lo ofrece. |
| 72 | `modulos/orquesta-web/nueva_app_director_client_v0.go` — `IsWebArrancarDirectorClientErrorCodeV0` | CONECTAR | Clasificación de errores del endpoint; debe alimentar respuestas/reintentos del cliente web. |
| 73 | `modulos/orquesta-web/nueva_app_i18n_v0.go` — `NuevaAppI18nTextV0` | BORRAR | Constructor sin caller verificable; preparación para claves futuras y consumo indirecto son hipótesis. |
| 74 | `modulos/orquesta-web/nueva_app_wizard_bot_endpoint_v0.go` — `NewWebNuevaAppWizardBotResponseV0` | CONECTAR | Constructor de respuesta del wizard bot; debe usarse en el handler para conservar schema estable. |
| 75 | `modulos/orquesta-web/public_error_catalog_v0.go` — `WebPublicErrorDescriptorV0` | CONSERVAR | Descriptor público de errores web, parte del catálogo de compatibilidad. |
| 76 | `modulos/orquesta-web/run_control_client_v0.go` — `IsWebRunControlClientErrorCodeV0` | CONECTAR | Clasifica errores de control de run; debe conectarse al cliente para distinguir reintento de bloqueo. |
| 77 | `modulos/orquesta-web/run_queue_client_v0.go` — `IsWebRunQueueClientErrorCodeV0` | CONECTAR | Clasifica errores de cola; debe usarse al proyectar respuestas del endpoint. |
| 78 | `cmd/orquesta-server/goal_first_app_http_flow_v0_test.go` — `goalFirstHTTPMCPClosureIssuesContainCodeForTestV0` | BORRAR | Helper test-only sin llamada en los tests ni en producción; la búsqueda exacta no encuentra referencias fuera de su declaración. |
| 79 | `cmd/orquesta-server/goal_first_app_http_flow_v0_test.go` — `postAutoprogrammingGoalFirstObserveForTestV0` | BORRAR | Fixture test-only sin llamada; no forma parte de un puerto ni de una API publicada. |
| 80 | `cmd/orquesta-server/goal_first_app_http_flow_v0_test.go` — `postAutoprogrammingGoalFirstStatusForTestV0` | BORRAR | Fixture test-only sin llamada; no forma parte de un puerto ni de una API publicada. |
| 81 | `cmd/orquesta-server/goal_first_app_http_flow_v0_test.go` — `postAutoprogrammingGoalFirstSuperviseForTestV0` | BORRAR | Fixture test-only sin llamada; no forma parte de un puerto ni de una API publicada. |
| 82 | `cmd/orquesta-server/idle_self_improvement_test_helpers_v0_test.go` — `minIntForTestV0` | BORRAR | Helper test-only sin llamada en el repositorio; no aporta un contrato de producción. |
| 83 | `modulos/orquesta-core/puertos_salida_v0_test.go` — `fakeGovernanceCatalogPortV0.ConsultarGovernanceCatalogV0` | CONSERVAR | Implementación de método de fake para satisfacer puerto en pruebas. |
| 84 | `modulos/orquesta-core/puertos_salida_v0_test.go` — `fakeOrquestaEventPublisherPortV0.PublicarOrquestaEventV0` | CONSERVAR | Método de fake del publisher; llamado indirectamente por el caso bajo prueba. |
| 85 | `modulos/orquesta-core/puertos_salida_v0_test.go` — `fakePersistenceRepositoryPortV0.GuardarProyectoBorradorV0` | CONSERVAR | Método de fake de persistencia requerido por interfaz. |
| 86 | `modulos/orquesta-core/puertos_salida_v0_test.go` — `fakeRuntimeLauncherPortV0.LanzarRuntimeV0` | CONSERVAR | Método de fake runtime requerido por el puerto y tests de composición. |
| 87 | `modulos/orquesta-domain-work/contracts_v0_test.go` — `fakeDomainWorkConnectorV0.CreateDomainWorkJobV0` | CONSERVAR | Método de fake contractual para probar creación sin backend externo. |
| 88 | `modulos/orquesta-domain-work/contracts_v0_test.go` — `fakeDomainWorkConnectorV0.ListDomainWorkJobRecordsV0` | CONSERVAR | Método de fake del puerto de listado; falso positivo por dispatch de interfaz. |
| 89 | `modulos/orquesta-domain-work/contracts_v0_test.go` — `fakeDomainWorkConnectorV0.SubmitDomainWorkArtifactV0` | CONSERVAR | Método de fake de entrega de artefactos; preserva contrato neutral. |
| 90 | `modulos/orquesta-run-control/contracts_v0_test.go` — `fakeRunControlPortV0.CancelRunV0` | CONSERVAR | Implementación de fake del control de cancelación. |
| 91 | `modulos/orquesta-run-control/contracts_v0_test.go` — `fakeRunControlPortV0.CompleteRunControlV0` | CONSERVAR | Implementación de fake del cierre de run. |
| 92 | `modulos/orquesta-run-control/contracts_v0_test.go` — `fakeRunControlPortV0.PauseRunV0` | CONSERVAR | Implementación de fake de pausa, necesaria para completar el puerto. |
| 93 | `modulos/orquesta-run-control/contracts_v0_test.go` — `fakeRunControlPortV0.ReadRunControlStateV0` | CONSERVAR | Implementación de fake de lectura de estado. |
| 94 | `modulos/orquesta-run-control/contracts_v0_test.go` — `fakeRunControlPortV0.RecordRunCheckpointV0` | CONSERVAR | Implementación de fake de checkpoint durable. |
| 95 | `modulos/orquesta-run-control/contracts_v0_test.go` — `fakeRunControlPortV0.ResumeRunV0` | CONSERVAR | Implementación de fake de reanudación. |
| 96 | `modulos/orquesta-run-control/contracts_v0_test.go` — `fakeRunControlPortV0.StopRunV0` | CONSERVAR | Implementación de fake de parada gobernada. |
| 97 | `modulos/orquesta-run-queue/contracts_v0_test.go` — `fakeRunQueueReaderPortV0.ListRunSchedulingCandidatesV0` | CONSERVAR | Método de fake del puerto de candidatos de scheduling. |
| 98 | `modulos/orquesta-server/supervisor_loop_test_helpers_v0_test.go` — `markNoExecutionWindowStartForTestV0` | BORRAR | Helper test-only sin llamada en el repositorio; la declaración no tiene consumidor verificable. |
| 99 | `modulos/orquesta-server/supervisor_loop_test_helpers_v0_test.go` — `resetNoExecutionWindowForTestV0` | BORRAR | Helper test-only sin llamada en el repositorio; la declaración no tiene consumidor verificable. |

## Resumen

- CONECTAR: 36 entradas.
- BORRAR/RETIRAR: 24 entradas (1–6, 8, 15, 18, 21, 28–29, 34, 55–56, 67, 73, 78–82, 98–99). Son símbolos sin caller ni contrato verificable; no se confunden con métodos de puerto ni errores tipados. La retirada de la fila 8 conserva la ruta tipada validada y el evento durable de expiración; la fila 5 conserva el helper privado y su validación causal en los consumidores existentes.
- CONSERVAR: 39 entradas.

## Método de evidencia

La lista de entradas se cotejó literalmente con `codigo_inalcanzable_2026-07-12.txt` (99/99, sin duplicados). Para cada fila se revisó la declaración y las referencias exactas con `rg`; una conservación solo se mantiene cuando la propia firma implementa un contrato (`error` o puerto), o es un tipo/constante contractual verificable. Las filas BORRAR tienen únicamente la declaración como referencia verificable. No se usa posibilidad de reflexión, uso futuro, compatibilidad hipotética ni “escenarios futuros” como motivo.

La prioridad de implementación posterior es conectar primero las garantías de validación (`ValidateStrictEventSequenceV0`, `ValidateOrchestrationEventPayloadBudgetV0`, leases, decisión del director y registries), y hacerlo en cambios separados con sus pruebas. Esta clasificación no autoriza esos cambios. La única excepción ejecutada en H4 es la retirada mínima de la fila 5, limitada al wrapper exportado sin caller.
