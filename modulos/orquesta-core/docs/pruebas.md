# Pruebas locales: orquesta-core

Registra pruebas obligatorias del modulo.

```text
Caso: CORE-CT-012 descriptor_source MCP sobre contratos core
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp ./modulos/orquesta-observability ./modulos/orquesta-governance ./modulos/orquesta-core ./cmd/orquesta-server
Evidencia esperada: los descriptors MCP que apuntan a core declaran owner,
fuente canonica, DTO/validador y errores publicos sin que core importe MCP,
HTTP, CLI, DB, runtime ni proveedor.
Ultima ejecucion: 2026-05-27, ok en paquete
`agent-ref-task-autoprogramming-c3678e9bc306-g01`.
Riesgos: Esta prueba valida el contrato cruzado existente; no crea adaptadores nuevos.
Revalidacion: `agent-ref-task-autoprogramming-c3678e9bc306-g01` usa el comando
obligatorio ampliado para conservar el cierre T198.
```

## Plantilla

```text
Caso:
Tipo: unit | contract | integration | smoke
Comando:
Evidencia esperada:
Ultima ejecucion:
Riesgos:
```

## Pruebas ejecutables para RegistrarProyectoDesdeAppSpec v0

Implementacion ejecutable 2026-05-04:

- Fichero: `registrar_proyecto_appspec_v0_test.go`.
- Comando: `go test -count=1 ./modulos/orquesta-core ./modulos/orquesta-factory`.
- Cubre caso OK, `app_spec_no_validada`, `backlog_vacio`, `backlog_incompatible_con_app_spec`, `idempotency_key_requerida`, idempotencia determinista y `contrato_factory_no_soportado`.
- Fixtures compactos de command: `docs/fixtures/registrar_proyecto_appspec_v0/command_ok_minimo.json`, `error_app_spec_no_validada.json`, `error_backlog_vacio.json`, `error_version_incompatible.json` y `error_idempotency_key_vacia.json`.
- Los fixtures no declaran DB, runtime, proveedor tecnico ni rutas HOME.

```text
Caso: registrar_app_spec_validada_con_backlog_minimo
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core ./modulos/orquesta-factory
Evidencia esperada: Dado AppSpecV0 validada y BacklogInicialPropuestoV0 compatible, devuelve RegistroProyectoAceptadoV0 con ProyectoPlanBorradorV0, ids estables y evento ProyectoRegistradoEnBorradorV0.
Ultima ejecucion: 2026-05-04, OK con `go test -count=1 ./modulos/orquesta-core`.
Riesgos: Usa constructores publicos de factory como fixtures vivos.
```

```text
Caso: rechazar_app_spec_sin_validacion
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core ./modulos/orquesta-factory
Evidencia esperada: Dado app_spec_validada=false o evidencia ausente, devuelve error publico app_spec_no_validada y no devuelve ProyectoPlanBorradorV0.
Ultima ejecucion: 2026-05-04, OK con `go test -count=1 ./modulos/orquesta-core`; fixture documental `error_app_spec_no_validada.json`.
Riesgos: Core solo comprueba `validation.estado == valida`; no revalida reglas internas de factory.
```

```text
Caso: rechazar_backlog_vacio
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core ./modulos/orquesta-factory
Evidencia esperada: Dado BacklogInicialPropuestoV0 sin items, devuelve backlog_vacio y no genera eventos de aceptacion.
Ultima ejecucion: 2026-05-04, OK con `go test -count=1 ./modulos/orquesta-core`; fixture documental `error_backlog_vacio.json`.
Riesgos: Puede requerir distinguir backlog vacio de backlog con preguntas abiertas si factory lo permite.
```

```text
Caso: rechazar_spec_id_incompatible
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core ./modulos/orquesta-factory
Evidencia esperada: Dado backlog.spec_id distinto de app_spec.spec_id, devuelve backlog_incompatible_con_app_spec.
Ultima ejecucion: 2026-05-04, OK con `go test -count=1 ./modulos/orquesta-core ./modulos/orquesta-factory`.
Riesgos: La compatibilidad v0 se limita a spec_id.
```

