# Incidencia: external-work podia presentar refs de rutas como si fueran paths

Fecha: 2026-07-02

## Sintoma

En OPES se observaron rutas locales deformadas por normalizacion de refs. Un
agente podia intentar usarlas como rutas reales de filesystem y fallar lecturas
que habrian funcionado con el valor original.

## Cierre aplicado

- Los `input_fields` con rutas absolutas locales se siguen compactando para no
  exponer paths crudos.
- Cuando el campo permite path operacional, el resumen inlineado usa objeto
  estructurado:
  - `kind: local_path_ref`
  - `ref: local-path-ref-...`
  - `basename_hint`
  - `executable_path: false`
  - `resolution: opaque_input_field_payload_ref_only`
- El criterio de aceptacion avisa que esas refs son opacas y no deben tratarse
  como paths ejecutables.

## Evidencia

```bash
go test -count=1 ./modulos/orquesta-external-work-run
```

## Residual

Las rutas reales permanecen en los contratos durables de AppChange/DomainWork.
Si un agente necesita resolverlas, debe hacerlo por el payload estructurado o
por un conector autorizado, no interpretando `context_refs.ref`.
