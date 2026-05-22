# orquesta-domain-work-http

Adaptador HTTP neutral para `orquesta-domain-work`.

Responsabilidad:

- enviar `DomainWorkJobRequestV0` a una app externa por HTTP;
- enviar `DomainWorkArtifactSubmissionV0` a una app externa por HTTP;
- devolver `DomainWorkJobV0` y `DomainWorkArtifactReceiptV0` sin conocer OPES,
  Codex, runtime, DB ni filesystem interno de la app.

Este modulo es opt-in desde composicion. La app externa debe persistir y validar
sus datos; Orquesta solo transporta refs opacas y artefactos por contrato.
