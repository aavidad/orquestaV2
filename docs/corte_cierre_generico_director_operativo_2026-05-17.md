# Corte cierre generico del Director Operativo - 2026-05-17

Este documento es el handoff del tramo siguiente al cierre de P1
`WaitAgentRefs`. El objetivo ya no es acotar la espera: eso queda cerrado para
el stack Codex. El objetivo ahora es cerrar un ciclo causal offline y generico
del Director Operativo, sin Codex real, sin OPES real y sin DB/producto nuevo.

El corte ya tiene un tramo offline integrado en core + `app-director-service` +
`orquesta-app-codex-stack`. Si una pieza de codigo no esta integrada o no tiene
prueba clara, se documenta como pendiente verificable, no como hecho.

## Punto de partida cerrado

P0 del Director Operativo queda cerrado para el primer tramo durable:

- un `OperationalDirectorPlanV0` `ready` puede materializar items
  `launch_subagents`;
- las tasks quedan en `WorkflowTaskStore` con metadata de ola/cohorte y linaje;
- `WorkflowTaskWaitAgentRefsV0` deriva agentes desde `wave_ref`,
  `cohort_ref` o `parent_task_ref`;
- `WorkflowTaskWaitStateV0` registra causa, tasks, agentes objetivo y
  pendientes;
- `ContinueAppDirectorV0` puede reentrar al loop con espera acotada.

P1 `WaitAgentRefs` tambien queda cerrado para el stack Codex:

- `WaitAgentRefs` no vacio limita pending, wait externo e ingesta de
  ACK/deliveries;
- `DrainRunRequestV0.WaitAgentRefs` llega a las observaciones;
- el source Codex no lee/verifica ACKs fuera de scope;
- `DrainRunV0` filtra antes de aplicar observaciones;
- `domain_work` no hace submit/recovery fuera del scope;
- `WaitAgentRefs` vacio conserva compatibilidad legacy de run completo.

La evidencia focal esta documentada en la matriz. Este handoff no reejecuta esa
evidencia.

## Nuevo foco

El corte nuevo es cierre causal offline generico:

```text
wait_subagents cerrado por scope
  -> entregas observadas de la ola/cohorte activa
  -> review_deliveries causal
  -> run_required_tests durables si el modo lo exige
  -> replan_or_close / close
  -> estado vivo del plan reentrable
```

Generico significa que el flujo no depende de Codex ni OPES. La capa neutral
debe razonar por refs, tasks, entregas, reviews, evidencias y puertos. Las
reglas de modo entran como validadores o politica inyectada:

- `programming`: requiere evidencias durables de tests declarados antes de
  cerrar;
- `domain_work`: requiere artefactos esperados y validacion del dominio por
  contrato; no inventa tests de programacion;
- modos desconocidos o incompletos: bloquean o piden contexto, no cierran por
  resumen textual.

La fuente de cierre queda inyectable. `OperationalClosureSource` es opcional:
cuando existe, aporta una decision/evaluacion offline sobre el estado operativo.
El stack Codex ya trae una fuente real para tasks del Director Operativo:
construye `OperationalDirectorClosureRequestV0` desde `WorkflowTaskStore`,
`LoadRunEventsV0`, `WaitAgentRefs` resuelto y refs causales de entrega/review.
No cierra tasks legacy sin metadata `operational_director`. La frontera se
mantiene: no importa Codex, OPES, DB concreta ni runtime real dentro de core.

## RequiredTestEvidenceV0 vigente

`RequiredTestEvidenceV0` es el comprobante durable minimo que el cierre entiende
para tests requeridos. No es un runner, no ejecuta comandos y no sustituye a
`review_deliveries`; solo dice que un test declarado ya tiene resultado
persistido y enlazado a la cadena causal correcta.

Campos relevantes del contrato:

- `evidence_ref`, `run_ref`, `task_ref` y `test_command`;
- `status`: solo `passed` o `failed`;
- `delivery_ref`, `review_request_id`, `review_result_ref` y
  `accepted_review_ref`;
- `occurred_at` y `evidence_refs` adicionales.

La regla de cierre actual es estricta:

- si la task no declara `WorkflowTaskV0.RequiredTests`, el cierre no exige
  `RequiredTestEvidenceV0`;
- si la task declara tests, `OperationalDirectorClosureV0` requiere
  `RequiredTestEvidenceRefs` y un `RequiredTestEvidenceReaderPortV0`;
