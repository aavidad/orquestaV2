# Contexto Codex: orquesta-http-gateway

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

Compone rutas HTTP estables a partir de handlers `net/http` inyectados.

## Reglas

- El gateway no conoce internals de web, MCP, factory, runtime, DB, cmd ni Codex.
- El gateway registra solo handlers ya construidos por el borde de composicion.
- Las rutas exportadas son contrato publico de entrada HTTP.
- Las rutas sin handler inyectado deben quedar sin registrar.

## Prohibido

- Importar otros modulos productivos de Orquesta.
- Crear handlers con logica de negocio dentro del gateway.
- Resolver DB, runtime, proveedor, CLI legacy, filesystem o configuracion global.
