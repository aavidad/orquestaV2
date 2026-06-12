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
- `EgressSanitizerConfigV0` es la entrada canonica de composicion para activar
  el adaptador y declarar `openai_privacy_filter_local` como modelo local
  opt-in.
- Sustituye claves, tokens, secretos, rutas privadas, URLs privadas, URLs con
  credenciales/query sensible y material no publicable por refs
  `sanitized-ref-*`.
- Conserva URLs publicas no sensibles como URLs. Un enlace publico sin token,
  credencial, host interno, path local ni query sensible es evidencia/contexto,
  no un dato a redactar por defecto.
- Si detecta material ambiguo como claves privadas o transcripts completos,
  degrada la entrada a `ref_only` y exige revision por director/humano.
- No bloquea por palabras operativas blandas como provider, model, runtime o
  capacity; esas quedan como diagnostico si no contienen valor sensible.

Regla para apps IA generadas por Orquesta:

- Toda app IA generada o modificada por Orquesta debe declarar en su
  composicion si activa este sanitizer local, un Privacy Filter local
  equivalente o un puerto de saneamiento sustituible.
- El core de Orquesta no recibe modelo, proveedor, endpoint, comando, HOME,
  token ni transporte del filtro; solo ve el puerto neutral, refs opacas y
  evidencia compacta.
- Si la app aun no tiene egress real, la ausencia de filtro se documenta como
  pendiente operativo antes de entregar o publicar la app.

Validacion:

```bash
go test -count=1 ./modulos/orquesta-context
go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestLocalSensitiveDataSanitizerV0|Test.*Sanitizer|Test.*Egress'
```
