# Guia de opciones del wizard `/nueva-app` - 2026-06-25

Esta guia documenta las opciones que el operador ve o debe poder ver en el
wizard `/nueva-app`. Es documentacion de uso y contrato de UI; no cambia codigo.

## Estado

- `/nueva-app` pertenece a `orquesta-web`, que es un adaptador inbound fino.
- La web captura intencion, preferencias y restricciones. No decide negocio, no
  elige proveedor real, no toca DB, no arranca runtime y no materializa backlog.
- El submit delega primero en `ArrancarDirectorAppClientV0` cuando esta
  inyectado: `POST /nueva-app -> /api/v0/apps/director -> goal_first`. El modo
  vacio de `director_execution_mode` arranca Codex Goal como loop interno; si
  falta backend Goal, el error publico es `goal_backend_unavailable` o el
  degradado `codex_app_server_*` correspondiente.
- `SolicitarNuevaAppClientV0` queda como fallback de compatibilidad cuando la
  web se ejecuta sin DirectorClient; en ese caso consume `SolicitarNuevaApp v0`
  y solo produce spec/backlog inicial, sin arrancar agentes.
- El contrato de captura local es `WebNuevaAppFormV0`; el contrato de dominio
  consumido es `AppSpecRequestV0`.
- El HTML actual implementa un modo basico guiado, un bloque experto opcional
  para detallar datos, almacenamiento e integraciones, y un bloque plegado de
  compatibilidad historica para forzar `legacy_director_loop` solo en smokes o
  rutas no migradas. El DTO y el POST JSON siguen soportando mas campos que los
  visibles en HTML.
- El bloque experto de datos e integraciones debe mantenerse opcional y plegarse
  sobre contratos publicos, sin meter DB concreta ni proveedor en web.

## Principios

1. La opcion debe describir necesidad funcional, no solucion interna.
2. Si una eleccion implica proveedor, runtime, modelo, DSN, token, HOME, ruta
   privada o backend concreto, debe quedar como restriccion/adaptador opt-in, no
   como regla del nucleo ni como decision de la web.
3. La arquitectura elegida es una preferencia/contrato de patron. Siempre debe
   conservar fronteras limpias: dominio y aplicacion separados de adaptadores,
   IO, proveedores y bootstrap.
4. i18n y documentacion estan activadas por defecto salvo justificacion.
5. La autonomia de agentes no elimina revision causal ni evidencias de cierre.
6. Los tooltips son ayuda corta; esta guia es la explicacion larga.
7. Las formas recuperables se normalizan o se revisan por Director/factory; la
   web no debe introducir rails por palabras sueltas.

## Modos Del Wizard

### Modo Basico

El modo basico es el flujo actual. Pide lo minimo para que Orquesta compile una
solicitud de app y, con `DirectorClient` inyectado, arranque el Director
Goal-first. La spec/backlog preview queda solo como fallback de compatibilidad
cuando no hay DirectorClient.

Usalo cuando:

- la app empieza desde cero;
- basta declarar una persistencia general;
- solo hay una integracion principal o ninguna;
- no hay sensibilidad distinta por tipo de dato;
- quieres que Orquesta proponga tecnologia y adaptadores.

### Modo Experto

El modo experto debe ser opcional y expansible. Sirve para describir datos e
integraciones con mas precision sin obligar al operador normal a rellenar un
formulario enorme.

Usalo cuando:

- hay varios tipos de datos con sensibilidad distinta;
- una app combina busqueda, objetos, historico, analitica o vectores;
- hay varias integraciones externas con direcciones y criticidad distintas;
- hay cumplimiento, retencion, auditoria, importacion/exportacion o offline;
- el contrato Goal/plan debe separar puertos, conectores y adaptadores desde el
  primer corte.

Regla de compatibilidad: mientras `AppSpecRequestV0` no tenga un DTO experto
dedicado, el bloque experto se debe resumir en campos existentes:
`datos.necesidad_funcional`, `datos.tipos_datos`, `datos.sensibilidad`,
`datos.retencion`, `integraciones[]`, `calidad.compliance` y `restricciones`.
No debe inventar campos privados ni acoplarse a tablas.

## Flujo General

1. `GET /nueva-app` renderiza el wizard localizado y no delega en el cliente.
2. El operador completa pasos. JavaScript mejora navegacion, presets, resumen
   vivo y puede consultar `POST /api/v0/apps/intake/guided-turn` para convertir
   una necesidad libre en decisiones editables de wizard. Si ese contrato no
   esta disponible, la pantalla conserva fallback local y el formulario debe
   seguir siendo usable sin JS.
3. `POST /nueva-app` acepta form-urlencoded o JSON local, construye
   `WebNuevaAppFormV0` y lo transforma a `AppSpecRequestV0`.
4. Si existe `ArrancarDirectorAppClientV0`, la web llama al bridge del Director.
   Con `director_execution_mode` vacio, `StartAppDirectorV0` normaliza a
   `goal_first`, compila/lanzar `GoalWorkSpecV0`, persiste `GoalWorkStateV0` y
   la pantalla observa por `/api/v0/apps/director/goal/observe`.
5. Si no existe DirectorClient, la web conserva el fallback legacy de
   `SolicitarNuevaAppClientV0`: `orquesta-factory` valida, normaliza, aplica
   defaults y devuelve `AppSpecV0` mas `BacklogInicialPropuestoV0` o errores
   publicos.
6. La web renderiza estado, resumen, errores, defaults, datos Goal/Director y
   preview de backlog sin materializar tareas por su cuenta.

## Compatibilidad Historica

`legacy_director_loop` ya no es el camino normal de `/nueva-app`. En la pantalla
queda dentro del bloque plegado "Compatibilidad historica" y debe usarse solo
para reproducir smokes antiguos, diagnosticar una ruta no migrada o comparar
comportamiento con evidencias previas.

Reglas:

- dejar `director_execution_mode` vacio significa automatico Goal-first;
- marcar "Forzar loop historico del Director" envia
  `director_execution_mode=legacy_director_loop`;
- no usar `legacy_director_loop` para nuevos trabajos productivos salvo orden
  explicita o ausencia documentada de backend Goal;
- el campo `execution_mode` es solo alcance de validacion (`normal` o `debug`),
  no cambia el loop del Director.

## Presets Rapidos

Los presets son atajos editables. No bloquean campos ni sustituyen al criterio
del operador.

### Web + API

Efecto actual:

- `tipo_app=web`
- marca plataformas `web` y `api`
- `preferencias_tecnicas.lenguaje=go`
- `deploy.target=contenedor`

Usalo para apps web con backend propio, paneles, CRUD, portales o productos
operativos. Revisalo si la app es solo estatica, solo mobile o no necesita API.

Errores frecuentes:

- dejar `objetivo` generico como "hacer una web";
- marcar `api` pero no describir que otros clientes la consumen;
- asumir que `contenedor` elige Kubernetes o proveedor cloud.

Tooltip relacionado: `nueva_app.ayuda.preset.webapp`.

### API REST

Efecto actual:

- `tipo_app=api`
- marca plataforma `api`
- `preferencias_tecnicas.lenguaje=go`
- `calidad.pruebas=alta`

Usalo para servicios consumidos por otros sistemas, integraciones o backends
headless. Si tambien necesitas UI, anade plataforma `web` o usa Web + API.

Errores frecuentes:

- no indicar consumidores ni contratos esperados;
- tratar integraciones como llamadas directas a DB ajena;
- elegir `api` cuando realmente se necesita automatizacion batch o CLI.

Tooltip relacionado: `nueva_app.ayuda.preset.api`.

### Panel Operativo

Efecto actual:

- `tipo_app=web`
- marca plataforma `web`
- `preferencias_tecnicas.framework=html/js + api`
- `calidad.observabilidad=true`

Usalo para backoffice, seguimiento de runs, colas, estados, expedientes,
auditoria o acciones operativas. Debe priorizar informacion densa, estados
claros, filtros y errores publicos.

Errores frecuentes:

- describirlo como landing comercial;
- omitir roles, permisos o estados de bloqueo;
- desactivar observabilidad en un panel que opera procesos vivos.

Tooltip relacionado: `nueva_app.ayuda.preset.ops`.

