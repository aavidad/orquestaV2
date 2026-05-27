# Contratos locales: orquesta-core

Registra puertos, DTOs y eventos que `orquesta-core` expone o consume.

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

## Propuesta v0: registrar proyecto desde AppSpec validada

Estado implementacion 2026-05-04:

- Implementacion pura en `registrar_proyecto_appspec_v0.go`.
- Importa DTOs publicos de factory con alias `orquestafactory "orquesta/modulos/orquesta-factory"`.
- No persiste, no crea IDs de DB, no asigna agentes, no consulta capacidad y no arranca runtime.
- La idempotencia v0 es determinismo logico por `idempotency_key`.

```text
Nombre: RegistrarProyectoDesdeAppSpec
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-core
Consumidores: orquesta-factory
Campos:
  - comando: RegistrarProyectoDesdeAppSpecCommandV0
  - salida_ok: RegistroProyectoAceptadoV0
  - salida_error: RegistrarProyectoDesdeAppSpecErrorV0
Invariantes:
  - Solo acepta AppSpecV0 ya validada por orquesta-factory.
  - Solo acepta BacklogInicialPropuestoV0 ya propuesto por orquesta-factory.
  - No persiste en DB en v0; devuelve un resultado de dominio serializable para que un adaptador futuro lo persista.
  - No arranca runtime, no asigna agentes y no consulta capacidad.
  - No revalida reglas internas de factory; core valida compatibilidad minima para construir un ProyectoPlan futuro.
  - La operacion debe ser idempotente respecto a idempotency_key.
  - El resultado debe poder evolucionar a ProyectoPlan sin depender de HTTP, CLI, MCP, DB o filesystem.
Errores:
  - app_spec_requerida
  - app_spec_no_validada
  - backlog_requerido
  - backlog_vacio
  - backlog_incompatible_con_app_spec
  - idempotency_key_requerida
  - contrato_factory_no_soportado
Pruebas de contrato:
  - Acepta AppSpecV0 validada y BacklogInicialPropuestoV0 compatible.
  - Rechaza AppSpecV0 sin marca/evidencia de validacion.
  - Rechaza backlog vacio o incompatible con el objetivo de AppSpecV0.
  - La misma idempotency_key produce el mismo resultado logico.
```

```text
Nombre: RegistrarProyectoDesdeAppSpecCommandV0
Tipo: dto
Version: v0
Propietario: orquesta-core
Consumidores: orquesta-factory
Campos:
  - app_spec: AppSpecV0
  - app_spec_validada: derivada de app_spec.validation.estado == "valida"
  - app_spec_version: string
  - backlog_inicial_propuesto: BacklogInicialPropuestoV0
  - backlog_version: string
  - origen: string
  - correlation_id: string
  - request_id: string
  - idempotency_key: string
  - solicitado_en: instant
Invariantes:
  - app_spec_version debe ser "v0".
  - backlog_version debe ser "v0".
  - origen identifica el adaptador o caso de uso llamante, no un proveedor tecnico de runtime.
  - correlation_id e idempotency_key son opacos para core.
  - app_spec y backlog se tratan como contratos publicos de factory; core no depende de tipos privados de factory.
Errores:
  - app_spec_requerida
  - app_spec_no_validada
  - backlog_requerido
  - idempotency_key_requerida
  - contrato_factory_no_soportado
Pruebas de contrato:
  - Fixture minimo con app_spec_version v0 y backlog_version v0.
  - Fixture invalido con version no soportada.
```

```text
Nombre: RegistroProyectoAceptadoV0
Tipo: dto
Version: v0
Propietario: orquesta-core
Consumidores: orquesta-factory, orquesta-web, orquesta-mcp, orquesta-observability
Campos:
  - registro_id: string
  - proyecto_plan_borrador: ProyectoPlanBorradorV0
  - eventos_dominio: list<EventoDominioCoreV0>
  - warnings: list<string>
Invariantes:
  - registro_id es estable para la misma idempotency_key.
  - proyecto_plan_borrador no confirma persistencia.
  - warnings no bloquean la aceptacion; los errores publicos si.
  - eventos_dominio son datos de dominio para publicar por un adaptador futuro, no llamadas directas a observability.
Errores:
  - backlog_incompatible_con_app_spec
Pruebas de contrato:
  - Salida serializable sin referencias a DB/runtime.
  - Incluye al menos un evento de dominio de proyecto registrado en borrador.
```

