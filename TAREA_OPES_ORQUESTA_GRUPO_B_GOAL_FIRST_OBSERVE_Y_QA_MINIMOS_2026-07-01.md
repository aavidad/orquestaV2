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

## No se ha tocado código

Esta incidencia solo documenta el problema para el agente de Orquesta. OPES no
ha modificado el núcleo.

## Prioridad

Alta para autonomía OPES: sin observe parcial útil, estados por tema y guardas
de mínimos, Orquesta puede aparentar trabajo activo mientras no permite al
director saber si debe esperar, replanificar o cortar.
