# Tarea Orquesta: Evitar Bucle De Rework Residente En OPES

Fecha: 2026-06-23.

## Contexto

Durante la producción del temario OPES `auxiliar-de-servicios-generales` en `/home/alberto/Trabajo/OPES`, Orquesta lanzó los 20 padres y produjo artefactos útiles. Después, el director residente siguió generando oleadas de `review-rework` y `assessment` sobre temas ya validados y con ACK.

Además, los 120 contratos de subagentes generados para el patrón 20 padres x 6 subagentes no se materializaron como respuestas de subagentes aceptadas.

## Síntomas

- Ticks de `resident_director_tick_start/result` cada pocos segundos.
- `supervisor_tick_result` con `queue_size` estable alrededor de 21.
- Nuevas carpetas `agent-ref-task-ref-review-rework-task-ref-review-rework...` sobre temas ya cerrados.
- Reworks encadenados sin cierre causal global del curso.
- Sin endpoint operativo claro para pausar solo el director residente manteniendo los procesos útiles ya lanzados.

## Arreglo Necesario

1. Añadir deduplicación causal de reworks OPES por `course_id + topic_id + finding_ref`.
2. No abrir un nuevo `review-rework` si el tema ya tiene ACK posterior al hallazgo y validación focal pasada.
3. Añadir estado terminal de tema OPES: `ready_locked`, `pending_global_closure`, `needs_rework_specific`.
4. Exigir que todo rework tenga una incidencia concreta: extensión, tests, HTML, texto público, visuales, RAG, audio o manifest.
5. Añadir endpoint o comando runtime para pausar/reanudar el director residente sin parar el servidor completo.
6. Materializar los 6 subagentes por padre como tareas hijas reales o registrar explícitamente que el padre ejecutó subroles sin subprocesos.
7. Exponer en estado/API una cuenta humana clara: padres vivos, subagentes vivos, reworks vivos, bloqueados, con ACK y sin ACK.

## Criterio De Aceptación

En un curso OPES de prueba con 3 temas:

- Orquesta lanza padres y subagentes según contrato.
- Un rework corregido no genera otro rework idéntico.
- El residente se detiene al quedar temas en `ready_locked` o pasa a `pending_global_closure`.
- La API de estado permite distinguir trabajo vivo real, cola pendiente y bucle bloqueado.
