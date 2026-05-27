# Decisiones locales: orquesta-runtime

Las decisiones de este archivo solo afectan a `orquesta-runtime`. Si afectan a otro modulo, deben elevarse al director y registrarse en `../../CONTRATOS.md`.

## RUNTIME-DEC-001

```text
Fecha: 2026-05-04
Decision: Definir RuntimeLaunchRequest v0 como contrato local validable en orquesta-runtime, con JSON Schema draft-07 solo para el request.
Motivo: El siguiente corte necesita coordinar core/capacity/runtime sin implementar transporte real; el contrato ya fue promovido globalmente por decision del director.
Alternativas:
  - Promover directamente a `modulos/CONTRATOS.md`: resuelto por decision del director fuera del write-set local.
  - Implementar adaptador runtime real: descartado; mezclaria contrato con transporte y filesystem.
  - Validar tambien response con schema: diferido hasta que exista caso de consumo desde core.
Impacto:
  - Runtime fija request, response, errores e invariantes locales.
  - CapacityDecisionV0 queda consumido solo como referencias opacas.
  - Multi-HOME separa identidad, proveedor, modelo, HOME, credencial, pool y cuota.
Contratos afectados: RuntimeLaunch v0; RuntimeLaunchRequestV0; RuntimeLaunchAcceptedV0; RuntimeLaunchErrorV0.
Estado: aceptada localmente; promocion global resuelta.
```

## RUNTIME-DEC-002

```text
Fecha: 2026-05-04
Decision: Implementar RuntimeLaunchRequestV0 como DTOs Go con validador puro local.
Motivo: Core ya puede emitir RuntimeLaunchRequest v0 globalmente y runtime necesita una barrera local antes de cualquier adaptador real.
Alternativas:
  - Solo validar por JSON Schema: insuficiente para invariantes relacionales como FunctionContract activo, modelo coincidente con CapacityDecision y safety cerrada.
  - Implementar puerto o runtime real: descartado; introduciria proceso, CLI/MCP/browser/filesystem/proveedor fuera del microcorte.
  - Consumir tipos privados de core/capacity: descartado; romperia el contrato hexagonal y el aislamiento entre modulos.
Impacto:
  - Runtime puede aceptar/rechazar requests por contrato local sin efectos externos.
  - Multi-HOME sigue expresado solo por referencias opacas.
  - Los errores publicos se devuelven como RuntimeLaunchErrorV0 con message_key i18n.
Contratos afectados: RuntimeLaunchRequestV0; RuntimeLaunchErrorV0.
Estado: aceptada localmente.
```

## RUNTIME-DEC-003

```text
Fecha: 2026-05-04
Decision: Definir AgentLauncherInboundV0 como puerto de entrada logico documental para validar LaunchRuntimeAgentRequestV0 compacto y traducirlo a RuntimeLaunchRequestV0 completo solo tras resolver refs por puertos versionados.
Motivo: orquesta-core-workflow ya emite un outbox logico `LaunchRuntimeAgent` con target `agent_launcher`; runtime necesita fijar la frontera contractual sin importar core-workflow ni implementar lanzamiento real.
Alternativas:
  - Importar DTOs de orquesta-core-workflow: descartado; romperia aislamiento entre modulos.
  - Ampliar RuntimeLaunchRequestV0 para aceptar payload compacto: descartado; mezclaria orden runtime completa con mensaje outbox.
  - Exigir que core-workflow envie FunctionContractV0, CapacityDecisionV0, HOME, modelo o credenciales: descartado; moveria decision runtime/capacity al workflow durable.
  - Implementar adaptador real ahora: descartado; faltan contratos de resolucion autorizada y quedaria fuera del microcorte documental.
Impacto:
  - Runtime documenta que el payload compacto actual no basta por si solo para lanzar.
  - FunctionContractV0, CapacityDecisionV0, binding/runtime refs y evidencias deben resolverse antes de construir RuntimeLaunchRequestV0.
  - Proveedor, modelo, HOME, credencial y cuota siguen expresados solo como referencias opacas.
  - La consulta de fuente autorizada queda resuelta por decision del director: FunctionContractResolverV0, CapacityDecisionResolverV0, RuntimeBindingResolverV0 y LaunchEvidenceResolverV0.
Contratos afectados: AgentLauncherInboundV0; LaunchRuntimeAgentRequestV0; RuntimeLaunchRequestV0.
Estado: aceptada localmente como contrato documental.
```

