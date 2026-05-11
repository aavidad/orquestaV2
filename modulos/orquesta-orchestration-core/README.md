# orquesta-orchestration-core

Capa de aplicacion para activar el nucleo limpio de Orquesta sin que REST, MCP,
DB, runtime ni proveedor entren en el dominio. El paquete coordina el loop del
director por puertos y deja todo efecto externo detras de conectores.

## Objetivo

Ejecutar una rafaga progresiva del director con frontera hexagonal:

- construir `DirectorCycleStepInputV0` desde un run y candidatos compactos;
- aplicar comandos del workflow y persistir el run por `RunStorePortV0`;
- registrar eventos por `EventSinkPortV0`;
- registrar y despachar outbox por puertos de reader, claim, executor y ACK;
- avanzar `RequestCapacity -> CapacityDecided -> RequestAgent`;
- entregar `LaunchRuntimeAgent` a un `AgentLauncherPortV0` inyectado;
- resolver `RuntimeLaunchRequestV0`, contexto materializado y
  `ExternalAgentLaunchSpecV0` solo mediante puertos autorizados;
- registrar `AgentStarted` cuando el launcher devuelve evidencia;
- validar readiness por `AgentReadinessProbePortV0` si el conector lo inyecta;
- aceptar observaciones compactas de receipt por `AgentDeliveryObservationProviderPortV0`
  y convertirlas en `RegisterDelivery` o `RegisterPhaseArtifact` segun fase;
- aceptar observaciones compactas de review gate por
  `ReviewGateObservationProviderPortV0` y convertirlas en `RequestReview`,
  `RecordReviewResult`, `AcceptReview` o `RequestRework` segun resultado;
- entregar `StopRuntimeAgent` a un `AgentStopperPortV0` inyectado;
- registrar `AgentStopConfirmed` cuando el stopper devuelve evidencia;
- aceptar evaluaciones compactas de lease por puerto externo y convertir
  timeouts en `LeaseActionCandidates` sin reloj, runtime ni DB en el nucleo;
- registrar replan desde `AgentWorkAssessed` ya proyectado y materializar
  followups explicitos de reemplazo por capacidad/agente sin comandos manuales;
- registrar replan desde `RequestRework` ya proyectado y materializar
  followups explicitos para volver a `programacion` cuando una revision pide cambios;
- parar por decision del supervisor, outbox no manejado, quietud, error o limite.

## Contrato

El nucleo no arranca procesos por si mismo. El contrato operativo de lanzamiento
es:

1. el workflow emite outbox `LaunchRuntimeAgent`;
2. `orquesta-outbox-dispatch` reclama el mensaje y construye un intent;
3. `AgentLauncherExecutorV0` valida `AgentLauncherInboundV0`;
4. el conector inyectado implementa `AgentLauncherPortV0`;
5. `ExternalAgentLaunchSpecResolverV0` puede enriquecer la orden compacta con
   dependencias resueltas, contexto materializado y perfil de conector por
   puertos;
6. el launcher puede esperar readiness por `AgentReadinessProbePortV0`;
7. el executor registra `AgentStarted` con referencias devueltas por el
   conector.

El contrato operativo de parada es:

1. el workflow emite outbox `StopRuntimeAgent`;
2. `AgentStopperExecutorV0` valida `AgentStopperInboundV0`;
3. el conector inyectado implementa `AgentStopperPortV0`;
4. el conector resuelve fuera del nucleo el mapeo estructural
   `agent_request_id -> process_ref + session_ref`;
5. el conector ejecuta la parada real por `process_ref` si gestiona un proceso;
6. el executor registra `AgentStopConfirmed` solo cuando el stopper devuelve
   evidencia aceptada.

