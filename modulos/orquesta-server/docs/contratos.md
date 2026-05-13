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

Configuracion externa relacionada:
- `ORQUESTA_OPES_BASE_URL`: si existe, el comando servidor crea un conector
  REST OPES y lo inyecta como executor `domain_work` en el stack de aplicacion.
  Si falta, `domain_work` queda apagado por opt-in.

Invariantes:
- no conoce Codex, DB, web, MCP ni modelos;
- no arranca agentes directamente;
- no guarda secretos ni rutas de credenciales en errores;
- cada pulso del supervisor es acotado.
- OPES se cablea desde `cmd/orquesta-server`, no desde el runtime residente.
