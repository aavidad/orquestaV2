# Contratos locales: orquesta-factory

Registra puertos, DTOs y eventos que `orquesta-factory` expone o consume.

## Propuesta v0: pedir una nueva app desde orquesta-web

Objetivo: permitir que `orquesta-web` envie una peticion estructurada de nueva
app y reciba una especificacion versionada, validable y suficiente para generar
un backlog inicial. Este contrato no ejecuta agentes, no crea tareas en DB y no
arranca runtime.

### Nombre: SolicitarNuevaApp

Tipo: puerto_entrada
Version: `v0`
Propietario: `orquesta-factory`
Consumidores: `orquesta-web`, `orquesta-mcp`, `orquesta-cli`

Campos:

- Entrada: `AppSpecRequestV0`
- Salida correcta: `AppSpecV0`
- Salida de propuesta: `BacklogInicialPropuestoV0`

Invariantes:

- El puerto solo valida, normaliza y propone; no persiste estado.
- La UI no crea tareas directas ni escribe en DB.
- El runtime queda fuera del flujo.
- Todo texto generado para usuario o documentacion debe declarar `locale`.
- Si falta informacion no inferible, la respuesta debe incluir preguntas o
  supuestos, no inventar contratos cerrados.

Errores:

- `app_spec_invalida`
- `opcion_incompatible`
- `target_no_soportado`
- `idioma_invalido`
- `conector_requerido_no_disponible`

Pruebas de contrato:

- Request minima valida produce `AppSpecV0` con defaults explicitos.
- Request sin `nombre` o sin `objetivo` devuelve `app_spec_invalida`.
- Request con DB requerida declara conector, no tablas ni proveedor concreto.
- Request con `i18n.enabled=false` requiere justificacion.
- Request con deploy no soportado devuelve `target_no_soportado`.

### Transporte REST v0

Implementacion actual: `orquesta/modulos/orquesta-factory-http`. Este modulo
mantiene el negocio y DTOs canonicos; el adaptador HTTP importa factory y no al
reves.

Ruta canonica: `POST /api/v0/apps/spec`

Request:

- Body JSON: `AppSpecRequestV0`.
- Headers: `Content-Type: application/json`, `Accept: application/json`,
  `X-Correlation-ID`.

Respuesta 2xx:

- Body JSON con `app_spec` (`AppSpecV0`) y `backlog`
  (`BacklogInicialPropuestoV0`).
- `X-Correlation-ID` replica el header entrante; si no existe, usa
  `request_id` cuando el body se puede decodificar.

Respuesta 400:

- Body JSON con `errores` (`[]ValidationIssue`) para JSON invalido,
  campos desconocidos o validacion de request/caso de uso/backlog.
- No replica el cuerpo privado de la request.

Metodo no permitido:

- Devuelve `405 Method Not Allowed`, header `Allow: POST` y body publico con
  `errores`.
- Usa `app_spec_invalida` como codigo estable local para mantener el contrato
  de errores v0 sin introducir un codigo global nuevo.

Invariantes del adaptador:

- No contiene reglas de negocio nuevas.
- Solo decodifica, llama `SolicitarNuevaAppV0`, genera
  `GenerarBacklogInicialPropuestoV0` y codifica el envelope.
- No arranca servidor real, router externo, DB, filesystem, runtime, MCP ni UI.

### Nombre: AppSpecRequestV0

Tipo: dto
Version: `v0`
Propietario: `orquesta-factory`
Consumidores: `orquesta-web`, `orquesta-mcp`, `orquesta-cli`

Campos:

- `schema_version`: valor fijo `app_spec_request.v0`.
- `request_id`: idempotencia de la peticion en el cliente.
- `source`: `orquesta-web`, `orquesta-mcp` u `orquesta-cli`.
- `locale`: locale principal de la peticion.
- `request_kind`: tipo de trabajo solicitado; default `crear_app_completa`.
- `execution_mode`: `normal` o `debug`; default `normal`.
- `project_source`: origen opcional del proyecto: `new`, `github` con
  `git_url` sin credenciales, o `local_path`.
