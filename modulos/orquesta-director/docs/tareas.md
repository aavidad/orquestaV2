# Tareas locales: orquesta-director

Cada tarea debe ser pequena y cerrada.

## Backlog inicial

```text
ID: DIR-023
Objetivo: Separar `orquesta-director` de `orquesta-factory` en el bootstrap inicial.
Write-set: bootstrap_appspec_v0.go, bootstrap_appspec_v0_test.go, bootstrap_appspec_smoke_v0_test.go, README.md, docs/contratos.md, docs/pruebas.md, docs/decisiones.md, docs/tareas.md
Contrato: BootstrapProyectoDesdeAppSpec v0
Validacion: 2026-05-23, ok; go test -count=1 ./modulos/orquesta-director.
Bloqueos: el adaptador desde salidas de `orquesta-factory` a DTOs de core queda para otro modulo.
Estado: completada ejecutable
```

```text
ID: DIR-022
Objetivo: Permitir `CreateMicrotask` como followup explicito de `split_task` en rework de revision.
Write-set: replan_followups_*_v0.go, replan_followups_microtasks_v0_test.go, docs locales.
Contrato: BuildReplanFollowups v0, RecordReplanDecision v0, OpenPhase v0, CreateMicrotask v0.
Validacion: 2026-05-10, ok; go test -count=1 ./modulos/orquesta-director.
Bloqueos: no persiste detalles de tarea, no aplica comandos, no inventa microtareas; consume candidates explicitos.
Estado: completada ejecutable
```

```text
ID: DIR-000
Objetivo: Crear miniproyecto de director/composicion con contexto local y contrato inicial.
Write-set: AGENTS.md, README.md, arrancar_codex.sh, docs/contratos.md, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Contrato: BootstrapProyectoDesdeAppSpec v0 local
Validacion: git diff --check -- modulos/orquesta-director
Bloqueos: ninguno
Estado: completada documental inicial
```

```text
ID: DIR-001
Objetivo: Implementar flujo puro BootstrapProyectoDesdeAppSpec v0.
Write-set: bootstrap_appspec_v0.go, bootstrap_appspec_v0_test.go, docs/contratos.md, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Contrato: BootstrapProyectoDesdeAppSpec v0
Validacion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-director ./modulos/orquesta-core ./modulos/orquesta-core-workflow ./modulos/orquesta-factory; git diff --check -- modulos/orquesta-director; wc -l Go tocados bajo 300 lineas.
Bloqueos: ninguno; usa contratos publicos ya existentes.
Estado: completada; resumen minimo promovido a ../CONTRATOS.md
```

```text
ID: DIR-002
Objetivo: Probar progresivamente el nucleo con una app simple, gobierno simulado de capacidad con `CapacityDecided`, arranque logico, supervision de bucle, parada logica de agentes y ciclo fake de runtime en memoria.
Write-set: bootstrap_appspec_smoke_v0_test.go, progressive_orchestration_smoke_v0_test.go, progressive_orchestration_asserts_v0_test.go, runtime_fake_outbox_dispatcher_v0_test.go, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Contrato: BootstrapProyectoDesdeAppSpec v0, OrchestrationRun v0, OutboxMessage v0, CapacityDecided v0, CapacityDecision v0, AgentLauncherInbound v0, AgentProgressReport v0, AssessAgentWork v0, AgentStopperInbound v0, RuntimeFakeLifecycle v0
Validacion: 2026-05-05, ok, gofmt sobre Go tocados; go test -count=1 ./modulos/orquesta-director ./modulos/orquesta-runtime ./modulos/orquesta-core-workflow; git diff --check -- modulos/orquesta-director.
Bloqueos: parada de proceso real y ejecucion real quedan fuera; el dispatcher fake solo traduce outbox `LaunchRuntimeAgent`/`StopRuntimeAgent` hacia `RuntimeFakeLifecycleV0` y no usa procesos, handles, provider, HOME ni OAuth.
Estado: completada ejecutable
```

```text
ID: DIR-003
Objetivo: Anadir prueba progresiva de flujo completo de app simple hasta cierre con capacidad decidida, launch runtime fake, AgentStarted, entrega, revision, resultado aceptado, cierre de tarea, validacion final y cierre de run.
Write-set: progressive_orchestration_full_flow_v0_test.go, progressive_orchestration_agent_lifecycle_v0_test.go, progressive_orchestration_close_commands_v0_test.go, progressive_orchestration_full_flow_asserts_v0_test.go, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Contrato: BootstrapProyectoDesdeAppSpec v0, RegisterAgentStarted v0, RegisterDelivery v0, RequestReview v0, RecordReviewResult v0, AcceptReview v0, CloseTask v0, RegisterFinalValidation v0, CloseRun v0, OrchestrationRun v0.
Validacion: 2026-05-06, ok, gofmt sobre Go tocados; go test -count=1 ./modulos/orquesta-director ./modulos/orquesta-core-workflow; flujo completo con 23 eventos, `AgentStarted` antes de delivery y `ReviewResultRecorded(accepted)` antes de accept.
Bloqueos: ninguno; el caso es fake/in-memory, no programa codigo real, no persiste, no usa runtime real, provider, HOME ni OAuth.
Estado: completada ejecutable
```