`agent_request_id`, `process_ref` y `session_ref` son referencias opacas. El
nucleo no deriva `process_ref` desde evidencias, transcripts, rutas, PID ni
detalles del proveedor. La persistencia del mapeo pertenece a un conector o
registro inyectado; si ese conector usa DB, broker o proveedor concreto,
tambien queda fuera del nucleo. Regla operativa: Orquesta solo puede parar
procesos/sesiones que haya arrancado y registrado por `run_id + agent_request_id`.
No se permite barrer procesos por nombre de comando porque podria cortar un
director, una sesion humana o un agente de otro run.

El contrato operativo de receipt de agente es:

1. el agente externo produce un receipt compacto validado por su conector;
2. el conector lo expone como `AgentDeliveryObservationV0`, sin paths, logs,
   transcripts, modelo, HOME, DB ni proveedor;
3. `DeliveryCandidateProviderV0` genera:
   - `SchedulableDeliveryCandidateV0` si la fase es `programacion`;
   - `SchedulablePhaseArtifactCandidateV0` si la fase no es `programacion`;
4. el scheduler exige agente existente, started_agent y fase actual;
5. el workflow registra `DeliveryRegistered` para programacion o
   `PhaseArtifactRegistered` para brainstorming/documentacion/revision/etc.

El contrato operativo de review gate es:

1. un adaptador externo evalua una entrega ya registrada y devuelve una
   `ReviewGateObservationV0` compacta;
2. `ReviewGateCandidateProviderV0` solo actua en fase `revision`;
3. el provider genera un `SchedulableReviewGateCandidateV0` con
   `RequestReview` y `RecordReviewResult`;
4. si el status es `accepted`, anade `AcceptReview`;
5. si el status es `changes_requested` o `rejected`, anade `RequestRework`;
6. el scheduler materializa un comando por tick segun estado durable observado;
7. el workflow deduplica con `Reviews`, `ReviewResults`, `AcceptedReviews` y
   `ReworkRequests`.

El nucleo no lee tests, ficheros, ACKs, transcripts, runtime ni proveedor para
decidir la revision. Esas evidencias llegan ya compactadas por puerto.

El contrato operativo de replan tras retrabajo de revision es:

1. `RequestRework` queda durable en `OrchestrationRunV0.ReworkRequests`;
2. un puerto `ReviewReworkReplanPlanProviderPortV0` aporta un plan compacto;
3. `ReviewReworkReplanCandidateProviderV0` valida que el retrabajo, la tarea y
   el resultado de revision ya existen en el run;
4. la decision de replan usa `SourceRef = rework_request_ref`, no el resultado
   de revision;
5. si hay que repetir programacion, el candidate incluye `OpenPhase(programacion)`
   antes de pedir nueva capacidad;
6. si el plan es `split_task`, aporta `WorkflowTaskV0` compactos, el provider
   guarda su detalle por `WorkflowTaskWriterPortV0` y emite `CreateMicrotask`
   como followup antes de planificarlos;
7. tras `CapacityDecided`, el mismo provider puede seguir en `programacion` solo
   si el replan ya esta registrado, para emitir el agente de seguimiento.

El provider no inventa proveedor, modelo, runtime ni DB. Si el plan requiere
split, las microtareas deben venir por puerto como DTOs compactos; si requiere
ask, debe traer la pregunta al director como candidate explicito.

El contrato operativo de microtarea programable es:

1. `OrchestrationRunV0.Tasks` contiene solo `task_id` aceptados por el
   workflow;
2. el detalle completo `WorkflowTaskV0` vive fuera del core en
   `WorkflowTaskStorePortV0`;
3. `WorkflowTaskCandidateProviderV0` carga solo las tareas autorizadas por el
   run, descarta cerradas y exige fase `programacion`;
4. cada microtarea se convierte en `SchedulableWorkCandidateV0` mediante
   `orquesta-director-candidates`, no con structs fabricados a mano;
5. el scheduler decide `RequestCapacity -> RequestAgent` sin conocer DB,
   runtime, proveedor, modelo, HOME ni OAuth.

