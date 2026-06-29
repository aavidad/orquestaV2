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

```text
Caso: MCP-CT-037 descriptor_source de resources T198 reconciliado
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp ./modulos/orquesta-observability ./modulos/orquesta-governance ./modulos/orquesta-core ./cmd/orquesta-server
Evidencia esperada: resources MCP registrados exponen `descriptor_source`
activo/stale/legacy con owner, fuente canonica, DTO/validador, errores
publicos y verificacion compacta, sin rutas locales ni datos sensibles.
Ultima ejecucion: 2026-05-27, ok en paquete
`agent-ref-task-autoprogramming-c3678e9bc306-g01`.
Riesgos: La prueba valida implementacion existente; esta entrada solo evita que
el backlog trate T198 cerrado como pendiente stale.
Revalidacion: `agent-ref-task-autoprogramming-c3678e9bc306-g01` debe declarar
el comando obligatorio del paquete como evidencia, sin reabrir codigo.
```

```text
Caso: MCP-CT-034 tool domain work por MCP/HTTP
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-mcp -run TestMCPDomainWork
Evidencia esperada: `orquesta.domain_work.v0` publica descriptor compacto,
ejecuta `create_job` solo por `DomainWorkJobCreatorPortV0`, ejecuta
`submit_artifact` solo por `DomainWorkArtifactSubmitterPortV0`, queda opt-in en
el transporte si falta puerto, expone `POST /api/v0/domain-work` como bridge
fino y reexporta `DomainWorkJobRecordSourcePortV0` cuando el backend lo soporta.
El test de arquitectura confirma que los ficheros `domain_work_*_v0.go` no
importan OPES ni conector REST.
Ultima ejecucion: 2026-05-13; pasa con bateria focal.
Riesgos: El conector productivo de OPES se inyecta desde un borde superior; el
tool no debe importar conectores reales ni crear stores.
```

## Pruebas previstas

```text
Caso: MCP-CT-038 supervise HTTP background deduplicado
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-mcp -run 'TestMCP.*SupervisorHTTPHandlerV0NoDuplicaOperacionActiva|TestMCP.*SupervisorHTTPHandlerV0DevuelveAcceptedSiExecutorSigueVivo|TestMCPRunSupervisorHTTPHandlerV0ClienteRealRecibeCuerpoSinColgar'
Evidencia esperada: `/api/v0/runs/supervise` y
`/api/v0/autoprogramming/supervise` devuelven `202 accepted_background` si el
executor sigue vivo; una segunda llamada con el mismo `operation_ref` no vuelve
a invocar el executor y devuelve diagnostico `*_operation_already_running`.
Tambien cubre body `{}` para cola global, observado en OPES, con
`operation-ref-*-queue`. Desde 2026-06-28 tambien cubre un cliente HTTP real
contra `/api/v0/runs/supervise` para verificar cuerpo JSON finito. Desde
2026-06-28 noche tambien valida flush HTTP en los `202 accepted_background` y
`poll_queue_global_status` como continuacion publica.
Ultima ejecucion: 2026-06-28; pasa con `go test -count=1 ./modulos/orquesta-mcp`.
Riesgos: el ledger es memoria del handler HTTP; tras restart la fuente de verdad
para progreso sigue siendo `director.stats`, `autoprogramming.status` y cola.
Cobertura servidor 2026-06-28: `go test -count=1 ./cmd/orquesta-server -run
'TestServer(AutoprogrammingSupervise|RunSupervise)HTTP.*SinColgarV0'` monta
`httptest.Server`, usa `http.Client` real y valida que el cliente recibe cuerpo
JSON `accepted_background` sin quedarse bloqueado en
`/api/v0/autoprogramming/supervise` y `/api/v0/runs/supervise`.
```

