# Decisiones

## T12: el conector no sustituye evidencia real

Fecha: 2026-05-27.

El conector mantiene responsabilidad de adaptador REST opt-in. Sus tests y el
fake local prueban forma, idempotencia y receipts, pero no prueban aceptacion
real de derivados en OPES temporal.

Decision operativa: T12 se declara bloqueada por falta de entorno temporal y
confirmacion de efectos, no por falta de codigo en el conector. La siguiente
reapertura debe aportar refs de OPES temporal y ejecutar la ruta real acotada.

No hacer:

- leer DB, filesystem, workers ni colas internas de OPES;
- interpretar `worktree_ref` o `branch_ref` como rutas o ramas;
- enviar modelo, runtime, lease o sesiones a OPES;
- relanzar trabajos fake para cerrar una evidencia real.
