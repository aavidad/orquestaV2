# Ejemplos operativos: orquesta-director-operativo

Este modulo solo construye y valida plan operativo puro. Los ejemplos describen
formas esperadas del contrato; ejecutar runtime, persistir tasks, esperar agentes
o cerrar runs pertenece a `orquesta-orchestration-core`,
`app-director-service`, stores y composiciones.

## Programacion lista

Supuesto minimo:

```text
mode=programming
context_status=sufficient
worktree_isolated=true
worktree_ref=worktree-ref-temporal
branch_ref=branch-ref-operational
write_set=modulos/orquesta-director-operativo,docs
required_tests=go test -count=1 ./modulos/orquesta-director-operativo
```

Resultado esperado:

- plan `ready`;
- pasos `gather_context -> split_work -> launch_subagents -> wait_subagents ->
  review_deliveries -> run_required_tests -> replan_or_close`;
- items `launch_subagents` materializables despues como `WorkflowTaskV0`;
- `required_tests` transportados hasta launch, review, tests y cierre.

Error frecuente: declarar `mode=programming` sin tests, rama, worktree aislada o
write-set. El builder debe rechazarlo; no se repara inventando defaults.

## Dominio con contexto insuficiente

Supuesto minimo:

```text
mode=domain_work
context_status=insufficient
missing_context=domain-ref-topic-snapshot,policy-ref-quality
```

Resultado esperado:

- plan `needs_context`;
- no hay `launch_subagents`;
- aparece `request_domain_context`;
- `ReadyToLaunch=false` y el materializador no debe crear tareas.

Error frecuente: lanzar subagentes para que inventen contexto de dominio. El
contrato debe pedir contexto o bloquear, porque la app externa conserva datos,
reglas y validadores.

## Dominio listo

Supuesto minimo:

```text
mode=domain_work
context_status=sufficient
domain_refs=domain-ref-job,domain-ref-topic-snapshot
write_set=domain:topic:123,domain:artifact:document_plan
required_tests=<vacio salvo politica de dominio inyectada fuera de este modulo>
```

Resultado esperado:

- usa el mismo ciclo operativo de subagentes, wait, review y replan/cierre;
- no fuerza `run_required_tests` de programacion;
- conserva refs opacas de dominio y write-set seguro;
- validadores, submit de artefactos y tests de dominio entran por adaptador o
  politica inyectada fuera del contrato puro.

Error frecuente: meter OPES, REST, DB, paths locales o validaciones de producto
en este paquete. El contrato solo transporta refs opacas.

## Olas, waits y recursion

La proyeccion a olas conserva `wave_ref`/`cohort_ref` y linaje para que las
capas posteriores esperen el scope activo. No usar el resultado para esperar
todos los agentes vivos del run.

En recursion gobernada:

- `AllowRecursiveDelegation=true` solo abre la posibilidad contractual;
- profundidad, fanout y presupuesto se conservan en el plan;
- parent/child refs reales se registran despues en `WorkflowTaskStore` y eventos;
- las decisiones de hijos no se consumen antes de ACK/entrega causal.

Error frecuente: tratar `govern_delegation` como permiso para spawn libre. La
composicion debe volver a pasar por workflow, outbox, waits acotados y review.

## Notas de prueba

Focal local:

```sh
go test -count=1 ./modulos/orquesta-director-operativo
```

Para cambios que afecten materializacion, waits, `PlanState`, review, tests
requeridos o cierre, el modulo local no basta. Usar tambien las baterias
documentadas en `docs/matriz_pruebas_reales_y_smoke_2026-05-17.md`,
especialmente `DIRECTOR-TARDE-OFFLINE`, `DIRECTOR-GENERIC-CLOSURE-OFFLINE` y
`DIRECTOR-PLAN-STATE-OFFLINE`.

No relanzar `CODEX-WAVE-REAL`, `CODEX-RECURSION-REAL` ni OPES temporal real de
derivados/cierre salvo regresion demostrada; OPES quedo cerrado funcionalmente
por el runbook goal-first real del 2026-06-28.
