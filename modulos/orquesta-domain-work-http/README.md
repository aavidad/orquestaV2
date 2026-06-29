# orquesta-domain-work-http

Adaptador HTTP neutral para `orquesta-domain-work`.

Responsabilidad:

- enviar `DomainWorkJobRequestV0` a una app externa por HTTP;
- enviar `DomainWorkArtifactSubmissionV0` a una app externa por HTTP;
- devolver `DomainWorkJobV0` y `DomainWorkArtifactReceiptV0` sin conocer OPES,
  Codex, runtime, DB ni filesystem interno de la app.

Este modulo es opt-in desde composicion. La app externa debe persistir y validar
sus datos; Orquesta solo transporta refs opacas y artefactos por contrato.

Politica de egress v0:

- la composicion que use `ORQUESTA_DOMAIN_WORK_HTTP_BASE_URL` debe declarar
  `ORQUESTA_DOMAIN_WORK_HTTP_EGRESS_MODE=smoke_local|allowlist`;
- `smoke_local` solo permite loopback temporal para smokes acotados;
- `allowlist` exige `ORQUESTA_DOMAIN_WORK_HTTP_ALLOWED_HOSTS` con hosts o
  `host:puerto` confirmados por la composicion;
- si la composicion declara `ORQUESTA_DOMAIN_WORK_HTTP_DOMAIN_REF=opes`, el
  servidor rechaza el backend HTTP neutral: OPES debe entrar por el conector
  dedicado con guardas de destino temporal;
- se rechazan credenciales, query y fragment en la base URL;
- `create_job` y `submit_artifact` siguen siendo paths relativos sin query ni
  host override.