```text
Nombre: ProyectoPlanBorradorV0
Tipo: dto
Version: v0
Propietario: orquesta-core
Consumidores: orquesta-factory, orquesta-web, orquesta-mcp
Campos:
  - proyecto_id_propuesto: string
  - nombre: string
  - estado: borrador
  - app_spec_ref: string
  - request_id: string
  - fases_iniciales: list<FasePlanificadaV0>
  - microtareas: list<ItemBacklogCoreV0>
  - backlog_normalizado: list<ItemBacklogCoreV0>
  - contratos_requeridos: list<string>
  - criterios_cierre: list<string>
  - dependencias_pendientes: list<string>
Invariantes:
  - Es un borrador; no sustituye al contrato global ProyectoPlan pendiente.
  - Debe conservar trazabilidad hacia AppSpecV0 y BacklogInicialPropuestoV0.
  - Las fases iniciales usan vocabulario de core, no pasos de runtime.
  - backlog_normalizado no contiene instrucciones de proveedor ni detalles de DB.
Errores:
  - backlog_vacio
  - backlog_incompatible_con_app_spec
Pruebas de contrato:
  - Mapea backlog propuesto a items de core con ids estables.
  - Mantiene dependencias y criterios de cierre cuando vienen informados.
```

```text
Nombre: FasePlanificadaV0
Tipo: dto
Version: v0
Propietario: orquesta-core
Consumidores: orquesta-factory, orquesta-web, orquesta-mcp
Campos:
  - id: string
  - nombre: string
  - orden: integer
  - estado: pendiente
  - criterios_entrada: list<string>
  - criterios_salida: list<string>
Invariantes:
  - orden es unico dentro de ProyectoPlanBorradorV0.
  - estado inicial siempre es pendiente.
  - No contiene comandos de runtime ni asignaciones de agentes.
Errores:
  - fase_duplicada
  - fase_invalida
Pruebas de contrato:
  - Orden estable para fases iniciales.
```

```text
Nombre: ItemBacklogCoreV0
Tipo: dto
Version: v0
Propietario: orquesta-core
Consumidores: orquesta-factory, orquesta-web, orquesta-mcp
Campos:
  - id: string
  - titulo: string
  - descripcion: string
  - fase_id: string
  - modulo_sugerido: string
  - prioridad: string
  - dependencias: list<string>
  - criterios_aceptacion: list<string>
  - contrato_requerido: string
  - write_set_previsto: list<string>
  - fuente_backlog_id: string
Invariantes:
  - id es estable dentro del proyecto.
  - fuente_backlog_id conserva trazabilidad al BacklogInicialPropuestoV0.
  - modulo_sugerido no concede permisos ni write-set.
  - criterios_aceptacion no sustituyen a FunctionContract.
Errores:
  - item_backlog_invalido
  - dependencia_backlog_invalida
Pruebas de contrato:
  - Mapeo conserva prioridad, dependencias y criterios de aceptacion cuando existan.
  - Mapeo conserva contrato requerido y write-set previsto como datos de planificacion, no como permisos efectivos.
```

```text
Nombre: EventoDominioCoreV0
Tipo: evento
Version: v0
Propietario: orquesta-core
Consumidores: orquesta-observability mediante adaptador futuro
Campos:
  - tipo: string
  - proyecto_id: string
  - occurred_at: instant
  - payload_version: string
  - payload: object
  - correlation_id: string
Invariantes:
  - Evento de dominio local; el puerto de salida `OrquestaEventPublisherPortV0` lo mapea a `OrquestaEvent v0`.
  - No contiene secretos ni transcripts completos.
  - No se publica directamente desde core.
Errores:
  - evento_invalido
Pruebas de contrato:
  - Evento minimo ProyectoRegistradoEnBorradorV0 serializable.
  - Mapper puro a OrquestaEvent v0 conserva correlation object, subject project id y payload compacto.
```

