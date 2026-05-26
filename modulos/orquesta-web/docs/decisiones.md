# Decisiones locales: orquesta-web

Las decisiones de este archivo solo afectan a `orquesta-web`. Si afectan a otro modulo, deben elevarse al director y registrarse en `../../CONTRATOS.md`.

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

## Decisiones iniciales

```text
Fecha: 2026-05-15
Decision: `stalled` informativo no cambia el panel a `attention`.
Motivo: un agente premium puede estar pensando varios ticks sin escribir artefactos. La web debe mostrar esa telemetria, pero solo pedir atencion si core marca `needs_attention`, hay `loop_detected`, proceso `stopped` o checkpoint pendiente.
Alternativas: mantener cualquier `stalled` como alerta; ocultar stalled; duplicar reglas de decision en web.
Impacto: `directorStatsAttentionStateV0` ya no usa `stalled_agents` como disparador de atencion. El contador sigue visible para supervision, pero no implica actuacion humana ni del director.
Contratos afectados: `WebDirectorStatsViewModelV0`, `WebDirectorStatsProgressV0`.
Estado: aceptada localmente.
```

```text
Fecha: 2026-05-04
Decision: `orquesta-web` arrancara el flujo de nueva app como adaptador fino de `SolicitarNuevaApp v0`.
Motivo: El contrato publico de `orquesta-factory` define que el puerto valida, normaliza y propone, sin persistir estado, crear tareas directas, arrancar runtime ni decidir reglas de negocio en la UI.
Alternativas: Reusar directamente `/nueva-app` heredado y `/api/proyectos/{ref}/fabricar-app`; crear una UI nueva con reglas locales; esperar a que exista core completo.
Impacto: La web solo traduce formulario/estado de UI a `AppSpecRequestV0` y renderiza `AppSpecV0`/`BacklogInicialPropuestoV0` como proyecciones compactas. Toda recomendacion o validacion de negocio queda en factory.
Contratos afectados: Consume `SolicitarNuevaApp v0`, `AppSpecRequestV0`, `AppSpecV0`, `BacklogInicialPropuestoV0`.
Estado: aceptada localmente.
```

```text
Fecha: 2026-05-04
Decision: Reutilizar de la web heredada solo inventario, vocabulario de campos, cobertura de casos y textos i18n; no copiar handlers, rutas ni JavaScript de recomendacion en el arranque.
Motivo: La herencia en `cmd/proyectos_web.go` contiene una taxonomia rica para pedir apps, pero mezcla render, transformacion, recomendaciones duplicadas en cliente, preview por API antigua y fabricacion/materializacion de backlog.
Alternativas: Copiar el template completo y recortar despues; migrar solo tests; descartar todo.
Impacto: Las microtareas empiezan por contratos, mapper y view-model antes de cualquier UI. Se evita importar reglas de negocio al adaptador web.
Contratos afectados: `WebNuevaAppFormV0`, `WebNuevaAppViewModelV0`, `SolicitarNuevaApp v0`.
Estado: aceptada localmente.
```

```text
Fecha: 2026-05-04
Decision: Mantener en cuarentena el endpoint heredado `/api/proyectos/{ref}/fabricar-app` para el primer flujo de `orquesta-web`.
Motivo: El endpoint antiguo usa proyecto existente, genera plan con `fabricaapp` y materializa tareas en DB. Eso contradice el corte pedido: `SolicitarNuevaApp v0` no persiste, no crea tareas directas y no arranca runtime.
Alternativas: Usarlo como backend temporal; envolverlo en otro adaptador; bloquear el flujo hasta que core tenga registro de proyecto.
Impacto: El primer slice web no promete crear proyecto ni backlog real. Solo solicita spec/propuesta y muestra fases o preguntas abiertas devueltas por factory.
Contratos afectados: No cambia contrato global; registra restriccion local de consumo.
Estado: aceptada localmente.
```

```text
Fecha: 2026-05-04
Decision: Las recomendaciones de DB, frontend, auth, deploy y artefacto se mostraran solo si vienen de `SolicitarNuevaApp v0` como defaults, warnings o campos normalizados; la web no recalculara esas reglas.
Motivo: La herencia duplica reglas en Go y JavaScript (`Recommend*` y funciones JS equivalentes). En el nuevo corte esas decisiones pertenecen a factory.
Alternativas: Portar las funciones JS; llamar a funciones de factory desde componentes; esconder recomendaciones.
Impacto: Las vistas deben distinguir valor elegido, default aplicado y warning del contrato. Cualquier recomendacion no cubierta por contrato se eleva como consulta antes de implementarse.
Contratos afectados: `AppSpecV0.defaults_applied`, `AppSpecV0.validation`.
Estado: aceptada localmente.
```

