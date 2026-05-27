# Contexto Codex: orquesta-rails

Lee primero `AGENTS.md` raiz y este archivo. Usa `README.md` solo como resumen
local y despues lee los documentos vigentes que pida la tarea.

## Responsabilidad

Define railes reutilizables, pequenos y puros para fronteras de Orquesta:
texto sensible, redaccion, filesystem y politicas de campo. Este modulo no
decide producto, runtime, proveedor, dominio ni composicion.

## Capa

Contrato/politica neutral reutilizable. Puede ser consumido por core,
adaptadores o composiciones, pero no debe importar desde ellos para evitar
ciclos y reglas acopladas a producto.

## Imports prohibidos

- Codex, OPES, web, MCP, CLI, HTTP server, runtime real o proveedor concreto.
- DB concreta, filesystem productivo, HOME, OAuth, tokens, secretos o rutas
  locales.
- Paquetes de composicion como `cmd/orquesta-server` o adaptadores de dominio.

## Pruebas focales

Usa:

```bash
go test -count=1 ./modulos/orquesta-rails
```

Si cambias falsos positivos o redaccion de rails, anade matrices con ejemplos
externos y casos sensibles efectivos.

## Docs vigentes

Fuentes canonicas: `AGENTS.md`, `docs/estado_actual_2026-05-17.md`,
`docs/guia_nucleo_orquestacion_2026-05-17.md` y
`docs/rail_errors_observados_2026-05-23.md`.

No conviertas una apertura temporal de rail en politica global sin evidencia y
sin actualizar el backlog vivo.
