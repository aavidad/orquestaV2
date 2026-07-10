# AGENTS: orquesta-document-extraction

Lee este archivo y `README.md` antes de editar este modulo.

## Reglas

- Mantener el modulo puro: contratos, IR, validacion y servicio de aplicacion.
- No importar PDF, OCR, SDKs, proveedores, red, filesystem, DB, runtime ni
  reglas de dominio. Los documentos y destinos se representan por refs opacas.
- No introducir PII ni tipos de negocio concretos. Personas y facturas solo
  declaran schemas tipados con campos definidos por refs de la app externa.
- Conservar `value_raw`, normalizacion, provenance y evidencia por separado.
- El servicio no exporta campos aceptados sin evidencia localizada.
- La politica por defecto es local. Un conector cloud requiere opt-in y refs de
  retencion, borrado, region, cifrado y auditoria.

## Verificacion minima

```bash
go test -count=1 -race ./modulos/orquesta-document-extraction ./modulos/orquesta-document-extraction-fake ./modulos/orquesta-document-extraction-json ./modulos/orquesta-document-extraction-csv
git diff --check
```
