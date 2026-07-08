# Runbook: golden tasks evals

Este runbook opera el banco inicial de MEJ-TASK-205. La cadencia recomendada es
semanal y tambien antes de cambiar prompts, modelos, runtime de Goal o reglas de
autoprogramacion.

## Artefactos

- Manifest: `docs/evals/orquesta_golden_tasks_v0.json`.
- Harness: `scripts/orquesta_golden_evals.sh`.
- Resultados datados: `docs/evals/results/orquesta_golden_eval_*.json`.

El manifest declara cinco clases distintas: nueva app de smoke, script acotado,
docs/scanner de backlog, fix con test congelado y exploracion de mapa publico.
Cada tarea tiene `write_set`, `expected_files`, `required_tests` y
`verifier_refs` evaluables por script. No hay juicio manual en la puntuacion.

## Preflight

Ejecuta siempre primero:

```sh
bash -n scripts/orquesta_golden_evals.sh
scripts/orquesta_golden_evals.sh --self-test
```

El self-test crea fixtures sinteticos en un temporal, evalua las cinco tareas y
escribe un JSON datado comparable bajo `docs/evals/results/` salvo que se pase
`--output`.

## Ejecucion opt-in aislada

No ejecutes el banco contra una instancia compartida. Usa una instancia o
worktree aislado y confirma explicitamente:

```sh
ORQUESTA_GOLDEN_EVALS_CONFIRM=isolated \
scripts/orquesta_golden_evals.sh --run --prepare-only \
  --results-dir /tmp/orquesta-golden-run
```

Ese modo solo materializa packets `orquesta_golden_task_request.v0` por tarea.
Para lanzar de verdad, proporciona un launcher opt-in que lea:

- `ORQUESTA_GOLDEN_TASK_ID`
- `ORQUESTA_GOLDEN_TASK_REQUEST`
- `ORQUESTA_GOLDEN_TASK_RESULT_DIR`

Ejemplo de forma:

```sh
ORQUESTA_GOLDEN_EVALS_CONFIRM=isolated \
scripts/orquesta_golden_evals.sh --run --parallel \
  --results-dir /tmp/orquesta-golden-run \
  --launcher-command './scripts/mi_launcher_aislado.sh' \
  --output docs/evals/results/orquesta_golden_eval_manual.json
```

El launcher debe escribir `result.json` en cada `ORQUESTA_GOLDEN_TASK_RESULT_DIR`
con `task_id`, `status`, `touched_files`, `tests`, `artifact_paths` y
`evidence`. El evaluador puntua tests pasados, ficheros esperados y ausencia de
escrituras fuera de `write_set`.

Desde 2026-07-08 el launcher puede anadir `metrics` para comparaciones A/B:

```json
{
  "metrics": {
    "input_tokens": 0,
    "output_tokens": 0,
    "reasoning_tokens": 0,
    "cached_input_tokens": 0,
    "total_tokens": 0,
    "tool_calls": 0,
    "elapsed_ms": 0,
    "files_touched": 0,
    "new_files_count": 0,
    "lines_added": 0,
    "lines_deleted": 0,
    "helpers_added": 0,
    "abstractions_added": 0,
    "rework_count": 0,
    "scope_expansion_reason": ""
  }
}
```

Tambien se acepta `metrics.tokens.{input,output,reasoning,cached_input,total}`.
El reporte agregado escribe `summary.metrics` y conserva `tasks[].metrics`.
Si el launcher no informa metricas, el evaluador conserva compatibilidad y solo
deriva `files_touched` desde `touched_files`.

Tambien existe un wrapper opt-in para no duplicar medicion comun en cada
launcher real:

```sh
ORQUESTA_GOLDEN_EVALS_CONFIRM=isolated \
ORQUESTA_GOLDEN_METRICS_INNER_LAUNCHER='./scripts/mi_launcher_aislado.sh' \
scripts/orquesta_golden_evals.sh --run --parallel \
  --results-dir /tmp/orquesta-golden-run \
  --launcher-command './scripts/orquesta_golden_metrics_launcher.sh' \
  --output docs/evals/results/orquesta_golden_eval_manual.json
```

El wrapper conserva el `result.json` del launcher interno y anade metricas
deterministas: `elapsed_ms`, exit code, ficheros tocados declarados, defaults de
diff y, si `ORQUESTA_GOLDEN_TASK_WORKTREE` apunta a un repo Git, lineas y
ficheros tocados por `git diff/status`. Los tokens reales siguen dependiendo
del launcher/proveedor: si el launcher interno escribe `metrics.tokens` o los
campos `*_tokens`, el wrapper los normaliza sin inventarlos.

## Evaluar Reglas De Programacion

Antes de declarar mejor una regla de agente que pretende ahorrar tokens o
reducir sobreingenieria, ejecuta comparacion A/B:

1. Baseline: mismas tareas doradas sin la skill nueva.
2. Variante: mismas tareas doradas con la `skill_ref` nueva inyectada por el
   launcher o por la composicion aislada.
3. Compara al menos: score del evaluador, tests pasados, ficheros tocados,
   ficheros nuevos, lineas anadidas/eliminadas, rework y coste/tiempo si el
   proveedor lo reporta.

Para `orquesta-programacion-minima`, la variante debe inyectar
`skill-ref-orquesta-programacion-minima-v0`. Si el score baja, aumentan los
fallos o el ahorro de diff se consigue rompiendo tests/contratos, la regla se
mantiene opt-in y se documenta como no apta para default amplio.

## Guardas OPES

El harness no llama a OPES. Si detecta variables de entorno OPES productivas,
aborta salvo override explicito de laboratorio. Nunca uses ese override contra
produccion.

## Criterios APG cubiertos

La tarea `golden-script-t265-idle-self-improvement-v0` exige evidencia de:

- default de `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS` a 60 segundos;
- valor `0` desactivando solo el reloj idle;
- preparacion por idle o capacidad libre bajo
  `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_TARGET_QUEUE`;
- causa general corregida usando evidencia del fallo.

La tarea `golden-docs-t280-backlog-scanner-v0` exige que el planner salte tareas
visibles, filtre secciones narrativas, cree `Escaneo backlog nuevos`, cite
`scan-ref-backlog-f6ae5d12b919` con lineas y hashes, conserve borrador si el
scanner cambio desde la foto y bloquee `T289` ambiguo con
`backlog_duplicate_task_id_ambiguous`.

La tarea `golden-public-map-exploration-v0` exige que la proyeccion publica
distinga `outbox_pending`, `wait_external` y `external_process_verified`.