```text
Caso: MCP-CT-039 alias operativo de cola para autoprogramming supervise
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPAutoprogrammingStatusExecutorV0DeclaraSuperviseColaAunqueFaltenStatsDeRunV0|TestMCPAutoprogrammingSuperviseHTTPHandlerV0AceptaAliasesOperativos|TestMCPAutoprogrammingSuperviseHTTPHandlerV0AliasDispatchesAmpliaColaV0'
Evidencia esperada: en scope `run_ref`, `max_dispatches` conserva la semantica
de `max_dispatches_per_wait`; en scope de cola, el mismo alias rellena
`max_runs_per_tick` y `max_executions` cuando no vienen explicitados. El
`safe_action` de cola expone payload con `resident_mode=true` y presupuesto 70
para evitar una supervision global demasiado estrecha.
Ultima ejecucion: 2026-06-26; pasa con bateria focal.
Riesgos: el alias no cambia la capacidad real del runtime ni limites de
proveedor; solo evita que la ruta HTTP reduzca la ola por defaults de entrada.
```

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
quietud. Si se inyecta `GoalStateStore`, publica `goal` como proyeccion de
estado goal-first persistido sin observar ni cerrar el goal. `/api/v0/director/stats`
mantiene el mismo envelope para web/API/MCP sin conocer internos.
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
Caso: MCP-CT-028 briefing de supervisor por MCP
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-mcp ./cmd/orquesta-server -run 'TestMCPDirectorSupervisorBriefing|TestMCPRealTransportV0DirectorSupervisorBriefingJSONRPCV0'
Evidencia esperada: `orquesta.director_supervisor.briefing.v0` recibe
`DirectorSupervisorBriefingInputV0`, delega en `orquesta-director-supervisor`,
devuelve `DirectorSupervisorBriefingV0` con `next_action`, `action_queue` y
timeline compacta, y puede invocarse por JSON-RPC real sin CLI.
Ultima ejecucion: 2026-06-08; pasa.
Riesgos: es proyeccion pura; la aplicacion de la accion sigue fuera del modulo.
```

```text
Caso: MCP-CT-026 catalogo workflow con quality gate durable
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-mcp
Evidencia esperada: El resource incluye 32 comandos y 32 eventos sincronizados contra el core, `RecordQualityGate`, `QualityGateRecorded`, `RegisterPhaseArtifact`, `PhaseArtifactRegistered`, `quality_gates` y `phase_artifacts`; el tool puro registra el gate sin outbox y devuelve contador compacto.
Ultima ejecucion: 2026-06-29; pasa dentro de `go test -count=1 ./...`.
Riesgos: El transporte MCP real existe como opt-in en `cmd/orquesta-server`; este resource/tool siguen siendo adaptadores puros sin persistencia productiva.
```

```text
Caso: MCP-SM-REAL-001 transporte MCP real opt-in en servidor
Tipo: smoke
Comando: go test -count=1 ./cmd/orquesta-server -run TestMCPRealTransportSmokeOptInV0
Evidencia esperada: La composicion `cmd/orquesta-server` registra el transporte
MCP por `RegisterMCPTransportV0`, levanta un servidor HTTP JSON-RPC temporal en
loopback, ejecuta una lectura de `orquesta.operator.operations.v0`, ejecuta una
propuesta no destructiva de `orquesta.autoprogramming.self_improvement.propose.v0`
y normaliza la ausencia de operador real como `operator_mcp_port_unavailable`.
Ultima ejecucion: 2026-05-24; pasa con bateria focal del paquete T16.
Riesgos: El servidor MCP real sigue siendo opt-in de composicion; no se habilita
en `run` por defecto ni lee stores, runtime, HOME, proveedor, DB, Git/worktrees
o paths locales.
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
Ultima ejecucion: 2026-06-29; pasa dentro de `go test -count=1 ./...`.
Riesgos: El transporte MCP real existe como opt-in en `cmd/orquesta-server`; este resource sigue siendo solo proyeccion contractual.
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
Comando: go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPNuevaAppDescriptorV0Compacto|TestMCPTransportToolInputSchemaV0CubreToolsRegistrados'
Evidencia esperada: El input MCP contiene `app_spec_request` compatible con AppSpecRequestV0 y envelope local compacto; el registro de transporte comprueba que el descriptor no anuncia campos stale.
Ultima ejecucion: 2026-06-29; pasa.
Riesgos: El detalle extenso del schema sigue viviendo en `orquesta-factory`; MCP solo publica el envelope y no duplica reglas de negocio.
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
Comando: go test -count=1 ./modulos/orquesta-mcp -run TestNewMCPNuevaAppErrorResultV0SerializaErroresPublicosCanonicos
Evidencia esperada: app_spec_invalida, opcion_incompatible, target_no_soportado, idioma_invalido y conector_requerido_no_disponible salen como output_error estable con code, message y field opcional.
Ultima ejecucion: 2026-06-29; pasa.
Riesgos: Puede cambiar el envelope si factory publica un formato de error canonico incompatible; en ese caso debe actualizarse el DTO MCP junto al contrato owner.
```

```text
Caso: MCP-CT-004 snapshot del resource contractual
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-mcp -run 'TestNewMCPSharedContractsResourceV0CompactoYSinDumps|TestMCPTransportV0SirveResourceIndividualSolicitarNuevaApp'
Evidencia esperada: Resource compacto con propietario, input/output, errores e invariantes; sin dumps de DB ni detalles internos.
Ultima ejecucion: 2026-06-29; pasa.
Riesgos: El resource individual sirve una proyeccion compacta; el detalle canonico sigue en `orquesta-factory`.
```

```text
Caso: MCP-CT-005 snapshot del prompt minimo
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-mcp -run TestMCPNuevaAppPromptV0MinimoPideAclaracionesSinRuntime
Evidencia esperada: Prompt orienta a pedir datos faltantes, respeta i18n y no sugiere CLI/DB/runtime.
Ultima ejecucion: 2026-06-29; pasa.
Riesgos: Prompt puro para clientes MCP; no ejecuta negocio ni sustituye la validacion owner.
```

```text
Caso: MCP-SM-001 smoke MCP nueva app
Tipo: smoke
Comando: go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPTransportV0SirveResourceIndividualSolicitarNuevaApp|TestMCPNuevaAppPromptV0MinimoPideAclaracionesSinRuntime|TestMCPNuevaAppToolExecutorV0ReturnsOKResultFromFactoryHTTPPort'
Evidencia esperada: Una IA puede leer el contrato desde el transporte MCP opt-in, usar el prompt minimo y obtener respuesta del puerto HTTP local sin usar CLI/DB/runtime.
Ultima ejecucion: 2026-06-29; pasa con transporte fake local y `httptest`.
Riesgos: No cubre un servidor MCP de red real; ese transporte debe envolver este registro opt-in sin duplicar logica.
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
Ultima ejecucion: 2026-06-29; pasa.
Riesgos: Si el cliente no envia `request_id` ni `correlation_id`, el adaptador MCP genera `req-mcp-nueva-app-<hex>` antes de llamar al puerto factory; no genera idempotency-key ni cambia reglas de negocio.
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
Ultima ejecucion: 2026-05-24; pasa con freshness y refs de backlog T25.
Riesgos: El resource no lee docs canonicos en runtime; si un modulo propietario cambia contrato, hay que versionar o actualizar el mapper puro.
```

```text
Caso: MCP-CT-010 saneamiento de resource y lookup de contratos compartidos
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-mcp
Evidencia esperada: Resource sin dumps de fixtures, endpoints REST, sink receipt, completions, OAuth, DSN ni tablas; lookup normalizado por nombre, slug y URI; mapper trim/dedupe; no duplica `pendiente_*` sin backlog vivo.
Ultima ejecucion: 2026-05-24; pasa.
Riesgos: Las proyecciones de progreso son keys compactas locales, no estado real de servidor MCP ni observability.
```

```text
Caso: MCP-CT-011 descriptor y resource puro de roadmap de nucleo
Tipo: contract
Comando: go test -count=1 .
Evidencia esperada: Descriptor `orquesta.project.roadmap.v0`, resource `orquesta://project/roadmap/v0`, scope `orquesta-nucleo-reutilizable`, 5 hitos de roadmap, 5 decisiones compactas y freshness/backlog refs con keys estables.
Ultima ejecucion: 2026-05-24; pasa.
Riesgos: El resource no lee docs globales ni docs de core en runtime; si el roadmap canonico cambia, hay que actualizar o versionar esta proyeccion.
```

```text
Caso: MCP-CT-012 lookup y saneamiento de roadmap de nucleo
Tipo: unit
Comando: go test -count=1 .
Evidencia esperada: Resource sin dumps de fixtures, endpoints REST, sink receipt, completions, OAuth, DSN ni rutas HOME; lookup normalizado por ID, area, contrato y key; mappers trim/dedupe; hitos abiertos tienen owner, backlog refs y verificacion.
Ultima ejecucion: 2026-05-24; pasa.
Riesgos: Las decisiones son compactas para IA y no sustituyen el detalle canonico de docs vigentes y modulos propietarios.
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
Evidencia esperada: `NewMCPArrancarDirectorAppHTTPHandlerV0` acepta `POST /api/v0/apps/director`, invoca el executor inyectado, propaga correlation id y devuelve el resultado compacto consumido por `RESTArrancarDirectorAppClientV0`; un resultado `ok` sin `run_ref` se convierte en error publico `run_ref_requerido`.
Ultima ejecucion: 2026-05-10; pasa.
Riesgos: No abre servidor real ni configura puertos productivos; solo fija el bridge HTTP.
```

```text
Caso: MCP-CT-024 tool preparar orquestacion de app
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-app-runner ./modulos/orquesta-app-planner
Evidencia esperada: `orquesta.apps.preparar_orquestacion.v0` recibe AppSpecV0 validada, delega en app-runner, devuelve run_ref, fase programacion, plan grande de 11 unidades, progreso inicial, `route_policy` de preview/compatibilidad con `orquesta.apps.arrancar_director.v0` como preferente y error publico con field exacto si la AppSpec no es valida.
Ultima ejecucion: 2026-05-09; pasa.
Riesgos: No arranca agentes reales ni persiste runtime; es el contrato MCP seco para que el wizard/director pidan una preparacion sin conocer scheduler ni provider interno.
```

```text
Caso: MCP-CT-025 tool ejecutar orquestacion de app
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-app-runner ./modulos/orquesta-app-planner
Evidencia esperada: `orquesta.apps.ejecutar_orquestacion.v0` conserva compatibilidad legacy por puertos inyectados solo cuando el caller envia `director_execution_mode=legacy_director_loop`; con modo vacio o `goal_first` devuelve `legacy_director_loop_required` y `preferred_entrypoint=orquesta.apps.arrancar_director.v0` sin ejecutar el loop.
Evidencia esperada: en legacy explicito prepara el run, guarda estado inicial, ejecuta el loop progresivo por puertos fake, arranca `agent-agenda-bootstrap`, devuelve `wait_external`, no filtra refs externas con detalle de adaptador al core/outbox y bloquea con `director_v2_required` cuando el caller declara que necesita Director V2.
Ultima ejecucion: 2026-06-27; pasa.
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
# Prueba external-work-run 2026-05-13

