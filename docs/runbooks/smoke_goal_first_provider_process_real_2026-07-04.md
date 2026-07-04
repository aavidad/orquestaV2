# Smoke goal-first con proveedores process reales

Fecha: 2026-07-04.

Estado: Claude real validado en runtime directo y por `cmd/orquesta-server`;
Gemini real bloqueado por tier externo del CLI actual.

## Alcance

Este smoke valida los backends goal-first opt-in de proceso:

- `ORQUESTA_CODEX_GOAL_BACKEND=claude_process`
- `ORQUESTA_CODEX_GOAL_BACKEND=gemini_process`

No toca OPES productivo. Ejecuta el backend de runtime contra un proyecto
temporal aislado y, en el smoke de servidor, arranca un `orquesta-server`
temporal con puerto local efimero. Ambos caminos exigen un
`orquesta_goal_result_v0.json` durable dentro del write-set.

## Preflight ejecutado

Versiones locales:

```bash
claude --version
gemini --version
```

Resultado observado:

- Claude Code `2.1.201`.
- Gemini CLI `0.45.1`.

Autenticacion minima Claude:

```bash
printf 'Responde solo: OK\n' |
  timeout 120 claude -p --model sonnet \
    --permission-mode bypassPermissions \
    --output-format text \
    --max-budget-usd 0.20
```

Resultado: `OK`.

Preflight Gemini:

```bash
printf 'Responde solo: OK\n' |
  timeout 120 gemini --prompt '' \
    --approval-mode auto_edit \
    --output-format text \
    --skip-trust
```

Resultado: bloqueado por `IneligibleTierError` /
`UNSUPPORTED_CLIENT`. Esto coincide con `BUG-ORQ-20260703-140`: el wrapper
Gemini ya materializa `orquesta_provider_diagnostic_v0.json` con
`provider_auth_or_tier_blocked`; no se abre bug nuevo de Orquesta.

## Smoke Claude real

Comando reproducible:

```bash
SMOKE_CLAUDE_GOAL_PROCESS_REAL=1 \
SMOKE_CLAUDE_MAX_BUDGET_USD=0.50 \
SMOKE_CLAUDE_KEEP_DIR=1 \
go test -count=1 ./modulos/orquesta-runtime-claude \
  -run TestClaudeGoalProcessBackendV0RealOptInEscribeResultadoDurableV0 -v
```

Resultado: `PASS`.

Evidencia retenida de la ejecucion validada:

- `/tmp/orquesta-claude-goal-real-smoke-3215275659/project/docs/provider_goal_smoke.txt`
- `/tmp/orquesta-claude-goal-real-smoke-3215275659/project/docs/orquesta_goal_result_v0.json`
- `/tmp/orquesta-claude-goal-real-smoke-3215275659/runtime/claude_goal_process_state_goal-ref-claude-process-real-smoke-001.json`
- `/tmp/orquesta-claude-goal-real-smoke-3215275659/runtime/claude_goal_prompt_goal-ref-claude-process-real-smoke-001.txt`
- `/tmp/orquesta-claude-goal-real-smoke-3215275659/runtime/claude_goal_wrapper_goal-ref-claude-process-real-smoke-001.sh`

Contenido clave validado:

- `provider_goal_smoke.txt`: `claude real smoke ok`
- `orquesta_goal_result_v0.json`: `status=complete`
- `artifact_refs`: 2 refs no vacias
- `artifact_paths`: `docs/provider_goal_smoke.txt` y
  `docs/orquesta_goal_result_v0.json`
- `materialized_artifacts`: 2 artefactos con `artifact_ref`, `path`,
  `status=valid` y `evidence_refs`
- `required_test_results`: `passed`

## Smoke Claude real por servidor

Comando reproducible validado:

```bash
SMOKE_CLAUDE_GOAL_PROCESS_SERVER_REAL=1 \
SMOKE_CLAUDE_GOAL_PROCESS_SERVER_SKIP_PREFLIGHT=1 \
SMOKE_CLAUDE_GOAL_PROCESS_SERVER_KEEP_DIR=1 \
SMOKE_CLAUDE_GOAL_PROCESS_SERVER_MAX_BUDGET_USD=0.80 \
./scripts/smoke_goal_first_claude_process_server_real.sh
```

Resultado: `smoke_goal_first_claude_process_server_real=ok`.

Evidencia retenida de la ejecucion aceptada:

- `smoke_root=/tmp/orquesta-claude-process-server.x5N8pq`
- `run_ref=run-spec-smoke-claude-process-server-req-smoke-claude-process-server-389d151256b512f15e130b3042aa99fe`
- `goal_ref=goal-ref-app-director-run-spec-smoke-claude-process-server-req-smoke-claude-process-server-389d151256b512f15e130b3042aa99fe`
- `external_goal_ref=claude-goal-c51e3c2c56c62b53`
- `result_file=/tmp/orquesta-claude-process-server.x5N8pq/project/generated-apps/smoke-claude-process-server/orquesta_goal_result_v0.json`

Salida terminal observada en `poll=31`:

- `director_execution_mode=goal_first`
- `goal_status=complete`
- `run_status=cerrada`
- `closure_status=accepted`
- `closure_accepted=true`
- `recommended_action=no_action_closed`

Contenido clave validado por `observe_response.json`:

- `artifact_refs`: 9 refs reconciliadas por Orquesta.
- `evidence_refs`: 16 refs reconciliadas por Orquesta.

Contenido clave validado por `orquesta_goal_result_v0.json`:

- `status=complete`
- `artifact_refs`: 3 refs requeridas (`source`, `handoff`,
  `technical-stack`).