## Paso 1: Idea E Identidad

### `nombre`

Tipo UI: input obligatorio.

Que implica:

- identifica la solicitud, el resumen y el slug propuesto por factory;
- no debe incluir secretos, rutas, tokens ni nombres internos innecesarios.

Cuando usarlo:

- siempre. Debe ser corto y reconocible.

Errores frecuentes:

- nombres demasiado genericos: `app`, `panel`, `sistema`;
- meter entorno o credencial en el nombre.

Tooltip: `nueva_app.ayuda.nombre`.

### `tipo_app`

Tipo UI: select obligatorio.

Opciones visibles:

- `web`: interfaz navegador como superficie principal.
- `api`: servicio consumido por clientes o sistemas externos.
- `cli`: herramienta de linea de comandos.
- `desktop`: aplicacion de escritorio o paquete local.
- `mobile`: app movil o experiencia mobile-first.
- `automation`: automatizacion, job, bot o proceso programado.
- `data`: proceso centrado en datos, analitica, ETL o reporting.
- `plugin`: extension para un host existente.
- `mixed`: combinacion de superficies.

Opciones soportadas por factory pero no visibles en el select actual:

- `documentacion`
- `documentation`

Que implica:

- ayuda a inferir plataformas por defecto si el operador no marca ninguna;
- condiciona GoalWorkSpec/plan, pruebas y documentacion esperada;
- no decide framework ni proveedor.

Cuando usar cada opcion:

- `web`: usuario humano interactua en navegador.
- `api`: el producto principal es contrato HTTP/API.
- `cli`: uso tecnico, scripts, ops o automatizacion local.
- `desktop`: empaquetado local o integracion con SO.
- `mobile`: experiencia especifica de iOS/Android o PWA mobile-first.
- `automation`: flujo sin UI principal, por eventos o calendario.
- `data`: ingesta, transformacion, consulta, dashboards o modelos de datos.
- `plugin`: host externo manda ciclo de vida y extension points.
- `mixed`: mas de una superficie igual de importante.

Errores frecuentes:

- elegir `web` para cualquier cosa aunque lo principal sea API;
- elegir `mixed` para evitar decidir, cuando hay una superficie dominante;
- pedir documentacion con `web` en vez de `request_kind=documentar_app`.

Tooltip: `nueva_app.ayuda.tipo_app`.

### `objetivo`

Tipo UI: textarea obligatorio.

Que implica:

- es el criterio central para spec, scope y backlog;
- debe incluir resultado verificable, usuario beneficiario y condicion de exito.

Buena forma:

```text
Permitir a coordinadores revisar solicitudes pendientes, filtrar por prioridad
y registrar resolucion con auditoria antes del cierre diario.
```

Errores frecuentes:

- "crear una app moderna";
- listar tecnologias sin explicar resultado;
- mezclar veinte objetivos sin prioridad.

Tooltip: `nueva_app.ayuda.objetivo`.

### `descripcion`

Tipo UI: textarea opcional.

Que implica:

- aporta contexto, usuarios, reglas de dominio, restricciones y ejemplos;
- ayuda a que factory proponga preguntas abiertas en lugar de inventar.

Cuando usarla:

- cuando el objetivo no basta para entender dominio, datos o integraciones;
- cuando hay restricciones de UX, compliance, offline, idioma o entorno.

Errores frecuentes:

- pegar logs, secretos o rutas internas;
- repetir el objetivo sin anadir contexto;
- usarla para imponer proveedor en vez de declarar restriccion.

Tooltip: `nueva_app.ayuda.descripcion`.

## Identidad Avanzada

Esta seccion existe en el paso 1 y esta plegada bajo `details`.

### `request_id`

Tipo UI: input opcional.

Que implica:

- correlaciona la solicitud con logs/cliente/transporte;
- si falta, el cliente REST puede generar `req-web-*`.

Cuando usarlo:

- reintentos, integracion con herramienta externa o auditoria manual.

Errores frecuentes:

- reutilizar el mismo id para solicitudes distintas;
- meter informacion sensible.

Tooltip: `nueva_app.ayuda.request_id`.

### `locale`

Tipo UI: select.

Opciones actuales:

- `es-ES`
- `en-US`

Que implica:

- idioma de la solicitud y textos generados por defecto;
- si `i18n.default_locale` esta vacio, el mapper usa este valor.

Errores frecuentes:

- usar `es_ES` o textos no BCP 47 en JSON;
- confundir locale de UI con idiomas finales de la app.

Tooltip: `nueva_app.ayuda.locale`.

### `request_kind`

Tipo UI: select.

Opciones actuales:

- `crear_app_completa`
- `documentar_app`
- `analizar_app`
- `brainstorming_arquitectura`
- `planificar_app`
- `programar_modulo`
- `modificar_app_existente`
- `revisar_codigo`
- `pruebas_y_validacion`
- `seguridad`
- `deploy`
- `operacion_soporte`
- `integracion_externa`
- `i18n_l10n`
- `migracion_refactor`
- `investigacion_tecnica`

Que implica:

- define entregables minimos esperados por la politica de request;
- algunos tipos requieren `project_source` existente.

Guia rapida:

- `crear_app_completa`: app nueva con arquitectura, codigo, tests, docs y cierre.
- `documentar_app`: manuales, decisiones, pruebas documentales y pendientes.
- `analizar_app`: inventario, hallazgos, riesgos, seguridad y deuda.
- `brainstorming_arquitectura`: opciones comparadas y decision aceptada.
- `planificar_app`: alcance, contratos, microtareas y criterios de cierre.
- `programar_modulo`: cambio focal con write-set pequeno y tests focales.
- `modificar_app_existente`: cambio sobre repo existente.
- `revisar_codigo`: hallazgos, riesgos y evidencias.
- `pruebas_y_validacion`: plan, ejecucion, evidencias y rework.
- `seguridad`: amenazas, datos sensibles, permisos y dependencias.
- `deploy`: entorno, configuracion, healthcheck y rollback.
- `operacion_soporte`: soporte operativo y manual de operacion.
- `integracion_externa`: contrato de conector, fixtures y pruebas.
- `i18n_l10n`: catalogos, locales, fallbacks y pruebas i18n.
- `migracion_refactor`: contexto actual, plan, regresion y riesgos.
- `investigacion_tecnica`: opciones, evidencias, comparativa y recomendacion.

Requieren proyecto existente:

- `analizar_app`
- `modificar_app_existente`
- `revisar_codigo`
- `seguridad`
- `migracion_refactor`

Errores frecuentes:

- usar `crear_app_completa` para revisar un repo existente sin origen;
- usar `debug` para cerrar productivo;
- elegir `deploy` sin restricciones de entorno.

Tooltip: `nueva_app.ayuda.request_kind`.

### `execution_mode`

Tipo UI: select.

Opciones:

- `normal`: exige minimos completos y permite cierre productivo.
- `debug`: permite recortar alcance, exige informe de omisiones y no debe cerrar
  productivo como si estuviera completo.

Cuando usar `debug`:

- smoke, prueba de transporte, reproduccion acotada, diagnostico rapido.

Errores frecuentes:

- usar `debug` para ahorrar trabajo de una entrega real;
- no documentar que queda omitido.

Tooltip: `nueva_app.ayuda.execution_mode`.

## Paso 2: Tipo, Plataformas Y Origen

### `plataformas`

Tipo UI: checkboxes multiples.

Opciones visibles:

- `web`
- `mobile`
- `desktop`
- `api`

Que implica:

- declara superficies objetivo;
- si queda vacio, factory infiere por `tipo_app`.

Cuando usar:

- marca solo plataformas necesarias para el primer cierre;
- anade `api` si hay consumidores externos o frontend separado;
- anade `mobile` si hay app movil real o mobile-first con requisitos propios.

Errores frecuentes:

- marcar todo por si acaso;
- marcar `mobile` solo porque la web debe ser responsive;
- no marcar `api` aunque se pidan integraciones consumidoras.

Tooltips:

- `nueva_app.ayuda.plataformas.web`
- `nueva_app.ayuda.plataformas.mobile`
- `nueva_app.ayuda.plataformas.desktop`
- `nueva_app.ayuda.plataformas.api`

### `project_source.kind`

