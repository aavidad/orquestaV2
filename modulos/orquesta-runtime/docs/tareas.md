# Tareas locales: orquesta-runtime

Cada tarea debe ser pequena y cerrada.

## RUNTIME-001

```text
ID: RUNTIME-001
Objetivo: Arrancar RuntimeLaunchRequest v0 como contrato local validable, sin implementar runtime real.
Write-set:
  - modulos/orquesta-runtime/docs/contratos.md
  - modulos/orquesta-runtime/docs/tareas.md
  - modulos/orquesta-runtime/docs/pruebas.md
  - modulos/orquesta-runtime/docs/decisiones.md
  - modulos/orquesta-runtime/docs/schemas/runtime_launch_request_v0.schema.json
  - modulos/orquesta-runtime/docs/fixtures/runtime_launch_request_v0/*
Simbolo foco: RuntimeLaunchRequestV0
Contrato: RuntimeLaunch v0; RuntimeLaunchRequestV0 compartido; RuntimeLaunchAcceptedV0; RuntimeLaunchErrorV0.
Validacion:
  - jq empty sobre schema y fixtures.
  - npx --yes ajv-cli validate --spec=draft7 para fixture positivo.
  - npx --yes ajv-cli validate --spec=draft7 debe fallar para fixtures negativos.
Bloqueos:
  - Ninguno para promocion global; queda resuelta por decision del director.
  - Consumo implementado desde core, resume y runtime real requieren microtareas posteriores.
Estado: cerrada localmente el 2026-05-04.
```

## RUNTIME-002

```text
ID: RUNTIME-002
Objetivo: Crear DTOs Go y validador puro para RuntimeLaunchRequestV0, sin implementar runtime real.
Write-set:
  - modulos/orquesta-runtime/runtime_launch_request_v0.go
  - modulos/orquesta-runtime/runtime_launch_request_v0_test.go
  - modulos/orquesta-runtime/docs/contratos.md
  - modulos/orquesta-runtime/docs/tareas.md
  - modulos/orquesta-runtime/docs/pruebas.md
  - modulos/orquesta-runtime/docs/decisiones.md
Simbolo foco: RuntimeLaunchRequestV0; ValidateRuntimeLaunchRequestV0.
Contrato: RuntimeLaunchRequestV0 compartido; RuntimeLaunchErrorV0 local.
Validacion:
  - go test -count=1 ./modulos/orquesta-runtime
  - git diff --check -- modulos/orquesta-runtime
Bloqueos:
  - Ninguno para validador puro.
  - Puerto runtime real, CLI, MCP, browser, filesystem, proveedor y lanzamiento de procesos quedan fuera.
Estado: cerrada localmente el 2026-05-04.
```

## RUNTIME-003

```text
ID: RUNTIME-003
Objetivo: Definir el puerto/adaptador logico AgentLauncherInboundV0 para alinear RuntimeLaunchRequestV0 con el outbox logico LaunchRuntimeAgent target agent_launcher de orquesta-core-workflow, sin tocar nucleo ni runtime real.
Write-set:
  - modulos/orquesta-runtime/docs/contratos.md
  - modulos/orquesta-runtime/docs/contratos_agent_launcher.md
  - modulos/orquesta-runtime/docs/tareas.md
  - modulos/orquesta-runtime/docs/pruebas.md
  - modulos/orquesta-runtime/docs/decisiones.md
Simbolo foco: AgentLauncherInboundV0; LaunchRuntimeAgentRequestV0; RuntimeLaunchRequestV0.
Contrato: AgentLauncherInboundV0 local; LaunchRuntimeAgentRequestV0 compacto; RuntimeLaunchRequestV0 compartido.
Validacion:
  - go test -count=1 ./modulos/orquesta-runtime
  - git diff --check -- modulos/orquesta-runtime
Bloqueos:
  - Implementacion real queda bloqueada hasta implementar FunctionContractResolverV0, CapacityDecisionResolverV0, RuntimeBindingResolverV0 y LaunchEvidenceResolverV0.
  - No se importan tipos de orquesta-core-workflow ni se modifica orquesta-core-workflow.
Estado: cerrada documentalmente el 2026-05-04.
```

## RUNTIME-004

