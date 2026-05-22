# Tareas locales: orquesta-mcp

Cada tarea debe ser pequena y cerrada.

## Plantilla

```text
ID:
Objetivo:
Write-set:
Simbolo foco:
Contrato:
Validacion:
Bloqueos:
Estado:
```

```text
ID: MCP-034
Objetivo: Exponer herramienta MCP fina para trabajo de dominio externo.
Write-set: domain_work_*_v0.go, domain_work_*_v0_test.go, docs locales.
Simbolo foco: orquesta.domain_work.v0
Contrato: DomainWorkJobCreatorPortV0 y DomainWorkArtifactSubmitterPortV0.
Validacion: go test -count=1 ./modulos/orquesta-mcp -run TestMCPDomainWork.
Bloqueos: No registra el tool en `mcp_transport_registry_v0.go` porque esta
tarea no tiene ese fichero en write-set. No importa OPES, conector REST, DB,
runtime ni filesystem.
Estado: hecho
```

```text
ID: MCP-032
Objetivo: Registrar `orquesta.domain_work.v0` en el transporte MCP global.
Write-set: mcp_transport_registry_v0.go, mcp_transport_registry_v0_test.go,
docs locales.
Simbolo foco: mcp.transport.registry.v0
Contrato: MCPDomainWorkExecutorPortV0 y mcpDomainWorkTransportHandlerV0.
Validacion: go test -count=1 ./modulos/orquesta-mcp.
Bloqueos: No inyecta OPES ni conector REST; solo publica binding opt-in para
que un borde superior conecte el executor productivo.
Estado: hecho
```

## Backlog inicial

```text
ID: MCP-028
Objetivo: Exponer por MCP la aplicacion de decisiones compactas del director.
Write-set: director_agent_decision_tool_*.go, director_agent_decision_transport_v0.go, mcp_transport_registry_v0.go, docs locales.
Simbolo foco: orquesta.director_agent.apply_decision.v0
Contrato: DirectorAgentDecisionV0, ApplyDirectorAgentDecisionV0.
Validacion: go test -count=1 ./modulos/orquesta-mcp.
Bloqueos: No implementa servidor MCP real, lector de ficheros, runtime ni store productivo; todo queda por puertos inyectados.
Estado: hecho
```

```text
ID: MCP-020
Objetivo: Exponer control de app completa por MCP/REST sin acoplarse al core.
Write-set: run_control_*_v0.go, run_control_*_v0_test.go,
mcp_transport_registry_v0.go.
Simbolo foco: orquesta.runs.control.v0
Contrato: RunControlWriterPortV0
Validacion: go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-run-control.
Bloqueos: No implementa UI ni parada fisica; solo solicita estado por puerto.
Estado: hecho
```

```text
ID: MCP-021
Objetivo: Exponer ranking y cambio de prioridad multiapp por MCP/REST.
Write-set: run_queue_priority_*_v0.go, run_queue_priority_*_v0_test.go,
mcp_transport_registry_v0.go.
Simbolo foco: orquesta.run_queue.priority.v0
Contrato: RunQueueReaderPortV0 y RunQueuePriorityWriterPortV0
Validacion: go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-run-queue.
Bloqueos: El runner global productivo queda fuera; esta tarea expone el puerto
para director/web/MCP.
Estado: hecho
```

```text
ID: MCP-027
Objetivo: Reflejar `RegisterPhaseArtifact`/`PhaseArtifactRegistered` y `phase_artifacts` en el resource MCP del workflow.
Write-set: core_workflow_contracts_resource_v0.go, core_workflow_contracts_resource_v0_test.go, docs locales.
Simbolo foco: orquesta.core_workflow.contracts.v0
Contrato: SupportedOrchestrationCommandTypesV0, SupportedOrchestrationEventTypesV0, OrchestrationRunV0.
Validacion: go test -count=1 ./modulos/orquesta-mcp.
Bloqueos: No ejecuta observaciones de ACK desde MCP; solo expone el contrato core para gobierno por IA/MCP. El cableado operativo vive en runtime-delivery y nucleo-app.
Estado: hecho
```

