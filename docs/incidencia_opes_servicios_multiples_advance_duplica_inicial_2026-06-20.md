# Incidencia OPES Servicios Múltiples: advance vuelve a trabajo inicial

Fecha: 2026-06-20.

## Contexto

En el curso OPES `oficial-servicios-multiples-c2-20260620` ya existían:

- `plan_temario` completado;
- 17 `content_block` aceptados;
- 17 `visual_asset` aceptados;
- duplicados obsoletos de temas 10 y 16 cancelados por API.

Tras llamar a:

```bash
POST /api/programs/f0f9cb912e353b9c7f3459e9e76a949a/advance
```

con `correlation_id=oficial-servicios-multiples-c2-20260620`, OPES/Orquesta
respondió con `action=initial_work` y creó 10 jobs nuevos de:

- `research_sources`;
- `split_syllabus_topic`;
- `draft_topic_outline`.

Todos correspondían a temas 1-4, ya cubiertos por el plan y los bloques
aceptados.

## Impacto

El flujo automático vuelve al inicio aunque el curso ya tiene artefactos de
contenido y visuales válidos. Si se dejara correr, duplicaría inventario,
outline y fuentes, consumiría cuota y podría generar versiones paralelas del
mismo temario.

## Acción manual aplicada

El director OPES canceló por API esos 10 jobs con motivo:

`avance automatico initial_work duplico temas ya cubiertos por plan_temario/content_block/visual_asset aceptados`.

No se lanzaron agentes para esos duplicados.

## Arreglo esperado

El Director OPES de avance debe consultar estado por `correlation_id`,
`program_id` dentro del payload y artefactos aceptados, no solo por tablas
canónicas/materialización parcial.

Antes de crear `initial_work`, debe detectar:

- `document_plan` aceptado;
- cobertura de `content_block` por tema oficial;
- cobertura de `visual_asset` por tema oficial;
- jobs cancelados por sustitución con rework aceptado.

Si esas piezas existen, debe avanzar a la siguiente fase pendiente real:
tests, tutor/RAG, HTML, revisiones, ensamblado, audio tras cierre, manuales y
paquete final.

## Criterio de cierre

Smoke con curso OPES basado en `document_plan` y artefactos no materializados
en tablas canónicas:

- `advance` no crea `research_sources`, `split_syllabus_topic` ni
  `draft_topic_outline` para temas ya cubiertos;
- crea o despierta solo fases siguientes;
- deja informe compacto con `skipped_initial_work_already_covered`.
