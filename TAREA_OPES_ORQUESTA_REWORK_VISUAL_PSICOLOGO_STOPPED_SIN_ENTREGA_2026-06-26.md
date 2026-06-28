# Tarea Orquesta - Rework Visual Psicólogo Detenido Sin Entrega

Fecha: 2026-06-26.

## Contexto

Desde OPES se lanzó una ola controlada de rework visual profesional para
Psicólogo/a A1/A2, temas 019-030. El objetivo era corregir el bloqueo de QA:
los temas tienen texto, tests y audios válidos, pero sus visuales actuales son
PNG de esquemas/cajas en `assets/esquemas`, no arte final profesional OPES.

Payload enviado:

`/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/rework_visual_psicologo_2026-06-26/orquesta_jobs/external_work_rework_visual_psicologo_019_030_20260626.json`

Endpoint:

`POST http://127.0.0.1:19023/api/v0/external-work/run`

Run:

`run-opes-psicologo-rework-visual-019-030-20260626`

Respuesta guardada:

`/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/rework_visual_psicologo_2026-06-26/responses/external_work_rework_visual_psicologo_019_030_20260626.response.json`

## Resultado observado

La API respondió `estado=ok` con evidencias:

- `evidence-ref-external-work-run-started`
- `evidence-ref-external-work-run-programacion`
- `evidence-ref-external-work-run-queued`
- `evidence-ref-run-coordinator-executed`

Pero el estado posterior de `autoprogramming/status` mostró:

- `queue_count=0`
- `active_runs=0`
- `agents_total=0`
- `agents_in_flight=0`
- `matching_terminal.status=stopped`
- sin artefactos nuevos en el curso
- sin informe `09_validacion/rework_visual_psicologo_019_030_20260626`
- sin nuevos assets en `08_assets/profesionales`
- sin nuevos assets en `html_final/img` o `html_ampliado/img` para temas 019-030

## Incidencia

`external-work/run` acepta la petición, registra evidencias de arranque/cola y
detiene la run sin materializar agentes ni producir entrega. Para OPES esto es
equivalente a `external_work_accepted_stopped_without_delivery`.

No es una incidencia de contenido OPES: el payload es JSON válido y contiene
write-set, criterios de aceptación, constraints, refs compactas y pruebas
requeridas.

## Impacto

Psicólogo/a A1/A2 no puede avanzar a 100% porque el rework visual profesional
queda sin ejecutar por Orquesta. Codex no debe fingir cierre promocionando los
PNG de esquemas actuales como arte final.

## Criterio esperado

Orquesta debe hacer una de estas cosas:

1. materializar agentes o tarea externa real;
2. dejar la run en estado no terminal si sigue pendiente;
3. cerrar con error explícito y causa accionable;
4. entregar artefactos o ACK de bloqueo real.

No debe cerrar `stopped` sin agente, sin entrega y sin motivo operativo visible.
