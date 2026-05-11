# Contrato local: AgentLauncherInbound v0

Define el puerto/adaptador logico que recibe la orden compacta emitida por `orquesta-core-workflow` y la convierte en `RuntimeLaunchRequestV0` solo despues de enriquecerla por puertos de resolucion versionados.

## AgentLauncherInboundV0

```text
Nombre: AgentLauncherInboundV0
Tipo: puerto_entrada_logico
Version: v0
Propietario: orquesta-runtime
Productor autorizado: orquesta-core-workflow, via outbox logico `LaunchRuntimeAgent` con target `agent_launcher`.
Consumidor local: adaptador futuro de orquesta-runtime que validara el mensaje, resolvera referencias autorizadas y construira `RuntimeLaunchRequestV0`.
Entrada compacta: LaunchRuntimeAgentRequestV0.
Salida contractual:
  - ok: RuntimeLaunchRequestV0 completo y validable localmente, solo tras enriquecimiento.
  - error: AgentLauncherInboundErrorV0.
Invariantes:
  - Es un puerto logico documental; no implementa runtime real, DB, filesystem productivo, CLI, MCP, HTTP, Docker, procesos ni conectores.
  - No importa tipos ni paquetes de orquesta-core-workflow; la compatibilidad se mantiene por nombre de evento, target y payload documentado.
  - Solo acepta mensajes con target `agent_launcher` y message_type/operacion logica `LaunchRuntimeAgent`.
  - No decide negocio, fases, asignacion, capacidad, modelo, HOME, proveedor, cuota ni credencial.
  - El payload emitido por core-workflow no contiene `FunctionContractV0`, `CapacityDecisionV0`, runtime binding ni evidencias mailbox/ACK/readiness.
  - Transforma a `RuntimeLaunchRequestV0` completo solo si los puertos autorizados resuelven `FunctionContractV0`, `CapacityDecisionV0`, runtime binding y evidencias de lanzamiento.
  - Si falta una ref externa necesaria o un puerto de resolucion, la transformacion queda bloqueada y debe devolver error contractual; no se inventan defaults.
  - No hardcodea proveedor, modelo, HOME, OAuth, runtime concreto ni herramienta concreta; solo transporta refs opacas.
  - El payload compacto no contiene secretos, rutas HOME reales, prompts, completions ni transcripts completos.
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
  - Revision documental RUNTIME-003: este contrato no introduce runtime real ni import de core-workflow.
  - DTOs y validador puro RUNTIME-004: `agent_launcher_inbound_v0.go` y `agent_launcher_inbound_v0_test.go`.
  - Revision de compatibilidad: los campos de `LaunchRuntimeAgentRequestV0` coinciden con el contrato local de `orquesta-core-workflow`.
  - Validacion heredada: el resultado de la transformacion debe cumplir `ValidateRuntimeLaunchRequestV0`.
```

## LaunchRuntimeAgentRequestV0

```text
Nombre: LaunchRuntimeAgentRequestV0
Tipo: dto_compacto_de_entrada
Version: v0
Propietario del payload: orquesta-core-workflow.
Propietario del adaptador receptor: orquesta-runtime.
Destino logico requerido: agent_launcher.
Operacion requerida: LaunchRuntimeAgent.
Campos minimos esperados:
  - agent_request_id: referencia opaca de la solicitud de agente.
  - run_id: referencia opaca del run durable.
  - phase_id: fase v0 del workflow.
  - task_ref: referencia opaca de la tarea; requerida para resolver FunctionContractV0.
  - capacity_request_ref: referencia opaca a la solicitud/decision de capacidad; requerida para resolver CapacityDecisionV0.
  - role: rol logico solicitado.
  - summary: resumen compacto de la solicitud.
  - evidence_refs: referencias opacas opcionales de evidencia de negocio.
Campos prohibidos:
  - nombres concretos de proveedor/modelo, rutas HOME reales, secretos, credenciales, prompts, completions y transcripts.
Invariantes:
  - Es deliberadamente compacto: no replica `FunctionContractV0`, `CapacityDecisionV0`, runtime binding ni evidencias de transporte.
  - `task_ref` debe resolver a un `FunctionContractV0` activo antes de lanzar.
  - `capacity_request_ref` debe resolver a una `CapacityDecisionV0` soportada antes de lanzar; si falta, runtime no decide capacidad.
  - El runtime binding se resuelve por puerto local de inventario runtime usando refs opacas ya decididas.
  - Mailbox, ACK y readiness se crean o resuelven por puerto local de evidencias de lanzamiento; no proceden del payload compacto.
  - La coincidencia entre `CapacityDecisionV0.model_ref` y `runtime_binding.model_ref` se comprueba al construir/validar `RuntimeLaunchRequestV0`.
```

