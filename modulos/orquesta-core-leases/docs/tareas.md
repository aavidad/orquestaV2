# Tareas locales: orquesta-core-leases

```text
ID: LSE-000
Objetivo: Crear microproyecto y contratos candidatos de leases/timeouts.
Write-set: orquesta-core-leases/**
Contrato: AgentLeasePolicyV0, AgentLeaseExpiredV0 candidato
Validacion: git diff --check -- modulos/orquesta-core-leases
Bloqueos: ninguno
Estado: completada documental inicial
```

```text
ID: LSE-001
Objetivo: Implementar DTOs puros `AgentLeasePolicyV0` y `AgentHeartbeatReportV0`.
Write-set: lease_policy_*.go, tests, docs/*
Contrato: AgentLeasePolicyV0, AgentHeartbeatReportV0
Validacion: go test -count=1 ./modulos/orquesta-core-leases
Bloqueos: no tocar runtime ni persistence
Estado: completada
```

```text
ID: LSE-002
Objetivo: Implementar evaluador puro `EvaluateAgentLeaseV0`.
Write-set: lease_evaluator_*.go, tests, docs/*
Contrato: AgentLeasePolicyV0 + AgentHeartbeatReportV0 + now externo -> AgentTimeoutAssessmentV0
Validacion: go test -count=1 ./modulos/orquesta-core-leases
Bloqueos: tiempos llegan como input; no usar time.Now
Estado: completada
Resultado: `EvaluateAgentLeaseV0` evalua heartbeat vigente, heartbeat expirado, launch timeout sin heartbeat y reportes stopped/failed sin runtime real.
```

```text
ID: LSE-003
Objetivo: Implementar contrato puro `AgentLeaseExpiredV0` y definir promocion minima a core-workflow.
Write-set: lease_expired_v0.go, lease_expired_v0_test.go, docs/*
Contrato: AgentTimeoutAssessmentV0 -> AgentLeaseExpiredV0 candidato
Validacion: go test -count=1 ./modulos/orquesta-core-leases
Bloqueos: requiere LSE-001/LSE-002
Estado: completada
Resultado: `continue` no produce evento; retry/ask_director/stop_agent/mark_failed/mark_stopped/replan_task/alert_only generan expiracion candidata compacta sin runtime real.
```

```text
ID: LSE-004
Objetivo: Validar que la promocion minima `RegisterAgentLeaseExpired -> AgentLeaseExpired` conserva la frontera de leases.
Write-set: docs/*
Contrato: AgentLeaseExpiredV0 candidato -> AgentLeaseExpired workflow
Validacion: go test -count=1 ./modulos/orquesta-core-leases ./modulos/orquesta-core-workflow ./modulos/orquesta-e2e
Bloqueos: acciones posteriores a lease quedan para StopAgent/AskDirector/replan.
Estado: completada documental tras NCW-046
Resultado: el workflow registra expiracion observada sin outbox, runtime, reloj interno ni efectos operativos.
```

```text
ID: LSE-005
Objetivo: Definir contrato candidato `OutboxDeliveryLeaseV0` para recuperar dispatchers muertos sin hardcodear DB.
Write-set: docs/contratos.md, docs/promocion_core_workflow.md
Contrato: OutboxDeliveryLeaseV0
Validacion: revision documental; no tocar persistence hasta que el contrato sea estable
Bloqueos: la implementacion productiva requiere consenso con orquesta-persistence; el contrato puro no bloquea el nucleo.
Estado: completada documental
Resultado: queda definido como DTO candidato de claim/lease para outbox sin SQL, Redis, locks, motor concreto ni cambio de `MarkDispatched`.
```

```text
ID: LSE-006
Objetivo: Probar E2E controlado de agente sin heartbeat que recomienda stop/replan sin runtime real en el core.
Write-set: modulos/orquesta-e2e/e2e_lease_no_heartbeat_v0_test.go, docs/pruebas.md, docs/decisiones.md
Contrato: AgentLeasePolicyV0
Validacion: go test -count=1 ./modulos/orquesta-core-leases ./modulos/orquesta-core-workflow ./modulos/orquesta-e2e -run 'TestE2ELeaseSinHeartbeat|TestE2EAgentLeaseExpired|TestEvaluateAgentLease|TestAgentLeaseExpiredFromAssessment'
Bloqueos: ninguno
Estado: completada
Resultado: `EvaluateAgentLeaseV0` sin heartbeat produce stop_agent/replan_task; `AgentLeaseExpiredV0` se registra en workflow como expiracion durable sin outbox, parada, fallo ni replan automatico.
```

```text
ID: H4-008
Task ref: task-ref-h4-lease-decoder-reclassify-011
Objetivo: Reclasificar `DecodeAgentTimeoutAssessmentV0` de CONECTAR a RETIRAR.
Write-set: lease_evaluator_v0.go, docs/*
Contrato: los assessments nacen tipados en `EvaluateAgentLeaseV0`, pasan por
`AgentLeaseExpiredFromAssessmentV0` y no tienen frontera JSON productiva.
Validacion: go test -count=1 ./modulos/orquesta-core-leases; inspeccion de la
cadena bridge -> candidate -> RegisterAgentLeaseExpired -> AgentLeaseExpired -> replay.
Estado: completada el 2026-07-13.
Resultado: se retiró el decoder exportado sin callers ni pruebas exclusivas.
`ValidateAgentTimeoutAssessmentV0`, `EvaluateAgentLeaseV0` y
`AgentLeaseExpiredFromAssessmentV0` preservan las garantías tipadas; la
frontera JSON durable permanece en `AgentLeaseExpiredV0`.
```
