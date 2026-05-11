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
- Si `sandbox=workspace-write`, `runtime_work_dir` debe estar dentro de
  `project_work_dir`; el conector rechaza perfiles que dejen el ACK en un
  sibling externo.
- En `workspace-write`, el wrapper tambien pasa `--add-dir runtime_work_dir`
  como defensa adicional, pero el contrato principal es que el runtime de
  control viva en un directorio oculto del proyecto.
- `ProcessRuntimeLaunchRequestV0` solo expone el wrapper y el workdir del proyecto; no expone argumentos, entorno, HOME, OAuth ni modelo.
- El agente externo debe escribir `agent_ack.json` en la ruta absoluta indicada en el prompt.
- El prompt exige `$caveman` o `compact` cuando este disponible, salida minima y evidencia corta.
- Varios agentes Codex pueden compartir `project_work_dir` si cada uno usa un `runtime_work_dir` distinto y write-sets disjuntos.
- `prompt_hints` es configuracion del conector para requisitos de producto que no deben entrar en el core.
- `path_env` solo existe dentro del wrapper opt-in para runtimes que lo necesitan; no se serializa al request publico.
- `codex_stdout.log` y `codex_stderr.log` son evidencia operacional local para diagnosticar fallos de conector.

## CodexAgentAckV0

ACK esperado del agente no controlado.

Campos minimos:

- `schema_version`: `codex_agent_ack.v0`
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

Receipt opt-in:

- Un ACK `completed` debe correlacionar `request_id`, `correlation_id`,
  `target_module`, `task_ref` y `ack_ref` con `ExternalAgentLaunchSpecV0` y
  `AgentStartPacketV0`.
- Un ACK `completed` debe declarar todos los artifacts exigidos por `write_set`
  y no puede declarar artifacts fuera de ese write-set.
- El receipt rechaza ACK corrupto, incompleto o con HOME real, token/secreto,
  OAuth, prompt/completion o transcript completo.
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
- Rechaza la observacion si cualquier ref compacta filtra detalles que el core
  prohibe, incluido `codex`, `provider`, `runtime`, `home`, `oauth`, `sql`,
  `docker`, `git`, `tmux`, secretos o tokens.
- Este DTO sigue perteneciendo al conector; el nucleo solo recibe la version
  neutral por puerto.