## Implementacion pura RUNTIME-004

```text
Archivos:
  - agent_launcher_inbound_v0.go
  - agent_launcher_inbound_v0_test.go
  - agent_launcher_transform_v0.go
  - agent_launcher_transform_helpers_v0.go
  - agent_launcher_transform_v0_test.go
DTOs:
  - AgentLauncherInboundV0: envelope local minimo del outbox con target_port, message_type, correlation_id, idempotency_key y payload.
  - LaunchRuntimeAgentRequestV0: payload compacto documentado por el contrato global.
  - AgentLauncherResolvedDependenciesV0: contenedor contractual para comprobar si ya existen FunctionContractV0, CapacityDecisionV0, RuntimeBindingV0 y RuntimeEvidenceRefsV0 resueltos.
Validadores:
  - ValidateAgentLauncherInboundV0.
  - ValidateLaunchRuntimeAgentRequestV0.
  - ValidateAgentLauncherResolvedDependenciesV0.
Puertos dejados como interfaces, sin implementacion:
  - FunctionContractResolverV0.
  - CapacityDecisionResolverV0.
  - RuntimeBindingResolverV0.
  - LaunchEvidenceResolverV0.
Bloqueo:
  - RUNTIME-004 no construia RuntimeLaunchRequestV0 completo porque faltaban resoluciones autorizadas.
  - RUNTIME-017 cierra la transformacion pura cuando esas dependencias ya vienen resueltas por puertos. Resolverlas y persistirlas sigue fuera de este helper.
```

## Transformacion compacta a RuntimeLaunchRequestV0

```text
Nombre: LaunchRuntimeAgentToRuntimeLaunchRequestV0
Tipo: adaptador_logico
Version: v0
Entrada: LaunchRuntimeAgentRequestV0 + resoluciones externas autorizadas.
Salida: RuntimeLaunchRequestV0.
Implementacion local: `LaunchRuntimeAgentToRuntimeLaunchRequestV0`.
Reglas:
  - schema_version se fija a `runtime_launch_request.v0`.
  - request_id deriva de `agent_request_id` o de una ref opaca equivalente del outbox; no se usa un identificador con datos reales.
  - correlation_id e idempotency_key proceden del envelope `OutboxMessageV0`.
  - requested_at lo fija el adaptador runtime en formato UTC al aceptar la transformacion.
  - locale usa la configuracion publica del adaptador o el locale por defecto del modulo; no se infiere del prompt.
  - task.task_ref se copia desde `task_ref`; si falta, no se puede construir `RuntimeLaunchRequestV0`.
  - task.phase_ref queda en `phase_id` si cumple formato opaco; si no, se transforma a una ref opaca equivalente.
  - task.priority se fija a `normal` en v0 porque el payload compacto no transporta prioridad.
  - `summary` queda como evidencia/auditoria compacta del inbound; no se inyecta en RuntimeLaunchTaskV0.
  - source.module se fija a `orquesta-core` para cumplir `RuntimeLaunchRequestV0`; source.adapter_ref referencia el outbox de core-workflow.
  - launch_mode se fija a `new_session` en v0.
  - function_contract se materializa desde `task_ref` solo si `FunctionContractResolverV0` devuelve un FunctionContractV0 activo con write_set cerrado.
  - capacity_decision se materializa desde `capacity_request_ref` solo si `CapacityDecisionResolverV0` devuelve CapacityDecisionV0 soportada.
  - runtime_binding se materializa desde `RuntimeBindingResolverV0` solo si existen logical_agent_ref, runtime_kind, connector_ref, provider_ref, model_ref, home_ref, credential_kind y credential_ref.
  - evidence_refs se materializa desde `LaunchEvidenceResolverV0` solo si existen mailbox_ref, ack_ref y readiness_ref.
  - delivery y safety se fijan a politicas contractuales de runtime v0: mailbox por ref, ACK obligatorio, refs opacas, secrets references_only y write_set closed.
  - La salida se acepta solo si `ValidateRuntimeLaunchRequestV0` no devuelve errores.
Bloqueos:
  - Si no existe contrato local para resolver una ref externa, devolver `agent_launcher_enrichment_requerida`; no ampliar campos sin decision versionada.
  - Si core-workflow cambia target, operacion o nombre del payload, versionar este puerto o elevar decision.
```