```text
ID: DIR-004
Objetivo: Implementar supervisor puro que traduce AgentProgressReportV0 en comandos publicos de orquesta-core-workflow.
Write-set: agent_progress_supervisor_v0.go, agent_progress_supervisor_v0_test.go, docs/contratos.md, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Contrato: AgentProgressReport v0, AssessAgentWork v0, DirectorQuestion v0, OutboxMessage v0.
Validacion: 2026-05-05, ok, gofmt sobre Go tocados; go test -count=1 ./modulos/orquesta-director ./modulos/orquesta-runtime ./modulos/orquesta-core-workflow; git diff --check -- modulos/orquesta-director.
Bloqueos: ninguno; la parada real sigue fuera y queda en outbox StopRuntimeAgent emitido por core-workflow.
Estado: completada ejecutable
```

```text
ID: DIR-005
Objetivo: Sanear tamano de BuildAgentProgressSupervisionV0 dividiendo responsabilidades sin cambiar comportamiento ni contratos.
Write-set: agent_progress_supervisor_v0.go, agent_progress_supervisor_types_v0.go, agent_progress_supervisor_validation_v0.go, agent_progress_supervisor_helpers_v0.go, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Contrato: BuildAgentProgressSupervision v0; se mantienen JSON tags, nombres publicos, errores y tests existentes.
Validacion: 2026-05-05, ok, gofmt sobre Go tocados; go test -count=1 ./modulos/orquesta-director ./modulos/orquesta-runtime ./modulos/orquesta-core-workflow; git diff --check -- modulos/orquesta-director; wc -l Go tocados bajo 250 lineas.
Bloqueos: ninguno; refactor interno puro, sin DB/runtime real/provider/HOME/OAuth ni ficheros grandes.
Estado: completada estructural
```

```text
ID: DIR-006
Objetivo: Probar integracion fake con InMemoryOutboxLedgerV0 entre core-workflow y dispatcher runtime fake antes del dispatch.
Write-set: progressive_orchestration_outbox_ledger_v0_test.go, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Contrato: OutboxMessage v0, InMemoryOutboxLedger v0, AgentLauncherInbound v0, AgentStopperInbound v0, RuntimeFakeLifecycle v0, BuildAgentProgressSupervision v0.
Validacion: 2026-05-05, ok, gofmt sobre Go tocados; go test -count=1 ./modulos/orquesta-director ./modulos/orquesta-persistence ./modulos/orquesta-core-workflow ./modulos/orquesta-runtime; git diff --check -- modulos/orquesta-director; wc -l Go tocado DIR-006: 215 lineas.
Bloqueos: ninguno; ledger y runtime son fakes en memoria, sin DB real, runtime real, provider, HOME, OAuth ni procesos.
Estado: completada ejecutable
```

```text
ID: DIR-007
Objetivo: Sanear tamano de runtime_fake_outbox_dispatcher_v0_test.go dividiendo helpers de test por responsabilidad.
Write-set: runtime_fake_outbox_dispatcher_v0_test.go, runtime_fake_outbox_conversion_v0_test.go, runtime_fake_outbox_errors_v0_test.go, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Contrato: RuntimeFakeLifecycle v0, OutboxMessage v0, AgentLauncherInbound v0, AgentStopperInbound v0; se mantienen nombres de helpers, errores codificados y comportamiento.
Validacion: 2026-05-05, ok, gofmt sobre Go tocados; go test -count=1 ./modulos/orquesta-director ./modulos/orquesta-runtime ./modulos/orquesta-core-workflow; git diff --check -- modulos/orquesta-director; wc -l Go tocados 106/48/154 lineas.
Bloqueos: ninguno; refactor de tests/harness pequeno, refs opacas, sin DB real, runtime real, provider, HOME, OAuth ni procesos.
Estado: completada estructural
```

