# Pruebas locales: orquesta-web

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
Caso: WEB-CT-001 mapper form minimo a AppSpecRequestV0
Tipo: contract
Comando: `go test ./modulos/orquesta-web`
Evidencia esperada: `schema_version=app_spec_request.v0`, `source=orquesta-web`, `request_id`, `locale`, `nombre`, `objetivo` y `tipo_app` presentes; sin DB directa ni runtime.
Ultima ejecucion: 2026-05-04; pasa.
Riesgos: Divergencia con schemas futuros de factory si cambian campos obligatorios.
```

```text
Caso: WEB-CT-002 errores publicos de SolicitarNuevaApp v0
Tipo: contract
Comando: `go test ./modulos/orquesta-web`
Evidencia esperada: 400 REST con errores publicos se convierte en `WebNuevaAppViewModelV0` estado `invalida`, preserva `request_id` y `locale`, no inventa backlog y no devuelve error de transporte.
Ultima ejecucion: 2026-05-04; pasa.
Riesgos: Que el transporte real envuelva errores sin codigo publico estable.
```

```text
Caso: WEB-CT-003 cliente REST SolicitarNuevaApp v0
Tipo: contract
Comando: `go test ./modulos/orquesta-web`
Evidencia esperada: `httptest` verifica `POST /api/v0/apps/spec`, JSON `AppSpecRequestV0`, header `X-Correlation-ID`, timeout configurable, creacion de `request_id` si falta, respuesta 2xx a viewmodel con spec/backlog y 500/transport timeout como `error_transporte`.
Ultima ejecucion: 2026-05-04; pasa.
Riesgos: El envelope REST canonico ya esta fijado en `../../CONTRATOS.md` como 2xx `app_spec` + `backlog` y 400 `errores`; los alias `spec` y `backlog_inicial_propuesto` solo cubren compatibilidad transitoria de arranque.
```

```text
Caso: WEB-UT-001 ViewModel compacto de AppSpecV0 y BacklogInicialPropuestoV0
Tipo: unit
Comando: `go test ./modulos/orquesta-web`
Evidencia esperada: Render model contiene resumen, defaults, warnings, preguntas abiertas, fases y microtareas; no contiene transcript, structs privados, tablas SQL ni detalles de proveedor.
Ultima ejecucion: 2026-05-04; pasa.
Riesgos: Sobreexponer `AppSpecV0` completo por comodidad de render.
```

```text
Caso: WEB-UT-003 mapper form rico preserva campos publicos
Tipo: unit
Comando: `go test ./modulos/orquesta-web`
Evidencia esperada: El mapper preserva i18n, documentacion, datos, deploy, calidad, agentes, conectores, restricciones, plataformas y preferencias sin invocar validacion de negocio.
Ultima ejecucion: 2026-05-04; pasa.
Riesgos: Si factory renombra campos publicos, el mapper debe actualizarse en el mismo contrato versionado.
```

```text
Caso: WEB-UT-004 errores publicos en viewmodel
Tipo: unit
Comando: `go test ./modulos/orquesta-web`
Evidencia esperada: Errores `ValidationIssue` publicos se proyectan como `errores_publicos`, preservando `request_id` y `locale`, sin inventar fases ni microtareas.
Ultima ejecucion: 2026-05-04; pasa.
Riesgos: Los errores de transporte quedan para WEB-002/WEB-006; este corte solo cubre errores publicos de validacion.
```

```text
Caso: WEB-UT-002 i18n de nueva app
Tipo: unit
Comando: `go test -count=1 ./modulos/orquesta-web`
Evidencia esperada: ES/EN cubren secciones principales, errores publicos, labels de campos y estados; fallback controlado para locale no soportado.
Ultima ejecucion: 2026-05-04; pasa.
Riesgos: El catalogo es minimo para la futura pantalla; al ampliar UI se deben anadir claves antes de renderizar texto nuevo.
```

```text
Caso: WEB-INT-001 GET nueva app no toca DB ni runtime
Tipo: integration
Comando: `go test -count=1 ./modulos/orquesta-web`
Evidencia esperada: `TestNuevaAppWebEndpointV0GETRenderInicialLocalizadoYOpcionesDelForm` valida render inicial localizado, formulario/proyeccion vacia, opciones de locale y campos derivados de `WebNuevaAppFormV0`; fake confirma cero llamadas al cliente.
Ultima ejecucion: 2026-05-04; pasa.
Riesgos: Reintroducir selector de proyecto heredado que fuerce DB.
```

```text
Caso: WEB-INT-002 POST nueva app usa SolicitarNuevaApp v0
Tipo: integration
Comando: `go test -count=1 ./modulos/orquesta-web`
Evidencia esperada: `TestNuevaAppWebEndpointV0POSTJSONDelegaAlClienteYRenderizaViewModel`, `TestNuevaAppWebEndpointV0POSTFormURLEncodedDelegaSinTemplates`, `TestNuevaAppWebEndpointV0POSTErrorClienteDevuelveErrorPublicoLocalizado` y `TestNuevaAppWebEndpointV0MetodoNoSoportadoNoDelega` validan delegacion al puerto `SolicitarNuevaAppClientV0`, render estable, locale preservado, errores publicos localizados y cero materializacion de backlog/DB/runtime.
Ultima ejecucion: 2026-05-04; pasa.
Riesgos: Confundir preview de backlog propuesto con fabricacion real.
```

```text
Caso: WEB-UT-007 preview compacto de BacklogInicialPropuestoV0
Tipo: unit
Comando: `go test -count=1 ./modulos/orquesta-web`
Evidencia esperada: `TestWebNuevaAppBacklogPreviewV0CompactaFasesYMicrotareas` valida snapshot controlado con key, fase, titulo, modulo/frontera, write-set previsto, bloqueo y criterio de cierre; no expone structs internos ni campos del DTO de factory como dump.
Ultima ejecucion: 2026-05-04; pasa.
Riesgos: Mantener compatibilidad con la proyeccion previa `fases`/`microtareas` hasta que exista UI final.
```

```text
Caso: WEB-INT-007 handler JSON renderiza backlog_preview
Tipo: integration
Comando: `go test -count=1 ./modulos/orquesta-web`
Evidencia esperada: `TestNuevaAppWebEndpointV0POSTJSONRenderizaBacklogPreviewCompacto` valida con `httptest` que el JSON de pagina incluye textos i18n del preview y semantica basica para fases/microtareas sin servidor real ni frontend pesado.
Ultima ejecucion: 2026-05-04; pasa.
Riesgos: La accesibilidad queda limitada a estructura JSON/i18n; una UI HTML futura necesitara pruebas propias.
```

```text
Caso: WEB-INT-006A flujo vertical REST nueva app
Tipo: integration
Comando: `go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-factory ./modulos/orquesta-factory-http`
Evidencia esperada: `RESTSolicitarNuevaAppClientV0` consume `orquestafactoryhttp.NewAppSpecHTTPHandlerV0` con `httptest.NewServer`; request valida desde `WebNuevaAppFormV0` devuelve `WebNuevaAppViewModelV0` estado `valida`, resumen, fases, microtareas, schema y correlation/request id coherentes; request invalida devuelve estado `invalida`, errores publicos y sin backlog inventado.
Ultima ejecucion: 2026-05-23; pasa.
Riesgos: No cubre UI real ni handler de submit; eso queda para WEB-006.
```

```text
Caso: WEB-CT-014 cliente REST ArrancarDirectorApp v0
Tipo: contract
Comando: `go test -count=1 ./modulos/orquesta-web`
Evidencia esperada: `RESTArrancarDirectorAppClientV0` envia `POST /api/v0/apps/director` con `app_spec_request`, correlation header y limites opcionales; respuesta `ok` con `run_ref` se proyecta a `WebNuevaAppViewModelV0` estado `director_arrancado`, y respuesta `error` se mantiene como error publico renderizable incluso si llega con status 5xx estructurado por backend Goal degradado.
Ultima ejecucion: 2026-05-10; pasa.
Riesgos: El bridge REST productivo debe inyectarse fuera de web; web no construye puertos internos.
```

```text
Caso: WEB-INT-014 POST nueva app arranca director por puerto opt-in
Tipo: integration
Comando: `go test -count=1 ./modulos/orquesta-web`
Evidencia esperada: `NuevaAppWebEndpointV0` usa `DirectorClient` si esta configurado, no llama `SolicitarNuevaAppClientV0`, transporta `director_execution_mode` al bridge REST y renderiza `run_ref`, datos goal-first o datos legacy (`director_tasks`/`started_agents`) y texto i18n `director_arrancado`.
Ultima ejecucion: 2026-05-10; pasa.
Riesgos: No sustituye aun la UI conversacional; solo cambia el destino del submit cuando el puerto existe.
```

```text
Caso: WEB-INT-014A flujo vertical web -> REST -> MCP -> director
Tipo: integration
Comando: `go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-mcp`
Evidencia esperada: Cliente web llama al bridge REST de `orquesta.apps.arrancar_director.v0`; el handler MCP invoca `StartAppDirectorV0`. Cuando se prueba la rama legacy debe enviarse `director_execution_mode=legacy_director_loop` y entonces devuelve `run_ref` y agente director arrancado; sin modo explicito, el servicio aplica `goal_first`.
Ultima ejecucion: 2026-05-10; pasa.
Riesgos: Usa fake lifecycle launcher; la prueba real con Codex queda en `orquesta-runtime-codex-delivery`.
```

```text
Caso: WEB-UT-015 director stats 400 publico
Tipo: unit
Comando: `go test -count=1 ./modulos/orquesta-web`
Evidencia esperada: `RESTDirectorStatsClientV0` renderiza `estado:error` con HTTP 400 como panel de errores publicos y conserva 400 ambiguo como `error_transporte`.
Ultima ejecucion: 2026-05-10; pasa.
Riesgos: No arranca RunStore ni registry real; solo alinea semantica cliente/bridge REST.
```

```text
Caso: WEB-INT-015A flujo vertical web -> REST -> MCP director stats
Tipo: integration
Comando: `go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-mcp`
Evidencia esperada: `RESTDirectorStatsClientV0` consume `NewMCPDirectorStatsHTTPHandlerV0`, el executor MCP lee un RunStore in-memory y la web proyecta run, fase, tareas, agentes y estado de fuente de progreso.
Ultima ejecucion: 2026-05-10; pasa.
Riesgos: No configura registry/progress source productivos; verifica el contrato de transporte y proyeccion.
```

```text
Caso: WEB-UT-016 director stats expone brainstorming inicial
Tipo: unit/integration
Comando: `go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-app-gateway`
Evidencia esperada: la web proyecta `counts.brainstorms` desde el nucleo y el
gateway verifica que arrancar director deja `tasks_total=0` pero
`brainstorms=1` y `agents_started=1`.
Ultima ejecucion: 2026-05-10; pasa.
Riesgos: No modela contador separado de `director_task`; mantiene la frontera
del workflow.
```

```text
Caso: WEB-CT-023 cliente REST autoprogramming prepare-run
Tipo: contract
Comando: `go test -count=1 ./modulos/orquesta-web`
Evidencia esperada: `RESTAutoprogrammingPrepareRunClientV0` envia
`POST /api/v0/autoprogramming/prepare-run`, correlation header,
`worktree_isolated=true`, `worktree_ref`, `branch_ref`, write-set y tests
obligatorios; `priority_score` cruza como dato de cola opcional; la respuesta
proyecta run, workflow tasks, agentes de espera, `goal_specs` opcional,
continue y errores publicos sin leer runtime/stores.
Ultima ejecucion: 2026-05-23; pasa.
Riesgos: El executor real de prepare-run se inyecta fuera de web; este corte
solo cubre cliente, DTO y viewmodel.
```

```text
Caso: WEB-ARCH-001 guard contra DB/runtime/fabricar-app en flujo nueva app
Tipo: smoke
Comando: `go test -count=1 ./modulos/orquesta-web`
Evidencia esperada: `TestNuevaAppNoDBNoRuntimeGuardV0` parsea ficheros productivos `nueva_app_*.go`, falla con imports directos a DB/runtime/cmd/herencia y con literales de `fabricar-app`, materializacion de backlog o referencias obligatorias a proveedor runtime/agente.
Ultima ejecucion: 2026-05-04; pasa.
Riesgos: El guard permite campos contractuales como `db_required` porque expresan necesidad funcional, no acceso directo a DB.
```

```text
Caso: WEB-UT-009 panel compacto de estado operativo
Tipo: unit
Comando: `go test -count=1 ./modulos/orquesta-web`
Evidencia esperada: `TestWebOperationalStatusQueryV0UsaContratoPublicoReadOnly`, `TestWebOperationalStatusPanelV0CompactaDiagnosticoSinDumpInterno` y `TestWebOperationalStatusPanelV0EstadoDesconocidoNoInventaFases` validan query `OperationalStatusQueryV0` para `orquesta-web/web`, proyeccion compacta de `DiagnosticoCompactoV0` con fase/progreso/bloqueos/salud/frescura y degradacion a `unknown` sin dumps internos.
Ultima ejecucion: 2026-05-04; pasa.
Riesgos: Todavia no existe cliente ni endpoint real para consultar observability; este corte solo fija el shape web puro.
```

```text
Caso: WEB-SMOKE-010 saneamiento de tamano de endpoint nueva app
Tipo: smoke
Comando: `gofmt` sobre Go tocados; `go test -count=1 ./modulos/orquesta-web`; `go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-factory ./modulos/orquesta-observability`; `git diff --check -- modulos/orquesta-web`; `wc -l` de Go tocados.
Evidencia esperada: El endpoint mantiene los tests existentes de GET/POST/error/i18n/backlog preview, no cambia shape JSON ni codigos HTTP, y los Go tocados quedan por debajo de 300 lineas.
Ultima ejecucion: 2026-05-04; pasa.
Riesgos: Refactor mecanico; cualquier cambio futuro en pagina JSON debe cuidar que request/response/handler sigan separados.
```

```text
Caso: WEB-SMOKE-011 saneamiento zona amarilla web
Tipo: smoke
Comando: `gofmt` sobre Go tocados; `go test -count=1 ./modulos/orquesta-web`; `git diff --check -- modulos/orquesta-web`; `wc -l` de Go tocados.
Evidencia esperada: i18n mantiene claves/textos ES/EN y fallback; viewmodel nueva app mantiene shape JSON y preview; panel operativo mantiene query/proyeccion compacta; tests endpoint conservan expectations por escenario; todos los Go tocados quedan por debajo de 300 lineas.
Ultima ejecucion: 2026-05-04; pasa.
Riesgos: Refactor mecanico; no anade cobertura nueva ni cambia contratos REST/JSON.
```

```text
Caso: WEB-INT-013 primera UI HTML nueva app
Tipo: integration
Comando: `go test -count=1 ./modulos/orquesta-web`
Evidencia esperada: `TestNuevaAppHTMLHandlerV0GETMuestraFormularioUsableSinDelegar`, `TestNuevaAppHTMLHandlerV0POSTValidoDelegaYRenderizaResultado` y `TestNuevaAppHTMLHandlerV0POSTInvalidoRenderizaErrorPublico` validan GET HTML en `/nueva-app`, POST form-urlencoded contra fake `SolicitarNuevaAppClientV0`, render de estado/resumen/backlog y error publico localizado, wizard con roles tab y teclado flechas/Home/End, arquitectura opcional con `sin_preferencia` y fallback posterior `hexagonal`, catalogo experto de storage alineado con factory, sin DB, runtime, provider/model, HOME ni OAuth.
Ultima ejecucion: 2026-05-06; pasa.
Riesgos: UI v0 server-rendered y deliberadamente basica; no preserva todos los valores del formulario tras submit ni incluye interacciones cliente.
```

```text
Caso: WEB-DOC-003 guia de opciones `/nueva-app`
Tipo: docs
Comando: `rg -n "Modo Experto|clean_architecture|modular_monolith|data_pipeline|calidad.accesibilidad|Preferencia De Persistencia|Varias Integraciones|Tooltips" modulos/orquesta-web/docs/guia_nueva_app_opciones_2026-06-25.md`
Evidencia esperada: la guia documenta modo basico, asistente guiado y modo
experto, catalogo ampliado de arquitectura con fallback `hexagonal`, datos
multiples, sensibilidad por tipo, almacenamiento por capacidad, varias
integraciones, niveles de accesibilidad elegibles por runtime/adaptador y
relacion con tooltips, sin fijar proveedor concreto; `/nueva-app` enlaza a
`/nueva-app/guia` y `NuevaAppGuideWebEndpointV0` sirve el Markdown embebido como
HTML de lectura.
Ultima ejecucion: 2026-06-25; documental y cubierta por tests web/factory.
Riesgos: las composiciones externas deben seguir tratando arquitectura, storage,
mapas y accesibilidad como contratos/capacidades, no como proveedor impuesto.
```

```text
Caso: WEB-UT-012 bootstrap director para web
Tipo: unit
Comando: `go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-director`
Evidencia esperada: `LocalBootstrapProyectoDesdeAppSpecClientV0` delega en un puerto publico del director o fake inyectado; errores publicos `director_bootstrap_invalido` se proyectan como viewmodel invalido; errores no publicos quedan con codigo estable de cliente; `WebBootstrapProyectoViewModelV0` muestra refs, registro, StartRun y eventos compactos sin payload ni entrada completa.
Ultima ejecucion: 2026-05-04; pasa.
Riesgos: El transporte inbound real del director aun no existe; si se define REST/MCP posterior, debe reemplazar el cliente local sin cambiar el viewmodel.
```

```text
Caso: WEB-UT-013 sesion conversacional nueva app
Tipo: unit
Comando: `GOCACHE=/tmp/orquesta-go-cache go test -count=1 ./modulos/orquesta-web -run 'TestWebNuevaAppIntakeSessionV0|TestApplyWebNuevaAppIntake|TestWebNuevaAppIntakeGuided|TestNuevaAppIntakeGuided'`
Evidencia esperada: `TestWebNuevaAppIntakeSessionV0CreaSesionDesdeIdeaYPideCamposCriticos`,
`TestApplyWebNuevaAppIntakeAnswerV0RegistraDecisionYDejaDraftListo`,
`TestApplyWebNuevaAppIntakeAnswerV0NoValidaEnumsDeFactory` y
`TestWebNuevaAppIntakeSessionV0SnapshotCompactoSinDumpsInternos` validan sesion
desde idea/nombre, pregunta pendiente, indice de campos, decision capturada,
claves i18n de pregunta, `AppSpecRequestV0` parcial listo para validar, handoff
compacto con refs opacas hacia Director/fallback, snapshot compacto y ausencia
de validacion de enums de factory. `TestWebNuevaAppIntakeGuided*` valida que
una necesidad libre de app movil para alquileres cercanos produce decisiones de
plataforma, datos, storage, mapas, arquitectura y accesibilidad compatibles con
factory. `TestNuevaAppIntakeGuidedHTTPHandler*` valida el endpoint JSON
`POST /api/v0/apps/intake/guided-turn`, acciones de seguimiento y errores HTTP,
sin DB, runtime, filesystem productivo, LLM real ni MCP directo.
Ultima ejecucion: 2026-06-25; pasa con `GOCACHE=/tmp/orquesta-go-cache`.
Riesgos: No arranca agente de intake real; el endpoint es calculo puro de
wizard y el handoff solo declara contrato compacto. La disponibilidad de la
ruta depende de la composicion/gateway.
```

## Validaciones realizadas en este arranque

```text
Caso: WEB-DOC-001 lectura de contexto obligatorio
Tipo: smoke
Comando: `sed -n` sobre AGENTS.md, README.md, docs locales, `modulos/CONTRATOS.md` y `orquesta-factory/docs/contratos.md`.
Evidencia esperada: Contexto leido y decisiones documentadas en docs locales.
Ultima ejecucion: 2026-05-04.
Riesgos: No valida comportamiento de codigo; solo reduce riesgo de arrancar contra contrato incorrecto.
```

```text
Caso: WEB-DOC-002 inventario acotado de herencia
Tipo: smoke
Comando: `rg -n "SolicitarNuevaApp|NuevaApp|nueva app|fabricar app|fabricar|solicitar.*app|app nueva|request.*app|create.*app|new app"` y lecturas acotadas de `cmd/proyectos_web.go`, `cmd/cliente_servidor_recursos.go`, `cmd/api.go`, tests e i18n.
Evidencia esperada: Reutilizacion/cuarentena registrada sin copiar codigo.
Ultima ejecucion: 2026-05-04.
Riesgos: Inventario deliberadamente acotado; puede haber UI relacionada fuera de los terminos buscados.
```

```text
Caso: WEB-UT-017 director stats proyecta usage summary
Tipo: unit
Comando: `go test -count=1 ./modulos/orquesta-web -run 'TestWebDirectorStats' -v`
Evidencia esperada: `NewWebDirectorStatsPanelV0` conserva totales de uso
agregado (`usage_total_tokens`, `usage_quota_status`) sin exponer provider,
modelo, coste, HOME, OAuth, rutas runtime ni credenciales.
Ultima ejecucion: 2026-05-11; pasa.
Riesgos: La web solo proyecta el contrato; la fuente real de cuota/tokens vive
en conectores del stack/director.
```

```text
Caso: WEB-T209 uso redactado y reporte ausente
Tipo: unit/html contract
Comando: `go test -count=1 ./modulos/orquesta-web`
Evidencia esperada: `director-stats` proyecta uso solo si el contrato trae
`usage_summary`/`agent.usage`; `/ops` no degrada una run viva a cuota `-` si hay
senales de agente y falta reporte, sino `unknown`/`unavailable` con reason code
publico. No expone provider, modelo, HOME, OAuth, coste, rutas, prompts ni
transcripts.
Ultima ejecucion: 2026-05-27; pasa en bateria transversal
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-app-codex-stack ./modulos/orquesta-orchestration-core ./modulos/orquesta-web`.
Riesgos: La fuente real queda en stack/runtime; web solo valida proyeccion.
```

