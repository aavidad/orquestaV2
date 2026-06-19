# Cierre 100 productizable Bolsa por Orquesta - 2026-06-19

## Estado

Run Orquesta cerrado:

```text
run_ref: bolsa-100-productizable-20260619-002
estado: cerrada
closure: closed
progreso: 100%
tareas: 6/6 cerradas
agentes: 6 pedidos, 6 arrancados, 6 entregados, 0 fallidos, 0 en vuelo
reviews: 6 aceptadas
validaciones: 1
blockers: 0
```

La tarea padre `task-autoprogramming-30149268b7e8-g03` aparece en
`closed_tasks`, no queda abierta y no figura bloqueo por
`required_test_evidence_refs`.

## Evidencia estable

```text
spec: /home/alberto/Trabajo/orquesta/docs/bolsa_100_productizable_local_orquesta_spec_2026-06-19.json
estado: /tmp/orquesta-bolsa-100-productizable-20260619-002/results/status.json
prepare: /tmp/orquesta-bolsa-100-productizable-20260619-002/results/prepare_run.json
supervise: /tmp/orquesta-bolsa-100-productizable-20260619-002/results/supervise.json
portal: /tmp/orquesta-bolsa-100-productizable-20260619-002/results/bolsa_api_portal.json
state_dir: /tmp/orquesta-bolsa-100-productizable-20260619-002/state/orchestration-state
closure_ref: closure-ref-operational-director-bolsa-100-productizable-20260619-002
validation_ref: validation-ref-operational-director-task-autoprogramming-30149268b7e8-g06
```

No se observan bloqueos `operational-closure-issues`,
`required_test_evidence_refs` ni `nucleo_orquestacion_invalido` en el estado
final.

## Comando Reproducible

```bash
ORQUESTA_BOLSA_REAL_SMOKE_CONFIRM=1 \
ORQUESTA_BOLSA_REAL_CODEX_EXECUTION_CONFIRMED=1 \
ORQUESTA_KEEP_SMOKE_DIR=1 \
ORQUESTA_SMOKE_ROOT=/tmp/orquesta-bolsa-100-productizable-repro \
ORQUESTA_BOLSA_REAL_RUN_REF=bolsa-100-productizable-repro \
ORQUESTA_BOLSA_SPEC_PATH=/home/alberto/Trabajo/orquesta/docs/bolsa_100_productizable_local_orquesta_spec_2026-06-19.json \
ORQUESTA_BOLSA_APP_SOURCE_DIR=/home/alberto/Trabajo/Bolsa_Diputacion_app \
ORQUESTA_BOLSA_APP_ADDR=127.0.0.1:18082 \
ORQUESTA_BOLSA_REAL_SUPERVISE_TICKS=4 \
scripts/smoke_autoprogramming_bolsa_real.sh
```

## Cierre Directo Posterior

Despues del run Orquesta se aplicaron correcciones directas acotadas sobre la
app final por hallazgos de revision:

- titularidad estricta `candidate_id == X-VEC-Subject` en rutas de candidato;
- rechazo de acuses de notificacion con receptor distinto al candidato;
- snapshot durable tambien para convocatorias/solicitudes;
- backup `.bak` y recuperacion desde ultimo JSON valido;
- manifest VEC con roles reales y rutas con `candidate_id` cuando es obligatorio;
- Docker Compose local en `18081`, bind de `var/bolsa` y exclusion de datos del
  build context;
- smoke local reforzado con tests Go, `node --check`, i18n JSON, flujo HTTP y
  reinicio.

Estas correcciones no relajan gates de Orquesta ni cambian el cierre causal del
run; documentan trabajo posterior de hardening para dejar la app usable al 100%
en local productizable.

