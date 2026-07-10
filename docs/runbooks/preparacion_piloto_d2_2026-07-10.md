# Preparacion del piloto D2 - listo para lanzar (2026-07-10)

Autor: Claude Fable (revisor/director). Este runbook deja el paso D2 de
`docs/guia_continuacion_agentes_2026-07-10.md` preparado al detalle: solo
falta el CONSENTIMIENTO EXPLICITO del operador para ejecutarlo (consume
cuota Codex y lanza ejecucion real). Base verificada: receta completa de
`docs/bitacora_correccion_pericial_2026-07-03.md` (pilotaje T265 verde).

## Que se lanza

Una tarea del backlog `docs/backlog_piloto_autonomia_2026-07-10.md`
(empezar por T9101), de una en una, con `MAX_REQUESTS=1` y revisor externo
(Claude) validando el cierre reejecutando los tests declarados. 208H esta
cerrado localmente, pero NO se acepta ningun cierre sin reejecutar sus tests
de atestacion independiente.

## Pasos exactos

1. Worktree aislado (NUNCA el arbol principal compartido):
   `git worktree add /home/alberto/Trabajo/orquesta-piloto-d2 -b pericial/pilot-d2-t9101`
2. Backlog minimo dentro del worktree: sustituir
   `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md` por un doc con
   SOLO la seccion `## T9101 revisar-evidencia-208h` copiada del backlog
   C2. OJO parser: etiquetas exactas `Objetivo:` (una linea), `Estado:
   pendiente.`, `Alcance:` (lista = write-set, `docs` primero), `Criterios:`
   (lista), `Tests:` (lista). El parser IGNORA "Write-set previsto:".
3. Compilar server y exportar el entorno de la receta (aislado):
   - `ORQUESTA_SERVER_ADDR=127.0.0.1:0`, `ORQUESTA_SERVER_STATE_DIR=<aislado>`
   - `ORQUESTA_SERVER_SELF_PROGRAMMING_ONLY=true`,
     `ORQUESTA_SERVER_SELF_PROGRAMMING_ROOT=<raiz aislada>`
   - `ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_ENABLED=false`,
     `ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED=false`
   - `ORQUESTA_CODEX_PROJECT_WORKDIR=<worktree>` y
     `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_PROJECT_WORKDIR=<worktree>`
   - `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_DISABLED=false`,
     `..._AFTER_SECONDS=5`, `..._MAX_REQUESTS=1`, `..._TARGET_QUEUE=1`
   - `ORQUESTA_SERVER_MAX_RUNS_PER_TICK=1`,
     `ORQUESTA_SERVER_MAX_EXECUTIONS_PER_TICK=1`
   - `ORQUESTA_CODEX_RUNTIME_WORKDIR=<aislado>`,
     `ORQUESTA_CODEX_COMMAND=$(command -v codex)`,
     `ORQUESTA_CODEX_CODE_HOME=$HOME/.codex`,
     `ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux`,
     `ORQUESTA_CODEX_GOAL_TIMEOUT_MS=900000`,
     `ORQUESTA_CODEX_APPROVAL_POLICY=never`,
     `ORQUESTA_CODEX_SANDBOX=workspace-write`, `ORQUESTA_OPES_BASE_URL=""`
   - lanzar `orquesta-server run` en background.
4. Addr real en `<state>/orquesta_server_state_v0.json` campo `addr`;
   readiness `GET /api/v0/server/readiness`.
5. Observacion: SOLO POST con body JSON:
   `POST /api/v0/autoprogramming/status` body `{}` y
   `POST /api/v0/autoprogramming/goal/observe` body `{"run_ref":"..."}`.
   Con la etapa A integrada, ambos publican `causal_verdict`/
   `causal_reason_code`: no aceptar `running` sin `running_confirmed`.
6. Cierre: el revisor reejecuta los `Tests:` declarados de la tarea (real,
   `ok` visible) antes de aceptar; diff del worktree revisado; integrar a
   la rama de trabajo SOLO tras revision.
7. Shutdown SIEMPRE: `POST /api/v0/server/shutdown` body
   `{"forced":true,"reason":"piloto d2 fin","idempotency_key":"idem-piloto-d2-<fecha>"}`
   y verificar que el PID sale y no quedan panes `orquesta-goal-*`.

## Trampas conocidas aplicables

- Los goals escriben el resultado durable bajo el PRIMER scope directorio
  del write-set: `docs` va primero en `Alcance:`.
- No versionar `checkpoint_started_*` ni `orquesta_goal_result_*` que
  genere el piloto (ver `docs/clasificacion_retencion_s13_2026-07-10.md`).
- tmux 3.6: no simplificar selectores (`=sesion:` con `:`).
- El bloqueador del ratchet de envs (536>511) afecta a los LOTES de tests,
  no al arranque del server; el piloto puede correr antes de esa
  consolidacion, pero sus `Tests:` declarados deben ser focales.

## Criterio de exito del piloto

Goal `complete` + closure `accepted` validado por revisor con tests
reejecutados + shutdown limpio sin procesos residuales. Cualquier
`blocked`/residuo se documenta como incidencia y NO se relanza en bucle.