```text
Caso: WEB-UT-018 director stats prepara refresco de agentes
Tipo: unit/integration
Comando: `go test -count=1 ./modulos/orquesta-web`
Evidencia esperada: `/director-stats` y el cliente REST activan
`include_process_refs=true`, `include_agent_progress=true` e
`include_agent_usage=true` cuando hay `run_ref`, proyectan `in_flight`, ticks sin
progreso y repeticiones de accion por agente, y devuelven `refresh.href`
localizable para polling GET.
Ultima ejecucion: 2026-05-12; pasa con `go test -count=1 ./modulos/orquesta-web`.
Riesgos: El progreso real depende de que el bridge MCP reciba un
`ProgressSource` configurado; la web solo consume el contrato existente.
```

```text
Caso: WEB-UT-024 T210 progreso vivo no vuelve a 0%
Tipo: contract
Comando: `go test -count=1 ./modulos/orquesta-orchestration-core ./modulos/orquesta-server ./modulos/orquesta-web ./cmd/orquesta-server`
Evidencia esperada: la web proyecta `percent_complete` y `progress_source` de
stats cuando hay entrega, agente vivo o proceso registrado; `tasks_closed` sigue
en 0 hasta cierre aceptado; `/ops` usa frescura/reason code si falta fuente.
Ultima ejecucion: reejecutada en reconciliacion T210 2026-05-27.
Riesgos: T211 mantiene la politica de cache/agregacion visual de `/ops`; esta
prueba no sustituye ese owner.
```

