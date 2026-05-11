# Contexto Codex: orquesta-director-cycle-outbox

Lee primero este archivo y `README.md`. Despues lee solo los docs locales necesarios para tu microtarea.

## Reglas comunes

- Contexto pequeno: este modulo registra outbox generada por un ciclo del director.
- Hexagonal siempre: el ledger es un puerto; la persistencia concreta vive fuera.
- i18n por defecto cuando haya texto visible de UI, CLI, MCP o documentacion generada.
- Funciones pequenas, ficheros pequenos y pruebas de invariantes.
- No cruces `internal/`, structs privados ni detalles internos de otro modulo.
- Si necesitas informacion de otro grupo, emite `CONSULTA AL DIRECTOR`.

## Alcance local

- Recibir outbox generada por `orquesta-director-runner`.
- Guardar mensajes pendientes mediante un ledger inyectado.
- Listar pendientes del run por refs compactas para alimentar el siguiente tick.
- No despachar mensajes.

## Prohibido

- DB, filesystem productivo, red, procesos, runtime real, OAuth, HOME, cuotas o proveedores.
- Dispatcher, ACK, reintentos, goroutines, sleeps o daemon.
- Aplicar comandos workflow.
- Construir candidates o snapshots.

## Entrega

Cada cambio debe incluir contrato local, prueba local y actualizacion del verificador si entra en nucleo.
