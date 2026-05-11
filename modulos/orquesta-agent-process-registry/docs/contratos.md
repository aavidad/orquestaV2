# Contratos locales: orquesta-agent-process-registry

```text
Nombre: AgentProcessRegistryPortV0
Tipo: puerto_salida
Version: v0
Propietario: orquesta-agent-process-registry
Consumidores: orquestacionnucleoapp; orquesta-persistence; conectores futuros de runtime/persistencia.
Campos:
  - RecordAgentProcessV0(contexto, AgentProcessRegistryRecordV0)
  - ResolveAgentProcessV0(contexto, run_id, agent_request_id)
Invariantes:
  - La clave logica es `run_id + agent_request_id`.
  - Todas las referencias son opacas.
  - No serializa ni transporta rutas, PID, HOME, OAuth, provider, prompt, transcript, DB ni DSN.
  - No decide persistencia, runtime, modelo, proveedor ni workflow.
Errores:
  - agent_process_registry_invalid
Pruebas de contrato:
  - `go test -count=1 ./modulos/orquesta-agent-process-registry`
```

```text
Nombre: AgentProcessRegistryRecordV0
Tipo: dto
Version: v0
Propietario: orquesta-agent-process-registry
Consumidores: nucleo y adaptadores de persistencia.
Campos:
  - run_id
  - agent_request_id
  - process_ref
  - session_ref
  - launch_ref
  - readiness_ref
  - evidence_refs
Invariantes:
  - Todos los campos requeridos salvo `evidence_refs`.
  - `evidence_refs` se compacta eliminando vacios y duplicados.
  - Solo refs opacas; los detalles operativos quedan en adaptadores privados.
Errores:
  - agent_process_registry_invalid
Pruebas de contrato:
  - Validacion de refs opacas, rechazo de detalle operativo y normalizacion.
```
