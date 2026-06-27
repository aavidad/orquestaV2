# Contratos locales: orquesta-observability

Registra puertos, DTOs y eventos que `orquesta-observability` expone o consume.

## `PublicarOrquestaEvent v0`

Nombre: `PublicarOrquestaEvent`
Tipo: puerto_entrada
Version: `v0`
Propietario: `orquesta-observability`
Estado global: promovido a contrato compartido en `modulos/CONTRATOS.md` por decision del director de 2026-05-04.
Consumidores/productores:

- `orquesta-core`, `orquesta-runtime`, `orquesta-capacity` y `orquesta-review` publican eventos compactos por este puerto.
- `orquesta-web` y `orquesta-mcp` son solo lectores futuros mediante puertos de consulta/proyeccion; no escriben eventos ni consumen el sink directamente.
- DB, bus, fichero, memoria u otro event sink son adaptadores de salida, nunca parte del contrato de dominio.

Request: `PublishOrquestaEventRequestV0`

Campos:

- `request_id`: identificador tecnico opaco de la llamada al puerto.
- `correlation_id`: identificador opaco compartido por flujo.
- `idempotency_key`: clave opaca obligatoria para evitar duplicados tecnicos en adaptadores.
- `event`: `OrquestaEventV0`.
- `dry_run`: booleano opcional para validacion sin entrega al adaptador.

Salida correcta: `PublishOrquestaEventAcceptedV0`

- `accepted`: `true`.
- `event_id`: eco de `event.event_id`.
- `correlation_id`: eco de `correlation_id`.
- `sink_receipt_id`: identificador tecnico opaco emitido por el adaptador cuando exista.
- `stored`: `false` en el corte v0 sin DB real; un adaptador futuro podra cambiar este valor bajo contrato versionado.

Errores publicos:

- `orquesta_event_invalido`
- `evento_demasiado_extenso`
- `secreto_detectado`
- `transcript_no_permitido`
- `correlation_id_requerido`
- `idempotency_key_requerida`
- `source_area_no_soportada`
- `event_type_incompatible`
- `sink_no_disponible`
- `duplicado_idempotente`

Invariantes:

- El puerto valida forma, tamanos, privacidad, origen y correlacion antes de entregar al adaptador.
- El puerto no decide negocio, capacidad, review, merge ni runtime; solo observa y normaliza.
- `request_id`, `correlation_id`, `event_id`, `trace_id`, `causation_id`, `subject.id` y `sink_receipt_id` son opacos; no codifican rutas, HOME, proveedor, tabla, SQL ni significado de negocio.
- Ningun evento contiene secretos, tokens, credenciales, rutas HOME reales, prompts completos, completions completas, transcripts completos, SQL, DSN, tablas ni detalles de conexion.
- Los eventos son compactos: resumen corto, payload acotado, enlaces por referencias opacas y metricas agregadas.
- No existe DB real en v0. Persistir, indexar o consultar eventos requiere adaptador y contrato/smoke posterior.
- MCP/web leen proyecciones compactas futuras; no acceden al event sink ni a tablas.
- Los eventos de `capacity` pueden exponer nivel, esfuerzo, cuota agregada y motivo compacto, pero no proveedor, modelo concreto, cuenta ni HOME real.
- Los eventos de `runtime` pueden exponer estado, duracion, contadores y referencias opacas, pero no transcript ni comandos con secretos.
- Los eventos de `review` pueden exponer resultado, conteos y enlaces a artifacts, pero no diffs completos ni contenido privado.
- Los eventos de `core` pueden exponer transiciones y referencias opacas, pero no entidades privadas ni reglas internas completas.

Pruebas de contrato:

- Schema canonico: `docs/schemas/orquesta_event_v0.schema.json`.
- Fixture valido: `docs/fixtures/orquesta_event_v0/core_event_minimo_valido.json`.
- Fixtures invalidos: `docs/fixtures/orquesta_event_v0/transcript_completo_invalido.json` y `docs/fixtures/orquesta_event_v0/secreto_detectado_invalido.json`.

## `OrquestaEventV0`

