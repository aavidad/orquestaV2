# Contratos: orquesta-runtime-codex

## CodexExecResolverV0

Propietario: `orquesta-runtime-codex`.

Consume:

- `orquesta-runtime.ExternalAgentLaunchSpecV0`
- `orquesta-runtime.AgentStartPacketV0`

Implementa:

- `orquesta-runtime.ExternalAgentProcessCommandResolverV0`

Salida:

- `orquesta-runtime.ProcessRuntimeLaunchRequestV0`

Invariantes:

- Requiere opt-in explicito.
- El spec mantiene refs opacas; no contiene HOME, OAuth, modelo ni path real.
- El resolver materializa `agent_packet.json`, `agent_prompt.txt`, `agent_ack.json` esperado y logs bajo `runtime_work_dir`, no bajo el workdir compartido del proyecto.
- `ProcessRuntimeLaunchRequestV0` ejecuta el wrapper con `working_dir=project_work_dir` para que el proceso externo materialice codigo/documentacion en el proyecto.
- El wrapper refuerza el mismo contrato con `cd project_work_dir` y `-C project_work_dir`, pero lee prompt desde `runtime_work_dir` y redirige stdout/stderr/last-message tambien a `runtime_work_dir`.
- Si `sandbox=workspace-write`, `runtime_work_dir` debe estar fuera de
  `project_work_dir` y no puede contenerlo. El conector rechaza runtime igual,
  interno o ancestro del proyecto para evitar contaminacion entre agentes.
- En `workspace-write`, el wrapper tambien pasa `--add-dir runtime_work_dir`
  para que Codex pueda escribir ACK/logs/control sin exponer esos ficheros al
  contexto normal del proyecto.
- `ProcessRuntimeLaunchRequestV0` solo expone el wrapper y el workdir del proyecto; no expone argumentos, entorno, HOME, OAuth ni modelo.

## Contexto requerido recuperable

El resolver Codex no bloquea el lanzamiento por entradas `required=true`
truncadas o `mode=ref_only` con `materialization_missing`. Ese contexto puede
ser recuperable por lectura local, consulta al Director o evidencia en ACK, asi
que el prompt conserva la advertencia y pide resolucion explicita:
`contexto_truncado_resuelto:*` o `contexto_ref_only_resuelto:*` en `notes`
cuando el agente cierre como `completed`.

La composicion solo debe cortar fuerte por seguridad, causalidad, refs
imposibles, datos sensibles o efectos externos no autorizados. Si el contexto
no se puede resolver, el agente debe devolver bloqueo/rework causal, no quedar
sin lanzar.
- El agente externo debe escribir `agent_ack.json` en la ruta absoluta indicada en el prompt.
- Si `agent_ack.json` esta fuera de `project_work_dir`, el prompt exige
  escribirlo desde `runtime_work_dir` por shell, no mediante `apply_patch` de
  proyecto. La linea visible `ACK ...` no sustituye el JSON durable.
- El prompt tambien publica rutas de control para apagado cooperativo:
  `orquesta_shutdown_request.json` y `agent_shutdown_checkpoint_ack.json`.
  El agente debe comprobar la request antes de bloques largos de edicion o
  pruebas y responder con ACK de checkpoint si Orquesta solicita cierre.
- El prompt exige `$caveman` o `compact` cuando este disponible, salida minima y evidencia corta.
- Varios agentes Codex pueden compartir `project_work_dir` si cada uno usa un `runtime_work_dir` externo distinto y write-sets disjuntos.
- `prompt_hints` es configuracion del conector para requisitos de producto que no deben entrar en el core.
- `path_env` solo existe dentro del wrapper opt-in para runtimes que lo necesitan; no se serializa al request publico.
- `codex_stdout.log` y `codex_stderr.log` son evidencia operacional local para diagnosticar fallos de conector.
- Al terminar `codex exec`, el wrapper genera `codex_usage_accounting.json` solo
  con contadores y cuota observados de forma redactada. El reporte no contiene
  provider, modelo, cuenta, HOME, paths, prompts, transcripts, completions,
  coste ni tokens secretos; si solo hay uso y no cuota real, publica
  `quota.status=unknown`.

## Startup Lock Compartido

El wrapper serializa el arranque de Codex con
`.orquesta-codex-startup.lock` bajo `CODEX_HOME` o `HOME` cuando el perfil lo
requiere o cuando el operador declara cualquier variable de startup-lock.

Variables contractuales:

- `ORQUESTA_CODEX_STARTUP_LOCK_SECONDS`
- `ORQUESTA_CODEX_STARTUP_LOCK_TIMEOUT_SECONDS`
- `ORQUESTA_CODEX_STARTUP_LOCK_STALE_SECONDS`

Invariantes:

- La presencia de cualquiera de esas variables cuenta como configuracion
  explicita aunque el default calculado del perfil sea `0`.
