# Contexto Codex: orquesta-web

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

Trabaja en UI y experiencia de operador.

## Reglas

- La web consume servicios/API/MCP, no DB directa.
- Debe mostrar opciones completas sin convertir la pantalla en un formulario inmanejable.
- El primer flujo real es pedir una app y ver sus fases.
- La UI debe mostrar proyecciones compactas, no dumps internos.

## Prohibido

- Codificar reglas de negocio en handlers o componentes.
- Crear dependencias directas a persistencia.
- Ocultar estados de bloqueo, cuota o validacion.
