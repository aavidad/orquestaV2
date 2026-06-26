# Contratos locales: orquesta-web

Registra puertos, DTOs y eventos que `orquesta-web` expone o consume.

## Contrato de salida: `SolicitarNuevaAppClient`

Tipo: puerto_salida
Version: `v0`
Propietario: `orquesta-web`
Contrato externo consumido: `SolicitarNuevaApp v0`
Primer conector: REST/API
Transporte REST canonico: definido en `../../CONTRATOS.md` como `POST /api/v0/apps/spec`.

Responsabilidad:

- Enviar `AppSpecRequestV0` desde la web sin que la UI conozca el transporte.
- Recibir `AppSpecV0` y `BacklogInicialPropuestoV0`.
- Mapear errores publicos a estado renderizable.

Invariantes:

- La UI y los handlers de pantalla dependen del puerto, no del cliente REST concreto.
- El adaptador REST no contiene reglas de negocio.
- No toca DB, runtime, filesystem ni internals de factory.
- Propaga `request_id`, `correlation_id` y `locale`.
- Debe soportar timeout y error de transporte.

Errores publicos:

- `app_spec_invalida`
- `opcion_incompatible`
- `target_no_soportado`
- `idioma_invalido`
- `conector_requerido_no_disponible`
- `error_transporte`
- `respuesta_invalida`

Pruebas de contrato:

- Fake de `SolicitarNuevaAppClient` para render y submit.
- Adaptador REST serializa `AppSpecRequestV0` y deserializa respuesta sin campos internos.
- Error REST se transforma en `error_transporte`.
- Errores de factory se renderizan sin reinterpretar reglas en web.

Implementacion actual:

- `nueva_app_client_v0.go` define el puerto local `SolicitarNuevaAppClientV0`.
- `RESTSolicitarNuevaAppClientV0` usa `net/http` estandar contra `POST /api/v0/apps/spec`.
- Envia `AppSpecRequestV0` derivado de `WebNuevaAppFormV0`, propaga `request_id` o crea uno local `req-web-*` si falta, y lo usa como `X-Correlation-ID`.
- El timeout es configurable en el cliente REST y se aplica tambien al `context.Context` de la peticion.
- La respuesta 2xx canonica es `app_spec` + `backlog`, segun `../../CONTRATOS.md`, y devuelve `WebNuevaAppViewModelV0` compacto.
- Los alias `spec` y `backlog_inicial_propuesto` solo son compatibilidad transitoria durante el arranque; no son contrato canonico.
- La respuesta 400 canonica usa `errores` con issues publicos de factory y devuelve `WebNuevaAppViewModelV0` en estado `invalida`.
- Status no 2xx sin cuerpo publico estructurado y errores de transporte
  devuelven `WebNuevaAppClientErrorV0` con codigo publico estable
  `error_transporte`, sin stack ni cuerpo privado.
- Status no 2xx con `estado=error` y `errores_publicos` se proyecta como error
  publico del Director. Esto permite mostrar `codex_app_server_*` y
  `run_ref_requerido` sin reactivar el loop legacy cuando Goal-first esta
  configurado pero degradado.

## Contrato de salida: `ArrancarDirectorAppClient`

Tipo: puerto_salida
Version: `v0`
Propietario: `orquesta-web`
Contrato externo consumido: `orquesta.apps.arrancar_director.v0`
Primer conector: REST/API
Transporte REST canonico local: `POST /api/v0/apps/director`.

Responsabilidad:

- Enviar `AppSpecRequestV0` desde el wizard/formulario al bridge REST del tool MCP.
- Recibir `run_ref`, fase, tareas del director y agentes arrancados.
- Proyectar el resultado como bloque `director` dentro de `WebNuevaAppViewModelV0`.

Invariantes:

- La web no crea stores, runtimes, agentes ni scheduler.
- El endpoint web usa este puerto solo si esta inyectado; si no, conserva `SolicitarNuevaAppClientV0`.
- No expone procesos, sesiones, HOME, OAuth, proveedor, modelo, prompts ni transcript.
- Los errores publicos del bridge se renderizan como estado `invalida`; errores de transporte quedan como `error_transporte`.

Pruebas de contrato:

- Cliente REST serializa envelope con `app_spec_request` y limites opcionales.
- Endpoint POST delega en director cuando el puerto existe y no llama al cliente de spec.
- Prueba vertical web -> REST -> MCP -> `StartAppDirectorV0` con puertos fake.

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
Nombre: WebAppChangeFormV0.external_work
Tipo: dto
Version: v0
Propietario: orquesta-web
Consumidores: MCP/REST `orquesta.apps.request_change.v0`
Campos:
- external_project_ref
- external_interface_refs
- external_work_kind
- external_work_refs
Invariantes:
- Solo transporta refs opacas compactas hacia `AppChangeRequestV0`.
- La web no interpreta el dominio externo ni decide agentes, modelo, DB,
  runtime, filesystem, proveedor ni rutas internas de la app propietaria.
- El campo es opcional; si no hay datos, no se materializa `external_work`.
Errores:
- Los errores de validacion pertenecen a `orquesta-app-change`.
Pruebas de contrato:
- Mapper de form a MCP conserva refs externas.
- POST form-urlencoded preserva refs externas hacia el puerto local.
```

```text
Nombre: WebRunQueuePanelV0
Tipo: puerto_entrada + dto
Version: v0
Propietario: orquesta-web
Consumidores: operador humano / navegador en `/run-queue`
Contrato externo consumido: `orquesta.run_queue.priority.v0`
Campos:
- query: action, queue_ref, app_refs, run_ref, app_ref, priority_score, limit.
- view_model: estado, action, queue_ref, count, ranked, updated y errores.
- cada candidato `ranked` incluye `stats_href` hacia `/director-stats` para
  consultar agentes, modelo, cuota y progreso por run sin que la cola replique
  ese contrato.
Invariantes:
- La web llama al bridge REST `/api/v0/runs/queue/priority`; no lee stores,
  runtime, DB, scheduler ni cola concreta.
- `rank` muestra candidatos ya saneados por MCP y `set_priority` delega la
  mutacion al puerto `RunQueuePriorityWriterPortV0`.
- La navegacion a stats usa solo `run_ref` opaca y `include_agent_progress`;
  no expone runtime ni stores.
- Solo expone refs compactas de run/app/evidencia; no procesos, HOME, OAuth,
  proveedor, modelo, DSN ni detalles de almacenamiento.
Errores:
- run_queue_error_transporte
- run_queue_respuesta_invalida
- errores publicos de `orquesta.run_queue.priority.v0`
Pruebas de contrato:
- Cliente REST serializa rank/set_priority hacia el bridge MCP REST.
- Endpoint GET `/run-queue` consulta ranking y prepara refresh compacto.
- Endpoint POST `/run-queue` delega cambio de prioridad.
```

```text
Nombre: WebRunControlPanelV0
Tipo: puerto_entrada + dto
Version: v0
Propietario: orquesta-web
Consumidores: operador humano / navegador en `/run-control`
Contrato externo consumido: `orquesta.runs.control.v0`
Campos:
- command: action, run_ref, requested_by, reason, forced, evidence_refs.
- view_model: estado, action, run_ref, status, checkpoint_recorded, forced,
  evidence_refs y errores.
Invariantes:
- La web llama al bridge REST `/api/v0/runs/control`; no lee RunControl store,
  runtime, procesos, scheduler, DB ni filesystem.
- `pause`, `resume`, `stop` y `cancel` solo se aceptan por POST para evitar
  mutaciones por polling o enlace GET.
- El checkpoint cooperativo y la parada fisica pertenecen a puertos de
  RunControl/shutdown/runtime, no al panel web.
Errores:
- run_control_error_transporte
- run_control_respuesta_invalida
- errores publicos de `orquesta.runs.control.v0`
Pruebas de contrato:
- Cliente REST serializa pause/resume/stop/cancel hacia el bridge MCP REST.
- Endpoint POST `/run-control` delega accion por cliente inyectado.
- Gateway app monta `/run-control` encima de `/api/v0/runs/control`.
```

```text
Nombre: WebAutoprogrammingPrepareRunV0
Tipo: puerto_salida + dto
Version: v0
Propietario: orquesta-web
Consumidores: operador humano / cliente web de autoprogramacion
Contrato externo consumido: `orquesta.autoprogramming.prepare_run.v0`
Campos:
- command: request_id, correlation_id, request_ref, project_ref, worktree_ref,
  branch_ref, tasks, write_set, required_tests, priority_score y limites de
  continuacion.
- view_model: estado, accepted, run_ref, project_ref, worktree_ref,
  branch_ref, phase_id, workflow_task_refs, wait_agent_refs, goal_specs
  opcional cuando el prepare-run ya trae contratos Goal, goal opcional cuando
  la composicion lo lanza, continue y errores_publicos.
