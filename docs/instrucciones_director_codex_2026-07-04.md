# Instrucciones para el director Codex (relevo de Claude) — 2026-07-04

Estado final 2026-07-04 tras relevo Codex: documento historico ejecutado.
No usar como plan vivo sin leer antes la seccion final de
`docs/bitacora_correccion_pericial_2026-07-03.md`.

Resumen de cierre:
- T294 integrado en `7a6dea0d`: stop/cancel forzado goal-first propaga al
  backend tmux y `observe_goal` respeta evidencia terminal forzada.
- T295 cerrado en `6fe19d06`: el planner no deja que el scanner idle sustituya
  una tarea ejecutable por interpretar `Dependencias: ninguna` como pendiente.
- T296 integrado en `67dd7fa9`: shutdown expone `goal_actions` y relectura de
  active work para backends Goal.
- BUG-166/167/168 quedaron cerrados en `0e0dcedc`; Sueldos accepted queda como
  validacion de campo anterior a este relevo.
- Actualizacion posterior: `BUG-073` queda cerrado por reejeucion real del
  smoke high-consumption; MEJ-104 queda cerrado tambien por smoke real acotado
  de `budget_deferred`; `BUG-075` queda reducido por rework causal de evidencia
  minima OPES faltante.
- Actualizacion noche Codex: `BUG-075` cubre ya la matriz terminal sin evidencia
  minima para la secuencia OPES completa; `BUG-079` tiene corte de ingesta
  `thread/read` > 256 KiB en app-server; `BUG-165` tiene timeout interno del
  observador residente con `goal_observer_timeout`, sin env nueva por ratchet
  MEJ-106.
- Queda pendiente no cerrado total: `BUG-165` para timeouts amplios
  status/observe y coordinacion real completa, `BUG-065/076` para smoke amplio,
  `BUG-058/066` para lifecycle OPES end-to-end, `BUG-075` para validadores OPES
  semanticos/editoriales por artefacto canonico y smoke OPES temporal, y
  `BUG-079` para enforcement runtime/proveedor de checkpoint temprano y salidas
  gigantes.

Eres el DIRECTOR del proyecto Orquesta. Claude queda fuera por límite de
tokens. Este documento es autocontenido: síguelo al pie de la letra y no
improvises fuera de él. Idioma de trabajo: español. Registra TODO avance en
`docs/bitacora_correccion_pericial_2026-07-03.md` (añade secciones al final,
no borres nada).

## 0. Reglas de oro (orden del operador)

1. Paraleliza todo lo que se pueda (es contrato de la app, ver `AGENTS.md`).
2. No lances tareas de automejora nuevas fuera de las dos misiones de abajo:
   la cola de automejora está CONGELADA por orden del operador.
3. Trabaja por eventos: lanza, observa con esperas acotadas, no hagas polling
   agresivo. Los endpoints de Orquesta son POST-only (GET devuelve
   `metodo_no_permitido`).
4. No toques `master`. La rama de trabajo es `trabajo/plataforma-agentes`.
5. Hay ficheros MODIFICADOS SIN COMMITEAR en el repo principal (fixes de
   BUG-167/168 de otra sesión): no los pises, no hagas `git checkout --` ni
   `git stash` en `/home/alberto/Trabajo/orquesta` salvo sobre ficheros que tú
   mismo hayas tocado.

## 1. Misión A — Integrar el pilotaje T294 (fix de BUG-ORQ-20260704-165)

Ya hay un pilotaje self-programming CORRIENDO que programa el arreglo de
BUG-165 (stop forzado que no llega al backend tmux + observe que sigue en
`running`). Rutas:

- Raíz del pilotaje: `/tmp/claude-1000/-home-alberto-Trabajo-orquesta/8be66426-b9e9-428a-a040-c4ee62477495/scratchpad/pilot-t294`
  (abajo `$PILOT`).
- Worktree git (rama `pilot-t294`): `$PILOT/worktree`
- URL del servidor: fichero `$PILOT/base_url.txt` (formato `base_url=http://127.0.0.1:PUERTO`)
- PID del servidor: `$PILOT/server.pid`; logs en `$PILOT/logs/`
- Backlog mínimo con la tarea: `$PILOT/worktree/docs/autoprogramacion_orquesta_pendientes_2026-05-23.md` (sección `## T294`).

### 1.1 Observar hasta terminar

Comprueba cada 2-3 minutos (máximo 60 minutos en total):

```sh
find $PILOT/worktree -name 'orquesta_goal_result_*t294*.json' -not -path '*/.git/*'
```

- Acepta SOLO un result cuyo `task_ref`/nombre case exactamente con `t294` y
  cuyo `status` sea `complete`, `blocked` o `rework`. Ignora placeholders
  `invalid` con summary `work_in_progress`.
