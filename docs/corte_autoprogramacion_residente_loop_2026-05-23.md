# Corte autoprogramacion residente - 2026-05-23

Este corte acota el contrato del supervisor residente de autoprogramacion en el
stack Codex.

## Contrato

- `resident_mode=true` y `run_ref` vacio operan sobre la cola durable del stack.
- Si una pasada global ejecuta una run y termina en estado `done`, el ciclo no
  se da por completo hasta observar una pasada posterior sin ejecuciones.
- El cierre real de una app sigue saliendo del loop normal:
  director, tasks, agentes, review, tests requeridos, replan/cierre y cola.
- Si una run queda bloqueada por un fallo reparable, el residente puede preparar
  una run de self-repair con el mismo write-set, tests requeridos y refs opacas
  de worktree/branch.
- Las tareas de autoprogramacion se marcan con
  `operational_director.task_source:autoprogramming` como `context_ref` opaca.
  Si la composicion inyecta `OperationalPlanStateStore/Writer`, el bridge
  siembra un `OperationalDirectorPlanState` reentrable y devuelve
  `operational_director_plan_ref` en el `Continue`. Ese campo se expone por
  MCP/web como opcional: una composicion legacy sin plan-store no debe fallar por
  no devolverlo, solo queda limitada para cierre autonomo completo.
- Una run de autoprogramacion entregada no sale de cola como `delivered` si
  quedan tareas abiertas pendientes de revision/tests/cierre; la decision source
  de composicion puede abrir `revision` cuando todas las entregas estan listas.
- El cierre operativo del stack reconoce tareas marcadas en `context_refs`,
  ademas de criterios/contratos/refs de linaje. Esto evita exigir que todos los
  adaptadores coloquen la misma senal en el mismo campo textual.

## Limites

- No hay bucle infinito: `max_ticks` sigue siendo el presupuesto de una llamada.
- No se introduce runtime, proveedor, DB, HTTP ni filesystem en el nucleo.
- `run_ref` explicito conserva la semantica puntual: empuja solo esa run.
- Una cola sin ejecuciones sigue cerrando la llamada con `done`.
- Si una composicion no inyecta store/writer de plan operativo, el bridge no
  puede sembrar plan-state y el cierre completo queda limitado a la ruta legacy.

## Evidencia focal

- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestAutoprogrammingResidentModeV0'`
- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'Test(AutoprogrammingDirectorDecisionSourceV0|StackDrainQueueStatus|PrepareAutoprogrammingRunV0Persiste|AutoprogrammingResidentModeV0)'`
- `go test -count=1 ./modulos/orquesta-app-codex-stack -run '^(TestOperationalClosureTaskClassifierV0AceptaMarkerEnContextRefsV0|TestCodexStackAutoprogrammingPrepareRunAPIV0CierraConPlanStateYTestsRealesV0)$'`
- `./scripts/test_autoprogramming_fast.sh`
- `./scripts/test_orquesta_fast_parallel.sh`
- `go test -count=1 ./modulos/orquesta-app-director-service -run 'TestEnsureContinueOperationalDirectorPlanStateFromWorkflowTasksV0AceptaMarkerEnContextRefsV0|TestStartAppDirectorV0DecisionCreateMicrotaskRequiredTestsCreaPlanStateReentrable'`
- `go test -count=1 ./modulos/orquesta-app-codex-stack`
- `go test -count=1 ./modulos/orquesta-app-director-service`
- `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-web`
- `ORQUESTA_AUTOPROGRAMMING_SUPERVISED_SMOKE_CONFIRM=1 SMOKE_ID=post-planstate-20260523 ORQUESTA_KEEP_SMOKE_DIR=0 ./scripts/smoke_autoprogramming_supervised.sh`
  paso con servidor temporal, Codex fake, `resident_supervisor_agents_started=1`,
  replay idempotente OK, `external-work/run` OK, `codex_real_executed=false` y
  `opes_touched=false`.
