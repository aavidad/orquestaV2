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
Nombre: mcp.tool.orquesta.runs.supervisor.v0
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-mcp
Consumidores: gateway HTTP, servidor residente, compatibilidad legacy y operadores automatizados
Campos:
  descriptor:
    name: orquesta.runs.supervisor.v0
    resource_uri: orquesta://contracts/run-supervisor/v0
  rest:
    method: POST
    path: /api/v0/runs/supervise
  input:
    request_id, correlation_id: refs externas opcionales
    run_ref: opcional; si existe limita la accion a esa run
    queue_ref: opcional; si no hay run_ref permite avanzar cola inyectada
    max_ticks, continue_message y limites acotados de drain/supervisor
    operator_advice: observaciones de operador no bloqueantes
  output_ok:
    estado: ok
    run_ref, stop_reason, ticks, last, history?, evidence_refs?,
    operator_advice?, diagnostics?; por HTTP, si el executor sigue vivo mas
    alla de la ventana de respuesta, devuelve `202 accepted` con
    `stop_reason=accepted_background`, `operation_ref`, diagnostico y acciones
    de consulta
  output_error:
    estado: error
    errores_publicos: issues compactos
Invariantes:
  - Adaptador inbound fino y opt-in por executor inyectado.
  - Compatibilidad legacy/resident: si la run tiene `GoalWorkStateV0`, el caller
    debe usar `observe_goal` y este tool no debe drenar el loop historico.
  - No usa stdin ni canal paralelo de agentes.
  - No conoce Codex, OPES, DB ni runtime concreto.
  - La composicion decide si el executor reentra por drain, cola global o
    supervisor residente.
  - El HTTP no mantiene al cliente bloqueado indefinidamente tras delegar en el
    executor; deja `operation_ref` y obliga a observar por stats/cola.
  - `operator_advice` se conserva como observacion; no decide runtime ni cierre.
Pruebas de contrato:
  - HTTP delega en executor fake y preserva correlation id.
  - HTTP devuelve `202 accepted_background` si el executor no responde dentro de
    la ventana acotada.
  - Transporte MCP queda opt-in y devuelve unbound si falta puerto.
  - Payload compacto sin secretos ni detalles internos.
```

```text
Nombre: mcp.tool.orquesta.external_work.run.v0
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-mcp
Consumidores: bridges de app externa, OPES legacy y operadores automatizados
Campos:
  descriptor:
    name: orquesta.external_work.run.v0
    resource_uri: orquesta://contracts/external-work-run/v0
  rest:
    method: POST
    path: /api/v0/external-work/run
  input:
    external_work_run_request?: StartExternalWorkRunRequestV0
    app_change_request?: AppChangeRequestV0
  output_ok:
    estado: ok
    route_policy: legacy_director_loop
    director_execution_mode: legacy_director_loop
    run_ref, change_ref, director_question_ref, evidence_refs?, next_actions?
  output_error:
    estado: error
    errores_publicos: issues compactos
Invariantes:
  - Compatibilidad legacy explicita hasta migrar external-work a GoalWorkSpecV0.
  - No crea GoalWorkStateV0 ni arranca Codex Goal.
  - Crea run operativo y encola para el loop historico por puertos inyectados.
  - No conoce OPES, DB, runtime, filesystem, proveedor ni credenciales.
Pruebas de contrato:
  - Descriptor y resultado declaran `route_policy=legacy_director_loop`.
  - HTTP delega en executor fake y preserva errores publicos.
  - Transporte MCP queda opt-in y devuelve unbound si falta puerto.
```

```text
Nombre: mcp.tool.orquesta.domain_work.v0
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-mcp
Consumidores: transporte/servidor MCP opt-in, cliente IA director y bridge HTTP local
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
  - OPES puede quedar detras de este tool mediante un adaptador REST inyectado;
    el tool no debe importar ni nombrar OPES.
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
      incluye `stop_control` con status `none`, `stop_requested`,
      `stop_propagated`, `stop_pending` o `stop_confirmed`, refs opacas de
      agentes pendientes/solicitados/confirmados y, si existe puerto
      `RunControl`, `run_control_status`, checkpoint, forced y evidencias
      compactas
    external_job: proyeccion compacta opcional con job_ref, work_kind,
      change_ref, run_ref, task_ref, agent_ref, status, status_reason,
      delivery_refs, issue_refs, evidence_refs y diagnostics publicos
    goal: proyeccion compacta opcional con `goal_ref`, `external_goal_ref`,
      status, cierre y evidencias si la composicion inyecta un `GoalStateStore`
    decision_context
