# Auditoría de variables de entorno que se pisan — 2026-07-04

Autor: Claude (director), a petición del operador tras el incidente T7A
(`umbral pedido 450000, efectivo 100000 defaulted`). Alimenta la TAREA-8
(config canónica) de `docs/instrucciones_director_codex_2026-07-04.md`.

## Método

- Registro central: 236 envs `ORQUESTA_*` en
  `cmd/orquesta-server/server_env_registry_v0.go`.
- Lectura real: `os.Getenv` en cmd/ y modulos/ (fuera de tests).
- La métrica de deuda cuenta 511 nombres en total: la diferencia aproximada
  son 275 envs
  leídas FUERA del registro central (guardian, smokes, scripts) — primera
  conclusión en sí misma.

## Pisadas confirmadas (mismo concepto, varios nombres)

1. **Home de Codex, TRES variantes**: `ORQUESTA_CODEX_CODE_HOME` (la
   canónica que usan los pilotajes), `ORQUESTA_CODEX_HOME`
   (`codex_wave_config_v0.go`) y `CODEX_HOME` sin prefijo
   (`codex_env_v0.go`). Riesgo real: un operador exporta una y el
   componente lee otra → mismo patrón que T7A. Acción: una canónica +
   alias con aviso `deprecated_env_used`.
2. **Base URL de OPES, dos variantes**: `ORQUESTA_OPES_BASE_URL` y
   `OPES_BASE_URL` (`opes_bridge_config.go`). Los scripts de pilotaje ya
   exportan AMBAS "por si acaso" — evidencia de que la duplicidad confunde.
3. **Timeouts con unidades mezcladas**: `ORQUESTA_CODEX_GOAL_TIMEOUT_MS`
   (milisegundos) convivía con `ORQUESTA_CODEX_SMOKE_TIMEOUT_SECONDS`,
   `SMOKE_CLAUDE_GOAL_PROCESS_TIMEOUT_SECONDS` y
   `SMOKE_GEMINI_GOAL_PROCESS_TIMEOUT_SECONDS` (segundos). Trampa de x1000
   esperando a ocurrir. Acción: sufijo de unidad único (_MS o _SECONDS) en
   toda la superficie; alias temporales.
4. **Familia fuera del registro central**: `cmd/orquesta-guardian` lee ~27
   `ORQUESTA_GUARDIAN_*` directamente de `os.Getenv` sin pasar por el
   registro → invisibles para `effective_config` y para el eco de
   configuración; los smokes (`SMOKE_*`) igual. Acción: registrarlas o
   declararlas explícitamente "solo-proceso-hijo".

## Familias con mayor superficie (candidatas a sección de fichero, no envs)

| Familia | nº envs |
|---|---|
| OPES_BRIDGE | 47 |
| CODEX_WAVE | 26 |
| SERVER_IDLE | 21 |
| OPES_REGISTRY | 15 |
| CODEBASE_BROKER | 10 |
| DOMAIN_WORK | 9 |

Estas 6 familias suman ~128 envs (~25% del total): son configuración de
componente, no operativa; deben vivir como secciones del fichero canónico
(TAREA-8.4) y desaparecer como envs.

## Relación con T7A

El umbral `ORQUESTA_AUTOPROGRAMMING_CHECKPOINT_ONLY_HIGH_CONSUMPTION_TOKENS`
está registrado y su lectura es única (sin pisada de nombre); su fallo fue
de PROYECCIÓN (daemon no hereda el env del shell) — clase distinta, cubierta
por TAREA-8.1/8.2 (fichero canónico + guard `config_projection_mismatch`).
Las pisadas de este documento son la otra mitad de la clase: mismo valor,
nombres distintos. Ambas se cierran con la misma medicina: fuente única
tipada + eco por setting + ratchet bidireccional (8.3).

## Acciones entregadas a Codex

- TAREA-8.4 usa esta auditoría como lista inicial de la ola 1:
  consolidar las 3 variantes de codex home, las 2 de OPES base URL y
  normalizar unidades de timeout (alias + aviso, sin romper).
- TAREA-8.3 añade el caso (d): env leída fuera del registro = rojo, salvo
  lista blanca explícita "solo-proceso-hijo".

## Actualización Codex 2026-07-04 tarde

Estado tras revisión de Codex + subagente read-only:

- **Codex home**: corregido parcialmente. `ORQUESTA_CODEX_CODE_HOME` queda
  como canónica para auth/config; `CODEX_HOME` queda como alias legacy
  diagnosticado. Si solo existe `CODEX_HOME`, `effective_config` publica el
  setting canónico con `source=legacy_alias` y diagnóstico
  `deprecated_env_used`. Si `ORQUESTA_CODEX_CODE_HOME` y `CODEX_HOME` existen
  con valores distintos, publica `env_alias_conflict` y gana la canónica. Se
  aclara en el registry que `ORQUESTA_CODEX_HOME` es HOME del proceso, no fuente
  de auth/config.
