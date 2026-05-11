# Contexto Codex: orquesta-director-candidates

Lee siempre, en este orden:

1. `AGENTS.md`
2. `README.md`
3. `docs/contratos.md`
4. `docs/pruebas.md`
5. `docs/decisiones.md`

Reglas:

- Este modulo construye candidates puros para `orquesta-director-scheduler`.
- No decide ticks, no aplica comandos, no persiste y no despacha outbox.
- No acceder a DB, colas, filesystem productivo, red ni runtime.
- No seleccionar proveedor, modelo, HOME, OAuth ni credenciales.
- Recibir refs, comandos, idempotency keys, claims o scopes de forma explicita.
- Devolver DTOs publicos validados, sin inventar datos externos.
- Mantener ficheros pequenos; dividir antes de mezclar validacion, construccion y docs.
