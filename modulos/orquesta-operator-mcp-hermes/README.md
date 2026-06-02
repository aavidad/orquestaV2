# orquesta-operator-mcp-hermes

Adaptador externo opt-in para conectar Hermes por HTTP/MCP. Construye un
`OperatorMCPConnectorV0` usando el cliente generico de
`orquesta-operator-mcp-client` y un transporte JSON-RPC remoto.

El paquete:

- llama `tools/call` en un endpoint MCP HTTP;
- envia `Authorization: Bearer` solo si la composicion inyecta token;
- aplica contexto/deadline desde el conector generico;
- conserva nombres de tools y refs opacas configurables;
- no usa CLI, procesos locales, HOME, PATH, DB ni imports de `orquesta-mcp`.

## Prueba

```bash
go test -count=1 ./modulos/orquesta-operator-mcp-hermes
```
