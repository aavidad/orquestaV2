# Smoke BUG-088 goal-first alto consumo

Fecha: 2026-07-03.

Objetivo: validar en Codex real `app_server_tmux` que un goal-first con uso
alto no queda consumiendo sin salida gobernada: o produce un segundo artefacto
recuperable tras checkpoint, o queda preparado para replan, y el shutdown no
deja `codex app-server` residual.

## Comando

Preflight sin ejecutar Codex:

```bash
ORQUESTA_GOAL_FIRST_SMOKE_PREFLIGHT_ONLY=1 \
ORQUESTA_CODEX_COMMAND="$(command -v codex)" \
./scripts/smoke_goal_first_checkpoint_only_high_consumption_real.sh
```

Smoke real opt-in:

```bash
ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_REAL_CONFIRM=1 \
ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_CODEX_EXECUTION_CONFIRMED=1 \
./scripts/smoke_goal_first_checkpoint_only_high_consumption_real.sh
```

El wrapper activa `ORQUESTA_GOAL_FIRST_SMOKE_HIGH_CONSUMPTION_MODE=1`, backend
`app_server_tmux`, observador residente, umbral bajo de uso alto para smoke y
cleanup gobernado. OPES debe estar vacio.

## Evidencia 2026-07-03

Smoke cerrado:

- `smoke_goal_first_high_consumption_real=ok`
- `bug088_path=second_artifact_or_partial_artifacts`
- `recommended_action=review_partial_artifacts`
- `app_server_tmux_processes_alive=0`
- `smoke_root=/tmp/orquesta-goal-first-app-server.lwNV5d`

Refs principales:

- `run_ref=run-spec-smoke-goal-first-bug088-req-smoke-goal-first-bug088-0044c2891a22e01c8c9be575c7b7181b`
- `external_goal_ref=019f29ef-0413-7403-b7d7-a6278fb15c9d`

Evidencia durable del temporal:

- `observe_response.json` publico `codex_app_server_goal_status_active_high_token_usage`, `tokens_used=13512`, `evidence-ref-goal-observer-no-checkpoint-high-consumption`, `evidence-ref-goal-observer-high-consumption-stop-requested` y `evidence-ref-goal-cooperative-stop-requested-run-control`.
- El write-set contiene `generated-apps/checkpoint_started_bug088.txt` y `generated-apps/bug088_second_artifact.txt`.
- La comprobacion final no encontro procesos `orquesta-server run`, `codex app-server --listen` ni `codebase-memory-mcp`.

## Residuales cerrados despues del smoke

Durante los intentos previos quedaron documentados dos residuales. El cierre
posterior de la misma noche los cubre por codigo y pruebas focales:

- `BUG-ORQ-20260703-161`: `observe_goal` puede tardar en proyectar como
  `artifact_refs` los ficheros directos del smoke aunque existan en el write-set.
  Cierre: `checkpoint_started*.txt` se reconoce como checkpoint,
  `*_artifact.txt` como artefacto materializado y
  `autoprogramming observe` reutiliza el enriquecimiento de refs materializadas.
- `BUG-ORQ-20260703-162`: en una rama de fallo, `/server/shutdown` devolvio
  `shutdown_ready=true` sin `exit_pending/pid`; el cleanup no dejo procesos
  vivos. Cierre: el runtime no hereda snapshots activos stale cuando el `ready`
  trae evidencia de cleanup de backend, por lo que vuelve a publicar
  `exit_pending/pid`.
