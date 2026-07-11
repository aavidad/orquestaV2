# Incidencia 208C: control y shutdown no cerraban en la primera orden

Fecha: 2026-07-11.
Estado: cerrado localmente por componentes; pendiente smoke API tras commit.
Alcance: control plane local. Sin remoto, OPES ni `uso-app`.

## Reproduccion

Un backend goal desaparecia, pero el primer `runs/control` devolvia 409 porque
la observacion `afterGoal` seguia `running`. `observe` reconciliaba despues el
goal a `blocked`, un segundo control terminaba y shutdown podia responder
`ready` conservando un snapshot activo previo. La instancia necesitaba una
intervencion adicional.

## Causas estructurales

1. La evidencia posterior a la orden podia quedar stale aunque el actuador de
   proceso confirmase cleanup.
2. Ausencia de una coincidencia en `ActiveShutdownWorkReader` se confundia con
   prueba de parada, incluso si otra fuente acababa de observar backend activo.
3. Un fallo parcial entre persistir el goal y completar run-control se absorbia
   y podia publicar `stopped`; el replay no terminaba la segunda escritura.
4. Un snapshot shutdown vacio exitoso no limpiaba el snapshot activo anterior,
   y el wrapper podia devolver el cuerpo upstream `shutdown_ready=true` aunque
   conservara el freeze.

## Cierre local

- El escalador solo confirma `Stopped=true` cuando limpia un conjunto
  atribuible, el recuento acredita todas las entradas y una relectura no deja
  residuos. Snapshot vacio, sin match o cleanup no confirmado quedan
  residuales.
- La confirmacion tipada del escalador puede superar un `afterGoal` stale sin
  depender de consumo alto. Goal y run-control terminan en la misma orden.
- Fallos de `SaveGoalWorkStateV0` o `CompleteRunControlV0` devuelven error y
  estado solicitado, nunca terminal. El mismo idempotency key reanuda el cierre
  si el goal ya conserva la evidencia de reconcile.
- Shutdown persiste tambien snapshots vacios para limpiar refs previas. Si
  queda trabajo, reescribe la respuesta a `stop_pending` y
  `shutdown_ready=false`; solo un estado limpio solicita salida.

## Evidencia

Pruebas focales cubren: snapshot sin match, cleanup no confirmado, consumo
cero, save fallido, complete fallido, replay del fallo parcial, snapshot activo
seguido de vacio y salida solo al quedar limpio.

```text
go test -count=1 ./modulos/orquesta-mcp
go test -count=1 ./modulos/orquesta-app-codex-stack
go test -count=1 ./modulos/orquesta-server
go test -race -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-server
```

## Residual operativo

Tras integrar el commit, ejecutar una secuencia API local con backend real
aislado: una sola orden forzada debe terminalizar, y un solo shutdown debe
devolver `exit_pending=true` y terminar `RunV0`. Esa validacion no autoriza
deploy remoto ni acceso a otras aplicaciones.