Tipo UI: select.

Opciones:

- vacio: factory infiere `new`, `github` o `local_path`.
- `new`: app nueva desde cero.
- `github`: repo remoto GitHub sin credenciales.
- `local_path`: ruta local autorizada por composicion.

Que implica:

- `github` requiere `git_url`;
- `local_path` requiere `local_path`;
- `new` no admite `git_url` ni `local_path`.

Errores frecuentes:

- declarar `new` y poner una URL;
- poner `git_url` y `local_path` a la vez;
- usar una URL con token o usuario embebido.

Tooltip: `nueva_app.ayuda.project_source.kind`.

### `project_source.git_url`

Tipo UI: input.

Usalo para:

- repos publicos o accesibles por el adaptador autorizado;
- trabajos sobre app existente cuando el origen es remoto.

No usar para:

- URLs con credenciales;
- DSN de DB;
- endpoints de API no Git.

Tooltip: `nueva_app.ayuda.project_source.git_url`.

### `project_source.branch`

Tipo UI: input.

Usalo para:

- rama, tag o ref publica de partida.

Errores frecuentes:

- asumir que la web ejecuta Git;
- usar una rama privada sin que el adaptador tenga acceso.

Tooltip: `nueva_app.ayuda.project_source.branch`.

### `project_source.project_ref`

Tipo UI: input.

Que implica:

- ref opaca si Orquesta o la composicion ya conoce el proyecto;
- la web no interpreta ni valida la ref.

Cuando usarlo:

- flujos integrados con un registro externo o run previo.

Tooltip: `nueva_app.ayuda.project_source.project_ref`.

### `project_source.local_path`

Tipo UI: input.

Usalo solo cuando:

- la composicion autoriza leer ese path;
- el operador entiende que esta exponiendo una ruta local.

Errores frecuentes:

- usar rutas privadas innecesarias;
- asumir que todos los runtimes comparten filesystem;
- mezclar `local_path` con `github`.

Tooltip: `nueva_app.ayuda.project_source.local_path`.

## Paso 3: Tecnologia E i18n

### `preferencias_tecnicas.arquitectura`

Tipo UI: select.

Opciones de producto esperadas:

- `hexagonal`
- `clean_architecture`
- `onion`
- `modular_monolith`
- `layered`
- `event_driven`
- `microservices`
- `serverless`
- `plugin_based`
- `data_pipeline`

Que implica:

- expresa el patron preferido para organizar el trabajo y sus fronteras;
- sigue siendo contrato de arquitectura, no proveedor ni framework;
- no permite mezclar dominio con adaptadores, handlers, DB, filesystem, colas,
  runtime, LLM, cloud ni bootstrap;
- cada patron debe traducirse a modulos, puertos, adaptadores y criterios de
  cierre verificables;
- si el operador no elige nada, el fallback/default conservador es `hexagonal`;
- `hexagonal` no es obligatorio: los patrones soportados deben conservar
  fronteras limpias y criterios verificables.

Cuando usar cada opcion:

- `hexagonal`: default general; dominio claro, puertos explicitos y adaptadores
  intercambiables.
- `clean_architecture`: cuando quieres capas concentricas, casos de uso
  explicitos y dependencia hacia dentro.
- `onion`: dominio muy central, reglas ricas y adaptadores externos variables.
- `modular_monolith`: producto unico desplegado junto, pero con modulos internos
  fuertes y contratos entre bounded contexts.
- `layered`: CRUD o app simple con capas clasicas, manteniendo dominio fuera de
  handlers y persistencia.
- `event_driven`: integraciones por eventos, asincronia, auditoria o procesos
  reactivos.
- `microservices`: equipos, despliegues o dominios realmente independientes;
  exige contratos, observabilidad y coste operativo mayores.
- `serverless`: funciones/eventos gestionados, cargas intermitentes y poco
  estado local; requiere frontera clara con almacenamiento y proveedores.
- `plugin_based`: extension points, host existente o ecosistema de plugins.
- `data_pipeline`: ingesta, transformacion, validacion, reporting o flujos de
  datos como producto principal.

Guia operativa por arquitectura:

| Patron | Para que sirve | Cuando conviene | Coste o tradeoff |
| --- | --- | --- | --- |
| `hexagonal` | Separar dominio, casos de uso, puertos y adaptadores. | Default cuando quieres cambiar DB/UI/API sin tocar reglas. | Mas contratos iniciales que un CRUD pequeno. |
| `clean_architecture` | Hacer que las dependencias apunten hacia casos de uso y entidades. | Productos con reglas y casos de uso claros. | Puede sentirse ceremonial si la app es trivial. |
| `onion` | Poner el dominio rico en el centro y rodearlo de servicios/adaptadores. | Dominios con reglas estables y muchos bordes externos. | Requiere disciplina para no filtrar infraestructura hacia dentro. |
| `modular_monolith` | Un despliegue unico con modulos internos fuertes. | Producto inicial grande, un equipo o pocos equipos, bajo coste operativo. | Si los limites son falsos, acaba siendo monolito acoplado. |
| `layered` | Ordenar UI/API, aplicacion, dominio y persistencia por capas. | CRUD simple, panel interno o app de bajo riesgo. | Es facil acabar con SQL o transporte en capas que no tocan. |
| `event_driven` | Comunicar cambios mediante eventos y procesos asincronos. | Auditoria, integraciones, colas, sincronizacion o flujos reactivos. | Debug, orden, reintentos e idempotencia cuestan mas. |
| `microservices` | Separar servicios desplegables y propietarios de datos. | Equipos/dominios autonomos con necesidad real de despliegue independiente. | Mucha observabilidad, contratos, red, seguridad y operaciones. |
| `serverless` | Ejecutar funciones/eventos con infraestructura gestionada. | Carga intermitente, jobs pequenos, integraciones eventuales. | Lock-in/adaptador, cold starts y limites de estado local. |
| `plugin_based` | Permitir extension mediante contratos de plugin. | Producto extensible, host existente o ecosistema de terceros. | Versionado, aislamiento y compatibilidad hacia atras. |
| `data_pipeline` | Modelar ingesta, transformacion, validacion y publicacion de datos. | ETL/ELT, reporting, analitica o feeds como producto central. | Idempotencia, linaje, calidad de datos y re-ejecuciones. |

Errores frecuentes:

- pedir "MVC rapido" para saltarse fronteras;
- meter handlers, persistencia y reglas de dominio en una misma pieza;
- pensar que microservicios o serverless son mejoras automaticas;
- usar `event_driven` para evitar modelar comandos y estados;
- elegir proveedor cloud como si fuera arquitectura;
- pedir `layered` y meter SQL en dominio.

Tooltip: `nueva_app.ayuda.preferencias_tecnicas.arquitectura`.

### `preferencias_tecnicas.lenguaje`

Tipo UI: input.

Que implica:

- preferencia tecnica si hay razon de equipo, entorno o librerias;
- no debe imponer runtime real dentro de core.

Cuando dejar vacio:

- si prefieres que Orquesta proponga stack.

Errores frecuentes:

- escribir varios lenguajes incompatibles sin prioridad;
- usarlo para pedir proveedor LLM o DB.

Tooltip: `nueva_app.ayuda.preferencias_tecnicas.lenguaje`.

### `preferencias_tecnicas.framework`

Tipo UI: input.

Que implica:

- framework deseado si hay criterio real;
- debe respetar el patron de arquitectura elegido y dejar IO en adaptadores.

Errores frecuentes:

- elegir framework por moda sin restricciones;
- escribir "React + Postgres + Redis + OpenAI" mezclando UI, DB, cache y LLM.

Tooltip: `nueva_app.ayuda.preferencias_tecnicas.framework`.

### `preferencias_tecnicas.preferencias`

Tipo UI: input de lista por comas.

Usalo para:

- librerias permitidas, estilo, patrones, convenciones, soporte offline,
  requisitos de build o packaging.

Errores frecuentes:

- meter secretos;
- meter rutas internas;
- usarlo como lista de tareas completa.

Tooltip: `nueva_app.ayuda.preferencias_tecnicas.preferencias`.

### `preferencias_tecnicas.restricciones`

Tipo UI: input de lista por comas en preferencias tecnicas.

