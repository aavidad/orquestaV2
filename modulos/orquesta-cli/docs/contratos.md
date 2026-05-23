# Contratos locales: orquesta-cli

Registra puertos, DTOs y eventos que `orquesta-cli` expone o consume.

## Principios locales

- La CLI consume contratos publicos; no define negocio.
- El transporte es adaptador: REST inicial, otros transportes solo si el contrato
  compartido los habilita.
- Toda salida automatizable debe usar envelope estable.
- Las opciones CLI no pueden introducir campos no contratados.
- Los detalles de DB, tablas, runtime, modelos, HOME, tokens, prompts y
  transcripts completos estan prohibidos.

## Contratos locales iniciales

```text
Nombre: CliInvocationContextV0
Tipo: dto
Version: v0
Propietario: orquesta-cli
Consumidores: comandos locales de orquesta-cli
Campos:
  - command: ruta logica del comando ejecutado.
  - request_id: identificador de request, generado si falta.
  - correlation_id: identificador para propagar a servicios publicos.
  - idempotency_key: requerido solo en comandos mutantes contractuales.
  - server_url: endpoint publico configurado; no contiene credenciales.
  - timeout: limite tecnico de llamada.
  - output_format: text | json | tsv.
  - locale: locale de salida cuando haya texto localizado.
  - dry_run: bool, solo si el contrato consumido lo admite.
  - input_source: stdin | file | arg, sin rutas absolutas en la salida publica.
Invariantes:
  - No contiene secretos, tokens, rutas HOME reales ni DSN.
  - No autoriza fallback local.
  - No crea campos de negocio fuera del contrato consumido.
Errores:
  - opcion_invalida
  - configuracion_cli_invalida
  - contrato_no_configurado
Pruebas de contrato:
  - Validar generacion y propagacion de request_id/correlation_id.
  - Validar rechazo de server_url con credenciales embebidas.
  - Validar que --dry-run solo se acepta si el contrato remoto lo declara.
```

```text
Nombre: CliOutputEnvelopeV0
Tipo: dto
Version: v0
Propietario: orquesta-cli
Consumidores: scripts, CI, operadores, tests de smoke
Campos:
  - ok: bool.
  - request_id: eco del contexto de invocacion.
  - correlation_id: eco propagado.
  - contract: nombre del contrato publico consumido.
  - version: version del contrato publico o local.
  - data: payload publico serializable.
  - errores: lista de errores publicos con codigo, campo, mensaje_i18n y detalle acotado.
  - warnings: lista de advertencias publicas acotadas.
  - meta: transporte, duracion_ms, status_code y retryable cuando aplique.
Invariantes:
  - El envelope no incluye cuerpo privado remoto en errores no 2xx.
  - `data` mantiene el shape canonico del contrato propietario.
  - Los mensajes humanos son secundarios frente a codigos estables.
Errores:
  - respuesta_invalida
  - error_transporte
  - salida_no_serializable
Pruebas de contrato:
  - Golden JSON estable para exito, validacion 400 y error de transporte.
  - Verificacion de ausencia de secretos y rutas absolutas.
```

```text
Nombre: SolicitarNuevaAppCliClientV0
Tipo: puerto_salida
Version: v0
Propietario: orquesta-cli
Consumidores: comando previsto `app spec solicitar`
Campos:
  - Entrada: AppSpecRequestV0 de `SolicitarNuevaApp v0`.
  - Salida correcta: envelope REST canonico con `app_spec` y `backlog`.
  - Salida error: errores publicos del contrato o errores de transporte CLI.
  - Headers: Content-Type, Accept, X-Correlation-ID.
Invariantes:
  - Consume el contrato canonico de `orquesta-factory`.
  - No valida reglas de negocio que pertenezcan a factory.
  - No persiste AppSpecV0 ni BacklogInicialPropuestoV0.
  - Acepta alias transitorios solo como respuesta de compatibilidad, no como shape propio.
Errores:
  - app_spec_invalida
  - opcion_incompatible
  - target_no_soportado
  - idioma_invalido
  - conector_requerido_no_disponible
  - error_transporte
  - respuesta_invalida
Pruebas de contrato:
  - Fixture minima valida de factory produce `ok=true`.
  - Fixture invalida de factory produce `ok=false` con errores publicos.
  - Respuesta 500 se traduce a `error_transporte` sin filtrar body privado.
```

