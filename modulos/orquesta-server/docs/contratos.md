# Contratos

## ServerRuntimeV0

Entrada:
- `http.Handler` de aplicacion.
- `SupervisorPortV0` opcional.
- `StateStorePortV0` para publicar reenganche.
- Configuracion explicita de direccion, statefile y ritmo de supervision.

Salida:
- `GET /healthz`
- `GET /api/status`
- statefile JSON `orquesta_server_state.v0`

Invariantes:
- no conoce Codex, DB, web, MCP ni modelos;
- no arranca agentes directamente;
- no guarda secretos ni rutas de credenciales en errores;
- cada pulso del supervisor es acotado.

