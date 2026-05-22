# Smoke autoprogramming supervisado sin Codex real

Objetivo: validar el camino opt-in minimo para autoprogramacion desatendida sin
ejecutar Codex real ni tocar OPES:

```text
request valida
  -> BuildAutoprogrammingProgrammableWorkV0
  -> ruta REST de validacion si existe
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
- Compila y arranca `cmd/orquesta-server` con `state_dir`, `runtime_dir`,
  `project_dir`, `CODEX_HOME` y comando Codex fake temporales.
- Llama a `POST /api/v0/autoprogramming/validate-request` si la ruta esta
  expuesta y exige `accepted=true`.
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
- `codex_real_executed=false`;
- `opes_touched=false`.

Los mensajes `skip:` son aceptables cuando una ruta o prueba focal todavia no
existe en la composicion actual. No deben ocultar fallos de contrato puro,
payload invalido, servidor que no arranca o tests focales existentes que fallen.

## Alcance

Este smoke prepara evidencia operacional no invasiva para autoprogramacion
desatendida, pero no cierra el smoke largo con agentes Codex reales. Tampoco
prueba recursion completa, replan generico de todos los blockers ni calidad
semantica de una entrega real.
