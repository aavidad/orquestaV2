# Decisiones

```text
Fecha: 2026-05-13
Decision: El apagado de servidor se modela como caso de uso separado, no como senal directa al PID.
Motivo: El servidor no debe cortar agentes vivos. La secuencia correcta es RunControl -> supervisor/drain -> stats -> parada del proceso por borde externo.
Impacto: El modulo coordina puertos existentes y deja la parada fisica del servidor fuera del core.
Estado: aceptada local
```

```text
Fecha: 2026-05-13
Decision: El deadline de checkpoint es parte del contrato de shutdown, no un sleep interno.
Motivo: Orquesta debe poder esperar checkpoint cuando hay trabajo real, pero tambien debe cortar un bloqueo si el operador/director declara vencida la ventana cooperativa. Meter sleeps o timeouts ocultos recrearia el bug de procesos colgados sin trazabilidad.
Impacto: `ServerShutdownCommandV0.checkpoint_deadline_at` se evalua contra `occurred_at`; si esta vencido y el checkpoint sigue pendiente, se pide `StopRunV0` con `forced=true` y se expone `forced_after_checkpoint_deadline` para MCP/web/director.
Estado: aceptada local
```

```text
Fecha: 2026-05-13
Decision: El checkpoint graceful es un puerto previo a StopRun, no una excepcion
en el drainer.
Motivo: El bug historico seria tratar `forced=false` como seguro solo porque no
hay agentes en vuelo en stats. La regla correcta es registrar ACK durable por
`RunControlCheckpointWriterPortV0` antes de pedir `StopRunV0`.
Impacto: `ShutdownServerV0` usa primero `PrepareAgentShutdownPortV0`, registra
el ACK durable y solo despues pide `StopRunV0`. Si falta ACK, devuelve
`waiting_checkpoint` sin pedir stop, ejecutar supervisor ni marcar readiness.
Estado: aceptada local
```
