# Contexto Codex: orquesta-run-queue

Lee siempre, en este orden:

1. `AGENTS.md`
2. `README.md`
3. `docs/contratos.md`
4. `docs/pruebas.md`
5. `docs/decisiones.md`

Reglas:

- Este modulo solo define contratos puros para cola global multi-app de runs.
- No tocar workflow core, scheduler interno, persistencia, red ni MCP.
- La entrada externa debe pasar por puertos; este paquete no implementa adaptadores.
- `RankRunCandidatesV0` debe ser determinista y no leer el reloj global.
- Mantener ficheros pequenos y pruebas locales de invariantes.
