# Tareas locales: orquesta-director-scheduler

```text
ID: SCH-012
Objetivo: Ordenar microtareas de split desde ReplanFollowupCandidates sin inventar payloads.
Causa raiz: el rework de revision podia reabrir programacion, pero `split_task` solo podia preguntar al director y no autorizar microtareas nuevas.
Tipo: contrato_scheduler
Contrato afectado: BuildDirectorSchedulerTick v0 + SchedulableReplanFollowupCandidateV0
Test rojo minimo: replan split de review rework en revision produce RecordReplanDecision, OpenPhase y CreateMicrotask en orden.
Write-set: scheduler_tick_replan_*_v0.go, scheduler_tick_collect_v0.go, tests, docs locales
Validacion: go test -count=1 ./modulos/orquesta-director-scheduler
Riesgo de acoplamiento: bajo; tasks y comandos llegan como candidates explicitos, sin DB/runtime/proveedor/modelo/HOME/OAuth.
Estado: completada
```

```text
ID: SCH-011
Objetivo: Planificar el corte de revision no aceptada como `RequestRework` durable desde candidates explicitos.
Causa raiz: `RecordReviewResult` deja durable el resultado, pero el tick necesitaba traducir resultados `changes_requested` o `rejected` a una solicitud de retrabajo sin inventar replan ni relanzar agentes.
Tipo: contrato_scheduler
Contrato afectado: BuildDirectorSchedulerTick v0 + RunSchedulingSnapshotV0 + SchedulableReviewGateCandidateV0
Test rojo minimo: candidate de revision emite `RequestReview`, despues `RecordReviewResult` y, si el resultado durable es `changes_requested` o `rejected`, `RequestRework`; si el resultado es `accepted`, solo puede emitir `AcceptReview`.
Write-set: scheduler_tick_review_gate_*_v0.go, scheduler_tick_*_v0.go, *_test.go, docs locales
Validacion: go test -count=1 ./modulos/orquesta-director-scheduler -run 'TestBuildDirectorSchedulerTickV0ReviewGate'
Riesgo de acoplamiento: bajo; solo refs opacas, sin DB/runtime/proveedor/modelo/HOME/OAuth.
Estado: completada
```

```text
ID: SCH-010
Objetivo: Registrar artefactos de fase desde candidates explicitos sin usar DeliveryRegistered.
Causa raiz: El director de brainstorming genera documentos validos, pero no hay task de programacion ni delivery de codigo.
Tipo: contrato_scheduler
Contrato afectado: BuildDirectorSchedulerTick v0 + RunSchedulingSnapshotV0 + RegisterPhaseArtifact
Test rojo minimo: candidate de brainstorming con agente started emite RegisterPhaseArtifact; artefacto ya registrado queda quiescent; agente no arrancado espera; run ajeno falla.
Write-set: scheduler_tick_phase_artifact_*_v0.go, scheduler_tick_*_v0.go, *_test.go, docs locales
Validacion: go test -count=1 ./modulos/orquesta-director-scheduler
Riesgo de acoplamiento: bajo; solo refs opacas, sin ACK/path/DB/runtime/proveedor/HOME.
Estado: completada
```

```text
ID: SCH-009
Objetivo: Bloquear work de documentacion/avance cuando exista quality gate bloqueante sin followup de replan explicito.
Causa raiz: El tick podia evaluar work posterior aunque el caller trajera evidencia de gate bloqueante.
Tipo: invariante
Contrato afectado: BuildDirectorSchedulerTick v0 + RunSchedulingSnapshotV0
Test rojo minimo: quality gate blocked en programacion exige replan/followup o bloqueo antes de docs/avance.
Write-set: orquesta-director-scheduler/*.go, *_test.go, docs locales
Validacion: go test -count=1 .
Riesgo de acoplamiento: bajo; solo refs opacas, sin DB/runtime/proveedor/HOME.
Estado: completada
```

```text
ID: SCH-000
Objetivo: Crear microproyecto de scheduler del director con contexto local.
Write-set: orquesta-director-scheduler/**
Contrato: BuildDirectorSchedulerTick v0
Validacion: git diff --check -- modulos/orquesta-director-scheduler
Bloqueos: ninguno
Estado: completada
```

