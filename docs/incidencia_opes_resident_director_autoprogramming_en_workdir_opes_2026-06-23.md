# Incidencia OPES: Resident Director Lanza Autoprogramación En Workdir Del Curso

Fecha: 2026-06-23.

## Contexto

Durante la creación del temario `Auxiliar de Servicios Generales` desde OPES se
arrancó Orquesta con:

- `ORQUESTA_CODEX_PROJECT_WORKDIR` apuntando al directorio del curso OPES;
- `ORQUESTA_OPES_PROJECT_WORKDIR` apuntando al mismo curso;
- `ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED=1`;
- `ORQUESTA_SERVER_AUTONOMY_ENABLED=1`.

El objetivo era que Orquesta dirigiera padres OPES por tema. Antes de enviar los
jobs OPES, el residente lanzó tareas de autoprogramación/backlog de Orquesta
dentro del `project_workdir` del curso.

## Evidencia

En el curso quedaron procesos y runtime bajo:

`opes-salidas/diputacion_granada/cursos_opes_2026/administracion_especial/auxiliar-de-servicios-generales/.orquesta-codex-runtime/`

con refs de tipo:

- `request-ref-autoprogramming-backlog-srv-task-*`;
- `request-ref-autoprogramming-backlog-t36-*`;
- `agent-ref-assessment-task-autoprogramming-*`.

El estado contaminado se archivó en OPES como evidencia:

- `00_control/orquesta_state_autoresidente_ruido_20260623_223007/`;
- `.orquesta-codex-runtime_autoresidente_ruido_20260623_223007/`.

## Problema

Cuando `project_workdir` se fija en un workspace OPES, el residente no debe usar
ese directorio para tareas internas de autoprogramación de Orquesta. Mezcla
artefactos de núcleo con artefactos de producto y puede arrancar agentes ajenos
al trabajo OPES solicitado.

## Corrección Esperada

- Separar `project_workdir` de producto OPES y `project_workdir` de
autoprogramación de Orquesta.
- Si `project_ref=opes` o `ORQUESTA_OPES_PROJECT_WORKDIR` está activo, el
resident director no debe lanzar backlog/autoprogramación salvo que exista un
`ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_PROJECT_WORKDIR` explícito distinto.
- Añadir guarda de arranque que impida `resident_director + autoprogramming`
sobre un directorio OPES sin opt-in explícito.
- Exponer en readiness/status una advertencia clara cuando se desactive el
residente por esta guarda.

## Workaround Operativo Usado

Para continuar el curso sin contaminarlo, se reinicia Orquesta con estado limpio
y `ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED=0`; los padres OPES se lanzan por
`POST /api/v0/external-work/run` y se supervisan por `POST /api/v0/runs/supervise`.

Este workaround no sustituye la corrección de la app.
