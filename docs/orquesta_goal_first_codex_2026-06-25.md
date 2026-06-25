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

`modulos/orquesta-runtime-codex-goal` define el adaptador Codex:

- `CodexGoalStartPacketV0`: paquete compacto para iniciar un Codex Goal.
- `CodexGoalStarterPortV0`: puerto real de composicion para crear el goal.
- `CodexGoalLauncherV0`: implementa el launcher neutral usando ese puerto.
- `CodexGoalObserverPortV0`: puerto real de composicion para observar el goal.
- `CodexGoalObserverV0`: convierte observaciones a `GoalWorkResultV0` validado.

El adaptador no conoce HOME, modelo, proveedor, OAuth, token, command path ni
filesystem productivo. El arranque real queda en composicion opt-in.

El 2026-06-25 se cablea en `cmd/orquesta-server` un transporte real opt-in con
`ORQUESTA_CODEX_GOAL_BACKEND=app_server_proxy`. La CLI local de Codex no expone
un subcomando estable `goal`, y usar `codex exec` no equivaldria al loop
persistente de Codex Goal; por eso el wiring usa `codex app-server proxy`
contra un daemon local de Codex ya disponible. Sin esa variable, la composicion
no expone launcher/observer y Orquesta publica capacidad goal faltante cuando
`IdleSelfImprovementGoalFirst` esta activo.

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
stack. Si falta el socket local o la instalacion standalone requerida por
`codex app-server daemon`, se inyecta un backend goal degradado con reason code
compacto (`codex_app_server_control_socket_missing`,
`codex_app_server_standalone_missing`, etc.). Esto evita volver al loop legacy
cuando el operador habia pedido goal-first y deja la accion pendiente clara.

## `/nueva-app` goal-first

El 2026-06-25 `/api/v0/apps/director` queda conectado de forma opt-in a
goal-first: `StartAppDirectorV0` persiste el intake/run, compila un
`GoalWorkSpecV0` desde `AppSpecV0` y, si la composicion inyecta
`GoalLauncher`, lanza el goal y devuelve `run_ref`, `goal_ref`,
`external_goal_ref`, `goal_status`, `goal_launch_receipt` y
`director_execution_mode=goal_first`. En ese camino no ejecuta el loop legacy ni
encola el run para el supervisor historico. Si la composicion no inyecta goal,
el resultado declara `director_execution_mode=legacy_director_loop`.

La web `/nueva-app` acepta esa respuesta, muestra las refs dentro del bloque
`director` y observa por `POST /api/v0/apps/director/goal/observe` con polling
acotado. Sin `ORQUESTA_CODEX_GOAL_BACKEND=app_server_proxy`, el puerto no se
inyecta y se conserva el flujo legacy de Director/agentes.

El backend `app_server_proxy` observa `thread/goal/get`; cuando el goal queda
terminal lee `thread/read` con `includeTurns=true` y extrae de la respuesta final
un marcador estructurado:

```text
ORQUESTA_GOAL_RESULT_V0 {"summary":"...","artifact_refs":[],"required_test_results":[],"domain_receipt_refs":[],"evidence_refs":[]}
```

Las refs del marcador se convierten a `GoalWorkResultV0` y pasan por el
validador de cierre de Orquesta. Si el marcador falta o no contiene las
evidencias/artefactos exigidos por el spec, Orquesta no inventa refs: el goal
puede estar `complete`, pero el run queda bloqueado por cierre no aceptado.

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

## Migracion propuesta

1. Mantener el loop historico intacto.
2. Usar `GoalWorkSpecV0` para nuevas tareas de autoprogramacion acotadas.
   Estado 2026-06-25: `orquesta-autoprogramming` ya compila `goal_specs[]`
   cuando `goal_migration=goal_ready`, y `POST
   /api/v0/autoprogramming/prepare-run` los expone con `run_ref` desde la
   composicion Codex stack; los gateways los conservan como passthrough y no
   lanzan runtime.
3. Cablear un launcher real de Codex Goal en `cmd/orquesta-server` solo cuando
   exista puerto seguro para crear/observar goals.
   Estado 2026-06-25: `ORQUESTA_CODEX_GOAL_BACKEND=app_server_proxy` inyecta
   starter/observer por `codex app-server proxy`, hace preflight de
   `thread/loaded/list`, observa `thread/read` y traduce
   `ORQUESTA_GOAL_RESULT_V0` a refs de cierre; falta ejecutar smoke real con
   daemon en ventana operativa.
4. Ejecutar smoke no-OPES temporal con repo de prueba.
5. Ejecutar smoke OPES temporal acotado de un derivado.
6. Marcar rutas antiguas como legacy cuando tengan equivalencia goal-first
   probada.
   Estado 2026-06-25: `StartAppDirectorV0`, MCP/REST y `/nueva-app` ya exponen
   `director_execution_mode` para distinguir `goal_first` de
   `legacy_director_loop`; falta cerrar la matriz completa y smokes reales
   equivalentes antes de retirar rutas historicas.
7. Conectar el contrato de `/nueva-app` a `GoalWorkSpecV0` y launcher
   goal-first.
   Estado 2026-06-25: hecho de forma opt-in para lanzamiento, refs publicas,
   observacion y cierre validado por Orquesta cuando el goal devuelve marcador
   estructurado.

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
