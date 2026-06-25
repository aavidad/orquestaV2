# Tarea Orquesta: OPES Integrador Social no materializa 6 subagentes por padre

Fecha: 2026-06-25

## Problema

En la ola real de Integrador Social B, temas 015-020, Orquesta aceptó seis
padres OPES en la instancia `127.0.0.1:8793`, pero no arrancó los seis
subagentes reales por padre.

El plan queda con un padre `in_progress` y seis tareas hijas/subroles
`pending`, pero el runtime sólo muestra un proceso Codex por tema. Esto no
cumple el patrón OPES vigente de producción de temario completo: un padre por
tema y seis subroles/subagentes preferentes por padre.

## Evidencia

Curso OPES:

```text
/home/alberto/Trabajo/OPES/opes-salidas/diputacion_granada/cursos_opes_2026/administracion_especial/auxiliar-tecnico-superior-de-integracion-social
```

Runs afectados:

```text
run-opes-integracion-social-b-t015-padre-20260625-ola2
run-opes-integracion-social-b-t016-padre-20260625-ola2
run-opes-integracion-social-b-t017-padre-20260625-ola2
run-opes-integracion-social-b-t018-padre-20260625-ola2
run-opes-integracion-social-b-t019-padre-20260625-ola2
run-opes-integracion-social-b-t020-padre-20260625-ola2
```

Ficheros de respuesta/stats:

```text
/home/alberto/Trabajo/OPES/opes-salidas/diputacion_granada/cursos_opes_2026/administracion_especial/auxiliar-tecnico-superior-de-integracion-social/00_control/director_20260625/orquesta_responses/stats_tema_015_ola2_8793.json
...
/home/alberto/Trabajo/OPES/opes-salidas/diputacion_granada/cursos_opes_2026/administracion_especial/auxiliar-tecnico-superior-de-integracion-social/00_control/director_20260625/orquesta_responses/stats_tema_020_ola2_8793.json
```

Resumen observado en los seis temas:

```text
status=activa
phase=programacion
tasks_total=6/7
agents_started=1
agents_in_flight=1
agents_delivered=0
tasks_closed=0
```

Los `agent_packet.json` declaran `max_child_agents: 6` y `child_task_refs`
para:

```text
subrole-fuentes
subrole-reutilizacion
subrole-redaccion
subrole-visuales
subrole-tests-tutor
subrole-html-rag-audio-qa
```

Pero no aparecen directorios/procesos hijos reales ni ACK de hijos en los
runtimes `run-opes-integracion-social-b-t015...t020-padre-20260625-ola2`.

## Fallo adicional detectado

El prompt del padre generado por `orquesta-app-codex-stack` decía:

```text
Delegacion operativa: si necesitas ayuda y el runtime lo permite, activa subagentes...
```

Eso es demasiado blando para un contrato OPES con `child_task_refs` ya
declaradas: el padre puede interpretarlo como opcional y cubrir roles en un
solo proceso, sin materializar hijos reales.

## Mitigación aplicada

Se cambia la composición Codex para que, cuando una `WorkflowTaskV0` trae
`child_task_refs`, el objetivo del padre indique que esas tareas hijas ya están
declaradas por Orquesta, que no son opcionales y que el padre no debe cerrar su
ACK sin evidencia de ACK, entrega, bloqueo o rework pendiente por cada hija.

Ficheros modificados:

```text
modulos/orquesta-app-codex-stack/spec_task_v0.go
modulos/orquesta-app-codex-stack/spec_task_programming_v0_test.go
```

## Pendiente técnico

La mitigación mejora el contrato entregado al padre, pero no cierra el problema
principal: Orquesta debe materializar realmente las seis tareas hijas de OPES
cuando detecte `opes.padre-tema-6-subroles.v1` o `subroles_required=6`.

## Criterios de aceptación

- Un external-work OPES con `subroles_required=6` arranca una cohorte causal:
  un padre y seis hijos reales.
- `agents_started` refleja padre + hijos, no sólo el padre.
- La API/ops distingue padre, hijos pendientes, vivos, completados, fallidos y
  bloqueados.
- El padre no puede cerrar como completo si sus `child_task_refs` siguen sin
  ACK, entrega, bloqueo o rework documentado.
- La admisión de olas grandes OPES devuelve rápido `queued` o `accepted`; no
  bloquea el endpoint HTTP por materializar muchos hijos.

## Actualización 2026-06-25 23:20

Revisión posterior de la misma ola Integrador Social B temas 015-020:

- Los seis padres sí dejaron `agent_ack.json` en disco con `status=completed`.
- Los borradores ampliados siguen por debajo del mínimo B de 10.800 palabras:
  tema 015 unas 3.936 palabras, tema 016 unas 4.771, tema 017 unas 3.397,
  tema 018 unas 3.880, tema 019 unas 4.008 y tema 020 unas 3.502.
- La API de la instancia OPES usada (`127.0.0.1:8793`) dejó de responder tras
  la ola, por lo que los `stats_tema_015...020_ola2_8793.json` quedaron
  obsoletos y seguían mostrando `agents_delivered=0` aunque los ACK existían en
  disco.
- La instancia viva `127.0.0.1:8787` no es una sustituta limpia para este flujo:
  su estado informa `wrong-project-work-dir`, supervisor `stalled`,
  `resident_director_status=disabled` y errores repetidos de persistencia.
- Materialización parcial observada en runtime:
  - tema 015: sólo padre, sin carpetas `subrole-*`;
  - tema 016: seis carpetas `subrole-*`, pero sin `agent_ack.json` de subroles;
  - tema 017: cinco carpetas `subrole-*`, falta al menos `html-rag-audio-qa`,
    sin ACK de subroles;
  - tema 018: seis carpetas `subrole-*`, pero sin ACK de subroles;
  - tema 019: sólo padre, sin carpetas `subrole-*`;
  - tema 020: seis carpetas `subrole-*`, pero sin ACK de subroles.

Conclusión operativa: hay entrega recuperable de padres, pero no hay cierre
causal verificable de los seis subagentes por padre. OPES debe continuar con
rework/expansión de contenido usando esos borradores como insumo, no marcarlos
como `ready`.
