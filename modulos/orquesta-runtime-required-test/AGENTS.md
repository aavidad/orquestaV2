# Contexto para agentes: orquesta-runtime-required-test

Adaptador externo para ejecutar tests requeridos del Director Operativo.

Reglas:

- Este modulo implementa puertos del nucleo, pero no pertenece al nucleo.
- Puede usar filesystem y procesos locales solo por configuracion explicita.
- No heredar entorno del proceso padre.
- No ejecutar shell; solo binarios permitidos por allowlist inyectada.
- No devolver rutas absolutas al nucleo; solo refs relativas/opacas de evidencia.
- No conocer Codex, OPES, HOME, OAuth, tokens, modelos ni DB concreta.

Validacion local:

```bash
go test -count=1 ./modulos/orquesta-runtime-required-test
git diff --check -- modulos/orquesta-runtime-required-test
```