Nombre: `OrquestaEventV0`
Tipo: dto | evento
Version: `v0`
Propietario: `orquesta-observability`
Consumidores:

- Publicadores autorizados: `orquesta-core`, `orquesta-runtime`, `orquesta-capacity`, `orquesta-review`.
- Lectores futuros: proyecciones MCP/web mediante puertos de lectura por definir.

Campos:

- `schema_version`: constante `orquesta_event.v0`.
- `event_id`: identificador opaco unico del evento.
- `event_type`: nombre compacto con prefijo `core.`, `runtime.`, `capacity.` o `review.`.
- `source_area`: `core`, `runtime`, `capacity` o `review`; debe coincidir con el prefijo de `event_type`.
- `occurred_at`: fecha/hora ISO-8601 del hecho observado.
- `severity`: `debug`, `info`, `warning` o `error`.
- `outcome`: `observed`, `accepted`, `rejected`, `running`, `completed`, `failed`, `degraded` o `skipped`.
- `correlation`: IDs opacos de correlacion, request, causa y trace.
- `subject`: referencia opaca al elemento observado.
- `producer`: modulo y puerto que emitieron el evento.
- `privacy`: declaracion obligatoria de ausencia de secretos y transcripts.
- `summary`: resumen operativo corto, no texto de transcript.
- `links`: enlaces opcionales a eventos, artifacts o ejecuciones por IDs opacos.
- `payload`: objeto compacto con datos agregados o normalizados.

Invariantes:

- `privacy.contains_secret` y `privacy.contains_transcript` siempre son `false`.
- `payload` admite solo claves snake_case compactas y rechaza nombres asociados a secretos, transcripts, SQL, tablas, DSN, conexiones, proveedores, prompts, completions y texto bruto.
- Los textos libres estan limitados y deben ser resumen operativo, no contenido fuente completo.
- Si un consumidor necesita detalle completo, debe obtenerlo por otro contrato autorizado; el evento solo enlaza con IDs opacos.

## `OperationalStatusQueryV0`

Nombre: `OperationalStatusQueryV0`
Tipo: puerto_entrada
Version: `v0`
Propietario: `orquesta-observability`
Estado global: promovido a contrato compartido en `modulos/CONTRATOS.md` por decision del director de 2026-05-04.
Consumidores autorizados:

- `orquesta-cli`, `orquesta-mcp` y `orquesta-web`, como adaptadores finos read-only para diagnostico/progreso compacto.
- `orquesta-core`, solo como lector de progreso agregado si existe un flujo superior autorizado.
- DB, runtime, event sink, filesystem, bus, procesos y proveedores son adaptadores o fuentes externas; no son consumidores del puerto ni quedan expuestos.

Request: `OperationalStatusQueryV0`

Campos:

- `schema_version`: constante `operational_status_query.v0`.
- `request_id`: identificador tecnico opaco de la consulta.
- `correlation_id`: identificador opaco para enlazar la consulta con el flujo solicitante.
- `consumer`: modulo y canal solicitante; pares permitidos: `orquesta-cli`/`cli`, `orquesta-mcp`/`mcp`, `orquesta-web`/`web` u `orquesta-core`/`core`.
- `locale`: locale solicitado para claves i18n o mensajes publicos compactos.
- `scope`: alcance logico de la consulta; valores permitidos: `sistema`, `proyecto`, `flujo`, `tarea`, `runtime`, `capacity` o `review`.
- `subject_ref`: referencia opaca opcional del elemento observado dentro del `scope`.
- `trace_ref`: referencia opaca opcional para acotar por traza.
- `time_window`: ventana temporal logica opcional, sin SQL ni nombres de tabla.
- `include_sections`: lista acotada de secciones solicitadas; valores permitidos: `estado`, `progreso`, `bloqueos`, `salud`, `actividad_reciente`, `contadores` o `referencias`.
- `limit`: limite compacto para listas de resultados, nunca usado como paginacion de DB directa.
- `freshness`: frescura solicitada de la proyeccion compacta, expresada como edad maxima o watermark opaco.

Salida correcta: `DiagnosticoCompactoV0`

Errores publicos:

