# Contratos locales: orquesta-core-workflow

Estos contratos son locales hasta que el director los promueva a `../../CONTRATOS.md`.

## Plantilla

```text
Nombre:
Tipo: dto | comando | evento | outbox | error | puerto
Version:
Propietario:
Consumidores:
Campos:
Invariantes:
Errores:
Pruebas de contrato:
Estado:
```

## Estado y Fases

Detalle: `docs/contratos_estado_fases.md`.

## `OrchestrationCommandV0`

```text
Nombre: OrchestrationCommandV0
Tipo: comando
Version: v0
Propietario: orquesta-core-workflow
Consumidores: adaptadores inbound futuros
Campos comunes:
  - command_id
  - command_type
  - run_id
  - idempotency_key
  - correlation_id
  - requested_by
  - occurred_at
  - payload_version
  - payload
Comandos candidatos:
  - StartRun
  - OpenPhase
  - RequestBrainstorm
  - RequestVote
  - AcceptDecision
  - CreateMicrotask
  - PublishFunctionContract
  - RequestCapacity
  - RequestAgent
  - StopAgent
  - AssessAgentWork
  - RegisterAgentLeaseExpired
  - RecordConcurrencyGate
  - RecordQualityGate
  - RegisterPhaseArtifact
  - RegisterDelivery
  - RegisterAgentStopConfirmed
  - RequestReview
  - RecordReviewResult
  - AcceptReview
  - RequestRework
  - RecordReplanDecision
  - CloseTask
  - RegisterFinalValidation
  - CloseRun
  - ClosePhase
  - BlockRun
  - AskDirector
  - AnswerDirectorQuestion
Invariantes:
  - Todo comando mutador requiere `idempotency_key`.
  - El handler no ejecuta efectos externos.
  - El dispatch interno de `HandleCommandV0` se declara en `command_router_v0.go` y debe estar sincronizado con el catalogo publico.
  - Un comando repetido con misma `idempotency_key` no duplica eventos efectivos.
Errores:
  - comando_no_soportado
  - comando_invalido
  - idempotency_key_requerida
  - transicion_invalida
  - payload_invalido
Notas:
  - NCW-003 implementa `StartRun`, `OpenPhase` y `BlockRun`; NCW-064 protege `RunStarted` y `RunBlocked`, y NCW-065 protege `PhaseOpened` con identidad fuerte por evento de apertura.
  - NCW-004 implementa `AskDirector` en el catalogo generico: `ValidateOrchestrationCommandV0` y `HandleCommandV0` lo aceptan.
  - NCW-036 implementa `AnswerDirectorQuestion`: registra respuesta compacta del director, sin outbox, y puede desbloquear `director-question-<question_id>`; NCW-063 lo protege con identidad fuerte por `answer_id`.
  - NCW-009 implementa `CreateMicrotask` con payload `WorkflowTaskV0` validado y evento durable compacto.
  - NCW-010 implementa `RequestCapacity`; ver `docs/contratos_capacidad.md`.
  - NCW-011 implementa `RequestAgent`; ver `docs/contratos_agentes.md`.
  - NCW-012..NCW-025 implementan fases operativas; ver contratos locales.
  - NCW-028 implementa `StopAgent`; ver `docs/contratos_agentes.md`.
  - NCW-029 implementa `AssessAgentWork`; ver `docs/contratos_agentes.md`.
  - NCW-042..NCW-045 implementan resultado de revision, rework y decision de replan; ver `docs/contratos_revisiones.md`.
  - NCW-046 implementa `RegisterAgentLeaseExpired`; ver `docs/contratos_agentes.md`.
  - NCW-047 implementa `RecordConcurrencyGate`; ver `docs/contratos_concurrencia.md`.
  - NCW-067 implementa `RecordQualityGate`; ver `docs/contratos_quality_gates.md`.
  - NCW-068 implementa `RegisterPhaseArtifact`; ver `docs/contratos_artefactos_fase.md`.
  - NCW-049 implementa `RegisterAgentStopConfirmed`; ver `docs/contratos_agentes.md`.
  - NCW-050/NCW-051 endurecen `RegisterDelivery`: exige agente arrancado y rechaza agentes parados o fallidos.
  - NCW-052 endurece lifecycle de agente: start/fail/stop no pueden contradecirse.
  - NCW-053 endurece `StopAgent`: no emite parada si el lanzamiento ya fallo.
  - NCW-054 permite `RecordReplanDecision` desde `AgentFailed` en `programacion`.
  - NCW-069 permite `RecordReplanDecision` desde `AgentWorkAssessed` en `programacion`, para reemplazar trabajo parado por bucle/basura sin inventar `ReworkRequested`.
  - La idempotencia de efectos v0 usa `OrchestrationRunV0.CommandEffects` para comparar subject ref, command/event id, idempotency key y hash de payload antes de reconstruir outbox o aceptar proyecciones criticas ya reflejadas.
  - Si el estado refleja una ref de efecto pero el retry no coincide con la huella durable, el handler devuelve `transicion_invalida` y no emite eventos/outbox.
Estado: implementado inicial en NCW-003; extensiones durables hasta NCW-069.
```

