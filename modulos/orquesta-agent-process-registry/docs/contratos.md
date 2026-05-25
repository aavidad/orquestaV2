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
  - ListAgentProcessesV0(contexto, AgentProcessRegistryListFilterV0), si el
    adaptador implementa tambien `AgentProcessRegistryListPortV0`
Invariantes:
  - La clave logica es `run_id + agent_request_id`.
  - El listado devuelve solo registros compactos por refs opacas y permite
    filtrar por `run_id`; no expone PID, rutas, HOME, OAuth ni payloads.
  - Todas las referencias son opacas.
  - No serializa ni transporta rutas, PID, HOME, OAuth, provider, prompt, transcript, DB ni DSN.
  - No decide persistencia, runtime, modelo, proveedor ni workflow.
Errores:
  - agent_process_registry_invalid
  - agent_process_registry_not_found, para adaptadores que no encuentran la clave
    solicitada.
Pruebas de contrato:
  - `go test -count=1 ./modulos/orquesta-agent-process-registry`
```

```text
Nombre: AgentProcessRegistryListPortV0
Tipo: puerto_salida
Version: v0
Propietario: orquesta-agent-process-registry
Consumidores: orquestacionnucleoapp para gates de capacidad viva.
Campos:
  - ListAgentProcessesV0(contexto, AgentProcessRegistryListFilterV0)
Invariantes:
  - `run_id` es opcional; vacio significa listado global.
  - La salida conserva el mismo DTO `AgentProcessRegistryRecordV0`.
  - El puerto no consulta estado vivo: solo enumera registros. El estado vivo se
    obtiene por snapshot de runtime inyectado en otra capa.
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
