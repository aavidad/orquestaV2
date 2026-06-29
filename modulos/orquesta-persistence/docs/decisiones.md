# Decisiones locales: orquesta-persistence

Las decisiones de este archivo solo afectan a `orquesta-persistence`. Si afectan a otro modulo, deben elevarse al director y registrarse en `../../CONTRATOS.md`.

## Plantilla

```text
Fecha:
Decision:
Motivo:
Alternativas:
Impacto:
Contratos afectados:
Estado:
```

## Decisiones iniciales

```text
Fecha: 2026-05-04
Decision: Arrancar `orquesta-persistence` como modulo de adaptadores, sin almacenamiento operativo ni migraciones en el primer corte.
Motivo: `RegistrarProyectoDesdeAppSpec v0` declara que core no persiste en almacenamiento operativo en v0; persistence debe preparar contratos locales antes de exponer un puerto global.
Alternativas: Implementar tablas iniciales; acoplar persistencia al borrador de core; posponer toda definicion hasta que exista `ProyectoPlan` definitivo.
Impacto: Se documentan puertos locales minimos y pruebas previstas, pero no se crean paquetes, queries, migraciones ni dependencias runtime.
Contratos afectados: `RepositorioProyectoPlanBorradorV0`, `TransaccionPersistenceV0`, `PersistenceRepository v0`.
Estado: aceptada_local
```

```text
Fecha: 2026-05-04
Decision: Implementar `PersistenceRepositoryV0.guardar_proyecto_borrador` solo como DTOs y validacion pura Go.
Motivo: Core ya tiene un puerto local y el contrato global esta promovido, pero este modulo aun no debe introducir almacenamiento operativo, migraciones, consultas ejecutables, adaptadores de proveedor ni motores concretos.
Alternativas: Crear adaptador de motor concreto; exponer un repositorio en memoria; importar structs de `orquesta-core`; mantener solo fixtures JSON.
Impacto: `orquesta-persistence` valida forma minima, `idempotency_key`, `payload_version = ProyectoPlanBorradorV0` y ausencia de detalles concretos de persistencia o `query` sin depender de otros modulos. La unidad de trabajo de contrato existe en memoria; el envelope global y el conector operativo quedan como construccion del adaptador futuro.
Contratos afectados: `PersistenceRepositoryV0`, `ProyectoPlanBorradorV0`.
Estado: aceptada_local
```

```text
Fecha: 2026-05-05
Decision: Endurecer `PersistenceRepositoryV0` para rechazar detalles concretos de persistencia en el envelope tecnico del request y metadata.
Motivo: El contrato debe aceptar solo conectores/versiones/refs opacas; el envelope tecnico del request y metadata no pueden transportar seleccion de motor, adaptador productivo, credenciales, nombres reservados ni consultas ejecutables.
Alternativas: Confiar solo en `additionalProperties: false`; documentar la regla sin test; esperar al primer conector real.
Impacto: Decode y validacion pura detectan terminos concretos en strings y campos reservados; el schema local bloquea esos terminos en IDs opacos y textos seguros. Las pruebas cubren request y `adapter_metadata` sin implementar adaptadores reales.
Contratos afectados: `PersistenceRepositoryV0`, `GuardarProyectoBorradorRequestV0`, `PersistenceAdapterMetadataV0`.
Estado: aceptada_local
```

```text
Fecha: 2026-05-04
Decision: Anadir adaptador en memoria solo para pruebas de contrato de `PersistenceRepositoryV0.guardar_proyecto_borrador`.
Motivo: PER-008 necesita ejercitar idempotencia y respuesta tecnica sin adelantar un adaptador productivo ni una unidad de trabajo real.
Alternativas: Crear repositorio productivo en memoria; introducir un motor concreto para pruebas; mantener solo validacion de DTOs.
Impacto: Las pruebas locales pueden validar guardado en RAM, replay idempotente y conflicto por payload distinto. El adaptador no usa almacenamiento operativo, migraciones, consultas ejecutables, motores reales, adaptadores de proveedor ni filesystem productivo.
Contratos afectados: `InMemoryPersistenceRepositoryContractAdapterV0`, `PersistenceRepositoryV0`.
Estado: aceptada_local
```

```text
Fecha: 2026-06-29
Decision: Anadir `InMemoryPersistenceUnitOfWorkV0` como unidad de trabajo en memoria solo para pruebas de contrato.
Motivo: El contrato local ya exige transacciones explicitas; hacia falta evidencia ejecutable de commit, rollback, rechazo de escritura sin transaccion activa y aislamiento abstracto sin aprobar un motor concreto.
Alternativas: Esperar a un conector productivo; activar auto-commit; introducir un motor real de prueba; mantener la transaccion solo como documento.
Impacto: Las pruebas locales validan el ciclo begin/commit/rollback y visibilidad de escrituras en memoria. No se introducen almacenamiento operativo, migraciones, queries ejecutables, motores reales, adaptadores de proveedor ni filesystem productivo.
Contratos afectados: `InMemoryPersistenceUnitOfWorkV0`, `PersistenceUnitOfWorkV0`, `TransaccionPersistenceV0`, `ProyectoPlanBorradorPersistidoV0`.
Estado: aceptada_local
```

```text
Fecha: 2026-05-04
Decision: Ningun motor de base de datos sera operativo prioritario en v0.
Motivo: El director corrige la regla del modulo: persistence debe exponer puertos y conectores opacos, no seleccionar ni documentar motores concretos como camino canonico.
Alternativas: Priorizar un motor concreto; mantener un perfil abstracto sin refs; elegir proveedor desde el request de core.
Impacto: Los contratos usan `connector_profile_refs` opacas. Las pruebas de integracion futuras validaran cualquier motor solo mediante conector aprobado, sin convertirlo en dependencia del nucleo ni del contrato compartido.
Contratos afectados: `RepositorioProyectoPlanBorradorV0`, `TransaccionPersistenceV0`.
Estado: aceptada_local
```

