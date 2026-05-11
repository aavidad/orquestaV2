# Contratos locales: orquesta-runtime

Registra puertos, DTOs y eventos que `orquesta-runtime` expone o consume.

## AgentLauncherInbound v0

```text
Nombre: AgentLauncherInboundV0
Tipo: puerto_entrada_logico
Version: v0
Propietario: orquesta-runtime
Productor autorizado: orquesta-core-workflow, via outbox logico `LaunchRuntimeAgent` con target `agent_launcher`.
Contrato canonico local:
  - docs/contratos_agent_launcher.md
Entrada compacta:
  - LaunchRuntimeAgentRequestV0
Salida contractual:
  - RuntimeLaunchRequestV0 completo, solo despues de enriquecer la orden compacta por puertos de resolucion autorizados.
Invariantes:
  - No implementa runtime real ni importa orquesta-core-workflow.
  - No decide capacidad, modelo, proveedor, HOME, cuota, fase ni asignacion.
  - El payload compacto actual contiene agent_request_id, run_id, phase_id, task_ref, capacity_request_ref, role, summary y evidence_refs.
  - No inventa FunctionContractV0, CapacityDecisionV0, binding/runtime refs ni evidencias.
  - Si falta una ref externa necesaria, devuelve error contractual; los resolvers nuevos se versionan antes de implementar.
Pruebas de contrato:
  - Revision documental RUNTIME-003.
  - DTOs y validador puro RUNTIME-004.
  - El resultado futuro debe pasar ValidateRuntimeLaunchRequestV0.
```

## AgentStopperInbound v0

```text
Nombre: AgentStopperInboundV0
Tipo: puerto_entrada_logico
Version: v0
Propietario: orquesta-runtime
Productor autorizado: orquesta-core-workflow, via outbox logico `StopRuntimeAgent` con target `agent_launcher`.
Contrato canonico local:
  - docs/contratos_agent_launcher.md
Entrada compacta:
  - StopRuntimeAgentRequestV0
Salida contractual:
  - ok: validacion local pura de la intencion de parada logica.
  - error: AgentStopperInboundErrorV0.
Invariantes:
  - No implementa parada real, runtime real, procesos, CLI, MCP, HTTP, Docker, filesystem, proveedor, HOME ni credenciales.
  - No importa tipos ni paquetes de orquesta-core-workflow.
  - Solo acepta target_port `agent_launcher` y message_type `StopRuntimeAgent`.
  - agent_request_id, run_id y evidence_refs viajan como referencias opacas.
  - reason_code es un codigo compacto, no texto libre con datos reales.
  - summary es resumen breve de auditoria; no contiene secretos, prompts, completions ni transcripts.
  - No transporta proveedor, modelo, HOME, OAuth, PID, pane, URL ni ruta real.
Errores:
  - agent_stopper_payload_invalido
  - agent_stopper_target_invalido
  - agent_stopper_operacion_invalida
  - referencia_no_opaca
  - secreto_detectado
Pruebas de contrato:
  - DTOs y validador puro RUNTIME-006.
```

## AgentLauncherInboundV0Validator

```text
Nombre: AgentLauncherInboundV0Validator
Tipo: validador
Version: v0
Propietario: orquesta-runtime
Implementacion local:
  - agent_launcher_inbound_v0.go
  - agent_launcher_inbound_v0_test.go
Entrada:
  - AgentLauncherInboundV0 con target_port, message_type, correlation_id, idempotency_key y payload LaunchRuntimeAgentRequestV0.
  - LaunchRuntimeAgentRequestV0 compacto para validacion directa del payload.
Salida:
  - Lista de AgentLauncherInboundErrorV0; lista vacia significa orden compacta aceptable por contrato local v0.
Invariantes:
  - Es puro: no importa orquesta-core-workflow ni arranca runtime, procesos, CLI, MCP, browser, filesystem, red, proveedor, credenciales, HOME real ni quota real.
  - Solo acepta target_port `agent_launcher` y message_type `LaunchRuntimeAgent`.
  - agent_request_id, run_id, phase_id, task_ref, capacity_request_ref y evidence_refs viajan como referencias opacas.
  - task_ref es la ref requerida para resolver FunctionContractV0; si falta devuelve function_contract_ref_requerida.
  - capacity_request_ref es la ref requerida para resolver CapacityDecisionV0; si falta devuelve capacity_decision_ref_requerida.
  - RuntimeBindingV0 y evidencias mailbox/ACK/readiness no se inventan desde el payload compacto; si no existen resoluciones externas se devuelven errores publicos.
  - No construye RuntimeLaunchRequestV0 completo en RUNTIME-004.
Errores:
  - agent_launcher_payload_invalido
  - agent_launcher_target_invalido
  - agent_launcher_operacion_invalida
  - agent_launcher_enrichment_requerida
  - function_contract_ref_requerida
  - function_contract_no_resuelta
  - capacity_decision_ref_requerida
  - capacity_decision_no_resuelta
  - runtime_binding_refs_requeridas
  - evidence_refs_requeridas
  - referencia_no_opaca
  - secreto_detectado
  - runtime_launch_request_invalida
Pruebas de contrato:
  - go test -count=1 ./modulos/orquesta-runtime
```