- `nombre`: nombre humano de la app.
- `objetivo`: resultado que la app debe conseguir.
- `descripcion`: contexto libre y breve.
- `tipo_app`: `web`, `api`, `cli`, `desktop`, `mobile`, `automation`,
  `data`, `plugin`, `mixed`, `documentacion` o `documentation`.
- `usuarios_objetivo`: perfiles principales.
- `plataformas`: plataformas objetivo.
- `integraciones`: conectores externos esperados.
- `preferencias_tecnicas`: lenguaje, framework, restricciones y preferencias.
- `datos`: si requiere persistencia, tipo de datos y sensibilidad.
- `deploy`: target preferido o restricciones de despliegue.
- `calidad`: nivel de pruebas, accesibilidad, compliance y observabilidad.
- `documentacion`: docs de usuario, desarrollo y sistemas requeridos.
- `i18n`: activacion, locale por defecto y locales adicionales.
- `agentes`: preferencias de colaboracion, revision y autonomia.
- `restricciones`: limites de tiempo, proveedor, licencias o seguridad.

Invariantes:

- `schema_version`, `request_id`, `source`, `locale`, `nombre`, `objetivo` y
  `tipo_app` son obligatorios.
- `execution_mode=debug` es la unica via para recortar entregables minimos y no
  permite marcar cierre productivo sin declarar lo omitido.
- `project_source.kind` se normaliza a `new`, `github` o `local_path`; `git_url`
  y `local_path` son mutuamente excluyentes.
- Las peticiones sobre app existente deben declarar `project_source` con
  `github` o `local_path`.
- `project_source.git_url` no puede incluir credenciales.
- `locale` y `i18n.default_locale` usan BCP 47.
- `i18n.enabled` es `true` por defecto; si es `false`, debe haber
  `i18n.justificacion`.
- `preferencias_tecnicas.arquitectura` es `hexagonal` por defecto y cualquier
  valor no hexagonal es incompatible.
- La hexagonalidad no es advisory: una app generada debe separar domain,
  application, ports, adapters y bootstrap/composicion.
- `datos.db_required=true` obliga a declarar necesidad funcional, no proveedor.
- Las integraciones se expresan como conectores, no como llamadas directas a DB,
  filesystem, runtime, LLM, cache, cola ni deploy.

Errores:

- `app_spec_invalida`
- `opcion_incompatible`
- `idioma_invalido`

Pruebas de contrato:

- Validar campos obligatorios.
- Validar defaults de arquitectura hexagonal, i18n y documentacion.
- Rechazar proveedor DB directo como requisito de dominio.
- Rechazar integracion runtime como parte de la peticion.

### Nombre: AppSpecV0

Tipo: dto
Version: `v0`
Propietario: `orquesta-factory`
Consumidores: `orquesta-web`, `orquesta-core`, `orquesta-mcp`,
`orquesta-cli`

Campos:

- `schema_version`: valor fijo `app_spec.v0`.
- `spec_id`: identificador estable de la especificacion.
- `request_id`: peticion origen.
- `created_at`: instante ISO 8601 asignado por el caso de uso.
- `locale`: locale principal.
- `project_source`: origen normalizado del proyecto.
- `app`: nombre, slug, objetivo, descripcion, tipo y usuarios objetivo.
- `scope`: objetivos, fuera de alcance, supuestos y preguntas abiertas.
- `architecture`: patron, modulos iniciales, fronteras y contratos esperados.
- `i18n`: enabled, default_locale, locales y justificacion si se desactiva.
- `data`: necesidades de persistencia expresadas como conector.
- `connectors`: conectores requeridos y opcionales.
- `platforms`: plataformas objetivo.
- `deploy`: target explicito o inferido y restricciones.
- `quality`: pruebas, accesibilidad, seguridad, compliance y observabilidad.
- `docs`: documentacion de usuario, desarrollo y sistemas.
- `agent_preferences`: preferencias no vinculantes para revision y autonomia.
- `defaults_applied`: defaults aplicados con motivo.
- `validation`: estado, warnings y errores publicos.