- **OPES base URL**: corregido parcialmente. `ORQUESTA_OPES_BASE_URL` queda
  como canónica; `OPES_BASE_URL` queda como alias legacy diagnosticado con el
  mismo patrón (`legacy_alias`, `deprecated_env_used`, `env_alias_conflict` si
  difiere).
- **T7A/proyección daemon**: corregido en código aparte. `orquesta-server
  start` ya proyecta `ORQUESTA_AUTOPROGRAMMING_*` al daemon y no bloquea el
  contador `...HIGH_CONSUMPTION_TOKENS` por contener la subcadena `TOKEN`.
- **No cerrado todavía**: fichero canónico `orquesta.config.*` de TAREA-8.1,
  ratchet bidireccional completo de TAREA-8.3 y retirada final de aliases
  legacy tras la ventana de compatibilidad.

Verificación ejecutada:

- `go test -count=1 ./cmd/orquesta-server -run 'TestServerConfigFromEnvV0Diagnostica(OPESBaseURLLegacyAlias|OPESBaseURLPisada|CodexCodeHomeLegacyAlias|CodexCodeHomePisado)V0|TestServerDaemonStartEnvironmentV0ProyectaPoliticaAutoprogramacion|TestServerCodexGoalBackendFromEnvV0TmuxNoArrancaAppServerEnConstruccionV0'`
- `go test -count=1 ./cmd/orquesta-server`
- `scripts/orquesta_metricas_deuda.sh --json` -> `env_vars_orquesta=511`

## Actualización Codex 2026-07-04 tarde 2

Estado tras implementación paralela con subagentes:

- **Orquesta server URL**: `ORQUESTA_SERVER_URL` queda como canónica y
  `ORQUESTA_BASE_URL` como alias legacy. `effective_config` publica el setting
  canónico sensible con `source=legacy_alias` si solo existe el alias,
  `env_alias_conflict` si ambas difieren y `deprecated_env_duplicate` si
  duplican valor. Los wrappers OPES que leían primero `ORQUESTA_BASE_URL` pasan
  por `smoke_orquesta_base_url_from_env_or_runtime`; el helper avisa por stderr
  sin cambiar stdout.
- **Timeouts de smoke**: los smokes Codex pasan a
  `ORQUESTA_CODEX_SMOKE_TIMEOUT_MS`; el alias legacy
  `ORQUESTA_CODEX_SMOKE_TIMEOUT_SECONDS` queda retirado en la ola 2026-07-09.
  Los smokes
  directos Claude/Gemini pasan a `SMOKE_CLAUDE_GOAL_PROCESS_TIMEOUT_MS` y
  `SMOKE_GEMINI_GOAL_PROCESS_TIMEOUT_MS`; el script Claude server acepta
  `SMOKE_CLAUDE_GOAL_PROCESS_SERVER_REQUEST_TIMEOUT_MS` y conserva el alias en
  segundos.
- **Guardian**: las envs `ORQUESTA_GUARDIAN_*` emitidas por el servidor quedan
  clasificadas como `child_process` mediante registry local de guardian; los
  tests impiden emitir claves no registradas y verifican que el runner no hereda
  `ORQUESTA_GUARDIAN_*` del padre.
- **Ratchet**: tras retirar el alias legacy Codex `_SECONDS` y las dos envs
  temporales Telegram operator, el conteo vigente vuelve a
  `env_vars_orquesta=511`. Telegram queda en `telegram_operator.*` dentro de
  `orquesta.config.json`; el token vive en fichero local con permisos
  restrictivos y se publica redactado en `effective_config`.

Verificación adicional ejecutada:

- `go test -count=1 ./cmd/orquesta-server`
- `go test -count=1 ./modulos/orquesta-runtime-claude ./modulos/orquesta-runtime-gemini ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-app-codex-stack`
- `bash -n scripts/lib/smoke_common.sh scripts/smoke_opes_plan_temario_operadores.sh scripts/smoke_opes_derivatives_rest.sh scripts/smoke_goal_first_claude_process_server_real.sh scripts/smoke_codex_real_recursive_tree.sh scripts/smoke_codex_real_operational_wave.sh scripts/smoke_codex_real_required_test_runner.sh`
- Smoke Orquesta temporal: `orquesta-server run` con state/runtime bajo `/tmp`
  publicó `env_alias_conflict` para `ORQUESTA_BASE_URL`, `deprecated_env_used`
  para `CODEX_HOME` y `deprecated_env_used` para `OPES_BASE_URL`, sin valores
  crudos.

## Actualización Codex 2026-07-04 tarde 3

TAREA-8.2 queda cerrada en primer corte para las dos entradas que lanzan trabajo:

- `POST /api/v0/autoprogramming/prepare-run` acepta
  `required_settings:[{key,value}]`.
- `POST /api/v0/apps/director` acepta el mismo campo.
- `cmd/orquesta-server` proyecta `effective_config.settings` al stack como DTO
  MCP pequeño (`key`, `value`, `sensitive`), sin importar `orquesta-server` en
  core ni en goal.
