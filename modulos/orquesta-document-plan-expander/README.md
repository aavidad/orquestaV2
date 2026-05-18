# orquesta-document-plan-expander

Expander neutral para convertir `DomainDocumentPlanV0` en jobs de dominio
derivados.

Entrada:

- `orquesta-domain-work.DomainDocumentPlanV0`

Salida:

- `[]orquesta-domain-work.DomainWorkJobRequestV0`

Responsabilidad:

- crear jobs para secciones, visuales y pasos de revision declarados por el
  plan;
- conservar refs compactas, criterios, constraints, evidencias e idempotencia;
- declarar `expected_artifact_type` compatible con los contratos de dominio.

La expansion pura no ejecuta jobs. El caso de uso hexagonal incluido puede
llamar a `DomainWorkJobCreatorPortV0`, pero solo mediante puerto inyectado. El
paquete no conoce apps concretas, DB, colas, Codex ni runtime.
Puede usarse con adaptadores externos como `orquesta-domain-work-memory` u
`orquesta-domain-work-file` sin importarlos desde este paquete.

## Estado del caso de uso hexagonal

Implementado hoy:

- expansion pura `DomainDocumentPlanV0 -> DomainWorkJobRequestV0[]`;
- validacion del plan y de cada request derivado;
- ausencia de jobs parciales si aparece cualquier issue;
- caso de uso de aplicacion `CreateDomainDocumentPlanDerivedJobsV0`, que
  expande y crea cada job con `DomainWorkJobCreatorPortV0`;
- guard arquitectonico contra DB, colas, filesystem, red, Codex, adaptadores
  concretos y runtime;
- idempotencia estable por `RequestID`/`IdempotencyKey`; la deduplicacion real
  queda en el adaptador que implemente el puerto.