La capacidad real de lanzamiento por proceso queda, por diseno, fuera del
nucleo: debe vivir en un conector de runtime configurado por el operador. Ese
conector puede hablar con procesos, tmux, contenedores, servicios remotos u otro
runtime, pero no puede hardcodear Codex, Claude, Gemini, HOME, OAuth, tokens,
cuotas ni una DB concreta dentro de este paquete. Ollama/vLLM/modelos locales
deben seguir siendo posibles por contrato futuro, pero no son alcance operativo
actual.

## Que cierra

- Loop de aplicacion `RunProgressiveLoopV0` sin pasos internos codificados en
  REST/MCP.
- Puertos para store, eventos, candidatos, outbox, capacidad y launcher.
- Dispatch de capacidad por puerto con claim/ACK.
- Planificacion de batch de outbox para reclamar varias ordenes por target y
  dejarlas listas para ejecucion paralela por adaptador externo.
- Dispatch de batch por puerto externo: el core reclama intents, llama a un
  `OutboxDispatchBatchExecutorPortV0` y cierra ACK solo para success
  correlacionado.
- Transicion validada en memoria `RequestCapacity -> CapacityDecided ->
  RequestAgent`.
- Entrega de `LaunchRuntimeAgent` a un launcher inyectado y registro de
  `AgentStarted` con `FakeLifecycleAgentLauncherV0`.
- Lanzamiento real minimo de proceso mediante `ExternalProcessAgentLauncherV0`,
  siempre con spec, resolver y runtime inyectados por puerto.
- Resolucion generica `LaunchRuntimeAgent -> RuntimeLaunchRequest ->
  AgentStartPacket -> ExternalAgentLaunchSpec` mediante
  `ExternalAgentLaunchSpecResolverV0`, sin `cmd`, `db` ni proveedor.
- Composicion de dependencias de launcher por puertos separados:
  FunctionContract, CapacityDecision, RuntimeBinding, EvidenceRefs y
  ContextBundle.
- Readiness por puerto opcional antes de registrar `AgentStarted`.
- Entrega de `StopRuntimeAgent` a stopper inyectado y registro de
  `AgentStopConfirmed` con fake lifecycle compartido.
- Registro estructural por puerto para mantener
  `run_id + agent_request_id -> process_ref + session_ref` con refs opacas.
- Stop real minimo de proceso por `process_ref` mediante stopper y runtime
  inyectados, con ACK solo tras resultado aceptado.
- Cortes anti-bucle: outbox pendiente, outbox no manejado, quietud, error de
  workflow y max steps.
- Loop gestionado `RunManagedProgressiveLoopV0`: cuando el loop queda en
  `wait_external`, delega la espera en un puerto externo y vuelve a entrar al
  loop para observar receipts/progreso sin que REST, MCP o el operador hagan
  polling manual.
- Preparacion multiagente por batch: `RunOutboxDispatchBatchPlanV0` selecciona
  y reclama varios intents sin ejecutar runtime ni crear goroutines en el core.
- Cierre de batch: `RunOutboxDispatchBatchV0` delega la ejecucion a un puerto
  externo y solo ACKea los items confirmados como success.
- Loop progresivo con `BatchDispatchers`: si `MaxOutboxPerCycle` se activa, el
  runner puede generar varios outbox y el dispatcher batch puede cerrar varios
  agentes en el mismo ciclo.
- Batch real de proceso controlado: `ExternalProcessAgentBatchExecutorV0`
  resuelve specs, lanza procesos por `orquesta-runtime` con concurrencia
  acotada y registra `AgentStarted` item a item.
- Supervision de progreso: un reporte `loop_detected` puede generar
  `AgentWorkAssessed`, `StopRuntimeAgent` y `AgentStopConfirmed` por outbox y
  stopper inyectado.
