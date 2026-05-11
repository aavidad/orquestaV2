# Contexto Codex: orquesta-persistence

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

Trabaja en persistencia, unidades de trabajo, repositorios y conectores abstractos.

## Reglas

- La persistencia concreta es adaptador, no core.
- Ningun motor de base de datos es prioridad canonica en este modulo.
- Los motores concretos se elegiran por conectores versionados y configuracion externa, no por constantes del nucleo ni del contrato v0.
- Las transacciones deben exponer contratos claros.

## Prohibido

- Decidir fases, modelos, autonomia o handoff en persistencia concreta.
- Asumir cualquier motor de persistencia como universal.
- Hardcodear nombres de motores, adaptadores de proveedor, credenciales, tablas o rutas locales en contratos.
- Crear queries que filtren reglas de negocio escondidas.
