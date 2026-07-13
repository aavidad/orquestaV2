# Pruebas locales: orquesta-core-leases

```text
Caso: lease_policy_compacta
Tipo: unit | contract
Comando: go test -count=1 ./modulos/orquesta-core-leases
Evidencia esperada: politica valida tiempos y acciones; permite vocabulario operativo generico en refs opacas; rechaza refs no opacas y secretos efectivos.
Estado: revalidada el 2026-06-04 tras separar JSON invalido de detalle sensible real.
```

```text
Caso: lease_evaluator_sin_reloj_interno
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-core-leases
Evidencia esperada: evaluador produce la misma decision para los mismos inputs; cubre heartbeat vigente, heartbeat expirado, launch timeout sin heartbeat y stopped/failed terminal; no usa time.Now ni estado global.
Estado: validada para LSE-002 el 2026-05-06
```

```text
Caso: lease_expired_desde_assessment
Tipo: unit | contract
Comando: go test -count=1 ./modulos/orquesta-core-leases
Evidencia esperada: `AgentLeaseExpiredV0` se deriva solo de decisiones no-continue; conserva refs compactas, `observed_at` externo y `recommended_action`; stopped/failed generan senal terminal sin proceso real; la traduccion es idempotente, clona evidencia, trata campos no contratados como JSON invalido y rechaza secretos efectivos como `detalle_prohibido`.
Estado: revalidada el 2026-06-04 tras cierre de rail blando por vocabulario operativo.
```

```text
Caso: workflow_registra_agent_lease_expired_sin_efectos
Tipo: integration_contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-e2e
Evidencia esperada: `RegisterAgentLeaseExpired` registra `AgentLeaseExpired`, proyecta `agent_lease_expirations` y no emite outbox ni cambia stopped/failed.
Estado: validada tras NCW-046 el 2026-05-06
```

```text
Caso: lease_sin_heartbeat_stop_replan_e2e_puro
Tipo: e2e_contract
Comando: go test -count=1 ./modulos/orquesta-core-leases ./modulos/orquesta-core-workflow ./modulos/orquesta-e2e -run 'TestE2ELeaseSinHeartbeat|TestE2EAgentLeaseExpired|TestEvaluateAgentLease|TestAgentLeaseExpiredFromAssessment'
Evidencia esperada: un agente solicitado sin heartbeat supera `launch_timeout_seconds`, produce assessment stop_agent o replan_task, se traduce a `AgentLeaseExpiredV0` y workflow registra `AgentLeaseExpired` sin outbox, parada, fallo ni replan automatico.
Estado: validada para LSE-006 el 2026-05-06
```

```text
Caso: h4_assessment_tipado_hasta_replay
Tipo: integration_contract
Comando: go test -count=1 ./modulos/orquesta-core-leases
Evidencia esperada: `EvaluateAgentLeaseV0` valida el assessment tipado y
`AgentLeaseExpiredFromAssessmentV0` lo transforma en candidato solo si no es
continue. La integración consumidora confirma después bridge -> candidato ->
`AgentLeaseExpired` y replay idempotente, sin deserializar assessments.
Estado: revalidada para H4-008 el 2026-07-13.
```
