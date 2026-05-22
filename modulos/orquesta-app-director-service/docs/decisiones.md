# Decisiones: orquesta-app-director-service

```text
Fecha: 2026-05-17
Decision: Materializar el primer tramo del Director Operativo desde
ContinueAppDirectorV0, no desde StartAppDirectorV0.
Motivo: el arranque normal abre el run y el equipo/director inicial, pero un
plan operativo listo necesita un run ya compatible, contratos de funcion
publicados y stores inyectados. Meterlo en Start mezclaria bootstrap con
programacion y repetiria el problema de crear microtareas antes de que el
director tenga estado causal.
Impacto: ContinueAppDirectorV0 acepta `OperationalDirectorPlanV0`, llama al
materializador, deriva wait por ola/cohorte, registra `WorkflowTaskWaitStateV0`
si hay writer y reentra al loop progresivo con refs acotadas. El siguiente
corte llevo la salida positiva de ese wait a review/tests/cierre offline; queda
smoke real de servidor con runner opt-in.
Estado: aceptada.
```

```text
Fecha: 2026-05-22
Decision: Tratar como quiescent scoped un `wait_external` global cuando el
scope operativo ya no tiene agentes pendientes.
Motivo: el scheduler global puede seguir viendo agentes vivos ajenos a la ola
operativa, pero el Director Operativo debe avanzar la ola que ya entrego sin
esperar todo el run.
Impacto: `ContinueAppDirectorV0` conserva el loop global, pero para actualizar
`OperationalDirectorPlanStateV0` y cerrar por fuente causal usa estado
quiescent solo si `WaitAgentRefs` resueltos estan entregados/fallidos/perdidos o
parados y no hay outbox pendiente. El cierre sigue rechazando tareas abiertas y
evidencias causales incompletas.
Estado: aceptada.
```

```text
Fecha: 2026-05-17
Decision: Guardar el estado vivo inicial del plan operativo como
OperationalDirectorPlanStateV0.
Motivo: el plan `OperationalDirectorPlanV0` declara intencion, pero no conserva
el avance observado. Despues de materializar la primera ola, una reentrada debe
saber que `launch_subagents` quedo aceptado, que `wait_subagents` esta activo y
que refs concretas pertenecen al scope.
Impacto: `StartAppDirectorPortsV0` acepta `OperationalPlanStateWriter` y
`OperationalPlanStateStore`; `ContinueAppDirectorV0` guarda
`OperationalDirectorPlanStateV0` con ola/cohorte, tasks, agentes, pendientes y
`wait_ref`, y puede leerlo para reentrada inicial por
`operational_director_plan_ref`. El corte restante debe actualizarlo al consumir
review/tests/replan/cierre.
Estado: aceptada.
```

```text
Fecha: 2026-05-17
Decision: Guardar la resolucion de wait del Director como
WorkflowTaskWaitStateV0.
Motivo: devolver solo `wait_external` no explica por que se espera ni que
agentes/tareas siguen pendientes. Los futuros agentes necesitan una foto
durable del bloqueo sin obligar al stack Codex a conocer cohortes u olas.
Impacto: `StartAppDirectorPortsV0` acepta `WaitStateWriter`; cuando un filtro
de espera se resuelve desde workflow tasks, el servicio guarda causa, scope,
task refs, agent refs y pending agent refs. La persistencia concreta sigue
siendo un adaptador.
Estado: aceptada.
```

```text
Fecha: 2026-05-17
Decision: Resolver esperas por cohorte, ola o parent task en el servicio del
director antes de entrar al loop progresivo.
Motivo: el stack Codex ya sabe esperar `WaitAgentRefs`, pero no debe conocer la
semantica operativa de `cohort_ref`, `wave_ref` ni linaje de microtareas. Esa
semantica pertenece al plano de aplicacion del director, que tiene acceso al
run y al `DirectorTaskStore`.
Impacto: `StartAppDirectorRequestV0` y `ContinueAppDirectorRequestV0` aceptan
`wait_cohort_ref`, `wait_wave_ref` y `wait_parent_task_ref`. El servicio carga
las `WorkflowTaskV0` autorizadas por el run, deriva refs de agente con
`WorkflowTaskWaitAgentRefsV0`, conserva refs explicitas y falla de forma
publica si se pide filtro sin `DirectorTaskStore`.
Estado: aceptada.
```

