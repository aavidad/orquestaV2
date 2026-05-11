# Contratos locales: orquesta-persistence

Registra puertos, DTOs y eventos que `orquesta-persistence` expone o consume.

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

## Contratos locales v0

```text
Nombre: InMemoryPersistenceRepositoryContractAdapterV0
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-persistence
Consumidores: pruebas de contrato locales de `orquesta-persistence`.
Campos:
  - `guardar_proyecto_borrador(contexto, request)`:
    - `contexto`: contexto tecnico de cancelacion para pruebas.
    - `request`: `GuardarProyectoBorradorRequestV0` validado con las reglas puras del contrato.
  - `response`: `GuardarProyectoBorradorResponseV0` sintetico en memoria.
Invariantes:
  - Es un adaptador puro de pruebas; no es productivo.
  - No usa almacenamiento operativo, migraciones, consultas ejecutables, motores reales, adaptadores de proveedor ni filesystem productivo.
  - Guarda solo en memoria del proceso y pierde el estado al descartar la instancia.
  - Reusar la misma `idempotency_key` con payload equivalente devuelve la misma respuesta.
  - Reusar la misma `idempotency_key` con payload distinto falla con `conflicto_idempotencia`.
  - No modela unidad de trabajo ni transacciones reales.
Errores publicos:
  - `payload_invalido`
  - `conflicto_idempotencia`
  - `persistencia_no_disponible`
Pruebas de contrato:
  - `memory_contract_adapter_v0_test.go` valida guardado, idempotencia equivalente, conflicto por payload distinto y rechazo de request invalido.
```

