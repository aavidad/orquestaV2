# Decisiones locales: orquesta-core-workflow

Las decisiones de este archivo afectan solo a este modulo. Si una decision cambia contratos globales, se registra tambien en `../../CONTRATOS.md` o se eleva `CONSULTA AL DIRECTOR`.

## Plantilla

```text
Fecha:
Decision:
Motivo:
Alternativas:
Impacto:
Contratos afectados:
Estado:
```

```text
Fecha: 2026-05-17
Decision: `MicrotaskCreated` conserva la autorizacion compacta de una
microtarea en el run, pero la metadata rica de `WorkflowTaskV0` pertenece al
`WorkflowTaskStore` de la capa de aplicacion.
Motivo: el workflow puro debe poder replayar refs de tareas sin importar stores,
DB, runtime ni dominio. Campos como `wave_ref`, `cohort_ref`,
`parent_task_ref`, `child_task_refs`, write-set, criterios y limites de
delegacion son necesarios para scheduler/waits/revision, pero no deben convertir
el evento compacto en un snapshot completo de trabajo externo.
Alternativas: guardar todo `WorkflowTaskV0` dentro de `MicrotaskCreated`;
duplicar metadata en `OrchestrationRunV0`; inferir ola/cohorte desde texto o
refs de agente; aceptar que waits no sean recuperables tras reinicio.
Impacto: cualquier bundle productivo debe persistir o rematerializar
`WorkflowTaskStore` junto al replay de eventos. Un replay event-only reconstruye
`run.Tasks`, no `wave_ref`/`cohort_ref` ni linaje completo. Los helpers de
orquestacion solo deben esperar por metadata rica cuando el store este
disponible.
Contratos afectados: CreateMicrotask, MicrotaskCreated, WorkflowTaskV0,
OrchestrationRunV0.Tasks, WorkflowTaskStorePortV0.
Estado: aceptada.
```

```text
Fecha: 2026-05-15
Decision: Los agentes arrancados cuyo proceso/sesion deja de ser controlable se
registran como `AgentLost`, separado de `AgentFailed` y de `AgentStopConfirmed`.
Motivo: tras reinicios o perdida del runtime, Orquesta puede conservar que un
agente fue arrancado y tener incluso una parada solicitada, pero no puede
probar que el proceso se haya detenido. Confirmar parada sin evidencia falsea
la historia; marcarlo como `AgentFailed` mezcla un fallo posterior con fallo de
launch.
Alternativas: reutilizar `failed_agents` con un `reason_code` especial;
confirmar parada por timeout; dejar el run bloqueado esperando un proceso que
ya no existe; guardar detalles del runtime dentro del core.
Impacto: `lost_agents` es una proyeccion compacta terminal para agentes
arrancados sin entrega. `StopAgent`, `RegisterDelivery`,
`RegisterPhaseArtifact`, `RegisterAgentStopConfirmed`, `RegisterAgentStarted`
y `RegisterAgentFailed` rechazan estados contradictorios. `RecordReplanDecision`
acepta `source_kind=agent_lost` en programacion para que el director cree un
reemplazo explicito.
Contratos afectados: RegisterAgentLost, AgentLost, StopAgent,
RegisterAgentStopConfirmed, RegisterDelivery, RegisterPhaseArtifact,
RecordReplanDecision, OrchestrationRunV0.LostAgents.
Estado: aceptada e implementada en NCW-074
```

```text
Fecha: 2026-05-07
Decision: `RecordQualityGate` se registra como transicion durable pura mediante `QualityGateRecorded` y proyeccion compacta `quality_gates`.
Motivo: La calidad debe quedar como evidencia reproducible antes de que director/revision decidan cerrar, pedir retrabajo, bloquear o consultar; mezclar esas acciones en el gate recrearia acoplamiento entre dominio, scheduler y adaptadores.
Alternativas: Guardar resultado completo de calidad en `ReviewResultV0`; generar `RequestRework` o `BlockRun` automaticamente; delegar el gate solo a observabilidad externa; introducir un motor de calidad dentro del core.
Impacto: El workflow guarda `gate_ref`, decision y sujeto compactos, valida decisiones e `issue_refs`, registra `CommandEffects` por `gate_ref` y no emite outbox ni ejecuta followups. `ReviewResultV0.quality_gate_ref` sigue siendo una ref opaca opcional, no una dependencia a DB/proveedor/HOME/runtime.
Contratos afectados: RecordQualityGate, QualityGateRecorded, OrchestrationRunV0.QualityGates, OrchestrationRunV0.CommandEffects.
Estado: aceptada e implementada en NCW-067
```

```text
Fecha: 2026-05-06
Decision: `PhaseOpened` entra en `CommandEffects` usando `event_id` como sujeto, no `phase_id`.
Motivo: El modelo actual permite reaperturas de fase y conserva el `opened_at` original. Usar `phase_id` como identidad unica bloquearia esa semantica; usar `event_id` protege el retry exacto y rechaza payloads distintos para la misma apertura sin impedir eventos nuevos.
Alternativas: Prohibir reaperturas; usar `phase_id` como ref unica; guardar historial completo de aperturas; mantener no-op por fase activa.
Impacto: Repetir la misma apertura ya reflejada es no-op aunque el run haya avanzado a otra fase; cambiar payload, command_id, idempotency_key o event_id para esa apertura se rechaza. Abrir la misma fase con otro idempotency key crea un evento durable nuevo.
Contratos afectados: OpenPhase, PhaseOpened, OrchestrationRunV0.CommandEffects.
Estado: aceptada e implementada en NCW-065
```

```text
Fecha: 2026-05-06
Decision: `RunStarted`, `RunBlocked` y `PhaseClosed` pasan a formar parte de `CommandEffects`; `OpenPhase` queda fuera de este corte.
Motivo: Arranque, bloqueo y cierre de fase son transiciones de lifecycle que no deben aceptar refs ya reflejadas con otro comando o payload. `OpenPhase` permite reapertura de fases en el modelo actual, asi que endurecerlo por `phase_id` cambiaria la semantica y requiere una decision aparte.
Alternativas: Guardar payload completo en estado; eliminar reapertura de fases; usar `phase_id` como identidad unica de apertura; aceptar no-op por proyeccion compacta.
Impacto: Un retry exacto de StartRun/BlockRun/ClosePhase sigue siendo no-op; un ClosePhase exacto puede repetirse aunque el run haya avanzado a otra fase. Cambios de payload, command_id, idempotency_key o event_id se rechazan. No se anaden outbox, DB, runtime, proveedor, HOME ni scheduler.
Contratos afectados: RunStarted, RunBlocked, PhaseClosed, OrchestrationRunV0.CommandEffects.
Estado: aceptada e implementada en NCW-064
```

```text
Fecha: 2026-05-06
Decision: `ConcurrencyGateRecorded`, `AgentLeaseExpired` y `DirectorQuestionAnswered` pasan a formar parte de `CommandEffects`.
Motivo: Estas senales gobiernan supervision, leases y respuestas humanas/IA. Si se aceptan como no-op solo por ref, un gate, una expiracion o una respuesta del director podrian cambiar payload/metadatos y dejar una historia aparentemente valida pero ambigua.
Alternativas: Guardar payload completo en estado; ampliar proyecciones compactas con todos los campos; resolver conflictos en director/scheduler; aceptar no-op por ref.
Impacto: Un retry exacto sigue siendo no-op, incluso si la respuesta del director ya desbloqueo el run; si la ref ya esta proyectada, cambiar payload, command_id, idempotency_key o event_id produce conflicto publico. No se anaden outbox, proveedor, modelo, DB, HOME, runtime ni timers.
Contratos afectados: ConcurrencyGateRecorded, AgentLeaseExpired, DirectorQuestionAnswered, OrchestrationRunV0.CommandEffects.
Estado: aceptada e implementada en NCW-063
```

```text
Fecha: 2026-05-06
Decision: `CapacityDecided`, `AgentStarted`, `AgentFailed`, `AgentStopConfirmed` y `DeliveryRegistered` pasan a formar parte de `CommandEffects`.
Motivo: Estas refs son evidencia operativa de programacion. Si se aceptan como no-op solo por ref, una decision de capacidad, un ACK de agente, un fallo, una parada confirmada o una entrega podrian cambiar payload/metadatos sin que replay lo detectase.
Alternativas: Guardar payload completo en estado; ampliar listas compactas con todos los campos; delegar conflictos a runtime/persistence; aceptar el riesgo por ser senales externas.
Impacto: Un retry exacto sigue siendo no-op; si la ref ya esta proyectada, cambiar payload, command_id, idempotency_key o event_id produce conflicto publico. No se anaden outbox, proveedor, modelo, DB, HOME ni runtime.
Contratos afectados: CapacityDecided, AgentStarted, AgentFailed, AgentStopConfirmed, DeliveryRegistered, OrchestrationRunV0.CommandEffects.
Estado: aceptada e implementada en NCW-062
```

