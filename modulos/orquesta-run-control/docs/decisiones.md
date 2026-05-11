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
