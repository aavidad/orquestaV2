# Contexto Codex: orquesta-observability

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

Trabaja en eventos, artifacts, traces y proyecciones compactas.

## Reglas

- Observa, no decide.
- Los eventos deben ser normalizados y enlazables.
- No volcar transcripts completos como contexto normal.
- Las proyecciones deben servir para MCP y web.

## Prohibido

- Mezclar auditoria con decisiones de negocio.
- Crear logs imposibles de filtrar.
- Usar observabilidad como persistencia paralela.