## Puertos de resolucion autorizados

```text
FunctionContractResolverV0:
  - Propietario: orquesta-core.
  - Entrada: task_ref opaco.
  - Salida: RuntimeFunctionContractV0 derivado de FunctionContractV0 activo.
CapacityDecisionResolverV0:
  - Propietario: orquesta-capacity.
  - Entrada: capacity_request_ref opaco.
  - Salida: RuntimeCapacityDecisionV0 derivado de CapacityDecisionV0 soportada.
RuntimeBindingResolverV0:
  - Propietario: orquesta-runtime.
  - Entrada: refs opacas de capacidad y rol logico.
  - Salida: RuntimeBindingV0.
LaunchEvidenceResolverV0:
  - Propietario: orquesta-runtime.
  - Entrada: agent_request_id, run_id, correlation_id e idempotency_key opacos.
  - Salida: RuntimeEvidenceRefsV0 con mailbox_ref, ack_ref y readiness_ref.
Regla comun: cada resolver es un puerto; su almacenamiento o proveedor queda detras de conectores hexagonales.
```

## AgentLauncherInboundErrorV0

```text
Nombre: AgentLauncherInboundErrorV0
Tipo: error
Version: v0
Campos:
  - code: codigo publico del puerto.
  - message_key: clave i18n.
  - field: campo contractual cuando aplique.
  - retryable: boolean.
  - correlation_id: referencia opaca.
  - evidence: lista corta de refs/checks, nunca secretos ni rutas reales.
Invariantes:
  - Los errores de refs no resueltas son retryable solo si la resolucion externa puede llegar despues.
  - `runtime_launch_request_invalida` envuelve errores del validador local sin exponer datos sensibles.
```

## CONSULTA AL DIRECTOR

```text
Modulo origen: orquesta-runtime
Modulos afectados: orquesta-core-workflow, contratos globales.
Bloqueo: No bloquea RUNTIME-003 porque el contrato local queda definido como puerto logico; la implementacion real se divide en resolvers versionados.
Pregunta concreta: Que modulo o contrato global sera fuente autorizada para resolver cada dato que falta en el payload compacto `LaunchRuntimeAgentRequestV0`?
Opcion recomendada: Mantener payload compacto en core-workflow y definir `FunctionContractResolverV0`, `CapacityDecisionResolverV0`, `RuntimeBindingResolverV0` y `LaunchEvidenceResolverV0`.
Impacto: Evita que runtime invente FunctionContractV0, CapacityDecisionV0, HOME, proveedor, credenciales o evidencias.
Decision del director: Aceptada la opcion recomendada. Los resolvers quedan como puertos versionados; cualquier implementacion debe pasar por conectores hexagonales y no por imports internos entre modulos.
Estado: cerrada para RUNTIME-003; pendiente implementar resolvers en microtareas separadas.
```

