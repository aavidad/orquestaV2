# Decisiones locales: orquesta-mcp

Las decisiones de este archivo solo afectan a `orquesta-mcp`. Si afectan a otro modulo, deben elevarse al director y registrarse en `../../CONTRATOS.md`.

## Plantilla

```text
Fecha:
Decision:
Motivo:
Alternativas:
Impacto:
Contratos afectados:
Estado:
```

## Decisiones tomadas

```text
Fecha: 2026-05-27
Decision: Tratar T198 como owner documental vigente de `descriptor_source` para resources MCP.
Motivo: Los intentos cerrados dejaron implementacion y pruebas focales, pero el
backlog podia seguir pareciendo pendiente para scanners posteriores.
Alternativas: Reabrir codigo; crear una tarea nueva duplicada por modulo.
Impacto: `orquesta-mcp` conserva el envelope/registro y cada modulo owner
mantiene DTO/validador/fuente canonica. La reconciliacion es documental y no
amplia el transporte ni los contratos de tools.
Contratos afectados: MCPTransportResourceEnvelopeV0; MCPResourceDescriptorSourceV0.
Estado: aceptada localmente
Revalidacion 2026-05-27: `agent-ref-task-autoprogramming-c3678e9bc306-g01`
conserva esta decision y no amplia el contrato.
```

```text
Fecha: 2026-05-13
Decision: Exponer `orquesta.external_work.run.v0` como tool MCP/REST propio.
Motivo: las apps externas como OPES necesitan pedir ejecucion de un job ya
definido sin arrancar el flujo de nueva app ni un director LLM inicial.
Alternativas: reutilizar `/api/v0/apps/director`; llamar a `request_change`
despues de crear el run manualmente; crear un endpoint OPES especifico. Se
descartan porque mezclan responsabilidades o acoplan Orquesta a una app.
Impacto: MCP delega en `orquesta-external-work-run`, queda opt-in por executor
inyectado y publica `POST /api/v0/external-work/run`.
Contratos afectados: mcp.tool.orquesta.external_work.run.v0;
rest.bridge.orquesta.external_work.run.v0; StartExternalWorkRunV0.
Estado: aceptada localmente
```

```text
Fecha: 2026-05-13
Decision: `orquesta.runs.control.v0` puede resolver la run asociada a
`external_job_ref`.
Motivo: OPES y otras apps externas pueden necesitar pausar, reanudar, parar o
cancelar el trabajo tecnico que Orquesta esta ejecutando para un job sin
interpretar la run completa.
Alternativas: exigir siempre `run_ref`; anadir control granular por task/agente;
crear un endpoint OPES especifico. Se evita el control granular en este corte
porque podria parar solo parte de una run compartida sin contrato claro.
Impacto: el input acepta `app_ref` y `external_job_ref`; el executor usa un
puerto de resolucion externo para obtener `run_ref` y aplica la accion normal
de RunControl. La accion controla la run asociada, no un bloque OPES aislado.
Contratos afectados: mcp.tool.orquesta.runs.control.v0;
rest.bridge.orquesta.runs.control.v0.
Estado: aceptada localmente
```

```text
Fecha: 2026-05-13
Decision: `orquesta.director.stats.v0` puede resolver una run por
`external_job_ref` mediante puerto inyectado.
Motivo: OPES dirige su propio backlog editorial y necesita preguntar por el
estado de un job concreto sin conocer la topologia interna de una run de
Orquesta.
Alternativas: obligar a OPES a guardar solo `run_ref`; crear un endpoint OPES
especifico dentro de Orquesta; importar app-change o OPES directamente desde
MCP.
Impacto: el input acepta `app_ref` y `external_job_ref`; el resultado puede
incluir `external_job` con refs compactas de job, task, agente, estado y
deliveries. MCP sigue siendo hexagonal porque la resolucion se delega en
`MCPDirectorExternalJobStatsSourcePortV0`.
Contratos afectados: mcp.tool.orquesta.director.stats.v0;
rest.bridge.orquesta.director.stats.v0.
Estado: aceptada localmente
```

```text
Fecha: 2026-05-13
Decision: `orquesta.domain_work.v0` se implementa como adaptador MCP fino sobre
puertos de `orquesta-domain-work`.
Motivo: el director necesita crear trabajo de dominio y entregar artefactos sin
conocer OPES, REST, DB, runtime ni conectores concretos.
Alternativas: importar el conector OPES desde MCP; registrar directamente un
cliente REST; esperar al servidor MCP real sin contrato local.
Impacto: el tool acepta `create_job` y `submit_artifact`, delega solo en
`DomainWorkJobCreatorPortV0` y `DomainWorkArtifactSubmitterPortV0`, y publica
handler MCP opt-in mas bridge HTTP local. El transporte central lo registra
como tool opt-in y devuelve `mcp_transport_tool_unbound` si falta executor.
Contratos afectados: mcp.tool.orquesta.domain_work.v0;
rest.bridge.orquesta.domain_work.v0; DomainWorkJobRequestV0;
DomainWorkArtifactSubmissionV0.
Estado: aceptada localmente
```

