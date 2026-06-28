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