## AgentStopperInboundV0Validator

```text
Nombre: AgentStopperInboundV0Validator
Tipo: validador
Version: v0
Propietario: orquesta-runtime
Implementacion local:
  - agent_stopper_inbound_v0.go
  - agent_stopper_inbound_v0_test.go
Entrada:
  - AgentStopperInboundV0 con target_port, message_type, correlation_id, idempotency_key y payload StopRuntimeAgentRequestV0.
  - StopRuntimeAgentRequestV0 compacto para validacion directa del payload.
Salida:
  - Lista de AgentStopperInboundErrorV0; lista vacia significa orden compacta aceptable por contrato local v0.
Invariantes:
  - Es puro: no detiene procesos ni toca runtime, CLI, MCP, browser, filesystem, red, proveedor, credenciales, HOME real ni quota real.
  - Solo acepta target_port `agent_launcher` y message_type `StopRuntimeAgent`.
  - agent_request_id, run_id, correlation_id, idempotency_key y evidence_refs son referencias opacas.
  - No decide lifecycle real ni interpreta estados de proceso.
Errores:
  - agent_stopper_payload_invalido
  - agent_stopper_target_invalido
  - agent_stopper_operacion_invalida
  - referencia_no_opaca
  - secreto_detectado
Pruebas de contrato:
  - go test -count=1 ./modulos/orquesta-runtime
```

## AgentProgressReport v0

```text
Nombre: AgentProgressReportV0
Tipo: dto_evidencia
Version: v0
Propietario: orquesta-runtime
Consumidores esperados: orquesta-director y orquesta-core como evidencia para decidir parada futura.
Implementacion local:
  - agent_progress_report_v0.go
  - agent_progress_report_v0_test.go
Campos:
  - report_id: referencia opaca del reporte.
  - run_id: referencia opaca del run durable.
  - agent_request_id: referencia opaca de la solicitud de agente.
  - status: progressing, stalled, loop_detected o stopped.
  - no_progress_ticks: contador local >= 0.
  - repeated_action_count: contador local >= 0.
  - summary: resumen compacto de auditoria.
  - evidence_refs: refs opacas de evidencias observadas.
Invariantes:
  - Es puro: no mata procesos, no consulta runtime real, no llama CLI/MCP/API, no usa filesystem, HOME, OAuth, proveedor ni modelo.
  - No decide parar un agente; solo reporta evidencia compacta para que director/core decidan.
  - Los status son cerrados en v0.
  - Si status es loop_detected, no_progress_ticks o repeated_action_count debe ser mayor que cero.
  - report_id, run_id, agent_request_id y evidence_refs son referencias opacas.
  - No transporta secretos, tokens, rutas HOME reales, URLs, proveedor/modelo concreto, prompts, completions ni transcripts.
Errores:
  - agent_progress_report_invalido
  - agent_progress_status_invalido
  - referencia_no_opaca
  - secreto_detectado
  - detalle_proveedor_detectado
  - loop_counter_requerido
Pruebas de contrato:
  - go test -count=1 ./modulos/orquesta-runtime
```

## AgentProgressHeartbeat v0

