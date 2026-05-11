# Plan de cierre del nucleo Orquesta

Fecha: 2026-05-10.

Objetivo: dejar el nucleo listo para una prueba real de app completa sin que la
sesion humana haga de orquestador. Orquesta debe arrancar director, consumir sus
decisiones, lanzar agentes, registrar entregas, aplicar cambios a mitad,
validar, cerrar o bloquear cierre con causa observable.

## Reglas no negociables

- Hexagonal: core sin runtime, proveedor, DB, HOME ni credenciales.
- DB y runtime siempre por conectores inyectados.
- i18n por defecto cuando haya texto visible.
- Ficheros Go manejables; objetivo menor de 300 lineas salvo excepcion
  justificada.
- Problemas grandes se cierran con unidades de trabajo acotadas. Microtarea es
  el modo por defecto para riesgo/debug, pero el director puede usar tareas
  medianas o grandes si tienen contrato, write-set, checkpoints y tests claros.
- Las pruebas reales validas pasan por agentes arrancados por Orquesta.

## Frentes abiertos

| Frente | Responsable | Alcance | Estado |
| --- | --- | --- | --- |
| Cierre en servicio director | Copernicus | Bloquear/permitir `register_final_validation` y `close_run` con evidencias reales | hecho |
| Contrato del director | Banach | Prompt/contrato para fases completas y cierre sin falsos positivos | hecho |
| Estadisticas MCP/API | Kierkegaard | Exponer progreso y causas de bloqueo de cierre | hecho |
| Prueba stack app+cambio | Zeno | Arranque por API/MCP, drain, app-change y agentes derivados | hecho |
| DTO de cierre director | sesion principal | Validar comandos compactos de cierre | hecho |
| Integracion stats en stack Codex | sesion principal | Inyectar registro de procesos y fuente de progreso al gateway web/API | hecho |
| Review gate de entregas Codex | Darwin + sesion principal | Aceptar o pedir cambios por ACK/tests/write-set/tamano con evidencias reales | hecho |
| Replan tras retrabajo de revision | sesion principal + Epicurus | Convertir `RequestRework` en `RecordReplanDecision`, `OpenPhase(programacion)` y followups explicitos | hecho |
| Decision `close_task` del director | Jason + sesion principal | Permitir que el director cierre una tarea tras revision aceptada sin comandos manuales | hecho |
| Replan de review en stack Codex | Volta + sesion principal | Cablear `ReviewReworkReplanSource` real derivando tarea desde receipt de entrega | hecho |
| Prueba progresiva de rework | Pauli + sesion principal | Verificar `changes_requested -> rework -> replan -> agente nuevo` sin intervencion manual | hecho |
| Split de rework en microtareas | Anscombe + sesion principal | Convertir `split_task` en `CreateMicrotask` y planificar cada microtarea sin pasos manuales | hecho |
| Smoke real app completa opt-in | Herschel + sesion principal | Arrancar desde `StartAppDirectorV0`, director real y microtareas API/web/docs con validacion de calidad | hecho |
| Wizard y cambio a mitad | Harvey + sesion principal | Intake conversacional por pasos y evento de cambio que alimenta replan por director | hecho |
| Contexto de decision MCP | Popper + sesion principal | Exponer `decision_context` compacto para director/API/MCP/web | hecho |
| Auditoria legacy final | Einstein + sesion principal | Decidir que queda reusable de v1/v2 sin copia masiva ni DB/CLI acoplado | hecho |

## Criterios de cierre

- `go test -count=1 ./modulos/... ./modulos/orquesta-orchestration-core` pasa.
- Existe prueba de app completa normal que no permite cerrar si solo hay docs.
- Existe prueba de app completa normal que permite cerrar con programacion,
  revision aceptada y validacion.
- Existe prueba de cambio a mitad que replanifica por director sin pasos
  manuales.
- Existe prueba de revision `changes_requested` que no cierra, registra rework y
  vuelve a programacion con un agente nuevo.
- Existe prueba de revision `changes_requested` que puede dividir trabajo en
  microtareas nuevas por `split_task` y programarlas despues.
