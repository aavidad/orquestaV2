# Tareas locales: orquesta-core-concurrency

```text
ID: CCY-000
Objetivo: Crear microproyecto y contratos candidatos de concurrencia logica.
Write-set: orquesta-core-concurrency/**
Contrato: WorksetClaimV0, WorksetConflictV0 candidato
Validacion: git diff --check -- modulos/orquesta-core-concurrency
Bloqueos: ninguno
Estado: completada documental inicial
```

```text
ID: CCY-001
Objetivo: Implementar DTO puro `WorksetClaimV0` con normalizacion de scopes.
Write-set: workset_claim_*.go, tests, docs/*
Contrato: WorksetClaimV0, ScopeRefV0
Validacion: go test -count=1 ./modulos/orquesta-core-concurrency
Bloqueos: no leer filesystem real
Estado: completada
```

```text
ID: CCY-002
Objetivo: Implementar detector puro de conflictos entre claims.
Write-set: workset_conflict_*.go, tests, docs/*
Contrato: WorksetClaimV0 -> WorksetConflictV0
Validacion: go test -count=1 ./modulos/orquesta-core-concurrency
Bloqueos: no tocar core-workflow
Estado: completada
```

```text
ID: CCY-003
Objetivo: Implementar dependencias logicas entre claims: missing, self-dependency, ciclos y dependencias no cerradas.
Write-set: dependency_*.go, tests, docs/*
Contrato: WorksetClaimV0 -> ParallelGroupPlanV0
Validacion: go test -count=1 ./modulos/orquesta-core-concurrency
Bloqueos: no tocar core-workflow
Estado: completada
Salida: `EvaluateWorksetDependenciesV0` emite `WorksetDependencyEvaluationV0` puro con ready/blocked e issues reutilizables por `ParallelGroupPlanV0`, sin scheduler real.
```

```text
ID: CCY-004
Objetivo: Mapear `WorkflowTaskV0` y `ContextBundleV0` a `WorksetClaimV0`.
Write-set: workflow_claim_mapper_*.go, tests, docs/*
Contrato: WorkflowTaskV0/ContextBundleV0 -> WorksetClaimV0
Validacion: 2026-05-06, ok, go test -count=1 ./modulos/orquesta-core-concurrency ./modulos/orquesta-core-workflow ./modulos/orquesta-context
Bloqueos: solo contratos publicos
Estado: completada
Salida: `BuildWorksetClaimFromWorkflowTaskV0` construye claims desde tarea/contexto publico, usa read-set/write-set declarados, conserva evidencia compacta y no inspecciona filesystem/Git/runtime.
```

```text
ID: CCY-005
Objetivo: Implementar `EvaluateParallelGroupsV0` para ready/blocked/conflicts sin scheduler real.
Write-set: parallel_groups_*.go, tests, docs/*
Contrato: WorksetClaimV0 -> ParallelGroupPlanV0
Validacion: go test -count=1 ./modulos/orquesta-core-concurrency
Bloqueos: requiere CCY-001/CCY-002/CCY-003
Estado: completada
Salida: `EvaluateParallelGroupsV0` compone `EvaluateWorksetDependenciesV0` y `DetectWorksetConflictsV0`; emite plan puro con ready/blocked/conflict refs y summary determinista, sin elegir ganador ni materializar scheduler.
```

```text
ID: CCY-006
Objetivo: Definir promocion minima a core-workflow para bloquear `RequestAgent` si hay conflicto de write-set.
Write-set: docs/promocion_core_workflow.md
Contrato: WorksetConflict candidate
Validacion: revision documental + CONSULTA AL DIRECTOR si cambia contrato global
Bloqueos: requiere CCY-001/CCY-002
Estado: completada documental inicial
Salida: promocion descrita como gate local antes de `RequestAgent`, sin implementacion en workflow.
```

```text
ID: CCY-007
Objetivo: Entregar contrato local pequeno para futuro comando/evento de gate de concurrencia.
Write-set: concurrency_gate_v0.go, concurrency_gate_v0_test.go, docs/*
Contrato: EvaluateParallelGroupsV0 -> ConcurrencyGateEvaluationV0
Validacion: go test -count=1 ./modulos/orquesta-core-concurrency
Bloqueos: no tocar core-workflow; no crear scheduler real
Estado: completada
Salida: `EvaluateConcurrencyGateV0` emite decision pura `allow_request_agent`, `block_request_agent` o `ask_director` para claims declarados.
```

```text
ID: CCY-008
Objetivo: Confirmar promocion minima `RecordConcurrencyGate -> ConcurrencyGateRecorded` sin mover politica al workflow.
Write-set: docs/*
Contrato: ConcurrencyGateEvaluationV0 -> ConcurrencyGateRecorded workflow
Validacion: go test -count=1 ./modulos/orquesta-core-concurrency ./modulos/orquesta-core-workflow
Bloqueos: director debe decidir si usa el gate antes de `RequestAgent`; no hacer obligatorio en workflow en este corte.
Estado: completada documental tras NCW-047
Salida: workflow registra gate evaluado sin scheduler, outbox ni lanzamiento de agentes.
```