```text
Fecha: 2026-05-04
Decision: El transporte inicial entre orquesta-web y orquesta-factory sera REST/API, oculto detras del puerto local `SolicitarNuevaAppClient`.
Motivo: La web debe poder consumir `SolicitarNuevaApp v0` sin saber que hay debajo. REST da una frontera clara para el primer adaptador y mantiene sustituible el transporte; el envelope REST canonico queda definido en `../../CONTRATOS.md`.
Alternativas: llamada directa en proceso; MCP tool; DB compartida; acoplar handler web a factory.
Impacto: La UI y handlers de pantalla dependen de `SolicitarNuevaAppClient`; la implementacion inicial sera un adaptador REST versionado con request `AppSpecRequestV0`, 2xx `app_spec` + `backlog` y 400 `errores`.
Contratos afectados: `SolicitarNuevaAppClient`, `SolicitarNuevaApp v0`.
Estado: aceptada por el director y registrada.
```

```text
Fecha: 2026-05-04
Decision: Implementar WEB-001 y WEB-003 como paquete Go puro `orquestaweb`, sin HTTP handlers, templates, filesystem, DB, runtime ni cliente de transporte.
Motivo: El corte pedido es de codigo puro y debe permitir arrancar Codex desde `modulos/orquesta-web` manteniendo contratos locales claros.
Alternativas: Esperar a definir framework UI; crear handlers falsos; usar fixtures JSON como frontera primaria.
Impacto: `WebNuevaAppFormV0` solo captura y normaliza entrada web superficial; `WebNuevaAppViewModelV0` solo proyecta DTOs publicos de factory a un estado compacto renderizable. La integracion real queda para WEB-002/WEB-006.
Contratos afectados: `WebNuevaAppFormV0`, `WebNuevaAppViewModelV0`, `AppSpecRequestV0`, `AppSpecV0`, `BacklogInicialPropuestoV0`.
Estado: aceptada localmente.
```

```text
Fecha: 2026-05-04
Decision: El flujo web v0 no requiere proyecto existente.
Motivo: `SolicitarNuevaApp v0` solo valida, normaliza y propone; exigir proyecto existente reintroduce dependencia prematura con core/persistencia y con el flujo heredado `fabricar-app`.
Alternativas: exigir selector de proyecto; crear proyecto automaticamente; bloquear la web hasta que core implemente registro.
Impacto: La primera pantalla pide una app independiente y muestra `AppSpecV0` + `BacklogInicialPropuestoV0`; el registro de proyecto/backlog queda para `RegistrarProyectoDesdeAppSpec v0`.
Contratos afectados: `WebNuevaAppFormV0`, `SolicitarNuevaAppClient`, futuro `RegistrarProyectoDesdeAppSpec v0`.
Estado: aceptada por el director y registrada.
```

```text
Fecha: 2026-05-04
Decision: WEB-004 fija el vocabulario i18n inicial de nueva app como catalogo Go puro en memoria.
Motivo: La futura pantalla debe renderizar texto visible desde claves estables sin copiar la UI heredada ni introducir templates antes de cerrar el contrato de vocabulario.
Alternativas: Cargar JSON heredado completo; esperar al render; usar literals temporales en la UI futura.
Impacto: `NuevaAppI18nCatalogV0` cubre `es-ES` y `en-US`, valida claves requeridas y ofrece fallback controlado al locale default para que WEB-005 pueda renderizar sin literals sueltos.
Contratos afectados: `NuevaAppI18nCatalogV0`, futuro `NuevaAppPage`.
Estado: aceptada localmente.
```

