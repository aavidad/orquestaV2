# Contexto Codex: orquesta-app-codex-stack

## Reglas comunes

- Contexto pequeno: lee este modulo, `README.md` y los docs locales citados.
- Hexagonal: este modulo es composition root exterior, no nucleo ni gateway
  productivo.
- El stack es opt-in. No debe activarse por defecto desde `cmd`, app-gateway
  productivo ni smokes unitarios.
- Puede importar runtime Codex y adaptadores reales porque vive fuera del core,
  pero solo mediante configuracion explicita del operador.
- No hay DB hardcodeada, proveedor hardcodeado, modelo hardcodeado, HOME
  hardcodeado ni paths globales inventados.
- Persistencia, runtime, filesystem, red, credenciales, provider, modelo y MCP
  son adaptadores/configuracion inyectada.
- Si falta una decision de arquitectura o de producto, documenta una
  `CONSULTA AL DIRECTOR`; no la resuelvas con defaults ocultos.

## Alcance

Mini-proyecto exterior para componer una ruta real opt-in:

```text
/nueva-app web/API/MCP
  -> app-gateway existente
  -> StartAppDirectorPortsV0 reales inyectados
  -> app-director-service
  -> runtime Codex delivery/progress opt-in
```

El objetivo es preparar el contexto para que web, API REST y MCP puedan probar
composicion real con Codex sin tocar core ni app-gateway productivo.

## Permitido

- Documentar contratos, tareas, decisiones y pruebas.
- Crear scripts de arranque opt-in que deleguen en comandos/configuracion
  explicitos.
- Importar en una fase futura los modulos exteriores necesarios para componer
  puertos reales.
- Reutilizar smokes reales existentes de `orquesta-runtime-codex-delivery` como
  evidencia y patron de guardas opt-in.

## Prohibido

- Tocar `orquestacionnucleoapp`, `cmd` o `modulos/orquesta-app-gateway` para
  activar este stack por defecto.
- Crear Go nuevo salvo que sea imprescindible para el contrato documentado.
- Elegir SQLite, Postgres, filesystem, provider, modelo, HOME, OAuth, puerto
  HTTP o comando Codex como default.
- Saltarse `StartAppDirectorPortsV0` o reconstruir rutas internas desde REST,
  web o MCP.
- Filtrar paths locales, tokens, prompts, transcripts, HOME, provider o modelo
  hacia el nucleo.

## Validacion local

```bash
git diff --check -- modulos/orquesta-app-codex-stack
```