- `orquesta-app-codex-stack` valida la proyección antes de lanzar goal, antes
  de persistir run legacy y antes de encolar. Si falta una clave o el valor
  efectivo difiere, devuelve error público `config_projection_mismatch`.
- El mensaje público no incluye valores esperados/efectivos, solo la clave, para
  no filtrar placeholders sensibles ni secretos.

Archivos principales:

- `modulos/orquesta-mcp/config_projection_v0.go`
- `modulos/orquesta-mcp/autoprogramming_prepare_run_tool_v0.go`
- `modulos/orquesta-mcp/arrancar_director_app_tool_v0.go`
- `modulos/orquesta-app-codex-stack/config_projection_guard_v0.go`
- `modulos/orquesta-app-codex-stack/autoprogramming_prepare_run_mcp_executor_v0.go`
- `modulos/orquesta-app-codex-stack/queued_arrancar_director_v0.go`
- `cmd/orquesta-server/stack.go`
- `modulos/orquesta-i18n-docs/public_error_catalog_v0.go`

Verificación:

- `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-i18n-docs`
- `go test -count=1 ./modulos/orquesta-app-codex-stack`
- `go test -count=1 ./cmd/orquesta-server`
- `go test -count=1 ./...`
- `git diff --check`
- `scripts/orquesta_metricas_deuda.sh --json` -> `env_vars_orquesta=512`
- Smoke Orquesta temporal con `orquesta-server run`: ambos endpoints devuelven
  `400` con `errores_publicos[0].code=config_projection_mismatch` al exigir
  `ORQUESTA_AUTOPROGRAMMING_CHECKPOINT_ONLY_HIGH_CONSUMPTION_TOKENS=999999`
  mientras la proyección efectiva publica `450000`.

Pendiente tras este corte:

- TAREA-8.1: fichero canónico `orquesta.config.*`.
- TAREA-8.3: ratchet bidireccional de variables registradas/leídas.
- Retirada final de aliases legacy (`*_SECONDS`, `CODEX_HOME`, `OPES_BASE_URL`,
  `ORQUESTA_BASE_URL`) tras ventana de compatibilidad.

## Actualización Codex 2026-07-04 tarde 4

TAREA-8.1 queda cerrada en primer corte estrecho para la política que causó T7A:

- `cmd/orquesta-server` lee `orquesta.config.json` desde
  `ORQUESTA_CODEX_PROJECT_WORKDIR` con `schema_version:"orquesta_config.v0"`.
- El primer bloque soportado es `autoprogramming`:
  `checkpoint_only_high_consumption_tokens`,
  `checkpoint_only_max_wait_seconds` y
  `no_checkpoint_warning_max_wait_seconds`.
- La precedencia efectiva es `env explícita > orquesta.config.json > default`.
- `effective_config.settings` marca `source=config_file` cuando el valor viene
  del fichero canónico.
- El valor configurado llega al stack, a `/api/v0/autoprogramming/status` y al
  guard `required_settings`.

Bug detectado durante el smoke y cerrado en el mismo corte:

- El handler REST de `orquesta-app-gateway` no heredaba
  `AutoprogrammingGoalProgressPolicy` aunque el transporte MCP sí la tenía.
  El primer smoke publicó defaults `100000/900/600` con fichero
  `450000/1500/900`.
- Se corrigió pasando `AutoprogrammingGoalProgressPolicy` por
  `orquesta-app-codex-stack -> orquesta-app-gateway ->
  MCPAutoprogrammingStatusToolExecutorV0`.

TAREA-8.3 avanza solo a diagnóstico no bloqueante:

- Nuevo `cmd/orquesta-server/server_env_registry_ast_v0_test.go` detecta
  lecturas `ORQUESTA_*` por AST en producción (`os.Getenv`, `os.LookupEnv` y
  helpers locales con literal o identificador resuelto).
- El diagnóstico actual encuentra 226 lecturas sin `serverEffectiveEnvRegistryV0`
  ni allowlist compacta. No se hace rojo todavía para no romper el árbol por
  deuda histórica.

Verificación:

- `go test -count=1 ./modulos/orquesta-app-gateway ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server -run 'AutoprogrammingStatusAPIRouteV0PublicaGoalProgressPolicy|FicheroCanonico|EnvExplicito|SchemaInvalido|BuildStackFromEnvV0'`
- `go test -count=1 ./modulos/orquesta-server`
- `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-app-codex-stack ./modulos/orquesta-i18n-docs`
- `go test -count=1 ./cmd/orquesta-server`
- `git diff --check`
- `scripts/orquesta_metricas_deuda.sh --json` -> `env_vars_orquesta=512`
- Smoke Orquesta temporal con `orquesta.config.json`: status publicó
  `450000/1500/900`; `required_settings=450000` no disparó
  `config_projection_mismatch`; `required_settings=999999` sí devolvió
  `400/config_projection_mismatch`.

Pendiente tras este corte:

- Ampliar `orquesta.config.json` o `--config` a familias completas
  (`server`, `codex_runtime`, `goal_backend`, `daemon_logs`, OPES/bridges).