```text
Caso: WEB-UT-019 director stats proyecta checkpoint pendiente preparado
Tipo: unit
Comando: `go test -count=1 ./modulos/orquesta-web -run 'TestWebDirectorStatsPanelV0ProyectaCheckpointPendienteSiLlegaPorContrato'`
Evidencia esperada: si el contrato consumido entrega
`checkpoint_agents_pending`, `pending_checkpoint_agent_refs` y
`checkpoint_evidence_refs`, el panel web marca atencion, expone refs opacas
deduplicadas y usa texto i18n para progreso de checkpoint.
Ultima ejecucion: 2026-05-13; pasa con `go test -count=1 ./modulos/orquesta-web -run 'TestWebDirectorStatsPanelV0ProyectaCheckpointPendienteSiLlegaPorContrato'`.
Riesgos: `/director-stats` aun no consume `orquesta.server.shutdown.v0`; la
proyeccion queda preparada sin construir un adaptador a shutdown desde web.
```

```text
Caso: WEB-UT-019B director stats proyecta stop_control pendiente
Tipo: unit
Comando: `go test -count=1 ./modulos/orquesta-web -run 'TestWebDirectorStatsPanelV0ProyectaStopControlPendiente'`
Evidencia esperada: si el contrato consumido entrega `stop_control` con
`stop_pending`, el panel web marca atencion, conserva `run_control_status`,
checkpoint y forced, y deduplica refs opacas sin leer runtime ni shutdown.
Ultima ejecucion: 2026-06-25; pasa con `go test -count=1 ./modulos/orquesta-web`.
Riesgos: la precision depende de `director.stats`; la web solo proyecta.
```

