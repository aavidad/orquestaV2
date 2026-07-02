# Incidencia: smoke real Nueva App no cierra app completa

Fecha: 2026-07-02.

Bug inventario: `BUG-ORQ-20260702-122`.

## Contexto

Se ejecuto `scripts/smoke_goal_first_app_server_real.sh` con el Orquesta local
`fe06470557`, backend `app_server_tmux`, OPES desactivado y workspace temporal
aislado en `/tmp/orquesta-smokes/orquesta-goal-first-app-server.1PvR9N`.

El flujo probado fue el de app autonoma real:

- `POST /api/v0/apps/director` con `director_execution_mode=goal_first`.
- Solicitud `crear_app_completa` para una app web/API minima en Go.
- Observacion por `POST /api/v0/apps/director/goal/observe`.

## Resultado observado

El arranque fue aceptado:

- `run_ref=run-spec-smoke-goal-first-req-smoke-goal-first-b3c054d4d4ed80a32a726f319297cdc1`
- `goal_ref=goal-ref-app-director-run-spec-smoke-goal-first-req-smoke-goal-first-b3c054d4d4ed80a32a726f319297cdc1`
- `external_goal_ref=019f2393-ed26-7061-bd3c-535cdba5be14`

En el poll 22 el goal quedo bloqueado:

- `goal_status=blocked`
- `run_status=bloqueada`
- `current_phase=brainstorming_arquitectura`
- `recommended_action=replan`
- `closure_status=blocked`
- `summary=codex_app_server_goal_active_timeout`
- `artifact_refs=0`

El unico fichero bajo `generated-apps` fue un receipt inicial invalido:

`project/generated-apps/smoke-goal-first/docs/orquesta_goal_result_goal-ref-app-director-run-spec-smoke-goal-first-req-smoke-goal-first-b3c054d4d4e.json`

Ese receipt declara:

- `schema_version=orquesta_goal_result.v0`
- `status=invalid`
- `summary=Resultado durable inicial creado antes de construir y verificar la app.`
- `artifact_refs=[]`
- `evidence_refs=[]`

Al fallar, el smoke intento shutdown normal del backend tmux y recibio HTTP 409
con `status=backend_still_running`. El proceso temporal de app-server quedo
vivo y se limpio manualmente por socket exacto del smoke.

## Lectura arquitectonica

No es un fallo aislado de la UI de Nueva App. Repite el eje de
Goal-first/runtime ya abierto en `BUG-ORQ-20260701-073`,
`BUG-ORQ-20260701-079`, `BUG-ORQ-20260701-085` y `BUG-ORQ-20260701-088`:

- Orquesta puede lanzar el goal y observar estado.
- El agente puede crear un receipt inicial invalido antes de materializar
  artefactos utiles.
- El timeout bloquea la ejecucion en fase temprana sin promover de forma
  automatica a rework/retry acotado.
- El shutdown normal conserva correctamente `backend_still_running`, pero el
  operador/smoke aun necesita limpieza posterior cuando el backend queda vivo.

## Accion esperada

Para poder decir que Orquesta programa una app completa de forma autonoma, el
flujo Nueva App debe demostrar en smoke real que:

1. Materializa codigo de aplicacion bajo `generated-apps`.
2. Mantiene write-set y arquitectura declarada.
3. Ejecuta verificacion local o publica evidencia suficiente.
4. Cierra con `goal_status=complete`, `run_status=cerrada`,
   `closure_status=accepted` y `artifact_refs/evidence_refs` no vacios.
5. Si queda en receipt inicial invalido o brainstorming, replanifica de forma
   acotada sin volver al loop legacy y sin quemar cuota hasta timeout.

## Avance Orquesta

`runs.supervisor` en `resident_mode` prepara ahora un rework goal acotado cuando
el estado goal-first ya quedo terminal/rework por
`codex_app_server_goal_active_timeout` y no hay artefactos ni receipts. La
accion conserva el mecanismo goal-first, devuelve `repair_run_refs`, marca el
goal fuente para idempotencia y no relanza si ya existen artefactos, para evitar
pisar trabajo recuperable.

Evidencia focal:

- `TestRunSupervisorGoalFirstResidentPreparaReworkPorTimeoutInicialSinArtefactosV0`
- `TestRunSupervisorGoalFirstResidentNoRelanzaTimeoutConArtefactosV0`

El `GoalWorkSpecV0` inicial de Nueva App incorpora ahora una politica de fase
obligatoria para `brainstorming_arquitectura`: debe producir un artefacto, plan
verificable o bloqueo terminal antes de ampliar contexto, y no basta con un
receipt inicial invalido. La politica viaja como `ContextRef` `phase_policy` y
como criterio de aceptacion del goal, de modo que el adaptador Goal-first recibe
la frontera antes del primer intento.

