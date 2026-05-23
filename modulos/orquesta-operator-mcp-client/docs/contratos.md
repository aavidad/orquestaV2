# Contratos: orquesta-operator-mcp-client

## OperatorMCPClientConnectorV0

Tipo: adaptador externo opt-in.

Implementa:

- `OperatorMCPStatusPortV0`
- `OperatorMCPBurstPortV0`
- `OperatorMCPOutboxPortV0`
- `OperatorMCPDirectedQueryPortV0`
- `OperatorMCPConnectorV0`

Entrada de configuracion:

- `GenericMCPClientV0`: cliente capaz de invocar un tool MCP por nombre.
- `OperatorMCPClientToolNamesV0`: nombres de tools para status, burst, outbox y
  consulta dirigida.
- `OperatorMCPClientConnectorRefsV0`: refs opacas que el adaptador puede usar
  para sobreescribir las refs entrantes antes de llamar al MCP remoto.

Invariantes:

- El paquete no importa `orquesta-mcp`; solo conoce el contrato publico de
  `orquesta-operator-mcp`.
- El cliente remoto devuelve el envelope compacto `estado`, payload especifico
  y `error_code` publico.
- Si el cliente no existe, falla con `operator_mcp_connector_unavailable`.
- Si el cliente falla sin error publico, falla con `operator_mcp_port_error`.
- Si el MCP remoto devuelve un `error_code` no catalogado como error publico
  de `orquesta-operator-mcp`, el adaptador lo reduce a `operator_mcp_port_error`.
- Hermes, OpenClaw u otro operador real son intercambiables si exponen tools
  MCP compatibles y refs opacas configuradas.
