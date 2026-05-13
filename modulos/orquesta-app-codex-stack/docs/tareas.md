# Tareas: orquesta-app-codex-stack

## APP-CODEX-STACK-001

Objetivo: crear contexto local del mini-proyecto exterior sin tocar core ni
app-gateway productivo.

Estado: hecho.

Validacion:

- `git diff --check -- modulos/orquesta-app-codex-stack`

## APP-CODEX-STACK-002

Objetivo: documentar el contrato de composition para inyectar
`StartAppDirectorPortsV0` reales desde web/API/MCP.

Estado: hecho.

Validacion:

- `docs/contratos.md` define config, puertos e invariantes opt-in.

## APP-CODEX-STACK-003

Objetivo: preparar un arranque manual guardado para smokes Codex reales.

Estado: hecho.

Validacion:

- `arrancar_codex.sh` exige `ORQUESTA_CODEX_STACK_OPT_IN=1`;
- no define proveedor, modelo, DB, HOME, CODEX_HOME, PATH ni comando por
  defecto.

## APP-CODEX-STACK-004

Objetivo: implementar composition Go real del stack externo.

Estado: hecho.

Write-set aplicado:

- `ConfigV0` con puertos/stores/runtime/capacidad inyectados;
- `BuildStackV0` como composition root HTTP para web/API/MCP;
- resolutor de launch spec y waiter de ACK por puertos;
- test de API y web con runtime fake inyectado.

Validacion:

- `go test -count=1 ./modulos/orquesta-app-codex-stack`
- `go test -count=1 ./modulos/orquesta-app-director-intake ./modulos/orquesta-app-director-service ./modulos/orquesta-app-codex-stack`

Reglas cerradas:

- no activar esta ruta desde `cmd` ni app-gateway productivo;
- no crear defaults de DB/proveedor/modelo;
- no duplicar logica de `orquesta-app-director-service`;
- no filtrar nombre de conector real en payloads, evidence refs ni refs
  publicas del core.

## APP-CODEX-STACK-022

Objetivo: cablear `domain_work` como executor opt-in en el stack HTTP.

Estado: hecho.

Write-set aplicado:

- `ConfigV0.DomainWork` como puerto MCP generico;
- `BuildStackV0` pasa ese executor a `orquesta-app-gateway`;
- test de `/api/v0/domain-work` con executor fake inyectado.

Validacion:

- `go test -count=1 ./modulos/orquesta-app-codex-stack`

Reglas cerradas:

- no importar OPES ni conectores REST en este modulo;
- no crear DB, runtime, proveedor ni modelo por defecto;
- si no hay executor inyectado, el endpoint queda apagado por opt-in.

## APP-CODEX-STACK-005

Objetivo: smoke opt-in desde `/nueva-app` con Codex real, ACK de director y
registro durable de artefacto/fase.

Estado: hecho.

Validacion:

- `ORQUESTA_CODEX_STACK_OPT_IN=1` con `arrancar_codex.sh`;
- `ORQUESTA_CODEX_STACK_SMOKE=1 go test ./modulos/orquesta-app-codex-stack -run TestNuevaAppWebCodexStackRealOptInV0 -count=1 -timeout 300s -v`;
- modelo configurado por operador: `ORQUESTA_CODEX_MODEL=gpt-5.5`;
- evidencia local: `/tmp/orquesta-smokes/app-codex-stack-real-2/project`;
- resultado: PASS en 88.09s;
- repeticion tras modularizar tests: PASS en 82.09s con evidencia local
  `/tmp/orquesta-smokes/app-codex-stack-real-3/project`;
- ACK real escrito en `agent_ack.json`;
- documentos creados por el agente: `docs/arquitectura.md` y
  `docs/plan_microtareas.md`;
- `PhaseArtifactRegistered` observado en el sink del run;
- proceso real detenido al cerrar la prueba.

## APP-CODEX-STACK-006

Objetivo: validar `/nueva-app` con cohorte multiagente real lanzada por
Orquesta, no por coordinacion manual.

Estado: hecho.

Validacion:

