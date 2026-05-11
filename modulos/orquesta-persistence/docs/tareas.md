# Tareas locales: orquesta-persistence

Cada tarea debe ser pequena y cerrada.

## Plantilla

```text
ID:
Objetivo:
Write-set:
Simbolo foco:
Contrato:
Validacion:
Bloqueos:
Estado:
```

## Microtareas

```text
ID: PER-000
Objetivo: Arrancar documentacion local del mini-proyecto persistence sin implementar almacenamiento operativo ni migraciones.
Write-set: docs/tareas.md, docs/decisiones.md, docs/contratos.md, docs/pruebas.md
Simbolo foco: orquesta-persistence
Contrato: N/A
Validacion: Lectura de AGENTS.md, README.md, docs locales y extracto global de `RegistrarProyectoDesdeAppSpec v0`; `PersistenceRepository v0` quedo promovido despues en `../../CONTRATOS.md`.
Bloqueos: Ninguno.
Estado: completada
```

```text
ID: PER-001
Objetivo: Definir contrato local minimo de repositorio para persistir `ProyectoPlanBorradorV0` en un corte futuro.
Write-set: docs/contratos.md
Simbolo foco: RepositorioProyectoPlanBorradorV0
Contrato: `RepositorioProyectoPlanBorradorV0`
Validacion: Revision documental de campos, invariantes, errores y pruebas de contrato previstas.
Bloqueos: `ProyectoPlanBorradorV0` no es contrato definitivo; `PersistenceRepository v0` ya esta compartido en `../../CONTRATOS.md`.
Estado: completada
```

```text
ID: PER-002
Objetivo: Definir contrato local minimo de transaccion/unidad de trabajo para repositorios de persistence.
Write-set: docs/contratos.md
Simbolo foco: TransaccionPersistenceV0
Contrato: `PersistenceUnitOfWorkV0`, `TransaccionPersistenceV0`
Validacion: Revision documental de ciclo begin/commit/rollback y propagacion de errores tecnicos.
Bloqueos: Falta decidir si core consumira un puerto global sincronico, asincronico o mixto.
Estado: completada
```

```text
ID: PER-003
Objetivo: Registrar decisiones iniciales de conectores opacos, alcance y separacion entre contrato local y contrato global.
Write-set: docs/decisiones.md
Simbolo foco: decisiones_per_2026_05_04
Contrato: `RepositorioProyectoPlanBorradorV0`, `TransaccionPersistenceV0`
Validacion: Coherencia con AGENTS.md y README.md del modulo.
Bloqueos: Ninguno para el contrato global; decision del director recibida y registrada en `../../CONTRATOS.md`.
Estado: completada
```

```text
ID: PER-004
Objetivo: Registrar pruebas previstas para repositorios, transacciones y conectores de persistencia sin crear implementacion.
Write-set: docs/pruebas.md
Simbolo foco: pruebas_persistence_v0
Contrato: `RepositorioProyectoPlanBorradorV0`, `TransaccionPersistenceV0`
Validacion: Casos documentados con tipo, comando futuro, evidencia esperada y riesgos.
Bloqueos: No existen paquetes, harness de almacenamiento operativo ni migraciones.
Estado: completada
```

```text
ID: PER-005
Objetivo: Preparar consulta al director para formalizar `PersistenceRepository v0` global cuando haya consumidor real.
Write-set: docs/contratos.md
Simbolo foco: consulta director PersistenceRepository v0
Contrato: `PersistenceRepository v0`
Validacion: Consulta cerrada tras promocion de `PersistenceRepository v0` en `../../CONTRATOS.md`.
Bloqueos: Ninguno; decision del director recibida.
Estado: completada
```

