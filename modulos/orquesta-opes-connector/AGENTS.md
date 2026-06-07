# AGENTS: orquesta-opes-connector

Lee este archivo y `README.md` antes de editar este modulo.

## Reglas

- Este modulo contiene un conector REST opt-in minimo. Cualquier ampliacion debe
  mantener tests de contrato.
- Mantener frontera hexagonal: este conector es adaptador externo y no debe
  contaminar `orquesta-domain-work` con semantica OPES.
- No acceder a DB, ficheros internos, rutas locales, workers ni estructura
  interna de OPES.
- No hardcodear backend de persistencia.
- No filtrar agentes, sesiones, `tmux`, leases, runtimes, proveedores ni
  modelos hacia OPES.
- Todo texto visible futuro debera pasar por i18n; los documentos de contrato
  pueden usar castellano como idioma de trabajo.
- Si el cambio afecta jobs de temarios o politicas editoriales, tratar
  `/home/alberto/Trabajo/OPES/AGENTS.md` como canon de dominio externo. El
  conector no lo lee en runtime; OPES/bridge lo referencian por politica.

## Write-set preferente

- ficheros pequenos por responsabilidad;
- `docs/*` actualizado si cambia contrato;
- tests con `httptest`, sin tocar OPES real.
