# Incidencia: handoff durable de active_work en shutdown

Fecha: 2026-07-02

## Estado

Avance acotado de BUG-065.

## Problema

Si `POST /api/v0/server/shutdown` observaba `backend_still_running`, el HTTP
podia devolver el cuerpo correcto, pero el state durable del servidor solo
conservaba contadores generales de shutdown. En un timeout posterior de parada,
el operador perdia parte de la evidencia compacta del trabajo activo.

## Cambio

La capa `shutdown_freeze` ahora parsea `active_work_count` y `active_works` del
resultado HTTP de shutdown y persiste en el state refs publicas saneadas:

- `shutdown_active_work_count`
- `shutdown_active_work_refs`

No incluye sockets, rutas locales ni datos sensibles.

## Limites

Esto no fuerza la parada del backend ni sustituye la coordinacion completa de
shutdown. Conserva mejor el handoff cuando la respuesta HTTP de shutdown llego
con active work antes de que el runtime fallara por timeout.

## Pruebas

- `TestShutdownProjectionFromHTTPV0ConservaActiveWorkRefsV0`
- `go test -count=1 ./modulos/orquesta-server`