```text
ID: SCH-008
Objetivo: Fijar prioridad base del tick y dedupe por refs compactas.
Write-set: orquesta-director-scheduler/*.go, *_test.go, docs locales
Contrato: BuildDirectorSchedulerTick v0
Validacion: 2026-05-06, ok; go test -count=1 ./modulos/orquesta-director-scheduler
Bloqueos: no listar outbox, no leer DB, no despachar runtime, no inventar refs ni payloads. La prioridad vigente se amplio despues a outbox > lease > phase_artifact > delivery > review_gate > progress > replan > work.
Estado: completada
```

```text
ID: SCH-007
Objetivo: Anadir ReplanFollowupCandidates a BuildDirectorSchedulerTick v0 usando BuildReplanFollowupsV0.
Write-set: orquesta-director-scheduler/*.go, *_test.go, docs locales
Contrato: BuildDirectorSchedulerTick v0 + orquesta-director.BuildReplanFollowups v0
Validacion: 2026-05-06, ok; go test -count=1 ./modulos/orquesta-director-scheduler
Bloqueos: no decidir replan, no crear capacity/agent refs, no saltar CapacityDecided, no reutilizar agentes bloqueados.
Estado: completada
```

```text
ID: SCH-006
Objetivo: Anadir ProgressSupervisionCandidates a BuildDirectorSchedulerTick v0 usando BuildAgentProgressSupervisionV0.
Write-set: orquesta-director-scheduler/*.go, *_test.go, docs locales
Contrato: BuildDirectorSchedulerTick v0 + orquesta-director.BuildAgentProgressSupervision v0
Validacion: 2026-05-06, ok; go test -count=1 ./modulos/orquesta-director-scheduler
Bloqueos: no evaluar progreso, no consultar runtime, no leer logs, no tocar provider/modelo/HOME/OAuth.
Estado: completada
```

```text
ID: SCH-005
Objetivo: Ampliar RunSchedulingSnapshotV0 con refs compactas para dedupe de progreso, preguntas y replan.
Write-set: orquesta-director-scheduler/*.go, *_test.go, docs locales
Contrato: RunSchedulingSnapshotV0
Validacion: 2026-05-06, ok; go test -count=1 ./modulos/orquesta-director-scheduler
Bloqueos: solo refs opacas; no payloads, transcripts, rutas, DB, provider, modelo, HOME, OAuth ni runtime real.
Estado: completada
```

```text
ID: SCH-004
Objetivo: Anadir lease action candidates a BuildDirectorSchedulerTick v0 usando BuildPostLeaseActionV0.
Write-set: orquesta-director-scheduler/*.go, *_test.go, docs locales
Contrato: BuildDirectorSchedulerTick v0 + orquesta-director.BuildPostLeaseAction v0
Validacion: 2026-05-06, ok; go test -count=1 .
Bloqueos: no evaluar leases, no usar reloj interno, no parsear AgentLeaseExpirations, no tocar director/workflow/e2e.
Estado: completada
```

```text
ID: SCH-001
Objetivo: Implementar tick puro minimo para programacion con capacidad, gate y agente.
Write-set: scheduler_tick_*_v0.go, *_test.go, docs/*
Contrato: BuildDirectorSchedulerTick v0
Validacion: 2026-05-06, ok; go test -count=1 ./modulos/orquesta-director-scheduler
Bloqueos: no implementar daemon, persistence, runtime, cuotas, HOME ni proveedor.
Estado: completada
```

```text
ID: SCH-002
Objetivo: Cubrir E2E pequeno con dos ticks aplicados contra workflow real hasta AgentRequested.
Write-set: ../orquesta-e2e/*scheduler*_test.go, docs locales
Contrato: BuildDirectorSchedulerTick v0 + orquesta-core-workflow
Validacion: 2026-05-06, ok; go test -count=1 ./modulos/orquesta-e2e -run TestE2EDirectorSchedulerTickProgramaAgenteSinRuntimeRealV0
Bloqueos: runtime real queda fuera.
Estado: completada
```

```text
ID: SCH-003
Objetivo: Ampliar BuildDirectorSchedulerTick v0 para procesar varios SchedulableWorkCandidateV0 por tick sin duplicar capacidad, gate ni agente.
Write-set: orquesta-director-scheduler/*.go, *_test.go, docs locales
Contrato: BuildDirectorSchedulerTick v0
Validacion: 2026-05-06, ok; go test -count=1 .
Bloqueos: no implementar daemon, persistence, DB, runtime, provider, HOME, OAuth ni refs inventadas.
Estado: completada
```