- `ORQUESTA_CODEX_STACK_MULTIAGENT_SMOKE=1 go test ./modulos/orquesta-app-codex-stack -run TestNuevaAppWebCodexStackRealMultiagentOptInV0 -count=1 -timeout 420s -v`;
- comando ejecutado mediante `arrancar_codex.sh` con opt-in explicito;
- modelo configurado por operador: `ORQUESTA_CODEX_MODEL=gpt-5.5`;
- resultado: PASS en 148.207s;
- evidencia local:
  `/tmp/orquesta-smokes/app-codex-stack-multiagent-real-12/project`;
- Orquesta arranco 4 Codex reales en paralelo: `director`, `web`, `api` y
  `persistencia`;
- cada agente escribio `agent_ack.json` en runtime aislado;
- documentos creados: `docs/arquitectura.md`, `docs/plan_microtareas.md`,
  `docs/web.md`, `docs/api.md`, `docs/persistencia.md`;
- el drenaje final registro los ACKs como artefactos de fase y la prueba cerro
  procesos sin dejar sesiones vivas.

Regla cerrada:

- una firma de runtime sin cambios no equivale a bucle; para no parar agentes
  reales que estan pensando, `loop_detected` solo debe salir de una senal real
  de accion repetida, no de ausencia temporal de ACK.

## APP-CODEX-STACK-007

Objetivo: cerrar el salto director-documentacion -> decisiones ejecutables ->
programacion.

Estado: hecho.

Write-set aplicado:

- contrato de decisiones del director principal en helper separado;
- criterio de cierre que exige `director_decisions.json` solo al area
  `director`;
- clasificacion de rol `implementacion` como area `programacion`;
- test fake que simula un director escribiendo decisiones y valida que Orquesta
  abre `programacion` y arranca el agente de microtarea.

Validacion:

- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestCodexAreaV0|TestDirectorTaskV0|TestCodexStackV0ConsumeDecisionFileYArrancaProgramacion|TestCodexStackV0GatewayAPIYWebArrancanEquipoDirectorConRuntimeInyectado' -v`

Reglas cerradas:

- el director no decide por memoria de Codex sino por fichero de control
  consumido por puerto;
- las areas especializadas no toman control del workflow;
- programacion no hereda identidad ni contrato de director.

## APP-CODEX-STACK-008

Objetivo: endurecer el contrato textual de `director_decisions.json` tras
prueba real con JSON no aplicable.

Estado: hecho.

Write-set aplicado:

- prompt del director con campos obligatorios `decision_ref`, `command_type`,
  `phase_id`, `command_ref`, `summary` y `evidence_refs`;
- campos internos obligatorios por payload tipado, incluido `task_id` frente a
  `task_ref`;
- prohibicion explicita de aliases genericos `action`, `decision_id`,
  `payload` y `refs`;
- prohibicion explicita de `evidence_refs` con espacios, slash, rutas o texto
  humano;
- IDs encadenados exactos entre `request_vote`, `accept_decision`,
  `publish_function_contract` y `create_microtask`;
- vocabulario seguro para campos de decision: `datos sensibles` en vez de
  terminos prohibidos literales;
- enum literal de capacidad: `low`, `medium`, `high`, `xhigh`; la votacion
  inicial usa `high`;
- test del objetivo del director para fijar esos nombres.

Validacion:

- local focal ok tras repetir suite con la restriccion de refs compactas:
  `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestDirectorTaskV0|TestDrainRunV0ConsumeDecisionFileTardioYArrancaProgramacion|TestCodexStackV0WebDrenaACKMultiagenteTardio|TestCodexStackV0ConsumeDecisionFileYArrancaProgramacion|TestCodexStackV0GatewayAPIYWebArrancanEquipoDirectorConRuntimeInyectado' -v`;
- primera repeticion real fallo en
  `/tmp/orquesta-smokes/app-codex-stack-director-decisions-real-6/project`;
- segunda repeticion real fallo en
  `/tmp/orquesta-smokes/app-codex-stack-director-decisions-real-7/project`
  por refs encadenadas inconsistentes y vocabulario prohibido en decision.
- tercera repeticion real fallo en
  `/tmp/orquesta-smokes/app-codex-stack-director-decisions-real-8/project`
  por capacidad localizada como `alta` en vez del enum `high`.
- cuarta repeticion real ok en
  `/tmp/orquesta-smokes/app-codex-stack-director-decisions-real-10/project`.

