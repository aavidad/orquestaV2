# Decisiones

## Pasada con presupuesto

El supervisor no es un daemon. Ejecuta una pasada acotada por `MaxTicks` y
`MaxExecutions`. El proceso que lo invoque decide cuando lanzar otra pasada.

## Sin espera interna

No hay `sleep`, goroutines ni timers. La espera de agentes externos pertenece a
los adaptadores o al drainer de cada run. Este modulo solo decide si hace otro
tick.

## No repetir runs por defecto

Dentro de una misma pasada, las runs ejecutadas se envian como
`ExcludeRunRefs` al siguiente tick. Asi una run con prioridad alta no monopoliza
la cola global si hay otras apps esperando. La cola no se muta; es una exclusion
temporal de supervision.
