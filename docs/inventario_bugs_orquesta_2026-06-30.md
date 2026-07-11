# Inventario vivo de bugs e incidencias Orquesta

Fecha de apertura: 2026-06-30.

Este documento no sustituye a las incidencias detalladas. Es el indice comun
para analizarlas como conjunto y detectar deuda de arquitectura. Cada bug nuevo
observado durante Orquesta, OPES, web, autoprogramacion, MCP, runtime remoto o
conectores debe quedar aqui como fila o enlazado desde aqui.

Regla operativa:

- No tratar un bug como "caso aislado" hasta clasificar si repite patron.
- Si se arregla con codigo, mantener la fila y marcar evidencia de cierre.
- Si solo se documenta, dejar estado `abierto` o `en investigacion`.
- Si el bug viene de OPES u otro consumidor, distinguir dominio consumidor de
  fallo del nucleo Orquesta.
- No abrir nuevos frentes de programacion solo por existir una fila: primero
  decidir si bloquea el cierre actual o queda como backlog agrupado.
- Cada tanda de cierre debe revisar el conjunto: si varios bugs comparten eje,
  fuente de verdad duplicada, contrato implicito o workaround repetido, abrir o
  actualizar una fila arquitectonica. No cerrar el bloque como "bugs sueltos"
  hasta dejar escrita esa lectura.

## Lectura vigente 2026-07-04

Este inventario conserva filas historicas y notas de avance. Para contar bugs
vivos, prevalece esta lectura sobre textos antiguos que digan `sigue abierto`
antes de su cierre posterior:

- Checkpoint de mantenimiento `task-ref-doc-cleanup-inventario-20260704`:
  alcance limitado a marcar supersedencias/estado vigente en este inventario,
  sin borrar historia ni cambiar el conteo de bugs vivos por texto antiguo.
- Avance 2026-07-05: se reduce el riesgo operativo de avisos remotos perdidos
  al introducir `operator_notifications.v0`, dedupe por clave estable y puerto
  Telegram/Hermes send para estados terminales de task/goal/run. Evidencia
  focal: `TestOperatorNotificationV0EnviaTerminalDeduplicado`,
  `TestOperatorNotificationServerV0NotificaGoalYRunTerminalDeduplicado` y
  `TestOperatorNotificationHermesTelegramNotifierV0UsaSendSinAuthCodex`.
  Pendiente externo: wiring productivo del puerto real Hermes send en la
  composicion desplegada y smoke real con Telegram temporal/autorizado.
- `BUG-ORQ-20260701-065` sigue abierto, pero reducido por `67dd7fa9` y el
  avance Codex 2026-07-04 noche 3: shutdown expone `goal_actions` tipadas para
  active work goal-first/backend (`wait_checkpoint`, `forced_stop_requested`,
  `cleanup_required`, `cleanup_requested`, `cleanup_completed`), las conserva en
  MCP/HTTP/status/CLI, las trata como bloqueo operativo si no estan resueltas y
  revalida active work antes de publicar ready. Pendiente: smoke real/corte
  externo amplio y coordinacion automatica completa
  backend/checkpoint/stop/cancel/wait.
- Avance Codex local 2026-07-09 tarde: el smoke real
  `scripts/smoke_goal_first_shutdown_coordination_real.sh` volvio a pasar con
  backend `app_server_tmux` real, status previo visible, shutdown amplio por
  `/api/v0/server/shutdown`, `runs_requested=1`, `runs_stopped=1`,
  `run_control_statuses=stopped`, `shutdown_ready=true` y
  `app_server_tmux_processes_alive=0`. Esto cierra el residual local de nucleo
  en el que un goal-first fuera de cola podia no coordinarse durante shutdown;
  `BUG-065` permanece abierto solo para evidencia externa/remota/stale o
  proveedor lento no reproducida en local.
- Avance 2026-07-04 noche 7: el supervisor residente goal-first completa
  `RunControl` como `stopped` cuando reconcilia un backend Goal ausente por
  cleanup externo y prepara rework causal. Usa razon/idempotencia de
  `external cleanup`, no `forced`, y conserva
  `evidence-ref-run-control-terminal-after-goal-reconcile`. Sigue abierto el
  smoke real amplio de shutdown/backend.
- Avance 2026-07-04 tarde 2: `BUG-165/065` queda reducido en la reconciliacion
  de snapshots `stopped`: `NormalizeStoppedServerSnapshotV0` ya no considera
  limpio un state `stopped` que conserva narrativa vieja de shutdown
  (`shutdown_status=stop_timeout` o `shutdown_ready=false`) aunque no tenga
  active work; lo normaliza a `shutdown_status=stopped`,
  `shutdown_ready=true` y limpia contadores residuales de shutdown
  (`shutdown_runs_*`, agentes y checkpoints), y `orquesta-server status`
  persiste esa correccion.
  Tests:
  `TestNormalizeStoppedServerSnapshotV0ReparaNarrativaShutdownContradictoriaV0`,
  `TestNormalizeStoppedServerSnapshotV0LimpiaContadoresShutdownResidualesV0` y
  `TestStatusServerCommandV0ReconciliaStoppedConShutdownNarrativoStaleV0`.
  Siguen abiertos los residuales amplios de smoke real lento y coordinacion
  automatica completa backend/checkpoint/stop/cancel/wait.
- Avance 2026-07-04 noche 8: `BUG-ORQ-20260701-079` queda reducido en el borde
  app-server: `turn/start` ya no envia solo `packet.Prompt`; inyecta contrato
  preventivo de checkpoint temprano, comandos acotados, `max_text_bytes`,
  `thread_read_max_bytes=256 KiB`, evidencia durable en write-set y final/ACK
  compacto. Sigue abierto para enforcement duro del proveedor/runtime antes de
  ejecutar herramientas y smoke real largo que demuestre checkpoint temprano sin
  consumo gigante previo.
- Avance 2026-07-09e: `BUG-ORQ-20260701-079` queda mas observable sin declarar
  enforcement. `orquesta-runtime-codex-appserver` anade evidencias de
  `turn/start` estructurado:
  `evidence-ref-codex-app-server-turn-start-tool-output-policy-sent`,
  `evidence-ref-codex-app-server-turn-start-tool-output-policy-accepted` y
  `evidence-ref-codex-app-server-turn-start-tool-output-policy-fallback`. Asi el
  siguiente smoke real puede distinguir si el app-server acepto
  `toolOutputPolicy` o si Orquesta tuvo que caer a contrato textual legacy.
  Verificado con
  `go test -count=1 ./modulos/orquesta-runtime-codex-appserver -run 'TestServerCodexAppServerGoalBackendV0TurnStart|TestServerCodexAppServerTurnStartParamsV0'`
  y `go test -count=1 ./modulos/orquesta-runtime-codex-appserver`. Sigue abierto
  hasta probar que el proveedor/app-server aplica el limite antes de que una
  herramienta genere stdout gigante.
- Avance 2026-07-09f: `BUG-ORQ-20260701-066` queda cerrado en alcance
  local/fake-residente. La evidencia vigente combina el smoke OPES lifecycle
  fake-residente con 24/24 work kinds, `settlement_status=settled_final` y sin
  procesos residuales, mas el smoke real local de shutdown goal-first con
  backend `app_server_tmux`, `runs_requested=1`, `runs_stopped=1`,
  `run_control_statuses=stopped` y cleanup completo. No cierra despliegue
  remoto, OPES temporal/preproduccion ni proveedor real/productivo; esos quedan
  como residual externo de evidencia, no como bug local abierto.
- Avance 2026-07-09g: `BUG-ORQ-20260701-079` queda mas acotado por una ola real
  de Orquesta (`codex-launch-director-wave`,
  `wave_ref=codex-bug079-smoke-policy-real-20260709`). El smoke
  `scripts/smoke_goal_first_app_server_real.sh` ahora falla si va a dar verde
  sin evidencia de transporte `toolOutputPolicy` enviada y aceptada o fallback;
  publica `tool_output_policy_transport=accepted|fallback|sent_without_accept_or_fallback|missing`.
  No cierra enforcement pre-tool del proveedor.
- Avance 2026-07-09h: smoke real app-server confirma que el proveedor actual
  acepta `toolOutputPolicy`. Primero se ejecuto el smoke normal con
  `run_ref=run-spec-smoke-goal-first-req-smoke-goal-first-4bde3b2d0c8fc9f42c555b714a3521f0`;
  no cerro en 50 polls y no se cuenta como verde, pero el shutdown publico
  evidencias `evidence-ref-codex-app-server-turn-start-tool-output-policy-sent`
  y `...-accepted`, y no quedaron procesos. Despues el smoke alto consumo
  `scripts/smoke_goal_first_checkpoint_only_high_consumption_real.sh` paso con
  `run_ref=run-spec-smoke-goal-first-bug088-req-smoke-goal-first-bug088-a93fa78e27925c5262c0b25f2445dc0f`,
  `tool_output_policy_transport=accepted`, `smoke_goal_first_high_consumption_real=ok`,
  `bug088_path=second_artifact_or_partial_artifacts`, `tokens_used=8392`,
  checkpoint durable y `app_server_tmux_processes_alive=0`. Evidencia retenida:
  `/tmp/orquesta-smokes-codex/orquesta-goal-first-app-server.CZC8Ek`.
  `BUG-079` sigue abierto solo para el smoke adversarial/largo que demuestre
  enforcement pre-tool ante stdout gigante; ya no queda duda local de transporte.
- `BUG-ORQ-20260709-199` queda cerrado: la misma ola real de Orquesta entrego
  cambios y `codex_last_message.txt`, pero no dejo `codex_process_done_v0`
  porque el marcador dependia de un goroutine `cmd.Wait()` en el CLI lanzador,
  que ya habia salido. Cierre: el wrapper `orquesta_codex_exec_v0.sh` escribe
  por si mismo `codex_process_done_v0=completed|failed` antes de salir, tambien
  en trap de senal. Reproduccion post-fix:
  `wave=codex-process-done-repro-20260709T135205Z`, runtime
  `/home/alberto/Trabajo/runtime/codex-process-done-repro-20260709T135205Z`,
  `agent-01/codex_process_done_v0=completed`. Esto convierte el fin de agente
  en evidencia durable del proceso que realmente vive.
- `BUG-ORQ-20260709-200` queda abierto: se creo por Orquesta el smoke
  adversarial `scripts/smoke_goal_first_tool_output_policy_adversarial_real.sh`
  para `BUG-079`, pero dos ejecuciones reales quedaron `running/observe_later`
  hasta timeout tras materializar solo checkpoint temprano. Evidencias:
  `/tmp/orquesta-smokes-codex/orquesta-goal-first-app-server.YAf0mK`
  (`run_ref=run-spec-smoke-goal-first-bug079-tool-output-policy-req-smoke-goal-first-bug079-tool-output-policy-42cbaac6e57fce5425a51a63a`)
  y `/tmp/orquesta-smokes-codex/orquesta-goal-first-app-server.6wB72G`
  (`run_ref=run-spec-smoke-goal-first-bug079-tool-output-policy-req-smoke-goal-first-bug079-tool-output-policy-5e97d87c7c9d9492c24b03907`).
  En ambos casos hubo `toolOutputPolicy accepted`, sin
  `thread-output-sanitized` ni `thread_read_response_too_large`, cleanup final
  sin procesos y servidor `stopped/shutdown_ready=true`. Pendiente: hacer que
  el harness adversarial fuerce/programe de forma fiable la ejecucion del probe
  stdout gigante o bajar esta prueba a un nivel app-server/protocolo directo.
- Avance 2026-07-09j: `BUG-ORQ-20260709-200` queda reducido por dos olas
  paralelas de Orquesta:
  `codex-core-bug200-harness-20260709T141956Z` y
  `codex-core-bug079-protocol-20260709T141956Z`. Ambas acabaron con
  `codex_process_done_v0=completed`. El harness adversarial ya no puede quedar
  como OK ni ambiguo si solo hay checkpoint: exige
  `generated-apps/smoke-goal-first-bug079-tool-output-policy/bug079-tool-output-policy/probe_result.txt`, publica
  `reason=bug200_probe_not_executed` cuando hay checkpoint sin probe, y
  `reason=no_probe_result` cuando falta resultado valido. Ademas se anadio
  cobertura app-server para que `toolOutputPolicy` solo caiga a fallback ante
  error de schema, no ante errores internos que solo mencionen policy.
  Verificado localmente con `bash -n`, `go test -count=1 ./cmd/orquesta-server
  -run 'TestSmokeGoalFirst'`, `go test -count=1
  ./modulos/orquesta-runtime-codex-appserver -run
  'TestServerCodexAppServerGoalBackendV0TurnStart|TestServerCodexAppServerTurnStartParamsV0'`
- Cierre local 2026-07-09k: `BUG-ORQ-20260709-200` queda cerrado como bug de
  harness/contrato local. El smoke adversarial incorpora un self-test
  determinista (`SMOKE_GOAL_FIRST_BUG079_GUARD_SELFTEST=1`) que prueba
  dos fixtures sin servidor ni Codex: checkpoint-only falla con
  `reason=bug200_probe_not_executed`, y `probe_result.txt` valido pasa solo si
  declara exit code, `probe_stdout.py`, `200000`,
  `BUG079_STDOUT_PROBE` y `executions=1`. Ademas el runtime app-server
  cubre el caso `toolOutputPolicy` aceptada + `thread/read` gigante: la
  observacion queda `blocked` con
  `codex_app_server_thread_read_response_too_large` y evidencia especifica
  `evidence-ref-codex-app-server-thread-read-response-too-large`. Evidencia:
  `SMOKE_GOAL_FIRST_BUG079_GUARD_SELFTEST=1
  scripts/smoke_goal_first_app_server_real.sh`, `bash -n
  scripts/smoke_goal_first_app_server_real.sh
  scripts/smoke_goal_first_tool_output_policy_adversarial_real.sh`,
  `go test -count=1 ./modulos/orquesta-runtime-codex-appserver -run
  'TestServerCodexAppServerGoalBackendV0(PolicyAceptadaYThreadReadGiganteBloquea|TurnStartToolOutputPolicy|ObservaThreadReadGigante|LanzaThreadGoalYTurnMigrado)|TestCodexAppServer(WebSocketThreadReadResponseBudget|CommandProtocolThreadReadResponseBudget)'`
  y `go test -count=1 ./cmd/orquesta-server -run
  'TestSmokeGoalFirstToolOutputPolicyAdversarial|TestSmokeGoalFirstAppServerRealExponeToolOutputPolicyTransport'`;
  `git diff --check`.
  Residual vivo: no cierra `BUG-ORQ-20260701-079`; queda probar o corregir el
  enforcement pre-tool real del proveedor/app-server ante stdout gigante.
- Avance real 2026-07-09l: el smoke adversarial de `BUG-079` ya pasa contra
  `codex app-server` real en modo `app_server_tmux` tras corregir el contrato
  de write-set. Ejecucion:
  `run_ref=run-spec-smoke-goal-first-bug079-tool-output-policy-req-smoke-goal-first-bug079-tool-output-policy-e4e76c6dacea00864e51d01ef`,
  `external_goal_ref=019f47b3-3b98-7e30-8fc1-6704b4f5bdfb`, evidencia
  retenida en `/tmp/orquesta-goal-first-app-server.ZHXRWf`. Resultado:
  `tool_output_policy_transport=accepted`,
  `bug079_probe_path=generated-apps/smoke-goal-first-bug079-tool-output-policy/bug079-tool-output-policy/probe_result.txt`,
  `bug079_probe_result=executed`,
  `smoke_goal_first_tool_output_policy_adversarial_real=ok`, forced-stop
  gobernado con `run_control_status=stopped`,
  `run_control_goal_status_after=blocked`,
  `observe_after_forced_stop_goal_status=blocked`,
  `autoprogramming_status_after_forced_stop_not_running=true` y
  `app_server_tmux_processes_alive=0`. `rg -a 'X{100,}'` sobre runtime,
  proyecto, JSON y logs del smoke no encontro salida gigante cruda. Lectura:
  esto cierra el smoke operativo real y confirma que el proveedor obedecio el
  contrato ejecutando el probe con stdout redirigido; no prueba todavia un cap
  duro si una herramienta vuelca stdout crudo sin redireccion, por lo que
  `BUG-079` mantiene solo ese residual de frontera proveedor.
  `BUG-ORQ-20260709-207` queda abierto como residual runtime/proveedor: durante
  el forced-stop del smoke real anterior, el log de app-server retuvo
  `Node.js[...] ResetStdio` / `Assertion failed` en
  `/tmp/orquesta-goal-first-app-server.ZHXRWf/runtime/goal-srv/orquesta-goal-ee58290e72150b73.log`.
  No hubo fuga funcional: `runs/control` paro el goal, `observe` posterior lo
  publico `blocked`, `status` no lo vio running y no quedaron procesos.
  Accion: reproducir/acotar si es crash esperado por terminacion forzada de
  tmux/app-server o si hay que limpiar el apagado cooperativo del adaptador.
  Cierre 2026-07-09m: el adaptador `app_server_tmux` arranca ahora el
  app-server con stdin desacoplado del PTY de tmux mediante una FIFO propia
  `stdin.pipe`, mantiene una fase de `SIGTERM` cooperativo sobre procesos
  propios antes de `kill-session`, y limpia socket/FIFO/owner marker al cerrar.
  El intento intermedio de usar `< /dev/null` se descarto porque el proveedor
  salia con `codex_app_server_tmux_session_exited`. Evidencia real:
  `scripts/smoke_goal_first_forced_stop_backend_real.sh` en modo real opt-in
  con directorio retenido termino con `smoke_goal_first_forced_stop_backend_real=ok`,
  `run_control_status=stopped`, `run_control_goal_status_after=blocked` y
  `app_server_tmux_processes_alive=0`; evidencia retenida en
  `/tmp/orquesta-goal-first-app-server.oj0Aat`. `rg -a
  'ResetStdio|Assertion failed'` sobre `runtime/goal-srv` y logs del smoke no
  encontro coincidencias y solo quedo el log del app-server, sin FIFO residual.
  Avance 2026-07-09n: la verificacion de `BUG-079`/`BUG-200` deja de depender
  solo de que un agente real decida ejecutar el probe. Se anadio una prueba
  determinista con servidor WebSocket Unix falso que recibe el JSON real de
  `turn/start`, comprueba `toolOutputPolicy` y despues inyecta un frame gigante
  en `thread/read`; Orquesta bloquea el goal con
  `codex_app_server_thread_read_response_too_large` y evidencia especifica.
  Esto cierra la cobertura de protocolo/ingesta sin LLM, pero no el residual de
  proveedor: un stdout crudo generado dentro del runtime antes de que exista
  `thread/read` sigue siendo frontera externa de `BUG-079`.
  Nota operativa: el agente Orquesta del harness no pudo
  ejecutar su `go test` dentro del CODEX_HOME aislado porque Go intento
  descargar `golang.org/x/text v0.38.0` y la sandbox bloqueo DNS; el mismo test
  paso en el entorno local con cache normal. `BUG-079` sigue abierto por
  enforcement pre-tool real del proveedor/app-server.
- `BUG-ORQ-20260709-201` queda cerrado en alcance local: el deploy del servidor
  dependia de pasos manuales para sync/build/swap/start/verificacion y podia
  repetir el fallo de arrancar sin config canonica o con binario no identificado.
  Cierre: `scripts/orquesta_server_deploy.sh` hace fast-forward-only, build
  desde arbol, sha256, backup/swap atomico, arranque por `orquesta_server_ctl.sh`,
  config canonica obligatoria opt-in y receipt JSON; `scripts/test_orquesta_server_deploy.sh`
  cubre success, config ausente, no-fast-forward e identidad/hash mismatch; el
  guard Go limita copia/arranque gestionado a `ctl`/`deploy`. No ejecuta remoto
  real ni cierra la verificacion productiva externa.
- `BUG-ORQ-20260709-202` queda cerrado en harness local: el smoke recursivo
  `scripts/smoke_codex_director_recursive_wave.sh` lanzaba
  `codex-launch-director-wave` sin el breakglass auditado que exige el guard
  vigente de lanzamientos no gestionados, por lo que fallaba con
  `unmanaged_launch_blocked` aunque el cierre recursivo offline estuviera verde;
  despues aparecieron el guard equivalente de purga
  `runtime_purge_confirmation_required` y aserciones stale del harness contra
  `registry_path` publico y `reasoning_effort=high`.
  Cierre: el smoke declara `--allow-unmanaged-launch`,
  `--unmanaged-launch-reason`, `--confirm-unmanaged-launch=recursive-wave-smoke`
  y `--confirm-purge-runtime=recursive-wave-smoke`; valida rutas internas desde
  el registry local derivado de `wave_ref` y permite `medium`/`high` sin aceptar
  `xhigh`.
- `BUG-ORQ-20260709-203` queda cerrado en smokes de conectores locales:
  `scripts/smoke_external_domain_fake_real.sh` esperaba todavia
  `domain_work_submitter_no_disponible` aunque el backend file-based ya activa
  submitter local y `DomainDelivery`; `scripts/smoke_external_domain_non_opes_real.sh`
  configuraba `ORQUESTA_DOMAIN_WORK_HTTP_BASE_URL` sin declarar la politica de
  egress exigida por el adaptador HTTP; ambos arrancaban el supervisor legacy
  sin `ORQUESTA_AUTOPROGRAMMING_LEGACY_DIRECTOR_LOOP=true`. Cierre: el smoke
  file valida receipt `accepted` y snapshot `domain_work_artifacts_v0.json`; el
  smoke HTTP exporta `ORQUESTA_DOMAIN_WORK_HTTP_EGRESS_MODE=smoke_local`; ambos
  servidores temporales activan el opt-in legacy requerido; el `codex-fake`
  file-based escribe `external_summary.md` para que el intake seleccione el
  artefacto por `artifact_type` cuando hay varios ficheros.
- `BUG-ORQ-20260709-204` queda cerrado en wizard/i18n local: E5 detecto dos
  huecos estructurales en Nueva App. Primero, el guard de placeholders i18n
  vivia copiado en un test del wizard y no protegía todo catalogo nuevo.
  Segundo, algunas respuestas del wizard podian escribir campos de lista
  acumulativa en el mismo batch y perder la decision anterior por reemplazo;
  el caso observado fue T7 resiliencia + T8 cumplimiento sobre
  `agentes.preferencias`. Cierre: el aplicador mezcla listas acumulativas de
  restricciones/preferencias/compliance/locales sin duplicar valores, T8 declara
  destino primario `calidad.compliance`, el selector de turno evita campos
  destino repetidos en la pantalla visible y los tests centralizan los patrones
  prohibidos de placeholders para todo `NuevaAppI18nRequiredKeysV0`. Evidencia:
  `go test -count=1 ./modulos/orquesta-web` y `git diff --check`.
- `BUG-ORQ-20260709-205` queda cerrado en alcance local para E4/proveedor:
  el puerto `IdleSelfImprovementBlockersV0` existia, pero
  `idleSelfImprovementRunHasProviderAuthBlockerV0` devolvia siempre `false`,
  asi que la automejora podia seguir planificando aunque hubiese runs con
  `codex_app_server_provider_unauthorized` o limite de proveedor ya observado.
  Cierre: el detector busca codigos canonicos de proveedor/auth/cuota en las
  proyecciones del run sin inferir por severidad critica generica, el reason
  publico pasa a `provider_unavailable_paused`, y se anade evento local
  `provider` en `operator_notifications.v0` para avisar una sola vez por
  `reason_code+run_ref` con accion de reauth/cuota. Evidencia:
  `go test -count=1 ./cmd/orquesta-server -run 'TestIdleSelfImprovementBlockersV0|TestOperatorNotificationServerV0Notifica'`,
  `go test -count=1 ./modulos/orquesta-server -run 'TestRuntimeV0SupervisorNoPreparaAutomejoraConProveedorAuthBloqueadoV0'`
  y `go test -count=1 ./modulos/orquesta-operator-notifications`.
  Residual externo: wiring/smoke con Telegram remoto real queda para la fase
  remota con credenciales y bot autorizados.
- `BUG-ORQ-20260709-206` queda cerrado en observabilidad local
  DirectorStats/estado-vivo: los smokes no-OPES cerraban causalmente el
  `OperationalPlanState` con `operational-closure-succeeded`, pero
  `/api/v0/director/stats` podia seguir publicando `closure.status=blocked`
  por una evidencia stale `estado_vivo_entregado_parcial`. Cierre:
  `orquesta.director.stats.v0` consume el `OperationalPlanStateStore`
  inyectado, aplica el PlanState cerrado despues de `estado_vivo` y solo
  conserva bloqueos duros (`proceso_vivo`, `conflicto`, `bloqueado`,
  `terminal_rework`). El stack Codex cablea el store real. Evidencia:
  `go test -count=1 ./modulos/orquesta-mcp`, focales de stack/server,
  `ORQUESTA_EXTERNAL_WORK_LEGACY_DIRECTOR_LOOP=1
  ./scripts/smoke_external_domain_fake_real.sh` con
  `external_cycle_closure_status=closed`, y
  `ORQUESTA_EXTERNAL_WORK_LEGACY_DIRECTOR_LOOP=1
  ./scripts/smoke_external_domain_non_opes_real.sh` con PlanState cerrado.
- Avance E4 local 2026-07-09b: el bloqueador
  `provider_unavailable_paused` ya esta cableado al notifier opcional del
  supervisor de `cmd/orquesta-server`. Si la configuracion canonica de Telegram
  tiene token y `notification_target_ref`, envia un evento operador `provider`
  deduplicado por `provider:reason:run_ref` y anade evidencia de notificacion
  al resultado del blocker; si no hay sender, es no-op. Esto sigue dejando para
  remoto solo el smoke con bot/credenciales reales. Evidencia:
  `TestIdleSelfImprovementBlockersV0NotificaProviderPausadoUnaVezV0` y
  `go test -count=1 ./cmd/orquesta-server -run 'TestIdleSelfImprovementBlockersV0|TestOperatorNotificationServerV0Notifica|TestOperatorNotificationHermesTelegramNotifierV0UsaSendSinAuthCodex|TestTelegramBotAPI'`.
- Avance local BUG-079 2026-07-09k: ademas de limitar la respuesta de
  `thread/read` en protocolo WebSocket/comando, la observacion activa de
  `orquesta-runtime-codex-appserver` ya convierte una respuesta de hilo
  sobredimensionada en `blocked` con issue exacto
  `codex_app_server_thread_read_response_too_large` y evidencia
  `evidence-ref-codex-app-server-active-goal-thread-read-failed`. Esto no
  cierra el enforcement pre-tool del proveedor, pero evita que el observador
  silencie un stdout gigante y siga publicando un falso `running` generico.
  Evidencia: `TestServerCodexAppServerGoalBackendV0ObservaThreadReadGiganteComoBloqueoV0`,
  `TestCodexAppServerCommandProtocolThreadReadResponseBudgetV0`,
  `TestCodexAppServerWebSocketThreadReadResponseBudgetV0` y
  `go test -count=1 ./modulos/orquesta-runtime-codex-appserver`.
- Avance E3 local 2026-07-09a/b: se anade inventario focal de superficies MCP
  internas: Nueva App (`solicitar_nueva`, `wizard`, `wizard_bot`),
  `autoprogramming.prepare_run`, `autoprogramming.status` y
  `operator_director.message`. El guard existente ya detectaba tools
  registradas sin DTO canonico y campos stale en `input_schema`, pero no
  detectaba el inverso: un campo nuevo en el DTO no anunciado por el
  descriptor. Nuevo test
  `TestMCPInternalContractSurfaceInventoryV0CubreCamposCanonicos` falla si una
  tool interna registrada no esta inventariada o si cualquier campo canonico
  del DTO queda fuera del `input_schema`. Cierre local: se alinean los
  descriptores de `orquesta.autoprogramming.status.v0` y
  `orquesta.operator.director.message.v0` con sus DTOs reales. Evidencia:
  `go test -count=1 ./modulos/orquesta-mcp -run 'TestMCP(TransportToolInputSchema|InternalContractSurfaceInventory)'`
  y `go test -count=1 ./modulos/orquesta-mcp`.
- Avance E3 local 2026-07-09c: Telegram queda cubierto en el borde local del
  adaptador, no en remoto. `modulos/orquesta-operator-telegram` declara
  `CommandCatalogV0` como inventario canonico de comandos/aliases/metadatos de
  seguridad; `ParseCommandV0` resuelve desde ese catalogo y
  `TestCommandCatalogV0CubreComandosPublicosYParserV0` falla si un comando
  publico no esta inventariado, si hay aliases duplicados o si `stop` pierde
  la confirmacion requerida. Evidencia:
  `go test -count=1 ./modulos/orquesta-operator-telegram`. Residual: la
  superficie HTTP/web debe entrar por inventario cuando se toque ese adaptador;
  la prueba real de bot/credenciales sigue en fase remoto, no en nucleo.
- Avance E3 local 2026-07-09d: la superficie HTTP/web focal deja de depender
  de rutas sueltas. `PublicRouteManifestV0` publica `contract_refs` para Nueva
  App (`solicitar`, `wizard`, `wizard_bot`), autoprogramming
  (`prepare_run`, `status`) y review operador-Director; el server resource
  discovery propaga esos refs en `/api/v0/server/resources`. Guards:
  `TestPublicRouteManifestV0DeclaraContratosE3InternosV0` y
  `TestServerResourcesRouteManifestIncluyeDiscoveryOPESV0`. Residual: no se
  fuerza comparacion 1:1 de campos web/MCP porque web usa formularios y
  viewmodels, no los envelopes MCP; si se necesita paridad de campos web debe
  entrar con aliases/nesting explicitos para evitar falsos rojos.
- Residual E6 guard scripts: los guards de scripts estan verdes, pero siguen
  repartidos en varios tests de
  `cmd/orquesta-server/smoke_goal_first_script_guard_v0_test.go`. Propuesta de
  cierre futuro: tabla unica `script -> contratos exigidos` para
  `shutdown_common_runtime_dir`, `direct_shutdown_http_contract`,
  `managed_endpoint`, `no_historical_port`,
  `no_managed_server_outside_ctl_deploy` y `delegated_start_cleanup`, sin tocar
  scripts. No bloquea el nucleo ni conectores locales.
- Avance E6 local 2026-07-09: se anade la primera tabla consolidada
  `scriptContractGuardV0` para contratos exactos `script -> snippets/forbidden`
  en `cmd/orquesta-server/smoke_goal_first_script_guard_v0_test.go`, cubriendo
  wrappers goal-first, forced-stop, shutdown coordination, Claude process,
  `smoke_common` endpoint gestionado y deploy gestionado. Los guards globales
  `WalkDir` y los checks de orden se conservan separados para no ocultar que
  una regla aplica a toda la flota de scripts. Residual: consolidar mas bloques
  solo cuando se vuelva a tocar esa zona.
- `BUG-ORQ-20260709-208` queda cerrado en inventario MCP local: el contrato
  HTTP `operator_director.review_plan.v0` ya estaba declarado en el manifest
  de rutas, pero la tool MCP `orquesta.director.human_work.review_plan.v0`
  quedaba fuera de `mcpInternalContractSurfaceInventoryForTestV0`. Eso dejaba
  un falso verde E3: un campo nuevo en el DTO MCP de review-plan podia no
  aparecer en `input_schema` sin que el guard focal lo detectase. Cierre: el
  inventario MCP incluye `operator_director.review_plan.v0` y la tool pasa a
  requerir inventario. Evidencia focal:
  `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-http-gateway -run 'TestMCPInternalContractSurfaceInventory|TestPublicRouteManifestV0DeclaraContratosE3InternosV0'`.
- `BUG-ORQ-20260709-209` queda cerrado en `operator_notifications.v0`: el
  dedupe se registraba antes del envio y, si el sender fallaba, el reintento
  quedaba suprimido. Cierre: los stores que soportan `ForgetV0` liberan la
  clave cuando `SendOperatorNotificationV0` devuelve error; el store en memoria
  implementa esa liberacion. Evidencia focal:
  `TestOperatorNotificationV0NoDeduplicaSiSenderFallaYPermiteRetry`.
- `BUG-ORQ-20260709-210` queda cerrado en shutdown goal-first local: el stack
  recogia readers/cleaners desde `AppGoalObserver`, `AppGoalLauncher` y
  `AppGoalReworkLauncher` y podia invocar tres veces el mismo backend envuelto.
  Cierre: `orquesta-server-shutdown` expone identidad opcional
  `ActiveShutdownWorkIdentityPortV0`, los wrappers goal-first de `cmd` propagan
  la identidad del backend `app_server_tmux`, y el stack deduplica por esa
  identidad antes de leer o limpiar active work. Evidencia focal:
  `TestStackShutdownActiveWorkCleanerV0DeduplicaPuertosConMismaIdentidadV0` y
  `TestServerGoalWorkPortsFromBackendV0PropaganActiveShutdownWorkV0`.
- `BUG-ORQ-20260709-211` queda cerrado en el borde E1 de deploy atomico: el
  script podia declarar `ok` aunque `ctl status` fallase, una URL de readiness
  configurada no respondiese o ninguna superficie expusiera `sha256` del binario
  vivo. Ademas, algunos fallos por `set -e` no dejaban recibo durable. Cierre:
  `scripts/orquesta_server_deploy.sh` registra fase, escribe recibo
  `deploy_unhandled_failure` para roturas no controladas, exige `ctl status`,
  bloquea URLs configuradas inalcanzables y requiere identidad runtime
  `binary_sha256`/`runtime_binary_sha256`/`orquesta_server_sha256`. El guard
  comprueba binario instalado, `deploy_status_failed` y
  `deploy_runtime_identity_missing`. Evidencia:
  `bash -n scripts/orquesta_server_deploy.sh scripts/test_orquesta_server_deploy.sh`
  y `bash scripts/test_orquesta_server_deploy.sh`.
- `BUG-ORQ-20260709-212` queda cerrado en el borde E2 nightly local: el JSON no
  registraba ref git y el cierre nightly podia quedar verde sin notificacion
  Telegram, incluso con canal operador activado pero incompleto. Cierre:
  `scripts/orquesta_smoke_nightly.sh` anade `git.ref`, `git.branch`,
  `git.dirty`, bloquea config Telegram con schema invalido, envia notificacion
  terminal por Bot API cuando `telegram_operator.enabled=true` en la config
  canonica, y convierte un fallo de envio/config en
  `phase_reached=notification_failed`. El guard usa Bot API falso y prueba tanto
  envio correcto como Telegram activado mal configurado. Evidencia:
  `bash -n scripts/orquesta_smoke_nightly.sh scripts/test_orquesta_smoke_nightly.sh`,
  `bash scripts/test_orquesta_smoke_nightly.sh` y
  `GOFLAGS=-buildvcs=false go test -count=1 ./modulos/orquesta-operator-notifications ./modulos/orquesta-operator-telegram ./cmd/orquesta-server -run 'TestOperatorNotification|TestTelegram(BotAPI|Operator)|TestAdapterV0DespachaMensajeAlCanalDirector'`.
- `BUG-ORQ-20260709-213` queda cerrado en E3 contratos multisuperficie local:
  el manifest HTTP ya tenia parte de los `contract_refs`, pero Telegram
  operador no figuraba como contrato opt-in y no habia una tabla cruzada que
  obligase a alinear HTTP manifest, discovery del servidor y DTOs MCP. Cierre:
  `RouteOperatorTelegramUpdateV0` vive en `orquesta-http-gateway`, el handler
  del servidor usa esa constante, el discovery publica
  `operator_telegram.update.v0` como ruta opt-in no montada por defecto, y
  `TestServerE3ContractSurfaceCatalogV0*` inventaria los contratos internos
  entre HTTP/discovery/MCP. Ademas `mcpInternalToolRequiresInventoryForTestV0`
  cubre por prefijo las familias `nueva_app`, `director.human_work` y
  `operator.director`, evitando que una tool interna futura quede fuera del
  inventario E3. Evidencia:
  `go test -count=1 ./modulos/orquesta-http-gateway ./modulos/orquesta-mcp ./cmd/orquesta-server -run 'TestPublicRouteManifestV0DeclaraContratosE3InternosV0|TestServerResourcesRouteManifestIncluyeDiscoveryOPESV0|TestServerE3ContractSurfaceCatalogV0|TestMCPInternal'`.
- `BUG-ORQ-20260709-214` queda cerrado en E5 wizard/i18n local: el guard
  anti-placeholders miraba el catalogo actual, pero no obligaba a inventariar
  nuevas superficies i18n, y el guard de preguntas deduplicadas ocultaba
  duplicados raw de campos destino que podian pisarse si cambiaba el selector.
  Cierre: `nuevaAppI18nCatalogGuardInventoryV0` enumera los catalogos bajo
  guard y falla ante claves fuera de inventario o locales incompletos; el
  wizard declara una allowlist exacta de campos raw duplicados intencionales y
  falla si aparece un duplicado nuevo o si la allowlist queda obsoleta.
  Evidencia:
  `go test -count=1 ./modulos/orquesta-web -run 'TestNuevaAppI18nCatalogV0|TestWizard'`,
  `go test -count=1 ./modulos/orquesta-web` y `git diff --check`.
- `BUG-ORQ-20260709-215` queda cerrado como flaky local de test app-server:
  `go test -count=1 ./...` fallo una vez en
  `TestServerCodexAppServerGoalBackendV0ToolOutputPolicyYThreadReadGigantePorWebSocketDeterministaV0`
  con `fake websocket app-server: write ... broken pipe`. El test validaba que
  el cliente corta/bloquea un `thread/read` gigante, pero el fake trataba como
  fatal que el cliente cerrase la conexion tras detectar el exceso de
  presupuesto. Cierre: el fake WebSocket conserva fallos reales, pero ignora
  cierres benignos (`net.ErrClosed`, `os.ErrClosed`, `EPIPE`, `ECONNRESET`);
  el contrato lo siguen comprobando las aserciones de policy enviada/aceptada y
  `codex_app_server_thread_read_response_too_large`. Evidencia:
  `go test -count=10 ./modulos/orquesta-runtime-codex-appserver -run 'TestServerCodexAppServerGoalBackendV0ToolOutputPolicyYThreadReadGigantePorWebSocketDeterministaV0'`
  y `go test -count=1 ./modulos/orquesta-runtime-codex-appserver`.
- `BUG-ORQ-20260709-216` queda cerrado en el deploy atomico antes de la primera
  prueba real remota: si el servidor ya estaba vivo, `scripts/orquesta_server_deploy.sh`
  hacia build y swap, pero `orquesta_server_ctl.sh start` podia ser un no-op
  (`ya vivo`) y la verificacion final comparar contra el binario viejo. Cierre:
  el deploy ejecuta `ctl stop` tras build y antes del swap; si falla, aborta
  con `deploy_stop_failed` y conserva el binario anterior. Evidencia:
  `bash -n scripts/orquesta_server_deploy.sh scripts/test_orquesta_server_deploy.sh`
  y `bash scripts/test_orquesta_server_deploy.sh`.
- `BUG-ORQ-20260709-217` queda cerrado tras la primera ejecucion real del deploy
  remoto: el servidor publico exponia el SHA en
  `runtime_identity.binary_sha256`, pero `scripts/orquesta_server_deploy.sh`
  solo buscaba `binary_sha256`/`runtime_binary_sha256`/`orquesta_server_sha256`
  en top-level y fallo con `deploy_runtime_identity_missing` pese a que el
  binario vivo coincidia. Cierre: el verificador acepta tambien
  `runtime_identity.binary_sha256` y el test reproduce esa forma anidada.
  Evidencia: primera ejecucion remota con receipt failed por
  `deploy_runtime_identity_missing`, `curl /api/status` mostrando
  `runtime_identity.binary_sha256=d8f42b0c746fc192f92d65c2b3905c55c646bf3e8de15da5765a3572e9105f85`,
  `bash -n scripts/orquesta_server_deploy.sh scripts/test_orquesta_server_deploy.sh`
  y `bash scripts/test_orquesta_server_deploy.sh`.
- `BUG-ORQ-20260709-218` queda cerrado antes de repetir el deploy real remoto:
  la primera ejecucion del deploy dejo
  `/srv/orquesta-self/worktrees/orquesta` en `detached HEAD`. El HEAD del
  worktree estaba en `7dcff1a`, pero la rama local
  `trabajo/plataforma-agentes` seguia en `c9027743`; al ejecutar de nuevo con
  `ORQUESTA_DEPLOY_REF=trabajo/plataforma-agentes`, el script resolvia la rama
  stale y abortaba con
  `deploy_not_fast_forward HEAD=7dcff1a... target=c9027743...`. Cierre: cuando
  `ORQUESTA_DEPLOY_REF` es una rama valida, el deploy hace
  `checkout -B <ref> <target_sha>` y conserva/actualiza la rama desplegable en
  vez de dejarla detached. Evidencia:
  `bash -n scripts/orquesta_server_deploy.sh scripts/test_orquesta_server_deploy.sh`
  y `bash scripts/test_orquesta_server_deploy.sh` con
  `test_success_preserves_branch_worktree`.
- Revalidacion OPES local/fake 2026-07-09h:
  `scripts/smoke_opes_lifecycle_real.sh` vuelve a pasar en local con
  `ORQUESTA_KEEP_SMOKE_DIR=1`, 24/24 `work_kind` cubiertos hasta
  `finalize_temario_package`, `finalpkg_dry_run=false`,
  `settlement_status=settled_final` y `no_residual_processes=true`. Evidencia
  retenida:
  `/tmp/orquesta-opes-lifecycle-real-20260709T172932Z/out/opes_lifecycle_result.json`.
  Un subagente read-only reviso `BUG-058/066/075` y no encontro bug local de
  codigo accionable; los residuales vigentes son OPES temporal/preproduccion,
  proveedor real/remoto y prueba de campo de no reescritura tardia.
- Reejeucion real 2026-07-04 noche 10:
  `smoke_goal_first_checkpoint_only_high_consumption_real=ok` con
  `run_ref=run-spec-smoke-goal-first-bug088-req-smoke-goal-first-bug088-6c8dc4317888c8e25bb0e91f7f910aab`,
  `tokens_used=13658`, checkpoint y segundo artefacto materializados, servidor
  final `stopped/shutdown_ready=true` y `app_server_tmux_processes_alive=0`.
  Evidencia retenida en `/tmp/orquesta-goal-first-app-server.gibwtZ` saneada
  sin `codex-home` ni binario temporal. Reduce `BUG-079/165/065`, pero no cierra
  el forced stop de Sueldos ni el enforcement duro previo a herramientas.
- Reejeucion real 2026-07-04 noche 11:
  `smoke_goal_first_forced_stop_backend_real=ok` cierra la ruta Sueldos de
  `BUG-ORQ-20260704-165`: alto consumo con backend `app_server_tmux` vivo,
  `POST /api/v0/runs/control` forced stop devuelve `estado=ok`,
  `status=stopped`, `final_status=stopped`, `goal_status_after=blocked`;
  el `observe_goal` posterior devuelve `goal_status=blocked`,
  `closure_status=blocked`, `recommended_action=replan`; cleanup final sin
  `orquesta-server run`, sin `codex app-server` y sin tmux `orquesta-goal-*`.
  Evidencia retenida saneada en `/tmp/orquesta-goal-first-app-server.Sc7e7K`;
  runbook:
  `docs/runbooks/smoke_goal_first_forced_stop_backend_real_2026-07-04.md`.
  `BUG-165` queda abierto solo para los residuales amplios de
  `status/observe` lento y coordinacion automatica completa de
  shutdown/backend/checkpoint/stop/cancel/wait.
- Avance 2026-07-09: `BUG-ORQ-20260709-198` queda cerrado. El nuevo smoke real
  `scripts/smoke_goal_first_shutdown_coordination_real.sh` reprodujo primero
  un falso `backend_still_running`: tras 12 POST a `/api/v0/server/shutdown`
  con `cleanup_goal_backends=true`, ya no quedaban tmux, socket, owner ni
  proceso `codex app-server`, pero el adaptador `app_server_tmux` seguia
  publicando active work por un estado degradado/preparando sin evidencia viva;
  ademas el harness fallaba con `session_name: unbound variable` en la ruta de
  diagnostico. Cierre: `ReadActiveShutdownWorkV0` solo reporta residuo
  configurado si observa owner, sesion, socket, pane o proceso real, y el
  harness inicializa `session_name`. Reejecucion real posterior:
  `run_ref=run-spec-smoke-goal-first-bug088-req-smoke-goal-first-bug088-e166f0e04a747b89605e44a56019b11c`,
  `external_goal_ref=019f46a9-4bad-7372-8196-1ba614ff51a0`,
  `autoprogramming_status_before_shutdown_visible=true`,
  `/server/shutdown status=ready`, `shutdown_ready=true`,
  `active_work_count=0`, `goal_actions[0].action_taken=cleanup_completed` y
  `app_server_tmux_processes_alive=0`. Evidencia saneada:
  `/tmp/orquesta-smokes-codex/orquesta-goal-first-app-server.Ya4ayF`
  (~404 KiB, sin `codex-home` ni binario temporal). Alcance: cierra el falso
  bloqueo de cleanup `app_server_tmux`; no cierra `BUG-165/065` global porque
  este smoke finaliza con `runs_requested=0` y queda pendiente la reconciliacion
  completa de runs goal-first fuera de cola durante shutdown amplio.
- Avance 2026-07-09b: queda cerrado el subcaso offline/focal de esa
  reconciliacion pendiente. El stack Codex de shutdown ahora suplementa la
  cola con candidatos goal-first terminales que no estan visibles en
  `RunQueue` ejecutable pero conservan `RunControl=stop_requested` o
  `cancel_requested`; el `ActiveWorkReader` los publica como `goal_first`
  pendiente mientras no haya backend vivo; y el writer de shutdown completa
  `RunControl` a `stopped/canceled` cuando el goal ya esta terminal. Prueba:
  `TestStackShutdownV0ForzadoCoordinaGoalFirstTerminalFueraDeColaV0` valida
  cola vacia, `runs_requested=1`, `runs_stopped=1`, `shutdown_ready=true`,
  `RunControl stopped` y evidencia
  `evidence-ref-server-shutdown-goal-terminal-run-control-reconciled`.
  Verificado con `go test -count=1 ./modulos/orquesta-app-codex-stack` y
  `go test -count=1 ./modulos/orquesta-server-shutdown`. Pendiente para cerrar
  `BUG-165/065` global: smoke real amplio con proveedor lento/stale/remoto.
- Avance 2026-07-09c: se amplia la cobertura semi-real por HTTP y se endurece
  el smoke real existente. Nuevo test
  `TestServerAppHTTPGoalFirstShutdownCoordinaControlPendienteFueraDeColaV0`
  arranca goal-first por `/api/v0/apps/director`, cierra el goal con observe,
  deja `RunControl=stop_requested` y `RunControl=cancel_requested` mediante
  `/api/v0/runs/control forced=false`, y verifica que
  `/api/v0/server/shutdown forced=true cleanup_goal_backends=true` devuelve
  `runs_requested=1`, `runs_stopped=1`, `shutdown_ready=true` y estado terminal
  `stopped/canceled` sin active work. Ademas
  `scripts/smoke_goal_first_app_server_real.sh` ya no acepta `ready` si
  `runs_requested<1`, `runs_stopped<runs_requested` o algun
  `runs[].control_status` no es `stopped` en el modo `shutdown_coordination`.
  Focales:
  `go test -count=1 ./cmd/orquesta-server -run 'TestServerAppHTTPGoalFirstShutdownCoordinaControlPendienteFueraDeCola|TestSmokeGoalFirstShutdownCoordinationReal'`
  y `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestStackShutdownV0ForzadoCoordinaGoalFirstTerminalFueraDeCola|TestStackShutdownRunControlWriterV0ForcedStopMarcaGoalTerminalReplanificable'`.
  Pendiente para cerrar `BUG-165/065` global: ejecutar smoke real amplio con
  proveedor lento/stale/remoto.
- Avance 2026-07-09d: se ejecuto el smoke real amplio local con proveedor
  `app_server_tmux` tras el endurecimiento anterior. Comando:
  `ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_REAL_CONFIRM=1 ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_CODEX_EXECUTION_CONFIRMED=1 ORQUESTA_GOAL_FIRST_SMOKE_POLLS=50 ORQUESTA_GOAL_FIRST_SMOKE_SLEEP_SECONDS=3 ORQUESTA_KEEP_SMOKE_DIR=1 ORQUESTA_SMOKE_PARENT=/tmp/orquesta-smokes-codex ./scripts/smoke_goal_first_shutdown_coordination_real.sh`.
  Resultado: `smoke_goal_first_shutdown_coordination_real=ok`,
  `autoprogramming_status_before_shutdown_visible=true`,
  `runs_requested=1`, `runs_stopped=1`,
  `shutdown_coordination_all_runs_stopped=true`,
  `run_control_statuses=stopped`, `shutdown_ready=true`,
  `goal_actions[0].action_taken=cleanup_completed` y
  `app_server_tmux_processes_alive=0`. Evidencia saneada:
  `/tmp/orquesta-smokes-codex/orquesta-goal-first-app-server.kALS5q` (~376 KiB,
  sin `codex-home`, `auth.json`, `config.toml` ni binario temporal). Esto cierra
  el hueco local `runs_requested=0` de `BUG-165/065` para el backend real
  `app_server_tmux`; mantener como residual externo solo una validacion en
  despliegue remoto/stale si se exige evidencia fuera de esta maquina.
- `BUG-ORQ-20260704-166` queda cerrado funcionalmente por `ff620ecf` y
  `0e0dcedc`, incluyendo el ajuste posterior de `.gocache-local`, para la causa
  observada: el escaneo de
  resultados saltaba caches voluminosas tarde y podia agotar el limite antes de
  encontrar el receipt terminal `complete`.
  Reintento de campo Sueldos/Orquesta posterior al fix aceptado:
  `run_status=cerrada`, `goal_status=complete`, `closure_status=accepted`.
- `BUG-ORQ-20260704-167` queda cerrado por `0e0dcedc`: el reintento de campo
  `request-ref-sueldos-cargos-partidos-20260704-003` arranco el goal con
  `goal_status=running` tras reiniciar Orquesta con el fix de snapshot.
  el snapshot runtime de write-set saltaba prefijos Orquesta conocidos pero no
  variantes nuevas como `.orquesta-feature-cargos`, de modo que un servidor
  temporal podia auditar su propio estado local y fallar el arranque con
  `codex_app_server_runtime_write_set_guard_snapshot_failed`.
- `BUG-ORQ-20260704-168` queda cerrado por `0e0dcedc`: el reintento de campo
  `request-ref-sueldos-cargos-partidos-20260704-004` ya no mezclo
  `artifact_refs` ni tests del run antiguo `...43c20...` tras rechazar receipts
  terminales con `goal_ref` distinto al goal actual.
- `BUG-ORQ-20260704-170` queda cerrado por el avance Codex 2026-07-04 noche
  20: el smoke servidor real `claude_process` detecto una forma recuperable de
  proveedor (`evidence_refs` como objetos `{ref, description}`) que antes
  invalidaba el cierre; Claude/Gemini normalizan ahora esa forma acotada y el
  prompt exige arrays de strings.
- `BUG-ORQ-20260704-171` queda cerrado por el avance Codex 2026-07-04 noche
  20: el smoke servidor real `claude_process` usa `--safe-mode` por defecto
  para no heredar MCPs locales de Claude. La ejecucion aceptada
  `/tmp/orquesta-claude-process-server.x5N8pq` no dejo procesos residuales ni
  `codebase-memory-mcp`.
- `BUG-ORQ-20260704-165` queda reducido por el smoke real de servidor
  `claude_process` con `cmd/orquesta-server`: launch/observe/closure por HTTP
  cierra `accepted` con proveedor externo real. El residual abierto se acota a
  `runs/control`/shutdown contra proveedor externo vivo o lento y coordinacion
  automatica completa.
- Avance 2026-07-04 noche 21: `BUG-ORQ-20260704-165` queda reducido tambien
  por `runs/control`/shutdown real con `claude_process` vivo. El smoke
  `SMOKE_CLAUDE_GOAL_PROCESS_SERVER_CONTROL_MODE=forced_stop` valida
  `stopped`, `goal_status_after=blocked`, observe posterior `blocked/replan`,
  shutdown `stopped/ready` y cero procesos residuales. Residual: Gemini real
  cuando el proveedor permita tier/credencial y observabilidad global larga si
  reaparece.
- Avance 2026-07-04 noche 22: las filas de descriptor compacto
  `BUG-ORQ-20260701-065/088` y `BUG-ORQ-20260701-066/088` para
  `director.stats`, `run_queue.priority`, `external_work.run` y
  `arrancar_director` quedan cerradas como residuales de contrato publico. Los
  bugs padre `065` y `066` siguen abiertos solo por sus filas propias de
  coordinacion automatica/OPES, no por discovery MCP.
- Avance 2026-07-04 noche 23: auditoria Codex + subagentes read-only, sin
  `codebase-memory-mcp`, cierra filas stale ya cubiertas por codigo/pruebas:
  subfilas de `BUG-065` sobre cleanup externo no forzado, `BUG-076` app-server
  residual, proyecciones `BUG-079` en status/queue/observe/efficiency/
  domain-work, y proyecciones `BUG-075`/`BUG-066/075` de QA, fase 0 y evidencia
  requerida. Conteo vigente de tabla: 208 filas, 173 IDs, 7 filas `abierto`.
  Quedan abiertos solo padres o residuales reales: `BUG-165`, `BUG-058`,
  `BUG-065`, `BUG-066`, `BUG-073`, `BUG-075` y `BUG-079`.
- Avance 2026-07-04 noche 24: `observe_active_goals` ya no queda bloqueado por
  un solo `observe_goal` lento; cada run tiene timeout propio, genera snapshot
  parcial con evidencia y el batch continua con los demas goals. Reduce
  `BUG-165`, que sigue abierto solo para el residual global de observabilidad/
  control largo si reaparece y automatizacion completa.
- Avance 2026-07-04 noche 25: `BUG-ORQ-20260704-165` queda reducido en la ruta
  de backend goal-first ausente/stale: `autoprogramming/status` publica
  safe actions y `runs/control` puede reconciliar cleanup externo cuando no hay
  proceso vivo. `safe_actions` hacia `/api/v0/runs/control` incluyen
  `run_control_reconcile_external_cleanup` y evidencia de cleanup externo; si
  `Goal.Status=running` viene de stats stale sin proceso vivo y
  `SafeToReconcile`, status/control ya no lo tratan como backend activo ni
  devuelven `control_not_propagated_to_goal_backend`.
- Reproduccion 2026-07-04 tarde Codex: `BUG-ORQ-20260704-165` sigue abierto
  para el residual amplio de observabilidad/control con varios goals reales.
  Se lanzaron por Orquesta siete pilotajes aislados T1-T7; los T1/T4/T6
  terminaron `goal_status=invalid` con solo checkpoint inicial y T2/T3/T5/T7
  quedaron `blocked` por alto consumo sin receipt terminal util. `runs/control
  stop forced=true` devolvio `estado=ok`, `status=stopped` y
  `goal_control_signal_confirmed=true`, pero `/api/v0/server/shutdown` devolvio
  `backend_still_running` para los backends `app_server_tmux` y fue necesaria
  limpieza local de procesos temporales. Evidencia: roots
  `/tmp/orquesta-autonomia-t7-single-20260704T142642Z` y
  `/tmp/orquesta-autonomia-isolated-batch-20260704T142715Z`; result placeholders
  `cmd/orquesta-server/docs/orquesta_goal_result_goal-ref-task-autoprogramming-1b05f5450521-g01.json`,
  `modulos/orquesta-context/docs/orquesta_goal_result_goal-ref-task-autoprogramming-b16dbf6de0af-g01.json`,
  `modulos/orquesta-runtime-codex-goal/docs/orquesta_goal_result_goal-ref-task-autoprogramming-5fb5b8417b08-g01.json`,
  `modulos/orquesta-server-shutdown/docs/orquesta_goal_result_goal-ref-task-autoprogramming-edd43a30277f-g01.json`,
  `modulos/orquesta-opes-director/docs/orquesta_goal_result_goal-ref-task-autoprogramming-8b61180dcd7d-g01.json`
  y `cmd/orquesta-server/docs/orquesta_goal_result_goal-ref-task-autoprogramming-d75ef5f8379d-g01.json`.
  Lectura: no son cierres funcionales; son reproduccion del residual
  `BUG-165/079/065`. Para continuar, relanzar goals pequenos con contexto
  estrecho, umbral correcto
  `ORQUESTA_AUTOPROGRAMMING_CHECKPOINT_ONLY_HIGH_CONSUMPTION_TOKENS` y
  validacion de cleanup final.
- Reproduccion adicional 2026-07-04 tarde Codex T7A: se relanzo un goal
  estrecho solo para `modulos/orquesta-web` con write-set aceptado
  (`run-autonomia-t7a-wizard-contracts-small-20260704-001`,
  `goal-ref-task-autoprogramming-1d01424da32d-g01`). Orquesta arranco Codex y
  materializo checkpoint, pero `observe_goal` quedo en
  `codex_app_server_goal_status_active_high_token_usage tokens_used=121675`
  sin receipt terminal. Aunque el proceso se arranco con
  `ORQUESTA_AUTOPROGRAMMING_CHECKPOINT_ONLY_HIGH_CONSUMPTION_TOKENS=450000`,
  el state efectivo publico el setting como `100000 defaulted`; por tanto el
  servidor `start` no proyecto el umbral esperado. `runs/control stop
  forced=true` cerro el run como `stopped` y `goal_control_signal_confirmed`,
  pero `/api/v0/server/shutdown` volvio a `backend_still_running` y hubo que
  parar el servidor temporal por `SIGINT`. Evidencia local:
  `/tmp/orquesta-autonomia-t7a-wizard-contracts-20260704T163755`.
  Lectura: T7A no se cierra; refuerza `BUG-165` y anade subcausa de
  configuracion efectiva/daemon start para el umbral de alto consumo.
- Avance 2026-07-04 noche 26: `BUG-ORQ-20260701-079` queda reducido en el borde
  app-server: el contrato runtime de `turn/start` ya no relaja
  `max_text_bytes` por un packet debil y conserva los hints acotados canonicos
  aunque el packet intente sustituirlos. Sigue pendiente enforcement real previo
  a herramientas/smoke largo de checkpoint temprano.
- Avance 2026-07-04 noche 27: `BUG-ORQ-20260701-079` queda reducido tambien en
  checkpoint previo a herramientas: `orquesta-runtime-codex-appserver`
  materializa `checkpoint_started.txt` dentro del `write_set` autorizado tras
  `thread/start` y antes de `turn/start`, de modo que existe evidencia durable
  antes de que el agente pueda ejecutar herramientas internas del app-server.
  Sigue pendiente el limite duro de stdout de herramientas internas, que no
  tiene campo compatible en el schema local de `turn/start` y requiere soporte
  del runtime/proveedor o interception del app-server.
- Avance 2026-07-04 noche 28: `BUG-ORQ-20260701-073` queda cerrado
  funcionalmente. El materializer no cuenta `checkpoint_started.txt` como
  `partial_artifacts_written` ni como `artifact-ref-materialized:*`, y
  `autoprogramming/status` separa `active_no_checkpoint_yet` de
  `active_checkpoint_only_yet`: con checkpoint inicial ya materializado pide
  `observe_goal_backend_require_next_artifact`, no otro checkpoint. El timeout
  activo con checkpoint reciente usa la misma accion de siguiente artefacto. El
  conteo vigente baja a 6 abiertos reales: `BUG-165`, `BUG-058`, `BUG-065`,
  `BUG-066`, `BUG-075` y `BUG-079`.
- Avance 2026-07-04 noche 29: `BUG-ORQ-20260704-165` y
  `BUG-ORQ-20260701-065` quedan reducidos en el borde CLI/HTTP de shutdown: si
  el `POST /api/v0/server/shutdown` inicial falla por transporte, timeout o
  conexion cortada sin cuerpo, `orquesta-server stop` consulta `/status` y, si
  hay `shutdown_in_progress`, `backend_still_running`, `active_work_refs` o
  acciones goal pendientes, devuelve el mismo `shutdown_not_ready ...` con refs
  compactas que ya bloquea la escalada por `--force`. Ya no queda un simple
  `shutdown_timeout`/`shutdown_request_failed` opaco cuando el state publico
  conserva causa accionable. Pendiente: smoke real amplio de proveedor/status
  lento y coordinacion automatica completa backend/checkpoint/stop/cancel/wait.
- Auditoria read-only 2026-07-04 noche 29b sobre residuales OPES: `BUG-075` es
  el cierre mas acotado por codigo/test, mediante validadores estructurados por
  `work_kind` para HTML, visual, audio, tutor/RAG, fuentes/reviews, casos,
  juegos/ayuda y rework causal; `BUG-058` puede reducirse con fake/temporal que
  demuestre tema asentado -> derivados pendientes -> paquete final solo con
  refs de calidad por tema; `BUG-066` no debe cerrarse solo con unit tests,
  porque requiere smoke temporal de external-work/OPES donde artefactos
  suficientes dejen de reescribirse y el cleanup/reconcile terminalice sin
  residuos SQLite/proceso. No usar rails por texto libre: solo campos
  estructurados, evidencias y validadores de dominio.
- Avance 2026-07-04 noche 30: `BUG-ORQ-20260701-075` queda reducido por
  `OPESArtifactQualityContractV0` en `orquesta-opes-director`: HTML, visuales,
  audio, tutor/RAG, fuentes, revisiones, supuestos, juegos, ayuda y
  reutilizacion visual ya tienen validacion estructurada minima. Una evidencia
  final nominal (`opes-final-evidence:*`) ya no basta si faltan manifest,
  refs, QA o campos estructurados del artefacto; el productor publica
  `artifact_quality_status=needs_rework`, settlement
  `artifact_quality_contract_failed` y rework causal
  `review_artifact_quality`. La ausencia total de evidencia minima conserva el
  diagnostico anterior `required_evidence_missing` para no mezclar causas.
  Pendiente: smoke temporal OPES/external-work que pruebe el ciclo completo con
  artefactos reales y sin reescritura tardia.
- Avance 2026-07-09: `BUG-ORQ-20260701-075` queda reducido en el adaptador
  `orquesta-opes-director` para aliases OPES emitidos por el bridge:
  `learning_games_package` se normaliza a `interactive_practice_package`,
  `help_manual_package` a `help_package`, `opes_quality_audit_report` exige
  `decision_global` y evidencias/rework, y `completed_syllabus_package` se trata
  como paquete final solo dentro del director OPES para liberar o replanificar
  por `manifest_cierre` sin tocar el core. Tests:
  `TestValidateOPESArtifactQualityContractV0NormalizaArtefactosOPESExtendidosV0`,
  `TestProduceOPESCausalJobsV0ArtefactoOPESExtendidoAplicaArtifactQualityV0`,
  `TestProduceOPESCausalJobsV0CompletedSyllabusPackageConManifestLiberaRegistroV0`;
  `go test -count=1 ./modulos/orquesta-opes-director`. No cierra `BUG-075`
  completo: siguen pendientes smoke OPES temporal/external-work con artefactos
  reales, proveedor y ausencia de reescritura tardia.
- Avance 2026-07-04 noche 31: `BUG-ORQ-20260701-058` queda reducido en el
  borde local de `orquesta-opes-director`: el registro OPES ya no libera un
  `completed_syllabus_package` aunque traiga `manifest_cierre` completo,
  evidencias de HTML/RAG/audio/tests/tutor/visual/QA y QA estricta, si faltan
  refs durables de resultados `OPESTopicQualityContractV0` por tema
  (`topic_quality_contract_result_refs` o `topic_quality_contract_results`).
  En ese caso conserva `pending_refs=final-package-manifest-closure-evidence-required`
  y rework causal `finalize_temario_package`. Pendiente: smoke temporal OPES
  que demuestre el arbol completo tema asentado -> derivados -> paquete final
  solo con refs de calidad por tema.
- Avance 2026-07-04 noche 32: `BUG-ORQ-20260701-058/066` queda reducido con
  smoke offline transversal en `orquesta-app-codex-stack`:
  `TestCodexStackV0OPESGoalFirstLifecycleAsientaDerivadosYCierraRegistroFinalConTopicQualityV0`
  cubre `external-work` goal-first -> receipt de dominio -> productor OPES ->
  `update_topic_registry`. El texto con checkpoint y QA pasada queda
  `settled_text`, `operational_status=waiting` y derivados pendientes; el
  paquete final con `manifest_cierre` y refs de resultados
  `OPESTopicQualityContractV0` libera registro como `settled_final`. Durante
  el smoke se corrigio una frontera real: los refs agregados
  `topic_quality_contract_result_refs` de un `final_domain_package` ya no
  activan por si solos la QA textual de tema; esa QA solo se declara por campos
  textuales/estado explicitos. Sigue abierto el smoke temporal OPES real y la
  prueba de no reescritura tardia/corte automatico tras entregas suficientes.
- Avance 2026-07-04 noche 32: `BUG-ORQ-20260701-079` queda documentado con
  frontera contractual explicita en
  `modulos/orquesta-runtime-codex-goal/docs/contratos.md`: `direction_contract`,
  checkpoint temprano, `max_text_bytes=16384` y `thread_read_max_bytes=256 KiB`
  limitan ingestion/observacion y salida compacta, pero no son enforcement duro
  pre-tool del proveedor. Sigue abierto hasta que el runtime/proveedor exponga
  ese corte o el app-server medie la ejecucion real de herramientas, mas smoke
  largo con proveedor real.
- Avance 2026-07-04 noche 33: `BUG-ORQ-20260701-075` queda reducido con smoke
  offline transversal en `orquesta-app-codex-stack`:
  `TestCodexStackV0OPESGoalFirstArtifactQualityHTMLDerivadoReworkYPassV0`
  cubre `external-work` goal-first -> receipt de dominio -> productor OPES para
  un derivado HTML. Si el artefacto terminal solo aporta evidencia nominal
  `opes-final-evidence:html_site_publicable` pero carece de manifest/reporte
  estructurado, el registro queda `artifact_quality_contract_failed` con
  rework causal `review_artifact_quality`; si trae
  `html_topic_pages_manifest` y `html_validation_report`, no crea rework. Sigue
  abierto el smoke temporal OPES/proveedor y la prueba de ausencia de
  reescritura tardia tras entrega suficiente.
- Avance 2026-07-04 noche 34: `BUG-ORQ-20260704-165` y
  `BUG-ORQ-20260701-065` quedan reducidos en el cliente CLI shutdown:
  `postServerShutdownRequestV0` ya usa timeout por intento y
  `waitServerShutdownReadyV0` acota cada re-POST al deadline restante. Si el
  POST inicial o un re-POST se cuelga, el cliente consulta `/status` y devuelve
  `shutdown_not_ready` con `active_work_refs` compactas, sin permitir signal
  forzada cuando el status conserva active work. Tests:
  `TestRequestServerShutdownV0PostColgadoConsultaStatusAccionableV0` y
  `TestWaitServerShutdownReadyV0RepostColgadoRespetaDeadlineYDevuelveStatusAccionableV0`.
  Siguen abiertos los bugs padre hasta smoke real amplio de proveedor/status
  lento y coordinacion automatica completa `backend/checkpoint/stop/cancel/wait`.
- Avance 2026-07-04 noche 35: `BUG-ORQ-20260701-058/066` quedan reducidos en
  dos bordes locales. `external_job_stats` ya cablea el ledger de DomainWork y
  no proyecta `completed` si un cierre goal-first aceptado con
  `DomainReceiptRefs` contradice el contrato durable del ledger; degrada a
  `blocked` con diagnostico `domain_work_receipt_artifact_incomplete` u otro
  issue causal del validador. Ademas, `orquesta-opes-director` ya no publica
  `review_director_consolidation` ni `assemble_topic` como
  `next_required_work_kinds` tras `settled_text`, evitando reescritura textual
  tardia sin rework causal. Tests:
  `TestCodexStackExternalJobStatsSourceV0GoalFirstAcceptedConLedgerIncompletoNoCompletaJob`,
  `TestProduceOPESCausalJobsV0NoBloqueaRegistroConQATemaCompletaV0` y
  `TestCodexStackV0OPESGoalFirstLifecycleAsientaDerivadosYCierraRegistroFinalConTopicQualityV0`.
  Siguen abiertos hasta smoke OPES temporal real, cierre agregado del arbol y
  prueba de ausencia de reescritura tardia con proveedor.
- Avance 2026-07-04 noche 35b: se materializa
  `scripts/smoke_opes_lifecycle_real.sh` para cerrar el residual comun
  `BUG-ORQ-20260701-058/066/075` con un arnes reproducible: lifecycle
  OPES fake/local de `external-work -> observe -> derivados ->
  finalize_temario_package`, fixture de registry finalpkg local,
  lanzamiento `dry_run=false` con `course_id`, `template_run_ref` y
  `template_topic_id` explicitos, y validacion de settlement final por el test
  focal `TestProduceOPESCausalJobsV0PaqueteFinalCompleteConManifestCompatibleYQATernaLiberaRegistro`.
  Evidencia local de este corte: `bash -n scripts/smoke_opes_lifecycle_real.sh`
  pasa. La ejecucion completa del smoke queda bloqueada en este sandbox porque
  Python no puede abrir sockets loopback para el fake OPES
  (`PermissionError: [Errno 1] Operation not permitted`); por tanto las filas
  padre siguen abiertas hasta repetir el script en entorno con sockets locales
  permitidos y conservar `out/opes_lifecycle_result.json`.
- Revalidacion Codex local 2026-07-09: el entorno local actual si permite el
  fake loopback. Se ejecutan `scripts/smoke_opes_derivatives_rest.sh` en modo
  `ORQUESTA_OPES_DERIVATIVES_FAKE_SERVER=1` con `dry-run-once` y despues
  `run-until-finalize` effectful fake (`ORQUESTA_OPES_DERIVATIVES_EXECUTE=1`,
  sleep 0, max 40 ticks). Resultado: `goal_receipts_manifest_expected=24`,
  `goal_receipts_manifest_covered=24`,
  `goal_receipts_manifest_final_work_kind=finalize_temario_package`,
  `goal_receipts_manifest_status=ok`, `run_until_status=completed`,
  `empty_after_final=true`. Manifest:
  `/tmp/opes-salidas/derivatives-rest-codex-local-finalize-20260709T154800Z/goal_receipts_manifest.json`.
  Esto valida el lifecycle local/fake del conector; no cierra los bugs padre
  porque siguen pendientes smoke OPES temporal/preprod con proveedor real y
  prueba de ausencia de reescritura tardia con agente real.
- Avance 2026-07-04 noche 36: `BUG-ORQ-20260701-079` queda reducido en el
  protocolo `app_server_command`: el presupuesto por linea JSON-RPC baja de
  1 MiB a 256 KiB tambien para metodos genericos como `turn/start`, y una
  respuesta sobredimensionada devuelve
  `codex_app_server_command_response_too_large` en vez de error opaco de
  scanner. `thread/read` conserva su issue especifico
  `codex_app_server_thread_read_response_too_large`. Tests:
  `TestCodexAppServerCommandProtocolTurnStartResponseBudgetV0` y
  `TestCodexAppServerCommandProtocolThreadReadResponseBudgetV0`. Sigue abierto
  el limite duro previo a herramientas internas del proveedor y el smoke largo
  real.
- Incidencia operativa 2026-07-04 noche 30: durante la verificacion de OPES,
  `go test` fallo antes de compilar por `/home` al 100% y cache Go
  `/home/alberto/.cache/go-build` de 17G. Se libero con `go clean -cache` y la
  tanda se reejecuto con `GOCACHE=/tmp/orquesta-codex-gocache` y
  `GOTMPDIR=/tmp/orquesta-codex-gotmp`. Queda registrada como fallo operativo
  cerrado de higiene de disco; no es bug funcional de Orquesta, pero futuras
  sesiones largas deben presupuestar cache fuera de `/home`.
- Avance 2026-07-04: `BUG-ORQ-20260704-165` queda parcialmente reducido por
  `7a6dea0d`: `runs/control stop/cancel forced=true` propaga el stop al backend
  Goal cuando el puerto de control esta disponible, guarda cierre terminal
  `blocked/canceled` con evidencias y `observe_goal` no vuelve a publicar
  `running` si ya existe evidencia terminal forzada. No se declara cerrado
  total sin reejecutar un caso real posterior de alto consumo/status lento.
- Avance 2026-07-04 noche: `BUG-ORQ-20260704-165` queda reducido tambien por
  timeout interno del observador residente: `GoalObserverTimeout`, default
  2000 ms, configurable por composicion `ConfigV0` sin env nueva por ratchet
  MEJ-106. El tick de `ObserveActiveGoalWorksV0` usa contexto acotado y
  persiste `goal_observer_timeout` como error accionable.
- Avance 2026-07-04 noche 2: el timeout del observador residente ya no depende
  de que el backend respete `context.Context`: la llamada al backend se aisla
  con deadline duro, el tick vuelve y publica `goal_observer_timeout`; mientras
  la llamada anterior siga viva, nuevos ticks publican
  `goal_observer_backend_call_in_flight` y no lanzan llamadas infinitas. El
  self-watchdog conserva esa llamada como causa operacional viva. Sigue abierto
  para revalidacion real amplia de `status/observe` lento y coordinacion
  completa con backend/proveedor.
- Avance 2026-07-04 noche 6: `BUG-ORQ-20260704-165` queda reducido en la ruta
  de snapshot/timeout de `observe_goal`: si `GoalStateStore` aun dice
  `running/accepted` pero `RunControl` ya esta terminal (`stopped` o
  `canceled`), el snapshot no publica `goal_status=running`; proyecta
  `goal_status=blocked`, `closure_status=blocked`, `recommended_action=replan`
  y evidencia `evidence-ref-observe-goal-run-control-terminal`. Sigue abierto
  para smoke real amplio con proveedor/backend lento o vivo tras stop forzado.
- Avance 2026-07-04 noche 9: `runs/control` ya trata `blocked` e `invalid`
  del backend goal-first como estados terminales confirmados; ademas reconoce
  los terminales de proveedor/presupuesto/politica
  `usageLimited`/`quotaLimited`/`providerLimited`/`budgetLimited`/`policyLimited`
  y variantes snake_case. Esto cubre la ruta exacta del app-server tras forced
  stop, que marca el thread goal como `blocked`, y evita degradar un backend ya
  limitado a senal no confirmada. Sigue abierto el smoke real amplio de
  BUG-165.
- Avance anterior 2026-07-04: `BUG-ORQ-20260704-165` queda parcialmente reducido para
  snapshots `stopped` heredados con shutdown activo stale: status normaliza y
  limpia `shutdown_in_progress`, `shutdown_active_work_count/refs`,
  `shutdown_async_work_active` y timeout de parada antes de publicar salida
  `stopped`. Esto no cierra los timeouts de `status/observe` ni la
  reconciliacion goal-first completa. Nuevo caso de campo Sueldos:
  `request-ref-sueldos-cargos-partidos-stop-20260704-001` devuelve
  `estado=ok` pero `final_status=stop_requested`; `observe_goal` sigue
  `goal_status=running` con alto consumo y el tmux `orquesta-goal-*` queda
  vivo hasta limpieza manual.
- Avance 2026-07-04: `BUG-ORQ-20260701-065` publica `recommended_action` en el
  resultado tipado de shutdown, MCP `orquesta.server.shutdown.v0` y el error
  compacto de `orquesta-server stop`; el bug sigue abierto para la coordinacion
  automatica backend/checkpoint/stop/cancel/wait.
- `BUG-ORQ-20260704-169` queda cerrado por `6fe19d06`: el planner de
  automejora ya no interpreta `Dependencias: ninguna` como dependencia real
  pendiente y, con capacidad libre, planifica la tarea ejecutable antes que el
  scanner idle. Esto cierra la regresion T295 observada como tres ciclos no-op
  del scanner; si la via idle vuelve a no ejecutar backlog pendiente, reabrir
  como regresion nueva enlazada. Revision Codex noche 8: un scanner que solo
  deje `## Escaneo backlog ...` sin materializar/actualizar secciones `## Txx`
  ejecutables no debe contarse como trabajo cerrado; si aparece evidencia nueva,
  abrir regresion acotada al contrato scanner -> `Txx pendiente`.
- `BUG-ORQ-20260701-085` queda historico/supersedido por
  `BUG-ORQ-20260704-164` para runtime/write-set: el residual de enforcement
  fuerte ya no cuenta como bug vivo. Solo queda como limite preventivo externo
  si el proveedor/FS no respeta el sandbox o si falta smoke OPES/productivo
  especifico.
- `BUG-ORQ-20260701-088` queda cerrado funcionalmente por el smoke real
  `smoke_goal_first_checkpoint_only_high_consumption_real=ok`; las menciones
  anteriores a "sigue abierto" son contexto historico, con residuales cerrados
  por `BUG-ORQ-20260703-161/162`.
- `BUG-ORQ-20260701-073` queda cerrado por reejeucion real acotada del mismo
  smoke el 2026-07-04:
  `smoke_goal_first_high_consumption_real=ok`,
  `bug088_path=second_artifact_or_partial_artifacts`,
  `recommended_action=review_partial_artifacts` y
  `app_server_tmux_processes_alive=0`. Esto cubre la ruta
  checkpoint/alto consumo -> segundo artefacto recuperable o revision sin
  falso `goal_first_blocked_no_artifacts` ni backend residual.
- `BUG-ORQ-20260703-154` / MEJ-104 queda cerrado tambien en evidencia real
  acotada: servidor temporal con budget diario agotado publico
  `budget_deferred` en `/api/v0/server/status` y
  `/api/v0/autoprogramming/status`, sin lanzar goal Codex. Runbook:
  `docs/runbooks/smoke_autoprogramming_idle_budget_2026-07-04.md`.
- `BUG-ORQ-20260701-075` queda reducido por el cierre de evidencia minima OPES:
  una entrega terminal sin `required-evidence-*` ya no queda como
  `not_settled` generico ni publicable, sino como
  `pendiente_rework_evidencia_minima`, `operational_status=needs_rework`,
  `settlement_status=needs_rework`, `settlement_scope=required_evidence` y
  `settlement_reason=required_evidence_missing`. El avance 2026-07-04 noche
  cubre la secuencia OPES completa: cada `work_kind` terminal sin evidencia
  minima produce rework causal `review_director_consolidation` con
  `rework_reason=required_evidence_missing`,
  `required_evidence_missing_refs`, `publication_status` no publicable y
  `recommended_action=review_required_evidence`. Sigue abierto para validadores
  OPES semanticos/editoriales por artefacto canonico y smoke OPES temporal
  end-to-end.
- Avance 2026-07-04 noche 5: `BUG-ORQ-20260701-079` queda reducido por corte
  semipreventivo de `thread/read` en el backend `app_server`: respuestas
  WebSocket y lineas stdout del protocolo command de `thread/read` mayores de
  256 KiB fallan antes de reservar/decodificar el payload completo con issue
  `codex_app_server_thread_read_response_too_large`; los RPCs no `thread/read`
  conservan sus limites globales. Sigue abierto porque esto corta la ingesta en
  Orquesta, pero no impide que el proveedor o runtime genere la salida gigante
  antes de responder.
- Avance 2026-07-04 noche 4: `BUG-ORQ-20260701-058/066` queda reducido para
  criterios `done/settled` de texto OPES: un tema con
  `settlement_status=settled_text`, QA de tema completa, sin rework pendiente y
  con checkpoint lifecycle requerido ya satisfecho publica
  `proposed_status=texto_asentado_pendiente_derivados` y
  `operational_status=waiting`, no `en_progreso_orquesta/working`. Asi el
  registro distingue texto asentado de trabajo aun escribiendose y deja
  pendientes los derivados sin promover a paquete final completo.
- Avance 2026-07-04 noche 13: `BUG-ORQ-20260701-058/075` queda reducido para
  bancos de preguntas OPES. `orquesta-opes-director` incorpora
  `ValidateOPESQuestionBankQualityContractV0` y lo cablea en
  `update_topic_registry`: una entrega `question_bank` con evidencia nominal
  `opes-final-evidence:question_bank_publicable` ya no cierra si el contrato no
  prueba al menos 50 preguntas, 4 opciones, una unica respuesta correcta,
  explicacion tutor, informe estructural, informe de dificultad/proximidad y
  revision triple Codex/Gemini/Claude. El registro publica
  `proposed_status=pendiente_rework_tests`,
  `operational_status=needs_rework`,
  `settlement_scope=question_bank_quality`,
  `settlement_reason=question_bank_quality_contract_failed` y crea rework causal
  `review_director_consolidation` con
  `recommended_action=review_question_bank_quality`. La ausencia total de
  evidencia minima sigue usando el camino previo `required_evidence_missing`.
  Evidencia: `TestValidateOPESQuestionBankQualityContractV0*`,
  `TestProduceOPESCausalJobsV0QuestionBankConEvidenceRefPeroContratoFallidoCreaReworkV0`,
  `TestProduceOPESCausalJobsV0QuestionBankConContratoPassNoCreaReworkV0`,
  `go test -count=1 ./modulos/orquesta-opes-director` y
  `go test -count=1 ./modulos/orquesta-opes-director ./modulos/orquesta-opes-bridge`;
  validacion global `go test -count=1 ./...`.
  Siguen abiertos el smoke OPES temporal end-to-end y la matriz completa de
  validadores semanticos/editoriales por artefacto canonico.
- Continuacion Codex 2026-07-04 tarde 3: `BUG-ORQ-20260701-079` queda reducido
  en bordes locales adicionales del app-server: el lector JSON-RPC legacy usa
  el mismo presupuesto de linea de 256 KiB que el protocolo command, `stderr`
  de comandos queda retenido como cola acotada para diagnostico y los logs de
  app-server se inspeccionan por tail acotado antes de clasificar errores de
  proveedor. Tests:
  `TestCodexAppServerLegacyRPCReaderResponseBudgetV0`,
  `TestCodexAppServerCommandProtocolStderrDiagnosticoAcotadoV0` y
  `TestCodexAppServerDiagnosticTailFileV0LeeSoloColaV0`. Sigue abierto el
  enforcement duro pre-tool dentro del runtime/proveedor.
- Continuacion Codex 2026-07-04 tarde 3: `BUG-ORQ-20260701-079` y el residual
  operativo de `BUG-ORQ-20260704-165` quedan reducidos en
  `autoprogramming/status`: un goal durable `running`, backend observado
  `active`, alto consumo y cero checkpoint/artefactos/receipts ya produce
  `stale_running.code=goal_active_no_checkpoint_high_consumption` con
  `recommended_action=replan_narrow_context`, sin degradarse a solo
  `observe_goal`. Test:
  `TestMCPAutoprogrammingStatusExecutorV0GoalRunningHighConsumptionNoDegradaAObserveGoal`.
- Continuacion Codex 2026-07-04 tarde 3: `BUG-ORQ-20260701-065` /
  `BUG-ORQ-20260704-165` quedan reducidos con test combinado de cliente
  shutdown para dos goals activos: `waiting_checkpoint` ->
  `waiting_drain` -> `backend_still_running` -> `ready`, siempre con
  `cleanup_goal_backends=true` y acciones finales `cleanup_completed` no
  bloqueantes. Test:
  `TestRequestServerShutdownV0CoordinaDosGoalsActivosHastaGoalActionsResueltasV0`.
  No sustituye al smoke real amplio con proveedor/status lento.
- Continuacion Codex 2026-07-04 tarde 3: `BUG-ORQ-20260701-058/066` queda
  reducido en el conector REST OPES: un receipt con `CompleteJob=true` acepta
  estados terminales nativos `completed`, `done` y `settled`, y mantiene
  `pending` como invalido. Test:
  `TestRESTClientV0SubmitDomainWorkArtifactAceptaEstadosTerminalesNativosOPES`.
  Sigue abierto el lifecycle OPES end-to-end con instancia temporal y
  external-work.
- Continuacion Codex 2026-07-04 tarde 3: TAREA-6/MEJ-106 no se endurece a
  `511` porque la metrica real actual de
  `scripts/orquesta_metricas_deuda.sh --json` es `env_vars_orquesta=513` y ya
  existe `env_vars_budget_test.go` con ese ratchet. El pendiente verificable es
  consolidar o retirar dos nombres `ORQUESTA_*` reales y solo entonces bajar el
  ratchet a `511`; cambiarlo ahora introduciria un rojo falso en `go test ./...`.
- Continuacion Codex 2026-07-04 tarde 4: TAREA-6/MEJ-106 queda reducida a
  `env_vars_orquesta=511` tras retirar de la superficie de entorno dos refs de
  capacidad configurables no usadas por operadores,
  `ORQUESTA_CAPACITY_MODEL_REF` y `ORQUESTA_CAPACITY_QUOTA_REF`. El stack
  conserva refs internas por defecto para decisiones de capacidad y mantiene
  configurables `ORQUESTA_CAPACITY_POLICY_REF` y `ORQUESTA_CAPACITY_POOL_REF`.
  Se anade `TestEnvVarsOrquestaRatchetMEJ106V0` en `cmd/orquesta-server` para
  fallar si el conteo sube sobre 511 sin marca explicita
  `env_vars_orquesta_allow_increase_to=<valor>` en este inventario o bitacora.
- Continuacion Codex 2026-07-09: TAREA-8.4 retira el alias legacy
  `ORQUESTA_CODEX_SMOKE_TIMEOUT_SECONDS`; los smokes Codex reales quedan solo
  con `ORQUESTA_CODEX_SMOKE_TIMEOUT_MS`, y los ejemplos convierten segundos a
  milisegundos. La metrica baja a `env_vars_orquesta=513`; el techo temporal
  queda en `env_vars_orquesta_allow_increase_to=513` por las dos envs
  operativas de Telegram (`enabled` y `token`) que no se eliminan en este
  corte. Pendiente: bajar a 511 solo cuando esas entradas pasen a config/secreto
  gestionado sin romper el canal operador.
- Continuacion Codex 2026-07-09: las dos envs temporales Telegram operator se
  retiran y toda la familia queda solo en `telegram_operator.*` dentro de
  `orquesta.config.json`. `notification_target_ref` derivado de
  `authorized_chat_refs[0]` publica `source=config_file`, no `defaulted`, y
  sigue redactado. La metrica vuelve a `env_vars_orquesta=511`; se elimina la
  necesidad de `env_vars_orquesta_allow_increase_to`.
- `BUG-ORQ-20260702-120` queda cerrado por la proyeccion
  `stopped/crashed/unreachable` y los contratos OPES asociados. Sus notas de
  avance que decian "no cierra el bug padre" son historicas y quedan
  supersedidas por "Con este avance se cierra BUG-ORQ-20260702-120".

## Ejes de arquitectura a vigilar

Actualizacion S13 2026-07-11: `BUG-ORQ-20260711-219` permanece abierto como
incidencia documental/operativa. El manifiesto de retencion ampliada contenia
25 `candidato_borrar` catalogados por JSON; se reclasifican a
`archivar_condicionado`. La reconciliacion distingue 68 rutas en auditoria
estricta, 103 en retencion ampliada, 61 comunes, 42 exclusivas del ampliado y 7
omitidas por este. No se mueve ni borra nada: la accion pendiente es comprobar
referencias por ruta y destino trazable en una ola gobernada. Evidencia:
[incidencia 219](incidencias/incidencia_orquesta_s13_clasificacion_borrado_referenciada_2026-07-11.md).

| Eje | Sintoma repetido | Riesgo arquitectonico |
| --- | --- | --- |
| Goal-first vs loop legacy | rutas que caen al director antiguo, modos implicitos, tests que no fijan opt-in | migracion incompleta y doble fuente de verdad del ciclo |
| Estado vivo/proyecciones | running_stale, procesos vivos no contados, markers goal missing, ACK sin cierre | estado repartido entre filesystem, stores, observadores y colas |
| Runtime Codex | stdio/proxy/tmux confundidos, timeouts, bwrap, identidad git remota | adaptadores mezclados con politica operativa y entorno |
| OPES/DomainWork | artefactos regenerables tratados como canonicos, RAG/audio/HTML con falsos verdes | contrato de dominio demasiado implicito y validadores divergentes |
| Subagentes/causalidad | padres escriben tabla de subroles sin materializar hijos reales | contrato causal no forzado por runtime/ACK/write-set |
| Web Nueva App | opciones visibles no llegan al contrato, ayuda incompleta, errores en idioma equivocado | catalogos UI, i18n, factory y docs duplican verdad |
| Remoto/produccion aislada | bundle stale, sin GitHub directo, worktrees divergentes | operacion remota no tiene protocolo unico de sync y handoff |

## Bugs inventariados

| ID | Estado | Area | Sintoma | Hipotesis arquitectonica | Evidencia / enlace | Accion |
| --- | --- | --- | --- | --- | --- | --- |
| BUG-ORQ-20260705-191 | cerrado parcial | Remoto/autoprogramacion staging e integracion Git | en `srv1651826`, el goal G5 `request-ref-remoto-wizard-bot-llm-20260705-001` termino con `orquesta_goal_result.v0 status=complete`, tests declarados verdes y cambios de codigo bajo `pilot-remoto-1`, pero el worktree principal/GitHub quedo sin los ficheros G5; el result no traia `commit_sha`; `/api/v0/autoprogramming/status` podia mostrar cola vacia; ademas el worktree principal tenia `origin=/tmp/orquesta-self.bundle`, no GitHub | Orquesta confundia entrega terminal del agente con integracion operacional del cambio; el efecto `pending_push` existia en promocion de staging, pero no estaba elevado a contrato de cierre con `integration_receipt` obligatorio ni a estado visible en `autoprogramming/status`; residual operativo: el remoto necesita remote Git canonico o protocolo oficial bundle/push | incidencia `docs/incidencias/incidencia_orquesta_remoto_g5_complete_sin_integracion_git_2026-07-05.md`; evidencia original: result durable G5 con `commit_sha=null`, worktree piloto con cambios recuperables y remote `/tmp/orquesta-self.bundle`; cierre remoto: tests `TestAutoprogrammingStagingPromotionV0DeclaraReciboIntegracionV0`, `TestCodexStackAutoprogrammingPromotionV0PendingPushExigeReciboIntegracionV0`, `TestMCPAutoprogrammingStatusExecutorV0RunClosedEnColaPublicaPendingIntegrationV0`; bundle remoto `28c562bccf` | Cierre de codigo: `AutoprogrammingStagingEffectResultV0` declara `integration_status`/`integration_receipt_ref`; el stack normaliza `promoted/clean` a `integrated`, `pending_push` a `pending_integration` y `blocked` a `blocked_push`; status publica accion `wait_for_integration_receipt` cuando la run esta cerrada causalmente pero la cola sigue no terminal. Pendiente operativo: validar en servidor con remote Git canonico o protocolo bundle/push de produccion |
| BUG-ORQ-20260704-172 | cerrado | Tooling/higiene de disco | durante la verificacion de OPES 2026-07-04 noche 30, `go test` fallo antes de compilar con `no space left on device` al escribir en `/home/alberto/.cache/go-build`; `/home` estaba al 100%, `go-build` ocupaba 17G y `pkg/mod` 12G | las sesiones largas pueden llenar la cache de build en `/home` y convertir tests verdes en fallos operativos si no se presupuestan caches temporales aisladas | `df -h /home/alberto /tmp` mostro `/home` 100%; `du -sh /home/alberto/.cache/go-build /home/alberto/go/pkg/mod` mostro 17G y 12G; `go clean -cache` dejo 17G libres; los tests OPES se reejecutaron con `GOCACHE=/tmp/orquesta-codex-gocache` y `GOTMPDIR=/tmp/orquesta-codex-gotmp`; `go test -count=1 ./modulos/orquesta-opes-director ./modulos/orquesta-opes-bridge` verde | Cierre operativo local: cache Go de build limpiada y verificacion reanudada con cache temporal en `/tmp`. Residual estructural: futuras sesiones largas deben declarar `GOCACHE`/`GOTMPDIR` fuera de `/home` o ejecutar limpieza gobernada antes de smokes/builds amplios |
| BUG-ORQ-20260711-208AH | cerrado localmente | Tooling/higiene de disco/reapertura de subagente | al reabrir un subagente fallo `No space left on device`; `/home` estaba al 100% con 0 disponible, aunque `/` tenia 134G y `/tmp` 44G | medir solo `/` o `/tmp` no detecta agotamiento del mount real de `$HOME`; faltaban preflight por mounts, presupuesto de cache y recibos durables. No se atribuye todo el uso a Orquesta ni se autoriza borrar caches ajenas | `514b86c49`; prueba real `bug-208ah-real-20260711` con cuatro mounts y presupuesto de 1 GiB; receipts en `/tmp/orquesta-sessions-208ah-evidence/bug-208ah-real-20260711/`; tests de preflight y batches, `go test -count=1 ./...`. Detalle en la [incidencia 208AH](incidencias/incidencia_orquesta_home_lleno_reapertura_subagente_2026-07-11.md), antecedente [BUG-ORQ-20260704-172](#bug-orq-20260704-172) | cierre: preflight integrado en el perfil aislado, caches privadas con marker y cleanup confirmado limitado al receipt; scripts fuera del perfil deben invocarlo explicitamente |
| BUG-ORQ-20260711-218 | cerrado localmente | Tooling/perfil aislado/lease de puertos | una raiz explicita bajo `/tmp` preparo caches correctamente, pero el lease intento crear `/srv/orquesta-self/runtime/test-cache/port-leases` y aborto por permisos | cache y lease resolvian defaults distintos; la raiz declarada no gobernaba todos los recursos auxiliares de la sesion | reproduccion al lanzar la auditoria viva; test de raiz explicita sin env global; [incidencia 218](incidencias/incidencia_orquesta_isolated_env_lease_default_remoto_2026-07-11.md) | cierre: override explicito conserva autoridad; con cache global se deriva de ella y, sin ella, del padre de la raiz aislada, manteniendo locks compartidos sin escapar a `/srv` |
| BUG-ORQ-20260711-220 | cerrado localmente | Config canonica/runtime_models/secretos | la primera migracion pasaba tests funcionales pero tenia TOCTOU/symlinks, raiz y owner no gobernados, secreto relativo al snapshot, lectura sin limite, URL/timeout laxos, reread best-effort y `enabled:false` ignorado | parser, snapshot, secreto, effective_config y adaptador no compartian una resolucion unica y atestada | `bc5a2cdcf`; pruebas adversariales de path/owner/permisos/tamano/URL/timeout/enabled; suite completa; [incidencia 220](incidencias/incidencia_orquesta_runtime_models_secret_config_insegura_2026-07-11.md) | cierre: seccion tipada `runtime_models`, env override deprecated, token por fichero confinado con `os.Root`, resolucion unica y startup fail-closed |
| BUG-ORQ-20260711-221 | cerrado localmente | Server/startup/readiness/runtime Codex | `orquesta-server run` acepto un `ORQUESTA_CODEX_COMMAND` absoluto inexistente, publico readiness y el primer goal termino `invalid` tras timeout de socket | `start` ejecutaba el preflight Codex y `run` no; readiness normal no demostraba que el backend obligatorio pudiera arrancar | run `run-hermes-config-20260711-001`, log retenido en `/tmp/orquesta-self-hermes-20260711/runtime`; [incidencia 221-225](incidencias/incidencia_orquesta_autoprogramacion_local_preflight_consumo_y_scope_2026-07-11.md) | cierre local: preflight compartido en `run`, despues de preservar `degraded_identity` y antes de construir runtime/listener; focal `CodexCommand` verde |
| BUG-ORQ-20260711-222 | cerrado localmente | Goal-first/consumo/control de proceso | un goal estrecho materializo 65 lineas y test verde, pero supero 1,1 M tokens sin result durable; el observer pidio stop por alto consumo y el proceso siguio hasta control forzado | la politica automatica persistia peticion cooperativa sin demostrar actuacion y ausencia del proceso | goal externo `019f4fa2-e2d1-7e62-9f4c-b3be100a7833`, estado/runtime `state-2`/`runtime-2`; pruebas de ruta tipada, no falso verde, replay y via cooperativa; [incidencia 221-226](incidencias/incidencia_orquesta_autoprogramacion_local_preflight_consumo_y_scope_2026-07-11.md) | cierre: alto consumo exige control tipado confirmado y reutiliza el executor forzado con backend escalator/reobservacion; `Requested` solo queda true tras terminal confirmado |
| BUG-ORQ-20260711-223 | cerrado localmente | Goal-first/observacion HTTP concurrente | `POST /api/v0/autoprogramming/goal/observe` devolvio 500 generico mientras el observer residente observaba el mismo goal con ticks `ok` | timeout/cancelacion por contencion se proyectaba como error interno | HTTP audit en `state-2`, tests de deadline/cancelacion/timeout/error real; [incidencia 221-226](incidencias/incidencia_orquesta_autoprogramacion_local_preflight_consumo_y_scope_2026-07-11.md) | cierre: timeout recuperable devuelve 504 parcial con `observe_later`, sin detalle sensible; error real conserva 500 |
| BUG-ORQ-20260711-224 | cerrado localmente | Autoprogramacion/write-set/promocion | el rework declaro dos ficheros pero modifico tambien `daemon.go`; la integracion manual tuvo que retirar el cambio | faltaba atestacion obligatoria del diff final contra baseline/write-set congelados antes de promocion | E2E dentro/fuera de scope, replay y store durable tras reinicio; [incidencia 221-226](incidencias/incidencia_orquesta_autoprogramacion_local_preflight_consumo_y_scope_2026-07-11.md) | cierre: prepare-run persiste baseline JSON atomico, Goal transporta ref tipada y promotion bloquea commit/archive ante ruta ajena, snapshot parcial/ausente o conflicto |
| BUG-ORQ-20260711-225 | cerrado localmente | Tooling/perfil aislado/umask | la suite aislada produjo falsos rojos en pruebas que crean permisos inseguros deliberados; la verificacion de `226` los reprodujo en dos pasadas | se corrigio la fuga de `orquesta_use_isolated_test_env`, pero `orquesta_test_batches.sh` imponia otro `umask 077`; tres fixtures tampoco fijaban con `Chmod` el modo inseguro que afirmaban probar | fallo retenido en `/tmp/orquesta-bug226-batches-20260711/receipt.json`; cierre en `/tmp/orquesta-bug225-focal-fixed-20260711/receipt.json`; self-test y focales bajo `umask 077`; [incidencia 221-226](incidencias/incidencia_orquesta_autoprogramacion_local_preflight_consumo_y_scope_2026-07-11.md) | cierre: runner restaura umask antes de Go, proceso hijo lo verifica y dos pasadas reales afectadas quedan verdes |
| BUG-ORQ-20260711-226 | abierto/solo prueba empirica | Autoprogramacion/eficiencia/progreso | el goal `019f4fb4-99f7-7632-b2c7-abc209bc4bd2` alcanzo 300.297 tokens sin modificar ningun fichero en una tarea estrecha | checkpoint de inicio y actividad del proveedor no equivalen a progreso material; faltaba presupuesto por tramo con replan temprano | state/runtime retenidos en `/tmp/orquesta-core-residuals-20260711`; contratos, CAS, uso por goal, gobierno, diff, test independiente y proyeccion durable MCP/control en `ab1896d68`..`cb87c6108`; [incidencia 221-226](incidencias/incidencia_orquesta_autoprogramacion_local_preflight_consumo_y_scope_2026-07-11.md) | programacion cerrada; queda goal real util T9104 con consentimiento del operador y `MAX_REQUESTS=1`; no reusar el goal fallido |
| BUG-ORQ-20260711-227 | cerrado empiricamente | Server/shutdown/snapshot causal | tras parar por control el unico goal de T9104, cuatro shutdown conservaron `stop_pending` con contadores a cero; la auditoria reprodujo ademas herencia de async/timeout stale ante `ready` | el snapshot fresco reemplazaba trabajo activo pero conservaba `ShutdownGoalActions`; la fusion tambien revivia async de un intento anterior y el resultado ready no limpiaba su timeout | respuestas iniciales en `/tmp/orquesta-bug226-t9104-runtime`; replay `r2` en `shutdown-bug229.json`; focales y suites server; [incidencia 227-229](incidencias/incidencia_orquesta_t9104_wakeup_material_progress_shutdown_stale_2026-07-11.md) | snapshot invalida acciones anteriores; ready no hereda async y limpia timeout; replay devolvio ready, cleanup_completed y proceso salio solo |
| BUG-ORQ-20260711-228 | cerrado localmente/wiring real confirmado | Composicion/wakeup/progreso material | el primer T9104 vivo publico `material_progress_state_reader_unbound`, dejando governor, MCP y control sin estado material | `serverWakeupGoalStateStoreV0` ocultaba interfaces opcionales del store al activar el relay residente | primer status en `/tmp/orquesta-bug226-t9104-runtime`; replay `r2` sin diagnostico unbound; wiring con relay y suites server; [incidencia 227-229](incidencias/incidencia_orquesta_t9104_wakeup_material_progress_shutdown_stale_2026-07-11.md) | decorador delega reader/CAS sin acoplar Goal; persistencia material completa se revalida tras BUG-229 |
| BUG-ORQ-20260711-229 | cerrado localmente/replay pendiente | Autoprogramacion/progreso material/snapshot | replay `r2` observo uso y diff real pero no creo `material_progress_states` con promocion Git desactivada | snapshot store y grabacion del baseline estaban acoplados a `promotion.Enabled`, aunque el governor solo necesita lectura/verificacion | goal `019f5052-2813-7943-a2f6-3e7137c37cb0`, state/runtime en `/tmp/orquesta-bug226-t9104-r2-*`; focal baseline con promotion disabled y wiring server; [incidencia 227-229](incidencias/incidencia_orquesta_t9104_wakeup_material_progress_shutdown_stale_2026-07-11.md) | snapshot durable siempre que el puerto exista; promotion sigue opt-in; falta replay nuevo con estado material persistido |
| BUG-ORQ-20260711-230 | cerrado localmente/replay pendiente | Server/autoprogramacion/goal-first | replay `r3` siguio lanzando una spec idle sin `worktree_baseline` aunque el snapshot store ya estaba cableado | el scheduler preferia launch directo al prepare-run cuando la composicion implementaba ambos, duplicando compilacion y omitiendo aislamiento/baseline | goal `019f5059-3866-7912-953f-83f63615b6a7`, evidencia en `/tmp/orquesta-bug226-t9104-r3-*`; regresion de precedencia y suites server; [incidencia 227-230](incidencias/incidencia_orquesta_t9104_wakeup_material_progress_shutdown_stale_2026-07-11.md) | capacidad goal-first preparada explicita; server prefiere prepare y launch directo queda solo como compatibilidad; falta replay nuevo |
| BUG-ORQ-20260704-171 | cerrado | Claude process/MCP heredado | un intento real de `claude_process` por servidor sin `--safe-mode` heredo la configuracion local de Claude y arranco `codebase-memory-mcp` como hijo del proceso Claude aunque el smoke no necesitaba consulta de grafo | el wrapper de proveedor real heredaba customizaciones/MCP del HOME de Claude; en Orquesta, `codebase-memory-mcp` debe ser opt-in por broker central y no aparecer como efecto lateral de un smoke de proveedor | observado durante el smoke servidor 2026-07-04 antes del ajuste; cierre validado con `/tmp/orquesta-claude-process-server.x5N8pq`, wrapper `claude -p ... --safe-mode`, `poll=31`, `closure_status=accepted`, y comprobacion posterior sin `orquesta-server run`, `claude_goal_wrapper`, `claude -p`, `codebase-memory-mcp`, `codex app-server` ni `orquesta-goal-*`; test `TestSmokeGoalFirstClaudeProcessServerRealEsOptInYLimpiaBackendV0` exige `SMOKE_CLAUDE_GOAL_PROCESS_SERVER_SAFE_MODE` y `--safe-mode` | Cierre: `scripts/smoke_goal_first_claude_process_server_real.sh` usa `SMOKE_CLAUDE_GOAL_PROCESS_SERVER_SAFE_MODE=1` por defecto, aplica `--safe-mode` en preflight y ejecucion real, y conserva fallback de limpieza por manifiesto interno del runtime |
| BUG-ORQ-20260704-170 | cerrado | Goal-first/proveedor Claude result durable | en el segundo intento real de smoke servidor, Claude escribio `evidence_refs` como objetos `{ref, description}` en vez de strings, y Orquesta proyecto `goal_status=invalid`, `run_status=bloqueada`, `closure_status=blocked`, `summary=claude_goal_result_invalid` pese a que el trabajo era recuperable | el contrato de result durable exigia `[]string`, pero los proveedores tienden a enriquecer refs con descripcion; esa forma es normalizable si conserva `ref` no vacio y no debe invalidar todo el cierre | fallo retenido en `/tmp/orquesta-claude-process-server.aCyaLx`; cierre validado con `/tmp/orquesta-claude-process-server.x5N8pq`, `goal_status=complete`, `run_status=cerrada`, `closure_status=accepted`, 9 `artifact_refs`, 16 `evidence_refs`, 3 refs requeridas en result durable y 1 test requerido `passed`; tests `TestClaudeGoalBackendV0ObserveNormalizaEvidenceRefsObjetoRecuperableV0` y `TestGeminiGoalBackendV0ObserveNormalizaEvidenceRefsObjetoRecuperableV0`; focal `go test -count=1 ./modulos/orquesta-runtime-claude ./modulos/orquesta-runtime-gemini` | Cierre: Claude/Gemini normalizan de forma acotada solo arrays `evidence_refs` cuyos items traen `ref` string no vacio, conservando bloqueo para objetos no recuperables; los prompts de ambos backends exigen arrays de strings y desplazan descripciones a summary/README/handoff |
| BUG-ORQ-20260704-173 | cerrado | Config/env alias pisados | auditoria `docs/auditoria_envs_pisadas_2026-07-04.md`: `ORQUESTA_CODEX_CODE_HOME`/`CODEX_HOME`, `ORQUESTA_OPES_BASE_URL`/`OPES_BASE_URL`, `ORQUESTA_SERVER_URL`/`ORQUESTA_BASE_URL`, timeouts de smoke y `ORQUESTA_GUARDIAN_*` podian coexistir o quedar fuera de eco canonico | habia aliases de compatibilidad sin fuente tipada ni diagnostico de conflicto en `effective_config`; esto repite la clase T7A, pero por nombres multiples del mismo concepto en vez de por proyeccion al daemon | tests `TestServerConfigFromEnvV0DiagnosticaOPESBaseURLLegacyAliasV0`, `TestServerConfigFromEnvV0DiagnosticaOPESBaseURLPisadaV0`, `TestServerConfigFromEnvV0DiagnosticaCodexCodeHomeLegacyAliasV0`, `TestServerConfigFromEnvV0DiagnosticaCodexCodeHomePisadoV0`, `TestServerConfigFromEnvV0DiagnosticaOrquestaBaseURLLegacyAliasV0`, `TestServerConfigFromEnvV0DiagnosticaOrquestaBaseURLPisadaV0`, tests de timeout Claude/Gemini/Codex, tests guardian child env, `TestCodexStackAutoprogrammingPrepareRunAPIV0BloqueaConfigProjectionMismatchV0`, `TestCodexStackAppsDirectorAPIV0BloqueaConfigProjectionMismatchV0`, `TestServerConfigProjectionSettingsForMCPV0ProyectaEffectiveConfigV0`; `go test -count=1 ./...`; smoke Orquesta temporal de `effective_config` y de `required_settings` divergente | Cierre ampliado ola 1 TAREA-8: `ORQUESTA_CODEX_CODE_HOME`, `ORQUESTA_OPES_BASE_URL` y `ORQUESTA_SERVER_URL` son settings canonicos sensibles con alias legacy diagnosticados; `ORQUESTA_CODEX_HOME` queda documentada como HOME del proceso; los timeouts de smoke nuevos usan `_MS` y conservan `_SECONDS` como alias temporal; `ORQUESTA_GUARDIAN_*` emitidas por servidor quedan clasificadas como `child_process`; `prepare-run` y `apps/director` bloquean `required_settings` divergentes con `config_projection_mismatch` antes de lanzar trabajo. Pendiente estructural: fichero canónico `orquesta.config.*`, ratchet bidireccional completo y retirada de aliases legacy tras compatibilidad |
| BUG-ORQ-20260709-205 | cerrado | Config/env alias OPES en scripts | tras cerrar la ola de config canonica, varios wrappers activos de OPES seguian aceptando `OPES_BASE_URL` como entrada publica o mencionandola en errores, aunque el inventario canonico ya declaraba `ORQUESTA_OPES_BASE_URL` | la compatibilidad historica quedo repartida en scripts y podia reintroducir la confusion de operador detectada por `BUG-ORQ-20260704-173` fuera del `effective_config` del servidor | `rg` focal sobre `scripts/smoke_opes_domain_work_real.sh`, `scripts/smoke_opes_visual_asset_real.sh`, `scripts/probe_opes_derivatives_rest_contract.sh`, `scripts/smoke_opes_plan_temario_operadores.sh` y `scripts/smoke_opes_derivatives_rest.sh`; `bash -n` requerido de scripts servidor/nightly; `git diff --check` | Cierre: los wrappers revisados leen solo `ORQUESTA_OPES_BASE_URL` como entrada publica de OPES temporal y actualizan mensajes de error; `OPES_BASE_URL` queda restringida a documentos historicos, tests fake o scripts no migrados que requieren ola separada |
| BUG-ORQ-20260704-174 | cerrado | Config canónica/status REST | al probar `orquesta.config.json` con `autoprogramming.checkpoint_only_high_consumption_tokens=450000`, `/api/v0/autoprogramming/status` seguía publicando defaults `100000/900/600`, aunque el stack MCP y `required_settings` ya tenían la política efectiva | había dos superficies de verdad: el transporte MCP recibía `AutoprogrammingGoalProgressPolicy`, pero el handler REST creado por `orquesta-app-gateway` no la transportaba y construía `MCPAutoprogrammingStatusToolExecutorV0` con policy vacía | primer smoke temporal fallido con status `goal_progress_policy=100000/900/600`; cierre con `TestAutoprogrammingStatusAPIRouteV0PublicaGoalProgressPolicyConfigurada`, `TestServerConfigFromEnvV0LeeUmbralCheckpointDesdeFicheroCanonicoV0`, smoke temporal `orquesta_temp_config_file_smoke=passed`, `git diff --check` y `go test -count=1 ./cmd/orquesta-server` | Cierre: `orquesta-app-gateway.ConfigV0` incorpora `AutoprogrammingGoalProgressPolicy`; `NewAPIRouteHandlersV0` la pasa al executor de status REST; `orquesta-app-codex-stack` cablea la policy desde bindings. Residual: TAREA-8.1 solo cubre por ahora la familia `autoprogramming`; faltan familias completas y ratchet AST estricto |
| BUG-ORQ-20260704-175 | cerrado | Config canónica/ratchet envs | tras registrar config canónica inicial, el AST seguía como diagnóstico no bloqueante y podía volver a subir el número de lecturas `ORQUESTA_*` no registradas sin romper tests | el ratchet global por conteo único no ve si una lectura nueva queda fuera de `serverEffectiveEnvRegistryV0`; hacía falta una baseline de no-incremento antes de poder clasificar toda la deuda por categorías | `TestServerEnvRegistryASTV0LecturasORQUESTARegistradas` pasa de diagnóstico a ratchet con baseline 220; focal `go test -count=1 ./cmd/orquesta-server -run 'FicheroCanonico|EnvExplicitoGana|AuditFileInvalido|EnvRegistryAST|EnvVars|Ratchet'`; métrica `scripts/orquesta_metricas_deuda.sh --json` conserva `env_vars_orquesta=512` | Cierre: se registran `ORQUESTA_SERVER_ADDR`, `ORQUESTA_SERVER_STATE_DIR`, `ORQUESTA_CODEX_RUNTIME_WORKDIR` y `ORQUESTA_SERVER_DAEMON_LOG_RAW_REASON`, el fichero canónico cubre `server`, `daemon_logs` y `codex_runtime`, y el AST falla si la deuda sube por encima de 220. Residual: bajar 220 por fases y separar effective/config, child_process, smoke/test y provider runtime |
| BUG-ORQ-20260704-169 | cerrado | Autoprogramacion idle/scanner | el pilotaje T295 reprodujo que, con `capacity_free` y una seccion pendiente ejecutable que declaraba `Dependencias: ninguna`, el planner devolvia una tarea de `backlog_scan`/fallback en vez del request ejecutable; la cadena idle parecia escanear y declarar no-op sin programar el backlog ya escrito | el parser normalizaba `ninguna` como dependencia real no completada, por lo que la tarea quedaba no ejecutable y el scanner ocupaba el ciclo; faltaba una regresion que impidiera sustituir trabajo pendiente por scanner cuando hay backlog minimo valido | request `request-ref-t295-scanner-noop-20260704-001`; commit `6fe19d06`; test `TestIdleSelfImprovementBacklogPlannerV0BacklogMinimoPendienteNoCedeCicloAlScannerV0`; `go test -count=1 ./cmd/orquesta-server -run 'TestIdleSelfImprovementBacklogPlannerV0(BacklogMinimoPendienteNoCedeCicloAlScanner|RespetaDependencias|PlanificaDependiente|SaltaTareasYaEnCola|AnadeScannerSiTodoEstaEnCola|NoInventaFallbackSiTodoEstaEnCola)'`; `go test -count=1 ./cmd/orquesta-server` | Cierre: `idleSelfImprovementNormalizeDependencyRefV0` descarta marcadores de ausencia de dependencias (`ninguna`, `none`, `sin dependencias`, etc.); el planner conserva el request `backlog_autoprogramming` y no anade scanner si ya hay tarea ejecutable |
| BUG-ORQ-20260704-163 | cerrado | Autoprogramacion/skills curadas | la validacion de skills curadas de MEJ-206 rechazaba rutas absolutas conocidas como `/home/`, `/srv/` o `/tmp/`, pero podia aceptar metadata con rutas absolutas genericas como `/workspaces/...`, `/project/.../private.md`, `C:\Users\...` o UNC `\\server\share\...` | el filtro de contexto reutilizable dependia de una allowlist corta de prefijos locales; una skill propuesta o cargada podia introducir paths privados de otros entornos aunque no contuviera secretos explicitos | observado en revision de handoff Claude MEJ-206; tests `TestValidateAutoprogrammingCuratedSkillCatalogV0RechazaRutasAbsolutasGenericas`, `TestBuildAutoprogrammingSkillDistillationReviewProposalV0RechazaRutaAbsolutaGenerica`, `TestServerCuratedSkillsFromProjectV0IgnoraRutaAbsolutaGenericaV0`; `go test -count=1 ./modulos/orquesta-autoprogramming ./cmd/orquesta-server -run 'Test(ValidateAutoprogrammingCuratedSkillCatalogV0RechazaRutasAbsolutasGenericas|BuildAutoprogrammingSkillDistillationReviewProposalV0RechazaRutaAbsolutaGenerica|ServerCuratedSkillsFromProjectV0IgnoraRutaAbsolutaGenerica)V0?'` | Cierre: el detector de detalle sensible conserva marcadores existentes y detecta rutas absolutas Unix genericas con fichero, prefijos `/workspace(s)`, `/private`, `/volumes`, rutas Windows con unidad y UNC. El loader de `skills/*/SKILL.md` descarta esas entradas antes de inyectar `skill_ref` en goals idle. El guard runtime fuerte de write-set queda cubierto despues por `BUG-ORQ-20260704-164` |
| BUG-ORQ-20260704-164 | cerrado | Goal-first/write-set runtime | el backend `orquesta-runtime-codex-appserver` podia aceptar un resultado `complete` aunque el agente hubiera modificado rutas fuera de `DirectionContract.allowed_write_set`, siempre que el recibo terminal no declarase esas rutas fuera de scope | la politica de `workspace_write_guard` viajaba en el start packet y en validacion de receipt, pero faltaba una verificacion runtime independiente del worktree antes de promover un cierre terminal | cierre residual de `BUG-ORQ-20260701-085`; tests `TestServerCodexAppServerGoalBackendV0RuntimeWriteSetGuardBloqueaCambioFueraDeScope`, `TestServerCodexAppServerGoalBackendV0RuntimeWriteSetGuardPermiteCambioDentroDeScope`, `TestVerifyWorktreeWriteSetV0RechazaCambioFueraDelWriteSet`, `TestVerifyWorktreeWriteSetV0AceptaCambiosDentroDelWriteSet`, `TestMCPAutoprogrammingStatusExecutorV0RuntimeWriteSetViolationPideReworkV0`, `TestMCPDirectorStatsToolExecutorV0GoalFirstProyectaRuntimeWriteSetViolationV0`, `TestEnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0RuntimeWriteSetViolationPideRework`, `TestMCPDomainWorkStatusHTTPHandlerV0NormalizaSenalesGoalFirstRecuperablesComoBloqueadas` | Cierre: appserver captura baseline del worktree al arrancar goals con `write_set_enforcement=workspace_write_guard`, verifica `VerifyWorktreeWriteSetV0` antes de fusionar un resultado `complete`, bloquea como `codex_app_server_runtime_write_set_violation` si detecta cambios fuera de scope, elimina recibos de dominio/rework refs de ese cierre y publica evidencias compactas por ruta fuera de scope. MCP/status, `director.stats`, `observe_goal`, `efficiency_summary` y `domain-work/status` lo proyectan como bloqueo recuperable con `rework_write_set_violation`. No se ejecuta smoke real OPES/productivo en este cierre |
| BUG-ORQ-20260704-165 | abierto | Goal-first/observabilidad y control Orquesta | durante la limpieza documental 2026-07-04, Orquesta acepto tres goals paralelos y escribio los documentos esperados, pero `/api/v0/autoprogramming/status` y `observe_goal` devolvieron timeouts; antes del reinicio con backend, un `runs/control stop forced=true` sobre T260 devolvio `control_not_propagated_to_goal_backend` pese a no observarse `codex app-server` local vivo; al cierre, `orquesta-server stop --force --reason ...` no devolvio en mas de 60s y el servidor publico `shutdown_in_progress` con active works stale hasta cortarlo por SIGINT local | la ejecucion goal-first puede avanzar por backend mientras las superficies de observacion/control quedan lentas o no reconcilian un estado stale sin proceso; esto vuelve a obligar a integracion manual para saber si el trabajo termino, fallo o sigue vivo, y puede dejar un state `stopped/degraded` con refs de active work ya sin procesos | observado con request `request-ref-doc-cleanup-historicos-falsos-20260704`, goals `goal-ref-task-autoprogramming-39ecd0187760-g01/g02/g03`, errores `autoprogramming_status_timeout` y `autoprogramming_observe_goal_timeout`; T260 previo `request-ref-autoprogramming-backlog-t260-goal-first-codex-loop-delgado-c13b7862` fallo con `control_not_propagated_to_goal_backend`; limpieza final: SIGINT local dejo sin `orquesta-server run`, `codex app-server` ni tmux `orquesta-goal-*`, pero `orquesta-server status` quedo `stopped/degraded` por snapshot de shutdown activo; avance CLI 2026-07-04 noche 34 con tests `TestRequestServerShutdownV0PostColgadoConsultaStatusAccionableV0`, `TestWaitServerShutdownReadyV0RepostColgadoRespetaDeadlineYDevuelveStatusAccionableV0` | Avance 2026-07-04 noche 25: `observe_active_goals` acota cada `observe_goal`; `autoprogramming/status` publica `safe_actions` `run_control_reconcile_external_cleanup` hacia `/api/v0/runs/control`; y `runs/control` reconcilia `Goal.Status=running` stale cuando la liveness confirma `running_stale_no_process` seguro, sin devolver `control_not_propagated_to_goal_backend`. Avance 2026-07-04 noche 34: el cliente `orquesta-server stop` acota cada POST de shutdown y, si el POST inicial o re-POST se cuelga, consulta `/status` y devuelve cuerpo accionable `shutdown_not_ready` con active work compacto en vez de esperar el timeout global sin informacion. Pendiente: validar smoke real amplio de observabilidad/control lento y coordinacion completa con backend/proveedor |
| BUG-ORQ-20260704-165 | cerrado | Goal-first/observabilidad y control Orquesta | actualizacion Sueldos 2026-07-04: el goal de cargos/partidos `run-spec-...8b296...` alcanzo `codex_app_server_goal_status_active_high_token_usage` con `tokens_used=183001`; Orquesta publico `evidence-ref-goal-observer-high-consumption-stop-requested`, pero `POST /api/v0/runs/control` con `forced=true` devolvio `estado=ok`, `final_status=stop_requested`, `goal_status_before=blocked`, `goal_status_after=blocked`, mientras `/api/v0/apps/director/goal/observe` seguia publicando `goal_status=running` y el tmux `orquesta-goal-807a9ad83922105c` continuaba vivo | run-control, observer y estado real del backend no comparten una transicion terminal unica: una superficie ve `blocked`, otra `running`, y el proceso app-server permanece aunque se pidio stop forzado por consumo alto; esto deja al operador sin senal fiable para continuar o limpiar sin inspeccion manual | request `request-ref-sueldos-cargos-partidos-20260704-004`; observes `request-ref-sueldos-cargos-partidos-observe-20260704-005/006/007`; stop `request-ref-sueldos-cargos-partidos-stop-20260704-001`; tmux `orquesta-goal-807a9ad83922105c`; estado parcial de app bajo `generated-apps/mapa-de-gasto-publico` con receipt placeholder invalid; avance 2026-07-04 noche 6 cubierto por `TestMCPObserveAppDirectorGoalToolExecutorV0TimeoutSnapshotNoPublicaRunningSiRunControlTerminalV0` y `TestCodexStackObserveAppDirectorGoalExecutorV0TimeoutSnapshotRespetaRunControlTerminalV0` | Avance: la ruta snapshot/timeout de `observe_goal` ya no publica `running` si `RunControl` esta `stopped/canceled`; proyecta bloqueo replanificable con evidencia. Pendiente: smoke real con backend/proveedor lento o proceso vivo tras stop forzado, y propagacion completa del stop cuando el backend siga activo |
| BUG-ORQ-20260704-165 | cerrado | Goal-first/control Sueldos forced stop | cierre 2026-07-04 noche 11: se reproduce el caso bloqueante de Sueldos con backend `app_server_tmux` vivo, alto consumo y `runs/control stop forced=true`; tras el fix, control devuelve `estado=ok`, `status=stopped`, `final_status=stopped`, `goal_status_after=blocked`, el observe posterior devuelve `goal_status=blocked`, `closure_status=blocked`, `recommended_action=replan`, y el cleanup no deja procesos ni tmux `orquesta-goal-*` | faltaban dos cierres de fuente de verdad: si el observer ya habia bloqueado el goal por alto consumo, run-control debia completar el control forzado aunque el goal ya no estuviera `running`; y el shutdown forzado del backend tmux no podia depender de un contexto ya agotado antes de limpiar proceso/socket | runbook `docs/runbooks/smoke_goal_first_forced_stop_backend_real_2026-07-04.md`; smoke real `smoke_goal_first_forced_stop_backend_real=ok`; `run_ref=run-spec-smoke-goal-first-bug088-req-smoke-goal-first-bug088-116f51fcff09efa9aa525ee7dfd94ce7`; `external_goal_ref=019f2c28-d0c3-7551-9972-dcba0f3daeb2`; evidencia saneada `/tmp/orquesta-goal-first-app-server.Sc7e7K`; tests `TestGoalFirstRunControlPortV0ForcedStopCompletaControlSiObserverYaBloqueoAltoConsumoV0`, `TestServerCodexAppServerGoalBackendV0StopForcedBloqueaGoalYApagaBackendV0`, `TestSmokeGoalFirstForcedStopWrapperEjercitaRunControlBackendVivoV0`; `go test -count=1 ./...`; `git diff --check` | Cierre de la ruta Sueldos: nuevo wrapper `scripts/smoke_goal_first_forced_stop_backend_real.sh`, modo interno `SMOKE_GOAL_FIRST_FORCED_STOP_MODE`, fast-path de run-control para goal ya terminal/bloqueado, puerto opcional `ShutdownForcedStopV0` y cleanup tmux con timeout fresco. Residual separado: `BUG-165` global sigue abierto para `status/observe` lento no cubierto por este smoke y para coordinacion automatica completa de shutdown/backend/checkpoint/stop/cancel/wait |
| BUG-ORQ-20260704-165 | cerrado | Goal-first/control Claude process forced stop | cierre 2026-07-04 noche 21: se valida la ruta equivalente con backend externo `claude_process` vivo, lanzado por `cmd/orquesta-server`, manifiesto interno presente y `runs/control stop forced=true`; el control devuelve `estado=ok`, `status=stopped`, `final_status=stopped`, `goal_status_before=running`, `goal_status_after=blocked`, `goal_control_signal_confirmed=true`; el observe posterior devuelve `goal_status=blocked`, `run_status=bloqueada`, `closure_status=blocked`, `recommended_action=replan`; el shutdown final deja `status=stopped`, `shutdown_status=stopped`, `shutdown_ready=true` y no quedan procesos | tras cerrar Codex/app-server, faltaba evidencia real de que el puerto neutral de control tambien terminaliza un proveedor externo de proceso no Codex y que el cleanup no depende de inspeccion manual | runbook `docs/runbooks/smoke_goal_first_provider_process_real_2026-07-04.md`; smoke real `smoke_goal_first_claude_process_forced_stop_server_real=ok`; `run_ref=run-spec-smoke-claude-process-server-req-smoke-claude-process-server-dff02568d5f2f00ac86d8f91237536c7`; `external_goal_ref=claude-goal-06148fd846fd9f64`; evidencia retenida `/tmp/orquesta-claude-process-server.GnFFpX`; evidencias `evidence-ref-claude-goal-process-stop-completed`, `evidence-ref-run-control-goal-forced-stop-terminal`, `evidence-ref-run-control-terminal-after-goal-forced-stop`; test guard `TestSmokeGoalFirstClaudeProcessServerRealEsOptInYLimpiaBackendV0` | Cierre de la ruta Claude process: `scripts/smoke_goal_first_claude_process_server_real.sh` gana `SMOKE_CLAUDE_GOAL_PROCESS_SERVER_CONTROL_MODE=forced_stop`, espera proceso vivo por manifiesto, llama `/api/v0/runs/control`, valida observe posterior y comprueba cero procesos. Residual separado: Gemini real bloqueado por tier/credencial y observabilidad global larga si reaparece |
| BUG-ORQ-20260704-165 | cerrado | Goal-first/observabilidad y control Orquesta | actualizacion 2026-07-04 noche: el observador residente de goals podia quedar bloqueado en una llamada lenta a `ObserveActiveGoalWorksV0`, impidiendo publicar una causa accionable en el state | el tick residente dependia de que cada backend retornase pronto; sin deadline propio, una observacion lenta degradaba `status/observe` y mezclaba progreso real con bloqueo de observabilidad | tests `TestRuntimeV0GoalObservationTickTimeoutPublicaErrorAccionableV0`, `TestRuntimeV0GoalObservationTickTimeoutNoBloqueaSiBackendIgnoraContextoV0`, `TestRunSelfWatchdogTickV0RespetaGoalObserverBackendActivoV0`, `TestNormalizeConfigV0ObservadorGoalFirstActivoPorDefectoV0`, `TestServerConfigFromEnvV0ObservadorGoalFirstResidentePorDefectoV0`; `go test -count=1 ./modulos/orquesta-server`; `TestEnvVarsBudgetMEJ106V0` conserva el presupuesto de envs | Avance: se anade `GoalObserverTimeout` interno con default 2000 ms, configurable por composicion `ConfigV0` sin crear env nueva; la llamada backend queda aislada con deadline duro, publica `goal_observer_timeout` aunque el backend ignore `context.Context`, nuevos ticks no abren llamadas infinitas y publican `goal_observer_backend_call_in_flight`; self-watchdog conserva la llamada backend como causa operacional viva. Residual: falta smoke real amplio de proveedor/status lento y coordinacion completa con `runs/control`/backend |
| BUG-ORQ-20260704-165 | cerrado | Goal-first/start daemon env autoprogramming | reproduccion 2026-07-04 tarde Codex T7A: se arranco un servidor temporal con `ORQUESTA_AUTOPROGRAMMING_CHECKPOINT_ONLY_HIGH_CONSUMPTION_TOKENS=450000`, pero el daemon publico `100000 defaulted`; el goal quedo bloqueado por `codex_app_server_goal_status_active_high_token_usage` con `tokens_used=121675`, aunque el operador habia subido el umbral | `orquesta-server start` filtraba el entorno por allowlist y no proyectaba `ORQUESTA_AUTOPROGRAMMING_*`; ademas el filtro anti-secretos bloqueaba la clave publica `...HIGH_CONSUMPTION_TOKENS` por contener la subcadena `TOKEN`, aunque es un contador, no un secreto | tests `TestServerDaemonStartEnvironmentV0ProyectaPoliticaAutoprogramacion`, `TestServerConfigFromEnvV0PublicaUmbralCheckpointGoalConfigurableV0`, `TestServerCodexGoalBackendFromEnvV0TmuxNoArrancaAppServerEnConstruccionV0`, `TestServerDaemonStartEnvPolicyV0PublicaCategoriasSinValoresCrudos`; smoke manual aislado `/tmp/orquesta-envcheck-20260704T164447` confirmo state efectivo `450000 explicit`, `900 explicit` y categoria `autoprogramming`; `scripts/orquesta_metricas_deuda.sh --json` conserva `env_vars_orquesta=511`; `go test -count=1 ./cmd/orquesta-server` | Cierre acotado: la allowlist de daemon incluye prefijo de autoprogramacion como categoria publica `autoprogramming`, `daemonStartEnvBlockedKeyV0` exceptua solo la clave exacta de conteo `envAutoprogrammingCheckpointOnlyHighConsumptionTokensV0`, y el test del backend tmux se actualiza para el nuevo starter envuelto por `serverCodexGoalCostRoutingStarterV0`. Residual: el `BUG-165` global sigue abierto para observabilidad/control largo, `backend_still_running` en shutdown de goals reales y smoke amplio con proveedor lento |
| BUG-ORQ-20260704-166 | cerrado | Nueva App/goal-first checkpoint invalido y recibo durable no reconciliado | al crear `/home/alberto/Trabajo/Sueldos/generated-apps/mapa-de-gasto-publico` via `POST /api/v0/apps/director` con `director_execution_mode=goal_first`, Orquesta acepto el run y lanzo el goal, pero primero solo materializo `checkpoint_started.txt` y un `orquesta_goal_result.v0` con `status=invalid` y `missing_refs=[source_tree,handoff_report,technical_stack_manifest,go_app,tests]`; en el reintento real del 2026-07-04 ya existia un `orquesta_goal_result.v0` durable `status=complete`, `artifact_refs=3`, `artifact_paths=32`, `materialized_artifacts=3`, `missing_refs=[]` y test `passed`, pero `/api/v0/apps/director/goal/observe` devolvia `estado=error`, `run_status=activa`, `goal_status=running`, `closure_status=blocked`, `summary=codex_app_server_goal_status_active`, `partial_artifacts_written` y `codex_goal_observation_rejected`; reintento posterior al fix con `request-ref-sueldos-gasto-publico-osm-observe-after-fix-20260704-001` devuelve `estado=ok`, `run_status=cerrada`, `goal_status=complete`, `closure_status=accepted`, `closure_accepted=true` y `recommended_action=no_action_closed` | la causa reproducida fue que el scanner de resultados podia gastar el limite de ficheros en `.gocache` o `.gocache-local` antes de encontrar el receipt terminal; tras `codex_app_server_tmux_pane_exit_timeout`, el observer acepta el recibo `complete` si el write-set valida, y conserva la ruta `codex_app_server_runtime_write_set_violation` para bloqueos reales de write-set | `docs/incidencia_sueldos_goal_first_app_invalid_checkpoint_2026-07-04.md`; run `run-spec-mapa-de-gasto-publico-req-mapa-de-gasto-publico-43c20b877af23ecc450d324a25c1e616`; goal `goal-ref-app-director-run-spec-mapa-de-gasto-publico-req-mapa-de-gasto-publico-43c20b877af23ecc450d324a25c1e616`; external goal `019f2a37-df1f-7e00-a60b-dab168d79d38`; request previo `request-ref-sueldos-gasto-publico-osm-observe-retry-20260704-001`; request de cierre `request-ref-sueldos-gasto-publico-osm-observe-after-fix-20260704-001`; fichero `/home/alberto/Trabajo/Sueldos/generated-apps/mapa-de-gasto-publico/docs/orquesta_goal_result_goal-ref-app-director-run-spec-mapa-de-gasto-publico-req-mapa-de-gasto-publico-4.json`; tests previos `TestObserveAppDirectorGoalV0LanzaReworkGoalSiNuevaAppSoloCheckpointInvalidV0`, `TestServerCodexAppServerGoalBackendV0NormalizaMissingRefsRaizEnChecklistV0`, `TestStackGoalMaterializedRefsSourceV0NormalizaMissingRefsRaizEnReceiptDurable`; cierre `ff620ecf` con `TestStackGoalMaterializedRefsSourceV0EncuentraReceiptTerminalTrasCacheVoluminosaV0` | Cierre: `goalMaterializedResultSkipDirV0` y `codexAppServerGoalResultSkipDirV0` saltan `.gocache`/`.gocache-local` ademas de `.git/.codex/node_modules/vendor`; el test reproduce receipt terminal tras cache voluminosa y verifica que no se proyecta `partial_artifacts_written`. El reintento de campo Sueldos/Orquesta cierra el run historico como `accepted`; el active work residual visto al apagar el servidor de prueba queda cubierto por `BUG-ORQ-20260704-165` |
| BUG-ORQ-20260704-167 | cerrado | Runtime worktree/write-set guard | al reintentar Orquesta para ampliar la app Sueldos con cargos publicos, partidos y retribuciones, `POST /api/v0/apps/director` fallo con `arrancar_director_http_error` y `codex_app_server_runtime_write_set_guard_snapshot_failed` antes de lanzar el goal; el workspace de producto estaba en `/home/alberto/Trabajo/Sueldos` y el estado temporal del servidor bajo `.orquesta-feature-cargos` quedaba dentro del mismo arbol | el snapshot runtime excluia prefijos de control Orquesta conocidos, pero no una variante nueva `.orquesta-*`; al capturar baseline de todo el worktree para proteger el write-set, podia incluir estado/audit/runtime local del propio servidor y fallar o contaminar la auditoria del producto | request fallido `request-ref-sueldos-cargos-partidos-20260704-002`; reintento validado `request-ref-sueldos-cargos-partidos-20260704-003` con `estado=ok`, `goal_status=running`; state temporal `/home/alberto/Trabajo/Sueldos/.orquesta-feature-cargos`; test `TestCaptureWorktreeSnapshotV0ExcluyeControlFilesPorDefectoV0`; `go test -count=1 ./modulos/orquesta-runtime-worktree`; `go test -count=1 ./modulos/orquesta-runtime-codex-appserver`; `go test -count=1 ./cmd/orquesta-server` | Cierre: `worktreeDefaultControlPrefixesV0` incluye `.orquesta`, lo que activa la regla existente de prefijos versionados y excluye cualquier primer segmento `.orquesta-*` como `runtime_control_dir`; la regresion cubre `.orquesta-feature-cargos/state/orquesta_server_state_v0.json`; el reintento de campo ya no falla en `codex_app_server_runtime_write_set_guard_snapshot_failed` |
| BUG-ORQ-20260704-168 | cerrado | Goal-first/materialized refs | tras cerrar `BUG-167`, el reintento Sueldos `request-ref-sueldos-cargos-partidos-20260704-003` lanzo goal nuevo `run-spec-...7c369...`, pero `observe_goal` proyecto `goal_status=complete` y `closure_status=blocked` mezclando `artifact_refs`, `required_test_results` y summary del run antiguo `...43c20...`; el unico receipt durable bajo `generated-apps/mapa-de-gasto-publico/docs` tenia `goal_ref=...43c20...` | el escaneo general de refs materializadas usaba una lectura permisiva de `orquesta_goal_result*.json` que aceptaba cualquier receipt terminal `complete` del write-set; la carga terminal estricta ya validaba `goal_ref`, pero el escaneo/repair podia contaminar el estado del run nuevo con artefactos/tests antiguos | request `request-ref-sueldos-cargos-partidos-20260704-003`; observe `request-ref-sueldos-cargos-partidos-observe-20260704-001`; validacion de campo `request-ref-sueldos-cargos-partidos-20260704-004` y observes posteriores sin mezclar refs del run `...43c20...`; receipt antiguo `generated-apps/mapa-de-gasto-publico/docs/orquesta_goal_result_goal-ref-app-director-run-spec-mapa-de-gasto-publico-req-mapa-de-gasto-publico-4.json`; test `TestStackGoalMaterializedRefsSourceV0IgnoraReceiptTerminalDeOtroGoalV0`; `go test -count=1 ./modulos/orquesta-app-codex-stack`; `go test -count=1 ./modulos/orquesta-mcp`; `go test -count=1 ./cmd/orquesta-server` | Cierre: el escaneo de `orquesta_goal_result*.json` usa una lectura scannable ligada al estado: acepta receipts legacy sin `goal_ref`, pero rechaza receipts con `goal_ref` no vacio y distinto al goal actual; la carga terminal estricta se mantiene. El reintento de campo ya no mezclo artefactos/tests antiguos; el bloqueo posterior por alto consumo queda separado en `BUG-ORQ-20260704-165` |
| BUG-ORQ-20260702-139 | cerrado | Goal-first/startup backend | tras limpiar y reiniciar `orquesta-server-latest` con HEAD, el servidor volvia a levantar `codex app-server` para un run goal-first ya `blocked`/`goal_backend_state_unreconciled`, antes de que hubiera trabajo vivo nuevo | la construccion del backend `app_server_tmux` hacia preflight eager y arrancaba el proceso como efecto lateral de montar runtime; startup/readiness, observacion y ejecucion compartian un mismo "asegurar backend" aunque solo launch/observe real deberian materializar proceso | observado en `server-latest` 2026-07-02 tras cleanup gobernado; tests `TestServerCodexGoalBackendFromEnvV0TmuxNoArrancaAppServerEnConstruccionV0`, `TestServerGoalObservationFingerprintFromBackendV0EsOptInV0`, `TestCodexAppServerGoalBackendFingerprintDetectaCambioDeEstadoV0`; `TMPDIR=$PWD/.tmp-go go test -count=1 ./cmd/orquesta-server -run 'TestServerCodexGoalBackendFromEnvV0TmuxNoArrancaAppServerEnConstruccionV0|TestServerGoalObservationFingerprintFromBackendV0EsOptInV0|TestCodexAppServerGoalBackendFingerprintDetectaCambioDeEstadoV0|TestObserveActiveGoalWorksV0'` | Cierre: `app_server_tmux` queda lazy en el wiring de servidor: construir runtime conserva launcher/observer/shutdown hook y diagnostico, pero no ejecuta `Ensure` ni `Probe` hasta una operacion real sobre el backend; el shutdown hook sigue disponible para limpiar restos propios |
| BUG-ORQ-20260703-141 | cerrado | Server lifecycle/status | el state aislado `server-latest` podia quedar `status=stopped` tras shutdown/timeout pero conservar `startup_ready=true` y `startup_status=startup_ready`, de modo que lectores directos del state veian una mezcla de servidor parado y startup consumible | las transiciones `MarkStoppedV0` y `MarkRuntimeStoppedV0` actualizaban el estado de servidor/shutdown pero heredaban campos de startup de una epoca anterior; estado vivo y readiness quedaban parcialmente desacoplados tras parada | observado en `/srv/orquesta-self/runtime/server-latest/state/orquesta_server_state_v0.json` el 2026-07-03; tests `TestStatusTrackerStoppedV0LimpiaStartupReadyV0`, `TestStatusTrackerRuntimeStoppedV0LimpiaStartupReadyV0`; `go test -count=1 ./modulos/orquesta-server -run 'TestStatusTracker(Stopped|RuntimeStopped)V0LimpiaStartupReadyV0|TestServerAvailabilityV0ExponeStoppedAccionableV0'` | Cierre: ambas transiciones a `stopped` limpian `StartupReady=false` y fijan `StartupStatus=stopped`, conservando la proyeccion accionable de disponibilidad `server_stopped` |
| BUG-ORQ-20260703-141 | cerrado | Server lifecycle/status | actualizacion 2026-07-03: el state aislado `server-latest` heredado seguia persistido como `status=stopped`, `startup_ready=true`, `startup_status=startup_ready` mientras el puerto `127.0.0.1:18787` no respondia y el log terminaba con `code-context-watchdog: shutdown_timeout` | las transiciones nuevas ya estaban corregidas, pero los snapshots heredados necesitaban reconciliacion al consultar `orquesta-server status`; si no, un operador podia leer readiness antigua desde statefile o cache temporal | test `TestStatusServerCommandV0ReconciliaStoppedConStartupReadyHeredado`; `go test -count=1 ./cmd/orquesta-server -run 'TestStatusServerCommandV0ReconciliaStoppedConStartupReadyHeredado|TestStatusServerCommandV0ReconciliaStatefileConPIDMuerto'` | Cierre residual: `orquesta-server status` normaliza snapshots `stopped` heredados, persiste `startup_ready=false` y `startup_status=stopped`, y publica salida snapshot `stopped` sin filtrar addr/path. No toca servidores vivos ni production |
| BUG-ORQ-20260703-141 | cerrado | Server lifecycle/status | actualizacion 2026-07-03: tras normalizar `startup_ready`, el payload de status aun podia conservar `startup_message`, `startup_operational_message` y evidencias antiguas con razon `startup_ready` en snapshots `stopped` heredados | limpiar solo el booleano dejaba senales narrativas contradictorias para consumidores compactos y paneles que leen mensajes/evidencias en vez del campo principal | tests `TestStatusServerCommandV0ReconciliaStoppedConStartupReadyHeredado`, `TestStatusTrackerStoppedV0LimpiaStartupReadyV0`, `TestStatusTrackerRuntimeStoppedV0LimpiaStartupReadyV0`; `go test -count=1 ./cmd/orquesta-server -run 'TestStatusServerCommandV0ReconciliaStoppedConStartupReadyHeredado'`; `go test -count=1 ./modulos/orquesta-server -run 'TestStatusTracker(Stopped|RuntimeStopped)V0LimpiaStartupReadyV0'` | Cierre residual: las transiciones `stopped` y la reconciliacion de snapshots heredados limpian tambien mensaje operacional, evidencias, blockers y revision de startup, evitando publicar `startup_ready` narrativo cuando el servidor esta parado |
| BUG-ORQ-20260703-141 | cerrado | Server lifecycle/status | actualizacion 2026-07-03: la primera reconciliacion residual no limpiaba snapshots `stopped` que ya tenian `startup_ready=false` y `startup_status=stopped`, pero conservaban `startup_message`/`startup_operational_message` antiguos; se reprodujo en `server-latest` tras ejecutar `orquesta-server status` | el predicado de "ya normalizado" solo miraba booleano/status y no los campos narrativos/evidencias de startup, dejando restos contradictorios en statefile y payload | test `TestStatusServerCommandV0ReconciliaStoppedConStartupReadyHeredado`; `go test -count=1 ./cmd/orquesta-server -run 'TestStatusServerCommandV0ReconciliaStoppedConStartupReadyHeredado'` | Cierre residual: `NormalizeStoppedServerSnapshotV0` considera limpio un snapshot stopped solo si tambien estan vacios mensaje, operational_message, blockers, evidencias y revision de startup; el test usa `startup_ready=false` para cubrir el caso real observado |
| BUG-ORQ-20260703-142 | cerrado | Tests/cmd orquesta-server T90 | `go test -count=1 ./cmd/orquesta-server` fallo en `TestResidualGoFileBudgetT90V0` porque `cmd/orquesta-server/codex_goal_app_server_v0.go` quedo por encima de 900 lineas y varios ficheros superan o crecen sobre baseline | el baseline de troceo T90 y el handoff de cierre de sesion quedaron desalineados con el arbol actual; un paquete que se reportaba verde vuelve a bloquear la validacion amplia de `cmd/orquesta-server` | observado el 2026-07-03 tras commit `2747f1e`; salida: `baseline T90 roto: cmd/orquesta-server/codex_goal_app_server_v0.go: fichero inmanejable >900 lineas`; cierre: helpers JSON-RPC extraidos a `cmd/orquesta-server/codex_goal_app_server_rpc_v0.go`, `cmd/orquesta-server/codex_goal_app_server_v0.go` queda en 788 lineas; tests `go test -count=1 ./cmd/orquesta-server -run 'TestResidualGoFileBudgetT90V0'` y `go test -count=1 ./cmd/orquesta-server -run 'TestCodexAppServer.*RPC|TestDecodeCodexAppServerRPC|TestServerCodexAppServerGoalBackend'` | Cierre: frente T90 separado sin tocar `cmd/orquesta-server/codex_goal_app_server_tmux_v0.go` ni helpers tmux; queda pendiente del inventario general reducir avisos no bloqueantes de otros ficheros >300 lineas por frentes propios |
| BUG-ORQ-20260703-143 | cerrado | Goal-first/observe receipt repair | `go test -count=1 ./modulos/orquesta-app-codex-stack` fallo en `TestCodexStackObserveAppDirectorGoalExecutorV0RepairReceiptRequiereReworkSinReceiptDominioV0`: tras intentar reparar receipt con artefactos y QA pasada pero sin recibo de dominio, `observe_goal` publicaba `review_partial_artifacts` aunque el cierre ya tenia `repair_receipt_requires_rework` | el enriquecimiento de refs materializadas recalculaba `recommended_action` y daba prioridad a `partial_artifacts_written` frente a un issue estructurado de reparacion de receipt que ya exige rework; el operador podia revisar parciales en vez de replanificar el rework causal requerido, y otras fachadas podian perder o degradar la misma prioridad | tests `TestObserveAppDirectorGoalRecommendedActionV0RepairReceiptRequiresReworkGanaAParciales`, `TestCodexStackObserveAppDirectorGoalExecutorV0RepairReceiptRequiereReworkSinReceiptDominioV0`, `TestMCPAutoprogrammingStatusExecutorV0RepairReceiptRequiresReworkPideReplanV0`, `TestMCPQueueGlobalStatusNormalizeRecommendedActionV0PreservaAccionesGoalFirstEspecificas`, `TestMCPDomainWorkStatusHTTPHandlerV0NormalizaSenalesGoalFirstRecuperablesComoBloqueadas`; `go test -count=1 ./modulos/orquesta-mcp -run 'Test(ObserveAppDirectorGoalRecommendedActionV0RepairReceiptRequiresReworkGanaAParciales|MCPAutoprogrammingStatusExecutorV0RepairReceiptRequiresReworkPideReplan|MCPQueueGlobalStatusNormalizeRecommendedActionV0PreservaAccionesGoalFirstEspecificas|MCPDomainWorkStatusHTTPHandlerV0NormalizaSenalesGoalFirstRecuperablesComoBloqueadas)'`; `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestCodexStackObserveAppDirectorGoalExecutorV0RepairReceiptRequiereReworkSinReceiptDominioV0'` | Cierre: MCP declara `repair_receipt_requires_rework` como issue goal-first estructurado y `observe_goal`, `autoprogramming/status`, `queue/global-status`, `efficiency_summary` y `domain-work/status` conservan accion `replan` antes de la revision generica de parciales, manteniendo ambos issues/evidencias recuperables |
| BUG-ORQ-20260703-140 | cerrado | OPES/proveedores reales | el smoke de revisiones OPES podia crear runs legacy sin `ORQUESTA_AUTOPROGRAMMING_LEGACY_DIRECTOR_LOOP=true`; Claude podia entregar por `DomainWork` y aun asi acabar con 500 posterior del supervisor; Gemini fallaba por `IneligibleTierError` sin diagnostico operativo estable | el borde de proveedores mezclaba tres problemas: opt-in legacy duplicado, estado de entrega aceptada frente a error de supervision posterior, y stderr de proveedor sin contrato estructurado | `external/opes/INCIDENCIA_OPES_SMOKE_PROVEEDORES_CLAUDE_GEMINI_2026-07-03.md`; runbook `docs/runbooks/smoke_opes_reviews_proveedores_real_2026-07-03.md`; tests `TestSmokeOPESReviewsProvidersRealExigeOptInLegacyYWorkdirOPESV0`, `TestCodexStackRunSupervisorErrorResultMCPV0NoFallaProveedorTrasEntregaAceptada`, `TestBuildGeminiWrapperScriptV0DiagnosticaIneligibleTier`, `TestCodexProgressReportWithProcessFailureContextV0ClasificaProviderAuthTierEstructurado` | Cierre: el smoke exige ambos opt-in legacy y documenta `project_work_dir`; el supervisor publica `delivered_with_post_delivery_supervisor_error` como OK accionable si ya hay entrega aceptada; Gemini escribe y Orquesta consume `orquesta_provider_diagnostic_v0.json` con `provider_auth_or_tier_blocked` |
| BUG-ORQ-20260703-144 | cerrado | Runtime aislado/Orquesta no disponible | en `/srv/orquesta-self/runtime/audit-13611445`, tras sincronizar `origin/trabajo/plataforma-agentes` en `0eaf83d`, el tmux `orquesta-server-latest` seguia vivo pero `GET /health`, `/healthz` y `/status` en `127.0.0.1:18787` no devolvian cuerpo; el log de `/srv/orquesta-self/runtime/server-latest/logs/orquesta-server.log` muestra `orquesta-server: orquesta_app_codex_stack: codex_profile.command_path` y ciclos `orquesta-server code-context-watchdog: shutdown_timeout` | el bootstrap del daemon proyectaba `ORQUESTA_CODEX_PATH`, pero `codexCommandPathV0` solo resolvia comandos relativos contra el `PATH` del proceso; si el daemon aislado arrancaba sin el directorio real de `codex` en `PATH`, el perfil Codex recibia `codex` relativo y fallaba antes de servir HTTP/status | observado 2026-07-03 con `curl --max-time` local y `tmux ls`; no hay procesos `codebase-memory-mcp` vivos; test `TestCodexCommandPathV0ResuelveConCodexPathProyectado`; `go test -count=1 ./cmd/orquesta-server -run 'TestCodexCommandPathV0ResuelveConCodexPathProyectado|TestServerDaemonStartEnvironmentV0UsaAllowlistYDerivaRuntime|TestCodexRuntimeConfigV0UsaSandboxWorkspaceWritePorDefecto'` | Cierre: `codexCommandPathV0` resuelve `ORQUESTA_CODEX_COMMAND` relativo primero contra `ORQUESTA_CODEX_PATH` proyectado y solo despues contra `PATH`; conserva el veto del perfil para comandos no resolubles y no abre puertos externos. Validacion viva de `/health`/`/status` queda como siguiente comprobacion operacional del runtime aislado |
| BUG-ORQ-20260703-145 | cerrado | Nueva App/contrato tecnico | el smoke real local `a4QRfx` pidio app Go/net/http desde Nueva App, Orquesta cerro `goal_status=complete`, `run_status=cerrada`, `closure_status=accepted`, `closure_accepted=true`, `artifact_refs=11` y `evidence_refs=11`, pero la app generada fue Python (`pyproject.toml`, `run.py`, `src/smoke_goal_first/...`) | raiz confirmada: `AppSpecRequestV0.preferencias_tecnicas.lenguaje/framework` no se preservaba en `AppSpecV0` y el cierre goal-first solo validaba artefactos/tests genericos; las opciones visibles podian desaparecer antes del contrato ejecutable | smoke conservado en `/tmp/orquesta-goal-first-app-server.a4QRfx`; `observe_response.json` aceptado sin `issues`; tests `TestSolicitarNuevaAppV0PreservaDatosExpertosArquitecturaYAccesibilidadV0`, `TestStartAppDirectorV0GoalFirstLanzaGoalYNoEjecutaLoopLegacy`, `TestObserveAppDirectorGoalV0BloqueaRunSiStackTecnicoContradiceAppSpecV0`, `TestServerNuevaAppHTMLGoalFirstPOSTRenderizaYObservaV0`; `go test -count=1 ./modulos/orquesta-factory`; `go test -count=1 ./modulos/orquesta-app-director-service`; `go test -count=1 ./cmd/orquesta-server -run 'TestServerNuevaAppHTMLGoalFirstPOSTRenderizaYObservaV0|TestSmokeGoalFirstAppServerReal'` | Cierre: `AppSpecV0` publica `technical`, el `GoalWorkSpecV0` de Nueva App anade refs/criterios/manifest de stack tecnico, y `ObserveAppDirectorGoalV0` envuelve la closure para bloquear/rework si una entrega terminal contradice el lenguaje/framework declarado; una entrega Python ya no cierra como aceptada cuando se pidio Go |
| BUG-ORQ-20260703-146 | cerrado | Shutdown/smoke Goal-first | tras el mismo smoke real `a4QRfx`, `/api/v0/server/shutdown` devolvio HTTP 200 con `status=ready`, `shutdown_ready=true`, `agents_in_flight=0` y `checkpoints_pending=0`, pero el wrapper fallo con `el servidor siguio vivo tras shutdown_ready=true`; despues del `trap` no quedaron procesos asociados al smoke | el contrato de shutdown mezcla "sin trabajo vivo/backends limpios" con "proceso HTTP contenedor terminado"; el endpoint publica drenaje operativo y el wrapper Nueva App interpretaba ese ready como garantia de terminacion inmediata del proceso temporal | smoke conservado en `/tmp/orquesta-goal-first-app-server.a4QRfx`; `shutdown_response.json` con evidencias `evidence-ref-codex-app-server-tmux-cleanup-requested`, `evidence-ref-codex-app-server-tmux-configured-cleaned`, `evidence-ref-shutdown-goal-backend-cleanup-requested`; tests `TestSmokeGoalFirstAppServerRealShutdownReadyUsaSenalCooperativaSiServidorSigueVivoV0`; `bash -n scripts/smoke_goal_first_app_server_real.sh`; `go test -count=1 ./cmd/orquesta-server -run 'TestSmokeGoalFirstAppServerRealShutdownReadyUsaSenalCooperativaSiServidorSigueVivoV0|TestSmokeGoalFirstAppServerRealShutdownLimpiaBackendPropioYReintentaV0'`; cierre T-PER-401 2026-07-03 con `exit_pending/pid`, `ForceExitPortV0`, `TestRuntimeV0ShutdownReadyProgramaSalidaForzadaSiNoTerminaV0` y paquete completo `go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-mcp ./modulos/orquesta-server-shutdown ./cmd/orquesta-server` | Cierre: `shutdown_ready=true` significa drenaje operativo y salida programada; el wrapper HTTP devuelve `exit_pending=true` y `pid`, el runtime programa salida forzada tras `ShutdownGracePeriod` por puerto inyectado, y el smoke espera la desaparicion de ese PID con timeout antes de usar fallback cooperativo |
| BUG-ORQ-20260703-147 | cerrado | MCP/catalogo herramientas | al integrar los 7 commits remotos, `go test -count=1 ./modulos/orquesta-mcp` fallo en `TestRegisterMCPTransportV0ExponeOperacionesExistentes`: el JSON del catalogo completo de tools crecio a 33.259 bytes y supero el limite 31.800 tras anadir diagnosticos/acciones en varias herramientas | los contratos completos de `InputShape`/`OutputShape` se estaban serializando dentro del registro masivo, duplicando esquemas largos que deben seguir vivos como contrato interno pero no viajar siempre en el indice compacto | fallo observado 2026-07-03 tras rebase sobre `7fb0568`; tests `TestMCPTransportToolEnvelopeMarshalJSONV0CompactaShapesSinMutarContrato`, `TestRegisterMCPTransportV0ExponeOperacionesExistentes`; `go test -count=1 ./modulos/orquesta-mcp`; `go test -count=1 ./cmd/orquesta-server` | Cierre: `MCPTransportToolEnvelopeV0.MarshalJSON` compacta shapes largas como `shape_ref:<resource>#input|output` solo al serializar el catalogo; los campos Go completos no se mutan y el MCP real sigue construyendo schemas desde `InputShape` |
| BUG-ORQ-20260703-148 | cerrado | Arquitectura/legacy director loop | `legacy_director_loop` seguia apareciendo como modo operativo compatible sin aviso publico de extincion y el smoke real OPES de proveedores dependia de que el operador exportase tambien `ORQUESTA_AUTOPROGRAMMING_LEGACY_DIRECTOR_LOOP=true` pese a declarar `director_execution_mode=legacy_director_loop` | la migracion goal-first mantenia el loop historico como compatibilidad pero sin contrato publico de sunset; los smokes legacy duplicaban opt-in en payload y entorno, aumentando falsos fallos por configuracion | T-PER-201; `AGENTS.md`; tests `TestStartAppDirectorV0LegacyLoopPublicaSunsetNoticeV0`, `TestMCPArrancarDirectorAppToolExecutorV0UsaServicioCanonico`, `TestSmokeOPESReviewsProvidersRealExigeOptInLegacyYWorkdirOPESV0`; `bash -n scripts/smoke_opes_reviews_providers_real.sh` | Cierre: `StartAppDirectorV0` y `orquesta.apps.arrancar_director.v0` publican `legacy_sunset_notice`; el smoke OPES exporta el opt-in de autoprogramacion solo en ese harness legacy con comentario de sunset. Politica: `legacy_director_loop` solo mantenimiento correctivo, sin features nuevas, candidato a retirada cuando goal-first cubra Claude y Gemini (T18) |
| BUG-ORQ-20260703-149 | cerrado | Remoto/WIP app-codex-stack | el bloque sucio remoto de `/srv/orquesta-self/worktrees/orquesta` (71 ficheros, 2741 inserciones incluyendo untracked) no era integrable tal cual: aplicado en `/tmp/orquesta-remote-dirty-review` sobre `uso/trabajo/plataforma-agentes` pasaba `git diff --check` pero fallaba `go test -count=1 ./modulos/orquesta-app-codex-stack` | el WIP mezclaba OPES capabilities, reconciliacion external-work, cambios Nueva App/web, runtime Codex y app-server/shutdown; varios tests revelaban contratos cruzados. Tras T270/T-PER-301-302, T-PER-401 y cierres posteriores, las piezas utiles quedaron absorbidas o superadas | patch local `/tmp/orquesta_remote_dirty_20260703.patch`; worktree `/tmp/orquesta-remote-dirty-review`; subagente Singer 2026-07-03 reejecuto focales citados y paquetes relacionados, todos verdes sobre HEAD actual; unico resto visible era hard-block de contexto requerido truncado/ref-only en `orquesta-runtime-codex`, descartado por riesgo de reintroducir rail duro por contexto omitido | Cierre: no integrar patch completo ni extraer mas piezas; conservar `/tmp/orquesta_remote_dirty_20260703.patch` solo como evidencia historica. Si se reabre, debe ser por bug nuevo con write-set propio y tests actuales, no por este WIP remoto |
| BUG-ORQ-20260703-150 | cerrado | Runtime Codex app-server / arquitectura T-PER-301-302 | el primer corte limpio habia dejado doble implementación entre `cmd/orquesta-server/codex_goal_app_server*.go` y `modulos/orquesta-runtime-codex-appserver`; `TransicionBackendV0` existia pero no gobernaba Ensure/shutdown/cleanup/active-work | doble fuente de verdad entre composicion y modulo del backend Codex app-server | cierre T270 en `docs/bitacora_correccion_pericial_2026-07-03.md`: legacy retirado/reducido a fachada fina, recolector unico de observacion backend y `TransicionBackendV0` gobernando ciclo; suites modulo+cmd+fronteras verdes | Cierre documentado 2026-07-03 noche; no reabrir salvo regresion del recolector unico o reintroduccion de implementacion tmux en `cmd` |
| BUG-ORQ-20260702-138 | cerrado | Web ops/shutdown | la consola `/ops` podia solicitar shutdown desde admin web sin `cleanup_goal_backends=true`, aunque CLI, guardian y smokes ya pedian limpieza gobernada | los consumidores UI/API del mismo endpoint no compartian un contrato operativo minimo para backends Goal propios; un operador web podia reiniciar sin activar el cleaner disponible | relacionado con BUG-065, BUG-076, BUG-136 y BUG-137; tests `TestOpsDashboardWebEndpointV0RenderizaPanelLiveCompleto`; `go test -count=1 ./modulos/orquesta-web` | Cierre: el payload de shutdown de `/ops` incluye `cleanup_goal_backends: true` y el endpoint HTML lo fija en test para evitar regresion |
| BUG-ORQ-20260702-137 | cerrado | Guardian/shutdown | `cmd/orquesta-guardian` hacia shutdown gobernado por HTTP sin enviar `cleanup_goal_backends=true`, aunque CLI y smokes ya lo pedian | las superficies de parada compartian endpoint pero no una politica comun para limpiar backends Goal propios; un guardian podia llegar a estado ready/forced sin activar el cleaner disponible | relacionado con BUG-065, BUG-076 y BUG-136; tests `TestGuardianV0ShutdownServerUsaCooperativoPorDefecto`; `go test -count=1 ./cmd/orquesta-guardian` | Cierre: el payload de shutdown del guardian incluye `cleanup_goal_backends=true` tanto en llamadas cooperativas como forzadas, manteniendo evidencia y lease existentes |
| BUG-ORQ-20260702-136 | cerrado | Shutdown/smokes operadores | algunos wrappers de smoke hacian `POST /api/v0/server/shutdown` directo sin `cleanup_goal_backends=true`: el shutdown forzado de `smoke_codex_required_test_runner_state_file.sh` y el retry manual de `smoke_goal_first_app_server_real.sh` podian saltarse la limpieza gobernada de backends Goal propios | BUG-077 no estaba completamente cubierto por exigir `runtime_dir` al helper comun; las llamadas HTTP directas tenian que compartir la misma politica de cleanup que `smoke_shutdown_orquesta_server` | relacionado con BUG-077, BUG-065 y BUG-076; tests `TestScriptsConShutdownDirectoPidenCleanupGoalBackendsV0`, `TestScriptShutdownCurlCommandsV0DetectaCleanupLejanoV0`, `TestScriptShutdownCurlCommandsV0DetectaPostSeparadoPorContinuacionV0`, `TestSmokeGoalFirstAppServerRealShutdownLimpiaBackendPropioYReintentaV0`, `TestScriptsQueUsanShutdownComunPasanRuntimeDirV0`, `TestHandoffCierreSesionNoReabreBUG077V0`; `bash -n scripts/smoke_goal_first_app_server_real.sh scripts/smoke_codex_required_test_runner_state_file.sh` | Cierre: los shutdown HTTP directos de scripts de smoke piden `cleanup_goal_backends=true`, incluido el retry tras limpieza manual, y una guarda recorre `scripts/*.sh` para impedir nuevas llamadas directas a `/api/v0/server/shutdown` sin cleanup gobernado, tambien si `curl -X` y `POST` quedan separados por continuaciones de shell |
| BUG-ORQ-20260702-135 | cerrado | Queue/status run control | `autoprogramming/status` podia recomendar `run_control_reconcile_external_cleanup` cuando un `GoalWorkState` seguia `running` pero el backend Goal ya no estaba activo, pero `queue/global-status` degradaba esa accion especifica al normalizarla | las superficies compactas de estado vivo tenian allowlists de acciones distintas; una accion gobernada de `runs/control` podia perderse al cruzar de status especializado a status global | relacionado con BUG-077, BUG-079 y BUG-088; tests `TestMCPQueueGlobalStatusHTTPHandlerV0GoalBackendAusenteConservaRunControlReconcile`, `TestMCPQueueGlobalStatusNormalizeRecommendedActionV0PreservaAccionesGoalFirstEspecificas`; `go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPQueueGlobalStatus(HTTPHandlerV0GoalBackendAusenteConservaRunControlReconcile|NormalizeRecommendedActionV0PreservaAccionesGoalFirstEspecificas)'` | Cierre: `queue/global-status` conserva `run_control_reconcile_external_cleanup` como accion publica especifica, de modo que el operador ve reconciliacion por `runs/control` en vez de una reparacion generica tras cleanup externo |
| BUG-ORQ-20260702-123 | cerrado | Run control/catalogo publico | tras mapear `control_not_propagated_to_goal_backend` a HTTP 409, el codigo seguia fuera del catalogo publico compartido `orquesta-i18n-docs`; futuras superficies podian normalizarlo a `executor_error` y perder la causa accionable | el contrato HTTP y el catalogo publico de errores estaban desacoplados: un error operacional retryable del backend Goal se publicaba por un handler, pero no era codigo estable de plataforma | relacionado con BUG-122; tests `TestPublicErrorCatalogV0RunControlGoalBackendActivoEsConflictRetryableV0`, `TestMCPPublicErrorCatalogV0CubreCodigosCompartidosYHTTPV0`; `go test -count=1 ./modulos/orquesta-i18n-docs ./modulos/orquesta-mcp -run 'TestPublicErrorCatalogV0RunControlGoalBackendActivoEsConflictRetryableV0|TestMCPPublicErrorCatalogV0CubreCodigosCompartidosYHTTPV0|TestMCPRunControlHTTPHandlerV0BackendGoalActivoEsConflictV0'` | Cierre: `control_not_propagated_to_goal_backend` queda declarado en el catalogo publico con HTTP 409, retryable y severidad warn; MCP lo reconoce como codigo compartido, de modo que las capas de transporte no lo degradan a `executor_error` |
| BUG-ORQ-20260702-122 | cerrado | Run control/HTTP | `POST /api/v0/runs/control` podia devolver HTTP 400 para `control_not_propagated_to_goal_backend`, mezclando un conflicto operacional por backend Goal vivo con una peticion malformada | la capa HTTP no distinguia validacion de entrada de un estado vivo no propagado; los consumidores podian tratar un backend activo tras stop/cancel como error de payload en vez de conflicto accionable | relacionado con BUG-063, BUG-065 y BUG-088; tests `TestMCPRunControlHTTPHandlerV0BackendGoalActivoEsConflictV0`, `TestMCPRunControlExecutorV0StopForcedNoPublicaStoppedSiGoalBackendSigueActive`; `go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPRunControlHTTPHandlerV0(BackendGoalActivoEsConflictV0|TimeoutDevuelveJSONPublico)|TestMCPRunControlExecutorV0StopForcedNoPublicaStoppedSiGoalBackendSigueActive'` | Cierre: el handler HTTP de `runs/control` mapea `control_not_propagated_to_goal_backend` a HTTP 409 y conserva el cuerpo publico con `estado=error`, `status=stop_requested|cancel_requested` y el issue accionable; las validaciones de input siguen en 400 |
| BUG-ORQ-20260702-121 | cerrado | Run control/Goal backend | `runs/control stop|cancel forced=true` podia reconciliar un `GoalWorkState` activo de alto consumo a `blocked/rework` tras confirmar que el backend Goal ya no estaba activo, pero dejar la proyeccion local de `RunControl` en `stop_requested` o `cancel_requested` si el puerto no devolvia estado terminal | la reconciliacion forzada del backend Goal cerraba la fuente de verdad de goal-first pero no completaba el estado de control cuando existia `RunControlTerminalWriterPortV0`, manteniendo una doble fuente de verdad entre rework terminal y control pendiente | relacionado con BUG-065 y BUG-088; tests `TestMCPRunControlExecutorV0StopForcedReconcilesGoalHighConsumptionCheckpointOnly`, `TestMCPRunControlExecutorV0StopForcedReconcilesGoalHighConsumptionSinCheckpoint`, `TestMCPRunControlExecutorV0CancelForcedCompletaRunControlTrasReconciliarGoalV0`; `go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPRunControlExecutorV0(StopForcedNoPublicaStoppedSiGoalBackendSigueActive|StopForcedReconcilesGoalHighConsumptionCheckpointOnly|StopForcedReconcilesGoalHighConsumptionSinCheckpoint|CancelForcedCompletaRunControlTrasReconciliarGoalV0|StopForcedReconcilesGoalBackendMissingAfterExternalCleanup)'` | Cierre: tras reconciliar un goal forzado a `blocked` con rework por alto consumo, checkpoint-only o cleanup externo, `runs/control` invoca `CompleteRunControlV0` cuando el puerto terminal esta disponible; `stop` publica `stopped`, `cancel` publica `canceled` y conserva evidencia `evidence-ref-run-control-terminal-after-goal-reconcile`. Si el backend sigue activo, se mantiene el bloqueo `control_not_propagated_to_goal_backend` y no se publica terminal |
| BUG-ORQ-20260630-001 | cerrado | Goal-first/autoprogramacion | `goal_first_state_missing` o estado goal no recuperado desde marker durable | estado goal y marker durable no estaban reconciliados como fuente de verdad | commits previos `3da7d88b`, `f9e330f1` | Mantener tests de status y no reabrir salvo regresion |
| BUG-ORQ-20260630-002 | cerrado | Runtime Codex | confusion entre `stdio`, `app_server_proxy` y `app_server_tmux` | contrato de backend operativo no estaba separado de diagnostico/proxy historico | docs/runbooks y commits `729e6695`, `ece98200`, `00ef13ac` | Canon: `app_server_tmux`; stdio no backend normal |
| BUG-ORQ-20260630-003 | cerrado | Self-programming remoto | bridge externo `disabled` seguia suprimiendo automejora | estado persistido de dominio interpretaba componente historico como sesion activa | commit remoto/local `e91fe0ebe` reportado por agente | Mantener test de `external_bridge_status=disabled` |
| BUG-ORQ-20260630-004 | cerrado | Readiness | backend Goal tmux degradado no bloqueaba readiness de forma publica | readiness no agregaba diagnosticos de backend goal como gate operativo | commit `00ef13ac` | Mantener readiness bloqueante sin filtrar rutas/tokens |
| BUG-ORQ-20260630-005 | cerrado | Autoprogramacion | fallo de launch Goal dejaba run huerfano o sin state bloqueado | launch y persistencia de `GoalWorkState` no eran transaccionales desde el punto de vista operacional | commit `97e134fd` | Persistir state invalid/reparable con reason accionable |
| BUG-ORQ-20260630-006 | cerrado | Supervision/autoprogramacion | `/autoprogramming/status` marcaba `running_stale` aunque habia procesos vivos | observadores de procesos, goals y runs no comparten modelo unico de estado vivo | incidencia OPES `TAREA_OPES_ORQUESTA_EXTERNAL_WORK_VALIDACION_Y_STREAM_2026-06-26.md`; test `TestMCPAutoprogrammingStatusExecutorV0NoMarcaStaleConDirectorStatsSnapshotRunningV0` | Cierre: el status usa `RunStore + ProcessRegistry + ProcessSnapshot`; proceso `running` proyecta `running_live=1`, `agents_live=1` y no publica `running_stale` |
| BUG-ORQ-20260630-007 | cerrado | Supervision HTTP | `/autoprogramming/supervise` despachaba agentes pero el cliente HTTP quedaba colgado | endpoints de control mezclaban accion larga, streaming/ack y respuesta sin contrato temporal claro | `modulos/orquesta-mcp/run_supervisor_http_v0.go`, `modulos/orquesta-mcp/autoprogramming_supervise_http_v0.go`, `cmd/orquesta-server/autoprogramming_supervise_http_flow_v0_test.go` | Cierre: ambos bridges devuelven `202 accepted_background`, `operation_ref`, dedupe de operacion activa y acciones de polling; tests de cliente real cubren cuerpo sin colgar |
| BUG-ORQ-20260630-008 | cerrado | Subagentes/OPES | padres OPES no materializaban 6 subagentes reales aunque declaraban subroles | contrato causal de hijos no se validaba en cierre padre/write-set | `docs/incidencia_opes_external_work_no_materializa_6_subagentes_por_padre_2026-06-23.md`; tests `TestCodexStackV0ExternalWorkRunOPESSubrolesMaterializaPadreYSeisHijosV0`, `TestOperationalClosureSourceV0NoCierraPadreOPESSubrolesSinHijosMaterializados`, goal-first OPES subrole receipt tests | Cierre de codigo: external-work OPES 1+6 materializa padre+6 hijos en fake runtime; cierre legacy y goal-first bloquean si faltan refs/evidencias causales de subroles. No equivale a claim de produccion OPES sin smoke propio |
| BUG-ORQ-20260630-009 | cerrado | Write-set/agentes | `agent_packet` estrechaba write-set e impedia consolidar Markdown canonico | permisos de escritura no distinguian borrador, evidencia y canon consolidado | reportes OPES 2026-06-26/27; `ExternalJobIntegrationDecisionSourceV0`; tests `TestCodexStackV0ExternalWorkRunOPESSubrolesPadreConservaWriteSetProductoAutorizadoV0`, `TestAppChangeDirectorDecisionSourceV0OPESSubrolesPadreConservaWriteSetProductoAutorizado`, `TestBuildExternalWorkGoalWorkSpecV0MarcaReworkSiSeisSubrolesSinWriteSetProducto` | Cierre de codigo: el padre conserva write-set de producto autorizado; hijos quedan acotados a subroles; si un legado quedo solo con coordinacion, se crea/expone integracion requerida o rework, no cierre silencioso |
| BUG-ORQ-20260630-010 | cerrado | OPES RAG | corrector ortografico procesaba `10_tutor_rag/corpus` regenerable | pipeline no marcaba artefactos regenerables vs canonicos | `/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/INCIDENCIA_OPES_CORRECTOR_PROCESA_RAG_REGENERABLE_2026-06-29.md` | Regla: no validar/corregir corpus regenerable como fuente canonica |
| BUG-ORQ-20260630-011 | cerrado | OPES finalpkg | RAG suelto `rag/chunks.jsonl` podia dar falso verde frente a `rag/corpus` canonico | validador y contrato de paquete final divergian | commits `f7859da0`, `7009e7bb`, `9180e70f` | Validar `rag/corpus/{chunks,summary}` y manifest refs |
| BUG-ORQ-20260630-012 | cerrado | OPES audio/finalpkg | `index.html`/portadas/listados contaban como manifiestos o fuentes de audio de tema | contrato de paginas tematicas no estaba codificado en validador | commits `c56d9b79`, `7009e7bb`, `9180e70f` | `audio/manifests/**` solo contra `html_final/tema_*.html` y `html_ampliado/tema_*.html` |
| BUG-ORQ-20260630-013 | cerrado | Nueva App UI/i18n | navegador mostraba `Please fill out this field` y backend exponia mensajes tecnicos/rutas | validacion HTML nativa y viewmodels no estaban localizados/sanitizados | commits `5f3e90e0`, `ffe8c7ba` | Validacion publica localizada; no filtrar HOME/tokens/proveedor |
| BUG-ORQ-20260630-014 | cerrado | Nueva App contrato | opciones expertas visibles no llegaban a factory/docs o se perdian filas 5-6 | catalogos UI, wizard, factory y guia eran fuentes duplicadas | commits `31929dc6`, `8266fd61`, `11165fb2` | Tests de catalogo visible e i18n |
| BUG-ORQ-20260630-015 | cerrado | MCP/external-work | faltaba cobertura explicita de que `goal_first` o modo no-legacy no activa legacy | migracion goal-first dependia de tests indirectos | patch remoto `orquesta-external-work-goal-first-explicit-a7940f5e-2026-06-30.patch` | Integrar/verificar commit remoto `a7940f5e` |
| BUG-ORQ-20260630-016 | cerrado | Operacion remota | agente remoto trabajaba sobre bundle stale y sin GitHub directo | protocolo de sync remoto no era canonico ni visible para agentes | bundle `orquesta-trabajo-plataforma-agentes-9180e70f.bundle`, nota `REMOTE_SYNC_AFTER_9180E70F_2026-06-30.md` | Usar bundles versionados y worktree fresco antes de editar |
| BUG-ORQ-20260630-017 | cerrado | Operacion remota | clone fresco no tenia identidad Git para commitear | bootstrap remoto no normalizaba config local de Git | observado en ciclo remoto `a7940f5e` | Config local `Orquesta Codex <orquesta-codex@local>` en clones frescos |
| BUG-ORQ-20260630-018 | cerrado | Supervision/runs | `/runs/supervise` podia devolver `runtime_error`/`failed` aunque el snapshot incluia procesos vivos | error del supervisor/outbox e idempotencia se confundia con fallo terminal del trabajo ejecutandose | incidencia OPES `TAREA_OPES_ORQUESTA_EXTERNAL_WORK_VALIDACION_Y_STREAM_2026-06-26.md`; `codexStackRunSupervisorLiveErrorDiagnosticsMCPV0`; test `TestCodexStackRunSupervisorErrorResultMCPV0DistingueErrorConAgenteVivo` | Cierre: publica diagnostico `supervisor_transition_error_but_agents_live` y acciones `wait_agents`, `retry_supervise`, `do_not_relaunch_same_run_ref_while_process_live` |
| BUG-ORQ-20260630-019 | cerrado | Review/DomainWork | review con `payload_invalido` tras entrega/ACK valido marcaba fallo global | fallo del reviewer se mezclaba con estado del trabajo de dominio ya materializado | `TAREA_OPES_ORQUESTA_REVIEW_RESULT_PAYLOAD_INVALID_TRAS_ACK_2026-06-25.md`; test `TestCodexStackRunSupervisorErrorResultMCPV0ExponeReviewPayloadInvalidoTrasEntrega` | Cierre: publica `domain_work_completed_review_failed`, conserva evidencia de entrega y recomienda `retry_review_with_compact_payload` sin relanzar agente de dominio |
| BUG-ORQ-20260630-020 | cerrado | Shutdown/checkpoint | `codex_shutdown_checkpoint_ack.v0` minimo con `status=checkpoint_ready` podia rechazarse como `forbidden_detail` | contrato pedido al agente y validador de shutdown estaban desalineados | `TAREA_OPES_ORQUESTA_CHECKPOINT_FORBIDDEN_DETAIL_Y_REVIEW_PAYLOAD_ASG_2026-06-25.md`; test `TestCodexShutdownCheckpointV0AceptaAckMinimoHidratadoDesdeRequest` | Cierre: ACK minimo se hidrata desde la request y deja de depender de detalle redundante en el fichero del agente |
| BUG-ORQ-20260630-021 | cerrado | External-work/dispatch | QA visual OPES quedaba con agente solicitado pero no arrancado ni entrega | estado publico no distinguia capacidad/runtime/outbox de cierre sin agente | `TAREA_OPES_ORQUESTA_QA_VISUAL_REMOTA_TCAE_SIN_AGENTE_2026-06-26.md`; tests `TestCodexStackRunSupervisorRequestedNotStartedDiagnosticsMCPV0ExponeExternalWorkQAVisual`, `TestMCPDirectorStatsToolExecutorV0DiagnosticaExternalWorkAgenteSolicitadoNoArrancado` | Cierre: publica `external_work_agent_requested_not_started`, contadores requested/started/in_flight y acciones `retry_materialization`/`check_capacity_auth_runtime_queue_outbox_policy` |
| BUG-ORQ-20260630-022 | cerrado | OPES capability/audio | TTS `edge-tts` en sandbox generaba reworks repetidos sin frontera de capacidad | capacidades externas estaban implícitas en agentes en vez de declaradas por adaptador/puerto | `TAREA_OPES_ORQUESTA_TTS_EDGE_SANDBOX_NETWORK_2026-06-25.md`; tests `TestDomainWorkExternalCapabilityRequirementsV0AudioRequiereSpeechSynthesis`, `TestRunOPESDrainOnceV0AudioSinSpeechSynthesisNoPosteaOrquestaV0`, `TestSmokeOPESDerivativesRESTWrapperPreflightRealBloqueaSpeechSynthesisSinEvidenciaV0` | Cierre de contrato: `audio_asset` requiere `speech_synthesis`; sin capacidad/evidencia se bloquea con razon operativa. Pendiente solo smoke OPES temporal real con runner/capacidad declarada |
| BUG-ORQ-20260630-023 | cerrado | Shutdown/runtime real | en reintento OPES goal-first, `/server/shutdown` devolvio `shutdown_ready=true` pero el proceso no salio o el cliente agoto timeout con goal activo | el hook de shutdown de `app_server_tmux` mataba la sesion tmux pero no esperaba la salida observable del `pane_pid` antes de publicar ready | `TAREA_OPES_ORQUESTA_GOAL_LAUNCHER_TMUX_EXITED_REVISION100_2026-06-29.md`; smoke temporal aislado `/srv/orquesta-self/runtime/smokes-14ab41f1/orquesta-goal-first-app-server.l460H8` reprodujo `pane_pid sigue vivo tras shutdown hook`; smoke temporal aislado `/srv/orquesta-self/runtime/smokes-14ab41f1/orquesta-goal-first-app-server.C3IDCo` confirmo `app_server_tmux_shutdown_ready=true`, sin sesion tmux ni PID vivo; tests `TestCodexAppServerTmuxBackendV0ShutdownMataSesionPropiaV0`, `TestCodexAppServerTmuxBackendV0EsperaPanePIDAntesDeReadyV0`, `TestRuntimeV0ServerShutdownReadyDetieneRuntimeHTTPV0`, `TestRuntimeV0ShutdownTimeoutPublicaStopTimeoutV0` | Cierre: `ShutdownV0` consulta `#{pane_pid}` antes de `kill-session` y espera su desaparicion dentro del timeout del backend; si no sale devuelve `codex_app_server_tmux_pane_exit_timeout` en vez de publicar ready falso |
| BUG-ORQ-20260630-024 | cerrado | OPES/autoprogramacion | el residente podia lanzar automejora/backlog de Orquesta dentro de un workdir OPES | la composicion no separaba sesion de dominio y autoprogramacion idle como scopes operativos distintos | `docs/incidencia_opes_resident_director_autoprogramming_en_workdir_opes_2026-06-23.md`; `modulos/orquesta-server/docs/tareas.md` SRV-TASK-027; tests `TestServerConfigFromEnvV0DesactivaAutomejoraIdleEnOPESSinWorkdirSeparadoV0`, `TestRuntimeV0IdleSelfImprovementBlockerImpidePrepareRunV0`, `TestStartupCheckV0SuprimeAutoprogrammingReadyEnSesionDominioV0` | Cierre local: en contexto OPES sin `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_PROJECT_WORKDIR` separado, la automejora idle se suprime con diagnostico `idle_self_improvement_suppressed_by_domain_session` |
| BUG-ORQ-20260630-025 | cerrado | OPES/audio | `generate_audio_asset` podia lanzar agente con ruta de herramienta audio inexistente | la capacidad `speech_synthesis` ya existia como contrato neutral, pero la composicion OPES no ejecutaba preflight real del tool-path antes del launch | `docs/incidencia_opes_asg_audio_tool_path_2026-06-24.md`; `modulos/orquesta-server/docs/tareas.md` SRV-TASK-023; `docs/incidencias/opes_orquesta_backlog_operativo_2026-06-28.md` ORQ-OPES-004; tests `TestOPESBridgeExternalCapabilitiesFromEnvV0PreflightTTSOKV0`, `TestOPESBridgeExternalCapabilitiesFromEnvV0PreflightTTSToolAusenteBloqueaV0`, `TestRunOPESDrainOnceV0AudioPreflightToolAusenteNoPosteaOrquestaV0` | Cierre: preflight opt-in con `ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_TOOL_WORKDIR`, comando por defecto `python3 scripts/opes_audio_app.py --help`, timeout corto y bloqueo `opes_audio_tool_missing` antes de postear a Orquesta |
| BUG-ORQ-20260630-026 | cerrado | OPES/visuales | SVG heredados o bocetos podian quedar como visual final publicable en HTML/paquete | el contrato historico de `visual_asset` aceptaba SVG seguro como artefacto, pero el cierre editorial OPES necesita distinguir insumo/borrador de arte final profesional | `docs/incidencia_opes_svg_visual_final_no_bloqueado_2026-06-24.md`; `docs/opes_flujo_temario_operativo_2026-06-02.md`; `modulos/orquesta-server/docs/tareas.md` SRV-TASK-019 relacionado; test `TestCodexStackV0ExternalWorkGoalFirstNoCierraOPESVisualFinalSVG` | Cierre de codigo: el cierre goal-first OPES bloquea `visual_asset` con `format/content_type/svg/body/ref` SVG mediante `domain_work_opes_visual_final_not_professional`. SVG sigue pudiendo existir como insumo/intermedio no terminal; pendiente solo smoke OPES temporal con validadores profesionales reales |
| BUG-ORQ-20260630-027 | cerrado | OPES/supuestos | olas de supuestos practicos podian parar procesos con cobertura parcial y esquemas no importables | Orquesta no modelaba contrato de salida por wave para carpetas esperadas, esquema JSON/JSONL y cobertura real de artefactos en el cierre goal-first | `docs/tarea_opes_supuestos_validacion_schema_y_cobertura_2026-06-23.md`; scripts OPES locales `normalizar_supuestos_tcae.py`, `validar_supuestos_tcae.py`, `consolidar_supuestos_tcae.py`; tests `TestCodexStackV0ExternalWorkGoalFirstNoCierraOPESSupuestosParcialesV0`, `TestCodexStackV0ExternalWorkGoalFirstCierraOPESSupuestosCompletosValidosV0` | Cierre de codigo: receipts OPES de supuestos practicos con `delivery_status=partial`, faltantes, conteos entregados menores que esperados, `schema_repairable/schema_invalid`, `tasks` sin `questions`, `development_task` o `process_status=stopped` sin entrega completa bloquean el cierre con `domain_work_opes_practical_cases_contract_incomplete`. Avance 2026-07-02: `practical_cases` queda como artefacto neutral de `domain-work`, el bridge OPES normaliza aliases de supuestos hacia ese contrato y el cierre goal-first emite refs de rework causales `domain-work-opes-practical-cases-rework:*` por faltante explicito, conteo derivado, schema y conversion de `development_task`; ya no queda bloqueo mudo sin siguiente trabajo |
| BUG-ORQ-20260630-028 | cerrado | OPES/API discovery | OPES encontro `/health` vivo en `8095`, pero `domain-work`, `external-work`, `runs/supervise` y autoprogramacion devolvian `404` | OPES no tenia contrato unico de descubrimiento/readiness para distinguir Orquesta correcta, composicion sin API, otro servicio en el puerto o endpoint con autenticacion/prefijo distinto | `/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/INCIDENCIA_ORQUESTA_API_DOMAIN_WORK_NO_EXPUESTA_2026-06-30.md`; `external/opes/INCIDENCIA_OPES_ORQUESTA_API_Y_TTS_TIMEOUT_2026-06-30.md`; tests `TestHandlerV0ExponeHealthStatusYDelegaV0`, `TestServerResourcesRouteManifestIncluyeDiscoveryOPESV0` | Cierre: OPES debe exigir `/api/v0/server/readiness ready=true` y despues validar `GET /api/v0/server/resources` con `route_manifest.routes` montadas para `domain-work`, `external-work/run`, `runs/supervise` y `autoprogramming/status`; si falta backend goal-first se marca `orquesta_degraded_not_ready`, si falta manifest/ruta se marca `orquesta_unavailable`/`orquesta_composition_missing_api`; no inferir disponibilidad por `/health` |
| BUG-ORQ-20260630-029 | cerrado | Runtime/startup | arranque desacoplado con `nohup` no dejo servidor escuchando pero si dejo `codex app-server`/tmux vivo | lifecycle de startup no era atomico: `EnsureV0` del backend Goal podia crear recursos antes de que el servidor publicara puerto/readiness | `external/opes/INCIDENCIA_OPES_ORQUESTA_API_Y_TTS_TIMEOUT_2026-06-30.md` hallazgo 4; test `TestCodexAppServerTmuxBackendV0LimpiaSesionSiSocketNoLlegaV0` | Cierre: si `app_server_tmux` falla tras `new-session` al escribir marker, esperar socket o pasar preflight, `EnsureV0` limpia sesion tmux propia, socket y owner marker antes de devolver la causa raiz; no deja app-server huerfano tras fallo post-start |
| BUG-ORQ-20260630-030 | cerrado | OPES/audio provider | `edge-tts` puede quedar esperando en bloques largos y detener una ola sin avance visible | trabajos externos largos no exponian heartbeat/progreso granular ni corte `provider_timeout` por proveedor/hijo; Orquesta ve running pero no progreso de lote | `external/opes/INCIDENCIA_OPES_ORQUESTA_API_Y_TTS_TIMEOUT_2026-06-30.md` hallazgo 2; mitigacion OPES `OPES_EDGE_TTS_TIMEOUT_SECONDS`/`OPES_EDGE_TTS_MAX_ATTEMPTS`; tests `TestDomainWorkExternalCapabilityEvaluationV0BloqueaAudioSinHeartbeatProveedor`, `TestDomainWorkExternalCapabilityEvaluationV0BloqueaAudioSinProviderTimeout`, `TestRunOPESDrainOnceV0AudioSinHeartbeatProveedorNoPosteaOrquestaV0`, `TestRunOPESDrainOnceV0AudioAlreadySubmittedProviderHeartbeatTimeoutProyectaSinReenviarV0`, `TestRunOPESDrainOnceV0AudioAlreadySubmittedSinProgresoProyectaSinReenviarV0`, `TestMCPDirectorStatsToolExecutorV0GoalFirstProyectaMetadataOperacionalV0`, `TestCodexStackExternalJobStatsSourceV0GoalFirstProyectaMetadataOPESAudioV0`, `TestOPESBridgeSupervisionFromDirectorStatsV0GoalFirstProyectaMetadataAudioV0`; OPES `scripts/test_run_orquesta_single_ref_queue_operational_metadata.sh` | Cierre: `audio_asset`/`speech_synthesis` exige progreso heartbeat, provider timeout y ventana maxima sin avance antes de que OPES bridge postee `generate_audio_asset` a Orquesta. Para jobs `generate_audio_asset` ya `already_submitted` o en recuperacion, el drain refresca el job OPES por `job_ref` y si el dominio publica `provider_timeout` o `running_no_recent_progress` como campo, external ref o payload, proyecta `supervision_status=blocked`, `supervision_stop_reason` y `retry_from_phase=tts` sin reenviar `/external-work/run`. La cola externa OPES `run_orquesta_single_ref_queue.sh` publica metadata operacional real al API OPES para audio (`current_phase`, contadores, `provider_timeout`, `running_no_recent_progress`) al iniciar, recuperar, no avanzar, completar o agotar timeout, sin tocar OPES productivo por defecto |
| BUG-ORQ-20260630-031 | cerrado | External-work/OPES guard | `external-work/run` rechazaba un trabajo OPES de control que solo escribia informe local en `external/opes/...` porque el runtime no tenia `project_work_dir` OPES | la guarda mezclaba dos casos: ejecucion OPES real, que debe exigir workspace OPES, y control/evidencia local de Orquesta, que puede vivir en `external/opes` sin tocar el dominio | `external/opes/INCIDENCIA_OPES_ORQUESTA_API_Y_TTS_TIMEOUT_2026-06-30.md` hallazgo 5; artefactos `external/opes/control_audio_psicologo_orquesta_20260630/external_work_audio_control_{request,response}.json`; tests `TestExternalWorkRunProjectWorkDirGuardPermiteOPESControlLocalExternalV0`, `TestExternalWorkRunProjectWorkDirGuardNoPermiteOPESLocalFueraDeExternalV0` | Cierre: la guarda mantiene bloqueo si el runtime no apunta a OPES, pero permite trabajos OPES cuyo `allowed_write_set` no vacio queda integramente bajo `external/opes/`; el error restante incluye accion para reiniciar con project workdir requerido o usar write-set local acotado |
| BUG-ORQ-20260630-032 | cerrado | Tooling/agentes | varias instancias `codebase-memory-mcp` quedaron vivas al usar sesiones/subagentes y consumian CPU sin trabajo activo | la norma operativa repartia Codebase entre agentes; faltaba un broker central por repo con indexacion singleton, limite de concurrencia, cache y watchdog | observacion local 2026-06-30: PIDs `809802`, `1232024`, `1232179`, `1232240`, `1232378` y `1375719` parados con `SIGTERM`; nueva observacion 2026-06-30: PIDs `1897351`, `1899925`, `2148912`, `2149006` y `2149084` parados con `SIGTERM`; tercera observacion 2026-06-30: PIDs `2325479` y `2325683` parados con `SIGTERM` tras intento MCP local fallido; cuarta observacion 2026-06-30: PID `2467281` hijo de Codex local parado con `SIGTERM` sin sockets abiertos; quinta observacion 2026-06-30: `codebase-memory-mcp/list_projects` devolvio `Transport closed` y quedaron PIDs `2791296` y `2791424`, parados con `SIGTERM`; sexta observacion 2026-06-30: PIDs `2811215`, `2811825`, `2889232`, `2897845`, `2898065`, `2898921`, `2899176` y `2899541` quedaron vivos bajo sesiones Codex/app-server temporales y se pararon con `SIGTERM`; septima observacion 2026-06-30: PID `3812284` quedo vivo 21m sin sockets y consumiendo CPU, parado con `SIGTERM`; smoke Orquesta `docs/orquesta_parallel_smoke_2026-06-30/tooling_cpu.md/work_delivery.md`; decision en `docs/diseno_orquesta_codebase_broker_2026-06-30.md`; tests `TestCodeContextBrokerV0UsaCacheCentralSinReinvocarProveedor`, `TestCodeContextBrokerV0DeduplicaConsultasConcurrentesIguales`, `TestCodeContextBrokerV0CacheDistingueFingerprintYWorktreeSucio`, `TestCodeContextBrokerV0ExigeLeaseCentralParaCodebaseMCP`, `TestEvaluateCodeContextToolLeaseV0PideParadaSiLeaseExpiraSinPeticiones`, `TestBuildCodeContextToolingStatusV0PideParadaParaLeaseExpirado`, `TestMCPCodebaseStatusTransportV0RegistradoYDelegado`, `TestMCPCodebaseStatusHTTPHandlerV0PostDelega`, `TestServerLimitedBufferV0DescartaExcesoSinBloquearWriter`, `TestMCPCodebaseQueryTransportV0RegistradoYDelegado`, `TestServerRGCodeContextProviderV0DevuelveCoincidenciasCompactas`, `TestCodexWaveCodeHomeProjectionV0DesactivaCodebaseMemoryMCPSinOptInCentral`, `TestCodeContextToolLeaseV0StoreMarcaLeaseStopped`, `TestServerCodeContextFileStateV0PersisteLeasesYCache`, `TestServerCodeContextToolWatchdogV0ParaLeaseExpiradoPorOwner`, `TestServerCodeContextToolWatchdogLoopConfigV0ExigeStateDirOptIn`, `TestRunServerCodeContextToolWatchdogLoopAsyncV0DesactivadoNoProyectaBridgeV0`, `TestServerCodeContextToolWatchdogV0OwnerMarkerAusenteNoBloqueaOtrosLeases`, `TestServerFileCodeContextToolOwnerStopperV0EscalaSiProcesoSigueVivo`, `TestRunServerCodeContextToolWatchdogLoopAsyncV0ParaLeaseExpiradoSinPIDV0`, `TestServerCodebaseMemoryCLIProviderV0ParseaSearchGraphYEscribeOwnerMarker`, `TestCodeContextBrokerWiringFromEnvV0ConfiguraCodebaseMemoryCLIConOptInV0`, `TestServerCodebaseMemoryCLIProviderV0RealSmokeOptIn` | Cierre: Codebase queda opt-in, prohibido por defecto en subagentes y detras de Orquesta. Implementado: puerto neutral `CodeContextQueryPortV0`, broker con cache/concurrencia/dedupe in-flight, cache sensible a `worktree_fingerprint`/`dirty_worktree`, lease central obligatorio para `codebase-memory-mcp`, evaluador puro TTL/CPU de leases, proyeccion publica `code_context_tooling_status.v0`, tool `orquesta.codebase.status.v0`, HTTP `/api/v0/codebase/status`, fallback `rg` central con stdout acotado, tool `orquesta.codebase.query.v0`, HTTP `/api/v0/codebase/query`, toolbelt y saneado de `CODEX_HOME` salvo opt-in central, completion `stopped`, persistencia file-based opt-in de cache/leases por `ORQUESTA_CODEBASE_BROKER_STATE_DIR`, watchdog invocable y loop residente opt-in `code_context_tool_watchdog_loop` que exige `ORQUESTA_CODEBASE_BROKER_STATE_DIR`, adaptador real `codebase-memory-mcp cli` por broker central con owner marker, smoke real opt-in contra repo indexado, no proyecta estados falsos cuando esta desactivado, tolera markers ausentes sin cortar el tick, espera salida con escalado SIGKILL y marca leases como `stopped`. Residual operativo: procesos lanzados fuera de Orquesta por otras sesiones no son gestionables hasta que esas sesiones usen el broker central |
| BUG-ORQ-20260630-033 | cerrado | Server/lifecycle | una Orquesta aislada de self-test publico `ready=true` y murio antes de aceptar `/external-work/run`, dejando `codex app-server` huerfano | el arranque desacoplado aceptaba una unica respuesta de readiness y no cerraba el backend Goal si el daemon lanzado ya habia desaparecido al fallar startup | self-test local `.orquesta-runtime/self-orquesta-parallel-20260630T142820Z`; tests `TestWaitForStateHealthyV0AceptaReadinessEstableV0`, `TestWaitForStateHealthyV0RechazaReadinessInestableTrasReadyV0`, `TestWaitForStateHealthyV0RechazaCambioDeIdentidadTrasReadyV0`, `TestCleanupCodexGoalBackendAfterStartupFailureIfDaemonGoneV0SoloMataSiProcesoCayo` | Cierre: `start` exige readiness estable con segunda consulta HTTP tras releer statefile y comprobar identidad de daemon; si startup falla y el PID lanzado ya no vive, apaga solo el `app_server_tmux` propio mediante owner marker, dejando intacto el backend si el daemon sigue vivo |
| BUG-ORQ-20260630-034 | cerrado | Startup/estado | arranque OPES con `project_work_dir` correcto aborta por `startup_dirty_runs_detected` y solo propone `ORQUESTA_STARTUP_CLEANUP_MODE=forced_stop` | la recuperacion de estado sucio era global y no distinguia trabajos vivos reales, runs colgados, cola desincronizada ni proyecto afectado | `external/opes/INCIDENCIA_OPES_ORQUESTA_API_Y_TTS_TIMEOUT_2026-06-30.md` hallazgo 6; patch remoto integrado `5d4365e6`; tests `TestStartupCleanupBlockersV0ExponeAccionYAntiguedadSinPathsV0`, `TestHandlerV0ReadinessExponeStartupBlockersV0`, `TestStartupSelectiveCleanupV0CierraScopeSinForcedStopGlobalV0`, `TestStartupSelectiveCleanupV0NoLimpiaFueraDeScopeV0` | Cierre: `startup_dirty_runs_detected` proyecta `startup_blockers` publicos en readiness con `run_ref`, `app_ref`, estado de cola, antiguedad, estado de proceso y accion segura, sin rutas locales. Ademas `ORQUESTA_STARTUP_CLEANUP_MODE=selective_project` con `ORQUESTA_STARTUP_CLEANUP_SCOPE_REFS` permite reconciliar cola/control solo para el scope autorizado y poner en cuarentena no forzada runs sin agentes vivos pendientes; si quedan candidatos fuera de scope o con agentes vivos, readiness sigue bloqueada y conserva `forced_stop` como opt-in explicito global |
| BUG-ORQ-20260630-035 | cerrado | Goal observe/API | `/api/v0/apps/director/goal/observe` devolvia `observe_app_director_goal_timeout` aunque el goal tenia resultado o proceso vivo materializado | observe mezclaba lectura rapida de estado, polling del ejecutor y cierre; cuando agotaba HTTP no devolvia estado parcial util ni accion segura | `external/opes/INCIDENCIA_OPES_ORQUESTA_API_Y_TTS_TIMEOUT_2026-06-30.md` hallazgo 7 y revalidacion 16:22; tests `TestMCPObserveAppDirectorGoalHTTPHandlerV0TimeoutIncluyeSnapshotParcialV0`, `TestMCPObserveAppDirectorGoalToolExecutorV0TimeoutSnapshotLeeGoalStateV0`, `TestCodexStackObserveAppDirectorGoalExecutorV0TimeoutSnapshotIncluyeProcessRefsV0` | Cierre: el timeout HTTP sigue siendo 504 con `observe_app_director_goal_timeout`, pero si el executor soporta snapshot rapido incluye `partial=true`, `goal_status`, `result_ref`, `last_event_at` desde eventos, `run_status/current_phase` desde run, `process_refs` desde `ProcessRegistry` del stack, evidencias y `recommended_action` (`observe_later`, `replan`, `blocked`) sin relanzar proveedor ni caer al loop legacy |
| BUG-ORQ-20260630-036 | cerrado | Goal artifacts/write-set | un write-set esperado terminado en `.md` se materializo como directorio con `docs/orquesta_goal_result_v0.json` dentro | el contrato de artefactos no distingue fichero final humano, directorio de paquete y carpeta auxiliar de resultado goal | `external/opes/INCIDENCIA_OPES_ORQUESTA_API_Y_TTS_TIMEOUT_2026-06-30.md` hallazgo 8; tests `TestBuildCodexGoalPromptV0NoTrataMarkdownWriteSetComoDirectorioV0`, `TestBuildCodexGoalPromptV0UsaSiguienteWriteSetDirectorioParaResultadoDurableV0` | Cierre: el adaptador Codex Goal trata write-sets `.md`/`.markdown` como archivo Markdown final y no cuelga `docs/orquesta_goal_result_v0.json` bajo ese nombre; si existe otro write-set de directorio, usa ese para el resultado durable; si no, conserva el cierre por marcador textual sin crear directorio falso |
| BUG-ORQ-20260630-037 | cerrado | OPES/audio workflow | Orquesta lanzo `generate-edge --all` antes de cierre textual final y antes de invalidar manifiestos tras cambios HTML | el contrato OPES de fases `text_qa -> prepare -> tts -> whisper -> rag -> qa` no era ejecutable por el bridge; vivia como instrucciones en prompt/herramienta | `external/opes/INCIDENCIA_OPES_ORQUESTA_API_Y_TTS_TIMEOUT_2026-06-30.md` revalidacion 16:36; tests `TestRunOPESDrainOnceV0AudioSinTextPublicPassNoPosteaOrquestaV0`, `TestRunOPESDrainOnceV0AudioSinPrepareRefsNoPosteaOrquestaV0`, `TestRunOPESDrainOnceV0AudioConRegeneracionAllNoPosteaOrquestaV0`, `TestRunOPESDrainOnceV0AudioConPrepareStaleNoPosteaOrquestaV0`, `TestRunOPESDrainOnceV0AudioConSpeechSynthesisPosteaOrquestaV0`, `TestEvaluateOPESAudioWorkflowPreconditionsV0BloqueaPrepareStalePorHashVigente`, `TestEvaluateOPESAudioWorkflowPreconditionsV0BloqueaPrepareStalePorStatus`, `TestMCPDirectorStatsToolExecutorV0GoalFirstProyectaMetadataOperacionalV0`, `TestOPESBridgeSupervisionFromDirectorStatsV0GoalFirstProyectaMetadataAudioV0`, OPES `TestAdvanceProgramWorkflowCoursePublicationOrdenaPrepareTTSWhisperRAG`, `TestUpdateJobOperationalMetadataPermiteAudioCursoExternoNoTerminal`, `scripts/test_run_orquesta_single_ref_queue_operational_metadata.sh` | Cierre: el drain OPES bloquea `generate_audio_asset` antes de `/external-work/run` si falta texto publicable cerrado, refs de preparacion (`audio_manifest_ref`/sidecar mas `source_content_ref` o hash), regeneracion selectiva, o si OPES declara prepare/audio manifest stale/invalidado o hashes/refs de texto actual vs preparado divergentes. OPES planifica curso por fases causales: prepare/captura HTML -> TTS `generate_audio_edge` -> Whisper `qa_audio_whisper` -> RAG de curso -> QA/publicacion, por lo que `generate_course_rag` ya no se crea antes de Whisper. El runner externo publica metadata operacional para `generate_audio_asset`, `generate_audio_edge` y `qa_audio_whisper`; no se activa en produccion sin opt-in `OPES_OPERATIONAL_METADATA_API_ENABLED=true` |
| BUG-ORQ-20260630-038 | cerrado | Status/eficiencia | `autoprogramming/status` muestra `queue.count=0`, `active_runs=0` y `overall_percentage=100` mientras `queue_health.blocked=2` y `stale_running=goal_first_blocked` | el resumen de eficiencia no penalizaba bloqueos goal-first persistidos cuando no hay cola activa, creando falso verde operacional | `external/opes/INCIDENCIA_OPES_ORQUESTA_API_Y_TTS_TIMEOUT_2026-06-30.md` revalidacion 16:55; test `TestMCPAutoprogrammingStatusExecutorV0NoDaCienConColaVaciaYGoalBloqueado` | Cierre: `efficiency_summary` degrada salud/estado ante `autoprogramming_goal_first_blocked` y `autoprogramming_goal_first_state_missing`; un goal bloqueado con cola vacia queda `attention_required`, no `idle/100`. Residual: status/observe-active aun debe ofrecer acciones mas especificas por fase (`retry_from_phase`, `close_superseded_by_new_evidence`) |
| BUG-ORQ-20260630-039 | cerrado | Runtime/readiness | OPES uso un servidor Orquesta obsoleto aunque existia un binario mas reciente aprobado | readiness/status no exponian identidad verificable del runtime activo ni compatibilidad minima por consumidor; OPES podia aceptar trabajos contra un binario viejo o un arbol que no compila | `external/opes/INCIDENCIA_OPES_ORQUESTA_API_Y_TTS_TIMEOUT_2026-06-30.md` revalidacion 17:15; proceso viejo `orquesta-server-latest` SHA `0994b0...`, binario nuevo SHA `a88167...`; tests `TestServerRuntimeIdentityV0SeProyectaSinFiltrarPathV0`, `TestStatusTrackerRestoreV0ReemplazaRuntimeIdentityDurablePorBinarioVivoV0`, `TestServerRuntimeIdentityFromExecutableV0CalculaSHAyBuildRefV0`, `TestVerifyLiveDaemonIdentityV0BloqueaRuntimeIdentityMismatch`, `TestExternalWorkRunRuntimeCompatibilityGuardBloqueaSHADistintoV0`, `TestExternalWorkRunRuntimeCompatibilityGuardPermiteSinOptInV0`, `TestExternalWorkRunRuntimeCompatibilityGuardPermiteIdentidadCoincidenteV0`, `TestExternalWorkRunRuntimeCompatibilityGuardRequiereIdentidadEsperadaSiOptInEstrictoV0`, `TestRunOPESDrainOnceV0BloqueaSinRuntimeIdentityCompatibleV0` | Cierre: readiness/status publican `runtime_identity` con `binary_path_ref`, `binary_name`, `binary_sha256`, `build_ref`, `commit_ref` y `started_at` sin filtrar rutas locales; el state interno conserva identidad del binario vivo y la restauracion reemplaza identidades obsoletas; `verifyLiveDaemonIdentityV0` detecta `daemon_runtime_identity_mismatch`; `external-work/run` pasa por una guarda opt-in del stack que bloquea antes de legacy/Goal si el consumidor declara `orquesta_runtime_binary_sha256`, `orquesta_runtime_build_ref`, `orquesta_runtime_commit_ref` o `orquesta_runtime_compatibility_required` y la identidad viva no coincide; el bridge OPES puede exigir readiness + SHA/build/commit mediante `ORQUESTA_OPES_BRIDGE_REQUIRE_RUNTIME_COMPATIBILITY=1` y bloquea sin claim/submission si no encaja. Sin opt-in no hay bloqueo global |
| BUG-ORQ-20260630-040 | cerrado | Queue/status goal-first OPES | con runtime nuevo vivo, runs OPES `goal_first_blocked` seguian recomendando `repair_runtime` aunque el runtime ya estaba aprobado | `queue/global-status` colapsaba acciones goal-first especificas (`review_replan_goal_first`) a reparacion generica de runtime; ademas el contrato OPES podia perder `superseded_by_local_evidence` por presupuesto de `input_fields` antes de llegar al estado goal-first | `external/opes/INCIDENCIA_OPES_ORQUESTA_API_Y_TTS_TIMEOUT_2026-06-30.md` revalidacion 17:38; tests `TestMCPQueueGlobalStatusHTTPHandlerV0GoalFirstBlockedConservaReviewReplan`, `TestMCPQueueGlobalStatusHTTPHandlerV0GoalFirstBlockedDesdeDiagnosticoNoReparaRuntime`, `TestMCPQueueGlobalStatusNormalizeRecommendedActionV0PreservaAccionesGoalFirstEspecificas`, `TestMCPQueueGlobalStatusHTTPHandlerV0PropagaFaseYCountersOPESRetryFromPhase`, `TestMCPDirectorStatsToolExecutorV0GoalFirstProyectaMetadataOperacionalV0`, `TestCodexStackExternalJobStatsSourceV0GoalFirstProyectaMetadataOPESAudioV0`, `TestBuildExternalWorkGoalWorkSpecV0PriorizaSupersededByLocalEvidenceOPESV0`, `TestMCPAutoprogrammingStatusExecutorV0GoalFirstCierraSupersededDesdeContextRefsV0`, `TestRunOPESDrainOnceV0AudioSupersededByLocalEvidencePosteaContratoGoalFirstV0` | Cierre: `queue/global-status` ya no transforma `goal_first_blocked` ni diagnostico `autoprogramming_goal_first_blocked` en `repair_runtime`; publica `review_replan_goal_first`, preserva `retry_from_phase`/`close_superseded_by_local_evidence` y transporta `current_phase`/`domain_counters` desde `stale_running`. El compilador `GoalWorkSpec` prioriza `superseded_by_local_evidence`, `superseded_by_local_evidence_ref`, `local_evidence_ref` y `local_artifact_ref` para que OPES pueda aportar cierre por evidencia local aunque haya muchos `input_fields`; `autoprogramming/status` recomienda `close_superseded_by_local_evidence` desde `context_refs` inlineados solo si hay flag y evidencia/ref local; el bridge OPES queda cubierto con smoke temporal aislado falso que postea `generate_audio_asset` conservando esas claves sin tocar OPES productivo |
| BUG-ORQ-20260630-041 | cerrado | Tests/lifecycle Codex wave | `TestCodexWaveStopCommandV0SolicitaParadaDesdeFicheroDedicado` podia leer `child.pid` vacio durante `go test ./...` y luego reaparecio como hijo vivo tras force stop | el helper esperaba existencia del fichero, no contenido parseable; ademas el force stop podia devolver antes de que el grupo de proceso completo cayese si el shell padre salia y el hijo seguia vivo unos instantes | fallo local `go test -count=1 ./...` 2026-06-30; test focal repetido `TestCodexWaveStopCommandV0SolicitaParadaDesdeFicheroDedicado`; regresion local en rama `remote-active-3f2cfc04` con `proceso 959501 sigue vivo tras force stop`; cierre focal `go test -count=1 ./cmd/orquesta-server -run 'TestCodexWaveStopCommandV0SolicitaParadaDesdeFicheroDedicado|TestCodexWaveStopCommandV0CooperativoAntesDeForceV0|TestProcessAliveV0TreatsZombieAsStopped|TestWaitUntilProcessDownV0EsperaSalidaReal'` | Cierre ampliado: el helper de test espera hasta que `child.pid` contenga un entero positivo; el force stop productivo envia SIGTERM al grupo, espera caida observable y escala a SIGKILL si el grupo sigue vivo antes de publicar stop solicitado |
| BUG-ORQ-20260630-042 | cerrado | OPES/visual reuse | cursos que reutilizan temas comunes pueden quedar con HTML local sin assets comunes importados y `visual_count=0` | el cierre OPES no tenia fase ejecutable `visual_asset_reuse` ni gate que distinguiera asset reutilizado, asset rechazado y visual profesional pendiente | `external/opes/INCIDENCIA_OPES_ASSETS_COMUNES_NO_IMPORTADOS_2026-06-23.md`; commit remoto integrado `470e0419`; tests `TestCodexStackV0ExternalWorkGoalFirstNoCierraOPESHTMLReadySinVisualesComunesV0`, `TestCodexStackV0ExternalWorkGoalFirstCierraOPESHTMLSinVisualesJustificadosV0`, `TestOPESFullTemarioJobTypeSequenceV0IncluyeCierreCompletoV0`, `TestOPESRequiredTestPolicyV0FinalTemarioExigeMinimosYComunes`, `TestBuildExternalWorkRunRequestV0MapeaDerivadosOPESConArtefactosEsperados` | Cierre: OPES declara fase `visual_asset_reuse` antes de `generate_html_site`, la mapea a `visual_reuse_manifest`, exige required test final `opes-visual-reuse-manifest-*` y criterios de matriz/copia/insercion/justificacion. El cierre goal-first bloquea HTML/final `ready/html_validado` con `visual_count=0` si existen visuales comunes/reutilizables pendientes de importar, copiar o insertar, salvo justificacion explicita de no aplicabilidad |
| BUG-ORQ-20260630-043 | cerrado | Runtime Codex Goal | `external-work/run` goal-first puede fallar por wrapper Node con `Node.js ResetStdio`, mientras el binario nativo funciona | stderr/log del app-server no se clasificaba y el adaptador caia a `codex_app_server_call_failed`, sin accion operativa clara | `external/opes/INCIDENCIA_OPES_ORQUESTA_INTEGRADOR_8788_CODEX_APP_SERVER_NODE_RESETSTDIO_2026-06-30.md`; tests `TestCodexAppServerIssueCodeForErrorV0ClasificaResetStdioV0`, `TestCodexAppServerIssueCodeFromLogFileV0ClasificaResetStdioV0`, `TestCodexAppServerWebSocketProtocolV0UsaLogResetStdioV0`, `TestCodexAppServerTmuxStartupFailureV0UsaLogResetStdioV0`, `TestServerCodexGoalBackendDiagnosticMessageV0RecomiendaBinarioNativo`, `TestExternalWorkGoalFirstKnownLaunchFailureReasonV0ClasificaResetStdio` | Cierre: `ResetStdio`/`node.cc:751` se clasifica como `codex_app_server_wrapper_stdio_failed`; `app_server_tmux` consulta el log acotado en startup y llamadas websocket caidas; readiness conserva `codex_goal_backend_degraded` pero el mensaje recomienda configurar `ORQUESTA_CODEX_COMMAND` con el binario nativo de Codex y reiniciar. No hay fallback automatico silencioso para no cambiar binario/runtime en produccion sin opt-in |
| BUG-ORQ-20260630-044 | cerrado | Shutdown/active goals | `/api/v0/server/shutdown` pudo apagar una instancia con goals OPES activos, dejando app-server huerfano/base bloqueada y sin drenaje claro | shutdown no forzado no consultaba trabajo goal-first activo persistido antes de pedir stops | `external/opes/INCIDENCIA_OPES_ORQUESTA_INTEGRADOR_8788_CODEX_APP_SERVER_NODE_RESETSTDIO_2026-06-30.md` actualizacion 21:16; tests `TestShutdownServerV0NoForzadoBloqueaConGoalActivo`, `TestShutdownServerV0ForzadoNoBloqueaConGoalActivo`, `TestMCPServerShutdownToolExecutorV0ExponeGoalsActivos`, `TestMCPServerShutdownHTTPHandlerV0ActiveGoalsDevuelveConflict`, `TestStackShutdownActiveWorkReaderV0ListaGoalFirstActivo` | Cierre: `orquesta-server-shutdown` tiene puerto neutral `ActiveShutdownWorkReader`; el stack Codex lista `AppGoalStateStore` con `ActiveOnly` y proyecta refs goal-first saneadas como `active_works`. Sin `forced`, shutdown devuelve `active_goals_present`, `shutdown_ready=false`, no pide `StopRunV0` y el HTTP responde `409`. Con `forced=true`, conserva la salida explicita de emergencia. Diagnostico de hook tmux/socket/pid queda cubierto por BUG-023 |
| BUG-ORQ-20260630-045 | cerrado | Director stats/goal timeout | `director/stats` publica `percent_complete=100`, `tasks_total=0`, `deliveries=0` para goals bloqueados que consumieron tokens y/o tocaron `work_delivery.*` | stats goal-first no reconciliaba artefactos materializados fuera de `GoalWorkResultV0`, y calculaba porcentaje desde ausencia de tareas como falso 100 | `external/opes/INCIDENCIA_OPES_ORQUESTA_INTEGRADOR_8788_CODEX_APP_SERVER_NODE_RESETSTDIO_2026-06-30.md` actualizaciones 21:23 y 21:55; tests `TestMCPDirectorStatsToolExecutorV0GoalFirstBloqueadoSinTareasNoDaCienV0`, `TestMCPDirectorStatsToolExecutorV0GoalFirstBloqueadoConReceiptParcialNoPierdeEntregaV0`, `TestMCPDirectorStatsToolExecutorV0GoalFirstReconcilaWorkDeliveryMaterializadoV0`, `TestStackGoalMaterializedRefsSourceV0DetectaWorkDeliveryEnWriteSet`, `TestStackGoalMaterializedRefsSourceV0NoSaleDelProjectWorkDir` | Cierre: `director/stats` ya no da 100% para goal-first bloqueado sin tareas observadas. Si `GoalWorkResultV0` trae `artifact_refs`/`domain_receipt_refs`, las proyecta como entregas derivadas. Ademas queda puerto neutral `MCPDirectorGoalMaterializedRefsSourcePortV0`; el stack Codex lo implementa escaneando solo `work_delivery.json` bajo write-set relativo a `ORQUESTA_CODEX_PROJECT_WORKDIR`, publica refs opacas `domain-receipt-ref-work-delivery:*`, marca `goal_first_materialized_work_delivery_detected`, cuenta entrega parcial y mantiene cierre bloqueado para QA/replan |
| BUG-ORQ-20260630-046 | cerrado | OPES/pre-audio text QA | validadores previos a audio pueden quedar verdes con mojibake (`mÃ`, `Ã`, `Â`, `�`, `â€`) en texto publicable | el preflight textual antes de `prepare`/`generate-edge` no bloqueaba corrupcion de codificacion ni obligaba regenerar derivados | `external/opes/INCIDENCIA_OPES_VALIDACION_TEXTO_PRE_AUDIO_NO_DETECTA_MOJIBAKE_2026-06-30.md`; commit remoto integrado `653b783a`; tests `TestEvaluateOPESAudioWorkflowPreconditionsV0BloqueaMojibakeEnTextoPublicable`, `TestRunOPESDrainOnceV0AudioConMojibakeNoPosteaOrquestaV0` | Cierre: `generate_audio_asset` OPES inspecciona campos textuales publicables (`text`, `markdown`, `html`, `rag`, `manifest`, `content`, `body`, `title`, `summary`, etc.) y bloquea antes de `/external-work/run` si detecta mojibake; publica `phase_precondition_missing:opes_audio:text_encoding_corrupted`, fase `text_qa`, acciones `repair_public_text_encoding`, `regenerate_audio_prepare_artifacts` y `retry_from_phase=text_qa` |
| BUG-ORQ-20260630-047 | cerrado | DomainWork/API observabilidad | OPES no tenia una comprobacion estable para ver trabajos vivos, colgados, en cola, terminados o bloqueados por cuota desde la superficie DomainWork | observabilidad de trabajos de dominio estaba repartida entre readiness, resources, `director/stats`, `queue/global-status` y `autoprogramming/status`, sin fachada estable para consumidores externos | `external/opes/INCIDENCIA_OPES_ORQUESTA_API_ESTADO_DOMAIN_WORK_NO_OBSERVABLE_2026-06-30.md`; tests `TestMCPDomainWorkStatusHTTPHandlerV0GetProyectaFachadaEstable`, `TestMCPDomainWorkStatusHTTPHandlerV0FiltroContextualSinRefDiagnostica`, `TestHandlerV0ExponeHealthStatusYDelegaV0`, `TestServerResourcesRouteManifestIncluyeDiscoveryOPESV0` | Cierre: se anade alias `GET /health` junto a `/healthz` y endpoint `GET /api/v0/domain-work/status` con schema `domain_work_status.v0`, resumen `queued/running/blocked/failed/completed/stale/waiting_quota`, items normalizados, acciones seguras y diagnostico cuando `project/course_slug` se usan sin `run_ref`, `app_ref` o `external_job_ref`. La fachada reutiliza `autoprogramming/status`/`queue/global-status`; no duplica estado ni mete OPES en core |
| BUG-ORQ-20260630-048 | cerrado | Web cockpit/i18n | `/ops` exponia textos visibles hardcodeados solo en castellano aunque `orquesta-web` exige i18n por defecto para UI | `/ops` habia crecido como HTML/JS estatico separado del owner i18n de Nueva App y sin contrato minimo por locale | auditoria local 2026-06-30; tests `TestOpsDashboardWebEndpointV0LocalizaTextosPrincipalesEnIngles`, `TestOpsDashboardWebEndpointV0RenderizaPanelLiveCompleto` | Cierre: `/ops` conserva castellano por defecto y acepta `?lang=en` con `Content-Language=en` y textos principales de cockpit localizados; queda como i18n acotado del panel, no como refactor completo del JS a catalogo externo |
| BUG-ORQ-20260630-049 | cerrado | Limpieza/runtime remoto | retencion y purga podian borrar `plan.json` o `artifacts_manifest.json` antiguos si no contenian `plan_state`/`outbox` en el nombre | las guardas de limpieza protegian solo nombres historicos de control, no artefactos durables genericos de plan/material util | auditoria local 2026-06-30; tests `scripts/test_orquesta_runtime_retention.sh`, `TestCodexWavePurgeRuntimeBlocksDurablePlansAndArtifactManifestsV0` | Cierre: retencion shell y purga Codex bloquean `plan.json`, `artifact_manifest.json`, `artifacts_manifest.json` y patrones `*artifact*manifest*` con razon `durable_plan_or_artifact_manifest`; dry-run sigue siendo el default |
| BUG-ORQ-20260630-050 | cerrado | Runtime worktree/VCS | `allow_push=true` podia ejecutar `git push` contra upstream implicito si faltaban `remote_name` y `remote_branch` | el opt-in de efecto externo autorizaba la clase de accion, pero no obligaba a declarar destino causal explicito | auditoria temporal 2026-06-30; tests `TestGitAppVCSConnectorV0BloqueaPushImplicitoSinDestinoExplicitoV0`, `TestGitStagingPromotionConnectorV0BloqueaPushImplicitoSinDestinoExplicitoV0` | Cierre: `AppVCS` y `StagingPromotion` rechazan `allow_push` sin `remote_name` y `remote_branch`; los pushes siguen permitidos solo con destino explicito |
| BUG-ORQ-20260630-051 | cerrado | OPES/html carcasa TCAE | `generate_html_site` o `completed_syllabus_package` podian cerrar con HTML `ready/html_validado` aunque OPES declarase carcasa legacy `topnav`, indices de variante ausentes o enlaces locales rotos | el contrato de carcasa USO/TCAE vivia en docs/prompts, pero el cierre goal-first no leia evidencias estructuradas de carcasa HTML | `/home/berserk/orquesta-inbox/opes-incidencias-local-20260701/INCIDENCIA_CARCASA_TCAE_INFORMATICA_B_2026-06-30.md`; tests `TestCodexStackV0ExternalWorkGoalFirstNoCierraOPESHTMLCarcasaTCAEIncompletaV0`, `TestCodexStackV0ExternalWorkGoalFirstCierraOPESHTMLCarcasaTCAEValidaV0` | Cierre: el cierre goal-first OPES bloquea HTML/final package listo si el receipt declara `topnav` legacy, carcasa incompleta, indices `html_final/html_ampliado/html_resumido/bases/tests` ausentes, links locales rotos o etiqueta publica `Documentacion oficial`; el bridge incorpora esos criterios en `generate_html_site`/`finalize_temario_package`. No lee OPES productivo ni filesystem de cursos |
| BUG-ORQ-20260701-052 | cerrado | Autoprogramming/status/codebase | `autoprogramming/status` podia devolver timeout poco accionable y `observe-active` podia mantener observacion colgada con goal activo/prepared/blank; `codebase-memory-mcp` podia fanoutear si habia consultas distintas mientras un lease del repo seguia activo | los handlers HTTP no cancelaban siempre el trabajo largo tras aceptar background y el broker de codebase deduplicaba por query, no por singleton de herramienta/repo | `/home/berserk/orquesta-inbox/REMOTE_ASSIGN_BUG_052_AUTOPROGRAMMING_STATUS_CODEBASE_2026-07-01.md`; tests `TestMCPAutoprogrammingObserveActiveGoalsHTTPHandlerV0CancelaExecutorAlAceptarBackground`, `TestMCPAutoprogrammingStatusHTTPHandlerV0TimeoutDevuelveJSONPublico`, `TestCodeContextBrokerV0NoArrancaCodebaseMCPConLeaseActivoDelRepo` | Cierre: `observe-active` cancela el contexto del executor al devolver `202 accepted_background`, `status` publica acciones concretas de diagnostico acotado en timeout y el broker bloquea una segunda ejecucion `codebase-memory-mcp` para el mismo repo/tool si ya existe lease activo, sin invocar proveedor ni arrancar proceso |
| BUG-ORQ-20260701-053 | cerrado | Runtime Codex Goal/codebase | `app_server_tmux` reutiliza un `CODEX_HOME` aislado bajo `.orquesta-runtime/goal-srv/codex-home` que puede quedar sucio entre arranques con `skills`, caches, SQLite y locks (`app-server-startup.lock`); el goal idle local quedo en `prepare_failed` pese a readiness `startup_ready` | el lifecycle de `CODEX_HOME` del app-server mezclaba credenciales proyectadas con estado mutable de Codex. El codigo copiaba `auth.json`/`config.toml`, pero no limpiaba el destino antes de arrancar, asi que se conservaban herramientas/plugins/cache no deseados y posibles locks entre generaciones | Observacion local 2026-07-01 en commit `3b154961`: `.orquesta-runtime/goal-srv/codex-home` contenia `skills/.system`, `.tmp/plugins*`, `state_5.sqlite`, `goals_1.sqlite`, `logs_2.sqlite`, `cache/codex_apps_*`, `models_cache.json` y `app-server-control/app-server-startup.lock`; readiness local: `idle_self_improvement_goal_status=error`, `idle_self_improvement_goal_reason_code=prepare_failed`. Tests de cierre: `TestCodexAppServerTmuxBackendV0PreparaCodeHomeLimpioV0`, `TestCodexAppServerTmuxBackendV0PreparaCodeHomeConservaCredencialesActualesSiNoHayFuenteV0`, `TestCodexAppServerTmuxBackendV0PreparaCodeHomeRechazaSymlinkV0`, `TestCodexAppServerTmuxBackendV0PreparaCodeHomeRechazaRutaNoGoalSrvV0` | Cierre: `prepareTmuxCodeHomeV0` valida que el destino sea `goal-srv/codex-home`, bloquea symlinks/rutas inseguras, reconstruye el `CODEX_HOME` aislado por arranque y proyecta solo `auth.json`/`config.toml` con modo `0600`; no arrastra `skills`, plugins, cache, SQLite, logs ni locks |
| BUG-ORQ-20260701-054 | cerrado | Tests/runtime Codex wave | `go test -count=1 ./...` podia fallar de forma no reproducida en `TestCodexLaunchWaveCommandV0LanzaOlaConCodexFalso` con `stdout agente 0 inesperado` vacio | carrera del test: el fake escribe `last_message` antes de imprimir stdout; el test esperaba solo la existencia de `LastMessagePath` y leia `StdoutPath` inmediatamente, por lo que podia observar stdout aun vacio aunque el wrapper siguiera completando salida | Observacion durante cierre BUG-053: primera ejecucion de `go test -count=1 ./...` fallo solo en `orquesta/cmd/orquesta-server` tras 133s; cierre BUG-054: `codex_wave_command_v0_test.go` espera contenido `fake stdout` con deadline corto y diagnostico del ultimo contenido; pasan `go test -count=1 ./cmd/orquesta-server -run '^TestCodexLaunchWaveCommandV0LanzaOlaConCodexFalso$' -v` y `go test -count=1 ./cmd/orquesta-server` | Cierre: helper local de test espera contenido esperado en stdout antes de validar; no se toca runtime productivo |
| BUG-ORQ-20260701-055 | cerrado | OPES/runtime discovery | una auditoria OPES de `Grupo B Informatica` intento usar Orquesta en puertos locales historicos `19023-19028` y no encontro runtime activo, por lo que cerro con validadores directos sin ola dirigida ni registro de `rework_mayor` en Orquesta | el consumidor OPES dependia de memoria operacional/puertos historicos para descubrir Orquesta y no tenia un work_kind explicito para auditar temario existente, dictaminar `rework_mayor` y preparar rework por tema | `TAREA_OPES_ORQUESTA_GRUPO_B_INFORMATICA_AUDITORIA_SIN_RUNTIME_LOCAL_2026-07-01.md`; `docs/incidencias/incidencia_orquesta_opes_auditoria_temario_existente_discovery_2026-07-02.md`; tests `TestBuildExternalWorkRunRequestV0MapeaAuditoriaTemarioExistenteV0`, `TestOPESBridgeArtifactContractMapConsumeOwnerNeutralV0`, `TestOPESBridgeTransportJobTypeForWorkKindV0`; `go test -count=1 ./modulos/orquesta-opes-bridge` | Cierre: BUG-028 cubre discovery generico por `/api/v0/server/readiness` y `/api/v0/server/resources.route_manifest`; el bridge OPES anade `audit_existing_syllabus_quality`/alias, transportable como `review_textual`, con `expected_artifact_type=opes_quality_audit_report`, contrato `opes_existing_syllabus_quality_audit.v0`, decision global `apto|revision|rework_menor|rework_mayor|bloqueado` y `rework_task_requests` por tema. No se debe volver a buscar Orquesta por puertos historicos |
| BUG-ORQ-20260701-056 | cerrado | Runtime/status vivo | en un runtime local aislado de rework OPES, el puerto dejo de responder y el PID de `server_goal_backend.pid` ya no existia, pero `orquesta_server_state_v0.json` seguia publicando `status=running`, `startup_ready` y supervisores `ok` | el estado persistido de servidor no reconcilia de forma suficiente muerte de proceso, causa de salida y estado vivo observable; repite el eje arquitectonico de fuente de verdad repartida entre proceso, pidfile, readiness, logs y state durable | `TAREA_OPES_ORQUESTA_GRUPO_B_RUNTIME_EXIT_STATE_STALE_2026-07-01.md`; tests `TestStatusServerCommandV0ReconciliaStatefileConPIDMuerto`, `TestStatusServerCommandV0FallbackStatefileNoPublicaEstadoCrudo` | Cierre acotado: `orquesta-server status` deja de proyectar un snapshot `running/startup_ready` si el endpoint no responde y el PID persistido esta muerto; reconcilia el statefile a `status=stale`, `startup_ready=false`, `startup_status=server_process_stale`, conserva `last_heartbeat_at` real y emite salida publica `stale` sin filtrar PID/rutas. Si el PID sigue vivo pero HTTP no responde conserva `degraded`. Lectores que abran el JSON sin ejecutar status siguen necesitando watchdog/reconciliador externo |
| BUG-ORQ-20260701-057 | cerrado | Goal-first/observe HTTP | `POST /api/v0/apps/director/goal/observe` para seis runs OPES P0 devolvio `504` o timeout de cliente sin snapshot parcial consumible (`goal_status`, `run_status`, `closure_status`, `partial`, `recommended_action`) | observe goal-first sigue mezclando operacion potencialmente lenta con contrato HTTP sin deadline parcial estable; el consumidor no puede distinguir goal vivo, colgado, lento, sin entrega o pendiente de replan | `TAREA_OPES_ORQUESTA_GRUPO_B_GOAL_FIRST_OBSERVE_Y_QA_MINIMOS_2026-07-01.md`; tests `TestMCPObserveAppDirectorGoalHTTPHandlerV0TimeoutDevuelveJSONPublico`, `TestMCPObserveAppDirectorGoalHTTPHandlerV0TimeoutIncluyeSnapshotParcialV0`, `TestServerObserveAppDirectorGoalHTTPClienteRealRecibeTimeoutJSONV0` | Cierre: el timeout HTTP cancela el executor completo, devuelve siempre cuerpo publico parcial con `partial=true`, `run_ref`, causa `observe_app_director_goal_timeout`, `recommended_action=observe_later`, resumen accionable y evidencia; si el executor soporta snapshot rapido, mezcla `goal_status`, `run_status`, `closure_status`, `last_event_at`, `process_refs` y refs disponibles sin relanzar proveedor ni caer al loop legacy |
| BUG-ORQ-20260701-058 | abierto | OPES/contrato de calidad | trabajos OPES de rework generaron artefactos parciales sin estado por tema, sin checkpoints/subroles/QA de minimos, temas B bajo 10.800 palabras, falso verde `9301 supera 10.800`, metacomentarios/frases de estrategia de examen visibles con informe local `pass` y visual raster decorativo aceptable por forma pero no por funcion didactica | el contrato OPES de temario sigue demasiado implicito: extension minima, QA textual, funcion didactica visual y estado operacional por tema no estan materializados como gates/metadata antes de cierre o candidato publicable; los informes de agente pueden interpretar mal separadores numericos o no detectar lenguaje de examen y no sustituyen al validador canonico | `TAREA_OPES_ORQUESTA_GRUPO_B_GOAL_FIRST_OBSERVE_Y_QA_MINIMOS_2026-07-01.md`; avance estado operacional por tema con tests `TestTopicRegistryOperationalStatusV0NormalizaAliasesCanonicosV0`, `TestProduceOPESCausalJobsV0CreaActualizacionRegistroPorTema`, `TestProduceOPESCausalJobsV0PreservaEstadoTextoMinimoPendiente`, `TestProduceOPESCausalJobsV0BloqueaRegistroPorQATemaFallidaV0`; avance cierre final con `docs/incidencias/incidencia_orquesta_opes_final_package_topic_quality_contract_2026-07-02.md` y `go test -count=1 ./modulos/orquesta-app-codex-stack`; avance OPES director 2026-07-04 noche 31 con `TestProduceOPESCausalJobsV0PaqueteFinalSinTopicQualityRefsNoLiberaRegistroV0`, `TestProduceOPESCausalJobsV0PaqueteFinalCompleteConManifestCompatibleYQATernaLiberaRegistro` y `go test -count=1 ./modulos/orquesta-opes-director` | Avance: `update_topic_registry` publica `operational_status` canonico `working|waiting|needs_rework|blocked|complete` y `operational_status_contract`; aliases de OPES/ingles se normalizan sin convertir strings recuperables en veto, los fallos de `OPESTopicQualityContractV0` fuerzan `needs_rework` y el cierre `completed_syllabus_package` exige `topic_quality_contract_result_refs`/`topic_quality_contract_results` por tema. Avance 2026-07-02: si un artefacto OPES goal-first intenta asentarse con QA pasada pero solo tiene heartbeat o no aporta checkpoint durable, el registro publica `operational_status=waiting`, `pending_refs=goal-first-topic-checkpoint-required` y no queda como `working` implicito. Avance 2026-07-04 noche 31: `orquesta-opes-director` alinea el registro con el stack Codex y ya no libera paquete final con manifest agregado si faltan refs de resultados `OPESTopicQualityContractV0` por tema; deja `pending_refs=final-package-manifest-closure-evidence-required` y followup `finalize_temario_package`. Siguen pendientes en esta fila los aspectos de lifecycle amplio: cierre agregado de todo el arbol OPES sin depender de estado implicito y smoke OPES real de extremo a extremo |
| BUG-ORQ-20260701-064 | cerrado | OPES/contrato de calidad ejecutable | BUG-058 necesitaba un primer contrato determinista que no dependiera de informes textuales de agentes para detectar falsos verdes de tema OPES | sin una pieza pura de composicion OPES, los minimos de palabras, metacomentarios y visuales decorativos quedaban como criterios narrativos no ejecutables | `TAREA_OPES_ORQUESTA_GRUPO_B_GOAL_FIRST_OBSERVE_Y_QA_MINIMOS_2026-07-01.md`; tests `TestValidateOPESTopicQualityContractV0BloqueaNivelBPorDebajoMinimoV0`, `TestValidateOPESTopicQualityContractV0DetectaMetacomentariosPublicosV0`, `TestValidateOPESTopicQualityContractV0NoAceptaRasterDecorativoComoDidacticoV0`, `go test -count=1 ./modulos/orquesta-opes-director` | Cierre parcial de BUG-058: `modulos/orquesta-opes-director` incorpora `ValidateOPESTopicQualityContractV0`, puro y sin acceso a OPES productivo, que parsea enteros con miles (`9.301` -> `9301`), exige minimo B `10.800`, devuelve `needs_expansion_min_words_B`, detecta metacomentarios publicos de examen/alcance y no acepta `raster=true` sin funcion didactica y ancla. BUG-058 sigue abierto hasta cablear este contrato en el runner/cierre OPES y estado vivo por tema. |
| BUG-ORQ-20260701-065 | abierto | Shutdown/Goal backend | `POST /api/v0/server/shutdown` agoto timeout sin cuerpo, `orquesta-server` termino con `orquesta_server: async_work_timeout` y el `codex app-server` goal-first siguio vivo hasta completar seis temas | el shutdown del servidor y el ciclo de vida del backend goal-first siguen siendo fuentes de verdad separadas; cuando la API cae, no queda handoff durable suficiente para saber si el backend debe esperar, parar, cancelar o quedar delegado | `TAREA_OPES_ORQUESTA_GRUPO_B_GOAL_FIRST_OBSERVE_Y_QA_MINIMOS_2026-07-01.md` apartado 12; relacionado con BUG-023 y BUG-044, pero esos cierres no cubren timeout sin JSON con backend que continua y completa; avance `docs/incidencias/incidencia_orquesta_shutdown_active_work_handoff_2026-07-02.md`; tests `TestShutdownProjectionFromHTTPV0ConservaActiveWorkRefsV0`, `TestRuntimeV0ServerShutdownSnapshotPrevioSobreviveRespuestaSinCuerpoV0`, `TestRuntimeV0ServerShutdownConservaCheckpointAgentsPendingV0`, `TestShutdownClientReadyV0BloqueaActiveWorkPersistido`, `TestShutdownRequestErrorAllowsSignalV0SoloConTimeoutYEstadoDrenado`, `TestRequestServerShutdownV0ReadyNoSaltaActiveWorkPersistido`, `TestRequestServerShutdownV0ReadyNoSaltaActiveWorksEstructuradosV0`, `TestWaitServerShutdownReadyV0CortaTrasRepostNoRecuperableV0`, `TestNormalizeServerShutdownClientResultV0ConvierteActiveWorksEnRefs`, `TestMCPRunControlExecutorV0CancelForcedCompletaRunControlTrasReconciliarGoalV0`, `TestRunSupervisorGoalFirstResidentReconciliaBackendMissingTrasCleanupExternoV0`, `TestRequestServerShutdownV0PostColgadoConsultaStatusAccionableV0`, `TestWaitServerShutdownReadyV0RepostColgadoRespetaDeadlineYDevuelveStatusAccionableV0`; `go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`; `go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPRunControlExecutorV0(CancelForcedCompletaRunControlTrasReconciliarGoal|StopForcedReconcilesGoalHighConsumption(SinCheckpoint|CheckpointOnly)|StopForcedReconcilesGoalBackendMissingAfterExternalCleanup)'` | Avance: `shutdown_freeze` conserva en state/status publico `shutdown_active_work_count` y refs saneadas de `active_works` si la respuesta HTTP de shutdown observo `backend_still_running` antes de un timeout posterior; nuevo avance de raiz: `orquesta-server` acepta un puerto neutral `ShutdownSnapshotPortV0` y toma un snapshot previo al handler HTTP, cableado en `cmd/orquesta-server` al lector active-work del stack Codex y al backend idle, de modo que una respuesta sin cuerpo no borra los refs del backend vivo. Nuevo avance de cliente: `orquesta-server stop` propaga `shutdown_active_work_count/refs` desde `/status` al resultado interno, no permite señal tras timeout si el handoff durable conserva active work del backend Goal y ya no acepta `shutdown_ready=true` como suficiente cuando el resultado inicial o `/status` siguen declarando active work. Nuevo avance: `checkpoint_agents_pending` viaja desde la respuesta HTTP de shutdown hasta `StateV0`, status publico y cliente CLI; una respuesta o status con agentes de checkpoint pendientes queda `stop_pending` y no permite signal cooperativa por timeout. Nuevo avance de cliente: `active_works` estructurado del body HTTP se normaliza a `active_work_refs` compactas y `active_work_count`, de modo que `orquesta-server stop` tampoco salta a signal si el endpoint no duplica `active_work_refs`. Nuevo avance de cliente: incluso con `--force`, un error de transporte del POST de shutdown no permite señal cooperativa si el `/status` publico conserva `active_work`, agentes, checkpoints o async work pendientes. Nuevo avance de cliente: durante la espera de cleanup, si un re-POST de shutdown devuelve un estado estructurado no recuperable como `active_goals_present`, el cliente corta con ese cuerpo en vez de depender de `/status` o agotar timeout. Nuevo avance 2026-07-04 noche 34: si el POST inicial o un re-POST se cuelga, el cliente usa timeout por intento, consulta `/status` y conserva `active_work_refs` en el error accionable en vez de quedar esperando 120s sin cuerpo. Nuevo avance de run-control: cuando `cancel forced=true` reconcilia un backend Goal de alto consumo ya ausente, el `GoalWorkState` terminal conserva summary/evidencia de `forced cancel` y no lo degrada a un handoff ambiguo de `forced stop`. Nuevo avance residente: el rework automatico tras cleanup externo conserva en el spec las evidencias `evidence-ref-goal-first-resident-backend-missing-reconciled` y `evidence-ref-autoprogramming-goal-backend-missing-after-external-cleanup`, evitando relanzar sin causa trazable. Avance 2026-07-04 noche 7: esa misma reconciliacion residente completa `RunControl` como `stopped` con reason/idempotencia `external cleanup` y evidencia `evidence-ref-run-control-terminal-after-goal-reconcile`, evitando que quede `RunControl` pendiente mientras el goal ya esta bloqueado/rework. Pendiente: coordinar backend goal-first con checkpoint/stop/cancel/wait controlado y garantizar decision operacional automatica tras cortes externos/manuales |
| BUG-ORQ-20260701-065 | cerrado | Shutdown/Goal backend | `run_control_reconcile_external_cleanup` podia quedar como recomendacion no ejecutiva si el operador llamaba `runs/control stop` con la evidencia de backend ausente pero sin `forced=true` | la reconciliacion de cleanup externo estaba acoplada a parada forzada, aunque la evidencia de `autoprogramming/status` ya declara que el backend observado no esta activo y que el `GoalWorkState` durable sigue pendiente de observacion | avance 2026-07-03; tests `TestMCPRunControlExecutorV0StopReconcilesExternalCleanupConEvidenciaSinForce`, `TestMCPRunControlExecutorV0CancelReconcilesExternalCleanupConEvidenciaSinForce`, `TestMCPRunControlHTTPHandlerV0StopExternalCleanupSinForceV0`, `TestMCPRunControlHTTPHandlerV0CancelExternalCleanupSinForceV0`, `TestMCPTransportV0RunControlStopExternalCleanupSinForce`, `TestMCPTransportV0RunControlCancelExternalCleanupSinForce`; `go test -count=1 ./modulos/orquesta-mcp -run 'TestMCP(RunControl(ExecutorV0((Stop|Cancel)ReconcilesExternalCleanupConEvidenciaSinForce|StopForcedReconcilesGoalBackendMissingAfterExternalCleanup|CancelForcedCompletaRunControlTrasReconciliarGoalV0)|HTTPHandlerV0(Stop|Cancel)ExternalCleanupSinForce)|TransportV0RunControl(Stop|Cancel)ExternalCleanupSinForce)'` | Avance: `runs/control` permite reconciliar cleanup externo no forzado para `stop` y `cancel` solo si la entrada conserva `evidence-ref-autoprogramming-goal-backend-missing-after-external-cleanup`, el backend no esta activo antes/despues y el estado Goal sigue pendiente de observacion; las superficies HTTP y transporte MCP conservan esa ruta para `stop`/`cancel` y publican `stopped`/`canceled` con diagnostico `goal_state_terminal_reconciled_after_external_cleanup`. Los cierres por alto consumo siguen exigiendo `forced=true`. BUG-065 sigue abierto para la coordinacion automatica completa backend/checkpoint/stop/cancel/wait |
| BUG-ORQ-20260701-065 | cerrado | Shutdown/Goal backend | actualizacion 2026-07-03: la reconciliacion no forzada por `run_control_reconcile_external_cleanup` persistia summary/diagnostico de `forced stop`, aunque el operador habia aportado evidencia de cleanup externo y `forced=false` | la evidencia durable podia mezclar una parada forzada con una reconciliacion gobernada posterior a limpieza externa, degradando auditoria y decisiones de operador | test `TestMCPRunControlExecutorV0StopReconcilesExternalCleanupConEvidenciaSinForce`; `go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPRunControlExecutorV0StopReconcilesExternalCleanupConEvidenciaSinForce'` | Avance: la ruta no forzada de cleanup externo usa summary `external cleanup`, diagnostico `goal_state_terminal_reconciled_after_external_cleanup` y evidencia `evidence-ref-run-control-goal-external-cleanup-reconciled`; la ruta forced conserva su evidencia y texto propios. BUG-065 sigue abierto para la coordinacion automatica completa backend/checkpoint/stop/cancel/wait |
| BUG-ORQ-20260701-065 | cerrado | Shutdown/Goal backend | actualizacion 2026-07-03: aunque la reconciliacion no forzada de cleanup externo ya corregia el `GoalWorkState`, el cierre terminal interno de `RunControl` podia usar reason default `forced ...` si el operador no aportaba texto propio | el cierre de control y el cierre de goal-first seguian usando vocabularios distintos para el mismo evento, introduciendo auditoria ambigua entre limpieza externa gobernada y parada forzada | test `TestMCPRunControlExecutorV0StopReconcilesExternalCleanupConEvidenciaSinForce`; `go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPRunControlExecutorV0StopReconcilesExternalCleanupConEvidenciaSinForce'` | Avance: `CompleteRunControlV0` usa reason default `external cleanup ...` cuando `forced=false` y la evidencia es `goal_backend_missing_after_external_cleanup`; la idempotencia publica derivada no contiene `forced` y la ruta forced mantiene defaults forced. BUG-065 sigue abierto para la coordinacion automatica completa backend/checkpoint/stop/cancel/wait |
| BUG-ORQ-20260701-065/165 | cerrado | Shutdown/Goal backend | actualizacion 2026-07-09: el servidor ya compactaba acciones stale de shutdown, pero el cliente CLI podia recibir por HTTP/status una mezcla `cleanup_completed` + `cleanup_requested` para la misma identidad y conservar la accion vieja como bloqueo aunque el work estuviera resuelto | la fuente de verdad de limpieza parcial debe compactarse tambien en el borde consumidor; no basta con que la composicion normal la emita limpia si una respuesta/status heredado conserva ambas acciones | test `TestNormalizeServerShutdownClientResultV0GoalActionCompletedOcultaAccionStaleV0`; focal `go test -count=1 ./cmd/orquesta-server -run 'Test(NormalizeServerShutdownClientResultV0GoalActionCompletedOcultaAccionStale|RequestServerShutdownV0ReadyNoSaltaGoalActionsSinActiveWork|RequestServerShutdownV0CoordinaDosGoalsActivosHastaGoalActionsResueltas|RequestServerShutdownV0PostColgadoConsultaStatusAccionable|WaitServerShutdownReadyV0RepostColgadoRespetaDeadline)'`; relacionado `go test -count=1 ./modulos/orquesta-server-shutdown ./modulos/orquesta-server ./cmd/orquesta-server` | Cierre acotado: `shutdownClientBlockingGoalActionsV0` descarta acciones no terminales si existe `cleanup_completed` para la misma identidad `kind/run_ref/work_ref/external_work_ref`; `normalizeServerShutdownClientResultV0` ya no infla `active_work_count` ni refs por acciones resueltas. BUG-065/165 sigue abierto para smoke real amplio con proveedor lento o corte externo, no por este borde local del cliente |
| BUG-ORQ-20260701-065 | cerrado | Shutdown/Goal backend | actualizacion 2026-07-03: la misma ambiguedad de auditoria podia reaparecer en `cancel` no forzado por cleanup externo si el operador no aportaba `reason` propio | `stop` y `cancel` son dos salidas terminales del mismo contrato de reconciliacion; cubrir solo `stop` dejaba margen para reintroducir texto/idempotencia de parada forzada al cancelar tras cleanup gobernado | test `TestMCPRunControlExecutorV0CancelReconcilesExternalCleanupConEvidenciaSinForce`; `go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPRunControlExecutorV0CancelReconcilesExternalCleanupConEvidenciaSinForce'` | Avance: el test de `cancel` verifica que `CompleteRunControlV0` usa reason default de `external cleanup`, no contiene `forced` en reason/idempotency y conserva evidencia `goal_backend_missing_after_external_cleanup`; BUG-065 sigue abierto para la coordinacion automatica completa backend/checkpoint/stop/cancel/wait |
| BUG-ORQ-20260701-065 | cerrado | Shutdown/Goal backend | actualizacion 2026-07-03: las superficies HTTP y transporte MCP de `runs/control` aun podian depender de que el cliente enviase `reason` para demostrar que `stop/cancel` por cleanup externo no forzado no se auditaba como forced | el contrato publico debe conservar la semantica de cleanup gobernado tambien cuando el cliente compacto solo envia accion, refs y evidencia; no debe depender de texto narrativo del operador | tests `TestMCPRunControlHTTPHandlerV0StopExternalCleanupSinForceV0`, `TestMCPRunControlHTTPHandlerV0CancelExternalCleanupSinForceV0`, `TestMCPTransportV0RunControlStopExternalCleanupSinForce`, `TestMCPTransportV0RunControlCancelExternalCleanupSinForce`; `go test -count=1 ./modulos/orquesta-mcp -run 'TestMCP(RunControlHTTPHandlerV0(Stop|Cancel)ExternalCleanupSinForce|TransportV0RunControl(Stop|Cancel)ExternalCleanupSinForce)'` | Avance: las cuatro pruebas quitan `reason` explicito y verifican que `CompleteRunControlV0` recibe reason default `external cleanup`, sin `forced` en reason/idempotency, conservando evidencia de cleanup externo. BUG-065 sigue abierto para la coordinacion automatica completa backend/checkpoint/stop/cancel/wait |
| BUG-ORQ-20260701-065/088 | cerrado | Director stats/status publico | actualizacion 2026-07-03: `orquesta.director.stats.v0` ya publicaba `external_job.evidence_refs/diagnostics` y `goal.evidence_refs/issue_codes`, pero el descriptor compacto no declaraba esos campos ni el `external_goal_ref` que ya transportaban `external_job`/`goal` | clientes MCP/HTTP compactos podian consumir `director.stats` como si no transportase evidencias de cleanup, alto consumo, receipt, goal externo o diagnosticos external-work, perdiendo contexto antes de decidir `runs/control` o rework | test `TestMCPDirectorStatsToolDescriptorV0ExponeContratoCompacto`; `go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPDirectorStatsToolDescriptorV0ExponeContratoCompacto'` | Cierre del residual descriptor: el descriptor declara `external_job?{status,status_reason?,external_goal_ref?,issue_refs?,evidence_refs?,diagnostics?}` y `goal?{goal_ref,external_goal_ref?,status,closure_status?,issue_codes?,evidence_refs?}`. Estado vigente: BUG-065 permanece abierto en sus filas propias para coordinacion automatica completa; BUG-088 quedo cerrado funcionalmente por el smoke real alto consumo/checkpoint |
| BUG-ORQ-20260701-065/088 | cerrado | Run queue/status publico | actualizacion 2026-07-03: `orquesta.run_queue.priority.v0` transportaba `evidence_refs` en candidatos rankeados, terminales y actualizados, pero el descriptor compacto no lo declaraba | consumidores compactos de cola podian perder la evidencia causal de rescate, supersedes, cleanup o decision antes de priorizar o coordinar `runs/control` | test `TestMCPRunQueuePriorityDescriptorV0DeclaraEvidenciaDeCandidatos`; `go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPRunQueuePriorityDescriptorV0DeclaraEvidenciaDeCandidatos'` | Cierre del residual descriptor: el descriptor declara `evidence_refs?` en `ranked`, `terminal` y `updated`, conservando la evidencia de prioridad/cola sin cambiar reglas de scheduling. Estado vigente: BUG-065 permanece abierto en sus filas propias para coordinacion automatica completa; BUG-088 quedo cerrado funcionalmente por el smoke real alto consumo/checkpoint |
| BUG-ORQ-20260701-066/088 | cerrado | External work/status publico | actualizacion 2026-07-03: `orquesta.external_work.run.v0` transportaba `external_goal_ref` y `evidence_refs`, pero el descriptor compacto solo declaraba `goal_ref`, `next_actions` y endpoints | consumidores de trabajo externo podian arrancar un goal-first y perder en discovery la evidencia/goal externo necesaria para observar, reconciliar alto consumo o coordinar `runs/control` sin mirar logs | test `TestMCPExternalWorkRunDescriptorV0DeclaraLegacyExplicito`; `go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPExternalWorkRunDescriptorV0DeclaraLegacyExplicito'` | Cierre del residual descriptor: el descriptor declara `external_goal_ref?` y `evidence_refs?` en la salida ok de `external_work.run`; no cambia routing ni toca OPES productivo. Estado vigente: siguen pendientes criterios OPES done/settled y coordinacion automatica de BUG-066 en sus filas propias; BUG-088 ya no aporta pendiente de smoke real |
| BUG-ORQ-20260701-066/088 | cerrado | Arranque director/status publico | actualizacion 2026-07-03: `orquesta.apps.arrancar_director.v0` transportaba `external_goal_ref` y `evidence_refs`, pero el descriptor de arranque solo anunciaba `goal_ref/goal_status` | clientes que arrancan apps goal-first podian perder en discovery el goal externo y la evidencia inicial necesaria para observar, continuar o reconciliar estados de alto consumo/shutdown | test `TestMCPAppSpecDescriptorsV0PublicanRoutePolicy`; `go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPAppSpecDescriptorsV0PublicanRoutePolicy'` | Cierre del residual descriptor: el descriptor declara `external_goal_ref?` y `evidence_refs?` en ok/error, y la prueba de route policy fija esa salida. Estado vigente: siguen pendientes criterios OPES done/settled y coordinacion automatica de BUG-066 en sus filas propias; BUG-088 ya no aporta pendiente de smoke real |
| BUG-ORQ-20260701-066 | abierto | Goal-first/OPES cierre y observacion | ola OPES 3 produjo artefactos utiles en seis temas, pero algunos goals siguieron reescribiendo tras entregas suficientes, quemaron millones de tokens, quedaron `active` en SQLite tras matar procesos, `server/shutdown` devolvio HTTP 400 sin JSON y `GET /api/v0/external-work/observe` devolvio 404 | faltan criterios nativos de done/settled para OPES goal-first, observacion estable por external-work y reconciliacion terminal tras corte de proceso; el director no puede distinguir esperar, cerrar, replanificar o cortar | `TAREA_OPES_ORQUESTA_GRUPO_B_GOAL_FIRST_OBSERVE_Y_QA_MINIMOS_2026-07-01.md` apartado 13; avance de endpoint estable con tests `TestNewAppGatewayMuxV0RegistersConfiguredRoutes`, `TestExternalWorkObserveAliasDelegaEnObserveGoalRESTV0`, `TestExternalWorkObserveGETLegacyDelegaEnObserveGoalRESTV0`; avance stats goal-first con `TestCodexStackExternalJobStatsSourceV0GoalFirstAcceptedConIssuesNoCompletaJob` y `TestCodexStackExternalJobStatsSourceV0GoalFirstAcceptedSinReceiptNoCompletaJob`; avance accionable con `TestMCPQueueGlobalStatusNormalizeRecommendedActionV0PreservaAccionesGoalFirstEspecificas`, `TestMCPQueueGlobalStatusHTTPHandlerV0ConservaRepairReceiptGoalFirst` y `TestMCPDomainWorkStatusHTTPHandlerV0ConservaRepairReceiptGoalFirst`; `go test -count=1 ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway`; `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestCodexStackExternalJobStatsSourceV0GoalFirst(AceptadoCompletaJob|AcceptedConIssuesNoCompletaJob|AcceptedSinReceiptNoCompletaJob)'`; `go test -count=1 ./modulos/orquesta-mcp -run 'Test(MCPQueueGlobalStatus(NormalizeRecommendedActionV0PreservaAccionesGoalFirstEspecificas|HTTPHandlerV0ConservaRepairReceiptGoalFirst)|MCPDomainWorkStatusHTTPHandlerV0ConservaRepairReceiptGoalFirst)'` | Avance: `/api/v0/external-work/observe` queda registrado como alias POST estable de `/api/v0/apps/director/goal/observe`, conserva cuerpo/correlacion y delega en el executor goal-first existente; el manifest lo marca como ruta control-plane mutation, eliminando el 404 historico para consumidores external-work. Nuevo avance: el alias tambien acepta GET legacy con `run_ref`, `request_id`, `correlation_id`, `occurred_at` y `requested_by` por query, lo traduce a POST interno JSON y mantiene el handler canonico como unica logica de observacion, cerrando el caso historico exacto `GET /api/v0/external-work/observe` sin reintroducir loop legacy. Nuevo avance: `external_job_stats` ya no proyecta un goal-first como `completed` si el cierre persistido es contradictorio (`Accepted=true` con issues o `NeedsRework`) ni si falta `DomainReceiptRefs`; los issues de cierre o la ausencia de recibo de dominio ganan y el job externo queda `blocked`, evitando falso settled de OPES cuando falta contrato de calidad, receipt reparable o artefacto asentado. Nuevo avance: la ausencia de recibo de dominio en un cierre aceptado publica `status_reason=goal_first_closure_missing_domain_receipt` y diagnostico especifico de reparar receipt terminal, separado de `goal_first_closure_blocked` por issues. Nuevo avance: `queue/global-status` conserva acciones goal-first reparables (`repair_receipt`, rework de write-set/texto, revision de parciales y continuacion de checkpoint) en vez de degradarlas a reparacion generica, el handler HTTP conserva `repair_receipt` de un stale-running goal-first y la fachada `/api/v0/domain-work/status` conserva la misma accion para OPES/domain-work, de modo que el operador no pierde la accion tras consultar estado global o de dominio. Pendiente: criterios OPES done/settled por artefactos requeridos, evitar reescritura tardia sin rework causal y reconciliacion automatica tras cortes externos/manuales |
| BUG-ORQ-20260701-067 | cerrado | OPES/contador canonico QA | en ola OPES 3, temas 038 y 041 parecian cumplir por `wc -w`, pero el contador estricto OPES daba 10336 y 10531 palabras; ademas un informe podia decir `status=fail` mientras el texto afirmaba que superaba 10.800 | hay doble contador de palabras y se toleran informes textuales contradictorios; el contrato ejecutable BUG-064 debia ampliarse para usar contador canonico real de OPES y no aceptar `wc` ni narrativa del agente como verde | `TAREA_OPES_ORQUESTA_GRUPO_B_GOAL_FIRST_OBSERVE_Y_QA_MINIMOS_2026-07-01.md` apartado 14; tests `TestValidateOPESTopicQualityContractV0PriorizaContadorCanonicoSobreWCV0`, `TestValidateOPESTopicQualityContractV0NoAceptaWCSinCanonicoV0`, `TestValidateOPESTopicQualityContractV0DetectaInformeContradictorioV0`, `go test -count=1 ./modulos/orquesta-opes-director` | Cierre en contrato OPES puro: `CanonicalWordCount`/`CanonicalWordCountText` tienen prioridad sobre `wc`, fuentes `wc` sin contador canonico emiten `opes_canonical_word_count_required`, el conteo por texto usa una expresion equivalente al contador OPES y los informes contradictorios emiten `invalid_report_contract`. El cableado residual en runner/cierre OPES queda dentro de BUG-058 |
| BUG-ORQ-20260701-068 | cerrado | API discovery/capabilities | en ola OPES 4, `GET /api/v0/routes` devolvio 404; `external-work/run` respondio `next_actions=["observe_active_goals"]`, pero no indico URL ni contrato canonico de observacion, y `GET /api/v0/external-work/observe` ya habia devuelto 404 | la composicion publica rutas reales en varios sitios, pero faltaba una superficie canonica de capacidades/descubrimiento que pueda consumir OPES sin memoria operacional de endpoints; esto repite el eje de discovery cerrado parcialmente en BUG-028, ahora para observacion goal-first | `TAREA_OPES_ORQUESTA_GRUPO_B_GOAL_FIRST_OBSERVE_Y_QA_MINIMOS_2026-07-01.md` apartado 15; tests `TestHandlerV0ExponeHealthStatusYDelegaV0`, `TestServerResourcesRouteManifestIncluyeDiscoveryOPESV0`, `TestCodexStackV0ExternalWorkRunConBackendGoalArrancaGoalFirstSinColaLegacy`; `go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-mcp ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server -run 'TestHandlerV0ExponeHealthStatusYDelegaV0|TestServerResourcesRouteManifestIncluyeDiscoveryOPESV0|TestCodexStackV0ExternalWorkRunConBackendGoalArrancaGoalFirstSinColaLegacy|TestMCPExternalWorkRun'` | Cierre: `/api/v0/routes` expone el manifest canonico de rutas y queda listado en `/api/v0/server/resources`; `external-work/run` devuelve `operation_endpoints.observe_goal=/api/v0/apps/director/goal/observe` y `operation_endpoints.observe_active_goals=/api/v0/autoprogramming/goals/observe-active` cuando esas acciones aparezcan en `next_actions`. BUG-069 conserva el estado operacional global cola/goals |
| BUG-ORQ-20260701-069 | cerrado | Supervisor/status goal-first | en ola OPES 4, `supervisor_tick_result` mostro `public_stop_reason=idle_no_execution` y `queue_size=0` mientras `goals_1.sqlite` tenia cuatro goals OPES `active`; el operador tuvo que combinar SQLite, audit log, `find -mmin` y procesos para saber si seguia vivo | el estado operacional mezclaba cola Orquesta y backend goal-first como fuentes separadas; `idle_no_execution` era verdad para cola pero falso como estado de trabajo global | `TAREA_OPES_ORQUESTA_GRUPO_B_GOAL_FIRST_OBSERVE_Y_QA_MINIMOS_2026-07-01.md` apartado 15; tests `TestRuntimeV0SupervisorDistingueColaIdleConGoalBackendActivoV0`, `TestRuntimeV0GoalObservationFingerprintSaltaRunSinCambiosV0`, `TestResidentOperationalStatusSourceV0ExponeGoalBackendActivoConColaVaciaV0`; `go test -count=1 ./modulos/orquesta-server`; `go test -count=1 ./cmd/orquesta-server` | Cierre: el servidor residente reconcilia cola vacia con ultimo snapshot activo del observer goal-first; `supervisor_tick_result` y status publico proyectan `queue_idle_but_goal_backend_active`/`goal_backend` con refs de run/goal saneadas y contadores `goal_backend_active`; el diagnostico operacional expone contadores/refs del backend activo y la automejora idle queda bloqueada mientras esa condicion siga viva. No toca OPES productivo ni lee SQLite directamente |
| BUG-ORQ-20260701-070 | cerrado | Goal-first/usage_limited | en olas OPES 5 y 6, goals lanzados por `external-work/run` terminaron como `usage_limited`; en un caso habia artefactos parciales utiles y en seis casos quedaron `tokens_used=0` sin artefactos, mientras `observe-active` devolvia solo `{"estado":"ok"}` | `usage_limited` no distinguia cuota, presupuesto, politica ni limite de backend; `external-work/run` podia publicar `goal-set` aunque el backend ya no pudiera trabajar y la observacion publica no devolvia estado terminal reciente ni next action | `TAREA_OPES_ORQUESTA_GRUPO_B_GOAL_FIRST_OBSERVE_Y_QA_MINIMOS_2026-07-01.md` apartados 16 y 17; tests `TestObserveActiveGoalWorksV0DevuelveSnapshotTerminalAccionable`, `TestMCPAutoprogrammingObserveActiveGoalsToolExecutorV0PublicaSnapshotUsageLimited`, `TestObserveActiveGoalWorksV0PublicaSnapshotTerminalSinReobservar`, `TestServerCodexAppServerGoalBackendV0BloqueaLaunchSiGoalYaEstaUsageLimitedV0`, `TestServerCodexAppServerGoalBackendV0ObservaUsageLimitedConCausaOperableV0`, `TestExternalWorkGoalFirstKnownLaunchFailureReasonV0ClasificaUsageLimit`, `go test -count=1 ./...` | Cierre: `usageLimited`/`quotaLimited`/`providerLimited` se proyectan como `codex_app_server_goal_provider_limited`, `budgetLimited` como `codex_app_server_goal_budget_limited` y `policyLimited` como `codex_app_server_goal_policy_limited`; `observe-active` lista tambien terminales `blocked/invalid`, devuelve snapshot parcial para terminales accionables o refs solicitadas y publica issues/next actions; el stack residente no reobserva estados terminales; `external-work/run` recibe launch failed accionable si el backend ya informa limite inmediato sin consumir tokens. No toca OPES productivo |
| BUG-ORQ-20260701-071 | cerrado | Goal-first/active timeout | en ola OPES 7, dos goals quedaron `active`, consumieron cientos de miles de tokens sin artefactos, la cola paso a `stopped/blocked` con `goal_active_timeout` y el backend siguio activo; shutdown devolvio `shutdown_ready=true` y dejo procesos `codex app-server` residuales | la politica de timeout/corte no conservaba suficiente snapshot operacional ni reconciliaba cola, backend y proceso; el cierre de servidor podia declarar ready aunque el app-server propio continuase consumiendo | `TAREA_OPES_ORQUESTA_GRUPO_B_GOAL_FIRST_OBSERVE_Y_QA_MINIMOS_2026-07-01.md` apartado 18; relacionado con BUG-065, BUG-066, BUG-069 y BUG-073; tests `TestMCPAutoprogrammingStatusExecutorV0GoalActiveTimeoutConBackendActivoEsReplanAccionable`, `TestShutdownServerV0NoForzadoBloqueaConBackendStillRunning`, `TestMCPServerShutdownHTTPHandlerV0BackendStillRunningDevuelveConflict`, `TestStackShutdownActiveWorkReaderV0BloqueaBackendActivoTrasGoalTimeout`, `TestStackGoalMaterializedRefsSourceV0DetectaCheckpointEnWriteSet` | Cierre acotado: `autoprogramming/status` publica `goal_active_timeout_backend_active` con accion `replan_goal_after_active_timeout`, snapshot de backend activo, tokens y refs; `checkpoint_started.txt` dentro del write-set se materializa como `artifact-ref-checkpoint`; `server/shutdown` devuelve `backend_still_running`/HTTP 409 cuando detecta backend goal vivo tras timeout. La politica fina de timeout tras checkpoint y snapshot rapido de `observe_goal` quedan abiertos en BUG-073 |
| BUG-ORQ-20260701-072 | cerrado | Goal-first/artefactos parciales | en ola OPES 8, una tarea minima produjo `checkpoint_started.txt`, `matriz_fuentes_reutilizacion.md`, `plan_rework_por_fases.md`, `docs/orquesta_goal_result_v0.json` y `orquesta_phase0_checkpoint_delivery.json`, pero la API publico `goal_first_blocked_no_artifacts`, `goal_backend_state_unreconciled`, `active_runs=0`, `observe_goal` timeout y `observe-active` vacio mientras SQLite seguia `active`; ademas el goal_result declaraba tests `passed` con `evidence_refs=[]` | el cierre goal-first no reconciliaba artefactos escritos/tardios, checklist esperado, required-test evidence, estado publico ni rework causal terminal, y podia decir "sin artefactos" aunque hubiera entrega util de fase 0 no publicable | `TAREA_OPES_ORQUESTA_GRUPO_B_GOAL_FIRST_OBSERVE_Y_QA_MINIMOS_2026-07-01.md` apartado 19; relacionado con BUG-058, BUG-060 y BUG-071; avance `docs/incidencias/incidencia_orquesta_goal_first_qa_failed_public_text_2026-07-02.md`; tests `TestStackGoalMaterializedRefsSourceV0DetectaQAFailedPublicTextEnContextoOPES`, `TestMCPDirectorStatsToolExecutorV0GoalFirstProyectaQAFailedPublicTextV0`, `TestMCPAutoprogrammingStatusExecutorV0QAFailedPublicTextPideReworkV0`, `TestEnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0QAFailedPublicTextPideRework`, `TestStackGoalMaterializedRefsSourceV0DetectaArtefactosParcialesSinReceiptTerminal`, `TestEnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0ArtefactosParcialesPideRevision`, `TestMCPDirectorStatsToolExecutorV0GoalFirstProyectaArtefactosParcialesV0`, `TestMCPAutoprogrammingStatusExecutorV0ArtefactosParcialesPideRevisionV0`, `TestStackGoalMaterializedRefsSourceV0DetectaRequiredTestEvidenceAusenteEnReceiptTerminal`, `TestEnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0RequiredTestEvidenceAusentePideRepairReceipt`, `TestMCPDirectorStatsToolExecutorV0GoalFirstProyectaRequiredTestEvidenceAusenteV0`, `TestMCPAutoprogrammingStatusExecutorV0RequiredTestEvidenceAusentePideRepairReceiptV0`, `TestStackGoalMaterializedRefsSourceV0PublicaChecklistEsperadoDesdeSpec`, `TestStackGoalMaterializedRefsSourceV0DetectaPhase0CheckpointDeliveryNoPublicable`, `TestEnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0Phase0NoPublicablePideContinuar`, `TestMCPDirectorStatsToolExecutorV0GoalFirstProyectaPhase0NoPublicableV0`, `TestMCPAutoprogrammingStatusExecutorV0Phase0NoPublicablePideContinuarV0`, `TestStackGoalMaterializedRefsSourceV0UsaRequiredTestEvidenceMaterializadaFueraDeReceipt`, `TestRunSupervisorGoalFirstResidentPreparaReworkPorArtefactosParcialesTerminalesV0`, `TestRunSupervisorGoalFirstResidentNoReworkPorIssueRecuperableActivoV0`, `go test -count=1 ./modulos/orquesta-app-codex-stack ./modulos/orquesta-mcp` | Cierre: materialized refs detecta JSON de QA fallida dentro del `write_set`, proyecta `qa_failed_public_text` como bloqueo recuperable en stats/status/observe y conserva artefactos; el cierre external-work deriva `artifact_paths` desde `file_ref` real del ledger aceptado; artefactos escritos sin receipt terminal publican `partial_artifacts_written` con accion `review_partial_artifacts`; receipts con tests `passed` sin `evidence_refs` publican `required_test_evidence_missing` y accion `repair_receipt`; el checklist esperado de `GoalWorkSpec` queda como refs opacas; `orquesta_phase0_checkpoint_delivery.json` se publica como `phase0_complete_non_publishable` con accion `continue_from_phase0_checkpoint`; `required_test_evidence.v0` materializado dentro del `write_set` se ingiere como evidencia; y `runs.supervisor` en `resident_mode` lanza rework causal, idempotente y acotado para issues recuperables terminales, sin relanzar si el goal sigue activo |
| BUG-ORQ-20260701-073 | cerrado funcionalmente | Goal-first/timeout tras checkpoint | con Orquesta `79f21136`, ola OPES 10 tema 001 fase 1 escribio `checkpoint_started.txt`, pero a los ~75 s corto con `goal_active_timeout`, `goal_first_blocked_no_artifacts`, `goal_backend_state_unreconciled`; `observe_goal` siguio dando timeout y `shutdown_ready=true` dejo vivo el `codex app-server` | la observacion mejoro respecto a BUG-070, pero el timeout/cierre seguia cortando demasiado pronto un goal que acababa de producir checkpoint; parte del falso "sin artefactos" y el falso `shutdown_ready` quedaron mitigados por BUG-071 y cierres posteriores, y el residual de limite duro proveedor queda en BUG-079 | `TAREA_OPES_ORQUESTA_GRUPO_B_GOAL_FIRST_OBSERVE_Y_QA_MINIMOS_2026-07-01.md` apartado 20; evidencia en `opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_responses/wave10_phase1_t001/status_after_75s.json`; tests `TestMCPAutoprogrammingObserveGoalHTTPHandlerV0TimeoutDevuelveJSONPublico`, `TestMCPAutoprogrammingObserveGoalHTTPHandlerV0TimeoutIncluyeSnapshotParcialV0`; avance `docs/incidencias/incidencia_orquesta_goal_first_timeout_checkpoint_reciente_2026-07-02.md`; tests `TestMCPAutoprogrammingStatusExecutorV0CheckpointOnlyBajoConsumoPideSiguienteArtefacto`, `TestMCPAutoprogrammingStatusExecutorV0CheckpointOnlyConsumoMedioAvisaAntesDeUmbralAlto`, `TestMCPAutoprogrammingStatusExecutorV0GoalActiveTimeoutConCheckpointRecienteNoReplanificaAun`, `TestMCPAutoprogrammingStatusExecutorV0GoalActiveTimeoutConCheckpointEstancadoPromueveReplanV0`, `TestMCPAutoprogrammingStatusExecutorV0GoalActiveTimeoutConCheckpointRespetaUmbralConfiguradoV0`, `TestStackGoalMaterializedRefsSourceV0DetectaCheckpointEnWriteSet`, `TestServerConfigFromEnvV0PublicaUmbralCheckpointGoalConfigurableV0`, `TestRunSupervisorGoalFirstResidentPreparaReworkPorCheckpointHighConsumptionV0`, `TestRunSupervisorGoalFirstResidentReworkEsIdempotenteV0`, `TestRunSupervisorGoalFirstNoResidentNoLanzaReworkV0`, `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server` | Cierre funcional: `POST /api/v0/autoprogramming/goal/observe` cancela el executor lento, intenta snapshot rapido con `GoalStateStore` y devuelve 504 publico con `partial=true`, refs/estado/artefactos/evidencias y accion heredada del snapshot. `autoprogramming/status` distingue `active_no_checkpoint_yet`, `active_checkpoint_only_yet`, `checkpoint_only_consumption_warning`, `active_timeout_checkpoint_recent` y `checkpoint_only_high_consumption`; si ya existe checkpoint inicial, pide `observe_goal_backend_require_next_artifact` y no otro checkpoint, y si se agota ventana/consumo promociona a `replan_narrow_context`. `queue/global-status` conserva esas acciones. La prueba materializada fija que `checkpoint_started.txt` no cuenta como artefacto parcial ni `artifact-ref-materialized:*`. El smoke real acotado de checkpoint -> segundo artefacto/review ya no reproduce falso `goal_first_blocked_no_artifacts` ni app-server residual; el residual de cap duro pre-tool/stdout queda en BUG-079 |
| BUG-ORQ-20260701-075 | abierto | Goal-first/QA de artefactos parciales | en ola OPES 11 tema 002, Orquesta `d40ad337` produjo ampliado, resumen, tests, HTML, visual y RAG, pero `autoprogramming/status` acabo en `goal_first_blocked`/`goal_backend_state_unreconciled` y `observe_goal` siguio devolviendo timeout parcial. Los artefactos quedaron recuperables pero no publicables: Markdown visible con anclas `{#...}`, tablas colapsadas en encabezados y efectos de correccion masiva como `órgaños`, `confíanza`, `instituciónal`, `propuestá`; no se emitio ACK final/cierre causal con QA fallida y rework concreto | el goal-first no separa bien entrega parcial recuperable, validacion de calidad publicable y cierre operacional. Puede bloquear tras escribir mucho material sin producir un resultado determinista que diga `partial_artifacts_written`, `qa_failed_public_text`, lista de ficheros validos/no validos y replan automatico focal | `TAREA_OPES_ORQUESTA_GRUPO_B_GOAL_FIRST_OBSERVE_Y_QA_MINIMOS_2026-07-01.md` apartado 21; evidencia local en `OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/02_temas/tema_002/` y respuestas `wave11_single_t002_post_fix/status_*.json`/`observe_goal_t002_*.json`; avance `docs/incidencias/incidencia_orquesta_goal_first_qa_failed_public_text_2026-07-02.md`; tests `TestStackGoalMaterializedRefsSourceV0DetectaQAFailedPublicTextEnContextoOPES`, `TestMCPDirectorStatsToolExecutorV0GoalFirstProyectaQAFailedPublicTextV0`, `TestMCPAutoprogrammingStatusExecutorV0QAFailedPublicTextPideReworkV0`, `TestEnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0QAFailedPublicTextPideRework`, `TestStackGoalMaterializedRefsSourceV0DetectaArtefactosParcialesSinReceiptTerminal`, `TestEnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0ArtefactosParcialesPideRevision`, `TestMCPDirectorStatsToolExecutorV0GoalFirstProyectaArtefactosParcialesV0`, `TestMCPAutoprogrammingStatusExecutorV0ArtefactosParcialesPideRevisionV0`, `TestRunSupervisorGoalFirstResidentPreparaReworkPorQAFallidaPublicaV0`, nuevo contrato `OPESArtifactQualityContractV0` y tests `TestValidateOPESArtifactQualityContractV0CubreWorkKindsMinimosV0`, `TestProduceOPESCausalJobsV0VisualConEvidenceRefPeroContratoFallidoCreaReworkV0`, `TestProduceOPESCausalJobsV0HTMLConContratoArtifactQualityPassNoCreaReworkV0`, `TestCodexStackV0OPESGoalFirstArtifactQualityHTMLDerivadoReworkYPassV0`; `go test -count=1 ./modulos/orquesta-opes-director ./modulos/orquesta-opes-bridge`; `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestCodexStackV0OPESGoalFirstArtifactQualityHTMLDerivadoReworkYPassV0'` | Avance: si el `write_set` contiene artefactos y un informe JSON de QA fallida, Orquesta publica `qa_failed_public_text`, accion `rework_public_text`, refs de artefacto y evidencia en `director.stats`, `autoprogramming/status` y `observe_goal`; en OPES se exige fallo estructurado de la terna canonica y no se usa texto libre como veto; los artefactos escritos sin receipt terminal publican `partial_artifacts_written` y accion `review_partial_artifacts`; `runs.supervisor` en `resident_mode` lanza rework causal idempotente para `qa_failed_public_text` terminal con refs de evidencia materializadas. Nuevo avance: los informes QA estructurados pueden declarar `valid_artifact_paths`/`invalid_artifact_paths` o artefactos con `path` + `status`/`valid`; materialized refs los proyecta como `artifact-ref-materialized-valid:*` y `artifact-ref-materialized-invalid:*` con evidencias `valid-artifact-list`/`invalid-artifact-list`, sin exponer paths absolutos ni descartar artefactos recuperables. Nuevo avance 2026-07-04 noche 30: `OPESArtifactQualityContractV0` valida campos estructurados minimos para HTML, visuales, audio, tutor/RAG, fuentes, revisiones, supuestos, juegos, ayuda y reutilizacion visual; si una entrega terminal tiene evidencia nominal pero carece de manifest/refs/QA estructurada, el registry queda `needs_rework` con `artifact_quality_contract_failed` y followup `review_artifact_quality`; si falta evidencia minima total, conserva `required_evidence_missing`. Nuevo avance 2026-07-04 noche 33: el stack goal-first ya prueba el puente completo para HTML derivado: receipt aceptado, productor OPES, rework causal si falta manifest/reporte estructurado y ausencia de rework si el contrato artifact quality pasa. Pendiente: smoke temporal OPES/external-work con artefactos reales y proveedor, y prueba de ausencia de reescritura tardia |
| BUG-ORQ-20260701-076 | cerrado | Shutdown forzado/app-server residual | tras la ola OPES 11, `server/shutdown` sin `forced` devolvio correctamente `backend_still_running`, pero el shutdown forzado con `shutdown_ready=true` dejo vivos app-servers Codex de la misma ola: PIDs `2779012` y `2779026` para socket `/tmp/oq-gsrv-1000-f5c5800ce032825e/s.sock`; ademas seguian vivos restos del primer arranque fallido de la ola (`2775615`, `2775616`, `2775630`) | el cierre forzado declaraba ready antes de consultar trabajo activo; corregido para no ignorar `goal_backend/backend_still_running` aunque `forced=true`. El cleanup de startup fallido tambien limpia la sesion `orquesta-goal-*` configurada aunque falte `owner.json` si el daemon lanzado ya no vive, y ahora reutiliza el cleaner activo para matar procesos propios sin socket recuperable bajo `RuntimeWorkDir/goal-srv`. Sigue abierto el residual arquitectonico: reconciliar tras limpieza manual externa o fuera de la identidad tmux con evidencia real amplia | `TAREA_OPES_ORQUESTA_GRUPO_B_GOAL_FIRST_OBSERVE_Y_QA_MINIMOS_2026-07-01.md` apartado 22; incidencia `docs/incidencias/incidencia_orquesta_shutdown_detecta_restos_tmux_goal_2026-07-02.md`; tests `TestShutdownServerV0ForzadoBloqueaConBackendStillRunning`, `TestShutdownServerV0ForzadoNoBloqueaConGoalActivo`, `TestStackShutdownActiveWorkReaderV0BloqueaBackendActivoTrasGoalTimeout`, `TestCleanupCodexGoalBackendAfterStartupFailureIfDaemonGoneV0MataSesionConfiguradaSinOwnerMarker`, `TestCleanupCodexGoalBackendAfterStartupFailureIfDaemonGoneV0MataProcesoPropioSinSocketV0`, `TestCodexAppServerTmuxBackendV0ShutdownNormalNoMataSesionSinOwnerMarkerV0`, `TestCodexAppServerTmuxBackendV0CleanupActiveWorkLimpiaOwnerMarkerRecuperableEscaneadoV0`, `TestCodexAppServerTmuxBackendV0ReadActiveShutdownWorkDetectaOwnerMarkerRecuperableFueraDeSesionConfiguradaV0`, `TestCodexAppServerTmuxBackendV0ReadActiveShutdownWorkDetectaProcesoConSocketConfiguradoSinMarkerV0`, `TestCodexAppServerTmuxBackendV0ReadActiveShutdownWorkDetectaProcesoEnRuntimeWorkdirSinSocketConfiguradoV0`, `TestCodexAppServerTmuxBackendV0ReadActiveShutdownWorkDetectaProcesoPropioSinSocketPorRuntimeWorkdirV0`, `TestCodexAppServerTmuxBackendV0CleanupActiveWorkMataProcesoPropioSinSocketV0`, `TestCodexAppServerTmuxCommandsContainRuntimeWorkdirSocketV0IgnoraSocketExternoV0`, `go test -count=1 ./cmd/orquesta-server` | Cierre parcial: forced shutdown consulta `ActiveWorkReader`, permite drenar goals normales, pero devuelve `backend_still_running` y no solicita stop si el backend Goal sigue vivo; si `start` falla y el daemon ya murio, se apaga la sesion configurada sin owner marker y se elimina socket. Nuevos avances: `app_server_tmux` escanea owner markers propios recuperables fuera de la sesion configurada (`RuntimeWorkDir/goal-srv` y fallback `oq-gsrv-*` cuando aplica) y tambien detecta un `codex app-server --listen` que apunta exactamente al socket configurado aunque falten marker/sesion; ambos se proyectan como `goal_backend/backend_still_running` con refs compactas, sin exponer sockets ni PIDs ni bloquear por nombre de proceso suelto; ademas detecta procesos `codex app-server` que escuchan en sockets bajo `RuntimeWorkDir/goal-srv` aunque el socket configurado ya no coincida, y procesos `codex app-server` sin `--listen` recuperable cuando conservan cwd o `CODEX_HOME` bajo el `RuntimeWorkDir/goal-srv` propio; el cleanup queda cubierto para owner markers propios recuperables escaneados fuera de la sesion configurada, para procesos propios sin socket recuperable bajo `RuntimeWorkDir/goal-srv` y para la ruta de fallo de arranque que solo se activa si el PID del daemon ya no vive, eliminando sesion, marker, sockets y procesos antes de dejar de reportar active work. Pendiente: smoke/escenario real de cleanup externo amplio que pruebe reconciliacion completa tras cortes manuales |
| BUG-ORQ-20260701-077 | cerrado | Startup/base_url y app-server residual | en Orquesta `433129d`, arranque aislado wave12 escribio `state/orquesta_server_state_v0.json` con `addr=127.0.0.1:46483`, `startup_ready=true` y ticks de auditoria, pero no materializo `base_url.txt`; el script operador trato la ausencia como fallo y al salir quedaron vivos tmux/app-server `f34a7f9d3c61522b` sin servidor HTTP alcanzable | el contrato de arranque no garantizaba una salida unica y atomica de endpoint consumible por OPES; corregido para materializar `base_url.txt` durable en `RuntimeWorkDir` antes de publicar estado `running` con `addr`. El residual amplio quedo auditado y cubierto por guardas: helper comun, smoke goal-first, smokes aislados principales, smoke OPES agent, restart-state, wrappers `modulos/*/arrancar_codex.sh`, wrapper manual `inicio_agente`, manual vigente `uso_actual_app_orquesta` y guardas automaticas para futuros scripts con servidor temporal o arranque delegado por helper, para no omitir `runtime_dir`, trap de cleanup ni shutdown HTTP directo sin cleanup gobernado | `TAREA_OPES_ORQUESTA_GRUPO_B_GOAL_FIRST_OBSERVE_Y_QA_MINIMOS_2026-07-01.md` apartado 23; tests `TestRuntimeV0PublicaBaseURLFileEnRuntimeWorkDirV0`, `TestSmokeCommonShutdownCleanupMataAppServerPropioSinBaseURLV0`, `TestSmokeCommonShutdownCleanupBackendGoalSiWrapperCancelaV0`, `TestSmokesAisladosPasanRuntimeDirAlShutdownComunV0`, `TestScriptsQueArrancanServidorTemporalUsanShutdownComunV0`, `TestScriptsQueDeleganArranqueServidorTemporalInstalanCleanupV0`, `TestScriptsQueUsanShutdownComunPasanRuntimeDirV0`, `TestScriptsConShutdownDirectoPidenCleanupGoalBackendsV0`, `TestInicioAgenteNoRecomiendaRuntimeManualV0`, `TestArrancarCodexModuloNoRecomiendaRuntimeManualV0`, `TestUsoActualAppOrquestaRecomiendaServidorGestionadoV0`; `bash -n scripts/lib/smoke_common.sh scripts/smoke_goal_first_app_server_real.sh scripts/smoke_autoprogramming_supervised.sh scripts/smoke_codex_required_test_runner_state_file.sh scripts/smoke_orquesta_server_rest_director.sh scripts/smoke_autoprogramming_bolsa_real.sh scripts/smoke_external_domain_fake_real.sh scripts/smoke_external_domain_non_opes_real.sh scripts/smoke_opes_reviews_providers_real.sh scripts/lib/opes_agent_smoke_ops.sh scripts/smoke_orquesta_server_restart_state.sh scripts/inicio_agente.sh`; `bash -n modulos/*/arrancar_codex.sh` | Cierre parcial: `RuntimeV0` escribe `base_url.txt` con permiso `0600` en el runtime antes de persistir serving, de modo que OPES no depende de reconstruir URL desde state. Nuevo avance: `smoke_shutdown_orquesta_server` envia `cleanup_goal_backends=true` cuando hay HTTP y acepta `runtime_dir` para limpiar, incluso sin `base_url`, recursos `app_server_tmux` propios bajo `RuntimeWorkDir/goal-srv`: owner/session `orquesta-goal-*`, sockets locales y procesos `codex app-server` con cwd o `CODEX_HOME` bajo ese goal-srv; lo usan ya `smoke_goal_first_app_server_real`, `smoke_autoprogramming_supervised`, `smoke_codex_required_test_runner_state_file`, `smoke_orquesta_server_rest_director`, `smoke_autoprogramming_bolsa_real`, `smoke_external_domain_fake_real`, `smoke_external_domain_non_opes_real`, `smoke_opes_reviews_providers_real`, `lib/opes_agent_smoke_ops` y `smoke_orquesta_server_restart_state`. Los wrappers historicos `modulos/*/arrancar_codex.sh` ya no recomiendan `go run ./cmd/orquesta-server run` como ruta vigente y apuntan a `orquesta-server start/stop` o servidor residente/cola; `inicio_agente` conserva el contrato de servidor residente y ya no recomienda `go run ./cmd/orquesta-server run` cuando readiness no responde; el manual vigente `uso_actual_app_orquesta` recomienda `orquesta-server start/stop` y limita `go run ./cmd/orquesta-server run` a harnesses aislados con runtime, state, loopback y cleanup explicitos; ademas, todo script que arranque servidor temporal mediante un comando real `orquesta-server run` en background, o con `ORQUESTA_SERVER_ADDR` y captura de PID via `$!`, queda cubierto por una guarda que exige shutdown comun, aunque no use el nombre exacto `server_pid`, toda invocacion a ese helper desde scripts debe pasar `runtime_dir`/`RUNTIME_DIR` para conservar cleanup sin `base_url`, toda llamada `curl -X POST` de scripts a `/api/v0/server/shutdown` queda cubierta por una guarda independiente de la variable de URL para exigir `cleanup_goal_backends`, y todo wrapper que delegue el arranque en un helper `*_start_orquesta_server` debe instalar `trap EXIT` con cleanup comun directo o delegado. Cierre: el residual de wrappers y docs operadores queda cubierto por guardas automaticas; las menciones restantes a runtime manual o puerto historico solo se permiten en incidencias historicas, backlog tecnico o harnesses aislados explicitos |
| BUG-ORQ-20260701-077 | cerrado | Startup/base_url y app-server residual | actualizacion 2026-07-03: el wrapper OPES external-work real delegaba arranque y cleanup en una libreria, por lo que convenia fijar explicitamente esa cobertura en guardas | un script principal puede cumplir el contrato mediante `trap smoke_cleanup EXIT` y helper compartido; la guarda debe verificar tambien la ruta delegada y no solo llamadas directas a `smoke_shutdown_orquesta_server`; un handoff de cierre puede quedar stale si fija un HEAD anterior como estado vigente sin exigir fetch/rebase antes de reanudar | test `TestSmokeOPESExternalWorkAgentRealUsaShutdownDelegadoConRuntimeDirV0`, `TestHandoffCierreSesionNoReabreBUG077V0`; `go test -count=1 ./cmd/orquesta-server -run 'Test(HandoffCierreSesionNoReabreBUG077|SmokeOPESExternalWorkAgentRealUsaShutdownDelegadoConRuntimeDir)V0'`; `go test -count=1 ./cmd/orquesta-server -run 'Test(SmokesAisladosPasanRuntimeDirAlShutdownComun|SmokeOPESExternalWorkAgentRealUsaShutdownDelegadoConRuntimeDir|ScriptsQueArrancanServidorTemporalUsanShutdownComun|ScriptsQueDeleganArranqueServidorTemporalInstalanCleanup|ScriptsQueUsanShutdownComunPasanRuntimeDir|ScriptsConShutdownDirectoPidenCleanupGoalBackends)V0'` | Cierre: `scripts/smoke_opes_external_work_agent_real.sh` queda fijado a `trap smoke_cleanup EXIT`, `RUNTIME_DIR` y helper `scripts/lib/opes_agent_smoke_ops.sh`; el helper pasa `RUNTIME_DIR` a `smoke_shutdown_orquesta_server`, exporta `ORQUESTA_CODEX_RUNTIME_WORKDIR`, arranca en loopback y captura `server_pid`. Nuevo avance: `docs/handoff_cierre_sesion_orquesta_2026-07-02.md` ya no presenta `777e027c` como HEAD vigente reutilizable; conserva la orden explicita de `git fetch origin trabajo/plataforma-agentes && git rebase origin/trabajo/plataforma-agentes` y `git status --short --branch` antes de reanudar |
| BUG-ORQ-20260701-078 | cerrado | Goal-first/reconciliacion tras QA pass | en ola OPES 12b tema 002 con Orquesta `433129d`, el agente materializo texto limpio, HTML, RAG, tests e informes `pass` (`informe_wave12b_qa_publica.json`), pero no escribio `trabajo/docs/orquesta_goal_result_v0.json` ni `trabajo/opes_topic_rework_delivery.json`; `autoprogramming/status` quedo con cola vacia, `active_runs=0`, `overall_percentage=100`, `recommended_action=observe_goal`, y `POST /api/v0/autoprogramming/goal/observe` devolvio timeout parcial | falta reconciliacion determinista entre artefactos publicables validados, ausencia de recibo terminal y estado del goal; el operador no puede saber si esperar, cerrar por evidencia local, replanificar solo el receipt o cortar el backend sin perder trabajo | `TAREA_OPES_ORQUESTA_GRUPO_B_GOAL_FIRST_OBSERVE_Y_QA_MINIMOS_2026-07-01.md` apartado 24; evidencia en `OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/02_temas/tema_002/validacion/informe_wave12b_qa_publica.json` y `00_control/orquesta_responses/wave12b_t002_rework_qa/observe_goal_after_artifacts.json`; cierre relacionado `BUG-ORQ-20260702-105`; incidencia `docs/incidencias/incidencia_orquesta_goal_first_repair_receipt_materializado_2026-07-02.md`; tests `TestStackGoalMaterializedRefsSourceV0DetectaReceiptTerminalAusenteTrasQAPass`, `TestStackGoalMaterializedRefsSourceV0ReparaReceiptTerminalConPuertosV0`, `TestCodexStackObserveAppDirectorGoalExecutorV0TimeoutSnapshotReparaReceiptMaterializadoV0`, `TestCodexStackObserveAppDirectorGoalExecutorV0RepairReceiptRequiereReworkSinReceiptDominioV0`; `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'Test(StackGoalMaterializedRefsSourceV0|CodexStackObserveAppDirectorGoalExecutorV0)'` | Cierre agregado: el stack escanea solo el `write_set`, requiere QA pass estructurado y en OPES la terna estricta; si hay artefactos + QA pass sin receipt terminal publica `missing_terminal_receipt_after_artifacts_pass`, `expected_terminal_receipt_refs` y accion `repair_receipt`. Con `GoalStateStore` + `GoalClosureValidator` puede materializar un `GoalWorkResultV0` terminal; si la validacion no acepta por receipt/evidencia/tests, persiste `repair_receipt_requires_rework` y no declara cierre por filesystem |
| BUG-ORQ-20260701-079 | abierto | Goal-first/sin checkpoint temprano y salidas gigantes | en ola OPES 13 tema 003 con Orquesta `0af9e345`, el goal siguio `active` tras ~249k tokens y 309 s sin escribir ningun fichero en el write-set del tema; el rollout mostro lecturas y comandos con salidas enormes (`lynx -dump`, `rg` de assets con miles de rutas, salida truncada de ~76k tokens) antes de materializar checkpoint. `autoprogramming/status` seguia en cola vacia, `overall_percentage=100`, `goal_status=running`, `recommended_action=observe_goal`, sin avisar de alto consumo sin artefactos. Shutdown normal bloqueo correctamente por active goal; shutdown forzado mato procesos, pero `goals_1.sqlite` quedo con el goal en `active` | falta limite duro de salida de herramientas internas antes de que el proveedor genere stdout enorme; el resto de politica residente de checkpoint temprano, deteccion de progreso por ficheros y corte/replan ante alto consumo ya esta reducido por avances posteriores. Tambien queda smoke largo que demuestre el comportamiento con proveedor real durante una sesion extensa | `TAREA_OPES_ORQUESTA_GRUPO_B_GOAL_FIRST_OBSERVE_Y_QA_MINIMOS_2026-07-01.md` apartado 25; evidencia en `OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_responses/wave13_t003_rework_qa/status_180s.json`, `observe_goal_1.json`, `shutdown_normal_after_bug079.json`, `shutdown_forced_after_bug079.json` y rollout `rollout-2026-07-01T18-04-02-019f1e6c-4dbe-7a02-ad2e-4e2d12d1eb70.jsonl`; cierres relacionados `BUG-ORQ-20260702-112` y tests `TestMCPRunControlExecutorV0StopForcedReconcilesGoalHighConsumptionSinCheckpoint`, `TestMCPRunControlExecutorV0StopForcedReconcilesGoalBackendMissingAfterExternalCleanup`, `TestMCPAutoprogrammingStatusExecutorV0SinCheckpointConsumoMedioAvisaAntesDeUmbralAlto`, `TestMCPAutoprogrammingStatusExecutorV0CheckpointOnlyBajoConsumoPideSiguienteArtefacto`, `TestMCPAutoprogrammingStatusExecutorV0GoalRunningSinBackendActivoPideReconciliarCleanupExternoV0`, `TestMCPAutoprogrammingStatusExecutorV0GoalRunningStaleSinProcesoPideReconciliarCleanupExternoV0`, `TestRunSupervisorGoalFirstResidentReconciliaBackendMissingTrasCleanupExternoV0`, `TestBuildCodexGoalStartPacketV0IncluyeContratoDeDireccion`; `go test -count=1 ./modulos/orquesta-runtime-codex-goal` | Avance: `autoprogramming/status` ya emite `goal_active_no_checkpoint_high_consumption` con `replan_narrow_context`, y `runs/control stop forced=true` reconcilia un backend activo de alto consumo sin checkpoint a `GoalWorkState` `blocked` replanificable cuando el control confirma que el backend dejo de estar vivo; antes del umbral alto, si el goal activo supera media ventana de consumo sin checkpoint/artefactos/receipt, publica `goal_active_no_checkpoint_consumption_warning` con accion `observe_goal_backend_require_checkpoint`. Nuevo avance: si queda `GoalWorkState` running con `external_goal_ref` pero la observacion gobernada ya no muestra backend Goal activo, o si el `Goal.Status=running` viene con liveness `running_stale_no_process` segura, `autoprogramming/status` publica `goal_backend_missing_after_external_cleanup`, accion/safe_action `run_control_reconcile_external_cleanup`, y `runs/control` persiste ese estado como `blocked` replanificable sin degradar a `control_not_propagated_to_goal_backend`; `runs.supervisor` en `resident_mode` reconcilia ese cleanup externo y lanza rework causal idempotente. Nuevo avance: el contrato Codex Goal pide materializar checkpoint temprano, el app-server no puede relajar el `max_text_bytes` ni sustituir los hints acotados canonicos en `turn/start`, y ahora materializa un checkpoint runtime dentro del `write_set` antes de `turn/start`. Nuevo avance 2026-07-04 noche 28: un checkpoint runtime inicial ya no se confunde en status con falta de checkpoint; `active_checkpoint_only_yet` y `active_timeout_checkpoint_recent` piden `observe_goal_backend_require_next_artifact`, y el materializer no lo cuenta como artefacto parcial. Pendiente residual: limites duros de stdout de herramientas internas del app-server/proveedor; Orquesta ya limita `thread/read`/sanitiza ingesta posterior, pero el schema local de `turn/start` no expone un cap pre-tool compatible |
| BUG-ORQ-20260701-079 | abierto | Goal-first/sin checkpoint temprano y salidas gigantes | actualizacion 2026-07-09: el residual indicaba que `turn/start` no exponia un cap estructurado compatible para salida de herramientas; el smoke real adversarial posterior ya prueba transporte `toolOutputPolicy=accepted`, probe ejecutado dentro de write-set y forced-stop gobernado sin procesos vivos | el contrato preventivo no debe depender solo del texto del prompt: si el app-server soporta schema propio, Orquesta debe transportar la politica como dato; si no lo soporta, no debe romper ejecuciones legacy. La capa Orquesta ya bloquea/sanea ingesta posterior; queda demostrar o corregir el cap duro pre-tool cuando una herramienta vuelca stdout crudo dentro del proveedor antes de que Orquesta pueda observar `thread/read` | tests `TestServerCodexAppServerTurnStartParamsV0SerializaToolOutputPolicyV0`, `TestServerCodexAppServerGoalBackendV0TurnStartInyectaContratoSalidaCompactaV0`, `TestServerCodexAppServerGoalBackendV0TurnStartNoRelajaContratoSalidaCompactaV0`, `TestServerCodexAppServerGoalBackendV0TurnStartToolOutputPolicyFallbackCompatibleV0`, `TestServerCodexAppServerGoalBackendV0PolicyAceptadaYThreadReadGiganteBloqueaV0`, `TestServerCodexAppServerGoalBackendV0ToolOutputPolicyYThreadReadGigantePorWebSocketDeterministaV0`; smoke real `ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_REAL_CONFIRM=1 ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_CODEX_EXECUTION_CONFIRMED=1 ORQUESTA_KEEP_SMOKE_DIR=1 ./scripts/smoke_goal_first_tool_output_policy_adversarial_real.sh`; evidencia `/tmp/orquesta-goal-first-app-server.ZHXRWf`; `go test -count=1 ./modulos/orquesta-runtime-codex-appserver` | Avance: `turn/start` serializa `toolOutputPolicy` con `maxTextBytes`, `threadReadMaxBytes`, `requireBoundedCommands`, `boundedCommandHints` y `durableEvidenceRequired`, derivado de `DirectionContract.ToolOutputPolicy` y clampado a los defaults canonicos. Si el app-server devuelve `invalid_params`/`invalid_request` por ese campo, el backend reintenta sin JSON estructurado, conserva el contrato textual y anota evidencia de fallback. Si tras policy aceptada `thread/read` devuelve una respuesta gigante, Orquesta bloquea el goal con `codex_app_server_thread_read_response_too_large` y evidencia especifica. Avance determinista 2026-07-09: un servidor WebSocket Unix falso valida el JSON real de `turn/start` con `toolOutputPolicy` y despues inyecta un frame gigante de `thread/read`, sin agente/LLM; Orquesta bloquea con `codex_app_server_thread_read_response_too_large`. Smoke real 2026-07-09: `tool_output_policy_transport=accepted`, `bug079_probe_result=executed`, `smoke_goal_first_tool_output_policy_adversarial_real=ok`, `run_control_status=stopped`, `goal_status_after=blocked`, sin salida cruda `X{100,}` observada. Pendiente residual: cap duro de stdout crudo sin redireccion en frontera proveedor/runtime |
| BUG-ORQ-20260709-200 | cerrado local | Goal-first/harness adversarial BUG-079 | el smoke adversarial de `BUG-079` quedaba `running/observe_later` tras checkpoint temprano sin ejecutar el probe stdout gigante, de modo que no demostraba enforcement pre-tool ni daba causa determinista de fallo | el arnes mezclaba dos contratos: transporte de `toolOutputPolicy` y decision libre del agente de ejecutar un probe; si el agente solo escribia checkpoint, el resultado quedaba inconcluso. Ademas una primera ejecucion real revelo que el probe se habia pedido fuera del write-set autorizado | olas Orquesta `codex-core-bug200-harness-20260709T141956Z` y `codex-core-bug079-protocol-20260709T141956Z`; cambios en `scripts/smoke_goal_first_app_server_real.sh`, `cmd/orquesta-server/smoke_goal_first_scripts_v0_test.go` y `modulos/orquesta-runtime-codex-appserver/codex_goal_app_server_migrated_v0_test.go`; tests `SMOKE_GOAL_FIRST_BUG079_GUARD_SELFTEST=1 scripts/smoke_goal_first_app_server_real.sh`; `bash -n scripts/smoke_goal_first_app_server_real.sh scripts/smoke_goal_first_tool_output_policy_adversarial_real.sh`; `go test -count=1 ./cmd/orquesta-server -run 'TestSmokeGoalFirstToolOutputPolicyAdversarial|TestSmokeGoalFirstAppServerRealExponeToolOutputPolicyTransport'`; `go test -count=1 ./modulos/orquesta-runtime-codex-appserver -run 'TestServerCodexAppServerGoalBackendV0(PolicyAceptadaYThreadReadGiganteBloquea|TurnStartToolOutputPolicy|ObservaThreadReadGigante|LanzaThreadGoalYTurnMigrado)|TestCodexAppServer(WebSocketThreadReadResponseBudget|CommandProtocolThreadReadResponseBudget)'`; smoke real `ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_REAL_CONFIRM=1 ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_CODEX_EXECUTION_CONFIRMED=1 ORQUESTA_KEEP_SMOKE_DIR=1 ./scripts/smoke_goal_first_tool_output_policy_adversarial_real.sh`; `git diff --check` | Cierre local: el modo adversarial exige `probe_result.txt` bajo `generated-apps/smoke-goal-first-bug079-tool-output-policy/bug079-tool-output-policy/` con exit code, `probe_stdout.py`, bytes esperados, sentinel `BUG079_STDOUT_PROBE` y `executions=1`; si lo detecta en la ruta legacy falla con `reason=bug079_probe_out_of_write_set`, si falta y hay checkpoint falla con `reason=bug200_probe_not_executed`, si falta resultado valido falla con `reason=no_probe_result`, y ya no declara OK ni inconcluso solo por checkpoint. El self-test ejecuta esos guards sin servidor/Codex y el smoke real 2026-07-09 pasa con forced-stop gobernado. Residual trasladado a `BUG-079`: cap duro de stdout crudo sin redireccion en proveedor/runtime |
| BUG-ORQ-20260709-207 | cerrado | Runtime/proveedor forced-stop app-server | durante el forced-stop del smoke real adversarial BUG-079, el control funcional cerro bien pero el log del app-server retuvo `Node.js[...] ResetStdio` / `Assertion failed` | el app-server arrancaba heredando stdin del PTY de tmux; matar/cerrar la sesion podia provocar un crash ruidoso de Node aunque Orquesta dejara el goal `blocked`, `autoprogramming/status` sin running y sin procesos vivos. `< /dev/null` no era viable porque el proveedor salia por EOF con `codex_app_server_tmux_session_exited` | evidencia inicial `/tmp/orquesta-goal-first-app-server.ZHXRWf/runtime/goal-srv/orquesta-goal-ee58290e72150b73.log`; cierre con `go test -count=1 ./modulos/orquesta-runtime-codex-appserver`; smoke real `scripts/smoke_goal_first_forced_stop_backend_real.sh` en modo real opt-in, evidencia `/tmp/orquesta-goal-first-app-server.oj0Aat`, `run_control_status=stopped`, `run_control_goal_status_after=blocked`, `app_server_tmux_processes_alive=0`, `rg -a 'ResetStdio|Assertion failed'` sin coincidencias en `runtime/goal-srv` ni logs | Cierre: `app_server_tmux` crea una FIFO propia `stdin.pipe` para desacoplar stdin del PTY de tmux sin enviar EOF al proveedor, intenta `SIGTERM` cooperativo sobre procesos propios antes de `kill-session`, y limpia socket/FIFO/owner marker al cerrar. No reabre `BUG-079`: el residual de cap duro stdout crudo sigue en frontera proveedor/runtime |
| BUG-ORQ-20260701-079 | cerrado | Goal-first/sin checkpoint temprano y salidas gigantes | actualizacion 2026-07-04 noche 8: el `CodexGoalStartPacketV0` ya llevaba contrato de direccion, pero el runtime app-server podia enviar a `turn/start` solo `packet.Prompt` si ese prompt venia incompleto o legacy | el contrato preventivo debe fijarse en la frontera final con el proveedor, no depender solo del builder upstream del packet | tests `TestServerCodexAppServerGoalBackendV0TurnStartInyectaContratoSalidaCompactaV0`, `TestServerCodexAppServerGoalBackendV0LanzaThreadGoalYTurnMigradoV0`; `go test -count=1 ./modulos/orquesta-runtime-codex-appserver -run 'TestServerCodexAppServerGoalBackendV0(TurnStartInyectaContratoSalidaCompacta|LanzaThreadGoalYTurnMigrado)V0'` | Avance: `turn/start` inyecta contrato runtime con checkpoint temprano dentro del write-set, no pegar salidas largas, `max_text_bytes`, `thread_read_max_bytes=256 KiB`, comandos acotados, evidencia durable y final/ACK compacto. Pendiente: enforcement duro del proveedor/runtime antes de ejecutar herramientas y smoke real largo que pruebe que no hay consumo gigante antes del checkpoint |
| BUG-ORQ-20260701-080 | cerrado | Goal-first/rutas de contexto saneadas usadas como paths | en ola OPES 14 tema 003, el payload llevaba rutas exactas con `/`, pero el goal uso rutas deformadas como `opes-salidas/orquesta_real/grupo_b_informatica_2026-06-04/rework-package_grupo_b_final_local_2026-06-04/...` e `opes-salidas/codex_directo/informatica-grupo_B/...`; los comandos `ls`/`wc` fallaron para fuentes que existian con ruta real. El spec summary proyecta `context_refs` saneadas sustituyendo separadores por guiones y el agente las trato como rutas ejecutables | la normalizacion de refs pierde la ruta original o no distingue `context_ref` opaco de `filesystem_path`; el agente puede descartar fuentes reales, abrir rework falso o crear matriz incorrecta por rutas corrompidas | `TAREA_OPES_ORQUESTA_GRUPO_B_GOAL_FIRST_OBSERVE_Y_QA_MINIMOS_2026-07-01.md` apartado 26; evidencia en `OPES/.../00_control/orquesta_payloads/wave14_t003_phase0/external_work_wave14_topic_003_phase0.json`, `dry_run_t003_phase0.json` y rollout `rollout-2026-07-01T18-15-31-019f1e76-d25a-70d3-8567-38e179007220.jsonl`; cierre relacionado `BUG-ORQ-20260702-103`; tests `TestBuildExternalWorkGoalWorkSpecV0NoExponeRutasLocalesComoContextRefsEjecutables`, `TestCodexStackV0ExternalWorkRunGoalFirstConservaInputFieldsOPES`; `go test -count=1 ./modulos/orquesta-external-work-run ./modulos/orquesta-app-codex-stack -run 'TestBuildExternalWorkGoalWorkSpecV0NoExponeRutasLocalesComoContextRefsEjecutables|TestCodexStackV0ExternalWorkRunGoalFirstConservaInputFieldsOPES'` | Cierre agregado: los paths locales absolutos en `input_fields` se resumen para `context_refs` como `local_path_ref` opaco con `executable_path:false`, `basename_hint` y criterio explicito de no tratarlos como rutas de filesystem; el stack conserva `input_fields` OPES como contrato operativo para que un adaptador autorizado resuelva la ruta original, sin exponer paths locales saneados como refs ejecutables |
| BUG-ORQ-20260701-081 | cerrado | Goal-first/view_image/base64 gigante en rollout | en ola OPES 14 tema 003, el agente uso `view_image` sobre un PNG de 328K y el rollout incorporo un `data:image/png;base64,...` enorme; el tail del rollout supero 120k tokens y el goal paso de 171k tokens sin entregar matriz/plan, aunque la fase era solo documental | faltaban limites y proyeccion segura de outputs multimodales en goal-first; una revision visual podia inyectar binarios/base64 al historial operativo y al contexto, degradando coste, observabilidad y cierre | `docs/incidencias/incidencia_orquesta_goal_first_sanitiza_tool_outputs_multimodales_2026-07-02.md`; tests `TestCodexAppServerThreadReadSanitizaDataURIMultimodalV0`, `TestDecodeCodexAppServerRPCResponseV0SanitizaThreadReadMultimodalV0`, `TestServerCodexAppServerGoalBackendV0SanitizaResultadoDurableConBase64V0`; `go test -count=1 ./cmd/orquesta-server` | Cierre: la decodificacion RPC/WebSocket de `thread/read` compacta `data:*;base64,` en refs con MIME, hash, bytes y dimensiones cuando son legibles, y aplica presupuesto a `itemsView`/textos operativos conservando la ventana del marcador terminal. El trabajo no se veta: queda como evidencia compacta recuperable |
| BUG-ORQ-20260701-082 | cerrado | Autoprogramming/status diagnostics | en remoto aislado `4383d4e0`, `/api/v0/server/readiness` devolvia HTTP 503 con `codex_goal_backend_degraded` y mensaje `codex_app_server_auth_missing`, pero `POST /api/v0/autoprogramming/status` seguia publicando solo bloqueo goal-first opaco (`goal_backend_state_unreconciled`/`goal_first_blocked`) sin la causa efectiva del backend | readiness y status consumian fuentes de diagnostico distintas; el operador podia ver la causa real en readiness pero no en la superficie usada para decidir autoprogramacion, creando replan inutil y falsa investigacion sobre goals | remoto `127.0.0.1:18787`: readiness 503 con `codex_app_server_auth_missing` y status sin ese diagnostico; tests `TestMCPAutoprogrammingStatusExecutorV0PublicaDiagnosticosConfigurados`, `TestAutoprogrammingStatusAPIRouteV0PublicaDiagnosticosDeComposicion`, `TestMCPTransportV0AutoprogrammingStatusPublicaDiagnosticosConfiguradosDesdeBindings`, `TestBuildStackV0PropagaDiagnosticosAutoprogramacionABindingsV0`, `TestServerAutoprogrammingStatusDiagnosticsFromEffectiveConfigV0PublicaBackendGoalV0` | Cierre: los diagnosticos efectivos del backend Goal se convierten en diagnosticos publicos de autoprogramacion y viajan por `cmd/orquesta-server -> orquesta-app-codex-stack -> MCP bindings -> app-gateway -> MCPAutoprogrammingStatusToolExecutorV0`; REST y MCP nativo publican `codex_goal_backend_degraded` con scope/evidencia, aunque la cola/stats no aporten una causa mejor |
| BUG-ORQ-20260701-083 | cerrado | Tooling/codebase-memory-mcp | durante la sesion local aparecieron dos procesos `/home/alberto/.local/bin/codebase-memory-mcp` vivos durante ~9 min consumiendo ~35% y ~29% CPU sin consulta activa de Orquesta; se cerraron con `kill` cooperativo | la herramienta MCP de contexto podia quedar viva fuera del broker/lease de Orquesta, por lo que los subagentes o sesiones manuales aun podian producir consumo sostenido aunque el broker central tuviera guardas | `docs/incidencias/incidencia_orquesta_codebase_memory_mcp_huerfanos_2026-07-02.md`; tests `scripts/test_bootstrap_agent_tooling.sh`, `TestServerCodeContextToolProcessGuardV0DetectaHuerfanoSinPararPorDefecto`, `TestServerCodeContextToolProcessGuardV0ParaHuerfanoSoloConOptInYEdadMinima`, `TestParseServerPSCodeContextToolProcessesV0`; `go test -count=1 ./cmd/orquesta-server -run 'TestServerCodeContextTool(ProcessGuard|Watchdog)|TestParseServerPSCodeContextToolProcessesV0'` | Cierre: el guard residente lista owner markers, detecta huerfanos y solo los para con opt-in/edad minima; el bootstrap `--status` expone `next_action=stop_orphan_codebase_memory_mcp_processes` si hay procesos vivos y la norma persistente indica parar cooperativamente huerfanos o usar el watchdog configurado, no dejar MCPs sueltos en sesiones |
| BUG-ORQ-20260701-079 | cerrado | Goal-first/sin checkpoint temprano y salidas gigantes | actualizacion 2026-07-03: `ObserveCodexGoalV0` ya sanea salidas gigantes de `thread/read` y conserva `evidence-ref-codex-app-server-thread-output-sanitized`, pero `autoprogramming/status` no proyectaba una accion publica especifica para el operador | la evidencia quedaba opaca: el operador podia ver un goal vivo con evidencia saneada sin recomendacion explicita de acotar contexto/salida antes de continuar o relanzar | test `TestMCPAutoprogrammingStatusExecutorV0OutputGiganteSaneadoPideContextoAcotadoV0`; `go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPAutoprogrammingStatusExecutorV0OutputGiganteSaneadoPideContextoAcotadoV0'` | Avance: `autoprogramming/status` convierte `evidence-ref-codex-app-server-thread-output-sanitized` en accion `codex_app_server_thread_output_sanitized`, severidad `warning`, evidencia propia y `recommended_action=replan_narrow_context`, conservando refs del goal observado. Pendiente: enforcement runtime real de checkpoint temprano y limites de salida de herramientas antes de que el agente queme contexto |
| BUG-ORQ-20260701-079 | cerrado | Goal-first/sin checkpoint temprano y salidas gigantes | actualizacion 2026-07-03: tras publicar `codex_app_server_thread_output_sanitized` en `autoprogramming/status`, faltaba cobertura para que `queue/global-status` no degradase la accion al cruzar vistas | las vistas globales pueden perder acciones goal-first especificas si solo se valida la superficie especializada; el operador necesita ver `replan_narrow_context` tambien desde estado global | test `TestMCPQueueGlobalStatusHTTPHandlerV0ConservaOutputSaneadoGoalFirst`; `go test -count=1 ./modulos/orquesta-mcp -run 'Test(MCPAutoprogrammingStatusExecutorV0OutputGiganteSaneadoPideContextoAcotado|MCPQueueGlobalStatusHTTPHandlerV0ConservaOutputSaneadoGoalFirst)V0'` | Avance: `queue/global-status` conserva el item `codex_app_server_thread_output_sanitized`, sus evidencias y `recommended_action=replan_narrow_context` cuando recibe la accion desde `autoprogramming/status`. Pendiente: enforcement runtime real de checkpoint temprano y limites de salida de herramientas |
| BUG-ORQ-20260701-079 | cerrado | Goal-first/sin checkpoint temprano y salidas gigantes | actualizacion 2026-07-03: `observe_goal` podia seguir recomendando `observe_later` para un goal running aunque el resultado ya trajera `evidence-ref-codex-app-server-thread-output-sanitized` | el operador que observaba el goal directo podia perder la misma senal que ya veia en `autoprogramming/status` y `queue/global-status`, retrasando el replan de contexto acotado | test `TestNewMCPObserveAppDirectorGoalResultV0OutputSaneadoPideContextoAcotado`; `go test -count=1 ./modulos/orquesta-mcp -run 'TestNewMCPObserveAppDirectorGoalResultV0OutputSaneadoPideContextoAcotado'` | Avance: `observe_goal` prioriza la evidencia estructurada de salida saneada y recomienda `replan_narrow_context` incluso si el goal sigue `running`, conservando la evidencia compacta. Pendiente: enforcement runtime real de checkpoint temprano y limites de salida de herramientas antes de que el agente queme contexto |
| BUG-ORQ-20260701-079 | cerrado | Goal-first/sin checkpoint temprano y salidas gigantes | actualizacion 2026-07-03: `autoprogramming/status` publicaba `stale_running[].recommended_action=replan_narrow_context` para `codex_app_server_thread_output_sanitized`, pero `efficiency_summary` podia publicar `repair_receipt:run:<run_ref>` aunque no hubiese diagnostico de receipt, o quedar sin razon especifica de salida saneada | los paneles compactos que consumen solo el resumen podian perder la indicacion de replan con contexto acotado y sugerir una reparacion de receipt no causal | test `TestMCPAutoprogrammingStatusExecutorV0OutputGiganteSaneadoPideContextoAcotadoV0`; `go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPAutoprogrammingStatusExecutorV0OutputGiganteSaneadoPideContextoAcotadoV0'` | Avance: `efficiency_summary` solo deriva `repair_receipt` si existe diagnostico causal de receipt, y para salida saneada deriva `state=attention_required`, razon `codex_app_server_thread_output_sanitized` y `recommended_action=replan_narrow_context:run:<run_ref>`. Pendiente: enforcement runtime real de checkpoint temprano y limites de salida de herramientas |
| BUG-ORQ-20260701-085 | historico/supersedido | Goal-first/write-set y recibo de dominio | en ola OPES 14 tema 003, el payload era fase 0 estrecha, con `allowed_write_set` limitado a `coordinacion_wave14` y `trabajo/docs`, y criterios explicitos de no redactar tema ni generar HTML/RAG/tests. El agente escribio tambien `trabajo/tema_003_ampliado_limpio.md`, `trabajo/tema_003_resumen_limpio.md`, `html/`, `tests/`, `tutor_rag/` y `validacion/`; el `orquesta_goal_result_v0.json` declaro solo fase 0 completa y no incluyo esos artefactos extra ni su QA | el runtime no esta aplicando write-set como frontera causal o no convierte la infraccion en bloqueo/replan; ademas el recibo terminal puede ocultar artefactos fuera de scope, dejando trabajo util no trazado y costes no explicados | `TAREA_OPES_ORQUESTA_GRUPO_B_GOAL_FIRST_OBSERVE_Y_QA_MINIMOS_2026-07-01.md` apartado 28; avances `docs/incidencias/incidencia_orquesta_goal_first_recibo_artifact_paths_2026-07-02.md` y `docs/incidencias/incidencia_orquesta_goal_first_artifact_paths_omitidos_2026-07-02.md`; tests `TestBuildCodexGoalStartPacketV0IncluyeContratoDeDireccion`, `TestValidateGoalWorkClosureV0BloqueaArtifactPathsFueraDeWriteSet`, `TestValidateGoalWorkClosureV0BloqueaMaterializedArtifactsFueraDeWriteSet`, `TestValidateGoalWorkClosureV0ExigeArtifactPathsCuandoLaPoliticaLoPide`, `TestValidateGoalWorkClosureV0BloqueaCompleteConArtefactosParciales`, `TestBuildExternalWorkGoalWorkSpecV0CompilaContratoNeutral`, `TestStackGoalMaterializedRefsSourceV0DetectaArtifactPathsOmitidosEnReceiptTerminal`, `TestMCPDirectorStatsToolExecutorV0GoalFirstProyectaArtifactPathsOmitidosV0`, `TestMCPAutoprogrammingStatusExecutorV0ArtifactPathsOmitidosPideRepairReceiptV0`, `TestEnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0ArtifactPathsOmitidosPideRepairReceipt`, `TestRunSupervisorGoalFirstResidentPreparaReworkPorArtifactPathsOmitidosV0`, `TestRunSupervisorGoalFirstResidentPreparaReworkPorArtefactosFueraDeWriteSetV0`; `go test -count=1 ./modulos/orquesta-goal ./modulos/orquesta-runtime-codex-goal ./modulos/orquesta-external-work-run ./modulos/orquesta-mcp`; `go test -count=1 ./modulos/orquesta-app-codex-stack ./modulos/orquesta-mcp` | Avance: el contrato Codex Goal exige `artifact_paths` en el JSON terminal y pide listar todas las rutas relativas creadas/modificadas/verificadas; si hubo escrituras fuera del `write_set`, debe declararlas y cerrar `blocked` con `summary=out_of_scope_artifacts`. El gate existente de `orquesta-goal` bloquea rutas declaradas fuera de scope. Ademas `GoalClosurePolicyV0.require_artifact_paths` permite que el cierre rechace `complete` con `artifact_paths` vacio y `external-work` goal-first lo activa por defecto; dry-run lo expone como `closure_requires_artifact_paths`. Nuevo avance: materialized refs construye manifest independiente bajo `write_set`, detecta `artifact_paths_omitted_materialized` cuando el recibo terminal omite rutas materializadas y lo publica en stats/status/observe con accion `repair_receipt`; el supervisor residente prepara rework causal para ese estado materializado. Avance 2026-07-02: todo `external-work` goal-first exige ahora `materialized_artifacts`, `checklist` y `rework_plan_refs` para parciales; el wrapper de domain receipt hidrata `materialized_artifacts` validos y checklist desde el `file_ref` aceptado en ledger, de forma que un cierre de dominio aceptado tiene manifest/checklist auditable sin depender solo del resumen terminal. Nuevo avance: `orquesta-goal` bloquea `materialized_artifacts.path` fuera del write-set con issue de campo especifico, no solo como `artifact_paths`, para que el rework causal distinga manifest materializado fuera de scope. Nuevo avance: si materialized refs detecta `out_of_scope_materialized_artifacts`, `runs.supervisor` en `resident_mode` lanza rework causal idempotente igual que para `artifact_paths_omitted_materialized`, conservando evidencias de artefactos recuperables fuera del write-set. Estado vigente 2026-07-04: supersedido por BUG-ORQ-20260704-164; no cuenta como BUG-085 abierto. Queda solo como limite preventivo proveedor/FS y ausencia de smoke OPES/productivo |
| BUG-ORQ-20260701-086 | cerrado | Startup/cleanup de backend Goal | al arrancar wave15 tema 004 con Orquesta `cf492f2b`, el servidor salio con `startup_dirty_runs_detected: runs transitorios activos=0 cola_desincronizada=8; usar ORQUESTA_STARTUP_CLEANUP_MODE=forced_stop`, pero antes de salir habia arrancado `orquesta-goal-901a77e254b3363c` y dejo vivos tmux, node y binario Codex app-server. OPES tuvo que matar la sesion tmux manualmente | el preflight de dirty runs se ejecuta despues de inicializar backend Goal, o el fallo de startup no invoca cleanup cooperativo del backend creado durante bootstrap. El operador recibe una accion de limpieza logica, pero queda consumo real vivo fuera del servidor | `TAREA_OPES_ORQUESTA_GRUPO_B_GOAL_FIRST_OBSERVE_Y_QA_MINIMOS_2026-07-01.md` apartado 29; evidencia: runtime `.orquesta-runtime/grupo-b-info-wave15-t004-rework-20260701T183900`, salida `startup_dirty_runs_detected`, tmux `orquesta-goal-901a77e254b3363c` y PIDs `3221515`/`3221529` antes del kill; test `TestRuntimeV0RunEjecutaShutdownHooksSiStartupBloqueaV0`; `go test -count=1 ./modulos/orquesta-server -run 'TestRuntimeV0RunEjecutaShutdownHooksSiStartupBloqueaV0|TestRuntimeV0PrepareStartup|TestRuntimeV0CompactaShutdownHooksV0'` | Cierre: si `prepareStartupV0` devuelve error o estado no listo, `RunWithShutdownCauseV0` ejecuta los `RuntimeShutdownHookPortV0` antes de salir. Asi cualquier backend Goal/app-server tmux registrado como hook se limpia tambien cuando el startup check aborta antes de abrir HTTP |
| BUG-ORQ-20260701-087 | cerrado | Remoto/Codex Home fallback | en el remoto aislado `cf492f2b`, `CODEX_HOME=/srv/orquesta-self/codex-home` contenia `auth.json` valido con `auth_mode` y `tokens`, pero readiness seguia degradando `app_goal` e `idle_goal` como `codex_app_server_auth_missing`; el proceso no tenia `ORQUESTA_CODEX_CODE_HOME` | el resolver de `CodeHomeDir` solo miraba `ORQUESTA_CODEX_CODE_HOME` y despues `HOME/.codex`; en entornos donde Codex real usa `CODEX_HOME` directo, Orquesta buscaba credenciales en una ruta derivada inexistente y producia falso `auth_missing` | remoto `127.0.0.1:18787` con `HOME`/`CODEX_HOME` apuntando al home aislado y sin `ORQUESTA_CODEX_CODE_HOME`; tests `TestCodeHomeDirV0UsaCODEXHOMEAntesDeHomeCodexV0`, `TestCodexAppServerTmuxBackendV0PreparaCodeHomeDesdeCODEXHOMEResueltoV0`; `go test -count=1 ./cmd/orquesta-server -run 'Test(CodeHomeDirV0UsaCODEXHOMEAntesDeHomeCodexV0|CodexAppServerTmuxBackendV0PreparaCodeHomeDesdeCODEXHOMEResueltoV0|ServerCodexGoalBackendFromEnvV0TmuxSinAuthDegradaYConservaShutdownV0|CodexAppServerAuthIssueCodeV0)'` | Cierre: `ORQUESTA_CODEX_CODE_HOME` sigue siendo la fuente explicita, pero si no esta definida Orquesta usa `CODEX_HOME` antes de caer a `HOME/.codex`; `app_server_tmux` puede proyectar `auth.json` y `config.toml` desde el home Codex real sin imprimir secretos ni tocar produccion |
| BUG-ORQ-20260701-074 | cerrado | Codebase broker/remoto sin rg | al probar Orquesta remota aislada `d40ad337`, `POST /api/v0/codebase/status` devolvio ok, pero `POST /api/v0/codebase/query` para `goal_active_timeout_backend_active` devolvio `code_context_proveedor_error`; el host remoto no tenia `rg`, por lo que el broker central de contexto quedaba inutil para agentes aunque no arrancase `codebase-memory-mcp` | el fallback de contexto central estaba acoplado al binario externo `rg`; en servidores minimos, los agentes no podian usar Orquesta como broker y podian volver a consultas directas o indexadores propios | smoke remoto `curl /api/v0/codebase/query` en `127.0.0.1:18787` antes del fix; `command -v rg` => `rg_missing`; tests `TestServerRGCodeContextProviderV0UsaFallbackGoSiRGAusente` y smoke remoto posterior | Cierre: `serverRGCodeContextProviderV0` mantiene `rg` cuando existe, pero si el comando no esta disponible usa un buscador interno Go con scopes saneados, limite de resultados/bytes, skips de runtime/cache y evidencia `evidence-ref-code-context-go-fallback-v0`. Asi Orquesta puede seguir siendo broker central de contexto sin instalar paquetes ni arrancar MCPs por agente |
| BUG-ORQ-20260701-059 | cerrado | Goal-first/lifecycle | aparecieron `trabajo/docs/orquesta_goal_result_v0.json` en temas OPES 006, 012 y 018, pero `goals_1.sqlite` solo mostraba `complete` para tema 012; temas 006 y 018 seguian `active` pese a tener resultado durable. En la ola P0-clean posterior, seis runs quedaron `stopped/blocked` en cola mientras los seis goals Codex seguian `active` con tokens y `updated_at` creciendo; luego se observo un goal `active` consumiendo tokens tras entrega durable/texto estable, y despues los seis goals pasaron a `complete` en Codex mientras `autoprogramming/status` seguia publicando `blocked/stale_running=6` | el lifecycle goal-first no reconciliaba suficientemente resultado durable, cierre externo, estado de cola y estado terminal del backend Codex; habia doble fuente de verdad entre cola/status/observe-active/goals SQLite, por lo que se podia consumir tokens o mantener stale aunque el backend ya cerro | `TAREA_OPES_ORQUESTA_GRUPO_B_GOAL_FIRST_OBSERVE_Y_QA_MINIMOS_2026-07-01.md` apartados 2.c y 6-9; mitigacion `f556b004e`; cierre residual BUG-059B con `TestMCPAutoprogrammingStatusExecutorV0GoalBloqueadoConBackendCompleteSaleDeStaleRunning` y `TestMCPAutoprogrammingStatusExecutorV0ResultadoDurableTerminalConBackendActivoNoQuedaOpaco`; `go test -count=1 ./modulos/orquesta-mcp` | Cerrado para lifecycle/status: `autoprogramming/status` publica snapshot accionable para bloqueos activos; cuando `director_stats` observa backend goal terminal, mueve el caso a `resolved_runs` con `goal_status=complete`, `recommended_action=reconcile_goal_terminal` y no lo mantiene en `stale_running`/`queue_health.blocked`; si hay resultado durable terminal pero backend sigue activo, expone `goal_terminal_reconcile_pending` en vez de `goal_backend_state_unreconciled` opaco. BUG-061 sobre contrato minimo de `orquesta_goal_result_v0.json` queda fuera de este cierre. |
| BUG-ORQ-20260701-060 | cerrado | Required tests/evidencia | `orquesta_goal_result_v0.json` de temas OPES 012 y 018 contiene `required_test_results` con `status=passed` y `evidence_refs=[]` | el contrato de tests requeridos permitia declarar passed sin evidencia durable o no exigia refs relativas al write-set; esto rompe cierre causal y auditoria posterior | `TAREA_OPES_ORQUESTA_GRUPO_B_GOAL_FIRST_OBSERVE_Y_QA_MINIMOS_2026-07-01.md`; tests `TestGoalDomainReceiptClosureValidatorV0HidrataRequiredTestDominioSinEvidenciaV0`, `TestGoalDomainReceiptClosureValidatorV0BloqueaRequiredTestComandoPassedSinEvidenciaV0` | Cierre: el cierre goal-first/domain receipt no trata `passed` sin `evidence_refs` como evidencia valida. Los required tests de dominio sin comando se hidratan desde receipts durables aceptados del ledger; los tests con comando o sin recibo durable bloquean con `domain_work_required_test_evidence_missing` en `required_test_results.evidence_refs` |
| BUG-ORQ-20260701-061 | cerrado | Goal result/contrato | `trabajo/docs/orquesta_goal_result_v0.json` del tema OPES 011 contenia resumen, artefactos y `required_test_results`, pero `schema_version`, `status` y `estado` estaban ausentes/null | el resultado durable goal-first podia materializarse como handoff textual sin contrato minimo de esquema/terminalidad; OPES no podia distinguir resultado terminal, preliminar, invalido o borrador | `TAREA_OPES_ORQUESTA_GRUPO_B_GOAL_FIRST_OBSERVE_Y_QA_MINIMOS_2026-07-01.md` apartado 8 | Cierre: `orquesta_goal_result_v0.json` y marcador final exigen `schema_version` y `status`/`estado` terminal (`complete`, `blocked` o `invalid`); si falta o es no terminal, observe devuelve issue accionable y no promueve goal activo a `complete`. Evidencia: `TestServerCodexAppServerGoalBackendV0NoPromueveResultadoDurableActivoSinContratoMinimoV0`, `TestCodexAppServerGoalResultFromWorkspaceV0RechazaEstadoNoTerminalV0`, `go test -count=1 ./cmd/orquesta-server`, `go test -count=1 ./modulos/orquesta-runtime-codex-goal` |
| BUG-ORQ-20260701-062 | cerrado | Goal-first/progreso activo | en la ola 2 OPES, seis goals Codex seguian `active` con tokens y `updated_at` recientes, pero `autoprogramming/status` paso a `blocked=6`/`stale_running=6` en pocos minutos y no habia checkpoints ni ficheros nuevos en los write-set | la deteccion de stale mezclaba ausencia inicial de artefactos con perdida real de progreso; no habia ventana `active_no_checkpoint_yet` antes de declarar stale fuerte | `TAREA_OPES_ORQUESTA_GRUPO_B_GOAL_FIRST_OBSERVE_Y_QA_MINIMOS_2026-07-01.md` apartado 10 | Cierre: si `director.stats` observa backend goal `active`/`running` con timestamp de actividad/artefacto o senal viva de runtime, `autoprogramming/status` publica accion `active_no_checkpoint_yet` de severidad `info` con `goal_ref`, `external_goal_ref`, `goal_status`, `tokens_used`, `last_output_at`/`last_artifact_at` si existen, y recomienda `observe_goal_backend_wait_for_checkpoint` en vez de `goal_first_blocked` fuerte. `tokens_used` solo enriquece snapshot: sin timestamp/live signal conserva bloqueo fuerte. Evidencia: `TestMCPAutoprogrammingStatusExecutorV0GoalBloqueadoConBackendActivoPublicaSnapshotAccionable`, `TestMCPAutoprogrammingStatusExecutorV0BackendActivoConSoloTokensHistoricosSigueBloqueado`, `TestMCPAutoprogrammingStatusExecutorV0ResultadoDurableTerminalConBackendActivoNoQuedaOpaco`, `go test -count=1 ./modulos/orquesta-mcp` |
| BUG-ORQ-20260701-063 | cerrado | Run control/Goal backend | `POST /api/v0/runs/control` con `action=stop` y `forced=true` devolvio `504` o timeout y despues la API marco seis runs `terminal/stopped`, pero `goals_1.sqlite` seguia con los seis goals `active` | el control de runs no propagaba de forma verificable stop/cancel al backend goal-first o podia publicar `stopped` sin confirmacion causal del goal/thread Codex | `TAREA_OPES_ORQUESTA_GRUPO_B_GOAL_FIRST_OBSERVE_Y_QA_MINIMOS_2026-07-01.md` apartado 11; cierre con `TestMCPRunControlExecutorV0StopForcedNoPublicaStoppedSiGoalBackendSigueActive`, `TestMCPRunControlExecutorV0StopForcedPermiteTerminalSiGoalBackendYaComplete`, `go test -count=1 ./modulos/orquesta-mcp` y `go test -count=1 ./modulos/orquesta-app-codex-stack` | `runs/control stop/cancel` observa `director.stats` antes/despues cuando el stack goal-first esta cableado. Si el backend sigue `active`, devuelve `control_not_propagated_to_goal_backend`, expone `run_ref`, `goal_ref`/`external_goal_ref`, estado previo/final, `goal_status_before/after`, senal no confirmada y accion segura, y fuerza la proyeccion publica a `stop_requested`/`cancel_requested` en vez de `stopped`. Si el backend ya esta terminal, permite publicar terminal confirmado. |

| BUG-ORQ-20260701-084 | cerrado | Remoto/Codex Goal auth | en el remoto aislado `5eee7283`, `autoprogramming/status` mostraba 8 goals `blocked` con `codex_app_server_thread_system_error`, pero el log de `orquesta-goal-4e18317da2b9ccb9` repetia `HTTP error: 401 Unauthorized` contra `wss://api.openai.com/v1/responses`; el `codex-home` aislado no contenia `auth.json` | readiness y observacion del backend Goal validaban socket/thread pero no proyectaban errores de autenticacion del proveedor; Orquesta podia seguir lanzando/observando goals contra un app-server vivo pero incapaz de ejecutar, con causa opaca y replan inutil | remoto `/srv/orquesta-self/runtime/server-latest/codex-runtime/goal-srv/orquesta-goal-4e18317da2b9ccb9.log`; `POST /api/v0/autoprogramming/status`; tests `TestServerCodexAppServerGoalBackendV0DiagnosticaSystemErrorConUnauthorizedDelLogV0`, `TestServerCodexAppServerGoalBackendV0DiagnosticaSystemErrorConAuthAusenteV0`, `TestServerCodexGoalBackendFromEnvV0TmuxSinAuthDegradaYConservaShutdownV0`, `TestCodexAppServerAuthIssueCodeV0DetectaAuthAusenteV0`, `TestCodexAppServerIssueCodeFromLogFileV0ClasificaUnauthorizedProviderV0`; `go test -count=1 ./cmd/orquesta-server`; `go test -count=1 ./...`; remoto `d740b2fb` readiness HTTP 503 con `codex_app_server_auth_missing` | Cierre de codigo: `401 Unauthorized` se clasifica como `codex_app_server_provider_unauthorized`; si `thread/read` devuelve `systemError`, el observer consulta el log diagnostico del backend y devuelve causa/evidencia operable; si el `CODEX_HOME` fuente/aislado no tiene `auth.json` y no hay auth por entorno, `app_server_tmux` se degrada antes de arrancar/probar socket, conserva shutdown hook y readiness queda no-ready con `codex_app_server_auth_missing`. Pendiente operativo externo: restaurar autenticacion Codex/OpenAI en el `CODEX_HOME` aislado remoto |
| BUG-ORQ-20260701-085 | historico/supersedido | Goal-first/write-set y recibo de dominio | actualizacion 2026-07-03: la reconciliacion de `artifact_paths_omitted_materialized` comparaba artefactos materializados contra `state.LastResult`, pero podia ignorar el `orquesta_goal_result_v0.json` durable leido del write-set si el estado persistido no traia `LastResult` | el manifest independiente de filesystem debe usar tambien el receipt terminal materializado, no solo el estado ya hidratado, para detectar rutas omitidas aun cuando el state este incompleto o retrasado | test `TestStackGoalMaterializedRefsSourceV0DetectaArtifactPathsOmitidosDesdeReceiptDurable`; `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestStackGoalMaterializedRefsSourceV0Detecta(ArtifactPathsOmitidos(EnReceiptTerminal|DesdeReceiptDurable)|ArtefactosOPESFueraDeWriteSet)'` | Avance: `goalMaterializedArtifactPathsOmittedV0` une `artifact_paths` declarados en `state.LastResult` y en el receipt durable leido durante el scan antes de compararlos con los artefactos encontrados bajo `write_set`; si el durable omite una ruta materializada, publica `artifact_paths_omitted_materialized` y evidencia recuperable aunque `LastResult` sea nil. Estado vigente 2026-07-04: supersedido por BUG-ORQ-20260704-164; no cuenta como BUG-085 abierto. Queda solo como limite preventivo proveedor/FS y ausencia de smoke OPES/productivo |
| BUG-ORQ-20260701-085 | historico/supersedido | Goal-first/write-set y recibo de dominio | actualizacion 2026-07-03: el paquete Codex Goal declaraba `allowed_write_set`, pero el backend app-server podia arrancar el thread con sandbox vacio, `read-only` o `danger-full-access`, dejando el enforcement real dependiente de configuracion externa | el contrato de write-set necesitaba viajar como politica ejecutable para el starter, no solo como instruccion textual y validacion posterior del receipt | tests `TestBuildCodexGoalStartPacketV0IncluyeContratoDeDireccion`, `TestServerCodexAppServerGoalBackendV0WriteSetGuardNoUsaDangerFullAccess`, `TestServerCodexAppServerGoalBackendV0WriteSetGuardExigeAllowedWriteSet`, `TestServerCodexAppServerGoalBackendV0WriteSetGuardBloqueaAllowedWriteSetDivergente`, `TestCodexAppServerGoalSandboxForPacketV0AplicaMinimoConWriteSetGuard`, `TestMCPAutoprogrammingStatusExecutorV0WriteSetGuardContractPideRepairPacketV0`, `TestMCPQueueGlobalStatusHTTPHandlerV0ConservaWriteSetGuardContract`, `TestMCPDomainWorkStatusHTTPHandlerV0ConservaWriteSetGuardContract`, `TestNewMCPObserveAppDirectorGoalPartialResultFromStateV0ConservaLaunchIssueWriteSetGuardContract`; `go test -count=1 ./modulos/orquesta-runtime-codex-goal ./cmd/orquesta-server -run 'Test(BuildCodexGoalStartPacketV0IncluyeContratoDeDireccion|ServerCodexAppServerGoalBackendV0WriteSetGuard(NoUsaDangerFullAccess|ExigeAllowedWriteSet|BloqueaAllowedWriteSetDivergente)|CodexAppServerGoalSandboxForPacketV0AplicaMinimoConWriteSetGuard)'`; `go test -count=1 ./modulos/orquesta-mcp -run 'Test(MCPAutoprogrammingStatusExecutorV0WriteSetGuardContractPideRepairPacket|MCPQueueGlobalStatusHTTPHandlerV0ConservaWriteSetGuardContract|MCPDomainWorkStatusHTTPHandlerV0ConservaWriteSetGuardContract|NewMCPObserveAppDirectorGoalPartialResultFromStateV0ConservaLaunchIssueWriteSetGuardContract)'` | Avance: `CodexGoalDirectionContractV0` publica `write_set_enforcement=workspace_write_guard` y `minimum_sandbox=workspace-write`; el backend app-server consume esa politica y degrada sandbox vacio o `danger-full-access` a `workspace-write`, bloquea `read-only`, y ahora rechaza antes de `thread/start` cualquier packet con `workspace_write_guard` sin `direction_contract.allowed_write_set` o con divergencia frente a `packet.write_set`, conservando evidencia compacta. `autoprogramming/status`, `queue/global-status`, `domain-work/status` y `observe_goal` propagan esos bloqueos como accion `repair_goal_write_set_contract`. Estado vigente 2026-07-04: supersedido por BUG-ORQ-20260704-164 para runtime/write-set; queda solo como limite preventivo proveedor/FS sin smoke OPES/productivo |
| BUG-ORQ-20260701-085 | historico/supersedido | Goal-first/write-set y recibo de dominio | actualizacion 2026-07-03: degradar `read-only` a `workspace-write` podia ocultar una configuracion explicitamente incompatible con goals que deben escribir en el write-set | la politica ejecutable debe reducir accesos peligrosos, pero no elevar permisos cuando el operador eligio modo solo lectura; ese caso debe ser bloqueo publico antes de arrancar thread | tests `TestServerCodexAppServerGoalBackendV0ReadOnlyBloqueaWriteSetGuard`, `TestCodexAppServerGoalSandboxForPacketV0AplicaMinimoConWriteSetGuard`; `go test -count=1 ./cmd/orquesta-server -run 'Test(ServerCodexAppServerGoalBackendV0(ReadOnlyBloqueaWriteSetGuard|WriteSetGuardNoUsaDangerFullAccess)|CodexAppServerGoalSandboxForPacketV0AplicaMinimoConWriteSetGuard)'` | Avance: con `write_set_enforcement=workspace_write_guard`, `danger-full-access` y sandbox vacio se reducen a `workspace-write`, pero `read-only` devuelve `codex_app_server_write_set_requires_workspace_write` y no llama `thread/start`; evita elevar permisos implicitamente y conserva el bloqueo como evidencia operativa. Estado vigente 2026-07-04: supersedido por BUG-ORQ-20260704-164 para runtime/write-set; queda solo como limite preventivo proveedor/FS sin smoke OPES/productivo |
| BUG-ORQ-20260701-085 | historico/supersedido | Goal-first/write-set y recibo de dominio | actualizacion 2026-07-03: cuando habia artefactos y QA pasada pero faltaba receipt terminal, `ExpectedReceiptRefs` seguia anunciando solo `orquesta_goal_result_v0.json` y `opes_topic_rework_delivery.json`, aunque el contrato vigente de Codex Goal materializa un JSON unico por `goal_ref` | el rework/operador podia reparar contra un nombre legacy y dejar sin cubrir el sidecar durable que el prompt actual exige | test `TestStackGoalMaterializedRefsSourceV0DetectaReceiptTerminalAusenteTrasQAPass`; `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestStackGoalMaterializedRefsSourceV0DetectaReceiptTerminalAusenteTrasQAPass'` | Avance: el scanner de materialized refs publica tambien `expected-terminal-receipt:<orquesta_goal_result_<goal_ref>.json>` usando `CodexGoalResultFileNameForGoalRefV0`, junto a los nombres legacy/OPES; el rework de receipt apunta ya al contrato durable vigente. Estado vigente 2026-07-04: supersedido por BUG-ORQ-20260704-164 para runtime/write-set; queda solo como limite preventivo proveedor/FS sin smoke OPES/productivo |
| BUG-ORQ-20260701-085 | historico/supersedido | Goal-first/write-set y recibo de dominio | actualizacion 2026-07-03: el bloqueo `codex_app_server_write_set_requires_workspace_write` devolvia issue code pero no evidencia compacta propia en el start receipt | las superficies que consumen solo refs/evidencias podian perder la causa operacional del bloqueo read-only y requerir inspeccion de logs | test `TestServerCodexAppServerGoalBackendV0ReadOnlyBloqueaWriteSetGuard`; `go test -count=1 ./cmd/orquesta-server -run 'TestServerCodexAppServerGoalBackendV0ReadOnlyBloqueaWriteSetGuard'` | Avance: el receipt invalido por read-only + write-set guard conserva `evidence-ref-codex-app-server-write-set-requires-workspace-write`, haciendo trazable el bloqueo sin logs. Estado vigente 2026-07-04: supersedido por BUG-ORQ-20260704-164 para runtime/write-set; queda solo como limite preventivo proveedor/FS sin smoke OPES/productivo |
| BUG-ORQ-20260701-085 | historico/supersedido | Goal-first/write-set y recibo de dominio | actualizacion 2026-07-03: el issue durable `codex_app_server_write_set_requires_workspace_write` podia llegar a `autoprogramming/status` como bloqueo generico `goal_first_blocked` con accion `review_replan_goal_first` | el operador podia replanificar un goal que en realidad necesitaba corregir configuracion de sandbox a `workspace-write`, perdiendo la causa estructurada del launch receipt | tests `TestMCPAutoprogrammingStatusExecutorV0WriteSetReadOnlyPideConfigurarSandboxV0`, `TestMCPQueueGlobalStatusHTTPHandlerV0ConservaWriteSetRequiresWorkspaceWrite`; `go test -count=1 ./modulos/orquesta-mcp -run 'TestMCP(AutoprogrammingStatusExecutorV0WriteSetReadOnlyPideConfigurarSandbox|QueueGlobalStatusHTTPHandlerV0ConservaWriteSetRequiresWorkspaceWrite)'` | Avance: `autoprogramming/status` especializa el bloqueo read-only + write-set con `code=codex_app_server_write_set_requires_workspace_write`, accion `configure_goal_backend_workspace_write` y evidencias del start receipt; `queue/global-status` conserva esa accion sin degradarla. Estado vigente 2026-07-04: supersedido por BUG-ORQ-20260704-164 para runtime/write-set; queda solo como limite preventivo proveedor/FS sin smoke OPES/productivo |
| BUG-ORQ-20260701-085 | historico/supersedido | Goal-first/write-set y recibo de dominio | actualizacion 2026-07-03: el fallback parcial de `observe_goal` desde `GoalWorkState` podia omitir `LaunchReceipt.Issues` y `LaunchReceipt.EvidenceRefs` cuando el launch ya habia fallado por sandbox `read-only` incompatible con write-set | el operador que observaba el goal tras timeout/error de backend podia ver un estado invalid sin la causa estructurada ni la accion de configurar `workspace-write` | test `TestNewMCPObserveAppDirectorGoalPartialResultFromStateV0ConservaLaunchIssueWriteSetReadOnly`; `go test -count=1 ./modulos/orquesta-mcp -run 'TestNewMCPObserveAppDirectorGoalPartialResultFromStateV0ConservaLaunchIssueWriteSetReadOnly'` | Avance: el resultado parcial de `observe_goal` conserva issues/evidencias del launch receipt y recomienda `configure_goal_backend_workspace_write` para `codex_app_server_write_set_requires_workspace_write`. Estado vigente 2026-07-04: supersedido por BUG-ORQ-20260704-164 para runtime/write-set; queda solo como limite preventivo proveedor/FS sin smoke OPES/productivo |
| BUG-ORQ-20260701-079/085 | cerrado/supersedido | Goal-first/domain-work status | actualizacion 2026-07-03: `/api/v0/domain-work/status` conservaba la accion `replan_narrow_context` o `configure_goal_backend_workspace_write`, pero podia publicar como `status` el codigo tecnico `codex_app_server_thread_output_sanitized` o `codex_app_server_write_set_requires_workspace_write` | la fachada de dominio debe traducir senales goal-first accionables a estado operacional `blocked`, sin perder la accion concreta para el operador OPES/domain-work | tests `TestMCPDomainWorkStatusHTTPHandlerV0ConservaOutputSaneadoGoalFirst`, `TestMCPDomainWorkStatusHTTPHandlerV0ConservaWriteSetRequiresWorkspaceWrite`; `go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPDomainWorkStatusHTTPHandlerV0Conserva(OutputSaneadoGoalFirst|WriteSetRequiresWorkspaceWrite|RepairReceiptGoalFirst)'` | Avance: `domain-work/status` normaliza `codex_app_server_thread_output_sanitized` y `codex_app_server_write_set_requires_workspace_write` como `blocked`, conservando `recommended_action=replan_narrow_context` o `configure_goal_backend_workspace_write` y evidencias compactas. Estado vigente 2026-07-04: queda abierto el enforcement runtime real de salidas gigantes por BUG-079; BUG-085/write-set queda supersedido por BUG-ORQ-20260704-164 |
| BUG-ORQ-20260701-075/085 | cerrado/supersedido | Goal-first/domain-work status | actualizacion 2026-07-03: la misma fachada podia publicar como `status` codigos recuperables de QA/receipt/write-set (`qa_failed_public_text`, `partial_artifacts_written`, `artifact_paths_omitted_materialized`, `out_of_scope_materialized_artifacts`) en vez de estado operacional `blocked` | OPES/domain-work necesita un estado estable para paneles y operadores, conservando la accion concreta (`rework_public_text`, `review_partial_artifacts`, `repair_receipt`, `rework_write_set_violation`) y sin descartar artefactos recuperables | test `TestMCPDomainWorkStatusHTTPHandlerV0NormalizaSenalesGoalFirstRecuperablesComoBloqueadas`; `go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPDomainWorkStatusHTTPHandlerV0(NormalizaSenalesGoalFirstRecuperablesComoBloqueadas|Conserva(OutputSaneadoGoalFirst|WriteSetRequiresWorkspaceWrite|RepairReceiptGoalFirst))'` | Avance: `domain-work/status` normaliza esas senales goal-first recuperables como `blocked`, mantiene `needs_action=true`, accion especifica y evidencias compactas. Estado vigente 2026-07-04: quedan abiertos validadores OPES minimos en todos los work kinds; BUG-085/write-set queda supersedido por BUG-ORQ-20260704-164 |
| BUG-ORQ-20260701-075/085 | cerrado/supersedido | Goal-first/efficiency summary | actualizacion 2026-07-03: `autoprogramming/status` publicaba QA fallida, artefactos parciales y violaciones de receipt/write-set en `stale_running`, pero `efficiency_summary` podia omitir la razon/accion agregada para consumidores compactos | los paneles que consumen solo el resumen podian perder si correspondia rework publico, revisar parciales, reparar receipt o rework por write-set, aunque la lista detallada si lo conservase | tests `TestMCPAutoprogrammingStatusExecutorV0QAFailedPublicTextPideReworkV0`, `TestMCPAutoprogrammingStatusExecutorV0ArtifactPathsOmitidosPideRepairReceiptV0`, `TestMCPAutoprogrammingStatusExecutorV0OutOfScopeMaterializedPideReworkV0`, `TestMCPAutoprogrammingStatusExecutorV0ArtefactosParcialesPideRevisionV0`; `go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPAutoprogrammingStatusExecutorV0(QAFailedPublicTextPideRework|ArtifactPathsOmitidosPideRepairReceipt|OutOfScopeMaterializedPideRework|ArtefactosParcialesPideRevision)V0'` | Avance: `efficiency_summary` deriva `state=attention_required`, razon causal y accion agregada `rework_public_text`, `review_partial_artifacts`, `repair_receipt` o `rework_write_set_violation` por run desde esos diagnosticos recuperables. Estado vigente 2026-07-04: quedan abiertos validadores OPES minimos en todos los work kinds; BUG-085/write-set queda supersedido por BUG-ORQ-20260704-164 |
| BUG-ORQ-20260701-075/085 | cerrado/supersedido | Goal-first/queue global status | actualizacion 2026-07-03: las acciones agregadas de `efficiency_summary` con forma `accion:run:<run_ref>` podian degradarse en `queue/global-status` si pasaban por la normalizacion generica, perdiendo el run causal de rework/receipt/write-set | las superficies compactas deben preservar tanto la accion especifica como su scope causal; normalizar solo acciones exactas dejaba un hueco para volver a `retry`/`repair_goal_state` al cruzar resumenes por run | test `TestMCPQueueGlobalStatusNormalizeRecommendedActionV0PreservaAccionesGoalFirstPorRun`; `go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPQueueGlobalStatusNormalizeRecommendedActionV0PreservaAccionesGoalFirst(PorRun|Especificas)'` | Avance: `queue/global-status` conserva acciones goal-first permitidas con sufijo `:run:<ref>` para replan acotado, timeout activo, sandbox/write-set, receipt, texto publico, artefactos parciales y fase 0. Estado vigente 2026-07-04: quedan abiertos validadores OPES minimos en todos los work kinds; BUG-085/write-set queda supersedido por BUG-ORQ-20260704-164 |
| BUG-ORQ-20260701-075/085 | cerrado/supersedido | Goal-first/ops snapshot | actualizacion 2026-07-03: `director_autonomous_ops_snapshot` podia construir `summary_key` dinamicas como `director.ops.decision.repair_receipt:run:<run_ref>` cuando una accion agregada ya venia acotada por run | las claves publicas/i18n de ops deben ser estables y el scope causal debe viajar en `run_ref` u otro campo de scope, no dentro de la key traducible | test `TestDirectorOpsDecisionFromAutoprogrammingActionableRunV0NormalizaSummaryKeyPorRunV0`; `go test -count=1 ./modulos/orquesta-mcp -run 'TestDirectorOpsDecisionFromAutoprogrammingActionableRunV0(RespetaSeveridad|NormalizaSummaryKeyPorRun)V0'` | Avance: la decision autonoma conserva `RunRef` y evidencias, pero recorta cualquier sufijo de scope `:<tipo>:<ref>` solo para `summary_key`, dejando claves estables para repair/rework/replan por run o futuros scopes. Estado vigente 2026-07-04: quedan abiertos validadores OPES minimos en todos los work kinds; BUG-085/write-set queda supersedido por BUG-ORQ-20260704-164 |
| BUG-ORQ-20260701-066/075 | cerrado | Goal-first/domain-work status | actualizacion 2026-07-03: la cobertura de la fachada domain-work debia incluir tambien `phase0_complete_non_publishable` y `required_test_evidence_missing`, que son bloqueos recuperables de cierre/fase antes de publicar OPES | el operador necesita distinguir continuar desde fase 0 o reparar receipt/evidencia de tests sin que el estado de dominio parezca un codigo tecnico o un cierre ambiguo | test `TestMCPDomainWorkStatusHTTPHandlerV0NormalizaSenalesGoalFirstRecuperablesComoBloqueadas`; `go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPDomainWorkStatusHTTPHandlerV0NormalizaSenalesGoalFirstRecuperablesComoBloqueadas'` | Avance: la prueba de domain-work/status cubre `phase0_complete_non_publishable -> continue_from_phase0_checkpoint` y `required_test_evidence_missing -> repair_receipt` como `blocked` accionable. Siguen pendientes criterios OPES done/settled y validadores por work kind |
| BUG-ORQ-20260701-066/075 | cerrado | Goal-first/efficiency summary | actualizacion 2026-07-03: `autoprogramming/status` publicaba `phase0_complete_non_publishable` y `required_test_evidence_missing` en `stale_running`, pero `efficiency_summary` podia quedar sin razon/accion agregada especifica para paneles compactos | los operadores que consumen solo el resumen podian no ver si correspondia continuar desde checkpoint de fase 0 o reparar receipt/evidencia de tests requerida | tests `TestMCPAutoprogrammingStatusExecutorV0Phase0NoPublicablePideContinuarV0`, `TestMCPAutoprogrammingStatusExecutorV0RequiredTestEvidenceAusentePideRepairReceiptV0`; `go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPAutoprogrammingStatusExecutorV0(Phase0NoPublicablePideContinuar|RequiredTestEvidenceAusentePideRepairReceipt)V0'` | Avance: `efficiency_summary` deriva `state=attention_required`, razon causal y `recommended_action=continue_from_phase0_checkpoint:run:<run_ref>` o `repair_receipt:run:<run_ref>` desde esos diagnosticos recuperables. Siguen pendientes criterios OPES done/settled y validadores por work kind |
| BUG-ORQ-20260701-075 | cerrado | Goal-first/observe | actualizacion 2026-07-03: `observe_goal` podia devolver `observe_later` para un goal `running` aunque el enriquecimiento de artefactos materializados ya hubiese detectado `qa_failed_public_text` recuperable | la prioridad de estado vivo tapaba issues deterministas de QA/artefactos; el operador podia seguir esperando un backend que ya tenia rework publico claro | test `TestEnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0QAFailedPublicTextRunningNoEspera`; `go test -count=1 ./modulos/orquesta-mcp -run 'Test(EnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0(QAFailedPublicTextPideRework|QAFailedPublicTextRunningNoEspera|ArtefactosParcialesPideRevision|ArtifactPathsOmitidosPideRepairReceipt|OutOfScopePideRework|Phase0NoPublicablePideContinuar|RequiredTestEvidenceAusentePideRepairReceipt|NoPisaCierreAceptado)|NewMCPObserveAppDirectorGoalResultV0OutputSaneadoPideContextoAcotado)'` | Avance: `observe_goal` prioriza issues/evidencias estructuradas recuperables antes de `GoalStatus=running|accepted`, manteniendo `observe_later` solo cuando no hay causa accionable mas especifica. La cobertura incluye QA fallida, receipt omitido, write-set fuera de scope, artefactos parciales, fase 0 no publicable y evidencia de tests requerida ausente. Siguen pendientes validadores OPES minimos en todos los work kinds |
| BUG-ORQ-20260701-088 | cerrado funcionalmente | Goal-first/status/shutdown alto consumo | en ola OPES 15 tema 004 con Orquesta `cf492f2b`, el goal quedo `active` con `tokens_used=139028` y solo checkpoint; `status` y `shutdown` expiraron sin cuerpo. En ola OPES 16 con Orquesta `264ed9f3`, startup/status mejoraron y `shutdown` normal devolvio 409 con backend vivo, pero el goal volvio a quedar `active` con `tokens_used=166009` y solo checkpoint; `shutdown forced=true` devolvio `ready` aunque dejo vivos `codex app-server` hasta limpieza manual | faltaban guardas operacionales para `active + tokens crecientes + solo checkpoint`, corte/replan por ausencia de segundo artefacto y parada efectiva del backend Goal; los residuales de proyeccion de artefactos y contrato HTTP quedan cerrados por `BUG-ORQ-20260703-161/162` | `TAREA_OPES_ORQUESTA_GRUPO_B_GOAL_FIRST_OBSERVE_Y_QA_MINIMOS_2026-07-01.md` apartados 30 y 31; avance `docs/incidencias/incidencia_orquesta_goal_first_alto_consumo_activo_2026-07-02.md`; smoke real `docs/runbooks/smoke_goal_first_checkpoint_only_high_consumption_real_2026-07-03.md`; tests `TestServerCodexAppServerGoalBackendV0ExponeAltoConsumoActivoV0`, `TestMCPAutoprogrammingStatusExecutorV0CheckpointOnlyHighConsumptionEsBloqueante`, `TestMCPQueueGlobalStatusHTTPHandlerV0ConservaReplanNarrowContext`, `TestMCPAutoprogrammingStatusExecutorV0SinCheckpointConsumoMedioAvisaAntesDeUmbralAlto`, `TestMCPAutoprogrammingStatusExecutorV0SinCheckpointWarningEstancadoPromueveReplanV0`, `TestMCPAutoprogrammingStatusExecutorV0GoalRunningSinBackendActivoPideReconciliarCleanupExternoV0`, `TestMCPRunControlExecutorV0StopForcedReconcilesGoalBackendMissingAfterExternalCleanup`, `TestMCPRunControlExecutorV0StopForcedReconcilesGoalHighConsumptionCheckpointOnly`, `TestMCPRunControlExecutorV0StopForcedRespetaUmbralConfiguradoV0`, `TestRunSupervisorGoalFirstResidentPreparaReworkPorCheckpointHighConsumptionV0`, `TestRunSupervisorGoalFirstResidentReworkEsIdempotenteV0`, `TestRunSupervisorGoalFirstResidentReconciliaBackendMissingTrasCleanupExternoV0`, `TestRunSupervisorGoalFirstNoResidentNoLanzaReworkV0`, `TestStackGoalMaterializedRefsSourceV0DetectaArtefactosTxtBUG088V0`, `TestCodexStackAutoprogrammingPrepareRunAPIV0GoalReadyLanzaGoalFirstSinColaLegacy`, `TestRuntimeV0ServerShutdownReadyTrasCleanupNoHeredaSnapshotPrevioActivoV0` | Cierre: `thread/goal/get` proyecta alto consumo; status/queue/run-control/run-supervisor llevan `replan_narrow_context` o stop gobernado; el smoke real opt-in produjo `smoke_goal_first_high_consumption_real=ok`, `bug088_path=second_artifact_or_partial_artifacts` y `app_server_tmux_processes_alive=0`. La proyeccion materializada reconoce `checkpoint_started*.txt` y `*_artifact.txt` estrechos, `autoprogramming observe` enriquece igual que `apps/director/goal/observe`, y `server/shutdown` no hereda snapshots activos stale cuando la respuesta `ready` trae evidencia de cleanup de backend |
| BUG-ORQ-20260701-089 | cerrado | Autoprogramming/status eficiencia | en el remoto aislado `127.0.0.1:18787`, `/api/v0/autoprogramming/status` devolvia `queue_health.running_live=8` y diagnosticos `observe_goal`, pero `efficiency_summary.state=idle`, `overall_percentage=100`, `active_runs=0`; readiness ya indicaba `queue_idle_but_goal_backend_active` | el resumen de eficiencia miraba cola/run/operator, pero no consumia `queue_health`; por tanto una cola legacy vacia ocultaba goals vivos y generaba falso verde operacional | remoto `orquesta-server-latest` arrancado con `93ee8e8c`; test `TestMCPAutoprogrammingStatusExecutorV0EfficiencyNoDeclaraIdleConGoalVivoYColaVacia`; `go test -count=1 ./modulos/orquesta-mcp` | Cierre: `efficiency_summary` incorpora `queue_health.running_live` como trabajo vivo, publica `state=live`, `active_runs`, `alive_stuck_status=observed`, razon `running_live=N` y cap de `overall_percentage` por debajo de 100 mientras haya goals vivos sin cierre. nota historica: BUG-088 quedo cerrado funcionalmente; los residuales de proyeccion/HTTP se absorbieron en BUG-161/162 |
| BUG-ORQ-20260701-090 | cerrado | Goal-first/observabilidad reconciliacion | en remoto aislado `d03f34bf`, `autoprogramming/status` publicaba `running_live=3`, pero el store durable `app_director_goal_states` tenia cuatro goals `running`, incluido `T260`; antes de converger tambien se observo intercambio transitorio donde `APG-002` aparecia como `running` aunque su JSON durable ya era `complete` | `autoprogramming/status` listaba estados goal-first con un unico `MaxItems=8` mezclando `running`, `complete`, `blocked` e `invalid`; cuando habia muchos `complete`, estos podian consumir el cupo y expulsar un `running` real del resumen publico | remoto `127.0.0.1:18787`, operacion `operation-ref-autoprogramming-observe-active-goals-manual-status-20260701`; lectura JSON mostro `T260 status=running last_result.status=running`, pero `status` no lo contaba; test `TestMCPAutoprogrammingStatusExecutorV0GoalsCompletosNoOcultanRunningPorMaxItems`; `go test -count=1 ./modulos/orquesta-mcp` | Cierre: `goalStatesForAutoprogrammingStatusV0` lista primero `running/blocked/invalid` y despues `complete`, con deduplicacion por `run_ref`; los completados pendientes de cierre ya no ocultan goals vivos ni generan falso subconteo en `queue_health`/`efficiency_summary` |
| BUG-ORQ-20260701-091 | cerrado | Server startup/lifecycle Goal | en ola OPES 17 tema 007 con Orquesta `daa44d9`, el servidor publico `startup_ready=true`, `status=running`, `pid=3443711` y `addr=127.0.0.1:36675`, pero segundos despues `/api/v0/server/readiness` devolvia conexion rechazada; el proceso HTTP ya no existia y quedaron vivos tmux/node/binario `codex app-server` del backend Goal `orquesta-goal-cd6f2a69a9b1f208`. En ola OPES 18 tema 026 con HEAD `393e7bfd`, readiness volvio verde y `POST /api/v0/external-work/run` fallo por conexion rechazada porque el HTTP habia muerto | readiness inicial no garantiza estabilidad del HTTP ni reconciliacion durable si el proceso muere tras publicar estado; ademas, si el HTTP desaparece despues de `startup_ready`, el backend Goal propio configurado puede quedar vivo sin servidor que lo gobierne | `TAREA_OPES_ORQUESTA_GRUPO_B_GOAL_FIRST_OBSERVE_Y_QA_MINIMOS_2026-07-01.md` apartados 32 y 35; evidencia wave17 en `OPES/.../00_control/orquesta_responses/wave17_t007_text/{base_url.txt,orquesta_head.txt,orquesta_runtime_root.txt,server.pid}` y runtime `.orquesta-runtime/grupo-b-info-wave17-t007-text-20260701T174254Z/state/orquesta_server_state_v0.json`; evidencia wave18 en `OPES/.../00_control/orquesta_responses/wave18_t026_text/` y runtime `.orquesta-runtime/grupo-b-info-wave18-t026-text-20260701T183443Z`; PIDs residuales Goal antes de limpieza manual; cierre lifecycle puro en `docs/incidencias/incidencia_orquesta_server_startup_lifecycle_puro_stale_2026-07-02.md`; tests `TestWaitForStateHealthyV0MarcaStaleSiServidorMuereTrasReadinessV0`, `TestMarkServerProcessStaleStateV0ExponeCausaTrasReadinessV0`, `TestWaitForStateHealthyV0LimpiaGoalBackendConfiguradoSiMuereTrasReadinessV0`; `go test -count=1 ./modulos/orquesta-server`; `go test -count=1 ./cmd/orquesta-server -run 'TestWaitForStateHealthyV0MarcaStaleSiServidorMuereTrasReadinessV0|TestWaitForStateHealthyV0LimpiaGoalBackendConfiguradoSiMuereTrasReadinessV0|TestCleanupCodexGoalBackendAfterStartupFailureIfDaemonGoneV0'` | Cierre: state/readiness/status no publican `running/startup_ready` tras reconciliar stale; la espera de arranque devuelve `server_exited_after_readiness` y, en ese punto causal, ejecuta cleanup del backend Goal `app_server_tmux` configurado si el PID del daemon ya no vive, incluyendo sesiones configuradas sin owner marker solo cuando pasan la guarda de nombre/socket propio. No convierte la limpieza de procesos externos no configurados en regla del core puro |
| BUG-ORQ-20260701-092 | cerrado | Autoprogramacion/remoto Runtime Codex | el remoto aislado volvio a proponer cambios en `modulos/orquesta-runtime-codex/codex_wrapper_v0.go` que estrechan `orquesta_codex_lock_explicit_v0`: primero ignoraba cualquier env explicito si el default era `0`, despues solo respetaba `ORQUESTA_CODEX_STARTUP_LOCK_SECONDS`, y en una tercera variante solo consideraba `TIMEOUT+STALE` juntos | la automejora goal-first estaba optimizando un caso local de lock sin un contrato completo de variables explicitas; al no fijar invariantes antes de editar, generaba parches distintos que rompian la misma semantica. Esto es patron de especificacion/gating de autoprogramacion, no bug puntual del wrapper | remoto `/srv/orquesta-self/runtime/audit-225a6752-next`; parches rechazados `/home/berserk/orquesta-inbox/rejected-codex-wrapper-default-zero-override-20260701T174836Z.patch` y `/home/berserk/orquesta-inbox/rejected-codex-wrapper-lock-explicit-timeout-20260701T180305Z.patch`; tercera variante observada 2026-07-01 con diff que exige `TIMEOUT` y `STALE` simultaneos; cierre `docs/incidencias/incidencia_orquesta_runtime_codex_lock_bug_092_2026-07-02.md`; tests `TestCodexWrapperV0RespetaCadaVariableStartupLockExplicitaAunqueDefaultSeaCeroV0`, `TestCodexWrapperV0TimeoutSoloYTimeoutStaleExplicitosNoSeSustituyenV0`, `TestCodexWrapperV0RespetaStartupLockTimeoutSoloAunqueDefaultSeaCeroV0`, `TestCodexWrapperV0RespetaStartupLockTimeoutExplicitoAunqueDefaultSeaCeroV0` | Cierre: el wrapper detecta presencia de `ORQUESTA_CODEX_STARTUP_LOCK_SECONDS`, `ORQUESTA_CODEX_STARTUP_LOCK_TIMEOUT_SECONDS` y `ORQUESTA_CODEX_STARTUP_LOCK_STALE_SECONDS` con `${VAR+x}`, por lo que una variable explicita cuenta aunque el default sea `0` o el valor este vacio. La automejora no debe tocar `codexSharedStartupLockShellV0`/tests asociados sin ejecutar la bateria focal documentada en `modulos/orquesta-runtime-codex/docs/pruebas.md`; `TIMEOUT` solo, `STALE` solo y `TIMEOUT+STALE` juntos quedan cubiertos |
| BUG-ORQ-20260701-093 | cerrado | OPES/QA editorial | en Grupo B Informatica, varios temas pasaban por extension y validadores de metanotas, pero contenian andamiaje interno, calendarios de estudio, trazabilidad, refs a `.md`/SVG, bloques de canon/maestro y mezcla de temas; tras QA estricta solo 4/50 pasaban extension + texto publico sin andamiaje | el contrato de cierre OPES no distinguia `extension_pass`, `official_text_qa_pass` y `strict_editorial_qa_pass`, por lo que Orquesta podia aceptar falsos verdes editoriales si el texto crecia con anexos o contaminacion cruzada | `TAREA_OPES_ORQUESTA_GRUPO_B_GOAL_FIRST_OBSERVE_Y_QA_MINIMOS_2026-07-01.md` apartado 34; validador OPES `opes-salidas/coordinacion_temarios/tools/validate_public_text_no_study_scaffolding.py`; informes `09_validacion/informe_texto_publico_sin_andamiaje_interno.json` e `informe_avance_estricto_grupo_b_informatica_2026-07-01.json`; cierres relacionados `BUG-ORQ-20260702-095`, `BUG-ORQ-20260702-097`, `BUG-ORQ-20260702-108`, `BUG-ORQ-20260702-109`, `BUG-ORQ-20260702-110`; tests `TestValidateOPESTopicQualityContractV0DetectaAndamiajeInternoConTildesYMayusculasV0`, `TestValidateOPESTopicQualityContractV0DetectaVariantesDeAndamiajeInternoV0`, `TestValidateOPESTopicQualityContractV0DetectaContaminacionEstructuralV0`, `TestProduceOPESCausalJobsV0BloqueaRegistroPorQATemaFallidaV0`, `TestProduceOPESCausalJobsV0BloqueaRegistroPorContaminacionEstructuralV0`, `TestProduceOPESCausalJobsV0PaqueteFinalCompleteConManifestYQAGenericaNoLiberaRegistro`; `go test -count=1 ./modulos/orquesta-opes-director`; `go test -count=1 ./modulos/orquesta-opes-bridge` | Cierre agregado: `required_tests` y `manifest_cierre` distinguen `extension_pass`, `official_text_qa_pass` y `strict_editorial_qa_pass`; OPES no acepta QA generica para reparar recibos, paquete final ni `topic_registry`; el productor causal ejecuta `OPESTopicQualityContractV0` antes de registrar `ready` y proyecta `pendiente_rework_editorial` con `pending_refs`/followup cuando detecta andamiaje interno o contaminacion estructural visible. No sustituye revision semantica IA titulo-cuerpo completa, pero cierra el falso verde determinista descrito en este bug |
| BUG-ORQ-20260701-094 | cerrado | Autoprogramacion/capacidad libre | en el remoto aislado `audit-13611445`, la automejora por capacidad quedaba `skipped/capacity_queue_unknown` cuando la cola estaba vacia, aunque `target_queue` dejaba capacidad libre y el planner podia crear un scanner de backlog | `idleSelfImprovementCapacityDecisionV0` confundia cola vacia con cola desconocida; eso bloqueaba el modelo goal-first precisamente cuando no habia trabajo en vuelo y Orquesta debia preparar automejora segura | `docs/incidencias/incidencia_orquesta_automejora_cola_vacia_capacity_queue_unknown_2026-07-02.md`; test `TestRuntimeV0SupervisorPreparaAutomejoraConColaVaciaBajoObjetivoV0`; `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` | Cierre: se elimina el veto `capacity_queue_unknown`; con planner disponible, sin intento en vuelo y `QueueSize=0`, la decision calcula `capacity_free`, `FreeCapacity=TargetQueue` y `MaxRequests=min(max_requests, free_capacity)`, lanzando el scanner si el puerto lo propone |
| BUG-ORQ-20260702-095 | cerrado | OPES/QA texto publicable | en Grupo B Informatica, temas 029 y 030 conservaban bloques visibles `Preguntas De Recuperacion`, `Preguntas de recuperacion`, `Repaso Espaciado`, `Día 0`, `mapa mental`, y frases como `La respuesta debe empezar` o `Una respuesta fuerte empieza`; el validador estricto/global podia dejar temas con extension y QA oficial en verde o solo marcar una parte, pese a que esos bloques son andamiaje de estudio no publicable | las reglas de `validate_public_text_no_study_scaffolding.py` y/o el contrato Orquesta OPES no normalizaban suficientemente mayusculas/minusculas, tildes, variantes de encabezado y texto de estrategia de respuesta; se generaba falso verde de cierre textual | `docs/incidencias/incidencia_orquesta_opes_public_text_study_scaffolding_variants_2026-07-02.md`; tests `TestValidateOPESTopicQualityContractV0DetectaAndamiajeInternoConTildesYMayusculasV0`, `TestValidateOPESTopicQualityContractV0DetectaVariantesDeAndamiajeInternoV0`, `TestProduceOPESCausalJobsV0BloqueaRegistroPorQATemaFallidaV0`; no se toco OPES productivo | Cierre Orquesta: `OPESTopicQualityContractV0` normaliza frases publicables con minusculas, tildes compuestas/descompuestas y separadores no alfanumericos antes de buscar andamiaje interno; el productor causal ya ejecuta el gate antes de `ready`, proyecta `topic-quality-opes_public_text_study_scaffolding` y crea rework por tema cuando falla |
| BUG-ORQ-20260702-096 | cerrado | Goal-first/lifecycle operativo | status, shutdown y start/readiness podian discrepar sobre goals vivos o proceso stale: `operator.active_runs` quedaba vacio con `running_live>0`, shutdown no bloqueaba un `GoalWorkState` `running` sin timeout, y start podia aceptar un snapshot `startup_ready` antes de reconciliar muerte del PID | el lifecycle goal-first estaba proyectado por superficies separadas sin una vista operacional comun de estado vivo, backend y statefile | `docs/incidencias/incidencia_orquesta_goal_first_lifecycle_status_shutdown_readiness_2026-07-02.md`; tests `TestMCPAutoprogrammingStatusExecutorV0EfficiencyNoDeclaraIdleConGoalVivoYColaVacia`, `TestStackShutdownActiveWorkReaderV0BloqueaBackendRunningSinTimeoutLocal`, `TestWaitForStateHealthyV0MarcaStaleSiServidorMuereTrasReadinessV0`; `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-mcp ./modulos/orquesta-app-codex-stack ./modulos/orquesta-opes-director` | Cierre acotado: `autoprogramming/status` proyecta goals goal-first en `operator.active_runs`, shutdown trata `running` como `goal_backend/backend_still_running` y start reconcilia liveness durante la espera estable para devolver `server_exited_after_readiness`/`server_process_stale`. Quedan abiertos los cortes finos por alto consumo y limpieza de huerfanos fuera del owner recuperable |
| BUG-ORQ-20260702-097 | cerrado | OPES/cierre paquete final | `completed_syllabus_package` podia cerrar con una categoria generica `qa` sin distinguir extension, QA oficial de texto y QA estricta editorial | el contrato OPES final agregaba evidencias y permitia falsos verdes editoriales si faltaba `strict_editorial_qa_pass` o sus informes | `docs/incidencias/incidencia_orquesta_opes_final_package_qa_passes_2026-07-02.md`; tests `go test -count=1 ./modulos/orquesta-opes-bridge ./modulos/orquesta-app-codex-stack ./modulos/orquesta-web` | Cierre: required tests separados `extension_pass`, `official_text_qa_pass` y `strict_editorial_qa_pass`; el stack exige `qa_passes` y `qa_report_refs` en `manifest_cierre`; si falta QA estricta devuelve `domain_work_opes_strict_editorial_qa_missing` y no completa el paquete |
| BUG-ORQ-20260702-098 | cerrado | Web Nueva App/wizard | el wizard guiado no mostraba con claridad preguntas pendientes ni estado `lista_para_solicitar`; `onion` no estaba como accion guiada visible aunque era arquitectura soportada | la UI consumia sesion guiada de forma parcial y no enlazaba ayuda larga por campo, de modo que el operador podia creer que el contrato estaba cerrado cuando quedaban dudas | `docs/incidencias/incidencia_orquesta_web_nueva_app_wizard_contrato_guiado_2026-07-02.md`; `go test -count=1 ./modulos/orquesta-web` | Cierre: muestra lista de preguntas pendientes, estado terminal, bloqueo de acciones finales mientras falten respuestas, accion `architecture_onion`, anchors de guia y enlaces contextuales desde campos principales |
| BUG-ORQ-20260702-099 | cerrado | Goal-first/alto consumo checkpoint-only | un goal Codex podia seguir `active` con mas de 100k tokens y solo refs de checkpoint, sin receipt de dominio ni entrega recuperable, y `status` no publicaba una causa especifica | faltaba politica operacional para distinguir progreso inicial recuperable de consumo alto con solo checkpoint; esto alimentaba falsos esperas/replans ambiguos | `docs/incidencias/incidencia_orquesta_goal_first_checkpoint_only_high_consumption_2026-07-02.md`; tests `TestMCPAutoprogrammingStatusExecutorV0CheckpointOnlyHighConsumptionEsBloqueante`; `go test -count=1 ./modulos/orquesta-mcp` | Cierre acotado: `autoprogramming/status` emite `checkpoint_only_high_consumption`, severidad `blocked`, `tokens_used`, checkpoint refs y accion `replan_narrow_context`. Quedan abiertos stop seguro/reconciliacion terminal y politica temporal fina |
| BUG-ORQ-20260702-100 | cerrado | Goal-first/resultados gigantes base64 | un resultado terminal goal-first podia traer `data:image/...;base64` o salidas enormes en `summary`, refs de artefactos/evidencias o evidencias de tests requeridos, propagando binario al receipt y al estado durable | faltaba sanitizacion en la frontera de ingestion del resultado Codex; aunque la herramienta visual debia limitar su salida, el cierre no podia confiar en que el marcador o JSON terminal ya vinieran compactos | `docs/incidencias/incidencia_orquesta_goal_first_sanitiza_base64_resultados_2026-07-02.md`; tests `TestServerCodexAppServerGoalBackendV0SanitizaResultadoDurableConBase64V0`, `TestCodexAppServerFinalMarkerTextV0RecortaPrefijoGrandeV0`; `go test -count=1 ./cmd/orquesta-server` | Cierre: `cmd/orquesta-server` reemplaza valores base64/gigantes por refs hash compactas, anade `evidence-ref-codex-app-server-goal-result-output-sanitized` y recorta la ventana del marcador final desde `ORQUESTA_GOAL_RESULT_V0`. Queda abierto el residual de tool-output multimodal antes de llegar al resultado |
| BUG-ORQ-20260702-101 | cerrado | Shutdown/Goal backend tmux residual | `server/shutdown forced=true` podia declarar listo si el `AppGoalStateStore` ya no exponia un goal `running`, aunque quedaran vivos owner marker, socket, sesion tmux o pane PID del backend Codex Goal propio | shutdown dependia demasiado del estado logico Goal y no interrogaba la identidad runtime del backend configurado; eso permitia falsos `shutdown_ready` con consumo real vivo | `docs/incidencias/incidencia_orquesta_shutdown_detecta_restos_tmux_goal_2026-07-02.md`; tests `TestCodexAppServerTmuxBackendV0ReadActiveShutdownWorkDetectaRestosPropiosV0`, `TestStackShutdownActiveWorkReaderV0BloqueaRestosBackendAunqueStateStoreNoRunningV0`; `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-app-codex-stack` | Cierre: `app_server_tmux` implementa `ReadActiveShutdownWorkV0`, reporta `goal_backend/backend_still_running` para restos propios y el stack consulta esos lectores ademas del store. No reclama ownership de procesos ajenos sin marker/sesion configurada |
| BUG-ORQ-20260702-102 | cerrado | Goal-first/write-set auditable | el cierre goal-first solo veia `artifact_refs` opacas y podia aceptar aunque el runtime hubiese materializado rutas fuera del `write_set` | faltaba un canal neutral para rutas/materializaciones observadas y una validacion de pertenencia exacta/descendiente contra scopes autorizados | `docs/incidencias/incidencia_orquesta_goal_first_artifact_paths_write_set_2026-07-02.md`; tests `TestValidateGoalWorkClosureV0BloqueaArtifactPathsFueraDeWriteSet`, `TestCodexGoalObserverV0LlamaObserverInyectado`, `TestServerCodexAppServerGoalBackendV0ObservaResultadoDurableSinMarcadorV0`; `go test -count=1 ./modulos/orquesta-goal ./modulos/orquesta-runtime-codex-goal ./cmd/orquesta-server` | Cierre: `GoalWorkResultV0.artifact_paths` viaja desde Codex Goal y el JSON/marker terminal; el validador bloquea `goal_artifact_path_out_of_scope`; el servidor anade la ruta relativa del `orquesta_goal_result_v0.json` durable observado |
| BUG-ORQ-20260702-103 | cerrado | External-work/rutas opacas | refs compactas derivadas de paths absolutos podian verse como rutas ejecutables deformadas en `context_refs[input_field_value]` | la representacion no distinguia claramente entre ref opaca y path de filesystem, induciendo a agentes a hacer `ls`/`wc` sobre valores normalizados | `docs/incidencias/incidencia_orquesta_external_work_rutas_opacas_no_ejecutables_2026-07-02.md`; tests `TestBuildExternalWorkGoalWorkSpecV0NoExponeRutasLocalesComoContextRefsEjecutables`; `go test -count=1 ./modulos/orquesta-external-work-run` | Cierre: los paths locales absolutos en input_fields se resumen como objeto `local_path_ref` con `executable_path:false`, `basename_hint` y resolucion opaca; acceptance criteria explicita que no son paths de filesystem |
| BUG-ORQ-20260702-104 | cerrado | Tooling/codebase-memory-mcp | procesos `codebase-memory-mcp` nacidos fuera del broker central podian quedar vivos sin lease/owner marker y consumir CPU aunque Orquesta ya protegiera sus propias consultas | el watchdog solo reconciliaba leases gestionados por Orquesta; faltaba observacion de procesos externos y parada opt-in por politica operativa | `docs/incidencias/incidencia_orquesta_codebase_memory_huerfanos_watchdog_2026-07-02.md`; tests `TestServerCodeContextToolProcessGuardV0DetectaHuerfanoSinPararPorDefecto`, `TestServerCodeContextToolProcessGuardV0ParaHuerfanoSoloConOptInYEdadMinima`, `TestParseServerPSCodeContextToolProcessesV0`; `go test -count=1 ./cmd/orquesta-server -run 'Test(ServerCodeContextToolProcessGuard|ParseServerPSCodeContextToolProcesses|RunServerCodeContextToolWatchdogLoopAsync|ServerCodeContextToolWatchdog|ServerFileCodeContextToolOwner)'` | Cierre: el watchdog lista procesos `codebase-memory-mcp`, protege PIDs con owner marker, reporta huerfanos antiguos y solo los para con `ORQUESTA_CODEBASE_BROKER_WATCHDOG_STOP_ORPHANS=true`; la edad minima queda en `ORQUESTA_CODEBASE_BROKER_WATCHDOG_ORPHAN_MIN_AGE_SECONDS` |
| BUG-ORQ-20260702-105 | cerrado | Goal-first/observe reconciliacion | si un goal materializaba artefactos y QA pass dentro de su `write_set` pero no escribia receipt terminal, `observe`/`status` no daban accion clara y podia parecer bloqueo indefinido o cierre recuperable no causal | el estado vivo goal-first no reconciliaba filesystem autorizado, QA estructurada y receipts terminales en una proyeccion accionable | `docs/incidencias/incidencia_orquesta_goal_first_repair_receipt_materializado_2026-07-02.md`; tests `TestStackGoalMaterializedRefsSourceV0DetectaReceiptTerminalAusenteTrasQAPass`, `TestStackGoalMaterializedRefsSourceV0ReparaReceiptTerminalConPuertosV0`, `TestCodexStackObserveAppDirectorGoalExecutorV0TimeoutSnapshotReparaReceiptMaterializadoV0`, `TestCodexStackObserveAppDirectorGoalExecutorV0RepairReceiptRequiereReworkSinReceiptDominioV0`; `go test -count=1 ./modulos/orquesta-app-codex-stack` | Cierre ampliado: el stack escanea solo el `write_set`, acepta QA pass solo desde JSON estructurado, publica refs opacas de artefacto y, con `GoalStateStore` + `GoalClosureValidator`, materializa `GoalWorkResultV0` terminal. Si el validador acepta persiste `LastClosure.accepted`; si falta receipt de dominio/evidencia/tests, persiste `repair_receipt_requires_rework` sin cerrar por filesystem |
| BUG-ORQ-20260702-106 | cerrado | Autoprogramacion/idle capacity | el remoto propuso que `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS=0` desactive solo el reloj idle, pero el servidor seguia interpretandolo como `IdleSelfImprovementDisabled=true`, apagando tambien `capacity_free` | habia dos conceptos mezclados en una sola bandera: apagado global por sesion de dominio y apagado del disparador por tiempo | `docs/incidencias/incidencia_orquesta_automejora_after_zero_capacity_free_2026-07-02.md`; tests `TestResolveAutoprogrammingIdleSelfImprovementConfigV0CeroDesactivaSoloRelojIdle`, `TestRuntimeV0SupervisorPreparaCapacidadAunqueRelojIdleEsteDesactivadoV0`, `TestServerConfigFromEnvV0ConfiguraAutomejoraIdleV0`; `go test -count=1 ./modulos/orquesta-autoprogramming`; `go test -count=1 ./modulos/orquesta-server -run 'Test(RuntimeV0SupervisorPreparaCapacidadAunqueRelojIdleEsteDesactivado|RuntimeV0SupervisorPreparaAutomejoraConColaVacia|RuntimeV0IdleSelfImprovementAfterZeroDesactivaPlanificacion|NormalizeConfigV0)'`; `go test -count=1 ./cmd/orquesta-server -run 'TestServerConfigFromEnvV0(ConfiguraAutomejoraIdle|AceptaAliasLegacyDeAutomejoraIdle|AutomejoraIdleDefaultYApagado|DesactivaAutomejoraIdle|DetectaOPES|CanonicaGana)'` | Cierre: `ConfigV0` separa `IdleSelfImprovementDisabled` global de `IdleSelfImprovementIdleDisabled`; `AFTER_SECONDS=0` permite `capacity_free` bajo `target_queue`; los contextos OPES/dominio no aislados siguen bloqueando toda automejora |
| BUG-ORQ-20260702-107 | cerrado | Autoprogramacion goal-first/state marker | `PrepareAutoprogrammingRunV0` persistia `GoalWorkStateV0` pero no `GoalWorkRunMarkerV0`, a diferencia de la ruta HTTP/app nueva; tras perdida/corrupcion de state, status/shutdown/reparacion quedaban sin evidencia durable para reconstruir el goal | contrato goal-first duplicado entre rutas de entrada; el marker se trataba como detalle de la ruta HTTP cuando es parte del lifecycle minimo | `docs/incidencias/incidencia_orquesta_autoprogramacion_goal_first_marker_ausente_2026-07-02.md`; tests `TestPrepareAutoprogrammingRunV0BackendGoalCompletoActivaGoalFirstPorComposicion`, `TestPrepareAutoprogrammingRunV0GoalReadyMultiGoalLanzaBatchSinLegacy`, `TestPrepareAutoprogrammingRunV0GoalReadyLaunchFailedPersisteStateBloqueadoV0`, `TestPrepareAutoprogrammingRunV0GoalReadyRunExistenteConStateSinMarkerReparaMarkerSinRelanzarV0`; `go test -count=1 ./modulos/orquesta-app-codex-stack` | Cierre: la autoprogramacion goal-first guarda marker junto a state, repara marker ausente en runs existentes sin relanzar y falla con `autoprogramming_goal_marker_save_failed` si el marker configurado no persiste |
| BUG-ORQ-20260702-108 | cerrado | Goal-first/QA OPES materializada | el detector de artefactos materializados aceptaba JSON generico `passed:true`, `ok:true` o `status:ok` como QA pass tambien en contexto OPES, pudiendo recomendar `repair_receipt` sin la terna editorial exigida | contrato QA generico y contrato OPES compartian detector sin distinguir dominio; esto reintroducia falsos verdes de QA editorial | `docs/incidencias/incidencia_orquesta_goal_materialized_qa_opes_generica_2026-07-02.md`; tests `TestStackGoalMaterializedRefsSourceV0NoAceptaQAGenericaEnContextoOPES`, `TestStackGoalMaterializedRefsSourceV0AceptaQAGenericaSinContextoOPES`, `TestStackGoalMaterializedRefsSourceV0AceptaTernaQAEnContextoOPES`; `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'Test(StackGoalMaterializedRefsSourceV0|CodexStackObserveAppDirectorGoalExecutorV0TimeoutSnapshot)'` | Cierre: apps no OPES conservan QA generica recuperable; en OPES el scanner exige `extension_pass`, `official_text_qa_pass` y `strict_editorial_qa_pass` o aliases estructurados antes de publicar QA pass |
| BUG-ORQ-20260702-109 | cerrado | OPES/topic_registry QA final | `topic_registry` podia liberar un paquete final con `opes-final-evidence:qa` generico aunque faltaran `extension_pass`, `official_text_qa_pass` y `strict_editorial_qa_pass` | contrato de cierre OPES divergente: stack/bridge exigian terna QA, pero el registro seguia aceptando la categoria historica `qa` | `docs/incidencias/incidencia_orquesta_opes_topic_registry_qa_generica_2026-07-02.md`; tests `TestProduceOPESCausalJobsV0PaqueteFinalCompleteConManifestYQAGenericaNoLiberaRegistro`, `TestProduceOPESCausalJobsV0PaqueteFinalCompleteConManifestCompatibleYQATernaLiberaRegistro`; `go test -count=1 ./modulos/orquesta-opes-director` | Cierre: el registro exige manifest compatible con `qa_passes` y `qa_report_refs` o evidencias estrictas separadas; `opes-final-evidence:qa` ya no libera por si sola |
| BUG-ORQ-20260702-110 | cerrado | OPES/QA por tema causal | `OPESTopicQualityContractV0` detectaba andamiaje, metacomentarios y minimos, pero el productor causal no lo ejecutaba antes de actualizar `topic_registry`; una entrega con `ready` podia no generar rework editorial | validador determinista aislado, sin wiring en `topicRegistryStatusForRecordV0` ni `followupRefsForRecordV0` | `docs/incidencias/incidencia_orquesta_opes_topic_quality_no_cableada_2026-07-02.md`; tests `TestProduceOPESCausalJobsV0BloqueaRegistroPorQATemaFallidaV0`, `TestProduceOPESCausalJobsV0NoBloqueaRegistroConQATemaCompletaV0`; `go test -count=1 ./modulos/orquesta-opes-director` | Cierre: cuando el artefacto declara QA textual de tema, el productor ejecuta el contrato, publica `pendiente_rework_editorial`, añade `pending_refs` e inicia followup causal si falla; si no declara QA, no introduce veto nuevo |
| BUG-ORQ-20260702-111 | cerrado | CLI/lifecycle Goal-first | `orquesta-server stop` fallaba con `daemon_identity_unavailable` cuando el HTTP ya habia muerto tras `startup_ready`, dejando sin ruta CLI para limpiar el backend Goal/tmux configurado aunque el PID registrado estuviera muerto; ademas `--force` podia convertir rechazos de shutdown por trabajo vivo en senal local | la parada exigia identidad HTTP viva antes de reconciliar statefile y reutilizar cleanup del backend propio, pero el modo forzado no distinguia fallo de transporte de conflicto explicito `backend_still_running`/`active_goals_present` | `docs/incidencias/incidencia_orquesta_stop_force_limpia_backend_goal_stale_2026-07-02.md`; tests `TestStopServerCommandV0ForceConDaemonMuertoLimpiaBackendGoalConfigurado`, `TestStopServerCommandV0SinForceConDaemonMuertoNoLimpiaBackendGoal`, `TestStopServerCommandV0ForceConDaemonIdentityMismatchNoLimpiaBackendGoal`, `TestShutdownRequestErrorAllowsSignalV0SoloConTimeoutYEstadoDrenado`; `go test -count=1 ./cmd/orquesta-server` | Cierre: sin `--force` conserva el fallo y no limpia; con `--force --reason` y PID no vivo + HTTP no alcanzable reconcilia `stale` y limpia `app_server_tmux`; si hay identidad viva distinta o shutdown publica trabajo vivo, no limpia ni senaliza |
| BUG-ORQ-20260702-112 | cerrado | Goal-first/status alto consumo | un goal activo con mas de 100k tokens y sin checkpoint, artefactos ni receipts se publicaba como `active_no_checkpoint_yet` informativo, recomendando esperar aunque no habia entrega recuperable | la politica diferenciaba espera inicial y `checkpoint_only_high_consumption`, pero no el caso sin ningun artefacto; alto consumo sin checkpoint necesita bloqueo operacional antes de relanzar | `docs/incidencias/incidencia_orquesta_goal_first_no_checkpoint_high_consumption_2026-07-02.md`; test `TestMCPAutoprogrammingStatusExecutorV0SinCheckpointHighConsumptionEsBloqueante`; `go test -count=1 ./modulos/orquesta-mcp` | Cierre acotado: `autoprogramming/status` emite `goal_active_no_checkpoint_high_consumption`, severidad `blocked`, `tokens_used`, sin refs de artefacto/receipt y accion `replan_narrow_context`; quedan fuera checkpoint temprano, limites de salida y reconciliacion terminal tras shutdown forzado |
| BUG-ORQ-20260702-119 | cerrado | DomainWork/HTTP transport | `orquesta-domain-work-http` aceptaba respuestas 2xx con `status=running`, `status=completed` o `pending` como job/receipt valido si traian `job_ref` o `receipt_ref`; el test de retry ocultaba el caso con `status=pending` | el cliente HTTP validaba transporte, JSON y refs, pero no validaba que el status perteneciera al contrato puro `accepted|invalid`; se mezclaba lifecycle externo con resultado de mutacion DomainWork | `docs/incidencias/incidencia_orquesta_domain_work_http_status_transport_2026-07-02.md`; tests `TestClientV0RechazaEstadosHTTPFueraDelContratoDomainWork`, `TestClientV0AceptaRespuestaInvalidDeDominioSinRefsTerminales`; `go test -count=1 ./modulos/orquesta-domain-work-http`; `go test -count=1 ./modulos/orquesta-domain-work ./modulos/orquesta-domain-work-memory ./modulos/orquesta-domain-work-file ./modulos/orquesta-domain-work-sql ./modulos/orquesta-domain-work-http` | Cierre: el cliente devuelve `domain_work_http_response_status_invalid` para estados fuera de contrato, exige ref solo en `accepted` y permite `invalid` como rechazo de dominio sin ref terminal; los estados de lifecycle deben ir por records/status, no por el puerto de mutacion |
| BUG-ORQ-20260702-116 | cerrado | Deploy/self-programming remoto | el perfil aislado dependia de revision manual para asegurar puerto solo loopback, binds bajo `/srv/orquesta-self`, ausencia de Docker socket/rutas productivas, usuario no-root, OPES/DomainWork/promocion desactivados y backend Goal por `app_server_tmux`; ademas el compose dejaba `read_only: false` | una regresion pequena en compose/env podia abrir superficie externa o productiva antes de ser detectada por `docker inspect` manual | `docs/incidencias/incidencia_orquesta_deploy_self_programming_contrato_remoto_seguro_2026-07-02.md`; test `TestSelfProgrammingComposeV0IsRemoteSafe`; `go test -count=1 ./deploy/self-programming`; `go test -count=1 ./modulos/orquesta-deploy` | Cierre: `docker-compose.yml` usa `read_only: true`; el contrato estatico valida puertos, binds, no Docker socket, no privilegios, env seguro y runbook; queda como limite que no arranca Docker remoto ni sustituye el `docker inspect` en host |
| BUG-ORQ-20260702-113 | cerrado | Autoprogramacion/backlog planner | el remoto aislado detecto que una entrada de backlog con solo `section_ref`/titulo, sin `narrative=true`, podia convertirse en run ejecutable aunque era una seccion documental ya registrada | el planner dependia de un booleano del adaptador y no de la estructura minima del contrato ejecutable, creando trabajo artificial y ruido de automejora | `modulos/orquesta-autoprogramming/docs/borrador_rebase_apg003_idle_self_improvement_2026-07-01.md`; tests `TestAutoprogrammingIdleSelfImprovementAPG003ContratoPuroYEvidencia`, `TestAutoprogrammingIdleSelfImprovementAPG005`; `go test -count=1 ./modulos/orquesta-autoprogramming` | Cierre: entradas sin `task_ref`, `write_set`, tests, criterios ni refs de contexto se proyectan como `skipped/narrative_section`; se conserva el scanner de backlog para descubrir huecos reales sin convertir narrativa en runs |
| BUG-ORQ-CODEBASE-20260702-001 | cerrado | Tooling/codebase status | `orquesta.codebase.status.v0` no aceptaba observaciones compactas de owner marker, por lo que un lease expirado con CPU alta podia publicarse como `request_stop` aunque el marker indicase `active_requests>0` | la proyeccion publica del broker evaluaba leases solo con defaults del store y no con telemetria viva del proceso observado, creando falso positivo para watchdog/operador | `docs/incidencias/incidencia_orquesta_codebase_status_observaciones_owner_marker_2026-07-02.md`; `docs/incidencias/incidencia_orquesta_codebase_owner_marker_wiring_2026-07-02.md`; tests `TestMCPCodebaseStatusHTTPHandlerV0ObservacionesEvitanParadaConPeticionesActivas`, `TestServerCodebaseStatusOwnerMarkerExecutorV0InyectaPeticionesActivas`, `TestBuildServerAppHandlerV0CodebaseStatusPublicoUsaOwnerMarkersFileBased`; `go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPCodebaseStatus'`; `go test -count=1 ./cmd/orquesta-server` | Cierre: el input MCP/HTTP acepta `observations[]` con `lease_ref`, CPU y peticiones activas; el servidor deriva automaticamente esas observaciones desde owner markers file-based para el binding MCP real y `/api/v0/codebase/status`; el status publica `continue/active_requests` y no recomienda `stop_expired_code_context_tool_lease` cuando hay trabajo activo |
| BUG-ORQ-CODEBASE-20260702-002 | cerrado | Tooling/CODEX_HOME subagentes | la proyeccion de `CODEX_HOME` podia conservar una configuracion directa de `codebase-memory-mcp` si el ledger antiguo tenia `tool=codebase-memory-mcp` y `enabled=1`, aunque no declarase `direct_mcp=true` | un subagente podia heredar MCP directo y saltarse el broker central, reabriendo el patron de CPU runaway por indexadores duplicados | revision subagente 2026-07-02; tests `TestCodexWaveCodeHomeProjectionV0DesactivaCodebaseMemoryMCPConEnabledSinOptInDirecto`, `TestCodexWaveCodeHomeProjectionV0RespetaCodebaseMemoryMCPConOptInDirecto`; `go test -count=1 ./cmd/orquesta-server -run 'TestCodexWaveCodeHomeProjectionV0(DesactivaCodebaseMemoryMCP|RespetaCodebaseMemoryMCP)'` | Cierre: `codexWaveCodebaseMemoryMCPOptInV0` exige `enabled=true` y `direct_mcp=true`; `enabled=1` solo conserva el broker central pero no autoriza MCP directo en subagentes. Runbook remoto y diseno del broker dejan de pedir procesos MCP activos por defecto |

| BUG-ORQ-20260702-114 | cerrado | Runtime Codex Goal | launcher/observer Codex Goal devolvian `codex_goal_*_rejected` generico cuando el backend fallaba sin `IssueCode`, ocultando `ResetStdio`, tmux salido o permisos `bwrap` | la frontera neutral dependia de que toda composicion rellenase `IssueCode`; los errores conocidos del backend no quedaban normalizados si cruzaban solo como `error` | `docs/incidencias/incidencia_orquesta_codex_goal_launcher_issue_code_fallback_2026-07-02.md`; tests `TestCodexGoalLauncherV0ClasificaErrorBackendSinIssueCodeV0`, `TestCodexGoalObserverV0ClasificaErrorBackendSinIssueCodeV0`; `go test -count=1 ./modulos/orquesta-runtime-codex-goal` | Cierre: `orquesta-runtime-codex-goal` conserva `IssueCode` explicito y, si falta, infiere codigos compactos para stdio wrapper, tmux exited, permiso/operation not permitted, auth/cuota, socket, standalone o comando ausente sin publicar stderr/rutas |
| BUG-ORQ-20260702-118 | cerrado | OPES/topic quality visual | `topic_registry` podia pedir rework `opes_visual_didactic_function_required` aunque el artefacto declarase `visuals` con `raster=true`, `anchor_ref`, `didactic_function` y evidencias | el productor causal activaba `RequireDidacticVisual`, pero no hidrataba `Visuals` desde `PayloadFields` antes de llamar a `OPESTopicQualityContractV0` | `docs/incidencias/incidencia_orquesta_opes_topic_quality_visuals_payload_2026-07-02.md`; tests `TestProduceOPESCausalJobsV0NoBloqueaRegistroConVisualDidacticoDeclaradoV0`; `go test -count=1 ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-director ./modulos/orquesta-opes-topic-registry` | Cierre: `orquesta-opes-director` normaliza evidencias visuales desde JSON `visuals`/aliases y campos escalares compactos; conserva el gate de raster + ancla + funcion didactica y evita falsos reworks cuando OPES ya aporta evidencia estructurada |
| BUG-ORQ-20260702-121 | cerrado | OPES/final package topic quality | el paquete final OPES podia cerrar con `qa_passes` y evidencias agregadas, pero sin refs durables a resultados de `OPESTopicQualityContractV0` por tema | el validador de tema y el cierre agregado eran contratos separados; faltaba convertir la QA por tema en evidencia obligatoria del `manifest_cierre` final | `docs/incidencias/incidencia_orquesta_opes_final_package_topic_quality_contract_2026-07-02.md`; tests `TestDefaultDomainWorkArtifactSubmissionBuilderV0NoCompletaOPESFinalSinTopicQualityContractRefs`, `TestGoalDomainReceiptClosureValidatorV0BloqueaOPESFinalSinTopicQualityContractRefsV0`, `TestOperationalClosureSourceV0NoCierraOPESFinalSinResultadosTopicQualityPorTema`; `go test -count=1 ./modulos/orquesta-app-codex-stack` | Cierre: `opes_final_package_evidence_manifest.v0` exige `topic_quality_contract_result_refs` o `topic_quality_contract_results`; el builder deja no terminal el artefacto sin esas refs y el validador bloquea `GoalWorkClosure` con `domain_work_opes_topic_quality_contract_missing` |
| BUG-ORQ-20260702-120 | cerrado | OPES/Orquesta disponibilidad | durante el cierre Grupo B Informatica, OPES tuvo que seguir por fallback local porque Orquesta estaba `server unreachable`, `autoprogramming unreachable`, sin procesos servidor ni Codex vivos; ademas se detectaron contratos pendientes de visual manifest, RAG metadata, QA tests/tutor y audio reanudable | la composicion OPES depende de Orquesta como director, pero no habia estado `stopped/crashed/unreachable` suficientemente accionable ni contrato ejecutable completo para esos derivados | `TAREA_OPES_ORQUESTA_GRUPO_B_GOAL_FIRST_OBSERVE_Y_QA_MINIMOS_2026-07-01.md`; `/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/INCIDENCIA_GRUPO_B_INFORMATICA_ORQUESTA_AUDIO_QA_2026-07-02.md`; avance RAG/visual `docs/incidencias/incidencia_orquesta_opes_finalpkg_rag_visual_2026-07-02.md` con tests `TestOPESValidateTopicPackageV1FinalPackage*`; avance audio/TTS `docs/incidencias/incidencia_orquesta_opes_audio_tts_reanudacion_2026-07-02.md` con tests `TestDomainWorkExternalCapabilityEvaluationV0AceptaAudioConReanudacionSinDuplicar`, `TestOPESRequiredTestPolicyV0AudioExigeTTSReanudableSinDuplicarMP3Validos`, `TestOPESRequiredTestPolicyV0FinalTemarioIncluyeAudioTTSReanudable`; avance QA tests/tutor con tests `TestOPESRequiredTestPolicyV0QuestionBankExigeQATripleYExplicacionesTutor`, `TestOPESRequiredTestPolicyV0TutorAssetsExigeFuentesCanonicasYRAGReconstruido`, `TestOPESRequiredTestPolicyV0FinalTemarioIncluyeQATestsYTutor`; avance finalpkg tests/tutor con `TestOPESValidateTopicPackageV1FinalPackageBloqueaTestsYTutorSinEvidencia`; avance preflight tutor con `TestOPESJobContextV0GenerateTutorAssetsExigeFuentesCanonicasV0`, `TestRunOPESDrainOnceV0TutorAssetsSinFuentesCanonicasNoPosteaOrquestaV0`; avance disponibilidad submit con `TestRunOPESDrainOnceV0BloqueaSubmitConOrquestaUnreachableV0`; cierre disponibilidad state/status con `TestMarkServerProcessStaleStateV0ExponeCausaTrasReadinessV0`, `TestServerAvailabilityV0ExponeStoppedAccionableV0`, `TestStatusServerCommandV0ReconciliaStatefileConPIDMuerto`; `python3 -m py_compile modulos/orquesta-opes-bridge/scripts/opes_validate_topic_package_v1.py`; `go test -count=1 ./modulos/orquesta-opes-bridge`; `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` | Cierre: RAG/visual avanzado valida `manifest_cierre.json`, RAG canonico con metadata `course_id`/`source_variant` o equivalentes y visuales manifestados presentes en HTML tras rebuild. Audio/TTS exige heartbeat/provider_timeout o evidencia de reanudacion sin duplicar MP3 validos antes de considerar audio listo. QA tests/tutor declara required tests publicables para `generate_question_bank`, `generate_tutor_assets` y `completed_syllabus_package`; el validador mecanico bloquea manifest sin evidencias tests/tutor, banco final insuficiente y tutor sin QA/scope report; el preflight de `generate_tutor_assets` no postea sin contenido/HTML aprobado, banco de preguntas y fuente/guarda de alcance tutor. Disponibilidad submit: si `/external-work/run` no es alcanzable, el bridge publica `status=orquesta_unreachable`, `operational_reason=orquesta_server_unreachable` y acciones para readiness/start/status sin fallback local silencioso. Disponibilidad state/status: `readiness`, `/api/v0/server/status` y `orquesta-server status` exponen `availability_status=crashed` para procesos muertos tras readiness, `availability_status=stopped` para state parado, causa publica, evidencias y acciones de inspeccion/restart/cleanup |
| BUG-ORQ-20260702-122 | cerrado | Nueva App/Goal-first real | el smoke real `scripts/smoke_goal_first_app_server_real.sh` con Orquesta `fe06470557` acepto `/api/v0/apps/director` en modo `goal_first`, pero en el poll 22 quedo `goal_status=blocked`, `run_status=bloqueada`, fase `brainstorming_arquitectura`, `summary=codex_app_server_goal_active_timeout` y `artifact_refs=0`; bajo `generated-apps` solo quedo un `orquesta_goal_result.v0` inicial con `status=invalid` y sin evidencias. Tras varios avances, el smoke real `m8rwBY` con Orquesta `a3d7d7df28` cerro Nueva App con `goal_status=complete`, `run_status=cerrada`, `closure_status=accepted`, `closure_accepted=true`, `artifact_refs=10` y `evidence_refs=10` | el conector Nueva App ya arranca goal-first, materializa app y permite cierre aceptado por evidencias; los fallos intermedios fueron timeout/rework, sandbox efectivo, ingestion de receipt terminal y reintento de receipt corregido | `docs/incidencias/incidencia_orquesta_nueva_app_smoke_goal_first_timeout_2026-07-02.md`; evidencia temporal conservada en `/srv/orquesta-self/runtime/smokes-goal-first/orquesta-goal-first-app-server.m8rwBY`; tests `TestRunSupervisorGoalFirstResidentPreparaReworkPorTimeoutInicialSinArtefactosV0`, `TestRunSupervisorGoalFirstResidentNoRelanzaTimeoutConArtefactosV0`, `TestStartAppDirectorV0GoalFirstLanzaGoalYNoEjecutaLoopLegacy`, `TestObserveAppDirectorGoalV0LanzaReworkGoalSiPolicyYPuertoDisponibles`, `TestObserveAppDirectorGoalV0LanzaReworkGoalPorTimeoutActivoV0`, `TestObserveAppDirectorGoalV0BloqueaSiReworkGoalAgotaPresupuesto`, `TestCodexStackObserveAppDirectorGoalExecutorV0IngiereReceiptTerminalMaterializadoV0`, `TestStackGoalMaterializedRefsSourceV0ReintentaReceiptTerminalCorregidoTrasReworkV0`; `go test -count=1 ./modulos/orquesta-app-codex-stack`; `go test -count=1 ./cmd/orquesta-server` | Cierre funcional: Nueva App genera app completa y Orquesta acepta el cierre. Residual separado en `BUG-ORQ-20260702-131`: shutdown posterior del smoke falla por residuos `app_server_tmux` y no debe invalidar el cierre funcional |
| BUG-ORQ-20260702-123 | cerrado | Autoprogramacion/smokes aislados | el smoke real Nueva App `UbU4HI` lanzo un goal de autoprogramacion idle (`request-ref-autoprogramming-backlog-scanner-7e8a6ae9`) dentro del servidor temporal aunque el escenario pretendia validar solo el conector Nueva App; el shutdown termino con `active_work_count=3` y trabajo idle ajeno al smoke | la composicion residente mezcla automejora por capacidad libre con smokes/conectores de dominio si no hay opt-out global; `AFTER_SECONDS=0` desactiva solo reloj idle, no capacidad libre | `docs/incidencias/incidencia_orquesta_nueva_app_smoke_goal_first_timeout_2026-07-02.md`; evidencia temporal conservada en `/tmp/orquesta-smokes/orquesta-goal-first-app-server.UbU4HI`; respuesta shutdown HTTP 409 con `goal_first` backlog scanner activo; tests `TestServerConfigFromEnvV0DesactivaAutomejoraIdlePorEnvCanonicaV0`, `TestSmokeGoalFirstAppServerRealNoLanzaAutomejoraIdleV0`, `TestRuntimeV0SupervisorNoPreparaCapacidadConAutomejoraDesactivadaV0` | Cierre: se anade `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_DISABLED=true` como opt-out canonico de toda automejora residente y el smoke Nueva App lo exporta. Los ceros de `AFTER_SECONDS`, `TARGET_QUEUE` y `MAX_REQUESTS` quedan como defensa documental, pero la garantia operativa es la bandera `DISABLED=true` |
| BUG-ORQ-20260702-124 | cerrado | Runtime Codex Goal/observe | tras el avance de materialized receipt, el smoke real `6945acc914` fallo en poll 8 con HTTP 500 `observe_app_director_goal_http_error` mientras el log del backend mostraba `rollout writer failed: Quota exceeded (os error 122)`; el operador no recibia un codigo accionable de almacenamiento/cuota local | la normalizacion de errores del backend Codex distinguia cuota de proveedor, permisos, socket y comandos, pero no cuota de filesystem/ENOSPC del app-server, por lo que `observe` no podia proyectarlo como issue publico recuperable | `docs/incidencias/incidencia_orquesta_nueva_app_smoke_goal_first_timeout_2026-07-02.md`; smoke conservado en `/tmp/orquesta-goal-first-app-server.u5tcIA`; tests `TestCodexAppServerIssueCodeForErrorV0ClasificaDiagnosticosV0`, `TestCodexGoalObserverV0ClasificaQuotaFilesystemBackendSinIssueCodeV0`, `TestMCPObserveAppDirectorGoalErrorResultFromErrorV0PreservaQuotaFilesystemV0`, `TestExternalWorkGoalFirstKnownLaunchFailureReasonV0ClasificaQuotaFilesystem` | Cierre: `Quota exceeded (os error 122)`, `disk quota exceeded`, `no space left on device` y `ENOSPC` se normalizan como `codex_app_server_storage_quota_exceeded` en el servidor, el paquete runtime Codex Goal y el ejecutor goal-first externo; `observe` puede devolver el issue publico en vez de caer a 500 generico |
| BUG-ORQ-20260702-125 | cerrado | Smoke Nueva App/diagnostico | tras precrear el directorio `generated-apps/smoke-goal-first`, `scripts/smoke_goal_first_app_server_real.sh` publicaba `generated_apps_present=1` aunque no habia ningun fichero ni app materializada | el diagnostico de fallo contaba directorios bajo `generated-apps`, mezclando preparacion de write-set con entrega real | smoke conservado en `/srv/orquesta-self/runtime/smokes-goal-first/orquesta-goal-first-app-server.gcZ6Yj`; test `TestSmokeGoalFirstAppServerRealDiagnosesAppServerAuthMissingV0` | Cierre: el smoke cuenta solo ficheros (`find ... -type f`) para `generated_apps_present`, de modo que un write-set vacio no parece app generada |
| BUG-ORQ-20260702-126 | cerrado | Smoke Nueva App/timeout | el smoke real Nueva App tenia `ORQUESTA_CODEX_GOAL_TIMEOUT_MS` por defecto en 90.000 ms aunque el `GoalWorkSpecV0` declara `MaxRuntimeSeconds=600` y el propio script observa hasta 120 polls de 5 segundos | el backend podia bloquear por `codex_app_server_goal_active_timeout` antes de agotar el presupuesto operativo del contrato, reabriendo BUG-122 por configuracion de smoke y no por incapacidad real del agente | `docs/incidencias/incidencia_orquesta_nueva_app_smoke_goal_first_timeout_2026-07-02.md`; test `TestSmokeGoalFirstAppServerRealRespetaPresupuestoNuevaAppV0`; `go test -count=1 ./cmd/orquesta-server -run 'TestSmokeGoalFirstAppServerReal(RespetaPresupuestoNuevaApp|DiagnosesAppServerAuthMissing)'` | Cierre: el smoke exporta `ORQUESTA_CODEX_GOAL_TIMEOUT_MS=600000` por defecto, alineado con el presupuesto Nueva App; BUG-122 sigue abierto hasta cierre real accepted con refs no vacias |
| BUG-ORQ-20260702-127 | cerrado | Runtime Codex Goal/presupuesto | el backend `app_server_tmux` calculaba `codex_app_server_goal_active_timeout` desde el timeout global del servidor aunque el `CodexGoalStartPacketV0` transportase un `Budget.MaxRuntimeSeconds` mayor | el contrato goal-first llevaba presupuesto por trabajo, pero el observador lo perdia al cruzar al runtime real y podia cortar goals largos antes de su limite causal | `docs/incidencias/incidencia_orquesta_nueva_app_smoke_goal_first_timeout_2026-07-02.md`; tests `TestServerCodexAppServerGoalBackendV0ActiveGoalRespetaBudgetMaxRuntimeMayorQueTimeoutGlobalV0`, `TestServerCodexAppServerGoalBackendV0ThreadReadRespetaBudgetMaxRuntimeMayorQueTimeoutGlobalV0`; `go test -count=1 ./cmd/orquesta-server -run 'TestServerCodexAppServerGoalBackendV0(ActiveGoalRespetaBudgetMaxRuntimeMayorQueTimeoutGlobal|ThreadReadRespetaBudgetMaxRuntimeMayorQueTimeoutGlobal)'` | Cierre: el runtime registra por `thread_id` el timeout efectivo de arranque y usa el mayor entre timeout global y `Budget.MaxRuntimeSeconds`; el fallback sin presupuesto conserva el timeout global |
| BUG-ORQ-20260702-128 | cerrado | Runtime Codex Goal/app-server tmux | el smoke real Nueva App `1aHPIu` ya no corto por 90s, pero bloqueo en poll 44 con resultado del agente indicando `Quota exceeded` al montar `.git`; el `config.toml` aislado del app-server conservaba entradas `[projects]` heredadas para `/workspace/project` y `/srv/orquesta-self/worktrees/orquesta` aunque el CWD real era el proyecto temporal del smoke | `app_server_tmux` copiaba `config.toml` completo desde `CODEX_HOME`, mezclando credenciales/config global con trust de proyectos ajenos; el app-server podia intentar preparar sandbox/git de workspaces que no pertenecen al goal actual | `docs/incidencias/incidencia_orquesta_nueva_app_smoke_goal_first_timeout_2026-07-02.md`; smoke conservado en `/srv/orquesta-self/runtime/smokes-goal-first/orquesta-goal-first-app-server.1aHPIu`; tests `TestCodexAppServerTmuxBackendV0FiltraProjectsAjenosDelConfigV0`, `TestCodexAppServerTmuxBackendV0PreparaCodeHomeDesdeCODEXHOMEResueltoV0`, `TestServerCodexGoalBackendFromEnvV0TmuxPreflightOKV0`; `go test -count=1 ./cmd/orquesta-server -run 'TestCodexAppServerTmuxBackendV0(FiltraProjectsAjenosDelConfig|PreparaCodeHomeDesdeCODEXHOMEResuelto)|TestServerCodexGoalBackendFromEnvV0TmuxPreflightOKV0'` | Cierre: la proyeccion tmux de `config.toml` conserva configuracion global, elimina secciones `[projects.*]` heredadas y declara como trusted solo el `ProjectWorkDir` aislado del backend; el test de preflight usa `t.TempDir` para no depender de `/tmp` con cuota |
| BUG-ORQ-20260702-129 | cerrado | Nueva App/Goal-first real | tras cerrar BUG-128, el smoke real Nueva App `cY3QQr` supera el poll 44 y el Codex home aislado queda con un unico `[projects]` para el proyecto temporal, pero termina bloqueado en poll 50 con `generated_apps_present=0`, `evidence-ref-codex-app-server-goal-high-token-usage` y resumen del agente: no puede ejecutar comandos ni escribir archivos dentro del workspace autorizado. Smokes posteriores con sandbox efectivo en proyecto temporal (`qvtugC`, `1VWYEr`) materializan app, docs, tests y receipt terminal | el smoke aislado usaba `workspace-write` aunque todo su `ProjectWorkDir`, runtime y `CODEX_HOME` eran temporales; en este entorno el app-server tmux no materializaba herramientas locales de forma fiable con ese sandbox | `docs/incidencias/incidencia_orquesta_nueva_app_smoke_goal_first_timeout_2026-07-02.md`; smoke bloqueado conservado en `/srv/orquesta-self/runtime/smokes-goal-first/orquesta-goal-first-app-server.cY3QQr`; smokes de cierre de diagnostico en `/srv/orquesta-self/runtime/smokes-goal-first/orquesta-goal-first-app-server.qvtugC` y `/srv/orquesta-self/runtime/smokes-goal-first/orquesta-goal-first-app-server.1VWYEr`; test `TestSmokeGoalFirstAppServerRealUsaSandboxEfectivoEnWorkspaceAisladoV0`; `bash -n scripts/smoke_goal_first_app_server_real.sh`; `go test -count=1 ./cmd/orquesta-server -run 'TestSmokeGoalFirstAppServerReal(RespetaPresupuestoNuevaApp|UsaSandboxEfectivoEnWorkspaceAislado|DiagnosesAppServerAuthMissing)'` | Cierre: el smoke Nueva App conserva override por env, pero por defecto usa `danger-full-access` solo dentro de su workspace temporal aislado; BUG-130 separa el siguiente bloqueo de cierre con artefactos ya materializados |
| BUG-ORQ-20260702-130 | cerrado | Nueva App/Goal-first cierre | el smoke real aislado `1VWYEr` genero app completa, 20 ficheros y `npm run verify` 4/4, y despues corrigio el receipt a `status=complete`, pero `observe` mantuvo `run_status=activa`, `closure_status=blocked` y `repair_receipt_requires_rework` | la reconciliacion de receipts materializados trataba una reparacion fallida previa como terminal y no revalidaba si el agente corregia despues `orquesta_goal_result*.json`; falso bloqueo de cierre tras evidencia ya valida | `docs/incidencias/incidencia_orquesta_nueva_app_smoke_goal_first_timeout_2026-07-02.md`; smoke conservado en `/srv/orquesta-self/runtime/smokes-goal-first/orquesta-goal-first-app-server.1VWYEr`; tests `TestStackGoalMaterializedRefsSourceV0ReintentaReceiptTerminalCorregidoTrasReworkV0`, `TestStackGoalMaterializedRefsSourceV0ReparaReceiptTerminalConPuertosV0`, `TestCodexStackObserveAppDirectorGoalExecutorV0IngiereReceiptTerminalMaterializadoV0`; `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestStackGoalMaterializedRefsSourceV0(ReintentaReceiptTerminalCorregidoTrasRework|ReparaReceiptTerminalConPuertos)|TestCodexStackObserveAppDirectorGoalExecutorV0IngiereReceiptTerminalMaterializadoV0'` | Cierre de codigo: un receipt terminal materializado se revalida aunque una reparacion previa hubiese fallado, salvo que el cierre ya este aceptado; la guarda de intento previo se mantiene para el caso sintetico sin receipt terminal. Pendiente operativo de BUG-122: reejecutar smoke real con este commit |
| BUG-ORQ-20260702-131 | cerrado | Shutdown/smoke Goal-first | el smoke real `m8rwBY` con Orquesta `a3d7d7df28` cerro Nueva App con `goal_status=complete`, `run_status=cerrada`, `closure_status=accepted`, `closure_accepted=true`, `artifact_refs=10` y `evidence_refs=10`, pero el script termino con fallo porque `/api/v0/server/shutdown` devolvio HTTP 409 `backend_still_running`, `shutdown_ready=false`, `active_work_count=3` por residuos `app_server_tmux`. Los smokes posteriores `2Ha1L8` y `0IJ0Jj` reprodujeron el mismo tipo de fallo con `active_work_count=1` para la propia sesion tmux tras cierre funcional accepted. Los reruns `eKAlp3`, `f6A9fO`, `JXbAW9` y `fFtCHQ` ya no escalaron a HTTP 500, pero el primer shutdown siguio en 409 hasta que el retry local mato el proceso `codex app-server` por socket. El smoke `4EBq6U` con HEAD `3198210b35` cerro Nueva App en poll 45 con `closure_status=accepted`, `artifact_refs=10`, `evidence_refs=10` y el primer `/api/v0/server/shutdown` devolvio HTTP 200 sin retry; `app_server_tmux_processes_alive=0` | BUG-065, BUG-076, BUG-088 y BUG-131 comparten la misma raiz: shutdown, estado vivo y lifecycle del backend Goal eran contratos separados; la limpieza estaba detras del bloqueo que la necesitaba. La solucion raiz es un puerto neutral `ActiveShutdownWorkCleanerPortV0` invocado por `cleanup_goal_backends=true`, limitado a backends Goal propios, con relectura obligatoria antes de publicar ready. La hipotesis refinada por `f6A9fO`: si el proceso no sale con `SIGTERM` antes del timeout, el cleanup no debe perder la escalada a `SIGKILL` por reutilizar un contexto ya vencido; `JXbAW9` demuestra que ese fix es insuficiente. Cierre: el cleanup nativo usa una ventana cooperativa propia mas larga que el timeout general del adaptador y, aunque el `pane_pid` no desaparezca dentro de esa ventana, continua con limpieza de procesos por socket y marker como el retry local | `docs/incidencias/incidencia_orquesta_nueva_app_smoke_goal_first_timeout_2026-07-02.md`; smokes fallidos conservados en `/srv/orquesta-self/runtime/smokes-goal-first/orquesta-goal-first-app-server.m8rwBY`, `/srv/orquesta-self/runtime/smokes-goal-first/orquesta-goal-first-app-server.2Ha1L8` y `/srv/orquesta-self/runtime/smokes-goal-first/orquesta-goal-first-app-server.0IJ0Jj`; smokes cerrados por retry local en `/srv/orquesta-self/runtime/smokes-goal-first/orquesta-goal-first-app-server.uSyxeu`, `/srv/orquesta-self/runtime/smokes-goal-first/orquesta-goal-first-app-server.eKAlp3`, `/srv/orquesta-self/runtime/smokes-goal-first/orquesta-goal-first-app-server.f6A9fO`, `/srv/orquesta-self/runtime/smokes-goal-first/orquesta-goal-first-app-server.JXbAW9` y `/srv/orquesta-self/runtime/smokes-goal-first/orquesta-goal-first-app-server.fFtCHQ`; smoke de cierre sin retry en `/srv/orquesta-self/runtime/smokes-goal-first/orquesta-goal-first-app-server.4EBq6U`; tests `TestShutdownServerV0CleanupGoalBackendsReleeYPermiteReadySiLimpiaV0`, `TestShutdownServerV0CleanupGoalBackendsNoPublicaReadySiSigueVivoV0`, `TestShutdownServerV0CleanupGoalBackendsNoLimpiaSiHayGoalFirstActivoV0`, `TestCodexAppServerTmuxBackendV0CleanupActiveWorkLimpiaSesionPropiaV0`, `TestCodexAppServerTmuxBackendV0CleanupActiveWorkMataProcesoSocketSinSesionV0`, `TestCodexAppServerTmuxBackendV0CleanupActiveWorkContinuaTrasPaneTimeoutV0`, `TestCodexAppServerTmuxBackendV0CleanupActiveWorkNoMataSesionSinOwnerNiConfigSeguraV0`, `TestSmokeGoalFirstAppServerRealPideCleanupGoalBackendsV0`, `TestSmokeGoalFirstAppServerRealShutdownLimpiaBackendPropioYReintentaV0`, `TestRequestServerShutdownV0DefaultCooperativo` | Cierre: `/server/shutdown` acepta `cleanup_goal_backends`, limpia solo recursos `app_server_tmux` propios por owner/config segura, conserva evidencias, termina procesos `codex app-server` que aun anuncian el socket propio, escala a `SIGKILL` aunque el contexto de espera de `SIGTERM` haya vencido, no aborta cleanup por timeout de pane en ruta de cleanup y vuelve a leer active work; si sigue vivo mantiene `backend_still_running`. Evidencia real: `4EBq6U` ya no necesita retry local |
| BUG-ORQ-20260702-132 | cerrado | Smoke Nueva App/shutdown tmux | el smoke real `aNcwnF` cerro funcionalmente en poll 44 con `goal_status=complete`, `run_status=cerrada`, `closure_status=accepted`, `artifact_refs=9` y `evidence_refs=10`, pero el wrapper salio con codigo 1 porque `assert_app_server_tmux_shutdown_ready` exigia que la sesion tmux siguiera viva antes del shutdown y encontro `tmux session no existe antes del shutdown: orquesta-goal-8c30b15879ac165a` | el shutdown del smoke no era idempotente ante un app-server tmux ya cerrado limpiamente antes de invocar `/api/v0/server/shutdown`; confundia cierre anticipado con fallo de limpieza | `docs/incidencias/incidencia_orquesta_nueva_app_smoke_goal_first_timeout_2026-07-02.md`; smoke conservado en `/srv/orquesta-self/runtime/smokes-goal-first/orquesta-goal-first-app-server.aNcwnF`; test `TestSmokeGoalFirstAppServerRealShutdownToleraTmuxYaCerradoV0`; `bash -n scripts/smoke_goal_first_app_server_real.sh`; `go test -count=1 ./cmd/orquesta-server -run 'TestSmokeGoalFirstAppServerReal(UsaSandboxEfectivoEnWorkspaceAislado|ShutdownToleraTmuxYaCerrado|RespetaPresupuestoNuevaApp|DiagnosesAppServerAuthMissing)'` | Cierre: el wrapper informa `tmux session ya estaba cerrada antes del shutdown`, sigue llamando al endpoint de shutdown del servidor y vuelve condicionales las comprobaciones post-shutdown de session/socket/pane cuando esos recursos ya no existen antes del shutdown |
| BUG-ORQ-20260702-133 | cerrado | Tests/cmd orquesta-server quota | `go test -count=1 ./cmd/orquesta-server` con `TMPDIR` local podia fallar igualmente en `TestServerCodexGoalBackendFromEnvV0TmuxSinAuthDegradaYConservaShutdownV0` por `disk quota exceeded` porque el test creaba su raiz con `os.MkdirTemp("/tmp", "og")` | una prueba de app_server_tmux ignoraba el entorno temporal aislado y escribia en `/tmp`, reabriendo falsos rojos de cuota local aunque el codigo bajo prueba no dependiese de `/tmp` | observado durante verificacion de BUG-065; test `TestServerCodexGoalBackendFromEnvV0TmuxSinAuthDegradaYConservaShutdownV0`; `go test -count=1 ./cmd/orquesta-server -run 'TestServerCodexGoalBackendFromEnvV0TmuxSinAuthDegradaYConservaShutdownV0'` | Cierre: el test usa `t.TempDir()` y respeta el directorio temporal del harness, evitando escribir en `/tmp` cuando la bateria se ejecuta con `TMPDIR` local por cuota |
| BUG-ORQ-20260702-134 | cerrado | Shutdown/smoke Goal-first | el smoke real aislado `A9gMud` con HEAD `fea88bb14c` cerro Nueva App en poll 53 con `goal_status=complete`, `run_status=cerrada`, `closure_status=accepted`, `artifact_refs=9` y `evidence_refs=10`, pero `/api/v0/server/shutdown` devolvio HTTP 500 `server_shutdown_executor_error`; despues el servidor quedo `stopped` por la limpieza del wrapper. El rerun `eKAlp3` con HEAD `6d21870330` cerro Nueva App en poll 49 y el primer shutdown devolvio HTTP 409 `backend_still_running` en vez de 500; el retry local posterior devolvio 200 | un error del `ActiveShutdownWorkCleanerPortV0` durante cleanup de backend Goal propio escapaba como error del caso de uso y el adaptador HTTP lo convertia en 500 generico, ocultando el estado operacional correcto: backend aun no confirmado como limpio | evidencia conservada en `/srv/orquesta-self/runtime/smokes-goal-first/orquesta-goal-first-app-server.A9gMud` y `/srv/orquesta-self/runtime/smokes-goal-first/orquesta-goal-first-app-server.eKAlp3`; test `TestShutdownServerV0CleanupGoalBackendsErrorNoEscalaAHTTP500V0`; `go test -count=1 ./modulos/orquesta-server-shutdown` | Cierre: si falla el cleaner de backend Goal propio, `ShutdownServerV0` devuelve `status=backend_still_running`, `shutdown_ready=false`, active work y evidencias `cleanup-requested`, `cleanup-error` y `backend-still-running`, sin escalar a error HTTP 500. El residual de cleanup nativo sin retry queda separado en `BUG-ORQ-20260702-131` |
| BUG-ORQ-20260703-151 | cerrado | Goal launch/observabilidad | el `launch_receipt` persistido de un launch goal-first fallido conservaba solo `issues[].code` sin mensaje ni detalle causal; las proyecciones lo disfrazaban como `goal_backend_state_unreconciled` en `autoprogramming/status` y `codex_goal_observation_rejected` en observe, aunque la verdad operacional era un fallo de prepare durante launch | faltaba una causa de primera clase en el contrato de goal-first desde launch hasta status/observe | cierre T276 en `docs/bitacora_correccion_pericial_2026-07-03.md`: commit `c8e9dbd1`, `Detail` saneado de causa raiz hasta status/observe con accion recomendada; suites verdes tras integracion | Cierre documentado; mantener tests de proyeccion de causa y no degradar de nuevo a estados genericos |
| BUG-ORQ-20260703-152 | cerrado | Backlog/parser | el parser de secciones de backlog `cmd/orquesta-server/idle_self_improvement_backlog_parser_v0.go` solo honraba etiquetas `Objetivo:`, `Alcance:`, `Criterios:`, `Tests:`, `Estado:` y `Dependencias:`; variantes como `Write-set previsto:` se descartaban en silencio y la tarea caia al write-set enlatado del area por defecto sin diagnostico | faltaba diagnostico o normalizacion de aliases recuperables para no perder scope operativo por formato equivalente | estado global en `docs/bitacora_correccion_pericial_2026-07-03.md`: "bugs 151/152/153 cerrados"; cierre posterior a la ola BUG-152 + CTX-801A/C/D antes del relevo a director Codex | Cierre documentado por bitacora; si reaparece una etiqueta equivalente sin scope, abrir regresion nueva enlazada a esta fila |
| BUG-ORQ-20260703-153 | cerrado | Goal idle/cierre | un goal de automejora idle funcionalmente terminado podia quedar `blocked`/`goal_first_blocked` para siempre sin `GoalWorkResultV0` ni validacion de cierre, o desaparecer de `active_runs` sin estado terminal visible | el packet goal-first idle no reconciliaba resultado durable terminal ni artefactos materializados como fuente de cierre | cierre T275 en `docs/bitacora_correccion_pericial_2026-07-03.md`: commit `e47c2561`, reconciliacion de goals idle por result materializado, decaimiento `goal_backend_gone_without_result` y terminal siempre visible; suites verdes tras integracion | Cierre documentado; vigilar como regresion si un goal con result formal completo vuelve a quedar running/blocked sin reconciliar |

Nota BUG-122 2026-07-02: el smoke real posterior con Orquesta `5a71671fc5`
y evidencia conservada en
`/srv/orquesta-self/runtime/smokes-goal-first/orquesta-goal-first-app-server.QsBxZZ`
ya no cae en HTTP 500 por cuota, pero bloquea en poll 56 con
`goal_status=blocked`, `run_status=bloqueada`, `closure_status=blocked`,
`current_phase=brainstorming_arquitectura` y summary empezando por
`external_runtime_blocker: sandbox shell still fails before execution with quota exceeded mounting .git`;
no materializa app ni artefactos. BUG-122 sigue
abierto hasta un smoke con cierre aceptado y refs no vacios, o hasta frontera
externa reproducible/documentada fuera de Orquesta.

Avance BUG-122 2026-07-02: `app_server_tmux` prepara directorios de `write_set`
seguros antes de `turn/start` para que Nueva App no dependa de `apply_patch`
al crear la raiz `generated-apps/...`; no crea rutas Markdown, escapes ni
`.git`. Evidencia:
`TestServerCodexAppServerGoalBackendV0PreparaDirectoriosWriteSetAntesDelTurnV0`
y `TestServerCodexAppServerGoalBackendV0LanzaThreadGoalYTurnV0`.

Revision BUG-072/BUG-075 2026-07-02: el patron de artefactos parciales deja de
ser una heuristica de scanner/log y pasa a contrato neutral ejecutable.
`GoalWorkResultV0` incorpora `materialized_artifacts`, `checklist` y
`rework_plan_refs`; `ValidateGoalWorkClosureV0` bloquea `complete` cuando hay
artefactos `partial`, `invalid` o `non_publishable`, checklist faltante o plan
de rework requerido; el adaptador Codex Goal y el parser `app_server` conservan
esos campos desde `ORQUESTA_GOAL_RESULT_V0`; la copia de estado del servidor no
los pierde al persistir resultado idle/automejora; y `observe_goal` no permite
que una issue de scanner parcial pise un cierre aceptado o una decision de
rework ya calculada. Evidencia:
`TestValidateGoalWorkClosureV0BloqueaCompleteConArtefactosParciales`,
`TestCodexGoalObserverV0ConservaArtefactosMaterializadosChecklistYRework`,
`TestServerCodexAppServerGoalBackendV0ObservaResultadoMarcadoV0`,
`TestCopyGoalWorkResultForServerStateV0ConservaContratoDeArtefactosParciales` y
`TestEnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0NoPisaCierreAceptado`.

## Riesgos arquitectonicos no funcionales

| ID | Estado | Area | Hallazgo | Riesgo | Evidencia | Accion |
| --- | --- | --- | --- | --- | --- | --- |
| ARCH-ORQ-20260630-001 | cerrado | Core workflow | `orquesta-core-workflow` concentra 154 ficheros Go no-test y `orquesta-orchestration-core` 144 | cambios pequenos tienen blast radius alto y hacen dificil aislar invariantes | `REVISION_ARQUITECTURA_HALLAZGOS_2026-06-28.md` H3; `architecture_boundaries_test.go` ya cubre ambos hubs; `docs/runbooks/plan_troceo_hubs_orquesta_2026-07-01.md` | Cierre: deuda gobernada, no refactor masivo; plan incremental por invariantes y frontera neutral verificada con tests |
| ARCH-ORQ-20260630-002 | cerrado | Hotspots UI/stack/server | ficheros monoliticos grandes, especialmente render HTML Nueva App y stack Codex | mezcla de presentacion, wiring y logica aumenta regresiones y coste de review | `REVISION_ARQUITECTURA_HALLAZGOS_2026-06-28.md` H5; medicion actual: `nueva_app_html_render_v0.go` 1803 lineas, `goal_domain_receipt_closure_validator_v0.go` 1147, `goal_first_v0.go` 924, `codex_goal_app_server_v0.go` 886 | Cierre: plan de troceo incremental documentado; cada cambio futuro en fichero >700 lineas debe mantener test focal y extraer responsabilidad pequena si aumenta acoplamiento |
| ARCH-ORQ-20260630-003 | cerrado | Operacion local | `.orquesta-runtime` ocupa aprox. 18 GB y purgas antiguas cientos de MB | busquedas/herramientas de auditoria lentas y presion de IO/disco | `du -sh .orquesta-runtime .orquesta-purged-*` local; `docs/runbooks/limpieza_runtime_local_orquesta_2026-07-01.md`; `scripts/test_orquesta_runtime_retention.sh` | Cierre: herramienta `scripts/orquesta_runtime_retention.sh` inventaria candidatos en dry-run por defecto y solo borra con `--delete --confirm-delete orquesta-runtime-retention`; no borra la raiz `.orquesta-runtime` ni contenedores globales |
| ARCH-ORQ-20260630-004 | cerrado | Codigo muerto | `orquesta-state-file/outbox` no tenia importadores externos y duplicaba mentalmente el outbox file-based vivo | superficie legacy podia confundir arquitectura de persistencia/outbox | `rg` sin importadores Go externos; `go list` solo listaba el propio paquete; sin build tags `//go:build` ni `// +build`; referencias restantes eran docs historicos | Cierre: paquete retirado en commit separado; el ledger file-based vivo queda en `orquesta-persistence`; mantener `go test ./...` como evidencia de no regresion |

## Lectura de conjunto inicial

Los bugs no parecen solo errores sueltos. Se agrupan en cuatro problemas de
diseno operativo:

1. Migracion incompleta de ciclo: Goal-first, director legacy, supervisor,
   status y MCP siguen coexistiendo. Cada ruta debe declarar si lanza Goal,
   observa Goal o exige opt-in legacy.
2. Estado repartido: proceso vivo, run, goal marker, ACK, delivery, outbox y
   registro OPES se proyectan desde stores distintos. Los falsos `stale` y los
   cierres incompletos nacen de ahi.
3. Contratos de dominio no ejecutables: OPES y Nueva App tenian reglas visibles
   en docs/UI que no siempre estaban en validadores/tests.
4. Operacion remota no canonica: sin GitHub directo, bundles, tmux, identidad
   Git y worktrees deben ser parte del contrato de trabajo, no memoria manual.

Revision 2026-06-30: los bugs de estado vivo y supervision (`006`, `007`,
`018`) quedan cerrados con tests focales y contrato de respuesta finita. Los
bugs OPES 1+6/write-set (`008`, `009`) se cierran en codigo/fake-runtime porque
la arquitectura ya fuerza materializacion causal, bloqueo por falta de hijos y
fase de integracion si el producto queda fuera del write-set canonico. La deuda
arquitectonica residual no es "otro parche de subroles": es mantener una unica
proyeccion publica que combine goal, run, procesos, ACK, deliveries y dominio
sin duplicar verdad por endpoint.

Revision adicional 2026-06-30 sobre el patch remoto `4cf62482`: sus incidencias
son utiles pero no se aplican con la numeracion original porque `019..022` ya
estan ocupadas en `14ab41f1`. Quedan integradas como `024..027`. El patron
comun de `025..027` no es un bug puntual de Orquesta sino contrato de dominio
OPES todavia no ejecutable: capacidades locales, calidad editorial visual y
cobertura/esquema de artefactos deben ser preflight/validadores opt-in con
estado publico, no instrucciones sueltas en prompts ni comprobaciones manuales.

Revision adicional 2026-06-30 sobre incidencias OPES API/TTS: `025` queda
cerrado para tool-path porque la capacidad `speech_synthesis` ya puede ejecutar
preflight real opt-in antes del launch. Lo que no queda cerrado se separa para
evitar falso verde: `028` cubre descubrimiento/readiness de instancia OPES,
`029` lifecycle atomico de arranque con backend Goal y `030` heartbeat/timeout
de proveedores largos como Edge TTS.

Revision adicional 2026-06-30 sobre Hallazgo 5 OPES: no se elimina la guarda
`project_work_dir` porque sigue protegiendo ejecucion OPES real desde el repo
equivocado. Se cierra solo el caso control/evidencia local: write-set relativo
integramente bajo `external/opes/`, sin escritura en OPES productivo.

Revision adicional 2026-06-30 sobre Hallazgos 3 y 4 OPES: el caso concreto de
H3 queda cubierto para launchers/smokes OPES porque el arranque local exporta
`ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux` por defecto y espera readiness
operativa, no `/healthz`. El discovery de rutas queda cubierto por
`/api/v0/server/resources.route_manifest`, que declara rutas montadas, metodos,
propietario y perfil de seguridad de la composicion. H4 queda cerrado en codigo:
el backend `app_server_tmux` limpia su sesion propia si el arranque falla
despues de crear tmux y antes de quedar operativo.

Revision adicional 2026-06-30 sobre BUG-027: se cierra el falso verde de cierre
goal-first para supuestos OPES. Orquesta ya no acepta como completo un receipt
que declare parcialidad, faltantes, esquema reparable/invalido, `tasks` legacy
sin `questions`, `development_task` o mero `process_status=stopped` sin
`delivery_status`/validacion de entrega completa. Queda como mejora posterior
materializar automaticamente la ola de rework por faltantes concretos.

Revision adicional 2026-06-30 sobre BUG-032: queda implementado el tramo de
owner marker file-based, observador PID/CPU/heartbeat/peticiones activas y
stopper concreto seguro para `codebase-memory-mcp`. El watchdog ya no depende
solo de CPU sintetica: puede no parar un lease expirado si el marker declara
peticiones activas, y el stopper solo envia SIGTERM al PID declarado por un
marker valido. Evidencia:
`TestServerCodeContextToolWatchdogV0UsaOwnerMarkerConPeticionesActivas`,
`TestServerFileCodeContextToolOwnerStopperV0UsaMarkerSeguro` y
`TestServerFileCodeContextToolOwnerRegistryV0RechazaOwnerInseguro`. Nueva
observacion local: PIDs `2681690` y `2681994`, hijos del Codex local, quedaron
consumiendo CPU tras uso MCP y se pararon con `SIGTERM`. Nueva observacion:
PIDs `2747740` y `2748020`, hijos de otro Codex local, quedaron vivos tras uso
MCP y se pararon con `SIGTERM`. Revision adicional: el loop residente opt-in
`code_context_tool_watchdog_loop` ya arranca desde `orquesta-server run` solo si
`ORQUESTA_CODEBASE_BROKER_WATCHDOG_ENABLED=true` y exige
`ORQUESTA_CODEBASE_BROKER_STATE_DIR`; usa el loop externo comun, markers,
observer/stopper file-based, no proyecta `external_bridge_loop disabled` cuando
esta apagado, conserva contadores `observed/stopped/errors`, un owner marker
ausente no corta el tick completo ni impide procesar otros leases, y el stopper
espera salida tras `SIGTERM` antes de escalar a `SIGKILL`.
Evidencia nueva:
`TestServerCodeContextToolWatchdogLoopConfigV0ExigeStateDirOptIn` y
`TestRunServerCodeContextToolWatchdogLoopAsyncV0DesactivadoNoProyectaBridgeV0` y
`TestServerCodeContextToolWatchdogV0OwnerMarkerAusenteNoBloqueaOtrosLeases` y
`TestServerFileCodeContextToolOwnerStopperV0EscalaSiProcesoSigueVivo` y
`TestRunServerCodeContextToolWatchdogLoopAsyncV0ParaLeaseExpiradoSinPIDV0`, `TestServerCodebaseMemoryCLIProviderV0ParseaSearchGraphYEscribeOwnerMarker`, `TestCodeContextBrokerWiringFromEnvV0ConfiguraCodebaseMemoryCLIConOptInV0`, `TestServerCodebaseMemoryCLIProviderV0RealSmokeOptIn`.
Cierre: el adaptador real usa `codebase-memory-mcp cli` detras del broker central, con lease file-based, owner marker y smoke real opt-in `ORQUESTA_CODEBASE_MEMORY_REAL_SMOKE=1`. Residual operativo: procesos nacidos fuera de Orquesta por sesiones Codex antiguas no se pueden reconciliar si no tienen lease/marker; se vigilan como incidencia operativa, no como pendiente de codigo de BUG-032.
Revision operativa posterior 2026-06-30: PIDs `3345397`, `3345721`,
`3346067`, `3346458`, `3346742` y `3347165`, hijos de un Codex antiguo y a
0% CPU, quedaron como instancias duplicadas de `codebase-memory-mcp`; se pararon
con `SIGTERM` y no quedaron procesos vivos. Nueva observacion local
2026-07-01: un intento directo de `codebase-memory-mcp/search_code` devolvio
`Transport closed` y dejo PIDs `74610` y `74987`; ambos se pararon con
`SIGTERM`. La conclusion no cambia: fuera del broker central opt-in, Codebase
puede dejar procesos vivos y no debe usarse por defecto en subagentes. Nueva
observacion local 2026-07-01: PID `2780041`, hijo de esta sesion Codex, quedo
idle con `_config.db` abierta y sin sockets tras la comprobacion MCP; se paro
con `SIGTERM`.

Revision adicional 2026-06-30 sobre BUG-030/037/040: se integra el tramo
remoto de metadata operacional OPES para goal-first. El ledger externo conserva
`current_phase`, `operational_reason` y `domain_counters`; el bloqueo de
heartbeat audio `already_submitted` persiste `current_phase=tts`,
`operational_reason=provider_timeout|running_no_recent_progress` y
`retry_from_phase=tts`; `GoalWorkSpec` prioriza input fields de fase/audio/
proveedor para que entren como `ContextRefs`; `autoprogramming/status` y
observe goal-first parcial proyectan `current_phase`, `retry_from_phase` y
`domain_counters` desde metadata durable. `close_superseded_by_local_evidence`
solo se recomienda si existe flag explicito y evidencia local/current durable,
no por una palabra suelta. Evidencia:
`TestExternalBridgeInputLedgerPersisteMetadataOperacionalV0`,
`TestMCPAutoprogrammingStatusExecutorV0GoalFirstBloqueadoProyectaMetadataOPESAudioV0`,
`TestMCPAutoprogrammingStatusExecutorV0GoalFirstBloqueadoCierraSupersededPorEvidenciaLocalV0`,
`TestMCPAutoprogrammingStatusExecutorV0GoalFirstSupersededSinEvidenciaLocalNoCierraV0`
y `TestMCPObserveAppDirectorGoalToolExecutorV0TimeoutSnapshotProyectaMetadataOPESAudioV0`.
Siguen `parcial` hasta cerrar smoke OPES temporal aislado, productor real de esa
metadata en run existente, heartbeat/progreso residente de proveedores largos y
sin tocar OPES productivo.

Revision adicional 2026-06-30 sobre BUG-030/037/040: se cierra el hueco de
proyeccion residente/director stats. `GoalWorkState` ahora expone
`current_phase`, `retry_from_phase`, `operational_reason` y `domain_counters`
tambien por `director.stats.goal`; `external_job` proyecta la misma metadata
desde el stack Codex; el bridge OPES residente lee esos campos desde `goal` o
`external_job`, conserva `retry_from_phase=*` en `next_actions` y usa
`operational_reason` como `supervision_stop_reason`. Evidencia:
`TestMCPDirectorStatsToolExecutorV0GoalFirstProyectaMetadataOperacionalV0`,
`TestCodexStackExternalJobStatsSourceV0GoalFirstProyectaMetadataOPESAudioV0` y
`TestOPESBridgeSupervisionFromDirectorStatsV0GoalFirstProyectaMetadataAudioV0`.
Revision adicional 2026-06-30 sobre BUG-030/037: queda cerrado el hueco de
contrato HTTP temporal entre OPES y Orquesta sin tocar OPES productivo. OPES
expone `POST /api/jobs/{id}/operational-metadata` solo con opt-in explicito
`OPES_OPERATIONAL_METADATA_API_ENABLED=true`; el caso de uso rechaza jobs que no
sean `generate_audio_asset` externos no terminales. Orquesta refresca
`GET /api/jobs/{id}` en `already_submitted`, da prioridad a `external_refs`
vivas sobre el payload inicial y conserva contadores a cero como
`pending_blocks=0`. Smoke temporal aislado:
`/tmp/orquesta-opes-audio-smoke.F7ZAV8/second_after_fix.json`, con OPES sqlite
loopback, Orquesta fake loopback, `provider_timeout`, `current_phase=tts`,
`generated_blocks=1`, `pending_blocks=0`, `submitted=0` y
`already_submitted=1`.
Revision adicional 2026-06-30 sobre BUG-030: la cola externa OPES
`scripts/run_orquesta_single_ref_queue.sh` emite metadata operacional para
audio en `POST /api/jobs/{id}/operational-metadata`: fase `tts`/`whisper`,
contadores, `provider_timeout` al agotar runtime/job y
`running_no_recent_progress` cuando no cambia la firma de progreso dentro de la
ventana configurada. Evidencia:
`scripts/test_run_orquesta_single_ref_queue_operational_metadata.sh`,
`TestRunOPESDrainOnceV0AudioAlreadySubmittedSinProgresoProyectaSinReenviarV0`
y el smoke temporal previo `second_after_fix.json`. BUG-030 queda cerrado.
Revision adicional 2026-06-30 sobre BUG-037: OPES ya materializa el orden
causal de publicacion de curso con tests (`prepare/capture -> tts -> whisper ->
rag -> qa`) y el runner temporal publica fases `tts`/`whisper` para jobs de
audio de tema y curso. Evidencia focal: `TestAdvanceProgramWorkflowCoursePublicationOrdenaPrepareTTSWhisperRAG`,
`TestUpdateJobOperationalMetadataPermiteAudioCursoExternoNoTerminal` y
`scripts/test_run_orquesta_single_ref_queue_operational_metadata.sh`. No se ha
tocado OPES productivo ni subido material; es cierre de contrato ejecutable e
integracion temporal aislada.

Revision adicional 2026-07-01 sobre BUG-088: con Orquesta `daa44d9`, ola OPES
17b tema 007, `autoprogramming/status` mejora el falso verde y publica
`state=live` con `running_live=1`, y luego `attention_required` con
`goal_first_blocked_with_partial_delivery` y
`evidence-ref-goal-materialized-checkpoint-detected`. El fallo de fondo sigue
abierto: solo se crea `coordinacion_wave17b/checkpoint_started.txt`, no se
materializan ampliado/resumen/validaciones/receipt, `observe_goal` devuelve
HTTP 504, SQLite queda `active|tokens_used=118258|time_used_seconds=166` y
`shutdown forced=true` evita el falso `ready` pero no cierra el backend Goal.
Evidencia en
`/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_responses/wave17b_t007_text/`
y apartado 33 de
`TAREA_OPES_ORQUESTA_GRUPO_B_GOAL_FIRST_OBSERVE_Y_QA_MINIMOS_2026-07-01.md`.

Nueva incidencia 2026-07-01 BUG-093: QA OPES/Orquesta puede producir falsos
verdes si valida solo extension y metanotas. En Grupo B Informatica, tras
limpieza de textos recuperados, 20/50 temas pasan minimo B, 18/50 pasan
extension + validadores oficiales (`no_exam_meta` y `no_author_notes`), pero
solo 4/50 pasan una QA estricta sin andamiaje interno. Se detectaron 215
hallazgos publicables de mapas mentales de estudio, calendarios `Dia 0`,
`reconstruye sin mirar`, trazabilidad B, derivaciones futuras, fuentes internas,
referencias a `.md`, SVG visibles, bloques de canon/maestro y mezcla de temas.
Se creo validador reproducible en OPES:
`opes-salidas/coordinacion_temarios/tools/validate_public_text_no_study_scaffolding.py`.
Evidencia:
`/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/09_validacion/informe_texto_publico_sin_andamiaje_interno.json`
y
`/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/09_validacion/informe_avance_estricto_grupo_b_informatica_2026-07-01.json`.
Accion esperada: el contrato de cierre debe exponer `extension_pass`,
`official_text_qa_pass` y `strict_editorial_qa_pass`; si falla andamiaje o
contaminacion cruzada, estado `pendiente_rework_editorial`, nunca `ready`.

Nueva incidencia OPES local 2026-07-02, relacionada con BUG-ORQ-20260701-093 y
BUG-ORQ-20260702-095: QA OPES/Orquesta sigue sin diferenciar
entre andamiaje superficial y contaminacion estructural de tema. En Grupo B
Informatica, seis auditorias paralelas sobre temas con extension suficiente
detectaron que 013, 014, 015, 017 y 046 no son cierres por limpieza puntual:
mezclan bloques de otros temas, referencias internas a canones/derivaciones,
fuentes de temas ajenos, menciones `.svg`/HTML, preguntas de recuperacion,
repaso espaciado y contenido que no responde al titulo. El tema 007 pudo
cerrarse solo tras cortar un bloque ajeno de administracion electronica y
ciberseguridad, reexpandir con contenido propio de firma/confianza y repetir
validaciones. Evidencia actual:
`/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/scripts/cleanup_tema_007_firma_confianza.py`,
`/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/09_validacion/informe_avance_estricto_grupo_b_informatica_2026-07-01.json`
y diagnosticos de subagentes 2026-07-02 sobre temas 007/013/014/015/017/046.
Accion esperada: el cierre automatico debe calcular coherencia titulo-cuerpo y
procedencia de bloques, no solo patrones textuales; si la extension depende de
contenido injertado de otros temas, el estado debe ser
`pendiente_rework_editorial_estructural` y Orquesta debe replanificar expansion
propia del titulo antes de RAG, HTML, tests o audios.

Seguimiento 2026-07-02: OPES tuvo que cerrar Grupo B Informatica mediante
fallback local documentado porque no habia runtime Orquesta activo y el trabajo
estructural seguia pendiente. Se reconstruyeron 013, 014, 015, 016, 017, 031,
034, 039, 046, 048 y 049 con scripts reproducibles en
`.../rework_profesional/00_control/scripts/`, se ampliaron los temas cortos y se
limpiaron residuos de texto publico. Resultado actual:
`extension_pass=50/50`, `official_text_qa_pass=50/50`,
`strict_text_qa_pass=50/50`, `strict_scaffolding_finding_count=0`. Evidencia:
`/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/09_validacion/informe_avance_estricto_grupo_b_informatica_2026-07-01.md`.
La incidencia sigue abierta para Orquesta: el resultado correcto se consiguio
por direccion externa y scripts OPES, no por un cierre autonomo del director.

Revision adicional 2026-07-01 sobre BUG-091: con Orquesta HEAD
`393e7bfdca1bce515180539059a71a8d3245643e` y árbol local modificado por otro
agente, OPES intentó lanzar wave18 tema 026. Readiness devolvió `ready=true`,
`status=running`, `startup_status=startup_ready` y se escribió `base_url.txt`
con `http://127.0.0.1:39989`; al enviar `POST /api/v0/external-work/run`,
`curl` devolvió conexión rechazada porque el proceso HTTP ya había muerto. El
state durable seguía con `status=running`, `pid=3592705` y
`startup_ready=true`. Quedaron residuos del backend Goal
`orquesta-goal-40dedd8d2b4bfd86` (`tmux`, `node codex app-server` y binario
`codex app-server`). Evidencia:
`/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_responses/wave18_t026_text/`
y runtime
`/home/alberto/Trabajo/orquesta/.orquesta-runtime/grupo-b-info-wave18-t026-text-20260701T183443Z`.
Accion esperada: si el HTTP muere tras readiness, actualizar state a
`crashed/stopped` con causa pública, limpiar backend Goal propio y aclarar la
semántica de `external_bridge_status=disabled` junto a
`external_bridge_ready=true` para trabajos `external-work/run`.

Nueva incidencia OPES local 2026-07-02, registrada en el inventario como
BUG-ORQ-20260702-120: cierre Grupo B Informatica siguio por
fallback local porque Orquesta estaba completamente inalcanzable (`server`
unreachable, `autoprogramming` unreachable, `server_processes=0`,
`codex_total_processes=0`) durante trabajo OPES que exige direccion Orquesta.
El cierre local detecto ademas contratos OPES no suficientemente ejecutables:
rebuild HTML retiro figuras del `visuals_manifest`, QA visual tecnica aceptaba
raster flojo sin hoja de contacto/semantica, RAG no llevaba `course_id` ni
`source_variant`, QA tests/tutor podia pasar bancos pobres y audio Edge TTS
necesita lotes reanudables con heartbeat/proveedor. Evidencia OPES:
`/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/INCIDENCIA_GRUPO_B_INFORMATICA_ORQUESTA_AUDIO_QA_2026-07-02.md`.
Accion esperada: Orquesta debe exponer estado `stopped/crashed/unreachable`
accionable para OPES, preservar visuales manifestados tras rebuild HTML,
validar contrato RAG completo, incorporar QA editorial tests/tutor y dirigir
TTS con progreso granular, timeout de proveedor y reanudacion sin duplicar MP3
validos.

Avance RAG/visual 2026-07-02:
`modulos/orquesta-opes-bridge/scripts/opes_validate_topic_package_v1.py`
incorpora `--final-package-dir` para validar `manifest_cierre.json`, RAG
canonico `rag/corpus/{chunks,summary}` con metadata de curso/variante y
coherencia de `visuals_manifest.json`/`visual_reuse_manifest.json` contra
`html_final`/`html_ampliado` tras rebuild. Evidencia:
`docs/incidencias/incidencia_orquesta_opes_finalpkg_rag_visual_2026-07-02.md`
y tests `TestOPESValidateTopicPackageV1FinalPackage*`. El bug agregado sigue
abierto por estado vivo Orquesta y QA editorial tests/tutor.

Avance audio/TTS 2026-07-02: `orquesta-domain-work` expresa el perfil neutral
`speech_synthesis` con heartbeat/timeout de proveedor o evidencia de
reanudacion/no duplicacion de salidas validas, y `orquesta-opes-bridge` exige el
required test OPES `audio_tts_resumable` para `generate_audio_asset` y
`completed_syllabus_package`. Evidencia:
`docs/incidencias/incidencia_orquesta_opes_audio_tts_reanudacion_2026-07-02.md`.
Nota historica previa al cierre de `BUG-ORQ-20260702-120`: en ese momento no
cerraba el bug padre porque seguian pendientes disponibilidad Orquesta y QA
tests/tutor. RAG/visual quedaba cubierto como avance parcial en la incidencia
`incidencia_orquesta_opes_finalpkg_rag_visual_2026-07-02.md`; el estado vigente
esta supersedido por el cierre posterior de disponibilidad state/status.

Avance QA tests/tutor 2026-07-02: `orquesta-opes-bridge` declara required tests
especificos `opes-question-bank-publicable-*` y
`opes-tutor-assets-publicable-*` para `generate_question_bank`,
`generate_tutor_assets` y `completed_syllabus_package`. El banco exige 50
preguntas por tema, 4 opciones, respuesta unica, distractores plausibles,
explicaciones tutor, informes estructural/dificultad y revision 100%
Codex/Gemini/Claude. El tutor exige paquete tutor/bots, guardas de alcance, QA
por tema/apartado y RAG canonico reconstruido desde HTML/tests/tutor limpios.
Evidencia: `TestOPESRequiredTestPolicyV0QuestionBankExigeQATripleYExplicacionesTutor`,
`TestOPESRequiredTestPolicyV0TutorAssetsExigeFuentesCanonicasYRAGReconstruido`,
`TestOPESRequiredTestPolicyV0FinalTemarioIncluyeQATestsYTutor` y
`go test -count=1 ./modulos/orquesta-opes-bridge`. Nota historica previa al
cierre de `BUG-ORQ-20260702-120`: en ese momento no cerraba el bug padre porque
seguian pendientes disponibilidad Orquesta y endurecimiento mecanico de
finalpkg/validator/preflight para tests/tutor; el estado vigente esta
supersedido por el cierre posterior de disponibilidad state/status.

Avance disponibilidad OPES 2026-07-02: el submit del bridge OPES normaliza los
fallos de transporte contra `/api/v0/external-work/run` como
`orquesta_unreachable`, `orquesta_unreachable_timeout` o
`orquesta_unreachable_cancelled`. `runOPESDrainOnceV0` proyecta esos errores con
`status=orquesta_unreachable`, `operational_reason` publico y acciones
`check_orquesta_readiness`, `start_orquesta_server_goal_first_app_server_tmux`,
`inspect_orquesta_server_status_or_statefile` y
`do_not_fallback_to_local_opes_without_operator_exception`. Evidencia:
`TestRunOPESDrainOnceV0BloqueaSubmitConOrquestaUnreachableV0` y
`go test -count=1 ./cmd/orquesta-server`. Nota historica previa al cierre de
`BUG-ORQ-20260702-120`: en ese momento no cerraba el bug padre porque seguian
pendientes la proyeccion `stopped/crashed` desde state/status y los validadores
mecanicos de tests/tutor; el estado vigente esta supersedido por el cierre
posterior de disponibilidad state/status.

Avance finalpkg tests/tutor 2026-07-02: el validador
`opes_validate_topic_package_v1.py --final-package-dir` comprueba que
`manifest_cierre.json` declare evidencias `question_bank_publicable`,
`tutor_assets_publicable` y `tutor`; valida el banco final con el mismo contrato
mecanico de 50 preguntas/4 opciones/respuesta/distractores; y exige paquete
tutor con `tutor_prompt.md`, `tutor_qa_report` y `tutor_scope_guard_report`.
Evidencia: `TestOPESValidateTopicPackageV1FinalPackageBloqueaTestsYTutorSinEvidencia`,
`python3 -m py_compile modulos/orquesta-opes-bridge/scripts/opes_validate_topic_package_v1.py`
y `go test -count=1 ./modulos/orquesta-opes-bridge`. Nota historica previa al
cierre de `BUG-ORQ-20260702-120`: en ese momento no cerraba el bug padre porque
seguian pendientes la proyeccion `stopped/crashed` desde state/status y el
preflight especifico de tutor antes de lanzar el job; el estado vigente esta
supersedido por el cierre posterior de disponibilidad state/status.

Avance finalpkg cierre stack 2026-07-02: el cierre agregado OPES en
`orquesta-app-codex-stack` ya no acepta `completed_syllabus_package` sin
evidencia final `tutor`, `qa_passes.question_bank_publicable`,
`qa_passes.tutor_assets_publicable` y sus `qa_report_refs` especificos. La
brecha era arquitectonica: `orquesta-opes-bridge` ya declaraba esos required
tests, pero el cierre real solo exigia QA editorial generica. Evidencia:
`docs/incidencias/incidencia_orquesta_opes_finalpkg_tutor_question_bank_qa_2026-07-02.md`,
`TestCodexStackOPESFinalPackageEvidenceIssueRefV0BloqueaSinTutorV0`,
`TestCodexStackOPESFinalPackageEvidenceIssueRefV0BloqueaQABancoYTutorV0`,
`TestCodexStackOPESFinalPackageEvidenceIssueRefV0AceptaTutorBancoYQAEstrictaV0`,
`go test -count=1 ./modulos/orquesta-app-codex-stack` y
`go test -count=1 ./modulos/orquesta-app-codex-stack ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-director ./modulos/orquesta-opes-topic-registry`.
No cierra BUG-058: siguen pendientes heartbeat/checkpoint durable por tema y
smoke OPES largo aislado.

Avance preflight tutor 2026-07-02: `runOPESDrainOnceV0` bloquea
`generate_tutor_assets` con `tutor_assets_source_context_required` antes de
postear a Orquesta si el payload no declara fuentes canonicas de tutor:
contenido/HTML aprobado, banco de preguntas y fuente/guarda de alcance del
tutor. Evidencia:
`TestOPESJobContextV0GenerateTutorAssetsExigeFuentesCanonicasV0`,
`TestRunOPESDrainOnceV0TutorAssetsSinFuentesCanonicasNoPosteaOrquestaV0` y
`go test -count=1 ./cmd/orquesta-server -run 'TestOPESJobContextV0GenerateTutorAssetsExigeFuentesCanonicasV0|TestRunOPESDrainOnceV0TutorAssetsSinFuentesCanonicasNoPosteaOrquestaV0|TestOPESJobContextV0GenerateQuestionBank'`.
Nota historica previa al cierre de `BUG-ORQ-20260702-120`: en ese momento no
cerraba el bug padre porque seguia pendiente la proyeccion `stopped/crashed`
desde state/status; el estado vigente esta supersedido por el cierre posterior
de disponibilidad state/status.

Avance disponibilidad state/status 2026-07-02: `orquesta-server` proyecta una
disponibilidad publica comun para readiness, `/api/v0/server/status` y
`orquesta-server status`. Si el proceso muere tras `startup_ready`, publica
`availability_status=crashed`, `availability_reason=server_crashed_after_readiness`,
evidencia statefile y acciones para inspeccionar, limpiar el backend Goal propio
si el owner coincide y reiniciar en modo `goal_first` con `app_server_tmux`. Si
el state durable esta `stopped`, publica `availability_status=stopped`,
`availability_reason=server_stopped` y acciones de arranque/readiness. Evidencia:
`TestMarkServerProcessStaleStateV0ExponeCausaTrasReadinessV0`,
`TestServerAvailabilityV0ExponeStoppedAccionableV0`,
`TestStatusServerCommandV0ReconciliaStatefileConPIDMuerto` y
`go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server`. Con este
avance se cierra BUG-ORQ-20260702-120.

Avance BUG-ORQ-20260701-058/066 2026-07-02: `update_topic_registry`
publica ahora un contrato estructurado de settlement por tema/fase:
`settlement_status`, `settlement_scope`, `settlement_reason`,
`settlement_contract`, `settlement_refs`, `settled_refs` y
`next_required_work_kinds`. Un texto con `OPESTopicQualityContractV0` completo
queda `settled_text` y apunta a derivados siguientes; QA fallida queda
`needs_rework` con rework director causal; paquete final con manifest completo
queda `settled_final`. Esto ataca la raiz de BUG-066 (goals que siguen
reescribiendo tras entrega suficiente) sin cambiar la herramienta OPES real ni
marcar curso completo por una fase textual. Tests:
`TestProduceOPESCausalJobsV0CreaActualizacionRegistroPorTema`,
`TestProduceOPESCausalJobsV0BloqueaRegistroPorQATemaFallidaV0`,
`TestProduceOPESCausalJobsV0NoBloqueaRegistroConQATemaCompletaV0` y
`TestProduceOPESCausalJobsV0PaqueteFinalCompleteConManifestCompatibleYQATernaLiberaRegistro`.
No cierra el bug padre: siguen pendientes heartbeat/checkpoint durable por tema,
vista unica por `run_ref`, reconciliacion tras cortes externos/manuales y smoke
real largo OPES.

Avance BUG-ORQ-20260701-058/066 2026-07-02 tarde:
`orquesta-opes-director` anade contrato de lifecycle goal-first por tema:
`goal_first_lifecycle_status`, `goal_first_lifecycle_reason`,
`goal_first_lifecycle_contract`, `goal_first_checkpoint_refs` y
`goal_first_heartbeat_refs`. Si una entrega OPES viene de `goal_first` y quiere
asentarse como texto o paquete final sin checkpoint durable, no queda
`settled_text`/`settled_final`: se publica
`pending_refs=goal-first-topic-checkpoint-required`,
`settlement_scope=goal_first_lifecycle` y rework causal del Director. Con
checkpoint presente, conserva el settlement normal. Ademas el manifest final
OPES del director ya exige banco de preguntas publicable y paquete tutor, no
solo la terna QA antigua. Incidencia:
`docs/incidencias/incidencia_orquesta_opes_goal_first_topic_lifecycle_checkpoint_2026-07-02.md`.
Tests: `TestProduceOPESCausalJobsV0GoalFirstTextoQAPassSinCheckpointNoAsientaTemaV0`,
`TestProduceOPESCausalJobsV0GoalFirstTextoQAPassConCheckpointAsientaTemaV0`,
`TestProduceOPESCausalJobsV0PaqueteFinalSinBancoYTutorNoLiberaRegistroV0` y
`go test -count=1 ./modulos/orquesta-opes-director ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-topic-registry ./modulos/orquesta-app-codex-stack`.
BUG-058/066 siguen abiertos hasta enforcement runtime fuerte y smoke OPES
temporal largo, pero ya no dependen de estado implicito para settlement por
tema goal-first.

Avance BUG-ORQ-20260701-058/066 2026-07-02 noche: el stack goal-first OPES ya
inyecta en `DomainWorkArtifactSubmission.PayloadFields` las senales que consume
el contrato anterior: `director_execution_mode=goal_first`,
`goal_first_status`, `goal_ref`, `external_goal_ref`,
`orquesta_goal_result_refs` y `goal_first_checkpoint_refs` cuando el resultado
trae checkpoint materializado. Esto evita que OPES dependa de que el agente
escriba manualmente lifecycle/checkpoint en el artefacto de dominio. Evidencia:
`TestCodexStackV0ExternalWorkGoalFirstCierraSecuenciaOPESDerivadosConReceiptsLedgerV0`
y `go test -count=1 ./modulos/orquesta-app-codex-stack ./modulos/orquesta-opes-director ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-topic-registry`.

Avance BUG-ORQ-20260701-058/066/075 2026-07-02 noche 2: OPES deja de depender
de un required-test generico por submit_artifact en derivados. El bridge genera
required tests especificos para toda la secuencia principal OPES por
`work_kind/artifact_type` (texto publicable, visual didactico, HTML, juegos,
manuales, revisiones, fuentes, supuestos, audio, tutor, tests y paquete final),
y el director de OPES proyecta gates de evidencia minima al registro de tema:
si falta evidencia estructurada, publica `pending_refs=required-evidence-*`,
`settlement_status=not_settled` y rework causal. Incidencia:
`docs/incidencias/incidencia_orquesta_opes_required_tests_y_settlement_por_work_kind_2026-07-02.md`.
Evidencia: `TestOPESRequiredTestPolicyV0SecuenciaCompletaTieneValidadoresEspecificos`,
`TestTopicRegistryRequiredEvidencePolicyV0CubreSecuenciaOPESCompletaV0`,
`TestProduceOPESCausalJobsV0BloqueaDerivadoOPESSinEvidenciaMinimaV0` y
`go test -count=1 ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-director ./modulos/orquesta-opes-topic-registry ./modulos/orquesta-app-codex-stack`.

Avance BUG-ORQ-20260701-075 2026-07-04 noche: la cobertura anterior deja de
ser solo policy/regla por artefacto y pasa a matriz ejecutiva de la secuencia
OPES. Para cada `work_kind` terminal de la secuencia principal que declara
entrega completa sin evidencia minima aceptada, `ProduceOPESCausalJobsV0`
emite `update_topic_registry` con
`proposed_status=pendiente_rework_evidencia_minima`,
`operational_status=needs_rework`,
`settlement_status=needs_rework`,
`settlement_scope=required_evidence` y
`settlement_reason=required_evidence_missing`; ademas crea follow-up
`review_director_consolidation` con `followup_ref` causal,
`rework_reason=required_evidence_missing`,
`required_evidence_missing_refs`, `publication_status` no publicable y
`recommended_action=review_required_evidence`. Evidencia:
`TestProduceOPESCausalJobsV0BloqueaSecuenciaOPESCompletaSinEvidenciaMinimaV0`,
`TestProduceOPESCausalJobsV0BloqueaDerivadoOPESSinEvidenciaMinimaV0`,
`TestTopicRegistryRequiredEvidencePolicyV0CubreSecuenciaOPESCompletaV0` y
`go test -count=1 ./modulos/orquesta-opes-director ./modulos/orquesta-opes-bridge`.

Avance BUG-ORQ-20260701-058/075 2026-07-02 noche 3:
`OPESTopicQualityContractV0` detecta anclas Markdown visibles `{#...}` en el
texto publico como `opes_public_text_structural_contamination`. Esto cubre el
fallo determinista observado en artefactos parciales OPES sin convertir texto
libre recuperable en veto generico: solo una marca estructural de exportacion
visible dispara rework editorial. Evidencia:
`TestValidateOPESTopicQualityContractV0DetectaAnclasMarkdownPublicasV0` y
`go test -count=1 ./modulos/orquesta-opes-director`.

Avance BUG-ORQ-20260701-058/075 2026-07-02 noche 4:
`OPESTopicQualityContractV0` detecta corrupciones ortograficas concretas
observadas tras correccion masiva (`órgaños`, `confíanza`, `instituciónal`,
`propuestá`) como texto publico corrupto antes de HTML/audio/paquete final. La
lista es cerrada y conserva artefactos como recuperables para rework, no como
descarte global por texto libre. Evidencia:
`TestValidateOPESTopicQualityContractV0DetectaCorreccionOrtograficaCorruptaV0`
y `go test -count=1 ./modulos/orquesta-opes-director`.

Avance BUG-ORQ-20260701-058/075 2026-07-02 noche 5:
`OPESTopicQualityContractV0` detecta tablas Markdown colapsadas dentro de un
encabezado (`## ... | ... | ...`) como contaminacion estructural del texto
publico. Es una marca formal de exportacion defectuosa, no una heuristica de
contenido, y conserva el artefacto como recuperable para rework. Evidencia:
`TestValidateOPESTopicQualityContractV0DetectaTablaColapsadaEnEncabezadoV0` y
`go test -count=1 ./modulos/orquesta-opes-director`.

Avance BUG-ORQ-20260701-077 2026-07-02 noche 6: los README operativos de
`modulos/orquesta-server` y `modulos/orquesta-opes-bridge` dejan de recomendar
`go run ./cmd/orquesta-server run` como receta de operador y pasan a
`orquesta-server start`/`orquesta-server stop`, conservando `app_server_tmux` y
apagado gobernado. Evidencia:
`TestReadmesOperativosNoRecomiendanRuntimeManualV0` y
`go test -count=1 ./cmd/orquesta-server -run 'Test(ReadmesOperativosNoRecomiendanRuntimeManual|UsoActualAppOrquestaRecomiendaServidorGestionado|ArrancarCodexModuloNoRecomiendaRuntimeManual)V0'`.

Avance BUG-ORQ-20260701-077 2026-07-02 noche 8: el launcher operador
`scripts/opes_a1_finalpkg_registry_launcher.py` deja de caer a
`http://127.0.0.1:8787` por defecto; resuelve Orquesta por
`ORQUESTA_SERVER_URL`, compatibilidad `ORQUESTA_BASE_URL` o
`ORQUESTA_RUNTIME_DIR/base_url.txt`, y si va a ejecutar efectos exige
`--orquesta-base-url` explicito cuando no hay endpoint gestionado. Evidencia:
`TestLauncherOPESA1NoUsaPuertoHistoricoPorDefectoV0`.

Avance BUG-ORQ-20260701-077 2026-07-03: los smokes OPES REST directos
`scripts/smoke_opes_domain_work_real.sh` y
`scripts/smoke_opes_visual_asset_real.sh` dejan de asumir un puerto Orquesta
historico por defecto; ahora resuelven endpoint por `ORQUESTA_SERVER_URL`,
compatibilidad `ORQUESTA_BASE_URL` o `ORQUESTA_RUNTIME_DIR/base_url.txt`, y
bloquean antes de tocar OPES temporal si no hay endpoint gestionado. Evidencia:
`TestSmokesOPESRESTDirectosUsanEndpointOrquestaGestionadoV0`.

Avance BUG-ORQ-20260701-077 2026-07-03 tarde: los wrappers generales
`scripts/inicio_agente.sh` y `scripts/orquesta_status_now.sh` dejan de caer al
puerto historico `127.0.0.1:8787`; ambos consumen el resolvedor comun de
endpoint gestionado (`ORQUESTA_SERVER_URL`, compatibilidad `ORQUESTA_BASE_URL`
o `ORQUESTA_RUNTIME_DIR/base_url.txt`) y bloquean con mensaje publico si no hay
servidor identificable. Evidencia: `TestInicioAgenteNoRecomiendaRuntimeManualV0`,
`TestOrquestaStatusNowUsaEndpointGestionadoV0`.

Avance BUG-ORQ-20260701-077 2026-07-03 tarde 2: los wrappers OPES largos
`scripts/smoke_opes_plan_temario_operadores.sh` y
`scripts/smoke_opes_derivatives_rest.sh` ya no exigen solo
`ORQUESTA_BASE_URL` para crear runs; aceptan el resolvedor comun
`ORQUESTA_SERVER_URL`/`ORQUESTA_BASE_URL`/`ORQUESTA_RUNTIME_DIR/base_url.txt` y
mantienen el bloqueo antes de efectos cuando no hay endpoint Orquesta
gestionado. Evidencia:
`TestSmokesOPESLargosAceptanEndpointOrquestaGestionadoV0` y
`TestSmokeOPESDerivativesRESTWrapperPreflightRealBloqueaSinOrquestaBaseURLV0`.

Avance BUG-ORQ-20260701-077 2026-07-03 tarde 3: el bridge OPES residente en
`cmd/orquesta-server` deja de exigir solo `ORQUESTA_BASE_URL` cuando no puede
derivar estado; resuelve endpoint por `ORQUESTA_SERVER_URL`, compatibilidad
`ORQUESTA_BASE_URL` o `ORQUESTA_RUNTIME_DIR/base_url.txt` antes del fallback de
arranque/estado. Evidencia:
`TestOPESBridgeLoopConfigUsaEndpointGestionadoV0`,
`TestOPESBridgeLoopConfigUsaBaseURLDeRuntimeGestionadoV0` y guardas del
registry de env del servidor.

Avance BUG-ORQ-20260701-077 2026-07-03 tarde 4: la receta publica de pruebas de
`modulos/orquesta-server` deja de recomendar `go run ./cmd/orquesta-server run`
con curls directos a `127.0.0.1:8787`; orienta a `orquesta-server start/status/stop`
y a consumir la URL gestionada desde `ORQUESTA_RUNTIME_DIR/base_url.txt` o
`ORQUESTA_SERVER_URL`. Evidencia:
`TestPruebasServidorNoRecomiendaPuertoHistoricoV0`.

Avance BUG-ORQ-20260701-077 2026-07-03 tarde 5: el README de
`modulos/orquesta-core-workflow` deja de indicar `go run ./cmd/orquesta-server run`
como ruta operativa normal; apunta a `orquesta-server start/status`, endpoint
gestionado y reserva `go run` para harnesses aislados con cleanup explicito.
Evidencia: `TestReadmesOperativosNoRecomiendanRuntimeManualV0`.

Avance BUG-ORQ-20260701-077 2026-07-03 tarde 6: el runbook activo de pruebas
locales `docs/runbooks/pruebas_locales_orquesta_2026-05-25.md` deja de fijar
`http://127.0.0.1:8787` en readiness y curls; exige resolver endpoint por
`orquesta-server status --json`, `ORQUESTA_SERVER_URL` o
`ORQUESTA_RUNTIME_DIR/base_url.txt`. Evidencia:
`TestRunbookPruebasLocalesUsaEndpointGestionadoV0`.

Avance BUG-ORQ-20260701-077 2026-07-03 tarde 7: el runbook
`docs/runbooks/autoprogramacion_cli_2026-05-23.md` deja de fijar
`--server-url http://127.0.0.1:8787`; documenta resolucion previa por
`orquesta-server status --json`, `ORQUESTA_SERVER_URL` o
`ORQUESTA_RUNTIME_DIR/base_url.txt` y usa `--server-url "$ORQUESTA_SERVER_URL"`.
Evidencia: `TestRunbookAutoprogramacionCLIUsaEndpointGestionadoV0`.

Avance BUG-ORQ-20260701-077 2026-07-03 tarde 8: el manual vigente
`docs/uso_actual_app_orquesta.md` deja de enseñar el comando literal de runtime
manual y de asumir `127.0.0.1:8787`; la seccion actual apunta a
`orquesta-server start/stop/status` y a `ORQUESTA_RUNTIME_DIR/base_url.txt`.
Evidencia: `TestUsoActualAppOrquestaRecomiendaServidorGestionadoV0`.

Avance BUG-ORQ-20260701-077 2026-07-03 tarde 9: el runbook OPES vigente de
`plan_temario` deja de arrancar Orquesta con el runtime manual en bloques con
efectos y usa `orquesta-server start` con backend `app_server_tmux`; la guarda
OPES bloquea nuevos bloques efectivos con `go run ./cmd/orquesta-server run`.
Evidencia: `TestOPESOperationalDocsGuardV0`.

Avance BUG-ORQ-20260701-077 2026-07-03 tarde 10: el runbook del panel ops deja
de publicar `http://127.0.0.1:8787/ops` como URL fija y exige resolver el
endpoint gestionado por `orquesta-server status --json`, `ORQUESTA_SERVER_URL`
o `ORQUESTA_RUNTIME_DIR/base_url.txt`. Evidencia:
`TestRunbookPanelOpsUsaEndpointGestionadoV0`.

Avance BUG-ORQ-20260701-077 2026-07-03 tarde 11: el runbook OPES real de
`domain_work` deja de arrancar Orquesta con `go run ./cmd/orquesta-server run`;
usa `orquesta-server start` con backend `app_server_tmux` y exige resolver el
endpoint gestionado antes de consultar o drenar. La guarda OPES cubre tambien
este runbook y bloquea cualquier bloque OPES que vuelva a publicar el runtime
manual. Evidencia: `TestOPESOperationalDocsGuardV0`.

Avance BUG-ORQ-20260701-077 2026-07-03 tarde 12: la guarda de wrappers que
arrancan `orquesta-server run` temporal en background ya no depende de una
ventana corta de cuatro lineas; sigue continuaciones multilinea hasta el cierre
del comando y distingue el `&` de background de redirecciones como `2>&1`. Esto
evita que un wrapper largo fuera de `smoke_common.sh` escape al requisito de
shutdown comun con `runtime_dir`. Evidencia:
`TestScriptRunsTemporaryOrquestaServerCommandV0DetectaBackgroundMultilineaLargoV0`
y `TestScriptRunsTemporaryOrquestaServerCommandV0IgnoraForegroundMultilineaV0`.

Avance BUG-ORQ-20260701-077 2026-07-03 tarde 13: el handoff vigente
`docs/handoff_terminar_orquesta_2026-06-19.md` deja de publicar como
reproduccion operativa un servidor en primer plano con puerto fijo y runtime
manual; usa `orquesta-server start/status/stop`, consume
`ORQUESTA_RUNTIME_DIR/base_url.txt` y limita `orquesta-server run` a harnesses
aislados con `smoke_shutdown_orquesta_server`, `runtime_dir` y
`cleanup_goal_backends`. Evidencia:
`TestHandoffTerminarOrquestaUsaServidorGestionadoV0`.

Avance BUG-ORQ-20260701-077 2026-07-03 tarde 14: la matriz vigente de smokes
para `OPES-DER-RESTO` deja de recomendar `go run ./cmd/orquesta-server run`
como servidor residente goal-first; documenta `orquesta-server start`,
`ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux`,
`ORQUESTA_RUNTIME_DIR/base_url.txt`, `ORQUESTA_SERVER_URL` para el wrapper y
`orquesta-server stop`. Evidencia:
`TestMatrizOPESDerivadosUsaServidorGestionadoV0`.

Avance BUG-ORQ-20260701-077 2026-07-03 tarde 15: el handoff
`docs/handoff_orquesta_goal_first_parada_2026-06-26.md` deja de pedir
reiniciar el servidor web local por puerto historico `127.0.0.1:8787`; orienta
a `orquesta-server start/status/stop`, `ORQUESTA_SERVER_URL` y
`ORQUESTA_RUNTIME_DIR/base_url.txt` sin tocar servidores OPES. Evidencia:
`TestHandoffGoalFirstParadaUsaServidorGestionadoV0`.

Avance BUG-ORQ-20260701-077 2026-07-03 tarde 16: el runbook OPES historico de
`plan_temario` deja de presentar `cmd/orquesta-server run` activo como supuesto
operativo; habla de servidor residente gestionado con `orquesta-server start`,
endpoint por `ORQUESTA_SERVER_URL`/`ORQUESTA_RUNTIME_DIR/base_url.txt` y
mantiene `legacy_director_loop` solo como opt-in explicito de replay. Evidencia:
`TestRunbookOPESPlanTemarioUsaServidorGestionadoV0`.

Avance BUG-ORQ-20260701-077 2026-07-03 tarde 17: las menciones residuales a
runtime manual o puerto historico en Markdown quedan limitadas por guarda a
incidencias historicas, backlog tecnico o harnesses aislados explicitos; nuevos
docs operativos que vuelvan a publicar `go run ./cmd/orquesta-server run`,
`cmd/orquesta-server run` o `127.0.0.1:8787` sin esa clasificacion fallan en
test. Evidencia:
`TestOperationalDocsRuntimeManualMentionsAreHistoricalOrHarnessV0`.

Avance BUG-ORQ-20260701-077 2026-07-03 tarde 19: la guarda de wrappers fija con
regresion explicita el caso `ORQUESTA_SERVER_ADDR` + `server_pid="$!"`, para que
un script que arranque servidor temporal por PID capturado siga exigiendo
`smoke_shutdown_orquesta_server` y cleanup gobernado. Evidencia:
`TestScriptStartsTemporaryOrquestaServerV0DetectaPIDConAddrGestionadoV0`.

Avance BUG-ORQ-20260701-077 2026-07-03 tarde 20: el handoff de cierre de sesion
`docs/handoff_cierre_sesion_orquesta_2026-07-02.md` deja de listar BUG-077 como
abierto tras integrar la guarda de wrappers; conserva la evidencia del detector
`ORQUESTA_SERVER_ADDR` + `server_pid="$!"` y una guarda documental impide
reabrirlo en ese handoff sin evidencia nueva. Evidencia:
`TestHandoffCierreSesionNoReabreBUG077V0`.

Avance BUG-ORQ-20260701-077 2026-07-03 tarde 21: los scripts que resuelven el
endpoint gestionado mediante `smoke_require_orquesta_base_url` o
`smoke_orquesta_base_url_from_env_or_runtime` quedan cubiertos por una guarda
que exige cargar `scripts/lib/smoke_common.sh`; asi se evita reintroducir
wrappers que documenten `ORQUESTA_RUNTIME_DIR/base_url.txt` pero fallen en
ejecucion o vuelvan a rutas manuales. Evidencia:
`TestScriptsConEndpointGestionadoCarganSmokeCommonV0`.

Avance BUG-ORQ-20260701-077 2026-07-03 tarde 22: el resolvedor comun de
endpoint Orquesta queda fijado por guarda sin fallback al puerto historico
`8787`: primero `ORQUESTA_SERVER_URL`, luego compatibilidad
`ORQUESTA_BASE_URL`, despues `ORQUESTA_RUNTIME_DIR/base_url.txt` y, si no hay
endpoint gestionado, bloqueo explicito. Evidencia:
`TestSmokeCommonEndpointGestionadoSinPuertoHistoricoV0`.

Avance BUG-ORQ-20260701-077 2026-07-03 tarde 23: una guarda generica recorre
`scripts/*.sh` y falla ante cualquier reintroduccion del puerto historico
Orquesta `127.0.0.1:8787`/`localhost:8787`. Esto complementa las guardas de
resolvedor comun y de scripts concretos para que nuevos wrappers no vuelvan a
asumir un endpoint local fijo. Evidencia:
`TestScriptsNoAsumenPuertoOrquestaHistoricoV0`.

Avance BUG-ORQ-20260701-077 2026-07-03 tarde 24: la guarda de shutdown HTTP
directo en scripts deja de depender de una ventana fija alrededor de
`/api/v0/server/shutdown`; ahora reconstruye el comando `curl` multilinea
completo y exige `cleanup_goal_backends` aunque el payload quede lejos del
endpoint. Evidencia:
`TestScriptShutdownCurlCommandsV0DetectaCleanupLejanoV0` y
`TestScriptsConShutdownDirectoPidenCleanupGoalBackendsV0`.

Avance BUG-ORQ-20260701-077 2026-07-03 tarde 25: el handoff vigente de cierre
de sesion queda fijado contra regresiones de runtime manual: debe conservar
`orquesta-server stop` como parada gestionada, seguir declarando
`smoke_shutdown_orquesta_server` para scripts temporales y no reintroducir
`go run ./cmd/orquesta-server run` ni el puerto historico `8787`. Evidencia:
`TestHandoffCierreSesionNoReabreBUG077V0`.

Avance BUG-ORQ-20260701-077 2026-07-03 tarde 26: el handoff vigente de cierre
tambien queda fijado a la guarda delegada
`TestSmokeOPESExternalWorkAgentRealUsaShutdownDelegadoConRuntimeDirV0`: el
wrapper OPES external-work debe conservar `trap smoke_cleanup EXIT`,
`scripts/lib/opes_agent_smoke_ops.sh`, `RUNTIME_DIR` y
`smoke_shutdown_orquesta_server`, evitando que un cleanup delegado reabra el
residual de app-server/base_url.

Avance BUG-ORQ-20260701-077 2026-07-03 tarde 27: la guarda del launcher Python
OPES A1 queda ampliada para fijar la misma resolucion de endpoint gestionado que
los wrappers shell: `ORQUESTA_SERVER_URL`, compatibilidad `ORQUESTA_BASE_URL`,
`ORQUESTA_RUNTIME_DIR/base_url.txt` y bloqueo sin fallback al puerto historico
`8787`. Evidencia: `TestLauncherOPESA1NoUsaPuertoHistoricoPorDefectoV0`.

Avance BUG-ORQ-20260701-077 2026-07-03 tarde 28: la guarda del handoff de cierre
permite conservar `777e027c` solo como corte historico observado y exige que la
reanudacion declare `fetch/rebase` antes de tratar cualquier HEAD como vigente;
ademas bloquea que el documento vuelva a publicar un `HEAD remoto/origin
vigente` o una ruta de reanudacion sin `fetch/rebase`. Evidencia:
`TestHandoffCierreSesionNoReabreBUG077V0`.

Avance BUG-ORQ-20260701-077 2026-07-03 tarde 29: la guarda de shutdown HTTP
directo en scripts detecta tambien `curl --request POST` y `curl --request=POST`
contra `/api/v0/server/shutdown`, no solo `-X POST`, y sigue exigiendo
`cleanup_goal_backends`. Evidencia:
`TestScriptShutdownCurlCommandsV0DetectaRequestPostLargoV0`.

Avance BUG-ORQ-20260701-077 2026-07-03 tarde 30: el contrato publico de
`/api/v0/domain-work/status` queda declarado como bridge read-only propio sobre
`autoprogramming/status` y `queue/global-status`, y la fachada conserva
`no_action_reason` en filas vivas no accionables para que paneles OPES/domain
work no muestren estados mudos durante espera, shutdown o reconciliacion.
Evidencia: `TestMCPDomainWorkStatusHTTPHandlerV0ConservaRazonSinAccionV0`.

Avance BUG-ORQ-20260701-065/088 2026-07-03 tarde 26: la reconciliacion
automatica de `goal_backend_missing_after_external_cleanup` queda fijada como
responsabilidad opt-in del supervisor residente; una llamada no residente a
`runs.supervisor` no consulta `DirectorStats`, no muta el `GoalWorkState`
running, no marca `blocked` y no lanza rework tras cleanup externo. Evidencia:
`TestRunSupervisorGoalFirstNoResidentNoReconciliaBackendMissingTrasCleanupExternoV0`.

Avance BUG-ORQ-20260701-065 2026-07-03 tarde 27: la evidencia
`evidence-ref-autoprogramming-goal-backend-missing-after-external-cleanup` queda
acotada a acciones ejecutivas de `runs/control`: `stop` y `cancel` pueden
reconciliar cleanup externo, pero `pause` y `resume` solo conservan la evidencia
en el comando y no consultan cierre terminal, no completan run-control, no
marcan el `GoalWorkState` como `blocked` y no generan rework implicito.
Evidencia:
`TestMCPRunControlExecutorV0PauseNoReconciliaExternalCleanupAunqueTraigaEvidencia`
y
`TestMCPRunControlExecutorV0ResumeNoReconciliaExternalCleanupAunqueTraigaEvidencia`.

Avance BUG-ORQ-20260701-065 2026-07-03 tarde 28: la misma frontera de acciones
no terminales queda cubierta en adaptadores: HTTP y transporte MCP nativo
propagan `pause`/`resume` con evidencia de cleanup externo al puerto de
run-control, pero no completan run-control, no publican `replan_narrow_context`,
no generan diagnostico `goal_state_terminal_reconciled_after_external_cleanup` y
no mutan el `GoalWorkState` running. Evidencia:
`TestMCPRunControlHTTPHandlerV0PauseNoReconciliaExternalCleanupSinForceV0`,
`TestMCPRunControlHTTPHandlerV0ResumeNoReconciliaExternalCleanupSinForceV0`,
`TestMCPTransportV0RunControlPauseNoReconciliaExternalCleanupSinForce` y
`TestMCPTransportV0RunControlResumeNoReconciliaExternalCleanupSinForce`.

Avance BUG-ORQ-20260701-065 2026-07-03 tarde 29: con un `RunMemoryStoreV0` real,
`pause` y `resume` con evidencia de cleanup externo conservan esa evidencia en
la respuesta publica y en el estado run-control durable, sin recomendar
`replan_narrow_context` ni emitir diagnostico de reconciliacion terminal.
Evidencia:
`TestMCPRunControlExecutorV0PauseResumeExternalCleanupConservanEvidenciaEnResultadoV0`.

Avance BUG-ORQ-20260701-065 2026-07-03 tarde 30: el adaptador file-based de
run-control conserva de forma durable la evidencia de cleanup externo en
`pause` y `resume`, normalizada y sin duplicados, y la recupera tras recrear el
store desde disco. Evidencia:
`TestRunFileStorePauseResumePersisteEvidenciaCleanupExternoV0`.

Avance BUG-ORQ-20260701-065 2026-07-03 tarde 31: el adaptador de memoria conserva
la misma evidencia de cleanup externo en `pause` y `resume`, normalizada y sin
duplicados, manteniendo paridad con el conector file-based para pruebas y
composiciones en memoria. Evidencia:
`TestRunMemoryStorePauseResumeConservaEvidenciaCleanupExternoV0`.

Avance BUG-ORQ-20260701-065 2026-07-03 tarde 32: la matriz local de pruebas de
los conectores `orquesta-run-memory` y `orquesta-run-file` documenta ya que
`pause`/`resume` conservan la evidencia
`evidence-ref-autoprogramming-goal-backend-missing-after-external-cleanup`, de
modo que el contrato queda visible junto a las pruebas focales de cada
adaptador.

Avance BUG-ORQ-20260701-085 2026-07-03 tarde 22: `efficiency_summary` conserva
ahora tambien las acciones compactas de launch write-set: si el goal queda
invalid por `codex_app_server_write_set_requires_workspace_write`, publica
`state=attention_required`, razon causal y
`configure_goal_backend_workspace_write:run:<run_ref>`; si falla el contrato
`allowed_write_set`, publica `repair_goal_write_set_contract:run:<run_ref>`.
Esto evita que consumidores compactos vean solo `unavailable` o un bloqueo
generico cuando la causa durable ya esta en `stale_running`. Evidencia:
`TestMCPAutoprogrammingStatusExecutorV0WriteSetReadOnlyPideConfigurarSandboxV0`
y `TestMCPAutoprogrammingStatusExecutorV0WriteSetGuardContractPideRepairPacketV0`.

Avance BUG-ORQ-20260701-085 2026-07-03 tarde 23: `ops_snapshot` de
`autoprogramming/status` tambien conserva los bloqueos write-set como decision
con `attention=true`, `run_ref`, `reason_code` especifico y evidencias del
launch receipt. Antes podia caer a `idle/no_safe_autoprogramming_action` porque
solo reconocia `goal_first_blocked` generico. Evidencia:
`TestMCPAutoprogrammingStatusExecutorV0WriteSetReadOnlyPideConfigurarSandboxV0`
y `TestMCPAutoprogrammingStatusExecutorV0WriteSetGuardContractPideRepairPacketV0`.

Avance BUG-ORQ-20260701-075/079/085 2026-07-03 tarde 24: la misma decision
autonoma de `ops_snapshot` ya no depende de una lista corta de codigos; cualquier
`stale_running` con `recommended_action` especifica y severidad `blocked` o
`warning` se proyecta como decision con atencion, `run_ref`, `reason_code`
causal y evidencias. Cubre QA publica fallida, artefactos/receipt/write-set
recuperables y salida gigante saneada sin convertir senales `info` de espera en
replan. Evidencia:
`TestMCPAutoprogrammingStatusExecutorV0QAFailedPublicTextPideReworkV0` y
`TestMCPAutoprogrammingStatusExecutorV0OutputGiganteSaneadoPideContextoAcotadoV0`.

Avance BUG-ORQ-20260701-073/079 2026-07-03 tarde 25: la regla anterior queda
fijada tambien como prueba unitaria de decision autonoma: severidad `blocked` y
`warning` con accion recomendada producen atencion, pero una senal `info` como
`active_timeout_checkpoint_recent`/esperar checkpoint no se convierte en replan
ni en atencion artificial. Evidencia:
`TestDirectorOpsDecisionFromAutoprogrammingActionableRunV0RespetaSeveridadV0`.

Avance BUG-ORQ-20260701-065/088 2026-07-03 tarde 4: `runs/control` ya no deja la
evidencia de `control_not_propagated_to_goal_backend` solo dentro de
`diagnostics`; cuando el backend Goal sigue activo tras stop/cancel, el resultado
publico de primer nivel conserva `evidence_refs` compactas del backend activo y
del goal observado para que clientes HTTP/MCP compactos no pierdan la causa del
409 ni publiquen un terminal falso. Evidencia:
`TestMCPRunControlExecutorV0StopForcedNoPublicaStoppedSiGoalBackendSigueActive`
y `TestMCPRunControlHTTPHandlerV0BackendGoalActivoEsConflictV0`.

Avance BUG-ORQ-20260701-065/076 2026-07-03 tarde 5: el cliente
`orquesta-server stop` conserva hasta dos refs compactas de `active_work` en el
error publico `shutdown_not_ready`, de modo que un timeout o una respuesta
`ready` bloqueada por backend Goal vivo no pierde la causa concreta al salir por
CLI/wrapper. Las refs se limitan y se filtran por formato compacto para no
publicar paths ni detalle runtime. Evidencia:
`TestShutdownClientNotReadyErrorV0IncluyeRefsActiveWorkCompactas`,
`TestShutdownClientReadyV0BloqueaActiveWorkPersistido` y
`TestRequestServerShutdownV0ReadyNoSaltaActiveWorksEstructuradosV0`.

Avance BUG-ORQ-20260701-065/076 2026-07-03 tarde 6: la politica de escalado de
`orquesta-server stop --force` clasifica `shutdown_not_ready` con
`active_work>0` o `active_work_refs` como conflicto vivo aunque no haya un
`/status` rico disponible, evitando que un wrapper convierta ese handoff parcial
en señal cooperativa prematura. Evidencia:
`TestShutdownRequestErrorAllowsSignalV0SoloConTimeoutYEstadoDrenado`.

Avance BUG-ORQ-20260701-065/076 2026-07-03 tarde 7: la tool MCP
`orquesta.server.shutdown.v0` eleva las evidencias de `active_works` al campo
`evidence_refs` superior del resultado, manteniendo tambien el detalle por
trabajo activo. Asi clientes MCP compactos que no recorren `active_works` no
pierden la causa del bloqueo por Goal/backend vivo. Evidencia:
`TestMCPServerShutdownToolExecutorV0ExponeGoalsActivos`.

Avance BUG-ORQ-20260701-065/076 2026-07-03 tarde 8: `orquesta-server stop
--force` ya no depende solo del `/status` rico para no escalar; si el propio
error `shutdown_not_ready` conserva `agents_in_flight`, `checkpoints`,
`checkpoint_agents` o `runs` pendientes, el cliente lo trata como trabajo vivo
y no envia senal cooperativa aunque el status posterior llegue vacio. Evidencia:
`TestShutdownRequestErrorAllowsSignalV0SoloConTimeoutYEstadoDrenado`.

Avance BUG-ORQ-20260701-065/076 2026-07-03 tarde 9: el cliente ya no acepta
`shutdown_ready=true` como atajo suficiente si el resultado HTTP o el status
publico todavia conservan agentes en vuelo, checkpoints, agentes de checkpoint
o async work pendientes. `shutdown_ready` solo permite senal cuando todo el
trabajo bloqueante esta drenado. Evidencia:
`TestShutdownClientReadyV0NoSaltaTrabajoPendienteAunqueReadyV0`.

Avance BUG-ORQ-20260701-065/076 2026-07-03 tarde 10: la proyeccion HTTP del
runtime servidor tampoco acepta `shutdown_ready=true` si la respuesta conserva
`active_work_count` o `active_works`; normaliza `active_works` a contador de
trabajos, cambia el estado a `stop_pending` y mantiene el shutdown congelado
para que el servidor no publique ready ni dispare cierre con backend Goal vivo.
Evidencia:
`TestShutdownProjectionFromHTTPV0ReadyConActiveWorkQuedaStopPendingV0`.

Avance BUG-ORQ-20260701-065/076 2026-07-03 tarde 11: la misma proyeccion HTTP
ingiere tambien `active_work_refs` compactas cuando un adaptador no devuelve
`active_works` estructurados; esas refs bastan para convertir un
`shutdown_ready=true` en `stop_pending` y preservar la causa compacta del
bloqueo. Evidencia:
`TestShutdownProjectionFromHTTPV0ReadyConActiveWorkRefsQuedaStopPendingV0`.

Avance BUG-ORQ-20260701-065/076 2026-07-03 tarde 12: las
`active_work_refs` directas se normalizan como refs publicas compactas antes de
persistir estado; si una entrada parece path o detalle runtime, se conserva la
causa como `shutdown-active-work-ref-redacted` sin publicar la ruta original.
Evidencia:
`TestShutdownProjectionFromHTTPV0ReadyConActiveWorkRefsQuedaStopPendingV0`.

Avance BUG-ORQ-20260701-065/076 2026-07-03 tarde 13: el cliente
`orquesta-server stop` ingiere `shutdown_async_work_active` del resultado HTTP
de shutdown y lo conserva tambien al convertir `/status` a resultado interno.
Un `shutdown_ready=true` con async work pendiente ya no permite senal local y el
error publico `shutdown_not_ready` conserva `async_work=N` para que
`--force` no lo trate como transporte drenado. Evidencia:
`TestShutdownClientReadyV0NoSaltaTrabajoPendienteAunqueReadyV0` y
`TestShutdownRequestErrorAllowsSignalV0SoloConTimeoutYEstadoDrenado`.

Avance BUG-ORQ-20260701-065/076 2026-07-03 tarde 14: la proyeccion HTTP del
runtime servidor tambien ingiere `shutdown_async_work_active` cuando el handler
de shutdown lo devuelve; si llega junto a `shutdown_ready=true`, lo convierte en
`stop_pending`, mantiene congelado el supervisor y publica el contador en
`/status`. Evidencia:
`TestRuntimeV0ServerShutdownConservaAsyncWorkActiveV0` y
`TestShutdownProjectionFromHTTPV0ReadySinConfirmacionesQuedaStopPendingV0`.

Avance BUG-ORQ-20260701-065/076 2026-07-03 tarde 15: si una respuesta parcial
de shutdown omite `shutdown_async_work_active`, la proyeccion HTTP conserva el
contador ya persistido en el snapshot previo del runtime y no publica un estado
drenado por falta del campo en el body. Evidencia:
`TestRuntimeV0ServerShutdownSnapshotPrevioConservaAsyncWorkActiveV0`.

Avance BUG-ORQ-20260701-065/076 2026-07-03 tarde 16: si el handler de shutdown
responde `shutdown_ready=true` pero omite el active work observado justo antes
por `ShutdownSnapshotPortV0`, el runtime ya no borra ese snapshot ni dispara
ready; normaliza a `stop_pending`, mantiene `SupervisorFrozen` y conserva refs
compactas del backend vivo. Evidencia:
`TestRuntimeV0ServerShutdownReadyNoBorraSnapshotPrevioActivoV0`.

Avance BUG-ORQ-20260701-065/076 2026-07-03 tarde 17: si el handler responde
HTTP 409 sin cuerpo util despues del snapshot previo, el runtime recupera el
active work observado antes del handler y mantiene el estado
`backend_still_running`/congelado en vez de perder la causa por body vacio.
Evidencia:
`TestRuntimeV0ServerShutdownConflictSinCuerpoConservaSnapshotPrevioActivoV0`.

Avance BUG-ORQ-20260701-079 2026-07-03 tarde 18: el start packet Codex Goal ya
incluye `direction_contract` estructurado con `require_early_checkpoint`,
`early_checkpoint_file=checkpoint_started.txt`, `tool_output_policy` con
`max_text_bytes`, comandos acotados y evidencia durable requerida, ademas de
campos terminales obligatorios, write-set y artefactos esperados. Esto evita que
checkpoint temprano y limite de salidas gigantes dependan solo de texto libre de
prompt. Evidencia:
`TestBuildCodexGoalStartPacketV0IncluyeContratoDeDireccion`.

Avance BUG-ORQ-20260701-079 2026-07-03 tarde 20: el prompt efectivo que consume
`app_server_tmux` se deriva del `direction_contract` para checkpoint temprano y
politica de salidas, incluyendo `max_text_bytes=16384` y hints acotados como
`rg --max-count`/`rg --files | head`. Esto reduce la deriva entre contrato
estructurado y superficie ejecutada por Codex Goal. Evidencia:
`TestBuildCodexGoalStartPacketV0IncluyeContratoDeDireccion`.

Avance BUG-ORQ-20260701-079 2026-07-03 tarde 21: el sanitizer de `thread/read`
del backend `app_server` usa el mismo limite `CodexGoalToolOutputMaxBytesV0`
que el `direction_contract`; una salida textual de herramienta que excede ese
limite se transforma en ref compacto `thread-output-ref-*` antes de alimentar
observacion/cierre. Evidencia:
`TestCodexAppServerThreadReadLimiteTextoSigueContratoDireccionV0`.

Avance BUG-ORQ-20260701-079 2026-07-03 tarde 22: la observacion de un goal activo
vuelve a sanear defensivamente `thread/read` y conserva
`evidence-ref-codex-app-server-thread-output-sanitized` cuando compacta una
salida textual gigante, sin marcar terminal ni bloquear el timeout/replan
posterior. Evidencia:
`TestServerCodexAppServerGoalBackendV0ObservaOutputGiganteConEvidenciaSaneadaV0`.

Avance BUG-ORQ-20260701-079 2026-07-04 noche 5: `thread/read` en
`orquesta-runtime-codex-appserver` tiene presupuesto especifico de respuesta
de 256 KiB tanto por WebSocket como por protocolo command. En WebSocket, si el
frame anunciado supera ese limite, falla antes de reservar y leer el payload
completo; en command, si la linea stdout JSON-RPC de `thread/read` supera ese
limite, el scanner devuelve `codex_app_server_thread_read_response_too_large`,
cierra stdin y mata el proceso hijo para no degradar a timeout. El lector
generico conserva 16 MiB para el resto de RPCs y el command conserva 1 MiB para
respuestas no `thread/read`. Es un corte semipreventivo de ingesta, no
enforcement total de proveedor: el runtime externo puede haber generado ya la
salida grande antes de que Orquesta la rechace.
Evidencia:
`TestCodexAppServerWebSocketThreadReadResponseBudgetV0`,
`TestCodexAppServerCommandProtocolThreadReadResponseBudgetV0`,
`TestCodexAppServerWebSocketDefaultFrameBudgetConserva16MiBV0` y
`go test -count=1 ./modulos/orquesta-runtime-codex-appserver`.

Avance BUG-ORQ-20260701-058/066 2026-07-02 noche 7:
`orquesta-opes-bridge` normaliza los aliases de cierre
`finalize_syllabus_package`, `completed_syllabus_package` y
`paquete_final_temario` al contrato OPES `completed_syllabus_package`, con
contexto `large` y criterios de cierre de temario, en vez de caer al paquete
generico. Evidencia:
`TestBuildExternalWorkRunRequestV0MapeaDerivadosOPESConArtefactosEsperados`,
`TestOPESBridgeArtifactContractMapConsumeOwnerNeutralV0` y
`go test -count=1 ./modulos/orquesta-opes-bridge`.

Avance BUG-ORQ-20260701-065/076 2026-07-03 tarde 23: el cliente
`orquesta-server stop` reintenta el POST gobernado a `/api/v0/server/shutdown`
cuando recibe `backend_still_running` con `cleanup_goal_backends=true`, sin
agentes, checkpoints ni async work pendientes, para dar ventana al cleaner de
backend Goal propio antes de fallar. No permite senal local ni declara ready si
el backend sigue vivo. Evidencia:
`TestRequestServerShutdownV0ReintentaCleanupBackendStillRunningHastaReady`.

Avance BUG-ORQ-20260701-065/076 2026-07-03 tarde 24: el mismo reintento de
cleanup cubre tambien el contrato HTTP real `409 backend_still_running`,
decodificando solo el JSON publico de shutdown y sin relajar el saneamiento
global de otros comandos. Evidencia:
`TestRequestServerShutdownV0ReintentaCleanupBackendStillRunningHTTP409HastaReady`
y `TestReadCommandHTTPResponseBodyAllowStatusV0PermiteConflictJSON`.

Avance BUG-ORQ-20260701-065/076 2026-07-03 tarde 25: la ventana de reintento
del cliente queda acotada a `backend_still_running` recuperable; un
`HTTP 409 active_goals_present`, incluso con `--force`, falla como
`shutdown_not_ready` sin rePOST ni polling de estado, evitando convertir goals
activos normales en cleanup de backend. Evidencia:
`TestRequestServerShutdownV0NoReintentaHTTP409ActiveGoalsPresent`.

Avance BUG-ORQ-20260701-065/076 2026-07-03 tarde 26: el reintento de cleanup
por `backend_still_running` exige tambien runs drenadas (`runs_stopped >=
runs_requested` cuando hay runs solicitadas); si quedan runs pendientes, el
cliente falla como `shutdown_not_ready` sin rePOST ni polling, aunque haya refs
de backend, para no limpiar un backend que aun podria pertenecer a trabajo vivo
no terminal. Evidencia:
`TestRequestServerShutdownV0NoReintentaBackendStillRunningConRunsPendientes`.

Avance BUG-ORQ-20260701-065/076 2026-07-03 tarde 27: si el cliente entro en
ventana de espera por un `backend_still_running` inicialmente recuperable, pero
el `/status` posterior revela runs pendientes o cualquier estado ya no
esperable, corta inmediatamente con `shutdown_not_ready` en vez de dormir hasta
timeout. Evidencia:
`TestRequestServerShutdownV0CortaEsperaSiStatusBackendTieneRunsPendientes`.

Avance BUG-ORQ-20260701-065/088 2026-07-03 tarde 28: si
`POST /api/v0/runs/control` excede la ventana HTTP acotada, el JSON publico de
timeout conserva `forced`, `evidence_refs` compactas y un diagnostico
`run_control_timeout` con scope de run. Esto evita que reintentos de
reconciliacion por cleanup externo o alto consumo pierdan la evidencia que ya
aportaba el operador/status. Evidencia:
`TestMCPRunControlHTTPHandlerV0TimeoutDevuelveJSONPublico`.

Avance BUG-ORQ-20260701-065/088 2026-07-03 tarde 29: si el executor de
`runs/control` falla antes de devolver resultado, la respuesta HTTP 500 conserva
`forced`, `evidence_refs` compactas y diagnostico `run_control_http_error` con
scope de run. Asi el operador puede repetir o reconciliar con la misma evidencia
sin depender de logs internos. Evidencia:
`TestMCPRunControlHTTPHandlerV0ExecutorErrorConservaEvidenciaV0`.

Avance BUG-ORQ-20260701-065/088 2026-07-03 tarde 30: el descriptor MCP de
`orquesta.runs.control.v0` ya declara que tanto respuestas `ok` como `error`
pueden transportar `forced`, `evidence_refs` y `diagnostics`; el contrato
publico queda alineado con los errores recuperables HTTP y evita que clientes
compactos descarten evidencia. Evidencia:
`TestMCPRunControlDescriptorV0EsAdaptadorFino`.

Avance BUG-ORQ-20260701-065/088 2026-07-03 tarde 30b: el mismo descriptor y los
docs de contrato declaran tambien los campos operacionales de Goal-first:
`previous_status`, `final_status`, refs/estado de Goal antes y despues, senal
Goal enviada/confirmada y `recommended_action`. Asi clientes MCP/HTTP compactos
no pierden el diagnostico de `control_not_propagated_to_goal_backend` ni la
reconciliacion de cleanup externo. Evidencia:
`TestMCPRunControlDescriptorV0EsAdaptadorFino`.

Avance BUG-ORQ-20260701-065/088 2026-07-03 tarde 31: el constructor comun de
errores de `orquesta.runs.control.v0` conserva ahora `forced`,
`evidence_refs` compactas y diagnostico con scope de run tambien para errores
de tool como accion no soportada, puerto ausente o validacion. Esto mantiene la
misma evidencia de reconciliacion fuera del transporte HTTP. Evidencia:
`TestMCPRunControlExecutorV0ErrorConservaEvidenciaV0`.

Avance BUG-ORQ-20260701-065/076 2026-07-03 tarde 32: el contrato MCP de
`orquesta.server.shutdown.v0` declara `evidence_refs` en respuestas `ok` y
`error`, y los errores tempranos de la tool conservan evidencia compacta de la
peticion. Esto evita que un fallo de identidad/transporte borre evidencia de
checkpoint, cleanup o shutdown gobernado antes de llegar al caso de uso.
Evidencia: `TestMCPServerShutdownDescriptorV0DeclaraEvidenciaV0` y
`TestMCPServerShutdownToolExecutorV0ErrorConservaEvidenciaV0`.

Avance BUG-ORQ-20260701-065/076 2026-07-03 tarde 32b: el cliente
`orquesta-server stop` ya no interpreta `runs=0/0` y ausencia de contadores como
estado drenado si el `status` explicito del shutdown sigue siendo
`backend_still_running`, `active_goals_present`, `waiting_drain` o
`waiting_checkpoint`. El atajo de cierre por runs drenadas queda limitado a
status terminal/compatible (`ready`, `stopped`, `handoff_ready` o vacio legacy),
evitando una senal local prematura ante cuerpos parciales de conflicto.
Evidencia: `TestShutdownClientReadyV0NoSaltaStatusConflictoSinContadoresV0`.

Avance BUG-ORQ-20260701-065/076 2026-07-03 tarde 32c: la politica de escalado
de `orquesta-server stop --force` tampoco permite senal local si el `/status`
posterior declara `shutdown_status=backend_still_running` o
`active_goals_present`, aunque ese snapshot haya perdido contadores y refs de
trabajo vivo. Esto conserva el veto operacional del estado explicito y evita
que un error de transporte convierta un conflicto vivo en parada local.
Evidencia: `TestShutdownRequestErrorAllowsSignalV0SoloConTimeoutYEstadoDrenado`.

Avance BUG-ORQ-20260701-065/076 2026-07-03 tarde 32d: el mismo veto aplica a
snapshots publicos que declaran `shutdown_status=waiting_drain` o
`waiting_checkpoint` sin contadores por una respuesta parcial. Esos estados de
espera no se tratan como drenados por omision, de modo que un timeout/error de
transporte no dispara senal local mientras el propio status dice que shutdown
sigue esperando drain o checkpoint. Evidencia:
`TestShutdownRequestErrorAllowsSignalV0SoloConTimeoutYEstadoDrenado`.

Avance BUG-ORQ-20260701-065/076 2026-07-03 tarde 32e: si el propio error
`shutdown_not_ready` conserva `status=waiting_drain`, `waiting_checkpoint` o
`stop_pending`, `orquesta-server stop --force` lo trata como estado no terminal
aunque el error no incluya contadores positivos. Asi un body parcial convertido
en error no puede autorizar senal local solo por `runs=0/0`.
Evidencia: `TestShutdownRequestErrorAllowsSignalV0SoloConTimeoutYEstadoDrenado`.

Avance BUG-ORQ-20260701-065/076 2026-07-03 tarde 33: los transportes HTTP y MCP
de `server.shutdown` quedan cubiertos para errores de executor: ambos devuelven
mensaje publico saneado sin rutas/tokens y conservan `evidence_refs` compactas
del request. Evidencia:
`TestMCPServerShutdownDescriptorV0DeclaraEvidenciaV0`,
`TestMCPServerShutdownToolExecutorV0ErrorConservaEvidenciaV0`,
`TestMCPServerShutdownHTTPHandlerV0ActiveGoalsDevuelveConflict` y
`TestMCPServerShutdownHTTPHandlerV0BackendStillRunningDevuelveConflict`,
`TestMCPServerShutdownHTTPHandlerV0NoPropagaErrorNoCatalogado` y
`TestMCPTransportV0ServerShutdownDevuelvePayloadPublicoSiExecutorFalla`.

Avance BUG-ORQ-20260701-065/076 2026-07-03 tarde 33b: los contratos y pruebas
documentales de `orquesta.server.shutdown.v0` quedan alineados con esa
superficie: `cleanup_goal_backends`, `active_work_count`, `active_works`,
`evidence_refs` en errores y HTTP 409 JSON para `active_goals_present`/
`backend_still_running` ya no quedan como comportamiento implicito de codigo.
Evidencia documental: `modulos/orquesta-mcp/docs/contratos.md`,
`modulos/orquesta-mcp/docs/pruebas.md` y
`modulos/orquesta-mcp/docs/decisiones.md`.

Avance BUG-ORQ-20260701-065/076 2026-07-03 tarde 33c: el descriptor ejecutable
de `orquesta.server.shutdown.v0` declara tambien los estados publicos
`active_goals_present` y `backend_still_running` en `Output`, no solo los
campos de active work. Evidencia:
`TestMCPServerShutdownDescriptorV0DeclaraEvidenciaV0`.

Avance BUG-ORQ-20260701-065/076 2026-07-03 tarde 33d: la politica de escalado
del cliente `orquesta-server stop --force` trata tambien
`shutdown_status=stop_pending` leido desde `/status` como conflicto vivo aunque
el snapshot haya perdido contadores. Antes ese veto existia si `stop_pending`
venia en el error `shutdown_not_ready`, pero no en el estado publico posterior.
Evidencia: `TestShutdownRequestErrorAllowsSignalV0SoloConTimeoutYEstadoDrenado`
y `TestShutdownClientReadyV0NoSaltaStatusConflictoSinContadoresV0`.

Avance BUG-ORQ-20260701-065/076 2026-07-03 tarde 33e: el descriptor y docs de
`orquesta.server.shutdown.v0` declaran `stop_pending` como estado publico no
terminal junto a `waiting_drain`, `waiting_checkpoint`, `active_goals_present`
y `backend_still_running`. Asi clientes MCP/REST compactos no tratan ese estado
como legacy desconocido ni autorizan senal local por omision. Evidencia:
`TestMCPServerShutdownDescriptorV0DeclaraEvidenciaV0`.

Avance BUG-ORQ-20260701-065/088 2026-07-03 tarde 34: el transporte MCP de
`orquesta.runs.supervisor.v0` tambien conserva `operation_ref`,
`evidence_refs` estables y diagnostico `run_supervisor_execute_error` cuando el
executor falla sin devolver resultado publico propio. Esto evita perder la
causa compacta durante supervisiones/reconciliaciones de lifecycle. Evidencia:
`TestMCPRunSupervisorTransportHandlerV0DevuelvePayloadPublicoSiExecutorNoDaResultado`
y `TestMCPRunSupervisorDescriptorV0DeclaraEvidenciaEnErrores`.

Avance BUG-ORQ-20260701-065/088 2026-07-03 tarde 34b: el descriptor ejecutable y
el contrato documental de `orquesta.runs.supervisor.v0` declaran
`operation_ref` y `repair_run_refs` en salidas `ok` y `error`, alineando el
contrato con errores recuperables y reworks causales que ya publica el executor.
Evidencia: `TestMCPRunSupervisorDescriptorV0DeclaraEvidenciaEnErrores`.

Avance BUG-ORQ-20260701-065/088 2026-07-03 tarde 35: el HTTP
`POST /api/v0/runs/supervise` usa el mismo error publico de executor que el
transporte MCP cuando no hay resultado propio: conserva correlacion,
`operation_ref`, `evidence_refs` y diagnostico `run_supervisor_execute_error`
sin filtrar rutas ni secretos. Evidencia:
`TestMCPRunSupervisorHTTPHandlerV0NoPropagaErrorNoCatalogado`.

Avance BUG-ORQ-20260701-065/088 2026-07-03 tarde 36: la fachada
`orquesta.autoprogramming.supervise.v0` alinea HTTP y transporte MCP con la
misma regla de evidencia: errores de executor sin resultado propio conservan
`operation_ref`, `evidence_refs` y diagnostico
`autoprogramming_supervise_executor_error`, y el descriptor declara
`evidence_refs` en errores. Evidencia:
`TestMCPAutoprogrammingSuperviseHTTPHandlerV0DevuelvePayloadPublicoSiExecutorFalla`,
`TestMCPAutoprogrammingSuperviseTransportV0DevuelvePayloadPublicoSiExecutorFalla`
y `TestMCPAutoprogrammingSuperviseDescriptorV0DeclaraEvidenciaEnErrores`.

Avance BUG-ORQ-20260701-065/088 2026-07-03 tarde 36b: el descriptor y contrato
de `orquesta.autoprogramming.supervise.v0` declaran tambien `idempotency_key`,
`repair_run_refs` y `next_actions` en salidas `ok` y `error`, alineandose con
el resultado compacto de `runs.supervisor` que transporta. Evidencia:
`TestMCPAutoprogrammingSuperviseDescriptorV0DeclaraEvidenciaEnErrores`.

Avance BUG-ORQ-20260701-065/088 2026-07-03 tarde 36c: la misma fachada
`orquesta.autoprogramming.supervise.v0` publica ahora la forma tipada de
`diagnostics` (`code/scope/message/evidence_refs`) en descriptor y contrato,
alineandose con `orquesta.runs.supervisor.v0` y evitando que clientes compactos
pierdan causa/evidencia de aceptacion en background, error de executor o rework
residente. Evidencia:
`TestMCPAutoprogrammingSuperviseDescriptorV0DeclaraEvidenciaEnErrores`.

Avance BUG-ORQ-20260701-065/088 2026-07-03 tarde 37: `autoprogramming/status`
publica `evidence_refs` de primer nivel y en diagnosticos para errores de
executor y timeout HTTP, y su descriptor declara esa evidencia en la rama de
error. Asi los consumidores compactos conservan la causa de status/shutdown
sin inspeccionar logs internos. Evidencia:
`TestMCPAutoprogrammingStatusHTTPHandlerV0NoPropagaErrorNoCatalogado`,
`TestMCPAutoprogrammingStatusHTTPHandlerV0TimeoutDevuelveJSONPublico`,
`TestMCPAutoprogrammingStatusTransportV0DevuelvePayloadPublicoSiExecutorFalla`
y `TestMCPAutoprogrammingStatusDescriptorV0EsAdaptadorFino`.

Avance BUG-ORQ-20260701-066/075 2026-07-03 tarde 34: la fachada
`/api/v0/domain-work/status` publica diagnostico compacto
`domain_work_status_http_error` con evidencia estable cuando falla el executor,
manteniendo mensaje publico saneado. Esto permite a OPES/domain-work distinguir
fallo operacional de consulta frente a un estado bloqueado real sin inspeccionar
logs internos. Evidencia:
`TestMCPDomainWorkStatusHTTPHandlerV0ExecutorErrorDevuelveDiagnosticoPublico`.

Avance BUG-ORQ-20260701-066/075 2026-07-03 tarde 35: la fachada
`/api/v0/domain-work/status` queda cubierta tambien para timeout HTTP acotado:
devuelve 504 con JSON publico, filtros conservados, diagnostico
`domain_work_status_timeout` y evidencia estable. Esto evita que conectores
OPES/domain-work interpreten un timeout de consulta como cola vacia o estado
no accionable. Evidencia:
`TestMCPDomainWorkStatusHTTPHandlerV0TimeoutDevuelveJSONPublico`.

Avance BUG-ORQ-20260701-073/075 2026-07-03 tarde 36: los descriptores MCP de
`orquesta.apps.observe_director_goal.v0` y
`orquesta.autoprogramming.observe_goal.v0` declaran `evidence_refs?` tambien en
respuestas de error. El contrato queda alineado con los timeouts parciales y
errores enriquecidos que ya conservan evidencia para decidir esperar, reparar
receipt o replanificar sin depender de logs internos. Evidencia:
`TestObserveGoalDescriptorsDeclaranEvidenciaEnErroresV0`.

Avance BUG-ORQ-20260701-073/075 2026-07-03 tarde 37: el descriptor MCP de
`orquesta.autoprogramming.observe_active_goals.v0` declara tambien
`evidence_refs?` en respuestas de error, alineandose con la agregacion de
evidencias de observaciones activas y con los contratos de `observe_goal`.
Evidencia:
`TestMCPAutoprogrammingObserveActiveGoalsDescriptorV0DeclaraEvidenciaEnErrores`.

Avance BUG-ORQ-20260701-073/075 2026-07-03 tarde 37b: el mismo descriptor y
contrato declaran `operation_ref`, `next_actions` y `diagnostics` en salidas
`ok` y `error`, de modo que los clientes no descartan el handoff de background
ni el diagnostico publico de observe-active. Evidencia:
`TestMCPAutoprogrammingObserveActiveGoalsDescriptorV0DeclaraEvidenciaEnErrores`.

Avance BUG-ORQ-20260701-073/075 2026-07-03 tarde 38: el HTTP handler de
`/api/v0/autoprogramming/goals/observe-active` devuelve diagnostico publico y
`evidence_refs` estables cuando falla el executor, con mensaje saneado sin
rutas ni secretos. Esto permite distinguir fallo operacional de observacion
batch frente a ausencia de goals o estado idle. Evidencia:
`TestMCPAutoprogrammingObserveActiveGoalsHTTPHandlerV0ExecutorErrorDevuelveDiagnosticoPublico`.

Avance BUG-ORQ-20260701-073/075/088 2026-07-03 tarde 39: la fachada
`/api/v0/domain-work/status` traduce las senales goal-first de checkpoint y alto
consumo a estados operacionales de dominio: `active_timeout_checkpoint_recent`,
`active_no_checkpoint_yet`, `checkpoint_only_consumption_warning` y
`goal_active_no_checkpoint_consumption_warning` se publican como `running` con
accion concreta de observar/exigir checkpoint o siguiente artefacto; los cortes
`checkpoint_only_high_consumption`, `goal_active_no_checkpoint_high_consumption`
y `goal_active_timeout_backend_active` se publican como `blocked`, conservando
la accion especifica de replan cuando aplica, incluido
`replan_goal_after_active_timeout`. Evidencia:
`TestMCPDomainWorkStatusHTTPHandlerV0NormalizaSenalesCheckpointComoEstadoOperacional`
y `TestMCPQueueGlobalStatusNormalizeRecommendedActionV0PreservaAccionesGoalFirstEspecificas`.

Avance BUG-ORQ-20260701-077 2026-07-03 tarde 28: la guarda del cliente
`orquesta-server stop` cubre tambien respuestas `ready` que conservan
`active_work_refs` pero omiten `active_work_count`; el cliente normaliza el
contador y no permite senal local mientras quede evidencia de backend Goal
activo. Evidencia:
`TestRequestServerShutdownV0ReadyNoSaltaActiveWorkRefsSinContadorV0`.

Avance BUG-ORQ-20260701-073/088 2026-07-03 tarde 40:
`autoprogramming/status` publica `goal_progress_policy` normalizada en el mismo
payload que decide `active_timeout_checkpoint_recent`,
`checkpoint_only_consumption_warning` y `checkpoint_only_high_consumption`.
Asi un operador o panel puede auditar los umbrales efectivos de tokens y espera
sin consultar aparte `/server/status`. Evidencia:
`TestMCPAutoprogrammingStatusDescriptorV0EsAdaptadorFino` y
`TestMCPAutoprogrammingStatusExecutorV0GoalActiveTimeoutConCheckpointRespetaUmbralConfiguradoV0`.

Avance BUG-ORQ-20260701-073/088 2026-07-03 tarde 41: la publicacion de
`goal_progress_policy` queda cubierta tambien por las superficies publicas
HTTP y transporte MCP registrado con bindings, no solo por el executor directo.
Esto evita que paneles o agentes remotos pierdan los umbrales efectivos al
consultar `/api/v0/autoprogramming/status` o
`orquesta.autoprogramming.status.v0`. Evidencia:
`TestMCPAutoprogrammingStatusHTTPHandlerV0SerializaGoalProgressPolicyConfigurada`
y
`TestMCPTransportV0AutoprogrammingStatusPublicaGoalProgressPolicyDesdeBindings`.

Avance BUG-ORQ-20260701-073/088 2026-07-03 tarde 42:
`/api/v0/queue/global-status` conserva `goal_progress_policy` desde
`autoprogramming/status` cuando publica acciones como `replan_narrow_context`
por `checkpoint_only_high_consumption`. Asi el panel global no pierde los
umbrales efectivos que explican un corte por alto consumo/checkpoint. Evidencia:
`TestMCPQueueGlobalStatusHTTPHandlerV0ConservaReplanNarrowContext`.

Avance BUG-ORQ-20260701-073/088 2026-07-03 tarde 43:
el contrato y las pruebas documentales de `rest.bridge.orquesta.queue.global_status.v0`
declaran `goal_progress_policy` como salida publica y enlazan la evidencia
`TestMCPQueueGlobalStatusHTTPHandlerV0ConservaReplanNarrowContext`, evitando que
la superficie global vuelva a quedar por detras de `autoprogramming/status`.

Avance BUG-ORQ-20260701-066/088 2026-07-03 tarde 44:
`orquesta.apps.request_change.v0` ya transportaba `evidence_refs` desde la
notificacion de app-change al Director, pero el descriptor compacto no las
declaraba. El contrato MCP y `docs/contratos.md` publican ahora
`evidence_refs?` en salidas `ok` y `error`, y la prueba focal verifica que el
adaptador conserva la evidencia del notifier. Evidencia:
`TestMCPRequestAppChangeV0DelegaEnCasoDeUso` y
`TestMCPRequestAppChangeDescriptorV0EsAdaptadorFino`.

Avance BUG-ORQ-20260701-065/088 2026-07-03 tarde 45:
`orquesta.runtime.models.v0` ya acepta evidencia operacional en acciones de
runtime y los resultados `RuntimeModelListResultV0`/`RuntimeModelActionResultV0`
pueden devolver `evidence`, pero el descriptor compacto solo anunciaba
`list_result`/`action_result`. El contrato MCP y `docs/contratos.md` declaran
ahora evidencia en ambas salidas para que clientes compactos no pierdan contexto
de serve/stop/pull/status antes de decidir shutdown o reconciliacion. Evidencia:
`TestMCPRuntimeModelsDescriptorV0DeclaraEvidenciaOperativa`.

Avance BUG-ORQ-20260701-066/088 2026-07-03 tarde 46:
`orquesta.external_work.run.v0` podia devolver `evidence_refs`, `next_actions`
y `operation_endpoints` tambien cuando el caso de uso rechazaba la peticion, por
ejemplo por backend Goal no disponible o necesidad de observar/reparar, pero el
descriptor compacto de error solo anunciaba `errores_publicos`. El descriptor y
`docs/contratos.md` declaran ahora esas salidas en error, y la prueba focal
verifica que un resultado rechazado conserva evidencia y acciones compactas.
Evidencia: `TestNewMCPExternalWorkRunResultV0ConservaEvidenciaEnError`.

Avance BUG-ORQ-20260701-066/088 2026-07-03 tarde 47:
`orquesta.external_work.dry_run.v0` devolvia evidencia del preview solo en
salidas `ok`; si la compilacion del spec fallaba, el cliente veia errores sin
ref compacta que distinguiese un preview fallido de un lanzamiento real. Los
errores de input, contrato externo y validacion GoalSpec conservan ahora
`evidence-ref-external-work-dry-run-v0`, y el descriptor/contrato declaran
`evidence_refs?` tambien en `error`. Evidencia:
`TestBuildExternalWorkDryRunV0RechazaEntradaAmbiguaV0` y
`TestMCPExternalWorkDryRunDescriptorV0DeclaraEvidenciaEnError`.

Avance BUG-ORQ-20260701-065/085/088 2026-07-03 tarde 48:
`orquesta.director.stats.v0` ya proyectaba en `goal` los artefactos,
`domain_receipt_refs` y `expected_terminal_receipt_refs`, pero el descriptor
compacto solo declaraba issue/evidence. El descriptor y `docs/contratos.md`
publican ahora esas refs para que clientes compactos puedan decidir `runs/control`,
reparacion de receipt o rework por write-set sin inferirlo desde logs. Evidencia:
`TestMCPDirectorStatsToolDescriptorV0ExponeContratoCompacto`.

Avance BUG-ORQ-20260701-066/075/085 2026-07-03 tarde 49:
la fachada `/api/v0/domain-work/status` queda cubierta para acciones goal-first
agregadas con scope `:run:<run_ref>`; asi no se pierde el run causal al cruzar
desde `queue/global-status` hacia estado de dominio OPES/domain-work. Evidencia:
`TestMCPDomainWorkStatusHTTPHandlerV0ConservaAccionGoalFirstPorRun`.

Avance BUG-ORQ-CODEBASE-20260702-001/BUG-ORQ-20260701-065 2026-07-03 tarde 50:
`orquesta.codebase.status.v0` ya transportaba `evidence_refs` de owner markers
y observaciones en `entries`, pero el descriptor compacto solo declaraba
`entries` genericas. El descriptor y docs de contrato publican ahora
`entries[].evidence_refs` y evidencia de primer nivel, manteniendo visible por
que un lease de codebase debe continuar o puede pararse sin inspeccionar logs.
Evidencia: `TestMCPCodebaseStatusDescriptorV0DeclaraEvidenciaPublica`.

Avance BUG-ORQ-20260701-066/075/088 2026-07-03 tarde 51:
`orquesta.domain_work.v0` transporta `evidence_refs` dentro de
`DomainWorkJobV0`, `DomainWorkArtifactReceiptV0` y evaluaciones de capabilities,
pero el descriptor compacto solo anunciaba `job`, `receipt` y
`external_capability_evaluation` genericos. El descriptor y docs publican ahora
`evidence_refs` en job/receipt, matched capabilities y missing requirements,
manteniendo evidencia causal para OPES/domain-work sin mirar logs internos.
Evidencia: `TestMCPDomainWorkDescriptorV0EsAdaptadorFino`.

Avance BUG-ORQ-20260701-066/088 2026-07-03 tarde 52:
`orquesta.autoprogramming.prepare_run.v0` ya no publica
`goal_spec_summaries` como array opaco en el descriptor compacto: declara
`schema_version`, refs publicas, `spec_hash`, refs de tests, tipos de artefacto
y contadores de cierre sin exponer `GoalWorkSpecV0` completo, objective,
write-set ni comandos. Evidencia:
`TestMCPAutoprogrammingPrepareRunDescriptorV0EsAdaptadorOptIn`.

Avance BUG-ORQ-20260701-066/088 2026-07-03 tarde 53:
`orquesta.apps.preparar_orquestacion.v0` ya transportaba `evidence_refs` de
primer nivel y dentro del plan compacto, pero el descriptor ejecutable solo
declaraba `plan`/`progress` genericos. El descriptor y docs publican ahora
`plan.evidence_refs?` y `evidence_refs?`, manteniendo trazabilidad de preview
legacy sin relanzar runtime ni ocultar que `arrancar_director` es la entrada
preferente goal-first. Evidencia:
`TestMCPAppSpecDescriptorsV0PublicanRoutePolicy`.

Avance BUG-ORQ-20260701-066/088 2026-07-03 tarde 54:
`orquesta.apps.ejecutar_orquestacion.v0` ya transportaba `evidence_refs` desde
el loop legacy por puertos, pero el descriptor compacto de salida `ok` no las
declaraba. El descriptor y docs publican ahora `evidence_refs?` sin cambiar la
frontera: sigue siendo compatibilidad legacy solo con
`director_execution_mode=legacy_director_loop`, y `arrancar_director` queda como
entrada preferente goal-first. Evidencia:
`TestMCPAppSpecDescriptorsV0PublicanRoutePolicy`.

Avance BUG-ORQ-20260701-065/088 2026-07-03 tarde 55:
`orquesta.autoprogramming.self_improvement.propose.v0` devolvia `accepted` y
`background`, pero el descriptor compacto no declaraba esos campos; clientes
finos podian tratar una automejora secundaria de baja prioridad como trabajo
principal o perder que el rechazo seguia siendo reparable sin descartar
evidencia. El descriptor y docs publican ahora `accepted`/`background` en `ok`
y `error`. Evidencia:
`TestMCPAutoprogrammingSelfImprovementDescriptorV0EsAdaptadorFino`.

Avance BUG-ORQ-20260701-077 2026-07-03 tarde 56: el handoff de cierre de
sesion conserva las verificaciones de `777e027c` solo como evidencia historica
del corte original y exige nueva prueba focal tras `fetch/rebase`, evitando que
un operador reutilice el verde antiguo para reabrir rutas manuales de servidor
o saltarse la guarda de shutdown gestionado. Evidencia:
`TestHandoffCierreSesionNoReabreBUG077V0`.

Avance BUG-ORQ-20260701-066/075/088 2026-07-03 tarde 57:
`orquesta.apps.observe_director_goal.v0` y
`orquesta.autoprogramming.observe_goal.v0` ya devolvian
`recommended_action` para esperar, reparar receipt, rework de QA/write-set,
continuar fase 0, replanificar o reintentar tras timeout, pero el descriptor
compacto no lo publicaba. Los descriptores y docs de contrato declaran ahora
`recommended_action?` en `ok` y `error`, evitando que clientes MCP compactos
traten un goal accionable como simple observacion pasiva. Evidencia:
`TestObserveGoalDescriptorsDeclaranEvidenciaYAccionRecomendadaV0`.

Avance BUG-ORQ-20260701-066/075/088 2026-07-03 tarde 58:
`orquesta.autoprogramming.observe_active_goals.v0` agrega resultados de
`observe_goal` por run, pero su descriptor declaraba `observations?` como bloque
opaco. El contrato compacto publica ahora `observations?[]{run_ref,goal_ref?,
goal_status?,recommended_action?,evidence_refs?}`, conservando accion y
evidencia por goal cuando se observa un lote de goals activos sin relanzar
supervision legacy. Evidencia:
`TestMCPAutoprogrammingObserveActiveGoalsDescriptorV0DeclaraObservacionesAccionables`.

Avance BUG-ORQ-20260701-066/075/088 2026-07-03 tarde 58b:
`orquesta.autoprogramming.observe_active_goals.v0` tambien publica diagnosticos
publicos de la pasada (`code/scope/message/evidence_refs`) para background,
errores de executor e incidencias por goal, pero el descriptor y contrato los
dejaban opacos. La salida `ok` y `error` declara ahora la forma tipada,
preservando causa/evidencia por lote sin inspeccionar logs. Evidencia:
`TestMCPAutoprogrammingObserveActiveGoalsDescriptorV0DeclaraObservacionesAccionables`.

Avance BUG-ORQ-20260701-065/075/085/088 2026-07-03 tarde 59:
`orquesta.autoprogramming.status.v0` ya publicaba acciones por run en
`stale_running` y una accion agregada en `efficiency_summary`, pero el
descriptor compacto dejaba ambos bloques sin forma publica. El descriptor y
contrato declaran ahora `stale_running?[]{code,severity?,run_ref?,status?,
goal_ref?,goal_status?,recommended_action?,evidence_refs?}` y
`efficiency_summary?{schema_version,state,recommended_action?,reasons?}`, de
modo que clientes compactos pueden preservar replan, repair receipt,
write-set/sandbox y cleanup externo sin inspeccionar logs ni listas internas.
Evidencia: `TestMCPAutoprogrammingStatusDescriptorV0EsAdaptadorFino`.

Avance BUG-ORQ-20260701-065/076/088 2026-07-03 tarde 60:
`orquesta.server.shutdown.v0` ya transportaba `active_works` de backends Goal y
resumenes `runs` con checkpoint por agente, pero el descriptor declaraba esos
bloques como opacos. El contrato compacto publica ahora
`active_works?[]{kind,run_ref?,work_ref?,external_work_ref?,status?,
evidence_refs?}` y `runs?[]{run_ref,control_status?,checkpoint_required?,
checkpoint_ref?,pending_checkpoint_agent_refs?,checkpoint_evidence_refs?,ready}`,
manteniendo visible por que un shutdown queda en `active_goals_present`,
`backend_still_running` o `stop_pending` sin leer logs. Evidencia:
`TestMCPServerShutdownDescriptorV0DeclaraEvidenciaV0`.

Avance BUG-ORQ-20260701-065/076/088 2026-07-03 tarde 61:
`orquesta.server.shutdown.v0` devolvia `checkpoints_pending` y los runbooks lo
describian, pero el descriptor compacto solo declaraba agentes pendientes y
deadlines expirados. El descriptor publica ahora tambien
`checkpoints_pending`, de modo que clientes MCP/REST pueden distinguir backlog
de checkpoints global de refs de agentes concretos antes de decidir wait,
repost o stop final. Evidencia:
`TestMCPServerShutdownDescriptorV0DeclaraEvidenciaV0`.

Avance BUG-ORQ-20260701-065/088 2026-07-03 tarde 62:
`orquesta.runs.control.v0` transportaba diagnosticos publicos con
`code/scope/message/evidence_refs`, pero el descriptor compacto y contrato solo
declaraban `diagnostics?` como bloque opaco. El contrato publica ahora la forma
tipada en `ok` y `error`, conservando la causa compacta de cleanup externo,
backend activo o reconciliacion terminal para clientes que deciden
`stop/cancel/wait` sin leer logs. Evidencia:
`TestMCPRunControlDescriptorV0EsAdaptadorFino`.

Avance BUG-ORQ-20260701-065/088 2026-07-03 tarde 63:
`orquesta.runs.supervisor.v0` tambien transporta diagnosticos publicos con
`code/scope/message/evidence_refs` para aceptacion en background, errores de
executor y reworks/reconciliaciones, pero el descriptor compacto los mantenia
opacos. El descriptor y contrato publican ahora esa forma tipada en `ok` y
`error`, evitando que clientes compactos pierdan causa/evidencia al decidir
consulta posterior, wait o escalado por `runs/control`. Evidencia:
`TestMCPRunSupervisorDescriptorV0DeclaraEvidenciaEnErrores`.

## Incidencias abiertas observadas en pilotos 2026-07-03 tarde 64

| ID | Estado | Area | Sintoma | Hipotesis arquitectonica | Evidencia / enlace | Accion |
| --- | --- | --- | --- | --- | --- | --- |
| BUG-ORQ-20260703-154 | cerrado | Autoprogramacion idle / presupuesto | el piloto MEJ-206 `goal-ref-autoprogramming-backlog-t290-biblioteca-habilidades-curada-5dfce62a` materializo solo un checkpoint `status=invalid` con `implementation_pending`/`tests_pending` y siguio como `running`, acumulando `codex_app_server_goal_status_active_high_token_usage tokens_used=157367 time_used_seconds=214` sin progreso observable | el loop goal-first idle no tenia presupuesto pre-launch ni corte cooperativo durante ejecucion para consumo creciente sin progreso util; un checkpoint invalido repetido podia parecer actividad recuperable y seguir consumiendo | piloto aislado `scratchpad/pilot-m206`; cierre MEJ-104/T290: `budget_deferred`/`budget_degraded` antes de launch, `goal_high_consumption_without_progress` durante observacion, stop cooperativo por run-control, estado durable `blocked` y rework accionable; evidencia durable `modulos/orquesta-server/docs/orquesta_goal_result_goal-ref-autoprogramming-backlog-t290-corte-durante-ejecucion-goal-sin-progreso.json`; tests `TestDecideAutoprogrammingIdleSelfImprovementBudgetV0AplazaAgotado`, `TestDecideAutoprogrammingIdleSelfImprovementBudgetV0DegradaLote`, `TestDecideAutoprogrammingIdleSelfImprovementBudgetV0UsaEstimacionRealParaAplazar`, `TestRuntimeV0AutomejoraIdleAplazaPorPresupuestoAgotadoV0`, `TestRuntimeV0AutomejoraIdleDegradaLotePorPresupuestoContextoV0`, `TestRuntimeV0GoalObservationBloqueaIdleGoalConsumoCrecienteSinProgresoV0`, `TestRuntimeV0GoalObservationNoCortaConProgresoUtilRecienteV0`, `TestRuntimeV0GoalObservationCheckpointInvalidoRepetidoCuentaSinProgresoV0` | Cierre: la causa raiz queda cubierta por presupuesto de automejora idle antes de lanzar y gobernador de progreso para goals activos. No se relanzo smoke real por congelacion operativa de automejora/pilotajes; antes de reactivar pilotos caros queda recomendada una validacion real acotada de `budget_deferred`/`budget_degraded`, pero no hay brecha de implementacion conocida en T290/MEJ-104 |
| BUG-ORQ-20260703-155 | cerrado | Autoprogramacion backlog / contrato de tarea | la request MEJ-206 generada por el planner mezclaba criterios propios de `biblioteca-habilidades-curada` con criterios ajenos de automejora idle (`ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS`, planner scanner, proyeccion publica de outbox/wait_external, etc.) | `idleSelfImprovementRequestForBacklogSectionV0` concatenaba `base.AcceptanceCriteria` con `section.Criteria`, convirtiendo politicas globales de idle/scanner en contrato funcional de una tarea concreta | state del piloto m206 en `idle_self_improvement_goal_spec.objective` y `acceptance_criteria`; cierre 2026-07-03 noche: `TestIdleSelfImprovementBacklogPlannerV0SeccionEjecutableNoHeredaCriteriosBaseV0` reproduce el fallo y pasa tras aislar criterios por seccion | Cierre: las secciones ejecutables y revisiones documentales ambiguas ya no heredan `base.AcceptanceCriteria`; conservan criterios declarados por la seccion y guardas causales propias de backlog. Scanner/fallback mantienen criterios globales porque son inventario, no tarea de implementacion |
| BUG-ORQ-20260703-156 | cerrado | Runtime piloto / procesos residentes | tras cerrar pilotos ya integrados, seguian vivos `orquesta-server run` de `pilot-t285`, `pilot-m202`, `pilot-m203`, `pilot-m205` y varios `codex app-server --listen` bajo sus `CODEX_HOME` temporales; MEJ-206 tambien dejo app-server huerfano tras parar el servidor | los cleaners y hooks de backend Goal existian, pero `shutdownRuntimeV0` y `stopRuntimeAfterServeClosedV0` retornaban por `async_work_timeout` antes de ejecutar `runShutdownHooksV0`; si un tick/preparacion no drenaba, el servidor terminaba con timeout visible pero sin apagar el app-server tmux propio | cierre 2026-07-03 noche: `runShutdownHooksOnceV0`/`runShutdownHooksOnExitV0` garantizan hooks una vez tambien en rutas de timeout; `TestRuntimeV0ShutdownTimeoutPublicaStopTimeoutV0` verifica `hook.calls==1` con `async_work_timeout`; focales `TestRuntimeV0ShutdownEsperaPreparacionIdleAntesDeStoppedV0` y `TestRuntimeV0CompactaShutdownHooksV0` verdes | Cierre: shutdown con timeout conserva evidencia `stop_timeout` y ejecuta limpieza best-effort de backends Goal antes de devolver error; no se tocan core puro ni contratos de dominio |
| BUG-ORQ-20260703-157 | cerrado | Harness de pilotos / ejecucion background | el arranque del piloto MEJ-206 como proceso background/nohup desde el wrapper murio al terminar la llamada de shell: state quedaba con pids `2600222`/`2602158` pero sin proceso vivo, logs vacios y auditoria de startup lista; solo funciono al mantener `orquesta-server run` en una sesion PTY foreground | `orquesta-server run` es foreground, no contrato daemon; los guards ya exigen `orquesta-server start/status/stop` o harness aislado con cleanup, pero el helper comun `smoke_wait_orquesta_readiness_from_state_file` aceptaba `addr` sin validar que el `pid` declarado por el state siguiera vivo | cierre 2026-07-03 noche: el helper comun rechaza statefiles con `pid` no numerico, cero o muerto antes de aceptar readiness HTTP; `TestSmokeCommonReadinessStateFileRechazaPIDMuertoV0` reproduce un HTTP ready con PID muerto y queda bloqueado; `bash -n scripts/lib/smoke_common.sh` verde | Cierre: wrappers versionados no deben usar `nohup ... orquesta-server run &`; para piloto gestionado usar `orquesta-server start/status/stop`, y los smokes que lean state validan PID vivo antes de declarar ready |
| BUG-ORQ-20260703-159 | cerrado | Tests/automejora idle async / idempotencia | durante la validacion global del cierre BUG-155, `TestRuntimeV0SupervisorPreparaAutomejoraTrasIdleV0` fallaba con `retry no lanzado tras cooldown`; reproducido localmente con `go test -count=100 ./modulos/orquesta-server -run TestRuntimeV0SupervisorPreparaAutomejoraTrasIdleV0` antes del fix | la deduplicacion de publicaciones idle conservaba la ultima publicacion `scheduled` aunque el intento terminara en `MarkIdleSelfImprovementErrorV0` o `MarkIdleSelfImprovementPrepareFailedV0`; un retry tras cooldown con el mismo `request_ref` podia silenciarse como duplicado identico antes de invocar `PrepareIdleSelfImprovementV0`; ademas el test mutaba el tracker sin esperar el drenaje del tick async previo | cierre 2026-07-03 noche: `resetIdleSelfImprovementPublicationLockedV0` libera la publicacion activa al cerrar por error/prepare_failed; `TestStatusTrackerV0IdleSelfImprovementErrorLiberaPublicacionParaRetryV0`, `TestStatusTrackerV0PrepareFailedLiberaPublicacionParaRetryV0` y `go test -count=100 ./modulos/orquesta-server -run TestRuntimeV0SupervisorPreparaAutomejoraTrasIdleV0` verdes | Cierre: retries fallidos ya no quedan bloqueados por idempotencia stale; la prueba async espera el drenaje del trabajo anterior antes de forzar el retry |
| BUG-ORQ-20260703-160 | cerrado | Director de escalada / trazabilidad stop | cuando el director de escalada por eventos decidia `stop`, el `GoalCooperativeStopRequestV0` no transportaba evidencias propias de la anomalia escalada; la auditoria posterior de run-control podia ver el stop de Claude/director externo sin refs compactas de codigo/campo que lo motivaron | la decision de escalada y la parada cooperativa estaban cableadas, pero la evidencia causal se quedaba en el resultado observado o en el estado de escalation director, no en el comando operacional que llega al puerto `GoalStopper` | cierre 2026-07-03 noche: `escalationDirectorStopEvidenceRefsV0` anade `evidence-ref-escalation-director-stop`, `evidence-ref-escalation-director-issue:<code>` y `evidence-ref-escalation-director-field:<field>` al stop cooperativo; test `TestRuntimeV0EscalationDirectorAplicaStopYEsIdempotentePorFirmaV0`; `go test -count=1 ./modulos/orquesta-server -run 'TestRuntimeV0EscalationDirectorAplicaStopYEsIdempotentePorFirmaV0|TestEscalationDirectorPendingIssuesV0IncluyeReviewYArtefactosParcialesObservadosV0|TestParseEscalationDirectorDecisionV0'` | Cierre: el stop emitido por director de escalada queda trazable sin inspeccionar logs; no cambia la lista blanca de decisiones ni escala codigos que ya tienen automatismo propio |

Cierre BUG-ORQ-20260703-154 2026-07-03 noche:
MEJ-104 aporta el gobernador pre-launch de automejora idle. La decision usa
presupuesto diario declarado de goals y bytes de contexto, consumo durable del
dia, estimacion de contexto de la siguiente goal y tokens de prompt cache. Si el
presupuesto no cabe, la automejora idle se aplaza con `budget_deferred`; si solo
cabe parte del lote, degrada a menos goals con `budget_degraded`. La decision se
persiste en state, se expone en `/api/v0/server/status` y en
`orquesta.autoprogramming.status.v0` por puerto opcional.

T290, commit `eab3be97`, completa la parte durante ejecucion: el observer de
goals idle aplica `goal_high_consumption_without_progress` cuando el consumo
crece sin progreso util fuera de la ventana, ignora checkpoints invalidos como
progreso, cuenta checkpoint invalido repetido como evidencia, persiste el goal
como `blocked`/`NeedsRework` y pide stop cooperativo por run-control. Evidencia
durable:
`modulos/orquesta-server/docs/orquesta_goal_result_goal-ref-autoprogramming-backlog-t290-corte-durante-ejecucion-goal-sin-progreso.json`.
Validacion focal reejecutada en este cierre documental:
`go test -count=1 ./modulos/orquesta-server -run 'TestRuntimeV0GoalObservation(AltoConsumoSinProgreso|NoCortaConProgresoUtilReciente|CheckpointInvalidoRepetidoCuentaSinProgreso)V0|TestRuntimeV0IdleSelfImprovement(Goal|Observe)'`;
`go test -count=1 ./modulos/orquesta-server -run 'TestRuntimeV0AutomejoraIdle(AplazaPorPresupuestoAgotado|DegradaLotePorPresupuestoContexto)V0|TestServerPublicStatusV0ExponePresupuestoAutomejoraIdleV0'`;
`go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPAutoprogrammingStatusExecutorV0PublicaPresupuestoIdleV0'`;
`go test -count=1 ./cmd/orquesta-server -run 'TestServerConfigFromEnvV0LeePresupuestoAutomejoraIdleV0|TestServerAutoprogrammingIdleBudgetSourceV0(LeeEstadoDurable|OmiteDecisionVacia)'`.
Cierre de residual 2026-07-04: smoke real acotado de presupuesto con servidor
temporal y backend `app_server_tmux` configurado publica `budget_deferred` en
`/api/v0/server/status` y `/api/v0/autoprogramming/status`, no lanza goal Codex
y deja limpieza sin procesos residentes. Runbook:
`docs/runbooks/smoke_autoprogramming_idle_budget_2026-07-04.md`.

Cierre T292/escalation director 2026-07-03 noche:
El riesgo de que el director de escalada por eventos quedara inaccesible desde
`cmd/orquesta-server` queda cubierto en este corte: la fontaneria
`ORQUESTA_SERVER_ESCALATION_DIRECTOR_*` entra por el registro canonico de env,
`_COMMAND` se publica redactado en effective config, y el recolector escala
anomalias existentes `review_*` y `partial_artifacts_written` desde issues
top-level, `Result.Issues` y `Closure.Issues`. Evidencia:
`TestServerConfigFromEnvV0LeeDirectorEscaladaV0`,
`TestEscalationDirectorPendingIssuesV0IncluyeReviewYArtefactosParcialesObservadosV0`,
`go test -count=1 ./cmd/orquesta-server -run 'TestServerConfigFromEnvV0LeeDirectorEscaladaV0|TestServerEnvRegistryV0'`,
`go test -count=1 ./modulos/orquesta-server -run 'Test(EscalationDirector|RuntimeV0EscalationDirector|ParseEscalationDirector)'`
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`,
`go test -count=1 ./...`, `go build ./...` y `git diff --check`. No se abre
`BUG-ORQ-20260703-158`: el hueco queda
cubierto por codigo y tests en esta sesion. Residual de esta tanda que sigue
abierto para revision estructural: `BUG-ORQ-20260703-149` (WIP remoto no
integrable tal cual). `BUG-ORQ-20260703-154`, `BUG-ORQ-20260703-156`,
`BUG-ORQ-20260703-157` y `BUG-ORQ-20260703-159` quedan cerrados con causa
reproducida y tests focales; para `BUG-154` queda solo smoke real acotado como
validacion antes de reactivar automejora/pilotajes, no como brecha de codigo
conocida.
Las filas largas antiguas deben leerse por sus partes vivas: `BUG-065`,
`BUG-066`, `BUG-075` y OPES calidad siguen abiertas como deuda amplia; las
menciones antiguas a `BUG-085` y `BUG-088` dentro de esas filas quedan
supersedidas por sus cierres/supersedencias posteriores. No son regresion nueva
de T292.

Cierre BUG-ORQ-20260703-159 2026-07-03 noche:
La incidencia deja de ser flake no explicado. Se reprodujo con
`go test -count=100 ./modulos/orquesta-server -run TestRuntimeV0SupervisorPreparaAutomejoraTrasIdleV0`.
La causa era una publicacion idle `scheduled` guardada para deduplicacion que no
se invalidaba cuando el intento terminaba por error o `prepare_failed`. Un retry
legitimo tras cooldown con el mismo `request_ref` podia quedar silenciado antes
de preparar automejora. El cierre introduce
`resetIdleSelfImprovementPublicationLockedV0` en el tracker y lo aplica desde
`MarkIdleSelfImprovementErrorV0` y
`MarkIdleSelfImprovementPrepareFailedV0`; el test async tambien espera el drenaje
del trabajo previo antes de forzar el retry. Evidencia focal:
`TestStatusTrackerV0IdleSelfImprovementErrorLiberaPublicacionParaRetryV0`,
`TestStatusTrackerV0PrepareFailedLiberaPublicacionParaRetryV0` y
`go test -count=100 ./modulos/orquesta-server -run TestRuntimeV0SupervisorPreparaAutomejoraTrasIdleV0`.

Cierre BUG-ORQ-20260703-156/157 2026-07-03 noche:
La limpieza de backends Goal no faltaba, pero no se ejecutaba en todas las
salidas: si shutdown acababa en `async_work_timeout`, `RuntimeV0` devolvia error
antes de llamar a los hooks de composicion, dejando posible `codex app-server`
tmux bajo el runtime temporal. El cierre garantiza hooks una vez por salida con
contexto fresco best-effort, tambien en `shutdownRuntimeV0` y
`stopRuntimeAfterServeClosedV0` cuando hay timeout. Para el harness background,
el contrato vigente queda reforzado: `orquesta-server run` no es daemon; los
wrappers versionados deben usar `orquesta-server start/status/stop` o harness
aislado con PID/trap/cleanup. El helper comun que espera readiness desde state
rechaza ahora un state con `pid` muerto aunque el `addr` responda readiness.
Evidencia focal: `TestRuntimeV0ShutdownTimeoutPublicaStopTimeoutV0`,
`TestRuntimeV0ShutdownEsperaPreparacionIdleAntesDeStoppedV0`,
`TestRuntimeV0CompactaShutdownHooksV0`,
`TestSmokeCommonReadinessStateFileRechazaPIDMuertoV0`,
`TestSmokeCommonShutdownCleanupMataAppServerPropioSinBaseURLV0` y
`bash -n scripts/lib/smoke_common.sh`.

Avance BUG-ORQ-20260701-088 / alto consumo goal-first 2026-07-03 noche:
El `GoalObserver` residente ya no limita el corte automatico al goal de
automejora idle. Si observa un goal-first `running` con alto consumo informado
por el backend Codex y sin artefacto util publicable, distingue
`checkpoint_only_high_consumption` cuando solo hay checkpoint y
`goal_active_no_checkpoint_high_consumption` cuando no hay checkpoint; en ambos
casos persiste `GoalWorkState` como `blocked`/`NeedsRework`, conserva refs de
evidencia, pide stop cooperativo por `GoalStopper`/run-control con accion
`replan_narrow_context` e idempotencia por run. No bloquea si ya existe
artefacto no-checkpoint o receipt de dominio. Evidencia:
`TestRuntimeV0GoalObserverAltoConsumoCheckpointOnlyPideStopCooperativoV0`,
`TestRuntimeV0GoalObserverAltoConsumoSinCheckpointPideStopCooperativoV0`,
`TestRuntimeV0GoalObserverNoParaSiHayArtefactoUtilV0` y
`go test -count=1 ./modulos/orquesta-server`. `BUG-088` seguia abierto en ese momento hasta
smoke real que confirme la ruta completa alto consumo/checkpoint -> segundo
artefacto o replan sin app-server residual.

Cierre funcional BUG-ORQ-20260701-088 2026-07-03 noche:
Se anadio el smoke real opt-in
`scripts/smoke_goal_first_checkpoint_only_high_consumption_real.sh`, separado
del smoke normal Nueva App. El backend real `app_server_tmux` toma ahora el
umbral de uso alto desde
`ORQUESTA_AUTOPROGRAMMING_CHECKPOINT_ONLY_HIGH_CONSUMPTION_TOKENS`, conservando
100000 tokens como default si no se configura. Evidencia real cerrada:
`ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_REAL_CONFIRM=1`
`ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_CODEX_EXECUTION_CONFIRMED=1`
`ORQUESTA_GOAL_FIRST_SMOKE_POLLS=50`
`./scripts/smoke_goal_first_checkpoint_only_high_consumption_real.sh` produjo
`smoke_goal_first_high_consumption_real=ok`,
`bug088_path=second_artifact_or_partial_artifacts`,
`recommended_action=review_partial_artifacts`,
`app_server_tmux_processes_alive=0` y
`smoke_root=/tmp/orquesta-goal-first-app-server.lwNV5d`.
El `observe_response.json` conserva
`codex_app_server_goal_status_active_high_token_usage tokens_used=13512`,
`evidence-ref-goal-observer-no-checkpoint-high-consumption`,
`evidence-ref-goal-observer-high-consumption-stop-requested` y
`evidence-ref-goal-cooperative-stop-requested-run-control`; el write-set
contiene `generated-apps/checkpoint_started_bug088.txt` y
`generated-apps/bug088_second_artifact.txt`. No quedaron procesos
`orquesta-server run`, `codex app-server --listen` ni `codebase-memory-mcp`.
Runbook:
`docs/runbooks/smoke_goal_first_checkpoint_only_high_consumption_real_2026-07-03.md`.
Estado: `BUG-088` queda cerrado funcionalmente para la ruta real acotada de
alto consumo -> segundo artefacto recuperable o replan/stop gobernado sin
app-server residual. Los residuales nuevos `BUG-ORQ-20260703-161` y
`BUG-ORQ-20260703-162` quedan cerrados en el corte posterior de la misma noche.

BUG nuevo `BUG-ORQ-20260703-161` (cerrado):
`observe_goal` puede publicar uso alto y stop cooperativo sin proyectar todavia
los ficheros directos del smoke como `artifact_refs`, aunque existan en el
write-set (`generated-apps/checkpoint_started_bug088.txt` y
`generated-apps/bug088_second_artifact.txt`). Evidencia:
`/tmp/orquesta-goal-first-app-server.uW9So4/observe_response.json` con
`summary=codex_app_server_goal_status_active_high_token_usage`,
`evidence-ref-goal-observer-high-consumption-stop-requested`,
`artifact_refs=null`, y ficheros presentes bajo
`/tmp/orquesta-goal-first-app-server.uW9So4/project/generated-apps/`.
Hipotesis: hay desfase entre scanner/materializer de artefactos y ficheros
directos del write-set generados por el agente; revisar si se debe proyectar
`generated-apps/*.txt` de smoke como artefacto parcial o declarar esa ruta como
evidencia de harness.

Cierre 2026-07-03 noche: el scanner de artefactos materializados reconoce de
forma estrecha `checkpoint_started*.txt` como checkpoint y `*_artifact.txt` como
artefacto materializado, sin aceptar `.txt` generico. Ademas
`orquesta.autoprogramming.observe_goal.v0` reutiliza el enriquecimiento de
materialized refs de `orquesta.apps.observe_director_goal.v0`, por lo que la
ruta del smoke no pierde `artifact_refs` frente a la ruta de apps. Evidencia:
`TestStackGoalMaterializedRefsSourceV0DetectaArtefactosTxtBUG088V0` y
`TestCodexStackAutoprogrammingPrepareRunAPIV0GoalReadyLanzaGoalFirstSinColaLegacy`.

BUG nuevo `BUG-ORQ-20260703-162` (cerrado):
En ramas de fallo del smoke alto consumo, `/api/v0/server/shutdown` pudo
devolver `status=ready`, `shutdown_ready=true` sin `exit_pending/pid`. El
cleanup de procesos fue efectivo, pero el contrato HTTP vuelve ambiguo el
resultado para wrappers que necesitan distinguir drenaje operativo de salida
programada del proceso temporal. Evidencia:
`/tmp/orquesta-goal-first-app-server.GG1Btr/shutdown_response.json` y
`/tmp/orquesta-goal-first-app-server.dmyZi0/shutdown_response.json`; ambos sin
`exit_pending/pid`. No quedaron procesos residuales. Hipotesis: la ruta con
`cleanup_goal_backends=true` y servidor temporal puede publicar `ready` sin
pasar por el mismo wrapper de salida programada que cerraba `BUG-146`.

Cierre 2026-07-03 noche: la causa era el snapshot previo de trabajo activo
tomado antes del cleanup HTTP. Cuando el handler limpiaba el backend y devolvia
`ready` con evidencia `evidence-ref-shutdown-goal-backend-cleanup-requested`,
la proyeccion podia reinyectar el snapshot stale y dejar la respuesta original
`shutdown_ready=true` sin `exit_pending/pid`. El runtime ahora no hereda ese
snapshot previo si el `ready` trae evidencia de cleanup de backend; conserva la
proteccion general contra falsos `ready` sin esa evidencia. Evidencia:
`TestRuntimeV0ServerShutdownReadyTrasCleanupNoHeredaSnapshotPrevioActivoV0`,
`TestRuntimeV0ServerShutdownReadyNoBorraSnapshotPrevioActivoV0` y
`TestRuntimeV0ServerShutdownConflictSinCuerpoConservaSnapshotPrevioActivoV0`.

BUG nuevo `BUG-ORQ-20260704-176` (cerrado):
Durante la ampliacion de TAREA-8.1, añadir metadata nueva directamente en
`cmd/orquesta-server/server_env_registry_v0.go` rompio el ratchet T90:
`TestResidualGoFileBudgetT90V0` fallo porque el fichero subio a 903 lineas
(`fichero inmanejable >900 lineas`). Causa estructural: el registro central ya
era un hotspot y no debe recibir nuevas familias completas. Cierre:
`codex_usage_accounting` y `worktree_snapshot` registran metadata en ficheros
familiares pequeños (`codex_usage_accounting_env_registry_v0.go` y
`worktree_snapshot_budget_env_registry_v0.go`), dejando
`server_env_registry_v0.go` en 878 lineas. Evidencia:
`go test -count=1 ./cmd/orquesta-server -run 'ResidualGoFileBudget|CodexUsage|WorktreeSnapshot|EnvRegistryAST|EnvVarsOrquestaRatchet' -v`.

BUG nuevo `BUG-ORQ-20260704-177` (cerrado):
El toolbelt y las instrucciones de agentes anunciaban
`POST /api/v0/codebase/query`, pero `buildServerAppHandlerV0` no montaba esa
ruta directa; solo exponia `codebase/status`. En servidor real temporal, la
peticion caia al handler general y devolvia 400, aunque el binding MCP de query
existia. Causa estructural: divergencia entre contrato publicado y wiring HTTP
directo del stack. Cierre: `cmd/orquesta-server/stack.go` registra
`orquestamcp.NewMCPCodebaseQueryHTTPHandlerV0` cuando existe
`MCPTransportBindings.CodebaseQuery`. Evidencia:
`TestBuildServerAppHandlerV0CodebaseQueryPublicoUsaBindingDirecto` y smoke
Orquesta temporal `codebase_query=passed`.

BUG nuevo `BUG-ORQ-20260704-178` (cerrado):
El broker `fallback_rg` descartaba el scope explicito `"."` y caia a los
defaults `cmd`, `modulos`, `docs`, `scripts`, `AGENTS.md`. En apps externas o
proyectos temporales con codigo en la raiz, `POST /api/v0/codebase/query`
devolvia `code_context_proveedor_error` aunque el fichero existiera. Causa
estructural: normalizacion demasiado estricta de scopes recuperables; `"."`
debe significar raiz del proyecto, no default canonico del repo Orquesta.
Cierre: `serverRGCodeContextScopesV0` conserva `"."` y sigue rechazando scopes
vacios, `..` y rutas absolutas. Evidencia:
`TestServerRGCodeContextProviderV0ScopePuntoBuscaRaizDelProyecto` y smoke
Orquesta temporal con `scope:["."]` -> `codebase_query=passed`.

BUG nuevo `BUG-ORQ-20260704-179` (cerrado):
Durante el smoke Orquesta de config canonica de limites, `/api/v0/server/status`
publico devolvia `ORQUESTA_SERVER_DRAIN_MAX_COMMANDS` como `redacted` y
`sensitive=true`, aunque es un limite numerico publico de supervisor. Causa
estructural: la redaccion publica por fragmentos trataba cualquier clave con
`COMMAND` como sensible y atrapaba tambien `*_MAX_COMMANDS`. Cierre: la capa
publica de `orquesta-server` conserva la redaccion de comandos reales, pero
exceptua claves numericas `*_MAX_COMMANDS` no marcadas explicitamente como
sensibles. Evidencia: `TestPublicServerEffectiveConfigV0NoRedactaLimitesNumericosMaxCommandsV0`
y smoke Orquesta temporal de config de limites.

BUG nuevo `BUG-ORQ-20260704-180` (cerrado):
Durante el smoke real de `orquesta-server start --config`, el comando devolvia
`readiness_timeout` aunque el daemon quedaba vivo, `running` y
`startup_ready`. Causa estructural: `start` usaba readiness estricta como unica
señal de arranque, y esa readiness puede responder 503 cuando hay diagnosticos
de conectores externos degradados (`external_work_goal_backend_required`) aunque
la disponibilidad del servidor ya sea `running/server_ready`. Cierre:
`serverReadinessOKV0` acepta para el arranque el payload 503 con
`startup_ready=true`, `status=running` y `availability_status=running`; la
readiness estricta sigue devolviendo 503 para consumidores que necesiten todos
los conectores listos. Evidencia:
`TestServerReadinessOKV0AceptaStartupReadyConReadinessDegradadaV0` y smoke real
`orquesta_start_config_snapshot_smoke=passed`.

BUG nuevo `BUG-ORQ-20260704-181` (cerrado):
Durante el smoke real de `rails_security` + `egress_sanitizer` con
`orquesta-server start --config`, el daemon hijo publicaba en `/status` las
claves de rails (`ORQUESTA_SECURITY_MODE`, `ORQUESTA_RAILS_MODE` y
`ORQUESTA_DETAIL_PROHIBITED_RAILS*`) con `source=explicit`, aunque el valor
procedia del snapshot canonico de `orquesta.config.json`. Causa estructural:
`start` proyecta defaults/derivados al entorno del daemon para estabilizar el
arranque, y el proceso hijo no distinguia una env proyectada por el padre de un
override manual. Cierre: `configSettingSourceFromConfigOrProjectConfigV0`
reclasifica como `config_file` solo cuando el proceso lee el snapshot daemon
`state/config-snapshots/orquesta.config.json` y la env coincide con el valor
efectivo normalizado del fichero; un `run --config` manual con env explicita
sigue apareciendo como `explicit`. Evidencia:
`TestServerEffectiveConfigV0ConservaFuenteFicheroTrasProyeccionDaemonRails`,
focal `Rails|EgressSanitizer|EnvRegistryAST|EnvVarsOrquestaRatchet|ResidualGoFileBudget|DaemonStartEnvironment`
y smoke real `orquesta_rails_egress_config_smoke=passed`.

BUG nuevo `BUG-ORQ-20260704-182` (cerrado):
Durante la primera ejecucion real de `scripts/orquesta_auditoria_codigo.sh`, la
herramienta recorria `**/*.go` desde la raiz del repositorio y solo despues
filtraba `cmd/` y `modulos/`; en un workspace grande eso podia dejar la
auditoria viva demasiado tiempo antes de producir evidencia. Causa
estructural: auditoria de codigo implementada como barrido global, no como
consulta acotada al write-set canonico del repo. Cierre: el script usa
`iter_project_go_files()` y solo camina `cmd/` y `modulos/` desde el origen.
Evidencia: `scripts/test_orquesta_auditoria_codigo.sh` y
`scripts/test_orquesta_smoke_nightly.sh`.

BUG nuevo `BUG-ORQ-20260704-183` (cerrado):
Durante la implementacion de la preparacion automatica del analizador para
goals de codigo, la auditoria fresca `scripts/orquesta_auditoria_codigo.sh`
detecto que `helper_duplicate_definitions` subia de 288 a 289 por introducir un
nuevo helper `compact*` local en `orquesta-app-codex-stack`. Causa estructural:
un parche nuevo volvia a copiar la familia de helpers que TAREA-9 precisamente
quiere reducir. Cierre: renombrado el helper local a
`uniqueCodeContextGoalStringsV0` para no incrementar la familia `compact*` y
mantener el ratchet de helpers. Evidencia: auditoria fresca posterior y focales
de `orquesta-app-codex-stack`.

BUG nuevo `BUG-ORQ-20260704-184` (cerrado):
Durante la verificacion de TAREA-9, `scripts/orquesta_auditoria_codigo.sh`
cayo a `docs/auditoria_codigo_deadcode_2026-07-04.txt` aunque `deadcode`
estaba instalado en `/home/alberto/go/bin/deadcode`: el script solo miraba
`PATH`, no `GOBIN`/`GOPATH/bin`. Causa estructural: una herramienta Go
instalada de forma canonica podia quedar fuera de la auditoria viva y reactivar
el snapshot historico de 1188 candidatos como falso verde. Cierre: el auditor
resuelve `deadcode` por `PATH`, `GOBIN` y `GOPATH/bin`, con test que fuerza
`GOPATH/bin/deadcode` sin `PATH`. Evidencia:
`scripts/test_orquesta_auditoria_codigo.sh` y auditoria fresca
`code_audit_source=deadcode_tool`, `deadcode_candidates=1228`,
`helper_duplicate_definitions=288`, `orphan_modules=1`,
`large_files_over_800=17`.

BUG nuevo `BUG-ORQ-20260704-185` (cerrado):
El nightly smoke aceptaba reportes de auditoria incompletos como no regresion:
si faltaban metricas, el ratchet convertia ausentes en `0`, y tampoco validaba
`schema_version` ni rechazaba `deadcode_source=snapshot_file`. Causa
estructural: el ratchet comparaba numeros sin validar la calidad del reporte,
con riesgo de publicar verde una auditoria obsoleta o corrupta. Cierre:
`scripts/orquesta_smoke_nightly.sh` exige schema
`orquesta_code_audit.v0`, metricas requeridas enteras y rechaza fuentes
`snapshot_file`, `unavailable` y `deadcode_tool_failed`. Evidencia:
`scripts/test_orquesta_smoke_nightly.sh` cubre regresion, snapshot, schema
invalido y metricas ausentes.

BUG nuevo `BUG-ORQ-20260704-186` (cerrado):
El runtime Gemini no-goal podia quedar con `OutputFormat` vacio aunque Claude y
el backend goal Gemini usan `text` por defecto. Causa estructural: dos caminos
de configuracion del mismo conector no compartian default operativo. Cierre:
`geminiRuntimeConfigV0` usa `envOrDefaultV0(ORQUESTA_GEMINI_OUTPUT_FORMAT,
"text")` y se anade test especifico. Evidencia:
`GOFLAGS=-buildvcs=false go test -count=1 ./cmd/orquesta-server -run 'TestGeminiRuntimeConfigV0|GoalBackend|Claude|Gemini'`.

BUG nuevo `BUG-ORQ-20260704-187` (cerrado):
El guard de `external_work` OPES y el descubrimiento del topic registry
inyectaban por defecto `/home/alberto/Trabajo/OPES` cuando no habia
`ORQUESTA_OPES_PROJECT_WORKDIR`. Causa estructural: una ruta local de una
instalacion concreta se habia convertido en default de composicion, reduciendo
portabilidad hexagonal del conector. Cierre: ambos caminos consumen ahora la
fuente canonica `opes.project_workdir` de `orquesta.config.json` o su env
`ORQUESTA_OPES_PROJECT_WORKDIR`; si no existe, no inventan ruta local y la
regla OPES queda inactiva hasta configuracion explicita. Evidencia:
`GOFLAGS=-buildvcs=false go test -count=1 ./cmd/orquesta-server -run 'TestExternalWorkRunProjectWorkDirGuardConfigV0|TestOPESTopicRegistryConfig|TestOPESTopicRegistryEffectiveConfig|TestServerConfigFromEnvV0ContextoOPES|TestServerConfigFromEnvV0PermiteOPES'`
y `GOFLAGS=-buildvcs=false go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestExternalWorkRunProjectWorkDirGuard'`.

BUG nuevo `BUG-ORQ-20260704-188` (cerrado ampliado):
Los prompts goal-first de Claude/Gemini estaban fijados en español dentro del
adaptador runtime, sin forma de elegir idioma desde composicion. Causa
estructural: el contrato i18n existia para docs/errores, pero el adaptador de
proveedor conservaba textos operativos monolingues. Cierre acotado: se anade
`PromptLocale` a los backends goal-first Claude/Gemini, builders
`Build*GoalPromptWithLocaleV0`, protocolo durable localizado `es-ES/en-US` y
config canonica `goal_backend.prompt_locale` en `orquesta.config.json`
cableada a `file_control` y `process`, sin nuevas variables `ORQUESTA_*`.
Default compatible: `es-ES`. Evidencia:
`GOFLAGS=-buildvcs=false go test -count=1 ./modulos/orquesta-runtime-claude ./modulos/orquesta-runtime-gemini ./cmd/orquesta-server -run 'Test(Build(Claude|Gemini)GoalPromptWithLocale|ClaudeGoalBackendV0Launch|GeminiGoalBackendV0Launch|ServerGoalBackendFromEnvV0(Claude|Gemini)FileControl|ClaudeRuntimeConfigV0|GeminiRuntimeConfigV0)'`.
Ratchet de envs y auditoria de deuda conservados: `EnvVarsOrquestaRatchet`
verde, `scripts/orquesta_metricas_deuda.sh --json` reporta
`env_vars_orquesta=512` sin aumento por BUG-188, y
`scripts/orquesta_auditoria_codigo.sh` mantiene
`deadcode_candidates=1228` / `helper_duplicate_definitions=288`.
Ampliacion Codex 2026-07-09: los prompts legacy de agente Claude/Gemini
(`Build*AgentPrompt*`) aceptan `prompt_locale` por perfil, conservan default
español y generan protocolo operativo en ingles para `en-*`. El locale se
propaga desde `goal_backend.prompt_locale` a backends `process` y desde la
composicion `orquesta-app-codex-stack` a los perfiles reales Gemini/Claude.
Evidencia: `go test -count=1 ./modulos/orquesta-runtime-claude
./modulos/orquesta-runtime-gemini ./modulos/orquesta-app-codex-stack` y
`go test -count=1 ./cmd/orquesta-server -run
'Test(ServerGoalBackendFromEnvV0|ClaudeRuntimeConfigV0|GeminiRuntimeConfigV0)'`.
Residual: no hay smoke real de proveedor por idioma; queda como validacion
operativa opt-in cuando haya credenciales/cuota.

BUG nuevo `BUG-ORQ-20260704-189` (cerrado):
El smoke REST `scripts/smoke_orquesta_server_rest_director.sh` fallaba en
readiness con HTTP 503 aunque el servidor estuviera arrancado con
`startup_ready=true`, `status=running` y `availability_status=running`. Causa
estructural: el harness shell seguia usando readiness estricta HTTP 200 como
unica senal de arranque, mientras el contrato vigente permite readiness
operativa degradada por conectores externos y los tests Go ya aceptaban ese
caso para `start`. Cierre: `scripts/lib/smoke_common.sh` anade
`smoke_orquesta_readiness_ok`, que acepta 2xx o JSON 503 con startup listo y
proceso running; el smoke REST usa ese helper. Evidencia:
`GOFLAGS=-buildvcs=false go test -count=1 ./cmd/orquesta-server -run 'TestSmokeCommon(Readiness|Shutdown)'`
y `ORQUESTA_LEGACY_DIRECTOR_LOOP_SMOKE_CONFIRM=1 GOFLAGS=-buildvcs=false ./scripts/smoke_orquesta_server_rest_director.sh`
verde con `POST /api/v0/apps/director -> HTTP 200`, stats con
`agents_started=1`, progress/usage verificados y parada limpia.

BUG nuevo `BUG-ORQ-20260704-190` (cerrado):
El productor OPES `finalpkg` podia quedar habilitado en modo live
(`dry_run=false` + confirmacion) usando `course_id`, `template_run_ref` y
`template_topic_id` hardcodeados de una instalacion OPES concreta si el
operador no los declaraba explicitamente. Causa estructural: los defaults de
fixture/dry-run se aplicaban antes de `validateOPESRegistryFinalPkgConfigV0`,
asi que la validacion no podia distinguir una configuracion live incompleta.
Cierre: `CourseID`, `TemplateRunRef` y `TemplateTopicID` solo reciben fallback
en `dry_run=true`; en live quedan vacios si no vienen de env/config canonica y
la validacion devuelve `opes_registry_finalpkg_config_incomplete` con las claves
faltantes. Evidencia:
`GOFLAGS=-buildvcs=false go test -count=1 ./cmd/orquesta-server -run 'TestOPESRegistryFinalPkgConfig|TestOPESRegistryFinalPkgDryRunSelectsCandidatesWithoutSubmit|TestOPESRegistryFinalPkgPostsExternalWorkRunEnvelope'`
y smoke temporal por API publica de Orquesta: servidor `running`,
`POST /api/v0/apps/intake/guided-turn` -> schema
`web_nueva_app_intake_guided_response.v0`, `wizard_questions=10`,
`POST /api/v0/autoprogramming/status` -> `estado=ok`, y stderr del loop
residente contiene bloqueo `opes-registry-finalpkg blocked` con
`ORQUESTA_OPES_REGISTRY_FINALPKG_COURSE_ID`,
`ORQUESTA_OPES_REGISTRY_FINALPKG_TEMPLATE_RUN_REF` y
`ORQUESTA_OPES_REGISTRY_FINALPKG_TEMPLATE_TOPIC_ID`. Residual: no cerrar
`BUG-058/066/075` sin smoke OPES temporal real con arbol tema -> derivados ->
paquete final y sin tocar OPES productivo.

## Pendientes de analisis agrupado

- Diagnostico residente de estado vivo parcialmente unificado: el endpoint
  `/api/v0/operational-status/query` ya puede consumir la fuente neutral de
  estado vivo con goals, procesos, runs, ACK/receipts y deliveries como
  contadores/referencias compactas. Queda pendiente smoke real amplio de
  observabilidad/control lento.
- Revisar endpoints largos: separar submit/ack/observe de operaciones que
  pueden colgar HTTP.
- Auditar todos los validadores OPES contra artefactos canonicos vs
  regenerables.
- Revisar write-set y consolidacion canonica para padres/subagentes.
- Protocolo remoto unico documentado en
  `docs/runbooks/protocolo_git_remoto_orquesta_2026-07-02.md`: bundle,
  checkout, identidad Git, patch, summary, recibo de integracion y no tocar
  produccion. Queda pendiente validarlo en servidor con remote Git canonico o
  flujo bundle/push oficial.
- Revalidar con smokes OPES temporales largos de audio/visual bajo entorno
  aislado; el contrato ejecutable de supuestos practicos ya publica rework
  causal por faltante y queda cubierto en BUG-ORQ-20260630-027.
- `app_server_proxy` queda decidido como valor historico no operacional: el
  servidor lo reconoce solo para devolver `opt_in_required` o
  `codex_goal_backend_proxy_diagnostic_not_operational`; los helpers que aun
  podian construir `app-server proxy` se retiraron en local el 2026-07-09.
- Internacionalizar prompts legacy de agentes Claude/Gemini si el contrato i18n
  se extiende tambien a flujos no goal-first.
- Wizard sigue parcial: esta tanda amplio packs de dominio y pruebas focales,
  pero faltan U1-U12 completos, T1-T8 efectivos, motor de exclusion runtime,
  ayudas HelpKey/ExampleKey, glosario y bot RAG antes de declararlo universal.
- OPES finalpkg queda mas seguro en config live, pero aun falta smoke temporal
  real end-to-end con external-work/observe, proveedor, derivados, cierre de
  paquete final y comprobacion de no reescritura tardia.

- Ejecucion del revisor 2026-07-05: scripts/smoke_opes_lifecycle_real.sh en
  verde fuera del sandbox (el goal quedo blocked solo por sockets loopback
  del sandbox): 24/24 job types del lifecycle cubiertos con receipts
  (goal_receipts_manifest_status=ok, covered=24/24), run_until=completed,
  final=finalize_temario_package, empty_after_final=true, cierres accepted y
  sin procesos residuales. Evidencia:
  /tmp/orquesta-opes-lifecycle-real-20260704T223204Z/out/derivatives/.
  Con esto el residual comun "smoke OPES temporal real" de BUG-058/066/075
  queda cubierto; los residuales que sigan abiertos deben citar un hueco
  concreto nuevo, no este smoke.
- Revalidacion Codex local 2026-07-09: al repetir
  `scripts/smoke_opes_lifecycle_real.sh` aparecio un bug del harness
  (`BUG-ORQ-20260709-196`) por stdout heredado del fake server `finalpkg`; el
  parche redirige stdout/stderr del server y el smoke completo vuelve a pasar:
  24/24 fases, `finalpkg_dry_run=false`, un POST fake a
  `run-ref-opes-a1-t002-finalpkg-20260612`, `settlement_status=settled_final` y
  sin procesos residuales. Evidencia local:
  `/tmp/orquesta-opes-lifecycle-real-20260709T065803Z/out/opes_lifecycle_result.json`.

BUG nuevo `BUG-ORQ-20260705-194` (cerrado en codigo local; residual operativo):
El field test OPES real post-G5 no podia ejecutarse desde el goal remoto por
falta de `required_settings` externas (`ORQUESTA_OPES_BASE_URL`,
`ORQUESTA_BASE_URL`, confirmacion OPES temporal/bridge y scope duro). Causa
operativa: Orquesta pidio una mision con efectos reales, pero el paquete goal no
transportaba la configuracion/scope necesaria para distinguir instancia temporal
de productiva ni para evitar colas ajenas. Resultado correcto:
`missing_required_settings`; no se inventario OPES real ni se creo temario.
Evidencia: `docs/runbooks/opes_real_field_test_2026-07-05.md` y
`cmd/orquesta-server/docs/orquesta_goal_result_goal-ref-task-autoprogramming-72bb10d53162-g04.json`.
Rework: abrir goal causal con OPES temporal/preproduccion confirmado,
`ORQUESTA_BASE_URL`, confirmacion de bridge, `limit=1` y scope por `job_ref` o
programa/tema/correlacion.

Avance Codex 2026-07-07: cerrada la parte de codigo local que mantenia abierto
el hueco de transporte del contrato. `cmd/orquesta-server` ahora publica en
`effective_config` las guardas OPES bridge relevantes
(`ORQUESTA_OPES_TEMPORAL_CONFIRM`, `ORQUESTA_OPES_BRIDGE_ENABLED`,
`ORQUESTA_OPES_BRIDGE_CONFIRM`, `ORQUESTA_OPES_BRIDGE_DRY_RUN`) junto a URL,
scope y limite. `orquesta-external-work-run` prioriza `required_settings`,
base URLs redactadas/configuradas, confirmaciones, `limit=1` y scope OPES duro
en `context_refs[input_field_value]`, y anade criterio de aceptacion: si falta
cualquier requisito, bloquear con `reason_code=missing_required_settings` sin
tocar colas ni OPES productivo. Pruebas verdes:
`go test -count=1 ./modulos/orquesta-external-work-run ./cmd/orquesta-server`.
Residual operativo: desplegar/sincronizar remoto y repetir field test solo con
instancia OPES temporal/preproduccion y scope real aportados por operador.
Comprobacion remota 2026-07-07: `srv1651826:/srv/orquesta-self/worktrees/pilot-remoto-1`
estaba en `0188739c`, 24 commits por detras de `origin/trabajo/plataforma-agentes`;
no habia un cierre mas avanzado publicado alli.

BUG nuevo `BUG-ORQ-20260705-195` (cerrado en codigo 2026-07-06):
El contrato de cierre pidio escribir el resultado durable en
`scripts/smoke_opes_lifecycle_real.sh/docs/orquesta_goal_result_...json`, pero
`scripts/smoke_opes_lifecycle_real.sh` es un fichero ejecutable, no un
directorio. Causa estructural: el generador de rutas de resultado durable puede
concatenar `docs/` bajo un artefacto de write-set que no es directorio. Cierre
temporal: conservar el script, escribir el resultado en
`cmd/orquesta-server/docs/` y devolver `blocked` si el validador exige la ruta
imposible. Rework: normalizar la ruta durable a un directorio real del
write-set antes de lanzar el goal. Cierre aplicado por Codex local: el runtime
Codex Goal ya no cuelga `docs/orquesta_goal_result_*.json` bajo scopes que
parecen fichero `.go` o `.sh`; si existe un siguiente scope directorio lo usa
para el JSON durable, y si todos los scopes son ficheros no pide una ruta de
fichero imposible. Evidencia:
`TestBuildCodexGoalPromptV0NoCuelgaResultadoDurableBajoFicherosCodigoOScriptV0`,
`TestBuildCodexGoalPromptV0SaltaFicheroCodigoYUsaSiguienteDirectorioV0` y
`go test -count=1 ./modulos/orquesta-runtime-codex-goal`.

BUG nuevo `BUG-ORQ-20260705-193` (cerrado en codigo 2026-07-07):
El goal `task-remote-telegram-inodo-connector-20260705` exigia consultar
`orquesta.codebase.query.v0` antes de leer codigo completo, pero el runtime de
este Codex Goal no expuso recursos ni templates MCP para ese broker. Causa
operativa probable: toolbelt/broker no inyectado en el paquete goal aunque el
prompt lo declaraba obligatorio. Mitigacion aplicada: busquedas `rg` acotadas y
lecturas parciales, sin arrancar indexadores propios ni `codebase-memory-mcp`.
Cierre aplicado por Codex local: el prompt estable de Codex Goal conserva el
broker como primera opcion, pero lo formula como disponible por toolbelt MCP
local o por HTTP `POST /api/v0/codebase/query`; si el broker no esta inyectado,
el agente debe continuar con `rg`/`sed` acotados, dejar evidencia
`codebase_broker_unavailable` y no arrancar indexadores propios ni
`codebase-memory-mcp`. Evidencia:
`TestBuildCodexGoalStartPacketV0AnalizadorDegradaSiBrokerNoInyectadoV0` y
`go test -count=1 ./modulos/orquesta-runtime-codex-goal`.

Nota MEJ-106 2026-07-05/2026-07-09: el adaptador remoto Telegram/Inodo entro
primero con una superficie temporal `ORQUESTA_TELEGRAM_OPERATOR_*`; esa ventana
queda retirada. La configuracion vigente es `telegram_operator.*` en
`orquesta.config.json`, con token/chats/target redactados en `effective_config`
y sin ampliar el ratchet de envs.
# Nota 2026-07-05 canal operador-Director

Durante la implementacion focal del canal operador-Director v0 no se observo
bug de dominio del canal. Si el store en memoria de composicion se usa fuera de
tests, debe registrarse como incidencia operativa porque produccion requiere un
`OperatorDirectorExchangeStorePortV0` durable. La incidencia de entorno de tests
queda registrada abajo como `BUG-ORQ-20260705-196`.

BUG nuevo `BUG-ORQ-20260705-196` (cerrado en goal actual):
Las pruebas Go focales fallaron inicialmente antes de compilar porque `GOCACHE`
apuntaba a `/srv/orquesta-self/runtime/server-latest/go-cache`, ruta de solo
lectura en este goal remoto. Mitigacion/cierre aplicado: reejecutar tests con
`TMPDIR` y `GOCACHE` aislados bajo `.orquesta-runtime/`, sin tocar caches
compartidas ni runtime productivo. Rework general si reaparece: el launcher
goal-first debe proyectar caches Go escribibles dentro del workspace/write-set
o declarar explicitamente la cache read-only como no usable por tests.

BUG nuevo `BUG-ORQ-20260705-197` (cerrado en goal actual):
El canal operador-Director podia dar falso cierre: el contrato MCP
`orquesta.operator.director.message.v0` y tests fake existian, pero
`buildStackMCPTransportBindingsV0` no inyectaba `OperatorDirectorMessage` en el
stack real, de modo que el tool quedaba registrado sin puerto funcional de
composicion. Cierre aplicado: el stack cablea `OperatorDirectorChannelServiceV0`
con store de exchange de composicion y dispatcher real sobre los executors MCP
existentes para `status`, `queue`, `observe`, `launch`, `handoff` y
`stop/control`; `stop/control` queda bloqueado si falta confirmacion explicita o
puerto real. Evidencia: `TestBuildStackOperatorDirectorMessageV0CableaServicioRealConQueueYControlSeguro`,
`TestOperatorDirectorMessageMCPV0RegistraToolYDelegaConStoreDurable` y
`TestMCPOperatorDirectorChannelServerJSONRPCV0ExponeToolYDevuelveAck`.

BUG nuevo `BUG-ORQ-20260705-198` (cerrado en codigo 2026-07-06):
El contrato de cierre de este goal pidio escribir el resultado durable en
`modulos/orquesta-app-codex-stack/stack_v0.go/docs/orquesta_goal_result_goal-ref-task-autoprogramming-90e96a780f43-g01.json`,
pero `stack_v0.go` es un fichero Go, no un directorio. Causa estructural
equivalente a `BUG-ORQ-20260705-195`: el generador de rutas durable concatena
`docs/` bajo una entrada de write-set que puede ser fichero. El codigo y tests
del goal cierran, pero el ACK debe quedar `blocked` si Orquesta exige esa ruta
exacta. Rework: normalizar la ruta durable a un directorio real del write-set o
anadir un artefacto JSON autorizado como ruta independiente. Cierre aplicado por
Codex local junto con `BUG-ORQ-20260705-195`: `codexGoalResultFilePathV0` salta
write-sets `.go`/`.sh` al construir la ruta durable, conserva el cierre por
`ORQUESTA_GOAL_RESULT_V0` y solo pide fichero JSON durable cuando puede
ubicarlo bajo un scope directorio autorizado. Evidencia:
`TestBuildCodexGoalPromptV0NoCuelgaResultadoDurableBajoFicherosCodigoOScriptV0`,
`TestBuildCodexGoalPromptV0SaltaFicheroCodigoYUsaSiguienteDirectorioV0` y
`go test -count=1 ./modulos/orquesta-runtime-codex-goal`.

- BUG-ORQ-20260705-TELEGRAM-NOLLM-ACCEPTED-INVISIBLE: `prepare-run` acepta `request-ref-remoto-telegram-nollm-runtime-20260705-001` pero no aparece en `autoprogramming/status`; control movil no-LLM no esta desplegado y Hermes LLM falla por 429. Avance 2026-07-06: el endpoint `POST /api/v0/operator/telegram/update` y el sender Bot API directo ya existen; D1 cierra en codigo la invisibilidad de `prepare-run` legacy con `RunRef` en lecturas de cola, `Reason=autoprogramming_prepare_run`, diagnostico `autoprogramming_prepare_run_pending_dispatch` y readback obligatorio tras encolar. Actualizacion 2026-07-08: D1 accepted-invisible ya esta desplegado y verificado en remoto; el bloqueo restante es configuracion/operacion Telegram. La prueba real devolvio `404` porque el servidor vivo arranco sin `orquesta.config.json`, asi que la ruta opt-in no se monto. Fix local: `scripts/orquesta_server_ctl.sh` pasa `--config` desde `ORQUESTA_CTL_CONFIG` o desde `$ORQUESTA_CTL_WORKDIR/orquesta.config.json` cuando existe, cubierto por `scripts/test_orquesta_server_ctl.sh`. Queda deploy/sync, crear config canonica con token/chats autorizados, webhook o poller y validar desde Telegram real; resolver aparte la auth Codex 401 si se quiere que el agente programe. Ver `docs/incidencias/incidencia_orquesta_telegram_nollm_accepted_invisible_2026-07-05.md`. Estado: cerrada/verificada para accepted-invisible; abierta operativamente para Telegram real.
| BUG-ORQ-20260705-SUPERVISOR-SCHEDULER-PAYLOAD | supervisor/autoprogramacion | cerrado/verificado remoto | Supervisor remoto repetia `director_tick_input_build_invalido: field=scheduler_input.payload`; T137 rank 1 bloqueaba cola. Reabierto el 2026-07-08: el servidor activo en `/srv/orquesta-self/worktrees/pilot-remoto-1` volvio a fallar con el mismo campo al procesar T137 sin carril activo, con 1333 assessments y 1333 preguntas en snapshot; el outbox estaba despachado. | `docs/incidencias/incidencia_orquesta_supervisor_scheduler_payload_2026-07-05.md` | Cerrado 2026-07-08 con commit `d650d30e977380fc7c69771c83fc1d04cd023de8`: `orquesta-director-tick-input` aplica compactacion final por presion de payload sobre snapshot historico, conserva agentes en vuelo y refs causales, y anade `TestBuildDirectorSchedulerTickInputV0CompactaSnapshotSobredimensionadoSinCarrilActivo`. Pruebas: focales de tick-input/scheduler/cycle, `go test -count=1 ./...`, build remoto de `./cmd/orquesta-server`. Desplegado en `/srv/orquesta-self/runtime/orquesta-server-claude`, hash `3a402a9dc1ac78db19d1c3641b70c5f280b6d7675b658576521079d9afd7386e`; `/api/status` remoto: `startup_ready`, supervisor `ok`, `last_supervisor_error=null`. |
| BUG-ORQ-20260705-CODEX-HOME-TOKEN-INVALIDADO | runtime-codex/proveedor | abierto | Agente Orquesta falla antes de programar con `token_invalidated` y `refresh_token_invalidated` en `/srv/orquesta-self/codex-home`; en la sesion local 2026-07-07 dos subagentes fallaron con "access token could not be refreshed", misma clase de auth externa. | `docs/incidencias/incidencia_orquesta_codex_home_token_invalidado_2026-07-05.md` | Avance codigo local 2026-07-07: `orquesta-runtime-codex-appserver` clasifica `token_invalidated`, `refresh_token_invalidated`, `refresh_token_reused`, `token_expired` y "access token could not be refreshed" como `codex_app_server_provider_unauthorized`, para que status/diagnostico pidan reautenticacion en vez de error generico. Sigue abierto operativo: requiere reautenticacion del Codex CLI del servidor y relanzar tarea hasta ver `agent_ack.json` nuevo. |

BUG-ORQ-20260706-BUDGET-CONTRACT-DESALINEADO (cerrado en codigo local):
La auditoria estructural P1 detecto presupuestos de eventos/payload definidos
en privado por capas: pagina de lectura 250, pagina store 1000, lectura total
10000, maximo store 20000 y payload scheduler 256 KiB. Esa dispersion ya habia
producido falsos diagnosticos de presupuesto y riesgo de payload excesivo.
Cierre aplicado por Codex local en `029d0d2a1`: nuevo paquete neutral
`modulos/orquesta-orchestration-budget` con constantes canonicas, consumo desde
`orquesta-state-file`, `orquesta-orchestration-core`,
`orquesta-app-director-service`, `orquesta-director-scheduler` y
`orquesta-director-tick-input`; tests de coherencia `pagina <= lectura <= store`
y `snapshot <= payload scheduler`; tests focales de lectores paginados.
Estado: commit integrado en `trabajo/plataforma-agentes`; pendiente solo
despliegue/verificacion remota cuando haya cuota/auth.

BUG nuevo `BUG-ORQ-20260706-SUPERVISOR-EVENTS-BUDGET-PARKING` (cerrado en codigo):
Tras cerrar el falso presupuesto por pagina y el dedupe de eventos duplicados,
seguia abierta la resiliencia de cola: si un run supera de verdad el presupuesto
de historial, `events_full_history_budget_exceeded` podia tumbar el tick del
supervisor o reintentar el mismo candidato indefinidamente. Cierre aplicado por
Codex local: `orquesta-app-codex-stack` clasifica el error tipado
`nucleo_orquestacion_store/events.budget`, aparca el candidato con
`Outcome=run_oversized`, `QueueStatus=stopped`, `RescueReason=run_oversized_events_budget`
cuando el error ocurre en preparacion, completa `RunControl` como `stopped` sin
auto-resume y continua con el resto de la cola cuando el error aparece en drain.
Evidencia:
`TestCodexStackV0RunGlobalTickAparcaRunSobredimensionadoYContinuaColaV0`,
`TestCodexStackV0RunSobredimensionadoAparcadoNoSeReanudaEnPreparacionV0`,
`go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestCodexStackV0RunGlobalTick|TestCodexStackV0RunSobredimensionado|TestCodexStackV0RunGlobalSupervisor|TestCoordinate|TestCodexSupervisorRuntimeStateFromGlobalSupervisor'`
y `go test -count=1 ./modulos/orquesta-run-coordinator`. Residual operativo:
redeploy/verificacion viva en remoto cuando haya cuota/auth; ver
`docs/incidencias/incidencia_orquesta_supervisor_events_budget_2026-07-06.md`.

BUG nuevo `BUG-ORQ-20260706-MCP-WIZARD-BOT-SCHEMA-STALE` (cerrado):
Durante `go test -count=1 ./...`, el contrato real MCP fallaba porque
`orquesta.nueva_app.wizard.bot.v0` estaba registrado en el transporte, pero no
en `MCPTransportToolInputFieldsV0`; el servidor publicaba schema
`schema_stale`. Ademas el test de paridad del modulo MCP no trataba `[...]`
como nesting y leia los campos internos de `wizard_answers` como top-level.
Cierre aplicado: el DTO canonico del bot queda registrado, `user_text` es
required y el parser de test ignora contenido anidado en arrays. Evidencia:
`go test -count=1 ./modulos/orquesta-mcp ./cmd/orquesta-server -run 'TestMCPTransportToolInputSchemaV0CubreToolsRegistrados|TestMCPRealTransportV0InputSchemaSaleDeDTOCanonico'`.

BUG nuevo `BUG-ORQ-20260706-ENV-RATCHET-ROOT-DIVERGENCE` (cerrado):
Durante `go test -count=1 ./...`, el test raiz de presupuesto de variables de
entorno fallaba con medicion 514 frente a base 513, aunque el test equivalente
de `cmd/orquesta-server` ya aceptaba la excepcion documentada en bitacora/
inventario. Cierre aplicado: el test raiz mantiene el limite base 513 pero
reusa la excepcion documentada exacta `env_vars_orquesta_allow_increase_to=<n>`
sin subir el ratchet. Evidencia:
`go test -count=1 . -run TestEnvVarsBudgetMEJ106V0`.

BUG nuevo `BUG-ORQ-20260706-WIZARD-I18N-PLACEHOLDER` (cerrado):
El wizard de nueva app tenia cobertura de presencia de claves i18n, pero no
detectaba textos genericos usados como relleno en U1-U12/T1-T8:
`Plain English explanation for ...`, `Plain explanation for this wizard choice`
y `Explica esta opcion en lenguaje llano`. Esto producia falso verde: HTTP/MCP
y el glosario podian mostrar ayuda no accionable aunque
`TestWizardTodaOpcionTieneAyudaV0` pasara. Cierre aplicado por Codex: el
catalogo universal del wizard tiene ayudas reales es/en para opciones y
rationales U/T, el test prohibe placeholders conocidos y
`docs/wizard_glosario_generado.md` se regenero desde el catalogo. Evidencia:
`go test -count=1 ./modulos/orquesta-web -run 'TestWizard|TestNuevaApp.*Wizard'`.
Residual separado detectado por subagente Codex: faltaban paridad MCP de
`justification`, `comprehension_query`, `glossary_expanded` y tool MCP
`orquesta.nueva_app.wizard.bot.v0`. La paridad del tool existente queda
cerrada en `BUG-ORQ-20260706-WIZARD-MCP-CONTRACT-PARITY`; el tool MCP del bot
queda cerrado en `BUG-ORQ-20260706-WIZARD-MCP-BOT-TOOL`.

BUG nuevo `BUG-ORQ-20260706-WIZARD-MCP-CONTRACT-PARITY` (cerrado parcial):
El motor web del wizard ya soportaba `WizardAnswerV0.justification`,
`WizardAnswerV0.comprehension_query` y `glossary_expanded`, pero el DTO MCP
`orquesta.nueva_app.wizard.v0` no los exponia ni los normalizaba. Esto rompia
la paridad del contrato 10.3/11.5 para clientes MCP: una desviacion podia
llegar sin justificacion y una pregunta libre de comprension no podia viajar
por el tool. Cierre aplicado: `modulos/orquesta-mcp/nueva_app_wizard_tool_v0.go`
incluye los tres campos en input/schema/normalizacion, los tests MCP cubren el
transporte al puerto y `modulos/orquesta-app-codex-stack/stack_flow_v0_test.go`
verifica que el stack real conserva `justification`, `glossary_expanded` y
`comprehension_query` hasta el handler web. Evidencia:
`go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPNuevaAppWizard|TestNormalizeMCPNuevaAppWizard|TestNewMCPNuevaAppWizard|TestMCPTransportV0NuevaAppWizard'`
y `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestBuildStackV0CableaNuevaAppWizardMCPRico'`.
Residual no cerrado: superficie chat web visible si producto exige alternar
formulario/chat en UI; este corte solo cierra paridad del tool wizard existente.

BUG nuevo `BUG-ORQ-20260706-WIZARD-MCP-BOT-TOOL` (cerrado):
El bot determinista/RAG del wizard existia en `modulos/orquesta-web`, y el
servidor tenia adaptador LLM opt-in por `wizard_bot.*`, pero MCP no exponia
`orquesta.nueva_app.wizard.bot.v0`. Esto dejaba incompleto el contrato de la
seccion 12.4 para clientes IA: solo podian usar el formulario/wizard, no la
capa conversacional. Cierre aplicado: `modulos/orquesta-mcp` define contrato,
descriptor, normalizacion, transporte opt-in y tests del tool bot; el stack
Codex adapta ese puerto al bot web sin importar `orquesta-web` desde MCP; el
servidor inyecta el asistente LLM opcional cuando `wizard_bot.llm_enabled=true`
y conserva modo determinista sin proveedor. Evidencia:
`go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPNuevaAppWizardBot|TestMCPTransportV0(NuevaAppWizardBot|ExponeOperaciones)'`,
`go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestBuildStackV0(CableaNuevaAppWizardBotMCPDeterminista|ExponeBindingsMCPNativos)'`
y `go test -count=1 ./cmd/orquesta-server -run 'TestWizardBot|TestBuildStack|TestServerCodexGoal'`.
Residual anterior: no se implementaba aqui panel chat web visible. Queda
cerrado posteriormente en `BUG-ORQ-20260706-WIZARD-WEB-CHAT-PANEL`.

BUG nuevo `BUG-ORQ-20260706-WIZARD-WEB-CHAT-PANEL` (cerrado):
El bot del wizard ya existia como motor web y como tool MCP
`orquesta.nueva_app.wizard.bot.v0`, pero `/nueva-app` no exponia un panel de
chat visible ni una ruta HTTP propia para navegador. Esto dejaba incompleto el
contrato conversacional: el usuario humano solo podia usar botones/formulario
guiado, mientras clientes MCP si podian hablar con el bot. Cierre aplicado:
`modulos/orquesta-web` anade endpoint `web_nueva_app_wizard_bot_response.v0`,
panel `data-wizard-bot` en la UI, JS que conserva `session` y aplica
`turn_result`, i18n es/en y ratchet de render; `modulos/orquesta-http-gateway`
registra `POST /api/v0/apps/intake/wizard-bot` como ruta exacta de lectura bajo
`/api/v0/apps/`; `modulos/orquesta-app-gateway` y
`modulos/orquesta-app-codex-stack` cablean el puerto LLM opcional existente al
endpoint HTTP real. Evidencia:
`go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway ./modulos/orquesta-app-codex-stack -run 'TestNuevaAppWizardBotHTTPHandler|TestNuevaAppHTMLHandlerV0GET|TestWizardBot|TestNuevaAppHTMLHelpKeysV0CubrenClavesUsadasEnPlantilla|TestNewAppGatewayMux|TestPublicRouteMutability|TestPublicRouteManifest|TestGatewayRouteRegistrations|TestNewHTTPHandlerV0ExponeNuevaApp(IntakeGuidedTurn|WizardBot)|TestNewHTTPHandlerV0InyectaNuevaAppIntakeAssistant|TestNewHTTPHandlerV0PropagaFallback|TestBuildStackV0(CableaNuevaAppWizardBotMCPDeterminista|ExponeNuevaAppWizardBotHTTP|ExponeBindingsMCPNativos)'`
y `git diff --check`.

BUG nuevo `BUG-ORQ-20260709-WIZARD-U12-AUTONOMIA-PISADA` (cerrado):
Una revision read-only del wizard detecto que `wizard-u6-colaboracion` y
`wizard-u12-autonomia` escribian ambos en `agentes.autonomia`. Responder U6
cerraba U12 aunque U12 representaba otra decision del diseno, y responder U12
podia sobrescribir la autonomia elegida. Cierre aplicado: U12 pasa a
`wizard-u12-historico-versiones`, escribe en
`datos.operacion.restricciones`, ofrece `sin_historico`,
`versionado_basico` e `historial_completo`, y materializa restricciones
operativas sin tocar `agentes.autonomia`. El catalogo i18n es/en y
`docs/wizard_glosario_generado.md` quedan actualizados. Evidencia:
`TestWizardU6YU12NoCompartenCampoAutonomiaV0`,
`ORQUESTA_UPDATE_WIZARD_GLOSSARY=1 go test -count=1 ./modulos/orquesta-web -run TestWizardGlosarioGeneradoV0`
y `go test -count=1 ./modulos/orquesta-web`.

BUG nuevo `BUG-ORQ-20260706-SMOKE-SHUTDOWN-PAYLOAD-STALE` (cerrado):
Durante el smoke manual local del binario
`/tmp/orquesta-builds/orquesta-server-221a4f07d`, el servidor arranco y
respondio `healthz`/`api/status`, pero el apagado cooperativo fallo primero por
`idempotency_key_requerida` y despues por `requester_not_authorized` al usar
payloads manuales no alineados con el contrato vigente. Al revisar el helper
comun se detecto que `scripts/lib/smoke_common.sh` tambien llamaba
`POST /api/v0/server/shutdown` sin `idempotency_key`, de modo que los smokes
podian depender del fallback por senal aunque el endpoint HTTP estuviera bien.
Avance 2026-07-07: al lanzar el smoke real `app_server_tmux` se detectaron dos
POST directos stale adicionales en `scripts/smoke_goal_first_app_server_real.sh`
y `scripts/smoke_codex_required_test_runner_state_file.sh`; quedan corregidos
con `idempotency_key` y `requested_by=orquesta-director`, y la guarda
`TestScriptsConShutdownDirectoPidenContratoShutdownV0` impide nuevas llamadas
directas sin contrato. Cierre aplicado: el helper comun envia `idempotency_key`
y `requested_by=orquesta-director`; se anade test focal
`TestSmokeCommonShutdownCleanupEnviaContratoDirectorV0`. Evidencia:
`go test -count=1 ./cmd/orquesta-server -run 'TestSmokeCommonShutdownCleanupEnviaContratoDirectorV0|TestSmokeCommonShutdownCleanupMataAppServerPropioSinBaseURLV0|TestSmokeCommonReadiness'`,
`go test -count=1 ./cmd/orquesta-server -run 'TestScriptsConShutdownDirectoPidenContratoShutdownV0|TestScriptShutdownCurlCommandsV0|TestSmokeCommonShutdownCleanupEnviaContratoDirectorV0'`,
`go test -count=1 ./cmd/orquesta-server`, `git diff --check` y
`docs/runbooks/smoke_manual_orquesta_server_2026-07-06.md`.

BUG nuevo `BUG-ORQ-20260707-SMOKE-GOAL-FIRST-USAGE-LIMITED` (bloqueo externo):
El smoke real minimo `scripts/smoke_goal_first_app_server_real.sh` con backend
`app_server_tmux` arranca Orquesta, crea run goal-first y materializa checkpoint,
pero el proveedor corta el goal como `codex_app_server_goal_status_usageLimited`
en fase `brainstorming_arquitectura`; Orquesta lo proyecta como
`goal_status=blocked`, `closure_status=blocked`,
`recommended_action=inspect_goal_backend_limits` y
`codex_app_server_goal_provider_limited`. No se observaron procesos residuales
tras el cleanup; la carpeta de evidencia se saneo eliminando el `codex-home`
temporal con credenciales. Estado: no es cierre funcional del smoke real; queda
pendiente reintento cuando haya cuota/modelo operativo. Reintento acotado con
`ORQUESTA_CODEX_MODEL=gpt-5` tambien queda `blocked` con solo checkpoint y sin
cierre `accepted`, por lo que no se insiste para no consumir mas intentos.
Evidencia:
`docs/incidencias/incidencia_orquesta_smoke_goal_first_usage_limited_2026-07-07.md`.

BUG nuevo `BUG-ORQ-20260707-D6-PLACEHOLDER-REASON-CODE` (cerrado en codigo):
El cierre D6 de reason codes tenia un residual: el prompt Goal no exigia
`reason_code`, el app-server no derivaba `checkpoint_started` desde el
checkpoint temprano si no habia resultado final, y `director_stats` podia
publicar `LastResult.Summary` como `issue_code`. Eso reabria la clase de falsos
diagnosticos por substrings libres (`started`, `checkpoint_started; ...`,
`in_progress...`). Cierre aplicado por Codex local: JSON obligatorio con
`reason_code`, soporte de `reason_code` explicito en resultados app-server,
derivacion `checkpoint_started` validada por `goal_ref`/`external_goal_ref`,
y MCP `issue_codes` solo desde `GoalWorkIssue.Code`. Evidencia:
`TestBuildCodexGoalStartPacketV0IncluyeContratoDeDireccion`,
`TestMergeCodexAppServerGoalResultV0ProyectaReasonCodeExplicitoV0`,
`TestServerCodexAppServerGoalBackendV0ObservaCheckpointStartedComoReasonCodeV0`,
`TestMCPDirectorStatsToolExecutorV0NoUsaSummaryComoIssueCodeV0`,
`go test -count=1 ./modulos/orquesta-runtime-codex-goal ./modulos/orquesta-runtime-codex-appserver ./modulos/orquesta-mcp ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`
y `git diff --check`.

BUG nuevo `BUG-ORQ-20260701-RECEIPT-DESRECONCILIADO` (cerrado en codigo local):
La incidencia remota
`docs/incidencias/incidencia_orquesta_remoto_automejora_goal_first_receipt_desreconciliado_2026-07-01.md`
mostraba un goal `complete` cuyo `last_result` conservaba artefactos/receipt,
pero el fichero versionado habia desaparecido del worktree y `status` no lo
proyectaba como problema resoluble. Cierre local 2026-07-08: nuevo reason code
`terminal_artifact_missing_after_goal_complete`; `orquesta-app-codex-stack`
detecta `ArtifactPaths` declarados por un resultado terminal que faltan dentro
del write-set; `orquesta-mcp` lo publica en `observe_goal`, `director/stats` y
`autoprogramming/status` con accion `replan` para recuperar o rehacer el
artefacto, sin confundirlo con `artifact_paths_omitted_materialized`.
Evidencia: tests focales
`TestStackGoalMaterializedRefsSourceV0DetectaArtifactPathDeclaradoPeroBorradoV0`,
`TestMCPDirectorStatsToolExecutorV0GoalFirstProyectaArtifactPathDeclaradoPeroBorradoV0`,
`TestMCPAutoprogrammingStatusExecutorV0ArtifactPathDeclaradoPeroBorradoPideReplanV0`
y `git diff --check`. Residual operativo: desplegar/sincronizar y repetir
automejora residente cuando haya proveedor/cuota.

Avance `BUG-ORQ-20260701-065` / `BUG-ORQ-20260704-165` 2026-07-08
(borde local cerrado):
Un subagente detecto un caso parcial de shutdown: si `cleanup_goal_backends`
recibia dos backends goal, limpiaba uno y el otro seguia vivo, Orquesta podia
conservar `cleanup_requested` para el backend ya desaparecido y publicar una
accion bloqueante stale aunque esa identidad estuviera resuelta. Cierre local:
`orquesta-server-shutdown` compara identidades estables
`kind/run_ref/work_ref/external_work_ref` antes y despues del cleanup, marca
`cleanup_completed` solo para los works que desaparecen, y la compactacion de
`goal_actions` elimina acciones previas no terminales de una identidad que ya
tiene `cleanup_completed`. Evidencia:
`TestShutdownServerV0CleanupGoalBackendsParcialNoConservaAccionBloqueanteDeWorkLimpioV0`,
`go test -count=1 ./modulos/orquesta-server-shutdown`,
`go test -count=1 ./cmd/orquesta-server -run 'Test(RequestServerShutdownV0CoordinaDosGoalsActivosHastaGoalActionsResueltas|RequestServerShutdownV0ReadyNoSaltaGoalActionsSinActiveWork|RequestServerShutdownV0PostColgadoConsultaStatusAccionable|WaitServerShutdownReadyV0RepostColgadoRespetaDeadlineYDevuelveStatusAccionable)'`
y
`go test -count=1 ./modulos/orquesta-server -run 'Test(ShutdownProjectionFromHTTPV0ReadyConGoalActionsQuedaStopPending|ServerPublicStatusV0ExponeShutdownGoalActions|StatusTracker|ServerPublicStatus)'`.
Residual: no cierra el BUG-165 global de observabilidad/control largo con
proveedor real; queda pendiente smoke real residente amplio tras deploy/sync.

Avance `BUG-ORQ-20260704-165` 2026-07-09 (diagnostico vivo reducido):
`ResidentOperationalStatusSourceV0` acepta ahora una fuente neutral
`orquesta-estado-vivo` y la ruta residente de operational-status agrega
contadores/referencias compactas de evidencias de run, goal, proceso,
ACK/receipt y delivery. El stack Codex reutiliza su
`AutoprogrammingEstadoVivoSource`, ya usado por MCP, y lo pasa a
`orquesta-server` sin importar la composicion desde el modulo servidor. La
consulta aplica timeout de 500 ms, limita evidencias a 32, respeta presupuesto
de 24 contadores/20 referencias y degrada a warning `estado_vivo_unavailable`
si la fuente falla, sin tumbar el status. Evidencia:
`TestResidentOperationalStatusSourceV0AgregaEstadoVivoCompactoV0`,
`TestResidentOperationalStatusSourceV0EstadoVivoErrorNoTumbaDiagnosticoV0`,
`go test -count=1 ./modulos/orquesta-server` y focal de
`cmd/orquesta-server`. Residual: no cierra el smoke real amplio con proveedor
lento ni la auditoria completa de submit/ack/observe largos.

Avance `BUG-ORQ-20260704-165` / `BUG-ORQ-20260701-065/079` 2026-07-09
(harness real ampliado, no cerrado completo):
El smoke real de forced-stop con backend vivo ya cubria `runs/control`,
`observe` posterior y shutdown/cleanup, pero no guardaba snapshots de
`/api/v0/autoprogramming/status`. Cierre local: el harness consulta status
antes de `runs/control` y despues del `observe` terminal, exige que el run/goal
sea visible y que tras forced-stop no se publique como `running`. Evidencia:
`bash -n scripts/smoke_goal_first_app_server_real.sh scripts/smoke_goal_first_forced_stop_backend_real.sh`
y
`go test -count=1 ./cmd/orquesta-server -run 'TestSmokeGoalFirst(AppServerReal|ForcedStop)'`.
Revalidacion real posterior del mismo corte: smoke Codex `app_server_tmux`
ejecutado con proveedor real, `run_ref=run-spec-smoke-goal-first-bug088-req-smoke-goal-first-bug088-90afbda75e7d98ff0be34efe74150090`,
`external_goal_ref=019f468d-b68f-7202-ad7a-0816a63663ae`,
`autoprogramming_status_before_forced_stop_visible=true`,
`run_control_status=stopped`, `run_control_goal_status_after=blocked`,
`observe_after_forced_stop_goal_status=blocked`,
`autoprogramming_status_after_forced_stop_not_running=true` y
`app_server_tmux_processes_alive=0`. Evidencia saneada:
`/tmp/orquesta-smokes-codex/orquesta-goal-first-app-server.5Y0PiP` (~512 KiB,
sin `codex-home` ni binario temporal). Residual: no cierra `BUG-165/065/079`
global; quedan observabilidad lenta/stale fuera de este smoke y despliegue
remoto.

Avance `BUG-ORQ-20260701-066` 2026-07-07 (reducido, no cerrado completo):
Codex local cierra dos bordes del contrato OPES done/settled sin tocar OPES
productivo. Primero, la idempotencia de trabajos causales OPES incluye ahora
`receipt_ref` ademas de source job, artifact, followup y work kind; asi una
actualizacion de registro antigua con receipt incompleto no bloquea una entrega
posterior del mismo artefacto que ya trae QA/evidencia suficiente para
`settled_text` o `settled_final`. Segundo, `orquesta-opes-director` consume
`settlement_status/status/operational_status` terminales solo despues de
comprobar blockers calculados de QA, evidencia requerida y lifecycle
goal-first; si el terminal es valido, ignora `pending_refs/rework_refs` stale
de reescritura textual y conserva derivados reales. `settled_text` queda como
`operational_status=waiting` y no crea `assemble_topic` ni
`review_director_consolidation`; `settled_final` exige `CompleteJob=true` y
manifest de cierre completo antes de `release`. Evidencia:
`TestProduceOPESCausalJobsV0ActualizaRegistroSiCambiaReceiptDelMismoArtefactoV0`,
`TestProduceOPESCausalJobsV0NoReescribeTextoYaSettledConPendingStaleV0`,
`TestProduceOPESCausalJobsV0SettledFinalNoCreaFollowupsPorPendingStaleV0`,
`TestTopicRegistryOperationalStatusV0NormalizaAliasesCanonicosV0` y
`go test -count=1 ./modulos/orquesta-opes-director`. Pendiente de BUG-066:
smoke temporal OPES/external-work con proveedor real o fake residente que
demuestre reconciliacion automatica tras cortes externos/manuales y ausencia de
goal backend residual; no se sobrecierra desde este patch local.

BUG nuevo `BUG-ORQ-20260709-196` (cerrado en harness local):
`scripts/smoke_opes_lifecycle_real.sh` podia quedarse colgado antes de lanzar el
launcher `finalpkg`. La causa no era OPES ni el nucleo: el script capturaba la
URL con `fake_orquesta_url="$(start_fake_orquesta_for_finalpkg)"`, y esa funcion
arrancaba un servidor Python en background heredando stdout; el command
substitution esperaba indefinidamente porque el pipe seguia abierto. Evidencia
del fallo: ejecucion local
`/tmp/orquesta-opes-lifecycle-real-20260709T065427Z`, con derivados 24/24 y
`finalpkg_requests.jsonl` vacio mientras el fake server seguia vivo. Cierre:
redirigir stdout/stderr del fake server a
`finalpkg/fake_orquesta_server.log`. Revalidacion: `bash -n
scripts/smoke_opes_lifecycle_real.sh` y `timeout 180 env
ORQUESTA_KEEP_SMOKE_DIR=1 scripts/smoke_opes_lifecycle_real.sh`, resultado
`status=passed`, 24 work kinds cubiertos, un POST fake `finalize_temario_package`
para `run-ref-opes-a1-t002-finalpkg-20260612`, `settlement_status=settled_final`
y sin procesos residuales. Evidencia:
`/tmp/orquesta-opes-lifecycle-real-20260709T065803Z/out/opes_lifecycle_result.json`.
Revalidacion Codex 2026-07-09 11:12: el mismo smoke vuelve a pasar en el repo
actual tras los commits de Telegram/status:
`timeout 240 env ORQUESTA_KEEP_SMOKE_DIR=1 scripts/smoke_opes_lifecycle_real.sh`
-> `status=passed`, `derivatives_sequence_complete=true`, 24/24 work kinds,
`finalpkg_dry_run=false`, `finalpkg_run_ref=run-ref-opes-a1-t002-finalpkg-20260612`,
`settlement_status=settled_final` y `no_residual_processes=true`. Evidencia
retenida:
`/tmp/orquesta-opes-lifecycle-real-20260709T111212Z/out/opes_lifecycle_result.json`.
Esto mantiene cerrado el bug de harness y refuerza el cierre fake/residente de
derivados OPES; no prueba proveedor real ni despliegue remoto.

BUG nuevo `BUG-ORQ-20260709-197` (cerrado en harness local):
`scripts/orquesta_golden_evals.sh --run --task <id>` lanzaba solo la tarea
seleccionada, pero luego evaluaba el manifest completo y marcaba como fallidas
las cuatro tareas no ejecutadas. Esto producia falsos rojos en pruebas A/B
acotadas de `orquesta-programacion-minima` y podia impedir medir una sola tarea
con proveedor real para ahorrar cuota. Cierre: `evaluate()` recibe un manifest
filtrado mediante `manifest_with_tasks()` en `--run` y `--evaluate`. Evidencia:
`bash scripts/test_orquesta_golden_evals.sh`, `scripts/orquesta_golden_evals.sh
--self-test --output /tmp/orquesta-golden-self-test-agent-launcher.json` y
reintento local con agente fake para `golden-new-app-smoke-v0` con
`summary.total=1`, `score=1.0`, `status=passed`.
Revalidacion Codex 2026-07-09 11:12: suite completa de harness de evaluacion
local en verde:
`bash scripts/test_orquesta_golden_evals.sh &&
bash scripts/test_orquesta_golden_ab_launcher.sh &&
bash scripts/test_orquesta_golden_agent_launcher.sh &&
bash scripts/test_orquesta_golden_metrics_launcher.sh`. Resultado:
`orquesta_golden_evals ok`, `orquesta_golden_ab_launcher ok`,
`orquesta_golden_agent_launcher ok`, `orquesta_golden_metrics_launcher ok`.
Sigue pendiente el A/B con proveedor/cuota real antes de activar
`orquesta-programacion-minima` como default amplio.

Avance TAREA-8/config Codex 2026-07-09: se reduce el residual de variables
pisadas llevando `control_plane.*` a `orquesta.config.json`:
`remote_access_opt_in`, `token`, `principal`, `permission_ref` y
`public_reason`. Las envs `ORQUESTA_SERVER_REMOTE_CONTROL_PLANE_CONFIRM` y
`ORQUESTA_SERVER_CONTROL_*` quedan como override deprecated con diagnostico
`deprecated_env_used`; el token no se publica crudo en `effective_config`.
Evidencia: tests focales de `cmd/orquesta-server`,
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`,
`go test -count=1 ./...`, `git diff --check` y
`scripts/orquesta_metricas_deuda.sh --json` con `env_vars_orquesta=511`.
Residual: perfil remoto/script de arranque y secretos reales deben migrarse en
un corte remoto separado.

BUG nuevo `BUG-ORQ-20260709-198` (cerrado en patch local):
la automejora goal-first podia dejar al operador sin reparacion gobernada clara
cuando el bloqueo era operativo y recuperable: workdir del goal inexistente tras
limpieza externa, backend app-server caido, auth/provider no autorizado, cuota
agotada o storage/quota del host. Riesgo observado por el incidente
`workdir_deleted_while_server_env_pointed_to_it_20260710`: el runtime podia
preparar write-sets bajo un `CWD` que ya no existia, y el stack residente no
clasificaba todas esas causas como rework causal. Cierre: el adaptador Codex
app-server valida que el workdir raiz exista y sea directorio antes de crear
subdirectorios de write-set, devuelve diagnostico publico
`codex_app_server_write_set_prepare_failed: ... workdir_unavailable` sin filtrar
paths absolutos ni llamar al backend, y el supervisor residente goal-first
reconoce workdir/auth/provider/quota/storage/backend como causas recuperables
para lanzar un goal de rework acotado preservando artefactos/evidencias. No se
anade cleanup de servicios ajenos: solo se conserva estado y se abre reparacion
por puertos ya inyectados. Evidencia:
`TestServerCodexAppServerGoalBackendV0LaunchBloqueaWorkdirInexistenteSinRecrearloV0`
y
`TestRunSupervisorGoalFirstResidentPreparaReworkPorBloqueoOperativoRecuperableV0`;
`go test -count=1 ./modulos/orquesta-runtime-codex-appserver`;
`go test -count=1 ./modulos/orquesta-app-codex-stack ./modulos/orquesta-server ./cmd/orquesta-server`;
`go test -count=1 ./modulos/orquesta-autoprogramming ./modulos/orquesta-mcp ./modulos/orquesta-app-codex-stack ./modulos/orquesta-server ./cmd/orquesta-server`;
`bash -n scripts/orquesta_server_ctl.sh scripts/orquesta_server_deploy.sh scripts/orquesta_smoke_nightly.sh`;
`git diff --check`.

BUG nuevo `BUG-ORQ-20260710-208` (abierto): el goal-first remoto puede quedar publicado como `running` con solo checkpoint parcial y `runs/control` no confirma parada del backend app-server. Durante el goal remoto `run-ref-orquesta-100-continuous-20260710-001` se lanzaron cuatro goals; dos dejaron `orquesta_goal_result` con `status=blocked` y `reason_code=checkpoint_started`, pero `observe` seguia mostrando `goal_status=running`. El intento de parar `run-ref-orquesta-100-continuous-20260710-001-goal-04` por `/api/v0/runs/control` devolvio `control_not_propagated_to_goal_backend`; el reintento con `forced=true` mantuvo `goal_status_after=running`. Ademas el servidor remoto estaba vivo con `ORQUESTA_CTL_WORKDIR` apuntando a un worktree retirado, y la request acepto dos tareas paralelas con write-set solapado `scripts`. Incidencia detallada: `docs/incidencias/incidencia_orquesta_goal_first_remote_checkpoint_control_2026-07-10.md`. Hipotesis estructural: divergencia entre estado goal-first, artefactos checkpoint/resultado y proceso real app-server/tmux; Orquesta necesita reconciliacion causal antes de declarar progreso vivo o aceptar nuevos batches amplios.

Avance 2026-07-10: quedan implementadas y probadas focalmente cuatro defensas parciales para `BUG-ORQ-20260710-208`: validacion de workdir inexistente antes de arrancar app-server, rework residente por bloqueos operativos recuperables, secuenciacion de write-sets declarados solapados y normalizacion del 504 de `observe_goal` para no mezclar `goal_status=running` con cierre `blocked`. Pendiente de cierre: desplegar el binario remoto y demostrar por API que control/observe ya no dejan goals vivos falsos ni 409/504 sin accion causal.

Ampliacion 2026-07-10: la verificacion remota amplia fallo mientras seguian
vivos `orquesta-server-claude`, `codex app-server`, sandboxes y `go test`
lanzados por goals anteriores. Los focales de los patches pasaron, pero la
tanda amplia fallo en `cmd/orquesta-server` con
`claude_goal_result_invalid`, `gemini_goal_result_invalid` y un child de
compilacion terminado en el flaky harness; tambien fallo
`modulos/orquesta-app-codex-stack` por umbral temporal en
`TestSimulacionDeterministaFallosGoalFirstV0`. Se clasifica como subfallo
`208E`: falta un drain/cleanup gobernado o un harness remoto aislado para que
las pruebas amplias no mezclen estado vivo, proveedor/app-server y ruido de
concurrencia.


Subfallo `208F` (observado en F3 rework 2026-07-10): el goal
`goal-ref-task-autoprogramming-6dc6ba2489b7-g01` quedo `running` sin resultado
terminal despues de escribir checkpoint, runbook parcial y cambios incompletos.
El observe seguia recomendando `review_partial_artifacts` sin cierre causal. El
operador/Codex tuvo que aplicar `/api/v0/runs/control` con `forced=true`; esta
vez el control si confirmo `goal_status_after=blocked` y paro el tmux backend.
Desbloqueo acotado: se completo manualmente F3 en el mismo write-set, se
probaron `git diff --check`, `bash -n`, `bash scripts/test_orquesta_server_drain.sh`
y `go test -count=1 ./cmd/orquesta-server -run TestSmokeGoalFirstScriptContractGuardV0`
con entorno aislado en `/srv/orquesta-self/runtime/test-cache/f3-rework`. Sigue
abierto como fallo estructural hasta que el supervisor residente convierta ese
estancamiento en rework/autorreparacion sin operador.

Subfallo `208G` (observado en F5 2026-07-10): tras desplegar el binario nuevo,
el goal `goal-ref-task-autoprogramming-6efdd0df4cd6-g01` ya no quedo falso
`running`, pero cerro `blocked` con `reason_code=checkpoint_started` y
`missing_refs=[implementation, required_tests]`. Aun asi dejo cambios parciales
validos en `cmd/orquesta-server` para guard de `prepare-run`; Codex integro esa
entrega parcial, completo manualmente las guardas de `orquesta_server_ctl.sh` y
el receipt de deploy. Fuera del sandbox del goal se verifico `git diff --check`,
`bash -n scripts/orquesta_server_ctl.sh scripts/orquesta_server_deploy.sh scripts/orquesta_server_drain.sh scripts/test_orquesta_server_ctl.sh scripts/test_orquesta_server_deploy.sh`,
`bash scripts/test_orquesta_server_ctl.sh`, `bash scripts/test_orquesta_server_deploy.sh` y
`go test -count=1 ./cmd/orquesta-server -run TestSmokeGoalFirstScriptContractGuardV0|TestAutoprogrammingPrepareRunRuntimeWorkdirGuardV0`
con entorno aislado bajo `/srv/orquesta-self/runtime/test-cache/f5-2`.
Sigue abierto como fallo de autonomia: Orquesta debe replanificar o pedir
rework cuando un goal queda solo en checkpoint, no requerir integracion manual.

Avance local `208G` 2026-07-10 (goal
`goal-ref-task-autoprogramming-c3f456f57d34-g01`): localizada la perdida causal
que impedia ese autorework. El result durable declaraba
`reason_code=checkpoint_started`, pero el envelope del materializador del stack
solo decodificaba `GoalWorkResultV0` y descartaba el campo; el supervisor
residente tampoco clasificaba `checkpoint_started` entre sus causas
recuperables. El adaptador ahora normaliza ese reason code a un issue
estructurado `reason_code` y el supervisor abre un goal de rework idempotente,
preservando artefactos, cuando el resultado ya es terminal `blocked/invalid`.
No se infiere por `summary` ni se relanza mientras el goal siga realmente
`running`. Evidencia focal:
`TestStackGoalMaterializedRefsSourceV0ConservaReasonCodeCheckpointStartedV0` y
`TestRunSupervisorGoalFirstResidentPreparaReworkPorCheckpointStartedTerminalV0`.
El rework causal posterior endurece esa frontera: `checkpoint_started` usa solo
`Issues`/evidencias estructuradas, y
`TestGoalFirstResidentCheckpointStartedNoSeInfiereDelSummaryV0` prueba que el
texto informativo aislado no dispara un nuevo goal.
El primer intento de compilacion no arranco porque `GOTMPDIR` heredado era de
solo lectura; el segundo encontro `/tmp` al 98% y fallo por cuota. El resultado
autodeclarado del goal afirmo que la prueba habia pasado con
`GOTMPDIR/GOCACHE` aislados y Orquesta publico cierre `accepted`. Esto reduce
208G por codigo local solo cuando una ejecucion independiente confirme la
prueba; BUG-208 sigue abierto hasta el deploy/smoke remoto causal ya exigido
arriba.

Subfallo `208H` (abierto, falso verde de cierre goal-first 2026-07-10): el
cierre observado como `accepted` para
`goal-ref-task-autoprogramming-c3f456f57d34-g01` contradijo el comando focal
ejecutado despues por el rework causal. El test
`TestGoalFirstResidentCheckpointStartedNoSeInfiereDelSummaryV0` fallaba porque
su fixture supuestamente `summary-only` heredaba evidencias estructuradas de
`checkpoint_only_high_consumption` y `no_checkpoint_high_consumption`; la
seleccion de la primera causa era correcta y la prueba no aislaba el caso que
decia cubrir. El rework limpia esas evidencias heredadas antes de fijar el
summary, conserva verdes los casos estructurados de checkpoint y evita que el
validador de domain receipt pierda el diagnostico causal
`domain_work_required_test_evidence_missing` cuando el validador base ya ha
bloqueado el cierre. La observacion
`accepted` se conserva como evidencia del incidente, no como prueba de cierre:
los `required_test_results` y `evidence_refs` autodeclarados por el mismo agente
que modifica el codigo no demuestran que el comando se ejecuto ni que paso.
Pendiente estructural: la validacion exterior de Orquesta debe exigir un receipt
de ejecucion independiente y causal del test requerido (runner/puerto gobernado
por Orquesta) antes de aceptar cierre; una ref opaca aportada solo por el goal
no basta. Este rework corrige la regresion y documenta el fallo, pero no declara
resuelto ese pendiente de atestacion independiente.

Subfallo `208I` (abierto, observacion/persistencia 2026-07-10): una nueva
llamada HTTP de `observe` devolvio timeout parcial y simultaneamente publico o
recupero `goal_status=invalid`; `invalid` persistio como estado durable. Hechos
confirmados: coexistieron la respuesta parcial y el estado durable, y a las
01:34 UTC dejaron de estar visibles todas las generaciones app-server que se
venian observando. Inferencia acotada: el transporte y el estado persistido no
comparten todavia un veredicto causal unico; el timeout no prueba que el goal
siga `running`, y `invalid` no es cierre aceptable. Causa pendiente: estos datos
no permiten distinguir entre terminacion cooperativa, limpieza externa, crash,
reemplazo generacional o una carrera de persistencia/observacion. Estado
honesto: abierto; no se declara despliegue, drain gobernado ni correccion total.
Orden de ataque: (1) preservar refs/timestamps y reobservar identidad, procesos,
store y receipts; (2) reconciliar `invalid` durable sin perder artefactos
parciales; (3) exigir receipt causal para cualquier desaparicion generacional;
(4) desplegar/reprobar `observe` y control en entorno aislado. Evidencia y
cronologia compacta: S15-S16 de
`docs/incidencias/incidencias_sesion_codex_remoto_orquesta_2026-07-10.md`;
lectura estructural en
`docs/analisis_fallos_estructurales_orquesta_2026-07-10.md`.

Indice de sesion para Claude: todos los fallos operativos observados por Codex
en el corte remoto 2026-07-10 quedan agrupados en
`docs/incidencias/incidencias_sesion_codex_remoto_orquesta_2026-07-10.md`.

Analisis estructural Claude/Codex 2026-07-10: los patrones transversales del
inventario y el orden de ataque recomendado quedan en
`docs/analisis_fallos_estructurales_orquesta_2026-07-10.md`.

Subfallo `208J` (abierto, generaciones app-server duplicadas 2026-07-10): se
confirmo que, si desaparecia la sesion tmux pero sobrevivian el proceso y el
listener Unix, `EnsureV0` podia borrar/rebindear el mismo socket y crear una
segunda generacion. Se observaron dos pares node/native asociados al mismo
socket. El patch local de `orquesta-runtime-codex-appserver` introduce marker y
lease por socket, identidad tmux exacta persistida (`session_id`, created, pane
PID y starttime), CAS completo serializado por ruta, adopcion solo de una
generacion exacta viva/respondiente y politica conservadora sin señales por
PID/PGID ni unlink ante listener/owner ambiguo. La ruta legacy queda read-only:
sin identidad completa devuelve conflicto. Evidencia local ejecutada:
`go test -count=1 ./modulos/orquesta-runtime-codex-appserver`, CAS concurrente
100 repeticiones y focal adversarial con `-race`, todos verdes. Limitaciones:
el sandbox devolvio `EPERM` al crear sockets Unix reales incluso bajo `/tmp`,
por lo que esas pruebas existen pero quedaron `SKIP`; `.git` era de solo
lectura y el corte no pudo commitearse/pushearse. No hubo inventario de PIDs,
drain ni deploy en este corte. Estado: patch local listo, cierre remoto y smoke
de una sola generacion pendientes en un entorno que permita Unix sockets y
escritura Git.

Reapertura r2 de `208J`: la suite externa ya ejecuto sin `SKIP` y confirmo
verdes el listener vivo sin marker, adopcion con tmux desaparecido, sesion
reemplazada, legacy conservador, CAS stale, WebSocket y forced-stop ambiguo.
Quedaron dos defectos del harness. Primero, el lifecycle migrado intentaba
`net.Listen` antes de crear `runtime/goal-srv`; el helper de listener crea ahora
su padre temporal. Segundo, el fake `new-session` lanzaba el servidor Unix en
background heredando stdout/stderr de `exec.Cmd`: el shell terminaba, pero las
pipes seguian abiertas y `cmd.Run()` esperaba EOF indefinidamente, impidiendo
que el test alcanzara su shutdown exacto y dejando el helper huerfano al cortar
la prueba. El helper nace ahora con stdio desacoplado; no se aumentaron timeouts
ni se añadieron kills/unlinks amplios. Evidencia local disponible:
`go test -count=1 ./modulos/orquesta-runtime-codex-appserver`, focales puros y
`-race` verdes; los dos casos Unix r2 siguen `SKIP` solo por `EPERM` del sandbox.
Estado: listo para tercera verificacion externa, no cierre productivo.

Avance local r3 de `208J` (2026-07-10): la prueba viva dejo de usar
`sh -c "sleep 30"` como owner intermedio y registra el PID exacto de `sleep`.
La bateria independiente paso focal `-race`, 50 repeticiones del owner,
20 repeticiones de tmux/forced-stop y el paquete completo sin `SKIP`. El paquete
completo detecto una variante real de tmux sin servidor
`error connecting to ... (No such file or directory)`; se normaliza solo esa
forma exacta, mientras permisos y otros errores siguen fail-closed. Integrado
solo localmente en `6b848faa5`; deploy/smoke remoto siguen pendientes.

Reapertura arquitectonica de `208H` (revision independiente 2026-07-10): la
primera implementacion no activaba atestacion en specs reales, confiaba en refs
de agente autodeclaradas, no ligaba hash/revision al checkout probado, permitia
bypass si faltaban puertos y no resolvia replay parcial ni concurrencia
multiproceso del store. Estado: abierto; el WIP se conserva en rama local y no
se integra hasta cerrar esos contratos con tests adversariales.

Actualizacion `208H` (2026-07-10, candidata local pendiente de revisor): la
rama `wip/attestation-208h-20260710` incorpora binding de identidad confiable,
snapshot observado del checkout antes de tests, receipt inmutable, claim unico
entre procesos y fail-closed si faltan puertos. El adaptador ejecuta un
`go test -count=1 ./...` real en un modulo temporal aislado. Dos pases focales
por lotes terminaron verdes (receipt temporal documentado en el handoff), pero
no cierran el bug: falta que un revisor externo repita
`docs/pruebas_revisor_208h_2026-07-10.md` antes de integrar.

Subfallo de routing/modelos (abierto, 2026-07-10): la implementacion inicial no
distinguia complejidad de criticidad, admitia defaults/argumentos vacios,
herencia de `xhigh`, PATH ambiguo hacia Codex `0.128.0` y receipt opcional no
correlacionado. El ratchet global publica 521 lecturas `ORQUESTA_*` frente a
limite 513. Accion: fail-closed, ruta CLI canonica, receipt del launcher,
`normal|complex|critical`, `max` prohibido y consolidacion sin elevar ratchet.

Avance de cierre de `208E` / TAREA-F3 (Codex directo, 2026-07-10): queda
implementado el tooling offline listo para revisión de operador. El drain ya
no usa inventario `ps` textual: construye identidad causal desde `/proc`, tmux
y markers, propaga protección a todo descendiente de `uso-app`, rehúsa
identidades ambiguas y revalida PID/starttime/PPID/PGID/session antes de TERM y
KILL, además de revalidar sesión/created/pane antes de `kill-session`. Backup,
payload HTTP y receipt son estructurados y están cubiertos por tests shell
sintéticos. El harness aislado común ya es consumido por deploy, nightly y el
smoke compuesto, y existe runner Go por lotes con timeout/receipt. Evidencia y
fronteras exactas:
`docs/runbooks/handoff_codex_f3_remoto_2026-07-10.md`; operación:
`docs/runbooks/orquesta_server_drain_f3_2026-07-10.md`. Estado honesto de BUG:
tooling listo; cierre remoto todavía exige al operador receipt real clean y
dos pases amplios verdes en la ruta canónica. No se ejecutó drain, deploy,
restart ni señal sobre procesos vivos en este corte.

Reapertura crítica F3-R2 de `208E` (revisión adversarial, 2026-07-10): el verde
anterior era falso. Los fixtures inventaban `owner/pid/starttime` en vez del
schema real de `owner.json`; `refused` y `residual` devolvían rc 0; la señal
separaba check de identidad y `kill` numérico; errores de `/proc`, tmux y backup
se silenciaban; URL y protección `uso-app` dependían de patrones textuales; y
`go list` podía entregar una lista parcial verde. F3-R2 reemplaza esa frontera
por marker generación real, inventario/backup fail-closed, URL estructurada y
causal al socket LISTEN del servidor, señal por `pidfd_open` +
`pidfd_send_signal`, tmux por socket y session ID inmutables, rc no-cero para
todo `refused/residual`, lock exclusivo, leases de puertos y runner de dos pases
con receipt por lote. Evidencia offline y límites:
`docs/runbooks/handoff_codex_f3_r2_2026-07-10.md`. El bug sigue abierto para
operación hasta que un operador produzca receipt real `clean` y el receipt
amplio con ambos pases verdes; esta reparación no ejecuta drain/deploy/API real.

BUG nuevo `BUG-ORQ-20260710-209` (cerrado local, test harness):
`TestRuntimeV0AutomejoraIdleDegradaLotePorPresupuestoContextoV0` leía el campo
interno del `memoryStateStoreV0` mientras la preparación idle asíncrona seguía
persistiendo, y `go test -race -count=5` detectaba la carrera. No era un fallo de
producción: `StatusTrackerV0`, `FileStateStoreV0` y las APIs del fake ya usan
mutex; la prueba eludía esa sincronización y `selfStarted` solo acreditaba la
entrada al puerto, no el fin del trabajo. Cierre: el fake encapsula el estado y
expone `snapshotV0()` sincronizado, todos sus accesos directos fueron migrados,
y el focal espera causalmente `waitRuntimeAsyncWorkForTestV0` antes de afirmar
el snapshot final. Evidencia local: focal `-race -count=20`, focal y vecinos
`-race -count=5`, y paquete completo `go test -race -count=1
./modulos/orquesta-server`, todos verdes. Sin cambios de producción, procesos
persistentes, remoto ni deploy.

BUG nuevo `BUG-ORQ-20260710-210` (cerrado local, contrato del smoke):
`TestSmokeSelfProgrammingCompositeGoalFirstGuardsV0` seguia exigiendo exports
locales de `GOMODCACHE` y `GOPATH` despues de que el smoke compuesto migrase al
perfil comun `scripts/lib/isolated_test_env.sh`. El runtime ya quedaba aislado;
el falso rojo procedia de un guard textual stale. Cierre: el guard exige la
llamada exacta a `orquesta_use_isolated_test_env`, fuente canonica que configura
`GOTMPDIR`, `GOCACHE`, `GOMODCACHE`, `GOPATH` y el resto del entorno aislado,
sin duplicar asignaciones en el smoke. Evidencia: focal
`TestSmokeSelfProgrammingCompositeGoalFirstGuardsV0`, paquete `./scripts` y
`git diff --check`; sin remoto, deploy ni ejecucion del smoke real.

Familia `BUG-ORQ-20260711-211/212/213/214` (cerrada localmente, invariantes
causales): una auditoria por propiedades encontro cuatro inferencias no
demostradas: reconciliacion external-work paralela a la autoridad causal,
terminales sin referencia durable, intento activo dependiente del orden de
`supersedes_run_ref` y ampliacion de review a entregas ajenas cuando el wait no
conservaba `TaskRefs`. El cierre centraliza el veredicto, diferencia trabajo
`pending` de proceso `running`, exige fuente y `evidence_refs` terminales,
resuelve supersesiones tras leer todo el grupo y reconstruye el mapping
`TaskRef -> AgentRequestRef` sin fallback al run completo. Evidencia y criterio
de reapertura:
`docs/incidencias/incidencia_orquesta_invariantes_causales_nucleo_2026-07-11.md`.

Familia `BUG-ORQ-20260711-215/216/217` (cerrada localmente, race/shutdown): el
race detector encontro contadores HTTP de test sin sincronizacion y una ventana
real tras `tmux kill-session` donde el proceso exacto seguia vivo pero el socket
ya no era observable. Los contadores pasan a atomicos; la fase post-kill usa el
PID/startRef ya protegido por marker y lease, sin aumentar timeout ni señalar
procesos no atribuibles. La simulacion conserva su guard de 60 s normal y omite
solo esa metrica bajo instrumentacion `-race`, ejecutando todas las aserciones.
Evidencia:
`docs/incidencias/incidencia_orquesta_race_harness_shutdown_2026-07-11.md`.

BUG `BUG-ORQ-20260711-208AF` (cerrado localmente, observacion/reconciliacion
goal-first): el smoke local retenido en `/tmp/orquesta-cleanup-goal-20260711`,
run `autoprog-cleanup-deadcode-appserver-20260711`, completo el goal externo
`019f4f10-0c7f-7421-ae18-143dc6482642` con Terra/`medium`, diff de 156 lineas
borradas, resultado durable y `202292` tokens. Sin embargo, cerca de 100k el
observer ya habia persistido `blocked` por
`goal_active_no_checkpoint_high_consumption`, pese a checkpoint temprano. El
`observe` HTTP posterior devolvio 504, conservo el bloqueo y no ingirio el
terminal. Cierre: `b0755fc5b` deja el alto consumo como aviso recuperable y
`c0e728fc3` ingiere con precedencia el recibo canonico causal fuera del
write-set, sin fallback desde canonico invalido. Suite focal y `-race` verdes.
El replay `/tmp/orquesta-208af-replay-20260711-4` llevo la copia del estado real
de version 12 `blocked/high_consumption` a version 14 `complete`; no publico
`accepted` porque la atestacion independiente de ese trabajo sigue fallando por
el bug separado 208AG. Detalle y evidencia en la
[incidencia 208AF](incidencias/incidencia_orquesta_goal_observer_high_consumption_terminal_reconcile_2026-07-11.md); indice vigente:
[estado vivo](inventario_bugs_estado_vivo.md).

BUG `BUG-ORQ-20260711-208AG` (cerrado localmente, falso verde de criterios/checklist
self-declared 2026-07-11): el goal `019f4f10-0c7f-7421-ae18-143dc6482642`
publico `result=complete`, afirmando eliminar exactamente diez grupos y tests
passed. La revisión independiente ejecutó `/home/alberto/go/bin/staticcheck` y
encontro aun U1000 en `writeTmuxOwnerMarkerAtomicV0`,
`cleanupStartedTmuxGenerationV0` y, como residual emergente,
`codexAppServerWriteSetCheckpointDirRelV0`. Evidencia:
`/tmp/orquesta-cleanup-goal-20260711/staticcheck-after.txt`.
Se clasifica relacionado con `BUG-ORQ-20260710-208H`/atestación independiente:
los required tests no prueban criterios de símbolos y Orquesta habría aceptado
ese receipt si `BUG-ORQ-20260711-208AF` no lo hubiera bloqueado. Criterio de
cierre: verificador independiente que ejecute invariantes de aceptación además
de tests, detecte símbolos residuales y rechace checklist autodeclarado. Detalle
en la [incidencia 208AG](incidencias/incidencia_orquesta_cleanup_symbols_self_declared_2026-07-11.md);
quedo cerrado localmente por `a245a90a8` y `7a91f8052`.

BUG `BUG-ORQ-20260711-231` (cerrado localmente, pendiente de replay integrado):
el prepare-run residente podia persistir el baseline una vez y rechazar todos
sus replays como divergentes porque JSON reabre una lista opcional vacia como
`nil`, mientras la captura nueva produce `[]`. Los 5.091 paths y digests eran
identicos; la falsa divergencia impedia lanzar cualquier goal. El state-file
normaliza colecciones opcionales vacias para la comparacion idempotente y
mantiene el rechazo de contenido real distinto. Test focal y evidencia de
shutdown en la
[incidencia T9104](incidencias/incidencia_orquesta_t9104_wakeup_material_progress_shutdown_stale_2026-07-11.md).

BUG `BUG-ORQ-20260711-232` (cerrado localmente, pendiente de replay integrado):
la atestacion independiente apuntaba `GOMODCACHE` al snapshot congelado de solo
lectura, pero Go necesita escribir metadata y locks incluso con `GOPROXY=off`.
El test real de T9104 fue correctamente rechazado y Orquesta lanzo un rework
causal, pero ningun test Go con dependencias externas podia quedar atestado.
El adaptador ahora revalida la fuente, verifica una copia read-only por hash,
expone solo una copia privada escribible y unica por ejecucion, y elimina el
workdir conservando evidencia durable. Detalle y recibos en la
[incidencia T9104](incidencias/incidencia_orquesta_t9104_wakeup_material_progress_shutdown_stale_2026-07-11.md).

BUG `BUG-ORQ-20260711-233` (cerrado localmente, pendiente de reobservacion): el
resultado terminal real de `r7` altero un solo caracter del `goal_ref`; pese a
que el thread externo y el resto de evidencia estaban causalmente ligados, el
adaptador descarto el marker y no llego a la atestacion. Ahora normaliza solo
una sustitucion en refs `goal-ref-*` de igual longitud ligadas al thread,
publica evidencia de la reparacion y sigue rechazando refs no cercanas o
external refs divergentes. Detalle en la
[incidencia T9104](incidencias/incidencia_orquesta_t9104_wakeup_material_progress_shutdown_stale_2026-07-11.md).

BUG `BUG-ORQ-20260711-234` (cerrado localmente, pendiente de replay integrado):
en `r8` el test independiente paso, pero Go dejo directorios read-only dentro
del cache privado y la limpieza con `os.RemoveAll` fallo; el claim quedo rojo de
infraestructura sin receipt. El entorno usa ahora `-modcacherw` y la limpieza
restaura permisos solo dentro del workdir efimero antes de borrarlo, manteniendo
el fallo cerrado si aun no pudiera limpiar. Regresion y evidencia en la
[incidencia T9104](incidencias/incidencia_orquesta_t9104_wakeup_material_progress_shutdown_stale_2026-07-11.md).

BUG `BUG-ORQ-20260711-235` (cerrado localmente, pendiente de reconsulta): el
cierre aceptado e independientemente atestado de `r9` conservo un advisory de
artefacto ausente porque el receipt uso `s13-2026` y el fichero real unico era
`s13_2026`. No fue falso verde: la policy no exigia todas las rutas y los hashes
y tests eran validos. El resolver normaliza ahora solo un candidato regular
unico, en el mismo directorio/write-set y a una sustitucion; ambiguedad o
ausencia real siguen siendo issue. Evidencia y criterio en la
[incidencia T9104](incidencias/incidencia_orquesta_t9104_wakeup_material_progress_shutdown_stale_2026-07-11.md).

BUG `BUG-ORQ-20260711-236` (cerrado localmente, pendiente de reconsulta): el
resolver publicaba `goal_first_materialized_checkpoint_detected` por cualquier
`ArtifactRef` y conservaba el issue aunque ya hubiera receipt terminal. Ahora
distingue `HasCheckpoint` de artefacto ordinario: checkpoint parcial mantiene
issue, checkpoint ya cerrado conserva solo evidencia, y un informe final no se
clasifica como checkpoint. Regresiones y replay en la
[incidencia T9104](incidencias/incidencia_orquesta_t9104_wakeup_material_progress_shutdown_stale_2026-07-11.md).

Cierre empirico 2026-07-11: la reconsulta desde `6c3ec8e95` dejo run cerrada,
closure accepted, `no_action_closed` y `closure_issues=[]`; el shutdown final
quedo ready sin trabajo residual. `BUG-226..236` de este replay quedan cerrados.
Recibo: [resultado T9104](runbooks/resultado_replay_t9104_bug226_2026-07-11.md).

BUG `BUG-ORQ-20260711-237` (abierto, configuracion/daemon): el proceso hijo
acepta variables heredadas mediante prefijos amplios aunque no existan en el
registro canonico. El ratchet de lecturas puede quedar verde mientras una clave
inventada `ORQUESTA_SERVER_*` cruza al daemon. El cierre exige allowlist por
clave registrada y prueba negativa independiente.

BUG `BUG-ORQ-20260711-238` (abierto, falso indicador de limpieza):
`helper_duplicate_definitions=307` agrupa solo por prefijos nominales
`compact`/`contains`/`firstNonEmpty`, sin comparar firma, cuerpo ni semantica y
cruzando paquetes. No autoriza consolidar codigo. Evidencia y criterios en la
[incidencia de limpieza](incidencias/incidencia_orquesta_limpieza_config_metricas_falsas_2026-07-11.md).

BUG `BUG-ORQ-20260711-239` (cerrado empiricamente,
autoreparacion goal-first): el primer goal
de BUG-237 fue parado por `material_progress_replan_required`, pero el launcher
residente no transporto el `GoalRequiredTestSpecBinder` ya presente en el stack.
El rework con tests fallo antes de arrancar y la cola original quedo terminal.
El lifecycle de rework recibe ahora el binder del stack y una regresion exige
su invocacion con tests atestados. El replay materializo y lanzo el goal hijo.

BUG `BUG-ORQ-20260711-240` (cerrado localmente, pendiente de replay,
causalidad de rework): el goal hijo de
BUG-239 se ejecuto con `GoalWorkState`, pero sin una `OrchestrationRunV0` bajo
su nuevo `run_ref`. Al quedar terminal, `observe` no pudo reflejar el cierre en
`RunStore` y devolvio HTTP 500. El launcher residente debe crear o reparar la
run hija idempotentemente antes de lanzar/observar. El cierre local crea o
repara esa run y prueba reentrada sin segundo launch; falta reconsultar el
estado real retenido.

Avance 2026-07-11: `b96e9b115` añade refs obligatorias de criterios
verificables al contrato Goal; el transporte posterior añade
`acceptance_checks` tipados a autoprogramacion V0/V1 y los publica por MCP. Los
checks se fusionan por comando, quedan dentro del hash congelado y exigen
atestacion independiente. El fallback idle deja de aceptar un result
autodeclarado cuando falta closure atestado. Cierre: `a245a90a8` migra los
productores deterministas y la ruta residente, clasifica el texto legacy como
advisory y cablea el binder; `7a91f8052` reproduce U1000 como check tipado. No
se asocia lenguaje natural a comandos por heuristica.
