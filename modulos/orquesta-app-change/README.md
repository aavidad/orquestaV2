# orquesta-app-change

Mini-proyecto para solicitar cambios sobre una app existente.

Responsabilidad:

- recibir un evento compacto de cambio a mitad de ejecucion y normalizarlo a
  solicitud;
- validar `AppChangeRequestV0`;
- persistir la solicitud por puerto inyectado;
- notificar al director por puerto inyectado;
- devolver un resultado compacto apto para web, REST y MCP.

No decide agentes, modelos, DB, runtime ni write-sets finales. Eso lo decide el
director al reentrar en Orquesta.