```text
Caso: rechazar_version_factory_no_soportada
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core ./modulos/orquesta-factory
Evidencia esperada: Dado app_spec_version o backlog_version diferente de v0, devuelve contrato_factory_no_soportado.
Ultima ejecucion: 2026-05-04, OK con `go test -count=1 ./modulos/orquesta-core`; fixture documental `error_version_incompatible.json`.
Riesgos: La regla de evolucion v1 debe pasar por contrato global si afecta a factory.
```

```text
Caso: idempotencia_por_idempotency_key
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core ./modulos/orquesta-factory
Evidencia esperada: Dos comandos equivalentes con la misma idempotency_key producen el mismo registro_id y el mismo proyecto_id_propuesto.
Ultima ejecucion: 2026-05-04, OK con caso duplicado explicito en `registrar_proyecto_appspec_v0_test.go`.
Riesgos: Sin persistencia real, la idempotencia v0 solo puede comprobar determinismo logico del caso de uso puro.
```

```text
Caso: rechazar_idempotency_key_vacia
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core
Evidencia esperada: Dado idempotency_key vacia o con solo espacios, devuelve idempotency_key_requerida antes de construir ProyectoPlanBorradorV0.
Ultima ejecucion: 2026-05-04, OK con `go test -count=1 ./modulos/orquesta-core`; fixture documental `error_idempotency_key_vacia.json`.
Riesgos: Sin persistencia real, este corte solo valida presencia de clave opaca; no comprueba unicidad historica.
```

```text
Caso: no_filtrar_detalles_de_adaptadores
Tipo: contract
Comando: pendiente; inspeccion de salida serializada
Evidencia esperada: RegistroProyectoAceptadoV0 y ProyectoPlanBorradorV0 no contienen DSN, tablas, rutas locales, comandos runtime, proveedor LLM ni detalles HTTP/CLI/MCP.
Ultima ejecucion: no ejecutada; documentacion inicial 2026-05-04
Riesgos: Puede requerir lista negativa mantenida cuando aparezcan nuevos adaptadores.
```

## Pruebas ejecutables para puertos de salida v0

```text
Caso: persistence_request_desde_proyecto_borrador
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core
Evidencia esperada: NewGuardarProyectoBorradorPersistenceRequestV0 fija schema_version/contrato PersistenceRepositoryV0, operacion guardar_proyecto_borrador, payload_version ProyectoPlanBorradorV0 y conserva request_id, correlation_id, idempotency_key y payload borrador sin confirmar DB real.
Ultima ejecucion: 2026-05-04, OK con `go test -count=1 ./modulos/orquesta-core`.
Riesgos: El adaptador persistence futuro debe construir el envelope global con adapter_metadata/unidad_trabajo y validar transaccion/idempotencia contra su contrato propio.
```

```text
Caso: observability_request_desde_eventos_dominio_core
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core
Evidencia esperada: NewPublishOrquestaEventRequestsV0 mapea ProyectoRegistradoEnBorradorV0 a OrquestaEvent v0 con event_id, source_area core, event_type core.*, severity, outcome, correlation object, subject, producer, privacy, idempotency_key derivada y payload copiado.
Ultima ejecucion: 2026-05-04, OK con `go test -count=1 ./modulos/orquesta-core`.
Riesgos: El sink real sigue fuera de alcance; observability debe aplicar su schema y controles de secretos.
```

```text
Caso: orquesta_event_shape_global_sin_campos_prohibidos
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core
Evidencia esperada: OrquestaEventV0 serializado contiene los campos requeridos por el schema global, no expone top-level correlation_id/references y su payload no contiene claves prohibidas ni marcadores sensibles.
Ultima ejecucion: 2026-05-04, OK con `go test -count=1 ./modulos/orquesta-core`.
Riesgos: No ejecuta AJV; observability mantiene la validacion JSON Schema completa.
```

```text
Caso: puertos_salida_sin_detalles_adaptador
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core
Evidencia esperada: Los DTOs serializados de persistence/observability no contienen DSN, SQL, tmux, Docker ni tokens.
Ultima ejecucion: 2026-05-04, OK con `go test -count=1 ./modulos/orquesta-core`.
Riesgos: La lista negativa es minima y debe ampliarse cuando aparezcan adaptadores reales.
```