```text
ID: MCP-017
Objetivo: Reflejar `RecordQualityGate`/`QualityGateRecorded` y `quality_gates` en los contratos MCP del workflow.
Write-set: core_workflow_contracts_resource_v0.go, core_workflow_contracts_resource_v0_test.go, core_workflow_command_tool_v0.go, core_workflow_command_tool_v0_test.go, docs locales.
Simbolo foco: orquesta.core_workflow.contracts.v0
Contrato: SupportedOrchestrationCommandTypesV0, SupportedOrchestrationEventTypesV0, OrchestrationRunV0.
Validacion: gofmt; go test -count=1 ./modulos/orquesta-mcp.
Bloqueos: No implementa servidor MCP real ni persistence; solo actualiza la proyeccion y el tool puro sobre puertos publicos.
Estado: hecho
```

```text
ID: MCP-016
Objetivo: Preparar frontera hexagonal para servidor/transporte MCP real opt-in sin implementar red productiva.
Write-set: mcp_transport_registry_v0.go, mcp_transport_registry_v0_test.go, docs/contratos.md, docs/tareas.md, docs/pruebas.md, docs/decisiones.md, ../orquesta-operator-mcp/docs/*
Simbolo foco: mcp.transport.registry.v0
Contrato: TransportPortV0, MCPTransportResourceEnvelopeV0, MCPTransportToolEnvelopeV0.
Validacion: go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp; git diff --check -- modulos/orquesta-mcp modulos/orquesta-operator-mcp.
Bloqueos: No abre sockets, no lee DB/outbox/runtime/event-store/filesystem productivo y no conoce HOME/OAuth/proveedor/modelo.
Estado: hecho
```

```text
ID: MCP-015
Objetivo: Registrar como contrato local los tools operativos de `orquesta-operator-mcp` ya expuestos por `orquesta-mcp`.
Write-set: operator_operations_resource_v0.go, operator_operations_tools_v0.go, operator_operations_tools_v0_test.go, docs/contratos.md, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Simbolo foco: orquesta.operator.operations.v0
Contrato: OperatorMCPCapabilitiesV0, OperatorMCP*PortV0
Validacion: 2026-05-06, ok, go test -count=1 ./modulos/orquesta-operator-mcp ./modulos/orquesta-mcp.
Bloqueos: No implementa servidor MCP real, red, DB, runtime, HOME, OAuth, proveedor ni modelo; usa puertos inyectados y errores publicos.
Estado: hecho
```

```text
ID: MCP-014
Objetivo: Reflejar `RegisterAgentStopConfirmed`/`AgentStopConfirmed` y `confirmed_stopped_agents` en el resource de contratos del workflow.
Write-set: core_workflow_contracts_resource_v0.go, core_workflow_contracts_resource_v0_test.go, docs locales.
Simbolo foco: orquesta.core_workflow.contracts.v0
Contrato: SupportedOrchestrationCommandTypesV0, SupportedOrchestrationEventTypesV0, OrchestrationRunV0.
Validacion: gofmt; go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-mcp.
Bloqueos: No implementa servidor MCP real ni persistence; solo actualiza la proyeccion que consume una IA.
Estado: hecho
```

```text
ID: MCP-013
Objetivo: Exponer un tool MCP puro para gobernar `HandleCommandV0` sin persistencia ni runtime.
Write-set: core_workflow_command_tool_v0.go, core_workflow_command_tool_v0_test.go, docs/contratos.md, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Simbolo foco: orquesta.core_workflow.handle_command.v0
Contrato: OrchestrationCommandV0, OrchestrationRunV0, OrchestrationCommandResultV0
Validacion: gofmt; go test -count=1 ./modulos/orquesta-mcp; ./scripts/verificar_nucleo_orquesta_v2.sh.
Bloqueos: No implementa servidor MCP real, persistence, DB, runtime, filesystem productivo ni dispatch de outbox; solo invoca core puro y devuelve salida compacta.
Estado: hecho
```

```text
ID: MCP-012
Objetivo: Sincronizar `orquesta.core_workflow.contracts.v0` con el catalogo publico completo del workflow tras NCW-048.
Write-set: core_workflow_contracts_resource_v0.go, core_workflow_contracts_resource_v0_test.go, docs/contratos.md, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Simbolo foco: orquesta.core_workflow.contracts.v0
Contrato: SupportedOrchestrationCommandTypesV0, SupportedOrchestrationEventTypesV0, mcp.resource.orquesta.core_workflow.contracts.v0.puro
Validacion: gofmt; go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-mcp; payload MCP menor de 10 KB.
Bloqueos: No implementa servidor MCP real, DB, runtime, filesystem productivo ni transporte; solo corrige la proyeccion para IA.
Estado: hecho
```

