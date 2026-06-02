# Contexto Codex: orquesta-operator-mcp-hermes

Lee primero este archivo y `README.md`. Despues lee solo los docs locales
necesarios para tu microtarea.

## Alcance

- Adaptador externo opt-in para consumir Hermes por HTTP/MCP.
- Implementa un cliente MCP JSON-RPC compatible con
  `orquesta-operator-mcp-client`.
- Hermes es un operador API/MCP, no CLI.

## Reglas

- No usar CLI, procesos locales, `os/exec`, `CommandPath`, HOME ni PATH.
- No importar `orquesta-mcp`; el transporte local de Orquesta no forma parte
  del cliente remoto.
- No leer DB, filesystem productivo, OAuth, tokens en claro, proveedor/modelo ni
  rutas internas.
- Los errores expuestos al nucleo deben reducirse a errores publicos de
  `orquesta-operator-mcp`.
