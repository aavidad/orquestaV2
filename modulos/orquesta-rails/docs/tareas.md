# Tareas locales: orquesta-rails

## RAILS-015 detail-rails-reactivation-reconcile

Estado: cerrada documentalmente el 2026-05-27.

Objetivo: reconciliar T15 tras intentos cerrados de reactivacion de
`detalle_prohibido`, sin reabrir codigo ni ampliar scopes fuera del write-set.

Write-set:

- `modulos/orquesta-rails/README.md`
- `modulos/orquesta-rails/docs/tareas.md`
- `modulos/orquesta-rails/docs/decisiones.md`
- `modulos/orquesta-rails/docs/pruebas.md`
- docs locales de consumidores asignados

Contrato: `orquesta-rails` conserva helpers comunes por campo; tolera refs
opacas y vocabulario operativo, y corta valores sensibles efectivos,
material crudo o efectos externos no autorizados.

Nota vigente 2026-06-04: la reactivacion de T15 queda historica. Hasta nueva
orden, los rails blandos no cortan y no se reactivan por variables de entorno.
`ORQUESTA_RAILS_MODE=enforced` se normaliza a `offline`. La redaccion
no bloqueante queda separada: puede sanitizar valores sensibles efectivos sin
usar marcadores genericos ni parar agentes.

Validacion vigente: build focal y prueba viva por API contra servidor temporal.
No reutilizar tests antiguos de bloqueo como criterio de cierre mientras la
politica productiva sea rails offline.

Bloqueos: los registros globales fuera del write-set (`docs/rails`, rail
errors y duplicaciones) quedan para tarea con alcance explicito; no son permiso
para relanzar T15.