```text
ID: MCP-000
Objetivo: Fijar el alcance del primer slice MCP para nueva app.
Write-set: docs/contratos.md, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Simbolo foco: orquesta.apps.solicitar_nueva.v0
Contrato: SolicitarNuevaApp v0
Validacion: Documentacion local revisada contra ../../CONTRATOS.md.
Bloqueos: Ninguno para documentar el adaptador; bloqueado para implementar hasta tener schema canonico.
Estado: hecho
```

```text
ID: MCP-001
Objetivo: Definir el contrato local del tool MCP fino que envuelve SolicitarNuevaApp v0.
Write-set: docs/contratos.md
Simbolo foco: mcp.tool.orquesta.apps.solicitar_nueva.v0
Contrato: SolicitarNuevaApp v0
Validacion: El contrato no persiste, no crea tareas, no arranca runtime y no depende de CLI/DB.
Bloqueos: ninguno; schemas canonicos publicados en `orquesta-factory/docs/schemas/` y referenciados desde `../../CONTRATOS.md`.
Estado: hecho
```

```text
ID: MCP-002
Objetivo: Definir resource minimo para que una IA lea el contrato antes de invocar el tool.
Write-set: docs/contratos.md
Simbolo foco: orquesta://contracts/solicitar-nueva-app/v0
Contrato: SolicitarNuevaApp v0
Validacion: Resource compacto, sin dumps ni datos internos.
Bloqueos: ninguno; ubicacion canonica definida en `../../CONTRATOS.md`.
Estado: hecho
```

```text
ID: MCP-003
Objetivo: Definir prompt minimo para convertir una peticion humana en entrada del tool sin tocar CLI/DB.
Write-set: docs/contratos.md
Simbolo foco: mcp.prompt.orquesta.solicitar_nueva_app.v0
Contrato: SolicitarNuevaApp v0
Validacion: Prompt pide aclaraciones si faltan datos obligatorios y no inventa contratos.
Bloqueos: ninguno para documentacion; depende de implementacion futura para ejecutar validacion.
Estado: hecho
```

```text
ID: MCP-004
Objetivo: Preparar tests de contrato para schema MCP, serializacion de errores y snapshots de resource/prompt.
Write-set: docs/pruebas.md
Simbolo foco: orquesta.apps.solicitar_nueva.v0
Contrato: SolicitarNuevaApp v0
Validacion: Casos previstos en docs/pruebas.md.
Bloqueos: Implementacion futura del adaptador MCP y fixtures canonicas de factory.
Estado: hecho
```

```text
ID: MCP-005
Objetivo: Implementar el tool MCP como adaptador fino cuando exista el puerto publicado.
Write-set: solicitar_nueva_app_tool_executor_v0.go, solicitar_nueva_app_tool_v0_test.go, docs/contratos.md, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Simbolo foco: orquesta.apps.solicitar_nueva.v0
Contrato: SolicitarNuevaApp v0
Validacion: go test -count=1 ./modulos/orquesta-mcp; git diff --check -- modulos/orquesta-mcp.
Bloqueos: No implementa servidor MCP real, DB, runtime ni filesystem productivo; usa solo el puerto HTTP publicado por orquesta-factory.
Estado: hecho
```

```text
ID: MCP-005A
Objetivo: Arrancar implementacion pura y pequena del tool MCP orquesta.apps.solicitar_nueva.v0 sin servidor MCP real.
Write-set: solicitar_nueva_app_tool_v0.go, solicitar_nueva_app_tool_v0_test.go, docs/contratos.md, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Simbolo foco: orquesta.apps.solicitar_nueva.v0
Contrato: SolicitarNuevaApp v0 como DTOs publicos de orquesta-factory.
Validacion: go test -count=1 ./modulos/orquesta-mcp; git diff --check.
Bloqueos: No implementa servidor MCP real ni llamada al puerto; queda para corte posterior.
Estado: hecho
```