- También puedes consultar estado (POST con body JSON `{}`):
  `curl -sS -X POST -H 'Content-Type: application/json' -d '{}' $BASE_URL/api/v0/autoprogramming/status`
- Si a los 60 minutos no hay result terminal: mata el pilotaje (ver 1.4),
  anota `timeout` en la bitácora y NO reintentes más de una vez.

### 1.2 Validar en el worktree

Cuando haya result terminal `complete`:

```sh
cd $PILOT/worktree
GOFLAGS=-buildvcs=false go test ./modulos/orquesta-runtime-codex-appserver \
  ./modulos/orquesta-app-director-service ./modulos/orquesta-app-codex-stack \
  ./modulos/orquesta-orchestration-core
```

Revisa el diff (`git -C $PILOT/worktree diff --stat`) y verifica que el cambio
hace lo que pide la sección T294 del backlog: propagar stop al backend,
estado terminal único `blocked` con evidencia, observe no devuelve `running`
tras stop forzado, y código accionable si la propagación falla. Si el result
es `blocked`, lee su JSON, apunta la causa en la bitácora y decide: si hay
código útil y los tests pasan, continúa igual; si no, cierra como fallido.

### 1.3 Integrar en la rama de trabajo

```sh
git -C $PILOT/worktree checkout -- docs/autoprogramacion_orquesta_pendientes_2026-05-23.md
git -C $PILOT/worktree add -A && git -C $PILOT/worktree commit -m "fix: propagar stop forzado goal-first al backend tmux (T294, BUG-165)"
git -C /home/alberto/Trabajo/orquesta cherry-pick <sha-del-commit>
```

- Conflictos: resuélvelos por UNIÓN (los cambios suelen ser aditivos);
  conserva SIEMPRE lo que ya había en la rama de trabajo.
- Después, en el repo principal, repite las mismas suites de 1.2. Si pasan,
  commit queda hecho; NO hagas push salvo que el operador lo pida.
- Actualiza `docs/inventario_bugs_orquesta_2026-06-30.md`: fila
  `BUG-ORQ-20260704-165` → estado según lo conseguido (cerrado o avance
  parcial con lo que falte), citando el test focal nuevo.

### 1.4 Limpieza del pilotaje (SIEMPRE, pase lo que pase)

```sh
kill "$(cat $PILOT/server.pid)" 2>/dev/null
tmux ls 2>/dev/null | grep -o 'orquesta-goal-[^:]*' | xargs -r -n1 tmux kill-session -t
pkill -f 'codex app-server' 2>/dev/null
git -C /home/alberto/Trabajo/orquesta worktree remove --force $PILOT/worktree  # solo tras integrar
git -C /home/alberto/Trabajo/orquesta branch -D pilot-t294                      # solo tras integrar
```

## 2. Misión B — Test de campo Sueldos (autonomía extremo a extremo)

Sueldos es una APP DE PRUEBA: no importa la app, importa verificar que
Orquesta termina un goal-first real sin intervención manual. Lanza esto
DESPUÉS de integrar la Misión A (para probar el fix) o en paralelo si la A va
lenta.

### 2.1 Arrancar servidor temporal aislado

```sh
ROOT=/home/alberto/Trabajo/Sueldos/.orquesta-feature-cargos-2
mkdir -p $ROOT/state $ROOT/runtime $ROOT/logs
cd /home/alberto/Trabajo/orquesta && GOFLAGS=-buildvcs=false go build -o $ROOT/orquesta-server ./cmd/orquesta-server
env ORQUESTA_SERVER_ADDR=127.0.0.1:0 \
  ORQUESTA_SERVER_STATE_DIR=$ROOT/state \
  ORQUESTA_CODEX_RUNTIME_WORKDIR=$ROOT/runtime \
  ORQUESTA_CODEX_PROJECT_WORKDIR=/home/alberto/Trabajo/Sueldos \
  ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux \
  ORQUESTA_CODEX_CODE_HOME=$HOME/.codex \
  ORQUESTA_CODEX_COMMAND="$(command -v codex)" \
  ORQUESTA_CODEX_APPROVAL_POLICY=never \
  ORQUESTA_CODEX_SANDBOX=workspace-write \
  ORQUESTA_CODEX_GOAL_TIMEOUT_MS=2700000 \
  ORQUESTA_AUTOPROGRAMMING_CHECKPOINT_ONLY_HIGH_CONSUMPTION_TOKENS=450000 \
  ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_DISABLED=true \
  $ROOT/orquesta-server run >$ROOT/logs/stdout.log 2>$ROOT/logs/stderr.log &
echo $! > $ROOT/server.pid
```

