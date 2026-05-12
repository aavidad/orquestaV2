# Contexto Codex: orquesta-run-file

Lee siempre, en este orden:

1. `AGENTS.md`
2. `README.md`
3. `docs/contratos.md`
4. `docs/tareas.md`
5. `docs/pruebas.md`
6. `docs/decisiones.md`

Reglas:

- Este modulo es un adaptador filesystem para puertos existentes.
- No importar `cmd`, DB, red, MCP, runtime real ni proveedores.
- Guardar JSON estructurado con escritura atomica y mutex.
- Mantener paridad de comportamiento con `orquesta-run-memory` e
  `InMemoryAppChangeStoreV0`.
- Mantener ficheros pequenos y pruebas de recreacion de instancia.
