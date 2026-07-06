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
terminales de Goal/RunControl/tarea a ese contrato y
`hermesTelegramNotifierV0` delega en un puerto `SendTelegramMessageV0`. El
puerto permite usar Hermes send como adaptador externo sin reactivar crons.

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
