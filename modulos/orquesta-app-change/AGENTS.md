# AGENTS: orquesta-app-change

Lee este contexto antes de editar este modulo.

- Mantener arquitectura hexagonal: este modulo solo define caso de uso y
  puertos.
- No importar `cmd`, `db`, runtime real, proveedor, modelo ni filesystem
  concreto.
- i18n queda en los bordes web/MCP; enums y DTOs internos no se localizan.
- Las solicitudes de cambio son intenciones versionadas sobre un run existente,
  no `AppSpecRequestV0` reutilizado.
- Ficheros pequenos y pruebas acotadas por contrato.