Usalo cuando:

- una tecnologia esta prohibida;
- hay licencias, version minima, SO o entorno restringido.

### `i18n.enabled`

Tipo UI: select booleano.

Opciones:

- `true`
- `false`

Default:

- `true` si falta.

Que implica:

- crea expectativa de catalogos, locales y pruebas i18n;
- si es `false`, factory exige `i18n.justificacion`.

Errores frecuentes:

- desactivarlo sin justificar;
- confundir idioma de operador con locales finales.

Tooltip: `nueva_app.ayuda.i18n.enabled`.

### `i18n.default_locale`

Tipo UI: input.

Que implica:

- locale base para textos iniciales;
- si queda vacio, usa `locale`.

Errores frecuentes:

- usar valores no BCP 47;
- poner varios locales aqui en vez de `i18n.locales`.

Tooltip: `nueva_app.ayuda.i18n.default_locale`.

### `i18n.locales`

Tipo UI: input de lista por comas.

Ejemplo:

```text
es-ES,en-US
```

Que implica:

- locales adicionales a generar o preparar;
- documentacion hereda locales i18n si `documentacion.locales` esta vacio.

Tooltip: `nueva_app.ayuda.i18n.locales`.

### `i18n.justificacion`

Tipo UI: input.

Obligatorio cuando:

- `i18n.enabled=false`.

Buena justificacion:

```text
Uso interno de un equipo monolingue, sin exposicion publica en este corte.
```

Tooltip: `nueva_app.ayuda.i18n.justificacion`.

## Paso 4: Datos E Integraciones

Este paso tiene modo basico actual y bloque experto opcional.

### Modo Basico: `datos.db_required`

Tipo UI: select booleano.

Opciones:

- `false`
- `true`

Que implica:

- `true` significa que la app debe persistir datos propios;
- obliga a declarar `datos.necesidad_funcional`;
- factory lo transforma en necesidad de conector `persistence`, no en DB real.

Cuando usar `true`:

- usuarios, expedientes, historico, configuracion, auditoria, favoritos,
  documentos, eventos, sesiones o estado durable.

Cuando usar `false`:

- calculadora sin estado, prototipo estatico, cliente de API externa sin cache
  propia, documentacion o exploracion.

Errores frecuentes:

- marcar `false` aunque se pida historico o auditoria;
- marcar `true` y escribir "Postgres" como necesidad funcional;
- pensar que `true` crea tablas.

Tooltip: `nueva_app.ayuda.datos.db_required`.

### Modo Basico: `datos.necesidad_funcional`

Tipo UI: input.

Que implica:

- explica por que se guardan o consultan datos;
- es obligatorio cuando `db_required=true`.

Ejemplos buenos:

- `guardar usuarios y permisos`
- `auditar cambios de estado`
- `mantener historico de solicitudes`
- `cachear resultados importados con fecha de caducidad`

Errores frecuentes:

- escribir solo `base de datos`;
- nombrar proveedor;
- omitir necesidad funcional.

Tooltip: `nueva_app.ayuda.datos.necesidad_funcional`.

### Modo Basico: `datos.tipos_datos`

Tipo UI: input de lista por comas.

Que implica:

- enumera categorias de datos, no tablas ni columnas;
- ayuda a separar entidades, puertos, fixtures y pruebas.

Ejemplos:

```text
usuarios,solicitudes,adjuntos,eventos_auditoria
```

Errores frecuentes:

- meter SQL;
- meter nombres de tablas definitivas;
- mezclar datos con integraciones externas.

Tooltip: `nueva_app.ayuda.datos.tipos_datos`.

### Modo Basico: `datos.sensibilidad`

Tipo UI: input.

Que implica:

- sensibilidad global si no se usa bloque experto;
- afecta preguntas, compliance, seguridad, logging y pruebas.

Valores orientativos:

- `publico`
- `interno`
- `personal`
- `sanitario`
- `financiero`
- `confidencial`
- `mixto`

Errores frecuentes:

- dejarlo vacio para apps con PII;
- decir `personal` sin indicar que tipos de datos lo son;
- usarlo para meter politicas largas que deben ir en `restricciones`.

Tooltip: `nueva_app.ayuda.datos.sensibilidad`.

### Modo Basico: `datos.retencion`

Tipo UI: input de texto en datos.

Usalo para:

- tiempo de conservacion;
- borrado, archivado, anonimizado o exportacion;
- requisitos legales.

Ejemplo:

```text
eventos de auditoria 5 anos; adjuntos 18 meses; borrado bajo solicitud
```

### Modo Experto: Tipos De Datos Multiples

El bloque experto debe permitir varias filas de tipo de dato. Cada fila debe
describir dominio, sensibilidad y ciclo de vida sin obligar a modelar tablas.
El HTML actual muestra seis filas, el mismo limite que parsea el endpoint v0.

Campos recomendados:

- `nombre`: categoria funcional, por ejemplo `usuarios`, `solicitudes`,
  `adjuntos`, `mensajes`, `embeddings`, `eventos_auditoria`.
- `proposito`: por que existe ese dato.
- `operaciones`: crear, leer, actualizar, borrar, buscar, exportar, auditar.
- `sensibilidad`: sensibilidad especifica de ese tipo.
- `retencion`: conservacion o borrado especifico.
- `volumen`: bajo, medio, alto o desconocido.
- `criticidad`: si su perdida bloquea el objetivo.
- `origen`: usuario, importacion, integracion, sistema, generado.
- `salida`: UI, API, exportacion, notificacion, reporte.

Como mapear a v0 sin nuevo DTO:

- `nombre` se agrega a `datos.tipos_datos`;
- sensibilidades se resumen en `datos.sensibilidad`;
- retenciones se resumen en `datos.retencion`;
- proposito y operaciones alimentan `datos.necesidad_funcional`;
- decisiones no representables pasan a `restricciones`.

Cuando usarlo:

- cualquier app con PII y datos no sensibles mezclados;
- apps con adjuntos u objetos junto a registros transaccionales;
- busqueda semantica, RAG o recomendadores con vectores;
- auditoria con retencion diferente al dato operacional;
- importaciones periodicas con datos temporales.

Errores frecuentes:

- crear una fila por columna;
- usar nombres de tablas como contrato cerrado;
- marcar todo como maxima sensibilidad sin distinguir;
- esconder datos sensibles en `descripcion`.

### Modo Experto: Sensibilidad Por Tipo

Niveles orientativos:

- `publico`: puede mostrarse publicamente.
- `interno`: operativo, no publico, sin PII fuerte.
- `personal`: identifica o puede identificar personas.
- `especial`: categorias protegidas como salud, menores, biometria u otras.
- `financiero`: pagos, facturacion, importes sensibles o riesgos economicos.
- `secreto`: credenciales, claves o material que no debe entrar en la request.
- `mixto`: hay varios niveles y deben separarse por tipo.

Reglas:

- nunca pegues secretos reales;
- si aparece `secreto`, describe la necesidad como "gestionar credenciales por
  puerto seguro", no copies la credencial;
- si hay PII, activa preguntas sobre minimizacion, logs, exportacion y borrado;
- si hay datos especiales, eleva compliance y revision humana.

### Modo Experto: Preferencia De Persistencia

La preferencia de persistencia describe forma de almacenamiento o consulta. No
es proveedor ni decision cerrada de adaptador.
El HTML actual muestra seis preferencias de almacenamiento, el mismo limite
que parsea el endpoint v0.

Opciones recomendadas:

- `sin_preferencia`: Orquesta propone segun dominio.
- `sin_persistencia`: no hay estado durable propio.
- `relacional`: entidades con relaciones, transacciones, consistencia y queries
  estructuradas.
- `documental`: documentos flexibles, estructura variable o agregados JSON.
- `objetos_blob`: ficheros, adjuntos, imagenes, PDFs, audio o binarios.
- `vectorial`: embeddings, busqueda semantica, RAG o similitud.
- `clave_valor_cache`: cache, sesiones, locks o datos efimeros.
- `busqueda`: indice full-text, filtros rapidos o ranking.
- `eventos_auditoria`: append-only, trazabilidad, event log o auditoria.
- `grafo`: relaciones profundas, dependencias, redes o recomendaciones por
  conexiones.
