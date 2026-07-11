# Runbook: conector operador hexagonal

Objetivo: permitir que una IA operadora invoque estado, burst supervisado,
outbox pendiente y consulta dirigida por MCP sin conocer internals de Orquesta.

## Frontera

- `orquesta-operator-mcp` define DTOs, puertos y un conector simulado local.
- `orquesta-mcp` registra resource/tools y delega en puertos inyectados.
- `orquesta-operator-mcp-client` implementa un conector externo opt-in sobre un
  cliente MCP generico, sin importar el transporte local.
- Hermes, OpenClaw u otro operador real deben vivir como adaptadores externos
  opt-in que implementen `OperatorMCPConnectorV0`.
- Hermes se consume por API/MCP. No usar CLI de proveedor, wrappers de proceso
  local ni rutas internas como sustituto de su conector.

## Uso

1. Descubrir `orquesta.operator.operations.v0`.
2. Invocar tools con refs opacas: `request_ref`, `run_ref`, `subject_ref` y
   `*_connector_ref`.
3. Si falta conector, tratar `operator_mcp_port_unavailable` como modo sin
   adaptador y pedir wiring al director.
4. Para pruebas offline, inyectar `NewOperatorMCPSimulatedConnectorV0` o un
   cliente MCP fake en `NewOperatorMCPClientConnectorV0`.
5. Para Hermes u OpenClaw, configurar nombres de tool y refs de conector en
   `OperatorMCPClientConfigV0`; Orquesta trata esas refs como opacas.

## Hermes API-only

- Usar el transporte MCP JSON-RPC opt-in en `/mcp` o la API publica versionada
  `/api/v0/*` cuando aplique.
- No usar `/api/mcp`, `/api/*` legacy, OpenClaw V1 ni CLI de proveedor para
  cerrar evidencias nuevas.
- Si falta `hermes_operator.base_url` o la credencial opt-in, reportar
  `operator_mcp_connector_unavailable` o dejar el smoke bloqueado; no leer DB,
  filesystem productivo ni outbox interno como alternativa.

Configuracion canonica del conector Hermes de servidor:

- seccion `hermes_operator.*` de `orquesta.config.json`;
- secreto solo por `hermes_operator.api_key_file` confinado al proyecto;
- `ORQUESTA_HERMES_*` queda como override deprecated de compatibilidad.

## Smoke real Hermes API/MCP opt-in

Estado: pendiente hasta tener una instancia Hermes temporal, endpoint y token si
aplica. No usar Hermes productivo para cerrar esta evidencia salvo confirmacion
operativa explicita, alcance acotado y salida publica sin secretos.

El harness existe en
`cmd/orquesta-server/hermes_operator_real_smoke_v0_test.go` y se ejecuta como
prueba Go. Esta guardado por `ORQUESTA_HERMES_REAL_SMOKE_CONFIRM=1`. El harness
actual aun configura el adaptador mediante overrides legacy; debe migrarse al
fichero canonico cuando se reabra la capa de conectores. Hasta entonces solo
acredita compatibilidad, no la ruta operativa canonica.

Comando base:

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

`ORQUESTA_HERMES_API_KEY` es opcional si la instancia temporal no exige token.
Los nombres de tools se toman de `ORQUESTA_HERMES_STATUS_TOOL`,
`ORQUESTA_HERMES_OUTBOX_TOOL`, `ORQUESTA_HERMES_QUERY_TOOL` y
`ORQUESTA_HERMES_BURST_TOOL`; si no se definen, aplican los defaults del
contrato operador. Las refs remotas son opacas y deben venir por
`ORQUESTA_HERMES_*_CONNECTOR_REF`, no por rutas ni ids internos.

Cobertura esperada sin efectos amplios:

- `resources/list` en el `/mcp` local de Orquesta con Hermes inyectado como
  conector;
- `tools/list` en el `/mcp` local de Orquesta;
- `status` por `OperatorMCPConnectorV0`, atravesando Hermes remoto por
  `tools/call`;
- `pending_outbox` por ref opaca, atravesando Hermes remoto por `tools/call`;
- `directed_query` con consulta acotada y respuesta compacta, atravesando
  Hermes remoto por `tools/call`.

`supervised_burst` solo puede entrar en el mismo harness si tambien se exporta
`ORQUESTA_HERMES_REAL_SMOKE_BURST_CONFIRM=1` y existe
`ORQUESTA_HERMES_BURST_CONNECTOR_REF` acotada. Sin esa confirmacion extra, el
smoke real debe omitir el burst y seguir validando descubrimiento, estado,
outbox y consulta dirigida.

Comando de burst acotado:

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

El criterio de cierre real es: todas las llamadas pasan por API/MCP publica,
las respuestas contienen refs/evidencia publica compacta, no aparecen tokens,
URLs privadas, prompts, transcripts, DB, paths locales ni outbox interno en la
salida, y el operador confirma que la instancia usada era temporal o de smoke.

## Validacion

```bash
go test -count=1 ./modulos/orquesta-operator-mcp ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp-client ./modulos/orquesta-operator-mcp-hermes ./cmd/orquesta-server
```

La validacion offline de servidor usa un Hermes HTTP fake y debe cubrir
descubrimiento MCP y las cuatro operaciones de operador. El smoke externo real
solo queda cerrado cuando el harness opt-in anterior se ejecute con
`ORQUESTA_HERMES_REAL_SMOKE_CONFIRM=1`, `ORQUESTA_HERMES_BASE_URL` apunte a una
instancia temporal real y todas las llamadas pasen por API/MCP. Hasta entonces,
la evidencia real Hermes queda pendiente, no ejecutada contra productivo.

El contrato no lee DB, outbox real, runtime, filesystem productivo, HOME,
OAuth, proveedor, modelo, prompts ni transcripts.