Invariantes:
- La web llama al bridge REST `/api/v0/autoprogramming/prepare-run`; no lee
  stores, runtime, procesos, Git, DB ni filesystem.
- `worktree_isolated=true` se fija en el envelope enviado al contrato externo;
  `worktree_ref` y `branch_ref` se transportan como refs opacas y no se
  interpretan como rutas, nombres Git ni comandos.
- Por defecto la pantalla marca la tarea con `goal_migration:goal-first` y las
  capacidades `goal_capability:*`; si el operador desactiva Goal, conserva la
  ruta legacy acotada.
- La preparacion legacy deja una run continuable; la supervision posterior usa
  `run_ref` y el endpoint acotado `/api/v0/autoprogramming/supervise`.
- Si el contrato externo devuelve `goal`, la web observa cierre por
  `/api/v0/autoprogramming/goal/observe` con polling acotado. Si solo devuelve
  `goal_specs[]`, lo proyecta como handoff sin empujar a supervisor legacy.
- `priority_score` se transporta al contrato prepare-run para que la cola
  mantenga el trabajo secundario acotado sin bloquear trabajo primario.
Errores:
- autoprogramming_prepare_run_error_transporte
- autoprogramming_prepare_run_respuesta_invalida
- errores publicos de `orquesta.autoprogramming.prepare_run.v0`
Pruebas de contrato:
- Cliente REST serializa la peticion preservando worktree aislada, rama opaca,
  write-set y tests obligatorios.
- Respuestas `error` del bridge HTTP se proyectan como errores publicos
  renderizables.
```

## Contratos consumidos

```text
Nombre: SolicitarNuevaApp
Tipo: puerto_salida
Version: v0
Propietario: orquesta-factory
Consumidores: orquesta-web
Campos:
- Entrada enviada por web: AppSpecRequestV0.
- Salida correcta consumida por web: AppSpecV0.
- Salida complementaria consumida por web: BacklogInicialPropuestoV0.
Invariantes:
- La web actua como adaptador fino: no valida reglas de negocio que pertenezcan a factory.
- La web no persiste estado, no crea tareas directas en DB y no arranca runtime.
- La web no pasa detalles de DB, filesystem, runtime, LLM, cache, cola, storage ni deploy como internals; solo como conectores/restricciones documentadas.
- La web debe preservar `request_id`, `source=orquesta-web`, `locale` y correlation id si el transporte lo soporta.
Errores:
- app_spec_invalida
- opcion_incompatible
- target_no_soportado
- idioma_invalido
- conector_requerido_no_disponible
- error_transporte_web (local, no dominio factory)
Pruebas de contrato:
- Fake de factory recibe `AppSpecRequestV0` sin campos internos.
- Errores publicos se renderizan sin ocultar bloqueo/cuota/validacion.
- Respuesta valida se convierte a proyeccion compacta sin materializar backlog.
```

```text
Nombre: AppSpecRequestV0
Tipo: dto
Version: v0
Propietario: orquesta-factory
Consumidores: orquesta-web
Campos:
- Los campos canonicos estan en `orquesta-factory/docs/contratos.md`.
- La web solo construye este DTO desde `WebNuevaAppFormV0`.
Invariantes:
- `schema_version`, `request_id`, `source`, `locale`, `nombre`, `objetivo` y `tipo_app` deben llegar al puerto.
- `source` siempre es `orquesta-web`.
- `locale` sale de la preferencia de UI/request, no de una constante oculta.
- `datos.db_required=true` expresa necesidad funcional; la web no envia tablas, SQL ni proveedor como decision cerrada de dominio.
Errores:
- app_spec_invalida
- idioma_invalido
- opcion_incompatible
Pruebas de contrato:
- Mapper de form a request con caso minimo.
- Mapper con datos persistentes mantiene DB como conector/restriccion.
- Mapper con i18n desactivado exige justificacion si el formulario permite esa opcion.
```

## Contratos locales expuestos por orquesta-web

```text
Nombre: WebNuevaAppIntakeSessionV0
Tipo: dto
Version: v0
Propietario: orquesta-web
Consumidores: vistas/handlers de nueva app, futuro API/MCP de intake
Campos:
- schema_version: `web_nueva_app_intake_session.v0`
- session_id
- session_ref
- locale
- estado: `requiere_datos | lista_para_solicitar`
- form: `WebNuevaAppFormV0`
- sections: indice compacto por seccion, campos requeridos, capturados y pendientes
- field_index: mapa estable campo -> seccion
- questions: preguntas pendientes como ids de campo y claves i18n
- pending_questions: ids de campos criticos pendientes
- decisions: respuestas capturadas como ids de campo y valores compactos
- app_spec_partial: `AppSpecRequestV0` derivado del formulario parcial
- handoff: refs opacas y contexto compacto para `orquesta.apps.arrancar_director.v0`
Invariantes:
- La sesion conversa y acumula decisiones; no reemplaza a `AppSpecRequestV0`.
- La web no decide arquitectura ni aplica reglas de negocio de factory.
- Las preguntas son ids de campo/seccion y claves i18n; la capa UI debe
  localizar textos por catalogo.