```text
Fecha: 2026-05-06
Decision: `TaskClosed`, `FinalValidationRegistered` y `RunClosed` pasan a formar parte de `CommandEffects`.
Motivo: Las refs de cierre son la evidencia final del producto. Reutilizar una ref de cierre con otro payload o metadatos de idempotencia podria cerrar tareas o runs con historia ambigua.
Alternativas: Guardar payload completo en estado; ampliar proyecciones de cierre con todos los campos; aceptar no-op por ref; resolver conflictos en observabilidad.
Impacto: Un retry exacto sigue siendo no-op, incluso despues de `RunClosed`; si la ref ya esta proyectada, cambiar payload, command_id, idempotency_key o event_id produce conflicto publico. No se anaden outbox, proveedor, modelo, DB, HOME ni runtime.
Contratos afectados: TaskClosed, FinalValidationRegistered, RunClosed, OrchestrationRunV0.CommandEffects.
Estado: aceptada e implementada en NCW-061
```

```text
Fecha: 2026-05-06
Decision: `ReviewResultRecorded`, `ReworkRequested` y `ReplanDecisionRecorded` pasan a formar parte de `CommandEffects`.
Motivo: Sus proyecciones compactas no guardan todos los campos del payload. Sin huella durable, una misma ref podria repetir status/fuente principales pero cambiar resumen, evidencias o metadatos de idempotencia sin que el replay lo detectase.
Alternativas: Guardar payload completo en `OrchestrationRunV0`; ampliar las proyecciones compactas con todos los campos; aceptar la perdida de detalle; resolver conflictos solo al cerrar tarea.
Impacto: Un retry exacto sigue siendo no-op; si la ref ya esta proyectada, cambiar payload completo, command_id, idempotency_key o event_id produce conflicto publico. No se anaden outbox, proveedor, modelo, DB, HOME ni runtime.
Contratos afectados: ReviewResultRecorded, ReworkRequested, ReplanDecisionRecorded, OrchestrationRunV0.CommandEffects.
Estado: aceptada e implementada en NCW-060
```

```text
Fecha: 2026-05-06
Decision: `ReviewRequested` y `ReviewAccepted` pasan a formar parte de `CommandEffects`.
Motivo: La revision es la barrera que decide si una entrega puede avanzar hacia cierre. Si una misma ref de revision o aceptacion pudiera repetirse con otro payload, el replay podria ocultar decisiones contradictorias.
Alternativas: Mantener solo listas compactas `reviews`/`accepted_reviews`; comparar payload completo en estado; resolver conflictos en cierre de tarea; crear un store especial de revision.
Impacto: Un retry exacto sigue siendo no-op; si la ref ya esta proyectada, cambiar payload, command_id, idempotency_key o event_id produce conflicto publico. No se anaden outbox, proveedor, modelo, DB, HOME ni runtime.
Contratos afectados: RequestReview, ReviewRequested, AcceptReview, ReviewAccepted, OrchestrationRunV0.CommandEffects.
Estado: aceptada e implementada en NCW-059
```

```text
Fecha: 2026-05-06
Decision: `BrainstormRequested` y `VoteRequested` pasan a formar parte de `CommandEffects`.
Motivo: Las fases iniciales deciden arquitectura y rumbo del proyecto; reutilizar `brainstorm_request_id` o `vote_request_id` con otro payload bajo apariencia de idempotencia ocultaria decisiones distintas y degradaria el replay.
Alternativas: Mantener solo listas compactas `brainstorms`/`votes`; comparar payload completo en estado; delegar el conflicto al director; crear un store especial para arquitectura.
Impacto: Un retry exacto sigue siendo no-op; si la ref ya esta proyectada, cambiar payload, command_id, idempotency_key o event_id produce conflicto publico. No se anaden outbox, proveedor, modelo, DB, HOME ni runtime.
Contratos afectados: RequestBrainstorm, BrainstormRequested, RequestVote, VoteRequested, OrchestrationRunV0.CommandEffects.
Estado: aceptada e implementada en NCW-058
```

```text
Fecha: 2026-05-06
Decision: `RecordReplanDecision` puede usar `AgentFailed` como fuente durable en `programacion`.
Motivo: Un fallo de lanzamiento necesita una decision durable de recuperacion sin pasar por revision ni `ReworkRequested`, porque todavia no existe entrega revisable. Mantenerlo como replan separado evita retries invisibles y conserva replay determinista.
Alternativas: Forzar AskDirector manual; crear ReworkRequested artificial; relanzar agente desde AgentFailed; mezclar fallo de launch con StopAgent.
Impacto: `source_ref` acepta refs en `rework_requests` durante `revision` o refs en `failed_agents` durante `programacion`; `RecordReplanDecision` sigue sin emitir outbox ni materializar followups.
Contratos afectados: RecordReplanDecision, ReplanDecisionRecorded, AgentFailed, OrchestrationRunV0.ReplanDecisions.
Estado: aceptada e implementada en NCW-054
```

```text
Fecha: 2026-05-09
Decision: `RecordReplanDecision` puede usar `AgentWorkAssessed` como fuente durable en `programacion`.
Motivo: Un agente parado por trabajo basura o `loop_detected` no siempre fallo al arrancar ni produjo una entrega revisable; la evidencia durable correcta es `assessment_ref`. Sin esta fuente, el reemplazo del agente exigiria inventar `ReworkRequested` o mezclar parada con replan.
Alternativas: Tratar la parada como `AgentFailed`; crear `ReworkRequested` artificial; permitir `source_ref` libre; relanzar automaticamente desde `AssessAgentWork`.
Impacto: `source_ref` acepta refs en `agent_assessments` durante `programacion`; `RecordReplanDecision` sigue sin emitir outbox, pedir capacidad, crear agentes ni materializar followups.
Contratos afectados: RecordReplanDecision, ReplanDecisionRecorded, AgentWorkAssessed, OrchestrationRunV0.ReplanDecisions.
Estado: aceptada e implementada en NCW-069
```

```text
Fecha: 2026-05-06
Decision: `StopAgent` no se emite para agentes con `AgentFailed` ya proyectado.
Motivo: Si el lanzamiento fallo, no hay runtime confirmado que parar para ese `agent_request_id`; emitir `StopRuntimeAgent` despues del fallo mezcla cancelacion con fallo de launch y crea historia ambigua.
Alternativas: Permitir stop posterior y confiar en runtime; reinterpretar failed como warning; convertir StopAgent en no-op tras fallo; crear estado mutable unico.
Impacto: `StopAgent` y `AgentStopRequested` rechazan `failed_agents`; parar agentes arrancados o cancelar agentes solicitados sigue permitido.
Contratos afectados: StopAgent, AgentStopRequested, AgentFailed.
Estado: aceptada e implementada en NCW-053
```

```text
Fecha: 2026-05-06
Decision: El lifecycle de lanzamiento de agente no admite outcomes contradictorios.
Motivo: Una senal tardia de `AgentStarted` despues de `AgentFailed` o `AgentStopRequested`, o un `AgentFailed` despues de `AgentStarted`, reabre estados imposibles y puede habilitar entregas/retries con evidencia falsa.
Alternativas: Permitir todos los eventos y resolver en revision; reinterpretar `AgentFailed` como fallo runtime posterior; dejarlo al adaptador; fusionar start/fail/stop en un unico status mutable.
Impacto: `RegisterAgentStarted`/`AgentStarted` rechazan failed/stopped previos; `RegisterAgentFailed`/`AgentFailed` rechazan started/stopped previos; los reintentos del mismo outcome ya reflejado siguen siendo no-op/idempotentes.
Contratos afectados: RegisterAgentStarted, AgentStarted, RegisterAgentFailed, AgentFailed, AgentStopRequested.
Estado: aceptada e implementada en NCW-052
```

```text
Fecha: 2026-05-06
Decision: `RegisterDelivery` exige agente arrancado y rechaza agentes fallidos.
Motivo: `AgentRequested` solo significa que el core pidio un agente; sin `AgentStarted` la entrega podria venir de trabajo no confirmado. Si `AgentFailed` ya quedo durable, aceptar entrega posterior contaminaria revision y cierre con una fuente invalida.
Alternativas: Reinterpretar `agents` como arrancados; resolverlo solo en revision; permitir entregas hasta parada confirmada; dejar que el adaptador descarte entregas tarde.
Impacto: `RegisterDelivery` y `DeliveryRegistered` requieren `agent_ref` en `started_agents`, rechazan `failed_agents` y mantienen tambien el bloqueo por `stopped_agents`.
Contratos afectados: RegisterDelivery, DeliveryRegistered, AgentStarted, AgentFailed, AgentStopRequested.
Estado: aceptada e implementada en NCW-051
```

```text
Fecha: 2026-05-06
Decision: `RegisterDelivery` no acepta entregas de agentes con parada solicitada.
Motivo: Si `StopAgent` ya quedo en la historia durable, aceptar una entrega posterior permitiria que trabajo basura o en bucle contaminara revision/cierre.
Alternativas: Permitir entrega hasta confirmacion de parada; resolverlo en revision; reabrir agente con otro comando; dejarlo al adaptador.
Impacto: `RegisterDelivery` y `DeliveryRegistered` rechazan `agent_ref` en `stopped_agents`; replan o relanzamiento deben crear comandos separados.
Contratos afectados: RegisterDelivery, DeliveryRegistered, AgentStopRequested.
Estado: aceptada e implementada en NCW-050
```

