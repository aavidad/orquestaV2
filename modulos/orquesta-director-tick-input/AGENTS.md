# MODULO CONGELADO 2026-07-03: solo fixes correctivos con bug enlazado. Features nuevas requieren decision documentada en docs/ raiz. Motivo: informe pericial P2 (docs/informe_pericial_claude_orquesta_2026-07-03.md)

# Contexto Codex: orquesta-director-tick-input

Lee primero este archivo y `README.md`. Despues lee solo los docs locales necesarios para tu microtarea.

## Reglas comunes

- Contexto pequeno: este modulo prepara inputs compactos para scheduler/runner.
- Hexagonal siempre: estado durable, outbox y candidates llegan como datos o puertos externos.
- i18n por defecto cuando haya texto visible de UI, CLI, MCP o documentacion generada.
- Funciones pequenas, ficheros pequenos y pruebas de invariantes.
- No cruces `internal/`, structs privados ni detalles internos de otro modulo.
- Si necesitas informacion de otro grupo, emite `CONSULTA AL DIRECTOR`.

## Alcance local

- Construir `DirectorSchedulerTickInputV0` desde `OrchestrationRunV0`.
- Traducir proyecciones compactas del workflow a refs que entiende el scheduler.
- Adjuntar candidates explicitos ya calculados por otros modulos.
- Adjuntar refs de outbox pendiente recibidas desde fuera.

## Prohibido

- DB, filesystem productivo, red, procesos, runtime real, OAuth, HOME, cuotas o proveedores.
- Calcular candidates de trabajo, lease, progreso o replan.
- Elegir modelo/capacidad real.
- Aplicar comandos o despachar outbox.

## Entrega

Cada cambio debe incluir contrato local, prueba local y actualizacion del verificador si entra en nucleo.