```text
ID: MCP-006
Objetivo: Implementar resource/descriptores compactos para que una IA consulte contratos compartidos v0 sin leer contexto global extenso.
Write-set: shared_contracts_resource_v0.go, shared_contracts_resource_v0_test.go, docs/contratos.md, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Simbolo foco: orquesta.contracts.shared.v0
Contrato: contratos compartidos v0 publicados en `../../CONTRATOS.md`: SolicitarNuevaApp, PersistenceRepository, RuntimeLaunchRequest, OrquestaEvent, GovernanceCatalog, DeploymentPlan y GenerarI18nDocsIniciales.
Validacion: go test -count=1 ./modulos/orquesta-mcp; git diff --check -- modulos/orquesta-mcp.
Bloqueos: No implementa servidor MCP real, transporte ni lectura productiva del filesystem; queda para corte posterior conectar estos descriptors al servidor MCP.
Estado: hecho
```

```text
ID: MCP-007
Objetivo: Implementar resource puro para que una IA lea roadmap y decisiones de nucleo sin cargar docs globales completos.
Write-set: project_roadmap_resource_v0.go, project_roadmap_resource_v0_test.go, docs/contratos.md, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Simbolo foco: orquesta.project.roadmap.v0
Contrato: mcp.resource.orquesta.project.roadmap.v0.puro
Validacion: go test -count=1 .; git diff --check -- .
Bloqueos: No implementa servidor MCP real, transporte, DB, runtime ni lectura de filesystem productivo; queda para corte posterior conectar el descriptor al servidor MCP.
Estado: hecho
```

```text
ID: MCP-008
Objetivo: Implementar un resource/descriptor puro, pequeno y testeado para que una IA consulte `OperationalStatusQuery v0`, su forma de invocacion y el shape compacto de `DiagnosticoCompactoV0`.
Write-set: operational_status_resource_v0.go, operational_status_resource_v0_test.go, docs/contratos.md, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Simbolo foco: orquesta.observability.operational_status.v0
Contrato: OperationalStatusQuery v0 + DiagnosticoCompactoV0
Validacion: go test -count=1 ./modulos/orquesta-mcp; git diff --check -- modulos/orquesta-mcp.
Bloqueos: No implementa tool, servidor MCP real, runtime, DB ni filesystem productivo; queda para corte posterior conectar este resource a un registro MCP.
Estado: hecho
```

```text
ID: MCP-009
Objetivo: Implementar resource/descriptor puro para que una IA consulte contratos compactos de `orquesta-core-workflow` tras NCW-025.
Write-set: core_workflow_contracts_resource_v0.go, core_workflow_contracts_resource_v0_test.go, docs/contratos.md, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Simbolo foco: orquesta.core_workflow.contracts.v0
Contrato: OrchestrationRunV0, OrchestrationCommandV0 y OrchestrationEventV0 hasta CloseRun/RunClosed.
Validacion: go test -count=1 ./modulos/orquesta-mcp; git diff --check -- modulos/orquesta-mcp.
Bloqueos: No implementa servidor MCP real, DB, runtime, filesystem productivo ni transporte; queda para corte posterior conectarlo a un registro MCP.
Estado: hecho
```

```text
ID: MCP-010
Objetivo: Sanear los resources rojos de roadmap y contratos compartidos dividiendolos por responsabilidades sin cambiar contratos MCP ni payloads.
Write-set: project_roadmap_resource_v0.go, shared_contracts_resource_v0.go, project_roadmap_*_v0.go, shared_contracts_*_v0.go, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Simbolo foco: orquesta.project.roadmap.v0; orquesta.contracts.shared.v0
Contrato: mcp.resource.orquesta.project.roadmap.v0.puro; mcp.resource.orquesta.contracts.shared.v0.puro
Validacion: gofmt; go test -count=1 ./modulos/orquesta-mcp; git diff --check -- modulos/orquesta-mcp; wc -l de ficheros Go tocados.
Bloqueos: Ninguno; refactor mecanico sin cambio de nombres publicos, payloads, errores ni comportamiento observado por tests.
Estado: hecho
```

