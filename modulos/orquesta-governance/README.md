# orquesta-governance

Responsabilidad: reglas, skills, workflows y permisos.

Incluye:

- catalogo efectivo;
- versionado;
- reglas por proyecto;
- reglas por agente;
- workflows;
- permisos;
- auditoria de cambios.

Gobernanza publica reglas; no ejecuta tareas.

Para T198, `GovernanceCatalogPublicQuery v0` y sus respuestas compactas son
fuente canonica del descriptor MCP de governance. El resource debe publicar
freshness, source refs y errores publicos sin activar reglas historicas ni leer
DB v1 como autoridad viva.
La reconciliacion `agent-ref-task-autoprogramming-c3678e9bc306-g01` mantiene
ese owner y no reactiva historicos ni amplia permisos.