Invariantes:

- `schema_version` es inmutable dentro de la version `v0`.
- No hay excepcion productiva a arquitectura hexagonal para apps programadas por
  Orquesta en `AppSpecV0`: si se solicita otra arquitectura, la request se
  rechaza como incompatible.
- Toda excepcion a i18n o documentacion por defecto debe estar en
  `defaults_applied` o en `validation.warnings`.
- `data` no contiene tablas, SQL, dialectos ni detalles de proveedor.
- `connectors` no contiene imports internos ni structs privados de otro modulo.
- `architecture.modulos_iniciales` solo propone fronteras; no genera codigo.
- `architecture.modulos_iniciales` debe incluir domain, application, ports,
  adapters y bootstrap.
- `architecture.fronteras` debe declarar que domain/application no importan
  adaptadores ni tecnologias externas, que los handlers son finos y que el
  wiring vive en bootstrap/cmd.
- Una spec con `validation.estado=valida` puede alimentar el backlog inicial.

Errores:

- `app_spec_invalida`
- `opcion_incompatible`
- `target_no_soportado`
- `idioma_invalido`
- `conector_requerido_no_disponible`

Pruebas de contrato:

- Serializacion estable de `AppSpecV0`.
- Validacion de invariantes hexagonales estrictas/i18n/docs.
- Validacion de ausencia de DB directa y runtime.
- Validacion de errores publicos estables.

### Nombre: BacklogInicialPropuestoV0

Tipo: dto
Version: `v0`
Propietario: `orquesta-factory`
Consumidores: `orquesta-web`, `orquesta-core`, `orquesta-mcp`,
`orquesta-cli`

Campos:

- `schema_version`: valor fijo `backlog_inicial_propuesto.v0`.
- `spec_id`: `AppSpecV0` origen.
- `fases`: lista corta de fases iniciales propuestas.
- `microtareas`: tareas pequenas con objetivo, write-set previsto, contrato,
  validacion y bloqueos.
- `contratos_requeridos`: contratos que deben existir antes de implementar.
- `riesgos`: riesgos de alcance, datos, integraciones o compliance.
- `preguntas_abiertas`: informacion pendiente para cerrar la spec.

Invariantes:

- Toda microtarea tiene criterio de cierre verificable.
- Toda microtarea tiene write-set previsto o declara bloqueo.
- El backlog incluye `ArquitecturaHexagonalEstricta v0` como contrato requerido
  y microtarea de frontera antes de implementar.
- El backlog propuesto no arranca agentes ni asigna runtime.
- El backlog propuesto no crea registros en DB.
- Si una microtarea afecta otro modulo, debe quedar como bloqueo o consulta.

Errores:

- `app_spec_invalida`
- `opcion_incompatible`

Pruebas de contrato:

- Una spec valida produce microtareas cerradas y pequenas.
- Una spec con preguntas abiertas produce backlog marcado como provisional.
- Ninguna microtarea contiene acceso directo a DB, runtime o internals ajenos.

### Nombre: AppSpecV0Validada

Tipo: evento
Version: `v0`
Propietario: `orquesta-factory`
Consumidores: `orquesta-web`, `orquesta-core`, `orquesta-observability`

Campos:

- `schema_version`: valor fijo `app_spec_validada.v0`.
- `spec_id`
- `request_id`
- `validation_estado`
- `warnings_count`
- `correlation_id`

Invariantes:

- Evento compacto, sin transcript completo y sin secretos.
- El evento describe una validacion, no una ejecucion.
- Su publicacion futura debe pasar por el conector de observabilidad.

Errores:

- No aplica como error de dominio; fallos de transporte pertenecen al adaptador.

Pruebas de contrato:

- Payload minimo no contiene datos sensibles ni descripcion completa de la app.
- `spec_id`, `request_id` y `correlation_id` son obligatorios.

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
