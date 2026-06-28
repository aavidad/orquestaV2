# Resultado smoke OPES derivados goal-first real 2026-06-28

Fecha: 2026-06-28.

## Resumen

Se ejecuto un smoke real acotado contra OPES temporal local y Orquesta temporal
con `ORQUESTA_CODEX_GOAL_BACKEND=app_server_stdio`. Tambien se intento recorrer
la secuencia real completa de derivados hasta `finalize_temario_package`.

Resultado: contrato REST y scope OPES temporales validados; unitario cerrado
para `update_topic_registry`; primer intento completo con timeout por defecto
bloqueado correctamente; segundo intento completo con timeout ampliado cierra el
primer goal real y deja el segundo goal activo al llegar a `MAX_TICKS=80`.
Orquesta acepta `POST /api/v0/external-work/run`, crea run goal-first, arranca
Goal Codex real observable, valida artefacto/receipt y cierra el run sin
reabrir el loop legacy. Lo pendiente no es volver al Director antiguo, sino
reducir contexto y paralelizar/continuar la secuencia real de 23 work kinds.

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
- Spec observado antes de la compactacion final de este corte:
  `spec_bytes=24154`, `context_refs=61`,
  `input_values=15`, `payloads=15`. Las politicas OPES masivas quedaron como
  `payload-ref`, no inlineadas. El contrato vigente reduce mas esas refs y se
  cubre por unitarias; queda pendiente reejecutar el smoke largo con ese spec.

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

Quinto intento de secuencia completa con timeout por defecto:

- Root temporal: `/tmp/orquesta-opes-real-20260628T112147Z`.
- OPES temporal: `http://127.0.0.1:18192`.
- Orquesta temporal: `http://127.0.0.1:19192`.
- Probe de contrato REST:
  `/tmp/orquesta-opes-real-20260628T112147Z/smoke/opes_derivatives_rest_contract_probe.json`;
  `status=ok`, `accepted_count=23`, `rejected_count=0`,
  `transport_compat=true`.
- Scope-probe:
  `/tmp/orquesta-opes-real-20260628T112147Z/smoke/opes_derivatives_scope_probe.json`;
  `scope_probe_status=ok`.
- Run:
  `run-external-work-opes-df79e68c50ba7bfc6e32c5ae895635c9-opes-job-df79e68c50ba7bfc6e32c5ae895635c9`.
- Resultado en tick 30, tras recuperacion acotada:
  `goal_status=blocked`, `closure_status=blocked`,
  `run_status=bloqueada`, `summary=codex_app_server_goal_active_timeout`.
- Lectura: el timeout por defecto de 30s no basta para el primer Goal Codex
  real de la secuencia. El bloqueo es controlado y no apunta a fallo de scope ni
  de contrato REST.

Sexto intento de secuencia completa con timeout ampliado:

- Root temporal: `/tmp/orquesta-opes-real-long-20260628T112901Z`.
- OPES temporal: `http://127.0.0.1:18192`.
- Orquesta temporal: `http://127.0.0.1:19192`.
- Configuracion relevante:
  `ORQUESTA_CODEX_GOAL_TIMEOUT_MS=600000`,
  `ORQUESTA_OPES_BRIDGE_MAX_TICKS=80`,
  `ORQUESTA_OPES_DERIVATIVES_TICK_SLEEP_SECONDS=5`,
  `ORQUESTA_OPES_BRIDGE_LIMIT=1`.
- Probe de contrato REST:
  `/tmp/orquesta-opes-real-long-20260628T112901Z/smoke/opes_derivatives_rest_contract_probe.json`;
  `status=ok`, `accepted_count=23`, `rejected_count=0`,
  `transport_compat=true`.
- Scope-probe:
  `/tmp/orquesta-opes-real-long-20260628T112901Z/smoke/opes_derivatives_scope_probe.json`;
  `scope_probe_status=ok`, `job_type=update_topic_registry`, `seen=1`,
  negative checks por `program_id`, `topic_id` y `correlation_id`.
- Primer run cerrado:
  `run-external-work-opes-180b79d04fa1f664c14a07594bf38fb0-opes-job-180b79d04fa1f664c14a07594bf38fb0`.
- Observacion tick 68:
  `/tmp/orquesta-opes-real-long-20260628T112901Z/smoke/run_until_finalize/observe_goal_68_run-external-work-opes-180b79d04fa1f664c14a07594bf38fb0-opes-job-180b79d04fa1f664c14a07594bf38fb0_response.json`;
  `run_status=cerrada`, `goal_status=complete`,
  `closure_status=accepted`, `closure_accepted=true`,
  `evidence-ref-goal-domain-receipt-ledger-accepted` y
  `opes-artifact-ref-1dfc63c0cfc29eb01e3fcbb592e262a9`.
- Artefactos del primer run:
  `/tmp/orquesta-opes-real-long-20260628T112901Z/project/external/opes/update_topic_registry/180b79d04fa1f664c14a07594bf38fb0/topic_registry_update.json`
  y
  `/tmp/orquesta-opes-real-long-20260628T112901Z/project/external/opes/update_topic_registry/180b79d04fa1f664c14a07594bf38fb0/docs/orquesta_goal_result_v0.json`.
- Segundo run observado al tick 80:
  `run-external-work-opes-0ff60cf481e5f87a5a0552091282fcc0-opes-job-0ff60cf481e5f87a5a0552091282fcc0`;
  `goal_status=running`, `summary=codex_app_server_goal_status_active`.
- Resultado global del wrapper: no alcanzo `finalize_temario_package` en 80
  ticks. La causa operativa observada es duracion/secuencialidad de goals reales
  con `limit=1`, no rechazo de contrato, scope ni vuelta al loop legacy.

## Cambios derivados del ajuste actual

- `BuildExternalWorkGoalWorkSpecV0` compacta `input_fields`: inlinea campos
  seguros/prioritarios, limita refs resolubles de payload y resume los campos
  masivos no prioritarios sin exponer nombre ni valor en el contexto del Goal.
  Esta mitigacion queda cubierta por unitarias y debe revalidarse en el
  siguiente smoke real largo.
- El criterio de aceptacion ya no bloquea por cualquier campo omitido por
  presupuesto; solo exige rework si falta un input imprescindible para el
  artefacto.

## Contexto previo del mismo corte

Los puntos siguientes pertenecen a cambios previos ya introducidos durante el
corte del 2026-06-28. El quinto y sexto intento documentados arriba verifican
sus efectos observables en el smoke temporal, pero no deben leerse como prueba
nueva e independiente de cada implementacion interna.

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
- Reejecutar el smoke largo despues de la compactacion actual del
  `GoalWorkSpecV0` para medir si baja el tiempo/tokens por goal.
- El wrapper ya soporta continuacion con
  `ORQUESTA_OPES_DERIVATIVES_RESUME=1`, mismo `SMOKE_OUT_DIR`, mismo ledger y
  misma Orquesta temporal. Falta usar esa continuacion en un run real largo
  hasta `finalize_temario_package`.
- Paralelizar por fases sigue pendiente cuando el contrato de dependencias lo
  permita; con `limit=1` y goals Codex reales, 80 ticks no bastan para 23 work
  kinds.
- No volver al loop legacy para tapar este caso: la ruta correcta sigue siendo
  goal-first con cierre por artefactos, tests y receipt de dominio.
