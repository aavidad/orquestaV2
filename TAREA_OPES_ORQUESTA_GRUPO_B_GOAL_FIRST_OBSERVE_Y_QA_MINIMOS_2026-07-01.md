# Tarea Orquesta - Grupo B Informática: observación goal-first y QA de mínimos

Fecha: 2026-07-01

## Contexto

OPES lanzó una ola P0 de rework del temario `Grupo B Informática` desde:

`/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional`

Servidor Orquesta:

- URL: `http://127.0.0.1:8787`
- `ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux`
- runtime `orquesta-server`
- `runtime_identity.binary_sha256=c6a4f250e49909d7890cb0d872097d5fcc996b6dfd5e8b184b32c9a1bd66a0ce`

Runs P0:

- `run-opes-grupo-b-info-rework-p0-t006-20260701`
- `run-opes-grupo-b-info-rework-p0-t010-20260701`
- `run-opes-grupo-b-info-rework-p0-t011-20260701`
- `run-opes-grupo-b-info-rework-p0-t012-20260701`
- `run-opes-grupo-b-info-rework-p0-t018-20260701`
- `run-opes-grupo-b-info-rework-p0-t019-20260701`

## Fallos observados

### 1. Observe bloqueante sin snapshot útil suficiente

Las llamadas a:

```bash
POST /api/v0/apps/director/goal/observe
```

con `run_ref` de los seis runs devolvieron:

- `504` sin campos útiles en algunos casos;
- timeout del cliente `curl` en otros;
- sin `goal_status`, `run_status`, `closure_status`, `partial` ni
  `recommended_action` en la salida consumible por OPES.

Impacto: OPES no puede distinguir de forma fiable entre goal vivo, goal
colgado, goal lento, goal sin entrega o goal que necesita replanificación.

Acción esperada:

- `observe` debe devolver siempre snapshot parcial rápido cuando agote ventana
  HTTP, con `partial=true`, estado conocido, última actividad, proceso asociado
  y acción recomendada.
- Si no puede observar, debe exponer causa operacional clara, no solo timeout.

### 2. Goals activos sin cierre ni artefactos contractuales completos

La base `goals_1.sqlite` mostraba goals `active` con consumo alto de tokens.
Durante la ventana revisada solo aparecieron artefactos parciales:

- `tema_011/trabajo/tema_011_ampliado_limpio.md`
- `tema_018/trabajo/tema_018_ampliado_limpio.md`
- `tema_010/trabajo/tema_010_ampliado_limpio.md`
- `tema_012/trabajo/tema_012_ampliado_limpio.md`
- `tema_019/trabajo/tema_019_ampliado_limpio.md`
- `tema_010/visuales/tema_010_igualdad_genero_transversalidad.webp`

No aparecieron informes de cierre por tema, matrices de reutilización,
checkpoints de subroles, QA de mínimos o estado terminal por tema.

Acción esperada:

- El runner de OPES/Orquesta debe materializar estado operacional por tema:
  `working`, `waiting`, `needs_rework`, `blocked`, `complete`.
- Debe haber heartbeat o checkpoint durable aunque el goal siga activo.

### 2.b. Resultado durable sin estado terminal

Después aparecieron ficheros `trabajo/docs/orquesta_goal_result_v0.json` en
temas 006, 012 y 018. Sin embargo, `goals_1.sqlite` solo mostraba `complete`
para el tema 012; temas 006 y 018 seguían `active` pese a tener resultado
durable.

Acción esperada:

- Si existe `orquesta_goal_result_v0.json` válido y el cierre externo lo acepta,
  el estado durable del goal debe pasar a terminal o publicar una razón clara
  por la que sigue `active`.
- Si el resultado está incompleto, `observe` debe exponer qué falta, no seguir
  consumiendo tokens sin explicación.

### 2.c. Tests `passed` sin evidencias

En `orquesta_goal_result_v0.json` de temas 012 y 018 hay
`required_test_results` con `status=passed` y `evidence_refs=[]`.

Acción esperada:

- Un required test no debe marcar `passed` sin evidencia durable.
- Si la evidencia vive en el filesystem, debe referenciarse con ruta relativa
  clara dentro del write-set.

### 3. No se bloquea entrega por debajo del mínimo OPES

La entrega nueva del tema 011 quedó primero en 6.051 palabras y después en
10.206 palabras. Para nivel B el mínimo del ampliado publicable es 10.800
palabras.

Además:

- tema 010: 10.795 palabras, por debajo del mínimo;
- tema 012: 9.301 palabras, por debajo del mínimo;
- tema 012 declara en `validacion/informe_calidad_tema.md`:
  `El texto ampliado limpio alcanza 9301 palabras, supera el mínimo de nivel B de 10.800 palabras`.

Esto es un falso verde crítico: 9.301 no supera 10.800. Parece un error de
comparación o de interpretación del punto de millar en español.

Acción esperada:

- En trabajos OPES de temario, el contrato del runner debe ejecutar o exigir
  validación de extensión antes de aceptar un artefacto como candidato.
- Si no alcanza mínimo, el estado debe ser `pendiente_continuar` o
  `needs_expansion_min_words_B`, nunca cierre aceptado.
- Los mínimos deben compararse como enteros en palabras, no como números con
  separador decimal/millar ambiguo.
- Ningún informe generado por agente puede sustituir al validador canónico de
  extensión.

### 4. No se bloquea texto visible con metacomentarios de alcance

El tema 018 alcanzó 11.407 palabras, pero conserva metacomentarios visibles
como `Para el nivel de este tema no se exige`, `Para este tema basta`,
`Lo evaluable en este tema` y `En examen`.

Otros ejemplos observados en la ola P0:

- tema 006: `En examen se debe recordar`, `En examen hay que explicar`,
  `debe estudiarse`;
- tema 010: `Nota de test`, `En examen`;
- tema 019: `En examen conviene`, `Para examen no se exige`.

Los informes locales de texto limpio no los detectaron y declararon `pass`.

Acción esperada:

- El contrato OPES debe pasar una QA textual antes de aceptar un tema:
  metacomentarios de alcance, estudio, agente, autor, proceso o examen deben
  moverse a tutor/notas internas o reescribirse como contenido didáctico.
- El validador de texto público debe ampliar patrones o permitir una política
  OPES específica para detectar frases de estrategia de examen dentro del texto
  publicable, aunque no contengan la cadena `Nota del autor`.

### 5. Visual raster pero no didáctico

El tema 010 generó un WebP visualmente limpio, pero de carácter decorativo:
reunión con iconos genéricos. Para un tema jurídico de igualdad se necesitaba
infografía de estructura normativa, transversalidad, obligaciones, planes,
conceptos y relaciones evaluables.

Acción esperada:

- El contrato visual OPES no debe aceptar `raster=true` como suficiente.
- Debe exigir función didáctica, anclaje a apartado y rechazo de visual
  decorativo.

### 6. Desalineación cola `stopped/blocked` frente a goals Codex `active`

En la ola correctiva P0-clean lanzada el 2026-07-01 a las 09:58 CEST, la API
de estado devolvió `queue.count=0` y seis runs terminales `status=stopped`,
pero a la vez `queue_health.blocked=6` y `stale_running[].code=goal_first_blocked`
para:

- `run-opes-grupo-b-info-p0-clean-t006-20260701`
- `run-opes-grupo-b-info-p0-clean-t010-20260701`
- `run-opes-grupo-b-info-p0-clean-t011-20260701`
- `run-opes-grupo-b-info-p0-clean-t012-20260701`
- `run-opes-grupo-b-info-p0-clean-t018-20260701`
- `run-opes-grupo-b-info-p0-clean-t019-20260701`

El motivo expuesto fue:

`goal-first state persisted as blocked or invalid`

Sin embargo, en paralelo, la base de goals de Codex seguía mostrando esos seis
goals como `active`, con tokens y `updated_at` creciendo:

- tema 006: `active`, 300.350 tokens, actualizado `2026-07-01 07:58:03Z`
- tema 010: `active`, 144.885 tokens, actualizado `2026-07-01 07:57:56Z`
- tema 011: `active`, 210.296 tokens, actualizado `2026-07-01 07:57:01Z`
- tema 012: `active`, 248.541 tokens, actualizado `2026-07-01 07:57:53Z`
- tema 018: `active`, 120.267 tokens, actualizado `2026-07-01 07:57:50Z`
- tema 019: `active`, 129.012 tokens, actualizado `2026-07-01 07:57:19Z`

Además, `POST /api/v0/autoprogramming/goals/observe-active` respondió `ok`,
pero no reconcilió el estado ni devolvió un snapshot útil por goal.

Impacto:

- OPES no puede saber si debe esperar, relanzar, cortar o consolidar desde
  artefactos.
- La cola puede decir `stopped/blocked` mientras el backend de Codex sigue
  trabajando y escribiendo ficheros.
- Riesgo de lanzar una ola encima de goals vivos o de cerrar antes de que el
  trabajo haya terminado realmente.

Acción esperada:

- Unificar el estado observable entre cola, `autoprogramming/status`,
  `observe-active` y `goals_1.sqlite`.
- Si un goal-first queda bloqueado, exponer por run: `goal_id`, estado del
  backend, última actividad, tokens, proceso, artefactos recientes y acción
  segura.
- `observe-active` debe devolver snapshot por goal, no solo `estado=ok`.
- Si la cola fuerza `stopped` por timeout, no debe ocultar que el goal Codex
  continúa `active`.

### 7. Goal sigue `active` tras entrega durable y texto estable

En la misma ola P0-clean, el tema 011 quedó con artefactos de entrega escritos:

- `trabajo/docs/orquesta_goal_result_v0.json`
- `trabajo/opes_topic_rework_delivery.json`
- `trabajo/opes_topic_rework_delivery.md`
- `validacion/informe_extension_temario.json`
- `validacion/informe_texto_publico_sin_meta_examen.json`
- `validacion/informe_texto_publico_sin_meta_examen_publicables_tema_011.json`

El texto publicable permaneció estable entre dos comprobaciones:

- `tema_011_ampliado_limpio.md`: SHA256
  `e97f5c63c91416b64f93e2ad2768f20599a533f0071f17c5320e29a26b88c38e`
- `tema_011_resumen_limpio.md`: SHA256
  `7017758d58385eab28d33cd10f85243ed985482cbd40e9750aea6f69df7e514b`
- recuento ampliado: 11.507 palabras.
- validadores de limpieza: `pass`, 0 hallazgos.

Pese a ello, `goals_1.sqlite` seguía mostrando:

`0f370215-fe7f-4745-bfff-c9ada2c0dd96|active|325523|1422|2026-07-01 08:12:57`

Además, el goal siguió consumiendo tokens y reescribiendo informes ya existentes
sin cambiar el texto.

Acción esperada:

- Si el texto está estable, las validaciones pasan y existe resultado durable,
  Orquesta debe cerrar el goal o pedir un rework concreto.
- Evitar bucles de cierre que solo reescriben informes equivalentes.
- Exponer una razón de continuidad si el goal no puede cerrarse: campo faltante,
  validador pendiente, contrato inválido o archivo rechazado.

### 8. `orquesta_goal_result_v0.json` sin contrato mínimo

En el tema 011, `trabajo/docs/orquesta_goal_result_v0.json` contiene resumen,
artefactos y `required_test_results`, pero no incluye campos mínimos:

- `schema_version`: `null`
- `status`: `null`
- `estado`: `null`

Impacto: OPES no puede decidir de forma segura si el resultado es terminal,
preliminar, inválido o solo un handoff textual.

Acción esperada:

- Todo `orquesta_goal_result_v0.json` debe incluir versión de esquema y estado
  terminal explícito.
- Si falta el estado, el validador de Orquesta debe rechazarlo y devolver error
  actionable antes de seguir consumiendo tokens.

### 9. API conserva `blocked/stale_running` después de goals `complete`

Después de que los seis goals P0-clean pasaran a `complete` en
`goals_1.sqlite`, `POST /api/v0/autoprogramming/status` seguía devolviendo:

- `queue.count=0`
- `queue_health={"blocked":6,"observed_runs":6}`
- `stale_running=6`
- los seis runs terminales como `status=stopped`

La base durable de Codex mostraba en cambio:

- tema 006: `complete`
- tema 010: `complete`
- tema 011: `complete`
- tema 012: `complete`
- tema 018: `complete`
- tema 019: `complete`

Impacto: el dashboard/API de Orquesta no refleja el cierre real del backend y
mantiene incidencias stale aunque el trabajo haya terminado.

Acción esperada:

- Reconciliar `autoprogramming/status` con `goals_1.sqlite` cuando los goals
  pasen a `complete`.
- Limpiar `stale_running` o transformarlo en historial resuelto.
- Exponer `goal_status=complete` en cada run terminal, no solo `stopped`.

