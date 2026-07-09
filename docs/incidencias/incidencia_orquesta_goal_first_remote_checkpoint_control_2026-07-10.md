# Incidencia: goal-first remoto queda vivo con checkpoint y control no propagado

Fecha: 2026-07-10  
ID: BUG-ORQ-20260710-208  
Estado: abierto  
Area: Orquesta remoto, goal-first, app-server Codex, control de runs

## Resumen

Durante el lanzamiento remoto de un goal de cierre 100% por
`POST /api/v0/autoprogramming/prepare-run`, Orquesta acepto cuatro goals
goal-first en paralelo, pero la observacion publica quedo inconsistente:

- `observe` devolvia `goal_status=running`.
- Varios goals solo habian materializado `checkpoint_started`.
- Dos goals escribieron `orquesta_goal_result` con
  `status=blocked` y `reason_code=checkpoint_started`.
- `POST /api/v0/runs/control` para parar el goal de Telegram/nightly devolvio
  `control_not_propagated_to_goal_backend`; incluso con `forced=true`, el
  backend siguio `running`.

Esto no se debe tratar como un fallo aislado de una tarea. Es un patron de
control/residencia: Orquesta puede presentar trabajo vivo sin ejecucion
verificable y no puede confirmar parada del backend goal-first.

## Evidencia observada

Servidor remoto:

- PID corregido tras restart gobernado: `/srv/orquesta-self/runtime/orquesta-server-claude run --config /srv/orquesta-self/secrets/orquesta.config.json`.
- Workdir efectivo tras correccion:
  `ORQUESTA_CODEX_PROJECT_WORKDIR=/srv/orquesta-self/worktrees/orquesta`.
- Antes de la correccion, el proceso vivo tenia
  `ORQUESTA_CTL_WORKDIR=/srv/orquesta-self/worktrees/orquesta-repair-checkpoint-20260709T215549Z`,
  un worktree retirado en la limpieza operativa previa.

Request lanzada:

- `/srv/orquesta-self/operator-requests/20260710/orquesta-100-continuous-goal.json`
- Respuesta:
  `/srv/orquesta-self/operator-requests/20260710/orquesta-100-continuous-goal.response.json`
- `run_ref` principal publicado por compatibilidad:
  `run-ref-orquesta-100-continuous-20260710-001-goal-01`

Goals creados:

- `run-ref-orquesta-100-continuous-20260710-001-goal-01`
  `goal-ref-task-autoprogramming-f0ad7bb152dd-g01`
- `run-ref-orquesta-100-continuous-20260710-001-goal-02`
  `goal-ref-task-autoprogramming-f0ad7bb152dd-g02`
- `run-ref-orquesta-100-continuous-20260710-001-goal-03`
  `goal-ref-task-autoprogramming-f0ad7bb152dd-g03`
- `run-ref-orquesta-100-continuous-20260710-001-goal-04`
  `goal-ref-task-autoprogramming-f0ad7bb152dd-g04`

Artefactos parciales visibles en el worktree remoto:

- `cmd/orquesta-server/docs/checkpoint_started_goal-ref-task-autoprogramming-f0ad7bb152dd-g01.txt`
- `scripts/checkpoint_started_goal-ref-task-autoprogramming-f0ad7bb152dd-g02.txt`
- `modulos/orquesta-web/docs/checkpoint_started_goal-ref-task-autoprogramming-f0ad7bb152dd-g03.txt`
- `scripts/checkpoint_started_goal-ref-task-autoprogramming-f0ad7bb152dd-g04.txt`
- `scripts/docs/orquesta_goal_result_goal-ref-task-autoprogramming-f0ad7bb152dd-g02.json`
- `scripts/docs/orquesta_goal_result_goal-ref-task-autoprogramming-f0ad7bb152dd-g04.json`

Control fallido:

```text
POST /api/v0/runs/control
action=stop
run_ref=run-ref-orquesta-100-continuous-20260710-001-goal-04
HTTP 409
code=control_not_propagated_to_goal_backend
goal_status_before=running
goal_status_after=running
recommended_action=observe_goal_backend_before_declaring_stopped
```

Reintento con `forced=true`:

```text
estado=error
status=stop_requested
final_status=stop_requested
forced=true
goal_status_before=running
goal_status_after=running
code=control_not_propagated_to_goal_backend
```

Ampliacion de observacion:

```text
POST /api/v0/autoprogramming/goal/observe
run_ref=goal-01..goal-04
HTTP 504
code=autoprogramming_observe_goal_timeout
```

