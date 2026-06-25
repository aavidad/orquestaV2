# Contexto local: orquesta-runtime-codex-goal

Lee este archivo y `README.md` antes de editar.

## Responsabilidad

Adaptador opt-in para ejecutar `GoalWorkSpecV0` con Codex Goal. Codex Goal actua
como Director operativo dentro del trabajo; Orquesta conserva contrato,
evidencias y cierre.

## Reglas

- Este modulo es adaptador externo, no nucleo.
- No importar `cmd`, DB, state-file, web, MCP, OPES ni filesystem productivo.
- No hardcodear HOME, proveedor, modelo, OAuth, token, comando ni path local.
- El arranque real de Codex Goal entra por puerto inyectado.
- No convertir `complete` de Codex Goal en cierre aceptado sin validador de
  Orquesta.

## Validacion

```bash
go test -count=1 ./modulos/orquesta-runtime-codex-goal
```