```text
Fecha: 2026-05-10
Decision: `orquesta.director.stats.v0` expone `DirectorRunStatsV0` completo
como contrato observable para director, API y web.
Motivo: el director necesita la misma informacion operativa que usa la
supervision humana para decidir si espera, replanifica, sube capacidad, para un
agente o bloquea el cierre; la web debe consultarlo sin conocer stores ni
runtime.
Alternativas: leer runtime/filesystem desde web; duplicar reglas del servicio
director en MCP; devolver solo contadores basicos.
Impacto: el tool lee el run por puerto, enriquece opcionalmente con registro de
procesos y observaciones de progreso, y publica counts/refs de tareas, agentes,
rework, replan, `progress` y `closure` compactos sin rutas, proveedor, modelo,
DB ni credenciales.
Contratos afectados: mcp.tool.orquesta.director.stats.v0;
rest.bridge.orquesta.director.stats.v0; DirectorRunStatsV0.
Estado: aceptada localmente
```

```text
Fecha: 2026-05-09
Decision: Exponer `orquesta.director_agent.apply_decision.v0` como tool MCP opt-in.
Motivo: para que Orquesta y el director se gobiernen solos, la IA directora debe poder devolver una decision compacta sin construir comandos internos del workflow a mano.
Alternativas: usar solo `orquesta.core_workflow.handle_command.v0`; leer ficheros de decision desde MCP; acoplar MCP al runtime del agente.
Impacto: MCP delega en `orquesta-director-agent-workflow`, usa puertos inyectados y devuelve un resultado compacto sin payloads completos.
Contratos afectados: DirectorAgentDecisionV0; ApplyDirectorAgentDecisionV0; mcp.transport.registry.v0.
Estado: aceptada localmente
```

```text
Fecha: 2026-06-08
Decision: Exponer `orquesta.director_supervisor.briefing.v0` como tool MCP puro.
Motivo: Hermes y otros directores externos necesitan consultar la siguiente
accion canonica del supervisor por API, sin CLI y sin interpretar structs
internos del burst.
Alternativas: depender solo del resultado de `run_supervisor`; duplicar la
proyeccion en web; acoplar el servidor al modulo de burst.
Impacto: MCP proyecta `DirectorSupervisorDecisionV0` a
`DirectorSupervisorBriefingV0`, sin aplicar efectos y sin conocer Codex, OPES,
runtime ni stores.
Contratos afectados: DirectorSupervisorBriefingV0; mcp.transport.registry.v0.
Estado: aceptada localmente
```

```text
Fecha: 2026-05-10
Decision: MCP-031 publica `POST /api/v0/director/stats` como bridge REST fino sobre `orquesta.director.stats.v0`.
Motivo: Web y adaptadores HTTP locales necesitan consultar stats del director sin conocer MCP ni crear puertos productivos dentro del handler.
Alternativas: Crear un handler que construya RunStore/registry/progress source; exponer un servidor MCP/HTTP completo; reutilizar un endpoint de app runner.
Impacto: `NewMCPDirectorStatsHTTPHandlerV0` solo decodifica JSON, delega en `MCPTransportDirectorStatsExecutorV0`, codifica JSON y propaga `X-Correlation-ID`.
Contratos afectados: rest.bridge.orquesta.director.stats.v0; mcp.tool.orquesta.director.stats.v0.
Estado: aceptada localmente
```

```text
Fecha: 2026-05-09
Decision: Preparar orquestacion de app por MCP se expone como tool puro separado de arrancar director.
Motivo: el wizard necesita obtener run, plan grande y progreso inicial sin inyectar stores/outbox ni conocer scheduler; arrancar director sigue siendo el flujo operativo con puertos.
Alternativas: ampliar solicitar_nueva_app; ampliar arrancar_director_app; duplicar planner dentro de MCP.
Impacto: `orquesta.apps.preparar_orquestacion.v0` delega solo en `orquesta-app-runner`, no expone CandidateProvider, mantiene payload compacto y no decide DB/runtime/proveedor/modelo.
Contratos afectados: mcp.tool.orquesta.apps.preparar_orquestacion.v0.puro; PrepareAppOrchestrationV0; AppMicrotaskPlanV0.
Estado: aceptada localmente
```

