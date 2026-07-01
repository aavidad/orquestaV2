# Incidencia: QA OPES no bloquea andamiaje interno publicable

Fecha: 2026-07-02.

Relacionado con: `BUG-ORQ-20260701-093` y `BUG-ORQ-20260702-095`.

## Sintoma

Temas OPES podian superar extension y validadores oficiales de metanotas, pero
mantener bloques visibles no publicables:

- `Preguntas de recuperacion`
- `Repaso espaciado`
- `Dia 0`
- `mapa mental`
- plantillas como `La respuesta debe empezar` o
  `Una respuesta fuerte empieza`

Eso permitia falsos verdes editoriales cuando el texto crecia por anexos,
andamiaje de estudio o contaminacion cruzada.

## Causa

El contrato determinista `OPESTopicQualityContractV0` detectaba extension,
contador canonico, metacomentarios publicos y visuales didacticos, pero no
normalizaba variantes de andamiaje interno con mayusculas, tildes o encabezados
de estudio.

## Cierre

Se anade el issue `opes_public_text_study_scaffolding` al contrato de calidad
de tema OPES. La deteccion normaliza minusculas y tildes antes de buscar
patrones de andamiaje interno.

Prueba focal:

```bash
go test -count=1 ./modulos/orquesta-opes-director
```
