# Corte Director funcionando esta tarde 2026-05-17

Este documento es el handoff operativo para futuros agentes. El objetivo es
dejar claro que parte de Orquesta puede usarse hoy y que falta para que el
Director Operativo gobierne agentes de forma completa.

El handoff siguiente, para el corte de cierre causal offline generico, esta en
`corte_cierre_generico_director_operativo_2026-05-17.md`. Este documento conserva
la ruta usable de tarde; el foco nuevo ya no es scope de espera.

## Regla de trabajo

No mezclar el nucleo con OPES, Codex, DB, HTTP ni filesystem. El Director debe
gobernar por contratos, refs opacas y puertos. Codex es el runtime productivo
actual, pero no debe definir el nucleo.

## Lo que funciona hoy

- `StartAppDirectorV0` y `ContinueAppDirectorV0` existen en
  `modulos/orquesta-app-director-service`.
- El stack Codex real inyecta `StartAppDirectorPortsV0` y expone
  `POST /api/v0/apps/director`.
- El loop progresivo acepta `WaitAgentRefs`.
- `app-director-service` puede derivar `WaitAgentRefs` desde
  `wait_cohort_ref`, `wait_wave_ref` o `wait_parent_task_ref` usando
  `WorkflowTaskStore`.
- `orquesta-director-operativo` modela un plan neutral con:
  `launch_subagents`, `wait_subagents`, `review_deliveries`,
  `run_required_tests`, `replan_or_close` y `govern_delegation`.
- `OperationalDirectorPlanMaterializerV0` convierte planes `ready` en
  `WorkflowTaskV0` y `CreateMicrotask`, pero solo para items
  `launch_subagents`.
- `ContinueAppDirectorV0` puede recibir un `OperationalDirectorPlanV0` listo,
  materializar su primera ola antes del loop, derivar `WaitAgentRefs` de esa
  ola y reentrar sin esperar agentes ajenos al scope.
- `WorkflowTaskWaitStateV0` registra la espera por `cohort_ref`, `wave_ref` o
  `parent_task_ref` con causa, tasks, agentes objetivo y pendientes. El store
  file-based de `orquesta-state-file` lo persiste.
- `OperationalDirectorPlanStateV0` registra el estado inicial del plan:
  `launch_subagents` aceptado, `wait_subagents` activo, ola/cohorte, tasks,
  agentes, pendientes y `wait_ref`. El store file-based tambien lo persiste.
- `ContinueAppDirectorV0` puede reentrar con `operational_director_plan_ref`,
  leer ese state y recuperar el wait activo sin mirar agentes vivos globales.
- `DrainRunV0` ya cierra P1 WaitAgentRefs para Codex: cuando
  `WaitAgentRefs` no esta vacio, pending, wait e ingesta de ACK/deliveries se
  limitan al mismo scope. Cuando esta vacio, el comportamiento legacy observa
  todo el run.
- `ReviewReworkReplanCandidateProviderV0` puede crear microtareas de rework
  tipo `split_task` si `app-director-service` le inyecta `DirectorTaskStore`.
- El stack tiene smokes y tests para director REST, stats, review/rework,
  waits por task metadata y reentrada por `ContinueAppDirectorV0`.

## Lo que no funciona todavia

- El materializador sigue siendo deliberadamente estrecho: solo convierte
  `launch_subagents` listos en `WorkflowTaskV0` y `CreateMicrotask`. Las
  transiciones posteriores ya no pertenecen a ese materializador sino al
  `PlanState`/`ContinueAppDirectorV0`.
- El ciclo offline posterior a `wait_subagents` ya cubre review causal por
  scope, `run_required_tests` con `RequiredTestEvidenceV0`, runner por puerto,
  puerta `replan_or_close`, cierre causal, cierre por reentradas de ola
  multitarea y varios blockers con reentrada. Lo que falta no es P1 ni el
  tramo feliz offline, sino smoke Codex real con runner, replan generico de
  blockers restantes y replay/idempotencia completa del replan.
