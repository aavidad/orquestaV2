# Tareas: orquesta-http-gateway

## Cerrado

- Crear contexto local del modulo.
- Definir rutas publicas v0.
- Implementar constructor de mux por handlers inyectados.
- Cubrir registro, preservacion de request y 404 en tests.
- Anadir test de arquitectura contra imports prohibidos en codigo productivo.
- Anadir ruta web `/director-stats` para montar panel de estadisticas sin
  mezclarlo con la API REST `/api/v0/director/stats`.
- Anadir ruta web `/run-control` para montar panel de control sin mezclarlo con
  la API REST/MCP `/api/v0/runs/control`.
- Anadir ruta web `/run-queue` para montar panel de cola multiapp sin mezclarlo
  con la API REST/MCP `/api/v0/runs/queue/priority`.
- Anadir ruta REST `/api/v0/domain-work` como handler inyectado para el bridge
  MCP de trabajo de dominio externo.
- Cubrir que `/api/v0/domain-work` se registra solo con handler inyectado y
  queda en 404 cuando no se inyecta handler.
- Clasificar `/api/v0/runs/supervise` y
  `/api/v0/autoprogramming/supervise` como mutaciones publicas para que el
  guard de intencion browser coincida con el manifiesto de control-plane.

## Pendiente

- Mantener este modulo como composition helper fino cuando aparezcan nuevas
  superficies HTTP.
- Revisar versionado si una ruta cambia de contrato.
- Mantener el gateway sin imports de `orquesta-mcp`, `orquesta-domain-work`,
  conectores, DB, runtime ni configuracion concreta.
