# Contexto para agentes: orquesta-core-leases

Lee siempre `AGENTS.md`, `README.md`, `docs/contratos.md`, `docs/tareas.md`, `docs/pruebas.md`, `docs/decisiones.md` y `../orquesta-core-workflow/docs/roadmap_cierre_nucleo.md`.

Reglas:

- No tocar runtime real ni persistence real.
- No usar reloj del sistema dentro de contratos puros; las fechas llegan como datos de comando/evento.
- No hardcodear DB, provider, modelo, HOME, OAuth, PID ni procesos.
- Todo debe ser refs opacas y payload compacto.
- Si necesitas ACK/heartbeat real de runtime o persistence, emite `CONSULTA AL DIRECTOR`.
- Antes de corregir un bug, clasificalo segun `../../docs/reinicio_orquesta_v2/protocolo_anti_bucles.md`.
