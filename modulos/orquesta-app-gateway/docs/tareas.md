# Tareas: orquesta-app-gateway

## Cerrado

- Crear contexto local del mini-proyecto.
- Definir `ConfigV0` y `NewHTTPHandlerV0`.
- Montar handlers API antes que handlers web.
- Crear transporte in-process para que web use REST sin loopback real.
- Probar `/nueva-app` -> REST `/api/v0/apps/director`.
- Probar `/director-stats` -> REST `/api/v0/director/stats`.
- Anadir guard de arquitectura contra legacy y conectores reales.
- Auditar cobertura de stats: el gateway no necesita campos propios porque
  conserva el bridge MCP/API con `DirectorRunStatsV0` completo.

## Pendiente

- Crear adaptador servidor externo opt-in si hace falta abrir puerto real.
- Conectar executors productivos con stores/runtime por conectores fuera de este
  modulo.
