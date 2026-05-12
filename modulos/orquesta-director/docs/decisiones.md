# Decisiones locales: orquesta-director

Las decisiones de este archivo solo afectan a `orquesta-director`. Si afectan a otro modulo, deben elevarse al director y registrarse en `../CONTRATOS.md`.

## Decisiones iniciales

```text
Fecha: 2026-05-10
Decision: `BuildReplanFollowupsV0` materializa `split_task` de `review_rework` con candidates `CreateMicrotask` explicitos.
Motivo: el retrabajo de una revision puede requerir dividir la tarea y autorizar microtareas nuevas antes de volver a programacion.
Alternativas: preguntar siempre al director; crear tareas en el replanner; hacer que el scheduler invente payloads de microtarea.
Impacto: el director sigue siendo puro: construye RecordReplanDecision, OpenPhase y CreateMicrotask, pero no persiste, no aplica comandos y no inventa refs ni payloads.
Contratos afectados: BuildReplanFollowups v0, CreateMicrotask v0, ReplanFollowupsInputV0.
Estado: aceptada_local
```

```text
Fecha: 2026-05-04
Decision: Crear `orquesta-director` como modulo de composicion, no meter el flujo global en web, factory ni core-workflow.
Motivo: web/MCP/CLI deben ser adaptadores finos; factory genera spec/backlog; core registra el proyecto; core-workflow mantiene estado durable. La composicion entre esos contratos necesita un propietario separado.
Alternativas: hacer que web llame a core directamente; hacer que core-workflow importe core/factory; meter el flujo en factory.
Impacto: el flujo de producto queda hexagonal y puede exponerse luego por REST/MCP/CLI sin duplicar reglas.
Contratos afectados: BootstrapProyectoDesdeAppSpec v0, RegistrarProyectoDesdeAppSpec v0, StartRunFromAppSpecV0.
Estado: aceptada_local
```

```text
Fecha: 2026-05-04
Decision: El primer flujo del director sera puro y sin efectos externos.
Motivo: persistence, observability y runtime ya tienen contratos separados; mezclarlos en el primer slice repetiria el acoplamiento de v1.
Alternativas: persistir inmediatamente; publicar eventos reales; arrancar agentes tras crear el run.
Impacto: DIR-001 solo devuelve resultados y eventos compactos para que adaptadores futuros los consuman.
Contratos afectados: BootstrapProyectoDesdeAppSpec v0.
Estado: aceptada_local
```

```text
Fecha: 2026-05-04
Decision: `BootstrapProyectoDesdeAppSpecV0` devuelve un resumen compacto del registro de core y no el plan/backlog completo.
Motivo: el resultado completo de core conserva trazabilidad y criterios internos del backlog; el contrato del director necesita una salida pequena para persistencia/outbox futura y no debe filtrar AppSpec, backlog ni detalles de infraestructura al workflow.
Alternativas: devolver `RegistroProyectoAceptadoV0` completo; reconstruir el plan en core-workflow; pasar AppSpec/backlog al comando StartRun.
Impacto: el director sigue consumiendo `RegistrarProyectoDesdeAppSpecV0`, pero expone solo ids, refs opacas, conteos, warnings y metadatos minimos junto al StartRun y su resultado puro.
Contratos afectados: BootstrapProyectoDesdeAppSpec v0.
Estado: aceptada_local
```

```text
Fecha: 2026-05-04
Decision: Promover `BootstrapProyectoDesdeAppSpec v0` a contrato compartido minimo en `../CONTRATOS.md`.
Motivo: el flujo compone factory, core y core-workflow, y web/MCP/CLI deben consumir una superficie unica sin duplicar orquestacion.
Alternativas: mantenerlo solo local; hacer que cada adaptador componga los tres contratos; esperar a persistence real.
Impacto: el detalle queda en este modulo y el registro global solo publica propietario, consumidores, DTOs, errores e invariantes.
Contratos afectados: BootstrapProyectoDesdeAppSpec v0.
Estado: aceptada_director
```

