# Contratos locales: orquesta-mcp

Registra puertos, DTOs y eventos que `orquesta-mcp` expone o consume.

## Plantilla

```text
Nombre:
Tipo: puerto_entrada | puerto_salida | dto | evento | error
Version:
Propietario:
Consumidores:
Campos:
Invariantes:
Errores:
Pruebas de contrato:
```

```text
Nombre: mcp.tool.orquesta.domain_work.v0
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-mcp
Consumidores: servidor MCP futuro, cliente IA director y bridge HTTP local
Campos:
  descriptor:
    name: orquesta.domain_work.v0
    version: v0
    resource_uri: orquesta://contracts/domain-work/v0
  input:
    request_id, correlation_id: refs externas opcionales
    action: create_job | submit_artifact
    job_request: DomainWorkJobRequestV0 cuando action=create_job
    artifact_submission: DomainWorkArtifactSubmissionV0 cuando
      action=submit_artifact
  output_ok:
    estado: ok
    action
    job: DomainWorkJobV0 para create_job
    receipt: DomainWorkArtifactReceiptV0 para submit_artifact
  output_error:
    estado: error
    errores_publicos
Invariantes:
  - Adaptador inbound fino.
  - `create_job` delega solo en `DomainWorkJobCreatorPortV0` inyectado.
  - `submit_artifact` delega solo en `DomainWorkArtifactSubmitterPortV0`
    inyectado.
  - No importa OPES, conector REST, DB, runtime, filesystem ni proveedor.
  - El registro en transporte central es opt-in: si no se inyecta executor, el
    tool devuelve `mcp_transport_tool_unbound`.
Pruebas de contrato:
  - Descriptor compacto del tool.
  - Executor delega `create_job` al creator.
  - Executor delega `submit_artifact` al submitter.
  - Transporte queda opt-in/unbound si falta puerto.
  - `RegisterMCPTransportV0` registra el tool en el catalogo global.
  - HTTP bridge `POST /api/v0/domain-work` delega y propaga correlacion.
  - Test de arquitectura impide importar OPES o conector REST.
```

```text
Nombre: mcp.tool.orquesta.director.stats.v0
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-mcp
Consumidores: web, REST local, director IA y apps externas como OPES
Campos:
  input:
    run_ref: ref de run opcional si se consulta por `external_job_ref`
    app_ref: app externa opcional
    external_job_ref: job externo de dominio, por ejemplo OPES
    include_process_refs, include_agent_progress, include_agent_usage
  output_ok:
    stats: DirectorRunStatsV0 completo
    external_job: proyeccion compacta opcional con job_ref, work_kind,
      change_ref, run_ref, task_ref, agent_ref, status y delivery_refs
    decision_context
Invariantes:
  - Si llega `external_job_ref`, MCP delega la resolucion en un puerto
    inyectado; no conoce OPES, app-change internals, filesystem ni runtime.
  - Si falta `run_ref`, el puerto puede resolverlo desde el job externo.
  - La salida conserva `DirectorRunStatsV0`; `external_job` es una proyeccion
    aditiva para que OPES no tenga que interpretar toda la run.
Pruebas de contrato:
  - Executor resuelve `run_ref` por job externo mediante puerto fake.
  - HTTP bridge conserva el mismo envelope REST.
```

```text
Nombre: rest.bridge.orquesta.domain_work.v0
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-mcp
Consumidores: adaptadores HTTP locales
Campos:
  path: POST /api/v0/domain-work
  input: mismo envelope que `orquesta.domain_work.v0`
  output: mismo resultado compacto del tool
Invariantes:
  - Bridge REST fino; no abre servidor ni crea puertos productivos.
  - Usa executor inyectado de `orquesta.domain_work.v0`.
  - No conoce OPES, conector REST, DB, runtime, proveedor, HOME ni OAuth.
Pruebas de contrato:
  - Handler HTTP invoca executor fake.
  - Propaga `X-Correlation-ID` desde resultado/input/header.
```

```text
Nombre: mcp.tool.orquesta.apps.arrancar_director.v0
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-mcp
Consumidores: web wizard, servidor MCP futuro y cliente IA
Campos:
  descriptor:
    name: orquesta.apps.arrancar_director.v0
    version: v0
    resource_uri: orquesta://contracts/arrancar-director-app/v0
  input:
    request_id: ref externa opcional; el adaptador la traduce a ref interna neutra
    correlation_id: ref externa opcional; el adaptador la traduce a ref interna neutra
    app_spec_request: AppSpecRequestV0 validado por factory
    max_bursts, max_steps_per_burst, max_dispatches_per_wait: limites operativos
  output_ok:
    estado: ok
    app_spec: resumen compacto
    run_ref: ref interna neutra
	    phase_id: fase inicial
	    director_task: task_ref, brainstorm_ref, agent_request_id y capacidad
	    director_tasks: lista compacta de tareas/directores arrancables cuando
	      la solicitud requiere equipo director
	    loop_status: estado compacto del loop
	    started_agents: agentes arrancados por Orquesta
  output_error:
    estado: error
    errores_publicos: issues de factory o servicio de aplicacion
Invariantes:
  - Delegacion en `orquesta-app-director-service`.
  - No elige proveedor, modelo, HOME, OAuth, DB ni runtime.
  - No filtra nombres de adaptador como mcp/web/api a refs internas del core.
  - Permite que el formulario web invoque Orquesta sin conocer factory, workflow ni launcher.
	Pruebas de contrato:
	  - Ejecucion con puertos fake arranca director y devuelve `started_agents`.
	  - Autonomia alta devuelve `director_tasks` y arranca equipo por batch.
	  - Request/correlation externos con `mcp` se traducen a refs internas neutras.
	  - Request invalida devuelve errores publicos sin crear run.
```

```text
Nombre: rest.bridge.orquesta.apps.arrancar_director.v0
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-mcp
Consumidores: orquesta-web
Campos:
  path: POST /api/v0/apps/director
  input: mismo envelope que `orquesta.apps.arrancar_director.v0`
  output: mismo resultado compacto del tool
Invariantes:
  - Bridge REST fino; no abre servidor por si mismo ni crea puertos productivos.
  - Usa executor inyectado de `orquesta.apps.arrancar_director.v0`.
  - No conoce DB, runtime, proveedor, HOME, OAuth ni modelos.
Pruebas de contrato:
  - Handler HTTP invoca executor y conserva correlation id.
  - Web puede usarlo con `RESTArrancarDirectorAppClientV0`.
```

