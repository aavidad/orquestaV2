# Pruebas

- `TestShutdownServerV0SolicitaStopDrenaYQuedaReady`
- `TestShutdownServerV0NoDrenaSiFaltaCheckpoint`
- `TestShutdownServerV0PreparaCheckpointAntesDeStopNoForzado`
- `TestShutdownServerV0NoPideStopSiCheckpointNoEstaListo`
- `TestShutdownServerV0SinRunsQuedaReady`

Evidencia esperada: el caso de uso registra checkpoint por puerto antes de
solicitar stop en modo no forzado, ejecuta supervisor cuando procede y calcula
`shutdown_ready` sin leer runtime ni filesystem. Cuando falta checkpoint, no
pide stop, propaga `pending_checkpoint_agent_refs`,
`checkpoint_evidence_refs` y `checkpoint_agents_pending` para que
MCP/web/director sepan que agentes faltan.
