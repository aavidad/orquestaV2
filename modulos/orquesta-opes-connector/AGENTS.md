# AGENTS: orquesta-opes-connector

Lee este archivo y `README.md` antes de editar este modulo.

## Reglas

- Este modulo contiene un conector REST opt-in minimo. Cualquier ampliacion debe
  mantener tests de contrato.
- Mantener frontera hexagonal: el conector futuro sera adaptador externo y no
  debe contaminar `orquesta-domain-work` con semantica OPES.
- No acceder a DB, ficheros internos, rutas locales, workers ni estructura
  interna de OPES.
- No hardcodear backend de persistencia.
- No filtrar agentes, sesiones, `tmux`, leases, runtimes, proveedores ni
  modelos hacia OPES.
- Todo texto visible futuro debera pasar por i18n; los documentos de contrato
  pueden usar castellano como idioma de trabajo.

## Write-set preferente

- ficheros pequenos por responsabilidad;
- `docs/*` actualizado si cambia contrato;
- tests con `httptest`, sin tocar OPES real.