```text
Nombre: mcp.tool.orquesta.director.stats.v0
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-mcp
Consumidores: director, orquesta-web y adaptadores HTTP locales
Campos:
  descriptor:
    name: orquesta.director.stats.v0
    version: v0
    resource_uri: orquesta://contracts/director-stats/v0
  input:
    request_id, correlation_id: refs externas opcionales
    run_ref: ref interna neutra obligatoria
    occurred_at: instante operacional opcional
    include_process_refs: activa enriquecimiento por registro de procesos
    include_agent_progress: activa observaciones de progreso por puerto
  output_ok:
    estado: ok
    run_ref
    stats: DirectorRunStatsV0 completo serializado como contrato publico
    decision_context: DirectorDecisionContextV0 compacto para decision del
      director/API/MCP/web
  output_error:
    estado: error
    errores_publicos
Cobertura de stats publicas:
  - tareas: counts.tasks_total, tasks_closed, tasks_open y refs open/closed;
  - agentes: counts agents_* y lista agents con estado/control/progreso opcional;
  - rework: counts.rework_requests y refs.rework_requests;
  - replan: counts.replan_decisions y refs.replan_decisions;
  - cierre: closure.status, blocked, ready, closed, blocked_by y blocker_refs;
  - progreso: progress.percent_complete, tasks, agents observados y issues.
Cobertura de decision_context:
  - progreso por fase/tarea/agente;
  - sesiones/procesos por refs opacas cuando `include_process_refs=true`;
  - actividad reciente compacta por claves i18n;
  - agentes vivos/parados/fallidos y control registrado/ausente;
  - bloqueos y causas de cierre bloqueado;
  - rework/replan con refs opacas;
  - duraciones de fase cuando hay timestamps y quietud por falta de senal/ticks.
Invariantes:
  - Delegacion en `RunStorePortV0`, `AgentProcessRegistryPortV0` opcional y
    `AgentProgressObservationProviderPortV0` opcional.
  - No crea stores, no lee DB, no consulta runtime real, no usa cmd y no expone
    rutas locales, credenciales, proveedor, modelo ni HOME.
  - No duplica reglas de cierre ni progreso; serializa lo disponible en
    `DirectorRunStatsV0`.
Pruebas de contrato:
  - `TestMCPDirectorStatsToolExecutorV0DevuelveStatsDeRunStore`.
  - `TestMCPDirectorStatsToolExecutorV0DecisionContextCompletoParaDirector`.
  - `TestMCPTransportV0DirectorStatsIncluyeProgressObservadoEnJSON`.
  - `TestMCPDirectorStatsHTTPHandlerV0OKConExecutorRealInMemory`.
```

```text
Nombre: rest.bridge.orquesta.director.stats.v0
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-mcp
Consumidores: orquesta-web y adaptadores HTTP locales
Campos:
  path: POST /api/v0/director/stats
  input: mismo envelope que `orquesta.director.stats.v0`
  output: mismo resultado compacto del tool, con `stats` como
    `DirectorRunStatsV0` completo para tareas, agentes, rework, replan,
    progreso y bloqueo de cierre; tambien incluye `decision_context`
    `DirectorDecisionContextV0`
Invariantes:
  - Bridge REST fino; no abre servidor por si mismo ni crea puertos productivos.
  - Usa executor inyectado de `orquesta.director.stats.v0`.
  - No conoce DB, runtime, cmd, procesos, proveedor, HOME, OAuth ni modelos.
  - Propaga `X-Correlation-ID` desde el header, input o resultado.
Errores:
  - 404 JSON publico si la ruta no coincide.
  - 405 JSON publico con `Allow: POST` si el metodo no coincide.
  - 503 JSON publico si falta executor.
  - 400 JSON publico si el body es invalido o el resultado del executor tiene `estado:error`.
Pruebas de contrato:
  - Handler HTTP invoca executor fake y conserva correlation id del resultado.
  - Handler HTTP invoca executor real in-memory con RunStore inyectado.
  - Metodo incorrecto, executor nil, body invalido, path incorrecto y validacion de `run_ref`.
```

```text
Nombre: mcp.tool.orquesta.director_agent.apply_decision.v0
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-mcp
Consumidores: servidor MCP futuro y cliente IA director
Campos:
  descriptor:
    name: orquesta.director_agent.apply_decision.v0
    version: v0
    resource_uri: orquesta://contracts/director-agent-decision/v0
  input:
    request_id, correlation_id: refs externas opcionales
    occurred_at: instante operacional
    decision: DirectorAgentDecisionV0 compacto
  output_ok:
    estado: ok
    run_ref, current_phase
    command: command_id, command_type, run_id
    events_count
  output_error:
    estado: error
    errores_publicos
Invariantes:
	  - Delegacion en `orquesta-director-agent-workflow`.
	  - No crea store ni sink; usa puertos inyectados.
	  - Para `create_microtask`, el transporte debe inyectar tambien `TaskStore`.
	  - No elige proveedor, modelo, HOME, OAuth, DB ni runtime.
	  - No expone payloads completos del workflow.
	Pruebas de contrato:
	  - Executor con store/sink fake aplica `request_brainstorm`.
	  - El puente de workflow cubre `publish_function_contract` y `create_microtask`.
	  - Sin puerto queda opt-in/unbound en el registro de transporte.
```

## Contratos propuestos