```text
Nombre: OperationalStatusCliClientV0
Tipo: puerto_salida
Version: v0
Propietario: orquesta-cli
Consumidores: comando previsto `doctor contratos`
Campos:
  - Entrada: OperationalStatusQueryV0 de `orquesta-observability`.
  - Salida correcta: envelope REST canonico con `DiagnosticoCompactoV0` read-only.
  - Salida error: errores publicos del contrato o errores de transporte CLI.
  - Headers: Content-Type, Accept, X-Correlation-ID.
Invariantes:
  - Consume el DTO/validador compartido de `orquesta-observability`.
  - La CLI fuerza `schema_version`, `request_id`, `correlation_id` y `consumer` desde `CliInvocationContextV0`.
  - No persiste diagnosticos, no lee DB/runtime/filesystem y no ejecuta recuperacion activa.
  - La salida mantiene el DTO canonico `DiagnosticoCompactoV0`; no crea shape paralelo.
Errores:
  - operational_status_query_invalida
  - consumidor_no_autorizado
  - scope_no_soportado
  - referencia_no_opaca
  - consulta_demasiado_amplia
  - proyeccion_no_disponible
  - diagnostico_no_disponible
  - frescura_no_garantizada
  - secreto_detectado
  - transcript_no_permitido
  - error_transporte
  - respuesta_invalida
Pruebas de contrato:
  - Query valida produce `ok=true` con `DiagnosticoCompactoV0` canonico.
  - Respuesta 400 conserva errores publicos del contrato.
  - Respuesta 500/timeout se traduce a `error_transporte` sin filtrar body privado.
  - JSON invalido o diagnostico no valido devuelve envelope de error sin internals.
```

```text
Nombre: CliRESTTransportAdapterV0
Tipo: adaptador_tecnico_local
Version: v0
Propietario: orquesta-cli
Consumidores: SolicitarNuevaAppCliClientV0, OperationalStatusCliClientV0 y futuros clientes REST finos
Campos:
  - BaseURL: base HTTP/HTTPS normalizada y sin credenciales.
  - Endpoint: ruta versionada del contrato publico consumido.
  - Timeout: timeout tecnico efectivo del request.
  - HTTPClient: cliente HTTP inyectable para tests o configuracion.
  - Request builder: POST JSON con `Content-Type`, `Accept` y `X-Correlation-ID`.
Invariantes:
  - No contiene logica de negocio, validadores funcionales ni shape de contratos remotos.
  - No interpreta cuerpos 2xx/4xx/5xx; eso sigue en el cliente por contrato.
  - No accede a DB, runtime, filesystem ni fallback local.
  - Reutiliza solo reglas tecnicas comunes: base URL, timeout, endpoint y headers canonicos.
Errores:
  - configuracion_cli_invalida
  - error_transporte
Pruebas de contrato:
  - Normaliza `server_url` y rechaza credenciales embebidas.
  - Usa timeout por defecto si no se indica otro.
  - Construye POST JSON con `X-Correlation-ID`.
  - Reutiliza `http.Client` inyectado cuando existe.
```

