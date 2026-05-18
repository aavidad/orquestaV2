# Contratos: orquesta-domain-work-memory

## InMemoryDomainWorkJobCreatorV0

Implementa `DomainWorkJobRecordStorePortV0` como conector in-memory neutral.

Entrada:

- `DomainWorkJobRequestV0` del modulo `orquesta-domain-work`.

Salida:

- `DomainWorkJobV0` con `status=accepted` y `job_ref` compacto si el request
  es valido;
- `DomainWorkJobV0` con `status=invalid` e `issues` si el request no valida o
  hay conflicto de idempotencia;
- `DomainWorkJobRecordV0{request, job}` para lecturas filtradas de jobs
  aceptados;
- error Go solo para fallos operativos como `context.Context` cancelado.

Idempotencia:

- la clave de ledger es `domain_ref + idempotency_key`;
- el request se normaliza antes de validar y antes de calcular la huella;
- la huella compara contrato de trabajo, no el `request_id` tecnico;
- un retry con la misma clave y la misma huella devuelve el mismo `job_ref`;
- la misma clave con huella distinta devuelve
  `domain_work_memory_idempotency_conflict` y no sobrescribe estado.

Lectura:

- `ListDomainWorkJobRecordsV0` filtra con AND por `domain_ref`, `work_kind`,
  `job_ref`, `correlation_id`, `idempotency_key`, `status` y pares exactos de
  `external_refs`;
- ordena por `job_ref` ascendente;
- aplica `limit` despues de filtrar; `limit <= 0` significa sin limite;
- devuelve copias defensivas de request y job.

Frontera:

- no usa DB, filesystem, HTTP, runtime, Codex, OPES ni conectores de producto;
- no decide plan, agente, modelo, proveedor, cuotas ni estrategia;
- no entrega artefactos: ese contrato pertenece a
  `DomainWorkArtifactSubmitterPortV0` y debe implementarse en otro adaptador.

Uso previsto:

- pruebas offline del nucleo y de futuros source ports;
- referencia volatil de comportamiento para conectores durables como
  `orquesta-domain-work-file` o futuros conectores de DB, cola, REST o app
  propietaria;
- integracion con casos de uso neutrales como
  `orquesta-document-plan-expander`.
