# Contratos v0

## Estado

`RunControlStateV0` representa la vista minima de control de una run:

- `running`: permite scheduling y dispatch.
- `paused`: bloquea scheduling y dispatch, pero no es terminal.
- `stop_requested`: bloquea scheduling y dispatch; requiere checkpoint antes de parar agentes salvo `forced=true`.
- `cancel_requested`: bloquea scheduling y dispatch; requiere checkpoint antes de parar agentes salvo `forced=true`.
- `stopped`: terminal.
- `canceled`: terminal.

## Puertos

`RunControlReaderPortV0` lee estado de control por `run_ref`.

Si una run no tiene estado explicito, el adaptador debe devolver
`RunControlStateNotFoundErrorV0`. El consumidor puede tratar ese caso como
`DefaultRunControlStateV0(run_ref)`, es decir `running`, sin confundirlo con un
fallo operativo del conector.

`RunControlWriterPortV0` expone comandos de aplicacion:

- `PauseRunV0`
- `ResumeRunV0`
- `StopRunV0`
- `CancelRunV0`

`RunControlTerminalWriterPortV0` expone `CompleteRunControlV0` para marcar la
run como `stopped` o `canceled` cuando el consumidor ya ha comprobado que el
drenaje es seguro. No acepta volver a `running`, `paused`,
`stop_requested` ni `cancel_requested`.

`CompleteRunControlCommandV0` contiene:

- `run_ref`
- `target_status`: solo `stopped` o `canceled`
- `requested_by`, `reason`, `idempotency_key` y `evidence_refs`

Los puertos no definen transporte ni almacenamiento. Cualquier adaptador real debe vivir fuera de este microproyecto.

## Evaluacion

`EvaluateRunControlV0` es una funcion pura:

- no lee reloj global;
- no consulta servicios externos;
- no muta el estado recibido;
- devuelve `scheduling_allowed`, `dispatch_allowed`, `checkpoint_required`, `stop_agents_allowed` y `terminal`.

Reglas:

1. `running` permite scheduling y dispatch.
2. `paused` bloquea scheduling y dispatch sin marcar terminalidad.
3. `stop_requested` y `cancel_requested` bloquean scheduling y dispatch.
4. `stop_requested` y `cancel_requested` requieren checkpoint antes de `stop_agents_allowed` salvo `forced=true`.
5. `stopped` y `canceled` son terminales.
