# Pruebas locales: orquesta-mcp

Registra pruebas obligatorias del modulo.

## Plantilla

```text
Caso:
Tipo: unit | contract | integration | smoke
Comando:
Evidencia esperada:
Ultima ejecucion:
Riesgos:
```

## Pruebas previstas

```text
Caso: MCP-CT-030 apagado controlado de servidor por MCP/REST
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway
Evidencia esperada: `orquesta.server.shutdown.v0` queda publicado como tool
MCP opt-in, `POST /api/v0/server/shutdown` delega en el executor inyectado,
conserva `X-Correlation-ID`, devuelve `shutdown_ready`, contadores de runs,
agentes en vuelo, checkpoints pendientes, agentes pendientes de checkpoint y
`checkpoint_ref` por run cuando exista. Si falta ACK, proyecta
`pending_checkpoint_agent_refs` y evidencia compacta por run. Falla de forma
publica si falta binding. MCP/REST no paran procesos ni leen runtime/DB.
Ultima ejecucion: 2026-05-13; pasa en bateria focal junto a gateway.
Riesgos: el endpoint coordina stop/drain/stats; el borde que cierre el proceso
servidor debe invocarlo antes de enviar la senal final.
```

```text
Caso: MCP-CT-029 stats del director con progreso y cierre observables
Tipo: contract
Comando: go test -count=1 ./orquestacionnucleoapp ./modulos/orquesta-mcp
Evidencia esperada: `orquesta.director.stats.v0` lee solo `RunStorePortV0`,
puede enriquecer por `AgentProcessRegistryPortV0` y
`AgentProgressObservationProviderPortV0`, devuelve `DirectorRunStatsV0`
completo en `stats` con tareas, agentes, rework, replan, `stats.progress` y
`stats.closure` compactos; ademas devuelve `decision_context`
`DirectorDecisionContextV0` con progreso por fase/tarea/agente, procesos
opacos, actividad reciente, bloqueos, cierre, rework/replan, duraciones y
quietud. `/api/v0/director/stats` mantiene el mismo envelope para web/API/MCP
sin conocer internos.
Ultima ejecucion: 2026-05-10; pasa con `go test -count=1 ./modulos/orquesta-mcp`.
Riesgos: la politica exacta de cierre por `request_kind` vive en
`orquesta-app-director-service`; este tool expone estado observable del run.
```

```text
Caso: MCP-AUD-033 cobertura minima stats director/API/web
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./modulos/orquesta-http-gateway
Evidencia esperada: `MCPDirectorStatsToolResultV0.Stats` serializa
`DirectorRunStatsV0` completo y `decision_context` serializa la proyeccion
compacta de decision para director/API/MCP/web: counts/refs de tareas, agentes,
rework, replan, cierre, progreso, procesos/sesiones opacos, actividad reciente,
duraciones y quietud. La web llega a ese contrato por `/director-stats` usando
REST in-process contra `/api/v0/director/stats`.
Ultima ejecucion: 2026-05-10; pasa.
Riesgos: la proyeccion visual de `orquesta-web` puede elegir subconjuntos del
contrato; el contrato API permanece completo y estable.
```

```text
Caso: MCP-CT-028 ejecucion completa de app por MCP con receipts externos
Tipo: integration
Comando: go test -count=1 ./modulos/orquesta-mcp -run TestMCPEjecutarOrquestacionAppToolExecutorV0CompletaAppConReceiptsExternos
Evidencia esperada: `orquesta.apps.ejecutar_orquestacion.v0` recibe AppSpecV0,
prepara plan grande, arranca agentes por puertos inyectados, espera receipts
externos por `ExternalWaiter` y devuelve progreso completo sin que MCP conozca
scheduler, outbox, DB, runtime, proveedor, modelo ni HOME.
Ultima ejecucion: 2026-05-09; pasa.
Riesgos: usa receipts fake por puerto; el conector productivo de agentes reales
queda fuera del modulo.
```

```text
Caso: MCP-CT-027 tool de decision de director por MCP
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-mcp
Evidencia esperada: `orquesta.director_agent.apply_decision.v0` recibe
`DirectorAgentDecisionV0`, delega en `orquesta-director-agent-workflow`, aplica
`request_brainstorm` con store/sink inyectados; el puente cubre contratos y
microtareas con `TaskStore`; queda opt-in/unbound si falta puerto.
Ultima ejecucion: 2026-05-09; pasa.
Riesgos: No implementa servidor MCP real ni lector de ficheros de decision; el
transporte productivo debe inyectar puertos y fuente del director.
```

