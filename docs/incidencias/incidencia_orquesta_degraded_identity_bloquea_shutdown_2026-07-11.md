# Incidencia: degraded identity bloquea shutdown

Fecha: 2026-07-11. Estado: cerrado localmente. ID:
`BUG-ORQ-20260711-208Z`.

## Reproduccion local aislada

Se arranco el binario actual en un worktree temporal sin upstream. La
readiness publico correctamente `degraded_identity` con
`worktree_ref_not_canonical` y bloqueo el lanzamiento de trabajo.

La misma composicion rechazo:

`POST /api/v0/server/shutdown`

con `server_work_launch_degraded_identity`, aunque la peticion era de parada
forzosa e idempotente. Se termino la instancia mediante SIGINT cooperativo en
la sesion propietaria y se verifico que el PID salio sin procesos residuales.

## Impacto estructural

La identidad degradada debe impedir prepare-run, dispatch y efectos externos,
pero no la parada segura. Bloquear shutdown obliga al operador a salir del
control plane justo cuando la composicion esta degradada y repite el patron de
control que depende del componente que necesita detener.

## Criterio de cierre

- `shutdown` permanece disponible bajo `degraded_identity` con contrato de
  identidad/idempotencia propio.
- No se habilita ninguna ruta de trabajo, provider, deploy ni mutacion ajena.
- Prueba HTTP reproduce readiness degradada, shutdown aceptado y cierre del
  proceso/cleanup; las rutas de prepare-run siguen rechazadas.
- No se toca `uso-app`, OPES ni servicios externos.

## Cierre local

El handler degradado conserva health/status/readiness y el bloqueo 503 de
prepare-run y de todas las rutas de trabajo. Solo `POST
/api/v0/server/shutdown` acepta una parada segura, responde `ready` con
`exit_pending`, y usa `sync.Once` para solicitar una unica vez el cierre
cooperativo del `http.Server`. Otros metodos sobre esa ruta siguen bloqueados.

Pruebas:

```text
TestDegradedIdentityHTTPHandlerV0ShutdownAceptaPOSTIdempotenteV0
TestDegradedIdentityHTTPHandlerV0ShutdownCierraServidorCooperativamenteV0
go test -count=1 ./cmd/orquesta-server
```

No se enviaron señales de proceso ni se tocaron remoto, OPES o `uso-app`.
