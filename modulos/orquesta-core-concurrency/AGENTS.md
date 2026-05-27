# Contexto para agentes: orquesta-core-concurrency

Lee siempre `AGENTS.md`, `README.md`, `docs/contratos.md`, `docs/tareas.md`, `docs/pruebas.md`, `docs/decisiones.md` y `../orquesta-core-workflow/docs/roadmap_cierre_nucleo.md`.

Reglas:

- No ejecutar agentes ni tocar runtime.
- No resolver conflictos leyendo ficheros reales; usar refs/read_set/write_set declarados.
- No hardcodear sistema operativo, Git, DB, provider, modelo, HOME ni OAuth.
- Las decisiones deben ser deterministas y reproducibles por replay.
- Si un conflicto requiere informacion de otro modulo, emitir `CONSULTA AL DIRECTOR`.
- Antes de corregir un bug de bucles/progreso, no uses la ruta historica
  `../../docs/reinicio_orquesta_v2/protocolo_anti_bucles.md`: no existe en la
  foto vigente. Clasifica con `../../docs/estado_actual_2026-05-17.md`,
  `../../docs/guia_nucleo_orquestacion_2026-05-17.md`,
  `../../docs/autoprogramacion_orquesta_pendientes_2026-05-23.md` y
  `../../docs/rail_errors_observados_2026-05-23.md`; si falta criterio vivo,
  emite `CONSULTA AL DIRECTOR`.
