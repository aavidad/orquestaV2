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

### 19. Ola 8: tarea mínima con checkpoint escribe un fichero, pero queda `goal_first_blocked` y backend `active`

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

Resultado positivo:

- Orquesta sí escribió el primer checkpoint temprano:

```text
/home/alberto/Trabajo/OPES/opes-salidas/orquesta_real/grupo_b_informatica_2026-07-01/rework_profesional/02_temas/tema_001/coordinacion_wave8/checkpoint_started.txt
```

Problema:

- No se materializaron los otros tres artefactos mínimos.
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
