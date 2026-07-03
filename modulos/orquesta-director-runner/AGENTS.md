# MODULO CONGELADO 2026-07-03: solo fixes correctivos con bug enlazado. Features nuevas requieren decision documentada en docs/ raiz. Motivo: informe pericial P2 (docs/informe_pericial_claude_orquesta_2026-07-03.md)

# Contexto Codex: orquesta-director-runner

Lee primero este archivo y `README.md`. Despues lee solo los docs locales necesarios para tu microtarea.

## Reglas comunes

- Contexto pequeno: este modulo es la fuente primaria para ejecutar ciclos del director.
- Hexagonal siempre: scheduler, workflow, persistencia, runtime y outbox se tratan como puertos.
- i18n por defecto cuando haya texto visible de UI, CLI, MCP o documentacion generada.
- Problema grande: descomponer antes de programar.
- Funciones pequenas, ficheros pequenos y pruebas de invariantes por microtarea.
- No cruces `internal/`, structs privados ni detalles internos de otro modulo.
- Si necesitas informacion de otro grupo, emite `CONSULTA AL DIRECTOR`.

## Alcance local

- Ejecutar un unico ciclo del director sobre un tick ya preparado.
- Pedir un plan al scheduler por puerto inyectado.
- Aplicar comandos publicos de workflow por puerto inyectado.
- Detener el ciclo cuando aparezca outbox pendiente, error, bloqueo o consulta al director.

## Prohibido

- Daemon, goroutines, sleeps, reloj interno o bucles infinitos.
- DB, filesystem productivo, red, procesos, runtime real, OAuth, HOME, cuotas o proveedores.
- Construir candidatos de scheduler dentro de este modulo.
- Despachar outbox directamente.
- Decidir modelos, despliegue, persistencia real o adaptadores concretos.

## Entrega

Cada cambio debe incluir write-set pequeno, contrato afectado, prueba local y actualizacion de docs locales si cambia el estado.
