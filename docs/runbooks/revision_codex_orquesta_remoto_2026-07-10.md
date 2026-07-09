# Revision Codex de Orquesta remoto

Fecha: 2026-07-10  
Rol Codex: revisor externo, no programador principal.  
Rama: `trabajo/plataforma-agentes`  
Commit revisado local/GitHub/remoto: `9583be39c1136c623d1deee7587ffd9a211a999d`

## Resumen para Claude

Orquesta remoto esta ejecutando y ha producido una entrega util, pero no puede
declararse autonoma ni cerrada. El goal `g01` del frente de control/no-cuelgues
quedo completado y su rework `g01-rework-1` paso de `blocked/checkpoint_started`
a `complete` con tests requeridos en verde. Sin embargo, el servidor vivo sigue
ejecutando el binario antiguo y el estado publico conserva runs stale,
incoherencias de estado vivo y artefactos de ejecucion dentro del arbol fuente.

Decision de revision:

- Aceptar como avance parcial la entrega `goal-ref-task-autoprogramming-f0ad7bb152dd-g01`
  y su rework `goal-ref-task-autoprogramming-f0ad7bb152dd-g01-rework-1`.
- No aceptar cierre global de autonomia.
- No lanzar mas tareas amplias hasta resolver F3/F5.

## Estado remoto observado

API `/api/status`:

- `status=running`
- `availability_status=running`
- `availability_reason=server_ready`
- Binario vivo: `9541e2f0019819e58577e8c2fcbc202e602f636321e3a66f42ad618e0a876b47`
- Inicio del proceso vivo: `2026-07-09T22:41:44Z`
- El binario vivo no corresponde al commit pusheado `9583be39c`.

Supervisor:

- `last_supervisor_queue_size=2`
- `last_supervisor_stop_public=budget_max_ticks`
- `supervisor_error_ticks=19683`
- `goal_observer` observado como `ok/skipped` segun tick, con issues acumulados.

API `/api/v0/autoprogramming/status`:

- `queue_count=0`
- `operator_active=13`
- estados activos: `invalid=6`, `running=4`, `blocked=2`, `complete=1`
- `stale_running=57`
- `resolved_runs` oscilo entre 0 y 1 durante la observacion.

Procesos vivos:

- `orquesta-server-claude`
- `codex app-server`
- Durante la revision aparecieron `go test` lanzados por Orquesta; terminaron
  antes de la decision final.

## Entrega aceptada parcialmente

Resultado base:

- `cmd/orquesta-server/docs/orquesta_goal_result_goal-ref-task-autoprogramming-f0ad7bb152dd-g01.json`
- `status=complete`
- `missing_refs=[]`
- Required tests: `5/5 passed`

Rework:

- `cmd/orquesta-server/docs/orquesta_goal_result_goal-ref-task-autoprogramming-f0ad7bb152dd-g01-rework-1.json`
- Estado inicial observado: `blocked`, `reason_code=checkpoint_started`
- Estado posterior observado: `complete`
- `missing_refs=[]`
- Required tests: `5/5 passed`
- Resumen declarado: se conserva la implementacion previa verificada y se
  corrige el resultado durable para declarar solo artefactos dentro del
  write-set autorizado.

Tests declarados por el resultado:

- `go test -count=1 ./modulos/orquesta-runtime-codex-appserver`
- `go test -count=1 ./modulos/orquesta-app-codex-stack ./modulos/orquesta-server ./cmd/orquesta-server`
- `go test -count=1 ./modulos/orquesta-autoprogramming ./modulos/orquesta-mcp ./modulos/orquesta-app-codex-stack ./modulos/orquesta-server ./cmd/orquesta-server`
- `bash -n scripts/orquesta_server_ctl.sh scripts/orquesta_server_deploy.sh scripts/orquesta_smoke_nightly.sh`
- `git diff --check`

Codex no reejecuto manualmente esos tests en remoto; la aceptacion parcial se
basa en la evidencia durable escrita por Orquesta y en la desaparicion de los
procesos `go test` vivos.

## Diff remoto pendiente de revision

El remoto quedo sucio con:

- `docs/inventario_bugs_orquesta_2026-06-30.md` modificado por Orquesta.
- `cmd/orquesta-server/docs/checkpoint_started_goal-ref-task-autoprogramming-f0ad7bb152dd-g01-rework-1.txt`
- `cmd/orquesta-server/docs/orquesta_goal_result_goal-ref-task-autoprogramming-f0ad7bb152dd-g01-rework-1.json`
- `cmd/orquesta-server/docs/orquesta_goal_result_goal-ref-task-autoprogramming-f0ad7bb152dd-g01.json`

Como revisor, no se deben commitear automaticamente los artefactos bajo
`cmd/orquesta-server/docs/`. Son evidencia operativa del patron F4: artefactos
de ejecucion aparecen en rutas versionables. Si se conservan, debe ser tras una
auditoria de retencion/evidencias; si no, deben moverse a la superficie canonica
de evidencias fuera del arbol fuente.

## No aceptado todavia

No queda cerrado:

- F3: drain gobernado + harness aislado.
- F5: deploy del binario nuevo + startup guard de identidad.
- F1/F2: reconciliador causal unico + control confirmado por identidad runtime.
- F4: destino canonico de artefactos de ejecucion y auditoria de artefactos ya
  versionados.

Motivo: aunque `g01` esta completo, la API sigue mostrando runs stale y estados
desconocidos. En concreto:

- `run-ref-orquesta-100-continuous-20260710-001-goal-01` seguia apareciendo en
  diagnosticos como `estado_vivo_desconocido`.
- Hay tareas remotas antiguas `invalid`, `blocked` y `running`, incluidas
  familias Telegram, envs y limpieza remota.
- El binario vivo sigue siendo anterior al codigo pusheado.

## Siguiente accion recomendada

1. Mantener a Codex como revisor, no como programador manual.
2. Pedir a Orquesta un corte F3 acotado: inventario de procesos propios,
   drain gobernado por identidad runtime y harness aislado reutilizable.
3. No basar ese drain en `runs/control`, porque 208C demuestra que no propaga
   parada fiable al backend.
4. Tras F3, ejecutar F5: deploy atomico del binario `9583be39c` o superior,
   verificando `runtime_identity.binary_sha256`.
5. Solo despues reanudar tareas amplias de autonomia.

## Riesgo principal

Si se lanzan mas goals antes de F3/F5, Orquesta puede seguir produciendo trabajo
util, pero aumenta el ruido: mas runs stale, mas artefactos de ejecucion en
fuente y mas dificultad para distinguir cierre real de reconciliacion parcial.