```text
Nombre: mcp.tool.orquesta.apps.solicitar_nueva.v0
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-mcp
Consumidores: clientes MCP con IA que quieran solicitar una app nueva sin usar CLI ni DB
Campos:
  tool_name: orquesta.apps.solicitar_nueva.v0
  input:
    app_spec_request: objeto compatible con AppSpecRequestV0 de SolicitarNuevaApp v0
    respuesta: compacta | completa; opcional; por defecto compacta
  output_ok:
    app_spec: AppSpecV0 devuelto por orquesta-factory
    backlog_inicial_propuesto: BacklogInicialPropuestoV0 devuelto por orquesta-factory, si aplica
    preguntas_abiertas: lista compacta de informacion no inferible, si el puerto la devuelve
    supuestos: lista compacta de supuestos aplicados, si el puerto los devuelve
  output_error:
    code: error publico de SolicitarNuevaApp v0
    message: texto corto orientado a IA
    field: ruta logica del campo cuando exista
Invariantes:
  - El tool es un adaptador inbound fino.
  - Llama al puerto SolicitarNuevaApp v0 del modulo propietario orquesta-factory.
  - No valida reglas de negocio fuera del contrato publicado por orquesta-factory.
  - No persiste estado, no crea tareas en DB, no arranca runtime y no asigna agentes.
  - No importa internal/ de otros modulos.
  - No expone dumps de DB ni estructuras privadas.
  - Si falta informacion no inferible, devuelve preguntas abiertas o supuestos del puerto, no inventa contrato.
Errores:
  - app_spec_invalida
  - opcion_incompatible
  - target_no_soportado
  - idioma_invalido
  - conector_requerido_no_disponible
Pruebas de contrato:
  - Validar que el schema MCP acepta solo el envelope local y delega app_spec_request al schema canonico de AppSpecRequestV0 cuando este publicado.
  - Fixture ok: tool devuelve AppSpecV0 y, si aplica, BacklogInicialPropuestoV0 sin campos internos.
  - Fixture error: cada error publico se serializa como output_error estable para IA.
```

```text
Nombre: mcp.resource.orquesta.contracts.solicitar_nueva_app.v0
Tipo: puerto_salida
Version: v0
Propietario: orquesta-mcp
Consumidores: clientes MCP con IA
Campos:
  uri: orquesta://contracts/solicitar-nueva-app/v0
  contenido:
    descripcion: resumen compacto del contrato SolicitarNuevaApp v0
    propietario: orquesta-factory
    input: AppSpecRequestV0
    output_ok: AppSpecV0 y BacklogInicialPropuestoV0
    errores_publicos: lista de errores publicos
    invariantes: lista corta de invariantes globales
Invariantes:
  - Resource compacto y estable, pensado para orientar a una IA antes de invocar el tool.
  - No duplica el detalle extenso del propietario; enlaza o referencia el contrato canonico cuando exista.
  - No incluye datos de proyectos, DB, runtime ni proveedor.
Errores:
  - contrato_no_publicado, si el schema canonico de AppSpecRequestV0 aun no esta disponible para renderizar detalle.
Pruebas de contrato:
  - Snapshot del resource sin campos internos ni referencias a CLI/DB.
```

```text
Nombre: mcp.prompt.orquesta.solicitar_nueva_app.v0
Tipo: puerto_salida
Version: v0
Propietario: orquesta-mcp
Consumidores: clientes MCP con IA
Campos:
  prompt_name: orquesta.solicitar_nueva_app.v0
  argumentos:
    idioma: codigo de idioma solicitado por el usuario
    contexto_usuario: texto libre con necesidad de negocio o producto
  resultado:
    instrucciones: guia breve para convertir contexto_usuario en AppSpecRequestV0
    preguntas_minimas: preguntas a hacer antes de invocar el tool si faltan datos obligatorios
    politica_supuestos: usar supuestos solo si el contrato los permite y devolverlos visibles
Invariantes:
  - Prompt orienta a la IA, no ejecuta negocio.
  - Prompt no sugiere tocar CLI, DB, filesystem ni runtime.
  - Prompt respeta i18n por defecto y pide aclaraciones cuando el contrato no permite inferir.
Errores:
  - idioma_invalido, si el idioma no es aceptado por SolicitarNuevaApp v0.
Pruebas de contrato:
  - Snapshot del prompt en modo minimo.
  - Caso con informacion insuficiente: el prompt pide preguntas antes del tool.
```

## Contratos implementados

```text
Nombre: mcp.transport.registry.v0.puro
Tipo: puerto_salida
Version: v0
Propietario: orquesta-mcp
Consumidores: adaptador externo de servidor/transporte MCP opt-in
Campos:
  TransportPortV0:
    RegisterResourceV0(MCPTransportResourceEnvelopeV0) error
    RegisterToolV0(MCPTransportToolEnvelopeV0) error
  resource_envelope:
    name, version, uri, content_type, summary_key, shape, mode=adapter_opt_in
    handler: funcion local excluida de JSON que devuelve payload compacto
  tool_envelope:
    name, version, resource_uri, input_shape, output_shape, mode=adapter_opt_in
    handler: funcion local excluida de JSON que recibe/entrega JSON compacto
  bindings:
    nueva_app executor opcional
    puertos OperatorMCP* opcionales e inyectados
Invariantes:
  - Registra resources/tools existentes; no abre sockets ni crea servidor real.
  - El servidor MCP real queda fuera como adaptador externo opt-in.
  - No lee DB, outbox, runtime, event-store, filesystem productivo, HOME, OAuth, proveedor ni modelo.
  - No conoce internals de otros modulos; usa DTOs publicos, funciones puras o puertos inyectados.
  - Si un tool requiere adaptador externo no inyectado, devuelve `mcp_transport_tool_unbound`.
Errores:
  - mcp_transport_port_unavailable
  - mcp_transport_tool_unbound
Pruebas de contrato:
  - Transporte fake en memoria registra resources/tools existentes.
  - El fake sirve un resource y ejecuta `orquesta.operator.status.query.v0` solo por puerto fake.
  - Envelopes y payloads servidos son compactos y no contienen HOME, OAuth, proveedor, modelo, secretos, DSN, SQL, event-store ni internal/.
```

```text
Nombre: mcp.resource.orquesta.operator.operations.v0.puro
Tipo: dto
Version: v0
Propietario: orquesta-mcp
Consumidores: futura capa de servidor MCP local y clientes MCP con IA
Campos:
  descriptor:
    name: orquesta.operator.operations.v0
    version: v0
    uri: orquesta://operator/operations/v0
    content_type: application/vnd.orquesta.operator.operations.v0+json
  resource:
    capabilities: OperatorMCPCapabilitiesV0
    tools: status, supervised_burst, outbox.pending, directed_query
    errores_publicos: validacion y puerto no disponible
Invariantes:
  - Adaptador puro sin servidor MCP real ni transporte.
  - Importa `orquesta-operator-mcp` solo para DTOs y puertos publicos.
  - No lee DB, event-store, outbox, runtime, filesystem, HOME ni credenciales.
  - Si falta puerto inyectado, devuelve `operator_mcp_port_unavailable`.
  - Nunca ejecuta scheduler, supervisor ni runtime por conocimiento interno.
Errores:
  - operator_mcp_required_field
  - operator_mcp_opaque_ref_invalid
  - operator_mcp_budget_invalid
  - operator_mcp_limit_invalid
  - operator_mcp_question_invalid
  - operator_mcp_section_invalid
  - operator_mcp_port_unavailable
  - operator_mcp_port_error
Pruebas de contrato:
  - Resource lista capabilities y tools.
  - Tools validan envelopes antes de llamar al puerto.
  - Tools delegan solo con puerto fake inyectado.
  - Payloads compactos sin internals ni terminos prohibidos.
```