```text
Fecha: 2026-05-05
Decision: Las pruebas progresivas del nucleo se ejecutan primero con runtime fake en memoria y contratos, no con un agente real.
Motivo: El workflow ya puede solicitar capacidad, arranque logico, evaluacion de trabajo y parada logica de agentes mediante eventos/outbox, y `orquesta-runtime` ya expone `RuntimeFakeLifecycleV0` para probar `launched -> loop_detected -> stopped` sin procesos reales. Esto permite validar gobierno y parada por evidencia sin introducir proveedor, HOME, OAuth, filesystem ni control de procesos dentro del director.
Alternativas: fingir una parada con `BlockRun`; saltar directamente a procesos reales; mezclar adaptador runtime dentro del director.
Impacto: DIR-002 prueba app simple, replay, fases, low->xhigh, un agente, dos agentes logicos, reporte de bucle, `AssessAgentWork`, `LaunchRuntimeAgent`, `StopRuntimeAgent` y `RuntimeFakeLifecycleV0`. La parada de proceso real queda para conector runtime con handle, heartbeat y ACK.
Contratos afectados: BootstrapProyectoDesdeAppSpec v0, OrchestrationRun v0, OutboxMessage v0, CapacityDecision v0, AgentLauncherInbound v0, AgentProgressReport v0, AssessAgentWork v0, AgentStopperInbound v0, RuntimeFakeLifecycle v0.
Estado: aceptada_local
```

```text
Fecha: 2026-05-05
Decision: El despacho fake de outbox runtime en pruebas vive en un helper local `_test.go` de `orquesta-director`.
Motivo: La prueba progresiva debe ejercer el contrato como lo haria un adaptador de test: recibir `OutboxMessageV0`, validar el envelope publico, convertirlo con `AgentLauncherInboundV0` o `AgentStopperInboundV0`, validar esos contratos publicos y llamar a `RuntimeFakeLifecycleV0`, sin conversion manual en el cuerpo del caso.
Alternativas: mantener conversion manual en DIR-P005; crear un adaptador productivo; tocar runtime o core-workflow.
Impacto: DIR-P005 usa `runtime_fake_outbox_dispatcher_v0_test.go`; los mensajes no soportados devuelven error codificado `runtime_fake_outbox_tipo_no_soportado` y no se ignoran.
Contratos afectados: OutboxMessage v0, AgentLauncherInbound v0, AgentStopperInbound v0, RuntimeFakeLifecycle v0.
Estado: aceptada_local
```

```text
Fecha: 2026-05-05
Decision: El cierre completo del director se valida con una prueba fake/in-memory que usa contratos publicos del workflow despues del bootstrap.
Motivo: `orquesta-core-workflow` ya expone contratos publicos para registrar lifecycle de agente, registrar entrega, solicitar revision, aceptar revision, cerrar tarea, registrar validacion final y cerrar el run. El director debe demostrar que puede componerlos desde una app simple sin tocar estado interno, persistencia, proveedor ni runtime real.
Alternativas: esperar a un adaptador runtime real; mutar la proyeccion del run en el test; probar solo contratos aislados en core-workflow.
Impacto: DIR-P007 cubre el flujo hasta `RunClosed` con refs compactas, solicitud de capacidad, launch runtime fake, `AgentStarted` antes de entrega y outbox vacio en los comandos de lifecycle/entrega/cierre. La programacion real y la persistencia externa quedan fuera.
Contratos afectados: BootstrapProyectoDesdeAppSpec v0, RegisterAgentStarted v0, RegisterDelivery v0, RequestReview v0, AcceptReview v0, CloseTask v0, RegisterFinalValidation v0, CloseRun v0, OrchestrationRun v0.
Estado: aceptada_local
```

