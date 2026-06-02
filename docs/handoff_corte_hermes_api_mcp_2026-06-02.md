# Handoff: Hermes API/MCP - 2026-06-02

## Estado

Actualizado tras continuar el trabajo: el conector Hermes API/MCP ya esta
implementado y cableado offline en la composition root del servidor. No usa CLI.

Orquesta si funciono como apoyo operativo en esta sesion:

- se usaron subagentes para revisar la frontera Hermes/API y el wiring MCP;
- las revisiones coincidieron en que Hermes debe ser API/MCP, no CLI;
- el servidor ahora puede inyectar Hermes como `OperatorConnector` para los
  tools MCP de operador.

La prueba real API-only de Hermes sigue pendiente porque en esta sesion no habia
servidor local en `127.0.0.1:16543` ni endpoint/token Hermes reales. La
configuracion canonica es `ORQUESTA_HERMES_*`.

## Cambios

Modulo nuevo:

- `modulos/orquesta-operator-mcp-hermes/AGENTS.md`
- `modulos/orquesta-operator-mcp-hermes/README.md`
- `modulos/orquesta-operator-mcp-hermes/hermes_connector_v0.go`
- `modulos/orquesta-operator-mcp-hermes/hermes_client_v0.go`
- `modulos/orquesta-operator-mcp-hermes/hermes_client_v0_test.go`

Wiring servidor:

- `cmd/orquesta-server/hermes_operator_env_v0.go`
- `cmd/orquesta-server/hermes_operator_config_v0.go`
- `cmd/orquesta-server/stack.go`
- `cmd/orquesta-server/effective_config_v0.go`
- tests en `cmd/orquesta-server/config_test.go` y
  `cmd/orquesta-server/stack_wiring_test.go`

Alcance:

- cliente HTTP JSON-RPC MCP para `tools/call`;
- endpoint `/mcp` por defecto;
- token `Authorization: Bearer` opt-in;
- limite de request/response;
- decodificacion de resultado MCP `content[].text` con JSON compacto;
- reduccion de errores remotos a errores publicos de `orquesta-operator-mcp`;
- rechazo de endpoints inseguros como userinfo y `/api/mcp`;
- wrapper `NewHermesOperatorMCPConnectorV0` que compone el cliente Hermes con
  `OperatorMCPClientConnectorV0`.
- `ORQUESTA_HERMES_BASE_URL` y `ORQUESTA_HERMES_API_KEY` quedan como settings
  sensibles/redacted en configuracion efectiva.

## Pruebas

```bash
go test -count=1 ./modulos/orquesta-operator-mcp-hermes
go test -count=1 ./cmd/orquesta-server
```

Validacion final recomendada antes de commit/push:

```bash
git diff --check
go test -count=1 ./modulos/orquesta-operator-mcp-hermes ./modulos/orquesta-operator-mcp-client ./modulos/orquesta-operator-mcp ./modulos/orquesta-mcp ./cmd/orquesta-server
```

## Riesgos

- Falta smoke real contra Hermes externo; el test actual usa `httptest.Server`
  remoto y valida API HTTP/MCP sin CLI.
- No se debe renombrar ni reutilizar el proveedor Gemini CLI como Hermes.
- No usar Orquesta CLI ni proveedor CLI para cerrar evidencia Hermes; solo API
  publica o MCP JSON-RPC con refs opacas.
