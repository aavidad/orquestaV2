# Pruebas locales: orquesta-core-concurrency

```text
Caso: workset_claim_compacto
Tipo: unit | contract
Comando: go test -count=1 ./modulos/orquesta-core-concurrency
Evidencia esperada: claim valido normaliza scopes y rechaza HOME, rutas absolutas privadas, DB/provider/model/runtime/OAuth/secretos.
Estado: pasa en CCY-001
```

```text
Caso: conflict_detector_determinista
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-core-concurrency
Evidencia esperada: detecta write-write por solape de write_set, read-write por read_set/write_set cruzados, respeta scopes hermanos y ordena conflictos de forma determinista sin depender de filesystem ni Git real.
Estado: pasa en CCY-002
```

```text
Caso: dependency_evaluator_puro
Tipo: unit | contract
Comando: go test -count=1 ./modulos/orquesta-core-concurrency
Evidencia esperada: detecta dependency_missing directo, dependency_self, dependency_cycle determinista y dependency_not_closed transitivo; emite ready/blocked e issues puros sin scheduler, Git, filesystem real, DB/provider/model/HOME/OAuth/runtime.
Estado: pasa en CCY-003
```

```text
Caso: workflow_claim_mapper_puro
Tipo: unit | contract
Comando: go test -count=1 ./modulos/orquesta-core-concurrency ./modulos/orquesta-core-workflow ./modulos/orquesta-context
Evidencia esperada: `BuildWorksetClaimFromWorkflowTaskV0` construye claim desde `WorkflowTaskV0` y `ContextBundleV0`, normaliza read/write-set declarados, conserva evidence refs compactas y rechaza scopes HOME sin leer filesystem, Git, DB/provider/model/HOME/OAuth/runtime.
Estado: pasa en CCY-004 el 2026-05-06
```

```text
Caso: parallel_group_evaluator_puro
Tipo: unit | contract
Comando: go test -count=1 ./modulos/orquesta-core-concurrency
Evidencia esperada: compone dependencias y conflictos; emite `ParallelGroupPlanV0` con ready_claim_refs, blocked_claim_refs, conflict_refs y summary deterministas sin scheduler, goroutines, locks, Git, filesystem real, DB/provider/model/HOME/OAuth/runtime.
Estado: pasa en CCY-005
```

```text
Caso: concurrency_gate_evaluator_puro
Tipo: unit | contract
Comando: go test -count=1 ./modulos/orquesta-core-concurrency
Evidencia esperada: el gate permite solo claims ready, bloquea claims en conflicto o dependencia, consulta al director para sujetos desconocidos y mantiene salida determinista sin workflow, scheduler, Git, filesystem real, DB/provider/model/HOME/OAuth/runtime.
Estado: pasa en CCY-007
```

```text
Caso: workflow_registra_concurrency_gate_sin_scheduler
Tipo: integration_contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `RecordConcurrencyGate` registra `ConcurrencyGateRecorded`, proyecta `concurrency_gates` y no emite outbox ni lanza agentes.
Estado: validada tras NCW-047 el 2026-05-06
```
