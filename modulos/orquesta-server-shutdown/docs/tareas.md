# Tareas

- SSH-001: Definir DTOs compactos de shutdown.
- SSH-002: Implementar caso de uso sobre puertos existentes.
- SSH-003: Cubrir stop forzado, checkpoint pendiente y cola vacia.
- SSH-004: Cablear REST/MCP por adaptador fino.
- SSH-005: Hecho. Preparar checkpoint no forzado por puerto y registrar ACK
  durable antes de `StopRunV0`; cubierto por
  `TestShutdownServerV0PreparaCheckpointAntesDeStopNoForzado` y
  `TestShutdownServerV0NoPideStopSiCheckpointNoEstaListo`.