## AgentStopperInboundV0

```text
Nombre: AgentStopperInboundV0
Tipo: puerto_entrada_logico
Version: v0
Propietario: orquesta-runtime
Productor autorizado: orquesta-core-workflow, via outbox logico `StopRuntimeAgent` con target `agent_launcher`.
Consumidor local: adaptador futuro de orquesta-runtime que validara la intencion de parada logica antes de cualquier lifecycle real.
Entrada compacta: StopRuntimeAgentRequestV0.
Salida contractual:
  - ok: mensaje aceptable por contrato local para una parada logica futura.
  - error: AgentStopperInboundErrorV0.
Invariantes:
  - Es un puerto logico documental con DTOs y validador puro; no implementa parada real, procesos, CLI, MCP, HTTP, Docker, filesystem, red, runtime concreto ni conectores.
  - No importa tipos ni paquetes de orquesta-core-workflow; la compatibilidad se mantiene por target, message_type y payload documentado.
  - Solo acepta mensajes con target `agent_launcher` y message_type `StopRuntimeAgent`.
  - No decide negocio, fase, asignacion, capacidad, modelo, proveedor, HOME, cuota ni credencial.
  - No hardcodea proveedor, modelo, HOME, OAuth, PID, pane, URL, runtime concreto ni herramienta concreta.
  - El payload compacto no contiene secretos, rutas HOME reales, prompts, completions ni transcripts completos.
Errores:
  - agent_stopper_payload_invalido
  - agent_stopper_target_invalido
  - agent_stopper_operacion_invalida
  - referencia_no_opaca
  - secreto_detectado
Pruebas de contrato:
  - DTOs y validador puro RUNTIME-006: `agent_stopper_inbound_v0.go` y `agent_stopper_inbound_v0_test.go`.
```

## StopRuntimeAgentRequestV0

```text
Nombre: StopRuntimeAgentRequestV0
Tipo: dto_compacto_de_entrada
Version: v0
Propietario del payload: orquesta-core-workflow.
Propietario del adaptador receptor: orquesta-runtime.
Destino logico requerido: agent_launcher.
Operacion requerida: StopRuntimeAgent.
Campos minimos esperados:
  - agent_request_id: referencia opaca de la solicitud de agente.
  - run_id: referencia opaca del run durable.
  - reason_code: codigo compacto de motivo logico.
  - summary: resumen breve de auditoria.
  - evidence_refs: referencias opacas opcionales de evidencia de negocio.
Campos prohibidos:
  - nombres concretos de proveedor/modelo, rutas HOME reales, secretos, credenciales, OAuth, PID, panes, URLs, prompts, completions y transcripts.
Invariantes:
  - Es deliberadamente compacto: declara intencion de parada logica, no instruccion de sistema operativo ni transporte.
  - `agent_request_id` y `run_id` deben poder relacionarse con estado persistible por adaptadores futuros; este contrato no consulta almacenamiento.
  - `reason_code` no contiene datos sensibles ni proveedor real.
```

## Implementacion pura RUNTIME-006

```text
Archivos:
  - agent_stopper_inbound_v0.go
  - agent_stopper_inbound_v0_test.go
DTOs:
  - AgentStopperInboundV0: envelope local minimo del outbox con target_port, message_type, correlation_id, idempotency_key y payload.
  - StopRuntimeAgentRequestV0: payload compacto documentado por el contrato local.
Validadores:
  - ValidateAgentStopperInboundV0.
  - ValidateStopRuntimeAgentRequestV0.
Bloqueo:
  - RUNTIME-006 no detiene agentes, no mata procesos, no escribe mailbox y no resuelve handles. Cualquier ejecucion real queda para puertos/adaptadores posteriores.
```

## AgentStopperInboundErrorV0