- `ContinueAppDirectorV0` depende aun del caller para limites como
  `MaxExternalWaits`; no asumir que mantiene vivo un wait largo si se llama con
  defaults vacios.
- El scope de ingesta de ACK/deliveries ya esta cerrado para Codex, pero eso no
  cierra una ola funcional: falta review agrupada por ola, pruebas requeridas y
  cierre/replan durable.
- `orquesta-app-runner` no participa todavia en `wait_cohort_ref`,
  `wait_wave_ref` ni `wait_parent_task_ref`. Tratarlo como flujo app-plan
  separado hasta cablear contrato de Director.
- La recursion Codex real con hijos/nietos, presupuesto global, fanout,
  profundidad y review causal sigue pendiente. Ya hay una guarda offline en el
  stack Codex: el cierre operativo no genera cierre para una task padre si sus
  `child_task_refs` faltan en el store o siguen abiertos; esto no completa la
  recursion real, pero evita cerrar un subarbol antes de sus hijos.
- No hay runtime real distinto de Codex probado end-to-end.

## Ruta para tener Orquesta usable hoy

Usar el camino ya integrado:

1. Arrancar por `/api/v0/apps/director` o por el servicio
   `StartAppDirectorV0`.
2. Dejar que el stack Codex lance tareas normales.
3. Reentrar por `ContinueAppDirectorV0` o por el drain del stack para consumir
   ACKs, entregas, review gates, rework y decisiones tardias.
4. Para esperar solo una cohorte/ola, pasar `wait_cohort_ref`, `wait_wave_ref` o
   `wait_parent_task_ref`. No esperar todos los agentes vivos del run.
5. Consultar `/api/v0/director/stats` para ver proceso, progreso y uso.

Comandos offline minimos:

```sh
go test -count=1 ./modulos/orquesta-director-operativo
go test -count=1 ./modulos/orquesta-orchestration-core -run 'Test.*OperationalDirector|Test.*WaitAgentRefs|Test.*ReviewRework|Test.*Replan'
go test -count=1 ./modulos/orquesta-app-director-service -run 'Test.*WaitAgentRefs|Test.*ContinueAppDirector|Test.*ConsumesDecisions|Test.*Closure'
go test -count=1 ./modulos/orquesta-app-codex-stack -run 'Test.*Review.*Rework|Test.*DrainRun|Test.*WaitAgentRefs|Test.*DirectorStats|Test.*ExternalWork'
```

Comando amplio antes de entregar:

```sh
go test -count=1 ./...
```

Smoke real existente, opt-in por operador:

```sh
./scripts/smoke_orquesta_server_rest_director.sh
```

Tratarlo como smoke real: puede lanzar Codex si el entorno lo permite y no debe
entrar en CI rapida sin decision explicita.

## P0 cerrado

Primer tramo durable del Director Operativo:

1. Tomar un plan `OperationalDirectorPlanV0` ready.
2. Materializar `launch_subagents` como hoy.
3. Guardar en `WorkflowTaskStore` las tasks con `cohort_ref`, `wave_ref`,
   `parent_task_ref`, `delegation_depth`, `max_child_agents` y `child_task_refs`.
4. Derivar los `WaitAgentRefs` de esa ola con `WorkflowTaskWaitAgentRefsV0`.
5. Reentrar al loop con esos refs, no con todos los agentes del run.
6. Registrar `WorkflowTaskWaitStateV0` con causa visible, refs de tasks, refs de
   agentes objetivo y pendientes.

No crear todavia un runtime nuevo ni meter SQL/OPES en el nucleo para esto.

La pieza de review/replan ya tiene el camino de escritura necesario para
`split_task`, pero sigue faltando integrarla como estado durable del ciclo
operativo: entrega observada -> review -> rework/replan -> nueva ola -> cierre.

## P1 WaitAgentRefs cerrado

