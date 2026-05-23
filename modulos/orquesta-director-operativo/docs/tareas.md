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
offline. Lo pendiente de este frente no es mas contrato puro, sino smoke con
Codex real amplio/recursivo y dominios reales por conectores.

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
