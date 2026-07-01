# Incidencia: lifecycle goal-first disperso entre status, shutdown y readiness

Fecha: 2026-07-02.

ID inventario: `BUG-ORQ-20260702-096`.

Relacionado con: `BUG-ORQ-20260701-088`, `BUG-ORQ-20260701-089` y
`BUG-ORQ-20260701-091`.

## Sintoma

Varias oleadas OPES mostraron el mismo fallo de arquitectura operacional:
Orquesta tenia senales vivas de goal-first en unas superficies, pero otras
seguian publicando una vista inofensiva o stale.

Casos concretos:

- `autoprogramming/status` podia mostrar `queue_health.running_live > 0` y
  acciones `observe_goal`, pero `operator.active_runs` quedaba vacio.
- `server/shutdown` no forzado ya bloqueaba algunos goals activos, pero no
  trataba un `GoalWorkState` persistido como `running` como backend vivo si
  aun no habia timeout activo.
- `orquesta-server start` aceptaba readiness con un snapshot `startup_ready`
  aunque, entre la primera lectura y la estabilizacion, el PID persistido
  hubiese muerto y el HTTP dejase de responder.

## Causa

El lifecycle goal-first estaba repartido entre cola legacy, estado durable del
goal, marcadores, statefile del servidor, endpoint HTTP de readiness y shutdown
sin una proyeccion uniforme. Eso permitia falsos verdes parciales: un
componente veia trabajo vivo y otro no.

## Cierre acotado

Se endurecen tres superficies sin cambiar el contrato del nucleo:

- MCP `autoprogramming/status` proyecta goals visibles en
  `operator.active_runs` aunque la cola legacy este vacia.
- Shutdown del stack Codex considera `GoalWorkState.status=running` como
  `goal_backend/backend_still_running`, aunque todavia no exista evidencia de
  timeout.
- El comando de arranque reconcilia liveness del statefile durante la espera de
  readiness estable; si el proceso muere tras publicar readiness, devuelve
  `server_exited_after_readiness` y persiste `server_process_stale`.

Este cierre no resuelve todo `BUG-ORQ-20260701-088` ni todo
`BUG-ORQ-20260701-091`: siguen pendientes las politicas finas de corte por
alto consumo con solo checkpoint y la limpieza/reconciliacion completa de
backends huerfanos nacidos fuera del owner recuperable.

## Evidencia

Pruebas locales:

```bash
git diff --check
go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-mcp ./modulos/orquesta-app-codex-stack ./modulos/orquesta-opes-director
```

Pruebas focales cubiertas:

- `TestMCPAutoprogrammingStatusExecutorV0EfficiencyNoDeclaraIdleConGoalVivoYColaVacia`
- `TestStackShutdownActiveWorkReaderV0BloqueaBackendRunningSinTimeoutLocal`
- `TestStackShutdownActiveWorkReaderV0ListaGoalFirstCompletoPendienteObservacion`
- `TestWaitForStateHealthyV0MarcaStaleSiServidorMuereTrasReadinessV0`
