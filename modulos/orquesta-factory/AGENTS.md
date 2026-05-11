# Contexto Codex: orquesta-factory

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

Trabaja en la especificacion de nueva app y en la generacion de backlog inicial.

## Reglas

- Hexagonal por defecto salvo excepcion explicita.
- i18n por defecto salvo que no tenga sentido.
- DB como conector.
- La spec debe ser versionada y validable.
- El backlog debe dividirse en microtareas cerradas.

## Prohibido

- Crear tareas directas en DB desde UI.
- Mezclar wizard con runtime.
- Generar proyectos sin contratos verificables.
