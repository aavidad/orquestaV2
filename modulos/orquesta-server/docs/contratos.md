# Contratos

## ServerRuntimeV0

Entrada:
- `http.Handler` de aplicacion.
- `SupervisorPortV0` opcional.
- `StateStorePortV0` para publicar reenganche.
- `StartupCheckPortV0` opcional para autodiagnostico/purga antes de exponer
  HTTP y antes de activar el supervisor.
- `IdleSelfImprovementBlockerPortV0` opcional, implementado por la composicion
  que tambien actua como `SupervisorPortV0`, para veto preventivo de
  automejora residente cuando hay blockers externos conocidos.
- Configuracion explicita de direccion, statefile y ritmo de supervision.

Salida:
- `GET /healthz`: liveness del proceso HTTP; no autoriza trabajo externo ni
  automejora.
- `GET /api/v0/server/readiness`: readiness operativa. Devuelve 200 solo con
  `ready=true`; si el startup cleanup/reconciliacion esta bloqueado devuelve
  503 con estado compacto y evidence refs.
- `GET /api/status` y `GET /api/v0/server/status`: estado/diagnostico publico.
- statefile JSON `orquesta_server_state.v0`

Configuracion externa relacionada:
- `ORQUESTA_SERVER_SUPERVISOR_MAX_TICKS`: numero maximo de ticks internos por
  pulso del supervisor residente. Por defecto se conserva acotado a `1`.
- `ORQUESTA_SERVER_ALLOW_REPEATED_RUNS=true`: permite que un mismo pulso del
  supervisor repita run si la politica de la composicion lo necesita.
- `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_TARGET_QUEUE`: tamano objetivo de cola
  de automejora. Si la cola visible queda por debajo y hay planner inyectado, el
  servidor puede pedir nuevas tareas aunque el supervisor no este idle.
- `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_MAX_REQUESTS`: maximo de tareas nuevas
  por tanda de automejora. El objetivo de cola nunca baja por debajo de este
  valor normalizado.
- `ORQUESTA_SERVER_MAX_RUNS_PER_TICK`,
  `ORQUESTA_SERVER_MAX_EXECUTIONS_PER_TICK`,
  `ORQUESTA_SERVER_DRAIN_MAX_EXTERNAL_WAITS`,
  `ORQUESTA_SERVER_TICK_INTERVAL_MS`,
  `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_TARGET_QUEUE` y
  `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_MAX_REQUESTS` son los nombres
  canonicos de paralelismo residente. No usar alias historicos para la misma
  funcion.
- `ORQUESTA_CODEX_MAX_CONCURRENCY` es, en la composicion Codex, el limite
  global de procesos Codex vivos. No es solo concurrencia interna de goroutines:
  el dispatcher batch consulta la capacidad viva antes de reclamar outbox de
  `LaunchRuntimeAgent` y, si no hay hueco, deja los mensajes pendientes sin
  consumirlos. No se aplica a `StopRuntimeAgent` ni a otros mensajes.
- `ORQUESTA_CODEX_MAX_BATCH_READY` conserva su significado separado: maximo de
  agentes ready que se intentan despachar en una tanda cuando existe capacidad.
- El runtime Codex del servidor debe leer entorno mediante
  `codexRuntimeEnvConfigFromEnvV0`; el panel efectivo y el wiring del stack no
  deben calcular por separado los mismos limites, timeouts o railes.
- `GET /api/v0/server/status` expone `effective_config.settings` con las
  variables canonicas efectivas y `restart_behavior=restart_required` cuando un
  cambio exige reinicio del proceso servidor.
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
- cada pulso del supervisor es acotado por configuracion explicita.
- el runtime residente respeta el `MaxTicks` configurado; no lo pisa
  silenciosamente salvo que llegue vacio o invalido.
- `cmd/orquesta-server` debe leer cada variable operativa por una ruta canonica
  y pasar el valor efectivo al stack; el stack no debe releer limites ya
  normalizados por `ConfigV0`.
- la automejora se puede disparar por idle o por capacidad libre. La ruta por
  capacidad requiere cola visible, sin skips pendientes en el tick y planner
  inyectado; no crea una tarea generica a ciegas.
- `IdleSelfImprovementBlockerPortV0` no es dependencia obligatoria del runtime:
  si la composicion no lo implementa, la comprobacion queda omitida y el resto
  de guardas decide. Si existe, solo puede bloquear automejora por causas
  observables de composicion, debe devolver refs opacas/evidencias compactas y
  no debe preparar runs ni inspeccionar secretos. Un error del puerto se audita
  como `idle_self_improvement_blocker_error` y no equivale a bloqueo.
- OPES y el backend file de `domain_work` se cablean desde `cmd/orquesta-server`,
  no desde el runtime residente.
- el autodiagnostico de arranque entra por puerto: el modulo residente solo
  publica `startup_status`, `startup_ready`, mensaje y evidencias; la purga real
  de estado transitorio pertenece a la composicion inyectada.
- la respuesta de readiness no expone paths, runtime dirs, prompts,
  transcripts, payloads de startup ni secretos; solo estado compacto y
  evidence refs.