```text
ID: DIR-008
Objetivo: Convertir el patron director + outbox ledger + dispatcher en un caso de uso/puerto reutilizable del director, fake/in-memory y sin acoplar producto a implementaciones concretas.
Write-set: outbox_dispatch_cycle_types_v0.go, outbox_dispatch_cycle_validation_v0.go, outbox_dispatch_cycle_usecase_v0.go, outbox_dispatch_cycle_adapter_v0_test.go, outbox_dispatch_cycle_helpers_v0_test.go, outbox_dispatch_cycle_usecase_v0_test.go, docs/contratos.md, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Contrato: RunOutboxDispatchCycle v0, OutboxLedgerPortV0, OutboxDispatcherPortV0, OutboxMessage v0.
Validacion: 2026-05-05, ok, gofmt sobre Go tocados; go test -count=1 ./modulos/orquesta-director ./modulos/orquesta-persistence ./modulos/orquesta-core-workflow; git diff --check -- modulos/orquesta-director; wc -l Go tocados 117/161/236/81/85/221 lineas.
Bloqueos: ninguno; el adapter a InMemoryOutboxLedgerV0 vive solo en tests y no hay DB real, runtime real, provider, HOME, OAuth, procesos, goroutines, sleeps ni colas.
Estado: completada ejecutable
```

```text
ID: DIR-009
Objetivo: Validar el nucleo fuera del director con procesos reales controlados, ledger durable explicito, parada por trabajo basura y cambio de capacidad low -> xhigh.
Write-set: ../orquesta-e2e/e2e_real_governance_v0_test.go, ../orquesta-e2e/e2e_real_governance_commands_v0_test.go, ../orquesta-e2e/e2e_real_governance_adapters_v0_test.go, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Contrato: OrchestrationRun v0, RunOutboxDispatchCycle v0, ProcessRuntimeConnectorV0, FileOutboxLedgerV0, CapacityDecision v0, AssessAgentWork v0.
Validacion: 2026-05-05, ok; go test -count=1 -v ./modulos/orquesta-e2e -run TestE2ERealGobiernaDosProcesosParaBasuraYEscalaCapacidadV0; go test -race -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-director ./modulos/orquesta-runtime ./modulos/orquesta-persistence ./modulos/orquesta-capacity ./modulos/orquesta-e2e.
Bloqueos: no ejecuta Codex real, proveedor externo, HOME/OAuth/multi-cuenta, cuotas reales, UI, CLI, MCP productivo ni deploy. Es prueba real de proceso local controlado y contratos publicos.
Estado: completada ejecutable
```

```text
ID: DIR-010
Objetivo: Construir ContextBundleV0 para un outbox LaunchRuntimeAgent antes de entregar la orden al runtime.
Write-set: context_bundle_for_launch_types_v0.go, context_bundle_for_launch_v0.go, context_bundle_for_launch_helpers_v0.go, context_bundle_for_launch_v0_test.go, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Contrato: ContextBundleV0, OutboxMessage v0, LaunchRuntimeAgent, RuntimeLaunchRequestV0.
Validacion: 2026-05-05, ok; gofmt sobre Go tocados; go test -count=1 ./modulos/orquesta-director ./modulos/orquesta-runtime ./modulos/orquesta-context; git diff --check -- modulos/orquesta-director.
Bloqueos: no materializa refs ni arranca runtime; target_module, write_set y capacity_level los aporta director/planificacion, no el payload compacto de core-workflow.
Estado: completada ejecutable
```

```text
ID: DIR-011
Objetivo: Implementar un usecase puro del director para evaluar concurrencia de WorksetClaimV0 y construir RecordConcurrencyGate mas RequestAgent solo si el gate permite.
Write-set: concurrency_gate_agent_requests_*_v0.go, concurrency_gate_agent_requests_v0_test.go, docs/contratos.md, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Contrato: WorksetClaimV0, EvaluateConcurrencyGateV0, RecordConcurrencyGate, RequestAgent.
Validacion: 2026-05-06, ok; gofmt sobre Go tocados; go test -count=1 ./modulos/orquesta-director.
Bloqueos: no hay scheduler, reducer, persistence, outbox ni runtime; los candidates RequestAgent ya deben venir dados por la planificacion/capacidad previa.
Estado: completada ejecutable
```

```text
ID: DIR-012
Objetivo: Implementar usecase puro para acciones posteriores a lease expirado.
Write-set: post_lease_action_types_v0.go, post_lease_action_validation_v0.go, post_lease_action_helpers_v0.go, post_lease_action_v0.go, post_lease_action_v0_test.go, docs/contratos.md, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Contrato: BuildPostLeaseAction v0, RegisterAgentLeaseExpired v0, StopAgent v0, AskDirector v0.
Validacion: 2026-05-06, ok; gofmt sobre Go tocados; go test -count=1 ./modulos/orquesta-director.
Bloqueos: no persiste, no ejecuta runtime, no materializa retry/mark_failed/mark_stopped/replan_task/alert_only; esas acciones quedan como unsupported/needs_director sin efectos inventados.
Estado: completada ejecutable
```

