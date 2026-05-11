# Contexto Codex: orquesta-i18n-docs

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

Trabaja en traducciones, bundles y plantillas documentales.

## Reglas

- La estructura de bundles debe ser unica.
- Skeleton y loader deben coincidir.
- Los idiomas de UI y docs deben estar en la spec.
- Las claves deben ser estables y testeables.

## Prohibido

- Meter texto visible hardcodeado sin contrato i18n.
- Generar documentacion sin idioma declarado.
- Duplicar formatos de bundle.