- `series_temporales`: metricas, sensores, eventos temporales densos.
- `mixta`: varias necesidades con conectores separados.

Aliases aceptados por compatibilidad:

- `objetos`: equivalente amplio cuando aun no se distingue `objetos_blob`.
- `clave_valor`: equivalente amplio cuando aun no se distingue cache/sesiones.
- `cache`: alias corto para `clave_valor_cache` cuando el dato es efimero.

Cuando usar cada una:

- `relacional`: CRUD transaccional, usuarios, permisos, expedientes, pagos.
- `documental`: formularios cambiantes, configuracion flexible, contenido.
- `objetos_blob`: archivos grandes o contenido no consultable como filas.
- `vectorial`: preguntas semanticas, deduplicacion por similitud, RAG.
- `clave_valor_cache`: acelerar lecturas o guardar estado temporal.
- `busqueda`: necesidad fuerte de buscar por texto, filtros y relevancia.
- `eventos_auditoria`: reconstruir acciones o cumplir trazabilidad.
- `grafo`: relaciones son el producto.
- `series_temporales`: volumen temporal y agregados por ventana.
- `mixta`: una app con registros relacionales, adjuntos y busqueda semantica.

Errores frecuentes:

- escribir `Postgres`, `MongoDB`, `Redis`, `S3`, `OpenSearch` u otro proveedor
  como si fuera decision de dominio;
- elegir vectorial porque "usa IA" sin necesidad de similitud;
- elegir cache para datos que deben ser fuente de verdad;
- mezclar objetos grandes dentro de relacional sin justificar;
- pedir `mixta` para todo sin explicar que datos van a cada forma.

### Modo Experto: Requisitos De Datos Avanzados

Campos recomendados:

- `consistencia`: eventual, fuerte, transaccional, no critica.
- `offline`: si debe funcionar sin red y reconciliar despues.
- `importacion_exportacion`: formatos esperados, frecuencia y volumen.
- `auditoria`: acciones que deben quedar trazadas.
- `anonimizacion`: borrado, minimizacion o pseudonimizacion.
- `backups`: criticidad de recuperacion y periodo aceptable.
- `migraciones`: si parte de datos existentes.

Mapeo v0:

- requisitos durables se resumen en `restricciones`;
- formatos externos tambien pueden generar `integraciones[]`;
- compliance se refleja en `calidad.compliance`.

## Integraciones

### Modo Basico: `integraciones.0.tipo`

Tipo UI: select de una integracion visible.

Opciones actuales:

- vacio
- `api`
- `webhook`
- `email`
- `calendar`
- `maps`
- `file_import`
- `file_export`
- `messaging`
- `llm`
- `storage`
- `payments`
- `auth`
- `analytics`
- `search`
- `notifications`
- `other`

Que implica:

- describe capacidad externa, no proveedor concreto;
- si los campos de integracion quedan vacios, no se materializa integracion.

Cuando usar:

- `api`: consultar o enviar datos a sistema externo.
- `webhook`: recibir eventos o notificar cambios.
- `email`: enviar, recibir o procesar correo.
- `calendar`: agenda, citas, eventos o disponibilidad.
- `maps`: mapas, geocodificacion o rutas como capacidad.
- `file_import`: entrada de ficheros externos.
- `file_export`: salida de ficheros, informes, paquetes o exportaciones.
- `messaging`: mensajeria, chat, colas de mensajes o canales asincronos.
- `llm`: integracion con modelos de lenguaje como capacidad opcional.
- `storage`: almacenamiento externo aportado por otro sistema.
- `payments`: pagos como capacidad, sin credenciales ni proveedor impuesto.
- `auth`: autenticacion externa o identidad federada.
- `analytics`: medicion de uso o eventos de producto.
- `search`: busqueda externa o indice especializado.
- `notifications`: avisos push, SMS u otros canales no email.
- `other`: integracion necesaria que no encaja en las categorias anteriores.

Errores frecuentes:

- usar `api` sin describir proposito;
- meter credenciales;
- escribir proveedor/backend concreto en `nombre`.

Tooltip: `nueva_app.ayuda.integraciones.0.tipo`.

### Modo Basico: `integraciones.0.nombre`

Tipo UI: input.

Que implica:

- nombre publico o categoria de sistema externo;
- factory rechaza credenciales, DSN y proveedores/backend impuestos en ciertos
  casos por politica de capacidad de conector.

Bueno:

- `servicio de notificaciones`
- `sistema de citas`
- `pasarela de pagos` si es una capacidad generica

Malo:

- `postgres://...`
- `api_key=...`
- `Redis cache obligatorio`
- `OpenAI SDK backend`

Tooltip: `nueva_app.ayuda.integraciones.0.nombre`.

### Modo Basico: `integraciones.0.proposito`

Tipo UI: input.

Que implica:

- explica que hace la integracion y por que;
- sin proposito, Orquesta no puede separar puertos ni errores.

Ejemplos:

- `enviar avisos de cambio de estado`
- `consultar disponibilidad de citas`
- `importar solicitudes aprobadas cada noche`

Tooltip: `nueva_app.ayuda.integraciones.0.proposito`.

### Modo Basico: `integraciones.0.requerido`

Tipo UI: select booleano.

Opciones:

- `false`
- `true`

Que implica:

- `true` coloca la integracion como requerida para cumplir objetivo;
- `false` la deja como opcional o mejora.

Errores frecuentes:

- marcar todo requerido;
- marcar opcional una integracion sin la que el objetivo no se puede verificar.

Tooltip: `nueva_app.ayuda.integraciones.0.requerido`.

### Modo Basico: `integraciones.0.restricciones`

Tipo UI: input de lista por comas.

Usalo para:

- limites de tasa, sandbox, formato, ventana horaria, requisitos de retry o
  compatibilidad.

No usar para:

- credenciales, tokens, DSN o secretos.

### Atajo UI: Conectores Frecuentes

Tipo UI: checkboxes multiples sin campo contractual propio.

Que implica:

- permite marcar varias capacidades externas habituales y pulsar "Aplicar
  conectores";
- rellena las primeras filas libres de `integraciones[]` con `tipo`, `nombre`,
  `proposito` y `auth` editables;
- no envia el formulario ni elige proveedor, credenciales, DB, runtime ni
  backend concreto;
- si JavaScript no esta disponible, las filas manuales de integraciones siguen
  siendo el contrato usable.

Opciones visibles del atajo:

- `api`
- `auth`
- `notifications`
- `payments`
- `search`
- `analytics`

Cuando usarlo:

- paneles que suelen necesitar identidad y notificaciones;
- apps con pagos, analitica o busqueda como capacidades externas;
- contratos iniciales donde conviene recordar varios conectores sin escribir
  cada fila desde cero.

Errores frecuentes:

- asumir que marcar `payments` selecciona un proveedor de pagos;
- dejar las filas generadas sin revisar `requerido`, direccion o datos;
- usar el atajo para esconder integraciones criticas sin proposito verificable.

Tooltips: `nueva_app.ayuda.connectors.quick` y
`nueva_app.ayuda.connectors.apply`.

### Modo Experto: Varias Integraciones

El DTO `WebNuevaAppFormV0` ya modela `integraciones[]`; el HTML muestra una
integracion principal y cinco filas expertas adicionales, hasta seis
integraciones visibles en total sin crear dependencias directas.

Campos recomendados por integracion:

- `tipo`: capacidad general.
- `nombre`: sistema o capacidad publica.
- `proposito`: tarea concreta.
- `requerido`: si bloquea el objetivo.
- `direccion`: entrada, salida o bidireccional.
- `datos_intercambiados`: categorias de datos, no payload sensible.
- `frecuencia`: realtime, bajo demanda, batch, horario.
- `errores`: timeout, reintento, compensacion, idempotencia.
- `autenticacion`: descripcion de mecanismo esperado sin secretos.
- `ambiente`: sandbox, staging, produccion, temporal.
- `restricciones`: limites publicos y no sensibles.

Tipos expertos recomendados:

- `api`
- `webhook`
- `email`
- `calendar`
- `file_import`
- `file_export`
- `auth`
- `payments`
- `messaging`
- `analytics`
- `search`
- `llm`
- `storage`
- `maps`
- `notifications`
- `other`

