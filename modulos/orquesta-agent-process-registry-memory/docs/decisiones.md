# Decisiones: orquesta-agent-process-registry-memory

## 2026-05-13: el adaptador de memoria sale del core

Decision: `InMemoryAgentProcessRegistryV0` vive en este modulo, no en
`orquesta-orchestration-core`.

Motivo: el nucleo debe depender de puertos, no de implementaciones de memoria.
El registro de procesos puede ser memoria, fichero, DB, servicio remoto u otro
conector.

Consecuencia: los tests del core usan un alias solo en `_test.go`; el paquete
productivo del core no importa este adaptador.
