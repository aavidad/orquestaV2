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

La ruta vigente para agentes OrquestaV2 es el servidor residente y la cola
gobernada, no el wrapper local. `./arrancar_codex.sh` queda reservado a
compatibilidad historica o recuperacion manual con error publico cuando no haya
contrato versionado.
