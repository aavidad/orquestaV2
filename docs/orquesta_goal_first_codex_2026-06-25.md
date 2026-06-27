# Orquesta goal-first con Codex - 2026-06-25

Este documento fija el corte arquitectonico para adelgazar Orquesta cuando el
runtime disponible ya tiene `goal` persistente. La decision no elimina Orquesta:
la cambia de loop operativo a plano de gobierno, contratos, contexto y
evidencias.

## Decision

Cuando una composicion tenga acceso a Codex Goal, el Codex que arranca el goal
actua como Director operativo interno de ese trabajo.

Orquesta conserva la direccion exterior:

- compila objetivo, reglas, contexto, write-set, tests y artefactos esperados;
- lanza un goal por adaptador opt-in;
- observa `running`, `complete`, `blocked` o `invalid`;
- valida evidencias, tests, artefactos y receipts de dominio;
- cierra o abre un goal de rework acotado.

Orquesta no debe duplicar dentro del servidor el loop que Codex Goal ya mantiene
por si mismo. El ciclo historico `wait_subagents -> review_deliveries ->
run_required_tests -> replan_or_close` queda como compatibilidad y como flujo
para composiciones que aun no tengan goal persistente.

## Nuevo reparto

```text
app externa / usuario / OPES
  -> Orquesta intake
  -> GoalWorkSpecV0
  -> adaptador Codex Goal
  -> Codex Goal dirige y ejecuta el bucle interno
  -> GoalWorkResultV0
  -> validacion de cierre por Orquesta
  -> cierre o nuevo goal de rework
```

Codex Goal decide plan operativo, pasos internos, ediciones, pruebas locales y
continuidad hasta `complete` o `blocked`. Orquesta no fuerza ticks, no espera
por todos los agentes vivos y no recompone el plan interno desde logs.

Orquesta decide si el resultado cierra:

- los tests requeridos deben estar evidenciados;
- los artefactos esperados deben existir por refs;
- los receipts de dominio deben coincidir si el dominio los exige;
- las reglas duras de seguridad, causalidad, refs, datos sensibles y efectos
  externos no autorizados siguen siendo cortes fuertes;
- las reglas blandas siguen siendo advisory y no descartan trabajo recuperable.

## Contratos nuevos

`modulos/orquesta-goal` define el contrato neutral:

- `GoalWorkSpecV0`: objetivo, refs, reglas, skills, write-set, tests,
  artefactos, presupuesto y politica de cierre/rework.
- `GoalLaunchReceiptV0`: receipt de lanzamiento aceptado/rechazado por el
  adaptador.
- `GoalWorkResultV0`: observacion de `running`, `complete`, `blocked` o
  `invalid`. `invalid` es diagnostico observable, no cierre aceptado.
- `GoalWorkLauncherPortV0`: puerto para lanzar un goal.
- `GoalWorkObservationPortV0`: puerto para observarlo.
- `GoalWorkClosureValidatorPortV0`: puerto para validar cierre.
- `StartGoalWorkV0`/`ObserveGoalWorkV0`: lifecycle neutral por puertos para
  lanzar, persistir estado, observar y validar closure sin importar runtime,
  servidor, filesystem, colas ni dominio.

`modulos/orquesta-runtime-codex-goal` define el adaptador Codex:

- `CodexGoalStartPacketV0`: paquete compacto para iniciar un Codex Goal.
- `CodexGoalStarterPortV0`: puerto real de composicion para crear el goal.
- `CodexGoalLauncherV0`: implementa el launcher neutral usando ese puerto.
- `CodexGoalObserverPortV0`: puerto real de composicion para observar el goal.
- `CodexGoalObserverV0`: convierte observaciones a `GoalWorkResultV0` validado.

El adaptador no conoce HOME, modelo, proveedor, OAuth, token, command path ni
filesystem productivo. El arranque real queda en composicion opt-in.

