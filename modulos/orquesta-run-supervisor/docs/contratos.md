# Contratos

## Caso de uso

`SuperviseRunsV0(ctx, deps, command)` ejecuta varios ticks globales de forma
determinista y con presupuesto.

Entradas principales:

- `QueueRef`, `AppRefs`, `QueueLimit`, `OccurredAt`, `CorrelationID`,
  `DrainLimits` y `RankingPolicy` se propagan al tick.
- `RankingPolicy` transporta tambien ventanas/limites de fairness por grupo; el
  supervisor no reordena por su cuenta.
- `MaxTicks` limita el numero de ticks. Si no se informa, vale uno.
- `MaxRunsPerTick` limita ejecuciones por tick. Si no se informa, vale uno.
- `MaxExecutions` limita ejecuciones acumuladas. Cero significa sin limite
  acumulado extra.
- `StopOnNoExecution` corta al primer tick sin ejecuciones.
- `AllowRepeatedRuns` permite repetir una misma run dentro de la misma pasada.
  Por defecto se excluyen temporalmente las runs ya ejecutadas.

Puerto:

- `RunSupervisorTickPortV0`.

Salidas:

- `Ticks`: resultados compactos de cada tick.
- `TotalExecutions`.
- `TotalSkips`.
- `StopReason`.
