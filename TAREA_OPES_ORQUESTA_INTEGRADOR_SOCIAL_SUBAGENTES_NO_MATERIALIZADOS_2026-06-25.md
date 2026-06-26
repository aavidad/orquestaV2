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

## Actualización 2026-06-25 23:31 - Ola3 expansión T017 abortada por cuota al materializar subagentes

Contexto OPES: curso `auxiliar-tecnico-superior-de-integracion-social`, nivel B, ola3 de expansión textual de temas 015-020 en servidor aislado `127.0.0.1:18793` con `ORQUESTA_CODEX_PROJECT_WORKDIR=/home/alberto/Trabajo/OPES` y residente desactivado según workaround vigente.

Evidencia nueva:

- `run-opes-integracion-social-b-t017-expansion-20260625-ola3` fue aceptada y supervisada con `stop_reason=dispatch_started`, `last.status=running_live`.
- El proceso del padre T017 dejó de aparecer en `ps` sin escribir `agent_ack.json` ni modificar `temas/tema_017/borrador_ampliado.md`.
- `codex_usage_accounting.json` del agente T017 indica `quota.status=exhausted` y `usage.total_tokens=133153`.
- El final de `codex_stderr.log` muestra que el padre intentó `SpawnAgent` para subroles y recibió: `You've hit your usage limit... try again at Jun 26th, 2026 4:16 AM`.
- No hay ACK causal ni entrega parcial útil en producto; el tema 017 sigue con 3.403 palabras y estado `pendiente_continuar`.

Impacto:

- Orquesta considera inicialmente el run como `running_live`, pero el agente puede morir por cuota durante materialización de subagentes sin ACK.
- El patrón de seis subagentes por padre queda bloqueado por cuota y no produce checkpoint de reanudación automático.

Tarea para agente de Orquesta:

- Detectar salida de Codex por cuota agotada y convertirla en estado recuperable explícito: `quota_exhausted_retry_after`, con `retry_after` cuando el proveedor lo informe.
- Escribir o exigir checkpoint/ACK de control mínimo cuando el proceso termina por cuota sin artefacto.
- No relanzar en bucle hasta que llegue la ventana de cuota; dejar follow-up deduplicado para reanudar el mismo run o crear run de rework con mismo write-set.
- Exponer en stats/director una diferencia clara entre `running_live`, `process_exited_without_ack`, `quota_exhausted` y `waiting_external`.

## Actualización 2026-06-25 23:36 - Retry T017 clasificado como capacity_limited por metadatos de reintento

Contexto: tras recuperar cuota se relanzó T017 como `run-opes-integracion-social-b-t017-expansion-20260625-ola3-retry1` en `127.0.0.1:18793`.

Evidencia:

- `/api/v0/external-work/run` aceptó el run con estado `ok`.
- `/api/v0/runs/supervise` devolvió `runtime_error` sin crear carpeta runtime de agente.
- `/api/v0/director/stats` para el run muestra `agents_started=1`, `agents_stop_requested=1`, `agents_in_flight=1`, `stop_control.status=stop_pending` y `agent_stop_requests` con `reason:capacity_limited`.
- El run no escribió `agent_ack.json` ni modificó `temas/tema_017/borrador_ampliado.md`.
- El encargo de retry incluía explícitamente metadatos como `previous_failure: quota_exhausted_without_ack`, `quota-now-available-retry` y una política de reintento. Es probable que la evaluación de Orquesta haya clasificado el trabajo como `capacity_limited` por esos términos en vez de lanzar el trabajo limpio.

Impacto:

- Un reintento legítimo puede quedar bloqueado si el propio contexto documenta el motivo de cuota anterior.
- La señal `capacity_limited` queda mezclada con metadatos históricos y no con capacidad real vigente.

Tarea para agente de Orquesta:

- Separar metadatos históricos de fallo anterior de la decisión de capacidad actual.
- No detener un run nuevo por aparecer términos como cuota/capacidad en `current_state_refs`, `director_brief` o notas de retry si el operador acaba de confirmar que la cuota está disponible.
- Exponer en diagnóstico qué texto o señal originó `capacity_limited` y si procede de estado actual, proveedor, política interna o metadato histórico.
- Permitir retry limpio deduplicado tras `quota_exhausted` sin contaminar el prompt de ejecución.

## Actualización 2026-06-25 23:40 - Relanzamiento limpio T017 ola4 queda falso-activo

Contexto: se creó un relanzamiento limpio de T017 como
`run-opes-integracion-social-b-t017-expansion-20260625-ola4`, eliminando del
prompt y de metadatos términos como cuota, `quota`, `capacity_limited` y
`retry-after` para evitar la contaminación detectada en `ola3-retry1`.

Evidencia:

- `/api/v0/external-work/run` aceptó el run con estado `ok`.
- `/api/v0/runs/supervise` devolvió `dispatch_started/running_live` y creó
  runtime para `agent-ref-task-ref-app-change-appchange-e2f06fd5b409d31aad5a89dd87593119`.
- El agente principal terminó sin `agent_ack.json`; `codex_stderr.log` muestra
  `401 Unauthorized: Missing bearer or basic authentication in header` contra
  `wss://api.openai.com/v1/responses` y después contra
  `https://api.openai.com/v1/responses`.
- Orquesta generó un agente de reemplazo `agent-ref-assessment-...`, pero ese
  proceso también terminó sin ACK y `codex_usage_accounting.json` indica
  `quota.status=exhausted`; el log del reemplazo dice `try again at 11:55 PM`.
- A las 23:39 CEST no quedaban procesos `t017/ola4` vivos en `ps`.
- `/api/v0/director/stats` mantenía el run como `status=activa`, con
  `agents_lost=1`, `agents_stop_requested=1`, `agents_in_flight=1`,
  `stop_control.status=stop_pending`, `progress.percent_complete=1` y tareas
  hijas pendientes.
- El producto no cambió: `temas/tema_017/borrador_ampliado.md` seguía en 3.403
  palabras, sin entrega parcial útil.

Impacto:

- Orquesta puede dejar un run como activo/falso-vivo aunque no exista proceso,
  no haya ACK y el producto no haya avanzado.
- Se mezclan dos causas distintas: autenticación HTTP 401 del agente principal y
  cuota agotada temporal del reemplazo hasta las 23:55 CEST.
- La cola/autoprogramming informa runs `running_stale`, pero no cierra ni
  reclasifica automáticamente `process_exited_without_ack`.

Tarea para agente de Orquesta:

- Clasificar `401 Unauthorized` como bloqueo de proveedor/autenticación
  diferente de cuota.
- Cuando `ps` no tenga proceso vivo y falte ACK, pasar el agente a
  `process_exited_without_ack` y no dejar `agents_in_flight=1` indefinidamente.
- Si un reemplazo queda `stop_requested` por cuota con hora de reintento, exponer
  `retry_after_local` y no mantener la run como trabajo vivo.
- La cola debe distinguir `running`, `running_stale`, `waiting_retry_after`,
  `auth_blocked` y `lost_without_ack` para que OPES no pierda horas esperando.

## Actualización 2026-06-25 23:44 - T017 ola5 confirma ventana real de cuota

Contexto: después de comprobar que otros temas de la ola seguían avanzando, se
creó `run-opes-integracion-social-b-t017-expansion-20260625-ola5` desde el
encargo limpio de `ola4`, cambiando solo identificadores a `ola5`, elevando
prioridad a 100 y verificando que el JSON no contenía términos de cuota,
`capacity_limited`, `retry-after`, `401` ni fallos previos.

Evidencia:

- `/api/v0/external-work/run` aceptó `ola5`.
- La primera supervisión dejó inicialmente `requested_agents=0`; una consulta
  posterior de `/api/v0/director/stats` mostró que Orquesta sí materializó 7
  tareas, respondió la pregunta de director y arrancó el padre
  `agent-ref-task-ref-app-change-appchange-65eeacb7de7cbe2a20d1455229c44c81`.
- El runtime se creó en
  `.../18793_runtime/run-opes-integracion-social-b-t017-expansion-20260625-ola5/...`.
- El agente terminó sin `agent_ack.json`; `codex_stderr.log` volvió a mostrar
  `You've hit your usage limit... try again at 11:55 PM`.
- A las 23:43 CEST no había proceso vivo para `ola5`, `codex_usage_accounting`
  indicaba `quota.status=exhausted` y el tema 017 seguía con 3.403 palabras.
- `/api/v0/director/stats` mantenía `status=activa`,
  `agents_stop_requested=1`, `agents_in_flight=1`,
  `stop_control.status=stop_pending` y motivo `capacity_limited`.

Impacto:

- La confirmación humana de "ya hay cuota" no basta: el runtime hijo de Codex
  puede seguir devolviendo una hora futura de reintento y Orquesta no la refleja
  como `waiting_retry_after`.
- Un relanzamiento limpio puede quedarse de nuevo como activo/falso-vivo con
  `agents_in_flight=1` aunque el proceso haya terminado y no haya ACK.

Tarea para agente de Orquesta:

- Extraer la hora de reintento del stderr de Codex y convertirla en campo
  estructurado (`retry_after_local`/`retry_after_provider`).
- Si el agente termina por cuota sin ACK, reconciliar inmediatamente a
  `waiting_retry_after` o `quota_exhausted`, no a `running/stop_pending`.
- Evitar que runs posteriores compitan con runs fallidos T017 anteriores cuando
  el bloqueo causal sea cuota con reintento explícito.

## Actualización 2026-06-26 00:24 - T017 ola6 cierra agentes pero no cierre lógico

Contexto: tras esperar a que terminara la ventana de cuota indicada por el
proveedor, OPES relanzó T017 con un encargo limpio como
`run-opes-integracion-social-b-t017-expansion-20260625-ola6`.

Evidencia:

- `/api/v0/external-work/run` aceptó `ola6`.
- `/api/v0/runs/supervise` materializó 7 agentes reales: padre y subroles
  `fuentes`, `reutilizacion`, `redaccion`, `visuales`, `tests-tutor` y
  `html-rag-audio-qa`.
- Los 7 agentes generaron `agent_ack.json`; `/api/v0/director/stats` mostró
  `agents_requested=7`, `agents_started=7`, `agents_delivered=7`,
  `agents_in_flight=0`, `agents_failed=0`, `agents_lost=0`,
  `reviews=7`, `review_results=7` y `accepted_reviews=7`.
- El producto avanzó: el padre dejó `temas/tema_017/borrador_ampliado.md` en
  10.942 palabras, por encima del mínimo B de 10.800.
- No quedaban procesos vivos para `t017-expansion-20260625-ola6` en `ps`.
- Una supervisión final devolvió `quiescent`, `required-tests-passed`,
  `review-accepted`, `open_tasks=0` en la proyección de drenaje, pero también
  `operational-closure-source-unavailable`.
- Después del drenaje, `/api/v0/director/stats` seguía mostrando
  `status=activa`, `current_phase=revision`, `tasks_open=7`,
  `tasks_closed=0`, `percent_complete=100` y cierre bloqueado por
  `programacion_entregas`, `validacion_final` y `fase_cierre`.

Impacto:

- Orquesta puede dejar una run al 100 %, sin agentes vivos y con todas las
  revisiones aceptadas, pero persistida como `activa` con tareas abiertas.
- Hay divergencia entre la proyección del drenaje (`open_tasks=0`) y
  `director/stats` (`tasks_open=7`).
