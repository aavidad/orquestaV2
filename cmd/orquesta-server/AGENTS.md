# Contexto Codex: cmd/orquesta-server

Lee primero `AGENTS.md` raiz y este archivo. Despues lee solo los documentos y
README locales necesarios para la microtarea.

## Responsabilidad

Es la composition root del servidor Orquesta. Cablea configuracion, HTTP, MCP,
runtime, stores file-based, bridges opt-in y stacks de dominio sin convertirlos
en nucleo.

## Capa

Composicion ejecutable. Puede depender de adaptadores concretos, pero debe
mantenerlos fuera de core, workflow, domain-work y contratos neutrales.

## Imports prohibidos

- No reintroducir `cmd/db/internal`, `ensureLocalDB` ni control-plane legacy.
- No mover Codex, OPES, HTTP, MCP, DB concreta, HOME, OAuth, tokens, paths
  locales, proveedor ni runtime real hacia modulos de nucleo.
- No compartir DB/filesystem interno de apps externas; usa puertos, refs opacas
  y conectores opt-in.

## Pruebas focales

Usa segun cambio:

```bash
go test -count=1 ./cmd/orquesta-server
go test -count=1 ./modulos/orquesta-rails ./modulos/orquesta-domain-work-http ./modulos/orquesta-factory-http ./modulos/orquesta-context ./cmd/orquesta-server
```

Para cambios transversales ejecuta tambien `git diff --check` y la bateria que
indique el paquete OrquestaV2.

## Docs vigentes

Fuentes canonicas: `AGENTS.md`, `docs/estado_actual_2026-05-17.md`,
`docs/guia_nucleo_orquestacion_2026-05-17.md`,
`docs/matriz_pruebas_reales_y_smoke_2026-05-17.md` y los cortes especificos de
Director/OPES cuando el wiring los toque.

Mantener configuracion canonica en la superficie unica del servidor; no dupliques
env vars con nombres distintos repartidos por el codigo.
