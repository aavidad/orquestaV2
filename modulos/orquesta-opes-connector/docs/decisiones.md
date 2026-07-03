# Decisiones

## T12: conector REST y cierre funcional OPES

Fecha: 2026-05-27.
Actualizacion: 2026-06-28.

El conector mantiene responsabilidad de adaptador REST opt-in. Sus tests y el
fake local prueban forma, idempotencia y receipts; el cierre funcional real de
derivados/cierre OPES queda documentado fuera del conector, en
`docs/runbooks/resultado_smoke_opes_derivados_goal_first_real_2026-06-28.md`.
Ese smoke temporal goal-first alcanzo `completed_syllabus_package` con OPES
temporal, Orquesta temporal, `ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux`,
receipts de dominio y cierre aceptado por Orquesta.

Decision operativa: T12 no se reabre como bloqueo funcional por falta de codigo
del conector ni por ausencia generica de entorno temporal. Las revalidaciones
con efectos se tratan como residuales operativos y deben aportar OPES temporal,
scope duro por `job_ref`, `program_id`, `topic_id`, `correlation_id` o cola
temporal dedicada, confirmacion explicita de efectos y backend goal-first
`app_server_tmux`. Sin esas precondiciones se registra bloqueo externo de
revalidacion; no se cambia el contrato del conector para compensarlo.

No hacer:

- leer DB, filesystem, workers ni colas internas de OPES;
- interpretar `worktree_ref` o `branch_ref` como rutas o ramas;
- enviar modelo, runtime, lease o sesiones a OPES;
- relanzar trabajos fake para cerrar una evidencia real.
- declarar de nuevo T12 como bloqueo vigente sin regresion demostrada del
  contrato REST, del scope temporal o del cierre goal-first documentado.