La URL sale en `$ROOT/state/orquesta_server_state_v0.json` (campo `addr`).
El umbral de 450000 tokens evita el corte por alto consumo que mató el
intento anterior a 183001 tokens.

### 2.2 Lanzar el goal (request EXACTO)

```sh
curl -sS -X POST -H 'Content-Type: application/json' "$BASE_URL/api/v0/apps/director" -d '{
  "request_id": "request-ref-sueldos-cargos-partidos-20260704-005",
  "director_execution_mode": "goal_first",
  "app_spec_request": {
    "schema_version": "app_spec_request.v0",
    "request_id": "req-sueldos-cargos-partidos-20260704-005",
    "source": "orquesta-mcp",
    "locale": "es-ES",
    "nombre": "Mapa de gasto publico",
    "objetivo": "Ampliar la app existente generated-apps/mapa-de-gasto-publico con cargos politicos y funcionarios publicos: retribuciones con fuente auditada, partido politico asociado por zona/entidad cuando la fuente lo permita, integrado con el mapa de gasto publico ya entregado. La app debe compilar, pasar go test ./... dentro del scope y actualizar handoff_report.md, source_tree.md y technical_stack_manifest.md.",
    "tipo_app": "web",
    "datos": { "db_required": true, "storage": [ { "tipo": "mixta" } ] }
  }
}'
```

Guarda del JSON de respuesta `run_ref` y `goal_ref`.

### 2.3 Observar y criterio de éxito

Observa cada 3-5 minutos (máximo 50 minutos):

```sh
curl -sS -X POST -H 'Content-Type: application/json' \
  -d '{"run_ref":"<RUN_REF>"}' "$BASE_URL/api/v0/apps/director/goal/observe"
```

- ÉXITO: `estado=ok`, `goal_status=complete`, `closure_status=accepted` y
  receipt durable nuevo bajo
  `generated-apps/mapa-de-gasto-publico/docs/orquesta_goal_result_*.json`
  con `goal_ref` del run NUEVO (no del antiguo `...43c20...`).
- Si publica `high_token_usage` + stop y luego observe sigue `running` con
  tmux vivo: eso es EXACTAMENTE BUG-165. Si ya integraste la Misión A, este
  caso no debería repetirse; si se repite, anótalo con todos los refs en el
  inventario (reabre la fila) y limpia a mano.
- Cualquier desviación (mezcla de refs antiguos, snapshot failed, timeouts de
  status): anótala en el inventario de bugs con request/run/goal refs.

### 2.4 Limpieza

Igual que 1.4 pero con `$ROOT/server.pid`. Si el servidor no muere con el
PID, usa SIGINT y comprueba que no quedan `orquesta-server`, `codex
app-server` ni sesiones tmux `orquesta-goal-*`.

## 3. Informe final al operador

Al terminar (o si te bloqueas), escribe en la bitácora una sección
`## Relevo director Codex 2026-07-04` con: qué se integró (SHAs), resultado
de las suites, estado de BUG-165 en el inventario, resultado del test de
campo Sueldos (refs + veredicto), procesos limpiados, y qué queda pendiente.
El operador solo lee ese resumen: hazlo corto y completo.

## 4. ACTUALIZACION 2026-07-04 ~06:30 — tres pilotajes corriendo, toma el relevo

La Mision B (Sueldos) YA ESTA CERRADA `accepted`. La via idle+escaner NO
funciona (regresion): usa SIEMPRE `POST $BASE/api/v0/autoprogramming/prepare-run`.
Hay TRES goals corriendo lanzados por prepare-run; supervisalos e integralos
(protocolo §1.2-§1.4, mismas rutas `$S=/tmp/claude-1000/-home-alberto-Trabajo-orquesta/8be66426-b9e9-428a-a040-c4ee62477495/scratchpad`):

| Pilotaje | Worktree/rama | base_url | run_ref | Tarea |
|---|---|---|---|---|
| pilot-t294 | $S/pilot-t294/worktree | fichero base_url.txt | request-ref-t294-stop-forzado-prepare-20260704-001 | fix BUG-165 (stop forzado→backend, terminal unico) |
| pilot-t295 | $S/pilot-t295/worktree | fichero base_url.txt | request-ref-t295-scanner-noop-20260704-001 | regresion escaner backlog (idle no ejecuta tareas) |
| pilot-t296 | $S/pilot-t296/worktree | fichero base_url.txt | request-ref-t296-shutdown-coord-20260704-001 | BUG-065/076 shutdown coordinado backend |

