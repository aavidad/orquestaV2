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

Actualizacion: el harness real opt-in ya existe en
`cmd/orquesta-server/hermes_operator_real_smoke_v0_test.go`. Se activa con
`ORQUESTA_HERMES_REAL_SMOKE_CONFIRM=1` y apunta a `ORQUESTA_HERMES_BASE_URL` de
una instancia Hermes temporal. `ORQUESTA_HERMES_API_KEY` es opcional y debe
tratarse como secreto. `supervised_burst` queda en un test separado y requiere
`ORQUESTA_HERMES_REAL_SMOKE_BURST_CONFIRM=1`; sin esa confirmacion extra, el
harness valida solo descubrimiento local, estado, outbox y consulta dirigida.

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
- harness real opt-in en
  `cmd/orquesta-server/hermes_operator_real_smoke_v0_test.go`

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
- `stack_wiring_test` valida por API HTTP fake: `resources/list`, `tools/list`,
  `status`, `supervised_burst`, `pending_outbox` y `directed_query`.

## Smoke real opt-in

Comando para validar lectura/consulta con Hermes temporal:

```bash
ORQUESTA_HERMES_REAL_SMOKE_CONFIRM=1 \
ORQUESTA_HERMES_ENABLED=1 \
ORQUESTA_HERMES_BASE_URL="https://hermes-temporal.example" \
ORQUESTA_HERMES_MCP_PATH="/mcp" \
ORQUESTA_HERMES_API_KEY="$HERMES_API_KEY" \
ORQUESTA_HERMES_STATUS_CONNECTOR_REF="hermes-status-ref-smoke" \
ORQUESTA_HERMES_OUTBOX_CONNECTOR_REF="hermes-outbox-ref-smoke" \
ORQUESTA_HERMES_QUERY_CONNECTOR_REF="hermes-query-ref-smoke" \
go test -count=1 ./cmd/orquesta-server -run 'TestHermesOperatorRealSmokeStatusOutboxQueryV0' -v
```

Con `ORQUESTA_HERMES_REAL_SMOKE_BURST_CONFIRM=1` y una
`ORQUESTA_HERMES_BURST_CONNECTOR_REF` acotada, ejecutar ademas:

```bash
ORQUESTA_HERMES_REAL_SMOKE_CONFIRM=1 \
ORQUESTA_HERMES_REAL_SMOKE_BURST_CONFIRM=1 \
ORQUESTA_HERMES_ENABLED=1 \
ORQUESTA_HERMES_BASE_URL="https://hermes-temporal.example" \
ORQUESTA_HERMES_MCP_PATH="/mcp" \
ORQUESTA_HERMES_API_KEY="$HERMES_API_KEY" \
ORQUESTA_HERMES_BURST_CONNECTOR_REF="hermes-burst-ref-smoke" \
go test -count=1 ./cmd/orquesta-server -run 'TestHermesOperatorRealSmokeSupervisedBurstV0' -v
```

Sin esa confirmacion, no debe disparar burst.

Cobertura esperada:

- `resources/list` y `tools/list` en el `/mcp` local de Orquesta con Hermes
  inyectado como conector;
- `status`, `pending_outbox` y `directed_query` atravesando Hermes remoto por
  API/MCP `tools/call` y `OperatorMCPConnectorV0`;
- `supervised_burst` solo con confirmacion extra.

El smoke real no esta cerrado todavia. No hay evidencia de ejecucion contra
Hermes productivo ni debe afirmarse como hecho hasta contar con endpoint/token,
instancia aprobada y salida con refs compactas sin secretos ni internals.

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
  remoto y valida API HTTP/MCP completa sin CLI.
- Si el harness real se ejecuta sin confirmacion de burst, debe omitir
  `supervised_burst`; si lo ejecuta, debe usar ref acotada y confirmacion
  `ORQUESTA_HERMES_REAL_SMOKE_BURST_CONFIRM=1`.
- No se debe renombrar ni reutilizar el proveedor Gemini CLI como Hermes.
- No usar Orquesta CLI ni proveedor CLI para cerrar evidencia Hermes; solo API
  publica o MCP JSON-RPC con refs opacas.