```text
Caso: governance_request_compacta_scope
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-core
Evidencia esperada: NewGovernanceCatalogRequestV0 fija schema_version v0, recorta request/correlation/scope y compacta tags sin governance real.
Ultima ejecucion: 2026-05-04, OK con `go test -count=1 ./modulos/orquesta-core`.
Riesgos: El catalogo efectivo real debe validarse en orquesta-governance.
```

```text
Caso: interfaces_puertos_salida_compilan_sin_adaptadores
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core
Evidencia esperada: Fakes de test implementan PersistenceRepositoryPortV0, RuntimeLauncherPortV0, OrquestaEventPublisherPortV0 y GovernanceCatalogPortV0 sin imports de otros modulos.
Ultima ejecucion: 2026-05-04, OK con `go test -count=1 ./modulos/orquesta-core`.
Riesgos: No sustituye pruebas de contrato cruzadas cuando existan adaptadores reales.
```

```text
Caso: core_009_split_mecanico_puertos_salida
Tipo: contract
Comando: gofmt puertos_salida_v0.go puertos_salida_*_v0.go; go test -count=1 ./modulos/orquesta-core; git diff --check -- modulos/orquesta-core; wc -l puertos_salida_v0.go puertos_salida_*_v0.go
Evidencia esperada: Los simbolos publicos de puertos de salida siguen compilando, `puertos_salida_v0.go` deja de concentrar la zona roja y los nuevos ficheros quedan separados por familia sin adaptadores reales.
Ultima ejecucion: 2026-05-04, OK con `gofmt`, `go test -count=1 ./modulos/orquesta-core`, `git diff --check -- modulos/orquesta-core` y `wc -l puertos_salida_v0.go puertos_salida_*_v0.go`.
Riesgos: Split mecanico; cualquier cambio semantico posterior debe tener tarea propia.
```

```text
Caso: core_010_split_mecanico_registrar_proyecto_appspec
Tipo: contract
Comando: gofmt registrar_proyecto_appspec_v0.go registrar_proyecto_appspec_*_v0.go; go test -count=1 ./modulos/orquesta-core; git diff --check -- modulos/orquesta-core; wc -l registrar_proyecto_appspec_v0.go registrar_proyecto_appspec_*_v0.go
Evidencia esperada: `RegistrarProyectoDesdeAppSpecV0` conserva contrato publico, errores, invariantes y comportamiento; el archivo principal queda limitado a constantes, DTOs, error y funcion publica, y validacion/ensamblado/helpers quedan en ficheros separados por responsabilidad.
Ultima ejecucion: 2026-05-04, OK con `gofmt`, `go test -count=1 ./modulos/orquesta-core`, `git diff --check -- modulos/orquesta-core` y `wc -l registrar_proyecto_appspec_v0.go registrar_proyecto_appspec_*_v0.go`.
Riesgos: Split mecanico; cualquier cambio semantico posterior debe tener tarea propia.
```

## Pruebas previstas para FunctionContract v0

```text
Caso: function_contract_minimo_activo
Tipo: contract
Comando: pendiente; futuro validador de contratos de orquesta-core
Evidencia esperada: Dado un FunctionContractV0 con titulo, objetivo, archivo_objetivo, simbolo_objetivo, write_set, tests_obligatorios, formato_entrega y criterio_cierre, la validacion lo acepta como estado activa.
Ultima ejecucion: no ejecutada; documentacion CORE-005 2026-05-04
Riesgos: Existe DTO Go publico, pero no hay aun runner ni schema ejecutable para validar invariantes completos de FunctionContractV0.
```

```text
Caso: function_contract_rechaza_write_set_vacio
Tipo: contract
Comando: pendiente; futuro validador de contratos de orquesta-core
Evidencia esperada: Dado write_set vacio, devuelve error publico write_set_vacio o function_contract_incompleto y no habilita ejecucion runtime.
Ultima ejecucion: no ejecutada; documentacion CORE-005 2026-05-04
Riesgos: Debe coordinarse con governance si el catalogo permite borradores incompletos.
```

