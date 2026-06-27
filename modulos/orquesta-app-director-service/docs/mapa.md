# Mapa: orquesta-app-director-service

Este modulo es servicio de aplicacion. Recibe solicitudes de arranque o
continuacion del Director, compone puertos inyectados y llama al loop del nucleo.
No decide runtime, proveedor, HOME, DB, REST, MCP, web, Codex ni OPES.

## Entradas

- `StartAppDirectorV0`: valida `AppSpecRequestV0`, prepara intake, persiste el
  run inicial y ejecuta el loop con los puertos aportados por la composicion.
- `ObserveAppDirectorGoalV0`: observa el goal externo de un arranque
  `goal-first`, persiste resultado/closure y refleja la salida terminal en la
  run del core como cierre aceptado o bloqueo causal por puertos inyectados.
- `ContinueAppDirectorV0`: reentra sobre un run existente, materializa un plan
  operativo si llega, recupera `OperationalDirectorPlanStateV0`, resuelve waits
  acotados y avanza review/tests/replan/cierre cuando hay causalidad suficiente.
  Si el run tiene `GoalWorkStateV0` persistido, no entra al loop legacy y
  redirige a observacion goal-first. Si solo queda `GoalWorkRunMarkerV0`,
  bloquea como state faltante y tampoco reactiva el director historico.

## Ciclo Conceptual

```text
Start/Continue request
  -> normalizacion y validacion local
  -> intake o carga del run existente
  -> decisiones del director / plan operativo materializado
  -> WaitAgentRefs por refs explicitas, ola, cohorte o parent task
  -> loop progresivo por puertos
  -> PlanState post-loop
  -> review_deliveries
  -> run_required_tests
  -> replan_or_close
  -> cierre operativo por fuente inyectada
```

El servicio solo traduce entre contratos de aplicacion y puertos neutrales. Los
efectos externos cruzan por outbox, dispatchers, stores, sources y runners
inyectados.

## Mapa De Ficheros

- `types_v0.go`: DTOs publicos y puertos del servicio.
- `start_v0.go`, `start_operational_director_plan_v0.go`: arranque normal y
  arranque con plan operativo inicial.
- `continue_v0.go`, `continue_existing_loop_v0.go`, `managed_loop_v0.go`:
  reentrada y ejecucion acotada del loop progresivo.
- `wait_refs_v0.go`, `continue_wait_state_persist_v0.go`,
  `operational_director_wait_state_*`: resolucion y persistencia de waits por
  refs concretas.
- `operational_director_decision_*`: aplicacion de decisiones del director y
  siembra/merge de `OperationalDirectorPlanStateV0`.
- `operational_director_plan_state_*`,
  `continue_plan_state_post_loop_v0.go`: avance vivo del plan despues del loop.
- `operational_director_review_*`: lectura de eventos de review, matching por
  scope y observacion de review aceptada o negativa.
- `operational_director_required_tests_*`: runner/evidencia durable de tests
  requeridos, quality gates y reapertura por replan causal.
- `operational_closure_*`: prerequisitos, request de cierre, emision de replan
  por issues y cierre del plan state.
- `provider_composition_v0.go`: composicion de candidate providers desde
  sources inyectadas.
- `closure_policy_v0.go`, `validation_v0.go`, `normalize_v0.go`: guardas
  locales y normalizacion de requests.

## Reglas De Ampliacion

- Mantener cada incremento en el grupo de ficheros que corresponde al paso del
  ciclo; si cruza grupos, preferir helper pequeno con prueba focal.
- Conservar `WaitAgentRefs` como scope efectivo. Cohorte, ola y parent task se
  resuelven aqui usando `DirectorTaskStore`; los adaptadores solo reciben refs
  de agente.
- No cerrar por resumen textual. Cierre operativo requiere entrega, review
  aceptada, evidencias durables de tests cuando apliquen y fuente de cierre
  inyectada.
- Bloquear por prerequisito ausente de composicion; no crear runtime, storage o
  proveedor por defecto dentro del servicio.
- Registrar tareas locales nuevas en `docs/tareas.md` solo si no estan cubiertas
  por la foto vigente o por un Txx global.
