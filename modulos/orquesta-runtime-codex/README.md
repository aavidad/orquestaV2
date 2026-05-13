# orquesta-runtime-codex

Conector externo opt-in para lanzar Codex CLI desde OrquestaV2 sin acoplar el nucleo a Codex.

Flujo:

```text
core/outbox -> RuntimeLaunchRequestV0 -> AgentStartPacketV0
  -> ExternalAgentLaunchSpecV0 -> CodexExecResolverV0
  -> ProcessRuntimeLaunchRequestV0 -> ProcessRuntimeConnectorV0
```

Este modulo no decide fases, capacidad ni tareas. Solo traduce una orden opaca ya validada a un proceso real configurado por el operador.

Para ejecucion paralela, cada agente debe recibir un `runtime_work_dir` distinto.
Los ficheros de control (`agent_packet.json`, `agent_prompt.txt`, `agent_ack.json`,
`orquesta_shutdown_request.json`, `agent_shutdown_checkpoint_ack.json` y logs)
viven ahi. El `project_work_dir` compartido queda solo para el codigo y
documentacion autorizados por write-set. El proceso externo arranca con
`working_dir=project_work_dir`; el wrapper tambien hace `cd project_work_dir` y
usa `-C project_work_dir` para evitar que un runtime aislado capture la salida
del agente.

Validacion local:

```bash
go test -count=1 ./modulos/orquesta-runtime-codex
git diff --check -- modulos/orquesta-runtime-codex
```

La ejecucion real de Codex es opt-in y queda fuera de los tests unitarios por defecto.

El ACK generado por el agente se valida como receipt local del conector: debe
correlacionar `request_id`, `correlation_id` y `ack_ref`, cerrar `files` al
write-set y no incluir HOME real, secretos, OAuth ni transcripts completos.
Tras validar el ACK, el conector puede construir `CodexDeliveryObservationV0`,
un DTO neutral para alimentar el puerto de entregas del nucleo sin pasar paths,
logs, stdout/stderr, modelo, proveedor, HOME ni DB.

Para apagado no forzado, Orquesta escribe una request de shutdown cooperativo
en el runtime del agente. El checkpoint solo se considera durable si el agente
responde con `codex_shutdown_checkpoint_ack.v0` correlado; si no, el cierre
queda pendiente o debe pasar a modo forzado por decision externa.