```text
Nombre: RegistrarProyectoDesdeAppSpecErrorV0
Tipo: error
Version: v0
Propietario: orquesta-core
Consumidores: orquesta-factory
Campos:
  - code: string
  - message: string
  - field: string
  - retryable: boolean
  - correlation_id: string
Invariantes:
  - code debe pertenecer a la lista de errores publicos del puerto.
  - message es diagnostico corto, no texto de UI final.
  - field apunta al campo contractual cuando aplique.
  - No contiene stack traces ni detalles internos.
Errores:
  - error_code_invalido
Pruebas de contrato:
  - Cada error publico del puerto tiene fixture minimo.
```

## Puertos de salida v0 para contratos globales promovidos

Estado implementacion 2026-05-04:

- Interfaces y DTOs publicos locales en `puertos_salida_v0.go`.
- Tests en `puertos_salida_v0_test.go`.
- No implementan DB, runtime, event sink, governance real ni adaptadores.
- No importan paquetes de `orquesta-persistence`, `orquesta-runtime`, `orquesta-observability` ni `orquesta-governance`.
- El contrato compartido minimo sigue siendo `../../CONTRATOS.md`; este documento fija como core consume esos contratos desde puertos de salida.

```text
Nombre: PersistenceRepositoryPortV0
Tipo: puerto_salida
Version: v0
Propietario: orquesta-core
Implementador esperado: orquesta-persistence mediante adaptador futuro de PersistenceRepository v0
Consumidores: casos de uso de orquesta-core que necesiten persistir borradores ya aceptados
Metodo:
  - GuardarProyectoBorradorV0(context, GuardarProyectoBorradorPersistenceRequestV0) -> GuardarProyectoBorradorPersistenceAcceptedV0 | PersistenceRepositoryErrorV0
Campos request:
  - schema_version: PersistenceRepositoryV0
  - contrato: PersistenceRepositoryV0
  - operacion: guardar_proyecto_borrador
  - request_id: string
  - correlation_id: string
  - idempotency_key: string
  - payload_version: ProyectoPlanBorradorV0
  - payload: ProyectoPlanBorradorV0
Campos accepted:
  - status: persistido
  - proyecto_id: string opaco tecnico
  - idempotency_key: string
  - payload_version: ProyectoPlanBorradorV0
  - created_at: instant tecnico opcional
  - updated_at: instant tecnico opcional
Invariantes:
  - Core entrega un request de dominio versionado; no construye el envelope global completo de persistence.
  - El adaptador de salida de persistence construye el envelope global `PersistenceRepositoryV0`, anyade `adapter_metadata`, abre/asocia `unidad_trabajo` activa y mapea el request de core al objeto `request` del schema global.
  - Core no entrega SQL, tablas, dialecto, driver, conexion ni unidad de trabajo concreta.
  - `proyecto_id` devuelto es tecnico y opaco; no sustituye `proyecto_id_propuesto` de dominio.
  - Idempotencia obligatoria por idempotency_key.
  - El adaptador de persistence abre/cierra transaccion y decide dialecto fuera de core.
Errores:
  - persistencia_no_disponible
  - payload_invalido
  - dialecto_no_soportado
  - conflicto_idempotencia
  - transaccion_fallida
  - regla_negocio_en_adaptador
Pruebas de contrato:
  - Mapper puro desde RegistroProyectoAceptadoV0 conserva schema_version, contrato, operacion, request_id, correlation_id, idempotency_key y payload borrador.
  - Serializacion del request no contiene DSN, SQL, tablas, dialecto ni comandos de adaptador.
```

