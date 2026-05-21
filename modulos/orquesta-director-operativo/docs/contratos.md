# Contratos: orquesta-director-operativo

## `OperationalDirectorRequestV0`

Entrada pura para pedir un plan operativo. Soporta dos modos de primer nivel:

- `programming`: autoprogramacion o cambios de software. Requiere worktree
  aislada, rama, write-set y tests obligatorios.
- `domain_work`: trabajo de dominio externo. Si el contexto es insuficiente,
  el plan pide contexto y no lanza subagentes.

## `OperationalDirectorPlanV0`

Plan vivo compacto con:

- presupuesto de bucles;
- maximo de subagentes paralelos;
- pasos `gather_context -> split_work -> launch_subagents -> wait_subagents ->
  review_deliveries -> replan_or_close`;
- paso `run_required_tests` cuando el modo es `programming`;
- paso `request_domain_context` cuando falta contexto de dominio.
- paso `govern_delegation` cuando se permite delegacion recursiva.

El contrato no ejecuta runtime ni decide proveedor/modelo. Una composicion
externa debe convertir estos pasos a comandos del workflow y outbox.
Cada paso puede transportar `work_profile_kind` neutral: estudio de contexto,
implementacion, trabajo de dominio, pruebas requeridas o revision. Ese dato no
elige proveedor; solo permite que el scheduler resuelva rol/capacidad por
contrato.

La version actual conserva el contrato de cohorte en dos niveles: las olas del
plan (`OperationalDirectorWaveWorkV0`) y la metadata neutral que el
materializador proyecta a `WorkflowTaskV0` (`wave_ref`, `cohort_ref`,
parent/child refs, profundidad y fanout). El wait del director debe esperar la
cohorte activa, no todo el run.

Primer corte real: el materializador de
`modulos/orquesta-orchestration-core/operational_director_materializer_v0.go`
consume planes `ready` y solo convierte items `launch_subagents` en
`WorkflowTaskV0` + `CreateMicrotask`. La espera por ola/cohorte se puede derivar
con `WorkflowTaskWaitAgentRefsV0`, pero todavia no esta conectada al ciclo
completo de espera, review, tests, rework/replan ni cierre.

## `OperationalDirectorWaveWorkV0`

Proyeccion pura de `OperationalDirectorPlanV0` a olas y work items neutrales.
Sirve como contrato previo para que una composicion externa convierta cada item
a tareas de workflow, outbox o runtime concreto.

La proyeccion:

- agrupa pasos por olas topologicas segun `depends_on`;
- conserva `work_profile_kind` para materializacion posterior como
  `WorkflowTaskV0`;
- conserva `domain_refs`, `write_set`, `required_tests`, evidencias y criterios;
- expresa parent/child refs de delegacion como ids de work item;
- no importa ni conoce orchestration-core, Codex, OPES, DB, HTTP, filesystem ni
  runtime;
- marca `ready_to_launch=false` cuando el plan esta en `needs_context`.

Si las dependencias del plan no son resolubles, devuelve issues y no inventa una
ola ejecutable.

## Delegacion Recursiva Gobernada

`AllowRecursiveDelegation` permite que un agente pida subagentes, por ejemplo
para que en OPES un agente de tema delegue bloques, visuales o revisiones. El
plan solo lo permite con limites:

- `MaxDelegationDepth`;
- `MaxSubagentsPerAgent`;
- `MaxParallelAgents`;
- `LoopBudget`;
- paso explicito `govern_delegation` antes de review.

La composicion real debe registrar parent/child refs, presupuestos consumidos y
evidencias. Un agente no puede lanzar hijos fuera del plan ni cerrar sin review.
Las decisiones producidas por un agente director hijo no deben consumirse hasta
que su ACK/artefacto quede registrado causalmente.

La recursion Codex productiva sigue pendiente. No basta con que el contrato
acepte delegacion: el stack real debe probar parent/child refs, limites,
`WaitAgentRefs` o refs de ola, presupuesto y review causal antes de anunciarla
como cerrada.
