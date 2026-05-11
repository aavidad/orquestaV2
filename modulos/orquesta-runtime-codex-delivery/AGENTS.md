# AGENTS.md

Contexto local para Codex en `modulos/orquesta-runtime-codex-delivery`.

- Leer este fichero, `README.md` y `docs/*.md` antes de modificar codigo.
- Este modulo es un adaptador exterior: puede depender de `orquestacionnucleoapp`
  y de `orquesta-runtime-codex`, pero el nucleo no debe depender de este modulo.
- No introducir DB, HOME, OAuth, provider, modelo, rutas reales ni transcripts en
  DTOs que salgan hacia el nucleo.
- Mantener ficheros pequenos y tests focales.
