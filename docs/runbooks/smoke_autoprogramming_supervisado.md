# Smoke autoprogramming supervisado sin Codex real

Objetivo: validar el camino opt-in minimo para autoprogramacion desatendida sin
ejecutar Codex real ni tocar OPES:

```text
request valida
  -> BuildAutoprogrammingProgrammableWorkV0
  -> PrepareAutoprogrammingRunV0 / CodexStackAutoprogrammingExecutorV0 con stack fake
  -> ruta REST de validacion si existe
  -> POST /api/v0/autoprogramming/prepare-run
  -> POST /api/v0/runs/supervise con run_ref explicito y Codex fake
  -> POST /api/v0/autoprogramming/prepare-run con el mismo payload para replay
  -> external-work/run + supervisor si la composicion lo expone
  -> pruebas focales de stack fake y cierre offline segun disponibilidad
```

Este smoke no modifica el repo. Crea harnesses, servidor, state, runtime y HOME
Codex falsos bajo `/tmp` o bajo `ORQUESTA_SMOKE_ROOT`.

## Comando

Desde la raiz del repo:

```bash
ORQUESTA_AUTOPROGRAMMING_SUPERVISED_SMOKE_CONFIRM=1 \
  ./scripts/smoke_autoprogramming_supervised.sh
```

Variables utiles:

```bash
ORQUESTA_KEEP_SMOKE_DIR=1
ORQUESTA_SMOKE_ROOT=/tmp/orquesta-autoprogramming-supervised
ORQUESTA_SMOKE_REQUEST_TIMEOUT_SECONDS=15
SMOKE_ID=manual-001
```

## Que comprueba

- `go test -count=1 ./modulos/orquesta-autoprogramming`.
- Un programa temporal fuera del repo importa `orquesta-autoprogramming` y
  ejecuta `BuildAutoprogrammingProgrammableWorkV0` sobre una request valida,
  verificando que produce grupos, perfiles y `WorkflowTaskV0`.
- Si los tests existen, ejecuta la entrada real no-HTTP del stack:
  `PrepareAutoprogrammingRunV0` y `CodexStackAutoprogrammingExecutorV0`. Esa
  cobertura valida que la request se transforma en run continuable,
  `WorkflowTaskV0` persistidas, `WaitAgentRefs` y `ContinueAppDirectorRequestV0`
  usando runtime fake.
- Compila y arranca `cmd/orquesta-server` con `state_dir`, `runtime_dir`,
  `project_dir`, `CODEX_HOME` y comando Codex fake temporales.
- Llama a `POST /api/v0/autoprogramming/validate-request` si la ruta esta
  expuesta y exige `accepted=true`.
- Llama a `POST /api/v0/autoprogramming/prepare-run`, exige `run_ref`,
  `wait_agent_refs` y `continue`, y despues llama a
  `POST /api/v0/runs/supervise` con ese `run_ref` y limites bajos usando
  comando Codex fake.
- Vuelve a llamar a `POST /api/v0/autoprogramming/prepare-run` con el mismo
  payload despues de `/api/v0/runs/supervise`; exige respuesta aceptada,
  `run_ref` estable, `wait_agent_refs` no vacio y `continue`, para cubrir
  idempotencia cuando el patch del stack este disponible.
- Intenta `POST /api/v0/external-work/run` con un trabajo
  `autoprogramming_programmable_work`; si falta la ruta o el executor, lo marca
  como `skip` y no falla.
- Si `external-work/run` devuelve `run_ref`, llama a
  `POST /api/v0/runs/supervise` con limites bajos; si la ruta no esta
  disponible, lo marca como `skip`.
- Ejecuta tests focales existentes cuando estan presentes:
  `external-work/run` con stack fake, supervisor con stack fake,
  `OperationalClosureSourceV0` y cierre offline de `app-director-service`.

## Criterio de exito

La salida debe incluir:

- `programmable_work_ok`;
- `validate_request_ok=true` si la ruta REST esta disponible;
- `prepare_run_ok=true`;
- `prepare_run_supervisor_estado=ok`;
- `prepare_run_replay_ok=true`;
- `codex_real_executed=false`;
- `opes_touched=false`.

Los mensajes `skip:` son aceptables cuando una ruta o prueba focal todavia no
existe en la composicion actual. No deben ocultar fallos de contrato puro,
payload invalido, servidor que no arranca o tests focales existentes que fallen.

## Entrada real actual

La entrada real esta disponible por composicion Go y por ruta publica opt-in del
stack Codex:

- `modulos/orquesta-app-codex-stack/PrepareAutoprogrammingRunV0`;
- `modulos/orquesta-app-codex-stack/CodexStackAutoprogrammingExecutorV0`.
- `POST /api/v0/autoprogramming/prepare-run`, expuesta por
  `orquesta.autoprogramming.prepare_run.v0` y cableada en
  `orquesta-app-codex-stack`.

El smoke la cubre con tests focales y por servidor temporal. La supervision
posterior no es global: usa siempre el `run_ref` devuelto por `prepare-run`.
Tras supervisar, repite `prepare-run` con el mismo payload y exige que el replay
idempotente conserve el `run_ref` y siga devolviendo refs causales.
`/api/v0/external-work/run` queda como fallback opcional de compatibilidad para
otros trabajos externos; no es el launcher de autoprogramacion.

## Alcance

Este smoke prepara evidencia operacional no invasiva para autoprogramacion
desatendida, pero no cierra el smoke largo con agentes Codex reales. Tampoco
prueba recursion completa, replan generico de todos los blockers ni calidad
semantica de una entrega real.