## RUNTIME-DEC-004

```text
Fecha: 2026-05-04
Decision: Implementar AgentLauncherInboundV0 y LaunchRuntimeAgentRequestV0 como DTOs Go con validadores puros y errores publicos de enriquecimiento pendiente.
Motivo: Runtime necesita una barrera local para la orden compacta `LaunchRuntimeAgent` emitida por outbox antes de introducir resolvers o cualquier adaptador real.
Alternativas:
  - Importar tipos de orquesta-core-workflow: descartado; romperia aislamiento hexagonal entre modulos.
  - Construir RuntimeLaunchRequestV0 completo ahora: descartado; faltan resolvers autorizados para FunctionContractV0, CapacityDecisionV0, RuntimeBindingV0 y evidencias mailbox/ACK/readiness.
  - Aceptar payload compacto sin validar refs opacas: descartado; permitiria filtrar proveedor, HOME, credenciales o rutas reales hacia runtime.
Impacto:
  - Runtime valida target_port, message_type, correlacion, idempotencia y payload compacto sin efectos externos.
  - task_ref y capacity_request_ref quedan como refs obligatorias para enriquecimiento posterior.
  - Los resolvers quedan definidos como interfaces contractuales sin implementacion.
Contratos afectados: AgentLauncherInboundV0; LaunchRuntimeAgentRequestV0; AgentLauncherInboundErrorV0.
Estado: aceptada localmente.
```

## RUNTIME-DEC-005

```text
Fecha: 2026-05-04
Decision: Dividir RuntimeLaunchRequestV0 en ficheros pequenos por responsabilidad: shell historico, tipos/constantes/errores, validador principal, validadores de componentes y helpers.
Motivo: runtime_launch_request_v0.go estaba en zona roja de tamano y la regla global exige sanear antes de seguir ampliando la zona.
Alternativas:
  - Mantener todo en un unico fichero: descartado; perpetua la deuda de tamano.
  - Crear abstracciones nuevas o cambiar el contrato: descartado; RUNTIME-005 es saneamiento sin comportamiento nuevo.
  - Compartir helpers con AgentLauncherInboundV0: descartado en este corte; exigiria tocar otro contrato local sin necesidad.
Impacto:
  - RuntimeLaunchRequestV0 conserva nombres publicos, JSON tags, constantes y codigos de error.
  - ValidateRuntimeLaunchRequestV0 conserva el flujo de validacion y tests existentes.
  - Los nuevos ficheros Go quedan por debajo del limite recomendado.
Contratos afectados: RuntimeLaunchRequestV0; RuntimeLaunchErrorV0.
Estado: aceptada localmente.
```

## RUNTIME-DEC-006

```text
Fecha: 2026-05-04
Decision: Implementar AgentStopperInboundV0 y StopRuntimeAgentRequestV0 como DTOs Go con validador puro para la intencion `StopRuntimeAgent` target `agent_launcher`.
Motivo: Runtime necesita una barrera local complementaria a AgentLauncherInboundV0 para aceptar o rechazar una parada logica antes de introducir lifecycle real.
Alternativas:
  - Implementar parada real ahora: descartado; mezclaria contrato con procesos, transporte, filesystem y estado operativo.
  - Reutilizar AgentLauncherInboundV0 con otro payload: descartado; confundiria lanzamiento con parada y mezclaria errores publicos.
  - Incluir proveedor, modelo, HOME, OAuth o handles reales: descartado; violaria opacidad, multi-HOME y frontera hexagonal.
Impacto:
  - Runtime valida target_port, message_type, correlacion, idempotencia y payload compacto sin efectos externos.
  - agent_request_id, run_id y evidence_refs quedan como referencias opacas.
  - Los errores publicos quedan versionados como AgentStopperInboundErrorV0.
Contratos afectados: AgentStopperInboundV0; StopRuntimeAgentRequestV0; AgentStopperInboundErrorV0.
Estado: aceptada localmente.
```

## RUNTIME-DEC-007