Invariantes:
  - Si llega `external_job_ref`, MCP delega la resolucion en un puerto
    inyectado; no conoce OPES, app-change internals, filesystem ni runtime.
  - Si falta `run_ref`, el puerto puede resolverlo desde el job externo.
  - La salida conserva `DirectorRunStatsV0`; `external_job` es una proyeccion
    aditiva para que OPES no tenga que interpretar toda la run.
  - `external_job.status_reason`, `issue_refs` y `diagnostics` son senales
    operativas advisory del puerto inyectado; no son rails duros del core.
  - `goal` solo lee estado goal-first persistido por `run_ref`; no observa,
    lanza, cierra ni reintenta Goals desde stats.
  - Si se inyecta `RunControlReaderPortV0`, el tool puede distinguir parada de
    run solicitada antes de que haya `StoppedAgents` propagados.
Pruebas de contrato:
  - Executor resuelve `run_ref` por job externo mediante puerto fake.
  - Executor proyecta `stop_control` desde `RunControl` sin exponer runtime.
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
    director_execution_mode: vacio/`goal_first` exige backend Goal y
      `legacy_director_loop` fuerza compatibilidad historica explicita
    app_spec_request: AppSpecRequestV0 validado por factory
    max_bursts, max_steps_per_burst, max_dispatches_per_wait: limites operativos
  output_ok:
    estado: ok
    route_policy: entrada operativa preferente por Director V2
    app_spec: resumen compacto
    run_ref: ref interna neutra obligatoria para observar director, goal,
      evidencias y cierre
    phase_id: fase inicial
    director_execution_mode: `legacy_director_loop` o `goal_first`
    director_task: task_ref, brainstorm_ref, agent_request_id y capacidad
    director_tasks: lista compacta de tareas/directores arrancables cuando
      la solicitud requiere equipo director
    loop_status: estado compacto del loop
    started_agents: agentes arrancados por Orquesta
    goal_ref, external_goal_ref, goal_status: refs compactas cuando la
      composicion usa goal-first
    goal_launch_receipt: receipt neutral de `orquesta-goal`, sin prompt ni
      payload interno
  output_error:
    estado: error
    errores_publicos: issues de factory o servicio de aplicacion
    goal_backend_unavailable: error publico cuando el caller pide goal-first y
      la composicion no inyecta backend Goal completo
Invariantes:
  - Delegacion en `orquesta-app-director-service`.
  - No elige proveedor, modelo, HOME, OAuth, DB ni runtime.
  - No filtra nombres de adaptador como mcp/web/api a refs internas del core.
  - Permite que el formulario web invoque Orquesta sin conocer factory, workflow ni launcher.
	Pruebas de contrato:
	  - Ejecucion legacy explicita con puertos fake arranca director y devuelve
	    `started_agents`.
	  - Ejecucion con `GoalLauncher` devuelve `run_ref` y `goal_ref` sin
	    arrancar agentes legacy.
	  - `director_execution_mode=goal_first` sin backend devuelve error publico
	    y no cae al loop historico.
	  - `director_execution_mode=legacy_director_loop` conserva la rama legacy
	    de forma explicita.
	  - Autonomia alta en `legacy_director_loop` devuelve `director_tasks` y
	    arranca equipo por batch.
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
  - Convierte `ok` sin `run_ref` en error publico `run_ref_requerido`.
  - No conoce DB, runtime, proveedor, HOME, OAuth ni modelos.
