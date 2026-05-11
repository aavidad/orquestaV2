# Contexto Codex: orquesta-core-workflow

Lee primero este archivo y `README.md`. Despues lee solo los docs locales necesarios para tu microtarea.

## Reglas comunes

- Contexto pequeno: usa este modulo como fuente primaria.
- Hexagonal siempre: conecta por comandos, eventos, puertos, DTOs y outbox.
- i18n por defecto cuando haya texto de UI, app o documentacion generada.
- Persistencia, runtime, LLM, filesystem, Git, cache, cola y deploy son adaptadores.
- Problema grande: primero descomponer; luego ejecutar una microtarea pequena.
- Funciones pequenas, nombres claros y tests de invariantes.
- No cruces `internal/`, tablas, structs privados ni detalles de proveedor de otro modulo.
- Si necesitas informacion o decision de otro grupo, emite `CONSULTA AL DIRECTOR`.
- No toques `modulos/orquesta-core/` salvo tarea explicita de extraccion registrada.
- Antes de corregir un bug de nucleo, clasificalo segun `../../docs/reinicio_orquesta_v2/protocolo_anti_bucles.md`.
- No hagas parches por sintomas: registra contrato afectado, test rojo minimo e invariante.

## Alcance local

Trabaja en el workflow durable:

- `OrchestrationRunV0`;
- fases;
- comandos;
- eventos;
- reducer;
- outbox;
- replay;
- idempotencia;
- bloqueo y consulta al director.

## Prohibido

- Importar adaptadores concretos.
- Depender de DB, HTTP, CLI, MCP, tmux, Docker, OAuth, HOME o runtime real.
- Hardcodear Codex, Claude, Ollama, vLLM o cuentas locales.
- Crear un director monolitico que decida todo por prompt.
- Crear ficheros largos como centro de todo el sistema.

## Limites de tamano

- Objetivo: funciones de menos de 40 lineas.
- Objetivo: ficheros de menos de 400 lineas.
- Si una funcion o fichero supera ese tamano, divide por responsabilidad antes de seguir.

## Entrega

Cada cambio debe incluir:

- write-set pequeno;
- contrato o invariante afectada;
- prueba unitaria o de replay/idempotencia;
- actualizacion de `docs/tareas.md` y `docs/pruebas.md` si cambia el estado.