- OPES puede continuar editorialmente porque hay artefactos útiles y ACKs, pero
  el operador no puede confiar en el cierre lógico del run para saber si debe
  lanzar la siguiente fase.

Tarea para agente de Orquesta:

- Reconciliar `tasks_open/tasks_closed` cuando todos los agentes asociados a
  tareas están `delivered/completed` y sus reviews están `accepted`.
- Resolver `operational-closure-source-unavailable`: debe indicar fuente
  faltante concreta o cerrar la fase si los prerequisitos existen.
- Evitar estados contradictorios como `percent_complete=100` +
  `agents_in_flight=0` + `accepted_reviews=7` + `status=activa` +
  `tasks_open=7`.
- Exponer un estado humano claro para OPES: `delivered_pending_global_qa`,
  `closed`, `blocked_by_missing_validation` o equivalente, pero no
  `activa` genérico.

## Actualización 2026-06-26 00:29 - Runs stale bloquean T021 hasta supervisión global

Contexto: después de cerrar materialmente T017 ola6, OPES creó un nuevo encargo
para `run-opes-integracion-social-b-t021-padre-20260626-ola1`.

Evidencia:

- `external-work/run` aceptó T021 y creó la entrada en cola.
- Dos llamadas dirigidas a `/api/v0/runs/supervise` contra T021 dejaron
  `tasks_total=7`, `tasks_open=7`, `director_answers=1`, pero
  `agents_requested=0`.
- `/api/v0/autoprogramming/status` mostró T021 en rango 10, `status=ready`,
  detrás de 9 runs antiguos marcados como `running_stale`: T017 ola5,
  T017 ola3-retry1, T017 ola4, T016 ola3, T017 ola3, T018 ola3, T019 ola3,
  T020 ola3 y T015 ola3.
- En `ps` no había procesos reales vivos de esos runs antiguos.
- Existían ACKs antiguos en runtime para T015/T016/T018/T019/T017 ola6, pero la
  cola seguía priorizando runs stale como activos.
- La acción recomendada por la API (`POST /api/v0/autoprogramming/supervise`)
  permitió finalmente que T021 pasara a `agents_requested=1`,
  `agents_started=1` y proceso Codex vivo.

Impacto:

- Una tarea nueva y válida puede quedar en cola detrás de runs stale sin
  procesos vivos.
- La supervisión dirigida del run nuevo no basta para saltar o reconciliar
  stale runs de mayor prioridad.
- OPES necesita intervención manual de supervisión global para que el siguiente
  tema arranque.

Tarea para agente de Orquesta:

- Reconciliar automáticamente runs `running_stale` sin proceso vivo y con ACK
  o fallo conocido antes de ordenar la cola.
- La cola debe degradar runs stale no ejecutables por debajo de `ready`
  ejecutables o marcarlos como `needs_reconciliation`, no bloquear nuevos
  trabajos.
- La supervisión dirigida de un run `ready` debería indicar explícitamente:
  `blocked_by_higher_priority_stale_runs` con refs, o activar reconciliación
  segura.

## Actualización 2026-06-26 00:36 - T021 arranca solo padre y no subroles

Contexto: OPES lanzó `run-opes-integracion-social-b-t021-padre-20260626-ola1`
para el tema 021, con contrato de padre más seis subroles por skill.

Evidencia:

- `/api/v0/director/stats` muestra `tasks_total=7`, `tasks_open=7`,
  `agents_requested=1`, `agents_started=1`, `agents_in_flight=1` y un único
  agente registrado.
- No existen `agent_ack.json` de subroles ni del padre en el runtime de T021
  en el momento de la comprobación.
- El log del padre indica literalmente el patrón operativo: primero esperó
  workers; después registró que no terminó ningún worker en 60 s y decidió
  continuar con entrega padre para no bloquear el trabajo.
- Una llamada dirigida a `/api/v0/runs/supervise` contra T021 quedó sin
  respuesta durante más de dos minutos y se tuvo que cortar el cliente `curl`;
  después, `director/stats` siguió devolviendo un único agente en vuelo.