En los goals 02, 03 y 04 la respuesta parcial seguia mostrando:

```text
run_status=activa
goal_status=running
closure_status=blocked
evidence-ref-codex-app-server-goal-rpc-unsupported
evidence-ref-goal-materialized-partial-artifacts-written
evidence-ref-goal-materialized-checkpoint-detected
```

Log del app-server vivo:

```text
ERROR codex_core::tools::router: apply_patch verification failed:
Failed to find expected lines in
/srv/orquesta-self/worktrees/orquesta/scripts/smoke_opes_domain_work_real.sh

ERROR codex_core::tools::router: apply_patch verification failed:
Failed to find expected lines in
/srv/orquesta-self/worktrees/orquesta/scripts/orquesta_server_deploy.sh
```

Esto refuerza la hipotesis: habia trabajo parcial real, pero el backend quedo
sin cierre util y la API de observacion/control no convirtio ese estado en
rework/blocked accionable.

## Cambios parciales en curso

Los goals remotos empezaron a modificar codigo, pero no hay cierre ni tests
todavia:

- `modulos/orquesta-app-codex-stack/goal_first_resident_rework_v0.go`
  anade razones de rework para workdir/auth/provider/quota/backend.
- `modulos/orquesta-runtime-codex-appserver/codex_goal_app_server_v0.go`
  valida que el workdir raiz exista y clasifica `workdir_unavailable`.
- `modulos/orquesta-web/nueva_app_wizard_types_v0.go`
  anade tipos iniciales para `WizardDossierV0`.

Estas modificaciones son recuperables, pero no cierran la incidencia hasta que
se integren, pasen tests y se demuestre reconciliacion real por API.

Actualizacion Codex 2026-07-10:

- Workdir inexistente: el adaptador app-server valida que el workdir raiz
  exista y no lo recrea silenciosamente; si falta, devuelve diagnostico publico
  `codex_app_server_write_set_prepare_failed` con causa
  `workdir_unavailable`.
- Rework operativo: el supervisor goal-first residente reconoce bloqueos
  recuperables de workdir/auth/provider/quota/storage/backend y prepara rework
  acotado.
- Write-sets declarados: `orquesta-autoprogramming` secuencia tareas que
  declaran write-sets solapados, en vez de lanzar goals paralelos sobre el
  mismo alcance.
- Timeout parcial: un 504 de `observe_goal` con snapshot `goal_status=running`
  conserva refs/evidencias, pero ya no publica `closure_status=blocked`,
  `closure_needs_rework=true` ni `recommended_action=replan`.

Tests focales ejecutados en remoto con PATH del servidor:

```text
go test -count=1 ./modulos/orquesta-autoprogramming -run 'TestBuildAutoprogrammingProgrammableWorkV0(SecuenciaWriteSetDeclaradoSolapado|SequencesRepairableWriteSetOverlap|HonraWriteSetYDependsOnPorTarea)'
go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPAutoprogrammingObserveGoalHTTPHandlerV0Timeout'
go test -count=1 ./modulos/orquesta-runtime-codex-appserver -run 'TestServerCodexAppServerGoalBackendV0LaunchBloqueaWorkdirInexistenteSinRecrearloV0'
go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestRunSupervisorGoalFirstResidentPreparaReworkPorBloqueoOperativoRecuperableV0'
go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-mcp
git diff --check
```

Pendiente para cierre: compilar/desplegar el binario remoto con estos cambios y
demostrar por API que los goals actuales o un repro nuevo no quedan en 504/409
ni `running` falso.

## Verificacion amplia pendiente

Tras los tests focales, Codex lanzo una verificacion remota mas amplia:

```text
go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server \
  ./modulos/orquesta-app-codex-stack \
  ./modulos/orquesta-runtime-codex-appserver \
  ./modulos/orquesta-autoprogramming ./modulos/orquesta-web ./modulos/orquesta-mcp
```

Resultado: fallo bajo concurrencia con goals/app-server vivos.

Fallos observados:

- `cmd/orquesta-server`:
  `TestServerGoalBackendFromEnvV0ClaudeProcessLanzaYObservaResultadoV0`
  termino con `claude_goal_result_invalid`.
- `cmd/orquesta-server`:
  `TestServerGoalBackendFromEnvV0GeminiProcessLanzaYObservaResultadoV0`
  termino con `gemini_goal_result_invalid`.