```text
Nombre: mcp.adapter.orquesta.apps.solicitar_nueva.v0.puro
Tipo: dto
Version: v0
Propietario: orquesta-mcp
Consumidores: futura capa de servidor MCP local
Campos:
  descriptor:
    name: orquesta.apps.solicitar_nueva.v0
    version: v0
    input_schema: envelope compacto que referencia AppSpecRequestV0
    resource_uri: orquesta://contracts/solicitar-nueva-app/v0
    prompt_name: orquesta.solicitar_nueva_app.v0
  input:
    request_id: opcional en envelope
    correlation_id: opcional en envelope
    respuesta: opcional; por defecto compacta
    app_spec_request: orquestafactory.AppSpecRequestV0
  output_ok:
    estado: ok
    request_id: heredado de AppSpecV0
    correlation_id: heredado del envelope o request_id
    app_spec: proyeccion compacta de AppSpecV0
    backlog_inicial_propuesto: proyeccion compacta de BacklogInicialPropuestoV0
    preguntas_abiertas: lista compacta desde spec/backlog
    supuestos: lista compacta desde spec
  output_error:
    estado: error
    request_id: normalizado
    correlation_id: normalizado
    errores_publicos: lista compacta de ValidationIssue publico
Invariantes:
  - Implementacion pura sin servidor MCP real.
  - No llama DB, CLI, runtime ni filesystem.
  - No ejecuta reglas de negocio ni validacion detallada; solo normaliza envelope, source, request_id/correlation_id y errores publicos.
  - Importa `orquesta/modulos/orquesta-factory` solo para DTOs publicos.
  - No inventa spec ni backlog cuando hay errores.
Errores:
  - app_spec_invalida
  - opcion_incompatible
  - target_no_soportado
  - idioma_invalido
  - conector_requerido_no_disponible
Pruebas de contrato:
  - Test de descriptor compacto sin dumps.
  - Test mapper input -> AppSpecRequestV0.
  - Test result ok con spec/backlog compactos.
  - Test result error sin backlog inventado.
```

```text
Nombre: mcp.adapter.orquesta.apps.solicitar_nueva.v0.http_local
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-mcp
Consumidores: futura capa de servidor MCP local
Campos:
  config:
    server_url: base http(s) del puerto publicado por orquesta-factory
    endpoint: por defecto /api/v0/apps/spec
    timeout: duracion opcional
  input:
    envelope: MCPNuevaAppToolInputV0
  output:
    result: MCPNuevaAppToolResultV0
Invariantes:
  - Adaptador fino local contra el puerto HTTP publicado por orquesta-factory.
  - Reutiliza AppSpecRequestV0, AppSpecHTTPResponseV0 y AppSpecHTTPErrorResponseV0 del modulo propietario.
  - No duplica reglas de negocio ni valida campos fuera de factory.
  - Traduce 400 publicos de factory a `MCPNuevaAppToolResultV0` con `estado=error`.
  - Fallos de transporte, configuracion o respuesta no canonica se elevan como error Go local y no como contrato de negocio nuevo.
  - No implementa servidor MCP real, DB, runtime ni filesystem productivo.
Errores:
  - errores publicos de factory cuando el puerto responde 400
  - error Go local para transporte, timeout, server_url invalida o respuesta HTTP no canonica
Pruebas de contrato:
  - Test contra `httptest` con handler publico de factory para caso ok.
  - Test contra `httptest` con 400 publico de factory para error canonico.
  - Test de rechazo de `server_url` con credenciales.
  - Test de 500 como fallo de transporte local, sin inventar contrato nuevo.
```

```text
Nombre: mcp.resource.orquesta.contracts.shared.v0.puro
Tipo: dto
Version: v0
Propietario: orquesta-mcp
Consumidores: futura capa de servidor MCP local y clientes MCP con IA
Campos:
  descriptor:
    name: orquesta.contracts.shared.v0
    version: v0
    uri: orquesta://contracts/shared/v0
    content_type: application/vnd.orquesta.contracts.shared.v0+json
    summary_key: mcp.resources.contracts.shared.summary.v0
  resource:
    uri: orquesta://contracts/shared/v0
    version: v0
    canonical_source: ../CONTRATOS.md
    contracts: lista compacta de contratos compartidos v0
  contract:
    contract: nombre publico con version
    resource_uri: orquesta://contracts/<slug>/v0
    owner: modulo propietario
    mcp_role: rol compacto de MCP respecto al contrato
    summary_key: message key del resumen
    canonical_refs: docs/schemas canonicos, sin fixtures extensas
    input: entrada publica resumida
    output: salida publica resumida
    errores_publicos: codigos publicos del contrato
    guardrails: claves compactas de invariantes
    progress_key: message key de estado/siguiente paso
Invariantes:
  - Implementacion pura sin servidor MCP real, transporte, DB, CLI, runtime ni filesystem productivo.
  - El detalle extenso queda en el modulo propietario y en `modulos/CONTRATOS.md`.
  - No incluye secretos, transcripts, dumps de fixtures, endpoints REST ni detalles de sink/tablas/driver.
  - Usa message keys para resumen y progreso en vez de texto largo localizado.
  - Cubre `SolicitarNuevaApp`, `PersistenceRepository`, `RuntimeLaunchRequest`, `OrquestaEvent`, `GovernanceCatalog`, `DeploymentPlan` y `GenerarI18nDocsIniciales` v0.
Errores:
  - contrato_no_encontrado, representado por lookup puro sin resultado.
Pruebas de contrato:
  - Test de descriptor catalogo + 7 resource descriptors.
  - Test de resource compacto sin dumps ni referencias productivas.
  - Test de lookup normalizado por nombre, slug y URI.
  - Test de mapper puro con trim/dedupe de listas.
```