- La UI muestra estado e indice de secciones; no obliga a rellenar todos los campos a mano.
- Si la sesion queda lista, el handoff entrega solo refs de sesion/request,
  contexto compacto y la ruta preferente al Director; si el puerto falta, el
  fallback documentado sigue siendo `SolicitarNuevaApp v0`.
- No toca DB, runtime, filesystem ni proveedor de agente concreto.
Errores:
- No define errores propios en este corte; los errores publicos siguen perteneciendo al cliente/handler que consuma la sesion.
Pruebas de contrato:
- Crear sesion desde nombre/idea inicial.
- Registrar pregunta pendiente, respuesta del usuario e indice campo/seccion.
- Proyectar AppSpecRequestV0 parcial sin validarlo como definitivo.
- Marcar `lista_para_solicitar` solo cuando no queden preguntas criticas.
- Exponer handoff compacto con refs opacas y fallback declarado.
```

```text
Nombre: WebNuevaAppIntakeGuidedTurnV0
Tipo: dto
Version: v0
Propietario: orquesta-web
Consumidores: vistas/handlers de nueva app, futuro API/MCP de intake
Campos:
- schema_version: `web_nueva_app_intake_guided_turn.v0`
- need: necesidad libre del operador.
- decisions: decisiones sugeridas como `WebNuevaAppIntakeDecisionV0`.
- followups: acciones guiadas disponibles, con `id`, `label_key` y decisiones
  asociadas.
- messages: claves i18n de mensajes de estado para la UI.
Invariantes:
- No sustituye `WebNuevaAppIntakeSessionV0`; solo propone decisiones aplicables
  a una sesion.
- No valida reglas de negocio ni proveedores; el cierre sigue en factory o
  Director.
- Las acciones de mapas, datos, arquitectura y accesibilidad se expresan como
  capacidades, preferencias y restricciones publicas, no como tokens, DSN,
  backend ni runtime.
- El caso de alquileres cercanos produce datos/storage/integraciones editables
  y compatibles con `AppSpecRequestV0`.
Pruebas de contrato:
- Necesidad libre de app movil de alquileres cercanos genera nombre, objetivo,
  `tipo_app=mobile`, plataforma mobile, datos detallados, storage e integracion
  de mapas validable por factory.
- Acciones de seguimiento actualizan arquitectura, calidad, accesibilidad y
  datos externos sin cambiar el contrato final.
```

```text
Nombre: NuevaAppIntakeGuidedHTTPHandlerV0
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-web
Ruta prevista por gateway: POST /api/v0/apps/intake/guided-turn
Request:
- schema_version: `web_nueva_app_intake_guided_request.v0` opcional.
- session_id, locale, nombre, idea: contexto inicial opcional.
- need: necesidad libre del operador.
- action_id/action_ids: acciones guiadas a aplicar.
- session: `WebNuevaAppIntakeSessionV0` opcional para continuar una sesion.
Response:
- schema_version: `web_nueva_app_intake_guided_response.v0`.
- turn: `WebNuevaAppIntakeGuidedTurnV0`.
- session: `WebNuevaAppIntakeSessionV0` resultante.
Invariantes:
- Es calculo puro de intake; no persiste sesion, no arranca Director, no llama
  LLM/MCP, no abre filesystem y no toca runtime.
- El handler acepta solo JSON y mantiene limites de body/control plane comunes.
- Las respuestas devuelven formulario parcial editable; factory y Director
  siguen cerrando validacion/ejecucion.
