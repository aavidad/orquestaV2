# Tareas locales: orquesta-web

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

## Microtareas iniciales

```text
ID: WEB-000
Objetivo: Congelar alcance inicial del mini-proyecto web y documentar reutilizacion/cuarentena de la herencia de nueva app.
Write-set: docs/tareas.md, docs/decisiones.md, docs/contratos.md, docs/pruebas.md
Simbolo foco: InventarioNuevaAppHerencia
Contrato: SolicitarNuevaApp v0 como unica frontera externa de nueva app.
Validacion: Lectura de AGENTS.md, README.md, docs locales, modulos/CONTRATOS.md y orquesta-factory/docs/contratos.md; busqueda acotada de `/nueva-app`, `fabricar-app` y `SolicitarNuevaApp`.
Bloqueos: Ninguno.
Estado: completada en este arranque documental.
```

```text
ID: WEB-001
Objetivo: Definir el DTO local `WebNuevaAppFormV0` y el mapper hacia `AppSpecRequestV0` sin reglas de negocio.
Write-set: `nueva_app_form_v0.go`, `nueva_app_form_v0_test.go`, docs locales.
Simbolo foco: WebNuevaAppFormV0
Contrato: Consume `AppSpecRequestV0`; expone DTO local de formulario.
Validacion: Test unitario de mapeo con request minima valida, i18n enabled por defecto expresado por factory, DB expresada como necesidad/conector y `source=orquesta-web`.
Bloqueos: Necesita confirmar ubicacion/capa tecnica del modulo web cuando exista codigo.
Estado: completada en corte puro sin HTTP/UI/persistencia. El mapper aplica defaults locales de `schema_version`, `source`, `i18n.default_locale` y arquitectura si faltan; no valida reglas de negocio.
```

```text
ID: WEB-002
Objetivo: Crear cliente/puerto de salida web para invocar `SolicitarNuevaApp v0` con timeout, correlation id y errores publicos.
Write-set: `nueva_app_client_v0.go`, `nueva_app_client_v0_test.go`, docs locales.
Simbolo foco: SolicitarNuevaAppClientV0
Contrato: Consume `SolicitarNuevaApp v0`; errores `app_spec_invalida`, `opcion_incompatible`, `target_no_soportado`, `idioma_invalido`, `conector_requerido_no_disponible`.
Validacion: `go test ./modulos/orquesta-web`; tests con `httptest` verifican metodo, path, JSON, header `X-Correlation-ID`, timeout configurable, envelope REST canonico de `../../CONTRATOS.md`, viewmodel de exito, 400 como viewmodel invalido y 500/transport error como codigo publico estable.
Bloqueos: ninguno; decision del director: puerto local `SolicitarNuevaAppClient` con adaptador REST/API inicial.
Estado: completada en corte cliente REST puro sin pantalla, submit real, servidor, handlers, DB, filesystem, runtime ni MCP.
```

```text
ID: WEB-003
Objetivo: Definir `WebNuevaAppViewModelV0` para renderizar `AppSpecV0` y `BacklogInicialPropuestoV0` como proyecciones compactas.
Write-set: `nueva_app_viewmodel_v0.go`, `nueva_app_viewmodel_v0_test.go`, docs locales.
Simbolo foco: WebNuevaAppViewModelV0
Contrato: Consume `AppSpecV0`, `BacklogInicialPropuestoV0`.
Validacion: Test unitario verifica que oculta dumps internos, muestra defaults, warnings, preguntas abiertas, fases y microtareas con write-set/validacion/bloqueos.
Bloqueos: Ninguno si se limita a campos publicos ya documentados.
Estado: completada en corte puro sin HTTP/UI/persistencia. La proyeccion compacta representa spec valida, estado `requiere_datos` por preguntas abiertas y errores publicos de validacion.
```

