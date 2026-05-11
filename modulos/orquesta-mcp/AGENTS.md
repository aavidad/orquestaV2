# Contexto Codex: orquesta-mcp

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

Expone capacidades de Orquesta por MCP sin meter logica de negocio en el adaptador.

## Reglas

- Cada tool debe llamar a un caso de uso.
- Cada resource debe ser compacto y estable.
- El E2E por MCP es criterio de cierre del nucleo.
- Las respuestas deben ser utiles para IA, no dumps de DB.

## Prohibido

- Ampliar un fichero MCP gigante.
- Saltar servicios para escribir en DB.
- Duplicar reglas que ya pertenecen al core.
