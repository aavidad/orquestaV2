# Decisiones: orquesta-director-runner

## DCR-DEC-001: Runner Como Ciclo, No Daemon

Decision: el runner ejecuta un unico ciclo y termina.

Motivo: evita bucles ocultos, facilita pruebas progresivas y permite que MCP, CLI, web o un futuro supervisor automatico decidan cuando repetir.

Consecuencia: no hay goroutines, sleeps ni reloj interno.

## DCR-DEC-002: Scheduler Y Workflow Son Puertos

Decision: el runner no llama a adaptadores concretos.

Motivo: mantiene el nucleo hexagonal y permite cambiar persistencia, event-store, scheduler remoto o workflow sin reescribir el ciclo.

Consecuencia: las pruebas pueden usar memoria, y produccion podra inyectar conectores reales.

## DCR-DEC-003: Parar En Outbox

Decision: cuando un comando produce outbox, el runner devuelve `outbox_pending` y no aplica mas comandos salvo que `max_outbox` se haya configurado explicitamente por encima de 1.

Motivo: persistir/despachar outbox es otra responsabilidad. Parar evita duplicados y permite que el ledger externo sea el punto de consistencia.

Consecuencia: el siguiente ciclo debe arrancar con snapshot actualizado y refs de outbox pendiente/resuelta.

## DCR-DEC-006: Acumulacion Explicita De Outbox

Decision: `max_outbox` permite acumular varios outbox en un unico ciclo, manteniendo valor por defecto 1.

Motivo: la orquestacion multiagente necesita poder generar varios `LaunchRuntimeAgent` antes de llamar a un dispatcher batch externo. Hacerlo opt-in conserva el comportamiento seguro del runner y evita volver a meter ejecucion de runtime dentro del ciclo.

Consecuencia: el runner sigue sin despachar outbox, sin goroutines y sin proveedor. El ensamblador superior decide cuando activar `max_outbox>1` y que adaptador batch ejecuta los intents.

## DCR-DEC-004: Sin Construccion De Contexto

Decision: el runner consume un tick preparado; no clasifica contexto ni candidatos.

Motivo: preparar contexto compacto para agentes es otro microproyecto. Mezclarlo aqui agrandaria el modulo y dificultaria debug.

Consecuencia: se necesitara un ensamblador superior que lea estado compacto y genere `DirectorSchedulerTickInputV0`.

## DCR-DEC-005: Comandos Durables Antes De Needs Director

Decision: si el scheduler devuelve `needs_director` o `blocked` con comandos, el runner los aplica antes de devolver ese estado final.

Motivo: el scheduler puede necesitar registrar un hecho durable previo, por ejemplo un lease expirado, antes de pedir una decision. Si el runner ignorase esos comandos, el siguiente tick repetiria el mismo plan y crearia bucle.

Consecuencia: `waiting` y `quiescent` siguen sin efectos; `needs_director` y `blocked` pueden tener efectos durables acotados, siempre por puerto workflow y parando si aparece outbox.

## DCR-DEC-007: Progreso Ajeno No Bloquea El Runner

Decision: el runner descarta candidatos de progreso cuyo `command_meta.run_id` o
`report.run_id` no coincida con el `run_ref` del ciclo.

Motivo: el scheduler conserva contrato estricto, pero el runner es frontera de
integracion y puede recibir estado durable antiguo de otros runs.

Consecuencia: no se construyen candidatos nuevos ni se corrige contenido; solo
se evita que progreso ajeno bloquee el ciclo del Director.

## DCR-DEC-008: T207 No Introduce Rotacion En El Runner

Decision: la rotacion experimental de sesiones no modifica
`RunDirectorCycleV0`.

Motivo: T207 requiere handoff durable y relevo opt-in, pero el runner no posee
sesiones, procesos, runtime ni politica de relanzamiento.

Consecuencia: el runner sigue parando por outbox/bloqueo/error; la directiva de
relevo se construye fuera y reingresa como trabajo normal del ciclo.