```text
Nombre: AgentStopperInboundErrorV0
Tipo: error
Version: v0
Campos:
  - code: codigo publico del puerto.
  - message_key: clave i18n.
  - field: campo contractual cuando aplique.
  - retryable: boolean.
  - correlation_id: referencia opaca.
  - evidence: lista corta de refs/checks, nunca secretos ni rutas reales.
Invariantes:
  - Los errores son publicos y versionados por el puerto.
  - El validador no expone datos sensibles en message_key ni code.
```

## ExternalAgentConnectorProfileV0

```text
Nombre: ExternalAgentConnectorProfileV0
Tipo: perfil_conector_agente_real
Version: v0
Propietario: orquesta-runtime
Uso: habilitar explicitamente un conector de agente real sin acoplar runtime a Codex/Ollama/vLLM/proveedor/modelo/HOME/OAuth.
Campos:
  - schema_version: external_agent_connector_profile.v0.
  - profile_ref: ref opaca del perfil opt-in.
  - connector_ref: ref opaca del conector permitido.
  - runtime_kind: tipo runtime cerrado en v0.
  - launch_mode: modo de lanzamiento, new_session en v0.
  - command: command_ref, executable_ref, arg_refs, env_refs y working_dir_ref opacos ya resueltos por adaptador superior.
  - security: politica cerrada con opt_in=true, shell/path inheritance prohibidos, env por refs, secretos references_only, HOME opaco, red cerrada por defecto y transcripts prohibidos.
Invariantes:
  - No contiene proveedor, modelo, HOME real, OAuth, token, secreto, transcript ni ruta absoluta.
  - No define comando por defecto ni usa shell, PATH heredado o entorno heredado.
  - El comando no se ejecuta en este contrato; solo queda como especificacion verificable para un adaptador posterior.
```

## ExternalAgentLaunchSpecV0

```text
Nombre: ExternalAgentLaunchSpecV0
Tipo: especificacion_ejecucion_opt_in
Version: v0
Propietario: orquesta-runtime
Entrada: RuntimeLaunchRequestV0 + AgentStartPacketV0 + ExternalAgentConnectorProfileV0.
Salida: spec validable para conector real futuro.
Invariantes:
  - BuildExternalAgentLaunchSpecV0 es helper puro y no ejecuta procesos ni resuelve refs.
  - Copia solo request_id, correlation_id, profile_ref, connector_ref, runtime_kind, launch_mode, command, security y AgentStartPacketV0.
  - No copia provider_ref, model_ref, home_ref ni credential_ref del RuntimeBindingV0.
  - Rechaza paquetes de agente invalidos, no coincidentes con el request o con detalle operacional prohibido.
  - Rechaza perfiles sin opt-in o con politica abierta.
Pruebas:
  - external_agent_connector_v0_test.go.
```

## ExternalAgentProcessAdapterV0

```text
Nombre: ExternalAgentProcessAdapterV0
Tipo: puente_runtime_opt_in
Version: v0
Propietario: orquesta-runtime
Entrada:
  - ExternalAgentLaunchSpecV0.
  - ExternalAgentProcessCommandResolverV0.
  - ExternalAgentProcessRuntimePortV0.
Salida:
  - ExternalAgentProcessLaunchResultV0.
Invariantes:
  - LaunchExternalAgentProcessV0 no conoce Codex, Ollama, vLLM, API remota, DB, HOME real ni OAuth.
  - El resolver de comando es un puerto: puede mapear refs opacas a command_path/args/env/working_dir solo en el adaptador externo.
  - El runtime es un puerto: puede ser ProcessRuntimeConnectorV0 u otro conector compatible.
  - Antes de lanzar se revalida ExternalAgentLaunchSpecV0 y ProcessRuntimeLaunchRequestV0.
  - El resultado solo contiene snapshot publico y errores i18n; nunca devuelve comando, env, PID, rutas, tokens ni transcript.
Pruebas:
  - external_agent_process_adapter_v0_test.go.
```
