# orquesta-core-leases

Responsabilidad: politica pura de leases, heartbeats y timeouts para el nucleo.

Este microproyecto convierte ausencia de progreso en senales durables. No observa procesos ni relojes reales: los adaptadores externos entregan evidencias compactas y este modulo valida la politica.

Incluye:

- lease de launch;
- heartbeat esperado;
- response timeout;
- timeout total;
- accion recomendada: retry, stop, ask_director, replan.

No incluye runtime real, procesos, PID, HOME, OAuth, DB ni proveedor/modelo concreto.

## Arranque

```bash
./arrancar_codex.sh "microtarea concreta"
```