Reglas cerradas:

- el stack no convierte formatos inventados por el agente;
- la autonomia se mantiene por DTO estricto y puerto de decisiones.

## APP-CODEX-STACK-009

Objetivo: hacer que el drenaje de un run existente consuma decisiones tardias
del director y lance agentes de programacion.

Estado: hecho.

Write-set aplicado:

- `DrainRunV0` reentra por `ContinueAppDirectorV0`;
- espera externa acotada en el borde del stack;
- dispatcher fino para `SendDirectorQuestion` no bloqueante;
- test de decision file tardio que exige abrir `programacion` y arrancar
  agente de microtarea;
- smoke real multiagente ahora exige al menos un descriptor de programacion.

Validacion:

- local focal ok:
  `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestDrainRunV0ConsumeDecisionFileTardioYArrancaProgramacion|TestCodexStackV0ConsumeDecisionFileYArrancaProgramacion|TestCodexStackV0WebDrenaACKMultiagenteTardio|TestNuevaAppWebCodexStackRealMultiagentOptInV0' -v`;
- integrada ok:
  `go test -count=1 ./modulos/orquesta-director ./modulos/orquesta-director-scheduler ./modulos/orquesta-runtime ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-codex-delivery ./orquestacionnucleoapp ./modulos/orquesta-app-director-service ./modulos/orquesta-app-director-intake ./modulos/orquesta-web ./modulos/orquesta-app-gateway ./modulos/orquesta-mcp ./modulos/orquesta-app-codex-stack`;
- smoke real opt-in con Codex ok en 194.108s:
  `/tmp/orquesta-smokes/app-codex-stack-director-decisions-real-10/project`.

Reglas cerradas:

- no se continua la orquestacion desde la sesion principal;
- Orquesta reentra sola usando puertos hexagonales y decisiones persistidas.

## APP-CODEX-STACK-010

Objetivo: cerrar el ciclo completo de programacion real, no solo el lanzamiento
de agentes de programacion.

Estado: hecho.

Write-set aplicado:

- `DrainRunV0` corta cuando lanza una nueva ola de agentes externos y deja la
  continuacion a otra reentrada;
- refs de entrega unicas por agente para evitar colisiones de ACK, mailbox y
  readiness;
- smoke real que espera ACKs de la ola de programacion;
- segundo drain que registra entregas de programacion;
- verificador de write-set compatible con ficheros y directorios;
- guarda de tamano para ficheros Go generados: maximo 300 lineas.

Validacion:

- local focal ok:
  `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestCodexStackRealSmokeVerifyProjectFileV0AceptaDirectorioConContenido|TestCodexStackRealSmokeVerifyGoFileSizesV0AceptaFicheroManejable|TestAgentPacketV0UsaDeliveryRefsUnicasPorAgente|TestNuevaAppWebCodexStackRealMultiagentOptInV0' -v`;
- integrada ok:
  `go test -count=1 ./modulos/orquesta-director ./modulos/orquesta-director-scheduler ./modulos/orquesta-runtime ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-codex-delivery ./orquestacionnucleoapp ./modulos/orquesta-app-director-service ./modulos/orquesta-app-director-intake ./modulos/orquesta-web ./modulos/orquesta-app-gateway ./modulos/orquesta-mcp ./modulos/orquesta-app-codex-stack`;
- smoke real `real-11`: fallo por bug del verificador, no por contrato de
  orquestacion;
- smoke real `real-12`: PASS en 500.099s con app Go/API/web generada,
  ACKs de programacion, pruebas Go de la app y cierre limpio de procesos.

Reglas cerradas:

- no meter bases de datos concretas por defecto en el core;
- no esperar programacion completa dentro del POST inicial;
- mantener microtareas y ficheros Go por debajo del tamano manejable;
- no continuar programacion desde la sesion principal: solo desde reentradas
  de Orquesta.

## APP-CODEX-STACK-011

Objetivo: convertir los resultados de programacion en estadisticas y decisiones
automaticas del director.

Estado: hecho para exposicion de estadisticas por puertos; pendiente separar
una politica productiva de rechazo/replanificacion automatica por entregas
invalidas.

## APP-CODEX-STACK-012

