# Contexto Codex: orquesta-operator-mcp-client

Lee primero este archivo y `README.md`. Despues lee solo los docs locales
necesarios para tu microtarea.

## Alcance

- Adaptador externo opt-in para consumir tools MCP de operador.
- Implementa `OperatorMCPConnectorV0` mediante un cliente MCP generico.
- Mantiene nombres de tools y refs de conectores como configuracion.

## Reglas

- No meter DB, runtime real, HOME, OAuth, tokens, proveedor, modelo ni rutas
  locales en el contrato.
- No cambiar contratos puros de `orquesta-operator-mcp` salvo docs coordinadas.
- No importar `orquesta-mcp` para evitar acoplar el cliente al transporte local.
- Tratar Hermes, OpenClaw u otros operadores como conectores intercambiables por
  MCP y refs opacas.