```text
ID: PER-006
Objetivo: Ejecutar corte de contrato material local para `PersistenceRepositoryV0` / guardado de `ProyectoPlanBorradorV0`, sin almacenamiento operativo, migraciones ni queries.
Write-set: docs/schemas/persistence_repository_v0.schema.json; docs/fixtures/persistence_repository_v0/guardar_proyecto_borrador_valido.json; docs/fixtures/persistence_repository_v0/conector_directo_invalido.json; docs/fixtures/persistence_repository_v0/regla_negocio_en_query_invalido.json; docs/contratos.md; docs/tareas.md; docs/pruebas.md; docs/decisiones.md
Simbolo foco: PersistenceRepositoryV0.guardar_proyecto_borrador
Contrato: `PersistenceRepositoryV0`, `ProyectoPlanBorradorV0`
Validacion: `jq empty docs/schemas/*.schema.json docs/fixtures/persistence_repository_v0/*.json`; validacion con `npx ajv-cli` si esta disponible; `git diff --check`.
Bloqueos: Ninguno para promocion global; `PersistenceRepository v0` ya esta compartido en `../../CONTRATOS.md`. No se implementan almacenamiento operativo, migraciones ni queries.
Estado: completada
```

```text
ID: PER-007
Objetivo: Crear DTOs y validacion pura Go para `PersistenceRepositoryV0.guardar_proyecto_borrador`, sin almacenamiento operativo ni adaptador productivo.
Write-set: persistence_repository_v0.go; persistence_repository_v0_test.go; docs/contratos.md; docs/tareas.md; docs/pruebas.md; docs/decisiones.md
Simbolo foco: GuardarProyectoBorradorRequestV0
Contrato: `PersistenceRepositoryV0`, `ProyectoPlanBorradorV0`
Validacion: `go test -count=1 ./modulos/orquesta-persistence`; `git diff --check -- modulos/orquesta-persistence`.
Bloqueos: Ninguno en este microcorte. La unidad de trabajo activa, el envelope global, almacenamiento operativo, migraciones, consultas ejecutables y motor operativo quedan para conectores futuros.
Estado: completada
```

```text
ID: PER-008
Objetivo: Crear adaptador en memoria puro para pruebas de contrato de `PersistenceRepositoryV0.guardar_proyecto_borrador`.
Write-set: memory_contract_adapter_v0.go; memory_contract_adapter_v0_test.go; docs/contratos.md; docs/tareas.md; docs/pruebas.md; docs/decisiones.md
Simbolo foco: InMemoryPersistenceRepositoryContractAdapterV0
Contrato: `PersistenceRepositoryV0`, `GuardarProyectoBorradorRequestV0`
Validacion: `go test -count=1 ./modulos/orquesta-persistence`; `git diff --check -- modulos/orquesta-persistence`.
Bloqueos: No es adaptador productivo, no modela transacciones reales y no introduce almacenamiento operativo, migraciones, consultas ejecutables, motores reales ni filesystem productivo.
Estado: completada
```

```text
ID: PER-009
Objetivo: Sanear tamano de `persistence_repository_v0.go` sin cambiar comportamiento, separando DTOs/constantes, decode estricto/prohibiciones, validacion y helpers.
Write-set: persistence_repository_v0.go; persistence_repository_types_v0.go; persistence_repository_decode_v0.go; persistence_repository_validation_v0.go; persistence_repository_helpers_v0.go; docs/tareas.md; docs/pruebas.md; docs/decisiones.md
Simbolo foco: PersistenceRepositoryV0.guardar_proyecto_borrador
Contrato: `PersistenceRepositoryV0`, `GuardarProyectoBorradorRequestV0`, `GuardarProyectoBorradorMaterialV0`
Validacion: `gofmt`; `go test -count=1 ./modulos/orquesta-persistence`; `git diff --check -- modulos/orquesta-persistence`; `wc -l` de archivos Go tocados.
Bloqueos: Ninguno. No se cambian contratos publicos, JSON tags, fixtures ni errores; no se implementan almacenamiento operativo, consultas ejecutables, migraciones, motores reales ni filesystem productivo.
Estado: completada
```

```text
ID: PER-010
Objetivo: Endurecer `PersistenceRepositoryV0` contra detalles concretos de persistencia en el envelope tecnico del request y metadata.
Write-set: persistence_repository_decode_v0.go; persistence_repository_helpers_v0.go; persistence_repository_validation_v0.go; persistence_repository_v0_test.go; docs/schemas/persistence_repository_v0.schema.json; docs/fixtures/persistence_repository_v0/*.json; docs/contratos.md; docs/tareas.md; docs/pruebas.md; docs/decisiones.md
Simbolo foco: rejectForbiddenPersistenceContractFieldsV0
Contrato: `PersistenceRepositoryV0`, `GuardarProyectoBorradorRequestV0`, `PersistenceAdapterMetadataV0`
Validacion: `gofmt`; `jq empty docs/schemas/*.schema.json docs/fixtures/persistence_repository_v0/*.json`; `go test -count=1 ./modulos/orquesta-persistence`; `git diff --check -- modulos/orquesta-persistence`; `rg` local para motores concretos activos.
Bloqueos: Ninguno. No se implementan adaptadores reales.
Estado: completada
```

```text
ID: PER-011
Objetivo: Implementar un ledger/outbox ACK en memoria para pruebas de contrato, no productivo.
Write-set: outbox_ledger_*_v0.go; outbox_ledger_*_v0_test.go; docs/contratos.md; docs/tareas.md; docs/pruebas.md; docs/decisiones.md
Simbolo foco: InMemoryOutboxLedgerV0
Contrato: `InMemoryOutboxLedgerV0`, `OutboxDispatchAckV0`, `orquestacoreworkflow.OutboxMessageV0`
Validacion: `gofmt`; `go test -count=1 ./modulos/orquesta-persistence ./modulos/orquesta-core-workflow`; `git diff --check -- modulos/orquesta-persistence`; `wc -l` de Go tocados.
Bloqueos: Ninguno. Es memoria de contrato; no introduce almacenamiento operativo, queries, migraciones, filesystem productivo, colas, goroutines de dispatch, sleeps, procesos, runtime real, provider, HOME ni OAuth.
Estado: completada
```

```text
ID: PER-012
Objetivo: Reforzar contrato de ACK `failed` en `InMemoryOutboxLedgerV0` sin cambiar codigo productivo.
Write-set: outbox_ledger_failed_ack_v0_test.go; docs/tareas.md; docs/pruebas.md; docs/decisiones.md
Simbolo foco: InMemoryOutboxLedgerV0.MarkDispatched
Contrato: `InMemoryOutboxLedgerV0`, `OutboxDispatchAckV0`
Validacion: `gofmt`; `go test -count=1 ./modulos/orquesta-persistence`; `git diff --check -- modulos/orquesta-persistence`; `wc -l` de Go tocados.
Bloqueos: Ninguno. El contrato actual marca `error_code` como opcional incluso para `failed`; no se endurece sin una decision contractual nueva.
Estado: completada
```