Regla de contrato:

- si el selector experto ofrece tipos fuera de los basicos, debe mapearlos a un
  contrato aceptado por factory o ampliar factory con prueba de contrato. Hasta
  entonces, se pueden transportar como texto en `nombre`, `proposito` y
  `restricciones` sin romper validacion.

Cuando usar varias integraciones:

- importacion de CSV, envio de email y API de consulta externa en la misma app;
- autenticacion corporativa mas notificaciones;
- RAG con fuente documental, almacenamiento de objetos y busqueda;
- pagos mas webhook de confirmacion;
- calendario mas email de recordatorio.

Errores frecuentes:

- confundir integracion con persistencia propia;
- meter "base de datos externa" como atajo para saltarse puertos;
- no distinguir requerido/opcional;
- no indicar direccion de datos;
- pegar payloads reales con PII;
- introducir secretos.

## Paso 5: Calidad Y Despliegue

### `calidad.pruebas`

Tipo UI: select.

Opciones:

- `basica`
- `media`
- `alta`

Que implica:

- `basica`: cobertura minima focal y smoke simple.
- `media`: unitarias/contrato y flujos principales.
- `alta`: cobertura amplia, smokes, regresion y evidencias antes de cierre.

Cuando usar:

- `basica`: prototipo, documentacion o prueba acotada.
- `media`: app interna no critica.
- `alta`: API, datos sensibles, integraciones, pagos, seguridad, ops.

Errores frecuentes:

- elegir `basica` para flujo critico;
- elegir `alta` sin aceptar mas tiempo y evidencias.

Tooltip: `nueva_app.ayuda.calidad.pruebas`.

### `calidad.accesibilidad`

Tipo UI: select.

Opciones visibles actuales:

- `basica`
- `normal`
- `wcag_aa`
- `no_aplica`

Contrato de opciones esperado para modo basico/experto:

- `basica`: nivel minimo aceptable para interfaces internas o prototipos.
- `normal`: alias operativo o nivel intermedio que la composicion/adaptador
  puede aceptar cuando quiera separar "minimo tecnico" de "WCAG formal".
- `wcag_aa`: objetivo WCAG AA para interfaces publicas, criticas o con
  obligacion normativa.
- `no_aplica`: solo para trabajos sin UI humana.

Que implica:

- el operador debe poder elegir el nivel aceptado por la composicion en tiempo
  de ejecucion/adaptador;
- la web solo transporta el nivel; no decide herramienta, proveedor de auditoria
  ni framework de accesibilidad;
- `normal` es un nivel canonico soportado por el contrato; si una composicion
  antigua no lo soporta, debe devolver error publico o normalizarlo en su
  adaptador con evidencia;
- el nivel elegido afecta criterios de cierre, pruebas visuales, componentes y
  revisiones, pero no justifica acoplar UI con negocio.

Errores frecuentes:

- usar `basica` o `normal` para servicios publicos que requieren WCAG AA;
- usar `no_aplica` en apps web.
- convertir el nivel en proveedor concreto, por ejemplo "auditoria con X".

Tooltip: `nueva_app.ayuda.calidad.accesibilidad`.

### `calidad.accesibilidad_opciones`

Tipo UI: checkboxes multiples.

Opciones visibles actuales:

- `normal`
- `wcag_aa`
- `teclado`
- `lectores_pantalla`
- `contraste_alto`
- `movimiento_reducido`
- `subtitulos_transcripciones`
- `no_aplica`

Contrato:

- `normal` y `wcag_aa` siguen siendo niveles aceptados por runtime/adaptador.
- `teclado`, `lectores_pantalla`, `contraste_alto`,
  `movimiento_reducido` y `subtitulos_transcripciones` son criterios
  adicionales de cierre y pruebas.
- el nivel base elegido en `calidad.accesibilidad` se conserva como primera
  opcion normalizada si no esta duplicado.

Cuando usar:

- `teclado`: toda accion debe poder completarse con teclado y foco visible.
- `lectores_pantalla`: etiquetas, landmarks, nombres accesibles y orden
  semantico deben ser verificables.
- `contraste_alto`: textos, controles, estados y graficos requieren contraste
  reforzado.
- `movimiento_reducido`: animaciones, transiciones o autoplay deben poder
  desactivarse o reducirse.
- `subtitulos_transcripciones`: audio, video o contenido multimedia requiere
  subtitulos o transcripcion.

Errores frecuentes:

- marcar criterios adicionales en un trabajo sin UI humana;
- confundir estos criterios con un proveedor de auditoria concreto;
- elegir `no_aplica` junto a requisitos de interfaz web o movil.

Tooltip: `nueva_app.ayuda.calidad.accesibilidad_opciones`.

### `calidad.observabilidad`

Tipo UI: select booleano.

Opciones:

- `true`
- `false`

Default:

- `true` si falta.

Que implica:

- logs, metricas o estado operativo desde primer corte;
- especialmente importante para panels, jobs, colas e integraciones.

Errores frecuentes:

- desactivarla en automatizaciones sin UI;
- confundir observabilidad con exponer logs privados al usuario.

Tooltip: `nueva_app.ayuda.calidad.observabilidad`.

### `calidad.compliance`

Tipo UI: input de lista por comas.

Usalo para:

- WCAG, GDPR/RGPD, ENS, HIPAA, PCI, ISO, auditoria interna u otra norma.

Errores frecuentes:

- escribir "seguro" sin norma o requisito verificable;
- meter datos sensibles;
- dejarlo vacio con datos personales.

Tooltip: `nueva_app.ayuda.calidad.compliance`.

### `deploy.target`

Tipo UI: select.

Opciones:

- `sin_preferencia`
- `local`
- `contenedor`
- `paas`
- `serverless`
- `kubernetes`
- `desktop`
- `mobile_store`

Que implica:

- preferencia de entrega, no ejecucion inmediata;
- si falta, factory aplica `sin_preferencia`.

Cuando usar:

- `sin_preferencia`: deja que Orquesta proponga.
- `local`: herramienta local, demo, escritorio ligero o entorno controlado.
- `contenedor`: deploy reproducible, API o web empaquetada.
- `paas`: plataforma gestionada sin Kubernetes propio.
- `serverless`: funciones/eventos con baja gestion operativa.
- `kubernetes`: plataforma ya existente con operacion preparada.
- `desktop`: instalador o paquete de escritorio.
- `mobile_store`: distribucion por tiendas moviles.

Errores frecuentes:

- elegir Kubernetes por defecto;
- elegir serverless con procesos largos o estado local;
- confundir `contenedor` con proveedor cloud concreto.

Tooltip: `nueva_app.ayuda.deploy.target`.

### `deploy.restricciones`

Tipo UI: input de lista por comas.

Usalo para:

- puertos, offline, SO, cloud/no cloud, limites de memoria, red, privacidad,
  empaquetado, dominios, certificados o entorno temporal.

Errores frecuentes:

- meter credenciales;
- escribir comandos de deploy como contrato;
- omitir restricciones de red en apps con integraciones.

Tooltip: `nueva_app.ayuda.deploy.restricciones`.

### `documentacion.usuario`

Tipo UI: select booleano.

Opciones:

- `true`
- `false`

Default:

- `true` si falta.

Que implica:

- pide manual funcional orientado a usuarios finales;
- debe cubrir pantallas, flujos principales, errores esperables y decisiones de
  uso;
- no sustituye al contrato ni a los tests, pero ayuda a validar que la app se
  puede operar por una persona real.

Cuando usar `true`:

- aplicaciones con UI, paneles internos, portales publicos, apps moviles,
  herramientas operativas o cualquier entrega que vaya a usar alguien distinto
  del programador.

Cuando usar `false`:

- librerias internas, tareas de investigacion, refactors sin interfaz o pruebas
  tecnicas acotadas donde no hay usuario final directo.

Errores frecuentes:

- marcar `false` para ahorrar tiempo en una app con usuarios reales;
- confundir documentacion de usuario con comentarios de codigo;
- no indicar idiomas cuando el producto es multilingue.

Tooltip: `nueva_app.ayuda.documentacion.usuario`.

### `documentacion.desarrollo`

Tipo UI: select booleano.