```text
ID: RUNTIME-004
Objetivo: Implementar DTOs Go y validador puro para AgentLauncherInboundV0 y LaunchRuntimeAgentRequestV0, sin importar orquesta-core-workflow ni construir RuntimeLaunchRequestV0 completo.
Write-set:
  - modulos/orquesta-runtime/agent_launcher_inbound_v0.go
  - modulos/orquesta-runtime/agent_launcher_inbound_v0_test.go
  - modulos/orquesta-runtime/docs/contratos.md
  - modulos/orquesta-runtime/docs/contratos_agent_launcher.md
  - modulos/orquesta-runtime/docs/tareas.md
  - modulos/orquesta-runtime/docs/pruebas.md
  - modulos/orquesta-runtime/docs/decisiones.md
Simbolo foco: AgentLauncherInboundV0; LaunchRuntimeAgentRequestV0; ValidateAgentLauncherInboundV0.
Contrato: AgentLauncherInboundV0 local; LaunchRuntimeAgentRequestV0 compacto; errores publicos de enriquecimiento.
Validacion:
  - gofmt agent_launcher_inbound_v0.go agent_launcher_inbound_v0_test.go
  - go test -count=1 ./modulos/orquesta-runtime
  - git diff --check -- modulos/orquesta-runtime
Bloqueos:
  - FunctionContractResolverV0, CapacityDecisionResolverV0, RuntimeBindingResolverV0 y LaunchEvidenceResolverV0 quedan solo como interfaces contractuales; sus implementaciones son microtareas separadas.
  - No se construye RuntimeLaunchRequestV0 completo en este corte.
Estado: cerrada localmente el 2026-05-04.
```

## RUNTIME-005

```text
ID: RUNTIME-005
Objetivo: Sanear tamano de runtime_launch_request_v0.go sin cambiar comportamiento ni contratos publicos.
Write-set:
  - modulos/orquesta-runtime/runtime_launch_request_v0.go
  - modulos/orquesta-runtime/runtime_launch_request_types_v0.go
  - modulos/orquesta-runtime/runtime_launch_request_validator_v0.go
  - modulos/orquesta-runtime/runtime_launch_request_components_v0.go
  - modulos/orquesta-runtime/runtime_launch_request_helpers_v0.go
  - modulos/orquesta-runtime/docs/tareas.md
  - modulos/orquesta-runtime/docs/pruebas.md
  - modulos/orquesta-runtime/docs/decisiones.md
Simbolo foco: RuntimeLaunchRequestV0; ValidateRuntimeLaunchRequestV0.
Contrato: RuntimeLaunchRequestV0 compartido; RuntimeLaunchErrorV0 local.
Validacion:
  - gofmt runtime_launch_request_v0.go runtime_launch_request_types_v0.go runtime_launch_request_validator_v0.go runtime_launch_request_components_v0.go runtime_launch_request_helpers_v0.go
  - go test -count=1 ./modulos/orquesta-runtime
  - git diff --check -- modulos/orquesta-runtime
  - wc -l runtime_launch_request_v0.go runtime_launch_request_types_v0.go runtime_launch_request_validator_v0.go runtime_launch_request_components_v0.go runtime_launch_request_helpers_v0.go
Bloqueos:
  - Ninguno para saneamiento local.
  - No se toca agent_launcher_inbound_v0.go.
Estado: cerrada localmente el 2026-05-04.
```

## RUNTIME-006

```text
ID: RUNTIME-006
Objetivo: Implementar DTOs Go y validador puro para AgentStopperInboundV0 y StopRuntimeAgentRequestV0, sin implementar parada real ni procesos.
Write-set:
  - modulos/orquesta-runtime/agent_stopper_inbound_v0.go
  - modulos/orquesta-runtime/agent_stopper_inbound_v0_test.go
  - modulos/orquesta-runtime/docs/contratos.md
  - modulos/orquesta-runtime/docs/contratos_agent_launcher.md
  - modulos/orquesta-runtime/docs/tareas.md
  - modulos/orquesta-runtime/docs/pruebas.md
  - modulos/orquesta-runtime/docs/decisiones.md
Simbolo foco: AgentStopperInboundV0; StopRuntimeAgentRequestV0; ValidateAgentStopperInboundV0.
Contrato: StopRuntimeAgent target agent_launcher; payload compacto de parada logica.
Validacion:
  - gofmt agent_stopper_inbound_v0.go agent_stopper_inbound_v0_test.go
  - go test -count=1 ./modulos/orquesta-runtime
  - git diff --check -- modulos/orquesta-runtime
Bloqueos:
  - No se detienen agentes ni procesos en este corte.
  - Handles, mailbox, ACK de parada y lifecycle real requieren puertos/adaptadores separados.
Estado: cerrada localmente el 2026-05-04.
```

