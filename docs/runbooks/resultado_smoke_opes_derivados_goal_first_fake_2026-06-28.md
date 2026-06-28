# Resultado smoke OPES derivados goal-first fake 2026-06-28

Objetivo: revalidar la ruta de derivados OPES por secuencia completa usando el
modo goal-first de Orquesta, sin tocar OPES productivo ni una cola real.

## Cambios de codigo previos

- `bc392a2d Recupera material OPES no canonico`: el integrador OPES legacy
  estrecho conserva y promueve material valido de rutas no canonicas
  (`02_markdown -> 04_markdown`) mediante
  `consolidate_checkpoint_topic_from_existing_material`.
- `82a3bcf1 Deduplica replan por assessment semantico`: el replan por
  assessment usa `assessment-recursion-guard` estable por run/tarea/delivery y
  tipo de evaluacion, evitando recursion artificial cuando cambia
  `assessment_ref`.

## Validacion ejecutada

Suite completa:

```bash
go test -count=1 ./...
```

Resultado: verde.

Preflight local sin OPES/Codex:

```bash
ORQUESTA_OPES_BASE_URL=http://127.0.0.1:18080 \
ORQUESTA_BASE_URL=http://127.0.0.1:19080 \
ORQUESTA_OPES_DERIVATIVES_REST_CONFIRM=1 \
ORQUESTA_OPES_TEMPORAL_CONFIRM=1 \
ORQUESTA_OPES_DERIVATIVES_EXECUTE=1 \
ORQUESTA_OPES_DERIVATIVES_SMOKE_MODE=preflight-only \
ORQUESTA_OPES_DERIVATIVES_PREFLIGHT_TARGET_MODE=run-until-finalize \
ORQUESTA_OPES_BRIDGE_PROGRAM_ID=program-ref-temporal-preflight \
ORQUESTA_OPES_BRIDGE_SCOPE_FILTER_CONFIRMED=1 \
ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_CAPABILITY=available \
ORQUESTA_CODEX_GOAL_BACKEND=app_server_stdio \
ORQUESTA_OPES_BRIDGE_LIMIT=1 \
scripts/smoke_opes_derivatives_rest.sh
```

Resultado: `preflight_status=ok`, `preflight_target_mode=run-until-finalize`.

Tests offline focales:

```bash
go test -count=1 ./modulos/orquesta-opes-bridge ./cmd/orquesta-server \
  -run 'TestSmokeOPESDerivativesRESTWrapperFakeServer|TestOPESTemarioCycle|TestRunOPESDrainOnceV0|TestOPESBridgeLoop'
```

Resultado: verde.

Smoke fake end-to-end hasta paquete:

```bash
ORQUESTA_OPES_DERIVATIVES_FAKE_SERVER=1 \
ORQUESTA_OPES_DERIVATIVES_SMOKE_MODE=run-until-finalize \
ORQUESTA_OPES_DERIVATIVES_EXECUTE=1 \
ORQUESTA_OPES_TEMPORAL_CONFIRM=1 \
ORQUESTA_OPES_BRIDGE_PROGRAM_ID=program-ref-fake-operario-001 \
ORQUESTA_OPES_BRIDGE_LIMIT=1 \
ORQUESTA_OPES_BRIDGE_MAX_TICKS=30 \
ORQUESTA_OPES_DERIVATIVES_TICK_SLEEP_SECONDS=0 \
SMOKE_ID=derivatives-fake-20260628T021625Z \
scripts/smoke_opes_derivatives_rest.sh
```

Resultado:

- `run_until_status=completed`.
- `run_until_mode=run-until-finalize`.
- `final_job_type=finalize_temario_package`.
- Director execution mode por derivado: `goal_first`.
- 23 ticks, uno por tipo de la secuencia canonica de derivados.
- Salida local:
  `/tmp/opes-salidas/derivatives-rest-derivatives-fake-20260628T021625Z`.
- Resumen final:
  `/tmp/opes-salidas/derivatives-rest-derivatives-fake-20260628T021625Z/opes_derivatives_rest_tick_23_drain_summary.json`.
- Ledger:
  `/tmp/opes-salidas/derivatives-rest-derivatives-fake-20260628T021625Z/external-bridge-input-ledger.json`.

## Estado

Cerrado para fake/offline:

- secuencia completa de derivados hasta `finalize_temario_package`;
- bridge por `JOB_TYPE_SEQUENCE`;
- ruta goal-first;
- observacion/cierre aceptado por goal fake;
- ledger y salidas por directorio de smoke;
- guardas de preflight para temporal, scope, audio y goal-first.

Pendiente real:

- Ejecutar la misma secuencia contra una instancia OPES temporal real con
  `ORQUESTA_OPES_TEMPORAL_CONFIRM=1`, scope duro confirmado y backend Codex
  Goal real.
- Demostrar receipts OPES reales por cada job, dedupe por `external_job_ref`,
  HTML local revisable, audios, tutor, juegos, manuales y paquete final.

No se ha ejecutado contra OPES real en este corte porque no habia una instancia
OPES temporal y scope real confirmados para Orquesta. No debe usarse OPES
productivo ni una cola amplia para completar ese pendiente.