```text
Nombre: mcp.resource.orquesta.project.roadmap.v0.puro
Tipo: dto
Version: v0
Propietario: orquesta-mcp
Consumidores: futura capa de servidor MCP local y clientes MCP con IA que necesiten contexto de nucleo
Campos:
  descriptor:
    name: orquesta.project.roadmap.v0
    version: v0
    uri: orquesta://project/roadmap/v0
    content_type: application/vnd.orquesta.project.roadmap.v0+json
    summary_key: mcp.project.roadmap.summary.v0
  resource:
    uri: orquesta://project/roadmap/v0
    version: v0
    scope: orquesta-core
    canonical_refs: referencias compactas a `../CONTRATOS.md` y `orquesta-core/docs/contratos.md`
    roadmap: lista compacta de hitos de nucleo
    decisiones: lista compacta de decisiones de nucleo
    guardrails: claves compactas de restricciones de adaptador
  roadmap_item:
    id: identificador estable local
    area: area de nucleo
    estado: estado compacto del hito
    owner: modulo propietario
    focus: resumen corto accionable para IA
    summary_key: clave i18n del resumen
    progress_key: clave i18n de progreso/siguiente paso
    contracts: contratos relacionados
    dependencies: dependencias contractuales, si aplica
    guardrails: restricciones compactas
    canonical_refs: refs documentales compactas, sin dumps
  decision:
    id: identificador estable local
    estado: estado compacto
    decision: frase corta de decision de nucleo
    decision_key: clave i18n de decision
    motivo: frase corta de motivo
    motivo_key: clave i18n de motivo
    applies_to: contratos o resources afectados
    guardrails: restricciones compactas
    canonical_refs: refs documentales compactas, sin dumps
Invariantes:
  - Implementacion pura sin servidor MCP real, transporte, DB, CLI, runtime ni filesystem productivo.
  - No lee `../CONTRATOS.md` ni docs de core en runtime; el resource es una proyeccion estatica versionada.
  - Permite a una IA leer roadmap/decisiones de nucleo sin cargar docs globales completos.
  - El detalle canonico permanece en `orquesta-core/docs/contratos.md` y `../CONTRATOS.md`.
  - No incluye secretos, transcripts, dumps de fixtures, endpoints REST, sinks operativos, DSN ni tablas.
  - Cubre registro desde AppSpec, FunctionContract, ProyectoPlanBorrador, persistence/events, runtime/capacity, governance/deploy.
Errores:
  - roadmap_item_no_encontrado, representado por lookup puro sin resultado.
  - decision_no_encontrada, representado por lookup puro sin resultado.
Pruebas de contrato:
  - Test de descriptor compacto.
  - Test de resource compacto y util para nucleo, sin dumps ni referencias productivas.
  - Test de lookup normalizado por ID, area, contrato y key.
  - Test de mappers puros con trim/dedupe de listas.
```

```text
Nombre: mcp.resource.orquesta.observability.operational_status.v0.puro
Tipo: dto
Version: v0
Propietario: orquesta-mcp
Consumidores: futura capa de servidor MCP local y clientes MCP con IA que necesiten entender `OperationalStatusQuery v0` sin leer docs globales completas
Campos:
  descriptor:
    name: orquesta.observability.operational_status.v0
    version: v0
    uri: orquesta://observability/operational-status/v0
    content_type: application/vnd.orquesta.operational-status.v0+json
    summary_key: mcp.resources.operational_status.summary.v0
  resource:
    uri: orquesta://observability/operational-status/v0
    version: v0
    contract_resource_uri: orquesta://contracts/operational-status-query/v0
    recommended_endpoint: /api/v0/operational-status/query
    canonical_refs: referencias compactas a `../CONTRATOS.md`, `orquesta-observability/docs/contratos.md` y al cliente fino ya existente en CLI
    allowed_consumers: pares modulo/canal autorizados por `OperationalStatusQuery v0`
    allowed_scopes: lista compacta de scopes validos
    allowed_sections: lista compacta de secciones validas
    request:
      schema_version: operational_status_query.v0
      input_shape: resumen compacto del envelope de consulta
      opaque_refs: request_id, correlation_id, subject_ref, trace_ref y watermark_ref
      defaults: locale por defecto y limites compactos
    response:
      schema_version: diagnostico_compacto.v0
      estados: ok, degraded, blocked, failed y unknown
      sections: progreso, salud, bloqueos, actividad_reciente, contadores, referencias y warnings
      privacy_flags: banderas que siempre deben ser false
    errores_publicos: lista compacta de errores publicos del contrato
    guardrails: restricciones compactas del adaptador
Invariantes:
  - Implementacion pura sin tool, sin servidor MCP real, sin transporte, sin DB, sin runtime y sin filesystem productivo.
  - El resource no ejecuta diagnosticos ni consulta proyecciones reales; solo describe el contrato compartido y la forma recomendada de invocacion.
  - Las referencias del request y del diagnostico son opacas.
  - El detalle canonico sigue en `orquesta-observability` y `../CONTRATOS.md`.
  - No incluye secretos, transcripts, prompts completos, completions, SQL, DSN, HOME ni detalles de conexion.
Pruebas de contrato:
  - Test de descriptor compacto.
  - Test de resource read-only con endpoint recomendado, consumers/scopes/secciones y errores publicos.
  - Test de lookup normalizado de consumers por modulo/canal.
```

