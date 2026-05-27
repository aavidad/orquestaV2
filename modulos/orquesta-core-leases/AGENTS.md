# Contexto para agentes: orquesta-core-leases

Lee siempre `AGENTS.md`, `README.md`, `docs/contratos.md`, `docs/tareas.md`, `docs/pruebas.md`, `docs/decisiones.md` y `../orquesta-core-workflow/docs/roadmap_cierre_nucleo.md`.

Reglas:

- No tocar runtime real ni persistence real.
- No usar reloj del sistema dentro de contratos puros; las fechas llegan como datos de comando/evento.
- No hardcodear DB, provider, modelo, HOME, OAuth, PID ni procesos.
- Todo debe ser refs opacas y payload compacto.
- Si necesitas ACK/heartbeat real de runtime o persistence, emite `CONSULTA AL DIRECTOR`.
- Antes de corregir un bug de bucles/progreso, no uses la ruta historica
  `../../docs/reinicio_orquesta_v2/protocolo_anti_bucles.md`: no existe en la
  foto vigente. Clasifica con `../../docs/estado_actual_2026-05-17.md`,
  `../../docs/guia_nucleo_orquestacion_2026-05-17.md`,
  `../../docs/autoprogramacion_orquesta_pendientes_2026-05-23.md` y
  `../../docs/rail_errors_observados_2026-05-23.md`; si falta criterio vivo,
  emite `CONSULTA AL DIRECTOR`.
