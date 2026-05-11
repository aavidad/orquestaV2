# Tareas: orquesta-http-gateway

## Cerrado

- Crear contexto local del modulo.
- Definir rutas publicas v0.
- Implementar constructor de mux por handlers inyectados.
- Cubrir registro, preservacion de request y 404 en tests.
- Anadir test de arquitectura contra imports prohibidos en codigo productivo.
- Anadir ruta web `/director-stats` para montar panel de estadisticas sin
  mezclarlo con la API REST `/api/v0/director/stats`.

## Pendiente

- Mantener este modulo como composition helper fino cuando aparezcan nuevas
  superficies HTTP.
- Revisar versionado si una ruta cambia de contrato.