```text
Caso: WEB-INT-020 panel de cola multiapp
Tipo: integration/contract
Comando: `go test -count=1 ./modulos/orquesta-web`
Evidencia esperada: `RESTRunQueueClientV0` serializa `rank` y `set_priority`
hacia `/api/v0/runs/queue/priority`, conserva errores publicos 400 del contrato
MCP, y `RunQueueWebEndpointV0` expone GET/POST `/run-queue` sin leer stores,
runtime, DB, scheduler ni cola concreta. El viewmodel incluye `stats_href` por
run para saltar al panel de progreso vivo.
Ultima ejecucion: 2026-05-13; pasa con `go test -count=1 ./modulos/orquesta-web`.
Riesgos: el panel muestra ranking/prioridad; progreso profundo por run no se
duplica y sigue en `/director-stats`.
```

```text
Caso: WEB-CT-024 cliente REST autoprogramming status
Tipo: contract
Comando: `go test -count=1 ./modulos/orquesta-web`
Evidencia esperada: `RESTAutoprogrammingPrepareRunClientV0` tambien implementa
`AutoprogrammingStatusClientV0`, envia `POST /api/v0/autoprogramming/status`
con correlation header, pide progreso de agentes por defecto y proyecta cola,
runs, agentes, diagnosticos y errores publicos sin leer runtime/stores. El
viewmodel conserva la separacion publica entre `running_without_recent_stats`,
`running_live` y `running_stale_no_process` para que la UI no trate una run viva
sin stats recientes como stale reconciliable.
Ultima ejecucion: 2026-05-23; pasa.
Riesgos: El estado real depende de que el bridge MCP tenga inyectados
`run_queue.priority` y `director.stats`; la web solo consume el contrato.
```

