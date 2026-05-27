# Pruebas: orquesta-operator-mcp-client

## Contrato local

- `OPMCP-CLIENT-001`: el conector implementa los cuatro puertos de
  `OperatorMCPConnectorV0`.
- `OPMCP-CLIENT-002`: los nombres de tools son configurables.
- `OPMCP-CLIENT-003`: las refs de conector configuradas sobreescriben refs
  entrantes sin exponer internals.
- `OPMCP-CLIENT-004`: los errores publicos se preservan y los fallos opacos se
  reducen a `operator_mcp_port_error`.
- `OPMCP-CLIENT-005`: los codigos remotos no catalogados no se exponen como
  contrato publico.
- `OPMCP-CLIENT-006`: cada llamada al cliente recibe deadline.
- `OPMCP-CLIENT-007`: timeout y cancelacion se mapean a errores publicos
  `operator_mcp_timeout` y `operator_mcp_cancelled`.

## Comando

```bash
go test -count=1 ./modulos/orquesta-operator-mcp-client
```
