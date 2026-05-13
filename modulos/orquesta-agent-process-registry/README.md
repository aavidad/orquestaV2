# orquesta-agent-process-registry

Contrato neutral del registro de procesos de agentes.

Responsabilidad:

- definir `AgentProcessRegistryPortV0`;
- definir `AgentProcessRegistryRecordV0`;
- definir codigos publicos `agent_process_registry_invalid` y
  `agent_process_registry_not_found`;
- validar refs opacas de `run_id`, `agent_request_id`, `process_ref`, `session_ref`, `launch_ref`, `readiness_ref` y evidencias.

No contiene adaptadores, runtime, DB, filesystem, proveedor ni reglas del workflow.
