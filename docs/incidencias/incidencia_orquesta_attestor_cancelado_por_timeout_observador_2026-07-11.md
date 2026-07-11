# Incidencia 208AC: el observador cancela la atestacion real a los dos segundos

Fecha: 2026-07-11
Estado: en verificacion E2E
Area: nucleo goal-first / observador residente / atestacion independiente

## Resumen

El E2E de 208AA llego a un goal real con sus dos tests autodeclarados verdes.
Al observar el cierre, el attestor independiente arranco el primer test, pero
el contexto fue cancelado a los dos segundos por
`DefaultGoalObserverTimeoutV0`. El claim durable quedo como
`goal_required_test_attestor_infrastructure_failed` y no hubo recibo.

El limite corto sirve para la respuesta HTTP y su snapshot parcial, pero no
puede gobernar el trabajo durable del observador residente: una compilacion o
suite real supera normalmente dos segundos.

## Evidencia

- runtime: `/tmp/orquesta-208aa-e2e-accepted`;
- run: `autoprog-attestor-e2e-accepted-20260711`;
- goal: `goal-ref-task-autoprogramming-3e356747e486-g01`;
- Codex termino `complete` con los dos required tests verdes y sin cambios;
- `POST /api/v0/autoprogramming/goal/observe` devolvio 504 parcial;
- claim fallido entre `01:06:57.545Z` y `01:06:59.083Z` con codigo de
  infraestructura;
- preflight y snapshot de modulos eran validos; no se genero artefacto de
  comando porque el contexto cancelo la ejecucion;
- shutdown publico `ready`, pero el proceso servidor necesito SIGINT
  cooperativo local, residual ya cubierto por 208A-D.

## Corte aplicado

La API HTTP conserva su timeout corto y snapshots parciales. El observador
residente, que es el motor autonomo de reconciliacion, usa una ventana separada
de 15 minutos para permitir atestaciones durables. Un test impide volver a
reducir esa ventana por debajo de diez minutos.

## Residual estructural

La atestacion sigue ejecutandose dentro del tick del observador y los goals se
recorren secuencialmente. El cierre completo requiere un scheduler de
atestaciones concurrente, con claims, limite por test, shutdown gobernado y
recibos independientes. Este residual no autoriza volver al timeout de dos
segundos ni aceptar resultados autodeclarados.

## Criterio de cierre

1. El observador residente completa los dos tests del E2E sin cancelacion.
2. Se persisten dos recibos independientes y el goal cierra `accepted`.
3. La API sigue devolviendo snapshot parcial acotado si el operador observa
   mientras la atestacion esta en curso.
4. No se lanza rework Codex por timeout o infraestructura del attestor.