```text
Fecha: 2026-05-05
Decision: El director traduce AgentProgressReportV0 a comandos de workflow mediante un caso de uso puro, sin llamar runtime ni adaptadores.
Motivo: El reporte pertenece a orquesta-runtime, pero la decision durable debe quedar registrada en orquesta-core-workflow como AssessAgentWork y, para stalled, como AskDirector separado. Mantener la traduccion en el director evita que runtime importe workflow o que core-workflow importe tipos runtime.
Alternativas: hacer que runtime emita directamente AssessAgentWork; meter la supervision dentro de core-workflow; resolver bucles con StopAgent manual sin evaluacion.
Impacto: DIR-004 valida el reporte con el validador publico de runtime, produce comandos compactos, no propaga resumen bruto ni evidencias prohibidas y deja que core-workflow emita AgentStopRequested/StopRuntimeAgent o SendDirectorQuestion.
Contratos afectados: AgentProgressReport v0, AssessAgentWork v0, DirectorQuestion v0, OutboxMessage v0.
Estado: aceptada_local
```

```text
Fecha: 2026-05-05
Decision: Dividir el supervisor de progreso por responsabilidades manteniendo una unica entrada publica.
Motivo: `agent_progress_supervisor_v0.go` habia crecido a 299 lineas mezclando tipos, validacion, decisiones, summaries, evidencias y prohibiciones. Separarlo en ficheros pequenos reduce acoplamiento de lectura sin cambiar JSON tags, nombres publicos, errores ni comportamiento.
Alternativas: dejar el fichero monolitico; crear abstracciones nuevas; mover logica a runtime o core-workflow.
Impacto: `BuildAgentProgressSupervisionV0` queda como orquestacion principal; tipos/constantes, validacion/errores y helpers compactos viven en ficheros locales del mismo paquete. No cambia ningun contrato ni se introduce DB, runtime real, proveedor, HOME u OAuth.
Contratos afectados: BuildAgentProgressSupervision v0 sin cambios.
Estado: aceptada_local
```

```text
Fecha: 2026-05-05
Decision: Dividir el dispatcher fake de outbox runtime por responsabilidades dentro de tests locales.
Motivo: `runtime_fake_outbox_dispatcher_v0_test.go` habia crecido a 289 lineas mezclando dispatcher principal, must helpers, conversion outbox -> inbound runtime, errores/asserts y el test de mensaje no soportado. Separarlo reduce el coste de lectura del harness sin cambiar nombres usados por otros tests ni comportamiento.
Alternativas: dejar el fichero monolitico; mover helpers a runtime; crear un adaptador productivo; reescribir el dispatcher.
Impacto: el dispatcher principal queda en `runtime_fake_outbox_dispatcher_v0_test.go`, la conversion contractual en `runtime_fake_outbox_conversion_v0_test.go` y errores/asserts/test negativo en `runtime_fake_outbox_errors_v0_test.go`. Sigue siendo test/harness pequeno, in-memory, con refs opacas y sin DB real, provider, HOME, OAuth ni procesos.
Contratos afectados: OutboxMessage v0, AgentLauncherInbound v0, AgentStopperInbound v0, RuntimeFakeLifecycle v0 sin cambios.
Estado: aceptada_local
```

```text
Fecha: 2026-05-05
Decision: Validar en el director el ledger de outbox antes del dispatcher fake solo como integracion de contratos en memoria.
Motivo: `InMemoryOutboxLedgerV0` pertenece a persistence y el dispatcher fake local ya traduce `LaunchRuntimeAgent`/`StopRuntimeAgent` a inbound runtime. Un test de director debe demostrar la frontera completa guardar -> listar pendientes por run/puerto -> despachar -> ACK sin activar DB real ni runtime real.
Alternativas: despachar directamente desde `OrchestrationCommandResultV0.Outbox`; crear un dispatcher productivo; mover la prueba a persistence; esperar a una cola real.
Impacto: DIR-006 guarda mensajes reales de core-workflow en el ledger, lista `agent_launcher`, despacha desde pendientes con el fake actual, registra ACK `dispatched` con `dispatch_ref` opaca y comprueba que solo desaparece el mensaje ACKed. El mismo corte cubre un `SendDirectorQuestion` generado por supervision `stalled` como pendiente/ACK de target `director`, sin usar el dispatcher runtime.
Contratos afectados: OutboxMessage v0, InMemoryOutboxLedger v0, AgentLauncherInbound v0, AgentStopperInbound v0, RuntimeFakeLifecycle v0, BuildAgentProgressSupervision v0 sin cambios.
Estado: aceptada_local
```