Pruebas de contrato:
  - Handler HTTP invoca executor y conserva correlation id.
  - Handler HTTP rechaza `ok` sin `run_ref`.
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
    goal: bloque opcional goal-first con refs y estado persistido
    stats: DirectorRunStatsV0 completo serializado como contrato publico
    decision_context: DirectorDecisionContextV0 compacto para decision del
      director/API/MCP/web
    ops_snapshot: DirectorAutonomousOpsSnapshotV0 read-only para `/ops` y
      cockpit operativo
  output_error:
    estado: error
    errores_publicos
Cobertura de stats publicas:
  - tareas: counts.tasks_total, tasks_closed, tasks_open y refs open/closed;
  - agentes: counts agents_* y lista agents con estado/control/progreso opcional;
  - rework: counts.rework_requests y refs.rework_requests;
  - replan: counts.replan_decisions y refs.replan_decisions;
  - cierre: closure.status, blocked, ready, closed, blocked_by y blocker_refs;
  - progreso: progress.percent_complete, tasks, agents observados y issues;
  - goal-first: bloque `goal` opcional solo si existe estado persistido.
Cobertura de decision_context:
  - progreso por fase/tarea/agente;
  - sesiones/procesos por refs opacas cuando `include_process_refs=true`;
  - actividad reciente compacta por claves i18n;
  - agentes vivos/parados/fallidos y control registrado/ausente;
  - bloqueos y causas de cierre bloqueado;
  - rework/replan con refs opacas;
  - duraciones de fase cuando hay timestamps y quietud por falta de senal/ticks.
Cobertura de ops_snapshot:
  - cola opcional, runs, agentes, progreso, cierre, rework/replan y uso/cuota
    observado;
  - decision operativa compacta (`action`, `scope`, `reason_code`, refs) para
    que `/ops` no reconstruya la llamada del Director con heuristica cliente;
  - waits, olas y cohortes quedan como campos opcionales vacios si la fuente no
    los publico todavia.
Invariantes:
  - Delegacion en `RunStorePortV0`, `AgentProcessRegistryPortV0` opcional y
    `AgentProgressObservationProviderPortV0` opcional.
  - No crea stores, no lee DB, no consulta runtime real, no usa cmd y no expone
    rutas locales, credenciales, proveedor, modelo ni HOME.
  - No duplica reglas de cierre ni progreso; `ops_snapshot` deriva de
    `DirectorRunStatsV0`, `DirectorDecisionContextV0` y cola inyectada cuando
    existe.
  - `ops_snapshot` no bloquea, filtra ni descarta entregas; solo resume estado
    observable.
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
    `DirectorDecisionContextV0` y `ops_snapshot`
    `DirectorAutonomousOpsSnapshotV0`
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
Nombre: mcp.tool.orquesta.director_supervisor.briefing.v0
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-mcp
Consumidores: Hermes/API, web operativa y directores externos
Campos:
  descriptor:
    name: orquesta.director_supervisor.briefing.v0
    version: v0
    resource_uri: orquesta://contracts/director-supervisor-briefing/v0
  input:
    request_id, correlation_id: refs externas opcionales
    briefing_input: DirectorSupervisorBriefingInputV0
  output_ok:
    estado: ok
    run_ref
    briefing: DirectorSupervisorBriefingV0 con next_action, action_queue y timeline
  output_error:
    estado: error
    errores_publicos
Invariantes:
  - Tool puro y opt-in; si no se inyecta executor usa el ejecutor local puro.
  - Delegacion exclusiva en `orquesta-director-supervisor`.
  - No ejecuta la accion, no despacha outbox, no espera agentes y no persiste.
  - No conoce Codex, OPES, web, DB, runtime, proveedor, modelo, HOME ni OAuth.
  - No corta por strings recuperables; solo proyecta decisiones ya validadas.
Pruebas de contrato:
  - Executor puro devuelve briefing canonico.
  - Errores de decision incompleta salen como `errores_publicos`.
  - Registro MCP expone el tool y sus campos de entrada.
  - Transporte MCP real JSON-RPC puede invocarlo por `tools/call`.
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
Consumidores: clientes MCP con IA que quieran validar/previsualizar una spec sin usar CLI ni DB
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
  - Es compatibilidad de validacion/spec preview; la entrada operativa preferente
    para crear o ejecutar apps nuevas es `orquesta.apps.arrancar_director.v0`.
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
  - operator_mcp_connector_unavailable
