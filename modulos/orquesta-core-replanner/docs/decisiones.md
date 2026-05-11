# Decisiones locales: orquesta-core-replanner

```text
Fecha: 2026-05-10
Decision: `changes_requested` puede recomendar `split_task` en el replanner puro.
Motivo: una revision que pide cambios puede detectar que repetir la misma tarea no basta y que el retrabajo debe dividirse en microtareas nuevas.
Alternativas: reservar split solo para `rejected`; pedir siempre al director; crear tareas dentro del replanner.
Impacto: el replanner solo emite la propuesta `split_task`; las microtareas completas, su autorizacion durable y la planificacion posterior viven fuera de este modulo por contratos explicitos.
Estado: aceptada
```

```text
Fecha: 2026-05-06
Decision: La replanificacion empieza fuera de `orquesta-core-workflow` como politica pura y contrato candidato.
Motivo: Evitar que el motor durable vuelva a crecer como scheduler monolitico.
Alternativas: Implementar `ReplanTask` directamente en core-workflow; resolver rework desde director; dejar que cada agente decida.
Impacto: Las propuestas compactas viven aqui; el workflow durable solo promociona comandos/eventos pequenos para resultado de revision, rework y decision aceptada.
Estado: aceptada; promocion minima cerrada en RPL-004/RPL-005
```

```text
Fecha: 2026-05-06
Decision: `ReplanDecisionV0` registra solo la decision aceptada y sus refs de seguimiento, no los efectos.
Motivo: Preparar `RecordReplanDecision -> ReplanDecisionRecorded` sin meter scheduler, runtime ni comandos derivados en el contrato puro.
Alternativas: Promover `ReplanProposalV0` directamente; fusionar decision y request de agente/tarea; admitir logs completos como evidencia.
Impacto: Workflow podra proyectar una decision durable compacta e idempotente por `replan_ref`; split/retry/sustitucion/capacidad/director siguen como comandos/eventos separados.
Estado: aceptada y promocionada como `RecordReplanDecision -> ReplanDecisionRecorded`
```

```text
Fecha: 2026-05-06
Decision: El cierre del bucle de rework debe registrar primero el resultado de revision y despues una intencion de rework/replan.
Motivo: `changes_requested`, `rejected`, trabajo basura y timeout no deben relanzar agentes de forma implicita.
Alternativas: Retry automatico por `Retryable`; reabrir tarea al registrar review; copiar controlplane legacy.
Impacto: `RecordReviewResult`, `RequestRework` y `RecordReplanDecision` quedan promocionados sin runtime ni outbox propia.
Estado: aceptada y cerrada en RPL-005
```

```text
Fecha: 2026-05-06
Decision: Un `AgentFailed` solo se traduce a `replace_agent` o `ask_director` en el replanner puro.
Motivo: Un fallo de lanzamiento no debe convertirse en retry automatico ni reusar el mismo `agent_request_id`; eso fue una fuente probable de bucles en versiones anteriores.
Alternativas: Aceptar `retry_task`; tratar `Retryable=true` como relanzamiento automatico; delegar la decision al runtime; fusionar fallo, capacidad y nuevo agente en un unico contrato.
Impacto: `AgentFailedToReplanProposalV0` mantiene `source_ref=agent_request_id`, exige `replacement_role` para sustitucion y deja capacidad/agente reales para comandos separados del director/workflow.
Estado: aceptada
```