```text
Fecha: 2026-05-04
Decision: WEB-002 implementa solo el puerto cliente REST de salida, no el submit web ni servidor API.
Motivo: El corte pedido necesita cerrar la frontera local para consumir `SolicitarNuevaApp v0` sin introducir handlers, pantalla, DB, runtime, filesystem ni MCP.
Alternativas: implementar handler de formulario junto al cliente; invocar factory en proceso; fijar un servidor REST de factory desde web.
Impacto: Los futuros handlers dependeran de `SolicitarNuevaAppClientV0`; la implementacion actual manda `AppSpecRequestV0` a `POST /api/v0/apps/spec`, usa `X-Correlation-ID`, timeout configurable y consume el envelope canonico global. Los alias `spec` y `backlog_inicial_propuesto` son compatibilidad transitoria, no contrato canonico.
Contratos afectados: `SolicitarNuevaAppClientV0`, `SolicitarNuevaApp v0`, `WebNuevaAppViewModelV0`.
Estado: aceptada localmente.
```

```text
Fecha: 2026-05-04
Decision: WEB-006A cubre la integracion real web -> REST -> factory solo con test vertical.
Motivo: Permite validar el contrato REST canonico con el handler publico de factory sin introducir UI, handler de submit, DB, filesystem, runtime, MCP ni servidor web propio.
Alternativas: Crear un handler web de POST; llamar factory en proceso; ampliar tests unitarios con fakes REST.
Impacto: El flujo demuestra que `WebNuevaAppFormV0` puede cruzar `RESTSolicitarNuevaAppClientV0` y volver como `WebNuevaAppViewModelV0`; el submit real queda fuera de este corte.
Contratos afectados: `SolicitarNuevaAppClientV0`, `SolicitarNuevaApp v0`, `WebNuevaAppFormV0`, `WebNuevaAppViewModelV0`.
Estado: aceptada localmente.
```

```text
Fecha: 2026-05-04
Decision: WEB-005/WEB-006 se implementan como handler `net/http` puro con respuesta JSON estructurada.
Motivo: El modulo todavia no tiene sistema de templates ni framework UI propio, y el corte pedido exige GET/POST pequenos sin arrancar servidor real ni introducir filesystem productivo.
Alternativas: Crear templates HTML; arrancar un router/servidor propio; posponer el submit hasta una UI real.
Impacto: `NuevaAppWebEndpointV0` renderiza textos i18n, opciones de locale y campos derivados de `WebNuevaAppFormV0`; POST delega en `SolicitarNuevaAppClientV0` y solo proyecta el `WebNuevaAppViewModelV0`.
Contratos afectados: `NuevaAppWebEndpointV0`, `WebNuevaAppFormV0`, `SolicitarNuevaAppClientV0`, `WebNuevaAppViewModelV0`, `NuevaAppI18nCatalogV0`.
Estado: aceptada localmente.
```

```text
Fecha: 2026-05-04
Decision: WEB-007 se implementa como `backlog_preview` JSON dentro de `WebNuevaAppViewModelV0`.
Motivo: El modulo actual todavia es adaptador web fino con handler JSON; un preview compacto satisface la necesidad de operador sin crear frontend pesado ni materializar backlog.
Alternativas: Crear template HTML especifico; ampliar la respuesta con dump completo de factory; posponer hasta una UI final.
Impacto: El preview deriva de `BacklogInicialPropuestoV0` y expone fases/microtareas con key, titulo, modulo/frontera, write-set, bloqueos y criterio de cierre. La respuesta conserva las proyecciones previas por compatibilidad.
Contratos afectados: `WebNuevaAppViewModelV0`, `WebNuevaAppBacklogPreviewV0`, `NuevaAppWebEndpointV0`.
Estado: aceptada localmente.
```

```text
Fecha: 2026-05-04
Decision: WEB-008 se protege con un test de arquitectura local sobre ficheros productivos `nueva_app_*.go`.
Motivo: El flujo nueva app debe seguir siendo adaptador fino y no volver al camino heredado de fabricar/materializar backlog desde web.
Alternativas: usar solo revision manual; bloquear cualquier texto `db`/`runtime`; crear linter global.
Impacto: El test falla con imports directos a DB/runtime/cmd/herencia y con literales de endpoints/materializacion heredados, pero permite campos contractuales como `db_required`.
Contratos afectados: NoDBNoRuntimeGuard, NuevaAppWebEndpointV0, SolicitarNuevaAppClientV0.
Estado: aceptada localmente.
```

