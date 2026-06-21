# Incidencia OPES app-change: microtarea durable sin proyección en run

Fecha: 2026-06-20.

## Contexto

Durante la comprobación del job OPES `plan_temario` para `oficial-de-servicios-multiples`, el bridge OPES consiguió enviar el trabajo a Orquesta, pero el run quedó bloqueado y sin agentes lanzados.

El estado durable contenía:

- evento `MicrotaskCreated`;
- fichero de `workflow_task`;
- evento `RunBlocked` para `app-change-task`;
- evento `RunBlockerResolved`;
- agregado `run` aún en estado `bloqueada`, con `tasks=[]`.

El supervisor terminaba sin lanzar agentes porque la microtarea existía en el almacén, pero no estaba proyectada en `run.tasks`.

## Causa

La recuperación de bloqueos técnicos de `app-change autoplan` solo resolvía el blocker recuperable. Si el evento `MicrotaskCreated` ya estaba persistido pero su efecto se había perdido del agregado, Orquesta no re-proyectaba esa microtarea antes de continuar.

Además, el estado podía conservar un efecto de `RunBlockerResolved` sin haber eliminado todavía el blocker del agregado, por lo que la reparación debía seguir siendo idempotente.

## Corrección

Se ha modificado `modulos/orquesta-app-codex-stack/run_coordinator_app_change_recovery_v0.go` para que `recoverBlockedAppChangeAutoPlanRunV0`:

1. calcule las microtareas `task-ref-app-change-*` faltantes a partir de las decisiones del director;
2. busque eventos durables `MicrotaskCreated` de esas tareas;
3. re-proyecte esos eventos sobre el agregado sin mover `last_event_id` ni `last_sequence`;
4. guarde el run reparado;
5. resuelva el blocker de forma idempotente.

La prueba `TestCodexStackV0RecoverAppChangeAutoplanProyectaMicrotareaDurableV0` reproduce el fallo observado y exige que el run quede activo, sin blockers y con la tarea proyectada.

## Evidencia

Pruebas ejecutadas:

```bash
go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestCodexStackV0RecoverAppChangeAutoplanProyectaMicrotareaDurableV0|TestCodexStackV0RecoverReanudaAppChangeAutoplanBloqueadoSinMicrotareaV0|TestCodexStackV0RecoverReanudaOpenPlanTardioConReviewAceptadaV0|TestCodexStackV0RecoverNoReanudaAppChange'
go test -count=1 ./modulos/orquesta-app-codex-stack
go test -count=1 ./cmd/orquesta-server
```

Resultado: correctas.