Comando:

```bash
go test -count=1 ./modulos/orquesta-mcp \
  -run 'TestRegisterMCPTransportV0ExponeOperacionesExistentes'
```

Evidencia esperada: el registro MCP publica `orquesta.external_work.run.v0`
como tool opt-in. Sin executor productivo devuelve unbound por transporte; con
executor inyectado el handler REST delega en el caso de uso y el descriptor/
resultado declaran Goal-first como ruta normal para trabajo externo nuevo; la
compatibilidad legacy queda limitada a composicion opt-in con
`director_execution_mode=legacy_director_loop`.

## Prueba stats con run_ref obsoleto 2026-05-13

Comando:

```bash
go test -count=1 ./modulos/orquesta-mcp \
  -run TestMCPDirectorStatsToolExecutorV0RecuperaRunCanonicoSiRunRefObsoleto
```

Evidencia esperada: si `run_ref` no carga pero `external_job_ref` es valido,
`orquesta.director.stats.v0` resuelve la run canonica por puerto externo y
devuelve `estado=ok`, `run_ref` canonico y `external_job`.

Evidencia adicional: la proyeccion `external_job` conserva `status_reason`,
`issue_refs` y `diagnostics` emitidos por el puerto externo, manteniendolos como
diagnostico publico y no como logica de negocio MCP.