```text
Caso: MCP-CT-026 catalogo workflow con quality gate durable
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-mcp
Evidencia esperada: El resource incluye 32 comandos y 32 eventos sincronizados contra el core, `RecordQualityGate`, `QualityGateRecorded`, `RegisterPhaseArtifact`, `PhaseArtifactRegistered`, `quality_gates` y `phase_artifacts`; el tool puro registra el gate sin outbox y devuelve contador compacto.
Ultima ejecucion: 2026-05-09; pasa.
Riesgos: El servidor MCP real sigue pendiente; el resource y el tool son adaptadores puros sin persistencia productiva.
```

```text
Caso: MCP-CT-025 registro de transporte MCP opt-in
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp
Evidencia esperada: `RegisterMCPTransportV0` registra resources/tools existentes en un `TransportPortV0` fake, sirve un resource compacto y ejecuta `orquesta.operator.status.query.v0` solo por puerto fake inyectado.
Ultima ejecucion: 2026-05-07; pasa.
Riesgos: No prueba servidor MCP real, red productiva ni adaptadores externos; esa integracion queda opt-in fuera del modulo.
```

```text
Caso: MCP-CT-024 catalogo workflow con confirmacion de parada
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-mcp
Evidencia esperada: El resource incluye 30 comandos y 30 eventos sincronizados contra el core, `RegisterAgentStopConfirmed`, `AgentStopConfirmed` y la ref `confirmed_stopped_agents`.
Ultima ejecucion: 2026-05-06; pasa.
Riesgos: El servidor MCP real sigue pendiente; el resource es solo proyeccion contractual.
```

```text
Caso: MCP-CT-022 descriptor y tool puro de HandleCommandV0
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-mcp
Evidencia esperada: `orquesta.core_workflow.handle_command.v0` recibe current+command, invoca solo `HandleCommandV0`/`ApplyEventV0`, devuelve event types, outbox compacto y state_after compacto sin payloads ni refs extensas. Por defecto omite `state_public`; puede devolverlo solo con `include_state=true`.
Ultima ejecucion: 2026-05-06; pasa.
Riesgos: No cubre servidor MCP real, persistence productiva ni dispatch de outbox; esos conectores deben integrarse en cortes separados.
```

```text
Caso: MCP-CT-023 errores publicos de HandleCommandV0 por MCP
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-mcp
Evidencia esperada: Un comando invalido devuelve `estado=error` con code/field publicos del core y no inventa eventos, outbox ni state_after.
Ultima ejecucion: 2026-05-06; pasa.
Riesgos: Nuevos tipos de error del core deben mapearse sin acoplar MCP a internals.
```

```text
Caso: MCP-CT-001 schema del tool orquesta.apps.solicitar_nueva.v0
Tipo: contract
Comando: pendiente; futuro test de contrato del adaptador MCP
Evidencia esperada: El input MCP contiene app_spec_request compatible con AppSpecRequestV0 y respuesta opcional; rechaza campos de control no documentados.
Ultima ejecucion: no ejecutada; solo documentada el 2026-05-04
Riesgos: Falta schema canonico publicado de AppSpecRequestV0.
```

```text
Caso: MCP-CT-002 delegacion al puerto SolicitarNuevaApp v0
Tipo: integration
Comando: go test -count=1 ./modulos/orquesta-mcp
Evidencia esperada: El handler MCP llama al puerto, devuelve AppSpecV0 y BacklogInicialPropuestoV0, y no toca DB, CLI, filesystem ni runtime.
Ultima ejecucion: 2026-05-04; pasa contra `httptest` con `NewAppSpecHTTPHandlerV0`.
Riesgos: El slice sigue sin servidor MCP real; el adaptador local consume el puerto HTTP publicado.
```

```text
Caso: MCP-CT-003 serializacion de errores publicos
Tipo: contract
Comando: pendiente; futuro test parametrizado por error publico
Evidencia esperada: app_spec_invalida, opcion_incompatible, target_no_soportado, idioma_invalido y conector_requerido_no_disponible salen como output_error estable con code, message y field opcional.
Ultima ejecucion: no ejecutada; solo documentada el 2026-05-04
Riesgos: Puede cambiar el envelope si factory publica un formato de error canonico incompatible.
```

