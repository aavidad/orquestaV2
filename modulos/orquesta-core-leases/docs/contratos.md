# Contratos locales: orquesta-core-leases

## Contrato candidato: AgentLeasePolicyV0

```text
Tipo: DTO puro
Estado: candidato
Campos:
  - lease_policy_ref
  - launch_timeout_seconds
  - heartbeat_timeout_seconds
  - total_timeout_seconds
  - timeout_action: retry | stop_agent | ask_director | replan_task | alert_only
  - max_retries
  - evidence_refs opcional
Invariantes:
  - No contiene reloj real, PID, runtime, provider, modelo, HOME, OAuth ni DB.
  - Los tiempos son umbrales de politica, no timers ejecutados por el core.
```

## Contrato candidato: AgentHeartbeatReportV0

```text
Tipo: DTO puro observado por adaptador
Estado: candidato
Campos:
  - heartbeat_ref
  - run_ref
  - agent_request_id
  - lease_ref
  - observed_at
  - status: alive | progressing | stalled | stopped | failed
  - progress_report_ref opcional
  - evidence_refs opcional
Invariantes:
  - `observed_at` llega desde fuera; no se calcula en core.
  - Heartbeat de vida y progreso semantico no son lo mismo. Pueden enlazarse por `progress_report_ref`.
  - No contiene PID, process_ref real, HOME, provider, modelo, OAuth, transcript ni log completo.
```

## Contrato candidato: AgentTimeoutAssessmentV0

```text
Tipo: salida pura de evaluacion
Estado: candidato
Campos:
  - assessment_ref
  - run_ref
  - agent_request_id
  - lease_ref
  - now_observed_at
  - decision: continue | ask_director | stop_agent | mark_failed | replan_task
  - reason_code
  - evidence_refs opcional
Invariantes:
  - `now_observed_at` se recibe como input externo.
  - No agenda timers ni lanza goroutines.
  - Si recomienda parar, fallar o replanificar, otro comando durable materializa la accion.
```

## Contrato candidato: AgentLeaseExpiredV0

```text
Tipo: evento candidato puro para core-workflow
Estado: implementado local
Campos:
  - run_ref
  - lease_ref
  - agent_request_id
  - reason_code
  - observed_at
  - recommended_action: retry | ask_director | stop_agent | mark_failed | mark_stopped | replan_task | alert_only
  - evidence_refs opcional
Invariantes:
  - Deriva de `AgentTimeoutAssessmentV0` solo cuando `decision != continue`.
  - `observed_at` llega desde un adaptador/dispatcher; el reducer no consulta el reloj.
  - Conserva refs compactas (`run_ref`, `agent_request_id`, `lease_ref`, `evidence_refs`).
  - La traduccion desde assessment es idempotente y no comparte slices mutables de evidencia.
  - `mark_stopped` y `mark_failed` son senales terminales; no paran procesos ni consultan runtime.
  - No acepta campos ni refs con DB, runtime, provider, HOME, OAuth, modelo ni secretos.
  - Puede derivar en StopAgent, AskDirector o Replan por comandos separados.
```

## Contrato candidato: OutboxDeliveryLeaseV0

```text
Tipo: DTO candidato para adaptadores de persistencia/outbox
Estado: candidato documental cerrado; implementacion productiva pendiente en persistence/outbox por puerto opt-in
Campos:
  - claim_ref
  - message_id
  - target_port
  - claimed_by_ref
  - claimed_at
  - lease_until
  - attempt
Invariantes:
  - No cambia `MarkDispatched`; modela claim/lease antes del ACK terminal.
  - No impone SQL, Redis, locks ni backend concreto.
  - Un dispatcher muerto debe permitir que el mensaje vuelva a ser elegible cuando expire el lease.
  - No pertenece al reducer de workflow y no autoriza DB por defecto.
```