El 2026-06-25 se cablea en `cmd/orquesta-server` un transporte real opt-in con
`ORQUESTA_CODEX_GOAL_BACKEND=app_server_proxy`. El 2026-06-26 se anade
`app_server_stdio` para instalaciones donde `codex app-server --stdio` responde
pero `codex app-server proxy` no obtiene respuesta del socket manual. La CLI
local de Codex no expone un subcomando estable `goal`, y usar `codex exec` no
equivaldria al loop persistente de Codex Goal; por eso el wiring usa
`codex app-server` como frontera de composicion. Sin esa variable, la
composicion no expone launcher/observer y Orquesta publica capacidad goal
faltante cuando `IdleSelfImprovementGoalFirst` esta activo.
Desde el 2026-06-26, si `ORQUESTA_CODEX_GOAL_BACKEND` esta configurada y
`ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_GOAL_FIRST_ENABLED` no esta definida,
la automejora residente deriva automaticamente goal-first. El operador puede
forzar compatibilidad legacy con
`ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_GOAL_FIRST_ENABLED=false`; la
configuracion efectiva publica la fuente
`derived_from_codex_goal_backend` y el diagnostico
`idle_self_improvement_goal_first_derived`.

`orquesta-server` ya conserva para automejora goal-first el spec emitido, el
receipt de lanzamiento, el resultado observado y la validacion de cierre. En
`GET /api/v0/server/status` esos datos aparecen como resumen publico
`idle_self_improvement_goal`: refs, estado y contadores. No se publica objetivo,
paths, comandos ni payloads completos.
`/api/v0/server/readiness` expone goal activo, estado, reason code y cierre
aceptado como senal informativa no bloqueante. `/api/v0/operational-status/query`
conserva contadores residentes y anade contadores `goal_*`, refs opacas, salud,
actividad y bloqueo semantico si el goal queda `invalid` o `blocked`. El
statefile local puede contener spec/receipt/result/closure completos para
restauracion, pero no es API publica.

La composicion hace un preflight rapido del backend app-server al arrancar el
stack. El cliente manda `initialize`, `initialized` y la llamada real
`thread/loaded/list`, manteniendo stdin abierto hasta recibir `id=2`; esto es
necesario para `app_server_stdio` y compatible con `app_server_proxy`. Si falta
el socket local, la instalacion standalone requerida por `codex app-server
daemon` o el backend no responde, se inyecta un backend goal degradado con
reason code compacto (`codex_app_server_control_socket_missing`,
`codex_app_server_standalone_missing`, etc.). Esto evita volver al loop legacy
cuando el operador habia pedido goal-first y deja la accion pendiente clara.

## `/nueva-app` goal-first

El 2026-06-27 `/api/v0/apps/director` queda goal-first estricto por defecto:
`StartAppDirectorV0` persiste el intake/run, compila un `GoalWorkSpecV0` desde
`AppSpecV0` y exige bundle Goal completo para `director_execution_mode` vacio o
`goal_first`. Si la composicion inyecta `GoalLauncher`, `GoalObserver`,
`GoalClosureValidator` y `GoalStateStore`, lanza el goal y devuelve `run_ref`,
`goal_ref`, `external_goal_ref`, `goal_status`, `goal_launch_receipt` y
`director_execution_mode=goal_first`. En ese camino no ejecuta el loop legacy ni
encola el run para el supervisor historico. Si no hay backend Goal configurado,
la llamada falla como `goal_backend_unavailable` y conserva la evidencia publica
del error; no cae al loop historico.

La web `/nueva-app` acepta esa respuesta, muestra las refs dentro del bloque
`director` y observa por `POST /api/v0/apps/director/goal/observe` con polling
acotado. La compatibilidad de Director/agentes historica solo se activa si el
caller transporta `director_execution_mode=legacy_director_loop` y la
composicion ha habilitado el opt-in legacy correspondiente; sin esa doble llave,
la ausencia de backend Goal es un error operativo y no un fallback.

Los backends `app_server_proxy` y `app_server_stdio` observan
`thread/goal/get`; cuando el goal queda terminal leen `thread/read` con
`includeTurns=true` y extraen de la respuesta final un marcador estructurado o,
si el hilo no aporta marcador legible, el archivo durable
`orquesta_goal_result_v0.json` escrito bajo el write-set:

