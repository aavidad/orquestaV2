# Tareas: orquesta-director-operativo

## DIR-OP-001: contrato puro del plan operativo

Estado: hecho.

Validacion:

```sh
go test -count=1 ./modulos/orquesta-director-operativo
```

## DIR-OP-002: olas y cohortes del plan

Estado: hecho.

`BuildOperationalDirectorWaveWorkV0` proyecta pasos pendientes en olas
topologicas, conserva write-set, tests, evidencias, parent/child refs,
profundidad y fanout.

Validacion: `TestBuildOperationalDirectorWaveWorkV0*`.

## DIR-OP-003: espera durable por ola/cohorte

Estado: hecho primer corte fuera del modulo puro.

Objetivo: que la composicion convierta una ola materializada en espera concreta
por `WaitAgentRefs`, usando `cohort_ref`, `wave_ref` o `parent_task_ref`, sin
esperar todos los agentes vivos del run.

No pertenece a este modulo puro. Debe cerrarse en
`orquesta-orchestration-core` y `orquesta-app-director-service`.

Implementacion: `WorkflowTaskWaitStateV0`,
`BuildWorkflowTaskWaitSnapshotV0` y resolucion de wait en
`ContinueAppDirectorV0`.

## DIR-OP-004: review/rework/replan/cierre durable

Estado: cerrado offline fuera del modulo puro.

Objetivo: convertir `review_deliveries`, `run_required_tests` y
`replan_or_close` en comandos/estado de workflow con evidencia causal.

No cerrar tareas por ACK textual. Cerrar solo con entrega, review aceptada,
tests requeridos si aplica y refs de evidencia.

Estado actual: el contrato sigue puro y no ejecuta runtime. La materializacion
real vive en `orquesta-orchestration-core` y `app-director-service`: review
aceptada, `RequestRework`, `RecordReplanDecision`, `split_task`, runner por
puerto de tests requeridos, cierre causal y replay `state-file` ya tienen corte
offline. `CODEX-WAVE-REAL` y `CODEX-RECURSION-REAL` cierran tambien el frente
Codex real opt-in. Lo pendiente de este frente no es mas contrato puro ni otro
smoke Codex generico; OPES temporal real de derivados/cierre quedo cerrado
funcionalmente por goal-first. Quedan residuales de calidad editorial, coste,
automatizacion larga o blockers nuevos con evidencia propia.

## DIR-OP-005: recursion gobernada real

Estado: cerrado en composicion Codex real opt-in; cerrado offline/fake-runtime
fuera del modulo puro.

Objetivo: demostrar con runtime real que un agente padre puede proponer hijos y
que Orquesta conserva control: limites de profundidad/fanout, presupuesto,
parent/child refs, wait por subarbol y review causal.

Estado actual: fake-runtime ya cubre arbol 1->2->4 con parent/child refs,
presupuesto global, profundidad/fanout, waits acotados, review causal y cierre
del arbol; el supervisor fake del stack avanza ese arbol sin llamadas manuales
por nivel. `CODEX-RECURSION-REAL` ejecuto el mismo recorrido con proveedor vivo,
ACK/entregas reales y cierre causal del arbol.

## Sincronizacion documental 2026-05-26

Este modulo no mantiene un backlog paralelo de integracion. Si una fuente local
usa la palabra "pendiente" para Director Operativo, debe clasificarla contra la
foto vigente:

| Frente | Clasificacion vigente |
| --- | --- |
| Contrato puro del plan, olas, presupuestos y delegacion | Cerrado en este modulo con tests locales. |
| Materializacion de `launch_subagents` | Cerrada fuera del modulo puro; el resto del ciclo no pertenece a este contrato. |
| Waits por ola/cohorte/parent e ingesta acotada | Cerrado para stack Codex; `WaitAgentRefs` vacio conserva compatibilidad legacy. |
| Review, tests requeridos, replan y cierre causal | Cerrado offline/fake-runtime; Codex real amplio y recursivo ya tienen evidencia opt-in. |
| OPES temporal real de derivados/cierre | Cerrado funcionalmente fuera de este modulo por goal-first; residuales editoriales/coste/automatizacion larga. |
