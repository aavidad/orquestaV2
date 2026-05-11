# Contexto local: orquesta-decision-council

## Reglas comunes

- Contexto pequeno: lee este modulo y solo contratos globales citados por la tarea.
- Hexagonal: conecta por puertos, DTOs/eventos y conectores.
- i18n por defecto cuando haya texto de UI, app o documentacion generada.
- Persistencia, runtime, LLM, filesystem, MCP y deploy son adaptadores.
- Problema grande: descomposicion primero; microtarea pequena despues.
- No cruces `internal/`, tablas, structs privados ni detalles de proveedor de otro modulo.
- Si necesitas informacion o decision de otro grupo, emite `CONSULTA AL DIRECTOR`.
- Al arrancar aqui, lee `README.md` y los docs locales necesarios en `docs/`.

## Alcance

Trabaja solo en la politica pura del consejo de decision: propuesta, critica, voto, quorum y consenso.

## Prohibido

- Lanzar procesos o agentes reales.
- Elegir modelos, proveedores o runtimes concretos por codigo fijo.
- Guardar HOME, OAuth, tokens, rutas reales, prompts completos o transcripts.
- Convertir votacion en decision aceptada sin pasar por `orquesta-core-workflow`.
