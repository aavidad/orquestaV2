# AGENTS

Contexto local obligatorio antes de editar:

- `README.md`
- `docs/contratos.md`
- `docs/pruebas.md`
- si el cambio toca reglas de temarios, jobs documentales o artefactos OPES:
  `/home/alberto/Trabajo/OPES/AGENTS.md`

Reglas:

- Este modulo es un adaptador OPES-Orquesta, no core.
- Solo usa API publica OPES y API publica Orquesta.
- No leer DB, ficheros internos, rutas locales ni procesos de OPES.
- No introducir runtime, modelo, proveedor ni sesiones de agentes en OPES.
- Las reglas editoriales de temarios salen del canon OPES externo y deben viajar
  como campos/refs de dominio del adaptador; no se meten en el nucleo Orquesta.
- Mantener funciones pequenas y contratos hexagonales.