Opciones:

- `true`
- `false`

Default:

- `true` si falta.

Que implica:

- pide guia para continuar el proyecto: arquitectura, comandos, estructura,
  tests, decisiones relevantes y puntos de extension;
- debe explicar la arquitectura elegida, por ejemplo hexagonal, clean,
  monolito modular, eventos u otra opcion seleccionada;
- sirve para que otro agente o programador pueda seguir sin reconstruir todo el
  contexto.

Cuando usar `true`:

- apps que seguiran evolucionando, codigo generado por agentes, integraciones,
  dominio no trivial o cualquier entrega que deba mantenerse.

Cuando usar `false`:

- prototipos descartables o entregas puramente documentales sin codigo.

Errores frecuentes:

- dejarla fuera en apps que luego se mantendran;
- pedir una guia de desarrollo y no exigir comandos verificables;
- mezclar secretos, rutas privadas o credenciales dentro de la guia.

Tooltip: `nueva_app.ayuda.documentacion.desarrollo`.

### `documentacion.sistemas`

Tipo UI: select booleano.

Opciones:

- `true`
- `false`

Default:

- `true` si falta.

Que implica:

- pide documentacion operativa: configuracion, despliegue, variables, logs,
  healthchecks, backups, restauracion, limites y diagnostico;
- no autoriza deploy real por si sola; solo describe como se opera la app cuando
  el adaptador autorizado exista.

Cuando usar `true`:

- apps con servidor, base de datos, colas, integraciones externas, tareas
  programadas, observabilidad o despliegue fuera del entorno local.

Cuando usar `false`:

- scripts locales pequenos, investigacion o artefactos sin runtime operativo.

Errores frecuentes:

- escribir credenciales o DSN reales;
- omitir rollback o recuperacion en una app productiva;
- elegir `deploy.target` sin documentar como se verifica el arranque.

Tooltip: `nueva_app.ayuda.documentacion.sistemas`.

### `documentacion.profundidad`

Tipo UI: select.

Opciones:

- `basica`
- `normal`
- `profunda`

Default:

- `normal` si falta.

Que implica:

- `basica`: documentacion corta para operar la entrega minima, con alcance,
  comandos esenciales y limitaciones conocidas.
- `normal`: manual de usuario, guia de desarrollo y guia de sistemas con
  estructura suficiente para mantenimiento ordinario.
- `profunda`: manuales desglosados por rol, explicacion de opciones elegidas,
  criterios de aceptacion, decisiones de arquitectura, contratos de datos,
  integraciones, pruebas, operacion, diagnostico y siguientes pasos.

Cuando usar `profunda`:

- apps que nacen para produccion, administracion publica, salud, formacion,
  finanzas, equipos no tecnicos, integraciones externas o continuidad por otros
  agentes;
- cuando el contrato de cierre exige que una persona pueda revisar la app sin
  leer codigo fuente;
- cuando se han seleccionado varias arquitecturas, tipos de datos, perfiles de
  accesibilidad o integraciones y hay que justificar su uso.

Errores frecuentes:

- usar `profunda` para compensar un contrato incompleto;
- pedir documentacion larga sin criterios de aceptacion ni comandos verificables;
- incluir secretos, rutas privadas o datos reales en manuales publicables.

Tooltip: `nueva_app.ayuda.documentacion.profundidad`.

### `documentacion.locales`

Tipo UI: input de lista por comas.

Ejemplos:

- `es-ES`
- `es-ES,en-US`
- `es-ES,ca-ES,gl-ES,eu-ES`

Default:

- vacio hereda `i18n.locales`;
- si tambien falta i18n, la factory puede mantener `es-ES`.

Que implica:

- controla idiomas de manuales y guias;
- puede ser distinto de los idiomas de la UI si el equipo solo necesita
  documentacion en un subconjunto;
- debe usar locales claros, no nombres ambiguos como `spanish`.

Errores frecuentes:

- poner idiomas no soportados por la composicion;
- pedir muchos idiomas sin justificarlo en producto;
- duplicar locales en `i18n.locales` y `documentacion.locales` con valores
  contradictorios.

Tooltip: `nueva_app.ayuda.documentacion.locales`.

## Paso 6: Revision, Agentes Y Restricciones

### `agentes.revision_humana`

Tipo UI: select booleano.

Opciones:

- `true`
- `false`

Default:

- `true` si falta.

Que implica:

- `true` mantiene revision humana antes de decisiones importantes;
- `false` permite mas continuidad, pero no elimina evidencias ni contratos.

Cuando usar `true`:

- datos sensibles, pagos, seguridad, produccion, integraciones criticas.

Cuando usar `false`:

- prototipo, documentacion, investigacion o tareas reversibles.

Errores frecuentes:

- desactivarla por comodidad en flujos de riesgo;
- pensar que `false` permite saltarse tests.

Tooltip: `nueva_app.ayuda.agentes.revision_humana`.

### `agentes.autonomia`

Tipo UI: select.

Opciones:

- `media`
- `baja`
- `alta`

Que implica:

- `baja`: Orquesta pregunta mas y decide menos.
- `media`: equilibrio por defecto.
- `alta`: Orquesta toma mas decisiones dentro de contratos y restricciones.

Cuando usar:

- `baja`: dominio ambiguo, riesgo alto, decisiones de producto abiertas.
- `media`: mayoria de apps internas o cambios normales.
- `alta`: objetivo claro, restricciones claras y decisiones reversibles.

Errores frecuentes:

- elegir `alta` sin objetivo verificable;
- elegir `baja` y no responder preguntas bloqueantes;
- confundir autonomia con permiso para tocar sistemas externos no autorizados.

Tooltip: `nueva_app.ayuda.agentes.autonomia`.

### `agentes.preferencias`

Tipo UI: input de lista por comas.

Usalo para:

- compactar contexto, priorizar tests, revisar seguridad, documentar decisiones,
  usar revisores, mantener cambios pequenos, generar evidencias.

Errores frecuentes:

- usarlo como prompt largo;
- pedir modelos/proveedores concretos como regla de nucleo;
- contradecir autonomia o calidad.

Tooltip: `nueva_app.ayuda.agentes.preferencias`.

### `restricciones`

Tipo UI: input de lista por comas.

Usalo para:

- plazos, exclusiones, tecnologias prohibidas, licencias, privacidad, reglas de
  producto, alcance fuera, entornos permitidos.

Errores frecuentes:

- meter secretos;
- meter requisitos funcionales principales que pertenecen a `objetivo`;
- usar restricciones para imponer DB o proveedor sin justificacion.

Tooltip: `nueva_app.ayuda.restricciones`.

## Campos Contractuales Visibles Que No Deben Perderse

El DTO local y el HTML basico ya comparten estos controles dedicados. No deben
perderse al ampliar la UI o reorganizar el wizard:

- `usuarios_objetivo`: perfiles de usuario.
- `preferencias_tecnicas.restricciones`: restricciones tecnicas separadas.
- `datos.retencion`: regla general de retencion del dato.

Recomendacion:

- mantenerlos en modo experto o en una seccion avanzada;
- cada campo visible necesita label i18n y tooltip;
- no duplicar el mismo dato bajo nombres distintos.

## Tooltips

Los tooltips del wizard salen de claves `nueva_app.ayuda.*` y se renderizan en
atributos `data-help`. Son ayudas breves para decision inmediata.
Con raton aparecen al pasar por encima o enfocar; en pantallas tactiles pueden
quedar abiertos al tocar el campo y se cierran tocando fuera o con Escape.

Relacion con esta guia:

- el tooltip dice que es el campo;
- esta guia explica implicaciones, cuando usarlo y errores frecuentes;
- si se anade una opcion experta, debe anadirse su clave i18n antes de mostrarla;
- no debe aparecer texto visible hardcodeado fuera del catalogo i18n.

Claves actuales por seccion:

- pasos: `step.idea`, `step.tipo`, `step.tecnologia`, `step.datos`,
  `step.calidad`, `step.revisar`;
- presets: `preset.webapp`, `preset.api`, `preset.ops`;
- identidad: `nombre`, `tipo_app`, `objetivo`, `descripcion`, `request_id`,
  `locale`, `request_kind`, `execution_mode`;
