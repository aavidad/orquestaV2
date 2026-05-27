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

Validacion: `go test -count=1 ./modulos/orquesta-rails ./modulos/orquesta-core-workflow ./modulos/orquesta-context ./modulos/orquesta-director-agent ./cmd/orquesta-server`.

Bloqueos: los registros globales fuera del write-set (`docs/rails`, rail
errors y duplicaciones) quedan para tarea con alcance explicito; no son permiso
para relanzar T15.
