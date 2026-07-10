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

## Hipotesis estructural

La composicion puede proyectar/copiar `CODEX_HOME` y retener stderr, wrappers y
recibos, pero no aplica una politica durable de cuota, compresion, expiracion,
anonimizacion ni exportacion de evidencia. Por tanto cada ola puede dejar un
runtime completo aunque su resultado durable ya este versionado o retenido por
otro store.

## Correccion requerida

Crear por composicion un retenedor gobernado que inventarie cada runtime por
identidad, referencias vivas y tamanos; separe evidencia durable de cache/home;
exporte o compacte lo necesario; y solo purgue despues de confirmar ausencia de
procesos, sesiones, locks, ACKs, outbox, estados y citas documentales. Debe
ofrecer dry-run, recibo de retencion/purga, limites configurados canonicamente y
prueba que protege rutas ajenas a Orquesta. No pertenece al core.