```text
Nombre: AgentProgressHeartbeatV0
Tipo: dto_observacion_compacta
Version: v0
Propietario: orquesta-runtime
Consumidores esperados: orquesta-director y scheduler via AgentProgressReportV0.
Implementacion local:
  - agent_progress_heartbeat_v0.go
  - agent_progress_heartbeat_v0_test.go
Entrada:
  - ProcessRuntimeSnapshotV0 publico.
  - AgentProgressHeartbeatV0 actual y heartbeat previo opcional.
  - AgentProgressHeartbeatPolicyV0 con umbrales numericos.
Salida:
  - AgentProgressReportV0 compacto.
Campos de heartbeat:
  - heartbeat_ref: referencia opaca del latido observado.
  - run_id: referencia opaca del run durable.
  - agent_request_id: referencia opaca de la solicitud de agente.
  - process_ref: referencia opaca del snapshot publico de proceso.
  - tick_counter: contador monotono del scheduler/adaptador.
  - progress_counter: contador monotono de avance observable.
  - repeated_action_count: contador de accion repetida observable.
  - evidence_refs: refs opacas adicionales.
Politica:
  - stalled_after_no_progress_ticks: ticks sin avance para reportar stalled.
  - loop_after_repeated_actions: acciones repetidas para reportar loop_detected.
Invariantes:
  - Es puro: no inspecciona memoria interna del agente, no lee transcripts, no consulta DB, no llama proveedor, no mata procesos ni cambia launch real.
  - Director/scheduler consumen status, counters y refs opacas; no necesitan PID, comando, HOME, proveedor, modelo, credencial ni ruta real.
  - Si process status es running y progress_counter aumenta, reporta progressing.
  - Si process status es running y progress_counter no aumenta durante la politica configurada, reporta stalled.
  - Si repeated_action_count alcanza la politica configurada, reporta loop_detected.
  - Si process status es stopped, reporta stopped.
  - La politica solo contiene umbrales numericos; no menciona proveedor, modelo, HOME, credencial ni DB.
Errores:
  - Reutiliza AgentProgressReportErrorV0.
Pruebas de contrato:
  - TestBuildAgentProgressReportFromHeartbeatV0DistingueProgresoEstancamientoYBucle.
```

## RuntimeFakeLifecycle v0

```text
Nombre: RuntimeFakeLifecycleV0
Tipo: adaptador_fake_memoria
Version: v0
Propietario: orquesta-runtime
Implementacion local:
  - runtime_fake_lifecycle_v0.go
  - runtime_fake_lifecycle_v0_test.go
Entrada:
  - LaunchAgentV0(AgentLauncherInboundV0)
  - ReportProgressV0(AgentProgressReportV0)
  - StopAgentV0(AgentStopperInboundV0)
Salida:
  - RuntimeFakeLifecycleSnapshotV0 compacto con agent_request_id, run_id, status, launch_ref, last_report_ref, stop_ref y progress_reports.
Invariantes:
  - Es puro y en memoria; no arranca ni detiene procesos reales.
  - No consulta proveedor, modelo, HOME, OAuth, filesystem, red, CLI, MCP ni APIs.
  - No importa orquesta-core-workflow.
  - Mantiene estado por agent_request_id y exige run_id consistente para reportes y parada.
  - LaunchAgentV0 valida ValidateAgentLauncherInboundV0 y registra status launched.
  - ReportProgressV0 valida ValidateAgentProgressReportV0, exige agente existente y actualiza status; loop_detected queda reflejado como estado del fake.
  - StopAgentV0 valida ValidateAgentStopperInboundV0, exige agente existente y marca stopped; parada repetida es idempotente.
  - LaunchAgentV0 repetido para el mismo agent_request_id/run_id es idempotente y no reabre un agente parado.
  - ReportProgressV0 tardio no reabre un agente parado; stopped es estado terminal del fake.
Errores:
  - inbound_invalido
  - agent_missing
  - run_mismatch
  - progress_invalid
Pruebas de contrato:
  - go test -count=1 ./modulos/orquesta-runtime
```

## ProcessRuntimeConnector v0