```text
Fecha: 2026-05-25
Decision: La ruta publica preferente de AppSpec es `orquesta.apps.arrancar_director.v0`.
Motivo: la foto vigente exige juicio del Director V2 para apps nuevas; el plan fijo de `orquesta-app-runner` no demuestra plan-state, waits por ola/cohorte, review/tests/cierre ni recursion.
Impacto: `preparar_orquestacion` y `ejecutar_orquestacion` publican `route_policy` de preview/compatibilidad; `ejecutar_orquestacion` bloquea con `director_v2_required` si el caller declara esa necesidad.
Contratos afectados: mcp.tool.orquesta.apps.arrancar_director.v0; mcp.tool.orquesta.apps.preparar_orquestacion.v0; mcp.tool.orquesta.apps.ejecutar_orquestacion.v0; RunPreparedAppOrchestrationV0.
Estado: aceptada localmente
```

```text
Fecha: 2026-05-09
Decision: Ejecutar orquestacion de app por MCP es opt-in y neutraliza refs externas antes de tocar core/outbox.
Motivo: el wizard necesita una orden unica para que Orquesta arranque el loop, pero el core prohibe detalles de transporte como mcp, web, cli, provider, HOME o runtime en referencias internas y outbox.
Alternativas: relajar validaciones del core; cambiar tests para evitar refs con mcp; acoplar MCP directamente a outbox. Se descartan porque reintroducen los fallos de v1/v2 y rompen la frontera hexagonal.
Impacto: `orquesta.apps.ejecutar_orquestacion.v0` exige executor/puertos inyectados, usa refs internas neutras con hash de refs externas y delega en `RunPreparedAppOrchestrationV0`.
Contratos afectados: mcp.tool.orquesta.apps.ejecutar_orquestacion.v0; PrepareAppOrchestrationV0; RunPreparedAppOrchestrationV0; mcp.transport.registry.v0.
Estado: aceptada localmente
```

```text
Fecha: 2026-05-07
Decision: El resource de workflow MCP expone el quality gate durable publicado por el core.
Motivo: Una IA directora necesita ver `RecordQualityGate` y `QualityGateRecorded` para gobernar validaciones de calidad sin leer internals del core ni deducirlas desde replay.
Alternativas: Esperar al servidor MCP real; dejar el catalogo en 30/30; tratar quality gates como detalle privado de revisiones.
Impacto: El resource pasa a 31 comandos y 31 eventos, incorpora `quality_gates` al shape compacto y mantiene sincronizacion contra los catalogos publicos del core.
Contratos afectados: orquesta.core_workflow.contracts.v0, RecordQualityGate, QualityGateRecorded.
Estado: aceptada localmente
```

```text
Fecha: 2026-05-09
Decision: MCP-012 anade `orquesta.apps.arrancar_director.v0` como adaptador fino hacia `orquesta-app-director-service`.
Motivo: El wizard/formulario debe poder pedir una app por MCP/REST sin conocer factory, workflow, scheduler, outbox ni runtime. El servicio canonico ya concentra esa composicion por puertos.
Alternativas:
  - Llamar al factory desde MCP y que el operador arranque agentes manualmente: descartado porque no prueba Orquesta.
  - Acoplar MCP al launcher Codex: descartado porque proveedor/modelo/HOME son conectores externos.
  - Pasar request_id/correlation_id del adaptador tal cual al core: descartado porque filtra detalles como mcp/web/api en refs internas.
Impacto: MCP traduce refs externas a refs internas neutras, delega en `StartAppDirectorV0` y devuelve una respuesta compacta con run, fase y director arrancado.
Contratos afectados: mcp.tool.orquesta.apps.arrancar_director.v0; StartAppDirectorV0; AppSpecRequestV0.
Estado: aceptada localmente
```

```text
Fecha: 2026-05-10
Decision: MCP-012A expone `/api/v0/apps/director` como bridge REST del tool `orquesta.apps.arrancar_director.v0`.
Motivo: `orquesta-web` necesita un conector REST/API sustituible para arrancar el director desde el wizard sin importar MCP ni construir puertos internos.
Alternativas: Que web importe el executor MCP; crear servidor productivo dentro de `orquesta-mcp`; mantener solo el registro MCP opt-in.
Impacto: El bridge recibe el mismo envelope del tool, usa executor inyectado y devuelve el mismo resultado compacto. El servidor productivo sigue fuera del modulo.
Contratos afectados: rest.bridge.orquesta.apps.arrancar_director.v0; mcp.tool.orquesta.apps.arrancar_director.v0.
Estado: aceptada localmente
```