- Las acciones guiadas no imponen proveedor de mapas, DB, cloud ni arquitectura
  obligatoria; `hexagonal` sigue siendo fallback, no imposicion.
Pruebas de contrato:
- POST JSON con necesidad de app movil de alquileres cercanos devuelve sesion
  con datos, storage e integracion de mapas.
- POST JSON con accion aplica la decision sobre una sesion existente.
- Metodo no soportado y content-type no JSON producen errores HTTP publicos.
```

```text
Nombre: NuevaAppI18nCatalogV0
Tipo: dto
Version: v0
Propietario: orquesta-web
Consumidores: vistas/handlers futuros de nueva app
Campos:
- locale soportado: es-ES, en-US
- default_locale: es-ES
- keys requeridas: titulo, nombre, objetivo, tipo_app, locale, submit, estados, backlog preview, fases y errores publicos.
Invariantes:
- Todo texto visible futuro del flujo nueva app debe salir de claves i18n estables.
- El catalogo es puro en memoria; no usa UI, templates, HTTP, DB, runtime ni filesystem productivo.
- Locale desconocido cae al default controlado.
- Clave inexistente devuelve error publico localizado o fallback controlado sin panic.
- No contiene reglas de negocio ni reinterpreta errores de factory.
Errores:
- nueva_app_i18n_clave_no_encontrada
- nueva_app_i18n_catalogo_incompleto
Pruebas de contrato:
- ES/EN cubren todas las claves requeridas.
- Lookup por locale, alias y fallback.
- Locale desconocido cae al default.
- Clave inexistente devuelve error publico sin panic.
Implementacion actual:
- `nueva_app_i18n_v0.go` define `NuevaAppI18nCatalogV0`, `NuevaAppI18nRequiredKeysV0()`, `Lookup`, `ValidateRequired`, `SupportedLocales` y helper `NuevaAppI18nTextV0`.
- Los catalogos minimos `es-ES` y `en-US` cubren campos/acciones/estados principales de nueva app, backlog preview, fases y errores publicos.
```

```text
Nombre: WebNuevaAppFormV0
Tipo: dto
Version: v0
Propietario: orquesta-web
Consumidores: orquesta-web
Campos:
- request_id
- locale
- request_kind
- execution_mode
- nombre
- objetivo
- descripcion
- tipo_app
- project_source
  - kind
  - git_url
  - branch
  - local_path
  - project_ref
- usuarios_objetivo
- plataformas
- integraciones
- preferencias_tecnicas
- datos
- deploy
- calidad
- documentacion
- i18n
- agentes
- restricciones
Invariantes:
- DTO de captura, no contrato de dominio.
- No contiene IDs de DB obligatorios ni asume proyecto existente.
- No contiene reglas de recomendacion; solo elecciones, omisiones explicitas y texto libre del operador.
- Todo texto visible asociado debe tener clave i18n.
- `project_source` solo transporta origen declarado por el operador hacia
  factory/director; web no valida si una ruta, repo o ref existe.
Errores:
- form_incompleto
- locale_no_soportado_web
- transporte_no_configurado
Pruebas de contrato:
- Serializacion estable del form local.
- Mapeo 1:1 o agrupado hacia `AppSpecRequestV0`.
- Ausencia de campos `db_table`, `sql`, `runtime_provider`, `agent_id` obligatorio o rutas internas.
Implementacion actual:
- `nueva_app_form_v0.go` define el DTO local y `ToAppSpecRequestV0`.
- El mapper importa factory como `orquestafactory "orquesta/modulos/orquesta-factory"`.
- Defaults locales: `schema_version=app_spec_request.v0`, `source=orquesta-web`, `request_kind=crear_app_completa`, `execution_mode=normal`, `i18n.default_locale=locale` si falta y `preferencias_tecnicas.arquitectura=hexagonal` como fallback si falta.
- No valida reglas de negocio; esa validacion queda en `orquesta-factory`.
- `project_source.kind` usa el mismo contrato que factory: `new`, `github` o
  `local_path`.
- `preferencias_tecnicas.arquitectura` es una preferencia/contrato de patron,
  no proveedor. Las opciones soportadas incluyen `hexagonal`,
  `clean_architecture`, `onion`, `modular_monolith`, `layered`,
  `event_driven`, `microservices`, `serverless`, `plugin_based` y
  `data_pipeline`; todas deben conservar fronteras limpias entre dominio,
  aplicacion, puertos, adaptadores y bootstrap. `hexagonal` queda como fallback
  si el operador no elige patron, pero no es obligatorio.
