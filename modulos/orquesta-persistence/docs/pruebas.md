# Pruebas locales: orquesta-persistence

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
Caso: Contrato documental de repositorio de borradores
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-persistence -run 'TestDecodeGuardarProyectoBorradorMaterialV0FixtureValido|TestInMemoryPersistenceRepositoryContractAdapterV0GuardaBorrador'
Evidencia esperada: Fixture de `ProyectoPlanBorradorV0` minimo decodifica sin issues y el adaptador en memoria devuelve `proyecto_id`, `idempotency_key` y timestamps tecnicos sin perdida de payload versionado.
Ultima ejecucion: 2026-06-29; pasa con DTOs Go y adaptador en memoria de contrato.
Riesgos: El adaptador en memoria no es productivo ni modela transacciones reales.
```

```text
Caso: Idempotencia de guardado de `ProyectoPlanBorradorV0`
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-persistence -run 'TestInMemoryPersistenceRepositoryContractAdapterV0EsIdempotenteConPayloadEquivalente|TestInMemoryPersistenceRepositoryContractAdapterV0DetectaConflictoIdempotencia'
Evidencia esperada: Misma `idempotency_key` con payload equivalente devuelve el mismo `proyecto_id`; payload incompatible falla con `duplicado_idempotente_incompatible`.
Ultima ejecucion: 2026-06-29; pasa en adaptador en memoria puro.
Riesgos: El alcance global de la idempotencia en un conector productivo sigue dependiendo de la composicion aprobada.
```

```text
Caso: Transaccion commit/rollback
Tipo: integration
Comando: bloqueo externo de producto/conector; no hay conector de persistencia productivo aprobado.
Evidencia esperada: Commit confirma escritura; rollback no deja escritura visible; doble rollback permite limpieza.
Ultima ejecucion: no ejecutada por ausencia de conector aprobado; el adaptador en memoria de contrato declara explicitamente que no modela transacciones reales.
Riesgos: El mapeo de aislamiento y errores depende del conector elegido. Accion operador: aprobar conector/version y perfil opaco para implementar harness de commit/rollback sin exponer motor al contrato publico.
```

```text
Caso: Rechazo de escritura sin transaccion activa
Tipo: contract
Comando: bloqueo de contrato; `InMemoryPersistenceRepositoryContractAdapterV0` no modela unidad de trabajo y `PersistenceUnitOfWorkV0` sigue sin implementacion aprobada.
Evidencia esperada: `guardar_borrador` falla con `transaccion_requerida` o `transaccion_no_activa`.
Ultima ejecucion: no ejecutada; queda acotado a la futura implementacion de `PersistenceUnitOfWorkV0`.
Riesgos: Si el puerto global permite auto-commit, esta invariante debera revisarse con el director antes de programarla.
```

```text
Caso: Conector de persistencia externo aprobado
Tipo: integration
Comando: bloqueo externo de producto/conector; futuro test de contrato para cualquier conector externo aprobado por el director.
Evidencia esperada: Preparacion tecnica, idempotencia y transacciones pasan sin exponer motor, adaptador de proveedor, credenciales ni tablas al contrato publico.
Ultima ejecucion: no ejecutada; no hay conector externo aprobado.
Riesgos: Riesgo de tratar un motor concreto como universal; debe bloquearse cualquier dependencia operativa no aprobada. Accion operador: aprobar conector externo y alcance de persistencia productiva.
```

```text
Caso: Sin reglas de negocio en queries
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-persistence -run 'TestDecodeGuardarProyectoBorradorMaterialV0RechazaQueryEnContratoMaterial|TestDecodeGuardarProyectoBorradorRequestV0RechazaConectorYQuery'
Evidencia esperada: Queries filtran por campos tecnicos documentados y no por fases, autonomia, modelos, capacidad, runtime ni handoff.
Ultima ejecucion: 2026-06-29; las requests/materiales con `query` o conector directo fallan por contrato.
Riesgos: Las optimizaciones futuras pueden introducir filtros semanticos que pertenecen a core; todo conector nuevo debe mantener esta prueba.
```

```text
Caso: Schema material `PersistenceRepositoryV0`
Tipo: contract
Comando: `jq empty docs/schemas/*.schema.json docs/fixtures/persistence_repository_v0/*.json`
Evidencia esperada: Schema y fixtures JSON parsean correctamente.
Ultima ejecucion: 2026-05-05; ejecutada correctamente.
Riesgos: `jq` solo valida sintaxis JSON, no cumplimiento de schema.
```

```text
Caso: Fixture valido de guardado de borrador
Tipo: contract
Comando: `npx ajv-cli validate -s docs/schemas/persistence_repository_v0.schema.json -d docs/fixtures/persistence_repository_v0/guardar_proyecto_borrador_valido.json`
Evidencia esperada: El fixture `guardar_proyecto_borrador_valido.json` cumple el schema.
Ultima ejecucion: 2026-05-04; ejecutada correctamente con `npx --yes ajv-cli validate --spec=draft7`.
Riesgos: El schema valida forma contractual; no prueba persistencia real.
```

```text
Caso: Fixtures invalidos de conector directo y regla en query
Tipo: contract
Comando: `npx ajv-cli validate -s docs/schemas/persistence_repository_v0.schema.json -d docs/fixtures/persistence_repository_v0/conector_directo_invalido.json` y `npx ajv-cli validate -s docs/schemas/persistence_repository_v0.schema.json -d docs/fixtures/persistence_repository_v0/regla_negocio_en_query_invalido.json`
Evidencia esperada: Ambos fixtures fallan: el primero por conector de persistencia en `request`; el segundo por propiedad `query` con filtros de negocio.
Ultima ejecucion: 2026-05-04; ejecutada correctamente con `npx --yes ajv-cli validate --spec=draft7`, confirmando fallo esperado de ambos fixtures.
Riesgos: Si se amplia el schema para permitir extensiones, debe mantenerse la prohibicion explicita de SQL/query/reglas de negocio.
```

```text
Caso: DTOs Go `PersistenceRepositoryV0.guardar_proyecto_borrador`
Tipo: unit
Comando: `go test -count=1 ./modulos/orquesta-persistence`
Evidencia esperada: El fixture valido decodifica sin issues; los fixtures con conector directo y `query` fallan; `idempotency_key` vacia y `payload_version` distinta de `ProyectoPlanBorradorV0` fallan.
Ultima ejecucion: 2026-05-05; ejecutada correctamente.
Riesgos: La validacion es de forma contractual minima; no prueba almacenamiento operativo, transacciones reales, migraciones ni conectores externos.
```

```text
Caso: Rechazo de detalles concretos en request y metadata
Tipo: unit
Comando: `go test -count=1 ./modulos/orquesta-persistence`
Evidencia esperada: `database`, `db`, `driver`, `connector_ref` y nombres de motor fallan si aparecen en el envelope tecnico del request o en `adapter_metadata`.
Ultima ejecucion: 2026-05-05; ejecutada correctamente.
Riesgos: La lista de terminos prohibidos es defensiva; cualquier conector real aprobado debera seguir entrando por refs opacas, no por nombres de motor en el contrato.
```

```text
Caso: Adaptador en memoria puro `PersistenceRepositoryV0`
Tipo: contract
Comando: `go test -count=1 ./modulos/orquesta-persistence`
Evidencia esperada: El adaptador en memoria guarda un borrador valido, devuelve la misma respuesta para replay idempotente, rechaza payload distinto con la misma clave y propaga issues de request invalido.
Ultima ejecucion: 2026-05-05; ejecutada correctamente.
Riesgos: Solo valida contrato en RAM; no prueba almacenamiento operativo, migraciones, consultas ejecutables, motores reales, adaptadores de proveedor, unidad de trabajo ni transacciones reales.
```

```text
Caso: Saneamiento de tamano `PersistenceRepositoryV0`
Tipo: unit
Comando: `gofmt`; `go test -count=1 ./modulos/orquesta-persistence`; `git diff --check -- modulos/orquesta-persistence`; `wc -l persistence_repository_v0.go persistence_repository_types_v0.go persistence_repository_decode_v0.go persistence_repository_validation_v0.go persistence_repository_helpers_v0.go memory_contract_adapter_v0.go memory_contract_adapter_v0_test.go persistence_repository_v0_test.go`
Evidencia esperada: Los tests existentes siguen pasando y los archivos Go tocados quedan por debajo de 300-350 lineas.
Ultima ejecucion: 2026-05-05; ejecutada correctamente.
Riesgos: Es una separacion mecanica; no cubre almacenamiento operativo, migraciones, consultas ejecutables, motores reales ni filesystem productivo.
```

```text
Caso: Ledger outbox ACK en memoria `InMemoryOutboxLedgerV0`
Tipo: contract
Comando: `go test -count=1 ./modulos/orquesta-persistence`
Evidencia esperada: Guarda mensajes `LaunchRuntimeAgent`, `StopRuntimeAgent` y `SendDirectorQuestion` validos con `ValidateOutboxMessageV0`; lista pendientes por `run_id` y `target_port`; un ACK `dispatched` retira solo el mensaje confirmado.
Ultima ejecucion: 2026-05-05; ejecutada correctamente.
Riesgos: Solo valida comportamiento de contrato en RAM; no prueba almacenamiento operativo, migraciones, queries, colas, dispatch real, sleeps, procesos ni runtime real.
```

```text
Caso: Idempotencia y conflictos del ledger outbox ACK
Tipo: contract
Comando: `go test -count=1 ./modulos/orquesta-persistence`
Evidencia esperada: Reintentar el mismo mensaje o ACK devuelve snapshot estable; reintentar con payload o ACK incompatible falla con `conflicto_idempotencia`; outbox invalido se rechaza con error publico del contrato de workflow.
Ultima ejecucion: 2026-05-05; ejecutada correctamente.
Riesgos: El alcance durable de idempotencia de un dispatcher productivo futuro debera definirse en su conector versionado.
```

```text
Caso: Snapshots del ledger sin detalles prohibidos
Tipo: contract
Comando: `go test -count=1 ./modulos/orquesta-persistence`
Evidencia esperada: JSON de pendientes y ACK no contiene motor concreto, DSN, SQL, provider, HOME, OAuth, transcript ni prompt.
Ultima ejecucion: 2026-05-05; ejecutada correctamente.
Riesgos: La denylist de pruebas es defensiva; cualquier conector real debe mantener refs opacas y contrato propio.
```

```text
Caso: ACK `failed` del ledger outbox
Tipo: contract
Comando: `go test -count=1 ./modulos/orquesta-persistence`
Evidencia esperada: Un mensaje valido guardado en `InMemoryOutboxLedgerV0` acepta ACK `failed`, normaliza `error_code` compacto y `evidence_refs` opacas, y deja de aparecer en `ListPending`.
Ultima ejecucion: 2026-05-05; ejecutada correctamente.
Riesgos: Solo valida contrato en RAM; no prueba almacenamiento operativo, queries, colas, goroutines de dispatch, sleeps, procesos ni runtime real.
```

```text
Caso: Ledger outbox file-based durable con claims y ACK terminales
Tipo: contract
Comando: `go test -count=1 ./modulos/orquesta-persistence`
Evidencia esperada: Tras recrear instancia, un claim sin ACK se puede reclamar de nuevo, un ACK `failed` queda visible en snapshots y un ACK `success` no vuelve a pendientes.
Ultima ejecucion: 2026-05-25; ejecutada correctamente.
Riesgos: Es adaptador JSON local del servidor residente; no sustituye un backend transaccional externo ni broker productivo.
```

```text
Caso: Idempotencia y conflictos de ACK `failed`
Tipo: contract
Comando: `go test -count=1 ./modulos/orquesta-persistence`
Evidencia esperada: Reintentar el mismo ACK `failed` devuelve el mismo snapshot; reintentar con `error_code` o `dispatch_ref` distinto falla con `conflicto_idempotencia`.
Ultima ejecucion: 2026-05-05; ejecutada correctamente.
Riesgos: El alcance durable de idempotencia de un dispatcher productivo futuro debera definirse en su conector versionado.
```

```text
Caso: ACK `failed` sin `error_code` segun contrato actual
Tipo: contract
Comando: `go test -count=1 ./modulos/orquesta-persistence`
Evidencia esperada: El ACK `failed` sin `error_code` es aceptado porque `OutboxDispatchAckV0.error_code` esta documentado como opcional; el ledger no define taxonomia de fallos ni cambia el contrato por implementacion.
Ultima ejecucion: 2026-05-05; ejecutada correctamente.
Riesgos: Si un conector futuro necesita `error_code` obligatorio para `failed`, debe formalizarlo como nueva decision contractual antes de endurecer la validacion.
```

```text
Caso: Snapshots del ledger para ACK `failed` sin detalles prohibidos
Tipo: contract
Comando: `go test -count=1 ./modulos/orquesta-persistence`
Evidencia esperada: JSON de mensaje guardado y snapshot ACK `failed` no contiene DB, DSN, SQL, provider, HOME, OAuth, transcript ni prompt.
Ultima ejecucion: 2026-05-05; ejecutada correctamente.
Riesgos: La denylist de pruebas es defensiva; refs opacas y errores compactos no sustituyen un contrato de conector productivo.
```