```text
Nombre: ProcessRuntimeConnectorV0
Tipo: adaptador_runtime_local
Version: v0
Propietario: orquesta-runtime
Implementacion local:
  - process_runtime_connector_types_v0.go
  - process_runtime_connector_v0.go
  - process_runtime_connector_v0_test.go
Entrada:
  - LaunchV0(context.Context, ProcessRuntimeLaunchRequestV0)
  - StopV0(context.Context, process_ref)
  - SnapshotV0(process_ref)
Salida:
  - ProcessRuntimeSnapshotV0 con schema_version, process_ref, session_ref, launch_ref, stop_ref y status.
Invariantes:
  - Es runtime real de proceso local solo para pruebas e2e controladas.
  - Arranca con os/exec usando command_path, args, env y working_dir explicitos en config/test.
  - command_path y working_dir pueden ser rutas absolutas reales del workspace ya resueltas por configuracion operacional; no pueden usar marcadores simbolicos de HOME ni credenciales.
  - No hay command por defecto, shell, sh -c, provider, OAuth, HOME real en salida publica, PATH heredado ni entorno heredado.
  - command_path, args, env y working_dir son configuracion operacional y no se serializan en el snapshot publico.
  - stdout y stderr se descartan; no se devuelve output completo, transcript, PID, cwd real, env, HOME, token ni ruta del binario.
  - Las refs publicas process_ref, session_ref, launch_ref y stop_ref son opacas y no codifican PID ni rutas.
  - session_ref identifica la sesion/proceso propiedad de Orquesta de forma opaca; cualquier limpieza o parada operativa debe resolverse desde el registry del run, nunca por barrido de procesos del sistema.
  - StopV0 usa os.Interrupt como senal portable y fallback a Kill bajo contexto de parada.
  - StopV0 repetido para un proceso ya parado es idempotente y conserva stop_ref.
  - Un proceso que sale solo queda observable como status stopped sin inventar stop_ref.
Errores:
  - process_runtime_config_invalida
  - process_runtime_shell_prohibida
  - process_runtime_env_prohibido
  - process_runtime_ref_invalida
  - process_runtime_no_encontrado
  - process_runtime_launch_fallido
  - process_runtime_stop_fallido
  - process_runtime_context_done
Pruebas de contrato:
  - go test -count=1 ./modulos/orquesta-runtime
```

## RuntimeLaunch v0

```text
Nombre: RuntimeLaunch
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-runtime
Consumidores: orquesta-core como consumidor autorizado; orquesta-capacity solo aporta referencias opacas ya decididas por CapacityDecisionV0.
Campos:
  - request: RuntimeLaunchRequestV0
  - salida_ok: RuntimeLaunchAcceptedV0
  - salida_error: RuntimeLaunchErrorV0
Invariantes:
  - El puerto acepta una orden de lanzamiento; no decide negocio, fases, asignacion ni politica de capacidad.
  - V0 no implementa runtime real, proceso, CLI, MCP, API ni filesystem; solo fija el contrato local validable.
  - El core debe entregar un FunctionContractV0 activo y con write_set cerrado.
  - CapacityDecisionV0 llega como evidencia previa; runtime no recalcula nivel, modelo, pool, HOME ni cuota.
  - Runtime transport no es fuente de verdad: mailbox, ACK, readiness y checkpoints son referencias de evidencia persistible.
  - Multi-HOME separa identidad logica, proveedor, modelo, HOME, credencial, pool y cuota mediante referencias opacas.
  - RuntimeBindingV0 consume el binding compatible con AgentHomeV0 solo por refs opacas; account/email, HOME real, token, proveedor y modelo concreto no viajan en runtime.
  - No se aceptan secretos, tokens OAuth, rutas HOME reales, rutas absolutas ni transcripts completos.
Errores:
  - runtime_launch_request_invalida
  - function_contract_requerido
  - function_contract_no_activa
  - write_set_requerido
  - write_set_invalido
  - capacity_decision_requerida
  - capacity_decision_no_soportada
  - pool_ref_requerido
  - model_ref_requerido
  - home_ref_requerido
  - credential_ref_requerido
  - referencia_no_opaca
  - secreto_detectado
  - ruta_home_real_detectada
  - mailbox_ref_requerido
  - ack_ref_requerido
  - readiness_ref_requerido
  - idempotency_key_requerida
  - runtime_no_disponible
  - context_bundle_requerido
  - context_bundle_invalido
Pruebas de contrato:
  - RUNTIME-CT-001 valida fixture minimo positivo contra JSON Schema draft-07.
  - RUNTIME-CT-002 rechaza request sin capacity_decision.
  - RUNTIME-CT-003 rechaza HOME no opaco con forma de ruta.
```

