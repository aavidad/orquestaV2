# Decisiones: orquesta-director-supervised-burst

## DSB-001: rafaga acotada, no daemon

Este modulo puede ejecutar varios pasos seguidos, pero solo dentro de un `max_steps` explicito.

Motivo:

- permite probar automatizacion real sin introducir un worker residente;
- evita bucles infinitos por contrato;
- mantiene timers, polling, dispatch y persistencia fuera del nucleo.

## DSB-002: input fresco por puerto

Cada paso pide `DirectorCycleStepInputV0` a un builder externo.

Motivo:

- el modulo no debe saber leer event-store, DB ni snapshots internos;
- el caller decide como obtener el estado actualizado del run;
- evita repetir comandos con un snapshot viejo.

## DSB-003: supervisor manda

La rafaga solo repite cuando `DecideDirectorSupervisorNextActionV0` devuelve `continue`.

Motivo:

- outbox, espera externa, bloqueo, pregunta al director y error son cortes reales;
- el modulo no interpreta efectos externos ni simula ACKs.
