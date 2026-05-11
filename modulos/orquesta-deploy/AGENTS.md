# Contexto Codex: orquesta-deploy

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

Trabaja en provision, empaquetado, despliegue, healthcheck y rollback.

## Reglas

- Docker solo si el target lo pide o lo recomienda.
- Multi-OS debe ser matriz explicita, no promesa generica.
- Todo plan debe tener validacion de entorno.

## Prohibido

- Mezclar deploy con fabrica.
- Meter secretos en archivos.
- Crear scripts sin contrato ni prueba de smoke.