Pruebas de contrato:
  - Resource lista capabilities y tools.
  - Tools validan envelopes antes de llamar al puerto.
  - Tools delegan solo con puerto fake inyectado.
  - Transporte propaga errores publicos del conector agregado y sirve estado,
    outbox y consulta dirigida con alias razonables.
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
    freshness: estado, fecha, refs de autoridad, refs de backlog y verificacion
    canonical_source: docs/estado_actual_2026-05-17.md
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
    backlog_refs: secciones vivas de backlog si el estado procede de automejora
    verification: prueba focal o runbook que valida la proyeccion
Invariantes:
  - Implementacion pura sin servidor MCP real, transporte, DB, CLI, runtime ni filesystem productivo.
  - El detalle extenso queda en el modulo propietario y en documentos vigentes de estado/nucleo.
  - No incluye secretos, transcripts, dumps de fixtures ni detalles de sink/tablas/driver; solo declara endpoints REST cuando forman parte del contrato publico vigente.
  - Usa message keys para resumen y progreso en vez de texto largo localizado.
  - No duplica estados `pendiente_*`: enlaza backlog vivo y freshness cuando un frente sigue abierto.
  - Cubre `SolicitarNuevaApp`, `PersistenceRepository`, `RuntimeLaunchRequest`, `OrquestaEvent`, `GovernanceCatalog`, `OperationalStatusQuery`, `DeploymentPlan` y `GenerarI18nDocsIniciales` v0.
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
    scope: orquesta-nucleo-reutilizable
    freshness: estado, fecha, refs de autoridad, refs de backlog y verificacion
    canonical_refs: referencias compactas a estado vigente, guia del nucleo y docs locales de modulos propietarios
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
    backlog_refs: secciones vivas de backlog si el hito sigue abierto o fue sincronizado por automejora
    verification: prueba focal o runbook que valida el hito
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
  - No lee docs de autoridad ni backlog en runtime; el resource es una proyeccion estatica versionada con freshness.
  - Permite a una IA leer roadmap/decisiones de nucleo sin cargar docs globales completos.
  - El detalle canonico permanece en docs vigentes y modulos propietarios.
  - No incluye secretos, transcripts, dumps de fixtures, endpoints REST, sinks operativos, DSN ni tablas.
  - No duplica estados `pendiente_*`: marca compatibilidad historica o enlaza backlog vivo con owner y verificacion.
  - Cubre compatibilidad AppSpec, WorkflowTask/WorkProfile, persistence/events, runtime/capacity y DomainWork para consumidores.
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
  - Queda como preview/compatibilidad, no como cierre operativo.
  - La entrada operativa preferente para apps nuevas es `orquesta.apps.arrancar_director.v0`.
  - No expone `CandidateProvider`; devuelve solo plan/progreso reconstruible.
  - No elige SQLite, Postgres, Docker, HOME, OAuth, modelo ni credenciales.
  - El plan grande se expresa por puertos/conectores y microtareas pequenas.
Pruebas de contrato:
  - AppSpec grande produce plan de 11 unidades y progreso inicial.
  - El resultado declara `route_policy` con `orquesta.apps.arrancar_director.v0` como preferente.
  - Error publico conserva el campo exacto del runner/planner.
  - Registro MCP ejecuta el tool puro sin binding externo.