- La presencia cuenta aunque el valor sea vacio; los valores vacios o invalidos
  siguen cayendo a los defaults efectivos de cada variable, pero no desactivan
  el lock por accidente.
- `ORQUESTA_CODEX_STARTUP_LOCK_TIMEOUT_SECONDS` por si sola activa el lock y
  puede cortar con `orquesta_codex_startup_lock_timeout`.
- `ORQUESTA_CODEX_STARTUP_LOCK_STALE_SECONDS` por si sola activa el lock y
  permite reaper un lock obsoleto.
- `TIMEOUT+STALE` juntos no sustituyen las garantias individuales: ambos casos
  deben pasar tambien por separado.
- La automejora remota no debe modificar `codexSharedStartupLockShellV0` ni sus
  tests si no ejecuta y conserva verde la bateria focal de startup-lock del
  wrapper.

## CodexShutdownCheckpointAckV0

Contrato local del conector para cierre cooperativo de un agente vivo.

Request escrita por Orquesta:

- `schema_version`: `codex_shutdown_request.v0`
- `run_ref`
- `agent_ref`
- `correlation_id`
- `requested_by`
- `reason`
- `checkpoint_ref`
- `evidence_refs`

ACK esperado del agente:

- `schema_version`: `codex_shutdown_checkpoint_ack.v0`
- `run_ref`
- `agent_ref`
- `shutdown_attempt_ref`
- `checkpoint_ref`
- `status`: `checkpoint_ready`
- `summary`
- `evidence_refs`

Invariantes:

- La request y el ACK viven en `runtime_work_dir`, no en el write-set de la
  app generada.
- El ACK debe correlacionar `run_ref`, `agent_ref` y `checkpoint_ref` con la
  request.
- Si el agente escribe un ACK minimo con `schema_version` y
  `status=checkpoint_ready`, el conector hidrata refs ausentes desde la request
  correlada. Refs explicitas pero incorrectas siguen bloqueando por causalidad.
- Si no existe ACK, el conector devuelve error retryable
  `checkpoint_ack_not_ready`; Orquesta no registra checkpoint.
- Un ACK corrupto, de otro agente o con detalles prohibidos no se acepta como
  cierre seguro.
- El contrato no expone HOME, OAuth, modelo, proveedor ni DB al nucleo.

## CodexAgentAckV0

ACK esperado del agente no controlado.

Campos minimos:

- `schema_version`: `orquesta_agent_ack.v0`; `codex_agent_ack.v0` queda como
  alias legacy aceptado por compatibilidad.
- `request_id`
- `correlation_id`
- `ack_ref`
- `target_module`
- `task_ref`
- `status`
- `files`
- `tests`
- `notes`

El ACK es evidencia de runtime, no sustituye `RegisterDelivery`, revision ni validacion final.
`files`, `tests` y `notes` pueden llegar como listas de strings o como objetos
estructurados; el conector los normaliza a resumen compacto para no romper la
orquestacion por diferencias menores de formato.
Los recibos de tests estructurados usan `orquesta_required_test_receipt.v0`;
`codex_required_test_receipt.v0` se acepta como alias legacy.

Receipt opt-in:

- Un ACK `completed` debe correlacionar `request_id`, `correlation_id`,
  `target_module`, `task_ref` y `ack_ref` con `ExternalAgentLaunchSpecV0` y
  `AgentStartPacketV0`.
- Un ACK `completed` debe declarar evidencia concreta, pero puede ser parcial
  dentro del `write_set` o traer ficheros fuera de alcance como rail blando si
  conserva identidad, rutas seguras y tests requeridos.
- El receipt rechaza ACK corrupto, incompleto, archivos de control, rutas
  inseguras y valores sensibles efectivos como tokens, secretos o claves. Los
  marcadores dudosos sin valor quedan como evidencia para review posterior.
- Los fallos se devuelven como `ExternalAgentConnectorErrorV0` con
  `message_key` publico y evidencia compacta; no incluyen el secreto, HOME ni
  path rechazado.

## CodexDeliveryObservationV0

DTO local del conector para convertir un ACK validado en una observacion de
entrega neutral que pueda mapearse a `AgentDeliveryObservationV0`.

Campos:

- `delivery_ref`
- `phase_id`
- `task_id`
- `agent_ref`
- `summary`
- `evidence_refs`

Invariantes:

- Se construye solo desde un ACK `completed` correlado con
  `ExternalAgentLaunchSpecV0`.
- Usa `ack_ref` como `delivery_ref` para no inventar refs fuera del protocolo.
- No incluye paths de artifacts, comandos de test, stdout/stderr, transcripts,
  HOME, OAuth, modelo, provider ni datos de DB.
- No convierte marcadores operativos genericos en rechazo duro; esos rails
  quedan para review/rework externo cuando no contienen valores sensibles.
- Este DTO sigue perteneciendo al conector; el nucleo solo recibe la version
  neutral por puerto.