- El director puede emitir `close_task`; ningun cierre normal depende de tareas
  cerradas preparadas manualmente en tests.
- MCP/API puede explicar progreso, bloqueo de cierre y contexto de decision con
  refs compactas.
- Docs locales registran decisiones, tareas y pruebas.
- El corte `revision no aceptada -> RequestRework durable` queda documentado
  en scheduler, tick-input y app de nucleo sin introducir DB/runtime/proveedor.

## Resultado de cierre

- `register_final_validation` y `close_run` existen como DTOs compactos del
  director y se traducen a comandos publicos del workflow.
- `ContinueAppDirectorV0` bloquea cierres de app completa normal sin contratos,
  programacion, revisiones aceptadas y validacion final.
- `DirectorRunStatsV0` expone `progress` y `closure`; MCP/API/web lo consultan
  por puertos hexagonales.
- El stack Codex publica stats con control de proceso y progreso cuando el
  operador inyecta `ProcessRegistry` y `ProgressSource`.
- El stack Codex inyecta `ReviewGateSource` y exige evidencia real de ficheros
  por puerto; una entrega valida se acepta y una entrega demasiado grande queda
  en `changes_requested`.
- `ReviewGateCandidateProviderV0` genera `RequestRework` para resultados
  `changes_requested` o `rejected`, y genera `AcceptReview` solo para
  `accepted`. Tick Input expone `ReworkRequests` en el snapshot para dedupe.
- `ReviewReworkReplanCandidateProviderV0` consume planes por puerto, usa
  `ReworkRequestRef` como `SourceRef`, reabre `programacion` con `OpenPhase`
  explicito y continua con capacidad/agente tras decision de capacidad.
- `ReviewReworkReplanCandidateProviderV0` tambien acepta planes `split_task`,
  persiste las tareas por `WorkflowTaskWriterPortV0`, genera
  `CreateMicrotask` por scheduler y deja esas microtareas planificables por
  `WorkflowTaskCandidateProviderV0`.
- El stack Codex inyecta `ReviewReworkReplanSourceV0`; el plan deriva la tarea
  desde el receipt de la entrega revisada, mantiene el plan hasta pedir el
  agente de retry y marca fallback si no hay descriptor usable.
- El director externo ya puede emitir `close_task`; el puente lo traduce a
  `CloseTask`, respetando la separacion `AcceptReview -> CloseTask ->
  RegisterFinalValidation`.
- El director externo tambien puede emitir `ask_director`, `ask_user`,
  `request_capacity`, `request_agent`, `request_rework` y
  `record_replan_decision` como DTOs compactos traducidos por puente.
- `orquesta.director.stats.v0` devuelve `stats` y `decision_context`; el
  director puede consultar progreso por fase/tarea/agente, refs opacas de
  proceso/sesion, actividad reciente, bloqueos, rework/replan, duraciones y
  quietud sin recibir runtime, DB, HOME ni proveedor.
- El wizard de intake puede avanzar por pasos, generar preguntas pendientes y
  normalizar respuestas hacia `AppSpec`; los cambios a mitad se registran como
  evento de app y se consumen por source de director para replan.
- El smoke opt-in de programacion real arranca desde `StartAppDirectorV0`,
  produce `director_decisions.json`, lanza microtareas API REST Go, web i18n y
  documentacion con agentes Codex reales, y deja ACKs de director/API/web/docs.
- La observacion de entregas Codex distingue dos modos de verificacion de
  worktree: `strict_diff` para agentes aislados y `ack_files` para worktree
  compartido concurrente. En modo concurrente se valida ACK, write-set y
  existencia de ficheros declarados; el diff global se reserva para revision.
- La auditoria legacy final concluye que no queda codigo v1/v2 para copiar tal
  cual; solo quedan como frentes futuros deploy con rollback, conectores no
  Codex/locales y capacidad/pools, siempre como conectores o policies pequenas.
