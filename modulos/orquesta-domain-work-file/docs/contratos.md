# Contratos: orquesta-domain-work-file

## FileDomainWorkJobCreatorV0

Implementa `DomainWorkJobRecordStorePortV0` como conector durable file-based.

Entrada:

- directorio absoluto para snapshot local;
- `DomainWorkJobRequestV0` normalizado/validado desde `orquesta-domain-work`.

Salida:

- `DomainWorkJobV0{status=accepted}` con `job_ref` compacto si el request es
  valido;
- `DomainWorkJobV0{status=invalid}` si el request no valida o si hay conflicto
  de idempotencia;
- `DomainWorkJobRecordV0{request, job}` para lectura filtrada de jobs
  persistidos;
- error Go para fallos operativos: contexto cancelado, snapshot corrupto,
  escritura fallida o directorio invalido.

Snapshot:

- archivo: `domain_work_jobs_v0.json`;
- schema: `domain_work_file_job_creator_snapshot.v0`;
- registros ordenados con `domain_ref`, `idempotency_key`, `fingerprint`,
  `request` normalizado y `job` aceptado.

Idempotencia:

- clave logica: `domain_ref + idempotency_key`;
- `request_id` no participa en la huella;
- replay equivalente tras reinicio devuelve el mismo `job_ref`;
- misma clave con contrato distinto devuelve
  `domain_work_file_idempotency_conflict`;
- los conflictos y requests invalidos no se escriben.

Lectura:

- `ListDomainWorkJobRecordsV0` lee del estado cargado desde el snapshot, no
  reinterpreta el archivo en cada llamada;
- filtra con AND por `domain_ref`, `work_kind`, `job_ref`, `correlation_id`,
  `idempotency_key`, `status` y pares exactos de `external_refs`;
- ordena por `job_ref` ascendente, igual que `ListDomainWorkJobsV0`;
- aplica `limit` despues del filtro y del orden; `limit <= 0` significa sin
  limite;
- devuelve copias defensivas de request y job;
- no expone fingerprint como contrato neutral: la huella sigue siendo detalle de
  idempotencia del adaptador.

Frontera:

- usa filesystem local y por tanto es adaptador, no nucleo puro;
- no importa DB, red, runtime, Codex, OPES, MCP, web ni composicion de servidor;
- no decide plan, agentes, modelos, proveedor, cuotas ni estrategia.
