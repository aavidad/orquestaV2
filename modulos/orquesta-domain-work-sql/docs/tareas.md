# Tareas: orquesta-domain-work-sql

## Pendientes antes de un bundle DB real

- Crear DDL de ejemplo por motor soportado, fuera del nucleo puro. Debe incluir
  unicidad por `(domain_ref, idempotency_key)` y por `job_ref`.
- Anadir smoke opt-in con DB temporal real por cada dialecto elegido. Debe cubrir
  placeholders, tipos JSON/TEXT, unicidad real, aislamiento y replay tras
  reinicio.
- Definir el bundle de composicion que abre `*sql.DB`, registra driver, lee DSN,
  aplica migraciones, configura pooling y selecciona backend.
- Mapear errores de unique violation por driver mediante `IsUniqueViolation`.
- Decidir si el conector productivo necesita queries con filtros empujados a SQL
  e indices para `domain_ref`, `work_kind`, `status`, `job_ref`,
  `correlation_id`, `idempotency_key` y refs externas.
- Mantener `submit_artifact`, stores de runs, workflow tasks, outbox y estado
  global fuera de este modulo salvo que exista un contrato nuevo revisado.

## No hacer aqui

- No registrar drivers.
- No abrir DSN.
- No crear migraciones automaticas desde `orquesta-domain-work`.
- No cablear el servidor por defecto.
- No compartir DB interna con apps externas.