```text
Fecha: 2026-05-04
Decision: Las transacciones se exponen como unidad explicita de persistencia y no como detalle implicito de repositorio.
Motivo: Las reglas del modulo piden contratos claros para transacciones y la persistencia concreta es adaptador, no core.
Alternativas: Auto-commit por metodo; transacciones gestionadas por cada repositorio; filtrar reglas de negocio en queries.
Impacto: El contrato local separa `PersistenceUnitOfWorkV0` de los repositorios concretos y limita las queries a lectura/escritura tecnica.
Contratos afectados: `TransaccionPersistenceV0`, `RepositorioProyectoPlanBorradorV0`.
Estado: aceptada_local
```

```text
Fecha: 2026-05-04
Decision: Formalizar `PersistenceRepository v0` como contrato global tras decision del director.
Motivo: El director promovio `PersistenceRepository v0` en `../../CONTRATOS.md` con `orquesta-persistence` como propietario y `orquesta-core` como consumidor autorizado.
Alternativas: Mantener solo contrato local; esperar a `ProyectoPlan` definitivo; implementar almacenamiento operativo antes del contrato compartido.
Impacto: La consulta queda cerrada y el contrato local pasa a ser detalle canonico del contrato compartido; sigue prohibido implementar almacenamiento operativo, migraciones o queries en este corte.
Contratos afectados: `PersistenceRepository v0`, `RepositorioProyectoPlanBorradorV0`, `PersistenceUnitOfWorkV0`.
Estado: consultada_aceptada
```

```text
Fecha: 2026-05-04
Decision: Materializar `PersistenceRepositoryV0` solo como contrato local validable por JSON Schema y fixtures.
Motivo: PER-006 necesita una superficie concreta para request/response/errores de persistencia de `ProyectoPlanBorradorV0`; el contrato global ya referencia este detalle local desde `../../CONTRATOS.md`.
Alternativas: Crear migraciones/tablas; implementar queries de repositorio; esperar a `ProyectoPlan` definitivo antes de cualquier fixture.
Impacto: Hay schema y fixtures locales para contrato de guardado, con perfiles de conector solo como refs opacas de adaptador y sin almacenamiento operativo, consultas ejecutables, tablas, provider hardcodeado ni reglas de negocio en queries.
Contratos afectados: `PersistenceRepositoryV0`, `RepositorioProyectoPlanBorradorV0`, `ProyectoPlanBorradorV0`.
Estado: aceptada_local
```

```text
Fecha: 2026-05-04
Decision: Dividir `persistence_repository_v0.go` por responsabilidades pequenas sin cambiar comportamiento.
Motivo: La regla global de deuda de tamano marcaba el archivo en zona roja y recomendaba separar DTOs, decode estricto, validacion de material y validacion de request/response.
Alternativas: Mantener el archivo monolitico; partir tambien tests o adaptador en memoria; introducir nuevas abstracciones funcionales.
Impacto: El contrato conserva nombres publicos, JSON tags, fixtures y errores; el codigo queda separado en tipos, decode, validacion y helpers, sin almacenamiento operativo, consultas ejecutables, migraciones, motores reales ni filesystem productivo.
Contratos afectados: `PersistenceRepositoryV0`, `GuardarProyectoBorradorRequestV0`, `GuardarProyectoBorradorMaterialV0`.
Estado: aceptada_local
```

```text
Fecha: 2026-05-05
Decision: Anadir `InMemoryOutboxLedgerV0` como ledger ACK en memoria solo para pruebas de contrato.
Motivo: PER-011 necesita verificar guardado de `OutboxMessageV0`, filtros de pendientes, ACK e idempotencia sin activar dispatchers ni adelantar un adaptador productivo.
Alternativas: Crear una cola real; persistir el outbox en almacenamiento operativo; modelar el dispatcher con goroutines; mantener solo tests del productor de workflow.
Impacto: `orquesta-persistence` consume el contrato publico `orquestacoreworkflow.OutboxMessageV0` y valida con `ValidateOutboxMessageV0`; los ACKs quedan como snapshots compactos locales. No se introducen almacenamiento operativo, queries, migraciones, filesystem productivo, procesos, sleeps, runtime real, provider, HOME ni OAuth.
Contratos afectados: `InMemoryOutboxLedgerV0`, `OutboxDispatchAckV0`, `OutboxMessageV0`.
Estado: aceptada_local
```

```text
Fecha: 2026-05-05
Decision: Reforzar con pruebas el comportamiento de ACK `failed` en `InMemoryOutboxLedgerV0` sin cambiar codigo productivo.
Motivo: PER-012 debe fijar que un ACK `failed` es una salida de pendientes igual que `dispatched`, mantiene idempotencia por snapshot equivalente y rechaza replays incompatibles.
Alternativas: Exigir `error_code` obligatorio para `failed`; dejar los mensajes fallidos en pendientes; crear cola, dispatcher o almacenamiento operativo para probar fallos.
Impacto: Los tests cubren `error_code` compacto, `evidence_refs` opacas, retirada de `ListPending`, replay idempotente, conflicto por `error_code` o `dispatch_ref` distinto y snapshots sin DB, DSN, SQL, provider, HOME, OAuth, transcript ni prompt. `error_code` sigue opcional porque el contrato actual lo declara asi y el ledger no debe definir una taxonomia de fallos.
Contratos afectados: `InMemoryOutboxLedgerV0`, `OutboxDispatchAckV0`.
Estado: aceptada_local
```
