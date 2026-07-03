# MODULO CONGELADO 2026-07-03: solo fixes correctivos con bug enlazado. Features nuevas requieren decision documentada en docs/ raiz. Motivo: informe pericial P2 (docs/informe_pericial_claude_orquesta_2026-07-03.md)

# Contexto Codex: orquesta-director-scheduler

Lee siempre, en este orden:

1. `AGENTS.md`
2. `README.md`
3. `docs/contratos.md`
4. `docs/tareas.md`
5. `docs/pruebas.md`
6. `docs/decisiones.md`
7. `../orquesta-director/docs/contratos.md`
8. `../orquesta-core-workflow/docs/roadmap_cierre_nucleo.md`

Reglas:

- Este modulo decide ticks puros del director; no es daemon ni worker.
- No aplicar comandos, no persistir, no despachar outbox y no arrancar runtime.
- No seleccionar proveedor, modelo, cuota, HOME, OAuth ni credenciales.
- No inventar refs de capacidad, agentes, tareas ni gates; usar candidates explicitos.
- Usar solo contratos publicos de workflow, director y politicas puras.
- Si hace falta dato de otro modulo, emitir `CONSULTA AL DIRECTOR`.
- Mantener ficheros pequenos; si una funcion crece, dividir antes de seguir.
- Antes de corregir bugs de bucles/progreso, no uses la ruta historica
  `../../docs/reinicio_orquesta_v2/protocolo_anti_bucles.md`: no existe en la
  foto vigente. Clasifica con `../../docs/estado_actual_2026-05-17.md`,
  `../../docs/guia_nucleo_orquestacion_2026-05-17.md`,
  `../../docs/autoprogramacion_orquesta_pendientes_2026-05-23.md` y
  `../../docs/rail_errors_observados_2026-05-23.md`; si falta criterio vivo,
  emite `CONSULTA AL DIRECTOR`.
