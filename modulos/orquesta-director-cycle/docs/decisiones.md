# Decisiones: orquesta-director-cycle

## DCS-DEC-001: Paso Unico, No Bucle

Decision: el modulo ejecuta un solo paso y termina.

Motivo: repetir pasos debe depender de presupuesto, cuotas, supervision y control externo. Meter loops aqui volveria dificil depurar bloqueos.

Consecuencia: un supervisor MCP, CLI, web o agente director podra invocarlo varias veces con contexto compacto.

## DCS-DEC-002: Candidates Explicitos

Decision: el coordinador no crea candidates.

Motivo: los candidates son decisiones de politicas vecinas y deben estar probadas por separado.

Consecuencia: si no hay candidates ni outbox pendiente, el paso acaba en `quiescent`.

## DCS-DEC-003: Outbox Como Barrera

Decision: se lista outbox pendiente antes del tick y se registra outbox nueva despues del runner.

Motivo: evita duplicar comandos cuando hay efectos externos pendientes.

Consecuencia: el siguiente paso recibira `pending_outbox_refs` y el scheduler esperara.

## DCS-DEC-004: WorkClaims Pasan Como Contexto Compacto

Decision: el paso de ciclo acepta `work_claims` compartidos y los pasa al tick-input.

Motivo: mantener la ola de concurrencia fuera de cada candidate evita payloads grandes y conserva el scheduler como autoridad de conflictos.

Consecuencia: el modulo sigue sin crear candidates ni claims; solo coordina puertos.

## DCS-DEC-005: Recovery No Inferido

Decision: el paso de ciclo no infiere perdida de outbox aunque el run tenga
eventos durables y el ledger aparezca vacio.

Motivo: un ledger vacio puede significar perdida, dispatcher externo ya
procesado o filtro de persistencia distinto. La decision de reconstruir outbox
debe venir en candidates explicitos y seguir pasando por scheduler y
core-workflow.

Consecuencia: `ExecuteDirectorCycleStepV0` pasa los candidates de recovery sin
interpretarlos; si el workflow devuelve outbox idempotente, la registra como
pendiente igual que cualquier outbox nueva.

## DCS-DEC-006: Integracion Programacion Por Candidates Externos

Decision: la integracion de programacion se prueba inyectando `WorkCandidates`
y `WorkClaims` ya construidos, no generandolos dentro del ciclo.

Motivo: el modulo solo coordina tick-input, scheduler, runner, workflow y
ledger. La construccion de candidates pertenece a politicas vecinas; aqui se
verifica que el paso conserva esa entrada, aplica comandos reales del core y
registra la outbox resultante.

Consecuencia: `DCS-005` queda cubierto sin abrir loops, runtime real ni
producto. `DCS-004` debe entrar como API separada si se implementa coordinacion
multi-step con presupuesto externo.

## DCS-DEC-007: Multi-Step Presupuestado Separado

Decision: `ExecuteDirectorCycleStepsV0` coordina varios steps solo como API
separada, con `max_steps` explicito y snapshot actualizado por puerto externo.

Motivo: el step atomico debe seguir siendo una llamada determinista. El
coordinador superior puede consumir presupuesto y refrescar el run sin meter
daemon, timers, runtime ni generacion de candidates en este modulo.

Consecuencia: el coordinador corta en outbox, waiting, blocked, needs_director,
quiescent, error o `stop_max_steps`. Si necesita continuar y no hay
`snapshot_port`, devuelve `director_cycle_steps_snapshot` con `stop_error` en
vez de repetir contra estado stale.
