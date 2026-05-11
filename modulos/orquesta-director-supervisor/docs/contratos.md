# Contratos: orquesta-director-supervisor

## DecideDirectorSupervisorNextActionV0

Entrada canonica: `DirectorSupervisorDecisionInputV0`.

Campos obligatorios:

- `run_ref`: run objetivo.
- `step_number`: numero del ultimo paso ya ejecutado.
- `max_steps`: presupuesto maximo permitido.
- `last_step_result`: salida de `ExecuteDirectorCycleStepV0`.

Campos opcionales:

- `last_error_code`: codigo publico si el ultimo paso fallo.
- `correlation_id`
- `evidence_refs`

Salida:

- `action`: `continue`, `wait_outbox`, `wait_external`, `needs_director`, `blocked`, `stop_quiescent`, `stop_max_steps` o `stop_error`.
- `should_continue`: solo `true` para `continue`.
- `autonomous_recommendation`: `continue`, `wait`, `needs_director` o `stop`.
- `reason_code`: motivo compacto.
- refs de outbox, waiting reasons y blocked refs copiadas del ultimo paso.

Invariantes:

- No ejecuta el siguiente paso.
- `autonomous_recommendation=wait` para `wait_outbox` y `wait_external`.
- `waiting_reasons=candidate_missing` bajo `needs_director` se clasifica como `wait_external` para ciclos autonomos.
- No tiene bucles, timers, daemon, goroutines ni sleeps.
- No persiste, no despacha outbox y no registra ACK.
- No llama scheduler, workflow, ledger, runtime ni filesystem.
- No contiene DB, provider, modelo, HOME, OAuth, secretos, prompts ni transcripts.
- `max_steps` debe ser explicito y no puede superar 50.

Errores publicos:

- `director_supervisor_decision_invalida`

## No Contratos

No forman parte de este modulo:

- control plane operativo;
- supervisor residente;
- conectores runtime;
- politicas de capacidad;
- dispatch de outbox;
- reintentos reales.
