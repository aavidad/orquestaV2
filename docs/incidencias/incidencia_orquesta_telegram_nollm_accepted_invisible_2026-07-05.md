# Incidencia: Telegram no-LLM accepted invisible y runtime no desplegado - 2026-07-05

## Resumen

Orquesta acepta el rework `request-ref-remoto-telegram-nollm-runtime-20260705-001`, pero la proyeccion publica de autoprogramacion no lo muestra despues de la aceptacion y no se materializa resultado durable visible. Esto impide cerrar el control movil no-LLM como operativo.

## Evidencia

- `POST /api/v0/autoprogramming/prepare-run` devolvio `accepted=true` para `request-ref-remoto-telegram-nollm-runtime-20260705-001`.
- La respuesta no contiene `goal`; contiene `workflow_task_refs=[task-autoprogramming-c20a585ff5c3-g01]`, `wait_agent_refs=[agent-ref-task-autoprogramming-c20a585ff5c3-g01]` y un bloque `continue` legacy.
- `POST /api/v0/autoprogramming/status` no devuelve entradas para ese `run_ref` tras la aceptacion.
- Existe ruta runtime: `/srv/orquesta-self/claude-director-20260705/runtime/request-ref-remoto-telegram-nollm-runtime-20260705-001`.
- La cola publica muestra 5 runs `ready` y diagnostico `run_stats_required_for_safe_supervision`.
- Hermes Telegram ya esta autenticado y puede enviar, pero el manejo conversacional por Hermes falla con `HTTP 429 usage_limit_reached`; por tanto el control movil no puede depender de Hermes LLM.
- El codigo focal de Telegram/notificaciones/canal operador-Director compila y pasa tests focales, y se genero `/srv/orquesta-self/runtime/builds/orquesta-server-telegram-verify-20260705`, pero el servidor vivo sigue usando `/srv/orquesta-self/runtime/orquesta-server-claude run` arrancado a `2026-07-05T12:18:56Z`.


## Actualizacion 2026-07-05T21:11:04Z: causa inmediata del agente

El directorio runtime del agente contiene `codex_stderr.log`. Leido con permisos de servidor, muestra que el agente arranco pero Codex CLI devolvio `401 Unauthorized` con `token_invalidated` y despues `refresh_token_invalidated`.

Conclusion actual: ademas de la invisibilidad/proyeccion del run, hay un bloqueo externo de autenticacion en el proveedor usado por Orquesta. El rework no puede completar hasta reautenticar el `CODEX_HOME` del servidor o cambiar a un backend/proveedor valido. Este bloqueo no invalida los tests focales ya pasados, pero impide afirmar que Orquesta este programando ahora mismo.

## Impacto

- Falso avance operativo: hay codigo y resultados `complete`, pero no hay prueba de runtime vivo no-LLM para Telegram.
- Alberto puede recibir mensajes Telegram por `hermes send`, pero el control desde movil no es fiable mientras dependa de Hermes LLM con cuota agotada.
- La autoprogramacion no es totalmente observable ni autonomamente reconciliada en este caso.

## Hipotesis

- El prepare-run ha caido a ruta legacy/continue sin materializar goal-first observable.
- El supervisor no avanza por el diagnostico `run_stats_required_for_safe_supervision` o por una frontera de proveedor/runtime.
- Falta integrar y desplegar un worker/endpoint Telegram no-LLM en el servidor vivo.

## Estado

Cerrada en codigo local para la parte `accepted invisible` el 2026-07-06.
Sigue abierta operativamente hasta desplegar en remoto y validar Telegram real.
Requiere una de estas salidas verificables:

1. Orquesta ejecuta el rework no-LLM y deja resultado durable con tests y despliegue; o
2. Se implementa/despliega un puente temporal no-LLM documentado, y se deja tarea de producto para integrarlo en Orquesta; o
3. Se corrige la proyeccion/reconciliacion para que el run aceptado sea observable y supervisable. Esta salida queda cerrada en codigo local por D1, pendiente de despliegue remoto.

## Avance manual 2026-07-06: endpoint no-LLM en servidor

Se materializa una primera entrada no-LLM propia de Orquesta:

- Ruta HTTP opt-in: `POST /api/v0/operator/telegram/update`.
- Montaje: solo aparece cuando `telegram_operator.enabled=true` en
  `orquesta.config.json` y el wiring Telegram esta activo.
- Entrada aceptada: update Telegram real (`update_id`, `message.chat.id`,
  `message.text`) o payload compacto (`update_ref`, `chat_ref`, `text`).
