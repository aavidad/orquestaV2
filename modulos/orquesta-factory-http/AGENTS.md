# Contexto Codex: orquesta-factory-http

Lee primero `AGENTS.md` raiz y este archivo. Usa `README.md` como resumen local
del endpoint antes de editar.

## Responsabilidad

Expone la frontera HTTP de `orquesta-factory` para `POST /api/v0/apps/spec`.
Debe parsear y proyectar la entrada publica sin meter logica de negocio,
runtime, proveedor ni persistencia en el adaptador.

## Capa

Adaptador HTTP de composicion sobre el puerto factory. La factoria conserva el
contrato de app; este modulo solo traduce request/response y errores publicos.

## Imports prohibidos

- Codex, OPES, runtime real, proveedor, DB concreta, MCP, web UI o filesystem
  productivo.
- HOME, OAuth, tokens, secretos, rutas locales, prompts o transcripts.
- Internals privados de otros modulos; usa puertos y DTOs publicos.

## Pruebas focales

Usa:

```bash
go test -count=1 ./modulos/orquesta-factory-http
```

Si cambias parsing JSON o errores HTTP, prueba payload valido, JSON invalido,
limites de entrada y que los errores publicos no filtren material crudo.

## Docs vigentes

Fuentes canonicas: `AGENTS.md`, `docs/estado_actual_2026-05-17.md`,
`docs/guia_nucleo_orquestacion_2026-05-17.md` y este `README.md`.

No conviertas la preview factory en planificador inteligente: las apps nuevas
deben entrar por Director cuando requieran juicio.
