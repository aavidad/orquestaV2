# Contratos: orquesta-agent-process-registry-memory

`InMemoryAgentProcessRegistryV0` implementa
`orquesta-agent-process-registry.AgentProcessRegistryPortV0`.

Entradas:

- `RecordAgentProcessV0(ctx, record)`;
- `ResolveAgentProcessV0(ctx, run_id, agent_request_id)`.

Invariantes:

- normaliza y valida refs con el contrato puro;
- repetir el mismo registro es idempotente;
- registrar el mismo `run_id + agent_request_id` con otra identidad falla;
- missing record devuelve `agent_process_registry_not_found`;
- no importa core, runtime, DB, filesystem ni proveedores.