```text
ID: MCP-011
Objetivo: Exponer BootstrapProyectoDesdeAppSpec v0 como contrato MCP compacto para IA, sin ejecutar persistencia/runtime.
Write-set: bootstrap_appspec_resource_v0.go, bootstrap_appspec_tool_v0.go, bootstrap_appspec_tool_v0_test.go, docs/tareas.md, docs/pruebas.md, docs/decisiones.md, docs/contratos.md
Simbolo foco: orquesta.director.bootstrap_appspec.v0
Contrato: BootstrapProyectoDesdeAppSpec v0 como DTOs publicos de orquesta-director.
Validacion: gofmt; go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-director; git diff --check -- modulos/orquesta-mcp; wc -l de ficheros Go tocados.
Bloqueos: No implementa servidor MCP real, persistencia, runtime, DB ni filesystem productivo; el tool puro invoca solo `orquesta-director.BootstrapProyectoDesdeAppSpecV0`.
Estado: hecho
```

```text
ID: MCP-012
Objetivo: Exponer `orquesta.apps.arrancar_director.v0` para que el formulario/wizard pueda pedir una app y dejar que Orquesta arranque el director por puertos.
Write-set: arrancar_director_app_tool_v0.go, arrancar_director_app_tool_executor_v0.go, arrancar_director_app_refs_v0.go, arrancar_director_app_tool_v0_test.go, mcp_transport_registry_v0.go, docs/*
Simbolo foco: orquesta.apps.arrancar_director.v0
Contrato: StartAppDirectorV0 como servicio canonico; AppSpecRequestV0 como entrada de negocio.
Validacion: gofmt; go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-app-director-service; git diff --check -- modulos/orquesta-mcp.
Bloqueos: No implementa servidor MCP real ni el wizard conversacional; el tool queda listo para transporte MCP/REST. El registro durable de artefactos del director en brainstorming se resuelve fuera de MCP mediante `RegisterPhaseArtifact`.
Estado: hecho
```

```text
ID: MCP-012A
Objetivo: Publicar bridge REST fino para que web consuma `orquesta.apps.arrancar_director.v0` sin importar MCP ni core.
Write-set: arrancar_director_app_http_v0.go, arrancar_director_app_tool_v0_test.go, docs/*
Simbolo foco: NewMCPArrancarDirectorAppHTTPHandlerV0
Contrato: POST /api/v0/apps/director como transporte REST del tool.
Validacion: gofmt; go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-web.
Bloqueos: No crea servidor productivo ni puertos reales; el caller debe inyectar executor con puertos configurados.
Estado: hecho
```

```text
ID: MCP-029
Objetivo: Exponer preparacion pura de orquestacion de app completa para que web/MCP obtengan run, plan grande y progreso inicial sin conocer scheduler interno.
Write-set: preparar_orquestacion_app_*_v0.go, mcp_transport_registry_v0.go, mcp_transport_operator_handlers_v0.go, preparar_orquestacion_app_tool_v0_test.go, docs/*
Simbolo foco: orquesta.apps.preparar_orquestacion.v0
Contrato: PrepareAppOrchestrationV0 como caso de uso canonico de orquesta-app-runner.
Validacion: gofmt; go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-app-runner ./modulos/orquesta-app-planner; git diff --check -- modulos/orquesta-mcp modulos/orquesta-app-runner.
Bloqueos: No implementa servidor MCP real ni conector productivo Codex/Claude/Gemini; solo prepara el run y expone plan/progreso compacto.
Estado: hecho
```

```text
ID: MCP-030
Objetivo: Exponer ejecucion de orquestacion de app por MCP para que web/MCP puedan pedir una app y dejar que el loop arranque agentes por puertos inyectados.
Write-set: ejecutar_orquestacion_app_*_v0.go, preparar_orquestacion_app_refs_v0.go, mcp_transport_registry_v0.go, docs/*
Simbolo foco: orquesta.apps.ejecutar_orquestacion.v0
Contrato: RunPreparedAppOrchestrationV0 como caso de uso operativo de orquesta-app-runner.
Validacion: gofmt; go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-app-runner ./modulos/orquesta-app-planner; git diff --check -- modulos/orquesta-mcp modulos/orquesta-app-runner.
Bloqueos: No implementa servidor MCP real ni conector productivo Codex/Claude/Gemini; esos conectores se inyectan como runtime/proveedor externo.
Estado: hecho
```