```text
ORQUESTA_GOAL_RESULT_V0 {"goal_ref":"...","summary":"...","artifact_refs":[],"required_test_results":[],"domain_receipt_refs":[],"evidence_refs":[]}
```

Las refs del marcador o del archivo durable se convierten a `GoalWorkResultV0`
y pasan por el validador de cierre de Orquesta. Si faltan las
evidencias/artefactos exigidos por el spec, Orquesta no inventa refs: el goal
puede estar `complete`, pero el run queda bloqueado por cierre no aceptado. El
archivo durable debe incluir `goal_ref` coincidente.

## Relacion con el Director actual

`orquesta-director-operativo`, `orquesta-app-director-service` y
`OperationalDirectorPlanStateV0` no se borran en este corte. Siguen siendo
compatibilidad para:

- runs antiguos;
- smokes ya documentados;
- composiciones sin Codex Goal;
- waits por ola/cohorte que aun dependan de `WorkflowTaskStore`;
- pruebas offline del cierre causal historico.

Para trabajo nuevo con Codex Goal, el "Director Operativo" efectivo vive dentro
del goal. Orquesta solo prepara y valida el contrato.

Actualizacion 2026-06-27: en autoprogramacion, las acciones operativas tambien
siguen esa frontera. `/api/v0/autoprogramming/status` publica `observe_goal`
para runs con `GoalWorkStateV0` y no publica `supervise`/`retry`/`review`
legacy salvo que la composicion habilite explicitamente compatibilidad legacy
(`AllowLegacyAutoprogrammingRun` / `ORQUESTA_AUTOPROGRAMMING_LEGACY_DIRECTOR_LOOP`).
El supervisor Codex bloquea ademas la supervision global sin `run_ref` si falta
la marca `director_execution_mode=legacy_director_loop` o si falta el opt-in de
composicion, devolviendo respectivamente
`legacy_supervise_requires_director_execution_mode` o
`legacy_supervise_requires_explicit_opt_in` en vez de entrar al loop historico.
Actualizacion adicional 2026-06-27: incluso con `run_ref`, la supervision de
una run legacy desde el stack Codex exige ahora opt-in de composicion y
`director_execution_mode=legacy_director_loop`. Sin modo legacy devuelve
`legacy_run_supervise_requires_director_execution_mode`; con modo legacy pero
sin opt-in devuelve `legacy_run_supervise_requires_explicit_opt_in`. Las
rutas goal-first se resuelven antes de ese bloqueo y siguen redirigiendo a
`observe_goal`, de modo que la marca legacy no se usa para observar Codex Goal.
Actualizacion adicional 2026-06-27 noche: `orquesta.external_work.run.v0` ya no
degrada a `StartExternalWorkRunV0` legacy solo porque la composicion tenga
`ORQUESTA_EXTERNAL_WORK_LEGACY_DIRECTOR_LOOP=1`. La rama historica requiere
tambien `director_execution_mode=legacy_director_loop`; sin backend Goal y sin
esa marca devuelve `external_work_legacy_director_mode_required`, y con marca
pero sin opt-in devuelve `external_work_legacy_director_loop_opt_in_required`.
Actualizacion adicional 2026-06-27 noche 2: `autoprogramming.prepare_run`
aplica la misma regla. `AllowLegacyAutoprogrammingRun` /
`ORQUESTA_AUTOPROGRAMMING_LEGACY_DIRECTOR_LOOP=1` solo habilita compatibilidad;
no crea runs legacy si el payload no declara
`director_execution_mode=legacy_director_loop`. Sin esa marca devuelve
`autoprogramming_legacy_director_mode_required`; con marca pero sin opt-in
devuelve `autoprogramming_legacy_director_loop_opt_in_required`. Si hay backend
Goal completo y no se pide legacy, la ruta normal sigue lanzando/observando
Goal.

## Migracion propuesta

