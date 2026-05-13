# Contexto Codex: orquesta-agent-process-registry-memory

Lee siempre, en este orden:

1. `AGENTS.md`
2. `README.md`
3. `docs/contratos.md`
4. `docs/tareas.md`
5. `docs/pruebas.md`
6. `docs/decisiones.md`

Reglas:

- Este modulo es un adaptador de memoria para `orquesta-agent-process-registry`.
- No importar `orquesta-orchestration-core`.
- No leer ni escribir filesystem, DB, red, procesos, HOME, OAuth ni proveedores.
- Mantener el adaptador idempotente: repetir el mismo registro no debe fallar.
- Los errores publicos deben usar `orquesta-agent-process-registry.ErrorV0`.
