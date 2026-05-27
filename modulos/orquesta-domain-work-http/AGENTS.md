# Contexto Codex: orquesta-domain-work-http

Lee primero `AGENTS.md` raiz y este archivo. Usa `README.md` para la politica
de egress HTTP antes de editar.

## Responsabilidad

Implementa el adaptador HTTP neutral de `orquesta-domain-work` para crear jobs y
enviar artefactos a una app externa por contrato. Transporta refs opacas; no
posee reglas de OPES, Codex, runtime ni persistencia de la app.

## Capa

Adaptador opt-in de composicion sobre contratos neutrales. El dominio externo
valida y persiste sus datos; Orquesta solo llama puertos HTTP configurados por
la composicion.

## Imports prohibidos

- OPES, Codex, runtime real, proveedor, MCP, web UI o DB concreta.
- HOME, OAuth, tokens, credenciales, rutas locales o filesystem interno de apps
  externas.
- Internals privados de `orquesta-domain-work`; usa solo contratos publicos.

## Pruebas focales

Usa:

```bash
go test -count=1 ./modulos/orquesta-domain-work-http
```

Si cambias egress, base URL, redaccion de errores o response policy, cubre
loopback, allowlist, rechazos de credenciales/query/fragment y diagnosticos
publicos sin cuerpo crudo.

## Docs vigentes

Fuentes canonicas: `AGENTS.md`, `docs/estado_actual_2026-05-17.md`,
`docs/guia_nucleo_orquestacion_2026-05-17.md`,
`docs/matriz_pruebas_reales_y_smoke_2026-05-17.md` y este `README.md`.

No promuevas este adaptador a persistencia global ni a conector de producto.
