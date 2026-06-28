# TAREA OPES - QA visual remota TCAE queda bloqueada sin agente arrancado

Fecha: 2026-06-26.

## Contexto

Desde OPES se lanzó una auditoría de cierre para Psicólogo, Auxiliar de
Servicios Generales y Operario, exigida antes de dar temarios por terminados:
capturas locales y remotas protegidas de todas las páginas publicables,
comparación con canon TCAE e informe de índices, fotos, infografías, tests,
audios y navegación.

Run:

`run-opes-qa-visual-remota-tcae-psicologo-asg-operario-20260626`

Payload OPES:

`/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/revision_100_psicologo_asg_operario_2026-06-26/orquesta_jobs/external_work_qa_visual_remota_tcae_20260626.json`

Respuesta de lanzamiento:

`/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/revision_100_psicologo_asg_operario_2026-06-26/orquesta_jobs/external_work_qa_visual_remota_tcae_20260626.response.json`

Supervisión:

`/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/revision_100_psicologo_asg_operario_2026-06-26/orquesta_jobs/supervise_qa_visual_remota_tcae_20260626.json`

Stats:

`/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/revision_100_psicologo_asg_operario_2026-06-26/orquesta_jobs/stats_qa_visual_remota_tcae_20260626.json`

## Evidencia

`POST /api/v0/external-work/run` devuelve `estado=ok` y encola la run.

`POST /api/v0/director/stats` muestra:

- `status=activa`;
- `tasks_total=1`;
- `tasks_open=1`;
- `agents_requested=1`;
- `agents_started=0`;
- `agents_in_flight=0`;
- agente en estado `requested`.

`POST /api/v0/runs/supervise` termina con:

- `stop_reason=stopped`;
- `last.status=stopped`;
- `operational-director-plan-state:blocked`;
- `wait-subagents-terminal-without-delivery`;
- `requested_agents=1`;
- sin agente arrancado;
- sin ACK;
- sin artefacto de producto.

## Impacto

Orquesta acepta un trabajo OPES válido de QA/cierre, pero no materializa el
agente solicitado ni ofrece un estado causal accionable para reintento o
diagnóstico de capacidad. El director humano tiene que continuar con validadores
locales para no bloquear la revisión.

En esta ejecución, la auditoría local sí detectó fallos reales:

- Operario: 42/42 capturas locales fallan por carcasa TCAE incompleta.
- Auxiliar de Servicios Generales: 86/90 capturas locales fallan por
  `course-index`/`topic-page` ausentes.

## Tarea técnica

1. Cuando una run `external_work` queda con `agents_requested>0`,
   `agents_started=0` y `agents_in_flight=0`, no dejarla solo como `blocked`
   genérico.
2. Emitir estado explícito:
   `external_work_agent_requested_not_started`.
3. Exponer causa pública: capacidad, autenticación, runtime, cola, outbox,
   política o error desconocido.
4. Proponer acción concreta: reintentar materialización, revisar runtime,
   consultar capacidad o cerrar con incidencia causal.
5. Añadir prueba de regresión con un payload OPES de QA visual que exige
   agentes y verifica que no queda en `wait-subagents-terminal-without-delivery`
   sin diagnóstico causal.
