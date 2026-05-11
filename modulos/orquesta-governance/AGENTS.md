# Contexto Codex: orquesta-governance

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

Trabaja en reglas, skills, workflows, permisos y catalogos efectivos.

## Reglas

- Toda regla debe tener version y alcance.
- Las reglas deben ser consultables por modulo/agente/fase.
- El catalogo efectivo debe ser compacto.

## Prohibido

- Esconder reglas en UI, DB o scripts.
- Crear reglas contradictorias sin estado documental.
- Mezclar permisos con implementacion de runtime.