```text
Fecha: 2026-05-07
Decision: El transporte MCP real se prepara como puerto de registro opt-in y no como servidor dentro de `orquesta-mcp`.
Motivo: La frontera hexagonal debe permitir enchufar un servidor MCP externo sin que este modulo abra sockets, lea estado productivo o conozca internals.
Alternativas: Implementar servidor MCP real aqui; registrar herramientas directamente contra DB/outbox/runtime; esperar a otro corte sin contrato de transporte.
Impacto: `RegisterMCPTransportV0` publica resources/tools existentes mediante envelopes compactos y handlers neutros. Los tools con dependencia externa usan puertos inyectados o quedan `mcp_transport_tool_unbound`.
Contratos afectados: mcp.transport.registry.v0.puro; orquesta.operator.operations.v0; orquesta.core_workflow.handle_command.v0; orquesta.director.bootstrap_appspec.v0; orquesta.apps.solicitar_nueva.v0.
Estado: aceptada localmente
```

```text
Fecha: 2026-05-06
Decision: El resource de workflow MCP expone tambien la confirmacion durable de parada.
Motivo: Una IA directora necesita distinguir stop solicitado (`stopped_agents`) de stop confirmado (`confirmed_stopped_agents`) para no replanificar sobre una suposicion falsa.
Alternativas: Esperar al servidor MCP real; dejar que la IA deduzca la confirmacion desde ACK de outbox; ocultar la nueva transicion hasta scheduler.
Impacto: El resource pasa a 30 comandos y 30 eventos y conserva sincronizacion automatica contra el catalogo publico del core.
Contratos afectados: orquesta.core_workflow.contracts.v0, RegisterAgentStopConfirmed, AgentStopConfirmed.
Estado: aceptada localmente
```

```text
Fecha: 2026-05-06
Decision: MCP-015 registra `orquesta.operator.operations.v0` como adaptador MCP puro sobre puertos de `orquesta-operator-mcp`.
Motivo: Una IA debe poder descubrir y pedir estado, burst supervisado, outbox pendiente y consultas dirigidas sin conocer internals del nucleo ni abrir un servidor MCP real en este corte.
Alternativas: conectar transporte MCP real ahora; hacer que MCP lea event-store/outbox directamente; duplicar reglas del director en MCP.
Impacto: `orquesta-mcp` queda como adaptador fino: valida envelopes, exige puerto inyectado y devuelve errores publicos. El transporte MCP real queda como conector opt-in posterior.
Contratos afectados: orquesta.operator.operations.v0; OperatorMCPStatusPortV0; OperatorMCPBurstPortV0; OperatorMCPOutboxPortV0; OperatorMCPDirectedQueryPortV0.
Estado: aceptada localmente
```

```text
Fecha: 2026-05-06
Decision: MCP expone un tool puro `orquesta.core_workflow.handle_command.v0` para gobernar comandos del workflow sin efectos externos.
Motivo: El cierre del nucleo exige que una IA pueda gobernar por MCP sin leer DB ni conocer runtime; un resource contractual no basta si no existe una forma testeada de aplicar comandos al core.
Alternativas: Esperar al servidor MCP real; ejecutar CLI; persistir desde MCP; exponer payloads completos de eventos/outbox.
Impacto: El tool recibe estado y comando publicos, invoca `HandleCommandV0`, aplica eventos con `ApplyEventV0` solo para devolver `state_after` compacto, y no ejecuta outbox ni guarda estado. Puede devolver `state_public` solo con `include_state=true` para harness o servidor MCP sin persistence productiva.
Contratos afectados: mcp.tool.orquesta.core_workflow.handle_command.v0.puro; OrchestrationCommandV0; OrchestrationRunV0; OrchestrationCommandResultV0.
Estado: aceptada localmente
```

```text
Fecha: 2026-05-06
Decision: El resource `orquesta.core_workflow.contracts.v0` se sincroniza contra los catalogos publicos del core y no mantiene una lista antigua paralela.
Motivo: El nucleo habia avanzado hasta stop, lifecycle, leases, concurrencia, resultado de revision, rework, replan y respuestas del director, pero MCP seguia exponiendo 18 comandos/eventos. Una IA externa no puede gobernar con un mapa incompleto.
Alternativas: Mantener 18/18 hasta servidor MCP real; copiar a mano comandos nuevos sin test contra core; leer docs en runtime.
Impacto: El resource expone el catalogo completo de comandos/eventos; los tests comparan contra `SupportedOrchestrationCommandTypesV0` y `SupportedOrchestrationEventTypesV0`; summary_key por transicion se deriva en lookup para mantener payload menor de 10 KB.
Contratos afectados: mcp.resource.orquesta.core_workflow.contracts.v0.puro; OrchestrationCommandV0; OrchestrationEventV0.
Estado: aceptada localmente
```

