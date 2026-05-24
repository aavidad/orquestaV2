# Frontera local de saneamiento de contexto sensible

Objetivo: revisar contexto saliente antes de construir packets para agentes
premium/remotos sin mover IA, proveedor, modelo ni transporte al nucleo.

Contrato:

- `orquesta-context` define `ContextSanitizerPortV0`.
- La composicion inyecta un sanitizador opt-in.
- La salida conserva `ContextSanitizationEvidenceV0` con refs opacas,
  categorias y contador de sustituciones.
- El dato sensible original no se persiste en el bundle ni en el packet.

Adaptador actual:

- `LocalSensitiveDataSanitizerV0` vive en `orquesta-app-codex-stack`.
- Sustituye claves, tokens, secretos, rutas privadas, URLs y material no
  publicable por refs `sanitized-ref-*`.
- Si detecta material ambiguo como claves privadas o transcripts completos,
  degrada la entrada a `ref_only` y exige revision por director/humano.

Validacion:

```bash
go test -count=1 ./modulos/orquesta-context ./modulos/orquesta-runtime ./modulos/orquesta-app-codex-stack
```