- Snapshot de configuración para `orquesta-server start`/daemon si se adopta el
  diseño completo.
- Convertir el diagnóstico AST de TAREA-8.3 en ratchet estricto por fases y
  reducir las 226 lecturas pendientes.

## Actualización Codex 2026-07-04 tarde 5

TAREA-8.1 se amplía más allá del umbral de autoprogramación:

- `orquesta.config.json` soporta ahora secciones `server`, `daemon_logs` y
  `codex_runtime`.
- Campos soportados:
  - `server.addr`, `server.state_dir`, `server.audit_file`,
    `server.audit_disabled`.
  - `daemon_logs.max_bytes`, `daemon_logs.max_rotated_files`,
    `daemon_logs.retention_days`, `daemon_logs.local_raw_enabled`,
    `daemon_logs.local_raw_reason`.
  - `codex_runtime.runtime_work_dir`.
- Las rutas sensibles (`state_dir`, `runtime_work_dir`) se publican en
  `effective_config` como refs (`server-state-dir-configured`,
  `codex-runtime-workdir-configured`), no como paths crudos.
- La precedencia sigue siendo por campo: env explícita > fichero > default.
- `audit_file` conserva la validación existente: debe ser basename `.jsonl`.

TAREA-8.3 mejora el ratchet:

- El test AST deja de ser solo diagnóstico y pasa a ratchet de no-incremento.
- La deuda baja de 226 a 220 lecturas `ORQUESTA_*` sin registry/allowlist; esa
  es la nueva baseline máxima.

Verificación adicional:

- `go test -count=1 ./cmd/orquesta-server -run 'FicheroCanonico|EnvExplicitoGana|AuditFileInvalido|EnvRegistryAST|EnvVars|Ratchet'`
- `go test -count=1 ./cmd/orquesta-server`
- `go test -count=1 ./modulos/orquesta-app-gateway ./modulos/orquesta-app-codex-stack ./modulos/orquesta-mcp ./modulos/orquesta-i18n-docs ./modulos/orquesta-server`
- Smoke Orquesta temporal ampliado: servidor arrancó solo con
  `ORQUESTA_CODEX_PROJECT_WORKDIR` por env y `addr/state/runtime` desde
  `orquesta.config.json`; status publicó `450000/1500`; `required_settings`
  coincidente no produjo mismatch y divergente produjo
  `400/config_projection_mismatch`.

Pendiente:

- Añadir familias restantes: `goal_backend`, límites Codex/agentes, OPES/bridges,
  codebase broker, rails/seguridad y domain work.
- Definir si `orquesta-server start` debe recibir `--config` o un snapshot
  materializado.
- Reducir por fases las 220 lecturas AST pendientes.

## Actualización Codex 2026-07-04 tarde 6

TAREA-8.1 añade otra familia de configuración del servidor:

- Nueva sección `server_http` en `orquesta.config.json`:
  `read_header_timeout_ms`, `read_timeout_ms`, `write_timeout_ms`,
  `idle_timeout_ms`, `max_header_bytes` y `control_body_max_bytes`.
- Nueva sección `server_lifecycle` con `shutdown_grace_ms`.
- `effective_config` publica esos settings con `source=config_file` cuando
  vienen del fichero.
- La precedencia sigue siendo `env explícita > fichero > default`.

Verificación:

- Test focal `LeeHTTPYLifecycle` y ratchets `EnvRegistryAST` /
  `EnvVarsOrquestaRatchet` en `./cmd/orquesta-server`.
- Smoke Orquesta temporal aislado: status público del servidor validó
  `ORQUESTA_SERVER_READ_HEADER_TIMEOUT_MS=1111` y
  `ORQUESTA_SERVER_SHUTDOWN_GRACE_MS=7777` desde `config_file`; el guard
  `required_settings` devolvió `config_projection_mismatch` para un valor
  divergente.

Nota de auditoría:

- No reponer `ORQUESTA_CAPACITY_MODEL_REF` ni `ORQUESTA_CAPACITY_QUOTA_REF` en
  el registro efectivo: su retirada es parte de MEJ-106 y mantiene
  `env_vars_orquesta=512`; reintroducirlas rompe el ratchet.

## Actualización Codex 2026-07-04 tarde 7

TAREA-8.1 añade la familia `worktree_snapshot`:

- `worktree_snapshot.max_files`
- `worktree_snapshot.max_file_bytes`
- `worktree_snapshot.max_total_bytes`

Efecto:

- El stack Codex usa el presupuesto efectivo desde fichero cuando existe.
- `effective_config` publica los tres settings con `source=config_file`.
- Las tres envs quedan en `serverEffectiveEnvRegistryV0`.
- El ratchet AST baja de 220 a 217 lecturas pendientes sin subir
  `env_vars_orquesta` (`512`).

Verificación:

- Test focal `WorktreeSnapshot` y ratchets `EnvRegistryAST` /
  `EnvVarsOrquestaRatchet`.