Objetivo: conectar el review gate de entregas de programacion al stack exterior
para que el director pueda aceptar o pedir cambios desde Orquesta.

Estado: hecho.

Write-set aplicado:

- `ConfigV0` exige `ReviewGate.FileEvidence` como puerto explicito;
- `buildDirectorPortsV0` inyecta `ReviewGateSource`;
- `reviewGateSourceV0` compone `CodexReviewGateObservationSourceV0`;
- el runtime fake de tests materializa ficheros reales declarados en ACK;
- test de stack valida aceptacion con fichero real y `changes_requested` por
  fichero de mas de 300 lineas.

Validacion:

- `go test -count=1 ./modulos/orquesta-runtime-codex-delivery`
- `go test -count=1 ./modulos/orquesta-app-codex-stack`

Reglas cerradas:

- el stack no decide internamente si una entrega vale: compone puertos;
- la lectura de ficheros queda en adaptador externo inyectado;
- no hay default de DB, proveedor, modelo, HOME ni path global;
- la revision no filtra detalles operacionales hacia el core.

Write-set aplicado:

- `stack_v0.go` inyecta `RunStore`, `ProcessRegistry` y `ProgressSource` en
  `orquesta.director.stats.v0`;
- `stack_flow_progress_stats_v0_test.go` prueba que el stack expone agentes con
  control de parada y progreso sin senal por web/API;
- `orquestacionnucleoapp` y `orquesta-mcp` publican `DirectorRunStatsV0` con
  `progress` y `closure`.

Validacion:

- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestCodexStackV0DirectorStatsIncluyeProcesoYProgresoPorPuertos|TestCodexStackV0GatewayAPIYWebArrancanEquipoDirectorConRuntimeInyectado' -v`;
- `go test -count=1 ./orquestacionnucleoapp ./modulos/orquesta-mcp`.

Pendiente separado:

- permitir que el director rechace/replanifique entregas con tests fallidos,
  write-sets incompletos, exceso de tamano o ausencia real de progreso;
- definir politica separada para specs no Go, por ejemplo OpenAPI largo.

Reglas de entrada:

- mantener hexagonalidad: estadisticas como puerto, no como lectura directa de
  ficheros del runtime desde web/MCP;
- no hardcodear DB, proveedor ni modelo;
- no crear ficheros grandes de agregacion.

## APP-CODEX-STACK-012

Objetivo: permitir que el usuario solicite cambios sobre una app existente y
que el director reciba la senal por Orquesta.

Estado: hecho.

Write-set aplicado:

- nuevo mini-proyecto `modulos/orquesta-app-change`;
- herramienta MCP `orquesta.apps.request_change.v0`;
- REST `POST /api/v0/apps/change` y
  `POST /api/v0/apps/{app_ref}/changes`;
- web `/app-change`;
- gateway con ruta dinamica sin romper `/api/v0/apps/spec` ni
  `/api/v0/apps/director`;
- stack Codex traduce el cambio a `AskDirector` y guarda outbox del director;
- store de cambios inyectado por puerto, sin DB concreta;
- fuente compuesta de decisiones que convierte cambios concretos en respuesta al
  director, votacion, contrato funcional, microtarea y vuelta a `programacion`;
- fuente de cambios abre `revision` cuando la entrega del cambio ya existe y el
  review gate generico acepta o pide rework con evidencia real.

Validacion:

- `go test -count=1 ./modulos/orquesta-app-change ./modulos/orquesta-app-change-director-source ./modulos/orquesta-mcp ./modulos/orquesta-web ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway ./modulos/orquesta-app-codex-stack`
- `TestCodexStackV0CambioProgresivoPasaReviewGate`.

## APP-CODEX-STACK-013

Objetivo: cerrar presupuestos por agente/tarea/ACK para que una app grande no
dependa de un timeout global.

Estado: hecho para helper/test local de stack; pendiente repetir smoke real
largo con Codex real.

Contexto real:

- el smoke real `/tmp/orquesta-smokes/multiagent-20260511143049/project`
  genero una app Go/API/web parcial que compila y pasa `go test ./...`;
- fallo por timeout global antes de que el tercer agente escribiera ACK;
- la tarea `HTTP/web/i18n` era demasiado amplia para el presupuesto de smoke.

Trabajo minimo:

- definir contrato de presupuesto por tarea: tiempo maximo esperado, tiempo sin
  actividad, tiempo desde ultimo cambio de fichero, tiempo desde ultimo ACK y
  severidad;
- separar en stats/MCP: `working`, `stalled`, `over_budget_but_active`,
  `over_budget_no_activity`, `ack_registered_cleanup`;
- hacer que el director pueda decidir split/replan cuando una tarea activa se
  pasa de presupuesto;
- exigir que el stack no mate directores protegidos por presupuesto automatico;
- mantener cleanup terminal tras ACK como operacion runtime, no como stop de
  workflow.

Trabajo aplicado en stack:

- el helper de app completa capea cada pasada de `DrainRunV0` para no consumir
  el timeout global en una sola espera externa;
- el criterio de progreso incluye secuencia, tareas, entregas, artefactos,
  assessments, descriptores y ACKs observables;
- si un descriptor de programacion tiene write-set completo, el proyecto Go
  compila y no hay ACK registrable, el smoke falla temprano con
  `project_compiles_but_ack_missing`;
- el smoke real multiagente exige una ola paralela de programacion con varios
  `AgentStarted` antes del primer `DeliveryRegistered`;
- `TestCodexStackRealSmokeDetectaACKFaltanteConProyectoCompilable` reproduce
  localmente el caso sin lanzar Codex real ni depender de intervencion manual.

Validacion esperada:

- test local de agente activo que supera tiempo pero modifica archivos: no se
  mata, se marca `over_budget_but_active`;
- test local de agente sin actividad: assessment y accion segun fase;
- smoke real acotado con tarea HTTP/web/i18n partida en microtareas menores;
- `go test ./modulos/orquesta-app-codex-stack -count=1`;
- bateria del nucleo/director/runtime/dispatch.

## APP-CODEX-STACK-014

Objetivo: exponer uso de recursos por agente para director, MCP y web.

Estado: hecho para puerto y acumulado por run; pendiente conector productivo de
proveedor.

Trabajo aplicado:

- `AgentUsageStatsProviderPortV0` en el nucleo;
- `DirectorAgentStatsV0.Usage` con modelo, capacidad, cuota y tokens;
- MCP `orquesta.director.stats.v0` acepta `include_agent_usage`;
- web proyecta `model_alias`, `capacity_level`, `quota_status`,
  `quota_remaining`, `quota_limit` y `total_tokens`;
- stack Codex publica modelo/capacidad desde configuracion y cuota
  `not_configured` hasta conectar proveedor real.
- `CodexStackAgentUsageMetricsProviderPortV0` permite inyectar metricas por
  agente sin que el stack conozca proveedor, HOME, OAuth ni API remota;
- `DirectorRunStatsV0.UsageSummary` acumula agentes observados, cuota, tokens y
  coste por run;
- web proyecta el resumen en `resumen.usage_*`.

Pendiente:

- conector real de cuota/tokens por proveedor o runtime;
- politica final para coste real si el proveedor no entrega precio;
- decidir politica de privacidad para mostrar modelo literal vs alias publico.

## APP-CODEX-STACK-015

Objetivo: soportar multiples apps simultaneas con cola global por prioridad.

Estado: hecho en contratos, MCP/REST, conector de memoria y runner global
interno.

Trabajo minimo:

- contrato `RunQueueReaderPortV0`, `RunQueuePriorityWriterPortV0` y
  `RunSchedulingCandidateV0`;
- ranking puro por `priority_score` descendente, aging/fairness y `updated_at`;
- filtro de runs no ejecutables: `paused`, `canceled`, `stopped`, `closed`;
- MCP/REST `orquesta.run_queue.priority.v0` con acciones `rank` y
  `set_priority`;
- `orquesta-run-memory` como conector de memoria thread-safe para pruebas;
- `orquesta-run-coordinator` como runner global puro que selecciona por ranking,
  respeta `RunControl` y drena por puerto;
- el stack registra automaticamente cada run arrancada en la cola global;
- `RunGlobalTickV0` en el stack ejecuta la run prioritaria sin conocer detalles
  internos de cada app;
- tests de stack que validan alta en cola, prioridad y pausa.

Cerrado separado en `orquesta-web` y gateway:

- vista web `/run-queue` para ranking de cola global y cambio de prioridad;
- progreso profundo por run sigue en `/director-stats` para no duplicar el
  contrato de estadisticas.

## APP-CODEX-STACK-017

Objetivo: cerrar el tick global multiapp sin abrir aun superficie MCP nueva.

Estado: hecho.

Trabajo aplicado:

- adaptador `RunGlobalTickV0` que compone `orquesta-run-coordinator` con
  `StackV0.DrainRunV0`;
- `QueuedArrancarDirectorExecutorV0` que encola la run al arrancar director;
- `RunQueueConfigV0` con `queue_ref`, prioridad inicial, limite de cola y
  ejecuciones maximas por tick;
- comandos de prioridad con `queue_ref` y `updated_at`;
- presupuesto conservador por tick para evitar que una run bloquee a las demas.

Validacion:

- `go test -count=1 ./modulos/orquesta-run-queue ./modulos/orquesta-run-memory ./modulos/orquesta-run-coordinator`
- `go test -count=1 ./modulos/orquesta-app-codex-stack`

Reglas cerradas:

- el coordinador global no importa stack, runtime, HTTP, MCP ni DB;
- el stack exterior traduce entre puertos, no mete politica global dentro del
  scheduler interno de cada run;
- la cola global queda alimentada desde el arranque real de app, no solo por
  llamadas manuales a `set_priority`.

## APP-CODEX-STACK-018

Objetivo: ejecutar una pasada supervisada de la cola global multiapp.

Estado: hecho.

Trabajo aplicado:

- nuevo mini-proyecto puro `orquesta-run-supervisor`;
- contrato `SuperviseRunsV0` con `MaxTicks`, `MaxExecutions`,
  `MaxRunsPerTick`, `StopOnNoExecution` y exclusion temporal de runs ya
  ejecutadas;
- `StackV0.RunGlobalSupervisorV0` compone el supervisor con
  `RunGlobalTickV0`;
- test de stack que arranca dos apps, ajusta prioridades y valida que la pasada
  ejecuta primero la de mayor prioridad y despues la siguiente, sin repetir la
  primera dentro de la misma pasada.

Validacion:

- `go test -count=1 ./modulos/orquesta-run-coordinator ./modulos/orquesta-run-supervisor ./modulos/orquesta-app-codex-stack`

Reglas cerradas:

- el supervisor no es daemon, no duerme y no crea goroutines;
- todo bucle tiene presupuesto explicito;
- el mini-proyecto puro no importa DB, HTTP, MCP, runtime ni stack;
- la repetition policy no muta la cola global.

## APP-CODEX-STACK-016

Objetivo: control humano/orquestador de una app completa: pause, resume, stop y
cancel.

Estado: hecho en contratos, MCP/REST, stack y gate del nucleo; pendiente
checkpoint/cierre terminal de estado para stop/cancel.

Trabajo minimo:

- mini-proyecto `orquesta-run-control` con estados `running`, `paused`,
  `stop_requested`, `stopped`, `cancel_requested`, `canceled`;
- MCP/REST `orquesta.runs.control.v0` con acciones `pause`, `resume`, `stop`
  y `cancel`;
- `orquesta-run-memory` como conector de memoria thread-safe;
- el nucleo consulta `RunControlReaderPortV0` antes de planificar y antes de
  despachar;
- sin estado de control tipado como `RunControlStateNotFoundErrorV0` se asume
  `running` por defecto;
- test de stack pausa una app por API y lee el estado por puerto.

Cerrado despues:

- escritura terminal `stopped/canceled` mediante
  `RunControlTerminalWriterPortV0` cuando todos los agentes vivos han sido
  confirmados o no habia agentes que parar;
- el cierre terminal queda bloqueado si todavia existe outbox
  `StopRuntimeAgent` pendiente.
- la politica autonoma del nucleo acepta `DirectorRunStatsV0` y usa senales de
  progreso/presupuesto para subir capacidad y acotar paralelismo.
