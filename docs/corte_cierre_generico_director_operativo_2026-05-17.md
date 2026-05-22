# Corte cierre generico del Director Operativo - 2026-05-17

Este documento es el handoff del tramo siguiente al cierre de P1
`WaitAgentRefs`. El objetivo ya no es acotar la espera: eso queda cerrado para
el stack Codex. El objetivo ahora es cerrar un ciclo causal offline y generico
del Director Operativo, sin Codex real, sin OPES real y sin DB/producto nuevo.

El corte ya tiene un tramo offline integrado en core + `app-director-service` +
`orquesta-app-codex-stack`, un smoke Codex real acotado
`CODEX-REQTEST-REAL-E2E` y un smoke no-OPES temporal `EXT-NO-OPES` con
`codex-fake`, submitter real opt-in, review, required-tests y cierre operativo.
Si una pieza de codigo no esta integrada o no tiene prueba clara, se documenta
como pendiente verificable, no como hecho.

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
para tests requeridos. El contrato no ejecuta comandos ni sustituye a
`review_deliveries`; solo dice que un test declarado ya tiene resultado
persistido y enlazado a la cadena causal correcta. La ejecucion entra por runner
inyectado por puerto/composicion opt-in.

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
Desde el corte del 2026-05-22, `RequiredTestRunnerV0` lee primero la evidencia
deterministica si el writer tambien expone reader, o si se inyecta
`EvidenceReader`: un replay con la misma cadena causal no reejecuta el comando
externo ni reescribe una evidencia distinta. El runtime local tiene prueba
focal con comando real que marca una sola invocacion externa.

`orquesta-state-file` ya persiste estas evidencias y
`orquesta-app-codex-stack` puede encontrarlas desde `RequiredTestEvidenceRefs`
del request o, por compatibilidad, desde los `EvidenceRefs` del review result,
ignorando refs mixtas que no sean evidencias de test. La fuente del stack solo
construye cierre cuando existe cadena causal completa: delivery -> review
requested -> review result accepted -> accepted review. El `PlanState` ya
consume `RequiredTestEvidenceV0` en `run_required_tests`: `passed` causal avanza
a `replan_or_close`, `failed` bloquea con `required-tests-failed` y las refs
aceptadas se pasan al cierre. Si falta evidencia causal y no hay runner
efectivo, el plan queda bloqueado con `required-tests-evidence-missing` y se
registra un `QualityGateRecorded(blocked)` idempotente para auditoria, pero no
se crea `ReplanDecisionRecorded` automatico: puede ser latencia o falta de
ingesta, no necesariamente trabajo defectuoso. El replay con `state-file` no
duplica ese gate y reentra a `replan_or_close` cuando aparece evidencia `passed`
causal posterior. El runner por puerto, el ejecutor local opt-in, el smoke Codex
real acotado `CODEX-REQTEST-REAL-E2E` y el smoke no-OPES temporal
`EXT-NO-OPES` ya tienen evidencia. Siguen pendientes ola/cohorte Codex real
amplia, recursion real y OPES temporal real de derivados/cierre.

Desde el corte del 2026-05-22, `ContinueAppDirectorV0` no bloquea el
`PlanState` si el cierre de una task devuelve solo `run.open_tasks`: conserva el
run actualizado, deja `replan_or_close` activo y permite reentrar para cerrar la
siguiente task abierta. La prueba focal cubre dos tasks hijas con
`parent_task_ref`, reviews aceptadas, evidencias de tests y cierre final solo
cuando ya no quedan tasks abiertas.