```text
Fecha: 2026-05-12
Decision: Una decision del director pendiente corta el lote actual.
Motivo: un director real emitio una cadena con `accept_decision.vote_ref`
incoherente. El servicio saltaba esa transicion pendiente y aplicaba decisiones
posteriores, llegando a abrir `programacion` sin microtareas. Eso recreaba el
problema historico de v1/v2: avanzar por apariencia de progreso aunque falte un
prerrequisito causal.
Impacto: si `ApplyDirectorAgentDecisionV0` devuelve `ErrTransicionInvalidaV0`,
el servicio conserva el progreso ya reflejado y detiene el consumo del lote.
Las decisiones posteriores se reintentan en otro ciclo solo cuando la decision
pendiente ya pueda aplicarse. Ademas `open_phase(programacion)` queda diferido
si no hay microtareas materializadas.
Estado: aceptada.
```

```text
Fecha: 2026-05-11
Decision: El servicio compone tambien el review gate como fuente externa
inyectada.
Motivo: REST, MCP y web no deben saber cuando hay que leer ACKs, evidencias de
ficheros o resultados de revision. El servicio ya compone delivery, progreso,
decisiones y replan; dejar review gate fuera obligaria a cada transporte a
conocer el scheduler.
Impacto: `StartAppDirectorPortsV0.ReviewGateSource` entra en el provider de
candidatos. `ContinueAppDirectorV0` puede registrar revision, resultado y
rework usando solo puertos inyectados, sin DB, filesystem ni runtime
hardcodeados.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: Separar servicio de app de los adaptadores REST/MCP/web.
Motivo: los adaptadores deben ser finos y no conocer scheduler, outbox ni
dispatchers.
Impacto: REST y MCP podran compartir este servicio sin duplicar logica.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: Bloquear cierre productivo si la politica de solicitud no tiene
evidencia minima.
Motivo: una app completa no puede declararse terminada si el director solo
genero documentos o artefactos parciales. Ese fue uno de los riesgos de v1/v2:
confundir avance aparente con entregable ejecutable.
Impacto: antes de aplicar decisiones `register_final_validation` o `close_run`,
el servicio resuelve `request_kind` y `execution_mode`. En modo `debug` permite
alcance reducido; en modo normal exige evidencias coherentes, por ejemplo
contratos, microtareas, entregas, tareas cerradas y revision aceptada para
`crear_app_completa`. La regla vive fuera del core para mantener hexagonalidad.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: Arrancar un director no materializa microtareas de producto.
Motivo: el servicio solo debe preparar el run, pedir brainstorming y lanzar
agentes por puertos. Crear tareas de producto antes de una decision del director
mezclaria bootstrap con planificacion y repetiria acoplamientos de v1/v2.
Impacto: `StartAppDirectorV0` puede devolver `started_agents` y `director_task`
con `Run.Tasks` vacio. La siguiente fase productiva crea microtareas mediante
decisiones explicitas del director y `DirectorTaskStore`.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: Permitir muchas esperas externas cortas por defecto.
Motivo: los agentes reales pueden tardar minutos. El director debe observar
progreso entre ciclos, pero no debe cortar una ola productiva por un presupuesto
por defecto de pocos segundos.
Impacto: `max_external_waits` por defecto pasa a 120. Los adaptadores REST/MCP
pueden enviar otro limite; cada conector de espera debe preferir esperas cortas
para dejar que el loop observe entregas, progreso y leases entre ciclos.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: No reejecutar decisiones del director ya reflejadas en el run.
Motivo: las decisiones reales viven en artefactos persistentes como
`director_decisions.json`; leer el mismo archivo en ciclos posteriores no debe
reabrir fases ni recrear microtareas ya aplicadas.
Impacto: el servicio consulta los `command_effects` del run antes de aplicar una
decision. Si el `idempotency_key` y `command_ref` ya existen, la decision se
omite sin error y el director puede seguir siendo la unica pieza que decide.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: Reentrar al loop tras aplicar decisiones del director.
Motivo: si el director abre fases, publica contratos o crea microtareas, Orquesta debe continuar sola hasta el siguiente estado estable.
Impacto: `StartAppDirectorV0` ejecuta ciclos acotados por `max_decision_cycles`; solo reentra cuando las decisiones generan eventos nuevos.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: Usar `DirectorTaskStore` como puerto de escritura y lectura de microtareas.
Motivo: crear microtareas sin que el scheduler pueda leerlas dejaba la programacion bloqueada.
Impacto: el mismo conector puede materializar la microtarea y despues alimentar `WorkflowTaskCandidateProviderV0`.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: Consumir decisiones del director desde un puerto opcional del servicio.
Motivo: Orquesta no debe depender de que la sesion principal invoque manualmente el tool tras arrancar directores reales.
Impacto: `StartAppDirectorV0` puede aplicar decisiones compactas mediante `orquesta-director-agent-workflow`; si la decision crea microtareas exige `DirectorTaskStore` inyectado.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: Requerir puertos inyectados para ejecutar el loop.
Motivo: crear DB/runtime/fake por defecto repetiria el error de v1/v2 y
ocultaria la arquitectura real.
Impacto: la composicion productiva decide conectores; el servicio solo orquesta.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: Componer observadores externos dentro del servicio.
Motivo: REST/MCP/web no deben cablear manualmente entregas, progreso, leases o
replans. Pasan puertos y Orquesta monta el provider del loop.
Impacto: el servicio puede arrancar director y consumir artefactos/progreso sin
conocer runtime, DB, modelo, HOME ni credenciales.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: Director arrancado con agentes vivos es `started` aunque el loop
quede en `wait_external`.
Motivo: esperar el artefacto de un agente real no es un estado pendiente de
arranque; es la situacion normal tras lanzar un director.
Impacto: la API/MCP puede devolver started y mostrar que espera entrega externa
sin declarar quiescent falsamente.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: Usar loop gestionado del nucleo cuando se inyecta `ExternalWaiter`.
Motivo: los directores reales pueden publicar decisiones despues del primer
`wait_external`; consumir la fuente inmediatamente deja decisiones sin leer.
Impacto: `StartAppDirectorV0` espera progreso externo de forma acotada,
consume decisiones disponibles y reentra al loop gestionado para arrancar
agentes derivados sin conocer runtime, DB ni rutas.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: Separar continuar un run existente de arrancar una app nueva.
Motivo: las decisiones ejecutables de agentes reales pueden aparecer despues
del primer arranque. Reusar `StartAppDirectorV0` para continuar obligaria a
recrear AppSpec/intake y mezclaria arranque con drenaje.
Impacto: `ContinueAppDirectorV0` reentra en el loop para un `run_ref` existente
con los mismos puertos hexagonales: delivery, progreso, decisiones, task store,
outbox y dispatchers. No crea runtime, DB, proveedor ni intake nuevo.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: Componer el replan de retrabajo de revision por puerto opcional.
Motivo: el servicio debe permitir que Orquesta continue tras `RequestRework`
sin que REST/MCP/web conozcan scheduler ni origen del plan. El core tampoco
debe saber si el plan viene de director, IA externa, regla local o conector.
Impacto: `StartAppDirectorPortsV0` agrega `ReviewReworkReplanSource`; si se
inyecta, el provider del nucleo convierte planes compactos en
`ReplanFollowupCandidates`. Si no se inyecta, el comportamiento no cambia.
Estado: aceptada.
```