- `artifact_paths`: 13 paths.
- `materialized_artifacts`: 3 artefactos.
- `required_test_results`: 1 test requerido `passed`.

Incidencias detectadas y cerradas durante el smoke de servidor:

- Primer intento: `tipo_app=web_app` no era aceptado por el contrato HTTP de
  Nueva App; el smoke usa `tipo_app=mixed`.
- Segundo intento: Claude devolvio `evidence_refs` como objetos `{ref,
  description}` y el observe quedo `goal_status=invalid`; los backends
  Claude/Gemini normalizan ahora solo esa forma recuperable y el prompt exige
  arrays de strings.
- Un intento sin `--safe-mode` heredo configuracion local de Claude y lanzo
  `codebase-memory-mcp`; el smoke servidor usa por defecto
  `SMOKE_CLAUDE_GOAL_PROCESS_SERVER_SAFE_MODE=1` y el wrapper retenido muestra
  `claude -p ... --safe-mode`. La comprobacion posterior no encontro
  `orquesta-server run`, `claude_goal_wrapper`, `claude -p`,
  `codebase-memory-mcp`, `codex app-server` ni `orquesta-goal-*` vivos.

## Smoke Claude runs/control real

Comando reproducible validado:

```bash
SMOKE_CLAUDE_GOAL_PROCESS_SERVER_REAL=1 \
SMOKE_CLAUDE_GOAL_PROCESS_SERVER_CONTROL_MODE=forced_stop \
SMOKE_CLAUDE_GOAL_PROCESS_SERVER_SKIP_PREFLIGHT=1 \
SMOKE_CLAUDE_GOAL_PROCESS_SERVER_KEEP_DIR=1 \
SMOKE_CLAUDE_GOAL_PROCESS_SERVER_MAX_BUDGET_USD=0.80 \
./scripts/smoke_goal_first_claude_process_server_real.sh
```

Resultado: `smoke_goal_first_claude_process_forced_stop_server_real=ok`.

Evidencia retenida:

- `smoke_root=/tmp/orquesta-claude-process-server.GnFFpX`
- `run_ref=run-spec-smoke-claude-process-server-req-smoke-claude-process-server-dff02568d5f2f00ac86d8f91237536c7`
- `goal_ref=goal-ref-app-director-run-spec-smoke-claude-process-server-req-smoke-claude-process-server-dff02568d5f2f00ac86d8f91237536c7`
- `external_goal_ref=claude-goal-06148fd846fd9f64`
- `claude_process_manifest_pid=4047077`

Salida clave de `/api/v0/runs/control`:

- `HTTP 200`
- `estado=ok`
- `status=stopped`
- `final_status=stopped`
- `goal_status_before=running`
- `goal_status_after=blocked`
- `goal_control_signal_confirmed=true`
- evidencia `evidence-ref-claude-goal-process-stop-completed`
- evidencia `evidence-ref-run-control-goal-forced-stop-terminal`
- evidencia `evidence-ref-run-control-terminal-after-goal-forced-stop`

Observe posterior:

- `goal_status=blocked`
- `run_status=bloqueada`
- `closure_status=blocked`
- `recommended_action=replan`
- `evidence_refs`: 16

Shutdown/limpieza posterior:

- `state/orquesta_server_state_v0.json`: `status=stopped`,
  `shutdown_status=stopped`, `shutdown_ready=true`.
- Comprobacion de procesos: sin `orquesta-server run`, sin
  `claude_goal_wrapper`, sin `claude -p`, sin `codebase-memory-mcp`, sin
  `codex app-server` y sin `orquesta-goal-*`.
- El wrapper retenido conserva `claude -p ... --safe-mode`.

## Cobertura permanente

Tests por defecto, sin proveedor real:

```bash
go test -count=1 ./modulos/orquesta-runtime-claude ./modulos/orquesta-runtime-gemini
```

Smoke opt-in Claude:

```bash
SMOKE_CLAUDE_GOAL_PROCESS_REAL=1 \
SMOKE_CLAUDE_MAX_BUDGET_USD=0.50 \
go test -count=1 ./modulos/orquesta-runtime-claude \
  -run TestClaudeGoalProcessBackendV0RealOptInEscribeResultadoDurableV0 -v
```

Smoke opt-in Claude por `cmd/orquesta-server`:

```bash
SMOKE_CLAUDE_GOAL_PROCESS_SERVER_REAL=1 \
./scripts/smoke_goal_first_claude_process_server_real.sh
```

Smoke opt-in Claude por `cmd/orquesta-server` con forced stop:

```bash
SMOKE_CLAUDE_GOAL_PROCESS_SERVER_REAL=1 \
SMOKE_CLAUDE_GOAL_PROCESS_SERVER_CONTROL_MODE=forced_stop \
./scripts/smoke_goal_first_claude_process_server_real.sh
```

Smoke opt-in Gemini, cuando exista tier/credencial valido:

```bash
SMOKE_GEMINI_GOAL_PROCESS_REAL=1 \
go test -count=1 ./modulos/orquesta-runtime-gemini \
  -run TestGeminiGoalProcessBackendV0RealOptInEscribeResultadoDurableV0 -v
```

## Pendiente

- Repetir el smoke Gemini cuando el proveedor deje de devolver
  `IneligibleTierError`.
- Repetir forced stop/cancel equivalente contra Gemini cuando exista
  tier/credencial valido.
- Mantener un smoke largo de `status/observe` lento si reaparece el bloqueo de
  observabilidad global de `BUG-ORQ-20260704-165`; Claude ya cubre
  launch/observe/closure accepted y forced stop por HTTP.