## RuntimeLaunchRequestV0

```text
Nombre: RuntimeLaunchRequestV0
Tipo: dto
Version: v0
Propietario: orquesta-runtime
Consumidores: orquesta-core como consumidor autorizado; orquesta-capacity no invoca este contrato y solo aporta referencias opacas via CapacityDecisionV0.
Contrato compartido:
  - Promovido a `../../CONTRATOS.md` por decision del director el 2026-05-04.
Schemas canonicos:
  - docs/schemas/runtime_launch_request_v0.schema.json
Fixtures canonicos:
  - docs/fixtures/runtime_launch_request_v0/request_minima_valida.json
  - docs/fixtures/runtime_launch_request_v0/request_sin_capacity_invalida.json
  - docs/fixtures/runtime_launch_request_v0/request_home_ref_no_opaca_invalida.json
Campos:
  - schema_version: "runtime_launch_request.v0".
  - request_id: identificador opaco de la peticion.
  - correlation_id: identificador opaco para trazabilidad entre adaptadores.
  - idempotency_key: clave opaca obligatoria.
  - requested_at: instant UTC serializado.
  - source: modulo llamante y adaptador/caso de uso como referencias opacas.
  - locale: locale para diagnosticos publicos; no es texto UI final.
  - launch_mode: new_session en v0.
  - task: referencias opacas a tarea, proyecto y fase.
  - function_contract: subset publico de FunctionContractV0 activo, con write_set, tests y criterio de cierre.
  - capacity_decision: referencias opacas a decision, pool, modelo y cuota calculadas fuera de runtime.
  - runtime_binding: identidad operativa compatible con AgentHomeV0 y separada en logical_agent_ref, runtime_kind, connector_ref, provider_ref, model_ref, home_ref, credential_kind y credential_ref.
  - evidence_refs: mailbox_ref, ack_ref, readiness_ref y checkpoint_ref opcional.
  - context_bundle: ContextBundleV0 preparado por orquesta-context y obligatorio para arrancar sesion.
  - delivery: protocolo de mailbox por referencia, ACK obligatorio y ventanas maximas de readiness/startup.
  - safety: politicas declarativas references_only, opaque_refs_only y write_set cerrado.
Invariantes:
  - request_id, correlation_id e idempotency_key son opacos y no codifican proveedor, HOME ni cuenta real.
  - provider_ref, model_ref, home_ref y credential_ref son referencias; no son nombres canonicos de proveedor, modelos concretos, rutas ni tokens.
  - account_ref, si procede del propietario de AgentHomeV0, no se materializa como email ni cuenta real en RuntimeLaunchRequestV0.
  - credential_kind puede declarar oauth_ref, pero el token OAuth nunca viaja en el contrato.
  - Capacity sigue siendo propietario del contrato de politica/capacidad; runtime no importa capacity ni duplica su nucleo.
  - runtime_binding.model_ref debe coincidir con capacity_decision.model_ref o quedar rechazado por el validador de aplicacion futuro.
  - archivo_objetivo debe pertenecer a write_set; JSON Schema valida forma relativa, la pertenencia exacta requiere harness posterior.
  - write_set contiene rutas relativas al repo o modulo; no acepta rutas absolutas, HOME, URL ni parent traversal.
  - launch_mode no cubre resume; resume tendra contrato propio o version nueva.
  - Los mensajes de error publicos deben mapearse a claves i18n en adaptadores de UI/CLI/MCP.
Errores:
  - runtime_launch_request_invalida
  - idempotency_key_requerida
  - function_contract_requerido
  - function_contract_no_activa
  - write_set_requerido
  - write_set_invalido
  - capacity_decision_requerida
  - capacity_decision_no_soportada
  - referencia_no_opaca
  - secreto_detectado
  - ruta_home_real_detectada
Pruebas de contrato:
  - `jq empty` sobre schema y fixtures.
  - `npx --yes ajv-cli validate --spec=draft7` para fixture positivo.
  - `npx --yes ajv-cli validate --spec=draft7` debe fallar para fixtures negativos.
```