- `operational_status_query_invalida`
- `consumidor_no_autorizado`
- `scope_no_soportado`
- `referencia_no_opaca`
- `consulta_demasiado_amplia`
- `proyeccion_no_disponible`
- `diagnostico_no_disponible`
- `frescura_no_garantizada`
- `secreto_detectado`
- `transcript_no_permitido`

Invariantes:

- El puerto es read-only: no persiste, no arranca runtime, no reintenta tareas, no decide negocio, no ejecuta recuperacion y no modifica estado.
- El contrato no autoriza acceso directo a DB, runtime, event sink, filesystem, bus, procesos, tablas, SQL, DSN, HOME real, proveedor ni modelo concreto.
- La implementacion futura debe leer proyecciones compactas mediante adaptadores autorizados; si no hay proyeccion disponible, devuelve error publico o diagnostico `unknown`, no cruza internals.
- Todas las referencias son opacas: `request_id`, `correlation_id`, `subject_ref`, `trace_ref`, `projection_ref`, `event_ref`, `artifact_ref`, `runtime_ref` y `watermark_ref`.
- No contiene secretos, tokens, credenciales, rutas HOME reales, prompts completos, completions completas, transcripts completos, diffs completos, SQL, tablas ni detalles de conexion.
- Los textos libres son resumen operativo corto o claves i18n; CLI/MCP/Web deciden solo la presentacion, no el significado contractual.
- El resultado es estable para CLI/MCP/Web: el mismo request logico produce el mismo shape de DTO aunque cambie el transporte.
- La promocion global ya esta cerrada; adaptadores reales, fuentes de proyeccion y recuperacion activa requieren contratos o microtareas separados.

Pruebas de contrato:

- DTO y validacion pura Go: `operational_status_v0.go`.
- Pruebas unitarias: `operational_status_v0_test.go`.
- Adaptador puro en memoria para contract tests de consumidores: `operational_status_memory_adapter_v0.go`.
- Schema JSON y fixtures quedan pendientes de una microtarea posterior si hacen falta para consumidores externos.

## `WorkspaceTimelineQueryV0`

Nombre: `WorkspaceTimelineQueryV0`
Tipo: puerto_entrada
Version: `v0`
Propietario: `orquesta-observability`
Consumidores autorizados:

- API residente, MCP y web como adaptadores read-only.
- Las fuentes reales de eventos, auditoria, cola, runtime/progreso, Git y
  uso/coste entran por adaptadores de lectura; si no estan cableadas devuelven
  `not_available`.

Request: `WorkspaceTimelineQueryV0`

Campos:

- `request_id`, `correlation_id`: refs opacas de consulta.
- `consumer`: modulo/canal autorizado.
- `locale`: locale de presentacion.
- `scope`: debe ser `workspace`.
- `agent_ref`, `project_ref`, `task_ref`: filtros opacos opcionales.
- `time_window`: ventana por `from`, `to` o `preset`.
- `page`: `limit` compacto y `cursor_ref` opaco opcional.
- `sources`: lista declarada entre `events`, `audit`, `run_queue`,
  `runtime_progress`, `git_stats` y `usage_cost`.

Salida correcta: `WorkspaceTimelineV0`

Errores publicos:

- `workspace_timeline_query_invalida`
- `workspace_timeline_no_disponible`
- `consumidor_no_autorizado`
- `scope_no_soportado`
- `referencia_no_opaca`
- `consulta_demasiado_amplia`
- `secreto_detectado`
- `transcript_no_permitido`

Invariantes:

- El puerto es read-only y no reconstruye estado desde shell, Git local,
  runtime filesystem ni transcripts crudos.
- Todas las fuentes solicitadas aparecen con estado `available`, `partial` o
  `not_available`; las fuentes ausentes deben incluir razon y evidencia compacta.
- Los items exponen resumen, refs opacas y clasificacion redactada; no incluyen
  prompts, completions, transcripts, HOME, tokens, secretos, SQL, DSN ni payloads
  HTTP crudos.
- API, MCP y web deben consumir este mismo puerto de lectura.

Pruebas de contrato:

- DTO y validacion pura Go: `workspace_timeline_*_v0.go`.
- Adaptadores offline: `WorkspaceTimelineUnavailableReaderV0` y
  `WorkspaceTimelineStaticReaderV0`.

## `SafeHistoricalSignalExtractionPolicyV0`

Nombre: `SafeHistoricalSignalExtractionPolicyV0`
Tipo: dto | politica_documental
Version: `v0`
Propietario: `orquesta-observability`
Consumidores autorizados:

- Adaptadores futuros de lectura historica usados por `orquesta-observability`.
- Revisores de contrato y equipos de integracion que necesiten implementar una proyeccion compacta.
- No habilita consumo directo desde CLI, MCP, Web, Core ni acceso a DB productiva.

Fuentes candidatas:

- `runtime_transcript`
- `runtime_telemetry_samples`
- `audit_log`

Extracciones permitidas:

- contadores agregados por ventana temporal, severidad, fase, outcome o area;
- tasas, percentiles y distribuciones compactas ya agregadas;
- top-N pequeno de codigos, clases o claves publicas;
- filtros por `correlation_id`, `trace_ref`, `subject_ref`, fase logica o rango temporal acotado;
- ejemplos pequenos ya saneados, con longitud estrictamente limitada y sin texto fuente completo;
- referencias opacas a evidencia cuando el detalle deba consultarse por otro contrato futuro autorizado.

Contenido prohibido:

- transcripts completos o parciales largos;
- prompts, completions, mensajes de sistema o cadenas de pensamiento;
- SQL, nombres de tabla, DSN, planes de consulta o payloads crudos de DB;
- secretos, tokens, credenciales, cookies, API keys, rutas HOME reales y detalles de conexion;
- comandos completos, diffs completos, stacktraces extensos o blobs multilinea sin saneado;
- nombres concretos de proveedor, cuenta, HOME o modelo cuando no sean imprescindibles para el agregado.

Invariantes:

- La politica no habilita adaptador DB, lectura productiva, servidor, job de backfill ni acceso operativo a V1.
- Toda extraccion debe producir salida compacta y estable, apta para `OrquestaEventV0` o `DiagnosticoCompactoV0`.
- Los filtros son logicos y opacos; no exponen SQL ni estructura fisica.
- Los ejemplos pequenos son opcionales y deben quedar reducidos a evidencia minima ya saneada.
- Si una senal solo puede obtenerse exponiendo material prohibido, la salida correcta es omitirla o devolver `unknown`.
- Cualquier uso de fuentes historicas reales requiere una microtarea posterior de adaptador y su validacion propia.

Pruebas de contrato:

- Reglas documentales en `docs/decisiones.md`, `docs/tareas.md` y `docs/pruebas.md`.
- La implementacion real queda fuera de este corte.

## `DiagnosticoCompactoV0`

Nombre: `DiagnosticoCompactoV0`
Tipo: dto
Version: `v0`
Propietario: `orquesta-observability`
Estado global: promovido a contrato compartido en `modulos/CONTRATOS.md`; salida de `OperationalStatusQueryV0`.
Consumidores autorizados:

- `orquesta-cli`, `orquesta-mcp` y `orquesta-web`, mediante adaptadores finos read-only.
- `orquesta-core`, solo como lector agregado si existe un flujo superior autorizado.
- Ningun consumidor accede a DB, runtime, event sink o tablas a traves de este DTO.

Campos:

- `schema_version`: constante `diagnostico_compacto.v0`.
- `diagnostic_id`: identificador opaco del diagnostico generado.
- `generated_at`: fecha/hora ISO-8601 de generacion.
- `correlation_id`: eco opaco de la consulta.
- `scope`: alcance logico diagnosticado.
- `subject_ref`: referencia opaca diagnosticada, si aplica.
- `projection_ref`: referencia opaca a la proyeccion compacta usada.
- `freshness`: estado de frescura con `watermark_ref`, `max_age_seconds`, `partial` y `stale`.
- `estado`: estado operativo compacto; valores permitidos: `ok`, `degraded`, `blocked`, `failed` o `unknown`.
- `progreso`: resumen agregado con contadores, porcentaje opcional si el total es conocido y fase logica compacta.
- `salud`: checks compactos por area, severidad, estado, clave i18n y referencias de evidencia.
- `bloqueos`: lista acotada de bloqueos observados con severidad, area propietaria, resumen corto y referencias opacas.
- `actividad_reciente`: lista acotada de cambios observados; enlaza eventos o artifacts por referencias opacas sin payload completo.
- `contadores`: metricas agregadas permitidas, sin series crudas ni muestras privadas.
- `referencias`: enlaces opacos a eventos, artifacts, trazas, runtime handles o consultas relacionadas.
- `warnings`: advertencias publicas de completitud, frescura o secciones omitidas.
- `privacy`: declaracion obligatoria de ausencia de secretos, transcripts, prompts, completions, SQL y detalles de conexion.

Invariantes:

- `privacy.contains_secret`, `privacy.contains_transcript`, `privacy.contains_prompt`, `privacy.contains_completion` y `privacy.contains_connection_detail` siempre son `false`.
- El DTO resume progreso y diagnostico; no prescribe acciones, no asigna responsables operativos nuevos y no cambia decisiones de core/capacity/runtime/review.
- `bloqueos`, `salud` y `actividad_reciente` contienen resumen y referencias, no contenido fuente completo.
- `contadores` contiene agregados compactos; no incluye samples crudos, transcript, comandos completos, diffs completos ni payloads de eventos completos.
- `contadores` queda acotado a 24 claves por diagnostico para permitir
  proyecciones residentes con metricas base y senales goal-first sin borrar
  contadores historicos.
- Si una seccion no puede calcularse sin cruzar internals, se omite con warning publico o se marca `unknown`.
- Si una seccion futura usa evidencia historica V1, debe respetar `SafeHistoricalSignalExtractionPolicyV0` y seguir siendo compacta, saneada y opaca.
- Cualquier campo nuevo que exponga consumidores globales, operaciones de recuperacion o fuentes reales requiere version nueva o `CONSULTA AL DIRECTOR`.

Pruebas de contrato:

- DTO y validacion pura Go: `operational_status_v0.go`.
- Adaptador puro en memoria para `OperationalStatusQueryV0`: `operational_status_memory_adapter_v0.go`.
- Pruebas unitarias: `operational_status_v0_test.go`.
- El corte no implementa DB, sink, servidor, filesystem, runtime ni recuperacion activa.

## `OperationalStatusMemoryAdapterV0`

Nombre: `OperationalStatusMemoryAdapterV0`
Tipo: adaptador_test | query_handler_memoria
Version: `v0`
Propietario: `orquesta-observability`
Consumidores:

- Contract tests de consumidores read-only (`orquesta-cli`, `orquesta-mcp`, `orquesta-web` y lector agregado autorizado de `orquesta-core`).
- No es backend productivo ni fuente canonica de estado.

Entrada constructor:

- Lista de `DiagnosticoCompactoV0` ya compactos, saneados y suministrados por el test.

Entrada query:

- `OperationalStatusQueryV0`.

Salida correcta:

- Copia de `DiagnosticoCompactoV0` encontrada por `scope`, `subject_ref` y `correlation_id`, filtrada por secciones solicitadas y limite compacto.

Errores publicos:

- Hereda errores de `ValidateOperationalStatusQueryV0`.
- Hereda errores de `ValidateDiagnosticoCompactoV0` durante construccion.
- `proyeccion_no_disponible` si no hay proyeccion/diagnostico suministrado.
- `diagnostico_no_disponible` si no existe match opaco para `scope`, `subject_ref` y `correlation_id`.
- `frescura_no_garantizada` si la frescura solicitada no puede satisfacerse con el diagnostico suministrado.

Invariantes:

- Es puro en memoria y read-only: no lee DB, event sink, runtime, filesystem, bus, procesos ni red.
- No persiste, no arranca runtime, no recupera tareas, no interpreta negocio y no modifica estado.
- Valida todas las queries con `ValidateOperationalStatusQueryV0` y todos los diagnosticos de entrada con `ValidateDiagnosticoCompactoV0`.
- Las claves de lookup se tratan como opacas y no codifican rutas, SQL, tablas, DSN, HOME, proveedor ni significado de negocio.
- Devuelve copias defensivas para que tests consumidores no muten el estado interno.
- No expone secretos, transcripts, prompts, completions, SQL, DSN, HOME ni detalles de conexion.