- Seguridad: el adaptador existente valida `authorized_chat_refs` antes de
  parsear comandos; si falta configuracion, la ruta devuelve `blocked` con
  `telegram_inodo_bot_link_missing` y `missing_fields`, no queda invisible.
- Salida: devuelve JSON compacto y puede enviar respuesta por puerto
  `hermesTelegramSendPortV0` sin LLM. El comando `/msg` usa el servicio real
  `OperatorDirectorMessage`, no una ruta inventada.
- Avance posterior: `cmd/orquesta-server` ya cablea un sender Bot API directo
  (`telegramBotAPISenderV0`) cuando existe `telegram_operator.token`; no anade
  variables `ORQUESTA_*` nuevas y no pasa Telegram al core.
- Actualizacion Codex 2026-07-09: `enabled` y `token` dejan de tener override
  `ORQUESTA_TELEGRAM_OPERATOR_*`; la ruta se gobierna solo por
  `telegram_operator.*` en `orquesta.config.json`.

Evidencia local:

- `TestTelegramOperatorUpdateHTTPV0DespachaUpdateAutorizadoSinLLM`.
- `TestTelegramOperatorUpdateHTTPV0BloqueoConfigVisible`.
- `TestTelegramBotAPISenderV0EnviaMensajeSinExponerTokenEnReceipt`.
- `TestTelegramBotAPISenderV0OcultaTokenEnError`.
- `TestTelegramBotAPISenderFromProjectConfigFileV0EsOptInPorToken`.
- `GOFLAGS=-buildvcs=false go test -count=1 ./cmd/orquesta-server -run 'TestTelegramOperator|TestOperatorDirector|TestOperatorNotificationHermes|TestEnvVarsOrquestaRatchetMEJ106V0'`.
- `GOFLAGS=-buildvcs=false go test -count=1 ./cmd/orquesta-server -run 'TestTelegram(BotAPI|Operator)|TestOperatorNotificationHermes'`.
- `scripts/orquesta_metricas_deuda.sh --json` baja a
  `env_vars_orquesta=511` tras retirar la ventana temporal de envs Telegram.
- `git diff --check`.

Residual: no se ha desplegado este ultimo commit en `srv1651826` ni se ha
validado con Telegram real/Bot API contra el servidor vivo. La cuota/auth del
proveedor solo afecta a agentes de programacion; no deberia bloquear los
comandos Telegram no-LLM tras desplegar el binario nuevo. Quedan pendientes:
pull/build/restart solo de Orquesta en remoto, alta de webhook o poller de
updates Telegram y prueba real desde el movil.

## Actualizacion 2026-07-06: D1 accepted legacy visible

Diagnostico nuevo: el fallo no estaba solo en Telegram. La ruta legacy de
`prepare-run` aceptaba el encargo, guardaba `workflow_task_refs` y encolaba el
run con razon `autoprogramming_prepare_run`, pero la lectura publica de cola no
transportaba `run_ref` hasta el store. Ademas, `orquesta-run-file` aplicaba el
limite antes de poder filtrar por run exacto; por tanto un run aceptado podia
quedar fuera de `autoprogramming/status` si habia candidatos delante.

Cierre de codigo aplicado:

- `RunQueueReadRequestV0` transporta `RunRef`.
- `RunSchedulingCandidateV0` conserva `Reason`.
- `orquesta-run-file` y `orquesta-run-memory` filtran por `RunRef` antes de
  rankear/limitar.
- `orquesta-mcp` pasa `run_ref` desde `autoprogramming/status` y expone
  `reason` en candidatos compactos.
- `autoprogramming/status` emite diagnostico especifico
  `autoprogramming_prepare_run_pending_dispatch` cuando ve un run aceptado por
  `prepare-run` aun en cola.
- `enqueuePreparedRunV0` hace readback inmediato tras `SetRunPriorityV0`; si el
  run aceptado no puede verse, devuelve error
  `autoprogramming_prepare_run_visibility_error` y no deja un
  `accepted=true` invisible.

Pruebas verdes:

- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestCodexStackAutoprogrammingPrepareRunAPIV0AcceptedLegacyVisibleEnStatusV0|TestCodexStackAutoprogrammingPrepareRunAPIV0NoAceptaSiColaNoProyectaRunV0|TestCodexStackAutoprogrammingPrepareRunAPIV0PreparaRunYSupervisorArranca'`
- `go test -count=1 ./modulos/orquesta-run-memory ./modulos/orquesta-run-file ./modulos/orquesta-mcp -run 'TestRunMemoryStoreListSchedulingCandidatesFiltraRunRefAntesDeLimitV0|TestRunFileStoreListSchedulingCandidatesFiltraRunRefAntesDeLimitV0|TestMCPRunQueuePriorityExecutorV0RankTransportaRunRefExactoV0|TestMCPRunQueuePriorityDescriptorV0DeclaraEvidenciaDeCandidatos|TestMCPAutoprogrammingStatusExecutorV0DiagnosticaQueuedNotDispatchedV0'`
- `go test -count=1 ./modulos/orquesta-run-queue ./modulos/orquesta-run-memory ./modulos/orquesta-run-file ./modulos/orquesta-mcp ./modulos/orquesta-app-codex-stack`
- `git diff --check`

Residual separado: si el agente llega a ejecutarse, el remoto aun puede fallar
por `401 Unauthorized token_invalidated` en el proveedor Codex del servidor. Ese
bloqueo esta documentado como bug de autenticacion independiente y no invalida
el cierre local de la invisibilidad.

## Actualizacion 2026-07-08: despliegue D1 y bloqueo de config Telegram

Verificacion local/remota posterior:

- El codigo de D1 `accepted invisible` ya esta integrado y desplegado en el
  servidor remoto usado por Orquesta (`75db992288`). El binario activo reporta
  hash `39b5007cff0f59c1d262bd41b2d03a145f6d8fe3d754cbeeb292068dc36f7cfe`.
- Pruebas focales verdes para visibilidad de `prepare-run`, stores de cola,
  MCP status y Telegram no-LLM:
  `go test -count=1 ./modulos/orquesta-app-codex-stack ./modulos/orquesta-run-memory ./modulos/orquesta-run-file ./modulos/orquesta-mcp -run 'TestCodexStackAutoprogrammingPrepareRunAPIV0AcceptedLegacyVisibleEnStatusV0|TestCodexStackAutoprogrammingPrepareRunAPIV0NoAceptaSiColaNoProyectaRunV0|TestCodexStackAutoprogrammingPrepareRunAPIV0PreparaRunYSupervisorArranca|TestRunMemoryStoreListSchedulingCandidatesFiltraRunRefAntesDeLimitV0|TestRunFileStoreListSchedulingCandidatesFiltraRunRefAntesDeLimitV0|TestMCPRunQueuePriorityExecutorV0RankTransportaRunRefExactoV0|TestMCPRunQueuePriorityDescriptorV0DeclaraEvidenciaDeCandidatos|TestMCPAutoprogrammingStatusExecutorV0DiagnosticaQueuedNotDispatchedV0'`
  y
  `go test -count=1 ./cmd/orquesta-server -run 'TestTelegram(BotAPI|Operator)|TestOperatorNotificationHermes'`.
- La prueba real contra el servidor vivo devolvio `404` en
  `POST /api/v0/operator/telegram/update`. Diagnostico: el servicio estaba
  arrancado sin `orquesta.config.json`; la ruta Telegram es opt-in y no se monta
  si `telegram_operator.enabled=true` no llega a la composicion.
- No se encontro `orquesta.config.json` canonico en
  `/srv/orquesta-self/worktrees/pilot-remoto-1`; los request files encontrados
  solo contienen objetivos, no credenciales ni token Telegram.

Fix local aplicado para el siguiente deploy: `scripts/orquesta_server_ctl.sh`
acepta `ORQUESTA_CTL_CONFIG` y, si no se indica, usa
`$ORQUESTA_CTL_WORKDIR/orquesta.config.json` cuando exista. El test
`scripts/test_orquesta_server_ctl.sh` cubre que `ctl start` no pase `--config`
sin fichero y si pase `run --config <workdir>/orquesta.config.json` cuando el
fichero existe.

Residual operativo: Telegram real sigue abierto hasta crear/instalar config
canonica con `telegram_operator.enabled=true`, token secreto, chats
autorizados, webhook o poller, y prueba desde el movil. No declarar este frente
cerrado solo por tener endpoint y `ctl` preparado.

## Refs

- request: `request-ref-remoto-telegram-nollm-runtime-20260705-001`
- workflow task: `task-autoprogramming-c20a585ff5c3-g01`
- wait agent: `agent-ref-task-autoprogramming-c20a585ff5c3-g01`
- response: `/srv/orquesta-self/operator-requests/20260705/prepare_run_telegram_nollm_runtime_20260705.response.json`
- build verificado: `/srv/orquesta-self/runtime/builds/orquesta-server-telegram-verify-20260705`