```

```text
Nombre: mcp.tool.orquesta.apps.ejecutar_orquestacion.v0
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-mcp
Consumidores: compatibilidad historica, smokes legacy y diagnostico operador
Campos:
  descriptor:
    name: orquesta.apps.ejecutar_orquestacion.v0
    version: v0
    input_schema: envelope compacto con `director_execution_mode=legacy_director_loop`, AppSpecV0 validada y limites de loop
    resource_uri: orquesta://contracts/ejecutar-orquestacion-app/v0
  input:
    request_id, correlation_id: refs externas opcionales; se traducen a refs internas neutras
    director_execution_mode: obligatorio como `legacy_director_loop`; vacio o `goal_first` no ejecutan el loop
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
  - Adaptador fino de compatibilidad: prepara mediante `PrepareAppOrchestrationV0` y ejecuta mediante `RunPreparedAppOrchestrationV0` solo con `director_execution_mode=legacy_director_loop`.
  - La entrada normal para apps nuevas, goal-first y trabajo productivo es `orquesta.apps.arrancar_director.v0`.
  - Vacio o `goal_first` devuelve `legacy_director_loop_required` con `preferred_entrypoint=orquesta.apps.arrancar_director.v0`, sin preparar ni ejecutar el loop antiguo.
  - Requiere puertos inyectados para run store, event sink, outbox ledger y dispatchers; sin binding queda `mcp_transport_tool_unbound`.
  - No elige DB, runtime, proveedor, modelo, HOME, OAuth ni credenciales.
  - No relaja validaciones del core; neutraliza detalles de transporte antes de crear run/outbox.
  - Los efectos salen solo por puertos hexagonales inyectados.
Pruebas de contrato:
  - Sin `director_execution_mode=legacy_director_loop` devuelve `legacy_director_loop_required`.
  - Con puertos fake y legacy explicito arranca la ola bootstrap desde AppSpec grande y queda esperando entrega externa.
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
   El tool solo valida el contrato de `orquesta-autoprogramming`; no ejecuta
   agentes, tests, VCS, comandos ni cambios en disco.
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
      "go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-autoprogramming"
    ]
  }
}
```

Reglas minimas:

- `project_ref`, `worktree_ref` y `branch_ref` son referencias opacas.
- AppVCS acepta `review_repo` como accion read-only por MCP y por
  `POST /api/v0/apps/vcs`; el conector real se inyecta desde composicion.
- `worktree_isolated=true` es obligatorio.
- `tasks` se agrupa por area y queda limitado por defecto a 3 tareas y 2 areas.
- Cada tarea puede transportar objetivo, contexto, context refs, criterios,
  tests y reglas compactas; MCP los pasa al contrato puro sin interpretarlos
  desde `task_ref`.
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
    programmable_work.groups: title, summary, criteria, required_tests y
      context_refs compactos generados para cada grupo
  output_error:
    estado: error
    accepted: false
    errores_publicos: issues de orquesta-autoprogramming como code, field y message
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

```text
Nombre: mcp.tool.orquesta.autoprogramming.self_improvement.propose.v0
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-mcp
Consumidores: cliente IA MCP, bridge HTTP local, operador humano y composiciones opt-in
Campos:
  descriptor:
    name: orquesta.autoprogramming.self_improvement.propose.v0
    resource_uri: orquesta://contracts/autoprogramming-self-improvement/v0
  rest:
    method: POST
    path: /api/v0/autoprogramming/self-improvement
  input:
    request_id, correlation_id, auto_prepare_run?
    proposal: AutoprogrammingSelfImprovementProposalV0 con refs opacas,
      worktree aislada, write-set propio, tests requeridos y evidencia
    operator_advice?: consejos humanos compactos, no bloqueantes; acepta
      aliases reparables `subject_ref`, `target`, `ref`, `run`, `task`,
      `kind`, `advice` y `text`
  output_ok:
    estado: ok
    autoprogramming_request, priority_score, prepare_run, prepared_run?
    operator_advice?: consejos normalizados con non_blocking=true
    next_actions
  output_error:
    estado: error
    errores_publicos y next_actions reparables sin descartar evidencia
    operator_advice?: consejos normalizados si llegaron en la entrada
Invariantes:
  - Adaptador inbound fino.
  - Convierte fallos observados en automejora de baja prioridad.
  - Solo prepara run si `auto_prepare_run=true` y hay executor inyectado.
  - `operator_advice` permite observar y aconsejar sin bloquear ni decidir
    runtime, proveedor, cola, Codex, OPES, DB ni filesystem.
  - El consejo normalizado se conserva como contexto/regla no bloqueante de la
    request de automejora para que el agente vea la orientacion humana.