- Supervision de leases: `AgentLeaseActionCandidateProviderV0` consume
  `AgentTimeoutAssessmentV0` por puerto, valida con `orquesta-core-leases`,
  descarta leases ya proyectadas y entrega al scheduler una accion posterior
  explicita. `stop_agent` registra `AgentLeaseExpired`, emite
  `StopRuntimeAgent` y confirma parada por stopper inyectado.
- Recuperacion por replan tras evaluacion: con una microtarea ya autorizada, un
  agente parado por `loop_detected` puede alimentar `RecordReplanDecision`,
  pedir nueva capacidad y arrancar un agente de reemplazo desde candidates
  explicitos, sin reutilizar el agente detenido ni inventar `ReworkRequested`.
- Puente de replan por assessment: `AgentAssessmentReplanCandidateProviderV0`
  consume planes de recuperacion por puerto, valida la propuesta con
  `orquesta-core-replanner` y entrega `ReplanFollowupCandidates` al scheduler.
- Puente de replan por retrabajo de revision:
  `ReviewReworkReplanCandidateProviderV0` consume planes por puerto, valida con
  `orquesta-core-replanner`, reabre `programacion` mediante `OpenPhase` explicito
  y permite continuar `RequestCapacity -> CapacityDecided -> RequestAgent` sin
  intervencion manual.
- Split de retrabajo de revision: un plan `split_task` tras
  `changes_requested` puede autorizar nuevas microtareas con `CreateMicrotask`,
  materializarlas en `WorkflowTaskStorePortV0` por puerto y dejar que
  `WorkflowTaskCandidateProviderV0` genere capacidad/agente para cada una.
- Observaciones de progreso por puerto: un conector externo puede entregar
  reportes compactos de heartbeat/progreso y el core los convierte en
  candidatos de supervision sin leer logs, transcripts, DB ni runtime concreto.
- Observaciones de receipt por puerto: un conector externo puede entregar un
  receipt compacto y el core lo convierte en `RegisterDelivery` o
  `RegisterPhaseArtifact` sin leer archivos, logs, transcripts, DB ni runtime
  concreto.
- Observaciones de review gate por puerto: un conector externo puede entregar
  resultado `accepted`, `changes_requested` o `rejected`; el core lo convierte
  en `RequestReview` + `RecordReviewResult` y despues `AcceptReview` o
  `RequestRework` segun estado durable.
- Microtareas de workflow a scheduling: `WorkflowTaskCandidateProviderV0`
  rehidrata `WorkflowTaskV0` por puerto externo, respeta `run.Tasks` como lista
  autorizada y construye work candidates por builder validado.
- Store de microtareas en memoria para tests y smoke: `InMemoryWorkflowTaskStoreV0`
  guarda DTOs validados sin introducir DB ni repositorio legacy.
- Integracion batch + supervision: dos procesos reales controlados pueden
  arrancar en paralelo por batch; si solo uno reporta `loop_detected`, el
  stopper para ese `process_ref` y el otro proceso sigue en ejecucion.
- Regla arquitectonica: sin imports de `cmd`, `db` ni conectores de runtime.

## Que no cierra

- Lanzamiento productivo de agentes Codex, Claude o Gemini.
- Seleccion de proveedor, modelo, credenciales, HOME, OAuth o cuotas.
- Eleccion de DB, cola, lock distribuido o runtime operativo.
- Protocolo productivo de readiness/ACK emitido por un proveedor real.
- Adaptador productivo durable que lea ACK real de proveedor fuera de memoria.
- Registro durable productivo del mapeo `agent_request_id -> process_ref`.
- Parada operativa de sesiones/procesos de proveedores reales con idempotencia
  demostrada por conector.
- Entrada HTTP/MCP como contrato estable de produccion.
- Prueba productiva continua con agentes reales arrancados por Orquesta; existe
  smoke opt-in con Codex real para formulario -> MCP -> director -> ACK ->
  `PhaseArtifactRegistered`.
- Ejecucion paralela productiva con proveedores reales; el batch de proceso
  controlado ya esta probado, pero Codex/Claude/Gemini requieren conectores
  operativos configurados por el operador.

