# Tareas locales: orquesta-agent-process-registry

```text
ID: APR-000
Objetivo: Extraer contrato neutral del registro de procesos para romper el ciclo nucleo -> director -> persistence -> nucleo.
Write-set: AGENTS.md; README.md; docs/*; types_v0.go; validation_v0.go; validation_v0_test.go
Simbolo foco: AgentProcessRegistryPortV0
Contrato: AgentProcessRegistryPortV0; AgentProcessRegistryRecordV0
Validacion: `go test -count=1 ./modulos/orquesta-agent-process-registry`; `go test -count=1 ./modulos/...`
Bloqueos: Ninguno. No incluye adaptador durable ni runtime.
Estado: completada
```
