# Pruebas: orquesta-app-gateway

## Comandos

```bash
go test -count=1 ./modulos/orquesta-app-gateway
git diff --check -- modulos/orquesta-app-gateway
```

## Cobertura local

- `/nueva-app` usa `RESTArrancarDirectorAppClientV0` contra el bridge MCP REST.
- `/director-stats` usa `RESTDirectorStatsClientV0` contra el bridge MCP REST.
- `/api/v0/apps/director` con executor real in-memory comparte RunStore con
  `/director-stats`.
- `/api/v0/director/stats` mantiene el contrato MCP/API completo de
  `DirectorRunStatsV0` y `DirectorDecisionContextV0`; el gateway solo compone
  rutas y transporte in-process.
- El arranque de director se observa como `brainstorms=1` y `agents_started=1`
  con `tasks_total=0`; las tareas quedan reservadas a microtareas de workflow.
- El transporte in-process conserva metodo, path, headers y status.
- Guard de arquitectura contra `cmd`, `db`, drivers, runtime real y proveedores.