- Las opciones expertas de datos, almacenamiento e integraciones deben entrar
  como necesidades, tipos, sensibilidad, retencion, integraciones y
  restricciones publicas; no como tablas, SQL, DSN, proveedor ni runtime.
- `calidad.accesibilidad` transporta el nivel de accesibilidad elegido por el
  operador. El HTML expone `basica`, `normal`, `wcag_aa` y `no_aplica`;
  `calidad.accesibilidad_opciones` permite declarar niveles aceptados por
  runtime/adaptador, por ejemplo `normal,wcag_aa`, sin imponer proveedor.
```

```text
Nombre: WebNuevaAppViewModelV0
Tipo: dto
Version: v0
Propietario: orquesta-web
Consumidores: vistas/handlers de orquesta-web
Campos:
- request_id
- locale
- estado: inicial | enviando | valida | requiere_datos | invalida | error
- resumen_app
- backlog_preview
- defaults_aplicados
- warnings
- errores_publicos
- preguntas_abiertas
- fases
- microtareas
- contratos_requeridos
- riesgos
Invariantes:
- Proyeccion compacta para operador; no dump de `AppSpecV0` completo.
- Muestra bloqueos, cuota o validacion si aparecen en errores/warnings publicos.
- No muestra secretos, transcript completo, structs privados ni detalles de proveedor.
- No permite accion destructiva ni arranque de runtime desde la respuesta v0.
Errores:
- view_model_incompleto
Pruebas de contrato:
- Render de spec valida con backlog propuesto.
- Render de spec con preguntas abiertas.
- Render de errores publicos sin perder locale.
Implementacion actual:
- `nueva_app_viewmodel_v0.go` define proyecciones compactas de resumen, defaults, warnings, errores publicos, fases, microtareas, contratos, riesgos y preguntas.
- Consume solo DTOs publicos de factory: `AppSpecV0`, `BacklogInicialPropuestoV0` y `ValidationIssue`.
- Representa estados `valida`, `requiere_datos`, `invalida`, `inicial` y `error` sin exponer dumps internos.
- Incluye `backlog_preview` como proyeccion compacta de `BacklogInicialPropuestoV0` para JSON/handler: fases con key/titulo/orden/conteo y microtareas con key, fase, titulo, modulo/frontera, write-set previsto, bloqueos y criterio de cierre.
```

```text
Nombre: WebNuevaAppBacklogPreviewV0
Tipo: dto
Version: v0
Propietario: orquesta-web
Consumidores: vistas/handlers de orquesta-web
Campos:
- schema_version: nueva_app_backlog_preview.v0
- fases: key, titulo, objetivo, orden, microtareas
- microtareas: key, fase, titulo, modulo_frontera, write_set_previsto, bloqueos, criterio_cierre
- total_fases
- total_microtareas
- tiene_bloqueos
Invariantes:
- Deriva solo de `BacklogInicialPropuestoV0`; no materializa backlog ni crea tareas.
- Es preview compacto para operador, no dump de `FaseInicialV0` ni `MicrotareaPropuestaV0`.
- No expone structs internos, runtime, DB, filesystem, agentes asignados ni proveedor.
- Conserva bloqueos y criterio de cierre verificable.
Errores:
- view_model_incompleto
Pruebas de contrato:
- Snapshot controlado de preview compacto con key, fase, titulo, modulo/frontera, write-set, bloqueo y criterio de cierre.
- `httptest` del endpoint JSON conserva textos i18n y semantica basica del preview.
Implementacion actual:
- `nueva_app_viewmodel_v0.go` define `WebNuevaAppBacklogPreviewV0` dentro de `WebNuevaAppViewModelV0`.
- `nueva_app_endpoint_v0.go` expone textos i18n de backlog preview en el JSON de pagina sin templates ni frontend pesado.
```

```text
Nombre: NuevaAppWebEndpointV0
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-web
Consumidores: operador humano / navegador
Campos:
- GET: locale opcional, estado inicial.
- POST: `WebNuevaAppFormV0` codificado por el formulario o JSON segun tecnologia elegida.
- Respuesta: `WebNuevaAppViewModelV0` renderizado.
Invariantes:
- Endpoint web inbound; no es contrato entre modulos.
- No accede a DB directa.
- No invoca runtime.
- No llama endpoints heredados de fabricacion/materializacion.
Errores:
- metodo_no_soportado
- form_incompleto
- error_transporte_web
- errores publicos de `SolicitarNuevaApp v0`
Pruebas de contrato:
- GET renderiza estado inicial localizado.
- POST con fake de factory renderiza respuesta valida.
- POST con fake de error renderiza bloqueo/validacion sin ocultarlo.
Implementacion actual:
- `nueva_app_endpoint_v0.go` define `NuevaAppWebEndpointV0` como `net/http` handler puro.
- `GET` devuelve JSON estructurado `NuevaAppWebPageV0` con estado inicial, textos i18n, opciones de locale y campos derivados por reflexion de `WebNuevaAppFormV0`.
- `POST` acepta JSON `WebNuevaAppFormV0` y formulario `application/x-www-form-urlencoded`, delega en `SolicitarNuevaAppClientV0` y renderiza el `WebNuevaAppViewModelV0` devuelto.
- Los errores locales de metodo, parseo, transporte y cliente no exponen cuerpos privados; se renderizan como errores publicos localizados.
- No arranca servidor real, no usa templates, DB, runtime, filesystem productivo ni MCP.
```

```text
Nombre: NuevaAppHTMLHandlerV0
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-web
Consumidores: operador humano / navegador en `/nueva-app`
Campos:
- GET: formulario HTML localizado para `WebNuevaAppFormV0`.
- POST: `application/x-www-form-urlencoded` con los nombres de campos del mapper local.
- Respuesta: HTML renderizado desde `NuevaAppWebPageV0` y `WebNuevaAppViewModelV0`.
Invariantes:
- Es adaptador HTML fino sobre `NuevaAppWebEndpointV0`/`NuevaAppWebPageV0`.
- Reutiliza `SolicitarNuevaAppClientV0`; no accede a DB, runtime, provider/model, HOME, OAuth ni endpoints heredados.
- El formulario expone asistente guiado, request_id, locale, nombre, objetivo,
  descripcion, tipo_app, plataformas, arquitectura, i18n, datos, almacenamiento,
  deploy, calidad, agentes e integraciones multiples sin decidir negocio.
