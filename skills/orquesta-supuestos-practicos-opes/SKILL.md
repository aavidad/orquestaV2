---
name: orquesta-supuestos-practicos-opes
description: Coordinar con Orquesta la creación, reutilización, revisión y cierre de supuestos prácticos OPES, incluidos supuestos tipo test, por tema o curso. Usar cuando un trabajo OPES pida casos prácticos, exámenes prácticos, supuestos clínicos/sanitarios, supuestos administrativos o bancos importables basados en temarios terminados.
---

# Orquesta Supuestos Prácticos OPES

Usar esta skill para convertir una petición de supuestos prácticos OPES en
trabajo dirigido por agentes. La especificación editorial vive en el repositorio
OPES; Orquesta solo coordina tareas, refs, agentes, revisiones, evidencias y
cierre.

Leer también la skill de dominio
`/home/alberto/Trabajo/OPES/skills/opes-supuestos-practicos/SKILL.md` cuando se
vaya a lanzar o revisar el trabajo.

## Principio

No crear supuestos a mano como sustituto de Orquesta. El Director debe:

- localizar curso, programa y temas terminados;
- inventariar exámenes, cuadernillos, bancos y supuestos previos;
- abrir tareas por tema o por lote controlado;
- pedir entregas compactas con refs, no corpus pegado;
- integrar solo artefactos validados por OPES;
- conservar lo reutilizable y mandar a rework lo que falle.

No meter reglas TCAE, sanitarias, jurídicas ni de la web USO en el núcleo de
Orquesta. Esas reglas van en OPES o en adaptadores de dominio.

## Topología

Para un banco grande de supuestos:

- usar un padre por tema específico si el usuario exige cobertura tema a tema;
- cada padre cubre seis subroles: fuentes/formato oficial, cobertura de
  temario, diseño del caso, preguntas tipo test, tutor/rúbrica y validación
  importable;
- si el trabajo es inventario o auditoría, se pueden usar lotes de hasta seis
  temas;
- si un tema tiene alto riesgo sanitario, normativo o de competencia profesional,
  aislarlo en un padre propio;
- no poner límites globales artificiales salvo límite duro de runtime, cuota,
  proveedor o instrucción expresa.

## Encargo Mínimo Al Padre

Cada tarea debe recibir:

- `course_id` y variante del curso;
- lista de temas asignados;
- refs del temario final aprobado;
- refs de cuadernillos/exámenes usados como modelo de formato;
- refs de bancos o supuestos previos reutilizables;
- skill de dominio OPES aplicable y, si procede, skill de tests de 4 respuestas;
- write-set acotado: carpeta de supuestos, importables, HTML de revisión e
  informes;
- mínimo exigido por tema;
- criterio de validación contra temario;
- formato de salida breve.

Pedir modo compacto:

```text
Usa salida breve. Lee solo refs necesarias con rg/sed/find. No pegues corpus.
No preguntes fuera del temario terminado. Entrega rutas, conteos, pruebas,
bloqueos y rework.
```

## Artefactos Esperados

El trabajo debe producir o actualizar:

- matriz de procedencia y reutilización;
- supuestos en Markdown para revisión humana;
- JSON/JSONL importable por la web o por el adaptador OPES;
- HTML local de revisión con la carcasa visual vigente del curso cuando el
  consumidor la tenga definida;
- banco de test separado e importable cuando haya supuestos `tipo_test`;
- informe de alcance contra temario;
- informe de validación estructural;
- incidencias de rework por tema.

Campos mínimos por supuesto:

```text
case_id
course_id
course_variant
topic_id
topic_title
case_type: desarrollo | tipo_test | mixto
source_style_refs
stem
clinical_or_operational_data
questions
model_answer_or_correct_options
rubric_or_explanations
tutor_help
topic_evidence_refs
scope_status
editorial_decision
```

Para JSON/JSONL importable, no aceptar alias libres si no existe normalizador
del consumidor. En particular:

- el array evaluable se llama `questions`, no `tasks`;
- `questions[].kind` debe ser `multiple_choice` u `open_response`;
- en tipo test, cada pregunta tiene opciones `A`, `B`, `C`, `D`, una correcta y
  explicación;
- en desarrollo, las preguntas abiertas usan `open_response`;
- cada pregunta declara evidencia del temario propio.

El cierre de supuestos no se considera completo si solo existen Markdown o
JSONL. Debe existir HTML revisable y, para `case_type: tipo_test`, un banco de
test derivado con una fila por pregunta, conservando `case_id`, escenario,
opciones `A-D`, correcta, explicación, evidencia y origen
`supuesto_tipo_test`.

## Validación

Antes de aceptar una entrega, el Director debe comprobar:

- cada supuesto declara tema, apartado/ancla o evidencia textual del temario;
- ningún supuesto pregunta contenido fuera del temario terminado;
- los supuestos tipo test tienen cuatro opciones, una correcta y distractores
  plausibles;
- no hay preguntas metaacadémicas ni plantillas repetidas;
- el idioma visible conserva tildes, eñes y signos completos;
- en sanidad, el caso respeta el rol profesional evaluado y no atribuye
  competencias ajenas;
- el origen de formato oficial queda citado como referencia de estilo o como
  reutilización declarada;
- el importable pasa validación estructural.

Si algo falla, no tirar todo el lote: marcar `needs_rework` solo en el supuesto,
pregunta, tema o evidencia afectada.

## Cierre

Cerrar solo cuando existan:

- conteo de supuestos por tema frente al mínimo pedido;
- matriz de reutilización y fuentes;
- banco importable;
- HTML revisable;
- banco de test importable si existen supuestos `tipo_test`;
- revisión de alcance contra temario;
- informe de rework cerrado o pendiente justificado;
- decisión del Director: `apto_local`, `pendiente_rework` o `bloqueado_real`.

No confundir wave parada con entrega completa. Antes de cerrar, Orquesta debe
comparar el contrato esperado con los artefactos reales: carpetas, ficheros,
conteos, esquema y errores. Si faltan carpetas o el esquema no encaja, el estado
de entrega es `partial` o `schema_invalid` aunque todos los procesos estén
`stopped`.

Si se va a publicar, OPES debe ejecutar sus validadores y el flujo de subida por
API. Orquesta no publica ni toca producción por sí misma.
