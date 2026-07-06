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

Abierta con avance de codigo local integrado el 2026-07-06. Requiere una de
estas salidas verificables:

1. Orquesta ejecuta el rework no-LLM y deja resultado durable con tests y despliegue; o
2. Se implementa/despliega un puente temporal no-LLM documentado, y se deja tarea de producto para integrarlo en Orquesta; o
3. Se corrige la proyeccion/reconciliacion para que el run aceptado sea observable y supervisable.

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

Evidencia local:

- `TestTelegramOperatorUpdateHTTPV0DespachaUpdateAutorizadoSinLLM`.
- `TestTelegramOperatorUpdateHTTPV0BloqueoConfigVisible`.
- `TestTelegramBotAPISenderV0EnviaMensajeSinExponerTokenEnReceipt`.
- `TestTelegramBotAPISenderV0OcultaTokenEnError`.
- `TestTelegramBotAPISenderFromProjectConfigFileV0EsOptInPorToken`.
- `GOFLAGS=-buildvcs=false go test -count=1 ./cmd/orquesta-server -run 'TestTelegramOperator|TestOperatorDirector|TestOperatorNotificationHermes|TestEnvVarsOrquestaRatchetMEJ106V0'`.
- `GOFLAGS=-buildvcs=false go test -count=1 ./cmd/orquesta-server -run 'TestTelegram(BotAPI|Operator)|TestOperatorNotificationHermes'`.
- `scripts/orquesta_metricas_deuda.sh --json` mantiene
  `env_vars_orquesta=514`.
- `git diff --check`.

Residual: no se ha desplegado este ultimo commit en `srv1651826` ni se ha
validado con Telegram real/Bot API contra el servidor vivo. La cuota/auth del
proveedor solo afecta a agentes de programacion; no deberia bloquear los
comandos Telegram no-LLM tras desplegar el binario nuevo. Quedan pendientes:
pull/build/restart solo de Orquesta en remoto, alta de webhook o poller de
updates Telegram y prueba real desde el movil.

## Refs

- request: `request-ref-remoto-telegram-nollm-runtime-20260705-001`
- workflow task: `task-autoprogramming-c20a585ff5c3-g01`
- wait agent: `agent-ref-task-autoprogramming-c20a585ff5c3-g01`
- response: `/srv/orquesta-self/operator-requests/20260705/prepare_run_telegram_nollm_runtime_20260705.response.json`
- build verificado: `/srv/orquesta-self/runtime/builds/orquesta-server-telegram-verify-20260705`