Evidencia focal:

- `TestStartAppDirectorV0GoalFirstLanzaGoalYNoEjecutaLoopLegacy`

Smoke real posterior con Orquesta `cab6104cd5`, backend `app_server_tmux` y
directorio temporal conservado en
`/tmp/orquesta-goal-first-app-server.SbBpHg`: la app ya materializo codigo,
docs, tests y handoff bajo `generated-apps/smoke-goal-first`, y `observe`
publico `artifact_refs`/`evidence_refs`. El cierre aun no fue aceptado porque,
tras disparar `goal-ref...-rework-1`, la respuesta mezclaba
`goal_status=running` del rework con `closure_status=blocked`,
`closure_needs_rework=true` y `summary=codex_app_server_goal_active_timeout`
del intento anterior.

Avance aplicado: `ObserveAppDirectorGoalV0` refresca la observacion publica
cuando un intento terminal lanza un nuevo goal de rework y el estado actual ya
esta `running`; en ese caso no publica la closure bloqueada ni el summary
terminal del intento previo como estado vigente del rework.

Evidencia focal:

- `TestObserveAppDirectorGoalV0LanzaReworkGoalSiPolicyYPuertoDisponibles`
- `TestObserveAppDirectorGoalV0LanzaReworkGoalPorTimeoutActivoV0`

## Smoke posterior con proyeccion corregida

Smoke real posterior con Orquesta `edc5a79e`, backend `app_server_tmux` y
directorio temporal conservado en
`/tmp/orquesta-smokes/orquesta-goal-first-app-server.UbU4HI`: la proyeccion ya
no publica la closure bloqueada previa mientras el rework causal esta
`running`. El flujo materializo una app real bajo
`generated-apps/smoke-goal-first` con `README.md`, `pyproject.toml`, codigo
hexagonal Python, tests `test_notes_service.py`/`test_http_api.py`, manuales y
handoff; `observe` publico `artifact_refs` materializadas.

El cierre sigue abierto: en el poll 112 el rework
`goal-ref...-rework-1` quedo `goal_status=blocked`, `run_status=bloqueada`,
`closure_status=blocked`, `current_phase=brainstorming_arquitectura`,
`recommended_action=replan` y `summary=codex_app_server_goal_active_timeout`.
El resultado durable seguia `status=invalid` con resumen `rework en progreso;
cierre pendiente de documentacion profunda, pruebas y verificacion de
artefactos`.

Lectura: el bug ya no es solo materializacion. Orquesta consigue lanzar,
observar, replanificar y detectar artefactos, pero no cierra de forma autonoma
el contrato final cuando el rework materializa codigo y pruebas sin escribir un
`orquesta_goal_result.v0` terminal aceptable. Hay que cerrar la regla de
reconciliacion/cierre para "artefactos + tests presentes + receipt no terminal"
sin volver al loop legacy ni aceptar falsos verdes.

Hallazgo colateral documentado aparte: el servidor temporal lanzo un goal de
autoprogramacion idle dentro del smoke aunque el escenario pretendia validar
solo Nueva App aislada. Ver `BUG-ORQ-20260702-123`, cerrado haciendo que el
servidor acepte `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_DISABLED=true` como
opt-out global y que el smoke lo exporte. Los ceros de `TARGET_QUEUE` y
`MAX_REQUESTS` quedan solo como defensa adicional.

## Avances posteriores remotos

Smoke real posterior con Orquesta `c703f4d0ad` y directorio temporal
conservado en `/tmp/orquesta-goal-first-app-server.6uPTwz`: el fix anterior
evito el falso `closure_status=blocked` mientras el rework estaba activo, pero
el rework agoto el unico intento permitido y termino de nuevo con
`codex_app_server_goal_active_timeout` pese a existir artefactos, docs y tests
materializados.

Avance aplicado: el spec goal-first de Nueva App declara ahora
`Budget.MaxRuntimeSeconds=600` y dos reworks causales (`MaxReworkGoals=2`) para
permitir un primer rework de materializacion y un segundo intento acotado de
reparacion/cierre del receipt final sin caer al loop legacy.

Evidencia focal:

- `TestStartAppDirectorV0GoalFirstLanzaGoalYNoEjecutaLoopLegacy`
- `TestObserveAppDirectorGoalV0BloqueaSiReworkGoalAgotaPresupuesto`

Smoke real posterior con Orquesta `0906935bf8` y directorio temporal conservado
en `/tmp/orquesta-goal-first-app-server.5dwqlc`: el segundo rework permitio que
el agente actualizara el receipt a `status=complete`, con `artifact_refs`,
`artifact_paths`, `required_test_results passed` y `evidence_refs`, pero
`observe` siguio viendo el backend como `running` hasta timeout y finalmente
bloqueo el run sin ingerir ese receipt terminal materializado.

