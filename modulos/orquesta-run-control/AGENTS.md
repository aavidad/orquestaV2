# Contexto Codex: orquesta-run-control

Lee siempre, en este orden:

1. `AGENTS.md`
2. `README.md`
3. `docs/contratos.md`
4. `docs/tareas.md`
5. `docs/pruebas.md`
6. `docs/decisiones.md`

Reglas:

- Este modulo solo define contratos puros para control de runs.
- No tocar workflow core, scheduler interno, persistencia, red ni MCP.
- La entrada externa debe pasar por puertos; este paquete no implementa adaptadores.
- `EvaluateRunControlV0` debe ser determinista y no leer estado externo.
- Mantener ficheros pequenos y pruebas locales de invariantes.
