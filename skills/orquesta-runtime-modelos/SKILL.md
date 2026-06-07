---
name: orquesta-runtime-modelos
description: Gestionar modelos locales o cloud desde Orquesta mediante puertos runtime opt-in, con API/tool de listar, descargar, servir, parar y consultar estado sin exponer base_url ni tokens como input publico.
---

# Orquesta Runtime Modelos

Usa esta skill cuando una composicion necesite descubrir, instalar, servir,
parar o auditar modelos locales/cloud, por ejemplo Ollama.

## Frontera

- El nucleo no conoce Ollama, Gemini, Claude, Codex ni proveedor concreto.
- La app/gateway llama al puerto neutral de runtime.
- El adaptador concreto se inyecta por composicion.
- La configuracion sensible vive en env/config de servidor, no en input publico.

## API vigente

- Tool: `orquesta.runtime.models.v0`
- HTTP: `POST /api/v0/runtime/models`
- Acciones: `list`, `status`, `pull`, `serve`, `stop`
- Inputs publicos: `action`, `provider_ref`, `endpoint_ref`, `model`,
  `keep_alive`, `tags`, `evidence_refs`
- No aceptar `base_url`, token, HOME, ruta local ni credencial en payload.

## Ollama

Variables de composicion:

- `ORQUESTA_OLLAMA_MODEL_MANAGER_ENABLED`
- `ORQUESTA_OLLAMA_MODEL_MANAGER_BASE_URL`
- `ORQUESTA_OLLAMA_MODEL_MANAGER_BEARER_TOKEN`
- `ORQUESTA_OLLAMA_MODEL_MANAGER_TIMEOUT_SECONDS`

## Validacion

Ejecutar pruebas focales:

```bash
go test -count=1 ./modulos/orquesta-runtime ./modulos/orquesta-runtime-ollama ./modulos/orquesta-mcp ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway ./cmd/orquesta-server
```

Verificar que `base_url` no aparece como entrada publica de runtime/MCP/HTTP.