```text
Fecha: 2026-05-05
Decision: Promover el patron ledger + dispatcher a un caso de uso del director con puertos locales.
Motivo: El patron ya estaba probado como harness, pero el director necesitaba una frontera reutilizable que guarde outbox opcional, liste pendientes por run/target, despache por puerto inyectado y registre ACK sin conocer persistence ni runtime concretos.
Alternativas: mantenerlo solo en tests progresivos; importar InMemoryOutboxLedgerV0 en producto; crear un dispatcher productivo ahora; usar goroutines/colas para el ciclo.
Impacto: `RunOutboxDispatchCycleV0` consume `OutboxMessageV0` de core-workflow y define `OutboxLedgerPortV0`/`OutboxDispatcherPortV0` locales. El adapter a `InMemoryOutboxLedgerV0` queda solo en tests. El ciclo es serial, determinista, requiere `dispatched_at`, exige `dispatch_ref`, registra ACK `failed` terminal en fallo de dispatcher y no introduce DB real, runtime real, provider, HOME, OAuth, procesos, goroutines, sleeps ni colas.
Contratos afectados: RunOutboxDispatchCycle v0, OutboxMessage v0.
Estado: aceptada_local
```

```text
Fecha: 2026-05-05
Decision: La prueba real de gobierno del nucleo vive en `orquesta-e2e`, no dentro de `orquesta-director`.
Motivo: El caso cruza publicamente core-workflow, director, runtime, persistence y capacity; meterlo en el director obligaria a importar conectores concretos y confundiria composicion de producto con adaptadores de prueba. `orquesta-e2e` puede validar el ensamblaje sin que ningun modulo conozca internals de otro.
Alternativas: poner la prueba en director; probar runtime y persistence por separado; esperar a un proveedor externo real.
Impacto: DIR-009/DIR-P013 documentan la decision en director, pero el codigo queda en `modulos/orquesta-e2e/e2e_real_governance_*_v0_test.go`. La prueba usa procesos locales controlados, ruta explicita de test para ledger durable y contratos publicos; no activa web, CLI, MCP productivo, deploy, HOME/OAuth ni multi-cuenta real.
Contratos afectados: OrchestrationRun v0, RunOutboxDispatchCycle v0, ProcessRuntimeConnectorV0, FileOutboxLedgerV0, CapacityDecision v0, AssessAgentWork v0.
Estado: aceptada_local
```

```text
Fecha: 2026-05-05
Decision: El director prepara ContextBundleV0 a partir de LaunchRuntimeAgent, pero no mete contexto en el outbox del workflow.
Motivo: core-workflow debe seguir emitiendo mensajes compactos sin detalles de modulo, runtime ni proveedor. El director conoce la planificacion, write_set, modulo objetivo y capacidad decidida, y puede componer esos datos con orquesta-context antes de entregar a runtime.
Alternativas: ampliar LaunchRuntimeAgent con contexto; dejar que runtime lea docs por filesystem; pedir al agente que busque contexto.
Impacto: `BuildLaunchContextBundleV0` consume `OutboxMessageV0` y `ContextBundleV0` publicos. El payload durable sigue pequeno; la materializacion de refs queda para conectores futuros.
Contratos afectados: ContextBundleV0, OutboxMessage v0, LaunchRuntimeAgent, RuntimeLaunchRequestV0.
Estado: aceptada_local
```