## `CommandEffects`

```text
Nombre: CommandEffects
Tipo: proyeccion durable compacta
Version: v0 candidato local
Propietario: orquesta-core-workflow
Consumidores: handlers/reducers del workflow
Campos por registro:
  - event_type
  - subject_ref
  - idempotency_key
  - causation_id opcional si el evento durable original no lo aporta
  - event_id
  - payload_hash
Invariantes:
  - Una pareja `(event_type, subject_ref)` aparece como maximo una vez.
  - Se registra para efectos que pueden emitir/reconstruir outbox: CapacityRequested, AgentRequested, AgentStopRequested, AgentWorkAssessed y DirectorQuestionRaised.
  - Se registra tambien para proyecciones de lifecycle/planificacion/revision/cierre/programacion/supervision que no deben ocultar payloads distintos: RunStarted, PhaseOpened, RunBlocked, PhaseClosed, BrainstormRequested, VoteRequested, ArchitectureDecisionAccepted, FunctionContractPublished, MicrotaskCreated, ReviewRequested, ReviewAccepted, ReviewResultRecorded, ReworkRequested, ReplanDecisionRecorded, TaskClosed, FinalValidationRegistered, RunClosed, CapacityDecided, AgentStarted, AgentFailed, AgentStopConfirmed, DeliveryRegistered, ConcurrencyGateRecorded, QualityGateRecorded, AgentLeaseExpired y DirectorQuestionAnswered.
  - Un replay exacto conserva la huella; un evento posterior con la misma ref y otra key, event_id, command_id o payload produce `evento_conflictivo`.
  - Un retry de comando con la misma ref debe coincidir con la huella o produce `transicion_invalida`.
  - La huella no guarda payload completo, DB, runtime, proveedor, HOME, OAuth, rutas ni secretos.
Estado: implementado local en NCW-055; extendido a planificacion en NCW-057, a brainstorming/votacion en NCW-058, a revision en NCW-059, a resultado/rework/replan en NCW-060, a cierre en NCW-061, a programacion operativa en NCW-062, a supervision/gobierno en NCW-063, a lifecycle de run/fase en NCW-064, a apertura de fase en NCW-065 y a quality gates en NCW-067.
```

## `OrchestrationCommandResultV0`