## DCR-DEC-010: Workflow Persistido Por Event-Store Externo

Decision: `StoredWorkflowCommandPortV0` aplica comandos de `core-workflow`
reconstruyendo el run desde eventos durables y anexando nuevos eventos mediante
`WorkflowEventStorePortV0`.

Motivo: el runner necesitaba una implementacion neutral de su puerto workflow
sin elegir DB ni guardar proyecciones propias. El event-store es una frontera
hexagonal: archivo, SQL o cualquier store real quedan en composiciones externas.

Consecuencia: `RunDirectorCycleV0` no cambia de responsabilidad. Quien use el
adaptador debe inyectar un store que devuelva eventos ordenados por run y acepte
append idempotente/causal. El runner sigue sin filesystem productivo, runtime,
daemon ni despacho de outbox.

## DCR-DEC-011: Presupuesto De Comandos No Es Rail De Rechazo

Decision: si el scheduler propone mas comandos que `max_commands`, el runner no
rechaza el plan completo; valida y aplica solo el prefijo dentro del presupuesto.

Motivo: el presupuesto es una compuerta de avance reentrable, no una regla de
strings exactos. Un plan razonable puede traer mas trabajo listo del que cabe en
un tick y el Director debe poder continuar en el siguiente ciclo.

Consecuencia: comandos fuera del presupuesto no tienen efectos en ese tick. El
estado queda `commands_applied` con `commands_exhausted`, salvo que el prefijo
aplicado genere outbox, bloqueo o necesidad de director.

## DCR-DEC-012: DCR-009 Es Ciclo Superior, No Runner

Decision: el ciclo progresivo multi-fase con app de prueba no se implementa en
`orquesta-director-runner`.

Motivo: requiere repeticion, snapshots actualizados y posible composicion/smoke
real. Esas responsabilidades viven por encima del tick acotado y ya empiezan en
`orquesta-director-cycle` con `ExecuteDirectorCycleStepsV0`.

Consecuencia: el runner mantiene una sola pasada sin daemon ni runtime real. La
evidencia local valida que recibe un tick preparado, aplica comandos por puerto
y corta en outbox/espera/bloqueo; los smokes reales deben documentarse en la
composicion que inyecte snapshots, event-store y dispatcher opt-in.

## DCR-DEC-009: DCR-007 Vive Fuera Del Runner

Decision: el conector superior que prepara candidates/snapshot desde contratos
compactos no se implementa en `orquesta-director-runner`.

Motivo: el runner es la frontera que ejecuta un ciclo con
`DirectorSchedulerTickInputV0` ya construido. Si leyera estado durable,
clasificara candidates o reconstruyera snapshots, mezclaria tick-input,
scheduler y persistencia en un modulo que debe seguir siendo hexagonal y
acotado.

Consecuencia: `orquesta-director-tick-input` y `orquesta-director-cycle`
pueden componer el tick desde contratos compactos; el runner solo valida,
filtra progreso ajeno por causalidad de `run_ref`, pide plan al scheduler y
aplica comandos al workflow por puerto.

## DCR-DEC-011: DCR-009 Vive En Composicion Externa

Decision: el ciclo progresivo multi-fase con orquestacion real de una app de
prueba no se implementa en `orquesta-director-runner`.

Motivo: el runner ejecuta un tick acotado. Una app real implica preparar
snapshots/candidates entre steps, supervisar presupuesto, tratar outbox y
posiblemente runtime; esas responsabilidades pertenecen a composiciones sobre
`orquesta-director-cycle`, `orquesta-director-supervised-burst` u
`orquesta-orchestration-core`.

Consecuencia: `DCR-009` queda reconciliado como tarea fuera del runner. El
runner conserva su contrato: scheduler por puerto, workflow/event-store por
puerto, sin daemon, sin runtime real y sin despacho de outbox.