## RuntimeLaunchRequestV0Validator

```text
Nombre: RuntimeLaunchRequestV0Validator
Tipo: validador
Version: v0
Propietario: orquesta-runtime
Consumidores: orquesta-runtime; orquesta-core podra consumirlo por puerto local futuro si se decide.
Implementacion local:
  - runtime_launch_request_v0.go
  - runtime_launch_request_v0_test.go
Campos:
  - entrada: RuntimeLaunchRequestV0.
  - salida: lista de RuntimeLaunchErrorV0; lista vacia significa request aceptable por contrato local v0.
Invariantes:
  - Es puro: no arranca procesos, CLI, MCP, browser, filesystem, red ni proveedor.
  - Runtime ejecuta ordenes ya decididas; no calcula fase, asignacion, capacidad, modelo, pool, HOME ni cuota.
  - FunctionContractV0 debe estar presente, activo y con write_set no vacio.
  - write_set solo contiene rutas contractuales relativas; no acepta ruta absoluta, parent traversal, URL, HOME ni tilde.
  - CapacityDecisionV0 debe estar presente y soportada en version v0.
  - runtime_binding.model_ref debe coincidir con capacity_decision.model_ref.
  - context_bundle debe estar presente, ser `ContextBundleV0` valido y coincidir con el modulo objetivo del FunctionContract cuando `archivo_objetivo` lo declara.
  - context_bundle.capacity_level debe coincidir con capacity_decision.nivel_capacidad.
  - Todas las referencias publicas viajan como refs opacas.
  - home_ref se valida como referencia opaca y se rechaza si tiene forma de ruta.
  - provider_ref y model_ref se rechazan si contienen proveedores o modelos hardcodeados aunque cumplan la forma sintactica de ref.
  - safety debe declarar secrets_policy references_only, home_paths_policy/provider_policy opaque_refs_only y write_set_policy closed.
  - No se aceptan patrones obvios de secreto/token en referencias.
Errores:
  - runtime_launch_request_invalida
  - idempotency_key_requerida
  - function_contract_requerido
  - function_contract_no_activa
  - write_set_requerido
  - write_set_invalido
  - capacity_decision_requerida
  - capacity_decision_no_soportada
  - pool_ref_requerido
  - model_ref_requerido
  - home_ref_requerido
  - credential_ref_requerido
  - referencia_no_opaca
  - secreto_detectado
  - ruta_home_real_detectada
  - mailbox_ref_requerido
  - ack_ref_requerido
  - readiness_ref_requerido
  - context_bundle_requerido
  - context_bundle_invalido
Pruebas de contrato:
  - go test -count=1 ./modulos/orquesta-runtime
```

## AgentStartPacketV0

```text
Nombre: AgentStartPacketV0
Tipo: dto_arranque_agente
Version: v0
Propietario: orquesta-runtime
Consumidores: conectores de agente CLI/MCP/API/local/remoto futuros.
Entrada fuente:
  - RuntimeLaunchRequestV0 validado.
  - ContextMaterializedBundleV0 validado y coincidente con context_bundle.
Constructor local:
  - BuildAgentStartPacketV0(request, materialized).
Salida:
  - Paquete neutral con request_id, correlation_id, work_order_ref, target_module, phase, capacity_level, locale, task, context materializado, delivery_refs y policies.
Invariantes:
  - Es un paquete de arranque para agentes; no es selector de proveedor, modelo, HOME ni credencial.
  - Solo se construye si RuntimeLaunchRequestV0 es valido y el ContextMaterializedBundleV0 coincide en bundle_ref, work_order_ref y target_module.
  - Contexto materializado viaja acotado y preparado por orquesta-context; runtime no lee filesystem ni expande refs por su cuenta.
  - No expone provider_ref, model_ref, home_ref, credential_ref, OAuth, tokens, HOME real, rutas operativas, PID, command_path, entorno, transcripts, prompts completos ni completions.
  - task deriva del FunctionContractV0 publico: task_ref, prioridad, titulo, objetivo, simbolo objetivo, write_set cerrado, tests obligatorios y criterio de cierre.
  - delivery_refs solo contiene mailbox_ref, ack_ref, readiness_ref y checkpoint_ref opacos.
  - policies son codigos compactos para el conector; no contienen texto UI final ni secretos.
  - Los conectores concretos traducen este paquete a su transporte sin ampliar el contrato con datos operativos privados.
Errores:
  - agent_start_packet_invalido
  - context_materialized_bundle_requerido
  - context_materialized_bundle_invalido
Pruebas de contrato:
  - go test -count=1 ./modulos/orquesta-runtime
```

