# orquesta-observability

Responsabilidad: observar Orquesta sin decidir negocio.

Incluye:

- eventos normalizados;
- runtime tree;
- traces;
- artifacts;
- deltas;
- replay parcial;
- auditoria operativa.

La observabilidad debe producir contexto compacto para humanos e IA.

Para T198, `OperationalStatusQueryV0` y `WorkspaceTimelineQueryV0` son fuentes
canonicas de descriptors MCP de observabilidad. El transporte debe publicar
owner/freshness/DTO/validador y no usar observabilidad como store paralelo ni
volcar transcripts, prompts, completions o payloads completos.
La reconciliacion `agent-ref-task-autoprogramming-c3678e9bc306-g01` conserva
estas fuentes y no abre sink, DB, runtime ni lectura de transcripts.