- cada test requerido debe tener una evidencia `passed` cuyo `test_command`
  coincida exactamente con el test declarado;
- la evidencia debe pertenecer al mismo run, task, delivery, review request,
  review result aceptado y accepted review que el cierre;
- la `evidence_ref` debe estar solicitada en `RequiredTestEvidenceRefs` y la
  evidencia debe traer `evidence_refs` de artefacto/salida real;
- una evidencia `failed`, de otra task, de otra entrega o solo mencionada en un
  summary bloquea el cierre.

`RequiredTestEvidenceStorePortV0` separa reader/writer por interfaz y las
evidencias son inmutables: guardar la misma ref con el mismo payload es
idempotente; guardar la misma ref con payload distinto es conflicto.

`orquesta-state-file` ya persiste estas evidencias y
`orquesta-app-codex-stack` puede encontrarlas desde `RequiredTestEvidenceRefs`
del request o, por compatibilidad, desde los `EvidenceRefs` del review result,
ignorando refs mixtas que no sean evidencias de test. La fuente del stack solo
construye cierre cuando existe cadena causal completa: delivery -> review
requested -> review result accepted -> accepted review. El `PlanState` ya
consume `RequiredTestEvidenceV0` en `run_required_tests`: `passed` causal avanza
a `replan_or_close`, `failed` bloquea con `required-tests-failed` y las refs
aceptadas se pasan al cierre. Lo pendiente es generar esas evidencias mediante
runner/adaptador real por puerto y propagar replan negativo ante fallo.

## Review negativa en PlanState

El codigo local de `app-director-service` ya contiene observacion de review
negativa para el `PlanState`: cuando el step activo es `review_deliveries`, el
outbox esta a cero y el historial durable enlaza `DeliveryRegistered`,
`ReviewRequested`, `ReviewResultRecorded` no aceptado, `ReworkRequested` y
`ReplanDecisionRecorded`, el state marca el step como `changes_requested`,
guarda `rework_request_refs`, `replan_decision_refs`, incrementa
`replan_attempts` y deja el blocker `review-rework-replan-recorded`.

Este tramo queda cerrado como observacion durable del `PlanState`: hay prueba
focal para `changes_requested` y `rejected`, y cobertura de normalizacion y
persistencia de `rework_request_refs`/`replan_decision_refs`. No equivale a
`replan_or_close` completo ni a runner/adaptador real de tests.

## Siguiente incremento

Este incremento debe cerrar el ciclo offline generico posterior a la espera ya
acotada. No dar por hecho codigo nuevo hasta que cada casilla tenga
implementacion, test focal y evidencia en la matriz.

- [x] `review_deliveries` por ola/cohorte: el `PlanState` avanza solo cuando
  todas las tasks del scope activo tienen cadena causal
  `DeliveryRegistered -> ReviewRequested -> ReviewResultRecorded(accepted) ->
  ReviewAccepted`. Una entrega aceptada de otra ola no avanza ni cierra la ola
  actual.
- [x] Tests durables por evidencia: `run_required_tests` consume
  `RequiredTestEvidenceV0` persistido, avanza con `passed`, bloquea con
  `required-tests-failed` ante `failed` y rechaza refs de otra review. En modo
  `programming`, un summary textual o un review sin evidencia de test no basta
  para cerrar. Pendiente separado: runner/validador por puerto, emision
  idempotente de evidencia y replan ante fallo.
- [~] Replan negativo: cerrada la observacion durable de review negativa. El
  `PlanState` guarda refs/attempt para `ReworkRequested` y
  `ReplanDecisionRecorded`. El corte del 2026-05-21 ya convierte followups
  materializados en una nueva espera acotada: `split_task` cuando las nuevas
  `WorkflowTaskV0` existen en `WorkflowTaskStore` y estan reflejadas en
  `run.Tasks`, y `retry_task`/`replace_agent` cuando los agentes followup ya
  estan reflejados en `run.Agents`; ver
  `docs/corte_replan_negativo_followups_split_2026-05-21.md`. Siguen pendientes
  blockers posteriores, tests fallidos y cierre insuficiente como nuevos efectos
  idempotentes de replan.
- [x] Plan state vivo inicial: contrato, stores y persistencia del tramo
  `launch_subagents -> wait_subagents`, con ola/cohorte activa, step activo,
  task refs, agent refs, pending agent refs y `wait_ref`.
- [x] Reentrada inicial desde plan state: `ContinueAppDirectorV0` acepta
  `operational_director_plan_ref`, lee el state y recupera scope de
  `wait_subagents` sin mirar agentes vivos globales.