```text
ID: WEB-004
Objetivo: Migrar por slice el vocabulario i18n de nueva app sin copiar la pantalla heredada completa.
Write-set: `nueva_app_i18n_v0.go`, `nueva_app_i18n_v0_test.go`, docs locales.
Simbolo foco: NuevaAppI18nCatalog
Contrato: Textos de UI locales; todo texto de usuario debe tener locale.
Validacion: Test de claves requeridas para ES/EN y fallback controlado; no usar literals sueltos en vistas.
Bloqueos: Definir estructura i18n del modulo web.
Estado: completada en corte puro sin UI real, templates, HTTP, DB, runtime ni filesystem productivo. Define catalogos minimos `es-ES` y `en-US`, lookup en memoria, fallback a `es-ES`, validacion de claves requeridas y error publico para clave inexistente.
```

```text
ID: WEB-005
Objetivo: Implementar primer render de pantalla nueva app con datos estaticos de contrato/fake, sin submit real y sin copiar UI heredada.
Write-set: `nueva_app_endpoint_v0.go`, `nueva_app_endpoint_v0_test.go`, docs locales.
Simbolo foco: NuevaAppWebEndpointV0
Contrato: Usa `WebNuevaAppFormV0`; no llama DB, runtime ni servicios externos en render.
Validacion: Smoke de render con `httptest`: GET devuelve estado inicial localizado, acciones, opciones de locale y campos derivados de `WebNuevaAppFormV0`; fake confirma cero llamadas al cliente.
Bloqueos: Depende de WEB-001 y WEB-004.
Estado: completada como respuesta JSON estructurada, sin templates, servidor real, DB, runtime, filesystem productivo ni MCP.
```

```text
ID: WEB-006
Objetivo: Conectar submit de nueva app al cliente `SolicitarNuevaApp v0` y renderizar respuesta sin persistir.
Write-set: `nueva_app_endpoint_v0.go`, `nueva_app_endpoint_v0_test.go`, docs locales.
Simbolo foco: NuevaAppWebEndpointV0
Contrato: Consume `SolicitarNuevaApp v0`; expone estado web local `solicitada|requiere_datos|invalida|error_transporte`.
Validacion: Tests con fake de `SolicitarNuevaAppClientV0`: POST JSON y form-urlencoded delegan al cliente, preservan locale/form, renderizan `WebNuevaAppViewModelV0`, error de transporte localizado y metodo no soportado sin delegar.
Bloqueos: Depende de WEB-002.
Estado: completada sin persistir, crear tareas, materializar backlog, arrancar runtime ni llamar DB.
```

```text
ID: WEB-006A
Objetivo: Ejecutar una prueba vertical web -> REST -> factory sin UI real, submit handler ni servidor web propio.
Write-set: `nueva_app_rest_flow_v0_test.go`, docs locales.
Simbolo foco: RESTSolicitarNuevaAppClientV0 + NewAppSpecHTTPHandlerV0
Contrato: Usa el cliente REST existente de web contra el handler publico REST de factory en `httptest.NewServer`.
Validacion: `go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-factory`; valida request correcta a viewmodel `valida`, schema, fases, microtareas, correlation/request id, formulario invalido a viewmodel `invalida` sin backlog inventado y flujo sin proyecto existente ni persistencia.
Bloqueos: Ninguno.
Estado: completada en corte de integracion sin UI real, handler de submit, DB, filesystem, runtime, MCP ni servidor web propio.
```

```text
ID: WEB-007
Objetivo: Renderizar preview compacto de fases y microtareas propuestas desde `BacklogInicialPropuestoV0`.
Write-set: `nueva_app_viewmodel_v0.go`, `nueva_app_endpoint_v0.go`, tests y docs locales.
Simbolo foco: NuevaAppBacklogPreview
Contrato: Consume `BacklogInicialPropuestoV0`.
Validacion: Test de accesibilidad/semantica basica y snapshot controlado: key, fase, titulo, modulo/frontera, write-set previsto, bloqueo y criterio de cierre; no mostrar structs internos.
Bloqueos: Depende de WEB-003.
Estado: completada como proyeccion JSON compacta en `WebNuevaAppViewModelV0` y textos i18n de handler, sin frontend pesado, servidor real, DB, runtime, filesystem productivo ni MCP.
```