```text
Caso: WEB-CT-025 pantalla autoprogramming goal-first
Tipo: contract/ui
Comando: `go test -count=1 ./modulos/orquesta-web -run TestAutoprogrammingWebEndpointV0`
Evidencia esperada: `/autoprogramming` renderiza los marcadores
`goal_migration:goal-first` y `goal_capability:*` en el payload JS, consume
`/api/v0/autoprogramming/goal/observe` para seguimiento Goal, y usa
`/api/v0/autoprogramming/supervise` solo como accion legacy acotada.
Ultima ejecucion: 2026-06-26; pasa.
Riesgos: El lanzamiento real depende de backend Goal opt-in; la web no arranca
proveedor ni sustituye la validacion de cierre del servicio.
```

```text
Caso: WEB-INT-021 panel de control de runs
Tipo: integration/contract
Comando: `go test -count=1 ./modulos/orquesta-web`
Evidencia esperada: `RESTRunControlClientV0` serializa acciones
`pause|resume|stop|cancel` hacia `/api/v0/runs/control`, conserva errores
publicos 400 del contrato MCP, y `RunControlWebEndpointV0` solo acepta POST
JSON/form sin leer stores, runtime, procesos, scheduler ni DB.
Ultima ejecucion: 2026-05-13; pasa con `go test -count=1 ./modulos/orquesta-web`.
Riesgos: checkpoint y parada fisica siguen fuera de la web; el panel solo emite
la orden publica.
```

