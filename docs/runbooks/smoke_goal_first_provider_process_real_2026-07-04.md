# Smoke goal-first con proveedores process reales

Fecha: 2026-07-04.

Estado: Claude real validado; Gemini real bloqueado por tier externo del CLI
actual.

## Alcance

Este smoke valida los backends goal-first opt-in de proceso:

- `ORQUESTA_CODEX_GOAL_BACKEND=claude_process`
- `ORQUESTA_CODEX_GOAL_BACKEND=gemini_process`

No toca OPES productivo ni arranca servidor residente. Ejecuta el backend de
runtime contra un proyecto temporal aislado y exige un
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

Smoke opt-in Gemini, cuando exista tier/credencial valido:

```bash
SMOKE_GEMINI_GOAL_PROCESS_REAL=1 \
go test -count=1 ./modulos/orquesta-runtime-gemini \
  -run TestGeminiGoalProcessBackendV0RealOptInEscribeResultadoDurableV0 -v
```

## Pendiente

- Repetir el smoke Gemini cuando el proveedor deje de devolver
  `IneligibleTierError`.
- Ejecutar una prueba real amplia a traves de `cmd/orquesta-server` y
  run-control, no solo runtime directo.
