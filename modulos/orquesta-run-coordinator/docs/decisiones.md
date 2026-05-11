# Decisiones

## Estado de control ausente

Si `RunControlReaderPortV0` devuelve `RunControlStateNotFoundErrorV0`, el
coordinador usa `DefaultRunControlStateV0(runRef)`. Esto evita bloquear runs
que aun no tienen una orden explicita de pausa, cancelacion o stop.

## Resultado compacto

El tick no devuelve candidatos completos. Expone solo `RunRef`, `AppRef`, rank,
prioridad, aging boost, skips y ejecuciones para facilitar logs y trazabilidad
sin acoplar al drainer.

## Sin adaptadores

El modulo solo orquesta puertos. La lectura fisica de cola, el control durable y
la ejecucion real pertenecen a modulos exteriores.

## Limites de drain

El coordinador no decide cuantos pasos internos debe ejecutar una run. Solo
transporta `DrainLimits` al `RunDrainerPortV0`. La composition exterior fija
presupuestos conservadores por tick para que una app no monopolice la cola
global.

## Exclusion temporal

`ExcludeRunRefs` permite que un supervisor haga una pasada justa sin mutar la
cola. El coordinador conserva el ranking visible y devuelve la run excluida
como skip `run_excluded`; la politica de exclusion vive fuera del coordinador.
