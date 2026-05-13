# Tareas: orquesta-agent-process-registry-memory

## Estado actual

- Extraido de `orquesta-orchestration-core`.
- Usado por tests y smokes como adaptador en memoria.
- Produccion debe poder sustituirlo por `orquesta-state-file` u otro conector.

## Backlog

- APRM-001: anadir snapshot compacto solo si un consumidor lo necesita.
- APRM-002: mantener parity de errores con futuros conectores durables.
