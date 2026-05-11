# Contexto Codex: orquesta-director-supervised-burst

Lee primero este archivo y `README.md`. Despues lee solo los docs locales necesarios para tu microtarea.

## Reglas comunes

- Contexto pequeno: este modulo ejecuta una rafaga acotada de pasos del director.
- Hexagonal siempre: input de paso, ejecucion de paso y politica de supervision son puertos.
- Funciones pequenas, ficheros pequenos y pruebas de invariantes.
- No cruces `internal/`, structs privados ni detalles internos de otro modulo.
- Si necesitas informacion de otro grupo, emite una `CONSULTA AL DIRECTOR`.

## Alcance local

- Pedir `DirectorCycleStepInputV0` a un puerto por cada paso.
- Ejecutar un unico paso por iteracion mediante puerto.
- Consultar `DecideDirectorSupervisorNextActionV0` mediante puerto.
- Cortar al primer estado no continuable o al presupuesto `max_steps`.

## Prohibido

- Daemon, goroutines, sleeps, timers, polling o espera activa.
- DB, filesystem productivo, red, procesos, runtime real, OAuth, HOME, cuotas o proveedores.
- Despachar outbox, registrar ACK, generar candidates o leer event-store.
- Mutar scheduler, workflow o supervisor para encajar este modulo.

## Entrega

Cada cambio debe incluir contrato local, prueba local y actualizacion del verificador si entra en nucleo.
