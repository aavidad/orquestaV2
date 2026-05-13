# Pruebas

- `TestShutdownServerV0SolicitaStopDrenaYQuedaReady`
- `TestShutdownServerV0NoDrenaSiFaltaCheckpoint`
- `TestShutdownServerV0PreparaCheckpointAntesDeStopNoForzado`
- `TestShutdownServerV0SinRunsQuedaReady`

Evidencia esperada: el caso de uso solicita stop por run, ejecuta supervisor
cuando procede, registra checkpoint por puerto en modo no forzado y calcula
`shutdown_ready` sin leer runtime ni filesystem. Cuando falta checkpoint,
propaga `pending_checkpoint_agent_refs`, `checkpoint_evidence_refs` y
`checkpoint_agents_pending` para que MCP/web/director sepan que agentes faltan.