```text
ID: WEB-008
Objetivo: Documentar y probar la cuarentena de la herencia para evitar regresion hacia fabricacion directa.
Write-set: `nueva_app_architecture_test.go`, docs locales.
Simbolo foco: NoDBNoRuntimeGuard
Contrato: Regla local de adaptador fino.
Validacion: Test o regla de arquitectura que falle si el flujo nueva app importa DB directa, runtime, materializacion de backlog o endpoints antiguos `fabricar-app`.
Bloqueos: Depende de que exista arbol de codigo web.
Estado: completada con test de arquitectura sobre ficheros productivos `nueva_app_*.go`; permite campos contractuales como `db_required`, pero bloquea imports directos a DB/runtime/cmd/herencia y literales de fabricacion/materializacion.
```

```text
ID: WEB-009
Objetivo: Crear proyeccion compacta de estado/fase de orquestacion para web usando contratos publicos de observability.
Write-set: `operational_status_viewmodel_v0.go`, `operational_status_viewmodel_v0_test.go`, docs locales.
Simbolo foco: WebOperationalStatusPanelV0
Contrato: Consume `OperationalStatusQueryV0` y `DiagnosticoCompactoV0` de `orquesta-observability`.
Validacion: `go test -count=1 ./modulos/orquesta-web`; valida query read-only con consumidor `orquesta-web/web`, secciones compactas y panel con estado, fase actual, progreso, bloqueos, salud, actividad, frescura y privacidad sin dumps internos.
Bloqueos: No implementa cliente, endpoint, servidor real, DB, runtime, filesystem productivo ni acciones de recuperacion.
Estado: completada en corte puro de view-model/builder local; la consulta real queda para un puerto/adaptador posterior.
```

```text
ID: WEB-010
Objetivo: Sanear tamano de `nueva_app_endpoint_v0.go` separando responsabilidades sin cambiar comportamiento.
Write-set: `nueva_app_endpoint_v0.go`, `nueva_app_endpoint_handler_v0.go`, `nueva_app_endpoint_response_v0.go`, `nueva_app_page_json_v0.go`, docs locales.
Simbolo foco: NuevaAppWebEndpointV0
Contrato: Mantiene rutas, shape JSON, codigos de error, i18n y contrato con `SolicitarNuevaAppClientV0`.
Validacion: `gofmt`; `go test -count=1 ./modulos/orquesta-web`; `go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-factory ./modulos/orquesta-observability`; `git diff --check -- modulos/orquesta-web`; `wc -l` de Go tocados.
Bloqueos: Ninguno.
Estado: completada como refactor mecanico; endpoint queda como identidad/configuracion, handler HTTP, DTO/escritura de respuesta y pagina JSON/textos en ficheros separados, todos por debajo de 300 lineas.
```

```text
ID: WEB-011
Objetivo: Sanear zona amarilla de web dividiendo i18n, viewmodels, estado operativo y tests endpoint sin cambiar comportamiento ni contratos REST/JSON.
Write-set: `nueva_app_i18n_v0.go`, `nueva_app_i18n_keys_v0.go`, `nueva_app_i18n_es_v0.go`, `nueva_app_i18n_en_v0.go`, `nueva_app_viewmodel_v0.go`, `nueva_app_backlog_preview_v0.go`, `nueva_app_viewmodel_projection_v0.go`, `operational_status_viewmodel_v0.go`, `operational_status_query_v0.go`, `operational_status_panel_helpers_v0.go`, `nueva_app_endpoint_v0_test.go`, `nueva_app_endpoint_post_v0_test.go`, `nueva_app_endpoint_errors_v0_test.go`, `nueva_app_endpoint_test_helpers_v0_test.go`, docs locales.
Simbolo foco: NuevaAppI18nCatalogV0, WebNuevaAppViewModelV0, WebOperationalStatusPanelV0, NuevaAppWebEndpointV0 tests.
Contrato: Mantiene textos i18n, claves, JSON, errores, snapshots y expectations existentes; no cambia contratos REST/JSON.
Validacion: `gofmt`; `go test -count=1 ./modulos/orquesta-web`; `git diff --check -- modulos/orquesta-web`; `wc -l` de Go tocados.
Bloqueos: Ninguno.
Estado: completada como split mecanico por catalogos i18n, preview/backlog, proyecciones, query/panel operativo y tests por escenario; todos los Go tocados quedan por debajo de 300 lineas.
```

