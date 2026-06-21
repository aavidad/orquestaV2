# Incidencia OPES Servicios Múltiples: generate_tutor_assets crea run vacía

Fecha: 2026-06-20.

## Contexto

Job OPES:

- `job_ref=c767e438ef13510d0a3dff3797e34f80`;
- `work_kind=generate_tutor_assets`;
- `expected_artifact_type=tutor_bot_package`;
- curso `oficial-servicios-multiples-c2-20260620`.

El dry-run de `opes-drain-once` confirmó que el job era convertible:

- `seen=1`;
- `status=dry_run`;
- `change_ref=opes-job-c767e438ef13510d0a3dff3797e34f80`.

Pero el submit real devolvió primero `effect_timeout` y después
`external_error`. La supervisión manual de:

`run-external-work-opes-c767e438ef13510d0a3dff3797e34f80-opes-job-c767e438ef13510d0a3dff3797e34f80`

respondió `done` con:

- `tasks=0`;
- `open_tasks=0`;
- `requested_agents=0`;
- sin carpeta de run útil en `.orquesta-runtime`;
- sin agente Codex ni artefacto en `external/opes/generate_tutor_assets/...`.

## Impacto

El bridge deja el job OPES `pending`, pero Orquesta cree que la run está
`done` sin haber creado ninguna tarea. Esto rompe la autonomía: el director
humano debe detectar que no hay trabajo real y relanzar o documentar rework.

## Arreglo esperado

Para `generate_tutor_assets` y trabajos equivalentes:

- si `/external-work/run` falla después de registrar run, la proyección debe
  contener al menos una tarea o una incidencia causal;
- una run `done` con cero tareas para un `external_work` convertible debe
  convertirse automáticamente en `failed_empty_run` o `retry_pending`, nunca en
  `done`;
- `opes-drain-once` debe reconciliar `submit_failed + empty_run` y permitir un
  relanzamiento causal sin `existing_change_conflict`;
- el status público debe exponer contador `external_work_empty_run`.

## Criterio de cierre

Smoke real acotado con `generate_tutor_assets`:

- dry-run convertible;
- submit crea una tarea y al menos un agente, o falla con causa recuperable;
- supervisión no devuelve `done` con cero tareas;
- OPES recibe `tutor_bot_package` o queda rework causal durable.
