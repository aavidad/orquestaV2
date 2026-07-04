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
- Queda pendiente no cerrado total: revalidacion real de BUG-165 post-fix y
  backlog vivo 058, 066, 073, 075, 079, mas smoke real MEJ-104 si sigue
  aplicando.

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