```text
Caso: MCP-CT-004 snapshot del resource contractual
Tipo: contract
Comando: pendiente; futuro snapshot de resource orquesta://contracts/solicitar-nueva-app/v0
Evidencia esperada: Resource compacto con propietario, input/output, errores e invariantes; sin dumps de DB ni detalles internos.
Ultima ejecucion: no ejecutada; solo documentada el 2026-05-04
Riesgos: Falta ubicacion definitiva del detalle canonico de SolicitarNuevaApp v0.
```

```text
Caso: MCP-CT-005 snapshot del prompt minimo
Tipo: contract
Comando: pendiente; futuro snapshot de prompt orquesta.solicitar_nueva_app.v0
Evidencia esperada: Prompt orienta a pedir datos faltantes, respeta i18n y no sugiere CLI/DB/runtime.
Ultima ejecucion: no ejecutada; solo documentada el 2026-05-04
Riesgos: Preguntas minimas no pueden cerrarse hasta conocer campos obligatorios de AppSpecRequestV0.
```

```text
Caso: MCP-SM-001 smoke MCP nueva app
Tipo: smoke
Comando: pendiente; futuro cliente MCP local invocando resource, prompt y tool
Evidencia esperada: Una IA puede leer el contrato, preparar una solicitud y obtener respuesta del puerto sin usar CLI/DB.
Ultima ejecucion: no ejecutada; solo documentada el 2026-05-04
Riesgos: Requiere servidor MCP implementado en una tarea futura; esta tarea no lo implementa.
```

```text
Caso: MCP-CT-015 ejecutor local HTTP del tool nueva app
Tipo: integration
Comando: go test -count=1 ./modulos/orquesta-mcp
Evidencia esperada: `MCPNuevaAppToolExecutorV0` hace `POST /api/v0/apps/spec`, fuerza `source=orquesta-mcp`, propaga `X-Correlation-ID` y devuelve `MCPNuevaAppToolResultV0` canonico.
Ultima ejecucion: 2026-05-04; pasa.
Riesgos: No cubre registro/servidor MCP real; solo adaptador local contra el puerto publicado.
```

```text
Caso: MCP-CT-016 errores locales de transporte/configuracion del ejecutor
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-mcp
Evidencia esperada: `server_url` con credenciales se rechaza; un 500 del puerto publicado se eleva como error Go local y no inventa errores de negocio nuevos.
Ultima ejecucion: 2026-05-04; pasa.
Riesgos: Los detalles del error local no forman parte del contrato MCP de negocio; solo protegen el adaptador.
```

```text
Caso: MCP-CT-006 descriptor puro del tool orquesta.apps.solicitar_nueva.v0
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-mcp
Evidencia esperada: Nombre, version, schema compacto, resource y prompt presentes; sin dumps de AppSpecV0 ni backlog.
Ultima ejecucion: 2026-05-04; pasa.
Riesgos: El schema detallado sigue referenciado a factory; MCP no lo renderiza.
```

```text
Caso: MCP-CT-007 mapper input MCP a AppSpecRequestV0
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-mcp
Evidencia esperada: Source se fija a orquesta-mcp; request_id/correlation_id se normalizan; campos clave se conservan sin reglas de negocio.
Ultima ejecucion: 2026-05-04; pasa.
Riesgos: La generacion de request_id si falta sigue pendiente de la capa de servidor o puerto posterior.
```

```text
Caso: MCP-CT-008 resultados compactos ok/error
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-mcp
Evidencia esperada: OK contiene proyecciones compactas de AppSpecV0 y BacklogInicialPropuestoV0; error publica ValidationIssue saneado y no inventa backlog.
Ultima ejecucion: 2026-05-04; pasa.
Riesgos: El formato puede ampliarse al integrar servidor MCP real, manteniendo compatibilidad v0 o versionando.
```