```text
Fecha: 2026-05-04
Decision: WEB-009 implementa estado operativo web como proyeccion pura de `DiagnosticoCompactoV0`.
Motivo: El panel de operador necesita ver fase actual, progreso y bloqueos sin que la web lea DB, runtime, event sink, filesystem o internals de observability.
Alternativas: crear endpoint/cliente REST ya; leer eventos directamente; reutilizar pantallas heredadas de progreso; esperar a un panel frontend completo.
Impacto: `WebOperationalStatusPanelV0` y `NewWebOperationalStatusQueryV0` dejan preparada la frontera read-only con consumidor `orquesta-web/web`, pero no hacen I/O ni acciones de recuperacion.
Contratos afectados: `OperationalStatusQueryV0`, `DiagnosticoCompactoV0`, `WebOperationalStatusPanelV0`.
Estado: aceptada localmente.
```

```text
Fecha: 2026-05-04
Decision: WEB-010 divide `nueva_app_endpoint_v0.go` por responsabilidad sin tocar contratos ni comportamiento.
Motivo: La regla global marca el fichero en zona roja y recomienda separar handler, request/response y render JSON cuando se toque esa zona.
Alternativas: Mantener el fichero monolitico; partir tambien tests; cambiar el handler a un framework/router real.
Impacto: `nueva_app_endpoint_v0.go` queda como identidad/configuracion; `nueva_app_endpoint_handler_v0.go` contiene dispatch HTTP y decode de request; `nueva_app_endpoint_response_v0.go` contiene DTOs/escritura de respuesta; `nueva_app_page_json_v0.go` contiene pagina JSON, textos, opciones y campos derivados. No cambia rutas, shape JSON, codigos, i18n ni el contrato con `SolicitarNuevaAppClientV0`.
Contratos afectados: Ninguno; saneamiento interno de `NuevaAppWebEndpointV0`.
Estado: aceptada localmente.
```

```text
Fecha: 2026-05-04
Decision: WEB-011 divide la zona amarilla web por catalogos, proyecciones y escenarios de test sin cambiar comportamiento.
Motivo: Los ficheros `nueva_app_i18n_v0.go`, `nueva_app_viewmodel_v0.go`, `operational_status_viewmodel_v0.go` y `nueva_app_endpoint_v0_test.go` superaban el umbral amarillo y mezclaban datos estaticos, builders, helpers y escenarios.
Alternativas: Mantener ficheros monoliticos; cambiar nombres publicos; reestructurar contratos JSON.
Impacto: i18n queda separado en shell, required keys y catalogos ES/EN; nueva app separa viewmodel publico, preview de backlog y proyecciones; estado operativo separa tipos/panel, query y helpers; tests endpoint quedan por GET, POST, errores y helpers. No cambia textos, claves, JSON, errores ni expectations.
Contratos afectados: Ninguno; saneamiento interno de `orquesta-web`.
Estado: aceptada localmente.
```

```text
Fecha: 2026-05-04
Decision: WEB-012 crea un cliente local sustituible para `BootstrapProyectoDesdeAppSpec v0` y un viewmodel compacto separado del flujo `nueva_app`.
Motivo: La composicion factory->core->workflow pertenece a `orquesta-director`; web solo debe invocar el puerto publico o un fake testable y proyectar el resultado para operador sin payloads internos.
Alternativas: Duplicar en web la composicion con factory/core/workflow; crear ya un handler HTTP real; ampliar el submit de nueva app.
Impacto: `BootstrapProyectoDesdeAppSpecClientV0` recibe el comando publico del director y devuelve `WebBootstrapProyectoViewModelV0`; el modelo muestra registro, refs opacas, StartRun y eventos compactos, y oculta `Payload`, AppSpec/backlog completos y detalles de adaptadores.
Contratos afectados: Consume `BootstrapProyectoDesdeAppSpecCommandV0`, `BootstrapProyectoDesdeAppSpecResultV0`, `BootstrapProyectoDesdeAppSpecErrorV0`, `OrchestrationCommandV0`, `OrchestrationCommandResultV0`.
Estado: aceptada localmente.
```

```text
Fecha: 2026-05-10
Decision: WEB-014 hace opt-in del arranque de director desde el POST de nueva app mediante `ArrancarDirectorAppClientV0`.
Motivo: El flujo real pedido por el usuario no debe quedarse en validar una spec; la web debe poder pedir una app y dejar que Orquesta arranque el director sin conocer core, scheduler, runtime ni agentes.
Alternativas: Sustituir `SolicitarNuevaAppClientV0` directamente; importar MCP en codigo productivo web; construir puertos de director desde web; seguir solo con preview de backlog.
Impacto: `NuevaAppWebEndpointV0` usa `DirectorClient` si esta configurado y conserva `SolicitarNuevaAppClientV0` como fallback. El resultado expone `director.run_ref`, `director_tasks` y `started_agents` compactos.
Contratos afectados: `ArrancarDirectorAppClientV0`, `WebNuevaAppDirectorV0`, `orquesta.apps.arrancar_director.v0`, bridge REST `/api/v0/apps/director`.
Estado: aceptada localmente.
```