- checkpoint real antes de stop/cancel no forzado queda cubierto en
  `orquesta-server-shutdown` por
  `TestShutdownServerV0PreparaCheckpointAntesDeStopNoForzado` y
  `TestShutdownServerV0NoPideStopSiCheckpointNoEstaListo`; el stack mantiene el
  puerto real con `TestStackShutdownCheckpointV0SolicitaAckSiHayAgenteEnVuelo`
  y `TestStackShutdownCheckpointV0RegistraCuandoTodosLosAgentesResponden`.

Pendiente separado:

- UI web de control queda cubierta por `/run-control`; cambio de prioridad por
  `/run-queue`.

## APP-CODEX-STACK-020

Objetivo: permitir que OPES y otras apps externas reciban artefactos producidos
por agentes de Orquesta.

Estado: hecho en bridge opt-in inicial.

Trabajo aplicado:

- `DomainWorkDeliveryBridgeConfigV0` con builder y ledger hexagonales;
- `DrainRunV0` intenta enviar artefactos tras observar ACKs y tambien reintenta
  deliveries ya registradas sin ledger;
- builder default usa `external_work.job_ref`, `input_fields`, ACK validado y
  fichero permitido por el write-set;
- mapeo generico de `draft_content_block -> content_block`,
  revisiones -> `block_revision` y fuentes -> `source`;
- cmd server activa el bridge solo si `ORQUESTA_OPES_BASE_URL` esta definido.

Validacion:

- `TestCodexStackV0OPESExternalWorkDeliveryEnviaArtefactoDomainWork`;
- `TestCodexLaunchSpecResolverV0MaterializaContextoDominioExterno`;
- `go test -count=1 ./modulos/orquesta-app-codex-stack`.

## APP-CODEX-STACK-021

Objetivo: permitir que OPES consulte progreso por job externo sin conocer la
estructura interna de la run.

Estado: hecho como proyeccion opt-in de stats.

Trabajo aplicado:

- `CodexStackExternalJobStatsSourceV0` resuelve `external_job_ref` desde
  `AppChangeStore`, deriva `task_ref/agent_ref` y proyecta status compacto;
- `/api/v0/director/stats` acepta `external_job_ref` y puede resolver `run_ref`
  mediante puerto MCP;
- `RunFileStoreV0` conserva `external_work.job_ref` e `input_fields` al
  persistir y reabrir estado;
- `orquesta.runs.control.v0` puede resolver `external_job_ref` y controlar la
  run asociada;
- el servidor usa ledger de artefactos persistente en
  `ORQUESTA_DOMAIN_DELIVERY_LEDGER_PATH` o en `StateDir` por defecto.

Validacion:

- `TestMCPDirectorStatsToolExecutorV0ResuelveRunPorJobExterno`;
- `TestCodexStackV0OPESExternalWorkRESTCreaMicrotareaSinWriteSetLocal`;
- `TestMCPRunControlExecutorV0ResuelveRunPorJobExterno`;
- `TestRunFileStoreAppChangePersistsAfterRecreateAndReplacesV0`;
- `TestFileDomainWorkArtifactSubmissionLedgerV0PersisteYRecupera`.

## APP-CODEX-STACK-019

Objetivo: preparar smoke opt-in de shutdown cooperativo real de Codex.

Estado: hecho como prueba opt-in; pendiente ejecucion manual con Codex real por
operador.

Trabajo aplicado:

- nuevo `TestCodexStackRealShutdownCheckpointOptInV0`, desactivado salvo
  `ORQUESTA_CODEX_STACK_SHUTDOWN_SMOKE=1`;
- el smoke lanza `/nueva-app`, exige proceso Codex vivo, llama
  `POST /api/v0/server/shutdown` con `forced=false` y valida
  `orquesta_shutdown_request.json`;
- el agente recibe instrucciones en el `AGENTS.md` temporal para esperar la
  request y responder con `agent_shutdown_checkpoint_ack.json`;
- una segunda llamada de shutdown exige checkpoint registrado o falla con
  diagnostico de pending/checkpoint y logs compactos;
- cleanup final usa el conector de proceso real ya inyectado.

Validacion:

- `go test ./modulos/orquesta-app-codex-stack -run Test.*Shutdown.* -count=1`

Reglas cerradas:

- no lanza Codex real por defecto;
- no introduce defaults de modelo, HOME, CODEX_HOME, PATH ni runtime;
- el endpoint REST/MCP sigue siendo el borde; el core solo ve refs compactas.
