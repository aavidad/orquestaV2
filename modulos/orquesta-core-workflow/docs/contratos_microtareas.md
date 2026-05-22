# Contratos de microtareas: orquesta-core-workflow

Este archivo contiene contratos locales de microtareas. El indice global del modulo
sigue en `docs/contratos.md`.

Nota de producto 2026-05-10: `CreateMicrotask` y `MicrotaskCreated` son nombres
durables v0. La semantica real es crear una `WorkflowTaskV0`, es decir una
unidad de trabajo cerrada. Normalmente sera pequena, pero puede ser mediana o
grande si el director justifica contrato estable, write-set controlado,
checkpoints, tests y observabilidad. Lo prohibido sigue siendo el macroparche
opaco o el fichero inmanejable, no una tarea cohesiva de mayor tamano.

## `WorkflowTaskV0`

```text
Nombre: WorkflowTaskV0
Tipo: dto
Version: v0 candidato local
Propietario: orquesta-core-workflow
Consumidores: core-workflow, planificador de unidades de trabajo futuro, observability como proyeccion
Campos:
  - schema_version
  - task_id
  - run_id
  - phase_id
  - work_profile_kind
  - title
  - summary
  - write_set
  - acceptance_criteria
  - function_contract_refs
  - context_refs
  - parent_task_ref, cohort_ref, wave_ref, delegation_depth,
    max_delegation_depth, max_child_agents, max_subagents_per_agent,
    max_recursive_agents, child_task_refs
Invariantes:
  - DTO puro sin runtime, DB, proveedor, HOME, agentes concretos ni adaptadores.
  - Representa una unidad de trabajo acotada; el tamano lo decide el director por politica, no el core.
  - `phase_id` pertenece al catalogo de fases v0.
  - `work_profile_kind` es opcional; si viene debe pertenecer al catalogo neutral de `WorkProfileV0`.
  - `task_id`, `run_id`, `phase_id` y `title` son obligatorios.
  - `write_set` y `acceptance_criteria` son listas compactas no vacias.
  - `function_contract_refs` contiene referencias opacas; el DTO base acepta `contract_ref` o `function_name` para compatibilidad local.
  - `context_refs` contiene refs opacas compactas opcionales de la composicion o app externa; no son campos de producto ni conceptos de programacion.
  - `max_delegation_depth`, `max_subagents_per_agent` y `max_recursive_agents`
    son presupuestos opcionales del arbol de tareas; `0` conserva
    compatibilidad sin limite estructurado.
  - Rechaza detalles de DB, SQL, runtime, provider/proveedor, HOME, OAuth, Docker, tmux y secretos.
Errores:
  - workflow_task_invalida
  - fase_no_soportada
  - detalle_prohibido
  - payload_invalido
Estado: implementado local en NCW-008.
```

## `WorkProfileV0`

```text
Nombre: WorkProfileV0
Tipo: dto/fabrica neutral
Version: v0 candidato local
Propietario: orquesta-core-workflow
Consumidores: director operativo, scheduler, conectores de dominio como entrada previa a WorkflowTaskV0
Campos:
  - schema_version
  - profile_ref
  - profile_kind
  - task_ref
  - run_ref
  - phase_id
  - title
  - objective
  - summary
  - scope_refs
  - acceptance_criteria
  - required_tests
  - depends_on
  - parent_task_ref
  - cohort_ref
  - wave_ref
  - delegation_depth
  - max_delegation_depth
  - max_child_agents
  - max_subagents_per_agent
  - max_recursive_agents
  - child_task_refs
  - function_contract_refs
Perfiles iniciales:
  - code_study
  - implementation
  - refactor
  - required_tests
  - documentation
  - review
  - domain_work
Invariantes:
  - DTO puro sin adaptador, proveedor, proceso real, HOME, credenciales ni conocimiento interno de conectores.
  - Reutiliza `WorkflowTaskV0`: `WorkflowTaskFromWorkProfileV0` valida el perfil y produce una tarea durable compacta con `work_profile_kind`.
  - `profile_kind` decide fase y criterios base por politica neutral; los conectores aportan refs, reglas y validadores.
  - `implementation`, `refactor` y `required_tests` exigen `required_tests` antes de poder materializarse.
  - Conserva linaje neutral de delegacion: parent task, cohorte, ola,
    profundidad, limite de profundidad, fanout local/global, presupuesto global
    recursivo e hijos.
  - Exige `function_contract_refs` para no crear tareas sin contrato invocable.
Errores:
  - work_profile_invalido
  - work_profile_task_invalida
Estado: implementado local en NCW-075.
```

## `CreateMicrotask`

```text
Nombre: CreateMicrotask
Tipo: comando
Version: v0 candidato local
Propietario: orquesta-core-workflow
Consumidores: adaptadores inbound futuros, planificador de unidades de trabajo futuro
Payload:
  - task: WorkflowTaskV0
Salida:
  - MicrotaskCreated
Invariantes:
  - Handler puro: no runtime, DB, proveedor, HOME, filesystem ni outbox.
  - El nombre del comando no fuerza granularidad minima; fuerza planificacion cerrada y verificable.
  - Valida `WorkflowTaskV0`: fase soportada, `write_set` no vacio, `acceptance_criteria` no vacio, detalles prohibidos y payload compacto.
  - Valida que la proyeccion `MicrotaskCreated` sea compacta antes de aceptar el comando.
  - Requiere fase actual activa `planificacion_microtareas`.
  - `function_contract_refs` es obligatorio en el comando durable.
  - Cada entrada del comando durable debe traer `contract_ref` explicito; `function_name` solo es metadato compatible del DTO base.
  - Cada `contract_ref` debe existir ya en `OrchestrationRunV0.FunctionContracts`.
  - `task.run_id` debe coincidir con `command.run_id`.
  - Si `task_id` ya esta en `OrchestrationRunV0.Tasks`, el comando debe coincidir con la huella `CommandEffects` original.
Errores:
  - payload_invalido
  - detalle_prohibido
  - fase_no_soportada
  - transicion_invalida
Estado: implementado local en NCW-009; identidad durable endurecida en NCW-057.
```

## `MicrotaskCreated`

```text
Nombre: MicrotaskCreated
Tipo: evento
Version: v0 candidato local
Propietario: orquesta-core-workflow
Consumidores: reducer, replay durable, observability futura
Payload:
  - task_id
  - phase_id
  - function_contract_refs: lista de strings compactos ya normalizados
Invariantes:
  - Evento compacto: no guarda `write_set`, criterios completos, runtime, DB, proveedor, HOME ni adaptadores.
  - `task_id` y `phase_id` son obligatorios.
  - `phase_id` pertenece al catalogo v0.
  - `function_contract_refs` son opacas y compactas; no guarda par redundante `contract_ref` + `function_name`.
  - Solo se aplica con fase actual activa `planificacion_microtareas`.
  - Cada ref debe existir ya en `OrchestrationRunV0.FunctionContracts`.
  - `ApplyEventV0` proyecta `task_id` en `tasks` y refs compactas en `function_contracts` sin duplicar.
  - Proyecta una huella `CommandEffects` por `(MicrotaskCreated, task_id)`.
  - Rechaza otra microtarea durable con el mismo `task_id` y distinta key, event_id, causation_id o payload.
  - `ReplayDurableEventsV0` acepta el evento con sequence estricta.
Errores:
  - evento_invalido
  - payload_invalido
  - detalle_prohibido
  - secuencia_invalida
Estado: implementado local en NCW-009; identidad durable endurecida en NCW-057.
```