Impacto:

- El contrato OPES de un padre con seis subroles no se materializa siempre en
  agentes reales.
- El padre puede producir material útil, pero Orquesta no garantiza el reparto
  paralelo esperado ni da un estado claro de por qué los subroles no arrancan.
- La supervisión dirigida puede quedar bloqueada sin devolver diagnóstico.

Tarea para agente de Orquesta:

- Asegurar que, cuando una run declara siete tareas OPES, Orquesta materializa
  los seis subroles o registra un bloqueo explícito por capacidad, cuota,
  permisos, cola o contrato inválido.
- Evitar que `/api/v0/runs/supervise` quede colgado sin salida diagnóstica.
- Exponer en `director/stats` un campo de espera útil: `waiting_for_subagents`,
  `subagents_not_materialized_reason` o equivalente.

### Evidencia posterior al ACK

Tras cerrar el padre de T021, `agent_ack.json` existe y el proceso terminó. En
ese momento `/api/v0/director/stats` pasó a `agents_requested=7`, pero solo
`agents_started=1`, `agents_delivered=1`, `agents_in_flight=0` y los seis
subroles quedaron en estado `requested` con `started=false`. El padre generó
ficheros `subrole_*.md` como coordinación interna, pero Orquesta no materializó
seis agentes reales.

## Actualización 2026-06-26 01:04 - Bucle de reconciliación tras ACKs de T021 ola2

Contexto: OPES lanzó `run-opes-integracion-social-b-t021-expansion-20260626-ola2`
para ampliar el tema 021 desde 6.542 palabras hasta el mínimo B.

Evidencia material útil:

- `temas/tema_021/borrador_ampliado.md` quedó en 10.824 palabras, por encima
  del mínimo B de 10.800.
- Existen ACKs para padre y seis subroles en el runtime de ola2:
  padre, fuentes, reutilización, redacción, visuales, tests/tutor y
  html-rag-audio-qa.
- Los subroles dejaron artefactos en
  `00_control/orquesta_runs/run-integracion-social-b-t021-expansion-20260626-ola2/`.

Fallo observado:

- Después de existir ACKs, `/api/v0/director/stats` seguía con
  `status=activa`, `tasks_open=7` y `tasks_closed=0`.
- Varios subroles aparecían simultáneamente como `lost` y con `agent_ack.json`
  existente.
- Orquesta lanzó agentes de evaluación/replan para subroles ya entregados.
  Cuando esos evaluadores generaban ACK, algunos quedaban también como `lost`
  y Orquesta lanzaba nuevos evaluadores. Se observó crecimiento de
  `agents_requested` de 7 a 15, `agent_assessments` de 0 a 8 y nuevos procesos
  de evaluación en vuelo, sin necesidad editorial porque el texto ya cumplía
  extensión.

Impacto:

- La run puede entrar en bucle de reconciliación aunque el producto OPES sea
  aprovechable y todos los subroles hayan entregado ACK.
- El estado `lost` no distingue entre "sin ACK real" y "ACK existe pero no se
  ha reconciliado".
- OPES no puede esperar indefinidamente a que la run cierre si el sistema
  genera reemplazos/evaluadores sobre entregas ya válidas.

Tarea para agente de Orquesta:

- Antes de marcar `lost` o lanzar `replace_agent`, comprobar si existe
  `agent_ack.json` válido y si el proceso terminó normalmente.
- Si existe ACK y artefacto requerido, reconciliar como `delivered` o
  `delivered_pending_review`, no crear evaluador de reemplazo.
- Limitar el número de replan/evaluadores por `task_ref` y cortar con estado
  explícito `reconciliation_loop_detected` cuando el mismo task tenga ACK y
  varios reemplazos.
- Exponer una acción/API segura para que OPES pueda cerrar una run materialmente
  entregada como `accepted_with_orquesta_reconciliation_issue` sin matar trabajo
  útil ni tocar el núcleo.

