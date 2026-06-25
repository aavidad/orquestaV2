# TAREA OPES - External work queda `done` sin agentes ni producto

Fecha: 2026-06-25.

## Contexto

Desde OPES se lanzó el cierre local de `Operario Tractorista Grupo 5` mediante
`POST /api/v0/external-work/run`.

Run:

`run-external-work-opes-tractorista-ap-cierre-integracion-20260625`

Payload local:

`/home/alberto/Trabajo/OPES/opes-salidas/diputacion_granada/cursos_opes_2026/administracion_especial/operario-tractorista-grupo-5/00_control/orquesta_jobs/cierre_20260625/external_work_cierre_tractorista_ap_integracion.json`

Respuesta de lanzamiento:

`/home/alberto/Trabajo/OPES/opes-salidas/diputacion_granada/cursos_opes_2026/administracion_especial/operario-tractorista-grupo-5/00_control/orquesta_jobs/cierre_20260625/external_work_cierre_tractorista_ap_integracion_response.json`

## Evidencia

La API aceptó el trabajo:

```text
estado=ok
evidence-ref-external-work-run-started
evidence-ref-external-work-run-programacion
evidence-ref-external-work-run-queued
```

Pero `POST /api/v0/runs/supervise` devolvió:

```text
estado=ok
stop_reason=done
last.status=done
evidence-ref-codex-supervisor-drain-projection-tasks-0
evidence-ref-codex-supervisor-drain-projection-open-tasks-0
evidence-ref-codex-supervisor-drain-projection-requested-agents-0
```

Evidencia guardada en OPES:

`/home/alberto/Trabajo/OPES/opes-salidas/diputacion_granada/cursos_opes_2026/administracion_especial/operario-tractorista-grupo-5/00_control/orquesta_jobs/cierre_20260625/supervise_after_zero_agents.json`

El curso no cambió: `html_final`, `html_ampliado`, `04_markdown`, `09_tests`
y RAG final seguían sin artefactos canónicos nuevos.

## Problema

Orquesta no puede considerar `done` una run de `external_work` OPES aceptada si
no materializa ninguna tarea, ningún agente, ningún ACK y ningún artefacto de
producto.

Esto rompe la autonomía OPES porque el director humano ve una run `done`, pero
el trabajo real queda sin ejecutar.

## Tarea Técnica

1. Añadir guardia de cierre: una run `external_work` con `requested_agents=0`,
   `tasks=0`, sin ACK y sin outputs esperados no puede cerrarse como `done`.
2. Emitir estado explícito:
   `external_work_accepted_no_agent_materialized`.
3. Exponerlo en `/api/v0/runs/supervise` con `stop_reason` distinguible y
   acción recomendada.
4. Hacer que el director residente reintente materialización o abra incidencia
   causal en vez de quedarse `idle`.
5. Añadir test de regresión con payload OPES válido que exige outputs y
   verifica que no se marca `done` hasta crear agente/ACK o error recuperable.

## Impacto OPES

En Tractorista AP el contenido textual y tests existen, pero faltan integración
HTML/RAG/visuales. Orquesta aceptó el cierre y no lanzó nada. El trabajo tuvo
que continuar con auditoría humana y subagentes externos, dejando Orquesta como
fuente de fallo documentada en lugar de director autónomo.
