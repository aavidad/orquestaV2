# Contexto Codex: orquesta-core

Lee primero este archivo y `README.md` de este modulo. Usa documentos globales solo si la tarea lo pide.

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

Trabaja en reglas de negocio puras: fases, tareas, contratos, propuestas, votos, decisiones y cierre por evidencia.

## Prohibido

- Importar adaptadores concretos.
- Depender de DB, HTTP, CLI, MCP o tmux.
- Meter nombres de proveedores en entidades de dominio.
- Crear funciones largas que coordinen todo el sistema.

## Entrega

Cada cambio debe tener contrato pequeno y prueba o ejemplo verificable.