Tambien desde el corte del 2026-05-22, si una ola multitarea esta en
`replan_or_close/running` y el cierre de una task concreta falla por
`closure_ref` o `validation_ref`, el servicio acepta replan causal siempre que
`closureRequest.TaskID` pertenezca al scope activo y exista cadena
task/delivery/review aceptada. Se emite una unica pareja
`QualityGateRecorded(blocked)` + `ReplanDecisionRecorded(retry_task)` para esa
task y la reentrada posterior solo espera el followup reflejado. Si la decision
causal ya existe pero el followup aun no esta reflejado en el run, la reentrada
explicita conserva el bloqueo estable sin devolver `active_step` ni duplicar
intentos; cuando el followup aparece, reabre la espera acotada. Si aparecen
varias decisiones candidatas en el mismo scope, no elige una al azar. El emisor
de replan de cierre tambien detecta la pareja deterministica ya reflejada en el
run y no vuelve a emitir comandos aunque el replay no tenga `CommandEffects`
frescos.

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
`replan_or_close` completo ni a replan generico de todos los blockers.

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
  para cerrar. El corte del 2026-05-21 ya permite salir del fallo cuando existe
  `QualityGateRecorded(blocked)` + `ReplanDecisionRecorded` causal y followups
  materializados, tanto en el mismo avance como en una reentrada posterior desde
  `required-tests-failed`. El corte del 2026-05-21 ya emite automaticamente esa
  cadena para un unico task causal y deja capacidad/agente en manos del
  scheduler/outbox; ver `docs/corte_required_tests_failed_replan_2026-05-21.md`.
  El corte posterior del 2026-05-22 consume tambien replan causal parcial en
  scopes multitarea cuando ya existe decision completa por task fallida. El
  corte del 2026-05-22 bloquea tambien `required-tests-evidence-missing` cuando
  no hay evidencia causal ni runner efectivo, registra un quality gate
  bloqueante idempotente sin replan automatico, y reentra a `replan_or_close` si
  la evidencia `passed` aparece despues. El smoke real acotado con runner esta
  cerrado; siguen pendientes smokes reales de ola/cohorte amplia y recursion
  real.
- [x] Replan negativo: cerrada la observacion durable de review negativa. El
  `PlanState` guarda refs/attempt para `ReworkRequested` y
  `ReplanDecisionRecorded`. El corte del 2026-05-21 ya convierte followups
  materializados en una nueva espera acotada: `split_task` cuando las nuevas
  `WorkflowTaskV0` existen en `WorkflowTaskStore` y estan reflejadas en
  `run.Tasks`, y `retry_task`/`replace_agent` cuando los agentes followup ya
  estan reflejados en `run.Agents`. El corte del 2026-05-22 cubre tambien la
  reentrada tardia: si `review_deliveries` ya estaba en `changes_requested` y
  los followups aparecen despues, `ContinueAppDirectorV0` reabre
  `wait_subagents` solo con esos refs sin duplicar `replan_attempts`; ver
  `docs/corte_replan_negativo_followups_split_2026-05-21.md`. Para tests
  fallidos ya se consume replan materializado por quality gate bloqueante. El
  cierre bloqueado por `required_test_evidence_refs` tambien reentra si despues
  aparece el followup causal reflejado por quality gate/replan. El cierre
  insuficiente causal (`closure_ref`/`validation_ref` con task, delivery y review
  aceptada) ya emite quality gate + replan retry y reabre solo el followup
  reflejado, tambien cuando el `replan_or_close` cubre una ola multitarea pero
  el fallo identifica una unica task causal. La emision de esa pareja es
  idempotente si el run ya refleja los refs deterministas de gate/replan. Un
  blocker nuevo sin decision causal debe entrar como caso nuevo con prueba
  propia, no como pendiente abierto de este corte.
- [x] `replan_or_close` como puerta de cierre: el cierre ya no se dispara por
  cualquier `PlanState` activo. Si hay estado vivo, solo se evalua cierre cuando
  el step activo es `replan_or_close` en `running`; un step anterior o
  `replan_or_close` pendiente no invoca `OperationalClosureSource`. Si en ese
  punto el loop esta `quiescent` pero aun hay outbox pendiente, tampoco invoca
  `OperationalClosureSource` y bloquea el `PlanState` con
  `operational-closure-outbox-pending`. Si faltan `OperationalClosureSource` o
  `DirectorTaskStore`, el `PlanState` se bloquea con causa durable. La reentrada
  ya reabre `replan_or_close` cuando el source aparece, el task store aparece o
  el outbox vuelve a estar drenado; si el prerequisito sigue faltando, el cierre
  se vuelve a bloquear sin consultar trabajo externo indebido.
- [x] Plan state vivo inicial: contrato, stores y persistencia del tramo
  `launch_subagents -> wait_subagents`, con ola/cohorte activa, step activo,
  task refs, agent refs, pending agent refs y `wait_ref`.