- La guia documental de opciones del wizard vive en
  `docs/guia_nueva_app_opciones_2026-06-25.md` y distingue asistente guiado,
  modo basico y modo experto opcional para datos, almacenamiento, accesibilidad
  e integraciones multiples.
- Renderiza estado, resumen, backlog preview y errores publicos sin materializar backlog.
Errores:
- metodo_no_soportado
- form_incompleto
- transporte_no_configurado
- error_transporte
- errores publicos de `SolicitarNuevaApp v0`
Pruebas de contrato:
- GET con `httptest` renderiza formulario HTML usable y no delega.
- POST form-urlencoded valido delega en fake `SolicitarNuevaAppClientV0` y renderiza estado/resumen/backlog.
- POST invalido renderiza error publico localizado sin ocultarlo.
Implementacion actual:
- `nueva_app_html_handler_v0.go` define `NuevaAppHTMLHandlerV0` como handler `net/http` puro.
- `nueva_app_html_render_v0.go` usa `html/template` en memoria, sin filesystem productivo.
- El submit compartido vive en `nueva_app_endpoint_post_page_v0.go` para mantener JSON y HTML sobre la misma delegacion.
```

```text
Nombre: NuevaAppGoalFirstPanelV0
Tipo: ui/html
Version: v0
Propietario: orquesta-web
Consumidores: operadores en `/nueva-app`
Campos visibles:
- run_ref obligatorio en respuestas `ok` del arranque; el panel lo muestra si
  esta presente
- goal_ref
- external_goal_ref
- goal_status
- director_execution_mode
- run_status y closure_status tras refresco
Invariantes:
- Se renderiza solo si `WebNuevaAppViewModelV0.Director` esta presente.
- El boton `Actualizar goal` y el polling automatico aparecen solo si hay
  `run_ref` y `goal_ref`; un resultado legacy con `run_ref` no intenta observar
  goal.
- El refresco llama por `fetch` a `POST /api/v0/apps/director/goal/observe`.
- Si hay `run_ref`, el panel observa automaticamente de forma acotada:
  `data-goal-poll-interval-ms=5000` y `data-goal-max-polls=60`.
- La observacion automatica se detiene al ver run `cerrada`/`bloqueada` o goal
  `complete`/`blocked`/`invalid`; el boton manual sigue disponible.