- [x] Avance inicial tras wait consumido: si el loop queda `quiescent` y los
  agentes pendientes entregaron, el state pasa de `wait_subagents` a
  `review_deliveries`.
- [~] Plan state vivo restante: `run_required_tests` durable, blocker de test
  fallido, observacion de review negativa y razon de cierre/bloqueo ya quedan
  persistidos. Falta materializar replan automatico para blockers posteriores y
  probar replay/idempotencia completa.
- [ ] Evento/comando idempotente: todo avance de review, test, replan y cierre
  debe tener clave idempotente estable por `run_ref`, `task_ref`, `wave_ref` o
  `cohort_ref` y refs causales. Replay no debe duplicar reviews, tests,
  reworks ni cierres.

El orden recomendado ahora es: runner/adaptador real de tests por puerto,
despues replan automatico para casos negativos y por ultimo replay/idempotencia
completa del ciclo. Si una composicion no aporta fuente real de cierre o
validacion, la casilla queda pendiente para esa composicion.

## Corte OperationalDirectorPlanStateV0

El corte inicial del estado vivo queda detallado en
`corte_operational_director_plan_state_v0_2026-05-17.md`.

`OperationalDirectorPlanStateV0` es la foto durable y reentrable del
avance operativo del plan. Su responsabilidad es enlazar el plan inicial con el
estado real observado: ola/cohorte activa, step actual, tasks del scope,
wait state, entregas, reviews, evidencias de tests, blockers, intentos de replan
y razon de cierre o bloqueo.

No sustituye al `OperationalDirectorPlanV0`, al `WorkflowTaskStore`, al
`WorkflowTaskWaitStateV0`, a `RequiredTestEvidenceV0`, al outbox ni a
`OperationalDirectorClosureV0`. Solo guarda refs y resumen compacto para que una
reentrada no dependa del plan inicial, de stats agregadas ni de agentes vivos
globales.

Invariantes iniciales del corte:

- scope por `wave_ref`, `cohort_ref` o `parent_task_ref`, nunca por todo el run;
- transiciones causales e idempotentes con refs de task, delivery, review,
  evidencia y outbox;
- blockers explicitos cuando falte entrega, review, test requerido, outbox cero,
  contexto o cierre de tasks;
- frontera neutral: sin Codex, OPES, DB concreta, rutas locales, proveedor ni
  runtime real;
- cierre solo si el scope esta `quiescent`, outbox cero, sin tasks abiertas y
  con reviews/evidencias/validaciones requeridas.

Implementado: DTO, normalizacion, validacion, puertos, store en memoria,
persistencia file-based, wiring del stack/servidor y escritura inicial desde
`ContinueAppDirectorV0` al materializar `launch_subagents`. Ese state guarda el
step activo `wait_subagents`, ola/cohorte, tasks, agentes objetivo, agentes
pendientes, `parent_task_ref` si aplica y `wait_ref`. `ContinueAppDirectorV0`
tambien puede leerlo por `operational_director_plan_ref` para reentrar al wait
activo, avanzar a `review_deliveries` cuando el wait queda consumido y mover
`review_deliveries` a `run_required_tests` o `replan_or_close` cuando la cadena
causal de review aceptada pertenece al scope activo.

Pendiente verificable para este corte: materializacion durable de tests, replan
y cierre, y pruebas offline de replay sin duplicar efectos.

## Criterios de cierre operativo

Un cierre generico solo puede emitirse si todos estos criterios estan cubiertos
por estado durable o por una fuente inyectada y verificable:

- `quiescent`: la ola/cohorte activa no tiene waits ni entregas esperadas
  pendientes dentro de su scope;
- outbox cero: no quedan comandos pendientes ni efectos causales sin ACK para
  la ola/cohorte que se pretende cerrar;
- review aceptada: las entregas esperadas tienen review aceptada o, si no,
  existe rework/replan causal en vez de cierre;
- tests requeridos durables: en modo `programming`, los tests declarados quedan
  registrados como evidencia durable antes de cerrar;
- sin tasks abiertas: no se cierra una task, ola o run mientras existan
  `WorkflowTaskV0` abiertas dentro del scope de cierre.

## Pendiente verificable

Estas piezas siguen pendientes o parciales hasta que existan implementacion y
pruebas claras. No deben describirse como hechas antes de cerrar la evidencia.

