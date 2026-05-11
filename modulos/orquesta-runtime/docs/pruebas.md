# Pruebas locales: orquesta-runtime

Registra pruebas obligatorias del modulo.

## RuntimeLaunchRequest v0

```text
Caso: Sintaxis JSON de schema y fixtures RuntimeLaunchRequest v0
Tipo: contract
Comando: jq empty docs/schemas/runtime_launch_request_v0.schema.json docs/fixtures/runtime_launch_request_v0/request_minima_valida.json docs/fixtures/runtime_launch_request_v0/request_sin_capacity_invalida.json docs/fixtures/runtime_launch_request_v0/request_home_ref_no_opaca_invalida.json
Evidencia esperada: comando sin salida y exit code 0.
Ultima ejecucion: 2026-05-04, OK.
Riesgos: jq valida sintaxis, no semantica de JSON Schema.
```

```text
Caso: Request minima valida cumple RuntimeLaunchRequestV0
Tipo: contract
Comando: npx --yes ajv-cli validate --spec=draft7 -s docs/schemas/runtime_launch_request_v0.schema.json -d docs/fixtures/runtime_launch_request_v0/request_minima_valida.json
Evidencia esperada: fixture valido y exit code 0.
Ultima ejecucion: 2026-05-04, OK.
Riesgos: JSON Schema valida estructura; invariantes relacionales con CapacityDecisionV0 y FunctionContractV0 requieren harness posterior.
```

```text
Caso: Request sin capacity_decision falla
Tipo: contract
Comando: ! npx --yes ajv-cli validate --spec=draft7 -s docs/schemas/runtime_launch_request_v0.schema.json -d docs/fixtures/runtime_launch_request_v0/request_sin_capacity_invalida.json
Evidencia esperada: falla por required capacity_decision.
Ultima ejecucion: 2026-05-04, OK.
Riesgos: el mapeo exacto a capacity_decision_requerida queda para el validador de aplicacion.
```

```text
Caso: Request con home_ref no opaca falla
Tipo: contract
Comando: ! npx --yes ajv-cli validate --spec=draft7 -s docs/schemas/runtime_launch_request_v0.schema.json -d docs/fixtures/runtime_launch_request_v0/request_home_ref_no_opaca_invalida.json
Evidencia esperada: falla por pattern de opaque_ref en runtime_binding.home_ref.
Ultima ejecucion: 2026-05-04, OK.
Riesgos: el schema detecta forma no opaca; la deteccion semantica de proveedor/modelo real se cubre en el validador Go local.
```

```text
Caso: RUNTIME-010 RuntimeBindingV0 compatible con AgentHomeV0 solo por refs opacas
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-runtime
Evidencia esperada: paquete orquesta/modulos/orquesta-runtime OK; rechaza HOME real, account/email en credential_ref, token, proveedor hardcodeado, modelo hardcodeado y home_ref no opaco.
Ultima ejecucion: 2026-05-05, OK.
Riesgos: prueba el contrato local puro; no resuelve AgentHomeV0 real, no importa capacity y no ejecuta runtime/proveedor/HOME/OAuth.
```

```text
Caso: DTOs y validador puro RuntimeLaunchRequestV0
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-runtime
Evidencia esperada: paquete orquesta/modulos/orquesta-runtime OK; acepta request minimo con ContextBundleV0 valido; rechaza function_contract ausente, capacity ausente, context_bundle ausente, context_bundle de otro modulo, HOME real, refs no opacas, secretos, safety abierta y modelo distinto al de capacity.
Ultima ejecucion: 2026-05-05, OK.
Riesgos: no arranca runtime real ni materializa refs; cubre solo validacion local pura de DTOs.
```

```text
Caso: Saneamiento de tamano RuntimeLaunchRequestV0
Tipo: unit
Comando:
  - cd modulos/orquesta-runtime && gofmt -w runtime_launch_request_v0.go runtime_launch_request_types_v0.go runtime_launch_request_validator_v0.go runtime_launch_request_components_v0.go runtime_launch_request_helpers_v0.go
  - go test -count=1 ./modulos/orquesta-runtime
  - git diff --check -- modulos/orquesta-runtime
  - cd modulos/orquesta-runtime && wc -l runtime_launch_request_v0.go runtime_launch_request_types_v0.go runtime_launch_request_validator_v0.go runtime_launch_request_components_v0.go runtime_launch_request_helpers_v0.go
Evidencia esperada: gofmt sin cambios pendientes, paquete orquesta/modulos/orquesta-runtime OK, diff sin whitespace errors y ficheros Go tocados por debajo de 300-350 lineas.
Ultima ejecucion: 2026-05-04, OK.
Riesgos: refactorizacion mecanica; la cobertura existente protege los errores contractuales principales, pero no hay harness exhaustivo de equivalencia campo a campo.
```