El nucleo ya valida el primer lanzamiento de proceso por puerto. El siguiente
cierre no es meter proveedores aqui, sino crear conectores operativos que
resuelvan Codex primero, y despues Claude/Gemini si se aprueban, desde
configuracion externa. Ollama/vLLM/local quedan preparados por arquitectura,
pero diferidos.

## Adaptadores incluidos

- `InMemoryRunStoreV0`;
- `InMemoryEventSinkV0`;
- `InMemoryOutboxLedgerV0`;
- `StaticCandidateProviderV0`;
- `WorkflowTaskCandidateProviderV0`, adaptador que convierte `WorkflowTaskV0`
  externos y autorizados por `OrchestrationRunV0.Tasks` en candidatos del
  scheduler.
- `InMemoryWorkflowTaskStoreV0`, store no productivo para tests/smoke de
  microtareas de workflow; implementa lectura y escritura por puertos separados.
- `FakeLifecycleAgentLauncherV0`, solo para `dry_run` y tests.
- `FakeLifecycleAgentStopperV0`, solo para `dry_run` y tests.
- `ExternalProcessAgentLauncherV0`, adaptador opt-in que delega en
  `orquesta-runtime` con `ExternalAgentLaunchSpecV0`,
  `ExternalAgentProcessCommandResolverV0`, runtime inyectado, registro de
  procesos y stopper de procesos. Si readiness o registro fallan tras arrancar
  un proceso, ejecuta cleanup best-effort con timeout corto.
- `ExternalProcessAgentBatchExecutorV0`, batch executor opt-in para varios
  `LaunchRuntimeAgent`; no ACKea items fallidos por arrastre y registra
  `AgentFailed` cuando el proceso externo queda bloqueado antes de arrancar.
  Si readiness, registro o workflow fallan tras arrancar un proceso, ejecuta
  cleanup best-effort antes de devolver ACK failed.

Contrato de ausencia: los stores de run deben devolver `RunNotFoundErrorV0`
cuando un run no existe. Los adaptadores superiores solo pueden crear estado
inicial ante ese error; otros fallos de store se propagan para evitar resets de
progreso.
- `ExternalAgentLaunchSpecResolverV0`, resolver generico por puertos para
  dependencias, contexto materializado y perfil/comando del conector.
- `ComposedAgentLauncherDependenciesResolverV0`, adaptador que compone
  resolutores inyectados sin consultar legacy, DB ni runtime concreto.
- `ProgressSupervisionCandidateProviderV0`, adaptador que agrega observaciones
  compactas de progreso a los candidatos del scheduler por puerto.
- `AgentLeaseActionCandidateProviderV0`, adaptador que convierte evaluaciones
  compactas de lease en `LeaseActionCandidates` usando `orquesta-core-leases`;
  no observa relojes, procesos, DB, logs ni proveedor.
- `AgentAssessmentReplanCandidateProviderV0`, adaptador que convierte planes
  externos de recuperacion por `AgentWorkAssessed` en followups explicitos de
  replan usando `orquesta-core-replanner`; solo materializa `retry_task`,
  `replace_agent` y `escalate_capacity`.
- `DeliveryCandidateProviderV0`, adaptador que agrega observaciones compactas
  de receipt a los candidatos del scheduler por puerto.
- `ReviewGateCandidateProviderV0`, adaptador que agrega observaciones compactas
  de revision a los candidatos del scheduler por puerto. Para
  `changes_requested` o `rejected` genera `RequestRework`; para `accepted`
  genera `AcceptReview`.
- `ContextBundleRuntimeMaterializerV0`, adaptador de materializacion de
  contexto sobre `orquesta-context` y reader inyectado.
- `AgentReadinessProbePortV0`, puerto opcional de readiness por refs opacas.
- `InMemoryAgentProcessRegistryV0`, registro en memoria para tests y dry-run.
- `ProcessAgentStopperV0`, stopper opt-in que resuelve `process_ref` por puerto
  y delega parada en runtime inyectado.
