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

## No se ha tocado código

Esta incidencia solo documenta el problema para el agente de Orquesta. OPES no
ha modificado el núcleo.

## Prioridad

Alta para autonomía OPES: sin observe parcial útil, estados por tema y guardas
de mínimos, Orquesta puede aparentar trabajo activo mientras no permite al
director saber si debe esperar, replanificar o cortar.