```

```text
Nombre: mcp.tool.orquesta.autoprogramming.prepare_run.v0
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-mcp
Consumidores: cliente IA MCP, bridge HTTP local y composiciones opt-in
Campos:
  descriptor:
    name: orquesta.autoprogramming.prepare_run.v0
    resource_uri: orquesta://contracts/autoprogramming-prepare-run/v0
  rest:
    method: POST
    path: /api/v0/autoprogramming/prepare-run
  input:
    request_id, correlation_id: refs externas opcionales
    autoprogramming_request: AutoprogrammingRequestV0 validado por el executor
      inyectado
    max_bursts, max_steps_per_burst, max_dispatches_per_wait, max_commands,
      max_outbox_per_cycle: limites operativos opcionales
  output_ok:
    estado: ok
    accepted: true
    run_ref?, project_ref, worktree_ref, branch_ref, phase_id?
    workflow_task_refs?, wait_agent_refs?
    goal?: estado durable de lanzamiento Goal-first cuando la composicion tiene
      backend inyectado: director_execution_mode, run_ref, goal_ref,
      external_goal_ref?, goal_status?, evidence_refs?
    goals?: lista canonica de estados Goal-first por lote; cuando hay mas de
      un goal, `goal` se omite y `run_ref` superior identifica el primer
      `goals[i].run_ref` solo por compatibilidad, no un run agregado
    goal_specs: contratos `GoalWorkSpecV0` opcionales cuando la composicion
      clasifica el trabajo como `goal_ready`; sin backend preparan el handoff a
      Goal sin materializar ni encolar un run legacy, y con backend quedan
      devueltos con `run_ref` del run contenedor o de cada goal derivado
    continue: request compacta opcional para supervision legacy posterior con
      `run_ref` explicito
  output_error:
    estado: error
    accepted: false
    errores_publicos: issues compactos
Invariantes:
  - Adaptador inbound fino.
  - Prepara un run legacy continuable, un handoff Goal-first o un lanzamiento
    Goal-first persistido por executor inyectado.
  - No arranca agentes por si mismo ni supervisa despues del prepare.
  - No conoce Codex, OPES, DB, filesystem, runtime productivo ni proveedor.
  - El registro MCP es opt-in: sin executor devuelve
    `mcp_transport_tool_unbound`.
Pruebas de contrato:
  - Descriptor compacto y saneado.
  - Registro MCP publica el tool.
  - Transporte bound invoca executor fake y devuelve resultado `ok`.
  - Descriptor y transporte publican `goal_specs` como salida opcional.
  - Descriptor y HTTP publican `goals[]` tipado para batch goal-first.
  - Transporte sin executor devuelve `mcp_transport_tool_unbound`.
  - HTTP `POST /api/v0/autoprogramming/prepare-run` delega en executor fake.
```

```text
Nombre: mcp.tool.orquesta.autoprogramming.observe_goal.v0
Campos:
  descriptor:
    name: orquesta.autoprogramming.observe_goal.v0
    resource_uri: orquesta://contracts/autoprogramming-observe-goal/v0
  rest:
    method: POST
    path: /api/v0/autoprogramming/goal/observe
  input:
    request_id, correlation_id: refs externas opcionales
    run_ref: ref durable de un run Goal-first; en batch usar
      `goals[i].run_ref`
    occurred_at, requested_by: metadata opcional
  output_ok:
    estado: ok
    run_ref, run_status?, director_execution_mode?
    goal_ref, external_goal_ref?, goal_status?
    closure_status?, closure_accepted?, closure_needs_rework?
    summary?, artifact_refs?, domain_receipt_refs?, evidence_refs?,
    closure_issues?
  output_error:
    estado: error
    errores_publicos
Invariantes:
  - Adaptador inbound fino.
  - Observa un `GoalWorkStateV0` ya persistido por `run_ref`.
  - No ejecuta loop legacy ni arranca proveedor.
  - La validacion de cierre queda en el servicio neutral de Goal.
  - El registro MCP es opt-in: sin executor devuelve
    `mcp_transport_tool_unbound`.