- La politica de granularidad queda fijada: `WorkflowTaskV0` es la unidad
  canonica; `CreateMicrotask` conserva el nombre durable v0 pero no obliga a
  que todo trabajo productivo sea minimo.
- La prueba progresiva cubre arranque por API, drenaje, cambio a mitad y agente
  derivado sin coordinacion manual.
- La prueba progresiva de rework cubre `changes_requested -> RequestRework ->
  RecordReplanDecision -> OpenPhase(programacion) -> RequestCapacity ->
  CapacityDecided -> RequestAgent`, y confirma que no se acepta revision ni se
  cierra el run por error.
- La prueba progresiva de split cubre `changes_requested -> RequestRework ->
  split_task -> CreateMicrotask -> OpenPhase(programacion) ->
  RequestCapacity/RequestAgent` por microtarea, sin intervencion manual.

## Bloqueos estructurales detectados en el cierre

- El workflow ya soporta `CloseTask`, pero el DTO del director no lo exponia. Sin
  ese puente, una app completa podia tener revision aceptada y aun asi no quedar
  lista para `register_final_validation` salvo que un test preparase
  `closed_tasks` manualmente.
- El nucleo ya acepta planes de rework por puerto, pero el stack Codex no los
  inyectaba. Sin ese conector, `changes_requested` quedaba durable como
  `RequestRework`, pero no podia relanzar programacion sin ayuda externa.
- El plan de rework debe derivar la tarea desde la entrega revisada. Usar siempre
  la primera tarea del run es un fallback debil y no sirve para apps grandes.

## Estado de cierre del nucleo

### Ya funciona

- El nucleo puede arrancar una app por `StartAppDirectorV0`, avanzar el ciclo del
  director y traducir decisiones compactas a comandos del workflow.
- El cierre normal exige contratos, programacion, revision aceptada,
  `CloseTask` y validacion final; no debe cerrar por tener solo documentacion.
- El flujo de revision no aceptada queda durable como `RequestRework` y puede
  reabrir programacion mediante `RecordReplanDecision` y `OpenPhase`.
- El rework puede relanzar un agente nuevo o dividir trabajo con `split_task`,
  persistiendo microtareas programables por el scheduler.
- MCP/API/web pueden exponer progreso, bloqueo de cierre y `decision_context`
  compacto sin filtrar runtime, DB, HOME ni proveedor al core.
- El wizard de intake y los cambios a mitad generan eventos consumibles por el
  director para replanificar.
- La auditoria legacy final queda cerrada como decision de no copiar v1/v2 en
  bloque; lo reusable pasa por conectores o policies pequenas.

### Validado con tests locales

- `go test -count=1 ./modulos/... ./modulos/orquesta-orchestration-core`: OK en
  2026-05-10.
- La suite local cubre los contratos principales de cierre: app completa normal,
  bloqueo por ausencia de evidencias suficientes, revision aceptada, rework,
  split en microtareas, cambio a mitad, stats/contexto de decision y smoke
  opt-in preparado.
- La validacion local no sustituye la prueba real con agentes Codex ni confirma
  consumo de cuota, latencias, credenciales, filesystem productivo o estabilidad
  del proveedor externo.

### Smoke real Codex

- Smoke real Codex opt-in:

```bash
ORQUESTA_CODEX_PROGRAMMING_TEAM_SMOKE=1 \
go test ./modulos/orquesta-runtime-codex-delivery \
  -run TestProgrammingTeamCodexRealOptInV0 -count=1 -timeout 520s -v
```

- Resultado real ejecutado en esta sesion:
  `/tmp/orquesta-smokes/programming-team-success-20260510`.
- Orquesta arranco director real, consumio 9 decisiones, creo microtareas,
  lanzo agentes Codex reales para API/web/docs en paralelo y recibio ACKs
  `completed` de los cuatro agentes.
- La app generada contiene `internal/api/handler.go`,
  `internal/api/handler_test.go`, `server/main.go`, `web/index.html`,
  `README.md`, `docs/arquitectura.md` y `docs/plan_microtareas.md`; todos los
  ficheros quedaron por debajo de 300 lineas.
