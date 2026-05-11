# Tareas: orquesta-operator-mcp

## Backlog local

- [x] Crear contexto local para futuros agentes.
- [x] Documentar contratos MCP operativos v0.
- [x] Crear contratos Go puros y pruebas de invariantes.
- [x] Anadir listado compacto de outbox pendiente por conector opaco.
- [x] Conectar con `orquesta-mcp` como adaptador puro por puertos inyectados.
- [x] Publicar frontera de registro para servidor/transporte MCP real opt-in.
- [ ] Implementar adaptador externo de servidor/transporte MCP real.

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