```text
Nombre: OrchestrationCommandResultV0
Tipo: dto
Version: v0
Propietario: orquesta-core-workflow
Consumidores: adaptadores inbound futuros, persistence/outbox futura
Campos:
  - events
  - outbox
  - idempotent
  - noop_reason
Invariantes:
  - `HandleCommandV0` es puro y no ejecuta efectos externos.
  - Los comandos `StartRun`, `OpenPhase` y `BlockRun` devuelven eventos de dominio y outbox vacio en NCW-003.
  - `HandleCommandV0` enruta `AskDirector`; devuelve `DirectorQuestionRaised`, outbox `SendDirectorQuestion` y `RunBlocked` cuando `blocking=true`.
  - `HandleCommandV0` enruta `AnswerDirectorQuestion`; devuelve `DirectorQuestionAnswered`, outbox vacio y, si el blocker activo es `director-question-<question_id>`, desbloquea el run con transicion durable explicita.
  - `HandleCommandV0` enruta brainstorming, votacion y decision aceptada sin outbox.
  - `HandleCommandV0` enruta `CreateMicrotask`; devuelve `MicrotaskCreated` y outbox vacio.
  - `HandleCommandV0` enruta `RequestCapacity`; devuelve `CapacityRequested` y outbox `RequestCapacityDecision`.
  - `HandleCommandV0` enruta `StopAgent`; devuelve `AgentStopRequested` y outbox `StopRuntimeAgent`.
  - `HandleCommandV0` enruta `RegisterAgentStopConfirmed`; devuelve `AgentStopConfirmed` y outbox vacio.
  - `HandleCommandV0` enruta `AssessAgentWork`; devuelve `AgentWorkAssessed` y solo emite parada logica cuando `action=stop_agent`.
  - `HandleCommandV0` enruta `RecordQualityGate`; devuelve `QualityGateRecorded` y outbox vacio.
  - Un no-op idempotente devuelve `events` y `outbox` vacios.
Errores:
  - Propaga errores publicos de comando, fase o evento.
Pruebas de contrato:
  - `StartRun` produce `RunStarted`.
  - `OpenPhase` produce `PhaseOpened` para fases soportadas.
  - `BlockRun` produce `RunBlocked`.
  - `AskDirector` produce pregunta compacta para el director y bloqueo opcional.
  - `AnswerDirectorQuestion` produce respuesta compacta, no emite outbox y puede desbloquear solo el blocker `director-question-<question_id>`.
  - Las fases de arquitectura producen eventos compactos y no duplican refs ya reflejadas.
  - `CreateMicrotask` produce `MicrotaskCreated` y repetir tarea ya reflejada no duplica.
  - `RequestCapacity` produce `CapacityRequested`, outbox target `capacity` y repetir request ya reflejada no duplica.
  - `RequestAgent` produce `AgentRequested`, outbox `LaunchRuntimeAgent` target `agent_launcher` y repetir request ya reflejada no duplica.
  - `StopAgent` produce `AgentStopRequested`, outbox `StopRuntimeAgent` target `agent_launcher`, exige agente ya solicitado, rechaza agentes fallidos y repetir stop ya reflejado no duplica.
  - `RegisterAgentStopConfirmed` produce `AgentStopConfirmed`, no emite outbox, exige parada solicitada previa y repetir confirmacion ya reflejada no duplica.
  - `RegisterAgentStarted` y `RegisterAgentFailed` rechazan lifecycle contradictorio con failed/started/stopped ya proyectado.
  - `AssessAgentWork` produce `AgentWorkAssessed`, exige agente ya solicitado, valida `delivery_ref` si existe y repetir assessment ya reflejado no duplica.
  - `RecordQualityGate` produce `QualityGateRecorded`, no emite outbox, exige fase actual activa y proyecta `quality_gates`.
  - `RegisterDelivery` exige `AgentStarted` previo y rechaza agentes fallidos o con parada solicitada.
  - `CloseTask` produce `TaskClosed`, no emite outbox y no cierra fase ni run.
  - `RegisterFinalValidation` debe producir `FinalValidationRegistered`, no emitir outbox y no cerrar fase ni run.
  - `CloseRun` debe producir `RunClosed`, no emitir outbox, exigir validacion ya proyectada y no cerrar fase automaticamente.
  - Comando desconocido devuelve `comando_no_soportado`.
  - Repetir comando ya reflejado no duplica eventos.
Estado: implementado inicial en NCW-003; extensiones durables hasta NCW-067.
```

## `StartRunFromAppSpecV0`

```text
Nombre: StartRunFromAppSpecV0
Tipo: comando
Version: v0 candidato local
Propietario: orquesta-core-workflow
Consumidores: adaptadores inbound futuros; puente global futuro desde AppSpec/ProyectoPlanBorrador
Campos de entrada:
  - run_id
  - project_ref
  - app_spec_ref
  - requested_by
  - correlation_id
  - idempotency_key
  - occurred_at
Salida:
  - OrchestrationCommandV0 con command_type `StartRun`
Invariantes:
  - Es un puente local compacto, no un contrato global promovido.
  - No importa `orquesta-core`, `orquesta-factory` ni otros modulos.
  - Solo transporta refs opacas y metadatos necesarios para construir `StartRun`.
  - Rechaza `run_id`, `project_ref`, `app_spec_ref`, `idempotency_key` y `occurred_at` vacios.
  - Rechaza detalles de DB, runtime, proveedor, HOME, credenciales y adaptadores concretos.
  - El `command_id` derivado es determinista para el mismo `run_id`.
Errores:
  - appspec_run_draft_invalido
  - idempotency_key_requerida
  - detalle_prohibido
Pruebas de contrato:
  - Mapping determinista desde draft a `StartRun`.
  - Refs vacios se rechazan con error publico.
  - Serializacion no contiene DB/runtime/proveedor/HOME.
  - El comando resultante valida como `StartRun` y el handler puro lo acepta.
Estado: implementado como candidato local en NCW-006; puente real pendiente de contrato global futuro.
```