```text
Nombre: FunctionContractCliClientV0
Tipo: puerto_salida
Version: v0
Propietario: orquesta-cli
Consumidores: comandos previstos `contratos funcion listar|ver|registrar`
Campos:
  - Entrada listar: `request_id`, `correlation_id`, filtros `modulo|estado|archivo_objetivo|simbolo_objetivo` y page `limit|cursor`.
  - Entrada ver: `request_id`, `correlation_id`, `function_contract_ref` opaco y `version` opcional.
  - Transporte REST candidato read-only: `POST /api/v0/core/function-contracts/list` y `POST /api/v0/core/function-contracts/view`.
  - Salida listar: `ListarFunctionContractsResultV0` con `FunctionContractResumenV0`, `next_cursor` y `warnings`.
  - Salida ver: `VerFunctionContractResultV0` con `FunctionContractV0` publico de `orquesta-core` y `warnings`.
  - Registrar: sin transporte; devuelve bloqueo publico `registrar_function_contract_bloqueado`.
  - Salida error: `FunctionContractErrorV0` publico de core o errores CLI `error_transporte|respuesta_invalida`.
Invariantes:
  - La CLI no valida microtareas por su cuenta.
  - La CLI no cruza `internal/` de core ni lee tareas historicas V1.
  - Todo write-set mostrado debe venir del contrato publico.
  - Listar y ver son solo lectura: no crean, actualizan, archivan, reemplazan, agendan workflow ni escriben Outbox.
  - Si el servidor no expone la ruta, el adaptador devuelve `error_transporte`; no usa fallback local.
  - El adaptador solo valida shape de respuesta serializable y referencias minimas, no reglas de negocio de FunctionContract.
Errores:
  - function_contract_incompleto
  - consulta_function_contract_no_disponible
  - filtro_no_soportado
  - cursor_invalido
  - function_contract_no_encontrado
  - operacion_no_promovida
  - registrar_function_contract_bloqueado
  - estado_no_ejecutable
  - error_transporte
  - respuesta_invalida
Pruebas de contrato:
  - Tests con `httptest` cubren listar ok, ver ok, 400 publico, respuesta invalida, timeout, rechazo de `server_url` con credenciales y registrar bloqueado.
Estado:
  - Completado ejecutable para listar/ver read-only mediante rutas REST candidatas; registrar sigue bloqueado por contrato.
```

```text
Nombre: GovernanceCatalogCliReaderV0
Tipo: puerto_salida
Version: v0
Propietario: orquesta-cli
Consumidores: comandos previstos `gobernanza catalogo listar|ver`
Campos:
  - Entrada: `GovernanceCatalogQueryV0` de `orquesta-governance`, restringido a filtros compactos `module`, `role`, `phase` y `tags`.
  - Transporte REST asumido: `POST /api/v0/governance/catalog/query`.
  - Request body canonico del adaptador: JSON compacto `{module, role, phase, tags}`.
  - Salida correcta: `GovernanceCatalogQueryResultV0` con `effective` y `counters.effective|proposed|quarantine`.
  - Salida error: `errores[]` publicos compactos con `codigo` y `campo`, o `error_transporte`/`respuesta_invalida` del adaptador CLI.
  - Headers: Content-Type, Accept, X-Correlation-ID.
Invariantes:
  - Solo lectura.
  - `effective` es el unico bloque tratable como regla vigente.
  - `proposed` y `quarantine` se muestran como evidencia, no se activan.
  - La CLI no devuelve `catalogs.proposed` ni `catalogs.quarantine` completos; solo consume `effective` y contadores compactos.
  - La CLI valida que toda entrada en `effective` pase `ValidateEffectiveGovernanceEntryV0`.
  - `counters.effective` debe coincidir con el numero de entradas `effective` devueltas.
  - El adaptador no lee docs locales, DB, runtime ni filesystem como fallback si la ruta no responde.
Errores:
  - governance_catalog_source_unavailable
  - governance_catalog_invalid_source
  - governance_catalog_entry_without_origin
  - governance_catalog_entry_without_promotion_criteria
  - governance_catalog_forbidden_activation
  - governance_catalog_secret_detected
  - governance_catalog_text_only_secret_control
  - governance_entry_effective_without_decision
  - governance_entry_text_only_secret_control
  - error_transporte
  - respuesta_invalida
Pruebas de contrato:
  - Query valida produce `ok=true`, propaga `X-Correlation-ID` y devuelve `GovernanceCatalogQueryResultV0` canonico.
  - Respuesta 400 conserva errores publicos compactos.
  - Respuesta 500/timeout se traduce a `error_transporte` sin filtrar body privado.
  - JSON invalido o `effective` no valido devuelve `respuesta_invalida`.
Acoplamiento minimo documentado:
  - Mientras `orquesta-governance` no publique aun la ruta, la CLI asume `POST /api/v0/governance/catalog/query` como endpoint REST compacto recomendado para `GovernanceCatalogQueryV0`.
```