1. Runner/adaptador de tests:
   en modo `programming`, generar `RequiredTestEvidenceV0` causal por
   comando/evento/artefacto de resultado. El consumo durable ya existe; un
   summary textual no basta.
2. `replan_or_close` causal:
   cerrar solo con review aceptada, evidencias y validaciones requeridas. Si
   falta algo, emitir rework/replan con causa, intento y refs de task, delivery
   y review.
3. Estado vivo del plan:
   el tramo inicial ya persiste step activo, ola/cohorte, tasks, agentes y
   `wait_ref`; avanza a `review_deliveries` cuando el wait queda consumido y a
   `run_required_tests`/`replan_or_close` cuando la review aceptada es causal.
  Tests durables ya registran evidencia aceptada o blocker de fallo. La
  observacion negativa de review con `ReworkRequested`/`ReplanDecisionRecorded`
  tiene prueba focal y persistencia de refs. El cierre exitoso marca el
  `PlanState` como `closed`, y los issues/source insuficiente lo bloquean con
  `closure_reason`; faltan replan automatico y replay/idempotencia completa.
4. Reentrada offline:
   wait y review inicial ya reconstruyen scope desde plan state; las
   transiciones posteriores deben reconstruirse desde run, task store, wait
   state, eventos y plan state, no desde agentes vivos globales ni stats.
5. `OperationalClosureSource` por composicion:
   el stack Codex ya tiene fuente para Director Operativo. Otras composiciones
   deben aportar su fuente por puerto. Si la fuente no existe o devuelve estado
   insuficiente, el cierre debe bloquearse o replanificarse con causa.

## Criterio de done del corte

El corte se considera cerrado solo cuando haya tests offline deterministas que
demuestren:

- una ola no cierra hasta que todas sus entregas esperadas tengan review
  aceptada o rework causal;
- entregas/ACKs de agentes fuera de `WaitAgentRefs` o de otra ola no afectan el
  cierre;
- no hay cierre si el scope no esta `quiescent`, si el outbox no esta a cero o
  si quedan tasks abiertas;
- un run `programming` no cierra sin evidencias durables de tests requeridos;
- refs cruzadas de task/delivery/review aceptada no cierran;
- un run `domain_work` cierra por artefactos y validadores de dominio, no por
  tests de programacion inventados;
- una entrega invalida produce `RequestRework` o `RecordReplanDecision` con refs
  causales;
- `CloseTask`/cierre de ola/run solo aparece despues de review, evidencias y
  validaciones;
- el estado vivo del plan conserva transiciones posteriores al wait inicial.

Nombres sugeridos de pruebas, si no existen aun:

```sh
TestOperationalDirectorGenericClosureV0NoCierraSinReviewDeOla
TestOperationalDirectorGenericClosureV0IgnoraEntregaFueraDeOla
TestOperationalDirectorGenericClosureV0ExigeQuiescentOutboxCeroYSinTasksAbiertas
TestOperationalDirectorGenericClosureV0ExigeTestsDurablesEnProgramming
TestOperationalDirectorGenericClosureV0CierraDomainWorkConArtefactoValidado
TestOperationalDirectorGenericClosureV0ReplanConRefsCausales
TestOperationalDirectorPlanStateV0SobreviveReentradaYActualizaTransiciones
```

## No hacer

- No reabrir P1 `WaitAgentRefs` salvo regresion demostrada.
- No esperar todos los agentes vivos del run.
- No cerrar por ACK, summary textual o "no queda pending" sin review causal.
- No convertir un smoke Codex en prueba de cierre generico.
- No tocar OPES productivo ni drenar colas amplias para este corte.
- No cablear DB real ni `orquesta-domain-work-sql` al Director.
- No anunciar recursion Codex completa hasta tener parent/child refs, limites,
  presupuesto y review causal con evidencia real.

## Orden recomendado

1. Reusar el plan state inicial y su lectura/reentrada ya disponibles.
2. Reusar ese ciclo para desglosar los casos ya documentados:
   `DIRECTOR-WAVE-REVIEW-OFFLINE`,
   `DIRECTOR-REQUIRED-TESTS-DURABLE-OFFLINE`,
   `DIRECTOR-REPLAN-CLOSE-OFFLINE` y
   `DIRECTOR-PLAN-STATE-OFFLINE`.
3. Solo despues, preparar smoke Codex real de ola/cohorte.
4. Solo despues, conectar OPES/conectores reales con guardas opt-in y filtro por
   job type.