## AgentLauncherInbound v0

```text
Caso: Contrato documental AgentLauncherInboundV0
Tipo: contract
Comando: revision documental de docs/contratos_agent_launcher.md
Evidencia esperada: el contrato define target `agent_launcher`, operacion `LaunchRuntimeAgent`, payload compacto real `LaunchRuntimeAgentRequestV0` de core-workflow y reglas para construir `RuntimeLaunchRequestV0` solo tras resolver FunctionContract, CapacityDecision, RuntimeBinding y evidencias por puertos versionados.
Ultima ejecucion: 2026-05-04, OK.
Riesgos: no existe harness automatico para validar compatibilidad con outbox de orquesta-core-workflow; implementar el adaptador requiere microtareas separadas para resolvers.
```

```text
Caso: DTOs y validador puro AgentLauncherInboundV0
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-runtime
Evidencia esperada: paquete orquesta/modulos/orquesta-runtime OK; acepta orden compacta valida, rechaza target/operacion/payload invalidos, refs no opacas, secretos y dependencias externas ausentes para FunctionContractV0, CapacityDecisionV0, RuntimeBindingV0, evidencias y ContextBundleV0.
Ultima ejecucion: 2026-05-05, OK.
Riesgos: no resuelve FunctionContractV0, CapacityDecisionV0, RuntimeBindingV0, ContextBundleV0 ni evidencias de transporte; solo declara interfaces y errores contractuales.
```

## AgentStopperInbound v0

```text
Caso: DTOs y validador puro AgentStopperInboundV0
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-runtime
Evidencia esperada: paquete orquesta/modulos/orquesta-runtime OK; acepta orden compacta valida, rechaza target/operacion/payload invalidos, refs no opacas y secretos.
Ultima ejecucion: 2026-05-04, OK.
Riesgos: no detiene agentes, no mata procesos, no escribe mailbox ni valida handles reales; cubre solo validacion local pura de DTOs.
```

## AgentProgressReport v0

```text
Caso: DTOs y validador puro AgentProgressReportV0
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-runtime
Evidencia esperada: paquete orquesta/modulos/orquesta-runtime OK; acepta reporte valido, mantiene JSON round-trip y rechaza status invalido, refs no opacas, secretos y loop_detected sin contadores.
Ultima ejecucion: 2026-05-05, OK.
Riesgos: no observa runtime real, no detecta bucles en procesos vivos y no decide parada; cubre solo validacion local pura del reporte.
```

```text
Caso: RUNTIME-016 heartbeat de proceso a reporte compacto consumible por director/scheduler
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-runtime
Evidencia esperada: paquete orquesta/modulos/orquesta-runtime OK; BuildAgentProgressReportFromHeartbeatV0 distingue trabajo vivo con avance, heartbeat vivo sin progreso y bucle por counters/refs opacas sin provider, DB, command_path, HOME, OAuth, PID ni token en policy o reporte.
Ultima ejecucion: 2026-05-07, OK.
Riesgos: no observa internals reales del agente, no lee transcripts, no decide parada y no cambia launch real; solo normaliza una senal compacta desde snapshot publico de proceso y heartbeat.
```

## RuntimeFakeLifecycle v0

```text
Caso: Lifecycle fake launch/progress/loop/stop/stopped
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-runtime
Evidencia esperada: paquete orquesta/modulos/orquesta-runtime OK; cubre ciclo completo, progress/stop de agente inexistente, run mismatch, stop repetido idempotente, launch repetido sin reapertura, progress tardio sin reapertura e inbound invalido.
Ultima ejecucion: 2026-05-05, OK.
Riesgos: fake puro en memoria; no prueba procesos reales, proveedor/modelo/HOME/OAuth ni integracion con orquesta-core-workflow.
```

```text
Caso: RUNTIME-009 lifecycle fake con dos agentes independientes en el mismo run
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-runtime
Evidencia esperada: paquete orquesta/modulos/orquesta-runtime OK; lanza dos AgentLauncherInboundV0 con agent_request_id, correlation_id e idempotency_key distintos en el mismo run; loop_detected y stop afectan solo al agente 1; el agente 2 conserva snapshot progressing estable, sin heredar stop_ref ni last_report_ref del agente 1.
Ultima ejecucion: 2026-05-05, OK.
Riesgos: prueba aislamiento del fake en memoria; no ejecuta runtime real, proveedor/modelo/HOME/OAuth, DB, procesos ni transporte.
```