- `AgentLauncherExecutorV0.FailureStopper`, puerto opcional para parar un
  agente ya lanzado si el registro `AgentStarted` falla despues del launch.
- `AgentProcessRegistryPortV0` y `AgentProcessRegistryRecordV0` son alias del
  contrato neutral `modulos/orquesta-agent-process-registry`; asi persistence
  puede implementar el registro durable sin importar el nucleo.
- `BuildDirectorRunStatsV0` expone `closure` con estado `blocked`, `ready` o
  `closed`, mas causas compactas (`contratos`, `programacion_entregas`,
  `revision_final`, `validacion_final`, `fase_cierre`, `run_blockers`). Es una
  proyeccion generica derivada solo del run; no importa factory, DB ni runtime.

Adaptador exterior disponible fuera del nucleo:

- `modulos/orquesta-runtime-codex-delivery`: implementa
  `AgentDeliveryObservationProviderPortV0` leyendo descriptors de ACK Codex por
  puerto externo y devolviendo observaciones neutrales. En programacion activa
  `RegisterDelivery`; en fases no-programacion activa `RegisterPhaseArtifact`.

## Pruebas esperadas

Validacion minima del nucleo:

```bash
go test ./modulos/orquesta-orchestration-core -count=1
```

Esta prueba cubre el contrato de aplicacion y fake/dry-run. Para declarar verde
el corte minimo de proceso real tambien debe pasar:

```bash
go test ./modulos/orquesta-orchestration-core -run 'TestExternalProcessAgentLauncherV0' -count=1
```

Ese corte demuestra proceso real de lanzamiento, correlacion/idempotencia, ACK
de outbox tras resultado aceptado, registro de `AgentStarted` y ausencia de
proveedor, HOME, OAuth, token o DB hardcodeados. Stop/readiness por puerto
quedan cubiertos por:

```bash
go test ./modulos/orquesta-orchestration-core -run 'Test(AgentStopper|ProbeAgentReadiness|ExternalProcessAgentLauncherV0).*' -count=1
go test ./modulos/orquesta-orchestration-core -run 'TestRunOutboxDispatchBatch' -count=1
go test ./modulos/orquesta-orchestration-core -run 'TestRunProgressiveLoopV0UsesBatchDispatcherForTwoAgentLaunches' -count=1
go test ./modulos/orquesta-orchestration-core -run 'TestRunManagedProgressiveLoopV0' -count=1
go test ./modulos/orquesta-orchestration-core -run 'TestExternalProcessAgentBatchExecutorV0' -count=1
go test ./modulos/orquesta-orchestration-core -run 'TestWorkflowTaskCandidateProviderV0' -count=1
go test ./modulos/orquesta-orchestration-core -run 'TestProgressSupervisionCandidateProviderV0' -count=1
go test ./modulos/orquesta-orchestration-core -run 'TestDeliveryCandidateProviderV0' -count=1
go test ./modulos/orquesta-orchestration-core -run 'TestEvaluateAutoprogrammingReviewGateV0' -count=1
go test ./modulos/orquesta-orchestration-core -run 'TestProgressSupervisionLoopDetectedStopsAgentThroughOutbox' -count=1
go test ./modulos/orquesta-orchestration-core -run 'TestProgressiveLoopV0StopsAgentFromLeaseAssessment' -count=1
go test ./modulos/orquesta-orchestration-core -run 'TestProgressiveLoopV0ReplansFromStoppedAgentAssessment' -count=1
go test ./modulos/orquesta-orchestration-core -run 'TestComposedAgentLauncherDependenciesResolverV0|TestExternalAgentLaunchSpecResolverV0' -count=1
go test ./modulos/orquesta-orchestration-core -run 'Test(InMemoryAgentProcessRegistryV0|ProcessAgentStopperV0)' -count=1
go test ./modulos/orquesta-orchestration-core -run 'TestAgentLauncherExecutorV0WorkflowFailureStopsLaunchedProcess' -count=1
go test ./modulos/orquesta-runtime -run 'TestProcessRuntimeConnectorV0StopEsIdempotente' -count=1
```

