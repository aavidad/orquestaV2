# MODULO CONGELADO 2026-07-03: solo fixes correctivos con bug enlazado. Features nuevas requieren decision documentada en docs/ raiz. Motivo: informe pericial P2 (docs/informe_pericial_claude_orquesta_2026-07-03.md)

# Contexto Codex: orquesta-director-cycle

Lee primero este archivo y `README.md`. Despues lee solo los docs locales necesarios para tu microtarea.

## Reglas comunes

- Contexto pequeno: este modulo coordina un unico paso de orquestacion.
- Hexagonal siempre: scheduler, workflow y outbox ledger son puertos.
- i18n por defecto cuando haya texto visible de UI, CLI, MCP o documentacion generada.
- Funciones pequenas, ficheros pequenos y pruebas de invariantes.
- No cruces `internal/`, structs privados ni detalles internos de otro modulo.
- Si necesitas informacion de otro grupo, emite `CONSULTA AL DIRECTOR`.

## Alcance local

- Listar outbox pendiente del run por ledger inyectado.
- Construir `DirectorSchedulerTickInputV0`.
- Ejecutar `RunDirectorCycleV0`.
- Registrar outbox nueva producida por el runner.
- Devolver resultado compacto del paso.

## Prohibido

- DB, filesystem productivo, red, procesos, runtime real, OAuth, HOME, cuotas o proveedores.
- Construir candidates de trabajo, lease, progreso o replan.
- Despachar outbox o registrar ACK.
- Ejecutar bucles, goroutines, sleeps o daemon.

## Entrega

Cada cambio debe incluir contrato local, prueba local y actualizacion del verificador si entra en nucleo.
