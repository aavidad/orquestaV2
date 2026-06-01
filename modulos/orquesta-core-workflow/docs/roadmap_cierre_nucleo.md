# Roadmap de cierre del nucleo durable

## Estado

`orquesta-core-workflow` queda como motor durable principal. Ya cubre fases, comandos/eventos, replay, outbox, capacidad decidida, agentes, lifecycle logico, evaluacion de trabajo, parada, preguntas al director, respuestas/desbloqueo, gates de concurrencia/calidad, entregas, revision, validacion y cierre.

## Lo que no debe crecer aqui directamente

- Replanificacion compleja.
- Leases, heartbeats y timeouts operacionales.
- Scheduler de concurrencia por read/write-set.
- Seleccion real de proveedor, modelo, cuota, HOME u OAuth.
- Persistencia o dispatch productivo.

## Microproyectos de cierre

```text
Modulo: orquesta-core-replanner
Objetivo: contratos puros de replanificacion/rework para salir de bucles de trabajo basura.
Promocion esperada: comandos/eventos versionados en core-workflow solo cuando el contrato sea estable.
```

```text
Modulo: orquesta-core-leases
Objetivo: contratos puros de leases/heartbeats/timeouts para convertir cuelgues en senales durables.
Promocion esperada: eventos compactos tipo lease/timeout sin runtime real ni reloj interno no determinista.
```

```text
Modulo: orquesta-core-concurrency
Objetivo: contratos puros de dependencias y conflictos de read/write-set para paralelizar agentes sin pisarse.
Promocion esperada: gates antes de RequestAgent/CreateMicrotask o eventos de bloqueo de conflicto.
```

## Criterio de cierre v0

El nucleo v0 se considera cerrado cuando:

- una app minima puede recorrer planificacion, programacion, revision, rework opcional, validacion y cierre;
- un agente sin progreso genera timeout/consulta/replan durable;
- dos agentes paralelos se aceptan solo si sus write-set no entran en conflicto;
- todos los efectos externos siguen saliendo por outbox;
- no hay DB/proveedor/HOME/OAuth/modelo concreto en estado, eventos ni comandos del core.

## Huecos concretos a cerrar por promocion

`orquesta-core-replanner` debe estabilizar primero los DTOs puros y luego proponer el corte minimo `RecordReviewResult`, `RequestRework` y `RecordReplanDecision`. Esto cierra el caso `changes_requested/rejected` sin retry automatico ni relanzar agentes de forma implicita.

`orquesta-core-leases` estabilizo `AgentHeartbeatReportV0`, `EvaluateAgentLeaseV0` y el candidato `AgentLeaseExpiredV0` con `now` externo. La promocion minima `AgentLeaseExpired` ya esta en workflow; NCW-049 agrega `AgentStopConfirmed` para distinguir parada solicitada de parada efectiva.

`orquesta-core-concurrency` debe estabilizar normalizacion de scopes, deteccion de conflictos y plan de grupos paralelos. La promocion minima candidata es un gate antes de `RequestAgent` o un evento de conflicto local que no bloquee todo el run.

Estos cortes se aplican uno por uno. Ninguno autoriza a importar runtime, persistence, DB, filesystem, Git productivo, proveedor, modelo, HOME, OAuth, prompts ni transcripts al workflow.

## Corte externo ya disponible

```text
Fecha: 2026-05-06
Modulos:
  - orquesta-core-replanner: ReplanProposalV0.
  - orquesta-core-leases: AgentLeasePolicyV0, AgentHeartbeatReportV0.
  - orquesta-core-concurrency: WorksetClaimV0, ScopeRefV0.
Estado:
  - contratos puros implementados y tests locales verdes.
Uso:
  - no importar todavia al workflow hasta que existan traductores/evaluadores y promocion minima.
```

## Corte externo 2 disponible

```text
Fecha: 2026-05-06
Modulos:
  - orquesta-core-replanner: ReviewResult -> ReplanProposalV0.
  - orquesta-core-leases: EvaluateAgentLeaseV0 -> AgentTimeoutAssessmentV0.
  - orquesta-core-concurrency: DetectWorksetConflictsV0 -> WorksetConflictV0.
Estado:
  - tests locales y core-workflow verdes.
Uso:
  - todavia no hay eventos nuevos en workflow; usar como politica candidata hasta cerrar promocion.
```

## Corte externo 3 disponible

```text
Fecha: 2026-05-06
Modulos:
  - orquesta-core-replanner: AgentWorkAssessed -> ReplanProposalV0.
  - orquesta-core-leases: AgentTimeoutAssessmentV0 -> AgentLeaseExpiredV0 candidato.
  - orquesta-core-concurrency: EvaluateWorksetDependenciesV0.
Estado:
  - tests locales y core-workflow verdes.
Siguiente promocion:
  - NCW-042 `RecordReviewResult -> ReviewResultRecorded`, sin rework automatico.
  - Despues: `RequestRework` y `RecordReplanDecision`, uno por corte.
```