## `WorkflowTaskV0`

```text
Nombre: WorkflowTaskV0
Tipo: dto
Version: v0 candidato local
Propietario: orquesta-core-workflow
Consumidores: core-workflow, planificador de microtareas futuro, observability como proyeccion
Detalle: ver `docs/contratos_microtareas.md`.
Estado: implementado local en NCW-008.
```

## `OrchestrationEventV0`

```text
Nombre: OrchestrationEventV0
Tipo: evento
Version: v0
Propietario: orquesta-core-workflow
Consumidores: reducer, persistence futura, observability futura
Campos comunes:
  - event_id
  - event_type
  - run_id
  - sequence
  - idempotency_key
  - correlation_id
  - causation_id
  - occurred_at
  - payload_version
  - payload
Eventos implementados:
  - RunStarted
  - PhaseOpened
  - RunBlocked
  - BrainstormRequested
  - VoteRequested
  - ArchitectureDecisionAccepted
  - MicrotaskCreated
  - FunctionContractPublished
  - CapacityRequested
  - CapacityDecided
  - AgentRequested
  - AgentStarted
  - AgentFailed
  - AgentLeaseExpired
  - AgentStopRequested
  - AgentStopConfirmed
  - AgentWorkAssessed
  - ConcurrencyGateRecorded
  - QualityGateRecorded
  - DeliveryRegistered
  - ReviewRequested
  - ReviewResultRecorded
  - ReviewAccepted
  - ReworkRequested
  - ReplanDecisionRecorded
  - TaskClosed
  - FinalValidationRegistered
  - RunClosed
  - PhaseClosed
  - DirectorQuestionRaised
  - DirectorQuestionAnswered
Invariantes:
  - Evento inmutable.
  - `sequence` crece dentro de un run.
  - No contiene secretos ni transcripts completos.
  - `Apply` debe ser determinista.
  - El dispatch interno de `ApplyEventV0` se declara en `event_router_v0.go` y debe estar sincronizado con el catalogo publico.
  - `RunStarted` proyecta el run inicial y registra huella `CommandEffects` por `run_id`.
  - `PhaseOpened` registra huella `CommandEffects` por `event_id`, no por `phase_id`, para permitir reaperturas reales como eventos nuevos.
  - `RunBlocked` proyecta `blocker_id`, registra huella `CommandEffects` y no emite outbox por si mismo.
  - `PhaseClosed` cierra la fase actual, registra huella `CommandEffects` por `closure_ref`, no cierra el run y permite retry exacto aunque el run haya avanzado a otra fase.
  - `DirectorQuestionRaised` esta en el catalogo durable v0, proyecta una ref compacta en `director_questions` y registra huella `CommandEffects` por `question_id`.
  - `DirectorQuestionAnswered` proyecta respuesta compacta del director por `answer_id`, tambien `question_id` en `director_answered_questions`, y registra huella `CommandEffects`; no transporta secretos, provider/proveedor, HOME, DB, runtime, prompts ni transcripts.
  - `CapacityDecided` proyecta una decision compacta por `capacity_request_id`, registra huella `CommandEffects` y habilita `RequestAgent`; no transporta proveedor/modelo/HOME/cuotas reales.
  - Eventos de arquitectura proyectan refs compactas en `brainstorms`, `votes` y `decisions`.
  - `MicrotaskCreated` proyecta `task_id` en `tasks` y refs compactas de contratos de funcion en `function_contracts`.
  - `DeliveryRegistered` proyecta `delivery_ref` en `deliveries`, `task_id` en `delivered_tasks` y `agent_ref` en `delivered_agents`; registra huella `CommandEffects`.
  - `DeliveryRegistered` exige agente arrancado y rechaza agentes fallidos o con parada solicitada.
  - `ReviewRequested` proyecta `review_request_id` en `reviews` y registra huella `CommandEffects`.
  - `ReviewAccepted` proyecta `accepted_review_ref` en `accepted_reviews` y registra huella `CommandEffects`.
  - `TaskClosed` proyecta `task_id` en `closed_tasks` y registra huella `CommandEffects`.
  - `FinalValidationRegistered` proyecta `validation_ref` en `validations` y registra huella `CommandEffects`.
  - `RunClosed` proyecta `closure_ref` en `closures`, registra huella `CommandEffects`, marca el run cerrado y no modifica la fase actual.
  - `CapacityRequested` proyecta `capacity_request_id` en `capacity_requests` sin duplicar y registra huella `CommandEffects`.
  - `AgentRequested` proyecta solicitudes en `agents` y registra huella `CommandEffects`; no significa agente arrancado.
  - `AgentStarted` proyecta `agent_request_id` en `started_agents` con refs opacas de launch/ack/readiness y registra huella `CommandEffects`.
  - `AgentFailed` proyecta `agent_request_id` en `failed_agents` con causa compacta, retryable y huella `CommandEffects`.
  - `AgentStarted` rechaza failed/stopped previos; `AgentFailed` rechaza started/stopped previos.
  - `AgentStopRequested` rechaza failed previo.
  - `AgentLeaseExpired` proyecta expiracion observada en `agent_lease_expirations`, registra huella `CommandEffects` y no emite outbox ni efectos operativos.
  - `AgentStopRequested` proyecta `agent_request_id` en `stopped_agents` sin duplicar, exige agente ya proyectado y registra huella `CommandEffects`.
  - `AgentStopConfirmed` proyecta `agent_request_id` en `confirmed_stopped_agents` sin duplicar, exige parada solicitada previa y registra huella `CommandEffects`.
  - `ConcurrencyGateRecorded` proyecta un gate evaluado en `concurrency_gates`, registra huella `CommandEffects` y no crea scheduler ni outbox.
  - `QualityGateRecorded` proyecta un gate de calidad en `quality_gates`, registra huella `CommandEffects` y no crea cierre, rework, bloqueo ni outbox.
  - `ReviewResultRecorded`, `ReworkRequested` y `ReplanDecisionRecorded` proyectan refs compactas, registran huella `CommandEffects` y no materializan efectos por si mismos.
  - `ReplanDecisionRecorded` acepta fuente `ReworkRequested` en `revision`, o `AgentFailed`, `AgentWorkAssessed` y quality gate bloqueante en `programacion`.
  - Si la pregunta bloquea, el bloqueo durable se refleja como evento separado `RunBlocked`.
  - Una respuesta del director puede desbloquear solo el blocker `director-question-<question_id>` asociado; blockers de otro origen no se limpian implicitamente.
  - `ReplayEventsV0` y `ReplayDurableEventsV0` validan historia durable: primer evento `RunStarted`, sequence desde 1, run_id constante y duplicados solo si tienen la misma huella durable.
Errores:
  - evento_no_soportado
  - secuencia_invalida
  - payload_invalido
  - evento_conflictivo
Estado: implementado inicial en NCW-002, reducer NCW-002R y replay/idempotencia NCW-005; extensiones implementadas hasta NCW-067.
```

