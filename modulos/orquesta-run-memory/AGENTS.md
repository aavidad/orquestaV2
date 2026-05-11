# Contexto Codex: orquesta-run-memory

Lee siempre, en este orden:

1. `AGENTS.md`
2. `README.md`
3. `docs/contratos.md`
4. `docs/tareas.md`
5. `docs/pruebas.md`
6. `docs/decisiones.md`

Reglas:

- Este modulo implementa memoria local para contratos de run-control y run-queue.
- No usar almacenamiento externo, red, MCP, procesos ni adaptadores reales.
- No leer ni escribir ficheros desde codigo de produccion.
- Mantener el store thread-safe y con copias defensivas.
- Mantener ficheros pequenos y pruebas locales de invariantes.
