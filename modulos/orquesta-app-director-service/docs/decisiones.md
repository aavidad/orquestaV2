# Decisiones: orquesta-app-director-service

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
Decision: Permitir varias esperas externas cortas por defecto.
Motivo: una sola espera larga oculta progreso, bloquea supervision y convierte
un agente sin ACK en `context deadline exceeded` en vez de en un estado
gobernable por Orquesta/director.
Impacto: `max_external_waits` por defecto pasa a 6. Los adaptadores REST/MCP
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
