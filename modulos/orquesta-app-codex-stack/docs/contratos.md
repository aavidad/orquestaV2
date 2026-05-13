# Contratos: orquesta-app-codex-stack

## `CodexAppStackConfigV0`

Contrato de configuracion previsto para la composition externa.

Campos conceptuales:

- `enabled`: opt-in explicito;
- `gateway`: handler web/API/MCP ya construido por `orquesta-app-gateway`;
- `director_service`: servicio `StartAppDirectorV0`;
- `director_ports`: `StartAppDirectorPortsV0` reales;
- `codex_runtime`: comando, HOME, CODEX_HOME, PATH, sandbox y approval policy;
- `delivery_store`: store externo para descriptors ACK;
- `progress_store`: store externo para estado compacto anti-bucle;
- `process_registry`: registro externo de procesos/sesiones;
- `worktree`: resolutores de proyecto, runtime dir, baseline y verificacion;
- `review_gate`: evidencia de ficheros, politica de tamano y estado de fallo;
- `capacity`: decision externa de capacidad/proveedor/modelo;
- `persistence`: referencia externa a persistencia operacional, si aplica.

Invariantes:

- `enabled` debe ser verdadero para arrancar Codex real.
- No existe default de proveedor, modelo, DB, HOME, CODEX_HOME, puerto HTTP ni
  path de runtime.
- El modelo puede venir en `capacity` o configuracion equivalente, pero nunca se
  fija dentro de este modulo.
- La DB puede ser SQLite, Postgres, filesystem, broker o memoria segun el
  operador, pero este modulo solo recibe puertos/refs ya configurados.
- Los errores publicos no incluyen paths absolutos, tokens, prompts,
  transcripts, HOME, provider ni modelo salvo refs opacas aptas para auditoria.

## `StartAppDirectorPortsV0` reales

El stack debe inyectar el mismo contrato que consume
`orquesta-app-director-service`.

Puertos requeridos por composicion:

- store de run;
- event sink;
- outbox ledger;
- dispatchers del loop;
- fuentes opcionales de entrega, progreso, leases, replan y decisiones;
- store de microtareas de director cuando una decision crea trabajo nuevo.

Reglas:

- REST, MCP y web no construyen estos puertos directamente.
- El gateway solo monta handlers; la composition externa decide los conectores.
- Los puertos de Codex delivery/progress se conectan como observadores del
  servicio, no como dependencias del core.
- Si falta un puerto real requerido, el arranque falla antes de lanzar agentes.

## Shutdown cooperativo de agentes Codex

Contrato interno del stack:

```text
PrepareAgentShutdownPortV0
  -> stats del run por puertos
  -> descriptors de agentes Codex por ReceiptStore
  -> request de checkpoint en runtime_work_dir
  -> ACK de checkpoint validado por runtime-codex
  -> RecordRunCheckpointV0 solo si todos responden
```

Invariantes:

- El stack no conoce detalles internos del core; usa `RunStore`,
  `ReceiptStore`, `ProcessRegistry`, progreso y uso por puertos.
- Si no hay agentes en vuelo, registra checkpoint conservador como antes.
- Si hay agentes en vuelo, escribe una request por agente y exige ACK
  `codex_shutdown_checkpoint_ack.v0`.
- Si falta descriptor, runtime dir o ACK valido, devuelve
  `checkpoint_recorded=false` y `pending_agent_refs`.
- No se exponen rutas de runtime, HOME, modelo, proveedor ni DB en el contrato
  publico; solo refs compactas.

## Ruta `/nueva-app` opt-in

Contrato funcional:

```text
AppSpecRequestV0 validada
  -> StartAppDirectorRequestV0
  -> StartAppDirectorV0
  -> resultado compacto para web/API/MCP
```

Invariantes:

- La superficie publica no conoce scheduler, outbox, runtime Codex, DB ni
  filesystem.
- El resultado puede exponer `director_tasks`, estado del loop, agentes
  arrancados y evidence refs compactas.