```text
Fecha: 2026-05-05
Decision: Implementar AgentProgressReportV0 como contrato puro de evidencia, no como mecanismo de parada.
Motivo: Director/core necesitan una senal compacta de progreso, estancamiento o bucle para decidir lifecycle sin que runtime mezcle observacion con accion destructiva.
Alternativas:
  - Matar procesos desde el reporte: descartado; violaria frontera de contrato puro y mezclaria evidencia con lifecycle real.
  - Incluir proveedor, modelo, HOME, OAuth, PID o transcript: descartado; filtraria detalles operativos y secretos fuera de refs opacas.
  - Reutilizar AgentStopperInboundV0: descartado; parada logica y reporte de evidencia tienen direcciones y responsabilidades distintas.
Impacto:
  - Runtime expone un DTO validable con status cerrado y contadores no negativos.
  - loop_detected requiere evidencia minima mediante no_progress_ticks o repeated_action_count mayor que cero.
  - Los errores publicos quedan versionados como AgentProgressReportErrorV0.
Contratos afectados: AgentProgressReportV0; AgentProgressReportErrorV0.
Estado: aceptada localmente.
```

## RUNTIME-DEC-008

```text
Fecha: 2026-05-05
Decision: Implementar RuntimeFakeLifecycleV0 como fake ejecutable, puro y en memoria para ejercitar lifecycle sin runtime real.
Motivo: El modulo necesita un harness local que pruebe el ciclo launch -> progress report -> loop_detected -> stop -> stopped usando los DTOs ya validados, sin cruzar la frontera con core-workflow ni con proveedores.
Alternativas:
  - Implementar procesos reales: descartado; mezclaria fake de contrato con transporte, filesystem, HOME y proveedor.
  - Importar tipos de orquesta-core-workflow: descartado; romperia aislamiento hexagonal.
  - Persistir estado: descartado; esta tarea requiere memoria pura y write-set pequeno.
Impacto:
  - Runtime dispone de snapshots compactos por agent_request_id para pruebas de lifecycle.
  - run_id inconsistente, agente inexistente e inbound invalido devuelven errores publicos versionados.
  - StopAgentV0 y LaunchAgentV0 repetido son idempotentes para el mismo agente/run.
  - Un progress tardio no reabre agentes parados; stopped queda como estado terminal del fake.
Contratos afectados: RuntimeFakeLifecycleV0; AgentLauncherInboundV0; AgentProgressReportV0; AgentStopperInboundV0.
Estado: aceptada localmente.
```

## RUNTIME-DEC-009

```text
Fecha: 2026-05-05
Decision: Cubrir RuntimeFakeLifecycleV0 con una prueba multiagente separada para verificar aislamiento por agent_request_id dentro del mismo run.
Motivo: El fake ya cubria el ciclo de un agente; faltaba probar que loop_detected y stop de un agente no contaminan snapshots de otro agente activo del mismo run.
Alternativas:
  - Ampliar el test monolitico existente: descartado; mezclaria el ciclo basico con aislamiento multiagente y haria menos claro el fallo.
  - Cambiar codigo productivo preventivamente: descartado; el comportamiento actual permite validar el contrato solo con prueba nueva.
  - Persistir estado o simular transporte: descartado; RUNTIME-009 mantiene memoria pura y frontera hexagonal.
Impacto:
  - Launch repetido del agente 2 debe devolver su snapshot estable.
  - stop_ref y last_report_ref quedan ligados al agent_request_id correspondiente.
  - La prueba no introduce runtime real, procesos, proveedor/modelo/HOME/OAuth, DB ni imports de otros modulos.
Contratos afectados: RuntimeFakeLifecycleV0; AgentLauncherInboundV0; AgentProgressReportV0; AgentStopperInboundV0.
Estado: aceptada localmente.
```

## RUNTIME-DEC-010

```text
Fecha: 2026-05-05
Decision: Tratar RuntimeBindingV0 como binding compatible con AgentHomeV0 consumido por runtime solo mediante refs opacas.
Motivo: Runtime necesita aceptar home/provider/model/credential ya decididos sin importar orquesta-capacity ni replicar su contrato de politica/capacidad.
Alternativas:
  - Importar tipos de orquesta-capacity: descartado; acoplaria runtime a internals de otro modulo.
  - Duplicar AgentHomeV0 completo en runtime: descartado; crearia dos fuentes de verdad para politica, capacidad y HOME.
  - Aceptar proveedores/modelos hardcodeados si cumplen el regex de ref: descartado; filtraria detalle operativo como si fuera ref opaca.
Impacto:
  - RuntimeLaunchRequestV0 mantiene provider_ref, model_ref, home_ref y credential_ref como referencias opacas.
  - El validador rechaza HOME real, emails/cuentas reales, tokens, proveedores y modelos hardcodeados.
  - Capacity conserva la propiedad de politica/capacidad; runtime solo verifica frontera contractual.
Contratos afectados: RuntimeLaunchRequestV0; RuntimeBindingV0.
Estado: aceptada localmente.
```

