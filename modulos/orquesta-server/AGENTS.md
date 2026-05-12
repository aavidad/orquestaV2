# Contexto local: orquesta-server

Lee este fichero antes de tocar el modulo.

## Responsabilidad

Servidor residente de Orquesta. Mantiene HTTP, supervisor global y statefile de
reenganche para que una sesion de Codex pueda cerrarse sin cortar Orquesta.

## Reglas

- Hexagonal: este modulo no importa stacks concretos, Codex, DB, web ni MCP.
- No elige runtime, proveedor, credenciales, HOME ni base de datos.
- El supervisor entra por puerto.
- El almacenamiento de statefile entra por puerto.
- No crear ficheros grandes; partir por responsabilidad.
- No esconder errores con paths sensibles.

