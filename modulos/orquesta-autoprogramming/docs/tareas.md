# Tareas: orquesta-autoprogramming

## Estado actual

- Extraido desde `orquesta-orchestration-core` para separar reglas de
  programacion del nucleo generico.
- MCP usa este modulo como preflight.
- `orquesta-runtime-codex-delivery` usa este modulo para evaluar ACKs de codigo.
- La automejora secundaria ya queda modelada como contrato puro:
  `BuildAutoprogrammingSelfImprovementRequestV0` produce una request de baja
  prioridad para que el adaptador la prepare/encole despues.
- `BuildAutoprogrammingProgrammableWorkV0` compacta textos largos antes de
  `WorkflowTaskV0` y devuelve issues con subcampo causal para reparacion.
- Los limites vigentes de ola admiten 10 padres por defecto y hasta 6
  subagentes por padre en el trabajo programable; limites menores pueden
  declararse explicitamente por request cuando una composicion quiera acotar la
  ola.
- `BuildAutoprogrammingProgrammableWorkV0` emite `goal_migration` para separar
  tareas que siguen en loop legacy de tareas candidatas/cubiertas por Goal,
  usando solo refs opacas en `context_refs`.
- Si `goal_migration=goal_ready`, el trabajo programable incluye
  `goal_specs[]` (`GoalWorkSpecV0`) por grupo, sin lanzar runtime ni depender de
  Codex real.

## Cerrado

- APG-001: contrato `v0` cerrado: OPES o apps grandes pueden declarar
  `max_task_refs`, `max_areas` y `max_write_set_entries` hasta 40 sin
  microfragmentar contenido; texto libre no amplia limites.
- APG-002: `AutoprogrammingRequestV1` permite perfiles de trabajo por tipo de
  app, area o tarea y `BuildAutoprogrammingProgrammableWorkV1` materializa esos
  perfiles sobre `WorkProfileV0`/`WorkflowTaskV0`; `v0` conserva su fallback de
  implementacion.
- APG-003: contrato puro `AutoprogrammingRequestSourceV0` para evidencia de
  entrada real web/MCP/CLI sin importar adaptadores. Las fixtures en
  `docs/fixtures/autoprogramming_request_v0/` son ejemplos versionados y pruebas
  de compatibilidad, no el mecanismo de cierre por si solas.
- APG-004: reconciliacion documental T208, 2026-05-27. El guardian break-glass
  queda tratado como umbrella historico ya cubierto por owners especificos
  T212-T237; este modulo no absorbe runtime, VCS, Codex, servidor ni guardian.
  Si aparece regresion, abrir tarea focal con write-set propio.
- APG-005: clasificacion y specs goal-first, 2026-06-25. El modulo distingue
  `legacy_loop_compatible`, `goal_ready`, `blocked_by_goal_capability`,
  `covered_by_goal_first` y `legacy_loop_required`; cuando hay `goal_ready`,
  compila `GoalWorkSpecV0` neutral con write-set/pruebas/criterios por grupo,
  sin lanzar Codex ni tocar servidor. La composicion sigue siendo responsable de
  crear/observar goals y validar smokes reales.