- [x] Reentrada inicial desde plan state: `ContinueAppDirectorV0` acepta
  `operational_director_plan_ref`, lee el state y recupera scope de
  `wait_subagents` sin mirar agentes vivos globales.
- [x] Avance inicial tras wait consumido: si el loop queda `quiescent` y los
  agentes pendientes entregaron, el state pasa de `wait_subagents` a
  `review_deliveries`.
- [x] Plan state vivo restante: `run_required_tests` durable, blocker de test
  fallido, blocker de evidencia de test faltante, observacion de review
  negativa, bloqueo por outbox pendiente antes de cierre, puerta explicita de
  `replan_or_close` y razon de cierre/bloqueo ya quedan persistidos. La
  reentrada de un cierre bloqueado por
  `required_test_evidence_refs` hacia `wait_subagents` con followup tardio causal
  tambien queda cubierta. Existe prueba integrada offline de
  `ContinueAppDirectorV0` para el camino
  `review_deliveries -> run_required_tests` con runner por puerto ->
  `replan_or_close -> close`. El replay con `state-file` recuperado ya cubre ese
  cierre sin duplicar `TaskClosed`, validacion, `RunClosed` ni evidencia. El
  replay directo por `ContinueAppDirectorV0` sobre un `PlanState` ya cerrado
  queda como no-op idempotente: no reentra por `active_step`, no ejecuta runner,
  no llama fuente de cierre y no duplica eventos.
- [x] Evento/comando idempotente del ciclo probado: el replay de cierre exitoso y bloqueo de
  cierre ya tiene prueba focal y no duplica refs del `PlanState` ni `RunClosed`.
  `state-file` cubre tambien el camino integrado review -> runner ->
  `replan_or_close` -> close con replay de cierre y replay directo por
  `ContinueAppDirectorV0` con plan cerrado. El replan automatico por cierre
  insuficiente ya corta la reemision cuando el run refleja los refs
  deterministas de `QualityGateRecorded` y `ReplanDecisionRecorded`, aun sin
  `CommandEffects` frescos.

El orden recomendado ahora es: smoke Codex real de ola/cohorte amplia, recursion
real y OPES temporal real de derivados/cierre. El smoke Codex real acotado con
runner ya quedo cerrado por `CODEX-REQTEST-REAL-E2E`; el no-OPES temporal con
runtime fake ya quedo cerrado por `EXT-NO-OPES`. Ninguno cubre recursion real,
ola Codex amplia ni derivados OPES reales.

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
- blockers explicitos cuando falte entrega, review, test requerido, contexto,
  outbox cero o cierre de tasks; el outbox pendiente en
  `replan_or_close/running` bloquea antes de consultar la fuente de cierre;
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

Pendiente verificable despues de este corte: smoke Codex real de ola/cohorte
amplia, recursion real y OPES temporal real de derivados/cierre. El runner real
acotado con un agente Codex vivo ya tiene evidencia en
`CODEX-REQTEST-REAL-E2E`; el cierre no-OPES temporal esta cubierto por
`EXT-NO-OPES`.

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

Estas piezas ya tienen cierre offline/fake-runtime donde se indica. Lo que sigue
pendiente es llevar la misma garantia a Codex real amplio/recursion y a OPES
temporal real de derivados/cierre.

1. Cobertura Codex real con runner:
   el runner por puerto, el executor local opt-in, el consumo de
   `RequiredTestEvidenceV0`, el replay sin reejecucion externa, el smoke
   `state-file` sin Codex vivo y el smoke Codex real acotado
   `CODEX-REQTEST-REAL-E2E` ya tienen evidencia. Sigue pendiente llevar esa
   cobertura a ola/cohorte amplia y recursion real con varios agentes vivos,
   parent/child refs, review causal y cierre del arbol.
2. `replan_or_close` causal:
   ya actua como puerta explicita para cierre cuando el `PlanState` esta activo:
   solo `replan_or_close` en `running` permite invocar la fuente de cierre, y la
   ausencia de source o task store bloquea con causa durable. Si el loop queda
   `quiescent` con `PendingOutboxCount > 0`, bloquea
   `operational-closure-outbox-pending` y no llama a
   `OperationalClosureSource`. Si el cierre queda bloqueado por
  `required_test_evidence_refs` y ya hay quality gate/replan con followup
  reflejado, `ContinueAppDirectorV0` reabre la espera acotada. Si el cierre
  insuficiente conserva refs causales de task, delivery y review aceptada, ya
  emite replan retry y reabre el followup concreto; en scope multitarea exige
  que una unica task causal del scope sea la afectada.
