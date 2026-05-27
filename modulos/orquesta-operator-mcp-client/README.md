# orquesta-operator-mcp-client

Adaptador opt-in que implementa `OperatorMCPConnectorV0` usando un cliente MCP
generico. Sirve para conectar operadores externos como Hermes u OpenClaw sin
meter esos productos en el nucleo ni en los contratos puros.

El paquete:

- llama a tools MCP configurables para status, burst, outbox y consulta dirigida;
- permite sobreescribir refs de conector por configuracion;
- aplica deadline por llamada y propaga cancelacion de composicion;
- conserva errores publicos estables de `orquesta-operator-mcp`;
- no importa `orquesta-mcp`, red productiva, DB, HOME, OAuth, proveedor ni modelo.

## Prueba

```bash
go test -count=1 ./modulos/orquesta-operator-mcp-client
```
