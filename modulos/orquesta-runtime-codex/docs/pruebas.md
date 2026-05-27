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
- la plantilla de ACK generada incluye rutas concretas de archivos para
  `files`, no globs/directorios del write-set, y los `tests` obligatorios
  esperados, no arrays vacios;
- el prompt advierte que el ACK externo al proyecto debe escribirse desde el
  directorio runtime por shell y que la linea visible no sustituye el JSON;
- el prompt aclara que el write-set se escribe relativo al workdir del
  proyecto, no al directorio de control;
- el prompt prohibe listar archivos de control (`agent_ack.json`,
  `director_decisions.json`, packet, prompts, logs, checkpoints) en
  `ACK.files`;
- con sandbox `workspace-write`, el wrapper autoriza `runtime_work_dir` con
  `--add-dir` para que el agente pueda escribir `agent_ack.json`;
- `decision_path` aparece como archivo de control y se marca obligatorio solo
  si objetivo o criterios de cierre lo piden;
- para target_module de director, el prompt exige `decision_path` antes de ACK
  `completed` y pide `failed` con `CONSULTA AL DIRECTOR` si no puede emitirlo;
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
- genera `codex_usage_accounting.json` redactado desde salida permitida del
  runtime y conserva el exit code real de Codex;
- no filtra provider/model/HOME/OAuth al request publico de proceso;
- valida ACK/receipt Codex opt-in correcto;
- normaliza `task_ref` y `correlation_id` desde el descriptor si `request_id`,
  `ack_ref` y `target_module` coinciden;
- rechaza ACK corrupto o incompleto con error publico;
- rechaza ACK `completed` con `files: []` sin evidencia;
- acepta ACK `completed` con artifacts fuera de write-set como rail blando para
  revision posterior;
- acepta entregas parciales dentro del write-set para revisor/corrector; la
  completitud queda en director/review/rework/tests, no en `ACK.files`;
- acepta ampliacion de write-set justificada en `notes`, pero rechaza archivos
  de control aunque la ampliacion este justificada;
- acepta ACK con artifacts concretos que encajan en write-set con glob cerrado;
- rechaza ACK `completed` que copia globs del write-set en `files`;
- advierte en el prompt cuando hay contexto requerido truncado;
- rechaza ACK `completed` con contexto requerido truncado si no justifica
  `contexto_truncado_resuelto`;
- conserva marcadores dudosos de HOME/token/OAuth/prompt como rail pendiente
  cuando no incluyen valores sensibles efectivos;
- rechaza valores sensibles efectivos como `access_token=...`,
  `client_secret: ...`, `authorization: bearer ...` o claves privadas;
- lee `agent_ack.json` y construye `CodexDeliveryObservationV0` neutral;
- conserva refs operativas dudosas como evidencia blanda para review externa,
  sin aceptar archivos de control ni rutas inseguras.
- reconciliacion T208 cubierta por la bateria cruzada requerida; no habilita
  ejecucion Codex real por defecto ni relaja ACK/ref_only.

Riesgo residual:

- No ejecuta Codex real por defecto.
- La prueba real pertenece al mini-proyecto E2E no controlado.

## RTCODEX-T209

Comando:

```bash
go test -count=1 ./modulos/orquesta-runtime-codex
```

Cobertura:

- `codex_usage_accounting.json` se genera con contadores y cuota redactada;
- reportes con proveedor, modelo, HOME, OAuth, cuenta, coste, rutas privadas,
  prompt, transcript o completion se rechazan;
- uso sin cuota real queda como `quota.status=unknown` y
  `quota_observed_unavailable`;
- el request publico de proceso no recibe modelo, proveedor, HOME ni secretos.
- T207 no anade prueba real por defecto: la cobertura local comprueba ACK,
  shutdown checkpoint y aislamiento de runtime; la directiva de rotacion se
  valida en `orquesta-runtime` y `orquesta-orchestration-core`.