```text
Fecha: 2026-05-06
Decision: Separar parada solicitada de parada confirmada mediante `RegisterAgentStopConfirmed -> AgentStopConfirmed`.
Motivo: `AgentStopRequested` significa que el core pidio parar y emitio outbox; no demuestra que el adaptador haya detenido el trabajo. Sin una confirmacion durable, revision/replan/scheduler podrian asumir una parada efectiva que solo fue solicitada.
Alternativas: Reinterpretar `stopped_agents` como parada efectiva; guardar ACK de persistence como estado de agente; consultar runtime desde el reducer; esperar a un scheduler productivo.
Impacto: `OrchestrationRunV0` agrega `confirmed_stopped_agents`; la confirmacion exige stop previo, no emite outbox y queda disponible para director/MCP/E2E.
Contratos afectados: RegisterAgentStopConfirmed, AgentStopConfirmed, OrchestrationRunV0.ConfirmedStoppedAgents, resource MCP de workflow.
Estado: aceptada e implementada en NCW-049
```

```text
Fecha: 2026-05-06
Decision: El catalogo de comandos y eventos soportados se publica desde `orquesta-core-workflow` como API publica pequena.
Motivo: MCP y otros adaptadores de gobierno no pueden mantener listas paralelas; si el workflow crece y el resource de IA queda desfasado, el director externo gobierna con un mapa incompleto.
Alternativas: Seguir duplicando listas en MCP; leer codigo/docs en runtime; ampliar un documento global manual despues de cada corte.
Impacto: `SupportedOrchestrationCommandTypesV0` y `SupportedOrchestrationEventTypesV0` son la fuente compacta para adaptadores; los validadores del core usan el mismo catalogo.
Contratos afectados: OrchestrationCommandV0, OrchestrationEventV0, resource MCP `orquesta.core_workflow.contracts.v0`.
Estado: aceptada e implementada en NCW-048
```

```text
Fecha: 2026-05-06
Decision: `RecordConcurrencyGate` registra la decision de un gate ya evaluado, pero no calcula scopes ni actua como scheduler.
Motivo: La concurrencia por read/write-set requiere politica pura y prueba independiente. Si el workflow calcula conflictos o lanza agentes desde el gate, volveriamos a mezclar planificador, runtime y estado durable.
Alternativas: Importar `orquesta-core-concurrency` al workflow; bloquear el run completo ante cualquier conflicto; hacer obligatorio el gate dentro de `RequestAgent` en este corte.
Impacto: `OrchestrationRunV0` agrega `concurrency_gates`; `allow_request_agent`/`block_request_agent` quedan como evidencia durable para que el director decida el siguiente comando.
Contratos afectados: RecordConcurrencyGate, ConcurrencyGateRecorded, OrchestrationRunV0.ConcurrencyGates.
Estado: aceptada e implementada en NCW-047
```

```text
Fecha: 2026-05-06
Decision: `RegisterAgentLeaseExpired` registra una expiracion observada como `AgentLeaseExpired`, sin convertir la recomendacion en efecto operativo.
Motivo: Un lease vencido debe quedar en la historia durable para auditoria y replanificacion, pero el core no puede ejecutar timers, parar procesos, marcar fallos ni decidir runtime durante replay.
Alternativas: Parar automaticamente desde el timeout; marcar `failed_agents` o `stopped_agents` desde la expiracion; dejar leases solo en runtime; importar `orquesta-core-leases` al workflow.
Impacto: `OrchestrationRunV0` agrega `agent_lease_expirations`; la idempotencia es por `lease_ref`; la accion posterior entra por comandos separados (`StopAgent`, `AskDirector`, replan u otro corte).
Contratos afectados: RegisterAgentLeaseExpired, AgentLeaseExpired, OrchestrationRunV0.AgentLeaseExpirations.
Estado: aceptada e implementada en NCW-046
```

```text
Fecha: 2026-05-06
Decision: `RecordReplanDecision` registra una decision compacta de replan solo si existe un `ReworkRequested` previo como fuente.
Motivo: Cerrar el bucle de retrabajo necesita una decision durable explicita antes de crear tareas, pedir capacidad o relanzar agentes. Registrar la decision sin materializar efectos evita retry automatico y mantiene el replay determinista.
Alternativas: Crear tareas/agentes desde `ReworkRequested`; relanzar por politica del replanner; permitir `source_ref` libre; mezclar decision de replan con capacidad.
Impacto: `OrchestrationRunV0` agrega `replan_decisions`; la idempotencia es por `replan_ref`; los `followup_refs` son trazas opacas, no efectos ejecutados.
Contratos afectados: RecordReplanDecision, ReplanDecisionRecorded, ReworkRequested, OrchestrationRunV0.ReplanDecisions.
Estado: aceptada e implementada en NCW-045
```

```text
Fecha: 2026-05-06
Decision: `RequestRework` registra solo una solicitud compacta de retrabajo si existe un `ReviewResultRecorded` previo con status `changes_requested` o `rejected` para la misma revision y entrega.
Motivo: El core necesita memoria durable de que una entrega requiere retrabajo sin convertirlo en relanzamiento automatico de agentes, replanificacion, cierre o decision de runtime.
Alternativas: Disparar rework automaticamente desde `ReviewResultRecorded`; relanzar agentes desde el core; mezclar `RequestRework` con `RecordReplanDecision`; permitir rework sobre resultados `accepted`.
Impacto: `OrchestrationRunV0` agrega `rework_requests`; la idempotencia es por `rework_request_ref`; conflictos de resultado/revision/entrega se rechazan; outbox y agentes no cambian.
Contratos afectados: RequestRework, ReworkRequested, ReviewResultRecorded, OrchestrationRunV0.ReworkRequests.
Estado: aceptada e implementada en NCW-044
```

```text
Fecha: 2026-05-06
Decision: `AcceptReview` solo puede aceptarse si existe un `ReviewResultRecorded` previo con status `accepted` para la misma revision y entrega.
Motivo: La aceptacion formal no debe saltarse la revision durable. Esto evita aceptar entregas por un comando suelto o por replay incompleto y mantiene separadas la evaluacion, la aceptacion y el cierre.
Alternativas: Mantener `AcceptReview` dependiente solo de `ReviewRequested`; fusionar `RecordReviewResult` y `AcceptReview`; cerrar la tarea desde `ReviewResultRecorded(accepted)`.
Impacto: Los flujos validos registran primero el resultado aceptado; `changes_requested` y `rejected` no habilitan aceptacion; `CloseTask` sigue dependiendo de `ReviewAccepted`.
Contratos afectados: AcceptReview, ReviewAccepted, ReviewResultRecorded, OrchestrationRunV0.ReviewResults.
Estado: aceptada e implementada en NCW-043
```

```text
Fecha: 2026-05-06
Decision: `RecordReviewResult` registra `ReviewResultV0` como `ReviewResultRecorded` y proyecta solo una ref compacta en `ReviewResults` con status, review y entrega.
Motivo: El core necesita memoria durable de `accepted|changes_requested|rejected` sin mezclarlo con aceptacion formal, cierre de tarea, rework automatico ni politicas de replanificacion.
Alternativas: Traducir `accepted` directamente a `AcceptReview`; cerrar tareas desde el resultado; disparar `RequestRework`; guardar payload completo en estado.
Impacto: La idempotencia es por `review_result_ref`; el evento no emite outbox y el reducer no modifica `AcceptedReviews` ni `ClosedTasks`.
Contratos afectados: RecordReviewResult, ReviewResultRecorded, OrchestrationRunV0.ReviewResults.
Estado: aceptada e implementada en NCW-042
```

```text
Fecha: 2026-05-05
Decision: `AskDirector` distingue pregunta emitida de pregunta respondida; mientras no exista respuesta puede reconstruir `SendDirectorQuestion`, y una pregunta respondida no puede re-bloquear el run por repetir el comando.
Motivo: Una perdida parcial de outbox no debe ocultar la consulta al director, pero repetir un comando antiguo tras `DirectorQuestionAnswered` tampoco debe volver a bloquear trabajo ya desbloqueado.
Alternativas: Mantener no-op cuando la pregunta ya existe; deducir respuestas solo por `director_answers`; dejar el control al adaptador UI/MCP/CLI.
Impacto: `OrchestrationRunV0` agrega `director_answered_questions`; `DirectorQuestionAnswered` proyecta `question_id`; `AskDirector` reemite outbox hasta respuesta y no reabre blockers respondidos.
Contratos afectados: AskDirector, DirectorQuestionRaised, DirectorQuestionAnswered, OrchestrationRunV0.
Estado: aceptada e implementada en NCW-041
```

