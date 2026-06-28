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
ORQUESTA_OPES_BRIDGE_DEDICATED_TEMPORAL_QUEUE=1 \
ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_CAPABILITY=available \
ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_EVIDENCE_REFS=evidence-ref-tts-temporal-preflight \
ORQUESTA_CODEX_GOAL_BACKEND=app_server_stdio \
ORQUESTA_OPES_BRIDGE_LIMIT=1 \
scripts/smoke_opes_derivatives_rest.sh
```

Resultado: `preflight_status=ok`, `preflight_target_mode=run-until-finalize`.
Revalidado despues de endurecer guardas de scope/TTS con
`SMOKE_ID=preflight-current-guard-check`; salida:
`preflight_scope=dedicated_temporal_queue,program_id` y
`preflight_output_dir=/tmp/opes-salidas/derivatives-rest-preflight-current-guard-check`.

Tests offline focales:

```bash
go test -count=1 ./modulos/orquesta-opes-bridge ./cmd/orquesta-server \
  -run 'TestSmokeOPESDerivativesRESTWrapperFakeServer|TestOPESTemarioCycle|TestRunOPESDrainOnceV0|TestOPESBridgeLoop'
```

Resultado: verde.

Scope probe fake, sin efectos:

```bash
ORQUESTA_OPES_DERIVATIVES_FAKE_SERVER=1 \
ORQUESTA_OPES_DERIVATIVES_SMOKE_MODE=scope-probe \
ORQUESTA_OPES_BRIDGE_PROGRAM_ID=program-ref-fake-operario-001 \
ORQUESTA_OPES_BRIDGE_LIMIT=1 \
SMOKE_ID=manual-scope-probe-test \
SMOKE_OUT_DIR=/tmp/opes-salidas/manual-scope-probe-test \
scripts/smoke_opes_derivatives_rest.sh
```

Resultado:

- `scope_probe_status=ok`.
- `scope_probe_job_type=update_topic_registry`.
- `scope_probe_negative_checks=program_id`.
- Salida local:
  `/tmp/opes-salidas/manual-scope-probe-test/opes_derivatives_scope_probe.json`.

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
- `empty_after_final=true`.
- `goal_receipts_manifest_status=ok`.
- `goal_receipts_manifest_entries=23`.
- Director execution mode por derivado: `goal_first`.
- 24 ticks: 23 tipos de la secuencia canonica de derivados mas un tick final
  vacio que demuestra que no quedan pendientes en la secuencia.
- Salida local:
  `/tmp/opes-salidas/manual-run-until-manifest-test`.
- Resumen final:
  `/tmp/opes-salidas/manual-run-until-manifest-test/opes_derivatives_rest_tick_24_drain_summary.json`.
- Manifest goal/receipts:
  `/tmp/opes-salidas/manual-run-until-manifest-test/goal_receipts_manifest.json`.
- Ledger:
  `/tmp/opes-salidas/manual-run-until-manifest-test/external-bridge-input-ledger.json`.

Revalidacion posterior con el estado actual del repo:

```bash
ORQUESTA_OPES_DERIVATIVES_FAKE_SERVER=1 \
ORQUESTA_OPES_DERIVATIVES_SMOKE_MODE=run-until-finalize \
ORQUESTA_OPES_DERIVATIVES_EXECUTE=1 \
ORQUESTA_OPES_TEMPORAL_CONFIRM=1 \
ORQUESTA_OPES_DERIVATIVES_REST_CONFIRM=1 \
ORQUESTA_OPES_BRIDGE_PROGRAM_ID=program-ref-fake-operario-001 \
ORQUESTA_OPES_BRIDGE_LIMIT=1 \
ORQUESTA_OPES_BRIDGE_MAX_TICKS=30 \
ORQUESTA_OPES_DERIVATIVES_TICK_SLEEP_SECONDS=0 \
scripts/smoke_opes_derivatives_rest.sh
```

Resultado 2026-06-28T02:56:33Z:

- `run_until_status=completed`.
- `final_job_type=finalize_temario_package`.
- `empty_after_final=true`.
- `goal_receipts_manifest_status=ok`.
- `goal_receipts_manifest_entries=23`.
- Salida local:
  `/tmp/opes-salidas/derivatives-rest-20260628T025633Z`.
- Resumen final:
  `/tmp/opes-salidas/derivatives-rest-20260628T025633Z/opes_derivatives_rest_tick_24_drain_summary.json`.
- Manifest goal/receipts:
  `/tmp/opes-salidas/derivatives-rest-20260628T025633Z/goal_receipts_manifest.json`.

Revalidacion posterior tras cierre goal-first y limpieza de tests:

```bash
ORQUESTA_OPES_DERIVATIVES_FAKE_SERVER=1 \
ORQUESTA_OPES_DERIVATIVES_SMOKE_MODE=run-until-finalize \
ORQUESTA_OPES_DERIVATIVES_EXECUTE=1 \
ORQUESTA_OPES_TEMPORAL_CONFIRM=1 \
ORQUESTA_OPES_DERIVATIVES_REST_CONFIRM=1 \
ORQUESTA_OPES_BRIDGE_PROGRAM_ID=program-ref-fake-operario-001 \
ORQUESTA_OPES_BRIDGE_LIMIT=1 \
ORQUESTA_OPES_BRIDGE_MAX_TICKS=30 \
ORQUESTA_OPES_DERIVATIVES_TICK_SLEEP_SECONDS=0 \
SMOKE_ID=derivatives-fake-current-20260628T141818Z \
scripts/smoke_opes_derivatives_rest.sh
```

Resultado 2026-06-28T14:18:22Z:

- `run_until_status=completed`.
- `run_until_mode=run-until-finalize`.
- `final_job_type=finalize_temario_package`.
- `empty_after_final=true`.
- `goal_receipts_manifest_status=ok`.
- `goal_receipts_manifest_entries=23`.
- Manifest verificado con 23 entradas, primer `work_kind=update_topic_registry`,
  ultimo `work_kind=finalize_temario_package`, sin `missing` ni `unexpected`.
- El tick final quedo `status=completed`, `seen=0`, `submitted=0` y con los 23
  tipos canonicos en `empty_job_types`.
- Salida local:
  `/tmp/opes-salidas/derivatives-rest-derivatives-fake-current-20260628T141818Z`.
- Resumen final:
  `/tmp/opes-salidas/derivatives-rest-derivatives-fake-current-20260628T141818Z/opes_derivatives_rest_tick_24_drain_summary.json`.
- Manifest goal/receipts:
  `/tmp/opes-salidas/derivatives-rest-derivatives-fake-current-20260628T141818Z/goal_receipts_manifest.json`.

## Estado

Cerrado para fake/offline:

- secuencia completa de derivados hasta `finalize_temario_package`;
- bridge por `JOB_TYPE_SEQUENCE`;
- ruta goal-first;
- observacion/cierre aceptado por goal fake;
- manifest `goal_receipts_manifest.json` con artefactos y receipts de dominio
  por cada run goal-first;
- tick final vacio despues de `finalize_temario_package`;
- ledger y salidas por directorio de smoke;
- `scope-probe` para confirmar filtros OPES antes de declarar
  `ORQUESTA_OPES_BRIDGE_SCOPE_FILTER_CONFIRMED=1`;
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