## RuntimeLaunchAcceptedV0

```text
Nombre: RuntimeLaunchAcceptedV0
Tipo: dto
Version: v0
Propietario: orquesta-runtime
Consumidores: orquesta-core
Campos:
  - schema_version: "runtime_launch_accepted.v0".
  - request_id: copia del request.
  - correlation_id: copia del request.
  - launch_id: identificador opaco e idempotente para la aceptacion.
  - status: accepted.
  - runtime_handle_ref: referencia opaca al handle futuro; no es PID, tmux pane, URL ni ruta.
  - mailbox_ref: referencia de mailbox persistible.
  - ack_ref: referencia de ACK persistible.
  - readiness_ref: referencia de readiness persistible.
  - checkpoint_ref: referencia opcional de checkpoint.
  - accepted_at: instant UTC serializado.
  - warnings: lista de codigos publicos; no texto UI final.
Invariantes:
  - Accepted confirma aceptacion contractual, no que un proceso real haya arrancado en v0.
  - runtime_handle_ref no concede acceso directo al transporte.
  - mailbox_ref, ack_ref y readiness_ref deben coincidir con las referencias aceptadas del request o equivalentes persistibles.
  - warnings no contienen secretos, rutas HOME ni proveedor concreto.
Errores:
  - runtime_no_disponible
  - evidencia_runtime_incompleta
Pruebas de contrato:
  - Fixture futuro de response accepted antes de implementar adaptador real.
```

## RuntimeLaunchErrorV0

```text
Nombre: RuntimeLaunchErrorV0
Tipo: error
Version: v0
Propietario: orquesta-runtime
Consumidores: orquesta-core
Campos:
  - code: codigo publico de RuntimeLaunch v0.
  - message_key: clave i18n de diagnostico publico.
  - field: campo contractual cuando aplique.
  - retryable: boolean.
  - correlation_id: referencia opaca.
  - evidence: lista corta de referencias o checks, nunca secretos ni transcripts completos.
Invariantes:
  - code debe pertenecer a la lista de errores publicos del puerto.
  - message_key evita fijar texto UI en el dominio runtime.
  - evidence no contiene stack traces, rutas HOME reales, tokens OAuth ni datos privados del proveedor.
Errores:
  - error_code_invalido
  - evidence_insegura
Pruebas de contrato:
  - Cada error publico tendra fixture minimo cuando exista validador de aplicacion.
```

## CONSULTA AL DIRECTOR

```text
Modulo origen: orquesta-runtime
Modulos afectados: orquesta-core, orquesta-capacity, contratos globales.
Bloqueo: Resuelto; `../../CONTRATOS.md` ya incluye RuntimeLaunchRequest v0 como contrato compartido.
Pregunta concreta: Promovemos el resumen minimo de RuntimeLaunchRequest v0 a `modulos/CONTRATOS.md` para que orquesta-core pueda consumirlo y orquesta-capacity cierre AgentHomeV0, o permanece local hasta que exista un adaptador runtime real?
Opcion recomendada: Promover solo el resumen global minimo despues de validar el consumo desde core/capacity; mantener schemas, fixtures e invariantes extensas como canon local de orquesta-runtime.
Impacto: core podra pedir lanzamiento sin conocer proveedor, HOME, OAuth ni runtime transport; capacity podra entregar pool/model/home como referencias opacas; runtime seguira sin implementar procesos reales en v0.
Decision del director: RuntimeLaunchRequest v0 queda promovido a contrato compartido en `modulos/CONTRATOS.md`.
Alcance confirmado: orquesta-core es consumidor autorizado; orquesta-capacity solo aporta referencias opacas via CapacityDecision v0; resume y runtime real quedan fuera de este corte.
Estado: cerrada.
```

## ExternalAgentConnector v0