1. Mantener el loop historico intacto.
2. Usar `GoalWorkSpecV0` para nuevas tareas de autoprogramacion acotadas.
   Estado 2026-06-26: `orquesta-autoprogramming` sigue siendo puro y compila
   `goal_specs[]` cuando `goal_migration=goal_ready`. En la composicion Codex
   stack, `POST /api/v0/autoprogramming/prepare-run` conserva dos rutas: si no
   hay backend goal-first completo, devuelve `goal_specs[]` sin `run_ref` como
   handoff; si estan inyectados `GoalLauncher`, `GoalObserver`,
   `GoalClosureValidator` y `GoalStateStore`, crea un run contenedor sin
   `WorkflowTaskV0` legacy, lanza el goal, persiste `GoalWorkStateV0`, devuelve
   `goal{run_ref,goal_ref,external_goal_ref,goal_status}` y no encola
   `runs/supervise`. Si una composicion configura launcher u observer pero deja
   el bundle incompleto, la ruta falla de forma explicita y no cae al loop
   legacy.
3. Cablear un launcher real de Codex Goal en `cmd/orquesta-server` solo cuando
   exista puerto seguro para crear/observar goals.
   Estado 2026-06-26: `ORQUESTA_CODEX_GOAL_BACKEND=app_server_proxy` o
   `app_server_stdio` inyecta starter/observer por `codex app-server`, hace
   preflight de `thread/loaded/list`, observa `thread/read` y traduce
   `ORQUESTA_GOAL_RESULT_V0` o `orquesta_goal_result_v0.json` a refs de cierre;
   smoke real local cerrado el 2026-06-26 con `app_server_stdio`.
4. Ejecutar smoke no-OPES temporal con repo de prueba.
5. Ejecutar smoke OPES temporal acotado de un derivado.
6. Marcar rutas antiguas como legacy cuando tengan equivalencia goal-first
   probada.
   Estado 2026-06-27: `StartAppDirectorV0`, MCP/REST, `/nueva-app`,
   `runs/supervise`, `autoprogramming/supervise`, `external-work/run`, CLI,
   `/ops` y scripts exponen
   `director_execution_mode` para distinguir `goal_first` de
   `legacy_director_loop`. Las vias de supervision legacy requieren opt-in de
   composicion y la marca explicita; falta cerrar la matriz completa y
   smokes reales equivalentes antes de retirar rutas historicas.
7. Conectar el contrato de `/nueva-app` a `GoalWorkSpecV0` y launcher
   goal-first.
   Estado 2026-06-25: hecho de forma opt-in para lanzamiento, refs publicas,
   observacion y cierre validado por Orquesta cuando el goal devuelve marcador
   estructurado o archivo durable.
8. Conectar autoprogramacion a observacion goal-first.
   Estado 2026-06-26: `orquesta.autoprogramming.observe_goal.v0` y `POST
   /api/v0/autoprogramming/goal/observe` observan el `GoalWorkStateV0` por
   `run_ref`, reutilizan el cierre neutral de `orquesta-app-director-service` y
   solo sincronizan cola como terminal no ejecutable si el goal cierra o bloquea
   el run.
   Estado 2026-06-27 noche: `orquesta-goal` centraliza el lifecycle neutral
   `StartGoalWorkV0`/`ObserveGoalWorkV0`; `/nueva-app` delega launch/observe en
   ese ciclo y `autoprogramming`/`external-work` reutilizan el constructor
   neutral de `GoalWorkStateV0`.

## No hacer

- No meter Codex Goal en `orquesta-core-workflow`, `orquesta-domain-work` ni
  `orquesta-director-operativo`.
- No convertir `complete` del goal en cierre automatico sin evidencias.
- No borrar waits, `PlanState` ni cierre causal historico hasta tener smokes
  equivalentes.
- No mover reglas OPES al contrato neutral.
- No reintroducir rails de contenido para cortar trabajo recuperable.

## Validacion inicial

```bash
go test -count=1 ./modulos/orquesta-goal ./modulos/orquesta-runtime-codex-goal
go test -count=1 .
go test -count=1 ./...
```
