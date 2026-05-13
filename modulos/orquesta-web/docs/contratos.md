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
- Status no 2xx distinto de 400 y errores de transporte devuelven `WebNuevaAppClientErrorV0` con codigo publico estable `error_transporte`, sin stack ni cuerpo privado.

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
- session_id
- request_id
- locale
- estado: inicial | esperando_agente | preguntando | requiere_datos | listo_para_validar | error
- idea_inicial
- app_spec_request_parcial
- preguntas_pendientes
- decisiones_tomadas
- alternativas_descartadas
- agente_intake_ref
Invariantes:
- La sesion conversa y acumula decisiones; no reemplaza a `AppSpecRequestV0`.
- La web no decide arquitectura ni aplica reglas de negocio de factory.
- El agente de intake puede preguntar al usuario cuando falten datos o haya ambiguedad.
- La UI muestra estado y preguntas; no obliga a rellenar todos los campos a mano.
- No toca DB, runtime, filesystem ni proveedor de agente concreto.
Errores:
- intake_session_invalida
- intake_agente_no_arrancado
- intake_requiere_respuesta_usuario
Pruebas de contrato:
- Crear sesion desde nombre/idea inicial.
- Registrar pregunta pendiente y respuesta del usuario.
- Proyectar AppSpecRequestV0 parcial sin validarlo como definitivo.
- Marcar listo para validar solo cuando no queden preguntas abiertas criticas.
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
- Defaults locales: `schema_version=app_spec_request.v0`, `source=orquesta-web`, `request_kind=crear_app_completa`, `execution_mode=normal`, `i18n.default_locale=locale` si falta y `preferencias_tecnicas.arquitectura=hexagonal` si falta.
- No valida reglas de negocio; esa validacion queda en `orquesta-factory`.
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
- El formulario expone request_id, locale, nombre, objetivo, descripcion, tipo_app, plataformas, arquitectura, i18n, datos, deploy, calidad, agentes e integracion basica sin decidir negocio.
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