### 10. `stale_running` prematuro en ola 2 sin checkpoint de disco

Tras arrancar un servidor limpio para la ola 2 (`binary_sha256`:
`eb040f8622a21b792c1ab1cf62b6b1f07f9e9e5762d6b5c8fcc78470ae48d334`) y lanzar
seis runs nuevos:

- `run-opes-grupo-b-info-wave2-t021-20260701`
- `run-opes-grupo-b-info-wave2-t022-20260701`
- `run-opes-grupo-b-info-wave2-t023-20260701`
- `run-opes-grupo-b-info-wave2-t024-20260701`
- `run-opes-grupo-b-info-wave2-t028-20260701`
- `run-opes-grupo-b-info-wave2-t036-20260701`

la API pasó de `running_live=6, stale=0` a `blocked=6, stale_running=6` en
pocos minutos, aunque `goals_1.sqlite` seguía mostrando los seis goals como
`active`, con tokens crecientes:

- tema 021: `active`, 243.831 tokens, actualizado `2026-07-01 08:25:22Z`
- tema 022: `active`, 216.569 tokens, actualizado `2026-07-01 08:25:11Z`
- tema 023: `active`, 242.589 tokens, actualizado `2026-07-01 08:24:55Z`
- tema 024: `active`, 387.293 tokens, actualizado `2026-07-01 08:25:21Z`
- tema 028: `active`, 151.884 tokens, actualizado `2026-07-01 08:24:26Z`
- tema 036: `active`, 334.527 tokens, actualizado `2026-07-01 08:25:15Z`

En ese momento no había ficheros nuevos en los write-set de los temas; solo los
inventarios previos. `POST /api/v0/autoprogramming/goals/observe-active` volvió
a responder únicamente `estado=ok`, sin snapshot por goal ni motivo de stale.

Impacto:

- El operador no puede distinguir si los agentes están leyendo/redactando, si
  están en bucle interno o si el runner ha perdido la observación.
- La cola declara stale antes de tener un checkpoint materializado que permita
  valorar progreso o replanificación.

Acción esperada:

- No declarar stale/blocked solo por ausencia inicial de ficheros si el backend
  muestra actividad reciente y tokens crecientes, o al menos exponerlo como
  `active_no_checkpoint_yet` con ventana temporal configurable.
- Exigir heartbeat materializado por goal (`trabajo/checkpoint_director.md` o
  equivalente) cada N minutos en trabajos OPES largos.
- `observe-active` debe devolver por cada goal: tokens, updated_at, edad del
  último fichero escrito, estado de checkpoint y acción recomendada.

### 11. `runs/control stop forced` no controla goals backend

Al detectar ola 2 sin checkpoints ni ficheros nuevos, OPES intentó parar los
seis runs por:

`POST /api/v0/runs/control`

con `action=stop` y `forced=true`.

Resultado HTTP observado:

- varias llamadas terminaron en `504`;
- varias llamadas agotaron timeout de cliente sin cuerpo;
- después, `autoprogramming/status` decía que los seis runs estaban
  `terminal/stopped`;
- pero `goals_1.sqlite` seguía mostrando los seis goals como `active`.

Impacto:

- El operador puede creer que los runs están parados mientras el backend Codex
  sigue activo.
- No hay confirmación causal de que `runs/control` haya detenido el goal-first.

Acción esperada:

- `runs/control stop/cancel` debe propagarse al backend goal-first o devolver
  explícitamente `control_not_propagated_to_goal_backend`.
- La respuesta debe incluir `run_ref`, `goal_id`, estado previo, estado final y
  si el proceso/hilo Codex recibió la señal.
- No se debe marcar `stopped` en la API pública si `goals_1.sqlite` continúa
  `active`.

### 12. `server/shutdown` con timeout mientras backend sigue y completa

Después de los fallos de `runs/control`, OPES pidió:

`POST /api/v0/server/shutdown`

El cliente agotó timeout sin cuerpo y el proceso `orquesta-server` terminó al
recibir interrupción con:

`orquesta-server: orquesta_server: async_work_timeout`

Sin embargo, el `codex app-server` del backend goal-first quedó vivo y continuó
trabajando. Tras la caída del servidor Orquesta, los agentes de ola 2
materializaron entregas completas para los seis temas 021, 022, 023, 024, 028
y 036; después `goals_1.sqlite` mostró los seis como `complete`.

Impacto:

- Orquesta servidor puede morir o quedar fuera de control mientras el backend
  sigue escribiendo artefactos.
- El operador pierde API de observación aunque los goals sigan produciendo.
- Hay que cerrar manualmente el app-server residual cuando todos los goals
  terminan.

Acción esperada:

- `server/shutdown` debe coordinar explícitamente el backend goal-first:
  checkpoint, stop/cancel o wait controlado, y devolver estado por goal.
- Si el servidor termina por timeout, debe dejar un fichero de estado claro con
  `backend_still_running`, socket, PID y objetivos vivos.
- Al cerrar, Orquesta debe limpiar o delegar de forma explícita el app-server;
  no dejar procesos residuales invisibles para el operador.

### 13. Ola 3: producción útil pero bucle de reescritura tardía y estados activos obsoletos

Con servidor limpio y binario nuevo:

- `runtime_identity.binary_sha256=e17f4fd632db71e0412e09277ff29e0ed5301b0bdf9aa321fb74905845f804b7`
- temas lanzados: 037, 038, 041, 042, 045 y 047.
- payloads:
  `/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_payloads/wave3_text_html_rag_tests/`

Resultado positivo: Orquesta sí produjo artefactos útiles en los seis temas:
texto ampliado/resumen, HTML final/ampliado, RAG canónico, tests y planes
visuales. El informe OPES quedó en:

`/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/09_validacion/informe_qa_wave3_grupo_b_informatica_2026-07-01.md`

Problemas observados:

- Temas 038, 041 y 047 siguieron reescribiendo artefactos después de haber
  materializado entregas suficientes.
- El consumo registrado en `state_5.sqlite` subió a varios millones de tokens
  por hilo, aunque los artefactos de salida estaban ya presentes.
- OPES tuvo que cortar manualmente para evitar seguir quemando cuota.
- Tras matar servidor y app-server, `goals_1.sqlite` todavía dejó 038 y 047
  como `active`; no había procesos vivos.
- `POST /api/v0/server/shutdown` devolvió HTTP 400 sin cuerpo JSON útil.
- `GET /api/v0/external-work/observe` devolvió 404 en este binario, por lo que
  no había observación fina por `external_work`.

Acción esperada:

- Añadir criterio nativo de cierre para OPES: si ya existen texto, HTML,
  RAG, tests, validadores y `orquesta_goal_result`, el goal debe cerrar o
  declarar exactamente qué falta.
- Evitar bucles de reescritura tardía: si los ficheros clave existen y no cambia
  su contenido sustancial, no seguir generando variantes sin una tarea de rework
  concreta.
- Persistir estado terminal coherente en `goals_1.sqlite` o en el puente
  goal-first cuando el proceso se corta después de entrega durable.
- `server/shutdown` debe devolver JSON operacional incluso si rechaza la
  petición; HTTP 400 sin explicación no sirve para operación.
- Exponer un endpoint de observación estable para `external-work/run`
  goal-first o documentar el endpoint correcto en la respuesta de lanzamiento.

### 14. Ola 3: mínimos OPES deben usar contador estricto, no `wc` como verde único

En la ola 3, dos temas parecían cumplir si se miraba solo `wc -w`, pero fallaban
con el contador estricto usado por OPES:

- tema 038: `wc=10800`, contador estricto OPES `10336`, debe quedar
  `needs_expansion_min_words_B`.
- tema 041: `wc=10983`, contador estricto OPES `10531`, debe quedar
  `needs_expansion_min_words_B`.

Además, el propio informe de tema 041 decía `status=fail`, pero el texto del
informe indicaba que superaba 10.800 palabras por "conteo interno", creando una
contradicción.

Acción esperada:

- El contrato OPES debe usar un único contador canónico, el de
  `validate_extension_temario_opes.py` o equivalente, y no aceptar `wc -w` como
  prueba suficiente.
- Si un agente genera un informe contradictorio (`status=fail` pero texto dice
  que pasa), Orquesta debe marcarlo como `invalid_report_contract` y pedir
  rework focal, no cerrar.
- La aceptación del tema debe bloquearse si el contador estricto no llega al
  mínimo del nivel.

### 15. Ola 4: falta introspección de rutas y estado operacional confuso con goals activos

En la ola 4 se arrancó un servidor nuevo para corregir solo los temas cortos
011, 012, 038 y 041:

- `runtime_identity.binary_sha256=b7d61a0eba0e73ee5899669a55450ae62ea0f776641c1bf33a60e969b92b8990`
- `ORQUESTA_SERVER_STATE_DIR=/home/alberto/Trabajo/orquesta/.orquesta-runtime/grupo-b-informatica-wave4-20260701/state`
- payloads:
  `/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_payloads/wave4_expand_strict_min/`

Los cuatro `POST /api/v0/external-work/run` fueron aceptados con
`estado=ok`, `route_policy=goal_first` y `next_actions=["observe_active_goals"]`.

Problemas observados:

- `GET /api/v0/routes` devolvió 404. El operador no tiene forma simple de
  descubrir endpoints disponibles ni el endpoint canónico de observación.
- `GET /api/v0/external-work/observe` ya había devuelto 404 en la ola 3; la
  respuesta de lanzamiento solo dice `observe_active_goals`, pero no indica URL
  ni contrato estable para hacerlo.
- El `supervisor_tick_result` siguió mostrando `public_stop_reason=idle_no_execution`
  y `queue_size=0` mientras `goals_1.sqlite` tenía cuatro goals OPES `active`.
  Esto es ambiguo: la cola está vacía, pero el trabajo real sigue vivo en
  goal-first.
- El operador debe combinar a mano `goals_1.sqlite`, audit log, `find -mmin` y
  procesos para saber si algo está vivo, colgado o simplemente fuera de cola.
- Tras unos minutos, los cuatro ficheros
  `orchestration-state/app_director_goal_states/*.json` pasaron a
  `status=blocked` con `summary=codex_app_server_goal_active_timeout` y
  `last_closure.issues[0].code=goal_closure_invalid`, pero el backend
  `goals_1.sqlite` seguía marcando los mismos `external_goal_ref` como
  `active` y aumentando `tokens_used`.
- Mientras se produjo ese bloqueo de Orquesta no aparecieron ficheros nuevos en
  los write-set de los temas 011, 012, 038 y 041; el operador no puede saber si
  debe esperar, reintentar o cancelar sin inspección manual.

Acción esperada:

- Exponer un endpoint de introspección estable, por ejemplo `/api/v0/routes` o
  `/api/v0/capabilities`, con rutas operativas y contratos mínimos.
- La respuesta de `external-work/run` debe incluir el endpoint de observación
  real que corresponde a `next_actions=["observe_active_goals"]`.
- El estado del supervisor debe distinguir `queue_idle_but_goal_backend_active`
  de `idle_no_execution` cuando hay goals activos en app-server.
- Añadir una vista operacional única que una cola Orquesta, runs, goal-first,
  PID/socket del app-server y artefactos recientes por run.
- Si Orquesta marca `codex_app_server_goal_active_timeout`, debe cancelar o
  pausar explícitamente el goal del backend, o dejar el estado como
  `backend_active_after_orquesta_blocked` con instrucciones de control. No debe
  quedar simultáneamente `blocked` en Orquesta y `active` consumiendo tokens en
  app-server.

Control intentado:

- `POST /api/v0/runs/control action=stop forced=true` sobre
  `run-opes-grupo-b-info-wave4-strict-t011-20260701` devolvió `estado=ok`,
  `status=stop_requested`, `goal_status_before=blocked`,
  `goal_status_after=blocked` y `checkpoint_recorded=true`.
- La misma llamada sobre `run-opes-grupo-b-info-wave4-strict-t012-20260701`,
  `run-opes-grupo-b-info-wave4-strict-t038-20260701` y
  `run-opes-grupo-b-info-wave4-strict-t041-20260701` devolvió
  `estado=error`, `code=run_control_timeout`,
  `message="control de run excedio la ventana HTTP acotada"`.

Acción esperada adicional:

- `runs/control` debe ser idempotente y barato sobre runs ya `blocked`; no debe
  agotar timeout HTTP al intentar cortar tres runs bloqueados sin artefactos.
- Si el stop no llega al backend goal-first, devolver
  `control_not_propagated_to_goal_backend` con estado del goal externo, no un
  timeout genérico que obliga al operador a matar procesos.

Efecto posterior observado:

- Aunque Orquesta ya había marcado los cuatro runs como `blocked`, y aunque se
  pidió `stop` al menos para el tema 011, el backend goal-first siguió activo y
  materializó artefactos útiles después: tema 038 regeneró HTML, RAG, tests e
  informes wave4; tema 041 regeneró HTML, RAG y tests; tema 011 regeneró tests
  y HTML ampliado.
- El tema 012 permaneció sin cambios de artefactos y por debajo del mínimo,
  pero su goal seguía `active` en `goals_1.sqlite`.

Acción esperada adicional:

- El estado público debe admitir un estado intermedio tipo
  `blocked_but_backend_still_producing` o reconciliar automáticamente a
  `running` cuando aparecen artefactos nuevos después del timeout.
- `stop_requested` no puede coexistir silenciosamente con escritura posterior
  de artefactos sin dejar trazabilidad: debe quedar como `stop_not_propagated`
  o `backend_continued_after_stop`.
- Si un goal no escribe artefactos ni aumenta `updated_at` tras un timeout,
  Orquesta debe separarlo de los goals que sí siguen produciendo y permitir
  cancelar solo ese hilo.
- Los agentes de contenido no deben dejar caches de ejecución como
  `__pycache__/` dentro de carpetas de temario. Si se ejecutan scripts de apoyo,
  las caches deben vivir fuera del paquete OPES o limpiarse antes del cierre.

Resolución operativa OPES de la ola 4:

- Los temas 011, 038 y 041 terminaron materializando artefactos wave4 válidos.
  Los temas 038 y 041 pasaron a `complete` en `goals_1.sqlite`; el tema 011
  dejó `trabajo/docs/orquesta_goal_result_v0.json` wave4 `complete`, pero
  siguió `active` en el backend durante varios minutos.
- El tema 012 reescribió el ampliado hasta pasar el contador estricto
  (`11436` palabras), pero no generó `informe_extension_estricto_wave4`,
  validadores wave4, tests `tema_012_tests_rework.json` ni JSON de cierre
  wave4. Su backend quedó `active` sin escribir artefactos adicionales.
- OPES decidió cortar la instancia antigua y relanzar una instancia nueva desde
  el commit posterior de Orquesta (`0b3e9966 Cierra BUG-068 discovery
  goal-first`) solo para el cierre derivado de 012.

Acción esperada adicional:

- Orquesta debe cerrar como terminal un goal que ya escribió
  `orquesta_goal_result_v0.json` válido, o explicar por qué lo mantiene activo.
- Si un goal consigue el texto mínimo pero no ejecuta derivados obligatorios,
  debe materializar una tarea causal de cierre derivado en vez de quedar activo
  sin artefactos.
- El shutdown forzado con `idempotency_key` devolvió `estado=ok`,
  `shutdown_ready=true`, `agents_in_flight=0`, pero dejó vivos procesos
  `codex app-server` asociados a
  `g-fe4c6324af0f5efc.sock`. OPES tuvo que matar esos procesos residuales de
  forma manual antes de relanzar Orquesta.

Acción esperada adicional:

- `shutdown_ready=true` no debe emitirse si queda vivo el backend app-server
  propio de la instancia, o debe devolver explícitamente
  `backend_still_running` con PIDs/refs saneadas y acción recomendada.

### 16. Ola 5: versión nueva mejora discovery, pero goal queda `usage_limited` antes de cierre

Tras relanzar desde `0b3e9966 Cierra BUG-068 discovery goal-first`:

- `runtime_identity.binary_sha256=cbbb643b09039793c5359248afba10deaf16c00ff625c8059733bc1bb3a68bd4`
- `GET /api/v0/routes` ya devolvió `orquesta_route_manifest.v0`.
- `POST /api/v0/external-work/run` para el cierre de 012 devolvió
  `operation_endpoints.observe_active_goals=/api/v0/autoprogramming/goals/observe-active`.

Resultado positivo:

- El endpoint de discovery ya está operativo.
- La respuesta del launch ya dice qué endpoint usar para observar goals.
- El goal wave5 de 012 materializó artefactos útiles: tests canónicos
  `tests/tema_012_tests_rework.json`, HTML y RAG actualizados.

Problema observado:

- El backend `goals_1.sqlite` marcó el goal wave5 de 012 como `usage_limited`
  con `tokens_used=0` y `time_used_seconds=1`, aunque había artefactos
  materializados.
- No generó los validadores wave5 ni actualizó
  `trabajo/docs/orquesta_goal_result_v0.json`; quedó el JSON antiguo P0-clean.
- `POST /api/v0/autoprogramming/goals/observe-active` devolvió solo
  `{"estado":"ok"}`, sin snapshot operativo de ese goal ni explicación de
  `usage_limited`.

Acción esperada:

- `usage_limited` debe incluir causa concreta: presupuesto agotado, cuota,
  límite del goal, límite del backend o política del servidor.
- Si hay artefactos materializados antes de `usage_limited`, Orquesta debe
  reconciliar required tests ya pasables o lanzar automáticamente un cierre
  determinista sin LLM para validadores locales.
- `observe-active` debe devolver al menos lista de goals observados, estado,
  updated_at, tokens, último issue, artefactos recientes y next action; un
  `{"estado":"ok"}` vacío no sirve para operar.

## No se ha tocado código

Esta incidencia solo documenta el problema para el agente de Orquesta. OPES no
ha modificado el núcleo.

## Prioridad

Alta para autonomía OPES: sin observe parcial útil, estados por tema y guardas
de mínimos, Orquesta puede aparentar trabajo activo mientras no permite al
director saber si debe esperar, replanificar o cortar.

### 17. Ola 6: `usage_limited` inmediato en seis goals con cero tokens y sin artefactos

Fecha observada: 2026-07-01 12:13 Europe/Madrid.

Contexto:

- Orquesta arrancó desde `0b3e9966 Cierra BUG-068 discovery goal-first`.
- `runtime_identity.binary_sha256` esperado de esta rama: `cbbb643b09039793c5359248afba10deaf16c00ff625c8059733bc1bb3a68bd4`.
- `GET /api/v0/routes` respondió correctamente con
  `schema_version=orquesta_route_manifest.v0`.
- Se lanzaron seis `POST /api/v0/external-work/run` para la wave6 de Grupo B
  Informática: temas `001`, `002`, `003`, `004`, `005` y `007`.
- Todas las respuestas devolvieron `estado=ok`, `route_policy=goal_first`,
  `external_goal_ref`, `evidence-ref-codex-app-server-goal-set` y
  `next_actions=["observe_active_goals"]`.

Problema:

- `POST /api/v0/autoprogramming/goals/observe-active` devolvió solo:

```json
{"estado":"ok"}
```

- En `goals_1.sqlite`, los seis goals quedaron terminales como
  `usage_limited`, con `tokens_used=0` y `time_used_seconds` entre 1 y 9
  segundos.
- No se materializaron artefactos nuevos en los directorios de los temas
  `001`, `002`, `003`, `004`, `005` ni `007`: solo permanecen los inventarios
  de entrada previos.
- El `codex app-server` seguía vivo tras esos estados terminales, por lo que no
  era un fallo visible de proceso muerto antes de crear goals.
- En los ficheros internos
  `orchestration-state/app_director_goal_states/*.json`, Orquesta sí guarda
  `state.last_result.summary=codex_app_server_goal_status_usageLimited`.
- La misma estructura deja `state.last_closure.status=blocked` con
  `state.last_closure.issues=[{"code":"goal_closure_invalid","field":"status"}]`.
- `run-state/queue_v0.json` marca los runs como `status=stopped` y añade
  `evidence-ref-goal-first-queue-terminal-sync`, pero el observador residente
  sigue reportando `terminal=0` e `issues=0`.
- El shutdown posterior devolvió `shutdown_ready=true` y esta vez no dejó
  procesos residuales `orquesta-server` ni `codex app-server`.

Evidencia local:

```text
opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_responses/wave6_common_topics_text_html_rag_tests/
  response_t001.json
  response_t002.json
  response_t003.json
  response_t004.json
  response_t005.json
  response_t007.json
  observe_active_initial.json
```

Consulta observada:

```sql
select thread_id,status,tokens_used,time_used_seconds,datetime(created_at_ms/1000,'unixepoch'),datetime(updated_at_ms/1000,'unixepoch'),objective
from thread_goals
where thread_id in (...wave6 external_goal_ref...);
```

Resultado resumido:

```text
001 usage_limited tokens=0 time=2s
002 usage_limited tokens=0 time=2s
003 usage_limited tokens=0 time=9s
004 usage_limited tokens=0 time=1s
005 usage_limited tokens=0 time=1s
007 usage_limited tokens=0 time=1s
```

Acción esperada:

- Si hay límite real de cuota/presupuesto, Orquesta debe devolver causa
  explícita y estado operable: `quota_limited`, `budget_limited`,
  `provider_limited`, `policy_limited` o equivalente, con recomendación de
  esperar, relanzar o reducir lote.
- `usage_limited` con `tokens_used=0` y sin artefactos no debe parecer un
  cierre normal de goal; debe generar tarea causal o error de lanzamiento.
- `observe-active` debe devolver también goals terminales recientes o, al
  menos, explicar que los refs solicitados no están activos e incluir su estado
  terminal, updated_at, causa y next action. Un `{"estado":"ok"}` vacío no
  permite operar.
- La API pública debe exponer el detalle que ya existe internamente:
  `codex_app_server_goal_status_usageLimited`,
  `goal_closure_invalid/status` y `queue_terminal_sync`.
- Si el backend ya no puede consumir tokens, `external-work/run` debería
  rechazar/encolar explícitamente en vez de devolver `goal-set` como si el
  trabajo hubiera arrancado.

Nota de dominio OPES:

- Los títulos de la wave6 coinciden con la matriz de rework de Grupo B
  Informática (`01_matrices/matriz_rework_inicial.md`). No confundirlos con el
  árbol genérico `materias-comunes-grupo-b`, que tiene otro programa común.

No se ha tocado código del núcleo de Orquesta. Esta sección queda para el
agente que está corrigiendo Orquesta.

#### Seguimiento 2026-07-02

OPES terminó el cierre textual estricto de Grupo B Informática mediante fallback
local documentado, sin editar núcleo de Orquesta. Se reconstruyeron los temas
con contaminación estructural y se ampliaron los que no llegaban al mínimo B con
scripts reproducibles:

```text
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/scripts/rebuild_remaining_structural_topics.py
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/scripts/expand_short_structural_topics.py
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/scripts/cleanup_remaining_text_qa.py
```

Resultado actual del informe estricto:

```text
extension_pass=50/50
official_text_qa_pass=50/50
strict_text_qa_pass=50/50
strict_scaffolding_finding_count=0
```

Evidencia:

```text
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/09_validacion/informe_avance_estricto_grupo_b_informatica_2026-07-01.md
```

La incidencia no se cierra para Orquesta: confirma que el director autonomo debe
detectar contaminacion estructural, replanificar expansion propia del titulo y
no depender de limpieza manual externa.

### 19. BUG-ORQ-20260702-095: QA OPES no detecta todas las variantes de andamiaje de estudio en texto publicable

Fecha observada: 2026-07-02 00:50 Europe/Madrid.

Contexto:

- Durante el cierre directo de Grupo B Informática, tras el fallo operativo de
  Orquesta ya documentado, se revisaron temas con extensión suficiente y QA
  oficial limpia.
- En los temas `029` y `030` aparecieron bloques visibles de estudio:
  `Preguntas De Recuperacion`, `Preguntas de recuperacion`, `Repaso Espaciado`,
  `Día 0`, `mapa mental`, `La respuesta debe empezar` y
  `Una respuesta fuerte empieza`.
- El informe estricto/global no bloqueaba todas las variantes, o solo marcaba
  una parte, por diferencias de mayúsculas/minúsculas, tildes y formulación.

Problema:

- El contrato de cierre OPES puede producir falsos verdes si el texto
  publicable conserva andamiaje de estudio, estrategias de respuesta o
  calendarios.
- Esto afecta a la calidad del tema aunque el conteo de palabras, metanotas y
  notas de autor estén en `pass`.

Evidencia local:

```text
OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/02_temas/tema_029/02_markdown/tema_ampliado.md
OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/02_temas/tema_030/02_markdown/tema_ampliado.md
OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/02_temas/tema_030/02_markdown/tema_resumen.md
```

Acción esperada:

- Ampliar `validate_public_text_no_study_scaffolding.py` con normalización
  `casefold` y sin tildes.
- Bloquear encabezados y frases equivalentes a preguntas de recuperación,
  repaso espaciado, mapas mentales, calendarios de estudio, plantillas de
  respuesta y consejos de estrategia de examen.
- Orquesta debe ejecutar este gate antes de `ready` y crear rework por tema si
  falla.
- No se ha tocado código del núcleo de Orquesta. Esta sección queda para el
  agente que está corrigiendo Orquesta.

