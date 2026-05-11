# Contexto Codex: orquesta-director-supervisor

Lee primero este archivo y `README.md`. Despues lee solo los docs locales necesarios para tu microtarea.

## Reglas comunes

- Contexto pequeno: este modulo decide si un paso del director puede repetirse.
- Hexagonal siempre: no conoce DB, runtime, proveedor, HOME, OAuth ni procesos.
- Funciones pequenas, ficheros pequenos y pruebas de invariantes.
- No cruces `internal/`, structs privados ni detalles internos de otro modulo.
- Si necesitas informacion de otro grupo, emite una `CONSULTA AL DIRECTOR`.

## Alcance local

- Clasificar el resultado de `ExecuteDirectorCycleStepV0`.
- Decidir `continue`, espera, bloqueo, pregunta al director o parada.
- Aplicar presupuesto maximo de pasos para evitar bucles.
- Devolver una decision compacta y serializable para un supervisor externo.

## Prohibido

- Ejecutar bucles, goroutines, sleeps, timers o daemon.
- Llamar scheduler, workflow, ledger, DB, runtime o filesystem productivo.
- Despachar outbox, registrar ACK o arrancar agentes.
- Elegir modelos, cuotas, HOME, OAuth o proveedores.

## Entrega

Cada cambio debe incluir contrato local, prueba local y actualizacion del verificador si entra en nucleo.
