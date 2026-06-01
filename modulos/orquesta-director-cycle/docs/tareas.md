# Tareas: orquesta-director-cycle

## Estado

- DCS-000: contexto local del microproyecto. Estado: cerrado.
- DCS-001: contrato `ExecuteDirectorCycleStepV0`. Estado: cerrado.
- DCS-002: composicion tick-input, runner y cycle-outbox. Estado: cerrado.
- DCS-003: pruebas de outbox nueva y outbox pendiente previa. Estado: cerrado.
- DCS-006: cierre de DTI-005/DTI-006 como ensamblador superior: lista outbox
  pendiente por puerto, pasa refs a tick-input y conserva candidates externos.
  Estado: cerrado.
- DCS-007: replay de evento durable con ledger vacio y candidate explicito de
  recovery. Estado: cerrado.
- DCS-004: coordinador multi-step controlado por presupuesto externo:
  `ExecuteDirectorCycleStepsV0` usa `max_steps`, conserva
  `ExecuteDirectorCycleStepV0` como step atomico, exige `snapshot_port` para
  pasos posteriores y corta en `wait_outbox`, `wait_external`, `blocked`,
  `needs_director`, `stop_quiescent`, `stop_error` o `stop_max_steps`.
  Estado: cerrado.
- DCS-005: integracion con candidates reales de planificacion/programacion:
  dos `WorkCandidates` con `WorkClaims` compartidos recorren tick-input,
  scheduler, runner, core-workflow y ledger; se registran gates, agentes y dos
  outbox `LaunchRuntimeAgent` sin despachar runtime. Estado: cerrado.

## Siguientes Microtareas

No hay siguientes microtareas locales abiertas tras DCS-004/DCS-005.
