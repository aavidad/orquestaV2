# Pruebas locales: orquesta-rails

## RAILS-P015 matriz de detalle por campo

Tipo: contract | regression

Comando: `go test -count=1 ./modulos/orquesta-rails ./modulos/orquesta-core-workflow ./modulos/orquesta-context ./modulos/orquesta-director-agent ./cmd/orquesta-server`

Evidencia esperada:

- `orquesta-rails` permite vocabulario operativo opaco como `runtime`,
  `provider`, `model`, `codex`, `git`, `db`, `sql`, `prompt policy` y
  `transcript policy`;
- rechaza valores sensibles efectivos, rutas privadas, material crudo y
  secretos por frontera/campo activado;
- el servidor mantiene default `on` acotado en produccion y `off` en modo
  `programming`.

Ultima ejecucion documentada: 2026-05-27, ACKs cerrados de T15 con la suite
requerida pasada.

Riesgos: no sustituye la matriz de un nuevo scope; cualquier frontera adicional
debe aportar casos externos antes de endurecer.