```text
Nombre: OrquestaEventPublisherPortV0
Tipo: puerto_salida
Version: v0
Propietario: orquesta-core
Implementador esperado: orquesta-observability mediante adaptador futuro de OrquestaEvent v0
Consumidores: casos de uso de orquesta-core que devuelvan eventos de dominio compactos
Metodo:
  - PublicarOrquestaEventV0(context, PublishOrquestaEventRequestV0) -> PublishOrquestaEventAcceptedV0 | OrquestaEventPublishErrorV0
Campos request:
  - request_id: string
  - correlation_id: string
  - idempotency_key: string derivado para el evento
  - event: OrquestaEventV0
  - dry_run: boolean opcional
Campos event:
  - schema_version: orquesta_event.v0
  - event_id: string opaco
  - event_type: string con prefijo core.
  - source_area: core
  - occurred_at: instant
  - severity: info | warning | error | debug
  - outcome: observed | accepted | rejected | running | completed | failed | degraded | skipped
  - correlation: object con correlation_id y request_id opcional
  - subject: object con kind, id y version
  - producer: object con module y port
  - privacy: object con contains_secret=false, contains_transcript=false, classification y redaction_level
  - summary: string compacto
  - links: list opcional
  - payload: object compacto
Invariantes:
  - Core no publica directamente; solo prepara requests para el puerto de salida.
  - Los eventos locales de core se mapean a OrquestaEvent v0 sin transcripts, prompts, completions, secretos, DSN, SQL ni rutas HOME reales.
  - event_type debe mantener prefijo `core.`.
  - El DTO local `OrquestaEventV0` usa el shape del schema global de observability y no expone campos top-level heredados como `correlation_id` o `references`.
  - `dry_run` permite probar adaptadores sin sink real.
Errores:
  - orquesta_event_invalido
  - evento_demasiado_extenso
  - secreto_detectado
  - transcript_no_permitido
  - correlation_id_requerido
  - idempotency_key_requerida
  - source_area_no_soportada
  - event_type_incompatible
  - sink_no_disponible
  - duplicado_idempotente
Pruebas de contrato:
  - Mapper puro `MapEventoDominioCoreAOrquestaEventV0` traduce ProyectoRegistradoEnBorradorV0 a `core.proyecto_registrado_en_borrador.v0` con event_id, severity, outcome, correlation, subject, producer y privacy.
  - El payload se copia para que cambios posteriores al evento local no muten el request publicado.
  - La serializacion del evento no contiene campos prohibidos de payload ni marcadores de secretos/transcripts/prompts/completions/SQL/DSN.
```

```text
Nombre: RuntimeLauncherPortV0
Tipo: puerto_salida
Version: v0
Propietario: orquesta-core
Implementador esperado: orquesta-runtime mediante adaptador futuro de RuntimeLaunchRequest v0
Consumidores: casos de uso de orquesta-core que conviertan FunctionContractV0 activo en lanzamiento runtime
Metodo:
  - LanzarRuntimeV0(context, RuntimeLaunchRequestCoreV0) -> RuntimeLaunchAcceptedCoreV0 | RuntimeLaunchErrorV0
Campos request:
  - schema_version: runtime_launch_request.v0
  - request_id: string
  - correlation_id: string
  - idempotency_key: string
  - function_contract: FunctionContractV0 activo
  - capacity_decision: CapacityDecisionEvidenceV0 con refs opacas
  - binding: RuntimeBindingRefsV0 con refs opacas
  - safety: RuntimeSafetyPolicyV0
Invariantes:
  - Core no arranca procesos, sesiones, CLI, MCP, browser, tmux ni filesystem.
  - Runtime ejecuta una orden; no decide negocio, capacidad, modelo, HOME ni cuotas.
  - `FunctionContractV0` debe estar activo y con write_set cerrado antes de llamar.
  - CapacityDecisionEvidenceV0 es una proyeccion local de `CapacityDecision v0`; no reimplementa capacity.
  - Todas las referencias de binding/capacity son opacas; no viajan secretos, rutas HOME reales ni transcripts.
Errores:
  - runtime_launch_request_invalida
  - function_contract_requerido
  - function_contract_no_activa
  - write_set_requerido
  - write_set_invalido
  - capacity_decision_requerida
  - capacity_decision_no_soportada
  - pool_ref_requerido
  - model_ref_requerido
  - home_ref_requerido
  - credential_ref_requerido
  - referencia_no_opaca
  - secreto_detectado
  - ruta_home_real_detectada
  - mailbox_ref_requerido
  - ack_ref_requerido
  - readiness_ref_requerido
  - idempotency_key_requerida
  - runtime_no_disponible
Pruebas de contrato:
  - Compilacion garantiza que existe puerto local e interfaz implementable sin runtime real.
  - Futuro harness debe validar FunctionContractV0 activo, write_set cerrado y referencias opacas antes de llamar.
```