```

```text
Nombre: mcp.tool.orquesta.autoprogramming.status.v0
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-mcp
Consumidores: cliente IA MCP, bridge HTTP local y composiciones opt-in
Campos:
  descriptor:
    name: orquesta.autoprogramming.status.v0
    resource_uri: orquesta://contracts/autoprogramming-status/v0
  rest:
    method: POST
    path: /api/v0/autoprogramming/status
  input:
    request_id, correlation_id, run_ref?, external_job_ref?, queue_ref?,
    app_refs?, queue_limit?, telemetry_flags?
  rest_extension:
    operator_advice?: consejos humanos compactos; el bridge HTTP normaliza
      aliases reparables (`subject_ref`, `target`, `ref`, `run`, `task`,
      `kind`, `advice`, `text`) y los devuelve sin alterar el resultado del
      executor, incluso cuando el puerto inyectado falta o falla
  output_ok:
    estado: ok
    queue?: resultado compacto de orquesta.run_queue.priority.v0
    run?: resultado compacto de orquesta.director.stats.v0
    queue_health?: separa `running_live`, `agents_live`,
      `running_without_recent_stats` y `running_stale_no_process`;
      `running_live` cuenta runs con liveness probado, `agents_live` cuenta
      agentes vivos observados por progreso o por `process.status`
      `running`/`stopping`, y
      `running_stale` agregado solo cuenta stale verificable sin proceso vivo
    stale_running?: acciones publicas; una run `running` sin liveness probado se
      expone como `running_without_recent_stats`, no como stale terminal
      tambien incluye external-work terminal `stopped` sin agentes ni entregas
      como `external_work_accepted_stopped_without_delivery`
    ops_snapshot?: DirectorAutonomousOpsSnapshotV0 agregado de cola/run para
      `/ops` y cockpit operativo
    diagnostics?: diagnostico publico de puertos/errores y consejo no bloqueante
      y issues de progreso del run, incluido
      `external_work_agent_requested_not_started` cuando hay agentes pedidos,
      ninguno arrancado y ninguna senal viva; para compatibilidad con
      consumidores existentes, las refs `opes...`/`app-spec-opes...` se tratan
      como trabajo externo aunque no incluyan literalmente `external-work`
  output_error:
    estado: error
    errores_publicos reparables
    operator_advice?: consejos normalizados si llegaron por bridge HTTP
    diagnostics?: diagnostico publico, incluido consejo no bloqueante
Invariantes:
  - Adaptador inbound fino.
  - Delega cola en `run_queue.priority` y run en `director.stats`.
  - Si la composicion inyecta `GoalWorkStateStore` y la run pertenece a
    goal-first, recomienda observar el goal por
    `/api/v0/autoprogramming/goal/observe` en vez de empujar supervision legacy
    de esa run. Cuando la consulta es de cola y hay candidatos goal-first, no
    recomienda `supervise queue`; emite `observe_goal` por run goal-first y
    `supervise run` acotado para candidatos legacy visibles.
  - Las clases `running_without_recent_stats`, `running_stale*` y las
    proyecciones legacy de tareas/agentes (`registered`, `process_ref`,
    `task_ref`) aplican solo a runs sin `GoalWorkStateV0`. En goal-first el
    estado operativo se expresa por `goal_status`, `closure_status`,
    `observe_goal` y evidencias de cierre del goal.
  - Para runs `running` visibles en cola puede consultar `director.stats` de
    forma acotada y con `include_process_refs`; si solo hay refs de proceso sin
    `process.status` verificado, publica `running_without_recent_stats`; solo
    publica `running_stale_no_process` cuando no hay proceso vivo verificado.
  - Si `director.stats` informa issues de progreso, los conserva como
    diagnosticos publicos accionables en vez de convertirlos en fallo terminal.
  - El bridge HTTP puede transportar consejo del operador como observacion no
    bloqueante sin tocar el caso de uso.
  - `ops_snapshot` es read-only, no decide runtime ni corta entregas; deriva de
    `queue` y `run` ya publicados.
  - No conoce stores, runtime, Codex, DB, filesystem ni proveedor.
