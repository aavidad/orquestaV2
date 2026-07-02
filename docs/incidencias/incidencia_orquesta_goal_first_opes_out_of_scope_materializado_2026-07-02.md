# Incidencia: artefactos OPES materializados fuera del write-set goal-first

Fecha: 2026-07-02

## Estado

Avance acotado de `BUG-ORQ-20260701-085`.

## Problema

En una fase OPES estrecha, el agente podia escribir artefactos utiles fuera del
`write_set` declarado y cerrar con un recibo que solo describia la fase
autorizada. Eso dejaba trabajo recuperable sin trazabilidad causal y podia
ocultar coste, QA pendiente o una violacion real de alcance.

## Cambio

- El scanner de materialized refs detecta contexto OPES y deriva la raiz del
  tema desde write-sets estrechos como `temas/tema_003/coordinacion_*`.
- Escanea solo esa raiz de tema, con limites bajos, y salta directorios ya
  autorizados por el `write_set`.
- Si encuentra artefactos OPES fuera del `write_set`, publica
  `out_of_scope_materialized_artifacts` con refs opacas.
- `director.stats`, `autoprogramming/status` y `observe_goal` proyectan el
  bloqueo con accion `rework_write_set_violation`.

## Limites

No sustituye al enforcement fuerte de runtime/sandbox ni a un snapshot
pre/post de filesystem. Es una reconciliacion conservadora para OPES: preserva
artefactos recuperables, impide cierre silencioso y pide rework de alcance.

## Pruebas

- `TestStackGoalMaterializedRefsSourceV0DetectaArtefactosOPESFueraDeWriteSet`
- `TestEnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0OutOfScopePideRework`
- `TestMCPDirectorStatsToolExecutorV0GoalFirstProyectaOutOfScopeMaterializedV0`
- `TestMCPAutoprogrammingStatusExecutorV0OutOfScopeMaterializedPideReworkV0`
- `go test -count=1 ./modulos/orquesta-app-codex-stack ./modulos/orquesta-mcp`