```text
Nombre: GovernanceCatalogPortV0
Tipo: puerto_salida
Version: v0
Propietario: orquesta-core
Implementador esperado: orquesta-governance mediante adaptador futuro de GovernanceCatalog v0
Consumidores: casos de uso de orquesta-core que necesiten reglas efectivas antes de validar o lanzar microtareas
Metodo:
  - ConsultarGovernanceCatalogV0(context, GovernanceCatalogRequestV0) -> GovernanceCatalogSnapshotV0 | GovernanceCatalogErrorV0
Campos request:
  - schema_version: governance_catalog.v0
  - request_id: string
  - correlation_id: string
  - scope: GovernanceCatalogScopeV0
Campos scope:
  - modulo: string
  - rol: string
  - fase: string
  - tags: list<string>
Campos snapshot:
  - schema_version: governance_catalog.v0
  - correlation_id: string
  - read_at: instant opcional
  - effective: list<GovernanceCatalogEntryV0>
  - proposed_count: integer
  - quarantine_count: integer
Invariantes:
  - Core solo consume `effective` como regla aplicable.
  - `proposed` y `quarantine` son contadores/diagnostico, no reglas efectivas para permisos o runtime.
  - Governance no ejecuta tareas, no concede permisos operativos y no decide runtime.
  - El request normaliza scope y tags; no incluye filtros libres con secretos.
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
Pruebas de contrato:
  - Constructor local compacta scope/tags y fija schema_version v0.
  - Compilacion garantiza que el puerto es implementable sin governance real.
```

## FunctionContract v0

Contrato promovido desde la evidencia forense de DB v1: tabla
`especificaciones_funcion`, reglas de microprogramacion y decision DBV1-000.
La DB v1 no es canon vivo: se reutiliza la forma contractual, no sus ids,
filas, rutas antiguas ni estado operativo.

Estado implementacion 2026-05-04:

- DTO publico local `FunctionContractV0` en `puertos_salida_v0.go` para coordinar `RuntimeLauncherPortV0`.
- No hay validador ejecutable completo, schema ni runtime launcher real en este microcorte.
- La validacion de FunctionContractV0 activo y write_set cerrado queda como siguiente harness de core.