```text
Caso: MCP-CT-009 descriptor y resource compacto de contratos compartidos
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-mcp
Evidencia esperada: Descriptor `orquesta.contracts.shared.v0`, catalogo + 8 descriptors de contrato, summary/progress por message keys y URIs `orquesta://contracts/.../v0`.
Ultima ejecucion: 2026-05-04; pasa.
Riesgos: El resource no lee `../../CONTRATOS.md` en runtime; si el contrato global cambia, hay que versionar o actualizar el mapper puro.
```

```text
Caso: MCP-CT-010 saneamiento de resource y lookup de contratos compartidos
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-mcp
Evidencia esperada: Resource sin dumps de fixtures, endpoints REST, sink receipt, completions, OAuth, DSN ni tablas; lookup normalizado por nombre, slug y URI; mapper trim/dedupe.
Ultima ejecucion: 2026-05-04; pasa.
Riesgos: Las proyecciones de progreso son keys compactas locales, no estado real de servidor MCP ni observability.
```

```text
Caso: MCP-CT-011 descriptor y resource puro de roadmap de nucleo
Tipo: contract
Comando: go test -count=1 .
Evidencia esperada: Descriptor `orquesta.project.roadmap.v0`, resource `orquesta://project/roadmap/v0`, scope `orquesta-core`, 5 hitos de roadmap y 5 decisiones compactas con keys estables.
Ultima ejecucion: 2026-05-04; pasa.
Riesgos: El resource no lee docs globales ni docs de core en runtime; si el roadmap canonico cambia, hay que actualizar o versionar esta proyeccion.
```

```text
Caso: MCP-CT-012 lookup y saneamiento de roadmap de nucleo
Tipo: unit
Comando: go test -count=1 .
Evidencia esperada: Resource sin dumps de fixtures, endpoints REST, sink receipt, completions, OAuth, DSN ni rutas HOME; lookup normalizado por ID, area, contrato y key; mappers trim/dedupe.
Ultima ejecucion: 2026-05-04; pasa.
Riesgos: Las decisiones son compactas para IA y no sustituyen el detalle canonico de `orquesta-core/docs/contratos.md` ni `../CONTRATOS.md`.
```

```text
Caso: MCP-CT-013 descriptor y resource puro de operational status
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-mcp
Evidencia esperada: Descriptor `orquesta.observability.operational_status.v0`, resource `orquesta://observability/operational-status/v0`, contrato canonico enlazado a `orquesta://contracts/operational-status-query/v0`, endpoint recomendado `/api/v0/operational-status/query` y shape compacto de request/response.
Ultima ejecucion: 2026-05-04; pasa.
Riesgos: El resource no ejecuta diagnostico real ni verifica disponibilidad de proyecciones; solo documenta el contrato compartido y la invocacion recomendada.
```

```text
Caso: MCP-CT-014 lookup de consumers y saneamiento del resource operacional
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-mcp
Evidencia esperada: Lookup normalizado por modulo/canal permitido; el payload serializado no incluye secretos, transcripts, SQL, DSN, HOME ni detalles de filesystem productivo.
Ultima ejecucion: 2026-05-04; pasa.
Riesgos: Si cambian consumers/scopes/secciones del contrato canonico, este resource debe actualizarse o versionarse.
```

```text
Caso: MCP-CT-017 descriptor y resource puro de contratos core-workflow
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-mcp
Evidencia esperada: Descriptor `orquesta.core_workflow.contracts.v0`, resource `orquesta://core-workflow/contracts/v0`, 10 fases, 31 comandos, 31 eventos, cierre con `CloseRun`/`RunClosed`, quality gate durable y sincronizacion contra `SupportedOrchestrationCommandTypesV0`/`SupportedOrchestrationEventTypesV0`.
Ultima ejecucion: 2026-05-07; pasa con go test -count=1 ./modulos/orquesta-mcp.
Riesgos: El resource no ejecuta workflow ni lee docs en runtime; si el catalogo publico del core cambia el test debe fallar hasta actualizar la proyeccion MCP.
```

```text
Caso: MCP-CT-018 lookup y saneamiento de contratos core-workflow
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-mcp
Evidencia esperada: Lookup normalizado por comando/evento/summary_key derivada y payload serializado menor de 10 KB, sin secretos, SQL, DSN, OAuth, HOME, transcripts ni filesystem productivo.
Ultima ejecucion: 2026-05-06; pasa.
Riesgos: Solo valida descriptor local; no prueba servidor MCP real, DB, runtime, filesystem productivo ni transporte.
```

```text
Caso: MCP-CT-019 saneamiento estructural de resources roadmap/shared-contracts
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-mcp; git diff --check -- modulos/orquesta-mcp
Evidencia esperada: Los constructors, lookups y mappers conservan payloads y strings existentes; los ficheros Go tocados quedan por debajo de 300 lineas.
Ultima ejecucion: 2026-05-04; pasa.
Riesgos: Refactor mecanico; no cubre servidor MCP real ni lectura runtime de documentos canonicos.
```

```text
Caso: MCP-CT-020 descriptor/resource compacto de BootstrapProyectoDesdeAppSpec v0
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-mcp
Evidencia esperada: Descriptor `orquesta.director.bootstrap_appspec.v0`, resource `orquesta://contracts/bootstrap-proyecto-desde-appspec/v0`, refs canonicas al director y shape compacto sin payloads completos de AppSpec/backlog.
Ultima ejecucion: 2026-05-04; pasa.
Riesgos: Resource estatico; si el contrato canonico del director cambia, debe actualizarse o versionarse esta proyeccion.
```

```text
Caso: MCP-CT-021 tool puro BootstrapProyectoDesdeAppSpec v0
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-director
Evidencia esperada: `ExecuteMCPBootstrapToolV0` invoca solo `orquesta-director.BootstrapProyectoDesdeAppSpecV0`, devuelve JSON compacto con refs/contadores/eventos y serializa errores publicos sin inventar salida ok.
Ultima ejecucion: 2026-05-04; pasa.
Riesgos: No cubre servidor MCP real ni adaptadores futuros de persistencia, observability o runtime.
```

```text
Caso: MCP-CT-022 resource/tools operativos de operador
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-operator-mcp ./modulos/orquesta-mcp
Evidencia esperada: `orquesta.operator.operations.v0` lista capabilities de operador; los tools de status, burst, outbox y consulta dirigida validan entrada, fallan sin puerto y delegan solo por puertos inyectados.
Ultima ejecucion: 2026-05-06; pasa.
Riesgos: No cubre servidor MCP real, transporte, DB, runtime, filesystem, HOME, OAuth, proveedor ni modelo.
```

```text
Caso: MCP-CT-023 tool arrancar director de app
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-mcp
Evidencia esperada: `orquesta.apps.arrancar_director.v0` convierte AppSpecRequestV0 en StartAppDirectorV0, arranca director o equipo director con puertos fake, devuelve run/phase/task/director_tasks/started_agents compactos y traduce request/correlation externos con `mcp` a refs internas neutras.
Ultima ejecucion: 2026-05-09; pasa, incluyendo autonomia alta con batch de 4 directores y lotes maximos de 2.
Riesgos: No cubre Codex real ni servidor MCP real; el smoke real vive en `orquesta-runtime-codex-delivery` porque el proveedor es conector externo, no dependencia de MCP.
```

```text
Caso: MCP-CT-023A bridge REST arrancar director
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-web
Evidencia esperada: `NewMCPArrancarDirectorAppHTTPHandlerV0` acepta `POST /api/v0/apps/director`, invoca el executor inyectado, propaga correlation id y devuelve el resultado compacto consumido por `RESTArrancarDirectorAppClientV0`.
Ultima ejecucion: 2026-05-10; pasa.
Riesgos: No abre servidor real ni configura puertos productivos; solo fija el bridge HTTP.
```

```text
Caso: MCP-CT-024 tool preparar orquestacion de app
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-app-runner ./modulos/orquesta-app-planner
Evidencia esperada: `orquesta.apps.preparar_orquestacion.v0` recibe AppSpecV0 validada, delega en app-runner, devuelve run_ref, fase programacion, plan grande de 11 unidades, progreso inicial y error publico con field exacto si la AppSpec no es valida.
Ultima ejecucion: 2026-05-09; pasa.
Riesgos: No arranca agentes reales ni persiste runtime; es el contrato MCP seco para que el wizard/director pidan una preparacion sin conocer scheduler ni provider interno.
```

```text
Caso: MCP-CT-025 tool ejecutar orquestacion de app
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-app-runner ./modulos/orquesta-app-planner
Evidencia esperada: `orquesta.apps.ejecutar_orquestacion.v0` prepara el run, guarda estado inicial, ejecuta el loop progresivo por puertos fake, arranca `agent-agenda-bootstrap`, devuelve `wait_external` y no filtra refs externas con detalle de adaptador al core/outbox.
Ultima ejecucion: 2026-05-09; pasa.
Riesgos: No valida proveedor Codex/Claude/Gemini real; el conector productivo debe enchufarse por puertos y mantener las mismas reglas de neutralizacion.
```

```text
Caso: MCP-CT-026 bridge REST director stats
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-mcp
Evidencia esperada: `NewMCPDirectorStatsHTTPHandlerV0` acepta `POST /api/v0/director/stats`, delega en `MCPTransportDirectorStatsExecutorV0`, propaga `X-Correlation-ID`, responde 405 con `Allow: POST`, 503 sin executor, 400 con body invalido o `estado:error`, y 404 con JSON publico para path incorrecto.
Ultima ejecucion: 2026-05-10; pasa con go test -count=1 ./modulos/orquesta-mcp.
Riesgos: No abre servidor real ni configura RunStore/registry/progress source productivos; solo fija el bridge HTTP.
```