Las estadisticas de cierre quedan cubiertas por:

```bash
go test ./modulos/orquesta-orchestration-core -run 'TestBuildDirectorRunStatsV0ExponeCierre' -count=1
```

El corte de proceso real minimo cubre lanzamiento, batch, readiness opcional,
registro estructural, `AgentFailed` ante launch bloqueado, parada por
`process_ref` y supervision selectiva con procesos locales controlados. Quedan
para conectores productivos el protocolo de readiness del agente real, registro
durable externo, parada idempotente de sesiones/procesos reales y smokes con
Codex, Claude o Gemini configurados.

El corte de revision no aceptada cubre la decision durable hasta
`RequestRework`: `ReviewGateCandidateProviderV0` traduce `changes_requested` y
`rejected` a candidates de rework, y traduce solo `accepted` a `AcceptReview`.
La apertura de microtareas de rework o replan productivo queda fuera de este
paquete hasta que exista contrato implementado.

## Decision 2026-05-09: batch failure visible

Cuando un item del batch de agentes no llega a `ExternalAgentProcessLaunchStartedV0`,
el executor registra `AgentFailed` con `reason_code=agent_launch_blocked` y
devuelve un ACK failed con issues sanitizados. No se copian paths, stderr,
HOME, proveedor, modelo ni credenciales al workflow. El objetivo es que el
scheduler deje de esperar indefinidamente a un agente solicitado que nunca
arranco y que el director pueda replanificar con evidencia compacta.

## Decision 2026-05-10: no lanzar procesos ingobernables

`ExternalProcessAgentLauncherV0` y `ExternalProcessAgentBatchExecutorV0`
requieren registro de procesos y stopper de procesos. La razon es que un
proceso externo sin `run_id + agent_request_id -> process_ref` durable no se
puede gobernar despues, y un fallo intermedio no debe dejar agentes vivos fuera
del workflow. La limpieza es best-effort, con timeout corto, y no sustituye al
registro durable externo de `orquesta-persistence`.

## Decision 2026-05-11: control externo de run

`ServiceV0` acepta `RunControlReaderPortV0` opcional. Si esta configurado, el
loop progresivo consulta el estado de la run antes de planificar y antes de
despachar outbox. `paused`, `stop_requested`, `cancel_requested`, `stopped` y
`canceled` cortan el avance de esa app sin acoplar el nucleo a DB, web, MCP ni
runtime.

Un estado ausente no debe romper una run recien creada: si el conector devuelve
`RunControlStateNotFoundErrorV0`, el nucleo usa `running` por defecto. Cualquier
otro error del conector se considera fallo operativo y detiene el loop.

## Decision 2026-05-11: stop/cancel drena agentes vivos

Cuando `RunControl` devuelve `stop_requested` o `cancel_requested` con
checkpoint registrado o modo forzado, el nucleo no planifica trabajo nuevo. En
su lugar:

1. carga la run por `RunStorePortV0`;
2. detecta agentes vivos desde `StartedAgents` menos `FailedAgents`,
   `StoppedAgents` y `ConfirmedStoppedAgents`;
3. materializa un `StopAgent` idempotente por agente vivo;
4. guarda el outbox `StopRuntimeAgent` en el ledger inyectado;
5. despacha solo dispatchers cuyo `message_type` sea `StopRuntimeAgent`.

Esto evita que una peticion de parada lance trabajo pendiente por accidente. El
nucleo sigue sin conocer procesos, sesiones, proveedores, HOME, OAuth ni DB; la
parada fisica real ocurre solo si existe un `AgentStopperPortV0` o batch
equivalente configurado por el operador.
