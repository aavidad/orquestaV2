# Incidencia: QA OPES no distinguia contaminacion estructural de tema

Fecha: 2026-07-02.

## Sintoma

Durante el cierre de Grupo B Informatica, varios temas parecian pasar por
extension y limpieza superficial, pero la extension dependia de bloques ajenos,
referencias internas a canones/derivaciones, rutas de trabajo (`tema_XXX`,
`02_temas/`, `.md`) o visuales internos visibles (`.svg`). Eso no es solo
andamiaje de estudio: el cuerpo no pertenece de forma fiable al titulo del tema.

## Causa

`OPESTopicQualityContractV0` ya detectaba metacomentarios y andamiaje interno,
pero no emitia una señal separada para contaminacion estructural. El registro de
temas podia dejar esos casos como QA textual generica pendiente o progreso
normal, sin forzar rework editorial estructural.

## Cierre

El contrato OPES emite `opes_public_text_structural_contamination` ante señales
publicables de bloques ajenos, canon/maestro, derivaciones internas, refs a
artefactos de trabajo y rutas internas. El registro de temas proyecta esa señal
como `pendiente_rework_editorial`, con `pending_refs` y followup de
`review_director_consolidation`.

Este cierre no sustituye una revision semantica completa titulo-cuerpo con IA o
RAG: cubre el falso verde determinista de contaminacion visible en texto
publicable y deja una causa causal para replanificar expansion propia.

Pruebas:

```bash
go test -count=1 ./modulos/orquesta-opes-director -run 'TestValidateOPESTopicQualityContractV0DetectaContaminacionEstructural|TestProduceOPESCausalJobsV0BloqueaRegistroPorContaminacionEstructural'
go test -count=1 ./modulos/orquesta-opes-director
```