## Prueba prepare-run autoprogramming 2026-05-22

Comando:

```bash
go test -count=1 ./modulos/orquesta-mcp \
  -run 'TestMCPAutoprogrammingPrepareRun|TestMCPTransportV0AutoprogrammingPrepareRun'
```

Evidencia esperada: el registro MCP publica
`orquesta.autoprogramming.prepare_run.v0` como tool opt-in y salida opcional
`goal_spec_summaries`; sin executor devuelve `mcp_transport_tool_unbound`, con
executor inyectado invoca el puerto fake sin filtrar specs completas, y el bridge HTTP
`POST /api/v0/autoprogramming/prepare-run` delega sin conocer Codex, OPES,
runtime, DB ni filesystem productivo.

## Prueba automejora con consejo de operador 2026-05-23

Comando:

```bash
go test -count=1 ./modulos/orquesta-mcp
```

Evidencia esperada: `orquesta.autoprogramming.self_improvement.propose.v0`,
`/api/v0/autoprogramming/status` y `/api/v0/autoprogramming/supervise` aceptan
`operator_advice` compacto, aceptan aliases reparables de refs, accion y texto,
lo normalizan con `non_blocking=true` y no lo usan para bloquear, arrancar
runtime ni decidir proveedor. En self-improvement, el consejo normalizado queda
como contexto/regla no bloqueante de la request de automejora. Los puertos de
cola, stats y supervisor siguen siendo inyectados; si falta un puerto o falla el
executor, el error publico sigue siendo reparable y el bridge HTTP devuelve el
consejo normalizado como observacion no bloqueante sin descartar evidencia de
automejora.

