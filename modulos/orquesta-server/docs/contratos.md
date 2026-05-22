# Contratos

## ServerRuntimeV0

Entrada:
- `http.Handler` de aplicacion.
- `SupervisorPortV0` opcional.
- `StateStorePortV0` para publicar reenganche.
- `StartupCheckPortV0` opcional para autodiagnostico/purga antes de exponer
  HTTP y antes de activar el supervisor.
- Configuracion explicita de direccion, statefile y ritmo de supervision.

Salida:
- `GET /healthz`
- `GET /api/status`
- statefile JSON `orquesta_server_state.v0`

Configuracion externa relacionada:
- `ORQUESTA_SERVER_SUPERVISOR_MAX_TICKS`: numero maximo de ticks internos por
  pulso del supervisor residente. Por defecto se conserva acotado a `1`.
- `ORQUESTA_SERVER_ALLOW_REPEATED_RUNS=true`: permite que un mismo pulso del
  supervisor repita run si la politica de la composicion lo necesita.
- `ORQUESTA_OPES_BASE_URL`: si existe, el comando servidor crea un conector
  REST OPES y lo inyecta como executor `domain_work` en el stack de aplicacion.
  Si falta, `domain_work` queda apagado por opt-in.
- `ORQUESTA_DOMAIN_WORK_FILE_ENABLED=1` o `ORQUESTA_DOMAIN_WORK_FILE_DIR`: si
  no hay OPES configurado, `cmd/orquesta-server` crea un conector durable
  file-based de `DomainWorkJobCreatorPortV0` y lo inyecta en
  `/api/v0/domain-work` solo para `create_job`.
- `ORQUESTA_DOMAIN_WORK_HTTP_BASE_URL`: si no hay OPES ni backend file, activa
  el adaptador HTTP neutral `orquesta-domain-work-http` para `create_job` y
  `submit_artifact`. Rutas opcionales:
  `ORQUESTA_DOMAIN_WORK_HTTP_CREATE_PATH` y
  `ORQUESTA_DOMAIN_WORK_HTTP_SUBMIT_PATH`.
- `ORQUESTA_OPES_BASE_URL`, `ORQUESTA_DOMAIN_WORK_FILE_*` y
  `ORQUESTA_DOMAIN_WORK_HTTP_BASE_URL` son excluyentes para evitar backends
  ambiguos de `domain_work`.
- `ORQUESTA_OPES_BRIDGE_ENABLED=1` activa el loop residente OPES desde
  `cmd/orquesta-server`. Requiere `ORQUESTA_OPES_BRIDGE_CONFIRM=1` y un filtro
  seguro: `ORQUESTA_OPES_BRIDGE_JOB_TYPE`, `ORQUESTA_OPES_BRIDGE_JOB_REF` o
  `ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE`.
- `ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE` automatiza pases por tipo de job. El
  loop consulta la secuencia en orden y drena solo el primer tipo con trabajos
  `pending`; el ledger evita relanzar inputs ya enviados y bloquea fases
  posteriores mientras OPES siga exponiendo pendientes de la fase actual.

Invariantes:
- no conoce Codex, DB, web, MCP ni modelos;
- no arranca agentes directamente;
- no guarda secretos ni rutas de credenciales en errores;
- cada pulso del supervisor es acotado.
- el runtime residente respeta el `MaxTicks` configurado; no lo pisa
  silenciosamente salvo que llegue vacio o invalido.
- OPES y el backend file de `domain_work` se cablean desde `cmd/orquesta-server`,
  no desde el runtime residente.
- el autodiagnostico de arranque entra por puerto: el modulo residente solo
  publica `startup_status`, `startup_ready`, mensaje y evidencias; la purga real
  de estado transitorio pertenece a la composicion inyectada.