```text
Caso: WEB-UT-025 panel ops expone flujo comparativo
Tipo: unit/html contract
Comando: `go test -count=1 ./modulos/orquesta-web -run TestOpsDashboardWebEndpointV0RenderizaPanelLiveCompleto`
Evidencia esperada: `/ops` contiene la franja `flujo operativo` con cola,
ejecucion, atencion y cierre; el resumen se calcula en `renderFlowSummary`
desde snapshots ya consumidos por el panel y no anade endpoints, DB, runtime ni
proveedor.
Ultima ejecucion: 2026-05-26; pasa con
`GOCACHE=/tmp/orquesta-go-cache go test -count=1 ./modulos/orquesta-web -run TestOpsDashboardWebEndpointV0RenderizaPanelLiveCompleto`.
Riesgos: La precision depende de `autoprogramming/status` y `director/stats`;
si esas fuentes degradan, el panel solo refleja la proyeccion publica.
```

```text
Caso: WEB-UT-028 panel ops muestra decision del Director
Tipo: unit/html contract
Comando: `go test -count=1 ./modulos/orquesta-web -run TestOpsDashboardWebEndpointV0RenderizaPanelLiveCompleto`
Evidencia esperada: `/ops` contiene el bloque `director autónomo`, `Estado del
Director`, `ops_snapshot`, `opsSnapshotDecision` y `renderDirectorDecision`;
la decision visible prefiere `DirectorAutonomousOpsSnapshotV0` publicado por
`director.stats`/`autoprogramming.status` y conserva fallback legacy si el
servidor aun no envia snapshot. No anade endpoint, store, DB, runtime ni
proveedor.
Ultima ejecucion: 2026-06-08, pasa con `go test -count=1 ./modulos/orquesta-web -run TestOpsDashboardWebEndpointV0RenderizaPanelLiveCompleto`.
Revalidacion 2026-06-28: la shell `/ops` contiene etiqueta y razon para
`repair_goal_state`, accion publicada por el snapshot cuando un goal-first tiene
marcador durable pero falta `GoalWorkStateV0`.
Riesgos: La fuente contractual ya es `ops_snapshot`; falta ampliar el snapshot
con modelos/runtime, waits, olas y cohortes cuando esos puertos se publiquen.
```

