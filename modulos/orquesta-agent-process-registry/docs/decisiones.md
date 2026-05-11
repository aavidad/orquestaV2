# Decisiones locales: orquesta-agent-process-registry

```text
Fecha: 2026-05-10
Decision: Extraer `AgentProcessRegistryPortV0` y `AgentProcessRegistryRecordV0` a un modulo neutral.
Motivo: `orquesta-persistence` no debe importar `orquestacionnucleoapp`; eso crea ciclos porque el nucleo consume modulos del director.
Alternativas: Mantener el import y excluir tests; duplicar tipos entre nucleo y persistence; mover el adaptador durable al nucleo.
Impacto: Nucleo usa alias del contrato neutral; persistence implementa el puerto sin depender del nucleo; los adaptadores futuros pueden reutilizar el contrato sin acoplarse al workflow.
Contratos afectados: AgentProcessRegistryPortV0; AgentProcessRegistryRecordV0; FileAgentProcessRegistryV0.
Estado: aceptada_local
```
