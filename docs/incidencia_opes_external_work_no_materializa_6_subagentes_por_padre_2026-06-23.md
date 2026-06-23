# Incidencia OPES: external-work no materializa 6 subagentes por padre

Fecha: 2026-06-23

## Contexto

Curso OPES: Auxiliar de Servicios Generales C2, Diputación de Granada.
Directorio de curso:
`/home/alberto/Trabajo/OPES/opes-salidas/diputacion_granada/cursos_opes_2026/administracion_especial/auxiliar-de-servicios-generales`

Se lanzaron 20 trabajos externos, uno por tema oficial, mediante:
`POST /api/v0/external-work/run`.

Cada contrato incluye:

- skill/contrato OPES de padre por tema;
- seis subroles obligatorios: fuentes, reutilización local, redacción/ampliación, visuales, tests/tutor y HTML/RAG/audio/QA;
- límite preferente de 6 subagentes por padre.

## Resultado observado

Orquesta acepta los 20 runs padre, pero `external-work/run` crea como máximo un agente Codex operativo por run padre. Los seis subroles quedan como instrucciones dentro del paquete del padre, no como seis tareas hijas reales con waits/cohorte/cierre causal.

Evidencia local:

- respuestas OK en `00_control/orquesta_jobs/responses/external_work_tema_001_response.json` ... `external_work_tema_020_response.json`;
- runtime con directorios `run-external-work-aux-servicios-generales-c2-padre-XXX`;
- procesos Codex por padre, no seis hijos por padre.

## Impacto

Para OPES completo, esto reduce paralelismo y trazabilidad. El padre puede entregar trabajo útil si cubre los subroles, pero no cumple la regla operativa preferente de `1 padre + 6 subagentes reales` por tema.

## Comportamiento esperado

Cuando un trabajo OPES declare `interface_refs` como `opes.padre-tema-6-subroles.v1` o campo equivalente `subroles_required`, Orquesta debe:

1. crear un run padre por tema;
2. crear 6 tareas hijas reales por padre, con `parent_task_ref`/`parent_run_ref`;
3. asignar write-set no solapado dentro del tema;
4. esperar ACK de cada subagente;
5. permitir al padre integrar, validar y cerrar;
6. reflejar en estado/API cuántos subagentes están pendientes, vivos, completados, fallidos o bloqueados.

## Tarea técnica propuesta

Implementar materializador OPES de subroles para `external-work/run`:

- detectar contrato OPES por `interface_refs`, `work_kind` o `input_fields.subroles_required`;
- expandir subroles en tareas hijas causales;
- conservar compatibilidad con trabajos no OPES;
- añadir tests: un external-work OPES con 1 tema produce 1 padre y 6 hijos; uno normal produce 1 agente;
- añadir endpoint o proyección de estado por `run_ref` con conteo padre/hijos.

## Workaround actual

Mantener la regla en el contrato del padre y aceptar entrega si cubre subroles con evidencia, pero documentar que no hubo 6 procesos hijos reales. No marcar como autonomía OPES 100% hasta corregir esto.