- La continuidad tras `wait_external` se resuelve por observadores y reentrada
  acotada del servicio, no por polling ad hoc desde web.

## Solicitar cambio sobre app existente

Contrato de entrada conceptual: `AppChangeRequestV0`.

Campos:

- `app_ref`: identificador opaco de la app existente;
- `change_ref`: identificador idempotente de la solicitud de cambio;
- `actor_ref`: identidad opaca del solicitante, si aplica;
- `locale`: etiqueta de idioma de la interaccion de usuario;
- `user_intent`: descripcion del cambio escrita por el usuario;
- `current_state_refs`: refs compactas a estado, specs, entregas o runs previos;
- `scope`: modulos, areas o capacidades que pueden cambiar;
- `acceptance_criteria`: criterios visibles y verificables del cambio;
- `constraints`: limites de seguridad, compatibilidad, datos y despliegue;
- `allowed_write_set`: ficheros o directorios permitidos para la ejecucion;
- `metadata_refs`: refs opacas adicionales para auditoria o trazabilidad.

Invariantes:

- `app_ref`, `change_ref` y `current_state_refs` no son rutas, DSN, IDs de DB ni
  nombres de proveedor.
- El contrato no hereda `AppSpecRequestV0`; crear app y cambiar app existente
  son intenciones distintas.
- Los textos visibles usan `locale`, pero enums, refs y DTOs internos permanecen
  estables y no localizados.
- Los criterios de aceptacion deben poder convertirse en pruebas, checks o
  evidencia compacta antes de cerrar el cambio.
- Ningun transporte puede ampliar el `allowed_write_set` despues de validar la
  solicitud.

Flujo por transportes:

```text
web / cambio de app existente
  -> cliente REST interno
  -> AppChangeRequestV0 validada
  -> servicio de director con puertos reales
  -> run de cambio y respuesta compacta

REST /api/v0/apps/{app_ref}/changes
  -> AppChangeRequestV0 validada
  -> servicio de director con puertos reales
  -> run de cambio y respuesta compacta

MCP app_change_request_v0
  -> AppChangeRequestV0 validada
  -> servicio de director con puertos reales
  -> run de cambio y respuesta compacta
```

Papel del director:

- reconstruye contexto solo desde puertos y `current_state_refs`;
- decide si falta informacion y puede emitir pregunta durable al usuario;
- abre fases y microtareas con linaje `change_ref`;
- compara entregas contra `acceptance_criteria`, write-set y estadisticas;
- rechaza o replanifica cuando hay tests fallidos, alcance incompleto,
  exceso de tamano o ausencia real de progreso;
- publica resultado compacto para web/API/MCP sin exponer prompts, rutas,
  credenciales, proveedor, modelo ni detalles de DB.

Reglas de replanificacion:

- replanificar crea nuevas decisiones y microtareas; no muta la solicitud
  original;
- cada nueva tarea conserva `app_ref`, `change_ref`, refs de evidencia y
  write-set acotado;
- el POST inicial solo arranca o registra el run de cambio; la continuidad se
  resuelve por `DrainRunV0` o worker equivalente;
- las reentradas consumen ACKs, estadisticas y decisiones persistidas por
  puertos hexagonales.

Estado implementado:

- `orquesta-app-change` valida `AppChangeRequestV0`;
- MCP expone `orquesta.apps.request_change.v0`;
- REST acepta `POST /api/v0/apps/change` y
  `POST /api/v0/apps/{app_ref}/changes`;
- web expone `/app-change`;
- `orquesta-app-codex-stack` registra el cambio como `AskDirector` durable en
  el workflow y deja outbox de director pendiente;
- `orquesta-app-change-director-source` lee cambios concretos por puerto y
  emite decisiones ejecutables para replanificar una microtarea de cambio.

Pendiente:

- cerrar validacion/revision especifica del cambio tras la entrega del agente.