```text
Fecha: 2026-05-09
Decision: Exponer `RegisterPhaseArtifact` y `PhaseArtifactRegistered` en el resource MCP del workflow.
Motivo: El director o una IA externa no debe deducir la nueva transicion leyendo codigo interno; MCP es la frontera de gobierno.
Alternativas: Esperar al cableado de ACK; dejar solo docs locales del core; ampliar el resource sin test de catalogo.
Impacto: `orquesta.core_workflow.contracts.v0` incluye 32 comandos/eventos, la ref `phase_artifacts` y mantiene el payload bajo 10 KB acortando guardrails y referencias canonicas.
Contratos afectados: mcp.resource.orquesta.core_workflow.contracts.v0.puro, RegisterPhaseArtifact, PhaseArtifactRegistered.
Estado: aceptada localmente
```

```text
Fecha: 2026-05-04
Decision: El primer slice MCP para nueva app sera un tool fino llamado orquesta.apps.solicitar_nueva.v0.
Motivo: SolicitarNuevaApp v0 ya esta declarado como contrato compartido para orquesta-mcp y su propietario es orquesta-factory.
Alternativas: Crear un flujo MCP propio; reutilizar CLI; acceder a DB para crear proyecto.
Impacto: MCP queda como adaptador inbound y no como nucleo de negocio.
Contratos afectados: mcp.tool.orquesta.apps.solicitar_nueva.v0; SolicitarNuevaApp v0 como dependencia externa.
Estado: aceptada localmente
```

```text
Fecha: 2026-05-04
Decision: No copiar ni ampliar un cmd/mcp.go gigante para este arranque.
Motivo: AGENTS.md prohibe ampliar un fichero MCP gigante y exige tools/resources pequenos conectados por puertos.
Alternativas: Copiar cmd/mcp.go existente; centralizar todos los tools en un unico entrypoint grande.
Impacto: La implementacion futura debera separar registro MCP, handlers de tools, resources/prompts y conectores hacia casos de uso.
Contratos afectados: Ningun contrato global; afecta estructura interna futura de orquesta-mcp.
Estado: aceptada localmente
```

```text
Fecha: 2026-05-04
Decision: El resource minimo sera contractual, no una proyeccion de estado.
Motivo: Para pedir una app nueva, la IA necesita conocer el contrato SolicitarNuevaApp v0 antes de invocar el tool; no necesita leer DB ni runtime.
Alternativas: Exponer proyectos existentes; exponer tablas; generar documentacion larga.
Impacto: orquesta://contracts/solicitar-nueva-app/v0 sera compacto, estable y sin datos internos.
Contratos afectados: mcp.resource.orquesta.contracts.solicitar_nueva_app.v0
Estado: aceptada localmente
```

```text
Fecha: 2026-05-04
Decision: El prompt minimo guiara aclaraciones antes del tool y no inventara campos obligatorios.
Motivo: El contrato global indica que, si falta informacion no inferible, la respuesta devuelve preguntas abiertas o supuestos, no contratos inventados.
Alternativas: Inferir todo desde texto libre; imponer campos desde MCP.
Impacto: El prompt queda bloqueado para ejemplos completos hasta que factory publique el schema canonico de AppSpecRequestV0.
Contratos afectados: mcp.prompt.orquesta.solicitar_nueva_app.v0
Estado: aceptada localmente
```

```text
Fecha: 2026-05-04
Decision: MCP consumira los schemas canonicos de `AppSpecRequestV0` y `AppSpecV0` desde `orquesta-factory/docs/schemas/`, enlazados por `modulos/CONTRATOS.md`.
Motivo: factory es propietario del contrato `SolicitarNuevaApp v0`; duplicar schemas en MCP crearia divergencia.
Alternativas: copiar schemas dentro de MCP; describir campos a mano en el resource; bloquear MCP hasta implementacion.
Impacto: MCP documenta y valida el envelope del tool, pero referencia los schemas/fixtures canonicos de factory para la carga de datos.
Contratos afectados: mcp.tool.orquesta.apps.solicitar_nueva.v0, mcp.resource.orquesta.contracts.solicitar_nueva_app.v0, SolicitarNuevaApp v0.
Estado: aceptada por el director y registrada
```

```text
Fecha: 2026-05-04
Decision: MCP-005A implementa solo descriptor, DTOs locales y mappers puros del tool orquesta.apps.solicitar_nueva.v0.
Motivo: El corte solicitado excluye servidor MCP real, transporte, DB, CLI, runtime, filesystem y reglas duplicadas de factory.
Alternativas: Crear handler MCP real; llamar directamente al caso de uso de factory; copiar schemas completos en MCP.
Impacto: El modulo ya tiene una superficie Go testeada para integrar el servidor futuro sin fijar transporte ni duplicar negocio.
Contratos afectados: mcp.adapter.orquesta.apps.solicitar_nueva.v0.puro; mcp.tool.orquesta.apps.solicitar_nueva.v0.
Estado: aceptada localmente
```

