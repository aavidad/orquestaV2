# Contratos: orquesta-director-supervised-burst

## RunDirectorSupervisedBurstV0

Entrada canonica: `DirectorSupervisedBurstInputV0`.

Campos obligatorios:

- `step_input_builder`: puerto que prepara el input fresco del paso.
- `step_executor`: puerto que ejecuta `ExecuteDirectorCycleStepV0`.
- `supervisor`: puerto que decide la siguiente accion.
- `run_ref`: run objetivo.
- `max_steps`: presupuesto maximo de la rafaga.

Salida:

- `executed_steps`: pasos realmente ejecutados.
- `final_action`: ultima decision del supervisor.
- `steps`: traza compacta de paso, status, decision y error publico si existe.

Invariantes:

- No espera ni reintenta por tiempo.
- No ejecuta mas de `max_steps`.
- No continua si el supervisor devuelve cualquier accion distinta de `continue`.
- No despacha outbox ni registra ACK.
- No genera candidates ni lee estado productivo por si mismo.
- No contiene DB, provider, modelo, HOME, OAuth, secretos, prompts ni transcripts.

Puertos:

- `DirectorCycleStepInputBuilderPortV0`
- `DirectorCycleStepExecutorPortV0`
- `DirectorSupervisorPolicyPortV0`

Errores publicos:

- `director_supervised_burst_invalido`
- `director_supervised_burst_step_input`
- `director_supervised_burst_step`
- `director_supervised_burst_supervisor`

## No Contratos

No forman parte de este modulo:

- daemon operativo;
- servidor MCP/REST;
- dispatch/ACK real;
- persistence productiva;
- seleccion de provider/model/HOME/cuota;
- construccion de candidates.