```text
Nombre: CompatV1InventoryV0
Tipo: dto
Version: v0
Propietario: orquesta-cli
Consumidores: documentacion y planificacion CLI
Campos:
  - familia_v1: comando o familia observada en V1.
  - evidencia_v1: referencia textual corta a comando o flag observado.
  - flags_v1: flags reutilizables como UX.
  - categoria_v2: reutilizable | traducible | cuarentena | descartado.
  - contrato_v2_requerido: contrato publico necesario para implementarlo.
  - motivo: razon breve.
Invariantes:
  - Es inventario documental, no compatibilidad ejecutable.
  - No copia codigo, structs ni rutas `cmd/` de V1.
  - Ningun comando en cuarentena se implementa sin contrato nuevo.
Errores:
  - inventario_incompleto
  - contrato_v2_requerido
Pruebas de contrato:
  - Revision documental confirma que flags reutilizados tienen contrato V2.
```

```text
Nombre: CliI18nPolicyV0
Tipo: politica_local
Version: v0
Propietario: orquesta-cli
Consumidores: adaptadores CLI presentes y futuros
Campos:
  - default_locale: `es`
  - supported_locales_iniciales: `es`, `en`
  - fallback_chain: locale pedido -> locale base -> default_locale
  - catalog_scope: local al modulo mientras no exista contrato compartido para adaptadores
  - string_policy: ningun texto visible nuevo sale sin clave de catalogo o codigo estable
  - message_shape: `codigo` + `mensaje_i18n` + detalle acotado opcional
Invariantes:
  - La traduccion vive en el adaptador CLI; nunca redefine reglas de negocio ni semantica contractual.
  - Los contratos compartidos siguen siendo canonicos por `codigo`, campos y shape; la CLI solo localiza presentacion.
  - Si falta una traduccion concreta, se permite fallback controlado por cadena; nunca texto libre improvisado.
  - No se introduce un contrato compartido nuevo para catalogos CLI en este slice.
  - Ayuda, warnings y errores visibles de comandos nuevos deben referenciar clave de catalogo local o codigo estable.
Errores:
  - locale_no_soportado
  - catalogo_cli_incompleto
  - clave_i18n_ausente
Pruebas de contrato:
  - Locale por defecto `es` cuando no se informa `--locale`.
  - `--locale en` conserva mismo `codigo` y mismo shape contractual.
  - Si falta clave en `en`, fallback a `es` sin cambiar `codigo` ni exponer internals.
  - Revision documental o automatizada detecta strings visibles nuevas fuera de catalogo o codigo.
Estado:
  - Local al modulo; no promovido a contrato compartido.
```

```text
Nombre: AutoprogrammingCliClientV0
Tipo: puerto_salida
Version: v0
Propietario: orquesta-cli
Consumidores: comandos `servidor estado`, `autoprogramacion preparar|cola listar|run ver`
Campos:
  - `servidor estado`: GET `/api/v0/server/status`.
  - `preparar`: POST `/api/v0/autoprogramming/prepare-run` con `MCPAutoprogrammingPrepareRunToolInputV0`.
  - `cola listar`: POST `/api/v0/runs/queue/priority` con action `rank`.
  - `run ver`: POST `/api/v0/director/stats` con `run_ref` opaco.
Invariantes:
  - Cliente fino server-first; no lee stores, runtime, worktrees, DB ni filesystem interno.
  - `branch_ref`, `worktree_ref` y `run_ref` se tratan como refs opacas y no se recomputan en CLI.
  - La CLI no arranca agentes ni supervisor por si misma; solo consume endpoints publicos.
Errores:
  - error_transporte
  - respuesta_invalida
  - opcion_invalida
Pruebas de contrato:
  - httptest valida rutas, headers, rechazo de respuestas invalidas y preservacion de refs opacas en prepare-run.
Estado:
  - Completado ejecutable como adaptador secundario.
```