## RUNTIME-DEC-011

```text
Fecha: 2026-05-05
Decision: Implementar ProcessRuntimeConnectorV0 como adaptador local real de proceso para pruebas e2e controladas.
Motivo: Runtime necesita una evidencia ejecutable de launch/stop/snapshot sobre proceso real sin introducir proveedor, OAuth, HOME real, shell ni transporte externo.
Alternativas:
  - Reusar RuntimeFakeLifecycleV0: descartado; no ejecuta procesos y no cubre el riesgo e2e de os/exec.
  - Implementar un conector CLI/proveedor: descartado; mezclaria runtime local con proveedor, credenciales y HOME.
  - Usar sh -c o PATH heredado: descartado; filtraria shell/entorno real y ampliaria la superficie de ejecucion.
Impacto:
  - LaunchV0 exige command_path absoluto, env y working_dir explicitos.
  - El proceso hijo no hereda env real y stdout/stderr se descartan.
  - SnapshotV0 expone solo process_ref, session_ref, launch_ref, stop_ref y status; no expone PID, rutas, cwd, env, HOME ni output.
  - session_ref queda como identidad opaca de propiedad para evitar parar sesiones externas, directores o shells humanos.
  - StopV0 usa os.Interrupt y fallback Kill bajo contexto; la parada repetida conserva stop_ref.
Contratos afectados: ProcessRuntimeConnectorV0; ProcessRuntimeLaunchRequestV0; ProcessRuntimeSnapshotV0; ProcessRuntimeErrorV0.
Estado: aceptada localmente.
```

## RUNTIME-DEC-013

```text
Fecha: 2026-05-07
Decision: Permitir `PATH=` exacto como env privado explicito de ProcessRuntime.
Motivo: agentes reales lanzados por conectores opt-in necesitan encontrar herramientas del host como `go`; v1 ya tenia esta pieza operativa y la prueba real fallo cuando el proceso quedo con entorno totalmente vacio.
Alternativas:
  - Heredar os.Environ completo: descartado; reintroduce HOME, OAuth, tokens y estado local no gobernado.
  - Meter rutas de herramientas en prompts o specs publicos: descartado; filtra infraestructura al core.
  - Resolver todas las herramientas por ruta absoluta en Orquesta: descartado; acopla Orquesta al toolchain de cada app.
Impacto: `Env=nil` sigue prohibido, `Env: []string{}` sigue siendo entorno vacio real y solo `PATH=` exacto acepta rutas absolutas sin marcas de credencial o secreto. El snapshot publico no expone env.
Estado: aceptada localmente.
```

## RUNTIME-DEC-012

```text
Fecha: 2026-05-05
Decision: RuntimeLaunchRequestV0 requiere ContextBundleV0 valido antes de aceptar una nueva sesion.
Motivo: ningun agente debe arrancar con contexto implicito, prompt gigante o lectura global no controlada. El contexto lo prepara orquesta-context y runtime solo verifica que la orden trae el bundle publico esperado.
Alternativas:
  - Pasar contexto como summary o evidence_refs: descartado; mezclaria contenido de trabajo con refs y romperia limites.
  - Dejar que runtime lea AGENTS/docs por filesystem: descartado; filesystem debe ser conector, no nucleo del contrato.
  - Dejarlo opcional: descartado; permitiria lanzamientos sin contexto pequeno.
Impacto:
  - AgentLauncherResolvedDependenciesV0 declara ContextBundleV0 como dependencia resuelta.
  - RuntimeLaunchRequestV0 incluye context_bundle y valida que coincide con capacity y modulo objetivo del FunctionContract.
  - La materializacion de refs queda fuera de runtime y pertenece a conectores futuros.
Contratos afectados: RuntimeLaunchRequestV0; AgentLauncherInboundV0; ContextBundleV0.
Estado: aceptada localmente.
```

## RUNTIME-DEC-013

