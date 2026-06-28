# Tarea Orquesta - Runs OPES Ready Sin Dispatch Real En Psicólogo

Fecha: 2026-06-26.

## Contexto

Workspace OPES: `/home/alberto/Trabajo/OPES`.
Servidor Orquesta de Psicólogo ya activo, no arrancado de nuevo:

- URL: `http://127.0.0.1:19024`
- State dir: `/home/alberto/Trabajo/OPES/opes-salidas/codex_directo/psicologia/A1_A2/revision_profesional_2026-06-26/orquesta_server/19024_state`
- Runtime dir: `/home/alberto/Trabajo/OPES/opes-salidas/codex_directo/psicologia/A1_A2/revision_profesional_2026-06-26/orquesta_server/19024_runtime`

## Síntoma

`/api/v0/autoprogramming/status` muestra cola viva y runs OPES en `ready`, pero no hay procesos Codex vivos asociados después de llamar a supervise.

Ejemplo observado:

- `run-opes-psicologo-a1a2-auditoria-transversal_001_090-20260626-19024`: `ready`
- `run-opes-psicologo-a1a2-auditoria-bloque_063_078-20260626-19024`: `ready`
- `run-opes-psicologo-a1a2-auditoria-bloque_079_090-20260626-19024`: `ready`
- `run-opes-psicologo-a1a2-auditoria-comunes_001_018-20260626-19024`: `ready`
- `run-opes-qa-visual-remota-tcae-psicologo-asg-operario-20260626`: `running`, pero diagnosticado como `running_stale_no_process`.

El servidor reporta:

- `last_supervisor_status=stalled`
- `last_supervisor_queue_size=7`
- `resident_director_status=disabled`
- `operator.safe_actions[0].endpoint=/api/v0/autoprogramming/supervise`

Después de:

```bash
curl -fsS -X POST http://127.0.0.1:19024/api/v0/autoprogramming/supervise \
  -H 'content-type: application/json' \
  -d '{"request_id":"opes-psicologo-supervise-comunes-strict-20260626","queue_ref":"global","max_dispatches":6,"max_outbox":12,"include_process_refs":true}'
```

la cola siguió `ready` y `ps -ef` no mostró nuevos procesos Codex de esas runs.

## Impacto OPES

Bloquea autonomía real del cierre de Psicólogo: el director OPES tiene que continuar por desbloqueo local/subagentes Codex aunque Orquesta debería materializar las runs listas.

## Resultado esperado

Cuando una run OPES está `ready` y `supervise` es acción segura, Orquesta debe:

1. despachar agente real o explicar bloqueo público concreto;
2. actualizar estado a `running` con process refs vivos, `blocked` con motivo o `failed` recuperable;
3. no dejar indefinidamente `ready` sin dispatch real ni ACK.

## Acción pedida al agente de Orquesta

Corregir en la app de Orquesta, no con workaround OPES: revisar reconciliación `ready`/`running_stale_no_process`, dispatch de outbox y contrato de `/api/v0/autoprogramming/supervise` para que una cola con candidatos materialice agentes o devuelva bloqueo operativo actionable.

## Revalidación Orquesta 2026-06-28

Estado: revalidada en Orquesta como incidencia cubierta para el stack actual,
sin reabrir el servidor productivo de Psicólogo `19024`.

Evidencia de cola y supervisor:

- `TestCodexStackV0RunGlobalTickReconciliaRunningStaleAntesDeReadyV0` valida
  que una run OPES en `running` con `process.status=stopped` se reconcilia como
  `stopped`, queda marcada con
  `evidence-ref-run-queue-running-stale-no-live-process-reconciled`, se registra
  el agente como `lost` y el mismo tick despacha la run `ready` siguiente.
- `TestCodexStackRunSupervisorQueueDiagnosticsMCPV0ExponeNoExecutionConReady`
  valida que una supervisión global que termina sin ejecuciones pero conserva
  candidatos ejecutables ya no queda muda: publica
  `run_supervisor_queue_no_execution_with_ready_candidates`, `executions=0`,
  límites efectivos, `top_candidates` y la acción
  `supervise_with_resident_mode_or_run_ref`.
- `TestCodexStackRunSupervisorQueueDiagnosticsMCPV0ExponePresionWaitingOutbox`
  cubre el diagnóstico de presión de cola cuando el runtime está en
  `waiting_outbox`, con conteo de `ready`, `running`, `stopped` y ejecutables.
- `TestServerAutoprogrammingSuperviseHTTPClienteRealRecibeCuerpoSinColgarV0`
  valida que `POST /api/v0/autoprogramming/supervise` responde a cliente HTTP
  real con `202 accepted_background`, `operation_ref` y acciones de polling en
  vez de dejar el cliente colgado.
- `TestServerQueueGlobalStatusHTTPClienteRealMontadoEnStackV0` valida que el
  endpoint público de estado global de cola está montado para observar la cola
  tras el `accepted_background`.

Lectura operativa:

- El resultado esperado queda cubierto por dos salidas válidas: dispatch real
  cuando el tick puede ejecutar, o diagnóstico público accionable cuando una
  supervisión termina con candidatos `ready` pero `executions=0`.
- La reconciliación `running_stale_no_process` ya no debe bloquear candidatos
  `ready`: el tick reconcilia primero el stale y después ejecuta el candidato
  listo.
- Para trabajos OPES nuevos con backend Goal, la ruta preferente no es volver al
  loop residente legacy: el smoke real temporal documentado en
  `docs/runbooks/resultado_smoke_opes_derivados_goal_first_real_2026-06-28.md`
  cerró 24/24 jobs del scope temporal hasta `completed_syllabus_package` con
  `ORQUESTA_EXTERNAL_WORK_LEGACY_DIRECTOR_LOOP=0`,
  `ORQUESTA_AUTOPROGRAMMING_LEGACY_DIRECTOR_LOOP=0` y
  `ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED=false`.

Frontera:

- No se tocó ni se drenó el Orquesta productivo de Psicólogo `19024`. La
  revalidación se basa en tests focales del stack/HTTP y en smoke OPES temporal
  goal-first. Si vuelve a aparecer una cola real `ready` sin dispatch, el dato
  decisivo a capturar es el diagnóstico público del supervisor y el estado de
  `/api/v0/runs/queue/global/status`, no sólo el `status` agregado legacy.