```text
Caso: WEB-UT-029 panel ops no anida helpers live/runtime dentro de chunks partidos
Tipo: unit/html contract
Comando: `go test -count=1 ./modulos/orquesta-web -run TestOpsDashboardWebEndpointV0`
Evidencia esperada: `opsDashboardHTMLV0` ensambla `opsDashboardHTMLChunk3RuntimeDetailV0`
y `opsDashboardHTMLLiveCachePolicyV0` en el punto final previo al primer
refresh; `refreshAll` puede ver `runtimeDetailHTML`, `buildRuns` y las
funciones de stats en scope global del script.
Ultima ejecucion: 2026-06-08, pasa con `go test -count=1 ./modulos/orquesta-web -run TestOpsDashboardWebEndpointV0`; validacion local adicional con Orquesta temporal `127.0.0.1:8788`, API `validate-request`/`prepare-run`/`runs/supervise`, CDP mostrando `live` y captura `/tmp/orquesta-ops-ui-8788-final.png`.
Riesgos: Cubre el orden de chunks; la prueba visual/API local sigue siendo
necesaria para comprobar el refresco real del navegador.
```

```text
Caso: WEB-UT-030 panel ops no convierte stalled en atencion dura
Tipo: unit/html contract
Comando: `go test -count=1 ./modulos/orquesta-web -run TestOpsDashboardWebEndpointV0`
Evidencia esperada: `agentNeedsAttention` y `runNeedsAttention` no usan
`stalled_agents`, `no_progress_ticks` ni `stall` para marcar atencion; el panel
sigue mostrando la señal stalled como telemetria/filtro.
Ultima ejecucion: 2026-06-08, pasa con `go test -count=1 ./modulos/orquesta-web -run TestOpsDashboardWebEndpointV0`.
Riesgos: No cambia la semantica de `director-stats`; solo evita que `/ops`
duplique el rail blando en la decision visible del Director.
```

```text
Caso: WEB-UT-030B autoprogramming status no convierte stalled-only en atencion dura
Tipo: unit/viewmodel
Comando: `go test -count=1 ./modulos/orquesta-web -run TestWebAutoprogrammingStatusViewModelV0StalledSoloNoEsAtencionDura`
Evidencia esperada: un run con `stalled_agents > 0`, sin agentes fallidos y sin
cierre bloqueado conserva la telemetria `stalled_agents` pero expone
`needs_attention=false`.
Ultima ejecucion: 2026-06-28, pasa con el comando focal.
Riesgos: No cambia el contrato MCP ni oculta la senal; solo evita tratarla como
bloqueo duro desde la web.
```

```text
Caso: WEB-UT-031 panel ops invoca run supervisor
Tipo: unit/html contract
Comando: `go test -count=1 ./modulos/orquesta-web -run TestOpsDashboardWebEndpointV0`
Evidencia esperada: `/ops` contiene boton global `Acción segura`, `Avanzar run`
y `Avanzar run desde fila`; para runs legacy usa `superviseSelectedRun`,
`superviseQueueRow` y `superviseOps` contra `POST /api/v0/runs/supervise` solo
cuando no hay accion Goal; para runs goal-first usa `advanceSelectedRun`,
`advanceQueueRow` y `observeGoalOps` contra
`POST /api/v0/apps/director/goal/observe`, mostrando `stop_reason`, `ticks`,
`last.status` o estado de Goal segun corresponda.
Ultima ejecucion: 2026-06-27, pasa con `go test -count=1 ./modulos/orquesta-web -run 'TestOpsDashboardWebEndpointV0RenderizaPanelLiveCompleto|TestOpsDashboardWebEndpointV0GoalFirstUsaObserveGoalEnAvance|TestOpsDashboardWebEndpointV0SupervisorPayloadAcotado'`.
Riesgos: La prueba HTML fija el contrato de UI; una prueba local con servidor
temporal debe validar que el executor real/fake responde.
```

