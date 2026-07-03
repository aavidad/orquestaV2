# Tarea Orquesta - Grupo B Informática: runtime caído con estado `running`

Fecha: 2026-07-01

## Contexto

Durante el rework OPES de `Grupo B Informática`, se arrancó Orquesta en un
`state_dir` aislado:

`/home/alberto/Trabajo/orquesta/.orquesta-runtime/grupo-b-informatica-rework-20260701/state`

El primer arranque sin backend Goal quedó en readiness degradado por falta de
`ORQUESTA_CODEX_GOAL_BACKEND`.

El segundo arranque se hizo con:

```bash
ORQUESTA_SERVER_ADDR=127.0.0.1:8787
ORQUESTA_SERVER_STATE_DIR=/home/alberto/Trabajo/orquesta/.orquesta-runtime/grupo-b-informatica-rework-20260701/state
ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux
ORQUESTA_SERVER_AUTONOMY_ENABLED=true
go run ./cmd/orquesta-server run
```

`/api/v0/server/readiness` devolvió `ready=true`.

## Fallo observado

Al intentar llamar a:

```bash
POST http://127.0.0.1:8787/api/v0/external-work/dry-run
```

el puerto ya no respondía.

El proceso indicado en `server_goal_backend.pid` no existía, pero
`orquesta_server_state_v0.json` seguía publicando:

```json
{
  "status": "running",
  "startup_status": "startup_ready",
  "last_supervisor_status": "ok",
  "resident_director_status": "ok",
  "goal_observer_status": "ok"
}
```

El log `server_goal_backend.log` estaba vacío.

## Impacto

OPES no puede saber de forma fiable si Orquesta sigue vivo, se cayó o mantiene
estado stale si el proceso desaparece sin actualizar el estado persistido.

## Acción esperada

Revisar la reconciliación de proceso vivo frente a estado persistido:

1. `server/status` o `readiness` no deben quedar como única fuente si el proceso ya no responde.
2. El estado persistido no debe decir `running` indefinidamente si el proceso murió.
3. El arranque en `go run ./cmd/orquesta-server run` debería dejar causa pública de salida o al menos log operacional.
4. El operador necesita una forma estable de saber: vivo, caído, en cola, bloqueado o stale.

No se ha tocado código de Orquesta. Esta tarea queda para el agente de Orquesta.
