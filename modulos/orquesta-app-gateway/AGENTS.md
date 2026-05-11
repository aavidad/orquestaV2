# Contexto Codex: orquesta-app-gateway

## Reglas comunes

- Contexto pequeno: usa este modulo y contratos citados por la tarea.
- Hexagonal: esta raiz compone puertos y handlers, no implementa negocio.
- i18n por defecto cuando haya texto visible; reutiliza catalogos de web.
- Persistencia, runtime, LLM, filesystem, red real y deploy son adaptadores.
- Problema grande: descomposicion primero; microtarea pequena despues.
- No cruces `internal/`, tablas, structs privados ni detalles de proveedor.
- Si necesitas informacion o decision de otro grupo, emite `CONSULTA AL DIRECTOR`.
- Al arrancar aqui, lee `README.md` y los docs locales necesarios en `docs/`.

## Alcance

Composition root HTTP para montar web + API REST + MCP bridges por puertos
inyectados, sin pasar por `cmd`.

## Reglas

- Puede importar web, MCP, factory y el gateway HTTP fino.
- No abre sockets ni llama `ListenAndServe`.
- No crea DB, runtime, procesos, proveedor, HOME, OAuth ni modelos.
- Los handlers API se construyen antes que la web; la web consume esas APIs por
  cliente REST in-process o cliente inyectado.

## Prohibido

- Importar `cmd`, `db`, drivers SQL, runtime real, Codex real, agentcli real o
  paquetes de proveedor.
- Parsear flags/env o leer configuracion global.
- Meter logica de scheduler, director, factory o UI dentro de este modulo.