```text
Fecha: 2026-05-05
Decision: Repetir `RequestCapacity` o `RequestAgent` con evento ya proyectado puede reconstruir solo el outbox pendiente si el efecto externo aun no esta cerrado por una decision/lifecycle posterior.
Motivo: Un fallo parcial de persistencia entre evento y outbox no debe dejar al sistema sin pedir capacidad ni sin lanzar agente. La idempotencia no puede significar "no-op" cuando falta el efecto externo derivado.
Alternativas: Resolverlo solo en persistence; exigir al dispatcher reconstruir mensajes desde eventos; mantener no-op y aceptar perdida de launch/capacity.
Impacto: `RequestCapacity` reemite `RequestCapacityDecision` hasta `CapacityDecided`; `RequestAgent` reemite `LaunchRuntimeAgent` hasta `AgentStarted`, `AgentFailed` o `AgentStopRequested`. No se duplican eventos.
Contratos afectados: RequestCapacity, RequestCapacityDecision, RequestAgent, LaunchRuntimeAgent.
Estado: aceptada e implementada en NCW-040
```

```text
Fecha: 2026-05-05
Decision: `RequestAgent` solo puede progresar si existe una `CapacityDecided` previa para `capacity_request_ref`; la decision se registra con `RegisterCapacityDecision` y se proyecta en `capacity_decisions`.
Motivo: El runtime no debe arrancar trabajo con una capacidad pendiente. La barrera durable permite reintentos, auditoria y replay sin filtrar provider, modelo concreto, HOME, OAuth, cuota real ni credenciales al core.
Alternativas: Mantener `capacity_request_ref` opcional; confiar en runtime para resolver capacidad; guardar la respuesta completa de `orquesta-capacity` en el core.
Impacto: `capacity_request_ref` pasa a ser obligatorio en `RequestAgent`/`AgentRequested`; director y e2e registran la decision compacta antes de lanzar agentes; solo hay una decision por `capacity_request_id`.
Contratos afectados: RegisterCapacityDecision, CapacityDecided, RequestAgent, AgentRequested, LaunchRuntimeAgentRequestV0, OrchestrationRunV0.
Estado: aceptada e implementada en NCW-039
```

```text
Fecha: 2026-05-05
Decision: `AgentRequested` sigue significando agente pedido; `AgentStarted` y `AgentFailed` proyectan estados separados `started_agents` y `failed_agents` con refs opacas.
Motivo: El director necesita distinguir pedido aceptado por el core, arranque confirmado por adaptador y fallo de lanzamiento sin introducir runtime, provider, HOME, PID ni detalles de proceso en el nucleo.
Alternativas: Reinterpretar `agents` como agentes arrancados; guardar snapshots de runtime; esperar a un contrato completo de ACK externo.
Impacto: Se agregan comandos `RegisterAgentStarted`/`RegisterAgentFailed`, eventos `AgentStarted`/`AgentFailed` y proyecciones compactas. E2E registra `AgentStarted` tras ACK de launch con proceso controlado.
Contratos afectados: RequestAgent, AgentStarted, AgentFailed, OrchestrationRunV0.
Estado: aceptada e implementada en NCW-038
```

```text
Fecha: 2026-05-05
Decision: `RequestCapacity`, `RequestAgent` y `AssessAgentWork` solo aceptan `phase_id` si coincide con `current_phase`; un reintento de `AssessAgentWork action=stop_agent` con assessment ya proyectada pero parada pendiente reconstruye `AgentStopRequested` y outbox.
Motivo: La orquestacion por fases no puede aceptar trabajo para fases no activas, y un fallo parcial de persistencia no debe dejar sin parada a un agente detectado en bucle o trabajo basura.
Alternativas: Mantener solo validacion de catalogo; resolver reintento parcial en persistence; bloquear todos los reintentos cuando cambia la fase.
Impacto: Los eventos/replay de capacidad, agente y evaluacion exigen fase activa; los tests progresivos abren `programacion` antes de solicitar capacidad/agentes. La idempotencia de assessment conserva el efecto pendiente de parada.
Contratos afectados: RequestCapacity, CapacityRequested, RequestAgent, AgentRequested, AssessAgentWork, AgentWorkAssessed.
Estado: aceptada e implementada en NCW-037
```

```text
Fecha: 2026-05-05
Decision: `AnswerDirectorQuestion` registra una respuesta compacta del director mediante `DirectorQuestionAnswered`, sin outbox, y solo puede desbloquear el blocker `director-question-<question_id>` asociado.
Motivo: La respuesta del director es conocimiento de dominio ya resuelto; reemitirla por outbox duplicaria efectos y mezclarla con adaptadores. El desbloqueo debe ser trazable y acotado a la pregunta que bloqueo el run.
Alternativas: Reutilizar `AskDirector`; emitir `SendDirectorAnswer`; limpiar cualquier blocker activo; guardar respuesta completa con contexto operacional.
Impacto: El payload conserva refs opacas y prohibe secretos, provider/proveedor, HOME, DB, runtime, prompts, transcripts y contexto masivo. Otros blockers no se limpian implicitamente.
Contratos afectados: AnswerDirectorQuestion, DirectorQuestionAnswered, OrchestrationRunV0.
Estado: aceptada e implementada en NCW-036
```

```text
Fecha: 2026-05-05
Decision: `ReviewResultV0` se mantiene como DTO/validador puro y no se conecta todavia a comandos, eventos, reducer ni outbox.
Motivo: El resultado compacto de una revision necesita validarse antes de traducirse a comandos durables existentes, pero mezclarlo ahora con `AcceptReview` o `CloseTask` abriria demasiado el nucleo.
Alternativas: Convertirlo directamente en nuevo comando/evento; ampliar `AcceptReview`; cerrar tareas desde resultados no aceptados.
Impacto: `accepted` queda como outcome apto para una aceptacion durable posterior; `changes_requested` y `rejected` no cierran tarea. El DTO rechaza provider/proveedor, HOME, OAuth, DB, prompts, transcripts, runtime, conectores y payload masivo.
Contratos afectados: ReviewResultV0.
Estado: aceptada local en NCW-035
```

```text
Fecha: 2026-05-05
Decision: `capacity_command_v0.go` queda como tipos/constructores y la transicion de capacidad se reparte en handler/idempotencia y validacion/helpers locales.
Motivo: El fichero habia crecido a 281 lineas y mezclaba DTOs, constructores, decodificacion, validacion, transicion y helpers compartidos con outbox.
Alternativas: Mantener el fichero monolitico; mover validacion a un archivo generico de comandos; cambiar nombres publicos junto con el saneamiento.
Impacto: `RequestCapacityCommandPayloadV0`, `CapacityRequestedPayloadV0`, `NewRequestCapacityCommandV0`, `NewCapacityRequestedEventV0`, errores publicos, JSON tags y outbox `RequestCapacityDecision` conservan comportamiento.
Contratos afectados: sin cambios funcionales.
Estado: aceptada local en NCW-034
```

```text
Fecha: 2026-05-05
Decision: `agent_stop_v0.go` queda como tipos/constructores y la parada de agente se reparte por handler/reducer, outbox, validacion y helpers locales.
Motivo: El fichero habia crecido a 280 lineas y mezclaba DTOs, transicion, efecto logico, validadores y normalizacion, aunque el contrato ya estaba estable.
Alternativas: Mantener el fichero unido; mover partes a ficheros genericos de agentes; cambiar nombres publicos junto con el saneamiento.
Impacto: `StopAgent`, `AgentStopRequested`, `StopRuntimeAgentRequestV0`, `newStopRuntimeAgentOutboxV0` y helpers reutilizados por `AssessAgentWork` conservan firma y comportamiento.
Contratos afectados: sin cambios funcionales.
Estado: aceptada local en NCW-033
```

```text
Fecha: 2026-05-05
Decision: `reducer_v0.go` queda como dispatch publico y el detalle del reducer se reparte por responsabilidad local.
Motivo: El fichero habia crecido a 299 lineas y mezclaba router, ciclo de vida, proyecciones auxiliares, helpers y clonado, aumentando el contexto necesario para cambios pequenos.
Alternativas: Mantener el fichero monolitico hasta superar 400 lineas; mover reducers a paquetes internos; cambiar nombres o contratos publicos junto con el refactor.
Impacto: `ApplyEventV0` y `ReplayEventsV0` siguen en `reducer_v0.go`; los apply de run/fase/bloqueo, proyecciones restantes, helpers y clone pasan a ficheros pequenos sin cambio funcional.
Contratos afectados: sin cambios funcionales.
Estado: aceptada local en NCW-032
```

```text
Fecha: 2026-05-05
Decision: `AssessAgentWork` puede registrar action `ask_director`, pero no crea `DirectorQuestionRaised` ni `SendDirectorQuestion`.
Motivo: Evaluar trabajo y consultar al director son fronteras distintas; ocultar una pregunta dentro de la evaluacion duplicaria efectos y haria menos auditable la transicion.
Alternativas: Hacer que `AssessAgentWork` dispare la consulta; crear un outbox especifico de evaluacion; mezclar bloqueo y evaluacion en un unico comando.
Impacto: La evaluacion proyecta solo `AgentAssessments`; la consulta y el bloqueo opcional siguen perteneciendo a `AskDirector`.
Contratos afectados: AssessAgentWork, AgentWorkAssessed, AskDirector, DirectorQuestionRaised, SendDirectorQuestion, RunBlocked.
Estado: aceptada local en NCW-031
```