## Inventario V1 resumido

| Familia V1 | Evidencia V1 observada | Flags V1 reutilizables | Categoria V2 | Contrato V2 requerido | Motivo |
| --- | --- | --- | --- | --- | --- |
| `app spec solicitar` | V1 tenia superficies tipo wizard/fabrica y carga por input; V2 fija este flujo como primer comando real | `--input`, `--json`, `--correlation-id`, `--timeout` | reutilizable | `SolicitarNuevaApp v0` | Ya existe contrato compartido y cliente CLI fino; no requiere fallback local. |
| `doctor contratos` | V1 tenia comandos de estado/diagnostico (`status`, `runtime diagnostico`, `exportar diagnostico`) | `--json`, `--correlation-id`, `--timeout` | reutilizable | `OperationalStatusQuery v0` + `DiagnosticoCompactoV0` | V2 lo reduce a consulta compacta read-only sin DB/runtime directo. |
| `contratos funcion listar|ver` | V2 documenta esta familia como prevista y CLI consume rutas REST candidatas read-only | `--json`, `--limit` | reutilizable | `FunctionContract v0` | Cliente fino sin fallback local; si core no publica la ruta devuelve error de transporte. |
| `contratos funcion registrar` | V2 la contempla como familia prevista, pero sin transporte ni ciclo durable cerrados | `--json`, `--input`, `--correlation-id`, `--timeout` | descartado | sin contrato vigente | No inventar write path antes de `OrchestrationRun/CommandHandler/Outbox` y promocion global. |
| `gobernanza catalogo listar|ver` | V1 tenia superficies de gobernanza/reglas/web; V2 solo admite lectura secundaria | `--json`, `--limit` | traducible | `GovernanceCatalog v0` | Falta puerto publico de `orquesta-governance`; no se reactiva logica local. |
| `runtime diagnostico` | `./orquesta runtime diagnostico --agente Codex2 --limit 5` en docs V1 | `--json`, `--limit`, `--agente`, `--proyecto` | cuarentena | futuro contrato read-only mas especifico, si hiciera falta | Parte del control plane viejo y mezclaba detalles de runtime; V2 usa `doctor contratos` compacto. |
| `runtime ordenes`, `runtime mailbox`, `runtime transcript`, `runtime checkpoints` | comandos observados repetidamente en docs y `cmd/runtime.go` | `--limit`, `--estado`, `--agente`, `--proyecto`, `--desde` | cuarentena | contratos publicos futuros por capacidad/runtime/observability | V1 inspeccionaba internals operativos; no se copia esa superficie. |
| `runtime purgar-*`, `runtime orden-nueva`, `runtime procesar-*`, `runtime nudge`, `runtime discordia` | presentes en `cmd/runtime.go` | `--estado`, `--tipo`, `--actor`, `--wait`, `--payload` | descartado | sin contrato vigente | Son mutaciones de runtime/control plane; fuera del alcance CLI V2 y prohibidas sin adaptador hexagonal publico. |
| `agente preparar`, `agente tick`, `agente ejecutar`, `agente handoff`, `agente reasignar-vivo`, `agente pausar` | presentes en `cmd/agente.go` | `--json`, `--proyecto`, `--conector`, `--modelo`, `--motivo` | descartado | sin contrato vigente | Mezclaban seleccion de modelo, cuota, sesiones, handoff y runtime local; deben vivir detras del futuro nucleo/ports, no en CLI. |
| `tarea *`, `repo *`, `worktree *`, `microprogramacion *` | familias V1 observadas en `cmd/` y docs historicas | `--json`, `--estado`, `--proyecto`, `--agente` | cuarentena | futuros contratos core/factory | El naming puede reutilizarse, pero la implementacion V1 estaba acoplada a DB/git/worktree y no es portable tal cual. |
| `pool *`, `modelo *`, `conector *` | familias V1 observadas en `cmd/` | `--json`, `--proveedor`, `--modelo`, `--runtime` | descartado | sin contrato vigente | Introducian politica de capacidad/modelos en la CLI; V2 lo mueve a conectores y decisiones del sistema. |
| `deploy *`, `persistencia *`, `respaldo *`, `server *`, `serve *` | familias V1 observadas en `cmd/` | `--json`, `--dry-run`, `--older-than-minutes` | descartado | sin contrato vigente | Ejecutaban efectos de sistema o acceso local; la CLI V2 solo puede consumir planes/diagnosticos publicos. |