## RUNTIME-007

```text
ID: RUNTIME-007
Objetivo: Implementar AgentProgressReportV0 como DTO Go y validador puro para reportar progreso, estancamiento, bucle o parada observada sin actuar sobre procesos.
Write-set:
  - modulos/orquesta-runtime/agent_progress_report_v0.go
  - modulos/orquesta-runtime/agent_progress_report_v0_test.go
  - modulos/orquesta-runtime/docs/contratos.md
  - modulos/orquesta-runtime/docs/tareas.md
  - modulos/orquesta-runtime/docs/pruebas.md
  - modulos/orquesta-runtime/docs/decisiones.md
Simbolo foco: AgentProgressReportV0; ValidateAgentProgressReportV0.
Contrato: AgentProgressReport v0 local.
Validacion:
  - gofmt agent_progress_report_v0.go agent_progress_report_v0_test.go
  - go test -count=1 ./modulos/orquesta-runtime
  - git diff --check -- modulos/orquesta-runtime
Bloqueos:
  - No observa runtime real, no mata procesos, no consulta proveedor/modelo/HOME/OAuth y no decide parada.
  - Integracion con director/core para consumir evidencia queda para microtareas separadas.
Estado: cerrada localmente el 2026-05-05.
```

## RUNTIME-008

```text
ID: RUNTIME-008
Objetivo: Implementar RuntimeFakeLifecycleV0 ejecutable, puro y en memoria para probar launch -> progress report -> loop_detected -> stop -> stopped sin procesos reales.
Write-set:
  - modulos/orquesta-runtime/runtime_fake_lifecycle_v0.go
  - modulos/orquesta-runtime/runtime_fake_lifecycle_v0_test.go
  - modulos/orquesta-runtime/docs/contratos.md
  - modulos/orquesta-runtime/docs/tareas.md
  - modulos/orquesta-runtime/docs/pruebas.md
  - modulos/orquesta-runtime/docs/decisiones.md
Simbolo foco: RuntimeFakeLifecycleV0; RuntimeFakeLifecycleSnapshotV0; RuntimeFakeLifecycleErrorV0.
Contrato: fake local en memoria sobre AgentLauncherInboundV0, AgentProgressReportV0 y AgentStopperInboundV0.
Validacion:
  - gofmt runtime_fake_lifecycle_v0.go runtime_fake_lifecycle_v0_test.go
  - go test -count=1 ./modulos/orquesta-runtime
  - git diff --check -- modulos/orquesta-runtime
Bloqueos:
  - No implementa runtime real, procesos, proveedor/modelo/HOME/OAuth, transporte ni import de orquesta-core-workflow.
  - Launch repetido y progress tardio se tratan de forma idempotente para no reabrir agentes parados.
Estado: cerrada localmente el 2026-05-05.
```

## RUNTIME-009

```text
ID: RUNTIME-009
Objetivo: Anadir una prueba progresiva pequena de RuntimeFakeLifecycleV0 con dos agentes independientes en el mismo run.
Write-set:
  - modulos/orquesta-runtime/runtime_fake_lifecycle_multi_agent_v0_test.go
  - modulos/orquesta-runtime/docs/tareas.md
  - modulos/orquesta-runtime/docs/pruebas.md
  - modulos/orquesta-runtime/docs/decisiones.md
Simbolo foco: RuntimeFakeLifecycleV0; AgentLauncherInboundV0; AgentProgressReportV0; AgentStopperInboundV0.
Contrato: fake local en memoria con aislamiento por agent_request_id y run_id consistente.
Validacion:
  - gofmt runtime_fake_lifecycle_multi_agent_v0_test.go
  - go test -count=1 ./modulos/orquesta-runtime
  - git diff --check -- modulos/orquesta-runtime
  - wc -l runtime_fake_lifecycle_multi_agent_v0_test.go docs/tareas.md docs/pruebas.md docs/decisiones.md
Bloqueos:
  - No introduce runtime real, procesos, proveedor/modelo/HOME/OAuth, DB, transporte ni imports de otros modulos.
  - No cambia codigo productivo; si el test descubre un bug real, debe documentarse antes de ampliar write-set.
Estado: cerrada localmente el 2026-05-05.
```