Evidencia adicional 2026-06-26: `autoprogramming/status` separa una run
`running` sin stats recientes (`running_without_recent_stats`) de una run
stale sin proceso vivo (`running_stale_no_process`) y consulta
`director.stats` de forma acotada para detectar procesos vivos antes de
publicar `running_stale`. `queue_health.agents_live` publica ademas cuantos
agentes vivos se han observado por progreso o por `process.status`
`running`/`stopping`. Desde 2026-06-27, un `process_ref` sin status verificado
queda como `running_without_recent_stats`, no como vivo ni como stale terminal;
un `process.status=stopped` si permite publicar `running_stale_no_process`.
`autoprogramming/supervise` devuelve `202 accepted` con `operation_ref` y
diagnostico si el executor sigue vivo mas alla de la ventana HTTP, en vez de
dejar al cliente colgado.
Evidencia adicional: `director.stats` publica
`external_work_agent_requested_not_started` en `progress.issues` cuando una run
external-work tiene agentes pedidos, ninguno arrancado y nada en vuelo;
`autoprogramming/status` propaga ese issue como `diagnostics[]` para que la UI y
el operador no lo vean como `wait-subagents-terminal-without-delivery` generico.
Evidencia adicional 2026-06-26: el mismo diagnostico cubre tambien runs OPES
directas que llegan con refs `opes...`/`app-spec-opes...` aunque el nombre del
spec no incluya `external-work`; asi `director.stats` no cae al codigo generico
`agent_requested_not_started` para QA visual external-work lanzada desde OPES.