```text
Fecha: 2026-05-05
Decision: `AssessAgentWork` registra evaluaciones compactas y reutiliza `AgentStopRequested`/`StopRuntimeAgent` cuando la accion exige parada logica.
Motivo: El nucleo necesita auditar progreso basura o bucles sin introducir evaluadores, modelos, runtime real ni proveedores en el handler.
Alternativas: Crear un outbox de evaluacion a modelos; guardar transcripts o contexto completo; detener agentes directamente desde el core.
Impacto: `AgentWorkAssessed` proyecta solo `assessment_ref` en `OrchestrationRunV0.AgentAssessments`; la parada sigue saliendo por el puerto logico `agent_launcher`.
Contratos afectados: AssessAgentWork, AgentWorkAssessed, AgentStopRequested, StopRuntimeAgentRequestV0, OrchestrationRunV0.
Estado: aceptada local en NCW-029
```

```text
Fecha: 2026-05-04
Decision: Crear `orquesta-core-workflow` como nucleo nuevo sin borrar ni pisar `orquesta-core`.
Motivo: El core actual contiene contratos utiles, pero no debe crecer como scheduler central. El usuario pidio no perder el nucleo actual y evitar repetir bucles de v1/v2.
Alternativas: Reescribir `orquesta-core` in-place; continuar ampliando el core actual; volver a v1.
Impacto: El nucleo durable se desarrolla en contexto propio, con microtareas y contratos locales. El core actual queda como referencia y compatibilidad.
Contratos afectados: ninguno global todavia.
Estado: aceptada
```

```text
Fecha: 2026-05-04
Decision: Guardar snapshot comprimido del core actual antes de abrir el nucleo nuevo.
Motivo: Hay que conservar lo que ya funciona sin introducir un segundo paquete Go duplicado que entre en `go test ./...`.
Alternativas: Copiar el directorio completo bajo `modulos/`; no hacer copia; usar solo Git.
Impacto: El estado actual queda preservado en `docs/reinicio_orquesta_v2/snapshots/orquesta-core-actual-2026-05-04.tgz`.
Contratos afectados: ninguno.
Estado: ejecutada
```

```text
Fecha: 2026-05-04
Decision: El nucleo v0 se implementa como workflow durable propio en Go con estado, comandos, eventos, reducer y outbox.
Motivo: Es el patron probado por motores de workflow, controladores declarativos y pipelines: estado pequeno, eventos inmutables, replay, idempotencia y efectos externos fuera de la transicion.
Alternativas: Director monolitico por prompt; adoptar Temporal/Durable Task desde el primer corte; copiar el control plane v1; dejar la DB como cerebro.
Impacto: El primer corte es testeable sin DB, runtime ni modelos. En el futuro se puede sustituir el motor por un conector externo si las pruebas lo justifican.
Contratos afectados: OrchestrationRunV0, OrchestrationCommandV0, OrchestrationEventV0, OutboxMessageV0.
Estado: aceptada local
```

```text
Fecha: 2026-05-04
Decision: Los puertos logicos del outbox no usan palabras reservadas de infraestructura aunque representen capacidades futuras.
Motivo: `runtime_agent` chocaba con la politica de no transportar detalles runtime en el core y haria fallar mensajes validos futuros.
Alternativas: Permitir `runtime` en envelope; quitar `LaunchRuntimeAgent`; renombrar el puerto a una capacidad logica.
Impacto: El mensaje candidato conserva nombre historico `LaunchRuntimeAgent`, pero el puerto logico pasa a `agent_launcher`.
Contratos afectados: OutboxMessageV0.
Estado: aceptada local
```

```text
Fecha: 2026-05-04
Decision: `StartRunFromAppSpecV0` se mantiene como candidato local basado en un draft compacto, no como puente directo al core actual.
Motivo: El modulo necesita cerrar el mapping hacia `StartRun` sin importar `orquesta-core` ni `orquesta-factory` y sin promover aun un contrato global entre AppSpec, ProyectoPlanBorrador y workflow durable.
Alternativas: Importar `RegistrarProyectoDesdeAppSpec v0`; copiar DTOs de factory/core; bloquear NCW-006 hasta contrato global; dejar solo documentacion sin prueba ejecutable.
Impacto: `orquesta-core-workflow` expone `AppSpecRunDraftV0` con refs opacas y metadatos minimos, valida detalles prohibidos y produce un comando local `StartRun` mediante constructores existentes.
Contratos afectados: StartRunFromAppSpecV0 candidato local, OrchestrationCommandV0.
Estado: aceptada local sin promocion global
```

```text
Fecha: 2026-05-04
Decision: Promover `OrchestrationRun v0`, `OutboxMessage v0` y `DirectorQuestion v0` como contratos globales minimos del workflow durable.
Motivo: NCW-001..NCW-005 estabilizaron estado, eventos, handler inicial, outbox, consulta al director, replay e idempotencia; otros modulos necesitan una frontera publica sin importar internals.
Alternativas: Mantenerlos solo locales; promover tambien comandos/eventos completos; esperar a adaptadores reales de persistence/runtime/observability.
Impacto: `../../CONTRATOS.md` declara propietario, consumidores, DTOs, invariantes, errores publicos y prohibiciones de detalles DB/runtime/proveedor/HOME/OAuth/transcripts/contexto masivo. Los detalles extensos siguen en docs locales y no se toca codigo Go.
Contratos afectados: OrchestrationRunV0, OutboxMessageV0, DirectorQuestionV0.
Estado: aceptada global en NCW-007
```

```text
Fecha: 2026-05-04
Decision: El handler y el reducer son puros.
Motivo: Los bucles anteriores aparecieron al mezclar decision de dominio con persistencia, runtime, entrega, prompts, SQL y recuperacion operacional.
Alternativas: Handler con repositorios y runtime inyectados; servicios que escriben DB dentro de cada comando; CLI/API como orquestadores reales.
Impacto: `Handle` devuelve eventos y outbox. `Apply` solo transforma estado. Los adaptadores ejecutan efectos despues.
Contratos afectados: OrchestrationCommandResultV0, OutboxMessageV0.
Estado: aceptada local
```

```text
Fecha: 2026-05-04
Decision: La capacidad es dinamica y externa al nucleo.
Motivo: La seleccion de modelo depende de fase, riesgo, calidad, cuota, disponibilidad, proveedor y evidencia. Hardcodearlo en core volveria a acoplar Codex/HOME/OAuth.
Alternativas: Tabla fija fase->modelo; configurar proveedor en core; usar siempre premium.
Impacto: Core puede emitir `RequestCapacityDecision`; `orquesta-capacity` decide low/medium/high/xhigh, proveedor local/remoto y escalado.
Contratos afectados: CapacityDecisionV0, OutboxMessageV0.
Estado: aceptada local
```

```text
Fecha: 2026-05-04
Decision: Las fases v0 se tratan como contrato de dominio, no como texto libre.
Motivo: Para reanudar, auditar y repartir grupos, el estado de fase debe ser reproducible y validable.
Alternativas: Guardar fase como string libre; dejar que cada agente decida su propia fase; inferir fase desde tareas.
Impacto: El nucleo tendra una lista cerrada inicial de fases y reglas de entrada/salida. Los cambios requieren version o consulta.
Contratos afectados: OrchestrationPhaseV0.
Estado: aceptada local
```

```text
Fecha: 2026-05-04
Decision: Cualquier consulta entre grupos se materializa como evento/outbox de consulta al director.
Motivo: El usuario quiere que los grupos no invadan contextos ajenos y que el director resuelva dudas cuando un modulo necesite informacion de otro.
Alternativas: Cargar docs de otros modulos; tocar varios modulos en una sola tarea; resolver por suposicion.
Impacto: El nucleo modela `AskDirector`, `DirectorQuestionRaised` y `SendDirectorQuestion` como via oficial; la proyeccion conserva refs compactas en `director_questions`.
Contratos afectados: OrchestrationCommandV0, OrchestrationEventV0, OrchestrationRunV0, DirectorQuestionV0, OutboxMessageV0.
Estado: aceptada local
```

```text
Fecha: 2026-05-04
Decision: `WorkflowTaskV0` modela microtareas como DTO puro con refs opacas, sin registrar aun comandos ni eventos.
Motivo: El nucleo necesita poder validar unidades pequenas de trabajo sin acoplarse a runtime, DB, proveedor, HOME, agentes concretos ni adaptadores; este corte separa primero el DTO antes de anadir otra transicion durable.
Alternativas: Crear directamente `CreateMicrotask`; guardar detalles de ejecucion en la tarea; importar contratos de otros modulos.
Impacto: El contrato local cubre task refs, fase, write_set, criterios y refs opacas de contratos de funcion. La promocion a evento/comando durable queda para otro corte autorizado.
Contratos afectados: WorkflowTaskV0, WorkflowFunctionContractRefV0.
Estado: aceptada local
```