```text
Fecha: 2026-05-10
Decision: WEB-015 acepta HTTP 400 como respuesta publica valida solo para `estado:error` de director stats.
Motivo: El bridge REST de `/api/v0/director/stats` debe distinguir validacion de usuario/run no disponible de fallo tecnico. La web debe renderizar esos errores para operador/director, no ocultarlos como transporte.
Alternativas: Tratar todo 400 como transporte; devolver siempre 200 en MCP HTTP; duplicar validaciones en web.
Impacto: `RESTDirectorStatsClientV0` mantiene 5xx y 400 ambiguo como `error_transporte`, pero proyecta `estado:error` como `WebDirectorStatsViewModelV0` con `errores_publicos`.
Contratos afectados: `DirectorStatsClientV0`, `WebDirectorStatsInboundResultV0`, `/api/v0/director/stats`.
Estado: aceptada localmente.
```

```text
Fecha: 2026-05-10
Decision: WEB-016 proyecta `brainstorms` como actividad inicial del director.
Motivo: `tasks_total` solo mide microtareas del workflow; al arrancar director
puede ser cero aunque el sistema haya solicitado brainstorming y arrancado
agentes. La web no debe inventar tareas para maquillar ese estado.
Alternativas: Meter `director_task` en `Run.Tasks`; crear un contador web local;
ocultar la fase inicial hasta que existan microtareas.
Impacto: `WebDirectorStatsCountsV0` expone `brainstorms` desde el contrato del
nucleo y mantiene `tasks_total` fiel a microtareas reales.
Contratos afectados: `WebDirectorStatsCountsV0`, `WebDirectorRunStatsContractV0`.
Estado: aceptada localmente.
```

```text
Fecha: 2026-05-13
Decision: El panel `director_stats` acepta y proyecta checkpoint pendiente solo
si llega ya saneado por el contrato consumido.
Motivo: `server-shutdown` y MCP ya exponen `checkpoint_agents_pending`,
`pending_checkpoint_agent_refs` y `checkpoint_evidence_refs`, pero la web no
tiene todavia un cliente/handler de shutdown. La UI puede preparar el DTO y el
viewmodel sin leer runtime, ficheros de ACK, RunStore ni server-shutdown.
Alternativas: llamar a `orquesta.server.shutdown.v0` desde `/director-stats`;
crear un endpoint web nuevo de shutdown; ignorar el campo hasta integrar
shutdown.
Impacto: `WebDirectorRunStatsContractV0` acepta los campos opcionales y
`WebDirectorStatsCheckpointV0` los muestra como refs opacas deduplicadas. El
estado del panel pasa a `attention` si hay agentes pendientes de checkpoint.
Contratos afectados: `WebDirectorRunStatsContractV0`,
`WebDirectorStatsViewModelV0`, `WebDirectorStatsCheckpointV0`.
Estado: aceptada localmente.
```

```text
Fecha: 2026-05-11
Decision: El panel de estadisticas proyecta uso agregado sin conocer proveedor.
Motivo: la web debe poder mostrar cuanto ha consumido una app al terminar o en
ejecucion, pero no puede depender de Codex, Claude, Gemini, HOME, OAuth ni APIs
de cuota concretas.
Impacto: `WebDirectorRunStatsContractV0` acepta `usage_summary` del nucleo y
`WebDirectorStatsSummaryV0` expone `usage_agents`, `usage_quota_status`,
`usage_total_tokens`. Coste, proveedor y modelo no forman parte del contrato
web de usage; los datos vienen ya saneados por MCP/director stats.
Contratos afectados: `WebDirectorRunStatsContractV0`,
`WebDirectorStatsSummaryV0`.
Estado: aceptada localmente.
```

