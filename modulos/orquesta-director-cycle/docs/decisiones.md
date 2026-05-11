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
