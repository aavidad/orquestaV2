# Smoke app externa no-OPES real

Fecha: 2026-05-22.

Objetivo: probar una app externa temporal no-OPES por HTTP/file con submitter
real opt-in. Orquesta usa `DomainWork` neutral, refs opacas y cierre operativo;
la app externa conserva jobs/artefactos en su propio snapshot JSON temporal.

Comando:

```bash
./scripts/smoke_external_domain_non_opes_real.sh
```

Opcionales:

```bash
ORQUESTA_KEEP_SMOKE_DIR=1 ./scripts/smoke_external_domain_non_opes_real.sh
SMOKE_ID=manual-001 ./scripts/smoke_external_domain_non_opes_real.sh
```

Ruta probada:

1. Arranca app HTTP temporal no-OPES con endpoints neutrales:
   `/api/domain-work/jobs` y `/api/domain-work/artifacts`.
2. Arranca `orquesta-server` temporal con
   `ORQUESTA_DOMAIN_WORK_HTTP_BASE_URL=<app-temporal>`.
3. `POST /api/v0/domain-work` crea job externo real por HTTP y replaya por
   idempotencia.
4. `POST /api/v0/domain-work` envia artefacto manual y recibe `receipt_ref`.
5. `POST /api/v0/external-work/run` crea run neutral sin OPES.
6. Supervisor ejecuta runtime fake acotado, registra ACK/delivery, envia
   `submit_artifact` por el submitter HTTP neutral, genera review aceptada,
   evidencia durable de required tests y cierre operativo.

Criterio de exito:

- salida `smoke=external-domain-non-opes-real`;
- `job_ref` y `replay_job_ref` iguales;
- `manual_receipt_ref` no vacio;
- `external_app_state` contiene al menos dos artefactos recibidos;
- `operational_plan_status=closed`;
- `opes_touched=false`.

Limites:

- No consume Codex real; usa `codex-fake` para mantener el smoke barato.
- No define politica productiva de tests de dominio; usa validador local
  allowlist por `RequiredTestRunner`.
- No comparte DB ni filesystem interno entre Orquesta y la app externa.