## ProcessRuntimeConnector v0

- `TestProcessRuntimeConnectorV0NoHeredaEntornoPadre` valida que un entorno
  padre contaminante no aparece en el proceso hijo cuando `Env` no lo declara.
- `TestProcessRuntimeConnectorV0AceptaPathOperativoExplicito` valida que un
  conector opt-in puede pasar `PATH=` privado para herramientas reales sin
  heredar el resto del entorno.
- `TestProcessRuntimeConnectorV0RechazaPathOperativoInseguro` valida que `PATH`
  vacio, relativo o con marcas de credencial sigue bloqueado.

```text
Caso: RUNTIME-011 lanza y para un proceso local real controlado
Tipo: integration
Comando: go test -count=1 ./modulos/orquesta-runtime
Evidencia esperada: el propio binario de test se lanza como proceso hijo con env explicito ORQUESTA_RUNTIME_TEST_CHILD=wait, snapshot running, StopV0 usa contexto y senal portable, y snapshot final stopped con stop_ref opaca.
Ultima ejecucion: 2026-05-05, OK.
Riesgos: prueba proceso local en entorno de test; no valida integracion con core, provider, OAuth, HOME real, red ni DB.
```

```text
Caso: RUNTIME-011 stop idempotente y proceso que sale solo
Tipo: integration
Comando: go test -count=1 ./modulos/orquesta-runtime
Evidencia esperada: StopV0 repetido conserva stop_ref; un hijo con ORQUESTA_RUNTIME_TEST_CHILD=exit queda stopped sin inventar stop_ref.
Ultima ejecucion: 2026-05-05, OK.
Riesgos: stdout/stderr se descartan y no se comprueba contenido de salida; el contrato publico solo expone refs opacas y estado.
```

## AgentStartPacket v0

```text
Caso: RUNTIME-013 paquete neutral de arranque para agentes
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-runtime
Evidencia esperada: paquete orquesta/modulos/orquesta-runtime OK; BuildAgentStartPacketV0 produce AgentStartPacketV0 valido desde RuntimeLaunchRequestV0 y ContextMaterializedBundleV0 coincidentes, incluye task/context/delivery_refs/policies y no serializa provider/model/HOME/OAuth/credential refs ni secretos.
Ultima ejecucion: 2026-05-05, OK.
Riesgos: no ejecuta conectores reales ni transporte de agente; valida la frontera neutral antes de adaptar CLI/MCP/API/local/remoto.
```

```text
Caso: RUNTIME-013 rechaza contexto materializado ausente o de otro bundle
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-runtime
Evidencia esperada: paquete orquesta/modulos/orquesta-runtime OK; devuelve context_materialized_bundle_requerido si falta contexto y context_materialized_bundle_invalido si bundle_ref/work_order_ref/target_module no coinciden.
Ultima ejecucion: 2026-05-05, OK.
Riesgos: no materializa refs ni lee filesystem; ContextMaterializedBundleV0 sigue siendo responsabilidad de orquesta-context y sus conectores.
```

## ExternalAgentConnector v0

```text
Caso: RUNTIME-014 perfil de conector externo opt-in cerrado
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-runtime
Evidencia esperada: paquete orquesta/modulos/orquesta-runtime OK; ValidateExternalAgentConnectorProfileV0 acepta refs opacas, connector_ref, runtime_kind, launch_mode, comando por refs y politica cerrada.
Ultima ejecucion: 2026-05-07, OK.
Riesgos: no ejecuta comando real ni comprueba disponibilidad del ejecutable; el comando queda como refs ya resueltas por adaptador superior.
```

```text
Caso: RUNTIME-014 rechaza detalles operacionales prohibidos
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-runtime
Evidencia esperada: paquete orquesta/modulos/orquesta-runtime OK; rechaza provider/model concretos, HOME real, OAuth, token, secret, ruta absoluta peligrosa, transcript y politica abierta.
Ultima ejecucion: 2026-05-07, OK.
Riesgos: validador local por patrones; conectores reales futuros deben mantener allowlists propias sin ampliar payloads sensibles.
```