```text
ID: WEB-012
Objetivo: Preparar adaptador fino web para `BootstrapProyectoDesdeAppSpec v0` sin UI pesada ni servidor real.
Write-set: `bootstrap_client_v0.go`, `bootstrap_viewmodel_v0.go`, `bootstrap_client_v0_test.go`, `bootstrap_viewmodel_v0_test.go`, docs locales.
Simbolo foco: BootstrapProyectoDesdeAppSpecClientV0, WebBootstrapProyectoViewModelV0
Contrato: Consume DTOs publicos de `orquesta-director` y tipos publicos de resultado de `orquesta-core-workflow`; no duplica composicion factory->core->workflow.
Validacion: `gofmt`; `go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-director`; `git diff --check -- modulos/orquesta-web`; `wc -l` de Go tocados.
Bloqueos: No existe adaptador inbound real del director; el cliente local delega en el puerto publico puro del director y queda sustituible.
Estado: completada como slice separado sin tocar flujo nueva app, DB, runtime, filesystem productivo, MCP, CLI, deploy, HTTP real ni servidor.
```

```text
ID: WEB-020
Objetivo: Permitir que `/app-change` envie refs opacas de trabajo externo sin
acoplar la web a OPES ni a otra app propietaria.
Write-set: app_change_*_v0.go, tests y docs locales.
Simbolo foco: WebAppChangeFormV0.external_work
Contrato: `AppChangeRequestV0.external_work` consume project/interface/work
refs compactas.
Validacion: `go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-mcp ./modulos/orquesta-app-codex-stack`.
Bloqueos: El wizard conversacional futuro debe construir estos campos por
puerto de intake/director, no por reglas de dominio en la web.
Estado: completada.
```

```text
ID: WEB-013
Objetivo: Sustituir el formulario largo de nueva app por un flujo conversacional guiado por agente de intake, sin perder `AppSpecV0` como contrato canonico.
Write-set: docs/tareas.md, docs/contratos.md, nueva_app_intake_session_v0.go, nueva_app_intake_session_v0_test.go
Simbolo foco: WebNuevaAppIntakeSessionV0
Contrato: `IntakeSession v0` consume/produce `AppSpecRequestV0` parcial y delega el cierre en `SolicitarNuevaApp v0`.
Validacion: tests puros de sesion inicial desde nombre/idea, pregunta pendiente, decision capturada, AppSpec parcial y estado `requiere_datos`; no DB, runtime, filesystem productivo, LLM real ni MCP directo.
Bloqueos: Necesita contrato equivalente en API/MCP para arrancar agente de intake real; en este corte solo se modela estado web y puerto.
Estado: pendiente.
```

```text
ID: WEB-014
Objetivo: Conectar el POST de nueva app al arranque de director cuando exista puerto configurado, manteniendo `SolicitarNuevaApp` como fallback.
Write-set: nueva_app_director_*_v0.go, nueva_app_endpoint_*_v0.go, i18n, tests y docs locales.
Simbolo foco: ArrancarDirectorAppClientV0
Contrato: Consume bridge REST de `orquesta.apps.arrancar_director.v0` en `/api/v0/apps/director`.
Validacion: `go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-mcp`.
Bloqueos: El servidor productivo debe inyectar el executor real con puertos de runtime/stores; web no lo construye.
Estado: completada como slice de adaptador fino: cliente REST, viewmodel `director`, endpoint opt-in y prueba vertical web -> REST -> MCP con puertos fake.
```

```text
ID: WEB-015
Objetivo: Alinear cliente web de estadisticas con el bridge REST de director stats.
Write-set: director_stats_client_v0.go, director_stats_client_v0_test.go, docs locales.
Simbolo foco: RESTDirectorStatsClientV0
Contrato: `estado:error` de `/api/v0/director/stats` es error publico renderizable aunque llegue con HTTP 400.
Validacion: `go test -count=1 ./modulos/orquesta-web`.
Bloqueos: Depende de bridge REST MCP equivalente; no construye RunStore, registry ni progress source.
Estado: completada. El cliente solo trata 400 como respuesta valida si el envelope declara `estado:error`; 5xx y 400 ambiguo siguen siendo transporte.
```