- La app generada paso `GOCACHE=/tmp/orquesta-go-build go test ./...`.
- Fallo detectado y corregido: el verificador del smoke exigia literalmente
  `"/api/"`; ahora reconoce API REST por `net/http`, `/health`, `/events` y
  metodos GET/POST.
- Repeticion posterior corregida:
  `/tmp/orquesta-smokes/programming-team-pass4-20260510`.
- Resultado: PASS en 250.52s. Orquesta arranco director real, el director
  produjo arquitectura y plan, Orquesta lanzo agentes Codex reales de API, web y
  docs en paralelo, recibio ACKs de los cuatro agentes y valido la app.
- La app generada paso `GOCACHE=/tmp/orquesta-go-build go test ./...`.
- Calidad observada: API REST Go con `GET /health`, `POST /events`,
  `GET /events`, memoria local por puerto, `description` preservada y probada;
  web estatica i18n ES/EN; README con endpoints y payload.
- Tamano observado: `handler.go` 185 lineas, `handler_test.go` 83,
  `server/main.go` 28, `web/index.html` 270, README 79, arquitectura 29 y plan
  41.

### Criterio de 100%

El cierre del nucleo esta al 100% cuando se cumplen simultaneamente:

- La suite local indicada pasa en limpio.
- El smoke real Codex opt-in pasa, o queda bloqueado con causa externa
  documentada y reproducible tras haber demostrado arranque real de director y
  agentes.
- Una app completa no cierra si solo hay docs o si la revision queda en
  `changes_requested`.
- Una app completa si cierra tras programacion, revision aceptada, `CloseTask` y
  `RegisterFinalValidation`, sin preparar estados cerrados manualmente en tests.
- Un cambio a mitad y un rework replanifican por director sin pasos manuales.
- MCP/API/web explican estado, progreso, bloqueo y contexto de decision con refs
  compactas.
- Las decisiones y pendientes operativos quedan documentados sin introducir
  acoplamientos de DB/runtime/proveedor en el core.

## Validacion local de cierre

Ejecutado tras integrar los subfrentes:

```bash
go test -count=1 ./modulos/... ./modulos/orquesta-orchestration-core
```

Resultado: OK. Revalidado en esta sesion el 2026-05-11 con el mismo comando.

Tambien se verifico `git diff --check` en los frentes tocados y que no quedan
ficheros Go por encima de 300 lineas en `modulos/` ni en
`modulos/orquesta-orchestration-core/`.

El smoke real largo de Codex ya se repitio con cuota disponible y paso en
`/tmp/orquesta-smokes/programming-team-pass4-20260510`.

## Actualizacion 2026-05-11: smoke app Go/API/web y cierres de nucleo

Se ejecuto un smoke real de `orquesta-app-codex-stack` con `gpt-5.5` para crear
una app Go con API REST y web desde `/nueva-app`.

Resultado:

- workdir: `/tmp/orquesta-smokes/multiagent-20260511143049/project`;
- FAIL por timeout global de 1200s;
- Orquesta lanzo agentes Codex reales en paralelo para direccion, API, web y
  persistencia;
- el director emitio arquitectura, plan y decisiones;
- Orquesta lanzo agentes reales de programacion;
- completaron con ACK valido bootstrap Go y dominio/casos de uso/puertos;
- el tercer agente genero HTTP/web/i18n, pero no llego a ACK antes del timeout;
- la app parcial paso `go test ./...`;
- todos los ficheros Go generados quedaron por debajo de 300 lineas.

El resultado no se considera cierre total, pero si evidencia util:

- el nucleo ya orquesta fases reales y agentes Codex reales sin que la sesion
  principal programe la app;
- el contrato de microtareas funciona cuando la tarea esta bien acotada;
- falta presupuesto por agente/tarea/ACK para apps grandes.

Cierres aplicados tras la prueba:

- parada segura por identidad de proceso antes de llamar al runtime;
- proteccion de directores frente a `StopRuntimeAgent` automatico;
- cleanup terminal tras ACK registrado sin contaminar `StoppedAgents` ni emitir
  `AgentStopConfirmed`;