- No valida cierre, no toca cola, no arranca runtime ni crea stores.
- Si un viewmodel defensivo llega sin `run_ref`, muestra refs de goal pero no
  fuerza una observacion imposible; la frontera REST/MCP ya debe rechazar ese
  `ok` como `run_ref_requerido`.
Pruebas de contrato:
- `TestNuevaAppHTMLV0RenderizaPanelGoalFirstConActualizacion`.
- `TestNuevaAppHTMLV0DefensivoGoalFirstSinRunNoObserva`.
```

```text
Nombre: WebAutoprogrammingStatusV0
Tipo: cliente_salida/dto
Version: v0
Propietario: orquesta-web
Consumidores: vistas web de autoprogramacion y operador
Campos:
- query: request_id, correlation_id, locale, run_ref?, app_ref?,
  external_job_ref?, queue_ref?, app_refs?, queue_limit? y flags de telemetria.
- viewmodel: queue_live, run_live, queue_ref, run_ref, runs, agents,
  ops_snapshot, diagnostics y errores_publicos.
Invariantes:
- Consume `POST /api/v0/autoprogramming/status`.
- Delega estado de cola/run en `orquesta.autoprogramming.status.v0`.
- Pide progreso de agentes por defecto cuando no se explicitan flags.
- Consume `ops_snapshot` si el servidor lo publica; no reconstruye la decision
  del Director salvo fallback legacy de compatibilidad.
- Proyecta refs opacas; no convierte worktree_ref/branch_ref en rutas ni ramas.
- No lee stores, DB, runtime, filesystem, Codex, OPES ni proveedor.
Errores:
- autoprogramming_status_error_transporte
- autoprogramming_status_respuesta_invalida
- errores publicos del contrato MCP status
Pruebas de contrato:
- `go test -count=1 ./modulos/orquesta-web` compila cliente y proyeccion junto
  al contrato prepare-run existente.
Implementacion actual:
- `autoprogramming_prepare_run_client_v0.go` expone el cliente REST fino de
  prepare-run y status reutilizando el transporte HTTP local.
- `autoprogramming_prepare_run_types_v0.go` define query y proyeccion compacta
  de cola, runs en progreso, agentes, diagnosticos y errores publicos.
```

```text
Nombre: WebOperationalStatusPanelV0
Tipo: dto
Version: v0
Propietario: orquesta-web
Consumidores: vistas/handlers futuros de estado operativo web
Campos:
- schema_version: web_operational_status_panel.v0
- locale
- estado y estado_visual
- scope, subject_ref, correlation_id, diagnostic_id y projection_ref
- fase_actual
- progreso compacto
- fases: key, estado, actual y progress
- bloqueos, salud, actividad_reciente, warnings y frescura
- tiene_bloqueos
- privacy_ok
Invariantes:
- Deriva solo de `DiagnosticoCompactoV0`; no consulta DB, runtime, event sink, filesystem, bus ni procesos.
- Es proyeccion compacta para operador, no dump completo del DTO de observability.
- No expone flags internos de privacidad, transcripts, prompts, completions, SQL, DSN, proveedor, modelo ni detalles de conexion.
- No prescribe acciones, no reintenta tareas, no asigna agentes y no recupera runtime.
- Estados futuros o desconocidos se degradan a `unknown` sin inventar fases ni bloqueos.
Errores:
- No define errores propios en este corte; los errores publicos siguen perteneciendo a `OperationalStatusQueryV0`.
Pruebas de contrato:
- `TestWebOperationalStatusQueryV0UsaContratoPublicoReadOnly` valida que la query local cumple el contrato publico con consumidor `orquesta-web/web`.
- `TestWebOperationalStatusPanelV0CompactaDiagnosticoSinDumpInterno` valida estado, fase actual, progreso, bloqueos, salud, actividad, frescura y ausencia de dumps/prohibidos.
- `TestWebOperationalStatusPanelV0EstadoDesconocidoNoInventaFases` valida degradacion estable a `unknown`.
Implementacion actual:
- `operational_status_viewmodel_v0.go` define `NewWebOperationalStatusQueryV0` y `NewWebOperationalStatusPanelV0`.
- El builder de query pide secciones `estado`, `progreso`, `bloqueos`, `salud` y `actividad_reciente` con limite compacto por defecto.
- No hay cliente HTTP, endpoint web, servidor real, DB, runtime, filesystem productivo ni acciones de recuperacion.
```