Scope de espera e ingesta para Codex:

1. `DrainRunRequestV0.WaitAgentRefs` se normaliza y viaja al wait externo.
2. `drainAvailableObservationsV0` lo pasa a
   `AgentDeliveryObservationRequestV0`.
3. El core lo propaga desde `ProgressiveLoopRequestV0` hasta
   `DeliveryCandidateProviderV0`; el source Codex acota descriptors antes de
   leer/verificar ACKs.
4. `continueDrainRunControlAfterExternalV0` envuelve el `DeliverySource` con
   `drainDeliverySourceForWaitAgentRefsV0`.
5. `drainObservationsForWaitAgentRefsV0` y
   `applyDrainObservationsV0` descartan ACK/delivery cuyo `agent_ref` no
   pertenezca al scope.
6. Las rutas de `domain_work` filtran antes de submit/recovery para no producir
   efectos de agentes fuera del scope.
7. `WaitAgentRefs` vacio conserva compatibilidad legacy de run completo.

Evidencia focal:

```sh
go test -count=1 ./modulos/orquesta-orchestration-core ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-app-codex-stack -run 'TestDeliveryCandidateProviderV0PropagaYFiltraWaitAgentRefs|TestRunProgressiveLoopV0PropagaWaitAgentRefsAObservationSource|TestCodexDeliveryObservationSourceV0WaitAgentRefs|TestDrainRunV0WaitAgentRefsNoIngiereACKFueraDeScope|TestDrainRunHasPendingExternalAgentsV0FiltraCohorteObjetivo|TestDomainWork.*WaitAgentRefs'
```

## Siguiente cambio: cierre generico causal offline

Continuar cierre generico causal offline sin reabrir P1:

- `review_deliveries`: ya avanza por cadena causal del scope y no por entregas
  ajenas.
- `run_required_tests`: ya consume evidencia durable, ejecuta runner por puerto
  cuando esta inyectado y bloquea si falta evidencia causal sin runner efectivo.
- `replan_or_close`/`close`: ya gobierna cierre con review aceptada, evidencias,
  outbox cero y source/task store disponibles; tambien bloquea/reintenta por
  prerequisitos y por cierre insuficiente causal en casos cubiertos.
- estado del plan: ya persiste refs de review, tests, blockers y cierre/bloqueo
  en los tramos cubiertos. Falta completar replan generico de blockers no
  cubiertos y replay/idempotencia completa de ese replan.

Este corte debe apoyarse en los comandos ya existentes de core-workflow:
`RecordReviewResult`, `AcceptReview`, `RequestRework`, `RecordReplanDecision` y
`CloseTask`.

Hasta que exista codigo y test claro, cada blocker restante queda como pendiente
verificable. La matriz vigente separa los tramos cerrados offline de los smokes
reales pendientes.

## Siguiente cambio P2

Recursion gobernada:

- aceptar propuestas de subagentes solo como decision estructurada;
- comprobar parent/child refs, profundidad, fanout y presupuesto;
- materializar hijos por workflow/outbox;
- esperar por `parent_task_ref` o nueva cohorte hija;
- no cerrar subarbol sin review causal. Parcial cerrado en stack Codex: una
  task padre con `child_task_refs` solo puede generar request de cierre cuando
  esos hijos existen en `WorkflowTaskStore` y constan en `run.closed_tasks`.

## No hacer

- No anunciar "Director recursivo completo" sin smoke real.
- No esperar "todos los agentes vivos".
- No consumir decision de agente hijo sin ACK/artefacto causal.
- No presentar una cohorte Codex como ciclo funcional completo solo porque la
  ingesta ya este acotada por `WaitAgentRefs`; aun faltan review por ola, tests
  requeridos, cierre/replan y actualizaciones posteriores del plan state.
- No cablear `orquesta-domain-work-sql` ni DB real al Director.
- No mover conocimiento de OPES dentro de `orquesta-director-operativo`.
