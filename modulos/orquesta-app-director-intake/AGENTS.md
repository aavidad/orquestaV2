# Contexto local: orquesta-app-director-intake

Lee primero este fichero y los documentos de `docs/`.

Reglas locales:

- preparar entrada de app hacia director, no ejecutar la app completa;
- arquitectura hexagonal: REST, MCP y web son adaptadores externos;
- no elegir proveedor, cuenta, credencial, runtime, HOME ni base de datos;
- no importar `cmd` ni `db`;
- ficheros pequenos y pruebas focales antes de ampliar.
