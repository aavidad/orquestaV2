# Incidencia OPES: Cierre De Temario Sin Mínimos Ni Comunes Maestros

Fecha: 2026-06-22.

## Contexto

Curso probado desde OPES:

`operario-tractorista-grupo-5`

Orquesta produjo y aceptó artefactos de texto, tests, HTML local y RAG
provisional. La revisión humana posterior detectó que el temario era
editorialmente demasiado pequeño: los ampliados rondaban 1.300-2.100 palabras
por tema y los comunes no demostraban derivación desde maestros comunes
A1/A1-A2.

## Síntoma

Orquesta convirtió un `domain_work` OPES aceptado por receipt en evidencia
suficiente de cierre operativo. El receipt demostraba entrega causal y aceptación
por el conector, pero no demostraba:

- informe de extensión por nivel;
- mínimos de palabras por tema;
- matriz de derivación/reutilización de comunes maestros;
- decisión explícita de que comunes no aplicaban.

## Impacto

Autonomía OPES podía cerrar una fase como si estuviera bien porque existían
ficheros y ACK/receipt, aunque el resultado no alcanzara calidad editorial.

## Arreglo Programado

- OPES define mínimos por nivel y validador local:
  `validate_extension_temario_opes.py`.
- El bridge OPES añade required tests para paquetes finales:
  `opes-extension-minima-nivel-*` y
  `opes-derivacion-comunes-maestro-*`.
- El cierre operativo de `orquesta-app-codex-stack` bloquea paquetes finales
  OPES si el receipt aceptado no trae evidencia de extensión y de comunes/no
  aplicabilidad.
- La causalidad de `domain_work` usa `expected_artifact_type` cuando OPES lo
  declara, no solo el tipo genérico por `work_kind`.

## Criterio De Cierre

Un paquete final OPES solo puede cerrar si el receipt aceptado incluye evidencia
equivalente a:

- `opes-extension-minima-passed` o informe de extensión;
- `opes-common-canonical-reuse-passed`, matriz de comunes o
  `opes-common-master-not-applicable`.

Si falta, Orquesta no construye `OperationalDirectorClosureRequestV0` para ese
delivery.
