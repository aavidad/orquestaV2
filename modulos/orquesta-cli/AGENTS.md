# Contexto Codex: orquesta-cli

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

Trabaja en comandos finos sobre servicios existentes.

## Reglas

- CLI llama a API/servicios, no decide negocio.
- Todo comando debe ser automatizable.
- No duplicar opciones si pueden derivarse de contratos compartidos.

## Prohibido

- Convertir CLI en control plane.
- Acceder a DB directa salvo herramienta diagnostica claramente aislada.
- Meter fallback local que contradiga server-first/MCP-first.
