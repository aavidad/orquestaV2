# Incidencia: variantes de andamiaje de estudio en texto publicable OPES

Fecha: 2026-07-02.

Relacionado con: `BUG-ORQ-20260702-095`.

## Sintoma

El QA de texto publicable podia no bloquear variantes de andamiaje interno con
mayusculas, tildes compuestas o descompuestas, separadores Markdown/puntuacion
o saltos de linea:

- `Preguntas de recuperacion`
- `Repaso espaciado`
- `Dia 0` / `Día 0`
- `mapa mental`
- `La respuesta debe empezar`
- `Una respuesta fuerte empieza`

Estos bloques pertenecen a estudio, tutor o rework, no al Markdown publicable
del temario ampliado/resumen.

## Cierre Orquesta

`OPESTopicQualityContractV0` normaliza las frases publicables antes de comparar:
minusculas, tildes compuestas, marcas Unicode descompuestas y separadores no
alfanumericos pasan a una forma canonica de tokens. La deteccion queda dentro
del validador deterministico OPES y se proyecta como
`opes_public_text_study_scaffolding`.

No se ha tocado OPES productivo ni se ha anadido un rail generico fuera del
contrato de dominio.

## Prueba focal

```bash
go test -count=1 ./modulos/orquesta-opes-director
```
