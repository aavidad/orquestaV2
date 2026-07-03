# Contexto local: orquesta-runtime-codex-appserver

## Responsabilidad

Adaptador Codex `app-server` para Goal. Encapsula protocolo RPC/WebSocket,
backend tmux opt-in, limpieza de procesos propios y diagnosticos del backend.

## Reglas

- No importar `cmd`, OPES, web, MCP, stores concretos ni persistencia producto.
- El runtime real entra por puertos/configuracion de composicion.
- Mantener decisiones de estado del backend como funciones puras cuando sea
  posible; observacion de tmux/procesos/socket queda separada de transicion.
- No matar procesos sin evidencia de propiedad: owner marker, runtime dir propio
  o socket configurado.