- tests locales actualizados para fijar estas reglas.

Validacion posterior:

```bash
go test ./modulos/orquesta-app-codex-stack -count=1
go test ./modulos/orquesta-orchestration-core ./modulos/orquesta-director ./modulos/orquesta-director-scheduler ./modulos/orquesta-runtime ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-outbox-dispatch -count=1
```

Resultado: OK.

## Actualizacion 2026-05-11: runner global multiapp interno

Se cerro el primer runner global real para que Orquesta pueda avanzar multiples
apps por prioridad sin que la sesion humana elija manualmente cada run.

Trabajo aplicado:

- nuevo mini-proyecto `modulos/orquesta-run-coordinator`;
- el coordinador lee candidatos por `RunQueueReaderPortV0`, rankea con
  `orquesta-run-queue`, valida `RunControlReaderPortV0` y drena por
  `RunDrainerPortV0`;
- el control ausente tipado sigue significando `running`; `QueueReader` y
  `Drainer` son obligatorios;
- `RunQueuePriorityCommandV0` acepta `queue_ref` y `updated_at`;
- `orquesta-run-memory` puede crear candidatos nuevos en una cola concreta al
  ajustar prioridad;
- el stack registra cada run arrancada desde `/api/v0/apps/director` en la cola
  global;
- `StackV0.RunGlobalTickV0` ejecuta la run prioritaria y respeta pausas antes
  de drenar.

Decision:

- no se abre aun `orquesta.run_coordinator.tick.v0` por MCP. MCP queda para
  control, prioridad y stats; el tick es API interna hasta estabilizar el
  contrato y evitar superficie publica prematura.

Validacion:

```bash
go test -count=1 ./modulos/orquesta-run-queue ./modulos/orquesta-run-memory ./modulos/orquesta-run-coordinator
go test -count=1 ./modulos/orquesta-app-codex-stack
go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./modulos/orquesta-http-gateway ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-director-service
```

Resultado: OK.

Pendiente para prueba real grande:

- usar `RunGlobalTickV0` en un loop supervisor opt-in que alterne apps por
  prioridad;
- conectar stats finales de tokens/cuota;
- checkpoint y parada fisica coordinada para `stop/cancel`;
- vista web de cola multiapp y progreso por run.

## Actualizacion 2026-05-11: pasada supervisada multiapp

Se cerro el siguiente nivel sobre el tick global: una pasada supervisada con
presupuesto explicito.

Trabajo aplicado:

- nuevo mini-proyecto `modulos/orquesta-run-supervisor`;
- `SuperviseRunsV0` ejecuta varios ticks mediante puerto inyectado;
- limites: `MaxTicks`, `MaxExecutions`, `MaxRunsPerTick` y
  `StopOnNoExecution`;
- sin sleeps, goroutines, daemon interno, DB, HTTP, MCP, runtime ni stack;
- `orquesta-run-coordinator` acepta `ExcludeRunRefs`;
- por defecto, el supervisor excluye temporalmente las runs ya ejecutadas para
  no repetir la misma app dentro de una pasada;
- `StackV0.RunGlobalSupervisorV0` compone supervisor y `RunGlobalTickV0`.

Validacion:

```bash
go test -count=1 ./modulos/orquesta-run-coordinator ./modulos/orquesta-run-supervisor ./modulos/orquesta-app-codex-stack
go test -count=1 ./modulos/orquesta-run-queue ./modulos/orquesta-run-control ./modulos/orquesta-run-memory ./modulos/orquesta-run-coordinator ./modulos/orquesta-run-supervisor ./modulos/orquesta-mcp ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway ./modulos/orquesta-app-director-service ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-codex-stack ./modulos/orquesta-runtime ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-director-scheduler ./modulos/orquesta-director ./modulos/orquesta-web
```

Resultado: OK.

Pendiente para autonomia completa:

- decidir quien invoca pasadas sucesivas: operador MCP opt-in, servicio externo
  o bucle de proceso con presupuesto observable;