```text
Fecha: 2026-05-04
Decision: `CreateMicrotask` usa `WorkflowTaskV0` como payload de comando, pero `MicrotaskCreated` solo persiste refs compactas normalizadas.
Motivo: El handler necesita validar write-set, criterios y fase antes de crear la microtarea, mientras que el estado durable debe seguir siendo una proyeccion pequena sin runtime, DB, proveedor, HOME ni adaptadores.
Alternativas: Persistir la tarea completa en el evento; guardar solo `task_id` y perder refs de contratos; ejecutar la tarea desde el handler.
Impacto: `OrchestrationRunV0.Tasks` recibe `task_id` y `FunctionContracts` recibe strings compactos derivados de `contract_ref` o `function_name` sin duplicar. El comando rechaza la proyeccion si el evento resultante seria demasiado grande.
Contratos afectados: CreateMicrotask, MicrotaskCreated, OrchestrationCommandV0, OrchestrationEventV0, OrchestrationRunV0.
Estado: aceptada local en NCW-009
```

```text
Fecha: 2026-05-04
Decision: `RequestCapacity` persiste la solicitud y emite outbox `RequestCapacityDecision`, pero no decide proveedor, modelo ni runtime.
Motivo: La capacidad dinamica debe quedar como frontera durable con el puerto `capacity`; el nucleo solo conserva la intencion, evidencia y minimo recomendado.
Alternativas: Guardar una decision completa en el evento; hardcodear proveedor/modelo en el handler; dejar la solicitud como efecto no durable.
Impacto: `CapacityRequested` proyecta `capacity_request_id` en `OrchestrationRunV0.CapacityRequests` sin duplicar. El outbox lleva un payload compacto `CapacityDecisionRequestV0` tipado y validado para que otro modulo decida.
Contratos afectados: RequestCapacity, CapacityRequested, CapacityDecisionRequestV0, OrchestrationCommandV0, OrchestrationEventV0, OrchestrationRunV0.
Estado: aceptada local en NCW-010
```

```text
Fecha: 2026-05-04
Decision: `RequestAgent` persiste la solicitud y emite outbox `LaunchRuntimeAgent`, pero no decide ni transporta runtime, proveedor, modelo, HOME, OAuth, cuenta, adaptador ni secreto.
Motivo: El nucleo durable debe conservar solo refs compactas y pedir el efecto al puerto logico `agent_launcher`; la ejecucion real pertenece a adaptadores externos.
Alternativas: Guardar configuracion de runtime en el evento; hardcodear proveedor/modelo en el handler; lanzar el agente directamente desde el core.
Impacto: `AgentRequested` proyecta `agent_request_id` en `OrchestrationRunV0.Agents` sin duplicar. El outbox lleva un payload compacto `LaunchRuntimeAgentRequestV0` tipado y validado.
Contratos afectados: RequestAgent, AgentRequested, LaunchRuntimeAgentRequestV0, OrchestrationCommandV0, OrchestrationEventV0, OrchestrationRunV0.
Estado: aceptada local en NCW-011
```

```text
Fecha: 2026-05-04
Decision: `ClosePhase` se modela como transicion durable pura sin outbox.
Motivo: Cerrar una fase es cambio de estado auditable; no debe ejecutar efectos externos ni avanzar implicitamente a otra fase.
Alternativas: Reusar `OpenPhase` para cerrar la anterior; cerrar y abrir en un unico comando; guardar cierre fuera del flujo de eventos.
Impacto: `PhaseClosed` marca la fase actual como `cerrada`, conserva `opened_at`, asigna `closed_at` desde el evento y deja `current_phase` estable hasta una apertura posterior.
Contratos afectados: ClosePhase, PhaseClosed, OrchestrationPhaseV0.
Estado: aceptada local en NCW-012
```

```text
Fecha: 2026-05-04
Decision: `RequestBrainstorm` se modela como transicion durable pura de la fase `brainstorming_arquitectura`, sin outbox.
Motivo: El brainstorming debe quedar auditable antes de pedir capacidad o agentes, pero el nucleo no debe arrancar sesiones ni decidir modelos.
Alternativas: Lanzar agentes directamente desde el handler; mezclar brainstorm con RequestCapacity/RequestAgent; permitir brainstorming en cualquier fase.
Impacto: `BrainstormRequested` proyecta `brainstorm_request_id` en `OrchestrationRunV0.Brainstorms`; la solicitud de agentes premium se hara con `RequestCapacity` y `RequestAgent` como pasos separados.
Contratos afectados: RequestBrainstorm, BrainstormRequested, OrchestrationRunV0.
Estado: aceptada local en NCW-013
```

```text
Fecha: 2026-05-04
Decision: `RequestVote` se modela como transicion durable pura de la fase `votacion_y_decision`, sin aceptar aun la decision final.
Motivo: El sistema de votaciones necesita trazabilidad propia, pero aceptar arquitectura requiere otro comando para validar resultado, evidencia y cierre de fase.
Alternativas: Guardar la decision final dentro de VoteRequested; usar `Decisions` para solicitudes de voto; ejecutar agentes votantes desde el handler.
Impacto: `VoteRequested` proyecta `vote_request_id` en `OrchestrationRunV0.Votes`; `Decisions` queda reservado para `AcceptDecision`.
Contratos afectados: RequestVote, VoteRequested, OrchestrationRunV0.
Estado: aceptada local en NCW-014
```

```text
Fecha: 2026-05-04
Decision: `AcceptDecision` acepta una decision de arquitectura votada solo si `vote_ref` ya existe en la proyeccion del run, pero no abre la siguiente fase ni crea microtareas.
Motivo: La decision aceptada debe quedar como hito durable separado de la votacion y de la planificacion posterior, sin permitir saltarse la evidencia de voto.
Alternativas: Mezclar decision final en `VoteRequested`; cerrar fase automaticamente; generar microtareas desde el mismo comando.
Impacto: `ArchitectureDecisionAccepted` proyecta `decision_ref` en `OrchestrationRunV0.Decisions`; si falta el voto citado, handler y reducer rechazan la transicion; `OpenPhase` y `CreateMicrotask` siguen siendo transiciones independientes.
Contratos afectados: AcceptDecision, ArchitectureDecisionAccepted, OrchestrationRunV0.
Estado: aceptada local en NCW-015
```

```text
Fecha: 2026-05-04
Decision: `StopAgent` queda como transicion durable pura y separada: comando `StopAgent`, evento `AgentStopRequested` y outbox `StopRuntimeAgent` hacia `agent_launcher`.
Motivo: El nucleo debe poder solicitar una parada auditable sin conocer runtime real, proveedor, modelo, HOME, OAuth, cuentas ni adaptadores.
Alternativas: Ejecutar la parada desde el handler; mezclar estado de parada dentro de `Agents`; guardar datos operativos del runtime en el evento.
Impacto: `AgentStopRequested` proyecta `agent_request_id` en `OrchestrationRunV0.StoppedAgents`; el handler exige que el agente ya exista en `Agents` y el payload de outbox es compacto.
Contratos afectados: StopAgent, AgentStopRequested, StopRuntimeAgentRequestV0, OrchestrationRunV0.
Estado: aceptada local en NCW-028
```

```text
Fecha: 2026-05-04
Decision: `PublishFunctionContract` publica refs compactas de contratos de funcion solo desde `planificacion_microtareas` y solo si la decision arquitectonica citada ya existe.
Motivo: Las microtareas deben poder referenciar contratos de funcion estabilizados sin que el nucleo almacene implementacion, runtime, filesystem, DB, proveedor, modelo ni adaptadores.
Alternativas: Seguir publicando contratos implicitamente desde `CreateMicrotask`; guardar el contrato completo en el evento; generar tareas desde el mismo comando.
Impacto: `FunctionContractPublished` proyecta `contract_ref` en `OrchestrationRunV0.FunctionContracts`; `CreateMicrotask` sigue separado y compatible con refs compactas.
Contratos afectados: PublishFunctionContract, FunctionContractPublished, OrchestrationRunV0.
Estado: aceptada local en NCW-016
```

```text
Fecha: 2026-05-04
Decision: `CreateMicrotask` queda subordinado a `planificacion_microtareas` activa y a contratos de funcion ya publicados con `contract_ref` explicito.
Motivo: Las tareas no deben inventar contratos ni saltarse la fase de planificacion; `WorkflowTaskV0` mantiene compatibilidad con `function_name`, pero el comando durable exige refs publicadas.
Alternativas: Permitir crear microtareas desde cualquier fase; aceptar `function_name` como autorizacion; seguir proyectando contratos nuevos desde `MicrotaskCreated`.
Impacto: `MicrotaskCreated` ya no introduce contratos nuevos; solo proyecta tareas que citan contratos publicados. Los fixtures deben abrir planificacion y publicar contrato antes de crear tarea.
Contratos afectados: CreateMicrotask, MicrotaskCreated, WorkflowTaskV0, OrchestrationRunV0.
Estado: aceptada local en NCW-017
```

