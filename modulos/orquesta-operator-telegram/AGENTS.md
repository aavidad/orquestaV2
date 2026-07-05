# Contexto Codex: orquesta-operator-telegram

Lee primero `AGENTS.md` raiz. Este modulo es un adaptador de operador para
Telegram y no pertenece al core de Orquesta.

## Responsabilidad

- Traducir mensajes de Telegram ya recibidos a comandos seguros de operador.
- Validar chat autorizado y confirmaciones explicitas antes de efectos.
- Redactar token, chat ids y payloads sensibles en respuestas publicas.
- Reutilizar un enlace/configuracion de bot existente, como Inodo Bot, por
  configuracion de composicion.

## Prohibido

- No guardar tokens reales, HOME, OAuth ni transcripts completos.
- No tocar core/workflow/domain-work ni acoplarse a rutas internas del servidor.
- No hacer red real desde tests; usa puertos/fakes.