## RUNTIME-010

```text
ID: RUNTIME-010
Objetivo: Cerrar compatibilidad runtime con AgentHomeV0 consumiendo RuntimeBindingV0 solo por refs opacas, sin importar orquesta-capacity ni duplicar politica/capacidad.
Write-set:
  - modulos/orquesta-runtime/runtime_launch_request_agent_home_v0_test.go
  - modulos/orquesta-runtime/runtime_launch_request_components_v0.go
  - modulos/orquesta-runtime/runtime_launch_request_helpers_v0.go
  - modulos/orquesta-runtime/docs/contratos.md
  - modulos/orquesta-runtime/docs/tareas.md
  - modulos/orquesta-runtime/docs/pruebas.md
  - modulos/orquesta-runtime/docs/decisiones.md
Simbolo foco: RuntimeLaunchRequestV0; RuntimeBindingV0.
Contrato: RuntimeLaunchRequestV0 local; binding compatible con AgentHomeV0 por refs opacas.
Validacion:
  - gofmt runtime_launch_request_agent_home_v0_test.go runtime_launch_request_components_v0.go runtime_launch_request_helpers_v0.go
  - go test -count=1 ./modulos/orquesta-runtime
  - git diff --check -- modulos/orquesta-runtime
Bloqueos:
  - No importa orquesta-capacity ni contratos internos de capacity.
  - No introduce runtime real, procesos, proveedor, HOME real, OAuth, DB, filesystem productivo, colas ni goroutines.
  - Capacity sigue siendo propietario de politica/capacidad; runtime solo valida refs opacas y rechaza detalles reales.
Estado: cerrada localmente el 2026-05-05.
```

## RUNTIME-011

```text
ID: RUNTIME-011
Objetivo: Implementar un conector runtime real de proceso local controlado para pruebas e2e, sin proveedor, HOME real, OAuth, shell ni entorno heredado.
Write-set:
  - modulos/orquesta-runtime/process_runtime_connector_types_v0.go
  - modulos/orquesta-runtime/process_runtime_connector_v0.go
  - modulos/orquesta-runtime/process_runtime_connector_v0_test.go
  - modulos/orquesta-runtime/README.md
  - modulos/orquesta-runtime/docs/contratos.md
  - modulos/orquesta-runtime/docs/tareas.md
  - modulos/orquesta-runtime/docs/pruebas.md
  - modulos/orquesta-runtime/docs/decisiones.md
Simbolo foco: ProcessRuntimeConnectorV0; ProcessRuntimeLaunchRequestV0; ProcessRuntimeSnapshotV0.
Contrato: adaptador runtime local versionado con LaunchV0, StopV0 y SnapshotV0.
Validacion:
  - gofmt process_runtime_connector_types_v0.go process_runtime_connector_v0.go process_runtime_connector_v0_test.go
  - go test -count=1 ./modulos/orquesta-runtime
  - git diff --check -- modulos/orquesta-runtime
Bloqueos:
  - No integra core, capacity, provider, OAuth, HOME real, DB, red, shell ni filesystem productivo.
  - command_path, args, env y working_dir son configuracion explicita de test/config y no forman parte del snapshot publico.
Estado: cerrada localmente el 2026-05-05.
```

## RUNTIME-012

```text
ID: RUNTIME-012
Objetivo: Exigir ContextBundleV0 resuelto antes de aceptar RuntimeLaunchRequestV0 y declararlo como dependencia de AgentLauncherInboundV0.
Write-set:
  - modulos/orquesta-runtime/agent_launcher_inbound_v0.go
  - modulos/orquesta-runtime/agent_launcher_inbound_v0_test.go
  - modulos/orquesta-runtime/runtime_launch_request_types_v0.go
  - modulos/orquesta-runtime/runtime_launch_request_validator_v0.go
  - modulos/orquesta-runtime/runtime_launch_request_components_v0.go
  - modulos/orquesta-runtime/runtime_launch_request_helpers_v0.go
  - modulos/orquesta-runtime/runtime_launch_request_v0_test.go
  - modulos/orquesta-runtime/docs/contratos.md
  - modulos/orquesta-runtime/docs/tareas.md
  - modulos/orquesta-runtime/docs/pruebas.md
  - modulos/orquesta-runtime/docs/decisiones.md
Simbolo foco: RuntimeLaunchRequestV0; AgentLauncherResolvedDependenciesV0; ContextBundleV0.
Contrato: RuntimeLaunchRequestV0 consume ContextBundleV0 por contrato publico de orquesta-context.
Validacion:
  - gofmt sobre Go tocados
  - go test -count=1 ./modulos/orquesta-runtime ./modulos/orquesta-context
  - go test -race -count=1 ./modulos/orquesta-context ./modulos/orquesta-core-workflow ./modulos/orquesta-director ./modulos/orquesta-runtime ./modulos/orquesta-persistence ./modulos/orquesta-capacity ./modulos/orquesta-e2e
Bloqueos:
  - No materializa refs ni lee filesystem; eso queda para adaptadores futuros.
  - No introduce proveedor, HOME real, OAuth, DB, prompts completos ni transcripts.
Estado: cerrada localmente el 2026-05-05.
```

