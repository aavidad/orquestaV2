# Tarea técnica - OPES supuestos: validación de esquema, cobertura y entrega parcial

## Contexto

Durante la creación de supuestos prácticos TCAE el 2026-06-23, Orquesta coordinó
olas reales de agentes, pero necesitó intervención del director para detectar y
corregir problemas que deberían ser automáticos.

Olas afectadas:

- `opes-supuestos-tcae-directa-20260623`
- `opes-supuestos-tcae-rework-lote1-20260623`
- `opes-supuestos-tcae-rework-lote2-20260623`
- `opes-supuestos-tcae-rework-lote3-20260623`
- `opes-supuestos-tcae-rework-lote4-20260623`
- `opes-supuestos-tcae-rework-lote5-20260623`

## Fallos Observados

1. La ola inicial consumió demasiado contexto en búsquedas amplias y no produjo
   artefactos suficientes.
2. Las olas de rework 1 a 4 dejaron 18 de 24 carpetas esperadas. Faltaban seis
   entregas completas, pero Orquesta no generó automáticamente la ola de
   recuperación a partir de la cobertura de salida.
3. Varios ficheros entregaron `tasks` en vez de `questions`.
4. Algunos supuestos de desarrollo usaron `kind: development_task` en vez del
   contrato importable `kind: open_response`.
5. El resumen de wave informaba procesos parados, pero no una métrica clara de
   cobertura de artefactos esperados ni de conformidad de esquema.

## Objetivo

Añadir a Orquesta capacidad nativa para cerrar trabajos OPES por contrato de
artefactos, no solo por estado de procesos.

## Requisitos Funcionales

- Permitir que una wave declare un contrato de salida con:
  - carpetas esperadas;
  - ficheros esperados;
  - conteo mínimo por tema;
  - esquema JSON/JSONL requerido;
  - valores permitidos por campo.
- Al terminar una wave, calcular automáticamente:
  - artefactos entregados;
  - artefactos faltantes;
  - artefactos inválidos;
  - cobertura por tema/lote;
  - tareas de rework necesarias.
- Si hay faltantes recuperables, generar propuesta de rework o relanzamiento
  con prompts compactos y write-set limitado.
- Diferenciar `process_status=stopped` de `delivery_status=complete`.
- Exponer en CLI/API un resumen humano:
  - agentes vivos;
  - agentes parados;
  - entregas esperadas;
  - entregas válidas;
  - entregas faltantes;
  - errores de esquema;
  - porcentaje de cierre real.

## Contrato mínimo para supuestos OPES

Para casos prácticos OPES importables, Orquesta debe poder validar al menos:

- `schema_version = opes_practical_case.v1`
- `case_id`
- `course_id`
- `course_variant`
- `topic_id`
- `topic_title`
- `case_type`
- `questions`
- `scope_status`
- `editorial_decision`
- `questions[].kind in {multiple_choice, open_response}`
- en `multiple_choice`: cuatro opciones `A-D`, una correcta y explicación;
- evidencia de temario en cada pregunta.

## Criterios De Aceptación

- Una wave con 24 carpetas esperadas y solo 18 entregadas queda como
  `delivery_status=partial`, no como cierre equivalente a correcto.
- Una entrega con `tasks` en vez de `questions` se marca `schema_repairable` o
  `schema_invalid`, según política del consumidor.
- Una pregunta con `kind: development_task` queda rechazada o normalizada solo si
  existe regla explícita del adaptador OPES.
- El CLI de estado muestra porcentaje de entrega real y lista breve de faltantes.
- El supervisor puede lanzar una ola de rework con solo las carpetas faltantes.
- El cierre global solo permite `complete` si el contrato de artefactos está
  satisfecho.

## Evidencia De Resolución Manual

En OPES se resolvió con scripts locales:

- `tools/normalizar_supuestos_tcae.py`
- `tools/validar_supuestos_tcae.py`
- `tools/consolidar_supuestos_tcae.py`

Ruta:

`/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/tcae_supuestos_practicos_2026-06-23/`

Esta tarea debe convertir esa comprobación manual en capacidad de Orquesta o en
un adaptador OPES invocable por Orquesta.
