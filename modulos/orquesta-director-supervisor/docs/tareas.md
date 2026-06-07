# Tareas: orquesta-director-supervisor

## Backlog local

### DSV-000 - Contrato local

Write-set: docs locales.

Cierre:

- contrato, decisiones, pruebas y tareas documentadas.

### DSV-001 - Politica de decision v0

Write-set: tipos, validacion, helpers y caso de uso.

Cierre:

- `DecideDirectorSupervisorNextActionV0` implementado sin efectos externos.

### DSV-002 - Pruebas de estados

Write-set: tests locales.

Cierre:

- todos los status del ciclo quedan cubiertos.

### DSV-003 - Registro global

Write-set: `modulos/README.md`, `modulos/CONTRATOS.md`, estado del nucleo y verificador.

Cierre:

- el modulo entra en `verificar_nucleo_orquesta_v2.sh`.

### DSV-004 - Recomendacion autonoma local

Write-set: tipos, helpers, caso de uso, tests y docs locales.

Cierre:

- la decision expone `autonomous_recommendation`;
- outbox y candidatos pendientes recomiendan `wait` sin operador manual.

### DSV-005 - Briefing canonico del Director

Write-set: `supervisor_briefing_*_v0.go`, tests y docs locales.

Cierre:

- una `DirectorSupervisorDecisionV0` se proyecta a
  `DirectorSupervisorBriefingV0`;
- el briefing expone `next_action`, `action_queue` y `timeline` compacta;
- las acciones quedan tipadas (`run_director_step`, `dispatch_outbox`,
  `wait_external_signal`, `ask_director`, `review_blocker`,
  `close_or_idle`, `stop_budget_exhausted`, `inspect_error`);
- el modulo no ejecuta acciones, no arranca runtime y no conoce proveedor,
  modelo, DB, web, OPES ni Codex.