## Corte 4 disponible

```text
Fecha: 2026-05-06
Cambios:
  - NCW-042 implementado: `RecordReviewResult -> ReviewResultRecorded`.
  - `ReviewResults` proyecta ref/status/review/delivery de forma compacta.
  - `EvaluateParallelGroupsV0` disponible en orquesta-core-concurrency.
Validacion:
  - workflow, replanner, leases, concurrency, director, runtime, persistence, capacity, context y e2e verdes juntos.
Pendiente consciente:
  - `AcceptReview` aun no exigia `ReviewResultRecorded(accepted)` en este corte; resuelto despues en Corte 5.
  - `RequestRework`, `RecordReplanDecision`, `AgentLeaseExpired` y gate de concurrencia aun no estan promovidos al workflow.
```

## Corte 5 disponible

```text
Fecha: 2026-05-06
Cambios:
  - NCW-043 implementado: `AcceptReview` exige `ReviewResultRecorded(status=accepted)`.
  - `ReviewAccepted` tambien rechaza replay si falta el resultado aceptado previo para la misma revision y entrega.
Validacion:
  - go test -count=1 ./modulos/orquesta-core-workflow.
  - go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core-replanner ./modulos/orquesta-core-leases ./modulos/orquesta-core-concurrency ./modulos/orquesta-director ./modulos/orquesta-runtime ./modulos/orquesta-persistence ./modulos/orquesta-capacity ./modulos/orquesta-context ./modulos/orquesta-e2e.
Lectura:
  - la aceptacion queda auditada por resultado de revision durable;
  - el cierre de tarea sigue separado en `CloseTask`;
  - `changes_requested/rejected` siguen pendientes de promocion a rework/replan.
Pendiente consciente:
  - `RequestRework`, `RecordReplanDecision`, `AgentLeaseExpired` y gate de concurrencia aun no estan promovidos al workflow.
```

## Corte 6 disponible

```text
Fecha: 2026-05-06
Cambios:
  - NCW-044 implementado: `RequestRework -> ReworkRequested`.
  - `ReworkRequested` exige `ReviewResultRecorded(status=changes_requested|rejected)` previo para la misma revision y entrega.
  - `orquesta-e2e` cubre el flujo minimo `ReviewResultRecorded(changes_requested) -> ReworkRequested`.
  - `orquesta-core-leases` deja listo el candidato `AgentLeaseExpiredV0` para promocion posterior.
  - `orquesta-core-concurrency` endurece `EvaluateParallelGroupsV0` para bloquear claims invalidos.
Validacion:
  - go test -count=1 ./modulos/orquesta-e2e.
  - go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core-replanner ./modulos/orquesta-core-leases ./modulos/orquesta-core-concurrency ./modulos/orquesta-director ./modulos/orquesta-runtime ./modulos/orquesta-persistence ./modulos/orquesta-capacity ./modulos/orquesta-context ./modulos/orquesta-e2e.
Lectura:
  - el nucleo ya distingue revision aceptada de retrabajo solicitado;
  - rework durable no relanza agentes ni decide replan por si solo;
  - leases y concurrencia siguen como candidatos puros, no promovidos al workflow.
Pendiente consciente:
  - `RecordReplanDecision`, `AgentLeaseExpired` y gate de concurrencia aun no estan promovidos al workflow.
```

## Corte 7 disponible

```text
Fecha: 2026-05-06
Cambios:
  - `orquesta-core-replanner` implementa `ReplanDecisionV0` como DTO puro.
  - NCW-045 implementado: `RecordReplanDecision -> ReplanDecisionRecorded`.
  - `ReplanDecisionRecorded` exige `ReworkRequested` previo como `source_ref` en este corte.
  - `orquesta-e2e` cubre `ReviewResultRecorded(rejected) -> ReworkRequested -> ReplanDecisionRecorded`.
Validacion:
  - go test -count=1 ./modulos/orquesta-core-replanner.
  - go test -count=1 ./modulos/orquesta-e2e.
  - go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core-replanner ./modulos/orquesta-core-leases ./modulos/orquesta-core-concurrency ./modulos/orquesta-director ./modulos/orquesta-runtime ./modulos/orquesta-persistence ./modulos/orquesta-capacity ./modulos/orquesta-context ./modulos/orquesta-e2e.
Lectura:
  - el core ya registra una decision de replan trazable sin ejecutar followups;
  - crear tareas, pedir capacidad o relanzar agentes sigue separado;
  - leases y concurrencia siguen como candidatos puros pendientes de promocion.
Pendiente consciente:
  - `AgentLeaseExpired` y gate de concurrencia aun no estan promovidos al workflow.
  - followups de replan desde assessment parado ya quedan cubiertos en `orquestacionnucleoapp`; quedan otros origenes de replan por cerrar con pruebas equivalentes.
```

## Corte 8 disponible

