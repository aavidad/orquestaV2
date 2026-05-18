# Contexto para agentes: orquesta-orchestration-core

Lee este fichero antes de trabajar en este directorio.

## Reglas

- Mantener funciones y ficheros pequenos.
- No importar `orquesta/cmd`, `orquesta/db` ni adaptadores de base de datos.
- No hardcodear SQLite, Postgres, MySQL, Redis ni ningun motor.
- No conocer proveedores de agentes concretos ni rutas HOME/OAuth/tokens.
- Todo efecto externo entra por puerto.
- No recuperar `cmd`, `db`, `internal` ni control-plane legacy desde este miniproyecto.
- No borrar materializadores, candidatos ni documentos historicos sin revisar
  llamadas, tests y docs vigentes.

## Objetivo local

Unir los modulos limpios del nucleo:

- `orquesta-director-supervised-burst`;
- `orquesta-director-cycle`;
- `orquesta-director-scheduler`;
- `orquesta-director-supervisor`;
- `orquesta-core-workflow`;
- `orquesta-core-replanner`;
- `orquesta-core-leases`;
- `orquesta-director-cycle-outbox`.

## Director Operativo

El materializador vigente esta en `operational_director_materializer_v0.go`.
Toma planes puros de `modulos/orquesta-director-operativo`, usa
`BuildOperationalDirectorWaveWorkV0` y materializa solo items
`launch_subagents` como `WorkflowTaskV0` + `CreateMicrotask`.

Las tasks materializadas deben conservar metadata neutral de linaje:
`wave_ref`, `cohort_ref`, `parent_task_ref`, `child_task_refs`,
`delegation_depth` y `max_child_agents`. Para waits por ola/cohorte usa
`WorkflowTaskWaitAgentRefsV0`; no metas semantica Codex/OPES aqui y no parsees
texto de criterios como fuente primaria.

`WorkflowTaskWaitAgentRefsV0` solo debe cargar tasks abiertas autorizadas por
`run.Tasks` y descartar `run.ClosedTasks`. La metadata completa vive en
`WorkflowTaskStore`; no asumas que un replay solo de eventos compactos puede
reconstruir `wave_ref`, `cohort_ref` o parent/child refs.

`BuildWorkflowTaskWaitSnapshotV0` y `WorkflowTaskWaitStateV0` son el primer
corte durable de espera por ola/cohorte/parent task. Guardan causa, scope,
tasks, agentes objetivo y agentes pendientes por puertos; no metas persistencia
file/DB aqui.

Pendiente antes de llamarlo "ciclo completo":

- review, tests requeridos, rework/replan y cierre como workflow durable;
- parent/child refs propagados hasta runtime cuando se autorice recursion;
- guardas para no lanzar OPES/domain-work con contexto insuficiente.

## Test minimo

```bash
go test ./modulos/orquesta-orchestration-core -count=1
```

Si se cambia un contrato importado desde `modulos/`, ejecutar tambien el test
del modulo afectado.
