# Resultado smoke OPES derivados goal-first real 2026-06-28

Fecha: 2026-06-28.

## Resumen

Se ejecuto un smoke real acotado contra OPES temporal local y Orquesta temporal
con `ORQUESTA_CODEX_GOAL_BACKEND=app_server_stdio`, limitado a
`update_topic_registry`.

Resultado: unitario cerrado para `update_topic_registry`. Orquesta supera el
guard de `project_work_dir`, acepta `POST /api/v0/external-work/run`, crea run
goal-first, arranca un Goal Codex real observable, recupera una entrega tardia
tras `codex_app_server_goal_active_timeout`, valida artefacto/receipt y cierra
el run sin reabrir el loop legacy.

## Configuracion valida

Para OPES temporal via REST, el servidor Orquesta temporal debe arrancar con
`ORQUESTA_CODEX_PROJECT_WORKDIR` y `ORQUESTA_OPES_PROJECT_WORKDIR` apuntando al
mismo workspace temporal. Si no, el guard OPES rechaza el submit con
`external_work_project_work_dir_mismatch` porque por defecto exige
`/home/alberto/Trabajo/OPES`.

Ejemplo usado:

```bash
ORQUESTA_SERVER_ADDR=127.0.0.1:0
ORQUESTA_CODEX_PROJECT_WORKDIR=/tmp/orquesta-effectful-slim-20260628T083731Z/project
ORQUESTA_OPES_PROJECT_WORKDIR=/tmp/orquesta-effectful-slim-20260628T083731Z/project
ORQUESTA_CODEX_RUNTIME_WORKDIR=/tmp/orquesta-effectful-slim-20260628T083731Z/runtime
ORQUESTA_SERVER_STATE_DIR=/tmp/orquesta-effectful-slim-20260628T083731Z/state
ORQUESTA_CODEX_GOAL_BACKEND=app_server_stdio
ORQUESTA_OPES_BASE_URL=http://127.0.0.1:18191
ORQUESTA_OPES_TEMPORAL_CONFIRM=1
```

## Evidencia

Primer intento:

- OPES temporal: `http://127.0.0.1:18191`.
- Orquesta temporal: `http://127.0.0.1:38607`.
- Probe: `effectful-single-20260628T082203Z`.
- Scope:
  `/tmp/opes-salidas/derivatives-rest-effectful-single-scope-20260628T082216Z/opes_derivatives_scope_probe.json`.
- Run:
  `/tmp/opes-salidas/derivatives-rest-effectful-single-run2-20260628T083138Z`.
- Resultado: submit correcto, `route_policy=goal_first`,
  `director_execution_mode=goal_first`, `external_goal_ref=019f0d5b-0e9b-76e0-9172-56736677c9ad`,
  `goal_status=running` hasta 12 ticks.

Segundo intento con contexto compactado:

- OPES temporal: `http://127.0.0.1:18191`.
- Orquesta temporal: `http://127.0.0.1:33009`.
- Probe: `effectful-slim-20260628T083713Z`.
- Scope:
  `/tmp/opes-salidas/derivatives-rest-20260628T083723Z/opes_derivatives_scope_probe.json`.
- Run:
  `/tmp/opes-salidas/derivatives-rest-effectful-slim-run-20260628T083831Z`.
- Resultado: submit correcto, `route_policy=goal_first`,
  `director_execution_mode=goal_first`, `external_goal_ref=019f0d61-58a1-7ad0-b634-27d67e84cb10`,
  `goal_status=running` hasta 30 ticks.
- Spec observado: `spec_bytes=24154`, `context_refs=61`,
  `input_values=15`, `payloads=15`. Las politicas OPES masivas quedaron como
  `payload-ref`, no inlineadas.

Tercer intento con recuperacion tardia de goal:

- Root temporal: `/tmp/orquesta-opes-goal-real-20260628T090415Z`.
- OPES temporal: `http://127.0.0.1:55323`.
- Orquesta temporal inicial: `http://127.0.0.1:34773`; reobservacion con binario
  parcheado sobre el mismo state en `http://127.0.0.1:40385`.
- Run:
  `run-external-work-opes-025eda715c48af87913b0f92a2f26fb3-opes-job-025eda715c48af87913b0f92a2f26fb3`.
- Goal externo: `019f0d7a-1dea-7b00-9ad3-d73a0d2f5579`.
- Tick 20 inicial: `goal_status=blocked`,
  `summary=codex_app_server_goal_active_timeout`, `run_status=bloqueada`.
- Artefactos tardios recuperados:
  `external/opes/update_topic_registry/025eda715c48af87913b0f92a2f26fb3/topic_registry_update.json`
  y
  `external/opes/update_topic_registry/025eda715c48af87913b0f92a2f26fb3/docs/orquesta_goal_result_v0.json`.
- Reobservacion parcheada: `estado=ok`, `goal_status=complete`,
  `closure_status=accepted`, `closure_accepted=true`, `run_status=cerrada`,
  `artifact_refs=1`, `domain_receipt_refs=1`.
- OPES temporal: `generation_jobs.status=completed` para 1 job y
  `job_artifacts_count=1`.

