# Contexto Codex: orquesta-director

Lee primero este archivo y `README.md`. Despues lee solo los docs locales necesarios para tu microtarea.

## Reglas comunes

- Contexto pequeno: usa este modulo como fuente primaria.
- Hexagonal siempre: el director compone contratos publicos, no internals.
- i18n por defecto cuando haya texto de UI, app o documentacion generada.
- Persistence, runtime, deploy, capacity, observability, web, CLI y MCP son puertos o adaptadores.
- Problema grande: primero descomponer; luego ejecutar una microtarea pequena.
- Funciones pequenas, nombres claros y tests de invariantes.
- No cruces `internal/`, tablas, structs privados ni detalles de proveedor de otro modulo.
- Si necesitas informacion o decision de otro grupo, emite `CONSULTA AL DIRECTOR`.

## Alcance local

Trabaja en la composicion/director de alto nivel:

- recibir una AppSpec y backlog ya validados;
- registrar un proyecto en core por contrato publico;
- preparar el arranque del workflow durable por refs opacas;
- devolver eventos/resultados compactos para persistencia/outbox futura;
- mantener el flujo puro, determinista y sin efectos externos.

## Prohibido

- Acceder a DB, filesystem productivo, runtime, procesos, red, OAuth, HOME o proveedores.
- Llamar a adaptadores concretos.
- Importar `internal/` de otro modulo.
- Convertir el director en un prompt monolitico.
- Decidir modelos, cuotas, deploy o persistence real dentro del flujo.

## Entrega

Cada cambio debe incluir:

- write-set pequeno;
- contrato o invariante afectada;
- prueba local;
- actualizacion de `docs/tareas.md`, `docs/pruebas.md` y `docs/decisiones.md` si cambia el estado.