```text
Fecha: 2026-05-12
Decision: `/director-stats` prepara refresco semitiempo-real de progreso de agentes reutilizando `DirectorStatsClientV0`.
Motivo: El contrato `/api/v0/director/stats` ya existe; la web puede pedir progreso compacto de agentes y publicar un `refresh.href` sin inventar backend, SSE, DB ni runtime propio.
Alternativas: Crear endpoint nuevo de progreso; leer registry/progress source desde web; esperar a una SPA.
Impacto: El endpoint web activa `include_agent_progress=true` cuando hay `run_ref`, conserva senales compactas por agente (`in_flight`, ticks sin progreso y repeticiones) y devuelve metadatos de polling GET localizados.
Contratos afectados: `DirectorStatsClientV0`, `WebDirectorStatsPageV0`, `WebDirectorStatsAgentV0`.
Estado: aceptada localmente.
```

## Inventario acotado de herencia

```text
Fecha: 2026-05-13
Decision: La cola multiapp se expone en web como panel JSON `/run-queue`
encima del bridge REST `orquesta.run_queue.priority.v0`.
Motivo: la web debe permitir ver ranking y cambiar prioridad, pero no debe leer
RunQueue, RunControl, stores ni scheduler. El contrato MCP/REST ya existe y es
el borde estable para humanos e IA.
Alternativas: leer el store de cola desde web; duplicar ranking en UI; mezclar
cola global dentro de `/director-stats`.
Impacto: `RunQueueWebEndpointV0` soporta GET rank y POST set_priority mediante
`RunQueueClientV0`; el refresco solo aplica a `rank`. El progreso profundo por
run sigue en `/director-stats`.
Contratos afectados: `WebRunQueueQueryV0`, `WebRunQueueViewModelV0`,
`RESTRunQueueClientV0`.
Estado: aceptada localmente.
```

```text
Fecha: 2026-05-13
Decision: El control humano/director de runs se expone en web como POST
`/run-control` encima del bridge REST `orquesta.runs.control.v0`.
Motivo: pausar, reanudar, parar o cancelar una run es una mutacion operativa.
La web debe poder emitirla, pero no debe leer ni mutar stores/runtime por su
cuenta.
Alternativas: botones GET con query params; duplicar RunControl en la web; leer
el store de control desde UI.
Impacto: `RunControlWebEndpointV0` solo acepta POST JSON/form y usa
`RunControlClientV0`. El estado devuelto es el resultado publico de MCP; el
checkpoint/parada cooperativa siguen fuera del panel.
Contratos afectados: `WebRunControlCommandV0`, `WebRunControlViewModelV0`,
`RESTRunControlClientV0`.
Estado: aceptada localmente.
```

```text
Fecha: 2026-05-13
Decision: `/app-change` acepta refs opacas de trabajo externo, pero no importa
ninguna logica de la app propietaria.
Motivo: Orquesta debe poder coordinar trabajos sobre OPES u otra app mediante
contratos, sin convertir orquesta-web en un frontend especifico de ese dominio.
Alternativas: crear un formulario OPES dentro de Orquesta; pasar paths reales;
duplicar contratos externos en la web.
Impacto: `WebAppChangeFormV0` mapea `external_project_ref`,
`external_interface_refs`, `external_work_kind` y `external_work_refs` a
`AppChangeRequestV0.external_work`. El core valida refs compactas y el stack las
mantiene como evidencia para el director.
Contratos afectados: `WebAppChangeFormV0`, `AppChangeRequestV0`,
`orquesta.apps.request_change.v0`.
Estado: aceptada localmente.
```

```text
Fecha: 2026-05-10
Decision: El wizard/formulario web expone `request_kind` y `execution_mode`
como entrada de operador.
Motivo: la web no debe decidir si una peticion es app completa, documentacion,
analisis, deploy o debug; debe capturarlo y delegarlo como contrato a Orquesta.
Alternativas: mantener un formulario unico de nueva app; meter la decision en
objetivo libre; duplicar logica de minimos en JavaScript.
Impacto: `WebNuevaAppFormV0` mapea los campos a AppSpecRequestV0, el HTML
ofrece las opciones canonicas y las ayudas i18n por `title`; el submit sigue
siendo un adaptador fino sin DB, runtime ni reglas de negocio.
Contratos afectados: WebNuevaAppFormV0, NuevaAppHTMLHandlerV0,
AppSpecRequestV0.
Estado: aceptada localmente.
```