```text
Fecha: 2026-05-04
Decision: MCP-006 publica un resource puro `orquesta.contracts.shared.v0` y descriptors por contrato compartido v0, sin servidor MCP real.
Motivo: MCP debe permitir que una IA consulte contratos y progreso compacto sin leer contextos globales enormes ni acceder a DB/CLI/runtime/filesystem productivo.
Alternativas: Leer `../../CONTRATOS.md` en runtime; duplicar documentacion extensa por contrato; implementar transporte MCP real en este corte.
Impacto: El modulo expone un catalogo compacto estable de contratos compartidos con message keys de resumen/progreso y referencias canonicas a los modulos propietarios.
Contratos afectados: mcp.resource.orquesta.contracts.shared.v0.puro; SolicitarNuevaApp v0; PersistenceRepository v0; RuntimeLaunchRequest v0; OrquestaEvent v0; GovernanceCatalog v0; OperationalStatusQuery v0; DeploymentPlan v0; GenerarI18nDocsIniciales v0.
Estado: aceptada localmente
```

```text
Fecha: 2026-05-04
Decision: MCP-007 publica un resource puro `orquesta.project.roadmap.v0` para roadmap y decisiones compactas de nucleo.
Motivo: Una IA necesita entender el estado accionable de `orquesta-core` sin cargar docs globales completos ni tocar DB, runtime, servidor MCP real o filesystem productivo.
Alternativas: Leer `../CONTRATOS.md` en runtime; copiar docs completos de core; exponer eventos o progreso desde observability en este corte.
Impacto: El modulo expone una proyeccion estatica versionada de hitos y decisiones de nucleo con refs canonicas, lookup puro y tests de saneamiento.
Contratos afectados: mcp.resource.orquesta.project.roadmap.v0.puro; RegistrarProyectoDesdeAppSpec v0; FunctionContract v0; ProyectoPlanBorradorV0; PersistenceRepository v0; OrquestaEvent v0; RuntimeLaunchRequest v0; CapacityDecision v0; GovernanceCatalog v0; DeploymentPlan v0.
Estado: aceptada localmente
```

```text
Fecha: 2026-05-04
Decision: MCP-008 se limita a un resource puro e independiente para `OperationalStatusQuery v0`, sin tool ni servidor MCP real.
Motivo: El contrato compartido ya existe y `shared_contracts_resource_v0.go` ya da catalogo general, pero una IA necesita una guia compacta y dedicada de invocacion/shape sin cargar docs largas ni abrir otro frente tecnico.
Alternativas: ampliar `shared_contracts_resource_v0.go` con mas detalle; crear un tool MCP read-only; conectar ya un servidor/transporte MCP.
Impacto: `operational_status_resource_v0.go` expone summary, consumers permitidos, endpoint recomendado, scopes, secciones, invariantes y errores publicos como proyeccion estatica testeada.
Contratos afectados: mcp.resource.orquesta.observability.operational_status.v0.puro; OperationalStatusQuery v0; DiagnosticoCompactoV0.
Estado: aceptada localmente
```

```text
Fecha: 2026-05-04
Decision: MCP-005 se ejecuta como adaptador HTTP local fino contra `POST /api/v0/apps/spec` publicado por orquesta-factory.
Motivo: El puerto publicado ya existe y permite cerrar el tool sin meter servidor MCP real, sin duplicar negocio y sin acoplar MCP al nucleo interno de factory.
Alternativas: Llamar directamente a `SolicitarNuevaAppV0` y `GenerarBacklogInicialPropuestoV0`; esperar a un servidor MCP completo; copiar validaciones de factory en MCP.
Impacto: `solicitar_nueva_app_tool_executor_v0.go` queda como conector pequeno de transporte; los 400 publicos se traducen al resultado canonico del tool y los fallos de transporte/configuracion quedan como error Go local.
Contratos afectados: mcp.adapter.orquesta.apps.solicitar_nueva.v0.http_local; SolicitarNuevaApp v0; AppSpecHTTPResponseV0; AppSpecHTTPErrorResponseV0.
Estado: aceptada localmente
```

```text
Fecha: 2026-05-04
Decision: MCP-009 publica un resource puro `orquesta.core_workflow.contracts.v0` para consultar el workflow durable tras NCW-025.
Motivo: Una IA necesita ver fases, comandos, eventos y shape de estado de `orquesta-core-workflow` sin cargar docs extensas ni ejecutar workflow.
Alternativas: Leer docs de workflow en runtime; crear un tool MCP mutador; duplicar contratos completos en MCP.
Impacto: MCP expone un descriptor compacto basado en constantes publicas del modulo propietario, sin servidor MCP real ni efectos externos.
Contratos afectados: mcp.resource.orquesta.core_workflow.contracts.v0.puro; OrchestrationRunV0; OrchestrationCommandV0; OrchestrationEventV0.
Estado: aceptada localmente
```