```text
Nombre: mcp.resource.orquesta.core_workflow.contracts.v0.puro
Tipo: dto
Version: v0
Propietario: orquesta-mcp
Consumidores: futura capa de servidor MCP local y clientes MCP con IA que necesiten consultar el workflow durable actual
Campos:
  descriptor:
    name: orquesta.core_workflow.contracts.v0
    version: v0
    uri: orquesta://core-workflow/contracts/v0
    content_type: application/vnd.orquesta.core-workflow.contracts.v0+json
    summary_key: mcp.resources.core_workflow.contracts.summary.v0
  resource:
    owner: orquesta-core-workflow
    canonical_refs: refs compactas a contratos locales de workflow
    state_shape: schema, estados, refs y contadores de OrchestrationRunV0
    phases: catalogo de 10 fases v0
    commands: catalogo compacto de 32 comandos soportados por `SupportedOrchestrationCommandTypesV0`, incluidos `RecordQualityGate` y `RegisterPhaseArtifact`
    events: catalogo compacto de 32 eventos soportados por `SupportedOrchestrationEventTypesV0`, incluidos `QualityGateRecorded` y `PhaseArtifactRegistered`
    errores_publicos: errores publicos de comandos, eventos y estado
    guardrails: restricciones compactas del adaptador
Invariantes:
  - Implementacion pura sin servidor MCP real, transporte, DB, runtime ni filesystem productivo.
  - No ejecuta handler, reducer, replay ni outbox; solo expone una proyeccion compacta para IA.
  - Usa el catalogo publico de `orquesta-core-workflow` y mantiene el detalle canonico en ese modulo.
  - Omite `summary_key` y guardrails repetidos por transicion en el payload serializado para mantener el resource por debajo de 10 KB; el lookup deriva la key cuando se consulta una transicion concreta.
  - No incluye secretos, SQL, DSN, HOME, OAuth, transcripts ni detalles de conectores.
Pruebas de contrato:
  - Test de descriptor compacto.
  - Test de resource con 10 fases, 32 comandos, 32 eventos, CloseRun/RunClosed, RecordQualityGate/QualityGateRecorded y RegisterPhaseArtifact/PhaseArtifactRegistered.
  - Test de sincronizacion contra los catalogos publicos del core.
  - Test de lookup normalizado por comando/evento.
```

```text
Nombre: mcp.tool.orquesta.core_workflow.handle_command.v0.puro
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-mcp
Consumidores: futura capa de servidor MCP local y clientes MCP con IA que gobiernen el workflow durable
Campos:
  descriptor:
    name: orquesta.core_workflow.handle_command.v0
    version: v0
    input_schema: envelope compacto con `current:OrchestrationRunV0`, `command:OrchestrationCommandV0` e `include_state?`
    resource_uri: orquesta://core-workflow/contracts/v0
  input:
    request_id: opcional
    correlation_id: opcional
    include_state: opcional; por defecto false
    current: estado publico del workflow
    command: comando publico del workflow
  output_ok:
    estado: ok
    command: command_id, command_type y run_id
    events: lista de event_type emitidos
    outbox: lista compacta de message_id, message_type y target_port
    state_after: run_id, status, current_phase, last_sequence y contadores compactos, incluido quality_gates
    state_public: OrchestrationRunV0 publico solo si `include_state=true`
  output_error:
    estado: error
    errores_publicos: code, field y correlation_id
Invariantes:
  - Implementacion pura sin servidor MCP real, transporte, DB, persistence, runtime ni filesystem productivo.
  - Invoca solo `orquesta-core-workflow.HandleCommandV0` y `ApplyEventV0`.
  - No persiste eventos, no ejecuta outbox, no arranca agentes y no selecciona proveedor/modelo/HOME/OAuth.
  - No expone payloads de command/event/outbox; devuelve tipos y contadores compactos para IA.
  - `state_public` es opt-in para harness o servidor MCP sin persistence productiva; no sustituye repositorio durable.
Pruebas de contrato:
  - Descriptor compacto.
  - StartRun produce RunStarted y state_after activa sin filtrar project_ref/app_spec_ref.
  - AskDirector bloqueante produce DirectorQuestionRaised, RunBlocked y outbox compacto SendDirectorQuestion sin payload.
  - Error publico conserva code/field y no inventa state_after.
```

```text
Nombre: mcp.tool.orquesta.director.bootstrap_appspec.v0.puro
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-mcp
Consumidores: futura capa de servidor MCP local y clientes MCP con IA
Campos:
  descriptor:
    name: orquesta.director.bootstrap_appspec.v0
    version: v0
    input_schema: envelope compacto que referencia BootstrapProyectoDesdeAppSpecCommandV0
    resource_uri: orquesta://contracts/bootstrap-proyecto-desde-appspec/v0
    prompt_name: orquesta.bootstrap_proyecto_desde_appspec.v0
  input:
    request_id: opcional en envelope
    correlation_id: opcional en envelope
    respuesta: opcional; por defecto compacta
    command: orquestadirector.BootstrapProyectoDesdeAppSpecCommandV0
  output_ok:
    estado: ok
    request_id: normalizado desde envelope, command o app_spec
    correlation_id: normalizado desde envelope o command
    registro_aceptado: resumen compacto con refs, estado, contadores, warnings y bootstrap_version
    project_ref: ref opaca
    app_spec_ref: ref opaca
    workflow: resumen compacto de StartRun/RunStarted, contadores e idempotencia
  output_error:
    estado: error
    errores_publicos: lista compacta con code, message, field, retryable y correlation_id
Invariantes:
  - Tool puro testable sin servidor MCP real.
  - Invoca solo `orquesta-director.BootstrapProyectoDesdeAppSpecV0`.
  - Usa DTOs publicos de `orquesta-director`; no importa internals.
  - No ejecuta persistencia, runtime, DB, CLI ni filesystem productivo.
  - No llama por separado a factory, core ni workflow; esa composicion pertenece al director.
  - Devuelve JSON compacto para IA, sin AppSpec completo, backlog completo ni payloads JSON de workflow.
Errores:
  - director_bootstrap_invalido
  - errores publicos propagados por orquesta-core
  - errores publicos propagados por orquesta-core-workflow
Pruebas de contrato:
  - Descriptor/resource compacto sin detalles productivos.
  - Ejecucion ok contra el caso de uso puro del director.
  - Error publico de director sin inventar salida ok.
  - Saneamiento de JSON compacto sin filtrar AppSpec/backlog completos.
```

```text
Nombre: mcp.resource.orquesta.director.bootstrap_appspec.v0.puro
Tipo: dto
Version: v0
Propietario: orquesta-mcp
Consumidores: futura capa de servidor MCP local y clientes MCP con IA
Campos:
  descriptor:
    name: orquesta.director.bootstrap_appspec.v0
    version: v0
    uri: orquesta://contracts/bootstrap-proyecto-desde-appspec/v0
    content_type: application/vnd.orquesta.bootstrap-appspec.v0+json
    summary_key: mcp.resources.bootstrap_appspec.summary.v0
  resource:
    owner: orquesta-director
    canonical_refs: `../CONTRATOS.md#BootstrapProyectoDesdeAppSpec-v0` y `orquesta-director/docs/contratos.md`
    tool_name: orquesta.director.bootstrap_appspec.v0
    input/output: shapes compactos para IA
    composed_contracts: SolicitarNuevaApp, RegistrarProyectoDesdeAppSpec, StartRunFromAppSpec y OrchestrationCommand v0
    errores_publicos: errores del director y propagados por contratos compuestos
    guardrails: restricciones compactas del adaptador