Para cada uno: espera result durable terminal (find worktree -name
'orquesta_goal_result_*.json' con status complete/blocked, ignora placeholders
work_in_progress), valida suites del write-set, cherry-pick a
`trabajo/plataforma-agentes`, actualiza inventario de bugs, limpia (server.pid,
tmux orquesta-goal-*, codex app-server del pilotaje). Tras integrar los tres:
quedan abiertos 058, 066, 073, 075, 079 — lanza un prepare-run por bug con
write-sets disjuntos (plantilla: los requests de arriba estan en la bitacora),
integra y cierra inventario. Objetivo del operador: Orquesta y conectores al
100%. Autorizado por el operador 2026-07-04: "si hazlo todo".

## 5. COLA DE TAREAS AUTORIZADA POR EL OPERADOR (2026-07-04 tarde)

Orden textual del operador: "dale todas las tareas a codex para que lo
programe". Esto DESCONGELA la cola solo para las tareas de esta seccion.
Ejecutalas via `POST /api/v0/autoprogramming/prepare-run` (goal_first) en
pilotajes aislados, PARALELIZANDO las que tienen write-sets disjuntos.
Protocolo de integracion: suites del write-set en el worktree -> commit ->
cherry-pick a `trabajo/plataforma-agentes` -> suites en main -> actualizar
inventario/bitacora -> limpiar procesos. Umbral consumo 450000, timeout 30m,
checkpoint temprano SIEMPRE como primera accion.

### TAREA-1 (prioridad maxima): smoke real acotado MEJ-104 + activar presupuesto

- Ejecutar el smoke real del gobernador de presupuesto con presupuesto bajo:
  `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_DAILY_GOAL_BUDGET=2` y
  `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_DAILY_CONTEXT_BUDGET_BYTES=262144`
  en un pilotaje aislado con backlog minimo de UNA tarea trivial.
- Verificar `budget_deferred`/`budget_degraded` en `autoprogramming/status`
  cuando se agota, y que el dia siguiente (simulado) repone cupo.
- Si el smoke va bien: documentar en el runbook los valores recomendados de
  produccion y dejar nota en el plan MEJ-104 como CERRADO.
- Write-set: cmd/orquesta-server, modulos/orquesta-autoprogramming, docs.

### TAREA-2: broker de contexto expuesto como herramienta MCP del agente

- Hallazgo previo: el sandbox workspace-write bloquea HTTP a localhost, asi que
  los agentes no pueden consultar el broker por red (experimento A/B 2026-07-04,
  brazo B bloqueado con socket Operation not permitted).
- Exponer `CodeContextQueryPortV0` como tool MCP en el codex-home del goal
  (toolbelt), de modo que el agente consulte contexto (rg + codebase-memory)
  sin red y sin contexto crudo inflado.
- Medir: tokens_used de un goal de documentacion CON tool vs linea base 164945
  (brazo A, tarea T293). Criterio de exito del experimento: >=15% menos.
- Write-set: modulos/orquesta-context, modulos/orquesta-runtime-codex-appserver,
  cmd/orquesta-server, docs.

### TAREA-3: enrutado por coste en el director de escalada

- Las tareas solo-documentales (escaneos, docs, inventarios) deben enrutarse a
  proveedor/effort barato (p.ej. gemini o codex con reasoning_effort=low), y
  las de codigo al perfil actual. Evidencia: el escaner de backlog gasto
  150k-210k tokens en ciclos documentales.
- Anadir al goal spec una pista `task_cost_class` (doc|code|mixed) derivada del
  write-set (solo .md => doc) y cablearla a la seleccion de backend/effort.
- Write-set: modulos/orquesta-runtime-codex-goal, cmd/orquesta-server, docs.

### TAREA-4: coordinacion automatica completa de shutdown (residual BUG-065/165)

- Smoke real amplio: servidor con 2 goals activos, shutdown cooperativo debe
  encadenar backend/checkpoint/stop/cancel/wait sin intervencion y publicar
  `goal_actions` resueltas; ningun proceso residual.
- Si el smoke revela huecos, fix focal (no refactor amplio) y test.
- Write-set: modulos/orquesta-server-shutdown, cmd/orquesta-server, docs.

### TAREA-5: residuales OPES 058/066/075

- Cerrar el contrato de calidad OPES pendiente: settlement durable, cierre con
  evidencia minima y QA de artefactos parciales, segun las filas del
  inventario. Solo tests focales + fixes acotados.
- Write-set: modulos/orquesta-opes-director, modulos/orquesta-opes-connector,
  docs.

### TAREA-6: ratchet de deuda de configuracion

- La pericial fijo la vigilancia: env_vars 500 -> hoy 511. Anadir test ratchet
  que falle si `env_vars_orquesta` supera el valor actual (511) sin actualizar
  explicitamente el limite con justificacion en el propio test.