```text
Fecha: 2026-05-04
Decision: MCP-010 divide `project_roadmap_resource_v0.go` y `shared_contracts_resource_v0.go` en shells publicos, payloads, lookup y descriptors estaticos.
Motivo: AGENTS.md prohibe ampliar ficheros MCP gigantes y estos resources habian superado 400 lineas cada uno.
Alternativas: Mantener ficheros rojos; extraer un paquete nuevo; cambiar contratos para reducir payload.
Impacto: La API publica, URIs, content-types, payloads, errores y textos observados por tests se mantienen; la responsabilidad queda separada en ficheros menores de 300 lineas.
Contratos afectados: mcp.resource.orquesta.project.roadmap.v0.puro; mcp.resource.orquesta.contracts.shared.v0.puro.
Estado: aceptada localmente
```

```text
Fecha: 2026-05-24
Decision: MCP-036 publica freshness, refs de autoridad y refs de backlog T25 en `orquesta.project.roadmap.v0` y `orquesta.contracts.shared.v0`.
Motivo: MCP es superficie preferente para IA; estados `pendiente_*` de una foto anterior podian relanzar trabajo cerrado u ocultar backlog vivo.
Alternativas: Leer docs/backlog en runtime; dejar los descriptores estaticos sin fuente; versionar un resource nuevo.
Impacto: El resource sigue siendo puro y estatico, pero cada hito/contrato declara verificacion y refs vivas. Lo historico se marca como compatibilidad y lo abierto enlaza backlog/doc local.
Contratos afectados: mcp.resource.orquesta.project.roadmap.v0.puro; mcp.resource.orquesta.contracts.shared.v0.puro.
Estado: aceptada localmente
```

```text
Fecha: 2026-05-04
Decision: MCP-011 expone `BootstrapProyectoDesdeAppSpec v0` como resource contractual compacto y tool puro local.
Motivo: La IA necesita coordinar el bootstrap desde AppSpec sin duplicar el flujo ni activar persistencia/runtime; el propietario del caso de uso es `orquesta-director`.
Alternativas: Crear un resource solo documental; implementar servidor MCP real; llamar por separado a factory, core y workflow desde MCP.
Impacto: `ExecuteMCPBootstrapToolV0` recibe `BootstrapProyectoDesdeAppSpecCommandV0`, normaliza solo el envelope, invoca solo `orquesta-director.BootstrapProyectoDesdeAppSpecV0` y devuelve refs/contadores/eventos compactos.
Contratos afectados: mcp.tool.orquesta.director.bootstrap_appspec.v0.puro; mcp.resource.orquesta.director.bootstrap_appspec.v0.puro; BootstrapProyectoDesdeAppSpec v0.
Estado: aceptada localmente
```

```text
Fecha: 2026-05-10
Decision: Los tools MCP de nueva app/director publican que AppSpecRequestV0
puede transportar `request_kind` y `execution_mode`.
Motivo: la IA que gobierne Orquesta debe poder pedir solo documentacion,
analisis, programacion, deploy o app completa sin depender de campos libres de
objetivo ni de una UI concreta.
Alternativas: crear un tool MCP por cada tipo de peticion; dejarlo como
convencion textual; duplicar validacion de factory.
Impacto: MCP conserva adaptadores finos y sigue delegando validacion en
factory, pero el descriptor ya informa del contrato de tipo/modo.
Contratos afectados: orquesta.apps.solicitar_nueva.v0,
orquesta.apps.arrancar_director.v0, AppSpecRequestV0.
Estado: aceptada localmente.
```

```text
Fecha: 2026-06-25
Decision: La observacion de goal-first de nueva app se expone como tool MCP y
bridge REST propio: `orquesta.apps.observe_director_goal.v0` y
`POST /api/v0/apps/director/goal/observe`.
Motivo: con runtimes que soportan goal, el loop automatico vive en el goal; la
superficie publica de Orquesta debe poder refrescar estado, validar cierre y
persistir la consecuencia causal sin relanzar el director legacy.
Impacto: MCP solo traduce DTOs y errores publicos. La validacion de cierre
sigue en `orquesta-app-director-service`; efectos de composicion como sincronizar
cola pertenecen al executor inyectado por el stack.
Contratos afectados: mcp.tool.orquesta.apps.observe_director_goal.v0;
rest.bridge.orquesta.apps.observe_director_goal.v0.
Estado: aceptada localmente.
```