Avance aplicado: el scanner de materialized refs lee ahora
`orquesta_goal_result*.json` completos, valida el cierre con el
`GoalClosureValidator` y persiste `LastResult`/`LastClosure` aceptados aunque el
backend todavia no haya devuelto el resultado por API.

Evidencia focal:

- `TestCodexStackObserveAppDirectorGoalExecutorV0IngiereReceiptTerminalMaterializadoV0`
- `go test -count=1 ./modulos/orquesta-app-codex-stack`

## Error de cuota local durante revalidacion

Smoke real posterior con Orquesta `6945acc914` y directorio temporal conservado
en `/tmp/orquesta-goal-first-app-server.u5tcIA`: la revalidacion no llego a
comprobar el cierre porque `observe` fallo en el poll 8 con HTTP 500
`observe_app_director_goal_http_error`; el log del backend Codex contenia
`rollout writer failed: Quota exceeded (os error 122)`.

Avance aplicado: los errores de cuota/espacio local del app-server Codex se
normalizan como `codex_app_server_storage_quota_exceeded` para que `observe`
los proyecte como issue publico accionable en vez de 500 generico. Ver
`BUG-ORQ-20260702-124`.

Evidencia focal:

- `TestCodexAppServerIssueCodeForErrorV0ClasificaDiagnosticosV0`
- `TestCodexGoalObserverV0ClasificaQuotaFilesystemBackendSinIssueCodeV0`
- `TestMCPObserveAppDirectorGoalErrorResultFromErrorV0PreservaQuotaFilesystemV0`
- `TestExternalWorkGoalFirstKnownLaunchFailureReasonV0ClasificaQuotaFilesystem`

Smoke real posterior con Orquesta `5a71671fc5` usando `ORQUESTA_SMOKE_PARENT`
y `TMPDIR` bajo `/srv/orquesta-self/runtime` para evitar `/tmp`, directorio
conservado en
`/srv/orquesta-self/runtime/smokes-goal-first/orquesta-goal-first-app-server.QsBxZZ`:
el 500 temprano por cuota ya no se reproduce, `observe` llega a respuesta
publica HTTP 200, pero en poll 56 termina bloqueado:

- `goal_status=blocked`
- `run_status=bloqueada`
- `closure_status=blocked`
- `current_phase=brainstorming_arquitectura`
- `generated_apps_present=0`
- `summary` empieza por `external_runtime_blocker: sandbox shell still fails
  before execution with quota exceeded mounting .git`; indica que `apply_patch`
  no puede crear los directorios de write-set requeridos y no puede materializar
  ni verificar artefactos.

El shutdown HTTP devuelve `backend_still_running` con `active_work_count=2`,
pero la limpieza cooperativa del smoke elimina la sesion tmux temporal; solo
queda el servidor persistente aislado esperado `orquesta-server-latest`. BUG-122
sigue abierto: falta un smoke real con `goal_status=complete`,
`run_status=cerrada`, `closure_status=accepted` y refs de artefactos/evidencias
no vacios, o documentar una frontera externa reproducible fuera de Orquesta.

Avance aplicado: el backend `app_server_tmux` prepara antes del primer turno los
directorios declarados en `write_set` cuando son rutas relativas seguras dentro
del `CWD`, sin convertir write-sets Markdown en directorios y sin aceptar
escapes ni `.git`. Esto evita que el agente dependa de `apply_patch` para crear
el directorio raiz del artefacto.

Evidencia focal:

- `TestServerCodexAppServerGoalBackendV0PreparaDirectoriosWriteSetAntesDelTurnV0`
- `TestServerCodexAppServerGoalBackendV0LanzaThreadGoalYTurnV0`

Smoke real posterior con Orquesta `7bff5cb89c`, directorio conservado en
`/srv/orquesta-self/runtime/smokes-goal-first/orquesta-goal-first-app-server.gcZ6Yj`:
el write-set raiz `generated-apps/smoke-goal-first` se crea antes del turno y
el smoke ya no bloquea en poll 56 por sandbox/cuota. Termina en poll 59 con
`goal_status=blocked`, `run_status=bloqueada`, `closure_status=blocked`,
`summary=codex_app_server_goal_active_timeout` y sin ficheros bajo el write-set.
El script informo `generated_apps_present=1` solo porque existia el directorio
preparado.

Avance aplicado: el diagnostico `generated_apps_present` del smoke cuenta ahora
ficheros reales bajo `generated-apps` y no directorios preparados.

Evidencia focal:

- `TestSmokeGoalFirstAppServerRealDiagnosesAppServerAuthMissingV0`