## RUNTIME-013

```text
ID: RUNTIME-013
Objetivo: Implementar y documentar AgentStartPacketV0 como paquete neutral de arranque para agentes, construido desde RuntimeLaunchRequestV0 y ContextMaterializedBundleV0 sin exponer proveedor, modelo, HOME, OAuth ni referencias de credencial.
Write-set:
  - modulos/orquesta-runtime/agent_start_packet_types_v0.go
  - modulos/orquesta-runtime/agent_start_packet_v0.go
  - modulos/orquesta-runtime/agent_start_packet_v0_test.go
  - modulos/orquesta-runtime/docs/contratos.md
  - modulos/orquesta-runtime/docs/tareas.md
  - modulos/orquesta-runtime/docs/pruebas.md
  - modulos/orquesta-runtime/docs/decisiones.md
  - modulos/CONTRATOS.md
Simbolo foco: AgentStartPacketV0; BuildAgentStartPacketV0; ContextMaterializedBundleV0.
Contrato: paquete de arranque neutral consumible por conectores de agente futuros.
Validacion:
  - go test -count=1 ./modulos/orquesta-runtime
  - git diff --check -- modulos/orquesta-runtime/docs/contratos.md modulos/orquesta-runtime/docs/tareas.md modulos/orquesta-runtime/docs/pruebas.md modulos/orquesta-runtime/docs/decisiones.md modulos/CONTRATOS.md
Bloqueos:
  - No implementa conectores de proveedor ni transporte real.
  - No modifica schemas ni fixtures en este corte.
  - Cada conector concreto debe traducir el paquete por puerto hexagonal sin DB hardcodeada ni secretos.
Estado: cerrada localmente el 2026-05-05.
```

## RUNTIME-014

```text
ID: RUNTIME-014
Objetivo: Preparar la sustitucion del proceso hijo controlado por un conector de agente real opt-in, con perfil y launch spec puros, sin hardcodear proveedor/modelo/HOME/OAuth ni tocar core.
Write-set:
  - modulos/orquesta-runtime/external_agent_connector_types_v0.go
  - modulos/orquesta-runtime/external_agent_connector_v0.go
  - modulos/orquesta-runtime/external_agent_connector_v0_test.go
  - modulos/orquesta-runtime/docs/contratos.md
  - modulos/orquesta-runtime/docs/contratos_agent_launcher.md
  - modulos/orquesta-runtime/docs/tareas.md
  - modulos/orquesta-runtime/docs/pruebas.md
  - modulos/orquesta-runtime/docs/decisiones.md
Simbolo foco: ExternalAgentConnectorProfileV0; ExternalAgentLaunchSpecV0; BuildExternalAgentLaunchSpecV0.
Contrato: conector de agente real opt-in por refs opacas y comando resuelto por adaptador superior.
Validacion:
  - gofmt external_agent_connector_types_v0.go external_agent_connector_v0.go external_agent_connector_v0_test.go
  - go test -count=1 ./modulos/orquesta-runtime
  - git diff --check -- modulos/orquesta-runtime
Bloqueos:
  - No ejecuta comandos ni introduce shell, PATH heredado, HOME real, red externa, proveedor/modelo concreto, OAuth, tokens ni secrets.
  - No sustituye todavia ProcessRuntimeConnectorV0; solo fija la frontera para un adaptador posterior.
Estado: cerrada localmente el 2026-05-07.
```

## RUNTIME-015

