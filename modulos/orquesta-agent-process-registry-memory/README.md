# orquesta-agent-process-registry-memory

Adaptador en memoria para `AgentProcessRegistryPortV0`.

Responsabilidad:

- registrar `run_id + agent_request_id -> process_ref`;
- resolver el proceso asociado a un agente;
- validar refs usando el contrato puro `orquesta-agent-process-registry`;
- detectar replay idempotente, conflicto y missing record.

Fuera de alcance:

- DB, filesystem, runtime, red, procesos reales, proveedor, HOME, OAuth o
  credenciales;
- reglas del workflow o del nucleo de orquestacion.

Validacion:

```sh
go test -count=1 ./modulos/orquesta-agent-process-registry-memory
```