- origen: `plataformas.*`, `project_source.*`;
- tecnologia: `preferencias_tecnicas.*`, `i18n.*`;
- datos: `datos.db_required`, `datos.necesidad_funcional`,
  `datos.tipos_datos`, `datos.sensibilidad`, `datos.retencion`;
- datos expertos: `datos.tipos_detallados.*`, `datos.fuentes.*`,
  `datos.storage.*` y `datos.operacion.*`;
- integraciones: `integraciones.0.tipo`, `integraciones.0.nombre`,
  `integraciones.0.proposito`, `integraciones.0.requerido`,
  `integraciones.0.restricciones` y campos expertos indexados
  `integraciones.N.*`;
- calidad/deploy/agentes: `calidad.*`, `deploy.*`, `agentes.*`,
  `restricciones`.

Claves recomendadas para el bloque experto:

- `datos.experto.enabled`
- `datos.experto.tipo.nombre`
- `datos.experto.tipo.proposito`
- `datos.experto.tipo.sensibilidad`
- `datos.experto.tipo.retencion`
- `datos.experto.persistencia.preferencia`
- `datos.experto.persistencia.consistencia`
- `datos.experto.auditoria`
- `integraciones.experto.enabled`
- `integraciones.experto.add`
- `integraciones.experto.direccion`
- `integraciones.experto.datos_intercambiados`
- `integraciones.experto.frecuencia`
- `integraciones.experto.errores`
- `integraciones.experto.autenticacion`

## Defaults Y Normalizaciones Relevantes

- `schema_version` se fija como `app_spec_request.v0`.
- `source` se fija como `orquesta-web`.
- `request_kind` vacio se normaliza a `crear_app_completa`.
- `execution_mode` vacio se normaliza a `normal`.
- `preferencias_tecnicas.arquitectura` vacia debe caer al fallback/default
  `hexagonal`.
- Patrones distintos de `hexagonal` son opciones soportadas de producto/contrato
  de arquitectura; `hexagonal` es fallback, no obligacion.
- `i18n.default_locale` vacio hereda `locale`.
- `i18n.enabled` vacio se interpreta como `true`.
- `documentacion.*` vacio se interpreta como documentacion activada por defecto
  en factory.
- `documentacion.locales` vacio hereda locales i18n.
- `deploy.target` vacio se interpreta como `sin_preferencia`.
- `calidad.pruebas` vacio se interpreta como `basica`.
- `calidad.accesibilidad` vacio se interpreta como `basica` en factory v0.
- `calidad.accesibilidad=normal` es un nivel canonico soportado entre `basica`
  y `wcag_aa`.
- `calidad.accesibilidad_opciones` acepta niveles (`normal`, `wcag_aa`) y
  criterios adicionales (`teclado`, `lectores_pantalla`, `contraste_alto`,
  `movimiento_reducido`, `subtitulos_transcripciones`).
- `calidad.observabilidad` vacio se interpreta como `true`.
- `agentes.revision_humana` vacio se interpreta como `true`.
- `agentes.autonomia` vacio se interpreta como `media`.
- `project_source.kind` vacio se infiere por `git_url`, `local_path` o `new`.

## Errores Publicos Frecuentes

### `app_spec_invalida`

Suele aparecer por:

- falta `schema_version`, `request_id`, `source`, `locale`, `nombre`,
  `objetivo` o `tipo_app` en JSON;
- `tipo_app`, `request_kind`, `execution_mode`, `calidad.pruebas`,
  `calidad.accesibilidad` o `agentes.autonomia` no soportados;
- `db_required=true` sin `datos.necesidad_funcional`;
- `project_source` requerido pero ausente.

Como corregir:

- completa campos obligatorios;
- usa opciones documentadas;
- describe necesidad funcional en vez de proveedor.

### `opcion_incompatible`

Suele aparecer por:

- arquitectura no soportada por la composicion/adaptador actual;
- `i18n.enabled=false` sin justificacion;
- `project_source.git_url` y `project_source.local_path` a la vez;
- `project_source.kind=new` con URL o ruta local;
- URL Git con credenciales.

Como corregir:

- separa origen nuevo de existente;
- elimina credenciales;
- justifica excepciones.

### `target_no_soportado`

Suele aparecer por:

- `deploy.target` fuera de las opciones soportadas.

Como corregir:

- usa `sin_preferencia`, `local`, `contenedor`, `paas`, `serverless`,
  `kubernetes`, `desktop` o `mobile_store`.

### `idioma_invalido`

Suele aparecer por:

- locale no BCP 47.

Como corregir:

- usa valores como `es-ES` o `en-US`.

### `conector_requerido_no_disponible`

Suele aparecer por:

- integracion con credenciales, DSN o cadena de conexion;
- integracion que impone proveedor/backend en vez de capacidad;
- mezclar DB/cache/runtime como integracion directa.

Como corregir:

- describe capacidad y proposito;
- mueve restricciones no sensibles a `restricciones`;
- deja proveedor real para adaptador/composicion.

### Errores Locales Web

- `form_incompleto`: body no se pudo leer o content-type no soportado.
- `metodo_no_soportado`: metodo distinto de GET/POST/OPTIONS.
- `error_transporte`: fallo de cliente o endpoint externo.
- `respuesta_invalida`: respuesta no cumple contrato publico.
- `transporte_no_configurado`: no hay conector inyectado.
- `run_ref_requerido`: el arranque Goal-first no devolvio `run_ref`; no hay
  clave observable para cierre, evidencias ni tests y no se reactiva el loop
  legacy.
- `codex_app_server_*`: Codex Goal esta configurado pero el backend app-server
  esta degradado. Revisa socket, comando, permisos o instalacion standalone,
  reinicia `codex app-server daemon` y despues `orquesta-server`.

## Recetas De Uso

### App Web Interna Simple

- `tipo_app=web`
- plataformas: `web`
- `datos.db_required=true` si hay usuarios/configuracion/historico
- `calidad.pruebas=media`
- `calidad.accesibilidad=basica`
- `agentes.autonomia=media`

Usa modo experto solo si hay PII, auditoria o varias integraciones.

### Panel Operativo Con Auditoria

- preset Panel Operativo
- `datos.db_required=true`
- `datos.necesidad_funcional=auditar decisiones y estados`
- modo experto: tipo `eventos_auditoria` con retencion propia
- `calidad.observabilidad=true`
- `agentes.revision_humana=true`

### API Con Integraciones Externas

- preset API REST
- plataformas: `api`
- una fila experta por integracion
- `calidad.pruebas=alta`
- `deploy.target=contenedor` o `paas`
- describe retries, idempotencia y errores.

### App RAG O Busqueda Semantica

- `tipo_app=web` o `api`
- modo experto datos:
  - documentos como `objetos_blob` o `documental`;
  - embeddings como `vectorial`;
  - auditoria como `eventos_auditoria` si aplica.
- integraciones: fuente documental, importacion/exportacion y busqueda si son
  externas.

No escribas proveedor LLM ni vector DB como decision del nucleo.

### Cambio Sobre Repo Existente

- `request_kind=modificar_app_existente`, `analizar_app`, `revisar_codigo`,
  `seguridad` o `migracion_refactor`;
- `project_source.kind=github` o `local_path`;
- evita `new`;
- `execution_mode=normal` salvo diagnostico acotado.

## Checklist Antes De Enviar

- El objetivo tiene usuario, resultado y criterio de exito.
- `tipo_app` y plataformas no estan sobredimensionados.
- Si hay repo existente, `project_source` es coherente y no contiene secretos.
- La arquitectura queda elegida o cae explicitamente al fallback `hexagonal`.
- i18n desactivado, si aplica, tiene justificacion.
- Si `db_required=true`, hay necesidad funcional.
- En modo experto, cada tipo de dato tiene sensibilidad y retencion cuando
  importa.
- Preferencias de persistencia son formas genericas, no proveedores.
- Cada integracion tiene proposito, direccion y requerido/opcional.
- No hay credenciales, DSN, tokens, HOME ni rutas privadas innecesarias.
- Calidad y deploy corresponden al riesgo real.
- La autonomia de agentes encaja con riesgo y revision humana.
- Las restricciones no contradicen el objetivo.