Evidencia adicional 2026-06-26: el bridge HTTP legacy `/api/v0/runs/supervise`
tambien devuelve `202 accepted_background` con `operation_ref`, diagnostico y
siguientes acciones si el executor ya fue delegado pero sigue vivo mas alla de
la ventana de respuesta. Es compatibilidad acotada para composiciones aun no
migradas a Goal/autoprogramming supervise; no anade logica al core ni relanza
trabajo por si mismo.

Evidencia adicional 2026-06-26: `autoprogramming/status` puede recibir un
`GoalWorkStateStore` opt-in. Si el `run_ref` pertenece a goal-first, publica
`autoprogramming_goal_first_observe_required` y sustituye acciones legacy de
supervision/retry/review de esa run por `observe_goal` contra
`/api/v0/autoprogramming/goal/observe`.

Evidencia adicional 2026-06-27:
`TestMCPAutoprogrammingStatusExecutorV0ColaMixtaGoalFirstNoSupervisaColaGlobal`
valida que, sin `run_ref` explicito, una cola mixta con candidato goal-first y
legacy no recomienda `supervise queue`; recomienda `observe_goal` para el
goal-first y `supervise run` para el candidato legacy concreto.
`TestMCPAutoprogrammingStatusExecutorV0ListaGoalActivoAunqueColaNoVisible`
valida que un `GoalWorkStateListPortV0` con un goal `running` basta para que
`autoprogramming/status` quede `ok` y recomiende `observe_goal` aunque no haya
cola visible.
Revalidacion adicional 2026-06-27: el mismo camino alinea
`ops_snapshot.decision` con la accion segura goal-first. Si hay `observe_goal`,
el snapshot publica `decision.action=observe_goal` y `reason_code=
goal_first_observe_required`, no `supervise_queue`.

Evidencia adicional 2026-06-28:
`TestMCPAutoprogrammingStatusExecutorV0GoalFirstMarkerSinStateNoSupervisaLegacy`
valida que un run con `GoalWorkRunMarkerV0` pero sin `GoalWorkStateV0` no cae al
supervisor legacy: publica diagnostico
`autoprogramming_goal_first_state_missing`, cuenta `queue_health.blocked=1`,
emite accion `goal_first_state_missing` con
`repair_goal_state_before_legacy_supervision`, oculta proyecciones legacy de
tareas/agentes y no recomienda `observe_goal`, `supervise run` ni
`supervise queue` hasta reparar el state. Tambien fija que
`ops_snapshot.decision.action=repair_goal_state` y no `wait_deliveries` aunque
existan stats legacy residuales con agentes en vuelo.
`TestMCPDirectorStatsToolExecutorV0GoalFirstMarkerSinStatePublicaRepairGoalState`
extiende la misma regla a `orquesta.director.stats.v0`: si hay marker goal-first
sin `GoalWorkStateV0`, `stats.status=goal_first_state_missing`,
`closure.blocked_by=goal_first_state_missing` y el snapshot de ops recomienda
`repair_goal_state`.
`TestMCPQueueGlobalStatusHTTPHandlerV0GoalFirstStateMissingRecomiendaRepararState`
fija que `/api/v0/queue/global-status` normaliza el item y la accion publica a
`goal_first_state_missing`/`repair_goal_state`, no a cancelar stale ni reencolar
legacy.
`TestMCPQueueGlobalStatusHTTPHandlerV0CadaRunVisibleTieneAccionORazon` fija el
contrato publico item por item: cada run visible publica una accion recomendada
si requiere operador o `no_action_reason` si puede esperar/cerrar sin accion.
`TestMCPQueueGlobalStatusHTTPHandlerV0AcceptedNoRequiereAccion` cubre que
`accepted` se clasifica como terminal/completado y no cae a
`repair_runtime`.