```text
Fecha: 2026-05-04
Decision: Los indices `docs/contratos.md` y `docs/pruebas.md` deben quedar como mapa compacto y delegar detalles a documentos tematicos.
Motivo: El modulo ya estaba cerca del limite local de 400 lineas por documento; contextos pequenos reducen errores de agentes y facilitan depuracion.
Alternativas: Seguir anadiendo secciones al indice; dividir solo cuando supere el limite; borrar detalle historico.
Impacto: Estado/fases y pruebas de microtareas pasan a docs tematicos sin cambiar contratos ni codigo.
Contratos afectados: documentacion local del modulo.
Estado: aceptada local en NCW-018
```

```text
Fecha: 2026-05-04
Decision: `RegisterDelivery` registra entregas compactas durante `programacion` solo si la tarea y el agente citados ya existen.
Motivo: La revision necesita una entrega durable, pero el nucleo no debe almacenar codigo, diffs, rutas locales, commits, transcripts ni datos de runtime.
Alternativas: Solicitar revision directamente desde el agente; guardar detalles completos de implementacion en el evento; permitir entregas sin agente.
Impacto: `DeliveryRegistered` proyecta `delivery_ref` en `OrchestrationRunV0.Deliveries`; `RequestReview` queda como corte posterior.
Contratos afectados: RegisterDelivery, DeliveryRegistered, OrchestrationRunV0.
Estado: aceptada local en NCW-019
```

```text
Fecha: 2026-05-04
Decision: `RequestReview` registra la intencion durable de revisar una entrega ya registrada desde la fase `revision`.
Motivo: El sistema necesita trazabilidad de revisiones sin ejecutar revisores ni decidir modelos dentro del nucleo.
Alternativas: Ejecutar revisor desde el handler; aceptar revision directamente desde la entrega; permitir revisar entregas no registradas.
Impacto: `ReviewRequested` proyecta `review_request_id` en `OrchestrationRunV0.Reviews`; `AcceptReview` queda como corte posterior.
Contratos afectados: RequestReview, ReviewRequested, OrchestrationRunV0.
Estado: aceptada local en NCW-020
```

```text
Fecha: 2026-05-04
Decision: Separar validacion/catalogo auxiliar de eventos fuera de `events_v0.go`.
Motivo: `events_v0.go` estaba cerca del limite de 400 lineas y el siguiente evento habria aumentado el riesgo de fichero dificil de depurar.
Alternativas: Mantener todo en un unico archivo; esperar a superar el limite; partir por cada tipo de evento.
Impacto: `events_v0.go` queda como envoltorio y constructores base; `events_validation_v0.go` concentra validacion generica y helpers sin cambiar contratos.
Contratos afectados: sin cambios funcionales.
Estado: aceptada local en NCW-021
```

```text
Fecha: 2026-05-04
Decision: `AcceptReview` acepta una revision solicitada solo si la solicitud y la entrega ya existen en la proyeccion durable.
Motivo: La aceptacion debe ser un hito auditable separado de solicitar revision, cerrar tarea o cerrar fase.
Alternativas: Aceptar revision directamente desde la entrega; cerrar tarea en el mismo comando; guardar resultado completo de revision en el evento.
Impacto: `ReviewAccepted` proyecta `accepted_review_ref` en `OrchestrationRunV0.AcceptedReviews`; las transiciones posteriores siguen separadas y el nucleo conserva solo refs compactas.
Contratos afectados: AcceptReview, ReviewAccepted, OrchestrationRunV0.
Estado: aceptada local en NCW-022
```

```text
Fecha: 2026-05-04
Decision: `CloseTask` cierra una tarea revisada solo si la tarea, la entrega y la revision aceptada ya existen en la proyeccion durable.
Motivo: El cierre de tarea debe ser un hito auditable posterior a `ReviewAccepted`, sin mezclarlo con cierre de fase, cierre de run, conectores ni efectos externos.
Alternativas: Cerrar la tarea dentro de `AcceptReview`; cerrar tambien la fase `revision`; guardar detalles completos de entrega o revision en el evento.
Impacto: `TaskClosed` proyecta `task_id` en `OrchestrationRunV0.ClosedTasks`; `ClosePhase`, cierre de run y cualquier efecto externo siguen como transiciones separadas.
Contratos afectados: CloseTask, TaskClosed, OrchestrationRunV0.
Estado: aceptada local en NCW-023
```

```text
Fecha: 2026-05-04
Decision: `RegisterFinalValidation` registra una validacion final compacta solo si la tarea cerrada citada ya existe en la proyeccion durable.
Motivo: La validacion final debe ser un hito auditable posterior al cierre de tareas, sin mezclar cierre de fase, cierre de run, conectores ni efectos externos.
Alternativas: Cerrar el run dentro de `CloseTask`; validar sin citar tarea cerrada; guardar resultados completos de validacion en el evento.
Impacto: `FinalValidationRegistered` proyecta `validation_ref` en `OrchestrationRunV0.Validations`; `ClosePhase` y el cierre de run siguen como transiciones separadas.
Contratos afectados: RegisterFinalValidation, FinalValidationRegistered, OrchestrationRunV0.
Estado: aceptada local en NCW-024
```

```text
Fecha: 2026-05-04
Decision: `CloseRun` cierra el run solo si la validacion final citada ya existe y sin cerrar fase automaticamente.
Motivo: El cierre de run debe ser un hito auditable posterior a `FinalValidationRegistered`, pero la regla local mantiene `ClosePhase` separado para no mezclar estado global del run con estado de fase.
Alternativas: Cerrar fase y run en el mismo comando; cerrar el run desde `RegisterFinalValidation`; permitir cerrar sin `validation_ref` proyectada.
Impacto: `RunClosed` proyecta `closure_ref` en `OrchestrationRunV0.Closures`, marca `status` del run como `cerrado` y conserva `current_phase` con refs compactas. Cualquier cierre de la fase `cierre` debe pasar por `ClosePhase`.
Contratos afectados: CloseRun, RunClosed, OrchestrationRunV0.
Estado: aceptada local en NCW-025
```

```text
Fecha: 2026-05-04
Decision: Separar `run_state_v0.go` en DTOs, catalogo, validacion y helpers sin cambiar contratos.
Motivo: El nucleo acumula fases y refs durables; mantener todo en un unico archivo aumenta el contexto que deben leer los agentes y dificulta el debug posterior.
Alternativas: Esperar a superar 400 lineas; mover validacion junto al reducer; crear paquetes internos separados.
Impacto: `run_state_v0.go` conserva tipos y constantes; `run_state_catalog_v0.go`, `run_state_validation_v0.go` y `run_state_helpers_v0.go` concentran responsabilidades pequenas.
Contratos afectados: sin cambios funcionales.
Estado: aceptada local en NCW-026
```

```text
Fecha: 2026-05-04
Decision: Dividir los ficheros en zona amarilla por escenario y responsabilidad sin cambiar contratos.
Motivo: `reducer_v0_test.go`, `commands_v0.go`, `outbox_v0.go` y `director_question_command_v0.go` estaban cerca del limite local y mezclaban fixtures, validacion, builders y helpers.
Alternativas: Esperar a superar 400 lineas; crear paquetes internos; refactorizar semantica junto con la division.
Impacto: Los tests del reducer quedan por run/fase, director/bloqueo, replay/errores y helpers; commands, outbox y director question separan DTOs/builders de validacion, decoders y efectos.
Contratos afectados: sin cambios funcionales.
Estado: aceptada local en NCW-027
```

```text
Fecha: 2026-05-06
Decision: Registrar `CommandEffects` como huella compacta de efectos externos reparables.
Motivo: La proyeccion solo guardaba refs compactas; un retry con la misma ref podia reconstruir outbox desde otro comando o payload y provocar bucles o efectos externos contradictorios.
Alternativas: Guardar payload completo en el run; delegar todo al ledger de outbox; permitir idempotencia solo por ref.
Impacto: El run conserva una huella por `(event_type, subject_ref)` con ids y hash de payload. Retries exactos reparan outbox; reutilizar ref con otra key, command/event id o payload se rechaza.
Contratos afectados: OrchestrationRunV0.CommandEffects, CapacityRequested, AgentRequested, AgentStopRequested, AgentWorkAssessed, DirectorQuestionRaised, OutboxMessageV0.
Estado: aceptada local en NCW-055
```

```text
Fecha: 2026-05-06
Decision: Mover el dispatch de comandos/eventos a routers internos del paquete.
Motivo: `HandleCommandV0` y `ApplyEventV0` estaban creciendo como switches repetidos; cada nueva transicion aumentaba el riesgo de olvidar una ruta o inflar ficheros centrales.
Alternativas: Mantener switches; crear subpaquetes internos; mover tambien validadores de payload en el mismo corte.
Impacto: `command_router_v0.go` y `event_router_v0.go` contienen rutas internas sincronizadas por tests contra el catalogo publico. La validacion y los errores publicos no cambian.
Contratos afectados: HandleCommandV0, ApplyEventV0, catalogo de comandos/eventos soportados.
Estado: aceptada local en NCW-056
```