Invariantes:
  - Resource puro sin lectura runtime de filesystem.
  - No reemplaza el detalle canonico del director.
  - Orienta a la IA antes de invocar el tool y evita dumps de contrato extensos.
Pruebas de contrato:
  - Descriptor/resource compacto.
  - Saneamiento sin secretos, DSN, HOME ni payloads completos.
```

```text
Nombre: mcp.tool.orquesta.apps.preparar_orquestacion.v0.puro
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-mcp
Consumidores: wizard web, servidor MCP futuro e IA directora
Campos:
  descriptor:
    name: orquesta.apps.preparar_orquestacion.v0
    version: v0
    input_schema: envelope compacto con AppSpecV0 validada
    resource_uri: orquesta://contracts/preparar-orquestacion-app/v0
  input:
    request_id, correlation_id: opcionales
    run_ref, project_ref, occurred_at, requested_by: opcionales
    app_spec: AppSpecV0 ya validada por orquesta-factory
  output_ok:
    estado: ok
    app_spec: resumen compacto
    run_ref, phase_id: refs del run preparado
    plan: resumen de unidades, dependencias, capacidades y write-set
    progress: total, listas, bloqueadas, pendientes y entregadas
    evidence_refs: refs compactas
  output_error:
    estado: error
    errores_publicos: code, field y message saneados
Invariantes:
  - Tool puro sin servidor MCP real, DB, runtime, filesystem productivo ni proveedor LLM.
  - Delega solo en `orquesta-app-runner.PrepareAppOrchestrationV0`.
  - No expone `CandidateProvider`; devuelve solo plan/progreso reconstruible.
  - No elige SQLite, Postgres, Docker, HOME, OAuth, modelo ni credenciales.
  - El plan grande se expresa por puertos/conectores y microtareas pequenas.
Pruebas de contrato:
  - AppSpec grande produce plan de 11 unidades y progreso inicial.
  - Error publico conserva el campo exacto del runner/planner.
  - Registro MCP ejecuta el tool puro sin binding externo.
```

```text
Nombre: mcp.tool.orquesta.apps.ejecutar_orquestacion.v0
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-mcp
Consumidores: wizard web, servidor MCP futuro e IA directora
Campos:
  descriptor:
    name: orquesta.apps.ejecutar_orquestacion.v0
    version: v0
    input_schema: envelope compacto con AppSpecV0 validada y limites de loop
    resource_uri: orquesta://contracts/ejecutar-orquestacion-app/v0
  input:
    request_id, correlation_id: refs externas opcionales; se traducen a refs internas neutras
    run_ref, project_ref: refs externas opcionales; no se propagan literalmente al core
    occurred_at: obligatorio
    app_spec: AppSpecV0 ya validada por orquesta-factory
    max_bursts, max_steps_per_burst, max_dispatches_per_wait, max_commands, max_outbox_per_cycle, max_external_waits: limites operativos
  output_ok:
    estado: ok
    app_spec: resumen compacto
    run_ref, phase_id: refs internas neutras del run
    plan: resumen de unidades y dependencias
    progress: avance calculado desde entregas durables
    loop_status: estado del loop progresivo
    started_agents: agentes arrancados por el loop
    evidence_refs: refs compactas
  output_error:
    estado: error
    errores_publicos: code, field y message saneados
Invariantes:
  - Adaptador fino: prepara mediante `PrepareAppOrchestrationV0` y ejecuta mediante `RunPreparedAppOrchestrationV0`.
  - Requiere puertos inyectados para run store, event sink, outbox ledger y dispatchers; sin binding queda `mcp_transport_tool_unbound`.
  - No elige DB, runtime, proveedor, modelo, HOME, OAuth ni credenciales.
  - No relaja validaciones del core; neutraliza detalles de transporte antes de crear run/outbox.
  - Los efectos salen solo por puertos hexagonales inyectados.
Pruebas de contrato:
  - Con puertos fake arranca la ola bootstrap desde AppSpec grande y queda esperando entrega externa.
  - Sin campos requeridos devuelve error publico.
  - El registro MCP publica el tool como opt-in y falla de forma compacta si no hay executor.
```

```text
Nombre: mcp.tool.orquesta.runs.control.v0
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-mcp
Consumidores: IA directora, web futura, operador humano y apps externas como
OPES
Campos:
  descriptor:
    name: orquesta.runs.control.v0
    input_schema: action pause|resume|stop|cancel, run_ref? o
      external_job_ref?, app_ref?, reason?, requested_by?, forced?,
      idempotency_key?, evidence_refs?
  output_ok:
    estado, run_ref, action, status, checkpoint_recorded, forced, evidence_refs
  output_error:
    errores_publicos compactos
Invariantes:
  - Adaptador inbound fino.
  - Delega solo en `RunControlWriterPortV0` inyectado.
  - `external_job_ref` se usa solo para resolver la run asociada por puerto; la
    accion controla la run completa, no un bloque OPES aislado dentro de una
    run compartida.
  - No usa DB, runtime, filesystem, scheduler interno ni proceso de agente.
Pruebas de contrato:
  - Executor delega en puerto fake.
  - Executor resuelve run por job externo mediante puerto fake.
  - Transporte queda opt-in sin binding.
  - HTTP POST delega y propaga correlacion.
```

```text
Nombre: mcp.tool.orquesta.run_queue.priority.v0
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-mcp
Consumidores: IA directora, runner global multiapp, web futura
Campos:
  descriptor:
    name: orquesta.run_queue.priority.v0
    input_schema: action rank|set_priority, queue_ref?, app_refs?, run_ref?,
      app_ref?, priority_score?, limit?, occurred_at?
  output_ok:
    rank: ranked[{rank,run_ref,app_ref,status,priority_score,aging_boost}]
    set_priority: updated{run_ref,app_ref,status,priority_score}
  output_error:
    errores_publicos compactos
