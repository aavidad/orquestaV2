# Decisiones: orquesta-director-supervisor

## DSV-001: politica antes que motor

Este modulo solo decide la siguiente accion. No ejecuta un bucle multi-step.

Motivo:

- Los motores durables probados separan decision reproducible y efectos externos.
- Meter un loop operacional dentro del nucleo reintroduce el problema de v1/v2: comportamiento dificil de reproducir y bugs en cascada.

## DSV-002: presupuesto explicito

`max_steps` es obligatorio. No hay valor implicito escondido en el codigo.

Motivo:

- El caller debe declarar cuanto trabajo permite en una rafaga.
- Una rafaga sin presupuesto es el origen natural de bucles y consumo de cuota.

## DSV-003: espera y efectos externos fuera

`wait_outbox` y `wait_external` no duermen ni hacen polling.

Motivo:

- El supervisor externo, MCP, REST, CLI o un workflow engine futuro decide cuando volver a llamar.
- El nucleo queda portable a Temporal, Conductor, Durable Task, local worker u otro conector.

## DSV-004: recomendacion autonoma explicita

La salida incluye `autonomous_recommendation` para separar la accion compacta del
consejo de bucle externo: continuar, esperar, pedir director o parar.

Motivo:

- Un adaptador autonomo no debe inferir manualmente que `wait_outbox` o
  candidatos pendientes equivalen a esperar.
- `candidate_missing` bajo `needs_director` puede representar una fuente externa
  de candidatos aun no lista, por lo que se conserva como espera sin efectos.