```text
Caso: function_contract_rechaza_archivo_fuera_de_write_set
Tipo: contract
Comando: pendiente; futuro validador de contratos de orquesta-core
Evidencia esperada: Dado archivo_objetivo no incluido en write_set, devuelve archivo_objetivo_fuera_de_write_set salvo excepcion documental explicita.
Ultima ejecucion: no ejecutada; documentacion CORE-005 2026-05-04
Riesgos: Las excepciones documentales deben quedar codificadas para no abrir permisos implicitos.
```

```text
Caso: function_contract_rechaza_dependencia_prohibida
Tipo: contract
Comando: pendiente; futuro validador de contratos de orquesta-core
Evidencia esperada: Dado uso o declaracion de una dependencia prohibida, devuelve dependencia_prohibida; si aparece tambien como permitida, prevalece la prohibicion.
Ultima ejecucion: no ejecutada; documentacion CORE-005 2026-05-04
Riesgos: Requiere definir como se extraen dependencias reales desde codigo o entrega documental.
```

```text
Caso: function_contract_rechaza_formato_entrega_desconocido
Tipo: contract
Comando: pendiente; futuro validador de contratos de orquesta-core
Evidencia esperada: Dado formato_entrega distinto de patch+evidencia, patch_unificado o ficheros+evidencia, devuelve formato_entrega_no_soportado.
Ultima ejecucion: no ejecutada; documentacion CORE-005 2026-05-04
Riesgos: Si runtime necesita otro formato, debe versionarse o elevar consulta.
```

```text
Caso: validar_entrega_respeta_write_set
Tipo: contract
Comando: pendiente; futuro validador de entrega de orquesta-core
Evidencia esperada: Dado FunctionContractV0 y lista de ficheros cambiados, acepta solo si todos los paths estan dentro de write_set; en caso contrario devuelve cambio_fuera_de_write_set con evidencia compacta.
Ultima ejecucion: no ejecutada; documentacion CORE-005 2026-05-04
Riesgos: Debe ignorar cambios generados fuera del control de la entrega solo si estan formalmente excluidos por el entorno.
```

## Pruebas previstas para consulta CLI FunctionContract v0

```text
Caso: listar_function_contracts_solo_lectura
Tipo: contract
Comando: pendiente; futuro harness de consulta FunctionContractV0. Validacion documental de este corte: git diff --check -- .
Evidencia esperada: Dado un catalogo aprobado de FunctionContractV0, listar devuelve resumenes serializables, filtros acotados y cursor opaco sin mutar estado, sin crear OrchestrationRun, sin invocar CommandHandler y sin escribir Outbox.
Ultima ejecucion: no ejecutada; documentacion CORE-008 2026-05-04
Riesgos: No existe aun fuente contractual ni adaptador CLI; la operacion sigue candidata local no promovida.
```

```text
Caso: ver_function_contract_solo_lectura
Tipo: contract
Comando: pendiente; futuro harness de consulta FunctionContractV0. Validacion documental de este corte: git diff --check -- .
Evidencia esperada: Dado un function_contract_ref opaco existente, ver devuelve FunctionContractV0 serializable sin secretos, rutas HOME, transcripts, queries, ids historicos canonicos ni detalles de adaptador; no muta estado ni produce eventos de workflow.
Ultima ejecucion: no ejecutada; documentacion CORE-008 2026-05-04
Riesgos: Requiere definir fuente aprobada antes de conectar CLI/MCP.
```

```text
Caso: registrar_function_contract_bloqueado
Tipo: contract
Comando: pendiente; futuro harness de borde CLI. Validacion documental de este corte: git diff --check -- .
Evidencia esperada: Cualquier intento de registrar FunctionContractV0 desde CLI responde registrar_function_contract_bloqueado u operacion_no_promovida hasta cerrar OrchestrationRun, CommandHandler y Outbox.
Ultima ejecucion: no ejecutada; documentacion CORE-008 2026-05-04
Riesgos: Registrar no puede desbloquearse como atajo documental porque implica mutacion, idempotencia, auditoria y eventos.
```
