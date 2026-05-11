# Contexto Codex: orquesta-runtime

## Reglas comunes

- Contexto pequeno: usa este modulo y solo contratos globales citados por la tarea.
- Hexagonal: conecta por puertos, DTOs/eventos y conectores.
- i18n por defecto cuando haya texto de UI, app o documentacion generada.
- Persistencia, runtime, LLM, filesystem y deploy son adaptadores.
- Problema grande: descomposicion primero; microtarea pequena despues.
- No cruces `internal/`, tablas, structs privados ni detalles de proveedor de otro modulo.
- Si necesitas informacion o decision de otro grupo, emite `CONSULTA AL DIRECTOR`.
- Al arrancar aqui, lee `README.md` y los docs locales necesarios en `docs/`.

## Alcance

Implementa conectores de agentes y gestion de runtime sin decidir fases ni asignaciones de negocio.

## Reglas

- Runtime transport no es fuente de verdad.
- Mailbox, ACK y readiness deben ser evidencias persistibles.
- Multi-HOME debe separar identidad logica, home, credencial, proveedor y cuota.
- Codex es un conector, no una dependencia del core.

## Prohibido

- Meter politica de asignacion.
- Escribir directo en UI o CLI.
- Mezclar proveedor concreto con dominio.