- Write-set: cmd/orquesta-server (o modulo de registry de envs), scripts, docs.

Orden sugerido: TAREA-1 sola primero (toca la cola idle). Despues 2+3 en
paralelo (eficiencia) y 4+5 en paralelo (residuales). TAREA-6 en cualquier
hueco. Al terminar todo: informe en bitacora con SHAs y estado del inventario;
los 6 bugs abiertos deberian quedar en 0-2 filas con residual justificado.

### TAREA-7: wizard de programacion conversacional (orden del operador 2026-07-04)

Peticion textual del operador: "necesito un wizard de programacion que me
consulte para llegar a una app completa, no solo un formulario. Debe dar
siempre las mejores opciones (incluso con sugerencias) y preguntarme por los
huecos que no he especificado".

Base existente (NO partir de cero): sesion de intake guiado en
`modulos/orquesta-web/nueva_app_intake_session_v0.go` con `PendingQuestions`
y `RecommendedQuestions` (hoy: listas planas de nombres de campo). El wizard
la evoluciona:

1. **Pregunta rica por hueco**: cada turno devuelve, por cada hueco detectado,
   un objeto pregunta con: texto de la pregunta (i18n es/en), por que importa,
   y 2-4 OPCIONES concretas derivadas de los catalogos ya validados por la
   factory (tipo_app, datos.storage.tipo, deploy.target, calidad, i18n...),
   con UNA marcada `recommended=true` y su justificacion breve. Siempre se
   acepta ademas texto libre.
2. **Sugerencias proactivas**: el wizard propone la mejor opcion para cada
   hueco segun el contexto ya respondido (ej.: tipo_app=web + datos con
   fuentes publicas -> storage `mixta` recomendado). Debe existir la accion
   "aceptar todas las recomendaciones" que completa el spec de una vez.
3. **Huecos cruzados, no solo campos vacios**: reglas de coherencia que
   pregunten cuando falta algo implicito: db_required sin storage;
   integraciones declaradas sin auth/criticidad; app movil sin plataformas;
   objetivo que menciona datos externos sin fuentes; deploy sin target
   compatible con tipo_app. Cada regla con test focal.
4. **Cierre verificable**: el wizard termina cuando el `AppSpecRequestV0`
   compuesto pasa la validacion de la factory Y no quedan huecos de
   importancia alta. Entonces ofrece resumen final + lanzamiento directo por
   `/api/v0/apps/director` (goal_first) reutilizando el contrato existente.
5. **Superficies**: endpoint HTTP POST (evolucion del intake guiado actual),
   tool MCP equivalente, y render web usable. Todo i18n por catalogo con test
   de propiedad (patron `nueva_app_i18n_owner_v0_test.go`).

Write-set: modulos/orquesta-web, modulos/orquesta-app-director-intake,
modulos/orquesta-mcp (tool nueva), modulos/orquesta-app-gateway (ruta),
modulos/orquesta-i18n-docs, docs. DISJUNTO de TAREA-1..6: puede ir en
paralelo desde ya.
Tests: go test ./modulos/orquesta-web ./modulos/orquesta-app-director-intake
./modulos/orquesta-mcp ./modulos/orquesta-app-gateway
Criterio de aceptacion global: una sesion de wizard simulada en test parte de
solo `nombre+objetivo` y llega a spec completo valido en <=6 turnos usando
recomendaciones; y una segunda sesion detecta al menos 3 huecos cruzados de
los listados arriba.

#### TAREA-7b: refinamiento del operador (2026-07-04, vinculante)

Ejemplo canonico del operador (usarlo como test de aceptacion literal):
entrada inicial "quiero una app para una agenda" y NADA mas. El wizard debe
preguntar, entre otras: ¿movil, PC o ambas?; ¿personal o para compartir?;
¿integrarla con la agenda de la empresa (y cual: Google/Microsoft/CalDAV)?;
y asi con cada dimension relevante hasta poder terminar la app completa.
Anadir test `TestWizardAgendaDesdeSoloObjetivoV0` que parta de ese texto y
verifique que el wizard genera esas familias de preguntas con opciones.

Reglas vinculantes adicionales:

1. **La recomendacion se muestra SIEMPRE, incluso contra la eleccion del
   usuario**: si el usuario elige una opcion distinta de la recomendada, la
   respuesta del turno debe conservar `recommended` + `rationale` visibles
   junto a la eleccion registrada (campo `user_choice` vs `recommended`).
   Nunca se sobreescribe la eleccion del usuario, pero nunca se oculta cual
   era la mejor opcion y por que. Test focal de este contraste.
