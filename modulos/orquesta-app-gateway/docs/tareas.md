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
- Montar `/api/v0/domain-work` como bridge REST/MCP con executor inyectado.
- Probar que `/api/v0/domain-work` delega en el executor sin OPES, DB, runtime
  ni sockets reales.
- Reutilizar el mux de `orquesta-http-gateway` para el bridge domain-work:
  `orquesta-app-gateway` compone el handler MCP ya construido, sin validar ni
  transformar contratos de `DomainWorkJobRequestV0` o
  `DomainWorkArtifactSubmissionV0`.

## Pendiente

- Crear adaptador servidor externo opt-in si hace falta abrir puerto real.
- Conectar executors productivos con stores/runtime por conectores fuera de este
  modulo.
- Mantener los conectores reales de dominios externos, como OPES, fuera de este
  modulo y pasarlos por configuracion explicita desde un borde superior.