Reglas de arquitectura:

- web, REST y MCP son transportes finos; no leen filesystem, runtime ni DB para
  inferir estado de la app;
- la persistencia concreta se inyecta por puertos: SQLite, Postgres,
  filesystem, broker o memoria son decisiones del operador;
- no hay DB, provider, modelo, idioma ni path hardcodeado en este modulo;
- i18n pertenece a los bordes visibles; el core recibe claves, locale o texto
  validado, no literales de UI mezclados con enums internos.

## ACK/progreso Codex

El stack reutiliza los contratos de `orquesta-runtime-codex-delivery`.

Reglas:

- cada agente real debe tener descriptor ACK registrable antes de `AgentStarted`;
- el ACK se observa por descriptor externo, no por escaneo desde el core;
- el progreso sin ACK se reduce a senales compactas `stalled`,
  `loop_detected` o `stopped`;
- la parada de proceso usa `process_ref + session_ref` registrado, no nombre ni
  PID visible;
- la verificacion de write-set se hace fuera del core y antes de aceptar el ACK.

## Review gate de programacion

Contrato de composicion:

```text
delivery registrada en programacion
  -> fase revision abierta por director
  -> ReviewGateSource inyectado
  -> RequestReview + RecordReviewResult
  -> AcceptReview solo si status=accepted
```

Puertos requeridos:

- `CodexReceiptDescriptorStorePortV0` para localizar el ACK correlado;
- `CodexReviewGateFileEvidenceResultProviderPortV0` para comprobar ficheros
  reales sin que el core conozca filesystem;
- politica de limite de lineas, si el operador quiere sobrescribir el default
  del core.

Invariantes:

- `BuildStackV0` falla si no se inyecta evidencia de review gate;
- el stack solo compone el source; no reimplementa scheduler ni workflow;
- una entrega aceptada genera `accepted_review_ref`;
- una entrega invalida queda como `changes_requested` y deja evidencia compacta;
- rework/replan posterior pertenece al director y a los puertos de workflow, no
  al adaptador de revision.

## Shutdown controlado de servidor

El stack publica el binding productivo del caso de uso
`orquesta-server-shutdown` para REST/MCP/CLI.

Contrato de composicion:

```text
POST /api/v0/server/shutdown
  -> tool MCP orquesta.server.shutdown.v0
  -> orquesta-server-shutdown.ShutdownServerV0
  -> RunQueueReaderPortV0 lista runs no terminales
  -> RunControlWriterPortV0 solicita stop
  -> RunGlobalSupervisorV0 drena stop/confirmaciones
  -> stats de director calculan readiness
```

Puertos usados:

- `RunQueueReaderPortV0` desde el store de cola;
- `RunControlReaderPortV0` y `RunControlWriterPortV0` desde el store de control;
- `RunControlCheckpointWriterPortV0` desde el store de control para dejar ACK
  durable cuando el shutdown no es forzado;
- `PrepareAgentShutdownPortV0` implementado por el stack: si no hay agentes en
  vuelo registra checkpoint conservador; si los hay, pide ACK cooperativo por
  runtime dir de cada agente Codex;
- `RunGlobalSupervisorV0` del propio stack como supervisor hexagonal;
- stats de shutdown calculadas desde `RunStore`, telemetria, progreso y usage
  inyectados en `StackConfigV0`.

Invariantes:

- el stack no mata procesos del servidor;
- el stack no lee DB, runtime, HOME, OAuth, proveedor ni modelo fuera de los
  puertos ya inyectados;
- `forced=false` devuelve `waiting_checkpoint` mientras existan agentes en
  vuelo o no haya ACK durable de checkpoint;
- `forced=true` drena por supervisor y exige stats sin agentes en vuelo antes
  de que CLI pueda enviar la senal final al servidor;
- la decision de cerrar el proceso servidor pertenece al borde operativo
  (`cmd/orquesta-server` o futuro runtime de servidor), no al caso de uso.
