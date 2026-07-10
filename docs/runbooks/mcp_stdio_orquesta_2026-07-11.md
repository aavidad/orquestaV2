# MCP stdio de Orquesta

`orquesta-server mcp-stdio` expone el mismo dispatcher MCP JSON-RPC que el
endpoint HTTP `/mcp`, pero por entrada y salida estandar. Es una composicion
local: no inicia listener HTTP ni cambia el nucleo.

Cada linea de entrada debe contener un unico objeto JSON-RPC. Cada request con
ID recibe una unica respuesta JSON en stdout; las notificaciones admitidas, como
`notifications/initialized`, no escriben respuesta. Diagnosticos y rechazos se
escriben en stderr, por lo que stdout queda reservado para el protocolo.

Ejemplo local:

```bash
printf '%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26"}}' \
  | go run ./cmd/orquesta-server mcp-stdio
```

El comando acepta las mismas opciones de configuracion que el servidor. No se
debe usar con datos sensibles sin una composicion de credenciales y politica
explicitas. Para transporte residente se mantiene `POST /mcp`.

## Frontera operativa

`mcp-stdio` es solo transporte de cliente MCP: JSON-RPC por stdin/stdout hacia
tools y recursos del dispatcher. No se usa para lanzar, pilotar, observar ni
parar agentes o backends goal. Esas operaciones permanecen en
`app_server_tmux`, con identidad runtime, lease, observacion y recibos
durables. Esta frontera no se amplia hasta completar nucleo y conectores.

Prueba focal:

```bash
go test -count=1 ./cmd/orquesta-server -run 'TestMCPStdioV0'
```