```text
Nombre: FunctionContract
Tipo: dto
Version: v0
Propietario: orquesta-core
Consumidores: orquesta-core, orquesta-governance, orquesta-runtime, orquesta-mcp, orquesta-cli
Objetivo:
  - Definir una unidad pequena de trabajo implementable y verificable.
  - Fijar el borde de escritura permitido antes de ejecutar o revisar una entrega.
  - Separar objetivo, foco tecnico, restricciones, pruebas y formato de entrega.
Archivo/simbolo foco:
  - archivo_objetivo: path relativo al repo o modulo propietario.
  - simbolo_objetivo: funcion, metodo, tipo, caso de uso, documento o contrato principal que concentra el cambio.
Campos:
  - titulo: string
  - objetivo: string
  - archivo_objetivo: string
  - simbolo_objetivo: string
  - descripcion: string
  - write_set: list<string>
  - dependencias_permitidas: list<string>
  - dependencias_prohibidas: list<string>
  - precondiciones: list<string>
  - postcondiciones: list<string>
  - tests_obligatorios: list<string>
  - formato_entrega: enum(patch+evidencia, patch_unificado, ficheros+evidencia)
  - criterio_cierre: list<string>
  - errores_publicos: list<FunctionContractErrorV0>
  - pruebas_contrato: list<string>
  - estado: enum(borrador, activa, reemplazada, archivada)
  - version: integer
  - origen_evidencia: string
Invariantes:
  - titulo, objetivo, archivo_objetivo, simbolo_objetivo, write_set, tests_obligatorios, formato_entrega y criterio_cierre son obligatorios.
  - write_set es la unica lista de paths que puede modificarse para cerrar la funcion.
  - archivo_objetivo debe estar dentro de write_set salvo que el contrato sea documental y lo justifique.
  - Cada path de write_set es relativo, no absoluto, y no contiene secretos, rutas HOME ni ids historicos como canon.
  - dependencias_permitidas y dependencias_prohibidas nombran modulos, paquetes, puertos o capacidades, no proveedores secretos ni internals de otro modulo.
  - dependencias_prohibidas prevalece si una dependencia aparece en ambas listas.
  - precondiciones describen lo que debe existir antes de empezar; no sustituyen la validacion del adaptador.
  - postcondiciones describen estado observable al terminar; deben poder verificarse por prueba, diff o evidencia documental.
  - tests_obligatorios contiene comandos o checks concretos cuando exista runner; si no existe, declara inspeccion verificable y riesgo.
  - formato_entrega por defecto es patch+evidencia.
  - estado activa es ejecutable; borrador no debe lanzarse a runtime sin decision explicita.
Dependencias permitidas:
  - Tipos publicos y contratos versionados del modulo propietario.
  - Puertos publicos declarados en contratos locales o globales.
  - Librerias estandar o dependencias ya aprobadas por el modulo.
  - Fixtures y documentos dentro del write_set cuando la tarea sea documental.
Dependencias prohibidas:
  - DB directa, HTTP directo, CLI directo, MCP directo, tmux, Docker, filesystem o runtime desde dominio core.
  - Importar `internal/`, structs privados, tablas, filas o detalles de proveedor de otro modulo.
  - Copiar secretos, tokens, rutas HOME, transcripts completos o ids historicos de DB v1 como canon.
  - Ampliar write_set durante la ejecucion sin nueva decision o consulta.
Precondiciones:
  - Existe un backlog o tarea que referencia el FunctionContract.
  - El modulo propietario y el write_set estan identificados antes de editar.
  - Las dependencias prohibidas no son necesarias para cumplir el objetivo.
  - Las pruebas obligatorias son ejecutables o tienen sustituto documental declarado.
Postcondiciones:
  - Solo se modifican paths incluidos en write_set.
  - El simbolo foco o documento foco refleja el objetivo.
  - Las pruebas obligatorias se ejecutan o se documenta por que no pudieron ejecutarse.
  - La entrega incluye evidencia: comandos, resultado, diff revisado o inspeccion documental.
Tests obligatorios:
  - Validar que todo fichero cambiado pertenece a write_set.
  - Validar que archivo_objetivo no queda fuera de write_set.
  - Validar que cada dependencia usada no pertenece a dependencias_prohibidas.
  - Ejecutar cada comando declarado en tests_obligatorios cuando aplique.
  - Ejecutar `git diff --check` limitado al write_set antes de cerrar.
Formato de entrega:
  - patch+evidencia: diff aplicado mas resumen de validaciones.
  - patch_unificado: diff unificado cuando el canal de ejecucion no puede aplicar cambios.
  - ficheros+evidencia: lista exacta de ficheros entregados mas checks verificables.
Criterio de cierre:
  - Contrato activo, completo y pequeno.
  - Write-set respetado.
  - Dependencias permitidas/prohibidas verificadas.
  - Tests obligatorios ejecutados o bloqueo documentado.
  - Errores publicos definidos cuando pueda fallar la validacion.
  - Evidencia incluida sin secretos ni ids historicos canonicos.
Errores:
  - function_contract_incompleto
  - archivo_objetivo_fuera_de_write_set
  - write_set_vacio
  - cambio_fuera_de_write_set
  - dependencia_prohibida
  - tests_obligatorios_ausentes
  - formato_entrega_no_soportado
  - evidencia_insuficiente
  - estado_no_ejecutable
Pruebas de contrato:
  - Fixture minimo activo con objetivo, archivo/simbolo foco, write_set, tests y criterio_cierre.
  - Fixture invalido sin write_set.
  - Fixture invalido con archivo_objetivo fuera de write_set.
  - Fixture invalido por dependencia prohibida.
  - Fixture invalido por formato_entrega desconocido.
  - Validacion de entrega rechaza ficheros cambiados fuera de write_set.
```