```text
Fecha: 2026-05-06
Decision: El director compone el gate de concurrencia como caso de uso puro antes de RequestAgent.
Motivo: `orquesta-core-concurrency` ya decide allow/block/ask_director y `orquesta-core-workflow` solo registra el gate. El director es la frontera adecuada para encadenar ambos contratos y emitir RequestAgent exclusivamente cuando la evaluacion permita los claims sujetos.
Alternativas: meter EvaluateConcurrencyGateV0 dentro de core-workflow; lanzar todos los candidates y confiar en el reducer; crear scheduler/persistence en este corte.
Impacto: `BuildConcurrencyGateAgentRequestsV0` construye siempre RecordConcurrencyGate para una evaluacion valida y construye RequestAgent solo para candidates por claim_ref cuando decision=allow_request_agent. Block/ask_director no generan RequestAgent ni outbox.
Contratos afectados: WorksetClaimV0, EvaluateConcurrencyGateV0, RecordConcurrencyGate, RequestAgent.
Estado: aceptada_local
```

```text
Fecha: 2026-05-06
Decision: Las acciones posteriores a lease expirado se traducen en el director como comandos separados y sin efectos locales inventados.
Motivo: core-workflow ya registra RegisterAgentLeaseExpired como observacion durable y documenta que la accion posterior entra por otro comando. El director puede componer StopAgent o AskDirector cuando la recomendacion lo permite, pero retry/mark_failed/mark_stopped/replan_task/alert_only necesitan contratos explicitos antes de producir efectos.
Alternativas: hacer que RegisterAgentLeaseExpired detenga o falle agentes automaticamente; crear efectos locales para retry/replan; mover la decision al runtime.
Impacto: BuildPostLeaseActionV0 siempre construye RegisterAgentLeaseExpired primero, y solo anade StopAgent o AskDirector como comandos independientes. El resto queda marcado `unsupported/needs_director`, sin persistence, runtime real, provider, HOME ni OAuth.
Contratos afectados: BuildPostLeaseAction v0, RegisterAgentLeaseExpired v0, StopAgent v0, AskDirector v0.
Estado: aceptada_local
```

```text
Fecha: 2026-05-06
Decision: `BuildReplanFollowupsV0` rechaza candidates de agente incluidos en `blocked_agent_refs`.
Motivo: Tras `AgentFailed`, reutilizar el mismo `agent_request_id` produce un replan aparentemente valido pero el workflow no vuelve a lanzar nada porque ese agente ya es terminal. El director debe exigir un candidate explicito de reemplazo.
Alternativas: Dejar que core-workflow devuelva no-op; inventar una ref nueva en el director; resolverlo en runtime; ampliar contexto para consultar historiales.
Impacto: `ReplanFollowupsInputV0` agrega `blocked_agent_refs`; el builder valida la ref antes de construir `RequestAgent` y sigue sin persistir, aplicar comandos ni elegir proveedor/runtime.
Contratos afectados: BuildReplanFollowups v0, RequestAgent v0.
Estado: aceptada_local
```

```text
Fecha: 2026-05-06
Decision: Los followups de replan se componen en el director solo desde candidates explicitos.
Motivo: RecordReplanDecision debe dejar memoria durable de la decision aceptada, pero crear capacidad, agentes o preguntas requiere payloads ya decididos por replanner/scheduler/capacidad. El director no debe derivar tareas, roles, capacidad ni refs nuevas desde una accion generica.
Alternativas: inventar RequestCapacity/RequestAgent desde accepted_action; mezclar replan con scheduler/persistence; emitir outbox directamente desde el director.
Impacto: BuildReplanFollowupsV0 siempre construye RecordReplanDecision y solo anade RequestCapacity, RequestAgent o AskDirector cuando el input trae meta/payload completos para esos comandos. split_task/abort_task o candidates ausentes quedan `needs_director/unsupported`.
Contratos afectados: BuildReplanFollowups v0, RecordReplanDecision v0, RequestCapacity v0, RequestAgent v0, AskDirector v0.
Estado: aceptada_local
```