```text
Fecha: 2026-05-23
Decision: La web prepara autoprogramacion solo como cliente fino de
`orquesta.autoprogramming.prepare_run.v0`.
Motivo: El operador necesita arrancar un flujo acotado preservando refs opacas
de worktree/rama, pero la web no debe conocer Git, runtime, stores ni proveedor.
Alternativas: llamar a runtime desde web; validar Git/worktree localmente;
esperar al panel completo.
Impacto: `RESTAutoprogrammingPrepareRunClientV0` envia
`worktree_isolated=true`, `worktree_ref` y `branch_ref` al bridge REST y solo
proyecta `run_ref`, refs de workflow, agentes de espera, `continue` y errores
publicos.
Contratos afectados: `WebAutoprogrammingPrepareRunV0`,
`orquesta.autoprogramming.prepare_run.v0`.
Estado: aceptada localmente.
```

```text
Fecha: 2026-05-26
Decision: El panel `/ops` muestra una franja visual de flujo operativo
derivada de contratos publicos ya consumidos.
Motivo: el operador necesita comparar rapidamente cola, ejecucion, atencion y
cierre sin inspeccionar cada tabla ni JSON crudo, pero la web no debe crear una
fuente canonica nueva ni leer stores/runtime.
Alternativas: duplicar una pantalla de director stats; crear endpoint nuevo;
mantener solo KPIs y tablas largas.
Impacto: `opsDashboardHTMLV0` agrega `renderFlowSummary` sobre los datos de
`autoprogramming/status`, `director/stats` y recursos ya cargados por el panel.
No cambia contratos REST ni introduce acoplamiento a DB, runtime o proveedor.
Contratos afectados: `/ops`, `orquesta.autoprogramming.status.v0`,
`director-stats`.
Estado: aceptada localmente.
```

```text
Fecha: 2026-05-26
Decision: El intake conversacional de nueva app se modela primero como sesion
web pura con indice de secciones y `AppSpecRequestV0` parcial.
Motivo: permite sustituir el formulario largo por un flujo incremental sin que
la web arranque agentes, valide negocio de factory ni conozca runtime/stores.
Alternativas: crear ya handler HTML completo; llamar MCP desde web; duplicar
validacion de `AppSpecRequestV0` en el adaptador.
Impacto: `WebNuevaAppIntakeSessionV0` conserva decisiones y preguntas criticas
como ids de campo/seccion, y delega el cierre a `SolicitarNuevaApp v0` o al
arranque de director cuando exista el puerto inyectado.
Contratos afectados: `WebNuevaAppIntakeSessionV0`, `AppSpecRequestV0`,
`SolicitarNuevaApp v0`.
Estado: aceptada localmente.
```

Reutilizable como referencia, no como copia directa:

- `cmd/proyectos_web.go`: estructura de secciones del wizard: briefing, arquitectura, interfaces, datos, plataformas, sistemas operativos, compliance, entrega/calidad.
- `cmd/cliente_servidor_recursos.go`: lista heredada de campos de `apiProyectoFabricarAppRequest`, util para contrastar cobertura contra `AppSpecRequestV0`.
- `i18n/es.json` e `i18n/en.json`: claves `projects.factory.*` y `nueva_app.*` como inventario inicial de vocabulario y textos a migrar por slices.
- `cmd/proyectos_web_test.go`: casos de idioma, render de wizard guiado, preservacion de `lang` y preview como inspiracion de tests web.
- `cmd/api_test.go` y `fabricaapp/service_test.go`: casos de cobertura de tipos, i18n, DB como conector, auth, plataformas y backlog propuesto.

En cuarentena para el primer corte:

- `webHandlerNuevaAppPOST` heredado: transforma form a request antigua y puede invocar fabricacion/materializacion.
- `apiHandlerProyectoFabricarApp`: crea tareas en DB mediante `fabricaapp.Materialize`.
- `apiHandlerProyectoFabricarAppPreview`: devuelve `fabricaapp.BlueprintTask`, no `BacklogInicialPropuestoV0`.
- JavaScript embebido de recomendaciones: contiene reglas de negocio duplicadas.
- Selector obligatorio de proyecto heredado: acopla nueva app a proyecto existente; `SolicitarNuevaApp v0` no define ese prerequisito.
- Imports directos a `db` o `fabricaapp` desde web: solo admisibles en adaptadores concretos si respetan puerto publico; no en componentes/handlers de UI.