```text
Nombre: FunctionContractErrorV0
Tipo: error
Version: v0
Propietario: orquesta-core
Consumidores: orquesta-core, orquesta-governance, orquesta-runtime, orquesta-mcp, orquesta-cli
Campos:
  - code: string
  - message: string
  - field: string
  - retryable: boolean
  - evidence: list<string>
Invariantes:
  - code debe pertenecer a la lista de errores publicos de FunctionContract v0.
  - message es diagnostico corto, no texto final de UI.
  - field apunta al campo contractual cuando aplique.
  - evidence contiene referencias compactas, nunca secretos, transcripts completos ni IDs historicos canonicos.
Errores:
  - error_code_invalido
Pruebas de contrato:
  - Cada error publico de FunctionContract v0 tiene fixture minimo.
```

## Consulta CLI candidata sobre FunctionContract v0

Respuesta local 2026-05-04 a la consulta CLI sobre `FunctionContractV0`.
Este corte solo documenta operaciones candidatas de lectura. No implementa
scheduler, workflow, `OrchestrationRun`, `CommandHandler`, `Outbox`, runtime ni
adaptadores CLI/MCP/HTTP. Para que CLI consuma estas operaciones como contrato
inter-modulo operativo, debe promocionarse el resumen global mediante decision
del director; este documento no modifica `../../CONTRATOS.md`.

```text
Nombre: ListarFunctionContractsV0
Tipo: puerto_entrada
Version: v0
Estado: candidato_local_no_promovido
Propietario: orquesta-core
Consumidores candidatos: orquesta-cli, orquesta-mcp, orquesta-core
Operacion publica candidata: listar FunctionContractV0
Campos entrada:
  - request_id: string
  - correlation_id: string
  - filtros.modulo: string opcional
  - filtros.estado: enum(borrador, activa, reemplazada, archivada) opcional
  - filtros.archivo_objetivo: string opcional
  - filtros.simbolo_objetivo: string opcional
  - page.limit: integer opcional
  - page.cursor: string opaco opcional
Campos salida:
  - items: list<FunctionContractResumenV0>
  - next_cursor: string opaco opcional
  - warnings: list<string>
Invariantes:
  - Operacion de solo lectura; no crea, actualiza, archiva ni reemplaza FunctionContractV0.
  - No agenda trabajo, no inicia workflow, no crea OrchestrationRun, no invoca CommandHandler y no escribe Outbox.
  - No persiste ni lee DB directa desde core; la fuente futura debe entrar por puerto/adaptador definido.
  - Los filtros son contractuales y acotados; no aceptan SQL, queries libres, paths absolutos ni IDs historicos de DB v1 como canon.
  - La respuesta no contiene secretos, rutas HOME, transcripts, prompts, completions ni detalles de proveedor.
  - Si no existe catalogo/lector formal, la operacion debe responder no disponible en vez de inferir desde filesystem.
Errores:
  - consulta_function_contract_no_disponible
  - filtro_no_soportado
  - cursor_invalido
  - operacion_no_promovida
Pruebas de contrato:
  - Listar devuelve solo resumenes serializables y no muta estado.
  - Listar rechaza filtros libres o paths absolutos.
  - Listar no expone campos prohibidos ni detalles de adaptador.
```

