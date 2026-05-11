# Decisiones locales: orquesta-core-leases

```text
Fecha: 2026-05-06
Decision: Los timeouts/leases se modelan como politica pura y eventos observados, no como timers internos del reducer.
Motivo: El core durable debe ser determinista y replayable; el reloj real pertenece a adaptadores externos.
Alternativas: Consultar time.Now en core-workflow; dejar timeouts solo en runtime; no modelar expiraciones.
Impacto: `orquesta-core-leases` define contratos compactos y luego promueve eventos al workflow si pasan pruebas.
Estado: aceptada inicial
```

```text
Fecha: 2026-05-06
Decision: `OutboxDeliveryLeaseV0` queda solo como contrato candidato de adaptador, no como dependencia del core.
Motivo: La recuperacion de dispatchers muertos necesita claim/lease, pero el nucleo no debe conocer SQL, Redis, locks, motor de cola ni `MarkDispatched` productivo.
Alternativas: Reintentar outbox sin lease; bloquear mensajes hasta ACK manual; meter locks de DB en el workflow.
Impacto: persistence/outbox podra implementar lease por puerto opt-in; el core mantiene eventos/outbox compactos y no cambia su reducer.
Estado: aceptada documental para LSE-005
```

```text
Fecha: 2026-05-06
Decision: La prueba de agente sin heartbeat debe cruzar leases y workflow, pero no ejecutar runtime ni replan automatico.
Motivo: El fallo real que queremos evitar es confundir una recomendacion de gobierno con un efecto ya ejecutado. El core solo registra la expiracion y deja StopAgent/Replan como comandos separados.
Alternativas: Probar solo `EvaluateAgentLeaseV0`; ejecutar StopAgent automaticamente al expirar; simular runtime dentro del test de leases.
Impacto: `TestE2ELeaseSinHeartbeatEvaluaYRegistraStopReplanSinRuntimeRealV0` valida stop_agent y replan_task como senales durables sin outbox ni cambio de estado operativo.
Estado: aceptada y validada para LSE-006
```

```text
Fecha: 2026-05-06
Decision: La promocion a workflow registra solo expiracion observada; la accion recomendada no ejecuta StopAgent, fallo ni replan.
Motivo: `orquesta-core-leases` produce una senal pura, pero el core durable debe conservar determinismo y separar observacion de efectos externos.
Alternativas: Parar automaticamente al expirar; marcar failed/stopped desde el evento; dejar la expiracion fuera del historial durable.
Impacto: NCW-046 agrega `AgentLeaseExpired` y `agent_lease_expirations`; acciones posteriores quedan para comandos separados.
Estado: aceptada tras promocion NCW-046
```

```text
Fecha: 2026-05-06
Decision: Heartbeat de vida y progreso semantico se separan.
Motivo: Un agente puede estar vivo pero en bucle, o sin heartbeat pero con ultimo progreso registrado; mezclar ambas senales oculta decisiones.
Alternativas: Usar solo AgentProgressReport; usar solo snapshots runtime; asumir que stop solicitado equivale a stop efectivo.
Impacto: `AgentHeartbeatReportV0` puede referenciar progress, pero `EvaluateAgentLeaseV0` decide con ambos tipos de evidencia.
Estado: aceptada inicial
```