3. Estado vivo del plan:
   el tramo inicial ya persiste step activo, ola/cohorte, tasks, agentes y
   `wait_ref`; avanza a `review_deliveries` cuando el wait queda consumido y a
   `run_required_tests`/`replan_or_close` cuando la review aceptada es causal.
  Tests durables ya registran evidencia aceptada o blocker de fallo. La
  observacion negativa de review con `ReworkRequested`/`ReplanDecisionRecorded`
  tiene prueba focal y persistencia de refs. El cierre exitoso marca el
  `PlanState` como `closed`; los issues, source insuficiente, source ausente,
  task store ausente u outbox pendiente lo bloquean con `closure_reason` y los
  prerequisitos de infraestructura reabren `replan_or_close` cuando vuelven a
  estar disponibles. El camino integrado de
  `ContinueAppDirectorV0` ya cubre review aceptada, runner de tests requerido,
  `replan_or_close` y cierre causal en un unico ciclo offline. El replay de
  cierre ya no duplica refs/eventos de cierre; la reentrada de cierre bloqueado
  por falta de evidencia requerida y por cierre insuficiente causal no duplica
  `replan_attempts`.
4. Reentrada offline:
   wait, review, tests, cierre exitoso, cierre bloqueado por prerequisitos y
   varios replans causales ya reconstruyen scope desde run, task store, wait
   state, eventos y plan state, no desde agentes vivos globales ni stats.
5. `OperationalClosureSource` por composicion:
   el stack Codex ya tiene fuente para Director Operativo y `EXT-NO-OPES` cubre
   una composicion externa temporal por refs opacas. OPES sigue pendiente de
   fuente/validacion real acotada contra instancia temporal.

## Criterio de done de huecos restantes

El tramo offline ya tiene tests deterministas para review positiva por scope,
tests durables, puerta de cierre, cierre causal, replay focal, cierre insuficiente
causal con replan retry y varios blockers. `CODEX-REQTEST-REAL-E2E` demuestra
entrega de un agente Codex vivo, review, evidencia durable de test y cierre en
caso acotado. `EXT-NO-OPES` demuestra una app externa temporal con runtime fake
y cierre por refs opacas.

Los huecos restantes no se consideran cerrados hasta tener evidencia propia:

- smoke Codex real de ola/cohorte amplia con varios agentes vivos, mismo ciclo
  de review/tests/cierre y scope formal de Director Operativo;
- recursion Codex real con parent/child refs, limites, presupuesto, entregas
  vivas, review causal y cierre del arbol;
- OPES temporal real de derivados/cierre hasta `assemble_topic`, con refs
  causales suficientes y sin asumir cierre desde dry-run o automatizacion
  offline;
- cualquier blocker nuevo no cubierto debe traer prueba focal de replan y replay
  idempotente antes de declararse parte del tramo cerrado.

Tests existentes que respaldan el tramo cerrado estan en la matriz:
`DIRECTOR-GENERIC-CLOSURE-OFFLINE`,
`DIRECTOR-REQUIRED-TESTS-DURABLE-OFFLINE`,
`DIRECTOR-REPLAN-CLOSE-OFFLINE` y `DIRECTOR-PLAN-STATE-OFFLINE`.
Nombres sugeridos solo para huecos restantes, si no existen aun:

```sh
TestOperationalDirectorPlanStateV0ReplanGenericoBlockerConRefsCausales
TestOperationalDirectorPlanStateV0ReplayNoDuplicaReplanGenerico
TestCodexStackRealRecursiveDelegationOptInV0
TestCodexStackRealOperationalDirectorWaveCohortOptInV0
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
3. Preparar smoke Codex real de ola/cohorte amplia; el caso acotado de runner ya
   esta cerrado por `CODEX-REQTEST-REAL-E2E`.
4. Solo despues, conectar OPES/conectores reales con guardas opt-in y filtro por
   job type.
