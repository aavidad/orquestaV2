# Tareas: orquesta-director-runner

## Estado

- DCR-000: contexto local del microproyecto. Estado: cerrado.
- DCR-001: contrato `RunDirectorCycleV0`. Estado: cerrado.
- DCR-002: puertos `DirectorSchedulerPortV0` y `WorkflowCommandPortV0`. Estado: cerrado.
- DCR-003: adaptador directo de scheduler sin efectos externos. Estado: cerrado.
- DCR-004: pruebas unitarias de espera, outbox y error workflow. Estado: cerrado.
- DCR-005: prueba integrada con scheduler real y workflow en memoria. Estado: cerrado.
- DCR-006: `max_outbox` opt-in para acumular varios outbox y habilitar batch externo. Estado: cerrado.

## Siguientes Microtareas

- DCR-007: conector superior que prepare candidates/snapshot desde contratos compactos.
- DCR-008: persistencia/event-store como puerto externo, sin elegir DB en nucleo.
- DCR-009: ciclo progresivo multi-fase con orquestacion real de app de prueba.