Evidencia adicional 2026-06-28:
`TestMCPAutoprogrammingStatusExecutorV0ProcesoParadoConAckCleanupEsperaACKV0`
fija que una run `running` con proceso verificado como `stopped` pero con
progreso en `ack_registered_cleanup` no recomienda reconciliacion agresiva:
publica `running_stale_no_process` con razon
`running_stale_no_live_process_pending_ack` y accion
`wait_for_ack_before_reconcile`. Tambien expone senales agregadas publicas
cuando hay stats causales: `process_alive_count`, `ack_detected`,
`last_ack_at`, `last_output_at` y `last_artifact_at`. La clasificacion neutral
sigue cubierta por `TestClassifyRunLivenessV0AckPendienteNoEsSeguroReconciliar`.

Evidencia adicional 2026-06-26: `autoprogramming/status` diagnostica
`external_work_accepted_stopped_without_delivery` cuando una run external-work
queda terminal `stopped` con evidencias de arranque/cola/coordinador, pero los
stats de la run muestran cero agentes arrancados, cero agentes en vuelo y cero
entregas.

Evidencia adicional 2026-06-27:
`TestMCPDirectorStatsToolExecutorV0DiagnosticaExternalWorkDoneSinAgenteMaterializado`
fija que `director.stats` publica
`external_work_accepted_no_agent_materialized` cuando una run external-work
queda terminal `done` sin agentes pedidos, arrancados, en vuelo ni entregas.
`TestMCPAutoprogrammingStatusExecutorV0DiagnosticaExternalWorkDoneSinAgenteMaterializadoV0`
valida que `autoprogramming/status` lo propaga como diagnostico y accion
`relaunch_or_replan_external_work_with_causal_error`, sin aceptar el cierre
silencioso como valido.

Evidencia adicional 2026-06-27:
`TestMCPAutoprogrammingStatusHTTPHandlerV0TimeoutDevuelveJSONPublico` y
`TestMCPRunQueuePriorityHTTPHandlerV0RankTimeoutDevuelveJSONPublico` fijan que
las lecturas HTTP de estado y cola devuelven `504` con JSON publico
decodificable si el executor no responde; no cuelgan el cliente ni dejan una
lectura opaca en background.
Evidencia adicional 2026-06-28:
`TestMCPRunQueuePriorityHTTPHandlerV0SetPriorityClienteRealRecibeTimeoutJSON`
fija la misma garantia para `set_priority`: el cliente HTTP real recibe `504`
con `run_queue_priority_timeout` y el puerto ve su contexto cancelado.
`TestMCPRunControlHTTPHandlerV0TimeoutDevuelveJSONPublico` y
`TestMCPExternalWorkRunHTTPHandlerV0TimeoutDevuelveJSONPublico` fijan la misma
garantia para control de run y external-work/run: `504` JSON publico,
correlacion preservada y cancelacion cooperativa del executor. En servidor
ensamblado lo cubren `TestServerRunControlHTTPClienteRealRecibeTimeoutJSONV0` y
`TestServerExternalWorkRunHTTPClienteRealRecibeTimeoutJSONV0`.
`TestMCPAutoprogrammingStatusExecutorV0DiagnosticaQueuedNotDispatchedV0` fija
`queued_not_dispatched` y `queue_health.queued_not_dispatched` para runs
`ready`/`queued`/`pending` sin goal-first ni dispatch observado.

## Prueba snapshot operativo del Director 2026-06-08

Comando:

```bash
go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-observability
```

Evidencia esperada: `orquesta.director.stats.v0` devuelve
`ops_snapshot` con `DirectorAutonomousOpsSnapshotV0` derivado de
`DirectorRunStatsV0` y `DirectorDecisionContextV0`; `orquesta.autoprogramming.status.v0`
agrega `queue` y `run` en el mismo snapshot. La decision contiene solo
`action`, `scope`, `reason_code` y refs opacas para UI/cockpit, sin arrancar
runtime, filtrar entregas ni leer Codex, OPES, DB, HOME o proveedor.

## Prueba stop_control publico en stats 2026-06-25

Comando:

```bash
go test -count=1 ./modulos/orquesta-orchestration-core ./modulos/orquesta-mcp ./modulos/orquesta-web
```

Evidencia esperada: `DirectorRunStatsV0.stop_control` distingue
`stop_requested`, `stop_propagated`, `stop_pending` y `stop_confirmed`.
`orquesta.director.stats.v0` enriquece esa proyeccion con `RunControl` si esta
inyectado, sin leer runtime ni procesos por su cuenta, y la web la proyecta
como estado de atencion cuando queda parada pendiente.

## Prueba observacion goal-first de nueva app 2026-06-25

Comando:

```bash
go test -count=1 ./modulos/orquesta-mcp \
  -run 'ObserveAppDirectorGoal|MCPTransportToolInputSchema|RegisterMCPTransport'
```

Evidencia esperada: `orquesta.apps.observe_director_goal.v0` queda registrado
como tool opt-in, `POST /api/v0/apps/director/goal/observe` acepta solo POST,
exige `run_ref`, propaga `X-Correlation-ID`, delega en el executor inyectado y
devuelve salida compacta con `run_ref`, `goal_ref`, `goal_status`,
`run_status`, `director_execution_mode`, `closure_status`, refs de
artefactos/evidencias y errores publicos. Sin executor devuelve
`mcp_transport_tool_unbound` por transporte.

## Prueba arranque app goal-first estricto 2026-06-27

Comando:

```bash
go test -count=1 ./modulos/orquesta-mcp \
  -run 'ArrancarDirectorApp.*GoalFirst|ToStartAppDirectorRequestV0TransportaDirectorExecutionMode|MCPTransportToolInputSchema'
```

Evidencia esperada: `orquesta.apps.arrancar_director.v0` transporta
`director_execution_mode` hasta `StartAppDirectorV0`; `goal_first` sin backend
Goal completo devuelve `errores_publicos.field=goal_backend_unavailable` por
HTTP y transporte MCP sin filtrar un error Go crudo; `legacy_director_loop`
queda como compatibilidad explicita y los resultados goal-first no arrancan
agentes legacy.

## Prueba observacion goal-first de autoprogramacion 2026-06-26

Comando:

```bash
go test -count=1 ./modulos/orquesta-mcp \
  -run 'AutoprogrammingObserveGoal|MCPTransportToolInputSchema|RegisterMCPTransport'
```

Evidencia esperada: `orquesta.autoprogramming.observe_goal.v0` queda registrado
como tool opt-in, `POST /api/v0/autoprogramming/goal/observe` acepta solo POST,
exige `run_ref`, delega en executor inyectado y devuelve la misma proyeccion
compacta de goal/cierre que la ruta de app-director. Sin executor devuelve
`mcp_transport_tool_unbound` por transporte.

## Prueba autoprogramacion observe active goals 2026-06-27

Comando:

```bash
go test -count=1 ./modulos/orquesta-mcp \
  -run 'AutoprogrammingObserveActiveGoals|RegisterMCPTransport|AutoprogrammingStatusExecutorV0PublicaAccionBatch'
```

Evidencia esperada: `orquesta.autoprogramming.observe_active_goals.v0` queda
registrado como tool opt-in, `POST
/api/v0/autoprogramming/goals/observe-active` delega en executor inyectado,
lista solo goals activos por defecto, conserva incidencias por run y
`autoprogramming/status` publica `observe_active_goals` cuando hay varios goals
activos.

## Prueba automejora goal-first 2026-06-27

Comando:

```bash
go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPAutoprogrammingSelfImprovement'
```

Evidencia esperada: automejora con `auto_prepare_run` conserva
`supervise_prepared_run_by_run_ref` solo cuando `prepare-run` devuelve rama
legacy con `run_ref`; si devuelve `goal`, recomienda
`observe_autoprogramming_goal`; si prepara specs internas, recomienda
`handoff_goal_first_specs_to_internal_backend`. No relanza supervisor legacy
para goal-first.