```text
ID: RUNTIME-015
Objetivo: Implementar el adaptador opt-in que convierte ExternalAgentLaunchSpecV0 en lanzamiento de proceso mediante resolver y runtime inyectados, sin hardcodear proveedor/modelo/HOME/OAuth.
Write-set:
  - modulos/orquesta-runtime/external_agent_process_adapter_types_v0.go
  - modulos/orquesta-runtime/external_agent_process_adapter_v0.go
  - modulos/orquesta-runtime/external_agent_process_adapter_v0_test.go
  - modulos/orquesta-runtime/docs/contratos.md
  - modulos/orquesta-runtime/docs/contratos_agent_launcher.md
  - modulos/orquesta-runtime/docs/tareas.md
  - modulos/orquesta-runtime/docs/pruebas.md
  - modulos/orquesta-runtime/docs/decisiones.md
Simbolo foco: LaunchExternalAgentProcessV0; ExternalAgentProcessCommandResolverV0; ExternalAgentProcessRuntimePortV0.
Contrato: puente opt-in ExternalAgentLaunchSpecV0 -> ProcessRuntimeLaunchRequestV0 por puertos.
Validacion:
  - gofmt external_agent_process_adapter_types_v0.go external_agent_process_adapter_v0.go external_agent_process_adapter_v0_test.go
  - go test -count=1 ./modulos/orquesta-runtime
  - git diff --check -- modulos/orquesta-runtime
Bloqueos:
  - El conector real de Codex/Ollama/vLLM/API sigue siendo configuracion/adaptador externo opt-in.
  - El core no recibe command_path, args, env, working_dir, provider, modelo, HOME ni credenciales.
Estado: cerrada localmente el 2026-05-07.
```

## RUNTIME-016

```text
ID: RUNTIME-016
Objetivo: Anadir contrato puro de heartbeat de proceso a AgentProgressReportV0 para que director/scheduler distingan trabajo vivo, sin progreso y bucle sin inspeccionar internals del agente.
Write-set:
  - modulos/orquesta-runtime/agent_progress_heartbeat_v0.go
  - modulos/orquesta-runtime/agent_progress_heartbeat_v0_test.go
  - modulos/orquesta-runtime/README.md
  - modulos/orquesta-runtime/docs/contratos.md
  - modulos/orquesta-runtime/docs/tareas.md
  - modulos/orquesta-runtime/docs/pruebas.md
  - modulos/orquesta-runtime/docs/decisiones.md
Simbolo foco: AgentProgressHeartbeatV0; AgentProgressHeartbeatPolicyV0; BuildAgentProgressReportFromHeartbeatV0.
Contrato: heartbeat compacto + ProcessRuntimeSnapshotV0 publico -> AgentProgressReportV0.
Validacion:
  - gofmt agent_progress_heartbeat_v0.go agent_progress_heartbeat_v0_test.go
  - go test -count=1 ./modulos/orquesta-runtime
Bloqueos:
  - No cambia LaunchV0, no mata procesos, no consulta proveedor/modelo/HOME/OAuth, no lee DB ni transcripts.
  - La integracion real del scheduler para emitir heartbeats queda para microtarea separada.
Estado: cerrada localmente el 2026-05-07.
```

## RUNTIME-017

```text
ID: RUNTIME-017
Objetivo: Implementar la transformacion pura AgentLauncherInboundV0 + dependencias resueltas -> RuntimeLaunchRequestV0 validado, sin proveedor, DB ni runtime real.
Write-set:
  - modulos/orquesta-runtime/agent_launcher_transform_v0.go
  - modulos/orquesta-runtime/agent_launcher_transform_helpers_v0.go
  - modulos/orquesta-runtime/agent_launcher_transform_v0_test.go
  - modulos/orquesta-runtime/docs/contratos_agent_launcher.md
  - modulos/orquesta-runtime/docs/tareas.md
  - modulos/orquesta-runtime/docs/pruebas.md
  - modulos/orquesta-runtime/docs/decisiones.md
Simbolo foco: LaunchRuntimeAgentToRuntimeLaunchRequestV0.
Contrato: AgentLauncherInboundV0; AgentLauncherResolvedDependenciesV0; RuntimeLaunchRequestV0.
Validacion:
  - gofmt agent_launcher_transform_v0.go agent_launcher_transform_helpers_v0.go agent_launcher_transform_v0_test.go
  - go test ./modulos/orquesta-runtime -run 'TestLaunchRuntimeAgentToRuntimeLaunchRequestV0|TestValidateAgentLauncherResolvedDependenciesV0|TestBuildAgentStartPacketV0|TestBuildExternalAgentLaunchSpecV0' -count=1
Bloqueos:
  - No resuelve FunctionContract, CapacityDecision, RuntimeBinding, EvidenceRefs ni ContextBundle; exige que lleguen por puertos externos.
  - No materializa contexto ni lanza procesos; esos pasos siguen en orquesta-context y conectores runtime.
Estado: cerrada localmente el 2026-05-08.
```

