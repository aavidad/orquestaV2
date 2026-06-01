# Tareas: orquesta-director-runner

## Estado

- DCR-000: contexto local del microproyecto. Estado: cerrado.
- DCR-001: contrato `RunDirectorCycleV0`. Estado: cerrado.
- DCR-002: puertos `DirectorSchedulerPortV0` y `WorkflowCommandPortV0`. Estado: cerrado.
- DCR-003: adaptador directo de scheduler sin efectos externos. Estado: cerrado.
- DCR-004: pruebas unitarias de espera, outbox y error workflow. Estado: cerrado.
- DCR-005: prueba integrada con scheduler real y workflow event-store en memoria. Estado: cerrado.
- DCR-006: `max_outbox` opt-in para acumular varios outbox y habilitar batch externo. Estado: cerrado.

## Siguientes Microtareas

- DCR-007: conector superior que prepare candidates/snapshot desde contratos
  compactos. Estado: reconciliado el 2026-06-01; no es tarea de codigo del
  runner. La preparacion del `DirectorSchedulerTickInputV0` vive en capas
  superiores como `orquesta-director-tick-input`/`orquesta-director-cycle`. El
  runner solo consume ese tick ya preparado y lo entrega al scheduler por
  puerto.
- DCR-008: persistencia/event-store como puerto externo, sin elegir DB en nucleo.
  Estado: cerrado el 2026-06-01 con `WorkflowEventStorePortV0` y
  `StoredWorkflowCommandPortV0`.
- DCR-009: ciclo progresivo multi-fase con orquestacion real de app de prueba.
  Estado: reclasificado fuera del runner el 2026-06-01. El runner es un tick
  acotado con `DirectorSchedulerTickInputV0` ya preparado; repetir fases,
  refrescar snapshots y ejecutar una app de prueba pertenecen a
  `orquesta-director-cycle`, `orquesta-director-supervised-burst`,
  `orquesta-orchestration-core` o a una composicion/smoke neutral opt-in.
- DCR-010: T207 rotacion de sesiones queda sin tarea de codigo en runner; el
  contrato opt-in vive en runtime/orchestration y el runner sigue como tick
  acotado sin daemon ni runtime real. Estado: reconciliado el 2026-05-27.
