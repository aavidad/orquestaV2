# Incidencia: retencion sin cuota de runtimes Codex

Fecha: 2026-07-10. Estado: abierta, operativa/composicion.

## Evidencia local

Tras retirar solo las caches de compilacion creadas en este corte,
`.orquesta-runtime` conserva 19 GB; `codex-waves` concentra 16 GB. Hay 406
ficheros `codex_stderr.log` (627.207.480 bytes) y 399 directorios `home` o
`codex-home`. La raiz dominante es
`.orquesta-runtime/codex-waves/psicologia` (16 GB); `opes-a1-dual-20260520`
ocupa 758 MB. Las cinco olas de este corte suman menos de 1,4 MB.

No se borraron esos runtimes: pueden contener evidencias citadas, ACKs,
receipts, estados de cierre o credenciales aisladas cuya retencion no se ha
clasificado. No hay procesos de las cinco olas de este corte vivos.

## Revision del retenedor existente

El script existente `scripts/orquesta_runtime_retention.sh` ya implementa
dry-run por defecto, confirmacion explicita, bloqueo de ACK/plan/outbox/
checkpoint/proceso vivo y recibo textual. Su propio test pasa. El dry-run real
de 2026-07-10 clasifica `opes-a1-dual-20260520` (758 MB) como candidato seguro,
pero bloquea `codex-waves/psicologia` (16 GB) con
`wave_registry_missing`. Tambien bloquea los artefactos con ACK o estado vivo
pendiente; no hubo borrado.

La laguna estructural no es ausencia total de politica: los runtimes legacy sin
registry no pueden probar identidad, estado ni referencias y quedan retenidos
indefinidamente. Sus 399 homes aislados explican la mayor parte del volumen.

## Correccion requerida

Extender el retenedor, no duplicarlo: importar/clasificar el contenedor legacy
sin registry en un manifest de retencion con identidad y referencias; separar
evidencia durable de cache/home; exportar o compactar lo necesario; y solo
habilitar su purga tras confirmar ausencia de procesos, sesiones, locks, ACKs,
outbox, estados y citas documentales. Debe conservar dry-run, recibo y la
proteccion de rutas ajenas a Orquesta. No pertenece al core.
