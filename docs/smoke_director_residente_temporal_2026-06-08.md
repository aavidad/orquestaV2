# Smoke Director Residente Temporal 2026-06-08

## Objetivo

Verificar el alojamiento residente opt-in del Director en `orquesta-server`
sin tocar OPES productivo ni ejecutar Codex real.

## Comando

```bash
SMOKE_ROOT="$(mktemp -d /tmp/orquesta-resident-director.XXXXXX)"
ORQUESTA_SMOKE_ROOT="$SMOKE_ROOT" \
ORQUESTA_KEEP_SMOKE_DIR=1 \
ORQUESTA_AUTOPROGRAMMING_SUPERVISED_SMOKE_CONFIRM=1 \
ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED=true \
ORQUESTA_SERVER_RESIDENT_DIRECTOR_MAX_ACTIONS=8 \
ORQUESTA_SERVER_TICK_INTERVAL_MS=250 \
ORQUESTA_SERVER_MAX_RUNS_PER_TICK=1 \
ORQUESTA_SERVER_MAX_EXECUTIONS_PER_TICK=1 \
ORQUESTA_SERVER_DRAIN_MAX_BURSTS=1 \
ORQUESTA_SERVER_DRAIN_MAX_STEPS=2 \
ORQUESTA_SERVER_DRAIN_MAX_DISPATCHES=1 \
ORQUESTA_SERVER_DRAIN_MAX_COMMANDS=4 \
ORQUESTA_SERVER_DRAIN_MAX_OUTBOX=4 \
ORQUESTA_SERVER_DRAIN_MAX_EXTERNAL_WAITS=1 \
ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS=0 \
ORQUESTA_STARTUP_CLEANUP_MODE=off \
ORQUESTA_OPES_BASE_URL= \
OPES_BASE_URL= \
./scripts/smoke_autoprogramming_supervised.sh
```

## Resultado

Ejecucion local: `/tmp/orquesta-resident-director.g6pERF`.

- `codex_real_executed=false`.
- `opes_touched=false`.
- `resident_director_status=ok`.
- `resident_director_ticks=19`.
- `resident_director_executed_actions=20`.
- `resident_director_last_result=needs_director`.
- Auditoria: 19 eventos `resident_director_tick_start` y 19 eventos
  `resident_director_tick_result`.
- `go test -count=1 ./...` paso despues del smoke.
- No quedo ningun `orquesta-server run` vivo al terminar.

## Alcance

Este smoke prueba servidor temporal real, estado file, cola, residente opt-in,
stack Codex con `codex-fake`, reentrada y cierre delegado al
`app-director-service`. No prueba proveedor Codex real, OPES temporal real ni
consejo/votacion live.