```text
Nombre: VerFunctionContractV0
Tipo: puerto_entrada
Version: v0
Estado: candidato_local_no_promovido
Propietario: orquesta-core
Consumidores candidatos: orquesta-cli, orquesta-mcp, orquesta-core
Operacion publica candidata: ver FunctionContractV0
Campos entrada:
  - request_id: string
  - correlation_id: string
  - function_contract_ref: string opaco
  - version: integer opcional
Campos salida:
  - function_contract: FunctionContractV0
  - warnings: list<string>
Invariantes:
  - Operacion de solo lectura; no cambia estado ni materializa borradores.
  - Solo devuelve contratos existentes en una fuente contractual aprobada.
  - `function_contract_ref` es opaco para el adaptador; no es ruta absoluta, query SQL ni ID historico de DB v1 como canon.
  - No agenda trabajo, no inicia workflow, no crea OrchestrationRun, no invoca CommandHandler y no escribe Outbox.
  - No arranca runtime ni prepara `RuntimeLaunchRequestV0`.
  - La respuesta conserva las invariantes de FunctionContractV0 y no anyade datos privados de adaptador.
Errores:
  - function_contract_no_encontrado
  - consulta_function_contract_no_disponible
  - operacion_no_promovida
Pruebas de contrato:
  - Ver devuelve un FunctionContractV0 serializable sin secretos ni detalles de adaptador.
  - Ver no permite usar rutas absolutas, queries ni IDs historicos como referencia canonica.
  - Ver no muta estado ni produce eventos de workflow.
```

```text
Nombre: RegistrarFunctionContractV0
Tipo: puerto_entrada
Version: v0
Estado: bloqueada
Propietario: orquesta-core
Consumidores candidatos: ninguno hasta nueva decision
Operacion solicitada: registrar FunctionContractV0
Bloqueos:
  - OrchestrationRun no esta cerrado como contrato.
  - CommandHandler no esta cerrado como contrato.
  - Outbox no esta cerrado como contrato.
Invariantes:
  - No debe exponerse como comando CLI operativo en este corte.
  - Registrar implica mutacion, idempotencia, auditoria y eventos; no puede modelarse como lectura.
  - No crea contratos desde CLI, no persiste, no publica eventos, no lanza runtime y no agenda workflow.
  - Cualquier desbloqueo requiere contrato pequeno, pruebas y decision de promocion si afecta a CLI/MCP.
Errores:
  - registrar_function_contract_bloqueado
  - operacion_no_promovida
Pruebas de contrato:
  - Cualquier intento de registrar desde el borde CLI debe responder bloqueo publico hasta cerrar OrchestrationRun/CommandHandler/Outbox.
```

## CONSULTA AL DIRECTOR

```text
Modulo origen: orquesta-core
Modulo afectado: contratos globales
Bloqueo: La tarea CORE-005 pidio confirmar la ubicacion del registro compartido antes de promocionar el resumen global de FunctionContract v0.
Pregunta concreta: Confirmar si el resumen global de `FunctionContract v0` debe vivir en `CONTRATOS.md` raiz o en `modulos/CONTRATOS.md`.
Opcion recomendada: Usar `modulos/CONTRATOS.md` como registro global entre mini-proyectos. Desde archivos en `modulos/orquesta-core/docs/`, la ruta relativa `../../CONTRATOS.md` es correcta; desde el directorio del modulo es `../CONTRATOS.md`.
Impacto: Evita crear dos fuentes globales de contratos y permite promocionar solo el resumen minimo manteniendo el detalle canonico en `orquesta-core/docs/contratos.md`.
Decision del director: Usar `modulos/CONTRATOS.md`. FunctionContract v0 queda promovido como contrato global minimo; esta consulta queda cerrada.
```
