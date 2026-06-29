# Smoke real autoprogramacion Bolsa

Objetivo: verificar la compatibilidad legacy de Orquesta preparando,
supervisando y cerrando un run real de autoprogramacion sobre una copia
temporal de `Bolsa_Diputacion_app`, sin tocar OPES y sin relajar gates de
cierre. No es la ruta normal goal-first.

## Comando

```bash
ORQUESTA_BOLSA_REAL_SMOKE_CONFIRM=1 \
ORQUESTA_BOLSA_REAL_CODEX_EXECUTION_CONFIRMED=1 \
ORQUESTA_LEGACY_DIRECTOR_LOOP_SMOKE_CONFIRM=1 \
ORQUESTA_AUTOPROGRAMMING_LEGACY_DIRECTOR_LOOP=1 \
ORQUESTA_BOLSA_APP_SOURCE_DIR=/home/alberto/Trabajo/Bolsa_Diputacion_app \
ORQUESTA_BOLSA_APP_ADDR=127.0.0.1:18082 \
scripts/smoke_autoprogramming_bolsa_real.sh
```

## Reproducir el cierre 100 productizable

```bash
ORQUESTA_BOLSA_REAL_SMOKE_CONFIRM=1 \
ORQUESTA_BOLSA_REAL_CODEX_EXECUTION_CONFIRMED=1 \
ORQUESTA_LEGACY_DIRECTOR_LOOP_SMOKE_CONFIRM=1 \
ORQUESTA_AUTOPROGRAMMING_LEGACY_DIRECTOR_LOOP=1 \
ORQUESTA_KEEP_SMOKE_DIR=1 \
ORQUESTA_SMOKE_ROOT=/tmp/orquesta-bolsa-100-productizable-repro \
ORQUESTA_BOLSA_REAL_RUN_REF=bolsa-100-productizable-repro \
ORQUESTA_BOLSA_SPEC_PATH=/home/alberto/Trabajo/orquesta/docs/bolsa_100_productizable_local_orquesta_spec_2026-06-19.json \
ORQUESTA_BOLSA_APP_SOURCE_DIR=/home/alberto/Trabajo/Bolsa_Diputacion_app \
ORQUESTA_BOLSA_APP_ADDR=127.0.0.1:18082 \
ORQUESTA_BOLSA_REAL_SUPERVISE_TICKS=4 \
scripts/smoke_autoprogramming_bolsa_real.sh
```

El script compila `cmd/orquesta-server`, arranca un servidor temporal, llama a
`/api/v0/autoprogramming/prepare-run` en modo legacy con
`ORQUESTA_AUTOPROGRAMMING_LEGACY_DIRECTOR_LOOP=1` y
`director_execution_mode=legacy_director_loop` en el payload, supervisa el
`run_ref` devuelto con `/api/v0/runs/supervise` por compatibilidad legacy tras
`ORQUESTA_LEGACY_DIRECTOR_LOOP_SMOKE_CONFIRM=1`, espera cierre causal, ejecuta
`go test -count=1 ./...` en la app resultante y arranca Bolsa para validar
`/healthz` y `/api/portal`. No cubre la rama `goal_ready`, donde `prepare-run`
debe devolver `goal_spec_summaries[]` sin `run_ref` legacy.

## Variables utiles

- `ORQUESTA_SMOKE_ROOT`: directorio temporal fijo para inspeccionar estado.
- `ORQUESTA_KEEP_SMOKE_DIR=1`: conserva estado, payloads y resultados.
- `ORQUESTA_BOLSA_REAL_RUN_REF`: fuerza un `run_ref` concreto.
- `ORQUESTA_BOLSA_SPEC_PATH`: spec JSON de entrada; por defecto usa
  `orquesta_spec_nucleo.json` dentro de la copia de Bolsa.
- `ORQUESTA_BOLSA_PROJECT_MODE=copy|empty`: copia la app fuente o parte de un
  directorio vacio.

## Gates adicionales

Si existe `web/static/index.html` y `web/static/app.js`, el smoke exige wiring
minimo de UI administrativa: navegacion por modulo, busqueda, filtros, tabs,
acciones de tabla, notificaciones y exportacion. Esto evita entregar una Bolsa
con botones visibles sin comportamiento basico.

## Evidencia esperada

El directorio `results/` contiene `prepare_run.json`, `status.json`,
`supervise.json` y `bolsa_api_portal.json`. En salida estandar deben aparecer:

```text
bolsa_real_run_closed=true
bolsa_ui_contract_ok=true
bolsa_app_health_ok=true
codex_real_executed=true
opes_touched=false
```
