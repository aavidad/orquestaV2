# Pruebas: orquesta-runtime-codex

## RTCODEX-001

Comando:

```bash
go test -count=1 ./modulos/orquesta-runtime-codex
```

Cobertura:

- bloquea si falta opt-in;
- bloquea si faltan command path, project workdir o runtime workdir;
- materializa packet, prompt y wrapper;
- el prompt exige `$caveman` o `compact` si esta disponible y salida minima;
- la plantilla de ACK generada incluye los `files` del write-set y los
  `tests` obligatorios esperados, no arrays vacios;
- con sandbox `workspace-write`, el wrapper autoriza `runtime_work_dir` con
  `--add-dir` para que el agente pueda escribir `agent_ack.json`;
- `decision_path` aparece como archivo de control y se marca obligatorio solo
  si objetivo o criterios de cierre lo piden;
- el prompt materializa `orquesta_shutdown_request.json` y
  `agent_shutdown_checkpoint_ack.json` como ficheros de control de cierre
  cooperativo;
- escribe request de shutdown, valida ACK `checkpoint_ready` y trata el ACK
  ausente como pendiente retryable;
- rechaza ACK de checkpoint de otro agente/run/checkpoint;
- con sandbox `workspace-write`, acepta `runtime_work_dir` externo al proyecto
  cuando el wrapper lo autoriza con `--add-dir`;
- con sandbox `workspace-write`, rechaza `runtime_work_dir` igual, interno o
  ancestro de `project_work_dir` para evitar contaminacion de contexto;
- fuera de `workspace-write`, el wrapper no anade writable roots extra;
- devuelve `ProcessRuntimeLaunchRequestV0` sin args/env operacionales;
- no filtra provider/model/HOME/OAuth al request publico de proceso;
- valida ACK/receipt Codex opt-in correcto;
- rechaza ACK corrupto o incompleto con error publico;
- rechaza ACK `completed` sin artifacts del write-set o con artifacts fuera de write-set;
- acepta ACK con artifacts concretos que encajan en write-set con glob cerrado;
- advierte en el prompt cuando hay contexto requerido truncado;
- rechaza ACK `completed` con contexto requerido truncado si no justifica
  `contexto_truncado_resuelto`;
- rechaza ACK con HOME real, token/secreto, OAuth, prompt/completion o transcript completo.
- lee `agent_ack.json` y construye `CodexDeliveryObservationV0` neutral;
- rechaza observaciones que filtren detalles prohibidos para el core, incluido
  el nombre del proveedor en refs.

Riesgo residual:

- No ejecuta Codex real por defecto.
- La prueba real pertenece al mini-proyecto E2E no controlado.