## Microtareas

`WorkflowTaskV0`, `CreateMicrotask` y `MicrotaskCreated`: `docs/contratos_microtareas.md`.
`PublishFunctionContract` y `FunctionContractPublished`: `docs/contratos_funcion.md`.

## Fases operativas

Detalle: `docs/contratos_brainstorm.md`, `docs/contratos_votaciones.md` y `docs/contratos_decisiones.md`.
Detalle gates: `docs/contratos_concurrencia.md` y `docs/contratos_quality_gates.md`.
Detalle entregas: `docs/contratos_entregas.md`.
Detalle revisiones: `docs/contratos_revisiones.md`.
Detalle cierre de tareas: `docs/contratos_cierre_tareas.md`.
Detalle validacion final: `docs/contratos_validacion_final.md`.
Detalle cierre de run: `docs/contratos_cierre_run.md`.

## Capacidad

`RequestCapacity`, `CapacityRequested` y `CapacityDecisionRequestV0` estan
detallados en `docs/contratos_capacidad.md`.

## `OutboxMessageV0`

```text
Nombre: OutboxMessageV0
Tipo: outbox
Version: v0
Propietario: orquesta-core-workflow
Consumidores: dispatchers/adaptadores futuros
Campos:
  - message_id
  - message_type
  - run_id
  - idempotency_key
  - correlation_id
  - causation_event_id
  - target_port
  - payload_version
  - payload
Tipos candidatos:
  - PersistRunEvents
  - PublishOrquestaEvent
  - RequestCapacityDecision
  - LaunchRuntimeAgent
  - StopRuntimeAgent
  - RequestDeployPlan
  - SendDirectorQuestion
Invariantes:
  - No se ejecuta dentro del handler.
  - Debe ser idempotente para reintento.
  - No conoce adaptador concreto.
  - `message_type` pertenece al catalogo v0 cerrado.
  - `target_port` es un puerto logico cerrado y coherente con `message_type`.
  - Los puertos logicos no usan nombres de adaptadores ni palabras reservadas de infraestructura; `LaunchRuntimeAgent` y `StopRuntimeAgent` apuntan a `agent_launcher`.
  - `payload` es JSON compacto y no contiene secretos, adaptadores concretos ni contexto masivo.
Errores:
  - outbox_invalido
  - outbox_tipo_no_soportado
  - payload_invalido
  - detalle_prohibido
Pruebas de contrato:
  - `SendDirectorQuestion` serializa sin adaptadores ni secretos.
  - Tipo desconocido devuelve `outbox_tipo_no_soportado`.
  - Payload con contexto masivo se rechaza.
  - `AskDirector` emite outbox con target `director`.
  - `RequestAgent` emite `LaunchRuntimeAgent` con target `agent_launcher`.
  - `StopAgent` emite `StopRuntimeAgent` con target `agent_launcher`.
Estado: implementado en NCW-004; extensiones de agentes hasta NCW-028.
```