```text
Caso: RUNTIME-014 build de ExternalAgentLaunchSpecV0 no filtra refs operacionales al agente
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-runtime
Evidencia esperada: paquete orquesta/modulos/orquesta-runtime OK; BuildExternalAgentLaunchSpecV0 convierte RuntimeLaunchRequestV0 + AgentStartPacketV0 + profile en spec opt-in y no serializa provider_ref, model_ref, home_ref, credential_ref ni OAuth.
Ultima ejecucion: 2026-05-07, OK.
Riesgos: no sustituye ProcessRuntimeConnectorV0 ni arranca runtime real; solo prepara la frontera contractual.
```

```text
Caso: RUNTIME-015 adaptador de agente externo lanza proceso por puertos inyectados
Tipo: integration
Comando: go test -count=1 ./modulos/orquesta-runtime
Evidencia esperada: paquete orquesta/modulos/orquesta-runtime OK; LaunchExternalAgentProcessV0 acepta ExternalAgentLaunchSpecV0 valido, usa ExternalAgentProcessCommandResolverV0 inyectado para obtener ProcessRuntimeLaunchRequestV0 y lanza ProcessRuntimeConnectorV0 sin filtrar detalles operacionales en el resultado.
Ultima ejecucion: 2026-05-07, OK.
Riesgos: el proceso de prueba es el binario de test controlado; un proveedor real sigue requiriendo resolver/configuracion opt-in fuera del nucleo.
```

```text
Caso: RUNTIME-015 bloqueos de frontera del adaptador externo
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-runtime
Evidencia esperada: paquete orquesta/modulos/orquesta-runtime OK; spec invalida, resolver/runtime ausentes, issues del resolver, request operacional invalida y fallo del runtime devuelven status blocked con errores i18n versionados.
Ultima ejecucion: 2026-05-07, OK.
Riesgos: no valida un CLI/proveedor concreto; valida que el nucleo no lo hardcodea y que el fallo queda en contrato publico.
```

## AgentLauncher transform v0

```text
Caso: RUNTIME-017 transforma orden compacta en RuntimeLaunchRequestV0 valido
Tipo: unit
Comando: go test ./modulos/orquesta-runtime -run 'TestLaunchRuntimeAgentToRuntimeLaunchRequestV0|TestValidateAgentLauncherResolvedDependenciesV0|TestBuildAgentStartPacketV0|TestBuildExternalAgentLaunchSpecV0' -count=1
Evidencia esperada: LaunchRuntimeAgentToRuntimeLaunchRequestV0 genera RuntimeLaunchRequestV0 valido, conserva request_id/correlation_id/idempotency_key, fija source/delivery/safety contractuales y no resuelve proveedor ni DB.
Ultima ejecucion: 2026-05-08, OK.
Riesgos: usa dependencias ya resueltas por puertos falsos de test; la integracion con resolvers productivos queda fuera de este corte.
```

```text
Caso: RUNTIME-017 bloquea dependencias incompletas y runtime invalido
Tipo: unit
Comando: go test ./modulos/orquesta-runtime -run 'TestLaunchRuntimeAgentToRuntimeLaunchRequestV0' -count=1
Evidencia esperada: devuelve errores de enriquecimiento si faltan FunctionContract, CapacityDecision, RuntimeBinding, EvidenceRefs o ContextBundle; si la request final viola RuntimeLaunchRequestV0, propaga AgentLauncherRuntimeLaunchInvalidaV0 con evidencia versionada.
Ultima ejecucion: 2026-05-08, OK.
Riesgos: no prueba materializacion de contexto, CLI real ni proceso real.
```

## ExternalAgent profile builder v0

```text
Caso: RUNTIME-018 builder de perfil externo cerrado
Tipo: unit
Comando: go test ./modulos/orquesta-runtime -run 'TestBuildClosedExternalAgentConnectorProfileV0|TestBuildExternalAgentLaunchSpecV0|TestExternalAgentProcessAdapterV0' -count=1
Evidencia esperada: BuildClosedExternalAgentConnectorProfileV0 crea ExternalAgentConnectorProfileV0 valido con politica cerrada y refs opacas; las refs invalidas siguen siendo rechazadas por el validador.
Ultima ejecucion: 2026-05-08, OK.
Riesgos: no resuelve comandos reales ni proveedor; solo estandariza la politica cerrada comun para conectores opt-in.
```

## Plantilla

```text
Caso:
Tipo: unit | contract | integration | smoke
Comando:
Evidencia esperada:
Ultima ejecucion:
Riesgos:
```
