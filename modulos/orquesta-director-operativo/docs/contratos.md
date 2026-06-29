# Contratos: orquesta-director-operativo

## `OperationalDirectorRequestV0`

Entrada pura para pedir un plan operativo. Soporta dos modos de primer nivel:

- `programming`: autoprogramacion o cambios de software. Requiere worktree
  aislada, rama, write-set y tests obligatorios.
- `domain_work`: trabajo de dominio externo. Si el contexto es suficiente,
  requiere `domain_refs` opacas y `write_set` seguro; si es insuficiente, el
  plan pide contexto y no lanza subagentes.

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

El plan transporta una politica general de reparacion: conservar trabajo
aprovechable y decidir `normalize`, `request_correction`, `delegate_review`,
`sequence_followup` o `postpone` antes de rechazar. El rechazo fuerte queda
reservado para seguridad, causalidad rota, refs imposibles o efectos externos no
autorizados. Esta regla es neutral para todo Orquesta y no depende de Codex,
OPES ni otro conector de producto.
La entrada normaliza alias conservadores de forma para `mode` y
`context_status` (`domain-work`, `domain work`, `needs context`, etc.) antes de
validar, porque son errores reparables de contrato y no riesgos causales.

La version actual conserva el contrato de cohorte en dos niveles: las olas del
plan (`OperationalDirectorWaveWorkV0`) y la metadata neutral que el
materializador proyecta a `WorkflowTaskV0` (`wave_ref`, `cohort_ref`,
parent/child refs, profundidad y fanout). El wait del director debe esperar la
cohorte activa, no todo el run.

Primer corte real: el materializador de
`modulos/orquesta-orchestration-core/operational_director_materializer_v0.go`
consume planes `ready` y solo convierte items `launch_subagents` en
`WorkflowTaskV0` + `CreateMicrotask`. La espera por ola/cohorte se puede derivar
con `WorkflowTaskWaitAgentRefsV0`; el ciclo de espera, review, tests,
rework/replan y cierre causal ya tiene implementacion fuera de este contrato
puro, en `app-director-service`/`orchestration-core` y composiciones por puerto.
La frontera local sigue siendo DTO/validacion/proyeccion: no runtime, no
proveedor, no OPES interno y no cierre ejecutado desde este modulo.

Mapa de responsabilidades del ciclo:

| Tramo | Dato principal | Owner vigente |
| --- | --- | --- |
| Plan neutral | `OperationalDirectorPlanV0` | Este modulo puro. |
| Olas/items | `OperationalDirectorWaveWorkV0` | Este modulo puro. |
| Tasks/outbox | `WorkflowTaskV0`, `CreateMicrotask` | `orquesta-orchestration-core`. |
| Espera acotada | `WorkflowTaskWaitStateV0`, `WaitAgentRefs` | `app-director-service` + stores. |
| Review/tests/cierre | refs de delivery, review y evidencias | `app-director-service` + `orquesta-orchestration-core`. |
| Smokes reales | `CODEX-*`, `EXT-NO-OPES`, OPES temporal | Composiciones opt-in y matriz. |

## `OperationalDirectorWaveWorkV0`

Proyeccion pura de `OperationalDirectorPlanV0` a olas y work items neutrales.
Sirve como contrato previo para que una composicion externa convierta cada item
a tareas de workflow, outbox o runtime concreto.

La proyeccion:

- agrupa pasos por olas topologicas segun `depends_on`;
- conserva `work_profile_kind` para materializacion posterior como
  `WorkflowTaskV0`;
- conserva `domain_refs`, `write_set`, `required_tests`, evidencias y criterios;
- el materializador proyecta `domain_refs` y `evidence_refs` compactas como
  `WorkflowTaskV0.context_refs` tipadas para trazabilidad; no usa summaries,
  rutas internas ni detalles de producto como fuente primaria;
- expande `MaxParallelAgents` en varios pasos/items `launch_subagents` dentro de
  la misma ola, con sharding de `write_set` como ownership inicial para evitar
  bloqueos de concurrencia artificiales;
- expresa parent/child refs de delegacion como ids de work item;
- no importa ni conoce orchestration-core, Codex, OPES, DB, HTTP, filesystem ni
  runtime;
- marca `ready_to_launch=false` cuando el plan esta en `needs_context`.
- rechaza planes armados externamente que declaren parent/child refs
  inexistentes, profundidad/fanout fuera del presupuesto del plan, un ciclo
  operativo listo sin `launch_subagents -> wait_subagents -> review_deliveries
  -> replan_or_close`, `programming` sin `run_required_tests` o `domain_work`
  listo sin refs opacas/write-set.
- rechaza planes `needs_context` que intenten lanzar subagentes en vez de pedir
  contexto.

Si las dependencias del plan no son resolubles, devuelve issues y no inventa una
ola ejecutable.

## Politica de reparacion

`DecideOperationalDirectorRepairPolicyV0` clasifica salidas razonables o
ambiguas sin ejecutar runtime: normaliza alias/refs derivables, pide correccion
dirigida cuando falta evidencia, delega revision si la ambiguedad requiere otro
juicio, secuencia un paso posterior o pospone hasta tener contexto suficiente.
Solo devuelve `hard_reject` ante seguridad, causalidad rota, refs imposibles o
efecto externo no autorizado.

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

La recursion Codex productiva queda cerrada en modo proveedor real por
`CODEX-RECURSION-REAL`; la ola/cohorte amplia queda cerrada por
`CODEX-WAVE-REAL`. Offline y fake-runtime ya hay arbol 1->2->4, parent/child
refs, limites, waits acotados, presupuesto, review causal, cierre de arbol y
supervisor fake que avanza sin llamadas manuales por nivel. El smoke real ejecuto
el mismo arbol con Codex vivo, ACK/entregas reales y cierre causal del arbol.
OPES temporal real de derivados/cierre quedo cerrado funcionalmente por
goal-first. Lo que queda fuera de este modulo son residuales editoriales, coste,
automatizacion larga o blockers nuevos con evidencia propia, no el contrato puro.

## Ejemplos y errores frecuentes

Ver `docs/ejemplos.md` para casos practicos de `programming`, `domain_work` con
contexto insuficiente, `domain_work` listo, waits por ola/cohorte, recursion
gobernada y notas de prueba. Esos ejemplos no amplian la frontera del modulo:
runtime, proveedor, stores, OPES, Codex y cierre real siguen en adaptadores o
capas de aplicacion.
