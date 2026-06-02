# Tareas: orquesta-operator-mcp

## Backlog local

- [x] Crear contexto local para futuros agentes.
- [x] Documentar contratos MCP operativos v0.
- [x] Crear contratos Go puros y pruebas de invariantes.
- [x] Anadir listado compacto de outbox pendiente por conector opaco.
- [x] Conectar con `orquesta-mcp` como adaptador puro por puertos inyectados.
- [x] Publicar frontera de registro para servidor/transporte MCP real opt-in.
- [x] Anadir puerto agregado `OperatorMCPConnectorV0`, modo sin conector y
  conector simulado offline.
- [ ] Implementar adaptador externo de servidor/transporte MCP real.
- [ ] Implementar adaptador Hermes API/MCP concreto, no CLI.

## OPMCP-012 - Hermes API/MCP no CLI

Estado: implementada offline/API fake 2026-06-02; smoke real externo pendiente.

Contrato: Hermes debe entrar como adaptador externo de composicion sobre
`OperatorMCPConnectorV0` o cliente MCP/API equivalente. No debe usar CLI,
wrapper shell, `CommandPath`, `HOME`, `PATH` ni proceso local como superficie
principal. La configuracion debe ser opt-in, con endpoint, token redacted,
nombres de tools, refs opacas, timeout, cancelacion y errores publicos.

Implementado: `modulos/orquesta-operator-mcp-hermes` y wiring opt-in
`ORQUESTA_HERMES_*` en `cmd/orquesta-server`, inyectando `OperatorConnector`
para `/mcp`.

Validacion esperada: smoke API-only contra instancia temporal de Hermes:
`resources/list`, `tools/list`, `status`, `pending_outbox`, `supervised_burst`
acotado y `directed_query`, sin DB, sin rutas internas y sin CLI de proveedor.

Validacion ejecutada: `go test -count=1 ./modulos/orquesta-operator-mcp-hermes
./cmd/orquesta-server`.

Bloqueos: falta endpoint/credenciales Hermes reales para smoke externo.
T16/T23/T124/T145 no se reabren salvo regresion: el pendiente restante es smoke
real con Hermes temporal, no los contratos puros ya cerrados.

## OPMCP-011 - Reconciliacion T198 descriptor_source

Estado: completada documental 2026-05-27.

Contrato: `OperatorMCPCapabilitiesV0` sigue siendo fuente canonica del resource
operativo registrado por `orquesta-mcp`; T198 no convierte el operador en
transporte real ni expone DB/runtime.

Validacion: `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp ./modulos/orquesta-observability ./modulos/orquesta-governance ./modulos/orquesta-core ./cmd/orquesta-server`.

Bloqueos: ninguno para documentar el cierre; el servidor/transporte MCP real
continua como adaptador opt-in separado.

Revalidacion: `agent-ref-task-autoprogramming-c3678e9bc306-g01` trata el
pendiente residual como stale y no abre codigo nuevo.

## OPMCP-007 - Conector operador hexagonal

Estado: completada local.

Contrato: `OperatorMCPConnectorV0` agrega estado, burst supervisado, outbox y
consulta dirigida. `orquesta-mcp` acepta el conector agregado como fallback de
los cuatro tools, sin conocer internals ni productos externos.

Validacion: 2026-05-23, ok, `go test -count=1 ./modulos/orquesta-operator-mcp ./modulos/orquesta-mcp`.

Bloqueos: Hermes y OpenClaw quedan como adaptadores externos opt-in; no hay
red productiva, DB, outbox real, runtime, HOME, OAuth, proveedor ni modelo en
este modulo.

## OPMCP-006 - Frontera de transporte MCP opt-in

Estado: completada local.

Contrato: `orquesta-mcp` registra resources/tools operativos en `TransportPortV0`; `orquesta-operator-mcp` sigue aportando solo DTOs y puertos publicos.

Validacion: 2026-05-07, ok, `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp`.

Bloqueos: el servidor real queda pendiente como adaptador externo opt-in; no hay red productiva, DB, outbox, runtime, HOME, OAuth, proveedor ni modelo en este modulo.

## OPMCP-005 - Adaptador MCP puro documentado

Estado: completada local.

Contrato: `orquesta-mcp` expone resource y tools operativas usando `OperatorMCP*PortV0`, sin conocer internals del nucleo.

Validacion: 2026-05-06, ok, `go test -count=1 ./modulos/orquesta-operator-mcp ./modulos/orquesta-mcp`.

Bloqueos: sigue pendiente un servidor/transporte MCP real; este modulo solo define contratos y puertos.