- Smoke Orquesta temporal aislado:
  `orquesta_config_snapshot_smoke=passed`, con mismatch de `required_settings`
  detectado como `config_projection_mismatch`.

## Actualización Codex 2026-07-04 tarde 8

TAREA-8.1 añade la familia `codex_usage_accounting`:

- `codex_usage_accounting.mode`
- `codex_usage_accounting.log_max_bytes`

Efecto:

- La métrica de uso Codex solo se activa con opt-in explícito
  (`redacted_report` o `runtime_usage_report`).
- `effective_config` publica `ORQUESTA_CODEX_USAGE_ACCOUNTING` y
  `ORQUESTA_CODEX_USAGE_LOG_MAX_BYTES` con `source=config_file`.
- Ambas envs quedan en `serverEffectiveEnvRegistryV0`.
- El ratchet AST baja de 217 a 215 sin subir `env_vars_orquesta` (`512`).

Verificación:

- Test focal `CodexUsage` y ratchets `EnvRegistryAST` /
  `EnvVarsOrquestaRatchet`.
- Smoke Orquesta temporal aislado:
  `orquesta_config_usage_smoke=passed`, con mismatch de `required_settings`
  detectado como `config_projection_mismatch`.

## Actualización Codex 2026-07-04 tarde 9

TAREA-8.1 añade la familia `codebase_broker`:

- `codebase_broker.provider_kind`
- `codebase_broker.external_indexer_enabled`
- `codebase_broker.max_concurrent`
- `codebase_broker.timeout_ms`
- `codebase_broker.state_dir`
- `codebase_broker.watchdog_enabled`
- `codebase_broker.watchdog_stop_orphans`
- `codebase_broker.watchdog_orphan_min_age_seconds`
- `codebase_broker.command`
- `codebase_broker.project_name`

Efecto:

- El broker central y su watchdog ya no dependen solo de envs dispersas.
- El opt-in a `codebase-memory-mcp` sigue centralizado; el smoke usa
  `fallback_rg` para no arrancar indexadores.
- `effective_config` publica `source=config_file` para la familia; rutas,
  comandos y provider quedan redactados en estado publico por politica de
  sensibilidad.
- El ratchet AST se mantiene en 215 y `env_vars_orquesta` no sube.

Incidencias cerradas durante el smoke:

- `BUG-ORQ-20260704-177`: faltaba montar `/api/v0/codebase/query` en HTTP
  directo.
- `BUG-ORQ-20260704-178`: `scope:["."]` no buscaba en la raiz del proyecto.

Verificación:

- Focales `CodeContextBroker`, `CodebaseQueryPublico`, `ScopePunto` y ratchets.
- Smoke Orquesta temporal aislado:
  `effective_config_codebase_broker=passed`, `codebase_status=passed` y
  `codebase_query=passed`.

## Actualización Codex 2026-07-04 tarde 10

TAREA-8.1 añade la familia `domain_work`:

- `domain_work.http_base_url`
- `domain_work.http_domain_ref`
- `domain_work.file_dir`
- `domain_work.file_enabled`
- `domain_work.http_create_path`
- `domain_work.http_submit_path`
- `domain_work.http_timeout_seconds`
- `domain_work.http_egress_mode`
- `domain_work.http_allowed_hosts`
- `domain_work.delivery_ledger_path`

Efecto:

- Se consolidan `ORQUESTA_DOMAIN_WORK_*` y
  `ORQUESTA_DOMAIN_DELIVERY_LEDGER_PATH` en la config canónica por fichero.
- El backend file y HTTP neutral respetan la precedencia
  `env explícita > fichero > default`.
- El ledger de entregas deja de depender solo de env.
- URL/rutas locales/ledger se publican como sensibles en `effective_config`.
- El ratchet AST baja de 215 a 202 sin subir `env_vars_orquesta` (`512`).

Verificación:

- Focal `DomainWork` + ratchets.
- Smoke Orquesta temporal aislado:
  `effective_config_domain_work=passed`, `domain_work_create=passed`,
  `domain_work_snapshot=passed`.

## Actualización Codex 2026-07-04 tarde 11

TAREA-8.1 añade las familias de límites de supervisor, Codex runtime y director:

- `server_supervisor.max_runs_per_tick`
- `server_supervisor.max_executions_per_tick`
- `server_supervisor.queue_limit`
- `server_supervisor.drain_max_dispatches`
- `server_supervisor.drain_max_commands`
- `server_supervisor.drain_max_outbox`
- `server_supervisor.drain_max_external_waits`
- `server_idle_self_improvement.max_requests`
- `server_resident_director.max_actions`
- `codex_runtime.execution_mode`
- `codex_runtime.reasoning_effort`
- `codex_runtime.max_expected_seconds`
- `codex_runtime.max_batch_ready`
- `codex_runtime.max_concurrency`
- `codex_director.wave_agents`
- `codex_director.max_subagents_per_agent`
- `codex_director.recursive_agent_budget`

Efecto:

- Se consolidan límites que antes solo podían entrar por envs dispersas.
- `execution_mode=serial` sigue capando los límites de agentes a `1`, también
  si los números vienen del fichero.
- `effective_config` publica `source=config_file` para todos estos límites.
- Se corrige la falsa sensibilidad de `ORQUESTA_SERVER_DRAIN_MAX_COMMANDS`
  (`BUG-ORQ-20260704-179`).
- El ratchet AST baja de 202 a 201 sin subir `env_vars_orquesta` (`512`).

Verificación:

- Focal de límites/config + ratchets.
- Focal de status público.
- Smoke Orquesta temporal aislado:
  `orquesta_config_limits_smoke=passed`.

## Actualización Codex 2026-07-04 tarde 12

TAREA-8.1 cierra el bloque `--config`/snapshot daemon:

- `run/start/status/stop --config <path>` consumen el fichero canonico
  explicito sin añadir envs nuevas.
- Si no hay `ORQUESTA_CODEX_PROJECT_WORKDIR`, el directorio del fichero
  explicito actua como fallback de proyecto; si la env existe, conserva
  precedencia.
- `start --config` persiste snapshot validado en
  `state/config-snapshots/orquesta.config.json` y arranca el daemon con ese
  snapshot.
- `effective_config` sigue publicando `source=config_file` desde el snapshot.
- Se corrige `BUG-ORQ-20260704-180`: `start` confundia readiness estricta
  degradada por conectores externos con fallo de arranque.

Verificación:

- Focal `ReadinessOK|ConfigPath|DaemonRunArgs|DaemonStart|Status|CommandPublic|EnvRegistryAST|EnvVarsOrquestaRatchet`.
- Smoke real temporal:
  `orquesta_start_config_snapshot_smoke=passed`.

## Actualización Codex 2026-07-04 tarde 13

TAREA-8.1 consolida `goal_backend`:

- `goal_backend.kind` cubre `ORQUESTA_CODEX_GOAL_BACKEND`.
- `goal_backend.timeout_ms` cubre `ORQUESTA_CODEX_GOAL_TIMEOUT_MS`.
- `goal_backend.preflight_timeout_ms` cubre
  `ORQUESTA_CODEX_GOAL_PREFLIGHT_TIMEOUT_MS`.
- `goal_backend.allow_app_server_proxy_diagnostic` cubre
  `ORQUESTA_ALLOW_APP_SERVER_PROXY_DIAGNOSTIC`.

Efecto:

- Launch/observe de Codex app-server, Claude y Gemini usan el backend efectivo
  desde fichero/snapshot.
- La automejora idle goal-first se deriva desde fichero si la env especifica no
  esta definida.
- `external_work_goal_backend_required` deja de aparecer cuando el backend esta
  configurado solo en `orquesta.config.json`.
- `self_programming_only` y cleanup tmux respetan el backend efectivo.
- No se añaden envs nuevas.

Verificación:

- Focal `GoalBackend|GoalFirst|ExternalWorkLegacy|EnvRegistryAST|EnvVarsOrquestaRatchet|ResidualGoFileBudget`.
- Smoke real temporal:
  `orquesta_goal_backend_config_smoke=passed`.

## Actualización Codex 2026-07-05 ola 1 consolidación

TAREA-8.4 queda cerrada para el write-set de esta ola:

- `ORQUESTA_CODEX_CODE_HOME` sigue como canónica de auth/config.
- `ORQUESTA_CODEX_HOME` queda alineada como alias legacy directo en
  `codeHomeDirV0`, igual que ya publicaba `effective_config`; si sólo existe
  ese alias no se transforma en `<valor>/.codex`.
- `CODEX_HOME` conserva compatibilidad como alias legacy posterior.
- `effective_config` ya diagnostica `deprecated_env_used`,
  `deprecated_env_duplicate` o `env_alias_conflict` para
  `ORQUESTA_CODEX_HOME` y `CODEX_HOME` frente a la canónica.
- `ORQUESTA_OPES_BASE_URL`/`OPES_BASE_URL` conservan el patrón canónico + alias
  diagnosticado ya existente.
- El ratchet de unidades de timeout queda cubierto por
  `TestServerEnvRegistryV0TimeoutsDeclaranUnidadEnNombre`; no se añaden
  nombres nuevos y el conteo global se mantiene en `env_vars_orquesta=512`.
- Se retira un centinela de test con prefijo `ORQUESTA_` que no era superficie
  operativa (`CODEX_USAGE_ACCOUNTING_TEST_VALUE`), para que el ratchet mida
  configuración real.
- Los fixtures fake de Claude/Gemini process escriben resultado durable completo
  con `artifact_refs`, `artifact_paths` y `materialized_artifacts`, evitando
  falsos rojos por schema de cierre actual.

Verificación:

- `go test -count=1 ./cmd/orquesta-server -run 'Test(CodeHomeDirV0|Int64EnvOrDefaultV0|ServerConfigFromEnvV0Diagnostica(OPESBaseURLLegacyAlias|OPESBaseURLPisada|CodexCodeHomeLegacyAlias|OrquestaCodexHomeLegacyAlias|CodexCodeHomePisado|OrquestaCodexHomePisado)V0|ServerEnvRegistryV0|EnvVarsOrquestaRatchetMEJ106V0|ServerEnvRegistryASTV0)'`
- `go test -count=1 ./cmd/orquesta-server -run 'TestServerGoalBackendFromEnvV0(ClaudeProcessLanzaYObservaResultado|GeminiProcessLanzaYObservaResultado)V0|TestEnvVarsOrquestaRatchetMEJ106V0'`
- `git diff --check`
- `scripts/orquesta_metricas_deuda.sh --json` -> `env_vars_orquesta=512`
- `go test ./cmd/orquesta-server ./modulos/orquesta-server`

## Actualización Codex 2026-07-04 tarde 14

TAREA-8.1 consolida `rails_security` y `egress_sanitizer`:

- `rails_security.security_mode` cubre `ORQUESTA_SECURITY_MODE`.
- `rails_security.rails_mode` cubre `ORQUESTA_RAILS_MODE`.
- `rails_security.detail_prohibited_rails` cubre
  `ORQUESTA_DETAIL_PROHIBITED_RAILS`.
- `rails_security.detail_prohibited_rails_scope` cubre
  `ORQUESTA_DETAIL_PROHIBITED_RAILS_SCOPE`.
- `egress_sanitizer.*` cubre la familia
  `ORQUESTA_EGRESS_SANITIZER_*`.

Efecto:

- Los rails siguen desactivados/normalizados (`offline` y `off`) aunque el
  fichero contenga valores historicos mas duros; esto preserva la regla vigente
  de no cortar trabajo por heuristicas blandas.
- El sanitizer de egress se cablea desde fichero/snapshot en la composicion
  `cmd/orquesta-server`; rutas, comandos, endpoints y refs sensibles quedan
  redactados en status publico.
- Se corrige `BUG-ORQ-20260704-181`: `start --config` proyectaba rails al
  entorno del daemon y el hijo lo publicaba como `explicit`; ahora el snapshot
  daemon conserva `source=config_file`, sin cambiar la precedencia de env
  manual en `run --config`.
- No se añaden envs nuevas y el ratchet AST queda en 201.

Verificación:

- Focal `Rails|EgressSanitizer|EnvRegistryAST|EnvVarsOrquestaRatchet|ResidualGoFileBudget|DaemonStartEnvironment`.
- Smoke real temporal:
  `orquesta_rails_egress_config_smoke=passed`.

## Actualización Codex 2026-07-04 tarde 15

TAREA-8.1 consolida el subconjunto OPES seguro:

- `opes.base_url` cubre `ORQUESTA_OPES_BASE_URL` manteniendo
  `OPES_BASE_URL` como alias legacy.
- `opes_bridge.*` cubre scope, limites, prioridad, espera residente,
  `dry_run`, `supervise_submitted` y compatibilidad runtime declarativa.
- `opes_registry_finalpkg.*` cubre el productor finalpkg residente en modo
  opt-in/dry-run por defecto.
- `opes_topic_registry.*` cubre herramienta, agente y `force` del registro por
  tema.

Efecto:

- `effective_config` publica `source=config_file` para OPES bridge/registry y
  redacta base URL, rutas de registro, raiz de curso y tool path.
- `cmd/orquesta-server/opes_config_file_v0.go` separa structs OPES para no
  seguir hinchando el parser canonico central.
- Ratchet AST baja de 201 a 191; `env_vars_orquesta` no sube.

Frontera:

- No se migran en este corte confirmaciones de loop/produccion, temporal
  confirm, `allow_unfiltered`, destino evidencia, ledger ni comandos/preflight
  live de speech/remote QA. Se quedan env/script-only porque pueden abrir
  efectos externos, saltar filtros, romper idempotencia o ejecutar shell local.

Verificación:

- Focal `OPESBridgeConfigLeeFicheroCanonico|OPESDrainConfig|OPESBridgeLoopConfig|OPESSpeechSynthesis|ServerConfigFromEnvV0PublicaConfiguracionEfectivaCanonica|ServerConfigFromEnvV0PublicaOPESSpeechSynthesisPreflightRedactado|EnvRegistryAST|EnvVarsOrquestaRatchet|ResidualGoFileBudget`.
- Smoke real temporal:
  `orquesta_opes_config_smoke=passed`.
- `go test -count=1 ./cmd/orquesta-server` verde.

## Actualización Codex 2026-07-04 tarde 16

TAREA-8.4 ola 1 consolida el frente estrecho de alias y unidades:

- `ORQUESTA_CODEX_CODE_HOME` queda como setting canónico de auth/config Codex.
  `ORQUESTA_CODEX_HOME` y `CODEX_HOME` se aceptan como aliases legacy:
  publican el setting canónico con `source=legacy_alias` si no hay canónica,
  emiten `deprecated_env_used`, y emiten `env_alias_conflict` cuando difieren
  de la canónica.