## RUNTIME-018

```text
ID: RUNTIME-018
Objetivo: Crear builder generico de ExternalAgentConnectorProfileV0 cerrado por defecto desde refs opacas para que los conectores no dupliquen politica de seguridad.
Write-set:
  - modulos/orquesta-runtime/external_agent_profile_builder_v0.go
  - modulos/orquesta-runtime/external_agent_profile_builder_v0_test.go
  - modulos/orquesta-runtime/docs/tareas.md
  - modulos/orquesta-runtime/docs/pruebas.md
  - modulos/orquesta-runtime/docs/decisiones.md
Simbolo foco: BuildClosedExternalAgentConnectorProfileV0; ClosedExternalAgentSecurityPolicyV0.
Contrato: ExternalAgentConnectorProfileV0.
Validacion:
  - gofmt external_agent_profile_builder_v0.go external_agent_profile_builder_v0_test.go
  - go test ./modulos/orquesta-runtime -run 'TestBuildClosedExternalAgentConnectorProfileV0|TestBuildExternalAgentLaunchSpecV0|TestExternalAgentProcessAdapterV0' -count=1
Bloqueos:
  - No resuelve refs a comandos reales ni proveedores.
  - No introduce defaults de Codex, Claude, Gemini, Ollama, vLLM ni DB.
Estado: cerrada localmente el 2026-05-08.
```

## RUNTIME-019

```text
ID: RUNTIME-019
Objetivo: Propagar en AgentStartPacketV0 la politica publica de saneamiento de contexto cuando ContextMaterializedBundleV0 trae evidencia.
Write-set:
  - modulos/orquesta-runtime/agent_start_packet_v0.go
  - modulos/orquesta-runtime/agent_start_packet_v0_test.go
  - modulos/orquesta-runtime/docs/tareas.md
  - modulos/orquesta-runtime/docs/pruebas.md
Simbolo foco: BuildAgentStartPacketV0.
Contrato: AgentStartPacketV0; ContextSanitizationEvidenceV0.
Validacion:
  - go test -count=1 ./modulos/orquesta-runtime
Bloqueos:
  - Runtime no materializa contexto ni ejecuta sanitizadores; solo conserva evidencia y politica compacta.
Estado: cerrada localmente.
```

## RUNTIME-020

```text
ID: RUNTIME-020
Objetivo: Cubrir E2E contractual de runtime externo neutral no-Codex desde orden compacta hasta launch/progress/stop por puertos inyectados.
Write-set:
  - modulos/orquesta-runtime/runtime_neutral_e2e_v0_test.go
  - modulos/orquesta-runtime-worktree
  - modulos/orquesta-runtime/docs/tareas.md
  - modulos/orquesta-runtime/docs/pruebas.md
Simbolo foco: LaunchRuntimeAgentToRuntimeLaunchRequestV0; BuildExternalAgentLaunchSpecV0; LaunchExternalAgentProcessV0; BuildAgentProgressReportFromHeartbeatV0.
Contrato: runtime externo opt-in sin proveedor concreto ni detalle operacional en resultados publicos.
Validacion:
  - go test -count=1 ./modulos/orquesta-runtime
  - go test -count=1 ./modulos/orquesta-runtime ./modulos/orquesta-runtime-worktree
Bloqueos:
  - Usa proceso controlado del binario de test; no valida CLI real, proveedor, cuota, red ni transporte productivo.
  - La verificacion de worktree comprueba paths relativos y write-set, no Git ni merge productivo.
Estado: cerrada localmente el 2026-05-24.
```

## Plantilla

```text
ID:
Objetivo:
Write-set:
Simbolo foco:
Contrato:
Validacion:
Bloqueos:
Estado:
```