```text
Caso: WEB-UT-032 nueva-app muestra panel goal-first
Tipo: unit/html contract
Comando: `go test -count=1 ./modulos/orquesta-web -run 'NuevaAppHTML|NuevaAppI18n|ArrancarDirector'`
Evidencia esperada: el HTML de `/nueva-app` contiene el panel `Director y goal`
cuando el viewmodel trae `Director`, muestra `run_ref`, `goal_ref`,
`external_goal_ref`, `goal_status`, `director_execution_mode` y boton
`Actualizar goal`; el JS llama a
`POST /api/v0/apps/director/goal/observe`, actualiza `run_status` y
`closure_status`, y arranca polling acotado con parada terminal si tambien hay
`run_ref`, sin introducir cliente Go, store, runtime ni proveedor.
Un resultado legacy con `director_execution_mode=legacy_director_loop` y sin
`goal_ref` renderiza el modo, pero deja `data-goal-auto-poll=false` y no muestra
boton de observacion de goal.
Un viewmodel defensivo goal-first con `goal_ref` pero sin `run_ref` muestra refs
de goal, pero deja `data-goal-auto-poll=false` y no muestra boton porque no
existe clave observable para `POST /api/v0/apps/director/goal/observe`; la
frontera REST/MCP debe rechazar esa respuesta como `run_ref_requerido`.
Ultima ejecucion: 2026-06-25; pasa con el comando indicado tras anadir polling
automatico goal-first.
Evidencia de servidor 2026-06-28: `go test -count=1 ./cmd/orquesta-server -run TestServerNuevaAppHTMLGoalFirstPOSTRenderizaYObservaV0`
monta el stack real con backend Goal fake, valida `GET /nueva-app`, `POST
/nueva-app` por formulario, render de panel goal-first con autopoll y observacion
manual por `POST /api/v0/apps/director/goal/observe` hasta cierre aceptado.
Riesgos: El test cubre contrato HTML/JS server-rendered; la respuesta real del
endpoint queda cubierta en `orquesta-mcp`, `orquesta-app-gateway` y stack Codex.
```

```text
Caso: WEB-UT-033 director-stats proyecta trabajo externo
Tipo: unit/web contract
Comando: `go test -count=1 ./modulos/orquesta-web -run 'DirectorStats.*TrabajoExterno|RESTDirectorStatsClient|DirectorStatsWebEndpoint'`
Evidencia esperada: `/director-stats` acepta `external_job_ref` sin `run_ref`,
preserva `app_ref`/`external_job_ref` en refresh, el cliente REST conserva el
bloque `external_job` y el viewmodel marca atencion cuando el trabajo externo
llega como `parent_ack_received/cohort_open` o `integration_required`, sin leer
OPES ni runtime.
Ultima ejecucion: 2026-06-26; pasa dentro de `go test -count=1 ./modulos/orquesta-web`.
Riesgos: El bloque depende de que MCP/stack inyecten `ExternalJobSource`; la
web no inventa diagnosticos si el contrato no los trae.
```

```text
Caso: WEB-UT-027 tablas responsivas de panel ops
Tipo: unit/html contract
Comando: `go test -count=1 ./modulos/orquesta-web -run TestOpsDashboardWebEndpointV0RenderizaPanelLiveCompleto`
Evidencia esperada: `/ops` contiene `tableCell`, `data-label`,
`content: attr(data-label)`, tabla de fases y tabla de uso; las filas siguen
seleccionando por refs opacas y no anaden endpoints, stores, DB ni runtime.
Ultima ejecucion: 2026-05-26; pasa con
`GOCACHE=/tmp/orquesta-gocache go test -count=1 ./modulos/orquesta-web -run TestOpsDashboardWebEndpointV0RenderizaPanelLiveCompleto`.
Riesgos: No sustituye una prueba visual con navegador; cubre contrato HTML
server-rendered.
```

```text
Caso: WEB-UT-026 errores publicos de render HTML
Tipo: unit
Comando: `go test -count=1 ./modulos/orquesta-web`
Evidencia esperada: `TestWebHTMLTemplateResponseV0RenderErrorDevuelveReasonPublico`,
`TestWebHTMLTemplateResponseV0WriteErrorDevuelveReasonCompacto` y
`TestAppChangeHTMLV0UsaFallbackComunAnteRenderError` validan buffer previo a
escritura, fallback HTML con `web_html_render_failed`, reason
`web_response_write_failed` en fallo de socket y ausencia de detalles privados
del template/error.
Ultima ejecucion: 2026-05-26; pasa.
Riesgos: El slice cubre renderers HTML locales; T183 transversal aun requiere
observability/gateway/cmd para auditoria y contadores compactos.
```

```text
Caso: WEB-UT-034 autoprogramming muestra acciones seguras
Tipo: unit/html+viewmodel
Comando: `go test -count=1 ./modulos/orquesta-web`
Evidencia esperada: `/autoprogramming` contiene `safe-actions-panel`,
`safe_actions` y "Acciones seguras"; el view-model conserva `observe_goal`
con endpoint permitido y payload compacto, descartando claves locales o de
proveedor como `local_path`/`provider_id`.
Ultima ejecucion: 2026-06-27; pasa con el comando indicado.
Riesgos: La web solo proyecta y dispara acciones publicadas por la API; no
calcula politicas nuevas ni recupera trabajos OPES automaticamente.
```
