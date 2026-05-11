# Contexto para agentes: orquesta-runtime-worktree

Lee siempre `AGENTS.md`, `README.md` y `docs/*.md` antes de modificar codigo.

Reglas:

- Este modulo es un conector externo de filesystem, no pertenece al nucleo.
- No introducir DB, proveedor, modelo, HOME, OAuth, tokens ni runtime concreto.
- No enviar rutas absolutas al nucleo; los resultados publicos usan paths
  relativos al proyecto.
- Mantener ficheros pequenos y pruebas focales.
- Si una verificacion necesita politica de otro modulo, emitir `CONSULTA AL DIRECTOR`.