```text
Fecha: 2026-05-05
Decision: Implementar y documentar AgentStartPacketV0 como paquete neutral de arranque que deriva de RuntimeLaunchRequestV0 validado y ContextMaterializedBundleV0 materializado.
Motivo: Los conectores de agente necesitan una entrada comun y acotada que contenga tarea, contexto y refs de entrega sin conocer provider/model/HOME/OAuth/credenciales ni detalles operativos del runtime.
Alternativas:
  - Pasar RuntimeLaunchRequestV0 completo al conector: descartado; filtraria refs de proveedor/modelo/HOME/credencial que el agente no necesita para trabajar.
  - Dejar que cada conector lea filesystem o contexto por su cuenta: descartado; romperia orquesta-context como propietario del materializado y mezclaria filesystem con runtime.
  - Crear paquetes por proveedor: descartado; acoplaria el contrato a proveedores concretos y duplicaria politica de capacidad.
Impacto:
  - Runtime conserva RuntimeLaunchRequestV0 como orden operativa y expone AgentStartPacketV0 como paquete de trabajo neutral.
  - ContextMaterializedBundleV0 debe coincidir con el ContextBundleV0 del request antes de construir el paquete.
  - Los conectores futuros traducen el paquete por puertos hexagonales explicitos, sin DB hardcodeada ni secretos.
Contratos afectados: AgentStartPacketV0; RuntimeLaunchRequestV0; ContextMaterializedBundleV0.
Estado: aceptada localmente y promovida como resumen global minimo.
```

## RUNTIME-DEC-014

```text
Fecha: 2026-05-07
Decision: Definir ExternalAgentConnectorProfileV0 y ExternalAgentLaunchSpecV0 como frontera opt-in para conectores de agente real, sin ejecutar comandos ni copiar provider/model/HOME/credential refs al paquete del agente.
Motivo: Runtime necesita preparar la sustitucion del proceso hijo controlado por un conector real, manteniendo el acoplamiento a proveedores y credenciales fuera del contrato local.
Alternativas:
  - Reutilizar ProcessRuntimeConnectorV0 directamente: descartado; es un adaptador de proceso local para pruebas controladas, no contrato de agente real.
  - Pasar RuntimeLaunchRequestV0 completo al conector: descartado; filtraria provider_ref, model_ref, home_ref y credential_ref.
  - Hardcodear Codex/Ollama/vLLM/proveedor/modelo/HOME: descartado; violaria multi-HOME y la frontera hexagonal.
Impacto:
  - El perfil exige opt-in, refs opacas, connector_ref, runtime_kind, launch_mode, comando por refs y politica cerrada.
  - BuildExternalAgentLaunchSpecV0 es puro y no toca shell, PATH, HOME real, red externa ni credenciales.
  - AgentStartPacketV0 sigue siendo el unico paquete de trabajo para el agente y no contiene refs operacionales sensibles.
Contratos afectados: ExternalAgentConnectorProfileV0; ExternalAgentLaunchSpecV0; AgentStartPacketV0; RuntimeLaunchRequestV0.
Estado: aceptada localmente.
```

## RUNTIME-DEC-015

```text
Fecha: 2026-05-07
Decision: Implementar LaunchExternalAgentProcessV0 como puente opt-in desde ExternalAgentLaunchSpecV0 hacia un runtime real solo mediante resolver y puerto runtime inyectados.
Motivo: El nucleo necesitaba dejar de estar bloqueado por "faltan fronteras publicas" sin caer en hardcodear Codex, proveedor, HOME, OAuth, PATH, shell o un comando por defecto.
Alternativas:
  - Meter command_path/args/env en ExternalAgentLaunchSpecV0: descartado; filtraria detalle operacional al contrato publico y al paquete de agente.
  - Hacer que runtime resuelva provider/model/HOME por su cuenta: descartado; romperia multi-HOME y duplicaria decisiones de capacity/conectores.
  - Ejecutar ProcessRuntimeConnectorV0 directamente desde E2E: descartado para agente externo; no permitiria sustituir el resolver por Codex/Ollama/vLLM/API sin tocar el nucleo.
Impacto:
  - El spec permanece puro y opaco.
  - El resolver es la unica pieza autorizada para traducir refs a configuracion operacional, y queda fuera del core.
  - El runtime real se invoca por interfaz, por lo que puede ser proceso local, remoto, contenedor o API si respeta el puerto.
  - Los fallos salen como ExternalAgentConnectorErrorV0 con message_key i18n.
Contratos afectados: ExternalAgentProcessCommandResolverV0; ExternalAgentProcessRuntimePortV0; ExternalAgentProcessLaunchResultV0; ExternalAgentLaunchSpecV0; ProcessRuntimeConnectorV0.
Estado: aceptada localmente.
```