Invariantes:
  - Ranking por `RunQueueReaderPortV0`.
  - Cambio de puntuacion solo por `RunQueuePriorityWriterPortV0`.
  - No lee DB ni muta scheduler interno.
Pruebas de contrato:
  - Ranking delega en `orquesta-run-queue`.
  - `set_priority` delega en writer fake.
  - Transporte queda opt-in sin binding.
```

```text
Nombre: mcp.tool.orquesta.server.shutdown.v0
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-mcp
Consumidores: IA directora, operador humano, CLI y web de administracion
Campos:
  descriptor:
    name: orquesta.server.shutdown.v0
    version: v0
    input_schema: envelope compacto con queue/app filters, forced,
      limites de supervisor, requested_by, reason, idempotency_key y
      evidence_refs
    resource_uri: orquesta://contracts/server-shutdown/v0
  input:
    request_id, correlation_id: refs externas opcionales
    queue_ref, app_refs, queue_limit: filtro de runs a cerrar
    max_ticks, max_runs_per_tick, max_executions: limites del drainer
    forced: true permite drenar sin exigir checkpoint previo
    requested_by, reason, idempotency_key, evidence_refs: auditoria compacta
  output_ok:
    estado: ok
    status: ready | waiting_drain | waiting_checkpoint
    shutdown_ready: true solo si todos los runs objetivo estan listos
    runs_requested, runs_stopped, agents_in_flight, checkpoints_pending,
      checkpoint_agents_pending
    runs: resumen por run con estado de control, checkpoint_ref opcional,
      pending_checkpoint_agent_refs, checkpoint_evidence_refs, stats y readiness
  output_error:
    estado: error
    errores_publicos compactos si faltan puertos obligatorios o falla HTTP
Invariantes:
  - Adaptador inbound fino: delega en `orquesta-server-shutdown`.
  - No envia senales al PID del servidor ni mata runtimes directamente.
  - No lee DB, filesystem, HOME, OAuth, proveedor ni modelo.
  - Usa solo puertos inyectados de RunQueue, RunControl, checkpoint,
    supervisor y stats.
  - El apagado real del proceso servidor queda en el borde externo cuando
    `shutdown_ready=true`.
Pruebas de contrato:
  - Descriptor/resource y registro MCP publican el tool.
  - Executor transforma input MCP y delega en puerto fake de shutdown.
  - HTTP `POST /api/v0/server/shutdown` conserva correlacion y errores
    publicos.
```

## Contrato minimo para mejorar una app existente

Orquesta acepta hoy dos entradas complementarias para pedir mejoras sobre una
app existente sin acoplar MCP/API a una implementacion concreta:

1. `AppSpecRequestV0` con `request_kind=modificar_app_existente` por el tool
   `orquesta.apps.arrancar_director.v0` o el bridge `POST /api/v0/apps/director`.
   Sirve para abrir un run de director sobre una app ya existente. El payload
   debe describir objetivo, tipo de app, alcance y restricciones; no debe
   incluir DB, runtime, proveedor, modelo, rutas HOME ni credenciales concretas.
2. `AutoprogrammingRequestV0` por el tool
   `orquesta.autoprogramming.validate_request.v0`. Sirve como preflight de
   alcance antes de pedir que Orquesta se modifique a si misma o a otro repo.
   El tool solo valida el contrato del core; no ejecuta agentes, tests, VCS,
   comandos ni cambios en disco.
   El mismo contrato esta expuesto por REST en
   `POST /api/v0/autoprogramming/validate-request` para que la web y otros
   adaptadores puedan validar una solicitud antes de arrancar director.

Payload minimo recomendado para mejorar este repo:

```json
{
  "request_id": "request-ref-autoprogramming-orquesta-001",
  "correlation_id": "corr-autoprogramming-orquesta-001",
  "autoprogramming_request": {
    "request_ref": "request-ref-autoprogramming-orquesta-001",
    "project_ref": "project-ref-orquesta",
    "worktree_ref": "worktree-ref-orquesta-aislada-001",
    "worktree_isolated": true,
    "branch_ref": "branch-ref-autoprogramming-orquesta-001",
    "tasks": [
      {
        "task_ref": "task-ref-mcp-autoprogramming-001",
        "area": "orquesta-mcp"
      }
    ],
    "write_set": [
      "modulos/orquesta-mcp/autoprogramming_validate_request_tool_v0.go",
      "modulos/orquesta-mcp/autoprogramming_validate_request_tool_v0_test.go",
      "modulos/orquesta-mcp/docs/contratos.md"
    ],
    "required_tests": [
      "go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-orchestration-core"
    ]
  }
}
```

Reglas minimas:

- `project_ref`, `worktree_ref` y `branch_ref` son referencias opacas.
- `worktree_isolated=true` es obligatorio.
- `tasks` se agrupa por area y queda limitado por defecto a 3 tareas y 2 areas.
- `write_set` debe ser relativo, pequeno y sin `..`; por defecto maximo 5 entradas.
- `required_tests` es obligatorio; la ejecucion real pertenece a un adaptador externo.

```text
Nombre: mcp.tool.orquesta.autoprogramming.validate_request.v0
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-mcp
Consumidores: cliente IA MCP, director futuro y operador
Campos:
  descriptor:
    name: orquesta.autoprogramming.validate_request.v0
    resource_uri: orquesta://contracts/autoprogramming-request/v0
  rest:
    method: POST
    path: /api/v0/autoprogramming/validate-request
  input:
    request_id, correlation_id: refs externas opcionales
    autoprogramming_request: AutoprogrammingRequestV0
  output_ok:
    estado: ok
    accepted: true
    groups, write_set, required_tests: proyecciones compactas validadas
  output_error:
    estado: error
    accepted: false
    errores_publicos: issues del core como code, field y message
Invariantes:
  - Adaptador inbound fino.
  - Delegacion unica en `ValidateAutoprogrammingRequestV0`.
  - No crea run, no persiste, no ejecuta tests, no toca VCS ni disco.
  - No elige implementacion de cambio, DB, runtime, proveedor ni modelo.
Pruebas de contrato:
  - Descriptor compacto y saneado.
  - Executor acepta una solicitud aislada pequena.
  - HTTP acepta JSON snake_case del contrato publico.
  - Executor devuelve issues publicos para branch/write-set invalidos.
  - Registro MCP publica el tool y permite invocarlo sin puertos productivos.
```
