# Decisiones

```text
Fecha: 2026-05-13
Decision: El apagado de servidor se modela como caso de uso separado, no como senal directa al PID.
Motivo: El servidor no debe cortar agentes vivos. La secuencia correcta es RunControl -> supervisor/drain -> stats -> parada del proceso por borde externo.
Impacto: El modulo coordina puertos existentes y deja la parada fisica del servidor fuera del core.
Estado: aceptada local
```