```text
Fecha: 2026-05-06
Decision: Extender `CommandEffects` a la cadena critica de planificacion durable.
Motivo: `decision_ref`, `contract_ref` y `task_id` alimentan la fase de programacion; aceptar no-op por ref sin comprobar payload ocultaria decisiones, contratos o microtareas distintas.
Alternativas: Crear otro ledger `ProjectionClaims`; aplicar identidad fuerte a todas las refs en un unico corte; dejarlo solo en tests de replay.
Impacto: ArchitectureDecisionAccepted, FunctionContractPublished y MicrotaskCreated registran huella compacta y rechazan conflictos por misma ref. No se cambia payload publico ni se introduce DB/runtime/proveedor.
Contratos afectados: CommandEffects, AcceptDecision, PublishFunctionContract, CreateMicrotask.
Estado: aceptada local en NCW-057
```

```text
Fecha: 2026-05-06
Decision: Dividir tests grandes por responsabilidad antes de seguir ampliando el workflow.
Motivo: Los tests de AskDirector, handler general y RequestAgent concentraban escenarios felices, idempotencia, conflictos y seguridad. Mantenerlos juntos aumenta el contexto de debug y va contra la regla de apps y ficheros pequenos.
Alternativas: Dejarlos crecer por debajo de 400 lineas; crear helpers globales nuevos; mezclar refactor de tests con cambios productivos.
Impacto: `director_question_command_*_test.go`, `handler_*_test.go` y `agent_request_*_test.go` quedan separados por flujo normal, conflictos, validacion y seguridad sin modificar contratos ni codigo productivo.
Contratos afectados: sin cambios funcionales; AskDirector, HandleCommandV0 y RequestAgent siguen cubiertos por las mismas pruebas.
Estado: aceptada local en NCW-066
```

```text
Fecha: 2026-05-09
Decision: Modelar artefactos del director fuera de programacion con `RegisterPhaseArtifact` y `PhaseArtifactRegistered`.
Motivo: La prueba real con Codex mostro que los documentos de arquitectura/plan en `brainstorming_arquitectura` no deben forzarse como `DeliveryRegistered`, porque las entregas de codigo pertenecen solo a `programacion`.
Alternativas: Relajar `DeliveryRegistered`; guardar documentos largos dentro del run; dejar los ACK del director fuera del workflow durable.
Impacto: El core proyecta `phase_artifacts` con refs compactas por `artifact_ref`, `phase_id` y `agent_ref`, exige agente arrancado, rechaza `programacion` y registra `CommandEffects` por `artifact_ref`.
Contratos afectados: RegisterPhaseArtifact, PhaseArtifactRegistered, OrchestrationRunV0.PhaseArtifacts, CommandEffects.
Estado: aceptada local en NCW-068
```

```text
Fecha: 2026-05-10
Decision: Tratar `WorkflowTaskV0` como unidad de trabajo de granularidad adaptativa.
Motivo: La regla de microtareas evita macroparches y contextos gigantes, pero no todo trabajo real debe partirse al minimo. Algunas funcionalidades cohesionadas son mas seguras como tarea mediana o grande si tienen contrato, write-set, checkpoints, tests y observabilidad.
Alternativas: Forzar microtareas siempre; renombrar ahora todos los comandos/eventos durables; permitir tareas grandes sin politica de control.
Impacto: `CreateMicrotask` y `MicrotaskCreated` conservan nombre v0 por compatibilidad, pero documentan que crean `WorkflowTaskV0`. El director decide tamano por politica; el core sigue validando contrato compacto, write-set, fase, refs y ausencia de runtime/DB/proveedor/HOME.
Contratos afectados: WorkflowTaskV0, CreateMicrotask, MicrotaskCreated.
Estado: aceptada local en NCW-070
```

```text
Fecha: 2026-05-11
Decision: `WorkflowTaskV0` transporta `required_tests` como contrato compacto
de ejecucion, pero no los proyecta en `MicrotaskCreated`.
Motivo: el agente necesita recibir pruebas obligatorias como `go test ./...`
para que el ACK pueda validarlas. La proyeccion durable de microtarea sigue
compacta para no volver a eventos gigantes; el detalle completo vive en el
store de tareas por puerto.
Impacto: `CreateMicrotask` acepta y normaliza `required_tests`; los bridges de
director lo propagan. El store de tareas rechaza sobrescrituras con el mismo
task_id y contrato distinto para evitar reintentos que cambien write-set,
criterios o tests en silencio.
Contratos afectados: WorkflowTaskV0, CreateMicrotask, WorkflowTaskStorePortV0.
Estado: aceptada local en NCW-071
```

```text
Fecha: 2026-05-13
Decision: Persistir agentes entregados como `delivered_agents`.
Motivo: La prueba real `director -> microtarea -> worker Codex -> ACK -> DeliveryRegistered` mostro que stats no puede depender de inferir el agente desde `delivery_ref`; algunos ACK incluyen el agent id y otros no. El payload de `DeliveryRegistered` ya contiene `agent_ref`, por lo que descartarlo era perdida de informacion.
Impacto: `DeliveryRegistered` proyecta `agent_ref` en `OrchestrationRunV0.DeliveredAgents`; stats usa esa proyeccion para marcar agente `completed`, limpiar progreso/assessment obsoleto y evitar falsos `no_signal`/`stalled`.
Contratos afectados: DeliveryRegistered, OrchestrationRunV0, DirectorRunStatsV0.
Estado: aceptada local en NCW-072
```

```text
Fecha: 2026-05-13
Decision: Persistir motivo compacto de parada de agente como `agent_stop_requests`.
Motivo: El director y la web necesitan saber por que se pidio parar un agente sin reconstruir payloads de eventos ni inferirlo de contadores. Esa informacion es necesaria para decidir reintentos, replanificacion o cierre controlado.
Alternativas: Leer siempre el event log completo; guardar resumen largo en el run; inferir motivo desde assessments. Leer el ledger acopla stats a historia interna, el resumen largo rompe la regla de contexto compacto y la inferencia pierde la razon exacta de `StopAgent`.
Impacto: `AgentStopRequested` proyecta `agent_request_id#reason:<reason_code>` en `OrchestrationRunV0.AgentStopRequests`; `DirectorRunStatsV0` expone `stop_reason_code/source/ref`. `#` queda prohibido en `agent_request_id` y `reason_code` de parada para mantener la proyeccion parseable.
Contratos afectados: StopAgent, AgentStopRequested, OrchestrationRunV0, DirectorRunStatsV0.
Estado: aceptada local en NCW-073
```

```text
Fecha: 2026-05-15
Decision: `timeout` es veredicto propio para parar agentes sin actividad reciente.
Motivo: reutilizar `loop_detected` o `garbage` para un agente excedido en tiempo
pero sin evidencia de bucle o mala calidad mezcla causas distintas y dificulta
que el director tome decisiones. El caso real OPES mostro que un `stalled`
previo no debe impedir una parada posterior cuando el presupuesto indica
`over_budget_no_activity`.
Impacto: `AssessAgentWork action=stop_agent` acepta `verdict=timeout`; el
outbox `StopRuntimeAgent` transporta ese reason_code compacto. El core no
decide proveedor, modelo, HOME, runtime ni DB: solo valida y persiste la causa
durable.
Contratos afectados: AssessAgentWork, AgentWorkAssessed, AgentStopRequested,
StopRuntimeAgentRequestV0.
Estado: aceptada local en NCW-074
```

```text
Fecha: 2026-05-22
Decision: Crear `WorkProfileV0` dentro de `orquesta-core-workflow` como fabrica neutral de `WorkflowTaskV0`.
Motivo: programacion, refactor, estudio de codigo, documentacion, revision y trabajo de dominio son perfiles de trabajo del nucleo, no detalles de OPES ni del adaptador Codex. Ya existia `WorkflowTaskV0` como unidad durable, por lo que crear otro modulo habria duplicado la rueda.
Alternativas: meter reglas de perfil en el stack Codex; crear `orquesta-work-profiles`; codificar heuristicas en texto del director. Todas mezclan responsabilidades o dificultan validacion focal.
Impacto: `WorkProfileV0` normaliza alias, aplica fase/criterios base, exige contratos de funcion, exige pruebas en perfiles de ejecucion y conserva linaje neutral antes de producir `WorkflowTaskV0`.
Contratos afectados: WorkProfileV0, WorkflowTaskFromWorkProfileV0, WorkflowTaskV0.
Estado: aceptada local en NCW-075
```

```text
Fecha: 2026-05-22
Decision: `WorkflowTaskV0` puede transportar `work_profile_kind` como metadata neutral opcional.
Motivo: el scheduler necesita derivar rol y capacidad sin inspeccionar textos ni conocer Codex, OPES o conectores. El perfil pertenece al mismo catalogo puro de `WorkProfileV0` y es compatible con tareas antiguas sin campo.
Impacto: `WorkflowTaskFromWorkProfileV0` rellena `work_profile_kind`; `NewWorkflowTaskV0` rechaza perfiles no soportados. Las tareas legacy sin perfil siguen validando y se resuelven por fase en la capa de orquestacion.
Contratos afectados: WorkflowTaskV0, WorkProfileV0, WorkflowTaskFromWorkProfileV0.
Estado: aceptada local en NCW-076
```