```text
Nombre: PersistenceRepositoryV0
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-persistence
Consumidores: `orquesta-core` autorizado por `../../CONTRATOS.md`; adaptadores internos de orquesta-persistence.
Campos:
  - `guardar_proyecto_borrador(contexto, unidad_trabajo, request)`:
    - `contexto`: contexto tecnico de cancelacion, trazabilidad, deadline y correlation_id.
    - `unidad_trabajo`: handle opaco activo de transaccion/unidad de trabajo de persistence.
    - `request.request_id`: identificador opaco de la solicitud.
    - `request.idempotency_key`: clave opaca requerida para deduplicacion tecnica.
    - `request.payload_version`: version del payload; v0 acepta `ProyectoPlanBorradorV0`.
    - `request.payload`: borrador serializable producido por core, sin reinterpretar reglas de negocio.
  - `response.status`: `persistido`.
  - `response.proyecto_id`: identificador tecnico opaco asignado por persistence.
  - `response.idempotency_key`: eco de la clave tecnica aceptada.
  - `response.payload_version`: version persistida.
  - `response.created_at` y `response.updated_at`: timestamps tecnicos.
  - `adapter_metadata.connector_profile_refs`: referencias opacas a perfiles de conector declarados por el adaptador.
  - `adapter_metadata.capacidades`: capacidades tecnicas declarativas como `transacciones`, `idempotencia` y `payload_versionado`.
Invariantes:
  - La persistencia concreta es adaptador; core no conoce tablas, consultas ejecutables, adaptador de proveedor, conexion, cursor ni motor operativo.
  - Los IDs del contrato son opacos y no sustituyen identificadores de dominio futuros.
  - El payload se conserva versionado como contrato de core; persistence no ejecuta ni valida reglas de negocio de core.
  - El conector de persistencia puede declararse solo como referencia opaca de adaptador o en la unidad de trabajo interna, nunca como campo de decision del request de core.
  - Toda escritura ocurre dentro de una unidad de trabajo/transaccion activa.
  - La idempotencia evita duplicados tecnicos para la misma clave y payload equivalente.
  - No se aceptan consultas ejecutables, tablas, provider hardcodeado, queries ni filtros por fases, modelos, autonomia, capacidad, runtime o handoff.
Errores publicos:
  - `persistencia_no_disponible`
  - `payload_invalido`
  - `conector_persistencia_no_soportado`
  - `conflicto_idempotencia`
  - `transaccion_fallida`
  - `regla_negocio_en_adaptador`
Pruebas de contrato:
  - `docs/schemas/persistence_repository_v0.schema.json` valida el fixture positivo `guardar_proyecto_borrador_valido.json`.
  - `conector_directo_invalido.json` debe fallar porque intenta poner un conector de persistencia en el request del contrato.
  - `regla_negocio_en_query_invalido.json` debe fallar porque intenta introducir una query/filtros de negocio en el contrato material.
  - `persistence_repository_v0.go` valida en Go la forma minima de `guardar_proyecto_borrador`, `idempotency_key`, `payload_version = ProyectoPlanBorradorV0` y rechazo de detalles concretos de persistencia, `request.query` o `query` material.
  - `GuardarProyectoBorradorMaterialV0` es solo DTO de fixture/contrato; la unidad de trabajo activa y el envelope global los construira un adaptador futuro.
  - `jq empty docs/schemas/*.schema.json docs/fixtures/persistence_repository_v0/*.json`.
```

```text
Nombre: RepositorioProyectoPlanBorradorV0
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-persistence
Consumidores: `orquesta-core` autorizado por `../../CONTRATOS.md`; adaptadores internos de orquesta-persistence.
Campos:
  - `guardar_borrador(contexto, transaccion, proyecto_plan_borrador, opciones)`:
    - `contexto`: contexto tecnico de cancelacion, trazabilidad y deadline.
    - `transaccion`: `TransaccionPersistenceV0` activa.
    - `proyecto_plan_borrador`: payload serializable compatible con `ProyectoPlanBorradorV0` de `RegistrarProyectoDesdeAppSpec v0`.
    - `opciones.idempotency_key`: clave requerida para evitar duplicados.
    - `opciones.version_contrato`: version del payload de entrada, inicialmente `ProyectoPlanBorradorV0`.
  - `obtener_borrador_por_id(contexto, transaccion, proyecto_id)`:
    - `proyecto_id`: identificador tecnico estable devuelto por persistence.
  - `obtener_borrador_por_idempotency_key(contexto, transaccion, idempotency_key)`:
    - `idempotency_key`: clave de idempotencia del registro de proyecto.
Invariantes:
  - No ejecuta reglas de negocio de core.
  - No decide fases, modelos, autonomia, capacidad, runtime ni handoff.
  - No asume que ningun motor concreto cubre toda la semantica de persistencia.
  - Conserva el payload versionado sin depender de structs privados de core.
  - Toda escritura ocurre dentro de una `TransaccionPersistenceV0` activa.
  - Reintentar con la misma `idempotency_key` no debe crear duplicados.
  - Los filtros son tecnicos: identificador, idempotencia, version y auditoria; no esconden reglas de negocio.
Errores:
  - `payload_invalido`
  - `version_contrato_no_soportada`
  - `idempotency_key_requerida`
  - `transaccion_requerida`
  - `transaccion_no_activa`
  - `duplicado_idempotente_incompatible`
  - `borrador_no_encontrado`
  - `conflicto_concurrencia`
  - `conector_persistencia_no_soportado`
  - `fallo_persistencia`
Pruebas de contrato:
  - Guardar y recuperar un `ProyectoPlanBorradorV0` minimo preserva el payload versionado.
  - Dos guardados con la misma `idempotency_key` y payload equivalente devuelven el mismo registro tecnico.
  - Dos guardados con la misma `idempotency_key` y payload incompatible fallan con `duplicado_idempotente_incompatible`.
  - Guardar sin transaccion activa falla con `transaccion_requerida` o `transaccion_no_activa`.
  - Los conectores reales se validaran por contrato propio cuando el director apruebe un motor concreto.
```

```text
Nombre: PersistenceUnitOfWorkV0
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-persistence
Consumidores: `orquesta-core` autorizado por `../../CONTRATOS.md`; adaptadores internos de orquesta-persistence.
Campos:
  - `begin(contexto, opciones)`:
    - `contexto`: contexto tecnico de cancelacion, trazabilidad y deadline.
    - `opciones.read_only`: indica si la transaccion impide escrituras.
    - `opciones.isolation`: aislamiento solicitado como valor abstracto documentado por persistence.
    - `opciones.connector_profile_ref`: referencia opaca al perfil de conector objetivo.
  - `commit(contexto, transaccion)`:
    - `transaccion`: `TransaccionPersistenceV0` activa.
  - `rollback(contexto, transaccion)`:
    - `transaccion`: `TransaccionPersistenceV0` activa.
Invariantes:
  - `begin` devuelve una unica transaccion activa.
  - `commit` cierra la transaccion y confirma solo si no hubo error previo incompatible.
  - `rollback` es idempotente para facilitar limpieza en errores.
  - No expone conexion, cursor, pool ni API privada de proveedor.
  - El aislamiento abstracto debe mapearse explicitamente por conector en la implementacion futura.
  - No mezcla transacciones de conectores distintos.
Errores:
  - `conector_persistencia_no_soportado`
  - `aislamiento_no_soportado`
  - `transaccion_no_activa`
  - `transaccion_ya_cerrada`
  - `commit_fallido`
  - `rollback_fallido`
  - `timeout_persistencia`
Pruebas de contrato:
  - Una transaccion con commit hace visible la escritura en lecturas posteriores.
  - Una transaccion con rollback no deja visible la escritura.
  - Doble rollback no rompe la limpieza.
  - Commit despues de rollback falla con `transaccion_ya_cerrada`.
  - Los niveles de aislamiento no soportados fallan antes de ejecutar queries.
```

```text
Nombre: TransaccionPersistenceV0
Tipo: dto
Version: v0
Propietario: orquesta-persistence
Consumidores: `RepositorioProyectoPlanBorradorV0`, repositorios locales futuros.
Campos:
  - `id`: identificador tecnico opaco de la transaccion.
  - `connector_profile_ref`: referencia opaca al perfil de conector de persistencia.
  - `read_only`: booleano.
  - `isolation`: aislamiento abstracto solicitado.
  - `estado`: `activa` | `confirmada` | `revertida` | `fallida`.
  - `started_at`: timestamp tecnico.
Invariantes:
  - Es un handle opaco, no contiene conexion ni detalle de proveedor.
  - Solo persistence crea y muta su estado.
  - No debe serializarse como contrato de dominio.
Errores:
  - `transaccion_no_activa`
  - `transaccion_ya_cerrada`
Pruebas de contrato:
  - Los repositorios rechazan handles cerrados.
  - El estado cambia a `confirmada` solo tras commit correcto.
  - El estado cambia a `revertida` tras rollback correcto.
```

```text
Nombre: ProyectoPlanBorradorPersistidoV0
Tipo: dto
Version: v0
Propietario: orquesta-persistence
Consumidores: `RepositorioProyectoPlanBorradorV0`; posible adaptador de lectura futuro.
Campos:
  - `proyecto_id`: identificador tecnico asignado por persistence.
  - `idempotency_key`: clave tecnica recibida.
  - `version_contrato`: version del payload persistido.
  - `payload`: `ProyectoPlanBorradorV0` serializable.
  - `created_at`: timestamp tecnico.
  - `updated_at`: timestamp tecnico.
Invariantes:
  - `payload` conserva el contrato de origen sin reinterpretar reglas de negocio.
  - `proyecto_id` no reemplaza identificadores de dominio si core define otros en el futuro.
  - `idempotency_key` es unica para el alcance definido por `PersistenceRepository v0` en `../../CONTRATOS.md`.
Errores:
  - `payload_invalido`
  - `version_contrato_no_soportada`
Pruebas de contrato:
  - Serializacion y deserializacion preservan campos desconocidos permitidos por el contrato de origen.
  - El identificador tecnico se mantiene estable entre lecturas.
```

```text
Nombre: InMemoryOutboxLedgerV0
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-persistence
Consumidores: pruebas de contrato locales de `orquesta-persistence`; dispatchers/adaptadores futuros solo como referencia de comportamiento esperado.
Campos:
  - `guardar_pendientes` / `save_pending(contexto, mensajes)`:
    - `mensajes`: lista de `orquestacoreworkflow.OutboxMessageV0`.
    - Cada mensaje se valida con `ValidateOutboxMessageV0` antes de guardarse.
  - `listar_pendientes` / `list_pending(contexto, filtro)`:
    - `filtro.run_id`: opcional, referencia opaca del run.
    - `filtro.target_port`: opcional, puerto contractual del outbox.
    - Devuelve mensajes publicos `OutboxMessageV0`, no registros internos.
  - `registrar_ack` / `mark_dispatched(contexto, ack)`:
    - `ack.message_id`: identificador publico del outbox.
    - `ack.run_id`: run opaco esperado.
    - `ack.target_port`: puerto contractual esperado.
    - `ack.status`: `dispatched` o `failed`.
    - `ack.dispatch_ref`: referencia opaca del intento de entrega.
    - `ack.dispatched_at`: timestamp tecnico.
    - `ack.error_code`: opcional.
    - `ack.evidence_refs`: referencias opacas opcionales.
Invariantes:
  - Es un ledger en memoria solo para pruebas de contrato; no es productivo.
  - No usa almacenamiento operativo, migraciones, queries, colas, filesystem productivo, goroutines de dispatch, sleeps, procesos ni runtime real.
  - No decide ni resuelve proveedor, HOME, OAuth, credenciales, secretos, transcripts, prompts ni contexto masivo.
  - Guarda solo snapshots publicos del mensaje y ACK normalizados; pierde el estado al descartar la instancia.
  - Reusar el mismo `message_id` o `idempotency_key` con mensaje equivalente devuelve snapshot estable y no duplica pendientes.
  - Reusar el mismo `message_id` o `idempotency_key` con mensaje incompatible falla con `conflicto_idempotencia`.
  - Un ACK equivalente es idempotente y devuelve el mismo snapshot.
  - Un ACK incompatible para el mismo mensaje falla con `conflicto_idempotencia`.
  - Un mensaje con ACK `dispatched` o `failed` deja de aparecer en pendientes.
Errores publicos:
  - `payload_invalido`
  - `conflicto_idempotencia`
  - `persistencia_no_disponible`
  - errores publicos propagados de `ValidateOutboxMessageV0`: `outbox_invalido`, `outbox_tipo_no_soportado`, `detalle_prohibido`.
Pruebas de contrato:
  - `outbox_ledger_memory_v0_test.go` valida guardado de `LaunchRuntimeAgent`, `StopRuntimeAgent` y `SendDirectorQuestion`, filtros por run/target, ACK, idempotencia, conflictos y snapshots sin detalles prohibidos.
```

## Consulta al director cerrada

```text
Modulo origen: orquesta-persistence
Modulo afectado: orquesta-core
Decision del director: `PersistenceRepository v0` queda promovido a contrato compartido en `../../CONTRATOS.md`.
Consumidor autorizado: `orquesta-core`, para persistir un `ProyectoPlanBorradorV0` producido por `RegistrarProyectoDesdeAppSpec v0`.
Resultado: La consulta queda cerrada; `PersistenceRepository` ya no esta pendiente globalmente.
Impacto: El detalle canonico local sigue en este modulo, con schema y fixtures locales; no se crean almacenamiento operativo, migraciones ni queries en este corte.
Estado: consultada_aceptada
```