```text
Nombre: ExternalAgentConnectorV0
Tipo: contrato_preparacion_runtime
Version: v0
Propietario: orquesta-runtime
Implementacion local:
  - external_agent_connector_types_v0.go
  - external_agent_connector_v0.go
  - external_agent_profile_builder_v0.go
  - external_agent_connector_v0_test.go
Entrada:
  - ExternalAgentConnectorProfileV0 opt-in con refs opacas y comando ya resuelto por adaptador superior como refs.
  - RuntimeLaunchRequestV0 validado.
  - AgentStartPacketV0 neutral ya construido.
Salida:
  - ExternalAgentLaunchSpecV0 validable, sin ejecutar proceso ni resolver proveedor.
Invariantes:
  - Es puro: no ejecuta comandos, shell, PATH heredado, HOME real, red externa, proveedor, modelo, OAuth ni secretos.
  - BuildClosedExternalAgentConnectorProfileV0 construye perfiles opt-in con politica cerrada comun desde refs opacas.
  - El perfil exige profile_ref, connector_ref, runtime_kind, launch_mode, comando por refs y politica de seguridad cerrada.
  - Solo copia connector_ref, runtime_kind y launch_mode desde el binding runtime; no copia provider_ref, model_ref, home_ref ni credential_ref.
  - El paquete para el agente sigue siendo AgentStartPacketV0 y no contiene provider/model/HOME/OAuth/credential refs.
  - Rechaza provider/model concretos, HOME real, OAuth, token, secret, rutas absolutas peligrosas y transcripts.
  - El comando queda como command_ref, executable_ref, arg_refs, env_refs y working_dir_ref opacos; no hay sh -c ni comando por defecto.
Errores:
  - external_agent_connector_profile_invalido
  - external_agent_launch_spec_invalido
  - external_agent_command_invalido
  - security_policy_closed_required
  - referencia_no_opaca
  - detalle_operacional_prohibido
  - agent_start_packet_invalido
  - runtime_launch_request_invalida
Pruebas de contrato:
  - go test -count=1 ./modulos/orquesta-runtime
```

## ExternalAgentProcessAdapter v0

```text
Nombre: ExternalAgentProcessAdapterV0
Tipo: adaptador_runtime_opt_in
Version: v0
Propietario: orquesta-runtime
Implementacion local:
  - external_agent_process_adapter_types_v0.go
  - external_agent_process_adapter_v0.go
  - external_agent_process_adapter_v0_test.go
Entrada:
  - ExternalAgentLaunchSpecV0 validable.
  - ExternalAgentProcessCommandResolverV0 inyectado.
  - ExternalAgentProcessRuntimePortV0 inyectado.
Salida:
  - ExternalAgentProcessLaunchResultV0 con status started o blocked.
Invariantes:
  - No resuelve proveedor, modelo, HOME, OAuth, PATH ni credenciales dentro del nucleo runtime.
  - El resolver traduce refs opacas del spec a ProcessRuntimeLaunchRequestV0 fuera del contrato publico.
  - El adaptador valida el spec y la request operacional antes de llamar al runtime real.
  - El resultado publico no serializa command_path, args, env, working_dir, PID, HOME, token, provider, modelo ni transcript.
  - Si falta resolver/runtime o la configuracion operacional es invalida, devuelve blocked con errores i18n versionados.
Errores:
  - external_agent_command_resolution_invalida
  - external_agent_command_resolver_unavailable
  - external_agent_runtime_unavailable
  - external_agent_runtime_launch_failed
Pruebas de contrato:
  - go test -count=1 ./modulos/orquesta-runtime
```

## Plantilla

```text
Nombre:
Tipo: puerto_entrada | puerto_salida | dto | evento | error
Version:
Propietario:
Consumidores:
Campos:
Invariantes:
Errores:
Pruebas de contrato:
```
Invariante de entorno:

- `Env` no puede ser `nil`; `nil` se rechaza para impedir herencia implicita.
- `Env: []string{}` significa entorno vacio real, no herencia del proceso padre.
- `PATH=` exacto puede venir solo como variable privada explicita de un conector
  opt-in superior. El valor debe ser una lista de rutas absolutas sin marcas de
  credencial o secreto.
- HOME, token, OAuth, credencial, provider, modelo y cualquier otro entorno
  operacional siguen prohibidos; el puerto de proceso no hereda nada por defecto.