### 34. BUG-ORQ-20260701-093: QA OPES acepta falsos verdes si solo mira extensión y metanotas, pero no andamiaje interno ni mezcla de temas

Fecha observada: 2026-07-01.

Nota de inventario: esta incidencia se consolida como
`BUG-ORQ-20260701-093` en el inventario central de Orquesta.

Contexto:

- En el rework de Grupo B Informática, varios temas parecían válidos por
  extensión y por los validadores existentes
  `validate_public_text_no_exam_meta.py` y
  `validate_public_text_no_author_notes.py`.
- La lectura humana y tres subagentes detectaron que muchos textos seguían
  conteniendo material no publicable: mapas mentales de estudio, calendarios
  `Día 0`, frases `reconstruye sin mirar`, referencias a SVG/HTML, trazabilidad
  B, derivaciones futuras, fuentes internas, bloques de canon/maestro y bloques
  completos de otros temas.
- Ejemplos:
  - Tema 020: el título es ofimática, pero el texto arrancaba como software
    libre/GNU y la ofimática real llegaba tarde.
  - Temas 016/017: almacenamiento/SAN/NAS/RAID contaminados con HPC/grid,
    visuales SVG y bloques de estudio.
  - Temas 025-031: bases de datos/redes con anexos o duplicados de temas
    vecinos.
  - Temas 043/044/046/049/050: extensión conseguida por acumulación de bloques
    de otros epígrafes.

Evidencia local nueva:

```text
/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/tools/validate_public_text_no_study_scaffolding.py
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/09_validacion/informe_texto_publico_sin_andamiaje_interno.json
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/09_validacion/informe_texto_publico_sin_andamiaje_interno.md
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/09_validacion/informe_avance_estricto_grupo_b_informatica_2026-07-01.json
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/09_validacion/informe_avance_estricto_grupo_b_informatica_2026-07-01.md
```

Resultado del informe estricto tras limpieza:

- Total temas: 50.
- Pasan mínimo B por extensión: 20.
- Pasan extensión + validadores oficiales de texto: 18.
- Pasan extensión + QA estricta sin andamiaje interno: 4.
- Hallazgos de andamiaje interno: 215.

Acción esperada en Orquesta/OPES:

- El contrato final de tema no puede usar solo extensión + metanotas como verde
  editorial.
- Añadir fase QA obligatoria equivalente a
  `validate_public_text_no_study_scaffolding.py` o integrar sus patrones en el
  validador de calidad OPES.
- Si falla, el estado debe ser `pendiente_rework_editorial` o
  `needs_remove_study_scaffolding_and_cross_topic_contamination`, no `ready`.
- `autoprogramming/status`, receipts y cierre de padre deben separar:
  `extension_pass`, `official_text_qa_pass` y `strict_editorial_qa_pass`.
- Orquesta debe detectar cuando la extensión se consigue por anexos ajenos y
  pedir rework focal, no dar por cerrado el tema.

No se ha tocado código del núcleo de Orquesta. Esta sección queda para el
agente que está corrigiendo Orquesta.

### 35. BUG-ORQ-20260701-091: con HEAD 393e7bfd readiness vuelve a quedar verde aunque el HTTP muere antes de `external-work/run`

Fecha observada: 2026-07-01 20:34-20:36 Europe/Madrid.

Nota de inventario: esta incidencia se consolida como
`BUG-ORQ-20260701-091` en el inventario central de Orquesta.

Contexto:

- Se intentó relanzar Orquesta desde OPES para que dirigiera el rework textual
  acotado del tema 026 de Grupo B Informática.
- Se comprobó que la rama local de Orquesta ya no estaba en el mismo estado que
  la prueba anterior: `git rev-parse --short HEAD` devolvió `393e7bfd`.
- El árbol de Orquesta tenía cambios locales de otro agente/programador, por lo
  que OPES no tocó código del núcleo.
- Runtime usado:
  `/home/alberto/Trabajo/orquesta/.orquesta-runtime/grupo-b-info-wave18-t026-text-20260701T183443Z`.
- Evidencia OPES:
  `/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_responses/wave18_t026_text/`.

Evidencia positiva:

- `GET /api/v0/server/readiness` devolvió `ready=true`, `status=running`,
  `startup_status=startup_ready`, `startup_ready=true`.
- La identidad runtime declaró:
  - `commit_ref=393e7bfdca1bce515180539059a71a8d3245643e`;
  - `build_ref=build-ref-orquesta-server-393e7bfdca1b-modified`;
  - `external_bridge_status=disabled`;
  - `external_bridge_ready=true`.
- El wrapper OPES escribió `base_url.txt` con:
  `http://127.0.0.1:39989`.

Problema:

- Justo al enviar `POST /api/v0/external-work/run` para el tema 026, `curl`
  devolvió:

```text
curl: (7) Failed to connect to 127.0.0.1 port 39989 after 0 ms: Could not connect to server
```

- El PID del servidor `3592705` ya no existía, pero el state durable seguía
  declarando:

```json
{
  "status": "running",
  "pid": 3592705,
  "addr": "127.0.0.1:39989",
  "startup_status": "startup_ready",
  "startup_ready": true,
  "startup_message": "director: orquesta preparada; autodiagnostico sin runs transitorios ni cola sucia"
}
```

- Quedaron procesos residuales del backend Goal creados por ese arranque:

```text
3592720 tmux new-session ... orquesta-goal-40dedd8d2b4bfd86
3592721 node ... codex app-server --listen unix:///tmp/oq-gsrv-1000-40dedd8d2b4bfd86/s.sock
3592738 .../bin/codex app-server --listen unix:///tmp/oq-gsrv-1000-40dedd8d2b4bfd86/s.sock
```

Acción esperada:

- Si el HTTP muere después de publicar readiness, el state durable debe pasar a
  `crashed/stopped` o exponer causa pública tipo `server_exited_after_readiness`.
- Readiness no debería declararse estable si el proceso muere antes de aceptar
  el primer `external-work/run`.
- Al morir el servidor debe limpiar o marcar claramente el backend Goal propio,
  evitando `tmux`/`codex app-server` residuales.
- `external_bridge_status=disabled` con `external_bridge_ready=true` necesita
  semántica pública: para OPES resulta confuso si esa ruta es necesaria para
  trabajos `external-work/run`.

No se ha tocado código del núcleo de Orquesta. OPES documentó la incidencia y
limpió manualmente los procesos residuales de esta prueba.

### 32. Ola 17: startup publica `running/startup_ready`, pero HTTP muere y queda app-server vivo

Fecha observada: 2026-07-01 19:43 Europe/Madrid.

Contexto:

- Se comprobó Orquesta en la rama `trabajo/plataforma-agentes`, con
  `HEAD...@{u}=0 0`.
- Commit usado para el binario:
  `daa44d9e61925d2a4f4fc91e1fdcbeeb363473dc`.
- Runtime:
  `/home/alberto/Trabajo/orquesta/.orquesta-runtime/grupo-b-info-wave17-t007-text-20260701T174254Z`.
- Objetivo OPES: preparar `wave17_t007_text` para rework textual acotado del
  tema 007, usando `/api/v0/external-work/run`.

Evidencia positiva:

- El servidor escribió `state/orquesta_server_state_v0.json` con:
  - `status=running`;
  - `pid=3443711`;
  - `addr=127.0.0.1:36675`;
  - `startup_status=startup_ready`;
  - `startup_ready=true`;
  - `startup_message=director: orquesta preparada...`.
- `base_url.txt` quedó materializado con:
  `http://127.0.0.1:36675`.

Problema:

- Segundos después, `curl /api/v0/server/readiness` contra ese `base_url`
  devolvió conexión rechazada.
- `ps` no mostraba ya el proceso `orquesta-server` `3443711`, pero seguían
  vivos procesos del backend Goal creados por ese arranque:

```text
3443726 tmux new-session ... orquesta-goal-cd6f2a69a9b1f208
3443727 node ... codex app-server --listen unix:///tmp/oq-gsrv-1000-cd6f2a69a9b1f208/s.sock
3443740 .../bin/codex app-server --listen unix:///tmp/oq-gsrv-1000-cd6f2a69a9b1f208/s.sock
```

- El estado durable seguía diciendo `status=running` y
  `startup_ready=true`, aunque el HTTP ya no estaba alcanzable.
- No se llegó a enviar el payload `external-work/run` del tema 007, por lo que
  el fallo ocurrió entre readiness inicial y uso inmediato del control-plane.

Evidencia local:

```text
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_responses/wave17_t007_text/base_url.txt
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_responses/wave17_t007_text/orquesta_head.txt
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_responses/wave17_t007_text/orquesta_runtime_root.txt
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_responses/wave17_t007_text/server.pid
/home/alberto/Trabajo/orquesta/.orquesta-runtime/grupo-b-info-wave17-t007-text-20260701T174254Z/state/orquesta_server_state_v0.json
```

Acción esperada:

- El arranque que publica `startup_ready=true` debe mantener el HTTP vivo o
  actualizar estado durable a `stopped/crashed` si el proceso termina.
- Si el proceso HTTP muere tras readiness, Orquesta debe ejecutar cleanup del
  backend Goal propio y cerrar tmux/app-server/sockets asociados.
- El wrapper/cliente no debería poder ver `running/startup_ready` con
  conexión rechazada y backend Goal residual sin diagnóstico tipo
  `server_exited_after_readiness`.
- La espera de readiness debería exigir estabilidad mínima durante varios
  polls o identidad estable del proceso antes de dar `READY`.

No se ha tocado código del núcleo de Orquesta. OPES documentó el fallo y cerró
manualmente los procesos residuales antes de seguir.

### 33. Ola 17b: Orquesta actual mejora `status`, pero vuelve a `checkpoint_only_high_consumption`

Fecha observada: 2026-07-01 19:47-19:52 Europe/Madrid.

Contexto:

- Se relanzó Orquesta con sesión persistente para evitar el fallo de la ola 17.
- Commit usado:
  `daa44d9e61925d2a4f4fc91e1fdcbeeb363473dc`.
- Runtime:
  `/home/alberto/Trabajo/orquesta/.orquesta-runtime/grupo-b-info-wave17b-t007-text-20260701T174636Z`.
- Run:
  `run-opes-grupo-b-info-wave17b-t007-text-20260701`.
- Objetivo OPES: rework textual acotado del tema 007, sin HTML/tests/RAG/
  visuales/audio.

Evidencia positiva:

- El arranque fue estable durante tres comprobaciones de readiness.
- Antes de lanzar el run, `autoprogramming/status` devolvió cola limpia.
- Después del launch, `autoprogramming/status` mejoró respecto al falso verde
  antiguo:
  - `queue_health.running_live=1`;
  - `efficiency_summary.state=live`;
  - `overall_percentage=99`;
  - acción recomendada `observe_goal`.
- Tras el bloqueo posterior, `status` ya no declaró 100% ni idle:
  - `queue_health.blocked=1`;
  - `efficiency_summary.state=attention_required`;
  - `overall_percentage=40`;
  - diagnóstico `goal_first_blocked_with_partial_delivery`;
  - evidencia `evidence-ref-goal-materialized-checkpoint-detected`.
- `server/shutdown` normal y `forced=true` devolvieron
  `backend_still_running`, `shutdown_ready=false`; esta parte evita el falso
  verde de apagado que se vio en olas anteriores.

Problema:

- Orquesta volvió a crear solo el checkpoint:

```text
02_temas/tema_007/coordinacion_wave17b/checkpoint_started.txt
```

- No creó:
  - `trabajo/tema_007_ampliado_limpio.md`;
  - `trabajo/tema_007_resumen_limpio.md`;
  - `validacion/informe_extension_temario.json`;
  - `trabajo/docs/orquesta_goal_result_v0.json`.
- `POST /api/v0/autoprogramming/goal/observe` devolvió HTTP 504 con:

```json
{
  "estado": "error",
  "partial": true,
  "recommended_action": "observe_later",
  "summary": "observacion de autoprogramacion incompleta por timeout HTTP"
}
```

- Snapshot de SQLite antes de limpieza manual:

```text
thread_id=019f1ecb-01d9-77e2-8c22-b7a8cdeeb433
status=active
tokens_used=118258
time_used_seconds=166
```

- Aunque `status` externo marcó el run como bloqueado, el backend Codex seguía
  `active` y el apagado no pudo cerrar el backend, por lo que OPES tuvo que
  cortar manualmente el servidor y los procesos:

```text
3453912 orquesta-server run
3454107 tmux ... orquesta-goal-64717c0721ef7f5b
3454108 node ... codex app-server
3454121 .../bin/codex app-server
```

Evidencia local:

```text
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_payloads/wave17b_t007_text/external_work_wave17b_topic_007_text_only.json
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_responses/wave17b_t007_text/response_t007_text_only.json
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_responses/wave17b_t007_text/status_30s.json
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_responses/wave17b_t007_text/observe_goal_1_http504_body.json
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_responses/wave17b_t007_text/status_after_observe_504.json
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_responses/wave17b_t007_text/shutdown_after_blocked.json
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_responses/wave17b_t007_text/shutdown_forced_after_blocked.json
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_responses/wave17b_t007_text/process_snapshot_before_manual_cleanup.txt
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_responses/wave17b_t007_text/sqlite_snapshot_before_manual_cleanup.txt
```

Acción esperada:

- Tras `checkpoint_started.txt`, si no aparece segundo artefacto dentro de un
  umbral de tokens/tiempo, Orquesta debe cortar antes de quemar cuota y emitir
  `checkpoint_only_high_consumption` con replan automático.
- `goal_first_blocked_with_partial_delivery` debe distinguir checkpoint de
  entrega parcial real: un checkpoint no basta para considerar que hay delivery
  de dominio.
- `observe_goal` no debe requerir una llamada HTTP larga para llegar al mismo
  diagnóstico; el snapshot de `status` ya tiene suficientes señales para
  devolver causa y acción.
- `shutdown forced=true` debería poder cerrar el backend Goal propio o devolver
  PIDs/acción concreta; ahora evita el falso `ready`, pero no detiene.

No se ha tocado código del núcleo de Orquesta. OPES limpió manualmente los
procesos tras guardar snapshot.

### 22. Ola 11: shutdown forzado declara `ready` pero deja app-servers vivos

Fecha observada: 2026-07-01 17:30 Europe/Madrid.

Contexto:

- Tras bloquearse `run-opes-grupo-b-info-wave11-t002-20260701`, se pidió
  shutdown no forzado con idempotency key.
- Resultado positivo parcial: ya no mintió con `shutdown_ready=true`; devolvió
  `status=backend_still_running`, `shutdown_ready=false` y listó el backend
  activo.
- Después se pidió shutdown forzado con:
  `idempotency_key=shutdown-wave11-t002-20260701-forced-1`.

Problema:

- El shutdown forzado devolvió:

```json
{"estado":"ok","status":"ready","shutdown_ready":true}
```

- Pero inmediatamente después seguían vivos procesos app-server:

```text
2779012 node ... codex app-server --listen unix:///tmp/oq-gsrv-1000-f5c5800ce032825e/s.sock
2779026 .../bin/codex app-server --listen unix:///tmp/oq-gsrv-1000-f5c5800ce032825e/s.sock
```

- Además seguían restos del primer intento fallido de arranque de la misma ola:

```text
2775615 tmux new-session ... /tmp/oq-gsrv-1000-47e2d66de469a8e9/s.sock
2775616 node ... codex app-server --listen unix:///tmp/oq-gsrv-1000-47e2d66de469a8e9/s.sock
2775630 .../bin/codex app-server --listen unix:///tmp/oq-gsrv-1000-47e2d66de469a8e9/s.sock
```

Acción esperada:

- `forced=true` no debe devolver `shutdown_ready=true` hasta comprobar que no
  quedan procesos/sockets app-server asociados al runtime.
- Si no puede cerrarlos, debe devolver `backend_still_running` con PIDs.
- Los arranques fallidos de app-server deben quedar registrados como owned por
  el runtime para que el shutdown posterior pueda recogerlos.

No se ha tocado código del núcleo de Orquesta. OPES cerró manualmente los
procesos residuales después de documentar la incidencia para poder seguir con el
temario.

### 23. Ola 12: startup `ready` sin `base_url.txt` y app-server residual si aborta el wrapper

Fecha observada: 2026-07-01 17:27-17:32 Europe/Madrid.

Contexto:

- Se construyó Orquesta desde `433129df0415203d1fa219617451da319db38278`.
- Runtime:
  `/home/alberto/Trabajo/orquesta/.orquesta-runtime/grupo-b-info-wave12-t002-rework-20260701T172740`.
- El wrapper OPES esperaba `base_url.txt` igual que en la ola 11.

Problema:

- Orquesta escribió `state/orquesta_server_state_v0.json` con:
  - `addr=127.0.0.1:46483`;
  - `startup_ready=true`;
  - `startup_message=director: orquesta preparada...`.
- La auditoría local mostró ticks residentes `supervisor_tick_start/result`.
- Pero no apareció `base_url.txt`.
- Al no aparecer el endpoint canónico, el wrapper OPES salió con `NO_BASE_URL`;
  después `curl http://127.0.0.1:46483` ya no conectaba y quedaron vivos
  procesos app-server/tmux `f34a7f9d3c61522b`.

Evidencia local:

```text
/home/alberto/Trabajo/orquesta/.orquesta-runtime/grupo-b-info-wave12-t002-rework-20260701T172740/state/orquesta_server_state_v0.json
/home/alberto/Trabajo/orquesta/.orquesta-runtime/grupo-b-info-wave12-t002-rework-20260701T172740/state/audit/orquesta_server_audit_v0.jsonl
tmux/app-server: orquesta-goal-f34a7f9d3c61522b
```

Acción esperada:

- Startup debe materializar `base_url.txt` o un endpoint canónico equivalente
  antes de publicar `startup_ready`.
- Si el wrapper cancela por falta de endpoint, Orquesta debe apagar los
  app-server/tmux arrancados durante bootstrap.
- El estado persistido no debe quedar `running` si el HTTP residente ya no está
  alcanzable.

No se ha tocado código del núcleo de Orquesta. OPES reintentó con arranque más
aislado para seguir el temario.

### 24. Ola 12b: artefactos con QA pública `pass`, pero el goal no cierra ni emite recibo terminal

Fecha observada: 2026-07-01 17:33-17:55 Europe/Madrid.

Contexto:

- Se ejecutó Orquesta desde `433129df0415203d1fa219617451da319db38278`.
- Runtime:
  `/home/alberto/Trabajo/orquesta/.orquesta-runtime/grupo-b-info-wave12b-t002-rework-20260701T173100`.
- Run:
  `run-opes-grupo-b-info-wave12b-t002-rework-qa-20260701`.
- Goal:
  `019f1e4f-e799-7181-99a5-ae370619263f`.
- El objetivo era rework focal del tema 002 de Grupo B Informática tras la ola
  11: limpiar anclas visibles, corregir errores ortográficos introducidos por
  cambios masivos, regenerar HTML/RAG/tests y dejar validación local.

Evidencia positiva:

- Orquesta materializó artefactos útiles y validables:
  - `trabajo/tema_002_ampliado_limpio.md`;
  - `trabajo/tema_002_resumen_limpio.md`;
  - `html/html_final/tema_002_organos_constitucionales.html`;
  - `html/html_ampliado/tema_002_organos_constitucionales.html`;
  - `tests/tema_002_tests_rework.json`;
  - `tutor_rag/rag/corpus/chunks.jsonl`;
  - `tutor_rag/rag/corpus/summary.json`;
  - `tutor_rag/rag/manifest.json`.
- El informe público consolidado quedó en `pass`:
  `validacion/informe_wave12b_qa_publica.json`.
- Conteo estricto OPES:
  `ampliado_words_WORD_RE=14090`, mínimo Grupo B `10800`.
- Tests:
  `question_count=24`, mínimo `20`, schema `pass`.
- Validaciones incluidas en el consolidado:
  conteo, anclas visibles, ortografía focal, HTML, RAG, tests, genericidad,
  ausencia de notas de autor y ausencia de metatexto de examen.

Problema:

- No aparecieron los recibos terminales esperados:
  - `trabajo/docs/orquesta_goal_result_v0.json`;
  - `trabajo/opes_topic_rework_delivery.json`.
- `POST /api/v0/autoprogramming/status` devolvió una combinación contradictoria:
  - cola vacía;
  - `running_live=1`;
  - `active_runs=0`;
  - `overall_percentage=100`;
  - `recommended_action=observe_goal`;
  - diagnóstico `goal_status=running`.
- `POST /api/v0/autoprogramming/goal/observe` devolvió:

```json
{
  "estado": "error",
  "partial": true,
  "recommended_action": "observe_later",
  "summary": "observacion de autoprogramacion incompleta por timeout HTTP"
}
```

Evidencia local:

```text
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/02_temas/tema_002/validacion/informe_wave12b_qa_publica.json
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/02_temas/tema_002/validacion/informe_wave12b_conteo_word_re.json
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/02_temas/tema_002/validacion/informe_wave12b_tests_schema.json
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_responses/wave12b_t002_rework_qa/observe_goal_after_artifacts.json
```

Acción esperada:

- Si el write-set tiene artefactos canónicos y validaciones `pass`, Orquesta
  debe materializar un recibo terminal verificable sin depender de otro turno
  largo del LLM, o devolver una causa precisa tipo
  `missing_terminal_receipt_after_artifacts_pass`.
- El estado público no debe mezclar `overall_percentage=100` con
  `goal_status=running` sin acción terminal concreta.
- La acción recomendada no debería ser solo `observe_goal` si ya hay evidencia
  local suficiente: debe ser `repair_receipt`, `reconcile_goal_terminal` o
  `shutdown_safe_after_artifacts_pass`, con lista de ficheros esperados y PIDs
  asociados.
- El cierre seguro debe distinguir claramente:
  `artefactos publicables validados`, `faltan recibos`, `backend vivo` y
  `riesgo de perder trabajo`.

No se ha tocado código del núcleo de Orquesta. Esta sección queda para el
agente que está corrigiendo Orquesta.

### 25. Ola 13: alto consumo sin checkpoint temprano ni artefactos en write-set

Fecha observada: 2026-07-01 18:04-18:10 Europe/Madrid.

Contexto:

- Orquesta usada: `0af9e345a45852f55ad76377f10845af8328e78c`
  (`Diagnostica auth fallida en backend goal`).
- Runtime:
  `/home/alberto/Trabajo/orquesta/.orquesta-runtime/grupo-b-info-wave13-t003-rework-20260701T180100`.
- Run:
  `run-opes-grupo-b-info-wave13-t003-rework-qa-20260701`.
- Goal:
  `019f1e6c-4dbe-7a02-ad2e-4e2d12d1eb70`.
- Tema OPES afectado: Grupo B Informática, tema 003.
- El payload ya pedía seis subroles, reutilización local, no audio, no tocar
  Orquesta y recibos terminales.

Problema:

- Tras unos 309 segundos y `tokens_used=249435` en `goals_1.sqlite`, el goal
  seguía `active`.
- No se creó ningún fichero nuevo en:

```text
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/02_temas/tema_003/
```

- No apareció `checkpoint_started.txt`, matriz, borrador, delivery ni receipt.
- El rollout sí mostraba actividad, pero era lectura/volcado de contexto antes
  de materializar avance:
  - `lynx -dump ... | sed -n '1,260p'`;
  - `rg` de assets visuales con salida enorme y truncada;
  - salida del `tail` de rollout de unas 76k tokens por acumulación de outputs.
- `autoprogramming/status` seguía devolviendo una combinación no operable:
  - cola vacía;
  - `running_live=1`;
  - `active_runs=0`;
  - `overall_percentage=100`;
  - `goal_status=running`;
  - `recommended_action=observe_goal`.
- `observe_goal` devolvió timeout parcial casi inmediato:

```json
{
  "estado": "error",
  "partial": true,
  "recommended_action": "observe_later",
  "errores_publicos": [
    {
      "code": "observe_app_director_goal_timeout"
    }
  ]
}
```

- `POST /api/v0/server/shutdown` sin `forced` bloqueó correctamente con
  `status=active_goals_present` y `shutdown_ready=false`.
- `POST /api/v0/server/shutdown` con `forced=true` devolvió
  `shutdown_ready=true` y cerró procesos/tmux/app-server, pero
  `goals_1.sqlite` conservó:

```text
019f1e6c-4dbe-7a02-ad2e-4e2d12d1eb70|active|249435|309|2026-07-01 16:09:11
```

Evidencia local:

```text
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_responses/wave13_t003_rework_qa/response_t003.json
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_responses/wave13_t003_rework_qa/observe_goal_1.json
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_responses/wave13_t003_rework_qa/status_180s.json
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_responses/wave13_t003_rework_qa/shutdown_normal_after_bug079.json
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_responses/wave13_t003_rework_qa/shutdown_forced_after_bug079.json
/home/alberto/Trabajo/orquesta/.orquesta-runtime/grupo-b-info-wave13-t003-rework-20260701T180100/codex-runtime/goal-srv/codex-home/sessions/2026/07/01/rollout-2026-07-01T18-04-02-019f1e6c-4dbe-7a02-ad2e-4e2d12d1eb70.jsonl
```

Acción esperada:

- Orquesta debe exigir checkpoint temprano en el write-set del dominio antes de
  permitir consumos altos. Ejemplo: `checkpoint_started.txt` o matriz mínima
  antes de superar un umbral configurable de tokens/tiempo.
- Las salidas de herramientas grandes no deben entrar completas al contexto del
  agente. Deben resumirse, limitarse o guardarse como fichero con muestra
  acotada.
- El estado residente debe detectar:
  `goal active + tokens crecientes + cero ficheros nuevos en write-set` y
  publicar una causa accionable, por ejemplo
  `goal_active_no_checkpoint_high_consumption`.
- En ese estado, la acción recomendada no debe ser solo `observe_goal`; debe ser
  `replan_narrow_context`, `stop_goal_safe` o `require_checkpoint_before_more`.
- `overall_percentage=100` no debe convivir con `goal_status=running` y cero
  artefactos del dominio.
- Si el operador fuerza cierre sin artefactos, el estado durable debe quedar
  reconciliado como terminal operativo (`stopped_no_artifacts_by_operator` o
  equivalente), no como `active` indefinido en SQLite.

No se ha tocado código del núcleo de Orquesta. Esta sección queda para el
agente que está corrigiendo Orquesta.

### 26. Ola 14: rutas exactas convertidas en refs saneadas y usadas como paths

Fecha observada: 2026-07-01 18:15-18:20 Europe/Madrid.

Contexto:

- Orquesta usada: `0af9e345a45852f55ad76377f10845af8328e78c`
  (`Diagnostica auth fallida en backend goal`).
- Runtime:
  `/home/alberto/Trabajo/orquesta/.orquesta-runtime/grupo-b-info-wave14-t003-phase0-20260701T181330`.
- Run:
  `run-opes-grupo-b-info-wave14-t003-phase0-20260701`.
- Goal:
  `019f1e76-d25a-70d3-8567-38e179007220`.
- Tema OPES afectado: Grupo B Informática, tema 003.
- La ola era una fase 0 estrecha: solo inventario, matriz de fuentes y plan,
  sin redacción completa ni revisión visual profunda.

Problema:

- El payload enviado por OPES contenía rutas reales con `/`, por ejemplo:

```text
opes-salidas/orquesta_real/grupo_b_informatica_2026-06-04/rework/package_grupo_b_final_local_2026-06-04/html_ampliado/tema_003_administracion_publica_y_organizacion_territorial_del_estado.html
opes-salidas/codex_directo/informatica/grupo_B/revision_tutor_visual_2026-06-04/package_source_local_2026-06-04_1630_material_saneado/html_final/img/tema_003__opes_const_territorial_v2_00001_.webp
```

- En el rollout, el agente intentó ejecutar comandos sobre rutas deformadas:

```text
opes-salidas/orquesta_real/grupo_b_informatica_2026-06-04/rework-package_grupo_b_final_local_2026-06-04/...
opes-salidas/codex_directo/informatica-grupo_B/...
```

- Los comandos `ls` y `wc` fallaron sobre fuentes que sí existían con la ruta
  original.
- La señal observada apunta a que el `spec summary` o la capa de contexto
  convirtió rutas reales en `context_refs` saneadas, sustituyendo separadores
  por guiones, y el agente las trató después como rutas de filesystem.
- Esto puede hacer que Orquesta descarte fuentes válidas, diagnostique
  falsamente `sin material`, o cree una matriz incorrecta desde rutas corruptas.

Evidencia local:

```text
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_payloads/wave14_t003_phase0/external_work_wave14_topic_003_phase0.json
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_responses/wave14_t003_phase0/dry_run_t003_phase0.json
/home/alberto/Trabajo/orquesta/.orquesta-runtime/grupo-b-info-wave14-t003-phase0-20260701T181330/codex-runtime/goal-srv/codex-home/sessions/2026/07/01/rollout-2026-07-01T18-15-31-019f1e76-d25a-70d3-8567-38e179007220.jsonl
```

Acción esperada:

- Orquesta debe diferenciar explícitamente entre `context_ref` opaco y
  `filesystem_path` ejecutable.
- Si necesita refs saneadas, debe conservar también el valor original en campos
  estructurados no destructivos, por ejemplo `filesystem_paths[]` o
  `input_fields.filesystem_paths`.
- Los prompts generados no deben presentar refs saneadas como si fueran rutas.
- La validación de existencia de fuentes debe hacerse siempre sobre la ruta
  original.

No se ha tocado código del núcleo de Orquesta. Esta sección queda para el
agente que está corrigiendo Orquesta.

### 27. Ola 14: `view_image` inserta base64 masivo en el rollout

Fecha observada: 2026-07-01 18:17-18:20 Europe/Madrid.

Contexto:

- Misma ola que el apartado 26:
  `run-opes-grupo-b-info-wave14-t003-phase0-20260701`.
- Goal:
  `019f1e76-d25a-70d3-8567-38e179007220`.
- La tarea pedía fase 0 documental. No hacía falta inspección visual profunda
  ni incorporar binarios al historial operativo.

Problema:

- El agente usó `view_image` sobre:

```text
/home/alberto/Trabajo/OPES/opes-salidas/codex_directo/temas_comunes/constitucion_espanola/html/assets/13_organizacion_territorial.png
```

- La imagen pesaba unos 328K y el rollout incorporó un
  `data:image/png;base64,...` enorme.
- El `tail` del rollout superó 120k tokens y el goal quedó por encima de
  `171225` tokens y `222` segundos sin entregar la matriz ni el plan de fase 0.
- El problema no es revisar imágenes; el problema es que el transcript
  operativo recibió el binario/base64 completo, que degrada coste, observación,
  límites de contexto y cierre causal.

Evidencia local:

```text
/home/alberto/Trabajo/orquesta/.orquesta-runtime/grupo-b-info-wave14-t003-phase0-20260701T181330/codex-runtime/goal-srv/codex-home/sessions/2026/07/01/rollout-2026-07-01T18-15-31-019f1e76-d25a-70d3-8567-38e179007220.jsonl
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_responses/wave14_t003_phase0/status_105s.json
```

Acción esperada:

- Las herramientas visuales bajo Orquesta deben guardar referencia, hash,
  dimensiones y resumen/miniatura segura, no base64 completo en el transcript.
- Orquesta debe imponer presupuesto de salida por herramienta y cortar o
  resumir outputs multimodales grandes.
- Para QA visual, pedir capturas, hojas de contacto o informes como archivos
  en el write-set, no como payload textual gigante dentro del historial del
  goal.

No se ha tocado código del núcleo de Orquesta. Esta sección queda para el
agente que está corrigiendo Orquesta.

### 28. Ola 14: escritura fuera de `allowed_write_set` y recibo que oculta artefactos extra

Fecha observada: 2026-07-01 18:22-18:35 Europe/Madrid.

Contexto:

- Misma ola que los apartados 26 y 27:
  `run-opes-grupo-b-info-wave14-t003-phase0-20260701`.
- Goal:
  `019f1e76-d25a-70d3-8567-38e179007220`.
- El payload declaraba fase 0 estrecha: crear checkpoint, matriz, plan,
  delivery y `trabajo/docs/orquesta_goal_result_v0.json`.
- `allowed_write_set` estaba limitado a:

```text
opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/02_temas/tema_003/coordinacion_wave14
opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/02_temas/tema_003/trabajo/docs
```

- Los criterios de aceptación incluían literalmente:
  `No redactar tema ampliado ni generar HTML/RAG/tests en esta fase`.

Problema:

- El agente terminó con `thread_goals.status=complete`, pero escribió mucho más
  que la fase 0:

```text
02_temas/tema_003/trabajo/tema_003_ampliado_limpio.md
02_temas/tema_003/trabajo/tema_003_resumen_limpio.md
02_temas/tema_003/html/
02_temas/tema_003/tests/tema_003_tests_rework.json
02_temas/tema_003/tutor_rag/rag/corpus/chunks.jsonl
02_temas/tema_003/validacion/
```

- Esos artefactos son útiles tras QA y limpieza editorial focal, pero no estaban
  autorizados por el write-set de esa ola.
- El recibo terminal `trabajo/docs/orquesta_goal_result_v0.json` declara solo
  fase 0 completa y sus evidencias son checkpoint/matriz/plan/delivery. No
  declara los artefactos extra, no los marca como fuera de scope y no adjunta su
  QA.
- Resultado operativo: OPES puede recuperar trabajo útil, pero Orquesta no
  dejó trazabilidad causal completa de lo que realmente hizo ni de por qué
  consumió mucho más que una fase 0.

Evidencia local:

```text
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_payloads/wave14_t003_phase0/external_work_wave14_topic_003_phase0.json
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/02_temas/tema_003/trabajo/docs/orquesta_goal_result_v0.json
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/02_temas/tema_003/trabajo/tema_003_ampliado_limpio.md
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/02_temas/tema_003/html/
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/02_temas/tema_003/tests/tema_003_tests_rework.json
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/02_temas/tema_003/tutor_rag/rag/corpus/chunks.jsonl
```

Acción esperada:

- En goal-first, `allowed_write_set` debe ser vinculante o, como mínimo,
  auditable en cierre.
- Si un agente escribe fuera del write-set, Orquesta debe bloquear o marcar el
  cierre como `out_of_scope_artifacts_written`, listar rutas y pedir replan o
  aceptación explícita del director.
- El recibo terminal debe incluir todos los artefactos creados. Si son extra,
  deben figurar como `out_of_scope_artifacts`, con estado `reusable`, `invalid`
  o `needs_qa`.
- No debe publicarse `complete` como si solo se hubiese cumplido fase 0 cuando
  el write-set real muestra redacción, HTML, tests, RAG y validaciones.

No se ha tocado código del núcleo de Orquesta. Esta sección queda para el
agente que está corrigiendo Orquesta.

### 29. Ola 15: startup aborta por cola desincronizada pero deja app-server vivo

Fecha observada: 2026-07-01 18:39 Europe/Madrid.

Contexto:

- Se intentó arrancar un runtime aislado para relanzar el tema 004.
- Orquesta usada: `cf492f2b`.
- Runtime:
  `/home/alberto/Trabajo/orquesta/.orquesta-runtime/grupo-b-info-wave15-t004-rework-20260701T183900`.
- Composición: `ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux`,
  `ORQUESTA_CODEX_PROJECT_WORKDIR=/home/alberto/Trabajo/OPES`,
  `ORQUESTA_CODEX_RUNTIME_WORKDIR=.../codex-runtime`.

Problema:

- El servidor salió durante el arranque con:

```text
orquesta-server: orquesta_server: startup_dirty_runs_detected: runs transitorios activos=0 cola_desincronizada=8; usar ORQUESTA_STARTUP_CLEANUP_MODE=forced_stop para purga logica
```

- Antes de salir ya había creado una sesión Goal:

```text
orquesta-goal-901a77e254b3363c
```

- Quedaron vivos:

```text
node ... codex app-server --listen unix:///tmp/oq-gsrv-1000-901a77e254b3363c/s.sock
.../bin/codex app-server --listen unix:///tmp/oq-gsrv-1000-901a77e254b3363c/s.sock
```

- OPES tuvo que ejecutar `tmux kill-session -t
  orquesta-goal-901a77e254b3363c` y verificar que no quedaban procesos.

Acción esperada:

- La detección de dirty runs o la purga lógica debe ocurrir antes de levantar
  el backend Goal.
- Si el arranque aborta después de crear backend, el servidor debe invocar
  cleanup cooperativo de cualquier tmux/app-server creado durante bootstrap.
- La instrucción `usar ORQUESTA_STARTUP_CLEANUP_MODE=forced_stop` no debe dejar
  procesos vivos que consuman recursos mientras el servidor ya no existe.

No se ha tocado código del núcleo de Orquesta. Esta sección queda para el
agente que está corrigiendo Orquesta.

### 30. Ola 15: goal activo con alto consumo, solo checkpoint, status y shutdown sin respuesta

Fecha observada: 2026-07-01 18:43-18:50 Europe/Madrid.

Contexto:

- Después del fallo del apartado 29 se reintentó el mismo runtime con
  `ORQUESTA_STARTUP_CLEANUP_MODE=forced_stop`.
- Orquesta usada: `cf492f2b`.
- Runtime:
  `/home/alberto/Trabajo/orquesta/.orquesta-runtime/grupo-b-info-wave15-t004-rework-20260701T183900`.
- Run:
  `run-opes-grupo-b-info-wave15-t004-rework-20260701`.
- Goal externo:
  `019f1e91-2388-7830-96f0-02251d8664f5`.
- Tema OPES afectado: Grupo B Informática, tema `004`.

Evidencia positiva:

- El servidor arrancó y readiness quedó `ready=true` tras la purga lógica.
- `external-work/run` aceptó el trabajo.
- El agente creó el checkpoint inicial:

```text
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/02_temas/tema_004/coordinacion_wave15/checkpoint_started.txt
```

Problema:

- Pasados más de 200 segundos, el único artefacto del tema seguía siendo el
  checkpoint inicial y los dos ficheros de entrada.
- `goals_1.sqlite` seguía mostrando el goal como `active`, con
  `tokens_used=139028` y `time_used_seconds=243`.
- `POST /api/v0/autoprogramming/status` no devolvió snapshot útil de la run:
  agotó la ventana HTTP y devolvió:

```json
{
  "code": "autoprogramming_status_timeout",
  "message": "consulta de estado excedio la ventana HTTP acotada"
}
```

- `POST /api/v0/server/shutdown` normal expiró sin cuerpo y devolvió `000`.
- `POST /api/v0/server/shutdown` forzado también expiró sin cuerpo y devolvió
  `000`.
- Tras los dos intentos por API quedaron vivos `orquesta-server` y los procesos
  `codex app-server` del goal. OPES tuvo que guardar snapshot de procesos y
  matar manualmente los PIDs de esa runtime con `kill -9`.

Evidencia local:

```text
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_payloads/wave15_t004_rework/external_work_wave15_topic_004_rework.json
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_responses/wave15_t004_rework/
  dry_run_t004_rework.json
  response_t004_rework.json
  status_after_checkpoint.json
  shutdown_normal_high_tokens_checkpoint_only.json
  shutdown_forced_high_tokens_checkpoint_only.json
  process_snapshot_before_manual_cleanup.txt
  sqlite_snapshot_before_manual_cleanup.txt
```

Acción esperada:

- Orquesta debe detectar `active + tokens crecientes + solo checkpoint` como
  progreso insuficiente y devolver estado accionable antes de quemar cuota.
- `autoprogramming/status` debe tener snapshot rápido por `run_ref`/goal aunque
  el executor principal esté ocupado.
- `server/shutdown` normal y forzado deben devolver JSON operativo incluso si
  no pueden parar el backend; no deben colgar el HTTP hasta timeout sin cuerpo.
- La parada forzada debe matar o reconciliar el `orquesta-server`, tmux y
  `codex app-server` propios de esa runtime, o devolver PIDs y acción segura.
- Tras limpieza manual, el estado durable no debe quedar como `active` sin
  causa terminal recuperable (`operator_forced_stop_no_artifacts`,
  `checkpoint_only_high_consumption` o equivalente).

No se ha tocado código del núcleo de Orquesta. Esta sección queda para el
agente que está corrigiendo Orquesta.

### 31. Ola 16: `264ed9f3` mejora startup/status, pero persiste alto consumo con solo checkpoint y shutdown forzado deja app-server

Fecha observada: 2026-07-01 18:58-19:05 Europe/Madrid.

Contexto:

- Se probó Orquesta actualizada a `264ed9f3` (`Limpia hooks si startup queda
  bloqueado`), alineada con `origin/trabajo/plataforma-agentes`.
- Runtime:
  `/home/alberto/Trabajo/orquesta/.orquesta-runtime/grupo-b-info-wave16-t004-text-20260701T185721`.
- Run:
  `run-opes-grupo-b-info-wave16-t004-text-20260701`.
- Goal externo:
  `019f1ea0-0f00-7c32-a510-1b07d9abf5bb`.
- Tema OPES afectado: Grupo B Informática, tema `004`.
- El payload se redujo a fase textual: checkpoint, ampliado, resumen y
  validaciones textuales. Se prohibieron HTML, tests, RAG, visuales y audio.

Evidencia positiva:

- Startup ya no reprodujo el fallo del apartado 29:

```text
startup_message=director: orquesta preparada; no habia runs transitorios activos
```

- Readiness quedó `ready=true`.
- `POST /api/v0/autoprogramming/status` respondió HTTP 200 y recomendó
  `observe_goal`, en vez de colgar como en la ola 15.
- `POST /api/v0/apps/director/goal/observe` devolvió HTTP 504, pero con
  snapshot parcial útil: `partial=true`, `goal_status=running`,
  `current_phase=programacion` y `recommended_action=observe_later`.
- El agente creó `coordinacion_wave16/checkpoint_started.txt`.
- `POST /api/v0/server/shutdown` normal devolvió HTTP 409 con
  `status=active_goals_present` y listó el goal vivo, en vez de expirar sin
  cuerpo.

Problema:

- A los 207 segundos el goal seguía `active`, con `tokens_used=166009`.
- El único artefacto nuevo seguía siendo:

```text
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/02_temas/tema_004/coordinacion_wave16/checkpoint_started.txt
```

- No apareció `trabajo/tema_004_ampliado_limpio.md`, resumen ni validaciones.
- `POST /api/v0/server/shutdown` forzado devolvió HTTP 200 con:

```json
{"status":"ready","shutdown_ready":true}
```

- Pero quedaron vivos los procesos `codex app-server` asociados al goal:

```text
node ... codex app-server --listen unix:///tmp/oq-gsrv-1000-37fad0bf84022972/s.sock
.../bin/codex app-server --listen unix:///tmp/oq-gsrv-1000-37fad0bf84022972/s.sock
```

- OPES guardó snapshot posterior y tuvo que matar manualmente esos PIDs con
  `kill -9`.

Evidencia local:

```text
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_payloads/wave16_t004_text/external_work_wave16_topic_004_text_only.json
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_responses/wave16_t004_text/
  dry_run_t004_text_only.json
  response_t004_text_only.json
  observe_goal_30s.json
  status_90s.json
  status_before_shutdown_high_tokens_checkpoint_only.json
  sqlite_snapshot_before_shutdown.txt
  shutdown_normal_high_tokens_checkpoint_only.json
  shutdown_forced_high_tokens_checkpoint_only.json
  process_snapshot_before_shutdown.txt
  process_snapshot_after_forced_shutdown_ready.txt
  sqlite_snapshot_after_manual_cleanup.txt
```

Acción esperada:

- Cerrar `BUG-088` no puede limitarse a startup ni a `status` accionable:
  falta cortar o replanificar automáticamente `active + tokens altos + solo
  checkpoint`.
- El contrato de progreso debe exigir un segundo artefacto material en un
  umbral razonable después del checkpoint, o devolver estado tipo
  `checkpoint_only_high_consumption`.
- `shutdown forced=true` no debe declarar `shutdown_ready=true` si quedan vivos
  `codex app-server`/socket/PIDs asociados al goal. Si no puede cerrarlos, debe
  devolver `backend_still_running` con PIDs.
- Tras parada forzada o limpieza manual, el estado durable no debe quedar
  simplemente `active` sin causa terminal recuperable.

No se ha tocado código del núcleo de Orquesta. Esta sección queda para el
agente que está corrigiendo Orquesta.

### 21. Ola 11: artefactos recuperables, pero cierre bloqueado y QA no publicable

Fecha observada: 2026-07-01 17:05-17:23 Europe/Madrid.

Contexto:

- Se probó Orquesta actualizada en commit `d40ad337de16bf05a30da308781a174dbbdc9268`.
- Se lanzó un único tema para no abrir más paralelismo hasta verificar el
  cierre: `run-opes-grupo-b-info-wave11-t002-20260701`, tema 002 de Grupo B
  Informática.
- Runtime local:
  `/home/alberto/Trabajo/orquesta/.orquesta-runtime/grupo-b-info-wave11b-20260701T170509`.
- Work-set OPES:
  `/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/02_temas/tema_002`.

Evidencia positiva:

- El agente sí escribió material útil:
  - `trabajo/tema_002_ampliado_limpio.md` con unas 14.407 palabras.
  - `trabajo/tema_002_resumen_limpio.md`.
  - `tests/tema_002_tests_rework.json` con 24 preguntas.
  - HTML reducido y ampliado.
  - visual WebP profesional.
  - RAG canónico en `tutor_rag/rag/corpus/chunks.jsonl` y
    `tutor_rag/rag/corpus/summary.json`.
- El RAG ya no quedó vacío tras el primer intento: `chunks.jsonl` tenía 52
  líneas.
- La telemetría inicial mantuvo acción segura `observe_goal` durante más tiempo
  que la ola 10, por lo que parte de BUG-073 parece mitigada.

Problema:

- `POST /api/v0/autoprogramming/status` terminó con:
  - `goal_first_blocked`
  - `goal_backend_state_unreconciled`
  - `closure_needs_rework=true`
  - `goal_status=blocked`
  - `recommended_action=review_replan_goal_first`
- `POST /api/v0/apps/director/goal/observe` siguió devolviendo timeout parcial:
  `observe_app_director_goal_timeout`.
- El run no emitió un ACK final útil tipo `partial_artifacts_written` +
  `qa_failed_public_text` + lista de artefactos válidos/no válidos + rework
  focal.
- Los artefactos quedaron recuperables, pero no publicables:
  - anclas Markdown visibles `{#...}` en texto público;
  - tablas colapsadas en encabezados;
  - errores introducidos por corrección masiva, como `órgaños`, `confíanza`,
    `instituciónal` y `propuestá`;
  - el propio agente detectó esos problemas en el rollout, intentó corregirlos
    y aun así el cierre quedó bloqueado/opaco.

Evidencia local:

```text
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_responses/wave11_single_t002_post_fix/
  response_t002.json
  status_early.json
  status_30s.json
  status_150s.json
  status_330s.json
  status_510s.json
  observe_goal_t002_1.json
  observe_goal_t002_2.json

/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/02_temas/tema_002/
  coordinacion_wave11/checkpoint_started.txt
  trabajo/tema_002_ampliado_limpio.md
  trabajo/tema_002_resumen_limpio.md
  tests/tema_002_tests_rework.json
  html/html_final/tema_002_organos_constitucionales.html
  html/html_ampliado/tema_002_organos_constitucionales.html
  tutor_rag/rag/corpus/chunks.jsonl
  tutor_rag/rag/corpus/summary.json
  visuales/final/tema_002_organos_constitucionales_equilibrio_profesional.webp
```

Acción esperada:

- Si el goal escribe ficheros, Orquesta debe distinguir entre:
  `no_artifacts`, `partial_artifacts_written`, `qa_failed_public_text`,
  `qa_failed_tests_schema`, `ready_local_non_publishable` y `ready_candidate`.
- El cierre goal-first debe devolver un contrato determinista con:
  artefactos escritos, validadores ejecutados, fallos concretos y rework
  automático propuesto.
- Si el agente detecta artefactos no publicables, no debe dejar `ready_local` en
  RAG/HTML sin matiz: el estado global debe quedar bloqueado por QA pública.
- `observe_goal` no debe quedar en timeout genérico cuando ya existe suficiente
  evidencia materializada para construir un snapshot de cierre/rework.

No se ha tocado código del núcleo de Orquesta. Esta sección queda para el
agente que está corrigiendo Orquesta.

### 20. Ola 10: Orquesta actualizada escribe checkpoint, pero corta fase 1 por `goal_active_timeout`

Fecha observada: 2026-07-01 16:37-16:42 Europe/Madrid.

Contexto:

- Se arrancó Orquesta con `HEAD=79f21136`, que incluye:
  - `13e9e0bd fix(server): expose active goal backend on idle queue`;
  - `f1f10c38 fix(goal-first): expose usage-limited goal snapshots`.
- Runtime aislado:
  `/home/alberto/Trabajo/orquesta/.orquesta-runtime/grupo-b-informatica-wave10-phase1-t001-20260701/state`.
- `ORQUESTA_CODEX_PROJECT_WORKDIR=/home/alberto/Trabajo/OPES`.
- `run_ref`:
  `run-opes-grupo-b-info-wave10-phase1-t001-20260701`.
- `external_goal_ref`:
  `019f1e1d-fa61-7993-9823-80b0be5249cc`.
- Objetivo OPES: tema 001, fase 1, crear solo:
  - `checkpoint_started.txt`;
  - `matriz_derivacion_b_desde_maestro.md`;
  - `lagunas_y_contaminacion.md`;
  - `decision_visuales_candidatos.md`;
  - `docs/orquesta_goal_result_v0.json`.

Mejora observada:

- `GET /api/v0/routes` funciona y muestra `/api/v0/external-work/run`.
- `POST /api/v0/autoprogramming/status` antes del run muestra cola viva.
- `POST /api/v0/autoprogramming/goals/observe-active` ya no devuelve solo
  `{"estado":"ok"}`: devuelve `operation_ref`,
  `autoprogramming_observe_active_goals_background_accepted` y recomienda
  consultar `autoprogramming/status`.
- `autoprogramming/status` durante el run expone `running_live=1`,
  `goal_status=running` y acción segura `observe_goal`.
- El goal escribió el checkpoint temprano:

```text
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/02_temas/tema_001/coordinacion_wave10_phase1/checkpoint_started.txt
```

Fallo residual:

- A los 75 segundos aproximados, sin crear los otros artefactos de fase 1,
  `autoprogramming/status` pasó a:
  - `queue.terminal.status=stopped`;
  - `queue_health.blocked=1`;
  - `goal_first_blocked`;
  - `goal_backend_state_unreconciled`;
  - `goal_first_blocked_no_artifacts`;
  - `evidence-ref-codex-app-server-goal-active-timeout`.
- El mensaje `goal_first_blocked_no_artifacts` vuelve a ser incorrecto porque
  sí existía `checkpoint_started.txt`.
- `POST /api/v0/autoprogramming/goal/observe` siguió agotando ventana HTTP con
  `autoprogramming_observe_goal_timeout`.
- `POST /api/v0/server/shutdown` devolvió `shutdown_ready=true`, pero dejó vivo
  el `codex app-server` asociado al socket:

```text
/home/alberto/Trabajo/OPES/.orquesta-runtime/goal-srv/g-fe4c6324af0f5efc.sock
```

OPES cerró manualmente ese `codex app-server` para no dejar procesos activos.

Evidencia local:

```text
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_payloads/wave10_phase1_t001/external_work_wave10_phase1_t001.json
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_responses/wave10_phase1_t001/
  response_t001.json
  observe_active_initial.json
  observe_goal_initial.json
  status_after_observe_active.json
  status_after_30s.json
  status_after_75s.json
  shutdown_after_blocked.json
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/02_temas/tema_001/coordinacion_wave10_phase1/checkpoint_started.txt
```

Acción esperada:

- El timeout por goal OPES no debe cortar una fase de planificación tras unos
  75 segundos si el backend acaba de escribir checkpoint y no hay evidencia de
  bucle.
- `goal_first_blocked_no_artifacts` debe contar como artefacto parcial el
  checkpoint escrito dentro del write-set.
- `observe_goal` debe devolver snapshot parcial rápido del goal bloqueado, no
  timeout opaco.
- `shutdown_ready=true` no debe emitirse si queda vivo el `codex app-server`
  propio del goal.

No se ha tocado código del núcleo de Orquesta. Esta sección queda para el
agente que está corrigiendo Orquesta.

### 19. Ola 8: tarea mínima con artefactos de fase 0, pero API la proyecta como `goal_first_blocked_no_artifacts`

Fecha observada: 2026-07-01 13:03-13:09 Europe/Madrid.

Contexto:

- Se reintentó con la última versión disponible de Orquesta:
  `0b3e9966`; `git fetch --prune origin` mostró `HEAD...@{u}=0 0`.
- Para aislar el problema de tamaño de tarea, se lanzó una fase 0 mínima para
  el tema `001` de Grupo B Informática.
- El encargo prohibía redactar el tema completo y pedía solo cuatro artefactos
  mínimos:
  - `checkpoint_started.txt`
  - `matriz_fuentes_reutilizacion.md`
  - `plan_rework_por_fases.md`
  - `docs/orquesta_goal_result_v0.json`
- `run_ref`:
  `run-opes-grupo-b-info-wave8-phase0-checkpoint-t001-20260701`.
- `external_goal_ref`:
  `019f1d59-24d0-7d30-88a1-da10e7a6ffb9`.

Resultado positivo revisado:

- Orquesta sí escribió el checkpoint temprano y, tras revisar el disco, también
  materializó los artefactos mínimos de fase 0:

```text
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/02_temas/tema_001/coordinacion_wave8/checkpoint_started.txt
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/02_temas/tema_001/coordinacion_wave8/matriz_fuentes_reutilizacion.md
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/02_temas/tema_001/coordinacion_wave8/plan_rework_por_fases.md
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/02_temas/tema_001/coordinacion_wave8/docs/orquesta_goal_result_v0.json
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/02_temas/tema_001/coordinacion_wave8/orquesta_phase0_checkpoint_delivery.json
```

Problema:

- La API pública siguió proyectando la tarea como bloqueada/sin artefactos
  aunque existían cinco ficheros de fase 0 en el write-set.
- `docs/orquesta_goal_result_v0.json` declara `status=complete`, pero sus
  `required_test_results` siguen con `evidence_refs=[]`; por tanto sirve como
  entrada de planificación OPES, no como cierre contractual pleno.
- `POST /api/v0/autoprogramming/status` informó:
  - cola terminal con `status=stopped`;
  - `goal_first_blocked`;
  - `goal_backend_state_unreconciled`;
  - `goal_first_blocked_no_artifacts`;
  - `overall_percentage=40`;
  - `active_runs=0`.
- La consulta directa a `goals_1.sqlite` seguía mostrando el goal como
  `active`, con `tokens_used=183929` y `time_used_seconds=192`.
- `POST /api/v0/autoprogramming/goal/observe` devolvió:

```json
{
  "estado": "error",
  "errores_publicos": [
    {
      "code": "autoprogramming_observe_goal_timeout",
      "field": "executor",
      "message": "autoprogramming observe_goal excedio la ventana HTTP acotada"
    }
  ]
}
```

- `POST /api/v0/autoprogramming/goals/observe-active` volvió a devolver solo:

```json
{"estado":"ok"}
```

- `POST /api/v0/server/shutdown` devolvió `shutdown_ready=true`, pero quedaron
  procesos `codex app-server` vivos. OPES los cerró manualmente. La SQLite
  siguió mostrando el goal como `active` y alcanzó `tokens_used=206492` y
  `time_used_seconds=375` antes del cierre forzado.

Evidencia local:

```text
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/ESTADO_WAVE8_PHASE0_GOAL_FIRST_BLOCKED_2026-07-01.md
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_responses/wave8_phase0_checkpoint_t001/
  response_t001.json
  autoprogramming_status_after_blocked.json
  observe_goal_after_blocked.json
  observe_active_after_blocked.json
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/02_temas/tema_001/coordinacion_wave8/
  matriz_fuentes_reutilizacion.md
  plan_rework_por_fases.md
  orquesta_phase0_checkpoint_delivery.json
  docs/orquesta_goal_result_v0.json
```

Acción esperada:

- La cola y el backend deben reconciliar el estado: no puede quedar
  `queue stopped/blocked` mientras `thread_goals.status` sigue `active`.
- Si hay artefactos parciales, `goal_first_blocked_no_artifacts` no debería
  ocultarlos: debe indicar qué se escribió, qué faltó y qué replan recomienda.
- `observe_goal` no debe intentar observar indefinidamente un run que la cola ya
  considera terminal.
- `observe-active` debe devolver estado útil para refs solicitados, aunque sean
  terminales o bloqueados.
- Un objetivo mínimo con checklist de archivos debe poder cerrarse sin LLM si
  ya se conocen los artefactos esperados, o declarar `partial_artifacts` con
  rework causal.
- `shutdown_ready=true` no debe emitirse mientras queden vivos procesos del
  `codex app-server` asociado al goal o mientras el backend siga consumiendo
  tokens.

No se ha tocado código del núcleo de Orquesta. Esta sección queda para el
agente que está corrigiendo Orquesta.

### 18. Ola 7: tras volver cuota, los goals consumen tokens pero acaban en `goal_active_timeout` sin artefactos

Fecha observada: 2026-07-01 12:56-13:02 Europe/Madrid.

Contexto:

- Se reintentó con Orquesta en la misma versión más reciente disponible:
  `0b3e9966`; `git fetch --prune origin` mostró `HEAD...@{u}=0 0`.
- Se lanzaron solo dos temas como smoke controlado tras el corte de cuota:
  `001` y `002`.
- Los `run_ref` nuevos fueron:
  - `run-opes-grupo-b-info-wave7-quota-retry-t001-20260701`
  - `run-opes-grupo-b-info-wave7-quota-retry-t002-20260701`
- Esta vez los goals sí quedaron `active` y consumieron tokens, por lo que el
  problema de cuota inmediata desapareció.

Evidencia positiva:

- `GET /api/v0/routes` funcionó.
- `POST /api/v0/external-work/run` devolvió `estado=ok` para ambos.
- `POST /api/v0/autoprogramming/status` devolvió telemetría mejor que en wave6:
  `queue_health.running_live=2`, safe actions y `goal_status=running`.
- `POST /api/v0/autoprogramming/goal/observe` devolvió
  `summary=codex_app_server_goal_status_active`.

Problema:

- Los dos goals consumieron muchos tokens sin escribir ningún artefacto en
  `02_temas/tema_001` ni `02_temas/tema_002`.
- Último estado observado en `goals_1.sqlite` antes de cortar:
  - tema 001: `status=active`, `tokens_used=338171`, `time_used_seconds=226`
  - tema 002: `status=active`, `tokens_used=304661`, `time_used_seconds=193`
- `POST /api/v0/autoprogramming/status` acabó devolviendo:
  - `goal_first_blocked`
  - `goal_backend_state_unreconciled`
  - `evidence-ref-codex-app-server-goal-active-timeout`
  - `goal_first_blocked_no_artifacts`
  - `overall_percentage=40`
- La cola marcó los runs como terminales `stopped`, pero el backend
  `thread_goals` seguía en `active`, sin reconciliación.
- `shutdown_ready=true` volvió a dejar procesos residuales `codex app-server`
  (`node ... codex app-server` y binario vendor). OPES tuvo que matar PIDs
  manualmente con `kill` y luego `kill -9` sobre el residual.

Evidencia local:

```text
opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/orquesta_responses/wave7_retry_after_quota_smoke_001_002/
  response_t001.json
  response_t002.json
  observe_active_initial.json
  observe_goal_t001_1.json
  observe_goal_t002_1.json
  autoprogramming_status_1.json
  shutdown_after_active_timeout.json
```

Acción esperada:

- Si un goal supera un umbral de tiempo/tokens sin artefactos, Orquesta debe
  cortar con una causa clara y conservar un snapshot operativo: últimos
  mensajes seguros, ruta de trabajo, archivos abiertos, tareas pendientes y
  recomendación de replan.
- La cola y el backend deben reconciliar estado: no puede quedar
  `queue stopped/blocked` mientras `thread_goals` sigue `active`.
- `shutdown_ready=true` no debe emitirse si queda vivo el app-server propio, o
  debe devolver `backend_still_running` con PIDs y acción recomendada.
- El estado `goal_first_blocked_no_artifacts` debe impedir relanzar más padres
  en paralelo hasta que el director replanifique; esto funcionó como señal, pero
  llegó después de un consumo alto de tokens.

No se ha tocado código del núcleo de Orquesta. Esta sección queda para el
agente que está corrigiendo Orquesta.

### 19. OPES local BUG-096: QA estricta debe detectar contaminación estructural, no solo patrones de andamiaje

Fecha observada: 2026-07-02.

Nota de inventario: esta observación local queda relacionada con
`BUG-ORQ-20260701-093` y `BUG-ORQ-20260702-095` en el inventario central.

Contexto:

- Se revisó Grupo B Informática en
  `/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional`.
- Se usaron seis subagentes de auditoría sobre temas 007, 013, 014, 015, 017 y
  046, sin editar Orquesta core.
- El informe global tras cerrar tema 007 queda en 18/50 temas verdes estrictos
  y 142 hallazgos de andamiaje pendientes.

Problema:

- Los validadores existentes pueden dar `pass` de extensión, metacomentarios y
  notas de autor aunque el tema esté inflado con bloques de otros temas.
- Los temas 013, 014, 015, 017 y 046 requieren rework estructural: contienen
  bloques pegados de arquitectura, cloud, administración electrónica, gestión
  documental, ciberseguridad, temas ajenos, fuentes incorrectas y referencias
  internas a canones o derivaciones.
- El tema 007 solo pudo cerrarse tras cortar el bloque ajeno final y sustituirlo
  por expansión propia de firma electrónica y servicios de confianza.

Evidencia local:

```text
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/00_control/scripts/cleanup_tema_007_firma_confianza.py
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/09_validacion/informe_avance_estricto_grupo_b_informatica_2026-07-01.md
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/09_validacion/informe_texto_publico_sin_andamiaje_interno.md
```

Acción esperada:

- Añadir al cierre OPES/Orquesta un control `structural_topic_coherence_pass`.
- Medir procedencia de bloques y similitud título-cuerpo antes de aceptar
  extensión.
- Si un tema supera palabras por injertos ajenos, marcar
  `pendiente_rework_editorial_estructural`, conservar insumos útiles y lanzar
  reexpansión propia del título.
- No permitir RAG, tests, HTML ni audios desde textos con contaminación
  estructural aunque pasen `no_exam_meta` y `no_author_notes`.

No se ha tocado código del núcleo de Orquesta. Esta sección queda para el
agente que está corrigiendo Orquesta.
