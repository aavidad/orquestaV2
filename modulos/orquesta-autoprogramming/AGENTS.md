# Contexto Codex: orquesta-autoprogramming

Lee siempre, en este orden:

1. `AGENTS.md`
2. `README.md`
3. `docs/contratos.md`
4. `docs/tareas.md`
5. `docs/pruebas.md`
6. `docs/decisiones.md`

Reglas:

- Este modulo contiene contratos puros de autoprogramacion y review gate de
  cambios de codigo.
- No importar `orquesta-orchestration-core`: el core puede orquestar, pero no
  debe poseer reglas especificas de programacion.
- No tocar DB, runtime, MCP, HTTP, filesystem, VCS, modelos, HOME, OAuth ni
  cuotas desde aqui.
- Los paths de `write_set` son relativos y seguros; no aceptar rutas absolutas,
  `..`, HOME ni variables.
- Mantener funciones pequenas y ficheros manejables; ampliar con nuevos ficheros
  por responsabilidad.
- Cada cambio debe conservar `go test -count=1 ./modulos/orquesta-autoprogramming`.