### Flags V1 reutilizables como patron, no como derecho adquirido

| Flag/familia V1 | Estado en V2 | Contrato minimo | Nota |
| --- | --- | --- | --- |
| `--json` | reutilizable | `CliOutputEnvelopeV0` | Ya respaldado por CLI-001/002/006. |
| `--tsv` | traducible | `CliOutputEnvelopeV0` + comando read-only tabular | No implementado; se conserva solo como opcion de automatizacion futura. |
| `--limit` | traducible | contrato consumido con paginacion o truncado explicito | No se expone donde el contrato no lo soporte. |
| `--estado` | traducible | enum publico del contrato remoto | Prohibido si el estado sale solo de tablas internas V1. |
| `--proyecto` | traducible | campo publico del contrato remoto | Se mantiene solo cuando el contrato lo nombre. |
| `--agente` | cuarentena | futuro contrato de capacity/runtime/observability | No existe contrato compartido V2 para identidad operativa de agente. |
| `--desde` | cuarentena | futuro contrato de observability con rango temporal | No se recrea sobre logs o transcript local. |
| `--dry-run` | traducible | contrato remoto que lo declare | La CLI no inventa `dry_run` si el puerto no lo soporta. |
| `--input` | reutilizable | contrato remoto con carga estructurada | Ya respaldado por `SolicitarNuevaApp v0`. |
| `--correlation-id` | reutilizable | `CliInvocationContextV0` | Ya respaldado por CLI-001/002/006. |
| `--timeout` | reutilizable | `CliInvocationContextV0` | Ya respaldado como concern tecnico de transporte. |

## Consultas al director

```text
CONSULTA AL DIRECTOR
Modulo origen: orquesta-cli
Modulos afectados: orquesta-core, orquesta-observability, orquesta-runtime, orquesta-persistence
Bloqueo: la responsabilidad local menciona diagnostico y recuperacion, pero no hay contrato compartido que autorice a CLI a consultar estado operativo compacto sin internals.
Pregunta concreta: debe existir un `OperationalStatusQuery v0`/`DiagnosticoCompacto v0` publico para CLI, o el diagnostico CLI debe esperar a recursos MCP/web futuros?
Opcion recomendada: crear un contrato read-only compacto propiedad de observability o core, con referencias opacas y sin DB/runtime directo.
Impacto: desbloquea `doctor contratos` y comandos de recuperacion sin violar server-first.
Decision del director: promover `OperationalStatusQuery v0` como contrato compartido read-only, propiedad de `orquesta-observability`. CLI puede preparar diagnostico compacto contra ese contrato; recuperacion activa queda fuera.
Estado: resuelta
```

```text
CONSULTA AL DIRECTOR
Modulo origen: orquesta-cli
Modulos afectados: orquesta-core
Bloqueo: `FunctionContract v0` autoriza a CLI como adaptador fino para consultar o registrar contratos, pero el transporte/operaciones publicas no estan definidos en el contrato global.
Pregunta concreta: que operaciones publicas debe consumir CLI para listar, ver y registrar FunctionContractV0?
Opcion recomendada: definir endpoints o puerto in-process versionado antes de cualquier comando CLI.
Impacto: evita reintroducir microprogramacion hardcodeada o tareas historicas V1.
Decision del director: core documenta `listar/ver FunctionContractV0` como operaciones candidatas de solo lectura; `registrar` queda bloqueado hasta cerrar OrchestrationRun/CommandHandler/Outbox y promocion global.
Estado: resuelta para cliente CLI read-only; registrar sigue bloqueado
```

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