2. **Defaults de ingenieria silenciosos**: si el usuario no dice nada al
   respecto, el spec compuesto SIEMPRE incluye: arquitectura hexagonal
   (puertos/adaptadores), i18n es/en por catalogo, tests unitarios +
   arquitectura (ratchet), linters/format, manejo de errores tipado con
   catalogo publico, logging estructurado, CI-ready (make/scripts de
   verificacion), y documentacion tecnica (handoff, source_tree, stack
   manifest). Estos defaults se declaran en `preferencias_tecnicas` y
   `calidad` del AppSpecRequestV0 sin preguntar, y el resumen final los lista
   en una seccion "decisiones tomadas por ti" para transparencia. Solo se
   pregunta si el usuario contradice un default explicitamente.
3. Las preguntas se agrupan por turnos tematicos (max 3-4 preguntas por
   turno) para no abrumar: primero uso/plataforma, luego datos/integraciones,
   luego despliegue/calidad. El flujo del ejemplo agenda debe caber en <=6
   turnos.

### TAREA-8: configuración canónica por fichero y reducción de envs (orden del operador 2026-07-04)

Motivación: el fallo T7A (umbral 450000 pedido, `100000 defaulted` efectivo)
es una clase entera de bugs: la config depende de heredar cientos de envs
entre shells/daemons y un valor puede perderse en silencio. Orden: que no
vuelva a pasar, y no tener 500 variables.

8.1 Fichero de configuración canónico:
- `orquesta.config.toml` (o json) con TODAS las opciones tipadas por
  secciones (server, goal_backend, autoprogramming, budgets, observability,
  ...). `orquesta-server run|start --config <fichero>`.
- Las envs quedan SOLO como override puntual y para secretos/rutas; el
  daemon/start SIEMPRE recibe el fichero resuelto, nunca depende de heredar
  el entorno del shell padre (esta es la causa raíz de T7A).
- Al arrancar, el servidor materializa `effective_config` completo en el
  state file con ORIGEN por setting: `file|env|default`. Ya existe parcial;
  hacerlo total.

8.2 Guard de lanzamiento (que no vuelva a pasar):
- `prepare-run`/`apps/director` aceptan `required_settings` opcionales
  (lista clave=valor críticos para ese run, ej. el umbral de consumo). Si la
  config efectiva difiere, el lanzamiento se RECHAZA con
  `config_projection_mismatch` y el detalle esperado-vs-efectivo. Test focal.
- El smoke nightly añade fase que verifica eco de config crítica.

8.3 Ratchet bidireccional de envs:
- Test que falla si: (a) una env registrada en server_env_registry no la
  consume ningún loader, (b) un loader lee una env no registrada, (c) el
  conteo sube sobre el techo actual sin marca explícita (ya existe el
  ratchet de conteo; añadir a+b).

8.4 Reducción por olas (meta: <100 envs reales):
- Clasificar las 511 en: CORE operativa (~30, siguen como env+fichero),
  TUNING (pasan a secciones del fichero; env deprecada con alias y aviso
  `deprecated_env_used` durante 2 versiones), SOLO-PILOTO/SMOKE (pasan a
  perfiles, ver 8.5), MUERTAS (borrar ya con evidencia de no-uso via rg).
- Ola 1 definida por docs/auditoria_envs_pisadas_2026-07-04.md (3 variantes codex home, 2 OPES base URL, unidades de timeout, familia GUARDIAN fuera de registro). Una ola por PR, con tabla antes/después en el inventario y el ratchet
  bajando en cada ola. No romper compatibilidad sin alias.

8.5 Perfiles predefinidos:
- `--profile production|pilot|smoke|self-programming` con los valores que
  hoy exigen exportar 15 envs a mano en cada pilotaje (esa fricción manual
  causó este incidente). Un perfil = sección del fichero canónico versionada
  en el repo. Los runbooks/scripts de pilotaje pasan a una línea:
  `orquesta-server run --profile self-programming --config ...`.

Write-set: cmd/orquesta-server, modulos/orquesta-server,
modulos/orquesta-autoprogramming, scripts, docs. Tests: los focales de 8.2 y
8.3 + suites de cmd/orquesta-server y modulos/orquesta-server.
Prioridad: 8.1+8.2 primero (cierran la clase del bug); 8.3 después; 8.4-8.5
por olas. TAREA-8 precede a reintentar el lote de 7 pilotajes de BUG-165.

### TAREA-9: limpieza de código muerto/duplicado guiada por herramientas (orden del operador 2026-07-04)

Base: docs/auditoria_codigo_muerto_duplicado_2026-07-04.md (+ listado
deadcode completo en docs/auditoria_codigo_deadcode_2026-07-04.txt).

