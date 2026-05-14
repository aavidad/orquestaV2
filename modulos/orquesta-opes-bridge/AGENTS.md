# AGENTS

Contexto local obligatorio antes de editar:

- `README.md`
- `docs/contratos.md`
- `docs/pruebas.md`

Reglas:

- Este modulo es un adaptador OPES-Orquesta, no core.
- Solo usa API publica OPES y API publica Orquesta.
- No leer DB, ficheros internos, rutas locales ni procesos de OPES.
- No introducir runtime, modelo, proveedor ni sesiones de agentes en OPES.
- Mantener funciones pequenas y contratos hexagonales.
