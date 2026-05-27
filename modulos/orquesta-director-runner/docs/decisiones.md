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
