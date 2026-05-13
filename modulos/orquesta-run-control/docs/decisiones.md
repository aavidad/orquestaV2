# Decisiones

## Hexagonal ligera

El modulo define contratos y politica pura, pero no implementa adaptadores. La frontera externa queda expresada con puertos.

## Estado minimo

`RunControlStateV0` solo contiene el estado necesario para decidir scheduling, dispatch, checkpoint, parada de agentes y terminalidad.

## `forced=true`

La fuerza vive en el estado evaluado para que `EvaluateRunControlV0` sea una funcion de un unico input y siga siendo determinista.

## Sin i18n de producto

No se crea esqueleto i18n porque este modulo no expone interfaz funcional ni textos de producto. La documentacion queda en castellano como contrato operativo minimo.

## Sin superficies externas

No se implementan DB, HTTP, MCP, runtime, scheduler interno ni workflow core.

## Escritura terminal separada

`stop_requested` y `cancel_requested` son intenciones de control, no cierres
reales. Por eso el cierre se expresa con `RunControlTerminalWriterPortV0` y
`CompleteRunControlV0`.

El nucleo o el director solo deben invocarlo despues de comprobar que no quedan
agentes vivos sin confirmar ni mensajes de parada pendientes. Esta separacion
evita parchear el estado con otro `StopRunV0` y deja claro quien toma la
decision de cierre.

## Checkpoint separado de stop

`RecordRunCheckpointV0` no cambia `status`; solo marca
`checkpoint_recorded=true` y adjunta evidencia compacta. Esto evita colar una
parada segura con otro `StopRunV0` y permite que `forced=false` tenga una
frontera verificable: primero ACK durable de checkpoint, despues stop/drain.