## RUNTIME-DEC-016

```text
Fecha: 2026-05-07
Decision: Definir AgentProgressHeartbeatV0 y BuildAgentProgressReportFromHeartbeatV0 como contrato puro para convertir snapshot publico de proceso y counters de heartbeat en AgentProgressReportV0.
Motivo: Director/scheduler necesitan distinguir trabajo vivo, sin progreso y bucle sin inspeccionar internals del agente ni depender de detalles del adaptador real.
Alternativas:
  - Ampliar LaunchV0 o el proceso real: descartado; mezclaria lanzamiento con supervision.
  - Leer transcripts, prompts o estado interno del agente: descartado; romperia la frontera de evidencia compacta.
  - Meter politica de proveedor, DB o capacidad en runtime: descartado; runtime solo normaliza counters y refs opacas.
Impacto:
  - El reporte conserva status cerrado, no_progress_ticks, repeated_action_count y evidence_refs opacas.
  - La politica local solo contiene umbrales numericos para stalled y loop_detected.
  - El adaptador de proceso no expone command_path, env, PID, HOME, proveedor, modelo, OAuth, DB ni transcript.
Contratos afectados: AgentProgressHeartbeatV0; AgentProgressHeartbeatPolicyV0; AgentProgressReportV0; ProcessRuntimeSnapshotV0.
Estado: aceptada localmente.
```

## RUNTIME-DEC-017

```text
Fecha: 2026-05-08
Decision: Implementar LaunchRuntimeAgentToRuntimeLaunchRequestV0 como transformacion pura entre la orden compacta del outbox y RuntimeLaunchRequestV0.
Motivo: El lanzamiento real no debe depender de pegamento implicito en cmd ni de una demo manual. La frontera necesita un paso validable que combine AgentLauncherInboundV0 con dependencias ya resueltas por puertos autorizados.
Alternativas:
  - Resolver FunctionContract, CapacityDecision, RuntimeBinding y ContextBundle dentro de la transformacion: descartado; mezclaria puertos externos y almacenamiento con un helper de contrato.
  - Pasar AgentLauncherInboundV0 directamente a conectores Codex/CLI: descartado; saltaria RuntimeLaunchRequestV0 y duplicaria reglas de seguridad/delivery.
  - Construir RuntimeLaunchRequestV0 en cmd: descartado; repetiria el problema legacy de logica de orquestacion en el entrypoint.
Impacto:
  - La transformacion fija source, delivery y safety contractuales y valida la salida con ValidateRuntimeLaunchRequestV0.
  - Las dependencias externas siguen entrando por AgentLauncherResolvedDependenciesV0; DB, proveedor, HOME, modelo y credenciales no se resuelven aqui.
  - El siguiente corte puede materializar contexto y construir AgentStartPacket/ExternalAgentLaunchSpec sin inventar campos.
Contratos afectados: AgentLauncherInboundV0; AgentLauncherResolvedDependenciesV0; RuntimeLaunchRequestV0.
Estado: aceptada localmente.
```

## RUNTIME-DEC-018

```text
Fecha: 2026-05-08
Decision: Crear BuildClosedExternalAgentConnectorProfileV0 y ClosedExternalAgentSecurityPolicyV0 como builder comun para perfiles externos cerrados.
Motivo: Cada conector opt-in necesita generar ExternalAgentConnectorProfileV0 con la misma politica de seguridad. Duplicar esa politica por conector aumenta el riesgo de abrir shell, entorno, HOME, secretos o transcripts por error.
Alternativas:
  - Mantener perfiles escritos a mano en cada conector: descartado; duplica invariantes y facilita divergencias.
  - Dar defaults por proveedor: descartado; reintroduce proveedor/modelo en runtime y rompe hexagonalidad.
  - Relajar el validador para aceptar detalles operativos: descartado; filtraria informacion sensible al contrato publico.
Impacto:
  - Los conectores solo aportan refs opacas y runtime_kind; la politica cerrada es comun.
  - Las refs invalidas no se ocultan: el builder no valida ni corrige, el validador sigue rechazando detalles prohibidos.
  - Las fixtures del nucleo usan el builder para probar el mismo contrato comun que usaran los conectores.
Contratos afectados: ExternalAgentConnectorProfileV0; ExternalAgentSecurityPolicyV0.
Estado: aceptada localmente.
```

## RUNTIME-DEC-019