## `DirectorQuestionV0`

```text
Nombre: DirectorQuestionV0
Tipo: dto
Version: v0
Propietario: orquesta-core-workflow
Consumidores: director, MCP/web/CLI futuro
Campos:
  - question_id
  - run_id
  - source_group
  - target_group
  - summary
  - options
  - evidence_refs
  - blocking
  - requested_at
Invariantes:
  - La pregunta debe ser concreta.
  - No adjunta contexto masivo.
  - `source_group` y `summary` son obligatorios.
  - `target_group` por defecto es `director` si el constructor recibe blanco.
  - `options` y `evidence_refs` son listas compactas.
  - Si bloquea una transicion, el run queda `blocked`.
Errores:
  - director_question_invalida
Pruebas de contrato:
  - Pregunta sin `summary` se rechaza.
  - Pregunta sin `source_group` se rechaza.
  - Payload `SendDirectorQuestion` no incluye contexto masivo.
  - `AskDirector` construye `DirectorQuestionV0` compacta y no ejecuta efectos externos.
Estado: implementado en NCW-004.
```

## `AnswerDirectorQuestion` / `DirectorQuestionAnswered`

```text
Nombre: AnswerDirectorQuestion / DirectorQuestionAnswered
Tipo: command_event_pair
Version: v0 candidato local
Propietario: orquesta-core-workflow
Consumidores: director, reducer, observability futura
Campos:
  - answer_id
  - question_id
  - decision
  - summary
  - evidence_refs
  - unblocks
  - blocker_id (solo en evento, si unblocks=true)
Invariantes:
  - La respuesta es compacta y accionable.
  - `answer_id`, `question_id`, `decision` y `summary` son obligatorios.
  - `evidence_refs` son refs opacas.
  - `decision` pertenece a `continue`, `replan` o `stop_agent`.
  - No contiene secretos, provider/proveedor, HOME, DB, runtime, prompts, transcripts ni contexto masivo.
  - `AnswerDirectorQuestion` no emite outbox.
  - La identidad fuerte se registra por `answer_id` en `CommandEffects`.
  - Repetir la misma respuesta ya reflejada solo es no-op si coinciden command_id, idempotency_key y payload normalizado.
  - Si `unblocks=true`, el evento debe incluir `blocker_id=director-question-<question_id>`.
  - Solo se elimina el blocker asociado; no desbloquea blockers no relacionados.
Errores:
  - transicion_invalida
  - payload_invalido
  - detalle_prohibido
Pruebas de contrato:
  - Respuesta valida produce `DirectorQuestionAnswered` y outbox vacio.
  - Refs opacas se conservan sin resolver adaptadores.
  - Payload con provider/HOME/DB/runtime/secretos se rechaza.
  - Bloqueo `director-question-<question_id>` queda desbloqueado; otro blocker permanece.
Estado: implementado en NCW-036; identidad fuerte extendida en NCW-063.
```
