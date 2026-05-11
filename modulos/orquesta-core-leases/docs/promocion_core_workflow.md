# Promocion a orquesta-core-workflow

## Corte minimo recomendado

`orquesta-core-workflow` no debe ejecutar timers. Solo debe registrar expiraciones observadas por adaptadores y permitir acciones durables posteriores.

LSE-003 deja la promocion como contrato minimo, sin modificar `orquesta-core-workflow`: `AgentTimeoutAssessmentV0` se traduce a `AgentLeaseExpiredV0` solo si la decision no es `continue`. El campo `observed_at` es copia del `now_observed_at` externo de la assessment y `recommended_action` conserva la decision durable propuesta.

El cierre candidato exige que el traductor sea idempotente para la misma assessment, clone `evidence_refs` y no filtre campos de evaluacion (`assessment_ref`, `now_observed_at`, politica) al evento promovible.

Estado 2026-05-06: la promocion minima `RegisterAgentLeaseExpired -> AgentLeaseExpired` ya quedo implementada en `orquesta-core-workflow` como NCW-046. La accion recomendada sigue sin ejecutarse dentro del core; StopAgent, AskDirector o replan entran por comandos separados.

LSE-006 valida el flujo puro completo para agente sin heartbeat: `EvaluateAgentLeaseV0` detecta launch timeout con `now_observed_at` externo, `AgentLeaseExpiredV0` conserva la recomendacion stop_agent/replan_task y `orquesta-core-workflow` solo registra `AgentLeaseExpired` sin outbox ni efectos operativos.

## Comandos/eventos candidatos

```text
RecordAgentHeartbeat -> AgentHeartbeatRecorded
Uso:
  - Registrar heartbeat compacto opcional si queremos historico durable.
Regla:
  - el reloj viene de fuera; no se consulta desde el reducer.
```

```text
RegisterAgentLeaseExpired -> AgentLeaseExpired
Uso:
  - Registrar que una politica externa/pura detecto expiracion.
Campos minimos:
  - lease_ref
  - run_ref
  - agent_request_id
  - reason_code
  - observed_at
  - recommended_action
  - evidence_refs
Regla:
  - no para ni relanza por si solo; StopAgent/AskDirector/Replan se ejecutan como comandos separados.
  - `mark_stopped` y `mark_failed` registran senal terminal observada; no matan ni inspeccionan procesos.
  - `continue` no se promociona a evento.
```

```text
RegisterAgentStopped -> AgentStopped
Uso:
  - Diferenciar parada solicitada de parada efectiva.
Regla:
  - la confirmacion llega desde runtime/adaptador como ref opaca.
```

## Criterios de promocion

- `EvaluateAgentLeaseV0` probado con `now` externo.
- `AgentLeaseExpiredV0` derivado solo de decisiones no-continue de `AgentTimeoutAssessmentV0`.
- JSON publico limitado a `run_ref`, `agent_request_id`, `lease_ref`, `reason_code`, `observed_at`, `recommended_action` y `evidence_refs`.
- Sin `time.Now`, goroutines, sleep, PID, process_ref real, DB ni runtime en core.
- Sin provider/modelo/HOME/OAuth/secretos en claves ni refs.
- Integracion con `StopAgent`, `AgentFailed` o replan solo por comandos durables separados.
- NCW-046 validado con tests de workflow y E2E minimo.
- LSE-006 valida stop_agent y replan_task sin heartbeat atravesando leases + workflow sin runtime real.
- `OutboxDeliveryLeaseV0` queda fuera del workflow; si se implementa sera en persistence/outbox por puerto opt-in.
