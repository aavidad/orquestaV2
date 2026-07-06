# Telegram Inodo Bot para Orquesta - 2026-07-05

Este runbook describe el adaptador operador Telegram para manejar Orquesta desde
un bot existente, preferentemente Inodo Bot. No crea un bot nuevo: reutiliza el
enlace/token/chat ya autorizado por la composicion del servidor.

## Configuracion canonica

En `orquesta.config.json`:

```json
{
  "schema_version": "orquesta_config.v0",
  "telegram_operator": {
    "enabled": true,
    "bot_link_ref": "inodo-bot-link-ref",
    "token": "<token del bot existente>",
    "authorized_chat_refs": ["chat-ref-operador"],
    "notification_target_ref": "telegram:39995054",
    "require_confirmation": true
  }
}
```

Equivalentes por entorno:

- `ORQUESTA_TELEGRAM_OPERATOR_ENABLED=true`
- `ORQUESTA_TELEGRAM_OPERATOR_BOT_LINK_REF=<ref opaca del enlace Inodo Bot>`
- `ORQUESTA_TELEGRAM_OPERATOR_TOKEN=<token del bot existente>`
- `ORQUESTA_TELEGRAM_OPERATOR_AUTHORIZED_CHAT_REFS=<chat refs separados por coma>`
- `ORQUESTA_TELEGRAM_OPERATOR_NOTIFICATION_TARGET_REF=<chat ref destino>`
- `ORQUESTA_TELEGRAM_OPERATOR_REQUIRE_CONFIRMATION=true`

El token, el enlace real y los chats autorizados se publican en
`effective_config` solo como valores redactados. No deben aparecer en logs,
tests ni repositorio.

## Bloqueo esperado

Si el operador activa Telegram pero falta enlace, token o chat autorizado, el
wiring debe quedar bloqueado con:

```text
telegram_inodo_bot_link_missing
```

Datos exactos que debe aportar el operador:

- `bot_link_ref`: ref opaca de la configuracion/enlace Inodo Bot existente.
- `token`: token real del bot existente, solo por secreto/env/config local.
- `authorized_chat_refs`: refs opacas de chats permitidos.

## Comandos seguros

- `/status [run-ref]`: consulta estado compacto.
- `/queue [subject-ref]`: consulta cola/outbox compacto.
- `/launch_task <spec compacto>`: lanza una tarea por el puerto de operador.
- `/observe_goal <goal-ref>`: observa un goal concreto.
- `/msg <target-ref> <texto>`: envia una instruccion compacta al canal
  operador-Director.
- `/stop <target-ref> confirm=<ref>`: detiene/controla con confirmacion.
- `/handoff [target-ref]`: devuelve handoff compacto.

El adaptador valida el chat antes de interpretar comandos. `stop/control`
requiere confirmacion cuando `require_confirmation=true`.

## Frontera

`modulos/orquesta-operator-telegram` no hace red real ni conoce runtime interno.
La recepcion Telegram, polling/webhook y transporte HTTP quedan en composicion
opt-in. El modulo solo traduce updates autorizados a puertos de operador.

Las notificaciones terminales usan el contrato `operator_notifications.v0` y se
envian por un puerto de composicion, compatible con Hermes send sin depender de
crons Hermes ni de auth Codex.
