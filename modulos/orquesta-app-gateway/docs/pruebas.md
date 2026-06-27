# Pruebas: orquesta-app-gateway

## Comandos

```bash
go test -count=1 ./modulos/orquesta-app-gateway
git diff --check -- modulos/orquesta-app-gateway
```

## Cobertura local

- `/nueva-app` usa `RESTArrancarDirectorAppClientV0` contra el bridge MCP REST.
- `/director-stats` usa `RESTDirectorStatsClientV0` contra el bridge MCP REST.
- `/run-control` usa `RESTRunControlClientV0` contra el bridge MCP REST de
  control; el gateway solo compone cliente in-process.
- `/run-queue` usa `RESTRunQueueClientV0` contra el bridge MCP REST de cola
  multiapp; el gateway solo compone cliente in-process.
- `/api/v0/apps/director` con executor real in-memory comparte RunStore con
  `/director-stats`; las pruebas de cohorte legacy declaran
  `director_execution_mode=legacy_director_loop`.
- `/api/v0/apps/director/goal/observe` delega en
  `ObserveDirectorGoal` inyectado y conserva `run_ref` sin interpretar cierre.
- `/api/v0/apps/intake/guided-turn` devuelve una sesion guiada de nueva app con
  datos/mapas desde una necesidad libre, sin stores ni runtime.
- `/api/v0/apps/vcs` precede al prefijo `/api/v0/apps/` y delega en AppVCS, no
  en el catch-all de app-change.
- `/api/v0/runs/control` y `/api/v0/runs/queue/priority` delegan en executors
  MCP inyectados.
- `/api/v0/runs/supervise` delega en executor MCP inyectado y queda protegido
  por el guard browser de mutaciones para origen cruzado.
- `/api/v0/autoprogramming/supervise` responde `202 accepted_background` si el
  supervisor sigue vivo tras el timeout publico y no duplica una operacion
  activa con el mismo `idempotency_key`.
- `/api/v0/autoprogramming/status` propaga el timeout JSON publico de MCP
  (`504`, `autoprogramming_status_timeout`) cuando la lectura de cola/stats no
  responde en la ventana acotada.
- `/api/v0/autoprogramming/goals/observe-active` delega en el executor MCP
  inyectado para observar goals activos sin pasar por supervision legacy.
- `/api/v0/director/stats` mantiene el contrato MCP/API completo de
  `DirectorRunStatsV0` y `DirectorDecisionContextV0`; el gateway solo compone
  rutas y transporte in-process.
- El arranque de director se observa como `brainstorms=1` y `agents_started=1`
  con `tasks_total=0`; las tareas quedan reservadas a microtareas de workflow.
- El transporte in-process conserva metodo, path, headers y status.
- Guard de arquitectura contra `cmd`, `db`, drivers, runtime real y proveedores.