- `ORQUESTA_OPES_BASE_URL` mantiene `OPES_BASE_URL` como alias legacy con el
  patrón ya verificado (`legacy_alias`, `deprecated_env_used` y conflicto si
  ambas difieren).
- El registro efectivo añade ratchet de unidades para timeouts: toda env
  registrada que contenga `TIMEOUT` debe declarar unidad en el nombre
  (`_TIMEOUT_MS`, `_TIMEOUT_SECONDS` o caso no duracional explícito como
  `_TIMEOUT_READY`).
- No se añaden envs nuevas; `scripts/orquesta_metricas_deuda.sh --json`
  permanece en `env_vars_orquesta=512`.

Verificación:

- Focal aliases/registro:
  `go test -count=1 ./cmd/orquesta-server -run 'Diagnostica.*(CodexCodeHome|OrquestaCodexHome|OPESBaseURL)|ServerEnvRegistryV0TimeoutsDeclaranUnidad|ServerEnvRegistryV0TieneMetadata'`.
- Ratchets:
  `go test -count=1 ./cmd/orquesta-server -run 'EnvRegistryAST|EnvVarsOrquestaRatchet'`.
- Suite requerida de cierre:
  `go test ./cmd/orquesta-server ./modulos/orquesta-server`.
- `git diff --check` verde.

## Actualización Codex 2026-07-05 ola 2 familias A config

TAREA-8.4 migra las dos familias grandes de tuning al fichero canónico
`orquesta.config.json`:

- `opes_bridge.*` cubre las 47 envs de la familia
  `ORQUESTA_OPES_BRIDGE_*`: activación/confirmación, `dry_run`, scope,
  secuencia de jobs, límites, timeouts, prioridad, espera residente,
  compatibilidad runtime, `allow_unfiltered`, ledger, confirmación productiva,
  evidencia de destino, speech synthesis y remote QA.
- `codex_wave.*` cubre las 26 envs de la familia `ORQUESTA_CODEX_WAVE_*`:
  agentes, refs, runtime, source code home, modelo, effort, perfil, sandbox,
  approval, args, aislamiento, proyección de credenciales, purge, unmanaged
  launch, `PATH`, tail reason y stop.
- Las envs se conservan como override deprecated con precedencia sobre fichero;
  cuando un valor existe en fichero y la env está seteada, `effective_config`
  publica diagnóstico `deprecated_env_used` con scope `opes_bridge` o
  `codex_wave`.
- Rutas, refs de confirmación, `PATH`, source home, ledger y comandos de
  preflight se publican redactados como refs configuradas en la configuración
  efectiva.
- No se añaden envs nuevas; el objetivo de reducción queda en desplazar uso
  operativo a fichero y dejar las envs como compatibilidad temporal.

Verificación:

- Focal:
  `go test -count=1 ./cmd/orquesta-server -run 'Test(CodexWaveConfigV0|OPESBridgeConfig|OPESDrainConfig|OPESBridgeLoopConfig|OPESSpeechSynthesis|ServerConfigFromEnvV0PublicaOPESSpeechSynthesisPreflightRedactadoV0)'`.
- Requerida:
  `go test ./cmd/orquesta-server ./modulos/orquesta-server`.

## Actualización Codex 2026-07-05 ola 3 familias restantes

TAREA-8.4 cierra la familia `server_idle` y verifica el estado ya presente de
las otras tres familias del paquete:

- `server_idle.*` cubre la familia `ORQUESTA_SERVER_IDLE_*`: 20 envs canónicas
  `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_*` más el alias legacy
  `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER`. Incluye espera, desactivación, refs,
  write-set, required tests, contexto/evidencia, aceptación, goal-first, tests
  congelados, reglas compactas, prioridad, límites de cola y presupuestos
  diarios. `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER` sigue como alias legacy
  específico de `after_seconds`.
- `server_idle_self_improvement` se conserva como sección legacy de lectura
  para no romper ficheros existentes, pero la sección canónica nueva es
  `server_idle`.
- `opes_registry_finalpkg.*` ya cubría las 15 envs de
  `ORQUESTA_OPES_REGISTRY_FINALPKG_*`.
- `codebase_broker.*` ya cubría las 10 envs de
  `ORQUESTA_CODEBASE_BROKER_*`.
- `domain_work.*` ya cubría las 9 envs de `ORQUESTA_DOMAIN_WORK_*` y
  `ORQUESTA_DOMAIN_DELIVERY_LEDGER_PATH`.
- Las envs explícitas siguen ganando como override deprecated; si coexisten
  con valor de fichero, `effective_config` emite `deprecated_env_used` con
  scope `server_idle`, `opes_registry_finalpkg`, `codebase_broker` o
  `domain_work` según familia.

Verificación:

- Focal `TestServerConfigFromEnvV0LeeServerIdleCanonicoYEnvDeprecatedOverrideV0`.
- Requerida de goal:
  `go test ./cmd/orquesta-server ./modulos/orquesta-server`.