Pruebas de contrato:

- Unitarias en `operational_status_memory_adapter_v0_test.go`.
- Validacion obligatoria: `go test -count=1 ./modulos/orquesta-observability`.

## `DirectorDecisionContextV0`

Nombre: `DirectorDecisionContextV0`
Tipo: dto | proyeccion_compacta
Version: `v0`
Propietario: `orquesta-observability`
Consumidores autorizados:

- `orquesta-mcp`, `orquesta-app-gateway`, API REST y web como lectores de la
  proyeccion ya calculada por puertos inyectados.

Campos:

- `run_ref`, `observed_at` y `current_phase`.
- `progress`: progreso agregado por tareas observadas/cerradas/abiertas y
  agentes con senal.
- `lifecycle`: agentes pedidos, vivos, parados, fallidos, en vuelo y con control
  de proceso registrado o ausente.
- `closure`: estado de cierre, bloqueo y causas `blocked_by`.
- `quietness`: quietud por falta de senal y ticks sin progreso.
- `rework_replan`: contadores y refs opacas de rework/replan.
- `phases`, `tasks`, `agents`, `blockers` y `activity_recent` como listas
  compactas con refs opacas y claves i18n.
- `privacy`: flags obligatoriamente falsos para secretos, transcripts, prompts,
  completions y detalles de conexion.

Invariantes:

- No decide negocio ni cierre; solo empaqueta lo que el run/progreso/procesos ya
  exponen.
- No lee DB, runtime, filesystem, procesos ni proveedor; esas fuentes entran por
  adaptadores anteriores.
- No expone rutas locales, HOME, credenciales, proveedor, modelo, transcripts,
  prompts, completions, SQL ni diffs completos.
- La actividad reciente es una timeline compacta derivada de refs y claves i18n,
  no un volcado de eventos ni de transcript.

Pruebas de contrato:

- DTO y validacion: `director_decision_context_v0_test.go`.
- Integracion MCP/API: `orquesta.director.stats.v0` devuelve
  `decision_context` junto a `stats`.

## `DirectorAutonomousOpsSnapshotV0`

Nombre: `DirectorAutonomousOpsSnapshotV0`
Tipo: dto | proyeccion_compacta
Version: `v0`
Propietario: `orquesta-observability`
Consumidores autorizados:

- `orquesta-mcp`, `/ops`, API REST y agentes directores como lectores de
  estado operativo.

Campos:

- `queue`: ref de cola, estado live, count y refs de runs ordenadas.
- `runs`: run, app, estado, fase, cierre, progreso, agentes, rework/replan,
  uso/cuota y campos opcionales para waits, olas y cohortes.
- `agents`: run, agente, tarea, estado, progreso, capacidad, runtime kind y
  uso/cuota si ya fueron publicados por puertos neutrales.
- `decision`: action, scope, run_ref, attention, reason_code, summary_key y
  evidence_refs.
- `privacy`: metadata-only.

Invariantes:

- Es read-only: no arranca runtime, no pausa agentes, no cierra runs y no
  decide efectos externos.
- No filtra, corta ni descarta entregas; solo resume estado observable para
  cockpit/UI/director.
- No conoce Codex, Gemini, Claude, OPES, DB, HOME, OAuth, proveedor ni modelo.
- `decision.action=observe_goal` es una senal neutral de cockpit para observar
  un goal ya publicado por puertos externos; no implica que observabilidad
  conozca ni ejecute el backend de Goal.
- Las formas no disponibles se dejan vacias; waits, olas, cohortes y modelos se
  rellenaran cuando entren por puertos publicos.

Pruebas de contrato:

- Integracion MCP/API: `orquesta.director.stats.v0` y
  `orquesta.autoprogramming.status.v0` devuelven `ops_snapshot`.

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
