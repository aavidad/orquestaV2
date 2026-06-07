# Contexto local: orquesta-runtime-claude

Adaptador CLI opt-in para Claude. Este modulo solo traduce un
`ExternalAgentLaunchSpecV0` neutral a ficheros de control y proceso local.

Reglas:

- No metas Claude en core, workflow, domain-work ni Director Operativo.
- No metas reglas OPES aqui. El dominio externo llega por el packet/contexto.
- El ACK durable sigue el recibo compatible `codex_agent_ack.v0` para reutilizar
  observacion/review sin bifurcar el nucleo.
- No proyectes tokens ni HOME reales en specs, prompts, issues o evidencia.
- Los prompts deben orientar, no cortar por strings exactos salvo seguridad,
  causalidad, refs imposibles o efectos externos no autorizados.
