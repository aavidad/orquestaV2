# Incidencia R4: lote amplio, configuracion y tmux transitorio

Fecha: 2026-07-11. Estado: parcialmente corregida; R4 sigue sin acreditar.

## Evidencia retenida

El runner aislado ejecuto dos pases sobre siete paquetes con cache y receipt
fuera del repositorio:

`/tmp/orquesta-r4-cleanup-batches/receipt.json`

Resultado durable: `status=failed`, `reason_code=one_or_more_batches_failed`,
14 ejecuciones, ningun pase completo. No quedan procesos del runner ni de sus
tests al cerrar la observacion.

## Hallazgos

1. Ambos pases fallaron de forma determinista por dos regresiones del corte
   R2: el T90 detecto `config_file_v0.go` por encima de 900 lineas y el
   registry contenia timeouts sin unidad en su nombre.
2. El segundo pase anadio
   `TestCleanupCodexGoalBackendAfterStartupFailureV0SoloConDaemonInactivoYGeneracionExactaV0` con
   `codex_app_server_tmux_generation_observation_transient`.

## Correccion aplicada a los hallazgos deterministas

`0b564992a` extrae el helper de listas a
`config_file_list_values_v0.go`, dejando `config_file_v0.go` en 892 lineas.
Los dos timeouts de promotion guardian pasan a `orquesta.config.json` tipado;
no tienen variable de entorno ni alias. Los focales T90, registry, guardian,
presupuesto y metricas locales pasan; la metrica queda 423 productivas y 103
solo-fixture sin subir su ratchet.

## Residual vivo: BUG-ORQ-20260711-208Y

La observacion tmux transitoria debe reproducirse con el focal aislado y
clasificarse: carrera del fixture, identidad de sesion o defecto del control
de cleanup. No se acepta reintentar el lote hasta hacerlo desaparecer; la
causa debe conservar `run_ref`, identidad/lease, estado antes/despues y log
del hijo. Hasta entonces, D3 y 208H no tienen dos pases independientes verdes.

## Retencion

No borrar `/tmp/orquesta-r4-cleanup-batches` ni su cache asociada antes de
extraer el detalle de 208Y o registrar una retencion gobernada. La ruta es
evidencia de esta incidencia, no cache anonima.
