# Contexto para agentes: orquesta-core-replanner

Lee siempre, en este orden:

1. `AGENTS.md`
2. `README.md`
3. `docs/contratos.md`
4. `docs/tareas.md`
5. `docs/pruebas.md`
6. `docs/decisiones.md`
7. `../orquesta-core-workflow/docs/roadmap_cierre_nucleo.md`

Reglas:

- No tocar `orquesta-core-workflow` salvo que una tarea lo indique explicitamente.
- No importar runtime, persistence, capacity, web, CLI ni MCP.
- No hardcodear DB, provider, modelo, HOME, OAuth ni cuotas reales.
- Crear contratos pequenos, testeables y con refs opacas.
- Si necesitas informacion de otro modulo, registra `CONSULTA AL DIRECTOR`.
- Mantener ficheros pequenos; si una funcion crece, dividir antes de seguir.
- Antes de corregir un bug de bucles/progreso, no uses la ruta historica
  `../../docs/reinicio_orquesta_v2/protocolo_anti_bucles.md`: no existe en la
  foto vigente. Clasifica con `../../docs/estado_actual_2026-05-17.md`,
  `../../docs/guia_nucleo_orquestacion_2026-05-17.md`,
  `../../docs/autoprogramacion_orquesta_pendientes_2026-05-23.md` y
  `../../docs/rail_errors_observados_2026-05-23.md`; si falta criterio vivo,
  emite `CONSULTA AL DIRECTOR`.