```text
Fecha: 2026-06-08
Decision: `director.stats` y `autoprogramming.status` publican
`ops_snapshot` read-only para el cockpit del Director.
Motivo: `/ops` estaba reconstruyendo la llamada visible del Director con
JavaScript a partir de cola, runs y agentes. El Director/cockpit necesita una
proyeccion contractual compacta y provider-agnostic sin abrir todavia un
daemon nuevo ni una ruta duplicada.
Impacto: `MCPDirectorStatsToolResultV0` y
`MCPAutoprogrammingStatusToolResultV0` incluyen
`DirectorAutonomousOpsSnapshotV0`. El snapshot deriva de stats, decision
context y cola inyectada; no conoce Codex, Gemini, Claude, OPES, DB, HOME,
modelo ni runtime real. La decision del snapshot es diagnostica/read-only y no
filtra, corta ni descarta entregas.
Contratos afectados: orquesta.director.stats.v0,
orquesta.autoprogramming.status.v0, DirectorAutonomousOpsSnapshotV0.
Estado: aceptada localmente.
```

```text
Fecha: 2026-05-13
Decision: El apagado de servidor se expone como tool MCP/REST propio y no como
senal directa desde MCP.
Motivo: cortar el proceso servidor sin drenar runs vivos repite el fallo de
orquestacion manual. El director y la web necesitan una orden unica observable
que solicite stop por RunControl, ejecute supervisor/drain y lea stats antes de
permitir cerrar el proceso.
Alternativas: reutilizar `orquesta.runs.control.v0` para cada run desde la UI;
hacer que CLI envie SIGINT directamente; meter logica de shutdown dentro del
gateway.
Impacto: `orquesta.server.shutdown.v0` delega en
`orquesta-server-shutdown`, queda opt-in por executor inyectado y tambien se
publica como `POST /api/v0/server/shutdown`. El proceso servidor solo debe
recibir senal final cuando `shutdown_ready=true`.
Contratos afectados: mcp.tool.orquesta.server.shutdown.v0,
orquesta://contracts/server-shutdown/v0, /api/v0/server/shutdown.
Estado: aceptada localmente.
```

```text
Fecha: 2026-05-13
Decision: Las stats del director recuperan la run canonica por
`external_job_ref` si el cliente envia un `run_ref` obsoleto.
Motivo: los trabajos externos pueden normalizar `run_ref` al aceptar la orden.
La web, OPES o un smoke antiguo pueden conservar el ref preliminar; fallar sin
intentar resolver por job externo rompe observabilidad.
Impacto: `orquesta.director.stats.v0` mantiene `run_ref` como preferente, pero
si no carga y existe `external_job_ref`, consulta el puerto externo sin filtrar
por el ref obsoleto y continua con la run canonica.
Contratos afectados: mcp.tool.orquesta.director.stats.v0;
rest.bridge.orquesta.director.stats.v0.
Estado: aceptada localmente.
```

```text
Fecha: 2026-05-22
Decision: `orquesta.autoprogramming.prepare_run.v0` queda como tool MCP/REST
opt-in separado de `external-work.run`.
Motivo: autoprogramacion necesita preparar una run continuable y devolver refs
causales antes de supervisar; mezclarlo con trabajos externos neutrales
acoplaria el contrato generico a una composicion concreta.
Impacto: MCP solo define DTO, descriptor, HTTP handler y puerto inyectado. La
composicion Codex implementa el executor real y la supervision posterior debe
usar `run_ref` explicito en la rama legacy. Actualizacion 2026-06-25: la rama
Goal-first puede devolver `goal_specs[]` sin `run_ref` legacy ni `continue`.
Contratos afectados: mcp.tool.orquesta.autoprogramming.prepare_run.v0;
rest.bridge.orquesta.autoprogramming.prepare_run.v0.
Estado: aceptada localmente.
```

```text
Fecha: 2026-06-25
Decision: `orquesta.autoprogramming.prepare_run.v0` acepta salida Goal-first sin
`run_ref` legacy.
Motivo: cuando la composicion clasifica el trabajo como `goal_ready`,
materializar una `run` y encolarla duplica el loop que ya debe llevar Codex
Goal. `prepare-run` debe entregar `goal_specs[]` y dejar que la composicion Goal
lance/observe, sin activar `runs/supervise`.
Impacto: `run_ref`, `workflow_task_refs`, `wait_agent_refs`, `phase_id` y
`continue` son campos de la rama legacy. La rama Goal-first devuelve
`goal_specs[]` validos y no arranca runtime por si misma.
Contratos afectados: mcp.tool.orquesta.autoprogramming.prepare_run.v0;
rest.bridge.orquesta.autoprogramming.prepare_run.v0.
Estado: aceptada localmente.
```