```

```text
Nombre: mcp.tool.orquesta.autoprogramming.supervise.v0
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-mcp
Consumidores: cliente IA MCP, bridge HTTP local y composiciones opt-in
Campos:
  descriptor:
    name: orquesta.autoprogramming.supervise.v0
    resource_uri: orquesta://contracts/autoprogramming-supervise/v0
  rest:
    method: POST
    path: /api/v0/autoprogramming/supervise
  input:
    request_id, correlation_id, run_ref?, queue_ref?, max_ticks?, limits?,
    operator_advice? por bridge HTTP con aliases reparables de refs, accion y
    mensaje humano
  output_ok:
    mismo resultado compacto de `orquesta.runs.supervisor.v0` y, por HTTP,
    eco opcional de operator_advice no bloqueante; si la ejecucion sigue viva
    mas alla de la ventana de respuesta HTTP, devuelve `202 accepted` con
    `operation_ref`, diagnostico y siguientes acciones de consulta
  output_error:
    mismo error publico reparable de `orquesta.runs.supervisor.v0` y, por HTTP,
    eco opcional de operator_advice no bloqueante si el puerto falta o falla
Invariantes:
  - Adaptador inbound fino.
  - Delega la supervision puntual en `runs.supervisor` inyectado.
  - El HTTP no mantiene al cliente bloqueado indefinidamente si ya delego el
    trabajo; responde parcial y deja evidencia consultable por estado de run/cola.
  - El consejo del operador no detiene ni reemplaza la decision del supervisor.
  - No conoce scheduler, runtime, Codex, DB, filesystem ni proveedor.
```

```text
Nombre: mcp.tool.orquesta.runtime.models.v0
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-mcp
Consumidores: cliente IA MCP, bridge HTTP local, UI operativa y composiciones
opt-in de runtime.
Campos:
  descriptor:
    name: orquesta.runtime.models.v0
    resource_uri: orquesta://contracts/runtime-models/v0
  rest:
    method: POST
    path: /api/v0/runtime/models
  input:
    action: list|status|pull|serve|stop
    provider_ref?, endpoint_ref?, model?, keep_alive?, tags?, evidence_refs?
  output_ok:
    estado: ok
    action
    list_result? o action_result?
  output_error:
    estado: error
    errores_publicos
Invariantes:
  - Adaptador inbound fino para `RuntimeModelManagerPortV0`.
  - No acepta `base_url` en el input publico; endpoints, tokens y timeouts se
    configuran en la composicion opt-in.
  - Sin puerto inyectado devuelve `mcp_transport_tool_unbound` por MCP o
    `runtime_models_no_configurado` por HTTP, sin intentar instalar nada.
  - No decide proveedor/modelo de una tarea, no arranca agentes, no toca DB,
    scheduler, OPES ni programacion.
Pruebas de contrato:
  - go test -count=1 ./modulos/orquesta-mcp
```

```text
Nombre: mcp.tool.orquesta.apps.observe_director_goal.v0
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-mcp
Consumidores: cliente IA MCP, bridge HTTP local, web nueva-app y composiciones
opt-in con runtime goal.
Campos:
  descriptor:
    name: orquesta.apps.observe_director_goal.v0
    resource_uri: orquesta://contracts/observe-app-director-goal/v0
  rest:
    method: POST
    path: /api/v0/apps/director/goal/observe
  input:
    request_id?, correlation_id?, run_ref, occurred_at?, requested_by?
  output_ok:
    estado: ok
    run_ref, run_status?, director_execution_mode?, goal_ref,
    external_goal_ref?, goal_status, closure_status?, closure_accepted?,
    closure_needs_rework?, artifact_refs?, domain_receipt_refs?,
    evidence_refs?
  output_error:
    estado: error
    errores_publicos
Invariantes:
  - Adaptador inbound fino para observar un goal ya lanzado.
  - No ejecuta loop legacy, no arranca proveedor y no decide cierre.
  - Delega observacion y validacion causal en `orquesta-app-director-service`.
  - `run_ref` es obligatorio porque acota el estado goal-first persistido.
  - La composicion puede envolver el executor para efectos propios, como
    sincronizar cola en `orquesta-app-codex-stack`.
Pruebas de contrato:
  - go test -count=1 ./modulos/orquesta-mcp -run 'ObserveAppDirectorGoal|MCPTransportToolInputSchema|RegisterMCPTransport'
```