```text
Fecha: 2026-05-06
Cambios:
  - NCW-046 implementado: `RegisterAgentLeaseExpired -> AgentLeaseExpired`.
  - `AgentLeaseExpired` registra expiracion observada sin outbox ni efectos operativos.
  - `orquesta-e2e` cubre agente solicitado -> `AgentLeaseExpired` sin parar ni fallar al agente.
  - `orquesta-core-concurrency` deja disponible `EvaluateConcurrencyGateV0` como contrato puro para promocion posterior.
Validacion:
  - go test -count=1 ./modulos/orquesta-core-workflow.
  - go test -count=1 ./modulos/orquesta-e2e.
Lectura:
  - el core ya puede recordar timeouts/leases observados sin contaminarse con reloj, runtime ni procesos;
  - la accion recomendada se ejecuta despues por comandos separados;
  - concurrencia queda como el siguiente corte fuerte del nucleo.
Pendiente consciente:
  - gate de concurrencia aun no esta promovido al workflow.
  - acciones posteriores a lease con `stop_agent` ya quedan materializadas en `orquestacionnucleoapp` por puerto externo de evaluacion; quedan `retry/mark/replan` como cortes explicitos o consulta al director.
  - followups de replan por assessment parado ya tienen prueba de aplicacion.
```

## Corte 9 disponible

```text
Fecha: 2026-05-06
Cambios:
  - NCW-047 implementado: `RecordConcurrencyGate -> ConcurrencyGateRecorded`.
  - `ConcurrencyGateRecorded` registra allow/block/ask_director sin scheduler ni outbox.
  - El workflow no importa `orquesta-core-concurrency`; consume una evaluacion compacta ya calculada.
Validacion:
  - go test -count=1 ./modulos/orquesta-core-workflow.
Lectura:
  - el core ya puede guardar una barrera durable de concurrencia;
  - el director debe decidir despues si pide agentes, pregunta o replanifica;
  - no se bloquea el run completo si hay conflictos locales.
Pendiente consciente:
  - falta E2E que use la politica pura de concurrency antes de pedir dos agentes.
  - falta hacer obligatorio el gate en flujos paralelos reales del director.
```

## Corte 10 disponible

```text
Fecha: 2026-05-06
Cambios:
  - NCW-049 implementado: `RegisterAgentStopConfirmed -> AgentStopConfirmed`.
  - NCW-050 implementado y actualizado por NCW-049: `RegisterDelivery` rechaza agentes con parada confirmada, no solo solicitada.
  - NCW-051 implementado: `RegisterDelivery` exige `AgentStarted` y rechaza `AgentFailed`.
  - NCW-052 implementado: `AgentStarted`, `AgentFailed` y `AgentStopRequested` no aceptan outcomes contradictorios.
  - NCW-053 implementado: `StopAgent` no se emite si `AgentFailed` ya quedo proyectado.
  - NCW-054 implementado: `RecordReplanDecision` acepta `AgentFailed` como fuente en programacion.
Validacion:
  - go test -count=1 ./modulos/orquesta-core-workflow.
  - go test -count=1 ./modulos/orquesta-mcp.
  - go test -count=1 ./modulos/orquesta-e2e -run TestE2EProgramacionTrabajoBasuraSolicitaParadaSinEntregaV0.
Lectura:
  - la frontera de entrega ya distingue agente pedido, arrancado, fallido, parado y parada confirmada;
  - el lifecycle de lanzamiento queda cerrado contra senales tardias contradictorias;
  - un launch fallido no genera parada runtime tardia;
  - un launch fallido puede abrir replan durable sin inventar `ReworkRequested`;
  - revision/cierre no reciben trabajo de un agente que no haya arrancado durablemente;
  - la sustitucion del agente sigue en replan/director, no en delivery ni en AgentFailed.
Pendiente consciente:
  - servidor MCP real y persistence productiva siguen fuera del workflow;
  - scheduler/cuotas/HOME deben entrar por conectores, no por estado de core.
```

## Corte 11 disponible

```text
Fecha: 2026-05-07
Cambios:
  - NCW-067 documentado: `RecordQualityGate -> QualityGateRecorded`.
  - `QualityGateRecorded` proyecta `quality_gates` como ref compacta `gate_ref + decision + subject_ref`.
  - El gate registra identidad fuerte en `CommandEffects` y valida decisiones, issue_refs, refs y detalles prohibidos.
Validacion:
  - 2026-05-07, ok, pruebas focalizadas de `quality_gate_v0_test.go` y validacion de `quality_gates`.
Lectura:
  - el core ya puede recordar una barrera durable de calidad sin ejecutar rework ni cierre;
  - el resultado de revision puede referenciar un gate por ref opaca, pero no lo materializa ni lo exige;
  - no entra DB, runtime, proveedor, HOME, OAuth, scheduler ni outbox en el workflow.
Pendiente consciente:
  - hacer obligatorio el gate para cierres o revisiones reales requiere decision separada del director.
```