```text
Fecha: 2026-05-09
Decision: Permitir rutas absolutas reales de workspace en ProcessRuntimeLaunchRequestV0 para command_path y working_dir, manteniendo su caracter privado.
Motivo: Orquesta debe crear directorios de trabajo reales del proyecto y lanzar agentes sobre ellos; bloquear cualquier ruta bajo el workspace del operador obliga a usar enlaces temporales y falsea las pruebas reales.
Alternativas:
  - Seguir usando /tmp o symlinks: descartado; oculta el problema y no prueba el contrato operativo real.
  - Relajar snapshots publicos: descartado; command_path, working_dir, PID, HOME y entorno siguen sin serializarse.
  - Aceptar marcadores explicitos de HOME o credenciales: descartado; solo se acepta una ruta absoluta ya resuelta por configuracion operacional.
Impacto:
  - LaunchV0 puede usar directorios creados dentro del proyecto.
  - Se siguen rechazando shell, entorno heredado, secretos, credenciales, HOME simbolico (`~`, `$HOME`, `%USERPROFILE%`) y rutas con marcadores sensibles.
  - La evidencia publica conserva solo refs opacas; no filtra rutas reales.
Contratos afectados: ProcessRuntimeConnectorV0; ProcessRuntimeLaunchRequestV0; ProcessRuntimeSnapshotV0.
Estado: aceptada localmente.
```

## RUNTIME-DEC-020

```text
Fecha: 2026-05-09
Decision: Exponer ValidateProcessRuntimeLaunchRequestV0 como validador publico del request privado de proceso.
Motivo: Los conectores opt-in deben poder rechazar un request no ejecutable antes de materializar wrappers, prompts o ficheros de control.
Alternativas:
  - Mantener solo la validacion interna de LaunchV0: descartado; dejaba artefactos preparados aunque el proceso nunca pudiera arrancar.
  - Duplicar reglas en cada conector: descartado; rompe la fuente unica de verdad sobre shell, env y rutas sensibles.
Impacto:
  - Codex y futuros conectores validan con la misma regla que usara LaunchV0.
  - El request sigue sin serializar command_path, env, working_dir, HOME, proveedor, modelo ni credenciales.
Contratos afectados: ProcessRuntimeLaunchRequestV0; ProcessRuntimeConnectorV0.
Estado: aceptada localmente.
```

## CONSULTA AL DIRECTOR

```text
Modulo origen: orquesta-runtime
Modulos afectados: orquesta-core, orquesta-capacity, contratos globales.
Bloqueo: Resuelto; `../../CONTRATOS.md` ya incluye RuntimeLaunchRequest v0 como contrato compartido.
Pregunta concreta: Promovemos un resumen minimo de RuntimeLaunchRequest v0 al contrato global para que core pueda emitirlo y capacity cierre AgentHomeV0?
Opcion recomendada: Promover solo el resumen global minimo en una microtarea dedicada; mantener schemas/fixtures como canon local de runtime.
Impacto: Evita que core o capacity inventen campos de runtime; conserva proveedor, modelo, HOME y OAuth como referencias opacas.
Decision del director: RuntimeLaunchRequest v0 queda promovido a contrato compartido en `modulos/CONTRATOS.md`; orquesta-core es consumidor autorizado; orquesta-capacity solo aporta referencias opacas via CapacityDecision v0; resume y runtime real quedan fuera.
Estado: cerrada.
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

## RUNTIME-DEC-021

```text
Fecha: 2026-05-27
Decision: T207 queda como contrato experimental `opt_in_low_priority`.
Motivo: la rotacion puede ser util para sesiones largas, pero no debe cortar
procesos vivos ni convertirse en rail estricto sin evidencia real de relevo.
Alternativas:
  - Cortar por presupuesto/contexto sin handoff: descartado; perderia contexto y podria duplicar trabajo.
  - Meter `session_epoch` en core puro: descartado; depende de runtime/adaptador.
Impacto:
  - `EvaluateRuntimeSessionRotationV0` solo permite relevo si el handoff es completo y seguro.
  - `StopCurrentAllowed` permanece `false`.
  - HOME, secretos, rutas absolutas, proveedor, modelo, PID, prompts y transcripts se rechazan o quedan fuera.
Contratos afectados: RuntimeSessionRotationRequestV0; RuntimeSessionRotationHandoffV0; RuntimeSessionRotationDecisionV0.
Estado: aceptada localmente.
```