```text
ID: MCP-031
Objetivo: Publicar bridge REST fino para `orquesta.director.stats.v0` sin acoplar HTTP a RunStore, registry, progress source, DB, runtime, cmd ni procesos.
Write-set: director_stats_http_v0.go, director_stats_http_v0_test.go, docs/*
Simbolo foco: NewMCPDirectorStatsHTTPHandlerV0
Contrato: POST /api/v0/director/stats como transporte REST del tool.
Validacion: gofmt; go test -count=1 ./modulos/orquesta-mcp; git diff --check -- modulos/orquesta-mcp.
Bloqueos: No crea servidor productivo ni puertos reales; el caller debe inyectar `MCPTransportDirectorStatsExecutorV0` configurado.
Estado: hecho
```

```text
ID: MCP-AUD-033
Objetivo: Auditar si MCP/API ya exponen stats suficientes para tareas, agentes,
rework, replan, bloqueo de cierre y progreso sin que director/web conozcan
internos.
Write-set: docs/*
Simbolo foco: MCPDirectorStatsToolResultV0.Stats
Contrato: `Stats` es `DirectorRunStatsV0`; ya incluye counts/refs de tareas,
agentes, rework, replan, `progress` y `closure`.
Validacion: go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./modulos/orquesta-http-gateway.
Bloqueos: Ninguno en MCP/API; no se anaden campos porque el contrato ya usa el
DTO canonico completo.
Estado: hecho
```

```text
ID: MCP-032
Objetivo: Exponer `stats.closure` en `orquesta.director.stats.v0` para que el
director y la web vean si el cierre esta bloqueado, listo o cerrado.
Write-set: director_stats_tool_v0.go, director_stats_tool_v0_test.go,
director_stats_http_v0_test.go, docs/*
Simbolo foco: MCPDirectorStatsToolResultV0.Stats.Closure
Contrato: DirectorRunStatsV0 incluye `closure` derivado del run por
orquestacionnucleoapp.
Validacion: gofmt; go test -count=1 ./orquestacionnucleoapp ./modulos/orquesta-mcp; git diff --check -- modulos/orquesta-mcp.
Bloqueos: No consulta AppSpec/factory ni DB; las causas son genericas y
compactas para no acoplar MCP a politica de producto.
Estado: hecho
```

```text
ID: MCP-035
Objetivo: Exponer `orquesta.autoprogramming.prepare_run.v0` como contrato MCP/REST opt-in para preparar una run de autoprogramacion continuable sin arrancar agentes.
Write-set: autoprogramming_prepare_run_*_v0.go, mcp_transport_registry_v0.go, docs/*
Simbolo foco: orquesta.autoprogramming.prepare_run.v0
Contrato: executor inyectado prepara run y devuelve `run_ref`, `workflow_task_refs`, `wait_agent_refs` y `continue` compacto; la supervision posterior usa `run_ref` explicito.
Validacion: gofmt; go test -count=1 ./modulos/orquesta-mcp; git diff --check -- modulos/orquesta-mcp.
Bloqueos: No implementa servidor MCP real ni conoce `PrepareAutoprogrammingRunV0`; la composicion consumidora inyecta el executor concreto.
Estado: hecho
```

## Consultas

```text
CONSULTA AL DIRECTOR
Modulo origen: orquesta-mcp
Modulo afectado: orquesta-factory
Bloqueo: El contrato global publica SolicitarNuevaApp v0, pero no publica el schema de campos de AppSpecRequestV0, AppSpecV0 ni BacklogInicialPropuestoV0 en el contexto autorizado para este slice.
Pregunta concreta: Cual es la ubicacion canonica y versionada que debe consumir MCP para validar y documentar AppSpecRequestV0 sin duplicar reglas de factory?
Opcion recomendada: Publicar fixtures/schema v0 en orquesta-factory/docs/contratos.md o en un artefacto estable referenciado desde ../../CONTRATOS.md; MCP solo lo referenciara y probara compatibilidad de envelope.
Impacto: Sin esta definicion, orquesta-mcp puede documentar el envelope MCP, resources y prompts, pero no debe implementar validacion detallada ni ejemplos completos.
Decision del director: La ubicacion canonica versionada son los schemas y fixtures de `orquesta-factory/docs/schemas/` y `orquesta-factory/docs/fixtures/app_spec_v0/`, referenciados desde `../../CONTRATOS.md`. MCP no duplica reglas; solo referencia esos artefactos y valida su envelope. Consulta cerrada.
Estado: resuelta
```
