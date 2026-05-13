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
Decision: El checkpoint graceful es un puerto previo a StopRun, no una excepcion
en el drainer.
Motivo: El bug historico seria tratar `forced=false` como seguro solo porque no
hay agentes en vuelo en stats. La regla correcta es registrar ACK durable por
`RunControlCheckpointWriterPortV0` antes de pedir `StopRunV0`.
Impacto: `ShutdownServerV0` usa `PrepareAgentShutdownPortV0` y, si falta ACK,
devuelve `waiting_checkpoint` sin ejecutar supervisor ni marcar readiness.
Estado: aceptada local
```