- exponer la pasada supervisada por REST/MCP solo cuando haya contrato de
  consumidor claro;
- enlazar stats finales de tokens/cuota y resumen de apps avanzadas.

Siguiente cierre obligatorio antes de llamar al nucleo 100%:

- presupuestos por agente/tarea/ACK;
- estadisticas de edad, ultima actividad, ultimo cambio y causa de espera;
- split/replan automatico de tareas demasiado grandes sin aumentar
  indefinidamente el timeout global.

## Actualizacion 2026-05-11: presupuestos, usage y multitarea

Se cerraron piezas locales que salian de la prueba real anterior:

- `AgentProgressReportV0` transporta metadatos temporales opcionales sin
  cambiar el estado principal `progressing/stalled/loop_detected`;
- `orquesta-runtime-codex-delivery` clasifica actividad como `working`,
  `stalled`, `over_budget_but_active` y `over_budget_no_activity`;
- el scheduler trata `over_budget_but_active` como consulta/revision, no como
  parada;
- `over_budget_no_activity` solo puede escalar a parada en programacion y con
  `StopAllowed=true`;
- `DirectorRunStatsV0` expone esos estados para MCP/web/director;
- `AgentUsageStatsProviderPortV0` expone uso por agente: modelo/alias,
  capacidad, cuota y tokens;
- MCP acepta `include_agent_usage=true` y web proyecta modelo, cuota y tokens;
- el stack Codex publica usage inicial desde configuracion y cuota
  `not_configured` hasta conectar un proveedor real de cuota/tokens;
- los stores in-memory de run y outbox quedaron protegidos con mutex para
  pruebas concurrentes basicas;
- se creo `modulos/orquesta-run-queue` como miniproyecto independiente para
  ranking global multi-app por `priority_score`.

Validacion local:

```bash
go test -count=1 ./modulos/orquesta-run-queue ./modulos/orquesta-app-codex-stack ./modulos/orquesta-mcp ./modulos/orquesta-web ./modulos/orquesta-orchestration-core ./modulos/orquesta-director-scheduler ./modulos/orquesta-director ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-runtime
```

Resultado: OK.

Pendiente para 100% real de nucleo:

- conector real saneado de cuota/tokens por proveedor;
- acumulado final por run: agentes usados, tokens, coste y cuota consumida;
- checkpoint y parada fisica coordinada de agentes para `stop/cancel`;
- runner global multi-app que elija el siguiente `run_ref` por prioridad;
- nuevo smoke real despues de integrar presupuestos en la ejecucion larga.

## Actualizacion 2026-05-11: control de app y prioridad multiapp

Se cerro el contrato local para que humano, director u orquestador puedan
gobernar desarrollos completos sin tocar el nucleo interno:

- `modulos/orquesta-run-control`: estados `running`, `paused`,
  `stop_requested`, `stopped`, `cancel_requested`, `canceled`;
- `modulos/orquesta-run-queue`: ranking por `priority_score`, aging y estado;
- `modulos/orquesta-run-memory`: conector de memoria thread-safe para control y
  cola, sin DB ni runtime;
- MCP/REST:
  - `orquesta.runs.control.v0` con `pause/resume/stop/cancel`;
  - `orquesta.run_queue.priority.v0` con `rank/set_priority`;
- `orquesta-orchestration-core` consulta `RunControlReaderPortV0` antes de planificar
  y antes de despachar;
- el stack expone control/prioridad por API y los cablea solo por puertos
  inyectados.

Decision de base:

- si no existe estado de control para una run, el error tipado
  `RunControlStateNotFoundErrorV0` equivale a `running` por defecto;
- cualquier otro error del conector sigue parando el loop.

Validacion:

```bash
go test -count=1 ./modulos/orquesta-run-queue ./modulos/orquesta-run-control ./modulos/orquesta-run-memory ./modulos/orquesta-mcp ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway ./modulos/orquesta-app-director-service ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-codex-stack ./modulos/orquesta-runtime ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-director-scheduler ./modulos/orquesta-director ./modulos/orquesta-web
```

Resultado: OK.
