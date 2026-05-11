# Tareas: orquesta-director-tick-input

## Estado

- DTI-000: contexto local del microproyecto. Estado: cerrado.
- DTI-001: contrato `BuildDirectorSchedulerTickInputV0`. Estado: cerrado.
- DTI-002: builder de snapshot desde `OrchestrationRunV0`. Estado: cerrado.
- DTI-003: normalizacion de proyecciones workflow -> scheduler. Estado: cerrado.
- DTI-004: prueba integrada con scheduler y runner. Estado: cerrado.
- DTI-007: exponer `ReviewGateCandidates` y refs durables de revision/rework en
  el snapshot compacto. Estado: cerrado.

## Siguientes Microtareas

- DTI-005: puerto externo para obtener refs de outbox pendiente.
- DTI-006: ensamblador superior que combine event-store, outbox ledger y candidates.
