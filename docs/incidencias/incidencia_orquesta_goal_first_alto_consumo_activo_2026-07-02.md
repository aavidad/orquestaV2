# Incidencia: goal-first debe exponer alto consumo activo

Fecha: 2026-07-02

## Sintoma

En ejecuciones OPES goal-first, un goal podia seguir `active` con mas de
100.000 tokens consumidos y solo checkpoint o artefactos insuficientes. Las
superficies operativas no siempre diferenciaban ese estado de un `running`
normal.

## Riesgo

El operador puede seguir esperando o ejecutar shutdown sin una causa publica
compacta que indique consumo alto. Eso degrada cuota, observabilidad y decision
de replan/corte.

## Avance aplicado

- El backend Codex app-server proyecta alto consumo cuando `thread/goal/get`
  devuelve un goal `running` con `tokensUsed >= 100000`.
- La observacion conserva `status=running`; no bloquea ni descarta el trabajo.
- El `summary` incluye `tokens_used`, `time_used_seconds` y `token_budget` si
  estan disponibles.
- Se anade evidencia
  `evidence-ref-codex-app-server-goal-high-token-usage`.
- `autoprogramming/status` ya distingue alto consumo sin checkpoint como
  `goal_active_no_checkpoint_high_consumption` y alto consumo con solo
  checkpoint como `checkpoint_only_high_consumption`, ambos con accion
  `replan_narrow_context`.
- `runs/control stop forced=true` reconcilia el estado Goal durable a
  `blocked` replanificable si el backend activo de alto consumo deja de estar
  vivo tras el control, tanto sin checkpoint como con solo checkpoint.
- `runs.supervisor` en `resident_mode` detecta esos estados terminales
  replanificables, lanza un follow-up por `GoalReworkLauncher`, devuelve
  `repair_run_refs` y guarda una marca en el goal fuente para que la accion sea
  idempotente.

## Evidencia

Tests:

```bash
go test -count=1 ./cmd/orquesta-server -run 'TestServerCodexAppServerGoalBackendV0(ExponeAltoConsumoActivo|BloqueaGoalActivoPorTimeout|ObservaUsageLimitedConCausaOperable)'
go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPAutoprogrammingStatusExecutorV0(CheckpointOnlyHighConsumptionEsBloqueante|SinCheckpointHighConsumptionEsBloqueante)|TestMCPRunControlExecutorV0StopForcedReconcilesGoalHighConsumption(CheckpointOnly|SinCheckpoint)'
go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestRunSupervisorGoalFirst(ResidentPreparaReworkPorCheckpointHighConsumption|ResidentReworkEsIdempotente|NoResidentNoLanzaRework)V0'
git diff --check
```

## Residual

Sigue pendiente la politica temprana de checkpoint por tiempo antes de llegar a
alto consumo y la reconciliacion automatica tras cortes externos/manuales que no
pasen por `runs/control` ni por el cleanup del backend propio.