```text
Fecha: 2026-05-07
Decision: `BuildReplanFollowupsV0` trata `quality_gate_blocked` como origen local explicito para followups progresivos.
Motivo: Un gate bloqueado debe dejar una decision de replan trazable y desbloquear trabajo pequeno sin saltar a documentacion ni emitir comandos de otra fase.
Alternativas: reutilizar split_task generico sin origen; pedir siempre al director; abrir documentacion; aplicar comandos contra el workflow dentro del builder.
Impacto: `ReplanFollowupsInputV0` agrega `source_kind`. Para `quality_gate_blocked` en programacion, el director solo construye followups aplicables en esa fase: RequestCapacity, RequestAgent tras capacidad decidida o AskDirector. No construye CreateMicrotask ni RequestRework porque pertenecen a planificacion y revision.
Contratos afectados: BuildReplanFollowups v0, RequestCapacity v0, RequestAgent v0, AskDirector v0.
Estado: aceptada_local
```

```text
Fecha: 2026-05-10
Decision: `AgentStalledV0` pregunta al director sin bloquear el run.
Motivo: Un agente real puede tardar mas que la ventana de supervision y aun asi
entregar un ACK valido. Bloquear el run por stalled hacia que una entrega tardia
del propio agente ya lanzado quedase rechazada por estado `bloqueada`.
Alternativas: permitir RegisterPhaseArtifact en runs bloqueados; subir
timeouts; desactivar supervision durante agentes reales; bloquear solo despues
de varios ciclos externos.
Impacto: BuildAgentProgressSupervisionV0 mantiene AssessAgentWork
ask_director y SendDirectorQuestion para estadisticas/decision, pero el
AskDirector de stalled lleva `blocking=false`. La parada sigue reservada a
loop_detected, lease/politica explicita o decision posterior del director.
Contratos afectados: BuildAgentProgressSupervision v0, AskDirector v0,
AgentProgressReport v0.
Estado: aceptada_local
```

```text
Fecha: 2026-05-10
Decision: `BuildReplanFollowupsV0` puede construir `OpenPhase` como followup explicito de replan.
Motivo: Un retrabajo generado en `revision` debe poder volver a `programacion` antes de pedir nueva capacidad o agente. Meter capacidad/agente directamente en fase `revision` rompe las precondiciones del workflow y empuja a parches.
Alternativas: pedir siempre al director; permitir RequestCapacity fuera de programacion; duplicar logica en el scheduler; crear tareas nuevas automaticamente.
Impacto: `ReplanFollowupsInputV0` acepta `OpenPhaseCandidate`; el scheduler lo emite antes de `RequestCapacity` y solo permite followups de otra fase si existe ese OpenPhase explicito en el mismo plan. `source_kind=review_rework` usa AskDirector para acciones no automaticas como split_task.
Contratos afectados: BuildReplanFollowups v0, RecordReplanDecision v0, OpenPhase v0, RequestCapacity v0, RequestAgent v0, AskDirector v0.
Estado: aceptada_local
```

```text
Fecha: 2026-05-12
Decision: `capacity_limited` cierra el agente parado y fuerza relevo controlado.
Motivo: si un runtime externo se detiene antes de ACK por capacidad limitada, el
director no debe esperar mas ciclos ni clasificar el trabajo como basura. El
trabajo no existe aun; la accion correcta es liberar el agente logico y dejar
que capacidad/replanificacion escojan otra opcion.
Alternativas:
  - AskDirector bloqueante siempre: descartado porque deja proceso ya parado
    como deuda operativa y puede bloquear la app.
  - Tratarlo como over_budget: descartado porque no hay evidencia de cuota
    consumida o agotada, solo capacidad externa no disponible.
  - Reintentar desde el director inventando modelo/proveedor: descartado por
    romper hexagonal y filtrar politica de capacidad al core.
Impacto: BuildAgentProgressSupervisionV0 emite assessment
`capacity_limited` y accion `stop_agent` cuando la parada es segura; si el
agente esta protegido, pregunta al director sin inventar runtime, modelo,
provider, HOME ni OAuth.
Contratos afectados: AgentProgressReport v0, AssessAgentWork v0,
BuildAgentProgressSupervision v0.
Estado: aceptada_local
```

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