9.1 Ola deadcode por módulo (empezar por orquesta-deploy y
   orquesta-capacity, los más señalados): para cada función candidata,
   verificar con rg que no la usan tests ni docs de contrato; borrar en
   PRs pequeños por módulo con suites verdes. Si algo debe conservarse
   (API pública intencional), anotarlo con comentario de contrato para que
   deje de contar. Meta medible: candidatas de 1188 a <300.
9.2 Decisión de módulos: orquesta-work-profiles (0 importadores) — borrar o
   justificar; revisar si deploy/capacity enteros son generación anterior
   del director y pueden retirarse a docs/historico como hizo T272.
9.3 Helpers compartidos: crear/usar orquesta-core para compact*/
   firstNonEmpty*/contains* y el símbolo repetido en 81 módulos; migrar por
   olas SIN tocar contratos hexagonales por módulo (contracts_v0/errors_v0
   se quedan donde están).
9.4 Herramienta EN Orquesta (orden explícita del operador): script
   `scripts/orquesta_auditoria_codigo.sh` que reproduce esta auditoría
   (deadcode+huérfanos+helpers copiados+ficheros grandes) volcando a
   SQLite/JSON; fase nueva del nightly que la ejecuta y RATCHET: falla si
   deadcode o copias SUBEN respecto al último verde (mismo patrón que el
   ratchet de envs). Así la limpieza no se revierte sola.
9.5 Partir los 15 ficheros >800 líneas solo cuando se toque su módulo por
   otra causa (no refactor gratuito).
Write-set: módulos afectados por ola + scripts + docs. Tests: suites de
cada módulo tocado + go build ./... por ola.

### TAREA-2 AMPLIADA (orden del operador 2026-07-04): el analizador de código es subsistema BÁSICO de Orquesta

Orden textual: "si no existe hay que crearlo, es básico para analizar código
y que el agente no pierda el contexto". Existe a medias (codebase-memory MCP
+ CodeContextQueryPortV0); hay que convertirlo en capacidad de primera clase:

A. **Índice siempre disponible**: al preparar cualquier goal con write-set de
   código, Orquesta garantiza índice fresco del repo (symbols, imports,
   callers, arquitectura por módulo). Si codebase-memory está caído o sin
   indexar, fallback determinista propio (go list + rg estructurado) — el
   agente NUNCA se queda sin analizador.
B. **Consulta desde el sandbox**: tool MCP en el toolbelt del goal (la ruta
   HTTP a localhost está vedada por el sandbox — hallazgo del experimento
   A/B). Operaciones mínimas: buscar símbolo, quién llama a X, qué expone el
   módulo Y, resumen de arquitectura de Z, y "dame SOLO los fragmentos
   relevantes para esta tarea" (presupuesto de bytes por respuesta).
C. **Contrato de uso en el prompt del goal**: el packet instruye al agente a
   consultar el analizador ANTES de leer ficheros enteros; el contexto crudo
   inicial se reduce al mínimo (esto ataca la causa de los goals de 100-200k
   tokens). Métrica: tokens_used medio por goal antes/después.
D. **Modo auditoría** (une TAREA-9.4): el mismo subsistema expone
   deadcode/huérfanos/duplicados para el nightly y su ratchet.
E. Tests: tool responde en sandbox real (smoke), fallback sin
   codebase-memory, presupuesto de bytes respetado, y goal de prueba que
   completa una tarea de código consultando el analizador con menos del 50%
   de los tokens de la línea base (165k, brazo A del experimento).
Prioridad: sube a la cabeza de la cola junto a TAREA-8 (ambas atacan las dos
causas orgánicas: config y contexto).

#### Regla de persistencia (aclaración del operador 2026-07-04, aplica a TAREA-2/9)

Un solo motor de BD por despliegue, nunca dos: si la infraestructura ya tiene
PostgreSQL, el analizador/auditoría/métricas usan ese (vía el adaptador SQL
neutral existente, placeholders dollar); en nodo local sin servidor, motor
embebido. El índice del analizador es caché DERIVADA y reconstruible desde el
repo: no es verdad operativa. La verdad operativa de Orquesta sigue siendo
los ficheros de estado/receipts. Prohibido introducir un segundo motor en un
despliegue que ya tiene uno.

### TAREA-10: retomar código construido-y-nunca-cableado (orden del operador 2026-07-04 noche)

Análisis Claude sobre docs/auditoria_codigo_deadcode_2026-07-04.txt: tres
piezas muertas NO son basura sino inversión parada. Regla general para toda
la TAREA-9: antes de borrar, clasificar "muerta-de-verdad" vs
"construida-y-nunca-cableada"; las segundas van a esta lista.