```text
ID: DIR-018
Objetivo: Evitar que `BuildReplanFollowupsV0` reutilice refs de agentes fallidos/bloqueados al construir `RequestAgent`.
Write-set: replan_followups_types_v0.go, replan_followups_helpers_v0.go, replan_followups_validation_v0.go, replan_followups_v0_test.go, docs locales.
Contrato: BuildReplanFollowups v0, RequestAgent v0, blocked_agent_refs.
Validacion: 2026-05-06, ok; go test -count=1 ./modulos/orquesta-director.
Bloqueos: No inventa refs nuevas; el caller debe aportar candidate explicito de reemplazo.
Estado: completada ejecutable
```

```text
ID: DIR-013
Objetivo: Implementar usecase puro pequeno para replan followups del director.
Write-set: replan_followups_types_v0.go, replan_followups_validation_v0.go, replan_followups_helpers_v0.go, replan_followups_v0.go, replan_followups_v0_test.go, docs/contratos.md, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Contrato: BuildReplanFollowups v0, RecordReplanDecision v0, RequestCapacity v0, RequestAgent v0, AskDirector v0.
Validacion: 2026-05-06, ok; gofmt sobre Go tocados; go test -count=1 ./modulos/orquesta-director; git diff --check -- modulos/orquesta-director.
Bloqueos: no persiste, no aplica comandos, no emite outbox, no inventa tareas/capacidad/agentes; RequestCapacity, RequestAgent y AskDirector solo salen si sus candidates explicitos vienen dados.
Estado: completada ejecutable
```

```text
ID: DIR-019
Objetivo: Soportar followups de replan cuando el origen es `quality_gate_blocked`.
Write-set: replan_followups_types_v0.go, replan_followups_helpers_v0.go, replan_followups_validation_v0.go, replan_followups_v0.go, replan_followups_quality_gate_v0_test.go, docs locales.
Contrato: BuildReplanFollowups v0, RecordReplanDecision v0, RequestCapacity v0, RequestAgent v0, AskDirector v0.
Validacion: 2026-05-07, ok; gofmt sobre Go tocados; go test -count=1 .
Bloqueos: no aplica comandos, no crea microtareas fuera de planificacion, no abre rework fuera de revision, no abre fase documental, no consulta DB/runtime/provider.
Estado: completada ejecutable
```

```text
ID: DIR-020
Objetivo: Convertir stalled de progreso en consulta no bloqueante al director.
Write-set: agent_progress_supervisor_helpers_v0.go, agent_progress_supervisor_v0_test.go, docs locales.
Contrato: BuildAgentProgressSupervision v0, AskDirector v0, AgentProgressReport v0.
Validacion: 2026-05-13, ok; go test -count=1 ./modulos/orquesta-director -run TestBuildAgentProgressSupervisionV0StalledPreguntaNoBloqueanteAlDirector; go test -count=1 ./modulos/orquesta-app-codex-stack -run TestCodexStackV0ProgressStalledProtegeDirectorInicial.
Bloqueos: no modifica loop_detected ni lease timeout; no permite entregas en runs bloqueados por otras causas.
Estado: completada ejecutable
```

```text
ID: DIR-021
Objetivo: Permitir que un replan de revision reabra programacion antes de pedir capacidad/agente.
Write-set: replan_followups_types_v0.go, replan_followups_helpers_v0.go, replan_followups_validation_v0.go, replan_followups_v0.go, replan_followups_v0_test.go, scheduler_tick_replan_* en orquesta-director-scheduler, docs locales.
Contrato: BuildReplanFollowups v0, ReplanOpenPhaseCandidateV0, RecordReplanDecision v0, OpenPhase v0.
Validacion: 2026-05-10, ok; go test -count=1 ./modulos/orquesta-director ./modulos/orquesta-director-scheduler.
Bloqueos: no crea microtareas ni decide proveedores; OpenPhase solo aparece si el caller lo aporta como candidate explicito.
Estado: completada ejecutable
```

## Plantilla

```text
ID:
Objetivo:
Write-set:
Contrato:
Validacion:
Bloqueos:
Estado:
```
```text
ID: DIR-023
Objetivo: Tratar `correlation_id` y refs opacas como identidad causal, no como
contenido sujeto a filtros de detalle prohibido en `BuildAgentProgressSupervisionV0`.
Write-set: agent_progress_supervisor_validation_v0.go, agent_progress_supervisor_v0_test.go, docs locales.
Contrato: BuildAgentProgressSupervision v0.
Validacion: pendiente en este corte.
Bloqueos: no cambia summaries compactos ni evidencias filtradas; solo evita falsos positivos en identidad causal opaca.
Estado: cerrada local
```