## Actualización 2026-06-26 01:12 - Stop público aceptado, pero stats queda sin cuerpo

Restricción de coordinación: hay otro agente trabajando en el núcleo de
Orquesta. Desde OPES no se ha tocado código ni ramas de Orquesta; solo se han
usado contratos públicos y ficheros runtime de parada cooperativa para no pisar
ese trabajo.

Acciones de control realizadas:

- Se escribieron `orquesta_shutdown_request.json` en los dos evaluadores de
  T021 ola2 que seguían en vuelo tras la entrega válida.
- Después se llamó a `POST /api/v0/runs/control` con:
  `action=stop`, `forced=true`,
  `run_ref=run-opes-integracion-social-b-t021-expansion-20260626-ola2`.
- La API respondió `estado=ok`, `status=stop_requested`,
  `checkpoint_recorded=true`, `forced=true` y evidencia
  `evidence-ref-mcp-run-control-checkpoint-recorded`.
- Los dos evaluadores vivos generaron finalmente
  `agent_shutdown_checkpoint_ack.json`; en `ps` ya no quedaron procesos Codex
  de esa run.

Fallo restante observado:

- Tras el stop, varias llamadas a `/api/v0/runs/supervise` y
  `/api/v0/director/stats` no devolvieron JSON útil antes del timeout del
  cliente.
- La parada material de procesos sí ocurrió, pero el plano de control no dio
  una respuesta final clara tipo `stop_confirmed`, `stopped` o
  `stop_pending_with_no_live_processes`.

Tarea adicional para agente de Orquesta:

- Garantizar que `runs/control action=stop forced=true` seguido de
  `runs/supervise` o `director/stats` nunca deja al cliente sin cuerpo JSON.
- Si no puede cerrar como `stopped`, devolver estado explícito y accionable:
  `stop_pending`, `stats_unavailable_after_stop` o
  `reconciliation_required_after_stop`.
- Confirmar cierre cruzando `agent_shutdown_checkpoint_ack.json`, ausencia de
  procesos hijos y `agent_ack.json`, y publicar `stop_confirmed` cuando cuadre.

## Actualización 2026-06-26 01:27 - Instancia OPES 18793 no responde a HTTP

Contexto: después del stop anterior, no quedan procesos Codex de la run T021
ola2 ni de Integración Social. La instancia usada por OPES sigue viva como
proceso `orquesta-server` con estado en:

`/home/alberto/Trabajo/OPES/opes-salidas/diputacion_granada/cursos_opes_2026/administracion_especial/auxiliar-tecnico-superior-de-integracion-social/00_control/director_20260625/orquesta_server/18793_state`

Evidencia:

- `ss -ltnp` muestra `127.0.0.1:18793` escuchando en `orquesta-server`.
- `GET /` contra `http://127.0.0.1:18793/` no devuelve cuerpo antes de un
  timeout de 8 segundos.
- `POST /api/v0/director/stats` tampoco devuelve cuerpo antes de timeout.
- La instancia `8787` responde, pero no se usa para OPES porque puede
  pertenecer al agente que está programando el núcleo de Orquesta.
- No se ha tocado código ni ramas de Orquesta desde OPES.

Impacto:

- OPES no puede lanzar con seguridad nuevos temas sobre `18793`.
- Si se usa otra instancia, debe ser una instancia OPES aislada con runtime y
  state propios, no el servidor de otro agente.

Tarea adicional para agente de Orquesta:

- Evitar que una instancia quede escuchando pero sin responder a rutas básicas
  tras `runs/control stop`.
- Añadir health/readiness con diagnóstico: `http_server_accepting_but_handlers_blocked`,
  `state_store_blocked`, `supervisor_lock_blocked` o equivalente.
- Proveer un cierre seguro por API que no deje el proceso HTTP bloqueado.