- `cmd/orquesta-server`:
  `TestFlakyHarnessV0RepiteCasoDirectorRecursiveFakeRuntimeV0` fallo porque
  un child de compilacion fue terminado.
- `modulos/orquesta-app-codex-stack`:
  `TestSimulacionDeterministaFallosGoalFirstV0` supero el umbral temporal
  esperado (`elapsed` cercano a 66s).

Paquetes que si pasaron en esa tanda:

- `modulos/orquesta-server`
- `modulos/orquesta-runtime-codex-appserver`
- `modulos/orquesta-autoprogramming`
- `modulos/orquesta-web`
- `modulos/orquesta-mcp`

Lectura provisional: no es una regresion focal de los patches ya aplicados,
pero si otro sintoma estructural del frente 208. El sistema mezcla pruebas que
asumen proveedor/proceso limpio, app-server residente y goals vivos en el mismo
host; cuando hay concurrencia, aparecen timeouts, resultados invalidos o kills
de compilacion. No se debe cerrar el bug hasta tener un harness remoto aislado
o un modo de drenaje/parada que deje el servidor en estado verificable.

## Subfallos a analizar juntos

### 208A - Workdir vivo stale

El servidor podia seguir vivo con una variable de control apuntando a un
worktree retirado. Orquesta debe detectar esta condicion antes de lanzar goals,
degradar con diagnostico accionable o reconfigurar por superficie canonica. No
debe arrancar trabajo contra rutas inexistentes.

### 208B - Checkpoint-only presentado como running

El backend/app-server materializa checkpoint temprano y artefactos parciales,
pero `observe` conserva `goal_status=running` aunque el resultado durable diga
`blocked/checkpoint_started`. Debe haber una fuente de verdad clara:
checkpoint parcial no equivale a progreso vivo indefinido.

### 208C - Run control no propaga stop/cancel al backend goal-first

`runs/control` escribe estado local `stop_requested`, pero no confirma parada
del backend. El modo `forced=true` tampoco reconcilio. Un control plane no puede
dejar el estado en "pedido" si el backend no recibe la senal o no se limpia.

### 208D - Batch goal-first acepto write-sets solapados

La request contenia dos tareas con `scripts` en el write-set. Fue error del
operador, pero Orquesta ya tenia write-sets por tarea y deberia serializar,
rechazar o pedir replan si dos goals paralelos pisan el mismo alcance.

### 208E - Verificacion amplia no aislada de procesos residentes

El remoto conserva `orquesta-server-claude`, `codex app-server`, sandboxes y
`go test` lanzados por goals anteriores. Las pruebas amplias de servidor y
backend provider no pueden considerarse diagnostico limpio mientras compartan
host, caches y backend con goals vivos. Orquesta necesita un modo de prueba
aislado o una fase de drain/cleanup gobernada para no convertir ruido de
concurrencia en falso rojo o falso verde.

## Hipotesis estructural

Hay al menos tres fuentes de verdad parcialmente desacopladas:

- estado goal-first persistido;
- artefactos `checkpoint_started` / `orquesta_goal_result`;
- estado/proceso real del backend app-server/tmux.

Cuando divergen, Orquesta conserva `running` o `stop_requested` en vez de
resolver a un terminal explicito (`blocked`, `stopped`, `needs_rework`) con
accion causal. Esto genera falsos vivos, cuelgues de cola y dificulta la
autorreparacion.

## Criterios de cierre

- `prepare-run` o el scheduler detecta write-sets solapados dentro del batch y
  los serializa/rechaza/replantea con evidencia.
- Si el workdir configurado no existe, el goal queda `blocked` con razon
  `workdir_unavailable` antes de arrancar el backend.
- `observe` reconcilia `orquesta_goal_result.status=blocked` y checkpoint-only
  como estado no terminal/publicable pero no como `running` indefinido.
- `runs/control stop|cancel forced=true` confirma backend parado o publica una
  accion de cleanup ejecutable; no queda `running` sin proceso verificable.
- Smoke remoto o focal demuestra que un goal de prueba puede lanzarse,
  bloquearse/pararse/repararse y limpiar procesos sin tocar `uso-app` ni otros
  servicios.

## Accion inmediata recomendada

No lanzar mas goals amplios hasta reconciliar los cuatro goals actuales o
aislarlos. Priorizar el cierre del control goal-first y la deteccion de
write-set solapado; Telegram/nightly queda por detras del nucleo.