10.1 Wizard viejo de intake (modulos/orquesta-app-director-intake/wizard_*.go,
   530 líneas, inalcanzable): SUPERSEDIDO por el wizard nuevo de orquesta-web
   en riqueza (sin recomendaciones, sin huecos cruzados, campos lineales).
   PERO tiene dos piezas maduras que el nuevo debe absorber si no las tiene
   ya: (a) aplicación de respuestas con rutas punteadas al draft
   (applyAppDirectorIntakeWizardAnswersV0, p.ej. "datos.necesidad_funcional");
   (b) el CIERRE REAL: validación de envelope + occurred_at +
   orquestafactory.SolicitarNuevaAppV0(draft, now) que produce el spec final
   con evidencia. Verificar que el flujo LaunchReady del wizard nuevo termina
   igual de sólido; absorber lo que falte con sus tests; después BORRAR el
   módulo viejo de wizard entero (no dejar dos wizards otra vez).

   Estado Codex 2026-07-04 noche: el wizard nuevo ya cubre rutas punteadas
   mas ricas que el viejo y queda probado el cierre "wizard listo -> factory
   real" por puerto con TestWizardNuevoListoCierraFactoryRealPorPuertoV0.
   Mantener frontera hexagonal: no llamar SolicitarNuevaAppV0 directo desde
   orquesta-web. No borrar todavia modulos/orquesta-app-director-intake ni sus
   wizard_*.go sin deprecacion/refactor, porque el modulo tiene consumidores
   historicos; retirar solo con evidencia propia.

10.2 Política autónoma de dirección (modulos/orquesta-orchestration-core/
   autonomous_director_policy.go + autonomous_quality_policy.go, ~31
   funciones): heurística completa que decide TeamSize, MaxParallelAgents,
   MaxBursts y recomendaciones de calidad a partir de DirectorRunStatsV0,
   implementando el puerto AutonomousDirectorPolicyPortV0. Nadie la invoca.
   RETOMAR: es exactamente el cerebro que falta para (a) TAREA-3 enrutado
   por coste y (b) dimensionar paralelismo por evidencia en vez de constantes.
   Cablearla al director goal-first actual detrás de su puerto, con test que
   demuestre una decisión distinta ante stats distintos. Si algún criterio
   está obsoleto respecto al director actual, adaptarlo, no borrarlo.

   Estado Codex 2026-07-04 noche: cableada al goal-first de Nueva App por el
   puerto `AutonomousDirectorPolicyPortV0`. `StartAppDirectorPortsV0` recibe la
   politica, `BuildStackV0` la transporta desde `ConfigV0` y
   `cmd/orquesta-server` inyecta `HeuristicAutonomousDirectorPolicyV0`. La
   decision ajusta `GoalWorkSpecV0.Budget.MaxSubgoals`, anade contexto/evidencia
   `autonomous_director_policy:v0` y conserva recomendaciones de calidad como
   criterios compactos. Tests nuevos:
   `TestStartAppDirectorV0GoalFirstAplicaPoliticaAutonomaV0`,
   `TestStartAppDirectorGoalSpecWithAutonomousPolicyV0StatsDistintosDecisionDistinta`
   y `TestBuildDirectorPortsV0CableaPoliticaAutonomaV0`. Residual de TAREA-3:
   `task_cost_class` ya se deriva en `orquesta-runtime-codex-goal`; falta
   seleccion real backend/effort barato para doc vs code, sin meter proveedor
   en el nucleo.

10.3 Cableado del broker de contexto (cmd/orquesta-server/
   code_context_broker_v0.go: codeContextBrokerFromEnvV0 y
   codeContextBrokerWiringFromEnvV0): el wiring COMPLETO del analizador ya
   existe (proveedor rg fallback, proveedor codebase-MCP, cache en memoria,
   leases, state dir desde project config) y nunca se llama en el arranque.
   La TAREA-2 AMPLIADA debe partir de aquí: conectar este constructor al
   stack del servidor y exponerlo como tool del goal, NO reescribirlo.

   Estado Codex 2026-07-04 noche: el HEAD ya llama a
   codeContextBrokerWiringFromEnvV0 desde el stack y queda probado por
   TestBuildStackFromEnvV0CableaBrokerContextoEnMCPHTTPYGoalV0: stack desde
   env, binding MCP, HTTP /api/v0/codebase/query y precarga goal-first
   code_context_prepared. No reabrir salvo regresion. Residual separado:
   proyeccion MCP local en CODEX_HOME aislado del agente si se exige invocacion
   sin HTTP, siempre usando el broker central.

Prioridad: 10.3 dentro de TAREA-2 (ya en cabeza); 10.2 junto a TAREA-3;
10.1 al cerrar el wizard. Cada retoma con test focal y nota en bitácora.
