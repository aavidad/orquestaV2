# Telegram operador remoto Orquesta - 2026-07-05

Este runbook fija la superficie remota para Alberto por Telegram. La ruta es
opt-in de composicion: Telegram no entra en core/workflow/domain-work y no
depende de auth Codex ni de crons LLM de Hermes.

## Notificaciones terminales

Cada cierre de tarea, goal o run debe proyectarse como
`operator_notifications.v0` con:

- `kind`: `task`, `goal` o `run`.
- `subject_ref`: ref opaca principal.
- `status`: `complete`, `blocked`, `invalid`, `stopped` o `canceled`.
- `dedupe_key`: `kind:subject_ref:status` salvo que el emisor aporte
  `event_ref`.

El servicio `modulos/orquesta-operator-notifications` compacta el texto para
Telegram, deduplica por clave estable y llama a un puerto de envio. En
`cmd/orquesta-server`, `operatorTaskTerminalNotifierV0` adapta estados
terminales de Goal/RunControl/tarea a ese contrato. El puerto
`SendTelegramMessageV0` puede apuntar a Hermes send como adaptador externo o al
sender Bot API directo de Orquesta; no requiere crons LLM ni cuota de proveedor.

## Configuracion canonica

En `orquesta.config.json`:

```json
{
  "telegram_operator": {
    "enabled": true,
    "bot_link_ref": "inodo-bot-link-ref",
    "token": "<secreto local>",
    "authorized_chat_refs": ["telegram:39995054"],
    "notification_target_ref": "telegram:39995054",
    "require_confirmation": true
  }
}
```

Variables equivalentes de operacion:

- `ORQUESTA_TELEGRAM_OPERATOR_ENABLED`
- `ORQUESTA_TELEGRAM_OPERATOR_TOKEN`

`bot_link_ref`, `authorized_chat_refs`, `notification_target_ref` y
`require_confirmation` quedan solo en `orquesta.config.json` para no duplicar
superficie de entorno. Token, chat refs y target de notificacion se publican
redactados en `effective_config`.

## Comandos autorizados

- Endpoint no-LLM de Orquesta: `POST /api/v0/operator/telegram/update`.
  Acepta update Telegram (`update_id`, `message.chat.id`, `message.text`) o
  payload compacto (`update_ref`, `chat_ref`, `text`). La ruta esta montada solo
  si `telegram_operator.enabled=true`; si falta configuracion devuelve
  `blocked` con campos pendientes.
- En el servidor, la ruta responde al chat por Bot API directo cuando
  `telegram_operator.token` esta configurado. El token vive en composicion
  (`orquesta.config.json` o `ORQUESTA_TELEGRAM_OPERATOR_TOKEN`) y no pasa al
  nucleo.
- `/status [run-ref]`: estado compacto.
- `/queue [subject-ref]`: cola/outbox compactos.
- `/observe_goal <goal-ref>`: observa un goal/run goal-first.
- `/launch_task <spec compacto>`: lanza una instruccion por el puerto operador.
- `/msg <target-ref> <texto>`: envia una instruccion al canal
  operador-Director ya cableado.
- `/stop <target-ref> confirm=<ref>`: solicita parada/control con confirmacion.
- `/handoff [target-ref]`: handoff compacto.

La autorizacion por chat ocurre antes de parsear comandos. Los efectos de
control requieren confirmacion cuando `require_confirmation=true`.

## Evidencia local

Tests focales añadidos:

- `TestOperatorNotificationV0EnviaTerminalDeduplicado`
- `TestOperatorNotificationServerV0NotificaGoalYRunTerminalDeduplicado`
- `TestOperatorNotificationHermesTelegramNotifierV0UsaSendSinAuthCodex`
- `TestAdapterV0DespachaMensajeAlCanalDirector`
- `TestTelegramOperatorUpdateHTTPV0DespachaUpdateAutorizadoSinLLM`
- `TestTelegramOperatorUpdateHTTPV0BloqueoConfigVisible`
- `TestTelegramBotAPISenderV0EnviaMensajeSinExponerTokenEnReceipt`
- `TestTelegramBotAPISenderV0OcultaTokenEnError`
- `TestTelegramBotAPISenderFromProjectConfigFileV0EsOptInPorToken`

## Pendiente remoto

El codigo local ya tiene endpoint y sender Bot API. Para que Alberto lo use
desde el movil falta desplegar el commit en `srv1651826`, reiniciar solo
Orquesta y validar una llamada real a `/api/v0/operator/telegram/update` desde
Telegram webhook/poller. Si el proveedor Codex/Claude/Gemini no tiene cuota, los
comandos no-LLM deben seguir funcionando; solo fallaran las acciones que lancen
agentes de programacion.
