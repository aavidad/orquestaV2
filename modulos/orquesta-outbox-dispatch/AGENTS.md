# Contexto para agentes: orquesta-outbox-dispatch

Lee siempre `AGENTS.md`, `README.md`, `docs/contratos.md`, `docs/tareas.md`,
`docs/pruebas.md` y `docs/decisiones.md`.

Reglas:

- Mantener arquitectura hexagonal: contratos y puertos primero, adaptadores fuera.
- No ejecutar runtime, ACK real, daemon, sleeps ni goroutines.
- No introducir DB, SQL, filesystem productivo, red, proveedor, modelo, HOME ni OAuth.
- Las decisiones de dispatch deben ser puras, deterministas y reproducibles por replay.
- Si falta informacion de otro modulo, devolver una decision sin intent o documentar consulta.
- No tocar `../orquesta-director`; usarlo solo como contexto conceptual de lectura.