```text
ID: WEB-016
Objetivo: Mostrar actividad inicial del director sin convertir `director_task`
en microtarea.
Write-set: director_stats_viewmodel_v0.go, tests y docs locales.
Simbolo foco: WebDirectorStatsCountsV0.Brainstorms
Contrato: `counts.brainstorms` viene de `DirectorRunStatsV0`; `tasks_total`
permanece reservado a microtareas aceptadas por workflow.
Validacion: `go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-app-gateway`.
Bloqueos: Si en el futuro se requiere "directores preparados", debe modelarse
como proyeccion canonica en nucleo, no como contador inventado por web.
Estado: completada.
```

```text
ID: WEB-018
Objetivo: Preparar `/director-stats` para refresco semitiempo-real de progreso y estado de agentes usando el endpoint existente.
Write-set: director_stats_*_v0.go, tests y docs locales.
Simbolo foco: WebDirectorStatsPageV0.Refresh
Contrato: Consume `DirectorStatsClientV0` contra `/api/v0/director/stats`; el polling vuelve a `/director-stats` con `include_agent_progress=true`.
Validacion: `go test -count=1 ./modulos/orquesta-web`.
Bloqueos: El progreso real solo aparecera si el gateway/MCP inyecta `AgentProgressObservationProviderPortV0`; sin ese puerto la web muestra `source_status` degradado/no configurado.
Estado: completada como preparacion UI semitiempo-real sobre contrato existente.
```

```text
ID: WEB-019
Objetivo: Preparar visibilidad web de checkpoint pendiente por run sin acoplar
la web a shutdown/runtime.
Write-set: director_stats_*_v0.go, tests y docs locales.
Simbolo foco: WebDirectorStatsCheckpointV0
Contrato: si el contrato consumido entrega `checkpoint_agents_pending`,
`pending_checkpoint_agent_refs` y `checkpoint_evidence_refs`, la web los
proyecta como refs opacas y marca atencion.
Validacion: `go test -count=1 ./modulos/orquesta-web`.
Bloqueos: `/director-stats` no consume todavia `orquesta.server.shutdown.v0`;
la integracion productiva debe llegar por un handler/API que entregue esos
campos saneados, no por acceso directo a stores/runtime desde web.
Estado: completada como DTO/proyeccion preparada.
```

## CONSULTA AL DIRECTOR

```text
Modulo origen: orquesta-web
Modulo afectado: orquesta-factory
Bloqueo: El contrato publico define `SolicitarNuevaApp v0`, pero no fija el transporte inicial que debe usar la web ni el nombre de endpoint/tool/conector.
Pregunta concreta: Para WEB-002/WEB-006, ¿la web debe invocar `SolicitarNuevaApp v0` por HTTP interno, llamada en proceso a puerto de factory, MCP tool o un conector todavia pendiente?
Opcion recomendada: Definir un puerto de salida web `SolicitarNuevaAppClient` y un adaptador HTTP/API inicial versionado, manteniendo la UI desacoplada del transporte.
Impacto: Sin esta decision se pueden escribir DTOs y view-models, pero no cerrar la integracion real del submit.
Decision del director: usar puerto local `SolicitarNuevaAppClient`; primer conector REST/API versionado, endpoint canonico `POST /api/v0/apps/spec` segun `../../CONTRATOS.md`. Consulta cerrada.
```

```text
Modulo origen: orquesta-web
Modulo afectado: orquesta-core / orquesta-factory
Bloqueo: La herencia exige seleccionar proyecto y puede crear tareas; `SolicitarNuevaApp v0` solo valida/normaliza/propone y no persiste.
Pregunta concreta: ¿El primer flujo web debe pedir una app independiente sin proyecto existente, o debe asociar la solicitud a un proyecto ya registrado por un contrato core aun pendiente?
Opcion recomendada: En v0 pedir app independiente y mostrar `AppSpecV0` + backlog propuesto; dejar registro de proyecto/backlog para un contrato posterior de core.
Impacto: Define si el formulario incluye selector de proyecto o solo correlation/request id en el primer corte.
Decision del director: en v0 se pide una app independiente sin proyecto existente. El registro de proyecto/backlog queda para `RegistrarProyectoDesdeAppSpec v0`. Consulta cerrada.
```