Cuarto intento con required tests de dominio reconciliadas por Orquesta:

- Root OPES temporal:
  `/tmp/orquesta-opes-contract-real-20260628T093857Z`.
- OPES temporal: `http://127.0.0.1:46733`.
- Root Orquesta temporal:
  `/tmp/orquesta-opes-goal-full-20260628T093857Z`.
- Orquesta temporal parcheada: `http://127.0.0.1:45103`.
- Probe de contrato REST real:
  `/tmp/orquesta-opes-contract-real-20260628T093857Z/probe/opes_derivatives_rest_contract_probe.json`;
  `contract_probe_status=ok`, `accepted_count=23`, `rejected_count=0`,
  `transport_compat=true`.
- Scope-probe real:
  `/tmp/orquesta-opes-contract-real-20260628T093857Z/scope/opes_derivatives_scope_probe.json`;
  `scope_probe_status=ok`, `scope_probe_job_type=update_topic_registry`,
  `scope_probe_seen=1` y negative checks por `program_id,correlation_id`.
- Reobservacion:
  `/tmp/orquesta-opes-goal-full-20260628T093857Z/manual_reobserve_after_durable_blocked_fix.json`.
- Run:
  `run-external-work-opes-ea21c0901c8440626d075d26c50ddfea-opes-job-ea21c0901c8440626d075d26c50ddfea`.
- Resultado: `run_status=cerrada`, `goal_status=complete`,
  `closure_status=accepted`, `closure_accepted=true`,
  `evidence-ref-goal-domain-receipt-ledger-accepted` presente.
- OPES temporal: `generation_jobs.status=completed` para el job
  `ea21c0901c8440626d075d26c50ddfea` y `job_artifacts_count=1`.

## Cambios derivados

- `BuildExternalWorkGoalWorkSpecV0` conserva campos masivos como
  `input_field_payload` y no los inlinea en `context_refs[input_field_value]`.
- El criterio de aceptacion ya no bloquea por cualquier campo omitido por
  presupuesto; solo exige rework si falta un input imprescindible para el
  artefacto.
- `BuildCodexGoalPromptV0` instruye a materializar artefactos DomainWork bajo
  el write-set autorizado con nombres detectables por Orquesta.
- El observer `app_server_stdio`/`app_server_proxy` marca como
  `blocked` con `codex_app_server_goal_active_timeout` si el goal remoto sigue
  `active` y `timeUsedSeconds` supera `ORQUESTA_CODEX_GOAL_TIMEOUT_MS`.
- El wrapper `run-until-finalize` ya corta en el primer `goal/observe` que
  devuelva `goal_status=blocked|invalid`, `closure_status=blocked|rejected`,
  `run_status=blocked` o `closure_needs_rework=true`; emite
  `run_until_status=blocked`, `run_ref`, `tick`, `observe_goal_response` y
  `stop_reason` en vez de esperar a `MAX_TICKS`.
- Si el bloqueo es exactamente `codex_app_server_goal_active_timeout`, el
  wrapper hace una recuperacion acotada configurable con
  `ORQUESTA_OPES_DERIVATIVES_GOAL_TIMEOUT_RECOVERY_ATTEMPTS` y
  `ORQUESTA_OPES_DERIVATIVES_GOAL_TIMEOUT_RECOVERY_SLEEP_SECONDS` antes de
  declarar fallo, porque Codex Goal puede escribir `orquesta_goal_result_v0.json`
  despues del primer corte activo.
- `ObserveAppDirectorGoalV0` es idempotente cuando un run ya tiene el blocker
  causal del goal: una reobservacion tardia puede incorporar artefactos y
  receipts sin volver a emitir un `RunBlocked` incompatible.
- El manifiesto `goal_receipts_manifest.json` de `run-until-finalize` ya no
  acepta solo "N runs con receipts": valida la secuencia configurada hasta el
  final esperado, detecta `work_kind` faltantes, inesperados o desordenados y
  exige que `finalize_temario_package` sea el ultimo cierre cuando ese es el
  objetivo del smoke. Las repeticiones de un mismo `work_kind` son validas para
  colas reales con varios bloques, visuales, tests o assets en una fase.
- El submit goal-first de DomainWork tambien se activa cuando el cierre falla
  solo por required tests de dominio sin comando y las pruebas con comando ya
  estan pasadas. El receipt aceptado del ledger sintetiza la evidencia
  `required_test_results=passed` para esas pruebas de dominio.
- El backend `app_server_stdio` promociona un
  `orquesta_goal_result_v0.json` durable aunque el goal remoto haya quedado
  `blocked` por timeout/conector, siempre que el fichero pertenezca al
  `goal_ref`. Esto conserva artefactos tardios y deja que Orquesta cierre por
  adaptador externo.

## Pendiente

- Extender de `update_topic_registry` a la secuencia completa de 23
  `work_kind`. El contrato REST y el scope real ya estan validados; falta
  ejecutar/cerrar toda la cola temporal real hasta `finalize_temario_package`.
- No volver al loop legacy para tapar este caso: la ruta correcta sigue siendo
  goal-first con cierre por artefactos, tests y receipt de dominio.
